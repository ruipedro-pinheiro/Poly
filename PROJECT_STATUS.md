# Poly-Go - Project Status (Feb 13, 2026)

## Summary
Multi-AI collaborative TUI. 9 commits pushed today across 5 phases.
~24K lines Go, 131 files, 21+ tools, 4 native providers + unlimited custom.

## Completed Features
- Multi-provider routing (@claude, @gemini, @gpt, @grok, @all)
- Agentic tool loops (all 4 native + custom providers in 3 formats)
- 21+ built-in tools + MCP tool bridge (JSON-RPC 2.0, auto-reconnect)
- Context compaction (auto at 80% + /compact command)
- POLY.md / MEMORY.md / Skills in system prompt
- Prompt caching (Anthropic ephemeral cache_control)
- Edit cascade (exact -> fuzzy -> line-based)
- Path validation on all file tools
- Sandbox mode (podman/docker, auto-pull, --network=none, --read-only)
- 30+ slash commands with tab completion
- Multi-line input (Shift+Enter, auto-grow 1-5 lines)
- Input history persisted across sessions
- OAuth PKCE (Anthropic, OpenAI, Google) + API key auth
- Session management (auto-save, resume, fork, export MD/JSON)
- Streaming with live tokens/s + elapsed time (500ms tick)
- Retry with exponential backoff on all providers
- Help view (categorized, /help <cmd>)
- Makefile (build/test/coverage/release/setup-42)
- 60+ unit tests
- Hybrid shell mode (ls -la | @claude explain)

## Known Issues
- Cascade mode (@all) works but UX is mediocre
- TUI needs visual redesign (Pedro's next priority)
- Shift+Enter might not work in all terminal emulators
- Test coverage is only on config/llm/permission (0% on providers/streaming/tools)

## Next Steps (Brainstorm with Pedro)
1. **TUI Redesign** - Pedro wants to discuss what/how
   - Consider Huh (Charm.sh) for forms
   - Better message layout, spacing, animations
   - Better splash screen
   - Dynamic themes
2. **Design Plugins** available:
   - frontend-design (official Anthropic skill)
   - Figma MCP server
   - Huh, VHS, Freeze from Charm.sh
3. **More tests** - providers, streaming, tools
4. **Docker sandbox** improvements - custom image with Go + build tools

## Git Log (today)
```
aa14cfd Clean up: remove 4 outdated .md files, rewrite README
31b3d68 Improve sandbox: auto-pull, network isolation, clear errors
7edfe46 Phase 4: multi-line input, tests, Makefile, help polish
7d4ca0b Phase 5: history persist, MCP reconnect, path validation, edit cascade
1f2930a Phase 3: wire MCP, skills prompt, new commands, fix streaming cancel
9daa1a5 Phase 2: fix build, audit & harden all 4 implementation tasks
efabb31 Add setup script for 42 machines
bd8a230 Merge remote main
60f0851 Initial commit: Poly-go v0.2.0
```

## Key Architecture Files
- main.go - Entry + flags + init sequence
- internal/llm/system.go - Dynamic system prompt (6 sections)
- internal/llm/anthropic.go - Claude provider (OAuth, prompt caching, agentic loop)
- internal/tui/model.go - Main TUI model
- internal/tui/commands.go - 30+ slash commands
- internal/tui/streaming.go - Stream management
- internal/tools/registry.go - Tool registry (21+ tools)
- internal/mcp/manager.go - MCP manager (auto-reconnect, tool bridge)

## Workflow That Works
- Agent teams: 4 opus agents in parallel per phase
- Each agent gets 1 focused task with clear file list
- Build verification after each phase
- Commit + push after all agents complete
