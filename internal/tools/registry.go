package tools

import (
	"sync"

	"github.com/pedromelo/poly/internal/hooks"
)

var (
	registry = make(map[string]Tool)
	mu       sync.RWMutex
)

// Register adds a tool to the registry
func Register(tool Tool) {
	mu.Lock()
	defer mu.Unlock()
	registry[tool.Name()] = tool
}

// Get returns a tool by name
func Get(name string) (Tool, bool) {
	mu.RLock()
	defer mu.RUnlock()
	tool, ok := registry[name]
	return tool, ok
}

// GetAll returns all registered tools
func GetAll() []Tool {
	mu.RLock()
	defer mu.RUnlock()
	tools := make([]Tool, 0, len(registry))
	for _, t := range registry {
		tools = append(tools, t)
	}
	return tools
}

// GetNames returns all registered tool names
func GetNames() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// GetDefinitions returns tool definitions for LLM APIs
func GetDefinitions() []ToolDefinition {
	mu.RLock()
	defer mu.RUnlock()
	defs := make([]ToolDefinition, 0, len(registry))
	for _, t := range registry {
		defs = append(defs, ToolDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			InputSchema: t.Parameters(),
		})
	}
	return defs
}

// Execute runs a tool by name with given arguments
// For tools that require permission (bash, write, edit), it blocks until user approves
func Execute(name string, args map[string]interface{}) ToolResult {
	tool, ok := Get(name)
	if !ok {
		return ToolResult{
			Content: "Unknown tool: " + name,
			IsError: true,
		}
	}

	// Block banned bash commands before anything else
	if name == "bash" {
		if cmd, ok := args["command"].(string); ok && IsBashBanned(cmd) {
			return ToolResult{
				Content: "Command blocked: this command is on the banned list",
				IsError: true,
			}
		}
	}

	// Check if tool needs approval before executing
	if NeedsApproval(name, args) {
		PendingChan <- PendingApproval{
			Name:    name,
			Args:    args,
			Summary: summarizeToolCall(name, args),
		}
		// Block until TUI sends approval decision
		if !<-ApprovedChan {
			return ToolResult{
				Content: "Tool execution denied by user",
				IsError: true,
			}
		}
	}

	// Run pre-tool hooks (non-blocking)
	hooks.RunPreToolHooks(name, args)

	result := tool.Execute(args)

	// Run post-tool hooks (non-blocking)
	hooks.RunPostToolHooks(name, result.IsError, result.Content)

	return result
}

// ExecuteCall runs a ToolCall and returns the result
func ExecuteCall(call ToolCall) ToolResult {
	result := Execute(call.Name, call.Arguments)
	result.ToolUseID = call.ID
	return result
}

// Init registers all built-in tools
func Init() {
	Register(&BashTool{})
	Register(&ReadFileTool{})
	Register(&ListFilesTool{})
	Register(&WriteFileTool{})
	Register(&EditFileTool{})
	Register(&GlobTool{})
	Register(&GrepTool{})
	Register(&TodosTool{})
	Register(&WebFetchTool{})
	Register(&WebSearchTool{})
	Register(&MultieditTool{})
	Register(&ProposeDiffTool{})
	Register(&ApplyDiffTool{})
	Register(&RejectDiffTool{})
	Register(&ListDiffsTool{})
	Register(&MemoryWriteTool{})
	Register(&GitStatusTool{})
	Register(&GitDiffTool{})
	Register(&GitLogTool{})
	Register(&SystemInfoTool{})
	Register(&DelegateTaskTool{})
}
