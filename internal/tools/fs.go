package tools

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxFileSize     = 256 * 1024 // 256KB
	defaultLineLimit = 2000
	maxLineLength    = 2000
)

var imageExtensions = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	".bmp": true, ".svg": true, ".webp": true, ".ico": true, ".tiff": true,
}

var binaryExtensions = map[string]bool{
	".wasm": true, ".exe": true, ".dll": true, ".so": true, ".dylib": true,
	".zip": true, ".tar": true, ".gz": true, ".7z": true, ".rar": true,
	".pdf": true, ".doc": true, ".xls": true, ".ppt": true,
}

// ReadFileTool reads file contents with line numbers
type ReadFileTool struct{}

func (t *ReadFileTool) Name() string {
	return "read_file"
}

func (t *ReadFileTool) Description() string {
	return "Read file contents with line numbers. Supports offset/limit for large files. Detects binary and image files."
}

func (t *ReadFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "File path to read (relative to cwd)",
			},
			"offset": map[string]interface{}{
				"type":        "number",
				"description": "Line number to start reading from (1-based). Default: 1",
			},
			"limit": map[string]interface{}{
				"type":        "number",
				"description": "Number of lines to read. Default: 2000",
			},
		},
		"required": []string{"path"},
	}
}

func (t *ReadFileTool) Execute(args map[string]interface{}) ToolResult {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return ToolResult{Content: "Error: path is required", IsError: true}
	}

	// Resolve path
	cwd, _ := os.Getwd()
	targetPath := filepath.Join(cwd, path)
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return ToolResult{Content: "Error: invalid path", IsError: true}
	}

	// Security check
	if !strings.HasPrefix(absPath, cwd) {
		return ToolResult{Content: "Error: access denied. Cannot read outside project root.", IsError: true}
	}

	// Check if exists and is not a directory
	info, err := os.Stat(absPath)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("Error: %v", err), IsError: true}
	}
	if info.IsDir() {
		return ToolResult{Content: fmt.Sprintf("Error: %s is a directory. Use list_files instead.", path), IsError: true}
	}

	// Check extension
	ext := strings.ToLower(filepath.Ext(absPath))
	if imageExtensions[ext] {
		return ToolResult{Content: fmt.Sprintf("Image file detected (%s). Size: %d bytes", ext, info.Size())}
	}
	if binaryExtensions[ext] {
		return ToolResult{Content: fmt.Sprintf("Binary file detected (%s). Cannot display content.", ext), IsError: true}
	}

	// Size check
	if info.Size() > maxFileSize {
		return ToolResult{
			Content: fmt.Sprintf("File too large (%dKB > %dKB limit). Use offset/limit to read a portion.",
				info.Size()/1024, maxFileSize/1024),
			IsError: true,
		}
	}

	// Read file
	data, err := os.ReadFile(absPath)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("Error: %v", err), IsError: true}
	}

	// Check for binary content (null bytes)
	if containsNullBytes(data[:min(len(data), 8192)]) {
		return ToolResult{Content: "Binary file detected. Cannot display content.", IsError: true}
	}

	// Parse offset and limit
	offset := 1
	if o, ok := args["offset"].(float64); ok {
		offset = int(o)
	}
	limit := defaultLineLimit
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	// Split into lines
	allLines := strings.Split(string(data), "\n")
	totalLines := len(allLines)

	// Apply offset (1-based) and limit
	startIdx := max(0, offset-1)
	endIdx := min(totalLines, startIdx+limit)
	lines := allLines[startIdx:endIdx]

	// Format with line numbers
	maxLineNo := endIdx
	pad := len(fmt.Sprintf("%d", maxLineNo))
	var formatted strings.Builder
	for i, line := range lines {
		lineNo := startIdx + i + 1
		if len(line) > maxLineLength {
			line = line[:maxLineLength] + "..."
		}
		formatted.WriteString(fmt.Sprintf("%*d\u2502%s\n", pad, lineNo, line))
	}

	// Info header
	var info_ []string
	if startIdx > 0 {
		info_ = append(info_, fmt.Sprintf("(starting from line %d)", startIdx+1))
	}
	if endIdx < totalLines {
		info_ = append(info_, fmt.Sprintf("(%d more lines below, total: %d)", totalLines-endIdx, totalLines))
	}

	result := formatted.String()
	if len(info_) > 0 {
		result = strings.Join(info_, " ") + "\n" + result
	}

	return ToolResult{Content: result}
}

