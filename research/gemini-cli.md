# Gemini CLI - Deep Research Report

**Version analysee** : 0.25.2 (npm @google/gemini-cli)
**Repo** : https://github.com/google-gemini/gemini-cli
**Licence** : Apache 2.0
**Date d'analyse** : 2026-02-10

---

## Architecture

### Stack Technique
- **Langage** : TypeScript compile en JavaScript (ESM modules)
- **Runtime** : Node.js >= 20
- **UI Framework** : React 19 + Ink 6 (TUI framework React-based pour le terminal)
- **Schema validation** : Zod
- **CLI parsing** : Yargs
- **LLM SDK** : `@google/genai` 1.30.0
- **MCP** : `@modelcontextprotocol/sdk` ^1.23.0
- **ACP** : `@agentclientprotocol/sdk` ^0.11.0

### Structure du code
```
@google/gemini-cli (CLI package)
  dist/src/
    gemini.js          -- Entry point principal (main())
    config/            -- Configuration, settings, auth, sandbox, policy
    core/              -- Initializer, auth, theme
    commands/          -- Sous-commandes CLI (mcp, extensions, skills, hooks)
    services/          -- CommandService, BuiltinCommandLoader, prompt processors
    ui/                -- React/Ink components (TUI complet)
      components/      -- 80+ composants React
      commands/        -- 30+ slash commands
      hooks/           -- 50+ React hooks
      themes/          -- 13 themes de couleur
      contexts/        -- 12 React contexts
    utils/             -- Sandbox, sessions, git, cleanup, events
    zed-integration/   -- Integration IDE Zed

@google/gemini-cli-core (Core library)
  dist/src/
    tools/             -- Built-in tools (13 tools)
    agents/            -- Agent system (local + remote A2A)
    mcp/               -- MCP auth providers
    hooks/             -- Hook system complet
    skills/            -- Skill loader & manager
    policy/            -- Policy engine (allow/deny/ask_user)
    routing/           -- Request routing
    safety/            -- Safety checks
    scheduler/         -- Task scheduling
    services/          -- Core services
    prompts/           -- System prompts
    telemetry/         -- OpenTelemetry integration
    fallback/          -- Model fallback logic
    ide/               -- IDE integration
```

### Process Model
1. `main()` dans `gemini.js` est le point d'entree
2. Detecte si on est dans un sandbox (variable `SANDBOX`)
3. Si pas en sandbox et sandboxing active : relance dans Docker/Podman/sandbox-exec
4. Sinon relance via `relaunchAppInChildProcess()` pour permettre les restarts internes
5. Charge les settings (4 niveaux : System > SystemDefaults > User > Workspace)
6. Initialise l'auth, les extensions, la policy engine
7. Mode interactif : lance l'UI React/Ink avec `render()`
8. Mode non-interactif : `runNonInteractive()` pour le mode headless/scripting

### Memory Management
- Alloue 50% de la RAM totale via `--max-old-space-size`
- Detecme auto si relance necessaire avec plus de memoire
- Variable `GEMINI_CLI_NO_RELAUNCH` pour desactiver

---

## Provider System

### Modeles supportes
- **Gemini 2.5 Pro** (modele par defaut via "auto")
- **Gemini 2.5 Flash** (via `gemini -m gemini-2.5-flash`)
- **Gemini 3 Pro** (nouveau, state-of-the-art reasoning)
- **Gemini 3 Flash** (SWE-bench 78%, agentic coding)
- **Preview models** : via flag `previewFeatures` dans settings
- **Variable d'env** : `GEMINI_MODEL` pour override

Le modele "auto" est un alias (`DEFAULT_GEMINI_MODEL_AUTO`) qui laisse Google choisir le meilleur modele.

### Authentication (3 methodes)

#### 1. Login with Google (OAuth)
- Flux OAuth via navigateur
- **Free tier** : 60 req/min, 1000 req/jour
- Context window 1M tokens
- Pas de gestion de cle API
- Support Gemini Code Assist License (enterprise)

#### 2. Gemini API Key
- `export GEMINI_API_KEY="..."`
- Free tier: 100 req/jour avec Gemini 2.5 Pro
- Selection de modele specifique
- Billing usage-based pour plus de limites

