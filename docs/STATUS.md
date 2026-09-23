# Status

The handoff point between sessions. Read this first. The reasoning behind every phase and
every ruling is in [`DECISIONS.md`](DECISIONS.md).

**Branch:** `dev`, the development branch; `master` takes a PR from it when the owner asks.
**Released:** `v2.0.0` on 2026-09-23. `dev` carries the fixes after it, not tagged yet.

## Where we are

**Phases 0 through 13 are complete, and the production hardening of 2026-09-22 and 23 with
them.** The application composes a Wails v3 window over an adiantum-encrypted SQLite store:
the entity graph, the page map, the templates, the WordPress adapter and its companion plugin,
the LLM stack, the run engine with its sixteen steps, the import, the agent, the schedules,
and fifteen bound services behind one screen contract. A master password locks the whole core,
a backup round trips through one encrypted archive, and the sweep retires artifacts and run
events on their own windows.

The hardening answered four complaints from the client's own use; the reasoning is under
**2026-09-22 and 23** in `DECISIONS.md`.

- **Links.** One `pagemap.Site` answers where an href points, for a run's validation and the
  Linking screen alike, so a link the body carries is no longer inserted twice and then graded
  an error. A template's link rules replace the policy's whole, a neighbour is relinked by its
  own rules, and "self" is the page being written.
- **The agent.** A field is required only where the use case refuses its absence, a tool takes
  its own argument structs, a call ends in the transcript as one of five states (`running ok
  cut denied error`), and an answer over the cap is shortened into a readable document.
- **Reversibility.** `RunsService.RevertRun` enqueues a `revert` run that trashes what the run
  created and writes back the body and the SEO meta it replaced, holding an item rather than
  overwriting a human's edit. A finished step is recorded even while the engine stops, and a
  failed `ImportBackup` puts back the database it replaced.
- **Cost.** Every model call is its own ledger row, `agent.usage` announces each round and
  `agent.waiting` says when a provider is holding one, the stored history replays a shortened
  tool result, and the tool schemas have a ceiling with a test on it.

**After 2.0.0 (2026-09-23),** from the client's first real hour on OpenAI; the reasoning is under
**2026-09-23 — after 2.0.0** in `DECISIONS.md`.

- **A child waits for its parent.** Publish holds a child whose parent is not on the site as
  `awaiting_parent` with a sentence naming the parent, and the sweep releases it as soon as the
  parent page has a `wpId`, whoever published it. A paused run is no longer reaped, and every revival
  re-arms its deadline. A run that publishes brings the parents that are not on the site.
- **Regenerate from the start.** `RunsService.Regenerate` and `runs_regenerate` write stopped items
  again from their first step inside the same run, against the template as it is now; an item that
  already wrote to the site is refused so its revert stays exact. The drawer offers it, a run with
  failed items offers it for all of them, and a held child names its parent and offers to open or
  regenerate it.
- **Start a run picks from the page tree**, with status, search, a not-on-the-site filter, whole
  branches and the parents that will be written first.
- **Cost.** The writer is `gpt-5.6-terra`, the editor and the judge `gpt-5.6-luna`; images ask for
  `images.openaiQuality` (medium) and every image is a ledger call, so it counts in spend and budget.
- **Two docker stacks.** `task sandbox:*` is the owner's WordPress on 8089, `task e2e:*` the suites'
  own on 8088; the plugin is installed from its zip, never bind-mounted, and the suites clean up.
- **Tor Browser.** A link opens as a new tab in the Tor Browser Postulator started; one the person
  opened by hand is left alone with a toast that says what to do and copies the link.

`relink`, `repair`, `sync` and `revert` own their recipes, so Relink no longer regenerates the
page and a seeded template that names no recipe runs `run.GenerateRecipe()` instead of being
refused. The companion plugin is **1.2.0**, adding `GET /seo-meta/{id}` behind the
`seo_meta_read` capability. `TestTheClientLoopFromTheSamples` drives the whole loop from the
client's own workbooks against docker: import, eight live pages, a green link audit, a relink,
a repair, a cancel, a byte-identical revert, a trash and a backup.

