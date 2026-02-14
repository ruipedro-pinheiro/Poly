package session

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Message represents a persisted chat message
type Message struct {
	Role         string    `json:"role"`
	Content      string    `json:"content"`
	Provider     string    `json:"provider,omitempty"`
	Thinking     string    `json:"thinking,omitempty"`
	Images       []string  `json:"images,omitempty"`
	InputTokens  int       `json:"input_tokens,omitempty"`
	OutputTokens int       `json:"output_tokens,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// Session represents a chat session
type Session struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Messages  []Message `json:"messages"`
	Provider  string    `json:"default_provider"`
	Model     string    `json:"model,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SessionEntry is a lightweight session reference for the index
type SessionEntry struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Provider     string    `json:"provider"`
	MessageCount int       `json:"message_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// SessionIndex tracks all sessions and the current one
type SessionIndex struct {
	Current  string         `json:"current"`
	Sessions []SessionEntry `json:"sessions"`
}

var sessionDir string
var currentSession *Session
var sessionIndex *SessionIndex

func init() {
	home, _ := os.UserHomeDir()
	sessionDir = filepath.Join(home, ".poly", "sessions")
}

// SetSessionDir allows overriding the session directory
func SetSessionDir(dir string) {
	sessionDir = dir
}

// GetSessionDir returns the current session directory
func GetSessionDir() string {
	return sessionDir
}

// Load loads the current session from disk (auto-migrates old format)
func Load() (*Session, error) {
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		return nil, err
	}

	// Load or create index
	if err := loadIndex(); err != nil {
		return nil, err
	}

	// If we have a current session ID, load it
	if sessionIndex.Current != "" {
		sess, err := loadSession(sessionIndex.Current)
		if err == nil {
			currentSession = sess
			return currentSession, nil
		}
	}

	// No current session - create new
	currentSession = newBlankSession()
	saveSession(currentSession)
	updateIndex()
	return currentSession, nil
}

// Save saves the current session to disk
func Save() error {
	if currentSession == nil {
		return nil
	}
	currentSession.UpdatedAt = time.Now()
	if err := saveSession(currentSession); err != nil {
		return err
	}
	return updateIndex()
}

// AddMessage adds a message to the current session and saves
func AddMessage(msg Message) error {
	if currentSession == nil {
		Load()
	}
	msg.Timestamp = time.Now()
	currentSession.Messages = append(currentSession.Messages, msg)

	// Auto-title from first user message
	if currentSession.Title == "" && msg.Role == "user" && msg.Content != "" {
		title := msg.Content
		if len(title) > 50 {
			title = title[:50]
		}
		// First line only
		if idx := strings.IndexByte(title, '\n'); idx > 0 {
			title = title[:idx]
		}
		currentSession.Title = title
	}

	return Save()
}

// GetMessages returns all messages in the current session
func GetMessages() []Message {
	if currentSession == nil {
		Load()
	}
	return currentSession.Messages
}

// SetMessages replaces all messages in the current session and saves
func SetMessages(msgs []Message) error {
	if currentSession == nil {
		Load()
	}
	currentSession.Messages = msgs
	return Save()
}

// Clear creates a new session, preserving the old one in the index
func Clear() error {
	// Save current session before clearing
	if currentSession != nil && len(currentSession.Messages) > 0 {
		Save()
	}
	currentSession = newBlankSession()
	if err := saveSession(currentSession); err != nil {
		return err
	}
	return updateIndex()
}

// SetProvider sets the default provider
func SetProvider(provider string) {
	if currentSession == nil {
		Load()
	}
	currentSession.Provider = provider
	Save()
}

// GetProvider returns the default provider
func GetProvider() string {
	if currentSession == nil {
		Load()
	}
	return currentSession.Provider
}

// ListSessions returns all sessions sorted by most recently updated
func ListSessions() []SessionEntry {
	if sessionIndex == nil {
		loadIndex()
	}
	entries := make([]SessionEntry, len(sessionIndex.Sessions))
	copy(entries, sessionIndex.Sessions)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].UpdatedAt.After(entries[j].UpdatedAt)
	})
	return entries
}

// SwitchSession saves the current session and loads a different one
func SwitchSession(id string) error {
	// Save current
	if currentSession != nil {
		Save()
	}

	sess, err := loadSession(id)
	if err != nil {
		return err
	}

	currentSession = sess
	sessionIndex.Current = id
	return saveIndex()
}

// ForkSession copies the current session with a new ID
func ForkSession() (*Session, error) {
	if currentSession == nil {
		Load()
	}

	forked := &Session{
		ID:        generateID(),
		Title:     currentSession.Title + " (fork)",
		Messages:  make([]Message, len(currentSession.Messages)),
		Provider:  currentSession.Provider,
		Model:     currentSession.Model,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	copy(forked.Messages, currentSession.Messages)

	if err := saveSession(forked); err != nil {
		return nil, err
	}

	currentSession = forked
	updateIndex()
	return forked, nil
}

// DeleteSession removes a session by ID
func DeleteSession(id string) error {
	// Can't delete current session
	if currentSession != nil && currentSession.ID == id {
		return nil
	}

	// Remove file
	os.Remove(filepath.Join(sessionDir, id+".json"))

	// Remove from index
	if sessionIndex != nil {
		filtered := make([]SessionEntry, 0, len(sessionIndex.Sessions))
		for _, s := range sessionIndex.Sessions {
			if s.ID != id {
				filtered = append(filtered, s)
			}
		}
		sessionIndex.Sessions = filtered
		return saveIndex()
	}
	return nil
}

// RenameSession changes the title of a session
func RenameSession(id, title string) error {
	sess, err := loadSession(id)
	if err != nil {
		return err
	}
	sess.Title = title
	if err := saveSession(sess); err != nil {
		return err
	}
	if currentSession != nil && currentSession.ID == id {
		currentSession.Title = title
	}
	return updateIndex()
}

// CurrentID returns the current session ID
func CurrentID() string {
	if currentSession == nil {
		return ""
	}
	return currentSession.ID
}

// --- Internal helpers ---

func newBlankSession() *Session {
	return &Session{
		ID:        generateID(),
		Messages:  []Message{},
		Provider:  "claude",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func generateID() string {
	suffix := make([]byte, 4)
	if _, err := rand.Read(suffix); err != nil {
		return time.Now().Format("20060102-150405")
	}
	return time.Now().Format("20060102-150405") + "-" + hex.EncodeToString(suffix)
}

func loadIndex() error {
	indexPath := filepath.Join(sessionDir, "index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Check for old format migration
			return migrateOldFormat()
		}
		return err
	}

	sessionIndex = &SessionIndex{}
	return json.Unmarshal(data, sessionIndex)
}

func saveIndex() error {
	if sessionIndex == nil {
		return nil
	}
	data, err := json.MarshalIndent(sessionIndex, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(sessionDir, "index.json"), data, 0600)
}

func updateIndex() error {
	if sessionIndex == nil {
		sessionIndex = &SessionIndex{}
	}

	if currentSession != nil {
		sessionIndex.Current = currentSession.ID

		// Update or add entry
		found := false
		for i, e := range sessionIndex.Sessions {
			if e.ID == currentSession.ID {
				sessionIndex.Sessions[i] = sessionToEntry(currentSession)
				found = true
				break
			}
		}
		if !found {
			sessionIndex.Sessions = append(sessionIndex.Sessions, sessionToEntry(currentSession))
		}
	}

	return saveIndex()
}

func sessionToEntry(s *Session) SessionEntry {
	return SessionEntry{
		ID:           s.ID,
		Title:        s.Title,
		Provider:     s.Provider,
		MessageCount: len(s.Messages),
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}
}

func loadSession(id string) (*Session, error) {
	path := filepath.Join(sessionDir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

func saveSession(s *Session) error {
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(sessionDir, s.ID+".json"), data, 0600)
}

// migrateOldFormat converts old current.json format to new multi-session format
func migrateOldFormat() error {
	sessionIndex = &SessionIndex{}

	oldPath := filepath.Join(sessionDir, "current.json")
	data, err := os.ReadFile(oldPath)
	if err != nil {
		// No old file either - fresh start
		return nil
	}

	var oldSession Session
	if err := json.Unmarshal(data, &oldSession); err != nil {
		return nil
	}

	if oldSession.ID == "" {
		oldSession.ID = generateID()
	}

	// Auto-title from first user message
	if oldSession.Title == "" {
		for _, msg := range oldSession.Messages {
			if msg.Role == "user" && msg.Content != "" {
				title := msg.Content
				if len(title) > 50 {
					title = title[:50]
				}
				if idx := strings.IndexByte(title, '\n'); idx > 0 {
					title = title[:idx]
				}
				oldSession.Title = title
				break
			}
		}
	}

	// Save as new format
	saveSession(&oldSession)
	sessionIndex.Current = oldSession.ID
	sessionIndex.Sessions = append(sessionIndex.Sessions, sessionToEntry(&oldSession))

	// Remove old file
	os.Remove(oldPath)

	return saveIndex()
}
