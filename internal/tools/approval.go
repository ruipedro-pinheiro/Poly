package tools

import (
	"fmt"
	"sync"

	"github.com/pedromelo/poly/internal/permission"
)

// PendingApproval represents a tool call waiting for user approval
type PendingApproval struct {
	Name    string
	Args    map[string]interface{}
	Summary string // Human-readable summary for the dialog
}

var (
	// YoloMode skips all approval dialogs
	YoloMode bool

	// PendingChan sends tool calls that need approval to the TUI
	PendingChan = make(chan PendingApproval, 1)

	// ApprovedChan receives approval decisions from the TUI (true = approved)
	ApprovedChan = make(chan bool, 1)

	// toolAllowList tracks tools that have been auto-approved (via 'a' key)
	toolAllowList   = make(map[string]bool)
	toolAllowListMu sync.RWMutex

	// ModifiedFiles tracks files modified by tools during the session
	ModifiedFiles []string
	modFilesMu    sync.Mutex
)

// NeedsApproval returns true if a tool requires user approval
func NeedsApproval(name string, args map[string]interface{}) bool {
	if YoloMode {
		return false
	}
	if IsToolAllowed(name) {
		return false
	}
	// For bash, use command-level classification
	if name == "bash" {
		if cmd, ok := args["command"].(string); ok {
			level := permission.ClassifyBashCommand(cmd)
			return level == permission.Ask
		}
	}
	return permission.ClassifyTool(name) == permission.Ask
}

// IsBashBanned returns true if a bash command is on the banned list
func IsBashBanned(command string) bool {
	return permission.ClassifyBashCommand(command) == permission.Deny
}

// IsToolAllowed checks if a tool has been auto-approved by the user
func IsToolAllowed(name string) bool {
	toolAllowListMu.RLock()
	defer toolAllowListMu.RUnlock()
	return toolAllowList[name]
}

// AllowTool adds a tool to the auto-approve list
func AllowTool(name string) {
	toolAllowListMu.Lock()
	defer toolAllowListMu.Unlock()
	toolAllowList[name] = true
}

// ResetAllowList clears the auto-approve list
func ResetAllowList() {
	toolAllowListMu.Lock()
	defer toolAllowListMu.Unlock()
	toolAllowList = make(map[string]bool)
}

// TrackModifiedFile adds a file to the modified files list (deduped)
func TrackModifiedFile(path string) {
	modFilesMu.Lock()
	defer modFilesMu.Unlock()
	for _, f := range ModifiedFiles {
		if f == path {
			return
		}
	}
	ModifiedFiles = append(ModifiedFiles, path)
}

// ClearModifiedFiles resets the modified files list
func ClearModifiedFiles() {
	modFilesMu.Lock()
	defer modFilesMu.Unlock()
	ModifiedFiles = nil
}

// GetModifiedFiles returns a copy of the modified files list
func GetModifiedFiles() []string {
	modFilesMu.Lock()
	defer modFilesMu.Unlock()
	result := make([]string, len(ModifiedFiles))
	copy(result, ModifiedFiles)
	return result
}

// summarizeToolCall creates a human-readable summary of a tool call
func summarizeToolCall(name string, args map[string]interface{}) string {
	switch name {
	case "bash":
		cmd, _ := args["command"].(string)
		if len(cmd) > 100 {
			cmd = cmd[:100] + "..."
		}
		return "$ " + cmd
	case "write_file":
		path, _ := args["path"].(string)
		return "Write to " + path
	case "edit_file":
		path, _ := args["path"].(string)
		return "Edit " + path
	case "multiedit":
		edits, _ := args["edits"].([]interface{})
		return fmt.Sprintf("Edit %d files", len(edits))
	default:
		return name
	}
}
