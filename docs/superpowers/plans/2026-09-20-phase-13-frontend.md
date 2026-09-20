# Postulator v2 — Phase 13: the product frontend

The plan for the orchestrated frontend rebuild on `rewrite/v2`. Orchestrator: this session (Fable). Workers: up to eight Opus agents in total, at most two at a time. Each big package is planned by its worker first, approved by the orchestrator (and the user when asked), then coded with TDD. Approved by the user on 2026-09-20.

## 1. Context

The Go backend (phases 0–12) is complete and gate-green. The frontend (~39k lines of TS/TSX) was written in ~25 hours on 2026-09-19/20 by four different model sessions. Four explore agents audited it on 2026-09-20 and agreed:

- **Genuinely good, keep as is:** `frontend/src/data/*` (typed endpoints, lock-aware queries, coalesced event bridge, error→reaction pipeline, run-event replay), `frontend/src/ui/theme.css` (matches `design/Postulator.dc.html` token for token), the single-copy-module discipline, `frontend/src/canvas/*` and `features/graph/model/*`, the Linking screen, the runs derivation layer, 51 vitest suites over pure modules.
- **Broken above that layer:** five competing navigation affordances; a 12-icon rail with no sections; picking a site lands on a "not built" stub that shows "wave 3, agent 7" to the client; `/onboarding` unreachable; three page-header conventions, three tab mechanisms, six hand-rolled segmented controls; the agent page duplicates the dock; a graph legend and ~80 explainer paragraphs doing the work the UI should do; giant files; four screens missing (Overview, Reports, Schedules, Import).
- **Critical runtime bugs the green tests cannot see** (agent chat "Working" forever, Stop inert): the frontend marks a turn "working" *after* `Agent.Send` returns, while Go starts the turn goroutine *before* returning, so a fast terminal `agent.done` (missing key, provider error) is overwritten and no further event ever arrives; Stop ignores Go's `cancelled=false` and never clears local state; `awaiting-confirm` is never swept; lock/unlock clears listener sets so every live store is detached for the rest of the session; OpenAI/Gemini turns have no deadline; the turn goroutine has no panic recovery; `Core.Lock/Unlock` hold the write mutex across `wg.Wait` and `compose`; three `*.changed` events are emitted by Go and never subscribed.
- **Nobody has ever seen this UI rendered.** Wails v3 beta.23 supports `Windows.AdditionalBrowserArgs` (`--remote-debugging-port`), so a build-tagged harness can launch the real window and screenshot every route over CDP.

Outcome: a product-grade, native-feeling, intuitive Windows desktop app the client can use without explanation. No stubs, no hardcode, no legends. Every screen on one screen contract, every external link through Tor Browser only.

## 2. Fixed decisions (settled with the user on 2026-09-20)

| Topic | Decision |
|---|---|
| Strategy | Keep the foundation (data layer, tokens, copy, canvas, models, tests). Rebuild the shell, define one screen contract, re-frame every existing screen onto it, build the four missing screens, fix the agent turn model in Go and TS. |
| Rail | 68px, icon (20) + 10px label, three sections with hairline separators: **Site** (Overview, Graph, Pages, Linking, Runs), **Production** (Templates, Schedules, Import, Reports), **Global** pinned at the bottom (Agent with pending badge, Sites, Settings). No tooltips needed, no legend. |
| Agent | Keep **both** the dock (on every screen, site-bound) and the full page, on **one** `ConversationView` and **one** store. Improve ergonomics; do not remove either. Inbox stays at `/agent/inbox`. |
| Tor Browser | Every external open goes through a Go `BrowserService.Open` that launches Tor Browser only. Auto-detect standard install locations; a path picker in Settings when not found; "Open" affordances say Tor is not set up and lead to Settings. Never the system browser. |
| In-app preview | The themed page preview iframe stays (desktop/tablet/phone), with an "Open in Tor Browser" button beside it. |
| Missing screens | All four are built in this phase. |
| Claude Design | Templates, Settings, Import, Schedules, Reports get mocks from Claude Design (prompts in appendix A). Overview, Graph, Runs, Pages, Agent, Locked start already have mocks in `design/Postulator.dc.html`. Linking, Sites keep their current layout re-framed onto the contract. |
| Agents | Eight agents total including review (raised from six by the user), max two concurrent. Reviews are done by the orchestrator from screenshots and targeted diffs; one final whole-branch review agent. |
| Tree | This session owns the working tree. Conventional commits on `rewrite/v2`, stage explicit paths only, never `git add -A`, never amend/rebase/force/`--no-verify`. Trailer: `Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>` for worker commits, `Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>` for the orchestrator's. |
| Rules that still bind | CLAUDE.md hard rules: no comments in Go/TS, no stubs/TODOs, consumer-declared interfaces, TDD with coverage gates, cursor paging, camelCase JSON, kernel errors at boundaries, secrets never in logs/artifacts/plain columns, no new `.md` beyond the allowed set, rewrite not patch. |

