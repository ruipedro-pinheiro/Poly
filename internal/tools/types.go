package tools

import (
	"sync"

	"github.com/pedromelo/poly/internal/types"
)

// Type aliases — these were previously identical copies of the types in internal/types.
// Using aliases preserves backward compatibility: existing code using tools.ToolCall
// continues to work without changes.
type ToolCall = types.ToolCall
type ToolResult = types.ToolResult
type ToolDefinition = types.ToolDefinition

// Tool interface that all tools must implement
type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]interface{} // JSON Schema
	Execute(args map[string]interface{}) ToolResult
}

// SubStreamEvent represents a streaming event from a sub-provider.
// Subset of llm.StreamEvent — only the fields needed for sub-conversations.
type SubStreamEvent struct {
	Type       string // "content", "tool_use", "tool_result", "done", "error"
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
var (
	subProvider   SubProvider
	subProviderMu sync.RWMutex
)

// SetSubProvider injects the sub-provider for delegate_task to use.
func SetSubProvider(sp SubProvider) {
	subProviderMu.Lock()
	defer subProviderMu.Unlock()
	subProvider = sp
}

// GetSubProvider returns the current sub-provider (may be nil).
func GetSubProvider() SubProvider {
	subProviderMu.RLock()
	defer subProviderMu.RUnlock()
	return subProvider
}
