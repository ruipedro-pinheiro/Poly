package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pedromelo/poly/internal/config"
	"github.com/pedromelo/poly/internal/llm"
	"github.com/pedromelo/poly/internal/sandbox"
	"github.com/pedromelo/poly/internal/session"
	"github.com/pedromelo/poly/internal/skills"
	"github.com/pedromelo/poly/internal/theme"
	"github.com/pedromelo/poly/internal/tools"
	"github.com/pedromelo/poly/internal/tui/styles"
	"github.com/pedromelo/poly/internal/updater"
)

// initCommands builds the command registry with all slash commands.
func initCommands() *CommandRegistry {
	r := NewCommandRegistry()

	r.Register(&Command{
		Name:        "clear",
		Aliases:     []string{"c"},
		Description: "Clear conversation",
		Usage:       "/clear",
		Handler: func(m *Model, args []string) {
			m.messages = []Message{}
			session.Clear()
			m.updateViewport()
			m.status = "Chat cleared"
		},
	})

	r.Register(&Command{
		Name:        "model",
		Aliases:     []string{"m"},
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
		Description: "Show available commands",
		Usage:       "/help",
		Handler: func(m *Model, args []string) {
			m.status = r.HelpString()
		},
	})

	r.Register(&Command{
		Name:        "providers",
		Aliases:     []string{"list"},
		Description: "List configured providers",
		Usage:       "/providers",
		Handler: func(m *Model, args []string) {
			names := llm.GetProviderNames()
			m.status = "Providers: " + strings.Join(names, ", ")
		},
	})

	r.Register(&Command{
		Name:        "add",
		Description: "Add file(s) to persistent context",
		Usage:       "/add <file_or_dir>",
		Handler: func(m *Model, args []string) {
			m.handleAddFileCommand(args)
		},
	})

	r.Register(&Command{
		Name:        "remove",
		Description: "Remove file from context",
		Usage:       "/remove <file>",
		Handler: func(m *Model, args []string) {
			m.handleRemoveFileCommand(args)
		},
	})

	r.Register(&Command{
		Name:        "context",
		Description: "List files in persistent context",
		Usage:       "/context",
		Handler: func(m *Model, args []string) {
			m.handleContextCommand()
		},
	})

	r.Register(&Command{
		Name:        "addprovider",
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
				Name:      strings.Title(args[0]),
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
		Name:        "sidebar",
		Description: "Toggle sidebar visibility",
		Usage:       "/sidebar",
		Handler: func(m *Model, args []string) {
			m.sidebarVisible = !m.sidebarVisible
			m.layout = ComputeLayout(m.width, m.height, m.sidebarVisible)
			if m.sidebarVisible {
				m.status = "Sidebar visible"
			} else {
				m.status = "Sidebar hidden"
			}
		},
	})

	r.Register(&Command{
		Name:        "compact",
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
		Name:        "export",
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
		Description: "Show version",
		Usage:       "/version",
		Handler: func(m *Model, args []string) {
			m.status = "Poly v" + updater.CurrentVersion
		},
	})

	r.Register(&Command{
		Name:        "undo",
		Description: "Undo last exchange",
		Usage:       "/undo",
		Handler: func(m *Model, args []string) {
			m.handleUndo()
		},
	})

	r.Register(&Command{
		Name:        "retry",
		Description: "Retry last message",
		Usage:       "/retry",
		Handler: func(m *Model, args []string) {
			m.handleRetry()
		},
	})

	r.Register(&Command{
		Name:        "compare",
		Description: "Compare providers side by side",
		Usage:       "/compare [providers] <prompt>",
		Handler: func(m *Model, args []string) {
			m.handleCompareCommand(args)
		},
	})

	r.Register(&Command{
		Name:        "config",
		Description: "Show active configuration",
		Usage:       "/config",
		Handler: func(m *Model, args []string) {
			m.handleConfigCommand()
		},
	})

	r.Register(&Command{
		Name:        "revert",
		Description: "Revert file changes",
		Usage:       "/revert [list|<file>]",
		Handler: func(m *Model, args []string) {
			m.handleRevertCommand(args)
		},
	})

	r.Register(&Command{
		Name:        "project",
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
		Description: "List or execute skills",
		Usage:       "/skill [name]",
		Handler: func(m *Model, args []string) {
			m.handleSkillCommand(args)
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
	if strings.Contains(content, "@claude") {
		return "claude"
	}
	if strings.Contains(content, "@gpt") {
		return "gpt"
	}
	if strings.Contains(content, "@gemini") {
		return "gemini"
	}
	if strings.Contains(content, "@grok") {
		return "grok"
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
			Role:     msg.Role,
			Content:  msg.Content,
			Provider: msg.Provider,
			Thinking: msg.Thinking,
			Images:   msg.Images,
		}
	}
	session.SetMessages(sessionMsgs)

	m.updateViewport()
	m.status = "Last exchange removed"
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
			Role:     msg.Role,
			Content:  msg.Content,
			Provider: msg.Provider,
			Thinking: msg.Thinking,
			Images:   msg.Images,
		}
	}
	session.SetMessages(sessionMsgs)

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
