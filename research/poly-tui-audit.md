# Poly-go TUI Audit Report

**Date:** 2026-02-10
**Source:** Full code audit of `/home/pedro/PROJETS-AI/Poly-go/internal/tui/` (~47 Go files)
**Version:** v0.2.0

---

## 1. Architecture Overview

### Stack
- **Framework:** BubbleTea v2 (Elm architecture)
- **Styling:** Lipgloss v2 (Catppuccin Mocha theme, accent Mauve `#cba6f7`)
- **Layout:** Custom component system with `layout.Model`, `layout.Sizeable`, `layout.Focusable` interfaces
- **Gradient rendering:** go-colorful (HCL/Luv blending for header and sidebar logos)

### Component Hierarchy
```
Model (tui/model.go)
  |-- headerBar     (components/header)
  |-- viewport      (bubbles/viewport - chat messages)
  |-- textarea      (bubbles/textarea - input)
  |-- sidebarCmp    (components/sidebar)
  |-- statusBar     (components/status)
  |-- dialogMgr     (components/dialogs - LIFO stack)
  |-- splashCmp     (components/splash)
```

### View States (Dual System)
The TUI uses TWO dialog systems in parallel:

1. **Legacy `viewState` enum** (currently active, handles all UI):
   - `viewSplash`, `viewChat`, `viewModelPicker`, `viewControlRoom`, `viewHelp`, `viewAddProvider`, `viewCommandPalette`, `viewSessionList`, `viewApproval`

2. **New DialogCmp stack** (components/dialogs - LIFO stack, partially wired but NOT used for main routing):
   - Dialog IDs: `help`, `model_picker`, `control_room`, `add_provider`, `command_palette`, `session_list`, `approval`
   - Each has a standalone component implementation in `components/dialogs/*/`
   - The `DialogCmp` manager receives `OpenDialogMsg`/`CloseDialogMsg` but `handleDialogClosed()` is a no-op placeholder

**Conclusion:** The new dialog component system is fully implemented but not yet wired into the main Update loop. Both legacy viewState rendering and new component rendering exist side by side.

---

## 2. Complete Keyboard Shortcuts

### Global Shortcuts
| Key | Action | Context |
|-----|--------|---------|
| `Ctrl+C` | Quit (cancels streaming if active) | Always |
| `Ctrl+H` | Toggle Help dialog | Always (except streaming) |
| `Ctrl+O` | Toggle Model Picker | Always (except streaming) |
| `Ctrl+D` | Toggle Control Room | Always (except streaming) |
| `Ctrl+K` | Toggle Command Palette | Always (except streaming) |
| `Ctrl+S` | Toggle Session List | Always (except streaming) |
| `Ctrl+T` | Toggle Thinking Mode (on/off) | Always (except streaming) |
| `Ctrl+N` | New Session (clears messages, tokens, cost) | Always (except streaming) |
| `Ctrl+L` | Clear Chat (clears messages only) | Always (except streaming) |
| `Esc` | Cancel streaming / Close dialog / Return to chat | Context-dependent |

### Chat Mode
| Key | Action |
|-----|--------|
| `Enter` | Send message |
| `Ctrl+V` | Paste text or image from clipboard |
| `Up/Down` | Scroll viewport (disabled in viewport, handled by viewState) |
| `PageUp/PageDown` | Page scroll |
| `Home/End` | Jump to top/bottom |
| `Tab/Shift+Tab` | Tab completion (defined but not yet functional) |
| `Ctrl+Y` | Copy (defined but implementation unclear) |

### Approval Dialog
| Key | Action |
|-----|--------|
| `A` | Allow (one-time) |
| `S` | Allow for Session (auto-approve same tool) |
| `D` / `Esc` / `N` | Deny |
| `Left/Right/Tab/Shift+Tab` | Navigate between buttons |
| `Enter` | Confirm selected button |
| `H/L` | Vim-style left/right navigation |

### Control Room
| Key | Action |
|-----|--------|
| `Up/Down` | Navigate provider list |
| `Enter` | Connect provider / Set as default (if already connected) |
| `Backspace/Delete` | Disconnect provider |
| `N` | Open Add Provider form |
| `Esc` | Close |

