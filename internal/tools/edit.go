package tools

import (
	"fmt"
	"math"
	"os"
	"strings"
)

// EditFileTool performs search and replace in files
type EditFileTool struct{}

func (t *EditFileTool) Name() string {
	return "edit_file"
}

func (t *EditFileTool) Description() string {
	return "Edit a file by replacing old_string with new_string. The old_string must be unique in the file. Use replace_all=true to replace all occurrences."
}

func (t *EditFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "File path to edit (relative to cwd)",
			},
			"old_string": map[string]interface{}{
				"type":        "string",
				"description": "The exact string to find and replace",
			},
			"new_string": map[string]interface{}{
				"type":        "string",
				"description": "The string to replace with",
			},
			"replace_all": map[string]interface{}{
				"type":        "boolean",
				"description": "Replace all occurrences (default: false, requires unique match)",
			},
		},
		"required": []string{"path", "old_string", "new_string"},
	}
}

func (t *EditFileTool) Execute(args map[string]interface{}) ToolResult {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return ToolResult{Content: "Error: path is required", IsError: true}
	}

	oldString, ok := args["old_string"].(string)
	if !ok {
		return ToolResult{Content: "Error: old_string is required", IsError: true}
	}

	newString, _ := args["new_string"].(string)

	replaceAll := false
	if ra, ok := args["replace_all"].(bool); ok {
		replaceAll = ra
	}

	// Path validation (resolves symlinks, blocks traversal)
	absPath, err := ValidatePath(path)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("Error: %v", err), IsError: true}
	}

	// Read file
	data, err := os.ReadFile(absPath)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("Error reading file: %v", err), IsError: true}
	}

	content := string(data)

	// Backup before modification
	BackupFile(absPath)

	// === Strategy 1: Exact match ===
	if result, matched := tryExactMatch(content, oldString, newString, replaceAll, path, absPath); matched {
		return result
	}

	// === Strategy 2: Fuzzy match (whitespace-normalized) ===
	if result, matched := tryFuzzyMatch(content, oldString, newString, path, absPath); matched {
		return result
	}

	// === Strategy 3: Line-based match (80%+ similarity) ===
	if result, matched := tryLineMatch(content, oldString, newString, path, absPath); matched {
		return result
	}

	// === All strategies failed: show helpful error ===
	closestLine, closestPreview := findClosestMatch(content, oldString)
	return ToolResult{
		Content: fmt.Sprintf("Error: old_string not found in file. All 3 strategies failed (exact, fuzzy, line-based).\n\nClosest match at line %d:\n%s", closestLine, closestPreview),
		IsError: true,
	}
}

// tryExactMatch performs a direct string match and replace (original behavior).
func tryExactMatch(content, oldString, newString string, replaceAll bool, path, absPath string) (ToolResult, bool) {
	count := strings.Count(content, oldString)
	if count == 0 {
		return ToolResult{}, false
	}

	if count > 1 && !replaceAll {
		return ToolResult{
			Content: fmt.Sprintf("Error: old_string found %d times. Use replace_all=true or make the string more specific.", count),
			IsError: true,
		}, true
	}

	var newContent string
	if replaceAll {
		newContent = strings.ReplaceAll(content, oldString, newString)
	} else {
		newContent = strings.Replace(content, oldString, newString, 1)
	}

	if err := os.WriteFile(absPath, []byte(newContent), 0644); err != nil {
		return ToolResult{Content: fmt.Sprintf("Error writing file: %v", err), IsError: true}, true
	}

	diffMsg := editDiffMsg(oldString, newString)
	TrackModifiedFile(path)
	diff := GenerateUnifiedDiff(content, newContent, path)

	if replaceAll && count > 1 {
		return ToolResult{Content: fmt.Sprintf("[exact match] Replaced %d occurrences in %s (%s)\n\n%s", count, path, diffMsg, diff)}, true
	}
	return ToolResult{Content: fmt.Sprintf("[exact match] Applied edit to %s (%s)\n\n%s", path, diffMsg, diff)}, true
}

// tryFuzzyMatch normalizes whitespace in both old_string and file content,
// then if a match is found, replaces the original (non-normalized) block.
func tryFuzzyMatch(content, oldString, newString, path, absPath string) (ToolResult, bool) {
	normalizedOld := normalizeWhitespace(oldString)
	normalizedContent := normalizeWhitespace(content)

	idx := strings.Index(normalizedContent, normalizedOld)
	if idx == -1 {
		return ToolResult{}, false
	}

	// Map normalized index back to original content position
	origStart, origEnd := mapNormalizedRange(content, idx, idx+len(normalizedOld))
	if origStart == -1 {
		return ToolResult{}, false
	}

	originalBlock := content[origStart:origEnd]
	newContent := content[:origStart] + newString + content[origEnd:]

	if err := os.WriteFile(absPath, []byte(newContent), 0644); err != nil {
		return ToolResult{Content: fmt.Sprintf("Error writing file: %v", err), IsError: true}, true
	}

	diffMsg := editDiffMsg(originalBlock, newString)
	TrackModifiedFile(path)
	diff := GenerateUnifiedDiff(content, newContent, path)

	return ToolResult{Content: fmt.Sprintf("[fuzzy match] Applied edit to %s (whitespace-normalized) (%s)\n\n%s", path, diffMsg, diff)}, true
}

