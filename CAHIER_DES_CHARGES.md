# Cahier des Charges - Poly TUI Redesign

> Version 1.0 - 13 février 2026
> Rédigé à partir de 3 études parallèles : analyse concurrentielle (10 outils), personas & use cases (6 personas, 8 UC), audit UX complet (9 vues, ~7300 lignes)

---

## 1. Executive Summary

**Poly** est un terminal AI multi-provider écrit en Go (Bubble Tea v2 + Lip Gloss v2). Il permet de router des prompts vers Claude, GPT, Gemini, Grok, ou tout provider OpenAI-compatible depuis une TUI unifiée.

**Constat** : L'interface actuelle souffre de clutter (sidebar permanente de 24 colonnes, header de 2 lignes, ~1400 lignes de code mort), de valeurs hardcodées (4 providers, 11 variants, couleurs fixes), et de frictions UX (pas de focus clavier sur le chat, inputs faits main, dialogs non scrollables).

**Objectif** : Moderniser la TUI pour qu'elle soit épurée, scalable (N providers), et alignée avec les tendances du marché — sans changer l'identité du produit.

**Positionnement unique** : Aucun concurrent n'offre la combinaison Go + Bubble Tea + MIT + multi-provider + TUI soignée. Crush (Charm.sh) a la beauté mais une licence restrictive. OpenCode a le multi-provider mais en TypeScript. Aider a 100+ modèles mais zéro TUI.

---

## 2. Contexte et Objectifs

### 2.1 Le marché en 2026

- **84%** des développeurs utilisent ou planifient d'utiliser des outils AI (Stack Overflow 2025)
- **51%** des devs pro utilisent l'AI quotidiennement
- **59%** utilisent 3+ outils AI en parallèle — le multi-provider est déjà une réalité
- **42%** exécutent des LLMs en local (Ollama, llama.cpp)
- **46%** ne font PAS confiance au output AI — la transparence est critique

### 2.2 Ce que Poly EST

- Un **chat terminal** : tu tapes un message, une IA répond, elle peut utiliser des tools
- Un **router multi-AI dans le même chat** : chaque message peut être adressé à un provider différent (`@claude` puis `@gemini` puis `@gpt`), et ils partagent tous le **même historique de conversation**. C'est une conversation multi-voix, pas des sessions séparées.
- Exemple concret :
  ```
  [You @claude]    Écris une fonction de tri en Go
  [claude]         Voici une implémentation avec sort.Slice...
  [You @gemini]    Revois ce code, tu vois des problèmes ?
  [gemini]         Le code de Claude a un edge case sur les slices nil...
  [You @gpt]       Propose une alternative plus performante
  [gpt]            Voici une version utilisant un heap sort...
  ```
- Le mode `@all` (cascade) envoie le même prompt à tous les providers séquentiellement. **L'orchestrateur actuel est basique et à revoir** : il manque un vrai système pour que les IAs interagissent entre elles (review croisée, consensus, débat).
- Poly n'est **PAS** du multi-AI parallèle côte-à-côte (pas de split-screen), ni un dashboard, ni un IDE. C'est un **unique fil de conversation** où plusieurs IAs peuvent contribuer.

### 2.3 Objectifs du redesign

| # | Objectif | Mesure de succès |
|---|----------|-----------------|
| O1 | Supprimer le clutter visuel | Sidebar permanente supprimée, header réduit à 1 ligne |
| O2 | Messages = 100% de la largeur disponible | Le chat prend tout l'écran |
| O3 | Scalable pour N providers | Zéro hardcode de provider, couleurs auto-assignées |
| O4 | Dialogs modernisés | Overlays centrés, scrollables, avec Huh forms |
| O5 | Code mort supprimé | ~1400 lignes de composants dupliqués retirés |
| O6 | Onboarding < 3 minutes | Premier message en < 30 secondes si déjà configuré |
| O7 | Suivre les tendances, pas réinventer | S'inspirer de ce qui marche (Claude Code, OpenCode, Lazygit) |

---

## 3. Analyse de l'Existant

### 3.1 Architecture TUI actuelle