### Control Room (Auth Input Mode)
| Key | Action |
|-----|--------|
| `Enter` | Submit OAuth code or API key |
| `Esc` | Cancel auth flow |
| `Ctrl+V/Ctrl+P` | Paste from clipboard |
| Single chars | Type into auth input |
| `Backspace` | Delete last char |

### Add Provider Form
| Key | Action |
|-----|--------|
| `Tab/Down` | Next field |
| `Shift+Tab/Up` | Previous field |
| `Left/Right` | Cycle format (OpenAI/Anthropic/Google) when on format field |
| `Enter` | Save provider (validates ID, URL, Model required) |
| `Esc` | Cancel |
| `Backspace` | Delete last char in current field |
| Single chars | Type into current field |

### Model Picker
| Key | Action |
|-----|--------|
| `Up/Down` | Navigate model list |
| `Enter` | Select model |
| `Esc` | Cancel |
| Single chars | Filter models by name |
| `Backspace` | Delete filter character |

### Command Palette
| Key | Action |
|-----|--------|
| `Up/Down` | Navigate command list |
| `Enter` | Execute selected command |
| `Esc` | Cancel |
| Single chars | Filter commands |
| `Backspace` | Delete filter character |

### Session List
| Key | Action |
|-----|--------|
| `Up/Down` / `K/J` | Navigate sessions |
| `Enter` | Open/switch to selected session |
| `N` | Create new session |
| `D` | Delete selected session (not current) |
| `F` | Fork current session |
| `Esc` | Close |

### Splash Screen
| Key | Action |
|-----|--------|
| Any key | Dismiss splash, enter chat |
| `Ctrl+C` | Quit |

---

## 3. Dialogs

### 3.1 Splash Screen (`viewSplash`)
- Centered fullscreen display
- Gradient logo: `◇  P  O  L  Y` with Mauve-to-Lavender gradient (HCL blending)
- Diagonal fill lines (`╱╱╱`) above and below
- Subtitle: "multi-model terminal interface" (italic, dim)
- Provider status row: check/cross for each provider (claude, gpt, gemini, grok) with color
- Hint: "Press any key" + version (v0.2.0)

### 3.2 Help Dialog (`viewHelp`)
- Sections: NAVIGATION, CHAT, MENTIONS, COMMANDS
- Lists all shortcuts with aligned key-description columns
- Key column width: 10 chars, Lavender bold
- "Esc to close" footer
- Centered in rounded border with Mauve accent

### 3.3 Control Room (`viewControlRoom`)
- Title with diagonal fill: "Control Room ╱╱╱╱╱╱"
- Provider list with:
  - Selection cursor (`> ` with thick left border accent)
  - Provider name (color-coded: Peach=Claude, Green=GPT, Blue=Gemini, Sky=Grok)
  - Connection badge: green background "API"/"OAuth" if connected, dim "- API Key"/"- OAuth" if not
  - Yellow `*` for default provider
- Auth input area (appears when connecting):
  - Bordered text input with placeholder
  - API keys are masked: `sk-x****xxxx`
  - Status messages (Exchanging code..., Error: ...)
  - Ctrl+V or Ctrl+P to paste
- Hints: navigate / connect / disconnect / add provider / close

### 3.4 Add Provider (`viewAddProvider`)
- 4 text fields: ID, URL, API Key, Model
- 1 format selector: OpenAI / Anthropic / Google (pill-style toggle)
- Tab/Shift+Tab to navigate fields, Left/Right for format
- Cursor indicator (`_`) on active field
- API key field is masked
- Placeholders: "mistral, ollama, groq...", "https://api.mistral.ai/v1", etc.
- Validates: ID + URL + Model are required
- Saves via `llm.SaveCustomProvider()` with auto-detection of format

