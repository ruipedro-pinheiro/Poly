# Crush - Reverse Engineering Report

**Module**: `github.com/charmbracelet/crush`
**Language**: Go 1.25.5
**Files**: 321 `.go` files
**By**: Charmbracelet (makers of Bubble Tea, Lip Gloss, Glamour)

---

## Architecture

### Package Structure

```
crush/
  main.go                          # Entry point -> cmd.Execute()
  crush.json                       # Self-config (LSP gopls)
  schema.json                      # JSON schema for config
  sqlc.yaml                        # SQL code generation config
  internal/
    cmd/                           # CLI commands (Cobra)
    app/                           # Application wiring & lifecycle
    agent/                         # AI agent orchestration (core)
      hyper/                       # Hyper provider (Charm's meta-provider)
      prompt/                      # System prompt template engine
      templates/                   # Prompt templates (coder, task, agent, fetch)
      tools/                       # All 19+ built-in tools
        mcp/                       # MCP client integration
    config/                        # Configuration loading & resolution
    commands/                      # Slash commands / custom commands
    db/                            # SQLite via sqlc + goose migrations
    session/                       # Session management (CRUD)
    message/                       # Message model & service
    permission/                    # Permission request/grant system
    shell/                         # POSIX shell emulator (mvdan.cc/sh)
    lsp/                           # LSP client (via powernap)
    skills/                        # Agent Skills standard (SKILL.md)
    tui/                           # OLD TUI (Bubble Tea v2)
      components/                  # Chat, sidebar, dialogs, etc.
      exp/                         # Experimental: diffview, list
      page/                        # Page-level models (chat)
      styles/                      # Theme system (charmtone)
    ui/                            # NEW TUI (behind CRUSH_NEW_UI flag)
      model/                       # New UI model
      chat/                        # Chat renderers
      dialog/                      # Dialog components
      styles/                      # New style system
    pubsub/                        # Generic pub/sub broker
    csync/                         # Concurrent data structures (Map, Slice, Value, VersionedMap)
    diff/                          # Diff computation
    event/                         # Telemetry / PostHog events
    history/                       # File history service
    filetracker/                   # Read file tracking per session
    update/                        # Version update checker
    version/                       # Version constant
    home/                          # Home directory resolution
    env/                           # Environment variable wrapper
    fsext/                         # Filesystem utilities (ls, ignore, paste)
    filepathext/                   # Smart filepath joining
    stringext/                     # String utilities
    ansiext/                       # ANSI utilities
    format/                        # Spinner for non-interactive mode
    log/                           # HTTP debug logging
    uicmd/                         # UI commands
    uiutil/                        # UI utilities
```

### Entry Point & Initialization Flow

1. **`main.go`**: Calls `cmd.Execute()`. Optional pprof on `CRUSH_PROFILE` env.

2. **`cmd/root.go`** (`Execute()`):
   - Uses `github.com/charmbracelet/fang` (wrapper around Cobra)
   - Calls `setupAppWithProgressBar()` -> `setupApp()`
   - `setupApp()` flow:
     a. Resolve CWD
     b. `config.Init()` loads config from multiple files
     c. Register project in projects list
     d. `db.Connect()` opens SQLite + runs goose migrations
     e. `app.New()` wires services together

3. **`app/app.go`** (`New()`):
   - Creates session, message, history, permission, filetracker services
   - Sets up event subscriptions (sessions, messages, permissions, MCP, LSP)
   - Initializes LSP clients in background
   - Initializes MCP clients in background (`mcp.Initialize()`)
   - Calls `InitCoderAgent()` -> `agent.NewCoordinator()`

4. **TUI Launch** (back in `cmd/root.go`):
   - Two UIs: old (`tui.New()`) or new (`ui.New()` via `CRUSH_NEW_UI=1`)
   - Creates `tea.Program` with Bubble Tea v2
   - `app.Subscribe(program)` bridges service events to TUI messages

### Dependency Injection Pattern

