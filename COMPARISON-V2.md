# Poly-go Comparison V2 - Full Research Report

> Compiled from 4 deep research agents + 2 reverse engineering agents
> V1: 2026-02-10 | V2 Update: 2026-02-10 (integrated claude-code-v2.md + gemini-cli-v2.md findings)

---

## Executive Summary

| Metric | Claude Code | Crush | OpenCode | Gemini CLI | **Poly-go** |
|--------|------------|-------|----------|-----------|-------------|
| **Runtime** | **Bun** (JSC, NOT Node.js) | Go | Bun | Node.js | **Go** |
| Language | TypeScript (compiled to ELF) | Go | TypeScript | TypeScript | **Go** |
| Files | ~500+ (minified in 213MB ELF) | 321 | 1006 | **1365** (cli 812 + core 553) | **94** |
| Providers | 1 (Anthropic) | 9 | 20+ | 1 (Google) | **4 built-in + unlimited custom** |
| Tools | 20+ | 19 | 17+ | **14+** | **15** (+ MCP tools auto-registered) |
| MCP | Full client | Full (Go SDK) | Full + OAuth | Full + **OAuth PKCE** | **Client + Manager** (stdio, auto-register, namespacing) |
| Agents/Teams | Full teams + sub-agents | 2 sub-agents | 7 agents + task | Local + **A2A protocol** | **None** |
| Hooks | 14 events, **3 types** (cmd/prompt/agent) | None | 15+ plugin hooks | **11 events** | **Pub/sub broker** (events system) |
| Skills | SKILL.md + **fork context** | agentskills.io | .claude/ compat | SKILL.md | **None** |
| Permissions | 6 modes, rules | Pub/sub dialog | Rules + glob | **Policy TOML 3 tiers** | **Allow/Ask/Deny + YoloMode + auto-allow** |
| TUI | Ink (React) | Bubble Tea v2 | SolidJS (@opentui) | React 19/Ink 6 | **Bubble Tea v2** |
| **Themes** | 1 (fixed) | 1 + Catppuccin | 1 | **14 built-in + custom** | **Catppuccin Mocha** |
| **Non-interactive** | `--print` + **stream-json** | None | None | JSON + stream-json | **None** |
| **Sandbox** | None | None | None | **Docker/Podman/Seatbelt** | **None** |
| **Plugins/Extensions** | **Plugin registry** | None | **15+ plugin hooks** | **Full extension system** | **None** |
| Storage | JSONL files | SQLite | JSON files + locks | JSON sessions | **JSON files** (multi-session, index, auto-migrate) |
| Config | CLAUDE.md + settings | JSON + Catwalk | JSONC 7 levels | JSON **5 levels + MergeStrategy** | **JSON** (defaults + user merge, provider colors) |
| Context mgmt | Compaction + **MEMORY.md** | Auto-summarize | Pruning + compaction | GEMINI.md + **/compress** | **None** (no compaction yet) |
| **Prompt caching** | **Ephemeral 5m/1h** | None | None | None | **None** |
| **Interactive Shell** | None | None | None | None | **Yes** (`-i` flag: AI pipes, variables, history) |
| **@mentions** | None | None | None | `@agent`, `@file` | **@claude @gpt @gemini @grok @all** (cascade) |
| **Cascade Mode** | None | None | None | None | **@all** (cheapest-first + parallel reviewers) |
| **OAuth/PKCE** | None | None | OAuth | OAuth | **OAuth** (Anthropic, OpenAI, Google) + **PKCE** |
| **Diff Propose/Review** | None | None | None | None | **propose_diff/apply_diff/reject_diff/list_diffs** |
| **Pricing/Cost** | Yes | No | No | No | **Yes** (per-model pricing table, session cost tracking) |
| **Image support** | Yes | No | No | Yes | **Yes** (multi-provider, auto-detection, paste) |
| **Surfaces** | Terminal, VS Code, JetBrains, Desktop, Web, iOS | Terminal | Terminal + Web | Terminal + **VS Code companion** | **Terminal + Interactive Shell** |
| Open Source | No | No | MIT | Apache 2.0 | **Yes** |

**Poly-go is at ~30-35% feature parity with the competition.** Notably strong in: multi-provider support, @all cascade orchestration, interactive shell, diff propose/review workflow, OAuth/PKCE, and TUI dialogs.

---

## 1. Architecture Comparison

### Entry Point & Lifecycle

| Tool | Pattern |
|------|---------|
| Claude Code | **Bun ELF binary** (213MB) -> CLI args -> enterprise MCP check -> tool permissions -> MCP configs -> input parse -> tool loading -> TUI (React/Ink) or **non-interactive mode** |
| Crush | Cobra CLI -> `app.New()` wires services -> `agent.NewCoordinator()` -> Bubble Tea |
| OpenCode | Yargs CLI -> Bootstrap (project discovery, config, storage, plugins) -> Server (Hono) -> TUI |
| Gemini CLI | Yargs -> **Sandbox detection** -> Settings (5 levels) -> **Memory relaunch** (50% RAM via V8 heap) -> Auth refresh -> React 19/Ink 6 `render()` or **non-interactive mode** |
| **Poly-go** | `main.go` -> `config.Load()` (defaults + user merge) -> `tools.Init()` (15 tools) -> `llm.LoadCustomProviders()` -> Bubble Tea TUI or Interactive Shell (`-i` flag) |

### Key Architectural Discoveries (V2)

**Claude Code - Bun Runtime (NOT Node.js)**:
- Binary analysis reveals JavaScriptCore (JSC) symbols, NOT V8
- 213MB ELF with embedded JS bundle (9119+ function refs)
- Source paths in binary: `src/entrypoints/cli.js`, `src/utils/bash/parser.ts`, etc.
- **AsyncLocalStorage** (`async_hooks`) for team context propagation across async calls

**Gemini CLI - Self-Relaunching Architecture**:
- Parent process detects memory, configures V8 args, launches child
- Child gets `--max-old-space-size` = 50% total RAM
- `GEMINI_CLI_NO_RELAUNCH` prevents infinite relaunch loop
- IPC channel for admin settings from parent to child

### What Poly Has
- **Initialization chain**: `config.Load()` -> `tools.Init()` -> `llm.LoadCustomProviders()` -> TUI or Shell
- **Dual mode**: Full Bubble Tea TUI (default) or Interactive Shell (`-i` flag with readline, pipes, variables)
- **Context propagation**: Uses `context.Context` for streaming cancellation

### What Poly Still Needs
- **Service layer** like Crush's `app.New()` pattern: formalized dependency injection
- **Agent coordinator** that owns the agentic loop
- **Server component** like OpenCode's Hono (enables web UI, SDK, remote access later)

---

## 2. Provider System

### Provider Count

