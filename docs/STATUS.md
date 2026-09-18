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
- **Path normalisation is one algorithm with a temporary home.** Strip scheme and host
  after deciding internal-ness by exact lowercase host equality, strip query and
  fragment, never percent-decode, collapse duplicate slashes, force exactly one leading
  and one trailing slash, ASCII-lowercase the whole path, and make no exception for a
  file extension. `wp.NormalizePath` holds it for now; the canonical home is
  `internal/domain/pagemap.NormalizePath` from Phase 2, and a follow-up task replaces
  the adapter's body with a call into the domain once Phase 2 merges.

## Known gaps

- Application events published before `cmd/postulator` connects the relay to the Wails
  event manager are dropped, by design: nothing listens before the window exists.
- Backward paging exists in every repository (`paging.Request.Before`) but not at the
  use-case boundary, because `dto.ListRequest` carries one cursor; the client replays the
  cursor it used to reach the current page.
- Step `params` keys in template recipes are unspecified until Phase 6 names them; the
  seeds carry none.
- `internal/runtime` does not exist yet; its dependency rule still skips.
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
- `wp.NormalizePath` duplicates what `internal/domain/pagemap.NormalizePath` will own
  once Phase 2 merges. A follow-up task then replaces the adapter's body with a call into
  the domain — adapters may import domain — and moves the table test with it.
  `wp.InternalPath` stays in the adapter, because deciding whether a host is this site's
  host is adapter knowledge.
- Core REST filters `modified_after` on the site-local `post_modified` while returning
  the UTC `modified_gmt`, so the sync use case (Phase 7) must page through the plugin's
  `/content?since=` or convert the bound with the site timezone before it trusts an
  incremental core-REST window.
