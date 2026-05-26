package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"learncode/internal/executor"
	"learncode/internal/llm"
	"learncode/internal/model"
	"learncode/internal/repo"
)

// KBBuildService orchestrates knowledge base construction for a language version.
type KBBuildService struct {
	VersionRepo   *repo.VersionRepo
	LangRepo      *repo.LanguageRepo
	KnowledgeRepo *repo.KnowledgeRepo
	LLM           *llm.Service
	Executor      *executor.Executor
	PromptDir     string
	VersionSvc    *VersionService
	Explorer      *KBExplorer
}

// TopicSpec is a single knowledge topic returned by the LLM topic generator.
type TopicSpec struct {
	Topic    string `json:"topic"`
	Scope    string `json:"scope"`
	Category string `json:"category"`
	Brief    string `json:"brief"`
}

// Build constructs the knowledge base using a layered approach.
func (s *KBBuildService) Build(ctx context.Context, versionID string) error {
	ver, err := s.VersionRepo.GetByID(ctx, versionID)
	if err != nil {
		return fmt.Errorf("get version %s: %w", versionID, err)
	}
	if !ver.Initialized {
		return fmt.Errorf("version %s is not initialized yet", versionID)
	}

	lang, err := s.LangRepo.GetByID(ctx, ver.LanguageID)
	if err != nil {
		return fmt.Errorf("get language %s: %w", ver.LanguageID, err)
	}

	if err := s.VersionRepo.UpdateKBStatus(ctx, versionID, "building"); err != nil {
		return fmt.Errorf("set kb_status=building: %w", err)
	}

	defer func() {
		if err != nil {
			slog.Error("kb build failed", "version_id", versionID, "error", err)
			if ferr := s.VersionRepo.UpdateKBStatus(ctx, versionID, "failed"); ferr != nil {
				slog.Error("failed to set kb_status=failed", "version_id", versionID, "error", ferr)
			}
		}
	}()

	// Phase 1: Core knowledge (only once per language)
	existingCore, _ := s.KnowledgeRepo.ListByScope(ctx, lang.ID, "core")
	coreSummary := summarizeEntries(existingCore)
	if len(existingCore) == 0 {
		slog.Info("building core knowledge", "language", lang.Name)
		coreTopics, err := s.generateTopics(ctx, ver, lang)
		if err != nil {
			return fmt.Errorf("generate core topics: %w", err)
		}
		s.buildLayer(ctx, ver, lang, coreTopics, "")
		existingCore, _ = s.KnowledgeRepo.ListByScope(ctx, lang.ID, "core")
		coreSummary = summarizeEntries(existingCore)
	}

	// Phase 2: Version knowledge (always)
	slog.Info("building version knowledge", "language", lang.Name, "version", ver.Version)
	verTopics, err := s.generateTopics(ctx, ver, lang)
	if err != nil {
		return fmt.Errorf("generate version topics: %w", err)
	}
	s.buildLayer(ctx, ver, lang, verTopics, coreSummary)

	// Phase 3: Idiom knowledge (only once per language)
	existingIdiom, _ := s.KnowledgeRepo.ListByScope(ctx, lang.ID, "idiom")
	if len(existingIdiom) == 0 {
		slog.Info("building idiom knowledge", "language", lang.Name)
		idiomTopics, err := s.generateTopics(ctx, ver, lang)
		if err != nil {
			return fmt.Errorf("generate idiom topics: %w", err)
		}
		s.buildLayer(ctx, ver, lang, idiomTopics, "")
	}

	if err := s.VersionSvc.CheckLanguageActivation(ctx, lang.ID); err != nil {
		slog.Warn("failed to check language activation", "language_id", lang.ID, "error", err)
	}

	if err := s.VersionRepo.UpdateKBStatus(ctx, versionID, "complete"); err != nil {
		return fmt.Errorf("set kb_status=complete: %w", err)
	}

	slog.Info("kb build complete",
		"language", lang.Name, "version", ver.Version,
		"core_entries", len(existingCore), "idiom_entries", len(existingIdiom),
	)

	return nil
}

