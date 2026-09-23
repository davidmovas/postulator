# Phase 2 — Domain Core, Schema, Repositories and Use Cases Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. This repository forbids sub-agents (spec §10), so the subagent-driven variant does not apply.

**Goal:** Land the foundation every later phase builds on: the pure domain packages `site`, `graph`, `pagemap`, `template` and `llm`; the SQLite schema for sites, the entity graph, the page map, templates, overrides and link policies; keyset-paginated repositories over that schema; and the `sites`, `graph`, `pages` and `templates` use cases wired into the composition root with the five starter templates seeded at startup.

**Architecture:** The domain is value types plus pure functions (a DAG on `parent` edges with weighted undirected `related` edges, a path-keyed page map with a path-derived tree, JSON-merge-patch template resolution) that import only `kernel`, sibling domain packages and the standard library. Migrations 0004–0010 add nine `STRICT` tables with `CHECK`-constrained enums, cascading foreign keys and `(site_id, <sort>, id)` indexes; one repository per aggregate in `internal/adapters/sqlite` turns driver errors into kernel codes at the `dbx` boundary and pages every list with `kernel/paging`. Each use case is a struct whose dependencies arrive through consumer-declared interfaces, exposes `(ctx, Request) (Response, error)` methods with camelCase JSON-tagged request and response structs, runs each mutation inside one `Store.Do` and publishes exactly one application event after that transaction commits.

**Tech Stack:** Go 1.27 (CGO off), `github.com/ncruces/go-sqlite3` v0.30.1, `github.com/pressly/goose/v3` v3.28.0, `github.com/Masterminds/squirrel` v1.5.4 (adapters only), Wails v3 `v3.0.0-beta.23`, golangci-lint `v2.13.2`, Task `v3.53.1`.

**Spec:** `docs/superpowers/specs/2026-09-17-postulator-v2-design.md` — Global Constraints, §3, §4, §5.1–5.4, §5.9, the `Cannibalization` bullet of §6, §9.1, §9.6, §15; the Phase 2 row of `docs/superpowers/plans/2026-09-17-postulator-v2-roadmap.md`. Archond shapes read for this phase: `internal/modules/clusters/{topicalmap,models,proposals,overrides}.go`, `internal/modules/sitemaps/models.go`, `sitemaps/service/{builder,differ}.go`, `internal/modules/content/cannibalization.go` (node roles and intents, tree flatten, canonical pair ordering, proposal status workflow, evidence-per-verdict).

## Global Constraints

- **No comments in Go, TypeScript, YAML or PHP** — not inline, not godoc, not package docs. Naming carries meaning; prose lives in `docs/` and `CLAUDE.md`.
- **No stubs, no TODOs, no placeholders.** If something cannot be done, stop and say so.
- **Interfaces are declared by the consumer**, never by the implementation and never in advance. Unexported, as `internal/app/settings.go` and `internal/adapters/secrets/store.go` already do.
- **TDD.** Failing test first, run it, implement, run it, commit. Table tests with named cases and `t.Parallel()`. Gates: `internal/domain` + `internal/application` ≥ 80% (this phase targets ≥ 85%), module ≥ 70%, kernel ≥ 90%.
- **Cursor pagination only.** No offset, anywhere.
- **JSON is camelCase**, timestamps are RFC3339 UTC (`kernel/dto.Time` on every boundary struct), ids are UUID v4 lowercase text from `kernel/id.New`.
- **Errors crossing a boundary are `*kernel/errors.Error`** with a frozen code: `NOT_FOUND CONFLICT INVALID UNAUTHORIZED RATE_LIMITED BUDGET_EXCEEDED EXTERNAL INTERNAL CANCELLED NEEDS_HUMAN LOCKED`. Adapters convert driver errors at their own edge through `dbx`.
- **Secrets never reach a log, an artifact or a plain column.** The WordPress application password goes through the secret store under `site:<id>:wp_password` and never into `sites`.
- **No new `.md` files** beyond `CLAUDE.md` and the set in spec §10. This plan is the one file this phase adds.
- **Dependency rule**, enforced by `internal/app/deps_test.go`: `domain` imports `kernel`, the standard library and other `domain` packages only; `application` imports `domain` and `kernel`; only `internal/app` imports `internal/transport`; only `kernel/paging` imports squirrel directly.
- **Existing code is reused exactly**: `errors.New|Wrap|CodeOf|IsCode`, `(*Error).WithDetail|WithInternal|WithRetry`; `paging.Keyset|SortKey|TextKey|TimeKey|Request|List|Slice|Cursor|Cursors`, `(Keyset).Apply|Cut|Encode|Position`; `id.New|Valid`; `clock.Clock|System|NewFake`; `dto.Time|NewTime|ListRequest|Sort`; `sqlite.Store|Open|Config|Do`, `execFrom|writeFrom|executor`, `dbx.Result|From|Convert|Classify|IsNotFound|IsConflict`, `sqlitetest.Open`; `events.Type|GraphChanged|PagesChanged|TemplatesChanged|GraphChangedPayload|PagesChangedPayload|TemplatesChangedPayload`; `wails.EventBridge|NewEventBridge|Emitter`; `app.Core|Config|Open|Services`. No kernel package is modified.
- **Migrations** follow `docs/CONVENTIONS.md`: `NNNN_snake_case.sql`, both `-- +goose Up` and `-- +goose Down`, `STRICT`, no `IF EXISTS`, one concern per file, scaffolded with `go run github.com/pressly/goose/v3/cmd/goose@v3.28.0 -s -dir internal/adapters/sqlite/migrations create <name> sql` and renamed to four digits.
- **Commits** are conventional (`<type>(<scope>): <subject>`), on `rewrite/v2`, one per task, never amended, rebased or force-pushed, and every message ends with the trailer `Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>`.
- **Lint** is `golangci-lint run` from `$(go env GOPATH)/bin` with 0 issues: `errcheck` with `check-blank`, `govet` shadow, `gocritic` diagnostic+style+performance (`unnamedResult` means every multi-result signature that is not exactly `(T, error)` names its results; `paramTypeCombine` means `(a, b string)`), `prealloc`, `unparam`, `misspell` US.
- Pinned: Go `1.27`, Wails v3 `v3.0.0-beta.23`, golangci-lint `v2.13.2`, Task `v3.53.1`, Node `22`, squirrel `v1.5.4`, ncruces/go-sqlite3 `v0.30.1`, goose `v3.28.0`.

---

## Design decisions

1. **`Parents(id, depth)` is breadth-first, deduplicated, nearest level first; inside one level the entities are sorted by name (case-insensitively) then id.** A diamond (`D → B → A`, `D → C → A`) yields `[B, C, A]` for depth 2; a shortcut (`D → A` as well) moves `A` into level 1, so `[A, B, C]`. The result is independent of the order edges were loaded in, which is what makes the link-context ordering in Phase 6 reproducible. `Children(id)` and `Roots()` use the same name-then-id order; `Related(id, minWeight)` returns `[]Neighbor{Entity, Weight}` ordered by weight descending then name, because Phase 6 needs the weight for `LinkTarget.Weight` and `[]Entity` would lose it.
2. **`Score()` is unweighted PageRank over approved edges only.** A `parent` edge is one outgoing link child → parent, a `related` edge is one outgoing link in each direction, a `related` edge's weight is a relevance threshold for linking and never an authority weight (a single 0.1-weight proposal must not move the ranking), proposed and rejected edges contribute nothing, dangling mass is redistributed uniformly, damping is 0.85 over 30 iterations, and the map is scaled so the maximum is exactly 1.0. A hub with three children and one grandchild therefore ranks hub > mid-level parent > related sibling > leaves, and two nodes with no incoming links score identically.
3. **`Resolve` is RFC 7396 over JSON and the overrides are merge-patch documents, not `*TemplateSpec`.** The spec's `Resolve(base, siteOverride, pageOverride *TemplateSpec)` cannot be a merge patch: a marshalled struct carries every field, so it would replace the whole base, and `omitempty` would make `false` and `0` unexpressible. The signature becomes `Resolve(base TemplateSpec, siteOverride, pageOverride json.RawMessage) (TemplateSpec, error)` with `MergePatch(target, patch json.RawMessage) (json.RawMessage, error)` as the RFC primitive, tested with all fifteen Appendix A vectors. This is a deliberate deviation from spec §5.4, approved by the orchestrator on 2026-09-18; the spec copy in `docs/` is amended in the same commit as the code (Task 27). **Arrays are replaced wholesale** (`sections`, `recipe`, `keywordRules.include` in a patch override the base list entirely; there is no per-item merge), `null` removes a key, an unknown key or a type mismatch in the result is `INVALID`, and the resolved spec always passes `Validate`. `template_overrides.patch` stores the document. Consequently `TemplateSpec` and its nested structs carry camelCase JSON tags (their persisted form is JSON), as do `llm.ModelRef`, `llm.ModelInfo` and `llm.Usage` (the Phase 4 catalog is JSON); every other domain type carries none.
4. **Anchors live in `entity_anchors (entity_id, position, text, source, weight)` and are loaded in one query per page.** A list reads `limit+1` entities, then one `WHERE entity_id IN (...)` over the collected ids, and attaches the anchors in `position` order: two statements per page, never N+1. `ListBySite` joins through `entities.site_id` for the same two-statement shape. `EntityRepo.Update` rewrites the anchor rows (`DELETE` then `INSERT`) so a single write path exists; the `SetAnchors` use case is `Get → NewAnchors → Update`. `UNIQUE (entity_id, text)` with `text COLLATE NOCASE` backs the domain's case-insensitive duplicate check.
5. **Template overrides are keyed by `(template_id, scope, target)`** where scope is `site` or `page` and the target is `site_id` or `page_id`, each a real foreign key with `ON DELETE CASCADE`. Because SQLite treats `NULL`s as distinct in a `UNIQUE` constraint, uniqueness is an expression index on `(template_id, scope, coalesce(site_id, page_id))`, and the repository upserts by looking the target up first inside the caller's transaction rather than relying on `ON CONFLICT` against an expression index. A page override is keyed by the template too: an override written against Hub must not silently apply when the page moves to Product.
6. **The publisher is invoked once per transaction, after it commits.** Every mutating use case runs its writes inside `uow.Do` and calls `publisher.Publish` only when `Do` returned nil, through one `changed(siteID)` helper per service invoked exactly once per successful transaction; nothing is published from inside the transaction, so a rolled-back write never announces itself. `graph.changed{siteId}` follows every graph mutation, `pages.changed{siteId}` every page mutation, `templates.changed` every template or policy mutation; `DeleteEntity` publishes both graph and pages because it nulls `pages.entity_id`, and `Unmap` and `SetCanonical` publish both because they touch `entities.canonical_page_id`. A caller that nests use cases inside its own `Do` (Phase 8 import) sees one event per nested call before its own commit and should call repositories directly and publish once itself.
7. **Cannibalization is applied on Create, on a path change, and on mapping.** `pages.Create` always runs the verdict with all three reasons before insert and refuses with `CONFLICT` carrying `details.evidence = [{pageId, path, reason, entityId}]`. `pages.Update` re-runs it only when `path` changes (the only input the update can alter), so a title edit never trips a keyword check. `MapToEntity` runs it for the entity the page is being mapped to (`same_entity_canonical`, `same_primary_keyword`). `SetCanonical` never runs it: it is the explicit way to move a canonical page.
8. **`Cannibalization` lives in `internal/domain/pagemap` and keeps the graph parameter.** It depends on `pagemap.Index` and `graph.Entity`/`graph.Graph` only, so it belongs beside the index rather than in `domain/content` (Phase 6). The graph stays in the signature because `same_primary_keyword` must compare the candidate's entity against every other entity's primary keyword and canonical page; the page index alone cannot see keywords. Signature: `Cannibalization(candidate Page, entity graph.Entity, index Index, g graph.Graph) Verdict`. The spec copy in `docs/` is amended in the same commit as the code (Task 27).
9. **The path is the source of truth for the page hierarchy; `parent_page_id` is a maintained cache, and `pagemap.NormalizePath` is the single canonical normaliser** (the WordPress adapter calls it after merge; the PHP plugin implements the identical rules). Binding rules: the input may be an absolute URL or a path; scheme and host are stripped; query and fragment are stripped; no percent-decoding (escapes are kept byte for byte); duplicate slashes collapse; exactly one leading and one trailing slash, the root being `/`; the whole path is lowercased with `strings.ToLower` (escapes are ASCII, so `%2F` and `%2f` are the same byte sequence after lowering); there is no file-extension exception, so `/Shop/index.HTML` is `/shop/index.html/`. A leading `//` in a bare path is duplicate slashes, not a network-path reference. Inputs the plugin never produces are refused as `INVALID` rather than normalised: whitespace or control characters inside the path, and `.` or `..` segments. `ParentPath("/koffein/powder/") == "/koffein/"`, `ParentPath("/") == ""`. `InternalPath(href, siteHost string) (path string, internal bool)` classifies a link target: internal when the href has no host (a relative reference, including same-document `#…` and `?…` references, which yield an empty path) or when its host equals `siteHost` exactly after lowercasing (port included, no `www.` stripping, scheme ignored, and `//host/…` counts as carrying a host); any other scheme such as `mailto:` or `javascript:`, a foreign host, or a path that `NormalizePath` refuses reports `("", false)`; an absolute URL with an empty path is the root `/`. `BuildTree` attaches every page to its nearest existing ancestor by path, so a child created before its parent still lands in the right subtree. `pages.Create` resolves the page's own direct parent and adopts direct children whose `parent_page_id` is still `NULL`; `pages.Update` re-resolves the parent when the path changes and refuses a path change with `CONFLICT` when the page has descendants, because renaming a subtree rewrites every descendant's URL and WordPress slug, which is a WordPress-sync operation of a later phase.
10. **Filters and sort enums are domain query types.** `site.Query`, `graph.EntityQuery`, `graph.EdgeQuery`, `pagemap.Query`, `template.Query`, `template.PolicyQuery` live next to the types they select, so the use case's consumer interface and the SQLite repository both import only `domain`, and no adapter type crosses into `application`. `paging.Request` stays outside the domain.
11. **Use-case list requests embed `kernel/dto.ListRequest`; `cursor` is the `after` cursor.** `docs/CONTRACTS.md` fixes the request shape as `{cursor, limit, sort?}`, and a single cursor cannot say whether it came from `nextCursor` or `prevCursor`. The repositories support both directions through `paging.Request{After, Before}` and are tested both ways; at the use-case boundary a client that wants the previous page replays the cursor it used to reach the current one. `sort.field` accepts `createdAt` (default), `name` for sites, entities, templates and policies, `path` for pages; anything else is `INVALID`, and a cursor issued for a different field or direction is rejected as `INVALID` by the kernel.
12. **Request and response structs are camelCase JSON views owned by the use cases.** Each application package declares its own `Site`, `Entity`, `Edge`, `Page`, `PageLink`, `Template`, `Override`, `LinkPolicy` view with `dto.Time` timestamps and materialised (never nil) slices, and maps the domain value into it; Phase 11 services and Phase 9 tools pass these structs through unchanged. `docs/CONTRACTS.md`'s "DTOs are declared in `internal/transport/wails`" line is amended in Task 27 to say the use cases own them.
13. **Repositories carry no clock.** The use case stamps `CreatedAt`/`UpdatedAt` from its injected `clock.Clock`, truncated to the second (`now().UTC().Truncate(time.Second)`) so a value read back from an RFC3339 column equals the value written; a repository persists exactly the timestamps it is handed.
14. **`Site` gains `Username`.** WordPress application passwords authenticate as `user:password` Basic auth; the spec's `Site` names the password's `SecretRef` but not the user. The user is not a secret, so it is a column. An empty password on `Create` writes no secret ("site has no credentials yet"); `Update` with `password: ""` deletes the secret, with a non-empty value rotates it.
15. **`internal/app` owns an `EventRelay` and exactly one `EventBridge`.** `app.Open` runs before `application.New`, and the Wails `EventManager` exists only after it, so the use cases receive a relay that forwards to the bridge once `Core.Events.Connect(emitter, clock)` has been called from `cmd/postulator` and drops events before that (consistent with the "live events are dropped, not buffered" ruling). `Connect` refuses a second call.
16. **Seed templates do not pin models.** `modelProfiles` is `{}` in every seed: the model catalog is Phase 4 data and a seed naming a provider/model would be a hard-coded value that the catalog may not carry. Roles are chosen per site (`site.Defaults.ModelProfiles`) and per template by the user or an agent. Seeds carry no step `params` either: the documented defaults (`validate` fails on error findings, `repair_links` runs at most twice) are the behaviour the seeds want, and a params key spelling is a Phase 6 decision.
17. **Cyclic foreign keys are real.** `entities.canonical_page_id → pages`, `sites.default_template_id → templates`, `sites.default_link_policy_id → link_policies` reference tables created by later migrations; SQLite resolves references at DML time, and a scratch run against ncruces/go-sqlite3 v0.30.1 confirmed forward references, `ON DELETE SET NULL` across the cycle, the expression unique index, and the down-migration drop order with rows present. Migration order is sites → link_policies → templates → entities → edges → pages → template_overrides.
18. **Parent edges carry weight 1.** `NewEdge` sets `Weight = 1` for `parent` (the weight belongs to `related` edges) and swaps the endpoints of a `related` edge so `FromEntityID < ToEntityID`; the `edges` table enforces both with `CHECK` constraints.
19. **`AddEdge` validates acyclicity as if the new edge were approved; `ApproveEdge` validates again.** A proposal that already contradicts the approved graph is refused at once; two individually fine proposals can only form a cycle when the second is approved, and that is where the second check runs. `RejectEdge` and `DeleteEdge` never validate.
20. **Repository writes report `CONFLICT` with a table-specific message and `NOT_FOUND` on zero affected rows.** `execWrite` passes the raw driver error through `dbx.From(...).Conflict(replacement).WrapErr(dbx.Convert)`, so a `UNIQUE` violation surfaces as "an entity with this name already exists in the site" rather than a driver string; a foreign-key failure stays `INVALID` from `dbx.Classify`; `Update` and `Delete` map zero affected rows to `NOT_FOUND`.
21. **`EnsureSeeded` seeds the five templates and one global link policy named `Default`.** Idempotent by `(scope=global, name)`; `GetEffective(siteID)` returns `site.Defaults.LinkPolicyID` when set, otherwise the global `Default`.
22. **`SetCanonical` maps an unmapped page to the entity and refuses a page mapped elsewhere.** One call is enough to make a page the entity's link target.
23. **`EdgeQuery` sorts by `createdAt` only**; edges have no name and no `updated_at`.
24. **Evidence and cycle paths are carried in `Details` as JSON-friendly values**: `details.cycle` is `[]string` of entity ids, `details.evidence` is a slice of the use case's tagged `Conflict{pageId, path, reason, entityId}` struct.

## Schema

Seven migrations, nine tables. Every file is `STRICT`, has both sections, and its down section drops what its up section created in reverse order.

`internal/adapters/sqlite/migrations/0004_sites.sql`

```sql
-- +goose Up
CREATE TABLE sites (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL COLLATE NOCASE,
    base_url TEXT NOT NULL,
    username TEXT NOT NULL DEFAULT '',
    secret_ref TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('active', 'paused', 'error')),
    allow_insecure INTEGER NOT NULL DEFAULT 0 CHECK (allow_insecure IN (0, 1)),
    plugin_installed INTEGER NOT NULL DEFAULT 0 CHECK (plugin_installed IN (0, 1)),
    plugin_version TEXT NOT NULL DEFAULT '',
    plugin_capabilities TEXT NOT NULL DEFAULT '[]',
    plugin_seo TEXT NOT NULL DEFAULT '',
    default_template_id TEXT REFERENCES templates (id) ON DELETE SET NULL,
    default_link_policy_id TEXT REFERENCES link_policies (id) ON DELETE SET NULL,
    model_profiles TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
) STRICT;

CREATE INDEX sites_created_at ON sites (created_at, id);
CREATE INDEX sites_name ON sites (name, id);

-- +goose Down
DROP INDEX sites_name;
DROP INDEX sites_created_at;
DROP TABLE sites;
```

`internal/adapters/sqlite/migrations/0005_link_policies.sql`

```sql
-- +goose Up
CREATE TABLE link_policies (
    id TEXT PRIMARY KEY,
    scope TEXT NOT NULL CHECK (scope IN ('global', 'site')),
    site_id TEXT REFERENCES sites (id) ON DELETE CASCADE,
    name TEXT NOT NULL COLLATE NOCASE,
    rules TEXT NOT NULL,
    forbid_external INTEGER NOT NULL CHECK (forbid_external IN (0, 1)),
    forbid_self INTEGER NOT NULL CHECK (forbid_self IN (0, 1)),
    anchor_strategy TEXT NOT NULL CHECK (anchor_strategy IN ('prefer_user', 'rotate')),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CHECK ((scope = 'global' AND site_id IS NULL) OR (scope = 'site' AND site_id IS NOT NULL))
) STRICT;

CREATE UNIQUE INDEX link_policies_scope_name ON link_policies (scope, coalesce(site_id, ''), name);
CREATE INDEX link_policies_created_at ON link_policies (created_at, id);
CREATE INDEX link_policies_name ON link_policies (name, id);
CREATE INDEX link_policies_site ON link_policies (site_id);

-- +goose Down
DROP INDEX link_policies_site;
DROP INDEX link_policies_name;
DROP INDEX link_policies_created_at;
DROP INDEX link_policies_scope_name;
DROP TABLE link_policies;
```

`internal/adapters/sqlite/migrations/0006_templates.sql`

```sql
-- +goose Up
CREATE TABLE templates (
    id TEXT PRIMARY KEY,
    scope TEXT NOT NULL CHECK (scope IN ('global', 'site')),
    site_id TEXT REFERENCES sites (id) ON DELETE CASCADE,
    name TEXT NOT NULL COLLATE NOCASE,
    page_kind TEXT NOT NULL,
    version INTEGER NOT NULL CHECK (version >= 1),
    spec TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CHECK ((scope = 'global' AND site_id IS NULL) OR (scope = 'site' AND site_id IS NOT NULL))
) STRICT;

CREATE UNIQUE INDEX templates_scope_name ON templates (scope, coalesce(site_id, ''), name);
CREATE INDEX templates_created_at ON templates (created_at, id);
CREATE INDEX templates_name ON templates (name, id);
CREATE INDEX templates_site_kind ON templates (site_id, page_kind);

-- +goose Down
DROP INDEX templates_site_kind;
DROP INDEX templates_name;
DROP INDEX templates_created_at;
DROP INDEX templates_scope_name;
DROP TABLE templates;
```

`internal/adapters/sqlite/migrations/0007_entities.sql`

```sql
-- +goose Up
CREATE TABLE entities (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL REFERENCES sites (id) ON DELETE CASCADE,
    name TEXT NOT NULL COLLATE NOCASE,
    kind TEXT NOT NULL CHECK (kind IN ('hub', 'product', 'topic', 'category', 'custom')),
    intent TEXT NOT NULL DEFAULT '',
    primary_keyword TEXT NOT NULL DEFAULT '',
    secondary_keywords TEXT NOT NULL DEFAULT '[]',
    canonical_page_id TEXT REFERENCES pages (id) ON DELETE SET NULL,
    score REAL NOT NULL DEFAULT 0 CHECK (score >= 0),
    source TEXT NOT NULL CHECK (source IN ('import', 'user', 'ai')),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (site_id, name)
) STRICT;

CREATE INDEX entities_site_created_at ON entities (site_id, created_at, id);
CREATE INDEX entities_canonical_page ON entities (canonical_page_id);

CREATE TABLE entity_anchors (
    entity_id TEXT NOT NULL REFERENCES entities (id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 0),
    text TEXT NOT NULL COLLATE NOCASE,
    source TEXT NOT NULL CHECK (source IN ('user', 'ai')),
    weight REAL NOT NULL CHECK (weight >= 0 AND weight <= 1),
    PRIMARY KEY (entity_id, position),
    UNIQUE (entity_id, text)
) STRICT;

-- +goose Down
DROP TABLE entity_anchors;
DROP INDEX entities_canonical_page;
DROP INDEX entities_site_created_at;
DROP TABLE entities;
```

`UNIQUE (site_id, name)` is the `(site_id, name, id)` keyset index for the name sort: the name is unique within a site, so the id tie-break never has to be read from a second index.

`internal/adapters/sqlite/migrations/0008_edges.sql`

```sql
-- +goose Up
CREATE TABLE edges (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL REFERENCES sites (id) ON DELETE CASCADE,
    from_entity_id TEXT NOT NULL REFERENCES entities (id) ON DELETE CASCADE,
    to_entity_id TEXT NOT NULL REFERENCES entities (id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('parent', 'related')),
    weight REAL NOT NULL CHECK (weight >= 0 AND weight <= 1),
    source TEXT NOT NULL CHECK (source IN ('import', 'user', 'ai')),
    status TEXT NOT NULL CHECK (status IN ('approved', 'proposed', 'rejected')),
    created_at TEXT NOT NULL,
    CHECK (from_entity_id <> to_entity_id),
    CHECK (kind <> 'related' OR from_entity_id < to_entity_id),
    CHECK (kind <> 'parent' OR weight = 1),
    UNIQUE (site_id, from_entity_id, to_entity_id, kind)
) STRICT;

CREATE INDEX edges_site_created_at ON edges (site_id, created_at, id);
CREATE INDEX edges_from_entity ON edges (from_entity_id);
CREATE INDEX edges_to_entity ON edges (to_entity_id);

-- +goose Down
DROP INDEX edges_to_entity;
DROP INDEX edges_from_entity;
DROP INDEX edges_site_created_at;
DROP TABLE edges;
```

`internal/adapters/sqlite/migrations/0009_pages.sql`

```sql
-- +goose Up
CREATE TABLE pages (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL REFERENCES sites (id) ON DELETE CASCADE,
    path TEXT NOT NULL,
    slug TEXT NOT NULL,
    parent_page_id TEXT REFERENCES pages (id) ON DELETE SET NULL,
    wp_type TEXT NOT NULL CHECK (wp_type IN ('page', 'post', 'product', 'product_cat')),
    wp_id INTEGER,
    title TEXT NOT NULL DEFAULT '',
    h1 TEXT NOT NULL DEFAULT '',
    meta_title TEXT NOT NULL DEFAULT '',
    meta_description TEXT NOT NULL DEFAULT '',
    canonical TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL CHECK (status IN ('planned', 'exists', 'published', 'archived')),
    entity_id TEXT REFERENCES entities (id) ON DELETE SET NULL,
    template_id TEXT REFERENCES templates (id) ON DELETE SET NULL,
    content_hash TEXT NOT NULL DEFAULT '',
    wp_modified_at TEXT,
    last_synced_at TEXT,
    drift INTEGER NOT NULL DEFAULT 0 CHECK (drift IN (0, 1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (site_id, path)
) STRICT;

CREATE INDEX pages_site_created_at ON pages (site_id, created_at, id);
CREATE INDEX pages_site_entity ON pages (site_id, entity_id);
CREATE INDEX pages_entity ON pages (entity_id);
CREATE INDEX pages_parent ON pages (parent_page_id);
CREATE INDEX pages_template ON pages (template_id);

CREATE TABLE page_links (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL REFERENCES sites (id) ON DELETE CASCADE,
    from_page_id TEXT NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
    to_page_id TEXT REFERENCES pages (id) ON DELETE SET NULL,
    to_url TEXT NOT NULL,
    anchor_text TEXT NOT NULL,
    origin TEXT NOT NULL CHECK (origin IN ('generated', 'observed')),
    observed_at TEXT NOT NULL
) STRICT;

CREATE INDEX page_links_from ON page_links (from_page_id, observed_at, id);
CREATE INDEX page_links_to ON page_links (to_page_id);
CREATE INDEX page_links_site ON page_links (site_id);

-- +goose Down
DROP INDEX page_links_site;
DROP INDEX page_links_to;
DROP INDEX page_links_from;
DROP TABLE page_links;
DROP INDEX pages_template;
DROP INDEX pages_parent;
DROP INDEX pages_entity;
DROP INDEX pages_site_entity;
DROP INDEX pages_site_created_at;
DROP TABLE pages;
```

`UNIQUE (site_id, path)` is the `(site_id, path, id)` keyset index for the path sort.

`internal/adapters/sqlite/migrations/0010_template_overrides.sql`

```sql
-- +goose Up
CREATE TABLE template_overrides (
    id TEXT PRIMARY KEY,
    template_id TEXT NOT NULL REFERENCES templates (id) ON DELETE CASCADE,
    scope TEXT NOT NULL CHECK (scope IN ('site', 'page')),
    site_id TEXT REFERENCES sites (id) ON DELETE CASCADE,
    page_id TEXT REFERENCES pages (id) ON DELETE CASCADE,
    patch TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CHECK ((scope = 'site' AND site_id IS NOT NULL AND page_id IS NULL) OR (scope = 'page' AND page_id IS NOT NULL AND site_id IS NULL))
) STRICT;

CREATE UNIQUE INDEX template_overrides_target ON template_overrides (template_id, scope, coalesce(site_id, page_id));
CREATE INDEX template_overrides_site ON template_overrides (site_id);
CREATE INDEX template_overrides_page ON template_overrides (page_id);

-- +goose Down
DROP INDEX template_overrides_page;
DROP INDEX template_overrides_site;
DROP INDEX template_overrides_target;
DROP TABLE template_overrides;
```

Cascade map, as the task requires: site delete → entities, entity_anchors (via entities), edges, pages, page_links, site-scoped templates, site-scoped link policies, site and page overrides; entity delete → edges and entity_anchors `CASCADE`, `pages.entity_id SET NULL`; page delete → `entities.canonical_page_id SET NULL`, `pages.parent_page_id SET NULL`, `page_links.from_page_id CASCADE`, `page_links.to_page_id SET NULL`, page overrides `CASCADE`; template delete → overrides `CASCADE`, `pages.template_id SET NULL`, `sites.default_template_id SET NULL`; link policy delete → `sites.default_link_policy_id SET NULL`.

## File structure

| File | Responsibility |
|---|---|
| `internal/domain/llm/llm.go` + `llm_test.go` | `Role`, `ModelRef`, `ModelInfo`, `Usage`, `Cost` |
| `internal/domain/site/site.go` + `query.go` + `site_test.go` | `Site`, `PluginState`, `Defaults`, `Status`, `SecretRef`, `NormalizeBaseURL`, `Validate`, `Sort`, `Query` |
| `internal/domain/graph/entity.go` + `entity_test.go` | `Entity`, `Anchor`, enums, `NewEntity`, `NewAnchors` |
| `internal/domain/graph/edge.go` + `edge_test.go` | `Edge`, enums, `NewEdge` |
| `internal/domain/graph/graph.go` + `graph_test.go` | `Graph`, `New`, `Neighbor`, traversals, `ValidateAcyclic` |
| `internal/domain/graph/score.go` + `score_test.go` | `Score` |
| `internal/domain/graph/query.go` | `EntitySort`, `EntityQuery`, `EdgeQuery` |
| `internal/domain/pagemap/path.go` + `path_test.go` | `NormalizePath`, `ParentPath`, `Slug` |
| `internal/domain/pagemap/page.go` + `page_test.go` | `Page`, `PageLink`, enums, `NewPage`, `NewPageLink`, `Unmapped` |
| `internal/domain/pagemap/tree.go` + `index.go` + `tree_test.go` + `index_test.go` | `Node`, `BuildTree`, `Index` |
| `internal/domain/pagemap/cannibalization.go` + `_test.go` | `Reason`, `Evidence`, `Verdict`, `Cannibalization` |
| `internal/domain/pagemap/query.go` | `Sort`, `Query` |
| `internal/domain/template/spec.go` | `TemplateSpec` and nested structs, `Template`, `Override`, `LinkPolicy`, enums |
| `internal/domain/template/validate.go` + `validate_test.go` | `Validate`, `ValidateLinkRules`, the three `Validate` methods |
| `internal/domain/template/merge.go` + `merge_test.go` | `MergePatch`, `Resolve` |
| `internal/domain/template/seed.go` + `seed_test.go` + `seed/*.json` | embedded starter templates, `Seed` |
| `internal/domain/template/query.go` | `Sort`, `Query`, `PolicyQuery` |
| `internal/adapters/sqlite/migrations/0004_sites.sql` … `0010_template_overrides.sql` | the schema above |
| `internal/adapters/sqlite/migrations_test.go` | inventory and round trip extended to version 10 |
| `internal/adapters/sqlite/rows.go` | `selectAll`, `selectOne`, `execWrite`, time/null/json helpers, `escapeLike` |
| `internal/adapters/sqlite/site_repo.go` + `site_repo_test.go` | `SiteRepo` |
| `internal/adapters/sqlite/entity_repo.go` + `entity_repo_test.go` | `EntityRepo` with anchors |
| `internal/adapters/sqlite/edge_repo.go` + `edge_repo_test.go` | `EdgeRepo` |
| `internal/adapters/sqlite/page_repo.go` + `page_repo_test.go` | `PageRepo` |
| `internal/adapters/sqlite/page_link_repo.go` + `page_link_repo_test.go` | `PageLinkRepo` |
| `internal/adapters/sqlite/template_repo.go` + `template_repo_test.go` | `TemplateRepo` with overrides |
| `internal/adapters/sqlite/link_policy_repo.go` + `link_policy_repo_test.go` | `LinkPolicyRepo` |
| `internal/adapters/sqlite/sqlitetest/fixtures.go` | `Site(t, store)` fixture shared by repository and use-case tests |
| `internal/application/publisher.go` | the consumer-side `Publisher` interface |
| `internal/application/paging.go` + `paging_test.go` | `PageRequest`, `MapList` |
| `internal/application/applicationtest/recorder.go` | `Recorder`, the recording publisher every use-case test injects |
| `internal/application/sites/{service,views,requests}.go` + `service_test.go` | sites use cases |
| `internal/application/graph/{service,views,requests,entities,edges}.go` + `entities_test.go` + `edges_test.go` | graph use cases |
| `internal/application/pages/{service,views,requests,pages,links}.go` + `pages_test.go` + `links_test.go` | pages use cases |
| `internal/application/templates/{service,views,requests,templates,overrides,policies,seed}.go` + `templates_test.go` + `policies_test.go` | templates and link-policy use cases |
| `internal/app/events.go` + `events_test.go` | `EventRelay` |
| `internal/app/core.go` + `core_test.go` | repositories, services, seeding at `Open` |
| `cmd/postulator/main.go` | connects the relay to the Wails event manager |
| `docs/superpowers/specs/2026-09-17-postulator-v2-design.md` | §5.3, §5.4, §6 amended |
| `docs/CONTRACTS.md`, `docs/STATUS.md` | DTO ownership line; Phase 2 entry and decisions |

---

# Half 1 — domain packages, migrations, repositories

### Task 1: `domain/llm` catalog types

**Files:**
- Create: `internal/domain/llm/llm.go`
- Test: `internal/domain/llm/llm_test.go`

**Interfaces:**
- Consumes: nothing beyond the standard library.
- Produces: `type Role string` with `RoleWriter|RoleEditor|RoleLinker|RoleJudge|RoleChat|RoleImage`, `Roles() []Role`, `(Role).Valid() bool`; `type ModelRef struct{Provider, Model string}` with `(ModelRef).Valid() bool` and `(ModelRef).String() string` (`provider:model`); `ModelInfo`, `Usage` with `(Usage).Add(Usage) Usage`; `Cost(usage Usage, info ModelInfo) float64`. Used by `site.Defaults.ModelProfiles` and `template.TemplateSpec.ModelProfiles`.

- [ ] **Step 1: Write the failing test**

```go
package llm_test

import (
	"encoding/json"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/llm"
)

func TestRoles(t *testing.T) {
	t.Parallel()

	want := []llm.Role{llm.RoleWriter, llm.RoleEditor, llm.RoleLinker, llm.RoleJudge, llm.RoleChat, llm.RoleImage}
	got := llm.Roles()
	if len(got) != len(want) {
		t.Fatalf("Roles() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Roles()[%d] = %q, want %q", i, got[i], want[i])
		}
		if !got[i].Valid() {
			t.Errorf("%q must be valid", got[i])
		}
	}
	if llm.Role("painter").Valid() {
		t.Error("an unknown role must not be valid")
	}
}

func TestModelRef(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		ref   llm.ModelRef
		valid bool
		text  string
	}{
		{name: "complete", ref: llm.ModelRef{Provider: "openai", Model: "gpt"}, valid: true, text: "openai:gpt"},
		{name: "no provider", ref: llm.ModelRef{Model: "gpt"}, valid: false, text: ":gpt"},
		{name: "no model", ref: llm.ModelRef{Provider: "openai"}, valid: false, text: "openai:"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.ref.Valid(); got != tc.valid {
				t.Errorf("Valid() = %v, want %v", got, tc.valid)
			}
			if got := tc.ref.String(); got != tc.text {
				t.Errorf("String() = %q, want %q", got, tc.text)
			}
		})
	}
}

func TestModelRefJSONIsCamelCase(t *testing.T) {
	t.Parallel()

	encoded, err := json.Marshal(llm.ModelRef{Provider: "openai", Model: "gpt"})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(encoded) != `{"provider":"openai","model":"gpt"}` {
		t.Fatalf("Marshal = %s", encoded)
	}
}

func TestCost(t *testing.T) {
	t.Parallel()

	info := llm.ModelInfo{Ref: llm.ModelRef{Provider: "openai", Model: "gpt"}, InputUSDPerM: 3, OutputUSDPerM: 15}

	cases := []struct {
		name  string
		usage llm.Usage
		want  float64
	}{
		{name: "nothing used", usage: llm.Usage{}, want: 0},
		{name: "one million input tokens", usage: llm.Usage{Input: 1_000_000}, want: 3},
		{name: "half a million output tokens", usage: llm.Usage{Output: 500_000}, want: 7.5},
		{name: "both", usage: llm.Usage{Input: 1_000_000, Output: 500_000, Total: 1_500_000}, want: 10.5},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := llm.Cost(tc.usage, info); got != tc.want {
				t.Errorf("Cost() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestUsageAdd(t *testing.T) {
	t.Parallel()

	sum := llm.Usage{Input: 10, Output: 5, Total: 15}.Add(llm.Usage{Input: 1, Output: 2, Total: 3})
	if sum != (llm.Usage{Input: 11, Output: 7, Total: 18}) {
		t.Fatalf("Add = %+v", sum)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/llm/ -v`
Expected: FAIL, the package does not exist.

- [ ] **Step 3: Write the implementation**

`internal/domain/llm/llm.go`:

```go
package llm

type Role string

const (
	RoleWriter Role = "writer"
	RoleEditor Role = "editor"
	RoleLinker Role = "linker"
	RoleJudge  Role = "judge"
	RoleChat   Role = "chat"
	RoleImage  Role = "image"
)

func Roles() []Role {
	return []Role{RoleWriter, RoleEditor, RoleLinker, RoleJudge, RoleChat, RoleImage}
}

func (r Role) Valid() bool {
	switch r {
	case RoleWriter, RoleEditor, RoleLinker, RoleJudge, RoleChat, RoleImage:
		return true
	default:
		return false
	}
}

type ModelRef struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

func (r ModelRef) Valid() bool {
	return r.Provider != "" && r.Model != ""
}

func (r ModelRef) String() string {
	return r.Provider + ":" + r.Model
}

type ModelInfo struct {
	Ref                ModelRef `json:"ref"`
	ContextTokens      int      `json:"contextTokens"`
	MaxOutputTokens    int      `json:"maxOutputTokens"`
	InputUSDPerM       float64  `json:"inputUsdPerM"`
	OutputUSDPerM      float64  `json:"outputUsdPerM"`
	RPM                int      `json:"rpm"`
	TPM                int      `json:"tpm"`
	SupportsStructured bool     `json:"supportsStructured"`
	SupportsImages     bool     `json:"supportsImages"`
	Reasoning          bool     `json:"reasoning"`
}

type Usage struct {
	Input  int `json:"input"`
	Output int `json:"output"`
	Total  int `json:"total"`
}

func (u Usage) Add(other Usage) Usage {
	return Usage{Input: u.Input + other.Input, Output: u.Output + other.Output, Total: u.Total + other.Total}
}

const tokensPerMillion = 1_000_000

func Cost(usage Usage, info ModelInfo) float64 {
	input := float64(usage.Input) / tokensPerMillion * info.InputUSDPerM
	output := float64(usage.Output) / tokensPerMillion * info.OutputUSDPerM
	return input + output
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race ./internal/domain/llm/ -v`
Expected: PASS. Also run `go test -race ./internal/app/ -run 'TestLayers|TestDomain' -v`: `internal/domain` now exists, so the domain rules stop skipping and must pass.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/llm/
git commit -m "feat(domain): llm catalog types and cost arithmetic

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 2: `domain/site`

**Files:**
- Create: `internal/domain/site/site.go`, `internal/domain/site/query.go`
- Test: `internal/domain/site/site_test.go`

**Interfaces:**
- Consumes: `llm.Role`, `llm.ModelRef` (Task 1); `errors.New`, `(*errors.Error).WithDetail|WithInternal`.
- Produces: `Status` (`StatusActive|StatusPaused|StatusError`, `Valid()`), `PluginState`, `Defaults`, `Site` (fields `ID, Name, BaseURL, Username, SecretRef string; Status Status; AllowInsecure bool; Plugin PluginState; Defaults Defaults; CreatedAt, UpdatedAt time.Time`), `SecretRef(siteID string) string`, `NormalizeBaseURL(raw string, allowInsecure bool) (string, error)`, `(Site).Validate() error`, `Sort` (`SortCreatedAt = "createdAt"`, `SortName = "name"`, `Valid()`), `Query{Status *Status; Sort Sort; Desc bool}`.

- [ ] **Step 1: Write the failing test**

```go
package site_test

import (
	stderrors "errors"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestSecretRef(t *testing.T) {
	t.Parallel()

	if got := site.SecretRef("9f0d0d22-6f4f-4c1a-9c07-5b6c1f6bd9a1"); got != "site:9f0d0d22-6f4f-4c1a-9c07-5b6c1f6bd9a1:wp_password" {
		t.Fatalf("SecretRef = %q", got)
	}
}

func TestNormalizeBaseURL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		raw      string
		insecure bool
		want     string
		code     errors.Code
	}{
		{name: "https is kept", raw: "https://Example.com", want: "https://example.com"},
		{name: "trailing slash and query are dropped", raw: " https://example.com/blog/?utm=1#x ", want: "https://example.com/blog"},
		{name: "path is kept", raw: "https://example.com/wp", want: "https://example.com/wp"},
		{name: "http refused by default", raw: "http://example.com", code: errors.Invalid},
		{name: "http allowed when insecure", raw: "http://example.com", insecure: true, want: "http://example.com"},
		{name: "empty", raw: "  ", code: errors.Invalid},
		{name: "no scheme", raw: "example.com", code: errors.Invalid},
		{name: "ftp", raw: "ftp://example.com", code: errors.Invalid},
		{name: "no host", raw: "https://", code: errors.Invalid},
		{name: "credentials in url", raw: "https://user:pw@example.com", code: errors.Invalid},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := site.NormalizeBaseURL(tc.raw, tc.insecure)
			if tc.code != "" {
				if !errors.IsCode(err, tc.code) {
					t.Fatalf("code = %q, want %q (err %v)", errors.CodeOf(err), tc.code, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeBaseURL: %v", err)
			}
			if got != tc.want {
				t.Errorf("NormalizeBaseURL = %q, want %q", got, tc.want)
			}
		})
	}
}

func validSite() site.Site {
	at := time.Date(2026, time.September, 18, 9, 0, 0, 0, time.UTC)
	return site.Site{
		ID:        "9f0d0d22-6f4f-4c1a-9c07-5b6c1f6bd9a1",
		Name:      "Shop",
		BaseURL:   "https://shop.example.com",
		Username:  "editor",
		SecretRef: site.SecretRef("9f0d0d22-6f4f-4c1a-9c07-5b6c1f6bd9a1"),
		Status:    site.StatusActive,
		Defaults:  site.Defaults{ModelProfiles: map[llm.Role]llm.ModelRef{llm.RoleWriter: {Provider: "openai", Model: "gpt"}}},
		CreatedAt: at,
		UpdatedAt: at,
	}
}

func TestSiteValidate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		mutate func(*site.Site)
		field  string
	}{
		{name: "valid", mutate: func(*site.Site) {}},
		{name: "no id", mutate: func(s *site.Site) { s.ID = "" }, field: "id"},
		{name: "blank name", mutate: func(s *site.Site) { s.Name = "  " }, field: "name"},
		{name: "http without allow", mutate: func(s *site.Site) { s.BaseURL = "http://shop.example.com" }, field: "baseUrl"},
		{name: "unknown status", mutate: func(s *site.Site) { s.Status = "sleeping" }, field: "status"},
		{name: "foreign secret ref", mutate: func(s *site.Site) { s.SecretRef = "site:other:wp_password" }, field: "secretRef"},
		{name: "unknown role", mutate: func(s *site.Site) { s.Defaults.ModelProfiles = map[llm.Role]llm.ModelRef{"painter": {Provider: "a", Model: "b"}} }, field: "defaults.modelProfiles.painter"},
		{name: "incomplete ref", mutate: func(s *site.Site) { s.Defaults.ModelProfiles = map[llm.Role]llm.ModelRef{llm.RoleChat: {Provider: "a"}} }, field: "defaults.modelProfiles.chat"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := validSite()
			tc.mutate(&s)
			err := s.Validate()
			if tc.field == "" {
				if err != nil {
					t.Fatalf("Validate: %v", err)
				}
				return
			}
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("code = %q, want INVALID (err %v)", errors.CodeOf(err), err)
			}
			var kernel *errors.Error
			if !stderrors.As(err, &kernel) || kernel.Details["field"] != tc.field {
				t.Errorf("details = %v, want field %q", err, tc.field)
			}
		})
	}
}

func TestStatusAndSort(t *testing.T) {
	t.Parallel()

	for _, status := range []site.Status{site.StatusActive, site.StatusPaused, site.StatusError} {
		if !status.Valid() {
			t.Errorf("%q must be valid", status)
		}
	}
	if site.Status("x").Valid() {
		t.Error("unknown status must be invalid")
	}
	if !site.SortCreatedAt.Valid() || !site.SortName.Valid() || site.Sort("age").Valid() {
		t.Error("sort validity is wrong")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/site/ -v`
Expected: FAIL, the package does not exist.

- [ ] **Step 3: Write the implementation**

`internal/domain/site/site.go`:

```go
package site

import (
	"net/url"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type Status string

const (
	StatusActive Status = "active"
	StatusPaused Status = "paused"
	StatusError  Status = "error"
)

func (s Status) Valid() bool {
	switch s {
	case StatusActive, StatusPaused, StatusError:
		return true
	default:
		return false
	}
}

type PluginState struct {
	Installed    bool
	Version      string
	Capabilities []string
	SEOPlugin    string
}

type Defaults struct {
	TemplateID    *string
	LinkPolicyID  *string
	ModelProfiles map[llm.Role]llm.ModelRef
}

type Site struct {
	ID            string
	Name          string
	BaseURL       string
	Username      string
	SecretRef     string
	Status        Status
	AllowInsecure bool
	Plugin        PluginState
	Defaults      Defaults
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func SecretRef(siteID string) string {
	return "site:" + siteID + ":wp_password"
}

func invalid(message, field string) *errors.Error {
	return errors.New(errors.Invalid, message).WithDetail("field", field)
}

func NormalizeBaseURL(raw string, allowInsecure bool) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", invalid("site base url must not be empty", "baseUrl")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", invalid("site base url is not a valid url", "baseUrl").WithInternal(err)
	}

	scheme := strings.ToLower(parsed.Scheme)
	switch {
	case scheme == "https":
	case scheme == "http" && allowInsecure:
	case scheme == "http":
		return "", invalid("site base url must use https unless insecure urls are allowed", "baseUrl")
	default:
		return "", invalid("site base url must be an absolute http or https url", "baseUrl")
	}
	if parsed.Host == "" {
		return "", invalid("site base url must name a host", "baseUrl")
	}
	if parsed.User != nil {
		return "", invalid("site base url must not carry credentials", "baseUrl")
	}

	path := strings.TrimRight(parsed.EscapedPath(), "/")
	return scheme + "://" + strings.ToLower(parsed.Host) + path, nil
}

func (s Site) Validate() error {
	if s.ID == "" {
		return invalid("site id must not be empty", "id")
	}
	if strings.TrimSpace(s.Name) == "" {
		return invalid("site name must not be empty", "name")
	}
	if _, err := NormalizeBaseURL(s.BaseURL, s.AllowInsecure); err != nil {
		return err
	}
	if !s.Status.Valid() {
		return invalid("site status is not recognised", "status")
	}
	if s.SecretRef != SecretRef(s.ID) {
		return invalid("site secret reference does not belong to the site", "secretRef")
	}
	for role, ref := range s.Defaults.ModelProfiles {
		field := "defaults.modelProfiles." + string(role)
		if !role.Valid() {
			return invalid("model profile role is not recognised", field)
		}
		if !ref.Valid() {
			return invalid("model profile needs a provider and a model", field)
		}
	}
	return nil
}
```

`internal/domain/site/query.go`:

```go
package site

type Sort string

const (
	SortCreatedAt Sort = "createdAt"
	SortName      Sort = "name"
)

func (s Sort) Valid() bool {
	switch s {
	case SortCreatedAt, SortName:
		return true
	default:
		return false
	}
}

type Query struct {
	Status *Status
	Sort   Sort
	Desc   bool
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race ./internal/domain/site/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/site/
git commit -m "feat(domain): site aggregate with base url and secret reference rules

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 3: `domain/graph` entities, anchors and edges

**Files:**
- Create: `internal/domain/graph/entity.go`, `internal/domain/graph/edge.go`, `internal/domain/graph/query.go`
- Test: `internal/domain/graph/entity_test.go`, `internal/domain/graph/edge_test.go`

**Interfaces:**
- Consumes: `errors.New`, `(*errors.Error).WithDetail`.
- Produces: `Kind` (`KindHub|KindProduct|KindTopic|KindCategory|KindCustom`), `Source` (`SourceImport|SourceUser|SourceAI`), `AnchorSource` (`AnchorUser|AnchorAI`), `EdgeKind` (`EdgeParent|EdgeRelated`), `EdgeStatus` (`StatusApproved|StatusProposed|StatusRejected`), each with `Valid() bool`; `Anchor{Text string; Source AnchorSource; Weight float64}`; `Entity{ID, SiteID, Name string; Kind Kind; Intent, PrimaryKeyword string; SecondaryKeywords []string; Anchors []Anchor; CanonicalPageID *string; Score float64; Source Source; CreatedAt, UpdatedAt time.Time}`; `Edge{ID, SiteID, FromEntityID, ToEntityID string; Kind EdgeKind; Weight float64; Source Source; Status EdgeStatus; CreatedAt time.Time}`; `NewEntity(Entity) (Entity, error)`, `NewAnchors([]Anchor) ([]Anchor, error)`, `NewEdge(Edge) (Edge, error)`; `EntitySort` (`EntitySortCreatedAt = "createdAt"`, `EntitySortName = "name"`), `EntityQuery{SiteID string; Kind *Kind; HasCanonicalPage *bool; NamePrefix string; Sort EntitySort; Desc bool}`, `EdgeQuery{SiteID string; Kind *EdgeKind; Status *EdgeStatus; EntityID string; Desc bool}` (an empty `SiteID` means every site, which only tests use).

- [ ] **Step 1: Write the failing tests**

`internal/domain/graph/entity_test.go`:

```go
package graph_test

import (
	stderrors "errors"
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	siteA = "0b6c2a4e-1f3d-4c8b-9a2e-5d7f8e9a0b1c"
	entA  = "1a1a1a1a-1a1a-4a1a-8a1a-1a1a1a1a1a1a"
	entB  = "2b2b2b2b-2b2b-4b2b-8b2b-2b2b2b2b2b2b"
	entC  = "3c3c3c3c-3c3c-4c3c-8c3c-3c3c3c3c3c3c"
	entD  = "4d4d4d4d-4d4d-4d4d-8d4d-4d4d4d4d4d4d"
	entE  = "5e5e5e5e-5e5e-4e5e-8e5e-5e5e5e5e5e5e"
)

func fieldOf(t *testing.T, err error) string {
	t.Helper()
	var kernel *errors.Error
	if !stderrors.As(err, &kernel) {
		t.Fatalf("error %v is not a kernel error", err)
	}
	field, _ := kernel.Details["field"].(string)
	return field
}

func validEntity() graph.Entity {
	at := time.Date(2026, time.September, 18, 9, 0, 0, 0, time.UTC)
	return graph.Entity{
		ID: entA, SiteID: siteA, Name: " Running Shoes ", Kind: graph.KindHub, Intent: "commercial",
		PrimaryKeyword: "running shoes", SecondaryKeywords: []string{"trail shoes", " Trail Shoes ", "", "road shoes"},
		Anchors: []graph.Anchor{{Text: "running shoes", Source: graph.AnchorUser, Weight: 1}},
		Source:  graph.SourceUser, CreatedAt: at, UpdatedAt: at,
	}
}

func TestNewEntityNormalises(t *testing.T) {
	t.Parallel()

	entity, err := graph.NewEntity(validEntity())
	if err != nil {
		t.Fatalf("NewEntity: %v", err)
	}
	if entity.Name != "Running Shoes" {
		t.Errorf("Name = %q, want trimmed", entity.Name)
	}
	if !slices.Equal(entity.SecondaryKeywords, []string{"trail shoes", "road shoes"}) {
		t.Errorf("SecondaryKeywords = %v, want trimmed and deduplicated", entity.SecondaryKeywords)
	}
}

func TestNewEntityRejects(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		mutate func(*graph.Entity)
		field  string
	}{
		{name: "no id", mutate: func(e *graph.Entity) { e.ID = "" }, field: "id"},
		{name: "no site", mutate: func(e *graph.Entity) { e.SiteID = "" }, field: "siteId"},
		{name: "blank name", mutate: func(e *graph.Entity) { e.Name = " " }, field: "name"},
		{name: "unknown kind", mutate: func(e *graph.Entity) { e.Kind = "planet" }, field: "kind"},
		{name: "unknown source", mutate: func(e *graph.Entity) { e.Source = "wind" }, field: "source"},
		{name: "negative score", mutate: func(e *graph.Entity) { e.Score = -1 }, field: "score"},
		{name: "empty canonical", mutate: func(e *graph.Entity) { empty := ""; e.CanonicalPageID = &empty }, field: "canonicalPageId"},
		{name: "anchor blank", mutate: func(e *graph.Entity) { e.Anchors = []graph.Anchor{{Text: " ", Source: graph.AnchorUser}} }, field: "anchors[0].text"},
		{name: "anchor source", mutate: func(e *graph.Entity) { e.Anchors = []graph.Anchor{{Text: "a", Source: "bot"}} }, field: "anchors[0].source"},
		{name: "anchor weight", mutate: func(e *graph.Entity) { e.Anchors = []graph.Anchor{{Text: "a", Source: graph.AnchorAI, Weight: 1.5}} }, field: "anchors[0].weight"},
		{name: "anchor repeated", mutate: func(e *graph.Entity) {
			e.Anchors = []graph.Anchor{{Text: "Shoes", Source: graph.AnchorUser}, {Text: "shoes", Source: graph.AnchorAI}}
		}, field: "anchors[1].text"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			entity := validEntity()
			tc.mutate(&entity)
			_, err := graph.NewEntity(entity)
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("code = %q, want INVALID", errors.CodeOf(err))
			}
			if got := fieldOf(t, err); got != tc.field {
				t.Errorf("field = %q, want %q", got, tc.field)
			}
		})
	}
}

func TestNewAnchorsKeepsOrderAndTrims(t *testing.T) {
	t.Parallel()

	anchors, err := graph.NewAnchors([]graph.Anchor{{Text: " b ", Source: graph.AnchorAI, Weight: 0.5}, {Text: "a", Source: graph.AnchorUser, Weight: 1}})
	if err != nil {
		t.Fatalf("NewAnchors: %v", err)
	}
	if len(anchors) != 2 || anchors[0].Text != "b" || anchors[1].Text != "a" {
		t.Fatalf("anchors = %+v", anchors)
	}

	empty, err := graph.NewAnchors(nil)
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("NewAnchors(nil) = %v, %v; want an empty slice", empty, err)
	}
}

func TestEnums(t *testing.T) {
	t.Parallel()

	for _, kind := range []graph.Kind{graph.KindHub, graph.KindProduct, graph.KindTopic, graph.KindCategory, graph.KindCustom} {
		if !kind.Valid() {
			t.Errorf("%q must be valid", kind)
		}
	}
	for _, source := range []graph.Source{graph.SourceImport, graph.SourceUser, graph.SourceAI} {
		if !source.Valid() {
			t.Errorf("%q must be valid", source)
		}
	}
	for _, status := range []graph.EdgeStatus{graph.StatusApproved, graph.StatusProposed, graph.StatusRejected} {
		if !status.Valid() {
			t.Errorf("%q must be valid", status)
		}
	}
	if graph.Kind("x").Valid() || graph.Source("x").Valid() || graph.AnchorSource("x").Valid() || graph.EdgeKind("x").Valid() || graph.EdgeStatus("x").Valid() {
		t.Error("unknown enum values must be invalid")
	}
	if !graph.EntitySortCreatedAt.Valid() || !graph.EntitySortName.Valid() || graph.EntitySort("x").Valid() {
		t.Error("entity sort validity is wrong")
	}
}
```

`internal/domain/graph/edge_test.go`:

```go
package graph_test

import (
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func validEdge() graph.Edge {
	return graph.Edge{
		ID: "6f6f6f6f-6f6f-4f6f-8f6f-6f6f6f6f6f6f", SiteID: siteA, FromEntityID: entB, ToEntityID: entA,
		Kind: graph.EdgeParent, Weight: 0.3, Source: graph.SourceUser, Status: graph.StatusApproved,
		CreatedAt: time.Date(2026, time.September, 18, 9, 0, 0, 0, time.UTC),
	}
}

func TestNewEdgeNormalises(t *testing.T) {
	t.Parallel()

	parent, err := graph.NewEdge(validEdge())
	if err != nil {
		t.Fatalf("NewEdge: %v", err)
	}
	if parent.Weight != 1 {
		t.Errorf("parent weight = %v, want 1", parent.Weight)
	}
	if parent.FromEntityID != entB || parent.ToEntityID != entA {
		t.Error("a parent edge keeps its direction")
	}

	related := validEdge()
	related.Kind = graph.EdgeRelated
	related.Weight = 0.7
	normalised, err := graph.NewEdge(related)
	if err != nil {
		t.Fatalf("NewEdge: %v", err)
	}
	if normalised.FromEntityID != entA || normalised.ToEntityID != entB {
		t.Errorf("related edge = %s -> %s, want the lower id first", normalised.FromEntityID, normalised.ToEntityID)
	}
	if normalised.Weight != 0.7 {
		t.Errorf("related weight = %v, want 0.7", normalised.Weight)
	}
}

func TestNewEdgeRejects(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		mutate func(*graph.Edge)
		field  string
	}{
		{name: "no id", mutate: func(e *graph.Edge) { e.ID = "" }, field: "id"},
		{name: "no site", mutate: func(e *graph.Edge) { e.SiteID = "" }, field: "siteId"},
		{name: "no from", mutate: func(e *graph.Edge) { e.FromEntityID = "" }, field: "fromEntityId"},
		{name: "self edge", mutate: func(e *graph.Edge) { e.ToEntityID = e.FromEntityID }, field: "toEntityId"},
		{name: "unknown kind", mutate: func(e *graph.Edge) { e.Kind = "cousin" }, field: "kind"},
		{name: "unknown source", mutate: func(e *graph.Edge) { e.Source = "wind" }, field: "source"},
		{name: "unknown status", mutate: func(e *graph.Edge) { e.Status = "maybe" }, field: "status"},
		{name: "weight above one", mutate: func(e *graph.Edge) { e.Kind = graph.EdgeRelated; e.Weight = 1.01 }, field: "weight"},
		{name: "weight below zero", mutate: func(e *graph.Edge) { e.Kind = graph.EdgeRelated; e.Weight = -0.1 }, field: "weight"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			edge := validEdge()
			tc.mutate(&edge)
			_, err := graph.NewEdge(edge)
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("code = %q, want INVALID", errors.CodeOf(err))
			}
			if got := fieldOf(t, err); got != tc.field {
				t.Errorf("field = %q, want %q", got, tc.field)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/domain/graph/ -v`
Expected: FAIL, the package does not exist.

- [ ] **Step 3: Write the implementation**

`internal/domain/graph/entity.go`:

```go
package graph

import (
	"strconv"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type Kind string

const (
	KindHub      Kind = "hub"
	KindProduct  Kind = "product"
	KindTopic    Kind = "topic"
	KindCategory Kind = "category"
	KindCustom   Kind = "custom"
)

func (k Kind) Valid() bool {
	switch k {
	case KindHub, KindProduct, KindTopic, KindCategory, KindCustom:
		return true
	default:
		return false
	}
}

type Source string

const (
	SourceImport Source = "import"
	SourceUser   Source = "user"
	SourceAI     Source = "ai"
)

func (s Source) Valid() bool {
	switch s {
	case SourceImport, SourceUser, SourceAI:
		return true
	default:
		return false
	}
}

type AnchorSource string

const (
	AnchorUser AnchorSource = "user"
	AnchorAI   AnchorSource = "ai"
)

func (s AnchorSource) Valid() bool {
	switch s {
	case AnchorUser, AnchorAI:
		return true
	default:
		return false
	}
}

type Anchor struct {
	Text   string
	Source AnchorSource
	Weight float64
}

type Entity struct {
	ID                string
	SiteID            string
	Name              string
	Kind              Kind
	Intent            string
	PrimaryKeyword    string
	SecondaryKeywords []string
	Anchors           []Anchor
	CanonicalPageID   *string
	Score             float64
	Source            Source
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func invalid(message, field string) *errors.Error {
	return errors.New(errors.Invalid, message).WithDetail("field", field)
}

func NewEntity(e Entity) (Entity, error) {
	e.Name = strings.TrimSpace(e.Name)
	e.Intent = strings.TrimSpace(e.Intent)
	e.PrimaryKeyword = strings.TrimSpace(e.PrimaryKeyword)

	switch {
	case e.ID == "":
		return Entity{}, invalid("entity id must not be empty", "id")
	case e.SiteID == "":
		return Entity{}, invalid("entity site id must not be empty", "siteId")
	case e.Name == "":
		return Entity{}, invalid("entity name must not be empty", "name")
	case !e.Kind.Valid():
		return Entity{}, invalid("entity kind is not recognised", "kind")
	case !e.Source.Valid():
		return Entity{}, invalid("entity source is not recognised", "source")
	case e.Score < 0:
		return Entity{}, invalid("entity score must not be negative", "score")
	case e.CanonicalPageID != nil && *e.CanonicalPageID == "":
		return Entity{}, invalid("canonical page id must not be empty when set", "canonicalPageId")
	}

	e.SecondaryKeywords = cleanKeywords(e.SecondaryKeywords)
	anchors, err := NewAnchors(e.Anchors)
	if err != nil {
		return Entity{}, err
	}
	e.Anchors = anchors
	return e, nil
}

func cleanKeywords(raw []string) []string {
	out := make([]string, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, keyword := range raw {
		trimmed := strings.TrimSpace(keyword)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func NewAnchors(anchors []Anchor) ([]Anchor, error) {
	out := make([]Anchor, 0, len(anchors))
	seen := make(map[string]struct{}, len(anchors))
	for i, anchor := range anchors {
		anchor.Text = strings.TrimSpace(anchor.Text)
		field := "anchors[" + strconv.Itoa(i) + "]"
		switch {
		case anchor.Text == "":
			return nil, invalid("anchor text must not be empty", field+".text")
		case !anchor.Source.Valid():
			return nil, invalid("anchor source is not recognised", field+".source")
		case anchor.Weight < 0 || anchor.Weight > 1:
			return nil, invalid("anchor weight must be between 0 and 1", field+".weight")
		}
		key := strings.ToLower(anchor.Text)
		if _, dup := seen[key]; dup {
			return nil, invalid("anchor text is repeated", field+".text")
		}
		seen[key] = struct{}{}
		out = append(out, anchor)
	}
	return out, nil
}
```

`internal/domain/graph/edge.go`:

```go
package graph

import "time"

type EdgeKind string

const (
	EdgeParent  EdgeKind = "parent"
	EdgeRelated EdgeKind = "related"
)

func (k EdgeKind) Valid() bool {
	switch k {
	case EdgeParent, EdgeRelated:
		return true
	default:
		return false
	}
}

type EdgeStatus string

const (
	StatusApproved EdgeStatus = "approved"
	StatusProposed EdgeStatus = "proposed"
	StatusRejected EdgeStatus = "rejected"
)

func (s EdgeStatus) Valid() bool {
	switch s {
	case StatusApproved, StatusProposed, StatusRejected:
		return true
	default:
		return false
	}
}

type Edge struct {
	ID           string
	SiteID       string
	FromEntityID string
	ToEntityID   string
	Kind         EdgeKind
	Weight       float64
	Source       Source
	Status       EdgeStatus
	CreatedAt    time.Time
}

func NewEdge(e Edge) (Edge, error) {
	switch {
	case e.ID == "":
		return Edge{}, invalid("edge id must not be empty", "id")
	case e.SiteID == "":
		return Edge{}, invalid("edge site id must not be empty", "siteId")
	case e.FromEntityID == "" || e.ToEntityID == "":
		return Edge{}, invalid("edge endpoints must not be empty", "fromEntityId")
	case e.FromEntityID == e.ToEntityID:
		return Edge{}, invalid("edge endpoints must differ", "toEntityId")
	case !e.Kind.Valid():
		return Edge{}, invalid("edge kind is not recognised", "kind")
	case !e.Source.Valid():
		return Edge{}, invalid("edge source is not recognised", "source")
	case !e.Status.Valid():
		return Edge{}, invalid("edge status is not recognised", "status")
	case e.Weight < 0 || e.Weight > 1:
		return Edge{}, invalid("edge weight must be between 0 and 1", "weight")
	}

	if e.Kind == EdgeParent {
		e.Weight = 1
	}
	if e.Kind == EdgeRelated && e.FromEntityID > e.ToEntityID {
		e.FromEntityID, e.ToEntityID = e.ToEntityID, e.FromEntityID
	}
	return e, nil
}
```

`internal/domain/graph/query.go`:

```go
package graph

type EntitySort string

const (
	EntitySortCreatedAt EntitySort = "createdAt"
	EntitySortName      EntitySort = "name"
)

func (s EntitySort) Valid() bool {
	switch s {
	case EntitySortCreatedAt, EntitySortName:
		return true
	default:
		return false
	}
}

type EntityQuery struct {
	SiteID           string
	Kind             *Kind
	HasCanonicalPage *bool
	NamePrefix       string
	Sort             EntitySort
	Desc             bool
}

type EdgeQuery struct {
	SiteID   string
	Kind     *EdgeKind
	Status   *EdgeStatus
	EntityID string
	Desc     bool
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -race ./internal/domain/graph/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/graph/
git commit -m "feat(domain): graph entities, anchors and edges with constructor invariants

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 4: `domain/graph` traversal and cycle detection

**Files:**
- Create: `internal/domain/graph/graph.go`
- Test: `internal/domain/graph/graph_test.go`

**Interfaces:**
- Consumes: Task 3 types, `errors.New`, `(*errors.Error).WithDetail`.
- Produces: `type Graph struct` (unexported fields), `New(entities []Entity, edges []Edge) (Graph, error)`, `Neighbor{Entity Entity; Weight float64}`, `(Graph).Entity(id string) (entity Entity, found bool)`, `(Graph).Entities() []Entity`, `(Graph).Edges() []Edge`, `(Graph).Parents(id string, depth int) []Entity`, `(Graph).Children(id string) []Entity`, `(Graph).Related(id string, minWeight float64) []Neighbor`, `(Graph).Roots() []Entity`, `(Graph).ValidateAcyclic() error`.

- [ ] **Step 1: Write the failing test**

```go
package graph_test

import (
	stderrors "errors"
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

var stamp = time.Date(2026, time.September, 18, 9, 0, 0, 0, time.UTC)

func entity(id, name string) graph.Entity {
	return graph.Entity{ID: id, SiteID: siteA, Name: name, Kind: graph.KindTopic, Source: graph.SourceUser, CreatedAt: stamp, UpdatedAt: stamp}
}

func parent(id, child, parentID string, status graph.EdgeStatus) graph.Edge {
	return graph.Edge{ID: id, SiteID: siteA, FromEntityID: child, ToEntityID: parentID, Kind: graph.EdgeParent, Weight: 1, Source: graph.SourceUser, Status: status, CreatedAt: stamp}
}

func related(id, a, b string, weight float64, status graph.EdgeStatus) graph.Edge {
	return graph.Edge{ID: id, SiteID: siteA, FromEntityID: a, ToEntityID: b, Kind: graph.EdgeRelated, Weight: weight, Source: graph.SourceAI, Status: status, CreatedAt: stamp}
}

func names(entities []graph.Entity) []string {
	out := make([]string, 0, len(entities))
	for _, e := range entities {
		out = append(out, e.Name)
	}
	return out
}

func diamond(t *testing.T, extra ...graph.Edge) graph.Graph {
	t.Helper()
	edges := []graph.Edge{
		parent("e1", entD, entB, graph.StatusApproved),
		parent("e2", entD, entC, graph.StatusApproved),
		parent("e3", entB, entA, graph.StatusApproved),
		parent("e4", entC, entA, graph.StatusApproved),
		parent("e5", entD, entE, graph.StatusProposed),
	}
	g, err := graph.New([]graph.Entity{entity(entA, "Alpha"), entity(entB, "bravo"), entity(entC, "Charlie"), entity(entD, "delta"), entity(entE, "Echo")}, append(edges, extra...))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return g
}

func TestParentsIsBreadthFirstAndDeduplicated(t *testing.T) {
	t.Parallel()

	g := diamond(t)

	cases := []struct {
		name  string
		id    string
		depth int
		want  []string
	}{
		{name: "direct parents sorted by name", id: entD, depth: 1, want: []string{"bravo", "Charlie"}},
		{name: "grandparent appears once", id: entD, depth: 2, want: []string{"bravo", "Charlie", "Alpha"}},
		{name: "depth beyond the graph", id: entD, depth: 9, want: []string{"bravo", "Charlie", "Alpha"}},
		{name: "root has none", id: entA, depth: 3, want: nil},
		{name: "zero depth", id: entD, depth: 0, want: nil},
		{name: "unknown id", id: "nope", depth: 1, want: nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := names(g.Parents(tc.id, tc.depth)); !slices.Equal(got, tc.want) {
				t.Errorf("Parents = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestParentsShortcutMovesTheAncestorToTheNearestLevel(t *testing.T) {
	t.Parallel()

	g := diamond(t, parent("e6", entD, entA, graph.StatusApproved))
	if got := names(g.Parents(entD, 2)); !slices.Equal(got, []string{"Alpha", "bravo", "Charlie"}) {
		t.Errorf("Parents = %v", got)
	}
}

func TestChildrenRootsAndRelated(t *testing.T) {
	t.Parallel()

	g := diamond(t, related("r1", entB, entC, 0.9, graph.StatusApproved), related("r2", entB, entE, 0.2, graph.StatusApproved), related("r3", entA, entB, 0.95, graph.StatusProposed))

	if got := names(g.Children(entA)); !slices.Equal(got, []string{"bravo", "Charlie"}) {
		t.Errorf("Children(A) = %v", got)
	}
	if got := names(g.Children(entD)); len(got) != 0 {
		t.Errorf("Children(D) = %v, want none", got)
	}
	if got := names(g.Roots()); !slices.Equal(got, []string{"Alpha", "Echo"}) {
		t.Errorf("Roots = %v", got)
	}

	neighbors := g.Related(entB, 0)
	if len(neighbors) != 2 || neighbors[0].Entity.Name != "Charlie" || neighbors[0].Weight != 0.9 || neighbors[1].Entity.Name != "Echo" {
		t.Errorf("Related(B, 0) = %+v", neighbors)
	}
	if got := g.Related(entB, 0.5); len(got) != 1 || got[0].Entity.Name != "Charlie" {
		t.Errorf("Related(B, 0.5) = %+v", got)
	}
	if got := g.Related(entC, 0); len(got) != 1 || got[0].Entity.Name != "bravo" {
		t.Errorf("related edges must be visible from both ends, got %+v", got)
	}
}

func TestAccessors(t *testing.T) {
	t.Parallel()

	g := diamond(t)
	if got := names(g.Entities()); !slices.Equal(got, []string{"Alpha", "bravo", "Charlie", "delta", "Echo"}) {
		t.Errorf("Entities = %v", got)
	}
	if len(g.Edges()) != 5 {
		t.Errorf("Edges = %d, want 5 including the proposed one", len(g.Edges()))
	}
	if e, found := g.Entity(entB); !found || e.Name != "bravo" {
		t.Errorf("Entity(B) = %+v, %v", e, found)
	}
	if _, found := g.Entity("nope"); found {
		t.Error("unknown id must not be found")
	}
}

func TestValidateAcyclic(t *testing.T) {
	t.Parallel()

	if err := diamond(t).ValidateAcyclic(); err != nil {
		t.Fatalf("a DAG must validate: %v", err)
	}

	cyclic, err := graph.New(
		[]graph.Entity{entity(entA, "a"), entity(entB, "b"), entity(entC, "c")},
		[]graph.Edge{parent("e1", entA, entB, graph.StatusApproved), parent("e2", entB, entC, graph.StatusApproved), parent("e3", entC, entA, graph.StatusApproved)},
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	err = cyclic.ValidateAcyclic()
	if !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("code = %q, want INVALID", errors.CodeOf(err))
	}
	var kernel *errors.Error
	if !stderrors.As(err, &kernel) {
		t.Fatal("not a kernel error")
	}
	cycle, _ := kernel.Details["cycle"].([]string)
	if !slices.Equal(cycle, []string{entA, entB, entC, entA}) {
		t.Errorf("cycle = %v", cycle)
	}

	proposed, err := graph.New(
		[]graph.Entity{entity(entA, "a"), entity(entB, "b")},
		[]graph.Edge{parent("e1", entA, entB, graph.StatusApproved), parent("e2", entB, entA, graph.StatusProposed)},
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err = proposed.ValidateAcyclic(); err != nil {
		t.Errorf("a proposed edge must not count: %v", err)
	}
}

func TestNewRejectsInconsistentInput(t *testing.T) {
	t.Parallel()

	other := entity(entB, "b")
	other.SiteID = "another-site"

	cases := []struct {
		name     string
		entities []graph.Entity
		edges    []graph.Edge
		field    string
	}{
		{name: "duplicate entity", entities: []graph.Entity{entity(entA, "a"), entity(entA, "a")}, field: "id"},
		{name: "two sites", entities: []graph.Entity{entity(entA, "a"), other}, field: "siteId"},
		{name: "unknown endpoint", entities: []graph.Entity{entity(entA, "a")}, edges: []graph.Edge{parent("e1", entA, entB, graph.StatusApproved)}, field: "toEntityId"},
		{name: "duplicate edge", entities: []graph.Entity{entity(entA, "a"), entity(entB, "b")}, edges: []graph.Edge{parent("e1", entA, entB, graph.StatusApproved), parent("e1", entB, entA, graph.StatusApproved)}, field: "id"},
		{name: "edge from another site", entities: []graph.Entity{entity(entA, "a"), entity(entB, "b")}, edges: []graph.Edge{{ID: "e1", SiteID: "x", FromEntityID: entA, ToEntityID: entB, Kind: graph.EdgeParent, Weight: 1, Source: graph.SourceUser, Status: graph.StatusApproved}}, field: "siteId"},
		{name: "self edge", entities: []graph.Entity{entity(entA, "a")}, edges: []graph.Edge{parent("e1", entA, entA, graph.StatusApproved)}, field: "toEntityId"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := graph.New(tc.entities, tc.edges)
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("code = %q, want INVALID", errors.CodeOf(err))
			}
			if got := fieldOf(t, err); got != tc.field {
				t.Errorf("field = %q, want %q", got, tc.field)
			}
		})
	}
}

func TestEmptyGraph(t *testing.T) {
	t.Parallel()

	g, err := graph.New(nil, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if len(g.Entities()) != 0 || len(g.Roots()) != 0 || g.ValidateAcyclic() != nil {
		t.Error("an empty graph is valid and has nothing in it")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/graph/ -run 'TestParents|TestChildren|TestAccessors|TestValidate|TestNew|TestEmpty' -v`
Expected: FAIL, `graph.New` and `graph.Graph` are undefined.

- [ ] **Step 3: Write the implementation**

`internal/domain/graph/graph.go`:

```go
package graph

import (
	"cmp"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type neighbor struct {
	id     string
	weight float64
}

type Neighbor struct {
	Entity Entity
	Weight float64
}

type Graph struct {
	entities map[string]Entity
	parents  map[string][]string
	children map[string][]string
	related  map[string][]neighbor
	ordered  []Entity
	edges    []Edge
}

func New(entities []Entity, edges []Edge) (Graph, error) {
	g := Graph{
		entities: make(map[string]Entity, len(entities)),
		parents:  make(map[string][]string),
		children: make(map[string][]string),
		related:  make(map[string][]neighbor),
		ordered:  make([]Entity, 0, len(entities)),
		edges:    make([]Edge, 0, len(edges)),
	}

	siteID := ""
	for _, e := range entities {
		if _, dup := g.entities[e.ID]; dup {
			return Graph{}, invalid("entity id is repeated", "id").WithDetail("entityId", e.ID)
		}
		if siteID == "" {
			siteID = e.SiteID
		} else if e.SiteID != siteID {
			return Graph{}, invalid("entities span more than one site", "siteId").WithDetail("entityId", e.ID)
		}
		g.entities[e.ID] = e
		g.ordered = append(g.ordered, e)
	}
	slices.SortFunc(g.ordered, byName)

	seen := make(map[string]struct{}, len(edges))
	for _, raw := range edges {
		e, err := NewEdge(raw)
		if err != nil {
			return Graph{}, err
		}
		if _, dup := seen[e.ID]; dup {
			return Graph{}, invalid("edge id is repeated", "id").WithDetail("edgeId", e.ID)
		}
		if e.SiteID != siteID {
			return Graph{}, invalid("edge belongs to another site", "siteId").WithDetail("edgeId", e.ID)
		}
		if _, ok := g.entities[e.FromEntityID]; !ok {
			return Graph{}, invalid("edge references an unknown entity", "fromEntityId").WithDetail("edgeId", e.ID).WithDetail("entityId", e.FromEntityID)
		}
		if _, ok := g.entities[e.ToEntityID]; !ok {
			return Graph{}, invalid("edge references an unknown entity", "toEntityId").WithDetail("edgeId", e.ID).WithDetail("entityId", e.ToEntityID)
		}
		seen[e.ID] = struct{}{}
		g.edges = append(g.edges, e)
		if e.Status != StatusApproved {
			continue
		}
		switch e.Kind {
		case EdgeParent:
			g.parents[e.FromEntityID] = append(g.parents[e.FromEntityID], e.ToEntityID)
			g.children[e.ToEntityID] = append(g.children[e.ToEntityID], e.FromEntityID)
		case EdgeRelated:
			g.related[e.FromEntityID] = append(g.related[e.FromEntityID], neighbor{id: e.ToEntityID, weight: e.Weight})
			g.related[e.ToEntityID] = append(g.related[e.ToEntityID], neighbor{id: e.FromEntityID, weight: e.Weight})
		}
	}
	return g, nil
}

func byName(a, b Entity) int {
	if c := strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)); c != 0 {
		return c
	}
	return strings.Compare(a.ID, b.ID)
}

func (g Graph) Entity(id string) (entity Entity, found bool) {
	entity, found = g.entities[id]
	return entity, found
}

func (g Graph) Entities() []Entity {
	return slices.Clone(g.ordered)
}

func (g Graph) Edges() []Edge {
	return slices.Clone(g.edges)
}

func (g Graph) collect(ids []string) []Entity {
	out := make([]Entity, 0, len(ids))
	for _, id := range ids {
		out = append(out, g.entities[id])
	}
	slices.SortFunc(out, byName)
	return out
}

func (g Graph) Parents(id string, depth int) []Entity {
	if _, ok := g.entities[id]; !ok || depth <= 0 {
		return nil
	}

	seen := map[string]struct{}{id: {}}
	frontier := []string{id}
	var out []Entity
	for level := 0; level < depth && len(frontier) > 0; level++ {
		next := make([]string, 0)
		for _, current := range frontier {
			for _, parentID := range g.parents[current] {
				if _, dup := seen[parentID]; dup {
					continue
				}
				seen[parentID] = struct{}{}
				next = append(next, parentID)
			}
		}
		out = append(out, g.collect(next)...)
		frontier = next
	}
	return out
}

func (g Graph) Children(id string) []Entity {
	return g.collect(g.children[id])
}

func (g Graph) Related(id string, minWeight float64) []Neighbor {
	out := make([]Neighbor, 0, len(g.related[id]))
	for _, n := range g.related[id] {
		if n.weight < minWeight {
			continue
		}
		out = append(out, Neighbor{Entity: g.entities[n.id], Weight: n.weight})
	}
	slices.SortFunc(out, func(a, b Neighbor) int {
		if a.Weight != b.Weight {
			return cmp.Compare(b.Weight, a.Weight)
		}
		return byName(a.Entity, b.Entity)
	})
	return out
}

func (g Graph) Roots() []Entity {
	out := make([]Entity, 0, len(g.ordered))
	for _, e := range g.ordered {
		if len(g.parents[e.ID]) == 0 {
			out = append(out, e)
		}
	}
	return out
}

func (g Graph) sortedParents(id string) []string {
	ids := slices.Clone(g.parents[id])
	slices.Sort(ids)
	return ids
}

func (g Graph) ValidateAcyclic() error {
	const (
		unvisited = iota
		active
		done
	)

	state := make(map[string]int, len(g.ordered))
	stack := make([]string, 0, len(g.ordered))

	var visit func(id string) []string
	visit = func(id string) []string {
		state[id] = active
		stack = append(stack, id)
		for _, parentID := range g.sortedParents(id) {
			switch state[parentID] {
			case active:
				start := slices.Index(stack, parentID)
				return append(slices.Clone(stack[start:]), parentID)
			case unvisited:
				if cycle := visit(parentID); cycle != nil {
					return cycle
				}
			}
		}
		stack = stack[:len(stack)-1]
		state[id] = done
		return nil
	}

	for _, e := range g.ordered {
		if state[e.ID] != unvisited {
			continue
		}
		if cycle := visit(e.ID); cycle != nil {
			return errors.New(errors.Invalid, "parent edges form a cycle").WithDetail("cycle", cycle)
		}
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race ./internal/domain/graph/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/graph/graph.go internal/domain/graph/graph_test.go
git commit -m "feat(domain): graph value type with breadth-first parents and cycle detection

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 5: `domain/graph` PageRank score

**Files:**
- Create: `internal/domain/graph/score.go`
- Test: `internal/domain/graph/score_test.go`

**Interfaces:**
- Consumes: `Graph` internals from Task 4.
- Produces: `(Graph).Score() map[string]float64`.

- [ ] **Step 1: Write the failing test**

```go
package graph_test

import (
	"math"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/graph"
)

func TestScoreOrdersAHubAboveItsDescendants(t *testing.T) {
	t.Parallel()

	g, err := graph.New(
		[]graph.Entity{entity(entA, "Hub"), entity(entB, "Mid"), entity(entC, "Sibling"), entity(entD, "Leaf"), entity(entE, "Grandchild")},
		[]graph.Edge{
			parent("e1", entB, entA, graph.StatusApproved),
			parent("e2", entC, entA, graph.StatusApproved),
			parent("e3", entD, entA, graph.StatusApproved),
			parent("e4", entE, entB, graph.StatusApproved),
			related("r1", entB, entC, 0.8, graph.StatusApproved),
			parent("e5", entA, entD, graph.StatusRejected),
		},
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	scores := g.Score()
	if len(scores) != 5 {
		t.Fatalf("Score has %d entries, want 5", len(scores))
	}
	if scores[entA] != 1 {
		t.Errorf("hub = %v, want exactly 1", scores[entA])
	}
	if !(scores[entA] > scores[entB] && scores[entB] > scores[entC] && scores[entC] > scores[entD]) {
		t.Errorf("order is wrong: hub %v mid %v sibling %v leaf %v", scores[entA], scores[entB], scores[entC], scores[entD])
	}
	if math.Abs(scores[entD]-scores[entE]) > 1e-12 {
		t.Errorf("nodes without incoming links must score alike: leaf %v grandchild %v", scores[entD], scores[entE])
	}
	for id, score := range scores {
		if score <= 0 || score > 1 {
			t.Errorf("%s = %v, want within (0, 1]", id, score)
		}
	}
}

func TestScoreTreatsRelatedEdgesSymmetrically(t *testing.T) {
	t.Parallel()

	g, err := graph.New(
		[]graph.Entity{entity(entA, "a"), entity(entB, "b")},
		[]graph.Edge{related("r1", entA, entB, 0.1, graph.StatusApproved)},
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	scores := g.Score()
	if scores[entA] != 1 || scores[entB] != 1 {
		t.Errorf("symmetric graph must score both ends 1, got %v", scores)
	}
}

func TestScoreIgnoresProposedEdgesAndHandlesEmptyGraphs(t *testing.T) {
	t.Parallel()

	g, err := graph.New(
		[]graph.Entity{entity(entA, "a"), entity(entB, "b")},
		[]graph.Edge{parent("e1", entA, entB, graph.StatusProposed)},
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if scores := g.Score(); scores[entA] != 1 || scores[entB] != 1 {
		t.Errorf("without approved edges every node scores 1, got %v", scores)
	}

	empty, err := graph.New(nil, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if scores := empty.Score(); len(scores) != 0 {
		t.Errorf("empty graph scores = %v", scores)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/graph/ -run TestScore -v`
Expected: FAIL, `Score` is undefined.

- [ ] **Step 3: Write the implementation**

`internal/domain/graph/score.go`:

```go
package graph

const (
	damping    = 0.85
	iterations = 30
)

func (g Graph) Score() map[string]float64 {
	n := len(g.ordered)
	scores := make(map[string]float64, n)
	if n == 0 {
		return scores
	}

	index := make(map[string]int, n)
	for i, e := range g.ordered {
		index[e.ID] = i
	}

	out := make([][]int, n)
	for _, e := range g.edges {
		if e.Status != StatusApproved {
			continue
		}
		from, to := index[e.FromEntityID], index[e.ToEntityID]
		out[from] = append(out[from], to)
		if e.Kind == EdgeRelated {
			out[to] = append(out[to], from)
		}
	}

	size := float64(n)
	rank := make([]float64, n)
	for i := range rank {
		rank[i] = 1 / size
	}

	for range iterations {
		next := make([]float64, n)
		dangling := 0.0
		for i, current := range rank {
			if len(out[i]) == 0 {
				dangling += current
				continue
			}
			share := current / float64(len(out[i]))
			for _, target := range out[i] {
				next[target] += share
			}
		}
		for i := range next {
			next[i] = (1-damping)/size + damping*(next[i]+dangling/size)
		}
		rank = next
	}

	highest := 0.0
	for _, value := range rank {
		highest = max(highest, value)
	}
	for i, e := range g.ordered {
		scores[e.ID] = rank[i] / highest
	}
	return scores
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race ./internal/domain/graph/ -v`
Expected: PASS. Run `go test -race -cover ./internal/domain/graph/` and confirm ≥ 90%.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/graph/score.go internal/domain/graph/score_test.go
git commit -m "feat(domain): pagerank entity scores over approved edges

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---
### Task 6: `domain/pagemap` paths, pages, tree and index

**Files:**
- Create: `internal/domain/pagemap/path.go`, `internal/domain/pagemap/page.go`, `internal/domain/pagemap/tree.go`, `internal/domain/pagemap/index.go`, `internal/domain/pagemap/query.go`
- Test: `internal/domain/pagemap/path_test.go`, `internal/domain/pagemap/page_test.go`, `internal/domain/pagemap/tree_test.go`, `internal/domain/pagemap/index_test.go`

**Interfaces:**
- Consumes: `errors.New`, `(*errors.Error).WithDetail|WithInternal`.
- Produces: `NormalizePath(raw string) (string, error)`, `ParentPath(path string) string` (`""` for the root), `Slug(path string) string`, `InternalPath(href, siteHost string) (path string, internal bool)`; `WPType` (`WPPage|WPPost|WPProduct|WPProductCategory`), `Status` (`StatusPlanned|StatusExists|StatusPublished|StatusArchived`), `LinkOrigin` (`OriginGenerated|OriginObserved`), each with `Valid()`; `Page{ID, SiteID, Path, Slug string; ParentPageID *string; WPType WPType; WPID *int64; Title, H1, MetaTitle, MetaDescription, Canonical string; Status Status; EntityID, TemplateID *string; ContentHash string; WPModifiedAt, LastSyncedAt *time.Time; Drift bool; CreatedAt, UpdatedAt time.Time}`; `PageLink{ID, SiteID, FromPageID string; ToPageID *string; ToURL, AnchorText string; Origin LinkOrigin; ObservedAt time.Time}`; `NewPage(Page) (Page, error)`, `NewPageLink(PageLink) (PageLink, error)`, `Unmapped([]Page) []Page`; `Node{Page Page; Children []Node}`, `BuildTree([]Page) []Node`; `Index`, `NewIndex([]Page) Index`, `(Index).ByID(id string) (page Page, found bool)`, `(Index).ByPath(path string) (page Page, found bool)`, `(Index).ByEntity(entityID string) []Page`, `(Index).Pages() []Page`, `(Index).Len() int`; `Sort` (`SortCreatedAt = "createdAt"`, `SortPath = "path"`), `Query{SiteID string; Status *Status; EntityID *string; Unmapped bool; PathPrefix string; Sort Sort; Desc bool}`.

- [ ] **Step 1: Write the failing tests**

`internal/domain/pagemap/path_test.go`:

```go
package pagemap_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestNormalizePath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		raw  string
		want string
		fail bool
	}{
		{name: "root", raw: "/", want: "/"},
		{name: "bare word", raw: "about", want: "/about/"},
		{name: "lowercased", raw: "/About/Us/", want: "/about/us/"},
		{name: "full url loses host query and fragment", raw: "https://Example.com/Shop/Bags?x=1#top", want: "/shop/bags/"},
		{name: "duplicate slashes collapse", raw: "//a///b//", want: "/a/b/"},
		{name: "query on a bare path", raw: "/a/?page=2", want: "/a/"},
		{name: "surrounding whitespace trimmed", raw: "  /a/b  ", want: "/a/b/"},
		{name: "unicode kept", raw: "/Обувь/", want: "/обувь/"},
		{name: "percent encoding kept", raw: "/a%20b/", want: "/a%20b/"},
		{name: "escape case is lowered", raw: "/a%2Fb/", want: "/a%2fb/"},
		{name: "no file extension exception", raw: "/Shop/index.HTML", want: "/shop/index.html/"},
		{name: "absolute url with no path is the root", raw: "https://example.com", want: "/"},
		{name: "koffein example", raw: "https://example.com/Koffein/Powder", want: "/koffein/powder/"},
		{name: "empty", raw: "   ", fail: true},
		{name: "inner whitespace", raw: "/a b/", fail: true},
		{name: "dot segment", raw: "/a/./b/", fail: true},
		{name: "parent segment", raw: "/a/../b/", fail: true},
		{name: "control character", raw: "/a\tb/", fail: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := pagemap.NormalizePath(tc.raw)
			if tc.fail {
				if !errors.IsCode(err, errors.Invalid) {
					t.Fatalf("code = %q, want INVALID (got %q)", errors.CodeOf(err), got)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizePath: %v", err)
			}
			if got != tc.want {
				t.Errorf("NormalizePath = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParentPathAndSlug(t *testing.T) {
	t.Parallel()

	cases := []struct {
		path   string
		parent string
		slug   string
	}{
		{path: "/", parent: "", slug: ""},
		{path: "/a/", parent: "/", slug: "a"},
		{path: "/a/b/", parent: "/a/", slug: "b"},
		{path: "/a/b/c/", parent: "/a/b/", slug: "c"},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			if got := pagemap.ParentPath(tc.path); got != tc.parent {
				t.Errorf("ParentPath = %q, want %q", got, tc.parent)
			}
			if got := pagemap.Slug(tc.path); got != tc.slug {
				t.Errorf("Slug = %q, want %q", got, tc.slug)
			}
		})
	}
}

func TestParentPathBindingExamples(t *testing.T) {
	t.Parallel()

	if got := pagemap.ParentPath("/koffein/powder/"); got != "/koffein/" {
		t.Errorf("ParentPath = %q, want /koffein/", got)
	}
	if got := pagemap.ParentPath("/"); got != "" {
		t.Errorf("ParentPath of the root = %q, want empty", got)
	}
}

func TestInternalPath(t *testing.T) {
	t.Parallel()

	const host = "shop.example.com"
	cases := []struct {
		name     string
		href     string
		path     string
		internal bool
	}{
		{name: "relative path", href: "/Shop/Bags?x=1#top", path: "/shop/bags/", internal: true},
		{name: "relative without leading slash", href: "shoes/", path: "/shoes/", internal: true},
		{name: "same host any case", href: "https://Shop.Example.com/Sale/", path: "/sale/", internal: true},
		{name: "scheme ignored", href: "http://shop.example.com/sale/", path: "/sale/", internal: true},
		{name: "absolute url with no path", href: "https://shop.example.com", path: "/", internal: true},
		{name: "network path reference", href: "//shop.example.com/x/", path: "/x/", internal: true},
		{name: "same document", href: "#top", path: "", internal: true},
		{name: "query only", href: "?page=2", path: "", internal: true},
		{name: "www is another host", href: "https://www.shop.example.com/x/", path: "", internal: false},
		{name: "port is part of the host", href: "https://shop.example.com:8443/x/", path: "", internal: false},
		{name: "foreign host", href: "https://other.example.com/x/", path: "", internal: false},
		{name: "mailto", href: "mailto:hello@shop.example.com", path: "", internal: false},
		{name: "javascript", href: "javascript:void(0)", path: "", internal: false},
		{name: "control character", href: "/a\tb/", path: "", internal: false},
		{name: "dot segments", href: "/a/../b/", path: "", internal: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			path, internal := pagemap.InternalPath(tc.href, host)
			if path != tc.path || internal != tc.internal {
				t.Errorf("InternalPath = %q, %v; want %q, %v", path, internal, tc.path, tc.internal)
			}
		})
	}
}
```

`internal/domain/pagemap/page_test.go`:

```go
package pagemap_test

import (
	stderrors "errors"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	siteA = "0b6c2a4e-1f3d-4c8b-9a2e-5d7f8e9a0b1c"
	pageA = "aaaaaaaa-1111-4aaa-8aaa-aaaaaaaaaaaa"
	pageB = "bbbbbbbb-2222-4bbb-8bbb-bbbbbbbbbbbb"
	pageC = "cccccccc-3333-4ccc-8ccc-cccccccccccc"
	pageD = "dddddddd-4444-4ddd-8ddd-dddddddddddd"
	entA  = "1a1a1a1a-1a1a-4a1a-8a1a-1a1a1a1a1a1a"
	entB  = "2b2b2b2b-2b2b-4b2b-8b2b-2b2b2b2b2b2b"
)

var stamp = time.Date(2026, time.September, 18, 9, 0, 0, 0, time.UTC)

func page(id, path string, entityID *string) pagemap.Page {
	return pagemap.Page{ID: id, SiteID: siteA, Path: path, Slug: pagemap.Slug(path), WPType: pagemap.WPPage, Status: pagemap.StatusPlanned, EntityID: entityID, CreatedAt: stamp, UpdatedAt: stamp}
}

func ptr(s string) *string {
	return &s
}

func fieldOf(t *testing.T, err error) string {
	t.Helper()
	var kernel *errors.Error
	if !stderrors.As(err, &kernel) {
		t.Fatalf("error %v is not a kernel error", err)
	}
	field, _ := kernel.Details["field"].(string)
	return field
}

func TestNewPageNormalises(t *testing.T) {
	t.Parallel()

	p := page(pageA, "/Shop/Bags", nil)
	p.Title = "  Bags  "
	got, err := pagemap.NewPage(p)
	if err != nil {
		t.Fatalf("NewPage: %v", err)
	}
	if got.Path != "/shop/bags/" || got.Slug != "bags" || got.Title != "Bags" {
		t.Errorf("NewPage = path %q slug %q title %q", got.Path, got.Slug, got.Title)
	}
}

func TestNewPageRejects(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		mutate func(*pagemap.Page)
		field  string
	}{
		{name: "no id", mutate: func(p *pagemap.Page) { p.ID = "" }, field: "id"},
		{name: "no site", mutate: func(p *pagemap.Page) { p.SiteID = "" }, field: "siteId"},
		{name: "bad path", mutate: func(p *pagemap.Page) { p.Path = "/a b/" }, field: "path"},
		{name: "unknown wp type", mutate: func(p *pagemap.Page) { p.WPType = "widget" }, field: "wpType"},
		{name: "unknown status", mutate: func(p *pagemap.Page) { p.Status = "lost" }, field: "status"},
		{name: "own parent", mutate: func(p *pagemap.Page) { p.ParentPageID = ptr(p.ID) }, field: "parentPageId"},
		{name: "empty entity", mutate: func(p *pagemap.Page) { p.EntityID = ptr("") }, field: "entityId"},
		{name: "empty template", mutate: func(p *pagemap.Page) { p.TemplateID = ptr("") }, field: "templateId"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := page(pageA, "/a/", nil)
			tc.mutate(&p)
			_, err := pagemap.NewPage(p)
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("code = %q, want INVALID", errors.CodeOf(err))
			}
			if got := fieldOf(t, err); got != tc.field {
				t.Errorf("field = %q, want %q", got, tc.field)
			}
		})
	}
}

func TestNewPageLink(t *testing.T) {
	t.Parallel()

	valid := pagemap.PageLink{ID: "l1", SiteID: siteA, FromPageID: pageA, ToPageID: ptr(pageB), ToURL: " https://a/b/ ", AnchorText: " bags ", Origin: pagemap.OriginGenerated, ObservedAt: stamp}
	link, err := pagemap.NewPageLink(valid)
	if err != nil {
		t.Fatalf("NewPageLink: %v", err)
	}
	if link.ToURL != "https://a/b/" || link.AnchorText != "bags" {
		t.Errorf("NewPageLink did not trim: %+v", link)
	}

	cases := []struct {
		name   string
		mutate func(*pagemap.PageLink)
		field  string
	}{
		{name: "no id", mutate: func(l *pagemap.PageLink) { l.ID = "" }, field: "id"},
		{name: "no site", mutate: func(l *pagemap.PageLink) { l.SiteID = "" }, field: "siteId"},
		{name: "no source page", mutate: func(l *pagemap.PageLink) { l.FromPageID = "" }, field: "fromPageId"},
		{name: "empty target page", mutate: func(l *pagemap.PageLink) { l.ToPageID = ptr("") }, field: "toPageId"},
		{name: "no target at all", mutate: func(l *pagemap.PageLink) { l.ToPageID = nil; l.ToURL = " " }, field: "toUrl"},
		{name: "unknown origin", mutate: func(l *pagemap.PageLink) { l.Origin = "guessed" }, field: "origin"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			l := valid
			tc.mutate(&l)
			_, err := pagemap.NewPageLink(l)
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("code = %q, want INVALID", errors.CodeOf(err))
			}
			if got := fieldOf(t, err); got != tc.field {
				t.Errorf("field = %q, want %q", got, tc.field)
			}
		})
	}
}

func TestUnmapped(t *testing.T) {
	t.Parallel()

	pages := []pagemap.Page{page(pageC, "/c/", nil), page(pageA, "/a/", ptr(entA)), page(pageB, "/b/", nil)}
	got := pagemap.Unmapped(pages)
	if len(got) != 2 || got[0].Path != "/b/" || got[1].Path != "/c/" {
		t.Errorf("Unmapped = %+v", got)
	}
}

func TestEnums(t *testing.T) {
	t.Parallel()

	for _, wp := range []pagemap.WPType{pagemap.WPPage, pagemap.WPPost, pagemap.WPProduct, pagemap.WPProductCategory} {
		if !wp.Valid() {
			t.Errorf("%q must be valid", wp)
		}
	}
	for _, status := range []pagemap.Status{pagemap.StatusPlanned, pagemap.StatusExists, pagemap.StatusPublished, pagemap.StatusArchived} {
		if !status.Valid() {
			t.Errorf("%q must be valid", status)
		}
	}
	if pagemap.WPType("x").Valid() || pagemap.Status("x").Valid() || pagemap.LinkOrigin("x").Valid() {
		t.Error("unknown enum values must be invalid")
	}
	if !pagemap.OriginGenerated.Valid() || !pagemap.OriginObserved.Valid() {
		t.Error("origins must be valid")
	}
	if !pagemap.SortCreatedAt.Valid() || !pagemap.SortPath.Valid() || pagemap.Sort("x").Valid() {
		t.Error("sort validity is wrong")
	}
}
```

`internal/domain/pagemap/index_test.go`:

```go
package pagemap_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func TestIndex(t *testing.T) {
	t.Parallel()

	index := pagemap.NewIndex([]pagemap.Page{
		page(pageC, "/shop/bags/", ptr(entA)),
		page(pageA, "/", nil),
		page(pageB, "/shop/", ptr(entA)),
		page(pageD, "/blog/", ptr(entB)),
	})

	if index.Len() != 4 {
		t.Fatalf("Len = %d", index.Len())
	}
	if p, found := index.ByID(pageB); !found || p.Path != "/shop/" {
		t.Errorf("ByID = %+v, %v", p, found)
	}
	if _, found := index.ByID("nope"); found {
		t.Error("unknown id found")
	}
	if p, found := index.ByPath("/Shop/Bags"); !found || p.ID != pageC {
		t.Errorf("ByPath must normalise its input, got %+v, %v", p, found)
	}
	if _, found := index.ByPath("/a b/"); found {
		t.Error("an invalid path must not be found")
	}
	byEntity := index.ByEntity(entA)
	if len(byEntity) != 2 || byEntity[0].Path != "/shop/" || byEntity[1].Path != "/shop/bags/" {
		t.Errorf("ByEntity = %+v", byEntity)
	}
	if len(index.ByEntity("nope")) != 0 {
		t.Error("unknown entity must have no pages")
	}
	pages := index.Pages()
	if len(pages) != 4 || pages[0].Path != "/" || pages[1].Path != "/blog/" || pages[3].Path != "/shop/bags/" {
		t.Errorf("Pages = %+v", pages)
	}
}
```

`internal/domain/pagemap/tree_test.go`:

```go
package pagemap_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func TestBuildTreeAttachesToTheNearestExistingAncestor(t *testing.T) {
	t.Parallel()

	tree := pagemap.BuildTree([]pagemap.Page{
		page(pageD, "/shop/bags/leather/", nil),
		page(pageA, "/shop/", nil),
		page(pageB, "/blog/", nil),
		page(pageC, "/shop/shoes/", nil),
	})

	if len(tree) != 2 || tree[0].Page.Path != "/blog/" || tree[1].Page.Path != "/shop/" {
		t.Fatalf("roots = %+v", tree)
	}
	shop := tree[1]
	if len(shop.Children) != 2 || shop.Children[0].Page.Path != "/shop/bags/leather/" || shop.Children[1].Page.Path != "/shop/shoes/" {
		t.Errorf("shop children = %+v", shop.Children)
	}
	if len(tree[0].Children) != 0 {
		t.Errorf("blog children = %+v", tree[0].Children)
	}
}

func TestBuildTreeWithARootPage(t *testing.T) {
	t.Parallel()

	tree := pagemap.BuildTree([]pagemap.Page{page(pageA, "/", nil), page(pageB, "/a/", nil), page(pageC, "/a/b/", nil)})
	if len(tree) != 1 || tree[0].Page.Path != "/" || len(tree[0].Children) != 1 || len(tree[0].Children[0].Children) != 1 {
		t.Errorf("tree = %+v", tree)
	}
	if len(pagemap.BuildTree(nil)) != 0 {
		t.Error("no pages, no tree")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/domain/pagemap/ -v`
Expected: FAIL, the package does not exist.

- [ ] **Step 3: Write the implementation**

`internal/domain/pagemap/path.go`:

```go
package pagemap

import (
	"net/url"
	"strings"
	"unicode"
)

func NormalizePath(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", invalid("path must not be empty", "path")
	}

	if strings.Contains(trimmed, "://") {
		parsed, err := url.Parse(trimmed)
		if err != nil {
			return "", invalid("path is not a valid url", "path").WithInternal(err)
		}
		trimmed = parsed.EscapedPath()
	} else {
		trimmed, _, _ = strings.Cut(trimmed, "?")
		trimmed, _, _ = strings.Cut(trimmed, "#")
	}

	for _, r := range trimmed {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return "", invalid("path must not contain whitespace or control characters", "path")
		}
	}

	var builder strings.Builder
	builder.WriteByte('/')
	for _, segment := range strings.Split(trimmed, "/") {
		switch segment {
		case "":
			continue
		case ".", "..":
			return "", invalid("path must not contain dot segments", "path")
		}
		builder.WriteString(strings.ToLower(segment))
		builder.WriteByte('/')
	}
	return builder.String(), nil
}

func InternalPath(href, siteHost string) (path string, internal bool) {
	parsed, err := url.Parse(strings.TrimSpace(href))
	if err != nil {
		return "", false
	}
	if parsed.Scheme != "" && parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", false
	}
	if parsed.Host != "" && strings.ToLower(parsed.Host) != strings.ToLower(siteHost) {
		return "", false
	}

	escaped := parsed.EscapedPath()
	if escaped == "" {
		if parsed.Host == "" {
			return "", true
		}
		return "/", true
	}
	normalized, err := NormalizePath(escaped)
	if err != nil {
		return "", false
	}
	return normalized, true
}

func ParentPath(path string) string {
	if path == "" || path == "/" {
		return ""
	}
	trimmed := strings.TrimSuffix(path, "/")
	return trimmed[:strings.LastIndex(trimmed, "/")+1]
}

func Slug(path string) string {
	trimmed := strings.TrimSuffix(path, "/")
	return trimmed[strings.LastIndex(trimmed, "/")+1:]
}
```

`internal/domain/pagemap/page.go`:

```go
package pagemap

import (
	"slices"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type WPType string

const (
	WPPage            WPType = "page"
	WPPost            WPType = "post"
	WPProduct         WPType = "product"
	WPProductCategory WPType = "product_cat"
)

func (t WPType) Valid() bool {
	switch t {
	case WPPage, WPPost, WPProduct, WPProductCategory:
		return true
	default:
		return false
	}
}

type Status string

const (
	StatusPlanned   Status = "planned"
	StatusExists    Status = "exists"
	StatusPublished Status = "published"
	StatusArchived  Status = "archived"
)

func (s Status) Valid() bool {
	switch s {
	case StatusPlanned, StatusExists, StatusPublished, StatusArchived:
		return true
	default:
		return false
	}
}

type LinkOrigin string

const (
	OriginGenerated LinkOrigin = "generated"
	OriginObserved  LinkOrigin = "observed"
)

func (o LinkOrigin) Valid() bool {
	switch o {
	case OriginGenerated, OriginObserved:
		return true
	default:
		return false
	}
}

type Page struct {
	ID              string
	SiteID          string
	Path            string
	Slug            string
	ParentPageID    *string
	WPType          WPType
	WPID            *int64
	Title           string
	H1              string
	MetaTitle       string
	MetaDescription string
	Canonical       string
	Status          Status
	EntityID        *string
	TemplateID      *string
	ContentHash     string
	WPModifiedAt    *time.Time
	LastSyncedAt    *time.Time
	Drift           bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type PageLink struct {
	ID         string
	SiteID     string
	FromPageID string
	ToPageID   *string
	ToURL      string
	AnchorText string
	Origin     LinkOrigin
	ObservedAt time.Time
}

func invalid(message, field string) *errors.Error {
	return errors.New(errors.Invalid, message).WithDetail("field", field)
}

func emptyRef(ref *string) bool {
	return ref != nil && *ref == ""
}

func NewPage(p Page) (Page, error) {
	switch {
	case p.ID == "":
		return Page{}, invalid("page id must not be empty", "id")
	case p.SiteID == "":
		return Page{}, invalid("page site id must not be empty", "siteId")
	case !p.WPType.Valid():
		return Page{}, invalid("page wordpress type is not recognised", "wpType")
	case !p.Status.Valid():
		return Page{}, invalid("page status is not recognised", "status")
	case emptyRef(p.ParentPageID):
		return Page{}, invalid("parent page id must not be empty when set", "parentPageId")
	case p.ParentPageID != nil && *p.ParentPageID == p.ID:
		return Page{}, invalid("page cannot be its own parent", "parentPageId")
	case emptyRef(p.EntityID):
		return Page{}, invalid("entity id must not be empty when set", "entityId")
	case emptyRef(p.TemplateID):
		return Page{}, invalid("template id must not be empty when set", "templateId")
	}

	path, err := NormalizePath(p.Path)
	if err != nil {
		return Page{}, err
	}
	p.Path = path
	p.Slug = Slug(path)
	p.Title = strings.TrimSpace(p.Title)
	p.H1 = strings.TrimSpace(p.H1)
	p.MetaTitle = strings.TrimSpace(p.MetaTitle)
	p.MetaDescription = strings.TrimSpace(p.MetaDescription)
	p.Canonical = strings.TrimSpace(p.Canonical)
	return p, nil
}

func NewPageLink(l PageLink) (PageLink, error) {
	l.ToURL = strings.TrimSpace(l.ToURL)
	l.AnchorText = strings.TrimSpace(l.AnchorText)
	switch {
	case l.ID == "":
		return PageLink{}, invalid("link id must not be empty", "id")
	case l.SiteID == "":
		return PageLink{}, invalid("link site id must not be empty", "siteId")
	case l.FromPageID == "":
		return PageLink{}, invalid("link source page must not be empty", "fromPageId")
	case emptyRef(l.ToPageID):
		return PageLink{}, invalid("target page id must not be empty when set", "toPageId")
	case l.ToPageID == nil && l.ToURL == "":
		return PageLink{}, invalid("link needs a target page or a target url", "toUrl")
	case !l.Origin.Valid():
		return PageLink{}, invalid("link origin is not recognised", "origin")
	}
	return l, nil
}

func byPath(a, b Page) int {
	if c := strings.Compare(a.Path, b.Path); c != 0 {
		return c
	}
	return strings.Compare(a.ID, b.ID)
}

func Unmapped(pages []Page) []Page {
	out := make([]Page, 0, len(pages))
	for _, p := range pages {
		if p.EntityID == nil {
			out = append(out, p)
		}
	}
	slices.SortFunc(out, byPath)
	return out
}
```

`internal/domain/pagemap/index.go`:

```go
package pagemap

import "slices"

type Index struct {
	byID     map[string]Page
	byPath   map[string]Page
	byEntity map[string][]Page
	pages    []Page
}

func NewIndex(pages []Page) Index {
	sorted := slices.Clone(pages)
	slices.SortFunc(sorted, byPath)

	index := Index{
		byID:     make(map[string]Page, len(sorted)),
		byPath:   make(map[string]Page, len(sorted)),
		byEntity: make(map[string][]Page),
		pages:    sorted,
	}
	for _, p := range sorted {
		index.byID[p.ID] = p
		index.byPath[p.Path] = p
		if p.EntityID != nil {
			index.byEntity[*p.EntityID] = append(index.byEntity[*p.EntityID], p)
		}
	}
	return index
}

func (i Index) ByID(id string) (page Page, found bool) {
	page, found = i.byID[id]
	return page, found
}

func (i Index) ByPath(path string) (page Page, found bool) {
	normalized, err := NormalizePath(path)
	if err != nil {
		return Page{}, false
	}
	page, found = i.byPath[normalized]
	return page, found
}

func (i Index) ByEntity(entityID string) []Page {
	return slices.Clone(i.byEntity[entityID])
}

func (i Index) Pages() []Page {
	return slices.Clone(i.pages)
}

func (i Index) Len() int {
	return len(i.pages)
}
```

`internal/domain/pagemap/tree.go`:

```go
package pagemap

type Node struct {
	Page     Page
	Children []Node
}

func BuildTree(pages []Page) []Node {
	index := NewIndex(pages)
	children := make(map[string][]Page)
	roots := make([]Page, 0)
	for _, p := range index.Pages() {
		parent, found := nearestAncestor(p.Path, index)
		if !found {
			roots = append(roots, p)
			continue
		}
		children[parent.ID] = append(children[parent.ID], p)
	}
	return nodes(roots, children)
}

func nearestAncestor(path string, index Index) (parent Page, found bool) {
	for ancestor := ParentPath(path); ancestor != ""; ancestor = ParentPath(ancestor) {
		if p, ok := index.byPath[ancestor]; ok {
			return p, true
		}
	}
	return Page{}, false
}

func nodes(pages []Page, children map[string][]Page) []Node {
	out := make([]Node, 0, len(pages))
	for _, p := range pages {
		out = append(out, Node{Page: p, Children: nodes(children[p.ID], children)})
	}
	return out
}
```

`internal/domain/pagemap/query.go`:

```go
package pagemap

type Sort string

const (
	SortCreatedAt Sort = "createdAt"
	SortPath      Sort = "path"
)

func (s Sort) Valid() bool {
	switch s {
	case SortCreatedAt, SortPath:
		return true
	default:
		return false
	}
}

type Query struct {
	SiteID     string
	Status     *Status
	EntityID   *string
	Unmapped   bool
	PathPrefix string
	Sort       Sort
	Desc       bool
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -race ./internal/domain/pagemap/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/pagemap/
git commit -m "feat(domain): page map with normalised paths, path tree and index

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 7: `domain/pagemap` cannibalization verdict

**Files:**
- Create: `internal/domain/pagemap/cannibalization.go`
- Test: `internal/domain/pagemap/cannibalization_test.go`

**Interfaces:**
- Consumes: `Index` (Task 6), `graph.Entity`, `graph.Graph`, `(graph.Graph).Entities()` (Tasks 3–4).
- Produces: `Reason` (`ReasonSamePrimaryKeyword = "same_primary_keyword"`, `ReasonSameEntityCanonical = "same_entity_canonical"`, `ReasonPathConflict = "path_conflict"`), `Evidence{PageID, Path string; Reason Reason; EntityID string}`, `Verdict{Allowed bool; Evidence []Evidence}`, `Cannibalization(candidate Page, entity graph.Entity, index Index, g graph.Graph) Verdict`.

- [ ] **Step 1: Write the failing test**

```go
package pagemap_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func entityWith(id, keyword string, canonical *string) graph.Entity {
	return graph.Entity{ID: id, SiteID: siteA, Name: id, Kind: graph.KindTopic, PrimaryKeyword: keyword, CanonicalPageID: canonical, Source: graph.SourceUser, CreatedAt: stamp, UpdatedAt: stamp}
}

func TestCannibalization(t *testing.T) {
	t.Parallel()

	index := pagemap.NewIndex([]pagemap.Page{
		page(pageA, "/shoes/", ptr(entA)),
		page(pageB, "/boots/", ptr(entB)),
	})
	owner := entityWith(entA, "Running Shoes", ptr(pageA))
	rival := entityWith(entB, "running shoes", ptr(pageB))
	g, err := graph.New([]graph.Entity{owner, rival}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	cases := []struct {
		name      string
		candidate pagemap.Page
		entity    graph.Entity
		reasons   []pagemap.Reason
		pages     []string
	}{
		{
			name:      "fresh page for a fresh entity",
			candidate: page(pageC, "/sandals/", nil),
			entity:    entityWith("e-new", "sandals", nil),
		},
		{
			name:      "unmapped page needs only a free path",
			candidate: page(pageC, "/sandals/", nil),
			entity:    graph.Entity{},
		},
		{
			name:      "path already taken",
			candidate: page(pageC, "/Shoes", nil),
			entity:    graph.Entity{},
			reasons:   []pagemap.Reason{pagemap.ReasonPathConflict},
			pages:     []string{pageA},
		},
		{
			name:      "entity already has a canonical page",
			candidate: page(pageC, "/trainers/", ptr(entA)),
			entity:    owner,
			reasons:   []pagemap.Reason{pagemap.ReasonSameEntityCanonical, pagemap.ReasonSamePrimaryKeyword},
			pages:     []string{pageA, pageB},
		},
		{
			name:      "the canonical page itself is not its own rival",
			candidate: page(pageA, "/shoes/", ptr(entA)),
			entity:    entityWith(entA, "unique phrase", ptr(pageA)),
		},
		{
			name:      "another entity owns the keyword",
			candidate: page(pageC, "/trainers/", ptr("e-new")),
			entity:    entityWith("e-new", "RUNNING shoes", nil),
			reasons:   []pagemap.Reason{pagemap.ReasonSamePrimaryKeyword, pagemap.ReasonSamePrimaryKeyword},
			pages:     []string{pageA, pageB},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			verdict := pagemap.Cannibalization(tc.candidate, tc.entity, index, g)
			if verdict.Allowed != (len(tc.reasons) == 0) {
				t.Fatalf("Allowed = %v, evidence %+v", verdict.Allowed, verdict.Evidence)
			}
			if len(verdict.Evidence) != len(tc.reasons) {
				t.Fatalf("evidence = %+v, want %d entries", verdict.Evidence, len(tc.reasons))
			}
			for i, reason := range tc.reasons {
				if verdict.Evidence[i].Reason != reason || verdict.Evidence[i].PageID != tc.pages[i] {
					t.Errorf("evidence[%d] = %+v, want %s on %s", i, verdict.Evidence[i], reason, tc.pages[i])
				}
				if verdict.Evidence[i].Path == "" {
					t.Errorf("evidence[%d] carries no path", i)
				}
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/pagemap/ -run TestCannibalization -v`
Expected: FAIL, `Cannibalization` is undefined.

- [ ] **Step 3: Write the implementation**

`internal/domain/pagemap/cannibalization.go`:

```go
package pagemap

import (
	"strings"

	"github.com/davidmovas/postulator/internal/domain/graph"
)

type Reason string

const (
	ReasonSamePrimaryKeyword  Reason = "same_primary_keyword"
	ReasonSameEntityCanonical Reason = "same_entity_canonical"
	ReasonPathConflict        Reason = "path_conflict"
)

type Evidence struct {
	PageID   string
	Path     string
	Reason   Reason
	EntityID string
}

type Verdict struct {
	Allowed  bool
	Evidence []Evidence
}

func Cannibalization(candidate Page, entity graph.Entity, index Index, g graph.Graph) Verdict {
	var evidence []Evidence

	if path, err := NormalizePath(candidate.Path); err == nil {
		if other, ok := index.byPath[path]; ok && other.ID != candidate.ID {
			evidence = append(evidence, Evidence{PageID: other.ID, Path: other.Path, Reason: ReasonPathConflict})
		}
	}

	if entity.ID == "" {
		return Verdict{Allowed: len(evidence) == 0, Evidence: evidence}
	}

	if entity.CanonicalPageID != nil && *entity.CanonicalPageID != candidate.ID {
		if owner, ok := index.byID[*entity.CanonicalPageID]; ok {
			evidence = append(evidence, Evidence{PageID: owner.ID, Path: owner.Path, Reason: ReasonSameEntityCanonical, EntityID: entity.ID})
		}
	}

	if entity.PrimaryKeyword != "" {
		for _, other := range g.Entities() {
			if other.ID == entity.ID || other.CanonicalPageID == nil || !strings.EqualFold(other.PrimaryKeyword, entity.PrimaryKeyword) {
				continue
			}
			owner, ok := index.byID[*other.CanonicalPageID]
			if !ok || owner.ID == candidate.ID {
				continue
			}
			evidence = append(evidence, Evidence{PageID: owner.ID, Path: owner.Path, Reason: ReasonSamePrimaryKeyword, EntityID: other.ID})
		}
	}

	return Verdict{Allowed: len(evidence) == 0, Evidence: evidence}
}
```

The "entity already has a canonical page" case yields two entries because `owner` (keyword "Running Shoes", canonical `pageA`) is also matched by `rival`'s identical keyword on `pageB`; `g.Entities()` is name-sorted, and the entity names are their ids, so `rival` (`2b2b…`) follows `owner` (`1a1a…`) and the keyword entry for `pageB` comes after the canonical entry for `pageA`. In "another entity owns the keyword" both `owner` and `rival` match, in that order.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race -cover ./internal/domain/pagemap/ -v`
Expected: PASS, coverage ≥ 90%.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/pagemap/cannibalization.go internal/domain/pagemap/cannibalization_test.go
git commit -m "feat(domain): cannibalization verdict with per-page evidence

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---
### Task 8: `domain/template` types and validation

**Files:**
- Create: `internal/domain/template/spec.go`, `internal/domain/template/validate.go`, `internal/domain/template/query.go`
- Test: `internal/domain/template/validate_test.go`

**Interfaces:**
- Consumes: `llm.Role`, `llm.ModelRef` (Task 1); `errors.New`, `(*errors.Error).WithDetail|WithInternal`.
- Produces: `Scope` (`ScopeGlobal|ScopeSite`), `ImageSource` (`ImagesAI = "ai"`, `ImagesWPMedia = "wpmedia"`, `ImagesLocal = "local"`), `AnchorStrategy` (`AnchorPreferUser = "prefer_user"`, `AnchorRotate = "rotate"`), `OverrideScope` (`OverrideSite = "site"`, `OverridePage = "page"`), each with `Valid()`; the JSON-tagged `SectionKeywordRules{Include []string; PrimaryInHeading bool}`, `Section{Heading, Intent string; TargetWords int; Required bool; KeywordRules SectionKeywordRules}`, `Length{Min, Max int}`, `KeywordRules{PrimaryInTitle, PrimaryInH1, PrimaryInFirstParagraph bool; MaxDensity float64}`, `LinkRules{UpDepth int; DownLinks bool; SiblingMinWeight float64; MaxLinks, MaxPerTarget, ParentLinkWithinParagraphs int; ChildrenSection bool}`, `MetaRules{TitlePattern string; DescriptionMax int}`, `Images{Featured bool; Inline int; Source ImageSource}`, `StepSpec{Name string; Enabled bool; Params map[string]any}`, `TemplateSpec{Sections []Section; Tone string; Length Length; KeywordRules KeywordRules; LinkRules LinkRules; MetaRules MetaRules; Images Images; ModelProfiles map[llm.Role]llm.ModelRef; Recipe []StepSpec}`; untagged `Template{ID string; Scope Scope; SiteID *string; Name, PageKind string; Version int; Spec TemplateSpec; CreatedAt, UpdatedAt time.Time}`, `Override{ID, TemplateID string; Scope OverrideScope; TargetID string; Patch json.RawMessage; CreatedAt, UpdatedAt time.Time}`, `LinkPolicy{ID string; Scope Scope; SiteID *string; Name string; Rules LinkRules; ForbidExternal, ForbidSelf bool; AnchorStrategy AnchorStrategy; CreatedAt, UpdatedAt time.Time}`; `Validate(spec TemplateSpec) error`, `ValidateLinkRules(rules LinkRules) error`, `(Template).Validate() error`, `(LinkPolicy).Validate() error`, `(Override).Validate() error`; `Sort` (`SortCreatedAt = "createdAt"`, `SortName = "name"`), `Query{Scope *Scope; SiteID *string; PageKind, Name string; Sort Sort; Desc bool}`, `PolicyQuery{Scope *Scope; SiteID *string; Name string; Sort Sort; Desc bool}`.

- [ ] **Step 1: Write the failing test**

`internal/domain/template/validate_test.go`:

```go
package template_test

import (
	"encoding/json"
	stderrors "errors"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

var stamp = time.Date(2026, time.September, 18, 9, 0, 0, 0, time.UTC)

func ptr(s string) *string {
	return &s
}

func fieldOf(t *testing.T, err error) string {
	t.Helper()
	var kernel *errors.Error
	if !stderrors.As(err, &kernel) {
		t.Fatalf("error %v is not a kernel error", err)
	}
	field, _ := kernel.Details["field"].(string)
	return field
}

func validSpec() template.TemplateSpec {
	return template.TemplateSpec{
		Sections: []template.Section{
			{Heading: "Overview", Intent: "Define the topic", TargetWords: 200, Required: true, KeywordRules: template.SectionKeywordRules{Include: []string{}, PrimaryInHeading: true}},
			{Heading: "Details", Intent: "Explain", TargetWords: 400, Required: true, KeywordRules: template.SectionKeywordRules{Include: []string{}}},
		},
		Tone:          "plain",
		Length:        template.Length{Min: 500, Max: 900},
		KeywordRules:  template.KeywordRules{PrimaryInTitle: true, PrimaryInH1: true, PrimaryInFirstParagraph: true, MaxDensity: 0.02},
		LinkRules:     template.LinkRules{UpDepth: 2, DownLinks: true, SiblingMinWeight: 0.5, MaxLinks: 10, MaxPerTarget: 1, ParentLinkWithinParagraphs: 2, ChildrenSection: true},
		MetaRules:     template.MetaRules{TitlePattern: "{primaryKeyword} | {siteName}", DescriptionMax: 155},
		Images:        template.Images{Featured: true, Inline: 1, Source: template.ImagesAI},
		ModelProfiles: map[llm.Role]llm.ModelRef{llm.RoleWriter: {Provider: "openai", Model: "gpt"}},
		Recipe:        []template.StepSpec{{Name: "resolve_context", Enabled: true}, {Name: "generate_body", Enabled: true}},
	}
}

func TestValidateSpec(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		mutate func(*template.TemplateSpec)
		field  string
	}{
		{name: "valid", mutate: func(*template.TemplateSpec) {}},
		{name: "no sections", mutate: func(s *template.TemplateSpec) { s.Sections = nil }, field: "sections"},
		{name: "blank heading", mutate: func(s *template.TemplateSpec) { s.Sections[1].Heading = " " }, field: "sections[1].heading"},
		{name: "negative target words", mutate: func(s *template.TemplateSpec) { s.Sections[0].TargetWords = -1 }, field: "sections[0].targetWords"},
		{name: "negative min", mutate: func(s *template.TemplateSpec) { s.Length.Min = -1 }, field: "length.min"},
		{name: "max below min", mutate: func(s *template.TemplateSpec) { s.Length.Max = 100 }, field: "length.max"},
		{name: "density above one", mutate: func(s *template.TemplateSpec) { s.KeywordRules.MaxDensity = 1.5 }, field: "keywordRules.maxDensity"},
		{name: "negative up depth", mutate: func(s *template.TemplateSpec) { s.LinkRules.UpDepth = -1 }, field: "linkRules.upDepth"},
		{name: "sibling weight", mutate: func(s *template.TemplateSpec) { s.LinkRules.SiblingMinWeight = 2 }, field: "linkRules.siblingMinWeight"},
		{name: "max links", mutate: func(s *template.TemplateSpec) { s.LinkRules.MaxLinks = -1 }, field: "linkRules.maxLinks"},
		{name: "max per target", mutate: func(s *template.TemplateSpec) { s.LinkRules.MaxPerTarget = -1 }, field: "linkRules.maxPerTarget"},
		{name: "paragraph window", mutate: func(s *template.TemplateSpec) { s.LinkRules.ParentLinkWithinParagraphs = -1 }, field: "linkRules.parentLinkWithinParagraphs"},
		{name: "description max", mutate: func(s *template.TemplateSpec) { s.MetaRules.DescriptionMax = -1 }, field: "metaRules.descriptionMax"},
		{name: "inline images", mutate: func(s *template.TemplateSpec) { s.Images.Inline = -1 }, field: "images.inline"},
		{name: "unknown image source", mutate: func(s *template.TemplateSpec) { s.Images.Source = "camera" }, field: "images.source"},
		{name: "images without a source", mutate: func(s *template.TemplateSpec) { s.Images.Source = "" }, field: "images.source"},
		{name: "no images no source is fine", mutate: func(s *template.TemplateSpec) { s.Images = template.Images{} }},
		{name: "unknown role", mutate: func(s *template.TemplateSpec) { s.ModelProfiles = map[llm.Role]llm.ModelRef{"painter": {Provider: "a", Model: "b"}} }, field: "modelProfiles.painter"},
		{name: "incomplete ref", mutate: func(s *template.TemplateSpec) { s.ModelProfiles = map[llm.Role]llm.ModelRef{llm.RoleJudge: {Model: "b"}} }, field: "modelProfiles.judge"},
		{name: "blank step", mutate: func(s *template.TemplateSpec) { s.Recipe[1].Name = "" }, field: "recipe[1].name"},
		{name: "repeated step", mutate: func(s *template.TemplateSpec) { s.Recipe[1].Name = "resolve_context" }, field: "recipe[1].name"},
		{name: "empty recipe is allowed", mutate: func(s *template.TemplateSpec) { s.Recipe = nil }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			spec := validSpec()
			tc.mutate(&spec)
			err := template.Validate(spec)
			if tc.field == "" {
				if err != nil {
					t.Fatalf("Validate: %v", err)
				}
				return
			}
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("code = %q, want INVALID", errors.CodeOf(err))
			}
			if got := fieldOf(t, err); got != tc.field {
				t.Errorf("field = %q, want %q", got, tc.field)
			}
		})
	}
}

func TestTemplateValidate(t *testing.T) {
	t.Parallel()

	base := template.Template{ID: "t1", Scope: template.ScopeGlobal, Name: "Hub", PageKind: "hub", Version: 1, Spec: validSpec(), CreatedAt: stamp, UpdatedAt: stamp}

	cases := []struct {
		name   string
		mutate func(*template.Template)
		field  string
	}{
		{name: "valid global", mutate: func(*template.Template) {}},
		{name: "valid site", mutate: func(x *template.Template) { x.Scope = template.ScopeSite; x.SiteID = ptr("s1") }},
		{name: "no id", mutate: func(x *template.Template) { x.ID = "" }, field: "id"},
		{name: "unknown scope", mutate: func(x *template.Template) { x.Scope = "galaxy" }, field: "scope"},
		{name: "global with site", mutate: func(x *template.Template) { x.SiteID = ptr("s1") }, field: "siteId"},
		{name: "site without site", mutate: func(x *template.Template) { x.Scope = template.ScopeSite }, field: "siteId"},
		{name: "blank name", mutate: func(x *template.Template) { x.Name = " " }, field: "name"},
		{name: "blank kind", mutate: func(x *template.Template) { x.PageKind = "" }, field: "pageKind"},
		{name: "zero version", mutate: func(x *template.Template) { x.Version = 0 }, field: "version"},
		{name: "bad spec", mutate: func(x *template.Template) { x.Spec.Sections = nil }, field: "sections"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			x := base
			tc.mutate(&x)
			err := x.Validate()
			if tc.field == "" {
				if err != nil {
					t.Fatalf("Validate: %v", err)
				}
				return
			}
			if got := fieldOf(t, err); got != tc.field {
				t.Errorf("field = %q, want %q (err %v)", got, tc.field, err)
			}
		})
	}
}

func TestLinkPolicyValidate(t *testing.T) {
	t.Parallel()

	base := template.LinkPolicy{ID: "p1", Scope: template.ScopeSite, SiteID: ptr("s1"), Name: "Strict", Rules: validSpec().LinkRules, ForbidExternal: true, ForbidSelf: true, AnchorStrategy: template.AnchorPreferUser, CreatedAt: stamp, UpdatedAt: stamp}

	cases := []struct {
		name   string
		mutate func(*template.LinkPolicy)
		field  string
	}{
		{name: "valid", mutate: func(*template.LinkPolicy) {}},
		{name: "no id", mutate: func(p *template.LinkPolicy) { p.ID = "" }, field: "id"},
		{name: "site without site", mutate: func(p *template.LinkPolicy) { p.SiteID = nil }, field: "siteId"},
		{name: "global with site", mutate: func(p *template.LinkPolicy) { p.Scope = template.ScopeGlobal }, field: "siteId"},
		{name: "blank name", mutate: func(p *template.LinkPolicy) { p.Name = "" }, field: "name"},
		{name: "bad rules", mutate: func(p *template.LinkPolicy) { p.Rules.MaxLinks = -3 }, field: "linkRules.maxLinks"},
		{name: "unknown strategy", mutate: func(p *template.LinkPolicy) { p.AnchorStrategy = "random" }, field: "anchorStrategy"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := base
			tc.mutate(&p)
			err := p.Validate()
			if tc.field == "" {
				if err != nil {
					t.Fatalf("Validate: %v", err)
				}
				return
			}
			if got := fieldOf(t, err); got != tc.field {
				t.Errorf("field = %q, want %q (err %v)", got, tc.field, err)
			}
		})
	}
}

func TestOverrideValidate(t *testing.T) {
	t.Parallel()

	base := template.Override{ID: "o1", TemplateID: "t1", Scope: template.OverrideSite, TargetID: "s1", Patch: json.RawMessage(`{"tone":"warm"}`), CreatedAt: stamp, UpdatedAt: stamp}

	cases := []struct {
		name   string
		mutate func(*template.Override)
		field  string
	}{
		{name: "valid", mutate: func(*template.Override) {}},
		{name: "no id", mutate: func(o *template.Override) { o.ID = "" }, field: "id"},
		{name: "no template", mutate: func(o *template.Override) { o.TemplateID = "" }, field: "templateId"},
		{name: "unknown scope", mutate: func(o *template.Override) { o.Scope = "galaxy" }, field: "scope"},
		{name: "no target", mutate: func(o *template.Override) { o.TargetID = "" }, field: "targetId"},
		{name: "patch is not json", mutate: func(o *template.Override) { o.Patch = json.RawMessage(`{`) }, field: "patch"},
		{name: "patch is not an object", mutate: func(o *template.Override) { o.Patch = json.RawMessage(`[1]`) }, field: "patch"},
		{name: "empty patch", mutate: func(o *template.Override) { o.Patch = nil }, field: "patch"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			o := base
			tc.mutate(&o)
			err := o.Validate()
			if tc.field == "" {
				if err != nil {
					t.Fatalf("Validate: %v", err)
				}
				return
			}
			if got := fieldOf(t, err); got != tc.field {
				t.Errorf("field = %q, want %q (err %v)", got, tc.field, err)
			}
		})
	}
}

func TestEnums(t *testing.T) {
	t.Parallel()

	if !template.ScopeGlobal.Valid() || !template.ScopeSite.Valid() || template.Scope("x").Valid() {
		t.Error("scope validity is wrong")
	}
	if !template.ImagesAI.Valid() || !template.ImagesWPMedia.Valid() || !template.ImagesLocal.Valid() || template.ImageSource("x").Valid() {
		t.Error("image source validity is wrong")
	}
	if !template.AnchorPreferUser.Valid() || !template.AnchorRotate.Valid() || template.AnchorStrategy("x").Valid() {
		t.Error("anchor strategy validity is wrong")
	}
	if !template.OverrideSite.Valid() || !template.OverridePage.Valid() || template.OverrideScope("x").Valid() {
		t.Error("override scope validity is wrong")
	}
	if !template.SortCreatedAt.Valid() || !template.SortName.Valid() || template.Sort("x").Valid() {
		t.Error("sort validity is wrong")
	}
}

func TestSpecJSONIsCamelCase(t *testing.T) {
	t.Parallel()

	encoded, err := json.Marshal(validSpec())
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	for _, key := range []string{`"sections"`, `"targetWords"`, `"keywordRules"`, `"primaryInHeading"`, `"linkRules"`, `"siblingMinWeight"`, `"metaRules"`, `"descriptionMax"`, `"modelProfiles"`, `"recipe"`} {
		if !strings.Contains(string(encoded), key) {
			t.Errorf("encoded spec lacks %s: %s", key, encoded)
		}
	}
}
```

The import block of this file is `encoding/json`, `stderrors "errors"`, `strings`, `testing`, `time`, the two domain packages and `kernel/errors`.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/template/ -v`
Expected: FAIL, the package does not exist.

- [ ] **Step 3: Write the implementation**

`internal/domain/template/spec.go`:

```go
package template

import (
	"encoding/json"
	"time"

	"github.com/davidmovas/postulator/internal/domain/llm"
)

type Scope string

const (
	ScopeGlobal Scope = "global"
	ScopeSite   Scope = "site"
)

func (s Scope) Valid() bool {
	switch s {
	case ScopeGlobal, ScopeSite:
		return true
	default:
		return false
	}
}

type ImageSource string

const (
	ImagesAI      ImageSource = "ai"
	ImagesWPMedia ImageSource = "wpmedia"
	ImagesLocal   ImageSource = "local"
)

func (s ImageSource) Valid() bool {
	switch s {
	case ImagesAI, ImagesWPMedia, ImagesLocal:
		return true
	default:
		return false
	}
}

type AnchorStrategy string

const (
	AnchorPreferUser AnchorStrategy = "prefer_user"
	AnchorRotate     AnchorStrategy = "rotate"
)

func (s AnchorStrategy) Valid() bool {
	switch s {
	case AnchorPreferUser, AnchorRotate:
		return true
	default:
		return false
	}
}

type OverrideScope string

const (
	OverrideSite OverrideScope = "site"
	OverridePage OverrideScope = "page"
)

func (s OverrideScope) Valid() bool {
	switch s {
	case OverrideSite, OverridePage:
		return true
	default:
		return false
	}
}

type SectionKeywordRules struct {
	Include          []string `json:"include"`
	PrimaryInHeading bool     `json:"primaryInHeading"`
}

type Section struct {
	Heading      string              `json:"heading"`
	Intent       string              `json:"intent"`
	TargetWords  int                 `json:"targetWords"`
	Required     bool                `json:"required"`
	KeywordRules SectionKeywordRules `json:"keywordRules"`
}

type Length struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

type KeywordRules struct {
	PrimaryInTitle          bool    `json:"primaryInTitle"`
	PrimaryInH1             bool    `json:"primaryInH1"`
	PrimaryInFirstParagraph bool    `json:"primaryInFirstParagraph"`
	MaxDensity              float64 `json:"maxDensity"`
}

type LinkRules struct {
	UpDepth                    int     `json:"upDepth"`
	DownLinks                  bool    `json:"downLinks"`
	SiblingMinWeight           float64 `json:"siblingMinWeight"`
	MaxLinks                   int     `json:"maxLinks"`
	MaxPerTarget               int     `json:"maxPerTarget"`
	ParentLinkWithinParagraphs int     `json:"parentLinkWithinParagraphs"`
	ChildrenSection            bool    `json:"childrenSection"`
}

type MetaRules struct {
	TitlePattern   string `json:"titlePattern"`
	DescriptionMax int    `json:"descriptionMax"`
}

type Images struct {
	Featured bool        `json:"featured"`
	Inline   int         `json:"inline"`
	Source   ImageSource `json:"source"`
}

type StepSpec struct {
	Name    string         `json:"name"`
	Enabled bool           `json:"enabled"`
	Params  map[string]any `json:"params,omitempty"`
}

type TemplateSpec struct {
	Sections      []Section                 `json:"sections"`
	Tone          string                    `json:"tone"`
	Length        Length                    `json:"length"`
	KeywordRules  KeywordRules              `json:"keywordRules"`
	LinkRules     LinkRules                 `json:"linkRules"`
	MetaRules     MetaRules                 `json:"metaRules"`
	Images        Images                    `json:"images"`
	ModelProfiles map[llm.Role]llm.ModelRef `json:"modelProfiles"`
	Recipe        []StepSpec                `json:"recipe"`
}

type Template struct {
	ID        string
	Scope     Scope
	SiteID    *string
	Name      string
	PageKind  string
	Version   int
	Spec      TemplateSpec
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Override struct {
	ID         string
	TemplateID string
	Scope      OverrideScope
	TargetID   string
	Patch      json.RawMessage
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type LinkPolicy struct {
	ID             string
	Scope          Scope
	SiteID         *string
	Name           string
	Rules          LinkRules
	ForbidExternal bool
	ForbidSelf     bool
	AnchorStrategy AnchorStrategy
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
```

`internal/domain/template/validate.go`:

```go
package template

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func invalid(message, field string) *errors.Error {
	return errors.New(errors.Invalid, message).WithDetail("field", field)
}

func Validate(spec TemplateSpec) error {
	if len(spec.Sections) == 0 {
		return invalid("template needs at least one section", "sections")
	}
	for i, section := range spec.Sections {
		field := "sections[" + strconv.Itoa(i) + "]"
		if strings.TrimSpace(section.Heading) == "" {
			return invalid("section heading must not be empty", field+".heading")
		}
		if section.TargetWords < 0 {
			return invalid("section target words must not be negative", field+".targetWords")
		}
	}
	if spec.Length.Min < 0 {
		return invalid("length minimum must not be negative", "length.min")
	}
	if spec.Length.Max < spec.Length.Min {
		return invalid("length maximum must not be below the minimum", "length.max")
	}
	if spec.KeywordRules.MaxDensity < 0 || spec.KeywordRules.MaxDensity > 1 {
		return invalid("keyword density must be between 0 and 1", "keywordRules.maxDensity")
	}
	if err := ValidateLinkRules(spec.LinkRules); err != nil {
		return err
	}
	if spec.MetaRules.DescriptionMax < 0 {
		return invalid("description maximum must not be negative", "metaRules.descriptionMax")
	}
	if spec.Images.Inline < 0 {
		return invalid("inline image count must not be negative", "images.inline")
	}
	if spec.Images.Source != "" && !spec.Images.Source.Valid() {
		return invalid("image source is not recognised", "images.source")
	}
	if (spec.Images.Featured || spec.Images.Inline > 0) && spec.Images.Source == "" {
		return invalid("image source is required when images are requested", "images.source")
	}
	for role, ref := range spec.ModelProfiles {
		field := "modelProfiles." + string(role)
		if !role.Valid() {
			return invalid("model profile role is not recognised", field)
		}
		if !ref.Valid() {
			return invalid("model profile needs a provider and a model", field)
		}
	}
	seen := make(map[string]struct{}, len(spec.Recipe))
	for i, step := range spec.Recipe {
		name := strings.TrimSpace(step.Name)
		field := "recipe[" + strconv.Itoa(i) + "].name"
		if name == "" {
			return invalid("recipe step name must not be empty", field)
		}
		if _, dup := seen[name]; dup {
			return invalid("recipe step name is repeated", field)
		}
		seen[name] = struct{}{}
	}
	return nil
}

func ValidateLinkRules(rules LinkRules) error {
	switch {
	case rules.UpDepth < 0:
		return invalid("up depth must not be negative", "linkRules.upDepth")
	case rules.SiblingMinWeight < 0 || rules.SiblingMinWeight > 1:
		return invalid("sibling minimum weight must be between 0 and 1", "linkRules.siblingMinWeight")
	case rules.MaxLinks < 0:
		return invalid("maximum links must not be negative", "linkRules.maxLinks")
	case rules.MaxPerTarget < 0:
		return invalid("maximum links per target must not be negative", "linkRules.maxPerTarget")
	case rules.ParentLinkWithinParagraphs < 0:
		return invalid("parent link paragraph window must not be negative", "linkRules.parentLinkWithinParagraphs")
	default:
		return nil
	}
}

func validateScope(scope Scope, siteID *string) error {
	if !scope.Valid() {
		return invalid("scope is not recognised", "scope")
	}
	if scope == ScopeGlobal && siteID != nil {
		return invalid("a global record carries no site id", "siteId")
	}
	if scope == ScopeSite && (siteID == nil || *siteID == "") {
		return invalid("a site record needs a site id", "siteId")
	}
	return nil
}

func (t Template) Validate() error {
	if t.ID == "" {
		return invalid("template id must not be empty", "id")
	}
	if err := validateScope(t.Scope, t.SiteID); err != nil {
		return err
	}
	if strings.TrimSpace(t.Name) == "" {
		return invalid("template name must not be empty", "name")
	}
	if strings.TrimSpace(t.PageKind) == "" {
		return invalid("template page kind must not be empty", "pageKind")
	}
	if t.Version < 1 {
		return invalid("template version starts at 1", "version")
	}
	return Validate(t.Spec)
}

func (p LinkPolicy) Validate() error {
	if p.ID == "" {
		return invalid("link policy id must not be empty", "id")
	}
	if err := validateScope(p.Scope, p.SiteID); err != nil {
		return err
	}
	if strings.TrimSpace(p.Name) == "" {
		return invalid("link policy name must not be empty", "name")
	}
	if err := ValidateLinkRules(p.Rules); err != nil {
		return err
	}
	if !p.AnchorStrategy.Valid() {
		return invalid("anchor strategy is not recognised", "anchorStrategy")
	}
	return nil
}

func (o Override) Validate() error {
	switch {
	case o.ID == "":
		return invalid("override id must not be empty", "id")
	case o.TemplateID == "":
		return invalid("override template id must not be empty", "templateId")
	case !o.Scope.Valid():
		return invalid("override scope is not recognised", "scope")
	case o.TargetID == "":
		return invalid("override target id must not be empty", "targetId")
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(o.Patch, &object); err != nil || object == nil {
		return invalid("override patch must be a json object", "patch")
	}
	return nil
}
```

`json.Unmarshal` of `null` into a map leaves it nil, and of `[1]` fails, so both non-object cases fall into the same branch; an empty `Patch` is a syntax error and lands there too.

`internal/domain/template/query.go`:

```go
package template

type Sort string

const (
	SortCreatedAt Sort = "createdAt"
	SortName      Sort = "name"
)

func (s Sort) Valid() bool {
	switch s {
	case SortCreatedAt, SortName:
		return true
	default:
		return false
	}
}

type Query struct {
	Scope    *Scope
	SiteID   *string
	PageKind string
	Name     string
	Sort     Sort
	Desc     bool
}

type PolicyQuery struct {
	Scope  *Scope
	SiteID *string
	Name   string
	Sort   Sort
	Desc   bool
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race ./internal/domain/template/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/template/spec.go internal/domain/template/validate.go internal/domain/template/query.go internal/domain/template/validate_test.go
git commit -m "feat(domain): template spec, link policy and override types with validation

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 9: `domain/template` merge patch and resolution

**Files:**
- Create: `internal/domain/template/merge.go`
- Test: `internal/domain/template/merge_test.go`

**Interfaces:**
- Consumes: `TemplateSpec`, `Validate` (Task 8).
- Produces: `MergePatch(target, patch json.RawMessage) (json.RawMessage, error)` (RFC 7396; an empty patch returns the target unchanged), `Resolve(base TemplateSpec, siteOverride, pageOverride json.RawMessage) (TemplateSpec, error)`.

- [ ] **Step 1: Write the failing test**

```go
package template_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func canonical(t *testing.T, raw json.RawMessage) any {
	t.Helper()
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("unmarshal %s: %v", raw, err)
	}
	return value
}

func TestMergePatchRFC7396Vectors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		target string
		patch  string
		want   string
	}{
		{`{"a":"b"}`, `{"a":"c"}`, `{"a":"c"}`},
		{`{"a":"b"}`, `{"b":"c"}`, `{"a":"b","b":"c"}`},
		{`{"a":"b"}`, `{"a":null}`, `{}`},
		{`{"a":"b","b":"c"}`, `{"a":null}`, `{"b":"c"}`},
		{`{"a":["b"]}`, `{"a":"c"}`, `{"a":"c"}`},
		{`{"a":"c"}`, `{"a":["b"]}`, `{"a":["b"]}`},
		{`{"a":{"b":"c"}}`, `{"a":{"b":"d","c":null}}`, `{"a":{"b":"d"}}`},
		{`{"a":[{"b":"c"}]}`, `{"a":[1]}`, `{"a":[1]}`},
		{`["a","b"]`, `["c","d"]`, `["c","d"]`},
		{`{"a":"b"}`, `["c"]`, `["c"]`},
		{`{"a":"foo"}`, `null`, `null`},
		{`{"a":"foo"}`, `"bar"`, `"bar"`},
		{`{"e":null}`, `{"a":1}`, `{"e":null,"a":1}`},
		{`[1,2]`, `{"a":"b","c":null}`, `{"a":"b"}`},
		{`{}`, `{"a":{"bb":{"ccc":null}}}`, `{"a":{"bb":{}}}`},
	}

	for _, tc := range cases {
		t.Run(tc.target+" + "+tc.patch, func(t *testing.T) {
			t.Parallel()

			got, err := template.MergePatch(json.RawMessage(tc.target), json.RawMessage(tc.patch))
			if err != nil {
				t.Fatalf("MergePatch: %v", err)
			}
			if !reflect.DeepEqual(canonical(t, got), canonical(t, json.RawMessage(tc.want))) {
				t.Errorf("MergePatch = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestMergePatchEdges(t *testing.T) {
	t.Parallel()

	same, err := template.MergePatch(json.RawMessage(`{"a":1}`), nil)
	if err != nil || string(same) != `{"a":1}` {
		t.Fatalf("empty patch must return the target: %s, %v", same, err)
	}
	if _, err = template.MergePatch(json.RawMessage(`{"a":1}`), json.RawMessage(`{`)); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("broken patch code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = template.MergePatch(json.RawMessage(`{`), json.RawMessage(`{"a":1}`)); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("broken target code = %q, want INVALID", errors.CodeOf(err))
	}
}

func TestResolveLayersOverrides(t *testing.T) {
	t.Parallel()

	base := validSpec()
	site := json.RawMessage(`{"linkRules":{"maxLinks":5},"tone":"warm"}`)
	page := json.RawMessage(`{"sections":[{"heading":"Only","intent":"one","targetWords":100,"required":true,"keywordRules":{"include":["x"],"primaryInHeading":false}}],"images":{"inline":0},"modelProfiles":{"writer":null,"editor":{"provider":"anthropic","model":"m"}}}`)

	resolved, err := template.Resolve(base, site, page)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Tone != "warm" {
		t.Errorf("tone = %q, want the site override", resolved.Tone)
	}
	if resolved.LinkRules.MaxLinks != 5 || resolved.LinkRules.UpDepth != 2 {
		t.Errorf("link rules = %+v, want maxLinks patched and upDepth kept", resolved.LinkRules)
	}
	if len(resolved.Sections) != 1 || resolved.Sections[0].Heading != "Only" || len(resolved.Sections[0].KeywordRules.Include) != 1 {
		t.Errorf("sections = %+v, want the page override to replace the list wholesale", resolved.Sections)
	}
	if !resolved.Images.Featured || resolved.Images.Inline != 0 || resolved.Images.Source != template.ImagesAI {
		t.Errorf("images = %+v, want inline patched and the rest kept", resolved.Images)
	}
	if _, writer := resolved.ModelProfiles["writer"]; writer || resolved.ModelProfiles["editor"].Provider != "anthropic" {
		t.Errorf("model profiles = %+v, want writer removed and editor added", resolved.ModelProfiles)
	}
	if len(resolved.Recipe) != 2 {
		t.Errorf("recipe = %+v, want untouched", resolved.Recipe)
	}

	untouched, err := template.Resolve(base, nil, nil)
	if err != nil || !reflect.DeepEqual(untouched, base) {
		t.Errorf("Resolve without overrides = %+v, %v; want the base", untouched, err)
	}
}

func TestResolveRejectsBrokenResults(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		site  string
		page  string
		field string
	}{
		{name: "invalid after merge", site: `{"length":{"min":9000}}`, field: "length.max"},
		{name: "sections removed", page: `{"sections":null}`, field: "sections"},
		{name: "unknown key", site: `{"tonee":"x"}`},
		{name: "wrong type", page: `{"length":{"min":"ten"}}`},
		{name: "broken json", site: `{`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := template.Resolve(validSpec(), json.RawMessage(tc.site), json.RawMessage(tc.page))
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("code = %q, want INVALID (err %v)", errors.CodeOf(err), err)
			}
			if tc.field != "" {
				if got := fieldOf(t, err); got != tc.field {
					t.Errorf("field = %q, want %q", got, tc.field)
				}
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/template/ -run 'TestMerge|TestResolve' -v`
Expected: FAIL, `MergePatch` and `Resolve` are undefined.

- [ ] **Step 3: Write the implementation**

`internal/domain/template/merge.go`:

```go
package template

import (
	"bytes"
	"encoding/json"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func MergePatch(target, patch json.RawMessage) (json.RawMessage, error) {
	if len(bytes.TrimSpace(patch)) == 0 {
		return target, nil
	}

	var patchValue any
	if err := json.Unmarshal(patch, &patchValue); err != nil {
		return nil, invalid("merge patch is not valid json", "patch").WithInternal(err)
	}
	patchObject, ok := patchValue.(map[string]any)
	if !ok {
		return marshal(patchValue)
	}

	var targetValue any
	if len(bytes.TrimSpace(target)) > 0 {
		if err := json.Unmarshal(target, &targetValue); err != nil {
			return nil, invalid("merge target is not valid json", "target").WithInternal(err)
		}
	}
	targetObject, ok := targetValue.(map[string]any)
	if !ok {
		targetObject = map[string]any{}
	}
	return marshal(mergeObjects(targetObject, patchObject))
}

func mergeObjects(target, patch map[string]any) map[string]any {
	for key, value := range patch {
		if value == nil {
			delete(target, key)
			continue
		}
		if child, ok := value.(map[string]any); ok {
			existing, _ := target[key].(map[string]any)
			if existing == nil {
				existing = map[string]any{}
			}
			target[key] = mergeObjects(existing, child)
			continue
		}
		target[key] = value
	}
	return target
}

func marshal(value any) (json.RawMessage, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "encode the merged document")
	}
	return encoded, nil
}

func Resolve(base TemplateSpec, siteOverride, pageOverride json.RawMessage) (TemplateSpec, error) {
	document, err := marshal(base)
	if err != nil {
		return TemplateSpec{}, err
	}
	if document, err = MergePatch(document, siteOverride); err != nil {
		return TemplateSpec{}, err
	}
	if document, err = MergePatch(document, pageOverride); err != nil {
		return TemplateSpec{}, err
	}

	var resolved TemplateSpec
	decoder := json.NewDecoder(bytes.NewReader(document))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&resolved); err != nil {
		return TemplateSpec{}, invalid("resolved template does not match the template spec", "spec").WithInternal(err)
	}
	if err = Validate(resolved); err != nil {
		return TemplateSpec{}, err
	}
	return resolved, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race ./internal/domain/template/ -v`
Expected: PASS. The "sections removed" case reaches `Validate` with a nil slice and reports `sections`; "invalid after merge" reports `length.max` because the merged minimum 9000 exceeds the base maximum 900.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/template/merge.go internal/domain/template/merge_test.go
git commit -m "feat(domain): rfc 7396 merge patch and layered template resolution

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---
### Task 10: `domain/template` starter seeds

**Files:**
- Create: `internal/domain/template/seed.go`, `internal/domain/template/seed/hub.json`, `internal/domain/template/seed/product.json`, `internal/domain/template/seed/guide.json`, `internal/domain/template/seed/comparison.json`, `internal/domain/template/seed/category.json`
- Test: `internal/domain/template/seed_test.go`

**Interfaces:**
- Consumes: `Template`, `TemplateSpec`, `(Template).Validate` (Task 8).
- Produces: `Seed() []Template` — five global templates (`Scope: ScopeGlobal`, `Version: 1`, empty `ID` and timestamps, filled in by the use case on insert), in file-name order: Category, Comparison, Guide, Hub, Product.

- [ ] **Step 1: Write the failing test**

```go
package template_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/template"
)

func TestSeedShipsFiveValidGlobalTemplates(t *testing.T) {
	t.Parallel()

	seeds := template.Seed()
	names := make([]string, 0, len(seeds))
	for _, seed := range seeds {
		names = append(names, seed.Name)
		if seed.Scope != template.ScopeGlobal || seed.SiteID != nil || seed.Version != 1 || seed.ID != "" {
			t.Errorf("%s: seed must be global, unversioned beyond 1 and carry no id, got %+v", seed.Name, seed)
		}
		if err := template.Validate(seed.Spec); err != nil {
			t.Errorf("%s: %v", seed.Name, err)
		}
		if seed.PageKind == "" || len(seed.Spec.Recipe) == 0 || seed.Spec.Length.Min == 0 || seed.Spec.MetaRules.DescriptionMax == 0 {
			t.Errorf("%s: seed is missing content: %+v", seed.Name, seed.Spec)
		}
		if len(seed.Spec.ModelProfiles) != 0 {
			t.Errorf("%s: seeds must not pin models", seed.Name)
		}
	}
	if !slices.Equal(names, []string{"Category", "Comparison", "Guide", "Hub", "Product"}) {
		t.Errorf("names = %v", names)
	}
}

func TestSeedRecipesUseTheStepCatalogue(t *testing.T) {
	t.Parallel()

	catalogue := []string{"resolve_context", "generate_body", "generate_meta", "insert_links", "repair_links", "generate_images", "validate", "judge", "publish", "relink_neighbors", "sync_back", "report"}
	for _, seed := range template.Seed() {
		steps := make([]string, 0, len(seed.Spec.Recipe))
		for _, step := range seed.Spec.Recipe {
			steps = append(steps, step.Name)
		}
		if !slices.Equal(steps, catalogue) {
			t.Errorf("%s recipe = %v, want the full catalogue in order", seed.Name, steps)
		}
	}
}

func TestSeedReturnsFreshCopies(t *testing.T) {
	t.Parallel()

	first := template.Seed()
	first[0].Spec.Sections[0].Heading = "mutated"
	if template.Seed()[0].Spec.Sections[0].Heading == "mutated" {
		t.Error("Seed must decode fresh values on every call")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/template/ -run TestSeed -v`
Expected: FAIL, `Seed` is undefined.

- [ ] **Step 3: Write the seeds and the loader**

`internal/domain/template/seed/hub.json`:

```json
{
  "name": "Hub",
  "pageKind": "hub",
  "spec": {
    "sections": [
      {"heading": "Overview", "intent": "Define the topic, say who it is for and state what the reader will be able to decide after reading the hub", "targetWords": 250, "required": true, "keywordRules": {"include": [], "primaryInHeading": true}},
      {"heading": "Key Areas", "intent": "Introduce every child topic in its own short paragraph that links to the child page with a descriptive anchor", "targetWords": 700, "required": true, "keywordRules": {"include": [], "primaryInHeading": false}},
      {"heading": "How to Choose", "intent": "Give the decision criteria a reader should weigh before going deeper into one child topic", "targetWords": 400, "required": true, "keywordRules": {"include": [], "primaryInHeading": false}},
      {"heading": "Common Questions", "intent": "Answer the five questions people ask most about the topic, one short paragraph each, phrased as the reader would search", "targetWords": 350, "required": true, "keywordRules": {"include": [], "primaryInHeading": false}},
      {"heading": "Related Topics", "intent": "Point to approved sibling topics with one sentence each explaining the connection", "targetWords": 150, "required": false, "keywordRules": {"include": [], "primaryInHeading": false}}
    ],
    "tone": "Authoritative and plain; second person; short paragraphs; no superlatives, no marketing claims, no filler transitions",
    "length": {"min": 1600, "max": 2400},
    "keywordRules": {"primaryInTitle": true, "primaryInH1": true, "primaryInFirstParagraph": true, "maxDensity": 0.025},
    "linkRules": {"upDepth": 1, "downLinks": true, "siblingMinWeight": 0.5, "maxLinks": 20, "maxPerTarget": 1, "parentLinkWithinParagraphs": 2, "childrenSection": true},
    "metaRules": {"titlePattern": "{primaryKeyword}: The Complete Guide | {siteName}", "descriptionMax": 155},
    "images": {"featured": true, "inline": 0, "source": "ai"},
    "modelProfiles": {},
    "recipe": [
      {"name": "resolve_context", "enabled": true},
      {"name": "generate_body", "enabled": true},
      {"name": "generate_meta", "enabled": true},
      {"name": "insert_links", "enabled": true},
      {"name": "repair_links", "enabled": true},
      {"name": "generate_images", "enabled": true},
      {"name": "validate", "enabled": true},
      {"name": "judge", "enabled": true},
      {"name": "publish", "enabled": true},
      {"name": "relink_neighbors", "enabled": true},
      {"name": "sync_back", "enabled": true},
      {"name": "report", "enabled": true}
    ]
  }
}
```

`internal/domain/template/seed/product.json`:

```json
{
  "name": "Product",
  "pageKind": "product",
  "spec": {
    "sections": [
      {"heading": "Overview", "intent": "State what the product is, the problem it solves and the one reason to choose it, in the customer's words", "targetWords": 150, "required": true, "keywordRules": {"include": [], "primaryInHeading": true}},
      {"heading": "Key Features", "intent": "List the features that matter to the buyer as benefit-led bullets, each tied to a concrete use", "targetWords": 300, "required": true, "keywordRules": {"include": [], "primaryInHeading": false}},
      {"heading": "Specifications", "intent": "Reproduce the supplied specifications exactly in a table; never invent a number", "targetWords": 200, "required": true, "keywordRules": {"include": [], "primaryInHeading": false}},
      {"heading": "Who It Is For", "intent": "Describe the buyer this product fits and the one it does not, and link to the parent category and the closest alternative", "targetWords": 200, "required": true, "keywordRules": {"include": [], "primaryInHeading": false}},
      {"heading": "Frequently Asked Questions", "intent": "Answer four purchase-blocking questions about compatibility, sizing, delivery and returns", "targetWords": 200, "required": true, "keywordRules": {"include": [], "primaryInHeading": false}}
    ],
    "tone": "Concrete and benefit-led; present tense; specific numbers over adjectives; no hype and no invented claims",
    "length": {"min": 800, "max": 1400},
    "keywordRules": {"primaryInTitle": true, "primaryInH1": true, "primaryInFirstParagraph": true, "maxDensity": 0.03},
    "linkRules": {"upDepth": 2, "downLinks": false, "siblingMinWeight": 0.6, "maxLinks": 8, "maxPerTarget": 1, "parentLinkWithinParagraphs": 1, "childrenSection": false},
    "metaRules": {"titlePattern": "{primaryKeyword} | {siteName}", "descriptionMax": 155},
    "images": {"featured": true, "inline": 0, "source": "wpmedia"},
    "modelProfiles": {},
    "recipe": [
      {"name": "resolve_context", "enabled": true},
      {"name": "generate_body", "enabled": true},
      {"name": "generate_meta", "enabled": true},
      {"name": "insert_links", "enabled": true},
      {"name": "repair_links", "enabled": true},
      {"name": "generate_images", "enabled": true},
      {"name": "validate", "enabled": true},
      {"name": "judge", "enabled": true},
      {"name": "publish", "enabled": true},
      {"name": "relink_neighbors", "enabled": true},
      {"name": "sync_back", "enabled": true},
      {"name": "report", "enabled": true}
    ]
  }
}
```

`internal/domain/template/seed/guide.json`:

```json
{
  "name": "Guide",
  "pageKind": "guide",
  "spec": {
    "sections": [
      {"heading": "Introduction", "intent": "Say what the reader will achieve, how long it takes and what they need before starting", "targetWords": 200, "required": true, "keywordRules": {"include": [], "primaryInHeading": true}},
      {"heading": "Step-by-Step Instructions", "intent": "Numbered steps in execution order, one action per step, with the expected result after each", "targetWords": 900, "required": true, "keywordRules": {"include": [], "primaryInHeading": false}},
      {"heading": "Tips and Common Mistakes", "intent": "The three mistakes beginners make and how to avoid each, followed by two tips an expert would give", "targetWords": 300, "required": true, "keywordRules": {"include": [], "primaryInHeading": false}},
      {"heading": "Tools and Materials", "intent": "What is needed, linking to product or category pages where the site sells them", "targetWords": 150, "required": false, "keywordRules": {"include": [], "primaryInHeading": false}},
      {"heading": "Frequently Asked Questions", "intent": "Answer the follow-up questions a reader has after finishing the steps", "targetWords": 250, "required": true, "keywordRules": {"include": [], "primaryInHeading": false}}
    ],
    "tone": "Instructional and direct; imperative mood for steps; explain why before how; no jargon without a one-line definition",
    "length": {"min": 1500, "max": 2200},
    "keywordRules": {"primaryInTitle": true, "primaryInH1": true, "primaryInFirstParagraph": true, "maxDensity": 0.02},
    "linkRules": {"upDepth": 2, "downLinks": true, "siblingMinWeight": 0.5, "maxLinks": 12, "maxPerTarget": 1, "parentLinkWithinParagraphs": 2, "childrenSection": false},
    "metaRules": {"titlePattern": "How to {primaryKeyword}: Step-by-Step Guide | {siteName}", "descriptionMax": 155},
    "images": {"featured": true, "inline": 2, "source": "ai"},
    "modelProfiles": {},
    "recipe": [
      {"name": "resolve_context", "enabled": true},
      {"name": "generate_body", "enabled": true},
      {"name": "generate_meta", "enabled": true},
      {"name": "insert_links", "enabled": true},
      {"name": "repair_links", "enabled": true},
      {"name": "generate_images", "enabled": true},
      {"name": "validate", "enabled": true},
      {"name": "judge", "enabled": true},
      {"name": "publish", "enabled": true},
      {"name": "relink_neighbors", "enabled": true},
      {"name": "sync_back", "enabled": true},
      {"name": "report", "enabled": true}
    ]
  }
}
```

`internal/domain/template/seed/comparison.json`:

```json
{
  "name": "Comparison",
  "pageKind": "comparison",
  "spec": {
    "sections": [
      {"heading": "Overview", "intent": "Name the options being compared, who each is for and the verdict in one sentence", "targetWords": 150, "required": true, "keywordRules": {"include": [], "primaryInHeading": true}},
      {"heading": "Comparison at a Glance", "intent": "A table with one row per criterion and one column per option; criteria are the ones buyers use to decide", "targetWords": 250, "required": true, "keywordRules": {"include": [], "primaryInHeading": false}},
      {"heading": "Detailed Differences", "intent": "One subsection per criterion explaining where the options differ and why it matters, citing the supplied data", "targetWords": 600, "required": true, "keywordRules": {"include": [], "primaryInHeading": false}},
      {"heading": "Which One Should You Choose", "intent": "A recommendation per reader situation, each linking to the option's own page", "targetWords": 250, "required": true, "keywordRules": {"include": [], "primaryInHeading": false}},
      {"heading": "Frequently Asked Questions", "intent": "Answer the questions readers ask when they are still undecided between the options", "targetWords": 200, "required": true, "keywordRules": {"include": [], "primaryInHeading": false}}
    ],
    "tone": "Even-handed and specific; state trade-offs rather than winners; every claim tied to a criterion in the table",
    "length": {"min": 1200, "max": 1800},
    "keywordRules": {"primaryInTitle": true, "primaryInH1": true, "primaryInFirstParagraph": true, "maxDensity": 0.02},
    "linkRules": {"upDepth": 2, "downLinks": false, "siblingMinWeight": 0.7, "maxLinks": 10, "maxPerTarget": 1, "parentLinkWithinParagraphs": 2, "childrenSection": false},
    "metaRules": {"titlePattern": "{primaryKeyword}: Which Is Better? | {siteName}", "descriptionMax": 155},
    "images": {"featured": true, "inline": 1, "source": "ai"},
    "modelProfiles": {},
    "recipe": [
      {"name": "resolve_context", "enabled": true},
      {"name": "generate_body", "enabled": true},
      {"name": "generate_meta", "enabled": true},
      {"name": "insert_links", "enabled": true},
      {"name": "repair_links", "enabled": true},
      {"name": "generate_images", "enabled": true},
      {"name": "validate", "enabled": true},
      {"name": "judge", "enabled": true},
      {"name": "publish", "enabled": true},
      {"name": "relink_neighbors", "enabled": true},
      {"name": "sync_back", "enabled": true},
      {"name": "report", "enabled": true}
    ]
  }
}
```

`internal/domain/template/seed/category.json`:

```json
{
  "name": "Category",
  "pageKind": "category",
  "spec": {
    "sections": [
      {"heading": "Overview", "intent": "Say what the category contains and who shops it, in two short paragraphs above the product grid", "targetWords": 120, "required": true, "keywordRules": {"include": [], "primaryInHeading": true}},
      {"heading": "What You Will Find Here", "intent": "One paragraph per child category or product group, each linking to its page", "targetWords": 200, "required": true, "keywordRules": {"include": [], "primaryInHeading": false}},
      {"heading": "How to Choose", "intent": "The three criteria that separate the products in this category, written for a first-time buyer", "targetWords": 180, "required": true, "keywordRules": {"include": [], "primaryInHeading": false}}
    ],
    "tone": "Brief and helpful; written for a shopper scanning above and below a product grid; no repetition of product names already listed",
    "length": {"min": 400, "max": 700},
    "keywordRules": {"primaryInTitle": true, "primaryInH1": true, "primaryInFirstParagraph": true, "maxDensity": 0.03},
    "linkRules": {"upDepth": 1, "downLinks": true, "siblingMinWeight": 0.5, "maxLinks": 15, "maxPerTarget": 1, "parentLinkWithinParagraphs": 1, "childrenSection": true},
    "metaRules": {"titlePattern": "{primaryKeyword} | {siteName}", "descriptionMax": 155},
    "images": {"featured": false, "inline": 0, "source": ""},
    "modelProfiles": {},
    "recipe": [
      {"name": "resolve_context", "enabled": true},
      {"name": "generate_body", "enabled": true},
      {"name": "generate_meta", "enabled": true},
      {"name": "insert_links", "enabled": true},
      {"name": "repair_links", "enabled": true},
      {"name": "generate_images", "enabled": false},
      {"name": "validate", "enabled": true},
      {"name": "judge", "enabled": false},
      {"name": "publish", "enabled": true},
      {"name": "relink_neighbors", "enabled": true},
      {"name": "sync_back", "enabled": true},
      {"name": "report", "enabled": true}
    ]
  }
}
```

`internal/domain/template/seed.go`:

```go
package template

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
)

//go:embed seed/*.json
var seedFiles embed.FS

type seedFile struct {
	Name     string       `json:"name"`
	PageKind string       `json:"pageKind"`
	Spec     TemplateSpec `json:"spec"`
}

func Seed() []Template {
	names, err := fs.Glob(seedFiles, "seed/*.json")
	if err != nil {
		panic(fmt.Errorf("list seed templates: %w", err))
	}

	out := make([]Template, 0, len(names))
	for _, name := range names {
		body, readErr := seedFiles.ReadFile(name)
		if readErr != nil {
			panic(fmt.Errorf("read seed template %s: %w", name, readErr))
		}

		var file seedFile
		decoder := json.NewDecoder(bytes.NewReader(body))
		decoder.DisallowUnknownFields()
		if decodeErr := decoder.Decode(&file); decodeErr != nil {
			panic(fmt.Errorf("decode seed template %s: %w", name, decodeErr))
		}
		out = append(out, Template{Scope: ScopeGlobal, Name: file.Name, PageKind: file.PageKind, Version: 1, Spec: file.Spec})
	}
	return out
}
```

The panics are the only way an embedded, compile-time file can fail to decode, and `TestSeedShipsFiveValidGlobalTemplates` runs on every build, so a broken seed never reaches a binary.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race -cover ./internal/domain/template/ -v`
Expected: PASS, coverage ≥ 90%.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/template/seed.go internal/domain/template/seed/ internal/domain/template/seed_test.go
git commit -m "feat(domain): five embedded starter templates

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 11: Migrations 0004–0010

**Files:**
- Create: `internal/adapters/sqlite/migrations/0004_sites.sql`, `0005_link_policies.sql`, `0006_templates.sql`, `0007_entities.sql`, `0008_edges.sql`, `0009_pages.sql`, `0010_template_overrides.sql`
- Modify: `internal/adapters/sqlite/migrations_test.go`, `internal/adapters/sqlite/store_test.go`

**Interfaces:**
- Consumes: `migrations()`, `(*Store).provider()`, `openStore` (existing, package-internal).
- Produces: the nine tables of the Schema section; goose version 10.

- [ ] **Step 1: Extend the failing tests**

In `internal/adapters/sqlite/migrations_test.go` replace the `want` slice in `TestMigrationsAreEmbedded` with:

```go
	want := []string{
		"0001_app_meta.sql", "0002_settings.sql", "0003_secrets.sql",
		"0004_sites.sql", "0005_link_policies.sql", "0006_templates.sql", "0007_entities.sql",
		"0008_edges.sql", "0009_pages.sql", "0010_template_overrides.sql",
	}
```

In `TestMigrationsRoundTrip` change both `version != 3` checks to `version != 10` (and the messages to `want 10`), and replace the table list with:

```go
	for _, table := range []string{"app_meta", "settings", "secrets", "sites", "link_policies", "templates", "entities", "entity_anchors", "edges", "pages", "page_links", "template_overrides"} {
```

Append to the same file a test that proves the cascade map with rows present, which is the case a plain up/down/up on an empty file cannot exercise:

```go
func TestSchemaCascades(t *testing.T) {
	t.Parallel()

	store := openStore(t, nil)
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := store.writer.ExecContext(t.Context(), query, args...); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
	}
	count := func(table string) int {
		t.Helper()
		var n int
		if err := store.reader.QueryRowContext(t.Context(), "SELECT count(*) FROM "+table).Scan(&n); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		return n
	}
	const at = "2026-09-18T09:00:00Z"

	exec(`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES ('s1', 'Shop', 'https://shop', 'site:s1:wp_password', 'active', ?, ?)`, at, at)
	exec(`INSERT INTO templates (id, scope, site_id, name, page_kind, version, spec, created_at, updated_at) VALUES ('t1', 'global', NULL, 'Hub', 'hub', 1, '{}', ?, ?)`, at, at)
	exec(`INSERT INTO link_policies (id, scope, site_id, name, rules, forbid_external, forbid_self, anchor_strategy, created_at, updated_at) VALUES ('p1', 'site', 's1', 'Strict', '{}', 1, 1, 'rotate', ?, ?)`, at, at)
	exec(`UPDATE sites SET default_template_id = 't1', default_link_policy_id = 'p1' WHERE id = 's1'`)
	exec(`INSERT INTO entities (id, site_id, name, kind, source, created_at, updated_at) VALUES ('e1', 's1', 'Shoes', 'hub', 'user', ?, ?)`, at, at)
	exec(`INSERT INTO entities (id, site_id, name, kind, source, created_at, updated_at) VALUES ('e2', 's1', 'Boots', 'topic', 'user', ?, ?)`, at, at)
	exec(`INSERT INTO entity_anchors (entity_id, position, text, source, weight) VALUES ('e1', 0, 'shoes', 'user', 1)`)
	exec(`INSERT INTO edges (id, site_id, from_entity_id, to_entity_id, kind, weight, source, status, created_at) VALUES ('g1', 's1', 'e2', 'e1', 'parent', 1, 'user', 'approved', ?)`, at)
	exec(`INSERT INTO pages (id, site_id, path, slug, wp_type, status, entity_id, template_id, created_at, updated_at) VALUES ('pg1', 's1', '/shoes/', 'shoes', 'page', 'planned', 'e1', 't1', ?, ?)`, at, at)
	exec(`INSERT INTO pages (id, site_id, path, slug, parent_page_id, wp_type, status, entity_id, created_at, updated_at) VALUES ('pg2', 's1', '/shoes/boots/', 'boots', 'pg1', 'page', 'planned', 'e2', ?, ?)`, at, at)
	exec(`UPDATE entities SET canonical_page_id = 'pg1' WHERE id = 'e1'`)
	exec(`INSERT INTO page_links (id, site_id, from_page_id, to_page_id, to_url, anchor_text, origin, observed_at) VALUES ('l1', 's1', 'pg2', 'pg1', '/shoes/', 'shoes', 'generated', ?)`, at)
	exec(`INSERT INTO page_links (id, site_id, from_page_id, to_page_id, to_url, anchor_text, origin, observed_at) VALUES ('l2', 's1', 'pg1', 'pg2', '/shoes/boots/', 'boots', 'generated', ?)`, at)
	exec(`INSERT INTO template_overrides (id, template_id, scope, site_id, page_id, patch, created_at, updated_at) VALUES ('o1', 't1', 'site', 's1', NULL, '{}', ?, ?)`, at, at)
	exec(`INSERT INTO template_overrides (id, template_id, scope, site_id, page_id, patch, created_at, updated_at) VALUES ('o2', 't1', 'page', NULL, 'pg1', '{}', ?, ?)`, at, at)

	if _, err := store.writer.ExecContext(t.Context(), `INSERT INTO entities (id, site_id, name, kind, source, created_at, updated_at) VALUES ('e3', 's1', 'shoes', 'hub', 'user', ?, ?)`, at, at); err == nil {
		t.Fatal("entity names must be unique per site regardless of case")
	}
	if _, err := store.writer.ExecContext(t.Context(), `INSERT INTO template_overrides (id, template_id, scope, site_id, page_id, patch, created_at, updated_at) VALUES ('o3', 't1', 'site', 's1', NULL, '{}', ?, ?)`, at, at); err == nil {
		t.Fatal("one override per template and target")
	}
	if _, err := store.writer.ExecContext(t.Context(), `INSERT INTO edges (id, site_id, from_entity_id, to_entity_id, kind, weight, source, status, created_at) VALUES ('g2', 's1', 'e2', 'e1', 'related', 0.5, 'user', 'approved', ?)`, at); err == nil {
		t.Fatal("a related edge must be stored with the lower id first")
	}

	exec(`DELETE FROM pages WHERE id = 'pg1'`)
	var canonical, parent, toPage sql.NullString
	if err := store.reader.QueryRowContext(t.Context(), `SELECT canonical_page_id FROM entities WHERE id = 'e1'`).Scan(&canonical); err != nil || canonical.Valid {
		t.Errorf("canonical after page delete = %v, %v; want NULL", canonical, err)
	}
	if err := store.reader.QueryRowContext(t.Context(), `SELECT parent_page_id FROM pages WHERE id = 'pg2'`).Scan(&parent); err != nil || parent.Valid {
		t.Errorf("parent after page delete = %v, %v; want NULL", parent, err)
	}
	if err := store.reader.QueryRowContext(t.Context(), `SELECT to_page_id FROM page_links WHERE id = 'l1'`).Scan(&toPage); err != nil || toPage.Valid {
		t.Errorf("link target after page delete = %v, %v; want NULL", toPage, err)
	}
	if got := count("page_links"); got != 1 {
		t.Errorf("page_links after page delete = %d, want the incoming link only", got)
	}
	if got := count("template_overrides"); got != 1 {
		t.Errorf("template_overrides after page delete = %d, want the site override only", got)
	}

	exec(`DELETE FROM entities WHERE id = 'e1'`)
	if got := count("edges"); got != 0 {
		t.Errorf("edges after entity delete = %d", got)
	}
	if got := count("entity_anchors"); got != 0 {
		t.Errorf("anchors after entity delete = %d", got)
	}

	exec(`DELETE FROM templates WHERE id = 't1'`)
	var defaultTemplate sql.NullString
	if err := store.reader.QueryRowContext(t.Context(), `SELECT default_template_id FROM sites WHERE id = 's1'`).Scan(&defaultTemplate); err != nil || defaultTemplate.Valid {
		t.Errorf("default template after template delete = %v, %v; want NULL", defaultTemplate, err)
	}

	exec(`DELETE FROM sites WHERE id = 's1'`)
	for _, table := range []string{"entities", "pages", "page_links", "link_policies", "template_overrides"} {
		if got := count(table); got != 0 {
			t.Errorf("%s after site delete = %d", table, got)
		}
	}
}
```

Add `"database/sql"` to the imports of `migrations_test.go`. In `internal/adapters/sqlite/store_test.go`, `TestOpenCreatesTheSchema` keeps its three-table check; the new tables are covered by the round trip.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/adapters/sqlite/ -run 'TestMigrations|TestSchemaCascades' -v`
Expected: FAIL, the inventory lists three files and the cascade test cannot find `sites`.

- [ ] **Step 3: Scaffold and write the migrations**

```bash
for name in sites link_policies templates entities edges pages template_overrides; do
  go run github.com/pressly/goose/v3/cmd/goose@v3.28.0 -s -dir internal/adapters/sqlite/migrations create "$name" sql
done
git mv internal/adapters/sqlite/migrations/00004_sites.sql internal/adapters/sqlite/migrations/0004_sites.sql
git mv internal/adapters/sqlite/migrations/00005_link_policies.sql internal/adapters/sqlite/migrations/0005_link_policies.sql
git mv internal/adapters/sqlite/migrations/00006_templates.sql internal/adapters/sqlite/migrations/0006_templates.sql
git mv internal/adapters/sqlite/migrations/00007_entities.sql internal/adapters/sqlite/migrations/0007_entities.sql
git mv internal/adapters/sqlite/migrations/00008_edges.sql internal/adapters/sqlite/migrations/0008_edges.sql
git mv internal/adapters/sqlite/migrations/00009_pages.sql internal/adapters/sqlite/migrations/0009_pages.sql
git mv internal/adapters/sqlite/migrations/00010_template_overrides.sql internal/adapters/sqlite/migrations/0010_template_overrides.sql
```

If the goose CLI emits untracked files, `git add` them first and then `git mv`. Replace each file's body with the corresponding block from the **Schema** section above, verbatim.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -race ./internal/adapters/sqlite/ -v`
Expected: PASS, including the round trip to version 10 and back to 0.

- [ ] **Step 5: Commit**

```bash
git add internal/adapters/sqlite/migrations/ internal/adapters/sqlite/migrations_test.go
git commit -m "feat(sqlite): schema for sites, entity graph, page map, templates and link policies

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---
### Task 12: Row helpers, the site fixture and `SiteRepo`

**Files:**
- Create: `internal/adapters/sqlite/rows.go`, `internal/adapters/sqlite/site_repo.go`, `internal/adapters/sqlite/sqlitetest/fixtures.go`
- Test: `internal/adapters/sqlite/site_repo_test.go`

**Interfaces:**
- Consumes: `executor`, `(*Store).execFrom|writeFrom` (existing), `dbx.From|Convert|IsNotFound`, `(Result).NotFound|Conflict|WrapErr|Unwrap`, `paging.Keyset|TimeKey|TextKey|Request|List`, `site.Site|Query|Sort|Status`.
- Produces (package-internal, used by every repository task): `selectAll[T](ctx, exec executor, query string, args []any, scan func(*sql.Rows) (T, error), message string) (items []T, err error)`, `selectOne[T](ctx, exec, query, args, scan, notFound *errors.Error, message string) (T, error)`, `execWrite(ctx, exec, query string, args []any, conflict *errors.Error, message string) (int64, error)`, `requireAffected(affected int64, err error, notFound *errors.Error) error`, `formatTime`, `parseTime`, `nullTime`, `parseNullTime`, `nullString`, `optString`, `nullInt`, `optInt`, `boolInt`, `encodeJSON`, `decodeJSON`, `escapeLike`, `buildQuery(builder squirrel.SelectBuilder, what string) (query string, args []any, err error)`.
- Produces (exported): `SiteRepo` with `NewSiteRepo(store *Store) *SiteRepo`, `Insert(ctx, site.Site) error`, `Update(ctx, site.Site) error`, `Delete(ctx, id string) error`, `Get(ctx, id string) (site.Site, error)`, `List(ctx, q site.Query, page paging.Request) (paging.List[site.Site], error)`; `sqlitetest.Stamp time.Time` and `sqlitetest.Site(t testing.TB, store *sqlite.Store, name string) site.Site`.

- [ ] **Step 1: Write the failing test**

`internal/adapters/sqlite/site_repo_test.go`:

```go
package sqlite_test

import (
	stderrors "errors"
	"reflect"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func fullSite(name string, at time.Time) site.Site {
	record := site.Site{
		ID: id.New(), Name: name, BaseURL: "https://" + name + ".example.com", Username: "editor", Status: site.StatusPaused, AllowInsecure: true,
		Plugin:    site.PluginState{Installed: true, Version: "1.2.0", Capabilities: []string{"bulk", "seo_meta"}, SEOPlugin: "yoast"},
		Defaults:  site.Defaults{ModelProfiles: map[llm.Role]llm.ModelRef{llm.RoleWriter: {Provider: "openai", Model: "gpt"}}},
		CreatedAt: at, UpdatedAt: at,
	}
	record.SecretRef = site.SecretRef(record.ID)
	return record
}

func detailOf(t *testing.T, err error, key string) any {
	t.Helper()
	var kernel *errors.Error
	if !stderrors.As(err, &kernel) {
		t.Fatalf("error %v is not a kernel error", err)
	}
	return kernel.Details[key]
}

func TestSiteRepoRoundTrip(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	repo := sqlite.NewSiteRepo(store)
	want := fullSite("shop", sqlitetest.Stamp)

	if err := repo.Insert(t.Context(), want); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	got, err := repo.Get(t.Context(), want.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Get = %+v\nwant %+v", got, want)
	}

	if err = repo.Insert(t.Context(), want); !errors.IsCode(err, errors.Conflict) {
		t.Errorf("second Insert code = %q, want CONFLICT", errors.CodeOf(err))
	}

	want.Name = "Shop Two"
	want.Status = site.StatusActive
	templateID := "t1"
	want.Defaults.TemplateID = &templateID
	want.UpdatedAt = sqlitetest.Stamp.Add(time.Hour)
	if err = repo.Update(t.Context(), want); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Update with an unknown template must fail the foreign key as INVALID, got %v", err)
	}
	want.Defaults.TemplateID = nil
	if err = repo.Update(t.Context(), want); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, err = repo.Get(t.Context(), want.ID)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Get after update = %+v\nwant %+v", got, want)
	}

	if err = repo.Delete(t.Context(), want.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err = repo.Get(t.Context(), want.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Get after delete code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if detail := detailOf(t, err, "siteId"); detail != want.ID {
		t.Errorf("siteId detail = %v", detail)
	}
	if err = repo.Delete(t.Context(), want.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Delete twice code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if err = repo.Update(t.Context(), want); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Update after delete code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}

func TestSiteRepoListPagesForwardAndBackward(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	repo := sqlite.NewSiteRepo(store)
	names := []string{"alpha", "Bravo", "charlie", "Delta", "echo"}
	for i, name := range names {
		record := fullSite(name, sqlitetest.Stamp.Add(time.Duration(i)*time.Hour))
		if i%2 == 0 {
			record.Status = site.StatusActive
		}
		if err := repo.Insert(t.Context(), record); err != nil {
			t.Fatalf("Insert %s: %v", name, err)
		}
	}
	listNames := func(list paging.List[site.Site]) []string {
		out := make([]string, 0, len(list.Items))
		for _, item := range list.Items {
			out = append(out, item.Name)
		}
		return out
	}

	first, err := repo.List(t.Context(), site.Query{Sort: site.SortCreatedAt}, paging.Request{Limit: 2})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got := listNames(first); !reflect.DeepEqual(got, []string{"alpha", "Bravo"}) || !first.HasMore || first.Next == "" || first.Prev != "" {
		t.Fatalf("first page = %v, hasMore %v, next %q, prev %q", got, first.HasMore, first.Next, first.Prev)
	}

	second, err := repo.List(t.Context(), site.Query{Sort: site.SortCreatedAt}, paging.Request{After: first.Next, Limit: 2})
	if err != nil {
		t.Fatalf("List after: %v", err)
	}
	if got := listNames(second); !reflect.DeepEqual(got, []string{"charlie", "Delta"}) || !second.HasMore || second.Prev == "" {
		t.Fatalf("second page = %v, hasMore %v, prev %q", got, second.HasMore, second.Prev)
	}

	back, err := repo.List(t.Context(), site.Query{Sort: site.SortCreatedAt}, paging.Request{Before: second.Prev, Limit: 2})
	if err != nil {
		t.Fatalf("List before: %v", err)
	}
	if got := listNames(back); !reflect.DeepEqual(got, []string{"alpha", "Bravo"}) || back.HasMore {
		t.Fatalf("backward page = %v, hasMore %v", got, back.HasMore)
	}

	last, err := repo.List(t.Context(), site.Query{Sort: site.SortCreatedAt}, paging.Request{After: second.Next, Limit: 2})
	if err != nil {
		t.Fatalf("List last: %v", err)
	}
	if got := listNames(last); !reflect.DeepEqual(got, []string{"echo"}) || last.HasMore || last.Next != "" {
		t.Fatalf("last page = %v, hasMore %v, next %q", got, last.HasMore, last.Next)
	}

	byName, err := repo.List(t.Context(), site.Query{Sort: site.SortName}, paging.Request{Limit: 10})
	if err != nil {
		t.Fatalf("List by name: %v", err)
	}
	if got := listNames(byName); !reflect.DeepEqual(got, []string{"alpha", "Bravo", "charlie", "Delta", "echo"}) {
		t.Errorf("name order must be case-insensitive, got %v", got)
	}

	descending, err := repo.List(t.Context(), site.Query{Sort: site.SortName, Desc: true}, paging.Request{Limit: 2})
	if err != nil {
		t.Fatalf("List desc: %v", err)
	}
	if got := listNames(descending); !reflect.DeepEqual(got, []string{"echo", "Delta"}) {
		t.Errorf("descending = %v", got)
	}

	active := site.StatusActive
	filtered, err := repo.List(t.Context(), site.Query{Status: &active, Sort: site.SortCreatedAt}, paging.Request{Limit: 10})
	if err != nil {
		t.Fatalf("List filtered: %v", err)
	}
	if got := listNames(filtered); !reflect.DeepEqual(got, []string{"alpha", "charlie", "echo"}) {
		t.Errorf("filtered = %v", got)
	}

	if _, err = repo.List(t.Context(), site.Query{Sort: site.SortName}, paging.Request{After: first.Next, Limit: 2}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("a cursor from another sort must be INVALID, got %v", err)
	}
}

func TestSiteFixture(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	fixture := sqlitetest.Site(t, store, "shop")
	got, err := sqlite.NewSiteRepo(store).Get(t.Context(), fixture.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !reflect.DeepEqual(got, fixture) {
		t.Errorf("fixture round trip = %+v\nwant %+v", got, fixture)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapters/sqlite/ -run 'TestSiteRepo|TestSiteFixture' -v`
Expected: FAIL, `NewSiteRepo`, `sqlitetest.Site` and `sqlitetest.Stamp` are undefined.

- [ ] **Step 3: Write the helpers, the repository and the fixture**

`internal/adapters/sqlite/rows.go`:

```go
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"

	"github.com/davidmovas/postulator/internal/adapters/sqlite/dbx"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func selectAll[T any](ctx context.Context, exec executor, query string, args []any, scan func(*sql.Rows) (T, error), message string) (items []T, err error) {
	rows, err := exec.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, dbx.Convert(err, message)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = dbx.Convert(closeErr, message)
		}
	}()

	items = make([]T, 0)
	for rows.Next() {
		item, scanErr := scan(rows)
		if scanErr != nil {
			return nil, dbx.Convert(scanErr, message)
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		return nil, dbx.Convert(err, message)
	}
	return items, nil
}

func selectOne[T any](ctx context.Context, exec executor, query string, args []any, scan func(*sql.Rows) (T, error), notFound *errors.Error, message string) (T, error) {
	var item T
	items, err := selectAll(ctx, exec, query, args, scan, message)
	switch {
	case err == nil && len(items) == 0:
		err = sql.ErrNoRows
	case err == nil:
		item = items[0]
	}
	return dbx.From(item, err).
		NotFound(notFound).
		WrapErr(func(cause error) error { return dbx.Convert(cause, message) }).
		Unwrap()
}

func execWrite(ctx context.Context, exec executor, query string, args []any, conflict *errors.Error, message string) (int64, error) {
	convert := func(cause error) error { return dbx.Convert(cause, message) }

	result, err := exec.ExecContext(ctx, query, args...)
	if err != nil {
		outcome := dbx.From(int64(0), err)
		if conflict != nil {
			outcome = outcome.Conflict(conflict)
		}
		return outcome.WrapErr(convert).Unwrap()
	}

	affected, err := result.RowsAffected()
	return dbx.From(affected, err).WrapErr(convert).Unwrap()
}

func requireAffected(affected int64, err error, notFound *errors.Error) error {
	if err != nil {
		return err
	}
	if affected == 0 {
		return notFound
	}
	return nil
}

func buildQuery(builder squirrel.SelectBuilder, what string) (query string, args []any, err error) {
	query, args, err = builder.ToSql()
	if err != nil {
		return "", nil, errors.Wrap(err, errors.Internal, "build the "+what+" query")
	}
	return query, args, nil
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func parseTime(raw string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, errors.Wrap(err, errors.Internal, "stored timestamp is not rfc3339")
	}
	return parsed.UTC(), nil
}

func nullTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return formatTime(*t)
}

func parseNullTime(raw sql.NullString) (*time.Time, error) {
	if !raw.Valid {
		return nil, nil
	}
	parsed, err := parseTime(raw.String)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func nullString(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

func optString(raw sql.NullString) *string {
	if !raw.Valid {
		return nil
	}
	value := raw.String
	return &value
}

func nullInt(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}

func optInt(raw sql.NullInt64) *int64 {
	if !raw.Valid {
		return nil
	}
	value := raw.Int64
	return &value
}

func boolInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func encodeJSON(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", errors.Wrap(err, errors.Internal, "encode a json column")
	}
	return string(encoded), nil
}

func decodeJSON(raw string, into any, message string) error {
	if err := json.Unmarshal([]byte(raw), into); err != nil {
		return errors.Wrap(err, errors.Internal, message)
	}
	return nil
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func escapeLike(s string) string {
	return likeEscaper.Replace(s)
}
```

`internal/adapters/sqlite/site_repo.go`:

```go
package sqlite

import (
	"context"
	"database/sql"

	"github.com/Masterminds/squirrel"

	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	siteColumns = `id, name, base_url, username, secret_ref, status, allow_insecure, plugin_installed, plugin_version, plugin_capabilities, plugin_seo, default_template_id, default_link_policy_id, model_profiles, created_at, updated_at`
	insertSite  = `INSERT INTO sites (` + siteColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	updateSite  = `UPDATE sites SET name = ?, base_url = ?, username = ?, status = ?, allow_insecure = ?, plugin_installed = ?, plugin_version = ?, plugin_capabilities = ?, plugin_seo = ?, default_template_id = ?, default_link_policy_id = ?, model_profiles = ?, updated_at = ? WHERE id = ?`
	deleteSite  = `DELETE FROM sites WHERE id = ?`
	selectSite  = `SELECT ` + siteColumns + ` FROM sites WHERE id = ?`
)

type SiteRepo struct {
	store *Store
}

func NewSiteRepo(store *Store) *SiteRepo {
	return &SiteRepo{store: store}
}

func siteNotFound(id string) *errors.Error {
	return errors.New(errors.NotFound, "site not found").WithDetail("siteId", id)
}

func siteConflict(id string) *errors.Error {
	return errors.New(errors.Conflict, "a site with this id already exists").WithDetail("siteId", id)
}

type siteColumnsJSON struct {
	capabilities string
	profiles     string
}

func encodeSiteColumns(s site.Site) (siteColumnsJSON, error) {
	capabilities := s.Plugin.Capabilities
	if capabilities == nil {
		capabilities = []string{}
	}
	profiles := s.Defaults.ModelProfiles
	if profiles == nil {
		profiles = map[llm.Role]llm.ModelRef{}
	}

	encodedCapabilities, err := encodeJSON(capabilities)
	if err != nil {
		return siteColumnsJSON{}, err
	}
	encodedProfiles, err := encodeJSON(profiles)
	if err != nil {
		return siteColumnsJSON{}, err
	}
	return siteColumnsJSON{capabilities: encodedCapabilities, profiles: encodedProfiles}, nil
}

func (r *SiteRepo) Insert(ctx context.Context, s site.Site) error {
	encoded, err := encodeSiteColumns(s)
	if err != nil {
		return err
	}
	_, err = execWrite(ctx, r.store.writeFrom(ctx), insertSite, []any{
		s.ID, s.Name, s.BaseURL, s.Username, s.SecretRef, string(s.Status), boolInt(s.AllowInsecure),
		boolInt(s.Plugin.Installed), s.Plugin.Version, encoded.capabilities, s.Plugin.SEOPlugin,
		nullString(s.Defaults.TemplateID), nullString(s.Defaults.LinkPolicyID), encoded.profiles,
		formatTime(s.CreatedAt), formatTime(s.UpdatedAt),
	}, siteConflict(s.ID), "insert the site")
	return err
}

func (r *SiteRepo) Update(ctx context.Context, s site.Site) error {
	encoded, err := encodeSiteColumns(s)
	if err != nil {
		return err
	}
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), updateSite, []any{
		s.Name, s.BaseURL, s.Username, string(s.Status), boolInt(s.AllowInsecure),
		boolInt(s.Plugin.Installed), s.Plugin.Version, encoded.capabilities, s.Plugin.SEOPlugin,
		nullString(s.Defaults.TemplateID), nullString(s.Defaults.LinkPolicyID), encoded.profiles,
		formatTime(s.UpdatedAt), s.ID,
	}, nil, "update the site")
	return requireAffected(affected, err, siteNotFound(s.ID))
}

func (r *SiteRepo) Delete(ctx context.Context, id string) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), deleteSite, []any{id}, nil, "delete the site")
	return requireAffected(affected, err, siteNotFound(id))
}

func (r *SiteRepo) Get(ctx context.Context, id string) (site.Site, error) {
	return selectOne(ctx, r.store.execFrom(ctx), selectSite, []any{id}, scanSite, siteNotFound(id), "read the site")
}

func siteKeyset(q site.Query) paging.Keyset[site.Site] {
	key := paging.TimeKey[site.Site]("createdAt", "created_at", func(s site.Site) any { return s.CreatedAt })
	if q.Sort == site.SortName {
		key = paging.TextKey[site.Site]("name", "name", func(s site.Site) any { return s.Name })
	}
	return paging.Keyset[site.Site]{
		IDColumn: "id",
		ID:       func(s site.Site) string { return s.ID },
		Keys:     []paging.SortKey[site.Site]{key},
		Desc:     q.Desc,
	}
}

func (r *SiteRepo) List(ctx context.Context, q site.Query, page paging.Request) (paging.List[site.Site], error) {
	builder := squirrel.Select(siteColumns).From("sites")
	if q.Status != nil {
		builder = builder.Where(squirrel.Eq{"status": string(*q.Status)})
	}

	keyset := siteKeyset(q)
	keyed, err := keyset.Apply(builder, page)
	if err != nil {
		return paging.List[site.Site]{}, err
	}
	query, args, err := buildQuery(keyed, "sites")
	if err != nil {
		return paging.List[site.Site]{}, err
	}
	rows, err := selectAll(ctx, r.store.execFrom(ctx), query, args, scanSite, "list the sites")
	if err != nil {
		return paging.List[site.Site]{}, err
	}
	return keyset.Cut(rows, page)
}

func scanSite(rows *sql.Rows) (site.Site, error) {
	var (
		s                             site.Site
		status                        string
		allowInsecure, pluginInstalled int64
		capabilities, profiles        string
		templateID, policyID          sql.NullString
		createdAt, updatedAt          string
	)
	if err := rows.Scan(&s.ID, &s.Name, &s.BaseURL, &s.Username, &s.SecretRef, &status, &allowInsecure, &pluginInstalled,
		&s.Plugin.Version, &capabilities, &s.Plugin.SEOPlugin, &templateID, &policyID, &profiles, &createdAt, &updatedAt); err != nil {
		return site.Site{}, err
	}

	s.Status = site.Status(status)
	s.AllowInsecure = allowInsecure == 1
	s.Plugin.Installed = pluginInstalled == 1
	s.Defaults.TemplateID = optString(templateID)
	s.Defaults.LinkPolicyID = optString(policyID)
	if err := decodeJSON(capabilities, &s.Plugin.Capabilities, "decode the site plugin capabilities"); err != nil {
		return site.Site{}, err
	}
	if err := decodeJSON(profiles, &s.Defaults.ModelProfiles, "decode the site model profiles"); err != nil {
		return site.Site{}, err
	}

	var err error
	if s.CreatedAt, err = parseTime(createdAt); err != nil {
		return site.Site{}, err
	}
	if s.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return site.Site{}, err
	}
	return s, nil
}
```

`internal/adapters/sqlite/sqlitetest/fixtures.go`:

```go
package sqlitetest

import (
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

var Stamp = time.Date(2026, time.September, 18, 9, 0, 0, 0, time.UTC)

func Site(t testing.TB, store *sqlite.Store, name string) site.Site {
	t.Helper()

	record := site.Site{
		ID:        id.New(),
		Name:      name,
		BaseURL:   "https://" + name + ".example.com",
		Username:  "editor",
		Status:    site.StatusActive,
		Plugin:    site.PluginState{Capabilities: []string{}},
		Defaults:  site.Defaults{ModelProfiles: map[llm.Role]llm.ModelRef{}},
		CreatedAt: Stamp,
		UpdatedAt: Stamp,
	}
	record.SecretRef = site.SecretRef(record.ID)
	if err := sqlite.NewSiteRepo(store).Insert(t.Context(), record); err != nil {
		t.Fatalf("insert the site fixture: %v", err)
	}
	return record
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race ./internal/adapters/sqlite/... -v`
Expected: PASS. The `Update` with an unknown `default_template_id` fails the foreign key, which `dbx.Classify` maps to `INVALID`.

- [ ] **Step 5: Commit**

```bash
git add internal/adapters/sqlite/rows.go internal/adapters/sqlite/site_repo.go internal/adapters/sqlite/site_repo_test.go internal/adapters/sqlite/sqlitetest/fixtures.go
git commit -m "feat(sqlite): site repository with keyset lists and shared row helpers

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---
### Task 13: `EntityRepo` with anchors

**Files:**
- Create: `internal/adapters/sqlite/entity_repo.go`
- Modify: `internal/adapters/sqlite/sqlitetest/fixtures.go`
- Test: `internal/adapters/sqlite/entity_repo_test.go`

**Interfaces:**
- Consumes: Task 12 helpers; `graph.Entity|Anchor|EntityQuery|EntitySort|Kind|Source|AnchorSource`.
- Produces: `EntityRepo` with `NewEntityRepo(store *Store) *EntityRepo`, `Insert(ctx, graph.Entity) error`, `Update(ctx, graph.Entity) error` (rewrites anchors), `Delete(ctx, id string) error`, `Get(ctx, id string) (graph.Entity, error)`, `List(ctx, q graph.EntityQuery, page paging.Request) (paging.List[graph.Entity], error)`, `ListBySite(ctx, siteID string) ([]graph.Entity, error)` (name order), `SetScore(ctx, id string, score float64) error`, `SetCanonicalPage(ctx, id string, pageID *string, updatedAt time.Time) error`; `sqlitetest.Entity(t testing.TB, store *sqlite.Store, siteID, name string) graph.Entity`.

- [ ] **Step 1: Write the failing test**

```go
package sqlite_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func fullEntity(siteID, name string, at time.Time) graph.Entity {
	return graph.Entity{
		ID: id.New(), SiteID: siteID, Name: name, Kind: graph.KindTopic, Intent: "informational", PrimaryKeyword: name + " keyword",
		SecondaryKeywords: []string{name + " one", name + " two"},
		Anchors:           []graph.Anchor{{Text: name, Source: graph.AnchorUser, Weight: 1}, {Text: "best " + name, Source: graph.AnchorAI, Weight: 0.4}},
		Score:             0.5, Source: graph.SourceImport, CreatedAt: at, UpdatedAt: at,
	}
}

func entityNames(list []graph.Entity) []string {
	out := make([]string, 0, len(list))
	for _, item := range list {
		out = append(out, item.Name)
	}
	return out
}

func TestEntityRepoRoundTrip(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	repo := sqlite.NewEntityRepo(store)
	want := fullEntity(owner.ID, "Shoes", sqlitetest.Stamp)

	if err := repo.Insert(t.Context(), want); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	got, err := repo.Get(t.Context(), want.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Get = %+v\nwant %+v", got, want)
	}

	duplicate := fullEntity(owner.ID, "shoes", sqlitetest.Stamp)
	if err = repo.Insert(t.Context(), duplicate); !errors.IsCode(err, errors.Conflict) {
		t.Errorf("case-insensitive duplicate code = %q, want CONFLICT", errors.CodeOf(err))
	}
	orphan := fullEntity("no-such-site", "Orphan", sqlitetest.Stamp)
	if err = repo.Insert(t.Context(), orphan); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("unknown site code = %q, want INVALID", errors.CodeOf(err))
	}

	want.Name = "Running Shoes"
	want.Anchors = []graph.Anchor{{Text: "running shoes", Source: graph.AnchorUser, Weight: 0.9}}
	want.SecondaryKeywords = []string{}
	want.UpdatedAt = sqlitetest.Stamp.Add(time.Minute)
	if err = repo.Update(t.Context(), want); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, err = repo.Get(t.Context(), want.ID)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Get after update = %+v\nwant %+v", got, want)
	}

	if err = repo.SetScore(t.Context(), want.ID, 0.25); err != nil {
		t.Fatalf("SetScore: %v", err)
	}
	pageID := "not-a-page"
	if err = repo.SetCanonicalPage(t.Context(), want.ID, &pageID, want.UpdatedAt); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("canonical to an unknown page code = %q, want INVALID", errors.CodeOf(err))
	}
	if err = repo.SetCanonicalPage(t.Context(), want.ID, nil, want.UpdatedAt); err != nil {
		t.Fatalf("SetCanonicalPage nil: %v", err)
	}
	got, err = repo.Get(t.Context(), want.ID)
	if err != nil {
		t.Fatalf("Get after score: %v", err)
	}
	if got.Score != 0.25 || got.CanonicalPageID != nil {
		t.Errorf("score = %v canonical = %v", got.Score, got.CanonicalPageID)
	}

	if err = repo.SetScore(t.Context(), "missing", 1); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("SetScore unknown code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if err = repo.Delete(t.Context(), want.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err = repo.Delete(t.Context(), want.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Delete twice code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if err = repo.Update(t.Context(), want); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Update missing code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = repo.Get(t.Context(), want.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Get missing code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}

func TestEntityRepoListAndFilters(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	other := sqlitetest.Site(t, store, "blog")
	repo := sqlite.NewEntityRepo(store)

	names := []string{"Boots", "sandals", "Shoes", "slippers", "Trainers"}
	for i, name := range names {
		record := fullEntity(owner.ID, name, sqlitetest.Stamp.Add(time.Duration(i)*time.Minute))
		if i%2 == 1 {
			record.Kind = graph.KindProduct
		}
		if err := repo.Insert(t.Context(), record); err != nil {
			t.Fatalf("Insert %s: %v", name, err)
		}
	}
	if err := repo.Insert(t.Context(), fullEntity(other.ID, "Shoes", sqlitetest.Stamp)); err != nil {
		t.Fatalf("Insert on the other site: %v", err)
	}

	bySite, err := repo.ListBySite(t.Context(), owner.ID)
	if err != nil {
		t.Fatalf("ListBySite: %v", err)
	}
	if got := entityNames(bySite); !reflect.DeepEqual(got, []string{"Boots", "sandals", "Shoes", "slippers", "Trainers"}) {
		t.Errorf("ListBySite = %v", got)
	}
	for _, item := range bySite {
		if len(item.Anchors) != 2 {
			t.Errorf("%s carries %d anchors, want 2", item.Name, len(item.Anchors))
		}
	}

	byCreation := graph.EntityQuery{SiteID: owner.ID, Sort: graph.EntitySortCreatedAt}
	first, err := repo.List(t.Context(), byCreation, paging.Request{Limit: 2})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got := entityNames(first.Items); !reflect.DeepEqual(got, []string{"Boots", "sandals"}) || !first.HasMore {
		t.Fatalf("first page = %v", got)
	}
	if len(first.Items[0].Anchors) != 2 {
		t.Errorf("list pages must carry anchors, got %d", len(first.Items[0].Anchors))
	}
	second, err := repo.List(t.Context(), byCreation, paging.Request{After: first.Next, Limit: 2})
	if err != nil {
		t.Fatalf("List after: %v", err)
	}
	if got := entityNames(second.Items); !reflect.DeepEqual(got, []string{"Shoes", "slippers"}) || !second.HasMore {
		t.Fatalf("second page = %v", got)
	}
	back, err := repo.List(t.Context(), byCreation, paging.Request{Before: second.Prev, Limit: 2})
	if err != nil {
		t.Fatalf("List before: %v", err)
	}
	if got := entityNames(back.Items); !reflect.DeepEqual(got, []string{"Boots", "sandals"}) {
		t.Fatalf("backward page = %v", got)
	}

	product := graph.KindProduct
	products, err := repo.List(t.Context(), graph.EntityQuery{SiteID: owner.ID, Kind: &product, Sort: graph.EntitySortName}, paging.Request{Limit: 10})
	if err != nil {
		t.Fatalf("List products: %v", err)
	}
	if got := entityNames(products.Items); !reflect.DeepEqual(got, []string{"sandals", "slippers"}) {
		t.Errorf("products = %v", got)
	}

	prefixed, err := repo.List(t.Context(), graph.EntityQuery{SiteID: owner.ID, NamePrefix: "s", Sort: graph.EntitySortName}, paging.Request{Limit: 10})
	if err != nil {
		t.Fatalf("List prefixed: %v", err)
	}
	if got := entityNames(prefixed.Items); !reflect.DeepEqual(got, []string{"sandals", "Shoes", "slippers"}) {
		t.Errorf("prefix search must be case-insensitive and name-ordered, got %v", got)
	}
	wild, err := repo.List(t.Context(), graph.EntityQuery{SiteID: owner.ID, NamePrefix: "%", Sort: graph.EntitySortName}, paging.Request{Limit: 10})
	if err != nil || len(wild.Items) != 0 {
		t.Errorf("a literal percent must be escaped and match nothing, got %d items, %v", len(wild.Items), err)
	}

	everySite, err := repo.List(t.Context(), graph.EntityQuery{NamePrefix: "sho", Sort: graph.EntitySortName}, paging.Request{Limit: 10})
	if err != nil || len(everySite.Items) != 2 {
		t.Errorf("an empty site id spans every site, got %d, %v", len(everySite.Items), err)
	}

	withCanonical := true
	none, err := repo.List(t.Context(), graph.EntityQuery{SiteID: owner.ID, HasCanonicalPage: &withCanonical, Sort: graph.EntitySortName}, paging.Request{Limit: 10})
	if err != nil || len(none.Items) != 0 {
		t.Errorf("no entity has a canonical page yet, got %d, %v", len(none.Items), err)
	}
	withoutCanonical := false
	all, err := repo.List(t.Context(), graph.EntityQuery{SiteID: owner.ID, HasCanonicalPage: &withoutCanonical, Sort: graph.EntitySortName}, paging.Request{Limit: 10})
	if err != nil || len(all.Items) != 5 {
		t.Errorf("every entity lacks a canonical page, got %d, %v", len(all.Items), err)
	}
}

func TestEntityFixture(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	fixture := sqlitetest.Entity(t, store, owner.ID, "Shoes")
	got, err := sqlite.NewEntityRepo(store).Get(t.Context(), fixture.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !reflect.DeepEqual(got, fixture) {
		t.Errorf("fixture round trip = %+v\nwant %+v", got, fixture)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapters/sqlite/ -run 'TestEntityRepo|TestEntityFixture' -v`
Expected: FAIL, `NewEntityRepo` and `sqlitetest.Entity` are undefined.

- [ ] **Step 3: Write the repository and the fixture**

`internal/adapters/sqlite/entity_repo.go`:

```go
package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/Masterminds/squirrel"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	entityColumns         = `id, site_id, name, kind, intent, primary_keyword, secondary_keywords, canonical_page_id, score, source, created_at, updated_at`
	insertEntity          = `INSERT INTO entities (` + entityColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	updateEntity          = `UPDATE entities SET name = ?, kind = ?, intent = ?, primary_keyword = ?, secondary_keywords = ?, canonical_page_id = ?, score = ?, source = ?, updated_at = ? WHERE id = ?`
	updateEntityScore     = `UPDATE entities SET score = ? WHERE id = ?`
	updateEntityCanonical = `UPDATE entities SET canonical_page_id = ?, updated_at = ? WHERE id = ?`
	deleteEntity          = `DELETE FROM entities WHERE id = ?`
	selectEntity          = `SELECT ` + entityColumns + ` FROM entities WHERE id = ?`
	selectEntitiesBySite  = `SELECT ` + entityColumns + ` FROM entities WHERE site_id = ? ORDER BY name, id`
	deleteAnchors         = `DELETE FROM entity_anchors WHERE entity_id = ?`
	insertAnchor          = `INSERT INTO entity_anchors (entity_id, position, text, source, weight) VALUES (?, ?, ?, ?, ?)`
	selectAnchorsBySite   = `SELECT a.entity_id, a.text, a.source, a.weight FROM entity_anchors a JOIN entities e ON e.id = a.entity_id WHERE e.site_id = ? ORDER BY a.entity_id, a.position`
)

type EntityRepo struct {
	store *Store
}

func NewEntityRepo(store *Store) *EntityRepo {
	return &EntityRepo{store: store}
}

func entityNotFound(id string) *errors.Error {
	return errors.New(errors.NotFound, "entity not found").WithDetail("entityId", id)
}

func entityConflict(name string) *errors.Error {
	return errors.New(errors.Conflict, "an entity with this name already exists in the site").WithDetail("name", name)
}

func anchorConflict(entityID string) *errors.Error {
	return errors.New(errors.Conflict, "an anchor text is repeated").WithDetail("entityId", entityID)
}

func (r *EntityRepo) Insert(ctx context.Context, e graph.Entity) error {
	keywords, err := encodeJSON(orEmpty(e.SecondaryKeywords))
	if err != nil {
		return err
	}
	if _, err = execWrite(ctx, r.store.writeFrom(ctx), insertEntity, []any{
		e.ID, e.SiteID, e.Name, string(e.Kind), e.Intent, e.PrimaryKeyword, keywords, nullString(e.CanonicalPageID),
		e.Score, string(e.Source), formatTime(e.CreatedAt), formatTime(e.UpdatedAt),
	}, entityConflict(e.Name), "insert the entity"); err != nil {
		return err
	}
	return r.writeAnchors(ctx, e.ID, e.Anchors)
}

func (r *EntityRepo) Update(ctx context.Context, e graph.Entity) error {
	keywords, err := encodeJSON(orEmpty(e.SecondaryKeywords))
	if err != nil {
		return err
	}
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), updateEntity, []any{
		e.Name, string(e.Kind), e.Intent, e.PrimaryKeyword, keywords, nullString(e.CanonicalPageID),
		e.Score, string(e.Source), formatTime(e.UpdatedAt), e.ID,
	}, entityConflict(e.Name), "update the entity")
	if err = requireAffected(affected, err, entityNotFound(e.ID)); err != nil {
		return err
	}
	if _, err = execWrite(ctx, r.store.writeFrom(ctx), deleteAnchors, []any{e.ID}, nil, "clear the entity anchors"); err != nil {
		return err
	}
	return r.writeAnchors(ctx, e.ID, e.Anchors)
}

func (r *EntityRepo) writeAnchors(ctx context.Context, entityID string, anchors []graph.Anchor) error {
	for position, anchor := range anchors {
		if _, err := execWrite(ctx, r.store.writeFrom(ctx), insertAnchor, []any{
			entityID, position, anchor.Text, string(anchor.Source), anchor.Weight,
		}, anchorConflict(entityID), "insert an entity anchor"); err != nil {
			return err
		}
	}
	return nil
}

func orEmpty(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func (r *EntityRepo) Delete(ctx context.Context, id string) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), deleteEntity, []any{id}, nil, "delete the entity")
	return requireAffected(affected, err, entityNotFound(id))
}

func (r *EntityRepo) SetScore(ctx context.Context, id string, score float64) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), updateEntityScore, []any{score, id}, nil, "update the entity score")
	return requireAffected(affected, err, entityNotFound(id))
}

func (r *EntityRepo) SetCanonicalPage(ctx context.Context, id string, pageID *string, updatedAt time.Time) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), updateEntityCanonical, []any{nullString(pageID), formatTime(updatedAt), id}, nil, "update the entity canonical page")
	return requireAffected(affected, err, entityNotFound(id))
}

func (r *EntityRepo) Get(ctx context.Context, id string) (graph.Entity, error) {
	entity, err := selectOne(ctx, r.store.execFrom(ctx), selectEntity, []any{id}, scanEntity, entityNotFound(id), "read the entity")
	if err != nil {
		return graph.Entity{}, err
	}
	entities := []graph.Entity{entity}
	if err = r.attachAnchorsByID(ctx, entities); err != nil {
		return graph.Entity{}, err
	}
	return entities[0], nil
}

func (r *EntityRepo) ListBySite(ctx context.Context, siteID string) ([]graph.Entity, error) {
	entities, err := selectAll(ctx, r.store.execFrom(ctx), selectEntitiesBySite, []any{siteID}, scanEntity, "list the site entities")
	if err != nil {
		return nil, err
	}
	rows, err := selectAll(ctx, r.store.execFrom(ctx), selectAnchorsBySite, []any{siteID}, scanAnchor, "list the site anchors")
	if err != nil {
		return nil, err
	}
	attachAnchors(entities, rows)
	return entities, nil
}

func entityKeyset(q graph.EntityQuery) paging.Keyset[graph.Entity] {
	key := paging.TimeKey[graph.Entity]("createdAt", "created_at", func(e graph.Entity) any { return e.CreatedAt })
	if q.Sort == graph.EntitySortName {
		key = paging.TextKey[graph.Entity]("name", "name", func(e graph.Entity) any { return e.Name })
	}
	return paging.Keyset[graph.Entity]{
		IDColumn: "id",
		ID:       func(e graph.Entity) string { return e.ID },
		Keys:     []paging.SortKey[graph.Entity]{key},
		Desc:     q.Desc,
	}
}

func (r *EntityRepo) List(ctx context.Context, q graph.EntityQuery, page paging.Request) (paging.List[graph.Entity], error) {
	builder := squirrel.Select(entityColumns).From("entities")
	if q.SiteID != "" {
		builder = builder.Where(squirrel.Eq{"site_id": q.SiteID})
	}
	if q.Kind != nil {
		builder = builder.Where(squirrel.Eq{"kind": string(*q.Kind)})
	}
	if q.HasCanonicalPage != nil {
		if *q.HasCanonicalPage {
			builder = builder.Where("canonical_page_id IS NOT NULL")
		} else {
			builder = builder.Where("canonical_page_id IS NULL")
		}
	}
	if q.NamePrefix != "" {
		builder = builder.Where(`name LIKE ? ESCAPE '\'`, escapeLike(q.NamePrefix)+"%")
	}

	keyset := entityKeyset(q)
	keyed, err := keyset.Apply(builder, page)
	if err != nil {
		return paging.List[graph.Entity]{}, err
	}
	query, args, err := buildQuery(keyed, "entities")
	if err != nil {
		return paging.List[graph.Entity]{}, err
	}
	entities, err := selectAll(ctx, r.store.execFrom(ctx), query, args, scanEntity, "list the entities")
	if err != nil {
		return paging.List[graph.Entity]{}, err
	}
	if err = r.attachAnchorsByID(ctx, entities); err != nil {
		return paging.List[graph.Entity]{}, err
	}
	return keyset.Cut(entities, page)
}

func (r *EntityRepo) attachAnchorsByID(ctx context.Context, entities []graph.Entity) error {
	if len(entities) == 0 {
		return nil
	}
	ids := make([]string, 0, len(entities))
	for _, e := range entities {
		ids = append(ids, e.ID)
	}
	query, args, err := buildQuery(squirrel.Select("entity_id", "text", "source", "weight").From("entity_anchors").
		Where(squirrel.Eq{"entity_id": ids}).OrderBy("entity_id", "position"), "entity anchors")
	if err != nil {
		return err
	}
	rows, err := selectAll(ctx, r.store.execFrom(ctx), query, args, scanAnchor, "list the entity anchors")
	if err != nil {
		return err
	}
	attachAnchors(entities, rows)
	return nil
}

type anchorRow struct {
	entityID string
	anchor   graph.Anchor
}

func attachAnchors(entities []graph.Entity, rows []anchorRow) {
	byEntity := make(map[string][]graph.Anchor, len(entities))
	for _, row := range rows {
		byEntity[row.entityID] = append(byEntity[row.entityID], row.anchor)
	}
	for i := range entities {
		anchors := byEntity[entities[i].ID]
		if anchors == nil {
			anchors = []graph.Anchor{}
		}
		entities[i].Anchors = anchors
	}
}

func scanAnchor(rows *sql.Rows) (anchorRow, error) {
	var (
		row    anchorRow
		source string
	)
	if err := rows.Scan(&row.entityID, &row.anchor.Text, &source, &row.anchor.Weight); err != nil {
		return anchorRow{}, err
	}
	row.anchor.Source = graph.AnchorSource(source)
	return row, nil
}

func scanEntity(rows *sql.Rows) (graph.Entity, error) {
	var (
		e                    graph.Entity
		kind, source         string
		keywords             string
		canonical            sql.NullString
		createdAt, updatedAt string
	)
	if err := rows.Scan(&e.ID, &e.SiteID, &e.Name, &kind, &e.Intent, &e.PrimaryKeyword, &keywords, &canonical, &e.Score, &source, &createdAt, &updatedAt); err != nil {
		return graph.Entity{}, err
	}
	e.Kind = graph.Kind(kind)
	e.Source = graph.Source(source)
	e.CanonicalPageID = optString(canonical)
	if err := decodeJSON(keywords, &e.SecondaryKeywords, "decode the entity keywords"); err != nil {
		return graph.Entity{}, err
	}

	var err error
	if e.CreatedAt, err = parseTime(createdAt); err != nil {
		return graph.Entity{}, err
	}
	if e.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return graph.Entity{}, err
	}
	return e, nil
}
```

Append to `internal/adapters/sqlite/sqlitetest/fixtures.go` (add `"github.com/davidmovas/postulator/internal/domain/graph"` to its imports):

```go
func Entity(t testing.TB, store *sqlite.Store, siteID, name string) graph.Entity {
	t.Helper()

	record := graph.Entity{
		ID: id.New(), SiteID: siteID, Name: name, Kind: graph.KindTopic, PrimaryKeyword: name,
		SecondaryKeywords: []string{}, Anchors: []graph.Anchor{{Text: name, Source: graph.AnchorUser, Weight: 1}},
		Source: graph.SourceUser, CreatedAt: Stamp, UpdatedAt: Stamp,
	}
	if err := sqlite.NewEntityRepo(store).Insert(t.Context(), record); err != nil {
		t.Fatalf("insert the entity fixture: %v", err)
	}
	return record
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race ./internal/adapters/sqlite/... -v`
Expected: PASS. Confirm with `go test -race ./internal/adapters/sqlite/ -run TestEntityRepoListAndFilters -v` that the page of two entities issued exactly two statements per page by reading the code path: `List` runs the keyset select, then one `IN (...)` select.

- [ ] **Step 5: Commit**

```bash
git add internal/adapters/sqlite/entity_repo.go internal/adapters/sqlite/entity_repo_test.go internal/adapters/sqlite/sqlitetest/fixtures.go
git commit -m "feat(sqlite): entity repository loading anchors in one query per page

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 14: `EdgeRepo`

**Files:**
- Create: `internal/adapters/sqlite/edge_repo.go`
- Test: `internal/adapters/sqlite/edge_repo_test.go`

**Interfaces:**
- Consumes: Task 12 helpers; `graph.Edge|EdgeQuery|EdgeKind|EdgeStatus`.
- Produces: `EdgeRepo` with `NewEdgeRepo(store *Store) *EdgeRepo`, `Insert(ctx, graph.Edge) error`, `Get(ctx, id string) (graph.Edge, error)`, `Delete(ctx, id string) error`, `SetStatus(ctx, id string, status graph.EdgeStatus) error`, `List(ctx, q graph.EdgeQuery, page paging.Request) (paging.List[graph.Edge], error)`, `ListBySite(ctx, siteID string) ([]graph.Edge, error)` (created_at, id order).

- [ ] **Step 1: Write the failing test**

```go
package sqlite_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func edgeBetween(siteID, from, to string, kind graph.EdgeKind, status graph.EdgeStatus, at time.Time) graph.Edge {
	edge, err := graph.NewEdge(graph.Edge{ID: id.New(), SiteID: siteID, FromEntityID: from, ToEntityID: to, Kind: kind, Weight: 0.6, Source: graph.SourceAI, Status: status, CreatedAt: at})
	if err != nil {
		panic(err)
	}
	return edge
}

func edgeIDs(edges []graph.Edge) []string {
	out := make([]string, 0, len(edges))
	for _, edge := range edges {
		out = append(out, edge.ID)
	}
	return out
}

func TestEdgeRepoRoundTrip(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	hub := sqlitetest.Entity(t, store, owner.ID, "Shoes")
	child := sqlitetest.Entity(t, store, owner.ID, "Boots")
	repo := sqlite.NewEdgeRepo(store)

	want := edgeBetween(owner.ID, child.ID, hub.ID, graph.EdgeParent, graph.StatusProposed, sqlitetest.Stamp)
	if err := repo.Insert(t.Context(), want); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	got, err := repo.Get(t.Context(), want.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Get = %+v\nwant %+v", got, want)
	}

	again := edgeBetween(owner.ID, child.ID, hub.ID, graph.EdgeParent, graph.StatusApproved, sqlitetest.Stamp)
	if err = repo.Insert(t.Context(), again); !errors.IsCode(err, errors.Conflict) {
		t.Errorf("duplicate edge code = %q, want CONFLICT", errors.CodeOf(err))
	}
	stranger := edgeBetween(owner.ID, child.ID, "no-such-entity", graph.EdgeParent, graph.StatusApproved, sqlitetest.Stamp)
	if err = repo.Insert(t.Context(), stranger); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("unknown endpoint code = %q, want INVALID", errors.CodeOf(err))
	}

	if err = repo.SetStatus(t.Context(), want.ID, graph.StatusApproved); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}
	got, err = repo.Get(t.Context(), want.ID)
	if err != nil || got.Status != graph.StatusApproved {
		t.Errorf("status after SetStatus = %q, %v", got.Status, err)
	}
	if err = repo.SetStatus(t.Context(), "missing", graph.StatusRejected); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("SetStatus missing code = %q, want NOT_FOUND", errors.CodeOf(err))
	}

	if err = repo.Delete(t.Context(), want.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err = repo.Delete(t.Context(), want.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Delete twice code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = repo.Get(t.Context(), want.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Get missing code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}

func TestEdgeRepoListAndFilters(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	hub := sqlitetest.Entity(t, store, owner.ID, "Shoes")
	boots := sqlitetest.Entity(t, store, owner.ID, "Boots")
	sandals := sqlitetest.Entity(t, store, owner.ID, "Sandals")
	repo := sqlite.NewEdgeRepo(store)

	edges := []graph.Edge{
		edgeBetween(owner.ID, boots.ID, hub.ID, graph.EdgeParent, graph.StatusApproved, sqlitetest.Stamp),
		edgeBetween(owner.ID, sandals.ID, hub.ID, graph.EdgeParent, graph.StatusProposed, sqlitetest.Stamp.Add(time.Minute)),
		edgeBetween(owner.ID, boots.ID, sandals.ID, graph.EdgeRelated, graph.StatusApproved, sqlitetest.Stamp.Add(2*time.Minute)),
	}
	for _, edge := range edges {
		if err := repo.Insert(t.Context(), edge); err != nil {
			t.Fatalf("Insert: %v", err)
		}
	}

	all, err := repo.ListBySite(t.Context(), owner.ID)
	if err != nil {
		t.Fatalf("ListBySite: %v", err)
	}
	if got := edgeIDs(all); !reflect.DeepEqual(got, edgeIDs(edges)) {
		t.Errorf("ListBySite = %v, want creation order", got)
	}

	first, err := repo.List(t.Context(), graph.EdgeQuery{SiteID: owner.ID}, paging.Request{Limit: 2})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got := edgeIDs(first.Items); !reflect.DeepEqual(got, edgeIDs(edges[:2])) || !first.HasMore {
		t.Fatalf("first page = %v", got)
	}
	rest, err := repo.List(t.Context(), graph.EdgeQuery{SiteID: owner.ID}, paging.Request{After: first.Next, Limit: 2})
	if err != nil {
		t.Fatalf("List after: %v", err)
	}
	if got := edgeIDs(rest.Items); !reflect.DeepEqual(got, edgeIDs(edges[2:])) || rest.HasMore {
		t.Fatalf("second page = %v", got)
	}
	back, err := repo.List(t.Context(), graph.EdgeQuery{SiteID: owner.ID}, paging.Request{Before: rest.Prev, Limit: 2})
	if err != nil {
		t.Fatalf("List before: %v", err)
	}
	if got := edgeIDs(back.Items); !reflect.DeepEqual(got, edgeIDs(edges[:2])) {
		t.Fatalf("backward page = %v", got)
	}

	relatedKind := graph.EdgeRelated
	related, err := repo.List(t.Context(), graph.EdgeQuery{SiteID: owner.ID, Kind: &relatedKind}, paging.Request{Limit: 10})
	if err != nil || len(related.Items) != 1 || related.Items[0].Kind != graph.EdgeRelated {
		t.Errorf("related filter = %+v, %v", related.Items, err)
	}
	proposed := graph.StatusProposed
	pending, err := repo.List(t.Context(), graph.EdgeQuery{SiteID: owner.ID, Status: &proposed}, paging.Request{Limit: 10})
	if err != nil || len(pending.Items) != 1 || pending.Items[0].ID != edges[1].ID {
		t.Errorf("status filter = %+v, %v", pending.Items, err)
	}
	touching, err := repo.List(t.Context(), graph.EdgeQuery{SiteID: owner.ID, EntityID: sandals.ID}, paging.Request{Limit: 10})
	if err != nil || len(touching.Items) != 2 {
		t.Errorf("entity filter must match either endpoint, got %d, %v", len(touching.Items), err)
	}
	newest, err := repo.List(t.Context(), graph.EdgeQuery{SiteID: owner.ID, Desc: true}, paging.Request{Limit: 1})
	if err != nil || len(newest.Items) != 1 || newest.Items[0].ID != edges[2].ID {
		t.Errorf("descending = %+v, %v", newest.Items, err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapters/sqlite/ -run TestEdgeRepo -v`
Expected: FAIL, `NewEdgeRepo` is undefined.

- [ ] **Step 3: Write the repository**

`internal/adapters/sqlite/edge_repo.go`:

```go
package sqlite

import (
	"context"
	"database/sql"

	"github.com/Masterminds/squirrel"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	edgeColumns       = `id, site_id, from_entity_id, to_entity_id, kind, weight, source, status, created_at`
	insertEdge        = `INSERT INTO edges (` + edgeColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	updateEdgeStatus  = `UPDATE edges SET status = ? WHERE id = ?`
	deleteEdge        = `DELETE FROM edges WHERE id = ?`
	selectEdge        = `SELECT ` + edgeColumns + ` FROM edges WHERE id = ?`
	selectEdgesBySite = `SELECT ` + edgeColumns + ` FROM edges WHERE site_id = ? ORDER BY created_at, id`
)

type EdgeRepo struct {
	store *Store
}

func NewEdgeRepo(store *Store) *EdgeRepo {
	return &EdgeRepo{store: store}
}

func edgeNotFound(id string) *errors.Error {
	return errors.New(errors.NotFound, "edge not found").WithDetail("edgeId", id)
}

func edgeConflict(e graph.Edge) *errors.Error {
	return errors.New(errors.Conflict, "an edge of this kind already links these entities").
		WithDetail("fromEntityId", e.FromEntityID).WithDetail("toEntityId", e.ToEntityID).WithDetail("kind", string(e.Kind))
}

func (r *EdgeRepo) Insert(ctx context.Context, e graph.Edge) error {
	_, err := execWrite(ctx, r.store.writeFrom(ctx), insertEdge, []any{
		e.ID, e.SiteID, e.FromEntityID, e.ToEntityID, string(e.Kind), e.Weight, string(e.Source), string(e.Status), formatTime(e.CreatedAt),
	}, edgeConflict(e), "insert the edge")
	return err
}

func (r *EdgeRepo) SetStatus(ctx context.Context, id string, status graph.EdgeStatus) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), updateEdgeStatus, []any{string(status), id}, nil, "update the edge status")
	return requireAffected(affected, err, edgeNotFound(id))
}

func (r *EdgeRepo) Delete(ctx context.Context, id string) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), deleteEdge, []any{id}, nil, "delete the edge")
	return requireAffected(affected, err, edgeNotFound(id))
}

func (r *EdgeRepo) Get(ctx context.Context, id string) (graph.Edge, error) {
	return selectOne(ctx, r.store.execFrom(ctx), selectEdge, []any{id}, scanEdge, edgeNotFound(id), "read the edge")
}

func (r *EdgeRepo) ListBySite(ctx context.Context, siteID string) ([]graph.Edge, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectEdgesBySite, []any{siteID}, scanEdge, "list the site edges")
}

func edgeKeyset(desc bool) paging.Keyset[graph.Edge] {
	return paging.Keyset[graph.Edge]{
		IDColumn: "id",
		ID:       func(e graph.Edge) string { return e.ID },
		Keys:     []paging.SortKey[graph.Edge]{paging.TimeKey[graph.Edge]("createdAt", "created_at", func(e graph.Edge) any { return e.CreatedAt })},
		Desc:     desc,
	}
}

func (r *EdgeRepo) List(ctx context.Context, q graph.EdgeQuery, page paging.Request) (paging.List[graph.Edge], error) {
	builder := squirrel.Select(edgeColumns).From("edges")
	if q.SiteID != "" {
		builder = builder.Where(squirrel.Eq{"site_id": q.SiteID})
	}
	if q.Kind != nil {
		builder = builder.Where(squirrel.Eq{"kind": string(*q.Kind)})
	}
	if q.Status != nil {
		builder = builder.Where(squirrel.Eq{"status": string(*q.Status)})
	}
	if q.EntityID != "" {
		builder = builder.Where(squirrel.Or{squirrel.Eq{"from_entity_id": q.EntityID}, squirrel.Eq{"to_entity_id": q.EntityID}})
	}

	keyset := edgeKeyset(q.Desc)
	keyed, err := keyset.Apply(builder, page)
	if err != nil {
		return paging.List[graph.Edge]{}, err
	}
	query, args, err := buildQuery(keyed, "edges")
	if err != nil {
		return paging.List[graph.Edge]{}, err
	}
	rows, err := selectAll(ctx, r.store.execFrom(ctx), query, args, scanEdge, "list the edges")
	if err != nil {
		return paging.List[graph.Edge]{}, err
	}
	return keyset.Cut(rows, page)
}

func scanEdge(rows *sql.Rows) (graph.Edge, error) {
	var (
		e                    graph.Edge
		kind, source, status string
		createdAt            string
	)
	if err := rows.Scan(&e.ID, &e.SiteID, &e.FromEntityID, &e.ToEntityID, &kind, &e.Weight, &source, &status, &createdAt); err != nil {
		return graph.Edge{}, err
	}
	e.Kind = graph.EdgeKind(kind)
	e.Source = graph.Source(source)
	e.Status = graph.EdgeStatus(status)

	var err error
	if e.CreatedAt, err = parseTime(createdAt); err != nil {
		return graph.Edge{}, err
	}
	return e, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race ./internal/adapters/sqlite/ -run TestEdgeRepo -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/adapters/sqlite/edge_repo.go internal/adapters/sqlite/edge_repo_test.go
git commit -m "feat(sqlite): edge repository with kind, status and endpoint filters

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---
### Task 15: `PageRepo` and `PageLinkRepo`

**Files:**
- Create: `internal/adapters/sqlite/page_repo.go`, `internal/adapters/sqlite/page_link_repo.go`
- Modify: `internal/adapters/sqlite/sqlitetest/fixtures.go`
- Test: `internal/adapters/sqlite/page_repo_test.go`, `internal/adapters/sqlite/page_link_repo_test.go`

**Interfaces:**
- Consumes: Task 12 helpers; `pagemap.Page|PageLink|Query|Sort|Status|WPType|LinkOrigin`.
- Produces: `PageRepo` with `NewPageRepo(store *Store) *PageRepo`, `Insert(ctx, pagemap.Page) error`, `Update(ctx, pagemap.Page) error`, `Delete(ctx, id string) error`, `Get(ctx, id string) (pagemap.Page, error)`, `List(ctx, q pagemap.Query, page paging.Request) (paging.List[pagemap.Page], error)`, `ListBySite(ctx, siteID string) ([]pagemap.Page, error)` (path order); `PageLinkRepo` with `NewPageLinkRepo(store *Store) *PageLinkRepo`, `ReplaceForPage(ctx, pageID string, links []pagemap.PageLink) error`, `ListForPage(ctx, pageID string) ([]pagemap.PageLink, error)`; `sqlitetest.Page(t testing.TB, store *sqlite.Store, siteID, path string) pagemap.Page`.

- [ ] **Step 1: Write the failing tests**

`internal/adapters/sqlite/page_repo_test.go`:

```go
package sqlite_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func fullPage(siteID, path string, at time.Time) pagemap.Page {
	wpID := int64(42)
	modified := at.Add(-time.Hour)
	return pagemap.Page{
		ID: id.New(), SiteID: siteID, Path: path, Slug: pagemap.Slug(path), WPType: pagemap.WPPost, WPID: &wpID,
		Title: "Title", H1: "Heading", MetaTitle: "Meta", MetaDescription: "Description", Canonical: "https://shop.example.com" + path,
		Status: pagemap.StatusExists, ContentHash: "abc", WPModifiedAt: &modified, LastSyncedAt: &at, Drift: true, CreatedAt: at, UpdatedAt: at,
	}
}

func pagePaths(pages []pagemap.Page) []string {
	out := make([]string, 0, len(pages))
	for _, p := range pages {
		out = append(out, p.Path)
	}
	return out
}

func TestPageRepoRoundTrip(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	entity := sqlitetest.Entity(t, store, owner.ID, "Shoes")
	repo := sqlite.NewPageRepo(store)

	parent := fullPage(owner.ID, "/shop/", sqlitetest.Stamp)
	if err := repo.Insert(t.Context(), parent); err != nil {
		t.Fatalf("Insert parent: %v", err)
	}
	want := fullPage(owner.ID, "/shop/shoes/", sqlitetest.Stamp)
	want.ParentPageID = &parent.ID
	want.EntityID = &entity.ID
	if err := repo.Insert(t.Context(), want); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	got, err := repo.Get(t.Context(), want.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Get = %+v\nwant %+v", got, want)
	}

	if err = repo.Insert(t.Context(), fullPage(owner.ID, "/shop/shoes/", sqlitetest.Stamp)); !errors.IsCode(err, errors.Conflict) {
		t.Errorf("duplicate path code = %q, want CONFLICT", errors.CodeOf(err))
	}
	if err = repo.Insert(t.Context(), fullPage("no-such-site", "/x/", sqlitetest.Stamp)); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("unknown site code = %q, want INVALID", errors.CodeOf(err))
	}
	stranger := fullPage(owner.ID, "/y/", sqlitetest.Stamp)
	missing := "no-such-entity"
	stranger.EntityID = &missing
	if err = repo.Insert(t.Context(), stranger); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("unknown entity code = %q, want INVALID", errors.CodeOf(err))
	}

	want.Title = "Renamed"
	want.Status = pagemap.StatusPublished
	want.WPID = nil
	want.LastSyncedAt = nil
	want.Drift = false
	want.UpdatedAt = sqlitetest.Stamp.Add(time.Minute)
	if err = repo.Update(t.Context(), want); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, err = repo.Get(t.Context(), want.ID)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Get after update = %+v\nwant %+v", got, want)
	}

	if err = repo.Delete(t.Context(), parent.ID); err != nil {
		t.Fatalf("Delete parent: %v", err)
	}
	got, err = repo.Get(t.Context(), want.ID)
	if err != nil || got.ParentPageID != nil {
		t.Errorf("child after parent delete = %+v, %v; want a NULL parent", got, err)
	}
	if err = repo.Delete(t.Context(), parent.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Delete twice code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if err = repo.Update(t.Context(), parent); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Update missing code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = repo.Get(t.Context(), parent.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Get missing code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}

func TestPageRepoListAndFilters(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	other := sqlitetest.Site(t, store, "blog")
	entity := sqlitetest.Entity(t, store, owner.ID, "Shoes")
	repo := sqlite.NewPageRepo(store)

	paths := []string{"/shop/", "/blog/", "/shop/shoes/", "/about/", "/shop/bags/"}
	for i, path := range paths {
		record := fullPage(owner.ID, path, sqlitetest.Stamp.Add(time.Duration(i)*time.Minute))
		if i%2 == 0 {
			record.Status = pagemap.StatusPlanned
		}
		if path == "/shop/shoes/" {
			record.EntityID = &entity.ID
		}
		if err := repo.Insert(t.Context(), record); err != nil {
			t.Fatalf("Insert %s: %v", path, err)
		}
	}
	if err := repo.Insert(t.Context(), fullPage(other.ID, "/shop/", sqlitetest.Stamp)); err != nil {
		t.Fatalf("Insert on the other site: %v", err)
	}

	bySite, err := repo.ListBySite(t.Context(), owner.ID)
	if err != nil {
		t.Fatalf("ListBySite: %v", err)
	}
	if got := pagePaths(bySite); !reflect.DeepEqual(got, []string{"/about/", "/blog/", "/shop/", "/shop/bags/", "/shop/shoes/"}) {
		t.Errorf("ListBySite = %v", got)
	}

	byPath := pagemap.Query{SiteID: owner.ID, Sort: pagemap.SortPath}
	first, err := repo.List(t.Context(), byPath, paging.Request{Limit: 2})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got := pagePaths(first.Items); !reflect.DeepEqual(got, []string{"/about/", "/blog/"}) || !first.HasMore {
		t.Fatalf("first page = %v", got)
	}
	second, err := repo.List(t.Context(), byPath, paging.Request{After: first.Next, Limit: 2})
	if err != nil {
		t.Fatalf("List after: %v", err)
	}
	if got := pagePaths(second.Items); !reflect.DeepEqual(got, []string{"/shop/", "/shop/bags/"}) {
		t.Fatalf("second page = %v", got)
	}
	back, err := repo.List(t.Context(), byPath, paging.Request{Before: second.Prev, Limit: 2})
	if err != nil {
		t.Fatalf("List before: %v", err)
	}
	if got := pagePaths(back.Items); !reflect.DeepEqual(got, []string{"/about/", "/blog/"}) {
		t.Fatalf("backward page = %v", got)
	}

	newest, err := repo.List(t.Context(), pagemap.Query{SiteID: owner.ID, Sort: pagemap.SortCreatedAt, Desc: true}, paging.Request{Limit: 1})
	if err != nil || len(newest.Items) != 1 || newest.Items[0].Path != "/shop/bags/" {
		t.Errorf("newest = %+v, %v", newest.Items, err)
	}

	planned := pagemap.StatusPlanned
	byStatus, err := repo.List(t.Context(), pagemap.Query{SiteID: owner.ID, Status: &planned, Sort: pagemap.SortPath}, paging.Request{Limit: 10})
	if err != nil {
		t.Fatalf("List by status: %v", err)
	}
	if got := pagePaths(byStatus.Items); !reflect.DeepEqual(got, []string{"/shop/", "/shop/bags/", "/shop/shoes/"}) {
		t.Errorf("planned = %v", got)
	}
	byEntity, err := repo.List(t.Context(), pagemap.Query{SiteID: owner.ID, EntityID: &entity.ID, Sort: pagemap.SortPath}, paging.Request{Limit: 10})
	if err != nil || len(byEntity.Items) != 1 || byEntity.Items[0].Path != "/shop/shoes/" {
		t.Errorf("by entity = %+v, %v", byEntity.Items, err)
	}
	unmapped, err := repo.List(t.Context(), pagemap.Query{SiteID: owner.ID, Unmapped: true, Sort: pagemap.SortPath}, paging.Request{Limit: 10})
	if err != nil || len(unmapped.Items) != 4 {
		t.Errorf("unmapped = %d, %v", len(unmapped.Items), err)
	}
	prefixed, err := repo.List(t.Context(), pagemap.Query{SiteID: owner.ID, PathPrefix: "/shop/", Sort: pagemap.SortPath}, paging.Request{Limit: 10})
	if err != nil {
		t.Fatalf("List by prefix: %v", err)
	}
	if got := pagePaths(prefixed.Items); !reflect.DeepEqual(got, []string{"/shop/", "/shop/bags/", "/shop/shoes/"}) {
		t.Errorf("prefixed = %v", got)
	}
	wild, err := repo.List(t.Context(), pagemap.Query{SiteID: owner.ID, PathPrefix: "/sh_p/", Sort: pagemap.SortPath}, paging.Request{Limit: 10})
	if err != nil || len(wild.Items) != 0 {
		t.Errorf("an underscore must be escaped, got %d, %v", len(wild.Items), err)
	}
}

func TestPageFixture(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	fixture := sqlitetest.Page(t, store, owner.ID, "/shop/")
	got, err := sqlite.NewPageRepo(store).Get(t.Context(), fixture.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !reflect.DeepEqual(got, fixture) {
		t.Errorf("fixture round trip = %+v\nwant %+v", got, fixture)
	}
}
```

`internal/adapters/sqlite/page_link_repo_test.go`:

```go
package sqlite_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func TestPageLinkRepoReplaceForPage(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	from := sqlitetest.Page(t, store, owner.ID, "/shop/")
	to := sqlitetest.Page(t, store, owner.ID, "/shop/shoes/")
	repo := sqlite.NewPageLinkRepo(store)

	empty, err := repo.ListForPage(t.Context(), from.ID)
	if err != nil || len(empty) != 0 {
		t.Fatalf("ListForPage on a fresh page = %v, %v", empty, err)
	}

	links := []pagemap.PageLink{
		{ID: id.New(), SiteID: owner.ID, FromPageID: from.ID, ToPageID: &to.ID, ToURL: "/shop/shoes/", AnchorText: "shoes", Origin: pagemap.OriginGenerated, ObservedAt: sqlitetest.Stamp},
		{ID: id.New(), SiteID: owner.ID, FromPageID: from.ID, ToURL: "https://elsewhere.example.com/", AnchorText: "elsewhere", Origin: pagemap.OriginObserved, ObservedAt: sqlitetest.Stamp.Add(time.Second)},
	}
	if err = repo.ReplaceForPage(t.Context(), from.ID, links); err != nil {
		t.Fatalf("ReplaceForPage: %v", err)
	}
	got, err := repo.ListForPage(t.Context(), from.ID)
	if err != nil {
		t.Fatalf("ListForPage: %v", err)
	}
	if !reflect.DeepEqual(got, links) {
		t.Errorf("ListForPage = %+v\nwant %+v", got, links)
	}

	replacement := []pagemap.PageLink{links[1]}
	if err = repo.ReplaceForPage(t.Context(), from.ID, replacement); err != nil {
		t.Fatalf("ReplaceForPage again: %v", err)
	}
	got, err = repo.ListForPage(t.Context(), from.ID)
	if err != nil || !reflect.DeepEqual(got, replacement) {
		t.Errorf("after replacement = %+v, %v", got, err)
	}

	if err = repo.ReplaceForPage(t.Context(), from.ID, nil); err != nil {
		t.Fatalf("ReplaceForPage with nothing: %v", err)
	}
	got, err = repo.ListForPage(t.Context(), from.ID)
	if err != nil || len(got) != 0 {
		t.Errorf("after clearing = %+v, %v", got, err)
	}

	foreign := links[0]
	foreign.FromPageID = to.ID
	if err = repo.ReplaceForPage(t.Context(), from.ID, []pagemap.PageLink{foreign}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("a link from another page code = %q, want INVALID", errors.CodeOf(err))
	}
	dangling := links[0]
	unknown := "no-such-page"
	dangling.ToPageID = &unknown
	if err = repo.ReplaceForPage(t.Context(), from.ID, []pagemap.PageLink{dangling}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("a link to an unknown page code = %q, want INVALID", errors.CodeOf(err))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/adapters/sqlite/ -run 'TestPageRepo|TestPageFixture|TestPageLinkRepo' -v`
Expected: FAIL, `NewPageRepo`, `NewPageLinkRepo` and `sqlitetest.Page` are undefined.

- [ ] **Step 3: Write the repositories and the fixture**

`internal/adapters/sqlite/page_repo.go`:

```go
package sqlite

import (
	"context"
	"database/sql"

	"github.com/Masterminds/squirrel"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	pageColumns       = `id, site_id, path, slug, parent_page_id, wp_type, wp_id, title, h1, meta_title, meta_description, canonical, status, entity_id, template_id, content_hash, wp_modified_at, last_synced_at, drift, created_at, updated_at`
	insertPage        = `INSERT INTO pages (` + pageColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	updatePage        = `UPDATE pages SET path = ?, slug = ?, parent_page_id = ?, wp_type = ?, wp_id = ?, title = ?, h1 = ?, meta_title = ?, meta_description = ?, canonical = ?, status = ?, entity_id = ?, template_id = ?, content_hash = ?, wp_modified_at = ?, last_synced_at = ?, drift = ?, updated_at = ? WHERE id = ?`
	deletePage        = `DELETE FROM pages WHERE id = ?`
	selectPage        = `SELECT ` + pageColumns + ` FROM pages WHERE id = ?`
	selectPagesBySite = `SELECT ` + pageColumns + ` FROM pages WHERE site_id = ? ORDER BY path, id`
)

type PageRepo struct {
	store *Store
}

func NewPageRepo(store *Store) *PageRepo {
	return &PageRepo{store: store}
}

func pageNotFound(id string) *errors.Error {
	return errors.New(errors.NotFound, "page not found").WithDetail("pageId", id)
}

func pageConflict(path string) *errors.Error {
	return errors.New(errors.Conflict, "a page with this path already exists in the site").WithDetail("path", path)
}

func (r *PageRepo) Insert(ctx context.Context, p pagemap.Page) error {
	_, err := execWrite(ctx, r.store.writeFrom(ctx), insertPage, []any{
		p.ID, p.SiteID, p.Path, p.Slug, nullString(p.ParentPageID), string(p.WPType), nullInt(p.WPID),
		p.Title, p.H1, p.MetaTitle, p.MetaDescription, p.Canonical, string(p.Status), nullString(p.EntityID), nullString(p.TemplateID),
		p.ContentHash, nullTime(p.WPModifiedAt), nullTime(p.LastSyncedAt), boolInt(p.Drift), formatTime(p.CreatedAt), formatTime(p.UpdatedAt),
	}, pageConflict(p.Path), "insert the page")
	return err
}

func (r *PageRepo) Update(ctx context.Context, p pagemap.Page) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), updatePage, []any{
		p.Path, p.Slug, nullString(p.ParentPageID), string(p.WPType), nullInt(p.WPID),
		p.Title, p.H1, p.MetaTitle, p.MetaDescription, p.Canonical, string(p.Status), nullString(p.EntityID), nullString(p.TemplateID),
		p.ContentHash, nullTime(p.WPModifiedAt), nullTime(p.LastSyncedAt), boolInt(p.Drift), formatTime(p.UpdatedAt), p.ID,
	}, pageConflict(p.Path), "update the page")
	return requireAffected(affected, err, pageNotFound(p.ID))
}

func (r *PageRepo) Delete(ctx context.Context, id string) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), deletePage, []any{id}, nil, "delete the page")
	return requireAffected(affected, err, pageNotFound(id))
}

func (r *PageRepo) Get(ctx context.Context, id string) (pagemap.Page, error) {
	return selectOne(ctx, r.store.execFrom(ctx), selectPage, []any{id}, scanPage, pageNotFound(id), "read the page")
}

func (r *PageRepo) ListBySite(ctx context.Context, siteID string) ([]pagemap.Page, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectPagesBySite, []any{siteID}, scanPage, "list the site pages")
}

func pageKeyset(q pagemap.Query) paging.Keyset[pagemap.Page] {
	key := paging.TimeKey[pagemap.Page]("createdAt", "created_at", func(p pagemap.Page) any { return p.CreatedAt })
	if q.Sort == pagemap.SortPath {
		key = paging.TextKey[pagemap.Page]("path", "path", func(p pagemap.Page) any { return p.Path })
	}
	return paging.Keyset[pagemap.Page]{
		IDColumn: "id",
		ID:       func(p pagemap.Page) string { return p.ID },
		Keys:     []paging.SortKey[pagemap.Page]{key},
		Desc:     q.Desc,
	}
}

func (r *PageRepo) List(ctx context.Context, q pagemap.Query, page paging.Request) (paging.List[pagemap.Page], error) {
	builder := squirrel.Select(pageColumns).From("pages")
	if q.SiteID != "" {
		builder = builder.Where(squirrel.Eq{"site_id": q.SiteID})
	}
	if q.Status != nil {
		builder = builder.Where(squirrel.Eq{"status": string(*q.Status)})
	}
	if q.EntityID != nil {
		builder = builder.Where(squirrel.Eq{"entity_id": *q.EntityID})
	}
	if q.Unmapped {
		builder = builder.Where("entity_id IS NULL")
	}
	if q.PathPrefix != "" {
		builder = builder.Where(`path LIKE ? ESCAPE '\'`, escapeLike(q.PathPrefix)+"%")
	}

	keyset := pageKeyset(q)
	keyed, err := keyset.Apply(builder, page)
	if err != nil {
		return paging.List[pagemap.Page]{}, err
	}
	query, args, err := buildQuery(keyed, "pages")
	if err != nil {
		return paging.List[pagemap.Page]{}, err
	}
	rows, err := selectAll(ctx, r.store.execFrom(ctx), query, args, scanPage, "list the pages")
	if err != nil {
		return paging.List[pagemap.Page]{}, err
	}
	return keyset.Cut(rows, page)
}

func scanPage(rows *sql.Rows) (pagemap.Page, error) {
	var (
		p                                     pagemap.Page
		parentID, entityID, templateID        sql.NullString
		wpType, status                        string
		wpID                                  sql.NullInt64
		wpModifiedAt, lastSyncedAt            sql.NullString
		drift                                 int64
		createdAt, updatedAt                  string
	)
	if err := rows.Scan(&p.ID, &p.SiteID, &p.Path, &p.Slug, &parentID, &wpType, &wpID, &p.Title, &p.H1, &p.MetaTitle, &p.MetaDescription, &p.Canonical,
		&status, &entityID, &templateID, &p.ContentHash, &wpModifiedAt, &lastSyncedAt, &drift, &createdAt, &updatedAt); err != nil {
		return pagemap.Page{}, err
	}
	p.ParentPageID = optString(parentID)
	p.WPType = pagemap.WPType(wpType)
	p.WPID = optInt(wpID)
	p.Status = pagemap.Status(status)
	p.EntityID = optString(entityID)
	p.TemplateID = optString(templateID)
	p.Drift = drift == 1

	var err error
	if p.WPModifiedAt, err = parseNullTime(wpModifiedAt); err != nil {
		return pagemap.Page{}, err
	}
	if p.LastSyncedAt, err = parseNullTime(lastSyncedAt); err != nil {
		return pagemap.Page{}, err
	}
	if p.CreatedAt, err = parseTime(createdAt); err != nil {
		return pagemap.Page{}, err
	}
	if p.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return pagemap.Page{}, err
	}
	return p, nil
}
```

`internal/adapters/sqlite/page_link_repo.go`:

```go
package sqlite

import (
	"context"
	"database/sql"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	linkColumns        = `id, site_id, from_page_id, to_page_id, to_url, anchor_text, origin, observed_at`
	insertLink         = `INSERT INTO page_links (` + linkColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	deleteLinksForPage = `DELETE FROM page_links WHERE from_page_id = ?`
	selectLinksForPage = `SELECT ` + linkColumns + ` FROM page_links WHERE from_page_id = ? ORDER BY observed_at, id`
)

type PageLinkRepo struct {
	store *Store
}

func NewPageLinkRepo(store *Store) *PageLinkRepo {
	return &PageLinkRepo{store: store}
}

func (r *PageLinkRepo) ReplaceForPage(ctx context.Context, pageID string, links []pagemap.PageLink) error {
	for _, link := range links {
		if link.FromPageID != pageID {
			return errors.New(errors.Invalid, "every link must start from the page being replaced").
				WithDetail("pageId", pageID).WithDetail("linkId", link.ID)
		}
	}
	if _, err := execWrite(ctx, r.store.writeFrom(ctx), deleteLinksForPage, []any{pageID}, nil, "clear the page links"); err != nil {
		return err
	}
	for _, link := range links {
		if _, err := execWrite(ctx, r.store.writeFrom(ctx), insertLink, []any{
			link.ID, link.SiteID, link.FromPageID, nullString(link.ToPageID), link.ToURL, link.AnchorText, string(link.Origin), formatTime(link.ObservedAt),
		}, nil, "insert a page link"); err != nil {
			return err
		}
	}
	return nil
}

func (r *PageLinkRepo) ListForPage(ctx context.Context, pageID string) ([]pagemap.PageLink, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectLinksForPage, []any{pageID}, scanPageLink, "list the page links")
}

func scanPageLink(rows *sql.Rows) (pagemap.PageLink, error) {
	var (
		l          pagemap.PageLink
		toPageID   sql.NullString
		origin     string
		observedAt string
	)
	if err := rows.Scan(&l.ID, &l.SiteID, &l.FromPageID, &toPageID, &l.ToURL, &l.AnchorText, &origin, &observedAt); err != nil {
		return pagemap.PageLink{}, err
	}
	l.ToPageID = optString(toPageID)
	l.Origin = pagemap.LinkOrigin(origin)

	var err error
	if l.ObservedAt, err = parseTime(observedAt); err != nil {
		return pagemap.PageLink{}, err
	}
	return l, nil
}
```

`ReplaceForPage` is called inside the use case's `Store.Do`; outside a transaction a failing insert leaves the earlier links removed, which is why the use-case tests in Task 21 exercise it through the unit of work.

Append to `internal/adapters/sqlite/sqlitetest/fixtures.go` (add `"github.com/davidmovas/postulator/internal/domain/pagemap"` to its imports):

```go
func Page(t testing.TB, store *sqlite.Store, siteID, path string) pagemap.Page {
	t.Helper()

	record := pagemap.Page{
		ID: id.New(), SiteID: siteID, Path: path, Slug: pagemap.Slug(path), WPType: pagemap.WPPage,
		Title: path, H1: path, Status: pagemap.StatusPlanned, CreatedAt: Stamp, UpdatedAt: Stamp,
	}
	if err := sqlite.NewPageRepo(store).Insert(t.Context(), record); err != nil {
		t.Fatalf("insert the page fixture: %v", err)
	}
	return record
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -race ./internal/adapters/sqlite/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/adapters/sqlite/page_repo.go internal/adapters/sqlite/page_repo_test.go internal/adapters/sqlite/page_link_repo.go internal/adapters/sqlite/page_link_repo_test.go internal/adapters/sqlite/sqlitetest/fixtures.go
git commit -m "feat(sqlite): page and page link repositories

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 16: `TemplateRepo` with overrides

**Files:**
- Create: `internal/adapters/sqlite/template_repo.go`
- Modify: `internal/adapters/sqlite/sqlitetest/fixtures.go`
- Test: `internal/adapters/sqlite/template_repo_test.go`

**Interfaces:**
- Consumes: Task 12 helpers; `template.Template|TemplateSpec|Override|OverrideScope|Query|Sort|Scope`, `template.Seed`.
- Produces: `TemplateRepo` with `NewTemplateRepo(store *Store) *TemplateRepo`, `Insert(ctx, template.Template) error`, `Update(ctx, template.Template) error` (name, page kind, version, spec, updated_at; scope and site are immutable), `Delete(ctx, id string) error`, `Get(ctx, id string) (template.Template, error)`, `List(ctx, q template.Query, page paging.Request) (paging.List[template.Template], error)`, `UpsertOverride(ctx, o template.Override) (template.Override, error)` (returns the stored row: an existing target keeps its id and `CreatedAt`), `GetOverride(ctx, templateID string, scope template.OverrideScope, targetID string) (template.Override, error)`, `DeleteOverride(ctx, id string) error`, `ListOverrides(ctx, templateID string) ([]template.Override, error)`; `sqlitetest.Template(t testing.TB, store *sqlite.Store, name string) template.Template` (global, Hub seed spec).

- [ ] **Step 1: Write the failing test**

```go
package sqlite_test

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func globalTemplate(name string, at time.Time) template.Template {
	seed := template.Seed()[3]
	return template.Template{ID: id.New(), Scope: template.ScopeGlobal, Name: name, PageKind: seed.PageKind, Version: 1, Spec: seed.Spec, CreatedAt: at, UpdatedAt: at}
}

func templateNames(list []template.Template) []string {
	out := make([]string, 0, len(list))
	for _, item := range list {
		out = append(out, item.Name)
	}
	return out
}

func TestTemplateRepoRoundTrip(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	repo := sqlite.NewTemplateRepo(store)
	want := globalTemplate("Hub", sqlitetest.Stamp)

	if err := repo.Insert(t.Context(), want); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	got, err := repo.Get(t.Context(), want.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Get = %+v\nwant %+v", got, want)
	}
	if err = repo.Insert(t.Context(), globalTemplate("hub", sqlitetest.Stamp)); !errors.IsCode(err, errors.Conflict) {
		t.Errorf("duplicate global name code = %q, want CONFLICT", errors.CodeOf(err))
	}

	owner := sqlitetest.Site(t, store, "shop")
	scoped := globalTemplate("Hub", sqlitetest.Stamp)
	scoped.Scope = template.ScopeSite
	scoped.SiteID = &owner.ID
	if err = repo.Insert(t.Context(), scoped); err != nil {
		t.Fatalf("a site may reuse a global name: %v", err)
	}
	if err = repo.Insert(t.Context(), scoped); !errors.IsCode(err, errors.Conflict) {
		t.Errorf("duplicate site name code = %q, want CONFLICT", errors.CodeOf(err))
	}

	want.Name = "Hub Page"
	want.Version = 2
	want.Spec.Tone = "warm"
	want.UpdatedAt = sqlitetest.Stamp.Add(time.Minute)
	if err = repo.Update(t.Context(), want); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, err = repo.Get(t.Context(), want.ID)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Get after update = %+v, %v\nwant %+v", got, err, want)
	}

	if err = repo.Delete(t.Context(), want.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err = repo.Delete(t.Context(), want.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Delete twice code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if err = repo.Update(t.Context(), want); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Update missing code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = repo.Get(t.Context(), want.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Get missing code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}

func TestTemplateRepoListAndFilters(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	repo := sqlite.NewTemplateRepo(store)

	names := []string{"Category", "Guide", "Hub"}
	for i, name := range names {
		record := globalTemplate(name, sqlitetest.Stamp.Add(time.Duration(i)*time.Minute))
		if name == "Guide" {
			record.PageKind = "guide"
		}
		if err := repo.Insert(t.Context(), record); err != nil {
			t.Fatalf("Insert %s: %v", name, err)
		}
	}
	scoped := globalTemplate("Landing", sqlitetest.Stamp.Add(time.Hour))
	scoped.Scope = template.ScopeSite
	scoped.SiteID = &owner.ID
	if err := repo.Insert(t.Context(), scoped); err != nil {
		t.Fatalf("Insert scoped: %v", err)
	}

	global := template.ScopeGlobal
	first, err := repo.List(t.Context(), template.Query{Scope: &global, Sort: template.SortName}, paging.Request{Limit: 2})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got := templateNames(first.Items); !reflect.DeepEqual(got, []string{"Category", "Guide"}) || !first.HasMore {
		t.Fatalf("first page = %v", got)
	}
	rest, err := repo.List(t.Context(), template.Query{Scope: &global, Sort: template.SortName}, paging.Request{After: first.Next, Limit: 2})
	if err != nil {
		t.Fatalf("List after: %v", err)
	}
	if got := templateNames(rest.Items); !reflect.DeepEqual(got, []string{"Hub"}) || rest.HasMore {
		t.Fatalf("second page = %v", got)
	}
	back, err := repo.List(t.Context(), template.Query{Scope: &global, Sort: template.SortName}, paging.Request{Before: rest.Prev, Limit: 2})
	if err != nil {
		t.Fatalf("List before: %v", err)
	}
	if got := templateNames(back.Items); !reflect.DeepEqual(got, []string{"Category", "Guide"}) {
		t.Fatalf("backward page = %v", got)
	}

	bySite, err := repo.List(t.Context(), template.Query{SiteID: &owner.ID, Sort: template.SortCreatedAt}, paging.Request{Limit: 10})
	if err != nil || len(bySite.Items) != 1 || bySite.Items[0].Name != "Landing" {
		t.Errorf("by site = %+v, %v", bySite.Items, err)
	}
	byKind, err := repo.List(t.Context(), template.Query{PageKind: "guide", Sort: template.SortCreatedAt}, paging.Request{Limit: 10})
	if err != nil || len(byKind.Items) != 1 || byKind.Items[0].Name != "Guide" {
		t.Errorf("by kind = %+v, %v", byKind.Items, err)
	}
	byName, err := repo.List(t.Context(), template.Query{Scope: &global, Name: "hub", Sort: template.SortCreatedAt}, paging.Request{Limit: 10})
	if err != nil || len(byName.Items) != 1 || byName.Items[0].Name != "Hub" {
		t.Errorf("by name must be case-insensitive, got %+v, %v", byName.Items, err)
	}
	newest, err := repo.List(t.Context(), template.Query{Sort: template.SortCreatedAt, Desc: true}, paging.Request{Limit: 1})
	if err != nil || len(newest.Items) != 1 || newest.Items[0].Name != "Landing" {
		t.Errorf("newest = %+v, %v", newest.Items, err)
	}
}

func TestTemplateRepoOverrides(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	page := sqlitetest.Page(t, store, owner.ID, "/shop/")
	base := sqlitetest.Template(t, store, "Hub")
	repo := sqlite.NewTemplateRepo(store)

	if _, err := repo.GetOverride(t.Context(), base.ID, template.OverrideSite, owner.ID); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("GetOverride before upsert code = %q, want NOT_FOUND", errors.CodeOf(err))
	}

	siteOverride := template.Override{ID: id.New(), TemplateID: base.ID, Scope: template.OverrideSite, TargetID: owner.ID, Patch: json.RawMessage(`{"tone":"warm"}`), CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp}
	stored, err := repo.UpsertOverride(t.Context(), siteOverride)
	if err != nil {
		t.Fatalf("UpsertOverride: %v", err)
	}
	if !reflect.DeepEqual(stored, siteOverride) {
		t.Errorf("stored = %+v\nwant %+v", stored, siteOverride)
	}

	replacement := siteOverride
	replacement.ID = id.New()
	replacement.Patch = json.RawMessage(`{"tone":"cold"}`)
	replacement.UpdatedAt = sqlitetest.Stamp.Add(time.Minute)
	stored, err = repo.UpsertOverride(t.Context(), replacement)
	if err != nil {
		t.Fatalf("UpsertOverride again: %v", err)
	}
	if stored.ID != siteOverride.ID || string(stored.Patch) != `{"tone":"cold"}` || !stored.UpdatedAt.Equal(replacement.UpdatedAt) || !stored.CreatedAt.Equal(siteOverride.CreatedAt) {
		t.Errorf("an existing target keeps its id and creation time: %+v", stored)
	}

	pageOverride := template.Override{ID: id.New(), TemplateID: base.ID, Scope: template.OverridePage, TargetID: page.ID, Patch: json.RawMessage(`{"length":{"min":100}}`), CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp}
	if _, err = repo.UpsertOverride(t.Context(), pageOverride); err != nil {
		t.Fatalf("UpsertOverride page: %v", err)
	}
	stranger := pageOverride
	stranger.ID = id.New()
	stranger.TargetID = "no-such-page"
	if _, err = repo.UpsertOverride(t.Context(), stranger); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("unknown page target code = %q, want INVALID", errors.CodeOf(err))
	}

	all, err := repo.ListOverrides(t.Context(), base.ID)
	if err != nil || len(all) != 2 {
		t.Fatalf("ListOverrides = %+v, %v", all, err)
	}
	got, err := repo.GetOverride(t.Context(), base.ID, template.OverridePage, page.ID)
	if err != nil || string(got.Patch) != `{"length":{"min":100}}` {
		t.Errorf("GetOverride page = %+v, %v", got, err)
	}

	if err = repo.DeleteOverride(t.Context(), got.ID); err != nil {
		t.Fatalf("DeleteOverride: %v", err)
	}
	if err = repo.DeleteOverride(t.Context(), got.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("DeleteOverride twice code = %q, want NOT_FOUND", errors.CodeOf(err))
	}

	if err = repo.Delete(t.Context(), base.ID); err != nil {
		t.Fatalf("Delete template: %v", err)
	}
	if _, err = repo.GetOverride(t.Context(), base.ID, template.OverrideSite, owner.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("overrides must cascade with their template, got %v", err)
	}
}

func TestTemplateFixture(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	fixture := sqlitetest.Template(t, store, "Hub")
	got, err := sqlite.NewTemplateRepo(store).Get(t.Context(), fixture.ID)
	if err != nil || !reflect.DeepEqual(got, fixture) {
		t.Errorf("fixture round trip = %+v, %v\nwant %+v", got, err, fixture)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapters/sqlite/ -run 'TestTemplateRepo|TestTemplateFixture' -v`
Expected: FAIL, `NewTemplateRepo` and `sqlitetest.Template` are undefined.

- [ ] **Step 3: Write the repository and the fixture**

`internal/adapters/sqlite/template_repo.go`:

```go
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/Masterminds/squirrel"

	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	templateColumns           = `id, scope, site_id, name, page_kind, version, spec, created_at, updated_at`
	insertTemplate            = `INSERT INTO templates (` + templateColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	updateTemplate            = `UPDATE templates SET name = ?, page_kind = ?, version = ?, spec = ?, updated_at = ? WHERE id = ?`
	deleteTemplate            = `DELETE FROM templates WHERE id = ?`
	selectTemplate            = `SELECT ` + templateColumns + ` FROM templates WHERE id = ?`
	overrideColumns           = `id, template_id, scope, site_id, page_id, patch, created_at, updated_at`
	insertOverride            = `INSERT INTO template_overrides (` + overrideColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	updateOverride            = `UPDATE template_overrides SET patch = ?, updated_at = ? WHERE id = ?`
	deleteOverride            = `DELETE FROM template_overrides WHERE id = ?`
	selectOverride            = `SELECT ` + overrideColumns + ` FROM template_overrides WHERE template_id = ? AND scope = ? AND coalesce(site_id, page_id) = ?`
	selectOverridesByTemplate = `SELECT ` + overrideColumns + ` FROM template_overrides WHERE template_id = ? ORDER BY scope, coalesce(site_id, page_id)`
)

type TemplateRepo struct {
	store *Store
}

func NewTemplateRepo(store *Store) *TemplateRepo {
	return &TemplateRepo{store: store}
}

func templateNotFound(id string) *errors.Error {
	return errors.New(errors.NotFound, "template not found").WithDetail("templateId", id)
}

func templateConflict(name string) *errors.Error {
	return errors.New(errors.Conflict, "a template with this name already exists in this scope").WithDetail("name", name)
}

func overrideNotFound(templateID string, scope template.OverrideScope, targetID string) *errors.Error {
	return errors.New(errors.NotFound, "template override not found").
		WithDetail("templateId", templateID).WithDetail("scope", string(scope)).WithDetail("targetId", targetID)
}

func (r *TemplateRepo) Insert(ctx context.Context, t template.Template) error {
	spec, err := encodeJSON(t.Spec)
	if err != nil {
		return err
	}
	_, err = execWrite(ctx, r.store.writeFrom(ctx), insertTemplate, []any{
		t.ID, string(t.Scope), nullString(t.SiteID), t.Name, t.PageKind, t.Version, spec, formatTime(t.CreatedAt), formatTime(t.UpdatedAt),
	}, templateConflict(t.Name), "insert the template")
	return err
}

func (r *TemplateRepo) Update(ctx context.Context, t template.Template) error {
	spec, err := encodeJSON(t.Spec)
	if err != nil {
		return err
	}
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), updateTemplate, []any{
		t.Name, t.PageKind, t.Version, spec, formatTime(t.UpdatedAt), t.ID,
	}, templateConflict(t.Name), "update the template")
	return requireAffected(affected, err, templateNotFound(t.ID))
}

func (r *TemplateRepo) Delete(ctx context.Context, id string) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), deleteTemplate, []any{id}, nil, "delete the template")
	return requireAffected(affected, err, templateNotFound(id))
}

func (r *TemplateRepo) Get(ctx context.Context, id string) (template.Template, error) {
	return selectOne(ctx, r.store.execFrom(ctx), selectTemplate, []any{id}, scanTemplate, templateNotFound(id), "read the template")
}

func templateKeyset(q template.Query) paging.Keyset[template.Template] {
	key := paging.TimeKey[template.Template]("createdAt", "created_at", func(t template.Template) any { return t.CreatedAt })
	if q.Sort == template.SortName {
		key = paging.TextKey[template.Template]("name", "name", func(t template.Template) any { return t.Name })
	}
	return paging.Keyset[template.Template]{
		IDColumn: "id",
		ID:       func(t template.Template) string { return t.ID },
		Keys:     []paging.SortKey[template.Template]{key},
		Desc:     q.Desc,
	}
}

func (r *TemplateRepo) List(ctx context.Context, q template.Query, page paging.Request) (paging.List[template.Template], error) {
	builder := squirrel.Select(templateColumns).From("templates")
	if q.Scope != nil {
		builder = builder.Where(squirrel.Eq{"scope": string(*q.Scope)})
	}
	if q.SiteID != nil {
		builder = builder.Where(squirrel.Eq{"site_id": *q.SiteID})
	}
	if q.PageKind != "" {
		builder = builder.Where(squirrel.Eq{"page_kind": q.PageKind})
	}
	if q.Name != "" {
		builder = builder.Where(squirrel.Eq{"name": q.Name})
	}

	keyset := templateKeyset(q)
	keyed, err := keyset.Apply(builder, page)
	if err != nil {
		return paging.List[template.Template]{}, err
	}
	query, args, err := buildQuery(keyed, "templates")
	if err != nil {
		return paging.List[template.Template]{}, err
	}
	rows, err := selectAll(ctx, r.store.execFrom(ctx), query, args, scanTemplate, "list the templates")
	if err != nil {
		return paging.List[template.Template]{}, err
	}
	return keyset.Cut(rows, page)
}

func overrideTarget(o template.Override) (siteID, pageID any) {
	if o.Scope == template.OverrideSite {
		return o.TargetID, nil
	}
	return nil, o.TargetID
}

func (r *TemplateRepo) UpsertOverride(ctx context.Context, o template.Override) (template.Override, error) {
	existing, err := r.GetOverride(ctx, o.TemplateID, o.Scope, o.TargetID)
	switch {
	case err == nil:
		if _, err = execWrite(ctx, r.store.writeFrom(ctx), updateOverride, []any{string(o.Patch), formatTime(o.UpdatedAt), existing.ID}, nil, "update the template override"); err != nil {
			return template.Override{}, err
		}
		existing.Patch = o.Patch
		existing.UpdatedAt = o.UpdatedAt
		return existing, nil
	case errors.IsCode(err, errors.NotFound):
		siteID, pageID := overrideTarget(o)
		if _, err = execWrite(ctx, r.store.writeFrom(ctx), insertOverride, []any{
			o.ID, o.TemplateID, string(o.Scope), siteID, pageID, string(o.Patch), formatTime(o.CreatedAt), formatTime(o.UpdatedAt),
		}, nil, "insert the template override"); err != nil {
			return template.Override{}, err
		}
		return o, nil
	default:
		return template.Override{}, err
	}
}

func (r *TemplateRepo) GetOverride(ctx context.Context, templateID string, scope template.OverrideScope, targetID string) (template.Override, error) {
	return selectOne(ctx, r.store.execFrom(ctx), selectOverride, []any{templateID, string(scope), targetID}, scanOverride, overrideNotFound(templateID, scope, targetID), "read the template override")
}

func (r *TemplateRepo) DeleteOverride(ctx context.Context, id string) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), deleteOverride, []any{id}, nil, "delete the template override")
	return requireAffected(affected, err, errors.New(errors.NotFound, "template override not found").WithDetail("overrideId", id))
}

func (r *TemplateRepo) ListOverrides(ctx context.Context, templateID string) ([]template.Override, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectOverridesByTemplate, []any{templateID}, scanOverride, "list the template overrides")
}

func scanTemplate(rows *sql.Rows) (template.Template, error) {
	var (
		t                    template.Template
		scope, spec          string
		siteID               sql.NullString
		createdAt, updatedAt string
	)
	if err := rows.Scan(&t.ID, &scope, &siteID, &t.Name, &t.PageKind, &t.Version, &spec, &createdAt, &updatedAt); err != nil {
		return template.Template{}, err
	}
	t.Scope = template.Scope(scope)
	t.SiteID = optString(siteID)
	if err := decodeJSON(spec, &t.Spec, "decode the template spec"); err != nil {
		return template.Template{}, err
	}

	var err error
	if t.CreatedAt, err = parseTime(createdAt); err != nil {
		return template.Template{}, err
	}
	if t.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return template.Template{}, err
	}
	return t, nil
}

func scanOverride(rows *sql.Rows) (template.Override, error) {
	var (
		o                    template.Override
		scope, patch         string
		siteID, pageID       sql.NullString
		createdAt, updatedAt string
	)
	if err := rows.Scan(&o.ID, &o.TemplateID, &scope, &siteID, &pageID, &patch, &createdAt, &updatedAt); err != nil {
		return template.Override{}, err
	}
	o.Scope = template.OverrideScope(scope)
	if siteID.Valid {
		o.TargetID = siteID.String
	} else {
		o.TargetID = pageID.String
	}
	o.Patch = json.RawMessage(patch)

	var err error
	if o.CreatedAt, err = parseTime(createdAt); err != nil {
		return template.Override{}, err
	}
	if o.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return template.Override{}, err
	}
	return o, nil
}
```

Append to `internal/adapters/sqlite/sqlitetest/fixtures.go` (add `"github.com/davidmovas/postulator/internal/domain/template"` to its imports):

```go
func Template(t testing.TB, store *sqlite.Store, name string) template.Template {
	t.Helper()

	seed := template.Seed()[3]
	record := template.Template{ID: id.New(), Scope: template.ScopeGlobal, Name: name, PageKind: seed.PageKind, Version: 1, Spec: seed.Spec, CreatedAt: Stamp, UpdatedAt: Stamp}
	if err := sqlite.NewTemplateRepo(store).Insert(t.Context(), record); err != nil {
		t.Fatalf("insert the template fixture: %v", err)
	}
	return record
}
```

`template.Seed()[3]` is Hub (file-name order: category, comparison, guide, hub, product). The round-trip `DeepEqual` holds because the seed decodes `keywordRules.include` as `[]string{}` and `modelProfiles` as an empty map, both of which survive the JSON column unchanged, and `StepSpec.Params` is nil on both sides.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race ./internal/adapters/sqlite/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/adapters/sqlite/template_repo.go internal/adapters/sqlite/template_repo_test.go internal/adapters/sqlite/sqlitetest/fixtures.go
git commit -m "feat(sqlite): template repository with site and page overrides

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 17: `LinkPolicyRepo`

**Files:**
- Create: `internal/adapters/sqlite/link_policy_repo.go`
- Test: `internal/adapters/sqlite/link_policy_repo_test.go`

**Interfaces:**
- Consumes: Task 12 helpers; `template.LinkPolicy|LinkRules|PolicyQuery|Sort|Scope|AnchorStrategy`.
- Produces: `LinkPolicyRepo` with `NewLinkPolicyRepo(store *Store) *LinkPolicyRepo`, `Insert(ctx, template.LinkPolicy) error`, `Update(ctx, template.LinkPolicy) error`, `Delete(ctx, id string) error`, `Get(ctx, id string) (template.LinkPolicy, error)`, `List(ctx, q template.PolicyQuery, page paging.Request) (paging.List[template.LinkPolicy], error)`.

- [ ] **Step 1: Write the failing test**

```go
package sqlite_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func policy(name string, siteID *string, at time.Time) template.LinkPolicy {
	scope := template.ScopeGlobal
	if siteID != nil {
		scope = template.ScopeSite
	}
	return template.LinkPolicy{
		ID: id.New(), Scope: scope, SiteID: siteID, Name: name,
		Rules:          template.LinkRules{UpDepth: 2, DownLinks: true, SiblingMinWeight: 0.5, MaxLinks: 10, MaxPerTarget: 1, ParentLinkWithinParagraphs: 2, ChildrenSection: true},
		ForbidExternal: true, ForbidSelf: true, AnchorStrategy: template.AnchorPreferUser, CreatedAt: at, UpdatedAt: at,
	}
}

func policyNames(list []template.LinkPolicy) []string {
	out := make([]string, 0, len(list))
	for _, item := range list {
		out = append(out, item.Name)
	}
	return out
}

func TestLinkPolicyRepoRoundTrip(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	repo := sqlite.NewLinkPolicyRepo(store)
	want := policy("Default", nil, sqlitetest.Stamp)

	if err := repo.Insert(t.Context(), want); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	got, err := repo.Get(t.Context(), want.ID)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Get = %+v, %v\nwant %+v", got, err, want)
	}
	if err = repo.Insert(t.Context(), policy("default", nil, sqlitetest.Stamp)); !errors.IsCode(err, errors.Conflict) {
		t.Errorf("duplicate global name code = %q, want CONFLICT", errors.CodeOf(err))
	}
	if err = repo.Insert(t.Context(), policy("Default", &owner.ID, sqlitetest.Stamp)); err != nil {
		t.Fatalf("a site may reuse a global name: %v", err)
	}

	want.Name = "Relaxed"
	want.ForbidExternal = false
	want.AnchorStrategy = template.AnchorRotate
	want.Rules.MaxLinks = 30
	want.UpdatedAt = sqlitetest.Stamp.Add(time.Minute)
	if err = repo.Update(t.Context(), want); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, err = repo.Get(t.Context(), want.ID)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Get after update = %+v, %v\nwant %+v", got, err, want)
	}

	if err = repo.Delete(t.Context(), want.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err = repo.Delete(t.Context(), want.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Delete twice code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if err = repo.Update(t.Context(), want); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Update missing code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = repo.Get(t.Context(), want.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Get missing code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}

func TestLinkPolicyRepoList(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	repo := sqlite.NewLinkPolicyRepo(store)

	for i, name := range []string{"Default", "loose", "Strict"} {
		if err := repo.Insert(t.Context(), policy(name, nil, sqlitetest.Stamp.Add(time.Duration(i)*time.Minute))); err != nil {
			t.Fatalf("Insert %s: %v", name, err)
		}
	}
	if err := repo.Insert(t.Context(), policy("Shop rules", &owner.ID, sqlitetest.Stamp.Add(time.Hour))); err != nil {
		t.Fatalf("Insert scoped: %v", err)
	}

	global := template.ScopeGlobal
	first, err := repo.List(t.Context(), template.PolicyQuery{Scope: &global, Sort: template.SortName}, paging.Request{Limit: 2})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got := policyNames(first.Items); !reflect.DeepEqual(got, []string{"Default", "loose"}) || !first.HasMore {
		t.Fatalf("first page = %v", got)
	}
	rest, err := repo.List(t.Context(), template.PolicyQuery{Scope: &global, Sort: template.SortName}, paging.Request{After: first.Next, Limit: 2})
	if err != nil {
		t.Fatalf("List after: %v", err)
	}
	if got := policyNames(rest.Items); !reflect.DeepEqual(got, []string{"Strict"}) {
		t.Fatalf("second page = %v", got)
	}
	back, err := repo.List(t.Context(), template.PolicyQuery{Scope: &global, Sort: template.SortName}, paging.Request{Before: rest.Prev, Limit: 2})
	if err != nil {
		t.Fatalf("List before: %v", err)
	}
	if got := policyNames(back.Items); !reflect.DeepEqual(got, []string{"Default", "loose"}) {
		t.Fatalf("backward page = %v", got)
	}
	bySite, err := repo.List(t.Context(), template.PolicyQuery{SiteID: &owner.ID, Sort: template.SortCreatedAt}, paging.Request{Limit: 10})
	if err != nil || len(bySite.Items) != 1 || bySite.Items[0].Name != "Shop rules" {
		t.Errorf("by site = %+v, %v", bySite.Items, err)
	}
	byName, err := repo.List(t.Context(), template.PolicyQuery{Scope: &global, Name: "DEFAULT", Sort: template.SortCreatedAt}, paging.Request{Limit: 10})
	if err != nil || len(byName.Items) != 1 || byName.Items[0].Name != "Default" {
		t.Errorf("by name must be case-insensitive, got %+v, %v", byName.Items, err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapters/sqlite/ -run TestLinkPolicyRepo -v`
Expected: FAIL, `NewLinkPolicyRepo` is undefined.

- [ ] **Step 3: Write the repository**

`internal/adapters/sqlite/link_policy_repo.go`:

```go
package sqlite

import (
	"context"
	"database/sql"

	"github.com/Masterminds/squirrel"

	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	policyColumns = `id, scope, site_id, name, rules, forbid_external, forbid_self, anchor_strategy, created_at, updated_at`
	insertPolicy  = `INSERT INTO link_policies (` + policyColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	updatePolicy  = `UPDATE link_policies SET name = ?, rules = ?, forbid_external = ?, forbid_self = ?, anchor_strategy = ?, updated_at = ? WHERE id = ?`
	deletePolicy  = `DELETE FROM link_policies WHERE id = ?`
	selectPolicy  = `SELECT ` + policyColumns + ` FROM link_policies WHERE id = ?`
)

type LinkPolicyRepo struct {
	store *Store
}

func NewLinkPolicyRepo(store *Store) *LinkPolicyRepo {
	return &LinkPolicyRepo{store: store}
}

func policyNotFound(id string) *errors.Error {
	return errors.New(errors.NotFound, "link policy not found").WithDetail("linkPolicyId", id)
}

func policyConflict(name string) *errors.Error {
	return errors.New(errors.Conflict, "a link policy with this name already exists in this scope").WithDetail("name", name)
}

func (r *LinkPolicyRepo) Insert(ctx context.Context, p template.LinkPolicy) error {
	rules, err := encodeJSON(p.Rules)
	if err != nil {
		return err
	}
	_, err = execWrite(ctx, r.store.writeFrom(ctx), insertPolicy, []any{
		p.ID, string(p.Scope), nullString(p.SiteID), p.Name, rules, boolInt(p.ForbidExternal), boolInt(p.ForbidSelf), string(p.AnchorStrategy),
		formatTime(p.CreatedAt), formatTime(p.UpdatedAt),
	}, policyConflict(p.Name), "insert the link policy")
	return err
}

func (r *LinkPolicyRepo) Update(ctx context.Context, p template.LinkPolicy) error {
	rules, err := encodeJSON(p.Rules)
	if err != nil {
		return err
	}
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), updatePolicy, []any{
		p.Name, rules, boolInt(p.ForbidExternal), boolInt(p.ForbidSelf), string(p.AnchorStrategy), formatTime(p.UpdatedAt), p.ID,
	}, policyConflict(p.Name), "update the link policy")
	return requireAffected(affected, err, policyNotFound(p.ID))
}

func (r *LinkPolicyRepo) Delete(ctx context.Context, id string) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), deletePolicy, []any{id}, nil, "delete the link policy")
	return requireAffected(affected, err, policyNotFound(id))
}

func (r *LinkPolicyRepo) Get(ctx context.Context, id string) (template.LinkPolicy, error) {
	return selectOne(ctx, r.store.execFrom(ctx), selectPolicy, []any{id}, scanPolicy, policyNotFound(id), "read the link policy")
}

func policyKeyset(q template.PolicyQuery) paging.Keyset[template.LinkPolicy] {
	key := paging.TimeKey[template.LinkPolicy]("createdAt", "created_at", func(p template.LinkPolicy) any { return p.CreatedAt })
	if q.Sort == template.SortName {
		key = paging.TextKey[template.LinkPolicy]("name", "name", func(p template.LinkPolicy) any { return p.Name })
	}
	return paging.Keyset[template.LinkPolicy]{
		IDColumn: "id",
		ID:       func(p template.LinkPolicy) string { return p.ID },
		Keys:     []paging.SortKey[template.LinkPolicy]{key},
		Desc:     q.Desc,
	}
}

func (r *LinkPolicyRepo) List(ctx context.Context, q template.PolicyQuery, page paging.Request) (paging.List[template.LinkPolicy], error) {
	builder := squirrel.Select(policyColumns).From("link_policies")
	if q.Scope != nil {
		builder = builder.Where(squirrel.Eq{"scope": string(*q.Scope)})
	}
	if q.SiteID != nil {
		builder = builder.Where(squirrel.Eq{"site_id": *q.SiteID})
	}
	if q.Name != "" {
		builder = builder.Where(squirrel.Eq{"name": q.Name})
	}

	keyset := policyKeyset(q)
	keyed, err := keyset.Apply(builder, page)
	if err != nil {
		return paging.List[template.LinkPolicy]{}, err
	}
	query, args, err := buildQuery(keyed, "link policies")
	if err != nil {
		return paging.List[template.LinkPolicy]{}, err
	}
	rows, err := selectAll(ctx, r.store.execFrom(ctx), query, args, scanPolicy, "list the link policies")
	if err != nil {
		return paging.List[template.LinkPolicy]{}, err
	}
	return keyset.Cut(rows, page)
}

func scanPolicy(rows *sql.Rows) (template.LinkPolicy, error) {
	var (
		p                          template.LinkPolicy
		scope, rules, strategy     string
		siteID                     sql.NullString
		forbidExternal, forbidSelf int64
		createdAt, updatedAt       string
	)
	if err := rows.Scan(&p.ID, &scope, &siteID, &p.Name, &rules, &forbidExternal, &forbidSelf, &strategy, &createdAt, &updatedAt); err != nil {
		return template.LinkPolicy{}, err
	}
	p.Scope = template.Scope(scope)
	p.SiteID = optString(siteID)
	p.ForbidExternal = forbidExternal == 1
	p.ForbidSelf = forbidSelf == 1
	p.AnchorStrategy = template.AnchorStrategy(strategy)
	if err := decodeJSON(rules, &p.Rules, "decode the link policy rules"); err != nil {
		return template.LinkPolicy{}, err
	}

	var err error
	if p.CreatedAt, err = parseTime(createdAt); err != nil {
		return template.LinkPolicy{}, err
	}
	if p.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return template.LinkPolicy{}, err
	}
	return p, nil
}
```

- [ ] **Step 4: Run tests to verify they pass and close Half 1**

```bash
go test -race -covermode=atomic -coverprofile=coverage.out ./...
go run ./cmd/covergate
go vet ./...
$(go env GOPATH)/bin/golangci-lint run
go tool cover -func=coverage.out | grep -E 'internal/(domain|adapters/sqlite)' | tail -n 30
```

Expected: everything green, 0 lint issues, `internal/domain/...` packages each ≥ 85%, `internal/adapters/sqlite` ≥ 85%. The `deps_test.go` domain rules now run and pass: `internal/domain` reaches only `kernel/errors` and the standard library.

- [ ] **Step 5: Commit**

```bash
git add internal/adapters/sqlite/link_policy_repo.go internal/adapters/sqlite/link_policy_repo_test.go
git commit -m "feat(sqlite): link policy repository

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

# Half 2 — use cases, wiring, seeding, docs

### Task 18: The application root: publisher port, paging helpers, recording publisher

**Files:**
- Create: `internal/application/publisher.go`, `internal/application/paging.go`, `internal/application/applicationtest/recorder.go`
- Test: `internal/application/paging_test.go`

**Interfaces:**
- Consumes: `events.Type` (existing), `dto.ListRequest`, `paging.Request|List|Cursor|Cursors`.
- Produces: `application.Publisher` interface `{ Publish(eventType events.Type, payload any) error }` (the consumer-side port every use case takes; `*wails.EventBridge` and `*app.EventRelay` satisfy it), `application.PageRequest(req dto.ListRequest) paging.Request`, `application.MapList[T, V any](list paging.List[T], convert func(T) V) paging.List[V]`; `applicationtest.Recorder` with `Publish`, `Events() []applicationtest.Event{Type events.Type; Payload any}` and `Reset()`.

- [ ] **Step 1: Write the failing test**

`internal/application/paging_test.go`:

```go
package application_test

import (
	"strconv"
	"testing"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func TestPageRequest(t *testing.T) {
	t.Parallel()

	got := application.PageRequest(dto.ListRequest{Cursor: "abc", Limit: 25})
	if got.After != paging.Cursor("abc") || got.Before != "" || got.Limit != 25 {
		t.Fatalf("PageRequest = %+v", got)
	}
	if empty := application.PageRequest(dto.ListRequest{}); empty.After != "" || empty.Limit != 0 {
		t.Fatalf("PageRequest of an empty request = %+v", empty)
	}
}

func TestMapList(t *testing.T) {
	t.Parallel()

	in := paging.List[int]{Cursors: paging.Cursors{Next: "n", Prev: "p"}, Items: []int{1, 2}, HasMore: true}
	out := application.MapList(in, strconv.Itoa)
	if out.Next != "n" || out.Prev != "p" || !out.HasMore || len(out.Items) != 2 || out.Items[0] != "1" || out.Items[1] != "2" {
		t.Fatalf("MapList = %+v", out)
	}

	empty := application.MapList(paging.List[int]{}, strconv.Itoa)
	if empty.Items == nil || len(empty.Items) != 0 {
		t.Fatalf("MapList of an empty list must materialise an empty slice, got %#v", empty.Items)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/application/ -v`
Expected: FAIL, the root package has no source yet.

- [ ] **Step 3: Write the implementation**

`internal/application/publisher.go`:

```go
package application

import "github.com/davidmovas/postulator/internal/application/events"

type Publisher interface {
	Publish(eventType events.Type, payload any) error
}
```

`internal/application/paging.go`:

```go
package application

import (
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func PageRequest(req dto.ListRequest) paging.Request {
	return paging.Request{After: paging.Cursor(req.Cursor), Limit: req.Limit}
}

func MapList[T, V any](list paging.List[T], convert func(T) V) paging.List[V] {
	items := make([]V, 0, len(list.Items))
	for _, item := range list.Items {
		items = append(items, convert(item))
	}
	return paging.List[V]{Cursors: list.Cursors, Items: items, HasMore: list.HasMore}
}
```

`internal/application/applicationtest/recorder.go`:

```go
package applicationtest

import (
	"slices"
	"sync"

	"github.com/davidmovas/postulator/internal/application/events"
)

type Event struct {
	Type    events.Type
	Payload any
}

type Recorder struct {
	mu       sync.Mutex
	recorded []Event
}

func (r *Recorder) Publish(eventType events.Type, payload any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.recorded = append(r.recorded, Event{Type: eventType, Payload: payload})
	return nil
}

func (r *Recorder) Events() []Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.recorded)
}

func (r *Recorder) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.recorded = nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race ./internal/application/... -v && go test -race ./internal/app/ -run 'TestLayers|TestOnly' -v`
Expected: PASS; `application` reaches only `domain`, `kernel` and (transitively through `kernel/paging`) squirrel, which the direct-import rule allows.

- [ ] **Step 5: Commit**

```bash
git add internal/application/publisher.go internal/application/paging.go internal/application/paging_test.go internal/application/applicationtest/
git commit -m "feat(application): publisher port, paging helpers and a recording publisher

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 19: `application/sites`

**Files:**
- Create: `internal/application/sites/service.go`, `internal/application/sites/views.go`, `internal/application/sites/requests.go`
- Test: `internal/application/sites/service_test.go`

**Interfaces:**
- Consumes: `site.*` (Task 2), `llm.Role|ModelRef`, `application.PageRequest|MapList`, `id.New`, `clock.Clock`, `dto.Time|NewTime|ListRequest`, `paging.List`, `errors.*`; test-side `sqlite.NewSiteRepo|NewSecretsRepo|NewEntityRepo|NewPageRepo`, `secrets.NewStore`, `sqlitetest.Open|Key|Entity|Page`.
- Produces: `sites.Service` with `New(store siteStore, secrets secretStore, uow unitOfWork, clk clock.Clock) *Service` and methods `Create(ctx, CreateRequest) (CreateResponse, error)`, `Update(ctx, UpdateRequest) (UpdateResponse, error)`, `Delete(ctx, DeleteRequest) (DeleteResponse, error)`, `Get(ctx, GetRequest) (GetResponse, error)`, `List(ctx, ListRequest) (paging.List[Site], error)`; views `Site`, `Plugin`, `Defaults`.

- [ ] **Step 1: Write the failing test**

```go
package sites_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/secrets"
	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type harness struct {
	service *sites.Service
	store   *sqlite.Store
	secrets *secrets.Store
	clock   *clock.Fake
}

func newHarness(t *testing.T) harness {
	t.Helper()

	store := sqlitetest.Open(t)
	clk := clock.NewFake(time.Date(2026, time.September, 18, 9, 0, 0, 0, time.UTC))
	vault := secrets.NewStore(sqlite.NewSecretsRepo(store, clk), sqlitetest.Key())
	return harness{
		service: sites.New(sqlite.NewSiteRepo(store), vault, store, clk),
		store:   store,
		secrets: vault,
		clock:   clk,
	}
}

func ptr[T any](v T) *T {
	return &v
}

func TestCreateStoresTheSiteAndItsPassword(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	created, err := h.service.Create(t.Context(), sites.CreateRequest{Name: " Shop ", BaseURL: "https://Shop.example.com/", Username: "editor", Password: "abcd efgh"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	got := created.Site
	if got.ID == "" || got.Name != "Shop" || got.BaseURL != "https://shop.example.com" || got.Username != "editor" || got.Status != "active" || got.AllowInsecure {
		t.Errorf("Create = %+v", got)
	}
	if got.CreatedAt.String() != "2026-09-18T09:00:00Z" || got.UpdatedAt.String() != got.CreatedAt.String() {
		t.Errorf("timestamps = %s / %s", got.CreatedAt, got.UpdatedAt)
	}

	password, err := h.secrets.Get(t.Context(), site.SecretRef(got.ID))
	if err != nil || password != "abcd efgh" {
		t.Errorf("stored password = %q, %v", password, err)
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	for _, key := range []string{`"id"`, `"baseUrl"`, `"allowInsecure"`, `"plugin":{"installed":false`, `"capabilities":[]`, `"defaults":{"templateId":null`, `"modelProfiles":{}`, `"createdAt":"2026-09-18T09:00:00Z"`} {
		if !strings.Contains(string(encoded), key) {
			t.Errorf("view lacks %s: %s", key, encoded)
		}
	}
	if strings.Contains(string(encoded), "secret") || strings.Contains(string(encoded), "abcd") {
		t.Errorf("the view must not carry the secret: %s", encoded)
	}
}

func TestCreateWithoutAPasswordStoresNoSecret(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	created, err := h.service.Create(t.Context(), sites.CreateRequest{Name: "Shop", BaseURL: "http://shop.local", AllowInsecure: true})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err = h.secrets.Get(t.Context(), site.SecretRef(created.Site.ID)); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("secret code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}

func TestCreateRejectsBadInput(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	cases := []struct {
		name  string
		req   sites.CreateRequest
		field string
	}{
		{name: "blank name", req: sites.CreateRequest{Name: " ", BaseURL: "https://a.example.com"}, field: "name"},
		{name: "http without allow", req: sites.CreateRequest{Name: "A", BaseURL: "http://a.example.com"}, field: "baseUrl"},
		{name: "no url", req: sites.CreateRequest{Name: "A"}, field: "baseUrl"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := h.service.Create(t.Context(), tc.req)
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("code = %q, want INVALID", errors.CodeOf(err))
			}
		})
	}
}

func TestUpdateAppliesOnlyTheGivenFields(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	created, err := h.service.Create(t.Context(), sites.CreateRequest{Name: "Shop", BaseURL: "https://shop.example.com", Password: "one"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	h.clock.Advance(time.Hour)

	updated, err := h.service.Update(t.Context(), sites.UpdateRequest{
		ID: created.Site.ID, Name: ptr("Shop Two"), Status: ptr("paused"), Password: ptr("two"),
		Defaults: &sites.Defaults{TemplateID: nil, ModelProfiles: map[string]llm.ModelRef{"writer": {Provider: "openai", Model: "gpt"}}},
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	got := updated.Site
	if got.Name != "Shop Two" || got.Status != "paused" || got.BaseURL != "https://shop.example.com" || got.Defaults.ModelProfiles["writer"].Model != "gpt" {
		t.Errorf("Update = %+v", got)
	}
	if got.UpdatedAt.String() != "2026-09-18T10:00:00Z" || got.CreatedAt.String() != "2026-09-18T09:00:00Z" {
		t.Errorf("timestamps = %s / %s", got.CreatedAt, got.UpdatedAt)
	}
	if password, getErr := h.secrets.Get(t.Context(), site.SecretRef(got.ID)); getErr != nil || password != "two" {
		t.Errorf("rotated password = %q, %v", password, getErr)
	}

	if _, err = h.service.Update(t.Context(), sites.UpdateRequest{ID: created.Site.ID, Password: ptr("")}); err != nil {
		t.Fatalf("Update clearing the password: %v", err)
	}
	if _, err = h.secrets.Get(t.Context(), site.SecretRef(got.ID)); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("an empty password must delete the secret, got %v", err)
	}

	if _, err = h.service.Update(t.Context(), sites.UpdateRequest{ID: created.Site.ID, Status: ptr("sleeping")}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad status code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.Update(t.Context(), sites.UpdateRequest{ID: created.Site.ID, BaseURL: ptr("http://shop.example.com")}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("http without allow code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.Update(t.Context(), sites.UpdateRequest{ID: "missing", Name: ptr("x")}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("missing site code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = h.service.Update(t.Context(), sites.UpdateRequest{ID: created.Site.ID, Defaults: &sites.Defaults{TemplateID: ptr("no-such-template")}}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("unknown default template code = %q, want INVALID", errors.CodeOf(err))
	}
}

func TestDeleteCascadesAndRemovesTheSecret(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	created, err := h.service.Create(t.Context(), sites.CreateRequest{Name: "Shop", BaseURL: "https://shop.example.com", Password: "one"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	entity := sqlitetest.Entity(t, h.store, created.Site.ID, "Shoes")
	page := sqlitetest.Page(t, h.store, created.Site.ID, "/shoes/")

	if _, err = h.service.Delete(t.Context(), sites.DeleteRequest{ID: created.Site.ID}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err = h.service.Get(t.Context(), sites.GetRequest{ID: created.Site.ID}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Get after delete code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = sqlite.NewEntityRepo(h.store).Get(t.Context(), entity.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("entity must cascade, got %v", err)
	}
	if _, err = sqlite.NewPageRepo(h.store).Get(t.Context(), page.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("page must cascade, got %v", err)
	}
	if _, err = h.secrets.Get(t.Context(), site.SecretRef(created.Site.ID)); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("secret must be removed, got %v", err)
	}
	if _, err = h.service.Delete(t.Context(), sites.DeleteRequest{ID: created.Site.ID}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Delete twice code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}

func TestGetAndList(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	for i, name := range []string{"alpha", "bravo", "charlie"} {
		created, err := h.service.Create(t.Context(), sites.CreateRequest{Name: name, BaseURL: "https://" + name + ".example.com"})
		if err != nil {
			t.Fatalf("Create %s: %v", name, err)
		}
		if i == 1 {
			if _, err = h.service.Update(t.Context(), sites.UpdateRequest{ID: created.Site.ID, Status: ptr("paused")}); err != nil {
				t.Fatalf("Update: %v", err)
			}
		}
		h.clock.Advance(time.Minute)
	}

	page, err := h.service.List(t.Context(), sites.ListRequest{ListRequest: dto.ListRequest{Limit: 2}})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.Items) != 2 || page.Items[0].Name != "alpha" || !page.HasMore || page.Next == "" {
		t.Fatalf("List = %+v", page)
	}
	rest, err := h.service.List(t.Context(), sites.ListRequest{ListRequest: dto.ListRequest{Cursor: string(page.Next), Limit: 2}})
	if err != nil || len(rest.Items) != 1 || rest.Items[0].Name != "charlie" {
		t.Errorf("List after = %+v, %v", rest, err)
	}
	paused, err := h.service.List(t.Context(), sites.ListRequest{Status: "paused"})
	if err != nil || len(paused.Items) != 1 || paused.Items[0].Name != "bravo" {
		t.Errorf("List paused = %+v, %v", paused, err)
	}
	byName, err := h.service.List(t.Context(), sites.ListRequest{ListRequest: dto.ListRequest{Sort: &dto.Sort{Field: "name", Desc: true}}})
	if err != nil || len(byName.Items) != 3 || byName.Items[0].Name != "charlie" {
		t.Errorf("List by name desc = %+v, %v", byName, err)
	}
	if _, err = h.service.List(t.Context(), sites.ListRequest{ListRequest: dto.ListRequest{Sort: &dto.Sort{Field: "age"}}}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("unknown sort code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.List(t.Context(), sites.ListRequest{Status: "sleeping"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("unknown status code = %q, want INVALID", errors.CodeOf(err))
	}

	got, err := h.service.Get(t.Context(), sites.GetRequest{ID: page.Items[0].ID})
	if err != nil || got.Site.Name != "alpha" {
		t.Errorf("Get = %+v, %v", got, err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/application/sites/ -v`
Expected: FAIL, the package does not exist.

- [ ] **Step 3: Write the implementation**

`internal/application/sites/views.go`:

```go
package sites

import (
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

type Plugin struct {
	Installed    bool     `json:"installed"`
	Version      string   `json:"version"`
	Capabilities []string `json:"capabilities"`
	SEOPlugin    string   `json:"seoPlugin"`
}

type Defaults struct {
	TemplateID    *string                 `json:"templateId"`
	LinkPolicyID  *string                 `json:"linkPolicyId"`
	ModelProfiles map[string]llm.ModelRef `json:"modelProfiles"`
}

type Site struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	BaseURL       string   `json:"baseUrl"`
	Username      string   `json:"username"`
	Status        string   `json:"status"`
	AllowInsecure bool     `json:"allowInsecure"`
	Plugin        Plugin   `json:"plugin"`
	Defaults      Defaults `json:"defaults"`
	CreatedAt     dto.Time `json:"createdAt"`
	UpdatedAt     dto.Time `json:"updatedAt"`
}

func view(s site.Site) Site {
	capabilities := s.Plugin.Capabilities
	if capabilities == nil {
		capabilities = []string{}
	}
	profiles := make(map[string]llm.ModelRef, len(s.Defaults.ModelProfiles))
	for role, ref := range s.Defaults.ModelProfiles {
		profiles[string(role)] = ref
	}
	return Site{
		ID:            s.ID,
		Name:          s.Name,
		BaseURL:       s.BaseURL,
		Username:      s.Username,
		Status:        string(s.Status),
		AllowInsecure: s.AllowInsecure,
		Plugin:        Plugin{Installed: s.Plugin.Installed, Version: s.Plugin.Version, Capabilities: capabilities, SEOPlugin: s.Plugin.SEOPlugin},
		Defaults:      Defaults{TemplateID: s.Defaults.TemplateID, LinkPolicyID: s.Defaults.LinkPolicyID, ModelProfiles: profiles},
		CreatedAt:     dto.NewTime(s.CreatedAt),
		UpdatedAt:     dto.NewTime(s.UpdatedAt),
	}
}

func profilesOf(profiles map[string]llm.ModelRef) map[llm.Role]llm.ModelRef {
	out := make(map[llm.Role]llm.ModelRef, len(profiles))
	for role, ref := range profiles {
		out[llm.Role(role)] = ref
	}
	return out
}
```

`internal/application/sites/requests.go`:

```go
package sites

import "github.com/davidmovas/postulator/internal/kernel/dto"

type CreateRequest struct {
	Name          string `json:"name"`
	BaseURL       string `json:"baseUrl"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	AllowInsecure bool   `json:"allowInsecure"`
}

type CreateResponse struct {
	Site Site `json:"site"`
}

type UpdateRequest struct {
	ID            string    `json:"id"`
	Name          *string   `json:"name,omitempty"`
	BaseURL       *string   `json:"baseUrl,omitempty"`
	Username      *string   `json:"username,omitempty"`
	Password      *string   `json:"password,omitempty"`
	AllowInsecure *bool     `json:"allowInsecure,omitempty"`
	Status        *string   `json:"status,omitempty"`
	Defaults      *Defaults `json:"defaults,omitempty"`
}

type UpdateResponse struct {
	Site Site `json:"site"`
}

type DeleteRequest struct {
	ID string `json:"id"`
}

type DeleteResponse struct{}

type GetRequest struct {
	ID string `json:"id"`
}

type GetResponse struct {
	Site Site `json:"site"`
}

type ListRequest struct {
	dto.ListRequest
	Status string `json:"status,omitempty"`
}
```

`internal/application/sites/service.go`:

```go
package sites

import (
	"context"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

type siteStore interface {
	Insert(ctx context.Context, s site.Site) error
	Update(ctx context.Context, s site.Site) error
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (site.Site, error)
	List(ctx context.Context, q site.Query, page paging.Request) (paging.List[site.Site], error)
}

type secretStore interface {
	Put(ctx context.Context, ref, value string) error
	Delete(ctx context.Context, ref string) error
}

type unitOfWork interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type Service struct {
	store   siteStore
	secrets secretStore
	uow     unitOfWork
	clock   clock.Clock
}

func New(store siteStore, secrets secretStore, uow unitOfWork, clk clock.Clock) *Service {
	return &Service{store: store, secrets: secrets, uow: uow, clock: clk}
}

func (s *Service) now() time.Time {
	return s.clock.Now().UTC().Truncate(time.Second)
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (CreateResponse, error) {
	baseURL, err := site.NormalizeBaseURL(req.BaseURL, req.AllowInsecure)
	if err != nil {
		return CreateResponse{}, err
	}

	now := s.now()
	record := site.Site{
		ID:            id.New(),
		Name:          strings.TrimSpace(req.Name),
		BaseURL:       baseURL,
		Username:      strings.TrimSpace(req.Username),
		Status:        site.StatusActive,
		AllowInsecure: req.AllowInsecure,
		Plugin:        site.PluginState{Capabilities: []string{}},
		Defaults:      site.Defaults{ModelProfiles: map[llm.Role]llm.ModelRef{}},
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	record.SecretRef = site.SecretRef(record.ID)
	if err = record.Validate(); err != nil {
		return CreateResponse{}, err
	}

	err = s.uow.Do(ctx, func(c context.Context) error {
		if insertErr := s.store.Insert(c, record); insertErr != nil {
			return insertErr
		}
		if req.Password == "" {
			return nil
		}
		return s.secrets.Put(c, record.SecretRef, req.Password)
	})
	if err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse{Site: view(record)}, nil
}

func (s *Service) Update(ctx context.Context, req UpdateRequest) (UpdateResponse, error) {
	var updated site.Site
	err := s.uow.Do(ctx, func(c context.Context) error {
		current, getErr := s.store.Get(c, req.ID)
		if getErr != nil {
			return getErr
		}
		next, applyErr := apply(current, req)
		if applyErr != nil {
			return applyErr
		}
		next.UpdatedAt = s.now()
		if validateErr := next.Validate(); validateErr != nil {
			return validateErr
		}
		if updateErr := s.store.Update(c, next); updateErr != nil {
			return updateErr
		}
		if req.Password != nil {
			if rotateErr := s.rotate(c, next.SecretRef, *req.Password); rotateErr != nil {
				return rotateErr
			}
		}
		updated = next
		return nil
	})
	if err != nil {
		return UpdateResponse{}, err
	}
	return UpdateResponse{Site: view(updated)}, nil
}

func apply(current site.Site, req UpdateRequest) (site.Site, error) {
	next := current
	if req.Name != nil {
		next.Name = strings.TrimSpace(*req.Name)
	}
	if req.Username != nil {
		next.Username = strings.TrimSpace(*req.Username)
	}
	if req.AllowInsecure != nil {
		next.AllowInsecure = *req.AllowInsecure
	}
	if req.BaseURL != nil {
		baseURL, err := site.NormalizeBaseURL(*req.BaseURL, next.AllowInsecure)
		if err != nil {
			return site.Site{}, err
		}
		next.BaseURL = baseURL
	}
	if req.Status != nil {
		next.Status = site.Status(*req.Status)
	}
	if req.Defaults != nil {
		next.Defaults = site.Defaults{
			TemplateID:    req.Defaults.TemplateID,
			LinkPolicyID:  req.Defaults.LinkPolicyID,
			ModelProfiles: profilesOf(req.Defaults.ModelProfiles),
		}
	}
	return next, nil
}

func (s *Service) rotate(ctx context.Context, ref, password string) error {
	if password == "" {
		return ignoreNotFound(s.secrets.Delete(ctx, ref))
	}
	return s.secrets.Put(ctx, ref, password)
}

func ignoreNotFound(err error) error {
	if errors.IsCode(err, errors.NotFound) {
		return nil
	}
	return err
}

func (s *Service) Delete(ctx context.Context, req DeleteRequest) (DeleteResponse, error) {
	err := s.uow.Do(ctx, func(c context.Context) error {
		current, getErr := s.store.Get(c, req.ID)
		if getErr != nil {
			return getErr
		}
		if deleteErr := s.store.Delete(c, req.ID); deleteErr != nil {
			return deleteErr
		}
		return ignoreNotFound(s.secrets.Delete(c, current.SecretRef))
	})
	if err != nil {
		return DeleteResponse{}, err
	}
	return DeleteResponse{}, nil
}

func (s *Service) Get(ctx context.Context, req GetRequest) (GetResponse, error) {
	record, err := s.store.Get(ctx, req.ID)
	if err != nil {
		return GetResponse{}, err
	}
	return GetResponse{Site: view(record)}, nil
}

func (s *Service) List(ctx context.Context, req ListRequest) (paging.List[Site], error) {
	q := site.Query{Sort: site.SortCreatedAt}
	if req.Status != "" {
		status := site.Status(req.Status)
		if !status.Valid() {
			return paging.List[Site]{}, errors.New(errors.Invalid, "site status is not recognised").WithDetail("field", "status")
		}
		q.Status = &status
	}
	if req.Sort != nil {
		q.Sort = site.Sort(req.Sort.Field)
		if !q.Sort.Valid() {
			return paging.List[Site]{}, errors.New(errors.Invalid, "sites cannot be sorted by this field").WithDetail("field", "sort.field")
		}
		q.Desc = req.Sort.Desc
	}

	list, err := s.store.List(ctx, q, application.PageRequest(req.ListRequest))
	if err != nil {
		return paging.List[Site]{}, err
	}
	return application.MapList(list, view), nil
}
```

The secret store's `Put` and `Delete` run inside the same transaction as the site row because `secrets.Store` writes through `SecretsRepo`, which resolves its executor from the context.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race -cover ./internal/application/sites/ -v`
Expected: PASS, coverage ≥ 85%.

- [ ] **Step 5: Commit**

```bash
git add internal/application/sites/
git commit -m "feat(application): site use cases with the password kept in the secret store

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---
### Task 20: `application/graph` — entities

**Files:**
- Create: `internal/application/graph/service.go`, `internal/application/graph/views.go`, `internal/application/graph/requests.go`, `internal/application/graph/entities.go`
- Test: `internal/application/graph/entities_test.go`

**Interfaces:**
- Consumes: `graph.*` from `internal/domain/graph` imported as `graphdomain` (the application package is also named `graph`), `site.Site`, `application.Publisher|PageRequest|MapList`, `events.GraphChanged|PagesChanged` and payloads, `id.New`, `clock.Clock`, `dto.*`, `paging.*`, `errors.*`; test-side `sqlite.NewEntityRepo|NewEdgeRepo|NewSiteRepo`, `sqlitetest.Open|Site`, `applicationtest.Recorder`.
- Produces: `graph.Service` with `New(entities entityStore, edges edgeStore, sites siteReader, uow unitOfWork, publisher application.Publisher, clk clock.Clock) *Service`; views `Entity`, `Anchor`, `Edge`; entity methods `CreateEntity`, `UpdateEntity`, `DeleteEntity`, `GetEntity`, `ListEntities`, `SetAnchors` with the request/response pairs below. Task 21 adds the edge methods on the same struct.

- [ ] **Step 1: Write the failing test**

```go
package graph_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/applicationtest"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type harness struct {
	service  *graph.Service
	store    *sqlite.Store
	recorder *applicationtest.Recorder
	clock    *clock.Fake
	siteID   string
}

func newHarness(t *testing.T) harness {
	t.Helper()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	recorder := &applicationtest.Recorder{}
	clk := clock.NewFake(time.Date(2026, time.September, 18, 9, 0, 0, 0, time.UTC))
	return harness{
		service:  graph.New(sqlite.NewEntityRepo(store), sqlite.NewEdgeRepo(store), sqlite.NewSiteRepo(store), store, recorder, clk),
		store:    store,
		recorder: recorder,
		clock:    clk,
		siteID:   owner.ID,
	}
}

func ptr[T any](v T) *T {
	return &v
}

func (h harness) entity(t *testing.T, name, kind string) graph.Entity {
	t.Helper()
	created, err := h.service.CreateEntity(t.Context(), graph.CreateEntityRequest{SiteID: h.siteID, Name: name, Kind: kind, PrimaryKeyword: name})
	if err != nil {
		t.Fatalf("CreateEntity %s: %v", name, err)
	}
	return created.Entity
}

func (h harness) wantEvents(t *testing.T, want ...events.Type) {
	t.Helper()
	got := h.recorder.Events()
	if len(got) != len(want) {
		t.Fatalf("published %d events, want %d: %+v", len(got), len(want), got)
	}
	for i, event := range got {
		if event.Type != want[i] {
			t.Errorf("event[%d] = %s, want %s", i, event.Type, want[i])
		}
		switch payload := event.Payload.(type) {
		case events.GraphChangedPayload:
			if payload.SiteID != h.siteID {
				t.Errorf("event[%d] site = %s", i, payload.SiteID)
			}
		case events.PagesChangedPayload:
			if payload.SiteID != h.siteID {
				t.Errorf("event[%d] site = %s", i, payload.SiteID)
			}
		default:
			t.Errorf("event[%d] carries %T", i, event.Payload)
		}
	}
	h.recorder.Reset()
}

func TestCreateEntity(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	created, err := h.service.CreateEntity(t.Context(), graph.CreateEntityRequest{
		SiteID: h.siteID, Name: " Running Shoes ", Kind: "hub", Intent: "commercial", PrimaryKeyword: "running shoes",
		SecondaryKeywords: []string{"trail shoes", "trail shoes"}, Anchors: []graph.Anchor{{Text: "running shoes", Source: "user", Weight: 1}},
	})
	if err != nil {
		t.Fatalf("CreateEntity: %v", err)
	}
	got := created.Entity
	if got.ID == "" || got.Name != "Running Shoes" || got.Kind != "hub" || got.Source != "user" || len(got.SecondaryKeywords) != 1 || len(got.Anchors) != 1 || got.Score != 0 {
		t.Errorf("CreateEntity = %+v", got)
	}
	h.wantEvents(t, events.GraphChanged)

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	for _, key := range []string{`"siteId"`, `"primaryKeyword"`, `"secondaryKeywords":["trail shoes"]`, `"anchors":[{"text":"running shoes","source":"user","weight":1}]`, `"canonicalPageId":null`, `"createdAt":"2026-09-18T09:00:00Z"`} {
		if !strings.Contains(string(encoded), key) {
			t.Errorf("view lacks %s: %s", key, encoded)
		}
	}

	if _, err = h.service.CreateEntity(t.Context(), graph.CreateEntityRequest{SiteID: h.siteID, Name: "running shoes", Kind: "topic"}); !errors.IsCode(err, errors.Conflict) {
		t.Errorf("duplicate name code = %q, want CONFLICT", errors.CodeOf(err))
	}
	if _, err = h.service.CreateEntity(t.Context(), graph.CreateEntityRequest{SiteID: "missing", Name: "x", Kind: "topic"}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("unknown site code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = h.service.CreateEntity(t.Context(), graph.CreateEntityRequest{SiteID: h.siteID, Name: "x", Kind: "planet"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("unknown kind code = %q, want INVALID", errors.CodeOf(err))
	}
	if len(h.recorder.Events()) != 0 {
		t.Error("a failed mutation must not publish")
	}
}

func TestUpdateSetAnchorsAndGet(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	entity := h.entity(t, "Shoes", "hub")
	h.recorder.Reset()
	h.clock.Advance(time.Minute)

	updated, err := h.service.UpdateEntity(t.Context(), graph.UpdateEntityRequest{ID: entity.ID, Name: ptr("Footwear"), Intent: ptr("informational"), SecondaryKeywords: ptr([]string{"boots"})})
	if err != nil {
		t.Fatalf("UpdateEntity: %v", err)
	}
	if updated.Entity.Name != "Footwear" || updated.Entity.Kind != "hub" || updated.Entity.PrimaryKeyword != "Shoes" || updated.Entity.Intent != "informational" || updated.Entity.UpdatedAt.String() != "2026-09-18T09:01:00Z" {
		t.Errorf("UpdateEntity = %+v", updated.Entity)
	}
	h.wantEvents(t, events.GraphChanged)

	anchored, err := h.service.SetAnchors(t.Context(), graph.SetAnchorsRequest{EntityID: entity.ID, Anchors: []graph.Anchor{{Text: "footwear", Source: "ai", Weight: 0.5}, {Text: "shoes", Source: "user", Weight: 1}}})
	if err != nil {
		t.Fatalf("SetAnchors: %v", err)
	}
	if len(anchored.Entity.Anchors) != 2 || anchored.Entity.Anchors[0].Text != "footwear" {
		t.Errorf("SetAnchors = %+v", anchored.Entity.Anchors)
	}
	h.wantEvents(t, events.GraphChanged)

	got, err := h.service.GetEntity(t.Context(), graph.GetEntityRequest{ID: entity.ID})
	if err != nil || got.Entity.Name != "Footwear" || len(got.Entity.Anchors) != 2 {
		t.Errorf("GetEntity = %+v, %v", got.Entity, err)
	}

	if _, err = h.service.SetAnchors(t.Context(), graph.SetAnchorsRequest{EntityID: entity.ID, Anchors: []graph.Anchor{{Text: "a", Source: "user"}, {Text: "A", Source: "ai"}}}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("repeated anchor code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.UpdateEntity(t.Context(), graph.UpdateEntityRequest{ID: "missing", Name: ptr("x")}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("missing entity code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = h.service.UpdateEntity(t.Context(), graph.UpdateEntityRequest{ID: entity.ID, Kind: ptr("planet")}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad kind code = %q, want INVALID", errors.CodeOf(err))
	}
}

func TestDeleteEntityPublishesGraphAndPages(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	entity := h.entity(t, "Shoes", "hub")
	page := sqlitetest.Page(t, h.store, h.siteID, "/shoes/")
	page.EntityID = &entity.ID
	if err := sqlite.NewPageRepo(h.store).Update(t.Context(), page); err != nil {
		t.Fatalf("map the page: %v", err)
	}
	h.recorder.Reset()

	if _, err := h.service.DeleteEntity(t.Context(), graph.DeleteEntityRequest{ID: entity.ID}); err != nil {
		t.Fatalf("DeleteEntity: %v", err)
	}
	h.wantEvents(t, events.GraphChanged, events.PagesChanged)

	unmapped, err := sqlite.NewPageRepo(h.store).Get(t.Context(), page.ID)
	if err != nil || unmapped.EntityID != nil {
		t.Errorf("page after entity delete = %+v, %v", unmapped, err)
	}
	if _, err = h.service.DeleteEntity(t.Context(), graph.DeleteEntityRequest{ID: entity.ID}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Delete twice code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}

func TestListEntities(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	for _, name := range []string{"Boots", "Sandals", "Shoes"} {
		h.entity(t, name, "product")
		h.clock.Advance(time.Minute)
	}
	h.entity(t, "Footwear", "hub")

	page, err := h.service.ListEntities(t.Context(), graph.ListEntitiesRequest{ListRequest: dto.ListRequest{Limit: 2}, SiteID: h.siteID})
	if err != nil || len(page.Items) != 2 || page.Items[0].Name != "Boots" || !page.HasMore {
		t.Fatalf("ListEntities = %+v, %v", page, err)
	}
	rest, err := h.service.ListEntities(t.Context(), graph.ListEntitiesRequest{ListRequest: dto.ListRequest{Cursor: string(page.Next), Limit: 2}, SiteID: h.siteID})
	if err != nil || len(rest.Items) != 2 || rest.Items[0].Name != "Shoes" || rest.Items[1].Name != "Footwear" {
		t.Errorf("ListEntities after = %+v, %v", rest, err)
	}
	hubs, err := h.service.ListEntities(t.Context(), graph.ListEntitiesRequest{SiteID: h.siteID, Kind: "hub"})
	if err != nil || len(hubs.Items) != 1 || hubs.Items[0].Name != "Footwear" {
		t.Errorf("hubs = %+v, %v", hubs, err)
	}
	prefixed, err := h.service.ListEntities(t.Context(), graph.ListEntitiesRequest{ListRequest: dto.ListRequest{Sort: &dto.Sort{Field: "name"}}, SiteID: h.siteID, NamePrefix: "s"})
	if err != nil || len(prefixed.Items) != 2 || prefixed.Items[0].Name != "Sandals" {
		t.Errorf("prefixed = %+v, %v", prefixed, err)
	}
	mapped, err := h.service.ListEntities(t.Context(), graph.ListEntitiesRequest{SiteID: h.siteID, HasCanonicalPage: ptr(true)})
	if err != nil || len(mapped.Items) != 0 {
		t.Errorf("mapped = %+v, %v", mapped, err)
	}
	if _, err = h.service.ListEntities(t.Context(), graph.ListEntitiesRequest{SiteID: h.siteID, Kind: "planet"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad kind code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.ListEntities(t.Context(), graph.ListEntitiesRequest{ListRequest: dto.ListRequest{Sort: &dto.Sort{Field: "score"}}, SiteID: h.siteID}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad sort code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.ListEntities(t.Context(), graph.ListEntitiesRequest{}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("missing site code = %q, want INVALID", errors.CodeOf(err))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/application/graph/ -v`
Expected: FAIL, the package does not exist.

- [ ] **Step 3: Write the implementation**

`internal/application/graph/views.go`:

```go
package graph

import (
	graphdomain "github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

type Anchor struct {
	Text   string  `json:"text"`
	Source string  `json:"source"`
	Weight float64 `json:"weight"`
}

type Entity struct {
	ID                string   `json:"id"`
	SiteID            string   `json:"siteId"`
	Name              string   `json:"name"`
	Kind              string   `json:"kind"`
	Intent            string   `json:"intent"`
	PrimaryKeyword    string   `json:"primaryKeyword"`
	SecondaryKeywords []string `json:"secondaryKeywords"`
	Anchors           []Anchor `json:"anchors"`
	CanonicalPageID   *string  `json:"canonicalPageId"`
	Score             float64  `json:"score"`
	Source            string   `json:"source"`
	CreatedAt         dto.Time `json:"createdAt"`
	UpdatedAt         dto.Time `json:"updatedAt"`
}

type Edge struct {
	ID           string   `json:"id"`
	SiteID       string   `json:"siteId"`
	FromEntityID string   `json:"fromEntityId"`
	ToEntityID   string   `json:"toEntityId"`
	Kind         string   `json:"kind"`
	Weight       float64  `json:"weight"`
	Source       string   `json:"source"`
	Status       string   `json:"status"`
	CreatedAt    dto.Time `json:"createdAt"`
}

func entityView(e graphdomain.Entity) Entity {
	keywords := e.SecondaryKeywords
	if keywords == nil {
		keywords = []string{}
	}
	anchors := make([]Anchor, 0, len(e.Anchors))
	for _, anchor := range e.Anchors {
		anchors = append(anchors, Anchor{Text: anchor.Text, Source: string(anchor.Source), Weight: anchor.Weight})
	}
	return Entity{
		ID:                e.ID,
		SiteID:            e.SiteID,
		Name:              e.Name,
		Kind:              string(e.Kind),
		Intent:            e.Intent,
		PrimaryKeyword:    e.PrimaryKeyword,
		SecondaryKeywords: keywords,
		Anchors:           anchors,
		CanonicalPageID:   e.CanonicalPageID,
		Score:             e.Score,
		Source:            string(e.Source),
		CreatedAt:         dto.NewTime(e.CreatedAt),
		UpdatedAt:         dto.NewTime(e.UpdatedAt),
	}
}

func entityViews(entities []graphdomain.Entity) []Entity {
	out := make([]Entity, 0, len(entities))
	for _, e := range entities {
		out = append(out, entityView(e))
	}
	return out
}

func edgeView(e graphdomain.Edge) Edge {
	return Edge{
		ID:           e.ID,
		SiteID:       e.SiteID,
		FromEntityID: e.FromEntityID,
		ToEntityID:   e.ToEntityID,
		Kind:         string(e.Kind),
		Weight:       e.Weight,
		Source:       string(e.Source),
		Status:       string(e.Status),
		CreatedAt:    dto.NewTime(e.CreatedAt),
	}
}

func edgeViews(edges []graphdomain.Edge) []Edge {
	out := make([]Edge, 0, len(edges))
	for _, e := range edges {
		out = append(out, edgeView(e))
	}
	return out
}

func anchorsOf(anchors []Anchor) []graphdomain.Anchor {
	out := make([]graphdomain.Anchor, 0, len(anchors))
	for _, anchor := range anchors {
		out = append(out, graphdomain.Anchor{Text: anchor.Text, Source: graphdomain.AnchorSource(anchor.Source), Weight: anchor.Weight})
	}
	return out
}
```

`internal/application/graph/requests.go`:

```go
package graph

import "github.com/davidmovas/postulator/internal/kernel/dto"

type CreateEntityRequest struct {
	SiteID            string   `json:"siteId"`
	Name              string   `json:"name"`
	Kind              string   `json:"kind"`
	Intent            string   `json:"intent"`
	PrimaryKeyword    string   `json:"primaryKeyword"`
	SecondaryKeywords []string `json:"secondaryKeywords"`
	Anchors           []Anchor `json:"anchors"`
	Source            string   `json:"source,omitempty"`
}

type CreateEntityResponse struct {
	Entity Entity `json:"entity"`
}

type UpdateEntityRequest struct {
	ID                string    `json:"id"`
	Name              *string   `json:"name,omitempty"`
	Kind              *string   `json:"kind,omitempty"`
	Intent            *string   `json:"intent,omitempty"`
	PrimaryKeyword    *string   `json:"primaryKeyword,omitempty"`
	SecondaryKeywords *[]string `json:"secondaryKeywords,omitempty"`
}

type UpdateEntityResponse struct {
	Entity Entity `json:"entity"`
}

type DeleteEntityRequest struct {
	ID string `json:"id"`
}

type DeleteEntityResponse struct{}

type GetEntityRequest struct {
	ID string `json:"id"`
}

type GetEntityResponse struct {
	Entity Entity `json:"entity"`
}

type ListEntitiesRequest struct {
	dto.ListRequest
	SiteID           string `json:"siteId"`
	Kind             string `json:"kind,omitempty"`
	HasCanonicalPage *bool  `json:"hasCanonicalPage,omitempty"`
	NamePrefix       string `json:"namePrefix,omitempty"`
}

type SetAnchorsRequest struct {
	EntityID string   `json:"entityId"`
	Anchors  []Anchor `json:"anchors"`
}

type SetAnchorsResponse struct {
	Entity Entity `json:"entity"`
}

type AddEdgeRequest struct {
	SiteID       string  `json:"siteId"`
	FromEntityID string  `json:"fromEntityId"`
	ToEntityID   string  `json:"toEntityId"`
	Kind         string  `json:"kind"`
	Weight       float64 `json:"weight"`
	Source       string  `json:"source,omitempty"`
	Status       string  `json:"status,omitempty"`
}

type AddEdgeResponse struct {
	Edge Edge `json:"edge"`
}

type ApproveEdgeRequest struct {
	ID string `json:"id"`
}

type ApproveEdgeResponse struct {
	Edge Edge `json:"edge"`
}

type RejectEdgeRequest struct {
	ID string `json:"id"`
}

type RejectEdgeResponse struct {
	Edge Edge `json:"edge"`
}

type DeleteEdgeRequest struct {
	ID string `json:"id"`
}

type DeleteEdgeResponse struct{}

type ListEdgesRequest struct {
	dto.ListRequest
	SiteID   string `json:"siteId"`
	Kind     string `json:"kind,omitempty"`
	Status   string `json:"status,omitempty"`
	EntityID string `json:"entityId,omitempty"`
}

type LoadGraphRequest struct {
	SiteID string `json:"siteId"`
}

type LoadGraphResponse struct {
	Entities []Entity `json:"entities"`
	Edges    []Edge   `json:"edges"`
}

type RecomputeScoresRequest struct {
	SiteID string `json:"siteId"`
}

type RecomputeScoresResponse struct {
	Scores map[string]float64 `json:"scores"`
}
```

`internal/application/graph/service.go`:

```go
package graph

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/application/events"
	graphdomain "github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

type entityStore interface {
	Insert(ctx context.Context, e graphdomain.Entity) error
	Update(ctx context.Context, e graphdomain.Entity) error
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (graphdomain.Entity, error)
	List(ctx context.Context, q graphdomain.EntityQuery, page paging.Request) (paging.List[graphdomain.Entity], error)
	ListBySite(ctx context.Context, siteID string) ([]graphdomain.Entity, error)
	SetScore(ctx context.Context, id string, score float64) error
}

type edgeStore interface {
	Insert(ctx context.Context, e graphdomain.Edge) error
	Get(ctx context.Context, id string) (graphdomain.Edge, error)
	Delete(ctx context.Context, id string) error
	SetStatus(ctx context.Context, id string, status graphdomain.EdgeStatus) error
	List(ctx context.Context, q graphdomain.EdgeQuery, page paging.Request) (paging.List[graphdomain.Edge], error)
	ListBySite(ctx context.Context, siteID string) ([]graphdomain.Edge, error)
}

type siteReader interface {
	Get(ctx context.Context, id string) (site.Site, error)
}

type unitOfWork interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type Service struct {
	entities  entityStore
	edges     edgeStore
	sites     siteReader
	uow       unitOfWork
	publisher application.Publisher
	clock     clock.Clock
}

func New(entities entityStore, edges edgeStore, sites siteReader, uow unitOfWork, publisher application.Publisher, clk clock.Clock) *Service {
	return &Service{entities: entities, edges: edges, sites: sites, uow: uow, publisher: publisher, clock: clk}
}

func (s *Service) now() time.Time {
	return s.clock.Now().UTC().Truncate(time.Second)
}

func (s *Service) changed(siteID string) error {
	return s.publisher.Publish(events.GraphChanged, events.GraphChangedPayload{SiteID: siteID})
}

func (s *Service) pagesChanged(siteID string) error {
	return s.publisher.Publish(events.PagesChanged, events.PagesChangedPayload{SiteID: siteID})
}

func requireSite(siteID string) error {
	if siteID == "" {
		return errors.New(errors.Invalid, "site id must not be empty").WithDetail("field", "siteId")
	}
	return nil
}
```

`internal/application/graph/entities.go`:

```go
package graph

import (
	"context"

	"github.com/davidmovas/postulator/internal/application"
	graphdomain "github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func entitySort(sort *dto.Sort) (key graphdomain.EntitySort, desc bool, err error) {
	if sort == nil {
		return graphdomain.EntitySortCreatedAt, false, nil
	}
	key = graphdomain.EntitySort(sort.Field)
	if !key.Valid() {
		return "", false, errors.New(errors.Invalid, "entities cannot be sorted by this field").WithDetail("field", "sort.field")
	}
	return key, sort.Desc, nil
}

func (s *Service) CreateEntity(ctx context.Context, req CreateEntityRequest) (CreateEntityResponse, error) {
	if err := requireSite(req.SiteID); err != nil {
		return CreateEntityResponse{}, err
	}
	if _, err := s.sites.Get(ctx, req.SiteID); err != nil {
		return CreateEntityResponse{}, err
	}

	source := graphdomain.Source(req.Source)
	if req.Source == "" {
		source = graphdomain.SourceUser
	}
	now := s.now()
	entity, err := graphdomain.NewEntity(graphdomain.Entity{
		ID:                id.New(),
		SiteID:            req.SiteID,
		Name:              req.Name,
		Kind:              graphdomain.Kind(req.Kind),
		Intent:            req.Intent,
		PrimaryKeyword:    req.PrimaryKeyword,
		SecondaryKeywords: req.SecondaryKeywords,
		Anchors:           anchorsOf(req.Anchors),
		Source:            source,
		CreatedAt:         now,
		UpdatedAt:         now,
	})
	if err != nil {
		return CreateEntityResponse{}, err
	}

	if err = s.uow.Do(ctx, func(c context.Context) error { return s.entities.Insert(c, entity) }); err != nil {
		return CreateEntityResponse{}, err
	}
	if err = s.changed(entity.SiteID); err != nil {
		return CreateEntityResponse{}, err
	}
	return CreateEntityResponse{Entity: entityView(entity)}, nil
}

func (s *Service) UpdateEntity(ctx context.Context, req UpdateEntityRequest) (UpdateEntityResponse, error) {
	updated, err := s.rewriteEntity(ctx, req.ID, func(next *graphdomain.Entity) {
		if req.Name != nil {
			next.Name = *req.Name
		}
		if req.Kind != nil {
			next.Kind = graphdomain.Kind(*req.Kind)
		}
		if req.Intent != nil {
			next.Intent = *req.Intent
		}
		if req.PrimaryKeyword != nil {
			next.PrimaryKeyword = *req.PrimaryKeyword
		}
		if req.SecondaryKeywords != nil {
			next.SecondaryKeywords = *req.SecondaryKeywords
		}
	})
	if err != nil {
		return UpdateEntityResponse{}, err
	}
	return UpdateEntityResponse{Entity: entityView(updated)}, nil
}

func (s *Service) SetAnchors(ctx context.Context, req SetAnchorsRequest) (SetAnchorsResponse, error) {
	updated, err := s.rewriteEntity(ctx, req.EntityID, func(next *graphdomain.Entity) {
		next.Anchors = anchorsOf(req.Anchors)
	})
	if err != nil {
		return SetAnchorsResponse{}, err
	}
	return SetAnchorsResponse{Entity: entityView(updated)}, nil
}

func (s *Service) rewriteEntity(ctx context.Context, entityID string, change func(*graphdomain.Entity)) (graphdomain.Entity, error) {
	var updated graphdomain.Entity
	err := s.uow.Do(ctx, func(c context.Context) error {
		current, getErr := s.entities.Get(c, entityID)
		if getErr != nil {
			return getErr
		}
		next := current
		change(&next)
		next.UpdatedAt = s.now()
		valid, validErr := graphdomain.NewEntity(next)
		if validErr != nil {
			return validErr
		}
		if updateErr := s.entities.Update(c, valid); updateErr != nil {
			return updateErr
		}
		updated = valid
		return nil
	})
	if err != nil {
		return graphdomain.Entity{}, err
	}
	if err = s.changed(updated.SiteID); err != nil {
		return graphdomain.Entity{}, err
	}
	return updated, nil
}

func (s *Service) DeleteEntity(ctx context.Context, req DeleteEntityRequest) (DeleteEntityResponse, error) {
	var siteID string
	err := s.uow.Do(ctx, func(c context.Context) error {
		current, getErr := s.entities.Get(c, req.ID)
		if getErr != nil {
			return getErr
		}
		siteID = current.SiteID
		return s.entities.Delete(c, req.ID)
	})
	if err != nil {
		return DeleteEntityResponse{}, err
	}
	if err = s.changed(siteID); err != nil {
		return DeleteEntityResponse{}, err
	}
	if err = s.pagesChanged(siteID); err != nil {
		return DeleteEntityResponse{}, err
	}
	return DeleteEntityResponse{}, nil
}

func (s *Service) GetEntity(ctx context.Context, req GetEntityRequest) (GetEntityResponse, error) {
	entity, err := s.entities.Get(ctx, req.ID)
	if err != nil {
		return GetEntityResponse{}, err
	}
	return GetEntityResponse{Entity: entityView(entity)}, nil
}

func (s *Service) ListEntities(ctx context.Context, req ListEntitiesRequest) (paging.List[Entity], error) {
	if err := requireSite(req.SiteID); err != nil {
		return paging.List[Entity]{}, err
	}
	q := graphdomain.EntityQuery{SiteID: req.SiteID, HasCanonicalPage: req.HasCanonicalPage, NamePrefix: req.NamePrefix}
	if req.Kind != "" {
		kind := graphdomain.Kind(req.Kind)
		if !kind.Valid() {
			return paging.List[Entity]{}, errors.New(errors.Invalid, "entity kind is not recognised").WithDetail("field", "kind")
		}
		q.Kind = &kind
	}
	key, desc, err := entitySort(req.Sort)
	if err != nil {
		return paging.List[Entity]{}, err
	}
	q.Sort, q.Desc = key, desc

	list, err := s.entities.List(ctx, q, application.PageRequest(req.ListRequest))
	if err != nil {
		return paging.List[Entity]{}, err
	}
	return application.MapList(list, entityView), nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race ./internal/application/graph/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/application/graph/
git commit -m "feat(application): entity use cases publishing graph.changed once per transaction

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 21: `application/graph` — edges, graph loading and scores

**Files:**
- Create: `internal/application/graph/edges.go`
- Test: `internal/application/graph/edges_test.go`

**Interfaces:**
- Consumes: Task 20's `Service`, stores and views; `graphdomain.New|NewEdge|(Graph).ValidateAcyclic|Score`.
- Produces: `AddEdge`, `ApproveEdge`, `RejectEdge`, `DeleteEdge`, `ListEdges(ctx, ListEdgesRequest) (paging.List[Edge], error)`, `Load(ctx, siteID string) (graphdomain.Graph, error)` (for Go callers in later phases), `LoadGraph(ctx, LoadGraphRequest) (LoadGraphResponse, error)`, `RecomputeScores(ctx, RecomputeScoresRequest) (RecomputeScoresResponse, error)`.

- [ ] **Step 1: Write the failing test**

```go
package graph_test

import (
	stderrors "errors"
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func (h harness) edge(t *testing.T, from, to, kind, status string) graph.Edge {
	t.Helper()
	added, err := h.service.AddEdge(t.Context(), graph.AddEdgeRequest{SiteID: h.siteID, FromEntityID: from, ToEntityID: to, Kind: kind, Weight: 0.7, Status: status})
	if err != nil {
		t.Fatalf("AddEdge %s -> %s: %v", from, to, err)
	}
	return added.Edge
}

func TestAddEdgeAndCycles(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	hub := h.entity(t, "Hub", "hub")
	mid := h.entity(t, "Mid", "topic")
	leaf := h.entity(t, "Leaf", "topic")
	h.recorder.Reset()

	first := h.edge(t, mid.ID, hub.ID, "parent", "")
	if first.Status != "approved" || first.Source != "user" || first.Weight != 1 {
		t.Errorf("AddEdge = %+v", first)
	}
	h.wantEvents(t, events.GraphChanged)
	h.edge(t, leaf.ID, mid.ID, "parent", "approved")
	h.recorder.Reset()

	_, err := h.service.AddEdge(t.Context(), graph.AddEdgeRequest{SiteID: h.siteID, FromEntityID: hub.ID, ToEntityID: leaf.ID, Kind: "parent"})
	if !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("cycle code = %q, want INVALID", errors.CodeOf(err))
	}
	var kernel *errors.Error
	if !stderrors.As(err, &kernel) {
		t.Fatal("not a kernel error")
	}
	cycle, _ := kernel.Details["cycle"].([]string)
	if !slices.Equal(cycle, []string{hub.ID, leaf.ID, mid.ID, hub.ID}) {
		t.Errorf("cycle = %v", cycle)
	}
	if len(h.recorder.Events()) != 0 {
		t.Error("a refused edge must not publish")
	}

	if _, err = h.service.AddEdge(t.Context(), graph.AddEdgeRequest{SiteID: h.siteID, FromEntityID: hub.ID, ToEntityID: leaf.ID, Kind: "parent", Status: "proposed"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("a proposal closing a cycle is refused at once, got %v", err)
	}
	if _, err = h.service.AddEdge(t.Context(), graph.AddEdgeRequest{SiteID: h.siteID, FromEntityID: mid.ID, ToEntityID: hub.ID, Kind: "parent"}); !errors.IsCode(err, errors.Conflict) {
		t.Errorf("duplicate edge code = %q, want CONFLICT", errors.CodeOf(err))
	}
	if _, err = h.service.AddEdge(t.Context(), graph.AddEdgeRequest{SiteID: h.siteID, FromEntityID: mid.ID, ToEntityID: "missing", Kind: "parent"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("unknown endpoint code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.AddEdge(t.Context(), graph.AddEdgeRequest{SiteID: "missing", FromEntityID: mid.ID, ToEntityID: hub.ID, Kind: "parent"}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("unknown site code = %q, want NOT_FOUND", errors.CodeOf(err))
	}

	related := h.edge(t, mid.ID, leaf.ID, "related", "proposed")
	if related.Kind != "related" || related.Weight != 0.7 || related.Status != "proposed" || related.FromEntityID > related.ToEntityID {
		t.Errorf("related edge = %+v", related)
	}
}

func TestApproveRejectDeleteAndList(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	hub := h.entity(t, "Hub", "hub")
	mid := h.entity(t, "Mid", "topic")
	leaf := h.entity(t, "Leaf", "topic")
	h.edge(t, mid.ID, hub.ID, "parent", "approved")
	leafMid := h.edge(t, leaf.ID, mid.ID, "parent", "proposed")
	closing := h.edge(t, hub.ID, leaf.ID, "parent", "proposed")
	h.recorder.Reset()

	pending, err := h.service.ListEdges(t.Context(), graph.ListEdgesRequest{SiteID: h.siteID, Status: "proposed"})
	if err != nil || len(pending.Items) != 2 {
		t.Fatalf("ListEdges proposed = %+v, %v", pending, err)
	}
	approved, err := h.service.ApproveEdge(t.Context(), graph.ApproveEdgeRequest{ID: leafMid.ID})
	if err != nil || approved.Edge.Status != "approved" {
		t.Fatalf("ApproveEdge = %+v, %v", approved, err)
	}
	h.wantEvents(t, events.GraphChanged)

	if _, err = h.service.ApproveEdge(t.Context(), graph.ApproveEdgeRequest{ID: closing.ID}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("approving the closing edge must be refused, got %v", err)
	}
	rejected, err := h.service.RejectEdge(t.Context(), graph.RejectEdgeRequest{ID: closing.ID})
	if err != nil || rejected.Edge.Status != "rejected" {
		t.Fatalf("RejectEdge = %+v, %v", rejected, err)
	}
	h.wantEvents(t, events.GraphChanged)
	if _, err = h.service.RejectEdge(t.Context(), graph.RejectEdgeRequest{ID: closing.ID}); err != nil {
		t.Fatalf("RejectEdge twice: %v", err)
	}
	if len(h.recorder.Events()) != 0 {
		t.Error("an unchanged status must not publish")
	}

	touching, err := h.service.ListEdges(t.Context(), graph.ListEdgesRequest{ListRequest: dto.ListRequest{Limit: 1}, SiteID: h.siteID, EntityID: leaf.ID})
	if err != nil || len(touching.Items) != 1 || !touching.HasMore {
		t.Errorf("ListEdges touching = %+v, %v", touching, err)
	}
	parents, err := h.service.ListEdges(t.Context(), graph.ListEdgesRequest{SiteID: h.siteID, Kind: "parent"})
	if err != nil || len(parents.Items) != 3 {
		t.Errorf("ListEdges parents = %+v, %v", parents, err)
	}
	if _, err = h.service.ListEdges(t.Context(), graph.ListEdgesRequest{SiteID: h.siteID, Kind: "cousin"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad kind code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.ListEdges(t.Context(), graph.ListEdgesRequest{SiteID: h.siteID, Status: "maybe"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad status code = %q, want INVALID", errors.CodeOf(err))
	}

	if _, err = h.service.DeleteEdge(t.Context(), graph.DeleteEdgeRequest{ID: closing.ID}); err != nil {
		t.Fatalf("DeleteEdge: %v", err)
	}
	h.wantEvents(t, events.GraphChanged)
	if _, err = h.service.DeleteEdge(t.Context(), graph.DeleteEdgeRequest{ID: closing.ID}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("DeleteEdge twice code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = h.service.ApproveEdge(t.Context(), graph.ApproveEdgeRequest{ID: "missing"}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("ApproveEdge missing code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}

func TestLoadGraphAndRecomputeScores(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	hub := h.entity(t, "Hub", "hub")
	mid := h.entity(t, "Mid", "topic")
	leaf := h.entity(t, "Leaf", "topic")
	h.edge(t, mid.ID, hub.ID, "parent", "approved")
	h.edge(t, leaf.ID, mid.ID, "parent", "approved")
	h.recorder.Reset()

	loaded, err := h.service.LoadGraph(t.Context(), graph.LoadGraphRequest{SiteID: h.siteID})
	if err != nil || len(loaded.Entities) != 3 || len(loaded.Edges) != 2 || loaded.Entities[0].Name != "Hub" {
		t.Fatalf("LoadGraph = %+v, %v", loaded, err)
	}
	domainGraph, err := h.service.Load(t.Context(), h.siteID)
	if err != nil || len(domainGraph.Roots()) != 1 {
		t.Errorf("Load = %d roots, %v", len(domainGraph.Roots()), err)
	}
	if _, err = h.service.LoadGraph(t.Context(), graph.LoadGraphRequest{SiteID: "missing"}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("unknown site code = %q, want NOT_FOUND", errors.CodeOf(err))
	}

	scored, err := h.service.RecomputeScores(t.Context(), graph.RecomputeScoresRequest{SiteID: h.siteID})
	if err != nil {
		t.Fatalf("RecomputeScores: %v", err)
	}
	if scored.Scores[hub.ID] != 1 || scored.Scores[mid.ID] >= 1 || scored.Scores[leaf.ID] >= scored.Scores[mid.ID] {
		t.Errorf("scores = %v", scored.Scores)
	}
	h.wantEvents(t, events.GraphChanged)

	got, err := h.service.GetEntity(t.Context(), graph.GetEntityRequest{ID: hub.ID})
	if err != nil || got.Entity.Score != 1 {
		t.Errorf("stored hub score = %v, %v", got.Entity.Score, err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/application/graph/ -run 'TestAddEdge|TestApprove|TestLoadGraph' -v`
Expected: FAIL, `AddEdge` and the other edge methods are undefined.

- [ ] **Step 3: Write the implementation**

`internal/application/graph/edges.go`:

```go
package graph

import (
	"context"

	"github.com/davidmovas/postulator/internal/application"
	graphdomain "github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func (s *Service) Load(ctx context.Context, siteID string) (graphdomain.Graph, error) {
	if err := requireSite(siteID); err != nil {
		return graphdomain.Graph{}, err
	}
	if _, err := s.sites.Get(ctx, siteID); err != nil {
		return graphdomain.Graph{}, err
	}
	return s.load(ctx, siteID)
}

func (s *Service) load(ctx context.Context, siteID string) (graphdomain.Graph, error) {
	entities, err := s.entities.ListBySite(ctx, siteID)
	if err != nil {
		return graphdomain.Graph{}, err
	}
	edges, err := s.edges.ListBySite(ctx, siteID)
	if err != nil {
		return graphdomain.Graph{}, err
	}
	return graphdomain.New(entities, edges)
}

func (s *Service) LoadGraph(ctx context.Context, req LoadGraphRequest) (LoadGraphResponse, error) {
	g, err := s.Load(ctx, req.SiteID)
	if err != nil {
		return LoadGraphResponse{}, err
	}
	return LoadGraphResponse{Entities: entityViews(g.Entities()), Edges: edgeViews(g.Edges())}, nil
}

func (s *Service) checkAcyclic(ctx context.Context, candidate graphdomain.Edge) error {
	entities, err := s.entities.ListBySite(ctx, candidate.SiteID)
	if err != nil {
		return err
	}
	edges, err := s.edges.ListBySite(ctx, candidate.SiteID)
	if err != nil {
		return err
	}

	candidate.Status = graphdomain.StatusApproved
	merged := make([]graphdomain.Edge, 0, len(edges)+1)
	for _, e := range edges {
		if e.ID != candidate.ID {
			merged = append(merged, e)
		}
	}
	merged = append(merged, candidate)

	g, err := graphdomain.New(entities, merged)
	if err != nil {
		return err
	}
	if candidate.Kind != graphdomain.EdgeParent {
		return nil
	}
	return g.ValidateAcyclic()
}

func (s *Service) AddEdge(ctx context.Context, req AddEdgeRequest) (AddEdgeResponse, error) {
	if err := requireSite(req.SiteID); err != nil {
		return AddEdgeResponse{}, err
	}
	if _, err := s.sites.Get(ctx, req.SiteID); err != nil {
		return AddEdgeResponse{}, err
	}

	source := graphdomain.Source(req.Source)
	if req.Source == "" {
		source = graphdomain.SourceUser
	}
	status := graphdomain.EdgeStatus(req.Status)
	if req.Status == "" {
		status = graphdomain.StatusApproved
	}
	edge, err := graphdomain.NewEdge(graphdomain.Edge{
		ID:           id.New(),
		SiteID:       req.SiteID,
		FromEntityID: req.FromEntityID,
		ToEntityID:   req.ToEntityID,
		Kind:         graphdomain.EdgeKind(req.Kind),
		Weight:       req.Weight,
		Source:       source,
		Status:       status,
		CreatedAt:    s.now(),
	})
	if err != nil {
		return AddEdgeResponse{}, err
	}

	err = s.uow.Do(ctx, func(c context.Context) error {
		if checkErr := s.checkAcyclic(c, edge); checkErr != nil {
			return checkErr
		}
		return s.edges.Insert(c, edge)
	})
	if err != nil {
		return AddEdgeResponse{}, err
	}
	if err = s.changed(edge.SiteID); err != nil {
		return AddEdgeResponse{}, err
	}
	return AddEdgeResponse{Edge: edgeView(edge)}, nil
}

func (s *Service) setStatus(ctx context.Context, edgeID string, status graphdomain.EdgeStatus) (graphdomain.Edge, error) {
	var (
		edge    graphdomain.Edge
		changed bool
	)
	err := s.uow.Do(ctx, func(c context.Context) error {
		current, getErr := s.edges.Get(c, edgeID)
		if getErr != nil {
			return getErr
		}
		edge = current
		if current.Status == status {
			return nil
		}
		if status == graphdomain.StatusApproved {
			if checkErr := s.checkAcyclic(c, current); checkErr != nil {
				return checkErr
			}
		}
		if setErr := s.edges.SetStatus(c, edgeID, status); setErr != nil {
			return setErr
		}
		edge.Status = status
		changed = true
		return nil
	})
	if err != nil {
		return graphdomain.Edge{}, err
	}
	if changed {
		if err = s.changed(edge.SiteID); err != nil {
			return graphdomain.Edge{}, err
		}
	}
	return edge, nil
}

func (s *Service) ApproveEdge(ctx context.Context, req ApproveEdgeRequest) (ApproveEdgeResponse, error) {
	edge, err := s.setStatus(ctx, req.ID, graphdomain.StatusApproved)
	if err != nil {
		return ApproveEdgeResponse{}, err
	}
	return ApproveEdgeResponse{Edge: edgeView(edge)}, nil
}

func (s *Service) RejectEdge(ctx context.Context, req RejectEdgeRequest) (RejectEdgeResponse, error) {
	edge, err := s.setStatus(ctx, req.ID, graphdomain.StatusRejected)
	if err != nil {
		return RejectEdgeResponse{}, err
	}
	return RejectEdgeResponse{Edge: edgeView(edge)}, nil
}

func (s *Service) DeleteEdge(ctx context.Context, req DeleteEdgeRequest) (DeleteEdgeResponse, error) {
	var siteID string
	err := s.uow.Do(ctx, func(c context.Context) error {
		current, getErr := s.edges.Get(c, req.ID)
		if getErr != nil {
			return getErr
		}
		siteID = current.SiteID
		return s.edges.Delete(c, req.ID)
	})
	if err != nil {
		return DeleteEdgeResponse{}, err
	}
	if err = s.changed(siteID); err != nil {
		return DeleteEdgeResponse{}, err
	}
	return DeleteEdgeResponse{}, nil
}

func (s *Service) ListEdges(ctx context.Context, req ListEdgesRequest) (paging.List[Edge], error) {
	if err := requireSite(req.SiteID); err != nil {
		return paging.List[Edge]{}, err
	}
	q := graphdomain.EdgeQuery{SiteID: req.SiteID, EntityID: req.EntityID}
	if req.Kind != "" {
		kind := graphdomain.EdgeKind(req.Kind)
		if !kind.Valid() {
			return paging.List[Edge]{}, errors.New(errors.Invalid, "edge kind is not recognised").WithDetail("field", "kind")
		}
		q.Kind = &kind
	}
	if req.Status != "" {
		status := graphdomain.EdgeStatus(req.Status)
		if !status.Valid() {
			return paging.List[Edge]{}, errors.New(errors.Invalid, "edge status is not recognised").WithDetail("field", "status")
		}
		q.Status = &status
	}
	if req.Sort != nil {
		if req.Sort.Field != string(graphdomain.EntitySortCreatedAt) {
			return paging.List[Edge]{}, errors.New(errors.Invalid, "edges are sorted by creation time only").WithDetail("field", "sort.field")
		}
		q.Desc = req.Sort.Desc
	}

	list, err := s.edges.List(ctx, q, application.PageRequest(req.ListRequest))
	if err != nil {
		return paging.List[Edge]{}, err
	}
	return application.MapList(list, edgeView), nil
}

func (s *Service) RecomputeScores(ctx context.Context, req RecomputeScoresRequest) (RecomputeScoresResponse, error) {
	if err := requireSite(req.SiteID); err != nil {
		return RecomputeScoresResponse{}, err
	}
	if _, err := s.sites.Get(ctx, req.SiteID); err != nil {
		return RecomputeScoresResponse{}, err
	}

	var scores map[string]float64
	err := s.uow.Do(ctx, func(c context.Context) error {
		g, loadErr := s.load(c, req.SiteID)
		if loadErr != nil {
			return loadErr
		}
		scores = g.Score()
		for entityID, score := range scores {
			if setErr := s.entities.SetScore(c, entityID, score); setErr != nil {
				return setErr
			}
		}
		return nil
	})
	if err != nil {
		return RecomputeScoresResponse{}, err
	}
	if err = s.changed(req.SiteID); err != nil {
		return RecomputeScoresResponse{}, err
	}
	return RecomputeScoresResponse{Scores: scores}, nil
}
```

In `TestAddEdgeAndCycles` the cycle detail is `[hub, leaf, mid, hub]`: the entities are named Hub, Leaf, Mid, so `ordered` visits Hub first, Hub's parent (the candidate) is Leaf, Leaf's parent is Mid, Mid's parent is Hub.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race -cover ./internal/application/graph/ -v`
Expected: PASS, coverage ≥ 85%.

- [ ] **Step 5: Commit**

```bash
git add internal/application/graph/edges.go internal/application/graph/edges_test.go
git commit -m "feat(application): edge use cases with acyclic validation, graph loading and scores

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---
### Task 22: `application/pages` — pages, mapping, canonical, tree

**Files:**
- Create: `internal/application/pages/service.go`, `internal/application/pages/views.go`, `internal/application/pages/requests.go`, `internal/application/pages/pages.go`
- Test: `internal/application/pages/pages_test.go`

**Interfaces:**
- Consumes: `pagemap.*` (Tasks 6–7), `graph.Entity|Graph|New`, `site.Site`, `application.Publisher|PageRequest|MapList`, `events.PagesChanged|GraphChanged`, `id.New`, `clock.Clock`, `dto.*`, `paging.*`, `errors.*`; test-side `sqlite.NewPageRepo|NewPageLinkRepo|NewEntityRepo|NewSiteRepo`, `sqlitetest.Open|Site|Entity`, `applicationtest.Recorder`.
- Produces: `pages.Service` with `New(pages pageStore, links linkStore, entities entityStore, sites siteReader, uow unitOfWork, publisher application.Publisher, clk clock.Clock) *Service`; views `Page`, `PageLink`, `TreeNode`, `Conflict`; methods `Create`, `Update`, `Delete`, `Get`, `List(ctx, ListRequest) (paging.List[Page], error)`, `MapToEntity`, `Unmap`, `SetCanonical`, `Tree`. Task 23 adds `ReplaceLinks`.

- [ ] **Step 1: Write the failing test**

```go
package pages_test

import (
	"encoding/json"
	stderrors "errors"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/applicationtest"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type harness struct {
	service  *pages.Service
	store    *sqlite.Store
	recorder *applicationtest.Recorder
	clock    *clock.Fake
	siteID   string
}

func newHarness(t *testing.T) harness {
	t.Helper()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	recorder := &applicationtest.Recorder{}
	clk := clock.NewFake(time.Date(2026, time.September, 18, 9, 0, 0, 0, time.UTC))
	return harness{
		service:  pages.New(sqlite.NewPageRepo(store), sqlite.NewPageLinkRepo(store), sqlite.NewEntityRepo(store), sqlite.NewSiteRepo(store), store, recorder, clk),
		store:    store,
		recorder: recorder,
		clock:    clk,
		siteID:   owner.ID,
	}
}

func ptr[T any](v T) *T {
	return &v
}

func (h harness) page(t *testing.T, path string, entityID *string) pages.Page {
	t.Helper()
	created, err := h.service.Create(t.Context(), pages.CreateRequest{SiteID: h.siteID, Path: path, Title: path, EntityID: entityID})
	if err != nil {
		t.Fatalf("Create %s: %v", path, err)
	}
	return created.Page
}

func (h harness) entity(t *testing.T, name string) graph.Entity {
	t.Helper()
	return sqlitetest.Entity(t, h.store, h.siteID, name)
}

func (h harness) wantEvents(t *testing.T, want ...events.Type) {
	t.Helper()
	got := h.recorder.Events()
	if len(got) != len(want) {
		t.Fatalf("published %d events, want %d: %+v", len(got), len(want), got)
	}
	for i, event := range got {
		if event.Type != want[i] {
			t.Errorf("event[%d] = %s, want %s", i, event.Type, want[i])
		}
	}
	h.recorder.Reset()
}

func evidenceOf(t *testing.T, err error) []pages.Conflict {
	t.Helper()
	var kernel *errors.Error
	if !stderrors.As(err, &kernel) {
		t.Fatalf("error %v is not a kernel error", err)
	}
	evidence, _ := kernel.Details["evidence"].([]pages.Conflict)
	return evidence
}

func TestCreateResolvesTheParentAndAdoptsChildren(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	child := h.page(t, "/Shop/Shoes/", nil)
	if child.Path != "/shop/shoes/" || child.Slug != "shoes" || child.ParentPageID != nil || child.Status != "planned" || child.WPType != "page" {
		t.Errorf("child = %+v", child)
	}
	h.wantEvents(t, events.PagesChanged)

	parent := h.page(t, "/shop/", nil)
	h.wantEvents(t, events.PagesChanged)
	adopted, err := h.service.Get(t.Context(), pages.GetRequest{ID: child.ID})
	if err != nil || adopted.Page.ParentPageID == nil || *adopted.Page.ParentPageID != parent.ID {
		t.Errorf("child after parent creation = %+v, %v", adopted.Page, err)
	}

	sibling := h.page(t, "/shop/bags/", nil)
	if sibling.ParentPageID == nil || *sibling.ParentPageID != parent.ID {
		t.Errorf("sibling parent = %v, want %s", sibling.ParentPageID, parent.ID)
	}

	encoded, err := json.Marshal(sibling)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	for _, key := range []string{`"siteId"`, `"parentPageId":"` + parent.ID, `"wpType":"page"`, `"wpId":null`, `"entityId":null`, `"wpModifiedAt":null`, `"createdAt":"2026-09-18T09:00:00Z"`} {
		if !strings.Contains(string(encoded), key) {
			t.Errorf("view lacks %s: %s", key, encoded)
		}
	}

	if _, err = h.service.Create(t.Context(), pages.CreateRequest{SiteID: "missing", Path: "/x/"}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("unknown site code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = h.service.Create(t.Context(), pages.CreateRequest{SiteID: h.siteID, Path: "/a b/"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad path code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.Create(t.Context(), pages.CreateRequest{SiteID: h.siteID, Path: "/x/", Status: "lost"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad status code = %q, want INVALID", errors.CodeOf(err))
	}
	if len(h.recorder.Events()) != 1 {
		t.Errorf("only the sibling creation should have published, got %+v", h.recorder.Events())
	}
}

func TestCreateRefusesCannibalization(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	shoes := h.entity(t, "Shoes")
	boots := h.entity(t, "Boots")
	canonical := h.page(t, "/shoes/", &shoes.ID)
	if _, err := h.service.SetCanonical(t.Context(), pages.SetCanonicalRequest{EntityID: shoes.ID, PageID: canonical.ID}); err != nil {
		t.Fatalf("SetCanonical: %v", err)
	}
	h.recorder.Reset()

	_, err := h.service.Create(t.Context(), pages.CreateRequest{SiteID: h.siteID, Path: "/Shoes"})
	if !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("duplicate path code = %q, want CONFLICT", errors.CodeOf(err))
	}
	evidence := evidenceOf(t, err)
	if len(evidence) != 1 || evidence[0].Reason != "path_conflict" || evidence[0].PageID != canonical.ID || evidence[0].Path != "/shoes/" {
		t.Errorf("evidence = %+v", evidence)
	}

	_, err = h.service.Create(t.Context(), pages.CreateRequest{SiteID: h.siteID, Path: "/trainers/", EntityID: &shoes.ID})
	if !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("second page for a canonical entity code = %q, want CONFLICT", errors.CodeOf(err))
	}
	evidence = evidenceOf(t, err)
	if len(evidence) != 1 || evidence[0].Reason != "same_entity_canonical" || evidence[0].EntityID != shoes.ID {
		t.Errorf("evidence = %+v", evidence)
	}

	rival := h.entity(t, "Rival")
	rival.PrimaryKeyword = "SHOES"
	if err = sqlite.NewEntityRepo(h.store).Update(t.Context(), rival); err != nil {
		t.Fatalf("give the rival the same keyword: %v", err)
	}
	_, err = h.service.Create(t.Context(), pages.CreateRequest{SiteID: h.siteID, Path: "/rival/", EntityID: &rival.ID})
	if !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("same keyword code = %q, want CONFLICT", errors.CodeOf(err))
	}
	if evidence = evidenceOf(t, err); len(evidence) != 1 || evidence[0].Reason != "same_primary_keyword" || evidence[0].EntityID != shoes.ID {
		t.Errorf("evidence = %+v", evidence)
	}

	if _, err = h.service.Create(t.Context(), pages.CreateRequest{SiteID: h.siteID, Path: "/boots/", EntityID: &boots.ID}); err != nil {
		t.Errorf("a distinct entity with a free path is allowed: %v", err)
	}
	if len(h.recorder.Events()) != 1 {
		t.Errorf("refused creations must not publish, got %+v", h.recorder.Events())
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	parent := h.page(t, "/shop/", nil)
	child := h.page(t, "/shop/shoes/", nil)
	other := h.page(t, "/blog/", nil)
	h.recorder.Reset()
	h.clock.Advance(time.Minute)

	updated, err := h.service.Update(t.Context(), pages.UpdateRequest{ID: child.ID, Title: ptr("Shoes"), Status: ptr("exists"), TemplateID: ptr("")})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Page.Title != "Shoes" || updated.Page.Status != "exists" || updated.Page.Path != "/shop/shoes/" || updated.Page.UpdatedAt.String() != "2026-09-18T09:01:00Z" {
		t.Errorf("Update = %+v", updated.Page)
	}
	h.wantEvents(t, events.PagesChanged)

	if _, err = h.service.Update(t.Context(), pages.UpdateRequest{ID: parent.ID, Path: ptr("/store/")}); !errors.IsCode(err, errors.Conflict) {
		t.Errorf("renaming a page with descendants code = %q, want CONFLICT", errors.CodeOf(err))
	}
	moved, err := h.service.Update(t.Context(), pages.UpdateRequest{ID: child.ID, Path: ptr("/blog/shoes/")})
	if err != nil {
		t.Fatalf("Update path: %v", err)
	}
	if moved.Page.Path != "/blog/shoes/" || moved.Page.Slug != "shoes" || moved.Page.ParentPageID == nil || *moved.Page.ParentPageID != other.ID {
		t.Errorf("moved = %+v", moved.Page)
	}
	if _, err = h.service.Update(t.Context(), pages.UpdateRequest{ID: child.ID, Path: ptr("/blog/")}); !errors.IsCode(err, errors.Conflict) {
		t.Errorf("moving onto a taken path code = %q, want CONFLICT", errors.CodeOf(err))
	}
	if _, err = h.service.Update(t.Context(), pages.UpdateRequest{ID: child.ID, WPType: ptr("widget")}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad wp type code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.Update(t.Context(), pages.UpdateRequest{ID: "missing", Title: ptr("x")}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("missing page code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}

func TestMapUnmapAndCanonical(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	shoes := h.entity(t, "Shoes")
	boots := h.entity(t, "Boots")
	page := h.page(t, "/shoes/", nil)
	entities := sqlite.NewEntityRepo(h.store)
	h.recorder.Reset()

	mapped, err := h.service.MapToEntity(t.Context(), pages.MapToEntityRequest{PageID: page.ID, EntityID: shoes.ID})
	if err != nil || mapped.Page.EntityID == nil || *mapped.Page.EntityID != shoes.ID {
		t.Fatalf("MapToEntity = %+v, %v", mapped.Page, err)
	}
	h.wantEvents(t, events.PagesChanged)

	if _, err = h.service.SetCanonical(t.Context(), pages.SetCanonicalRequest{EntityID: boots.ID, PageID: page.ID}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("canonical for a page mapped elsewhere code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.SetCanonical(t.Context(), pages.SetCanonicalRequest{EntityID: shoes.ID, PageID: page.ID}); err != nil {
		t.Fatalf("SetCanonical: %v", err)
	}
	h.wantEvents(t, events.GraphChanged, events.PagesChanged)
	stored, err := entities.Get(t.Context(), shoes.ID)
	if err != nil || stored.CanonicalPageID == nil || *stored.CanonicalPageID != page.ID {
		t.Errorf("canonical after SetCanonical = %v, %v", stored.CanonicalPageID, err)
	}

	second := h.page(t, "/boots/", nil)
	h.recorder.Reset()
	if _, err = h.service.SetCanonical(t.Context(), pages.SetCanonicalRequest{EntityID: boots.ID, PageID: second.ID}); err != nil {
		t.Fatalf("SetCanonical on an unmapped page must map it: %v", err)
	}
	h.wantEvents(t, events.GraphChanged, events.PagesChanged)
	got, err := h.service.Get(t.Context(), pages.GetRequest{ID: second.ID})
	if err != nil || got.Page.EntityID == nil || *got.Page.EntityID != boots.ID {
		t.Errorf("auto-mapped page = %+v, %v", got.Page, err)
	}

	unmapped, err := h.service.Unmap(t.Context(), pages.UnmapRequest{PageID: page.ID})
	if err != nil || unmapped.Page.EntityID != nil {
		t.Fatalf("Unmap = %+v, %v", unmapped.Page, err)
	}
	h.wantEvents(t, events.PagesChanged, events.GraphChanged)
	stored, err = entities.Get(t.Context(), shoes.ID)
	if err != nil || stored.CanonicalPageID != nil {
		t.Errorf("canonical after Unmap = %v, %v", stored.CanonicalPageID, err)
	}
	if _, err = h.service.Unmap(t.Context(), pages.UnmapRequest{PageID: page.ID}); err != nil {
		t.Fatalf("Unmap twice: %v", err)
	}
	if len(h.recorder.Events()) != 0 {
		t.Error("an idempotent unmap must not publish")
	}

	foreignSite := sqlitetest.Site(t, h.store, "blog")
	foreign := sqlitetest.Entity(t, h.store, foreignSite.ID, "Elsewhere")
	if _, err = h.service.MapToEntity(t.Context(), pages.MapToEntityRequest{PageID: page.ID, EntityID: foreign.ID}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("mapping across sites code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.MapToEntity(t.Context(), pages.MapToEntityRequest{PageID: page.ID, EntityID: "missing"}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("unknown entity code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}

func TestListAndTree(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	shoes := h.entity(t, "Shoes")
	h.page(t, "/shop/", nil)
	h.clock.Advance(time.Minute)
	h.page(t, "/shop/shoes/", &shoes.ID)
	h.clock.Advance(time.Minute)
	h.page(t, "/blog/", nil)

	byPath, err := h.service.List(t.Context(), pages.ListRequest{ListRequest: dto.ListRequest{Limit: 2, Sort: &dto.Sort{Field: "path"}}, SiteID: h.siteID})
	if err != nil || len(byPath.Items) != 2 || byPath.Items[0].Path != "/blog/" || !byPath.HasMore {
		t.Fatalf("List by path = %+v, %v", byPath, err)
	}
	rest, err := h.service.List(t.Context(), pages.ListRequest{ListRequest: dto.ListRequest{Cursor: string(byPath.Next), Limit: 2, Sort: &dto.Sort{Field: "path"}}, SiteID: h.siteID})
	if err != nil || len(rest.Items) != 1 || rest.Items[0].Path != "/shop/shoes/" {
		t.Errorf("List after = %+v, %v", rest, err)
	}
	unmapped, err := h.service.List(t.Context(), pages.ListRequest{SiteID: h.siteID, Unmapped: true})
	if err != nil || len(unmapped.Items) != 2 {
		t.Errorf("unmapped = %+v, %v", unmapped, err)
	}
	byEntity, err := h.service.List(t.Context(), pages.ListRequest{SiteID: h.siteID, EntityID: shoes.ID})
	if err != nil || len(byEntity.Items) != 1 {
		t.Errorf("by entity = %+v, %v", byEntity, err)
	}
	prefixed, err := h.service.List(t.Context(), pages.ListRequest{SiteID: h.siteID, PathPrefix: "/shop/", Status: "planned"})
	if err != nil || len(prefixed.Items) != 2 {
		t.Errorf("prefixed = %+v, %v", prefixed, err)
	}
	if _, err = h.service.List(t.Context(), pages.ListRequest{SiteID: h.siteID, Status: "lost"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad status code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.List(t.Context(), pages.ListRequest{ListRequest: dto.ListRequest{Sort: &dto.Sort{Field: "title"}}, SiteID: h.siteID}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad sort code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.List(t.Context(), pages.ListRequest{}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("missing site code = %q, want INVALID", errors.CodeOf(err))
	}

	tree, err := h.service.Tree(t.Context(), pages.TreeRequest{SiteID: h.siteID})
	if err != nil || len(tree.Roots) != 2 || tree.Roots[1].Page.Path != "/shop/" || len(tree.Roots[1].Children) != 1 {
		t.Errorf("Tree = %+v, %v", tree, err)
	}
	if _, err = h.service.Tree(t.Context(), pages.TreeRequest{SiteID: "missing"}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("unknown site code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	shoes := h.entity(t, "Shoes")
	page := h.page(t, "/shoes/", nil)
	if _, err := h.service.SetCanonical(t.Context(), pages.SetCanonicalRequest{EntityID: shoes.ID, PageID: page.ID}); err != nil {
		t.Fatalf("SetCanonical: %v", err)
	}
	h.recorder.Reset()

	if _, err := h.service.Delete(t.Context(), pages.DeleteRequest{ID: page.ID}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	h.wantEvents(t, events.PagesChanged, events.GraphChanged)
	stored, err := sqlite.NewEntityRepo(h.store).Get(t.Context(), shoes.ID)
	if err != nil || stored.CanonicalPageID != nil {
		t.Errorf("canonical after page delete = %v, %v", stored.CanonicalPageID, err)
	}
	if _, err = h.service.Delete(t.Context(), pages.DeleteRequest{ID: page.ID}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Delete twice code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/application/pages/ -v`
Expected: FAIL, the package does not exist.

- [ ] **Step 3: Write the implementation**

`internal/application/pages/views.go`:

```go
package pages

import (
	"time"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

type Page struct {
	ID              string   `json:"id"`
	SiteID          string   `json:"siteId"`
	Path            string   `json:"path"`
	Slug            string   `json:"slug"`
	ParentPageID    *string  `json:"parentPageId"`
	WPType          string   `json:"wpType"`
	WPID            *int64   `json:"wpId"`
	Title           string   `json:"title"`
	H1              string   `json:"h1"`
	MetaTitle       string   `json:"metaTitle"`
	MetaDescription string   `json:"metaDescription"`
	Canonical       string   `json:"canonical"`
	Status          string   `json:"status"`
	EntityID        *string  `json:"entityId"`
	TemplateID      *string  `json:"templateId"`
	ContentHash     string   `json:"contentHash"`
	WPModifiedAt    dto.Time `json:"wpModifiedAt"`
	LastSyncedAt    dto.Time `json:"lastSyncedAt"`
	Drift           bool     `json:"drift"`
	CreatedAt       dto.Time `json:"createdAt"`
	UpdatedAt       dto.Time `json:"updatedAt"`
}

type PageLink struct {
	ID         string   `json:"id"`
	SiteID     string   `json:"siteId"`
	FromPageID string   `json:"fromPageId"`
	ToPageID   *string  `json:"toPageId"`
	ToURL      string   `json:"toUrl"`
	AnchorText string   `json:"anchorText"`
	Origin     string   `json:"origin"`
	ObservedAt dto.Time `json:"observedAt"`
}

type TreeNode struct {
	Page     Page       `json:"page"`
	Children []TreeNode `json:"children"`
}

type Conflict struct {
	PageID   string `json:"pageId"`
	Path     string `json:"path"`
	Reason   string `json:"reason"`
	EntityID string `json:"entityId,omitempty"`
}

func optionalTime(t *time.Time) dto.Time {
	if t == nil {
		return dto.Time{}
	}
	return dto.NewTime(*t)
}

func view(p pagemap.Page) Page {
	return Page{
		ID:              p.ID,
		SiteID:          p.SiteID,
		Path:            p.Path,
		Slug:            p.Slug,
		ParentPageID:    p.ParentPageID,
		WPType:          string(p.WPType),
		WPID:            p.WPID,
		Title:           p.Title,
		H1:              p.H1,
		MetaTitle:       p.MetaTitle,
		MetaDescription: p.MetaDescription,
		Canonical:       p.Canonical,
		Status:          string(p.Status),
		EntityID:        p.EntityID,
		TemplateID:      p.TemplateID,
		ContentHash:     p.ContentHash,
		WPModifiedAt:    optionalTime(p.WPModifiedAt),
		LastSyncedAt:    optionalTime(p.LastSyncedAt),
		Drift:           p.Drift,
		CreatedAt:       dto.NewTime(p.CreatedAt),
		UpdatedAt:       dto.NewTime(p.UpdatedAt),
	}
}

func linkView(l pagemap.PageLink) PageLink {
	return PageLink{
		ID:         l.ID,
		SiteID:     l.SiteID,
		FromPageID: l.FromPageID,
		ToPageID:   l.ToPageID,
		ToURL:      l.ToURL,
		AnchorText: l.AnchorText,
		Origin:     string(l.Origin),
		ObservedAt: dto.NewTime(l.ObservedAt),
	}
}

func linkViews(links []pagemap.PageLink) []PageLink {
	out := make([]PageLink, 0, len(links))
	for _, l := range links {
		out = append(out, linkView(l))
	}
	return out
}

func nodeViews(nodes []pagemap.Node) []TreeNode {
	out := make([]TreeNode, 0, len(nodes))
	for _, node := range nodes {
		out = append(out, TreeNode{Page: view(node.Page), Children: nodeViews(node.Children)})
	}
	return out
}

func conflicts(evidence []pagemap.Evidence) []Conflict {
	out := make([]Conflict, 0, len(evidence))
	for _, e := range evidence {
		out = append(out, Conflict{PageID: e.PageID, Path: e.Path, Reason: string(e.Reason), EntityID: e.EntityID})
	}
	return out
}
```

`internal/application/pages/requests.go`:

```go
package pages

import "github.com/davidmovas/postulator/internal/kernel/dto"

type CreateRequest struct {
	SiteID          string  `json:"siteId"`
	Path            string  `json:"path"`
	WPType          string  `json:"wpType,omitempty"`
	Title           string  `json:"title"`
	H1              string  `json:"h1"`
	MetaTitle       string  `json:"metaTitle"`
	MetaDescription string  `json:"metaDescription"`
	Canonical       string  `json:"canonical"`
	Status          string  `json:"status,omitempty"`
	EntityID        *string `json:"entityId,omitempty"`
	TemplateID      *string `json:"templateId,omitempty"`
}

type CreateResponse struct {
	Page Page `json:"page"`
}

type UpdateRequest struct {
	ID              string  `json:"id"`
	Path            *string `json:"path,omitempty"`
	WPType          *string `json:"wpType,omitempty"`
	Title           *string `json:"title,omitempty"`
	H1              *string `json:"h1,omitempty"`
	MetaTitle       *string `json:"metaTitle,omitempty"`
	MetaDescription *string `json:"metaDescription,omitempty"`
	Canonical       *string `json:"canonical,omitempty"`
	Status          *string `json:"status,omitempty"`
	TemplateID      *string `json:"templateId,omitempty"`
}

type UpdateResponse struct {
	Page Page `json:"page"`
}

type DeleteRequest struct {
	ID string `json:"id"`
}

type DeleteResponse struct{}

type GetRequest struct {
	ID string `json:"id"`
}

type GetResponse struct {
	Page  Page       `json:"page"`
	Links []PageLink `json:"links"`
}

type ListRequest struct {
	dto.ListRequest
	SiteID     string `json:"siteId"`
	Status     string `json:"status,omitempty"`
	EntityID   string `json:"entityId,omitempty"`
	Unmapped   bool   `json:"unmapped,omitempty"`
	PathPrefix string `json:"pathPrefix,omitempty"`
}

type MapToEntityRequest struct {
	PageID   string `json:"pageId"`
	EntityID string `json:"entityId"`
}

type MapToEntityResponse struct {
	Page Page `json:"page"`
}

type UnmapRequest struct {
	PageID string `json:"pageId"`
}

type UnmapResponse struct {
	Page Page `json:"page"`
}

type SetCanonicalRequest struct {
	EntityID string `json:"entityId"`
	PageID   string `json:"pageId"`
}

type SetCanonicalResponse struct {
	Page Page `json:"page"`
}

type TreeRequest struct {
	SiteID string `json:"siteId"`
}

type TreeResponse struct {
	Roots []TreeNode `json:"roots"`
}

type LinkInput struct {
	ToPageID   *string `json:"toPageId,omitempty"`
	ToURL      string  `json:"toUrl"`
	AnchorText string  `json:"anchorText"`
	Origin     string  `json:"origin,omitempty"`
}

type ReplaceLinksRequest struct {
	PageID string      `json:"pageId"`
	Links  []LinkInput `json:"links"`
}

type ReplaceLinksResponse struct {
	Links []PageLink `json:"links"`
}
```

`internal/application/pages/service.go`:

```go
package pages

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

type pageStore interface {
	Insert(ctx context.Context, p pagemap.Page) error
	Update(ctx context.Context, p pagemap.Page) error
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (pagemap.Page, error)
	List(ctx context.Context, q pagemap.Query, page paging.Request) (paging.List[pagemap.Page], error)
	ListBySite(ctx context.Context, siteID string) ([]pagemap.Page, error)
}

type linkStore interface {
	ReplaceForPage(ctx context.Context, pageID string, links []pagemap.PageLink) error
	ListForPage(ctx context.Context, pageID string) ([]pagemap.PageLink, error)
}

type entityStore interface {
	Get(ctx context.Context, id string) (graph.Entity, error)
	ListBySite(ctx context.Context, siteID string) ([]graph.Entity, error)
	SetCanonicalPage(ctx context.Context, id string, pageID *string, updatedAt time.Time) error
}

type siteReader interface {
	Get(ctx context.Context, id string) (site.Site, error)
}

type unitOfWork interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type Service struct {
	pages     pageStore
	links     linkStore
	entities  entityStore
	sites     siteReader
	uow       unitOfWork
	publisher application.Publisher
	clock     clock.Clock
}

func New(pages pageStore, links linkStore, entities entityStore, sites siteReader, uow unitOfWork, publisher application.Publisher, clk clock.Clock) *Service {
	return &Service{pages: pages, links: links, entities: entities, sites: sites, uow: uow, publisher: publisher, clock: clk}
}

func (s *Service) now() time.Time {
	return s.clock.Now().UTC().Truncate(time.Second)
}

func (s *Service) changed(siteID string) error {
	return s.publisher.Publish(events.PagesChanged, events.PagesChangedPayload{SiteID: siteID})
}

func (s *Service) graphChanged(siteID string) error {
	return s.publisher.Publish(events.GraphChanged, events.GraphChangedPayload{SiteID: siteID})
}

func requireSite(siteID string) error {
	if siteID == "" {
		return errors.New(errors.Invalid, "site id must not be empty").WithDetail("field", "siteId")
	}
	return nil
}

func (s *Service) entityFor(ctx context.Context, page pagemap.Page) (graph.Entity, error) {
	if page.EntityID == nil {
		return graph.Entity{}, nil
	}
	entity, err := s.entities.Get(ctx, *page.EntityID)
	if err != nil {
		return graph.Entity{}, err
	}
	if entity.SiteID != page.SiteID {
		return graph.Entity{}, errors.New(errors.Invalid, "entity belongs to another site").WithDetail("entityId", entity.ID)
	}
	return entity, nil
}

func (s *Service) verdict(ctx context.Context, candidate pagemap.Page, index pagemap.Index, entity graph.Entity) error {
	g := graph.Graph{}
	if entity.ID != "" {
		entities, err := s.entities.ListBySite(ctx, candidate.SiteID)
		if err != nil {
			return err
		}
		built, err := graph.New(entities, nil)
		if err != nil {
			return err
		}
		g = built
	}

	result := pagemap.Cannibalization(candidate, entity, index, g)
	if result.Allowed {
		return nil
	}
	return errors.New(errors.Conflict, "page would cannibalize an existing page").WithDetail("evidence", conflicts(result.Evidence))
}
```

`internal/application/pages/pages.go`:

```go
package pages

import (
	"context"
	"strings"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func (s *Service) Create(ctx context.Context, req CreateRequest) (CreateResponse, error) {
	if err := requireSite(req.SiteID); err != nil {
		return CreateResponse{}, err
	}
	if _, err := s.sites.Get(ctx, req.SiteID); err != nil {
		return CreateResponse{}, err
	}

	status := pagemap.Status(req.Status)
	if req.Status == "" {
		status = pagemap.StatusPlanned
	}
	wpType := pagemap.WPType(req.WPType)
	if req.WPType == "" {
		wpType = pagemap.WPPage
	}
	now := s.now()
	page, err := pagemap.NewPage(pagemap.Page{
		ID:              id.New(),
		SiteID:          req.SiteID,
		Path:            req.Path,
		WPType:          wpType,
		Title:           req.Title,
		H1:              req.H1,
		MetaTitle:       req.MetaTitle,
		MetaDescription: req.MetaDescription,
		Canonical:       req.Canonical,
		Status:          status,
		EntityID:        req.EntityID,
		TemplateID:      req.TemplateID,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if err != nil {
		return CreateResponse{}, err
	}

	err = s.uow.Do(ctx, func(c context.Context) error {
		entity, entityErr := s.entityFor(c, page)
		if entityErr != nil {
			return entityErr
		}
		siblings, listErr := s.pages.ListBySite(c, page.SiteID)
		if listErr != nil {
			return listErr
		}
		index := pagemap.NewIndex(siblings)
		if verdictErr := s.verdict(c, page, index, entity); verdictErr != nil {
			return verdictErr
		}
		if parent, found := index.ByPath(pagemap.ParentPath(page.Path)); found {
			page.ParentPageID = &parent.ID
		}
		if insertErr := s.pages.Insert(c, page); insertErr != nil {
			return insertErr
		}
		return s.adopt(c, page, siblings)
	})
	if err != nil {
		return CreateResponse{}, err
	}
	if err = s.changed(page.SiteID); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse{Page: view(page)}, nil
}

func (s *Service) adopt(ctx context.Context, parent pagemap.Page, siblings []pagemap.Page) error {
	for _, child := range siblings {
		if child.ParentPageID != nil || pagemap.ParentPath(child.Path) != parent.Path {
			continue
		}
		child.ParentPageID = &parent.ID
		child.UpdatedAt = parent.UpdatedAt
		if err := s.pages.Update(ctx, child); err != nil {
			return err
		}
	}
	return nil
}

func applyUpdate(current pagemap.Page, req UpdateRequest) (pagemap.Page, error) {
	next := current
	if req.Path != nil {
		next.Path = *req.Path
	}
	if req.WPType != nil {
		next.WPType = pagemap.WPType(*req.WPType)
	}
	if req.Title != nil {
		next.Title = *req.Title
	}
	if req.H1 != nil {
		next.H1 = *req.H1
	}
	if req.MetaTitle != nil {
		next.MetaTitle = *req.MetaTitle
	}
	if req.MetaDescription != nil {
		next.MetaDescription = *req.MetaDescription
	}
	if req.Canonical != nil {
		next.Canonical = *req.Canonical
	}
	if req.Status != nil {
		next.Status = pagemap.Status(*req.Status)
	}
	if req.TemplateID != nil {
		next.TemplateID = nil
		if *req.TemplateID != "" {
			next.TemplateID = req.TemplateID
		}
	}
	return pagemap.NewPage(next)
}

func countDescendants(path string, pages []pagemap.Page) int {
	count := 0
	for _, p := range pages {
		if p.Path != path && strings.HasPrefix(p.Path, path) {
			count++
		}
	}
	return count
}

func (s *Service) Update(ctx context.Context, req UpdateRequest) (UpdateResponse, error) {
	var updated pagemap.Page
	err := s.uow.Do(ctx, func(c context.Context) error {
		current, getErr := s.pages.Get(c, req.ID)
		if getErr != nil {
			return getErr
		}
		next, applyErr := applyUpdate(current, req)
		if applyErr != nil {
			return applyErr
		}
		next.UpdatedAt = s.now()

		if next.Path != current.Path {
			siblings, listErr := s.pages.ListBySite(c, current.SiteID)
			if listErr != nil {
				return listErr
			}
			if descendants := countDescendants(current.Path, siblings); descendants > 0 {
				return errors.New(errors.Conflict, "a page with descendants cannot change its path").
					WithDetail("pageId", current.ID).WithDetail("descendants", descendants)
			}
			index := pagemap.NewIndex(siblings)
			if verdictErr := s.verdict(c, next, index, graph.Entity{}); verdictErr != nil {
				return verdictErr
			}
			next.ParentPageID = nil
			if parent, found := index.ByPath(pagemap.ParentPath(next.Path)); found && parent.ID != next.ID {
				next.ParentPageID = &parent.ID
			}
		}

		if updateErr := s.pages.Update(c, next); updateErr != nil {
			return updateErr
		}
		updated = next
		return nil
	})
	if err != nil {
		return UpdateResponse{}, err
	}
	if err = s.changed(updated.SiteID); err != nil {
		return UpdateResponse{}, err
	}
	return UpdateResponse{Page: view(updated)}, nil
}

func (s *Service) Delete(ctx context.Context, req DeleteRequest) (DeleteResponse, error) {
	var siteID string
	err := s.uow.Do(ctx, func(c context.Context) error {
		current, getErr := s.pages.Get(c, req.ID)
		if getErr != nil {
			return getErr
		}
		siteID = current.SiteID
		return s.pages.Delete(c, req.ID)
	})
	if err != nil {
		return DeleteResponse{}, err
	}
	if err = s.changed(siteID); err != nil {
		return DeleteResponse{}, err
	}
	if err = s.graphChanged(siteID); err != nil {
		return DeleteResponse{}, err
	}
	return DeleteResponse{}, nil
}

func (s *Service) Get(ctx context.Context, req GetRequest) (GetResponse, error) {
	page, err := s.pages.Get(ctx, req.ID)
	if err != nil {
		return GetResponse{}, err
	}
	links, err := s.links.ListForPage(ctx, page.ID)
	if err != nil {
		return GetResponse{}, err
	}
	return GetResponse{Page: view(page), Links: linkViews(links)}, nil
}

func pageSort(sort *dto.Sort) (key pagemap.Sort, desc bool, err error) {
	if sort == nil {
		return pagemap.SortCreatedAt, false, nil
	}
	key = pagemap.Sort(sort.Field)
	if !key.Valid() {
		return "", false, errors.New(errors.Invalid, "pages cannot be sorted by this field").WithDetail("field", "sort.field")
	}
	return key, sort.Desc, nil
}

func (s *Service) List(ctx context.Context, req ListRequest) (paging.List[Page], error) {
	if err := requireSite(req.SiteID); err != nil {
		return paging.List[Page]{}, err
	}
	q := pagemap.Query{SiteID: req.SiteID, Unmapped: req.Unmapped, PathPrefix: strings.ToLower(req.PathPrefix)}
	if req.Status != "" {
		status := pagemap.Status(req.Status)
		if !status.Valid() {
			return paging.List[Page]{}, errors.New(errors.Invalid, "page status is not recognised").WithDetail("field", "status")
		}
		q.Status = &status
	}
	if req.EntityID != "" {
		entityID := req.EntityID
		q.EntityID = &entityID
	}
	key, desc, err := pageSort(req.Sort)
	if err != nil {
		return paging.List[Page]{}, err
	}
	q.Sort, q.Desc = key, desc

	list, err := s.pages.List(ctx, q, application.PageRequest(req.ListRequest))
	if err != nil {
		return paging.List[Page]{}, err
	}
	return application.MapList(list, view), nil
}

func (s *Service) MapToEntity(ctx context.Context, req MapToEntityRequest) (MapToEntityResponse, error) {
	var updated pagemap.Page
	err := s.uow.Do(ctx, func(c context.Context) error {
		page, getErr := s.pages.Get(c, req.PageID)
		if getErr != nil {
			return getErr
		}
		page.EntityID = &req.EntityID
		entity, entityErr := s.entityFor(c, page)
		if entityErr != nil {
			return entityErr
		}
		siblings, listErr := s.pages.ListBySite(c, page.SiteID)
		if listErr != nil {
			return listErr
		}
		if verdictErr := s.verdict(c, page, pagemap.NewIndex(siblings), entity); verdictErr != nil {
			return verdictErr
		}
		page.UpdatedAt = s.now()
		if updateErr := s.pages.Update(c, page); updateErr != nil {
			return updateErr
		}
		updated = page
		return nil
	})
	if err != nil {
		return MapToEntityResponse{}, err
	}
	if err = s.changed(updated.SiteID); err != nil {
		return MapToEntityResponse{}, err
	}
	return MapToEntityResponse{Page: view(updated)}, nil
}

func (s *Service) Unmap(ctx context.Context, req UnmapRequest) (UnmapResponse, error) {
	var (
		updated          pagemap.Page
		touched          bool
		clearedCanonical bool
	)
	err := s.uow.Do(ctx, func(c context.Context) error {
		page, getErr := s.pages.Get(c, req.PageID)
		if getErr != nil {
			return getErr
		}
		updated = page
		if page.EntityID == nil {
			return nil
		}

		now := s.now()
		entity, entityErr := s.entities.Get(c, *page.EntityID)
		if entityErr != nil && !errors.IsCode(entityErr, errors.NotFound) {
			return entityErr
		}
		if entityErr == nil && entity.CanonicalPageID != nil && *entity.CanonicalPageID == page.ID {
			if clearErr := s.entities.SetCanonicalPage(c, entity.ID, nil, now); clearErr != nil {
				return clearErr
			}
			clearedCanonical = true
		}
		page.EntityID = nil
		page.UpdatedAt = now
		if updateErr := s.pages.Update(c, page); updateErr != nil {
			return updateErr
		}
		updated = page
		touched = true
		return nil
	})
	if err != nil {
		return UnmapResponse{}, err
	}
	if touched {
		if err = s.changed(updated.SiteID); err != nil {
			return UnmapResponse{}, err
		}
	}
	if clearedCanonical {
		if err = s.graphChanged(updated.SiteID); err != nil {
			return UnmapResponse{}, err
		}
	}
	return UnmapResponse{Page: view(updated)}, nil
}

func (s *Service) SetCanonical(ctx context.Context, req SetCanonicalRequest) (SetCanonicalResponse, error) {
	var updated pagemap.Page
	err := s.uow.Do(ctx, func(c context.Context) error {
		entity, entityErr := s.entities.Get(c, req.EntityID)
		if entityErr != nil {
			return entityErr
		}
		page, getErr := s.pages.Get(c, req.PageID)
		if getErr != nil {
			return getErr
		}
		if page.SiteID != entity.SiteID {
			return errors.New(errors.Invalid, "page and entity belong to different sites").WithDetail("pageId", page.ID).WithDetail("entityId", entity.ID)
		}

		now := s.now()
		switch {
		case page.EntityID == nil:
			page.EntityID = &entity.ID
			page.UpdatedAt = now
			if updateErr := s.pages.Update(c, page); updateErr != nil {
				return updateErr
			}
		case *page.EntityID != entity.ID:
			return errors.New(errors.Invalid, "page is mapped to another entity").WithDetail("pageId", page.ID).WithDetail("entityId", *page.EntityID)
		}
		if setErr := s.entities.SetCanonicalPage(c, entity.ID, &page.ID, now); setErr != nil {
			return setErr
		}
		updated = page
		return nil
	})
	if err != nil {
		return SetCanonicalResponse{}, err
	}
	if err = s.graphChanged(updated.SiteID); err != nil {
		return SetCanonicalResponse{}, err
	}
	if err = s.changed(updated.SiteID); err != nil {
		return SetCanonicalResponse{}, err
	}
	return SetCanonicalResponse{Page: view(updated)}, nil
}

func (s *Service) Tree(ctx context.Context, req TreeRequest) (TreeResponse, error) {
	if err := requireSite(req.SiteID); err != nil {
		return TreeResponse{}, err
	}
	if _, err := s.sites.Get(ctx, req.SiteID); err != nil {
		return TreeResponse{}, err
	}
	all, err := s.pages.ListBySite(ctx, req.SiteID)
	if err != nil {
		return TreeResponse{}, err
	}
	return TreeResponse{Roots: nodeViews(pagemap.BuildTree(all))}, nil
}
```

The `MapToEntity` verdict sees the page itself in the index under its own id, so `path_conflict` never fires for the page being mapped; `same_entity_canonical` and `same_primary_keyword` are the checks that matter there.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race ./internal/application/pages/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/application/pages/
git commit -m "feat(application): page use cases with path-derived hierarchy and cannibalization refusal

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 23: `application/pages` — link replacement

**Files:**
- Create: `internal/application/pages/links.go`
- Test: `internal/application/pages/links_test.go`

**Interfaces:**
- Consumes: Task 22's `Service`, `linkStore.ReplaceForPage`, `pagemap.NewPageLink`.
- Produces: `ReplaceLinks(ctx, ReplaceLinksRequest) (ReplaceLinksResponse, error)`.

- [ ] **Step 1: Write the failing test**

```go
package pages_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestReplaceLinks(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	from := h.page(t, "/shop/", nil)
	to := h.page(t, "/shop/shoes/", nil)
	h.recorder.Reset()

	replaced, err := h.service.ReplaceLinks(t.Context(), pages.ReplaceLinksRequest{PageID: from.ID, Links: []pages.LinkInput{
		{ToPageID: &to.ID, ToURL: "/shop/shoes/", AnchorText: "shoes"},
		{ToURL: "https://elsewhere.example.com/", AnchorText: "elsewhere", Origin: "observed"},
	}})
	if err != nil {
		t.Fatalf("ReplaceLinks: %v", err)
	}
	if len(replaced.Links) != 2 || replaced.Links[0].Origin != "generated" || replaced.Links[1].Origin != "observed" || replaced.Links[0].FromPageID != from.ID || replaced.Links[0].ObservedAt.String() != "2026-09-18T09:00:00Z" {
		t.Errorf("ReplaceLinks = %+v", replaced.Links)
	}
	h.wantEvents(t, events.PagesChanged)

	got, err := h.service.Get(t.Context(), pages.GetRequest{ID: from.ID})
	if err != nil || len(got.Links) != 2 {
		t.Errorf("Get links = %+v, %v", got.Links, err)
	}

	foreignSite := sqlitetest.Site(t, h.store, "blog")
	foreign := sqlitetest.Page(t, h.store, foreignSite.ID, "/x/")
	if _, err = h.service.ReplaceLinks(t.Context(), pages.ReplaceLinksRequest{PageID: from.ID, Links: []pages.LinkInput{{ToPageID: &foreign.ID, ToURL: "/x/", AnchorText: "x"}}}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("link to another site code = %q, want INVALID", errors.CodeOf(err))
	}
	missing := "missing"
	if _, err = h.service.ReplaceLinks(t.Context(), pages.ReplaceLinksRequest{PageID: from.ID, Links: []pages.LinkInput{{ToPageID: &missing, ToURL: "/y/", AnchorText: "y"}}}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("link to an unknown page code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = h.service.ReplaceLinks(t.Context(), pages.ReplaceLinksRequest{PageID: from.ID, Links: []pages.LinkInput{{AnchorText: "nowhere"}}}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("link without a target code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.ReplaceLinks(t.Context(), pages.ReplaceLinksRequest{PageID: "missing"}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("unknown page code = %q, want NOT_FOUND", errors.CodeOf(err))
	}

	kept, err := sqlite.NewPageLinkRepo(h.store).ListForPage(t.Context(), from.ID)
	if err != nil || len(kept) != 2 {
		t.Errorf("a refused replacement must leave the previous links untouched, got %d, %v", len(kept), err)
	}
	if len(h.recorder.Events()) != 0 {
		t.Error("refused replacements must not publish")
	}

	cleared, err := h.service.ReplaceLinks(t.Context(), pages.ReplaceLinksRequest{PageID: from.ID})
	if err != nil || len(cleared.Links) != 0 {
		t.Errorf("clearing = %+v, %v", cleared.Links, err)
	}
	h.wantEvents(t, events.PagesChanged)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/application/pages/ -run TestReplaceLinks -v`
Expected: FAIL, `ReplaceLinks` is undefined.

- [ ] **Step 3: Write the implementation**

`internal/application/pages/links.go`:

```go
package pages

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func (s *Service) ReplaceLinks(ctx context.Context, req ReplaceLinksRequest) (ReplaceLinksResponse, error) {
	var (
		stored []pagemap.PageLink
		siteID string
	)
	err := s.uow.Do(ctx, func(c context.Context) error {
		page, getErr := s.pages.Get(c, req.PageID)
		if getErr != nil {
			return getErr
		}
		siteID = page.SiteID
		now := s.now()

		links := make([]pagemap.PageLink, 0, len(req.Links))
		for _, input := range req.Links {
			if input.ToPageID != nil {
				target, targetErr := s.pages.Get(c, *input.ToPageID)
				if targetErr != nil {
					return targetErr
				}
				if target.SiteID != page.SiteID {
					return errors.New(errors.Invalid, "link target belongs to another site").WithDetail("toPageId", target.ID)
				}
			}
			origin := pagemap.LinkOrigin(input.Origin)
			if input.Origin == "" {
				origin = pagemap.OriginGenerated
			}
			link, linkErr := pagemap.NewPageLink(pagemap.PageLink{
				ID:         id.New(),
				SiteID:     page.SiteID,
				FromPageID: page.ID,
				ToPageID:   input.ToPageID,
				ToURL:      input.ToURL,
				AnchorText: input.AnchorText,
				Origin:     origin,
				ObservedAt: now,
			})
			if linkErr != nil {
				return linkErr
			}
			links = append(links, link)
		}

		if replaceErr := s.links.ReplaceForPage(c, page.ID, links); replaceErr != nil {
			return replaceErr
		}
		stored = links
		return nil
	})
	if err != nil {
		return ReplaceLinksResponse{}, err
	}
	if err = s.changed(siteID); err != nil {
		return ReplaceLinksResponse{}, err
	}
	return ReplaceLinksResponse{Links: linkViews(stored)}, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race -cover ./internal/application/pages/ -v`
Expected: PASS, coverage ≥ 85%.

- [ ] **Step 5: Commit**

```bash
git add internal/application/pages/links.go internal/application/pages/links_test.go
git commit -m "feat(application): replace a page's links inside one unit of work

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---
### Task 24: `application/templates` — templates, overrides, resolution, seeding

**Files:**
- Create: `internal/application/templates/service.go`, `internal/application/templates/views.go`, `internal/application/templates/requests.go`, `internal/application/templates/templates.go`, `internal/application/templates/overrides.go`, `internal/application/templates/seed.go`
- Test: `internal/application/templates/templates_test.go`

**Interfaces:**
- Consumes: `template.*` (Tasks 8–10), `pagemap.Page`, `site.Site`, `application.Publisher|PageRequest|MapList`, `events.TemplatesChanged`, `id.New`, `clock.Clock`, `dto.*`, `paging.*`, `errors.*`; test-side `sqlite.NewTemplateRepo|NewLinkPolicyRepo|NewPageRepo|NewSiteRepo`, `sqlitetest.*`, `applicationtest.Recorder`.
- Produces: `templates.Service` with `New(templates templateStore, policies policyStore, pages pageReader, sites siteReader, uow unitOfWork, publisher application.Publisher, clk clock.Clock) *Service`; views `Template`, `Override`, `LinkPolicy`; `EnsureSeeded(ctx) error`; `CreateTemplate`, `UpdateTemplate`, `DeleteTemplate`, `GetTemplate`, `ListTemplates(ctx, ListTemplatesRequest) (paging.List[Template], error)`, `SetOverride`, `DeleteOverride`, `ResolveForPage`; the constant `DefaultPolicyName = "Default"`. Task 25 adds the policy methods.

- [ ] **Step 1: Write the failing test**

```go
package templates_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/applicationtest"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type harness struct {
	service  *templates.Service
	store    *sqlite.Store
	recorder *applicationtest.Recorder
	clock    *clock.Fake
	siteID   string
}

func newHarness(t *testing.T) harness {
	t.Helper()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	recorder := &applicationtest.Recorder{}
	clk := clock.NewFake(time.Date(2026, time.September, 18, 9, 0, 0, 0, time.UTC))
	return harness{
		service:  templates.New(sqlite.NewTemplateRepo(store), sqlite.NewLinkPolicyRepo(store), sqlite.NewPageRepo(store), sqlite.NewSiteRepo(store), store, recorder, clk),
		store:    store,
		recorder: recorder,
		clock:    clk,
		siteID:   owner.ID,
	}
}

func ptr[T any](v T) *T {
	return &v
}

func (h harness) wantEvents(t *testing.T, count int) {
	t.Helper()
	got := h.recorder.Events()
	if len(got) != count {
		t.Fatalf("published %d events, want %d: %+v", len(got), count, got)
	}
	for i, event := range got {
		if event.Type != events.TemplatesChanged {
			t.Errorf("event[%d] = %s, want templates.changed", i, event.Type)
		}
		if _, ok := event.Payload.(events.TemplatesChangedPayload); !ok {
			t.Errorf("event[%d] carries %T", i, event.Payload)
		}
	}
	h.recorder.Reset()
}

func (h harness) hub(t *testing.T) templates.Template {
	t.Helper()
	seed := template.Seed()[3]
	created, err := h.service.CreateTemplate(t.Context(), templates.CreateTemplateRequest{Name: "Hub", PageKind: seed.PageKind, Spec: seed.Spec})
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}
	return created.Template
}

func TestEnsureSeededIsIdempotent(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	for range 2 {
		if err := h.service.EnsureSeeded(t.Context()); err != nil {
			t.Fatalf("EnsureSeeded: %v", err)
		}
	}
	seeded, err := h.service.ListTemplates(t.Context(), templates.ListTemplatesRequest{Scope: "global"})
	if err != nil || len(seeded.Items) != 5 {
		t.Fatalf("seeded templates = %d, %v", len(seeded.Items), err)
	}
	for _, item := range seeded.Items {
		if item.ID == "" || item.Version != 1 || item.CreatedAt.String() != "2026-09-18T09:00:00Z" {
			t.Errorf("seed %s = %+v", item.Name, item)
		}
	}
	if len(h.recorder.Events()) != 0 {
		t.Error("seeding at startup must not publish")
	}
}

func TestTemplateLifecycle(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	hub := h.hub(t)
	if hub.Scope != "global" || hub.SiteID != nil || hub.Version != 1 || len(hub.Spec.Sections) == 0 {
		t.Errorf("CreateTemplate = %+v", hub)
	}
	h.wantEvents(t, 1)

	encoded, err := json.Marshal(hub)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	for _, key := range []string{`"pageKind":"hub"`, `"siteId":null`, `"spec":{"sections":[`, `"linkRules":{"upDepth":1`, `"createdAt":"2026-09-18T09:00:00Z"`} {
		if !strings.Contains(string(encoded), key) {
			t.Errorf("view lacks %s: %s", key, encoded)
		}
	}

	h.clock.Advance(time.Minute)
	spec := hub.Spec
	spec.Tone = "warm"
	updated, err := h.service.UpdateTemplate(t.Context(), templates.UpdateTemplateRequest{ID: hub.ID, Name: ptr("Hub Page"), Spec: &spec})
	if err != nil {
		t.Fatalf("UpdateTemplate: %v", err)
	}
	if updated.Template.Name != "Hub Page" || updated.Template.Version != 2 || updated.Template.Spec.Tone != "warm" || updated.Template.UpdatedAt.String() != "2026-09-18T09:01:00Z" {
		t.Errorf("UpdateTemplate = %+v", updated.Template)
	}
	h.wantEvents(t, 1)

	if _, err = h.service.CreateTemplate(t.Context(), templates.CreateTemplateRequest{Name: "hub page", PageKind: "hub", Spec: spec}); !errors.IsCode(err, errors.Conflict) {
		t.Errorf("duplicate name code = %q, want CONFLICT", errors.CodeOf(err))
	}
	broken := spec
	broken.Sections = nil
	if _, err = h.service.CreateTemplate(t.Context(), templates.CreateTemplateRequest{Name: "Broken", PageKind: "hub", Spec: broken}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("invalid spec code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.CreateTemplate(t.Context(), templates.CreateTemplateRequest{Scope: "site", Name: "Local", PageKind: "hub", Spec: spec}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("site scope without a site code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.CreateTemplate(t.Context(), templates.CreateTemplateRequest{Scope: "site", SiteID: ptr("missing"), Name: "Local", PageKind: "hub", Spec: spec}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("unknown site code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	local, err := h.service.CreateTemplate(t.Context(), templates.CreateTemplateRequest{Scope: "site", SiteID: &h.siteID, Name: "Hub Page", PageKind: "hub", Spec: spec})
	if err != nil || local.Template.SiteID == nil {
		t.Fatalf("site-scoped template = %+v, %v", local.Template, err)
	}
	h.recorder.Reset()

	got, err := h.service.GetTemplate(t.Context(), templates.GetTemplateRequest{ID: hub.ID})
	if err != nil || got.Template.Version != 2 || len(got.Overrides) != 0 {
		t.Errorf("GetTemplate = %+v, %v", got, err)
	}

	bySite, err := h.service.ListTemplates(t.Context(), templates.ListTemplatesRequest{Scope: "site", SiteID: h.siteID})
	if err != nil || len(bySite.Items) != 1 || bySite.Items[0].ID != local.Template.ID {
		t.Errorf("ListTemplates by site = %+v, %v", bySite, err)
	}
	byKind, err := h.service.ListTemplates(t.Context(), templates.ListTemplatesRequest{ListRequest: dto.ListRequest{Sort: &dto.Sort{Field: "name", Desc: true}}, PageKind: "hub"})
	if err != nil || len(byKind.Items) != 2 {
		t.Errorf("ListTemplates by kind = %+v, %v", byKind, err)
	}
	if _, err = h.service.ListTemplates(t.Context(), templates.ListTemplatesRequest{Scope: "galaxy"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad scope code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.ListTemplates(t.Context(), templates.ListTemplatesRequest{ListRequest: dto.ListRequest{Sort: &dto.Sort{Field: "version"}}}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad sort code = %q, want INVALID", errors.CodeOf(err))
	}

	if _, err = h.service.DeleteTemplate(t.Context(), templates.DeleteTemplateRequest{ID: hub.ID}); err != nil {
		t.Fatalf("DeleteTemplate: %v", err)
	}
	h.wantEvents(t, 1)
	if _, err = h.service.DeleteTemplate(t.Context(), templates.DeleteTemplateRequest{ID: hub.ID}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("DeleteTemplate twice code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = h.service.UpdateTemplate(t.Context(), templates.UpdateTemplateRequest{ID: hub.ID, Name: ptr("x")}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("UpdateTemplate missing code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}

func TestOverridesAndResolveForPage(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	hub := h.hub(t)
	page := sqlitetest.Page(t, h.store, h.siteID, "/shop/")
	orphan := sqlitetest.Page(t, h.store, h.siteID, "/orphan/")
	h.recorder.Reset()

	if _, err := h.service.ResolveForPage(t.Context(), templates.ResolveForPageRequest{PageID: page.ID}); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("a page without a template and a site without a default code = %q, want NOT_FOUND", errors.CodeOf(err))
	}

	owner, err := sqlite.NewSiteRepo(h.store).Get(t.Context(), h.siteID)
	if err != nil {
		t.Fatalf("Get site: %v", err)
	}
	owner.Defaults.TemplateID = &hub.ID
	if err = sqlite.NewSiteRepo(h.store).Update(t.Context(), owner); err != nil {
		t.Fatalf("set the site default: %v", err)
	}
	viaDefault, err := h.service.ResolveForPage(t.Context(), templates.ResolveForPageRequest{PageID: orphan.ID})
	if err != nil || viaDefault.TemplateID != hub.ID || viaDefault.Version != 1 || viaDefault.Spec.Tone != hub.Spec.Tone {
		t.Errorf("ResolveForPage via the site default = %+v, %v", viaDefault, err)
	}

	siteOverride, err := h.service.SetOverride(t.Context(), templates.SetOverrideRequest{TemplateID: hub.ID, Scope: "site", TargetID: h.siteID, Patch: json.RawMessage(`{"tone":"warm","linkRules":{"maxLinks":5}}`)})
	if err != nil {
		t.Fatalf("SetOverride site: %v", err)
	}
	if siteOverride.Override.Scope != "site" || siteOverride.Override.TargetID != h.siteID {
		t.Errorf("site override = %+v", siteOverride.Override)
	}
	h.wantEvents(t, 1)

	page.TemplateID = &hub.ID
	if err = sqlite.NewPageRepo(h.store).Update(t.Context(), page); err != nil {
		t.Fatalf("assign the page template: %v", err)
	}
	if _, err = h.service.SetOverride(t.Context(), templates.SetOverrideRequest{TemplateID: hub.ID, Scope: "page", TargetID: page.ID, Patch: json.RawMessage(`{"length":{"min":100,"max":200},"images":{"inline":3}}`)}); err != nil {
		t.Fatalf("SetOverride page: %v", err)
	}
	h.wantEvents(t, 1)

	resolved, err := h.service.ResolveForPage(t.Context(), templates.ResolveForPageRequest{PageID: page.ID})
	if err != nil {
		t.Fatalf("ResolveForPage: %v", err)
	}
	if resolved.Spec.Tone != "warm" || resolved.Spec.LinkRules.MaxLinks != 5 || resolved.Spec.Length.Min != 100 || resolved.Spec.Images.Inline != 3 || !resolved.Spec.Images.Featured {
		t.Errorf("ResolveForPage = %+v", resolved.Spec)
	}

	replaced, err := h.service.SetOverride(t.Context(), templates.SetOverrideRequest{TemplateID: hub.ID, Scope: "site", TargetID: h.siteID, Patch: json.RawMessage(`{"tone":"cold"}`)})
	if err != nil || replaced.Override.ID != siteOverride.Override.ID {
		t.Errorf("replacing an override keeps its id: %+v, %v", replaced.Override, err)
	}
	h.recorder.Reset()

	if _, err = h.service.SetOverride(t.Context(), templates.SetOverrideRequest{TemplateID: hub.ID, Scope: "site", TargetID: h.siteID, Patch: json.RawMessage(`{"sections":null}`)}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("an override that breaks the spec code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.SetOverride(t.Context(), templates.SetOverrideRequest{TemplateID: hub.ID, Scope: "site", TargetID: h.siteID, Patch: json.RawMessage(`[1]`)}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("a non-object patch code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.SetOverride(t.Context(), templates.SetOverrideRequest{TemplateID: hub.ID, Scope: "page", TargetID: "missing", Patch: json.RawMessage(`{}`)}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("unknown page target code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = h.service.SetOverride(t.Context(), templates.SetOverrideRequest{TemplateID: "missing", Scope: "site", TargetID: h.siteID, Patch: json.RawMessage(`{}`)}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("unknown template code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = h.service.SetOverride(t.Context(), templates.SetOverrideRequest{TemplateID: hub.ID, Scope: "galaxy", TargetID: h.siteID, Patch: json.RawMessage(`{}`)}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad scope code = %q, want INVALID", errors.CodeOf(err))
	}
	if len(h.recorder.Events()) != 0 {
		t.Error("refused overrides must not publish")
	}

	got, err := h.service.GetTemplate(t.Context(), templates.GetTemplateRequest{ID: hub.ID})
	if err != nil || len(got.Overrides) != 2 {
		t.Errorf("GetTemplate overrides = %+v, %v", got.Overrides, err)
	}
	if _, err = h.service.DeleteOverride(t.Context(), templates.DeleteOverrideRequest{ID: siteOverride.Override.ID}); err != nil {
		t.Fatalf("DeleteOverride: %v", err)
	}
	h.wantEvents(t, 1)
	if _, err = h.service.DeleteOverride(t.Context(), templates.DeleteOverrideRequest{ID: siteOverride.Override.ID}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("DeleteOverride twice code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	afterDelete, err := h.service.ResolveForPage(t.Context(), templates.ResolveForPageRequest{PageID: page.ID})
	if err != nil || afterDelete.Spec.Tone != hub.Spec.Tone || afterDelete.Spec.Length.Min != 100 {
		t.Errorf("ResolveForPage after deleting the site override = %+v, %v", afterDelete.Spec, err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/application/templates/ -v`
Expected: FAIL, the package does not exist.

- [ ] **Step 3: Write the implementation**

`internal/application/templates/views.go`:

```go
package templates

import (
	"encoding/json"

	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

type Template struct {
	ID        string                `json:"id"`
	Scope     string                `json:"scope"`
	SiteID    *string               `json:"siteId"`
	Name      string                `json:"name"`
	PageKind  string                `json:"pageKind"`
	Version   int                   `json:"version"`
	Spec      template.TemplateSpec `json:"spec"`
	CreatedAt dto.Time              `json:"createdAt"`
	UpdatedAt dto.Time              `json:"updatedAt"`
}

type Override struct {
	ID         string          `json:"id"`
	TemplateID string          `json:"templateId"`
	Scope      string          `json:"scope"`
	TargetID   string          `json:"targetId"`
	Patch      json.RawMessage `json:"patch"`
	CreatedAt  dto.Time        `json:"createdAt"`
	UpdatedAt  dto.Time        `json:"updatedAt"`
}

type LinkPolicy struct {
	ID             string             `json:"id"`
	Scope          string             `json:"scope"`
	SiteID         *string            `json:"siteId"`
	Name           string             `json:"name"`
	Rules          template.LinkRules `json:"rules"`
	ForbidExternal bool               `json:"forbidExternal"`
	ForbidSelf     bool               `json:"forbidSelf"`
	AnchorStrategy string             `json:"anchorStrategy"`
	CreatedAt      dto.Time           `json:"createdAt"`
	UpdatedAt      dto.Time           `json:"updatedAt"`
}

func templateView(t template.Template) Template {
	return Template{
		ID:        t.ID,
		Scope:     string(t.Scope),
		SiteID:    t.SiteID,
		Name:      t.Name,
		PageKind:  t.PageKind,
		Version:   t.Version,
		Spec:      t.Spec,
		CreatedAt: dto.NewTime(t.CreatedAt),
		UpdatedAt: dto.NewTime(t.UpdatedAt),
	}
}

func overrideView(o template.Override) Override {
	return Override{
		ID:         o.ID,
		TemplateID: o.TemplateID,
		Scope:      string(o.Scope),
		TargetID:   o.TargetID,
		Patch:      o.Patch,
		CreatedAt:  dto.NewTime(o.CreatedAt),
		UpdatedAt:  dto.NewTime(o.UpdatedAt),
	}
}

func overrideViews(overrides []template.Override) []Override {
	out := make([]Override, 0, len(overrides))
	for _, o := range overrides {
		out = append(out, overrideView(o))
	}
	return out
}

func policyView(p template.LinkPolicy) LinkPolicy {
	return LinkPolicy{
		ID:             p.ID,
		Scope:          string(p.Scope),
		SiteID:         p.SiteID,
		Name:           p.Name,
		Rules:          p.Rules,
		ForbidExternal: p.ForbidExternal,
		ForbidSelf:     p.ForbidSelf,
		AnchorStrategy: string(p.AnchorStrategy),
		CreatedAt:      dto.NewTime(p.CreatedAt),
		UpdatedAt:      dto.NewTime(p.UpdatedAt),
	}
}
```

`internal/application/templates/requests.go`:

```go
package templates

import (
	"encoding/json"

	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

type CreateTemplateRequest struct {
	Scope    string                `json:"scope,omitempty"`
	SiteID   *string               `json:"siteId,omitempty"`
	Name     string                `json:"name"`
	PageKind string                `json:"pageKind"`
	Spec     template.TemplateSpec `json:"spec"`
}

type CreateTemplateResponse struct {
	Template Template `json:"template"`
}

type UpdateTemplateRequest struct {
	ID       string                 `json:"id"`
	Name     *string                `json:"name,omitempty"`
	PageKind *string                `json:"pageKind,omitempty"`
	Spec     *template.TemplateSpec `json:"spec,omitempty"`
}

type UpdateTemplateResponse struct {
	Template Template `json:"template"`
}

type DeleteTemplateRequest struct {
	ID string `json:"id"`
}

type DeleteTemplateResponse struct{}

type GetTemplateRequest struct {
	ID string `json:"id"`
}

type GetTemplateResponse struct {
	Template  Template   `json:"template"`
	Overrides []Override `json:"overrides"`
}

type ListTemplatesRequest struct {
	dto.ListRequest
	Scope    string `json:"scope,omitempty"`
	SiteID   string `json:"siteId,omitempty"`
	PageKind string `json:"pageKind,omitempty"`
}

type SetOverrideRequest struct {
	TemplateID string          `json:"templateId"`
	Scope      string          `json:"scope"`
	TargetID   string          `json:"targetId"`
	Patch      json.RawMessage `json:"patch"`
}

type SetOverrideResponse struct {
	Override Override `json:"override"`
}

type DeleteOverrideRequest struct {
	ID string `json:"id"`
}

type DeleteOverrideResponse struct{}

type ResolveForPageRequest struct {
	PageID string `json:"pageId"`
}

type ResolveForPageResponse struct {
	TemplateID string                `json:"templateId"`
	Version    int                   `json:"version"`
	Spec       template.TemplateSpec `json:"spec"`
}

type CreatePolicyRequest struct {
	Scope          string             `json:"scope,omitempty"`
	SiteID         *string            `json:"siteId,omitempty"`
	Name           string             `json:"name"`
	Rules          template.LinkRules `json:"rules"`
	ForbidExternal bool               `json:"forbidExternal"`
	ForbidSelf     bool               `json:"forbidSelf"`
	AnchorStrategy string             `json:"anchorStrategy,omitempty"`
}

type CreatePolicyResponse struct {
	Policy LinkPolicy `json:"policy"`
}

type UpdatePolicyRequest struct {
	ID             string              `json:"id"`
	Name           *string             `json:"name,omitempty"`
	Rules          *template.LinkRules `json:"rules,omitempty"`
	ForbidExternal *bool               `json:"forbidExternal,omitempty"`
	ForbidSelf     *bool               `json:"forbidSelf,omitempty"`
	AnchorStrategy *string             `json:"anchorStrategy,omitempty"`
}

type UpdatePolicyResponse struct {
	Policy LinkPolicy `json:"policy"`
}

type DeletePolicyRequest struct {
	ID string `json:"id"`
}

type DeletePolicyResponse struct{}

type GetPolicyRequest struct {
	ID string `json:"id"`
}

type GetPolicyResponse struct {
	Policy LinkPolicy `json:"policy"`
}

type ListPoliciesRequest struct {
	dto.ListRequest
	Scope  string `json:"scope,omitempty"`
	SiteID string `json:"siteId,omitempty"`
}

type GetEffectivePolicyRequest struct {
	SiteID string `json:"siteId"`
}

type GetEffectivePolicyResponse struct {
	Policy LinkPolicy `json:"policy"`
}
```

`internal/application/templates/service.go`:

```go
package templates

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

type templateStore interface {
	Insert(ctx context.Context, t template.Template) error
	Update(ctx context.Context, t template.Template) error
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (template.Template, error)
	List(ctx context.Context, q template.Query, page paging.Request) (paging.List[template.Template], error)
	UpsertOverride(ctx context.Context, o template.Override) (template.Override, error)
	GetOverride(ctx context.Context, templateID string, scope template.OverrideScope, targetID string) (template.Override, error)
	DeleteOverride(ctx context.Context, id string) error
	ListOverrides(ctx context.Context, templateID string) ([]template.Override, error)
}

type policyStore interface {
	Insert(ctx context.Context, p template.LinkPolicy) error
	Update(ctx context.Context, p template.LinkPolicy) error
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (template.LinkPolicy, error)
	List(ctx context.Context, q template.PolicyQuery, page paging.Request) (paging.List[template.LinkPolicy], error)
}

type pageReader interface {
	Get(ctx context.Context, id string) (pagemap.Page, error)
}

type siteReader interface {
	Get(ctx context.Context, id string) (site.Site, error)
}

type unitOfWork interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type Service struct {
	templates templateStore
	policies  policyStore
	pages     pageReader
	sites     siteReader
	uow       unitOfWork
	publisher application.Publisher
	clock     clock.Clock
}

func New(templates templateStore, policies policyStore, pages pageReader, sites siteReader, uow unitOfWork, publisher application.Publisher, clk clock.Clock) *Service {
	return &Service{templates: templates, policies: policies, pages: pages, sites: sites, uow: uow, publisher: publisher, clock: clk}
}

func (s *Service) now() time.Time {
	return s.clock.Now().UTC().Truncate(time.Second)
}

func (s *Service) changed() error {
	return s.publisher.Publish(events.TemplatesChanged, events.TemplatesChangedPayload{})
}

func scopeOf(raw string, siteID *string) (scope template.Scope, err error) {
	scope = template.Scope(raw)
	if raw == "" {
		scope = template.ScopeGlobal
		if siteID != nil {
			scope = template.ScopeSite
		}
	}
	if !scope.Valid() {
		return "", errors.New(errors.Invalid, "scope is not recognised").WithDetail("field", "scope")
	}
	return scope, nil
}

func (s *Service) requireTarget(ctx context.Context, scope template.Scope, siteID *string) error {
	if scope != template.ScopeSite {
		return nil
	}
	if siteID == nil || *siteID == "" {
		return errors.New(errors.Invalid, "a site record needs a site id").WithDetail("field", "siteId")
	}
	_, err := s.sites.Get(ctx, *siteID)
	return err
}

func sortOf(sort *dto.Sort) (key template.Sort, desc bool, err error) {
	if sort == nil {
		return template.SortCreatedAt, false, nil
	}
	key = template.Sort(sort.Field)
	if !key.Valid() {
		return "", false, errors.New(errors.Invalid, "templates and policies are sorted by createdAt or name").WithDetail("field", "sort.field")
	}
	return key, sort.Desc, nil
}

func scopeFilter(raw string) (*template.Scope, error) {
	if raw == "" {
		return nil, nil
	}
	scope := template.Scope(raw)
	if !scope.Valid() {
		return nil, errors.New(errors.Invalid, "scope is not recognised").WithDetail("field", "scope")
	}
	return &scope, nil
}

func siteFilter(raw string) *string {
	if raw == "" {
		return nil
	}
	return &raw
}
```

`internal/application/templates/templates.go`:

```go
package templates

import (
	"context"
	"strings"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func (s *Service) CreateTemplate(ctx context.Context, req CreateTemplateRequest) (CreateTemplateResponse, error) {
	scope, err := scopeOf(req.Scope, req.SiteID)
	if err != nil {
		return CreateTemplateResponse{}, err
	}
	if err = s.requireTarget(ctx, scope, req.SiteID); err != nil {
		return CreateTemplateResponse{}, err
	}

	now := s.now()
	record := template.Template{
		ID:        id.New(),
		Scope:     scope,
		SiteID:    req.SiteID,
		Name:      strings.TrimSpace(req.Name),
		PageKind:  strings.TrimSpace(req.PageKind),
		Version:   1,
		Spec:      req.Spec,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if scope == template.ScopeGlobal {
		record.SiteID = nil
	}
	if err = record.Validate(); err != nil {
		return CreateTemplateResponse{}, err
	}
	if err = s.uow.Do(ctx, func(c context.Context) error { return s.templates.Insert(c, record) }); err != nil {
		return CreateTemplateResponse{}, err
	}
	if err = s.changed(); err != nil {
		return CreateTemplateResponse{}, err
	}
	return CreateTemplateResponse{Template: templateView(record)}, nil
}

func (s *Service) UpdateTemplate(ctx context.Context, req UpdateTemplateRequest) (UpdateTemplateResponse, error) {
	var updated template.Template
	err := s.uow.Do(ctx, func(c context.Context) error {
		current, getErr := s.templates.Get(c, req.ID)
		if getErr != nil {
			return getErr
		}
		next := current
		if req.Name != nil {
			next.Name = strings.TrimSpace(*req.Name)
		}
		if req.PageKind != nil {
			next.PageKind = strings.TrimSpace(*req.PageKind)
		}
		if req.Spec != nil {
			next.Spec = *req.Spec
		}
		next.Version = current.Version + 1
		next.UpdatedAt = s.now()
		if validErr := next.Validate(); validErr != nil {
			return validErr
		}
		if updateErr := s.templates.Update(c, next); updateErr != nil {
			return updateErr
		}
		updated = next
		return nil
	})
	if err != nil {
		return UpdateTemplateResponse{}, err
	}
	if err = s.changed(); err != nil {
		return UpdateTemplateResponse{}, err
	}
	return UpdateTemplateResponse{Template: templateView(updated)}, nil
}

func (s *Service) DeleteTemplate(ctx context.Context, req DeleteTemplateRequest) (DeleteTemplateResponse, error) {
	if err := s.uow.Do(ctx, func(c context.Context) error { return s.templates.Delete(c, req.ID) }); err != nil {
		return DeleteTemplateResponse{}, err
	}
	if err := s.changed(); err != nil {
		return DeleteTemplateResponse{}, err
	}
	return DeleteTemplateResponse{}, nil
}

func (s *Service) GetTemplate(ctx context.Context, req GetTemplateRequest) (GetTemplateResponse, error) {
	record, err := s.templates.Get(ctx, req.ID)
	if err != nil {
		return GetTemplateResponse{}, err
	}
	overrides, err := s.templates.ListOverrides(ctx, record.ID)
	if err != nil {
		return GetTemplateResponse{}, err
	}
	return GetTemplateResponse{Template: templateView(record), Overrides: overrideViews(overrides)}, nil
}

func (s *Service) ListTemplates(ctx context.Context, req ListTemplatesRequest) (paging.List[Template], error) {
	scope, err := scopeFilter(req.Scope)
	if err != nil {
		return paging.List[Template]{}, err
	}
	key, desc, err := sortOf(req.Sort)
	if err != nil {
		return paging.List[Template]{}, err
	}
	q := template.Query{Scope: scope, SiteID: siteFilter(req.SiteID), PageKind: req.PageKind, Sort: key, Desc: desc}

	list, err := s.templates.List(ctx, q, application.PageRequest(req.ListRequest))
	if err != nil {
		return paging.List[Template]{}, err
	}
	return application.MapList(list, templateView), nil
}
```

`internal/application/templates/overrides.go`:

```go
package templates

import (
	"context"
	"encoding/json"

	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func (s *Service) SetOverride(ctx context.Context, req SetOverrideRequest) (SetOverrideResponse, error) {
	var stored template.Override
	err := s.uow.Do(ctx, func(c context.Context) error {
		base, getErr := s.templates.Get(c, req.TemplateID)
		if getErr != nil {
			return getErr
		}
		now := s.now()
		override := template.Override{
			ID:         id.New(),
			TemplateID: base.ID,
			Scope:      template.OverrideScope(req.Scope),
			TargetID:   req.TargetID,
			Patch:      req.Patch,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if validErr := override.Validate(); validErr != nil {
			return validErr
		}
		siteOverride, pageOverride, chainErr := s.chain(c, base, override)
		if chainErr != nil {
			return chainErr
		}
		if _, resolveErr := template.Resolve(base.Spec, siteOverride, pageOverride); resolveErr != nil {
			return resolveErr
		}
		saved, upsertErr := s.templates.UpsertOverride(c, override)
		if upsertErr != nil {
			return upsertErr
		}
		stored = saved
		return nil
	})
	if err != nil {
		return SetOverrideResponse{}, err
	}
	if err = s.changed(); err != nil {
		return SetOverrideResponse{}, err
	}
	return SetOverrideResponse{Override: overrideView(stored)}, nil
}

func (s *Service) chain(ctx context.Context, base template.Template, override template.Override) (siteOverride, pageOverride json.RawMessage, err error) {
	switch override.Scope {
	case template.OverrideSite:
		if _, err = s.sites.Get(ctx, override.TargetID); err != nil {
			return nil, nil, err
		}
		return override.Patch, nil, nil
	case template.OverridePage:
		page, getErr := s.pages.Get(ctx, override.TargetID)
		if getErr != nil {
			return nil, nil, getErr
		}
		siteOverride, err = s.patch(ctx, base.ID, template.OverrideSite, page.SiteID)
		if err != nil {
			return nil, nil, err
		}
		return siteOverride, override.Patch, nil
	default:
		return nil, nil, errors.New(errors.Invalid, "override scope is not recognised").WithDetail("field", "scope")
	}
}

func (s *Service) patch(ctx context.Context, templateID string, scope template.OverrideScope, targetID string) (json.RawMessage, error) {
	override, err := s.templates.GetOverride(ctx, templateID, scope, targetID)
	if errors.IsCode(err, errors.NotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return override.Patch, nil
}

func (s *Service) DeleteOverride(ctx context.Context, req DeleteOverrideRequest) (DeleteOverrideResponse, error) {
	if err := s.uow.Do(ctx, func(c context.Context) error { return s.templates.DeleteOverride(c, req.ID) }); err != nil {
		return DeleteOverrideResponse{}, err
	}
	if err := s.changed(); err != nil {
		return DeleteOverrideResponse{}, err
	}
	return DeleteOverrideResponse{}, nil
}

func (s *Service) ResolveForPage(ctx context.Context, req ResolveForPageRequest) (ResolveForPageResponse, error) {
	page, err := s.pages.Get(ctx, req.PageID)
	if err != nil {
		return ResolveForPageResponse{}, err
	}

	templateID := page.TemplateID
	if templateID == nil {
		owner, siteErr := s.sites.Get(ctx, page.SiteID)
		if siteErr != nil {
			return ResolveForPageResponse{}, siteErr
		}
		templateID = owner.Defaults.TemplateID
	}
	if templateID == nil {
		return ResolveForPageResponse{}, errors.New(errors.NotFound, "page has no template and its site has no default template").WithDetail("pageId", page.ID)
	}

	base, err := s.templates.Get(ctx, *templateID)
	if err != nil {
		return ResolveForPageResponse{}, err
	}
	siteOverride, err := s.patch(ctx, base.ID, template.OverrideSite, page.SiteID)
	if err != nil {
		return ResolveForPageResponse{}, err
	}
	pageOverride, err := s.patch(ctx, base.ID, template.OverridePage, page.ID)
	if err != nil {
		return ResolveForPageResponse{}, err
	}
	spec, err := template.Resolve(base.Spec, siteOverride, pageOverride)
	if err != nil {
		return ResolveForPageResponse{}, err
	}
	return ResolveForPageResponse{TemplateID: base.ID, Version: base.Version, Spec: spec}, nil
}
```

`internal/application/templates/seed.go`:

```go
package templates

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const DefaultPolicyName = "Default"

func defaultPolicy() template.LinkPolicy {
	return template.LinkPolicy{
		Scope: template.ScopeGlobal,
		Name:  DefaultPolicyName,
		Rules: template.LinkRules{
			UpDepth:                    2,
			DownLinks:                  true,
			SiblingMinWeight:           0.5,
			MaxLinks:                   12,
			MaxPerTarget:               1,
			ParentLinkWithinParagraphs: 2,
			ChildrenSection:            true,
		},
		ForbidExternal: true,
		ForbidSelf:     true,
		AnchorStrategy: template.AnchorPreferUser,
	}
}

func (s *Service) EnsureSeeded(ctx context.Context) error {
	global := template.ScopeGlobal
	return s.uow.Do(ctx, func(c context.Context) error {
		for _, seed := range template.Seed() {
			existing, err := s.templates.List(c, template.Query{Scope: &global, Name: seed.Name, Sort: template.SortCreatedAt}, paging.Request{Limit: 1})
			if err != nil {
				return err
			}
			if len(existing.Items) > 0 {
				continue
			}
			now := s.now()
			seed.ID = id.New()
			seed.CreatedAt = now
			seed.UpdatedAt = now
			if err = seed.Validate(); err != nil {
				return err
			}
			if err = s.templates.Insert(c, seed); err != nil {
				return err
			}
		}

		policies, err := s.policies.List(c, template.PolicyQuery{Scope: &global, Name: DefaultPolicyName, Sort: template.SortCreatedAt}, paging.Request{Limit: 1})
		if err != nil {
			return err
		}
		if len(policies.Items) > 0 {
			return nil
		}
		policy := defaultPolicy()
		now := s.now()
		policy.ID = id.New()
		policy.CreatedAt = now
		policy.UpdatedAt = now
		if err = policy.Validate(); err != nil {
			return err
		}
		return s.policies.Insert(c, policy)
	})
}
```

`EnsureSeeded` publishes nothing: it runs before any window exists and a `templates.changed` at startup would be dropped anyway.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race ./internal/application/templates/ -v`
Expected: PASS. The seeded `Default` link policy is asserted through `ListPolicies` in Task 25, once that method exists.

- [ ] **Step 5: Commit**

```bash
git add internal/application/templates/
git commit -m "feat(application): template use cases with overrides, resolution and startup seeding

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 25: `application/templates` — link policies

**Files:**
- Create: `internal/application/templates/policies.go`
- Test: `internal/application/templates/policies_test.go`

**Interfaces:**
- Consumes: Task 24's `Service`, `policyStore`, `siteReader`, `template.LinkPolicy|PolicyQuery|AnchorStrategy`.
- Produces: `CreatePolicy`, `UpdatePolicy`, `DeletePolicy`, `GetPolicy`, `ListPolicies(ctx, ListPoliciesRequest) (paging.List[LinkPolicy], error)`, `GetEffectivePolicy`.

- [ ] **Step 1: Write the failing test**

```go
package templates_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestPolicyLifecycleAndEffectivePolicy(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	if err := h.service.EnsureSeeded(t.Context()); err != nil {
		t.Fatalf("EnsureSeeded: %v", err)
	}
	h.recorder.Reset()

	seededPolicies, err := h.service.ListPolicies(t.Context(), templates.ListPoliciesRequest{Scope: "global"})
	if err != nil || len(seededPolicies.Items) != 1 || seededPolicies.Items[0].Name != templates.DefaultPolicyName {
		t.Fatalf("seeded policies = %+v, %v", seededPolicies.Items, err)
	}

	effective, err := h.service.GetEffectivePolicy(t.Context(), templates.GetEffectivePolicyRequest{SiteID: h.siteID})
	if err != nil || effective.Policy.Name != templates.DefaultPolicyName || effective.Policy.Scope != "global" {
		t.Fatalf("GetEffectivePolicy fallback = %+v, %v", effective.Policy, err)
	}

	rules := template.LinkRules{UpDepth: 1, DownLinks: true, SiblingMinWeight: 0.8, MaxLinks: 6, MaxPerTarget: 1, ParentLinkWithinParagraphs: 1}
	created, err := h.service.CreatePolicy(t.Context(), templates.CreatePolicyRequest{SiteID: &h.siteID, Name: "Strict", Rules: rules, ForbidExternal: true, ForbidSelf: true})
	if err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}
	if created.Policy.Scope != "site" || created.Policy.AnchorStrategy != "prefer_user" || created.Policy.Rules.MaxLinks != 6 {
		t.Errorf("CreatePolicy = %+v", created.Policy)
	}
	h.wantEvents(t, 1)

	owner, err := sqlite.NewSiteRepo(h.store).Get(t.Context(), h.siteID)
	if err != nil {
		t.Fatalf("Get site: %v", err)
	}
	owner.Defaults.LinkPolicyID = &created.Policy.ID
	if err = sqlite.NewSiteRepo(h.store).Update(t.Context(), owner); err != nil {
		t.Fatalf("set the site policy: %v", err)
	}
	effective, err = h.service.GetEffectivePolicy(t.Context(), templates.GetEffectivePolicyRequest{SiteID: h.siteID})
	if err != nil || effective.Policy.ID != created.Policy.ID {
		t.Errorf("GetEffectivePolicy via the site default = %+v, %v", effective.Policy, err)
	}

	updated, err := h.service.UpdatePolicy(t.Context(), templates.UpdatePolicyRequest{ID: created.Policy.ID, Name: ptr("Stricter"), ForbidExternal: ptr(false), AnchorStrategy: ptr("rotate")})
	if err != nil || updated.Policy.Name != "Stricter" || updated.Policy.ForbidExternal || updated.Policy.AnchorStrategy != "rotate" || updated.Policy.Rules.MaxLinks != 6 {
		t.Errorf("UpdatePolicy = %+v, %v", updated.Policy, err)
	}
	h.wantEvents(t, 1)

	if _, err = h.service.UpdatePolicy(t.Context(), templates.UpdatePolicyRequest{ID: created.Policy.ID, Rules: &template.LinkRules{MaxLinks: -1}}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad rules code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.CreatePolicy(t.Context(), templates.CreatePolicyRequest{Name: "default", Rules: rules}); !errors.IsCode(err, errors.Conflict) {
		t.Errorf("duplicate global name code = %q, want CONFLICT", errors.CodeOf(err))
	}
	if _, err = h.service.CreatePolicy(t.Context(), templates.CreatePolicyRequest{Scope: "site", Name: "x", Rules: rules}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("site scope without a site code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.CreatePolicy(t.Context(), templates.CreatePolicyRequest{Name: "x", Rules: rules, AnchorStrategy: "random"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad strategy code = %q, want INVALID", errors.CodeOf(err))
	}

	got, err := h.service.GetPolicy(t.Context(), templates.GetPolicyRequest{ID: created.Policy.ID})
	if err != nil || got.Policy.Name != "Stricter" {
		t.Errorf("GetPolicy = %+v, %v", got.Policy, err)
	}
	bySite, err := h.service.ListPolicies(t.Context(), templates.ListPoliciesRequest{SiteID: h.siteID})
	if err != nil || len(bySite.Items) != 1 {
		t.Errorf("ListPolicies by site = %+v, %v", bySite, err)
	}
	if _, err = h.service.ListPolicies(t.Context(), templates.ListPoliciesRequest{Scope: "galaxy"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad scope code = %q, want INVALID", errors.CodeOf(err))
	}

	if _, err = h.service.DeletePolicy(t.Context(), templates.DeletePolicyRequest{ID: created.Policy.ID}); err != nil {
		t.Fatalf("DeletePolicy: %v", err)
	}
	h.wantEvents(t, 1)
	effective, err = h.service.GetEffectivePolicy(t.Context(), templates.GetEffectivePolicyRequest{SiteID: h.siteID})
	if err != nil || effective.Policy.Name != templates.DefaultPolicyName {
		t.Errorf("after deleting the site policy the default applies again: %+v, %v", effective.Policy, err)
	}
	if _, err = h.service.DeletePolicy(t.Context(), templates.DeletePolicyRequest{ID: created.Policy.ID}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("DeletePolicy twice code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = h.service.GetEffectivePolicy(t.Context(), templates.GetEffectivePolicyRequest{SiteID: "missing"}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("unknown site code = %q, want NOT_FOUND", errors.CodeOf(err))
	}

	other := sqlitetest.Site(t, h.store, "blog")
	if _, err = h.service.GetEffectivePolicy(t.Context(), templates.GetEffectivePolicyRequest{SiteID: other.ID}); err != nil {
		t.Errorf("every site falls back to the default policy: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/application/templates/ -run 'TestPolicy|TestEnsureSeeded' -v`
Expected: FAIL, the policy methods are undefined.

- [ ] **Step 3: Write the implementation**

`internal/application/templates/policies.go`:

```go
package templates

import (
	"context"
	"strings"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func (s *Service) CreatePolicy(ctx context.Context, req CreatePolicyRequest) (CreatePolicyResponse, error) {
	scope, err := scopeOf(req.Scope, req.SiteID)
	if err != nil {
		return CreatePolicyResponse{}, err
	}
	if err = s.requireTarget(ctx, scope, req.SiteID); err != nil {
		return CreatePolicyResponse{}, err
	}

	strategy := template.AnchorStrategy(req.AnchorStrategy)
	if req.AnchorStrategy == "" {
		strategy = template.AnchorPreferUser
	}
	now := s.now()
	record := template.LinkPolicy{
		ID:             id.New(),
		Scope:          scope,
		SiteID:         req.SiteID,
		Name:           strings.TrimSpace(req.Name),
		Rules:          req.Rules,
		ForbidExternal: req.ForbidExternal,
		ForbidSelf:     req.ForbidSelf,
		AnchorStrategy: strategy,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if scope == template.ScopeGlobal {
		record.SiteID = nil
	}
	if err = record.Validate(); err != nil {
		return CreatePolicyResponse{}, err
	}
	if err = s.uow.Do(ctx, func(c context.Context) error { return s.policies.Insert(c, record) }); err != nil {
		return CreatePolicyResponse{}, err
	}
	if err = s.changed(); err != nil {
		return CreatePolicyResponse{}, err
	}
	return CreatePolicyResponse{Policy: policyView(record)}, nil
}

func (s *Service) UpdatePolicy(ctx context.Context, req UpdatePolicyRequest) (UpdatePolicyResponse, error) {
	var updated template.LinkPolicy
	err := s.uow.Do(ctx, func(c context.Context) error {
		current, getErr := s.policies.Get(c, req.ID)
		if getErr != nil {
			return getErr
		}
		next := current
		if req.Name != nil {
			next.Name = strings.TrimSpace(*req.Name)
		}
		if req.Rules != nil {
			next.Rules = *req.Rules
		}
		if req.ForbidExternal != nil {
			next.ForbidExternal = *req.ForbidExternal
		}
		if req.ForbidSelf != nil {
			next.ForbidSelf = *req.ForbidSelf
		}
		if req.AnchorStrategy != nil {
			next.AnchorStrategy = template.AnchorStrategy(*req.AnchorStrategy)
		}
		next.UpdatedAt = s.now()
		if validErr := next.Validate(); validErr != nil {
			return validErr
		}
		if updateErr := s.policies.Update(c, next); updateErr != nil {
			return updateErr
		}
		updated = next
		return nil
	})
	if err != nil {
		return UpdatePolicyResponse{}, err
	}
	if err = s.changed(); err != nil {
		return UpdatePolicyResponse{}, err
	}
	return UpdatePolicyResponse{Policy: policyView(updated)}, nil
}

func (s *Service) DeletePolicy(ctx context.Context, req DeletePolicyRequest) (DeletePolicyResponse, error) {
	if err := s.uow.Do(ctx, func(c context.Context) error { return s.policies.Delete(c, req.ID) }); err != nil {
		return DeletePolicyResponse{}, err
	}
	if err := s.changed(); err != nil {
		return DeletePolicyResponse{}, err
	}
	return DeletePolicyResponse{}, nil
}

func (s *Service) GetPolicy(ctx context.Context, req GetPolicyRequest) (GetPolicyResponse, error) {
	record, err := s.policies.Get(ctx, req.ID)
	if err != nil {
		return GetPolicyResponse{}, err
	}
	return GetPolicyResponse{Policy: policyView(record)}, nil
}

func (s *Service) ListPolicies(ctx context.Context, req ListPoliciesRequest) (paging.List[LinkPolicy], error) {
	scope, err := scopeFilter(req.Scope)
	if err != nil {
		return paging.List[LinkPolicy]{}, err
	}
	key, desc, err := sortOf(req.Sort)
	if err != nil {
		return paging.List[LinkPolicy]{}, err
	}
	q := template.PolicyQuery{Scope: scope, SiteID: siteFilter(req.SiteID), Sort: key, Desc: desc}

	list, err := s.policies.List(ctx, q, application.PageRequest(req.ListRequest))
	if err != nil {
		return paging.List[LinkPolicy]{}, err
	}
	return application.MapList(list, policyView), nil
}

func (s *Service) GetEffectivePolicy(ctx context.Context, req GetEffectivePolicyRequest) (GetEffectivePolicyResponse, error) {
	owner, err := s.sites.Get(ctx, req.SiteID)
	if err != nil {
		return GetEffectivePolicyResponse{}, err
	}
	if owner.Defaults.LinkPolicyID != nil {
		record, getErr := s.policies.Get(ctx, *owner.Defaults.LinkPolicyID)
		if getErr != nil {
			return GetEffectivePolicyResponse{}, getErr
		}
		return GetEffectivePolicyResponse{Policy: policyView(record)}, nil
	}

	global := template.ScopeGlobal
	list, err := s.policies.List(ctx, template.PolicyQuery{Scope: &global, Name: DefaultPolicyName, Sort: template.SortCreatedAt}, paging.Request{Limit: 1})
	if err != nil {
		return GetEffectivePolicyResponse{}, err
	}
	if len(list.Items) == 0 {
		return GetEffectivePolicyResponse{}, errors.New(errors.NotFound, "no link policy applies to the site").WithDetail("siteId", req.SiteID)
	}
	return GetEffectivePolicyResponse{Policy: policyView(list.Items[0])}, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race -cover ./internal/application/... -v`
Expected: PASS for every application package, each ≥ 85%.

- [ ] **Step 5: Commit**

```bash
git add internal/application/templates/policies.go internal/application/templates/policies_test.go
git commit -m "feat(application): link policy use cases with the effective-policy lookup

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---
### Task 26: Composition root — event relay, use cases, seeding at `Open`

**Files:**
- Create: `internal/app/events.go`
- Modify: `internal/app/core.go`, `cmd/postulator/main.go`
- Test: `internal/app/events_test.go`, `internal/app/core_test.go`

**Interfaces:**
- Consumes: `wails.EventBridge|NewEventBridge|Emitter`, `events.Type`, `sites.New`, `graph.New`, `pages.New`, `templates.New|EnsureSeeded`, every `sqlite.New*Repo`, `secrets.NewStore`.
- Produces: `app.EventRelay` with `Connect(emitter wails.Emitter, now clock.Clock) error` and `Publish(eventType events.Type, payload any) error` (satisfies `application.Publisher`); `Core` gains `Events *EventRelay`, `Sites *sites.Service`, `Graph *graph.Service`, `Pages *pages.Service`, `Templates *templates.Service`; `Open` seeds the starter templates; `Services()` is unchanged.

- [ ] **Step 1: Write the failing tests**

`internal/app/events_test.go`:

```go
package app_test

import (
	"sync"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

var _ application.Publisher = (*app.EventRelay)(nil)

type countingEmitter struct {
	mu    sync.Mutex
	names []string
}

func (c *countingEmitter) Emit(name string, _ ...any) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.names = append(c.names, name)
	return true
}

func (c *countingEmitter) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.names)
}

func TestEventRelayDropsBeforeConnectAndForwardsAfter(t *testing.T) {
	t.Parallel()

	relay := &app.EventRelay{}
	if err := relay.Publish(events.GraphChanged, events.GraphChangedPayload{SiteID: "s1"}); err != nil {
		t.Fatalf("Publish before Connect must be a no-op, got %v", err)
	}

	emitter := &countingEmitter{}
	now := clock.NewFake(time.Date(2026, time.September, 18, 9, 0, 0, 0, time.UTC))
	if err := relay.Connect(emitter, now); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if err := relay.Publish(events.GraphChanged, events.GraphChangedPayload{SiteID: "s1"}); err != nil {
		t.Fatalf("Publish after Connect: %v", err)
	}
	if emitter.count() != 1 {
		t.Errorf("emitted %d events, want 1", emitter.count())
	}
	if err := relay.Publish(events.Type("nope"), events.GraphChangedPayload{}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("unknown type code = %q, want INVALID", errors.CodeOf(err))
	}
	if err := relay.Connect(emitter, now); !errors.IsCode(err, errors.Internal) {
		t.Errorf("second Connect code = %q, want INTERNAL", errors.CodeOf(err))
	}
}
```

Append to `internal/app/core_test.go` (add `"github.com/davidmovas/postulator/internal/application/templates"` to its imports):

```go
func TestOpenWiresTheUseCasesAndSeedsTheStarterTemplates(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	cfg := app.Config{DatabasePath: filepath.Join(home, "postulator.db"), KeyDir: home}

	for round := range 2 {
		core, err := app.Open(t.Context(), cfg)
		if err != nil {
			t.Fatalf("Open round %d: %v", round, err)
		}
		if core.Events == nil || core.Sites == nil || core.Graph == nil || core.Pages == nil || core.Templates == nil {
			t.Fatal("the core must carry the relay and the four services")
		}

		seeded, err := core.Templates.ListTemplates(t.Context(), templates.ListTemplatesRequest{Scope: "global"})
		if err != nil {
			t.Fatalf("ListTemplates: %v", err)
		}
		if len(seeded.Items) != 5 {
			t.Errorf("round %d: seeded templates = %d, want 5", round, len(seeded.Items))
		}
		policies, err := core.Templates.ListPolicies(t.Context(), templates.ListPoliciesRequest{Scope: "global"})
		if err != nil || len(policies.Items) != 1 {
			t.Errorf("round %d: seeded policies = %d, %v; want 1", round, len(policies.Items), err)
		}

		if err = core.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/app/ -run 'TestEventRelay|TestOpenWires' -v`
Expected: FAIL, `app.EventRelay` and the new `Core` fields are undefined.

- [ ] **Step 3: Write the relay, rewire the core and connect the bridge**

`internal/app/events.go`:

```go
package app

import (
	"sync/atomic"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/transport/wails"
)

type EventRelay struct {
	bridge atomic.Pointer[wails.EventBridge]
}

func (r *EventRelay) Connect(emitter wails.Emitter, now clock.Clock) error {
	if !r.bridge.CompareAndSwap(nil, wails.NewEventBridge(emitter, now)) {
		return errors.New(errors.Internal, "the event bridge is already connected")
	}
	return nil
}

func (r *EventRelay) Publish(eventType events.Type, payload any) error {
	bridge := r.bridge.Load()
	if bridge == nil {
		return nil
	}
	return bridge.Publish(eventType, payload)
}
```

Replace `internal/app/core.go` with:

```go
package app

import (
	"context"
	stderrors "errors"
	"os"
	"path/filepath"

	"github.com/davidmovas/postulator/internal/adapters/secrets"
	"github.com/davidmovas/postulator/internal/adapters/secrets/masterkey"
	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/settings"
)

const (
	homeDirectory = "Postulator"
	databaseFile  = "postulator.db"
)

type Config struct {
	DatabasePath string
	KeyDir       string
}

func (c Config) recovery() string {
	return "remove " + filepath.Join(c.KeyDir, masterkey.FileName) + " and " + c.DatabasePath +
		" to reset the application state"
}

func DefaultConfig() (Config, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return Config{}, errors.Wrap(err, errors.Internal, "locate the user configuration directory")
	}

	home := filepath.Join(base, homeDirectory)
	return Config{DatabasePath: filepath.Join(home, databaseFile), KeyDir: home}, nil
}

type Core struct {
	Store           *sqlite.Store
	Secrets         *secrets.Store
	Settings        *settings.Values
	UnknownSettings []string
	Events          *EventRelay
	Sites           *sites.Service
	Graph           *graph.Service
	Pages           *pages.Service
	Templates       *templates.Service
}

func Open(ctx context.Context, cfg Config) (*Core, error) {
	if cfg.DatabasePath == "" {
		return nil, errors.New(errors.Invalid, "the database path must not be empty")
	}
	if cfg.KeyDir == "" {
		return nil, errors.New(errors.Invalid, "the key directory must not be empty")
	}

	recovery := cfg.recovery()

	key, err := masterkey.Load(masterkey.Config{Dir: cfg.KeyDir, Recovery: recovery})
	if err != nil {
		return nil, err
	}

	store, err := sqlite.Open(sqlite.Config{Path: cfg.DatabasePath, Key: key, Recovery: recovery})
	if err != nil {
		return nil, err
	}

	now := clock.System{}
	values, unknown, err := LoadSettings(ctx, sqlite.NewSettingsRepo(store, now), settings.Default())
	if err != nil {
		return nil, stderrors.Join(err, store.Close())
	}

	secretStore := secrets.NewStore(sqlite.NewSecretsRepo(store, now), key)
	relay := &EventRelay{}
	siteRepo := sqlite.NewSiteRepo(store)
	entityRepo := sqlite.NewEntityRepo(store)
	edgeRepo := sqlite.NewEdgeRepo(store)
	pageRepo := sqlite.NewPageRepo(store)
	linkRepo := sqlite.NewPageLinkRepo(store)
	templateRepo := sqlite.NewTemplateRepo(store)
	policyRepo := sqlite.NewLinkPolicyRepo(store)

	core := &Core{
		Store:           store,
		Secrets:         secretStore,
		Settings:        values,
		UnknownSettings: unknown,
		Events:          relay,
		Sites:           sites.New(siteRepo, secretStore, store, now),
		Graph:           graph.New(entityRepo, edgeRepo, siteRepo, store, relay, now),
		Pages:           pages.New(pageRepo, linkRepo, entityRepo, siteRepo, store, relay, now),
		Templates:       templates.New(templateRepo, policyRepo, pageRepo, siteRepo, store, relay, now),
	}
	if err = core.Templates.EnsureSeeded(ctx); err != nil {
		return nil, stderrors.Join(err, store.Close())
	}
	return core, nil
}

func (c *Core) Close() error {
	return c.Store.Close()
}
```

In `cmd/postulator/main.go`, add the import `"github.com/davidmovas/postulator/internal/kernel/clock"` and, directly after the `wails := application.New(...)` statement, insert:

```go
	if err = core.Events.Connect(wails.Event, clock.System{}); err != nil {
		return err
	}
```

- [ ] **Step 4: Run the tests and the dependency rule**

Run: `go build ./... && go test -race ./internal/app/ -v`
Expected: PASS, including `TestLayersDoNotReachUpwards`, `TestDomainCarriesNoThirdPartyDependencies`, `TestOnlyPagingUsesTheQueryBuilder` and `TestOnlyTheCompositionRootImportsTransport` with `domain` and `application` now enforced.

- [ ] **Step 5: Commit**

```bash
git add internal/app/events.go internal/app/events_test.go internal/app/core.go internal/app/core_test.go cmd/postulator/main.go
git commit -m "feat(app): wire the use cases, seed templates at open and relay events to one bridge

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 27: Spec amendments, contracts, status and the full gate

**Files:**
- Modify: `docs/superpowers/specs/2026-09-17-postulator-v2-design.md`, `docs/CONTRACTS.md`, `docs/STATUS.md`

**Interfaces:**
- Consumes: everything above.
- Produces: the documentation that records where `Cannibalization` and `Resolve` ended up, who owns the DTOs, and the Phase 2 handoff.

- [ ] **Step 1: Amend the spec copy**

In `docs/superpowers/specs/2026-09-17-postulator-v2-design.md`:

Append this bullet to §5.3 after the "Link target resolution" bullet:

```markdown
- `Cannibalization(candidate Page, entity graph.Entity, index Index, g graph.Graph) Verdict{Allowed bool, Evidence []Evidence{PageID, Path, Reason(same_primary_keyword|same_entity_canonical|path_conflict), EntityID}}` lives here since Phase 2 (moved from §6): it depends only on the page index and the entity graph, and `pages.Create`, `pages.MapToEntity` and the import preview call it. `InternalPath(href, siteHost) (path, internal)` classifies a link target against the site host; `NormalizePath` is the one canonical path normaliser and the WordPress adapter and plugin implement its rules.
```

In §5.4 replace the `Resolve` bullet with:

```markdown
- `Resolve(base TemplateSpec, siteOverride, pageOverride json.RawMessage) (TemplateSpec, error)` implements RFC 7396 merge patch over the JSON form of the spec (Phase 2): overrides are stored as merge-patch documents, arrays are replaced wholesale, `null` removes a key, and the resolved spec must pass `Validate`. `TemplateSpec` and its nested structs carry camelCase JSON tags because that JSON is their persisted form.
```

In §6 replace the `Cannibalization` bullet with:

```markdown
- `Cannibalization` moved to §5.3 (`domain/pagemap`) in Phase 2.
```

- [ ] **Step 2: Amend `docs/CONTRACTS.md`**

Replace the DTO bullet `- DTOs are declared in `internal/transport/wails` and mapped by hand. Domain types carry no JSON tags.` with:

```markdown
- Request and response structs are declared by the application use cases
  (`internal/application/<context>`) with camelCase JSON tags and `kernel/dto.Time`
  timestamps; a Wails service passes them through unchanged and maps nothing. Domain
  types carry no JSON tags, except `template.TemplateSpec` and the `llm` catalog types,
  whose persisted form is JSON.
```

Add after the pagination paragraph:

```markdown
A use case's list request embeds `kernel/dto.ListRequest`; `cursor` is always the
`nextCursor` of the previous page (forward paging), `sort.field` is `createdAt` by default
and `name` or `path` where a context offers it. A client that needs the previous page
replays the cursor it used to reach the current one.
```

- [ ] **Step 3: Update `docs/STATUS.md`**

After the "1B reviewed" paragraph insert:

```markdown
**Phase 2 (domain core) is complete.** The plan is
`docs/superpowers/plans/2026-09-18-phase-2-domain-core.md`; its twenty-seven tasks landed
one commit each. `internal/domain/{llm,site,graph,pagemap,template}` hold the pure model:
the entity graph with breadth-first parents, cycle detection and PageRank scores, the
path-keyed page map with its tree, index and cannibalization verdict, the template spec
with RFC 7396 resolution and five embedded starter templates. Migrations 0004–0010 add
`sites`, `link_policies`, `templates`, `entities`, `entity_anchors`, `edges`, `pages`,
`page_links` and `template_overrides`; `internal/adapters/sqlite` holds one keyset-paged
repository per aggregate; `internal/application/{sites,graph,pages,templates}` hold the use
cases, each publishing one event after its transaction commits; `internal/app` wires them,
seeds the templates at `Open` and relays events to the single `EventBridge` once
`cmd/postulator` connects it. Phase 3 (WordPress) and Phase 4 (LLM) follow.
```

Add a `## Decisions taken in Phase 2` section holding, one bullet each, decisions 1–24 of the plan's Design decisions in their short form (breadth-first parents; unweighted PageRank; merge-patch overrides as documents; anchors as a child table loaded per page; overrides keyed by template, scope and target through an expression index; one event after commit; cannibalization on create, path change and mapping; cannibalization in `pagemap` with the graph parameter; path as the hierarchy's source of truth with the binding normalisation rules and `InternalPath`; query types in the domain; `dto.ListRequest` with forward paging; JSON views owned by the use cases; clockless repositories; `Site.Username`; the `EventRelay`; seeds without pinned models or step params; cyclic foreign keys; parent edges weigh 1; acyclicity checked on add and approve; `CONFLICT`/`NOT_FOUND` mapping in `execWrite`; the seeded `Default` policy; `SetCanonical` auto-maps; `EdgeQuery` sorts by creation only; evidence and cycles in `Details`).

In "Known gaps" delete the `EventBridge has no publisher yet` and `internal/domain and internal/runtime do not exist yet` items and add:

```markdown
- Application events published before `cmd/postulator` connects the relay to the Wails
  event manager are dropped, by design: nothing listens before the window exists.
- Backward paging exists in every repository (`paging.Request.Before`) but not at the
  use-case boundary, because `dto.ListRequest` carries one cursor; the client replays the
  cursor it used to reach the current page.
- Step `params` keys in template recipes are unspecified until Phase 6 names them; the
  seeds carry none.
- `internal/runtime` does not exist yet; its dependency rule still skips.
```

- [ ] **Step 4: Run the whole gate**

```bash
go build ./...
go vet ./...
$(go env GOPATH)/bin/golangci-lint run
go test -count=1 -race -covermode=atomic -coverprofile=coverage.out ./...
go run ./cmd/covergate
go tool cover -func=coverage.out | grep -E 'internal/(domain|application)' | awk '{print $NF, $1}' | sort | head -n 60
task build
```

Expected: every command green, 0 lint issues, `covergate` reporting `domain+application` ≥ 85% and `total` ≥ 70%, every `internal/domain/*` and `internal/application/*` package ≥ 85% in the per-function listing, `task build` producing `bin/postulator.exe` with `npm run typecheck` passing (no TypeScript changed this phase). If any package is short, add the missing table cases to its existing test file rather than a new one.

- [ ] **Step 5: Commit**

```bash
git add docs/superpowers/specs/2026-09-17-postulator-v2-design.md docs/CONTRACTS.md docs/STATUS.md
git commit -m "docs(status): phase 2 domain core, spec and contract amendments

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

## Self-review

**Spec coverage.** §5.1 is Task 2 (with `Username` added, decision 14). §5.2 is Tasks 3–5: every field of `Entity`, `Anchor` and `Edge`, the constructor invariants, `Parents/Children/Related/Roots/ValidateAcyclic/Score`. §5.3 is Tasks 6–7: `Page`, `PageLink`, `NormalizePath`, `ParentPath`, `BuildTree`, `Unmapped`, `Index` with `ByID/ByEntity/ByPath`, plus the `InternalPath` classifier and the `Cannibalization` verdict the task moved here. §5.4 is Tasks 8–10: `Template`, every nested spec struct, `StepSpec`, `Resolve` as a merge patch (decision 3 records the signature change), `Validate`, `LinkPolicy`, and the five seeds with the §6 step names. §5.9 is Task 1. §9.1's Phase 2 tables are Task 11 with `dbx.Result` at every repository boundary in Tasks 12–17. §9.6's later needs (`ListBySite` bulk readers, `ReplaceForPage`, entity names deduplicated case-insensitively, a cannibalization verdict for the preview) are provided by Tasks 13–15 and 22. The task brief's D section maps one-to-one onto Tasks 19–25 and its E section onto Task 26. No `internal/domain/settings` package is created and no setting is declared, because no use case in this phase reads one.

**Placeholder scan.** No task says "similar to", "TBD" or "handle errors"; every code step is complete Go, SQL or JSON with no comments, and every code block is the final text of the file it names.

**Type consistency.** `graphdomain` is the alias for `internal/domain/graph` in every `internal/application/graph` file. `EntityQuery`, `EdgeQuery`, `pagemap.Query`, `template.Query` and `template.PolicyQuery` carry the same fields in Tasks 3, 6, 8, 13–17 and 19–25. `EntityRepo.SetCanonicalPage(ctx, id, pageID *string, updatedAt time.Time)` matches the `entityStore` in Task 22. `TemplateRepo.UpsertOverride` returns `(template.Override, error)` in Tasks 16 and 24. `sqlitetest.Site|Entity|Page|Template` and `Stamp` are introduced in Tasks 12, 13, 15 and 16 before any later task uses them. The service constructors `sites.New`, `graph.New`, `pages.New`, `templates.New` have the same parameter order in their tasks and in Task 26. `events.GraphChanged|PagesChanged|TemplatesChanged` and their payload structs are the existing Phase 1B names.

## Open questions for the orchestrator

1. **`Resolve` signature.** The spec's `*TemplateSpec` overrides cannot express a merge patch (decision 3); this plan stores and passes `json.RawMessage` patches. Confirm, or the alternative is a pointer-per-field `TemplatePatch` struct that reimplements the RFC by hand.
2. **`Site.Username`.** Added because Basic auth needs it and the spec's `Site` lacks it (decision 14). Confirm, or Phase 3 will have to store it elsewhere.
3. **Backward paging at the use-case boundary.** `dto.ListRequest` has one cursor, so the use cases page forward only (decision 11). If backward paging must reach the UI, the kernel needs `Before` on `dto.ListRequest` and CONTRACTS.md changes; this plan does not touch the kernel.
4. **Seeds pin no models** (decision 16). If the seeds should name a provider/model per role, say which; the catalog is Phase 4 data and the plan avoids hard-coding it.
5. **`docs/CONTRACTS.md` DTO ownership.** The task's "JSON-tagged request/response structs owned by the use cases" contradicts the current CONTRACTS line; Task 27 amends it. Confirm that amendment.
