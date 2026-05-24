package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

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
