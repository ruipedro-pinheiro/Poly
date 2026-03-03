package session

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	SetSessionDir(dir)
	// Reset global state
	currentSession = nil
	sessionIndex = nil
	return dir
}

func TestGenerateID_Format(t *testing.T) {
	id := generateID()
	// Format: YYYYMMDD-HHMMSS-XXXXXXXX (8 hex chars)
	pattern := regexp.MustCompile(`^\d{8}-\d{6}-[0-9a-f]{8}$`)
	if !pattern.MatchString(id) {
		t.Errorf("generateID() = %q, does not match expected format YYYYMMDD-HHMMSS-XXXXXXXX", id)
	}
}

func TestGenerateID_Unique(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := generateID()
		if ids[id] {
			t.Errorf("generateID() produced duplicate ID: %s", id)
		}
		ids[id] = true
	}
}

func TestGenerateID_HexSuffix(t *testing.T) {
	id := generateID()
	parts := strings.Split(id, "-")
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts in ID, got %d: %s", len(parts), id)
	}
	suffix := parts[2]
	if len(suffix) != 8 {
		t.Errorf("expected 8-char hex suffix, got %d chars: %s", len(suffix), suffix)
	}
	_, err := hex.DecodeString(suffix)
	if err != nil {
		t.Errorf("suffix %q is not valid hex: %v", suffix, err)
	}
}

func TestNewBlankSession(t *testing.T) {
	sess := newBlankSession()
	if sess.ID == "" {
		t.Error("new session should have an ID")
	}
	if sess.Provider != "claude" {
		t.Errorf("expected default provider 'claude', got %q", sess.Provider)
	}
	if len(sess.Messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(sess.Messages))
	}
	if sess.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestLoad_CreatesNewSession(t *testing.T) {
	setupTestDir(t)

	sess, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if sess == nil {
		t.Fatal("Load() returned nil session")
	}
	if sess.ID == "" {
		t.Error("loaded session should have an ID")
	}
}

func TestSave_And_Load(t *testing.T) {
	dir := setupTestDir(t)

	// Load creates a new session
	sess, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Add a message
	sess.Messages = append(sess.Messages, Message{
		Role:    "user",
		Content: "hello world",
	})
	currentSession = sess

	// Save
	if err := Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Verify file on disk
	sessionFile := filepath.Join(dir, sess.ID+".json")
	data, err := os.ReadFile(sessionFile)
	if err != nil {
		t.Fatalf("session file not found: %v", err)
	}

	var loaded Session
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to parse session file: %v", err)
	}
	if len(loaded.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(loaded.Messages))
	}
	if loaded.Messages[0].Content != "hello world" {
		t.Errorf("expected message content 'hello world', got %q", loaded.Messages[0].Content)
	}
}

func TestAddMessage(t *testing.T) {
	setupTestDir(t)

	_, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	err = AddMessage(Message{Role: "user", Content: "test message"})
	if err != nil {
		t.Fatalf("AddMessage() error: %v", err)
	}

	// Verify through internal state since GetMessages was removed (dead code)
	sessionMu.Lock()
	msgs := currentSession.Messages
	sessionMu.Unlock()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Content != "test message" {
		t.Errorf("expected 'test message', got %q", msgs[0].Content)
	}
}

func TestAddMessage_AutoTitle(t *testing.T) {
	setupTestDir(t)

	Load()

	AddMessage(Message{Role: "user", Content: "My first question about Go programming"})

	if currentSession.Title != "My first question about Go programming" {
		t.Errorf("expected auto-title to be set, got %q", currentSession.Title)
	}
}

func TestAddMessage_AutoTitleTruncate(t *testing.T) {
	setupTestDir(t)
	Load()

	longMsg := strings.Repeat("a", 100)
	AddMessage(Message{Role: "user", Content: longMsg})

	if len(currentSession.Title) > 50 {
		t.Errorf("expected title truncated to 50 chars, got %d", len(currentSession.Title))
	}
}

func TestAddMessage_AutoTitleFirstLine(t *testing.T) {
	setupTestDir(t)
	Load()

	AddMessage(Message{Role: "user", Content: "First line\nSecond line"})

	if currentSession.Title != "First line" {
		t.Errorf("expected title to be first line only, got %q", currentSession.Title)
	}
}

func TestClear_CreatesNewSession(t *testing.T) {
	setupTestDir(t)

	Load()
	AddMessage(Message{Role: "user", Content: "hello"})
	oldID := currentSession.ID

	err := Clear()
	if err != nil {
		t.Fatalf("Clear() error: %v", err)
	}

	if currentSession.ID == oldID {
		t.Error("Clear() should create a new session with different ID")
	}
	if len(currentSession.Messages) != 0 {
		t.Errorf("expected 0 messages after clear, got %d", len(currentSession.Messages))
	}
}

