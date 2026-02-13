# Poly

Multi-AI collaborative terminal. Route prompts to Claude, Gemini, GPT, Grok, or any OpenAI-compatible API from a unified TUI.

Built with Go, Bubble Tea v2, Lip Gloss v2, and the Catppuccin Mocha theme.

## Features

- **Multi-provider routing**: `@claude`, `@gemini`, `@gpt`, `@grok`, `@all`
- **Cascade mode**: `@all` asks the cheapest provider first, others review
- **Agentic tool use**: Full tool loop with approval system (Allow / Always / YOLO)
- **21+ built-in tools**: File I/O, bash, web search, diffs, git, delegation
- **MCP support**: Connect external tool servers via JSON-RPC 2.0 (stdio)
- **Edit cascade**: 3-strategy file editing (exact, fuzzy, line-based)
- **Context compaction**: Auto-summarize old messages near context limit
- **POLY.md / MEMORY.md**: Project instructions + persistent memory
- **Skills system**: Load custom skills from `~/.poly/skills/`
- **Hooks**: Pre/post tool hooks with Go templates
- **OAuth + API key auth**: PKCE flows for Anthropic, OpenAI, Google
- **Sandbox mode**: Container isolation (podman/docker) with auto-pull
- **Path validation**: File tools restricted to workspace
- **30+ slash commands**: `/compact`, `/rewind`, `/stats`, `/mcp`, `/help`, ...
- **Multi-line input**: Shift+Enter for newlines, auto-grow (1-5 lines)
- **Session management**: Auto-save, resume, fork, export (Markdown/JSON)
- **Hybrid shell mode**: `ls -la | @claude explain`
- **Prompt caching**: Anthropic ephemeral cache on system prompt

## Quick Start

```bash
# Build
make build

# Run
./poly

# Or install
make install
```

## Setup at 42

```bash
bash scripts/setup-42.sh
```

## Configuration

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

Or use OAuth: open the Control Room with `Ctrl+D` and connect interactively.

## Usage

```
@claude what is this project?        # Direct to Claude
@gpt explain this function           # Direct to GPT
@all compare Go vs Rust               # Cascade: all providers respond
ls -la | @claude find the largest     # Shell pipe mode
/compact                              # Summarize old context
/rewind 4                             # Remove last 4 messages
/stats                                # Token count + cost
/help compact                         # Help for a command
Ctrl+K                                # Command palette
Ctrl+D                                # Control Room (providers)
Ctrl+O                                # Model picker
```

## Tools

| Category | Tools |
|----------|-------|
| **Files** | `read_file`, `write_file`, `edit_file`, `list_files`, `glob`, `grep` |
| **Exec** | `bash` (sandboxable, timeout 60s-10min) |
| **Diffs** | `propose_diff`, `apply_diff`, `reject_diff`, `list_diffs` |
| **Web** | `web_search`, `web_fetch` |
| **Git** | `git_status`, `git_diff`, `git_log` |
| **Utils** | `delegate_task`, `memory_write`, `system_info`, `todos` |
| **MCP** | Any tools from connected MCP servers (auto-namespaced) |

## Development

```bash
make build          # Build binary
make test           # Run tests
make test-coverage  # Coverage report (HTML)
make dev            # Build with race detector
make release        # Optimized build
make lint           # go vet
make fmt            # gofmt
make sandbox-setup  # Pull sandbox container image
```

## Architecture

```
poly/
├── main.go                    # Entry point + flags
├── Makefile                   # Build system
├── scripts/setup-42.sh        # 42 campus setup
├── internal/
│   ├── llm/                   # Provider implementations + system prompt
│   │   ├── anthropic.go       #   Claude (OAuth + API key)
│   │   ├── gpt.go             #   GPT (OAuth + API key)
│   │   ├── gemini.go          #   Gemini (OAuth + API key)
│   │   ├── grok.go            #   Grok (API key)
│   │   ├── custom.go          #   Custom providers (OpenAI/Anthropic/Google format)
│   │   ├── compaction.go      #   Context compaction
│   │   ├── retry.go           #   Retry with exponential backoff
│   │   └── system.go          #   Dynamic system prompt builder
│   ├── tools/                 # Tool implementations + registry
│   │   ├── pathcheck.go       #   Path validation (security)
│   │   └── edit.go            #   Edit cascade (exact/fuzzy/line)
│   ├── mcp/                   # MCP client (JSON-RPC 2.0, auto-reconnect)
│   ├── config/                # Config, POLY.md, MEMORY.md, history
│   ├── tui/                   # Terminal UI (Bubble Tea v2)
│   │   └── components/        #   Header, sidebar, status bar, splash, dialogs
│   ├── permission/            # Bash command classification
│   ├── sandbox/               # Container sandbox (podman/docker)
│   ├── auth/                  # OAuth PKCE + token storage
│   ├── hooks/                 # Pre/post tool hooks
│   ├── skills/                # Skill loader (.md files)
│   ├── session/               # Session persistence + export
│   ├── shell/                 # Hybrid shell mode
│   └── theme/                 # Catppuccin Mocha theme
├── research/                  # Competitor analysis (reference)
└── *.md                       # Orchestrator visions (Claude/Gemini/Mistral)
```

## Stats

- **~24K lines** of Go across **131 files**
- **8 test files** with **60+ test cases**
- **21+ built-in tools** + unlimited MCP tools
- **4 native providers** + unlimited custom

## Requirements

- Go 1.25.6+
- API keys or OAuth for desired providers
- Optional: podman/docker for sandbox mode

## License

MIT
