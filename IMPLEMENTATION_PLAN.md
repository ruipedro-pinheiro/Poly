# POLY-GO IMPLEMENTATION PLAN

> Ce plan est destiné à un Claude Opus 4.6 FRAIS (aucun contexte préalable).
> Tu DOIS lire ce fichier en entier avant de commencer.
> Tu DOIS utiliser des Agent Teams (CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1 est activé).
> Tu DOIS utiliser le modèle opus pour les teammates (`model: "opus"`).
> Tu DOIS travailler dans `/home/pedro/PROJETS-AI/Poly-go/`.
> L'utilisateur (Pedro) veut de l'ACTION, pas du blabla.
> Langue de communication : FRANÇAIS.

---

## CONTEXTE DU PROJET

**Poly-go** est un TUI multi-AI collaboratif écrit en Go (Bubble Tea v2 + Lip Gloss v2, thème Catppuccin Mocha).
Il permet de chatter avec 4+ providers LLM (Claude, GPT, Gemini, Grok + custom) avec tool use complet (boucle agentique).

### État actuel (audité le 2026-02-10)
- **94 fichiers Go**, 15 tools, 4 providers + custom, OAuth PKCE, sessions JSON multi-session
- **TUI** : 47 fichiers, 7 dialogs, 20+ raccourcis, splash screen, sidebar, status bar
- **Feature parity** : ~30-35% vs Claude Code/Gemini CLI/Crush/OpenCode
- **Unique features** : @all cascade mode (cheapest-first + reviewers), interactive shell (`-i`), session forking, YOLO mode, anti-gaslighting system prompt

### Rapports d'audit détaillés (à lire si besoin)
- `/home/pedro/PROJETS-AI/Poly-go/research/poly-tui-audit.md` (TUI complet)
- `/home/pedro/PROJETS-AI/Poly-go/research/poly-providers-audit.md` (providers + auth)
- `/home/pedro/PROJETS-AI/Poly-go/COMPARISON-V2.md` (comparaison vs concurrents)

---

## CE QUI MANQUE (les vrais gaps)

### CRITIQUE
1. **Context compaction / auto-summarization** - Sans ça, les conversations longues font exploser le context window
2. **Markdown rendering** - Tout le texte est plain text, pas de bold/headers/code blocks/listes
3. **Syntax highlighting** - Les code blocks sont du texte monospace sans couleur

### HAUTE PRIORITÉ
4. **POLY.md loader** - Instructions projet (comme CLAUDE.md pour Claude Code)
5. **Custom providers agentic loop** - Les custom providers n'exécutent PAS les tools (1 seul tour)
6. **Safe/banned command lists** - Le shell ne filtre pas les commandes dangereuses

### MOYENNE PRIORITÉ
7. **Retry/backoff** - Aucun provider ne retry sur erreur 429/500
8. **Grok extended thinking** - Le param thinking n'est pas propagé

---

## ARCHITECTURE DU CODEBASE

