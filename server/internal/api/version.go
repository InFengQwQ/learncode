package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"learncode/internal/model"
	"learncode/internal/service"
)

type VersionHandler struct {
	Svc *service.VersionService
}

func (h *VersionHandler) Routes(r chi.Router) {
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{versionId}", h.Get)
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
