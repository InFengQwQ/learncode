package repo

import (
	"context"

	"github.com/jmoiron/sqlx"

	"learncode/internal/model"
)

type KnowledgeRepo struct {
	DB *sqlx.DB
}

const knowledgeCols = `id, language_id, version_id, scope, category, topic, content, source, created_at, updated_at`

func (r *KnowledgeRepo) ListByLanguage(ctx context.Context, languageID string) ([]model.KnowledgeEntry, error) {
	var entries []model.KnowledgeEntry
	err := r.DB.SelectContext(ctx, &entries,
		`SELECT `+knowledgeCols+` FROM knowledge_entries
		 WHERE language_id = $1
		 ORDER BY scope, category, topic`,
		languageID,
	)
	return entries, err
}

func (r *KnowledgeRepo) ListSharedByLanguage(ctx context.Context, languageID string) ([]model.KnowledgeEntry, error) {
	var entries []model.KnowledgeEntry
	err := r.DB.SelectContext(ctx, &entries,
		`SELECT `+knowledgeCols+` FROM knowledge_entries
		 WHERE language_id = $1 AND scope IN ('core', 'idiom') AND version_id IS NULL
		 ORDER BY scope, category, topic`,
		languageID,
	)
	return entries, err
}

func (r *KnowledgeRepo) ListByVersion(ctx context.Context, versionID string) ([]model.KnowledgeEntry, error) {
	var entries []model.KnowledgeEntry
	err := r.DB.SelectContext(ctx, &entries,
		`SELECT `+knowledgeCols+` FROM knowledge_entries
		 WHERE version_id = $1 AND scope = 'version'
		 ORDER BY category, topic`,
		versionID,
	)
	return entries, err
}

// ListByScope returns all entries for a language filtered by scope.
// scope must be one of: "core", "version", "idiom".
func (r *KnowledgeRepo) ListByScope(ctx context.Context, languageID string, scope string) ([]model.KnowledgeEntry, error) {
	var entries []model.KnowledgeEntry
	err := r.DB.SelectContext(ctx, &entries,
		`SELECT `+knowledgeCols+` FROM knowledge_entries
		 WHERE language_id = $1 AND scope = $2
		 ORDER BY category, topic`,
		languageID, scope,
	)
	return entries, err
}

func (r *KnowledgeRepo) Create(ctx context.Context, entry *model.KnowledgeEntry) error {
	return r.DB.GetContext(ctx, entry,
		`INSERT INTO knowledge_entries (language_id, version_id, scope, category, topic, content, source)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING `+knowledgeCols,
		entry.LanguageID, entry.VersionID, entry.Scope, entry.Category, entry.Topic, entry.Content, entry.Source,
	)
}

func (r *KnowledgeRepo) DeleteByLanguage(ctx context.Context, languageID string) error {
	_, err := r.DB.ExecContext(ctx,
		"DELETE FROM knowledge_entries WHERE language_id = $1",
		languageID,
	)
	return err
}

func (r *KnowledgeRepo) CountByLanguage(ctx context.Context, languageID string) (int, error) {
	var count int
	err := r.DB.GetContext(ctx, &count,
		"SELECT COUNT(*) FROM knowledge_entries WHERE language_id = $1",
		languageID,
	)
	return count, err
}