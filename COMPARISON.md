# POLY-GO vs CONCURRENCE : Rapport de Comparaison Exhaustif

> Compilé le 10 février 2026 par 4 agents de recherche en parallèle
> Sources : code source Crush (75k lignes), docs Claude Code, docs Gemini CLI, audit Poly-go

---

## RÉSUMÉ EXÉCUTIF

| Métrique | Poly-go | Claude Code | Gemini CLI | Crush |
|----------|---------|-------------|------------|-------|
| **Lignes de code** | ~17.8k | N/A (closed) | ~30k (TS) | ~75k |
| **Langage** | Go | TypeScript | TypeScript | Go |
| **Tools** | 13 | 15+ | 13 | 29 |
| **Providers** | 4 | 1 (Claude) | 1 (Gemini) | 11 |
| **MCP** | Stub | Complet | Complet | Complet |
| **LSP** | Non | Non | Non | Oui |
| **Sub-agents** | Non | Oui (6 types) | Oui (2 types) | Oui |
| **Agent Teams** | Non | Oui (expérimental) | Non | Non |
| **Hooks** | Non | 14 events | 11 events | Non |
| **Skills** | Non | Oui | Oui | Oui |
| **Plugins/Extensions** | Non | 834+ plugins | 100+ extensions | Non |
| **Context compression** | Non | Auto ~95% | Configurable | Auto (200k) |
| **Sandbox** | Non | OS-level | Docker/Podman/Seatbelt | Non |
| **IDE Integration** | Non | VS Code + JetBrains + Desktop | VS Code | Non |
| **Open Source** | Non (perso) | Non | Oui (Apache 2.0) | Non |
| **Prix** | Gratuit (self-hosted) | $20-200/mois | Gratuit (1000 req/j) | Gratuit (self-hosted) |

---

## 1. SYSTÈME D'AGENTS / ORCHESTRATION

### Ce que Poly a

| Feature | Status | Détails |
|---------|--------|---------|
| Cascade @all | ✅ Implémenté | Envoie à tous les providers, cheapest first, puis reviewers |
| Reviewer mode | ✅ Basique | Les reviewers voient la réponse du responder et disent "✓" ou corrigent |
| Parallélisme | ✅ Partiel | Responder séquentiel, reviewers en parallèle |

### Ce que Poly n'a PAS

| Feature | Claude Code | Gemini CLI | Crush | Priorité |
|---------|-------------|------------|-------|----------|
| **Sub-agents (spawn isolés)** | 6 types built-in + custom (.md) | Codebase Investigator + Dispatcher | agent_tool.go spawn parallèle | 🔴 CRITIQUE |
| **Agent Teams (peer-to-peer)** | TeammateTool, 13 ops, mailbox, shared tasks, delegate mode | Non | Non | 🟡 HAUTE |
| **Consensus entre IAs** | Non (mais teams permettent la coordination) | Non | Non | 🟡 HAUTE |
| **Plan Mode** | Shift+Tab x2, read-only exploration, plan approval | Non | Non | 🟡 HAUTE |
| **Delegate Mode** | Lead restreint à coordination-only (spawn, message, tasks) | Non | Non | 🟢 MOYENNE |
| **Background agents** | Oui, auto-deny permissions non approuvées | Non | Sessions parallèles | 🟢 MOYENNE |
| **Agent resume** | Reprendre un agent avec son contexte complet | Non | Non | 🟢 MOYENNE |
| **Quality gates** | Hooks TeammateIdle + TaskCompleted bloquent si critères pas remplis | Non | Non | 🟢 MOYENNE |

**Verdict** : Poly a un orchestrateur séquentiel basique. Il manque TOUTE la couche agent (spawn, isolation, communication, coordination). C'est le gap le plus critique.

---

## 2. TOOLS

### Comparaison complète

