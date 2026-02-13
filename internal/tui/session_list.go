package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/pedromelo/poly/internal/session"
	"github.com/pedromelo/poly/internal/theme"
	"github.com/pedromelo/poly/internal/tools"
)

// renderSessionList renders the session list dialog
func (m Model) renderSessionList() string {
	width := m.width - 8
	if width > 70 {
		width = 70
	}
	if width < 40 {
		width = 40
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(theme.Mauve).
		Bold(true)

	selectedStyle := lipgloss.NewStyle().
		Foreground(theme.Text).
		Bold(true)

	normalStyle := lipgloss.NewStyle().
		Foreground(theme.Overlay1)

	currentStyle := lipgloss.NewStyle().
		Foreground(theme.Green)

	dimStyle := lipgloss.NewStyle().
		Foreground(theme.Overlay0)

	providerStyle := lipgloss.NewStyle().
		Foreground(theme.Peach)

	var content strings.Builder

	content.WriteString(titleStyle.Render("Sessions") + "\n\n")

	sessions := session.ListSessions()
	currentID := session.CurrentID()

	if len(sessions) == 0 {
		content.WriteString(dimStyle.Render("  No sessions yet. Start chatting!"))
	} else {
		for i, s := range sessions {
			// Selection indicator
			prefix := "  "
			if i == m.sessionListIndex {
				prefix = "> "
			}

			// Current session marker
			marker := " "
			if s.ID == currentID {
				marker = currentStyle.Render("*")
			}

			// Title
			title := s.Title
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

			// Provider
			prov := providerStyle.Render("@" + s.Provider)

			// Time ago
			ago := timeAgo(s.UpdatedAt)

			// Messages count
			msgs := dimStyle.Render(fmt.Sprintf("%d msgs", s.MessageCount))

			// Render line
			var line string
			if i == m.sessionListIndex {
				line = selectedStyle.Render(prefix+marker+" "+title) + "  " + prov + "  " + msgs + "  " + dimStyle.Render(ago)
			} else {
				line = normalStyle.Render(prefix+marker+" "+title) + "  " + prov + "  " + msgs + "  " + dimStyle.Render(ago)
			}

			content.WriteString(line + "\n")
		}
	}

	content.WriteString("\n")

	// Footer with keybindings
	keyStyle := lipgloss.NewStyle().Foreground(theme.Subtext0)
	descStyle := lipgloss.NewStyle().Foreground(theme.Overlay0)

	content.WriteString(
		keyStyle.Render("enter") + descStyle.Render(" open  ") +
			keyStyle.Render("n") + descStyle.Render(" new  ") +
			keyStyle.Render("d") + descStyle.Render(" delete  ") +
			keyStyle.Render("f") + descStyle.Render(" fork  ") +
			keyStyle.Render("esc") + descStyle.Render(" close"),
	)

	// Wrap in dialog box
	dialog := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(theme.Mauve).
		Padding(1, 2).
		Width(width).
		Render(content.String())

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)
}

// handleSessionListKey handles key presses in the session list
func (m Model) handleSessionListKey(keyStr string) (Model, bool) {
	sessions := session.ListSessions()

	switch keyStr {
	case "up", "k":
		if m.sessionListIndex > 0 {
			m.sessionListIndex--
		}
		return m, false

	case "down", "j":
		if m.sessionListIndex < len(sessions)-1 {
			m.sessionListIndex++
		}
		return m, false

	case "enter":
		if m.sessionListIndex < len(sessions) {
			s := sessions[m.sessionListIndex]
			session.SwitchSession(s.ID)
			// Reload messages from the switched session
			m.reloadSession()
			m.state = viewChat
			m.status = "Switched to: " + s.Title
		}
		return m, true

	case "n", "N":
		session.Clear()
		tools.ClearModifiedFiles()
		m.reloadSession()
		m.state = viewChat
		m.status = "New session"
		return m, true

	case "d", "D":
		if m.sessionListIndex < len(sessions) {
			s := sessions[m.sessionListIndex]
			if s.ID != session.CurrentID() {
				session.DeleteSession(s.ID)
				if m.sessionListIndex >= len(session.ListSessions()) {
					m.sessionListIndex = len(session.ListSessions()) - 1
				}
				if m.sessionListIndex < 0 {
					m.sessionListIndex = 0
				}
				m.status = "Deleted session"
			} else {
				m.status = "Can't delete current session"
			}
		}
		return m, false

	case "f", "F":
		_, err := session.ForkSession()
		if err == nil {
			m.reloadSession()
			m.state = viewChat
			m.status = "Forked session"
			return m, true
		}
		m.status = "Fork failed"
		return m, false

	case "esc":
		m.state = viewChat
		return m, true
	}

	return m, false
}

// reloadSession reloads messages from the current session
func (m *Model) reloadSession() {
	sess, _ := session.Load()
	if sess != nil {
		m.messages = make([]Message, 0, len(sess.Messages))
		for _, msg := range sess.Messages {
			m.messages = append(m.messages, Message{
				Role:     msg.Role,
				Content:  msg.Content,
				Provider: msg.Provider,
				Thinking: msg.Thinking,
				Images:   msg.Images,
			})
		}
		if sess.Provider != "" {
			m.defaultProvider = sess.Provider
		}
	} else {
		m.messages = nil
	}
	m.updateViewport()
}

// timeAgo returns a human-readable time difference
func timeAgo(t time.Time) string {
	diff := time.Since(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		return fmt.Sprintf("%dm ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		return fmt.Sprintf("%dh ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	default:
		return t.Format("Jan 2")
	}
}
