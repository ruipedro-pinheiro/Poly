package tui

import (
	"context"
	"sort"
	"time"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/pedromelo/poly/internal/config"
	"github.com/pedromelo/poly/internal/llm"
	"github.com/pedromelo/poly/internal/sandbox"
	"github.com/pedromelo/poly/internal/session"
	"github.com/pedromelo/poly/internal/theme"
	"github.com/pedromelo/poly/internal/tools"
	"github.com/pedromelo/poly/internal/updater"
	tuiLayout "github.com/pedromelo/poly/internal/tui/layout"
	"github.com/pedromelo/poly/internal/tui/styles"
	"github.com/pedromelo/poly/internal/tui/components/header"
	"github.com/pedromelo/poly/internal/tui/components/infopanel"
	"github.com/pedromelo/poly/internal/tui/components/splash"
	"github.com/pedromelo/poly/internal/tui/components/status"
)

// Model is the main TUI model
type Model struct {
	width  int
	height int
	layout LayoutContext

	state           viewState
	defaultProvider string
	ready           bool

	viewport viewport.Model
	textarea textarea.Model
	keys     KeyMap

	messages   []Message
	status     string
	focused    string // "input" or "messages"
	isStreaming bool

	// Providers
	providers map[string]llm.Provider
	cancelCtx context.CancelFunc

	// Model settings
	modelVariant string // "default", "fast", "think", "opus"
	thinkingMode bool

	// Control Room state
	controlRoomIndex     int
	controlRoomProviders []string
	oauthPending         string // provider waiting for OAuth code
	apiKeyPending        string // provider waiting for API key
	authInput            string // user input for code/key
	authStatusMsg        string // success/error message shown in Control Room

	// Model Picker state
	modelPickerIndex   int
	modelPickerModels  []modelOption
	modelPickerFilter  string
	recentModels       []modelOption

	// Cascade state for @all orchestration
	cascade *cascadeState

	// Add Provider form state
	addProviderField  int      // 0=id, 1=url, 2=apikey, 3=model, 4=format
	addProviderValues []string // [id, url, apikey, model]
	addProviderFormat int      // 0=openai, 1=anthropic, 2=google

	// Pending image attachments for next message
	pendingImages     [][]byte
	pendingImageTypes []string

	// Token/Cost tracking (Phase 1)
	sessionInputTokens         int
	sessionOutputTokens        int
	sessionCacheCreationTokens int
	sessionCacheReadTokens     int
	sessionCost                float64
	sessionStartTime           time.Time

	// Response time tracking
	streamStartTime  time.Time
	streamTokenCount int

	modifiedFiles   []string

	// Command palette state
	paletteFilter string
	paletteIndex  int
	paletteCommands []CommandEntry

	// Thinking display state
	thinkingExpanded map[int]bool

	// Permission/Approval state (Phase 2)
	pendingApproval tools.PendingApproval
	approvedTools   map[string]bool
	approvalIndex   int // 0=Allow, 1=Allow Always, 2=YOLO

	// Input history (arrow up/down like a shell)
	inputHistory      []string
	inputHistoryIdx   int    // -1 means "new message" (not browsing history)
	inputHistoryDraft string // saves current draft when browsing

	// Desktop notifications
	notificationsOn bool

	// Context compaction state
	isCompacting bool

	// Persistent file context (/add, /remove, /context)
	contextFiles []string

	// Retry state: when set, handleSendKey will re-send this content
	retryContent string

	// Skill state: when set, handleSendKey will send this as user message
	skillContent string

	// Compare state (/compare)
	compareExpected int
	compareReceived int
	comparePending  []tea.Cmd // pending commands from /compare

	// Command registry
	commands *CommandRegistry

	// Tab completion state
	completion completionState

	// Session list state (Phase 3)
	sessionListIndex     int
	sessionListFilter    string // text filter for session names
	sessionListFiltering bool   // true when filter input is active

	// Components
	headerBar     header.Header
	statusBar     status.StatusCmp
	splashCmp     splash.Splash
	infoPanelCmp  infopanel.InfoPanel
}

