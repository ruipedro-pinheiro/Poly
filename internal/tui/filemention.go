package tui

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/pedromelo/poly/internal/config"
)

// maxFileSize is the maximum file size to include (50KB)
const maxFileSize = 50 * 1024

// fileMentionRegex matches @path patterns that contain a file extension
// Must have at least one dot in the path to distinguish from @provider mentions
var fileMentionRegex = regexp.MustCompile(`@([\w./\-]+\.\w+)`)

// isKnownProvider checks dynamically if a name is a configured provider
func isKnownProvider(name string) bool {
	if name == "all" {
		return true
	}
	for _, p := range config.GetProviderNames() {
		if p == name {
			return true
		}
	}
	return false
}

// ParseFileMentions scans user input for @path patterns and reads the files.
// Returns the enriched message content with file contents appended,
// and the list of files that were successfully included.
func ParseFileMentions(input string) (enrichedContent string, files []string) {
	matches := fileMentionRegex.FindAllStringSubmatch(input, -1)
	if len(matches) == 0 {
		return input, nil
	}

	seen := make(map[string]bool)
	var fileBlocks []string

	for _, match := range matches {
		path := match[1]

		// Skip known provider names (e.g. @all.go would still match, but @claude won't)
		if isKnownProvider(path) {
			continue
		}

		// Deduplicate
		if seen[path] {
			continue
		}
		seen[path] = true

		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		content := string(data)

		// Truncate if too large
		if len(data) > maxFileSize {
			content = content[:maxFileSize] + fmt.Sprintf("\n... [truncated, file is %d bytes]", len(data))
		}

		fileBlocks = append(fileBlocks, fmt.Sprintf("<file path=%q>\n%s\n</file>", path, content))
		files = append(files, path)
	}

	if len(fileBlocks) == 0 {
		return input, nil
	}

	enrichedContent = input + "\n\n" + strings.Join(fileBlocks, "\n\n")
	return enrichedContent, files
}
