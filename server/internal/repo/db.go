package repo

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"learncode/internal/config"
)

func NewDB(cfg config.DatabaseConfig) (*sqlx.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)
	return sqlx.Connect("postgres", dsn)
}

func RunMigrations(db *sqlx.DB, migrationsDir string) error {
	// Ensure schema_migrations table exists
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	if err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	// Read migration files
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, f := range files {
		var count int
		err := db.Get(&count, "SELECT COUNT(*) FROM schema_migrations WHERE filename = $1", f)
		if err != nil {
			return fmt.Errorf("check migration %s: %w", f, err)
		}
		if count > 0 {
			slog.Info("migration already applied, skipping", "file", f)
			continue
		}

		content, err := os.ReadFile(filepath.Join(migrationsDir, f))
		if err != nil {
			return fmt.Errorf("read migration file %s: %w", f, err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin tx for %s: %w", f, err)
		}

		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("execute migration %s: %w", f, err)
		}

		if _, err := tx.Exec("INSERT INTO schema_migrations (filename) VALUES ($1)", f); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %s: %w", f, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", f, err)
		}

		slog.Info("migration applied", "file", f)
	}

	return nil
}
