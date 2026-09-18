# Status

The handoff point between sessions. Read this first.

**Branch:** `rewrite/v2`, cut from `dev`. **Target release:** `v2.0.0`.

## Where we are

**Phase 0 is complete and reviewed.** Tasks 0.1 through 0.10 are done, one commit per
task. The review ran on 2026-09-17: `task build`, `go vet ./...`, `golangci-lint run`
(v2.13.2, 0 issues), `go test -race -cover ./...` and `go run ./cmd/covergate` are all
green, the built binary opens its window, and the dependency rule was re-verified by
planting a domain package that imports squirrel directly and again through
`kernel/paging` — both were caught. Module coverage is 94.67%; every kernel package is
at or above 95.9%.

The fixes the review asked for landed on 2026-09-17 and are listed under Decisions
below.

**Phase 1A (infrastructure) is complete.** The plan is
`docs/superpowers/plans/2026-09-17-phase-1a-sqlite-secrets.md`; its sixteen tasks landed
one commit each. `internal/adapters/sqlite` holds the encrypted store, the embedded goose
migrations, the unit of work, the test helper and the settings and secrets repositories;
`internal/adapters/secrets` holds DPAPI, AES-GCM and the master key; `internal/app` opens
and closes the whole thing at startup. `go build`, `go vet`, `golangci-lint run` (0
issues), `go test -race -covermode=atomic ./...`, `go run ./cmd/covergate` and
`task build` are green, and the built binary starts, writes
`%APPDATA%\Postulator\master.key` and an adiantum-encrypted `postulator.db` whose first
bytes are not the SQLite magic header. Module coverage is 92.50%; every package this
phase added is at or above 87.2%.

**1A reviewed: approved, 2026-09-17.** The independent review re-ran the full gate and
probed the unit of work, the DSN and the recovery path; the four findings it raised landed
in `0eafaa3`, `922a831`, `d190c30` and `055d35e`. Module coverage is 92.55% of 1221
statements.

**Phase 1B (the Wails contracts spike) is complete** on the `phase-1b` worktree. The plan
is `docs/superpowers/plans/2026-09-17-phase-1b-wails-contracts.md`; its seven tasks landed
one commit each, plus three line-ending fixes. The spike answered the three open contract
questions and `docs/CONTRACTS.md` is final. `internal/application/events` owns the event
names, payloads and envelope; `internal/transport/wails` holds the error conversion, the
service wrapper, the health service and the event bridge; `internal/transport/wails/gen`
renders `frontend/src/generated/events.ts` and a test fails when it is stale; and
`frontend/src/lib` carries the error, event and paging helpers with `npm run typecheck`
wired into `task build`. Phase 2 follows.

**1B reviewed: approved** on 2026-09-17. `go build`, `go vet`, `golangci-lint run` (0 issues),
`go test -count=1 -race -covermode=atomic ./...`, `go run ./cmd/covergate` and `task build`
(with `npm run typecheck`) are green; the generator is byte-deterministic and its in-sync test
catches a stale module; no internal cause chain reaches the webview.

**Phase 2 (domain core) is complete.** The plan is
`docs/superpowers/plans/2026-09-18-phase-2-domain-core.md`; its twenty-seven tasks landed
one commit each. `internal/domain/{llm,site,graph,pagemap,template}` hold the pure model:
the entity graph with breadth-first parents, cycle detection and PageRank scores, the
path-keyed page map with its tree, index and cannibalization verdict, the template spec
with RFC 7396 resolution and five embedded starter templates. Migrations 0004–0010 add
`sites`, `link_policies`, `templates`, `entities`, `entity_anchors`, `edges`, `pages`,
`page_links` and `template_overrides`; `internal/adapters/sqlite` holds one keyset-paged
repository per aggregate; `internal/application/{sites,graph,pages,templates}` hold the use
cases, each publishing one event after its transaction commits; `internal/app` wires them,
seeds the templates at `Open` and relays events to the single `EventBridge` once
`cmd/postulator` connects it. Phase 3 (WordPress) and Phase 4 (LLM) follow.

**Phase 3A (the WordPress Go adapter) is complete** on the `phase-3a` worktree. The plan
is `docs/superpowers/plans/2026-09-18-phase-3a-wp-adapter.md`; its thirteen tasks landed
one commit each. `wp-plugin/openapi.yaml` freezes the `postulator/v1` contract that track
B implements in PHP; `internal/adapters/wp` holds the stdlib client for core REST,
WooCommerce and the plugin namespace, with proxy, retry, rate limiting and error mapping;
`internal/adapters/wp/wptest` is the in-memory fake that Phases 6, 7 and 12 test against.
The sync use case is not in this track and follows once Phase 2 lands.

**3A reviewed: approved** on 2026-09-18. The independent review re-ran the full gate
(`go build`, `go vet`, `golangci-lint` 0 issues, `go test -count=1 -race
-covermode=atomic ./...`, `covergate`, `gofmt -l .`), repeated the adapter suite under
`-race` for flakes, probed the application password through errors, details, log lines
and every `fmt` verb of `Config` and `Client`, and confirmed the contract test now fails
on a renamed field in `openapi.yaml`. Its four findings landed in `4f547fa` and
`f07b29b`. Coverage is 87.0% for `internal/adapters/wp` and 95.6% for `wptest`.

