package tools

import (
	"fmt"
	"os"
	"strings"
)

// MultieditTool performs multiple file edits in a single operation
type MultieditTool struct{}

func (t *MultieditTool) Name() string { return "multiedit" }

func (t *MultieditTool) Description() string {
	return "Edit multiple files in a single atomic operation. Each edit replaces old_string with new_string in the specified file. All edits are validated before any are applied."
}

func (t *MultieditTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"edits": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
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
					},
					"required": []string{"path", "old_string", "new_string"},
				},
				"description": "List of edits to apply atomically",
			},
		},
		"required": []string{"edits"},
	}
}

func (t *MultieditTool) Execute(args map[string]interface{}) ToolResult {
	editsRaw, ok := args["edits"].([]interface{})
	if !ok || len(editsRaw) == 0 {
		return ToolResult{Content: "Error: edits array is required and must not be empty", IsError: true}
	}

	// Phase 1: Validate all edits before applying any
	type editOp struct {
		absPath   string
		relPath   string
		oldString string
		newString string
		content   string // current file content
	}

	ops := make([]editOp, 0, len(editsRaw))
	for i, raw := range editsRaw {
		edit, ok := raw.(map[string]interface{})
		if !ok {
			return ToolResult{Content: fmt.Sprintf("Error: edit[%d] is not an object", i), IsError: true}
		}

		path, _ := edit["path"].(string)
		oldStr, _ := edit["old_string"].(string)
		newStr, _ := edit["new_string"].(string)

		if path == "" || oldStr == "" {
			return ToolResult{Content: fmt.Sprintf("Error: edit[%d] missing path or old_string", i), IsError: true}
		}

		// Path validation (resolves symlinks, blocks traversal)
		absPath, err := ValidatePath(path)
		if err != nil {
			return ToolResult{Content: fmt.Sprintf("Error: edit[%d] %v", i, err), IsError: true}
		}

		data, err := os.ReadFile(absPath)
		if err != nil {
			return ToolResult{Content: fmt.Sprintf("Error: edit[%d] cannot read %s: %v", i, path, err), IsError: true}
		}

		content := string(data)
		if !strings.Contains(content, oldStr) {
			return ToolResult{Content: fmt.Sprintf("Error: edit[%d] old_string not found in %s", i, path), IsError: true}
		}

		count := strings.Count(content, oldStr)
		if count > 1 {
			return ToolResult{Content: fmt.Sprintf("Error: edit[%d] old_string found %d times in %s (must be unique)", i, count, path), IsError: true}
		}

		ops = append(ops, editOp{
			absPath:   absPath,
			relPath:   path,
			oldString: oldStr,
			newString: newStr,
			content:   content,
		})
	}

	// Phase 2: Backup all files before applying edits
	for _, op := range ops {
		BackupFile(op.absPath)
	}

	// Phase 3: Apply all edits (validation passed)
	var results strings.Builder
	successCount := 0
	for _, op := range ops {
		newContent := strings.Replace(op.content, op.oldString, op.newString, 1)

		// Generate diff
		diff := GenerateUnifiedDiff(op.content, newContent, op.relPath)

		if err := os.WriteFile(op.absPath, []byte(newContent), 0600); err != nil {
			results.WriteString(fmt.Sprintf("\n\n--- %s ---\nERROR: %v", op.relPath, err))
		} else {
			successCount++
			TrackModifiedFile(op.relPath)
			if diff != "" {
				results.WriteString(fmt.Sprintf("\n\n%s", diff))
			} else {
				results.WriteString(fmt.Sprintf("\n--- %s: edited", op.relPath))
			}
		}
	}

	return ToolResult{Content: fmt.Sprintf("Multiedit: %d/%d files modified%s", successCount, len(ops), results.String())}
}
