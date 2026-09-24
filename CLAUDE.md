# CLAUDE.md — Postulator v2 (session contract)

Read [`docs/STATUS.md`](docs/STATUS.md) first; it says what landed and what is next, and [`docs/DECISIONS.md`](docs/DECISIONS.md) says why. Then [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md), [`docs/CONVENTIONS.md`](docs/CONVENTIONS.md), [`docs/CONTRACTS.md`](docs/CONTRACTS.md), [`docs/ORCHESTRATION.md`](docs/ORCHESTRATION.md), [`docs/VISION.md`](docs/VISION.md). The binding design is [`docs/superpowers/specs/2026-09-17-postulator-v2-design.md`](docs/superpowers/specs/2026-09-17-postulator-v2-design.md); the phase list is [`docs/superpowers/plans/2026-09-17-postulator-v2-roadmap.md`](docs/superpowers/plans/2026-09-17-postulator-v2-roadmap.md).

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
10. **Bad code is rewritten, not patched.** Commit your own work on `dev` with conventional commits; never amend, rebase, force-push or `--no-verify`. `master` takes a PR from `dev` only when the owner asks.

## Layout and workflow

`cmd/postulator` (bootstrap) · `internal/kernel` (shared) · `internal/domain` (pure) · `internal/application` (use cases) · `internal/adapters` · `internal/runtime` (run engine) · `internal/transport` (wails, agent) · `internal/app` (composition root) · `frontend` (Vite stub) · `wp-plugin` (PHP).

Dependency rule, enforced by `internal/app/deps_test.go`: domain sees only kernel and stdlib, application sees domain and kernel, only `app` imports `transport`, only `kernel/paging` imports squirrel.

```
task build                 bin/postulator.exe, builds the frontend and bindings first
task package               the NSIS installer; task e2e:full the whole loop on docker
go test -race ./...        golangci-lint run from $(go env GOPATH)/bin
go run ./cmd/covergate     needs coverage.out from -coverprofile
```

## Footguns

- **Two kernel rules**: `log.Logger.Close()` does not call `Sync()`, because syncing a pipe fails on Windows and lumberjack writes unbuffered, so closing the sinks is the whole job; `middleware.Timeout` runs the next handler on its own goroutine and re-panics on the caller's, without which a panic underneath it kills the process.
- **Three rules that look like bugs and are not**: a cursor records the sort it was issued for and refuses another `ORDER BY`; `kernel/dto` keeps its own limit constants rather than importing `kernel/paging`, which would drag squirrel into every DTO; a step a run kind owns (`repair_hierarchy`, `sync_site`, `relink_page`, `revert`) is a `PerKindStepName` and never a `StepName`, which is what keeps it out of `vocab.ts`, the blank template's recipe and three tool enums at once.
- **Live events are dropped, not buffered** — a progress UI replays `RunsService.ListEvents` — and `frontend/src/generated/events.ts` and `vocab.ts` are generated, committed and checked by a test, while `frontend/bindings/` is generated, gitignored and regenerated only by `task build`, so `npm run typecheck` alone cannot see a bound field a Go change removed: run `task build` before a tag. Those generated files and `go.mod` are pinned to LF, and `frontend/dist/.gitkeep` is committed on purpose because `frontend/assets.go` embeds `all:dist`, so Vite runs with `emptyOutDir: false` and unhashed names or a fresh clone stops building.
- **`TestTheToolSchemasFitTheirCeiling` is a deliberate tripwire**: every tool schema is resent on every round of every turn, so a new tool raises `schemaCeilingBytes` in the same commit and a change that shrinks the schemas lowers it again.
- **PowerShell splits `-coverprofile=coverage.out` into two arguments**, which is why every go and task step of the CI Windows job runs under `shell: bash`.
- **`runs.kind` carries a `CHECK` rebuilt by migration 0025** to accept `repair` and `revert`; a ninth kind needs the same rebuild (`NO TRANSACTION`, foreign keys off, explicit `BEGIN`/`COMMIT`) or the insert fails.
- **golangci-lint must be built by the Go release the module targets** and run from `GOPATH/bin`; it runs `errcheck` with `check-blank`, `govet` `enable-all` with non-strict `shadow`, `fieldalignment` off because field order is the JSON order, `gocritic` without `hugeParam`, `misspell` US English ignoring `cancelled`.
- **Docker and WordPress quirks**: the WP-CLI image's `www-data` is uid 82 and the WordPress image's is uid 33, so the compose `bootstrap` service runs as `33:33`, needs the `WORDPRESS_DB_*` variables and `WP_ENVIRONMENT_TYPE=local` for application passwords over HTTP; `modified_gmt` has no timezone suffix and core REST filters `modified_after` on the site-local column, which is why the plugin's `/content?since=` exists; `"meta"` arrives as `[]`, not `{}`, for a post type with no registered meta, and `wp.metaBag` takes the array, the object and null.
- **Two docker stacks from one compose file**: `task sandbox:*` is the owner's WordPress on 8089 (project `postulator-e2e`, never wiped by a task but `sandbox:reset`), `task e2e:*` is the suites' own on 8088 (project `postulator-test`; Windows reserves 8091–8190 here). The e2e suites force-delete whole sections of the site they run on, so both harnesses refuse 8089; the plugin is installed from `bin/postulator-companion.zip`, never bind-mounted, because wp-admin's Delete removed the repository's sources through the mount. The sandbox comes up bare, like a client's site, unless `E2E_PLUGIN=1`.
- **`go test -tags e2e` is the only thing that compiles `internal/e2e` and `internal/adapters/wp/e2e`**, `task lint:e2e` is not optional and `gofmt -l .` is what catches formatting there; `wptest` must never import `wp`, because the fake is a second, independent implementation of the same contract.
- **`VACUUM INTO` cannot write through the adiantum VFS** — it opens the target with no key — so snapshots and restores go through the SQLite online backup API with a plain `file:` URI.
- **A template placeholder is expanded in `templates.ResolveForPage` and nowhere else**; `template.Validate` refuses any `{…}` outside `template.Placeholders()`, so a heading the validator compares is never the placeholder itself. The writer answers a `content.DraftAnswer` with a slot per section and `content.Assemble` puts the brief's headings on; do not compare a body with the raw spec.
- **`run_items.blocked_by` has no foreign key on purpose** — SQLite's `DROP COLUMN` refuses a constrained column, so the down migration would not round-trip — and the release predicate lives twice by design, in `selectRunnableItems` and `selectItemsAwaitingParent`: the blocker's page when there is a blocker, else `pages.parent_page_id`. Change one and the other.
- **A step that returns `TransitionWait` with no artifact and no checkpoint is recorded `ExecStarted`**, never `ExecDone`, or `reuse()` skips it when the item wakes. A step that returns `TransitionPause` keeps its artifacts and is recorded `ExecFailed` with the note, and `sc.Accepted()` is how it learns a person accepted its findings.
- **`Engine.EstimateRun` takes the run alone** and resolves each target's template itself; `runtime.Deps.Keys` and `steps.Deps.ImageModel` are required, not optional, and every harness that builds an engine passes a key store. The writer's ceiling is `writerCeiling(spec, attempts)` and the adapter clamps it; `tokensPerWord` in `runtime/budget.go` (2.2) is the estimate's and `steps.tokensPerWord` (3) the ceiling's, on purpose.

