package tui

import (
	"testing"

	"github.com/pedromelo/poly/internal/config"
	"github.com/pedromelo/poly/internal/llm"
)

// --- filterMessagesForProvider tests ---

func TestFilterMessagesForProvider_EmptySlice(t *testing.T) {
	result := filterMessagesForProvider(nil, "claude")
	if len(result) != 0 {
		t.Errorf("expected 0 messages, got %d", len(result))
	}
}

func TestFilterMessagesForProvider_CopiesContent(t *testing.T) {
	msgs := []llm.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi back"},
	}
	result := filterMessagesForProvider(msgs, "gpt")

	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}
	if result[0].Role != "user" || result[0].Content != "hello" {
		t.Errorf("unexpected first message: %+v", result[0])
	}
	if result[1].Role != "assistant" || result[1].Content != "hi back" {
		t.Errorf("unexpected second message: %+v", result[1])
	}
}

func TestFilterMessagesForProvider_StripsImagesForNonSupportingProvider(t *testing.T) {
	imgs := []llm.Image{{Data: []byte("img"), MediaType: "image/png"}}
	msgs := []llm.Message{
		{Role: "user", Content: "look at this", Images: imgs},
	}

	// grok typically doesn't support images (depends on SupportsImages)
	// The function uses llm.SupportsImages which checks the registry
	result := filterMessagesForProvider(msgs, "grok")

	// For providers that don't support images, Images should be nil
	if llm.SupportsImages("grok") {
		// If grok does support images in this config, skip this check
		t.Skip("grok supports images in this config, skipping")
	}
	if result[0].Images != nil {
		t.Error("expected nil images for non-supporting provider")
	}
}

// --- buildLLMMessages tests ---

func TestBuildLLMMessages_Empty(t *testing.T) {
	m := testModel()
	result := m.buildLLMMessages()

	if len(result) != 0 {
		t.Errorf("expected 0 messages, got %d", len(result))
	}
}

func TestBuildLLMMessages_SkipsEmptyAssistant(t *testing.T) {
	m := testModel()
	m.messages = []Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: ""}, // empty slot — should be skipped
		{Role: "assistant", Content: "actual response"},
	}

	result := m.buildLLMMessages()
	if len(result) != 2 {
		t.Fatalf("expected 2 messages (skip empty assistant), got %d", len(result))
	}
	if result[0].Content != "hello" {
		t.Errorf("expected first message 'hello', got '%s'", result[0].Content)
	}
	if result[1].Content != "actual response" {
		t.Errorf("expected second message 'actual response', got '%s'", result[1].Content)
	}
}

func TestBuildLLMMessages_SkipsEmptyContent(t *testing.T) {
	m := testModel()
	m.messages = []Message{
		{Role: "user", Content: "   "}, // whitespace only
		{Role: "user", Content: "real question"},
	}

	result := m.buildLLMMessages()
	if len(result) != 1 {
		t.Fatalf("expected 1 message (skip whitespace), got %d", len(result))
	}
	if result[0].Content != "real question" {
		t.Errorf("expected 'real question', got '%s'", result[0].Content)
	}
}

func TestBuildLLMMessages_SystemMappedToUser(t *testing.T) {
	m := testModel()
	m.messages = []Message{
		{Role: "system", Content: "info msg"},
		{Role: "user", Content: "hello"},
	}

	result := m.buildLLMMessages()
	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}
	// system messages get mapped to "user" role (not "system")
	if result[0].Role != "user" {
		t.Errorf("expected system message mapped to 'user' role, got '%s'", result[0].Role)
	}
}

func TestBuildLLMMessages_AssistantRolePreserved(t *testing.T) {
	m := testModel()
	m.messages = []Message{
		{Role: "user", Content: "hi"},
		{Role: "assistant", Content: "hello"},
	}

	result := m.buildLLMMessages()
	if len(result) != 2 {
		t.Fatalf("expected 2, got %d", len(result))
	}
	if result[1].Role != "assistant" {
		t.Errorf("expected assistant role preserved, got '%s'", result[1].Role)
	}
}

// --- convertImages tests ---

func TestConvertImages_EmptyPaths(t *testing.T) {
	result := convertImages(nil)
	if result != nil {
		t.Errorf("expected nil for empty paths, got %v", result)
	}

	result = convertImages([]string{})
	if result != nil {
		t.Errorf("expected nil for empty slice, got %v", result)
	}
}

func TestConvertImages_NonexistentFile(t *testing.T) {
	result := convertImages([]string{"/nonexistent/file.png"})
	if len(result) != 0 {
		t.Errorf("expected 0 images for nonexistent file, got %d", len(result))
	}
}

func TestConvertImages_DetectsMediaType(t *testing.T) {
	// We can't test with real files easily, but we can test the extension mapping
	// by using /dev/null (empty file)
	result := convertImages([]string{"/dev/null"})
	if len(result) != 1 {
		t.Fatalf("expected 1 image from /dev/null, got %d", len(result))
	}
	// /dev/null has no extension, defaults to image/png
	if result[0].MediaType != "image/png" {
		t.Errorf("expected 'image/png' for no extension, got '%s'", result[0].MediaType)
	}
}

// --- convertMessageImages tests ---

func TestConvertMessageImages_Combined(t *testing.T) {
	msg := Message{
		ImageData:  [][]byte{[]byte("rawdata")},
		ImageTypes: []string{"image/jpeg"},
	}
	result := convertMessageImages(msg)
	if len(result) != 1 {
		t.Fatalf("expected 1 image, got %d", len(result))
	}
	if result[0].MediaType != "image/jpeg" {
		t.Errorf("expected image/jpeg, got '%s'", result[0].MediaType)
	}
}

func TestConvertMessageImages_DefaultMediaType(t *testing.T) {
	msg := Message{
		ImageData: [][]byte{[]byte("rawdata")},
		// No ImageTypes — should default to image/png
	}
	result := convertMessageImages(msg)
	if len(result) != 1 {
		t.Fatalf("expected 1 image, got %d", len(result))
	}
	if result[0].MediaType != "image/png" {
		t.Errorf("expected image/png default, got '%s'", result[0].MediaType)
	}
}

// --- Stream channel helpers tests ---

func TestStreamChanHelpers(t *testing.T) {
	// Start clean
	clearStreamChans()

	ch := make(chan llm.StreamEvent)
	setStreamChan("test-provider", ch)

	got, ok := getStreamChan("test-provider")
	if !ok {
		t.Fatal("expected to find stream chan")
	}
	if got != ch {
		t.Error("expected same channel back")
	}

	_, ok = getStreamChan("nonexistent")
	if ok {
		t.Error("expected false for nonexistent key")
	}

	clearStreamChans()
	_, ok = getStreamChan("test-provider")
	if ok {
		t.Error("expected false after clear")
	}
}

// --- getProviderMentions tests ---

func TestGetProviderMentions(t *testing.T) {
	_ = config.Get()
	mentions := getProviderMentions()

	if len(mentions) == 0 {
		t.Fatal("expected at least 1 mention")
	}
	if mentions[0] != "@all" {
		t.Errorf("expected first mention '@all', got '%s'", mentions[0])
	}

	// Should contain at least the default providers
	found := map[string]bool{}
	for _, m := range mentions {
		found[m] = true
	}
	if !found["@all"] {
		t.Error("expected @all in mentions")
	}
}
