package tui

import (
	"strings"
	"testing"

	"github.com/pedromelo/poly/internal/config"
)

// testModel returns a minimal Model suitable for command tests.
func testModel() *Model {
	// Ensure default config is loaded (provides provider names).
	_ = config.Get()
	initMarkdown(80)

	return &Model{
		commands:         initCommands(),
		defaultProvider:  "claude",
		thinkingExpanded: make(map[int]bool),
		providerCosts:    make(map[string]float64),
		approvedTools:    make(map[string]bool),
	}
}

// --- CommandRegistry tests ---

func TestNewCommandRegistry(t *testing.T) {
	r := NewCommandRegistry()
	if r == nil {
		t.Fatal("NewCommandRegistry returned nil")
	}
	if len(r.commands) != 0 {
		t.Errorf("expected empty commands map, got %d entries", len(r.commands))
	}
	if len(r.ordered) != 0 {
		t.Errorf("expected empty ordered slice, got %d entries", len(r.ordered))
	}
}

func TestCommandRegistry_Register(t *testing.T) {
	r := NewCommandRegistry()
	cmd := &Command{
		Name:        "test",
		Aliases:     []string{"t", "tst"},
		Category:    "Test",
		Description: "A test command",
		Usage:       "/test",
		Handler:     func(m *Model, args []string) {},
	}
	r.Register(cmd)

	// Should be findable by name
	if r.Get("test") == nil {
		t.Error("command not found by name 'test'")
	}
	// Should be findable by aliases
	if r.Get("t") == nil {
		t.Error("command not found by alias 't'")
	}
	if r.Get("tst") == nil {
		t.Error("command not found by alias 'tst'")
	}
	// Should NOT be found by unknown name
	if r.Get("unknown") != nil {
		t.Error("expected nil for unknown command")
	}
	// ordered should have 1 entry
	if len(r.ordered) != 1 {
		t.Errorf("expected 1 ordered entry, got %d", len(r.ordered))
	}
}

func TestCommandRegistry_Execute(t *testing.T) {
	r := NewCommandRegistry()
	called := false
	r.Register(&Command{
		Name: "ping",
		Handler: func(m *Model, args []string) {
			called = true
		},
	})

	m := &Model{}
	found := r.Execute(m, "ping", nil)
	if !found {
		t.Error("Execute should return true for registered command")
	}
	if !called {
		t.Error("handler was not called")
	}

	found = r.Execute(m, "nonexistent", nil)
	if found {
		t.Error("Execute should return false for unregistered command")
	}
}

func TestCommandRegistry_Names(t *testing.T) {
	r := NewCommandRegistry()
	r.Register(&Command{Name: "beta", Aliases: []string{"b"}, Handler: func(m *Model, args []string) {}})
	r.Register(&Command{Name: "alpha", Handler: func(m *Model, args []string) {}})

	names := r.Names()
	// Should be sorted alphabetically
	if len(names) != 3 {
		t.Fatalf("expected 3 names, got %d: %v", len(names), names)
	}
	// /alpha, /b, /beta (sorted)
	if names[0] != "/alpha" {
		t.Errorf("expected /alpha first, got %s", names[0])
	}
	if names[1] != "/b" {
		t.Errorf("expected /b second, got %s", names[1])
	}
	if names[2] != "/beta" {
		t.Errorf("expected /beta third, got %s", names[2])
	}
}

func TestCommandRegistry_ByCategory(t *testing.T) {
	r := NewCommandRegistry()
	r.Register(&Command{Name: "a", Category: "Chat", Handler: func(m *Model, args []string) {}})
	r.Register(&Command{Name: "b", Category: "Config", Handler: func(m *Model, args []string) {}})
	r.Register(&Command{Name: "c", Category: "Chat", Handler: func(m *Model, args []string) {}})

	order, groups := r.ByCategory()
	if len(order) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(order))
	}
	if order[0] != "Chat" {
		t.Errorf("expected first category Chat, got %s", order[0])
	}
	if order[1] != "Config" {
		t.Errorf("expected second category Config, got %s", order[1])
	}
	if len(groups["Chat"]) != 2 {
		t.Errorf("expected 2 commands in Chat, got %d", len(groups["Chat"]))
	}
	if len(groups["Config"]) != 1 {
		t.Errorf("expected 1 command in Config, got %d", len(groups["Config"]))
	}
}

