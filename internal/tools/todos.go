package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// Todo matches Crush format
type Todo struct {
	Content    string `json:"content"`
	Status     string `json:"status"`
	ActiveForm string `json:"active_form"`
}

type TodosTool struct{}

func (t *TodosTool) Name() string {
	return "todos"
}

func (t *TodosTool) Description() string {
	return "Updates the todo list in ~/.poly/todos.json. Full replacement array. Returns updated list. Crush format: [{content, status (pending/in_progress/completed), active_form}]. UI updates in real-time."
}

func (t *TodosTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"todos": map[string]interface{}{
				"type": "array",
				"description": "New todo list (full replacement)",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"content":    map[string]interface{}{"type": "string", "description": "Task description (imperative)"},
						"status":     map[string]interface{}{"type": "string", "enum": []string{"pending", "in_progress", "completed"}},
						"active_form": map[string]interface{}{"type": "string", "description": "Present continuous form for UI"},
					},
					"required": []string{"content", "status", "active_form"},
				},
			},
		},
		"required": []string{"todos"},
	}
}

func (t *TodosTool) Execute(args map[string]interface{}) ToolResult {
	u, err := user.Current()
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("Error getting home dir: %v", err), IsError: true}
	}
	todosPath := filepath.Join(u.HomeDir, ".poly", "todos.json")

	// Ensure dir
	dir := filepath.Dir(todosPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return ToolResult{Content: fmt.Sprintf("Error creating dir: %v", err), IsError: true}
	}

	// Load existing
	data, err := os.ReadFile(todosPath)
	if err != nil {
		data = []byte("[]")
	}

	var current []Todo
	if err := json.Unmarshal(data, &current); err != nil {
		return ToolResult{Content: fmt.Sprintf("Error reading todos: %v", err), IsError: true}
	}

	// Get new todos from args
	newTodosI, ok := args["todos"].([]interface{})
	if !ok {
		return ToolResult{Content: "Error: todos must be array", IsError: true}
	}

	newTodos := make([]Todo, len(newTodosI))
	for i, ti := range newTodosI {
		to, ok := ti.(map[string]interface{})
		if !ok {
			return ToolResult{Content: fmt.Sprintf("Invalid todo at index %d", i), IsError: true}
		}
		content, _ := to["content"].(string)
		status, _ := to["status"].(string)
		activeForm, _ := to["active_form"].(string)
		newTodos[i] = Todo{Content: content, Status: status, ActiveForm: activeForm}
	}

	// Write
	jsonData, err := json.MarshalIndent(newTodos, "", "  ")
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("Error marshaling: %v", err), IsError: true}
	}
	if err := os.WriteFile(todosPath, jsonData, 0644); err != nil {
		return ToolResult{Content: fmt.Sprintf("Error writing: %v", err), IsError: true}
	}

	// Pretty output
	var pretty strings.Builder
	pretty.WriteString("Updated todos:\n")
	for _, todo := range newTodos {
		statusEmoji := "○"
		if todo.Status == "completed" {
			statusEmoji = "✓"
		} else if todo.Status == "in_progress" {
			statusEmoji = "▶"
		}
		pretty.WriteString(fmt.Sprintf("  %s %s (%s)\n", statusEmoji, todo.ActiveForm, todo.Status))
	}
	pretty.WriteString(fmt.Sprintf("\nSaved to %s", todosPath))
	return ToolResult{Content: pretty.String()}
}