# Architecture

Modular monolith, hexagonal. A pure domain, one application layer of use cases, and two
driving adapters over it: Wails services for the UI and a tool registry for agents. Both
call the same use cases, so an agent can do whatever the UI can do without a parallel
code path.

## Layers

```
cmd/postulator        Wails bootstrap only
internal/kernel       errors, paging, id, clock, log, ctx, dto, settings, middleware
internal/domain       pure logic: site graph pagemap template content run schedule settings llm
internal/application  use cases, UnitOfWork, tool registry, event registry
internal/adapters     sqlite, wp, llm, images, secrets, importer
internal/runtime      the run engine and its step catalog
internal/transport    wails services and event bridge, agent runner and tools
internal/app          composition root, manual wiring, no DI container
wp-plugin             the WordPress companion plugin
frontend              React + Tailwind over the generated bindings; src/canvas is the 2D engine under the graph map, src/features/agent the dock, the agent screens and the confirmation cards
```

## The dependency rule

- `domain` imports `kernel` and the standard library. Nothing else. The one third-party
  package it may ever reach is `golang.org/x/net/html`, because the content model is an
  HTML document.
- `application` imports `domain` and `kernel`.
- `adapters`, `runtime` and `transport` import `application`, `domain` and `kernel`.
- Only `app` imports `transport`.
- `paging` is the only kernel package allowed to import a query builder. Domain never
  sees SQL.

`internal/app/deps_test.go` enforces all of this with `go list`. Domain purity is
checked as an absence of third-party packages in its transitive closure, which is what
catches a domain package quietly importing `kernel/paging`: the query builder shows up
in the closure even though the import looked local. A rule whose tree does not exist
yet skips, so each layer starts being enforced the day its first package lands.

## Interfaces

Interfaces are declared by the consumer, never by the implementation and never in
advance. A use case that needs to read pages declares the smallest interface it uses;
the SQLite repository returns a concrete type and happens to satisfy it. Ports named in
the spec (`SecretStore`, the LLM `Client`, `ImageProvider`, `Step`) live in the package
that calls them, not in the package that implements them.

## Kernel

The shared vocabulary every layer is allowed to speak.

- `errors` — a frozen `Code` set (`NotFound Conflict Invalid Unauthorized RateLimited
  BudgetExceeded External Internal Cancelled NeedsHuman Locked`), a stack captured at
  construction, `Is` comparing by code, clone-and-mutate enrichment. `CodeOf` answers
  `Internal` for foreign errors, so everything is classifiable at a boundary.
- `paging` — opaque base64url cursors carrying `{o,d,v,i}`, typed sort keys, a row-value
  keyset predicate with an id tie-break, and `Cut` to turn `limit+1` rows into a page.
  Offset pagination does not exist in this codebase.
- `id`, `clock`, `ctx`, `dto` — UUID v4 text, an injectable clock, run/conversation/actor
  context values, and the RFC3339 UTC time type every DTO uses.
- `log` — zap over a console core and two lumberjack files (`app.log`, `errors.log`),
  with `password`, `apiKey`, `token` and `authorization` redacted by key.
- `settings` — settings-as-code: each setting is declared next to the code that reads
  it, with its default and its validators, and the registry can describe the whole set
  as a schema for the UI.
- `middleware` — generic `Recover`, `Timeout` and `Audit` wrappers over
  `func(ctx, In) (Out, error)`.

## Domain

Value types and pure functions. `site`, `graph` (a DAG on `parent` edges plus weighted
undirected `related` edges), `pagemap` (paths, tree, index), `template` (spec plus link
policy, resolved global → site → page by JSON merge patch), `content` (an HTML document,
link context, link insertion, compliance and structure reports), `run`, `schedule`,
`settings`, and the `llm` model catalog types.

The heart of it is `content`: `PlanLinks` turns a graph, a page index and a
`content.Subject{Site, PageID, PagePath, EntityID}` into the set of links that page owes,
`InsertLinks` places them with a decision recorded per candidate, and `Compliance` grades
the result back against the graph. The subject is the page in hand, never its entity's
canonical page, so a run's validation and the Linking screen bind "self" to the same page.

There is one answer to where an href points, and it is `pagemap.Site{Scheme, Host, Base}`,
built once per site from its `BaseURL`. `Site.Resolve` answers a normalised site path and a
kind (`path`, `same_document`, `external`, `unresolved`) by mirroring the companion plugin's
`internal_href_to_path` rule by rule, so what the plugin records and what the domain judges
cannot drift. `content.LinkContext` carries that site and every classifier — insertion,
compliance, the audit and the link counter — resolves through it.

## Run engine

A run is a recipe of named steps over a set of target pages, and a run kind may own that
recipe. `relink` (`resolve_context relink_page sync_back report`), `repair`
(`repair_hierarchy sync_back report`), `sync` (`sync_site`) and `revert` (`revert`) hand out
their own steps and refuse a recipe that disagrees; `generate` takes the template's, or
`run.GenerateRecipe()` when neither the request nor the template names one; `custom` is the
one kind exempt from every step rule. The four steps a kind owns — `repair_hierarchy`,
`sync_site`, `relink_page` and `revert` — are `PerKindStepName` values, a type of their own,
because `StepName` is the vocabulary a *template* recipe may draw from and it is rendered into
`vocab.ts`, into the blank template's recipe and into three tool enums. Sixteen steps are
registered; twelve of them a template may name.

`relink_page` places the links a page's own rules ask for against the page's own cap and
writes the body back under a hash compare-and-swap. It calls no model, so a relink, a repair
and a sync are all estimated at nothing.

