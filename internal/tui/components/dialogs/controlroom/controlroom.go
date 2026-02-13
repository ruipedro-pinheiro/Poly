package controlroom

import (
	"image/color"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/pedromelo/poly/internal/tui/components/dialogs"
	"github.com/pedromelo/poly/internal/tui/core"
	"github.com/pedromelo/poly/internal/tui/styles"
)

// ProviderInfo describes a provider for display
type ProviderInfo struct {
	Name        string
	IsConnected bool
	AuthType    string // "OAuth" or "API Key"
	IsDefault   bool
	Color       color.Color
}

// ActionResult is sent when an action is taken
type ActionResult struct {
	Action   string // "connect", "disconnect", "set_default", "add_provider"
	Provider string
}

// AuthInputResult is sent when auth input is provided
type AuthInputResult struct {
	Provider string
	IsOAuth  bool
	Input    string
}

type controlRoomDialog struct {
	width, height int
	index         int
	providers     []ProviderInfo

	// Auth input mode
	oauthPending string
	apiKeyPending string
	authInput     string
}

// New creates a new control room dialog
func New(providers []ProviderInfo) dialogs.DialogModel {
	return &controlRoomDialog{
		providers: providers,
	}
}

func (c *controlRoomDialog) ID() dialogs.DialogID { return dialogs.DialogControlRoom }

func (c *controlRoomDialog) Init() tea.Cmd { return nil }

func (c *controlRoomDialog) Update(msg tea.Msg) (dialogs.DialogModel, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		keyStr := msg.String()

		// Handle pasted text in auth mode
		if msg.Text != "" && len(msg.Text) > 1 && (c.oauthPending != "" || c.apiKeyPending != "") {
			c.authInput = strings.TrimSpace(msg.Text)
			return c, nil
		}

		// Auth input mode
		if c.oauthPending != "" || c.apiKeyPending != "" {
			return c.handleAuthInput(keyStr)
		}

		switch {
		case keyStr == "esc" || keyStr == "ctrl+d":
			return c, func() tea.Msg { return dialogs.CloseDialogMsg{} }

		case keyStr == "up":
			if c.index > 0 {
				c.index--
			}
			return c, nil

		case keyStr == "down":
			if c.index < len(c.providers)-1 {
				c.index++
			}
			return c, nil

		case keyStr == "enter":
			if c.index < len(c.providers) {
				p := c.providers[c.index]
				if p.IsConnected {
					// Set as default
					return c, func() tea.Msg {
						return dialogs.DialogClosedMsg{
							ID:     dialogs.DialogControlRoom,
							Result: ActionResult{Action: "set_default", Provider: p.Name},
						}
					}
				}
				// Connect
				return c, func() tea.Msg {
					return dialogs.DialogClosedMsg{
						ID:     dialogs.DialogControlRoom,
						Result: ActionResult{Action: "connect", Provider: p.Name},
					}
				}
			}
			return c, nil

		case keyStr == "backspace" || keyStr == "delete":
			if c.index < len(c.providers) {
				p := c.providers[c.index]
				if p.IsConnected {
					return c, func() tea.Msg {
						return dialogs.DialogClosedMsg{
							ID:     dialogs.DialogControlRoom,
							Result: ActionResult{Action: "disconnect", Provider: p.Name},
						}
					}
				}
			}
			return c, nil

		case keyStr == "n" || keyStr == "N":
			return c, func() tea.Msg {
				return dialogs.DialogClosedMsg{
					ID:     dialogs.DialogControlRoom,
					Result: ActionResult{Action: "add_provider"},
				}
			}
		}
	}
	return c, nil
}

func (c *controlRoomDialog) handleAuthInput(keyStr string) (dialogs.DialogModel, tea.Cmd) {
	switch {
	case keyStr == "enter":
		provider := c.oauthPending
		isOAuth := true
		if provider == "" {
			provider = c.apiKeyPending
			isOAuth = false
		}
		input := c.authInput
		c.oauthPending = ""
		c.apiKeyPending = ""
		c.authInput = ""
		return c, func() tea.Msg {
			return dialogs.DialogClosedMsg{
				ID: dialogs.DialogControlRoom,
				Result: AuthInputResult{
					Provider: provider,
					IsOAuth:  isOAuth,
					Input:    input,
				},
			}
		}

	case keyStr == "esc":
		c.oauthPending = ""
		c.apiKeyPending = ""
		c.authInput = ""
		return c, nil

	case keyStr == "backspace":
		if len(c.authInput) > 0 {
			c.authInput = c.authInput[:len(c.authInput)-1]
		}
		return c, nil

	case keyStr == "ctrl+v" || keyStr == "ctrl+p":
		// Clipboard handled by parent
		return c, nil

	default:
		if len(keyStr) == 1 {
			c.authInput += keyStr
		}
		return c, nil
	}
}

