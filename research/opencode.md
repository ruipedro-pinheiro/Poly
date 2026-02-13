# OpenCode - Reverse Engineering Report

**Version analysee**: 1.1.53
**Repository**: https://github.com/anomalyco/opencode
**Licence**: MIT
**Langage**: TypeScript (monorepo)
**Runtime**: Bun (pas Node.js)
**Date d'analyse**: 2026-02-10

---

## 1. Architecture Monorepo

### 1.1 Build System

- **Package Manager**: Bun 1.3.8 (declare dans `package.json` via `"packageManager": "bun@1.3.8"`)
- **Monorepo orchestration**: Turbo (`turbo.json`) pour le build et typecheck
- **Workspaces**: Bun workspaces definis dans le root `package.json`
- **TypeScript**: 5.8.2, avec support du TypeScript natif (`@typescript/native-preview` 7.0.0-dev)
- **Typecheck**: Utilise `tsgo --noEmit` (le nouveau type-checker natif TS)
- **Build script**: `packages/opencode/script/build.ts` (custom Bun build)

### 1.2 Packages Breakdown

| Package | Role | Chemin |
|---------|------|--------|
| **opencode** | Coeur du CLI - agent, session, tools, provider, MCP | `packages/opencode/` |
| **plugin** | API et types pour les plugins | `packages/plugin/` |
| **sdk** | SDK JavaScript client pour l'API OpenCode | `packages/sdk/js/` |
| **ui** | Interface web (SolidJS) | `packages/ui/` |
| **app** | Application web SolidStart | `packages/app/` |
| **desktop** | Application desktop Tauri 2 | `packages/desktop/` |
| **console** | Console d'administration (4 sous-packages) | `packages/console/` |
| **web** | Commande `opencode web` | `packages/web/` |
| **util** | Utilitaires partages (NamedError, slug) | `packages/util/` |
| **script** | Scripts de build/deploy | `packages/script/` |
| **enterprise** | Features entreprise | `packages/enterprise/` |
| **function** | Serverless functions | `packages/function/` |
| **identity** | Service d'identite | `packages/identity/` |
| **containers** | Container configs | `packages/containers/` |
| **docs** | Documentation | `packages/docs/` |
| **slack** | Integration Slack | `packages/slack/` |
| **extensions/zed** | Extension Zed editor | `packages/extensions/zed/` |
| **sdks/vscode** | Extension VS Code | `sdks/vscode/` |

### 1.3 Infrastructure

- **IaC**: SST (sst.config.ts) pour le deploiement cloud (AWS)
- **Nix**: `flake.nix` pour l'environment de dev reproductible
- **Containers**: Docker configs dans `packages/containers/`

---

## 2. Entry Point et CLI

**Fichier**: `packages/opencode/src/index.ts`

### 2.1 Framework CLI

Utilise **yargs** pour le parsing de commandes. Structure:

```
opencode [command] [options]
```

### 2.2 Commandes Disponibles

| Commande | Fichier | Description |
|----------|---------|-------------|
| `run [message..]` | `cli/cmd/run.ts` | Commande principale - envoie un message a l'agent |
| `(default)` | `cli/cmd/tui/` | TUI interactif (quand lance sans args) |
| `serve` | `cli/cmd/serve.ts` | Lance le serveur HTTP/API |
| `web` | `cli/cmd/web.ts` | Lance l'interface web |
| `auth` | `cli/cmd/auth.ts` | Gestion de l'authentification |
| `agent` | `cli/cmd/agent.ts` | Gestion des agents |
| `models` | `cli/cmd/models.ts` | Liste les modeles disponibles |
| `mcp` | `cli/cmd/mcp.ts` | Gestion MCP |
| `generate` | `cli/cmd/generate.ts` | Generation d'agents |
| `upgrade` | `cli/cmd/upgrade.ts` | Mise a jour |
| `uninstall` | `cli/cmd/uninstall.ts` | Desinstallation |
| `debug` | `cli/cmd/debug.ts` | Debug tools |
| `stats` | `cli/cmd/stats.ts` | Statistiques d'utilisation |
| `export` | `cli/cmd/export.ts` | Export de sessions |
| `import` | `cli/cmd/import.ts` | Import de sessions |
| `github` | `cli/cmd/github.ts` | Integration GitHub |
| `pr` | `cli/cmd/pr.ts` | Pull request workflows |
| `session` | `cli/cmd/session.ts` | Gestion de sessions |
| `acp` | `cli/cmd/acp.ts` | Agent Communication Protocol |
| `attach` | `cli/cmd/tui/attach.ts` | Attach a un serveur existant |
| `thread` | `cli/cmd/tui/thread.ts` | Threads TUI |

### 2.3 Bootstrap

