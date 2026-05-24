package api

import (
	"encoding/json"
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
}

// Routes registers execution routes under /api/v1.
func (h *ExecuteHandler) Routes(r chi.Router) {
	r.Post("/execute", h.Run)
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

	// Look up the version to get the associated language_id.
	version, err := h.VersionRepo.GetByID(r.Context(), input.VersionID)
	if err != nil {
		RespondError(w, http.StatusNotFound, "version not found")
		return
	}

	// Look up the language to get the slug (used to select interpreter).
	lang, err := h.LanguageRepo.GetByID(r.Context(), version.LanguageID)
	if err != nil {
		RespondError(w, http.StatusNotFound, "language not found for version")
		return
	}

	result, err := executor.Execute(r.Context(), executor.Request{
		Language: lang.Slug,
		Code:     input.Code,
	})
	if err != nil {
		RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	RespondJSON(w, http.StatusOK, result)
}