### 3.5 Model Picker (`viewModelPicker`)
- Title: "Select Model"
- Type-to-filter with instant filtering (appears only when active)
- Recently Used section (up to 3, shown when no filter)
- Grouped by provider with colored provider headers
- Each row: variant name (28 chars) + provider badge (colored) + current marker (green dot)
- Selection: thick left border accent
- Variants ordered: default, fast, nano, lite, mini, think, opus, pro, sonnet4, o3, o3pro

### 3.6 Command Palette (`viewCommandPalette`)
- Title: "Commands"
- Bordered filter input with "> " prefix
- Type-to-filter (case-insensitive)
- Available commands:
  1. New Session (ctrl+n)
  2. Switch Model (ctrl+o)
  3. Toggle Thinking
  4. Session List (ctrl+s)
  5. Toggle Sidebar
  6. Control Room (ctrl+d)
  7. Toggle Help (ctrl+h)
  8. Clear Chat (ctrl+l)
  9. Toggle YOLO Mode
  10. Quit (ctrl+c)
- Selection with thick left border accent + `> ` cursor

### 3.7 Session List (`viewSessionList`)
- Title: "Sessions"
- Each session shows: selection cursor, current marker (`*`), title, provider (@name), message count, time ago
- Actions: open, new, delete (not current), fork
- Time formatting: "just now", "5m ago", "3h ago", "2d ago", "Jan 2"

### 3.8 Approval Dialog (`viewApproval`)
- Title: "Permission Required" with diagonal fill
- Shows tool name (Peach, bold) + "wants to execute:"
- Tool-specific content rendering:
  - **bash**: Command with `$ ` prefix, multiline with indentation, max 500 chars
  - **write_file**: Path (Blue, bold) + line count
  - **edit_file**: Path (Blue, bold)
  - **multiedit**: File count + file paths
  - Default: Summary text
- 3 buttons with underlined hotkeys: `[A]llow  Allow for [S]ession  [D]eny`
- Selected button: Mauve background, others: Surface1 background
- Buttons stack vertically if too wide

---

## 4. Visual Components

### 4.1 Header Bar (`components/header`)
- Single line, full width
- Left: `◇ POLY` with Mauve-to-Lavender gradient (per-character HCL blending, bold)
- Center: Diagonal fill `╱╱╱` with same gradient (non-bold)
- Right: CWD path + context percentage (token usage / 200K) + "ctrl+d open"
- Parts separated by ` • `

### 4.2 Sidebar (`components/sidebar`)
- Width: 24px (hidden when terminal < 80 cols)
- Left border: NormalBorder, Surface1 color
- Sections:
  1. **Logo**: Gradient diagonal lines + `◇ P O L Y` gradient text
  2. **Model Info**: Provider name (colored, bold), thinking status, tokens (formatted: K/M), context % (yellow >75%, red >90%), cost
  3. **YOLO Warning**: Red background `YOLO` badge (when active)
  4. **Modified Files**: Section header with separator, file basenames with +/- diff stats (green/red)
  5. **Providers**: 2-per-row grid, check/cross icons with names
  6. **Todos**: Section header, counts (pending/active/done), list of active/pending items with icons (active: `>` Mauve, pending: `○`)
- Caches: Todos from `~/.poly/todos.json` (5s TTL), MCP messages from AI bridge data.json (5s TTL)

### 4.3 Status Bar (`components/status`)
- Full width, Mantle background
- Right-aligned content: token count (Xk) + cost ($X.XX) + status badge + @provider
- Status badges with colored backgrounds: ERROR (Red), OK (Green), WARN (Yellow)
- Auto-clear after 5s TTL (configurable per message)
- Provider name in bold with provider color

### 4.4 Input Area (inline in views.go)
- Rounded border box, Mauve border when focused, Surface2 when blurred
- Provider color dot (`●`) prefix
- Image attachment indicator: `[img:N]` (Green)
- Single-line textarea (height=1), no line numbers, no newline insertion
- Below box: hints line ("enter send . ctrl+k commands . @provider or @all")
- During streaming: "esc stop streaming"

