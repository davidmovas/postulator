# Conventions

## Pinned toolchain

| Tool | Version | Where it is pinned |
|---|---|---|
| Go | `go 1.27` directive, toolchain `go1.27.0` resolved by `GOTOOLCHAIN=auto` | `go.mod`, `.github/workflows/ci.yml` |
| Wails v3 CLI | `v3.0.0-beta.23` (latest v3 tag, 2026-09-16) | `go.mod` require, `.github/workflows/*.yml` |
| golangci-lint | `v2.11.4` | `.github/workflows/ci.yml` |
| lefthook | `v1.13.6` | `lefthook.yml` header is absent by the no-comments rule; the version lives in CI and in this table |
| Task | `v3` (`go-task/task`) | `Taskfile.yml` `version: '3'` |
| Node | `22` | `.github/workflows/ci.yml` |

`wails3 doctor` must pass before any Wails work. It verified WebView2 `153.0.4234.32` and NSIS `v3.12` on the development machine.

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
