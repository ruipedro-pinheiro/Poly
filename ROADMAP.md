# Poly-Go — Roadmap

> Dernière mise à jour : 14 février 2026 (v0.4.0)
> Basée sur le [Cahier des Charges](./CAHIER_DES_CHARGES.md) et les [Personas/Use Cases](./docs/ux/02-personas-use-cases.md)

---

## Légende

| Symbole | Signification |
|---------|---------------|
| P0 | Indispensable — bloquant pour la suite |
| P1 | Important — différenciateur clé |
| P2 | Nice-to-have — polish et avantage compétitif |
| P3 | Vision — long terme, pas avant stabilisation |

---

## v0.3.0 — TUI Rework (Février 2026) ✅

> Objectif : Passer de "prototype fonctionnel" à "TUI clean et scalable"

### Sprint 1-3 — Nettoyage + Dynamique + Layout ✅

- [x] Supprimé ~2000 lignes de code mort (dialogs/, messages/, sidebar)
- [x] Providers dynamiques depuis config (N illimité, palette Catppuccin cyclique)
- [x] Header 1 ligne, status bar 1 ligne, messages 100% largeur
- [x] Sidebar supprimée du layout principal
- [x] Focus clavier viewport (Tab/Shift+Tab, j/k scroll)
- [x] Command palette synced avec CommandRegistry (30+ commands)

### Sprint 4 — Info Panel Overlay ✅

- [x] Nouveau composant InfoPanel (overlay droit, 35 cols, Ctrl+I toggle)
- [x] Session (tokens, coût, context bar), providers, files, MCP, status badges
- [x] Sidebar supprimée (594 lignes)

### Sprint 5 — Hardcoded Values ✅

- [x] Tous les noms de providers supprimés du TUI (parseProvider, filemention, splash, Control Room)
- [x] OAuth dispatch centralisé et config-driven
- [x] Default provider depuis config

### Sprint 6 — Polish ✅

- [x] Thinking blocks collapsés par défaut (▶/▼, touche `t`)
- [x] Tool calls batch summary (✓ 4 tools: read x2, bash, edit)
- [x] Per-provider costs dans InfoPanel ($0.12 ou N/A)
- [x] Cascade cost estimation (message système avant @all)

---

## v0.3.1 — Stabilité (Février 2026) ✅

> Objectif : Rendre le produit fiable et testable

### Tests (P0) ✅

- [x] 28+ tests : pricing_test.go (15), edit_test.go (13+), pathcheck_test.go (8)
- [x] Tests config, memory, polymd, provider registry, permission, retry, compaction, MCP manager
- [ ] Coverage providers : tests pour anthropic.go, gpt.go, gemini.go, grok.go, custom.go *(reporté v0.5.0)*
- [ ] Coverage streaming : tests pour update_stream.go, streaming.go *(reporté v0.5.0)*
- [ ] Objectif : 40%+ de coverage globale *(reporté v0.5.0, actuellement ~15-25%)*

### CI/CD (P0) ✅

- [x] GitHub Actions : build + vet + test -race sur chaque push/PR
- [x] `.github/workflows/ci.yml`
- [x] Makefile targets : `make ci`, `make test`, `make build`, `make lint`

### Responsive (P0) ✅

- [x] Header responsive 3 breakpoints (<60, 60-79, 80+ cols)

### Non fait *(reporté)*

- [ ] Cascade @all : UX à revoir (lag, affichage confus) *(reporté v0.4.0)*
- [ ] Stress test avec 8+ providers configurés *(reporté v0.5.0)*

---

## v0.3.2 — UX Polish (Février 2026) ✅

> Objectif : Finir le polish de l'interface

### Dialogs (P1) ✅

- [x] Help dialog : providers dynamiques depuis config, keybindings complètes
- [ ] Add Provider via Huh forms (cursor, sélection, validation) *(reporté v0.4.0)*
- [ ] Framework de dialog commun (extraire boilerplate) *(reporté v0.5.0)*

### Streaming UX (P1) ✅

- [x] Annulation streaming (Esc) : résumé "Cancelled after ~X tokens, Y tools"
- [ ] Animations minimales (spinner streaming) *(reporté v0.5.0)*

### Responsive (P2) ✅

- [x] InfoPanel auto-hide < 100 cols

---

## v0.3.3 — Hardening (Février 2026) ✅

> Objectif : Sécurité, nettoyage, et correction des wiring manquants

### Sécurité (P0) ✅

- [x] `apply_diff` : ajout ValidatePath() avant écriture (bypass critique corrigé)
- [x] `git_diff`/`git_log`/`git_status` : ValidatePath() sur argument path
- [x] Custom provider `IsConfigured()` vérifie la clé API (au lieu de toujours true)