## The gate

Green on 2026-09-23 over the code at `a035b62` on `dev`; every commit after it is documentation.

- `gofmt -l .` silent, `task check:go:comments`, `go build ./...`, `go vet ./...`.
- `golangci-lint run` and `task lint:e2e` 0 issues; `go test -race -count=1 -p 2` green.
- `go run ./cmd/covergate`: **domain+application 85.60% of 6673** (gate 80%), **total 86.44% of 17973** (gate 70%).
- `task bindings`: **15 services, 119 methods**; `task events` and `task vocab` leave no diff.
- **90 tools**, 75,045 bytes of schema against the 75,100 the registry test allows.
- `npm run typecheck` clean; `npx vitest run` **1116 tests in 113 files** over two projects.
- `task ui:lint` 0 issues plus the harness tests, and `task build`. The seeded window was walked on
  2026-09-23 for the held run, its child's drawer and the failed-items banner.
- The docker suites ran on the e2e stack (8088) on 2026-09-23: `task e2e:test` 25 s, the whole-loop
  package 76 s including the new held-parent scenario, and the stack held none of their pages after.
- Tor Browser 15.0.23 by hand: three https links from the adapter, one window, no dialog.

## How to run

```
task vocab · events · bindings    the three generated TypeScript surfaces
task build                        bin/postulator.exe, the frontend and the bindings first
task package · plugin:zip         the NSIS installer and the plugin archive, into bin/
task ui:run · walk · shot · reset the harness window, seeded, DevTools on 9222
task ui:lint                      lint and tests behind the uiharness build tag
task lint:e2e                     the build-tagged sources golangci-lint run skips
task e2e:up · test · full · full:noplugin · down     the suites' WordPress on 8088
task sandbox:up · down · reset    the owner's WordPress on 8089, plugin from its zip
go test -race -count=1 -p 2 -covermode=atomic -coverprofile=coverage.out ./...
go run ./cmd/covergate            the coverage gates on that profile
$(go env GOPATH)/bin/golangci-lint.exe run
frontend: npm run typecheck · npm run test:run
```

`npm run test:run` runs two vitest projects: `model` on node over `*.test.ts` and `screens` on
jsdom over `*.test.tsx`. The Go suite needs `-p 2` here, because the race detector under the
default parallelism wants more than this machine's paging file holds, and it wants the machine
to itself: the engine's deadlines are wall clock, so beside another heavy job `internal/runtime`
fails with cancelled WordPress requests that mean nothing. **Run `task build` before a tag**,
not only `npm run typecheck`: only the build regenerates the gitignored bindings.

## Phases

| # | Phase | Status |
|---|---|---|
| 0 | Wipe, scaffold, kernel, CI, docs | done, reviewed |
| 1 | SQLite store, migrations, secrets; the Wails contracts spike and error conversion | done, reviewed |
| 2 | Domain core: graph, page map, templates, settings | done |
| 3 | WordPress adapter, `wptest`, the companion plugin and the docker e2e harness | done, reviewed |
| 4 | LLM port, gollem client, catalog, ledger, limiter, retry | done |
| 5 | Run engine, checkpoints, recovery | done |
| 6 | Content factory: document, links, compliance, first steps | done |
| 7 | Remaining steps, images, sync, site reports | done |
| 8 | Import and export of the client page map | done |
| 9 | Tool registry, agent runner, confirmations | done |
| 10 | Schedules and the ticker | done |
| 11 | Wails services, event bridge, TypeScript generation | done |
| 12 | Master password, backup, retention, e2e, release | done, reviewed |
| 13 | The product frontend on one screen contract | done |
| — | Hardening 2026-09-22/23: links, agent reliability, reversibility, agent cost, relink and repair as kinds, the content steps, the client scenario | done, gate green per wave |

## Known gaps

