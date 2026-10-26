# Postulator v2 — Phase 17: 2.6.0, page classification, templates per type, the Pages screen's views

> **For agentic workers:** one phase, written here as its task list when it opens and closed with a checkpoint; tasks run with TDD under `superpowers:executing-plans`, at most two agents at once on disjoint packages, one accountable owner.

**Goal:** a sync tells what each page is by rules the client can read and change, he sees the site by aggregates and gives a whole aggregate one template, and the Pages screen shows the site's structure in three views with one selection model and bulk actions.

**Why now:** the client's second request after 2.5.1. Everything that is not a product, a hub or technical falls into Entity, the rules are fixed in code, and nothing reassigns pages in bulk or previews a change.

**Tech stack:** Go 1.27, SQLite through goose, Wails v3, React with `@tanstack/react-virtual` (already in use), docker compose for the stands. The plugin does not change (1.5.2 stays): every fact a rule reads is already stored.

**Spec:** `docs/superpowers/specs/2026-09-17-postulator-v2-design.md`, part 16 (16.6 amended, 16.12 new); the decisions are in `docs/DECISIONS.md` under 2026-10-26, "phase 17".

## Global constraints

- Every hard rule of `CLAUDE.md`, and phase 16's global constraints.
- Writers that share the tree commit with `git commit -- <paths>`; one writer per Go package. All Go is the owner session's; the frontend is one agent's.
- Stands one at a time after `task plugin:deploy STAND=`, never beside a race group.
- A leg's duration is estimated before it starts; a leg over half an hour is offered in a shorter form first.

## The owner's answers at the plan (2026-10-26)

1. **A sync re-types the auto pages the rules placed** (reason `rule` or `unmatched`) as well as the untyped ones; a page typed before 2.6 moves only when the person applies a classification; a manual type is never touched. It amends stage 2a's "the observation types only an untyped page".
2. **A rule's conditions combine with AND only.** The brief's rule 8 is seeded as four rules and its rule 9 as two, each with its own reason: fifteen rules and the fallback.

## Decided at the plan

1. **A preset is a named starting point over a fixed role.** `pagetype.Role` is hub, archive, catalog, product, entity or technical, and `Preset.Role()` never changes. The presets are Landing and Category (hub), Store category (archive), Catalog, Product, Guide and Info (entity), Technical, plus Hub and Entity, which existing sites keep and a person may still create from. Every behavioural switch reads the role, never the preset.
2. **A new site is seeded with the brief's eight** (no Hub, no Entity), so every lookup that named a preset goes by role with a preference order: the untyped page's guess in `Set.TypeOf`, the import's synonyms, and the delete guard, which becomes "the last type of its role".
3. **Existing sites get the presets they lack once, by migration 0076**, on every site that has types; `Seed` still fills only a site with none. A name already taken by a type of the same role stands for the preset (the rules point at it); one taken by another role adds the preset as "<Name> pages", then "<Name> pages 2". Nothing is renamed or deleted and no page moves.
4. **Templates and scopes:** Landing and Category take the Hub built-in, Guide and Info the Guide built-in, under 0066's rule (not on a site whose default template a person chose). Info enables only the SEO pair; its body is off.
5. **A type's body switch binds the run start**: when every target's type has the body off, the run is scoped to their enabled fields; a mix of body-on and body-off types is refused in plain words; otherwise nothing changes.
6. **The conditions** are the brief's list and three it needs: "is the front page", "is the shop", "at least N H2s". The URL's segment mode matches a whole segment or a run of its hyphenated words (`guides` matches `/lab-guides/`); a glob is our own matcher (`*` inside a segment, `**` across), compiled and refused at save. The level counts from the front page: the front page is 0, a root or a child of the front page is 1. The technical condition is every technical reason but the shop's.
7. **One classification, two callers.** `pagetype.Classify` walks the folded tree root-first (`contextpack.Map`), so an ancestor's type is settled before its children; a page whose next rule needs a render it lacks waits, and so does one whose ancestor waits when a condition asks for its type; a rule never puts a product or a term into a role it cannot hold; the fallback (Info) answers `unmatched` and raises `type_by_default`. The walk classifies inside derive's unit, over the Map it builds from its own pages, before the map refresh; the preview reads the same facts through `sitemodel` in a fixed number of statements.
8. **Apply is the preview made true**: it takes all groups or chosen ones closed under their dependencies, refuses a preview whose hash moved, writes in units of at most 1,000 pages with each row guarded by the type it was read with, and says what it moved and what changed since. Bulk retyping by hand is manual and role-guarded; "Back to the rules" hands pages back.
9. **Deletions**: deleting a type repoints its rules, the fallback and the ancestor conditions that name it; deleting a rule leaves its pages' reason "a rule since removed" until the next walk; every manual writer clears `page_type_rule_id`.
10. **The agent** gets one tool, `pagetypes_classify` (preview or apply); apply waits for the person in either mode, as the proposal writes do.

### Tasks

- [ ] **0.0 — the plan.** This list and the roadmap row.
- [ ] **P1 — roles.** `pagetype.Role`, the four presets, the sweep of every preset switch, vocab, the tool enum, the ceilings.
- [ ] **P2 — existing sites.** 0076 (the preset CHECK widened by column swap, the index and `page_map_type_moved` recreated, the presets added), `Seed`, the role lookups; the migration tests pinned to the six old presets where they prove 0060.
- [ ] **P3 — the body switch** at the run start.
- [ ] **R1 — rules.** The model, `Validate`, the glob matcher.
- [ ] **R2 — `Classify` and `Explain`** over a table that hits every default rule and edge: first match, waiting, ancestors, the role guard, the level.
- [ ] **R3 — the store.** 0077 (`page_type_rules`, seeded per site, equal to Go's defaults), 0078 (`page_type_reason` widened, `page_type_rule_id`), the repository, CRUD and reorder, the deletions.
- [ ] **R4 — the walk.** The facts function, the classification in derive, `type_by_default`.
- [ ] **A1 — preview and apply.** The hash, dependencies and batches; the preview equals what the apply does; a manual type is kept.
- [ ] **A2 — bulk.** `pages.Selection`, `SetTypes`, `SetTemplates` (replacing `AssignTemplate`'s loop), Back to the rules.
- [ ] **A3 — the agent's tool**, the ceilings and the token report.
- [ ] **T1 — the PageType view**: role, overrides, suggested template.
- [ ] **L1 — the Pages reads.** List filters (type, body, findings, search), title and status sorts, `Count`, `MapRequest.Within`.
- [ ] **G1 — the proposals** name their source page's type.
- [ ] **B — the bindings**, `task build` with no drift.
- [ ] **F0–F6 — the screens** (agent F): the role switches; the shared selection model; the Pages views and the bulk bar; Settings → Classification and the preview; Page types; the dialog's additions; the harness shots and the large seeded site.
- [ ] **E1 — the stands**: the replica's and vanilla's classification tables; a run on auto over a Landing, a Category and a Guide page picks Hub, Hub and Guide.
- [ ] **S1 — scale**: the facts, the preview and the apply flat at 2,000 and 5,000 pages; a second writer's wait during an apply.
- [ ] **D — the docs.**
- [ ] **V — 2.6.0** in the seven version files.
- [ ] **G — the gate**, exit codes checked.
- [ ] **K — the build**: `task package VERSION=2.6.0`.
- [ ] **PR — the pull request** from `dev` to `master`; not merged, not tagged.

**Cut line:** if the phase must split, it splits after B; the screens and the stands finish in a second session.
