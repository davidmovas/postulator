# Architecture

Modular monolith, hexagonal. A pure domain, one application layer of use cases, and two
driving adapters over it: Wails services for the UI and a tool registry for agents. Both
call the same use cases, so an agent can do whatever the UI can do without a parallel
code path.

## Layers

```
cmd/postulator        Wails bootstrap only
internal/kernel       errors, paging, id, clock, log, ctx, dto, settings, middleware
internal/domain       pure logic: site graph pagemap keyword importmap template content run schedule settings llm
internal/application  use cases, UnitOfWork, tool registry, event registry
internal/adapters     sqlite, wp, llm (openai, tuning, retry, limiter, ledger, recordreplay, catalog, profiles, fake), images, secrets, importer
internal/runtime      the run engine and its step catalog
internal/transport    wails services and event bridge, agent runner and tools
internal/app          composition root, manual wiring by area in build_*.go, no DI container
wp-plugin             the WordPress companion plugin
frontend              React + Tailwind over the generated bindings; src/canvas is the 2D engine under the graph map, src/features/agent the dock, the agent screens and the confirmation cards
```

## The dependency rule

- `domain` imports `kernel` and the standard library. Nothing else. The third-party packages
  it may reach are `golang.org/x/net/html`, because the content model is an HTML document,
  and `github.com/robfig/cron/v3`, because a schedule is a cron expression.