| Tool | Providers | SDK |
|------|-----------|-----|
| OpenCode | 20+ (Anthropic, OpenAI, Google, Azure, Bedrock, xAI, Mistral, Groq, etc.) | Vercel AI SDK v5 |
| Crush | 9 (OpenAI, Anthropic, OpenRouter, Azure, Bedrock, Google, Vertex, OpenAI-compat, Hyper) | charm.land/fantasy |
| Claude Code | 1 (Anthropic) | Direct API |
| Gemini CLI | 1 (Google Gemini) | @google/genai |
| **Poly-go** | **4 built-in (Anthropic, OpenAI, Gemini, xAI) + unlimited custom** (`/addprovider`) | **Custom per-provider** (anthropic/openai/google formats) |

### Auth Patterns

| Tool | Methods |
|------|---------|
| Claude Code | Anthropic API key, Max plan |
| Crush | API keys (env vars), OAuth (Copilot, Hyper), token refresh on 401 |
| OpenCode | API keys, OAuth (Codex, Copilot, GitLab), well-known discovery, plugin auth |
| Gemini CLI | Google OAuth (free tier), API key, Vertex AI, Service Account |
| **Poly-go** | **API keys + OAuth** (Anthropic OAuth code exchange, OpenAI OAuth callback, Google OAuth callback) + **PKCE** (`auth/pkce.go`) |

### Streaming

| Tool | Pattern |
|------|---------|
| Crush | Fantasy `Agent.Stream()` with callbacks (OnTextDelta, OnToolCall, etc.) |
| OpenCode | Vercel AI SDK `streamText()` with middleware |
| Gemini CLI | @google/genai native streaming |
| **Poly-go** | **Custom per-provider streaming** (SSE parsing for all 3 formats: Anthropic, OpenAI, Google + thinking/reasoning support) |

### Model Registry

| Tool | Pattern |
|------|---------|
| Crush | **Catwalk** - remote registry at catwalk.charm.sh with ETag caching |
| OpenCode | **models.dev** - centralized model registry with Zod schemas |
| Claude Code | Hardcoded (3 models: Opus, Sonnet, Haiku) |
| Gemini CLI | Hardcoded + auto model |
| **Poly-go** | **Config-driven** (16+ default models across 4 providers, dynamic from `config.json`, model variants: default/fast/think/opus/pro/nano/lite/mini) |

### What Poly Has
- **4 built-in providers** with full streaming support (Anthropic, OpenAI, Google, xAI)
- **Custom providers** via `/addprovider` command (supports any OpenAI/Anthropic/Google-compatible API)
- **OAuth flows** for Anthropic (code exchange), OpenAI (callback), Google (callback) + **PKCE** generation
- **Model variants** per provider (default, fast, think, opus, etc.) - config-driven, not hardcoded
- **Pricing table** with per-model cost calculation
- **Image support** with auto-detection per provider
- **Thinking mode** support across all 3 API formats

### What Poly Still Needs (Priority: MEDIUM)
1. **OpenAI-compatible endpoint** as a generic provider type (covers 80% more providers)
2. **Token refresh on 401** like Crush
3. **Remote model registry** or auto-discovery

---

## 3. Tool System

### Tool Count & Coverage

| Tool | Count | Notable Extras |
|------|-------|---------------|
| Claude Code | 20+ | WebSearch, WebFetch, NotebookEdit, AskUserQuestion, Skill, TodoRead/Write, **EnterPlanMode, ExitPlanMode, TeamCreate, TeamDelete, SendMessage, TaskCreate/Update/List/Get** |
| Crush | 19 | Sourcegraph search, download, LSP diagnostics/references/restart |
| OpenCode | 17+ | CodeSearch (Exa), batch, apply_patch, LSP, plan enter/exit |
| Gemini CLI | **14+** | google_web_search (grounding), save_memory, activate_skill, **read_many_files, ask_user, write_todos, exit_plan_mode** |
| **Poly-go** | **15** | **bash, read_file, list_files, write_file, edit_file, glob, grep, multiedit, web_fetch, web_search, todos, propose_diff, apply_diff, reject_diff, list_diffs** (+ MCP tools auto-registered with namespacing) |

### Tool Definition Patterns

| Tool | Pattern |
|------|---------|
| Crush | `fantasy.AgentTool` closures capturing services. Descriptions via `//go:embed` |
| OpenCode | `Tool.define(id, init)` with Zod schema validation |
| Gemini CLI | **`BaseDeclarativeTool<TParams, TResult>` -> `ToolInvocation` class hierarchy** with `Kind` enum (Read, Edit, Delete, Move, Search, Execute, Think, Fetch, Communicate, Plan, Other) |
| Claude Code | Internal tool definitions in system prompt (injected dynamically per tool) |
| **Poly-go** | **`Tool` interface** (`Name()`, `Description()`, `Parameters()` JSON Schema, `Execute()`) with thread-safe registry (`sync.RWMutex`), approval channel system, and auto-registration of MCP tools |

### Edit Tool Strategies (NEW from V2)

| Tool | Strategy |
|------|----------|
| Claude Code | Exact string replacement only (`old_string` must be unique) |
| Crush | Exact replacement |
| OpenCode | `apply_patch` with unified diff |
| Gemini CLI | **3 strategies in cascade**: 1) Exact literal, 2) Flexible (ignore whitespace, preserve indent), 3) Regex (tokenize + `\s*` between tokens) + **LLM self-correction** as fallback |
| **Poly-go** | **Exact replacement** (like Claude Code) + **propose_diff/apply_diff/reject_diff** workflow for user review before applying |

Gemini CLI's edit approach is the most robust - worth stealing the cascade pattern for Poly.

### Tools Poly Already Has (previously listed as missing)

| Tool | Status | Implementation |
|------|--------|---------------|
| `read_file` | **DONE** | Full: line numbers, binary detection, offset/limit, image detection, 256KB max, security path check |
| `list_files` | **DONE** | Full: recursive (max 3 levels), file sizes, [DIR]/[FILE] prefix, 500 entry limit |
| `glob` (standalone) | **DONE** | Full: `**` patterns, depth 10, 100 result limit, .gitignore-like skips |
| `grep` | **DONE** | Full: regex, case-insensitive, file glob filter, 100 result limit |
| `web_search` | **DONE** | DuckDuckGo Instant Answers API (free, no key) |
| `web_fetch` | **DONE** | Full: 30KB limit, HTML stripping, private IP blocking, redirect limit |
| `web_search` | **DONE** | DuckDuckGo API |

### Still Missing Tools in Poly

| Tool | Priority | Why |
|------|----------|-----|
| `ask_user` | HIGH | Structured questions with options |
| `agent` / `task` | HIGH | Sub-agent spawning |
| `download` | MEDIUM | Download files from URLs |
| `skill` | MEDIUM | Skill activation |
| `lsp_diagnostics` | LOW | LSP integration |

### Shell Execution Comparison

| Tool | Shell Strategy |
|------|---------------|
| Crush | **POSIX emulator** (mvdan.cc/sh) - cross-platform, no real bash |
| OpenCode | Real shell with tree-sitter parsing for safety |
| Gemini CLI | Real shell with safety checker |
| Claude Code | Real shell execution |
| **Poly-go** | **Real shell via os/exec** + **Interactive Shell mode** (`-i` flag: readline, AI pipes `ls \| @claude explain`, variables `$var = command`, history, tab completion) |