| Tool | Poly | Claude Code | Gemini CLI | Crush |
|------|------|-------------|------------|-------|
| Read file | ✅ | ✅ (+images, PDF, notebooks) | ✅ (+read_many_files) | ✅ (view, 5MB max) |
| Write file | ✅ | ✅ | ✅ | ✅ (atomic + diff) |
| Edit file | ✅ | ✅ (exact string replace) | ✅ (find-replace) | ✅ (smart context) |
| Multi-edit | ✅ | ✅ | Non | ✅ |
| Glob | ✅ | ✅ | ✅ | ✅ (100 max) |
| Grep | ✅ | ✅ (ripgrep, multiline) | ✅ (search_file_content) | ✅ |
| Bash | ✅ | ✅ (600s timeout, background) | ✅ | ✅ (30k limit, bg jobs) |
| Web fetch | ✅ | ✅ (HTML→MD, cache 15min) | ✅ | ✅ (sub-agent sans permissions) |
| Web search | ✅ | ✅ (domain filtering) | ✅ (Google Search natif) | ✅ |
| Todos | ✅ | ✅ (TaskCreate/Update/List) | ✅ (write_todos) | ✅ (session-based) |
| Diff propose | ✅ | Non (Edit direct) | Non | Non |
| Diff apply | ✅ | Non | Non | Non |
| List files | ✅ | Non (Glob suffit) | ✅ (list_directory) | ✅ (ls, 1000 max) |

### Tools que Poly n'a PAS

| Tool | Où ça existe | Description | Priorité |
|------|-------------|-------------|----------|
| **NotebookEdit** | Claude Code | Éditer cellules Jupyter (.ipynb) | 🟢 BASSE |
| **Skill** | Claude Code, Gemini CLI | Invoquer un skill par nom | 🟡 HAUTE |
| **AskUserQuestion** | Claude Code | Questions structurées avec options | 🟢 MOYENNE |
| **EnterPlanMode / ExitPlanMode** | Claude Code | Mode exploration read-only | 🟡 HAUTE |
| **Task (spawn agent)** | Claude Code | Créer un sub-agent isolé | 🔴 CRITIQUE |
| **TeammateTool** | Claude Code | 13 ops de coordination d'équipe | 🟡 HAUTE |
| **save_memory** | Gemini CLI | Persister des faits en GEMINI.md | 🟢 MOYENNE |
| **codebase_investigator** | Gemini CLI | Agent autonome d'analyse de codebase | 🟡 HAUTE |
| **LSP diagnostics** | Crush | Erreurs/warnings IDE en temps réel | 🟡 HAUTE |
| **LSP references** | Crush | Go-to-definition, find references | 🟡 HAUTE |
| **LSP restart** | Crush | Redémarrer un LSP serveur | 🟢 BASSE |
| **Download** | Crush | Télécharger fichiers (600s timeout) | 🟢 BASSE |
| **Sourcegraph** | Crush | Recherche de code cross-repo | 🟢 BASSE |
| **Job output / Job kill** | Crush | Gestion de jobs background | 🟡 HAUTE |
| **Ripgrep (rg)** | Crush | Backup search avec ripgrep natif | 🟢 BASSE |

**Verdict** : Poly a un bon set de base (13 tools), mais il manque les outils d'agent (Task, Skill), les outils LSP (diagnostics), et la gestion de jobs background.

---

## 3. MCP (Model Context Protocol)

### Ce que Poly a

| Feature | Status |
|---------|--------|
| JSON-RPC 2.0 transport | ✅ |
| Stdio-based connections | ✅ |
| Initialize handshake | ✅ |
| Tool discovery | ✅ |
| Tool execution | ✅ |
| Multi-server manager | ✅ |

### Ce que Poly n'a PAS

| Feature | Claude Code | Gemini CLI | Crush | Priorité |
|---------|-------------|------------|-------|----------|
| **MCP wired into main flow** | Oui, tools apparaissent comme `mcp__server__tool` | Oui, `/mcp list` | Oui, permission gating | 🔴 CRITIQUE |
| **Tool Search / Lazy Loading** | Réduit contexte de 95% | Non | Non | 🟡 HAUTE |
| **HTTP/SSE transport** | Oui | Oui (SSE) | Oui (HTTP, SSE) | 🟡 HAUTE |
| **Auto-reconnect** | Non | Non | Oui (ping + retry) | 🟢 MOYENNE |
| **OAuth pre-configured** | Oui (ex: Slack) | Non | Non | 🟢 BASSE |
| **MCP dans sub-agents** | Oui (par nom ou inline) | Non | Non | 🟢 MOYENNE |
| **Config multi-scope** | Managed > Project > Local > User | Project > Global | Project-local | 🟢 MOYENNE |
| **Disabled tools per MCP** | Non | Non | Oui | 🟢 BASSE |
| **Docker MCP servers** | Non | Oui | Non | 🟢 BASSE |
| **FastMCP integration** | Non | Oui (Python) | Non | 🟢 BASSE |

