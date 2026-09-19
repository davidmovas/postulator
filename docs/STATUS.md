# Status

The handoff point between sessions. Read this first. The reasoning behind every
phase is in [`DECISIONS.md`](DECISIONS.md).

**Branch:** `rewrite/v2`, cut from `dev`. **Target release:** `v2.0.0`, not tagged yet.

## Where we are

**Phases 0 through 12 are complete.** The application composes a Wails v3 window over an
adiantum-encrypted SQLite store: the entity graph, the page map, the templates, the WordPress
adapter and its companion plugin, the LLM stack, the run engine with its twelve page steps, the
import, the agent, the schedules, the fourteen bound services and their TypeScript surface. A
master password locks and unlocks the whole core, a backup of the database round trips through
one encrypted archive, and the sweep retires artifacts and run events on their own windows.

The bound surface is a hundred and thirteen methods since 2026-09-19: `SettingsService.ProviderKeys`
and `DeleteProviderKey` report and revoke an LLM key as a boolean, `SitesService.TestConnection`
probes a site before it is saved and after it is edited, `RunsService.ListArtifacts` names the
artifacts a run item holds without their blobs, and `RunsService.Estimate` prices a run before it
is enqueued. `sites.changed`, `schedules.changed` and `settings.changed` join the three `*.changed`
events, and a run's deadline is the `runs.deadline` setting rather than a constant.
`ReportsService.LinkAudit` and `LinkAuditPage` answer, per mapped page, the links the graph asks
it to carry and whether a stored link satisfies each one; `reports_link_audit` and
`reports_link_audit_page` hand the same reads to the agent. An edge
carries a `reason` since migration 0019: the related proposer keeps the model's sentence, the
page proposer derives one from the page paths, and `AddEdge` accepts one.
`AgentService.RenameConversation` and `DeleteConversation` manage a conversation, and a first
message titles an untitled one. `PagesService.PreviewLink` answers where a page can be seen as the
site's theme renders it: a published page's public address, or an hour-long link the companion
plugin 1.1.0 signs for a draft; `pages_preview_link` hands it to the agent as a `write` tool,
eighty-six tools in all.

`RunsService.ListItems` carries `retryable` and `retryBlockedReason` since 2026-09-19, computed
from the current step's `Requires` against the item's purged artifacts; `inputs_expired` is the
only value the reason takes and `Engine.RetryStep` refuses with the same fact before it touches
the item. `frontend/src/generated/vocab.ts` is the second generated TypeScript module: `task vocab`
renders every string union, the five sort field lists and six derived groupings from the Go const
blocks in declaration order, and a byte-comparing test fails when it is stale.

The last gate run on 2026-09-19: `task events`, `task vocab`, `task bindings`, `gofmt -l .`,
`task check:go:comments`, `go vet`, `golangci-lint` (v2.13.2, 0 issues, the build-tagged sources
included), `go test -count=1 -race -covermode=atomic ./...`, `go run ./cmd/covergate`,
`npm run typecheck`, `vitest`, `task build`, `task plugin:lint`, `task e2e:test`, `task e2e:full`
and `task e2e:full:noplugin` are green. `task package` was last run on 2026-09-18, when
`bin/postulator.exe` opened its window and answered `health.ping`.

The frontend is a React application being built on that surface. Its toolchain, typed data layer,
event bridge, run-event replay, app shell, router and lock gate landed in `0a9a97e`, with vitest
over the data layer wired into CI; the design system, the sites, pages, runs and templates screens
followed. The graph screen (`/s/:siteId/graph`) draws the topical map on an own Canvas 2D engine
(`frontend/src/canvas`): a folding left-to-right tree over the parent edges, related arcs, badges
for the problems a node carries, three levels of detail by zoom, lenses, search, an outline view, an
inspector that edits an entity, its anchors and its canonical page, connect and drag-to-reparent
with the cycle refusal shown, a proposal queue with keyboard and bulk decisions, the model actions
with a real cancel, a pulse on what changed, and a link-proof overlay. The Linking screen
(`/s/:siteId/links`) reads `LinkAudit` and `LinkAuditPage`: meters, a filter rail, the pages worst
first, a panel naming every link a page owes and carries, and a relink run over the selection.
The agent (`frontend/src/features/agent`) is a dock on every screen bound to the site in the route,
toggled with Ctrl+J and resized by its edge, plus `/agent` for every conversation grouped by site
and `/agent/inbox` for the actions waiting on approval. The transcript streams text and tool calls,
and a write stops at a confirmation card that describes the action in sentences, one describer per
tool family, `dangerous` visibly apart. "Ask the agent about this" sits on the graph inspector, the
page drawer, the template editor and the run review. The template editor starts blank or from a
copy, edits a page's own changes through `?page=`, types the two step settings, sets the site
default and draws the page a template asks for. A page drawer and the run review open the page on
the site: a published page by its address, a draft through the plugin's link in a frame with
desktop, tablet and phone widths. Not built yet: the overview, reports, schedules, import and
settings; every one of them is a `NotBuilt` panel in `router.tsx`.

## How to run

