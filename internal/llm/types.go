package llm

// LLMConfigResponse is the response for GET /llm/config.
type LLMConfigResponse struct {
	Provider     string `json:"provider"`                // "ollama", "openai", "anthropic"
	Model        string `json:"model"`
	URL          string `json:"url,omitempty"`            // only for ollama
	CredentialID string `json:"credential_id,omitempty"` // for openai/anthropic
}

// LLMConfigRequest is the request body for PUT /llm/config.
type LLMConfigRequest struct {
	Provider     string `json:"provider"`
	Model        string `json:"model,omitempty"`
	URL          string `json:"url,omitempty"`
	CredentialID string `json:"credential_id,omitempty"`
}

// LLMTestResponse is the response for POST /llm/test.
type LLMTestResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Model   string `json:"model,omitempty"`
}