## Standing rulings

Dated in full in [`docs/DECISIONS.md`](docs/DECISIONS.md); these are the ones that change how you write code here.

- **2026-09-17** — UUIDs come from the Go 1.27 standard library; `kernel/id.Valid` does its own canonical v4 check. `paging.Cut` is a `Keyset` method. `errors.Wrap` returns `error`, never `*Error`. The kernel exports nothing whose only caller is its own test.
- **2026-09-17 (phase 1B)** — A service method returns `wails.Convert(err)`, never the raw error; the error marshaller is set per service with `NewServiceWithOptions`; `application.RegisterEvent` is not used, the Go registry renders the typings.
- **2026-09-18 (phase 3)** — `adapters/wp` is the one place page numbers are legal; `wp.ContentHash` is `hex(sha256(raw))` and frozen; `wp-plugin/openapi.yaml` is the contract neither side changes alone; the plugin never removes a kses filter and passes `wp_slash` on every write.
- **2026-09-18 (phases 9-10)** — `robfig/cron/v3` and `x/net/html` are the only third-party packages the domain may import; `internal/transport/agent` is the only package that may name gollem; a tool call's arguments are masked before they reach a ledger, an event or a summary.
- **2026-09-18 (phase 11)** — A Wails service declares the use-case interface it consumes and `Services` takes a `wails.Deps` from the composition root; `SettingsService` is the only surface that accepts a credential.
- **2026-09-18 (phase 12)** — A Wails service resolves its use case per call through a `Source[T]`, which answers `Locked` while the core carries no composition; `app.Open` starts locked when `master.key.pw` exists; `app.Config.Provider` is the seam an end-to-end harness uses to compose the real application over `adapters/llm/fake`.
- **2026-09-22/23 (hardening)** — A run kind that owns its steps hands them out and no template overrides them; a revert is a run, and every write to the site keeps what it replaced beside it in the artifact the step already produces; one `pagemap.Site` answers where an href points and `content.Subject` says which page is "self"; a tool takes its own argument structs, never the window's DTOs.
- **2026-09-24 (2.2.0)** — An import never plans `/`; a draft is assembled from the brief and a missing required section is tried again, never failed; the linker owes phrases and writes a plain one itself before it gives up; validation grades and a person accepts, through `Engine.Accept`; a child is queued behind its parent through `blocked_by`; the estimate prices each page on its own template and refuses a run its preflight blocks; every step event carries the step's sentence.

## Product guardrail

Postulator turns an entity graph into graph-compliant WordPress pages; reject features that do not serve that loop.
