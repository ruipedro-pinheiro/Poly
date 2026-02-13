# Poly-go Providers & Auth Audit

> Audit date: 2026-02-10
> Source: `internal/llm/` and `internal/auth/`

---

## 1. Architecture Generale

### Interface Provider (`provider.go`)

Tous les providers implementent l'interface `Provider`:

```
Name() string
DisplayName() string
Color() string
ToolFormat() ToolFormat
Send(ctx, messages, tools) (*Response, error)
Stream(ctx, messages, tools, opts) <-chan StreamEvent
IsConfigured() bool
SetModel(model)
GetModel() string
SupportsTools() bool
```

- `Send()` n'est implemente nulle part (retourne toujours "not implemented") -- tout passe par `Stream()`
- Le streaming utilise un `chan StreamEvent` avec buffer de 64
- `StreamEvent.Type` : "content", "thinking", "tool_use", "tool_result", "done", "error"

### Structures de donnees cles

| Struct | Role |
|--------|------|
| `Message` | role/content/images/tool_calls/tool_result |
| `Image` | data ([]byte) + media_type + path |
| `ToolCall` | id/name/arguments |
| `ToolResult` | tool_use_id/content/is_error |
| `Response` | content/provider/model/input_tokens/output_tokens/tool_calls/stop_reason |
| `StreamOptions` | Role ("default"/"responder"/"reviewer") + ThinkingMode bool |

### Registry

- Map globale `providerRegistry` protegee par `sync.RWMutex`
- Chaque provider s'enregistre via `init()` dans son fichier
- Ordre d'affichage : claude, gpt, gemini, grok, puis custom
- Image support track auto-detecte (essaye si inconnu, memorise le resultat)

---

## 2. Providers Detailles

### 2.1 Claude / Anthropic (`anthropic.go`)

| Propriete | Valeur |
|-----------|--------|
| Name | "claude" |
| Color | #D97706 (ambre) |
| ToolFormat | `ToolFormatAnthropic` |
| Endpoint | `https://api.anthropic.com/v1/messages` |
| API version | `2023-06-01` |

**Authentication duale :**
1. **OAuth** : Verifie `auth.GetStorage().IsConnected("claude")` d'abord
   - Si OAuth : `Authorization: Bearer <token>`, URL `?beta=true`, headers beta `oauth-2025-04-20,interleaved-thinking-2025-05-14`
   - System prompt FORCE a `ClaudeOAuthSystemPrompt` ("You are Claude Code, Anthropic's official CLI for Claude.")
   - L'identite Poly est injectee via un echange user/assistant fake au debut de la conversation
   - Les noms d'outils sont prefixes avec `mcp_` en mode OAuth (et deprefixes au retour)
2. **API Key** : `x-api-key: <key>`, system prompt libre via `body["system"]`
3. **Fallback** : `ANTHROPIC_API_KEY` env var

**Streaming SSE :**
- Parse event-by-event : `message_start`, `content_block_start`, `content_block_delta`, `content_block_stop`, `message_delta`, `message_stop`
- Supporte les deltas : `text_delta`, `thinking_delta`, `signature_delta`, `input_json_delta`
- Tokens : input depuis `message_start.message.usage`, output depuis `message_delta.usage`

**Boucle agentique :**
- Max turns configurable via `GetMaxToolTurns()` (defaut 50)
- Execute les outils via `tools.Execute()` a chaque tour
- Emet `tool_use` avant et `tool_result` apres chaque execution
- En thinking mode : `thinkingBlocks` avec signature preservee pour le multi-turn

**Extended Thinking :**
- `budget_tokens: 10000`, max_tokens augmente si necessaire
- Header beta `interleaved-thinking-2025-05-14`
- Parse thinking blocks avec signature pour re-injection dans la conversation

**Images :**
- Format base64 inline : `{ type: "image", source: { type: "base64", media_type, data } }`

---

### 2.2 GPT / OpenAI (`gpt.go`)

| Propriete | Valeur |
|-----------|--------|
| Name | "gpt" |
| Color | #10A37F (vert OpenAI) |
| ToolFormat | `ToolFormatOpenAI` |
| Endpoint | `https://api.openai.com/v1/chat/completions` |

