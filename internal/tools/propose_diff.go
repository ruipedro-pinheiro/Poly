package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DiffProposal represents a proposed change
type DiffProposal struct {
	ID          string
	FilePath    string
	OldContent  string
	NewContent  string
	Diff        string
	Description string
	CreatedAt   time.Time
}

var (
	proposals   = make(map[string]*DiffProposal)
	proposalsMu sync.RWMutex
	nextID      int
)

// ProposeDiffTool proposes file changes without applying them
type ProposeDiffTool struct{}

func (t *ProposeDiffTool) Name() string {
	return "propose_diff"
}

func (t *ProposeDiffTool) Description() string {
	return "Propose file changes as a diff for user review before applying. Returns a diff ID that can be used with apply_diff or reject_diff."
}

func (t *ProposeDiffTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "File path to modify (relative to cwd)",
			},
			"new_content": map[string]interface{}{
				"type":        "string",
				"description": "Proposed new content for the file",
			},
			"description": map[string]interface{}{
				"type":        "string",
				"description": "Brief description of what this change does",
			},
		},
		"required": []string{"path", "new_content"},
	}
}

func (t *ProposeDiffTool) Execute(args map[string]interface{}) ToolResult {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return ToolResult{Content: "Error: path is required", IsError: true}
	}

	newContent, ok := args["new_content"].(string)
	if !ok {
		return ToolResult{Content: "Error: new_content is required", IsError: true}
	}

	description, _ := args["description"].(string)
	if description == "" {
		description = "No description provided"
	}

	// Path validation (resolves symlinks, blocks traversal)
	absPath, err := ValidatePath(path)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("Error: %v", err), IsError: true}
	}

	// Read current file content (or empty if new file)
	var oldContent string
	data, err := os.ReadFile(absPath)
	if err == nil {
		oldContent = string(data)
	} else if !os.IsNotExist(err) {
		return ToolResult{Content: fmt.Sprintf("Error reading file: %v", err), IsError: true}
	}

	// Generate diff
	var diff string
	if oldContent == "" {
		diff = fmt.Sprintf("New file: %s\n\n%s", path, newContent)
	} else {
		diff = GenerateUnifiedDiff(oldContent, newContent, path)
	}

	// Create proposal
	proposalsMu.Lock()
	nextID++
	id := fmt.Sprintf("diff-%d", nextID)
	proposals[id] = &DiffProposal{
		ID:          id,
		FilePath:    path,
		OldContent:  oldContent,
		NewContent:  newContent,
		Diff:        diff,
		Description: description,
		CreatedAt:   time.Now(),
	}
	proposalsMu.Unlock()

	output := fmt.Sprintf("📝 Diff proposal created: %s\n\n", id)
	output += fmt.Sprintf("Description: %s\n", description)
	output += fmt.Sprintf("File: %s\n\n", path)
	output += diff
	output += fmt.Sprintf("\n\nTo apply: use apply_diff with id '%s'", id)
	output += fmt.Sprintf("\nTo reject: use reject_diff with id '%s'", id)
	output += fmt.Sprintf("\nTo list all: use list_diffs")

	return ToolResult{Content: output}
}

// ApplyDiffTool applies a previously proposed diff
type ApplyDiffTool struct{}

func (t *ApplyDiffTool) Name() string {
	return "apply_diff"
}

func (t *ApplyDiffTool) Description() string {
	return "Apply a previously proposed diff by its ID. This writes the changes to disk."
}

func (t *ApplyDiffTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type":        "string",
				"description": "The diff proposal ID to apply (e.g. 'diff-1')",
			},
		},
		"required": []string{"id"},
	}
}

func (t *ApplyDiffTool) Execute(args map[string]interface{}) ToolResult {
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return ToolResult{Content: "Error: id is required", IsError: true}
	}

	proposalsMu.Lock()
	proposal, exists := proposals[id]
	if !exists {
		proposalsMu.Unlock()
		return ToolResult{Content: fmt.Sprintf("Error: diff proposal '%s' not found. Use list_diffs to see available proposals.", id), IsError: true}
	}
	delete(proposals, id) // Remove after applying
	proposalsMu.Unlock()

	cwd, _ := os.Getwd()
	targetPath := filepath.Join(cwd, proposal.FilePath)
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return ToolResult{Content: "Error: invalid path", IsError: true}
	}

	// Create parent directories if needed
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return ToolResult{Content: fmt.Sprintf("Error creating directories: %v", err), IsError: true}
	}

	// Write the new content
	if err := os.WriteFile(absPath, []byte(proposal.NewContent), 0644); err != nil {
		return ToolResult{Content: fmt.Sprintf("Error writing file: %v", err), IsError: true}
	}

	// Track modified file
	TrackModifiedFile(proposal.FilePath)

	output := fmt.Sprintf("✅ Applied diff %s\n", id)
	output += fmt.Sprintf("Description: %s\n", proposal.Description)
	output += fmt.Sprintf("File: %s\n\n", proposal.FilePath)
	output += proposal.Diff

	return ToolResult{Content: output}
}

// RejectDiffTool rejects a proposed diff
type RejectDiffTool struct{}

func (t *RejectDiffTool) Name() string {
	return "reject_diff"
}

func (t *RejectDiffTool) Description() string {
	return "Reject a previously proposed diff by its ID. This discards the proposal without applying it."
}

func (t *RejectDiffTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type":        "string",
				"description": "The diff proposal ID to reject (e.g. 'diff-1')",
			},
		},
		"required": []string{"id"},
	}
}

func (t *RejectDiffTool) Execute(args map[string]interface{}) ToolResult {
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return ToolResult{Content: "Error: id is required", IsError: true}
	}

	proposalsMu.Lock()
	proposal, exists := proposals[id]
	if !exists {
		proposalsMu.Unlock()
		return ToolResult{Content: fmt.Sprintf("Error: diff proposal '%s' not found", id), IsError: true}
	}
	delete(proposals, id)
	proposalsMu.Unlock()

	return ToolResult{Content: fmt.Sprintf("❌ Rejected diff %s: %s", id, proposal.Description)}
}

// ListDiffsTool lists all pending diff proposals
type ListDiffsTool struct{}

func (t *ListDiffsTool) Name() string {
	return "list_diffs"
}

func (t *ListDiffsTool) Description() string {
	return "List all pending diff proposals that haven't been applied or rejected yet."
}

func (t *ListDiffsTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{},
	}
}

func (t *ListDiffsTool) Execute(args map[string]interface{}) ToolResult {
	proposalsMu.RLock()
	defer proposalsMu.RUnlock()

	if len(proposals) == 0 {
		return ToolResult{Content: "No pending diff proposals"}
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("📋 %d pending diff proposal(s):\n\n", len(proposals)))

	for _, proposal := range proposals {
		age := time.Since(proposal.CreatedAt)
		output.WriteString(fmt.Sprintf("• %s - %s\n", proposal.ID, proposal.Description))
		output.WriteString(fmt.Sprintf("  File: %s\n", proposal.FilePath))
		output.WriteString(fmt.Sprintf("  Created: %s ago\n\n", formatDuration(age)))
	}

	output.WriteString("Use apply_diff or reject_diff with the ID to proceed.")

	return ToolResult{Content: output.String()}
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}
