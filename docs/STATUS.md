# Status

The handoff point between sessions. Read this first. The reasoning behind every phase and
every ruling is in [`DECISIONS.md`](DECISIONS.md).

**Branch:** `dev`, the development branch; `master` takes a PR from it when the owner asks.
**Released:** `v2.3.0` on 2026-09-25 (2.2.0 and the 2026-09-25 work). **The 2026-10-02,
2026-10-03 and 2026-10-03/04 work is on `dev`**, not yet released; what its gate did and did not
run is under **The gate**.

## Where we are

**2026-10-03 and 04** answers the client's two complaints, $10 of OpenAI spend that bought little
and the Category and Subcategory of his sheets never reaching the site, and reads again the code
that had been rewritten many times. The reasoning is under the three sections of those dates in
`DECISIONS.md`.

- **OpenAI only, through our own Responses API client** (`internal/adapters/llm/openai`, faked
  by `openaitest`); gollem and the Anthropic, Gemini and go-openai SDKs are gone, and migration
  0038 drops the profiles of the removed providers. A structured answer is one strict call, a
  credit refusal waits for a person instead of five retries, `Retry-After` is read, the writer runs
  on flex and falls back to the default tier, and a single call stays out of the prompt cache.
- **Reasoning and speed per role** are settings in Settings → Models (`llm.effort.*`,
  `llm.tier.*`, `llm.flexPatience`); the effort left the catalog row (0037). The writer is paid
  for at most two truncated answers, then the page waits for a person. The estimate prices each
  step at its role's tier and reasoning.
- **The spend is in view** (0035, 0036): the ledger books reasoning, cache writes, the served tier
  and a cancelled stream; Settings → Models shows the spend by purpose, by model and tier and the
  recent calls, and a run's report what each step cost. Five spend bugs are fixed, and a price an
  override left at zero takes the built-in price.
- **The agent runs on the same chain as the runs**, with a ledger row per attempt and its own
  `responses/1` history: a conversation from before keeps its transcript and the model forgets
  it. Its tool schemas went from 85,991 to 63,304 bytes a round; loading tools on demand
  (`agent.toolLoading: deferred`, 6,506 bytes a round) is built and off until a funded key has
  tried it.
- **WordPress categories are records of their own** (`categories`, `category_terms`,
  `pages.category_id`, migrations 0039 to 0041; the first model, a flag on an entity, was undone
  by 0042 and 0043). The Category and Subcategory columns of a sheet become categories and file
  each page under its chain; Root Entity columns make entity groups, and a root's name is never a
  category. Publish files a page or a post under the whole chain, a product gains our product
  categories beside the client's, a revert takes back only what the run added, and a sync adopts
  the terms the site already has. The Pages rail filters by a branch of the category tree, and the
  pages, the entities, the runs and the import preview name the categories.
- **The companion plugin is 1.3.0**: it files pages under categories and lists them on category
  archives (`page_categories`). Without it posts and products are still filed and a page says it
  needs the plugin.
- **A workbook imports in one go**: one preview and one apply over the chosen sheets in the
  workbook's order, a tab per sheet, every finding naming its sheet, a scope clash caught in the
  preview, and a saved mapping or the sheet's own headers resolved before use, which is what makes
  the agent's import tools work without a mapping.
- **The refactor**: the composition root builds by area, the steps live by concern, the run engine
  reads by life cycle (`Engine.Wake` and `Recover` are gone), the import is a pipeline of named
  stages, the frontend shares its helpers, and the exports nothing called are deleted. The reports
  Pages tab no longer colours a failed item green.

**2026-10-03** edits the products the client creates by hand in WooCommerce; the reasoning is
under **2026-10-03** in `DECISIONS.md`.

- **A product run writes** the description through the plugin's raw route, then the short
  description, the attributes the product lacks and an image where it has none through
  WooCommerce's REST API, and the SEO meta; the name, price, stock, SKU, status and slug are never
  written, and since 2026-10-04 its categories only gain the product categories of its chain. A
  template declares the outputs in its `product` block.
