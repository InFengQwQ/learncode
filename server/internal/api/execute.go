package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"learncode/internal/executor"
	"learncode/internal/repo"
)

// ExecuteHandler handles code execution requests.
type ExecuteHandler struct {
	VersionRepo  *repo.VersionRepo
	LanguageRepo *repo.LanguageRepo
	Executor     *executor.Executor
}

// Routes registers execution routes under /api/v1.
func (h *ExecuteHandler) Routes(r chi.Router) {
	r.Post("/", h.Run)
}

func (h *ExecuteHandler) Run(w http.ResponseWriter, r *http.Request) {
	var input struct {
		VersionID string `json:"version_id"`
		Code      string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.VersionID == "" {
		RespondError(w, http.StatusBadRequest, "version_id is required")
		return
	}
	if strings.TrimSpace(input.Code) == "" {
		RespondError(w, http.StatusBadRequest, "code is required")
		return
	}

	version, err := h.VersionRepo.GetByID(r.Context(), input.VersionID)
	if err != nil {
		RespondError(w, http.StatusNotFound, "version not found")
		return
	}

	lang, err := h.LanguageRepo.GetByID(r.Context(), version.LanguageID)
	if err != nil {
		RespondError(w, http.StatusNotFound, "language not found for version")
		return
	}

	// Activation gate: refuse code execution for inactive languages.
	// Knowledge base must be built before the language can be used.
	if lang.Status != "active" {
		RespondError(w, http.StatusForbidden,
			fmt.Sprintf("language %q is not active yet — knowledge base must be built first", lang.Name))
		return
	}

	// Parse the version's runtime_config. If it's empty or incomplete, fall back
	// to the default config derived from the language slug.
	rc, err := executor.ParseRuntimeConfig(version.RuntimeConfig)
	if err != nil || !rc.IsComplete() {
		rc = executor.DefaultRuntimeConfig(lang.Slug)
	}

	if !rc.IsComplete() {
		RespondError(w, http.StatusBadRequest,
			fmt.Sprintf("未找到 %q 的运行时配置 — 请在版本设置中手动配置 runtime_config（需要 interpreter、extension、run_cmd）", lang.Slug))
		return
	}

	// Use the Executor instance (Docker-first with os/exec fallback).
	result, err := h.Executor.Execute(r.Context(), rc, input.Code)
	if err != nil {
		RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	RespondJSON(w, http.StatusOK, result)
}
