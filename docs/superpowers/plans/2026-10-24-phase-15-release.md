# Postulator v2 — Phase 15: the release

> **For agentic workers:** two sessions, each written here as its task list when it opens and closed with a checkpoint; tasks run with TDD under `superpowers:executing-plans`, at most two agents at once on disjoint packages, one accountable owner.

**Goal:** the client gets a release that holds on day one without a live smoke test: the companion plugin installs on the PHP and WordPress his sites run, every product he has can be written once a person accepts an entity for it, the estimate stops refusing runs that would fit, the screens stay fast on a site of thousands of pages, and the owner can judge the writer's prompts by hand.

**Why now:** the owner has no funded OpenAI key, so the pre-release smoke test of `STATUS.md` cannot run; this phase closes what the smoke test would not have caught anyway.

**Tech stack:** Go 1.27, SQLite through goose, Wails v3, React, PHP 7.2 to 8.4 for the plugin, docker compose for the stands.

**Spec:** `docs/superpowers/specs/2026-09-17-postulator-v2-design.md`, part 16; the decisions are in `docs/DECISIONS.md` under 2026-10-24, "phase 15".

## Global constraints

- Every hard rule of `CLAUDE.md`, and phase 14's global constraints (`2026-10-17-phase-14-site-model.md`), PHP 7.2 in place of 7.4.
- Writers that share the tree commit with `git commit -- <paths>`; one writer per Go package.
- Stands one at a time after `task plugin:deploy STAND=`, never beside a race group; the adapters race group at `-p 1 -timeout 30m`.

## Session 1 — what must hold before the client gets it (2026-10-24)

- **Touches.** `wp-plugin` (headers, tests), `docker/e2e` (the legacy stand), `Taskfile.yml`, `.github/workflows/ci.yml`, `internal/adapters/wp`; `internal/domain/{finding,graph,llm,contextpack}`; `internal/application/{graph,findings,observe,pages,imports,sitemodel,tools}`; `internal/runtime` and `runtime/steps`; `internal/adapters/sqlite` (a trace hook, `page_map`, indexes); `internal/app/scale` (new, tests only); `cmd/tokenreport`; `internal/transport/wails`; `frontend/src/features/{findings,graph,runs}`; `internal/e2e`; the docs.
- **Migrations.** 0074: `page_map` and the indexes the plans ask for. **Tools.** `graph_preview_from_pages` gains `source`; `graph_apply_proposals` and `graph_propose_from_pages` wait for the person in either mode. **Plugin.** 1.5.2, 1.5.1 still accepted.
- **Exit.** The plugin's lint, tests and function scan on PHP 7.2, 7.4 and 8.4, and its contract suite on the oldest WordPress it declares; a replica product with no entity written through an accepted proposal; the estimate within ±25% of the real input on every captured page with no pack refused or over budget; the query counts of five operations equal at 2,000 and 5,000 pages on a real store; `out/prompts` written by one command; the whole gate.

### Decided at the plan