```
┌─ Header (2 lignes) ─────────────────────────────────────┐
│ ◇ POLY  cwd               ─────── 32% · ctrl+d providers│
├─────────────────────────────────────┬─ Sidebar (24 col) ─┤
│                                     │ Active model       │
│   Chat (viewport)                   │ YOLO badge         │
│   Messages scrollables              │ Modified files     │
│                                     │ Providers status   │
│                                     │ Todos              │
├─────────────────────────────────────┴────────────────────┤
│ ╭─ Input (1-5 lignes, bordure arrondie) ──────────────╮  │
│ │ ● message...                                        │  │
│ ╰─────────────────────────────────────────────────────╯  │
│ enter send · shift+enter newline · ctrl+k · @provider    │
├──────────────────────────────────────────────────────────┤
│ Status bar (streaming speed │ tokens+cost │ provider)    │
└──────────────────────────────────────────────────────────┘
```

**Espace perdu** : La sidebar occupe 24 colonnes en permanence (~15-20% de l'écran) pour des infos utiles 5% du temps.

### 3.2 Les 9 vues existantes

| Vue | Accès | Lignes | Problèmes |
|-----|-------|--------|-----------|
| **Splash** | Démarrage | ~200 | Version hardcodée "v0.2.0", providers hardcodés (4 natifs seulement) |
| **Chat** | Vue principale | ~600 | Pas de focus clavier viewport, welcome incohérent ("dashboard" vs "Control Room") |
| **Model Picker** | Ctrl+O | 152 | `variantOrder` hardcodé (11 variants), `strings.Title()` déprécié |
| **Control Room** | Ctrl+D | 155 | Input auth fait main (pas de cursor), masquage API key incomplet, custom providers invisibles |
| **Add Provider** | "N" dans Control Room | 96 | Input char-par-char sans cursor/selection/copier-coller |
| **Command Palette** | Ctrl+K | 149 | Seulement 16 entrées sur 30+ commandes, noms inconsistants |
| **Help** | Ctrl+H | 91 | Providers hardcodés, keybindings incomplets, PAS scrollable |
| **Approval** | Auto (tool use) | 314 | Bonne UX globalement, manque "Allow for this tool only" |
| **Session List** | Ctrl+S | 248 | Ctrl+S = "save" partout ailleurs, pas de recherche, pas scrollable |

### 3.3 Code mort identifié (~1400 lignes)

| Chemin | Lignes | Raison |
|--------|--------|--------|
| `components/dialogs/help/` | ~125 | Jamais importé, doublon de `dialogs.go:renderHelp()` |
| `components/dialogs/controlroom/` | ~321 | Jamais importé |
| `components/dialogs/modelpicker/` | ~235 | Jamais importé |
| `components/dialogs/addprovider/` | ~195 | Jamais importé |
| `components/dialogs/palette/` | ~178 | Jamais importé |
| `components/dialogs/approval/` | ~280 | Jamais importé |
| `components/dialogs/sessionlist/` | ~230 | Jamais importé |
| `components/messages/user.go` | ~127 | Jamais importé |
| `components/messages/assistant.go` | ~189 | Jamais importé |
| `components/messages/messages.go` | ~8 | Interface jamais utilisée |
| `sidebar.getCachedMCP()` | ~30 | Méthode jamais appelée dans View() |

### 3.4 Valeurs hardcodées critiques

| Valeur | Emplacement | Impact |
|--------|-------------|--------|
| `order := []string{"claude","gpt","gemini","grok"}` | config.go:182 | Custom providers ignorés dans l'ordre |
| `variantOrder := []string{"default","fast","nano",...}` | model.go:189 | Doit être mis à jour manuellement par modèle |
| `"v0.2.0"` | splash.go:38 | Jamais synchronisé avec git version |
| `DefaultContextWindow = 200_000` | layout/constants.go | Pas adapté au provider/modèle (Gemini = 1M) |
| `SidebarWidth: 24` | layout/constants.go | Fixe, pas responsive |
| `maxContextFiles = 10` | commands.go:645 | Arbitraire, pas configurable |
| Dialog widths (62, 52, 54, 46, 40-80) | Chaque dialog | Pas centralisé |
| Input history max `100` | update_keys.go | Incohérent avec `config/history.go` (500) |

### 3.5 Frictions UX majeures

1. **Deux systèmes de commandes concurrents** : Command palette (16 entrées) vs CommandRegistry (30+), pas synchronisés
2. **Pas de focus clavier** : Impossible de naviguer le chat au clavier, seulement à la souris
3. **Custom providers invisibles** : N'apparaissent pas dans splash, sidebar, help, ni Control Room badges
4. **Inputs faits main** : Add Provider et auth input = implémentation char-par-char sans cursor/sélection/undo
5. **Dialog stack inutilisé** : Un système complet de dialog stack (`components/dialogs/`) existe mais est bypassé par le `viewState`
6. **Duplication** : `formatTokens()` x2, provider lists x4, `session.Clear()` vs `SetMessages()` inconsistant
7. **Code dupliqué** : Chaque dialog réimplémente `dialogStyle()`, `placeDialog()`, `titleStyle`

---

## 4. Analyse Concurrentielle

### 4.1 Benchmark des 10 concurrents

| Outil | Stars | Stack | Multi-Provider | TUI Riche | Licence |
|-------|-------|-------|:--------------:|:---------:|---------|
| **OpenCode** | 104k | TypeScript | ✅ 75+ | ✅ | MIT |
| **Gemini CLI** | 94k | TypeScript | ❌ | ❌ | Apache-2.0 |
| **Claude Code** | 67k | Shell/Python/TS | ❌ | ❌ | Propriétaire |
| **Codex CLI** | 60k | Rust | ❌ | ❌ | Apache-2.0 |
| **Cline** | 58k | TypeScript | ✅ | N/A (VS Code) | Apache-2.0 |
| **Aider** | 41k | Python | ✅ 100+ | ❌ | Apache-2.0 |
| **Roo Code** | 22k | TypeScript | ✅ | N/A (VS Code) | Apache-2.0 |
| **Crush** | 20k | **Go** | ✅ | ✅ | **Charm License** |
| **Kiro** | 3k | TypeScript | Partiel | ❌ | Propriétaire |
| **amux** | 29 | **Go** | N/A (orchestrateur) | ✅ | MIT |

### 4.2 Patterns UX communs (baseline obligatoire)

Ce que **tous** les outils font et que Poly doit impérativement avoir :

1. Chat conversationnel (input en bas, messages scrollables)
2. MCP support (devenu standard en 2026)
3. File read/write/edit (outils de base)
4. Shell execution avec permissions
5. Git awareness
6. Context files (POLY.md / CLAUDE.md / équivalent)
7. Streaming responses avec affichage progressif
8. Markdown rendering dans le terminal

### 4.3 Différenciateurs clés par outil

| Différenciateur | Outil | Leçon pour Poly |
|----------------|-------|-----------------|
| Autonomie maximale | Claude Code | La boucle agentique doit être robuste |
| Multi-provider champion | OpenCode (75+), Aider (100+) | Le switch de provider doit être trivial |
| Free tier généreux | Gemini CLI (1000 req/jour gratuit) | Supporter les tiers gratuits et locaux |
| Plus belle TUI | Crush (Charm stack) | Poly utilise la même stack — on peut rivaliser |
| Plus rapide (Rust) | Codex CLI | Go est rapide aussi, optimiser le démarrage |
| Deep reasoning | Amp (Deep mode) | Supporter les modes thinking/extended |
| Modes spécialisés | Roo Code | Les skills Poly peuvent remplir ce rôle |

### 4.4 Le gap que Poly comble

**Aucun concurrent n'offre** : Go + Bubble Tea + MIT + multi-provider **dans le même chat** + TUI soignée.

- Crush a la beauté Go/Bubble Tea mais Charm License (restrictions commerciales)
- OpenCode a le multi-provider 75+ mais en TypeScript, et c'est du switch de provider, pas du multi-voix dans le même fil
- Aider a 100+ modèles mais zéro TUI (CLI brut), et c'est un provider à la fois
- Claude Code est le plus autonome mais lock-in Anthropic, zéro TUI

**Le vrai différenciateur de Poly** : un unique fil de conversation où `@claude` répond au message 1, `@gemini` critique au message 2, et `@gpt` propose une alternative au message 3. Aucun concurrent ne fait ça — ils switchent de provider, Poly les fait **collaborer dans le même chat**.

### 4.5 Tendances 2026

1. **Multi-provider = MUST** — les devs refusent le lock-in
2. **Agents autonomes > pair programming** — exécuter, pas juste suggérer
3. **MCP est le standard** — 100% des outils le supportent
4. **TUI riche vs CLI minimaliste** — la bataille fait rage
5. **Persistent memory** — mémoriser les patterns entre sessions
6. **Transparence des coûts** — afficher en temps réel
7. **Modèles locaux en croissance** — 42% des devs, tendance forte

---

## 5. Personas et Use Cases

### 5.1 Les 6 personas

| Persona | Profil | Besoin principal | Multi-Provider | Budget |
|---------|--------|-----------------|:--------------:|--------|
| **A. Power User CLI** | Dev mid/senior, terminal-native | Contrôle total, workflows clavier | ✅ Par tâche | $50-200/mois |
| **B. Étudiant 42/Bootcamp** | 18-30 ans, en formation | Apprendre, comprendre le code | Gratuits/locaux | $0-20/mois |
| **C. Freelance Multi-Stack** | Dev indépendant, jongle entre projets | Productivité, optimiser les coûts | ✅ Le moins cher | Sensible |
| **D. Privacy-First** | Entreprise réglementée | Zero data au cloud | Local obligatoire | L'entreprise paie |
| **E. Architecte** | Senior/Lead, 10+ ans | Raisonnement profond, review | ✅ Par capacité | Pas un problème |
| **F. Expérimentateur** | Curieux, teste tout | Comparer les modèles | ✅ ABSOLUMENT | $50-100/mois |

**Cible primaire** : Personas A et F (power users multi-provider).
**Cible secondaire** : Personas B et C (budget-conscious, besoin de locaux/gratuits).
**Cible tertiaire** : Personas D et E (enterprise/privacy, architecture).

### 5.2 Les 8 use cases

| UC | Description | Personas | Fréquence | Priorité |
|----|-------------|----------|-----------|----------|
| **UC1** | Chat simple (Q&A rapide) | Tous | Quotidienne | P0 |
| **UC2** | Agentic coding (tool use, édition, bash) | A, E | Quotidienne | P0 |
| **UC3** | Code review / debug | A, E | Quotidienne | P1 |
| **UC4** | Compare/Cascade (même prompt, N IAs) | F, C | Hebdomadaire | P1 |
| **UC5** | Modèles locaux (Ollama) | B, D | Quotidienne | P1 |
| **UC6** | Modèles remote (serveur maison, API relay) | D | Config ponctuelle | P2 |
| **UC7** | Session management (reprendre, fork, exporter) | A, C, E | Quotidienne | P1 |
| **UC8** | Project context (POLY.md, MEMORY.md, skills) | A, E | Config ponctuelle | P1 |

### 5.3 Pain points critiques (dealbreakers)

| Pain Point | Impact | Stat | Réponse Poly |
|------------|--------|------|-------------|
| Code "presque correct" mais buggé | Debug plus long que coder soi-même | 66% frustrés | Approval granulaire, diff preview |
| Hallucinations confiantes | Perte de confiance totale | 46% ne font pas confiance | Transparence, mode thinking visible |
| Token burn sur des échecs | Argent gaspillé | Préoccupation #1 | Coûts affichés en temps réel |
| Context perdu mid-session | Re-expliquer le projet | Pain récurrent | MEMORY.md + compaction intelligente |
| Vendor lock-in | Impossible de changer | Rate limits inattendus | Multi-provider natif |

---

## 6. Exigences Fonctionnelles

### 6.1 Chat (P0 - Indispensable)

| # | Exigence | Actuel | Cible |
|---|----------|--------|-------|
| F1.1 | Messages prennent 100% de la largeur disponible | 75-85% (sidebar mange 24 col) | 100% (sidebar supprimée) |
| F1.2 | Focus clavier sur le viewport (scroll au clavier) | ❌ Souris uniquement | ✅ Tab/Shift+Tab pour basculer focus |
| F1.3 | Markdown rendu proprement | ✅ Glamour | ✅ Maintenir |
| F1.4 | Tool calls affichés inline avec collapse | ✅ Basique | ✅ Améliorer (collapse par défaut quand terminé) |
| F1.5 | Thinking block collapsable par défaut | ❌ Toujours visible | ✅ Collapsé par défaut, clic pour ouvrir |
| F1.6 | Streaming avec tok/s en temps réel | ✅ | ✅ Maintenir |

### 6.2 Provider Management (P0)

| # | Exigence | Actuel | Cible |
|---|----------|--------|-------|
| F2.1 | N providers dynamiques (pas de limite hardcodée) | 4 natifs hardcodés + custom | N providers, zéro hardcode |
| F2.2 | Couleurs auto-assignées depuis palette | 4 couleurs fixes + fallback gris | Palette Catppuccin cyclique (12 couleurs) |
| F2.3 | Ordre des providers dynamique | `["claude","gpt","gemini","grok"]` fixe | Alphabétique ou par usage récent |
| F2.4 | Model picker avec fuzzy search | ✅ Basique | ✅ + tags [local]/[api]/[remote], context window affiché |
| F2.5 | Variants dynamiques depuis config | 11 variants hardcodées | Lus depuis la config provider |
| F2.6 | Custom providers visibles partout | ❌ Invisibles dans splash/help/sidebar | ✅ Traités identiquement aux natifs |
| F2.7 | Context window adapté au provider/modèle | 200k fixe | Lu depuis la config (Gemini=1M, local=32k, etc.) |

### 6.3 Layout (P0)

| # | Exigence | Actuel | Cible |
|---|----------|--------|-------|
| F3.1 | Header réduit à 1 ligne | 2 lignes (logo + séparateur + hints) | 1 ligne : provider actif + variant + contexte% + cwd |
| F3.2 | Sidebar permanente supprimée | 24 colonnes fixes | Remplacée par un overlay toggleable (raccourci) |
| F3.3 | Status bar compacte (1 ligne) | 2 lignes (bordure + bar) | 1 ligne : tokens + coût + vitesse + provider |
| F3.4 | Hints contextuels, pas permanents | Toujours affichés sous l'input | Affichés seulement quand pertinents (input vide > 5s, premier lancement) |
| F3.5 | Info panel en overlay | N/A | Toggle avec raccourci : tokens, providers, files modifiés, MCP status |

### 6.4 Multi-Voix et Orchestration (P1)

| # | Exigence | Actuel | Cible |
|---|----------|--------|-------|
| F4A.1 | Chat multi-voix : chaque message peut cibler un provider différent | ✅ Fonctionne (@claude, @gemini, @gpt dans le même chat) | ✅ Maintenir et renforcer visuellement |
| F4A.2 | Historique partagé : chaque provider voit les messages de tous les autres | ✅ | ✅ Maintenir |
| F4A.3 | Identité visuelle par provider dans le fil | ✅ Bordure gauche colorée | ✅ Maintenir + couleurs auto-assignées |
| F4A.4 | `@all` cascade améliorée | Séquentiel basique (le 1er répond, les autres reviewent) | À repenser : meilleur orchestrateur (consensus, débat, vote, review croisée) |
| F4A.5 | Le dernier provider utilisé devient le défaut pour le prochain message | ✅ | ✅ Maintenir |
| F4A.6 | Meta-routes intelligentes | ❌ | `@fast` (le plus rapide), `@cheap` (le moins cher), `@local` (tous les locaux) — long terme |
| F4A.7 | Orchestrateur configurable | ❌ Cascade fixe | Modes d'orchestration : séquentiel, review croisée, consensus — long terme |

**Note** : L'orchestration (`@all` et au-delà) est le cœur de la proposition de valeur unique de Poly. L'orchestrateur actuel est basique et devra évoluer significativement, mais c'est un chantier à part entière, pas du redesign TUI.

### 6.5 Dialogs (P1)

| # | Exigence | Actuel | Cible |
|---|----------|--------|-------|
| F4.1 | Dialogs en overlay centré | ✅ Déjà centré | ✅ Maintenir + rendre scrollables |
| F4.2 | Command palette complète | 16/30+ commandes | Synchronisée avec le CommandRegistry complet |
| F4.3 | Add Provider avec Huh forms | Input char-par-char sans cursor | Formulaire interactif (Huh) avec validation |
| F4.4 | Control Room dynamique | Providers hardcodés, statique | Liste dynamique de tous les providers (natifs + custom) |
| F4.5 | Help scrollable et complet | Non scrollable, keybindings incomplets | Scrollable, providers dynamiques, toutes les keybindings |
| F4.6 | Session list avec recherche | Pas de filtre, pas scrollable | Fuzzy search + scrollable |
| F4.7 | Keybinding Ctrl+S → autre chose | Ctrl+S = sessions (conflit "save") | Évaluer un autre raccourci |

### 6.5 Coûts et Transparence (P1)

| # | Exigence | Actuel | Cible |
|---|----------|--------|-------|
| F5.1 | Coûts affichés en temps réel par provider | Global seulement | Par provider/session, "N/A" si pricing inconnu |
| F5.2 | Token count par message | ❌ | ✅ Optionnel, visible dans l'overlay info |
| F5.3 | Estimation avant envoi pour cascade | ❌ | ✅ "Ce cascade coûtera ~$0.15" |
| F5.4 | Support providers sans pricing (local, custom) | ❌ Coût hardcodé | ✅ Graceful "N/A" |

### 6.6 Code Cleanup (P0)

| # | Exigence | Action |
|---|----------|--------|
| F6.1 | Supprimer `components/dialogs/` (7 sous-dossiers) | ~1200 lignes mortes |
| F6.2 | Supprimer `components/messages/user.go`, `assistant.go`, `messages.go` | ~325 lignes mortes |
| F6.3 | Supprimer `sidebar.getCachedMCP()` | Code mort |
| F6.4 | Unifier `formatTokens()` | Dupliqué dans sidebar.go et status.go |
| F6.5 | Unifier `session.Clear()` → `session.SetMessages()` | Incohérent |
| F6.6 | Extraire un dialog framework commun | Chaque dialog réimplémente le même boilerplate |
| F6.7 | Remplacer `strings.Title()` (déprécié) | Utilisé dans modelpicker.go et update_keys.go |

---

## 7. Exigences Non-Fonctionnelles

### 7.1 Performance

| # | Exigence | Cible |
|---|----------|-------|
| NF1.1 | Temps de démarrage (cold start) | < 500ms |
| NF1.2 | Premier token affiché | < 1s après envoi |
| NF1.3 | Rendering FPS (scroll, streaming) | 60 FPS stable |
| NF1.4 | Mémoire au repos | < 30 MB |
| NF1.5 | Mémoire en streaming | < 100 MB |

### 7.2 Scalabilité

| # | Exigence | Cible |
|---|----------|-------|
| NF2.1 | Nombre de providers | Illimité (N), testé avec 10+ |
| NF2.2 | Nombre de modèles dans le picker | Testé avec 50+, fuzzy search performant |
| NF2.3 | Taille de l'historique de session | 1000+ messages avec compaction |
| NF2.4 | Nombre de sessions sauvegardées | Illimité, avec recherche |

### 7.3 Compatibilité

| # | Exigence | Cible |
|---|----------|-------|
| NF3.1 | Terminaux supportés | Kitty, Alacritty, WezTerm, iTerm2, GNOME Terminal, Windows Terminal |
| NF3.2 | OS | Linux, macOS. Windows best-effort |
| NF3.3 | Taille minimum du terminal | 80x24 (dégradation gracieuse) |
| NF3.4 | Go version minimum | 1.22+ (pour la compatibilité 42 via setup-42.sh) |

### 7.4 Accessibilité

| # | Exigence | Cible |
|---|----------|-------|
| NF4.1 | Navigation 100% clavier | Toutes les fonctionnalités accessibles sans souris |
| NF4.2 | Pas de dépendance aux couleurs seules | Toujours combiner couleur + icône/texte |
| NF4.3 | Responsive aux petits terminaux | Layout adaptatif (pas de sidebar < 120 cols, etc.) |

---

## 8. Architecture UI Proposée

### 8.1 Layout cible

```
┌─────────────────────────────────────────────────────────┐
│ ◇ POLY │ @claude/think │ 12% │ ~/PROJETS-AI/Poly-go    │  ← 1 ligne
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┃ You                                                  │
│  ┃ Explique le fichier main.go                          │
│                                                         │
│  ┃ claude ──────────────────────────────────             │
│  ┃ Le fichier main.go est le point d'entrée...          │
│  ┃                                                      │
│  ┃ ▶ Thinking (cliquer pour ouvrir)                     │  ← Collapsé par défaut
│  ┃                                                      │
│  ┃ ```go                                                │
│  ┃ func main() {                                        │
│  ┃ ```                                                  │
│  ┃                                                      │
│  ┃ ✓ read_file main.go · ✓ edit_file model.go           │  ← Tools compacts
│                                                         │
│                    (100% largeur)                        │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ ╭───────────────────────────────────────────────────╮   │
│ │ ● message...                                      │   │
│ ╰───────────────────────────────────────────────────╯   │
│ 3.2k tok · $0.02 · 45 tok/s · 12%              ctrl+i ⓘ│  ← 1 ligne
└─────────────────────────────────────────────────────────┘
```

**Gain** : ~24 colonnes récupérées (sidebar) + 2 lignes (header réduit + status compacté) = significativement plus d'espace pour le contenu.

### 8.2 Info Panel (overlay, remplace la sidebar)

Accessible via `Ctrl+I` (ou autre raccourci à définir) :

```
╭─ Info ─────────────────────────╮
│                                │
│ Session                        │
│   Tokens: 3.2k in / 1.8k out  │
│   Coût:   $0.02               │
│   Durée:  4m 12s               │
│                                │
│ Providers (3 connectés)        │
│   ● claude   opus-4.6          │
│   ● gpt      gpt-5.1-turbo    │
│   ● ollama   qwen3:72b [local] │
│   ○ gemini   (pas de clé)     │
│                                │
│ Fichiers modifiés (2)          │
│   main.go        +12 -3       │
│   model.go       +45 -20      │
│                                │
│ MCP (2 serveurs, 14 tools)     │
│   pedro-tracker  ✓ connecté   │
│   ai-bridge      ✓ connecté   │
│                                │
│ Sandbox: OFF · YOLO: OFF       │
╰────────────────────────────────╯
```

Se ferme avec `Esc` ou `Ctrl+I`. Prend 0 pixel quand fermé.

### 8.3 Composants à refactorer

| Composant | Avant | Après |
|-----------|-------|-------|
| **Header** | 2 lignes, provider/thinking non affichés malgré les setters | 1 ligne, tout visible |
| **Sidebar** | 24 col permanentes, 594 lignes | Supprimée → Info Panel overlay |
| **Status Bar** | 2 lignes (bordure + contenu) | 1 ligne compacte |
| **Input** | Bordure arrondie + hints permanents | Bordure arrondie + hints contextuels |
| **Dialogs** | 7 implémentations indépendantes | Framework commun + Huh forms pour Add Provider |
| **Messages** | Bordure gauche épaisse, thinking toujours visible | Idem + thinking collapsé par défaut, tools plus compacts |

### 8.4 Navigation clavier

| Raccourci | Action | Changement |
|-----------|--------|------------|
| `Tab` / `Shift+Tab` | Basculer focus input ↔ viewport | **NOUVEAU** |
| `j/k` ou `↑/↓` | Scroll messages (quand focus viewport) | **NOUVEAU** |
| `Ctrl+I` | Toggle info panel (overlay) | **NOUVEAU** (remplace sidebar) |
| `Ctrl+O` | Model picker | Maintenu |
| `Ctrl+D` | Control Room | Maintenu |
| `Ctrl+K` | Command palette (complète) | Amélioré (30+ commandes) |
| `Ctrl+H` | Help (scrollable) | Amélioré |
| `Ctrl+N` | Nouvelle session | Maintenu |
| `Esc` | Fermer dialog / annuler streaming | Maintenu |

---

## 9. Design System

### 9.1 Thème

- **Défaut** : Catppuccin Mocha (identité Poly)
- **Accent principal** : Mauve (#cba6f7)
- **Extensible** : Support d'autres variantes Catppuccin (Latte pour light mode, Frappé, Macchiato)
- **Custom** : Possibilité de thèmes custom via config (long terme)

### 9.2 Couleurs providers (auto-assignation)

Palette cyclique Catppuccin Mocha (12 couleurs) :

| Index | Couleur | Hex |
|-------|---------|-----|
| 0 | Mauve | #cba6f7 |
| 1 | Blue | #89b4fa |
| 2 | Green | #a6e3a1 |
| 3 | Peach | #fab387 |
| 4 | Pink | #f5c2e7 |
| 5 | Teal | #94e2d5 |
| 6 | Yellow | #f9e2af |
| 7 | Red | #f38ba8 |
| 8 | Flamingo | #f2cdcd |
| 9 | Rosewater | #f5e0dc |
| 10 | Sky | #89dceb |
| 11 | Lavender | #b4befe |

Assignation : par ordre d'ajout dans la config. Override possible via `provider_colors` dans config.json.

### 9.3 Typographie terminal

- **Titres** : Bold + couleur accent
- **Contenu** : Normal
- **Dimmed** : Overlay0/Overlay1 pour info secondaire
- **Code** : Rendu par Glamour avec syntax highlighting
- **Bordures** : Arrondies (RoundedBorder) pour les dialogs/input, épaisses (ThickBorder) pour les messages

### 9.4 Icônes

| Icône | Usage |
|-------|-------|
| ◇ | Logo Poly (header) |
| ● / ○ | Provider connecté / déconnecté |
| ✓ | Tool terminé avec succès |
| ✗ | Tool en erreur |
| ⟳ | Tool en cours |
| ▶ | Thinking block collapsé |
| ▼ | Thinking block ouvert |

---

## 10. Roadmap de Migration

### Phase 1 : Nettoyage (P0, ~2h)

- [ ] Supprimer les ~1400 lignes de code mort (`components/dialogs/`, messages dead code)
- [ ] Unifier `formatTokens()`, `session.Clear()` → `SetMessages()`
- [ ] Remplacer `strings.Title()` déprécié
- [ ] Centraliser les constantes de dialog (largeurs, styles communs)
- [ ] Vérifier que le build passe

### Phase 2 : Layout (P0, ~4h)

- [ ] Réduire le header à 1 ligne (afficher provider actif + variant + contexte% + cwd)
- [ ] Supprimer la sidebar permanente du layout
- [ ] Compacter la status bar à 1 ligne
- [ ] Recalculer le layout : messages = 100% largeur
- [ ] Rendre les hints contextuels (pas permanents)
- [ ] Ajouter focus clavier sur le viewport (Tab/Shift+Tab)

### Phase 3 : Provider Dynamique (P0, ~3h)

- [ ] Supprimer `providerOrder` hardcodé → dynamique depuis config
- [ ] Supprimer `variantOrder` hardcodé → dynamique depuis config
- [ ] Auto-assigner les couleurs depuis la palette Catppuccin cyclique
- [ ] `DefaultContextWindow` → lu depuis la config provider
- [ ] Custom providers visibles dans splash, help, Control Room, partout
- [ ] Splash dynamique (lister TOUS les providers configurés)

### Phase 4 : Info Panel Overlay (P1, ~3h)

- [ ] Créer le composant Info Panel (overlay droit)
- [ ] Y migrer les infos de la sidebar : tokens, providers, fichiers, MCP
- [ ] Toggle via `Ctrl+I`
- [ ] Scrollable si le contenu dépasse
- [ ] Supprimer l'ancien composant sidebar

### Phase 5 : Dialogs Modernisés (P1, ~4h)

- [ ] Rendre tous les dialogs scrollables
- [ ] Synchroniser la command palette avec le CommandRegistry complet
- [ ] Remplacer l'input Add Provider par Huh forms
- [ ] Rendre le Control Room dynamique (tous les providers)
- [ ] Help : providers dynamiques, keybindings complètes
- [ ] Session list : ajouter fuzzy search
- [ ] Évaluer remplacement de Ctrl+S

### Phase 6 : Polish (P2, ~2h)

- [ ] Thinking blocks collapsés par défaut (▶ pour ouvrir)
- [ ] Tool calls plus compacts (one-liner quand terminés)
- [ ] Version dans splash synchronisée avec git tag
- [ ] Estimation coût avant cascade
- [ ] Tester avec 10+ providers simulés

---

## 11. Métriques de Succès

| Métrique | Actuel | Cible | Comment mesurer |
|----------|--------|-------|----------------|
| Lignes de code TUI | ~7300 (dont ~1400 mortes) | ~5500 (-25%) | `wc -l` |
| Espace chat (% écran) | ~75-80% | 100% (sans overlay) | Mesure visuelle |
| Providers supportés sans modification | 4 natifs | N (illimité) | Tester avec 10+ |
| Temps premier message (déjà configuré) | ~1-2s | < 1s | Benchmark |
| Valeurs hardcodées provider | 12+ | 0 | Grep dans le code |
| Dialogs scrollables | 0/7 | 7/7 | Test manuel |
| Navigation 100% clavier | ❌ (pas de focus viewport) | ✅ | Test manuel |

---

## Annexes

### A. Sources

**Analyse concurrentielle :**
- GitHub repositories de chaque outil (stars, forks, contributors au 13/02/2026)
- Hacker News discussions (OpenCode, Claude Code, Aider)
- Builder.io, Pinggy, Kevnu comparisons

**Personas et use cases :**
- Stack Overflow Developer Survey 2025
- JetBrains State of Developer Ecosystem 2025
- Faros AI - Best AI Coding Agents 2026
- IEEE Spectrum - AI Coding Assistants
- METR - AI Developer Productivity Study

**Audit UX :**
- Lecture directe du code source Poly-Go (131 fichiers, ~24K lignes)

### B. Outils de référence design

| Outil | Ce qu'on en prend |
|-------|-------------------|
| [Lazygit](https://github.com/jesseduffield/lazygit) | Overlays, focus zones, keyboard-first |
| [OpenCode](https://opencode.ai) | Dialog overlay system, multi-provider UX |
| [Sidecar](https://github.com/marcus/sidecar) | Plugin tabs, split-pane, 453 themes |
| [parllama](https://github.com/paulrobello/parllama) | Multi-provider dropdown, session config collapsible |
| [amux](https://github.com/andyrewlee/amux) | Orchestration multi-agents parallèles (inspiration future) |
| [Crush](https://github.com/charmbracelet/crush) | Référence esthétique TUI Go (même stack) |

### C. Plugins design installés

| Plugin | Type | Usage |
|--------|------|-------|
| frontend-design | Claude Code Plugin | Principes de design (typo, couleurs, animations) |
| Figma MCP | MCP Server (HTTP) | Import de maquettes Figma |
| Superdesign MCP | MCP Server (stdio) | Génération de specs design structurées |
| VHS | CLI (Go) | Enregistrement terminal → GIF |
| Freeze | CLI (Go) | Screenshots terminal |