**Phase 3B (the WordPress companion plugin and the docker e2e harness) is complete** on
the `phase-3b` worktree. The plan is
`docs/superpowers/plans/2026-09-18-phase-3b-wp-plugin-e2e.md`; its tasks landed one commit
each. `wp-plugin/postulator-companion` serves `/wp-json/postulator/v1` (manifest, keyset
content listing, SEO meta writes, raw read and compare-and-swap write) in seven
dependency-free PHP files; `cmd/pluginzip` packages it byte-deterministically;
`docker/e2e` runs a pinned WordPress stack that provisions itself; and
`internal/adapters/wp/e2e` holds build-tagged Go tests that pass against it in all three
SEO modes. Track A's Go adapter is a separate worktree and this suite depends on nothing
from it.

**3B reviewed: approved** on 2026-09-18. The independent review re-ran the Go gate, both
lint passes, deterministic packaging and all three SEO modes; its findings landed in
`0d13d9f`, `06e85f3` and `19e928a`. The blocking one was a stored XSS: a payload in
`_postulator_seo_title` closed `<title>` and executed, because `pre_get_document_title`
short-circuits `wp_get_document_title()` before core's `esc_html`. `since` was inclusive
against a contract that freezes it exclusive, and a slug-less draft synthesised a path that
collided with a published page of the same title. All three are re-verified by e2e cases;
the suite is 31 tests, green in `none`, `yoast` and `rankmath`, with no PHP diagnostics
from the plugin under `WORDPRESS_DEBUG=1`.

**Phase 4 (the LLM layer) is complete.** `internal/application/llm` declares the port —
`Client`, `Request`, `Response`, `Delta`, `Structured[T]` — and imports no provider
library. `internal/adapters/llm` holds `gollemclient` over gollem v0.28.4, `catalog`,
`profiles`, `ledger`, `limiter`, `retry`, `recordreplay` and the scripted `fake`;
`internal/application/models` adds the seven use cases; `internal/app` composes the client
as retry → limiter → ledger → record/replay → gollemclient. Migrations 0011–0013 add
`model_catalog`, `model_profiles` and `llm_calls`, and the Phase 3A follow-up landed with
it: `wp.NormalizePath` and `wp.InternalPath` now call `internal/domain/pagemap`. The full
gate is green and every package this phase added is at or above 86%.

**Phases 5 and 6 (the run engine and the content factory core) are complete.**
`internal/domain/run` holds the vocabulary of section 7, the JSON checkpoint, the run,
item, step-execution, artifact and event records and the step registry that validates a
recipe before it is planned. Migration 0014 adds `runs`, `run_items`, `artifacts`,
`step_execs` and `run_events`. `internal/runtime` is the engine: `advance` claims an item
by compare-and-swap on its advance sequence, runs the step under its timeout, classifies
the outcome and persists the item, the checkpoint, the execution, the artifacts and the
appended events in one transaction; a sweep re-arms due and stalled items, reaps runs past
their deadline and purges published artifact bodies. `internal/domain/content` holds the
document, the link context, the link insertion with a decision per candidate, and the
compliance and structure reports; `internal/runtime/steps` holds `resolve_context`,
`generate_body`, `insert_links`, `repair_links` and `validate` with their prompts as
embedded Go templates. `internal/application/runs` exposes the ten use cases and
`internal/app` starts the engine after the recovery sweep.

**Phase 7 (the remaining steps, the WordPress sync and the site reports) is complete.**
`internal/runtime/steps` holds all twelve page steps plus `sync_site`; `internal/adapters/images` adds the
local folder, the WordPress media library and the OpenAI image adapters; `internal/adapters/wp/registry`
builds and caches one client per site and probes its plugin; `internal/adapters/wp/plugin` packs the
companion from an embedded tree and `cmd/pluginzip` is a thin caller; `internal/application/sync` queues a
site-scoped sync run, checks the plugin and hands back the zip; `internal/application/reports` answers the
site overview, the page report and the run report; `internal/app` composes all of it. The full gate is
green, module coverage is 90.0% of 10315 statements and every package this phase added is at or above
86.8%. Phase 8 follows.

**Phase 8 (the import and the export of the client page map) is complete.**
`internal/domain/importmap` holds the fourteen canonical fields, the saved mapping and the header alias
table behind `AutoDetect`; `internal/adapters/importer` reads the first sheet with excelize and a
separated-values file with a sniffed delimiter, and writes the export workbook; migration 0015 adds
`import_mappings`; `internal/application/imports` holds `Inspect`, `Preview`, `Apply`, `Export` and the
three mapping use cases, and `internal/app` composes it as `Core.Imports`. The preview normalises paths,
fills the gaps, merges repeats, deduplicates entity names without case, resolves edges by name, checks for
cycles and reports cannibalization without writing; `Apply` refuses a preview carrying an error. Module
coverage is 90.2% of 11201 statements. Phase 9 follows.

## What landed in Phase 0

- The v1.6.2 codebase is gone: `internal/`, `pkg/`, `frontend/`, `main.go`, `Makefile`,
  `scripts/`, `wails.json` and `go.sum`. Kept: git history, the release workflow, the
  build icons, the sitemap import samples and the module path.
- A Wails v3 skeleton that builds and runs. `task build` produces `bin/postulator.exe`,
  the window opens, and `HealthService.Ping` returns the ldflags-injected version.
- The full kernel: `errors`, `paging`, `id`, `clock`, `ctx`, `dto`, `log`, `settings`,
  `middleware`, all test-first, all at or above 95% statement coverage.