Services are created in `app.New()` and passed down:
- `session.Service` -> sessions CRUD
- `message.Service` -> messages CRUD
- `permission.Service` -> permission requests with pub/sub
- `history.Service` -> file version history
- `filetracker.Service` -> tracks which files were read per session
- All wrapped in `App` struct, passed to TUI and agent coordinator

The `Coordinator` receives all services and passes them to tools at construction time. Each tool closure captures the services it needs.

---

## Provider System

### Provider Abstraction via Fantasy

Crush uses **`charm.land/fantasy`**, Charmbracelet's own multi-provider LLM SDK. Fantasy abstracts:
- `fantasy.Provider` - creates `LanguageModel` instances
- `fantasy.LanguageModel` - the actual model to query
- `fantasy.Agent` - orchestrates tool loops with streaming
- `fantasy.AgentTool` - tool interface for the agent

### Supported Providers

Built in `coordinator.buildProvider()` (`internal/agent/coordinator.go:726`):

| Provider Type | SDK Used | Build Function |
|---|---|---|
| `openai` | `fantasy/providers/openai` | `buildOpenaiProvider()` - Uses Responses API |
| `anthropic` | `fantasy/providers/anthropic` | `buildAnthropicProvider()` - X-Api-Key or Bearer |
| `openrouter` | `fantasy/providers/openrouter` | `buildOpenrouterProvider()` |
| `azure` | `fantasy/providers/azure` | `buildAzureProvider()` - with API version |
| `bedrock` | `fantasy/providers/bedrock` | `buildBedrockProvider()` - AWS auth |
| `google` | `fantasy/providers/google` | `buildGoogleProvider()` - Gemini API key |
| `google-vertex` | `fantasy/providers/google` | `buildGoogleVertexProvider()` - project/location |
| `openai-compat` | `fantasy/providers/openaicompat` | `buildOpenaiCompatProvider()` - for Copilot, z.ai, etc. |
| `hyper` | `internal/agent/hyper` | `buildHyperProvider()` - Charm's meta-provider |

### Streaming Implementation

Via `fantasy.Agent.Stream()` in `agent.go:241`. Callbacks:
- `OnReasoningStart/Delta/End` - reasoning/thinking content
- `OnTextDelta` - streaming text
- `OnToolInputStart` - tool input streaming
- `OnToolCall` - tool execution completed
- `OnToolResult` - tool result received
- `OnStepFinish` - step completed with usage data
- `OnRetry` - retry on provider error (TODO)
- `PrepareStep` - called before each agent step to prepare messages
- `StopWhen` - stop conditions (context window threshold for auto-summarize)

### Auth: OAuth Flow, API Keys, Token Refresh

**API Keys**: Resolved via `config.Resolve()` which uses `VariableResolver`. Supports `$ENV_VAR` syntax.

**OAuth**: Used for Copilot and Hyper:
- `internal/oauth/copilot/` - GitHub Copilot device flow OAuth
- `internal/oauth/hyper/` - Charm Hyper device flow OAuth
- Token refresh on 401: `coordinator.refreshOAuth2Token()` (`coordinator.go:851`)
- API key template re-resolution on 401: `coordinator.refreshApiKeyTemplate()` (`coordinator.go:862`)

**Copilot special handling**: Custom HTTP client in `copilot.NewClient()`, custom headers via `SetupGitHubCopilot()`.

### Model Registry & Model Switching

**Catwalk** (`github.com/charmbracelet/catwalk`): Remote model registry at `https://catwalk.charm.sh`. Loaded in `config/load.go` -> `Providers()`. Contains model definitions with context windows, costs, capabilities, etc.

**Model types**: `large` and `small`. Large for main generation, small for title generation.

**Recent models**: Tracked in config via `recordRecentModel()`, up to 5 per type.

**Dynamic model switching**: `coordinator.UpdateModels()` rebuilds models before each run. TUI has model picker dialog.

### Provider Options & Reasoning

Provider-specific options are merged from 3 layers (`getProviderOptions()` in `coordinator.go:198`):
1. Catwalk model defaults
2. Provider config
3. User model config

