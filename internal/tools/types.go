package tools

// ToolCall represents a tool invocation from an LLM
type ToolCall struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error,omitempty"`
}

// Tool interface that all tools must implement
type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]interface{} // JSON Schema
	Execute(args map[string]interface{}) ToolResult
}

// ToolDefinition for sending to LLMs (Anthropic format)
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// SubStreamEvent represents a streaming event from a sub-provider.
// Mirrors llm.StreamEvent without importing the llm package.
type SubStreamEvent struct {
	Type       string      // "content", "tool_use", "tool_result", "done", "error"
	Content    string
	ToolCall   *ToolCall
	ToolResult *ToolResult
	Error      error
}

// SubProvider is a minimal interface for spawning sub-conversations.
// Defined here (in tools) to avoid circular imports with llm.
type SubProvider interface {
	// SubStream starts a streaming sub-conversation with the given system prompt,
	// messages, and tool definitions. Returns a channel of SubStreamEvents.
	SubStream(systemPrompt string, messages []SubMessage, tools []ToolDefinition) <-chan SubStreamEvent
}

// SubMessage is a simplified message for sub-conversations.
type SubMessage struct {
	Role    string // "user", "assistant"
	Content string
}

// subProvider holds the injected SubProvider instance.
var subProvider SubProvider

// SetSubProvider injects the sub-provider for delegate_task to use.
func SetSubProvider(sp SubProvider) {
	subProvider = sp
}

// GetSubProvider returns the current sub-provider (may be nil).
func GetSubProvider() SubProvider {
	return subProvider
}
