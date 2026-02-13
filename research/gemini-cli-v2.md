# Gemini CLI - Reverse Engineering Complet (v2)

**Source** : `/home/pedro/PROJETS-AI/gemini-cli/` (TypeScript source, monorepo)
**Date** : 2026-02-10
**Methode** : Lecture directe du code source TypeScript

---

## 1. Architecture Generale

### 1.1 Structure Monorepo

```
packages/
  cli/      -> 812 fichiers TS - Interface utilisateur (React/Ink TUI)
  core/     -> 553 fichiers TS - Logique metier, tools, agents, policy
  a2a-server/   -> Serveur Agent-to-Agent (protocole Google A2A)
  test-utils/   -> Utilitaires de test partages
  vscode-ide-companion/ -> Extension VSCode companion
```

Total : **1365 fichiers TypeScript**.

### 1.2 Point d'entree principal

**Fichier** : `packages/cli/src/gemini.tsx`

La fonction `main()` (ligne ~100) orchestre le demarrage :

1. **Chargement des settings** via `loadSettings()` (5 niveaux de precedence)
2. **Configuration DNS** via `setDefaultResultOrder()` (ipv4first par defaut)
3. **Refresh auth** via `refreshAuth()` (tokens Google)
4. **Detection sandbox** : verifie si deja dans un sandbox, sinon lance `start_sandbox()`
5. **Relaunch memoire** : `relaunchAppInChildProcess()` alloue 50% RAM via `getNodeMemoryArgs()` (calcule V8 heap = `totalMem * 0.5`, verifie `GEMINI_CLI_NO_RELAUNCH`)
6. **Session resume** : `--resume` restaure une session precedente
7. **Branchement** : interactif (React/Ink) vs non-interactif (pipe/prompt)

### 1.3 Mode Interactif

**Fonction** : `startInteractiveUI()` dans `gemini.tsx`

Arbre de composants React/Ink :
```
SettingsContext
  > KeypressProvider
    > MouseProvider
      > TerminalProvider
        > ScrollProvider
          > SessionStatsProvider
            > VimModeProvider
              > AppContainer
```

### 1.4 Mode Non-Interactif

**Fichier** : `packages/cli/src/nonInteractiveCli.ts`

`runNonInteractive()` (534 lignes) :
- Cree un `Scheduler` pour la boucle d'execution
- Traite les `@commands` et `/commands` dans le prompt
- Boucle de turns avec `geminiClient.sendMessageStream()`
- Formats de sortie : `OutputFormat.JSON`, `STREAM_JSON`, `TEXT`
- Ctrl+C via readline keypress events + AbortController

### 1.5 Modele de Processus

Le CLI se relance lui-meme dans un processus fils :
- **Parent** : detecte la memoire, configure les args V8, lance le fils
- **Fils** : processus reel avec `--max-old-space-size` augmente
- **IPC** : `setupAdminControlsListener()` recoit des settings admin du parent via messages IPC
- Variable `GEMINI_CLI_NO_RELAUNCH` empeche la boucle infinie de relaunch

---

## 2. Systeme de Tools (13+ outils)

### 2.1 Architecture des Tools

**Fichier principal** : `packages/core/src/tools/tools.ts` (828 lignes)

#### Hierarchie de classes :

```
DeclarativeTool (interface)
  build() -> ToolInvocation
  buildAndExecute()
  validateBuildAndExecute()

BaseDeclarativeTool<TParams, TResult> (classe abstraite)
  - Validation des schemas via SchemaValidator.validate()
  - createInvocation() (abstract) -> ToolInvocation

ToolInvocation<TParams, TResult> (interface)
  shouldConfirmExecute() -> ToolCallConfirmationDetails | false
  execute(abortSignal) -> TResult
  getDescription() -> string
  toolLocations() -> ToolLocation[]

BaseToolInvocation (classe abstraite)
  - getMessageBusDecision() : envoie au PolicyEngine via MessageBus
  - Timeout 30s pour la reponse du bus
  - publishPolicyUpdate() pour "Always allow"
```

#### Kind enum (`tools.ts:Kind`) :
```typescript
enum Kind {
  Read, Edit, Delete, Move, Search, Execute, Think, Fetch, Communicate, Plan, Other
}
```

#### ToolConfirmationOutcome :
- `ProceedOnce` : approuve une fois
- `ProceedAlways` : approuve toujours (session)
- `ProceedAlwaysAndSave` : approuve toujours + sauvegarde dans policy TOML
- `ProceedAlwaysTool` : approuve tout l'outil
- `ProceedAlwaysServer` : approuve tout le serveur MCP
- `ModifyWithEditor` : ouvre dans l'editeur pour modifier
- `Cancel` : refuse

#### 6 types de confirmation details :
1. `edit` - ToolEditConfirmationDetails (diff de fichier)
2. `exec` - ToolExecuteConfirmationDetails (commande shell)
3. `mcp` - ToolMcpConfirmationDetails (outil MCP)
4. `info` - ToolInfoConfirmationDetails (info generique)
5. `ask_user` - ToolAskUserConfirmationDetails
6. `exit_plan_mode` - ToolExitPlanModeConfirmationDetails

### 2.2 Registre des Tools

**Fichier** : `packages/core/src/tools/tool-registry.ts` (602 lignes)