### Dead Code Removal (P1) ✅

- [x] ~773 lignes supprimées : editor component, list package, old palette system, DefaultStyles, min/max helpers, Send() interface
- [x] Provider interface simplifiée (Send() retiré, Stream() seul)

### TUI Wiring (P1) ✅

- [x] Header reçoit les tokens/coûts (context% fonctionne)
- [x] InfoPanel affiche les serveurs MCP
- [x] InfoPanel affiche le badge sandbox
- [x] @mentions dynamiques depuis config (plus de hardcoded)
- [x] /compare résultats persistés en session
- [x] Warning max tool turns standardisé (emoji partout)

---

## v0.4.0 — Multi-Voix v2 (Avril - Mai 2026)

> Objectif : Renforcer le différenciateur unique de Poly

### Table Ronde — Conversation inter-IAs (P0) ✅

- [x] **Inter-mentions dynamiques** : tout provider peut `@mentionner` n'importe quel autre provider configuré
- [x] **Détection des @mentions** dans les réponses IA → déclenche une réponse du provider mentionné
- [x] **Contexte partagé** : chaque provider reçoit toute la conversation (y compris les réponses des autres)
- [x] **Max turns** : limite configurable `/rounds [N]` (défaut 5, max 20)
- [x] **Esc coupe tout** : l'user peut stopper la table ronde à tout moment
- [x] **System prompt enrichi** : role "participant", chaque IA sait quels providers sont dans le chat
- [x] **Affichage clair** : bordure couleur provider (existant), messages système par round
- [x] Ancien code cascade supprimé (CascadePhase, CascadeStreamMsg, cascadeState)

### Add Provider Rework (P1) ✅

- [x] Add Provider via custom forms (textinput.Model, Catppuccin theme, password mask, validation)
- [x] Format selector (openai/anthropic/google) avec cycle h/l

### Coûts Avancés (P1) ✅

- [x] Token count par message (InputTokens/OutputTokens, annotation sous chaque message)
- [x] Export des coûts (`/costs csv` ou `/costs json`)
- [x] Per-provider costs dans InfoPanel

### Ollama First-Class (P1) *(reporté — pas testable sur ThinkPad)*

- [ ] Auto-détection d'Ollama (probe `localhost:11434` au démarrage)
- [ ] Liste des modèles locaux dans le model picker (tag `[local]`)
- [ ] Config simplifiée : `"ollama": { "url": "http://localhost:11434" }`
- [ ] Support serveur distant (`http://192.168.1.100:11434`)

---

## v0.5.0 — Qualité (Mai - Juin 2026)

> Objectif : Produit solide avant la release

### Hardening (P0) ✅

- [x] File permissions : 0644→0600 (config, session, providers, history, auth, updater)
- [x] Race conditions : sync.RWMutex/Mutex/Once sur globals (streaming, gemini, auth)
- [x] Security : Google OAuth secret → env var, hooks injection fix, shared blocklist, SSRF check
- [x] Best practices : crypto/rand UUID, error checking, dead code removal

### Docker & Portabilité (P0) ✅