`cli/bootstrap.ts` initialise une "instance" pour le projet courant:
- Decouvre le projet (git worktree, etc.)
- Initialise la config, le storage, les plugins
- Fournit un contexte AsyncLocalStorage via `Instance.provide()`

---

## 3. TUI System

**Dossier**: `packages/opencode/src/cli/cmd/tui/`

### 3.1 Framework de Rendu

- **Bibliotheque**: `@opentui/core` + `@opentui/solid` (un framework TUI custom base sur SolidJS)
- **PAS Ink** (React), PAS blessed - c'est **SolidJS dans le terminal** via @opentui
- **Rendu reactif**: SolidJS signals/effects pour le state management
- **JSX**: Fichiers `.tsx` utilisant la syntaxe JSX SolidJS

### 3.2 Architecture TUI

**`app.tsx`** - Point d'entree principal du TUI:
- Detecte la couleur de fond du terminal (dark/light) via escape sequences
- Wrappe tout dans des Providers SolidJS (contextes imbriques):
  - `ArgsProvider` - Arguments CLI
  - `ThemeProvider` - Theming
  - `SDKProvider` - Client SDK
  - `SyncProvider` - Synchronisation de l'etat
  - `LocalProvider` - State local
  - `DialogProvider` - Systeme de dialogues
  - `KeybindProvider` - Raccourcis clavier
  - `ToastProvider` - Notifications toast
  - `ExitProvider` - Gestion de la sortie
  - `PromptHistoryProvider` - Historique des prompts
  - `FrecencyProvider` - Frequence/recence des modeles
  - `PromptStashProvider` - Stash de prompts
  - `CommandProvider` - Commandes slash
  - `RouteProvider` - Routing entre vues

### 3.3 Routes TUI

- **Home** (`routes/home.tsx`) - Ecran d'accueil
- **Session** (`routes/session/`) - Vue de conversation

### 3.4 Composants TUI

Dossier `component/`:
- `dialog-agent.tsx` - Selection d'agent
- `dialog-model.tsx` - Selection de modele
- `dialog-command.tsx` - Menu de commandes
- `dialog-session-list.tsx` - Liste des sessions
- `dialog-session-rename.tsx` - Renommage
- `dialog-mcp.tsx` - Status MCP
- `dialog-status.tsx` - Status du systeme
- `dialog-theme-list.tsx` - Themes
- `dialog-skill.tsx` - Skills
- `dialog-stash.tsx` - Stash
- `dialog-provider.tsx` - Providers
- `dialog-tag.tsx` - Tags
- `prompt/` - Composants de la zone d'input
- `spinner.tsx` - Indicateur d'activite
- `tips.tsx` - Astuces
- `todo-item.tsx` - Items de todo
- `logo.tsx` - Logo ASCII
- `border.tsx` - Bordures

### 3.5 Gestion du Clavier

Keybinds extremement configurables via `Config.Keybinds` (defini dans `config/config.ts`).
Plus de 70 keybinds configurables incluant:
- Navigation (page up/down, half page, first/last message)
- Edition (undo, redo, copy, paste)
- Session management (new, list, fork, compact, share)
- Model/Agent cycling
- Input editing complet (word forward/backward, delete word, etc.)
- Leader key system (par defaut `ctrl+x`)

### 3.6 Themes

Dossier `context/theme/` - Systeme de theming complet pour le TUI.

---

## 4. Provider System

**Fichiers principaux**: `provider/provider.ts`, `provider/models.ts`, `provider/transform.ts`

### 4.1 LLM Providers Supportes (Bundled)

OpenCode supporte un nombre massif de providers via le **Vercel AI SDK** (`ai` v5.0.124):

| Provider | Package AI SDK |
|----------|---------------|
| Anthropic | `@ai-sdk/anthropic` |
| OpenAI | `@ai-sdk/openai` |
| Google Gemini | `@ai-sdk/google` |
| Google Vertex AI | `@ai-sdk/google-vertex` |
| Vertex Anthropic | `@ai-sdk/google-vertex/anthropic` |
| Amazon Bedrock | `@ai-sdk/amazon-bedrock` |
| Azure OpenAI | `@ai-sdk/azure` |
| xAI (Grok) | `@ai-sdk/xai` |
| Mistral | `@ai-sdk/mistral` |
| Groq | `@ai-sdk/groq` |
| DeepInfra | `@ai-sdk/deepinfra` |
| Cerebras | `@ai-sdk/cerebras` |
| Cohere | `@ai-sdk/cohere` |
| TogetherAI | `@ai-sdk/togetherai` |
| Perplexity | `@ai-sdk/perplexity` |
| Vercel | `@ai-sdk/vercel` |
| OpenRouter | `@openrouter/ai-sdk-provider` |
| AI Gateway | `@ai-sdk/gateway` |
| GitLab | `@gitlab/gitlab-ai-provider` |
| GitHub Copilot | Custom OpenAI-compatible SDK |
| Cloudflare Workers AI | Via OpenAI-compatible |
| Cloudflare AI Gateway | Via `ai-gateway-provider` |
| OpenAI-Compatible | `@ai-sdk/openai-compatible` (fallback pour tout provider custom) |
| SAP AI Core | Custom loader |

