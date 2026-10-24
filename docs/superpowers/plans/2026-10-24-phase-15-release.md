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

- [ ] **0.0 — the plan.** This list and the roadmap row. Commit `docs(plan): phase 15, the release, session 1's task list`.
- [ ] **R — research.** The `@since` of every WordPress function and hook the plugin uses; the docker tags for that WordPress and its WP-CLI; the last WooCommerce, Rank Math and Yoast releases for it.
- [ ] **P1 — the PHP matrix** (agent). The function scan, strict tests, `plugin:lint` and `plugin:test` on 7.2, 7.4 and 8.4, the CI matrix, the headers. Commit `build(plugin): lint and test the plugin on PHP 7.2, 7.4 and 8.4`.
- [ ] **P2 — 1.5.2** (agent). The version on both sides, `PluginMinimum` 1.5.1, the pinned tests. Commit `feat(plugin): 1.5.2 declares the PHP and WordPress it runs on`.
- [ ] **P3 — the legacy stand** (agent). `WP_IMAGE`, `WP_CLI_IMAGE`, `STAND=legacy`, the contract suite run once and recorded. Commit `test(e2e): a stand on the oldest WordPress the plugin declares`.
- [ ] **E1 — the finding.** Code, option, kind, rule; derive raises and clears it. Commit `feat(observe): a writable page no entity carries is a finding`.
- [ ] **E2 — the site proposal.** `graph.ProposeFromSite`, `PreviewFromPages` `source`. Commit `feat(graph): propose an entity for each page from the site itself`.
- [ ] **E3 — apply under the parent.** Approved edge, scope, scope-and-key match, depth order, marked clashes; the mapping writers clear the finding. Commit `feat(graph): a site proposal is written under its parent`.
- [ ] **E4 — the resolution and the tools.** `Resolve` and `ResolveMany` answer the preview; `Explain`; `source`; the two `Confirm` summaries and their registry test; the ceilings; `vocab.ts`; the bindings. Commit `feat(findings): propose entities for the pages without one`.
- [ ] **C1 — the counter.** `llm.Tokens` on its own constants. Commit `feat(llm): count a pack's tokens as the report measured them`.
- [ ] **C2 — the input estimate.** Per-step overhead, no share of the output, the judge's body; the ±25% and `Fit` tests. Commit `feat(runtime): price a step's input as its prompt measures`.
- [ ] **S1 — the trace hook.** Commit `test(sqlite): count the statements and rows a store runs`.
- [ ] **S2 — the generator and the counts.** Red where a count differs. Commit `test(scale): a generated site of 2,000 and 5,000 pages on a real store`.
- [ ] **S3 — `page_map`.** 0074, the function and its writers, the differential test. Commit `feat(sitemodel): the folded map is stored as it is derived`.
- [ ] **S4 — the Site map over `page_map`.** Commit `perf(sitemodel): a map page reads its nodes, never the site`.
- [ ] **S5 — the findings list.** Commit `perf(findings): filter and page the findings in SQL`.
- [ ] **S6 — the picker.** The map's API with what the picker reads; the parents from the run's plan; `Pages.Tree` gone. Commit `feat(pages): the run start picks from the site map`.
- [ ] **S7 — the pack, the estimate and the observation.** Commit `perf(runtime): a run's estimate reads the site once`, and `perf(observe): a walk reads the model once per stage`.
- [ ] **S8 — the plans.** `EXPLAIN QUERY PLAN` over every traced statement; the indexes. Commit `perf(sqlite): index what the large reads scan`.
- [ ] **F1 — the inbox's proposal** (agent). Commit `feat(findings): propose entities from the inbox`.
- [ ] **F2 — the picker** (agent). Commit `feat(runs): the run start picks pages from the site map`.
- [ ] **X1 — the replica proof.** A product with no entity gets one through the proposal, a product run writes its description on the fake provider, the wire log carries no denied key, the store facts stand; then the replica capture. Commit `test(e2e): a product without an entity is written through a proposal`.
- [ ] **W1 — the report over the new capture.** Meta measured, the product measured. Commit `feat(tokenreport): the meta writer and a product are measured`.
- [ ] **W2 — the prompts.** `-prompts`, the agent's first round, `out/` ignored. Commit `feat(tokenreport): write the steps' prompts for a person to read`.
- [ ] **G — the gate**, one leg at a time on the final code.
- [ ] **D — the docs and the checkpoint.** `DECISIONS.md` (the 2026-10-12 ruling amended), `CLAUDE.md`, `ARCHITECTURE.md`, `STATUS.md`, this checkpoint. Commit `docs: phase 15, session 1`.

## Session 2 — polish, docs, the build and the PR

Planned at its start from session 1's checkpoint.
