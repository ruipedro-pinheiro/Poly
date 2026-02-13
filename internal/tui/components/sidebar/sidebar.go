package sidebar

import (
	"encoding/json"
	"fmt"
	"image/color"
	"math"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/pedromelo/poly/internal/tui/core"
	"github.com/pedromelo/poly/internal/tui/layout"
	"github.com/pedromelo/poly/internal/tui/styles"
)

// ModifiedFile represents a tracked file with diff stats
type ModifiedFile struct {
	Path      string
	Additions int
	Deletions int
}

// ProviderStatus represents a provider's connection state for the sidebar
type ProviderStatus struct {
	Name      string
	Connected bool
	Color     color.Color
}

// Sidebar is the interface for the sidebar panel
type Sidebar interface {
	layout.Model
	layout.Sizeable
	SetProvider(name string, color color.Color)
	SetThinkingMode(bool)
	SetTokenInfo(input, output int, cost float64)
	SetModifiedFiles([]ModifiedFile)
	SetYoloMode(bool)
	SetProviders([]ProviderStatus)
}

type sidebarTodo struct {
	Content    string `json:"content"`
	Status     string `json:"status"`
	ActiveForm string `json:"active_form"`
}

type sidebarCmp struct {
	width, height int

	provider      string
	providerColor color.Color
	thinkingMode  bool
	inputTokens   int
	outputTokens  int
	cost          float64
	modifiedFiles []ModifiedFile
	yoloMode      bool
	providers     []ProviderStatus

	// Cache
	cacheMu      sync.Mutex
	todosCache   []sidebarTodo
	todosCacheAt time.Time
}

const cacheTTL = 5 * time.Second

// New creates a new sidebar component
func New() Sidebar {
	return &sidebarCmp{}
}

func (s *sidebarCmp) Init() tea.Cmd {
	return nil
}

func (s *sidebarCmp) Update(msg tea.Msg) (layout.Model, tea.Cmd) {
	return s, nil
}

// gradientText is a convenience wrapper around core.GradientText.
func gradientText(text string, from, to color.Color) string {
	return core.GradientText(text, from, to, false)
}

// ---------------------------------------------------------------------------
// Token formatting
// ---------------------------------------------------------------------------

func formatTokens(total int) string {
	if total >= 1_000_000 {
		m := float64(total) / 1_000_000.0
		return fmt.Sprintf("%.1fM", m)
	}
	if total >= 1_000 {
		k := float64(total) / 1_000.0
		if k >= 100 {
			return fmt.Sprintf("%dK", int(k))
		}
		return fmt.Sprintf("%.0fK", math.Round(k))
	}
	return fmt.Sprintf("%d", total)
}

