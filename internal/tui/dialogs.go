package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/pedromelo/poly/internal/auth"
	"github.com/pedromelo/poly/internal/config"
	"github.com/pedromelo/poly/internal/theme"
	splashPkg "github.com/pedromelo/poly/internal/tui/components/splash"
	"github.com/pedromelo/poly/internal/tui/core"
)

// renderDialogFrame wraps content in a styled dialog box centered on screen.
// It uses the dialog width from m.layout.DialogWidth, applies the theme border,
// adds a title at the top, and handles vertical overflow with a scroll indicator.
func (m Model) renderDialogFrame(title string, content string, preferredWidth int) string {
	w := dialogWidth(preferredWidth, m.width, 40)
	innerW := w - 6 // padding (2*2) + border (2*1)

	// Build title header
	header := core.Title(title, innerW, theme.Mauve, theme.Surface2)

	// Combine header + content
	body := header + "\n\n" + content

	// Compute available height for dialog content (screen - 6 for border/padding/margins)
	maxContentLines := m.height - 6
	if maxContentLines < 5 {
		maxContentLines = 5
	}

	// Truncate if content exceeds available height
	lines := strings.Split(body, "\n")
	if len(lines) > maxContentLines {
		lines = lines[:maxContentLines-1]
		scrollHint := lipgloss.NewStyle().
			Foreground(theme.Overlay1).
			Italic(true).
			Render("  ... more below ...")
		lines = append(lines, scrollHint)
		body = strings.Join(lines, "\n")
	}

	dialog := dialogStyle(w).Render(body)
	return placeDialog(dialog, m.width, m.height)
}

// dialogStyle returns the standard dialog container style
func dialogStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(theme.Surface2).
		Background(theme.Mantle).
		Padding(1, 2).
		Width(width)
}

// dialogWidth computes a responsive dialog width: preferred, clamped to terminal
func dialogWidth(preferred, termWidth, minWidth int) int {
	w := preferred
	if termWidth-6 < w {
		w = termWidth - 6
	}
	if w < minWidth {
		w = minWidth
	}
	return w
}

// placeDialog centers a dialog string on the full screen
func placeDialog(content string, width, height int) string {
	return lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		content,
	)
}

func (m Model) renderSplash() string {
	storage := auth.GetStorage()
	names := config.GetProviderNames()
	var statuses []splashPkg.ProviderStatus
	for _, name := range names {
		statuses = append(statuses, splashPkg.ProviderStatus{
			Name:      name,
			Connected: storage.IsConnected(name),
			Color:     theme.ProviderColor(name),
		})
	}
	m.splashCmp.SetProviders(statuses)
	m.splashCmp.SetSize(m.width, m.height)
	return m.splashCmp.View()
}

