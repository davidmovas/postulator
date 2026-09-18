# Conventions

## Pinned toolchain

| Tool | Version | Where it is pinned |
|---|---|---|
| Go | `go 1.27` directive, toolchain `go1.27.0` resolved by `GOTOOLCHAIN=auto` | `go.mod`, `.github/workflows/ci.yml` |
| Wails v3 CLI | `v3.0.0-beta.23` (latest v3 tag, 2026-09-16) | `go.mod` require, `.github/workflows/*.yml` |
| golangci-lint | `v2.13.2` | `.github/workflows/ci.yml` |
| lefthook | latest; install with `go install github.com/evilmartians/lefthook@latest && lefthook install` | `lefthook.yml` |
| Task | `v3.53.1` | `.github/workflows/*.yml` |
| Node | `22` | `.github/workflows/ci.yml` |
| squirrel / zap / lumberjack | `v1.5.4` / `v1.27.0` / `v2.2.1` | `go.mod` |
| ncruces/go-sqlite3 | `v0.30.1` | `go.mod` |
| goose | `v3.28.0` | `go.mod` |
| golang.org/x/sys | `v0.47.0` | `go.mod` |

`wails3 doctor` must pass before any Wails work. It verified WebView2 `153.0.4234.32`
and NSIS `v3.12` on the development machine. golangci-lint has to be built by the same
Go release the module targets: a binary built with go1.26 refuses a `go 1.27` module, so
install it with `go install`, not from a prebuilt archive.

The reasoning behind each tuned lint rule is in `CLAUDE.md`, not in `.golangci.yml`,
because this repository forbids comments in YAML.

## Identifiers

Go 1.27 ships a stdlib `uuid` package (import path `uuid`, RFC 9562). It provides
`New`, `NewV4`, `NewV7`, `Parse`, `MustParse`, `Nil`, `Max` and `UUID.String`.
**Decision: use the stdlib package; `github.com/google/uuid` is not a dependency.**

The stdlib package has no `Version()`/`Variant()` accessor, so `kernel/id.Valid`
validates the canonical 8-4-4-4-12 textual form and checks the version nibble is
`4` and the variant nibble is one of `8 9 a b` itself. `Parse` alone is not enough:
it also accepts braced, URN and unhyphenated forms and any version.

`kernel/id.New()` returns `uuid.NewV4().String()` — UUID v4 as lowercase TEXT.

## Timestamps

RFC3339 with a UTC offset, in SQLite TEXT columns and in every DTO. `kernel/dto.Time`
is the only marshaller; it normalises to UTC on the way out and parses RFC3339 on the
way in. Domain code uses `time.Time` and takes the current instant from
`kernel/clock.Clock`, never from `time.Now()` directly.

## Migrations

Migrations are embedded SQL run by goose v3 at `Store.Open`. They live in
`internal/adapters/sqlite/migrations` and are embedded with `//go:embed migrations/*.sql`.

- Filenames are `NNNN_snake_case.sql`: four digits, zero-padded, gapless, never
  renumbered once committed.
- Every file carries both `-- +goose Up` and `-- +goose Down`. A migration that cannot be
  reversed does not ship; the round-trip test applies up, down and up again for all of
  them.
- One concern per migration. The down section drops exactly what the up section created,
  in reverse order.
- Tables are `STRICT`. Identifiers are lowercase snake_case, keywords uppercase, one
  column per line. Text primary keys are `TEXT PRIMARY KEY`; timestamps are
  `TEXT NOT NULL` holding RFC3339 UTC.
- No `IF NOT EXISTS` and no `IF EXISTS`: a database that is not in a known state must
  fail loudly rather than drift.
- `goose_db_version` belongs to goose and is never read from application code.

Files are scaffolded with the goose CLI and renamed to the four-digit form:

```
go run github.com/pressly/goose/v3/cmd/goose@v3.28.0 -s -dir internal/adapters/sqlite/migrations create <name> sql
```

The CLI is never pointed at a Postulator database: its `sqlite3` dialect is backed by
`modernc.org/sqlite`, which cannot read an adiantum-encrypted file. Migrations are applied
only through `Store.Open`, which uses `goose.NewProvider` so that two stores opening in
parallel tests cannot race on goose's package-level filesystem and dialect.

## Code rules

1. No comments in Go, TypeScript, YAML or PHP — no inline comments, no package docs,
   no godoc. Naming carries the meaning. Prose lives in `docs/` and `CLAUDE.md`.
2. Interfaces are declared by the consumer. A package that needs a dependency declares
   the smallest interface it uses; producers return concrete types.
