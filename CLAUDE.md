# CLAUDE.md — Postulator v2 (session contract)

Read [`docs/STATUS.md`](docs/STATUS.md) first; it says what landed and what is next. Then [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md), [`docs/CONVENTIONS.md`](docs/CONVENTIONS.md), [`docs/CONTRACTS.md`](docs/CONTRACTS.md), [`docs/ORCHESTRATION.md`](docs/ORCHESTRATION.md), [`docs/VISION.md`](docs/VISION.md). The binding design is [`docs/superpowers/specs/2026-09-17-postulator-v2-design.md`](docs/superpowers/specs/2026-09-17-postulator-v2-design.md); the phase list is [`docs/superpowers/plans/2026-09-17-postulator-v2-roadmap.md`](docs/superpowers/plans/2026-09-17-postulator-v2-roadmap.md).

Windows-only Wails v3 desktop app. Go 1.27, CGO off, local SQLite, no server.

## Hard rules (do not violate)

1. **No comments in Go, TypeScript, YAML or PHP** — not inline, not godoc, not package docs. Naming carries meaning; prose lives in `docs/` and here.
2. **No stubs, no TODOs, no placeholders.** If something cannot be done, stop and say so. A hole that looks finished is worse than an unfinished task.
3. **Interfaces are declared by the consumer**, never by the implementation and never in advance.
4. **TDD.** Failing test first, run it, implement, run it, commit. Table tests. Gates: `domain`+`application` ≥ 80%, module ≥ 70%, kernel ≥ 90%.
5. **Cursor pagination only.** No offset, anywhere.
6. **JSON is camelCase**, timestamps are RFC3339 UTC, ids are UUID v4 text.
7. **Errors crossing a boundary are `*kernel/errors.Error`** with a frozen code. Adapters convert driver and transport errors at their own edge.
8. **Secrets never reach a log, an artifact or a plain column.**
9. **No new `.md` files** beyond `CLAUDE.md` and the set in section 10 of the spec.
10. **Bad code is rewritten, not patched.** Commit your own work on `rewrite/v2` with conventional commits; never amend, rebase, force-push or `--no-verify`.

## Layout and workflow

`cmd/postulator` (bootstrap) · `internal/kernel` (shared) · `internal/domain` (pure) · `internal/application` (use cases) · `internal/adapters` · `internal/runtime` (run engine) · `internal/transport` (wails, agent) · `internal/app` (composition root) · `frontend` (Vite stub) · `wp-plugin` (PHP).

Dependency rule, enforced by `internal/app/deps_test.go`: domain sees only kernel and stdlib, application sees domain and kernel, only `app` imports `transport`, only `kernel/paging` imports squirrel.

```
task build                 bin/postulator.exe, builds the frontend and bindings first
go test -race ./...
golangci-lint run
go run ./cmd/covergate     needs coverage.out from -coverprofile
```

Before an agent implements a module, it reads the Archond files that section 15 of the spec names for it, and ports the shapes — never Asynq, Postgres, jet, pgx, fx, Fiber or credits.

## Footguns