### 4.2 Model Registry

- Modeles charges depuis **models.dev** (`provider/models.ts`) - un registre centralisee de modeles
- Chaque modele a un schema Zod detaille: capabilities, cost, limits, variants
- Les capabilities incluent: temperature, reasoning, attachment, toolcall, input/output modalities (text, audio, image, video, pdf), interleaved thinking
- Systeme de **variants** par modele (ex: reasoning effort high/low)

### 4.3 Auth Flow

**`auth.ts`** + `provider/auth.ts` - Multiples methodes d'auth:

1. **API Key** - Via variables d'environnement ou `opencode auth`
2. **OAuth** - Flow OAuth complet (pour OpenAI Codex, GitHub Copilot, GitLab)
3. **Well-known** - Decouverte automatique via `.well-known/opencode`
4. **Plugin Auth** - Les plugins peuvent definir des methodes d'auth custom

Stockage des credentials dans le storage local (pas en clair dans la config).

### 4.4 Custom Loaders

Chaque provider peut avoir un **custom loader** (`CUSTOM_LOADERS`) qui gere:
- La decouverte automatique (autoload)
- La creation de modeles custom (getModel)
- Les options specifiques au provider

Exemple notable: Amazon Bedrock a un loader complexe qui gere le cross-region inference prefixing.

### 4.5 Streaming

Utilise `streamText()` du Vercel AI SDK v5 pour le streaming.
Le streaming est gere dans `session/llm.ts`:
- Wrapping du modele avec middleware pour transformer les messages
- Support du reasoning/thinking interleave
- Gestion fine des tokens (input, output, cache read/write, reasoning)

---

## 5. Tool System

**Fichiers**: `tool/tool.ts`, `tool/registry.ts`, `tool/*.ts`

### 5.1 Definition des Tools

```typescript
// tool/tool.ts
namespace Tool {
  interface Info<Parameters, Metadata> {
    id: string
    init: (ctx?: InitContext) => Promise<{
      description: string
      parameters: ZodType       // Schema Zod pour les parametres
      execute(args, ctx): Promise<{
        title: string
        metadata: Metadata
        output: string
        attachments?: FilePart[]
      }>
    }>
  }
}
```

Les tools sont definis via `Tool.define(id, init)` qui:
- Valide les parametres avec Zod avant execution
- Applique automatiquement la troncation de l'output
- Gere les erreurs de validation

### 5.2 Tools Built-in

| Tool | Fichier | Description |
|------|---------|-------------|
| `bash` | `bash.ts` | Execution de commandes shell (avec tree-sitter pour parser bash) |
| `read` | `read.ts` | Lecture de fichiers |
| `write` | `write.ts` | Ecriture de fichiers |
| `edit` | `edit.ts` | Edition de fichiers (remplacement de texte) |
| `glob` | `glob.ts` | Recherche de fichiers par pattern |
| `grep` | `grep.ts` | Recherche dans le contenu des fichiers |
| `list` | `ls.ts` | Listing de repertoires |
| `task` | `task.ts` | Lancement de sous-agents |
| `webfetch` | `webfetch.ts` | Fetch de pages web |
| `websearch` | `websearch.ts` | Recherche web (via Exa) |
| `codesearch` | `codesearch.ts` | Recherche de code (via Exa) |
| `question` | `question.ts` | Poser une question a l'utilisateur |
| `todowrite` | `todo.ts` | Gestion de todo list |
| `skill` | `skill.ts` | Execution de skills |
| `batch` | `batch.ts` | Execution en batch (experimental) |
| `apply_patch` | `apply_patch.ts` | Application de patches (format Codex) |
| `multiedit` | `multiedit.ts` | Editions multiples |
| `lsp` | `lsp.ts` | Integration LSP (experimental) |
| `plan` | `plan.ts` | Mode plan (enter/exit) |
| `invalid` | `invalid.ts` | Fallback pour les tool calls invalides |

### 5.3 Tool Registration

`tool/registry.ts` - `ToolRegistry`:
- Charge les tools built-in
- Decouvre les tools custom depuis les repertoires de config (`{tool,tools}/*.{js,ts}`)
- Charge les tools des plugins
- Filtre les tools par modele (ex: `apply_patch` uniquement pour GPT-5+)
- Les tools conditionnels: `websearch`/`codesearch` seulement pour OpenCode provider ou avec flag
- Le tool `batch` est experimental et necessite une config

### 5.4 Troncation