#### 3. Vertex AI
- `export GOOGLE_API_KEY="..."` + `GOOGLE_GENAI_USE_VERTEXAI=true`
- Enterprise features (securite, compliance)
- Rate limits plus eleves
- Integration Google Cloud

#### Auth additionnelle
- **Compute ADC** : pour Cloud Shell
- **Service Account Impersonation** : `sa-impersonation-provider.js`
- **External Auth** : `settings.security.auth.useExternal`
- Tokens stockes dans `~/.gemini/` via `oauth-token-storage.js`

### Streaming
- Streaming natif via `@google/genai` SDK
- Output formats : `text`, `json`, `stream-json` (newline-delimited JSON events)
- Hook `useGeminiStream` dans l'UI React

---

## Tool System

### 13 Built-in Tools

| Tool Name | Description | Confirmation |
|-----------|-------------|-------------|
| `read_file` | Lire un fichier | Non |
| `read_many_files` | Lire plusieurs fichiers | Non |
| `write_file` | Ecrire un fichier | Oui (sauf AUTO_EDIT/YOLO) |
| `replace` (edit) | Remplacer du texte dans fichier | Oui (sauf AUTO_EDIT/YOLO) |
| `run_shell_command` | Executer une commande shell | Oui (sauf YOLO) |
| `glob` | Recherche de fichiers par pattern | Non |
| `search_file_content` (grep) | Recherche dans le contenu (+ ripgrep) | Non |
| `list_directory` (ls) | Lister un repertoire | Non |
| `google_web_search` | Recherche Google (grounding) | Non |
| `web_fetch` | Recuperer contenu web | Oui (sauf YOLO) |
| `save_memory` | Sauvegarder dans GEMINI.md | Non |
| `write_todos` | Ecrire des TODOs | Non |
| `activate_skill` | Activer un skill | Non |
| `delegate_to_agent` | Deleguer a un sous-agent | Depends |

### Implementation des Tools
- Chaque tool herite de `BaseDeclarativeTool`
- Chaque invocation herite de `BaseToolInvocation`
- Schema defini via Zod, converti en JSON Schema pour le LLM
- Categorisation : `Kind.Think` (read-only), `Kind.Act` (modifications)
- `shouldConfirmExecute()` determine si confirmation necessaire
- Resultat : `{ llmContent, returnDisplay, isError }`

### Google Web Search (Grounding)
- **Integration directe avec Google Search** via le SDK Genai
- Le modele `web-search` est utilise en backend
- Retourne des resultats avec `groundingMetadata` (sources, chunks, supports)
- Citations inline avec indices des sources
- C'est un **differentiateur majeur** : pas besoin d'API externe

### Web Fetch
- Timeout de 10 secondes par URL
- Max 100K caracteres de contenu
- Conversion HTML -> texte via `html-to-text`
- Verification d'IP privee (securite)
- Retry avec backoff

### Tool Discovery (externe)
- `settings.tools.discoveryCommand` : commande shell pour decouvrir des tools
- `settings.tools.callCommand` : commande shell pour appeler des tools
- Prefix `discovered_tool_` pour les tools externes

### Exclusion et Permission des Tools
- `settings.tools.exclude` : liste de tools a exclure
- `settings.tools.allowed` : liste de tools auto-approuves
- `--allowed-tools` : override en CLI
- En mode non-interactif : shell, edit, write, web_fetch exclus par defaut

---

## MCP Integration

### Support MCP Client
- **SDK** : `@modelcontextprotocol/sdk` ^1.23.0
- **Client complet** : `mcp-client.js`, `mcp-client-manager.js`
- `mcp-tool.js` : wrapper de tools MCP en tools Gemini

### Configuration
```json
// ~/.gemini/settings.json
{
  "mcpServers": {
    "mon-server": {
      "command": "node",
      "args": ["/path/to/server.js"]
    }
  }
}
```

### Commandes CLI de gestion
- `gemini mcp add <name>` : ajouter un serveur MCP
- `gemini mcp list` : lister les serveurs MCP
- `gemini mcp remove <name>` : supprimer un serveur MCP

