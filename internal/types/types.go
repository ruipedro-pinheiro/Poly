// Package types defines shared domain types used across tools, llm, and mcp packages.
// This eliminates type duplication between packages and the manual conversion
// that was previously needed in subprovider.go.
package types

// ToolCall represents a request from the LLM to execute a tool.
type ToolCall struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolResult represents the output of a tool execution.
type ToolResult struct {
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error,omitempty"`
}

// ToolDefinition describes a tool available for the LLM to use.
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}
