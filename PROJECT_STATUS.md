# Poly-Go - Project Status (Feb 20, 2026)

## Summary

Multi-AI collaborative TUI en Go. v0.4.0 (Table Ronde) livree.
~24K lines Go, 131 files, 21+ tools, 4 native providers + custom illimite.

## Ce qui marche

- Multi-provider routing (@claude, @gemini, @gpt, @grok, custom)
- Table Ronde v0.4.0 : inter-IA @mentions, conversation multi-voix, rounds configurables
- Agentic tool loops (4 native providers, max 50 turns)
- 21+ built-in tools + MCP tool bridge (JSON-RPC 2.0, auto-reconnect)
- Context compaction (auto at 80% + /compact)
- POLY.md / MEMORY.md / Skills dans le system prompt
- Prompt caching (Anthropic ephemeral cache_control)
- Edit cascade (exact -> fuzzy -> line-based)
- Path validation sur tous les file tools
- Sandbox mode (podman/docker, --network=none, --read-only)
- 30+ slash commands avec tab completion
- Multi-line input (Shift+Enter, auto-grow 1-5 lines)
- Input history persiste entre sessions
- OAuth PKCE (Anthropic, OpenAI, Google) + API key auth
- Session management (auto-save, resume, fork, export MD/JSON)
- Streaming avec live tokens/s + elapsed time
- TUI clean : header 1 ligne, sidebar supprimee, InfoPanel overlay (Ctrl+I)
- Thinking blocks collapses par defaut
- Tool calls batch summary
- Help dialog scrollable
- 60+ unit tests, CI/CD GitHub Actions

## Ce qui ne marche PAS

### Critique

- **Custom providers sans agentic loop** : ils recoivent les tools mais ne les executent jamais (single-turn). C'est le pire bug du projet — le selling point "N providers illimites" est casse.
- **Retry/backoff declare mais pas utilise** : retry.go existe avec la logique, mais aucun provider ne l'appelle.

*Note : Markdown rendering (Glamour) et syntax highlighting (Chroma/catppuccin-mocha) SONT implementes dans le code (markdown.go, views.go). A verifier visuellement en runtime.*

### Important

- **Custom providers sans images** : architecture map[string]string au lieu de interface{}
- **Custom providers sans token tracking** : InputTokens/OutputTokens toujours a 0, donc pas de cout
- **Test coverage ~15-25%** : TUI a 0.5%, providers streaming a 0%, auth a 0%
- **Send() pas implemente** : tous les providers retournent "not implemented", streaming-only
- **strings.Title() deprecie** : utilise dans modelpicker.go et update_keys.go
- **Grok : extended thinking ignore** : le parametre est envoye mais rien ne se passe

### Moyen

- **Code duplique** : gradientText() x3 (header, sidebar, splash), timeAgo() x2
- **Dialog system partiellement wire** : nouveaux composants existent mais ancien viewState toujours actif
- **Help incomplete** : mentions de providers hardcodees
- **DefaultContextWindow fixe a 200K** : pas adapte au provider (Gemini = 1M)
- **Dependencies en RC/beta** : BubbleTea v2 (rc.2), Lipgloss v2 (beta.3)

## Positionnement competitif

### Forces uniques
- Go + Bubble Tea + MIT (vs Crush = Charm License restrictive)
- Multi-voix dans le meme chat (vs OpenCode/Aider = switch de provider)
- Table Ronde (inter-IA @mentions) — personne d'autre ne fait ca
- Custom provider system (meme casse, l'idee est la)
- Per-session cost tracking
- YOLO mode, session forking, thinking toggle

### Gaps vs concurrents
- Pas de file tree navigation
- Pas de git integration profonde (vs Claude Code)
- Pas de sandbox gVisor (vs Gemini CLI)
- Pas de plugins/extensions

## Priorites reelles (avis audit externe)

1. **P0** : Custom providers agentic loop — sans ca, le selling point multi-provider est un mensonge
2. **P1** : Custom providers images + token tracking
3. **P1** : Activer retry/backoff (le code existe, faut le brancher)
4. **P1** : Test coverage a 40%+ (surtout streaming et providers)
5. **P1** : Verifier le rendu markdown/syntax highlighting en runtime (code present, pas teste visuellement)
6. **P2** : Supprimer la duplication de code
7. **P2** : Migrer vers les versions stables de BubbleTea/Lipgloss quand disponibles

## Key Architecture Files

- main.go — Entry + flags + init sequence
- internal/llm/system.go — Dynamic system prompt
- internal/llm/anthropic.go — Claude provider (OAuth, caching, agentic loop)
- internal/tui/model.go — Main TUI model
- internal/tui/commands.go — 30+ slash commands
- internal/tui/streaming.go — Stream management
- internal/tools/registry.go — Tool registry (21+ tools)
- internal/mcp/manager.go — MCP manager (auto-reconnect, tool bridge)
