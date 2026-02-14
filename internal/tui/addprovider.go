package tui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/pedromelo/poly/internal/llm"
	"github.com/pedromelo/poly/internal/theme"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	apFieldID     = 0
	apFieldURL    = 1
	apFieldAPIKey = 2
	apFieldModel  = 3
	apFieldFormat = 4
	apFieldCount  = 5
)

var formatOptions = []string{"openai", "anthropic", "google"}

type addProviderForm struct {
	inputs      [4]textinput.Model // ID, URL, API Key, Model
	formatIndex int                // 0=openai, 1=anthropic, 2=google
	focusIndex  int                // which field is focused (0-4)
	width       int
	done        bool
	err         error
}

func newAddProviderForm(width int) addProviderForm {
	f := addProviderForm{
		width: width,
	}

	// ID
	id := textinput.New()
	id.Placeholder = "mistral, ollama, groq..."
	id.Prompt = ""
	id.CharLimit = 32
	id.SetWidth(width - 20)
	id.Validate = func(s string) error { return nil }

	// URL
	url := textinput.New()
	url.Placeholder = "https://api.mistral.ai/v1"
	url.Prompt = ""
	url.CharLimit = 256
	url.SetWidth(width - 20)

	// API Key
	apiKey := textinput.New()
	apiKey.Placeholder = "sk-xxx (empty for local)"
	apiKey.Prompt = ""
	apiKey.EchoMode = textinput.EchoPassword
	apiKey.EchoCharacter = '*'
	apiKey.CharLimit = 256
	apiKey.SetWidth(width - 20)

	// Model
	model := textinput.New()
	model.Placeholder = "mistral-large, llama3..."
	model.Prompt = ""
	model.CharLimit = 128
	model.SetWidth(width - 20)

	f.inputs = [4]textinput.Model{id, url, apiKey, model}
	return f
}

func (f *addProviderForm) Init() tea.Cmd {
	f.focusIndex = 0
	return f.inputs[0].Focus()
}

func (f *addProviderForm) Update(msg tea.Msg) tea.Cmd {
	// Only forward to the focused text input (not the format selector)
	if f.focusIndex < 4 {
		var cmd tea.Cmd
		f.inputs[f.focusIndex], cmd = f.inputs[f.focusIndex].Update(msg)
		return cmd
	}
	return nil
}

func (f *addProviderForm) NextField() tea.Cmd {
	// Blur current
	if f.focusIndex < 4 {
		f.inputs[f.focusIndex].Blur()
	}
	f.focusIndex = (f.focusIndex + 1) % apFieldCount
	// Focus new
	if f.focusIndex < 4 {
		return f.inputs[f.focusIndex].Focus()
	}
	return nil
}

func (f *addProviderForm) PrevField() tea.Cmd {
	// Blur current
	if f.focusIndex < 4 {
		f.inputs[f.focusIndex].Blur()
	}
	f.focusIndex = (f.focusIndex + apFieldCount - 1) % apFieldCount
	// Focus new
	if f.focusIndex < 4 {
		return f.inputs[f.focusIndex].Focus()
	}
	return nil
}

func (f *addProviderForm) CycleFormat(delta int) {
	f.formatIndex = (f.formatIndex + delta + len(formatOptions)) % len(formatOptions)
}

func (f *addProviderForm) Completed() bool {
	return f.done
}

func (f *addProviderForm) Error() error {
	return f.err
}

func (f *addProviderForm) SaveProvider() error {
	id := strings.TrimSpace(f.inputs[apFieldID].Value())
	url := strings.TrimSpace(f.inputs[apFieldURL].Value())
	apiKey := strings.TrimSpace(f.inputs[apFieldAPIKey].Value())
	model := strings.TrimSpace(f.inputs[apFieldModel].Value())

	if id == "" || url == "" || model == "" {
		return nil // nothing to save
	}

	cfg := llm.CustomProviderConfig{
		ID:        id,
		Name:      cases.Title(language.English).String(id),
		BaseURL:   url,
		APIKey:    apiKey,
		Model:     model,
		Format:    formatOptions[f.formatIndex],
		MaxTokens: 4096,
		Color:     "#888888",
	}
	f.err = llm.SaveCustomProvider(cfg)
	f.done = true
	return f.err
}

func (f *addProviderForm) ProviderID() string {
	return strings.TrimSpace(f.inputs[apFieldID].Value())
}

func (f *addProviderForm) View() string {
	labels := []string{"ID", "URL", "API Key", "Model"}

	var content strings.Builder

	for i, label := range labels {
		isFocused := f.focusIndex == i

		labelStyle := lipgloss.NewStyle().Foreground(theme.Overlay1).Width(10)
		if isFocused {
			labelStyle = labelStyle.Foreground(theme.Mauve).Bold(true)
		}
		content.WriteString(labelStyle.Render(label+":") + " ")

		inputStyle := lipgloss.NewStyle().
			Foreground(theme.Text).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(theme.Surface2).
			Padding(0, 1).
			Width(f.width - 16)

		if isFocused {
			inputStyle = inputStyle.BorderForeground(theme.Mauve)
		}

		content.WriteString(inputStyle.Render(f.inputs[i].View()) + "\n")
	}

	// Format selector
	content.WriteString("\n")
	isFormatFocused := f.focusIndex == apFieldFormat
	formatLabel := lipgloss.NewStyle().Foreground(theme.Overlay1).Width(10)
	if isFormatFocused {
		formatLabel = formatLabel.Foreground(theme.Mauve).Bold(true)
	}
	content.WriteString(formatLabel.Render("Format:") + " ")

	displayFormats := []string{"OpenAI", "Anthropic", "Google"}
	for i, name := range displayFormats {
		style := lipgloss.NewStyle().Foreground(theme.Overlay0).Padding(0, 1)
		if i == f.formatIndex {
			style = style.Background(theme.Mauve).Foreground(theme.Base).Bold(true)
		}
		content.WriteString(style.Render(name))
	}
	content.WriteString("\n")

	// Validation hint
	id := strings.TrimSpace(f.inputs[apFieldID].Value())
	url := strings.TrimSpace(f.inputs[apFieldURL].Value())
	model := strings.TrimSpace(f.inputs[apFieldModel].Value())
	if id != "" || url != "" || model != "" {
		missing := []string{}
		if id == "" {
			missing = append(missing, "ID")
		}
		if url == "" {
			missing = append(missing, "URL")
		}
		if model == "" {
			missing = append(missing, "Model")
		}
		if len(missing) > 0 {
			content.WriteString("\n")
			warnStyle := lipgloss.NewStyle().Foreground(theme.Yellow).Italic(true)
			content.WriteString(warnStyle.Render("  Required: " + strings.Join(missing, ", ")))
			content.WriteString("\n")
		}
	}

	// Hints
	content.WriteString("\n")
	hintKey := lipgloss.NewStyle().Foreground(theme.Subtext0)
	hintDesc := lipgloss.NewStyle().Foreground(theme.Overlay0)
	content.WriteString(hintKey.Render("Tab") + hintDesc.Render(" next  "))
	content.WriteString(hintKey.Render("Shift+Tab") + hintDesc.Render(" prev  "))
	if isFormatFocused {
		content.WriteString(hintKey.Render("h/l") + hintDesc.Render(" format  "))
	}
	content.WriteString(hintKey.Render("Enter") + hintDesc.Render(" save  "))
	content.WriteString(hintKey.Render("Esc") + hintDesc.Render(" cancel"))

	return content.String()
}