// tokenPercent returns a rough percentage of context used.
func tokenPercent(total int) int {
	const maxCtx = layout.DefaultContextWindow
	if total <= 0 {
		return 0
	}
	pct := int(math.Round(float64(total) / float64(maxCtx) * 100))
	if pct > 100 {
		pct = 100
	}
	return pct
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (s *sidebarCmp) View() string {
	if s.width == 0 || s.height == 0 {
		return ""
	}

	sidebarStyle := lipgloss.NewStyle().
		Width(s.width).
		Height(s.height).
		BorderStyle(styles.Borders.Panel).
		BorderLeft(true).
		BorderForeground(styles.Surface1).
		PaddingLeft(1).
		PaddingRight(1)

	innerWidth := s.width - 4

	var sb strings.Builder

	// 1. Logo section
	s.renderLogo(&sb, innerWidth)
	sb.WriteString("\n")

	// 2. Model info section with centered header
	s.renderModelInfo(&sb, innerWidth)
	sb.WriteString("\n")

	// 3. YOLO mode warning
	if s.yoloMode {
		yoloStyle := lipgloss.NewStyle().
			Background(styles.Red).
			Foreground(styles.Base).
			Bold(true).
			Padding(0, 1)
		sb.WriteString("  " + yoloStyle.Render("YOLO") + "\n\n")
	}

	// 4. Modified files section
	s.renderModifiedFiles(&sb, innerWidth)
	sb.WriteString("\n")

	// 5. Providers section
	s.renderProviders(&sb, innerWidth)
	sb.WriteString("\n")

	// 6. Todos section
	s.renderTodos(&sb, innerWidth)

	return sidebarStyle.Render(sb.String())
}

// ---------------------------------------------------------------------------
// Logo section
// ---------------------------------------------------------------------------

func (s *sidebarCmp) renderLogo(sb *strings.Builder, width int) {
	diagLen := width
	if diagLen < 3 {
		diagLen = 3
	}

	diagLine := gradientText(strings.Repeat(core.IconDiag, diagLen), styles.Mauve, styles.Lavender)

	logoText := core.IconModel + " P O L Y"
	logoRendered := gradientText(logoText, styles.Mauve, styles.Lavender)

	sb.WriteString(diagLine + "\n")
	sb.WriteString(logoRendered + "\n")
	sb.WriteString(diagLine + "\n")
}

// ---------------------------------------------------------------------------
// Model info section
// ---------------------------------------------------------------------------

func (s *sidebarCmp) renderModelInfo(sb *strings.Builder, width int) {
	// Centered section header - "Active" not "Model" because Poly is multi-AI
	sb.WriteString(core.CenteredSection("Active", width, styles.Mauve, styles.Surface2) + "\n")

	provColor := s.providerColor
	if provColor == nil {
		provColor = styles.Overlay1
	}

	// Provider name with icon
	modelStyle := lipgloss.NewStyle().
		Foreground(provColor).
		Bold(true)
	sb.WriteString("  " + modelStyle.Render("@"+s.provider) + "\n")

	dimStyle := lipgloss.NewStyle().Foreground(styles.Overlay0)

	// Thinking mode
	if s.thinkingMode {
		thinkStyle := lipgloss.NewStyle().Foreground(styles.Lavender)
		sb.WriteString("  " + thinkStyle.Render("Thinking: on") + "\n")
	} else {
		sb.WriteString("  " + dimStyle.Render("Thinking: off") + "\n")
	}

	// Tokens + progress bar
	totalTokens := s.inputTokens + s.outputTokens
	tokenStr := formatTokens(totalTokens)
	pct := tokenPercent(totalTokens)
	costStr := fmt.Sprintf("$%.2f", s.cost)

	if totalTokens > 0 {
		valueStyle := lipgloss.NewStyle().Foreground(styles.Text)
		sb.WriteString("  " + valueStyle.Render(tokenStr+" tokens") + "\n")

		// Progress bar
		barWidth := width - 4
		if barWidth < 5 {
			barWidth = 5
		}
		bar := renderTokenBar(pct, barWidth)
		sb.WriteString("  " + bar + "\n")

		sb.WriteString("  " + dimStyle.Render("Cost: "+costStr) + "\n")
	} else {
		sb.WriteString("  " + dimStyle.Render("0 tokens") + "\n")
		barWidth := width - 4
		if barWidth < 5 {
			barWidth = 5
		}
		bar := renderTokenBar(0, barWidth)
		sb.WriteString("  " + bar + "\n")
		sb.WriteString("  " + dimStyle.Render("Cost: $0.00") + "\n")
	}
}

// renderTokenBar renders a visual progress bar like "▓▓▓░░░░░░░ 21%"
func renderTokenBar(pct int, width int) string {
	// Reserve space for " XX%" suffix (4 chars max)
	pctLabel := fmt.Sprintf(" %d%%", pct)
	barWidth := width - len(pctLabel)
	if barWidth < 3 {
		barWidth = 3
	}

	filled := int(math.Round(float64(barWidth) * float64(pct) / 100.0))
	if filled > barWidth {
		filled = barWidth
	}
	empty := barWidth - filled

	// Color based on percentage
	var filledColor color.Color
	switch {
	case pct >= 80:
		filledColor = styles.Red
	case pct >= 50:
		filledColor = styles.Yellow
	default:
		filledColor = styles.Mauve
	}

	filledStyle := lipgloss.NewStyle().Foreground(filledColor)
	emptyStyle := lipgloss.NewStyle().Foreground(styles.Surface1)
	pctStyle := lipgloss.NewStyle().Foreground(styles.Overlay0)

	return filledStyle.Render(strings.Repeat("\u2593", filled)) +
		emptyStyle.Render(strings.Repeat("\u2591", empty)) +
		pctStyle.Render(pctLabel)
}

// ---------------------------------------------------------------------------
// Modified files section
// ---------------------------------------------------------------------------

func (s *sidebarCmp) renderModifiedFiles(sb *strings.Builder, width int) {
	sb.WriteString(core.CenteredSection("Files", width, styles.Mauve, styles.Surface2) + "\n")

	dimStyle := lipgloss.NewStyle().Foreground(styles.Overlay0)

	if len(s.modifiedFiles) == 0 {
		sb.WriteString("  " + dimStyle.Render("None") + "\n")
		return
	}

	addStyle := lipgloss.NewStyle().Foreground(styles.Green)
	delStyle := lipgloss.NewStyle().Foreground(styles.Red)
	fileStyle := lipgloss.NewStyle().Foreground(styles.Text)

	for _, f := range s.modifiedFiles {
		// Build the stats suffix first so we know how much room the filename gets
		var stats string
		if f.Additions > 0 || f.Deletions > 0 {
			parts := make([]string, 0, 2)
			if f.Additions > 0 {
				parts = append(parts, addStyle.Render(fmt.Sprintf("+%d", f.Additions)))
			}
			if f.Deletions > 0 {
				parts = append(parts, delStyle.Render(fmt.Sprintf("-%d", f.Deletions)))
			}
			stats = " " + strings.Join(parts, " ")
		}

		// Truncate filename to fit
		display := filepath.Base(f.Path)
		maxName := width - 4
		if stats != "" {
			// rough estimate: +N -N takes about 8 visible chars
			maxName = width - 10
		}
		if maxName < 6 {
			maxName = 6
		}
		if len(display) > maxName {
			display = display[:maxName-3] + "..."
		}

		sb.WriteString("  " + fileStyle.Render(display) + stats + "\n")
	}
}

// ---------------------------------------------------------------------------
// Providers section
// ---------------------------------------------------------------------------

func (s *sidebarCmp) renderProviders(sb *strings.Builder, width int) {
	sb.WriteString(core.CenteredSection("Providers", width, styles.Mauve, styles.Surface2) + "\n")

	dimStyle := lipgloss.NewStyle().Foreground(styles.Overlay0)

	if len(s.providers) == 0 {
		sb.WriteString("  " + dimStyle.Render("None") + "\n")
		return
	}

	// Render providers in rows of 2
	checkStyle := lipgloss.NewStyle().Foreground(styles.Green)
	crossStyle := lipgloss.NewStyle().Foreground(styles.Red)

	row := make([]string, 0, 2)
	for i, p := range s.providers {
		var icon string
		if p.Connected {
			icon = checkStyle.Render(core.IconCheck)
		} else {
			icon = crossStyle.Render(core.IconError)
		}

		nameStyle := lipgloss.NewStyle().Foreground(styles.Text)
		if !p.Connected {
			nameStyle = dimStyle
		}

		// Pad name to fixed width for alignment
		name := p.Name
		if len(name) > 7 {
			name = name[:7]
		}
		entry := icon + " " + nameStyle.Render(name)

		row = append(row, entry)

		// Two per line
		if len(row) == 2 || i == len(s.providers)-1 {
			sb.WriteString("  " + strings.Join(row, "  ") + "\n")
			row = row[:0]
		}
	}
}

// ---------------------------------------------------------------------------
// Todos section
// ---------------------------------------------------------------------------

func (s *sidebarCmp) renderTodos(sb *strings.Builder, width int) {
	sb.WriteString(core.CenteredSection("Todos", width, styles.Mauve, styles.Surface2) + "\n")

	dimStyle := lipgloss.NewStyle().Foreground(styles.Overlay0)

	todos := s.getCachedTodos()
	if todos == nil {
		sb.WriteString("  " + dimStyle.Render("No todos") + "\n")
		return
	}

	pending, inProgress, completed := 0, 0, 0
	for _, todo := range todos {
		switch todo.Status {
		case "pending":
			pending++
		case "in_progress":
			inProgress++
		case "completed":
			completed++
		}
	}

	countStyle := lipgloss.NewStyle().Foreground(styles.Subtext0)
	sb.WriteString("  " + countStyle.Render(
		fmt.Sprintf("%d pending  %d active  %d done", pending, inProgress, completed),
	) + "\n")

	// Filter: only show active/pending todos in the list, completed just in count
	var activeTodos []sidebarTodo
	for _, todo := range todos {
		if todo.Status != "completed" {
			activeTodos = append(activeTodos, todo)
		}
	}

	// Sort: in_progress first, then pending
	sort.Slice(activeTodos, func(i, j int) bool {
		if activeTodos[i].Status != activeTodos[j].Status {
			return activeTodos[i].Status == "in_progress"
		}
		return false
	})

	// Show all active/pending todos (no cap)
	for _, todo := range activeTodos {
		icon := "\u25CB"
		iconColor := styles.Overlay0
		if todo.Status == "in_progress" {
			icon = core.IconActive
			iconColor = styles.Mauve
		}

		iconStyle := lipgloss.NewStyle().Foreground(iconColor)

		short := todo.ActiveForm
		if short == "" {
			short = todo.Content
		}
		maxLen := width - 6
		if maxLen < 10 {
			maxLen = 10
		}
		if len(short) > maxLen {
			short = short[:maxLen-3] + "..."
		}

		textStyle := lipgloss.NewStyle().Foreground(styles.Text)
		sb.WriteString("  " + iconStyle.Render(icon) + " " + textStyle.Render(short) + "\n")
	}
}

// ---------------------------------------------------------------------------
// Cache helpers
// ---------------------------------------------------------------------------

func (s *sidebarCmp) getCachedTodos() []sidebarTodo {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	if time.Since(s.todosCacheAt) < cacheTTL && s.todosCache != nil {
		return s.todosCache
	}

	u, _ := user.Current()
	if u == nil {
		return nil
	}
	todosPath := filepath.Join(u.HomeDir, ".poly", "todos.json")
	data, err := os.ReadFile(todosPath)
	if err != nil {
		return nil
	}

	var todos []sidebarTodo
	json.Unmarshal(data, &todos)
	s.todosCache = todos
	s.todosCacheAt = time.Now()
	return todos
}


// ---------------------------------------------------------------------------
// Setters
// ---------------------------------------------------------------------------

func (s *sidebarCmp) SetSize(width, height int) tea.Cmd {
	s.width = width
	s.height = height
	return nil
}

func (s *sidebarCmp) GetSize() (int, int) {
	return s.width, s.height
}

func (s *sidebarCmp) SetProvider(name string, color color.Color) {
	s.provider = name
	s.providerColor = color
}

func (s *sidebarCmp) SetThinkingMode(on bool) {
	s.thinkingMode = on
}

func (s *sidebarCmp) SetTokenInfo(input, output int, cost float64) {
	s.inputTokens = input
	s.outputTokens = output
	s.cost = cost
}

func (s *sidebarCmp) SetModifiedFiles(files []ModifiedFile) {
	s.modifiedFiles = files
}

func (s *sidebarCmp) SetYoloMode(on bool) {
	s.yoloMode = on
}

func (s *sidebarCmp) SetProviders(providers []ProviderStatus) {
	s.providers = providers
}