Reasoning support:
- **Anthropic**: `thinking` with `budget_tokens`
- **OpenAI**: `reasoning_effort` (low/medium/high), Responses API with encrypted reasoning
- **OpenRouter**: `reasoning.effort`
- **Google**: `thinking_config` with `thinking_budget`

---

## Tool System

### Tool Interface

All tools implement `fantasy.AgentTool`. Two constructors:
- `fantasy.NewAgentTool()` - sequential execution
- `fantasy.NewParallelAgentTool()` - can run in parallel

Each tool is a closure that captures services at construction time. Tool descriptions are embedded via `//go:embed` from `.md` or `.tpl` files.

### Registration Pattern

Tools are registered in `coordinator.buildTools()` (`coordinator.go:369`):
1. Conditional tools first (agent, agentic_fetch)
2. Core tools added directly
3. LSP tools added if LSP is configured
4. MCP tools added dynamically
5. Filtered by `agent.AllowedTools`
6. MCP tools filtered by `agent.AllowedMCP`
7. Sorted alphabetically

### Execution Flow (Sandboxing & Permissions)

1. Tool called by Fantasy agent loop
2. Tool checks parameters
3. Non-safe tools request permission via `permissions.Request()`
4. Permission service publishes request to TUI
5. TUI shows permission dialog to user
6. User grants/denies
7. Permission service returns result
8. Tool executes

**Safe commands** (no permission needed): defined in `safe.go` - git read operations, system info commands (ls, pwd, whoami, etc.)

**Banned commands** (blocked at shell level): Network tools (curl, wget, ssh), system admin (sudo, su), package managers (apt, dnf, brew), system modification (systemctl, crontab, mount).

### Complete Tool List (19 Built-in Tools)

| # | Tool Name | File | Description |
|---|---|---|---|
| 1 | `agent` | `agent_tool.go` | Sub-agent for search tasks. Read-only tools (glob, grep, ls, view). Runs in separate session. |
| 2 | `agentic_fetch` | `agentic_fetch_tool.go` | Sub-agent for web fetching. Has web_fetch + web_search tools. |
| 3 | `bash` | `bash.go` | Shell execution. POSIX emulation via `mvdan.cc/sh`. Auto-backgrounds after 1 minute. Permission required for non-safe commands. Max output 30000 chars. |
| 4 | `job_output` | `job_output.go` | Read output from background shell jobs. |
| 5 | `job_kill` | `job_kill.go` | Kill background shell jobs. |
| 6 | `download` | `download.go` | Download files from URLs. Permission required. Max 5 minute timeout. |
| 7 | `edit` | `edit.go` | Edit files with exact string matching. Permission required. Creates diff, tracks file history, notifies LSP. |
| 8 | `multiedit` | `multiedit.go` | Multiple edits in a single file atomically. Same permission/history/LSP as edit. |
| 9 | `view` | `view.go` | Read files. Max 5MB, 2000 lines default. Line numbers. Binary detection. Permission for files outside working dir. Tracks reads in filetracker. |
| 10 | `write` | `write.go` | Write/create files. Permission required. Creates diff, tracks file history, notifies LSP. Checks file modification time vs last read. |
| 11 | `glob` | `glob.go` | File pattern matching via `doublestar`. Uses ripgrep if available. |
| 12 | `grep` | `grep.go` | Content search. Uses ripgrep (`rg`) if available, falls back to Go implementation. Regex support, context lines, file filtering. |
| 13 | `ls` | `ls.go` | Directory listing with depth/items limits. Permission for dirs outside working dir. |
| 14 | `fetch` | `fetch.go` | Fetch web pages. Converts HTML to markdown. Permission required. Supports text/markdown/html formats. |
| 15 | `sourcegraph` | `sourcegraph.go` | Search Sourcegraph.com for code. No authentication needed (public search). |
| 16 | `todos` | `todos.go` | Manage todo lists per session. Stored in session as JSON. Status: pending/in_progress/completed. |
| 17 | `lsp_diagnostics` | `diagnostics.go` | Get LSP diagnostics for file or project. |
| 18 | `lsp_references` | `references.go` | Find references to a symbol via LSP. |
| 19 | `lsp_restart` | `lsp_restart.go` | Restart LSP clients. |