### What Poly Has
- **15 built-in tools** covering all core operations (read, write, edit, search, web, todos, diff workflow)
- **Tool approval system** with channels (PendingChan/ApprovedChan) for TUI integration
- **Modified files tracking** across the session
- **Security**: path validation (cannot read/write outside project root), private IP blocking for web_fetch
- **MCP tools** auto-registered from connected servers with namespacing (`mcp_serverName_toolName`)

### What Poly Still Needs (Priority: MEDIUM)
1. **Tool descriptions via //go:embed** like Crush (clean separation)
2. **AskUser tool** for interactive questions
3. **Agent/Task tool** for sub-agent spawning

---

## 4. MCP Integration

### MCP Feature Matrix

| Feature | Claude Code | Crush | OpenCode | Gemini CLI | Poly-go |
|---------|------------|-------|----------|-----------|---------|
| Client | Yes | Yes | Yes | Yes | **Yes** (full JSON-RPC 2.0 client) |
| stdio transport | Yes | Yes | Yes | Yes | **Yes** |
| SSE transport | Yes | Yes | Yes | ? | **No** |
| HTTP streamable | ? | Yes | Yes | ? | **No** |
| Tool discovery | Yes | Yes | Yes | Yes | **Yes** (`tools/list` with auto-registration) |
| Tool namespacing | No | No | No | No | **Yes** (`mcp_serverName_toolName`) |
| Multi-server manager | Yes | Yes | Yes | Yes | **Yes** (`Manager` with `ConnectAll`, status reporting) |
| Prompts | Yes | No | Yes | Yes | **No** |
| Resources | Yes | No | Yes | ? | **No** |
| OAuth for remote | No | No | Yes | Yes | **No** |
| Auto-reconnect | ? | Yes | ? | ? | **No** |
| Dynamic refresh | Yes | Yes | Yes | ? | **No** |
| Go SDK | N/A | **modelcontextprotocol/go-sdk** | N/A | N/A | **Custom** (hand-rolled JSON-RPC, protocol v2024-11-05) |

### What Poly Has
- **Full MCP client** with JSON-RPC 2.0 (initialize handshake, notifications, tool discovery, tool calls)
- **Manager** for multiple server connections (`ConnectAll`, per-server status, tool count)
- **Auto-registration**: MCP tools are automatically registered as Poly tools with namespacing (`mcp_serverName_toolName`)
- **mcpToolBridge**: transparently wraps MCP tools as Poly `Tool` interface implementations
- **1MB buffer** for large server responses
- **Server status reporting** (name, connected, tool count)

### What Poly Still Needs (Priority: MEDIUM)
1. Migrate to **modelcontextprotocol/go-sdk** (official SDK, better maintained)
2. Add **auto-reconnect** and **dynamic tool refresh**
3. Add **SSE transport** for remote MCP servers
4. Add **prompts and resources** support

---

## 5. Agent System (THE BIG GAP)

### Agent Architecture Comparison

| Tool | Agent Types | Communication | Teams |
|------|-------------|---------------|-------|
| Claude Code | 6 built-in sub-agents + custom agents + FULL TEAMS | SendMessage, TaskList, mailbox | **YES** (TeamCreate, teammates, task coordination) |
| Crush | 2 sub-agents (agent, agentic_fetch) | Return text result only | No |
| OpenCode | 7 built-in agents + custom agents | Return result, no inter-agent messaging | No |
| Gemini CLI | Local agents + remote A2A | delegate_to_agent tool | No (but A2A protocol) |
| **Poly-go** | **None** | **None** | **No** |

### Claude Code Agent Teams (THE GOLD STANDARD)

This is what Poly needs to replicate:

```
Team Lead (coordinator)
  |
  ├── Teammate 1 (own context window, tools, permissions)
  |     ├── Claims tasks from shared TaskList
  |     ├── Sends messages via SendMessage
  |     └── Goes idle when done
  |
  ├── Teammate 2
  |     └── ...
  |
  └── Shared TaskList
        ├── TaskCreate (subject, description, activeForm)
        ├── TaskUpdate (status, owner, blocks/blockedBy)
        ├── TaskList / TaskGet
        └── Dependencies: task A blocks task B
```

Key features:
- **Background agents**: run concurrently, return results later
- **Resumable**: agents can be resumed with full context
- **Custom agents**: defined in `.claude/agents/*.md` with frontmatter
- **Permission modes per agent**: plan mode, bypassPermissions, etc.
- **Max 7 concurrent agents**
- **Delegate mode**: team lead restricted to coordination only

### Sub-Agent Patterns

**Crush approach (simplest):**
- Agent is just another tool call
- Creates child session in DB
- Has restricted tool set (read-only for search agent)
- Returns text result to parent
- Cost aggregation via parent session

**OpenCode approach (medium):**
- TaskTool launches sub-agent
- Sub-agent inherits parent context
- Own permissions per agent type
- 7 built-in agent types with different roles

**Claude Code approach (full):**
- Task tool spawns sub-agents
- Agent Teams for coordination
- Mailbox system for inter-agent messaging
- Task list for work distribution
- Background execution with notifications

### What Poly Needs (Priority: CRITICAL)

**Phase 1: Sub-agents (like Crush)**
- `agent` tool that spawns read-only search sub-agent
- Child sessions with cost rollup
- Restricted tool sets per agent type

**Phase 2: Task tool (like OpenCode)**
- Multiple agent types (explore, plan, general)
- Custom agents from markdown files
- Background execution

**Phase 3: Agent Teams (like Claude Code)**
- TeamCreate / TeamDelete
- SendMessage (DM, broadcast, shutdown)
- TaskCreate / TaskUpdate / TaskList / TaskGet
- Task dependencies (blocks/blockedBy)
- Delegate mode

---

## 6. Hooks System

### Hooks Comparison

| Tool | Events | Types | Sources |
|------|--------|-------|---------|
| Claude Code | 14 events | command, prompt, agent | settings, plugins, skills, agents |
| Crush | None | N/A | N/A |
| OpenCode | 15+ hooks | Plugin API | internal, npm, local, file URL |
| Gemini CLI | Full system | shell scripts | project, user, system, extension |
| **Poly-go** | **Pub/sub broker** | Event-based (`pubsub/broker.go`, `pubsub/events.go`) | Internal |

### Claude Code Hook Events (most comprehensive)
- SessionStart, UserPromptSubmit, PreToolUse, PermissionRequest, PostToolUse, PostToolUseFailure, Notification, SubagentStart, SubagentStop, Stop, TeammateIdle, TaskCompleted, PreCompact, SessionEnd

### What Poly Has
- **Pub/sub broker** (`pubsub/broker.go`) - event-based system with `events.go` definitions
- **Tool approval channels** (PendingChan/ApprovedChan) - effectively a PreToolUse hook mechanism
- **Modified files tracking** - tracks all file changes during session