- `internal/app/deps_test.go` enforcing the dependency rule, with absent trees skipped.
- `.golangci.yml`, `lefthook.yml`, `.github/workflows/ci.yml` on windows-latest, and
  `cmd/covergate`. `release.yml` now builds with `task build` and publishes `v2.*` tags,
  marking a tag with a hyphen as a prerelease.
- The docs in this directory, plus the verbatim spec and roadmap under
  `docs/superpowers/`.

## Pinned versions

| Thing | Version |
|---|---|
| Go | `go 1.27` directive; `GOTOOLCHAIN=auto` resolves go1.27.0 |
| Wails v3 | `v3.0.0-beta.23` (CLI and module) |
| golangci-lint | `v2.13.2` |
| Task | `v3.53.1` |
| Node | 22 |
| squirrel | `v1.5.4` |
| zap | `v1.27.0` |
| lumberjack | `v2.2.1` |

## Decisions taken in Phase 0

- **UUIDs come from the standard library.** Go 1.27 ships a `uuid` package (RFC 9562).
  `github.com/google/uuid` is not a dependency. `kernel/id.Valid` does its own canonical
  v4 check because `uuid.Parse` also accepts braced, URN and unhyphenated text and any
  version.
- **`Cut` is a `Keyset` method**, not the free function the spec sketched. The value and
  id accessors already live on the keyset, and passing them again per call is the one
  way to make a cursor disagree with the query that produced it.
- **`kernel/ctx` exposes `ActorFrom`**, not `Actor`: `Actor` is the type name the domain
  uses for `Run.CreatedBy`, so the getter could not share it.
- **`kernel/dto` does not import `kernel/paging`.** It keeps its own limit constants and
  a test asserts the two agree. Importing paging would drag squirrel into the closure of
  everything that touches a DTO, including domain.
- **The Wails template is `vanilla`**, not `vanilla-ts`; beta.23 renamed it. It is still
  Vanilla + TypeScript + Vite.
- **No comments in `.golangci.yml`.** The repository rule forbidding comments in YAML
  won over the plan's request for an inline rationale per tuned rule. The rationales are
  in `CLAUDE.md`.
- **`errors.Wrap` returns `error`, not `*Error`.** Returning the concrete pointer meant
  `Wrap(nil, ...)` handed back a typed nil that is not nil once it is returned as an
  `error`, and `CodeOf` then dereferenced it. `Error`, `Unwrap`, `Is`, `CodeOf` and
  `Stack` are nil-receiver safe as well. Enrichment chains now start from `New(...)`,
  which cannot be nil.
- **The kernel exports nothing that only its own tests call.** `errors.Codes`,
  `errors.RetryAfter`, `paging.Order.String` and `log.RedactedKeys` are gone, and
  `SortKey.Literal` is unexported as `literal` per the spec, with its table test moved
  in-package.
- **`@wailsio/runtime` is pinned to `3.0.0-beta.23`**, matching the CLI tag, and the
  build task runs `npm ci` against the committed lockfile rather than `npm install`.
- **covergate cross-checks the filesystem.** If `internal/domain` or
  `internal/application` holds a non-test `.go` file but the profile reports no
  statements for it, the gate fails instead of reporting itself skipped. That is the
  case a narrowed `go test` package list would otherwise hide.
- **Vite writes unhashed asset names with `emptyOutDir` off**, so the committed
  `frontend/dist/.gitkeep` survives a build. Without it `go build ./...` fails on a
  fresh clone, because `frontend/assets.go` embeds a directory the frontend build has
  not created yet.

## Decisions taken in Phase 1A

- **Two pools over one file.** A writer `*sql.DB` capped at one connection with
  `_txlock=immediate`, and a reader `*sql.DB` opened `mode=ro` with four connections. One
  pool capped at a single connection would serialise reads behind long writes; an
  uncapped pool would let two goroutines both begin write transactions and turn WAL's
  single-writer rule into `SQLITE_BUSY` at commit time rather than at begin time.
- **`execFrom` reads, `writeFrom` writes.** Both return the ambient `*sql.Tx` when
  `Store.Do` is active, found through an unexported context key. Outside a transaction
  `execFrom` returns the reader and `writeFrom` returns the writer. The spec sketched one
  accessor; with two pools one accessor would make the reader dead code.
- **Nested `Do` reuses the outer transaction** and opens no savepoint. SQLite has one
  writer, and a nested savepoint would let an inner rollback be swallowed while the outer
  commits — the partial write the unit of work exists to prevent.
- **Migrations run through `goose.NewProvider`, not `goose.UpContext`.** `UpContext` reads
  the package-level filesystem and dialect that `SetBaseFS`/`SetDialect` mutate, which
  races when parallel tests open stores under `-race`.
- **The adiantum key travels as the `hexkey` URI parameter.** The VFS reads it when it
  opens the file, strictly before `journal_mode(WAL)` runs; the PRAGMA form would need SQL
  quoting inside a `_pragma=` value, and a hex string starting with a digit is not a safe
  bare pragma token.
- **The master key lives at `%APPDATA%\Postulator\master.key` via `os.UserConfigDir()`.**
  `xdg.ConfigHome` resolves to `%LOCALAPPDATA%` on Windows and would put it elsewhere;
  `xdg` also stays an indirect dependency this way.
- **No `//go:build windows` tags.** The application is Windows-only, `x/sys/windows`
  compiles nowhere else, and a tag would demand a second file that could only be a stub.
