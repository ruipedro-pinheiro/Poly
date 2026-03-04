package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pedromelo/poly/internal/config"
	"github.com/pedromelo/poly/internal/llm"
	"github.com/pedromelo/poly/internal/mcp"
	"github.com/pedromelo/poly/internal/sandbox"
	"github.com/pedromelo/poly/internal/session"
	"github.com/pedromelo/poly/internal/skills"
	"github.com/pedromelo/poly/internal/theme"
	"github.com/pedromelo/poly/internal/tools"
	"github.com/pedromelo/poly/internal/tui/styles"
	"github.com/pedromelo/poly/internal/updater"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// initCommands builds the command registry with all slash commands.
func initCommands() *CommandRegistry {
	r := NewCommandRegistry()

	r.Register(&Command{
		Name:        "clear",
		Aliases:     []string{"c"},
		Category:    "Chat",
		Description: "Clear conversation",
		Usage:       "/clear",
		Handler: func(m *Model, args []string) {
			m.messages = []Message{}
			_ = session.Clear()
			m.updateViewport()
			m.status = "Chat cleared"
		},
	})

	r.Register(&Command{
		Name:        "model",
		Aliases:     []string{"m"},
		Category:    "Config",
		Description: "Show or set model variant",
		Usage:       "/model [fast|think|opus|default]",
		Handler: func(m *Model, args []string) {
			if len(args) == 0 {
				variant := m.modelVariant
				if variant == "" {
					variant = "default"
				}
				m.status = "Model: " + variant + " (use /model fast|think|opus)"
			} else {
				switch args[0] {
				case "fast", "think", "opus", "default":
					m.modelVariant = args[0]
					m.status = "Model set to: " + args[0]
				default:
					m.status = "Unknown variant. Use: fast, think, opus, default"
				}
			}
		},
	})

	r.Register(&Command{
		Name:        "think",
		Aliases:     []string{"t"},
		Category:    "Chat",
		Description: "Toggle thinking mode",
		Usage:       "/think",
		Handler: func(m *Model, args []string) {
			m.thinkingMode = !m.thinkingMode
			if m.thinkingMode {
				m.modelVariant = "think"
				m.status = "Thinking mode ON"
			} else {
				m.modelVariant = "default"
				m.status = "Thinking mode OFF"
			}
		},
	})

	r.Register(&Command{
		Name:        "provider",
		Aliases:     []string{"p"},
		Category:    "Config",
		Description: "Show or set default provider",
		Usage:       "/provider [name]",
		Handler: func(m *Model, args []string) {
			if len(args) == 0 {
				m.status = "Provider: @" + m.defaultProvider
			} else {
				if _, ok := llm.GetProvider(args[0]); ok {
					m.defaultProvider = args[0]
					m.status = "Default provider: @" + args[0]
				} else {
					names := llm.GetProviderNames()
					m.status = "Unknown provider. Use: " + strings.Join(names, ", ")
				}
			}
		},
	})

	r.Register(&Command{
		Name:        "help",
		Aliases:     []string{"h"},
		Category:    "General",
		Description: "Show help or command details",
		Usage:       "/help [command]",
		Handler: func(m *Model, args []string) {
			if len(args) == 0 {
				m.state = viewHelp
				return
			}
			name := strings.TrimPrefix(strings.ToLower(args[0]), "/")
			cmd := r.Get(name)
			if cmd == nil {
				m.status = "Unknown command: /" + name + ". Try /help"
				return
			}
			aliasStr := ""
			if len(cmd.Aliases) > 0 {
				aliases := make([]string, len(cmd.Aliases))
				for i, a := range cmd.Aliases {
					aliases[i] = "/" + a
				}
				aliasStr = " (aliases: " + strings.Join(aliases, ", ") + ")"
			}
			m.status = cmd.Usage + aliasStr + " - " + cmd.Description
		},
	})

	r.Register(&Command{
		Name:        "providers",
		Aliases:     []string{"list"},
		Category:    "Config",
		Description: "List configured providers",
		Usage:       "/providers",
		Handler: func(m *Model, args []string) {
			names := llm.GetProviderNames()
			m.status = "Providers: " + strings.Join(names, ", ")
		},
	})

	r.Register(&Command{
		Name:        "add",
		Category:    "Context",
		Description: "Add file(s) to persistent context",
		Usage:       "/add <file_or_dir>",
		Handler: func(m *Model, args []string) {
			m.handleAddFileCommand(args)
		},
	})

	r.Register(&Command{
		Name:        "remove",
		Category:    "Context",
		Description: "Remove file from context",
		Usage:       "/remove <file>",
		Handler: func(m *Model, args []string) {
			m.handleRemoveFileCommand(args)
		},
	})

	r.Register(&Command{
		Name:        "context",
		Category:    "Context",
		Description: "List files in persistent context",
		Usage:       "/context",
		Handler: func(m *Model, args []string) {
			m.handleContextCommand()
		},
	})

	r.Register(&Command{
		Name:        "addprovider",
		Category:    "Config",
		Description: "Add a custom provider",
		Usage:       "/addprovider <id> <url> <apikey> <model> [format] [color]",
		Handler: func(m *Model, args []string) {
			if len(args) < 4 {
				m.status = "Usage: /addprovider <id> <url> <apikey> <model> [format] [color]"
				return
			}
			format := "openai"
			color := "#888888"
			if len(args) >= 5 {
				if args[4] == "openai" || args[4] == "anthropic" || args[4] == "google" {
					format = args[4]
					if len(args) >= 6 {
						color = args[5]
					}
				} else {
					color = args[4]
				}
			}
			apiKey := args[2]
			if apiKey == "-" || apiKey == "none" || apiKey == "local" {
				apiKey = ""
			}
			cfg := llm.CustomProviderConfig{
				ID:        args[0],
				Name:      cases.Title(language.English).String(args[0]),
				BaseURL:   args[1],
				APIKey:    apiKey,
				Model:     args[3],
				Color:     color,
				Format:    format,
				MaxTokens: 4096,
			}
			if err := llm.SaveCustomProvider(cfg); err != nil {
				m.status = "Error: " + err.Error()
			} else {
				m.providers = llm.GetAllProviders()
				m.controlRoomProviders = llm.GetProviderNames()
				m.status = "Added @" + args[0] + " (" + format + ")"
			}
		},
	})

	r.Register(&Command{
		Name:        "delprovider",
		Aliases:     []string{"del"},
		Category:    "Config",
		Description: "Delete a custom provider",
		Usage:       "/delprovider <id>",
		Handler: func(m *Model, args []string) {
			if len(args) == 0 {
				m.status = "Usage: /delprovider <id>"
				return
			}
			if err := llm.DeleteCustomProvider(args[0]); err != nil {
				m.status = "Error: " + err.Error()
			} else {
				m.status = "Deleted @" + args[0]
			}
		},
	})

	r.Register(&Command{
		Name:        "compact",
		Category:    "Chat",
		Description: "Compact conversation context",
		Usage:       "/compact",
		Handler: func(m *Model, args []string) {
			if m.isStreaming {
				m.status = "Cannot compact while streaming"
			} else if len(m.messages) <= llm.MinMessagesToKeep {
				m.status = "Not enough messages to compact"
			} else {
				m.status = "Compacting context..."
				m.isCompacting = true
			}
		},
	})

	r.Register(&Command{
		Name:        "theme",
		Category:    "Config",
		Description: "Switch color theme",
		Usage:       "/theme [mocha|macchiato|frappe|latte]",
		Handler: func(m *Model, args []string) {
			if len(args) == 0 {
				next := styles.NextTheme()
				theme.SetTheme(next)
				config.SetColorTheme(string(next))
				m.status = "Theme: " + string(next)
			} else {
				name := styles.ThemeName(strings.ToLower(args[0]))
				if _, ok := styles.Palettes[name]; ok {
					styles.SetTheme(name)
					theme.SetTheme(name)
					config.SetColorTheme(string(name))
					m.status = "Theme: " + string(name)
				} else {
					m.status = "Unknown theme. Use: mocha, macchiato, frappe, latte"
				}
			}
		},
	})

	r.Register(&Command{
		Name:        "rounds",
		Category:    "Config",
		Description: "Show or set max Table Ronde rounds",
		Usage:       "/rounds [N]",
		Handler: func(m *Model, args []string) {
			if len(args) == 0 {
				current := llm.GetMaxTableRounds()
				m.status = fmt.Sprintf("Max Table Ronde rounds: %d", current)
				return
			}
			n, err := strconv.Atoi(args[0])
			if err != nil || n < 1 || n > 20 {
				m.status = "Usage: /rounds [1-20]"
				return
			}
			config.SetMaxTableRounds(n)
			m.status = fmt.Sprintf("Max Table Ronde rounds set to %d", n)
		},
	})

	r.Register(&Command{
		Name:        "export",
		Category:    "Session",
		Description: "Export conversation to file",
		Usage:       "/export [json|md]",
		Handler: func(m *Model, args []string) {
			format := "md"
			if len(args) > 0 && args[0] == "json" {
				format = "json"
			}
			var path string
			var exportErr error
			if format == "json" {
				path, exportErr = session.ExportJSON()
			} else {
				path, exportErr = session.ExportMarkdown()
			}
			if exportErr != nil {
				m.status = "Export failed: " + exportErr.Error()
			} else {
				m.status = "Exported to " + path
			}
		},
	})

	r.Register(&Command{
		Name:        "search",
		Aliases:     []string{"s"},
		Category:    "Session",
		Description: "Search across sessions",
		Usage:       "/search <query>",
		Handler: func(m *Model, args []string) {
			if len(args) == 0 {
				m.status = "Usage: /search <query>"
				return
			}
			query := strings.Join(args, " ")
			results, err := session.SearchAll(query, 20)
			if err != nil {
				m.status = "Search error: " + err.Error()
				return
			}
			if len(results) == 0 {
				m.status = fmt.Sprintf("No results for \"%s\" in %d sessions", query, session.CountSessions())
				return
			}
			var b strings.Builder
			b.WriteString(fmt.Sprintf("Search: \"%s\" - %d result(s)\n\n", query, len(results)))
			for i, r := range results {
				b.WriteString(fmt.Sprintf("[%s] (%s): %s\n", r.SessionTitle, r.MessageRole, r.Content))
				if i < len(results)-1 {
					b.WriteString("\n")
				}
			}
			m.messages = append(m.messages, Message{
				Role:    "system",
				Content: b.String(),
			})
			m.updateViewport()
			m.status = fmt.Sprintf("Found %d result(s) for \"%s\"", len(results), query)
		},
	})

	r.Register(&Command{
		Name:        "notify",
		Category:    "Config",
		Description: "Toggle desktop notifications",
		Usage:       "/notify",
		Handler: func(m *Model, args []string) {
			m.notificationsOn = !m.notificationsOn
			config.SetNotifications(m.notificationsOn)
			if m.notificationsOn {
				m.status = "Notifications ON"
			} else {
				m.status = "Notifications OFF"
			}
		},
	})

	r.Register(&Command{
		Name:        "sandbox",
		Category:    "Tools",
		Description: "Toggle sandbox mode",
		Usage:       "/sandbox",
		Handler: func(m *Model, args []string) {
			if !sandbox.Available() {
				m.status = "No container runtime found (install podman or docker)"
				return
			}
			sandbox.Enabled = !sandbox.Enabled
			config.SetSandbox(sandbox.Enabled)
			if sandbox.Enabled {
				m.status = "Sandbox ON (" + sandbox.Detect() + ", image: " + sandbox.Image + ")"
			} else {
				m.status = "Sandbox OFF - commands run locally"
			}
		},
	})

	r.Register(&Command{
		Name:        "yolo",
		Category:    "Tools",
		Description: "Toggle YOLO mode (auto-approve tools)",
		Usage:       "/yolo",
		Handler: func(m *Model, args []string) {
			tools.YoloMode = !tools.YoloMode
			if tools.YoloMode {
				m.status = "YOLO mode ON - auto-approving all tools"
			} else {
				tools.ResetAllowList()
				m.status = "YOLO mode OFF - tools require approval"
			}
		},
	})

	r.Register(&Command{
		Name:        "version",
		Aliases:     []string{"v"},
		Category:    "General",
		Description: "Show version",
		Usage:       "/version",
		Handler: func(m *Model, args []string) {
			m.status = "Poly v" + updater.CurrentVersion
		},
	})

	r.Register(&Command{
		Name:        "undo",
		Category:    "Chat",
		Description: "Undo last exchange",
		Usage:       "/undo",
		Handler: func(m *Model, args []string) {
			m.handleUndo()
		},
	})

	r.Register(&Command{
		Name:        "rewind",
		Aliases:     []string{"rw"},
		Category:    "Chat",
		Description: "Remove last N messages",
		Usage:       "/rewind [N]",
		Handler: func(m *Model, args []string) {
			m.handleRewind(args)
		},
	})

	r.Register(&Command{
		Name:        "retry",
		Category:    "Chat",
		Description: "Retry last message",
		Usage:       "/retry",
		Handler: func(m *Model, args []string) {
			m.handleRetry()
		},
	})

	r.Register(&Command{
		Name:        "compare",
		Category:    "Chat",
		Description: "Compare providers side by side",
		Usage:       "/compare [providers] <prompt>",
		Handler: func(m *Model, args []string) {
			m.handleCompareCommand(args)
		},
	})

	r.Register(&Command{
		Name:        "config",
		Category:    "Config",
		Description: "Show active configuration",
		Usage:       "/config",
		Handler: func(m *Model, args []string) {
			m.handleConfigCommand()
		},
	})

	r.Register(&Command{
		Name:        "revert",
		Category:    "Tools",
		Description: "Revert file changes",
		Usage:       "/revert [list|<file>]",
		Handler: func(m *Model, args []string) {
			m.handleRevertCommand(args)
		},
	})

	r.Register(&Command{
		Name:        "project",
		Category:    "Context",
		Description: "Show project info",
		Usage:       "/project",
		Handler: func(m *Model, args []string) {
			if info := config.DetectProject(); info != nil {
				var b strings.Builder
				b.WriteString(fmt.Sprintf("Project: %s (%s)\n", info.Name, info.Type))
				for _, d := range info.Details {
					b.WriteString(fmt.Sprintf("  %s\n", d))
				}
				m.messages = append(m.messages, Message{
					Role:    "system",
					Content: b.String(),
				})
				m.updateViewport()
				m.status = fmt.Sprintf("Project: %s (%s)", info.Name, info.Type)
			} else {
				m.status = "No project detected in current directory"
			}
		},
	})

	r.Register(&Command{
		Name:        "skill",
		Aliases:     []string{"skills"},
		Category:    "Tools",
		Description: "List or execute skills",
		Usage:       "/skill [name]",
		Handler: func(m *Model, args []string) {
			m.handleSkillCommand(args)
		},
	})

	r.Register(&Command{
		Name:        "stats",
		Category:    "Session",
		Description: "Show session statistics",
		Usage:       "/stats",
		Handler: func(m *Model, args []string) {
			m.handleStatsCommand()
		},
	})

	r.Register(&Command{
		Name:        "costs",
		Category:    "Session",
		Description: "Export session costs to CSV or JSON",
		Usage:       "/costs [csv|json]",
		Handler: func(m *Model, args []string) {
			format := "csv"
			if len(args) > 0 && args[0] == "json" {
				format = "json"
			}
			m.handleCostsExport(format)
		},
	})

	r.Register(&Command{
		Name:        "memory",
		Category:    "Session",
		Description: "Show or clear MEMORY.md",
		Usage:       "/memory [show|clear]",
		Handler: func(m *Model, args []string) {
			m.handleMemoryCommand(args)
		},
	})

	r.Register(&Command{
		Name:        "mcp",
		Category:    "Tools",
		Description: "List MCP servers and tools",
		Usage:       "/mcp",
		Handler: func(m *Model, args []string) {
			m.handleMCPCommand()
		},
	})

	return r
}

