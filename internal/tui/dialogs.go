package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/pedromelo/poly/internal/auth"
	"github.com/pedromelo/poly/internal/config"
	"github.com/pedromelo/poly/internal/theme"
	"github.com/pedromelo/poly/internal/tui/core"
	splashPkg "github.com/pedromelo/poly/internal/tui/components/splash"
	"github.com/pedromelo/poly/internal/tui/styles"
)

// renderDialogFrame wraps content in a styled dialog box centered on screen.
// It uses the dialog width from m.layout.DialogWidth, applies the theme border,
// adds a title at the top, and handles vertical overflow with a scroll indicator.
func (m Model) renderDialogFrame(title string, content string, preferredWidth int) string {
	w := dialogWidth(preferredWidth, m.width, 40)
	innerW := w - 6 // padding (2*2) + border (2*1)

	// Build title header
	header := core.Title(title, innerW, styles.Mauve, styles.Surface2)

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
			Foreground(theme.Overlay0).
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
		BorderForeground(theme.Mauve).
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
			default:
				authLabel = "Custom"
			}
		}

		var row strings.Builder

		// Selection cursor
		if isSelected {
			row.WriteString(lipgloss.NewStyle().Foreground(theme.Mauve).Bold(true).Render(" > "))
		} else {
			row.WriteString("   ")
		}

		// Icon + Name
		provColor := theme.ProviderColor(providerName)
		iconStyle := lipgloss.NewStyle().Foreground(provColor)
		nameStyle := lipgloss.NewStyle().
			Foreground(provColor).
			Bold(true)
		row.WriteString(iconStyle.Render(">>") + " ")
		row.WriteString(nameStyle.Width(8).Render(providerName))

		// Status badge
		if isConnected {
			connAuthInfo := storage.GetAuth(providerName)
			connLabel := authLabel
			if connAuthInfo != nil && connAuthInfo.Type == "oauth" {
				connLabel = "OAuth"
			}
			badge := lipgloss.NewStyle().
				Foreground(theme.Base).
				Background(theme.Green).
				Padding(0, 1).
				Render(connLabel)
			row.WriteString(" " + badge)
		} else {
			badge := lipgloss.NewStyle().
				Foreground(theme.Overlay0).
				Render("- " + authLabel)
			row.WriteString(" " + badge)
		}

		// Default star
		if isDefault {
			row.WriteString(lipgloss.NewStyle().Foreground(theme.Yellow).Render("  *"))
		}

		// Row style: selected gets a subtle left accent
		rowStr := row.String()
		if isSelected {
			rowStr = lipgloss.NewStyle().
				BorderStyle(lipgloss.ThickBorder()).
				BorderLeft(true).
				BorderRight(false).
				BorderTop(false).
				BorderBottom(false).
				BorderForeground(theme.Mauve).
				Width(innerWidth).
				Render(rowStr)
		}
		content.WriteString(rowStr + "\n")
	}

	content.WriteString("\n")

	// Auth input area
	if m.oauthPending != "" || m.apiKeyPending != "" {
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
			BorderForeground(theme.Surface2).
			Padding(0, 1).
			Width(innerWidth - 2).
			Render(inputContent)

		content.WriteString(inputBox + "\n\n")
		hintKey := lipgloss.NewStyle().Foreground(theme.Subtext0)
		hintDesc := lipgloss.NewStyle().Foreground(theme.Overlay0)
		content.WriteString(hintKey.Render("Enter") + hintDesc.Render(" submit · "))
		content.WriteString(hintKey.Render("Esc") + hintDesc.Render(" cancel"))
	} else {
		hintKey := lipgloss.NewStyle().Foreground(theme.Subtext0)
		hintDesc := lipgloss.NewStyle().Foreground(theme.Overlay0)

		content.WriteString(hintKey.Render("↑↓") + hintDesc.Render(" navigate · "))
		content.WriteString(hintKey.Render("Enter") + hintDesc.Render(" connect · "))
		content.WriteString(hintKey.Render("Del") + hintDesc.Render(" disconnect\n"))
		content.WriteString(hintKey.Render("n") + hintDesc.Render(" add provider · "))
		content.WriteString(hintKey.Render("Esc") + hintDesc.Render(" close"))
	}

	return m.renderDialogFrame("Control Room", content.String(), 52)
}

func (m Model) renderAddProvider() string {
	w := dialogWidth(46, m.width, 36)

	var content strings.Builder

	fields := []struct {
		label       string
		placeholder string
	}{
		{"ID", "mistral, ollama, groq..."},
		{"URL", "https://api.mistral.ai/v1"},
		{"API Key", "sk-xxx (empty for local)"},
		{"Model", "mistral-large, llama3..."},
	}

	for i, field := range fields {
		isSelected := i == m.addProviderField

		labelStyle := lipgloss.NewStyle().Foreground(theme.Overlay1).Width(10)
		if isSelected {
			labelStyle = labelStyle.Foreground(theme.Mauve).Bold(true)
		}
		content.WriteString(labelStyle.Render(field.label + ":"))

		value := ""
		if i < len(m.addProviderValues) {
			value = m.addProviderValues[i]
		}

		inputStyle := lipgloss.NewStyle().
			Foreground(theme.Text).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(theme.Surface2).
			Padding(0, 1).
			Width(w - 16)

		if isSelected {
			inputStyle = inputStyle.BorderForeground(theme.Mauve)
		}

		displayValue := value
		if displayValue == "" {
			displayValue = lipgloss.NewStyle().Foreground(theme.Overlay0).Italic(true).Render(field.placeholder)
		} else if i == 2 && len(displayValue) > 4 {
			displayValue = displayValue[:2] + strings.Repeat("*", len(displayValue)-4) + displayValue[len(displayValue)-2:]
		}

		if isSelected {
			displayValue += "_"
		}

		content.WriteString(inputStyle.Render(displayValue) + "\n")
	}

	// Format selector
	content.WriteString("\n")
	isFormatSelected := m.addProviderField == 4
	formatLabel := lipgloss.NewStyle().Foreground(theme.Overlay1).Width(10)
	if isFormatSelected {
		formatLabel = formatLabel.Foreground(theme.Mauve).Bold(true)
	}
	content.WriteString(formatLabel.Render("Format:"))

	formats := []string{"OpenAI", "Anthropic", "Google"}
	for i, f := range formats {
		style := lipgloss.NewStyle().Foreground(theme.Overlay0).Padding(0, 1)
		if i == m.addProviderFormat {
			style = style.Background(theme.Mauve).Foreground(theme.Base).Bold(true)
		}
		content.WriteString(style.Render(f))
	}
	content.WriteString("\n")

	// Hints
	content.WriteString("\n")
	hintKey := lipgloss.NewStyle().Foreground(theme.Subtext0)
	hintDesc := lipgloss.NewStyle().Foreground(theme.Overlay0)
	content.WriteString(hintKey.Render("Tab") + hintDesc.Render(" next · "))
	content.WriteString(hintKey.Render("◁▷") + hintDesc.Render(" format · "))
	content.WriteString(hintKey.Render("Enter") + hintDesc.Render(" save · "))
	content.WriteString(hintKey.Render("Esc") + hintDesc.Render(" cancel"))

	return m.renderDialogFrame("+ Add Provider", content.String(), 46)
}
