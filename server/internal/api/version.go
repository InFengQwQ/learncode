package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"learncode/internal/model"
	"learncode/internal/repo"
	"learncode/internal/service"
)

type VersionHandler struct {
	Svc           *service.VersionService
	Init          *service.InitService
	KnowledgeRepo *repo.KnowledgeRepo
	KBBuild       *service.KBBuildService
}

func (h *VersionHandler) Routes(r chi.Router) {
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{versionId}", h.Get)
	r.Post("/{versionId}/initialize", h.Initialize)
	r.Get("/{versionId}/knowledge", h.Knowledge)
	r.Post("/{versionId}/build-knowledge", h.BuildKnowledge)
}

func (h *VersionHandler) List(w http.ResponseWriter, r *http.Request) {
	languageID := chi.URLParam(r, "id")
	versions, err := h.Svc.ListByLanguageID(r.Context(), languageID)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, "failed to list versions")
		return
	}
	RespondJSON(w, http.StatusOK, versions)
}

func (h *VersionHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "versionId")
	v, err := h.Svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "version not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, "failed to get version")
		return
	}
	RespondJSON(w, http.StatusOK, v)
}

func (h *VersionHandler) Create(w http.ResponseWriter, r *http.Request) {
	languageID := chi.URLParam(r, "id")

	var input struct {
		Version string `json:"version"`
		Status  string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.Version == "" {
		RespondError(w, http.StatusBadRequest, "version is required")
		return
	}
	if input.Status == "" {
		input.Status = "active"
	}

	v := &model.LanguageVersion{
		LanguageID: languageID,
		Version:    input.Version,
		Status:     input.Status,
	}
	if err := h.Svc.Create(r.Context(), v); err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	RespondJSON(w, http.StatusCreated, v)
}

func (h *VersionHandler) Initialize(w http.ResponseWriter, r *http.Request) {
	versionID := chi.URLParam(r, "versionId")

	result, err := h.Init.Initialize(r.Context(), versionID)
	if err != nil {
		RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	RespondJSON(w, http.StatusOK, result)
}

// Knowledge returns all knowledge entries for a version (shared language-level + version-specific).
func (h *VersionHandler) Knowledge(w http.ResponseWriter, r *http.Request) {
	versionID := chi.URLParam(r, "versionId")

	v, err := h.Svc.GetByID(r.Context(), versionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "version not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, "failed to get version")
		return
	}

	shared, err := h.KnowledgeRepo.ListSharedByLanguage(r.Context(), v.LanguageID)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, "failed to get shared knowledge")
		return
	}
	if shared == nil {
		shared = []model.KnowledgeEntry{}
	}

	private, err := h.KnowledgeRepo.ListByVersion(r.Context(), versionID)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, "failed to get version knowledge")
		return
	}
	if private == nil {
		private = []model.KnowledgeEntry{}
	}

	RespondJSON(w, http.StatusOK, map[string]interface{}{
		"shared":  shared,
		"private": private,
	})
}

// SetStatus changes a version's status between "active" and "archived".
// For strict languages, activating a new version archives the old one automatically.
func (h *VersionHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	versionID := chi.URLParam(r, "versionId")

	var input struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.Status != "active" && input.Status != "archived" {
		RespondError(w, http.StatusBadRequest, "status must be 'active' or 'archived'")
		return
	}

	v, err := h.Svc.SetStatus(r.Context(), versionID, input.Status)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondJSON(w, http.StatusOK, v)
}

// BuildKnowledge triggers asynchronous knowledge base construction for a version.
// It starts the build in a goroutine and returns 202 Accepted immediately.
func (h *VersionHandler) BuildKnowledge(w http.ResponseWriter, r *http.Request) {
	versionID := chi.URLParam(r, "versionId")

	v, err := h.Svc.GetByID(r.Context(), versionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "version not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, "failed to get version")
		return
	}

	if v.KBStatus != "" && v.KBStatus != "pending" && v.KBStatus != "failed" {
		RespondError(w, http.StatusConflict, "knowledge base build already in progress or complete")
		return
	}

	if !v.Initialized {
		RespondError(w, http.StatusBadRequest, "version must be initialized before building knowledge base")
		return
	}

	if h.KBBuild == nil {
		RespondError(w, http.StatusServiceUnavailable, "knowledge build service not available")
		return
	}

	// Run build asynchronously — use background context since the request
	// context is canceled when the HTTP handler returns (202 Accepted).
	go func() {
		if err := h.KBBuild.Build(context.Background(), versionID); err != nil {
			slog.Error("knowledge base build failed", "version_id", versionID, "error", err)
		}
	}()

	RespondJSON(w, http.StatusAccepted, map[string]interface{}{
		"status":     "building",
		"version_id": versionID,
	})
}