```
/home/pedro/PROJETS-AI/Poly-go/
├── main.go                          # Entry point: config.Load -> tools.Init -> llm.LoadCustomProviders -> TUI/Shell
├── internal/
│   ├── config/config.go             # Config JSON (~/.poly/config.json), defaults + user merge
│   ├── llm/
│   │   ├── provider.go              # Provider interface, registry, image support tracking
│   │   ├── anthropic.go             # Claude provider (OAuth + API key, SSE streaming, agentic loop)
│   │   ├── gpt.go                   # GPT provider (OpenAI format, reasoning models)
│   │   ├── gemini.go                # Gemini provider (dual mode: API key / Code Assist OAuth)
│   │   ├── grok.go                  # Grok provider (OpenAI format)
│   │   ├── custom.go                # Custom providers (OpenAI/Anthropic/Google formats, PAS d'agentic loop!)
│   │   ├── system.go                # BuildSystemPrompt() - 6 sections dynamiques
│   │   ├── pricing.go               # CalculateCost() - table de prix par modèle
│   │   └── tools_format.go          # ConvertToolsForProvider() - Anthropic/OpenAI/Google conversion
│   ├── tools/
│   │   ├── types.go                 # Tool interface, ToolCall, ToolResult, ToolDefinition
│   │   ├── registry.go              # Register/Get/GetAll/Execute/Init (15 tools)
│   │   ├── approval.go              # PendingChan/ApprovedChan, YoloMode, auto-allow list
│   │   ├── fs.go                    # ReadFileTool, ListFilesTool (avec security check cwd)
│   │   ├── search.go                # GlobTool, GrepTool (regex, file filter)
│   │   ├── bash.go                  # BashTool (os/exec)
│   │   ├── edit.go                  # EditFileTool (exact string replacement)
│   │   ├── write.go                 # WriteFileTool
│   │   ├── multiedit.go             # MultieditTool (batch edits)
│   │   ├── web.go                   # WebFetchTool, WebSearchTool (DuckDuckGo)
│   │   ├── todos.go                 # TodosTool (~/.poly/todos.json)
│   │   ├── diff.go                  # ApplyDiffTool, RejectDiffTool, ListDiffsTool
│   │   └── propose_diff.go          # ProposeDiffTool (propose/review workflow)
│   ├── session/session.go           # JSON multi-session (switch/fork/delete/rename/auto-title)
│   ├── permission/permission.go     # Allow/Ask/Deny classification + IsReadOnly
│   ├── auth/                        # OAuth: anthropic.go, google.go, openai.go, pkce.go, storage.go
│   ├── mcp/                         # MCP client + manager (stdio, auto-register, namespacing)
│   ├── pubsub/                      # Event broker (broker.go, events.go)
│   ├── shell/                       # Interactive shell (-i flag, AI pipes, variables)
│   └── tui/
│       ├── model.go                 # Main Model struct (toutes les données d'état)
│       ├── tui.go                   # New() constructeur
│       ├── update.go                # Update() - event handling (keys, messages, streaming)
│       ├── views.go                 # View() - rendering principal (chat, input, components)
│       ├── streaming.go             # Streaming goroutines (single + cascade)
│       ├── commands.go              # Slash commands handler (/clear /model /think /yolo etc.)
│       ├── keys.go                  # Key bindings
│       ├── messages.go              # Message types (Message, ToolCallData, ContentBlock)
│       ├── diff_render.go           # Diff coloring (line-by-line)
│       ├── approval.go              # Approval dialog rendering
│       ├── sidebar.go               # Sidebar rendering (model info, todos, providers)
│       ├── modelpicker.go           # Model picker dialog
│       ├── palette.go               # Command palette dialog
│       ├── session_list.go          # Session list dialog
│       ├── dialogs.go               # Dialog manager (LIFO stack, partially wired)
│       ├── clipboard.go             # Clipboard integration (Wayland/X11/macOS/Windows)
│       ├── components/              # 31 component files (header, sidebar, status, editor, splash, messages, dialogs)
│       ├── styles/                  # Theme (Catppuccin Mocha), styles
│       ├── layout/                  # Layout interfaces (Sizeable, Focusable)
│       ├── list/                    # Virtualized list (implemented but NOT used yet)
│       └── core/                    # Icons
```

---

## PLAN D'IMPLÉMENTATION (4 TÂCHES PARALLÈLES)

### TÂCHE 1 : MARKDOWN RENDERING + SYNTAX HIGHLIGHTING
**Agent : `markdown-dev`**
**Priorité : CRITIQUE**
**Fichiers à modifier : `internal/tui/views.go`, `go.mod`**

#### Objectif
Quand l'assistant renvoie du markdown (headers, bold, code blocks, listes, liens), le TUI doit le renderer correctement au lieu d'afficher du texte brut.

#### Dépendances Go à ajouter
```
github.com/charmbracelet/glamour   # Markdown rendering pour terminal
github.com/alecthomas/chroma/v2    # Syntax highlighting (utilisé par glamour)
```

#### Implémentation

1. **Ajouter les dépendances** :
   ```bash
   cd /home/pedro/PROJETS-AI/Poly-go && go get github.com/charmbracelet/glamour github.com/alecthomas/chroma/v2
   ```