func (m Model) renderHelp() string {
	w := dialogWidth(62, m.width, 44)
	innerW := w - 6

	sectionStyle := lipgloss.NewStyle().
		Foreground(theme.Mauve).
		Bold(true)

	keyStyle := lipgloss.NewStyle().
		Foreground(theme.Lavender).
		Bold(true)

	descStyle := lipgloss.NewStyle().
		Foreground(theme.Subtext1)

	cmdStyle := lipgloss.NewStyle().
		Foreground(theme.Lavender).
		Bold(true)

	aliasStyle := lipgloss.NewStyle().
		Foreground(theme.Overlay1)

	dimStyle := lipgloss.NewStyle().
		Foreground(theme.Overlay0)

	var content strings.Builder

	// Keybindings section
	content.WriteString(sectionStyle.Render("  KEYBINDINGS") + "\n")
	keybindings := []struct{ key, desc string }{
		{"Ctrl+H", "This help"},
		{"Ctrl+D", "Control Room"},
		{"Ctrl+O", "Model Picker"},
		{"Ctrl+K", "Command Palette"},
		{"Ctrl+I", "Info Panel"},
		{"Ctrl+S", "Session List"},
		{"Ctrl+L", "Clear chat"},
		{"Ctrl+N", "New session"},
		{"Ctrl+T", "Toggle thinking"},
		{"Tab", "Focus viewport"},
		{"Shift+Tab", "Focus input"},
		{"Enter", "Send message"},
		{"Esc", "Cancel / Close"},
	}
	for _, k := range keybindings {
		content.WriteString("  " + keyStyle.Width(12).Render(k.key) + descStyle.Render(k.desc) + "\n")
	}
	content.WriteString("\n")

	// Mentions section
	content.WriteString(sectionStyle.Render("  PROVIDERS") + "\n")
	// Dynamic provider mentions from config
	names := config.GetProviderNames()
	for _, name := range names {
		content.WriteString("  " + keyStyle.Width(12).Render("@"+name) + descStyle.Render(name) + "\n")
	}
	content.WriteString("  " + keyStyle.Width(12).Render("@all") + descStyle.Render("All providers (cascade)") + "\n")
	content.WriteString("\n")

	// Commands by category from registry
	content.WriteString(sectionStyle.Render("  COMMANDS") + "\n")
	content.WriteString(dimStyle.Render("  Use /help <cmd> for details") + "\n\n")

	catOrder, catMap := m.commands.ByCategory()
	for _, cat := range catOrder {
		cmds := catMap[cat]
		content.WriteString("  " + lipgloss.NewStyle().Foreground(theme.Mauve).Render(cat) + "\n")
		for _, cmd := range cmds {
			name := "/" + cmd.Name
			alias := ""
			if len(cmd.Aliases) > 0 {
				parts := make([]string, len(cmd.Aliases))
				for i, a := range cmd.Aliases {
					parts[i] = "/" + a
				}
				alias = " " + aliasStyle.Render("("+strings.Join(parts, ", ")+")")
			}
			line := "    " + cmdStyle.Width(16).Render(name) + descStyle.Render(cmd.Description) + alias
			content.WriteString(line + "\n")
		}
	}

	content.WriteString("\n")
	content.WriteString(dimStyle.Render("  Esc to close"))

	_ = innerW
	return m.renderDialogFrame("Help", content.String(), 62)
}