`ToolRegistry` :
- `allKnownTools` : Map<string, DeclarativeTool>
- `discoverAllTools()` : supprime les anciens outils decouverts, lance `discoverAndRegisterToolsFromCommand()` (limite stdout 10MB)
- Types : `DiscoveredTool` (command discovery) et `DiscoveredMCPTool`
- **Tri** : built-in d'abord, puis discovered, puis MCP (par nom de serveur)
- `getActiveTools()` : filtre les outils exclus avec expansion des legacy aliases

### 2.3 Noms des Tools

**Fichier** : `packages/core/src/tools/tool-names.ts` (163 lignes)

#### Tools built-in :
| Nom interne | Outil |
|---|---|
| `glob` | Glob (recherche fichiers) |
| `grep_search` | Grep (recherche contenu) |
| `read_file` | ReadFile |
| `read_many_files` | ReadManyFiles |
| `list_directory` | ListDirectory (ls) |
| `write_new_file` | WriteFile |
| `replace` | Edit (remplacement) |
| `run_shell_command` | Shell |
| `google_web_search` | WebSearch |
| `web_fetch` | WebFetch |
| `save_memory` | Memory (GEMINI.md) |
| `write_todos` | WriteTodos |
| `activate_skill` | ActivateSkill |
| `ask_user` | AskUser |

#### Plan Mode tools (lecture seule) :
`glob`, `grep_search`, `read_file`, `list_directory`, `google_web_search`, `ask_user`, `exit_plan_mode`

#### Alias legacy :
`search_file_content` -> `grep_search`

#### Format MCP : `server__tool` (double underscore)

### 2.4 Shell Tool

**Fichier** : `packages/core/src/tools/shell.ts` (524 lignes)

`ShellTool` / `ShellToolInvocation` :

- **Execution** via `ShellExecutionService.execute()`
- **Wrapping commande** (non-Windows) : enveloppe dans `{ command; }; __code=$?; pgrep -g 0 >tempfile; exit $__code;` pour capturer les PIDs de fond
- **Background** : `is_background` parameter, delay 200ms puis `ShellExecutionService.background(pid)`
- **Timeout inactivite** : configurable, abort si pas de sortie pendant N minutes
- **Detection binaire** : `binary_detected` event, arrete le streaming
- **Summarization** : optionnelle via LLM (`summarizer-shell` model)
- **Policy update** : fournit `commandPrefix` depuis les root commands parses (`getCommandRoots()`, `stripShellWrapper()`)
- **Redirection** : downgrade ALLOW -> ASK_USER pour commandes avec `>`, `>>`, `|` (sauf flag `allowRedirection`)

### 2.5 Edit Tool

**Fichier** : `packages/core/src/tools/edit.ts` (1057 lignes)

`EditTool` (nom: `replace`) - **3 strategies de remplacement en cascade** :

1. **`calculateExactReplacement()`** : match litteral avec `safeLiteralReplace()` - plus rapide
2. **`calculateFlexibleReplacement()`** : match ligne par ligne en ignorant les espaces de debut/fin, preserve l'indentation originale
3. **`calculateRegexReplacement()`** : tokenise le old_string en mots, insere `\s*` entre chaque token pour un match flexible

- **LLM self-correction** : `FixLLMEditWithInstruction()` quand les 3 strategies echouent
- `expected_replacements` : support pour remplacements multiples
- **IDE integration** : `IdeClient.openDiff()` pour review dans VS Code

### 2.6 WriteFile Tool

**Fichier** : `packages/core/src/tools/write-file.ts` (547 lignes)

`WriteFileTool` :
- **LLM correction** via `ensureCorrectEdit()` (fichier existant) ou `ensureCorrectFileContent()` (nouveau fichier)
- `getCorrectedFileContent()` : lit le fichier existant, applique la correction LLM, retourne contenu corrige
- **Preservation CRLF** : `detectLineEnding()` detecte si le fichier original utilise `\r\n`
- Implemente `ModifiableDeclarativeTool` : l'utilisateur peut editer le contenu propose avant ecriture
- `ai_proposed_content` / `modified_by_user` : tracking des modifications utilisateur

### 2.7 ReadFile Tool

**Fichier** : `packages/core/src/tools/read-file.ts` (249 lignes)

- Lecture avec offset/limit optionnels
- Validation d'acces via `config.validatePathAccess()`
- Respect des patterns d'ignore via `FileDiscoveryService`
- Telemetrie de lecture

### 2.8 Glob Tool

**Fichier** : `packages/core/src/tools/glob.ts`

`GlobTool` :
- Parametres : `pattern`, `dir_path`, `case_sensitive`, `respect_gitignore`
- **Tri intelligent** : `sortFileEntries()` - fichiers recents d'abord (par `mtimeMs`), puis alphabetique pour les anciens
- Utilise la librairie `glob` avec `escape()`

### 2.9 Grep Tool

**Fichier** : `packages/core/src/tools/grep.ts`

`GrepTool` :
- Parametres : `pattern` (regex), `dir_path`, `include` (glob filter)
- **Strategies multiples** : `git grep` (si repo git), system `grep`, fallback `globStream`
- Limites : `DEFAULT_TOTAL_MAX_MATCHES`, `DEFAULT_SEARCH_TIMEOUT_MS`
- Utilise `FileExclusions` pour les patterns d'exclusion

### 2.10 WebSearch Tool

**Fichier** : `packages/core/src/tools/web-search.ts`

`WebSearchTool` (nom: `google_web_search`) :
- Utilise le `GeminiClient` directement avec Grounding metadata
- Retourne `WebSearchToolResult` avec sources (GroundingChunks, GroundingSupport)
- Pas de fetch HTTP direct : le LLM fait la recherche via API Gemini

