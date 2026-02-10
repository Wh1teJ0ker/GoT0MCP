package abi

// Role defines the role of the message sender
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Frame represents a unified conversation frame
type Frame struct {
	Role Role   `json:"role"`
	Text string `json:"text"`

	ToolCalls  []ToolCall      `json:"tool_calls,omitempty"`  // assistant initiated
	ToolResult *ToolResult     `json:"tool_result,omitempty"` // tool response
	Meta       map[string]any  `json:"meta,omitempty"`        // extensible: attachments, source, trace
}

// ToolSpec defines a tool's structure
type ToolSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"` // JSON Schema
	OutputSchema map[string]any `json:"output_schema,omitempty"` // Optional
}

// ToolCall represents a call to a tool
type ToolCall struct {
	ID   string         `json:"id"`   // Required
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

// ToolResult represents the result of a tool call
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Name       string `json:"name"`
	Content    any    `json:"content"` // string or object
	IsError    bool   `json:"is_error"`
}