- **A product is refused before anything is spent** when the store is not editable, the plugin is
  missing, the row has no product or the run is a draft; a revert hands back a product a human
  changed since and otherwise puts back exactly what the run replaced.
- **A sheet says its rows are products** (`rowType`), each row finds the client's product by its
  address, its slug or its name and keeps the sheet's URL as `planned_path` (migration 0032), and a
  product created after the import claims the row that waited for it on the next sync. A site
  records whether its store can be edited (`sites.commerce`, migration 0031).
- **`sync_back` warns when a published product's page does not show its description** to a
  visitor, the sign of a page builder, a cache or coming-soon mode.
- `internal/e2e/products_test.go` runs the whole loop against WooCommerce 11.1.2 on the docker
  stack.

**2026-10-02** makes the client's SEO workbooks import: the reasoning is under **2026-10-02** in
`DECISIONS.md`, and `samples/client-sheets.xlsx` carries the four sheet shapes the client uses.

- **Keywords are one list with volumes** on pages and entities (migration 0029), read from one
  cell as `bpc 157 (12000), buy bpc 157 (5,400), bpc-157`, sorted by volume, merged on a
  re-import, and given to the writer numbered with their volume; a template says how many must
  appear (`keywordRules.requiredKeywords`) and the missing ones are one `keywords_missing` warning.
- **An entity's name is unique under its parent** (`entities.scope_entity_id`, migration 0030),
  so a form's name repeats under every product; a shared name is labelled with its parent for the
  model and the anchors, and shown as a path on the screens.