### 2.11 WebFetch Tool

**Fichier** : `packages/core/src/tools/web-fetch.ts`

`WebFetchTool` :
- **Timeout** : `URL_FETCH_TIMEOUT_MS = 10000` (10s)
- **Limite contenu** : `MAX_CONTENT_LENGTH = 100000` (100k chars)
- `parsePrompt()` : extrait les URLs valides du texte
- Protection contre les IPs privees : `isPrivateIp()`
- Conversion HTML -> text via `html-to-text` (`convert()`)
- **Retry** : `retryWithBackoff()` pour les erreurs reseau
- Grounding metadata support (comme WebSearch)
- Protocoles autorises : `http:` et `https:` uniquement

### 2.12 Memory Tool

**Fichier** : `packages/core/src/tools/memoryTool.ts`

`MemoryTool` (nom: `save_memory`) :
- Sauvegarde dans `GEMINI.md` (configurable via `setGeminiMdFilename()`)
- Section cible : `## Gemini Added Memories`
- Chemin : `~/.gemini/GEMINI.md` (global)
- Implemente `ModifiableDeclarativeTool` pour review utilisateur
- **Contexte global uniquement** : interdit de sauvegarder du contexte workspace-specific

---

## 3. Systeme d'Agents

### 3.1 Types d'Agents

**Fichier** : `packages/core/src/agents/types.ts` (207 lignes)

#### LocalAgentDefinition :
```typescript
{
  kind: 'local',
  name: string,           // slug
  description: string,
  displayName?: string,
  promptConfig: {
    systemPrompt: string,
    query: string,        // template avec ${input_name}
  },
  modelConfig?: {
    model?: string,       // 'inherit' pour heriter du parent
    temperature?: number,
  },
  runConfig?: {
    maxTurns: number,     // default 15
    maxTimeMinutes: number, // default 5
  },
  toolConfig?: {
    tools: string[],      // noms d'outils autorises
  },
  inputConfig: {
    inputSchema: object,  // JSON schema des parametres
  },
}
```

#### RemoteAgentDefinition :
```typescript
{
  kind: 'remote',
  agentCardUrl: string,   // URL de la carte agent A2A
  a2aAuth?: A2AAuthConfig,
}
```

#### AgentTerminateMode :
`ERROR`, `TIMEOUT`, `GOAL`, `MAX_TURNS`, `ABORTED`

### 3.2 Chargement des Agents

**Fichier** : `packages/core/src/agents/agentLoader.ts` (380 lignes)

- Format : fichiers Markdown (`.md`) avec frontmatter YAML
- **FRONTMATTER_REGEX** : `/^---\r?\n([\s\S]*?)\r?\n---(?:\r?\n([\s\S]*))?/`
- `parseAgentMarkdown()` : parse le YAML + body
- Validation Zod : `localAgentSchema` et `remoteAgentSchema`
- `markdownToAgentDefinition()` : conversion DTO -> AgentDefinition
- `loadAgentsFromDirectory()` : scanne `*.md` (exclut prefixe `_`)

### 3.3 Registre des Agents

**Fichier** : `packages/core/src/agents/registry.ts` (484 lignes)

`AgentRegistry` :

#### Agents built-in :
- `CodebaseInvestigatorAgent` : exploration de codebase
- `CliHelpAgent` : aide CLI
- `GeneralistAgent` : agent generaliste

#### Ordre de chargement :
1. Built-in
2. User (`~/.gemini/agents/`)
3. Project (`.gemini/agents/` - necessite trust + acknowledgment)
4. Extensions

#### Securite :
- Les agents projet necessitent un **acknowledgment par hash** via `AcknowledgedAgentsService`
- Les agents locaux recoivent politique ALLOW par defaut
- Les agents remote recoivent politique ASK_USER par defaut
- Heritage de modele : `model: 'inherit'` utilise le modele du parent

### 3.4 SubagentToolWrapper

**Fichier** : `packages/core/src/agents/subagent-tool-wrapper.ts` (91 lignes)

`SubagentToolWrapper extends BaseDeclarativeTool<AgentInputs, ToolResult>` :
- Expose un sous-agent comme un `DeclarativeTool` standard
- Genere dynamiquement le JSON schema depuis `inputConfig.inputSchema`
- `createInvocation()` : dispatch vers `LocalSubagentInvocation` ou `RemoteAgentInvocation` selon `definition.kind`

### 3.5 SubagentTool (v2)

**Fichier** : `packages/core/src/agents/subagent-tool.ts`

`SubagentTool` : version plus recente (copyright 2026) avec :
- Validation du schema a la construction
- `SubAgentInvocation` interne qui delegue a un `SubagentToolWrapper` pour l'execution reelle
- Confirmation delegue au sous-invocation

### 3.6 Execution locale d'agent

**Fichier** : `packages/core/src/agents/local-invocation.ts` (144 lignes)

`LocalSubagentInvocation` :
- Cree un `LocalAgentExecutor` via `LocalAgentExecutor.create()`
- Streaming : callback `onActivity()` bridge les events (`THOUGHT_CHUNK`) vers `updateOutput()`
- Resultat : `terminate_reason` + `result` text
- Erreur : ToolErrorType.EXECUTION_FAILED

---

## 4. Systeme de Policy

### 4.1 Policy Engine

**Fichier** : `packages/core/src/policy/policy-engine.ts` (519 lignes)

