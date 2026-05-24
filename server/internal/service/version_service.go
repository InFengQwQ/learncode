package service

import (
	"context"

	"learncode/internal/model"
	"learncode/internal/repo"
)

type VersionService struct {
	Repo *repo.VersionRepo
}

func (s *VersionService) ListByLanguageID(ctx context.Context, languageID string) ([]model.LanguageVersion, error) {
	return s.Repo.ListByLanguageID(ctx, languageID)
}

func (s *VersionService) GetByID(ctx context.Context, id string) (*model.LanguageVersion, error) {
	return s.Repo.GetByID(ctx, id)
}

func (s *VersionService) Create(ctx context.Context, v *model.LanguageVersion) error {
	return s.Repo.Create(ctx, v)
}