// tryLineMatch splits old_string into lines and finds the best consecutive block
// in the file with >= 80% line similarity.
func tryLineMatch(content, oldString, newString, path, absPath string) (ToolResult, bool) {
	oldLines := strings.Split(oldString, "\n")
	contentLines := strings.Split(content, "\n")

	if len(oldLines) == 0 {
		return ToolResult{}, false
	}

	bestStart := -1
	bestScore := 0.0
	windowSize := len(oldLines)

	for i := 0; i <= len(contentLines)-windowSize; i++ {
		matches := 0
		for j := 0; j < windowSize; j++ {
			if strings.TrimSpace(contentLines[i+j]) == strings.TrimSpace(oldLines[j]) {
				matches++
			}
		}
		score := float64(matches) / float64(windowSize)
		if score > bestScore {
			bestScore = score
			bestStart = i
		}
	}

	if bestScore < 0.9 || bestStart == -1 {
		return ToolResult{}, false
	}

	// Rebuild content replacing the matched block
	var beforeLines, afterLines []string
	if bestStart > 0 {
		beforeLines = contentLines[:bestStart]
	}
	if bestStart+windowSize < len(contentLines) {
		afterLines = contentLines[bestStart+windowSize:]
	}

	var parts []string
	if len(beforeLines) > 0 {
		parts = append(parts, strings.Join(beforeLines, "\n"))
	}
	parts = append(parts, newString)
	if len(afterLines) > 0 {
		parts = append(parts, strings.Join(afterLines, "\n"))
	}
	newContent := strings.Join(parts, "\n")

	if err := os.WriteFile(absPath, []byte(newContent), 0644); err != nil {
		return ToolResult{Content: fmt.Sprintf("Error writing file: %v", err), IsError: true}, true
	}

	matchedBlock := strings.Join(contentLines[bestStart:bestStart+windowSize], "\n")
	diffMsg := editDiffMsg(matchedBlock, newString)
	TrackModifiedFile(path)
	diff := GenerateUnifiedDiff(content, newContent, path)

	pct := int(math.Round(bestScore * 100))
	return ToolResult{Content: fmt.Sprintf("[line match] Applied edit to %s (%d%% line similarity) (%s)\n\n%s", path, pct, diffMsg, diff)}, true
}

// normalizeWhitespace collapses multiple spaces/tabs into single space and trims each line.
func normalizeWhitespace(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		var prev rune
		var normalized strings.Builder
		for _, r := range line {
			if r == ' ' || r == '\t' {
				if prev != ' ' {
					normalized.WriteRune(' ')
				}
				prev = ' '
			} else {
				normalized.WriteRune(r)
				prev = r
			}
		}
		lines[i] = strings.TrimSpace(normalized.String())
	}
	return strings.Join(lines, "\n")
}

// mapNormalizedRange maps a byte range in normalized content back to the original content.
// It walks the original string, tracking how normalization collapses whitespace,
// to find the corresponding byte offsets in the original.
func mapNormalizedRange(original string, normStart, normEnd int) (int, int) {
	origBytes := []byte(original)
	normIdx := 0
	origStart := -1
	inWhitespace := false

	for i := 0; i < len(origBytes) && normIdx < normEnd; i++ {
		b := origBytes[i]

		if b == ' ' || b == '\t' {
			if !inWhitespace {
				if normIdx == normStart && origStart == -1 {
					origStart = i
				}
				normIdx++
				inWhitespace = true
			}
			// Additional whitespace chars: consumed in original, no normalized advance
		} else if b == '\n' {
			if normIdx == normStart && origStart == -1 {
				origStart = i
			}
			normIdx++
			inWhitespace = false
		} else {
			if normIdx == normStart && origStart == -1 {
				origStart = i
			}
			normIdx++
			inWhitespace = false
		}

		if normIdx == normEnd {
			origEnd := i + 1
			// Consume trailing whitespace on the same line
			for origEnd < len(origBytes) && (origBytes[origEnd] == ' ' || origBytes[origEnd] == '\t') {
				origEnd++
			}
			return origStart, origEnd
		}
	}

	if origStart == -1 {
		return -1, -1
	}
	return origStart, len(origBytes)
}

// findClosestMatch finds the line with the best partial match to old_string's first line.
func findClosestMatch(content, oldString string) (int, string) {
	contentLines := strings.Split(content, "\n")
	oldLines := strings.Split(oldString, "\n")
	firstOldLine := strings.TrimSpace(oldLines[0])

	bestLine := 1
	bestScore := 0

	for i, line := range contentLines {
		trimmed := strings.TrimSpace(line)
		score := commonPrefixLen(trimmed, firstOldLine)
		if score > bestScore {
			bestScore = score
			bestLine = i + 1
		}
	}

	// Show up to 3 lines starting from the best match
	startIdx := bestLine - 1
	endIdx := startIdx + 3
	if endIdx > len(contentLines) {
		endIdx = len(contentLines)
	}
	preview := strings.Join(contentLines[startIdx:endIdx], "\n")
	return bestLine, preview
}

// commonPrefixLen returns the length of the common prefix between two strings.
func commonPrefixLen(a, b string) int {
	maxLen := len(a)
	if len(b) < maxLen {
		maxLen = len(b)
	}
	for i := 0; i < maxLen; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return maxLen
}

// editDiffMsg computes a short summary of line count changes.
func editDiffMsg(oldString, newString string) string {
	oldLines := strings.Count(oldString, "\n")
	newLines := strings.Count(newString, "\n")
	lineDiff := newLines - oldLines

	if lineDiff > 0 {
		return fmt.Sprintf("+%d lines", lineDiff)
	} else if lineDiff < 0 {
		return fmt.Sprintf("%d lines", lineDiff)
	}
	return "same line count"
}
