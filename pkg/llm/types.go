package llm

// Message represents a single message in a chat conversation.
type Message struct {
	Role    string `json:"role"`    // One of RoleSystem, RoleUser, RoleAssistant.
	Content string `json:"content"`
}

// Role constants for the Message.Role field.
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

// Response contains the LLM's generated text and metadata.
type Response struct {
	Content string `json:"content"` // Generated text.
	Model   string `json:"model"`   // Model that produced this response.
	Usage   Usage  `json:"usage"`   // Token consumption stats.
	Done    bool   `json:"done"`    // True if generation completed (false if truncated).
}

// Usage tracks token consumption for a single LLM call.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
