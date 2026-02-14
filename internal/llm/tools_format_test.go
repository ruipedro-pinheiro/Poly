package llm

import "testing"

func TestConvertToolsForProvider_Empty(t *testing.T) {
	result := ConvertToolsForProvider(nil, ToolFormatOpenAI)
	if result != nil {
		t.Errorf("expected nil for empty tools, got %v", result)
	}
}

func TestConvertToolsForProvider_Anthropic(t *testing.T) {
	tools := []ToolDefinition{
		{
			Name:        "read_file",
			Description: "Read a file",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{"type": "string"},
				},
			},
		},
	}

	result := ConvertToolsForProvider(tools, ToolFormatAnthropic)
	if len(result) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(result))
	}

	tool := result[0]
	if tool["name"] != "read_file" {
		t.Errorf("expected name=read_file, got %v", tool["name"])
	}
	if tool["description"] != "Read a file" {
		t.Errorf("expected description='Read a file', got %v", tool["description"])
	}
	if tool["input_schema"] == nil {
		t.Error("expected input_schema to be set")
	}
	// Anthropic should NOT have "type" or "function" wrapper
	if _, ok := tool["type"]; ok {
		t.Error("Anthropic format should not have 'type' field")
	}
}

func TestConvertToolsForProvider_OpenAI(t *testing.T) {
	tools := []ToolDefinition{
		{
			Name:        "bash",
			Description: "Run command",
			InputSchema: map[string]interface{}{"type": "object"},
		},
	}

	result := ConvertToolsForProvider(tools, ToolFormatOpenAI)
	if len(result) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(result))
	}

	tool := result[0]
	if tool["type"] != "function" {
		t.Errorf("expected type=function, got %v", tool["type"])
	}
	fn, ok := tool["function"].(map[string]interface{})
	if !ok {
		t.Fatal("expected function field to be a map")
	}
	if fn["name"] != "bash" {
		t.Errorf("expected function.name=bash, got %v", fn["name"])
	}
}

func TestConvertToolsForProvider_Google(t *testing.T) {
	tools := []ToolDefinition{
		{
			Name:        "search",
			Description: "Search the web",
			InputSchema: map[string]interface{}{"type": "object"},
		},
	}

	result := ConvertToolsForProvider(tools, ToolFormatGoogle)
	if len(result) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(result))
	}

	tool := result[0]
	if tool["name"] != "search" {
		t.Errorf("expected name=search, got %v", tool["name"])
	}
	if tool["parameters"] == nil {
		t.Error("expected parameters to be set for Google format")
	}
}

func TestConvertToolsForProvider_MultipleTools(t *testing.T) {
	tools := []ToolDefinition{
		{Name: "tool1", Description: "d1", InputSchema: map[string]interface{}{}},
		{Name: "tool2", Description: "d2", InputSchema: map[string]interface{}{}},
		{Name: "tool3", Description: "d3", InputSchema: map[string]interface{}{}},
	}

	for _, format := range []ToolFormat{ToolFormatAnthropic, ToolFormatOpenAI, ToolFormatGoogle} {
		result := ConvertToolsForProvider(tools, format)
		if len(result) != 3 {
			t.Errorf("format %s: expected 3 tools, got %d", format, len(result))
		}
	}
}

func TestWrapToolsForGoogle_Empty(t *testing.T) {
	result := WrapToolsForGoogle(nil)
	if result != nil {
		t.Errorf("expected nil for empty tools, got %v", result)
	}
}

func TestWrapToolsForGoogle_WrapsCorrectly(t *testing.T) {
	tools := []map[string]interface{}{
		{"name": "tool1"},
		{"name": "tool2"},
	}

	result := WrapToolsForGoogle(tools)
	if len(result) != 1 {
		t.Fatalf("expected 1 wrapper object, got %d", len(result))
	}

	decls, ok := result[0]["functionDeclarations"]
	if !ok {
		t.Fatal("expected functionDeclarations key")
	}
	declsSlice, ok := decls.([]map[string]interface{})
	if !ok {
		t.Fatal("expected functionDeclarations to be []map[string]interface{}")
	}
	if len(declsSlice) != 2 {
		t.Errorf("expected 2 function declarations, got %d", len(declsSlice))
	}
}

func TestBuildRequestWithTools_Empty(t *testing.T) {
	body := map[string]interface{}{}
	BuildRequestWithTools(body, nil, ToolFormatOpenAI)
	if _, ok := body["tools"]; ok {
		t.Error("expected no tools key for empty tool list")
	}
}

func TestBuildRequestWithTools_Anthropic(t *testing.T) {
	body := map[string]interface{}{}
	tools := []ToolDefinition{
		{Name: "test", Description: "test tool", InputSchema: map[string]interface{}{}},
	}
	BuildRequestWithTools(body, tools, ToolFormatAnthropic)

	if _, ok := body["tools"]; !ok {
		t.Error("expected tools key to be set")
	}
}

func TestBuildRequestWithTools_OpenAI(t *testing.T) {
	body := map[string]interface{}{}
	tools := []ToolDefinition{
		{Name: "test", Description: "test tool", InputSchema: map[string]interface{}{}},
	}
	BuildRequestWithTools(body, tools, ToolFormatOpenAI)

	if _, ok := body["tools"]; !ok {
		t.Error("expected tools key to be set")
	}
}

func TestBuildRequestWithTools_Google(t *testing.T) {
	body := map[string]interface{}{}
	tools := []ToolDefinition{
		{Name: "test", Description: "test tool", InputSchema: map[string]interface{}{}},
	}
	BuildRequestWithTools(body, tools, ToolFormatGoogle)

	toolsVal, ok := body["tools"]
	if !ok {
		t.Fatal("expected tools key to be set")
	}
	toolsSlice, ok := toolsVal.([]map[string]interface{})
	if !ok {
		t.Fatal("expected tools to be []map[string]interface{}")
	}
	// Google wraps in functionDeclarations
	if _, ok := toolsSlice[0]["functionDeclarations"]; !ok {
		t.Error("expected functionDeclarations wrapper for Google format")
	}
}