1. **The owner's answers**: the run start's picker reads the Site map's tree; accepting a site proposal approves its parent edge and scopes the entity under that parent; the old-WordPress stand stays as `STAND=legacy` (port 8086, the contract suite only, not part of `e2e:full`).
2. **The plugin's floors.** Its sources use nothing newer than PHP 7.1 and declare no class, so nothing is rewritten. `php -l` cannot see a newer function, so `tests/functions.php` tokenizes every plugin file and requires each called function to be the plugin's own, on a kept list of WordPress, WooCommerce and SEO functions and hooks with their `@since`, or present in the PHP running it; it also asserts the list's newest `@since` is the header's `Requires at least`. The tests run under `E_ALL` and fail on a notice or deprecation. `Requires PHP: 7.2`; `Requires at least:` the newest `@since` used (5.7, `wp_robots`, expected), with no shim to go lower. Version 1.5.2; Go keeps `PluginMinimum` 1.5.1 apart from the bundled `PluginVersion`.
3. **The legacy stand** runs the oldest PHP image that exists for that WordPress, with WooCommerce, Rank Math and Yoast at their last releases that support it; a contract test needing what it lacks skips with its reason, and the record names exactly what ran.
4. **`page_without_entity`**: a warning per page, subject its path alone, one bulk option `propose_entity` of a new option kind `proposal`. The derive stage raises it, after `untyped()` has typed the walk, for every live, unarchived, non-technical page but the root whose effective type's preset is writable, and clears it once an entity carries the page; every writer that maps a page (`ApplyProposals`, `pages.MapToEntity`, `SetCanonical`, the import's apply) clears it in its own unit as the observation does, so an unmap opens it again and a dismissal stays.
5. **The site proposal** `graph.ProposeFromSite` is pure: the name is the H1, else the title; the parent is the twin hub's entity of the nearest ancestor store category, else the nearest ancestor page's entity, else none; a product's store attributes become its card's facts; proposals sort by trail depth; a name already taken under the same scope, in the graph or the batch, is marked with its reason and left unticked, so one clash never refuses a batch. `PreviewFromPages` takes `source` (`model`, the default, or `site`, which makes no model call); without page ids a site preview covers every page the finding names.
6. **Apply** writes a site proposal's parent edge approved and its entity scoped under that parent, and matches an existing entity by scope and `graph.Key`, never by a bare label; model proposals keep their proposed edges. `findings.Resolve` and `ResolveMany` answer `propose_entity` with the preview request before any unit, as a run option answers a `StartRequest`; the inbox opens the Graph screen's proposal dialog at its preview, and one confirmed apply writes the ticked proposals. Both proposal-writing tools gain a `Confirm` summary, and a registry test requires one on every tool that maps a page.
7. **The estimate's input side.** `llm.Tokens` takes its own constants (1.7 tokens per word, 3.7 runes per token); `llm.WordTokens` stays 2.2 and `steps.tokensPerWord` 3. Each step's price carries its own measured prompt overhead; `inputShareOfOutput` goes to 0, the constant "rest" of a prompt folded into the overhead; the judge prices the body it reads. `cmd/tokenreport`'s tests hold every measured page and step within ±25% of the real count, and at every budget `Fit` refuses no pack whose real count fits and keeps none over it; a margin, if one is needed, lives in `Fit`, never in the counter.
8. **Scale.** `sqlite.Config.Trace`, set only by `sqlitetest`, counts statements and rows through the driver's trace. A seeded generator writes about 2,000 and about 5,000 pages into a real store (hubs four deep, products with variations, every link type, fields, renders, findings, an import). The map, the picker, the findings list under each filter, the pack and the estimate keep their statement counts equal at both sizes, and the map and the picker their rows too; the sync is measured at 5% and 10% touched, and its fixed statements and statements per touched page are equal at both sizes. Every distinct traced statement passes `EXPLAIN QUERY PLAN` without a scan of a large table, or is named a whole-site read by design. Timings are logged, never asserted.
9. **The map is materialized**: a recursive query would keep counts flat while its work grows with the site. One function computes the folded map from the model (twins, the loop cut, depth, a root-first trail of ids, descendants) and writes `page_map`, only the rows that moved; derive runs it, and so does every writer that moves a parent or a name the twins read. A differential test proves the SQL reads equal `contextpack.Map` over the generator's store, and `sitemodeltest.Memory` serves every new read.
10. **The prompts**: `go run ./cmd/tokenreport -prompts out/prompts` writes the body writer's, the meta writer's and the judge's system and user messages and schema for `/peptides/`, `/tb-500/` and the replica product the proof gave an entity, from a replica capture taken after the proof, and one file with an agent turn's first round; each file opens with its page, step, role, model, effort and commit. `out/` is gitignored.

### Tasks