func (m Model) renderControlRoom() string {
	storage := auth.GetStorage()
	cfg := config.Get()

	w := dialogWidth(52, m.width, 40)
	innerWidth := w - 6

	var content strings.Builder
	content.WriteString(
		lipgloss.NewStyle().
			Foreground(theme.Overlay0).
			Render("  provider      auth        default") + "\n",
	)
	content.WriteString(
		lipgloss.NewStyle().
			Foreground(theme.Surface1).
			Render("  "+strings.Repeat("─", max(6, innerWidth-2))) + "\n",
	)

	for i, providerName := range m.controlRoomProviders {
		isSelected := i == m.controlRoomIndex
		isConnected := storage.IsConnected(providerName)
		isDefault := m.defaultProvider == providerName

		// Get auth type label from config
		authLabel := "Custom"
		if p, ok := cfg.Providers[providerName]; ok {
			switch p.AuthType {
			case "oauth":
				authLabel = "OAuth"
			case "api_key":
				authLabel = "API"
			case "device_flow":
				authLabel = "GitHub"
			default:
				authLabel = "Custom"
			}
		}

		var row strings.Builder
		cursor := listCursor(isSelected)
		row.WriteString(cursor)

		// Name
		provColor := theme.ProviderColor(providerName)
		nameStyle := lipgloss.NewStyle().
			Foreground(provColor).
			Bold(true)
		row.WriteString(nameStyle.Width(12).Render(truncateToWidth(providerName, 12)))

		// Status badge
		statusColWidth := 10
		if isConnected {
			connAuthInfo := storage.GetAuth(providerName)
			connLabel := authLabel
			if connAuthInfo != nil && connAuthInfo.Type == "oauth" {
				connLabel = "OAuth"
			}
			badge := lipgloss.NewStyle().
				Foreground(theme.Green).
				Bold(true).
				Width(statusColWidth).
				Render(connLabel)
			row.WriteString(" " + badge)
		} else {
			statusText := lipgloss.NewStyle().
				Foreground(theme.Overlay0).
				Width(statusColWidth).
				Render("- " + authLabel)
			row.WriteString(" " + statusText)
		}

		// Default star
		if isDefault {
			row.WriteString(lipgloss.NewStyle().Foreground(theme.Yellow).Render("   *"))
		}

		content.WriteString(renderListRow(innerWidth, row.String(), isSelected) + "\n")
	}

	content.WriteString("\n")

	// Auth input area
	if m.oauthPending != "" || m.apiKeyPending != "" {
		// Check if this is a device flow (no user input needed)
		isDeviceFlow := false
		if m.oauthPending != "" {
			cfg := config.Get()
			if provCfg, ok := cfg.Providers[m.oauthPending]; ok {
				isDeviceFlow = provCfg.AuthType == "device_flow"
			}
		}

		if isDeviceFlow {
			// Device flow: show code and wait, no input box
			label := "GitHub Device Flow"
			content.WriteString(lipgloss.NewStyle().Foreground(theme.Yellow).Render(label) + "\n")

			if m.authStatusMsg != "" {
				statusColor := theme.Mauve
				if strings.HasPrefix(m.authStatusMsg, "Error:") {
					statusColor = theme.Red
				}
				content.WriteString(lipgloss.NewStyle().Foreground(statusColor).Bold(true).Render(m.authStatusMsg) + "\n")
			} else {
				content.WriteString(lipgloss.NewStyle().Foreground(theme.Overlay1).Italic(true).Render("Starting...") + "\n")
			}

			content.WriteString("\n")
			content.WriteString(renderDialogHints(innerWidth, "Esc cancel"))
		} else {
			// Standard OAuth code paste or API key input
			var label, placeholder string
			if m.oauthPending != "" {
				label = "Paste OAuth code for " + m.oauthPending
				placeholder = "waiting for code..."
			} else {
				label = "Paste API key for " + m.apiKeyPending
				placeholder = "waiting for key..."
			}

			content.WriteString(lipgloss.NewStyle().Foreground(theme.Yellow).Render(label) + "\n")

			if m.authStatusMsg != "" {
				statusColor := theme.Green
				if strings.HasPrefix(m.authStatusMsg, "Error:") {
					statusColor = theme.Red
				} else if m.authStatusMsg == "Exchanging code..." {
					statusColor = theme.Overlay1
				}
				content.WriteString(lipgloss.NewStyle().Foreground(statusColor).Render(m.authStatusMsg) + "\n")
			}
			content.WriteString("\n")

			inputContent := m.authInput
			if inputContent == "" {
				inputContent = lipgloss.NewStyle().Foreground(theme.Overlay0).Italic(true).Render(placeholder)
			} else {
				if m.apiKeyPending != "" {
					if len(inputContent) > 8 {
						inputContent = inputContent[:4] + strings.Repeat("*", len(inputContent)-8) + inputContent[len(inputContent)-4:]
					}
				}
				if len(inputContent) > 38 {
					inputContent = inputContent[:35] + "..."
				}
			}

			inputBox := lipgloss.NewStyle().
				Foreground(theme.Text).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(theme.Mauve).
				Background(theme.Surface0).
				Padding(0, 1).
				Width(innerWidth - 2).
				Render(inputContent)

			content.WriteString(inputBox + "\n\n")
			content.WriteString(renderDialogHints(innerWidth, "Enter submit · Esc cancel"))
		}
	} else {
		line1 := "↑↓ navigate · Enter connect · Del disconnect"
		line2 := "n add provider · Esc close"
		content.WriteString(renderDialogHints(innerWidth, line1, line2))
	}

	return m.renderDialogFrame("Control Room", content.String(), 52)
}

func (m Model) renderAddProvider() string {
	if m.addProviderForm == nil {
		return m.renderDialogFrame("+ Add Provider", "Loading...", 46)
	}
	return m.renderDialogFrame("+ Add Provider", m.addProviderForm.View(), 46)
}