func TestCommandRegistry_HelpString(t *testing.T) {
	r := NewCommandRegistry()
	r.Register(&Command{Name: "clear", Aliases: []string{"c"}, Handler: func(m *Model, args []string) {}})
	r.Register(&Command{Name: "help", Handler: func(m *Model, args []string) {}})

	hs := r.HelpString()
	if !strings.Contains(hs, "/clear") {
		t.Error("HelpString should contain /clear")
	}
	if !strings.Contains(hs, "/c") {
		t.Error("HelpString should contain alias /c")
	}
	if !strings.Contains(hs, "/help") {
		t.Error("HelpString should contain /help")
	}
}

func TestCommandRegistry_HelpDetailed(t *testing.T) {
	r := NewCommandRegistry()
	r.Register(&Command{Name: "foo", Usage: "/foo", Description: "Do foo", Handler: func(m *Model, args []string) {}})

	hd := r.HelpDetailed()
	if !strings.Contains(hd, "/foo") {
		t.Error("HelpDetailed should contain /foo")
	}
	if !strings.Contains(hd, "Do foo") {
		t.Error("HelpDetailed should contain description")
	}
}

// --- initCommands tests ---

func TestInitCommands_AllRegistered(t *testing.T) {
	r := initCommands()

	// All expected command names
	expected := []string{
		"clear", "model", "think", "provider", "help", "providers",
		"add", "remove", "context", "addprovider", "delprovider",
		"compact", "theme", "rounds", "export", "search", "notify",
		"sandbox", "yolo", "version", "undo", "rewind", "retry",
		"compare", "config", "revert", "skill", "stats", "costs",
		"memory", "mcp", "project",
	}

	for _, name := range expected {
		if r.Get(name) == nil {
			t.Errorf("expected command '%s' to be registered", name)
		}
	}
}

func TestInitCommands_Aliases(t *testing.T) {
	r := initCommands()

	aliases := map[string]string{
		"c":    "clear",
		"m":    "model",
		"t":    "think",
		"p":    "provider",
		"h":    "help",
		"list": "providers",
		"v":    "version",
		"rw":   "rewind",
		"s":    "search",
		"del":  "delprovider",
	}

	for alias, expectedName := range aliases {
		cmd := r.Get(alias)
		if cmd == nil {
			t.Errorf("alias '%s' not found", alias)
			continue
		}
		if cmd.Name != expectedName {
			t.Errorf("alias '%s' should resolve to '%s', got '%s'", alias, expectedName, cmd.Name)
		}
	}
}

// --- handleCommand tests ---

func TestHandleCommand_Clear(t *testing.T) {
	m := testModel()
	m.messages = []Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "world"},
	}
	m.handleCommand("/clear")

	if len(m.messages) != 0 {
		t.Errorf("expected 0 messages after /clear, got %d", len(m.messages))
	}
	if m.status != "Chat cleared" {
		t.Errorf("expected status 'Chat cleared', got '%s'", m.status)
	}
}

func TestHandleCommand_Model_ShowCurrent(t *testing.T) {
	m := testModel()
	m.handleCommand("/model")

	if !strings.Contains(m.status, "Model:") {
		t.Errorf("expected status to contain 'Model:', got '%s'", m.status)
	}
}

func TestHandleCommand_Model_SetValid(t *testing.T) {
	m := testModel()
	m.handleCommand("/model fast")

	if m.modelVariant != "fast" {
		t.Errorf("expected modelVariant 'fast', got '%s'", m.modelVariant)
	}
	if m.status != "Model set to: fast" {
		t.Errorf("expected status 'Model set to: fast', got '%s'", m.status)
	}
}

func TestHandleCommand_Model_SetInvalid(t *testing.T) {
	m := testModel()
	m.handleCommand("/model invalid")

	if strings.Contains(m.status, "Model set to") {
		t.Errorf("should not set model to invalid variant, status: %s", m.status)
	}
	if !strings.Contains(m.status, "Unknown variant") {
		t.Errorf("expected 'Unknown variant' in status, got '%s'", m.status)
	}
}

func TestHandleCommand_Think(t *testing.T) {
	m := testModel()
	m.thinkingMode = false

	m.handleCommand("/think")
	if !m.thinkingMode {
		t.Error("expected thinkingMode ON after first /think")
	}
	if m.status != "Thinking mode ON" {
		t.Errorf("expected status 'Thinking mode ON', got '%s'", m.status)
	}

	m.handleCommand("/think")
	if m.thinkingMode {
		t.Error("expected thinkingMode OFF after second /think")
	}
	if m.status != "Thinking mode OFF" {
		t.Errorf("expected status 'Thinking mode OFF', got '%s'", m.status)
	}
}