- [x] **0.0 — the plan** (`a16f683`). This list and the roadmap row.
- [x] **R — research.** A Sonnet researcher: 5.7.0 from `wp_robots`, the docker tags for 5.7, the plugin releases for it. Corrected three times by the stand (WP-CLI 2.6.0, Rank Math 1.0.98.1, Yoast 18.3).
- [x] **P1 — the PHP matrix** (agent P, `07d80d4`). The function scan, strict tests, `plugin:lint` and `plugin:test` on 7.2, 7.4 and 8.5 (8.4 in the plan; 8.5 is the current 8.x), the CI matrix, the headers.
- [x] **P2 — 1.5.2** (agent P, `434f3fb`). The version on both sides, `PluginMinimum` 1.5.1, the pinned tests.
- [x] **P3 — the legacy stand** (agent P, `7ecb29f`). `WP_IMAGE`, `WP_CLI_IMAGE`, `STAND=legacy`, the contract suite: 60 passed, none failed, the skips named.
- [x] **E1 — the finding** (`d526041`).
- [x] **E2 — the site proposal** (`e17ff06`). `contextpack.Map.EntityProposals` in place of `graph.ProposeFromSite`, `contextpack` being the one package that sees the folded map, the fields and the graph.
- [x] **E3 — apply under the parent** (`52e36f3`).
- [x] **E4 — the resolution and the tools** (`0565751`). The registry test covers the two proposal writers, not every tool that maps a page (decision below).
- [x] **C1 — the counter** (`692f710`).
- [x] **C2 — the input estimate** (`faa5242`), with `Fit`'s headroom.
- [x] **S1 to S8** (agent S): `99f7183`, `26df77d`, `d0979c9`, `f7643eb`, `93910b6`, `4c54bf9`, `aab40fd`, `87840d4`, `8fd5828`, `ffc2b72`, `ad5e692`. `Pages.Tree` stays: the Overview and the Reports still call it.
- [x] **F1 — the inbox's proposal** (agent F, `42bb440`).
- [x] **F2 — the picker** (agent F, `8f215d8`). "Select the whole branch" is gone.
- [x] **X1 — the replica proof and the recapture** (`958d677`).
- [x] **W1 — the report over the new capture** (`347a04e`). The linker's overhead became 290 once the product was measured.
- [x] **W2 — the prompts** (`16553bd`).
- [x] **G — the gate.** The memory guard first stopped the application race group (with `covergate`) and both stands' `e2e:full`; the owner had them run the same evening, all green, after one fix the replica found (`feaf2d6`: the proposal proof reverts its run). The record is under "The gate" in `STATUS.md`.
- [x] **D — the docs and the checkpoint** (`15f3e22` and this commit).

#### Decided while building

- **Only the proposal writers wait in either mode.** `pages_map_to_entity` and `pages_set_canonical` map a page to an entity a person or the agent names, and they follow the conversation's mode, as resolving a finding does. The plan's "every tool that maps a page" was narrowed to the two that write entity proposals.
- **The writers clear `page_without_entity` as `model` with `page_mapped`**, not as the observation, so the record says who cleared it and an unmap still opens it again.
- **The report judges a stand-in body** of the template's length, cut from the captured bodies' HTML, and measures the meta writer over a stand-in draft; their prompt files say so.
- **The replica's capture accepts one product's site proposal**: every e2e test opens a database of its own, so the product only X1 mapped could not reach the capture.
- **The scale tests skip under `-race`**, and the gate runs them without it.
- **Agent S took Part 4 whole** rather than the orchestrator, so that its packages had one writer while the orchestrator ran the replica proof and the report.

#### Checkpoint, 2026-10-24 (session 1 closed, the whole gate green)

**Landed on `dev`**, the reasoning under "phase 15, session 1" in `DECISIONS.md`: the plugin's floors and 1.5.2 with the legacy stand; `page_without_entity` and the site proposals with the inbox and the replica proof; the input side of the estimate; `page_map` and the reads over it; the run start's picker over the map; the prompt files.

**Owed**: nothing. The legs the memory guard stopped ran green on the owner's word, the replica after `feaf2d6`.