**Sub-agent tools** (not directly exposed):
- `web_fetch` (`web_fetch.go`) - Simple URL fetch for agentic_fetch sub-agent
- `web_search` (`web_search.go`) - DuckDuckGo search for agentic_fetch sub-agent

### Shell Execution Details

`internal/shell/shell.go`: Uses `mvdan.cc/sh/v3` for POSIX shell emulation (works cross-platform including Windows). Features:
- Built-in coreutils via `mvdan.cc/sh/moreinterp/coreutils`
- Command blocking via `BlockFunc` pattern
- Working directory tracking
- Background job management via `BackgroundShellManager` (`background.go`)
- Auto-background threshold: 1 minute

---

## TUI (Bubble Tea)

### Model/Update/View Pattern

Two TUI implementations:

**Old TUI** (`internal/tui/tui.go`):
- `appModel` struct with pages, dialogs, status
- Standard Bubble Tea `Init()`, `Update()`, `View()` pattern
- Mouse event filtering to throttle trackpad (15ms debounce)

**New TUI** (`internal/ui/model/ui.go`, behind `CRUSH_NEW_UI=1`):
- Uses `common.Common` base struct for shared state
- More modular component architecture

### Layout System (Lip Gloss v2)

Layout managed via `internal/tui/components/core/layout/layout.go`:
- Responsive to window size
- Components calculate their own dimensions
- Lip Gloss v2 (beta) for styling, borders, positioning

### Components

| Component | Path | Purpose |
|---|---|---|
| Chat | `tui/components/chat/chat.go` | Main chat area with messages |
| Editor | `tui/components/chat/editor/` | Input editor with clipboard support |
| Header | `tui/components/chat/header/` | Session title, model info |
| Messages | `tui/components/chat/messages/` | Message rendering, tool output rendering |
| Sidebar | `tui/components/chat/sidebar/` | Session list sidebar |
| Splash | `tui/components/chat/splash/` | Landing screen |
| Todos | `tui/components/chat/todos/` | Todo list display |
| Completions | `tui/components/completions/` | File/command completions |
| Status | `tui/components/core/status/` | Status bar |
| Anim | `tui/components/anim/` | Animation component |
| Logo | `tui/components/logo/` | ASCII logo with randomization |
| Image | `tui/components/image/` | Terminal image rendering |
| Files | `tui/components/files/` | File browser component |
| LSP | `tui/components/lsp/` | LSP status display |
| MCP | `tui/components/mcp/` | MCP status display |

### Dialogs

| Dialog | Path | Purpose |
|---|---|---|
| Commands | `tui/components/dialogs/commands/` | Slash command picker |
| Models | `tui/components/dialogs/models/` | Model selector with API key input |
| Sessions | `tui/components/dialogs/sessions/` | Session browser |
| Permissions | `tui/components/dialogs/permissions/` | Permission grant/deny |
| Quit | `tui/components/dialogs/quit/` | Quit confirmation |
| File Picker | `tui/components/dialogs/filepicker/` | File picker dialog |
| Copilot | `tui/components/dialogs/copilot/` | GitHub Copilot OAuth device flow |
| Hyper | `tui/components/dialogs/hyper/` | Charm Hyper OAuth device flow |
| Reasoning | `tui/components/dialogs/reasoning/` | Reasoning/thinking display |

### Keyboard Shortcuts

From `tui/keys.go`:
- `ctrl+c` - Quit
- `ctrl+g` - Help/more
- `ctrl+p` - Commands (slash command palette)
- `ctrl+z` - Suspend
- `ctrl+l` / `ctrl+m` - Models
- `ctrl+s` - Sessions

### Theme System

`tui/styles/theme.go`: Full theme struct with:
- Primary, Secondary, Tertiary, Accent colors
- Background layers: Base, Lighter, Subtle, Overlay
- Foreground: Base, Muted, HalfMuted, Subtle, Selected
- Semantic: Success, Error, Warning, Info
- Built on `charmtone` color palette
- Light/dark theme detection
- Markdown styling via Glamour v2
- Diff view styling via custom `diffview.Style`

