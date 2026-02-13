package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WriteFileTool writes content to a file
type WriteFileTool struct{}

func (t *WriteFileTool) Name() string {
	return "write_file"
}

func (t *WriteFileTool) Description() string {
	return "Write content to a file. Creates the file if it doesn't exist, or overwrites if it does. Creates parent directories if needed."
}

func (t *WriteFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "File path to write to (relative to cwd)",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "Content to write to the file",
			},
		},
		"required": []string{"path", "content"},
	}
}

func (t *WriteFileTool) Execute(args map[string]interface{}) ToolResult {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return ToolResult{Content: "Error: path is required", IsError: true}
	}

	content, ok := args["content"].(string)
	if !ok {
		return ToolResult{Content: "Error: content is required", IsError: true}
	}

	cwd, _ := os.Getwd()
	targetPath := filepath.Join(cwd, path)
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return ToolResult{Content: "Error: invalid path", IsError: true}
	}

	// Security check
	if !strings.HasPrefix(absPath, cwd) {
		return ToolResult{Content: "Error: access denied. Cannot write outside project root.", IsError: true}
	}

	// Create parent directories if needed
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return ToolResult{Content: fmt.Sprintf("Error creating directory: %v", err), IsError: true}
	}

	// Read old content if file exists (for diff and reporting)
	existed := false
	var oldContent string
	if data, err := os.ReadFile(absPath); err == nil {
		existed = true
		oldContent = string(data)
	}

	// Backup before modification (only if file existed)
	if existed {
		BackupFile(absPath)
	}

	// Write file
	if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
		return ToolResult{Content: fmt.Sprintf("Error writing file: %v", err), IsError: true}
	}

	// Track modified file
	TrackModifiedFile(path)

	lines := strings.Count(content, "\n") + 1
	action := "Created"
	if existed {
		action = "Updated"
	}

	// Generate diff for updates
	if existed {
		diff := GenerateUnifiedDiff(oldContent, content, path)
		return ToolResult{Content: fmt.Sprintf("%s %s (%d lines, %d bytes)\n\n%s", action, path, lines, len(content), diff)}
	}

	return ToolResult{Content: fmt.Sprintf("%s %s (%d lines, %d bytes)", action, path, lines, len(content))}
}

