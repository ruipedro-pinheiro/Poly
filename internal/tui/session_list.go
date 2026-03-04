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

// renderSessionList renders the session list dialog with scrolling and filter
func (m Model) renderSessionList() string {
	w := dialogWidth(70, m.width, 40)

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

	// Filter input
	if m.sessionListFilter != "" {
		filterDisplay := m.sessionListFilter
		filterBox := lipgloss.NewStyle().
			Foreground(theme.Text).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(theme.Mauve).
			Padding(0, 1).
			Width(w - 12).
			Render("/ " + filterDisplay)
		content.WriteString(filterBox + "\n\n")
	}

	sessions := m.filteredSessions()
	currentID := session.CurrentID()

	if len(sessions) == 0 {
		if m.sessionListFilter != "" {
			content.WriteString(dimStyle.Render("  No sessions matching \"" + m.sessionListFilter + "\""))
		} else {
			content.WriteString(dimStyle.Render("  No sessions yet. Start chatting!"))
		}
	} else {
		// Compute visible range for scrolling
		maxVisible := m.height - 12
		if maxVisible < 3 {
			maxVisible = 3
		}

		startIdx := 0
		if m.sessionListIndex >= maxVisible {
			startIdx = m.sessionListIndex - maxVisible + 1
		}
		endIdx := startIdx + maxVisible
		if endIdx > len(sessions) {
			endIdx = len(sessions)
		}

		// Scroll indicator top
		if startIdx > 0 {
			content.WriteString(dimStyle.Italic(true).Render(
				fmt.Sprintf("  ... %d more above ...", startIdx)) + "\n")
		}

		for i := startIdx; i < endIdx; i++ {
			s := sessions[i]
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
			maxTitle := w - 30
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

		// Scroll indicator bottom
		if endIdx < len(sessions) {
			content.WriteString(dimStyle.Italic(true).Render(
				fmt.Sprintf("  ... %d more below ...", len(sessions)-endIdx)) + "\n")
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
			keyStyle.Render("/") + descStyle.Render(" filter  ") +
			keyStyle.Render("esc") + descStyle.Render(" close"),
	)

	return m.renderDialogFrame("Sessions", content.String(), 70)
}

// filteredSessions returns sessions matching the current filter
func (m Model) filteredSessions() []session.SessionEntry {
	sessions := session.ListSessions()
	if m.sessionListFilter == "" {
		return sessions
	}

	filter := strings.ToLower(m.sessionListFilter)
	var result []session.SessionEntry
	for _, s := range sessions {
		title := strings.ToLower(s.Title)
		provider := strings.ToLower(s.Provider)
		if strings.Contains(title, filter) || strings.Contains(provider, filter) {
			result = append(result, s)
		}
	}
	return result
}

// handleSessionListKey handles key presses in the session list
func (m Model) handleSessionListKey(keyStr string) (Model, bool) {
	// If filtering, handle text input
	if m.sessionListFiltering {
		switch keyStr {
		case "esc":
			m.sessionListFiltering = false
			m.sessionListFilter = ""
			m.sessionListIndex = 0
			return m, false
		case "enter":
			m.sessionListFiltering = false
			return m, false
		case "backspace":
			if len(m.sessionListFilter) > 0 {
				m.sessionListFilter = m.sessionListFilter[:len(m.sessionListFilter)-1]
				m.sessionListIndex = 0
			}
			return m, false
		default:
			if len(keyStr) == 1 {
				m.sessionListFilter += keyStr
				m.sessionListIndex = 0
			}
			return m, false
		}
	}

	sessions := m.filteredSessions()

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

	case "/":
		m.sessionListFiltering = true
		m.sessionListFilter = ""
		return m, false

	case "enter":
		if m.sessionListIndex < len(sessions) {
			s := sessions[m.sessionListIndex]
			_ = session.SwitchSession(s.ID)
			// Reload messages from the switched session
			m.reloadSession()
			m.state = viewChat
			m.status = "Switched to: " + s.Title
			m.sessionListFilter = ""
		}
		return m, true

	case "n", "N":
		_ = session.Clear()
		tools.ClearModifiedFiles()
		m.reloadSession()
		m.state = viewChat
		m.status = "New session"
		m.sessionListFilter = ""
		return m, true

	case "d", "D":
		if m.sessionListIndex < len(sessions) {
			s := sessions[m.sessionListIndex]
			if s.ID != session.CurrentID() {
				_ = session.DeleteSession(s.ID)
				filtered := m.filteredSessions()
				if m.sessionListIndex >= len(filtered) {
					m.sessionListIndex = len(filtered) - 1
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
			m.sessionListFilter = ""
			return m, true
		}
		m.status = "Fork failed"
		return m, false

	case "esc":
		m.state = viewChat
		m.sessionListFilter = ""
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
