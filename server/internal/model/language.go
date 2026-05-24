package model

import (
	"encoding/json"
	"time"
)

type Language struct {
	ID                 string          `db:"id"                  json:"id"`
	Name               string          `db:"name"                json:"name"`
	Slug               string          `db:"slug"                json:"slug"`
	Icon               string          `db:"icon"                json:"icon"`
	CompatibilityModel string          `db:"compatibility_model" json:"compatibility_model"`
	SourceURLs         json.RawMessage `db:"source_urls"         json:"source_urls"`
	CreatedAt          time.Time       `db:"created_at"          json:"created_at"`
}

type LanguageVersion struct {
	ID                 string          `db:"id"                   json:"id"`
	LanguageID         string          `db:"language_id"          json:"language_id"`
	Version            string          `db:"version"              json:"version"`
	Status             string          `db:"status"               json:"status"`
	RuntimeConfig      json.RawMessage `db:"runtime_config"       json:"runtime_config"`
	SourceURLs         json.RawMessage `db:"source_urls"          json:"source_urls"`
	LastVersionCheckAt *time.Time      `db:"last_version_check_at" json:"last_version_check_at,omitempty"`
	Initialized        bool            `db:"initialized"          json:"initialized"`
	CreatedAt          time.Time       `db:"created_at"           json:"created_at"`
	UpdatedAt          time.Time       `db:"updated_at"           json:"updated_at"`
}
