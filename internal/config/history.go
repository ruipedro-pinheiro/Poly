package config

import (
	"os"
	"path/filepath"
	"strings"
)

const maxHistoryEntries = 500

// historyPath returns the path to the history file
func historyPath() string {
	return filepath.Join(configDir, "history")
}

// LoadHistory loads input history from ~/.poly/history (one entry per line).
// Returns at most maxHistoryEntries entries.
func LoadHistory() []string {
	data, err := os.ReadFile(historyPath())
	if err != nil {
		return nil
	}
	lines := strings.Split(string(data), "\n")
	var result []string
	for _, line := range lines {
		if line != "" {
			result = append(result, line)
		}
	}
	// Keep only the last maxHistoryEntries
	if len(result) > maxHistoryEntries {
		result = result[len(result)-maxHistoryEntries:]
	}
	return result
}

// AddHistory appends an entry to the history file.
// Skips empty strings and consecutive duplicates (compared to the last line on disk).
func AddHistory(entry string) {
	entry = strings.TrimSpace(entry)
	if entry == "" {
		return
	}

	// Read existing to check for consecutive duplicate
	existing := LoadHistory()
	if len(existing) > 0 && existing[len(existing)-1] == entry {
		return
	}

	// Ensure config dir exists
	os.MkdirAll(configDir, 0700)

	// Append to file
	f, err := os.OpenFile(historyPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(entry + "\n")

	// Truncate if over limit
	if len(existing)+1 > maxHistoryEntries {
		trimmed := append(existing, entry)
		trimmed = trimmed[len(trimmed)-maxHistoryEntries:]
		os.WriteFile(historyPath(), []byte(strings.Join(trimmed, "\n")+"\n"), 0600)
	}
}