`PolicyEngine` :
- **Rules** : triees par priorite decroissante
- **SafetyCheckerRules** : checkers de securite (peuvent override en DENY/ASK_USER)
- **HookCheckerRules** : hooks qui agissent comme checkers de policy

#### `check(toolName, args)` :
1. Essaie les aliases de l'outil (`getToolAliases()`)
2. Pour les MCP : essaie `serverName__toolName` et `serverName__*` (wildcard)
3. Pour shell : delegue a `checkShellCommand()` qui parse les sous-commandes recursivement
4. **Redirection downgrade** : ALLOW -> ASK_USER si la commande a des redirections (sauf `allowRedirection`)
5. En mode non-interactif : ASK_USER devient DENY

#### Decisions :
- `PolicyDecision.ALLOW` : execution autorisee
- `PolicyDecision.DENY` : bloquee (avec `denyMessage` optionnel)
- `PolicyDecision.ASK_USER` : demande confirmation

### 4.2 Types de Policy

**Fichier** : `packages/core/src/policy/types.ts` (287 lignes)

```typescript
interface PolicyRule {
  toolName: string,
  argsPattern?: RegExp,       // match sur les arguments
  decision: PolicyDecision,
  priority: number,
  modes?: ApprovalMode[],     // dans quels modes appliquer
  allowRedirection?: boolean, // bypass du downgrade shell
  source?: string,
  denyMessage?: string,
}
```

#### ApprovalMode :
- `DEFAULT` : tout demande confirmation
- `AUTO_EDIT` : edits auto-approuves
- `YOLO` : tout auto-approuve
- `PLAN` : mode planification (tools lecture seule)

#### HookSource :
`'project'`, `'user'`, `'system'`, `'extension'`

### 4.3 TOML Policy Loader

**Fichier** : `packages/core/src/policy/toml-loader.ts` (466 lignes)

Format TOML avec validation Zod :

```toml
[[rules]]
toolName = "run_shell_command"
commandPrefix = "git"
decision = "ALLOW"
priority = 100
```

#### Systeme de tiers (priorite finale = `tier + priority/1000`) :
| Tier | Valeur | Source |
|------|--------|--------|
| default | 1 | `.gemini/policies/` |
| user | 2 | `~/.gemini/policies/` |
| admin | 3 | systeme `/etc/` |

#### Syntaxe shell :
- `commandPrefix` : prefixe de commande (ex: `"git"`, `["npm", "yarn"]`)
- `commandRegex` : regex sur la commande (mutuellement exclusif avec commandPrefix)
- Les deux necessitent `toolName = "run_shell_command"`

#### MCP :
- `mcpName` : transforme en format `mcpName__toolName`

#### Safety checkers :
```toml
[[safety_checker]]
type = "in_process"
name = "ALLOWED_PATH"
# OU
type = "external"
name = "my-checker"
command = "./check.sh"
```

---

## 5. Systeme de Hooks

### 5.1 HookSystem

**Fichier** : `packages/core/src/hooks/hookSystem.ts` (428 lignes)

`HookSystem` coordonne : `HookRegistry`, `HookRunner`, `HookAggregator`, `HookPlanner`, `HookEventHandler`

#### 11 types d'events :

| Event | Peut bloquer | Peut modifier |
|-------|-------------|---------------|
| `BeforeTool` | Oui | Input du tool |
| `AfterTool` | Non | Non |
| `BeforeAgent` | Non | Non |
| `AfterAgent` | Non | clearContext |
| `SessionStart` | Non | Non |
| `SessionEnd` | Non | Non |
| `PreCompress` | Non | Non |
| `BeforeModel` | Oui | Request LLM (+ reponse synthetique) |
| `AfterModel` | Non | Response LLM |
| `BeforeToolSelection` | Non | Tool config |
| `Notification` | Non | Non |

### 5.2 Types de Hooks

**Fichier** : `packages/core/src/hooks/types.ts` (668 lignes)

```typescript
interface HookConfig {
  type: 'command',
  command: string,
  name?: string,
  description?: string,
  timeout?: number,
  env?: Record<string, string>,
}
```

#### ConfigSource : `Project`, `User`, `System`, `Extensions`

#### Outputs specialises :
- `BeforeModelHookOutput` : peut fournir une `syntheticResponse` (bypass le LLM) ou modifier la request
- `AfterModelHookOutput` : peut modifier la reponse du LLM
- `BeforeToolSelectionHookOutput` : peut modifier la config des tools
- `BeforeToolHookOutput` : peut modifier les inputs d'un tool
- `AfterAgentHookOutput` : peut `clearContext`

#### McpToolContext :
```typescript
{
  server_name: string,
  tool_name: string,
  // connection info:
  command?: string, args?: string[], cwd?: string,  // stdio
  url?: string,  // SSE/HTTP
  tcp?: { host, port },  // WebSocket
}
```

---

## 6. Systeme Sandbox

**Fichier** : `packages/cli/src/utils/sandbox.ts` (860 lignes)

`start_sandbox()` supporte 3 backends :

### 6.1 Docker / Podman

- Flags : `--rm --init`
- Montages : workdir, user settings, tmpdir, homedir, gcloud config, ADC credentials
- **Proxy** : `GEMINI_SANDBOX_PROXY_COMMAND` avec isolation reseau Docker interne
- **Utilisateur** : cree un user dans le container matching host UID/GID via `useradd` dans l'entrypoint
- Variables d'environnement passees : `GEMINI_API_KEY`, `GOOGLE_API_KEY`, `GEMINI_MODEL`, `TERM`, `COLORTERM`, vars IDE, `VIRTUAL_ENV`

