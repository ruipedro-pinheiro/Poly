package infopanel

import (
	"fmt"
	"image/color"
	"math"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/pedromelo/poly/internal/theme"
	"github.com/pedromelo/poly/internal/tui/core"
	"github.com/pedromelo/poly/internal/tui/layout"
)

// PanelWidth is the fixed width of the info panel overlay.
const PanelWidth = 35

// ModifiedFile represents a tracked file with diff stats.
type ModifiedFile struct {
	Path      string
	Additions int
	Deletions int
}

// ProviderStatus represents a provider's connection state.
type ProviderStatus struct {
	Name       string
	Connected  bool
	Color      color.Color
	Cost       float64
	HasPricing bool
}

// MCPServer represents an MCP server's connection state.
type MCPServer struct {
	Name      string
	Connected bool
	ToolCount int
}

// InfoPanel is the interface for the info panel overlay component.
type InfoPanel interface {
	layout.Model
	layout.Sizeable
	SetProvider(name string, color color.Color)
	SetThinkingMode(bool)
	SetTokenInfo(input, output, cacheCreation, cacheRead int, cost float64)
	SetModifiedFiles([]ModifiedFile)
	SetYoloMode(bool)
	SetProviders([]ProviderStatus)
	SetMCPServers([]MCPServer)
	SetSandboxMode(bool)
	IsVisible() bool
	Toggle()
}

type infoPanelCmp struct {
	width, height int
	visible       bool
	viewport      viewport.Model

	provider      string
	providerColor color.Color
	thinkingMode  bool
	inputTokens   int
	outputTokens  int
	cacheCreation int
	cacheRead     int
	cost          float64
	modifiedFiles []ModifiedFile
	yoloMode      bool
	sandboxMode   bool
	providers     []ProviderStatus
	mcpServers    []MCPServer
}

// New creates a new info panel component.
func New() InfoPanel {
	vp := viewport.New()
	return &infoPanelCmp{
		viewport: vp,
	}
}

func (p *infoPanelCmp) Init() tea.Cmd {
	return nil
}

func (p *infoPanelCmp) Update(msg tea.Msg) (layout.Model, tea.Cmd) {
	if !p.visible {
		return p, nil
	}

	var cmd tea.Cmd
	p.viewport, cmd = p.viewport.Update(msg)
	return p, cmd
}

func (p *infoPanelCmp) View() string {
	if !p.visible || p.width == 0 || p.height == 0 {
		return ""
	}

	innerWidth := PanelWidth - 4 // border (1) + padding (2) + breathing room (1)

	var sb strings.Builder

	// 1. Title
	p.renderTitle(&sb, innerWidth)
	sb.WriteString("\n")

	// 2. Session (tokens, cost, context bar)
	p.renderSession(&sb, innerWidth)
	sb.WriteString("\n")

	// 3. Providers
	p.renderProviders(&sb, innerWidth)
	sb.WriteString("\n")

	// 4. Modified Files
	p.renderModifiedFiles(&sb, innerWidth)
	sb.WriteString("\n")

	// 5. MCP
	p.renderMCP(&sb, innerWidth)
	sb.WriteString("\n")

	// 6. Status badges
	p.renderStatus(&sb, innerWidth)

	content := sb.String()

	// Use viewport for scrolling if content exceeds height
	availableHeight := p.height - 2 // account for border chrome
	contentLines := strings.Count(content, "\n") + 1

	if contentLines > availableHeight {
		p.viewport.SetWidth(PanelWidth - 2) // inside border
		p.viewport.SetHeight(availableHeight)
		p.viewport.SetContent(content)
		content = p.viewport.View()
	}

	panelStyle := lipgloss.NewStyle().
		Width(PanelWidth).
		Height(p.height).
		Background(theme.Mantle).
		BorderStyle(lipgloss.NormalBorder()).
		BorderLeft(true).
		BorderForeground(theme.Surface1).
		PaddingLeft(1).
		PaddingRight(1)

	return panelStyle.Render(content)
}

// ---------------------------------------------------------------------------
// Title section
// ---------------------------------------------------------------------------

func (p *infoPanelCmp) renderTitle(sb *strings.Builder, width int) {
	diagLen := width
	if diagLen < 3 {
		diagLen = 3
	}
	diagLine := core.GradientText(strings.Repeat(core.IconDiag, diagLen), theme.Mauve, theme.Lavender, false)

	title := core.IconInfo + " Info"
	titleRendered := core.GradientText(title, theme.Mauve, theme.Lavender, true)

	hintStyle := lipgloss.NewStyle().Foreground(theme.Overlay0)
	hint := hintStyle.Render("(ctrl+i)")

	// Center the hint on the right
	titleWidth := lipgloss.Width(titleRendered)
	hintWidth := lipgloss.Width(hint)
	gap := width - titleWidth - hintWidth
	if gap < 1 {
		gap = 1
	}

	sb.WriteString(diagLine + "\n")
	sb.WriteString(titleRendered + strings.Repeat(" ", gap) + hint + "\n")
	sb.WriteString(diagLine + "\n")
}

