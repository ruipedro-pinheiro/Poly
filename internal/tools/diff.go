package tools

import (
	"fmt"
	"strings"
)

// GenerateUnifiedDiff generates a unified diff between old and new content
func GenerateUnifiedDiff(oldContent, newContent, filename string) string {
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	var diff strings.Builder
	diff.WriteString(fmt.Sprintf("--- a/%s\n", filename))
	diff.WriteString(fmt.Sprintf("+++ b/%s\n", filename))

	// Simple line-by-line diff (not full Myers algorithm, but good enough for display)
	chunks := computeDiffChunks(oldLines, newLines)
	for _, chunk := range chunks {
		diff.WriteString(chunk)
	}

	return diff.String()
}

// computeDiffChunks generates diff hunks
func computeDiffChunks(oldLines, newLines []string) []string {
	var chunks []string

	i, j := 0, 0
	for i < len(oldLines) || j < len(newLines) {
		// Find matching lines (context)
		if i < len(oldLines) && j < len(newLines) && oldLines[i] == newLines[j] {
			i++
			j++
			continue
		}

		// Found a difference - collect the hunk
		hunkStart := i
		hunkStartNew := j

		// Look ahead for resync point
		syncOld, syncNew := findSync(oldLines, newLines, i, j)

		var hunk strings.Builder
		hunk.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n",
			hunkStart+1, syncOld-hunkStart,
			hunkStartNew+1, syncNew-hunkStartNew))

		// Add context before (up to 3 lines)
		contextStart := hunkStart - 3
		if contextStart < 0 {
			contextStart = 0
		}
		for k := contextStart; k < hunkStart; k++ {
			hunk.WriteString(" " + oldLines[k] + "\n")
		}

		// Removed lines
		for k := hunkStart; k < syncOld; k++ {
			hunk.WriteString("-" + oldLines[k] + "\n")
		}

		// Added lines
		for k := hunkStartNew; k < syncNew; k++ {
			hunk.WriteString("+" + newLines[k] + "\n")
		}

		chunks = append(chunks, hunk.String())
		i = syncOld
		j = syncNew
	}

	return chunks
}

// findSync finds the next sync point where old and new lines match again
func findSync(oldLines, newLines []string, startOld, startNew int) (int, int) {
	// Look ahead up to 50 lines for a match
	maxLook := 50

	for look := 1; look <= maxLook; look++ {
		// Try advancing old
		if startOld+look < len(oldLines) && startNew < len(newLines) {
			if oldLines[startOld+look] == newLines[startNew] {
				return startOld + look, startNew
			}
		}
		// Try advancing new
		if startOld < len(oldLines) && startNew+look < len(newLines) {
			if oldLines[startOld] == newLines[startNew+look] {
				return startOld, startNew + look
			}
		}
		// Try advancing both
		if startOld+look < len(oldLines) && startNew+look < len(newLines) {
			if oldLines[startOld+look] == newLines[startNew+look] {
				return startOld + look, startNew + look
			}
		}
	}

	// No sync found - consume all remaining
	return len(oldLines), len(newLines)
}
