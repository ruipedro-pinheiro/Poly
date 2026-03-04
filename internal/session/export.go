package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// GetCurrentSession returns the current session (loads if needed)
func GetCurrentSession() *Session {
	if currentSession == nil {
		_, _ = Load()
	}
	return currentSession
}

// ExportMarkdown exports the current session as a readable markdown file.
// Returns the path of the exported file.
func ExportMarkdown() (string, error) {
	sess := GetCurrentSession()
	if sess == nil || len(sess.Messages) == 0 {
		return "", fmt.Errorf("no messages to export")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	filename := fmt.Sprintf("poly-export-%s.md", sess.ID)
	path := filepath.Join(home, filename)

	var b strings.Builder

	// Header
	title := sess.Title
	if title == "" {
		title = "Untitled"
	}
	b.WriteString(fmt.Sprintf("# Poly Session: %s\n", title))

	model := sess.Model
	if model == "" {
		model = "default"
	}
	provider := sess.Provider
	if provider == "" {
		provider = "unknown"
	}
	exportDate := time.Now().Format("2006-01-02 15:04:05")
	b.WriteString(fmt.Sprintf("> Exported on %s | Provider: %s | Model: %s\n\n", exportDate, provider, model))
	b.WriteString("---\n\n")

	// Messages
	for _, msg := range sess.Messages {
		ts := msg.Timestamp.Format("15:04:05")

		switch msg.Role {
		case "user":
			b.WriteString(fmt.Sprintf("### User (%s)\n", ts))
			b.WriteString(msg.Content)
			b.WriteString("\n\n")
		case "assistant":
			prov := msg.Provider
			if prov == "" {
				prov = provider
			}
			b.WriteString(fmt.Sprintf("### Assistant [%s] (%s)\n", prov, ts))
			if msg.Thinking != "" {
				b.WriteString("<details>\n<summary>Thinking</summary>\n\n")
				b.WriteString(msg.Thinking)
				b.WriteString("\n\n</details>\n\n")
			}
			b.WriteString(msg.Content)
			b.WriteString("\n\n")
		default:
			// Tool results or other roles
			b.WriteString(fmt.Sprintf("### %s (%s)\n", msg.Role, ts))
			b.WriteString(msg.Content)
			b.WriteString("\n\n")
		}

		b.WriteString("---\n\n")
	}

	if err := os.WriteFile(path, []byte(b.String()), 0600); err != nil {
		return "", err
	}

	return path, nil
}

// ExportJSON exports the current session as raw JSON.
// Returns the path of the exported file.
func ExportJSON() (string, error) {
	sess := GetCurrentSession()
	if sess == nil || len(sess.Messages) == 0 {
		return "", fmt.Errorf("no messages to export")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	filename := fmt.Sprintf("poly-export-%s.json", sess.ID)
	path := filepath.Join(home, filename)

	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return "", err
	}

	return path, nil
}