func TestHandleCommand_Rounds_ShowCurrent(t *testing.T) {
	m := testModel()
	m.handleCommand("/rounds")

	if !strings.Contains(m.status, "Max Table Ronde rounds") {
		t.Errorf("expected status to contain rounds info, got '%s'", m.status)
	}
}

func TestHandleCommand_Rounds_SetValid(t *testing.T) {
	m := testModel()
	m.handleCommand("/rounds 10")

	if !strings.Contains(m.status, "10") {
		t.Errorf("expected status to contain '10', got '%s'", m.status)
	}
}

func TestHandleCommand_Rounds_SetInvalid(t *testing.T) {
	m := testModel()
	m.handleCommand("/rounds 0")

	if !strings.Contains(m.status, "Usage") {
		t.Errorf("expected usage message for invalid round, got '%s'", m.status)
	}

	m.handleCommand("/rounds abc")
	if !strings.Contains(m.status, "Usage") {
		t.Errorf("expected usage message for non-numeric, got '%s'", m.status)
	}

	m.handleCommand("/rounds 21")
	if !strings.Contains(m.status, "Usage") {
		t.Errorf("expected usage message for > 20, got '%s'", m.status)
	}
}

func TestHandleCommand_Version(t *testing.T) {
	m := testModel()
	m.handleCommand("/version")

	if !strings.Contains(m.status, "Poly v") {
		t.Errorf("expected status with version, got '%s'", m.status)
	}
}

func TestHandleCommand_Unknown(t *testing.T) {
	m := testModel()
	m.handleCommand("/nonexistent_command_xyz")

	if !strings.Contains(m.status, "Unknown") {
		t.Errorf("expected 'Unknown' in status for unknown command, got '%s'", m.status)
	}
}

func TestHandleCommand_AliasWorks(t *testing.T) {
	m := testModel()
	m.messages = []Message{{Role: "user", Content: "hi"}}
	m.handleCommand("/c") // alias for /clear

	if len(m.messages) != 0 {
		t.Errorf("expected 0 messages after /c (alias for /clear), got %d", len(m.messages))
	}
}

func TestHandleCommand_Yolo(t *testing.T) {
	m := testModel()
	m.handleCommand("/yolo")

	if !strings.Contains(m.status, "YOLO mode ON") {
		t.Errorf("expected YOLO mode ON, got '%s'", m.status)
	}

	m.handleCommand("/yolo")
	if !strings.Contains(m.status, "YOLO mode OFF") {
		t.Errorf("expected YOLO mode OFF, got '%s'", m.status)
	}
}

func TestHandleCommand_Compact_WhileStreaming(t *testing.T) {
	m := testModel()
	m.isStreaming = true
	m.handleCommand("/compact")

	if m.status != "Cannot compact while streaming" {
		t.Errorf("expected streaming error, got '%s'", m.status)
	}
}

func TestHandleCommand_Compact_NotEnoughMessages(t *testing.T) {
	m := testModel()
	m.messages = []Message{{Role: "user", Content: "hi"}}
	m.handleCommand("/compact")

	if !strings.Contains(m.status, "Not enough") {
		t.Errorf("expected not enough messages, got '%s'", m.status)
	}
}

func TestHandleCommand_Help_WithValidCommand(t *testing.T) {
	m := testModel()
	m.handleCommand("/help clear")

	if !strings.Contains(m.status, "/clear") {
		t.Errorf("expected /clear in help output, got '%s'", m.status)
	}
	if !strings.Contains(m.status, "Clear conversation") {
		t.Errorf("expected description in help output, got '%s'", m.status)
	}
}

func TestHandleCommand_Help_WithUnknownCommand(t *testing.T) {
	m := testModel()
	m.handleCommand("/help nonexistent")

	if !strings.Contains(m.status, "Unknown command") {
		t.Errorf("expected 'Unknown command' in status, got '%s'", m.status)
	}
}

func TestHandleCommand_Help_NoArgs(t *testing.T) {
	m := testModel()
	m.handleCommand("/help")

	if m.state != viewHelp {
		t.Errorf("expected state viewHelp, got %d", m.state)
	}
}

// --- parseProvider tests ---

