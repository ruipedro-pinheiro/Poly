# Claude Code - Internal Architecture Report V2

> Source: Reverse engineering du binaire ELF (strings extraction) + auto-documentation
> Binary: /home/pedro/.local/share/claude/versions/2.1.38 (213MB)
> Version: 2.1.38, Model: Opus 4.6
> Date: 2026-02-10

---

## 1. Architecture Runtime

### Binary Analysis
- **Format**: ELF 64-bit, **Bun runtime** (PAS Node.js - JSC/JavaScriptCore symbols confirmes)
- **Taille**: 213MB (`~/.local/share/claude/versions/2.1.38`)
- **Contenu**: Bun runtime + JS bundle minifie embarque (9119+ function/export/import refs)
- **Installation**: `~/.local/bin/claude` -> symlink vers `~/.local/share/claude/versions/2.1.38`
- **UI Framework**: React + Ink (composants: Button.ts, card.ts, input.ts, label.ts, select.ts, textarea.ts)
- **Source paths embarques**: src/entrypoints/cli.js, src/utils/bash/parser.ts, src/utils/claudeInChrome/setup.ts, src/utils/ripgrep.ts, src/utils/ide.ts

### Startup Flow (extrait du binaire)
```
1. Parse CLI args (--model, --print, --output-format, etc.)
2. Detect enterprise MCP config (MmH() check)
3. Setup tool permissions (xsD({allowedToolsCli, permissionMode, ...}))
4. Load MCP configs ([STARTUP] Loading MCP configs...)
5. Parse input (ok1(prompt, format)) - supports "text" and "stream-json"
6. Load tools (oK(toolPermissionContext))
7. Optional: structured output (tengu) via --json-schema
8. Launch TUI (React/Ink render) or non-interactive mode
```

### Non-Interactive Mode (DECOUVERTE MAJEURE)
Claude Code supporte un mode non-interactif complet:
- `--print` / `-p`: mode headless (pas de TUI)
- `--output-format text|stream-json`: format de sortie
- `--input-format text|stream-json`: format d'entree
- `--sdk-url`: connexion SDK externe (necessite stream-json)
- `--replay-user-messages`: replay de messages
- `--include-partial-messages`: messages partiels dans le stream
- `--no-session-persistence`: pas de sauvegarde de session
- `--json-schema`: structured output (codename "tengu")
- Pipe stdin -> prompt (mode scripting)

### Boucle Agentique (The Core Loop)

```
User message
  -> API call (Claude model)
  -> Response includes tool_use blocks?
     YES -> Execute tool(s) -> Feed results back -> Loop
     NO  -> Display text response -> Wait for user
```

- **Single-threaded** pour la boucle principale
- Les tool calls peuvent etre **paralleles** (plusieurs tools dans un meme message)
- La boucle continue tant que la reponse contient des `tool_use` content blocks
- Quand Claude produit du texte sans tool call, la boucle se termine

### Process Model
- **Process principal**: Boucle agentique + TUI rendering
- **Sub-processes**: Chaque sub-agent/teammate = process Node.js separe
- **Background tasks**: Commandes Bash en background avec IDs
- **MCP servers**: Processes enfants (stdio) ou connexions HTTP (SSE/streamable)

### Communication API
- **Protocole**: HTTP vers api.anthropic.com
- **Endpoint**: /v1/messages (Messages API)
- **Context window**: 200K tokens (Opus 4.6)
- **Buffer reserve**: ~33K tokens (16.5%) pour la reponse
- **Streaming**: Server-Sent Events pour le streaming de la reponse
- **Cache**: Support du prompt caching Anthropic (ephemeral 5m et 1h)
  - `cache_creation_input_tokens` et `cache_read_input_tokens` dans usage
  - System prompt + premiers messages caches pour reduire les couts

---

## 2. System Prompt Structure

Le system prompt de Claude Code est compose de plusieurs sections injectees dynamiquement:

### Sections du System Prompt (dans l'ordre)

1. **Identity**: "You are Claude Code, Anthropic's official CLI for Claude."
2. **System instructions**: Regles generales (securite, style, outils)
3. **Tool instructions**: Pour CHAQUE tool, description complete + parametres + exemples
4. **Git instructions**: Workflow de commit, PR creation, regles de securite git
5. **Auto memory**: MEMORY.md charge (200 premieres lignes)
6. **CLAUDE.md**: Instructions projet (hierarchie parent -> enfant)
7. **Environment**: OS, platform, date, model info, working directory, git status
8. **Language**: Instruction de langue si configuree
9. **MCP tool descriptions**: Descriptions des tools MCP decouverts
10. **Skill context**: Skills actifs injectes dynamiquement
11. **Active team context**: Info equipe si Agent Teams actif
12. **Browser automation**: Instructions Chrome si MCP claude-in-chrome actif
13. **Copyright**: Regles de copyright pour le contenu web