- [x] `Dockerfile` multi-stage : build avec Go 1.25.6, binaire statique final
- [x] `.dockerignore` pour build propre
- [ ] Sandbox Docker activé par défaut (les tools LLM s'exécutent dans un conteneur) *(reporté v1.0)*
- [ ] Volumes/bind mounts *(reporté v1.0)*
- [ ] `docker run` one-liner pour lancer Poly sans installation

### Tests & Performance (P0)

- [x] 10 nouveaux fichiers de tests (security 100%, config 47%, session 41%, hooks 37%, llm 23%, tools 22%)
- [ ] 60%+ coverage globale *(en cours — TUI à 0.5% tire la moyenne vers le bas)*
- [ ] Coverage TUI : tests pour les interactions clavier principales
- [ ] Coverage providers streaming : 0% actuellement, critique
- [ ] Coverage auth/OAuth : 0% actuellement
- [ ] Stress test 20+ providers
- [ ] Temps premier message < 500ms
- [ ] Image support dans le chat (drag & drop ou path)

### Rendering (P2)

- [ ] Side-by-side diff view (actuellement line-by-line seulement)
- [ ] Vérifier rendu Glamour + Chroma en runtime (code présent dans markdown.go, à confirmer visuellement)

### Custom Providers Fix (P0) — BLOQUANT POUR CREDIBILITE

- [ ] **Agentic loop** pour custom providers (actuellement single-turn, tools jamais exécutés)
- [ ] **Image support** pour custom providers (map[string]string → interface{})
- [ ] **Token tracking** pour custom providers (InputTokens/OutputTokens toujours à 0)
- [ ] **Cost calculation** pour custom providers (dépend du token tracking)

### Wiring manquant (P1)

- [ ] **Retry/backoff** : retry.go existe mais aucun provider ne l'appelle — brancher
- [ ] **Send()** : tous les providers retournent "not implemented" — implémenter ou supprimer de l'interface
- [ ] **strings.Title()** déprécié : remplacer dans modelpicker.go et update_keys.go
- [ ] **DefaultContextWindow** : adapter au provider (200K fixe → Gemini=1M, local=32K, etc.)

---

## v1.0.0 — Release Publique (Juin - Juillet 2026)

> Objectif : Prêt pour des utilisateurs externes

### Documentation (P0)

- [ ] README enrichi avec GIFs (VHS) et screenshots (Freeze)
- [ ] Documentation utilisateur (site ou wiki)
- [ ] CONTRIBUTING.md + templates issues/PR

### Distribution (P0)

- [ ] `go install` depuis GitHub
- [ ] Homebrew tap (macOS)
- [ ] AUR package (Arch Linux)
- [ ] Publication sur Terminal Trove, Hacker News, Reddit r/commandline

---

## v1.x — Vision (Post-release)

> Objectif : Devenir LA référence multi-AI terminal

### Meta-Routes Intelligentes (P2)

- [ ] `@fast` → provider avec la plus basse latence
- [ ] `@cheap` → provider le moins cher
- [ ] `@local` → tous les providers locaux
- [ ] `@best` → meilleur score historique pour ce type de tâche

### Thèmes et Personnalisation (P2)

- [ ] Support Catppuccin complet (Latte, Frappé, Macchiato)
- [ ] Thèmes custom via config.json

### Plugin System (P3)

- [ ] Skills écrites en Go (chargement dynamique)
- [ ] Marketplace / registry communautaire
- [ ] Hooks configurables riches (pre/post tool avec conditions)

### Benchmarks Personnels (P3)

- [ ] Évaluation : même prompt → N providers → scoring
- [ ] Recommandation automatique

### Exploration (P3, non engagé)

- [ ] Mode conversation inter-IAs
- [ ] Intégration git profonde
- [ ] Voice input (whisper.cpp local)
- [ ] Web UI optionnelle

---

## Dépendances et Risques

| Risque | Impact | Mitigation |
|--------|--------|------------|
| Huh (Charm.sh) pas assez flexible | Bloque Add Provider rework | Fallback sur textinput amélioré |
| Bubble Tea v2 breaking changes | Bloque tout | Verrouiller la version, surveiller les releases |
| API providers changent leurs formats | Bloque providers | L'abstraction custom.go + retry/backoff absorbent |
| Pedro à 42 = temps limité | Ralentit la roadmap | Sprints courts (2-4h), autonomie max des agents |
| Competition (Crush, OpenCode) | Gap qui se réduit | Le multi-voix dans le même chat est unique |

---

## Métriques de Succès

### v0.3.x ✅

| Métrique | v0.2.x | v0.3.3 |
|----------|--------|--------|
| Code mort TUI | ~2000 lignes | 0 (+ ~773 lignes nettoyées en v0.3.3) |
| Espace chat (% écran) | ~75% | 100% (sans overlay) |
| Providers hardcodés | 4 | 0 (N dynamique) |
| Valeurs hardcodées provider | 12+ | 0 |
| Navigation 100% clavier | Non | Oui |
| CI/CD | Non | GitHub Actions (build + vet + test -race) |
| Vulnérabilités connues | 2 (apply_diff, git path) | 0 |

### v0.4.0 (actuel — Table Ronde)

| Métrique | v0.3.3 | v0.4.0 |
|----------|--------|--------|
| @all orchestration | 2-phase rigide (cascade) | Multi-round Table Ronde |
| Inter-IA @mentions | Non | Oui (case-insensitive, dedup) |
| Contexte partagé | Tronqué pour reviewers | Full context tous rounds |
| Max rounds configurable | Non | `/rounds [N]` (défaut 5) |
| Tests Table Ronde | 0 | 7 (extractMentions + config + system) |

### v0.5.0

| Métrique | Actuel | Cible |
|----------|--------|-------|
| Test coverage | ~15-25% | 60%+ |
| Docker portabilité | Non | Dockerfile + sandbox par défaut |
| Stress test providers | Non testé | 20+ |

### v1.0.0

| Métrique | Cible |
|----------|-------|
| Distribution | go install + homebrew + AUR |
| Documentation | README + GIFs + CONTRIBUTING |
| Providers testés | 20+ |
| Temps premier message | < 500ms |