---

## MCP Integration

### MCP Client Implementation

`internal/agent/tools/mcp/init.go`: Uses `github.com/modelcontextprotocol/go-sdk` (official MCP Go SDK v1.2.0).

**Initialization** (`Initialize()`, line 137):
1. Iterates over `cfg.MCP` config
2. Skips disabled MCPs
3. Sets state to `StateStarting`
4. Creates transport (stdio, SSE, or HTTP)
5. Creates `mcp.Client` with handlers for tool/prompt list changes
6. Connects via `client.Connect()`
7. Lists tools and prompts
8. Updates global state

**States**: `StateDisabled`, `StateStarting`, `StateConnected`, `StateError`

### Tool Discovery

`internal/agent/tools/mcp/tools.go`:
- Tools stored in global `csync.Map[string, []*Tool]`
- `getTools()` calls `session.ListTools()`
- `RefreshTools()` for dynamic updates
- `filterDisabledTools()` respects `disabled_tools` config
- Tool naming: `mcp_{serverName}_{toolName}`

### Protocol Handling

Three transport types supported (`createTransport()`, init.go:354):

1. **stdio** (`mcp.CommandTransport`): Executes command with args, env vars. Most common.
2. **SSE** (`mcp.SSEClientTransport`): Server-Sent Events. HTTP endpoint with custom headers.
3. **HTTP** (`mcp.StreamableClientTransport`): Streamable HTTP. Custom headers via `headerRoundTripper`.

**Auto-reconnect**: `getOrRenewClient()` pings the server, reconnects on failure.

**Error handling**: `maybeStdioErr()` detects when stdio MCP crashes on startup and captures stderr for better error messages.

---

## Sub-agents

### Agent Tool (`agent_tool.go`)

The `agent` tool spawns a sub-agent with read-only tools:
- `glob`, `grep`, `ls`, `view`, `sourcegraph`
- No bash, edit, write, or MCP tools
- Uses task prompt template instead of coder prompt
- Creates a child session linked to parent via `CreateTaskSession()`
- Costs are rolled up to parent session
- Uses `fantasy.NewParallelAgentTool()` - can run multiple in parallel

### Agentic Fetch (`agentic_fetch_tool.go`)

The `agentic_fetch` tool spawns a sub-agent for web research:
- `web_fetch` - fetch URLs
- `web_search` - DuckDuckGo search
- `view` - read local files
- Has its own prompt template (`agentic_fetch_prompt.md.tpl`)
- Also creates child session, costs rolled up

### Communication Pattern

Sub-agents are fully independent:
- Separate `SessionAgent` instance
- Own session in DB
- No inter-agent messaging
- Return final text result to parent agent
- Cost aggregation via parent session update

---

## LSP Integration

### LSP Client (`internal/lsp/client.go`)

