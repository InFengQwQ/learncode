package repo

import (
	"context"

	"github.com/jmoiron/sqlx"

	"learncode/internal/model"
)

type LanguageRepo struct {
	DB *sqlx.DB
}

func (r *LanguageRepo) List(ctx context.Context) ([]model.Language, error) {
	var langs []model.Language
	err := r.DB.SelectContext(ctx, &langs, "SELECT id, name, slug, icon, compatibility_model, source_urls, created_at FROM languages ORDER BY created_at")
	return langs, err
}

func (r *LanguageRepo) GetByID(ctx context.Context, id string) (*model.Language, error) {
	var lang model.Language
	err := r.DB.GetContext(ctx, &lang, "SELECT id, name, slug, icon, compatibility_model, source_urls, created_at FROM languages WHERE id = $1", id)
	if err != nil {
		return nil, err
	}
	return &lang, nil
}

func (r *LanguageRepo) GetBySlug(ctx context.Context, slug string) (*model.Language, error) {
	var lang model.Language
	err := r.DB.GetContext(ctx, &lang, "SELECT id, name, slug, icon, compatibility_model, source_urls, created_at FROM languages WHERE slug = $1", slug)
	if err != nil {
		return nil, err
	}
	return &lang, nil
}

func (r *LanguageRepo) Delete(ctx context.Context, id string) error {
	_, err := r.DB.ExecContext(ctx, "DELETE FROM languages WHERE id = $1", id)
	return err
}

func (r *LanguageRepo) Create(ctx context.Context, lang *model.Language) error {
	return r.DB.GetContext(ctx, lang,
		"INSERT INTO languages (name, slug, icon, compatibility_model, source_urls) VALUES ($1, $2, $3, $4, $5) RETURNING id, name, slug, icon, compatibility_model, source_urls, created_at",
		lang.Name, lang.Slug, lang.Icon, lang.CompatibilityModel, lang.SourceURLs,
	)
}