- **`frontend/dist/.gitkeep` is committed on purpose.** `frontend/assets.go` embeds `all:dist`, so `go build ./...` fails on a fresh clone if that directory is absent. Vite therefore runs with `emptyOutDir: false` and unhashed asset names — turn either back on and the placeholder is deleted on every build, which shows up as a dirty tree and a broken clone rather than as a frontend problem.
- **`log.Logger.Close()` does not call `Sync()`.** Syncing stdout fails on Windows whenever the output is a pipe, which is every CI run. Lumberjack writes unbuffered, so closing the sinks is the whole job; adding a buffered sink means revisiting this.
- **`middleware.Timeout` runs the next handler on its own goroutine** and re-panics on the caller's. Without that, a panic underneath it kills the process no matter where `Recover` sits in the chain.
- **A cursor records the sort it was issued for.** Replaying one against a different `ORDER BY` is rejected as `Invalid` rather than silently skipping or repeating rows. Do not "fix" that by dropping the check.
- **`kernel/dto` must not import `kernel/paging`.** It would drag squirrel into the closure of everything that touches a DTO, including domain, and the dependency test would fail somewhere unrelated. The duplicated limit constants are held in step by a test.
- **golangci-lint must be built by the Go release the module targets.** A binary built with go1.26 refuses a `go 1.27` module outright, so install it with `go install`, not from an archive.
- **Live events are dropped, not buffered.** v3 dispatches a custom event to the windows that exist at that instant; with no window, or across a page reload, it is gone. Any progress UI must replay `RunsService.ListEvents(runId, sinceSeq, limit)` on connect.
- **`frontend/src/generated/events.ts` is generated and committed.** `task events` rewrites it and a Go test fails when it is stale. `frontend/bindings/` is the opposite: generated and gitignored, rewritten by every build. That file and `go.mod` are pinned to LF in `.gitattributes`, because `core.autocrlf` otherwise hands a fresh checkout CRLF that no generator ever writes and the in-sync test fails on a clean clone.
- **The tuned lint rules and why**: `errcheck` runs with `check-blank` and `check-type-assertions` and no baseline, so `_ = f()` is a finding rather than an escape hatch. `govet` is `enable-all` with `shadow` non-strict, because strict mode reports every `if err := f(); err != nil`. `fieldalignment` is off: it orders struct fields by machine layout, and our field order is the JSON order we owe the frontend. `gocritic` runs diagnostic, style and performance with `hugeParam` off. `misspell` is US English and ignores `cancelled`, which is a domain status value.
- **The WP-CLI image's `www-data` is uid 82, the WordPress image's is uid 33.** The compose `bootstrap` service therefore runs as `user: "33:33"`; without it WP-CLI cannot write `.htaccess` or install a plugin into the shared volume, and the failure reads as a WordPress permissions error rather than a container mismatch. The same container also needs the `WORDPRESS_DB_*` variables, because the official image's `wp-config.php` calls `getenv` at runtime instead of baking the values in, and application passwords need `WP_ENVIRONMENT_TYPE=local` to work over plain HTTP at all.
- **`go test -tags e2e` is the only thing that compiles `internal/adapters/wp/e2e`.** `go build`, `go vet`, `golangci-lint run` and `cmd/covergate` all skip it silently, so `task lint:e2e` is not optional and `gofmt -l .` is what catches formatting there.

- **WordPress `modified_gmt` has no timezone suffix**, so `time.Parse(time.RFC3339, ...)`
  fails on it; `wp.parseWPTime` falls back to `2006-01-02T15:04:05` read as UTC. The
  reverse is worse: core REST filters `modified_after` on the site-local `post_modified`
  while returning `modified_gmt`, so an incremental core-REST sync of a non-UTC site can
  miss or repeat items inside the offset. The plugin's `/content?since=` compares
  `post_modified_gmt` and is the accurate path; that is one of the reasons the plugin
  exists.
- **WordPress sends `"meta": []`, not `{}`,** for a post type with no registered meta, so
  decoding it straight into a map fails. `wp.metaBag` accepts the array, the object and
  null.
- **`wptest` must never import `wp`.** The fake is a second, independent implementation of
  the WordPress contract; sharing the client's types or its hash would let one bug satisfy
  both sides of every assertion.

## Standing rulings