### Taille Typique du System Prompt
- Base: ~10K tokens
- Avec MCP tools: +2-5K tokens
- Avec CLAUDE.md: +1-10K tokens selon le projet
- Avec skills actifs: +1-5K tokens
- **Total typique**: 15-25K tokens

---

## 3. Tools Internes (Liste Complete)

### Tools de Fichiers

| Tool | Parametres | Comportement |
|------|-----------|-------------|
| **Read** | `file_path` (absolu), `offset`, `limit`, `pages` (PDF) | Lit fichiers, images, PDFs, notebooks. Cat -n format (numeros de lignes). Max 2000 lignes par defaut. Lignes > 2000 chars tronquees. |
| **Write** | `file_path` (absolu), `content` | Ecrase le fichier. DOIT lire avant d'ecrire (verification). Refuse les .md proactifs. |
| **Edit** | `file_path`, `old_string`, `new_string`, `replace_all` | Remplacement exact de string. Echoue si old_string pas unique (sauf replace_all). |
| **NotebookEdit** | `notebook_path`, `cell_id`, `new_source`, `cell_type`, `edit_mode` | Edit/insert/delete cellules Jupyter. |

### Tools de Recherche

| Tool | Parametres | Comportement |
|------|-----------|-------------|
| **Glob** | `pattern`, `path` | Matching de fichiers par pattern glob. Resultats tries par date de modification. |
| **Grep** | `pattern` (regex), `path`, `glob`, `type`, `output_mode`, `-i`, `-A/-B/-C`, `multiline`, `head_limit`, `offset` | Ripgrep-based. 3 modes: content, files_with_matches, count. Supporte multiline. |

### Tools Systeme

| Tool | Parametres | Comportement |
|------|-----------|-------------|
| **Bash** | `command`, `description`, `timeout` (max 600s), `run_in_background` | Execute des commandes shell. Output tronque a 30K chars. Background via `run_in_background`. |
| **TaskOutput** | `task_id`, `block`, `timeout` | Lit la sortie d'un Bash en background ou d'un agent. |
| **TaskStop** | `task_id` | Arrete un background task. |

### Tools Web

| Tool | Parametres | Comportement |
|------|-----------|-------------|
| **WebFetch** | `url`, `prompt` | Fetch URL, HTML->markdown, process avec modele rapide. Cache 15 min. |
| **WebSearch** | `query`, `allowed_domains`, `blocked_domains` | Recherche web. Resultats en markdown avec liens. Oblige d'inclure Sources. |

### Tools d'Agent

| Tool | Parametres | Comportement |
|------|-----------|-------------|
| **Task** | `prompt`, `description`, `subagent_type`, `model`, `run_in_background`, `name`, `team_name`, `mode`, `resume` | Spawne un sub-agent ou teammate. 7 types built-in. |
| **AskUserQuestion** | `questions[]` (1-4 questions, 2-4 options chacune, multiSelect) | Pose des questions structurees a l'utilisateur. Header court (12 chars max). |

### Tools de Plan

| Tool | Parametres | Comportement |
|------|-----------|-------------|
| **EnterPlanMode** | (aucun) | Demande a l'utilisateur de passer en plan mode. |
| **ExitPlanMode** | `allowedPrompts[]`, `pushToRemote` | Signal que le plan est pret pour review. |

### Tools de Taches (Task Management)

| Tool | Parametres | Comportement |
|------|-----------|-------------|
| **TaskCreate** | `subject`, `description`, `activeForm`, `metadata` | Cree une tache. Status initial: pending. |
| **TaskUpdate** | `taskId`, `status`, `subject`, `description`, `activeForm`, `owner`, `addBlocks[]`, `addBlockedBy[]`, `metadata` | Met a jour une tache. Status: pending/in_progress/completed/deleted. |
| **TaskList** | (aucun) | Liste toutes les taches avec id, subject, status, owner, blockedBy. |
| **TaskGet** | `taskId` | Details complets d'une tache. |

### Tools d'Equipe (Agent Teams)