// handleCommand processes slash commands via the registry.
func (m *Model) handleCommand(input string) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return
	}

	cmd := strings.ToLower(parts[0])
	args := parts[1:]

	// Strip the leading slash for registry lookup
	name := strings.TrimPrefix(cmd, "/")

	if m.commands.Execute(m, name, args) {
		return
	}

	// Fallback: try matching a skill by name
	if sk := skills.GetSkill(name); sk != nil {
		m.skillContent = sk.Content
		m.status = "Skill: " + sk.Name
	} else {
		m.status = "Unknown command. Try /help"
	}
}

// handleSkillCommand handles /skill [name] - lists or executes skills
func (m *Model) handleSkillCommand(args []string) {
	if len(args) == 0 {
		names := skills.ListSkills()
		if len(names) == 0 {
			m.status = "No skills found. Add .md files to ~/.poly/skills/ or .poly/skills/"
			return
		}
		var b strings.Builder
		b.WriteString(fmt.Sprintf("Available skills (%d):\n", len(names)))
		for _, name := range names {
			sk := skills.GetSkill(name)
			if sk != nil {
				desc := sk.Content
				if idx := strings.Index(desc, "\n"); idx > 0 {
					desc = desc[:idx]
				}
				if len(desc) > 60 {
					desc = desc[:57] + "..."
				}
				b.WriteString(fmt.Sprintf("  /%s - %s\n", name, desc))
			}
		}
		m.messages = append(m.messages, Message{
			Role:    "system",
			Content: b.String(),
		})
		m.updateViewport()
		m.status = fmt.Sprintf("%d skill(s) available", len(names))
		return
	}

	name := strings.ToLower(args[0])
	sk := skills.GetSkill(name)
	if sk == nil {
		m.status = "Unknown skill: " + name
		return
	}

	m.skillContent = sk.Content
	m.status = "Skill: " + sk.Name
}

