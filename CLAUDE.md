# Poly — AI Development Guide

> This file is the source of truth for any AI working on this codebase.
> Read this FIRST before making any changes.

## Project

Multi-AI terminal client in Go. Routes prompts to Claude, GPT, Gemini, Grok, Copilot, Ollama from a single TUI. Full agentic tool use (21 tools), MCP support, session management.

- **Language:** Go 1.25
- **TUI:** Bubble Tea v2 + Lip Gloss v2, Catppuccin Mocha theme
- **Repo:** `git@github.com:ruipedro-pinheiro/Poly.git` (branch: `main`)
- **Status:** Beta (v0.6.x), preparing for public launch

## Critical Rules

### 1. Lint before push — ALWAYS
```bash
go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run ./...
```
This is the EXACT command CI uses (v1 path, NOT v2). Multiple commits have broken CI from lint failures. Run this locally before every push. No exceptions.

### 2. Zero hardcoded model names in Go code
All model detection is config-driven via prefix matching:
- `config.ProviderConfig.ReasoningModels []string`
- `config.ProviderConfig.ReasoningEffortModels []string`
- `IsReasoningModel(providerID, model)` / `SupportsReasoningEffort(providerID, model)` in `provider.go`

If you need to add a new model, update `DefaultConfig` in `config/config.go` — NOT the provider logic.

### 3. No secrets in code or errors
- `security.SanitizeResponseBody()` for HTTP error messages (never dump raw response bodies)
- `security.SafeEnv()` for subprocess environments (strips API keys/tokens)
- All provider `Stream()` goroutines have `recover()` to prevent stack trace token leaks

### 4. errcheck is enforced
All error returns must be handled. Use `_ = fn()` for intentional fire-and-forget with a comment if non-obvious. Test files are excluded from errcheck.

## Architecture

```
internal/
├── llm/           # LLM providers + agentic loop
│   ├── oai_base.go     # Shared base for OpenAI-compatible (GPT/Grok/Copilot)
│   ├── anthropic.go    # Claude (own SSE format)
│   ├── gemini.go       # Gemini (own SSE format, public API + Code Assist)
│   ├── ollama.go       # Local models (NDJSON, not SSE)
│   ├── custom.go       # User-defined providers (OpenAI/Anthropic/Google format)
│   ├── provider.go     # Provider interface, registry, IsReasoningModel
│   ├── openai_types.go # Shared OAI request/response types
│   ├── pricing.go      # Per-model pricing (longest-prefix-match)
│   ├── system.go       # Dynamic system prompt, GetDefaultModel, GetMaxToolTurns
│   └── retry.go        # MaxRetries, ShouldRetry, RetryDelay
├── tools/         # 21 built-in tools + registry + approval system
├── mcp/           # MCP client (JSON-RPC 2.0, stdio, auto-reconnect)
├── tui/           # Terminal UI (Bubble Tea)
├── config/        # Config loading, POLY.md, MEMORY.md, command history
├── auth/          # OAuth PKCE (Anthropic/OpenAI/Google) + Device Flow (Copilot) + token storage
├── security/      # Blocklist, path validation, SanitizeResponseBody, SafeEnv
├── sandbox/       # Container isolation (podman/docker)
├── session/       # Session persistence + fork + export
├── permission/    # Bash command classification
├── hooks/         # Pre/post tool execution hooks
├── skills/        # Skill loader (.md → system prompt)
├── shell/         # Hybrid shell pipe mode
└── theme/         # Catppuccin Mocha palette
```

### Key design patterns

**OAIBaseProvider** (`oai_base.go`): GPT, Grok, Copilot all embed `OAIBaseProvider` which implements the full `Provider` interface. Provider-specific behavior via struct fields:
- `endpoint` — API URL
- `setHeaders` — custom HTTP headers (nil = default Bearer auth)
- `handleStreamError` — error recovery (Copilot 401 token refresh)
- `hasReasoningContent` — parse `reasoning_content` from SSE (Grok)
- `alwaysUseReasoningTokens` — auto-reasoning models (Grok)

**Tool system** (`tools/`): Each tool implements `Tool` interface (`Name`, `Description`, `Parameters`, `Execute`). Registered in global registry. The agentic loop in each provider calls tools and feeds results back.

**Config-driven reasoning**: `IsReasoningModel()` checks `config.ProviderConfig.ReasoningModels` using prefix matching. This controls `max_completion_tokens` vs `max_tokens`. `SupportsReasoningEffort()` controls whether `reasoning_effort` param is sent.

## Milestone: v0.7.0 — Public Beta

Track progress at: https://github.com/ruipedro-pinheiro/Poly/milestone/1

### Completed
- Phase 1: Config-driven reasoning + 3 P0 bug fixes (#14)
- Phase 2: OAIBaseProvider refactor, -786 lines (#15)
- Security hardening: error sanitization, env filtering, panic recovery (#16)
- errcheck enabled in CI, 70+ violations fixed (#7)
- README beta disclaimer, issue templates, CONTRIBUTING.md

### Open — Refactor phases
- #8  Phase 3: Auth generic helpers (-280 lines)
- #9  Phase 4: Theme single source of truth (-85 lines)
- #10 Phase 5: TUI dedup (-95 lines)
- #11 Phase 6: Shared HTTP retry (-230 lines)
- #12 Phase 7: Dead code cleanup (-50 lines)

### Open — Security
- #13 Sandbox hardening (capabilities, resource limits, container escape)

## Build & Test

```bash
make build          # Compile binary
make dev            # Build with race detector
make test           # Run all tests
make ci             # Full CI locally (build + vet + test -race)
make fmt            # gofmt

# Lint (EXACT CI command):
go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run ./...
```

## Maintainer

Pedro (rpinheir @ 42 Lausanne). Communication: direct, frank, in French. No corporate BS. Action over talk.
