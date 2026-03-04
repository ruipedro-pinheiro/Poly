<div align="center">

# Poly

**Multi-AI terminal client with agentic tool use**

Route prompts to Claude, Gemini, GPT, Grok, Copilot, Ollama — or any OpenAI/Anthropic/Google-compatible API — from a single TUI.

[![CI](https://github.com/ruipedro-pinheiro/Poly/actions/workflows/ci.yml/badge.svg)](https://github.com/ruipedro-pinheiro/Poly/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

</div>

> **BETA (v0.7.1)** — Poly is in active development. Expect breaking changes, rough edges, and missing features. [Open an issue](https://github.com/ruipedro-pinheiro/Poly/issues/new/choose) if something breaks.

---

## Stability

Not everything is production-ready. Here's what works and what doesn't:

| Status | Feature | Notes |
|--------|---------|-------|
| **Stable** | Core TUI (input, rendering, keybindings) | Bubble Tea v2 + Catppuccin Mocha |
| **Stable** | Provider integrations (Claude, GPT, Gemini, Grok, Ollama) | Streaming, tool calling, agentic loop |
| **Stable** | Built-in tools (21) | File ops, bash, git, web, diffs |
| **Stable** | Session management | Auto-save, resume, fork, export |
| **Stable** | OAuth + API key auth | PKCE for Anthropic/OpenAI/Google, Device Flow for Copilot |
| **Stable** | Copilot provider | Device Flow auth, session token refresh |
| **Stable** | Table Ronde (`@all`) | Multi-provider conversations |
| **Stable** | Custom providers | OpenAI/Anthropic/Google-compatible endpoints |
| **Stable** | Sandboxed bash (Podman/Docker) | Hardened isolation, no network, read-only root |
| **Stable** | Context compaction | Auto-summarize near context limit |
| **In Test** | Hooks system | Pre/post tool execution, Go templates |
| **In Test** | Skills system | Custom .md behaviors in `~/.poly/skills/` |

---

## Demo

<!-- TODO: Record a short terminal demo (asciinema or GIF) showing:
     1. Basic @claude / @gpt routing
     2. @all Table Ronde
     3. Shell pipe mode (ls | @gemini ...)
     4. Tool use with approval flow
-->

*Demo recordings coming soon. See [Features](#features) below for details.*

---

## Features

### Multi-Provider
- **6 native providers**: Claude, GPT, Gemini, Grok, Copilot, Ollama (local)
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
- **Multi-line input** with Shift+Enter, auto-grow (1-5 lines)
- **Session management**: auto-save, resume, fork, export (Markdown/JSON)

<!-- TODO: Add screenshots of:
     - Main TUI interface with a conversation
     - Control Room (Ctrl+D)
     - Command palette (Ctrl+K)
     - Tool approval dialog
-->

### Under the Hood
- **Context compaction**: auto-summarize old messages near context limit
- **Prompt caching**: Anthropic ephemeral cache on system prompt
- **POLY.md / MEMORY.md**: project instructions + persistent memory
- **Skills system**: load custom behaviors from `~/.poly/skills/`
- **Hooks**: pre/post tool execution hooks with Go templates
- **OAuth + API key auth**: PKCE flows for Anthropic, OpenAI, Google; Device Flow for Copilot
- **Edit cascade**: 3-strategy file editing (exact, fuzzy, line-based)

## Quick Start

### From releases (no Go required)

Download the latest binary from [Releases](https://github.com/ruipedro-pinheiro/Poly/releases), extract, and run:

```bash
tar xzf poly_*_linux_amd64.tar.gz
./poly
```

### From source

```bash
git clone https://github.com/ruipedro-pinheiro/Poly.git
cd Poly
make build
./poly

# Or install to ~/go/bin/
make install
```

Requires **Go 1.25+**.

### Configuration

Open the **Control Room** with `Ctrl+D` and connect providers via OAuth interactively.

Or create `~/.poly/config.json`:

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

For local models:

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

## Architecture

```
poly/
├── main.go
├── Makefile
├── internal/
│   ├── llm/                 # LLM providers + agentic loop
│   │   ├── oai_base.go      #   Shared OpenAI-compatible base (GPT/Grok/Copilot)
│   │   ├── anthropic.go     #   Claude
│   │   ├── gemini.go        #   Gemini (public API + Code Assist)
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
│   ├── auth/                # OAuth PKCE + Device Flow + token storage
│   ├── hooks/               # Pre/post tool hooks (Go templates)
│   ├── skills/              # Skill loader (.md → system prompt)
│   ├── session/             # Session persistence + fork + export
│   ├── security/            # Blocklist, path validation, secret sanitization
│   ├── shell/               # Hybrid shell pipe mode
│   └── theme/               # Catppuccin Mocha palette
└── scripts/
```

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md) for the full guide.

```bash
make build           # Compile binary
make dev             # Build with race detector
make test            # Run all tests
make ci              # Full CI locally (build + vet + test -race)
make fmt             # gofmt
```

## Requirements

- **Go 1.25+** (or download a pre-built binary from [Releases](https://github.com/ruipedro-pinheiro/Poly/releases))
- API keys or OAuth credentials for desired providers
- Optional: `podman` or `docker` for sandbox mode
- Optional: `ollama` for local models

## License

MIT
