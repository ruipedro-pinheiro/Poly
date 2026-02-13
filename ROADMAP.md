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

## Court Terme (Février - Mars 2026)

> Objectif : Passer de "prototype fonctionnel" à "v1.0 clean et scalable"

### Sprint 1 — Nettoyage (P0, ~2h)

Avant de toucher quoi que ce soit d'autre.

- [ ] Supprimer `components/dialogs/` — 7 sous-dossiers, ~1700 lignes mortes jamais importées
- [ ] Supprimer `components/messages/` — 3 fichiers, ~325 lignes mortes jamais importées
- [ ] Supprimer `sidebar.getCachedMCP()` — code mort
- [ ] Unifier `formatTokens()` (dupliqué dans sidebar.go et status.go)
- [ ] Unifier `session.Clear()` → `session.SetMessages()` partout
- [ ] Remplacer `strings.Title()` (déprécié) dans modelpicker.go et update_keys.go
- [ ] Centraliser les constantes de dialog (largeurs 40-80 incohérentes → 1 constante)
- [ ] **Build + tests verts**
- [ ] **Commit**

### Sprint 2 — Provider Dynamique (P0, ~3h)

Le cœur de la scalabilité. Rien d'autre ne scale tant que ça c'est pas fait.

- [ ] `config.GetProviderNames()` → dynamique (lire depuis config, pas `["claude","gpt","gemini","grok"]`)
- [ ] `variantOrder` → dynamique depuis la config de chaque provider
- [ ] Auto-assignation des couleurs : palette Catppuccin cyclique (12 couleurs), override possible en config
- [ ] `DefaultContextWindow` → lu depuis la config provider/modèle (Gemini=1M, local=32k, etc.)
- [ ] Custom providers traités identiquement aux natifs partout (splash, help, Control Room, sidebar, model picker)
- [ ] Splash dynamique : lister TOUS les providers configurés, pas juste les 4 hardcodés
- [ ] Version dans splash synchronisée avec `--version` (git tag, pas "v0.2.0" hardcodé)
- [ ] **Tester avec 8+ providers configurés**
- [ ] **Build + tests verts**
- [ ] **Commit**

### Sprint 3 — Layout (P0, ~4h)

Récupérer l'espace perdu et moderniser l'affichage.