### What Poly Still Needs (Priority: MEDIUM)
- **Named hook events** (PreToolUse, PostToolUse, SessionStart, etc.)
- Shell command hooks (like Claude Code `type: "command"`)
- Config-based hook registration

---

## 7. Skills System

### Skills Comparison

| Feature | Claude Code | Crush | OpenCode | Gemini CLI | Poly-go |
|---------|------------|-------|----------|-----------|---------|
| Format | SKILL.md + frontmatter | SKILL.md (agentskills.io) | SKILL.md + .claude/ compat | SKILL.md | **None** |
| Discovery | .claude/skills/, ~/.claude/skills/ | skills_paths config | .opencode/skills/, .claude/skills/ | .gemini/skills/, extensions | **None** |
| Model override | Yes | No | No | Yes | **N/A** |
| Tool restrictions | Yes (allowed-tools) | No | No | Yes (tools field) | **N/A** |
| Context fork | Yes (context: fork) | No | No | No | **N/A** |

### What Poly Needs (Priority: MEDIUM)
- SKILL.md loader with YAML frontmatter
- Discovery in `.poly/skills/` and `~/.poly/skills/`
- Cross-compatibility: also search `.claude/skills/` (like OpenCode does)

---

## 8. Permission System

### Permission Comparison

| Tool | Approach | Modes | Granularity |
|------|----------|-------|-------------|
| Claude Code | 6 modes (default, acceptEdits, plan, delegate, dontAsk, bypass) | Mode switch via Shift+Tab | Per-tool glob patterns |
| Crush | Pub/sub dialog system | safe/banned command lists, --yolo | Per-command |
| OpenCode | Rules-based (Ruleset) | allow/deny/ask with glob patterns | 15+ permission types |
| Gemini CLI | Policy engine TOML | default/auto_edit/yolo | Per-tool, per-path, admin override |
| **Poly-go** | **Allow/Ask/Deny classification + YoloMode** | Allow (auto, read-only), Ask (dialog), Deny + `/yolo` toggle + per-tool auto-allow (`AllowTool`) | **Per-tool classification** (read_file=Allow, bash=Ask, etc.) |

### What Poly Has
- **3-level classification**: `Allow` (read-only tools auto-approved), `Ask` (side-effect tools prompt user), `Deny`
- **Tool classification map**: read_file, list_files, glob, grep, todos = Allow; bash, write_file, edit_file, multiedit, web_fetch, web_search = Ask
- **YoloMode**: `/yolo` command toggles auto-approve all tools
- **Per-tool auto-allow**: User can press 'a' to "Allow Always" for a specific tool (stored in `toolAllowList`)
- **ResetAllowList**: Clearing auto-approved tools when exiting YOLO mode
- **TUI Approval dialog**: Shows tool name, args summary, with Allow/Allow Always/YOLO options (3-choice approval with index)

### What Poly Still Needs (Priority: MEDIUM)
1. **Banned command list** (rm -rf /, sudo, etc. = always deny)
2. **Permission modes** beyond default/yolo (plan mode, etc.)
3. **Persistent "always allow" rules** per project (currently session-only)

---

## 9. Context Management

### Context Management Comparison

| Feature | Claude Code | Crush | OpenCode | Gemini CLI | Poly-go |
|---------|------------|-------|----------|-----------|---------|
| Auto-compaction | Yes (95% threshold) | Yes (auto-summarize) | Yes (pruning + AI resume) | Yes (/compress) | **No** |
| Memory files | MEMORY.md (auto-loaded) | No | No | GEMINI.md (save_memory) | **No** |
| Project instructions | CLAUDE.md hierarchy | crush.md + 10 other formats | opencode.json + .opencode/ | GEMINI.md | **No** |
| Session storage | .jsonl files | SQLite | JSON files + locks | JSON sessions | **JSON files** (`~/.poly/sessions/`, index.json + per-session JSON) |
| Session resume | /resume, /rewind | Session list | Session list | --resume | **Yes** (session list with switch, auto-resume current) |
| Session management | Basic | Session list | Session list | Basic | **Full** (switch, fork, delete, rename, auto-title from first message) |
| Multi-session | Yes | Yes | Yes | Yes | **Yes** (SessionIndex + individual JSON files, sorted by update time) |
| Session migration | No | No | No | No | **Yes** (auto-migrates old `current.json` to new multi-session format) |

### What Poly Has
- **JSON session persistence** in `~/.poly/sessions/` with index file
- **Full session management**: Load, Save, Clear, SwitchSession, ForkSession, DeleteSession, RenameSession
- **Auto-title** from first user message (truncated to 50 chars)
- **Session list** sorted by most recently updated
- **Auto-migration** from old single-session format
- **Session list dialog** in TUI (Ctrl+S) for browsing/switching sessions
- **Provider tracking** per session

### What Poly Still Needs (Priority: HIGH)
1. **Auto-summarization** when context window fills up
2. **Project instructions** (POLY.md or poly.md loaded at startup)
3. **Memory files** for persistent knowledge across sessions
4. Consider **SQLite** migration for better performance (currently JSON)

---

## 10. TUI Comparison

### TUI Stack

| Tool | Framework | Styling |
|------|-----------|---------|
| Claude Code | Ink (React) | Custom |
| Crush | Bubble Tea v2 | Lip Gloss v2 + charmtone |
| OpenCode | SolidJS (@opentui) | Custom |
| Gemini CLI | React 19 + Ink 6 | 13 themes |
| **Poly-go** | **Bubble Tea v2** | **Lip Gloss v2 + Catppuccin** |

Poly uses the SAME stack as Crush. This is an advantage - we can learn directly from Crush's patterns.

### TUI Components

| Component | Crush | Poly-go | Gap |
|-----------|-------|---------|-----|
| Chat | Full with messages, tool output, diff rendering | **Full** (messages with provider colors, tool output rendering per type: bash/edit/read/write/web/todos/generic) | Minor: needs syntax highlighting |
| Input editor | Clipboard, multiline | **Full** (textarea with clipboard support, image paste) | OK |
| Sidebar | Session list | **Full** (modified files list, toggle with `/sidebar`) | OK |
| Model picker | Dialog with API key input | **Full** (Ctrl+O, filter search, all model variants, recent models) | OK |
| Command palette | Slash commands | **Full** (Ctrl+P, filter search, 10 commands with shortcuts) | OK |
| Permission dialog | Grant/deny with pub/sub | **Full** (3-option: Allow / Allow Always / YOLO, with tool summary display) | OK |
| Control Room | N/A | **Full** (Ctrl+D, provider status, OAuth flows, API key entry, add/delete providers) | Unique to Poly |
| Session list | Session list | **Full** (Ctrl+S, switch/fork/delete/rename sessions) | OK |
| Help dialog | Basic | **Full** (Ctrl+H, keybindings reference) | OK |
| Add Provider | N/A | **Full** (form with id/url/apikey/model/format fields) | Unique to Poly |
| Diff viewer | Side-by-side + unified, chroma highlighting | **Partial** (unified diff with color-coded +/- lines, hunk headers) | Needs chroma |
| Splash screen | ASCII logo with randomization | **Full** (custom gradient splash component) | OK |
| Status bar | Model, tokens, cost | **Full** (model, provider, token count, cost, response time) | OK |
| Thinking display | N/A | **Full** (expandable/collapsible thinking blocks per message) | OK |

