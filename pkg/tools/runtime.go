package tools

import (
	"context"

	"github.com/Wh1teJ0ker/GoT0MCP/pkg/abi"
)

// Runtime defines the interface for managing and invoking tools
type Runtime interface {
	// ListTools returns the list of available tools for a session
	ListTools(ctx context.Context) ([]abi.ToolSpec, error)

	// InvokeTool executes a tool call and returns the result
	InvokeTool(ctx context.Context, call abi.ToolCall) (*abi.ToolResult, error)
}

// SimpleRuntime is a basic implementation of Runtime
type SimpleRuntime struct {
	tools map[string]func(context.Context, map[string]any) (any, error)
	specs []abi.ToolSpec
}

func NewSimpleRuntime() *SimpleRuntime {
	return &SimpleRuntime{
		tools: make(map[string]func(context.Context, map[string]any) (any, error)),
		specs: make([]abi.ToolSpec, 0),
	}
}

func (r *SimpleRuntime) Register(spec abi.ToolSpec, handler func(context.Context, map[string]any) (any, error)) {
	r.specs = append(r.specs, spec)
	r.tools[spec.Name] = handler
}

func (r *SimpleRuntime) ListTools(ctx context.Context) ([]abi.ToolSpec, error) {
	return r.specs, nil
}

func (r *SimpleRuntime) InvokeTool(ctx context.Context, call abi.ToolCall) (*abi.ToolResult, error) {
	handler, ok := r.tools[call.Name]
	if !ok {
		return &abi.ToolResult{
			ToolCallID: call.ID,
			Name:       call.Name,
			Content:    "Tool not found",
			IsError:    true,
		}, nil
	}

	// TODO: Add schema validation here (Section 7.2)

	res, err := handler(ctx, call.Args)
	if err != nil {
		return &abi.ToolResult{
			ToolCallID: call.ID,
			Name:       call.Name,
			Content:    err.Error(),
			IsError:    true,
		}, nil
	}

	return &abi.ToolResult{
		ToolCallID: call.ID,
		Name:       call.Name,
		Content:    res,
		IsError:    false,
	}, nil
}
