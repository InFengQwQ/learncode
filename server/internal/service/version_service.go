package service

import (
	"context"
	"fmt"
	"log/slog"

	"learncode/internal/executor"
	"learncode/internal/model"
	"learncode/internal/repo"
)

type VersionService struct {
	Repo     *repo.VersionRepo
	LangRepo *repo.LanguageRepo
}

func (s *VersionService) ListByLanguageID(ctx context.Context, languageID string) ([]model.LanguageVersion, error) {
	return s.Repo.ListByLanguageID(ctx, languageID)
}

func (s *VersionService) GetByID(ctx context.Context, id string) (*model.LanguageVersion, error) {
	return s.Repo.GetByID(ctx, id)
}

// SetStatus changes a version's status between "active" and "archived".
// For strict languages, archiving the active version also activates the next candidate if one exists.
func (s *VersionService) SetStatus(ctx context.Context, versionID string, newStatus string) (*model.LanguageVersion, error) {
	v, err := s.Repo.GetByID(ctx, versionID)
	if err != nil {
		return nil, fmt.Errorf("get version: %w", err)
	}
	if v.Status == newStatus {
		return v, nil
	}

	lang, err := s.LangRepo.GetByID(ctx, v.LanguageID)
	if err != nil {
		return nil, fmt.Errorf("get language: %w", err)
	}

	if lang.CompatibilityModel == "strict" && newStatus == "active" {
		// Archive the previously active version first
		if err := s.Repo.ArchiveActiveByLanguage(ctx, v.LanguageID); err != nil {
			return nil, fmt.Errorf("archive previous: %w", err)
		}
	}

	if err := s.Repo.UpdateStatus(ctx, versionID, newStatus); err != nil {
		return nil, err
	}
	v.Status = newStatus
	return v, nil
}

func (s *VersionService) Create(ctx context.Context, v *model.LanguageVersion) error {
	// Look up the language to get slug and compatibility_model.
	lang, err := s.LangRepo.GetByID(ctx, v.LanguageID)
	if err != nil {
		return fmt.Errorf("lookup language %s: %w", v.LanguageID, err)
	}

	// Prerequisite: the language must have been researched, or have a known
	// runtime config that can be auto-provisioned. Without either, creating
	// a version is meaningless — there are no resources to build from.
	if lang.ResearchedAt == nil {
		rc := executor.DefaultRuntimeConfig(lang.Slug)
		if !rc.IsComplete() {
			return fmt.Errorf("language %q has not been researched yet and has no known runtime configuration — please run deep research first", lang.Name)
		}
	}

	// Auto-populate runtime_config from the default lookup table.
	// Only fill it in if the caller didn't provide one.
	if len(v.RuntimeConfig) == 0 || string(v.RuntimeConfig) == "null" {
		rc := executor.DefaultRuntimeConfig(lang.Slug)
		v.RuntimeConfig = rc.Marshal()
	}

	// For strict languages, only one active version is allowed at a time.
	// Archive the currently active version before creating the new one.
	if lang.CompatibilityModel == "strict" {
		if err := s.Repo.ArchiveActiveByLanguage(ctx, v.LanguageID); err != nil {
			return fmt.Errorf("archive previous version for strict language %s: %w", lang.Slug, err)
		}
	}

	return s.Repo.Create(ctx, v)
}

// CheckLanguageActivation checks whether a language should be active or inactive
// based on its versions' kb_status values. If any version has kb_status = 'complete',
// the language is activated; otherwise it is deactivated.
func (s *VersionService) CheckLanguageActivation(ctx context.Context, languageID string) error {
	count, err := s.Repo.CountByKBStatus(ctx, languageID, "complete")
	if err != nil {
		return fmt.Errorf("check kb_status for language %s: %w", languageID, err)
	}

	lang, err := s.LangRepo.GetByID(ctx, languageID)
	if err != nil {
		return fmt.Errorf("get language %s: %w", languageID, err)
	}

	newStatus := "inactive"
	if count > 0 {
		newStatus = "active"
	}

	if lang.Status != newStatus {
		slog.Info("updating language activation status", "language_id", languageID, "old", lang.Status, "new", newStatus)
		if err := s.LangRepo.UpdateStatus(ctx, languageID, newStatus); err != nil {
			return fmt.Errorf("update language %s status to %s: %w", languageID, newStatus, err)
		}
	}

	return nil
}