- **DPAPI is called with constant application entropy.** A fixed 32-byte literal in
  `adapters/secrets/dpapi` is passed to `CryptProtectData` and `CryptUnprotectData`, so
  any other process running as the same user cannot unprotect our master key by handing
  the blob straight back to DPAPI. The entropy is a compile-time literal, never derived
  at runtime, because changing it orphans every key already on disk. It is part of the
  on-disk format and is frozen from here on.
- **`sqlite.Open(Config{Path, Key, Recovery})` and `masterkey.Load(Config{Dir, Recovery})`
  replace the spec's `Open(path, key)`/`Load(dir)`** so recovery messages carry resolved
  paths.
- **The master key is written atomically.** `writeAtomically` writes `master.key.tmp`,
  `Sync`s, closes and then renames over `master.key`, and removes the temporary file if
  any step fails. A crash can therefore leave a stale `.tmp`, which the next write
  truncates and which `Load` never reads.
- **A key that cannot be unprotected reports `Locked`**, with the message
  `master key cannot be unprotected; remove %APPDATA%\Postulator\master.key to reset`.
  The file is never deleted automatically: the user decides whether to lose every stored
  secret.
- **A panic inside `Store.Do` rolls the transaction back and re-panics.** Leaving the
  `*sql.Tx` open stranded the only writer connection, so every later write blocked until
  its context expired. `Do` also reports `ctx.Err()` when a commit fails on a cancelled
  context, because `sql.ErrTxDone` was surfacing as `Internal` whenever database/sql's own
  rollback beat the commit.
- **Adiantum, WAL and the busy timeout work together.** The Phase 0 open question is
  answered: `TestOpenAppliesPragmas` reports `journal_mode=wal`, `foreign_keys=1`,
  `busy_timeout=5000` and `synchronous=1` on both the plain and the encrypted store, and
  the reader pool is proven to refuse writes.

## Decisions taken in Phase 1B

- **`ServiceOptions.MarshalError`, not `Options.MarshalError`.** `Bindings.Add` overwrites
  each bound method's marshaller with the service-level hook, so the application-level one
  never runs in beta.23. Services are registered with `application.NewServiceWithOptions`.
- **The transport error is `{code, message, details?, retry:{afterMs}?}`** delivered as
  the `cause` of the JavaScript rejection, not a JSON string in the message.
  `wails.Convert` strips the internal chain first, because `CallError.Message` is
  `err.Error()` and would otherwise leak the wrapped driver text.
- **`details` is dropped for `INTERNAL`.** `middleware.Recover` puts the panic text in
  `Details["panic"]`, and that must not reach the UI.
- **Generics survive the TypeScript generator.** `paging.List[T]` generates `List<T>`, so
  no concrete `XxxList` DTOs are needed. A custom `MarshalJSON` generates as `any`, which
  is why `paging.Slice[T]` and `dto.Time` lose their shape and `frontend/src/lib/paging.ts`
  restores it.
- **`application.RegisterEvent` is not used.** It would duplicate the registry in a second
  hand-written list and emit its typings into the gitignored `frontend/bindings`. Our own
  generator owns `frontend/src/generated/events.ts`, and a Go test keeps it in sync.
- **The generated TypeScript carries no header comment**, because the no-comments rule
  covers TypeScript including generated files. Its provenance is the `src/generated/` path
  and `docs/CONTRACTS.md`.
- **Envelope timestamps are `kernel/dto.Time`**, not `time.Time`: seconds precision,
  always UTC, matching every other DTO.
- **The health service moved to `internal/transport/wails`** and now has the contract shape
  `Ping(ctx, PingRequest) (BuildInfo, error)`, so the wrapper, the error hook and the
  frontend helper are exercised in production rather than only in tests.
- **`Services` is a method on `*app.Core`.** The composition root owns the Core and hands
  services what they need; `internal/transport/wails` cannot import `internal/app` because
  `internal/app` imports it, and the dependency rule allows only that direction.
- **Live events are best-effort.** v3 buffers nothing for a missing window, so
  `ListEvents(runId, sinceSeq, limit)` catch-up is mandatory.
- **`.gitattributes` pins `frontend/src/generated/events.ts` and `go.mod` to LF.** Both are
  rewritten by tools that emit LF; with `core.autocrlf=true` a fresh checkout would hand
  back CRLF and the generated-file-in-sync test failed exactly that way.

## Decisions taken in Phase 2

- **`Parents(id, depth)` is breadth-first, deduplicated, nearest level first**, name-then-id
  within a level; `Related` returns `[]Neighbor{Entity, Weight}` by weight descending.
- **`Score` is unweighted PageRank over approved edges only**, related edges counted both
  ways, damping 0.85, 30 iterations, scaled so the maximum is 1.0.
- **`Resolve(base TemplateSpec, siteOverride, pageOverride json.RawMessage)`** is RFC 7396
  over JSON; overrides are stored as merge-patch documents, arrays replace wholesale, and
  `TemplateSpec` plus the `llm` catalog types carry JSON tags. A deliberate deviation from
  spec §5.4, approved 2026-09-18; the spec copy is amended.
- **Anchors live in `entity_anchors`** and are loaded in one `IN` query per page, never N+1;
  `EntityRepo.Update` rewrites them.
- **Template overrides are keyed by `(template_id, scope, coalesce(site_id, page_id))`**
  through an expression unique index; the repository upserts by lookup.
- **One event per transaction, after commit**, through a `changed(siteID)` helper per
  service. `DeleteEntity`, `Unmap`, `SetCanonical` and page `Delete` publish both graph and
  pages because they touch both aggregates.