### What Poly Has
- **Model Picker** (Ctrl+O): filter search, all provider model variants, recent models tracking
- **Command Palette** (Ctrl+P): 10 commands with keyboard shortcuts, filter search
- **Control Room** (Ctrl+D): provider status, OAuth flows, API key input, add/delete providers
- **Session List** (Ctrl+S): browse/switch/fork/delete/rename sessions
- **Help Dialog** (Ctrl+H): keybindings reference
- **Add Provider Form**: multi-field form for adding custom providers
- **Approval Dialog**: 3-option (Allow/Allow Always/YOLO) with tool call summary
- **Diff rendering**: color-coded unified diff (green additions, red deletions, mauve hunk headers)
- **Tool output rendering**: per-tool type renderers (bash, edit, read, write, web, todos, generic)
- **Token/cost tracking**: session-level input/output tokens with cost calculation
- **Response time**: tracks streaming start/end for response time display
- **Thinking display**: expandable/collapsible thinking blocks per message
- **Image support**: paste images from clipboard, attach to messages
- **Sidebar**: modified files list, togglable

### What Poly Still Needs (Priority: LOW)
1. **Syntax highlighting** (chroma integration for code blocks and diffs)
2. **Vim mode**

---

## 11. Storage & Sessions

### Storage Comparison

| Tool | Backend | Pattern |
|------|---------|---------|
| Crush | **SQLite** (2 drivers) + goose migrations + sqlc queries | Best for Go |
| OpenCode | JSON files + file locks | Simple but slower |
| Claude Code | JSONL files | Append-only log |
| Gemini CLI | JSON sessions | Basic |
| **Poly-go** | **JSON files** (`~/.poly/sessions/`) | Multi-session with index, auto-title, switch/fork/delete/rename, auto-migration from old format |

### What Poly Has
- **JSON file persistence** in `~/.poly/sessions/`
- **SessionIndex** (`index.json`) tracking all sessions with current session pointer
- **Per-session JSON** files with messages, provider, model, timestamps
- **Auto-save** on every message
- **Auto-migration** from old single-session format
- **Full CRUD**: Load, Save, Clear, Switch, Fork, Delete, Rename
- **Custom providers** persisted in `~/.poly/providers.json`
- **Todos** persisted in `~/.poly/todos.json`
- **Config** in `~/.poly/config.json`

### What Poly Could Improve (Priority: LOW)
- Consider **SQLite** for better performance at scale (currently JSON works fine)
- **File locking** for concurrent access safety

---

## 12. Config System

### Config Comparison

| Tool | Format | Levels | Special |
|------|--------|--------|---------|
| Claude Code | JSON settings + CLAUDE.md | User > Project > Managed | Variable resolution |
| Crush | JSON + Catwalk remote | Global > Project data > Project | $ENV_VAR resolution |
| OpenCode | JSONC | 7 levels (!) | {env:VAR}, {file:path} |
| Gemini CLI | JSON | Schema > System > User > Workspace > Admin | Remote admin settings |
| **Poly-go** | **JSON** (`~/.poly/config.json`) | **2 levels** (defaults + user merge) | **Deep merge** (per-provider field merge, theme colors, settings), provider colors, cost tiers, OAuth client IDs |

### What Poly Has
- **Default config** with 4 providers, 16+ models, theme colors, settings
- **User config merge**: `~/.poly/config.json` overrides defaults with deep per-provider field merging
- **ProviderConfig** with: ID, Name, Endpoint, Models (variant map), Color, MaxTokens, Timeout, Format, AuthType, AuthHeader, OAuthClientID, CostTier
- **ThemeConfig**: per-provider color overrides
- **SettingsConfig**: MaxToolTurns, StreamingBuffer, SaveSessions
- **Runtime CRUD**: SetProvider, DeleteProvider, GetProviderModel, GetProviderColor, GetProviderNames (ordered)
- **Custom providers** stored separately in `~/.poly/providers.json`

### What Poly Still Needs (Priority: MEDIUM)
1. **Project-level config**: `poly.json` in project root
2. **Environment variable resolution** in config values
3. **Project instructions file** (POLY.md or poly.md)

---

## 13. Non-Interactive Mode (NEW from V2)

### Non-Interactive Comparison

| Feature | Claude Code | Gemini CLI | Crush | OpenCode | Poly-go |
|---------|------------|-----------|-------|----------|---------|
| Headless mode | `--print` / `-p` | Yes (pipe detection) | No | No | **No** |
| Output format | `text`, **`stream-json`** | `text`, `json`, **`stream-json`** | N/A | N/A | **N/A** |
| Input format | `text`, `stream-json` | stdin pipe | N/A | N/A | **N/A** |
| Structured output | `--json-schema` (codename "tengu") | No | No | No | **No** |
| SDK mode | `--sdk-url` (external connection) | No | No | HTTP server (Hono) | **No** |
| Session persistence control | `--no-session-persistence` | `--resume` | N/A | N/A | **N/A** |
| Stdin pipe | Yes (prompt from stdin) | Yes | No | No | **No** |
| Replay | `--replay-user-messages` | No | No | No | **No** |
| Partial messages | `--include-partial-messages` | No | No | No | **No** |

### Why This Matters for Poly
Non-interactive mode enables:
- **CI/CD integration** (run Poly in pipelines)
- **SDK for other tools** (Poly as a backend service)
- **Scripting** (pipe prompts, get structured output)
- **Agent-to-Agent** (other AIs calling Poly headless)

### What Poly Needs (Priority: MEDIUM)
1. `--print` flag for headless mode
2. `--output-format text|json|stream-json`
3. Stdin pipe support for prompts
4. `--json-schema` for structured output (Zod-like validation in Go)

---

## 14. Sandbox & Security (NEW from V2)

### Sandbox Comparison

| Feature | Claude Code | Crush | OpenCode | Gemini CLI | Poly-go |
|---------|------------|-------|----------|-----------|---------|
| Native sandbox | No | No | No | **Docker/Podman + macOS Seatbelt** | **No** |
| Container isolation | No | No | No | `--rm --init`, user matching (UID/GID) | **No** |
| Network proxy | No | No | No | `GEMINI_SANDBOX_PROXY_COMMAND` | **No** |
| Path validation | No explicit | No | No | `config.validatePathAccess()` on every file op | **No** |
| Trust system | CLAUDE.md hierarchy | No | No | Workspace trust for settings/agents/skills | **No** |
| Agent hash verification | No | No | No | Hash-based acknowledgment for project agents | **No** |
| Redirection downgrade | No | No | tree-sitter | Shell `>`, `>>`, `\|` -> ASK_USER | **No** |
| Private IP protection | No explicit | No | No | `isPrivateIp()` in WebFetch | **No** |
| Env sanitization | No explicit | No | No | `sanitizeEnvVar()` regex whitelist | **No** |