// ListFilesTool lists files in a directory
type ListFilesTool struct{}

func (t *ListFilesTool) Name() string {
	return "list_files"
}

func (t *ListFilesTool) Description() string {
	return "List files and directories in a given path. Returns [DIR] or [FILE] prefixes with file sizes."
}

func (t *ListFilesTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Directory path to list (relative to cwd). Defaults to '.'",
			},
			"recursive": map[string]interface{}{
				"type":        "boolean",
				"description": "List recursively (max 3 levels deep). Default: false",
			},
		},
	}
}

func (t *ListFilesTool) Execute(args map[string]interface{}) ToolResult {
	path := "."
	if p, ok := args["path"].(string); ok && p != "" {
		path = p
	}
	recursive := false
	if r, ok := args["recursive"].(bool); ok {
		recursive = r
	}

	cwd, _ := os.Getwd()
	targetPath := filepath.Join(cwd, path)
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return ToolResult{Content: "Error: invalid path", IsError: true}
	}

	// Security check
	if !strings.HasPrefix(absPath, cwd) {
		return ToolResult{Content: "Error: access denied. Cannot list outside project root.", IsError: true}
	}

	var results []string
	maxEntries := 500

	var listDir func(dir string, depth int) error
	listDir = func(dir string, depth int) error {
		if depth > 3 {
			return nil
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			if len(results) >= maxEntries {
				return nil
			}

			// Skip hidden files in subdirectories
			if strings.HasPrefix(entry.Name(), ".") && depth > 0 {
				continue
			}

			fullPath := filepath.Join(dir, entry.Name())
			relPath, _ := filepath.Rel(absPath, fullPath)

			if entry.IsDir() {
				results = append(results, fmt.Sprintf("[DIR]  %s/", relPath))
				if recursive {
					listDir(fullPath, depth+1)
				}
			} else {
				info, err := entry.Info()
				sizeStr := ""
				if err == nil {
					sizeStr = formatSize(info.Size())
				}
				results = append(results, fmt.Sprintf("[FILE] %s (%s)", relPath, sizeStr))
			}
		}
		return nil
	}

	if err := listDir(absPath, 0); err != nil {
		return ToolResult{Content: fmt.Sprintf("Error: %v", err), IsError: true}
	}

	if len(results) == 0 {
		return ToolResult{Content: "(empty directory)"}
	}

	output := strings.Join(results, "\n")
	if len(results) >= maxEntries {
		output += "\n... (truncated at 500 entries)"
	}

	return ToolResult{Content: output}
}

func containsNullBytes(data []byte) bool {
	for _, b := range data {
		if b == 0 {
			return true
		}
	}
	return false
}

func formatSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%dB", size)
	}
	if size < 1024*1024 {
		return fmt.Sprintf("%dKB", size/1024)
	}
	return fmt.Sprintf("%.1fMB", float64(size)/(1024*1024))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// WalkDir helper that respects .gitignore patterns (simplified)
func WalkDir(root string, maxDepth int, fn func(path string, d fs.DirEntry) error) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip common ignored directories
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == "__pycache__" ||
				name == ".venv" || name == "vendor" || name == "dist" || name == "build" {
				return filepath.SkipDir
			}
		}

		// Check depth
		rel, _ := filepath.Rel(root, path)
		depth := strings.Count(rel, string(filepath.Separator))
		if depth > maxDepth {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		return fn(path, d)
	})
}