2. **Créer `internal/tui/markdown.go`** :
   ```go
   package tui

   import (
       "github.com/charmbracelet/glamour"
   )

   var mdRenderer *glamour.TermRenderer

   func initMarkdown(width int) {
       mdRenderer, _ = glamour.NewTermRenderer(
           glamour.WithAutoStyle(),        // auto dark/light
           glamour.WithWordWrap(width),     // wrap au viewport width
       )
   }

   func renderMarkdown(content string, width int) string {
       if mdRenderer == nil || width <= 0 {
           return content
       }
       rendered, err := mdRenderer.Render(content)
       if err != nil {
           return content // fallback to plain text
       }
       return rendered
   }
   ```

3. **Modifier `internal/tui/views.go`** :
   - Dans la fonction qui rend les messages assistant, remplacer l'affichage brut du `content` par `renderMarkdown(content, m.viewport.Width)`.
   - Le contenu assistant se trouve dans `renderMessage()` ou son équivalent - cherche où `msg.Content` est affiché pour les messages avec `role == "assistant"`.
   - NE PAS appliquer le markdown aux messages user (input), seulement aux réponses assistant.
   - NE PAS appliquer le markdown au contenu des tool results.

4. **Mettre à jour le width quand le terminal resize** :
   - Dans `Update()`, quand un `tea.WindowSizeMsg` arrive, appeler `initMarkdown(msg.Width)`.

5. **Initialiser au démarrage** :
   - Dans `New()` ou au premier `tea.WindowSizeMsg`, appeler `initMarkdown(width)`.

#### Tests manuels
```bash
cd /home/pedro/PROJETS-AI/Poly-go && go build -o poly . && ./poly
```
Puis taper : "Écris-moi un hello world en Go avec des explications" → le code devrait être coloré et les headers formatés.

---

### TÂCHE 2 : CONTEXT COMPACTION / AUTO-SUMMARIZATION
**Agent : `compaction-dev`**
**Priorité : CRITIQUE**
**Fichiers à créer/modifier : `internal/llm/compaction.go`, `internal/tui/update.go`, `internal/tui/commands.go`**

#### Objectif
Quand la conversation approche la limite du context window, résumer automatiquement les anciens messages pour libérer de la place.

#### Design

**Seuil de compaction** : 80% du context window (le context window dépend du provider, mais on peut utiliser 200K tokens comme défaut pour Anthropic, 128K pour GPT, 1M pour Gemini).

**Stratégie** : Résumer les N premiers messages en un seul message "summary", garder les derniers messages intacts.

#### Implémentation

1. **Créer `internal/llm/compaction.go`** :
   ```go
   package llm

   import (
       "context"
       "fmt"
       "strings"
   )

   const (
       DefaultContextWindow = 200000  // tokens
       CompactionThreshold  = 0.80    // 80%
       MinMessagesToKeep    = 6       // garder au moins les 6 derniers messages
       CharsPerToken        = 4       // approximation grossière
   )

   // EstimateTokens estimates token count from text (rough: ~4 chars per token)
   func EstimateTokens(messages []Message) int {
       total := 0
       for _, m := range messages {
           total += len(m.Content) / CharsPerToken
           for _, tc := range m.ToolCalls {
               total += 50 // overhead par tool call
           }
           if m.ToolResult != nil {
               total += len(m.ToolResult.Content) / CharsPerToken
           }
       }
       return total
   }

   // NeedsCompaction returns true if messages exceed threshold
   func NeedsCompaction(messages []Message, contextWindow int) bool {
       if contextWindow <= 0 {
           contextWindow = DefaultContextWindow
       }
       estimated := EstimateTokens(messages)
       threshold := int(float64(contextWindow) * CompactionThreshold)
       return estimated > threshold
   }

   // CompactMessages summarizes old messages, keeping recent ones intact.
   // Uses the given provider to generate the summary.
   func CompactMessages(ctx context.Context, provider Provider, messages []Message, keepLast int) ([]Message, error) {
       if keepLast < MinMessagesToKeep {
           keepLast = MinMessagesToKeep
       }
       if len(messages) <= keepLast {
           return messages, nil // nothing to compact
       }

       // Split: old (to summarize) + recent (to keep)
       oldMessages := messages[:len(messages)-keepLast]
       recentMessages := messages[len(messages)-keepLast:]

       // Build summary prompt
       var summaryContent strings.Builder
       summaryContent.WriteString("Summarize the following conversation concisely. ")
       summaryContent.WriteString("Preserve: key decisions, file paths, code changes, errors encountered, and user preferences. ")
       summaryContent.WriteString("Be factual and brief.\n\n")
       for _, m := range oldMessages {
           summaryContent.WriteString(fmt.Sprintf("[%s]: %s\n", m.Role, truncate(m.Content, 500)))
       }

       // Call provider for summary (no tools, just text)
       summaryMessages := []Message{
           {Role: "user", Content: summaryContent.String()},
       }

       events := provider.Stream(ctx, summaryMessages, nil)
       var summary strings.Builder
       for event := range events {
           if event.Type == "content" {
               summary.WriteString(event.Content)
           }
           if event.Error != nil {
               return messages, event.Error // fallback: return original
           }
       }

       // Build compacted conversation
       compacted := []Message{
           {
               Role:    "user",
               Content: "[Previous conversation summary]\n" + summary.String(),
           },
           {
               Role:    "assistant",
               Content: "Understood. I have the context from the previous conversation. Let's continue.",
           },
       }
       compacted = append(compacted, recentMessages...)

       return compacted, nil
   }

   func truncate(s string, maxLen int) string {
       if len(s) <= maxLen {
           return s
       }
       return s[:maxLen] + "..."
   }
   ```

