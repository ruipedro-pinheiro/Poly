# Claude Code - Documentation Technique Complete

> Recherche exhaustive de l'architecture, fonctionnalites et mecaniques internes de Claude Code (Anthropic).
> Derniere mise a jour : Fevrier 2026

---

## Table des matieres

1. [Architecture](#1-architecture)
2. [Agent Teams](#2-agent-teams)
3. [Sub-agents (Task tool)](#3-sub-agents-task-tool)
4. [Skills System](#4-skills-system)
5. [Hooks System](#5-hooks-system)
6. [MCP Integration](#6-mcp-integration)
7. [Permission Modes](#7-permission-modes)
8. [Tools](#8-tools)
9. [Context Management](#9-context-management)
10. [Git Integration](#10-git-integration)
11. [Model Selection](#11-model-selection)

---

## 1. Architecture

### Vue d'ensemble

Claude Code est un outil de codage agentique qui vit dans le terminal, comprend le codebase, edite des fichiers, execute des commandes et s'integre avec les outils de developpement. Il est disponible dans le terminal, les IDE (VS Code, JetBrains), le navigateur et comme application desktop.

### Boucle Agentique (Agentic Loop)

L'architecture de Claude Code repose sur une boucle agentique minimaliste mais puissante :

```
while(tool_call) -> execute tool -> feed results -> repeat
```

La boucle continue tant que la reponse du modele inclut des appels d'outils. Quand Claude produit une reponse en texte brut sans appel d'outil, la boucle se termine naturellement.

**Architecture interne :**
- Boucle maitre single-threaded (codenamed "nO")
- File de pilotage en temps reel ("h2A queue")
- Kit d'outils developeur (Read, Write, Edit, Bash, etc.)
- Planification intelligente via TODO lists
- Spawn controle de sub-agents
- Mesures de securite (memory management, diff-based workflows)

### Communication avec l'API Anthropic

- Claude Code envoie des requetes a l'API Claude via le protocole standard HTTP
- Context window de 200K tokens (certains modeles/plans peuvent atteindre 1M tokens)
- Le buffer de contexte est d'environ 33,000 tokens (~16.5% reserve)
- Token compression peut reduire les couts API de 30-50%

### Process Model

- **CLI locale** : installe via `curl`, Homebrew, WinGet ou npm
- **Process principal** : boucle agentique qui gere les interactions utilisateur
- **Sub-processes** : agents specialises spawnes pour des taches specifiques
- **Background tasks** : agents pouvant tourner en arriere-plan pendant que l'utilisateur continue

### Surfaces disponibles

| Surface | Description |
|---------|-------------|
| **Terminal** | CLI complete pour le terminal |
| **VS Code** | Extension avec diffs inline, @-mentions, plan review |
| **JetBrains** | Plugin pour IntelliJ, PyCharm, WebStorm |
| **Desktop** | Application standalone |
| **Web** | Sessions cloud sans setup local |
| **iOS** | Via l'app Claude |

---

## 2. Agent Teams

### Vue d'ensemble

Les Agent Teams sont une fonctionnalite experimentale permettant de coordonner plusieurs instances Claude Code. Un "team lead" orchestre le travail, assigne des taches, et synthetise les resultats. Les teammates travaillent independamment, chacun dans sa propre fenetre de contexte.

**Activation requise :**
```json
{
  "env": {
    "CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS": "1"
  }
}
```

### Architecture d'une equipe

| Composant | Role |
|-----------|------|
| **Team Lead** | Session principale qui cree l'equipe, spawne les teammates, coordonne |
| **Teammates** | Instances Claude Code separees travaillant sur des taches assignees |
| **Task List** | Liste partagee de taches que les teammates claim et completent |
| **Mailbox** | Systeme de messagerie inter-agents |

**Stockage local :**
- Config equipe : `~/.claude/teams/{team-name}/config.json`
- Liste taches : `~/.claude/tasks/{team-name}/`

### Tools specifiques aux equipes

| Tool | Description | Parametres cles |
|------|-------------|----------------|
| **TeamCreate** | Cree une nouvelle equipe | Nom, configuration, spawn de teammates |
| **TeamDelete** | Supprime une equipe et ses ressources | Nom de l'equipe |
| **SendMessage** | Envoie un message entre agents | `type` (message/broadcast/shutdown_request/shutdown_response/plan_approval_response), `recipient`, `content`, `summary` |
| **TaskCreate** | Cree une tache dans la liste partagee | `subject`, `description`, `activeForm` |
| **TaskUpdate** | Met a jour le statut d'une tache | `taskId`, `status` (pending/in_progress/completed/deleted), `addBlocks`, `addBlockedBy` |
| **TaskList** | Liste toutes les taches | - |
| **TaskGet** | Recupere les details d'une tache | `taskId` |

### Communication inter-agents

**Types de messages SendMessage :**

1. **message** : DM a un teammate specifique (recipient obligatoire)
2. **broadcast** : Message a TOUS les teammates (utiliser avec parcimonie - cout lineaire avec la taille de l'equipe)
3. **shutdown_request** : Demande de shutdown graceful a un teammate
4. **shutdown_response** : Reponse a un shutdown request (approve/reject avec request_id)
5. **plan_approval_response** : Approbation/rejet du plan d'un teammate en plan mode

### Team Lead vs Teammates

**Team Lead :**
- Session qui cree l'equipe, fixe pour toute la duree
- Coordonne, assigne des taches, synthetise les resultats
- Recoit automatiquement les messages et notifications idle des teammates
- Peut passer en "delegate mode" (Shift+Tab) pour se limiter a la coordination

**Teammates :**
- Instances independantes avec leur propre fenetre de contexte
- Chargent le meme contexte projet que les sessions normales (CLAUDE.md, MCP, skills)
- Recoivent le prompt de spawn du lead (pas l'historique de conversation)
- Ne peuvent PAS spawner leurs propres equipes (pas de nested teams)
- Peuvent self-claim les taches non assignees et non bloquees

### Modes d'affichage

| Mode | Description |
|------|-------------|
| **In-process** | Tous les teammates dans le terminal principal. Shift+Up/Down pour naviguer |
| **Split panes** | Chaque teammate dans son propre pane (necessite tmux ou iTerm2) |
| **auto** (defaut) | Utilise split panes si deja dans tmux, sinon in-process |

### Plan Approval pour teammates

- Le lead peut exiger l'approbation du plan avant implementation
- Le teammate travaille en plan mode (read-only) jusqu'a approbation
- Si rejete, le teammate revise et resoumet
- Le lead approuve/rejette de maniere autonome selon les criteres donnes

### Delegate Mode

- Restreint le lead aux outils de coordination uniquement
- Empeche le lead d'implementer lui-meme
- Active via Shift+Tab apres creation de l'equipe

### Cas d'utilisation optimaux

- **Research et review** : plusieurs perspectives simultanees
- **Nouveaux modules** : chaque teammate owns un morceau
- **Debugging** : hypotheses concurrentes en parallele
- **Cross-layer** : frontend/backend/tests chacun par un teammate

### Limitations

- Pas de resumption de session avec in-process teammates
- Statut des taches peut trainer (teammates oublient parfois de marquer completed)
- Shutdown peut etre lent
- Une seule equipe par session
- Pas de nested teams
- Lead fixe (pas de promotion de teammate)
- Permissions fixees au spawn (changeable individuellement apres)

---

## 3. Sub-agents (Task tool)

### Vue d'ensemble

Les sub-agents sont des assistants IA specialises executant des taches specifiques. Chacun tourne dans sa propre fenetre de contexte avec un system prompt personnalise, des outils specifiques et des permissions independantes.

### Sub-agents built-in

| Agent | Model | Tools | Usage |
|-------|-------|-------|-------|
| **Explore** | Haiku (rapide) | Read-only (pas de Write/Edit) | Recherche codebase, decouverte fichiers |
| **Plan** | Herite du parent | Read-only (pas de Write/Edit) | Recherche pour planning en plan mode |
| **General-purpose** | Herite du parent | Tous les outils | Taches complexes multi-etapes |
| **Bash** | Herite du parent | Terminal commands | Commandes dans un contexte separe |
| **statusline-setup** | Sonnet | - | Configuration de la status line via /statusline |
| **Claude Code Guide** | Haiku | - | Questions sur les fonctionnalites Claude Code |

### Niveaux de thoroughness (Explore)

Quand Claude invoque Explore, il specifie un niveau :
- **quick** : recherches ciblees
- **medium** : exploration equilibree
- **very thorough** : analyse comprehensive

### Task Tool - Parametres

| Parametre | Description |
|-----------|-------------|
| `prompt` | La tache a effectuer |
| `description` | Description courte de la tache |
| `subagent_type` | Type d'agent : "Explore", "Plan", "general-purpose", ou agent custom |
| `model` | Alias de modele optionnel (sonnet, opus, haiku) |
| `run_in_background` | Executer en arriere-plan (concurrent) |

### Foreground vs Background

**Foreground (blocant) :**
- Bloque la conversation principale jusqu'a completion
- Prompts de permission passes a l'utilisateur
- Questions de clarification (AskUserQuestion) transmises

**Background (concurrent) :**
- Tourne en parallele pendant que l'utilisateur continue
- Permissions pre-approuvees au lancement
- Auto-deny pour tout ce qui n'est pas pre-approuve
- Outils MCP non disponibles
- Si questions de clarification : le tool call echoue mais le subagent continue
- Ctrl+B pour passer une tache en background

### Custom Sub-agents

Definis dans des fichiers Markdown avec YAML frontmatter :

**Emplacements :**

| Emplacement | Scope | Priorite |
|-------------|-------|----------|
| `--agents` CLI flag | Session courante | 1 (plus haute) |
| `.claude/agents/` | Projet courant | 2 |
| `~/.claude/agents/` | Tous les projets | 3 |
| Plugin `agents/` | Ou le plugin est active | 4 |

**Champs frontmatter supportes :**

| Champ | Requis | Description |
|-------|--------|-------------|
| `name` | Oui | Identifiant unique (lettres minuscules + tirets) |
| `description` | Oui | Quand Claude devrait deleguer a ce subagent |
| `tools` | Non | Tools autorises (herite tous si omis) |
| `disallowedTools` | Non | Tools interdits |
| `model` | Non | sonnet, opus, haiku, ou inherit (defaut) |
| `permissionMode` | Non | default, acceptEdits, delegate, dontAsk, bypassPermissions, plan |
| `maxTurns` | Non | Nombre max de tours agentiques |
| `skills` | Non | Skills a charger au demarrage |
| `mcpServers` | Non | Serveurs MCP disponibles |
| `hooks` | Non | Hooks de cycle de vie |
| `memory` | Non | Scope de memoire persistante : user, project, local |

### Memoire persistante des sub-agents

Quand `memory` est configure :
- L'agent a un repertoire persistant entre conversations
- System prompt inclut les 200 premieres lignes de `MEMORY.md`
- Read, Write, Edit auto-actives pour gerer la memoire

| Scope | Emplacement |
|-------|------------|
| `user` | `~/.claude/agent-memory/<name>/` |
| `project` | `.claude/agent-memory/<name>/` |
| `local` | `.claude/agent-memory-local/<name>/` |

### Resume de sub-agents

- Chaque invocation cree une instance fresh par defaut
- Possible de reprendre un subagent existant (conserve tout l'historique)
- Transcripts stockes dans `~/.claude/projects/{project}/{sessionId}/subagents/agent-{agentId}.jsonl`
- Auto-compaction a ~95% de capacite

### Regles importantes

- Les sub-agents NE PEUVENT PAS spawner d'autres sub-agents
- Jusqu'a 7 agents peuvent tourner simultanement
- Le sub-agent ne recoit PAS le system prompt complet de Claude Code (seulement son propre prompt + infos d'environnement)

---

## 4. Skills System

### Vue d'ensemble

Les Skills etendent les capacites de Claude Code. Chaque skill est defini dans un fichier `SKILL.md` avec du frontmatter YAML et des instructions Markdown.

**Depuis la version 2.1.3** : les slash commands et skills ont ete fusionnes. Un fichier dans `.claude/commands/review.md` et un skill dans `.claude/skills/review/SKILL.md` creent tous les deux `/review`.

### Structure d'un skill

```
my-skill/
  SKILL.md           # Instructions principales (requis)
  template.md        # Template pour Claude
  examples/
    sample.md        # Exemple de sortie
  scripts/
    validate.sh      # Script executable
```

### Format SKILL.md

```yaml
---
name: my-skill
description: Description du skill
disable-model-invocation: true
allowed-tools: Read, Grep, Glob
context: fork
agent: Explore
model: sonnet
argument-hint: [issue-number]
user-invocable: true
hooks:
  PreToolUse:
    - matcher: "Bash"
      hooks:
        - type: command
          command: "./scripts/check.sh"
---

Instructions en Markdown...
```

### Champs frontmatter

| Champ | Requis | Description |
|-------|--------|-------------|
| `name` | Non | Nom d'affichage (defaut: nom du dossier). Minuscules, chiffres, tirets (max 64 chars) |
| `description` | Recommande | Quand utiliser le skill. Claude utilise ca pour decider quand l'appliquer |
| `argument-hint` | Non | Hint pour l'autocompletion (ex: `[issue-number]`) |
| `disable-model-invocation` | Non | `true` empeche Claude de charger automatiquement. Defaut: `false` |
| `user-invocable` | Non | `false` cache du menu `/`. Defaut: `true` |
| `allowed-tools` | Non | Tools sans approbation quand le skill est actif |
| `model` | Non | Modele a utiliser |
| `context` | Non | `fork` pour executer dans un subagent isole |
| `agent` | Non | Type de subagent quand `context: fork` (Explore, Plan, general-purpose, custom) |
| `hooks` | Non | Hooks scoped au cycle de vie du skill |

### Emplacements des skills

| Emplacement | Chemin | Portee |
|-------------|--------|--------|
| Enterprise | Managed settings | Toute l'organisation |
| Personnel | `~/.claude/skills/<name>/SKILL.md` | Tous vos projets |
| Projet | `.claude/skills/<name>/SKILL.md` | Ce projet uniquement |
| Plugin | `<plugin>/skills/<name>/SKILL.md` | Ou le plugin est active |

Priorite : enterprise > personal > project. Les plugins utilisent un namespace `plugin-name:skill-name`.

### Controle d'invocation

| Frontmatter | Vous pouvez invoquer | Claude peut invoquer |
|-------------|---------------------|---------------------|
| (defaut) | Oui | Oui |
| `disable-model-invocation: true` | Oui | Non |
| `user-invocable: false` | Non | Oui |

### Substitutions de variables

| Variable | Description |
|----------|-------------|
| `$ARGUMENTS` | Tous les arguments passes |
| `$ARGUMENTS[N]` | Argument par index (0-based) |
| `$N` | Raccourci pour `$ARGUMENTS[N]` |
| `${CLAUDE_SESSION_ID}` | ID de session courante |

### Injection dynamique de contexte

La syntaxe `` !`command` `` execute des commandes shell AVANT que le contenu du skill soit envoye a Claude. La sortie remplace le placeholder.

### Execution dans un sub-agent (context: fork)

- `context: fork` execute le skill dans un contexte isole
- L'agent ne recoit PAS l'historique de conversation
- `agent` specifie le type d'agent (Explore, Plan, general-purpose, custom)
- Resultats resumes et retournes a la conversation principale

### Skill Tool

Le Skill tool est l'outil interne que Claude utilise pour invoquer des skills programmatiquement. Il est reference dans le system prompt comme un outil disponible.

---

## 5. Hooks System

### Vue d'ensemble

Les hooks sont des commandes shell, prompts LLM ou agents qui s'executent automatiquement a des points specifiques du cycle de vie de Claude Code.

### Types de hooks

| Type | Description |
|------|-------------|
| **command** (`type: "command"`) | Commande shell. Recoit JSON sur stdin, communique via exit codes et stdout |
| **prompt** (`type: "prompt"`) | Prompt envoye a un modele Claude pour evaluation single-turn (oui/non) |
| **agent** (`type: "agent"`) | Sub-agent spawne avec acces aux outils (Read, Grep, Glob) pour verification multi-turn |

### Evenements de hook

| Evenement | Quand | Matcher | Peut bloquer |
|-----------|-------|---------|-------------|
| `SessionStart` | Debut/resume de session | startup, resume, clear, compact | Non |
| `UserPromptSubmit` | Soumission de prompt utilisateur | Pas de matcher | Oui |
| `PreToolUse` | Avant un appel d'outil | Nom de l'outil | Oui |
| `PermissionRequest` | Quand un dialog de permission apparait | Nom de l'outil | Oui |
| `PostToolUse` | Apres un appel d'outil reussi | Nom de l'outil | Non |
| `PostToolUseFailure` | Apres un echec d'outil | Nom de l'outil | Non |
| `Notification` | Quand une notification est envoyee | Type de notification | Non |
| `SubagentStart` | Quand un subagent est spawne | Type d'agent | Non |
| `SubagentStop` | Quand un subagent termine | Type d'agent | Oui |
| `Stop` | Quand Claude finit de repondre | Pas de matcher | Oui |
| `TeammateIdle` | Quand un teammate va devenir idle | Pas de matcher | Oui |
| `TaskCompleted` | Quand une tache est marquee complete | Pas de matcher | Oui |
| `PreCompact` | Avant la compaction de contexte | manual, auto | Non |
| `SessionEnd` | Fin de session | Raison de fin | Non |

### Exit codes

| Code | Comportement |
|------|-------------|
| **0** | Succes. stdout est parse pour du JSON. L'action continue |
| **2** | Erreur bloquante. stderr est renvoye a Claude comme message d'erreur. L'action est bloquee (si supportee) |
| **Autre** | Erreur non-bloquante. stderr affiche en mode verbose, l'execution continue |

### Input JSON commun (stdin)

Tous les hooks recoivent ces champs :
- `session_id` : identifiant de session
- `transcript_path` : chemin du JSON de conversation
- `cwd` : repertoire de travail courant
- `permission_mode` : mode de permission actif
- `hook_event_name` : nom de l'evenement

Plus des champs specifiques selon l'evenement (ex: `tool_name`, `tool_input` pour PreToolUse).

### Decision Control par evenement

**PreToolUse** utilise `hookSpecificOutput` :
```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "allow|deny|ask",
    "permissionDecisionReason": "Raison",
    "updatedInput": { "field": "new value" },
    "additionalContext": "Contexte supplementaire"
  }
}
```

**Stop, PostToolUse, UserPromptSubmit, SubagentStop** utilisent top-level `decision` :
```json
{
  "decision": "block",
  "reason": "Explication"
}
```

**TeammateIdle, TaskCompleted** utilisent exit code 2 + stderr uniquement.

**PermissionRequest** utilise `hookSpecificOutput.decision.behavior` :
```json
{
  "hookSpecificOutput": {
    "hookEventName": "PermissionRequest",
    "decision": {
      "behavior": "allow|deny",
      "updatedInput": {},
      "updatedPermissions": {},
      "message": "Pour deny"
    }
  }
}
```

### Configuration

**Emplacements :**

| Emplacement | Portee |
|-------------|--------|
| `~/.claude/settings.json` | Tous les projets |
| `.claude/settings.json` | Projet (committable) |
| `.claude/settings.local.json` | Projet (gitignore) |
| Managed policy | Organisation |
| Plugin `hooks/hooks.json` | Quand le plugin est active |
| Skill/Agent frontmatter | Pendant que le composant est actif |

### Hooks asynchrones

- `"async": true` sur les hooks de type command
- Tourne en arriere-plan sans bloquer Claude
- Ne peut PAS bloquer les actions
- Le resultat est delivre au prochain tour de conversation

### Hooks basees sur des prompts

- `type: "prompt"` envoie le prompt + input JSON a un modele Claude rapide
- Repond avec `{ "ok": true/false, "reason": "..." }`
- Si `ok: false`, l'action est bloquee
- Supportee pour : PreToolUse, PostToolUse, PostToolUseFailure, PermissionRequest, UserPromptSubmit, Stop, SubagentStop, TaskCompleted

### Hooks basees sur des agents

- `type: "agent"` spawne un sub-agent multi-turn avec acces a Read, Grep, Glob
- Jusqu'a 50 tours pour verifier des conditions
- Meme format de reponse que les prompt hooks
- Defaut timeout : 60 secondes

### Variables d'environnement disponibles

- `$CLAUDE_PROJECT_DIR` : racine du projet
- `${CLAUDE_PLUGIN_ROOT}` : racine du plugin
- `$CLAUDE_CODE_REMOTE` : "true" si environnement web distant
- `$CLAUDE_ENV_FILE` : fichier pour persister des variables d'env (SessionStart uniquement)

---

## 6. MCP Integration

### Vue d'ensemble

Claude Code se connecte a des outils et sources de donnees externes via le Model Context Protocol (MCP), un standard ouvert pour les integrations outil-IA.

### Configuration

Les serveurs MCP sont configures dans un fichier JSON :

```json
{
  "mcpServers": {
    "server-name": {
      "type": "stdio",
      "command": "node",
      "args": ["/path/to/server/index.js"]
    }
  }
}
```

**Emplacements de configuration :**
- `~/.claude.json` : Configuration globale
- `.claude/settings.json` : Configuration projet
- Via `claude mcp add` : Assistant CLI

### Tool Discovery

- Claude Code decouvre automatiquement les outils disponibles sur chaque serveur MCP
- Le Tool Search s'active automatiquement quand les descriptions d'outils MCP depasseraient 10% de la fenetre de contexte
- Support des notifications `list_changed` pour la mise a jour dynamique des outils

### Naming Pattern des outils MCP

```
mcp__<server>__<tool>
```

Exemples :
- `mcp__memory__create_entities`
- `mcp__filesystem__read_file`
- `mcp__github__search_repositories`

### ListMcpResourcesTool et ReadMcpResourceTool

- **ListMcpResourcesTool** : Liste les ressources disponibles sur un serveur MCP
- **ReadMcpResourceTool** : Lit le contenu d'une ressource specifique

**Limitation connue** : Ces outils echouent parfois avec les serveurs HTTP MCP.

### MCP dans les sub-agents

- Les outils MCP sont herites par les sub-agents par defaut
- Les sub-agents en arriere-plan n'ont PAS acces aux outils MCP
- Les sub-agents custom peuvent specifier quels serveurs MCP utiliser via le champ `mcpServers`

### Permissions MCP

- Les outils MCP suivent le pattern `mcp__<server>__<tool>` dans les regles de permission
- `mcp__puppeteer` : match tous les outils du serveur puppeteer
- `mcp__puppeteer__puppeteer_navigate` : match un outil specifique
- Les hooks PreToolUse/PostToolUse fonctionnent avec les outils MCP

---

## 7. Permission Modes

### Systeme de permission

Claude Code utilise un systeme de permissions a plusieurs niveaux :

| Type d'outil | Exemple | Approbation | Comportement "Ne plus demander" |
|-------------|---------|-------------|--------------------------------|
| Read-only | Lecture fichiers, Grep | Non requise | N/A |
| Bash commands | Execution shell | Oui | Permanent par projet et commande |
| File modification | Edit/Write | Oui | Jusqu'a fin de session |

### Modes de permission

| Mode | Description |
|------|-------------|
| `default` | Comportement standard : demande d'approbation a la premiere utilisation |
| `acceptEdits` | Auto-accepte les modifications de fichiers pour la session |
| `plan` | Mode Plan : Claude peut analyser mais PAS modifier les fichiers ou executer des commandes |
| `delegate` | Mode coordination pour team leads. Restreint aux outils de gestion d'equipe. Disponible uniquement avec une equipe active |
| `dontAsk` | Auto-deny les outils sauf ceux pre-approuves via `/permissions` ou `permissions.allow` |
| `bypassPermissions` | Saute TOUS les checks de permission (uniquement dans des environnements isoles) |

### Changement de mode

- **Shift+Tab** : cycle entre les modes pendant une session
- **`defaultMode` dans settings.json** : mode par defaut
- **`--dangerously-skip-permissions`** : active bypassPermissions depuis la CLI

### EnterPlanMode / ExitPlanMode

- **Plan Mode** : Claude peut lire et analyser mais ne peut PAS ecrire, editer ou executer des commandes
- **ExitPlanMode** : outil interne que Claude utilise pour proposer la sortie du plan mode
- En agent teams, les teammates en plan mode doivent soumettre leur plan au lead pour approbation

### Syntaxe des regles de permission

**Format general** : `Tool` ou `Tool(specifier)`

**Exemples :**

| Regle | Effet |
|-------|-------|
| `Bash` | Match toutes les commandes Bash |
| `Bash(npm run build)` | Match la commande exacte |
| `Bash(npm run *)` | Wildcard glob pattern |
| `Read(./.env)` | Match la lecture d'un fichier specifique |
| `WebFetch(domain:example.com)` | Match les fetches vers un domaine |
| `Task(Explore)` | Match le subagent Explore |
| `Edit(/src/**/*.ts)` | Match les editions dans un pattern gitignore |

**Ordre d'evaluation** : deny -> ask -> allow (deny toujours prioritaire)

### Regles Read/Edit (gitignore)

| Pattern | Sens |
|---------|------|
| `//path` | Chemin absolu depuis la racine filesystem |
| `~/path` | Chemin depuis le home directory |
| `/path` | Chemin relatif au fichier settings |
| `path` ou `./path` | Chemin relatif au repertoire courant |

### Managed Settings (entreprise)

Fichiers deployes par les administrateurs IT :
- **macOS** : `/Library/Application Support/ClaudeCode/managed-settings.json`
- **Linux** : `/etc/claude-code/managed-settings.json`
- **Windows** : `C:\Program Files\ClaudeCode\managed-settings.json`

**Settings managed-only :**
- `disableBypassPermissionsMode` : empeche le mode bypassPermissions
- `allowManagedPermissionRulesOnly` : seules les regles managed s'appliquent
- `allowManagedHooksOnly` : seuls les hooks managed et SDK sont autorises
- `strictKnownMarketplaces` : controle les marketplaces de plugins

### Precedence des settings

Managed > CLI arguments > Local project > Shared project > User settings

---

## 8. Tools

### Liste complete des outils internes

| Outil | Description | Parametres cles |
|-------|-------------|----------------|
| **Read** | Lit un fichier du filesystem | `file_path` (absolu), `offset`, `limit`, `pages` (PDF) |
| **Write** | Ecrit/ecrase un fichier | `file_path` (absolu), `content` |
| **Edit** | Remplacement exact de string dans un fichier | `file_path`, `old_string`, `new_string`, `replace_all` |
| **MultiEdit** | Plusieurs edits dans un seul appel | Fichier + liste d'edits |
| **NotebookEdit** | Edite une cellule Jupyter | `notebook_path`, `cell_id`, `new_source`, `cell_type`, `edit_mode` |
| **Glob** | Pattern matching rapide de fichiers | `pattern` (ex: `**/*.ts`), `path` |
| **Grep** | Recherche de contenu (ripgrep) | `pattern` (regex), `path`, `glob`, `output_mode`, `-i`, `multiline`, `head_limit` |
| **Bash** | Execute des commandes shell | `command`, `description`, `timeout`, `run_in_background` |
| **BashOutput** | Lit la sortie d'une commande Bash en background | Task ID |
| **KillBash** | Tue un process Bash | PID |
| **WebFetch** | Fetch et traite du contenu web | `url`, `prompt` |
| **WebSearch** | Recherche web | `query`, `allowed_domains`, `blocked_domains` |
| **Task** | Spawne un sub-agent | `prompt`, `description`, `subagent_type`, `model`, `run_in_background` |
| **TodoRead** | Lit la todo list | - |
| **TodoWrite** | Ecrit dans la todo list | Taches |
| **ExitPlanMode** | Propose la sortie du plan mode | - |
| **Skill** | Invoque un skill | `skill`, `args` |
| **AskUserQuestion** | Pose une question a l'utilisateur | Question |
| **LS** | Liste les fichiers d'un repertoire | Path |

### Outils specifiques aux Agent Teams

| Outil | Description |
|-------|-------------|
| **TeamCreate** | Cree une equipe |
| **TeamDelete** | Supprime une equipe |
| **SendMessage** | Messagerie inter-agents |
| **TaskCreate** | Cree une tache |
| **TaskUpdate** | Met a jour une tache |
| **TaskList** | Liste les taches |
| **TaskGet** | Details d'une tache |

### Outils MCP (dynamiques)

Les outils MCP sont decouverts dynamiquement selon les serveurs configures. Ils suivent le pattern `mcp__<server>__<tool>`.

### Regles d'utilisation des outils

**Priorites dans le system prompt :**
- Utiliser **Read** au lieu de `cat`, `head`, `tail`
- Utiliser **Edit** au lieu de `sed`, `awk`
- Utiliser **Write** au lieu de `echo >` ou heredoc
- Utiliser **Glob** au lieu de `find` ou `ls`
- Utiliser **Grep** au lieu de `grep` ou `rg`
- Utiliser **Bash** uniquement pour les commandes systeme et operations terminal

---

## 9. Context Management

### Fichiers CLAUDE.md

CLAUDE.md est un fichier Markdown ajoute a la racine du projet que Claude Code lit au debut de chaque session. Il contient :
- Standards de code
- Decisions architecturales
- Librairies preferees
- Checklists de review

**Hierarchie de chargement :**
- Fichiers CLAUDE.md dans les repertoires PARENTS du working directory : charges entierement au lancement
- Fichiers CLAUDE.md dans les repertoires ENFANTS : charges a la demande quand Claude lit des fichiers dans ces repertoires

### Memory Files

**Auto Memory** : repertoire persistant a `~/.claude/projects/{project-hash}/memory/`

- `MEMORY.md` : toujours charge dans le system prompt (200 premieres lignes)
- Fichiers separees par sujet (ex: `debugging.md`, `patterns.md`)
- Referencies depuis MEMORY.md

**Quoi sauvegarder :**
- Patterns stables confirmes sur plusieurs interactions
- Decisions architecturales, chemins importants, structure projet
- Preferences utilisateur
- Solutions aux problemes recurrents

**Quoi NE PAS sauvegarder :**
- Contexte specifique a une session
- Info potentiellement incomplete
- Ce qui duplique les instructions CLAUDE.md
- Conclusions speculatives

### Context Compression (Compaction)

Quand la fenetre de contexte se remplit, Claude Code compacte la conversation :

1. **Auto-compaction** : se declenche a ~95% de capacite (configurable via `CLAUDE_AUTOCOMPACT_PCT_OVERRIDE`)
2. **Manuel** : commande `/compact`
3. **Process** : l'historique est condense par un LLM en un resume compact
4. **Perte** : la compaction est lossy - des details sont perdus

**Buffer de contexte** : environ 33K tokens (16.5%) reserves, donnant ~12K tokens d'espace utilisable supplementaire par rapport aux versions precedentes.

### Sessions de conversation

- Stockees comme fichiers `.jsonl` dans `~/.claude/projects/{project}/{sessionId}/`
- Resume possible via `/resume` et `/rewind`
- Transcripts de sub-agents dans un sous-dossier `subagents/`
- Cleanup automatique base sur `cleanupPeriodDays` (defaut: 30 jours)

---

## 10. Git Integration

### Workflow de commit

1. Claude analyse les changements staged (git diff)
2. Claude redige un message de commit concis
3. Le commit inclut automatiquement :
   - Tag `Generated with Claude Code`
   - `Co-Authored-By: Claude <model> <noreply@anthropic.com>`

**Regles du system prompt :**
- JAMAIS mettre a jour la git config
- JAMAIS de commandes git destructives (push --force, reset --hard, etc.) sauf demande explicite
- JAMAIS skip les hooks (--no-verify, --no-gpg-sign)
- JAMAIS force push sur main/master
- TOUJOURS creer de NOUVEAUX commits plutot qu'amender (sauf demande explicite)
- Preferer ajouter des fichiers specifiques plutot que `git add -A`
- NE JAMAIS committer sauf demande explicite

### Creation de PR

- Utilise `gh` CLI pour toutes les taches GitHub
- Analyse tous les commits depuis la divergence du branch de base
- Titre court (<70 chars), description detaillee
- Format : Summary + Test plan + watermark Claude Code

### Branch Management

- Support des git worktrees pour checkout de branches multiples
- Chaque worktree a son propre working directory avec des fichiers isoles
- Partage le meme historique Git

### Skill /commit

Le skill `/commit` (built-in) :
1. Analyse le diff
2. Genere un message Conventional Commit
3. Execute le commit

---

## 11. Model Selection

### Modeles disponibles

| Modele | Model ID | Caracteristiques |
|--------|----------|-----------------|
| **Opus 4.6** | `claude-opus-4-6` | Le plus puissant et intelligent. Raisonnement complexe, architecture, debugging profond |
| **Sonnet 4.5** | `claude-sonnet-4-5-20250929` | Equilibre intelligence/vitesse. Usage quotidien recommande (defaut) |
| **Haiku 4.5** | `claude-haiku-4-5-20251001` | Le plus rapide et economique. Taches simples, reponses instantanees |

### Changement de modele

**Au lancement :**
```bash
claude --model claude-opus-4-6
```

**Pendant la session :**
```
/model opus
```

Le changement prend effet immediatement pour tous les prompts suivants.

### Fast Mode

- Toggle avec `/fast`
- Utilise le MEME modele (Opus 4.6) avec une sortie plus rapide
- NE CHANGE PAS de modele

### Mode Opusplan (hybride)

L'alias `opusplan` fournit une approche hybride automatique :
- **Plan mode** : utilise Opus pour le raisonnement complexe et les decisions d'architecture
- **Implementation** : switch automatiquement a Sonnet pour la generation de code

### Selection de modele par sub-agent

| Sub-agent | Modele par defaut |
|-----------|-------------------|
| Explore | Haiku |
| Plan | Herite du parent |
| General-purpose | Herite du parent |
| statusline-setup | Sonnet |
| Claude Code Guide | Haiku |
| Custom | Configurable via frontmatter (`model: sonnet|opus|haiku|inherit`) |

---

## Sources

- [Claude Code Overview - Documentation officielle](https://code.claude.com/docs/en/overview)
- [Agent Teams - Documentation officielle](https://code.claude.com/docs/en/agent-teams)
- [Sub-agents - Documentation officielle](https://code.claude.com/docs/en/sub-agents)
- [Skills - Documentation officielle](https://code.claude.com/docs/en/skills)
- [Hooks Reference - Documentation officielle](https://code.claude.com/docs/en/hooks)
- [Permissions - Documentation officielle](https://code.claude.com/docs/en/permissions)
- [MCP Integration - Documentation officielle](https://code.claude.com/docs/en/mcp)
- [Model Configuration - Documentation officielle](https://code.claude.com/docs/en/model-config)
- [Memory - Documentation officielle](https://code.claude.com/docs/en/memory)
- [GitHub - anthropics/claude-code](https://github.com/anthropics/claude-code)
- [Anthropic releases Opus 4.6 with new 'agent teams' - TechCrunch](https://techcrunch.com/2026/02/05/anthropic-releases-opus-4-6-with-new-agent-teams/)
- [From Tasks to Swarms: Agent Teams - alexop.dev](https://alexop.dev/posts/from-tasks-to-swarms-agent-teams-in-claude-code/)
- [Claude Code Behind-the-scenes - PromptLayer](https://blog.promptlayer.com/claude-code-behind-the-scenes-of-the-master-agent-loop/)
- [Tracing Claude Code's LLM Traffic - Medium](https://medium.com/@georgesung/tracing-claude-codes-llm-traffic-agentic-loop-sub-agents-tool-use-prompts-7796941806f5)