- **Cannibalization runs on `pages.Create`, on a path change in `Update`, and on
  `MapToEntity`**; never on `SetCanonical`. It lives in `domain/pagemap` and keeps the graph
  parameter because the keyword check needs the other entities.
- **The path is the hierarchy's source of truth**; `parent_page_id` is a maintained cache.
  `NormalizePath` is the single canonical normaliser (scheme and host stripped, no
  percent-decoding, duplicate slashes collapsed, one leading and one trailing slash,
  lowercased, no file-extension exception; whitespace, control characters and dot segments
  refused), `InternalPath(href, siteHost)` classifies link targets, `Create` adopts direct
  children, and a path change with descendants is refused.
- **Filter and sort types live in the domain packages** (`site.Query`, `graph.EntityQuery`,
  `graph.EdgeQuery`, `pagemap.Query`, `template.Query`, `template.PolicyQuery`).
- **Use-case lists embed `dto.ListRequest` and page forward only**; backward paging is a
  repository capability (`paging.Request.Before`). The kernel is untouched.
- **Request and response structs are camelCase JSON views owned by the use cases**;
  `docs/CONTRACTS.md` records that transport maps only where the wire shape must differ.
- **Repositories carry no clock**; use cases stamp second-truncated times.
- **`Site.Username` exists**; the WordPress application password lives only in the secret
  store under `site:<id>:wp_password`, written on `Create` when given and rotated or removed
  by `Update`.
- **`app.EventRelay` holds the single `EventBridge`**, connected from `cmd/postulator`
  after `application.New`; publishes before `Connect` are dropped.
- **Seeds pin no models and carry no step params.** `EnsureSeeded` also seeds one global
  `Default` link policy; `GetEffectivePolicy` falls back to it.
- **Cyclic foreign keys are real** (`entities.canonical_page_id → pages`,
  `sites.default_template_id → templates`, `sites.default_link_policy_id → link_policies`);
  the migration order is sites → link_policies → templates → entities → edges → pages →
  template_overrides, and `TestSchemaCascades` proves the cascade map with rows present.
- **Parent edges weigh 1**, `related` edges store the lower id first, `AddEdge` checks
  acyclicity as if approved and `ApproveEdge` checks again.
- **Repository writes map `UNIQUE` to `CONFLICT` with a table-specific message and zero
  affected rows to `NOT_FOUND`**; foreign-key failures stay `INVALID`.
- **`SetCanonical` maps an unmapped page** and refuses one mapped elsewhere.
- **Helpers live in the file of their first caller** (`escapeLike` in `entity_repo.go`, the
  nullable-column helpers in `page_repo.go`, the view helpers next to their use cases),
  because the `unused` linter refuses a helper committed ahead of its caller.

## Decisions taken in Phase 3A

- **WordPress page numbers, not our cursors.** Core REST is offset-based and exposes no
  keyset, so `ListItems` takes `ListQuery{Page, PerPage}` and returns
  `ItemPage{Total, TotalPages, Page, HasMore}`. Synthesising a cursor would claim a
  stability `LIMIT/OFFSET` does not have and would hide the `400
  rest_post_invalid_page_number` that is the real end-of-list signal. The plugin
  namespace is the one surface with a keyset, and its cursor is opaque end to end: the
  Go client never decodes, validates or builds one.
- **`contentHash` is `hex(sha256(raw post_content))` with no normalisation**, so Go's
  `wp.ContentHash` and PHP's `hash('sha256', $post->post_content)` agree byte for byte.
  It is deliberately not `domain/content.Document.Hash()`, which hashes normalised HTML
  for drift detection; this one is a compare-and-swap token and must be exact. `wptest`
  implements it a second time rather than importing it, and both are pinned against the
  same externally computed digest.
- **`wptest` shares no type, no JSON shape and no hash with `wp`.** A fake that imports
  the client's decoding cannot disagree with it, which is the only thing that makes it
  worth having. Its own tests drive it with plain `net/http`.
- **Reads send `context=edit`.** Without it WordPress returns the theme's rendering
  instead of the stored post, and relink would write rendered HTML back into
  `post_content`.
- **`CreateItem` re-reads the created id.** WordPress rewrites a colliding slug to
  `-2` and computes `link` from the permalink structure, so the create response is the
  only truth about both, and one extra GET buys a slug and a link that are true.
- **Tri-state fields are a `map[string]any`.** `omitempty` erases a legitimate
  `parent: 0`, so `UpdateItem` builds the body key by key: `nil` keeps, `&0` moves to the
  top level, `&id` reparents; `nil` categories keep and `[]int64{}` clears.
- **The generic write methods are core-only.** `CreateItem`, `UpdateItem` and
  `DeleteItem` report `Invalid` for `product` and `product_cat`; `UpdateProduct` is the
  one write path into a shop, and it names WooCommerce's own fields.
- **The no-redirect client is not an option.** `Probe` gets its own `*http.Client` over
  the shared transport; letting a caller switch redirect-following off for ordinary calls
  would break every site that 301s a REST path to its trailing-slash form. `Probe` never
  follows a redirect: an http-to-https upgrade is `ProbeUpgradeRequired` plus a warning,
  and a redirect to `wp-login.php` or `/wp-admin` is `Unauthorized`.
- **Retries wrap the rate limiter, not the other way round.** A retry is a new request
  against the same site and pays the same budget; the backoff sleep happens before the
  limiter grants, so a token is never held idle. `Retry-After` wins over the backoff and
  is parsed as both delta-seconds and HTTP-date. The backoff is deterministic and has no
  jitter: one desktop process against one site is not a herd.