- **The import reads the client's headers**: level columns (`Root Entity | Category |
  Subcategory`) make groups that take a page only on evidence (since 2026-10-04 only a root level
  does, and the others are WordPress categories), every row with a page gets an
  entity, a parent comes from the parent cell, the group or the URL tree, `Entity?` and other
  questions stay ignored, `own_entity: no` makes a technical page, the entity level reads as the
  kind, note columns travel with the page to the writer and back out, and the preview says what
  each column became. Matching is by parent and name, an ambiguous name is a blocking finding, and
  a repeated import changes nothing.
- **A group without a page is passed through by the link plan** (`no_page`, optional), and the
  page list filters by an entity and everything under it.

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

- **The import never plans `/`**, and a row's keywords land on the page (migration 0027; one
  list with volumes since 0029) even when the row names no entity.
- **Entities are proposed for chosen pages or pasted keywords and written only once reviewed.**
  `GraphService.PreviewFromPages`, `ProposeFromKeywords` and `ApplyProposals`, the three tools
  beside them, and a three-step dialog on the Graph screen: pick pages or paste keywords, preview,
  keep the proposals you want.
- **Placeholders work on the first try.** `{primaryKeyword} {entityName} {siteName} {pageTitle}`
  are expanded per page in `ResolveForPage`, anything else is refused by `Validate`, and the writer
  gets the brief's headings back through `content.Assemble`, so `section_missing` cannot happen.
- **The pipeline stops itself less.** A truncated or malformed answer is tried again with more room
  and a repair round (since 2026-10-03 with double the room once, and with no repair round, the
  schema being strict); the linker writes the phrases the body owes and falls back to a plain sentence
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

`relink`, `repair`, `sync` and `revert` own their recipes. The companion plugin was then
**1.2.0**; it is **1.3.0** since 2026-10-04.
`TestTheClientLoopFromTheSamples` drives the whole loop from the client's own workbooks against
docker, and `TestAChildWaitsForItsParentAndGoesOnOnceTheParentIsRegenerated` now stops the parent
by exhausting the writer, because an incomplete draft is tried again instead of failing at validate.

## The gate

**2026-10-03 and 04, partial: the closing gate over this work has not run yet.** What the units
ran: the docker suites, `task e2e:test`, `task e2e:full` and `task e2e:full:noplugin`, with the
contract and the loops again under `E2E_SEO=yoast`, green on 8088 with WooCommerce on, at
`2cd8f18` (unit K6); `task ui:lint`, `npm run typecheck` and the whole vitest suite, **1624 tests
in 151 files**, green at `31230cc` (unit K5); and in every unit `gofmt -l`, the comment check,
golangci-lint and `go test -race -count=1 -p 2` over the packages it touched. **Not yet run over
the head of this work:** the whole `go test -race -count=1 -p 2 -covermode=atomic
-coverprofile=coverage.out ./...`, `go run ./cmd/covergate`, a whole `golangci-lint run`, `task
lint:e2e`, `task build` (until it runs, the gitignored bindings are stale), and the docker suites
over the commits after `2cd8f18`. Migrations end at **0043**; **93 tools**, 63,304 bytes of schema
against the 63,400 the registry test allows; **125 bound methods**.

**2026-10-03, on Windows, whole, over the 2026-10-02 and 2026-10-03 work.** `go test -race
-count=1 -p 2 -covermode=atomic -coverprofile=coverage.out ./...` green; `go run ./cmd/covergate`
**domain+application 87.22% of 8819** (gate 80%), **total 87.28% of 21287** (gate 70%);
`golangci-lint run`, `task lint:e2e` and the comment check 0 issues, `gofmt -l .` silent; `task
build` green and leaves no generated diff; `npm run typecheck` clean and **1238 vitest tests in 121
files**; `task e2e:test` and `task e2e:full` green on 8088 with WooCommerce 11.1.2, the adapter
suite and the product loop green again under `E2E_SEO=yoast`, and `task e2e:full:noplugin` green.
`task ui:lint` failed once at 369 s on a test its output did not keep and passed on the two runs
after it. Migrations end at **0032**; **93 tools**, 84,923 bytes of schema against the 85,000 the
registry test allows.

**2026-10-02, on Windows, partial.** Run: `gofmt -l .` silent, the comment check,
`golangci-lint run` 0 issues, `go vet -tags uiharness` and `go vet -tags e2e` clean, `task ui:lint`
green (at `116a77c`), `npm run typecheck` clean and **1202 vitest tests in 119 files**; `go test
-race -count=1 -p 2 ./...` green through `0701b03` apart from the wall-clock flakes of
`internal/runtime`, each green alone, and every later commit green over the packages it touches,
because the full run for `9d84f48` was stopped by the machine running low on memory. Migrations
end at **0030**; **93 tools**, schema ceiling 80,700 bytes. **Not run:** `go run ./cmd/covergate`,
`task build`, `task lint:e2e`, `task e2e:full`; the import side of the e2e client loop was checked
without docker and holds (61 entities, 56 edges, the planted faults with their rows).

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
| — | 2026-10-03/04: OpenAI only on our own client, the spend in view, WordPress categories, the whole workbook, the refactor | done; the closing gate is pending |

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
  with the composition, and the Settings copy says so. So do `llm.timeout`,
  `llm.openai.baseUrl` and `llm.flexPatience`, read when the model client is composed.
- **What only a funded key can show.** The live probe ran on a key with no credit, which proves a
  request's shape and nothing it costs. Owed at the pre-release smoke: the probe's open tests
  (minimal effort and sampling, a stateless tool loop replayed across rounds, the replay forms of
  an assistant message, a stream's `[DONE]` and obfuscation, what breaks the cache prefix, flex
  served on terra and luna, the strict keyword matrix, whether output counts reasoning, field
  limits, image tokens per quality), the body of a flex capacity refusal, `anyOf` for a nullable
  object or array, and `cache_write_tokens` against a real bill. **The estimate prices an image at
  1,056 output tokens whatever its quality.**
- **Seven checks before `agent.toolLoading` becomes `deferred`**, with a funded key: a turn that
  triggers `tool_search` parses its `tool_search_call` and `tool_search_output` items and its calls
  carry the namespace; the next round, under `store: false`, accepts the search items replayed
  without `id` or `status` and the namespaced call; a second turn accepts history calls replayed
  without their namespace, and the setting switched both ways between turns works; `cached_tokens`
  shows from round two and the tools cost about 1,600 input tokens a round before a search; the
  model finds the
  right group from the descriptions (`templates_create`, `imports_preview`, `runs_start`); a turn
  stays within the loop limit of twelve; the loaded tool definitions show in the usage and the
  ledger.
- **A flex attempt abandoned for its patience returns no usage**, so partial work OpenAI may bill
  for it is not metered, and a capacity refusal is assumed unbilled.
- **A page is filed only by a sheet.** A page planned by hand or by the agent carries no category
  until a sheet's Category column names its chain; nothing files a page by hand.
- **One edge of the root rule stays open**: the site's half of the root set is read as the site was
  before the import, so a request that relabels a top-level hub as a topic and uses its name as a
  category level drops that level on its first import and makes the category on the second.
- **The UI harness's fake site loses its categories on a restart**, because `harnessrestore.go`
  does not put them back; the stored term ids stay until a sync drops them.
  `frontend/scripts/routes.json` has no route for the category filter yet.
- **`relink_page` reads the site record only after it wrote the page**: if that read fails, the
  site holds the relinked body while the map keeps the old hash, and the next sync reports the
  run's own write as drift.
- **For the owner to decide in the UI:** the rail's category chip reads `#12` or `new` with a
  legend, the full sentence in its tooltip, because the rail is 212 px wide; a trail in a narrow
  page table shrinks to its leaf; the graph inspector keeps its own drift badge.