**Verdict** : Le code MCP existe dans Poly mais n'est PAS branché dans le flow principal. C'est un stub. Il faut le wirer pour que les tools MCP soient disponibles comme les autres tools.

---

## 4. PROVIDERS / MODÈLES

### Ce que Poly a

| Feature | Status |
|---------|--------|
| 4 providers (Claude, GPT, Gemini, Grok) | ✅ |
| Custom providers via JSON | ✅ |
| OAuth (Claude, Gemini) | ✅ |
| API key (GPT, Grok) | ✅ |
| Model variants (default/fast/think/opus) | ✅ |
| Cost tier ordering | ✅ |
| Streaming SSE | ✅ |
| Agentic loop (50 turns max) | ✅ |
| Thinking mode display | ✅ |

### Ce que Poly n'a PAS

| Feature | Claude Code | Gemini CLI | Crush | Priorité |
|---------|-------------|------------|-------|----------|
| **AWS Bedrock** | Oui | Non | Oui | 🟢 BASSE |
| **Google Vertex AI** | Oui | Oui | Oui | 🟢 BASSE |
| **Microsoft Foundry** | Oui | Non | Non | 🟢 BASSE |
| **Azure OpenAI** | Non | Non | Oui | 🟢 BASSE |
| **OpenRouter** | Non | Non | Oui (exacto routing) | 🟢 MOYENNE |
| **GitHub Copilot** | Non | Non | Oui (device code OAuth) | 🟢 MOYENNE |
| **Ollama/LM Studio** | Non | Non | Oui (OpenAI-compatible) | 🟡 HAUTE |
| **Model auto-selection** | Non | Oui (Auto mode) | Oui (large/small) | 🟡 HAUTE |
| **Token refresh automatique** | Partiel | Oui | Oui (401 → retry) | 🟡 HAUTE |
| **Provider-specific caching** | Non | Non | Oui (Anthropic ephemeral cache) | 🟢 MOYENNE |
| **Reasoning effort levels** | Non | Oui (thinkingBudget) | Oui (OpenAI reasoning) | 🟢 MOYENNE |
| **Provider prompt prefixes** | Non | Non | Oui (system prompt per provider) | 🟢 BASSE |

**Verdict** : Poly a un bon setup de providers. Les gaps principaux sont : pas d'Ollama/LM Studio (local models), pas d'auto-selection model, et le token refresh est fragile.

---

## 5. PERMISSION / SÉCURITÉ

### Ce que Poly a

| Feature | Status |
|---------|--------|
| Allow / Ask levels | ✅ |
| Per-call approval | ✅ |
| Per-session auto-approve | ✅ |
| YOLO mode | ✅ |
| Approval dialog TUI | ✅ |
| Dangerous command blocking | ✅ |

### Ce que Poly n'a PAS

| Feature | Claude Code | Gemini CLI | Crush | Priorité |
|---------|-------------|------------|-------|----------|
| **Deny rules** | Oui (always overrides allow) | Non | Non | 🟢 MOYENNE |
| **Glob/regex matching** | `Bash(npm run *)`, `Read(./.env)` | `ShellTool(git status)` | Non | 🟡 HAUTE |
| **Permission modes** | 6 modes (default, acceptEdits, plan, dontAsk, bypass, delegate) | 3 modes (default, auto_edit, yolo) | YOLO flag | 🟢 MOYENNE |
| **Trust hierarchy** | Managed > CLI > Local > Project > User | Project > Global | Non | 🟢 BASSE |
| **Sandbox (filesystem)** | Oui (directories, network isolation) | Docker/Podman/Seatbelt | Non | 🟡 HAUTE |
| **Sandbox auto-activate in YOLO** | Non | Oui | Non | 🟢 MOYENNE |
| **Enterprise lockdown** | Oui (managed settings override tout) | Non | Non | 🟢 BASSE |
| **Root directory restriction** | Non | Oui (CWD scoping) | Non | 🟡 HAUTE |

