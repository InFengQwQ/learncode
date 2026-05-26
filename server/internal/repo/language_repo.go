package repo

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jmoiron/sqlx"

	"learncode/internal/model"
)

type LanguageRepo struct {
	DB *sqlx.DB
}

const languageCols = `id, name, slug, icon, compatibility_model, source_urls, research_data, researched_at, status, created_at`

func (r *LanguageRepo) List(ctx context.Context) ([]model.Language, error) {
	var langs []model.Language
	err := r.DB.SelectContext(ctx, &langs,
		`SELECT `+languageCols+` FROM languages ORDER BY created_at`)
	return langs, err
}

func (r *LanguageRepo) GetByID(ctx context.Context, id string) (*model.Language, error) {
	var lang model.Language
	err := r.DB.GetContext(ctx, &lang,
		`SELECT `+languageCols+` FROM languages WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	return &lang, nil
}

func (r *LanguageRepo) GetBySlug(ctx context.Context, slug string) (*model.Language, error) {
	var lang model.Language
	err := r.DB.GetContext(ctx, &lang,
		`SELECT `+languageCols+` FROM languages WHERE slug = $1`, slug)
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
		`INSERT INTO languages (name, slug, icon, compatibility_model, source_urls)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING `+languageCols,
		lang.Name, lang.Slug, lang.Icon, lang.CompatibilityModel, lang.SourceURLs,
	)
}

// UpdateFromResearch saves research results and updated source_urls to the language record.
func (r *LanguageRepo) UpdateFromResearch(ctx context.Context, id string, researchData json.RawMessage, researchedAt time.Time, sourceURLs json.RawMessage) error {
	_, err := r.DB.ExecContext(ctx,
		`UPDATE languages SET research_data = $1, researched_at = $2, source_urls = $3 WHERE id = $4`,
		researchData, researchedAt, sourceURLs, id,
	)
	return err
}

// UpdateStatus sets the language status (inactive or active).
func (r *LanguageRepo) UpdateStatus(ctx context.Context, id string, status string) error {
	_, err := r.DB.ExecContext(ctx,
		`UPDATE languages SET status = $2 WHERE id = $1`,
		id, status,
	)
	return err
}