// parseProvider extracts the target provider from a message
func (m Model) parseProvider(content string) string {
	content = strings.ToLower(content)
	if strings.Contains(content, "@all") {
		return "all"
	}
	// Check all configured providers dynamically
	for _, name := range config.GetProviderNames() {
		if strings.Contains(content, "@"+name) {
			return name
		}
	}
	return m.defaultProvider
}

// maxContextFiles is the maximum number of persistent context files
const maxContextFiles = 10

// maxContextTotalSize is the max total size of all context files (100KB)
const maxContextTotalSize = 100 * 1024

// handleAddFileCommand handles /add <path> - adds file(s) to persistent context
func (m *Model) handleAddFileCommand(args []string) {
	if len(args) == 0 {
		m.status = "Usage: /add <file_or_dir>"
		return
	}

	path := args[0]
	info, err := os.Stat(path)
	if err != nil {
		m.status = "File not found: " + path
		return
	}

	var filesToAdd []string
	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			m.status = "Cannot read dir: " + err.Error()
			return
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") {
				filesToAdd = append(filesToAdd, filepath.Join(path, e.Name()))
			}
		}
		if len(filesToAdd) == 0 {
			m.status = "No .go files in " + path
			return
		}
	} else {
		filesToAdd = []string{path}
	}

	added := 0
	for _, f := range filesToAdd {
		if len(m.contextFiles) >= maxContextFiles {
			m.status = fmt.Sprintf("Max %d context files reached", maxContextFiles)
			return
		}
		already := false
		for _, cf := range m.contextFiles {
			if cf == f {
				already = true
				break
			}
		}
		if already {
			continue
		}
		m.contextFiles = append(m.contextFiles, f)
		added++
	}

	if added == 0 {
		m.status = "Files already in context"
	} else {
		m.status = fmt.Sprintf("Added %d file(s) to context (%d total)", added, len(m.contextFiles))
	}
}