func TestParseProvider_Default(t *testing.T) {
	_ = config.Get()
	m := Model{defaultProvider: "claude"}
	result := m.parseProvider("just a normal message")

	if result != "claude" {
		t.Errorf("expected 'claude' for no mention, got '%s'", result)
	}
}

func TestParseProvider_AtAll(t *testing.T) {
	_ = config.Get()
	m := Model{defaultProvider: "claude"}
	result := m.parseProvider("@all what do you think?")

	if result != "all" {
		t.Errorf("expected 'all' for @all mention, got '%s'", result)
	}
}

func TestParseProvider_SpecificProvider(t *testing.T) {
	_ = config.Get()
	m := Model{defaultProvider: "claude"}

	// Test with known providers from default config
	for _, p := range []string{"gpt", "gemini", "grok"} {
		result := m.parseProvider("hey @" + p + " what do you think?")
		if result != p {
			t.Errorf("expected '%s', got '%s'", p, result)
		}
	}
}

func TestParseProvider_CaseInsensitive(t *testing.T) {
	_ = config.Get()
	m := Model{defaultProvider: "claude"}
	result := m.parseProvider("Hey @Claude what's up?")

	if result != "claude" {
		t.Errorf("expected 'claude' (case-insensitive), got '%s'", result)
	}
}

// --- Undo tests ---

func TestHandleUndo_Empty(t *testing.T) {
	m := testModel()
	m.handleUndo()

	if m.status != "Nothing to undo" {
		t.Errorf("expected 'Nothing to undo', got '%s'", m.status)
	}
}

func TestHandleUndo_WhileStreaming(t *testing.T) {
	m := testModel()
	m.isStreaming = true
	m.messages = []Message{{Role: "user", Content: "hi"}}
	m.handleUndo()

	if m.status != "Cannot undo while streaming" {
		t.Errorf("expected streaming error, got '%s'", m.status)
	}
}

func TestHandleUndo_RemovesLastExchange(t *testing.T) {
	m := testModel()
	m.messages = []Message{
		{Role: "user", Content: "first"},
		{Role: "assistant", Content: "resp1"},
		{Role: "user", Content: "second"},
		{Role: "assistant", Content: "resp2"},
	}
	m.handleUndo()

	if len(m.messages) != 2 {
		t.Fatalf("expected 2 messages after undo, got %d", len(m.messages))
	}
	if m.messages[0].Content != "first" {
		t.Errorf("expected first message preserved, got '%s'", m.messages[0].Content)
	}
	if m.messages[1].Content != "resp1" {
		t.Errorf("expected resp1 preserved, got '%s'", m.messages[1].Content)
	}
}

func TestHandleUndo_NoUserMessage(t *testing.T) {
	m := testModel()
	m.messages = []Message{
		{Role: "system", Content: "system msg"},
	}
	m.handleUndo()

	if m.status != "No user message to undo" {
		t.Errorf("expected 'No user message to undo', got '%s'", m.status)
	}
}

// --- Rewind tests ---

func TestHandleRewind_Empty(t *testing.T) {
	m := testModel()
	m.handleRewind(nil)

	if m.status != "Nothing to rewind" {
		t.Errorf("expected 'Nothing to rewind', got '%s'", m.status)
	}
}

func TestHandleRewind_WhileStreaming(t *testing.T) {
	m := testModel()
	m.isStreaming = true
	m.messages = []Message{{Role: "user", Content: "hi"}}
	m.handleRewind(nil)

	if m.status != "Cannot rewind while streaming" {
		t.Errorf("expected streaming error, got '%s'", m.status)
	}
}

func TestHandleRewind_Default(t *testing.T) {
	m := testModel()
	m.messages = []Message{
		{Role: "user", Content: "1"},
		{Role: "assistant", Content: "2"},
		{Role: "user", Content: "3"},
		{Role: "assistant", Content: "4"},
	}
	m.handleRewind(nil) // default removes 2

	if len(m.messages) != 2 {
		t.Errorf("expected 2 messages after default rewind, got %d", len(m.messages))
	}
}

func TestHandleRewind_CustomN(t *testing.T) {
	m := testModel()
	m.messages = []Message{
		{Role: "user", Content: "1"},
		{Role: "assistant", Content: "2"},
		{Role: "user", Content: "3"},
	}
	m.handleRewind([]string{"1"})

	if len(m.messages) != 2 {
		t.Errorf("expected 2 messages after rewind 1, got %d", len(m.messages))
	}
}