### 4.5 Editor Component (`components/editor`)
- Standalone component (not yet used in main model - the main model uses inline textarea)
- 3-line height textarea
- `>` prompt (red in YOLO mode, Mauve otherwise)
- Provider indicator: `@provider`
- Image count indicator: `[img:N]`
- Hints: "enter send . esc cancel . ctrl+v paste"

### 4.6 Chat Messages (inline in views.go)
- **User messages**: Mauve thick left border, `▌ You` label, content
- **Assistant messages**: Provider-colored thick left border, `◇ Provider` label + separator fill (`───`)
- **Thinking block**: Rounded border (Lavender), `⟳ Thinking` label, italic content
- **Tool calls**: Rendered inline between text blocks (interleaved `ContentBlock` system)
  - Running: Rounded border, Surface2
  - Error: Rounded border, Red
  - Success: Dim compact line, no box
- **Legacy tool block** (old messages): All tools in one container, collapses completed

### 4.7 Splash Component (`components/splash`)
- Separate reusable component (used by main model)
- Centered fullscreen with provider status grid

---

## 5. Streaming System

### Architecture
- Uses Go channels (`<-chan llm.StreamEvent`) for receiving stream events
- Global `streamEventChan` for single-provider streaming
- Global `cascadeStreamChans` map for @all multi-provider streaming
- BubbleTea commands (`readStreamEvent`, `readCascadeEvent`) poll channels in goroutines

### Event Types
| Event | Handling |
|-------|----------|
| `content` | Appends to last message Content + Blocks |
| `thinking` | Appends to last message Thinking (if thinkingMode on) |
| `tool_use` | Creates new ToolCallData (status=running), adds ContentBlock(type="tool") |
| `tool_result` | Updates matching ToolCallData with result, sets status (success/error) |
| `done` | Tracks tokens (input/output), calculates cost, shows response time |
| `error` | Displays error, special handling for image support errors (auto-retry without images) |

### Token & Cost Tracking
- Session-level accumulation of input/output tokens
- Cost calculated via `llm.CalculateCost()` using pricing table
- Response time tracking (`time.Since(streamStartTime)`)
- Displayed in status bar and sidebar

### Image Support
- Auto-detection per provider via `llm.SupportsImages()`
- Fallback: If image error received, disables image support for that provider and retries
- Images from clipboard (Ctrl+V) or file paths
- Clipboard: Wayland (`wl-paste`), X11 (`xclip`), macOS (`pngpaste`), Windows (PowerShell)

### @all Cascade System
- Sorts configured providers by cost tier (ascending)
- Phase 1: Cheapest provider responds (as "responder")
- Phase 2: All other providers review the response (as "reviewers")
- Reviewers get a structured review prompt with the original question + responder's answer
- Reviewers output `✓` if correct, or corrections if not
- Each provider gets its own message slot in the chat
- All streams run in parallel during review phase
- Images forwarded to reviewers if supported

### Approval System
- Tools that need permission send `PendingApproval` via `tools.PendingChan`
- TUI polls with `watchForApprovals()` command
- Auto-approved if tool is in `approvedTools` map (from "Allow for Session")
- Approved/denied via `tools.ApprovedChan` (bool channel)
- YOLO mode: `tools.YoloMode` bypasses all approval

---

## 6. Diff Rendering

### Implementation (`diff_render.go`)
Line-by-line styling with state machine tracking `inDiff` flag:

| Line Pattern | Style | Color |
|-------------|-------|-------|
| `--- a/` or `+++ b/` | Bold | Blue |
| `@@ ... @@` | Normal | Mauve |
| `+...` (in diff) | Normal | Green |
| `-...` (in diff) | Normal | Red |
| ` ...` (context, in diff) | Normal | Overlay1 (dim) |
| `\...` (no newline) | Italic | Overlay0 |
| `Edited`/`Created`/`Updated` | Normal | Green |
| `Error:`/`ERROR:` | Normal | Red |
| Everything else | Normal | Text |

### Limitations
- No syntax highlighting within diff blocks
- No line numbers in diffs
- No side-by-side diff view
- No collapsible diff sections
- State resets on non-diff lines (no hunk grouping)

---

## 7. Tool Output Rendering

