package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jmoiron/sqlx"

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
	lang, err := s.LangRepo.GetByID(ctx, v.LanguageID)
	if err != nil {
		return fmt.Errorf("lookup language %s: %w", v.LanguageID, err)
	}

	if lang.ResearchedAt == nil {
		rc := executor.DefaultRuntimeConfig(lang.Slug)
		if !rc.IsComplete() {
			return fmt.Errorf("language %q has not been researched yet and has no known runtime configuration — please run deep research first", lang.Name)
		}
	}

	if len(v.RuntimeConfig) == 0 || string(v.RuntimeConfig) == "null" {
		rc := executor.DefaultRuntimeConfig(lang.Slug)
		v.RuntimeConfig = rc.Marshal()
	}

	if lang.CompatibilityModel == "strict" {
		tx, err := s.Repo.DB.BeginTxx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback()

		if err := s.Repo.ArchiveActiveByLanguageTx(ctx, tx, v.LanguageID); err != nil {
			return fmt.Errorf("archive previous version for strict language %s: %w", lang.Slug, err)
		}
		if err := s.Repo.CreateTx(ctx, tx, v); err != nil {
			return fmt.Errorf("create version: %w", err)
		}
		return tx.Commit()
	}

	return s.Repo.Create(ctx, v)
}

func (s *VersionService) CreateWithTx(ctx context.Context, tx *sqlx.Tx, v *model.LanguageVersion) error {
	lang, err := s.LangRepo.GetByID(ctx, v.LanguageID)
	if err != nil {
		return fmt.Errorf("lookup language %s: %w", v.LanguageID, err)
	}

	if lang.ResearchedAt == nil {
		rc := executor.DefaultRuntimeConfig(lang.Slug)
		if !rc.IsComplete() {
			return fmt.Errorf("language %q has not been researched yet and has no known runtime configuration", lang.Name)
		}
	}

	if len(v.RuntimeConfig) == 0 || string(v.RuntimeConfig) == "null" {
		rc := executor.DefaultRuntimeConfig(lang.Slug)
		v.RuntimeConfig = rc.Marshal()
	}

	if lang.CompatibilityModel == "strict" {
		if err := s.Repo.ArchiveActiveByLanguageTx(ctx, tx, v.LanguageID); err != nil {
			return fmt.Errorf("archive previous version for strict language %s: %w", lang.Slug, err)
		}
	}

	return s.Repo.CreateTx(ctx, tx, v)
}

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
