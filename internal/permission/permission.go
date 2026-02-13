package permission

import "strings"

// Level represents a permission level
type Level int

const (
	Allow Level = iota // Auto-approve (read-only tools)
	Ask                // Ask user before executing
	Deny               // Block execution
)

// Policy holds the permission configuration
type Policy struct {
	YoloMode bool // If true, auto-approve everything
}

// DefaultPolicy returns the default permission policy
func DefaultPolicy() *Policy {
	return &Policy{YoloMode: false}
}

// toolClassification maps tool names to their permission level
var toolClassification = map[string]Level{
	// Read-only tools - always allow
	"read_file":  Allow,
	"list_files": Allow,
	"glob":       Allow,
	"grep":       Allow,
	"todos":      Allow,

	// LSP tools - read-only
	"lsp_diagnostics": Allow,
	"lsp_hover":       Allow,
	"lsp_definition":  Allow,

	// Git tools - read-only
	"git_status": Allow,
	"git_diff":   Allow,
	"git_log":    Allow,

	// System info - read-only
	"system_info": Allow,

	// Delegation - ask permission (spawns API call)
	"delegate_task": Ask,

	// Side-effect tools - ask permission
	"bash":       Ask,
	"write_file": Ask,
	"edit_file":  Ask,
	"multiedit":  Ask,
	"web_fetch":  Ask,
	"web_search":   Ask,
	"memory_write": Ask,
}

// ClassifyTool returns the permission level for a tool
func ClassifyTool(name string) Level {
	if level, ok := toolClassification[name]; ok {
		return level
	}
	return Ask // Unknown tools require permission
}

// ShouldAsk returns true if the tool requires user approval
func (p *Policy) ShouldAsk(toolName string) bool {
	if p.YoloMode {
		return false
	}
	return ClassifyTool(toolName) == Ask
}

// IsReadOnly returns true if the tool is read-only
func IsReadOnly(name string) bool {
	return ClassifyTool(name) == Allow
}

// safeCommands are bash commands that are auto-approved (read-only)
var safeCommands = []string{
	"ls", "cat", "head", "tail", "wc", "file", "which", "whereis", "whoami",
	"pwd", "echo", "date", "uname", "hostname",
	"git status", "git log", "git diff", "git branch", "git show",
	"go version", "go env", "node --version", "python --version",
	"cargo --version", "rustc --version",
}

// bannedCommands are bash commands that are always denied
var bannedCommands = []string{
	"rm -rf /", "rm -rf ~", "rm -rf *",
	"sudo rm", "sudo shutdown", "sudo reboot", "sudo halt",
	"mkfs", "dd if=", ":(){:|:&};:",
	"chmod -R 777 /", "chown -R",
	"> /dev/sda", "mv / ",
}

// ClassifyBashCommand returns the permission level for a bash command
func ClassifyBashCommand(command string) Level {
	cmd := strings.TrimSpace(strings.ToLower(command))
	for _, banned := range bannedCommands {
		if strings.Contains(cmd, strings.ToLower(banned)) {
			return Deny
		}
	}
	for _, safe := range safeCommands {
		if strings.HasPrefix(cmd, strings.ToLower(safe)) {
			return Allow
		}
	}
	return Ask
}
