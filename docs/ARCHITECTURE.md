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
frontend              Vite + TypeScript stub, generated bindings
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

The heart of it is `content`: `BuildLinkContext` turns a graph plus a page index into
the set of links a page owes, `InsertLinks` places them with a decision recorded per
candidate, and `Compliance` grades the result back against the graph.

## Run engine

A run is a recipe of named steps over a set of target pages. `advance(itemID)` claims an
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

## WordPress companion plugin

`wp-plugin/postulator-companion` (PHP ≥ 8.1, WP ≥ 6.4, no dependencies) serves
`/wp-json/postulator/v1`; every permission callback requires `edit_posts`, reads included.
The password must belong to an **administrator** — raw writes rely on `unfiltered_html`
rather than removing kses filters, so **multisite is unsupported in v2.0**.

| Route | Purpose |
|---|---|
| `GET /manifest` | version, capabilities, detected SEO plugin, WP version, site URL |
| `GET /content` | keyset page over posts then `product_cat` terms: hash, links, h1, meta |
| `PUT /seo-meta/{id}` | writes the SEO fields present in the body; posts only |
| `GET`/`PUT /content/{id}/raw` | raw `post_content` and its sha256; the write is a compare-and-swap returning `409 hash_mismatch` on a stale hash |

| Field | Yoast | Rank Math | none |
|---|---|---|---|
| title | `_yoast_wpseo_title` | `rank_math_title` | `_postulator_seo_title` |
| description | `_yoast_wpseo_metadesc` | `rank_math_description` | `_postulator_seo_description` |
| canonical | `_yoast_wpseo_canonical` | `rank_math_canonical_url` | `_postulator_canonical` |
| ogTitle | `_yoast_wpseo_opengraph-title` | `rank_math_facebook_title` | `_postulator_og_title` |
| ogDescription | `_yoast_wpseo_opengraph-description` | `rank_math_facebook_description` | `_postulator_og_description` |

With no SEO plugin it replaces core's head values instead of adding tags, through
`pre_get_document_title`, `get_canonical_url` and `wp_head` for description and OG; a
theme that hardcodes `<title>` is out of scope.

See `docs/CONTRACTS.md` for the wire shapes and `docs/superpowers/specs/` for the full
design.