- **A cancelled caller context is `Cancelled`, our own timeout is `External`.** Retrying
  a context the caller cancelled can only fail the same way.
- **`AllowInsecure` unlocks the `http` scheme and nothing else.** `TLSClientConfig` is
  never set in this package, and there is no option that could set it.
- **An empty `wp.proxyUrl` means no proxy**, not `http.ProxyFromEnvironment`: a stale
  `HTTPS_PROXY` must not silently route WordPress credentials through a host the user
  never configured here.
- **The manifest is cached per client, and only a 404 is cached as absent.** A 5xx or a
  transport failure is re-probed, so a site that was briefly down is not written off for
  the life of the client. Plugin methods then fail with `Invalid` and
  `Details["code"] == "plugin_missing"` without a request, which is how a caller degrades
  to core REST.
- **A log line carries method, path, status, durationMs and the error codes, never a
  body, a query string or a header.** `kernel/log` redacts by field key, which protects
  nothing if a body is logged as one blob.
- **Path normalisation is one algorithm and the domain owns it.** Decide internal-ness by
  exact lowercase host equality, strip scheme, host, query and fragment, never
  percent-decode, collapse duplicate slashes, force exactly one leading and one trailing
  slash, lowercase the whole path, and make no exception for a file extension.
  `wp.NormalizePath` and `wp.InternalPath` are now one-line calls into
  `internal/domain/pagemap`.

## Decisions taken in Phase 3B

- **The plugin never removes a kses filter.** `kses_init()` re-runs on `set_current_user`,
  and an administrator holds `unfiltered_html`, so core removes the filters itself before
  a REST callback runs. The raw write returns the hash of what is **actually stored**, not
  of what was sent, so any alteration is visible to the caller instead of silent.
- **Every write passes `wp_slash`.** `wp_insert_post` and `update_metadata` both unslash,
  so without it a single backslash or escaped quote is eaten on every save. A dedicated
  e2e case writes a Windows path and an escaped quote and asserts byte equality.
- **`normalize_path` is the only place a path is produced.** Collapse duplicate slashes,
  force one leading and one trailing slash with no file-extension exception, then
  `strtolower`. Host comparison is exact and lowercase with no `www` stripping, query and
  fragment are dropped, and nothing is percent-decoded.
- **The listing has two phases.** Posts keyset on `(post_modified_gmt, ID)`, then
  `product_cat` terms on `term_id`; the cursor names its phase. `nextCursor` is `null` at
  the end of the list and an undecodable cursor is a `400`, never a silent restart.
- **A term's `modified` is plugin-maintained.** WordPress has no such field, so
  `_postulator_modified` (RFC3339 UTC) is written on `created_term`/`edited_term` and
  backfilled once on first read.
- **A draft reports the permalink it will have**, computed from a clone with
  `post_status = publish`, rather than the `?page_id=` URL `get_permalink` returns for a
  draft. A tool whose whole model is paths cannot report `/` for every unpublished page.
- **`since` is strict RFC3339**, validated by regex before `strtotime`, because
  `strtotime` cheerfully accepts `yesterday`.
- **Write routes are post-only.** Post ids and term ids collide, and the contract has no
  discriminator, so `/seo-meta/{id}` and `/content/{id}/raw` resolve through `get_post()`
  and a term id is a `404`. Term SEO is still read in `/content`, from Yoast's
  `wpseo_taxonomy_meta` option or Rank Math's term meta.
- **The 409 is a `WP_REST_Response`, not a `WP_Error`.** `WP_Error` can only carry extra
  fields under `data`, and `currentHash` must be top-level next to `code` and `message`.
- **With no SEO plugin the head is replaced, not appended.** `pre_get_document_title` and
  `get_canonical_url` make a duplicate `<title>` or canonical impossible; the e2e suite
  asserts a count of exactly one of each.
- **`cmd/pluginzip` is Go, not `zip` or `ZipArchive`.** Neither exists reliably on Windows
  and neither is deterministic without argument archaeology. Sorted names, the ZIP epoch
  and mode 0644 make two runs byte-identical.
- **WordPress is pinned to 7.0.1, not the 6.9.2 the plan named.** WooCommerce 11.1
  requires WordPress 7.0, so 6.9.2 could not install it and `product`/`product_cat` would
  have gone untested.
- **Application passwords need `WP_ENVIRONMENT_TYPE=local` over plain HTTP**, and the
  WP-CLI container needs the `WORDPRESS_DB_*` variables because the official image's
  `wp-config.php` reads them at runtime rather than baking them in.
- **Rank Math is supported.** It keeps its entire front end silent until the setup wizard
  is past: the meta keys wrote and read back correctly while the rendered page carried
  core's title and no description. `rank_math_is_configured` and
  `rank_math_registration_skip` are the two gates, and the bootstrap sets both.

## Decisions taken in Phase 4

- **The port owns the schema, the adapter owns the provider.** `Structured[T]` derives the
  schema by reflection from `json`, `description` and `enum:"a,b"` tags, refuses recursive
  types, maps and channels before the first call, appends one JSON instruction to the
  system prompt and repairs exactly one decode failure by feeding the decoder error back.
  `gollemclient` turns it into a `gollem.Parameter` with `ContentTypeJSON` and
  `WithSessionResponseSchema`, so the constraint is provider-native.
