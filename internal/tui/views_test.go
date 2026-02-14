package tui

import (
	"strings"
	"testing"

	"github.com/pedromelo/poly/internal/tui/components/messages/tools"
)

// --- formatTokenCount tests ---

func TestFormatTokenCount_SmallNumbers(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{42, "42"},
		{100, "100"},
		{999, "999"},
	}

	for _, tt := range tests {
		got := formatTokenCount(tt.input)
		if got != tt.expected {
			t.Errorf("formatTokenCount(%d) = %s, want %s", tt.input, got, tt.expected)
		}
	}
}

func TestFormatTokenCount_ThousandPlus(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{1000, "1.0k"},
		{1234, "1.2k"},
		{1500, "1.5k"},
		{10000, "10.0k"},
		{99999, "100.0k"},
		{1000000, "1000.0k"},
	}

	for _, tt := range tests {
		got := formatTokenCount(tt.input)
		if got != tt.expected {
			t.Errorf("formatTokenCount(%d) = %s, want %s", tt.input, got, tt.expected)
		}
	}
}

// --- toolCallStatus tests ---

func TestToolCallStatus(t *testing.T) {
	tests := []struct {
		input    int
		expected tools.ToolStatus
	}{
		{0, tools.ToolStatusPending},
		{1, tools.ToolStatusRunning},
		{2, tools.ToolStatusSuccess},
		{3, tools.ToolStatusError},
		{-1, tools.ToolStatusPending},  // default
		{99, tools.ToolStatusPending},  // default
	}

	for _, tt := range tests {
		got := toolCallStatus(tt.input)
		if got != tt.expected {
			t.Errorf("toolCallStatus(%d) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

// --- overlayRight tests ---

func TestOverlayRight_EmptyPanel(t *testing.T) {
	base := "hello\nworld"
	result := overlayRight(base, "", 80)
	if result != base {
		t.Errorf("expected base unchanged for empty panel, got '%s'", result)
	}
}

func TestOverlayRight_PanelShorterThanBase(t *testing.T) {
	base := "line1\nline2\nline3"
	panel := "P1"
	result := overlayRight(base, panel, 20)

	lines := strings.Split(result, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	// First line should have panel appended
	if !strings.Contains(lines[0], "P1") {
		t.Error("expected first line to contain panel text")
	}
	// Other lines should be unchanged
	if lines[2] != "line3" {
		t.Errorf("expected 'line3' on third line, got '%s'", lines[2])
	}
}

func TestOverlayRight_PanelAndBasePreserved(t *testing.T) {
	base := "aaa\nbbb"
	panel := "XX\nYY"
	result := overlayRight(base, panel, 10)

	lines := strings.Split(result, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "XX") {
		t.Errorf("expected first line to contain 'XX', got '%s'", lines[0])
	}
	if !strings.Contains(lines[1], "YY") {
		t.Errorf("expected second line to contain 'YY', got '%s'", lines[1])
	}
}

// --- addMessage tests ---

func TestAddMessage(t *testing.T) {
	m := testModel()
	msg := Message{
		Role:    "user",
		Content: "test message",
	}
	m.addMessage(msg)

	if len(m.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(m.messages))
	}
	if m.messages[0].Content != "test message" {
		t.Errorf("expected 'test message', got '%s'", m.messages[0].Content)
	}
}

// --- saveMessageAt bounds check ---

func TestSaveMessageAt_OutOfBounds(t *testing.T) {
	m := testModel()
	m.messages = []Message{{Role: "user", Content: "hi"}}

	// Should not panic
	m.saveMessageAt(-1)
	m.saveMessageAt(5)
}

// --- chatWidth / contentWidth ---

func TestChatWidth(t *testing.T) {
	m := testModel()
	m.layout = LayoutContext{ChatWidth: 120, ContentWidth: 100}

	if m.chatWidth() != 120 {
		t.Errorf("expected chatWidth 120, got %d", m.chatWidth())
	}
	if m.contentWidth() != 100 {
		t.Errorf("expected contentWidth 100, got %d", m.contentWidth())
	}
}
