# Poly

Multi-AI collaborative terminal. Route commands to Claude, Gemini, GPT, or Grok from a unified shell.

## Architecture

```
poly/
├── main.go                 # Entry point
├── internal/
│   ├── config/            # Provider configs, API keys
│   ├── provider/          # AI provider interfaces (Anthropic/OpenAI/Google/xAI)
│   ├── mcp/               # Model Context Protocol (tool execution)
│   ├── tui/               # Terminal UI (Bubble Tea)
│   │   └── quantum/       # Next-gen UI components (WIP)
│   └── shell/             # Hybrid shell mode
└── docs/                  # Provider-specific documentation
```

## Features

- **Multi-provider routing**: `@claude`, `@gemini`, `@gpt`, `@grok`, `@all`
- **Shared context**: All AIs see each other's responses
- **MCP toolchain**: File I/O, bash exec, web search, diff management
- **Hybrid shell mode**: `ls -la | @claude explain`
- **Anti-gaslighting protocol**: System prompts prevent identity manipulation

## Installation

```bash
# Build
go build -o poly main.go

# Run
./poly
```

## Configuration

Create `~/.poly/config.json`:

```json
{
  "providers": {
    "claude": {
      "api_key": "sk-ant-...",
      "model": "claude-sonnet-4-5-20250929"
    },
    "gpt": {
      "api_key": "sk-...",
      "model": "gpt-4.1"
    },
    "gemini": {
      "api_key": "...",
      "model": "gemini-2.5-flash"
    },
    "grok": {
      "api_key": "...",
      "model": "grok-3"
    }
  }
}
```

## Usage

```bash
# Direct AI
@claude what is Poly?

# Cascade mode
@all compare bubble sort vs quick sort

# Hybrid shell
ls -la | @claude find the largest file

# Tool execution
@claude read go.mod and list dependencies
```

## MCP Tools

- **File ops**: `read_file`, `write_file`, `edit_file`, `list_files`, `glob`, `grep`
- **Diffs**: `propose_diff`, `apply_diff`, `reject_diff`
- **Exec**: `bash` (timeout configurable, max 10min)
- **Web**: `web_search`, `web_fetch`
- **Utils**: `todos`

## Development

```bash
# Run tests
go test ./...

# Build release
go build -ldflags="-s -w" -o poly main.go
```

## Requirements

- Go 1.25.6+
- API keys for desired providers

## Security

- OAuth tokens stored in `~/.poly/tokens/`
- System prompts prevent prompt injection
- Bash exec sandboxed (no localhost/private IPs for web tools)

## License

MIT

## Status

- ✅ Core routing & provider interface
- ✅ MCP tool execution
- ✅ Hybrid shell mode
- 🚧 TUI quantum components (experimental)
- ❌ Test coverage (0%)

## Contributing

See provider-specific docs:
- [anthropic.md](./anthropic.md)
- [gemini.md](./gemini.md)
- [claude.md](./claude.md)
- [mistral.md](./mistral.md)