- **Usage is `gollem.Response.InputToken`/`OutputToken`**, on the completion and on the
  last streamed chunk. gollem exposes no finish reason, so `stop` and `length` come from
  the token ceiling and `content_filter` from `gollem.ErrProhibitedContent`. `Delta`
  carries an `Err` like `gollem.Response.Error`; without it a failure after the first
  chunk would close the channel silently.
- **Three gollem v0.28.4 limits are worked around or accepted.** Its OpenAI session drops
  the system prompt, so the adapter sends it as a `RoleSystem` history message for that
  provider only. Its Claude `Session.Stream` calls the non-streaming Messages API and
  emits one chunk. Its Gemini client is Vertex-only — no API key, no endpoint override,
  and `WithGoogleCloudOptions` is ignored — so the settings are `llm.gemini.projectId` and
  `llm.gemini.location` rather than the `llm.gemini.baseUrl` the brief named, and Gemini
  has no httptest coverage. `go-openai` also rejects a temperature other than 1 for any
  `gpt-5*` or o-series model before the request leaves the process.
- **The embedded catalog carries only models whose price was read off a vendor page** on
  2026-09-18: OpenAI `gpt-6-astra` and `gpt-5.6-{sol,terra,luna}`; Anthropic
  `claude-fable-5-1`, `claude-opus-5`, `claude-sonnet-5`, `claude-haiku-4-5`; Google
  `gemini-3.5-flash{,-lite}` and `gemini-2.5-flash{,-lite}`. Tiered and promotional prices
  are left out, because `ModelInfo` holds one flat rate per direction; rate limits are not
  published per model, so every entry carries one conservative `rpm`/`tpm` and
  `model_catalog` is the correction. An override row is the whole entry, so disabling is
  `enabled = 0`, re-enabling is another upsert, and neither repository has a delete.
- **The ledger stamps `created_at` from `kernel/clock` and measures latency with
  `time.Since`**; the clock abstraction is for timestamps, not elapsed time. The
  record/replay key excludes `Request.Meta`, so a fixture survives a new run id.

## Decisions taken in Phases 5 and 6

- **One `Status` for runs and for items**, the seven values of section 7. Section 5.6
  sketched two vocabularies; a single one keeps `Advanceable`, `Active` and `Terminal`
  meaningful on both rows and is what the claim query filters on.
- **`Classify` maps `NeedsHuman` to `exhausted`**, not to a class of its own. Exhausted is
  the class whose default action is `pause`, and a run that needs a human is in exactly
  that position: it cannot proceed without something outside the process.
- **A recovered panic is a transient fault.** A step that panics is retried under its own
  retry ceiling rather than killing the worker or failing the item outright; the engine
  logs the fault without the panic text reaching the event payload.
- **The claim, the step and the settle are three phases, two transactions.** The step runs
  between them with no transaction open, because an LLM call must not hold the single
  SQLite writer. The settle lands only against the advance sequence the claim took, so a
  pause, a cancel or a reclaim while the step ran discards the transition — while still
  recording the execution and its artifacts, which is what makes the input-hash reuse work.
- **`advance_seq` is the only optimistic lock.** `Claim`, `Requeue` and `StopAll` each bump
  it; `Persist` matches it. A lease is a hint for the sweep, never the lock itself.
- **Artifacts are immutable and keyed by `(item, step, kind)`**, so a retry replaces its own
  step's rows and a later step reads the newest earlier producer of a kind. A step
  execution's `attempt` counts the rows already recorded, so a manual `RetryStep` cannot
  collide with the unique index, and a step with no `Retry.Max` gets three attempts.
- **`WakeAt` resolution is the sweep interval.** There is no per-item timer: a waiting item
  is re-armed by the next sweep, which is also the path a crash recovers through.
- **The template spec governs link shaping, the site policy governs the site-wide rules.**
  `effectivePolicy` takes `LinkRules` from the resolved spec and `ForbidExternal`,
  `ForbidSelf` and `AnchorStrategy` from the site's effective policy.
- **`InsertLinks` counts an existing link to a target as placed** and re-runs to the same
  document, which is what makes `repair_links` and a re-run of the whole item safe; a
  property test runs two hundred random bodies through it twice. `Structure` never sees the
  meta title, because it reads a body fragment, so `validate` adds the
  `primary_missing_in_title` finding from the draft artifact instead.
- **A report scores `1 - 0.25 per error - 0.05 per warning`, floored at zero**, and the
  validation artifact carries the lower of the two scores. Prompts are
  `{{define "<step>.system"}}` and `{{define "<step>.user"}}` in one embedded template per
  step.

## Decisions taken in Phase 7

- **A sync run has no target page.** `run.Kind.PageScoped()` is false for `sync` and `import`, and the
  engine then claims an item without resolving a page or a template spec; the item's `page_id` names the
  site. A synthetic page row per site would have leaked into every page listing instead.
- **A step that returns `wait` with a `WakeAt` that is not in the future is re-dispatched at once**, not at
  the next sweep. That is what makes `sync_site` a batch loop whose cursor is persisted after every batch,
  so a crash resumes from the last one rather than from the beginning.
- **`report` does not write `Run.Stats`**, against the letter of the phase brief. The engine recomputes the
  stats on every settle from the item counts and the ledger, so a second writer could only disagree with
  it; the aggregate lives in the `final_report` artifact.
- **Image failures are recorded, not raised.** `generate_images` puts `no_image_source`, `images_failed` or
  `image_upload_failed` into the `images` artifact and carries on, because a missing stock folder must not
  fail a page. `judge` swallows its own model failure the same way, except on a cancelled context.
