# Personas & Use Cases - Poly (AI Coding Terminal Multi-Provider)

> Recherche UX - 13 fevrier 2026
> Sources : Stack Overflow 2025, JetBrains 2025, Faros AI 2026, Reddit, HN, articles specialises

---

## 1. Qui utilise les AI coding terminals en 2026 ?

### Chiffres cles d'adoption

| Metrique | Valeur | Source |
|----------|--------|--------|
| Devs utilisant ou planifiant AI tools | **84%** (vs 76% en 2024) | Stack Overflow 2025 |
| Usage quotidien (pro devs) | **51%** | Stack Overflow 2025 |
| Usage hebdomadaire minimum | **65%** | Stack Overflow 2025 |
| Devs utilisant 3+ outils AI en parallele | **59%** | Stack Overflow 2025 |
| Code ecrit par AI dans les projets | **41%** | Netcorp 2026 |
| Devs utilisant au moins 1 assistant AI | **62%** | JetBrains 2025 |
| Devs executant des LLMs en local | **42%+** | Cohorte/iProyal 2026 |

### Adoption par niveau d'experience (usage quotidien)

| Niveau | Usage quotidien |
|--------|-----------------|
| Early career (< 3 ans) | **55.5%** |
| Mid career (3-10 ans) | **52.8%** |
| Pro devs (tous) | **50.6%** |
| Experienced (10+ ans) | **47.3%** |
| Apprenants | **39.5%** |

**Insight** : Les devs early/mid career sont les plus gros utilisateurs. Les seniors sont plus mefiants (20.6% "highly distrust") mais restent des utilisateurs reguliers.

### Impact sur le gain de temps

- **90%** des devs utilisant l'AI economisent au moins 1h/semaine
- **20%** economisent 8h+ (une journee entiere)

---

## 2. Les 6 Personas Principaux

### Persona A : "Le Power User CLI" (Cible primaire de Poly)

| Attribut | Description |
|----------|-------------|
| **Profil** | Dev mid/senior, 3-15 ans d'XP, confortable en terminal |
| **Stack** | Linux/macOS, tmux/zellij, vim/neovim, Git natif |
| **Outils actuels** | Aider, Claude Code, OpenCode |
| **Motivation** | Controle total, pas de GUI inutile, workflows clavier |
| **Frustrations** | Vendor lock-in, tokens bruules en hallucinations, context perdu |
| **Budget** | Paie ses propres API keys ($50-200/mois) |
| **Multi-provider** | OUI - veut choisir le modele par tache |
| **Representation** | ~15-20% des devs AI (ceux qui preferent le terminal) |

**Citation typique** : "I stopped using Copilot and didn't notice a decrease in productivity." (Reddit, cite par Faros AI)

**Ce qu'il veut de Poly** :
- Changer de provider en 2 touches
- Sessions persistantes qui survivent aux redemarrages
- Git-aware (comme Aider)
- Pas de bloat, pas de features inutiles

---

### Persona B : "L'Etudiant 42 / Bootcamp"

| Attribut | Description |
|----------|-------------|
| **Profil** | 18-30 ans, en formation intensive (42, bootcamp, etc.) |
| **Stack** | Ce qu'on lui donne (souvent C, Python, ou JS) |
| **Outils actuels** | ChatGPT gratuit, parfois Copilot etudiant |
| **Motivation** | Apprendre plus vite, comprendre du code existant |
| **Frustrations** | Pas de budget API, models gratuits limites, pas de terminal skills |
| **Budget** | $0-20/mois max |
| **Multi-provider** | Interesse par les modeles gratuits/locaux |
| **Representation** | Large (55.5% des early career sont users quotidiens) |

**Citation typique** : "68% des devs pensent que les employeurs vont bientot exiger la maitrise d'outils AI" (JetBrains 2025)

**Ce qu'il veut de Poly** :
- Fonctionne avec des modeles gratuits (Ollama, Gemini gratuit)
- Aide a COMPRENDRE, pas juste a generer
- Onboarding simple (premier message en < 30 secondes)
- Pas intimidant pour un debutant terminal