func (s *KBBuildService) buildLayer(ctx context.Context, ver *model.LanguageVersion, lang *model.Language, topics []TopicSpec, existingCoreSummary string) {
	var factualEntries []model.KnowledgeEntry
	for _, spec := range topics {
		entry, buildErr := s.buildEntry(ctx, ver, lang, spec, existingCoreSummary)
		if buildErr != nil {
			slog.Warn("failed to build knowledge entry, skipping", "topic", spec.Topic, "scope", spec.Scope, "error", buildErr)
			continue
		}
		if spec.Category == "factual" {
			factualEntries = append(factualEntries, *entry)
		}
	}
	s.verifyFactualEntries(ctx, ver, factualEntries)
}

func summarizeEntries(entries []model.KnowledgeEntry) string {
	if len(entries) == 0 {
		return ""
	}
	var lines []string
	for _, e := range entries {
		lines = append(lines, "- "+e.Topic)
	}
	return strings.Join(lines, "\n")
}

func (s *KBBuildService) generateTopics(ctx context.Context, ver *model.LanguageVersion, lang *model.Language) ([]TopicSpec, error) {
	if s.LLM == nil {
		return nil, fmt.Errorf("llm service not available")
	}

	vars := map[string]string{
		"LanguageName":       lang.Name,
		"Slug":               lang.Slug,
		"Version":            ver.Version,
		"CompatibilityModel": lang.CompatibilityModel,
		"Description":        "", // populated from language research data when available
		"ExistingKnowledge":  "", // populated when building version-specific topics
	}

	tmpl, err := llm.LoadTemplate(s.PromptDir+"/kb_topics.yaml", vars)
	if err != nil {
		return nil, fmt.Errorf("load topics template: %w", err)
	}

	content, _, err := s.LLM.ChatWithTemp(ctx, tmpl.SystemPrompt, tmpl.UserPrompt, tmpl.Temperature, tmpl.MaxTokens)
	if err != nil {
		return nil, fmt.Errorf("llm chat: %w", err)
	}

	var topics []TopicSpec
	if err := llm.ParseLLMJSON(content, &topics); err != nil {
		var wrapper struct {
			Topics []TopicSpec `json:"topics"`
		}
		if err2 := llm.ParseLLMJSON(content, &wrapper); err2 != nil {
			return nil, fmt.Errorf("parse topics response: %w", err)
		}
		topics = wrapper.Topics
	}

	if lang.CompatibilityModel == "strict" {
		for i := range topics {
			topics[i].Scope = "core"
		}
	}

	return topics, nil
}