## 3. The screen contract (binding for every worker)

All values come from `frontend/src/ui/theme.css`; nothing else defines a color, size or radius. The design export `design/Postulator.dc.html` and the five Claude Design mocks are the visual authority; the contract below is what makes them one product.

**Frame.** One `Screen` primitive (`ui/screen.tsx`) wraps every route element:
- Header: 40px, `border-b border-hairline`, `px-4`. Left: title (`title` scale 16/600) plus at most one badge (count or status). Centre: optional tabs. Right: actions, at most three buttons, the rest in an overflow `Menu`. **No subtitle, no paragraph, ever.**
- Body variants: `plain` (scrolling content, `p-4`), `split` (optional left rail 212px `border-r`, main, optional right panel 320px `border-l`; rails and panels scroll independently, main is `min-w-0`), `full` (canvas or table fills, no padding).
- Toolbar inside the body when a screen needs filters or view toggles: 36px, `px-3`, `gap-2`, `border-b border-hairline`.
- Widths: dock narrow 392px, dock wide 50% of the window, drawer 688px, bottom sheet ≤ 40% height, dialog `min(28rem, 100vw - 2rem)` reserved for destructive confirmation and short forms.

**Primitives, one of each** (`ui/*`; the six inline segmented controls, the three tab mechanisms and the three inline Radix dropdowns are replaced by these):
- `Tabs` — underline style, works route-driven (`to`) and state-driven (`value`) through one component.
- `Segmented` — view toggles (table/tree, desktop/tablet/phone, layer chooser).
- `Menu` — Radix dropdown styled once; used for overflow, context menus and the dock's conversation switcher.
- `Toolbar`, `Screen`, `Kbd` (keyboard hint chip) are new; `Panel`, `DenseTable`, `VirtualRows`, `Drawer`, `Sheet`, `Dialog`, `Banner`, `EmptyState`, `StatusBadge`, `Field`, `Select`, `Switch`, `ChipInput`, `ProgressBar`, `BudgetGauge`, `Skeleton*`, `Toast` stay.
- A primitive never sizes itself (`Select` rule from DECISIONS 2026-09-20) and never takes free-form `className` for layout.

**Density.** 13px body, 12.5px table rows at 30px, controls 28px, 4px grid, spacing steps 4/8/12/16/24. Panels `p-3`, 16px between panels. Tables and lists virtualise above 200 rows.

**Prose policy.** Explanatory text exists in exactly three places: an empty state (headline + at most one line + one or two CTAs), inline field validation, and a tooltip of at most one sentence on hover. Legends are banned; a visual encoding is either self-evident or explained by a tooltip on the element itself. The `copy` keys `hint`, `help`, `blurb`, `subtitle` are pruned to what those three surfaces use.

**States.** Every data screen has skeleton (matching its layout), error (`Banner tone="danger"` with retry), empty (`EmptyState` with a CTA that exists), and populated. A button that cannot act is disabled with a tooltip saying why, never hidden.

**Keyboard.** Ctrl+K palette, Ctrl+J dock, Esc closes the top-most overlay, Enter sends in the composer and Shift+Enter breaks a line, J/K move in drawers over lists, A/R decide proposals and pending actions. Shortcuts are shown as `Kbd` chips in the palette and on the confirmation card, nowhere else.

**Numbers and time.** Through `domain/format.ts` only: `$0.00`, `12.3k` tokens, relative time with the exact RFC3339 instant in a tooltip.

**Files.** A feature file stays under 300 lines; a screen is composed from parts under `features/<name>/`. `copy/index.ts` splits into `copy/<feature>.ts` modules re-exported from `copy/index.ts`; `features/*/labels.ts` tone/icon maps stay beside their feature.