```
task vocab                 frontend/src/generated/vocab.ts from the Go const blocks
task events                frontend/src/generated/events.ts from the Go event registry
task build                 bin/postulator.exe, builds the frontend and the bindings first
task package               the NSIS installer in bin/
task plugin:zip            bin/postulator-companion.zip
go test -count=1 -race -covermode=atomic -coverprofile=coverage.out ./...
go run ./cmd/covergate     enforces the coverage gates on that profile
$(go env GOPATH)/bin/golangci-lint.exe run
task e2e:up                the docker WordPress stack on http://localhost:8089
task e2e:test              the plugin suite against it
task e2e:full              the whole loop: sync, import, five drafts, relink, backup
task e2e:full:noplugin     the same loop against a stack with the plugin deactivated
task lint:e2e              the build-tagged sources, which `golangci-lint run` skips
task e2e:down              stops the stack and drops its volumes
```

## Phases

| # | Phase | Status |
|---|---|---|
| 0 | Wipe, scaffold, kernel, CI, docs | done, reviewed |
| 1A | SQLite store, migrations, unit of work, secrets | done, reviewed |
| 1B | Wails contracts spike, events, error conversion | done, reviewed |
| 2 | Domain core: graph, page map, templates, settings | done |
| 3A | WordPress adapter and `wptest` | done, reviewed |
| 3B | Companion plugin and the docker e2e harness | done, reviewed |
| 4 | LLM port, gollem client, catalog, ledger, limiter, retry | done |
| 5 | Run engine, checkpoints, recovery | done |
| 6 | Content factory: document, links, compliance, first steps | done |
| 7 | Remaining steps, images, sync, site reports | done |
| 8 | Import and export of the client page map | done |
| 9 | Tool registry, agent runner, confirmations | done |
| 10 | Schedules and the ticker | done |
| 11 | Wails services, event bridge, TypeScript generation | done |
| 12 | Master password, backup, retention, e2e, release | done |

Module coverage is 87.6% of 14439 statements; `domain` + `application` sit at 86.1%.

## Known gaps

- Application events published before `cmd/postulator` connects the relay to the Wails event
  manager are dropped, by design: nothing listens before the window exists.
- Backward paging exists in every repository (`paging.Request.Before`) but not at the use-case
  boundary, because `dto.ListRequest` carries one cursor; the client replays the cursor it used
  to reach the current page.
- Step `params` keys are `allowErrors` on `validate` and `iterations` on `repair_links`; the
  shipped seeds carry none, so both take their defaults.
- `lefthook` is not installed on the development machine. Install it with
  `go install github.com/evilmartians/lefthook@latest && lefthook install`.
- **`golangci-lint` on this machine must be run from `$(go env GOPATH)/bin`.** A scoop shim
  earlier on `PATH` is v2.11.4 built with go1.26 and refuses a `go 1.27` module outright.
- `ProposeFromPages` is one bound call that makes one model call per forty unmapped pages, so a
  site with thousands of unmapped pages holds the window's call for minutes. The dialog says how
  many calls it will make, counts the seconds, and its stop aborts the call so earlier batches stay;
  turning it into a run with progress events is the proper fix.
- The map, the agent dock and screens, the confirmation cards, the template skeleton and the
  preview frame cannot be seen from this machine's automation, so their rendering has been checked
  by tests over the pure modules, the typecheck and by reading the code, not by looking at a window.
- A preview link rotates on every issue, so two windows previewing one draft invalidate each
  other's frame; each holds its link fifty minutes and "New link" recovers it. A security plugin
  or host header that refuses framing blanks the frame without telling the parent; "Open in your
  browser" sits beside it.
- A turn's streamed text is not persisted, so a window opened mid-turn sees the answer only when
  `agent.done` lands; a turn silent for ninety seconds is shown as stalled with the saved rows.
- Nothing purges an artifact for an item that never published, so every failed or unpublished item
  stays retryable. The dead end `retryable` reports is narrow by construction: a published item
  past its retention window whose current step consumes `body_html`, `draft` or `images`. No
  shipped recipe puts such a step after `publish`, so only a custom recipe reaches it today.
- Core REST filters `modified_after` on the site-local `post_modified` while returning the UTC
  `modified_gmt`; the sync uses the plugin's `/content?since=`, which compares `post_modified_gmt`.
- `internal/adapters/wp/e2e` carries its own small HTTP client rather than `internal/adapters/wp`,
  which had not landed when it was written. `internal/e2e` goes through the real adapter.
- `content.Assess` has no Wails method; `ReportsService.JudgePage` binds `content.Judge` and runs
  it synchronously, and `PageReport` still reads what a run already recorded.
- `internal/application/agent` is exercised mostly through `internal/transport/agent`, so the
  profile under-reports it without `-coverpkg`; the aggregate gate still passes and the behaviour
  is covered. The package now carries its own tests over the confirmation fence and the views.
- `ledger.List` has no caller: `ModelsService.UsageSummary` answers from the aggregate, and a
  per-call ledger screen is what would read the list.
- `content.Compliance` classifies an absolute link to the site's own host as `external`, because
  it passes an empty host; the link audit takes the host from the site's base URL and classifies
  the same link as `graph`. A run's validation and the Linking screen can therefore disagree on
  an absolute own-host link until the step learns the host too.