func TestHandleRewind_MoreThanAvailable(t *testing.T) {
	m := testModel()
	m.messages = []Message{
		{Role: "user", Content: "1"},
	}
	m.handleRewind([]string{"100"})

	if len(m.messages) != 0 {
		t.Errorf("expected 0 messages when rewinding more than available, got %d", len(m.messages))
	}
}

func TestHandleRewind_InvalidN(t *testing.T) {
	m := testModel()
	m.messages = []Message{{Role: "user", Content: "1"}}
	m.handleRewind([]string{"abc"})

	if !strings.Contains(m.status, "Usage") {
		t.Errorf("expected usage message for invalid N, got '%s'", m.status)
	}
}

func TestHandleRewind_NegativeN(t *testing.T) {
	m := testModel()
	m.messages = []Message{{Role: "user", Content: "1"}}
	m.handleRewind([]string{"-1"})

	if !strings.Contains(m.status, "Usage") {
		t.Errorf("expected usage message for negative N, got '%s'", m.status)
	}
}

// --- Retry tests ---

func TestHandleRetry_Empty(t *testing.T) {
	m := testModel()
	m.handleRetry()

	if m.status != "Nothing to retry" {
		t.Errorf("expected 'Nothing to retry', got '%s'", m.status)
	}
}

func TestHandleRetry_WhileStreaming(t *testing.T) {
	m := testModel()
	m.isStreaming = true
	m.messages = []Message{{Role: "user", Content: "hi"}}
	m.handleRetry()

	if m.status != "Cannot retry while streaming" {
		t.Errorf("expected streaming error, got '%s'", m.status)
	}
}

func TestHandleRetry_CapturesLastUserMsg(t *testing.T) {
	m := testModel()
	m.messages = []Message{
		{Role: "user", Content: "question"},
		{Role: "assistant", Content: "answer"},
	}
	m.handleRetry()

	if m.retryContent != "question" {
		t.Errorf("expected retryContent 'question', got '%s'", m.retryContent)
	}
	if len(m.messages) != 0 {
		t.Errorf("expected 0 messages after retry (user msg at idx 0 removed), got %d", len(m.messages))
	}
}

func TestHandleRetry_NoUserMessage(t *testing.T) {
	m := testModel()
	m.messages = []Message{
		{Role: "system", Content: "sys"},
		{Role: "assistant", Content: "resp"},
	}
	m.handleRetry()

	if m.status != "No user message to retry" {
		t.Errorf("expected 'No user message to retry', got '%s'", m.status)
	}
}

// --- Context file commands ---

func TestHandleRemoveFileCommand_NoArgs(t *testing.T) {
	m := testModel()
	m.handleRemoveFileCommand(nil)

	if m.status != "Usage: /remove <file>" {
		t.Errorf("expected usage message, got '%s'", m.status)
	}
}

func TestHandleRemoveFileCommand_NotInContext(t *testing.T) {
	m := testModel()
	m.contextFiles = []string{"/a.go"}
	m.handleRemoveFileCommand([]string{"/b.go"})

	if !strings.Contains(m.status, "Not in context") {
		t.Errorf("expected 'Not in context', got '%s'", m.status)
	}
}

func TestHandleRemoveFileCommand_RemovesFile(t *testing.T) {
	m := testModel()
	m.contextFiles = []string{"/a.go", "/b.go", "/c.go"}
	m.handleRemoveFileCommand([]string{"/b.go"})

	if len(m.contextFiles) != 2 {
		t.Errorf("expected 2 context files, got %d", len(m.contextFiles))
	}
	for _, f := range m.contextFiles {
		if f == "/b.go" {
			t.Error("/b.go should have been removed")
		}
	}
}

func TestHandleContextCommand_Empty(t *testing.T) {
	m := testModel()
	m.handleContextCommand()

	if m.status != "No files in context. Use /add <file>" {
		t.Errorf("expected empty context message, got '%s'", m.status)
	}
}

func TestHandleAddFileCommand_NoArgs(t *testing.T) {
	m := testModel()
	m.handleAddFileCommand(nil)

	if m.status != "Usage: /add <file_or_dir>" {
		t.Errorf("expected usage message, got '%s'", m.status)
	}
}

func TestHandleAddFileCommand_FileNotFound(t *testing.T) {
	m := testModel()
	m.handleAddFileCommand([]string{"/nonexistent/path/xyz.go"})

	if !strings.Contains(m.status, "File not found") {
		t.Errorf("expected 'File not found', got '%s'", m.status)
	}
}

