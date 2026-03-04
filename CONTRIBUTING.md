# Contributing to Poly

## Setup

```bash
# Clone
git clone https://github.com/ruipedro-pinheiro/Poly.git
cd Poly

# Build
make build

# Build with race detector (for development)
make dev

# Run tests
make test

# Run CI checks locally (build + vet + test -race)
make ci

# Format code
make fmt

# Lint (same command as CI)
go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run ./...
```

Requirements: **Go 1.25+**. Optional: `podman` or `docker` for sandbox testing.

## Architecture

```
internal/
├── llm/          # LLM providers + agentic loop
├── tools/        # 21 built-in tools + registry
├── mcp/          # MCP client (JSON-RPC 2.0, stdio)
├── tui/          # Terminal UI (Bubble Tea v2)
├── config/       # Config loading, POLY.md, MEMORY.md
├── auth/         # OAuth PKCE + Device Flow + token storage
├── security/     # Blocklist, path validation, secret sanitization
├── sandbox/      # Container isolation (podman/docker)
├── permission/   # Bash command classification
├── session/      # Session persistence + export
├── hooks/        # Pre/post tool hooks
├── skills/       # Skill loader
├── shell/        # Hybrid shell pipe mode
├── theme/        # Catppuccin Mocha palette
└── types/        # Shared types (ToolCall, ToolResult)
```

### Key patterns

**Providers** (`internal/llm/`): Each LLM provider implements the `Provider` interface. OpenAI-compatible providers (GPT, Grok, Copilot) share a common base in `oai_base.go` — concrete providers only define endpoint, headers, and behavioral hooks. Anthropic and Gemini have their own implementations due to different API formats.

**Tools** (`internal/tools/`): Each tool implements the `Tool` interface (`Name`, `Description`, `Parameters`, `Execute`). Tools are registered in a global registry. The agentic loop calls tools and feeds results back to the LLM.

**Config-driven model detection**: No model names are hardcoded in Go code. Reasoning model detection, pricing, and defaults all come from `config.ProviderConfig` with prefix matching. If you add a new model, update the config defaults — not the provider logic.

## Rules

1. **`golangci-lint` must pass.** CI uses `go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run ./...` (v1 path). Two commits already broke CI from lint failures. Run it locally before pushing.

2. **No hardcoded model names in Go code.** All model detection is config-driven via prefix matching in `config.ProviderConfig.ReasoningModels` and `ReasoningEffortModels`.

3. **No API keys or tokens in code or commits.** Use environment variables or the OAuth flow. The `security.SafeEnv()` function strips sensitive vars from child processes — use it if you spawn subprocesses.

4. **Tests must pass.** Run `make test` before opening a PR. If you change pricing, update `pricing_test.go`.

5. **Keep it modular.** The codebase went through a deduplication audit. Don't re-introduce copy-paste patterns. If three providers need the same logic, it belongs in a shared base.

## Submitting changes

1. Fork the repo
2. Create a branch from `main`
3. Make your changes
4. Run `make ci` + `golangci-lint`
5. Open a PR with the [template](.github/PULL_REQUEST_TEMPLATE.md)