### Utilisation
- Prefix `@servername` pour cibler un serveur MCP specifique
- Ex: `@github List my open pull requests`
- Tools MCP nommes : `servername__toolname`

### Controle Admin
- `settings.admin.mcp.enabled` : activer/desactiver MCP globalement
- `settings.mcp.allowed` : liste blanche de serveurs MCP
- `settings.mcp.excluded` : liste noire de serveurs MCP
- `--allowed-mcp-server-names` : override en CLI

### MCP Auth Providers
- `google-auth-provider.js` : authentification Google pour MCP
- `oauth-provider.js` : OAuth generique pour MCP
- `sa-impersonation-provider.js` : Service Account pour MCP

---

## Agent System

### Architecture Multi-Agent
- **`delegate_to_agent`** : tool pour deleguer a un sous-agent
- Support agents **locaux** et **distants** (A2A protocol)

### Agents Locaux
- Definis via fichiers Markdown avec YAML frontmatter
- Stockes dans `.gemini/agents/` (workspace) ou `~/.gemini/agents/` (user)
- Chaque agent a :
  - `name` : identifiant slug
  - `description` : description pour le LLM
  - `system_prompt` : le corps du Markdown
  - `tools` : liste de tools autorises (pas `delegate_to_agent` = pas de recursion)
  - `model` : modele specifique ou `inherit`
  - `temperature`, `max_turns`, `timeout_mins`
- Execution via `local-executor.js` et `local-invocation.js`

### Agents Distants (A2A)
- **Agent-to-Agent Protocol** via `@agentclientprotocol/sdk`
- Definis avec `kind: remote` et `agent_card_url`
- `remote-invocation.js` pour l'execution
- `a2a-client-manager.js` pour gerer les connexions

### Agents Built-in
- **`codebase-investigator`** : agent specialise pour explorer le codebase
- **`cli-help-agent`** : agent d'aide pour le CLI
- Configuration via `settings.experimental.codebaseInvestigatorSettings`

### SubagentToolWrapper
- Enveloppe les agents en tools utilisables par le LLM
- `subagent-tool-wrapper.js` gere la conversion

### Agent Registry
- `registry.js` : registre central de tous les agents
- `agentLoader.js` : charge les agents depuis les fichiers .md
- Validation Zod stricte des definitions

---

## Context Management

### GEMINI.md (Context Files)
- Equivalent de CLAUDE.md pour Claude Code
- Hierarchie : project root, puis parents, puis `~/.gemini/`
- Fichier de contexte personnalise via `settings.context.fileName`
- Charge via `loadServerHierarchicalMemory()`

### Memory System
- Tool `save_memory` pour sauvegarder des notes persistantes
- `memoryTool.js` dans le core
- Import format configurable : `tree` (defaut) ou autre
- Memory file filtering configurable

### Session Management
- **Checkpointing** : sauvegarde/restauration de conversations
  - `--resume` / `--resume latest` pour reprendre
  - `--list-sessions` pour lister
  - `--delete-session` pour supprimer
- Sessions avec ID unique
- Compression automatique du contexte (`/compress` command)
- `settings.model.compressionThreshold` pour le seuil

### JIT Context (experimental)
- `settings.experimental.jitContext` : charge le contexte a la demande
- Au lieu de charger tout le GEMINI.md au demarrage

### Include Directories
- `--include-directories` : ajouter des repertoires au workspace
- `settings.context.includeDirectories` pour la config permanente

---

## Sandbox & Permissions

### Sandboxing
3 backends supportes :

#### 1. macOS Seatbelt (`sandbox-exec`)
- Detecte automatiquement sur macOS
- Profiles `.sb` personnalisables
- Profile par defaut : `permissive-open`
- Profiles custom dans `.gemini/sandbox-macos-*.sb`

#### 2. Docker
- Image officielle : `us-docker.pkg.dev/gemini-code-dev/gemini-cli/sandbox:0.25.2`
- Monte le workspace, settings, gcloud config, tmp
- Support proxy via `GEMINI_SANDBOX_PROXY_COMMAND`
- Network interne pour isolation
- Custom Dockerfile : `.gemini/sandbox.Dockerfile`

#### 3. Podman
- Alternative a Docker, meme interface
- Support rootless