3. No stubs, no `TODO`, no placeholder implementations. Unfinished work is not merged.
4. No hardcoded values that belong to data: model prices, templates, event names and
   step names live in registries, catalogs or seed files.
5. Bad code is rewritten, not patched.
6. JSON is camelCase on every boundary: `siteId`, `nextCursor`, `createdAt`.
7. Cursor (keyset) pagination everywhere. Offset pagination does not exist in this
   codebase.
8. Errors crossing a package boundary are `*kernel/errors.Error` with a frozen `Code`.
   Adapters convert driver and transport errors at their own boundary.
9. Secrets never reach logs, artifacts or plain DB columns. `kernel/log` redacts
   `password`, `apiKey`, `token` and `authorization` by key.
10. Generated content is English only.

## Layout and dependency rule

```
internal/kernel      shared primitives, imports stdlib (+ squirrel in paging, zap in log)
internal/domain      pure logic; imports kernel and stdlib only
internal/application use cases; imports domain and kernel
internal/adapters    sqlite, wp, llm, images, secrets, importer
internal/runtime     run engine
internal/transport   wails services, agent tools
internal/app         composition root; the only package that may import transport
```

`internal/app/deps_test.go` enforces this with `go list -deps`. `paging` is the only
kernel package allowed to import squirrel; domain never sees a query builder.

The squirrel rule is deliberately checked on **direct imports only**: application may
reach squirrel transitively through `kernel/paging`, domain may not, and the domain
closure test is what catches that transitive case.

## Naming

- Packages are singular nouns (`site`, `graph`, `run`), never `utils`, `common` or
  `helpers`.
- Constructors are `New`, `NewX` or `Open`; no `Make`, no `Create` for constructors.
- Test helpers live in `<pkg>test` packages (`sqlitetest`, `wptest`).
- Exported enum values carry their type as a prefix only when the bare word would be
  ambiguous (`StatusRunning`, not `RunStatusRunning`).
- Files are named after what they hold: one step per file, one tool per file, one
  repository per bounded context.

## Testing

- TDD: the failing test is written and run before the implementation.
- Table tests with named cases are the default shape; `t.Parallel()` where the test
  owns no shared state.
- Coverage gates: `internal/domain` + `internal/application` ≥ 80%, repository total
  ≥ 70%, enforced by `go run ./cmd/covergate`. `internal/kernel` targets ≥ 90%.
- Every test runs under `-race`.
- No mocks of types we own where a fake is cheaper: `clock.Fake`, `wptest.Server`,
  the scripted LLM client and a real SQLite file are preferred over generated mocks.
- LLM behaviour is tested by record/replay fixtures under `testdata/llm`, never by a
  live provider call.

## Commits

Conventional commits: `<type>(<scope>): <subject>` with
`type ∈ feat|fix|perf|refactor|docs|test|style|chore|ci|build|revert`. The scope names
the package or the phase task (`feat(kernel): cursor pagination`). The commit-msg hook
in `lefthook.yml` rejects anything else. Every commit ends with the attribution line
the session was given. Commit on `rewrite/v2`; never amend, rebase or force-push.

## Verification

```
task build
go vet ./...
golangci-lint run
go test -race -cover ./...
go run ./cmd/covergate
```

All five must be green before a phase is declared done, and `docs/STATUS.md` is updated
in the same commit.

## Docker end-to-end

`docker/e2e/compose.yaml` pins WordPress 7.0.1 (PHP 8.3), MariaDB 11.4.12 and WP-CLI 2.12.0
on `127.0.0.1:8089`; `task e2e:up` provisions the site and writes `.env.generated`.

```
task e2e:up                E2E_SEO=none|yoast|rankmath, E2E_WOO=0|1
task e2e:test              the plugin contract, ./internal/adapters/wp/e2e/...
task e2e:full              the whole loop, ./internal/e2e/...
task e2e:down              e2e:reset does both
```

`internal/e2e` composes the real application over `adapters/llm/fake`, syncs the docker
site, imports `examples/sitemap-import-example.xlsx`, generates five guide pages as drafts
and reads the result back through the companion plugin. It deletes everything under
`/menu/` on the site before it starts, so it can be run again without resetting the stack.

Both suites are `_test.go` behind `//go:build e2e`: it adds nothing to the coverage
profile and the default lint never sees it, so use `task lint:e2e` and `gofmt -l .`.
`task plugin:lint` uses the pinned image's `php` on Windows and a local `php` on CI,
which also packages the plugin and never starts the stack.
