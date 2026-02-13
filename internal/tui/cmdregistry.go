package tui

import (
	"fmt"
	"sort"
	"strings"
)

// Command represents a slash command in the TUI.
type Command struct {
	Name        string   // primary name without slash (e.g., "clear")
	Aliases     []string // short aliases (e.g., ["c"])
	Category    string   // grouping for help display (e.g., "Chat", "Session")
	Description string   // short description for help/completion
	Usage       string   // e.g., "/clear" or "/theme <name>"
	Handler     func(m *Model, args []string)
}

// CommandRegistry manages all slash commands.
type CommandRegistry struct {
	commands map[string]*Command // name/alias -> command (includes aliases)
	ordered  []*Command          // registration order for help display
}

// NewCommandRegistry creates an empty registry.
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		commands: make(map[string]*Command),
	}
}

// Register adds a command to the registry, mapping its name and all aliases.
func (r *CommandRegistry) Register(cmd *Command) {
	r.ordered = append(r.ordered, cmd)
	r.commands[cmd.Name] = cmd
	for _, alias := range cmd.Aliases {
		r.commands[alias] = cmd
	}
}

// Get returns the command for the given name (without slash), or nil.
func (r *CommandRegistry) Get(name string) *Command {
	return r.commands[name]
}

// Execute looks up and runs the command. Returns true if found.
func (r *CommandRegistry) Execute(m *Model, name string, args []string) bool {
	cmd := r.commands[name]
	if cmd == nil {
		return false
	}
	cmd.Handler(m, args)
	return true
}

// Names returns all slash command names (with "/" prefix) for tab completion.
// Includes primary names and aliases, sorted alphabetically.
func (r *CommandRegistry) Names() []string {
	seen := make(map[string]bool)
	var names []string
	for _, cmd := range r.ordered {
		// Add primary name
		n := "/" + cmd.Name
		if !seen[n] {
			seen[n] = true
			names = append(names, n)
		}
		// Add aliases
		for _, alias := range cmd.Aliases {
			a := "/" + alias
			if !seen[a] {
				seen[a] = true
				names = append(names, a)
			}
		}
	}
	sort.Strings(names)
	return names
}

// HelpString returns a formatted help string listing all commands.
func (r *CommandRegistry) HelpString() string {
	var parts []string
	for _, cmd := range r.ordered {
		name := "/" + cmd.Name
		if len(cmd.Aliases) > 0 {
			aliases := make([]string, len(cmd.Aliases))
			for i, a := range cmd.Aliases {
				aliases[i] = "/" + a
			}
			name += " (" + strings.Join(aliases, ", ") + ")"
		}
		parts = append(parts, name)
	}
	return strings.Join(parts, " ")
}

// HelpDetailed returns a multi-line help with descriptions.
func (r *CommandRegistry) HelpDetailed() string {
	var b strings.Builder
	b.WriteString("Available commands:\n\n")
	for _, cmd := range r.ordered {
		usage := cmd.Usage
		if usage == "" {
			usage = "/" + cmd.Name
		}
		b.WriteString(fmt.Sprintf("  %-30s %s\n", usage, cmd.Description))
	}
	return b.String()
}

// ByCategory returns commands grouped by category, preserving registration order.
// Returns category names in order of first appearance and a map of category -> commands.
func (r *CommandRegistry) ByCategory() ([]string, map[string][]*Command) {
	groups := make(map[string][]*Command)
	var order []string
	seen := make(map[string]bool)
	for _, cmd := range r.ordered {
		cat := cmd.Category
		if cat == "" {
			cat = "Other"
		}
		if !seen[cat] {
			seen[cat] = true
			order = append(order, cat)
		}
		groups[cat] = append(groups[cat], cmd)
	}
	return order, groups
}