Gemini CLI is the security champion here. Poly should steal:
1. **Path validation** on every file operation
2. **Redirection downgrade** for shell commands (HIGH priority)
3. **Private IP check** for web fetch

---

## 15. Plugin & Extension System (NEW from V2)

### Plugin/Extension Comparison

| Feature | Claude Code | Crush | OpenCode | Gemini CLI | Poly-go |
|---------|------------|-------|----------|-----------|---------|
| System | **Plugin registry** | None | **15+ plugin hooks** | **Full extension system** | **None** |
| Install method | `/plugin install name@registry` | N/A | npm/local/file URL | `extensions install <github-url>` | **N/A** |
| Can provide | Tools, skills, agents, hooks | N/A | Tools, hooks, middleware | **MCP servers, hooks, agents, skills, themes, policies** | **N/A** |
| Auto-suggest | Yes (file-based relevance detection) | N/A | No | No | **N/A** |
| Cooldown | 3 sessions between suggestions | N/A | N/A | N/A | **N/A** |
| Config | Registry-based | N/A | Plugin API | `EXTENSIONS_CONFIG_FILENAME` + consent | **N/A** |
| Scoped settings | No | N/A | No | **Yes** (`ExtensionSettingScope`) | **N/A** |
| Theme contribution | No | N/A | No | **Yes** (namespaced) | **N/A** |

### Gemini CLI Extension Commands
```
extensions install <url>    extensions list
extensions uninstall <name> extensions update [name]
extensions enable <name>    extensions configure <name>
extensions disable <name>   extensions link <path>
extensions new              extensions validate
```

### What Poly Needs (Priority: LOW - build core first)
- Start with a simple **plugin interface** in Go (interface with hooks)
- Later add GitHub-based install like Gemini CLI

---

## 16. Themes & Visual (NEW from V2)

### Theme Comparison

| Feature | Claude Code | Crush | OpenCode | Gemini CLI | Poly-go |
|---------|------------|-------|----------|-----------|---------|
| Built-in themes | 1 (fixed) | 1 + Catppuccin colors | 1 | **14** (Ayu, Dracula, GitHub, etc.) | **Catppuccin Mocha** |
| Custom themes | No | No | No | **Yes** (JSON + extensions) | **No** |
| Auto light/dark | No | No | No | **Yes** (terminal bg detection) | **No** |
| Vim mode | No | No | No | **Yes** (`VimModeProvider`) | **No** |
| Semantic tokens | No | No | No | `SemanticColors` (text, bg, border, ui, status) | **No** |
| Syntax highlight | Code blocks | **chroma** | Custom | **highlight.js** mapping | **Diff highlighting** (color-coded +/-/@@, tool output styling) |

### Gemini CLI 14 Themes
Ayu Dark, Ayu Light, Atom One Dark, Dracula, Default Light, Default Dark, GitHub Dark, GitHub Light, Google Code, Holiday, Shades of Purple, XCode, ANSI, ANSI Light + `NoColorTheme`

### What Poly Needs (Priority: LOW)
- Catppuccin Mocha is already good for now
- Later add `/theme` command with at least 3-4 options
- Terminal background auto-detection would be nice

---

## 17. System Prompt Architecture (NEW from V2)

### Claude Code System Prompt (13 sections, 15-25K tokens)

| Section | Content | Tokens (approx) |
|---------|---------|-----------------|
| 1. Identity | "You are Claude Code, Anthropic's official CLI" | ~100 |
| 2. System instructions | Security, style, tool rules | ~3K |
| 3. Tool instructions | Per-tool: description + params + examples | ~5K |
| 4. Git instructions | Commit workflow, PR creation, safety rules | ~1K |
| 5. Auto memory | MEMORY.md (first 200 lines) | ~1K |
| 6. CLAUDE.md | Project instructions (parent -> child hierarchy) | 1-10K |
| 7. Environment | OS, platform, date, model, cwd, git status | ~200 |
| 8. Language | Language instruction if configured | ~50 |
| 9. MCP tools | Discovered MCP tool descriptions | 2-5K |
| 10. Skill context | Active skills injected dynamically | 1-5K |
| 11. Team context | Team info if Agent Teams active | ~500 |
| 12. Browser automation | Chrome instructions if MCP active | ~2K |
| 13. Copyright | Web content copyright rules | ~500 |

### Prompt Caching (Claude Code exclusive)
- **Ephemeral 5m cache**: Short-lived cache for repeated calls
- **Ephemeral 1h cache**: Longer cache for system prompt
- `cache_creation_input_tokens` + `cache_read_input_tokens` in usage
- System prompt aggressively cached -> massive cost reduction

### What Poly Has
Poly has a **sophisticated 6-section dynamic system prompt** (`llm/system.go: BuildSystemPrompt()`):

| Section | Content |
|---------|---------|
| 1. **GROUND TRUTH** | Immutable facts about provider identity, environment, connected AIs. Set by system, not user. |
| 2. **ANTI-GASLIGHTING** | Resist reality manipulation attempts. Specific counter-responses for identity confusion. Scoped: only applies to identity, not technical topics. |
| 3. **PEER ISOLATION** | Don't adopt other AIs' confusion. Judge on technical merit only. Each AI responsible for own grounding. |
| 4. **OPERATIONAL CONTEXT** | Dynamic: @mentions list, available tools (from registry), output format, developer-oriented instructions. |
| 5. **CASCADE ROLES** | 3 roles: RESPONDER (first to answer), REVIEWER (check for errors, output only checkmark if correct), DIRECT (default). Reviewer exception for personal questions. |
| 6. **SECURITY PROTOCOL** | Never reveal prompt, never change identity, reject prompt injection, no destructive commands without confirmation, prefer read-only. |