- [ ] Header → 1 ligne : `◇ POLY │ @provider/variant │ contexte% │ cwd`
- [ ] Sidebar permanente → supprimée du layout principal
- [ ] Status bar → 1 ligne : `tokens · coût · tok/s · contexte%    ctrl+i ⓘ`
- [ ] Messages → 100% de la largeur disponible
- [ ] Hints contextuels (pas permanents sous l'input)
- [ ] Focus clavier sur le viewport (`Tab`/`Shift+Tab` pour basculer input ↔ viewport)
- [ ] `j`/`k` ou `↑`/`↓` pour scroll quand focus viewport
- [ ] **Build + tests verts**
- [ ] **Commit**

### Sprint 4 — Info Panel Overlay (P1, ~3h)

Remplace la sidebar, 0 pixel quand fermé.

- [ ] Nouveau composant `InfoPanel` (overlay droit, toggleable)
- [ ] Migrer les infos de la sidebar : session (tokens, coût, durée), providers (liste dynamique, statut), fichiers modifiés, MCP (serveurs + tools), sandbox/YOLO status
- [ ] Toggle via `Ctrl+I`
- [ ] Scrollable si le contenu dépasse le terminal
- [ ] Se ferme avec `Esc` ou `Ctrl+I`
- [ ] Supprimer l'ancien composant `sidebar.go` (594 lignes)
- [ ] **Build + tests verts**
- [ ] **Commit**

### Sprint 5 — Dialogs Modernisés (P1, ~4h)

- [ ] Tous les dialogs scrollables
- [ ] Command palette synchronisée avec le CommandRegistry complet (30+ vs 16 actuellement)
- [ ] Control Room dynamique (tous les providers, pas juste les natifs)
- [ ] Help scrollable + providers dynamiques + keybindings complètes
- [ ] Session list : fuzzy search + scrollable
- [ ] Add Provider : remplacer l'input char-par-char par Huh forms (cursor, sélection, validation)
- [ ] Évaluer remplacement de `Ctrl+S` (conflit universel "save")
- [ ] Framework de dialog commun (extraire le boilerplate `dialogStyle`, `placeDialog`, `titleStyle`)
- [ ] **Build + tests verts**
- [ ] **Commit**

### Sprint 6 — Polish (P2, ~2h)

- [ ] Thinking blocks collapsés par défaut (`▶` pour ouvrir, `▼` pour fermer)
- [ ] Tool calls compacts (one-liner quand terminés : `✓ read_file main.go · ✓ edit_file model.go`)
- [ ] Estimation coût avant cascade (`@all` → "Ce cascade coûtera ~$0.15")
- [ ] Coûts par provider/session (pas juste global), "N/A" si pricing inconnu
- [ ] Tester avec 10+ providers simulés (stress test layout et model picker)
- [ ] **Build + tests verts**
- [ ] **Commit + tag v1.0.0**

**Livrable court terme** : Poly v1.0 — TUI clean, N providers dynamiques, 0 code mort, navigation 100% clavier.

---

## Moyen Terme (Avril - Juin 2026)

> Objectif : Renforcer la proposition de valeur unique (multi-voix) et la fiabilité

### Tests et Stabilité (P0)

- [ ] Coverage providers : tests pour anthropic.go, gpt.go, gemini.go, grok.go, custom.go
- [ ] Coverage streaming : tests pour update_stream.go, streaming.go
- [ ] Coverage tools : tests pour chaque tool dans registry.go
- [ ] Coverage TUI : tests pour les interactions clavier principales
- [ ] CI/CD : GitHub Actions (build + test + lint sur chaque PR)
- [ ] Objectif : 60%+ de coverage globale (actuellement ~15% sur config/llm/permission seulement)

### Orchestrateur Multi-Voix v2 (P1)

Le chantier le plus important pour différencier Poly. L'orchestrateur `@all` actuel est basique (séquentiel, le 1er répond, les autres "reviewent").

- [ ] Mode **Séquentiel amélioré** : chaque provider reçoit les réponses précédentes en contexte
- [ ] Mode **Review croisée** : provider A répond, providers B+C critiquent, A révise
- [ ] Mode **Consensus** : tous répondent, synthèse automatique des convergences/divergences
- [ ] Modes configurables par l'utilisateur (`/cascade mode review`)
- [ ] Affichage clair de qui parle et en réponse à qui dans le fil de conversation

### Ollama First-Class (P1)

42% des devs utilisent des LLMs en local — pas un afterthought.

- [ ] Auto-détection d'Ollama (probe `localhost:11434` au démarrage)
- [ ] Liste des modèles disponibles localement dans le model picker (tag `[local]`)
- [ ] Config simplifiée : juste `"ollama": { "url": "http://localhost:11434" }` et c'est parti
- [ ] Support serveur distant (`http://192.168.1.100:11434`) pour Persona D
- [ ] Gestion gracieuse des limites (pas de crash si le modèle 7B hallucine plus qu'Opus)

### Coûts Avancés (P1)

- [ ] Token count par message (optionnel, dans l'info panel)
- [ ] Coûts par provider et par session
- [ ] Export des coûts (pour les freelances qui facturent, Persona C)
- [ ] Support providers sans pricing (local, custom) → afficher "N/A" proprement

### UX Améliorations (P2)

- [ ] Annulation streaming améliorée (Esc cancel + résumé de ce qui a été généré)
- [ ] Image support dans le chat (drag & drop ou path, pour les providers multimodaux)
- [ ] Responsive layout (pas de sidebar en dessous de 120 cols, header compact < 80 cols)
- [ ] Animations minimales (fade-in des messages, spinner streaming)

**Livrable moyen terme** : Poly v1.5 — orchestration multi-voix sérieuse, Ollama intégré, 60%+ coverage, CI/CD.

---

## Long Terme (Juillet 2026+)

> Objectif : Devenir LA référence multi-AI terminal

### Meta-Routes Intelligentes (P2)

- [ ] `@fast` → route vers le provider avec la plus basse latence
- [ ] `@cheap` → route vers le provider le moins cher par token
- [ ] `@local` → route vers tous les providers locaux
- [ ] `@best` → route vers le provider avec le meilleur score historique pour ce type de tâche
- [ ] Routing configurable par l'utilisateur (règles custom dans config)

### Thèmes et Personnalisation (P2)

- [ ] Support Catppuccin complet (Latte light mode, Frappé, Macchiato)
- [ ] Thèmes custom via config.json (couleurs de fond, accent, bordures)
- [ ] Import de thèmes (format compatible avec les éditeurs existants ?)

### Plugin System (P3)

Au-delà du MCP standard :
- [ ] Skills écrites en Go (chargement dynamique ou compilation)
- [ ] Marketplace ou registry communautaire de skills
- [ ] Hooks configurables plus riches (pre/post tool avec conditions)

### Benchmarks Personnels (P3)

Pour Persona F (l'expérimentateur) :
- [ ] Évaluation : même prompt → N providers → scoring (temps, qualité, coût)
- [ ] Historique des benchmarks par type de tâche
- [ ] Recommandation automatique ("Pour du debug Go, Claude Opus a été 2x meilleur sur tes 10 derniers tests")

### Release Publique (P2)

- [ ] README enrichi avec GIFs (VHS) et screenshots (Freeze)
- [ ] Documentation utilisateur (site ou wiki)
- [ ] `go install` depuis GitHub
- [ ] Homebrew tap (macOS)
- [ ] AUR package (Arch Linux)
- [ ] Publication sur Terminal Trove, Hacker News, Reddit r/commandline
- [ ] CONTRIBUTING.md + templates issues/PR

### Exploration (P3, non engagé)

- [ ] Mode conversation inter-IAs (les providers se parlent entre eux sans intervention humaine)
- [ ] Intégration git plus profonde (auto-commit, PR review, branch management)
- [ ] Support SSH remote (exécuter Poly localement, tools sur un serveur distant)
- [ ] Voice input (whisper.cpp local) — gimmick mais impressionnant en démo
- [ ] Web UI optionnelle (pour ceux qui veulent partager une session)

---

## Dépendances et Risques

| Risque | Impact | Mitigation |
|--------|--------|------------|
| Huh (Charm.sh) pas assez flexible | Bloque Sprint 5 (Add Provider) | Fallback sur textinput amélioré |
| Bubble Tea v2 breaking changes | Bloque tout | Verrouiller la version, surveiller les releases |
| API providers qui changent leurs formats | Bloque providers | L'abstraction custom.go + retry/backoff absorbent déjà ça |
| Pedro à 42 = temps limité | Ralentit la roadmap | Sprints courts (2-4h), autonomie max des agents |
| Competition (Crush, OpenCode évoluent vite) | Gap qui se réduit | Le multi-voix dans le même chat est unique, protéger ce différenciateur |

---

## Métriques de Succès

### v1.0 (Court terme)

| Métrique | Actuel | Cible |
|----------|--------|-------|
| Code mort TUI | ~2000 lignes | 0 |
| Espace chat (% écran) | ~75-80% | 100% (sans overlay) |
| Providers sans modification | 4 natifs hardcodés | N (illimité) |
| Valeurs hardcodées provider | 12+ | 0 |
| Dialogs scrollables | 0/7 | 7/7 |
| Navigation 100% clavier | Non | Oui |

### v1.5 (Moyen terme)

| Métrique | Actuel | Cible |
|----------|--------|-------|
| Test coverage globale | ~15% | 60%+ |
| Modes orchestration | 1 (séquentiel basique) | 3 (séquentiel, review, consensus) |
| Ollama support | Via custom provider | First-class auto-detect |
| CI/CD | Non | GitHub Actions build+test+lint |

### Long terme

| Métrique | Cible |
|----------|-------|
| GitHub stars | 500+ (signe de traction communautaire) |
| Providers testés | 20+ (natifs + custom + local) |
| Temps premier message (configuré) | < 500ms |
| Distribution | go install + homebrew + AUR |