`tool/truncation.ts` - `Truncate`:
- Limite la taille des outputs de tools
- `MAX_LINES` et `MAX_BYTES` configurables via flags
- Ecrit les outputs tronques dans un fichier temporaire pour reference

### 5.5 Bash Tool Specifiquement

- Utilise **tree-sitter** (WebAssembly) pour parser les commandes bash
- Detecte le shell acceptable via `Shell.acceptable()`
- Timeout par defaut: 2 minutes (configurable)
- Support de `workdir` pour changer le repertoire d'execution
- Integration avec le systeme de permissions pour `BashArity`

---

## 6. MCP Integration

**Fichiers**: `mcp/index.ts`, `mcp/auth.ts`, `mcp/oauth-provider.ts`, `mcp/oauth-callback.ts`

### 6.1 MCP Client

OpenCode est un **client MCP** utilisant `@modelcontextprotocol/sdk` v1.25.2.

### 6.2 Transports Supportes

1. **StdioClientTransport** - Pour les serveurs MCP locaux (commande stdio)
2. **StreamableHTTPClientTransport** - Pour les serveurs MCP distants (HTTP streaming)
3. **SSEClientTransport** - Fallback SSE pour les serveurs distants

Tentative de connexion HTTP streaming d'abord, puis fallback SSE.

### 6.3 Configuration MCP

Deux types de serveurs MCP:

```jsonc
// Local
{
  "type": "local",
  "command": ["node", "server.js"],
  "environment": { "KEY": "value" },
  "enabled": true,
  "timeout": 30000
}

// Remote
{
  "type": "remote",
  "url": "https://mcp.example.com",
  "headers": { "Authorization": "Bearer xxx" },
  "oauth": { "clientId": "...", "scope": "..." },
  "timeout": 30000
}
```

### 6.4 OAuth pour MCP Distant

Support complet OAuth 2.0 pour les serveurs MCP distants:
- `McpOAuthProvider` - Gestion du flow OAuth
- `McpOAuthCallback` - Serveur de callback local pour le redirect
- Dynamic Client Registration (RFC 7591) en fallback
- Stockage securise des tokens
- Protection CSRF via state parameter

### 6.5 Tool Discovery

- `MCP.tools()` - Liste et convertit tous les tools MCP en tools AI SDK (`dynamicTool`)
- Prefixe les noms de tools avec le nom du serveur MCP (sanitise)
- Notification de changement de tools via `ToolListChangedNotification`
- Support des prompts MCP via `MCP.prompts()`
- Support des resources MCP via `MCP.resources()`
- Timeout configurable par serveur ou globalement

### 6.6 Status Management

Chaque serveur MCP a un status:
- `connected` - Connecte et fonctionnel
- `disabled` - Desactive dans la config
- `failed` - Erreur de connexion
- `needs_auth` - Authentification necessaire
- `needs_client_registration` - Client ID necessaire

---

## 7. Agent System

**Fichier**: `agent/agent.ts`

### 7.1 Agents Built-in

| Agent | Mode | Role |
|-------|------|------|
| `build` | primary | Agent par defaut. Execute les tools selon les permissions configurees. |
| `plan` | primary | Mode plan. Interdit toutes les modifications sauf les plans (.md). |
| `general` | subagent | Agent general pour les taches complexes multi-etapes. |
| `explore` | subagent | Agent rapide pour l'exploration de code (read-only). |
| `compaction` | primary (hidden) | Agent pour la compaction de contexte. |
| `title` | primary (hidden) | Agent pour generer les titres de session. |
| `summary` | primary (hidden) | Agent pour generer les resumes de session. |

### 7.2 Definition d'un Agent

```typescript
interface Agent.Info {
  name: string
  description?: string
  mode: "subagent" | "primary" | "all"
  native?: boolean        // Built-in ou custom
  hidden?: boolean        // Cache du menu
  topP?: number
  temperature?: number
  color?: string          // Hex ou theme color
  permission: Ruleset     // Regles de permission
  model?: { modelID, providerID }  // Modele dedie
  variant?: string
  prompt?: string         // System prompt custom
  options: Record<string, any>
  steps?: number          // Max iterations
}
```

### 7.3 Agents Custom

Les agents custom se definissent de trois manieres:

1. **Fichiers Markdown** dans `.opencode/agents/*.md` avec frontmatter YAML
2. **Configuration JSON** dans `opencode.json` sous `"agent": { ... }`
3. **Generation par IA** via `Agent.generate()` (utilise un LLM pour creer un agent)

### 7.4 Sub-Agent Communication