- **2026-09-17** — UUIDs come from the Go 1.27 standard library `uuid` package; `github.com/google/uuid` is not a dependency. `kernel/id.Valid` does its own canonical v4 check, because `uuid.Parse` also accepts braced, URN and unhyphenated text and any version.
- **2026-09-17** — `paging.Cut` is a `Keyset` method, not the free function the spec sketched: the accessors already live on the keyset, and passing them per call is the one way to make a cursor disagree with its query.
- **2026-09-17** — `.golangci.yml` carries no inline rationale, against the letter of task 0.8, because the no-comments-in-YAML rule outranks it. The rationale is the Footguns entry above.
- **2026-09-17** — The Wails template is `vanilla` (Vanilla + TypeScript + Vite); beta.23 renamed `vanilla-ts`.
- **2026-09-17 (review)** — `errors.Wrap` returns `error`, never `*Error`: the concrete pointer made `Wrap(nil, …)` a typed nil that is not nil once returned, and `CodeOf` dereferenced it. Chain enrichment off `New(...)` instead. The kernel exports nothing whose only caller is its own test.
- **2026-09-17 (phase 1B)** — The error marshaller must be set per service with `application.NewServiceWithOptions`; `application.Options.MarshalError` is dead code in beta.23 because `Bindings.Add` overwrites each method's marshaller with the service option.
- **2026-09-17 (phase 1B)** — A service method returns `wails.Convert(err)`, never the raw error: `CallError.Message` is `err.Error()`, so a wrapped driver message would cross into the webview.
- **2026-09-17 (phase 1B)** — `application.RegisterEvent` is not used. The Go registry plus `go run ./internal/transport/wails/gen` owns the event typings; a second list would drift and its output lands in the gitignored `frontend/bindings`.
- **2026-09-17 (phase 1B)** — `Services` is a method on `*app.Core`. `internal/transport/wails` cannot import `internal/app`, because `internal/app` imports it; the composition root is what hands a service its dependencies.

- **2026-09-18 (phase 3A)** — `adapters/wp` is the one place in the codebase where page
  numbers are legal. WordPress core REST has no keyset; `kernel/paging` is untouched by
  it, and the plugin's own cursor is passed through opaque.
- **2026-09-18 (phase 3A)** — `wp.ContentHash` is `hex(sha256(raw))` and is frozen. It is
  not `domain/content.Document.Hash()`, and normalising it would silently break the
  optimistic-concurrency check on `PUT /content/{id}/raw`.
- **2026-09-18 (phase 3A)** — `wp-plugin/openapi.yaml` is the contract between the two
  Phase 3 tracks. Neither track changes it alone.

- **2026-09-18 (phase 3B)** — The companion plugin never removes a kses filter. It relies on the authenticated administrator's `unfiltered_html` and returns the hash of what is actually stored, so a filtered write is visible rather than silent. Multisite is unsupported for that reason.
- **2026-09-18 (phase 3B)** — Every plugin write passes `wp_slash`, because `wp_insert_post` and `update_metadata` unslash what they are handed.
- **2026-09-18 (phase 3B)** — `normalize_path` has no file-extension exception: collapse duplicate slashes, one leading and one trailing slash, then `strtolower`. Host comparison is exact and lowercase with no `www` stripping. One rule with no branches is how the PHP and Go sides stay in step.
- **2026-09-18 (phase 3B)** — The `/content` cursor names its phase (`post`, then `term`), `nextCursor` is `null` at the end of the list, and an undecodable cursor is a `400` rather than a restart from the beginning.
- **2026-09-18 (phase 3B)** — `/seo-meta/{id}` and `/content/{id}/raw` address posts only; post and term ids collide and the contract has no discriminator, so a term id is a `404`.
- **2026-09-18 (phase 3B)** — `since` is validated as RFC3339 by regex before `strtotime`, which otherwise accepts `yesterday` and every other English phrase.

- **2026-09-18 (phases 9-10)** — `github.com/robfig/cron/v3` joins `golang.org/x/net/html` as a third-party package the domain may import; `internal/app/deps_test.go` holds the allowance and a hand-written cron parser is not worth the bugs.
- **2026-09-18 (phases 9-10)** — `internal/transport/agent` is the only package that may name gollem, which `deps_test.go` enforces. `RunSpec` and `RunResult` live in `internal/application/agent` because the consumer declares the interface it calls.
- **2026-09-18 (phases 9-10)** — A tool's arguments are masked with `log.IsSensitiveKey` before they reach the tool call ledger, an event or a confirmation summary. Only `pending_actions.args` keeps them whole, because that row is the command the approval replays.

## Product guardrail

Postulator turns an entity graph into graph-compliant WordPress pages; reject features that do not serve that loop.