**Authentication :**
- Uniquement via `auth.GetStorage().GetAccessToken("gpt")` (OAuth ou API key)
- Header `Authorization: Bearer <token>`
- Pas de fallback env var

**Streaming SSE :**
- Format standard OpenAI : `choices[0].delta.content` et `choices[0].delta.tool_calls[]`
- `stream_options: { include_usage: true }` pour recuperer les tokens en fin de stream
- Tool calls construites incrementalement via map indexee `toolCallsMap[index]`
- Arguments accumules en raw string puis parses en JSON a la fin

**Boucle agentique :**
- Meme pattern que Claude : boucle avec max tool turns
- Assistant message inclut `tool_calls` array au format OpenAI
- Tool results envoyes comme messages role "tool" avec `tool_call_id`

**Extended Thinking (Reasoning) :**
- Detection modeles reasoning : `isReasoningModel()` = prefixe "o3" ou "o4"
- Pour o3/o4 : `reasoning_effort: "high"`, `max_completion_tokens` au lieu de `max_tokens`
- Emet un event thinking fake "(reasoning...)" car OpenAI ne stream pas le raisonnement

**Images :**
- Format data URL : `{ type: "image_url", image_url: { url: "data:mime;base64,..." } }`

---

### 2.3 Gemini / Google (`gemini.go`)

| Propriete | Valeur |
|-----------|--------|
| Name | "gemini" |
| Color | #4285F4 (bleu Google) |
| ToolFormat | `ToolFormatGoogle` |
| Endpoints | Public API + Code Assist |

**Deux modes d'operation :**

#### Mode API Key (Public API)
- Detection : token commence par `AIza`
- Endpoint : `https://generativelanguage.googleapis.com/v1beta/models/{model}:streamGenerateContent?alt=sse&key={key}`
- Modele par defaut : `gemini-2.5-flash`

#### Mode OAuth (Code Assist)
- Endpoint : `https://cloudcode-pa.googleapis.com/v1internal:streamGenerateContent?alt=sse`
- Modele par defaut : `gemini-2.5-pro`
- Necessite un `projectID` resolu via `loadCodeAssist` (env vars `GOOGLE_CLOUD_PROJECT`, `GOOGLE_CLOUD_PROJECT_ID`, `GCLOUD_PROJECT`)
- Body wrapper avec `model`, `project`, `user_prompt_id`, `session_id`
- Parsing special : les SSE events sont dans `response.candidates[0].content.parts[]`
- Deltas calcules manuellement (text courant - text deja envoye)

**System prompt :**
- Pas de champ `system` natif utilise -- injecte via paire user/model au debut
- `"Understood. I'm Gemini, chatting through Poly."`

**Streaming :**
- Parse `candidates[0].content.parts[]` pour chaque SSE event
- Chaque part peut etre : `text`, `thought: true` + `text`, ou `functionCall`
- `usageMetadata` : `promptTokenCount` + `candidatesTokenCount`

**Function Calling (Tools) :**
- Format Google : `{ functionDeclarations: [{ name, description, parameters }] }`
- Pas d'ID pour les function calls (utilise le nom comme ID)
- Responses : `{ functionResponse: { name, response: { content } } }` en role "user"

**Extended Thinking :**
- `generationConfig.thinkingConfig.thinkingBudget: 8192`
- Parts avec `thought: true` parsees separement

**Images :**
- Format `inlineData` : `{ inlineData: { mimeType, data } }`

---

### 2.4 Grok / xAI (`grok.go`)

| Propriete | Valeur |
|-----------|--------|
| Name | "grok" |
| Color | #1DA1F2 (bleu xAI) |
| ToolFormat | `ToolFormatOpenAI` |
| Endpoint | `https://api.x.ai/v1/chat/completions` |

**Authentication :**
- Via `auth.GetStorage().GetAccessToken("grok")`
- Header `Authorization: Bearer <token>`

