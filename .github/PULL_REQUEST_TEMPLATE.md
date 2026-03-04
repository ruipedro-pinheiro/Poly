## What

<!-- One-sentence summary of the change -->

## Why

<!-- What problem does this solve? Link to issue if applicable: Fixes #123 -->

## How

<!-- Brief description of the approach. Mention any trade-offs or design decisions. -->

## Checklist

- [ ] `make build` passes
- [ ] `make test` passes (all tests green)
- [ ] `make fmt` applied (no formatting diff)
- [ ] `go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run ./...` passes
- [ ] No hardcoded model names in Go code (use config-driven detection)
- [ ] No API keys, tokens, or secrets in the diff
