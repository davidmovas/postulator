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
eighty-seven tools in all since `graph_create_entities` landed on 2026-09-22.

`RunsService.ListItems` carries `retryable` and `retryBlockedReason` since 2026-09-19, computed
from the current step's `Requires` against the item's purged artifacts; `inputs_expired` is the
only value the reason takes and `Engine.RetryStep` refuses with the same fact before it touches
the item. `frontend/src/generated/vocab.ts` is the second generated TypeScript module: `task vocab`
renders every string union, the five sort field lists and six derived groupings from the Go const
blocks in declaration order, and a byte-comparing test fails when it is stale.

The last gate run on 2026-09-21: `task events`, `task vocab`, `task bindings` (15 services, 116
methods), `gofmt -l .`, `task check:go:comments`, `go vet`, `golangci-lint` (0 issues, plain and
`--build-tags uiharness`), `go test -count=1 -race -covermode=atomic ./...`, `go run
./cmd/covergate` (84.98% domain+application, 86.57% total), `npm run typecheck`, `vitest` (919
tests in 90 files), the frontend import bans, `task build`, `task ui:walk` (64 routes) and `task
package` (the 16 MB installer) are green. The docker suites (`task e2e:test`, `task e2e:full`,
`task e2e:full:noplugin`) were last green on 2026-09-19 and are run by hand before a tag.
`llm.usage` was registered as a run event while the ledger published it without a run sequence,
so every model call with usage failed as soon as a window was attached, since phase 1; no gate saw
it because the Go suite fakes the publisher and the harness seeds before the window exists. It is
fixed, and a release check now starts a run after the window is up. The user's first walk on
2026-09-21 found a working OpenAI key reported as rejected (the test probed the costliest catalog
row and every 403 was called a bad key), the whole window scrolling on the template editor (a
`sr-only` input escaping its label), and asked for drag-and-drop, a centred Settings column, a
roles table in two columns and an export format select; all landed the same day, with the catalog
re-verified against the providers' pages and a reasoned default model per role.

A second walk on 2026-09-21 found six things and three roots. The provider test called a working
OpenAI key rejected, because it asked a reasoning model for one output token; output budgets now
carry a reasoning allowance from the catalog row and `reasoning_effort` survives a catalog edit
(migration 0021). `cx` cannot beat a component's own base class, so every `w-` and `h-` a
caller passed to `Input` was dead; `Input` has a size scale and a test refuses the next one.
A stored score of zero means unscored and reads as a dash. The Graph screen's pill, node badges,
outline grid, review queue and propose dialog were rebuilt around those. `ProposeFromPages` and
`ProposeRelated` now name their step, which both attributes their spend and lets the harness
answer them for the first time.

**The agent was rebuilt on 2026-09-22** after the client's walk found it unable to complete a
single tool call. Four faults and their roots are in `DECISIONS.md`: the spend was double counted
on every tool round and charged cached input at the full rate, so $0.085 of real work read as
$0.30; sixty tools handed the model a DTO written for the window, with no enum and no description,
so every domain choice was a guess; arguments were only decoded after the client approved them,
and a result that arrived while the conversation was answering was dropped; the markdown reader
covered the common shapes and broke on the rest. `graph_create_entities` makes a tree one decision,
the runner retries a rate limit instead of dying on it, the prompt lets the model correct itself
and answer in the language it was asked in, and `POSTULATOR_OPENAI_KEY` points the UI harness at a
real provider. Verified on a live OpenAI key against the seeded site: an entity tree and a template
each landed on the first call, from one card each.

**The client's first run against a real WordPress on 2026-09-22 found the hierarchy broken and
five faults behind it**, all fixed and recorded in `DECISIONS.md`. Seventeen imported pages were
published at the top level because `parent: 0` meant both "deliberately at the top" and "I do not
know", because every item of a run shared one `created_at` and therefore published in random UUID
order, and because nothing compared the address WordPress answered with against the one that was
asked for. A `sync_site` after such a publish overwrote the imported plan with the damage. The
publish step now derives the parent from the path, holds an item whose parent is not on the site
yet, compares the write against the plan and repairs once before pausing for a human; the sync
records the site's view beside the plan instead of over it; `repair_hierarchy` and the `repair`
run kind move a page without rewriting it, and `PagesService.Delete{onSite}` can take one off the
site into the WordPress trash. Migrations 0023 and 0024 add `run_items.seq` and the five
`pages.wp_*` mirror columns.

**The Graph carries page state since 2026-09-22.** `LoadGraph` answers a state per mapped page —
its status, whether a run is working on it and whether the site disagrees with the plan — and the
map fills each node by it while the outline names it in a badge; the toolbar filters by state with
counts and the legend explains each one.

**The import reads a whole workbook since 2026-09-22.** Inspect names every sheet, a mapping picks
which to read and rows carry the sheet and line they came from; `indentColumns` builds the path
from the column a cell sits in and `noHeader` addresses columns by their spreadsheet letters.
`imports_preview` answers the agent with a bounded summary rather than a report the 16 KB
tool-result cap would cut into invalid JSON. `samples/messy-sheets.xlsx` is the client-shaped
workbook that exercises all of it: seven sheets, about four thousand rows, three shapes and four
deliberate faults.

