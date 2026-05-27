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
	"learncode/internal/docker"
	"learncode/internal/executor"
	"learncode/internal/llm"
	"learncode/internal/repo"
	"learncode/internal/scraper"
	"learncode/internal/service"
)

func newScraperClient() *scraper.Client {
	if p := os.Getenv("WIKI_PROXY"); p != "" {
		return scraper.NewClientWithProxy(p)
	}
	return scraper.NewClient()
}

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

	// docker client
	dockerClient, err := docker.NewClient()
	if err != nil {
		slog.Warn("docker not available, falling back to host execution", "error", err)
	}
	if dockerClient != nil && dockerClient.Available() {
		slog.Info("docker client initialized successfully")
	} else {
		slog.Warn("docker daemon not reachable, using host execution fallback")
	}

	// repos
	langRepo := &repo.LanguageRepo{DB: db}
	versionRepo := &repo.VersionRepo{DB: db}
	knowledgeRepo := &repo.KnowledgeRepo{DB: db}

	// services
	langSvc := &service.LanguageService{Repo: langRepo}
	versionSvc := &service.VersionService{Repo: versionRepo, LangRepo: langRepo}

	// executor (Docker-first, os/exec fallback)
	exec := executor.NewExecutor(dockerClient)

	// llm
	llmSvc, err := llm.NewService(cfg.LLM)
	if err != nil {
		slog.Warn("llm service not available", "error", err)
	}

	// version init service (Docker-based environment initialization)
	initVersionSvc := &service.InitService{
		VersionRepo: versionRepo,
		LangRepo:    langRepo,
		Docker:      dockerClient,
	}

	// kb build service — created before initSvc so it can be wired in.
	var kbBuildSvc *service.KBBuildService
	if llmSvc != nil {
		explorer := &service.KBExplorer{
			LLM:       llmSvc,
			Executor:  exec,
			PromptDir: "prompts",
		}
		kbBuildSvc = &service.KBBuildService{
			VersionRepo:   versionRepo,
			LangRepo:      langRepo,
			KnowledgeRepo: knowledgeRepo,
			LLM:           llmSvc,
			Executor:      exec,
			PromptDir:     "prompts",
			VersionSvc:    versionSvc,
			Explorer:      explorer,
		}
	}

	// init service
	var initSvc *service.LanguageInitService
	if llmSvc != nil {
		initSvc = &service.LanguageInitService{
			LangSvc:        langSvc,
			VersionSvc:     versionSvc,
			LLM:            llmSvc,
			PromptDir:      "prompts",
			Scraper:        newScraperClient(),
			InitVersionSvc: initVersionSvc,
			KBBuildSvc:     kbBuildSvc,
		}
	}

	// handlers
	langHandler := &api.LanguageHandler{Svc: langSvc, InitSvc: initSvc}
	versionHandler := &api.VersionHandler{Svc: versionSvc, Init: initVersionSvc, KnowledgeRepo: knowledgeRepo, KBBuild: kbBuildSvc}
	configHandler := &api.ConfigHandler{Cfg: cfg, Path: *configPath, LLMSvc: llmSvc}
	executeHandler := &api.ExecuteHandler{VersionRepo: versionRepo, LanguageRepo: langRepo, Executor: exec}

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/languages", langHandler.Routes)
		r.Route("/languages/{id}/versions", versionHandler.Routes)
		r.Route("/versions", func(r chi.Router) {
			r.Get("/{versionId}", versionHandler.Get)
			r.Post("/{versionId}/initialize", versionHandler.Initialize)
			r.Get("/{versionId}/knowledge", versionHandler.Knowledge)
			r.Post("/{versionId}/build-knowledge", versionHandler.BuildKnowledge)
			r.Patch("/{versionId}/status", versionHandler.SetStatus)
		})
		r.Route("/config", configHandler.Routes)
		r.Route("/execute", executeHandler.Routes)
	})

	slog.Info("server starting", "port", cfg.Server.Port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Server.Port), r); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