- **`agent.Deps.Allowed` is never set**, so `Permit` can never deny in production and the
  transcript's refused tool row has no producer; `TestTheSeededConversationShowsADeniedToolCall`
  is skipped with that reason. Either per-conversation permissions get a producer, or the
  state and its copy go.
- **`LinkAudit` resolves the template of every mapped page**, one `ResolveForPage` each, so
  five thousand mapped pages cost about two seconds. The frontend caches it for thirty seconds.
- **What a revert cannot put back:** media a run uploaded, because a delete needs `force=true`
  and sweeping media a human may have reused is worse; and, against plugin 1.1.0, the SEO meta,
  because the read is refused from the manifest and `revert_meta_kept` names it instead.
- **The agent ledger is coarse and OpenAI's `Retry-After` is unreachable**: `go-openai` drops
  the response headers, so the delay is read out of the provider's sentence and honoured up to
  two minutes; `llm_calls` has no `message_id` or `round`, and `ledger.List` has no caller.
- **`ProposeFromPages` is one bound call** making one model call per forty unmapped pages, so
  thousands of them hold the window's call for minutes. Turning it into a run is the fix.
- **UI residue, all seen in the walk, none of it wrong output:** the Models table shows about
  ten characters of a model name with the dock open; the run table pushes "took" outside 960;
  the layer chooser is clipped at 960 with the dock; the graph legend covers the canvas at 1280.
- **The harness's fake WordPress serves no signed draft preview**, so the page drawer shows
  `rest_not_logged_in` there; `assertDraftPreviews` covers the real path on docker. **The
  docker stack is never run in CI** — `windows-latest` cannot run Linux containers — so every
  suite is run by hand before a tag.
- **Without the companion plugin** the SEO meta is skipped with a warning, the neighbour relink
  stands down, no draft can be previewed and a revert of an updated page pauses; the loop still
  generates, publishes and reads back.
- **Application events published before the window exists are dropped**, by design, and
  backward paging exists in every repository but not at the use-case boundary.
  **`settings.changed`** is published by `models.SetProviderKey` and `DeleteProviderKey` only,
  because `SettingsService.Set` writes at the transport, the one layer that does not publish.
- **A pending action keeps the arguments it will replay**, so a confirmation for a tool
  carrying a credential holds it in the encrypted database until it is settled; the event, the
  summary and the ledger carry it masked.
- **Migration 0016 runs outside goose's transaction**, so a crash between its explicit `COMMIT`
  and goose recording the version leaves a database the next start cannot migrate; the recovery
  is the reset the startup error names. **A backup carried to another machine** keeps its site
  credentials sealed with the key of the machine that wrote them.
- **Three review items stand, none blocking a tag:** the master key is hex encoded into the
  SQLite DSN, which `Lock()` cannot zero; `export.opener.complete` would call a valid archive
  truncated if the tar stream were an exact multiple of `FrameSize`; `task package` is not
  byte reproducible, `BUILD_DATE` being `now`.

## Next steps

1. Move the sandbox onto the new compose with `task sandbox:up` (it recreates the WordPress
   container without the plugin mount and keeps the volumes), then regenerate the seven pages the
   2026-09-23 e2e run left on it (`/components/`, `/electric-bikes/`, `/components/batteries/`,
   `/electric-bikes/cargo/`, `/electric-bikes/commuter/`, `…/hauler-cargo-max/`,
   `…/volt-commuter-500/`) with an ordinary run; they are parents of real pages, so they are
   rewritten, not deleted.
2. A human walk of the new flows on a real provider: a failed parent regenerated in place, its
   children going on by themselves, the tree picker on a real site.
3. When the owner says so: PR `dev` into `master` and a `v2.0.1` tag; `release.yml` publishes from it.
4. The residue above: the denied tool row's decision, the four narrow-width UI items, the ledger
   screen, `ProposeFromPages` and `Import.Apply` as runs, `settings.changed` for a declared value,
   and pricing images in the pre-run estimate.
