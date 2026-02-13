package addprovider

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/pedromelo/poly/internal/tui/components/dialogs"
	"github.com/pedromelo/poly/internal/tui/core"
	"github.com/pedromelo/poly/internal/tui/styles"
)

// ProviderResult is sent when a provider is saved
type ProviderResult struct {
	ID     string
	URL    string
	APIKey string
	Model  string
	Format string // "openai", "anthropic", "google"
}

type addProviderDialog struct {
	width, height int
	field         int      // 0=id, 1=url, 2=apikey, 3=model, 4=format
	values        []string // [id, url, apikey, model]
	format        int      // 0=openai, 1=anthropic, 2=google
}

// New creates a new add provider dialog
func New() dialogs.DialogModel {
	return &addProviderDialog{
		values: []string{"", "", "", ""},
	}
}

func (a *addProviderDialog) ID() dialogs.DialogID { return dialogs.DialogAddProvider }

func (a *addProviderDialog) Init() tea.Cmd { return nil }

func (a *addProviderDialog) Update(msg tea.Msg) (dialogs.DialogModel, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		keyStr := msg.String()
		switch keyStr {
		case "tab", "down":
			a.field = (a.field + 1) % 5
			return a, nil
		case "shift+tab", "up":
			a.field = (a.field + 4) % 5
			return a, nil
		case "left":
			if a.field == 4 {
				a.format = (a.format + 2) % 3
			}
			return a, nil
		case "right":
			if a.field == 4 {
				a.format = (a.format + 1) % 3
			}
			return a, nil
		case "enter":
			if a.values[0] != "" && a.values[1] != "" && a.values[3] != "" {
				formats := []string{"openai", "anthropic", "google"}
				return a, func() tea.Msg {
					return dialogs.DialogClosedMsg{
						ID: dialogs.DialogAddProvider,
						Result: ProviderResult{
							ID:     a.values[0],
							URL:    a.values[1],
							APIKey: a.values[2],
							Model:  a.values[3],
							Format: formats[a.format],
						},
					}
				}
			}
			return a, func() tea.Msg { return dialogs.CloseDialogMsg{} }
		case "esc":
			return a, func() tea.Msg { return dialogs.CloseDialogMsg{} }
		case "backspace":
			if a.field < 4 && len(a.values[a.field]) > 0 {
				a.values[a.field] = a.values[a.field][:len(a.values[a.field])-1]
			}
			return a, nil
		default:
			if a.field < 4 && len(keyStr) == 1 {
				a.values[a.field] += keyStr
			}
			return a, nil
		}
	}
	return a, nil
}

func (a *addProviderDialog) View() string {
	titleStr := core.Title("+ Add Provider", 38, styles.Mauve, styles.Surface2)

	var content strings.Builder
	content.WriteString(titleStr + "\n\n")

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
		isSelected := i == a.field

		labelStyle := lipgloss.NewStyle().Foreground(styles.Overlay1).Width(10)
		if isSelected {
			labelStyle = labelStyle.Foreground(styles.Mauve).Bold(true)
		}
		content.WriteString(labelStyle.Render(field.label + ":"))

		value := ""
		if i < len(a.values) {
			value = a.values[i]
		}

		inputStyle := lipgloss.NewStyle().
			Background(styles.Surface1).
			Foreground(styles.Text).
			Padding(0, 1).
			Width(28)

		if isSelected {
			inputStyle = inputStyle.Background(styles.Surface2)
		}

		displayValue := value
		if displayValue == "" {
			displayValue = lipgloss.NewStyle().Foreground(styles.Overlay0).Italic(true).Render(field.placeholder)
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
	isFormatSelected := a.field == 4
	formatLabel := lipgloss.NewStyle().Foreground(styles.Overlay1).Width(10)
	if isFormatSelected {
		formatLabel = formatLabel.Foreground(styles.Mauve).Bold(true)
	}
	content.WriteString(formatLabel.Render("Format:"))

	formats := []string{"OpenAI", "Anthropic", "Google"}
	for i, f := range formats {
		style := lipgloss.NewStyle().Foreground(styles.Overlay0).Padding(0, 1)
		if i == a.format {
			style = style.Background(styles.Mauve).Foreground(styles.Base).Bold(true)
		}
		content.WriteString(style.Render(f))
	}
	content.WriteString("\n")

	// Hints
	content.WriteString("\n")
	hintKey := lipgloss.NewStyle().Foreground(styles.Subtext0)
	hintDesc := lipgloss.NewStyle().Foreground(styles.Overlay0)
	content.WriteString(hintKey.Render("Tab") + hintDesc.Render(" next  "))
	content.WriteString(hintKey.Render("<>") + hintDesc.Render(" format  "))
	content.WriteString(hintKey.Render("Enter") + hintDesc.Render(" save  "))
	content.WriteString(hintKey.Render("Esc") + hintDesc.Render(" cancel"))

	dialog := lipgloss.NewStyle().
		Background(styles.Surface0).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(styles.Mauve).
		Padding(1, 2).
		Width(46).
		Render(content.String())

	return lipgloss.Place(
		a.width, a.height,
		lipgloss.Center, lipgloss.Center,
		dialog,
		lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Background(styles.Base)),
	)
}

func (a *addProviderDialog) SetSize(width, height int) {
	a.width = width
	a.height = height
}