// ---------------------------------------------------------------------------
// Session section (tokens, cost, context %)
// ---------------------------------------------------------------------------

func (p *infoPanelCmp) renderSession(sb *strings.Builder, width int) {
	sb.WriteString(core.CenteredSection("Session", width, theme.Mauve, theme.Surface2) + "\n")

	dimStyle := lipgloss.NewStyle().Foreground(theme.Overlay0)
	valueStyle := lipgloss.NewStyle().Foreground(theme.Text)

	// Input / Output tokens
	inputStr := formatTokens(p.inputTokens)
	outputStr := formatTokens(p.outputTokens)
	totalTokens := p.inputTokens + p.outputTokens

	if totalTokens > 0 {
		tokenLine := valueStyle.Render(inputStr+"↑") + " " + valueStyle.Render(outputStr+"↓")
		sb.WriteString("  " + tokenLine + "\n")

		// Cache info if available
		if p.cacheRead > 0 || p.cacheCreation > 0 {
			cacheStyle := lipgloss.NewStyle().Foreground(theme.Surface2)
			parts := make([]string, 0, 2)
			if p.cacheRead > 0 {
				parts = append(parts, formatTokens(p.cacheRead)+" cached")
			}
			if p.cacheCreation > 0 {
				parts = append(parts, formatTokens(p.cacheCreation)+" new")
			}
			sb.WriteString("  " + cacheStyle.Render("("+strings.Join(parts, ", ")+")") + "\n")
		}
	} else {
		sb.WriteString("  " + dimStyle.Render("0 tokens") + "\n")
	}

	// Cost
	costStr := fmt.Sprintf("$%.2f", p.cost)
	if p.cost > 0 {
		costStyle := lipgloss.NewStyle().Foreground(theme.Subtext0)
		sb.WriteString("  " + costStyle.Render("Cost: "+costStr) + "\n")
	} else {
		sb.WriteString("  " + dimStyle.Render("Cost: $0.00") + "\n")
	}

	// Context % bar
	pct := tokenPercent(totalTokens)
	barWidth := width - 4
	if barWidth < 5 {
		barWidth = 5
	}
	bar := renderTokenBar(pct, barWidth)
	sb.WriteString("  " + bar + "\n")
}

// ---------------------------------------------------------------------------
// Providers section
// ---------------------------------------------------------------------------

func (p *infoPanelCmp) renderProviders(sb *strings.Builder, width int) {
	sb.WriteString(core.CenteredSection("Providers", width, theme.Mauve, theme.Surface2) + "\n")

	dimStyle := lipgloss.NewStyle().Foreground(theme.Overlay0)

	if len(p.providers) == 0 {
		sb.WriteString("  " + dimStyle.Render("None") + "\n")
		return
	}

	checkStyle := lipgloss.NewStyle().Foreground(theme.Green)
	crossStyle := lipgloss.NewStyle().Foreground(theme.Red)

	for _, prov := range p.providers {
		var icon string
		if prov.Connected {
			icon = checkStyle.Render(core.IconCheck)
		} else {
			icon = crossStyle.Render(core.IconError)
		}

		nameColor := theme.Text
		if prov.Color != nil {
			nameColor = prov.Color
		}
		if !prov.Connected {
			nameColor = theme.Overlay0
		}
		nameStyle := lipgloss.NewStyle().Foreground(nameColor)

		name := prov.Name
		maxName := width - 6
		if maxName < 5 {
			maxName = 5
		}
		if len(name) > maxName {
			name = name[:maxName-2] + ".."
		}

		// Cost display
		costInfo := ""
		if prov.Cost > 0 {
			costStyle := lipgloss.NewStyle().Foreground(theme.Subtext0)
			costInfo = " " + costStyle.Render(fmt.Sprintf("$%.2f", prov.Cost))
		} else if !prov.HasPricing {
			naStyle := lipgloss.NewStyle().Foreground(theme.Surface2)
			costInfo = " " + naStyle.Render("N/A")
		}

		sb.WriteString("  " + icon + " " + nameStyle.Render(name) + costInfo + "\n")
	}
}

// ---------------------------------------------------------------------------
// Modified Files section
// ---------------------------------------------------------------------------