### 6.2 macOS Seatbelt

- Utilise `sandbox-exec` avec profils `.sb`
- Profils built-in + custom depuis `.gemini/`
- Jusqu'a 5 `INCLUDE_DIRs`

### 6.3 Detection

- Detecte si deja dans un sandbox avant de relancer
- Variable d'environnement de flag

---

## 7. Systeme MCP (Model Context Protocol)

### 7.1 Structure MCP

**Repertoire** : `packages/core/src/mcp/`

Fichiers :
- `auth-provider.ts` : interface `McpAuthProvider extends OAuthClientProvider` avec `getRequestHeaders()`
- `oauth-provider.ts` : OAuth complet avec PKCE (RFC 7636), Dynamic Client Registration (RFC 7591)
- `google-auth-provider.ts` : auth specifique Google
- `sa-impersonation-provider.ts` : Service Account impersonation
- `oauth-token-storage.ts` : stockage des tokens OAuth
- `oauth-utils.ts` : utilitaires OAuth

### 7.2 Token Storage

**Repertoire** : `packages/core/src/mcp/token-storage/`

Hierarchie :
- `base-token-storage.ts` : classe de base abstraite
- `file-token-storage.ts` : stockage fichier
- `keychain-token-storage.ts` : stockage keychain OS
- `hybrid-token-storage.ts` : combine file + keychain

### 7.3 OAuth Provider

**Fichier** : `packages/core/src/mcp/oauth-provider.ts`

```typescript
interface MCPOAuthConfig {
  enabled?: boolean,
  clientId?: string,
  clientSecret?: string,
  authorizationUrl?: string,
  tokenUrl?: string,
  scopes?: string[],
  audiences?: string[],
  redirectUri?: string,
  tokenParamName?: string,  // pour SSE
  registrationUrl?: string,
}
```

- Flow PKCE complet avec serveur HTTP local pour le callback
- Dynamic Client Registration (RFC 7591) si pas de clientId
- `openBrowserSecurely()` pour l'ouverture du navigateur
- Consentement utilisateur via `getConsentForOauth()`

### 7.4 MCP Server Enablement

**Fichier** : `packages/cli/src/config/mcp/mcpServerEnablement.ts`

`McpServerEnablementManager` :
- Gestion de l'etat d'activation des serveurs MCP
- `canLoadServer()` : verifie si un serveur peut etre charge
- `normalizeServerId()` : normalisation des identifiants
- `isInSettingsList()` : verification dans la config

### 7.5 Naming Convention MCP

Format des noms d'outils MCP : `serverName__toolName` (double underscore)
- Wildcard supporte dans les policies : `serverName__*`
- Validation : `isValidToolName()` dans `tool-names.ts` verifie le format `server__tool` avec regex `/^[a-z0-9-_]+$/i`

---

## 8. Systeme de Skills

### 8.1 SkillManager

**Fichier** : `packages/core/src/skills/skillManager.ts` (199 lignes)

Precedence (du plus bas au plus haut) :
1. **Built-in** (lowest)
2. **Extensions**
3. **User** (`~/.gemini/skills/`)
4. **Workspace** (`.gemini/skills/`) - necessite trusted folder (highest)

Cherche aussi dans les alias agents :
- `~/.gemini/agents/skills/`
- `.gemini/agents/skills/`

### 8.2 SkillLoader

**Fichier** : `packages/core/src/skills/skillLoader.ts` (191 lignes)

Format `SKILL.md` :
```markdown
---
name: my-skill
description: Description du skill
---

Instructions du skill ici...
```

- Pattern de decouverte : `SKILL.md` ou `*/SKILL.md` dans les repertoires de skills
- Frontmatter YAML avec `name` et `description`
- Le body du markdown EST le contenu du skill (instructions systeme)

---

## 9. Systeme d'Extensions

### 9.1 ExtensionManager

**Fichier** : `packages/cli/src/config/extension-manager.ts`

`ExtensionManager extends ExtensionLoader` :

```typescript
interface ExtensionManagerParams {
  enabledExtensionOverrides?: string[],
  settings: MergedSettings,
  requestConsent: (consent: string) => Promise<boolean>,
  requestSetting: ((setting: ExtensionSetting) => Promise<string>) | null,
  workspaceDir: string,
  eventEmitter?: EventEmitter<ExtensionEvents>,
  clientVersion?: string,
}
```

#### Fonctionnalites :
- `ExtensionEnablementManager` : active/desactive les extensions
- Installation depuis GitHub : `cloneFromGit()`, `downloadFromGitHubRelease()`, `tryParseGithubUrl()`
- **Consentement** : `maybeRequestConsentOrFail()` avant chargement
- **Settings d'extension** : `maybePromptForSettings()`, scoped (`ExtensionSettingScope`)
- **Variables** : `recursivelyHydrateStrings()` pour templating dans la config
- **Themes** : `themeManager.registerExtensionThemes()` / `unregisterExtensionThemes()`
- Telemetrie : events install/uninstall/enable/disable/update

#### Commandes CLI d'extension :
```
gemini extensions install <url>
gemini extensions uninstall <name>
gemini extensions enable <name>
gemini extensions disable <name>
gemini extensions list
gemini extensions update [name]
gemini extensions configure <name>
gemini extensions link <path>
gemini extensions new
gemini extensions validate
```