2. **Ajouter `/compact` dans `internal/tui/commands.go`** :
   Dans le `switch cmd` du `handleCommand()`, ajouter :
   ```go
   case "/compact":
       m.status = "Compacting conversation..."
       // Trigger compaction via a tea.Cmd
       // (implementation: send a CompactMsg, handle in update.go)
   ```

3. **Ajouter le message type dans `internal/tui/messages.go`** :
   ```go
   type CompactMsg struct{}
   type CompactDoneMsg struct {
       Messages []Message
       Error    error
   }
   ```

4. **Gérer dans `internal/tui/update.go`** :
   - Sur `CompactMsg` : lancer une goroutine qui appelle `llm.CompactMessages()` avec le provider actuel
   - Sur `CompactDoneMsg` : remplacer `m.messages` par les messages compactés, mettre à jour le viewport
   - **Auto-compaction** : après chaque réponse complète (dans le handler de `StreamEvent{Type: "done"}`), vérifier `llm.NeedsCompaction()` et déclencher automatiquement si nécessaire

#### Notes
- L'estimation de tokens est grossière (4 chars/token) mais suffisante pour un seuil. Les vrais tokens sont trackés dans `StreamEvent{Type: "done"}` mais seulement pour le dernier tour.
- La compaction utilise le provider ACTUEL pour résumer. C'est OK, ça utilise des tokens mais c'est le prix à payer.

---

### TÂCHE 3 : POLY.MD LOADER + SAFE/BANNED COMMANDS
**Agent : `config-dev`**
**Priorité : HAUTE**
**Fichiers à créer/modifier : `internal/config/polymd.go`, `internal/llm/system.go`, `internal/permission/permission.go`**

#### Partie A : POLY.md Loader

1. **Créer `internal/config/polymd.go`** :
   ```go
   package config

   import (
       "os"
       "path/filepath"
       "strings"
   )

   // LoadPolyMD searches for POLY.md (or poly.md) in cwd and parent directories.
   // Returns the concatenated content of all found files (parent first, then child).
   func LoadPolyMD() string {
       cwd, err := os.Getwd()
       if err != nil {
           return ""
       }

       var contents []string

       // Walk up from cwd to root, collecting POLY.md files
       dir := cwd
       for {
           for _, name := range []string{"POLY.md", "poly.md", ".poly/POLY.md"} {
               path := filepath.Join(dir, name)
               data, err := os.ReadFile(path)
               if err == nil && len(data) > 0 {
                   contents = append([]string{string(data)}, contents...) // parent first
               }
           }

           parent := filepath.Dir(dir)
           if parent == dir {
               break // reached root
           }
           dir = parent
       }

       if len(contents) == 0 {
           return ""
       }
       return strings.Join(contents, "\n\n---\n\n")
   }
   ```