// handleRemoveFileCommand handles /remove <path> - removes file from context
func (m *Model) handleRemoveFileCommand(args []string) {
	if len(args) == 0 {
		m.status = "Usage: /remove <file>"
		return
	}

	path := args[0]
	filtered := make([]string, 0, len(m.contextFiles))
	removed := false
	for _, cf := range m.contextFiles {
		if cf == path {
			removed = true
		} else {
			filtered = append(filtered, cf)
		}
	}

	if !removed {
		m.status = "Not in context: " + path
		return
	}

	m.contextFiles = filtered
	m.status = fmt.Sprintf("Removed from context (%d files left)", len(m.contextFiles))
}

// handleContextCommand handles /context - lists files in persistent context
func (m *Model) handleContextCommand() {
	if len(m.contextFiles) == 0 {
		m.status = "No files in context. Use /add <file>"
		return
	}

	var b strings.Builder
	var totalSize int64
	b.WriteString(fmt.Sprintf("Context files (%d):\n", len(m.contextFiles)))
	for _, f := range m.contextFiles {
		info, err := os.Stat(f)
		if err != nil {
			b.WriteString(fmt.Sprintf("  %s (error: %s)\n", f, err.Error()))
		} else {
			size := info.Size()
			totalSize += size
			b.WriteString(fmt.Sprintf("  %s (%d bytes)\n", f, size))
		}
	}
	b.WriteString(fmt.Sprintf("Total: %d bytes / %d max", totalSize, maxContextTotalSize))

	m.messages = append(m.messages, Message{
		Role:    "system",
		Content: b.String(),
	})
	m.updateViewport()
	m.status = fmt.Sprintf("%d context file(s), %d bytes", len(m.contextFiles), totalSize)
}