| Tool | Parametres | Comportement |
|------|-----------|-------------|
| **TeamCreate** | `team_name`, `description`, `agent_type` | Cree equipe + task list. Fichiers dans ~/.claude/teams/ et ~/.claude/tasks/. |
| **TeamDelete** | (aucun) | Supprime equipe. Echoue si membres encore actifs. |
| **SendMessage** | `type`, `recipient`, `content`, `summary`, `approve`, `request_id` | 5 types: message, broadcast, shutdown_request, shutdown_response, plan_approval_response. |

### Tools Skills

| Tool | Parametres | Comportement |
|------|-----------|-------------|
| **Skill** | `skill`, `args` | Invoque un skill par nom. Charge le SKILL.md et l'injecte. |

### Tools MCP (Dynamiques)

Pattern de nommage: `mcp__<serveur>__<tool>`
Decouverts automatiquement au demarrage. Tool Search active si descriptions > 10% du context.
Includes aussi:
- **ListMcpResourcesTool**: Liste les ressources MCP disponibles
- **ReadMcpResourceTool**: Lit une ressource MCP specifique

---

## 4. Sub-Agent System

### Types Built-in

| Type | Model | Tools Disponibles | Usage |
|------|-------|-------------------|-------|
| **Explore** | Haiku (rapide) | Read, Glob, Grep, Bash (lecture), WebFetch, WebSearch | Recherche codebase rapide. 3 niveaux: quick/medium/very thorough |
| **Plan** | Herite parent | Read, Glob, Grep (PAS Write/Edit/Bash) | Recherche pour planification |
| **general-purpose** | Herite parent | TOUS les tools | Taches complexes multi-etapes |
| **Bash** | Herite parent | Bash uniquement | Commandes terminal isolees |
| **statusline-setup** | Sonnet | Read, Edit | Configuration statusline |
| **claude-code-guide** | Haiku | Glob, Grep, Read, WebFetch, WebSearch | Questions sur Claude Code |

### Execution Model

**Foreground (bloquant)**:
- Bloque la conversation principale
- Permissions passees a l'utilisateur
- AskUserQuestion transmis au parent