// New creates a new TUI model
func New() Model {
	ta := textarea.New()
	ta.Placeholder = "Type a message..."
	ta.Focus()
	ta.Prompt = ""
	ta.CharLimit = 0
	ta.ShowLineNumbers = false
	ta.SetStyles(textarea.Styles{
		Focused: textarea.StyleState{
			Base:        lipgloss.NewStyle(),
			CursorLine:  lipgloss.NewStyle(),
			Placeholder: lipgloss.NewStyle().Foreground(theme.Overlay0).Italic(true),
			Prompt:      lipgloss.NewStyle().Foreground(theme.Mauve),
		},
		Blurred: textarea.StyleState{
			Base:        lipgloss.NewStyle(),
			CursorLine:  lipgloss.NewStyle(),
			Placeholder: lipgloss.NewStyle().Foreground(theme.Overlay0).Italic(true),
			Prompt:      lipgloss.NewStyle().Foreground(theme.Overlay0),
		},
	})
	ta.SetHeight(1)
	ta.KeyMap.InsertNewline.SetKeys("shift+enter")
	ta.KeyMap.InsertNewline.SetEnabled(true)

	// Get providers from registry (auto-registered via init())
	providers := llm.GetAllProviders()

	// Provider list from registry (sorted for consistent display)
	providerList := llm.GetProviderNames()

	// Build model list from config (dynamic, not hardcoded)
	modelList := []modelOption{}
	modelVariants := llm.GetModelVariants()
	for _, prov := range providerList {
		if variants, ok := modelVariants[prov]; ok {
			// Build sorted variant keys, with "default" first
			keys := make([]string, 0, len(variants))
			for k := range variants {
				if k != "default" {
					keys = append(keys, k)
				}
			}
			sort.Strings(keys)
			if _, hasDefault := variants["default"]; hasDefault {
				keys = append([]string{"default"}, keys...)
			}
			for _, v := range keys {
				modelName := variants[v]
				display := prov + " " + v
				if v == "default" {
					display = prov + " (" + modelName + ")"
				}
				modelList = append(modelList, modelOption{
					provider: prov,
					variant:  v,
					display:  display,
				})
			}
		}
	}

	// Load session (persisted messages)
	sess, _ := session.Load()
	var messages []Message
	if sess != nil {
		messages = make([]Message, 0, len(sess.Messages))
		for _, m := range sess.Messages {
			messages = append(messages, Message{
				Role:     m.Role,
				Content:  m.Content,
				Provider: m.Provider,
				Thinking: m.Thinking,
				Images:   m.Images,
			})
		}
	}

	defaultProvider := ""
	names := config.GetProviderNames()
	if len(names) > 0 {
		defaultProvider = names[0]
	}
	if sess != nil && sess.Provider != "" {
		defaultProvider = sess.Provider
	}

	// Initialize markdown renderer with a default width
	initMarkdown(80)

	return Model{
		state:                viewSplash,
		defaultProvider:      defaultProvider,
		keys:                 DefaultKeyMap(),
		textarea:             ta,
		messages:             messages,
		status:               "Ready",
		focused:              "input",
		providers:            providers,
		controlRoomProviders: providerList,
		modelPickerModels:    modelList,
		notificationsOn:      config.NotificationsEnabled(),
		thinkingMode:         true,
		modelVariant:         "think",
		thinkingExpanded:     make(map[int]bool),
		paletteCommands:      buildCommandList(),
		commands:             initCommands(),
		inputHistory:         config.LoadHistory(),
		inputHistoryIdx:      -1,
		approvedTools:        make(map[string]bool),
		sessionStartTime:     time.Now(),
		headerBar:            header.New(),
		statusBar:            status.New(),
		splashCmp:            splash.New(),
		infoPanelCmp:         infopanel.New(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, watchForApprovals(), checkForUpdate())
}

// syncTextareaHeight adjusts textarea height (1 to InputMaxLines) based on content,
// then recalculates the layout so the viewport shrinks/grows accordingly.
func (m *Model) syncTextareaHeight() {
	lines := m.textarea.LineCount()
	if lines < 1 {
		lines = 1
	}
	if lines > tuiLayout.InputMaxLines {
		lines = tuiLayout.InputMaxLines
	}

	// Only update if height actually changed
	if m.textarea.Height() == lines {
		return
	}

	m.textarea.SetHeight(lines)

	// Recalculate layout with the new editor height
	editorH := lines + tuiLayout.InputBoxChrome
	m.layout = ComputeLayoutWithEditor(m.width, m.height, editorH)
	m.viewport.SetHeight(m.layout.ViewportHeight)
	m.updateViewport()
}

// checkForUpdate runs the update check in a goroutine and returns an UpdateAvailableMsg
func checkForUpdate() tea.Cmd {
	return func() tea.Msg {
		if v := updater.CheckForUpdate(); v != "" {
			return UpdateAvailableMsg{Version: v}
		}
		return nil
	}
}

// buildCommandList returns all available commands for the palette
func buildCommandList() []CommandEntry {
	return []CommandEntry{
		{Name: "New Session", Shortcut: "ctrl+n", Action: func(m *Model) {
			m.messages = []Message{}
			session.Clear()
			m.updateViewport()
			m.status = "New session"
		}},
		{Name: "Switch Model", Shortcut: "ctrl+o", Action: func(m *Model) {
			m.state = viewModelPicker
			m.modelPickerIndex = 0
			m.modelPickerFilter = ""
		}},
		{Name: "Toggle Thinking", Shortcut: "", Action: func(m *Model) {
			m.thinkingMode = !m.thinkingMode
			if m.thinkingMode {
				m.modelVariant = "think"
				m.status = "Thinking mode ON"
			} else {
				m.modelVariant = "default"
				m.status = "Thinking mode OFF"
			}
		}},
		{Name: "Session List", Shortcut: "ctrl+s", Action: func(m *Model) {
			m.state = viewSessionList
			m.sessionListIndex = 0
		}},
		{Name: "Control Room", Shortcut: "ctrl+d", Action: func(m *Model) {
			m.state = viewControlRoom
			m.controlRoomIndex = 0
		}},
		{Name: "Toggle Help", Shortcut: "ctrl+h", Action: func(m *Model) {
			if m.state == viewHelp {
				m.state = viewChat
			} else {
				m.state = viewHelp
			}
		}},
		{Name: "Clear Chat", Shortcut: "ctrl+l", Action: func(m *Model) {
			m.messages = []Message{}
			session.Clear()
			m.updateViewport()
			m.status = "Chat cleared"
		}},
		{Name: "Compact Context", Shortcut: "", Action: func(m *Model) {
			if !m.isStreaming && len(m.messages) > llm.MinMessagesToKeep {
				m.isCompacting = true
				m.status = "Compacting context..."
			} else {
				m.status = "Not enough messages to compact"
			}
		}},
		{Name: "Cycle Theme", Shortcut: "", Action: func(m *Model) {
			next := styles.NextTheme()
			theme.SetTheme(next)
			config.SetColorTheme(string(next))
			m.status = "Theme: " + string(next)
		}},
		{Name: "Toggle Notifications", Shortcut: "", Action: func(m *Model) {
			m.notificationsOn = !m.notificationsOn
			config.SetNotifications(m.notificationsOn)
			if m.notificationsOn {
				m.status = "Notifications ON"
			} else {
				m.status = "Notifications OFF"
			}
		}},
		{Name: "Toggle Sandbox", Shortcut: "", Action: func(m *Model) {
			if !sandbox.Available() {
				m.status = "No container runtime found (install podman or docker)"
			} else {
				sandbox.Enabled = !sandbox.Enabled
				config.SetSandbox(sandbox.Enabled)
				if sandbox.Enabled {
					m.status = "Sandbox ON (" + sandbox.Detect() + ")"
				} else {
					m.status = "Sandbox OFF"
				}
			}
		}},
		{Name: "Toggle YOLO Mode", Shortcut: "", Action: func(m *Model) {
			tools.YoloMode = !tools.YoloMode
			if tools.YoloMode {
				m.status = "YOLO mode ON"
			} else {
				tools.ResetAllowList()
				m.status = "YOLO mode OFF"
			}
		}},
		{Name: "Export Markdown", Shortcut: "", Action: func(m *Model) {
			if path, err := session.ExportMarkdown(); err != nil {
				m.status = "Export failed: " + err.Error()
			} else {
				m.status = "Exported to " + path
			}
		}},
		{Name: "Export JSON", Shortcut: "", Action: func(m *Model) {
			if path, err := session.ExportJSON(); err != nil {
				m.status = "Export failed: " + err.Error()
			} else {
				m.status = "Exported to " + path
			}
		}},
		{Name: "Quit", Shortcut: "ctrl+c", Action: nil}, // handled specially
	}
}
