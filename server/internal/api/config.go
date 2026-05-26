package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"learncode/internal/config"
	"learncode/internal/llm"
)

type ConfigHandler struct {
	Cfg    *config.Config
	Path   string
	LLMSvc *llm.Service
}

func (h *ConfigHandler) Routes(r chi.Router) {
	r.Get("/llm", h.GetLLM)
	r.Put("/llm", h.PutLLM)
}

func (h *ConfigHandler) GetLLM(w http.ResponseWriter, r *http.Request) {
	RespondJSON(w, http.StatusOK, h.Cfg.LLM.ToResponse())
}

func (h *ConfigHandler) PutLLM(w http.ResponseWriter, r *http.Request) {
	var input config.LLMConfigResponse
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(input.Providers) == 0 {
		RespondError(w, http.StatusBadRequest, "at least one provider is required")
		return
	}

	providers := make([]config.LLMProviderConfig, len(input.Providers))
	for i, p := range input.Providers {
		providers[i] = config.LLMProviderConfig{
			Name:     p.Name,
			Endpoint: p.Endpoint,
			Models:   p.Models,
			APIKey:   resolveKey(p, h.Cfg.LLM.Providers),
		}
	}

	prevLLM := h.Cfg.LLM // rollback snapshot

	h.Cfg.LLM.Default = input.Default
	h.Cfg.LLM.Providers = providers

	if err := config.Save(h.Path, h.Cfg); err != nil {
		h.Cfg.LLM = prevLLM
		RespondError(w, http.StatusInternalServerError, "failed to save config")
		return
	}

	// Hot-reload the LLM service so new settings take effect immediately
	if h.LLMSvc != nil {
		if err := h.LLMSvc.Reload(h.Cfg.LLM); err != nil {
			slog.Warn("llm reload failed after config save", "error", err)
		} else {
			slog.Info("llm service reloaded", "default", h.Cfg.LLM.Default, "providers", len(h.Cfg.LLM.Providers))
		}
	} else {
		slog.Warn("llm service is nil, cannot reload")
	}

	RespondJSON(w, http.StatusOK, h.Cfg.LLM.ToResponse())
}

// resolveKey returns the real API key for a provider update.
// If the user provided a plain key (not masked), use it directly.
// If masked, try to match by name first, then by masked fingerprint
// (handles renames). Falls back to empty string for unmatched masked keys.
func resolveKey(input config.LLMProviderResponse, existing []config.LLMProviderConfig) string {
	if !isMasked(input.APIKey) {
		return input.APIKey
	}
	// Try exact name match first.
	for i := range existing {
		if existing[i].Name == input.Name {
			return existing[i].APIKey
		}
	}
	// Provider was renamed — search by masked fingerprint.
	masked := input.APIKey
	for i := range existing {
		if config.MaskKey(existing[i].APIKey) == masked {
			return existing[i].APIKey
		}
	}
	return ""
}

func isMasked(key string) bool {
	if key == "" || key == "****" {
		return true
	}
	for i := 0; i < len(key)-2; i++ {
		if key[i] == '.' && key[i+1] == '.' && key[i+2] == '.' {
			return true
		}
	}
	return false
}