**Phase 13, the product frontend, landed on 2026-09-20 and 21** (plan:
`docs/superpowers/plans/2026-09-20-phase-13-frontend.md`). Every screen sits on one screen
contract (`ui/screen.tsx` with `Toolbar`, `Tabs`, `Segmented`, `Menu`, `Kbd`), the window is
frameless with its own title bar (site pill, search on `F`, dock toggle, window controls), the rail
is icon-over-label in three sections, and the four missing screens exist: Overview
(`/s/:siteId/overview`, readiness, run summary, tiles, depth, edge coverage), Import (a four-step
wizard over `Import.Inspect/Preview/Apply`, a dropped sheet lands in step one, export as `.xlsx`
or `.csv`), Schedules (interval or cron, shown in UTC,
targets rather than a kind because a schedule has none), Reports (site, runs, pages). Sites,
Pages, Runs, Templates, Graph and Linking were re-framed onto the contract; the graph's legend
became a hover card over the node. Settings is six tabs by intent (Models, Runs, Agent, Browser,
Security, About) with human labels, units in the control and one collapsed Advanced card per tab;
a test places every declared key exactly once. Every external link opens in Tor Browser only,
through `BrowserService.Open` over `internal/adapters/browser/tor` (auto-detected or
`browser.torPath`; `tor_missing` leads to the Browser tab). The agent turn is settled by
`AgentService.Status` and `agent.done.code`, never by an error string, so a fast failure shows an
error row at once and Stop stops; a live lock shows the gate in place. The app refuses to run
twice. The UI harness (`task ui:run`) composes the real window over `adapters/llm/fake`, an
in-process `wptest` WordPress and a seeded espresso site with a DevTools port, and
`frontend/scripts/shot.mjs` photographs any route; every screen was reviewed from those PNGs.

## How to run

```
task vocab                 frontend/src/generated/vocab.ts from the Go const blocks
task events                frontend/src/generated/events.ts from the Go event registry
task build                 bin/postulator.exe, builds the frontend and the bindings first
task package               the NSIS installer in bin/
task ui:run                the harness window over the fakes, seeded, DevTools on 9222
task ui:walk               a PNG per route in frontend/shots/ (task ui:shot for one route)
task ui:reset              stops the harness and deletes its home
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
- The UI harness sees every screen, but not the window chrome: dragging, snapping and the resize
  edges of the frameless window, and the close button's hover, are checked by hand.
- A preview link rotates on every issue, so two windows previewing one draft invalidate each
  other's frame; each holds its link fifty minutes and "New link" recovers it. A security plugin
  or host header that refuses framing blanks the frame without telling the parent; "Open in Tor
  Browser" sits beside it.
- A turn's streamed text is not persisted, so a window opened mid-turn adopts the running turn
  from `AgentService.Status` and sees the answer when `agent.done` lands; thirty seconds of silence
  reconciles with Go rather than guessing.
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
  The screen covers the case it can: its own write updates the cache the read came from, a return
  to the window rereads every value, and "Reread" invalidates the whole settings key.
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

0. Repair the seven pages the client's run published flat (WordPress ids 116 to 122 on the docker
   stack): a `repair` run over them moves each under its parent and WordPress recomputes the
   permalink. Nothing else on that stand is trustworthy until it is done, because a `sync_site`
   over a flat page now records the disagreement rather than adopting it.
1. A human walk of what automation cannot see: dragging and snapping the frameless window, the
   close button's hover, a real provider key and a real WordPress, Tor on a machine with a
   different install path.
2. Tag `v2.0.0`; `release.yml` publishes the executable, the NSIS installer and the companion
   plugin archive, now 1.1.0, from that tag.
3. The backlog the frontend made visible, in `docs/superpowers/plans/2026-09-20-phase-13-frontend.md`'s
   workspace notes and the decisions: a schedule kind and a per-schedule run list, cron in the
   machine's zone, `Import.Apply` and `ProposeFromPages` as runs with progress, drag-and-drop into
   Import through a Go event, the ledger screen, `settings.changed` for a declared value.

## Final review

Reviewed 2026-09-19 over phases 9-12; the gate is green. Three fixes landed: `cc69ace` fences a
confirmed tool result, which reached the model as unfenced user text, `98b7896` masks the
credential the pending action view handed back, and `01c72a4` runs a confirmed action through the
allow list, the audit row and the result cap instead of calling the registry bare. Open, none
blocking a tag: the master key is hex encoded into the SQLite DSN string, which `Lock()` cannot
zero; a failed `ImportBackup` leaves the core locked until a restart; `export.opener.complete`
would call a valid archive truncated if the tar stream were an exact multiple of `FrameSize`;
`task package` is not byte reproducible, `BUILD_DATE` being `now`.