// handleUndo removes the last exchange (last user message + all subsequent assistant/system messages)
func (m *Model) handleUndo() {
	if m.isStreaming {
		m.status = "Cannot undo while streaming"
		return
	}
	if len(m.messages) == 0 {
		m.status = "Nothing to undo"
		return
	}

	lastUserIdx := -1
	for i := len(m.messages) - 1; i >= 0; i-- {
		if m.messages[i].Role == "user" {
			lastUserIdx = i
			break
		}
	}

	if lastUserIdx == -1 {
		m.status = "No user message to undo"
		return
	}

	m.messages = m.messages[:lastUserIdx]

	sessionMsgs := make([]session.Message, len(m.messages))
	for i, msg := range m.messages {
		sessionMsgs[i] = session.Message{
			Role:         msg.Role,
			Content:      msg.Content,
			Provider:     msg.Provider,
			Thinking:     msg.Thinking,
			Images:       msg.Images,
			InputTokens:  msg.InputTokens,
			OutputTokens: msg.OutputTokens,
		}
	}
	_ = session.SetMessages(sessionMsgs)

	m.updateViewport()
	m.status = "Last exchange removed"
}

// handleRewind removes the last N messages from the conversation
func (m *Model) handleRewind(args []string) {
	if m.isStreaming {
		m.status = "Cannot rewind while streaming"
		return
	}
	if len(m.messages) == 0 {
		m.status = "Nothing to rewind"
		return
	}

	n := 2 // default: remove last user + assistant pair
	if len(args) > 0 {
		parsed, err := strconv.Atoi(args[0])
		if err != nil || parsed < 1 {
			m.status = "Usage: /rewind [N] (N must be a positive number)"
			return
		}
		n = parsed
	}

	// Cap to available messages
	if n > len(m.messages) {
		n = len(m.messages)
	}

	remaining := len(m.messages) - n
	m.messages = m.messages[:remaining]

	// Re-persist the session
	sessionMsgs := make([]session.Message, len(m.messages))
	for i, msg := range m.messages {
		sessionMsgs[i] = session.Message{
			Role:         msg.Role,
			Content:      msg.Content,
			Provider:     msg.Provider,
			Thinking:     msg.Thinking,
			Images:       msg.Images,
			InputTokens:  msg.InputTokens,
			OutputTokens: msg.OutputTokens,
		}
	}
	_ = session.SetMessages(sessionMsgs)

	m.updateViewport()
	m.status = fmt.Sprintf("Rewound %d messages (%d remaining)", n, remaining)
}

