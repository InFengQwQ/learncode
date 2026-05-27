package repo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"

	"learncode/internal/model"
)

type VersionRepo struct {
	DB *sqlx.DB
}

const versionCols = `id, language_id, version, status, runtime_config, source_urls, last_version_check_at, initialized, image, kb_status, initialized_at, created_at, updated_at`

func (r *VersionRepo) ListByLanguageID(ctx context.Context, languageID string) ([]model.LanguageVersion, error) {
	var versions []model.LanguageVersion
	err := r.DB.SelectContext(ctx, &versions,
		`SELECT `+versionCols+` FROM language_versions WHERE language_id = $1 ORDER BY created_at`,
		languageID,
	)
	return versions, err
}

func (r *VersionRepo) GetByID(ctx context.Context, id string) (*model.LanguageVersion, error) {
	var v model.LanguageVersion
	err := r.DB.GetContext(ctx, &v,
		`SELECT `+versionCols+` FROM language_versions WHERE id = $1`,
		id,
	)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *VersionRepo) Create(ctx context.Context, v *model.LanguageVersion) error {
	return r.DB.GetContext(ctx, v,
		`INSERT INTO language_versions (language_id, version, status, runtime_config)
		 VALUES ($1, $2, $3, $4)
		 RETURNING `+versionCols,
		v.LanguageID, v.Version, v.Status, v.RuntimeConfig,
	)
}

// Update modifies an existing version's mutable fields (status, runtime_config, image, initialized, initialized_at).
func (r *VersionRepo) Update(ctx context.Context, v *model.LanguageVersion) error {
	return r.DB.GetContext(ctx, v,
		`UPDATE language_versions
		 SET status = $2, runtime_config = $3, image = $4, initialized = $5, initialized_at = $6, updated_at = NOW()
		 WHERE id = $1
		 RETURNING `+versionCols,
		v.ID, v.Status, v.RuntimeConfig, v.Image, v.Initialized, v.InitializedAt,
	)
}

// MarkInitialized sets initialized=true, image, and initialized_at for a version.
func (r *VersionRepo) MarkInitialized(ctx context.Context, id string, image string, runtimeConfig json.RawMessage) error {
	_, err := r.DB.ExecContext(ctx,
		`UPDATE language_versions
		 SET initialized = true, image = $2, runtime_config = $3, initialized_at = NOW(), updated_at = NOW()
		 WHERE id = $1`,
		id, image, runtimeConfig,
	)
	return err
}

// UpdateKBStatus sets the knowledge base status for a version.
func (r *VersionRepo) UpdateKBStatus(ctx context.Context, id string, kbStatus string) error {
	_, err := r.DB.ExecContext(ctx,
		`UPDATE language_versions SET kb_status = $2, updated_at = NOW() WHERE id = $1`,
		id, kbStatus,
	)
	return err
}

// CountByKBStatus counts versions with the given kb_status for a language.
func (r *VersionRepo) CountByKBStatus(ctx context.Context, languageID string, kbStatus string) (int, error) {
	var count int
	err := r.DB.GetContext(ctx, &count,
		`SELECT COUNT(*) FROM language_versions WHERE language_id = $1 AND kb_status = $2`,
		languageID, kbStatus,
	)
	return count, err
}

// UpdateStatus changes a single version's status.
func (r *VersionRepo) UpdateStatus(ctx context.Context, id string, status string) error {
	result, err := r.DB.ExecContext(ctx,
		`UPDATE language_versions SET status = $1, updated_at = NOW() WHERE id = $2`,
		status, id,
	)
	if err != nil {
		return fmt.Errorf("update version status: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("version %s not found", id)
	}
	return nil
}

// ArchiveActiveByLanguage sets all active versions for a language to "archived".
// Used when creating a new version for a "strict" language (only one active at a time).
func (r *VersionRepo) ArchiveActiveByLanguage(ctx context.Context, languageID string) error {
	result, err := r.DB.ExecContext(ctx,
		`UPDATE language_versions SET status = 'archived', updated_at = NOW()
		 WHERE language_id = $1 AND status = 'active'`,
		languageID,
	)
	if err != nil {
		return fmt.Errorf("archive active versions: %w", err)
	}
	n, _ := result.RowsAffected()
	if n > 0 {
		// Logging is handled by the caller via API middleware.
		_ = n
	}
	return nil
}