## 4. Shell and navigation

- `/` → the last opened site's overview (remembered in `localStorage`) when sites exist, else `/sites`. `/onboarding` and `NotBuilt` are deleted; readiness lives in Overview.
- Title bar: Postulator mark, site switcher (`Select`, shows name and base URL), a centred Ctrl+K search field, window controls. Switching a site goes to that site's overview.
- Rail as decided in §2. Site section hidden when no site is in the route.
- Status bar 26px: active run for the current site (clickable, live status), today's spend, plugin version/health, lock state. Each item is a link to where it is managed.
- Command palette (`features/palette`): sections *Go to* (screens), *Entities* (`Graph.ListEntities` with `namePrefix`), *Pages* (`Pages.List` with `pathPrefix`), *Runs* (recent), *Actions* (New run, Sync site, New entity, Ask the agent). Debounced, keyboard-only, closes on Esc.
- Dock: header = conversation `Menu` (this site's conversations, "New conversation"), width toggle (narrow/wide), open full page, close. Everything else is `ConversationView`.

## 5. Agent turn model (Go + TS contract)

**Go (`internal/application/agent`, `internal/transport/wails`):**
- `AgentService.Send` returns `{messageId, assistantMessageId}`; the assistant id is the one every `agent.delta`/`agent.done` for the turn carries.
- New `AgentService.Status({conversationId}) → {running, messageId, startedAt, lastSeq}` from a turn registry that records the assistant message id, start time and last delta seq; `Turns.Running` is exposed through it.
- `agent.done` payload gains `code` (a frozen `kernel/errors` code, empty on success) and `error` is the described message (same redaction as `MarshalError`; `INTERNAL` collapses to the generic message). `CANCELLED` is how "stopped" is recognised; no string comparison.
- New setting `agent.turnTimeout` (duration, default 10m, 1m–2h); `Turns.Start` derives the turn context with that deadline for every provider.
- The turn goroutine recovers panics and emits `agent.done{code: INTERNAL}`; the stream forwarder drains the upstream channel on exit.
- `Core.Lock`/`Unlock`/`ImportBackup`: the kit is swapped under the write lock, quiesce and compose run outside it, a `composing` guard refuses a concurrent second unlock with `CONFLICT`. `live()` keeps returning `LOCKED` while no store is installed.
- `EventBridge`: no change; the three `*.changed` events already exist.

**TS (`frontend/src/data/agent`, `frontend/src/data/hooks/agent.ts`):**
- The turn store is keyed by `conversationId`; status ∈ `idle | working | awaiting-confirm | stopping | done | error`. `useSendMessage` begins the turn in `onMutate`, attaches `assistantMessageId` when `Send` resolves, and reconciles with `Agent.Status` after `Send`, on mount, on window focus and whenever no event has arrived for 30s. A store that says `working` while Go says not running refetches messages and settles to `idle` or `error`.
- Stop → `stopping`; `Cancel` resolving `cancelled=false` triggers an immediate reconcile; the transcript shows "Stopped" on `code=CANCELLED`.
- `dropAllTurns`/`dropAllLogs` reset snapshots and notify; they never clear listener sets.
- The auto-pagination effect stops on error; `StartConversation` creates and sends in one mutation; `sites.changed`, `settings.changed`, `schedules.changed` are subscribed in `bridge.tsx`; a generated exhaustive handler map makes a missing subscription a type error.

## 6. Tor Browser opener (Go + TS contract)

- Setting `browser.torPath` (string, group `browser`; validator: empty or an absolute path to an existing file).
- `internal/adapters/browser/tor`: `Locate(configured string) → (path, source, ok)` checks the configured path first, then `%USERPROFILE%\Desktop`, `%USERPROFILE%\OneDrive\Desktop`, `%LOCALAPPDATA%`, `%PROGRAMFILES%`, `%USERPROFILE%\Downloads`, each as `<root>\Tor Browser\Browser\firefox.exe` with a sibling `TorBrowser\` directory as proof it is Tor Browser and not Firefox. `Open(path, url)` starts `firefox.exe <url>` detached (`DETACHED_PROCESS | CREATE_NEW_PROCESS_GROUP`, `Dir` = the exe's folder) and returns once the process started.
- `internal/application/browser`: the use case; refuses non-http(s) URLs with `INVALID`, reports a missing browser as `INVALID` with `details.code = "tor_missing"`, a launch failure as `EXTERNAL`.
- `BrowserService` (Wails): `Open({url})`, `Locate() → {path, source: setting|detected|"", installed}`.
- Frontend: `data/host.ts` `openExternal` calls the endpoint; the runtime `Browser` import, `window.open` and `target=` are banned by `check:frontend:imports`. The `tor_missing` reaction is a toast with a "Set up Tor Browser" action that routes to Settings. Settings shows a Browser card: detected/configured path, "Choose…" (native file dialog filtered to `firefox.exe`), "Test" (opens `https://check.torproject.org/`). The `HtmlPreview` sandbox keeps `allow-popups` and `allow-top-navigation` off.

## 7. UI harness (the first deliverable, both waves depend on it)

- `cmd/postulator/harness.go` (`//go:build uiharness`) and its `!uiharness` twin: the tagged build composes the core over `adapters/llm/fake`, starts an in-process `wptest` WordPress, seeds a site, a graph, pages, templates, a finished run and a conversation through the real use cases when the database is empty, and passes `--remote-debugging-port=<POSTULATOR_DEVTOOLS_PORT>` through `Windows.AdditionalBrowserArgs`. `POSTULATOR_HOME` overrides the data directory in both builds.
- `task ui:run` builds `bin/postulator-ui.exe` with the tag and launches it against a scratch `POSTULATOR_HOME`; `task ui:shot -- <hash-route> <name>` runs `frontend/scripts/shot.mjs` (`playwright-core` dev dependency, `connectOverCDP`, no browser download) and writes `frontend/shots/<name>.png` (gitignored). `task ui:walk` screenshots every route in `frontend/scripts/routes.json`.
- Every worker verifies its screens with the harness and attaches the PNG paths to its report. The orchestrator reviews from the PNGs.

## 8. Work packages

Eight agents, Opus 5, `frontend-design:frontend-design` loaded for UI work. Each package: the worker reads its brief, writes a short plan (files, order, decisions) to its report file, waits for the orchestrator's approval, then codes with TDD and commits per unit. Disjoint file ownership per wave; a worker never touches another's paths in the same wave.

### A1 — Go patch set (wave 1, parallel with A2)
Owns `cmd/postulator`, `internal/app`, `internal/application/agent`, `internal/application/browser`, `internal/adapters/browser`, `internal/transport/wails/{agent,browser,service,gen,vocabgen}*`, `internal/transport/agent/{stream,turns}*`, `internal/kernel/settings`, `Taskfile.yml` (`ui:*` tasks, `check:frontend:imports`), generated `events.ts`/`vocab.ts`/bindings.
1. Harness (§7) first, so A2 can screenshot as soon as possible.
2. Agent turn model (§5, Go half). Tests: the race (done before Send returns), timeout, panic → done, cancel → `CANCELLED`, Status before/during/after a turn, lock during a running turn does not block `Ping`/`LockState`.
3. Browser opener (§6, Go half). Tests over a temp directory tree for `Locate`; `Open` tested with a stub executable path.
4. `task events`, `task vocab`, `task bindings` regenerate; `go test -race`, covergate, lint, comment gate green.
Acceptance: all gates green; `task ui:run` opens a seeded window with a devtools port; bindings for `AgentService.Status`, `BrowserService` exist.

### A2 — Frontend foundation and shell (wave 1, parallel with A1)
Owns `frontend/src/{app,ui,copy,lib,domain}`, `frontend/scripts`, `frontend/package.json`, `frontend/tsconfig.json`, `frontend/src/features/palette`.
1. `shot.mjs`, `routes.json`, `playwright-core` dev dependency (Taskfile entries come from A1; coordinate through the orchestrator).
2. Primitives of §3 (`Screen`, `Tabs`, `Segmented`, `Menu`, `Toolbar`, `Kbd`), `noImplicitAny: true`, drop `clsx`, drop `--tracking-key`, delete `smoke.ts`, split `copy/`.
3. Shell of §4: rail, title bar, status bar, palette, routing (`/` rule, delete `/onboarding` and `NotBuilt`; the three not-yet-built routes render a plain `EmptyState` with no jargon until A7 lands).
4. Prove the contract on one existing screen end to end: the Sites screen re-framed (header, toolbar, detail panel, prose pruned) so A4 has a worked example.
Acceptance: typecheck, vitest, `task build` green; screenshots of `/sites`, the palette open, the rail with and without a site, at 1280×820 and 960×600 attached.

### A4 — Pages and Runs re-framed (wave 2, parallel with A3)
Owns `frontend/src/features/{pages,runs}`.
Pages (table/tree, filters, drawer with links, mapping, report, preview) and Runs (list, start dialog with estimate, detail with items, events, review drawer, artifact panes) onto the contract: one header, `Segmented`/`Tabs`/`Menu` primitives, prose pruned, `runs/panes.tsx` and `pages/drawer.tsx` split under 300 lines, "Open in Tor Browser" wording where a link opens externally (A4 keeps calling `openExternal`; A3 changes what it does).
Acceptance: screenshots of `/s/:id/pages` (table and tree, drawer open on each tab), `/s/:id/runs`, `/s/:id/runs/:runId` (running and finished), the review drawer, at both window sizes.

### A3 — Agent, Tor wiring, Overview (wave 2, parallel with A4)
Owns `frontend/src/features/{agent,overview,onboarding}`, `frontend/src/data/{agent,hooks/agent.ts,host.ts,endpoints/browser.ts,hooks/browser.ts,bridge.tsx,lock.ts,errors.ts}`.
1. Turn store and hooks of §5 (TS half), with vitest over the store: race, stall, stop, lock reset.
2. Dock and page on one `ConversationView`, one store (`dock-state.ts` + `model/dock.ts` merged), narrow/wide, conversation `Menu`, composer keys, turn status chip, retry on error, "Ask the agent" prefill opens the dock.
3. `openExternal` through `BrowserService`, `tor_missing` reaction, the import ban.
4. Overview from design screen 1: readiness (reuse `onboarding/readiness.ts`, delete the rest of `onboarding`), run summary, entity/page tiles, depth histogram, edge coverage, drift alert, setup-needed banner, loading/empty/error states.
Acceptance: a missing provider key shows an error row within a second and the composer is usable again; Stop on a running fake turn shows "Stopped"; lock then unlock keeps the transcript live; screenshots of the dock, `/agent`, `/agent/inbox`, `/s/:id/overview`.

### A5 — Templates from the Claude Design mock (wave 3, parallel with A6)
Owns `frontend/src/features/templates`.
Needs the Templates mock (appendix A, prompt 1). Rebuild the list and the editor onto the contract following the mock: three layers, the form groups, policies, page skeleton preview, provenance, save conflict. Prose pruned, files under 300 lines.
Acceptance: screenshots of the list on both tabs and the editor in each layer with and without unsaved changes; `?page=` still opens the page layer; the site default still sets.

### A6 — Settings from the Claude Design mock (wave 3, parallel with A5)
Owns `frontend/src/features/settings`.
Needs the Settings mock (prompt 2). Rebuild onto the contract: General from the schema with per-row reset and search, the Browser tab of §6, Models (providers, profiles, catalog, spend), Security, About. Prose pruned, files under 300 lines.
Acceptance: screenshots of every tab including Browser not found and found; every setting still writes per field with client-side bounds; the Browser tab locates, chooses and tests.

### A7 — Import, Schedules, Reports from the mocks (wave 4, parallel with the orchestrator)
Owns `frontend/src/features/{imports,schedules,reports}`, the three routes.
Needs prompts 3–5's mocks. Import: file pick → inspect → column mapping with saved mappings → preview → apply, plus export. Schedules: list, create/edit (cron or interval, kind, enable), run now, last/next run, `schedules.changed`. Reports: site report (coverage, orphans, cannibalisation, drift, depth, top entities), run reports, page report drill-down, link audit summary linking to Linking.
Acceptance: screenshots of each screen in all four states; the `EmptyState` placeholders are gone.

### Orchestrator — Graph and Linking (wave 4)
Re-frame `features/graph` and `features/links` onto the contract: header and toolbar, legend deleted in favour of badge tooltips and the lens bar, empty-state CTA routes to the real Import, inspector header shared with Linking's panel, file splits. Rolling screenshot reviews of every package. STATUS.md and DECISIONS.md at the end.

### A8 — Final whole-branch review and one fix wave (wave 5)
Opus. Reads the branch diff package, the ledger's deferred minors, and the screenshot set; one fix dispatch on the same agent; one scoped re-review by the orchestrator.

## 9. Waves and process

| Wave | Agents | Gate to next |
|---|---|---|
| 1 | A1 ∥ A2 | harness runs; primitives, shell and Sites screenshotted; all gates green |
| 2 | A3 ∥ A4 | agent chat proven against the fake LLM in the harness; Pages and Runs on the contract |
| 3 | A5 ∥ A6 | needs Templates + Settings mocks (the user generates them during waves 1–2) |
| 4 | A7 ∥ orchestrator | needs Import + Schedules + Reports mocks |
| 5 | A8 | review clean, fix wave merged, docs updated |

Ledger: `.superpowers/sdd/abundant-sauteeing-pillow/progress.md` (git-ignored), one line per package event, every ruling recorded. Worker briefs and reports live beside it. Workers get: the brief, the contract (§3), the relevant §5/§6 half, their file ownership, and the harness commands. They never see this file whole.

Each worker's plan is approved by the orchestrator; the user is asked only when a plan changes a §2 decision.

## 10. Verification

- Per package: `npm run typecheck`, `npm run test:run`, `task build`, `go test -count=1 -race ./...` on touched Go packages, `go run ./cmd/covergate`, `golangci-lint` from `$(go env GOPATH)/bin`, `task check:go:comments`, `task check:frontend:imports`, `gofmt -l .`.
- Visual: `task ui:walk` after every wave; the orchestrator reads the PNGs at 1280×820 and 960×600 and rejects off-contract headers, prose blocks, legends, unlabelled controls, clipped layouts.
- Behavioural, in the harness: send a message with no provider key → error row, composer usable; send with the fake provider → streamed answer; Stop mid-turn → "Stopped"; lock → unlock → send again; confirm a proposed write from the dock and from the inbox; start a run and watch it complete; open a page in Tor Browser with no Tor installed → toast leads to Settings; with a path set → the process starts.
- Final: `task e2e:full` and `task e2e:full:noplugin` on docker, `task package`, then the user walks the installer build.

## Appendix A — Claude Design prompts

Common preamble for every prompt (paste first): *Same product and design language as the existing Postulator design (dark-only, Archivo + Roboto Mono, 13px base on a 4px grid, oklch surfaces, orange accent #E8813A, hairline borders, no shadows, Material Symbols Rounded). Window 1280×820 with the Windows title bar, the 68px left rail (icon + 10px label, three sections: Overview/Graph/Pages/Linking/Runs, Templates/Schedules/Import/Reports, Agent/Sites/Settings at the bottom), a 26px status bar. Every screen has a 40px header: title on the left, optional underline tabs in the centre, at most three actions on the right, never a subtitle or a paragraph. No legends anywhere; a visual encoding is explained by a tooltip on the element or not at all. Empty states: headline, one line, one or two buttons. Drawers over dialogs; dialogs only for destructive confirmation. Show the states listed for each screen as separate frames.*

**Prompt 1 — Templates.** Two screens. (a) `Templates` list: underline tabs *Templates* / *Link policies*; a dense table of templates (name, scope global/site, page kind, version, "site default" marker, updated) with a filter toolbar (scope, page kind, search) and a right-side detail panel showing the selected template's summary (sections, word range, models per role) with *Open editor*, *Duplicate*, *Set as site default*. Policies tab: the same shape for link policies (name, target kinds, min/max links, anchors rule). Actions: *New template* (drawer: blank or copy of), *New policy*. States: populated, empty (no templates yet), loading. (b) `Template editor`: header with the template name (inline rename), a `Segmented` layer chooser *Global / This site / This page* (the page layer only when opened from a page), badges for version and provenance, actions *Preview page*, *Ask the agent*, *Save*. Left 212px outline of the form groups (Sections, Rules, Models, Recipe, Overrides, Policies); main area is the selected group's form: sections as reorderable rows (name, intent, word range, required links), rules as a compact list of switches and numeric fields, models as role→model selects (writer, editor, linker, judge, image, titler, chat), recipe as an ordered step list with two typed step settings (`allowErrors` on validate, `iterations` on repair_links), overrides as a diff-like list of what this layer changes versus the layer below with per-row *Revert*. Right 320px panel: a page skeleton preview that redraws from the sections. States: global layer, site layer with three overrides, page layer, unsaved changes, save conflict banner.

**Prompt 2 — Settings.** A layout route with underline tabs *General / Models / Browser / Security / About*. *General*: settings grouped by their Go group (agent, images, import, llm, runs, schedules, sync, wp) as cards of rows; each row is label + control (switch, number with min/max, text, enum select, duration), value written on blur, an "is default" marker and a per-row *Reset*; a search box in the toolbar and a *Changed only* switch. *Models*: provider cards (OpenAI, Anthropic, Gemini, Gemini via OpenAI) each with key status (set/not set), *Set key…*, *Revoke*, *Test*; a role profiles table (writer, editor, linker, judge, chat, image, titler → provider/model select); the model catalog table (provider, model, input/output price, limits, enabled) with edit drawer; a lifetime spend tile. *Browser*: one card: "Tor Browser" with detected path or "not found", source (detected / chosen), *Choose…* and *Test in Tor* buttons; a one-line inline validation when the chosen file is not Tor Browser. *Security*: master password card (set/change/remove), *Lock now*, backup export/import cards with the native file dialog and password field, a banner when a run is active. *About*: version, commit, build date, counts. States: General populated with one row in validation error, Models with one provider set and a failed test, Browser not found vs found, Security locked-out banner.

**Prompt 3 — Import.** A stepper wizard inside the screen body (no modal): *1 File → 2 Columns → 3 Preview → 4 Apply*. Step 1: drop zone or *Choose file…* (CSV/XLSX), the last five files, saved mappings list on the right (name, columns, last used) with *Use*. Step 2: a table of detected columns (header, sample values, detected type) each with a target select (entity name, kind, parent, page path, title, status, ignore); *Save mapping as…*. Step 3: a preview table of the resulting entities and pages with counts of new/updated/skipped and a warnings list (bad rows with row numbers); *Back*, *Apply*. Step 4: progress and the result (created/updated/skipped, link to Graph and Pages). A second underline tab *Export*: choose what to export (pages, entities, both) and a destination file. States: step 1 empty, step 2 with two unmapped columns, step 3 with warnings, step 4 done.

**Prompt 4 — Schedules.** A dense table of schedules (name, kind generate/relink/audit/sync, cadence as a sentence "every day at 03:00" or "every 6 hours", next run, last run with its status badge, enabled switch) with a filter toolbar (kind, enabled) and a right 320px detail panel for the selected schedule: cadence editor (interval vs cron with a plain-language preview), kind and its parameters (template, publish mode, budget cap), *Run now*, *Disable*, *Delete*, and the last five runs as rows linking to Runs. *New schedule* opens the same panel empty. States: populated with one disabled and one paused-by-budget, empty, editing with a validation error on the cron field.

**Prompt 5 — Reports.** Underline tabs *Site / Runs / Pages*. *Site*: a tile row (pages mapped %, entities with a canonical page %, orphans, cannibalised entities, drifted pages, average depth), a depth histogram, a "worst first" table of entities lacking coverage with a link to Graph, a drift table (page, last published, changed on site) with *Open in Tor Browser* and *Re-sync*, and a link-audit summary tile linking to Linking. *Runs*: a table of finished runs (kind, when, items, succeeded/failed, cost) → selecting one shows its report in a right panel (per-item outcomes with findings counts) and *Open run*. *Pages*: search a page → its last report as sections Validation (findings list), Judge (score and reasons), Publish (result, address, *Open in Tor Browser*), Relink (neighbours touched). States: site populated, site with no runs yet, runs empty, page with a failed validation.

## Appendix B — Docs to update at the end (orchestrator)

- `docs/STATUS.md`: "Where we are" rewritten for the frontend (≤150 lines), the harness in "How to run", gaps closed (agent turn status, `settings.changed`-independent settings screen, Tor opener), next steps → tag `v2.0.0`.
- `docs/DECISIONS.md`: 2026-09-20 entries for the screen contract, the rail, the agent turn model, Tor-only opening, the UI harness.
- `docs/CONTRACTS.md`: `AgentService.Status`, `agent.done.code`, `BrowserService`.
- Memory: update `frontend-plan` and `agent-templates-preview`, add the harness and the Tor rule.