2. **Injecter dans le system prompt** (`internal/llm/system.go`) :
   Dans `BuildSystemPrompt()`, APRÈS la section "OPERATIONAL CONTEXT" et AVANT la section "ROLE", ajouter :
   ```go
   // SECTION: Project Instructions (POLY.md)
   polyMD := config.LoadPolyMD()
   if polyMD != "" {
       prompt.WriteString("=== PROJECT INSTRUCTIONS (from POLY.md) ===\n")
       prompt.WriteString(polyMD)
       prompt.WriteString("\n\n")
   }
   ```

#### Partie B : Safe/Banned Command Lists

3. **Modifier `internal/permission/permission.go`** :
   Ajouter des listes de commandes safe et banned :
   ```go
   // safeCommands are auto-approved shell commands (read-only, harmless)
   var safeCommands = []string{
       "ls", "cat", "head", "tail", "wc", "file", "which", "whereis", "whoami",
       "pwd", "echo", "date", "uname", "hostname",
       "git status", "git log", "git diff", "git branch", "git show",
       "go version", "go env", "node --version", "python --version",
       "cargo --version", "rustc --version",
   }

   // bannedCommands are ALWAYS denied (destructive, dangerous)
   var bannedCommands = []string{
       "rm -rf /", "rm -rf ~", "rm -rf *",
       "sudo rm", "sudo shutdown", "sudo reboot", "sudo halt",
       "mkfs", "dd if=", ":(){:|:&};:",
       "chmod -R 777 /", "chown -R",
       "> /dev/sda", "mv / ",
   }

   // ClassifyBashCommand checks if a bash command is safe, banned, or needs asking
   func ClassifyBashCommand(command string) Level {
       cmd := strings.TrimSpace(strings.ToLower(command))

       // Check banned first (deny always wins)
       for _, banned := range bannedCommands {
           if strings.Contains(cmd, strings.ToLower(banned)) {
               return Deny
           }
       }

       // Check safe commands
       for _, safe := range safeCommands {
           if strings.HasPrefix(cmd, strings.ToLower(safe)) {
               return Allow
           }
       }

       return Ask
   }
   ```

4. **Intégrer dans l'approbation** (`internal/tools/approval.go`) :
   Dans `NeedsApproval()`, pour le tool "bash", utiliser la classification spéciale :
   ```go
   func NeedsApproval(name string, args map[string]interface{}) bool {
       if YoloMode {
           return false
       }
       if IsToolAllowed(name) {
           return false
       }
       // Special handling for bash
       if name == "bash" {
           if cmd, ok := args["command"].(string); ok {
               level := permission.ClassifyBashCommand(cmd)
               if level == permission.Allow {
                   return false
               }
               if level == permission.Deny {
                   return true // will be denied in Execute
               }
           }
       }
       return permission.ClassifyTool(name) == permission.Ask
   }
   ```

   **ATTENTION** : La signature actuelle de `NeedsApproval` est `NeedsApproval(name string)`. Il faut la changer en `NeedsApproval(name string, args map[string]interface{})` et mettre à jour l'appel dans `Execute()` de `registry.go` pour passer `args`.

5. **Bloquer les commandes bannies dans `Execute()`** :
   Dans `registry.go`, avant d'exécuter le tool "bash", vérifier si la commande est bannée et retourner une erreur.

---

### TÂCHE 4 : CUSTOM PROVIDERS AGENTIC LOOP + RETRY/BACKOFF
**Agent : `providers-dev`**
**Priorité : HAUTE**
**Fichiers à modifier : `internal/llm/custom.go`, `internal/llm/anthropic.go` (retry pattern)**

#### Objectif
Les custom providers ne font actuellement qu'un seul tour de streaming - pas d'exécution de tools. Il faut leur donner la même boucle agentique que les providers natifs.

#### Implémentation