- **What a revert cannot put back:** media a run uploaded, because a delete needs `force=true`
  and sweeping media a human may have reused is worse; and, against plugin 1.1.0, the SEO meta,
  because the read is refused from the manifest and `revert_meta_kept` names it instead.
- **`llm_calls` has no `message_id` or `round`**, so an agent turn's rows are found by its
  conversation and their time. `Retry-After` is read since 2026-10-03, and `ledger.List` has its
  caller in `ModelsService.ListCalls`.
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
  stands down, no draft can be previewed, a page goes up without its categories and a revert of
  an updated page pauses; the loop still generates, publishes and reads back. **Plugin 1.3.0
  changes the client's site**: its category archives, feeds and category sitemaps list pages, and
  the page editor gains a Categories box.
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

1. **The closing gate over the head of this work**, everything listed as not yet run under **The
   gate**, and the docker suites again over the commits after `2cd8f18`.
2. **The pre-release smoke with a funded OpenAI key**: one generate run with the writer on flex
   (the tier served and the reasoning shown in the ledger), one agent turn with tools (cached
   tokens from its second round), and the spend panel naming both by purpose and model; then the
   probe's open tests and the seven checks under **Known gaps**, and only then a decision on
   `agent.toolLoading`.
3. **A sandbox walk with the client's sheets and a real provider** (`task sandbox:up
   E2E_PLUGIN=1 E2E_WOO=1`): import `samples/client-sheets.xlsx` as one workbook and check each
   sheet's tab, the Groups and Categories segments and that no category is named Peptides; sync;
   run a page under TB-500 › Liquid and see it filed on the site and listed on the category
   archive; create a product by hand in WooCommerce under a category of the client's own, import
   the variation sheet in products mode, run it and see it keep the client's category and gain
   ours; revert both.
4. **The client's own sites**: plugin 1.3.0 lists pages on his category archives, feeds and
   sitemaps, which he should hear before he updates it; what a real model writes into the
   attributes, and whether a page builder hides the description, which
   `product_description_hidden` will say.
5. **A release** when the owner asks: the 2026-10-02, 2026-10-03 and 2026-10-03/04 work as one
   minor version.
6. The residue above: the denied tool row's decision, the four narrow-width UI items, the UI
   questions on the category trail, `ProposeFromPages` and `Import.Apply` as runs, and
   `settings.changed` for a declared value.