### Approval Modes (Policy Engine)
3 modes de permission :

| Mode | Shell | Edit/Write | Read | Description |
|------|-------|-----------|------|-------------|
| `default` | Demande | Demande | Auto | Mode par defaut |
| `auto_edit` | Demande | Auto | Auto | Auto-approve edits |
| `yolo` | Auto | Auto | Auto | Tout auto-approuve |

- `--yolo` / `-y` pour activer YOLO
- `--approval-mode` pour choisir
- `settings.security.disableYoloMode` pour interdire YOLO
- `settings.admin.secureModeEnabled` : mode securise admin

### Policy Engine (`policy-engine.js`)
- Decisions : `ALLOW`, `DENY`, `ASK_USER`
- Policies chargeables en TOML (`toml-loader.js`)
- Repertoire `policies/` dans le core
- `PolicyDecision` enum pour chaque tool call
- `shell-safety` checker pour les commandes dangereuses
- `allowed-path` checker pour les chemins autorises

### Trusted Folders
- `~/.gemini/trustedFolders.json`
- Niveaux : `TRUST_FOLDER`, `TRUST_PARENT`, `DO_NOT_TRUST`
- En dossier non-trusted : approval mode force a `default`
- Workspace settings ignores en dossier non-trusted
- Integration IDE : l'IDE peut signaler si le workspace est trusted

### Environment Variable Redaction
- `settings.security.environmentVariableRedaction.enabled`
- `settings.security.environmentVariableRedaction.blocked` : liste de vars a masquer

---

## Extensions & Customization

### Extension System
- **Complet** : prompts, MCP servers, custom commands, hooks, sub-agents, skills
- Installation depuis GitHub (git clone ou GitHub releases)
- `gemini extensions install <url>`
- `gemini extensions list/enable/disable/uninstall/update`
- `settings.admin.extensions.enabled` pour controle admin

### Extension Structure
```
my-extension/
  extensions.json       -- Configuration de l'extension
  INSTALL_METADATA.json -- Metadata d'installation
  .env                  -- Variables d'environnement
  commands/             -- Custom commands
  skills/               -- Agent skills
  agents/               -- Sub-agents
```

### Custom Commands
- Definis dans des extensions ou dans le workspace
- `FileCommandLoader.js` et `McpPromptLoader.js`
- `BuiltinCommandLoader.js` pour les commandes built-in

### 30+ Slash Commands Built-in
```
/help, /chat, /clear, /compress, /copy, /corgi,
/directory, /docs, /editor, /extensions, /agents,
/auth, /bug, /ide, /init, /mcp, /memory, /model,
/permissions, /policies, /privacy, /profile, /quit,
/restore, /resume, /settings, /stats, /theme, /tools,
/vim, /skills, /hooks, /terminal-setup, /about
```

### Themes (13 built-in)
- Default (dark), Default Light
- Dracula, Atom One Dark, GitHub Dark, GitHub Light
- Ayu, Ayu Light, Shades of Purple, XCode
- GoogleCode, Holiday, ANSI, ANSI Light, No Color
- Auto-detection du fond terminal pour choisir light/dark
- `/theme` command pour changer

### Vim Mode
- `settings.general.vimMode` : keybindings Vim
- `vim-buffer-actions.js` pour les actions
- `/vim` command pour toggle

### Screen Reader Support
- `--screen-reader` flag
- Layout alternatif (`ScreenReaderAppLayout.js`)
- Pas de buffer alterne, pas de line wrapping

---

## Hooks System

### Architecture
```
HookSystem
  HookRegistry     -- Enregistre et charge les hooks
  HookRunner       -- Execute les hooks (shell scripts)
  HookAggregator   -- Agrege les resultats
  HookPlanner      -- Planifie quels hooks executer
  HookEventHandler -- Gere les evenements
```