- `application` imports `domain` and `kernel`.
- `adapters`, `runtime` and `transport` import `application`, `domain` and `kernel`.
- Only `app` imports `transport`.
- `internal/transport/agent` never reaches `internal/adapters/llm/openai`: the agent asks the
  llm port, and the composition root picks the provider behind it.
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
  BudgetExceeded External Internal Cancelled NeedsHuman Locked`), `Is` comparing by code,
  clone-and-mutate enrichment. `CodeOf` answers `Internal` for foreign errors, so everything
  is classifiable at a boundary. No stack is captured: nothing ever read it.
- `paging` — opaque base64url cursors carrying `{o,d,v,i}`, typed sort keys, a row-value
  keyset predicate with an id tie-break, and `Cut` to turn `limit+1` rows into a page.
  Offset pagination does not exist in this codebase.
- `id`, `clock`, `ctx`, `dto` — UUID v4 text, an injectable clock, the actor carried on a
  context, and the RFC3339 UTC time type every DTO uses with `dto.TimeOf` for an instant that
  may be missing.
- `log` — zap over a console core and two lumberjack files (`app.log`, `errors.log`),
  with `password`, `apiKey`, `token` and `authorization` redacted by key.
- `settings` — settings-as-code: each setting is declared next to the code that reads
  it, with its default and its validators, and the registry can describe the whole set
  as a schema for the UI.
- `middleware` — generic `Recover` and `Audit` wrappers over `func(ctx, In) (Out, error)`.

## Domain

Value types and pure functions. `site`, `graph` (a DAG on `parent` edges plus weighted
undirected `related` edges, an entity's name unique under the parent it is scoped to, and the
labels that tell a shared name apart), `pagemap` (paths, tree, index, a page's keywords and
notes, its identity and terms, the typed links and `DeriveParent`, which reads the parent from
the JSON-LD trails, the breadcrumb nav, the WordPress parent and the URL in that order, each
with its reason), `block` (the body as addressable blocks, classic or block editor, serialised
byte for byte), `render` (the extraction of a public render: the content root found by the
content's words, the regions, the trails, the typed links, the JSON-LD facts, the headings and
word counts, the builder and the robots meta), `field` (the catalog with its deny list and a
field's current and base values), `finding` (the persistent findings with their codes, severities
and options), `keyword` (the ordered list with volumes and the one cell grammar), `importmap` (the
mapping of a sheet's headers, its level and note columns, and what each column became),
`template` (spec plus link policy, resolved global → site → page by JSON merge patch),
`content` (an HTML document, link context, link insertion, compliance and structure reports),
`contextpack` (the read functions over a snapshot of the model, the writer's pack and its fit),
`run`, `schedule`, `settings`, and the `llm` model catalog types with the service tiers, the
reasoning allowance per effort, `Cost` and the purpose rules the spend report groups by.

The heart of it is `content`: `PlanLinks` turns a graph, a page index and a
`content.Subject{Site, PageID, PagePath, EntityID}` into the set of links that page owes,
`InsertLinks` places them with a decision recorded per candidate, and `Compliance` grades
the result back against the graph. The subject is the page in hand, never its entity's
canonical page, so a run's validation and the Linking screen bind "self" to the same page.

There is one answer to where an href points, and it is `pagemap.Site{Scheme, Host, Base}`,
built once per site from its `BaseURL`. `Site.Resolve` answers a normalised site path and a
kind (`path`, `same_document`, `external`, `unresolved`), and the companion plugin's
`internal_href_to_path` follows it: the host with its port, the path escaped as `EscapedPath` and
lowercased, a dot segment or an invalid escape dropped. One table, `wp-plugin/testdata/hrefs.json`,
is answered by both and by the fake, so what the plugin records and what the domain judges
cannot drift; since 2026-10-18 a relative href resolves against the document on every side,
`Site.ResolveAt(document, href)` taking the page's path up to its last slash as a browser does.
`content.LinkContext` carries that site and every classifier — insertion,
compliance, the audit and the link counter — resolves through it.

Since stage 2b one layer reads the site model for everyone who writes or advises. `contextpack`
holds the pure read functions over a `Snapshot` (the index, the types, the graph, terms, fields,
bodies, the blocks we wrote, renders, links and findings): `Facts` for one page (its type, card,
trail, children and siblings, listed products or its twin's, terms, keywords, targets, typed links
in and out, fields with their scopes, regions, notes, findings), `Summary`, `LinkForm`,
`Overview`, `Gaps` and `Nodes`; `Build` and `Fit` turn a page's facts into the writer's pack under
a budget, and `Brief` renders the overview with the link audit's totals for the agent.
`application/sitemodel` alone loads snapshots: a page's neighbourhood by ids inside one read-only
`Store.Read` for a pack, the whole site once for the brief, the tree and the gaps, and the link
audit through `reports` rather than a second computation. The writer's steps, the judge, the Pages
drawer's "What the writer will see", the agent's brief and `sites_explore` all ask it, so none of
them computes a fact differently.

## Run engine

A run is a recipe of named steps over a set of target pages, and a run kind may own that
recipe. `relink` (`resolve_context relink_page sync_back report`), `sync` (`sync_site`) and
`revert` (`revert`) hand out their own steps and refuse a recipe that disagrees; `generate`
takes the template's, or `run.GenerateRecipe()` when neither the request nor the template names
one; `custom` is the one kind exempt from every step rule; `repair` owns nothing since
2026-10-18 and cannot be started, the hierarchy being read-only, while its word stays for old
rows. The three steps a kind owns — `sync_site`, `relink_page` and `revert` — are
`PerKindStepName` values, a type of their own, because `StepName` is the vocabulary a
*template* recipe may draw from and it is rendered into `vocab.ts`, into the blank template's
recipe and into three tool enums. Fifteen steps are registered; twelve of them a template may
name.

`sync_site` is the observation. Each execution runs one batch of one stage of
`application/observe` — inventory (the plugin's manifest and outdated verdict, the post types,
the technical page ids), content (the plugin's `/content` with `since` and the types named, or
core's items type by type), store (the products and their variations), render (a few renders at
once, `settings.observe.renderConcurrency` of them, for the pages that changed or every page on a
full walk or a `renderRescan`; a page whose visit lands on another keeps only its body's state and
the landing, and every crumb whose internal link names no modelled page is looked up once with
`wp.Client.Resolve`, its final path and status kept on the crumb, before the write unit), derive
(offline: technical and read-only reasons, the parent with its reason and method through
`pagemap.DeriveParent`, a parent loop broken, the findings, the archive after a full walk, the
site's observation state; it writes only the derived page columns and the settings' observation
keys) — and keeps the `observe.State` in its checkpoint, so a crash resumes at the
batch. The first observation, one asked for with `full`, every Nth (`fullWalkEvery`) and every
one without the plugin walk the whole site; the others pass `since`. The step's adapters in
`observe_readers.go` are the only place `*wp.Client` meets the observation's reader
interfaces, which the use case declares over its own item types. Writes go the other way through
`application/edit`, the field registry, which declares a core, a store and a plugin writer over
its own types; `edit_writers.go` adapts `*wp.Client` to them as `observe_readers.go` does for the
readers, and a run builds the service per site client. `publish`, `relink_page`,
`relink_neighbors` and `revert` record the body **the site stored** as the page's current and
base, with the blocks of ours carried onto it (`block.Landed`, `block.Align`), so the next
observation raises no finding over our own write; `sync_back` records its read-back as the
current body only and reports a difference from the base.

`relink_page` places the links a page's own rules ask for against the page's own cap and
writes the body back under a hash compare-and-swap. It calls no model, so a relink, a repair
and a sync are all estimated at nothing. For a target on the site whose anchor the body lacks it
writes a plain sentence carrying the link and says so; for a target that is not on the site it
only wraps an anchor that is already there. `relink_neighbors`, after a publish, visits the
page's own targets and every page that may owe it a link (`content.MayLinkTo`: its parents, all
its descendants and its related pages), plans each on the site policy its own template
overrides, and back-fills the same way.

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
input hash of `(step, params, required artifact hashes, template version, checkpoint)`. A
sweep daemon re-arms waiting items, reclaims items whose lease expired because the process
died mid-step, releases held children whose parent reached the site, parks the children of
an item that stopped, and fails runs past their deadline. Crash recovery is one sweep at
startup, not a separate mechanism.

The queue knows the page tree. `Enqueue` orders the targets ancestors first and, in a run
that publishes, records in `blocked_by` the item of the parent page when that parent is a
planned page of the same run. The dispatcher skips a blocked item until its blocker completes
or the parent page has a `wpId`, so a child never pays for a body it cannot publish yet; when
the blocker fails, is cancelled or is held for a decision, the sweep parks the child as
`awaiting_parent` with a note naming the parent and what to do, and repeats until the
grandchildren are parked too. The release rule is the same for the dispatcher and the sweep:
the blocker's page when there is a blocker, else the parent from the page map. A regeneration
of the parent keeps the item id, so the relation holds.

Before a run costs anything, `EstimateRun` resolves the template of every target, or the
template the start is about to assign to it, runs the preflight each step declares, checks each
model role it will call (profile, API key, catalog entry) and prices every page on its own spec,
each step at its role's service tier and, on a model that reasons, with half the allowance of the
role's effort as output, both read from the same tuning policy the calls are sent with;
a finding graded `error` stops `Start`. The step that completes an item may return a notice, which
the engine keeps as the item's note and sends on `item.done`; the report step words what the
page lacks there. A page
whose validation finds an error is held as `needs_human`, and `Engine.Accept` requeues the step
with `accept=<step>` in the checkpoint, which `StepContext.Accepted()` reads.

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
`internal/application/agent`; the runner in `internal/transport/agent` calls them for every
call the model makes and `Confirm` calls them directly, so a confirmed tool is fenced, permitted
and capped by construction rather than by a second implementation. Nothing is injected through
`Deps` and nothing in `application` imports `transport`, so the dependency rule holds without a
new seam.

A turn runs on the same `llm.Client` chain as the runs. Each round streams from it, its calls
are dispatched in order, the history is saved after every complete round as
`responses/1`, and the turn ends with a budget refusal at the loop limit. Every attempt the chain
makes is its own `llm_calls` row with `step = "chat"`, written by the ledger decorator outside the
caller's cancellation, so a stopped turn is still billed for what ran; each round announces its
cost as `agent.usage`, priced with the same `Cost` the ledger uses, so the spend badge and the
ledger agree by construction.

## The model client

One chain carries every model call, composed in `build_llm.go`: `tuning` fills the effort and the
service tier of the role a call names, then `retry`, `limiter`, `ledger` and `recordreplay`, then
`openai`, our own client for OpenAI's Responses API. The tuning decorator sits outermost so that
the ledger and a fixture record what was sent. The client sends `store: false`, a strict
`json_schema` for a structured answer, an explicit reasoning effort to a model that reasons and
flex only where the catalog prices it; it falls back to the default tier once when flex has no
capacity or answers nothing within the patience, classifies an error that arrives inside a
stream like the same error over HTTP, and reads a rate limit's delay from the response headers.
A call with no tools and no cache key is kept out of the prompt cache, because a single call
would pay to write its whole prompt and never read it; the agent names its prefix family.
`openaitest` fakes the API from what a live probe recorded, and `fake` scripts a model on the port
for the harnesses, which `Config.Provider` puts in place of `openai` at the bottom of the chain.

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

`wp-plugin/postulator-companion` **1.5.2** (PHP ≥ 7.2, WP ≥ 5.7 since phase 15, proven by a function
scan on PHP 7.2 and the contract suite on WordPress 5.7.2; no dependencies, `Update URI`
so wordpress.org never offers a plugin of the same slug) serves `/wp-json/postulator/v1` and
advertises `bulk seo_meta seo_meta_read content_hash raw preview rendered meta terms settings
meta_write guarded_write`; a client below 1.5.1 (`wp.PluginMinimum`; the bundled zip is
`wp.PluginVersion`, 1.5.2, with the 1.5.1 contract) or missing one of them writes nothing through it;
every permission callback requires `edit_posts`, and the password must belong to an
**administrator**, so **multisite is unsupported in v2.0**. A capability the manifest does not
name is refused from the cached manifest before any request, so a site on an older plugin answers
`plugin_outdated` rather than a 404, and `wp.Capabilities.Outdated` names a plugin below 1.4.1 or
missing a capability for the Sites panel; 1.4.1 resolves a relative href in `/content` against
the page's URL as a browser does, where 1.4.0 used the page's parent. The post routes answer a content post only, a post of
a public post type other than an attachment; a variation, a reusable block, a navigation menu or
a template answers 404. `uninstall.php` deletes the term clock and the preview tokens and keeps
the `_postulator_seo_*` values; there is no activation hook, an upload over the installed plugin
never firing one.

| Route | Purpose |
|---|---|
| `GET /manifest` | version, capabilities, detected SEO plugin, WP version, site URL, the content post types and public taxonomies (since 1.4.0) |
| `GET /content` | keyset page over posts then terms: hash, links, h1, meta, excerpt and terms; `types` takes any listed post type or taxonomy and defaults to the four it always listed |
| `GET /content/{id}/rendered` | `the_content` over the raw inside a singular main query, theme-free, and its hash (since 1.4.0) |
| `GET /meta/{id}?keys=` | at most 50 keys asked, `is_protected_meta` keys refused with `meta_key_denied` (since 1.4.0) |
| `GET /terms/{id}` | every term of every taxonomy the post's type carries, custom and internal ones included (since 1.4.0) |
| `GET /settings` | the front, posts and privacy pages and the store's shop, cart, checkout, account and terms pages (since 1.4.0) |
| `PUT /seo-meta/{id}` | writes the SEO fields present in the body under an `expected` hash each, answering `409 hash_mismatch` naming what moved, and the hashes of what the SEO plugin stored; posts only |
| `PUT /meta/{id}` | writes post meta by key under an expected hash each, refusing the deny set, protected keys and keys the user may not edit, and answers each key's stored value and hash (since 1.5.0) |
| `GET /seo-meta/{id}` | the five SEO fields the post holds and the detected SEO plugin, behind the `seo_meta_read` capability; posts only (since 1.2.0) |
| `GET`/`PUT /content/{id}/raw` | raw `post_content` and its sha256, and `saveStable` (whether this user's whole-post save leaves the content, title and excerpt as stored; the title and excerpt since 1.5.1); the write is a compare-and-swap returning `409 hash_mismatch` on a stale hash and answers the content WordPress stored |
| `POST /content/{id}/preview` | an hour-long link that renders a draft through the theme; the token is kept only as its sha256 and carried in `preview_token` (since 1.4.0), and `posts_results` flips that one post to `publish` for that request, uncached and noindex under core, Rank Math and Yoast alike (since 1.1.0) |

| Field | Yoast | Rank Math | none |
|---|---|---|---|
| title | `_yoast_wpseo_title` | `rank_math_title` | `_postulator_seo_title` |
| description | `_yoast_wpseo_metadesc` | `rank_math_description` | `_postulator_seo_description` |
| canonical | `_yoast_wpseo_canonical` | `rank_math_canonical_url` | `_postulator_canonical` |
| ogTitle | `_yoast_wpseo_opengraph-title` | `rank_math_facebook_title` | `_postulator_og_title` |
| ogDescription | `_yoast_wpseo_opengraph-description` | `rank_math_facebook_description` | `_postulator_og_description` |

Raw writes rely on that role's `unfiltered_html` rather than removing kses filters. With no
SEO plugin the head is replaced, not appended, and every value is escaped on the way out.

The post routes read their number through `get_post()`, so a product is written there like a
page and a product category never is: a term id names whatever post shares its number, which is
why the client names the item type on every call and refuses `product_cat` before the request.
A product's other fields go through WooCommerce's `wc/v3` with the same application password, the
store's fields before the description, because saving the short description makes WooCommerce
save the whole post again.

See `docs/CONTRACTS.md` for the wire shapes and `docs/superpowers/specs/` for the full
design.
