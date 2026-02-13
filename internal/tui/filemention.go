package tui

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// maxFileSize is the maximum file size to include (50KB)
const maxFileSize = 50 * 1024

// fileMentionRegex matches @path patterns that contain a file extension
// Must have at least one dot in the path to distinguish from @provider mentions
var fileMentionRegex = regexp.MustCompile(`@([\w./\-]+\.\w+)`)

// knownProviders lists provider names that should NOT be treated as file mentions
var knownProviders = map[string]bool{
	"claude": true,
	"gpt":    true,
	"gemini": true,
	"grok":   true,
	"all":    true,
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
		if knownProviders[path] {
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
