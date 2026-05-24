package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"

	"learncode/internal/api"
	"learncode/internal/config"
	"learncode/internal/llm"
	"learncode/internal/repo"
	"learncode/internal/scraper"
	"learncode/internal/service"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	db, err := repo.NewDB(cfg.Database)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := repo.RunMigrations(db, "migrations"); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	r := chi.NewRouter()
	r.Use(api.Recovery)
	r.Use(api.Logging)
	r.Use(api.CORS(cfg.Server.CORSOrigins))

	// repos
	langRepo := &repo.LanguageRepo{DB: db}
	versionRepo := &repo.VersionRepo{DB: db}

	// services
	langSvc := &service.LanguageService{Repo: langRepo}
	versionSvc := &service.VersionService{Repo: versionRepo}

	// llm
	llmSvc, err := llm.NewService(cfg.LLM)
	if err != nil {
		slog.Warn("llm service not available", "error", err)
	}

	// init service
	var initSvc *service.LanguageInitService
	if llmSvc != nil {
		initSvc = &service.LanguageInitService{
			LangSvc:   langSvc,
			LLM:       llmSvc,
			PromptDir: "prompts",
			Scraper:   scraper.NewClient(),
		}
	}

	// handlers
	langHandler := &api.LanguageHandler{Svc: langSvc, InitSvc: initSvc}
	versionHandler := &api.VersionHandler{Svc: versionSvc}
	configHandler := &api.ConfigHandler{Cfg: cfg, Path: *configPath}

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/languages", langHandler.Routes)
		r.Route("/languages/{id}/versions", versionHandler.Routes)
		r.Route("/versions", func(r chi.Router) {
			r.Get("/{versionId}", versionHandler.Get)
		})
		r.Route("/config", configHandler.Routes)
	})

	slog.Info("server starting", "port", cfg.Server.Port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Server.Port), r); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
