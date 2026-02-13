package tools

import (
	"fmt"
	"strings"
)

// DelegateTaskTool spawns a sub-conversation for focused work
type DelegateTaskTool struct{}

func (t *DelegateTaskTool) Name() string { return "delegate_task" }

func (t *DelegateTaskTool) Description() string {
	return "Delegate a focused sub-task to a helper AI. The helper has read-only tools " +
		"(read_file, list_files, glob, grep, git_status, git_diff, git_log). " +
		"Use this for research, code review, or information gathering that would clutter the main conversation."
}

func (t *DelegateTaskTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"task": map[string]interface{}{
				"type":        "string",
				"description": "Description of the task to delegate",
			},
			"context": map[string]interface{}{
				"type":        "string",
				"description": "Additional context to provide to the helper",
			},
			"max_turns": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum conversation turns for the sub-agent (default: 5)",
			},
		},
		"required": []string{"task"},
	}
}

// readOnlyToolNames lists the tools available to sub-agents
var readOnlyToolNames = []string{
	"read_file", "list_files", "glob", "grep",
	"git_status", "git_diff", "git_log",
}

// getReadOnlyTools returns ToolDefinitions for read-only tools only
func getReadOnlyTools() []ToolDefinition {
	var defs []ToolDefinition
	for _, name := range readOnlyToolNames {
		tool, ok := Get(name)
		if !ok {
			continue
		}
		defs = append(defs, ToolDefinition{
			Name:        tool.Name(),
			Description: tool.Description(),
			InputSchema: tool.Parameters(),
		})
	}
	return defs
}

// executeReadOnly executes a tool only if it's in the read-only list
func executeReadOnly(name string, args map[string]interface{}) ToolResult {
	for _, allowed := range readOnlyToolNames {
		if name == allowed {
			tool, ok := Get(name)
			if !ok {
				return ToolResult{Content: "Tool not found: " + name, IsError: true}
			}
			return tool.Execute(args)
		}
	}
	return ToolResult{
		Content: "Tool not allowed in sub-agent: " + name + ". Only read-only tools are available.",
		IsError: true,
	}
}

func (t *DelegateTaskTool) Execute(args map[string]interface{}) ToolResult {
	sp := GetSubProvider()
	if sp == nil {
		return ToolResult{
			Content: "Sub-provider not configured. Cannot delegate tasks.",
			IsError: true,
		}
	}

	task, _ := args["task"].(string)
	if task == "" {
		return ToolResult{Content: "Missing required parameter: task", IsError: true}
	}

	contextStr, _ := args["context"].(string)
	maxTurns := 5
	if mt, ok := args["max_turns"].(float64); ok && mt > 0 {
		maxTurns = int(mt)
		if maxTurns > 20 {
			maxTurns = 20
		}
	}

	// Build the user message
	userContent := task
	if contextStr != "" {
		userContent = fmt.Sprintf("Task: %s\n\nContext: %s", task, contextStr)
	}

	systemPrompt := "You are a helper agent inside Poly, a multi-AI terminal tool. " +
		"Complete the given task concisely. You have read-only tools available: " +
		strings.Join(readOnlyToolNames, ", ") + ". " +
		"Focus on gathering information and providing a clear, complete answer. " +
		"Do NOT attempt to modify any files."

	toolDefs := getReadOnlyTools()
	messages := []SubMessage{{Role: "user", Content: userContent}}

	// Agentic loop
	var finalContent strings.Builder
	for turn := 0; turn < maxTurns; turn++ {
		ch := sp.SubStream(systemPrompt, messages, toolDefs)

		var turnContent strings.Builder
		var toolCalls []ToolCall
		hasToolUse := false

		for event := range ch {
			switch event.Type {
			case "content":
				turnContent.WriteString(event.Content)
			case "tool_use":
				if event.ToolCall != nil {
					toolCalls = append(toolCalls, *event.ToolCall)
					hasToolUse = true
				}
			case "error":
				if event.Error != nil {
					return ToolResult{
						Content: fmt.Sprintf("Sub-agent error: %v", event.Error),
						IsError: true,
					}
				}
			}
		}

		content := turnContent.String()

		if !hasToolUse {
			// No tool calls - the sub-agent is done
			finalContent.WriteString(content)
			break
		}

		// Add assistant message to history
		messages = append(messages, SubMessage{Role: "assistant", Content: content})

		// Execute tool calls and build results
		var toolResultsContent strings.Builder
		for _, tc := range toolCalls {
			result := executeReadOnly(tc.Name, tc.Arguments)
			toolResultsContent.WriteString(fmt.Sprintf("[Tool: %s]\n%s\n\n", tc.Name, result.Content))
		}

		// Add tool results as user message
		messages = append(messages, SubMessage{Role: "user", Content: toolResultsContent.String()})

		// If this was the last turn, note it
		if turn == maxTurns-1 {
			finalContent.WriteString(content)
			finalContent.WriteString("\n\n(Sub-agent reached max turns)")
		}
	}

	result := finalContent.String()
	if result == "" {
		result = "(Sub-agent returned no content)"
	}

	return ToolResult{Content: result}
}