func TestCurrentID(t *testing.T) {
	setupTestDir(t)

	// Before load
	currentSession = nil
	if CurrentID() != "" {
		t.Error("CurrentID should be empty before Load()")
	}

	Load()
	if CurrentID() == "" {
		t.Error("CurrentID should not be empty after Load()")
	}
}

func TestListSessions(t *testing.T) {
	setupTestDir(t)

	Load()
	AddMessage(Message{Role: "user", Content: "session 1"})

	Clear()
	AddMessage(Message{Role: "user", Content: "session 2"})

	entries := ListSessions()
	if len(entries) < 2 {
		t.Errorf("expected at least 2 sessions, got %d", len(entries))
	}
}

func TestSwitchSession(t *testing.T) {
	setupTestDir(t)

	Load()
	AddMessage(Message{Role: "user", Content: "first session"})
	firstID := currentSession.ID

	Clear()
	AddMessage(Message{Role: "user", Content: "second session"})

	// Switch back to first
	err := SwitchSession(firstID)
	if err != nil {
		t.Fatalf("SwitchSession() error: %v", err)
	}
	if currentSession.ID != firstID {
		t.Errorf("expected session ID %s, got %s", firstID, currentSession.ID)
	}
	if len(currentSession.Messages) != 1 || currentSession.Messages[0].Content != "first session" {
		t.Error("switched session should have the first session's messages")
	}
}

func TestDeleteSession(t *testing.T) {
	setupTestDir(t)

	Load()
	AddMessage(Message{Role: "user", Content: "first"})
	firstID := currentSession.ID

	Clear()
	AddMessage(Message{Role: "user", Content: "second"})

	// Delete the first session
	err := DeleteSession(firstID)
	if err != nil {
		t.Fatalf("DeleteSession() error: %v", err)
	}

	// Verify it's gone from the index
	entries := ListSessions()
	for _, e := range entries {
		if e.ID == firstID {
			t.Error("deleted session should not appear in list")
		}
	}
}

func TestDeleteSession_CannotDeleteCurrent(t *testing.T) {
	setupTestDir(t)
	Load()
	AddMessage(Message{Role: "user", Content: "current"})

	err := DeleteSession(currentSession.ID)
	if err != nil {
		t.Fatalf("DeleteSession() error: %v", err)
	}

	// Current session should still exist
	if currentSession == nil {
		t.Error("current session should not be nil after deleting itself")
	}
}

func TestSessionDir(t *testing.T) {
	dir := "/tmp/test-sessions"
	SetSessionDir(dir)
	if GetSessionDir() != dir {
		t.Errorf("expected %q, got %q", dir, GetSessionDir())
	}
}

func TestForkSession(t *testing.T) {
	setupTestDir(t)

	Load()
	AddMessage(Message{Role: "user", Content: "original message"})
	originalID := currentSession.ID

	forked, err := ForkSession()
	if err != nil {
		t.Fatalf("ForkSession() error: %v", err)
	}
	if forked.ID == originalID {
		t.Error("forked session should have a different ID")
	}
	if len(forked.Messages) != 1 {
		t.Errorf("expected 1 message in fork, got %d", len(forked.Messages))
	}
	if !strings.Contains(forked.Title, "(fork)") {
		t.Errorf("expected forked title to contain '(fork)', got %q", forked.Title)
	}
}

func TestSaveNilSession(t *testing.T) {
	setupTestDir(t)
	currentSession = nil
	err := Save()
	if err != nil {
		t.Fatalf("Save() with nil session should not error: %v", err)
	}
}

func TestSetMessages(t *testing.T) {
	setupTestDir(t)
	Load()

	msgs := []Message{
		{Role: "user", Content: "msg1"},
		{Role: "assistant", Content: "reply1"},
	}
	err := SetMessages(msgs)
	if err != nil {
		t.Fatalf("SetMessages() error: %v", err)
	}

	// Verify through internal state since GetMessages was removed (dead code)
	sessionMu.Lock()
	got := len(currentSession.Messages)
	sessionMu.Unlock()
	if got != 2 {
		t.Errorf("expected 2 messages, got %d", got)
	}
}

func TestSessionFilePermissions(t *testing.T) {
	dir := setupTestDir(t)

	Load()
	Save()

	sessionFile := filepath.Join(dir, currentSession.ID+".json")
	info, err := os.Stat(sessionFile)
	if err != nil {
		t.Fatalf("could not stat session file: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("expected session file permissions 0600, got %o", perm)
	}
}