A **revert** is a run like any other: kind `revert`, a parent run, one item per page the
source run still holds a `publish_result` for, and a reversed order so a parent is never
trashed before its children. A page the run created goes to the WordPress trash and its row
returns to `planned` with no WordPress id, no hash, no mirror and no links; a page it updated
has `previousContent` written back under a CAS and `previousMeta` restored through the
plugin's `seo_meta_read`; every neighbour a relink rewrote has its `before.html` put back.
A human edit since the run, a missing record, a page already gone or a site without the
plugin holds that one item with both facts named, and every other item still goes back. Media
stays.

`advance(itemID)` claims an
item in one SQLite transaction by compare-and-swap, runs its current step under a
timeout, classifies the outcome into a `Fault`, and persists the new state, the
checkpoint, a `StepExec` row and an appended event together. Large step outputs go to
`Artifact` rows and the checkpoint keeps their ids.

Steps are idempotent: a step is skipped if a `StepExec` already records it done for an
input hash of `(step, params, required artifact hashes, template version)`. A sweep
daemon re-arms waiting items, reclaims items whose lease expired because the process
died mid-step, and fails runs past their deadline. Crash recovery is one sweep at
startup, not a separate mechanism.

An item that fails never fails the run. The run completes with `Stats.Failed > 0`, or
fails only when every item failed.

## Transport

Wails services are thin: DTO mapping and nothing else. Any mutation longer than a
second returns `{runId}` immediately and progress arrives as events. The agent side
adapts the same use cases into tools, with a guard chain of `fence → audit → capResult
→ permission`, and a write in confirm mode becomes a `PendingAction` row rather than a
goroutine, so a confirmation survives a restart. The composition root constructs exactly
one `EventBridge` per process, because the application-event `seq` is a counter held by
that instance and a second bridge would restart it.

The guard itself is not transport's. `Fence`, `Permit` and `Cap` live in
`internal/application/agent`; `internal/transport/agent` adapts them to gollem middleware
and `Confirm` calls them directly, so a confirmed tool is fenced, permitted and capped by
construction rather than by a second implementation. Nothing is injected through `Deps` and
nothing in `application` imports `transport`, so the dependency rule holds without a new
seam.

Every model call of an agent turn is its own `llm_calls` row with `step = "chat"`, priced
against the catalog per round and written outside the turn's cancellation, so a stopped turn
is still billed for the rounds that ran. The turn's cost is the sum of those rows, which is
what makes the spend badge and the ledger agree by construction instead of by two independent
pricings; each round also announces itself as `agent.usage`.

## The locked core

`app.Open` reads `%APPDATA%/Postulator/master.key`, or refuses when `master.key.pw` says a
master password wraps it. A locked `Core` carries the event relay and an empty `kit`: no
store, no services, no engine. `Unlock` derives the key with Argon2id, unwraps the DPAPI
blob, opens the store and composes the whole application into that `kit`; `Lock` stops the
engine, the scheduler and the agent, closes the store, zeroes the key and puts the zero
`kit` back. Because the Wails services are bound once, before any of that exists, each one
resolves its use case per call through a `Source[T]` that refuses with `LOCKED` while the
`kit` is empty. A backup is the same composition in reverse: the store copies itself into a
plain snapshot through the SQLite online backup API, the archive seals it under the export
password, and a restore copies it back into the encrypted database and composes the core
again from what it read.

## WordPress companion plugin

`wp-plugin/postulator-companion` **1.2.0** (PHP ≥ 8.1, WP ≥ 6.4, no dependencies) serves
`/wp-json/postulator/v1` and advertises `bulk seo_meta seo_meta_read content_hash raw
preview`; every permission callback requires `edit_posts`, and the password must belong to an
**administrator**, so **multisite is unsupported in v2.0**. A capability the manifest does not
name is refused from the cached manifest before any request, so a site still on 1.1.0 answers
`plugin_outdated` for the SEO read rather than a 404.

| Route | Purpose |
|---|---|
| `GET /manifest` | version, capabilities, detected SEO plugin, WP version, site URL |
| `GET /content` | keyset page over posts then `product_cat` terms: hash, links, h1, meta |
| `PUT /seo-meta/{id}` | writes the SEO fields present in the body, deleting a key given an empty value; posts only |
| `GET /seo-meta/{id}` | the five SEO fields the post holds and the detected SEO plugin, behind the `seo_meta_read` capability; posts only (since 1.2.0) |
| `GET`/`PUT /content/{id}/raw` | raw `post_content` and its sha256; the write is a compare-and-swap returning `409 hash_mismatch` on a stale hash |
| `POST /content/{id}/preview` | an hour-long link that renders a draft through the theme; the token is kept only as its sha256, and `posts_results` flips that one post to `publish` for that request, uncached and noindex (since 1.1.0) |

| Field | Yoast | Rank Math | none |
|---|---|---|---|
| title | `_yoast_wpseo_title` | `rank_math_title` | `_postulator_seo_title` |
| description | `_yoast_wpseo_metadesc` | `rank_math_description` | `_postulator_seo_description` |
| canonical | `_yoast_wpseo_canonical` | `rank_math_canonical_url` | `_postulator_canonical` |
| ogTitle | `_yoast_wpseo_opengraph-title` | `rank_math_facebook_title` | `_postulator_og_title` |
| ogDescription | `_yoast_wpseo_opengraph-description` | `rank_math_facebook_description` | `_postulator_og_description` |

Raw writes rely on that role's `unfiltered_html` rather than removing kses filters. With no
SEO plugin the head is replaced, not appended, and every value is escaped on the way out.

See `docs/CONTRACTS.md` for the wire shapes and `docs/superpowers/specs/` for the full
design.
