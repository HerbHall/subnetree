package llm

import (
	"encoding/json"
	"net/http"

	"github.com/HerbHall/subnetree/pkg/llm"
	"go.uber.org/zap"
)

// handleGetConfig returns the current LLM provider configuration.
//
//	@Summary		Get LLM config
//	@Description	Returns the current LLM provider configuration.
//	@Tags			llm
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200 {object} LLMConfigResponse
//	@Router			/llm/config [get]
func (m *Module) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	resp := LLMConfigResponse{
		Provider: m.cfg.Provider,
	}

	switch m.cfg.Provider {
	case "ollama":
		resp.Model = m.cfg.Ollama.Model
		resp.URL = m.cfg.Ollama.URL
	case "openai":
		resp.Model = m.cfg.OpenAI.Model
		resp.CredentialID = m.cfg.OpenAI.CredentialID
	case "anthropic":
		resp.Model = m.cfg.Anthropic.Model
		resp.CredentialID = m.cfg.Anthropic.CredentialID
	}

	writeJSON(w, http.StatusOK, resp)
}

// handlePutConfig updates the LLM provider configuration.
//
//	@Summary		Update LLM config
//	@Description	Update the LLM provider and model configuration.
//	@Tags			llm
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request body LLMConfigRequest true "LLM config"
//	@Success		200 {object} LLMConfigResponse
//	@Failure		400 {object} map[string]any
//	@Failure		500 {object} map[string]any
//	@Router			/llm/config [put]
func (m *Module) handlePutConfig(w http.ResponseWriter, r *http.Request) {
	var req LLMConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	switch req.Provider {
	case "ollama", "openai", "anthropic":
	default:
		writeError(w, http.StatusBadRequest, "provider must be ollama, openai, or anthropic")
		return
	}

	// Update config based on provider.
	switch req.Provider {
	case "ollama":
		if req.URL != "" {
			m.cfg.Ollama.URL = req.URL
		}
		if req.Model != "" {
			m.cfg.Ollama.Model = req.Model
		}
	case "openai":
		if req.CredentialID != "" {
			m.cfg.OpenAI.CredentialID = req.CredentialID
		}
		if req.Model != "" {
			m.cfg.OpenAI.Model = req.Model
		}
	case "anthropic":
		if req.CredentialID != "" {
			m.cfg.Anthropic.CredentialID = req.CredentialID
		}
		if req.Model != "" {
			m.cfg.Anthropic.Model = req.Model
		}
	}
	m.cfg.Provider = req.Provider

	// Create new provider.
	provider, err := newProvider(m.cfg, m.plugins, m.logger)
	if err != nil {
		m.logger.Error("failed to create provider", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to create provider: "+err.Error())
		return
	}
	m.provider = provider

	m.logger.Info("llm provider updated",
		zap.String("provider", req.Provider),
	)

	// Return updated config.
	m.handleGetConfig(w, r)
}

// handleTestConnection tests the current LLM provider connection.
//
//	@Summary		Test LLM connection
//	@Description	Tests connectivity to the configured LLM provider.
//	@Tags			llm
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200 {object} LLMTestResponse
//	@Failure		500 {object} LLMTestResponse
//	@Router			/llm/test [post]
func (m *Module) handleTestConnection(w http.ResponseWriter, r *http.Request) {
	if m.provider == nil {
		writeJSON(w, http.StatusOK, LLMTestResponse{
			Success: false,
			Message: "no provider configured",
		})
		return
	}

	hr, ok := m.provider.(llm.HealthReporter)
	if !ok {
		writeJSON(w, http.StatusOK, LLMTestResponse{
			Success: false,
			Message: "provider does not support health checks",
		})
		return
	}

	if err := hr.Heartbeat(r.Context()); err != nil {
		writeJSON(w, http.StatusOK, LLMTestResponse{
			Success: false,
			Message: "connection failed: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, LLMTestResponse{
		Success: true,
		Message: "connected",
		Model:   m.currentModel(),
	})
}

func (m *Module) currentModel() string {
	switch m.cfg.Provider {
	case "ollama":
		return m.cfg.Ollama.Model
	case "openai":
		return m.cfg.OpenAI.Model
	case "anthropic":
		return m.cfg.Anthropic.Model
	default:
		return ""
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":   "https://subnetree.com/problems/" + http.StatusText(status),
		"title":  http.StatusText(status),
		"status": status,
		"detail": detail,
	})
}
