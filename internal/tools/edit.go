package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EditFileTool performs search and replace in files
type EditFileTool struct{}

func (t *EditFileTool) Name() string {
	return "edit_file"
}

func (t *EditFileTool) Description() string {
	return "Edit a file by replacing old_string with new_string. The old_string must be unique in the file. Use replace_all=true to replace all occurrences."
}

func (t *EditFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "File path to edit (relative to cwd)",
			},
			"old_string": map[string]interface{}{
				"type":        "string",
				"description": "The exact string to find and replace",
			},
			"new_string": map[string]interface{}{
				"type":        "string",
				"description": "The string to replace with",
			},
			"replace_all": map[string]interface{}{
				"type":        "boolean",
				"description": "Replace all occurrences (default: false, requires unique match)",
			},
		},
		"required": []string{"path", "old_string", "new_string"},
	}
}

func (t *EditFileTool) Execute(args map[string]interface{}) ToolResult {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return ToolResult{Content: "Error: path is required", IsError: true}
	}

	oldString, ok := args["old_string"].(string)
	if !ok {
		return ToolResult{Content: "Error: old_string is required", IsError: true}
	}

	newString, _ := args["new_string"].(string)

	replaceAll := false
	if ra, ok := args["replace_all"].(bool); ok {
		replaceAll = ra
	}

	cwd, _ := os.Getwd()
	targetPath := filepath.Join(cwd, path)
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return ToolResult{Content: "Error: invalid path", IsError: true}
	}

	// Security check
	if !strings.HasPrefix(absPath, cwd) {
		return ToolResult{Content: "Error: access denied. Cannot edit outside project root.", IsError: true}
	}

	// Read file
	data, err := os.ReadFile(absPath)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("Error reading file: %v", err), IsError: true}
	}

	content := string(data)

	// Backup before modification
	BackupFile(absPath)

	// Count occurrences
	count := strings.Count(content, oldString)

	if count == 0 {
		// Show context to help debug
		preview := content
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		return ToolResult{
			Content: fmt.Sprintf("Error: old_string not found in file.\n\nFile preview:\n%s", preview),
			IsError: true,
		}
	}

	if count > 1 && !replaceAll {
		return ToolResult{
			Content: fmt.Sprintf("Error: old_string found %d times. Use replace_all=true or make the string more specific.", count),
			IsError: true,
		}
	}

	// Perform replacement
	var newContent string
	if replaceAll {
		newContent = strings.ReplaceAll(content, oldString, newString)
	} else {
		newContent = strings.Replace(content, oldString, newString, 1)
	}

	// Write back
	if err := os.WriteFile(absPath, []byte(newContent), 0644); err != nil {
		return ToolResult{Content: fmt.Sprintf("Error writing file: %v", err), IsError: true}
	}

	// Calculate diff stats
	oldLines := strings.Count(oldString, "\n")
	newLines := strings.Count(newString, "\n")
	lineDiff := newLines - oldLines

	var diffMsg string
	if lineDiff > 0 {
		diffMsg = fmt.Sprintf("+%d lines", lineDiff)
	} else if lineDiff < 0 {
		diffMsg = fmt.Sprintf("%d lines", lineDiff)
	} else {
		diffMsg = "same line count"
	}

	// Track modified file
	TrackModifiedFile(path)

	// Generate diff
	diff := GenerateUnifiedDiff(content, newContent, path)

	if replaceAll && count > 1 {
		return ToolResult{Content: fmt.Sprintf("Replaced %d occurrences in %s (%s)\n\n%s", count, path, diffMsg, diff)}
	}
	return ToolResult{Content: fmt.Sprintf("Edited %s (%s)\n\n%s", path, diffMsg, diff)}
}

