package abi

import "context"

// LLMRequest represents a unified model request
type LLMRequest struct {
	Frames []Frame    `json:"frames"`
	Tools  []ToolSpec `json:"tools"`

	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
	TopP        float64 `json:"top_p"`

	ResponseMode string         `json:"response_mode"` // "text" | "json"
	JsonSchema   map[string]any `json:"json_schema,omitempty"`

	Stream bool `json:"stream"`
}

// LLMEvent represents a stream event from the model
type LLMEvent struct {
	Type      string    `json:"type"` // "text_delta" | "tool_call" | "done" | "error"
	TextDelta string    `json:"text_delta,omitempty"`
	ToolCall  *ToolCall `json:"tool_call,omitempty"`
	Err       string    `json:"error,omitempty"`
}

// ProviderCaps defines the capabilities of a provider
type ProviderCaps struct {
	SupportsToolCalls         bool
	SupportsParallelToolCalls bool
	SupportsJsonSchema        bool
	MaxContextTokens          int
}

// LLMProvider is the interface that all model providers must implement
type LLMProvider interface {
	Generate(ctx context.Context, req LLMRequest) (<-chan LLMEvent, error)
	Caps() ProviderCaps
	Name() string
}