// handleRetry removes the last exchange and re-sends the last user message
func (m *Model) handleRetry() {
	if m.isStreaming {
		m.status = "Cannot retry while streaming"
		return
	}
	if len(m.messages) == 0 {
		m.status = "Nothing to retry"
		return
	}

	lastUserIdx := -1
	for i := len(m.messages) - 1; i >= 0; i-- {
		if m.messages[i].Role == "user" {
			lastUserIdx = i
			break
		}
	}

	if lastUserIdx == -1 {
		m.status = "No user message to retry"
		return
	}

	retryContent := m.messages[lastUserIdx].Content
	m.messages = m.messages[:lastUserIdx]

	sessionMsgs := make([]session.Message, len(m.messages))
	for i, msg := range m.messages {
		sessionMsgs[i] = session.Message{
			Role:         msg.Role,
			Content:      msg.Content,
			Provider:     msg.Provider,
			Thinking:     msg.Thinking,
			Images:       msg.Images,
			InputTokens:  msg.InputTokens,
			OutputTokens: msg.OutputTokens,
		}
	}
	_ = session.SetMessages(sessionMsgs)

	m.retryContent = retryContent
	m.status = "Retrying last message..."
}

// buildContextPrefix reads all contextFiles and builds the <context> XML block
func (m *Model) buildContextPrefix() string {
	if len(m.contextFiles) == 0 {
		return ""
	}

	var b strings.Builder
	var totalSize int
	b.WriteString("<context>\n")

	for _, f := range m.contextFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		totalSize += len(data)
		if totalSize > maxContextTotalSize {
			b.WriteString(fmt.Sprintf("<!-- skipped %s: total context size exceeded %d bytes -->\n", f, maxContextTotalSize))
			continue
		}
		b.WriteString(fmt.Sprintf("<file path=%q>\n%s\n</file>\n", f, string(data)))
	}

	b.WriteString("</context>\n\n")
	return b.String()
}

// handleRevertCommand handles /revert - restores backed up files
func (m *Model) handleRevertCommand(args []string) {
	if len(args) == 0 {
		path, err := tools.RevertLast()
		if err != nil {
			m.status = err.Error()
		} else {
			m.status = "Reverted: " + path
		}
		return
	}

	if args[0] == "list" {
		backups := tools.GetAllBackups()
		if len(backups) == 0 {
			m.status = "No backups available"
			return
		}
		var b strings.Builder
		b.WriteString(fmt.Sprintf("Backups (%d):\n", len(backups)))
		for i := len(backups) - 1; i >= 0; i-- {
			bk := backups[i]
			b.WriteString(fmt.Sprintf("  %s (%d bytes, %s)\n", bk.Path, len(bk.Content), bk.Time.Format("15:04:05")))
		}
		m.messages = append(m.messages, Message{
			Role:    "system",
			Content: b.String(),
		})
		m.updateViewport()
		m.status = fmt.Sprintf("%d backup(s) available", len(backups))
		return
	}

	path := strings.Join(args, " ")
	if err := tools.RevertFile(path); err != nil {
		m.status = err.Error()
	} else {
		m.status = "Reverted: " + path
	}
}