- The link audit rebinds "self" to the audited page. A second page mapped to the same entity is
  audited with that entity's targets, and its link to the entity's canonical page reads as
  `unknown_internal`, not `self`; `Compliance` would call it `self`.
- `LinkAudit` resolves the template of every mapped page, one `ResolveForPage` each, so a site of
  five thousand mapped pages costs about two seconds. The frontend caches it for thirty seconds
  and refreshes on events, not on focus.
- `settings.changed` is published by `models.SetProviderKey` and `models.DeleteProviderKey` only.
  `SettingsService.Set` writes a declared value at the transport, which is the one layer that does
  not publish, so a second window on the settings screen does not learn that `runs.workers`
  changed. Closing that needs an application settings use case, which the service does not have.
- `SitesService.TestConnection` builds a throwaway `wp.Client` for the candidate rather than the
  one `registry.Client` caches, because a candidate has no row to key the cache on; a test
  therefore neither warms nor invalidates the cached client.
- `TestConnection` reports `hasPlugin` from the REST namespaces the root advertises, not from the
  plugin manifest. `SyncService.CheckPlugin` is what reads the manifest and writes the site row.
- The docker e2e stack is never run in CI: `windows-latest` cannot run Linux containers, and the
  Ubuntu job exists only to lint and package the plugin. Every suite is run by hand before a tag:
  `task e2e:test` and `task e2e:full` on a default stack, then `task e2e:full:noplugin`.
- The degraded path, the client who refuses the companion plugin, is covered against a live
  WordPress by `TestTheWholeLoopDegradesWithoutThePlugin`: the plugin check reports it absent,
  every plugin-only call is refused as `plugin_missing`, the map arrives through core REST
  without archiving the products core cannot see, and the loop generates, publishes and reads
  back. Two things are simply unavailable there and say so rather than failing the item: the SEO
  meta is skipped with a warning finding, and the neighbor relink stands down, because the only
  content core REST offers to write back is WordPress's rendered output and writing that would
  replace what a human wrote. A core pull also rebuilds the path of every draft, which core REST
  reports as `/?page_id=42`; the companion plugin is what knows where a draft would land. A draft
  cannot be previewed there: `PagesService.PreviewLink` answers `INVALID` with `details.code =
  plugin_missing`, the refusal every plugin-only call carries, while a published page still gets
  its address, and the UI offers WordPress's own draft address, which asks for a login.
- **Migration 0016 runs outside goose's transaction.** Rebuilding `run_items` means dropping it,
  and a drop with foreign keys on performs an implicit delete that would cascade into `artifacts`
  and `step_execs`. The statements are wrapped in an explicit `BEGIN`/`COMMIT`, so the rebuild is
  atomic, but a crash between that commit and goose recording the version leaves a database the
  next start cannot migrate. The recovery is the reset the startup error already names.
- A pending action keeps the arguments it will replay, so a confirmation for a tool that carries
  a credential holds that credential in the encrypted database until the action is settled. The
  event, the summary and the tool call ledger carry it masked.
- An agent tool that addresses a record by id relies on the use case to scope the write, exactly
  as the Wails services do. The use cases are where that is closed.
- **Media uploaded by `generate_images` is orphaned when the item later fails.** Accepted: an
  upload is cheap, a delete needs `force=true` to skip the trash, and a sweep that removes media
  a human may already have reused elsewhere is the worse failure.
- A backup is restored into the same installation. Carrying one to another machine works because
  the archive holds a plain database, but nothing yet checks that the site credentials inside it
  still decrypt with the new master key: the secrets are sealed with the key of the machine that
  wrote them, so they are unreadable after a restore elsewhere and have to be entered again.
- `Lock()` refuses while no master password is set, because nothing could unlock the core again.
  Setting a password while the core is locked also unlocks it, since the caller proved the old one.

## Next steps

1. Build the remaining screens: the overview, reports, schedules, import and settings.
2. Walk the agent dock, the inbox, the cards, the template editor and the page preview in the
   built window, which no automation here can see.
3. Tag `v2.0.0` when the UI lands; `release.yml` publishes the executable, the NSIS installer and
   the companion plugin archive, now 1.1.0, from that tag.
4. Close the gaps above that the UI reaches: the ledger screen and `settings.changed` for a
   declared value, which needs an application settings use case.

## Final review

Reviewed 2026-09-19 over phases 9-12; the gate is green. Three fixes landed: `cc69ace` fences a
confirmed tool result, which reached the model as unfenced user text, `98b7896` masks the
credential the pending action view handed back, and `01c72a4` runs a confirmed action through the
allow list, the audit row and the result cap instead of calling the registry bare. Open, none
blocking a tag: the master key is hex encoded into the SQLite DSN string, which `Lock()` cannot
zero; a failed `ImportBackup` leaves the core locked until a restart; `export.opener.complete`
would call a valid archive truncated if the tar stream were an exact multiple of `FrameSize`;
`task package` is not byte reproducible, `BUILD_DATE` being `now`.