1. **Ajouter la boucle agentique aux custom providers** (`internal/llm/custom.go`) :

   Le pattern est identique à celui des providers natifs. Regarde `anthropic.go` ou `gpt.go` pour le pattern `agenticLoop()`. Le custom provider doit :

   a. Après le streaming, vérifier si la réponse contient des `tool_calls`
   b. Si oui, exécuter les tools via `tools.Execute()`
   c. Envoyer les résultats au provider et recommencer
   d. Boucler jusqu'à `max_tool_turns` ou fin sans tool calls

   Le fichier `custom.go` a déjà des parsers par format (`parseOpenAIStream`, `parseAnthropicStream`, `parseGoogleStream`). Ces parsers doivent être étendus pour parser les tool calls, pas juste le texte.

   **Pattern de référence** (depuis `gpt.go` ou `grok.go` - format OpenAI) :
   - Les tool calls arrivent dans `choices[0].delta.tool_calls[].function.name` et `choices[0].delta.tool_calls[].function.arguments`
   - Elles sont accumulées progressivement pendant le streaming
   - À la fin du stream, si `stop_reason == "tool_use"` ou si des tool calls existent, les exécuter

2. **Ajouter le support images aux custom providers** :
   Les messages sont actuellement construits comme `map[string]string`. Changer en `map[string]interface{}` et ajouter le support images selon le format (OpenAI: image_url, Anthropic: base64 source, Google: inlineData).

3. **Ajouter le retry/backoff** :
   Créer un helper `internal/llm/retry.go` :
   ```go
   package llm

   import (
       "math"
       "net/http"
       "time"
   )

   const (
       MaxRetries    = 3
       BaseDelay     = 1 * time.Second
       MaxDelay      = 30 * time.Second
   )

   // ShouldRetry returns true for retryable HTTP status codes
   func ShouldRetry(statusCode int) bool {
       return statusCode == http.StatusTooManyRequests ||
              statusCode == http.StatusInternalServerError ||
              statusCode == http.StatusBadGateway ||
              statusCode == http.StatusServiceUnavailable ||
              statusCode == http.StatusGatewayTimeout
   }

   // RetryDelay returns the delay for attempt n (exponential backoff with jitter)
   func RetryDelay(attempt int) time.Duration {
       delay := float64(BaseDelay) * math.Pow(2, float64(attempt))
       if delay > float64(MaxDelay) {
           delay = float64(MaxDelay)
       }
       return time.Duration(delay)
   }
   ```

   Puis intégrer le retry dans la boucle HTTP de chaque provider (dans la fonction qui fait le `http.Do(req)` - wraper avec un for loop de retry).

---

## INSTRUCTIONS D'EXÉCUTION

### Étape 1 : Créer l'équipe
```
TeamCreate("poly-dev", description="Implement Phase 2 features for Poly-go")
```

### Étape 2 : Créer les 4 tâches
Créer les tâches avec TaskCreate pour chacune des 4 tâches ci-dessus.

### Étape 3 : Lancer 4 agents en parallèle
Spawner 4 teammates de type `general-purpose` avec `model: "opus"` et `mode: "bypassPermissions"` :
1. `markdown-dev` → Tâche 1
2. `compaction-dev` → Tâche 2
3. `config-dev` → Tâche 3
4. `providers-dev` → Tâche 4

Chaque agent reçoit un prompt détaillé avec :
- Sa section de ce plan (copier/coller la tâche correspondante)
- L'instruction de lire le code existant AVANT de modifier
- L'instruction de compiler (`go build -o poly .`) après chaque modification
- L'instruction de marquer sa tâche completed quand fini

### Étape 4 : Supervision
Le team lead (toi) supervise, résout les conflits de merge si les agents modifient le même fichier, et compile le résultat final.

### Vérification finale
```bash
cd /home/pedro/PROJETS-AI/Poly-go && go build -o poly . && echo "BUILD OK"
```

---

## RÈGLES STRICTES

1. **LIRE avant d'écrire** : Chaque agent DOIT lire le fichier complet avant de le modifier.
2. **Compiler souvent** : `go build -o poly .` après chaque changement.
3. **Pas de sur-ingénierie** : Le minimum viable qui marche. Pas d'abstractions pour le futur.
4. **Pas de fichiers .md** : Ne créer AUCUN README ou documentation. Code seulement.
5. **Garder le style** : Suivre les patterns existants du codebase (nommage, structure, imports).
6. **Pas de tests unitaires** : Pedro ne les a pas demandés. Focus sur le code fonctionnel.
7. **Ne pas toucher aux features qui marchent** : Ne rien casser de ce qui existe.
