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

**Phase 1 is next:** infrastructure and the contracts spike, two Opus agents in
parallel. One takes the SQLite store, migrations, unit of work and test helper plus the
secrets adapter (DPAPI, AES-GCM, adiantum); the other spikes Wails v3 on errors, events
and generics and rewrites the open sections of `docs/CONTRACTS.md` with the answers.

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
- **Vite writes unhashed asset names with `emptyOutDir` off**, so the committed
  `frontend/dist/.gitkeep` survives a build. Without it `go build ./...` fails on a
  fresh clone, because `frontend/assets.go` embeds a directory the frontend build has
  not created yet.

## Open questions for Phase 1

- The transport error format, the event envelope and whether the TypeScript generator
  survives generics in exported signatures. All three are marked open in
  `docs/CONTRACTS.md`.
- Whether the adiantum VFS and `ncruces/go-sqlite3` behave under the WAL and busy-timeout
  pragmas the spec asks for.

## Known gaps

- `internal/domain`, `internal/application`, `internal/adapters`, `internal/runtime` and
  `internal/transport` do not exist yet. The dependency-rule test skips each rule whose
  tree is absent and starts enforcing it the day the first package lands. The
  domain+application coverage gate reports itself skipped for the same reason.
- `lefthook` is not installed on the development machine. Install it with
  `go install github.com/evilmartians/lefthook@latest && lefthook install`.
- The frontend is a stub that prints the build info. It is replaced in Phase 11.

## Fixes the review asks for before Phase 1 code lands

- `errors.Wrap(nil, code, msg)` returns a typed nil. Returned straight out of a function
  whose result is `error` it is not nil, and `CodeOf` on it dereferences a nil pointer.
  Guard the nil receiver in `Error`, `Unwrap` and `CodeOf`, or have `Wrap` return
  `error`. Adapters will write `return errors.Wrap(rows.Err(), External, …)`.
- `internal/app` has no test of its own code: `HealthService.Ping`, `BuildInfo` and
  `Build` are at 0% while `deps_test.go` only shells out to `go list`.
- `frontend/package.json` pins `@wailsio/runtime` to `latest` and the build task runs
  `npm install`. Pin the version and use `npm ci` so a CI build cannot drift.
- `.github/workflows/ci.yml` installs Task `v3.45.6` while `release.yml`, this file and
  `docs/CONVENTIONS.md` all say `v3.53.1`.