func (p *infoPanelCmp) renderModifiedFiles(sb *strings.Builder, width int) {
	sb.WriteString(core.CenteredSection("Files", width, theme.Mauve, theme.Surface2) + "\n")

	dimStyle := lipgloss.NewStyle().Foreground(theme.Overlay0)

	if len(p.modifiedFiles) == 0 {
		sb.WriteString("  " + dimStyle.Render("None") + "\n")
		return
	}

	addStyle := lipgloss.NewStyle().Foreground(theme.Green)
	delStyle := lipgloss.NewStyle().Foreground(theme.Red)
	fileStyle := lipgloss.NewStyle().Foreground(theme.Text)

	for _, f := range p.modifiedFiles {
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

		display := filepath.Base(f.Path)
		maxName := width - 4
		if stats != "" {
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
// MCP section
// ---------------------------------------------------------------------------

func (p *infoPanelCmp) renderMCP(sb *strings.Builder, width int) {
	sb.WriteString(core.CenteredSection("MCP", width, theme.Mauve, theme.Surface2) + "\n")

	dimStyle := lipgloss.NewStyle().Foreground(theme.Overlay0)

	if len(p.mcpServers) == 0 {
		sb.WriteString("  " + dimStyle.Render("None") + "\n")
		return
	}

	checkStyle := lipgloss.NewStyle().Foreground(theme.Green)
	crossStyle := lipgloss.NewStyle().Foreground(theme.Red)
	countStyle := lipgloss.NewStyle().Foreground(theme.Overlay0)

	for _, srv := range p.mcpServers {
		var icon string
		if srv.Connected {
			icon = checkStyle.Render(core.IconCheck)
		} else {
			icon = crossStyle.Render(core.IconError)
		}

		nameStyle := lipgloss.NewStyle().Foreground(theme.Text)
		if !srv.Connected {
			nameStyle = dimStyle
		}

		name := srv.Name
		maxName := width - 12
		if maxName < 5 {
			maxName = 5
		}
		if len(name) > maxName {
			name = name[:maxName-2] + ".."
		}

		toolInfo := ""
		if srv.ToolCount > 0 {
			toolInfo = " " + countStyle.Render(fmt.Sprintf("(%d tools)", srv.ToolCount))
		}

		sb.WriteString("  " + icon + " " + nameStyle.Render(name) + toolInfo + "\n")
	}
}

// ---------------------------------------------------------------------------
// Status section (badges)
// ---------------------------------------------------------------------------

func (p *infoPanelCmp) renderStatus(sb *strings.Builder, width int) {
	sb.WriteString(core.CenteredSection("Status", width, theme.Mauve, theme.Surface2) + "\n")

	var badges []string

	if p.yoloMode {
		badge := lipgloss.NewStyle().
			Background(theme.Red).
			Foreground(theme.Base).
			Bold(true).
			Padding(0, 1).
			Render("YOLO")
		badges = append(badges, badge)
	}

	if p.sandboxMode {
		badge := lipgloss.NewStyle().
			Background(theme.Blue).
			Foreground(theme.Base).
			Bold(true).
			Padding(0, 1).
			Render("SANDBOX")
		badges = append(badges, badge)
	}

	if p.thinkingMode {
		badge := lipgloss.NewStyle().
			Background(theme.Lavender).
			Foreground(theme.Base).
			Bold(true).
			Padding(0, 1).
			Render("THINKING")
		badges = append(badges, badge)
	}

	if len(badges) == 0 {
		dimStyle := lipgloss.NewStyle().Foreground(theme.Overlay0)
		sb.WriteString("  " + dimStyle.Render("Standard mode") + "\n")
		return
	}

	sb.WriteString("  " + strings.Join(badges, " ") + "\n")
}

// ---------------------------------------------------------------------------
// Token helpers
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

func renderTokenBar(pct int, width int) string {
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

	var filledColor color.Color
	switch {
	case pct >= 80:
		filledColor = theme.Red
	case pct >= 50:
		filledColor = theme.Yellow
	default:
		filledColor = theme.Mauve
	}

	filledStyle := lipgloss.NewStyle().Foreground(filledColor)
	emptyStyle := lipgloss.NewStyle().Foreground(theme.Surface1)
	pctStyle := lipgloss.NewStyle().Foreground(theme.Overlay0)

	return filledStyle.Render(strings.Repeat("\u2593", filled)) +
		emptyStyle.Render(strings.Repeat("\u2591", empty)) +
		pctStyle.Render(pctLabel)
}

// ---------------------------------------------------------------------------
// Setters
// ---------------------------------------------------------------------------

func (p *infoPanelCmp) SetSize(width, height int) tea.Cmd {
	p.width = width
	p.height = height
	return nil
}

func (p *infoPanelCmp) GetSize() (int, int) {
	return p.width, p.height
}

func (p *infoPanelCmp) SetProvider(name string, clr color.Color) {
	p.provider = name
	p.providerColor = clr
}

func (p *infoPanelCmp) SetThinkingMode(on bool) {
	p.thinkingMode = on
}

func (p *infoPanelCmp) SetTokenInfo(input, output, cacheCreation, cacheRead int, cost float64) {
	p.inputTokens = input
	p.outputTokens = output
	p.cacheCreation = cacheCreation
	p.cacheRead = cacheRead
	p.cost = cost
}

func (p *infoPanelCmp) SetModifiedFiles(files []ModifiedFile) {
	p.modifiedFiles = files
}

func (p *infoPanelCmp) SetYoloMode(on bool) {
	p.yoloMode = on
}

func (p *infoPanelCmp) SetProviders(providers []ProviderStatus) {
	p.providers = providers
}

func (p *infoPanelCmp) SetMCPServers(servers []MCPServer) {
	p.mcpServers = servers
}

func (p *infoPanelCmp) SetSandboxMode(on bool) {
	p.sandboxMode = on
}

func (p *infoPanelCmp) IsVisible() bool {
	return p.visible
}

func (p *infoPanelCmp) Toggle() {
	p.visible = !p.visible
}
