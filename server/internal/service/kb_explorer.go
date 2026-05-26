package service

import (
	"context"
	"encoding/json"
	"fmt"

	"learncode/internal/executor"
	"learncode/internal/llm"
	"learncode/internal/model"
)

// KBExplorer orchestrates environment-interactive knowledge discovery.
// Instead of generating entries from LLM training data, the Explorer:
//  1. Asks the LLM to generate exploratory code snippets (probes)
//  2. Executes all probes in the language's runtime environment
//  3. Asks the LLM to synthesize observations into a knowledge entry
//
// This keeps each LLM call within token limits while enabling the LLM
// to learn from actual observed behavior.
type KBExplorer struct {
	LLM       *llm.Service
	Executor  *executor.Executor
	PromptDir string
}

type probeSpec struct {
	Code      string `json:"code"`
	Hypothesis string `json:"hypothesis"`
}

type probeResult struct {
	Code       string `json:"code"`
	Hypothesis string `json:"hypothesis"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ExitCode   int    `json:"exit_code"`
	Error      string `json:"error,omitempty"`
}

// ExploreTopic discovers knowledge about a specific topic by generating probes,
// executing them in the runtime environment, and synthesizing the results.
func (e *KBExplorer) ExploreTopic(
	ctx context.Context,
	ver *model.LanguageVersion,
	lang *model.Language,
	rc executor.RuntimeConfig,
	spec TopicSpec,
) (*model.KnowledgeEntry, error) {
	// Step 1: Generate exploratory code snippets
	probes, err := e.generateProbes(ctx, lang, ver, spec)
	if err != nil {
		return nil, fmt.Errorf("generate probes for %q: %w", spec.Topic, err)
	}
	if len(probes) == 0 {
		return nil, fmt.Errorf("no probes generated for %q", spec.Topic)
	}

	// Step 2: Execute all probes
	results := e.executeProbes(ctx, rc, probes)

	// Step 3: Synthesize into knowledge entry
	entry, err := e.synthesize(ctx, lang, ver, spec, results)
	if err != nil {
		return nil, fmt.Errorf("synthesize entry for %q: %w", spec.Topic, err)
	}

	return entry, nil
}

// generateProbes asks the LLM to create exploratory code snippets for a topic.
func (e *KBExplorer) generateProbes(
	ctx context.Context,
	lang *model.Language,
	ver *model.LanguageVersion,
	spec TopicSpec,
) ([]probeSpec, error) {
	vars := map[string]string{
		"LanguageName": lang.Name,
		"Slug":         lang.Slug,
		"Version":      ver.Version,
		"Topic":        spec.Topic,
		"Brief":         spec.Brief,
	}

	tmpl, err := llm.LoadTemplate(e.PromptDir+"/kb_probe.yaml", vars)
	if err != nil {
		return nil, fmt.Errorf("load probe template: %w", err)
	}

	content, _, err := e.LLM.ChatWithTemp(ctx, tmpl.SystemPrompt, tmpl.UserPrompt, tmpl.Temperature, tmpl.MaxTokens)
	if err != nil {
		return nil, fmt.Errorf("llm probes: %w", err)
	}

	var probes []probeSpec
	if err := llm.ParseLLMJSON(content, &probes); err != nil {
		return nil, fmt.Errorf("parse probes: %w", err)
	}
	return probes, nil
}

// executeProbes runs all probe code snippets and collects results.
func (e *KBExplorer) executeProbes(ctx context.Context, rc executor.RuntimeConfig, probes []probeSpec) []probeResult {
	results := make([]probeResult, 0, len(probes))
	for _, p := range probes {
		r := probeResult{Code: p.Code, Hypothesis: p.Hypothesis}
		result, err := e.Executor.Execute(ctx, rc, p.Code)
		if err != nil {
			r.Error = err.Error()
		} else {
			r.Stdout = result.Stdout
			r.Stderr = result.Stderr
			r.ExitCode = result.ExitCode
		}
		results = append(results, r)
	}
	return results
}

// synthesize asks the LLM to build a knowledge entry from probe execution results.
func (e *KBExplorer) synthesize(
	ctx context.Context,
	lang *model.Language,
	ver *model.LanguageVersion,
	spec TopicSpec,
	results []probeResult,
) (*model.KnowledgeEntry, error) {
	resultsJSON, _ := json.MarshalIndent(results, "", "  ")

	vars := map[string]string{
		"LanguageName": lang.Name,
		"Slug":         lang.Slug,
		"Version":      ver.Version,
		"Topic":        spec.Topic,
		"Brief":         spec.Brief,
		"Results":      string(resultsJSON),
	}

	tmpl, err := llm.LoadTemplate(e.PromptDir+"/kb_synthesize.yaml", vars)
	if err != nil {
		return nil, fmt.Errorf("load synthesize template: %w", err)
	}

	content, _, err := e.LLM.ChatWithTemp(ctx, tmpl.SystemPrompt, tmpl.UserPrompt, tmpl.Temperature, tmpl.MaxTokens)
	if err != nil {
		return nil, fmt.Errorf("llm synthesize: %w", err)
	}

	var entryContent json.RawMessage
	if err := llm.ParseLLMJSON(content, &entryContent); err != nil {
		return nil, fmt.Errorf("parse synthesize response: %w", err)
	}

	// Determine version_id: core and idiom are shared (nil), version is per-version
	var versionID *string
	if spec.Scope == "version" {
		versionID = &ver.ID
	}

	return &model.KnowledgeEntry{
		LanguageID: lang.ID,
		VersionID:  versionID,
		Scope:      spec.Scope,
		Category:   spec.Category,
		Topic:      spec.Topic,
		Content:    entryContent,
		Source:     "env",
	}, nil
}
