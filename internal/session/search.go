package session

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SearchResult represents a match found in a saved session
type SearchResult struct {
	SessionID    string
	SessionTitle string
	MessageRole  string
	Content      string // excerpt around the match
	Timestamp    time.Time
}

// SearchAll searches through all saved sessions for a query string.
// Returns matching results with context snippets, limited to maxResults.
func SearchAll(query string, maxResults int) ([]SearchResult, error) {
	if maxResults <= 0 {
		maxResults = 20
	}

	query = strings.ToLower(query)

	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		return nil, err
	}

	var results []SearchResult

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") || entry.Name() == "index.json" {
			continue
		}

		id := strings.TrimSuffix(entry.Name(), ".json")
		sess, err := loadSession(id)
		if err != nil {
			continue
		}

		for _, msg := range sess.Messages {
			if len(results) >= maxResults {
				return results, nil
			}

			lower := strings.ToLower(msg.Content)
			idx := strings.Index(lower, query)
			if idx < 0 {
				continue
			}

			excerpt := extractSnippet(msg.Content, idx, len(query), 100)

			title := sess.Title
			if title == "" {
				title = sess.ID
			}

			results = append(results, SearchResult{
				SessionID:    sess.ID,
				SessionTitle: title,
				MessageRole:  msg.Role,
				Content:      excerpt,
				Timestamp:    msg.Timestamp,
			})
		}
	}

	return results, nil
}

// extractSnippet returns a substring of ~totalLen chars centered on the match.
func extractSnippet(content string, matchIdx, matchLen, totalLen int) string {
	// Calculate window around the match
	padding := (totalLen - matchLen) / 2
	start := matchIdx - padding
	end := matchIdx + matchLen + padding

	if start < 0 {
		start = 0
	}
	if end > len(content) {
		end = len(content)
	}

	// Avoid cutting in the middle of a UTF-8 sequence by working with runes
	runes := []rune(content)
	runeStart := 0
	runeEnd := len(runes)

	// Convert byte offsets to approximate rune offsets
	pos := 0
	for i, r := range runes {
		if pos >= start && runeStart == 0 && i > 0 {
			runeStart = i
		}
		pos += len(string(r))
		if pos >= end {
			runeEnd = i + 1
			break
		}
	}

	snippet := string(runes[runeStart:runeEnd])

	// Clean up newlines
	snippet = strings.ReplaceAll(snippet, "\n", " ")

	prefix := ""
	suffix := ""
	if runeStart > 0 {
		prefix = "..."
	}
	if runeEnd < len(runes) {
		suffix = "..."
	}

	return prefix + strings.TrimSpace(snippet) + suffix
}

// CountSessions returns the total number of saved sessions
func CountSessions() int {
	dir := filepath.Join(sessionDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") && e.Name() != "index.json" {
			count++
		}
	}
	return count
}
