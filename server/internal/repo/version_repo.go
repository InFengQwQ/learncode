package repo

import (
	"context"

	"github.com/jmoiron/sqlx"

	"learncode/internal/model"
)

type VersionRepo struct {
	DB *sqlx.DB
}

func (r *VersionRepo) ListByLanguageID(ctx context.Context, languageID string) ([]model.LanguageVersion, error) {
	var versions []model.LanguageVersion
	err := r.DB.SelectContext(ctx, &versions,
		"SELECT id, language_id, version, status, runtime_config, source_urls, last_version_check_at, initialized, created_at, updated_at FROM language_versions WHERE language_id = $1 ORDER BY created_at",
		languageID,
	)
	return versions, err
}

func (r *VersionRepo) GetByID(ctx context.Context, id string) (*model.LanguageVersion, error) {
	var v model.LanguageVersion
	err := r.DB.GetContext(ctx, &v,
		"SELECT id, language_id, version, status, runtime_config, source_urls, last_version_check_at, initialized, created_at, updated_at FROM language_versions WHERE id = $1",
		id,
	)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *VersionRepo) Create(ctx context.Context, v *model.LanguageVersion) error {
	return r.DB.GetContext(ctx, v,
		`INSERT INTO language_versions (language_id, version, status)
		 VALUES ($1, $2, $3)
		 RETURNING id, language_id, version, status, runtime_config, source_urls, last_version_check_at, initialized, created_at, updated_at`,
		v.LanguageID, v.Version, v.Status,
	)
}