// handleConfigCommand handles /config - displays active configuration
func (m *Model) handleConfigCommand() {
	cfg := config.Get()

	var b strings.Builder
	b.WriteString("Active Configuration\n\n")

	b.WriteString("Sources:\n")
	b.WriteString(fmt.Sprintf("  Global: %s/config.json\n", config.GetConfigDir()))
	if config.ProjectConfigLoaded() {
		b.WriteString(fmt.Sprintf("  Project: %s (active)\n", config.ProjectConfigPath()))
	} else {
		b.WriteString("  Project: none\n")
	}

	b.WriteString(fmt.Sprintf("\nDefault provider: %s\n", cfg.DefaultProvider))

	b.WriteString("\nSettings:\n")
	b.WriteString(fmt.Sprintf("  Max tool turns: %d\n", cfg.Settings.MaxToolTurns))
	b.WriteString(fmt.Sprintf("  Streaming buffer: %d\n", cfg.Settings.StreamingBuffer))
	b.WriteString(fmt.Sprintf("  Save sessions: %v\n", cfg.Settings.SaveSessions))
	if cfg.Settings.ColorTheme != "" {
		b.WriteString(fmt.Sprintf("  Theme: %s\n", cfg.Settings.ColorTheme))
	}
	b.WriteString(fmt.Sprintf("  Notifications: %v\n", config.NotificationsEnabled()))
	b.WriteString(fmt.Sprintf("  Sandbox: %v\n", cfg.Settings.Sandbox))

	nPre := len(cfg.Hooks.PreTool)
	nPost := len(cfg.Hooks.PostTool)
	nMsg := len(cfg.Hooks.OnMessage)
	total := nPre + nPost + nMsg
	if total > 0 {
		b.WriteString(fmt.Sprintf("\nHooks (%d total):\n", total))
		b.WriteString(fmt.Sprintf("  pre_tool: %d\n", nPre))
		b.WriteString(fmt.Sprintf("  post_tool: %d\n", nPost))
		b.WriteString(fmt.Sprintf("  on_message: %d\n", nMsg))
	} else {
		b.WriteString("\nHooks: none\n")
	}

	m.messages = append(m.messages, Message{
		Role:    "system",
		Content: b.String(),
	})
	m.updateViewport()

	if config.ProjectConfigLoaded() {
		m.status = "Config loaded (global + project)"
	} else {
		m.status = "Config loaded (global only)"
	}
}

// handleMCPCommand handles /mcp - lists MCP servers and their tools
func (m *Model) handleMCPCommand() {
	if mcp.Global == nil {
		m.status = "MCP not initialized"
		return
	}

	statuses := mcp.Global.Status()
	if len(statuses) == 0 {
		m.status = "No MCP servers configured"
		return
	}

	var b strings.Builder
	totalTools := 0
	connected := 0
	b.WriteString(fmt.Sprintf("MCP Servers (%d):\n\n", len(statuses)))
	for _, s := range statuses {
		icon := "x"
		if s.Connected {
			icon = "+"
			connected++
		}
		totalTools += s.ToolCount
		b.WriteString(fmt.Sprintf("  [%s] %s (%d tools)\n", icon, s.Name, s.ToolCount))

		// List tools for this server
		if s.Connected {
			if client, ok := mcp.Global.GetClient(s.Name); ok {
				for _, t := range client.Tools() {
					desc := t.Description
					if len(desc) > 50 {
						desc = desc[:47] + "..."
					}
					b.WriteString(fmt.Sprintf("      - %s: %s\n", t.Name, desc))
				}
			}
		}
	}

	m.messages = append(m.messages, Message{
		Role:    "system",
		Content: b.String(),
	})
	m.updateViewport()
	m.status = fmt.Sprintf("MCP: %d/%d connected, %d tools", connected, len(statuses), totalTools)
}

// handleMemoryCommand handles /memory [show|clear]
func (m *Model) handleMemoryCommand(args []string) {
	sub := "show"
	if len(args) > 0 {
		sub = strings.ToLower(args[0])
	}

	switch sub {
	case "show":
		content := config.LoadMemoryMD()
		if content == "" {
			m.status = "No memory file found (~/.poly/MEMORY.md)"
			return
		}
		m.messages = append(m.messages, Message{
			Role:    "system",
			Content: "MEMORY.md:\n\n" + content,
		})
		m.updateViewport()
		m.status = fmt.Sprintf("Memory loaded (%d bytes)", len(content))

	case "clear":
		if err := config.ClearMemoryMD(); err != nil {
			m.status = "No memory file to clear"
		} else {
			m.status = "Memory cleared"
		}

	default:
		m.status = "Usage: /memory [show|clear]"
	}
}

