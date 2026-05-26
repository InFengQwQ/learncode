package model

import (
	"encoding/json"
	"time"
)

type KnowledgeEntry struct {
	ID         string          `db:"id"          json:"id"`
	LanguageID string          `db:"language_id" json:"language_id"`
	VersionID  *string         `db:"version_id"  json:"version_id"` // NULL = core/idiom (shared across versions)
	Scope      string          `db:"scope"       json:"scope"`      // "core" | "version" | "idiom"
	Category   string          `db:"category"    json:"category"`   // "factual" | "normative"
	Topic      string          `db:"topic"       json:"topic"`
	Content    json.RawMessage `db:"content"     json:"content"`
	Source     string          `db:"source"      json:"source"` // "llm" | "env" | "manual"
	CreatedAt  time.Time       `db:"created_at"  json:"created_at"`
	UpdatedAt  time.Time       `db:"updated_at"  json:"updated_at"`
}