**Implementation :**
- Quasi identique a GPT (meme format OpenAI)
- Meme structure de streaming, tool calls, parsing
- `stream_options: { include_usage: true }`
- **PAS de support extended thinking** (le param `thinkingMode` n'est pas passe a `agenticLoop`)

**Images :**
- Meme format data URL qu'OpenAI

---

### 2.5 Custom Providers (`custom.go`)

| Propriete | Valeur |
|-----------|--------|
| Config file | `~/.poly/providers.json` |
| Formats supportes | "openai", "anthropic", "google" |

**Configuration :**
```json
{
  "name": "Display Name",
  "id": "unique-id",
  "base_url": "https://api.example.com/v1",
  "api_key": "...",
  "model": "model-name",
  "format": "openai",
  "color": "#hex",
  "max_tokens": 4096,
  "auth_header": "Bearer"
}
```

**Fonctionnalites :**
- `IsConfigured()` retourne TOUJOURS `true` (pas de verification API key -- concu pour les modeles locaux)
- URL construite dynamiquement selon format : `/messages` (anthropic), `/chat/completions` (openai), `/models/{model}:streamGenerateContent?alt=sse` (google)
- Auth header configurable : `Bearer` (defaut), `x-api-key`, ou rien (modeles locaux)
- Supporte `BuildRequestWithTools()` pour tous les formats

**Limitations majeures :**
- PAS de boucle agentique ! Un seul tour de streaming, pas d'execution de tools
- Les parsers (parseOpenAIStream, parseAnthropicStream, parseGoogleStream) ne gerent PAS les tool calls -- seulement le contenu texte et thinking
- Pas de support images (messages construits avec `map[string]string`, pas `interface{}`)
- Pas de token tracking (Response.InputTokens/OutputTokens toujours 0)

**CRUD :**
- `SaveCustomProvider()` : ajoute ou met a jour dans providers.json
- `DeleteCustomProvider()` : supprime par ID
- `GetCustomProviders()` : liste toutes les configs
- `LoadCustomProviders()` : charge et enregistre au demarrage

---

## 3. Tool Format Conversion (`tools_format.go`)

Trois formats supportes, conversion centralisee :

| Format | Structure |
|--------|-----------|
| **Anthropic** | `{ name, description, input_schema }` |
| **OpenAI** | `{ type: "function", function: { name, description, parameters } }` |
| **Google** | `{ functionDeclarations: [{ name, description, parameters }] }` |

**Fonctions :**
- `ConvertToolsForProvider(tools, format)` : convertit un slice de ToolDefinition
- `WrapToolsForGoogle(tools)` : wrappe dans `{ functionDeclarations: [...] }`
- `BuildRequestWithTools(body, tools, format)` : ajoute directement au body de la requete

**Note :** La ToolDefinition interne utilise `input_schema` (format Anthropic) comme format canonique. La conversion vers OpenAI/Google se fait au moment de l'envoi.

---

## 4. Systeme OAuth / Auth

### 4.1 Storage (`storage.go`)

- Fichier : `~/.config/poly/auth.json` (respecte XDG_CONFIG_HOME)
- Permissions : `0600` (lecture/ecriture owner uniquement)
- Singleton thread-safe avec `sync.Once` + `sync.RWMutex`

**Structure ProviderAuth :**
- `type` : "oauth" ou "apikey"
- `provider` : nom du provider
- `tokens` : access_token, refresh_token, expires_at
- `api_key` : cle API directe

**Auto-refresh :** `GetAccessToken()` verifie `expires_at` avec buffer de 10 minutes et refresh automatiquement selon le provider.

### 4.2 Anthropic OAuth PKCE (`anthropic.go`)

- **Flow** : Authorization Code + PKCE (S256)
- **Scopes** : `org:create_api_key user:profile user:inference`
- **Redirect URI** : `https://console.anthropic.com/oauth/code/callback` (pas de serveur local)
- **Client ID** : depuis config (`config.Get().Providers["claude"].OAuthClientID`)
- **Deux modes** : claude.ai (Max) ou console
- **Echange** : Accepte URL callback complete, format `CODE#STATE`, ou code brut
- **Refresh** : Standard OAuth2 refresh_token grant

### 4.3 Google OAuth (`google.go`)

- **Client ID/Secret** : Hardcodes (credentials publiques du Gemini CLI)
- **Flow** : Authorization Code avec serveur local sur port 8086
- **Scopes** : cloud-platform, userinfo.email, userinfo.profile
- **Serveur callback** : HTTP local `127.0.0.1:8086/callback`, page HTML de succes
- **Timeout** : 5 minutes
- **Refresh** : Google ne retourne pas de nouveau refresh_token

### 4.4 OpenAI OAuth PKCE (`openai.go`)

- **Client ID** : `app_EMoamEEZ73f0CkXaXp7hrann` (credentials publiques Codex CLI)
- **Flow** : Authorization Code + PKCE (S256) avec serveur local port 1455
- **Scopes** : `openid profile email offline_access model.request`
- **Params speciaux** : `id_token_add_organizations: true`, `codex_cli_simplified_flow: true`, `originator: poly`
- **Serveur callback** : HTTP local `127.0.0.1:1455/callback`
- **Timeout** : 5 minutes
- **Refresh** : Garde ancien refresh_token si le nouveau est vide

### 4.5 PKCE (`pkce.go`)

- Verifier : 32 bytes aleatoires, base64url-encoded
- Challenge : SHA256(verifier), base64url-encoded
- State : 16 bytes aleatoires, base64url-encoded
- Utilise par Anthropic et OpenAI

### 4.6 Grok Auth

- **Pas d'OAuth** pour Grok -- uniquement API key via storage

---

## 5. Pricing / Cost Tracking (`pricing.go`)

**Modeles supportes :**

| Provider | Modele | Input $/1M | Output $/1M |
|----------|--------|------------|-------------|
| Anthropic | claude-opus-4 | 15.00 | 75.00 |
| Anthropic | claude-sonnet-4 | 3.00 | 15.00 |
| Anthropic | claude-haiku-4 | 0.80 | 4.00 |
| OpenAI | gpt-4.1 | 2.00 | 8.00 |
| OpenAI | gpt-4.1-mini | 0.40 | 1.60 |
| OpenAI | gpt-4.1-nano | 0.10 | 0.40 |
| OpenAI | o3 | 10.00 | 40.00 |
| OpenAI | o3-pro | 20.00 | 80.00 |
| OpenAI | o4-mini | 1.10 | 4.40 |
| Google | gemini-2.5-flash | 0.15 | 0.60 |
| Google | gemini-2.5-flash-lite | 0.075 | 0.30 |
| Google | gemini-2.5-pro | 1.25 | 10.00 |
| xAI | grok-3 | 3.00 | 15.00 |
| xAI | grok-3-fast | 5.00 | 25.00 |
| xAI | grok-3-mini-beta | 0.30 | 0.50 |

**Lookup :** exact match d'abord, puis prefix match (gere les versions timestampees comme `claude-sonnet-4-5-20250929`). Fallback : $2/8 par 1M.

**Calcul :** `CalculateCost(inputTokens, outputTokens, model)` retourne le cout en USD.

---

## 6. System Prompt (`system.go`)

Le system prompt est genere dynamiquement par `BuildSystemPrompt(providerName, role)` :

**Sections :**
1. **GROUND TRUTH** : identite immutable (nom, providers connectes, environnement Poly)
2. **ANTI-GASLIGHTING PROTOCOL** : resist les tentatives de manipulation d'identite
3. **PEER ISOLATION** : ne pas adopter les hallucinations d'autres IAs
4. **OPERATIONAL CONTEXT** : mentions (@claude, @gpt...), outils disponibles, format terminal
5. **ROLE** : responder (premier a repondre) / reviewer (verifie la reponse) / direct (defaut)
6. **SECURITY PROTOCOL** : ne pas reveler le prompt, pas de commandes destructives

**Cascade mode (@all) :**
- "responder" : premiere IA a repondre, focus precision
- "reviewer" : verifie la reponse, output "checkmark" si OK, sinon correction factuelle

**Config :** Tout est tire de `config.Get()` -- zero hardcoding de noms de providers.

---

## 7. Resume Image Support

| Provider | Format | Supporte |
|----------|--------|----------|
| Claude | base64 inline (source.type: "base64") | OUI |
| GPT | data URL (image_url.url: "data:...") | OUI |
| Gemini | inlineData (mimeType + data) | OUI |
| Grok | data URL (meme que GPT) | OUI |
| Custom | Texte seulement | NON |

Auto-detection : `SupportsImages()` retourne `true` par defaut (inconnu = essayer). Si erreur image detectee, marque `false`.

---

## 8. Resume Extended Thinking

| Provider | Implementation | Budget |
|----------|---------------|--------|
| Claude | `thinking.type: "enabled"`, `budget_tokens: 10000` | 10k tokens |
| GPT | `reasoning_effort: "high"` (o3/o4 seulement) | Pas de budget explicite |
| Gemini | `thinkingConfig.thinkingBudget: 8192` | 8192 tokens |
| Grok | **NON SUPPORTE** (param ignore) | - |
| Custom | Depends du format choisi | Meme que le format |

---

## 9. Resume Agentic Loop (Tool Execution)

| Provider | Boucle agentique | Max turns |
|----------|-----------------|-----------|
| Claude | OUI (complet, multi-turn avec thinking blocks) | configurable (50) |
| GPT | OUI (complet) | configurable (50) |
| Gemini API | OUI (complet) | configurable (50) |
| Gemini CodeAssist | OUI (complet) | configurable (50) |
| Grok | OUI (complet) | configurable (50) |
| Custom | **NON** (un seul tour, pas d'execution de tools) | 1 |

---

## 10. Ce Qui Manque vs Concurrents

### Manques critiques

1. **Custom providers : pas de boucle agentique** -- les tools sont envoyes mais jamais executes. Les custom providers sont text-only en pratique.

2. **Custom providers : pas de support images** -- les messages sont construits avec `map[string]string` au lieu de `map[string]interface{}`.

3. **Custom providers : pas de token tracking** -- InputTokens/OutputTokens toujours 0, donc pas de calcul de cout.

4. **Send() non implemente** -- tous les providers retournent "not implemented". Pas de mode non-streaming.

5. **Grok : pas d'extended thinking** -- le parametre `thinkingMode` n'est pas propage depuis `Stream()` vers `agenticLoop()`.

### Manques secondaires

6. **Pas de streaming de raisonnement pour GPT** -- seulement un message statique "(reasoning...)" emis avant la requete. Le vrai streaming du raisonnement o3/o4 n'est pas parse.

7. **Pas de retry/backoff** -- `maxRetries` et `retryDelay` sont declares dans anthropic.go mais jamais utilises. Aucun provider ne retry sur erreur 429/500.

8. **Gemini Code Assist : generateUUID() faible** -- utilise `time.Now().UnixNano()` au lieu de crypto/rand. Pas un vrai UUID.

9. **Google OAuth credentials hardcodees** -- Client ID et Secret du Gemini CLI directement dans le code source.

10. **Pas de support pour Mistral, DeepSeek, Cohere natifs** -- ils doivent passer par le systeme custom (sans agentic loop ni images).

11. **Pas de fonction calling pour les modeles reasoning OpenAI** -- les o3/o4 n'ont pas de gestion speciale des tool calls en mode reasoning.

12. **Cost tracking incomplet** -- pas de totalisation par session/conversation, juste un calcul unitaire.

### Vs Claude Code

- Claude Code a un mode non-streaming pour certaines operations
- Claude Code a un retry avec exponential backoff
- Claude Code gere les rate limits (429) nativement

### Vs Gemini CLI

- Gemini CLI utilise les memes credentials OAuth (hardcoded dans les deux)
- Gemini CLI a un meilleur support du Code Assist project resolution
- Poly gere mieux le multi-provider (Gemini CLI = Gemini only)

### Vs OpenCode

- OpenCode a un systeme de providers plus modulaire (plugins)
- Poly a un meilleur cascade mode (@all responder/reviewer)
- Poly a un support OAuth plus complet (3 providers vs API key seulement dans OpenCode)

---

## 11. Tests (`provider_test.go`)

Tests existants :
- `TestProviderRegistry` : verifie que le registre n'est pas vide
- `TestGetProvider` : lookup par nom (claude), non-existent
- `TestGetConfiguredProviders` : verifie `IsConfigured()` consistency
- `TestGetProviderNames` : verifie que les noms ne sont pas vides
- `TestMessageValidation` : validation manuelle role/content
- `TestToolCallStructure` : verification champs ToolCall
- `TestImageSupport` : set/get image support

**Manques dans les tests :**
- Aucun test de streaming (necessite mock HTTP)
- Aucun test de la boucle agentique
- Aucun test de tool format conversion
- Aucun test de pricing/cost calculation
- Aucun test du system prompt generation
- Aucun test d'OAuth flow