// buildEntry builds a single knowledge entry.
// Factual entries MUST go through Explorer (environment-interactive discovery).
// Normative entries use LLM with existing factual knowledge as context.
func (s *KBBuildService) buildEntry(ctx context.Context, ver *model.LanguageVersion, lang *model.Language, spec TopicSpec, existingCoreSummary string) (*model.KnowledgeEntry, error) {
	if s.LLM == nil {
		return nil, fmt.Errorf("llm service not available")
	}

	if spec.Category == "factual" {
		if s.Explorer == nil {
			return nil, fmt.Errorf("explorer not available for factual topic %q", spec.Topic)
		}
		rc, err := executor.ParseRuntimeConfig(ver.RuntimeConfig)
		if err != nil {
			rc = executor.DefaultRuntimeConfig(lang.Slug)
		}
		// Same logic as Executor.Execute: Docker path only needs Image + Interpreter.
		// Host path needs complete config (IsComplete).
		if rc.Image == "" || s.Executor == nil {
			if !rc.IsComplete() {
				return nil, fmt.Errorf("runtime config incomplete for topic %q", spec.Topic)
			}
		}
		entry, err := s.Explorer.ExploreTopic(ctx, ver, lang, rc, spec)
		if err != nil {
			return nil, fmt.Errorf("explore topic %q: %w", spec.Topic, err)
		}
		if err := s.KnowledgeRepo.Create(ctx, entry); err != nil {
			return nil, fmt.Errorf("save explorer entry %q: %w", spec.Topic, err)
		}
		return entry, nil
	}

	// Normative entries
	vars := map[string]string{
		"LanguageName": lang.Name,
		"Slug":         lang.Slug,
		"Version":      ver.Version,
		"Topic":        spec.Topic,
		"Brief":        spec.Brief,
	}
	if existingCoreSummary != "" {
		vars["ExistingCoreTopics"] = existingCoreSummary
	}

	tmpl, err := llm.LoadTemplate(s.PromptDir+"/kb_entry_normative.yaml", vars)
	if err != nil {
		return nil, fmt.Errorf("load normative template: %w", err)
	}
	content, _, err := s.LLM.ChatWithTemp(ctx, tmpl.SystemPrompt, tmpl.UserPrompt, tmpl.Temperature, tmpl.MaxTokens)
	if err != nil {
		return nil, fmt.Errorf("llm chat for topic %q: %w", spec.Topic, err)
	}

	var entryContent json.RawMessage
	if err := llm.ParseLLMJSON(content, &entryContent); err != nil {
		return nil, fmt.Errorf("parse normative entry for topic %q: %w", spec.Topic, err)
	}

	var versionID *string
	if spec.Scope == "version" {
		versionID = &ver.ID
	}

	entry := &model.KnowledgeEntry{
		LanguageID: lang.ID,
		VersionID:  versionID,
		Scope:      spec.Scope,
		Category:   spec.Category,
		Topic:      spec.Topic,
		Content:    entryContent,
		Source:     "llm",
	}

	if err := s.KnowledgeRepo.Create(ctx, entry); err != nil {
		return nil, fmt.Errorf("save knowledge entry %q: %w", spec.Topic, err)
	}

	return entry, nil
}

func (s *KBBuildService) verifyFactualEntries(ctx context.Context, ver *model.LanguageVersion, entries []model.KnowledgeEntry) {
	rc, err := executor.ParseRuntimeConfig(ver.RuntimeConfig)
	if err != nil {
		slog.Warn("failed to parse runtime config for verification", "version_id", ver.ID, "error", err)
		return
	}

	for _, entry := range entries {
		var content struct {
			VerificationCode string `json:"verification_code"`
			ExpectedOutput   string `json:"expected_output"`
		}
		if err := json.Unmarshal(entry.Content, &content); err != nil {
			slog.Warn("failed to parse entry content for verification", "topic", entry.Topic, "error", err)
			continue
		}
		if content.VerificationCode == "" {
			continue
		}

		result, err := s.Executor.Execute(ctx, rc, content.VerificationCode)
		if err != nil {
			slog.Warn("verification execution failed", "topic", entry.Topic, "error", err)
			continue
		}

		if result.ExitCode != 0 {
			slog.Warn("verification code exited non-zero", "topic", entry.Topic, "exit_code", result.ExitCode, "stderr", result.Stderr)
			continue
		}

		actual := strings.TrimSpace(result.Stdout)
		expected := strings.TrimSpace(content.ExpectedOutput)
		if actual != expected {
			slog.Warn("verification output mismatch", "topic", entry.Topic, "expected", expected, "actual", actual)
			continue
		}

		var raw map[string]json.RawMessage
		if err := json.Unmarshal(entry.Content, &raw); err == nil {
			verified, _ := json.Marshal(true)
			raw["verified"] = verified
			if updated, err := json.Marshal(raw); err == nil {
				entry.Content = updated
			}
		}

		slog.Info("verification passed", "topic", entry.Topic, "version", ver.Version)
	}
}
