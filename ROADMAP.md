# Poly-Go — Roadmap

> Dernière mise à jour : 13 février 2026
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

## v0.3.1 — Stabilité (Mars 2026)

> Objectif : Rendre le produit fiable et testable

### Tests (P0)

- [ ] Coverage providers : tests pour anthropic.go, gpt.go, gemini.go, grok.go, custom.go
- [ ] Coverage streaming : tests pour update_stream.go, streaming.go
- [ ] Coverage tools : tests pour chaque tool dans registry.go
- [ ] Objectif : 40%+ de coverage globale (actuellement ~15%)

### CI/CD (P0)

- [ ] GitHub Actions : build + test + lint sur chaque push
- [ ] Makefile targets pour CI (make ci)

### Bug Fixes (P0)

- [ ] Cascade @all : UX à revoir (lag, affichage confus)
- [ ] Stress test avec 8+ providers configurés
- [ ] Responsive layout (header compact < 80 cols)

---

## v0.3.2 — UX Polish (Mars - Avril 2026)

> Objectif : Finir le polish de l'interface

### Dialogs (P1)

- [ ] Add Provider via Huh forms (cursor, sélection, validation)
- [ ] Help scrollable + keybindings complètes
- [ ] Framework de dialog commun (extraire boilerplate)

### Streaming UX (P1)

- [ ] Annulation streaming améliorée (Esc + résumé du contenu généré)
- [ ] Animations minimales (spinner streaming)

### Responsive (P2)

- [ ] Header compact < 80 cols
- [ ] InfoPanel auto-hide < 100 cols

---

## v0.4.0 — Multi-Voix v2 (Avril - Mai 2026)

> Objectif : Renforcer le différenciateur unique de Poly

### Orchestrateur Multi-Voix v2 (P1)

L'orchestrateur `@all` actuel est basique (séquentiel, le 1er répond, les autres "reviewent").

- [ ] Mode **Séquentiel amélioré** : chaque provider reçoit les réponses précédentes en contexte
- [ ] Mode **Review croisée** : provider A répond, providers B+C critiquent, A révise
- [ ] Mode **Consensus** : tous répondent, synthèse automatique des convergences/divergences
- [ ] Modes configurables par l'utilisateur (`/cascade mode review`)
- [ ] Affichage clair de qui parle et en réponse à qui

### Ollama First-Class (P1)

- [ ] Auto-détection d'Ollama (probe `localhost:11434` au démarrage)
- [ ] Liste des modèles locaux dans le model picker (tag `[local]`)
- [ ] Config simplifiée : `"ollama": { "url": "http://localhost:11434" }`
- [ ] Support serveur distant (`http://192.168.1.100:11434`)

### Coûts Avancés (P1)

- [ ] Token count par message (optionnel, dans l'info panel)
- [ ] Export des coûts (CSV/JSON, pour freelances)

---

## v0.5.0 — Qualité (Mai - Juin 2026)

> Objectif : Produit solide avant la release

- [ ] 60%+ coverage globale
- [ ] Coverage TUI : tests pour les interactions clavier principales
- [ ] Stress test 20+ providers
- [ ] Temps premier message < 500ms
- [ ] Image support dans le chat (drag & drop ou path)

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

### v0.3.0 ✅ (actuel)

| Métrique | Avant | Après |
|----------|-------|-------|
| Code mort TUI | ~2000 lignes | 0 |
| Espace chat (% écran) | ~75% | 100% (sans overlay) |
| Providers hardcodés | 4 | 0 (N dynamique) |
| Valeurs hardcodées provider | 12+ | 0 |
| Navigation 100% clavier | Non | Oui |

### v0.5.0

| Métrique | Actuel | Cible |
|----------|--------|-------|
| Test coverage | ~15% | 60%+ |
| CI/CD | Non | GitHub Actions |
| Cascade @all UX | Bancal | Utilisable |

### v1.0.0

| Métrique | Cible |
|----------|-------|
| Distribution | go install + homebrew + AUR |
| Documentation | README + GIFs + CONTRIBUTING |
| Providers testés | 20+ |
| Temps premier message | < 500ms |
