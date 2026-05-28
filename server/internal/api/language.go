package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"learncode/internal/service"
)

type LanguageHandler struct {
	Svc     *service.LanguageService
	InitSvc *service.LanguageInitService
}

func (h *LanguageHandler) Routes(r chi.Router) {
	r.Get("/", h.List)
	r.Get("/{id}", h.Get)
	r.Post("/", h.Create)
	r.Post("/init", h.Init)
	r.Delete("/{id}", h.Delete)
	r.Post("/{id}/research", h.Research)
		r.Post("/{id}/discover-versions", h.DiscoverVersions)
}

func (h *LanguageHandler) List(w http.ResponseWriter, r *http.Request) {
	langs, err := h.Svc.List(r.Context())
	if err != nil {
		RespondError(w, http.StatusInternalServerError, "failed to list languages")
		return
	}
	RespondJSON(w, http.StatusOK, langs)
}

func (h *LanguageHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	lang, err := h.Svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "language not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, "failed to get language")
		return
	}
	RespondJSON(w, http.StatusOK, lang)
}

func (h *LanguageHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.Svc.Delete(r.Context(), id); err != nil {
		RespondError(w, http.StatusInternalServerError, "failed to delete language")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *LanguageHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input service.CreateLanguageInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	lang, err := h.Svc.Create(r.Context(), input)
	if err != nil {
		RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	RespondJSON(w, http.StatusCreated, lang)
}

func (h *LanguageHandler) Init(w http.ResponseWriter, r *http.Request) {
	step := r.URL.Query().Get("step")
	if h.InitSvc == nil {
		RespondError(w, http.StatusInternalServerError, "init service not configured")
		return
	}

	switch step {
	case "query":
		var input struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if input.Name == "" {
			RespondError(w, http.StatusBadRequest, "name is required")
			return
		}

		if r.URL.Query().Get("stream") == "true" {
			pw, pwErr := NewProgressWriter(w)
			if pwErr != nil {
				RespondError(w, http.StatusInternalServerError, "streaming not supported")
				return
			}
			streamResult, streamErr := h.InitSvc.QueryWithProgress(r.Context(), input.Name,
				func(stepName, status, msg string) {
					pw.Emit(ProgressStep{Step: stepName, Status: status, Message: msg})
				})
			if streamErr != nil {
				pw.Emit(ProgressStep{Step: "fatal", Status: "error", Message: streamErr.Error()})
				return
			}
			b, _ := json.Marshal(APIResponse{OK: true, Data: streamResult})
			w.Write(b)
			w.Write([]byte("\n"))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			return
		}

		result, err := h.InitSvc.Query(r.Context(), input.Name)
		if err != nil {
			RespondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		RespondJSON(w, http.StatusOK, result)

	case "confirm":
		var input service.InitConfirmInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		result, err := h.InitSvc.Confirm(r.Context(), input)
		if err != nil {
			RespondError(w, http.StatusBadRequest, err.Error())
			return
		}
		RespondJSON(w, http.StatusCreated, result)

	default:
		RespondError(w, http.StatusBadRequest, "step must be 'query' or 'confirm'")
	}
}

func (h *LanguageHandler) Research(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if h.InitSvc == nil {
		RespondError(w, http.StatusInternalServerError, "init service not configured")
		return
	}

	lang, err := h.Svc.GetByID(r.Context(), id)
	if err != nil {
		RespondError(w, http.StatusNotFound, "language not found")
		return
	}

	result, err := h.InitSvc.Research(r.Context(), lang)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	researchJSON, _ := json.Marshal(result)
	now := time.Now()
	sourceJSON, _ := json.Marshal(lang.SourceURLs)
	if err := h.Svc.UpdateFromResearch(r.Context(), lang.ID, researchJSON, now, sourceJSON); err != nil {
		slog.Warn("failed to persist research result", "error", err)
	}

	RespondJSON(w, http.StatusOK, result)
}

func (h *LanguageHandler) DiscoverVersions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if h.InitSvc == nil {
		RespondError(w, http.StatusInternalServerError, "init service not configured")
		return
	}

	if r.URL.Query().Get("stream") == "true" {
		pw, pwErr := NewProgressWriter(w)
		if pwErr != nil {
			RespondError(w, http.StatusInternalServerError, "streaming not supported")
			return
		}
		versions, err := h.InitSvc.DiscoverHistoricalVersions(r.Context(), id,
			func(stepName, status, msg string) {
				pw.Emit(ProgressStep{Step: stepName, Status: status, Message: msg})
			})
		if err != nil {
			pw.Emit(ProgressStep{Step: "fatal", Status: "error", Message: err.Error()})
			return
		}
		b, _ := json.Marshal(APIResponse{OK: true, Data: versions})
		w.Write(b)
		w.Write([]byte("\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		return
	}

	versions, err := h.InitSvc.DiscoverHistoricalVersions(r.Context(), id, nil)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	RespondJSON(w, http.StatusOK, versions)
}
