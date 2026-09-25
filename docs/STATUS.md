# Status

The handoff point between sessions. Read this first. The reasoning behind every phase and
every ruling is in [`DECISIONS.md`](DECISIONS.md).

**Branch:** `dev`, the development branch; `master` takes a PR from it when the owner asks. The
2026-09-25 work is on `claude/cool-archimedes-5qgsre`, to reach `dev` when the owner asks.
**Released:** `v2.1.0` on 2026-09-23; **`v2.2.0` is on `dev` awaiting the owner's build and tag**
(this session ran on Linux, so `task build`, `task e2e:full` and the sandbox walk are listed
under **Next steps** rather than under **The gate**).

## Where we are

**Phases 0 through 13 are complete, and the production hardening of 2026-09-22 and 23 with
them.** The application composes a Wails v3 window over an adiantum-encrypted SQLite store:
the entity graph, the page map, the templates, the WordPress adapter and its companion plugin,
the LLM stack, the run engine with its sixteen steps, the import, the agent, the schedules,
and fifteen bound services behind one screen contract. A master password locks the whole core,
a backup round trips through one encrypted archive, and the sweep retires artifacts and run
events on their own windows.

**2.2.0 (2026-09-24)** answers the client's second week: eighteen pages, a third of them held
for a human, links not placed, the order wrong, every retry paid for again. The reasoning is
under **2026-09-24 — 2.2.0** in `DECISIONS.md`.

- **The import never plans `/`**, and a row's keywords land on the page (`pages.primary_keyword`,
  `pages.keywords`, migration 0027) even when the row names no entity.
- **Entities are proposed for chosen pages or pasted keywords and written only once reviewed.**
  `GraphService.PreviewFromPages`, `ProposeFromKeywords` and `ApplyProposals`, the three tools
  beside them, and a three-step dialog on the Graph screen: pick pages or paste keywords, preview,
  keep the proposals you want.
- **Placeholders work on the first try.** `{primaryKeyword} {entityName} {siteName} {pageTitle}`
  are expanded per page in `ResolveForPage`, anything else is refused by `Validate`, and the writer
  gets the brief's headings back through `content.Assemble`, so `section_missing` cannot happen.
- **The pipeline stops itself less.** A truncated or malformed answer is tried again with more room
  and a repair round; the linker writes the phrases the body owes and falls back to a plain sentence
  with a warning; validation only grades and holds a page with residual errors for a decision that
  `Engine.Accept` settles in one click; the judge and the image step never fail a page on their own
  clock; a post is sent no parent.
- **The queue knows the tree.** A child is queued behind its parent (`run_items.blocked_by`,
  migration 0028), never runs before it, and is parked with an explanation when the parent stops;
  items are listed in the order they run and the table names the page each waits after.
- **Before the run costs anything**, the estimate prices every page on its own template, checks
  each model role (profile, key, catalog), prices images on the image model, runs each step's
  preflight and refuses a run a finding blocks.
- **The window follows.** Every step event carries the step's sentence, a resumed or regenerated
  run stays live, `step.started` refreshes the rows, the drawer picks its primary action from what
  stopped the page, and the header counts the pages that wait for a decision.

**2026-09-25** answers the owner's third week, on `claude/cool-archimedes-5qgsre`: the reasoning is
under **2026-09-25** in `DECISIONS.md`.

- **The built-ins ask for no image**, and a built-in nobody edited is refreshed to the new version
  on start from the copies under `domain/template/seed/superseded`; an edited one is left alone.
- **Images are drawn when a template asks for them.** The OpenAI adapter no longer sends the
  `response_format` GPT image models reject, the generate recipe carries the image step, the
  estimate warns (`images_step_off`) when a run will not draw what a page asks for, the editor
  turns the step on with the images, and the step says "placed N of M images" and why.
- **A template picked at the start is assigned to the chosen pages** (the drawer, schedules and
  `runs_start`); it used to be a label.
- **An owed link is placed or said.** The brief and `repair_links` owe every target within the
  budget, down and sideways included, the children section follows the effective rules, a link to
  a page not on the site is named (`target_not_published`), and after a publish every neighbor
  that owes the page a link gets it, in a plain sentence when it has no anchor, or says why not
  (`relink_phrase_templated`, `neighbor_link_missing`). A neighbor is planned on its own rules
  over the site policy.