### Types d'evenements
- `SessionStart` : au demarrage de la session
- `SessionEnd` : a la fin de la session
- Hooks de tools (avant/apres execution d'un tool)
- Sources : `project`, `user`, `system`, `extension`

### Configuration
```json
// settings.json ou .gemini/settings.json
{
  "hooks": {
    "pre-tool": [
      {
        "command": "./scripts/validate.sh",
        "tools": ["run_shell_command"]
      }
    ]
  }
}
```

### Trusted Hooks
- `trustedHooks.js` pour valider les hooks
- Hooks de projet necessitent un dossier trusted

### Commandes CLI
- `gemini hooks migrate` : migrer d'ancien format
- `/hooks` : gerer les hooks dans l'UI

---

## Skills System

### Structure
- Fichier `SKILL.md` avec YAML frontmatter :
```yaml
---
name: deploy-app
description: Deploy the application to production
---
Instructions pour le skill...
```

### Scopes
1. **Workspace** : `.gemini/skills/` (versionne)
2. **User** : `~/.gemini/skills/` (personnel)
3. **Extension** : embarques dans les extensions

### Activation
- Tool `activate_skill` pour activer un skill
- `skillLoader.js` decouvre les SKILL.md
- `skillManager.js` gere le cycle de vie

### Commandes CLI
- `gemini skills install <url>`
- `gemini skills list`
- `gemini skills enable/disable`
- `gemini skills uninstall`
- `settings.skills.disabled` : liste des skills desactives

---

## Unique Features (vs concurrents)

### 1. Google Search Grounding (natif)
- Integration directe avec Google Search, pas besoin d'API externe
- Citations inline avec sources
- Grounding metadata avec supports de verification
- **Aucun concurrent** n'a cette integration aussi profonde

### 2. Free Tier genereux
- 60 req/min, 1000 req/jour avec simple compte Google
- 1M token context window gratuit
- Pas besoin d'API key pour commencer

### 3. Sandbox Docker/Podman natif
- Isolation complete dans un container
- Image Docker officielle maintenue
- Seatbelt macOS en plus
- Proxy support pour les environnements corporate

### 4. Extension System complet
- Install depuis GitHub (git + releases)
- Package : commands + skills + agents + hooks + MCP servers
- Consent system pour les extensions
- Variable resolution dans les configs d'extension
- **Plus riche que Claude Code** (qui n'a que les skills)

### 5. Agent-to-Agent Protocol (A2A)
- Support natif du protocole A2A de Google
- Agents distants via `agent_card_url`
- Pas juste des sub-agents locaux

### 6. Hooks System (middleware)
- Scripts qui s'executent a des points specifiques du cycle
- Injection de contexte, validation, audit
- 4 sources : project, user, system, extension
- **Plus flexible que Claude Code hooks** (qui sont plus limites)

### 7. IDE Integrations
- Integration Zed (experimental)
- VS Code companion
- IDE trust delegation

### 8. Policy Engine en TOML
- Policies declaratives en fichiers TOML
- Granularite fine : per-tool, per-path
- Shell safety checker
- Admin override via remote settings (CCPA)

### 9. Non-Interactive Mode avance
- `--output-format json` pour parsing structure
- `--output-format stream-json` pour streaming events
- `--prompt` ou stdin pipe
- Excludes automatiques des tools dangereux

### 10. Theming riche
- 13 themes built-in
- Detection auto du fond terminal
- Gradient ASCII art
- Ink-based rendering avec React 19

### 11. ACP (Agent Client Protocol) experimental
- `--experimental-acp` pour mode ACP
- Protocole standard pour communication agent-client

### 12. Prompt Processors
- `@file` pour inclure des fichiers dans le prompt
- `!command` pour executer des commandes shell inline
- `injectionParser.js` pour la detection d'injection

---

## Settings Hierarchy

### 5 niveaux (priorite croissante)
1. **Schema Defaults** : valeurs par defaut codees
2. **System Defaults** : `/etc/gemini-cli/system-defaults.json`
3. **User Settings** : `~/.gemini/settings.json`
4. **Workspace Settings** : `.gemini/settings.json`
5. **System Settings** (overrides admin) : `/etc/gemini-cli/settings.json`

### Remote Admin Settings
- Depuis CCPA (Cloud Code Policy API)
- Override tout : `secureModeEnabled`, `mcp.enabled`, `extensions.enabled`
- Les settings admin fichier sont ignores au profit des remotes

### Environment Files
- `.gemini/.env` (prioritaire) ou `.env` dans le workspace
- Recherche remontante jusqu'a `~/.gemini/.env`
- Variables exclues configurables (`advanced.excludedEnvVars`)

---

## Telemetry & Monitoring

- **OpenTelemetry** integration complete
- Session events : start, end, errors
- Tool usage tracking
- Model latency metrics
- Extension events : install, enable, disable, update
- IDE connection events
- `settings.telemetry` pour configuration
- `settings.privacy.usageStatisticsEnabled` pour opt-in/out

---

## GitHub Integration

### Gemini CLI GitHub Action
- `google-github-actions/run-gemini-cli`
- PR Reviews automatiques
- Issue triage (labeling, prioritisation)
- Mention `@gemini-cli` dans issues/PRs
- Custom workflows (scheduled, on-demand)

### Git Integration (built-in)
- `simple-git` pour les operations git
- `useGitBranchName` hook pour le branch name
- `gitUtils.js` pour les utilities
- `git-commit.js` (generated) pour les commits

---

## Comparaison rapide vs Claude Code

| Feature | Gemini CLI | Claude Code |
|---------|-----------|-------------|
| LLM Provider | Gemini (Google) | Claude (Anthropic) |
| Free Tier | 60 req/min, 1000/jour | Non (Max plan minimum) |
| Context Window | 1M tokens | ~200K tokens |
| Built-in Tools | 13 | ~10 |
| Web Search | Google Search natif | WebSearch tool |
| Sandbox | Docker/Podman/Seatbelt | Non (juste permissions) |
| Extensions | Complet (github install) | Skills seulement |
| Agents | Local + Remote (A2A) | Task tool (sub-agents) |
| Hooks | System complet | Hooks basiques |
| MCP | Client | Client |
| Themes | 13 themes | Non |
| Vim Mode | Oui | Non |
| Non-Interactive | JSON/stream-json | Limited |
| Policy Engine | TOML-based, granulaire | Permission modes |
| IDE Integration | Zed + VS Code | VS Code |
| UI Framework | React/Ink | Custom (Rust-based) |
| Language | TypeScript/Node.js | Rust + TypeScript |
| Open Source | Oui (Apache 2.0) | Partiellement |

---

## Dependencies cles

| Package | Usage |
|---------|-------|
| `@google/genai` 1.30.0 | SDK Gemini API |
| `@modelcontextprotocol/sdk` ^1.23.0 | MCP client |
| `@agentclientprotocol/sdk` ^0.11.0 | A2A protocol |
| `ink` (fork @jrichman/ink) 6.4.7 | TUI React framework |
| `react` ^19.2.0 | UI rendering |
| `zod` ^3.23.8 | Schema validation |
| `yargs` ^17.7.2 | CLI argument parsing |
| `simple-git` ^3.28.0 | Git operations |
| `highlight.js` ^11.11.1 | Syntax highlighting |
| `diff` ^7.0.0 | File diff display |
| `dotenv` ^17.1.0 | Environment variables |
| `glob` ^12.0.0 | File globbing |
| `fzf` ^0.5.2 | Fuzzy finding |
| `undici` ^7.10.0 | HTTP client |
| `clipboardy` ^5.0.0 | Clipboard access |
| `extract-zip` ^2.0.1 | Extension extraction |

---

## Points forts pour Poly-go

1. **Le Google Search grounding est imbattable** - integration directe, pas d'API externe
2. **L'extension system est le plus riche** - git install, packages complets
3. **Le free tier est le plus genereux** - 1000 req/jour gratuit
4. **Le sandbox est le plus robuste** - Docker/Podman avec images officielles
5. **Le policy engine est le plus granulaire** - TOML, per-tool, per-path, admin remote
6. **A2A protocol** - seul a supporter les agents distants standards
7. **UI la plus personnalisable** - 13 themes, vim mode, screen reader

## Points faibles pour Poly-go

1. **Node.js only** - pas de binary standalone, besoin de Node 20+
2. **Google-locked** - uniquement modeles Gemini (pas d'OpenAI, Anthropic, etc.)
3. **Complexite** - 300+ fichiers source, architecture lourde
4. **React/Ink overhead** - plus lent au demarrage que les alternatives Go/Rust
5. **Pas de multi-provider** - contrairement a OpenCode qui supporte tout