**Verdict** : Poly a les bases. Il manque le sandboxing filesystem, les règles de permission granulaires (glob patterns), et le root directory scoping.

---

## 6. SESSION / CONTEXTE

### Ce que Poly a

| Feature | Status |
|---------|--------|
| Auto-save sessions | ✅ |
| Session browser | ✅ |
| Session load/create/delete | ✅ |
| Export to JSON | ✅ |
| Session title auto-gen | Non |

### Ce que Poly n'a PAS

| Feature | Claude Code | Gemini CLI | Crush | Priorité |
|---------|-------------|------------|-------|----------|
| **Context auto-compression** | ~95% capacity trigger | Configurable threshold | 200k buffer trigger | 🔴 CRITIQUE |
| **Manual /compact** | Oui + instructions optionnelles | /compress | Non | 🔴 CRITIQUE |
| **/rewind** | /rewind | Oui + file revert option | Non | 🟡 HAUTE |
| **/restore (file checkpoint)** | Non | Oui | Non | 🟡 HAUTE |
| **Session forking** | Oui (branch from history) | /chat save tag | Non | 🟢 MOYENNE |
| **Session from PR** | `--from-pr 123` | Non | Non | 🟢 BASSE |
| **Auto title generation** | Non | Non | Oui (small model) | 🟢 MOYENNE |
| **Token usage /stats** | Non | /stats | Non | 🟢 MOYENNE |
| **Session cleanup policy** | cleanupPeriodDays | maxAge + maxCount | Non | 🟢 BASSE |
| **Chat export Markdown** | Non | /chat export (MD + JSON) | Non | 🟢 MOYENNE |

**Verdict** : Le gap le plus critique est l'ABSENCE TOTALE de context compression. Quand le contexte explose, Poly crash. Claude Code et Gemini CLI gèrent ça automatiquement.

---

## 7. HOOKS / ÉVÉNEMENTS

### Ce que Poly a

**RIEN.** Zéro système de hooks.

### Comparaison Claude Code vs Gemini CLI vs Crush

| Event | Claude Code | Gemini CLI | Crush |
|-------|-------------|------------|-------|
| SessionStart | ✅ | ✅ | Non |
| SessionEnd | ✅ | ✅ | Non |
| UserPromptSubmit | ✅ | Non | Non |
| PreToolUse | ✅ (can block/modify) | ✅ (BeforeTool) | Non |
| PostToolUse | ✅ | ✅ (AfterTool) | Non |
| PostToolUseFailure | ✅ | Non | Non |
| PermissionRequest | ✅ (can allow/deny) | Non | Non |
| SubagentStart | ✅ | ✅ (BeforeAgent) | Non |
| SubagentStop | ✅ (can block) | ✅ (AfterAgent) | Non |
| Stop | ✅ (can block) | Non | Non |
| TeammateIdle | ✅ | Non | Non |
| TaskCompleted | ✅ | Non | Non |
| PreCompact | ✅ | ✅ (PreCompress) | Non |
| Notification | ✅ | ✅ | Non |
| BeforeModel | Non | ✅ | Non |
| AfterModel | Non | ✅ | Non |
| BeforeToolSelection | Non | ✅ | Non |

### Handler types (Claude Code uniquement)

| Type | Description |
|------|-------------|
| **command** | Shell command, JSON via stdin |
| **prompt** | Single-turn LLM evaluation |
| **agent** | Sub-agent multi-turn avec tools |

**Priorité** : 🟡 HAUTE — Les hooks permettent l'automatisation, la validation, le logging, et les quality gates. Sans eux, pas de CI/CD integration ni de workflows custom.