### Registry System (`components/messages/tools/`)
Factory pattern with `init()` auto-registration:

| Tool | Renderer | Display |
|------|----------|---------|
| `bash` | bashRenderer | `✓ Bash(command here)` + 3-line result preview |
| `edit_file` | editRenderer | `✓ Edit(path)` + `+N -N lines` stats |
| `read_file` | readRenderer | `✓ Read(path)` + `N lines` count |
| `glob` | readRenderer | `✓ Glob(pattern)` + `N files` count |
| `grep` | readRenderer | `✓ Grep(pattern)` + `N matches` count |
| `write_file` | writeRenderer | `✓ Write(path)` + `N lines` count |
| `web_fetch` | webRenderer | `✓ Fetch(url)` + 2-line result preview |
| `web_search` | webRenderer | `✓ Search(query)` + 2-line result preview |
| `todos` | todosRenderer | `✓ Todos  2 pending . 1 active . 3 done` (compact summary) |
| `_generic` | genericRenderer | Fallback: name + first string arg + 2-line preview |

### Status Icons
| Status | Icon | Color |
|--------|------|-------|
| Pending | `●` | Yellow |
| Running | `⟳` | Mauve |
| Success | `✓` | Green |
| Error | `×` | Red |
| Denied | `×` | Red |

### Result Preview Format
```
│  line 1
│  line 2
│  line 3
   ... +N lines
```

---

## 8. Slash Commands

| Command | Aliases | Description |
|---------|---------|-------------|
| `/clear` | `/c` | Clear chat + session |
| `/model` | `/m` | Show/change model variant (fast/think/opus/default) |
| `/think` | `/t` | Toggle thinking mode |
| `/provider` | `/p` | Show/change default provider |
| `/help` | `/h` | Show available commands |
| `/providers` | `/list` | List all providers |
| `/addprovider` | `/add` | Add custom provider (ID, URL, APIKey, Model, [Format], [Color]) |
| `/delprovider` | `/del` | Delete custom provider |
| `/sidebar` | - | Toggle sidebar visibility |
| `/yolo` | - | Toggle YOLO mode (auto-approve all tools) |

---

## 9. Provider Mentions

| Mention | Target |
|---------|--------|
| `@claude` | Claude provider |
| `@gpt` | GPT provider |
| `@gemini` | Gemini provider |
| `@grok` | Grok provider |
| `@all` | Cascade: cheapest responds first, others review |

---

## 10. Session Management

- **Persistence:** Messages saved to disk via `session` package
- **Operations:** New, Switch, Delete, Fork
- **Auto-save:** Messages persisted on every stream completion
- **Provider per session:** Saved and restored on session switch
- **Session listing:** Shows title, provider, message count, last updated time

---

## 11. Virtualized List Component (`list/`)

A custom high-performance list with:
- **Virtualization:** Only renders visible items
- **Scroll modes:** Normal and reverse (newest-at-bottom for chat)
- **Cache:** Rendered items cached, invalidated on size change or item update
- **Operations:** SetItems, UpdateItem, AppendItem, ScrollBy, ScrollToIndex, ScrollToBottom
- **Interfaces:** Item (Render), RawRenderable, Focusable, Highlightable, RenderCallback
- **NOT yet used** in the main model (viewport still uses bubbles/viewport)

---

## 12. Missing Features vs Competitors

### vs Claude Code
| Feature | Claude Code | Poly-go | Status |
|---------|-------------|---------|--------|
| Markdown rendering (bold, headers, lists, code blocks) | Yes | **No** | Missing |
| Syntax highlighting in code blocks | Yes (via bat/chroma) | **No** | Missing |
| File content preview in tool results | Yes (with line numbers) | Partial (line count only) | Partial |
| Task/Todo system (inline) | Yes | Partial (sidebar todos from file) | Partial |
| Git integration | Yes (auto-commit, diff preview) | **No** | Missing |
| MCP server support | Yes | **No** (MCP bridge is AI-to-AI only) | Missing |
| Multi-turn tool use (agent loop) | Yes | Yes | Done |
| Image paste (Ctrl+V) | Yes | Yes | Done |
| Permission system | Yes (allow/deny/always) | Yes (Allow/Allow Session/Deny) | Done |
| Context window compaction | Yes | **No** | Missing |
| `/compact` command | Yes | **No** | Missing |
| Web search tool | Yes | Yes | Done |
| File glob/grep tools | Yes | Yes | Done |
| Cost tracking | Limited | Yes (per-session) | Done |
| Multi-model support | No (Claude only) | Yes (4+ providers) | Advantage |

