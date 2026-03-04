package tools

import (
	"fmt"
	"os"
	"path/filepath"
)

// MemoryWriteTool writes or appends to persistent memory (~/.poly/MEMORY.md)
type MemoryWriteTool struct{}

func (t *MemoryWriteTool) Name() string {
	return "memory_write"
}

func (t *MemoryWriteTool) Description() string {
	return "Write or append to persistent memory (~/.poly/MEMORY.md). This file is injected into the system prompt across sessions."
}

func (t *MemoryWriteTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"content": map[string]interface{}{
				"type":        "string",
				"description": "The text to write or append to MEMORY.md",
			},
			"mode": map[string]interface{}{
				"type":        "string",
				"description": "Write mode: 'append' (default) adds to end, 'replace' overwrites entire file",
				"enum":        []string{"append", "replace"},
			},
		},
		"required": []string{"content"},
	}
}

func (t *MemoryWriteTool) Execute(args map[string]interface{}) ToolResult {
	content, ok := args["content"].(string)
	if !ok || content == "" {
		return ToolResult{Content: "Error: content is required", IsError: true}
	}

	mode, _ := args["mode"].(string)
	if mode == "" {
		mode = "append"
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("Error: cannot get home dir: %v", err), IsError: true}
	}

	dir := filepath.Join(home, ".poly")
	path := filepath.Join(dir, "MEMORY.md")

	// Ensure ~/.poly/ exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return ToolResult{Content: fmt.Sprintf("Error creating ~/.poly/: %v", err), IsError: true}
	}

	if mode == "replace" {
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			return ToolResult{Content: fmt.Sprintf("Error writing MEMORY.md: %v", err), IsError: true}
		}
		return ToolResult{Content: fmt.Sprintf("Replaced MEMORY.md (%d bytes)", len(content))}
	}

	// Append mode
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("Error opening MEMORY.md: %v", err), IsError: true}
	}
	defer f.Close()

	// Add newline before appending if file isn't empty
	info, _ := f.Stat()
	if info != nil && info.Size() > 0 {
		content = "\n" + content
	}

	if _, err := f.WriteString(content); err != nil {
		return ToolResult{Content: fmt.Sprintf("Error appending to MEMORY.md: %v", err), IsError: true}
	}

	return ToolResult{Content: fmt.Sprintf("Appended to MEMORY.md (%d bytes added)", len(content))}
}