---

## 8. SKILLS / PLUGINS / EXTENSIONS

### Ce que Poly a

**RIEN.** Ni skills, ni plugins, ni extensions.

### Comparaison

| Feature | Claude Code | Gemini CLI | Crush |
|---------|-------------|------------|-------|
| **Skills** | .md avec YAML frontmatter, hot-reload, auto-invocable | Dossiers avec SKILL.md, auto-sélection | SKILL.md dans ~/.config/crush/skills/ |
| **Plugins** | Bundle skills+hooks+agents+MCP+commands | Non | Non |
| **Extensions** | Non | Bundle MCP+GEMINI.md+commands+tool restrictions | Non |
| **Marketplace** | 43 marketplaces, 834+ plugins | 100+ extensions (Figma, Shopify, Snyk...) | Non |
| **Slash commands custom** | Via skills | Via .toml files | Non |
| **Installation** | `/plugin marketplace add` | `gemini extensions install <url>` | Copier dossier |

**Priorité** : 🟡 HAUTE — Les skills sont le moyen d'étendre les capacités sans toucher au code. Les plugins/extensions sont le moyen de distribuer ces extensions.

---

## 9. MÉMOIRE / CONTEXT FILES

### Ce que Poly a

| Feature | Status |
|---------|--------|
| Config file (~/.poly/config.json) | ✅ |
| Session persistence | ✅ |

### Ce que Poly n'a PAS

