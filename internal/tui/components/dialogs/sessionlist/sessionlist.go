package sessionlist

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/pedromelo/poly/internal/tui/components/dialogs"
	"github.com/pedromelo/poly/internal/tui/core"
	"github.com/pedromelo/poly/internal/tui/styles"
)

// SessionInfo describes a session for display
type SessionInfo struct {
	ID           string
	Title        string
	Provider     string
	MessageCount int
	UpdatedAt    time.Time
	IsCurrent    bool
}

// ActionResult is sent when an action is taken
type ActionResult struct {
	Action    string // "open", "new", "delete", "fork"
	SessionID string
}

type sessionListDialog struct {
	width, height int
	index         int
	sessions      []SessionInfo
}

// New creates a new session list dialog
func New(sessions []SessionInfo) dialogs.DialogModel {
	return &sessionListDialog{
		sessions: sessions,
	}
}

func (s *sessionListDialog) ID() dialogs.DialogID { return dialogs.DialogSessionList }

func (s *sessionListDialog) Init() tea.Cmd { return nil }

func (s *sessionListDialog) Update(msg tea.Msg) (dialogs.DialogModel, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		keyStr := msg.String()
		switch keyStr {
		case "esc":
			return s, func() tea.Msg { return dialogs.CloseDialogMsg{} }

		case "up", "k":
			if s.index > 0 {
				s.index--
			}
			return s, nil

		case "down", "j":
			if s.index < len(s.sessions)-1 {
				s.index++
			}
			return s, nil

		case "enter":
			if s.index < len(s.sessions) {
				sess := s.sessions[s.index]
				return s, func() tea.Msg {
					return dialogs.DialogClosedMsg{
						ID:     dialogs.DialogSessionList,
						Result: ActionResult{Action: "open", SessionID: sess.ID},
					}
				}
			}
			return s, nil

		case "n", "N":
			return s, func() tea.Msg {
				return dialogs.DialogClosedMsg{
					ID:     dialogs.DialogSessionList,
					Result: ActionResult{Action: "new"},
				}
			}

		case "d", "D":
			if s.index < len(s.sessions) {
				sess := s.sessions[s.index]
				if !sess.IsCurrent {
					return s, func() tea.Msg {
						return dialogs.DialogClosedMsg{
							ID:     dialogs.DialogSessionList,
							Result: ActionResult{Action: "delete", SessionID: sess.ID},
						}
					}
				}
			}
			return s, nil

		case "f", "F":
			return s, func() tea.Msg {
				return dialogs.DialogClosedMsg{
					ID:     dialogs.DialogSessionList,
					Result: ActionResult{Action: "fork"},
				}
			}
		}
	}
	return s, nil
}

func (s *sessionListDialog) View() string {
	width := s.width - 8
	if width > 70 {
		width = 70
	}
	if width < 40 {
		width = 40
	}

	titleStr := core.Title("Sessions", width-8, styles.Mauve, styles.Surface2)

	selectedStyle := lipgloss.NewStyle().
		Foreground(styles.Text).
		Bold(true)

	normalStyle := lipgloss.NewStyle().
		Foreground(styles.Overlay1)

	currentStyle := lipgloss.NewStyle().
		Foreground(styles.Green)

	dimStyle := lipgloss.NewStyle().
		Foreground(styles.Overlay0)

	providerStyle := lipgloss.NewStyle().
		Foreground(styles.Peach)

	var content strings.Builder
	content.WriteString(titleStr + "\n\n")

	if len(s.sessions) == 0 {
		content.WriteString(dimStyle.Render("  No sessions yet. Start chatting!"))
	} else {
		for i, sess := range s.sessions {
			prefix := "  "
			if i == s.index {
				prefix = "> "
			}

			marker := " "
			if sess.IsCurrent {
				marker = currentStyle.Render("*")
			}

			title := sess.Title
			if title == "" {
				title = "Untitled"
			}
			maxTitle := width - 30
			if maxTitle < 10 {
				maxTitle = 10
			}
			if len(title) > maxTitle {
				title = title[:maxTitle-3] + "..."
			}

			prov := providerStyle.Render("@" + sess.Provider)
			ago := timeAgo(sess.UpdatedAt)
			msgs := dimStyle.Render(fmt.Sprintf("%d msgs", sess.MessageCount))

			var line string
			if i == s.index {
				line = selectedStyle.Render(prefix+marker+" "+title) + "  " + prov + "  " + msgs + "  " + dimStyle.Render(ago)
			} else {
				line = normalStyle.Render(prefix+marker+" "+title) + "  " + prov + "  " + msgs + "  " + dimStyle.Render(ago)
			}

			content.WriteString(line + "\n")
		}
	}

	content.WriteString("\n")

	keyStyle := lipgloss.NewStyle().Foreground(styles.Subtext0)
	descStyle := lipgloss.NewStyle().Foreground(styles.Overlay0)

	content.WriteString(
		keyStyle.Render("enter") + descStyle.Render(" open  ") +
			keyStyle.Render("n") + descStyle.Render(" new  ") +
			keyStyle.Render("d") + descStyle.Render(" delete  ") +
			keyStyle.Render("f") + descStyle.Render(" fork  ") +
			keyStyle.Render("esc") + descStyle.Render(" close"),
	)

	dialog := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(styles.Mauve).
		Padding(1, 2).
		Width(width).
		Render(content.String())

	return lipgloss.Place(
		s.width, s.height,
		lipgloss.Center, lipgloss.Center,
		dialog,
	)
}

func timeAgo(t time.Time) string {
	diff := time.Since(t)
	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	case diff < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
	default:
		return t.Format("Jan 2")
	}
}

func (s *sessionListDialog) SetSize(width, height int) {
	s.width = width
	s.height = height
}