**What session 2 starts with**:
- the Overview's and the Reports' whole-tree reads, the last callers of `Pages.Tree`;
- the picker's whole-branch pick, if the owner wants it back;
- the docs, `task package` and the PR when the owner asks.

## Session 2 — polish, docs, the build and the PR (2026-10-24)

Planned at its start from session 1's checkpoint and the owner's brief: the owner asks for the release, so the session ends with a pull request from `dev` to `master`, neither merged nor tagged.

- **Touches.** `frontend/src/features/{map,graph,palette,findings}`, `frontend/src/ui`, `frontend/scripts` (the harness walk); the version files (`Taskfile.yml`, `build/config.yml`, `build/windows/info.json`, `internal/app/version.go`, `frontend/package.json`); the docs. No Go behaviour, migration, tool or plugin change.
- **Exit.** The Site map fits at 960 with the dock open, proven by the walk; no screen file over 300 lines; a refused bulk batch reported in plain words; the docs current; 2.5.0 built; the whole gate on the final code; the installer and the plugin zip; the PR open.

### Decided at the plan

1. **The version is 2.5.0.** `v2.4.0` was released on 2026-10-04 (PR #30, the workbooks, the products, OpenAI only and the categories' removal), after the `v2.3.0` the brief names, so the next minor is 2.5.0. DECISIONS holds no versioning rule beyond the tag pattern `v2.x.y` of `release.yml`; a major is proposed to the owner, not taken.
2. **Bulk resolution stays in batches of 1,000, each its own unit, and the dialog reports a partial result.** Measured on the 5,003-page generated store: a unit holds SQLite's one writer about 0.23 ms per finding it dismisses (941 ms for 4,131) and about 0.8 ms per finding whose site value it accepts (97 ms for 125). A code of 10,000 findings in one unit would hold the writer 2.3 to 8 s, past the 5 s `busy_timeout` every other writer waits (the observation, the engine, every screen's write), while a batch holds it under a second. A unit also refuses whole on one finding it cannot settle (an `accept_site` over 5,000 was refused by one page without an observed value), so one transaction would settle none of them.
3. **The Site map measures its table and picks its columns**: every column while they fit; then without the parent, the links and the children counts; then the page column shrinks and truncates before anything scrolls. The tree, the type, the body's state and the open findings are always shown; what collapses stays in the page drawer. The walk gains a `fits` step that fails a shot whose element scrolls sideways.
4. **The two files split by concern, behaviour unchanged**: the graph screen into its state hook, its map view and its empty state; the palette into its sections hook and its list.

### Tasks

- [ ] **0.0 — the plan.** This list.
- [ ] **P1 — the Site map beside the dock.** The layout by width, failing first in the model project; the walk's `fits` step and two shots at 960 × 600 with the dock open, narrow and wide.
- [ ] **P2 — the two files under 300 lines.** The existing vitest suites and the walk's graph and palette shots, before and after.
- [ ] **P3 — bulk's partial result.** The batches' outcome as a pure function, failing first; the dialog's sentence; the decision in DECISIONS.
- [ ] **D — the docs.** STATUS (the release, what Postulator does, the known limits), VISION, ARCHITECTURE, CONTRACTS (the plugin's floors and 1.5.2), `CLAUDE.md` (the rulings short, the footguns current).
- [ ] **V — 2.5.0** in the five version files.
- [ ] **G — the gate**, exit codes checked: `task build` with no drift; `go vet` (three builds); `golangci-lint` (three builds), `task lint:e2e` alone; the five race groups and `covergate`; the comment check and `gofmt`; `npm run typecheck`, `npm run test:run`, `task ui:lint`; the plugin on 7.2, 7.4 and 8.5; `e2e:full` on both stands from fresh resets and `e2e:full:noplugin` on vanilla.
- [ ] **B — the build.** `task package VERSION=2.5.0`; `task plugin:zip` (1.5.2).
- [ ] **R — the pull request** from `dev` to `master` with the release notes; not merged, not tagged.