| Feature | Claude Code | Gemini CLI | Crush | Priorité |
|---------|-------------|------------|-------|----------|
| **Auto-memory** (apprend en travaillant) | ~/.claude/projects/*/memory/ | save_memory tool → GEMINI.md | Non | 🟡 HAUTE |
| **CLAUDE.md / GEMINI.md / CRUSH.md** | Oui (6 scopes) | Oui (3 scopes + subdirs) | Oui (auto-discover copilot-instructions, cursorrules, etc.) | 🟡 HAUTE |
| **@import syntax** | Oui (recursive, 5 hops) | Non | Non | 🟢 MOYENNE |
| **Path-specific rules** | YAML frontmatter `paths` field | Subdirectory GEMINI.md | Non | 🟢 BASSE |
| **Managed policy** | /etc/claude-code/CLAUDE.md | Non | Non | 🟢 BASSE |
| **Context auto-discovery** | Scans CLAUDE.md up from cwd | Scans 200 subdirs | .cursorrules, CLAUDE.md, GEMINI.md, CRUSH.md | 🟡 HAUTE |
| **/memory command** | Oui (ouvre l'éditeur) | /memory add, /memory show | Non | 🟢 MOYENNE |
| **/init command** | Bootstrap CLAUDE.md | Non | Non | 🟢 BASSE |

**Verdict** : Poly n'a AUCUN système de mémoire persistante entre sessions ni de context files auto-découverts. C'est un gap important pour l'utilisabilité.

---

## 10. GIT INTEGRATION

### Ce que Poly a

**RIEN de spécifique.** Git via bash tool uniquement.

### Ce que les autres ont

| Feature | Claude Code | Gemini CLI | Crush |
|---------|-------------|------------|-------|
| **Commit attribution** | `Co-Authored-By: Claude` (configurable) | Non | `assisted-by` / `co-authored-by` / `none` |
| **Git safety protocol** | Intégré (never force push, never amend, etc.) | Non | Non |
| **GitHub Actions** | `@claude` dans PR/issues → CI/CD | `@gemini-cli` dans issues/PRs | Non |
| **PR creation** | Via `gh` CLI | Via shell | Non |
| **Banned git commands** | Oui (destructive ops blocked) | Non | Non |

**Priorité** : 🟢 MOYENNE — L'attribution et le safety protocol sont nice-to-have. Le GitHub Actions est plus avancé.

---

## 11. TUI / INTERFACE

### Ce que Poly a (et qui est bien)

| Feature | Status |
|---------|--------|
| BubbleTea + Lipgloss TUI | ✅ |
| Provider-colored borders | ✅ |
| Thinking blocks (expand/collapse) | ✅ |
| Sidebar (providers, tokens, files, todos) | ✅ |
| Model picker | ✅ |
| Control Room (OAuth/API key) | ✅ |
| Command palette (Ctrl+K) | ✅ |
| Session list | ✅ |
| Help dialog | ✅ |
| Splash screen | ✅ |
| Token/cost tracking | ✅ |
| Response time display | ✅ |
| Image paste | ✅ |
| Approval dialog | ✅ |

### Ce que Poly n'a PAS (TUI)

| Feature | Claude Code | Gemini CLI | Crush | Priorité |
|---------|-------------|------------|-------|----------|
| **Diff viewer (syntax highlight)** | VS Code inline diffs | VS Code companion diffs | Split/unified diff viewer intégré | 🟡 HAUTE |
| **Syntax highlighting code blocks** | Markdown rendering | Rich terminal rendering | Oui | 🟡 HAUTE |
| **Markdown rendering** | Complet | Complet | Partiel | 🟡 HAUTE |
| **Theming system** | Non (monochrome) | Color theme selection | Catppuccin-based | 🟢 MOYENNE |
| **/rewind TUI** | /rewind conversation | Esc Esc → rewind | Non | 🟢 MOYENNE |
| **Output format (JSON)** | `--output-format json/stream-json` | `--output-format json` | Non | 🟢 BASSE |
| **Status line terminal** | /statusline | Non | Non | 🟢 BASSE |
| **Clipboard paste images** | Drag onto terminal | Ctrl+V → auto-saved | Non | 🟢 MOYENNE |

**Verdict** : La TUI de Poly est déjà solide. Les gaps principaux sont le diff viewer, le syntax highlighting, et le markdown rendering.

---

## 12. IDE INTEGRATION

### Ce que Poly a

**RIEN.**

### Ce que les autres ont

| Feature | Claude Code | Gemini CLI | Crush |
|---------|-------------|------------|-------|
| **VS Code extension** | Oui (inline diffs, @-mention, conversations) | Oui (companion, native diffing) | Non |
| **JetBrains plugin** | Oui (IntelliJ, PyCharm, GoLand...) | Non | Non |
| **Desktop app** | Oui (macOS, Windows) | Non | Non |
| **Web app** | claude.ai/code | Non | Non |
| **Slack integration** | @Claude → tasks | Non | Non |
| **Chrome extension** | MCP-based browser debug | Non | Non |

**Priorité** : 🟢 BASSE pour Poly — Poly est un outil terminal-first, les IDE integrations ne sont pas prioritaires.

---

## 13. LSP (Language Server Protocol)

### Ce que Poly a

**RIEN.**

### Ce que Crush a (unique)

| Feature | Description |
|---------|-------------|
| **Diagnostic collection** | Erreurs, warnings, hints en temps réel |
| **Symbol references** | Go-to-definition |
| **Code actions** | Suggestions de fix |
| **Semantic tokens** | Coloration sémantique |
| **Multi-LSP** | Un LSP par langage (gopls, pylsp...) |
| **File type routing** | *.go → gopls, *.py → pylsp |
| **Diagnostic caching** | Versionné par fichier |
| **LSP restart** | Par serveur ou tous |

**Priorité** : 🟡 HAUTE — Les diagnostics LSP permettent à l'agent de voir les erreurs de compilation en temps réel, pas juste après un `go build`. C'est un avantage massif de Crush.

---

## 14. CONTEXT COMPRESSION

### Ce que Poly a

**RIEN.** Quand le contexte explose, c'est game over.

### Comparaison

| Feature | Claude Code | Gemini CLI | Crush |
|---------|-------------|------------|-------|
| **Auto trigger** | ~95% capacity | Configurable threshold | 200k+ = 20k buffer |
| **Manual command** | /compact [instructions] | /compress | Non |
| **Stratégie** | Summarize + preserve critical | XML snapshot (goal, knowledge, file state, plan) | Small model summary |
| **Sub-agent compaction** | Indépendant par agent | Non | Non |
| **Configurable** | CLAUDE_AUTOCOMPACT_PCT_OVERRIDE | model.chatCompression.contextPercentageThreshold | Non |

**Priorité** : 🔴 CRITIQUE — Sans compression, Poly est limité à une seule fenêtre de contexte. Les sessions longues sont impossibles.

---

## 15. STREAMING / PERFORMANCE

### Ce que Poly a

| Feature | Status |
|---------|--------|
| Token-by-token streaming | ✅ |
| Cancel streaming (Ctrl+C) | ✅ |
| SSE parsing | ✅ |
| Thinking blocks streaming | ✅ |
| Tool calls streaming | ✅ |

### Ce que Poly n'a PAS

| Feature | Claude Code | Gemini CLI | Crush | Priorité |
|---------|-------------|------------|-------|----------|
| **Background bash** | run_in_background, TaskOutput | Non | Auto-bg si >1min, job_output/job_kill | 🟡 HAUTE |
| **Ctrl+B (background task)** | Oui | Non | Non | 🟢 MOYENNE |
| **Output format JSON** | --print -p, stream-json | --output-format json | Non | 🟢 BASSE |
| **Headless/CI mode** | Partiel (--print) | --non-interactive + --yolo | Non | 🟢 MOYENNE |
| **Background subagents** | Oui (continue working pendant) | Non | Sessions parallèles | 🟡 HAUTE |

---

## 16. DOCUMENTATION (.md files)

### Audit des fichiers existants

| Fichier | Status | Problème |
|---------|--------|----------|
| `README.md` | ⚠️ OUTDATED | Manque : sessions, thinking, costs, permissions, cascade details |
| `CLAUDE.md` | ✅ Correct | Mineur : MCP Bridge pas mentionné comme non-branché |
| `anthropic.md` | ❌ HORS SUJET | Doc d'orchestrateur infrastructure, PAS un doc Poly |
| `gemini.md` | ❌ HORS SUJET | Doc d'orchestrateur infrastructure, PAS un doc Poly |
| `mistral.md` | ❌ HORS SUJET | Doc d'orchestrateur infrastructure, PAS un doc Poly |
| `SHELL_README.md` | ✅ Excellent | Exhaustif et précis |

### Actions recommandées

1. **SUPPRIMER** `anthropic.md`, `gemini.md`, `mistral.md` — ce sont des concepts d'orchestrateur cloud, rien à voir avec Poly
2. **METTRE À JOUR** `README.md` avec toutes les features actuelles
3. **CRÉER** `POLY.md` (context file pour les agents, équivalent de CLAUDE.md/GEMINI.md)
4. **CRÉER** `PROVIDERS.md` pour documenter le setup custom providers

---

## 17. MATRICE DES FEATURES MANQUANTES (RÉSUMÉ)

### 🔴 CRITIQUE (blocker pour la viabilité)

| # | Feature | Existe dans | Effort estimé |
|---|---------|-------------|---------------|
| 1 | **Context auto-compression** | Claude Code, Gemini CLI, Crush | Moyen (résumer avec small model, injecter summary) |
| 2 | **MCP branché dans le flow** | Claude Code, Gemini CLI, Crush | Moyen (wirer manager → tool registry → LLM) |
| 3 | **Sub-agents (spawn isolés)** | Claude Code, Crush | Gros (architecture agent, context isolation) |

### 🟡 HAUTE (important pour la compétitivité)

| # | Feature | Existe dans | Effort estimé |
|---|---------|-------------|---------------|
| 4 | LSP integration | Crush | Gros (multi-LSP, diagnostics, routing) |
| 5 | Hooks system | Claude Code, Gemini CLI | Moyen (event bus + handler types) |
| 6 | Skills system | Claude Code, Gemini CLI, Crush | Petit (lire .md, injecter dans prompt) |
| 7 | Context files auto-discovery | Claude Code, Gemini CLI, Crush | Petit (scan POLY.md, CLAUDE.md, .cursorrules) |
| 8 | Background jobs | Claude Code, Crush | Moyen (async exec, job tracking) |
| 9 | Diff viewer | Crush | Moyen (syntax highlight, split/unified) |
| 10 | Syntax highlighting code blocks | Tous | Moyen (chroma/glamour) |
| 11 | Markdown rendering | Claude Code, Gemini CLI | Moyen (glamour library) |
| 12 | Plan mode | Claude Code | Moyen (read-only state, plan approval flow) |
| 13 | Ollama/LM Studio support | Crush | Petit (OpenAI-compatible endpoint) |
| 14 | Glob pattern permissions | Claude Code, Gemini CLI | Petit (regex matching sur tool params) |
| 15 | /rewind session | Claude Code, Gemini CLI | Moyen (snapshot messages, revert) |
| 16 | Model auto-selection | Gemini CLI, Crush | Petit (smart/fast routing par complexité) |
| 17 | Token refresh automatique | Claude Code, Gemini CLI, Crush | Petit (retry on 401, refresh token) |
| 18 | Agent Teams (peer-to-peer) | Claude Code | Gros (mailbox, shared tasks, delegate mode) |

### 🟢 MOYENNE/BASSE (nice-to-have)

| # | Feature | Existe dans |
|---|---------|-------------|
| 19 | Sandbox filesystem | Claude Code, Gemini CLI |
| 20 | Git attribution | Claude Code, Crush |
| 21 | IDE integration (VS Code) | Claude Code, Gemini CLI |
| 22 | Plugins/marketplace | Claude Code, Gemini CLI |
| 23 | Headless/CI mode | Claude Code, Gemini CLI |
| 24 | Session export Markdown | Gemini CLI |
| 25 | Auto-memory across sessions | Claude Code, Gemini CLI |
| 26 | Custom slash commands | Claude Code, Gemini CLI |
| 27 | /stats token usage | Gemini CLI |
| 28 | Clipboard image paste | Gemini CLI |
| 29 | NotebookEdit | Claude Code |
| 30 | Prompt suggestions | Claude Code |

---

## 18. ROADMAP RECOMMANDÉE

### Phase 1 : Survie (rendre Poly viable pour sessions longues)
1. **Context compression** — auto-summarize quand on approche la limite
2. **MCP wiring** — brancher le client MCP existant dans le tool registry

### Phase 2 : Compétitivité (rattraper les concurrents)
3. **Context files** — auto-discover POLY.md, CLAUDE.md, .cursorrules
4. **Skills system** — lire des .md, les injecter dans le system prompt
5. **Markdown rendering** — utiliser glamour pour le chat
6. **Syntax highlighting** — chroma pour les code blocks
7. **Background jobs** — bash en background avec job tracking

### Phase 3 : Différenciation (ce que personne d'autre ne fait bien)
8. **Orchestrateur multi-AI amélioré** — consensus, voting, debate mode
9. **Sub-agents** — spawn des agents isolés par provider
10. **Hooks system** — events pour automation et CI/CD
11. **LSP integration** — diagnostics en temps réel

### Phase 4 : Écosystème
12. **Plugins/extensions** — distribuer des skills + MCP + hooks
13. **Agent Teams** — peer-to-peer coordination
14. **IDE integration** — VS Code extension

---

## 19. CE QUE POLY FAIT MIEUX QUE LES AUTRES

Malgré les gaps, Poly a des avantages uniques :

| Feature | Poly | Pourquoi c'est mieux |
|---------|------|---------------------|
| **Multi-provider natif** | 4 providers + custom | Claude Code = Claude only, Gemini CLI = Gemini only |
| **Cascade @all** | Tous les providers en parallèle | Personne d'autre ne fait ça (Crush a 11 providers mais pas de cascade) |
| **Shell mode** | Hybrid shell avec pipes AI | Unique — `ls -la \| @claude explain` |
| **Provider flexibility** | Switch avec @mention | Plus naturel que /model |
| **Cost-tier ordering** | Cheapest first | Économise de l'argent automatiquement |
| **Reviewer cross-model** | GPT review Claude, Gemini review GPT | Cross-validation unique |
| **Diff tools** | propose_diff + apply_diff | Workflow de review avant apply |

---

*Fin du rapport. 30 features manquantes identifiées, 3 critiques, 15 hautes, 12 moyennes/basses.*