### 9.2 Contenu d'une Extension

Une extension peut fournir :
- **MCP servers** : configurations `MCPServerConfig`
- **Hooks** : `HookDefinition[]` par event
- **Agents** : charges via `loadAgentsFromDirectory()`
- **Skills** : charges via `loadSkillsFromDir()`
- **Themes** : `CustomTheme[]`
- **Settings** : `ResolvedExtensionSetting[]`
- **Policies** : regles de policy
- **Config** : `EXTENSIONS_CONFIG_FILENAME` avec metadata

---

## 10. Systeme de Themes

### 10.1 Theme Class

**Fichier** : `packages/cli/src/ui/themes/theme.ts` (497 lignes)

```typescript
type ThemeType = 'light' | 'dark' | 'ansi' | 'custom';

interface ColorsTheme {
  type: ThemeType,
  Background, Foreground,
  LightBlue, AccentBlue, AccentPurple, AccentCyan,
  AccentGreen, AccentYellow, AccentRed,
  DiffAdded, DiffRemoved, Comment, Gray, DarkGray,
  GradientColors?: string[],
}
```

- `Theme` class : mappe les classes highlight.js vers des couleurs Ink
- `SemanticColors` : tokens semantiques (text, background, border, ui, status)
- `createCustomTheme()` : cree un theme custom depuis `CustomTheme` config
- `pickDefaultThemeName()` : auto-detection light/dark selon la couleur de fond du terminal

### 10.2 ThemeManager

**Fichier** : `packages/cli/src/ui/themes/theme-manager.ts` (464 lignes)

`ThemeManager` (singleton exporte comme `themeManager`) :

#### 14 themes built-in :
1. Ayu Dark
2. Ayu Light
3. Atom One Dark
4. Dracula
5. Default Light
6. **Default Dark** (defaut)
7. GitHub Dark
8. GitHub Light
9. Google Code
10. Holiday
11. Shades of Purple
12. XCode
13. ANSI
14. ANSI Light

+ `NoColorTheme` (si `NO_COLOR` env var)

#### Sources de themes custom :
- `settingsThemes` : depuis settings.json
- `extensionThemes` : depuis extensions (namespace: `"theme_name (extension_name)"`)
- `fileThemes` : depuis fichiers JSON (securite : doit etre dans `$HOME`)

#### Tri d'affichage : Dark > Light > ANSI > Custom

---

## 11. Systeme de Settings

### 11.1 5 Niveaux de Precedence

**Fichier** : `packages/cli/src/config/settings.ts`

```typescript
enum SettingScope {
  User = 'User',           // ~/.gemini/settings.json
  Workspace = 'Workspace', // .gemini/settings.json
  System = 'System',       // /etc/gemini-cli/settings.json (Linux)
  SystemDefaults = 'SystemDefaults', // /etc/gemini-cli/system-defaults.json
  Session = 'Session',     // extensions seulement
}
```

#### Ordre de merge (last wins) :
1. **Schema Defaults** (built-in dans le code)
2. **System Defaults** (`/etc/gemini-cli/system-defaults.json`)
3. **User Settings** (`~/.gemini/settings.json`)
4. **Workspace Settings** (`.gemini/settings.json` - si trusted)
5. **System Settings** (`/etc/gemini-cli/settings.json` - overrides admin)

**Exception** : les settings `admin` ignorent les fichiers et ne viennent que du remote admin + defaults.

#### Chemins par OS :
- **Linux** : `/etc/gemini-cli/settings.json`
- **macOS** : `/Library/Application Support/GeminiCli/settings.json`
- **Windows** : `C:\ProgramData\gemini-cli\settings.json`

### 11.2 Schema des Settings

**Fichier** : `packages/cli/src/config/settingsSchema.ts`

```typescript
enum MergeStrategy {
  REPLACE = 'replace',       // defaut
  CONCAT = 'concat',         // concatene les arrays
  UNION = 'union',           // merge arrays unique
  SHALLOW_MERGE = 'shallow_merge', // merge objets (1 niveau)
}
```

#### Categories principales de settings :
- `mcpServers` : config MCP (shallow_merge)
- `general` : preferredEditor, vimMode, devtools, enableAutoUpdate, checkpointing, enablePromptCompletion, sessionRetention
- `model` : configuration du modele LLM
- `theme` : nom du theme
- `telemetry` : configuration telemetrie
- `sandbox` : configuration sandbox
- `admin` : settings admin (remote only)
- `accessibility` : enableLoadingPhrases, screenReader

### 11.3 LoadedSettings

`LoadedSettings` class :
- Charge les 4 fichiers de settings
- `mergeSettings()` : merge avec `customDeepMerge()` + `getMergeStrategyForPath()`
- `setTrusted()` : active/desactive le workspace settings dynamiquement
- `setRemoteAdminSettings()` : override admin depuis IPC
- `sanitizeEnvVar()` : protection injection shell dans les variables d'env

---

## 12. Context Management (GEMINI.md)

### 12.1 Hierarchie GEMINI.md

**Fichier** : `packages/core/src/tools/memoryTool.ts`

Nom configurable via `setGeminiMdFilename()` (peut etre un array de noms).

#### Emplacements :
1. **Global** : `~/.gemini/GEMINI.md` - preferences utilisateur cross-workspace
2. **Projet** : `.gemini/GEMINI.md` - instructions du projet
3. **Sous-repertoires** : potentiellement dans les sous-dossiers