func (c *controlRoomDialog) View() string {
	titleStr := core.Title("Control Room", 40, styles.Mauve, styles.Surface2)

	var content strings.Builder
	content.WriteString(titleStr + "\n\n")

	for i, p := range c.providers {
		isSelected := i == c.index

		var row strings.Builder

		if isSelected {
			row.WriteString(lipgloss.NewStyle().Foreground(styles.Mauve).Bold(true).Render(" > "))
		} else {
			row.WriteString("   ")
		}

		color := p.Color
		if color == nil {
			color = styles.Overlay1
		}
		iconStyle := lipgloss.NewStyle().Foreground(color)
		nameStyle := lipgloss.NewStyle().Foreground(color).Bold(true)

		row.WriteString(iconStyle.Render(">>") + " ")
		row.WriteString(nameStyle.Width(8).Render(p.Name))

		if p.IsConnected {
			badge := lipgloss.NewStyle().
				Foreground(styles.Base).
				Background(styles.Green).
				Padding(0, 1).
				Render(p.AuthType)
			row.WriteString(" " + badge)
		} else {
			badge := lipgloss.NewStyle().
				Foreground(styles.Overlay0).
				Render("- " + p.AuthType)
			row.WriteString(" " + badge)
		}

		if p.IsDefault {
			row.WriteString(lipgloss.NewStyle().Foreground(styles.Yellow).Render("  *"))
		}

		rowStr := row.String()
		if isSelected {
			rowStr = lipgloss.NewStyle().
				Background(styles.Surface1).
				Width(41).
				Render(rowStr)
		}
		content.WriteString(rowStr + "\n")
	}

	content.WriteString("\n")

	// Auth input area
	if c.oauthPending != "" || c.apiKeyPending != "" {
		var label, placeholder string
		if c.oauthPending != "" {
			label = "Paste OAuth code for " + c.oauthPending
			placeholder = "waiting for code..."
		} else {
			label = "Paste API key for " + c.apiKeyPending
			placeholder = "waiting for key..."
		}

		content.WriteString(lipgloss.NewStyle().Foreground(styles.Yellow).Render(label) + "\n\n")

		inputContent := c.authInput
		if inputContent == "" {
			inputContent = lipgloss.NewStyle().Foreground(styles.Overlay0).Italic(true).Render(placeholder)
		} else {
			if c.apiKeyPending != "" && len(inputContent) > 8 {
				inputContent = inputContent[:4] + strings.Repeat("*", len(inputContent)-8) + inputContent[len(inputContent)-4:]
			}
			if len(inputContent) > 38 {
				inputContent = inputContent[:35] + "..."
			}
		}

		inputBox := lipgloss.NewStyle().
			Foreground(styles.Text).
			Background(styles.Surface1).
			Padding(0, 1).
			Width(40).
			Render(inputContent)

		content.WriteString(inputBox + "\n\n")
		hintKey := lipgloss.NewStyle().Foreground(styles.Subtext0)
		hintDesc := lipgloss.NewStyle().Foreground(styles.Overlay0)
		content.WriteString(hintKey.Render("Enter") + hintDesc.Render(" submit  "))
		content.WriteString(hintKey.Render("Esc") + hintDesc.Render(" cancel"))
	} else {
		hintKey := lipgloss.NewStyle().Foreground(styles.Subtext0)
		hintDesc := lipgloss.NewStyle().Foreground(styles.Overlay0)

		content.WriteString(hintKey.Render("up/dn") + hintDesc.Render(" navigate  "))
		content.WriteString(hintKey.Render("Enter") + hintDesc.Render(" connect  "))
		content.WriteString(hintKey.Render("Del") + hintDesc.Render(" disconnect\n"))
		content.WriteString(hintKey.Render("n") + hintDesc.Render(" add provider  "))
		content.WriteString(hintKey.Render("Esc") + hintDesc.Render(" close"))
	}

	dialog := lipgloss.NewStyle().
		Background(styles.Surface0).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(styles.Surface2).
		Padding(1, 2).
		Width(46).
		Render(content.String())

	return lipgloss.Place(
		c.width, c.height,
		lipgloss.Center, lipgloss.Center,
		dialog,
		lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Background(styles.Base)),
	)
}

// SetOAuthPending sets the OAuth pending state
func (c *controlRoomDialog) SetOAuthPending(provider string) {
	c.oauthPending = provider
	c.authInput = ""
}

// SetAPIKeyPending sets the API key pending state
func (c *controlRoomDialog) SetAPIKeyPending(provider string) {
	c.apiKeyPending = provider
	c.authInput = ""
}

func (c *controlRoomDialog) SetSize(width, height int) {
	c.width = width
	c.height = height
}