- **`publish` looks up by `Page.WPID` first and by slug plus parent second**, passing the five editable
  statuses, because core REST lists only published pages by default. A create whose response was lost is
  therefore reconciled by the next attempt rather than duplicated.
- **`relink_neighbors` writes a neighbour only when a link was actually inserted**, never on
  `already_linked`, and a `409` becomes a `relink_conflict` warning finding of class `needs_human` on an
  item that still completes. Without the plugin every neighbour is `skipped`.
- **The sync reconciles in one pass and repairs in a second.** Parent ids and link targets that arrive in a
  later batch are resolved once the pull is done, because a batch only knows the pages it has already seen.
- **The core-REST pull covers `page` and `post` only**, and therefore never archives a `product` or a
  `product_cat` row; the plugin's `/content` listing covers all four. Neither path passes `since`: a full
  pull is what makes "absent from the site" mean archived, which is the answer to the `modified_after` gap.
- **`wp-plugin` is a Go package.** `//go:embed` cannot reach a parent directory, so the companion tree is
  embedded at its own root and `internal/adapters/wp/plugin` packs it deterministically from an `fs.FS`.
- **The client registry caches by `site.UpdatedAt`.** One rate limiter and one manifest cache per site are
  worth keeping; a changed base URL or a rotated password bumps `UpdatedAt` and the next call rebuilds.

## Known gaps

- Application events published before `cmd/postulator` connects the relay to the Wails
  event manager are dropped, by design: nothing listens before the window exists.
- Backward paging exists in every repository (`paging.Request.Before`) but not at the
  use-case boundary, because `dto.ListRequest` carries one cursor; the client replays the
  cursor it used to reach the current page.
- Step `params` keys are `allowErrors` on `validate` and `iterations` on `repair_links`;
  the shipped seeds still carry none, so both take their defaults.
- `lefthook` is not installed on the development machine. Install it with
  `go install github.com/evilmartians/lefthook@latest && lefthook install`.
- **`golangci-lint` on this machine must be run from `$(go env GOPATH)/bin`.** A scoop
  shim earlier on `PATH` is v2.11.4 built with go1.26 and refuses a `go 1.27` module
  outright. `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2`
  puts the correct binary in `GOPATH/bin`; the shim still wins on `PATH`.
- The frontend is a stub that prints the build info. It is replaced in Phase 11.
- Phase 3A ships no sync use case: `internal/application` gains its WordPress consumer
  only after Phase 2 lands the page map and the repositories. Nothing in `internal/app`
  constructs a `wp.Client` yet, so `wp.timeout`, `wp.retries`, `wp.rateLimitPerSecond`
  and `wp.proxyUrl` are declared and validated but not yet read at startup.
- Core REST filters `modified_after` on the site-local `post_modified` while returning
  the UTC `modified_gmt`, so the sync use case (Phase 7) must page through the plugin's
  `/content?since=` or convert the bound with the site timezone before it trusts an
  incremental core-REST window.

- The e2e suite carries its own small HTTP client rather than using `internal/adapters/wp`
  from track A, which had not landed when it was written. Switching it to the adapter is a
  follow-up that deletes `client` from `harness_test.go`.
- `models.SetProviderKey` writes `llm:<provider>:api_key`; the Phase 11 settings surface
  still has to call it. `ledger.List` likewise has no caller until the Phase 11 read models,
  and the scripted `gollem.LLMClient` waits for the Phase 9 agent runner rather than ship
  dead.
- A run's deadline is the `runtime.DefaultRunDeadline` constant, not a setting: nothing in
  the UI sets one yet, and a second knob with no reader would be dead configuration.
- The artifact purge keys on a `publish_result` artifact, which only the Phase 7 publish
  step produces; until then the sweep finds nothing to purge and a synthetic artifact is
  what proves the query.
- The docker e2e stack is never run in CI: `windows-latest` cannot run Linux containers,
  and the Ubuntu job exists only to lint and package the plugin.

## Decisions taken in Phase 8

- **The mapping, the fields and the auto-detection live in `internal/domain/importmap`**, not in
  `adapters/importer` as section 9.6 sketched: `application/imports` must name those types and the
  dependency rule forbids it from importing an adapter. The adapter is the file reader and writer only.
- **The migration is `0015`, not `0016`** — the tree ended at `0014_runs.sql`.
- **A header folds every non-alphanumeric run to one space**, unlike the Archond mapper that drops them,
  and a miss retries with the spaces removed. Otherwise the export's own `primary_keyword` header would not
  survive a round trip through its own alias table.
- **Cannibalization is a warning, never an error**, as are an unrecognised entity kind, WordPress type or
  page kind, which fall back. Only an unknown parent or related entity, a self-edge, a cycle and an
  unreadable path are errors, and `Apply` refuses while any of them stands.
- **`page_kind` resolves to a template**, preferring a site-scoped one over the global one of that kind: it
  is the only field with nowhere else to land, and read-but-unused would be a hole.
- **An existing entity keeps its spelling and its primary keyword.** The import unions the keywords and the
  anchors and overrides the kind only when the row names one; it never renames what the operator curated.
- **`SaveMappingAs` reuses the mapping already saved under that name**, because the unique index is on
  `(site_id, name)` and a second import would otherwise roll back on a constraint nobody sees.
- **`examples/sitemap-import-example.json` and `examples/sitemap.json` are gone.** JSON import is out of
  scope; the csv and xlsx samples stay and a test proves `AutoDetect` still opens both.