#### Section automatique : `## Gemini Added Memories`
- Le tool `save_memory` ajoute des faits dans cette section
- Uniquement du contexte GLOBAL (jamais workspace-specific)

### 12.2 Context Files

Le fichier de contexte est equivalent au `CLAUDE.md` de Claude Code. Il est charge au demarrage dans le system prompt.

---

## 13. Slash Commands (40+)

### 13.1 Interface SlashCommand

**Fichier** : `packages/cli/src/ui/commands/types.ts` (229 lignes)

```typescript
interface SlashCommand {
  name: string,
  altNames?: string[],     // aliases
  description: string,
  hidden?: boolean,
  kind: CommandKind,        // BUILT_IN, FILE, MCP_PROMPT, AGENT
  autoExecute?: boolean,
  action: (context: CommandContext) => Promise<void>,
  completion?: (args: string, context: CommandContext) => Promise<string[]>,
  subCommands?: Map<string, SubCommand>,
}
```

#### CommandKind :
- `BUILT_IN` : commandes du CLI
- `FILE` : commandes depuis fichiers
- `MCP_PROMPT` : prompts MCP
- `AGENT` : commandes d'agents

### 13.2 Liste des Commandes

| Commande | Description |
|----------|-------------|
| `/about` | Info sur le CLI |
| `/agents` | Gestion des agents |
| `/auth` | Gestion de l'authentification |
| `/bug` | Rapporter un bug |
| `/chat` | Gestion des conversations |
| `/clear` | Effacer l'ecran |
| `/compress` | Compresser le contexte |
| `/copy` | Copier dans le presse-papier |
| `/corgi` | Easter egg corgi |
| `/directory` | Changer de repertoire |
| `/docs` | Documentation |
| `/editor` | Ouvrir dans l'editeur |
| `/extensions` | Gestion des extensions |
| `/help` | Aide |
| `/hooks` | Gestion des hooks |
| `/ide` | Integration IDE |
| `/init` | Initialiser `.gemini/` |
| `/mcp` | Gestion MCP |
| `/memory` | Gestion de la memoire |
| `/model` | Changer de modele |
| `/oncall` | Debug oncall |
| `/permissions` | Gestion permissions |
| `/plan` | Mode planification |
| `/policies` | Gestion des policies |
| `/privacy` | Parametres de confidentialite |
| `/profile` | Profil utilisateur |
| `/quit` | Quitter |
| `/restore` | Restaurer une session |
| `/resume` | Reprendre une session |
| `/rewind` | Revenir en arriere dans la conversation |
| `/settings` | Parametres |
| `/setupGithub` | Config GitHub |
| `/shells` | Gestion des shells |
| `/shortcuts` | Raccourcis clavier |
| `/skills` | Gestion des skills |
| `/stats` | Statistiques de session |
| `/terminalSetup` | Config terminal |
| `/theme` | Changer de theme |
| `/tools` | Lister les outils |
| `/vim` | Mode Vim |

### 13.3 Commandes @ (At-Commands)

Detection via `isAtCommand()` dans `commandUtils.ts`.

Les at-commands (`@agent`, `@file`) sont des raccourcis pour invoquer des agents ou inclure des fichiers dans le contexte.

---

## 14. Session & Checkpointing

### 14.1 Resume de Session

- Flag `--resume` dans le CLI
- `SessionRetentionSettings` dans settings : `enabled`, `maxAge` (ex: "30d"), `maxCount`, `minRetention` (defaut "1d")
- Commandes : `/restore`, `/resume`, `/rewind`

### 14.2 Checkpointing

- Setting : `general.checkpointing.enabled` (defaut: false)
- Permet la recuperation de sessions apres crash
- Necessite restart pour activer

---

## 15. MessageBus & Confirmation Flow

### 15.1 Architecture

Le `MessageBus` est un systeme pub/sub avec correlation IDs pour les confirmations de tools :

1. Tool demande confirmation via `shouldConfirmExecute()`
2. `getMessageBusDecision()` publie sur le bus avec un `correlationId`
3. Le bus route vers le PolicyEngine
4. Reponse : ALLOW, DENY, ou ASK_USER
5. Si ASK_USER : UI affiche le prompt de confirmation
6. Timeout : 30 secondes

### 15.2 Flow complet d'execution d'un tool :

```
LLM -> tool call
  -> DeclarativeTool.build(params) -> validation
    -> ToolInvocation.shouldConfirmExecute()
      -> MessageBus -> PolicyEngine
        -> ALLOW : execute()
        -> DENY : retourne erreur
        -> ASK_USER : UI confirmation
          -> User approve : execute()
          -> User modify : editor -> execute(modified)
          -> User cancel : retourne erreur
```

---

## 16. Telemetrie

### 16.1 Events

- `FileOperationEvent` : operations fichier (CREATE, UPDATE)
- `ExtensionInstallEvent`, `ExtensionUninstallEvent`, etc.
- `WebFetchFallbackAttemptEvent`
- `logFileOperation()` dans `packages/core/src/telemetry/loggers.ts`

### 16.2 Metriques

- `FileOperation` enum : CREATE, UPDATE
- Collecte : mimetype, extension, langage, nombre de lignes

---

## 17. IDE Integration

### 17.1 IdeClient

**Reference** : `packages/core/src/ide/ide-client.ts`

- `IdeClient.getInstance()` : singleton
- `isDiffingEnabled()` : verifie si l'IDE supporte les diffs
- `openDiff(path, content)` : ouvre une vue diff dans VS Code
- Utilise par EditTool et WriteFileTool pour la review