- **A finished page says what it lacks**: the report's notice is the item's `note` and rides on
  `item.done`; the item table marks it **Done, check** and the run header counts such pages.
- **The Linking screen is honest**: every owed link has a state, a planned page reads **Not
  written yet** instead of missing everything and being an orphan, and what waits for a page to
  be written or published is counted apart from what is missing.

`relink`, `repair`, `sync` and `revert` own their recipes. The companion plugin is **1.2.0**.
`TestTheClientLoopFromTheSamples` drives the whole loop from the client's own workbooks against
docker, and `TestAChildWaitsForItsParentAndGoesOnOnceTheParentIsRegenerated` now stops the parent
by exhausting the writer, because an incomplete draft is tried again instead of failing at validate.

## The gate

Green on 2026-09-25 over the head of `claude/cool-archimedes-5qgsre`, on Linux, where
`cmd/postulator`, `internal/app`, `adapters/browser/tor` and `adapters/secrets/{dpapi,masterkey}`
build only under `GOOS=windows`, `transport/wails` needs GTK to build its tests, and `TestDSN` in
`adapters/sqlite` fails on the path separator alone.

- `gofmt -l .` silent, the comment check of `task check:go:comments`, `GOOS=windows go vet ./...`
  and `GOOS=windows go vet -tags e2e ./internal/e2e/...`.
- `golangci-lint run` (v2.13.2 built with go1.27, run with `GOOS=windows`) and the e2e sources
  0 issues; `go test -race -count=1 -p 2` green over every package that builds on Linux.
- `go run ./cmd/covergate`: **domain+application 86.17% of 7353** (gate 80%), **total 86.80% of
  18013** (gate 70%).
- `wails3 generate bindings`: **15 services, 122 methods**; `task events` and `task vocab` leave no
  diff; migrations up to **0028**, none added.
- **93 tools**, 78,973 bytes of schema against the 79,000 the registry test allows.
- `npm run typecheck` clean; `npx vitest run` **1162 tests in 115 files** over two projects.

## How to run

```
task vocab · events · bindings    the three generated TypeScript surfaces
task build                        bin/postulator.exe, the frontend and the bindings first
task package · plugin:zip         the NSIS installer and the plugin archive, into bin/
task ui:run · walk · shot · reset the harness window, seeded, DevTools on 9222
task ui:lint                      lint and tests behind the uiharness build tag
task lint:e2e                     the build-tagged sources golangci-lint run skips
task e2e:up · test · full · full:noplugin · down     the suites' WordPress on 8088
task sandbox:up · down · reset    the owner's WordPress on 8089, bare unless E2E_PLUGIN=1
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
  `relink_neighbors` does the same for every page that may owe a newly published page a link, so
  publishing the root of a large tree resolves every descendant once.
- **The estimate prices the linker on up links and the lead keyword**, not on the down and
  sideways phrases `repair_links` now writes when the body lacks them (256 output tokens each).
- **The image settings apply after a restart**: the adapter and the price reference are built
  with the composition, and the Settings copy says so.
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

1. **Owner, on Windows and docker, for the 2026-09-25 work:** `task build`, `task ui:lint`,
   `task e2e:full` (the client loop now asks the linker for down and sideways phrases and sees
   `target_not_published` on a parent written before its children), then a sandbox walk with a
   real provider: a hub and a child in one run, where the hub links to the child and a missing
   link reads **Done, check** on the row; a template with AI images picked in the start drawer,
   which moves the pages to it and draws the images; a planned page on the Linking screen, which
   reads **Not written yet**; and an untouched built-in, which shows as version 2 without images.
2. **Owner, before the 2.2.0 tag, on Windows and docker:** `task build` (the bindings and the
   frontend under the real toolchain), `task ui:lint`, `task e2e:full` (the held-parent scenario
   now exhausts the writer), and `task package`; then a walk on the sandbox with a real provider:
   import a workbook with a `/` row and keywords, preview and apply entities for one branch and for
   a pasted keyword list, a template with `{primaryKeyword}` in a heading, a run over a parent and
   two children with one page held at validate and accepted, a regenerated parent whose children go
   on by themselves, and the start dialog refusing a run whose provider has no key.
3. The residue above: the denied tool row's decision, the four narrow-width UI items, the ledger
   screen, `ProposeFromPages` and `Import.Apply` as runs, and `settings.changed` for a declared
   value.