**Background (concurrent)**:
- `run_in_background: true`
- Tourne en parallele (jusqu'a 7 simultanes)
- Permissions pre-approuvees au lancement
- Auto-deny pour tout non pre-approuve
- PAS d'acces aux tools MCP
- Ctrl+B pour passer en background

### Custom Agents

Definis dans des fichiers Markdown avec YAML frontmatter:

```
Emplacements (priorite decroissante):
1. --agents CLI flag
2. .claude/agents/*.md (projet)
3. ~/.claude/agents/*.md (global)
4. Plugin agents/
```

Frontmatter supporte:
- `name` (requis): identifiant unique
- `description` (requis): quand deleguer a cet agent
- `tools`: liste des tools autorises
- `disallowedTools`: tools interdits
- `model`: sonnet/opus/haiku/inherit
- `permissionMode`: default/acceptEdits/delegate/dontAsk/bypassPermissions/plan
- `maxTurns`: limite de tours agentiques
- `skills`: skills a charger
- `mcpServers`: serveurs MCP disponibles
- `hooks`: hooks de cycle de vie
- `memory`: scope de memoire persistante (user/project/local)

### Contraintes
- Sub-agents NE PEUVENT PAS spawner d'autres sub-agents
- Max 7 agents simultanes
- Le sub-agent ne recoit PAS le system prompt complet du parent
- Auto-compaction a ~95% de capacite

---

## 5. Agent Teams System

### Architecture

```
Session principale (Team Lead)
  |
  ├── TeamCreate("my-team") -> cree ~/.claude/teams/my-team/config.json
  |                          -> cree ~/.claude/tasks/my-team/
  |
  ├── Task("teammate-1", team_name="my-team") -> spawn process
  |     ├── Propre context window (200K tokens)
  |     ├── Propres tools + permissions
  |     ├── Acces a TaskList/TaskUpdate/SendMessage
  |     └── Charge CLAUDE.md, MCP, skills (meme config que session normale)
  |
  ├── Task("teammate-2", team_name="my-team") -> spawn process
  |
  └── Shared Task List
        ├── Taches avec status: pending -> in_progress -> completed
        ├── Ownership: taches assignees a des teammates par nom
        └── Dependencies: blocks/blockedBy entre taches
```

### Communication Inter-Agents

**Mailbox system**:
- Chaque agent a une mailbox
- Messages delivres automatiquement au prochain tour
- Si l'agent est busy (mid-turn): messages queues
- UI montre une notification quand messages en attente

**Types de messages SendMessage**:
1. `message` -> DM a un teammate (recipient obligatoire, summary obligatoire)
2. `broadcast` -> Message a TOUS (couteux: N teammates = N deliveries)
3. `shutdown_request` -> Demande de shutdown (recipient recoit et doit repondre)
4. `shutdown_response` -> Reponse au shutdown (approve: true/false, request_id obligatoire)
5. `plan_approval_response` -> Approuver/rejeter le plan d'un teammate

### Idle State
- Les teammates vont idle apres CHAQUE tour (normal)
- Idle != termine - ils attendent juste du input
- Envoyer un message a un idle teammate le reveille
- Idle notifications sont automatiques
- Les DMs entre pairs sont resumes dans l'idle notification du lead

### Team Lead
- Session qui cree l'equipe (fixe)
- Coordonne, assigne, synthetise
- Recoit auto les messages + idle notifications
- **Delegate mode** (Shift+Tab): se limite a la coordination, ne code pas

### Teammates
- Instances independantes, propre context window
- Recoivent le prompt de spawn (PAS l'historique de conversation du lead)
- NE PEUVENT PAS creer de nested teams
- Peuvent self-claim des taches non assignees/non bloquees
- Preference: travailler les taches par ID croissant

### Modes d'affichage
- **In-process**: tous dans le meme terminal, Shift+Up/Down pour naviguer
- **Split panes**: chaque teammate dans son pane (tmux/iTerm2)
- **auto** (defaut): split si deja dans tmux, sinon in-process

### Limitations
- Pas de resumption avec in-process teammates
- Status des taches peut trainer (teammates oublient de marquer completed)
- Shutdown peut etre lent
- Une seule equipe par session
- Pas de nested teams
- Lead fixe (pas de promotion)

---

## 6. Permission System

### Modes de Permission

| Mode | Comportement | Activation |
|------|-------------|-----------|
| `default` | Demande approbation a la premiere utilisation de chaque type | Par defaut |
| `acceptEdits` | Auto-accepte Write/Edit, demande pour Bash | Mode rapide |
| `plan` | LECTURE SEULEMENT - pas de Write/Edit/Bash | EnterPlanMode |
| `delegate` | Coordination seulement (tools equipe uniquement) | Shift+Tab avec equipe active |
| `dontAsk` | Auto-deny sauf pre-approuves via /permissions | Mode strict |
| `bypassPermissions` | Saute TOUS les checks | `--dangerously-skip-permissions` |

### Categories de Tools

| Categorie | Exemples | Permission requise |
|-----------|----------|-------------------|
| Read-only | Read, Glob, Grep | Jamais |
| File modification | Write, Edit | Oui (session-scoped) |
| Bash commands | Bash | Oui (permanent par projet+commande) |
| Web | WebFetch, WebSearch | Oui |
| MCP tools | mcp__*__* | Oui |

### Regles de Permission (settings.json)

Format: `Tool` ou `Tool(specifier)`

```
permissions.allow:
  - "Bash(npm run *)"        # Wildcard
  - "Read(./.env)"           # Fichier specifique
  - "Edit(/src/**/*.ts)"     # Pattern gitignore
  - "WebFetch(domain:example.com)"  # Domaine
  - "Task(Explore)"          # Type de sub-agent
  - "mcp__github__*"         # Tous tools d'un serveur MCP

permissions.deny:
  - "Bash(rm -rf *)"
```

Ordre d'evaluation: deny -> ask -> allow (deny prioritaire)

### Changement de Mode Runtime
- **Shift+Tab**: cycle entre modes
- `defaultMode` dans settings.json
- `--dangerously-skip-permissions` en CLI
- Chaque teammate peut avoir son propre mode (fixe au spawn via `mode` param)

---

## 7. Hooks System

### Types de Hooks

| Type | Description |
|------|-------------|
| `command` | Commande shell. Recoit JSON sur stdin, communique via exit codes + stdout |
| `prompt` | Prompt envoye a un modele Claude rapide pour evaluation yes/no |
| `agent` | Sub-agent spawne avec Read/Grep/Glob pour verification multi-turn (max 50 turns, 60s timeout) |

### Evenements (14 total)

| Evenement | Quand | Peut bloquer | Matcher |
|-----------|-------|-------------|---------|
| `SessionStart` | Debut/resume/clear/compact | Non | startup, resume, clear, compact |
| `UserPromptSubmit` | Soumission prompt utilisateur | Oui | Pas de matcher |
| `PreToolUse` | Avant appel d'outil | Oui | Nom de l'outil |
| `PermissionRequest` | Dialog de permission | Oui | Nom de l'outil |
| `PostToolUse` | Apres appel reussi | Non | Nom de l'outil |
| `PostToolUseFailure` | Apres echec | Non | Nom de l'outil |
| `Notification` | Notification envoyee | Non | Type de notification |
| `SubagentStart` | Sub-agent spawne | Non | Type d'agent |
| `SubagentStop` | Sub-agent termine | Oui | Type d'agent |
| `Stop` | Claude finit de repondre | Oui | Pas de matcher |
| `TeammateIdle` | Teammate va devenir idle | Oui | Pas de matcher |
| `TaskCompleted` | Tache marquee complete | Oui | Pas de matcher |
| `PreCompact` | Avant compaction | Non | manual, auto |
| `SessionEnd` | Fin de session | Non | Raison de fin |

### Exit Codes (hooks command)
- **0**: Succes, stdout parse comme JSON, action continue
- **2**: Erreur bloquante, stderr -> message d'erreur a Claude, action bloquee
- **Autre**: Erreur non-bloquante, continue

### Hook Input (JSON sur stdin)
Commun a tous:
```json
{
  "session_id": "...",
  "transcript_path": "...",
  "cwd": "...",
  "permission_mode": "...",
  "hook_event_name": "..."
}
```
Plus champs specifiques par evenement (tool_name, tool_input pour PreToolUse, etc.)

### Decision Control

**PreToolUse** -> `hookSpecificOutput.permissionDecision`: "allow|deny|ask"
  + `updatedInput` pour modifier les parametres du tool
  + `additionalContext` pour injecter du contexte

**Stop, PostToolUse, UserPromptSubmit, SubagentStop** -> `decision`: "block" + `reason`

**PermissionRequest** -> `hookSpecificOutput.decision.behavior`: "allow|deny"
  + `updatedInput`, `updatedPermissions`

### Hooks Async
- `"async": true` sur type command
- Tourne en background sans bloquer
- NE PEUT PAS bloquer les actions
- Resultat delivre au prochain tour

### Configuration
Emplacements:
1. `~/.claude/settings.json` (global)
2. `.claude/settings.json` (projet, committable)
3. `.claude/settings.local.json` (projet, gitignore)
4. Managed policy (organisation)
5. Plugin hooks
6. Skill/Agent frontmatter (scoped au composant actif)

---

## 8. Skills System

### Format SKILL.md

```yaml
---
name: my-skill                    # Identifiant unique
description: Description          # Quand l'utiliser
disable-model-invocation: true    # Claude ne peut PAS l'invoquer seul
user-invocable: true              # Visible dans le menu /
allowed-tools: Read, Grep, Glob   # Tools sans approbation
model: sonnet                     # Modele a utiliser
context: fork                     # Execute dans un sub-agent isole
agent: Explore                    # Type de sub-agent si fork
argument-hint: [issue-number]     # Hint autocompletion
hooks:                            # Hooks scoped au skill
  PreToolUse:
    - matcher: "Bash"
      hooks:
        - type: command
          command: "./check.sh"
---

Instructions en Markdown...
Variables: $ARGUMENTS, $ARGUMENTS[N], $N, ${CLAUDE_SESSION_ID}
Dynamic: !`command` execute avant envoi a Claude
```

### Emplacements (priorite: enterprise > personal > project)
- Enterprise: Managed settings
- Personnel: `~/.claude/skills/<name>/SKILL.md`
- Projet: `.claude/skills/<name>/SKILL.md`
- Plugin: `<plugin>/skills/<name>/SKILL.md`

### Fusion avec Slash Commands
Depuis v2.1.3: `.claude/commands/review.md` et `.claude/skills/review/SKILL.md` creent tous les deux `/review`.

---

## 9. Context Management

### CLAUDE.md Hierarchy
- Fichiers dans repertoires **parents** du CWD: charges entierement au lancement
- Fichiers dans repertoires **enfants**: charges a la demande quand Claude lit des fichiers dans ces dirs
- Support de CLAUDE.md, CLAUDE.local.md

### Memory Files
- Repertoire: `~/.claude/projects/{project-hash}/memory/`
- `MEMORY.md`: TOUJOURS charge dans le system prompt (200 premieres lignes max)
- Fichiers separes par sujet (debugging.md, patterns.md) references depuis MEMORY.md
- Claude met a jour proactivement quand il apprend quelque chose de stable

### Compaction (Context Compression)
- **Auto-compaction**: se declenche a ~95% de capacite
  - Configurable via `CLAUDE_AUTOCOMPACT_PCT_OVERRIDE`
- **Manuel**: commande `/compact`
- **Process**: historique condense par un LLM en resume compact
- **Lossy**: des details sont perdus
- **Buffer**: ~33K tokens (16.5%) reserves

### Sessions
- Stockees comme `.jsonl` dans `~/.claude/projects/{project}/{sessionId}/`
- Sub-agent transcripts dans `subagents/agent-{agentId}.jsonl`
- Resume via `/resume` et `/rewind`
- Cleanup auto base sur `cleanupPeriodDays` (defaut: 30 jours)

---

## 10. Model System

### Modeles Disponibles

| Modele | ID | Vitesse | Intelligence | Usage |
|--------|-----|---------|-------------|-------|
| **Opus 4.6** | claude-opus-4-6 | Lent | Maximum | Architecture, debugging complexe |
| **Sonnet 4.5** | claude-sonnet-4-5-20250929 | Moyen | Elevee | Usage quotidien (defaut) |
| **Haiku 4.5** | claude-haiku-4-5-20251001 | Rapide | Bonne | Taches simples, sub-agents rapides |

### Selection de Modele
- Au lancement: `claude --model claude-opus-4-6`
- Pendant session: `/model opus`
- **Fast mode** (`/fast`): meme modele, sortie plus rapide
- **Opusplan**: hybride Opus (plan) + Sonnet (implementation)
- Per sub-agent: param `model` dans Task tool

### Token Accounting
- `input_tokens`: tokens d'entree factures
- `cache_creation_input_tokens`: tokens caches crees (cout reduit)
- `cache_read_input_tokens`: tokens lus depuis le cache (cout tres reduit)
- `output_tokens`: tokens de sortie
- Le system prompt est cache agressivement (ephemeral_1h)

---

## 11. Git Integration

### Workflow Commit
1. `git status` + `git diff` + `git log` en parallele
2. Analyse tous les changements (staged + unstaged)
3. Redige message de commit (why > what)
4. `git add` fichiers specifiques (JAMAIS `git add -A`)
5. Commit avec HEREDOC pour le formatting
6. Tag: `Co-Authored-By: Claude <model> <noreply@anthropic.com>`
7. `git status` apres pour verifier

### Regles de Securite Git
- JAMAIS modifier git config
- JAMAIS push --force, reset --hard, checkout ., clean -f sauf demande explicite
- JAMAIS skip hooks (--no-verify)
- JAMAIS force push sur main/master
- TOUJOURS creer de NOUVEAUX commits (pas amender sauf demande)
- JAMAIS committer sans demande explicite

### Creation PR
- `gh pr create` avec format: Summary + Test plan + watermark
- Analyse TOUS les commits depuis la divergence (pas juste le dernier)
- Titre court (<70 chars), corps detaille

---

## 12. MCP Integration

### Configuration
```json
// ~/.claude.json ou .claude/settings.json
{
  "mcpServers": {
    "server-name": {
      "type": "stdio",
      "command": "node",
      "args": ["/path/to/server.js"]
    }
  }
}
```

### Tool Discovery
- Decouverte auto des tools sur chaque serveur MCP au demarrage
- Pattern de nommage: `mcp__<server>__<tool>`
- Tool Search auto si descriptions > 10% du context window
- Support `list_changed` notifications pour refresh dynamique

### Resources
- `ListMcpResourcesTool`: liste les ressources d'un serveur
- `ReadMcpResourceTool`: lit une ressource par URI

### Permissions MCP
- Pattern `mcp__<server>__<tool>` dans les regles de permission
- Wildcard: `mcp__github` match tous les tools du serveur github
- Hooks PreToolUse/PostToolUse fonctionnent avec les tools MCP

### MCP dans les Sub-agents
- Tools MCP herites par defaut dans les sub-agents foreground
- Sub-agents background: PAS d'acces MCP
- Custom agents: `mcpServers` field pour specifier quels serveurs

---

## 13. TUI & Interface

### Surfaces Disponibles
- **Terminal**: CLI complete
- **VS Code**: Extension avec diffs inline, @-mentions, plan review
- **JetBrains**: Plugin IntelliJ, PyCharm, WebStorm
- **Desktop**: Application standalone
- **Web**: Sessions cloud
- **iOS**: Via l'app Claude

### Raccourcis Terminal
- **Shift+Tab**: cycle entre modes de permission
- **Shift+Up/Down**: naviguer entre teammates (in-process teams)
- **Ctrl+B**: passer une tache en background
- **/help**: aide
- **/compact**: compaction manuelle
- **/model**: changer de modele
- **/resume**: reprendre une session
- **/rewind**: revenir en arriere
- **/permissions**: gerer les permissions
- **/fast**: toggle fast mode

### Rendering
- Markdown GitHub-flavored (CommonMark) en monospace
- Code blocks avec syntax highlighting
- Diffs inline pour les edits de fichiers
- Spinner pendant les tool calls
- Status line avec infos agent/model

---

## 14. Reverse Engineering Findings (Binary Extraction)

### Team System Internals (AsyncLocalStorage pattern)
```javascript
// Extrait du binaire - fonctions exportees du module team
exports = {
  waitForTeammatesToBecomeIdle: S0A,
  setDynamicTeamContext: Tb0,
  runWithTeammateContext: mf$,   // AsyncLocalStorage.run()
  isTeammate: IE,
  isTeamLead: J7,
  isPlanModeRequired: AOH,
  isInProcessTeammate: X7,       // AsyncLocalStorage.getStore() !== undefined
  hasWorkingInProcessTeammates: x0A,
  hasActiveInProcessTeammates: df$,
  getTeammateContext: eq,         // AsyncLocalStorage.getStore()
  getTeammateColor: T4,
  getTeamName: l9,
  getParentSessionId: un,
  getDynamicTeamContext: $OH,
  getAgentName: b1,
  getAgentId: OK,
  createTeammateContext: pf$,     // Returns {...context, isInProcess: true}
  clearDynamicTeamContext: zb0
}
```

Key insight: **AsyncLocalStorage** (FeL = require("async_hooks")) est utilise pour propager
le contexte d'equipe a travers les appels async. Chaque teammate a son propre store
avec parentSessionId, agentId, agentName, teamName, color, planModeRequired.

### TodoWrite Internals
```javascript
// Zod schema pour les todos
Zb0 = S.enum(["pending", "in_progress", "completed"])
qb0 = S.object({
  content: S.string().min(1),
  status: Zb0,
  activeForm: S.string().min(1)
})
YXH = S.array(qb0)  // Array de todos
```

### Telemetry & Observability
- System prompt hash tracking (evite re-envoi si identique)
- Tool hash tracking (dedup via Set R7$)
- Span attributes: system_prompt_hash, system_prompt_preview, tools_count
- Truncation pour les gros contenus (Ap() function)
- Event logging: LW("system_prompt", {...}), LW("tool", {...})

### Plugin System (DECOUVERTE)
```javascript
// Plugin recommendations extraites du binaire
{
  id: "frontend-design@claude-code-plugins",
  // Active quand des fichiers .html/.css/.htm sont lus
  isRelevant: (H) => files.some(f => /\.(html|css|htm)$/i.test(f))
}
```
- Plugins installes via `/plugin install <name>@<registry>`
- Cooldown entre suggestions (cooldownSessions: 3)
- Relevant detection basee sur les fichiers lus

### Monetization Features
- **Guest passes**: `/passes` command, `Ee(count)` pour afficher le nombre
- **Extra usage**: `/extra-usage` command, "$50 free extra usage"
- **Fast mode promotion** pour les utilisateurs eligibles
- `zO$()` pour verifier l'eligibilite

### Session Management Internals
```javascript
// Session lite loading (metadata sans transcript complet)
{
  isLite: true,
  fullPath: path,
  sessionId: id,
  projectPath: projectPath,
  created: Date,
  modified: Date,
  firstPrompt: "",
  messageCount: 0,
  fileSize: size,
  isSidechain: false
}
// Full loading: parse JSONL pour extraire firstPrompt, gitBranch, teamName, customTitle, tag
```

### Idle Notification System
```javascript
// Extrait du binaire
if (!isLoading && !localJSX && idleTimer === undefined && timeSinceLastQuery >= messageIdleNotifThresholdMs) {
  notify({
    message: "Claude is waiting for your input",
    notificationType: "idle_prompt"
  }, isNonInteractive)
}
```

### Queued Commands System
- `promptQueueUseCount` tracks how many queued prompts have been processed
- `executeQueuedInput` processes queue after loading completes
- `hasActiveLocalJsxUI` flag prevents queue execution during local JSX UI

### Agent Definition Resolution
```javascript
// On session resume, agent definitions are re-resolved
function resolveAgentDefinition(agentSetting, mainThreadAgent, agentDefinitions) {
  let agent = agentDefinitions.activeAgents.find(a => a.agentType === agentSetting)
  if (!agent) {
    log(`Resumed session had agent "${agentSetting}" but it is no longer available`)
    return { agentDefinition: undefined, agentType: undefined }
  }
  // If agent has custom model, switch to it
  if (!isTeammate() && agent.model && agent.model !== "inherit") {
    setModel(resolveModel(agent.model))
  }
  return { agentDefinition: agent, agentType: agent.agentType }
}
```

### File History & Stale Edit Detection
```javascript
// fileHistorySnapshots are restored on session resume
function restoreFileHistory(snapshots, setState) {
  restoreFromSnapshots(snapshots, (history) => {
    setState(prev => ({...prev, fileHistory: history}))
  })
}
```

---

## 15. Configuration & Settings

### Fichiers de Configuration (priorite croissante)
1. User settings: `~/.claude/settings.json`
2. Shared project: `.claude/settings.json` (committable)
3. Local project: `.claude/settings.local.json` (gitignore)
4. CLI arguments
5. Managed settings (enterprise): `/etc/claude-code/managed-settings.json` (Linux)

### Settings Cles
```json
{
  "permissions": {
    "allow": ["Bash(npm run *)"],
    "deny": ["Bash(rm -rf *)"]
  },
  "env": {
    "CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS": "1"
  },
  "hooks": { ... },
  "defaultMode": "default",
  "cleanupPeriodDays": 30
}
```

### Managed Settings (Enterprise)
- `disableBypassPermissionsMode`: empeche bypass
- `allowManagedPermissionRulesOnly`: seules regles managed
- `allowManagedHooksOnly`: seuls hooks managed
- Priorite maximale, ecrase tout

---

## 15. Differences Architecturales Cles vs Crush/OpenCode

| Aspect | Claude Code | Crush | OpenCode |
|--------|------------|-------|----------|
| **Agentic loop** | while(tool_call) simple | Fantasy Agent.Stream() avec callbacks | Vercel AI SDK streamText() |
| **Multi-provider** | Non (Anthropic only) | Oui (9 via Fantasy) | Oui (20+ via AI SDK) |
| **Teams** | FULL (TeamCreate, SendMessage, TaskList) | Non | Non |
| **Sub-agents** | 6 types + custom + background | 2 types (agent, agentic_fetch) | 7 types |
| **Hooks** | 14 events, 3 types (command/prompt/agent) | Aucun | 15+ via plugin API |
| **Skills** | SKILL.md complet avec fork context | agentskills.io basic | .claude/ compat |
| **Storage** | JSONL files | SQLite | JSON files + locks |
| **Config** | CLAUDE.md + settings.json | JSON + Catwalk remote | JSONC 7 niveaux |
| **Plan mode** | Oui (read-only analysis) | Non | Oui |
| **Memory** | MEMORY.md persistent | Non | Non |
| **TUI** | Ink (React) | Bubble Tea v2 | SolidJS (@opentui) |

---

## 16. Ce que Poly-go Doit Voler a Claude Code

### Priorite CRITIQUE
1. **Agent Teams complet** - TeamCreate, SendMessage, TaskList, mailbox
2. **Sub-agents** - Task tool avec 6+ types
3. **Context compaction** - auto a 95%, resume par IA
4. **Memory files** - MEMORY.md persistent entre sessions

### Priorite HAUTE
5. **Hooks system** - au moins PreToolUse + PostToolUse + command type
6. **Permission modes** - default, plan, yolo, delegate
7. **Skills system** - SKILL.md avec frontmatter
8. **Custom agents** - .poly/agents/*.md avec frontmatter
9. **Plan mode** - read-only analysis avant implementation

### Priorite MOYENNE
10. **AskUserQuestion** - questions structurees avec options
11. **Background agents** - run_in_background + notifications
12. **Session resume** - /resume et /rewind
13. **Git workflow** - commit + PR avec conventions
14. **Managed settings** - enterprise config override
