package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"learncode/internal/model"
	"learncode/internal/repo"
)

type LanguageService struct {
	Repo *repo.LanguageRepo
}

func (s *LanguageService) List(ctx context.Context) ([]model.Language, error) {
	return s.Repo.List(ctx)
}

func (s *LanguageService) GetByID(ctx context.Context, id string) (*model.Language, error) {
	return s.Repo.GetByID(ctx, id)
}

func (s *LanguageService) Delete(ctx context.Context, id string) error {
	return s.Repo.Delete(ctx, id)
}

func (s *LanguageService) Create(ctx context.Context, input CreateLanguageInput) (*model.Language, error) {
	if input.Name == "" {
		return nil, errors.New("name is required")
	}
	if input.Slug == "" {
		return nil, errors.New("slug is required")
	}
	if input.CompatibilityModel != "strict" && input.CompatibilityModel != "versioned" {
		return nil, fmt.Errorf("invalid compatibility_model: %s", input.CompatibilityModel)
	}

	existing, err := s.Repo.GetBySlug(ctx, input.Slug)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("slug %q already exists", input.Slug)
	}

	lang := &model.Language{
		Name:               input.Name,
		Slug:               input.Slug,
		Icon:               input.Icon,
		CompatibilityModel: input.CompatibilityModel,
		SourceURLs:         input.SourceURLs,
	}
	if err := s.Repo.Create(ctx, lang); err != nil {
		return nil, err
	}
	return lang, nil
}

type CreateLanguageInput struct {
	Name               string          `json:"name"`
	Slug               string          `json:"slug"`
	Icon               string          `json:"icon"`
	CompatibilityModel string          `json:"compatibility_model"`
	SourceURLs         json.RawMessage `json:"source_urls"`
}
