# Vision

## The loop

Postulator turns an **entity graph** into **graph-compliant WordPress pages**.

One client, an SEO specialist rather than a developer, runs a network of WordPress
sites. Each site has an entity graph (what the site is about and how those things
relate) and a page map (what pages exist or should exist). Postulator generates the
pages so that their internal links *prove* the graph: up to parents, down to own
children, sideways only to siblings the graph explicitly relates, never outside it.

That is the loop. Import or build the graph, plan the pages, generate them, link them,
check the links against the graph, publish to WordPress, read the site back. Every
feature earns its place by serving that loop.

## Who uses it

- **The client**, through a desktop UI: a canvas over the graph and page map, run
  monitors, quality reports.
- **Local AI agents**, through the same use cases exposed as tools: read and explain
  his data, edit the graph and templates, start and watch pipelines, build a graph from
  scratch for a bare site, create schedules. An agent can do everything the UI can do,
  because both call the same application layer.

## What it is

A single Windows desktop application. Wails v3 shell, Go backend, local SQLite, no
server, no account, no cloud state. WordPress is the source of truth for content;
locally we keep metadata, content hashes and run artifacts under a retention policy.

Long work runs on a durable engine: a run is a sequence of steps with checkpoints,
artifacts and a sequenced event log, so a crash resumes from the last completed step
instead of leaving zombie rows and orphan WordPress posts behind.

## Boundaries

- **Windows only**, amd64, CGO disabled.
- **English content only.**
- **Per site.** The graph and page map belong to one site. There are no cross-site
  links, ever.
- **Never outside the graph.** A generated link either has an approved edge behind it
  or it does not get written.
- **Secrets stay local**: DPAPI-protected master key, AES-GCM columns, full-database
  encryption, optional master password. Nothing sensitive reaches a log, an artifact or
  a plain column.
- **Budgeted.** Every model call is priced and recorded. A run carries a hard cap and
  pauses itself rather than overspending.

## Non-goals

- Not a general-purpose SEO suite: no rank tracking, no backlink research, no
  competitor analysis, no keyword research product.
- Not a CMS. WordPress owns the content; we own the plan and the proof.
- Not multi-user, not a SaaS, not a web service. No accounts, no tenancy, no billing.
- Not a hosted agent platform. The agents are local, scoped to this data, and every
  write they attempt is confirmable.
- Not multi-language. Not macOS or Linux.
- No MCP server.

## Why rebuild rather than refactor

v1.6.2 had one pattern worth keeping (`StartX → taskID → Get/List/Cancel`) and little
else: Wails calls that blocked for minutes, an event bus that never reached the UI, no
per-step persistence, a page map missing H1, meta, anchors and page type, hierarchy
stored three different ways and drifting, SEO metadata generated and then never
written, plaintext secrets, and one test file in the whole repository. The failure was
structural, so the structure is what changed.

## Where the detail lives

- The full design: `docs/superpowers/specs/2026-09-17-postulator-v2-design.md`.
- The phase plan: `docs/superpowers/plans/2026-09-17-postulator-v2-roadmap.md`.
- The shape of the code: `docs/ARCHITECTURE.md`.
- Where the work stands: `docs/STATUS.md`.
