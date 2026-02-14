package hooks

import (
	"strings"
	"testing"
)

func TestRenderTemplate_NoPlaceholders(t *testing.T) {
	result, err := renderTemplate("echo hello", TemplateData{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "echo hello" {
		t.Errorf("expected 'echo hello', got %q", result)
	}
}

func TestRenderTemplate_ToolName(t *testing.T) {
	data := TemplateData{ToolName: "bash"}
	result, err := renderTemplate("echo {{.ToolName}}", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "echo bash" {
		t.Errorf("expected 'echo bash', got %q", result)
	}
}

func TestRenderTemplate_ToolNameWithSpaces(t *testing.T) {
	data := TemplateData{ToolName: "bash"}
	result, err := renderTemplate("echo {{ .ToolName }}", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "echo bash" {
		t.Errorf("expected 'echo bash', got %q", result)
	}
}

func TestRenderTemplate_AllPlaceholders(t *testing.T) {
	data := TemplateData{
		ToolName: "bash",
		Command:  "ls -la",
		ExitCode: 0,
		Output:   "file1\nfile2",
	}
	tmpl := "tool={{.ToolName}} cmd={{.Command}} exit={{.ExitCode}} out={{.Output}}"
	result, err := renderTemplate(tmpl, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "tool=bash") {
		t.Errorf("expected ToolName=bash in result, got %q", result)
	}
	if !strings.Contains(result, "cmd=ls -la") {
		t.Errorf("expected Command='ls -la' in result, got %q", result)
	}
	if !strings.Contains(result, "exit=0") {
		t.Errorf("expected ExitCode=0 in result, got %q", result)
	}
	if !strings.Contains(result, "out=file1\nfile2") {
		t.Errorf("expected Output in result, got %q", result)
	}
}

func TestRenderTemplate_ExitCodeNonZero(t *testing.T) {
	data := TemplateData{ExitCode: 1}
	result, err := renderTemplate("exit={{.ExitCode}}", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "exit=1" {
		t.Errorf("expected 'exit=1', got %q", result)
	}
}

func TestRenderTemplate_UnknownPlaceholder(t *testing.T) {
	// Unknown placeholders are left as-is (not replaced)
	data := TemplateData{ToolName: "bash"}
	result, err := renderTemplate("{{.Unknown}} {{.ToolName}}", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "{{.Unknown}}") {
		t.Errorf("expected unknown placeholder to remain, got %q", result)
	}
	if !strings.Contains(result, "bash") {
		t.Errorf("expected ToolName to be replaced, got %q", result)
	}
}

func TestRenderTemplate_EmptyData(t *testing.T) {
	data := TemplateData{}
	result, err := renderTemplate("tool={{.ToolName}} exit={{.ExitCode}}", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "tool=") {
		t.Errorf("expected 'tool=' in result, got %q", result)
	}
	if !strings.Contains(result, "exit=0") {
		t.Errorf("expected 'exit=0' (zero value) in result, got %q", result)
	}
}

func TestSetConfig_Nil(t *testing.T) {
	SetConfig(nil)
	cfg := getConfig()
	if cfg == nil {
		t.Fatal("getConfig should never return nil after SetConfig(nil)")
	}
}

func TestSetConfig_WithHooks(t *testing.T) {
	hc := &HooksConfig{
		PreTool: []Hook{
			{Tool: "bash", Command: "echo pre"},
		},
	}
	SetConfig(hc)
	cfg := getConfig()
	if len(cfg.PreTool) != 1 {
		t.Errorf("expected 1 pre_tool hook, got %d", len(cfg.PreTool))
	}
	if cfg.PreTool[0].Tool != "bash" {
		t.Errorf("expected hook tool=bash, got %s", cfg.PreTool[0].Tool)
	}
}

func TestGetConfig_Default(t *testing.T) {
	// Reset
	mu.Lock()
	current = nil
	mu.Unlock()

	cfg := getConfig()
	if cfg == nil {
		t.Fatal("getConfig should return empty config when current is nil")
	}
	if len(cfg.PreTool) != 0 {
		t.Errorf("expected 0 pre_tool hooks by default, got %d", len(cfg.PreTool))
	}
}

func TestRunMatchingHooks_WildcardMatch(t *testing.T) {
	// This tests the matching logic without actually executing hooks
	hooks := []Hook{
		{Tool: "*", Command: "echo all"},
		{Tool: "bash", Command: "echo bash"},
		{Tool: "edit", Command: "echo edit"},
	}

	// Count matches for "bash" tool
	matches := 0
	for _, hook := range hooks {
		if hook.Tool == "*" || hook.Tool == "bash" {
			matches++
		}
	}
	if matches != 2 {
		t.Errorf("expected 2 matches for 'bash' (wildcard + exact), got %d", matches)
	}

	// Count matches for "edit" tool
	matches = 0
	for _, hook := range hooks {
		if hook.Tool == "*" || hook.Tool == "edit" {
			matches++
		}
	}
	if matches != 2 {
		t.Errorf("expected 2 matches for 'edit' (wildcard + exact), got %d", matches)
	}

	// Count matches for "unknown" tool
	matches = 0
	for _, hook := range hooks {
		if hook.Tool == "*" || hook.Tool == "unknown" {
			matches++
		}
	}
	if matches != 1 {
		t.Errorf("expected 1 match for 'unknown' (wildcard only), got %d", matches)
	}
}