Le **TaskTool** (`tool/task.ts`) permet de lancer des sous-agents:
- Un sous-agent herite du contexte de la session parente
- Il a ses propres permissions (l'agent `explore` est read-only)
- Les resultats sont retournes au contexte parent
- Pas de communication bidirectionnelle directe entre sous-agents

### 7.5 System Prompts

`session/system.ts` - Prompts specifiques par famille de modele:
- `anthropic.txt` - Pour Claude
- `beast.txt` - Pour GPT (non-GPT5)
- `gemini.txt` - Pour Gemini
- `codex_header.txt` - Pour GPT-5 (Codex/Responses API)
- `trinity.txt` - Pour Trinity
- `qwen.txt` - Variante sans todo pour modeles tiers

---

## 8. Session & Conversation Management

**Fichiers**: `session/index.ts`, `session/message-v2.ts`, `session/prompt.ts`, `session/processor.ts`

### 8.1 Session

```typescript
interface Session.Info {
  id: string              // Identifiant descendant (ulid inverse)
  slug: string            // Slug human-readable
  projectID: string       // Projet associe
  directory: string       // Repertoire de travail
  parentID?: string       // Session parente (pour les threads enfants)
  title: string
  version: string         // Version d'OpenCode
  share?: { url: string } // URL de partage
  summary?: { additions, deletions, files, diffs }
  revert?: { messageID, partID, snapshot, diff }
  permission?: Ruleset    // Override de permissions pour cette session
  time: { created, updated, compacting?, archived? }
}
```

### 8.2 Messages

Format MessageV2 avec separation info/parts:
- **User messages**: texte, fichiers attaches, system prompt
- **Assistant messages**: role, modelID, agent, tokens, error, cost

### 8.3 Parts (sous-unites de message)

- `TextPart` - Texte genere
- `ReasoningPart` - Blocs de raisonnement/thinking
- `ToolPart` - Appels de tools avec status (pending, running, completed, error)
- `StepStartPart` / `StepFinishPart` - Debut/fin d'iteration

### 8.4 Session Processor

`session/processor.ts` - `SessionProcessor`:
- Boucle d'agent: streame les reponses du LLM et traite chaque part
- Gestion du streaming: text deltas, reasoning deltas, tool calls
- **Doom Loop Detection**: Detecte les boucles infinies (seuil = 3 repetitions)
- Retry automatique en cas d'erreur
- Compaction automatique quand le contexte deborde
- Snapshot git pour le revert

### 8.5 Compaction

`session/compaction.ts` - `SessionCompaction`:
- Detecte le depassement de contexte (`isOverflow`)
- **Pruning**: Supprime les outputs des vieux tool calls (garde les 40k derniers tokens)
- **Full Compaction**: Resume la conversation via un agent dedie (`compaction`)
- Seuil minimum: 20k tokens avant de commencer a pruner

### 8.6 Instructions

`session/instruction.ts` - `InstructionPrompt`:
- Charge les fichiers d'instructions du projet (equivalent de CLAUDE.md)
- Cherche dans les repertoires de config

### 8.7 Sharing

Support du partage de sessions via `share/share-next.ts`:
- Partage auto ou manuel (configurable)
- Generation d'URLs de partage

---

## 9. Permission System

**Fichiers**: `permission/next.ts`, `permission/arity.ts`, `permission/index.ts`

### 9.1 Modele de Permissions

Systeme base sur des regles (Ruleset):

```typescript
interface Rule {
  permission: string   // Type de permission (bash, edit, read, etc.)
  pattern: string      // Pattern glob (*, fichier specifique)
  action: "allow" | "deny" | "ask"
}
```

### 9.2 Permissions Disponibles

- `bash` - Execution de commandes shell
- `read` - Lecture de fichiers (par defaut: ask pour `*.env*`)
- `edit` - Modification de fichiers
- `glob`, `grep`, `list` - Recherche/listing
- `task` - Lancement de sous-agents
- `external_directory` - Acces hors du projet
- `question` - Questions a l'utilisateur
- `webfetch`, `websearch`, `codesearch` - Acces web
- `lsp` - Integration LSP
- `skill` - Execution de skills
- `doom_loop` - Protection boucle infinie
- `plan_enter`, `plan_exit` - Mode plan
- `todowrite`, `todoread` - Todo list

### 9.3 Resolution des Permissions

- Les regles sont evaluees dans l'ordre (derniere correspondance gagne)
- Supporte les patterns glob et l'expansion `~`, `$HOME`
- Fusionnable: `PermissionNext.merge()` combine plusieurs rulesets
- Les agents ont des permissions par defaut et des overrides

### 9.4 Permission Runtime

Quand une permission est `ask`:
1. Un event `permission.asked` est emis
2. Le TUI affiche un dialogue de confirmation
3. L'utilisateur peut repondre: allow, deny, allow always
4. "Always" ajoute une regle permanente

---

## 10. Config System

**Fichier**: `config/config.ts`

### 10.1 Format de Configuration

- **Format**: JSONC (JSON with Comments) via `jsonc-parser`
- **Fichier**: `opencode.json` ou `opencode.jsonc`
- **Schema**: Validation complete via Zod

### 10.2 Ordre de Precedence (bas -> haut)

1. Remote `.well-known/opencode` (config organisationnelle)
2. Config globale (`~/.config/opencode/opencode.json`)
3. Config custom (`OPENCODE_CONFIG`)
4. Config projet (`opencode.json` dans le projet)
5. Repertoires `.opencode/` (agents, commands, plugins, config)
6. Config inline (`OPENCODE_CONFIG_CONTENT`)
7. **Config managed** (`/etc/opencode/` - enterprise, priorite maximale)

### 10.3 Features de Configuration Notables

- **Variable substitution**: `{env:VARIABLE}` dans les configs
- **File inclusion**: `{file:path/to/file}` pour inclure du contenu
- **Plugin deduplication**: Les plugins de plus haute priorite ecrasent
- **Enterprise config**: `/etc/opencode/` ou `/Library/Application Support/opencode/` (admin-controlled)

### 10.4 Sections de Configuration

- `provider` - Configuration des providers LLM
- `mcp` - Serveurs MCP
- `agent` - Configuration des agents
- `command` - Commandes slash custom
- `skills` - Repertoires de skills additionnels
- `permission` - Regles de permission globales
- `keybinds` - Raccourcis clavier
- `tui` - Parametres TUI (scroll speed, diff style)
- `server` - Config serveur (port, hostname, mDNS, CORS)
- `formatter` - Configuration des formatteurs de code
- `lsp` - Serveurs LSP
- `plugin` - Plugins a charger
- `compaction` - Auto-compaction et pruning
- `experimental` - Features experimentales

---

## 11. Plugin System

**Fichiers**: `packages/plugin/src/index.ts`, `packages/opencode/src/plugin/index.ts`

### 11.1 Plugin API

```typescript
type Plugin = (input: PluginInput) => Promise<Hooks>

interface PluginInput {
  client: OpencodeClient    // SDK client
  project: Project          // Infos projet
  directory: string         // Repertoire de travail
  worktree: string          // Racine du worktree
  serverUrl: URL            // URL du serveur
  $: BunShell               // Bun shell pour les commandes
}
```

### 11.2 Hooks Disponibles

| Hook | Description |
|------|-------------|
| `event` | Ecoute tous les events |
| `config` | Reagit aux changements de config |
| `tool` | Definit des tools custom |
| `auth` | Fournit des methodes d'authentification pour les providers |
| `chat.message` | Appele quand un nouveau message est recu |
| `chat.params` | Modifie les parametres envoyes au LLM |
| `chat.headers` | Modifie les headers HTTP |
| `permission.ask` | Intercepte les demandes de permission |
| `command.execute.before` | Hook pre-execution de commande |
| `tool.execute.before` | Hook pre-execution de tool |
| `tool.execute.after` | Hook post-execution de tool |
| `shell.env` | Modifie les variables d'environnement shell |
| `experimental.chat.messages.transform` | Transforme l'historique de messages |
| `experimental.chat.system.transform` | Transforme le system prompt |
| `experimental.session.compacting` | Customise la compaction |
| `experimental.text.complete` | Hook post-generation de texte |

### 11.3 Plugins Built-in Internes

- `CodexAuthPlugin` - Auth OAuth OpenAI Codex
- `CopilotAuthPlugin` - Auth GitHub Copilot
- `GitlabAuthPlugin` - Auth GitLab (`@gitlab/opencode-gitlab-auth`)

### 11.4 Plugin Builtin NPM

- `opencode-anthropic-auth@0.0.13` - Auth Anthropic

### 11.5 Chargement des Plugins

1. **Internes**: Directement importes (pas npm)
2. **NPM**: Installes via `BunProc.install()` dans un cache
3. **Locaux**: Charges depuis `.opencode/plugins/*.{ts,js}`
4. **File URLs**: `file://` pour les plugins locaux absolus

---

## 12. Storage System

**Fichier**: `storage/storage.ts`

### 12.1 Format

- **File-based JSON storage** dans `~/.local/share/opencode/storage/`
- Chaque entite est un fichier JSON: `{type}/{id}.json`
- Structure de repertoires: `session/{projectID}/{sessionID}.json`, `message/{sessionID}/{messageID}.json`, etc.

### 12.2 Operations

- `Storage.read<T>(key[])` - Lecture avec lock
- `Storage.write<T>(key[], content)` - Ecriture avec lock
- `Storage.update<T>(key[], fn)` - Read-modify-write avec lock
- `Storage.remove(key[])` - Suppression
- `Storage.list(prefix[])` - Listing

### 12.3 Locking

Systeme de lock fichier (`util/lock.ts`) avec:
- Read locks (partages)
- Write locks (exclusifs)
- Utilise `using` pour le cleanup automatique (TC39 Explicit Resource Management)

### 12.4 Migrations

Systeme de migration integre (`MIGRATIONS[]`):
- Trackage du numero de migration dans un fichier
- Migration 0: Migration depuis l'ancien format projet
- Migration 1: Extraction des diffs de session

---

## 13. Server / API

**Fichier**: `server/server.ts`

### 13.1 Framework

- **Hono** (v4.10.7) - Framework HTTP
- Support SSE pour les events en temps reel
- OpenAPI auto-genere via `hono-openapi`
- CORS configure (localhost, tauri, opencode.ai)
- Auth optionnelle via Basic Auth

### 13.2 Routes API

| Route | Description |
|-------|-------------|
| `/session/*` | CRUD sessions, prompt, share |
| `/provider/*` | Liste providers et modeles |
| `/config/*` | Configuration |
| `/mcp/*` | Status et gestion MCP |
| `/permission/*` | Gestion des permissions runtime |
| `/question/*` | Questions a l'utilisateur |
| `/project/*` | Infos projet |
| `/pty/*` | Pseudo-terminal |
| `/tui/*` | Routes specifiques au TUI |
| `/file/*` | Operations fichiers |
| `/experimental/*` | Features experimentales |
| `/global/*` | Routes globales (cross-instance) |
| `/event` | SSE event stream |
| `/agent` | Liste des agents |
| `/skill` | Liste des skills |
| `/command` | Liste des commandes |
| `/vcs` | Info VCS (git branch) |
| `/path` | Chemins du systeme |
| `/lsp` | Status LSP |
| `/formatter` | Status formatteur |
| `/auth/:providerID` | CRUD auth credentials |
| `/log` | Ecriture de logs |
| `/doc` | OpenAPI spec |
| `/instance/dispose` | Nettoyage d'instance |

### 13.3 Event System

`bus/bus-event.ts` + `bus/index.ts`:
- Event bus pub/sub avec types Zod
- `Bus.publish()` / `Bus.subscribeAll()`
- SSE streaming vers les clients
- Heartbeat toutes les 30s
- Events: session.*, message.*, permission.*, mcp.*, etc.

### 13.4 mDNS

`server/mdns.ts` - Discovery mDNS via `bonjour-service` pour decouvrir les instances OpenCode sur le reseau local.

---

## 14. Desktop App

**Dossier**: `packages/desktop/`

### 14.1 Stack

- **Tauri 2** (v2.9.5) avec Rust backend
- Frontend: reutilise le meme code UI que l'app web
- Vite pour le build frontend

### 14.2 Plugins Tauri

- `tauri-plugin-opener` - Ouverture de fichiers/URLs
- `tauri-plugin-deep-link` - Deep links
- `tauri-plugin-shell` - Execution de commandes
- `tauri-plugin-dialog` - Dialogues natifs
- `tauri-plugin-updater` - Auto-update
- `tauri-plugin-process` - Gestion des processus
- `tauri-plugin-store` - Stockage cle-valeur
- `tauri-plugin-window-state` - Persistence de l'etat des fenetres
- `tauri-plugin-clipboard-manager` - Presse-papiers
- `tauri-plugin-http` - Requetes HTTP
- `tauri-plugin-notification` - Notifications systeme
- `tauri-plugin-single-instance` - Instance unique avec deep-link
- `tauri-plugin-os` - Infos OS

### 14.3 Ce que le Desktop Ajoute

- Fenetre native avec titre bar custom (`macos-private-api`)
- Auto-update integre
- Deep links (`opencode://`)
- Notifications systeme natives
- Instance unique (pas de double lancement)
- Persistence de position/taille de fenetre

---

## 15. SDK (packages/sdk/js/)

**Fichiers**: `packages/sdk/js/src/index.ts`, `packages/sdk/js/src/client.ts`

### 15.1 SDK Client

```typescript
function createOpencodeClient(config?: { baseUrl, directory, fetch }): OpencodeClient
```

- Client HTTP genere automatiquement depuis l'OpenAPI spec
- Namespace `OpencodeClient` avec methodes typees pour chaque endpoint
- Support de custom fetch (pour l'usage in-process)
- Header `x-opencode-directory` pour specifier le repertoire

### 15.2 SDK Server

```typescript
async function createOpencodeServer(options?: ServerOptions): Promise<{ url, server }>
```

Permet d'embarquer un serveur OpenCode dans un processus.

### 15.3 SDK v2

Dossier `packages/sdk/js/src/v2/` - Version 2 du SDK avec l'API streaming events.

---

## 16. Extensions

### 16.1 Zed Extension

`packages/extensions/zed/` - Extension minimale:
- `extension.toml` avec configuration
- Integration d'OpenCode dans l'editeur Zed

### 16.2 VS Code Extension

`sdks/vscode/` - Extension VS Code completes.

---

## 17. ACP (Agent Communication Protocol)

**Dossier**: `packages/opencode/src/acp/`

- `agent.ts` - Agent ACP
- `session.ts` - Sessions ACP
- `types.ts` - Types ACP
- Utilise `@agentclientprotocol/sdk` v0.14.1
- Protocole pour la communication inter-agents standardisee

---

## 18. Skill System

**Fichier**: `skill/skill.ts`, `skill/discovery.ts`

### 18.1 Format

Les skills sont des fichiers `SKILL.md` avec frontmatter YAML:

```markdown
---
name: my-skill
description: Description de la skill
---

Contenu du skill (prompt/instructions)
```

### 18.2 Discovery

Recherche les skills dans:
1. `.opencode/skills/**/SKILL.md` (projet)
2. `~/.opencode/skills/**/SKILL.md` (global)
3. `.claude/skills/**/SKILL.md` et `.agents/skills/**/SKILL.md` (compatibilite)
4. Paths additionnels via config `skills.paths`
5. URLs via config `skills.urls`

### 18.3 Compatibilite Claude Code

OpenCode cherche explicitement dans `.claude/` et `.agents/` pour la compatibilite avec Claude Code.

---

## 19. Autres Systemes Notables

### 19.1 LSP Integration

`lsp/` - Serveurs LSP integres pour le formatting et les diagnostics.
- Lancement automatique de serveurs LSP selon les extensions de fichiers
- Support de serveurs LSP custom via config

### 19.2 Formatter

`format/` - Formatage de code post-edition:
- Detection automatique du formatteur selon l'extension
- Commandes custom configurables

### 19.3 Snapshot & Revert

`snapshot/` - Snapshots git pour le revert:
- Cree un snapshot git avant les modifications
- Permet de revenir en arriere sur une serie de modifications

### 19.4 Worktree

`worktree/` - Gestion des git worktrees pour l'isolation.

### 19.5 PTY

`pty/` - Pseudo-terminal via `bun-pty` v0.4.8.
- Expose des terminaux via l'API
- Utilise pour le shell interactif dans le TUI/web

### 19.6 Command System

`command/` - Systeme de commandes slash:
- Definies en Markdown avec frontmatter
- Templates avec substitution de variables
- Support d'agent et modele dedies par commande

### 19.7 Installation & Upgrade

`installation/` - Gestion de l'installation:
- Detection du mode d'installation (npm, homebrew, nix, local)
- Version tracking
- Auto-update

---

## 20. Comparaison avec les Concurrents

### Points Forts d'OpenCode

1. **Support massif de providers** - 20+ providers LLM natifs (le plus large de tous les CLI agents)
2. **Plugin system mature** - API riche avec 15+ hooks
3. **MCP complet** - Client MCP avec OAuth, transports multiples, prompts et resources
4. **TUI SolidJS** - Interface terminal reactive et tres configurable (70+ keybinds)
5. **Multi-plateforme** - CLI, TUI, Web, Desktop (Tauri)
6. **SDK client** - API ouverte pour l'integration
7. **Enterprise features** - Config managed, permissions granulaires
8. **ACP support** - Agent Communication Protocol
9. **Skill system** - Compatible avec les skills Claude Code

### Points Faibles

1. **Dependance a Bun** - Pas de support Node.js natif
2. **Complexite** - Monorepo massif, courbe d'apprentissage elevee
3. **Build time** - TypeScript natif preview, build complexe

### Differences Cles vs Claude Code

| Feature | OpenCode | Claude Code |
|---------|----------|-------------|
| Runtime | Bun | Node.js |
| TUI | SolidJS (@opentui) | Ink (React) |
| Providers | 20+ via AI SDK | Anthropic only |
| Plugin API | Plugin package + hooks | Non |
| MCP | Client complet | Client complet |
| Desktop | Tauri 2 | Non |
| Web UI | SolidJS (SolidStart) | Non |
| Config | JSONC + Markdown agents | CLAUDE.md |
| Skill compat | .claude/ + .opencode/ | .claude/ |
| Open Source | MIT | Non (source-available) |
| Agent Protocol | ACP support | Proprietary |

---

## 21. Architecture Diagram (Textual)

```
                    CLI (yargs)
                        |
            +-----------+-----------+
            |           |           |
          TUI         run         serve
       (SolidJS)    (headless)   (HTTP)
            |           |           |
            +-----------+-----------+
                        |
                 Server (Hono)
                    /   |   \
                   /    |    \
            Session  Provider  MCP
               |        |       |
          Processor   AI SDK  Client
               |        |       |
           LLM.stream  Model   Tools
               |        |       |
          Tool System   |   MCP Tools
               |        |
           Permission   |
               |        |
            Storage   Plugin
           (JSON FS)  System
```