Built on `github.com/charmbracelet/x/powernap` (Charmbracelet's LSP library).

**Client struct features**:
- Diagnostic caching with versioned map (`csync.VersionedMap`)
- Open file tracking
- Auto-detect file types
- Server state tracking

### Connection

`app/lsp.go` initializes LSP clients:
- Based on `crush.json` or auto-detected via root markers
- Auto-LSP detection: looks for `go.mod` -> gopls, `package.json` -> typescript-language-server, etc. (`config/lsp_defaults_test.go`)
- Spawns command with args and env

### LSP Usage

Three tools use LSP:
1. **`lsp_diagnostics`**: Gets errors/warnings for files
2. **`lsp_references`**: Finds all references to a symbol
3. **`lsp_restart`**: Restarts LSP servers

LSP is also used by `edit`, `multiedit`, and `write` tools to notify the server of file changes (`textDocument/didOpen`, `textDocument/didChange`, `textDocument/didSave`).

### LSP Events

`app/lsp_events.go`: LSP diagnostic changes are published as events to the TUI for real-time display.

---

## Diff Viewer

### Implementation

`tui/exp/diffview/diffview.go`:

Two layouts:
- **Unified** (`layoutUnified`): Traditional unified diff
- **Split** (`layoutSplit`): Side-by-side diff

Features:
- Syntax highlighting via `alecthomas/chroma/v2`
- Diff computation via `aymanbagabas/go-udiff`
- Line numbers
- Scrollable (x and y offset)
- Custom styling (`diffview/style.go`)
- Tab width configuration
- Context lines control
- xxHash for content comparison (`zeebo/xxh3`)

### Where Diffs Are Used

- `edit` tool: shows old vs new content
- `multiedit` tool: shows old vs new content
- `write` tool: shows diff of existing vs new file
- TUI message renderer: renders diffs inline in chat

---

## Config System

### Configuration Loading

`config/load.go` - Multi-file config system:

**Config file locations** (merged in order):
1. `~/.config/crush/crush.json` (global)
2. `.crush/config.json` (project data dir)
3. `crush.json` in working directory (project)

**Context files** (`defaultContextPaths` in `config.go:35`):
- `.github/copilot-instructions.md`
- `.cursorrules`, `.cursor/rules/`
- `CLAUDE.md`, `CLAUDE.local.md`
- `GEMINI.md`, `gemini.md`
- `crush.md`, `crush.local.md`, `CRUSH.md`, `CRUSH.local.md`
- `AGENTS.md`, `agents.md`, `Agents.md`

These are loaded as context in the system prompt.

### Config Format (JSON)

```json
{
  "$schema": "https://charm.land/crush.json",
  "models": {
    "large": { "model": "...", "provider": "..." },
    "small": { "model": "...", "provider": "..." }
  },
  "providers": { ... },
  "mcp": { ... },
  "lsp": { ... },
  "options": { ... },
  "permissions": { ... },
  "tools": { ... }
}
```

### Variable Resolution

`config/resolve.go`: `VariableResolver` interface with `ShellVariableResolver`:
- Resolves `$ENV_VAR` from environment
- Resolves `$(command)` via shell execution
- Used for API keys, base URLs, etc.

### Runtime Config Changes

- `SetConfigField()` / `RemoveConfigField()` - JSON manipulation via `tidwall/sjson`
- `HasConfigField()` - reads via `tidwall/gjson`
- Config file re-read on change
- Model preferences persisted in data dir config

### Catwalk Integration

`config/catwalk.go`: Fetches provider/model definitions from `https://catwalk.charm.sh`:
- Cached via `charmbracelet/x/etag` (ETag-based HTTP caching)
- Fallback to embedded defaults if offline
- Auto-update with `disable_provider_auto_update` option

---

## Session Management

### Storage: SQLite

`internal/db/`:
- Database: `{dataDir}/crush.db`
- Migration tool: `pressly/goose/v3` with embedded SQL migrations
- Query generation: `sqlc` (see `sqlc.yaml`)
- Two SQLite drivers: `ncruces/go-sqlite3` (WASM-based) and `modernc.org/sqlite` (pure Go)

**Tables** (from migrations):
1. `sessions` - id, title, parent_session_id, message_count, prompt_tokens, completion_tokens, summary_message_id, cost, todos (JSON), created_at, updated_at
2. `messages` - id, session_id, role, parts (JSON), model, provider, is_summary_message, created_at
3. `files` - file version history for undo
4. `read_files` - tracks file reads per session

### Session Service

`internal/session/session.go`:
- Create, Get, List, Save, Delete operations
- Agent tool session management (`CreateAgentToolSessionID`, `IsAgentToolSession`)
- Todo management stored as JSON in session
- Pub/sub for session events

### Context Management

**Auto-summarization** (`agent.go:529`):
- Triggered when token count approaches context window limit
- Large context (>200k): 20k token buffer
- Small context: 20% ratio
- Creates a summary message using the same model
- Sets `SummaryMessageID` on session
- Subsequent messages start from summary
- If agent was mid-tool-use, re-queues the original prompt

**Conversation History**:
- Messages stored in DB
- Loaded per session via `messages.List()`
- Converted to `fantasy.Message` format
- Anthropic cache control markers on system prompt and last 2 messages

---

## Agent Skills System

`internal/skills/skills.go`: Implements the Agent Skills open standard (https://agentskills.io).

**SKILL.md format**: YAML front matter + markdown instructions:
```yaml
---
name: my-skill
description: What this skill does
---
Instructions for the agent...
```

**Discovery**: Searches configured `skills_paths` for directories containing `SKILL.md` files.

**Integration**: Available skills are serialized as XML and injected into the system prompt.

---

## Event/Telemetry System

`internal/event/`: PostHog-based telemetry:
- `event.Init()` - initializes PostHog client
- Events: AppInitialized, AppExited, SessionCreated, PromptSent, PromptResponded, TokensUsed, Error
- Machine ID via `denisbrodbeck/machineid`
- Disable via `CRUSH_DISABLE_METRICS=1`, `DO_NOT_TRACK=1`, or `options.disable_metrics`

---

## Pub/Sub System

`internal/pubsub/broker.go`: Generic typed pub/sub:
- `Broker[T]` with `Publish()` and `Subscribe()` methods
- `Event[T]` with `EventType` (Created, Updated, Deleted)
- Used by: sessions, messages, permissions, MCP, LSP
- Channels with context-based cancellation

---

## Key Dependencies

| Dependency | Purpose |
|---|---|
| `charm.land/fantasy` | Multi-provider LLM SDK (agents, tools, streaming) |
| `charm.land/bubbletea/v2` | TUI framework |
| `charm.land/lipgloss/v2` | TUI styling |
| `charm.land/glamour/v2` | Markdown rendering |
| `charm.land/bubbles/v2` | TUI components (textarea, key bindings, etc.) |
| `charmbracelet/catwalk` | Remote model registry |
| `charmbracelet/x/powernap` | LSP client library |
| `charmbracelet/fang` | Cobra wrapper with signal handling |
| `charmbracelet/ultraviolet` | Environment variable handling |
| `modelcontextprotocol/go-sdk` | Official MCP SDK |
| `spf13/cobra` | CLI framework |
| `ncruces/go-sqlite3` | SQLite (WASM) |
| `modernc.org/sqlite` | SQLite (pure Go) |
| `pressly/goose/v3` | DB migrations |
| `alecthomas/chroma/v2` | Syntax highlighting |
| `aymanbagabas/go-udiff` | Unified diff |
| `mvdan.cc/sh/v3` | POSIX shell emulator |
| `openai/openai-go/v2` | OpenAI SDK |
| `posthog/posthog-go` | Telemetry |
| `JohannesKaufmann/html-to-markdown` | HTML to markdown conversion |

---

## Notable Design Decisions

1. **POSIX shell emulator**: Instead of running real bash, Crush uses `mvdan.cc/sh` for cross-platform POSIX emulation. This means it works on Windows without WSL.

2. **Dual TUI**: Old and new TUI coexist behind a feature flag (`CRUSH_NEW_UI`). The old one is the default.

3. **Fantasy SDK**: Charmbracelet built their own LLM SDK rather than using LangChain-style libraries. It's tightly integrated with Bubble Tea's message passing.

4. **Catwalk remote registry**: Model definitions are fetched from a remote registry, cached locally with ETags. This allows updating model info without app updates.

5. **Hyper meta-provider**: Charm's own proxy service that handles model routing. Optional, enabled via env var.

6. **Auto-summarization**: When context window runs low, automatically summarizes the conversation and continues. The summarized session continues from the summary point.

7. **Agent tool sessions**: Sub-agent runs create child sessions in the DB. Costs aggregate up to the parent.

8. **File tracking**: The `filetracker` service records when files were read per session. The `write` tool checks if the file was modified since last read to warn about stale edits.

9. **Permission system**: Pub/sub based. The agent blocks waiting for user approval via channels. Auto-approve mode via `--yolo` flag or per-session for non-interactive mode.

10. **Agent Skills standard**: Crush implements the open `agentskills.io` spec for reusable agent instructions, making skills portable across different AI coding tools.
