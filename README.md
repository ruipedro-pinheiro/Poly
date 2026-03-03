<div align="center">

# Poly

**Multi-AI terminal client with agentic tool use**

Route prompts to Claude, Gemini, GPT, Grok, Ollama — or any OpenAI/Anthropic/Google-compatible API — from a single TUI.

[![CI](https://github.com/ruipedro-pinheiro/Poly/actions/workflows/ci.yml/badge.svg)](https://github.com/ruipedro-pinheiro/Poly/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Bubble Tea](https://img.shields.io/badge/Bubble%20Tea-v2-FF5F87?logo=data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAyNCAyNCI+PC9zdmc+)](https://github.com/charmbracelet/bubbletea)
[![Catppuccin](https://img.shields.io/badge/theme-Catppuccin%20Mocha-cba6f7?logo=data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAyNCAyNCI+PC9zdmc+)](https://catppuccin.com/)
[![Lines](https://img.shields.io/badge/Go%20LOC-~28K-blue)]()
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

</div>

---

## What is Poly?

Poly is a terminal-native AI client that lets you talk to multiple LLM providers from one interface. It's not a wrapper — it's a full agentic client with tool calling, MCP support, session management, and a TUI built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

```
@claude refactor this function       → routes to Claude
@gpt explain the error               → routes to GPT
@all debate: Go vs Rust              → all providers discuss together
ls -la | @gemini find the largest    → shell pipe mode
```

## Features

### Multi-Provider
- **5 native providers**: Claude, GPT, Gemini, Grok, Ollama (local)
- **Custom providers**: any OpenAI, Anthropic, or Google-compatible endpoint
- **`@all` Table Ronde**: multi-round conversation where providers @mention and respond to each other
- **Per-message cost tracking** with live token stats

### Agentic Tool Use
- **21 built-in tools** with full agentic loop (multi-step, approval-gated)
- **MCP support**: connect external tool servers via JSON-RPC 2.0 (stdio)
- **3-tier approval**: Allow once / Always / YOLO mode
- **Sandboxed execution**: bash runs in podman/docker containers

### TUI
- **Catppuccin Mocha** theme with per-provider accent colors
- **30+ slash commands**: `/compact`, `/rewind`, `/stats`, `/mcp`, `/export`, ...
- **Command palette** (`Ctrl+K`), **Control Room** (`Ctrl+D`), **Model picker** (`Ctrl+O`)
- **Multi-line input** with Shift+Enter, auto-grow (1→5 lines)
- **Session management**: auto-save, resume, fork, export (Markdown/JSON)

### Under the Hood
- **Context compaction**: auto-summarize old messages near context limit
- **Prompt caching**: Anthropic ephemeral cache on system prompt
- **POLY.md / MEMORY.md**: project instructions + persistent memory
- **Skills system**: load custom behaviors from `~/.poly/skills/`
- **Hooks**: pre/post tool execution hooks with Go templates
- **OAuth + API key auth**: PKCE flows for Anthropic, OpenAI, Google
- **Edit cascade**: 3-strategy file editing (exact → fuzzy → line-based)

## Quick Start

```bash
# Clone & build
git clone https://github.com/ruipedro-pinheiro/Poly.git
cd Poly
make build

# Run
./poly

# Or install to ~/go/bin/
make install
```

### Configuration

Create `~/.poly/config.json`:

```json
{
  "providers": {
    "claude": { "api_key": "sk-ant-..." },
    "gpt": { "api_key": "sk-..." },
    "gemini": { "api_key": "..." },
    "grok": { "api_key": "..." }
  }
}
```

Or skip the config — open the **Control Room** with `Ctrl+D` and connect via OAuth interactively.

For local models, just add Ollama:

```json
{
  "providers": {
    "ollama": { "models": ["llama3", "codellama"] }
  }
}
```

## Tools

| Category | Tools |
|----------|-------|
| **Files** | `read_file`, `write_file`, `edit_file`, `multiedit`, `list_files`, `glob`, `grep` |
| **Exec** | `bash` (sandboxable, configurable timeout) |
| **Diffs** | `propose_diff`, `apply_diff`, `reject_diff`, `list_diffs` |
| **Web** | `web_search`, `web_fetch` |
| **Git** | `git_status`, `git_diff`, `git_log` |
| **Utils** | `delegate_task`, `memory_write`, `system_info`, `todos` |
| **MCP** | Any tools from connected MCP servers (auto-namespaced) |

## Usage

```
@claude what is this project?         # Route to a specific provider
@all compare Go vs Rust               # Table Ronde — all providers discuss
ls -la | @claude find the largest     # Hybrid shell pipe

/compact                              # Summarize old context
/rewind 4                             # Remove last 4 messages
/stats                                # Token count + cost
/export markdown                      # Export session
/mcp status                           # MCP server status

Ctrl+K                                # Command palette
Ctrl+D                                # Control Room (manage providers)
Ctrl+O                                # Model picker
```

## Development

```bash
make build           # Compile binary
make dev             # Build with race detector
make release         # Optimized build (-s -w)
make test            # Run all tests
make test-coverage   # Coverage report (text + HTML)
make ci              # Full CI locally (build + vet + test -race)
make lint            # go vet
make fmt             # gofmt
make clean           # Remove artifacts
```

## Architecture

```
poly/
├── main.go
├── Makefile
├── internal/
│   ├── llm/                 # LLM providers + agentic loop
│   │   ├── anthropic.go     #   Claude
│   │   ├── gpt.go           #   GPT
│   │   ├── gemini.go        #   Gemini (public API + Code Assist)
│   │   ├── grok.go          #   Grok
│   │   ├── ollama.go        #   Ollama (local)
│   │   ├── custom.go        #   Custom (OpenAI/Anthropic/Google format)
│   │   ├── openai_types.go  #   Typed API structs (OAI*)
│   │   ├── compaction.go    #   Context compaction
│   │   ├── retry.go         #   Exponential backoff + jitter
│   │   └── system.go        #   Dynamic system prompt
│   ├── types/               # Shared types (ToolCall, ToolResult, ToolDefinition)
│   ├── tools/               # 21 tool implementations + registry
│   ├── mcp/                 # MCP client (JSON-RPC 2.0, stdio, auto-reconnect)
│   ├── config/              # Config, POLY.md, MEMORY.md, history
│   ├── tui/                 # Terminal UI (Bubble Tea v2 + Lip Gloss v2)
│   │   └── components/      #   Header, messages, status, infopanel, splash
│   ├── permission/          # Bash command classification + word-boundary matching
│   ├── sandbox/             # Container isolation (podman/docker)
│   ├── auth/                # OAuth PKCE + token storage
│   ├── hooks/               # Pre/post tool hooks (Go templates)
│   ├── skills/              # Skill loader (.md → system prompt)
│   ├── session/             # Session persistence + fork + export
│   ├── security/            # File permissions, path validation
│   ├── shell/               # Hybrid shell pipe mode
│   └── theme/               # Catppuccin Mocha palette
└── scripts/
```

## Stats

| Metric | Value |
|--------|-------|
| Go source lines | ~28K |
| Go files | 140 |
| Test files | 27 |
| Test functions | 338 |
| Built-in tools | 21 |
| Native providers | 5 + unlimited custom |
| Slash commands | 30+ |

## Requirements

- **Go 1.25+**
- API keys or OAuth credentials for desired providers
- Optional: `podman` or `docker` for sandbox mode
- Optional: `ollama` for local models

## License

MIT
