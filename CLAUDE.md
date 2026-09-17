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
- **The tuned lint rules and why**: `errcheck` runs with `check-blank` and `check-type-assertions` and no baseline, so `_ = f()` is a finding rather than an escape hatch. `govet` is `enable-all` with `shadow` non-strict, because strict mode reports every `if err := f(); err != nil`. `fieldalignment` is off: it orders struct fields by machine layout, and our field order is the JSON order we owe the frontend. `gocritic` runs diagnostic, style and performance with `hugeParam` off. `misspell` is US English and ignores `cancelled`, which is a domain status value.

## Standing rulings

- **2026-09-17** — UUIDs come from the Go 1.27 standard library `uuid` package; `github.com/google/uuid` is not a dependency. `kernel/id.Valid` does its own canonical v4 check, because `uuid.Parse` also accepts braced, URN and unhyphenated text and any version.
- **2026-09-17** — `paging.Cut` is a `Keyset` method, not the free function the spec sketched: the accessors already live on the keyset, and passing them per call is the one way to make a cursor disagree with its query.
- **2026-09-17** — `.golangci.yml` carries no inline rationale, against the letter of task 0.8, because the no-comments-in-YAML rule outranks it. The rationale is the Footguns entry above.
- **2026-09-17** — The Wails template is `vanilla` (Vanilla + TypeScript + Vite); beta.23 renamed `vanilla-ts`.

## Product guardrail

Postulator turns an entity graph into graph-compliant WordPress pages; reject features that do not serve that loop.