### vs Crush (Codeium)
| Feature | Crush | Poly-go | Status |
|---------|-------|---------|--------|
| Inline diff preview | Yes (side-by-side) | **No** (line-by-line coloring only) | Missing |
| File tree navigation | Yes | **No** | Missing |
| Auto-complete suggestions | Yes | **No** | Missing |
| Undo/Redo for edits | Yes | **No** | Missing |
| Progress spinner on tools | Yes (animated) | Partial (static `⟳` icon) | Partial |
| Permission dialog | Yes | Yes (inspired by Crush) | Done |
| Collapsible tool outputs | Yes | Yes (auto-collapse completed) | Done |

### vs Gemini CLI
| Feature | Gemini CLI | Poly-go | Status |
|---------|-----------|---------|--------|
| Sandbox execution | Yes (gVisor) | **No** | Missing |
| Extensions/Plugins | Yes | **No** | Missing |
| Multi-turn conversation | Yes | Yes | Done |
| Session persistence | Yes | Yes | Done |
| Markdown rendering | Yes | **No** | Missing |

### Critical Missing Features (Priority Order)
1. **Markdown rendering** - All text appears as plain text, no formatting for headers, bold, code blocks, lists
2. **Syntax highlighting** - Code blocks show as plain monospace text
3. **Side-by-side diff view** - Only line-by-line coloring, no unified/split diff
4. **Context compaction** - No way to compress conversation when approaching context limits
5. **MCP (Model Context Protocol) server support** - Only has AI-to-AI bridge, not the standard MCP protocol
6. **Git integration** - No auto-commit, branch management, or diff staging
7. **File tree/navigator** - No way to browse project structure
8. **Animated spinners** - Static icons for running tools

### Unique Advantages of Poly-go
1. **Multi-model support** - Claude, GPT, Gemini, Grok + custom providers
2. **@all cascade** - Cheapest-first respond + review by others
3. **Custom provider system** - Add any OpenAI/Anthropic/Google-compatible API
4. **Thinking mode toggle** - Per-request thinking control
5. **YOLO mode** - Auto-approve all tool executions
6. **Session forking** - Branch conversations
7. **Per-session cost tracking** - Real-time cost display

---

## 13. Code Quality Notes

### Duplication
- `gradientText()` is implemented 3 times: header, sidebar, splash (each slightly different blending)
- `timeAgo()` is duplicated in session_list.go and components/dialogs/sessionlist/sessionlist.go
- Dialog rendering exists in BOTH legacy viewState system AND new component system
- `renderButton()` duplicated in approval.go and components/dialogs/approval/approval.go

### Architecture Observations
- The new component system (`components/dialogs/*`) is fully implemented but not wired into the main event loop
- The `list/` virtualized list is implemented but unused (viewport from bubbles is used instead)
- The `editor/` component is implemented but unused (inline textarea is used instead)
- The `messages/` component types (AssistantMessageItem, UserMessageItem) exist but rendering still uses inline `renderMessage()` in views.go
- `dialogMgr` receives messages in Update() but `handleDialogClosed()` returns no-op

### File Count
- **Root level:** 16 files (tui.go, model.go, update.go, views.go, streaming.go, diff_render.go, keys.go, commands.go, messages.go, approval.go, clipboard.go, sidebar.go, palette.go, modelpicker.go, session_list.go, dialogs.go)
- **Components:** 31 files across header, sidebar, status, editor, splash, messages, dialogs, core, styles, layout, list
- **Total:** ~47 Go files

---

*Report generated from full source code audit of all 47 TUI files.*
