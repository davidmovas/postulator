# Postulator v2 — Phase 18: 2.7.0, the assistant reads, proposes and works; the writer takes a brief

> **For agentic workers:** one phase, written here as its task list when it opens and closed with a checkpoint; tasks run with TDD under `superpowers:executing-plans`, at most two agents at once on disjoint packages, one accountable owner, no critic (the owner, 2026-10-26).

**Goal:** every business-critical job the client gives the in-app assistant has all three parts, proven end to end: it reads the facts completely, turns its conclusions into a concrete plan, and carries the plan out once the person approves. A run carries what the person and the assistant want to give the writer for it.

**Why now:** the client asked the assistant to design a section's links and to read the pages involved; it said `ParentId` was empty and stopped. The entity read carries no parent (a parent is an edge only), his graph is flat (URL-less rows and proposals), the page side stores `none` for five causes and answers a bare null, and no tool walks a subtree, pages siblings or pages the whole tree.

**Tech stack:** Go 1.27, SQLite through goose, Wails v3, React. The plugin does not change (1.5.2 stays).

**Spec:** part 16 of `docs/superpowers/specs/2026-09-17-postulator-v2-design.md`, 16.13 new; the decisions are in `docs/DECISIONS.md` under 2026-10-26, "phase 18".

## Global constraints

- Every hard rule of `CLAUDE.md`, and phase 17's global constraints.
- Writers that share the tree commit with `git commit -- <paths>`; one writer per Go package. All untagged Go is the owner session's; the frontend is agent F's (frontend-design first), the tag-built capability suite agent E's.
- Stands one at a time after `task plugin:deploy STAND=`, never beside a race group. A leg's duration is estimated before it starts.

## The owner's answers at the plan (2026-10-26)

1. **Must-link targets: the graph first.** A target the graph does not connect to the page is refused at the start, naming the missing edge; the assistant's plan adds the edge as an earlier step, and the run start dialog offers "add as a related edge", written approved when the run starts.
2. **The catalog's admin leaves the assistant.** `models_upsert`, `models_disable`, `models_set_profile`, `models_test_provider` and `models_set_provider_key` go; the assistant keeps the model list, the profiles and the usage.
3. **A plan card edits a step's text inline** (a link's anchor, a field's value, a run's brief and directives, a type picked from a list); anything else goes back to the assistant, which issues a revised plan.
4. **No critic** on this phase or any other.

## The capability matrix

R reads, P proposes, A acts; S single, B bulk. "Today" is the state at `8f98e55`; "Proven by" is filled as each row's test lands. Paths: `T` `internal/application/tools`, `SM` `internal/application/sitemodel`, `CP` `internal/domain/contextpack`.