---

### Persona C : "Le Freelance Multi-Stack"

| Attribut | Description |
|----------|-------------|
| **Profil** | Dev independant, 2-8 ans d'XP, jongle entre projets/clients |
| **Stack** | Variable selon le client (React, Python, Go, PHP...) |
| **Outils actuels** | Cursor principalement, ChatGPT en backup |
| **Motivation** | Productivite maximale, basculer vite entre contextes |
| **Frustrations** | Cout cumule des abonnements ($20 Cursor + $20 ChatGPT + ...) |
| **Budget** | Sensible au cout, optimise chaque dollar |
| **Multi-provider** | OUI - utilise le modele le moins cher selon la tache |
| **Representation** | 51% des utilisateurs actifs sont dans des equipes <= 10 devs |

**Citation typique** : "Will this burn my tokens?" (preoccupation #1 selon Faros AI)

**Ce qu'il veut de Poly** :
- Routing intelligent : taches simples -> modele cheap, taches complexes -> Claude/GPT
- Suivi des couts en temps reel
- Basculer entre projets avec contexte separe
- Export des conversations (facturation client)

---

### Persona D : "Le Privacy-First / Regulated"

| Attribut | Description |
|----------|-------------|
| **Profil** | Dev en entreprise reglementee (finance, sante, gouvernement) |
| **Stack** | Souvent restrictif (reseau interne, pas d'acces cloud libre) |
| **Outils actuels** | Ollama/LM Studio en local, parfois rien |
| **Motivation** | Avoir l'AI sans envoyer le code a un tiers |
| **Frustrations** | "Ou va mon code ?" - frein principal a l'adoption |
| **Budget** | L'entreprise paie, mais procurement est lent |
| **Multi-provider** | Modeles locaux principalement, cloud approuve en option |
| **Representation** | En croissance : 42% des devs executent des LLMs en local |

**Citation typique** : "Where does my code go?" (preoccupation recurrente, Faros AI)

**Ce qu'il veut de Poly** :
- Support Ollama/local first-class (pas un afterthought)
- Aucune telemetrie, aucun appel reseau non autorise
- Configuration air-gapped possible
- Compatible avec les politiques de securite enterprise

---

### Persona E : "L'Architecte / Senior Lead"

| Attribut | Description |
|----------|-------------|
| **Profil** | Senior dev / tech lead, 10+ ans d'XP, gere une equipe |
| **Stack** | Opinions fortes sur l'architecture, review le code des autres |
| **Outils actuels** | Claude Code pour le raisonnement, Cursor pour la vitesse |
| **Motivation** | Raisonnement profond, refactoring multi-fichiers, code review |
| **Frustrations** | "Almost right but not quite" (66% des devs), outils qui comprennent pas le codebase |
| **Budget** | Pas un probleme (l'entreprise paie ou c'est un investissement) |
| **Multi-provider** | OUI - Claude pour raisonner, GPT pour generer, local pour iterer |
| **Representation** | 47.3% des seniors utilisent l'AI quotidiennement |

**Citation typique** : "It's incredibly exhausting trying to get these models to operate correctly, even when I provide extensive context." (cite par Faros AI)

**Ce qu'il veut de Poly** :
- Gros context window sans degradation
- Comprehension du repo (dependances, architecture)
- Memoire cross-session (rappeler les decisions architecturales)
- Pas de bullshit : si l'AI sait pas, qu'elle le dise

---

### Persona F : "L'Experimentateur / Early Adopter"

| Attribut | Description |
|----------|-------------|
| **Profil** | Dev curieux, teste tous les nouveaux modeles/outils |
| **Stack** | Tout et n'importe quoi, change souvent |
| **Outils actuels** | Teste Aider, Claude Code, OpenCode, Codex, Gemini CLI en parallele |
| **Motivation** | Comparer les modeles, trouver le meilleur pour chaque cas |
| **Frustrations** | Chaque outil est en silo, impossible de comparer facilement |
| **Budget** | Moyen ($50-100/mois en API keys divers) |
| **Multi-provider** | ABSOLUMENT - c'est sa raison d'etre |
| **Representation** | 59% des devs utilisent 3+ outils AI en parallele |

**Ce qu'il veut de Poly** :
- Ajouter un nouveau provider en 2 minutes
- Mode cascade : meme prompt -> N modeles -> comparer
- Benchmarks perso (quel modele a mieux repondu sur MON codebase)
- Support rapide des nouveaux modeles (jour du lancement)

---

## 3. Pourquoi Multi-Provider ?

### Les 5 raisons principales (par ordre d'importance)

#### 1. Capacites complementaires (Aucun modele n'est bon a tout)
> "No single model handles every type of task well - a model that excels at writing new functions may struggle with large projects" - JetBrains AI Blog 2026

| Tache | Meilleur modele (consensus 2026) |
|-------|----------------------------------|
| Raisonnement complexe / debug | Claude (Opus/Sonnet) |
| Generation rapide / boilerplate | GPT-5 / GPT-5.1 |
| Frontend / multimodal | Gemini |
| Code ouvert / self-hosted | DeepSeek, Qwen |
| Taches simples / cout minimal | Modeles locaux (Ollama) |

#### 2. Optimisation des couts
Le "token burn" est la preoccupation #1. Les devs veulent :
- Taches simples -> modele cheap
- Taches complexes -> modele premium
- Le ratio OpenAI/Anthropic est passe de 47:1 a 4.2:1 en 2 ans, montrant la diversification

#### 3. Vendor independence
> "Provider lock-in makes your codebase tightly coupled to a given provider's API format, and switching means rewriting everything" - Helicone 2025

Les devs ont ete brules par :
- Rate limits inattendus (Anthropic 2025)
- Changements de pricing (OpenAI GPT-5)
- Degradation de qualite percue (model plateau/decline fin 2025)

#### 4. Fiabilite / failover
Un gateway multi-provider permet :
- Basculement automatique si un provider est down
- Adoption des nouveaux modeles en < 24h au lieu de jours de refactoring
- Pas de disruption du workflow en cas de panne

#### 5. Privacy / compliance
- 42% des devs executent des LLMs en local
- Certaines entreprises INTERDISENT l'envoi de code au cloud
- Un outil multi-provider permet : local par defaut, cloud en option

---

## 4. Use Cases Concrets

### UC1 : Chat Simple (Q&A Rapide)

**Persona principal** : Tous (B, C principalement)
**Frequence** : Plusieurs fois par jour
**Flow** :
1. Dev ouvre le terminal, tape sa question
2. Reponse streamee en < 2 secondes
3. Peut copier/coller un snippet
4. Ferme ou enchaine

**Attentes** :
- Temps de demarrage < 500ms
- Premier token < 1s
- Pas besoin de contexte projet
- Markdown bien rendu dans le terminal

**Metrique de succes** : 54.1% des devs utilisent deja "mostly AI" pour chercher des reponses (vs StackOverflow)

---

### UC2 : Agentic Coding (Tool Use, Edition, Bash)

**Persona principal** : A, E (power users)
**Frequence** : Plusieurs fois par jour
**Flow** :
1. Dev decrit une tache ("ajoute une route API pour les users")
2. L'agent lit le codebase, identifie les fichiers
3. Propose des modifications (diff preview)
4. Dev approuve/modifie/refuse chaque changement
5. Agent execute (edit, create, run tests)
6. Boucle jusqu'a completion

**Attentes** :
- Permission granulaire (lire oui, ecrire demander, bash dangereux bloquer)
- Annulation a tout moment (Esc)
- Pas de modifications silencieuses
- Historique complet des actions

**Pain point critique** : "AI solutions that are almost right but not quite" (66% des devs) - le mode agentic AMPLIFIE ce probleme car les erreurs se cascadent.

---

### UC3 : Code Review / Debug

**Persona principal** : E (architecte), A (power user)
**Frequence** : Quotidienne pour les seniors
**Flow** :
1. Dev pointe vers un fichier/PR/diff
2. AI analyse et identifie les problemes
3. Suggestions categorisees (bugs, perf, security, style)
4. Dev peut demander des explications detaillees
5. Optionnel : appliquer les fixes automatiquement

**Attentes** :
- Comprendre le CONTEXTE du projet (pas juste le fichier isole)
- Ne pas flag du code correct comme bug
- Suggestions actionnables (pas juste "consider refactoring")
- 71% des devs ne mergent PAS sans review manuelle

**Stat cle** : 48% du code genere par AI contient des vulnerabilites de securite -> la review est CRITIQUE

---

### UC4 : Compare/Cascade (Meme Prompt, Plusieurs IAs)

**Persona principal** : F (experimentateur), C (freelance)
**Frequence** : Hebdomadaire a mensuelle
**Flow** :
1. Dev tape un prompt
2. Selectionne 2-4 providers
3. Meme prompt envoye en parallele
4. Reponses affichees cote-a-cote (ou sequentiellement en terminal)
5. Dev evalue et choisit la meilleure
6. Optionnel : noter/tagger les resultats

**Attentes** :
- Pas de surcharge visuelle (terminal = espace limite)
- Cout affiche avant envoi ("Ce cascade coutera ~$0.15")
- Pouvoir choisir quels providers pour ce cascade
- Historique des comparaisons

**Contexte marche** : 59% des devs utilisent 3+ outils AI -> ils font deja ce cascade MANUELLEMENT (copier-coller entre outils). L'automatiser = proposition de valeur forte.

---

### UC5 : Modeles Locaux (Ollama)

**Persona principal** : D (privacy-first), A (power user), B (etudiant sans budget)
**Frequence** : Quotidienne pour ceux qui l'adoptent
**Flow** :
1. Dev configure Ollama avec un modele (qwen2.5-coder, deepseek, etc.)
2. Dans Poly, selectionne le provider local
3. Utilise normalement (chat, code, etc.)
4. Zero donnees envoyees au cloud

**Attentes** :
- Detection automatique d'Ollama (localhost:11434)
- Liste des modeles disponibles localement
- Pas de config compliquee (Just Works)
- Fallback vers cloud si le modele local echoue/est trop lent
- Honnetete sur les limites (modele local 7B != Claude Opus)

**Stats** :
- 42% des devs executent des LLMs en local
- Cout marginal par requete ~ 0 (juste l'electricite)
- Gap de qualite qui se reduit mais reste reel

---

### UC6 : Modeles Remote (Serveur Maison, API Relay)

**Persona principal** : D (privacy), equipes
**Frequence** : Usage permanent (config une fois)
**Flow** :
1. Dev/admin configure un endpoint custom (OpenAI-compatible)
2. Dans Poly, ajoute le provider avec l'URL custom
3. Toutes les requetes passent par ce relay
4. L'admin peut logger, filtrer, rate-limiter

**Attentes** :
- Support de toute API OpenAI-compatible
- Headers custom (auth tokens, org IDs)
- Timeout configurable
- Monitoring basic (requetes/heure, tokens utilises)

**Outils existants** : LiteLLM (router open-source, 100+ providers), vLLM (serving optimise)

---

### UC7 : Session Management

**Persona principal** : A, C, E (tous les power users)
**Frequence** : Quotidienne
**Flow** :
1. Dev travaille sur une feature pendant 2h
2. Ferme le terminal / switch de projet
3. Plus tard, reprend EXACTEMENT ou il en etait
4. Optionnel : fork une session, exporter en markdown

**Attentes** :
- Sessions nommees et listables
- Persistence automatique (pas de "save" manuel)
- Compaction intelligente (garder le contexte utile, virer le bruit)
- Export en markdown/JSON pour documentation

**Etat de l'art** :
- Codex CLI : resume via fichier JSONL
- Aider : contexte via git graph compresse
- SaveContext MCP : memoire persistante cross-session
- OpenCode : multi-session avec stockage SQLite

---

### UC8 : Project Context (Instructions, Memoire)

**Persona principal** : A, E (power users sur un projet long)
**Frequence** : Config une fois, benefice permanent
**Flow** :
1. Dev cree un POLY.md (ou CLAUDE.md equivalent) a la racine du projet
2. A chaque demarrage, Poly charge ce contexte automatiquement
3. L'AI "connait" le projet : conventions, stack, decisions
4. La memoire cross-session retient les interactions precedentes

**Attentes** :
- Chargement automatique au demarrage (walk cwd -> racine)
- Format simple (markdown)
- Pas de token waste sur du contexte inutile
- Memoire qui se "compacte" intelligemment avec le temps

**Pain point** : "AI coding agents are great at remembering what you just told them... but they're not great at remembering everything you've ever told them" (Faros AI)

---

## 5. Pain Points Majeurs (Classes par Impact)

### Tier 1 - Dealbreakers (font abandonner l'outil)

| Pain Point | Impact | Stat |
|------------|--------|------|
| Code "presque correct" mais bugge | Debug plus long que coder soi-meme | 66% frustres |
| Hallucinations / code confiant mais faux | Perte de confiance totale | 46% ne font pas confiance |
| Token burn sur des echecs | Argent gaspille sans resultat | Preoccupation #1 (Faros AI) |
| Context perdu mid-session | Doit re-expliquer le projet | Pain recurrent sur Reddit/HN |

### Tier 2 - Frustrations significatives (reduisent l'usage)

| Pain Point | Impact | Stat |
|------------|--------|------|
| Degradation percue de qualite | "Les modeles stagnent ou reculent" | Sentiment favorable: 70% -> 60% |
| Vendor lock-in | Impossible de changer facilement | Rate limit Anthropic 2025 |
| Securite du code genere | 48% contient des vulnerabilites | 71% reviewent manuellement |
| Comprend pas le contexte du repo | Suggestions hors-sujet sur gros codebases | Pain architectes |

### Tier 3 - Irritants (genants mais toleres)

| Pain Point | Impact |
|------------|--------|
| Lenteur (latence reseau + gros contexte) | Casse le flow |
| UI/UX des outils CLI basique | Fonctionnel mais pas agreable |
| Pas de suivi des couts | Surprise en fin de mois |
| Onboarding complique | Nouveau user perdu |

---

## 6. Premier Contact - Attentes au First Launch

### Ce que le dev veut voir en ouvrant Poly pour la premiere fois

**Les 30 premieres secondes** (critique) :
1. L'outil demarre **instantanement** (< 500ms)
2. Un prompt clair indique qu'on peut taper
3. Le provider par defaut est deja configure (ou le setup est trivial)
4. Le premier message obtient une reponse en < 2s

**La premiere minute** :
5. Le dev comprend comment changer de provider
6. Il voit que ses fichiers projet sont accessibles
7. Il sait comment annuler/interrompre une generation

**Les 5 premieres minutes** :
8. Il a teste le tool use (edition de fichier ou bash)
9. Il comprend le systeme de permissions
10. Il sait ou trouver l'aide (/help)

### Erreurs fatales au premier contact

| Erreur | Consequence |
|--------|------------|
| Setup de 10+ minutes avant le premier message | Abandon immediat |
| Crash au demarrage | Desinstallation |
| Erreur d'API key obscure | Frustration -> concurrent |
| UI confuse (trop d'infos d'un coup) | "C'est pour les nerds" |
| Pas de feedback pendant le chargement | "C'est bloque ?" |

### Benchmark des concurrents (temps au premier message)

| Outil | Setup -> Premier message |
|-------|--------------------------|
| ChatGPT (web) | Instantane (login suffit) |
| Cursor | ~2 min (installer + login) |
| Claude Code | ~3 min (npm install + API key) |
| Aider | ~5 min (pip install + API key + config) |
| OpenCode | ~5 min (go install + API key) |

**Objectif Poly** : < 3 minutes pour le setup complet, < 30 secondes si deja configure

---

## 7. Matrice Persona x Use Case

| Use Case | A (Power CLI) | B (Etudiant) | C (Freelance) | D (Privacy) | E (Architecte) | F (Experimentateur) |
|----------|:---:|:---:|:---:|:---:|:---:|:---:|
| UC1 Chat simple | ** | *** | ** | ** | * | * |
| UC2 Agentic coding | *** | * | *** | ** | *** | ** |
| UC3 Code review | ** | * | ** | ** | *** | * |
| UC4 Compare/Cascade | ** | * | ** | - | ** | *** |
| UC5 Local (Ollama) | ** | *** | * | *** | * | ** |
| UC6 Remote custom | * | - | * | *** | * | ** |
| UC7 Session mgmt | *** | * | *** | ** | *** | ** |
| UC8 Project context | *** | * | ** | ** | *** | * |

Legende : *** = critique, ** = important, * = nice-to-have, - = pas pertinent

---

## 8. Insights Cles pour le Design de Poly

### 1. Le multi-provider est LE differenciateur
59% des devs utilisent deja 3+ outils. Poly doit rendre le switch trivial, pas juste possible.

### 2. La confiance se gagne, pas se presume
46% ne font pas confiance a l'AI. Poly doit etre TRANSPARENT : montrer les couts, montrer les limites, montrer quand l'AI ne sait pas.

### 3. Le terminal est un choix philosophique
Les devs CLI ne veulent PAS une GUI deguisee. Ils veulent : clavier-first, rapide, sans bloat, composable avec leur workflow existant.

### 4. Le budget compte enormement
Le "token burn" est la preoccupation #1. Afficher les couts, permettre le routing intelligent, supporter les modeles gratuits/locaux.

### 5. La memoire cross-session est un game-changer
C'est le pain point le plus cite : l'AI oublie tout entre les sessions. Un systeme de memoire persistante + compaction = avantage competitif.

### 6. L'onboarding doit etre fulgurant
< 3 min setup, < 30s premier message. Chaque seconde de friction = risque d'abandon.

### 7. Pedro (notre user #0) est Persona A + B + F
Etudiant 42 (B), power user terminal (A), teste tout (F). Si Poly marche pour lui, il marche pour la majorite.

---

## Sources

- [Stack Overflow Developer Survey 2025 - AI](https://survey.stackoverflow.co/2025/ai)
- [JetBrains State of Developer Ecosystem 2025](https://blog.jetbrains.com/research/2025/10/state-of-developer-ecosystem-2025/)
- [Faros AI - Best AI Coding Agents 2026](https://www.faros.ai/blog/best-ai-coding-agents-2026)
- [Claude Code vs OpenCode - Infralovers](https://www.infralovers.com/blog/2026-01-29-claude-code-vs-opencode/)
- [Top CLI Coding Agents 2026 - Pinggy](https://pinggy.io/blog/top_cli_based_ai_coding_agents/)
- [JetBrains AI Blog - Best AI Models for Coding 2026](https://blog.jetbrains.com/ai/2026/02/the-best-ai-models-for-coding-accuracy-integration-and-developer-fit/)
- [Ollama Privacy-First AI - Cohorte](https://www.cohorte.co/blog/run-llms-locally-with-ollama-privacy-first-ai-for-developers-in-2025)
- [AI Coding Agents Pain Points - Smiansh](https://www.smiansh.com/blogs/the-real-struggle-with-ai-coding-agents-and-how-to-overcome-it/)
- [Index.dev - Developer Productivity Statistics 2026](https://www.index.dev/blog/developer-productivity-statistics-with-ai-tools)
- [MIT Technology Review - AI Coding 2026](https://www.technologyreview.com/2025/12/15/1128352/rise-of-ai-coding-developers-2026/)
- [Helicone - Top LLM Gateways 2025](https://www.helicone.ai/blog/top-llm-gateways-comparison-2025)
- [IEEE Spectrum - AI Coding Assistants Failing](https://spectrum.ieee.org/ai-coding-degrades)
- [METR - AI Developer Productivity Study](https://metr.org/blog/2025-07-10-early-2025-ai-experienced-os-dev-study/)
- [Netcorp - AI-Generated Code Statistics 2026](https://www.netcorpsoftwaredevelopment.com/blog/ai-generated-code-statistics)