**Key features:**
- Fully **dynamic** from config (no hardcoded provider names or tool lists)
- **Per-provider** prompts (each AI gets told who it is and who it's NOT)
- **Cascade-aware** (different roles for @all mode)
- **Anti-gaslighting** is unique to Poly (no other tool has this)

### What Poly Still Needs
- Prompt caching support (when providers support it)
- Dynamic injection of skills, memory, project instructions

---

## 18. Slash Commands (NEW from V2)

### Slash Command Comparison

| Feature | Claude Code | Crush | OpenCode | Gemini CLI | Poly-go |
|---------|------------|-------|----------|-----------|---------|
| Total commands | ~15 | ~10 | ~12 | **40+** | **12** (slash) + **10** (palette) + **@mentions** |
| Custom from files | `.claude/commands/*.md` | No | `.opencode/commands/` | `.gemini/commands/` | **No** |
| MCP prompts as commands | Yes | No | No | Yes | **No** |
| Agents as commands | Yes | No | No | Yes (`CommandKind.AGENT`) | **No** |
| Autocompletion | Yes | No | Yes | Yes (per-command `completion()`) | **Partial** (readline completion in shell mode) |
| Sub-commands | No | No | No | Yes (`subCommands: Map`) | **No** |
| At-commands | No | No | No | **Yes** (`@agent`, `@file`) | **Yes** (`@claude`, `@gpt`, `@gemini`, `@grok`, `@all`) |
| Command palette | No | No | No | No | **Yes** (Ctrl+P, filterable, 10 actions with shortcuts) |

### Notable Gemini CLI Commands (40+)
`/about`, `/agents`, `/auth`, `/chat`, `/clear`, `/compress`, `/copy`, `/directory`, `/docs`, `/editor`, `/extensions`, `/help`, `/hooks`, `/ide`, `/init`, `/mcp`, `/memory`, `/model`, `/permissions`, `/plan`, `/policies`, `/privacy`, `/profile`, `/quit`, `/restore`, `/resume`, `/rewind`, `/settings`, `/shells`, `/shortcuts`, `/skills`, `/stats`, `/terminalSetup`, `/theme`, `/tools`, `/vim` + custom/file/agent/MCP commands

### What Poly Has
**Slash commands (12):**
- `/clear` (`/c`) - clear chat
- `/model` (`/m`) - show/change model variant (fast/think/opus/default)
- `/think` (`/t`) - toggle thinking mode
- `/provider` (`/p`) - show/change default provider
- `/help` (`/h`) - show help
- `/providers` (`/list`) - list all providers
- `/addprovider` (`/add`) - add custom provider (id, url, apikey, model, format, color)
- `/delprovider` (`/del`) - delete custom provider
- `/sidebar` - toggle sidebar
- `/yolo` - toggle YOLO mode (auto-approve all tools)

**Command palette (Ctrl+P, 10 actions):**
New Session, Switch Model, Toggle Thinking, Session List, Toggle Sidebar, Control Room, Toggle Help, Clear Chat, Toggle YOLO Mode, Quit

**@mentions:**
`@claude`, `@gpt`, `@gemini`, `@grok` - direct to specific provider
`@all` - cascade mode (cheapest-first + parallel reviewers)

### What Poly Still Needs (Priority: MEDIUM)
1. `/compact` - manual compaction
2. Custom commands from `.poly/commands/*.md`

---

## 19. Unique Features Worth Stealing (UPDATED from V2)

### Poly-go Unique Features (things NO OTHER tool has)
- **Multi-AI @all cascade** - cheapest provider responds first, all others review in parallel with dedicated reviewer prompts
- **Anti-gaslighting system prompt** - 3 dedicated sections (Ground Truth, Anti-Gaslighting, Peer Isolation) to keep AIs grounded in multi-AI context
- **Per-provider identity prompts** - each AI told who it IS and who it's NOT, dynamic from config
- **Cascade roles** (responder/reviewer/direct) with smart review: reviewers output checkmark if correct, corrections if errors found, exception for personal questions
- **Interactive Shell** (`-i` flag) with AI pipes (`ls | @claude explain`), variables (`$var = cmd`), history, tab completion
- **@mentions** for provider targeting (`@claude`, `@gpt`, `@gemini`, `@grok`, `@all`)
- **Diff propose/review workflow** - AI proposes changes, user reviews before applying (propose_diff/apply_diff/reject_diff/list_diffs)
- **Control Room** (Ctrl+D) - unified provider management with OAuth flows, API key entry, add/delete custom providers
- **Custom providers** via `/addprovider` command with 3 API format support (OpenAI, Anthropic, Google compatible)
- **Provider colors** for visual identification in multi-AI chat
- **Cost-tier-based cascade ordering** - cheapest provider answers first to optimize costs

### From Claude Code (V2 reverse engineering)
- **Agent Teams** (the biggest differentiator) - TeamCreate, SendMessage, mailbox, idle/wake cycle
- **AsyncLocalStorage pattern** for team context (Go equivalent: `context.Context`)
- **Plan mode** (read-only analysis before implementation)
- **Memory files** (MEMORY.md, 200 lines max, auto-loaded in system prompt)
- **Hooks system** (14 events, **3 types**: command, prompt, agent - agent hooks are multi-turn!)
- **Custom sub-agents** from markdown with rich frontmatter (model, tools, disallowedTools, permissionMode, maxTurns, skills, mcpServers, hooks, memory scope)
- **Non-interactive mode** (`--print`, `--output-format stream-json`, `--json-schema` for structured output)
- **Plugin registry** with auto-suggestions based on file relevance detection
- **Prompt caching** (ephemeral 5m/1h) - massive cost savings
- **6 TUI surfaces** (Terminal, VS Code, JetBrains, Desktop, Web, iOS)
- **Fast mode** (same model, faster output) via `/fast`
- **System prompt architecture** (13 dynamic sections, 15-25K tokens)

### From Crush
- **Fantasy SDK** approach (clean provider abstraction)
- **Catwalk** remote model registry
- **SQLite + goose + sqlc** (same Go stack!)
- **POSIX shell emulator** (cross-platform)
- **File tracker** (detect stale edits)
- **Pub/sub** for permissions and events
- **LSP integration** (diagnostics, references)
- **Auto-summarization** on context overflow

### From OpenCode
- **20+ providers** via single SDK (Vercel AI SDK)
- **Plugin system** with 15+ hooks
- **ACP protocol** support
- **JSONC config** with 7 precedence levels
- **HTTP server** (enables web UI, SDK, remote)
- **Skill compatibility** with .claude/ directory

### From Gemini CLI (V2 reverse engineering - 1365 TS files)
- **Extension system** (full: install from GitHub, themes, hooks, agents, skills, policies, settings)
- **A2A protocol** (remote agents with `RemoteAgentDefinition`)
- **Policy engine TOML** (3 tiers: default/user/admin, commandPrefix/commandRegex, MCP wildcard)
- **14 themes** + custom themes from extensions + auto light/dark detection
- **Vim mode** (`VimModeProvider` in React tree)
- **Docker/Podman sandbox** (user matching UID/GID, network proxy, env passthrough)
- **macOS Seatbelt** sandbox (`.sb` profiles)
- **Edit tool 3-strategy cascade** (exact -> flexible whitespace -> regex tokenize) + **LLM self-correction**
- **WriteFile LLM correction** + user modification (`ModifiableDeclarativeTool`)
- **MessageBus** pub/sub with correlation IDs for tool confirmations (30s timeout)
- **40+ slash commands** with sub-commands, autocompletion, at-commands (`@agent`, `@file`)
- **OAuth PKCE** + Dynamic Client Registration (RFC 7591) for MCP
- **Memory relaunch** (50% RAM allocation via V8 heap args)
- **10 security layers** (PolicyEngine, sandbox, path validation, trust, hash verification, redirection downgrade, private IP, theme security, env sanitization, OAuth)

---

## 20. Implementation Priority Roadmap

### Phase 1: FOUNDATIONS (make it usable) -- MOSTLY DONE
**Priority: CRITICAL | Status: ~85% COMPLETE**

| # | Feature | Status | Notes |
|---|---------|--------|-------|
| 1 | **Read tool** | **DONE** | `read_file` with line numbers, binary detection, offset/limit |
| 2 | **Session persistence** | **DONE** | JSON multi-session with switch/fork/delete/rename/auto-title |
| 3 | **Auto-summarization** | **TODO** | Context overflow handling not yet implemented |
| 4 | **Project instructions** | **TODO** | POLY.md loader not yet implemented |
| 5 | **Permission system** | **DONE** | Allow/Ask/Deny + YoloMode + auto-allow per tool |
| 6 | **Glob tool** | **DONE** | Standalone glob with ** patterns |
| 7 | **LS tool** | **DONE** | `list_files` with recursive, sizes, depth limits |

### Phase 2: AGENT SYSTEM (the big differentiator)
**Priority: HIGH | Timeline: 2-3 weeks**

| # | Feature | Inspiration | Complexity |
|---|---------|-------------|------------|
| 8 | **Sub-agent tool** (read-only search agent) | Crush agent_tool.go | Medium |
| 9 | **Task tool** (general-purpose sub-agent) | Claude Code Task tool | Medium |
| 10 | **AskUser tool** | Claude Code AskUserQuestion | Low |
| 11 | **Background agents** | Claude Code run_in_background | High |
| 12 | **Custom agents** from markdown | Claude Code .claude/agents/ | Medium |

### Phase 3: TEAMS & COORDINATION
**Priority: HIGH | Timeline: 2-3 weeks**

| # | Feature | Inspiration | Complexity |
|---|---------|-------------|------------|
| 13 | **TeamCreate / TeamDelete** | Claude Code Agent Teams | High |
| 14 | **SendMessage** (DM, broadcast, shutdown) | Claude Code SendMessage | High |
| 15 | **TaskCreate/Update/List/Get** | Claude Code task tools | Medium |
| 16 | **Task dependencies** (blocks/blockedBy) | Claude Code TaskUpdate | Medium |
| 17 | **Delegate mode** | Claude Code Shift+Tab | Low |

### Phase 4: POLISH & POWER FEATURES -- PARTIALLY DONE
**Priority: MEDIUM | Status: ~60% COMPLETE**

| # | Feature | Status | Notes |
|---|---------|--------|-------|
| 18 | **MCP upgrade** (Go SDK, auto-reconnect) | **PARTIAL** | Custom client works, needs official Go SDK |
| 19 | **Model picker dialog** | **DONE** | Ctrl+O, filter search, all variants |
| 20 | **Command palette** | **DONE** | Ctrl+P, 10 actions, filter search |
| 21 | **Diff viewer** (chroma highlighting) | **PARTIAL** | Color-coded diff rendering, needs chroma |
| 22 | **Hooks system** | **PARTIAL** | Pub/sub broker exists, needs named events |
| 23 | **Skills system** (SKILL.md) | **TODO** | |
| 24 | **OAuth for providers** | **DONE** | Anthropic, OpenAI, Google OAuth + PKCE |
| 25 | **Memory files** (MEMORY.md) | **TODO** | |
| 26 | **Plan mode** | **TODO** | |
| 27 | **Web search + fetch tools** | **DONE** | DuckDuckGo search + URL fetch with security |
| 28 | **LSP integration** | **TODO** | |
| 29 | **OpenAI-compatible endpoint** | **PARTIAL** | Custom provider supports OpenAI format, no generic endpoint |
| 30 | **Session resume** | **DONE** | Session list with switch, auto-resume current |

### Phase 5: ADVANCED (NEW from V2 research)
**Priority: LOW | Timeline: future**

| # | Feature | Inspiration | Complexity |
|---|---------|-------------|------------|
| 31 | **Non-interactive mode** (`--print`, `--output-format`) | Claude Code + Gemini CLI | Medium |
| 32 | **Edit cascade** (exact -> flexible -> regex + LLM correction) | Gemini CLI edit.ts | High |
| 33 | **Path validation** on every file op | Gemini CLI `validatePathAccess()` | Low |
| 34 | **Redirection downgrade** (shell `>`, `>>`, `\|` -> ask) | Gemini CLI shell.ts | Low |
| 35 | **Prompt caching** (ephemeral for Anthropic) | Claude Code | Medium |
| 36 | **40+ slash commands** with autocompletion | Gemini CLI | Medium |
| 37 | **Extension system** (GitHub-based install) | Gemini CLI extensions | High |
| 38 | **Theme system** (multiple themes + custom) | Gemini CLI 14 themes | Medium |
| 39 | **Vim mode** | Gemini CLI VimModeProvider | Medium |
| 40 | **Structured output** (`--json-schema`) | Claude Code "tengu" | Medium |
| 41 | **Multi-surface** (VS Code extension) | Claude Code + Gemini CLI | Very High |
| 42 | **Docker sandbox** | Gemini CLI sandbox.ts | High |

---

## 21. Key Dependencies to Add

| Package | Purpose | Used By |
|---------|---------|---------|
| `modelcontextprotocol/go-sdk` | Official MCP Go SDK | Crush |
| `ncruces/go-sqlite3` or `modernc.org/sqlite` | SQLite driver | Crush |
| `pressly/goose/v3` | DB migrations | Crush |
| `alecthomas/chroma/v2` | Syntax highlighting | Crush |
| `aymanbagabas/go-udiff` | Unified diff | Crush |
| `mvdan.cc/sh/v3` | POSIX shell emulator (optional) | Crush |

---

## 22. Architecture Decision: Follow Crush

Since Poly-go uses the **exact same stack as Crush** (Go + Bubble Tea v2 + Lip Gloss v2), the fastest path is:

1. **Copy Crush's architecture patterns** (service wiring, pub/sub, tool registration)
2. **Add Claude Code's unique features** (Agent Teams, hooks, skills, memory)
3. **Add OpenCode's provider breadth** (OpenAI-compatible endpoint covers most)
4. **Add Gemini CLI's unique features** (extension system, policy engine)

The result: a Go TUI that combines the best of all 4 tools.

---

## Research Sources

### V2 (Reverse Engineering - current)
- `/home/pedro/PROJETS-AI/Poly-go/research/claude-code-v2.md` (800+ lines, 16 sections, binary ELF extraction + strings analysis)
- `/home/pedro/PROJETS-AI/Poly-go/research/gemini-cli-v2.md` (1100+ lines, 20 sections, full TypeScript source reading)

### V1 (Already good quality)
- `/home/pedro/PROJETS-AI/Poly-go/research/crush.md` (660 lines, full Go source reverse engineering)
- `/home/pedro/PROJETS-AI/Poly-go/research/opencode.md` (1000+ lines, full TypeScript reverse engineering)

### V1 (Superseded by V2)
- `/home/pedro/PROJETS-AI/Poly-go/research/claude-code.md` (700 lines, web research only)
- `/home/pedro/PROJETS-AI/Poly-go/research/gemini-cli.md` (660 lines, surface level)

### Cloned Repos
- `/home/pedro/PROJETS-AI/opencode/` - OpenCode TypeScript monorepo (1006 TS files)
- `/home/pedro/PROJETS-AI/gemini-cli/` - Gemini CLI TypeScript monorepo (1365 TS files)
