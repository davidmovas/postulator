# Vision

## The loop

Postulator builds **one structured model of a client's site** and **edits the site's content
through it**.

One client, an SEO specialist rather than a developer, runs dozens of WordPress and WooCommerce
sites, each with up to a few thousand pages he builds with his own tools: a CSV import of variable
products, code snippets, the theme. He keeps the plan in xlsx workbooks: entities, keywords with
volumes, notes. Postulator reads the site as it is (every page of every type: its identity, its
fields, its body as blocks, what its render adds, its links by type, its JSON-LD, its place in the
hierarchy), reconciles it with his workbooks and his entity graph, and gives the AI writer the
full picture of a page before it writes a field: the page's type, the entity, the breadcrumb path,
the children, the siblings, the products the page lists, the keywords, the anchors, the notes, the
instruction for that field. The writer improves what exists, block by block, and links it as the
graph says: up to parents, down to children, sideways only to siblings the graph relates, never
outside it. A page that does not exist yet is one case of the same model.

That is the loop. Observe the site into the model, import the plan, reconcile the two into
findings a person or the agent resolves, write the fields a page type allows with the context
pack, place the links, publish under the hash of what was read, read the site back. Every feature
earns its place by serving that loop.

## Who uses it

- **The client**, through a desktop UI: the site as a model, a canvas over the graph, page types
  and field scopes he names and configures, findings, run monitors, quality reports.
- **The in-app agent**, through the same use cases exposed as tools: read and explain his data,
  observe a site, resolve findings, edit the graph, the types and the scopes, start and watch
  pipelines, create schedules. The agent can do everything the UI can do, because both call the
  same application layer, and the client does very little by hand.

## What it is

A single Windows desktop application. Wails v3 shell, Go backend, local SQLite, no server, no
account, no cloud state. The site owns its theme, its snippets and its structure; we own the model
of it. Locally we keep that model: the raw body and the field values of every page, current and
base (the last version both sides agreed on, or the last we wrote), the extraction from its render
(regions, typed links, JSON-LD facts, the breadcrumb trail, headings, word counts), the findings,
the run artifacts under a retention policy. We never keep the rendered HTML. Every write to the
site goes against the hash of what we read and keeps what it replaced beside it, so a revert puts
back exactly that.

Long work runs on a durable engine: a run is a sequence of steps with checkpoints, artifacts and
a sequenced event log, so a crash resumes from the last completed step instead of leaving zombie
rows and half-written pages behind.

## Boundaries

- **Windows only**, amd64, CGO disabled.
- **English content only.**
- **Per site.** The model, the graph and the page map belong to one site. No cross-site links.
- **Adaptive, never site-specific.** No theme name, CSS class or site path in Go or PHP. Structure
  comes from the difference between raw and render, from schema.org, ARIA and semantic HTML, and
  from REST; what cannot be derived is a per-site setting with a default.
- **The site's structure is read-only.** Slugs, prices, stock, SKUs, statuses, taxonomies, the
  WordPress parent and a variation's attributes are never written to an existing page; what the
  render adds (breadcrumbs, grids, menus) is read and understood, never written.
- **Never outside the graph.** A contextual link we write has an approved edge behind it or it
  does not get written. Every other link type is counted, not owed.
- **No markers.** Nothing we write carries a class, an attribute or a comment of ours; we remember
  our blocks by their hashes.
- **Zero footprint.** Nothing on a client's site names or identifies this program: not the plugin,
  its namespace or its keys, not the User-Agent, not a media file, not the HTML.
- **Secrets stay local**: DPAPI-protected master key, AES-GCM columns, full-database encryption,
  optional master password. Nothing sensitive reaches a log, an artifact or a plain column.
- **Budgeted.** Every model call is priced and recorded. A run carries a hard cap and pauses
  itself rather than overspending.

## Non-goals

- Not a general-purpose SEO suite: no rank tracking, no backlink research, no competitor
  analysis, no keyword research product.
- Not a CMS. The site owns its theme, its snippets and its structure; we edit page content and
  fields through the model, and only the fields a page type's scope allows.
- Not a site builder. We create no products, no terms, no menus; a page builder's content is
  detected and left read-only.
- Not multi-user, not a SaaS, not a web service. No accounts, no tenancy, no billing.
- Not a hosted agent platform. The agents are local, scoped to this data, and every write they
  attempt is confirmable.
- Not multi-language. Not macOS or Linux. Not multisite.
- No MCP server.

## Why rebuild rather than refactor

v1.6.2 had one pattern worth keeping (`StartX → taskID → Get/List/Cancel`) and little else: Wails
calls that blocked for minutes, an event bus that never reached the UI, no per-step persistence, a
page map missing H1, meta, anchors and page type, hierarchy stored three different ways and
drifting, SEO metadata generated and then never written, plaintext secrets, and one test file in
the whole repository. The failure was structural, so the structure is what changed. The 2026-10-17
redesign is not a rebuild: it extends v2 with the site model and keeps its engine, its adapters
and its import.

## Where the detail lives

- The full design: `docs/superpowers/specs/2026-09-17-postulator-v2-design.md` (the site model is
  part 16).
- The phase plan: `docs/superpowers/plans/2026-09-17-postulator-v2-roadmap.md`; the site model's
  four stages: `docs/superpowers/plans/2026-10-17-phase-14-site-model.md`.
- The shape of the code: `docs/ARCHITECTURE.md`.
- Where the work stands: `docs/STATUS.md`.