| # | Job | Today | Gap | Proven by |
|---|---|---|---|---|
| 1 | The site at a glance | R partial: the conversation's brief, frozen at its start (`CP/brief.go:44-104`); `reports_site_overview` unpaged | one live overview: types and counts, top of the tree, content and link health, findings by code | |
| 2 | The page tree | R partial: one level (`sites_explore tree`, `SM/explore.go:115-139`), the trail without ids (`SM/page.go:41-45`), `pages_list` `parentPageId` bare null (`application/pages/views.go:21`) | subtree to a depth, ancestors, siblings paged, the whole tree by cursor, type filters; a parent id always, an empty one with its reason (`pagemap/parent.go:102-138` stores `none` for five causes) | |
| 3 | The entity graph | R partial: no parent in `graph_get_entity`/`graph_list_entities` (`application/graph/views.go:28-44`); parent only through `graph_list_edges`; `graph_load` unpaged. A S+B | parents, children, related edges approved and proposed, anchors, keywords, canonical page and card in one read; entity by name | |
| 4 | The page tree against the graph | missing (no reader of `EdgeParent` in observe or findings) | side by side; `entity_parent_differs`; adoption | |
| 5 | A page in full | R partial: `pages_get` (`SM/page.go`), body as 120-rune excerpts (`SM/edits.go:61-69,188-213`), whole body only through `pages_fields get body` under the 16 KB cap | the body as text block by block, paged; scopes; template with reason; trail with ids; findings with ids | |
| 6 | Many pages at once | R partial: `pages_list` (filters, `under`, max 500) | by ids, by type, by subtree, with key facts, fitted to the cap | |
| 7 | Links | R partial: one page (`sites_explore links`, unpaged), the whole site (`reports_link_audit`, unpaged, halved by the cap); A: `pages_blocks place_link` (S), relink run over the graph's owed links (`runtime/steps/relink_page.go:42-101`) | a section's links in and out with anchors, owed and missing, orphans by section, the section's link map, all paged | |
| 8 | Search | R partial: path prefix on `pages_list` | pages by title, path or keyword; entities by name | |
| 9 | Findings, runs, costs, the import's state | R yes: `findings op list` (cursor), `runs_list`/`runs_list_items` (cursor), `models_usage_summary`, `imports_list_mappings` | one tool each, fitted to the cap | |
| 10 | Types, scopes, templates, rules | R partial: `pagetypes_list` unpaged, `templates_list`; no rule read | rules read | |
| 11 | A link plan for a page or section | missing | facts (`links view suggest`) and a plan: links with anchors and reasons, missing edges first | |
| 12 | Entities, cards, related edges, parents from the tree | P partial: `graph_preview_from_pages`, `graph_propose_related` (writes proposed edges), `graph_propose_cards` | parents from the page tree; as plan steps | |
| 13 | Structure | P partial: `pagetypes_classify preview` | types, rules, templates per type, scopes as plan steps | |
| 14 | A content plan | missing | pages and fields to write, with a brief and directives and the estimate, as a plan | |
| 15 | Resolutions for findings | P none (resolve is an act) | resolutions as plan steps | |
| 16 | Edit fields and blocks | A S: `pages_fields set`, `pages_blocks` (edit runs, `T/pages_blocks.go:119-155`) | in plans | |
| 17 | Place links | A S: `place_link`; relink run (S+B over ids) | over a section, with must-link anchors (Part 4) | |
| 18 | Start runs | A B over ids only (`T/runs_start.go:16-26`) | a selection, a subtree or a type (`pages.Selection`, `application/pages/requests.go:97-111`); the run's brief and directives | |
| 19 | Change types, templates, scopes and rules | A S types and scopes; B classify apply; no rules, no bulk templates for the agent | rules; bulk types and templates over a selection | |
| 20 | Edit the graph in bulk | A B: `graph_create_entities`, `graph_apply_proposals`; edges one by one | adoption of parents; edges in bulk through plans | |
| 21 | Resolve findings in bulk | A S only for the agent (`T/findings.go`) | by ids, `finding.BulkOptions()` | |
| 22 | Revert | A S: `runs_revert` | in plans | |

**Cross-cutting gaps**: a capped list keeps its `nextCursor` (`application/agent/guard.go:34-67`), so the next page skips the rows cut; no tool result is tested at scale; one card is one call; a run holds no brief (`application/runs/requests.go:9-19`, `migrations/0014_runs.sql`).

## Tasks

- [ ] **0.0 — the plan.** This file, the roadmap row.
- [ ] **W1 — the read layer.** `ParentWhy`; the tree views over `page_map` (0082's index, the generation guard); entity parents and children; the body as text; links by section, map, orphans, suggest; `tools.Fit` and the cursor fix in `Cap`.
- [ ] **W2 — the two hierarchies.** `entity_parent_differs`; `graph.AdoptParents` preview and apply; proposals' parents proven; the import's URL-less parent; the flat-graph fixture.
- [ ] **W3 — the run brief.** `run.Brief`, `run.Directive`, 0081, the start, the step context, the pack section, the writer, judge and linker, must-link refusal and the edge offer, `WriterPack` with a draft or a run.
- [ ] **W4 — plans.** The domain, 0080, the store, the executor, `ConfirmPlan`, the events, the resume on open.
- [ ] **W5 — the toolset and the instructions.** The fold by group, the ceilings, the history format, `chat.tmpl`, the token report before and after.
- [ ] **W6 — the support bundle.**
- [ ] **W7 — the screens** (agent F).
- [ ] **W8 — the capability suite** (agent E, after W5).
- [ ] **D — the docs.**
- [ ] **V — 2.7.0** in the seven version files, `task package`.
- [ ] **G — the gate**, exit codes checked.
- [ ] **PR** — #32 retitled and rebodied for 2.7.0.
