package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- findClosestMatch ---

func TestFindClosestMatch_ExactFirstLine(t *testing.T) {
	content := "line one\nline two\nline three"
	lineNum, preview := findClosestMatch(content, "line two\nline three")
	if lineNum != 2 {
		t.Errorf("expected line 2, got %d", lineNum)
	}
	if !strings.Contains(preview, "line two") {
		t.Errorf("expected preview to contain 'line two', got %q", preview)
	}
}

func TestFindClosestMatch_NoMatch(t *testing.T) {
	content := "aaa\nbbb\nccc"
	lineNum, _ := findClosestMatch(content, "zzz")
	// Should return line 1 (first line, score 0 for all)
	if lineNum < 1 {
		t.Errorf("expected positive line number, got %d", lineNum)
	}
}

func TestFindClosestMatch_PartialMatch(t *testing.T) {
	content := "func TestSomething(t *testing.T) {\n\tt.Log(\"hello\")\n}"
	lineNum, preview := findClosestMatch(content, "func TestSomething(t *testing.T) {")
	if lineNum != 1 {
		t.Errorf("expected line 1, got %d", lineNum)
	}
	if !strings.Contains(preview, "func TestSomething") {
		t.Errorf("expected 'func TestSomething' in preview, got %q", preview)
	}
}

// --- EditFileTool interface ---

func TestEditFileTool_Name(t *testing.T) {
	tool := &EditFileTool{}
	if tool.Name() != "edit_file" {
		t.Errorf("expected Name()=edit_file, got %q", tool.Name())
	}
}

func TestEditFileTool_Description(t *testing.T) {
	tool := &EditFileTool{}
	if tool.Description() == "" {
		t.Error("expected non-empty description")
	}
}

func TestEditFileTool_Parameters(t *testing.T) {
	tool := &EditFileTool{}
	params := tool.Parameters()
	if params == nil {
		t.Fatal("expected non-nil parameters")
	}
	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("expected properties")
	}
	for _, field := range []string{"path", "old_string", "new_string"} {
		if _, ok := props[field]; !ok {
			t.Errorf("expected %q property", field)
		}
	}
}

func TestEditFileTool_Execute_MissingPath(t *testing.T) {
	tool := &EditFileTool{}
	result := tool.Execute(map[string]interface{}{
		"old_string": "foo",
		"new_string": "bar",
	})
	if !result.IsError {
		t.Error("expected error for missing path")
	}
}

func TestEditFileTool_Execute_ExactReplace(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("hello world"), 0600)

	tool := &EditFileTool{}
	result := tool.Execute(map[string]interface{}{
		"path":       testFile,
		"old_string": "hello",
		"new_string": "goodbye",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "exact match") {
		t.Errorf("expected 'exact match' in result, got %q", result.Content)
	}

	data, _ := os.ReadFile(testFile)
	if string(data) != "goodbye world" {
		t.Errorf("expected 'goodbye world', got %q", string(data))
	}
}

func TestEditFileTool_Execute_ReplaceAll(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("foo bar foo baz foo"), 0600)

	tool := &EditFileTool{}
	result := tool.Execute(map[string]interface{}{
		"path":        testFile,
		"old_string":  "foo",
		"new_string":  "qux",
		"replace_all": true,
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}

	data, _ := os.ReadFile(testFile)
	if string(data) != "qux bar qux baz qux" {
		t.Errorf("expected all foo replaced, got %q", string(data))
	}
}

func TestEditFileTool_Execute_MultipleMatchNoReplaceAll(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("foo bar foo"), 0600)

	tool := &EditFileTool{}
	result := tool.Execute(map[string]interface{}{
		"path":       testFile,
		"old_string": "foo",
		"new_string": "baz",
	})
	if !result.IsError {
		t.Error("expected error for multiple matches without replace_all")
	}
	if !strings.Contains(result.Content, "found 2 times") {
		t.Errorf("expected 'found 2 times', got %q", result.Content)
	}
}

func TestEditFileTool_Execute_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("hello world"), 0600)

	tool := &EditFileTool{}
	result := tool.Execute(map[string]interface{}{
		"path":       testFile,
		"old_string": "nonexistent string that does not appear",
		"new_string": "replacement",
	})
	if !result.IsError {
		t.Error("expected error for not found")
	}
	if !strings.Contains(result.Content, "not found") {
		t.Errorf("expected 'not found' in error, got %q", result.Content)
	}
}

func TestEditFileTool_Execute_FuzzyMatch(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	// File has extra spaces
	os.WriteFile(testFile, []byte("hello    world    here"), 0600)

	tool := &EditFileTool{}
	result := tool.Execute(map[string]interface{}{
		"path":       testFile,
		"old_string": "hello world here",
		"new_string": "replaced",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "fuzzy match") {
		t.Errorf("expected 'fuzzy match', got %q", result.Content)
	}
}

func TestEditFileTool_Execute_PathTraversal(t *testing.T) {
	tool := &EditFileTool{}
	result := tool.Execute(map[string]interface{}{
		"path":       "/etc/passwd",
		"old_string": "root",
		"new_string": "hacked",
	})
	if !result.IsError {
		t.Error("expected error for path traversal")
	}
	if !strings.Contains(result.Content, "Access denied") {
		t.Errorf("expected 'Access denied', got %q", result.Content)
	}
}

// --- tryExactMatch ---

func TestTryExactMatch_Found(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	result, matched := tryExactMatch("hello world", "hello", "goodbye", false, testFile, testFile)
	if !matched {
		t.Fatal("expected match")
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
}

func TestTryExactMatch_NotFound(t *testing.T) {
	_, matched := tryExactMatch("hello world", "xyz", "abc", false, "test.txt", "/tmp/test.txt")
	if matched {
		t.Error("expected no match")
	}
}

// --- normalizeWhitespace additional ---

func TestNormalizeWhitespace_OnlySpaces(t *testing.T) {
	got := normalizeWhitespace("   ")
	if got != "" {
		t.Errorf("expected empty after normalizing only spaces, got %q", got)
	}
}

func TestNormalizeWhitespace_TabsAndSpaces(t *testing.T) {
	got := normalizeWhitespace("\t\t  \t")
	if got != "" {
		t.Errorf("expected empty after normalizing tabs+spaces, got %q", got)
	}
}

func TestNormalizeWhitespace_PreservesNewlines(t *testing.T) {
	got := normalizeWhitespace("a\n\nb\n\nc")
	if got != "a\n\nb\n\nc" {
		t.Errorf("expected newlines preserved, got %q", got)
	}
}

// --- commonPrefixLen additional ---

func TestCommonPrefixLen_BothEmpty(t *testing.T) {
	got := commonPrefixLen("", "")
	if got != 0 {
		t.Errorf("expected 0 for both empty, got %d", got)
	}
}

func TestCommonPrefixLen_IdenticalLong(t *testing.T) {
	s := strings.Repeat("x", 100)
	got := commonPrefixLen(s, s)
	if got != 100 {
		t.Errorf("expected 100 for identical strings, got %d", got)
	}
}

// --- editDiffMsg additional ---

func TestEditDiffMsg_EmptyStrings(t *testing.T) {
	got := editDiffMsg("", "")
	if got != "same line count" {
		t.Errorf("expected 'same line count' for empty, got %q", got)
	}
}

func TestEditDiffMsg_SingleNewline(t *testing.T) {
	got := editDiffMsg("", "\n")
	if got != "+1 lines" {
		t.Errorf("expected '+1 lines', got %q", got)
	}
}