// handleStatsCommand handles /stats - displays session statistics
func (m *Model) handleStatsCommand() {
	var b strings.Builder
	b.WriteString("Session Statistics\n\n")

	// Message count
	userMsgs, assistantMsgs, toolCalls := 0, 0, 0
	for _, msg := range m.messages {
		switch msg.Role {
		case "user":
			userMsgs++
		case "assistant":
			assistantMsgs++
			toolCalls += len(msg.ToolCalls)
		}
	}
	b.WriteString(fmt.Sprintf("Messages: %d total (%d user, %d assistant)\n", userMsgs+assistantMsgs, userMsgs, assistantMsgs))

	// Token usage
	b.WriteString(fmt.Sprintf("Input tokens: %d\n", m.sessionInputTokens))
	b.WriteString(fmt.Sprintf("Output tokens: %d\n", m.sessionOutputTokens))
	if m.sessionCacheCreationTokens > 0 || m.sessionCacheReadTokens > 0 {
		b.WriteString(fmt.Sprintf("Cache tokens: %d created, %d read\n", m.sessionCacheCreationTokens, m.sessionCacheReadTokens))
	}

	// Cost
	b.WriteString(fmt.Sprintf("Cost: $%.4f\n", m.sessionCost))

	// Provider + model
	variant := m.modelVariant
	if variant == "" {
		variant = "default"
	}
	modelName := llm.GetDefaultModel(m.defaultProvider)
	if v, ok := llm.GetModelVariants()[m.defaultProvider]; ok {
		if mn, ok := v[variant]; ok {
			modelName = mn
		}
	}
	b.WriteString(fmt.Sprintf("Provider: @%s (%s)\n", m.defaultProvider, modelName))

	// Tool calls
	if toolCalls > 0 {
		b.WriteString(fmt.Sprintf("Tool calls: %d\n", toolCalls))
	}

	// Session duration
	duration := time.Since(m.sessionStartTime).Truncate(time.Second)
	b.WriteString(fmt.Sprintf("Session duration: %s\n", duration))

	m.messages = append(m.messages, Message{
		Role:    "system",
		Content: b.String(),
	})
	m.updateViewport()
	m.status = fmt.Sprintf("Stats: %d msgs, %d tokens, $%.4f", userMsgs+assistantMsgs, m.sessionInputTokens+m.sessionOutputTokens, m.sessionCost)
}

// handleCostsExport exports per-message cost data to CSV or JSON
func (m *Model) handleCostsExport(format string) {
	type costEntry struct {
		Index        int     `json:"index"`
		Provider     string  `json:"provider"`
		InputTokens  int     `json:"input_tokens"`
		OutputTokens int     `json:"output_tokens"`
		Cost         float64 `json:"cost"`
	}

	var entries []costEntry
	for i, msg := range m.messages {
		if msg.Role != "assistant" || (msg.InputTokens == 0 && msg.OutputTokens == 0) {
			continue
		}
		cost := calculateCost(msg.InputTokens, msg.OutputTokens, msg.Provider)
		entries = append(entries, costEntry{
			Index:        i,
			Provider:     msg.Provider,
			InputTokens:  msg.InputTokens,
			OutputTokens: msg.OutputTokens,
			Cost:         cost,
		})
	}

	if len(entries) == 0 {
		m.status = "No cost data to export"
		return
	}

	homeDir, _ := os.UserHomeDir()
	timestamp := time.Now().Format("20060102-150405")

	if format == "json" {
		path := filepath.Join(homeDir, "poly-costs-"+timestamp+".json")
		data, err := json.MarshalIndent(entries, "", "  ")
		if err != nil {
			m.status = "Export failed: " + err.Error()
			return
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			m.status = "Export failed: " + err.Error()
			return
		}
		m.status = "Costs exported to " + path
	} else {
		path := filepath.Join(homeDir, "poly-costs-"+timestamp+".csv")
		var b strings.Builder
		b.WriteString("index,provider,input_tokens,output_tokens,cost\n")
		for _, e := range entries {
			b.WriteString(fmt.Sprintf("%d,%s,%d,%d,%.6f\n", e.Index, e.Provider, e.InputTokens, e.OutputTokens, e.Cost))
		}
		if err := os.WriteFile(path, []byte(b.String()), 0644); err != nil {
			m.status = "Export failed: " + err.Error()
			return
		}
		m.status = "Costs exported to " + path
	}
}