func TestHandleAddFileCommand_AlreadyInContext(t *testing.T) {
	m := testModel()
	// Use a file that exists
	m.contextFiles = []string{"/dev/null"}
	m.handleAddFileCommand([]string{"/dev/null"})

	if m.status != "Files already in context" {
		t.Errorf("expected 'Files already in context', got '%s'", m.status)
	}
}

// --- buildContextPrefix ---

func TestBuildContextPrefix_Empty(t *testing.T) {
	m := testModel()
	result := m.buildContextPrefix()
	if result != "" {
		t.Errorf("expected empty string for no context files, got '%s'", result)
	}
}

func TestBuildContextPrefix_WithFiles(t *testing.T) {
	m := testModel()
	m.contextFiles = []string{"/dev/null"}
	result := m.buildContextPrefix()
	if !strings.Contains(result, "<context>") {
		t.Error("expected <context> tag in prefix")
	}
	if !strings.Contains(result, "</context>") {
		t.Error("expected </context> tag in prefix")
	}
	if !strings.Contains(result, "/dev/null") {
		t.Error("expected file path in prefix")
	}
}

// --- Costs export format ---

func TestHandleCommand_Costs_NoCostData(t *testing.T) {
	m := testModel()
	m.handleCommand("/costs")

	if m.status != "No cost data to export" {
		t.Errorf("expected 'No cost data to export', got '%s'", m.status)
	}
}

func TestHandleCommand_Costs_JsonFormat(t *testing.T) {
	m := testModel()
	m.handleCommand("/costs json")

	// No data, should get the same message regardless of format
	if m.status != "No cost data to export" {
		t.Errorf("expected 'No cost data to export', got '%s'", m.status)
	}
}

// --- Memory command ---

func TestHandleCommand_Memory_UnknownSub(t *testing.T) {
	m := testModel()
	m.handleMemoryCommand([]string{"invalid"})

	if m.status != "Usage: /memory [show|clear]" {
		t.Errorf("expected usage message, got '%s'", m.status)
	}
}

// --- Notify toggle ---

func TestHandleCommand_Notify(t *testing.T) {
	m := testModel()
	m.notificationsOn = false

	m.handleCommand("/notify")
	if !m.notificationsOn {
		t.Error("expected notifications ON")
	}
	if m.status != "Notifications ON" {
		t.Errorf("expected 'Notifications ON', got '%s'", m.status)
	}

	m.handleCommand("/notify")
	if m.notificationsOn {
		t.Error("expected notifications OFF")
	}
	if m.status != "Notifications OFF" {
		t.Errorf("expected 'Notifications OFF', got '%s'", m.status)
	}
}

// --- AddProvider parse tests ---

func TestHandleCommand_AddProvider_NotEnoughArgs(t *testing.T) {
	m := testModel()
	m.handleCommand("/addprovider myid http://x apikey")

	if !strings.Contains(m.status, "Usage") {
		t.Errorf("expected usage message with < 4 args, got '%s'", m.status)
	}
}

// --- Stats command (reads model state, no I/O) ---

func TestHandleStatsCommand(t *testing.T) {
	m := testModel()
	m.messages = []Message{
		{Role: "user", Content: "hi"},
		{Role: "assistant", Content: "hello", ToolCalls: []ToolCallData{{Name: "test"}}},
	}
	m.sessionInputTokens = 100
	m.sessionOutputTokens = 200
	m.sessionCost = 0.0025

	m.handleStatsCommand()

	if len(m.messages) != 3 { // 2 original + 1 system stats
		t.Errorf("expected 3 messages after /stats, got %d", len(m.messages))
	}
	statsMsg := m.messages[2].Content
	if !strings.Contains(statsMsg, "1 user") {
		t.Errorf("expected '1 user' in stats, got '%s'", statsMsg)
	}
	if !strings.Contains(statsMsg, "1 assistant") {
		t.Errorf("expected '1 assistant' in stats, got '%s'", statsMsg)
	}
	if !strings.Contains(statsMsg, "100") {
		t.Errorf("expected input tokens in stats")
	}
	if !strings.Contains(statsMsg, "200") {
		t.Errorf("expected output tokens in stats")
	}
	if !strings.Contains(statsMsg, "Tool calls: 1") {
		t.Errorf("expected tool calls in stats")
	}
}
