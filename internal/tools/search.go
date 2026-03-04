package tools

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const maxSearchResults = 100

// GlobTool searches for files matching a glob pattern
type GlobTool struct{}

func (t *GlobTool) Name() string {
	return "glob"
}

func (t *GlobTool) Description() string {
	return "Search for files matching a glob pattern (e.g., '**/*.go', 'src/**/*.ts'). Returns matching file paths."
}

func (t *GlobTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "Glob pattern to match (e.g., '**/*.go', 'src/*.ts')",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Directory to search in (default: current directory)",
			},
		},
		"required": []string{"pattern"},
	}
}

func (t *GlobTool) Execute(args map[string]interface{}) ToolResult {
	pattern, ok := args["pattern"].(string)
	if !ok || pattern == "" {
		return ToolResult{Content: "Error: pattern is required", IsError: true}
	}

	basePath := "."
	if p, ok := args["path"].(string); ok && p != "" {
		basePath = p
	}

	// Path validation (resolves symlinks, blocks traversal)
	absPath, err := ValidatePath(basePath)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("Error: %v", err), IsError: true}
	}

	var matches []string

	// Handle ** patterns manually
	if strings.Contains(pattern, "**") {
		// Walk directory and match
		parts := strings.Split(pattern, "**")
		prefix := strings.TrimSuffix(parts[0], "/")
		suffix := ""
		if len(parts) > 1 {
			suffix = strings.TrimPrefix(parts[1], "/")
		}

		_ = WalkDir(absPath, 10, func(path string, d os.DirEntry) error {
			if d.IsDir() {
				return nil
			}
			if len(matches) >= maxSearchResults {
				return filepath.SkipAll
			}

			rel, _ := filepath.Rel(absPath, path)

			// Match prefix
			if prefix != "" && !strings.HasPrefix(rel, prefix) {
				return nil
			}

			// Match suffix (extension pattern)
			if suffix != "" {
				matched, _ := filepath.Match(suffix, filepath.Base(rel))
				if !matched {
					return nil
				}
			}

			matches = append(matches, rel)
			return nil
		})
	} else {
		// Simple glob
		fullPattern := filepath.Join(absPath, pattern)
		files, err := filepath.Glob(fullPattern)
		if err != nil {
			return ToolResult{Content: fmt.Sprintf("Error: %v", err), IsError: true}
		}
		for _, f := range files {
			rel, _ := filepath.Rel(absPath, f)
			matches = append(matches, rel)
			if len(matches) >= maxSearchResults {
				break
			}
		}
	}

	if len(matches) == 0 {
		return ToolResult{Content: "No files found"}
	}

	sort.Strings(matches)
	result := strings.Join(matches, "\n")
	if len(matches) >= maxSearchResults {
		result += fmt.Sprintf("\n... (truncated at %d results)", maxSearchResults)
	}

	return ToolResult{Content: result}
}

// GrepTool searches for content in files
type GrepTool struct{}

func (t *GrepTool) Name() string {
	return "grep"
}

func (t *GrepTool) Description() string {
	return "Search for a pattern in files. Returns matching lines with file paths and line numbers."
}

func (t *GrepTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "Regex pattern to search for",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "File or directory to search in (default: current directory)",
			},
			"glob": map[string]interface{}{
				"type":        "string",
				"description": "File pattern to filter (e.g., '*.go', '*.ts')",
			},
			"case_insensitive": map[string]interface{}{
				"type":        "boolean",
				"description": "Case insensitive search (default: false)",
			},
		},
		"required": []string{"pattern"},
	}
}

func (t *GrepTool) Execute(args map[string]interface{}) ToolResult {
	pattern, ok := args["pattern"].(string)
	if !ok || pattern == "" {
		return ToolResult{Content: "Error: pattern is required", IsError: true}
	}

	basePath := "."
	if p, ok := args["path"].(string); ok && p != "" {
		basePath = p
	}

	fileGlob := ""
	if g, ok := args["glob"].(string); ok {
		fileGlob = g
	}

	caseInsensitive := false
	if ci, ok := args["case_insensitive"].(bool); ok {
		caseInsensitive = ci
	}

	// Compile regex
	if caseInsensitive {
		pattern = "(?i)" + pattern
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("Error: invalid regex: %v", err), IsError: true}
	}

	// Path validation (resolves symlinks, blocks traversal)
	absPath, err := ValidatePath(basePath)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("Error: %v", err), IsError: true}
	}

	cwd, _ := os.Getwd()
	var results []string
	matchCount := 0

	searchFile := func(path string) error {
		if matchCount >= maxSearchResults {
			return nil
		}

		// Check file glob
		if fileGlob != "" {
			matched, _ := filepath.Match(fileGlob, filepath.Base(path))
			if !matched {
				return nil
			}
		}

		// Skip binary files
		ext := strings.ToLower(filepath.Ext(path))
		if binaryExtensions[ext] || imageExtensions[ext] {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		rel, _ := filepath.Rel(cwd, path)
		scanner := bufio.NewScanner(file)
		lineNum := 0

		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			if re.MatchString(line) {
				matchCount++
				if matchCount > maxSearchResults {
					return nil
				}

				// Truncate long lines
				if len(line) > 200 {
					line = line[:200] + "..."
				}
				results = append(results, fmt.Sprintf("%s:%d: %s", rel, lineNum, line))
			}
		}
		return nil
	}

	// Check if path is a file or directory
	info, err := os.Stat(absPath)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("Error: %v", err), IsError: true}
	}

	if info.IsDir() {
		_ = WalkDir(absPath, 10, func(path string, d os.DirEntry) error {
			if d.IsDir() {
				return nil
			}
			return searchFile(path)
		})
	} else {
		_ = searchFile(absPath)
	}

	if len(results) == 0 {
		return ToolResult{Content: "No matches found"}
	}

	result := strings.Join(results, "\n")
	if matchCount > maxSearchResults {
		result += fmt.Sprintf("\n... (truncated at %d matches)", maxSearchResults)
	}

	return ToolResult{Content: result}
}