### 17.2 VSCode Companion

**Package** : `packages/vscode-ide-companion/`

Extension VSCode qui :
- Recoit les diffs depuis le CLI
- Affiche les differences inline
- Permet l'approbation/rejet depuis VS Code

---

## 18. Securite

### 18.1 Couches de securite

1. **PolicyEngine** : regles ALLOW/DENY/ASK_USER par outil
2. **Sandbox** : Docker/Podman/Seatbelt isolation
3. **Path validation** : `config.validatePathAccess()` sur chaque operation fichier
4. **Trust system** : workspace trust pour settings/agents/skills
5. **Agent acknowledgment** : hash-based pour les agents projet
6. **Redirection downgrade** : shell avec `>`, `>>`, `|` force ASK_USER
7. **Private IP protection** : `isPrivateIp()` dans WebFetch
8. **Theme file security** : themes JSON uniquement depuis `$HOME`
9. **Env var sanitization** : `sanitizeEnvVar()` regex whitelist
10. **OAuth** : PKCE, Dynamic Client Registration, token storage securise

### 18.2 Modeles de confiance

- **Non-interactif** : ASK_USER -> DENY automatique
- **Projet non-trusted** : workspace settings ignores, agents necessitent acknowledgment
- **Extensions** : consentement requis avant chargement

---

## 19. Differences Architecturales avec Claude Code

| Aspect | Gemini CLI | Claude Code |
|--------|-----------|-------------|
| **Langage** | TypeScript (source ouverte) | Binary ELF (Node.js compile) |
| **UI** | React/Ink TUI | Custom terminal |
| **Tools** | 14+ built-in + MCP + discovered | 15+ built-in |
| **Policy** | TOML avec 3 tiers | Built-in avec hooks |
| **Agents** | Markdown frontmatter (local/remote) | Task-based subagents |
| **Extensions** | Systeme complet (install/themes/hooks) | MCP servers uniquement |
| **Sandbox** | Docker/Podman/Seatbelt | Pas de sandbox natif |
| **Context** | GEMINI.md | CLAUDE.md |
| **Memory** | save_memory tool + GEMINI.md | Auto-memory ~/.claude/ |
| **Edit** | 3 strategies + LLM correction | Exact string replacement |
| **Skills** | SKILL.md avec precedence | Slash commands |

---

## 20. Fichiers Cles Reference

| Fichier | Lignes | Role |
|---------|--------|------|
| `cli/src/gemini.tsx` | ~850 | Entry point principal |
| `cli/src/nonInteractiveCli.ts` | 534 | Mode non-interactif |
| `core/src/tools/tools.ts` | 828 | Abstractions tools |
| `core/src/tools/tool-registry.ts` | 602 | Registre des tools |
| `core/src/tools/tool-names.ts` | 163 | Noms et aliases |
| `core/src/tools/shell.ts` | 524 | Shell tool |
| `core/src/tools/edit.ts` | 1057 | Edit tool (3 strategies) |
| `core/src/tools/write-file.ts` | 547 | WriteFile + LLM correction |
| `core/src/tools/web-fetch.ts` | ~300 | WebFetch tool |
| `core/src/tools/web-search.ts` | ~150 | WebSearch (Grounding) |
| `core/src/tools/glob.ts` | ~200 | Glob tool |
| `core/src/tools/grep.ts` | ~350 | Grep tool |
| `core/src/tools/memoryTool.ts` | ~200 | Memory (GEMINI.md) |
| `core/src/agents/types.ts` | 207 | Types d'agents |
| `core/src/agents/agentLoader.ts` | 380 | Chargement agents .md |
| `core/src/agents/registry.ts` | 484 | Registre agents |
| `core/src/agents/subagent-tool-wrapper.ts` | 91 | Wrapper agent->tool |
| `core/src/agents/subagent-tool.ts` | ~130 | SubagentTool v2 |
| `core/src/agents/local-invocation.ts` | 144 | Execution locale agent |
| `core/src/policy/policy-engine.ts` | 519 | Moteur de policy |
| `core/src/policy/types.ts` | 287 | Types de policy |
| `core/src/policy/toml-loader.ts` | 466 | Loader TOML policy |
| `core/src/hooks/hookSystem.ts` | 428 | Systeme de hooks |
| `core/src/hooks/types.ts` | 668 | Types de hooks |
| `core/src/skills/skillManager.ts` | 199 | Gestionnaire de skills |
| `core/src/skills/skillLoader.ts` | 191 | Chargeur SKILL.md |
| `core/src/mcp/auth-provider.ts` | 19 | Interface auth MCP |
| `core/src/mcp/oauth-provider.ts` | ~400 | OAuth complet MCP |
| `cli/src/utils/sandbox.ts` | 860 | Sandbox Docker/Seatbelt |
| `cli/src/config/settings.ts` | ~500 | 5 niveaux settings |
| `cli/src/config/settingsSchema.ts` | ~600 | Schema des settings |
| `cli/src/config/extension-manager.ts` | ~600 | Gestionnaire extensions |
| `cli/src/ui/themes/theme.ts` | 497 | Theme class |
| `cli/src/ui/themes/theme-manager.ts` | 464 | ThemeManager singleton |
| `cli/src/ui/commands/types.ts` | 229 | SlashCommand interface |

Tous les chemins relatifs sont depuis `/home/pedro/PROJETS-AI/gemini-cli/packages/`.
