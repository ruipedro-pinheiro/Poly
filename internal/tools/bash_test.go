package tools

import (
	"strings"
	"testing"
)

func TestTruncateOutput_Short(t *testing.T) {
	input := "short output"
	result := truncateOutput(input)
	if result != input {
		t.Errorf("expected unchanged output, got %q", result)
	}
}

func TestTruncateOutput_ExactLimit(t *testing.T) {
	input := strings.Repeat("a", maxOutput)
	result := truncateOutput(input)
	if result != input {
		t.Error("expected unchanged output at exact limit")
	}
}

func TestTruncateOutput_OverLimit(t *testing.T) {
	// Build a long output with many lines
	var lines []string
	for i := 0; i < 1000; i++ {
		lines = append(lines, strings.Repeat("x", 50))
	}
	input := strings.Join(lines, "\n")

	result := truncateOutput(input)
	if len(result) >= len(input) {
		t.Error("expected truncated output to be shorter")
	}
	if !strings.Contains(result, "truncated") {
		t.Error("expected truncated marker in output")
	}
}

func TestTruncateOutput_Empty(t *testing.T) {
	result := truncateOutput("")
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestBashTool_Name(t *testing.T) {
	tool := &BashTool{}
	if tool.Name() != "bash" {
		t.Errorf("expected Name()=bash, got %q", tool.Name())
	}
}

func TestBashTool_Description(t *testing.T) {
	tool := &BashTool{}
	if tool.Description() == "" {
		t.Error("expected non-empty description")
	}
}

func TestBashTool_Parameters(t *testing.T) {
	tool := &BashTool{}
	params := tool.Parameters()
	if params == nil {
		t.Fatal("expected non-nil parameters")
	}
	if params["type"] != "object" {
		t.Errorf("expected type=object, got %v", params["type"])
	}
	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("expected properties to be a map")
	}
	if _, ok := props["command"]; !ok {
		t.Error("expected 'command' property")
	}
}

func TestBashTool_Execute_EmptyCommand(t *testing.T) {
	tool := &BashTool{}
	result := tool.Execute(map[string]interface{}{})
	if !result.IsError {
		t.Error("expected error for empty command")
	}
	if !strings.Contains(result.Content, "command is required") {
		t.Errorf("expected 'command is required', got %q", result.Content)
	}
}

func TestBashTool_Execute_BlockedCommand(t *testing.T) {
	tool := &BashTool{}
	result := tool.Execute(map[string]interface{}{
		"command": "rm -rf /",
	})
	if !result.IsError {
		t.Error("expected error for blocked command")
	}
	if !strings.Contains(result.Content, "Blocked") {
		t.Errorf("expected 'Blocked', got %q", result.Content)
	}
}

func TestBashTool_Execute_SimpleCommand(t *testing.T) {
	tool := &BashTool{}
	result := tool.Execute(map[string]interface{}{
		"command": "echo hello",
	})
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "hello") {
		t.Errorf("expected 'hello' in output, got %q", result.Content)
	}
}

func TestBashTool_Execute_FailingCommand(t *testing.T) {
	tool := &BashTool{}
	result := tool.Execute(map[string]interface{}{
		"command": "false",
	})
	if !result.IsError {
		t.Error("expected error for failing command")
	}
	if !strings.Contains(result.Content, "Exit code") {
		t.Errorf("expected 'Exit code' in output, got %q", result.Content)
	}
}

func TestBashTool_Execute_Stderr(t *testing.T) {
	tool := &BashTool{}
	result := tool.Execute(map[string]interface{}{
		"command": "echo 'err msg' >&2 && exit 1",
	})
	if !result.IsError {
		t.Error("expected error")
	}
	if !strings.Contains(result.Content, "STDERR") {
		t.Errorf("expected STDERR in output, got %q", result.Content)
	}
}

func TestBashTool_Execute_WithTimeout(t *testing.T) {
	tool := &BashTool{}
	// Very short timeout - should not cause issues with echo
	result := tool.Execute(map[string]interface{}{
		"command": "echo fast",
		"timeout": float64(5000), // 5 seconds
	})
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
}

func TestBashTool_Execute_NoOutput(t *testing.T) {
	tool := &BashTool{}
	result := tool.Execute(map[string]interface{}{
		"command": "true",
	})
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "(no output)") {
		t.Errorf("expected '(no output)', got %q", result.Content)
	}
}
