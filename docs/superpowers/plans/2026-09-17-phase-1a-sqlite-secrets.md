# Phase 1A — SQLite Store, Migrations, Unit of Work and Secrets Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the persistence and secrets infrastructure of Postulator v2 — an encrypted SQLite store with embedded goose migrations, a single-writer unit of work, a driver-error-to-kernel-error boundary, a settings repository with a loader, and a DPAPI-backed master key with AES-GCM secrets — so Phase 2 can write domain tables and repositories against a finished substrate.

**Architecture:** One `*Store` owns two `database/sql` pools over the same file: a writer pinned to a single connection and a read-only reader pool, both opened through `github.com/ncruces/go-sqlite3` (pure Go, CGO off) with the adiantum VFS when a key is supplied. `Store.Do` begins one immediate transaction, puts it in the context, and repositories read it back through `execFrom`/`writeFrom`; every driver error is converted to a `*kernel/errors.Error` by extended result code at the `dbx` boundary, never by substring. Secrets are AES-GCM sealed with a 32-byte master key that lives DPAPI-protected on disk, and only the ciphertext ever reaches a column.

**Tech Stack:** Go 1.27 (CGO off), `github.com/ncruces/go-sqlite3` v0.30.1 (`driver`, `embed`, `vfs/adiantum`), `github.com/pressly/goose/v3` v3.26.0, `golang.org/x/sys/windows` v0.46.0, stdlib `crypto/aes`, `crypto/cipher`, `crypto/rand`, `database/sql`, `embed`.

**Spec:** `docs/superpowers/specs/2026-09-17-postulator-v2-design.md` (sections 3, 4, 9.1, 9.2, 15); phase row in `docs/superpowers/plans/2026-09-17-postulator-v2-roadmap.md` section 11 and verification in section 13.

## Global Constraints

- **No comments in Go, TypeScript, YAML or PHP** — not inline, not godoc, not package docs. Naming carries meaning; prose lives in `docs/` and `CLAUDE.md`.
- **No stubs, no TODOs, no placeholders.** If something cannot be done, stop and say so.
- **Interfaces are declared by the consumer**, never by the implementation and never in advance.
- **TDD.** Failing test first, run it, implement, run it, commit. Table tests. Gates: `domain`+`application` >= 80%, module >= 70%, kernel >= 90%. This phase additionally targets >= 85% for every package it creates.
- **Cursor pagination only.** No offset, anywhere.
- **JSON is camelCase**, timestamps are RFC3339 UTC, ids are UUID v4 text.
- **Errors crossing a boundary are `*kernel/errors.Error`** with a frozen code. Adapters convert driver and transport errors at their own edge.
- **Secrets never reach a log, an artifact or a plain column.**
- **No new `.md` files** beyond `CLAUDE.md` and the set in section 10 of the spec.
- **Bad code is rewritten, not patched.** Commit on `rewrite/v2` with conventional commits; never amend, rebase, force-push or `--no-verify`.
- Windows only. Go 1.27 directive, `GOTOOLCHAIN=auto`. Wails v3 `v3.0.0-beta.23`, golangci-lint `v2.13.2`, Task `v3.53.1`, Node `22`, squirrel `v1.5.4`, zap `v1.27.0`, lumberjack `v2.2.1`.
- Never port Asynq, Postgres, jet, pgx, fx, Fiber, credits or HTTP status plumbing from Archond. Port shapes only.
- Every commit message ends with the trailer `Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>`.
- Kernel names are reused exactly: `errors.NotFound|Conflict|Invalid|External|Internal|Cancelled`, `errors.New`, `errors.Wrap`, `errors.CodeOf`, `errors.IsCode`, `(*Error).WithRetry`, `(*Error).WithDetail`; `settings.Registry`, `settings.Values`, `settings.Default()`, `(*Registry).NewValues`, `(*Registry).Apply`; `clock.Clock`, `clock.System`, `clock.NewFake`; `log.Logger`; `id.New`, `id.Valid`.

---

## Design decisions taken by this plan

1. **`execFrom(ctx)` distinguishes tx from db by an unexported context key.** `Store.Do` stores the live `*sql.Tx` under `txKey{}`; `execFrom(ctx)` type-asserts it back. Because two pools exist, the single accessor the spec sketches becomes two: `execFrom(ctx)` returns the transaction when `Do` is active and the **reader** pool otherwise (the read path), and `writeFrom(ctx)` returns the transaction when `Do` is active and the **writer** pool otherwise (the write path). One accessor cannot serve both without making the reader pool dead code.
2. **Two pools, not one.** A writer `*sql.DB` with `SetMaxOpenConns(1)` and `_txlock=immediate`, plus a second `*sql.DB` opened `mode=ro` with `SetMaxOpenConns(4)` — because one pool capped at a single connection would serialise every read behind a long write, while an uncapped pool would let two goroutines both begin write transactions and turn WAL's single-writer rule into `SQLITE_BUSY` at commit time instead of at begin time.
3. **Nested `Do` reuses the outer transaction and never opens a savepoint.** `Do` returns `fn(ctx)` directly when a transaction is already in the context: SQLite has exactly one writer, and a real nested savepoint would let an inner rollback be swallowed while the outer commits, which is precisely the partial-write bug the unit of work exists to make impossible. The outermost `Do` owns commit and rollback.
4. **Migrations run through `goose.NewProvider`, not `goose.UpContext`.** `UpContext` reads the package-level `baseFS` and dialect that `SetBaseFS`/`SetDialect` mutate; `sqlitetest.Open` is called from parallel tests under `-race`, and two stores opening at once would race on those globals. `NewProvider(goose.DialectSQLite3, db, fsys)` carries the filesystem, the dialect and the version table on the instance and has the same up/down semantics.
5. **The adiantum key travels as the `hexkey` URI parameter, not as a `PRAGMA`.** `hbshVFS.OpenFilename` reads `hexkey` from the URI at file-open time, which is strictly before any SQL runs, so the key can never be ordered after `journal_mode(WAL)`; the PRAGMA form would have to be quoted inside a `_pragma=` query value and a 64-character hex string starting with a digit is not a safe bare pragma token.
6. **The master key directory comes from `os.UserConfigDir()`, not `github.com/adrg/xdg`.** On Windows `os.UserConfigDir()` returns `%APPDATA%`, which is the path section 9.2 names; `xdg.ConfigHome` resolves to `%LOCALAPPDATA%` (see `paths_windows.go`, `baseDirs.configHome = kf.localAppData`) and would put the key in the wrong place. `xdg` also stays an indirect dependency this way.
7. **No `//go:build windows` tags.** The application is Windows-only, `golang.org/x/sys/windows` compiles nowhere else, and a tag would demand a second implementation file that would be a stub.
8. **Deviation, task 2:** commit `634aedd` changed `errors.Wrap` to return `error` and deleted `errors.RetryAfter`, so `dbx.Convert` builds the busy case as `errors.New(code, message).WithInternal(err).WithRetry(BusyRetryAfter)` and the test reads `Retry.After` through `errors.As` instead.
9. **Deviation, task 10:** the "protected blob does not contain the plaintext" assertion only runs for plaintexts of at least 8 bytes, because a one-byte `0x00` occurs naturally in every DPAPI blob header and the check was vacuous rather than failing.

## The adiantum open sequence

```
Open(path, key)
  key == nil  -> dsn has no vfs= and no hexkey=; the default VFS is used
  key != nil  -> dsn carries vfs=adiantum&hexkey=<64 hex chars>
1. writer := driver.Open(dsn(path, key, false))
     the blank import of vfs/adiantum registered the "adiantum" VFS at init
     sqlite3_open_v2 runs, hbshVFS.OpenFilename reads hexkey and builds the cipher
     then the _pragma list executes in URI order:
       busy_timeout(5000), foreign_keys(1), journal_mode(WAL), synchronous(NORMAL)
2. writer.SetMaxOpenConns(1)
3. store.migrate(ctx) -> goose.NewProvider(DialectSQLite3, writer, migrations).Up
4. reader := driver.Open(dsn(path, key, true))
     same vfs= and hexkey=, plus mode=ro
     _pragma list is busy_timeout(5000), foreign_keys(1) only:
     journal_mode and synchronous are writer-side and a read-only handle refuses them
5. reader.SetMaxOpenConns(4)
```

The reader is opened **after** the writer has migrated, so the file, `-wal` and `-shm` all exist. A wrong key surfaces as `SQLITE_NOTADB` from the first statement, which `dbx.Classify` maps to `Internal`.

Tests use the helper both ways: `sqlitetest.Open(t)` passes `nil` (no crypto, fast, used by every repository test) and `sqlitetest.OpenEncrypted(t, key)` passes 32 bytes (used by the adiantum round-trip and wrong-key tests in `store_test.go`).

## Migration conventions (written into `docs/CONVENTIONS.md` by task 14)

- Filenames are `NNNN_snake_case.sql`, four digits, zero-padded, gapless, never renumbered once committed.
- Every file has both `-- +goose Up` and `-- +goose Down`. A migration that cannot be reversed does not ship.
- One concern per migration; the down section drops exactly what the up section created, in reverse order.
- Tables are `STRICT`. Text primary keys are `TEXT PRIMARY KEY`. Timestamps are `TEXT NOT NULL` holding RFC3339 UTC.
- No `IF NOT EXISTS`, no `IF EXISTS`: a migration that is not in a known state must fail loudly.
- SQL keywords uppercase, identifiers lowercase snake_case, one column per line, trailing comma never.
- Goose's own `goose_db_version` table is not ours and is never referenced from application code.

The goose CLI is used only to scaffold files; it is never pointed at a Postulator database, because its `sqlite3` dialect is backed by `modernc.org/sqlite`, which cannot read an adiantum-encrypted file.

```
go run github.com/pressly/goose/v3/cmd/goose@v3.26.0 -s -dir internal/adapters/sqlite/migrations create app_meta sql
go run github.com/pressly/goose/v3/cmd/goose@v3.26.0 -s -dir internal/adapters/sqlite/migrations create settings sql
go run github.com/pressly/goose/v3/cmd/goose@v3.26.0 -s -dir internal/adapters/sqlite/migrations create secrets sql
git -C . mv internal/adapters/sqlite/migrations/00001_app_meta.sql internal/adapters/sqlite/migrations/0001_app_meta.sql
git -C . mv internal/adapters/sqlite/migrations/00002_settings.sql internal/adapters/sqlite/migrations/0002_settings.sql
git -C . mv internal/adapters/sqlite/migrations/00003_secrets.sql internal/adapters/sqlite/migrations/0003_secrets.sql
```

## File structure

| File | Responsibility |
|---|---|
| `internal/adapters/sqlite/migrations/0001_app_meta.sql` | `app_meta` key/value table seeded with `schema_version` |
| `internal/adapters/sqlite/migrations/0002_settings.sql` | `settings` table: key, JSON value, updated_at |
| `internal/adapters/sqlite/migrations/0003_secrets.sql` | `secrets` table: ref, ciphertext blob, created_at, updated_at |
| `internal/adapters/sqlite/migrations.go` | `//go:embed migrations/*.sql` and the `fs.Sub` accessor |
| `internal/adapters/sqlite/migrations_test.go` | embedded-file inventory and the up/down/up round trip |
| `internal/adapters/sqlite/dsn.go` | builds the `file:` URI with pragmas, vfs, hexkey and mode |
| `internal/adapters/sqlite/dsn_test.go` | table test over the generated URI |
| `internal/adapters/sqlite/store.go` | `Open`, `Close`, the two pools, `migrate`, `provider` |
| `internal/adapters/sqlite/store_test.go` | pragmas, adiantum with and without a key, wrong key, reader is read-only |
| `internal/adapters/sqlite/tx.go` | `Do`, `execFrom`, `writeFrom`, the tx context key, the `executor` interface |
| `internal/adapters/sqlite/tx_test.go` | commit, rollback, nesting, reader-vs-tx routing |
| `internal/adapters/sqlite/dbx/errors.go` | extended result code to kernel code mapping, `Convert`, `IsNotFound`, `IsConflict` |
| `internal/adapters/sqlite/dbx/errors_test.go` | table test of the mapping, including retry on busy |
| `internal/adapters/sqlite/dbx/result.go` | `Result[T]` with `NotFound`, `Conflict`, `WrapErr`, `Unwrap` |
| `internal/adapters/sqlite/dbx/result_test.go` | table test of the monad |
| `internal/adapters/sqlite/settings_repo.go` | `SettingsRepo.Get/Set/All` over the `settings` table |
| `internal/adapters/sqlite/settings_repo_test.go` | round trip against a real file |
| `internal/adapters/sqlite/secrets_repo.go` | `SecretsRepo.Put/Get/Delete` over the `secrets` table |
| `internal/adapters/sqlite/secrets_repo_test.go` | round trip and the not-found code |
| `internal/adapters/sqlite/sqlitetest/sqlitetest.go` | `Open(t)` and `OpenEncrypted(t, key)` on `t.TempDir()` |
| `internal/adapters/secrets/dpapi/dpapi.go` | `Protect`/`Unprotect` over CryptProtectData, user scope |
| `internal/adapters/secrets/dpapi/dpapi_test.go` | round trip and tamper rejection |
| `internal/adapters/secrets/aesgcm/aesgcm.go` | `Seal`/`Open` with a random nonce and the `v1:` envelope |
| `internal/adapters/secrets/aesgcm/aesgcm_test.go` | round trip, nonce freshness, bad envelope, wrong key |
| `internal/adapters/secrets/masterkey/masterkey.go` | `Load(dir)` creating or reading `master.key` |
| `internal/adapters/secrets/masterkey/masterkey_test.go` | creation, stability across calls, protection on disk |
| `internal/adapters/secrets/store.go` | `Store` sealing values into the vault and reading them back |
| `internal/adapters/secrets/store_test.go` | round trip against a fake vault, ciphertext is not plaintext |
| `internal/app/settings.go` | the consumer-side `settingsSource` interface and `LoadSettings` |
| `internal/app/settings_test.go` | hydration, validation failure, unknown keys |
| `internal/app/core.go` | `Config`, `DefaultConfig`, `Core`, `Open`, `Close` |
| `internal/app/core_test.go` | open/close round trip, `HealthService.Ping`, `Build` |
| `cmd/postulator/main.go` | opens the core before the window and closes it after `Run` |
| `docs/CONVENTIONS.md` | gains the migration section |
| `docs/STATUS.md` | gains the Phase 1A entry |

---

### Task 1: Dependencies and embedded migrations

**Files:**
- Create: `internal/adapters/sqlite/migrations/0001_app_meta.sql`
- Create: `internal/adapters/sqlite/migrations/0002_settings.sql`
- Create: `internal/adapters/sqlite/migrations/0003_secrets.sql`
- Create: `internal/adapters/sqlite/migrations.go`
- Test: `internal/adapters/sqlite/migrations_test.go`
- Modify: `go.mod`, `go.sum`

**Interfaces:**
- Consumes: nothing.
- Produces: `func migrations() (fs.FS, error)` in package `sqlite`, returning the embedded migration directory rooted so that `fs.Glob(fsys, "*.sql")` lists the three files.

- [ ] **Step 1: Add the dependencies**

```bash
go get github.com/ncruces/go-sqlite3@v0.30.1
go get github.com/pressly/goose/v3@v3.26.0
go get golang.org/x/sys@v0.46.0
go mod tidy
```

- [ ] **Step 2: Write the failing test**

Create `internal/adapters/sqlite/migrations_test.go`:

```go
package sqlite

import (
	"io/fs"
	"slices"
	"strings"
	"testing"
)

func TestMigrationsAreEmbedded(t *testing.T) {
	t.Parallel()

	fsys, err := migrations()
	if err != nil {
		t.Fatalf("migrations: %v", err)
	}

	names, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}

	want := []string{"0001_app_meta.sql", "0002_settings.sql", "0003_secrets.sql"}
	if !slices.Equal(names, want) {
		t.Fatalf("embedded migrations = %v, want %v", names, want)
	}

	for _, name := range names {
		body, readErr := fs.ReadFile(fsys, name)
		if readErr != nil {
			t.Fatalf("read %s: %v", name, readErr)
		}
		for _, marker := range []string{"-- +goose Up", "-- +goose Down", "STRICT"} {
			if !strings.Contains(string(body), marker) {
				t.Errorf("%s is missing %q", name, marker)
			}
		}
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/adapters/sqlite/ -run TestMigrationsAreEmbedded -v`
Expected: FAIL, the package does not build because `migrations` is undefined.

- [ ] **Step 4: Write the three migrations**

`internal/adapters/sqlite/migrations/0001_app_meta.sql`:

```sql
-- +goose Up
CREATE TABLE app_meta (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    created_at TEXT NOT NULL
) STRICT;

INSERT INTO app_meta (key, value, created_at)
VALUES ('schema_version', '1', strftime('%Y-%m-%dT%H:%M:%SZ', 'now'));

-- +goose Down
DROP TABLE app_meta;
```

`internal/adapters/sqlite/migrations/0002_settings.sql`:

```sql
-- +goose Up
CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TEXT NOT NULL
) STRICT;

-- +goose Down
DROP TABLE settings;
```

`internal/adapters/sqlite/migrations/0003_secrets.sql`:

```sql
-- +goose Up
CREATE TABLE secrets (
    ref TEXT PRIMARY KEY,
    ciphertext BLOB NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
) STRICT;

-- +goose Down
DROP TABLE secrets;
```

- [ ] **Step 5: Write the embed accessor**

Create `internal/adapters/sqlite/migrations.go`:

```go
package sqlite

import (
	"embed"
	"io/fs"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func migrations() (fs.FS, error) {
	fsys, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "read embedded migrations")
	}
	return fsys, nil
}
```

- [ ] **Step 6: Run test to verify it passes**

Run: `go test ./internal/adapters/sqlite/ -run TestMigrationsAreEmbedded -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add go.mod go.sum internal/adapters/sqlite/migrations.go internal/adapters/sqlite/migrations/ internal/adapters/sqlite/migrations_test.go
git commit -m "build(sqlite): embed the goose migrations for app_meta, settings and secrets

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 2: Driver error classification

**Files:**
- Create: `internal/adapters/sqlite/dbx/errors.go`
- Test: `internal/adapters/sqlite/dbx/errors_test.go`

**Interfaces:**
- Consumes: `errors.Code`, `errors.NotFound`, `errors.Conflict`, `errors.Invalid`, `errors.External`, `errors.Internal`, `errors.Cancelled`, `errors.New`, `errors.Wrap`, `errors.IsCode`, `(*errors.Error).WithRetry` from `internal/kernel/errors`.
- Produces:
  - `func Classify(err error) errors.Code`
  - `func Convert(err error, message string) error`
  - `func IsNotFound(err error) bool`
  - `func IsConflict(err error) bool`
  - `const BusyRetryAfter = 250 * time.Millisecond`

- [ ] **Step 1: Write the failing test**

Create `internal/adapters/sqlite/dbx/errors_test.go`:

```go
package dbx

import (
	"context"
	"database/sql"
	stderrors "errors"
	"io"
	"testing"

	"github.com/ncruces/go-sqlite3"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestClassify(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want errors.Code
	}{
		{name: "no error", err: nil, want: ""},
		{name: "no rows", err: sql.ErrNoRows, want: errors.NotFound},
		{name: "not found opcode", err: sqlite3.NOTFOUND, want: errors.NotFound},
		{name: "unique", err: sqlite3.CONSTRAINT_UNIQUE, want: errors.Conflict},
		{name: "primary key", err: sqlite3.CONSTRAINT_PRIMARYKEY, want: errors.Conflict},
		{name: "foreign key", err: sqlite3.CONSTRAINT_FOREIGNKEY, want: errors.Invalid},
		{name: "not null", err: sqlite3.CONSTRAINT_NOTNULL, want: errors.Invalid},
		{name: "check", err: sqlite3.CONSTRAINT_CHECK, want: errors.Invalid},
		{name: "busy", err: sqlite3.BUSY, want: errors.External},
		{name: "busy timeout", err: sqlite3.BUSY_TIMEOUT, want: errors.External},
		{name: "locked", err: sqlite3.LOCKED, want: errors.External},
		{name: "interrupt", err: sqlite3.INTERRUPT, want: errors.Cancelled},
		{name: "context cancelled", err: context.Canceled, want: errors.Cancelled},
		{name: "read only", err: sqlite3.READONLY, want: errors.Internal},
		{name: "foreign error", err: io.EOF, want: errors.Internal},
		{name: "wrapped unique", err: stderrors.Join(io.EOF, sqlite3.CONSTRAINT_UNIQUE), want: errors.Conflict},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := Classify(tc.err); got != tc.want {
				t.Errorf("Classify(%v) = %q, want %q", tc.err, got, tc.want)
			}
		})
	}
}

func TestConvert(t *testing.T) {
	t.Parallel()

	if Convert(nil, "read row") != nil {
		t.Fatal("Convert(nil) must be nil")
	}

	already := errors.New(errors.NeedsHuman, "left alone")
	if got := Convert(already, "read row"); !stderrors.Is(got, already) {
		t.Errorf("Convert must not rewrap a kernel error, got %v", got)
	}

	converted := Convert(sqlite3.CONSTRAINT_UNIQUE, "insert secret")
	if !errors.IsCode(converted, errors.Conflict) {
		t.Errorf("code = %q, want %q", errors.CodeOf(converted), errors.Conflict)
	}
	if !stderrors.Is(converted, sqlite3.CONSTRAINT_UNIQUE) {
		t.Error("the driver error must stay reachable through Unwrap")
	}

	busy := Convert(sqlite3.BUSY, "begin transaction")
	after, ok := errors.RetryAfter(busy)
	if !ok || after != BusyRetryAfter {
		t.Errorf("RetryAfter = %v, %v, want %v, true", after, ok, BusyRetryAfter)
	}
}

func TestPredicates(t *testing.T) {
	t.Parallel()

	if !IsNotFound(sql.ErrNoRows) || !IsNotFound(errors.New(errors.NotFound, "gone")) {
		t.Error("IsNotFound must accept the driver error and the kernel error")
	}
	if IsNotFound(io.EOF) {
		t.Error("IsNotFound must reject a foreign error")
	}
	if !IsConflict(sqlite3.CONSTRAINT_PRIMARYKEY) || !IsConflict(errors.New(errors.Conflict, "taken")) {
		t.Error("IsConflict must accept the driver error and the kernel error")
	}
	if IsConflict(io.EOF) {
		t.Error("IsConflict must reject a foreign error")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapters/sqlite/dbx/ -v`
Expected: FAIL, the package does not build because `Classify`, `Convert`, `IsNotFound`, `IsConflict` and `BusyRetryAfter` are undefined.

- [ ] **Step 3: Write the implementation**

Create `internal/adapters/sqlite/dbx/errors.go`:

```go
package dbx

import (
	"context"
	"database/sql"
	stderrors "errors"
	"time"

	"github.com/ncruces/go-sqlite3"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const BusyRetryAfter = 250 * time.Millisecond

func Classify(err error) errors.Code {
	switch {
	case err == nil:
		return ""
	case stderrors.Is(err, sql.ErrNoRows), stderrors.Is(err, sqlite3.NOTFOUND):
		return errors.NotFound
	case stderrors.Is(err, sqlite3.CONSTRAINT_UNIQUE), stderrors.Is(err, sqlite3.CONSTRAINT_PRIMARYKEY):
		return errors.Conflict
	case stderrors.Is(err, sqlite3.CONSTRAINT_FOREIGNKEY), stderrors.Is(err, sqlite3.CONSTRAINT):
		return errors.Invalid
	case stderrors.Is(err, sqlite3.BUSY), stderrors.Is(err, sqlite3.LOCKED):
		return errors.External
	case stderrors.Is(err, context.Canceled), stderrors.Is(err, sqlite3.INTERRUPT):
		return errors.Cancelled
	default:
		return errors.Internal
	}
}

func Convert(err error, message string) error {
	if err == nil {
		return nil
	}
	if isKernel(err) {
		return err
	}

	code := Classify(err)
	converted := errors.Wrap(err, code, message)
	if code == errors.External {
		return converted.WithRetry(BusyRetryAfter)
	}
	return converted
}

func IsNotFound(err error) bool {
	return errors.IsCode(err, errors.NotFound) || Classify(err) == errors.NotFound
}

func IsConflict(err error) bool {
	return errors.IsCode(err, errors.Conflict) || Classify(err) == errors.Conflict
}

func isKernel(err error) bool {
	var kernel *errors.Error
	return stderrors.As(err, &kernel)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race ./internal/adapters/sqlite/dbx/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/adapters/sqlite/dbx/errors.go internal/adapters/sqlite/dbx/errors_test.go
git commit -m "feat(dbx): map sqlite extended result codes to kernel codes

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 3: The `Result[T]` monad

**Files:**
- Create: `internal/adapters/sqlite/dbx/result.go`
- Test: `internal/adapters/sqlite/dbx/result_test.go`

**Interfaces:**
- Consumes: `IsNotFound`, `IsConflict`, `isKernel` from task 2.
- Produces:
  - `type Result[T any] struct{ value T; err error }`
  - `func Ok[T any](value T) Result[T]`
  - `func Err[T any](err error) Result[T]`
  - `func From[T any](value T, err error) Result[T]`
  - `func (r Result[T]) NotFound(replacement error) Result[T]`
  - `func (r Result[T]) Conflict(replacement error) Result[T]`
  - `func (r Result[T]) WrapErr(wrapper func(error) error) Result[T]`
  - `func (r Result[T]) Unwrap() (T, error)`

- [ ] **Step 1: Write the failing test**

Create `internal/adapters/sqlite/dbx/result_test.go`:

```go
package dbx

import (
	"database/sql"
	stderrors "errors"
	"io"
	"testing"

	"github.com/ncruces/go-sqlite3"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestResultUnwrap(t *testing.T) {
	t.Parallel()

	value, err := Ok("secret").Unwrap()
	if value != "secret" || err != nil {
		t.Fatalf("Ok = %q, %v", value, err)
	}

	value, err = Err[string](io.EOF).Unwrap()
	if value != "" || !stderrors.Is(err, io.EOF) {
		t.Fatalf("Err = %q, %v", value, err)
	}

	value, err = From("row", sql.ErrNoRows).Unwrap()
	if value != "row" || !stderrors.Is(err, sql.ErrNoRows) {
		t.Fatalf("From = %q, %v", value, err)
	}
}

func TestResultReplacements(t *testing.T) {
	t.Parallel()

	missing := errors.New(errors.NotFound, "secret is not stored")
	taken := errors.New(errors.Conflict, "secret already exists")

	cases := []struct {
		name   string
		result Result[int]
		want   errors.Code
	}{
		{
			name:   "not found is replaced",
			result: From(0, sql.ErrNoRows).NotFound(missing),
			want:   errors.NotFound,
		},
		{
			name:   "conflict is replaced",
			result: From(0, sqlite3.CONSTRAINT_UNIQUE).Conflict(taken),
			want:   errors.Conflict,
		},
		{
			name:   "unrelated error survives NotFound",
			result: From(0, sqlite3.BUSY).NotFound(missing).WrapErr(wrap),
			want:   errors.External,
		},
		{
			name:   "unrelated error survives Conflict",
			result: From(0, sqlite3.BUSY).Conflict(taken).WrapErr(wrap),
			want:   errors.External,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := tc.result.Unwrap()
			if errors.CodeOf(err) != tc.want {
				t.Errorf("code = %q, want %q", errors.CodeOf(err), tc.want)
			}
		})
	}
}

func TestResultWrapErr(t *testing.T) {
	t.Parallel()

	_, err := Ok(1).WrapErr(wrap).Unwrap()
	if err != nil {
		t.Fatalf("WrapErr on a value must not produce an error, got %v", err)
	}

	already := errors.New(errors.NeedsHuman, "left alone")
	_, err = From(0, already).WrapErr(wrap).Unwrap()
	if !stderrors.Is(err, already) {
		t.Fatalf("WrapErr must leave a kernel error alone, got %v", err)
	}

	_, err = From(0, io.EOF).WrapErr(wrap).Unwrap()
	if errors.CodeOf(err) != errors.Internal || !stderrors.Is(err, io.EOF) {
		t.Fatalf("WrapErr must convert a foreign error, got %v", err)
	}
}

func wrap(cause error) error {
	return Convert(cause, "read row")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapters/sqlite/dbx/ -run TestResult -v`
Expected: FAIL, the package does not build because `Result`, `Ok`, `Err` and `From` are undefined.

- [ ] **Step 3: Write the implementation**

Create `internal/adapters/sqlite/dbx/result.go`:

```go
package dbx

type Result[T any] struct {
	value T
	err   error
}

func Ok[T any](value T) Result[T] {
	return Result[T]{value: value}
}

func Err[T any](err error) Result[T] {
	return Result[T]{err: err}
}

func From[T any](value T, err error) Result[T] {
	return Result[T]{value: value, err: err}
}

func (r Result[T]) NotFound(replacement error) Result[T] {
	if r.err != nil && IsNotFound(r.err) {
		return Result[T]{err: replacement}
	}
	return r
}

func (r Result[T]) Conflict(replacement error) Result[T] {
	if r.err != nil && IsConflict(r.err) {
		return Result[T]{err: replacement}
	}
	return r
}

func (r Result[T]) WrapErr(wrapper func(error) error) Result[T] {
	if r.err == nil || isKernel(r.err) {
		return r
	}
	return Result[T]{err: wrapper(r.err)}
}

func (r Result[T]) Unwrap() (T, error) {
	return r.value, r.err
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race -cover ./internal/adapters/sqlite/dbx/ -v`
Expected: PASS, coverage at or above 85%.

- [ ] **Step 5: Commit**

```bash
git add internal/adapters/sqlite/dbx/result.go internal/adapters/sqlite/dbx/result_test.go
git commit -m "feat(dbx): result monad at the repository boundary

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 4: The data source name

**Files:**
- Create: `internal/adapters/sqlite/dsn.go`
- Test: `internal/adapters/sqlite/dsn_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `func dsn(path string, key []byte, readOnly bool) string` in package `sqlite`, and `const keyLength = 32`.

- [ ] **Step 1: Write the failing test**

Create `internal/adapters/sqlite/dsn_test.go`:

```go
package sqlite

import (
	"net/url"
	"slices"
	"strings"
	"testing"
)

func TestDSN(t *testing.T) {
	t.Parallel()

	key := make([]byte, keyLength)
	for i := range key {
		key[i] = byte(i)
	}

	cases := []struct {
		name        string
		path        string
		key         []byte
		readOnly    bool
		wantOpaque  string
		wantParams  map[string]string
		wantPragmas []string
		wantAbsent  []string
	}{
		{
			name:       "plain writer",
			path:       `C:\data\postulator.db`,
			key:        nil,
			readOnly:   false,
			wantOpaque: "C:/data/postulator.db",
			wantParams: map[string]string{"_txlock": "immediate"},
			wantPragmas: []string{
				"busy_timeout(5000)",
				"foreign_keys(1)",
				"journal_mode(WAL)",
				"synchronous(NORMAL)",
			},
			wantAbsent: []string{"vfs", "hexkey", "mode"},
		},
		{
			name:       "encrypted writer",
			path:       `C:\data\postulator.db`,
			key:        key,
			readOnly:   false,
			wantOpaque: "C:/data/postulator.db",
			wantParams: map[string]string{
				"_txlock": "immediate",
				"vfs":     "adiantum",
				"hexkey":  "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f",
			},
			wantPragmas: []string{
				"busy_timeout(5000)",
				"foreign_keys(1)",
				"journal_mode(WAL)",
				"synchronous(NORMAL)",
			},
			wantAbsent: []string{"mode"},
		},
		{
			name:        "encrypted reader",
			path:        `C:\Program Files\postulator.db`,
			key:         key,
			readOnly:    true,
			wantOpaque:  "C:/Program%20Files/postulator.db",
			wantParams:  map[string]string{"mode": "ro", "vfs": "adiantum"},
			wantPragmas: []string{"busy_timeout(5000)", "foreign_keys(1)"},
			wantAbsent:  []string{"_txlock"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			raw := dsn(tc.path, tc.key, tc.readOnly)
			if !strings.HasPrefix(raw, "file:") {
				t.Fatalf("dsn = %q, want a file: URI", raw)
			}

			parsed, err := url.Parse(raw)
			if err != nil {
				t.Fatalf("parse %q: %v", raw, err)
			}
			if parsed.Opaque != tc.wantOpaque {
				t.Errorf("opaque = %q, want %q", parsed.Opaque, tc.wantOpaque)
			}

			query := parsed.Query()
			for name, want := range tc.wantParams {
				if query.Get(name) != want {
					t.Errorf("%s = %q, want %q", name, query.Get(name), want)
				}
			}
			for _, name := range tc.wantAbsent {
				if query.Has(name) {
					t.Errorf("%s must be absent, got %q", name, query.Get(name))
				}
			}
			if !slices.Equal(query["_pragma"], tc.wantPragmas) {
				t.Errorf("_pragma = %v, want %v", query["_pragma"], tc.wantPragmas)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapters/sqlite/ -run TestDSN -v`
Expected: FAIL, the package does not build because `dsn` and `keyLength` are undefined.

- [ ] **Step 3: Write the implementation**

Create `internal/adapters/sqlite/dsn.go`:

```go
package sqlite

import (
	"encoding/hex"
	"net/url"
	"path/filepath"
	"strconv"
)

const (
	keyLength         = 32
	busyTimeoutMillis = 5000
)

func dsn(path string, key []byte, readOnly bool) string {
	query := make(url.Values)
	if key != nil {
		query.Set("vfs", "adiantum")
		query.Set("hexkey", hex.EncodeToString(key))
	}
	if readOnly {
		query.Set("mode", "ro")
	} else {
		query.Set("_txlock", "immediate")
	}

	query.Add("_pragma", "busy_timeout("+strconv.Itoa(busyTimeoutMillis)+")")
	query.Add("_pragma", "foreign_keys(1)")
	if !readOnly {
		query.Add("_pragma", "journal_mode(WAL)")
		query.Add("_pragma", "synchronous(NORMAL)")
	}

	location := url.URL{Path: filepath.ToSlash(path)}
	return "file:" + location.EscapedPath() + "?" + query.Encode()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race ./internal/adapters/sqlite/ -run TestDSN -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/adapters/sqlite/dsn.go internal/adapters/sqlite/dsn_test.go
git commit -m "feat(sqlite): build the file uri with pragmas and the adiantum key

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 5: Opening, migrating and closing the store

**Files:**
- Create: `internal/adapters/sqlite/store.go`
- Test: `internal/adapters/sqlite/store_test.go`
- Modify: `internal/adapters/sqlite/migrations_test.go`

**Interfaces:**
- Consumes: `dsn` and `keyLength` (task 4), `migrations()` (task 1), `dbx.Convert` (task 2).
- Produces:
  - `type Store struct{ writer *sql.DB; reader *sql.DB; path string }`
  - `func Open(path string, key []byte) (*Store, error)`
  - `func (s *Store) Close() error`
  - `func (s *Store) Path() string`
  - `func (s *Store) provider() (*goose.Provider, error)`
  - `func (s *Store) migrate(ctx context.Context) error`

- [ ] **Step 1: Write the failing test**

Create `internal/adapters/sqlite/store_test.go`:

```go
package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func openStore(t *testing.T, key []byte) *Store {
	t.Helper()

	store, err := Open(filepath.Join(t.TempDir(), "postulator.db"), key)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Errorf("close: %v", closeErr)
		}
	})
	return store
}

func testKey() []byte {
	key := make([]byte, keyLength)
	for i := range key {
		key[i] = byte(i + 1)
	}
	return key
}

func TestOpenAppliesPragmas(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		key  []byte
	}{
		{name: "plain", key: nil},
		{name: "adiantum", key: testKey()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := openStore(t, tc.key)
			pragmas := []struct {
				name string
				want string
			}{
				{name: "journal_mode", want: "wal"},
				{name: "foreign_keys", want: "1"},
				{name: "busy_timeout", want: "5000"},
				{name: "synchronous", want: "1"},
			}

			for _, pragma := range pragmas {
				var got string
				if err := store.writer.QueryRowContext(t.Context(), "PRAGMA "+pragma.name).Scan(&got); err != nil {
					t.Fatalf("PRAGMA %s: %v", pragma.name, err)
				}
				if got != pragma.want {
					t.Errorf("PRAGMA %s = %q, want %q", pragma.name, got, pragma.want)
				}
			}
		})
	}
}

func TestOpenCreatesTheSchema(t *testing.T) {
	t.Parallel()

	store := openStore(t, testKey())
	for _, table := range []string{"app_meta", "settings", "secrets"} {
		var name string
		err := store.reader.QueryRowContext(t.Context(),
			`SELECT name FROM sqlite_schema WHERE type = 'table' AND name = ?`, table).Scan(&name)
		if err != nil {
			t.Fatalf("table %s: %v", table, err)
		}
	}

	var version string
	err := store.reader.QueryRowContext(t.Context(),
		`SELECT value FROM app_meta WHERE key = 'schema_version'`).Scan(&version)
	if err != nil {
		t.Fatalf("schema_version: %v", err)
	}
	if version != "1" {
		t.Errorf("schema_version = %q, want %q", version, "1")
	}
}

func TestOpenRejectsBadArguments(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		path string
		key  []byte
	}{
		{name: "empty path", path: "", key: nil},
		{name: "short key", path: filepath.Join(t.TempDir(), "a.db"), key: []byte("too short")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store, err := Open(tc.path, tc.key)
			if store != nil {
				t.Fatal("no store may be returned")
			}
			if !errors.IsCode(err, errors.Invalid) {
				t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
			}
		})
	}
}

func TestOpenRejectsTheWrongKey(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "postulator.db")
	first, err := Open(path, testKey())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err = first.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	other := testKey()
	other[0] ^= 0xff

	second, err := Open(path, other)
	if err == nil {
		if closeErr := second.Close(); closeErr != nil {
			t.Errorf("close: %v", closeErr)
		}
		t.Fatal("a database opened with the wrong key must fail")
	}
	if errors.CodeOf(err) == "" {
		t.Errorf("the failure must carry a kernel code, got %v", err)
	}
}

func TestReaderPoolRefusesWrites(t *testing.T) {
	t.Parallel()

	store := openStore(t, nil)
	_, err := store.reader.ExecContext(t.Context(),
		`INSERT INTO settings (key, value, updated_at) VALUES ('a', '1', '2026-09-17T00:00:00Z')`)
	if err == nil {
		t.Fatal("the reader pool must refuse a write")
	}
}

func TestPath(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "postulator.db")
	store, err := Open(path, nil)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Errorf("close: %v", closeErr)
		}
	})

	if store.Path() != path {
		t.Errorf("Path = %q, want %q", store.Path(), path)
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	t.Parallel()

	store := openStore(t, nil)
	if err := store.migrate(context.Background()); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
}
```

Append to `internal/adapters/sqlite/migrations_test.go`:

```go
func TestMigrationsRoundTrip(t *testing.T) {
	t.Parallel()

	store := openStore(t, nil)

	provider, err := store.provider()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}

	version, err := provider.GetDBVersion(t.Context())
	if err != nil {
		t.Fatalf("version after up: %v", err)
	}
	if version != 3 {
		t.Fatalf("version after up = %d, want 3", version)
	}

	if _, err = provider.DownTo(t.Context(), 0); err != nil {
		t.Fatalf("down: %v", err)
	}

	for _, table := range []string{"app_meta", "settings", "secrets"} {
		var name string
		scanErr := store.writer.QueryRowContext(t.Context(),
			`SELECT name FROM sqlite_schema WHERE type = 'table' AND name = ?`, table).Scan(&name)
		if scanErr == nil {
			t.Errorf("%s survived the down migration", table)
		}
	}

	if _, err = provider.Up(t.Context()); err != nil {
		t.Fatalf("up again: %v", err)
	}

	version, err = provider.GetDBVersion(t.Context())
	if err != nil {
		t.Fatalf("version after the second up: %v", err)
	}
	if version != 3 {
		t.Errorf("version after the second up = %d, want 3", version)
	}
}
```

The import block of `migrations_test.go` becomes `"io/fs"`, `"slices"`, `"strings"`, `"testing"`.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapters/sqlite/ -v`
Expected: FAIL, the package does not build because `Store`, `Open`, `Close`, `Path`, `provider` and `migrate` are undefined.

- [ ] **Step 3: Write the implementation**

Create `internal/adapters/sqlite/store.go`:

```go
package sqlite

import (
	"context"
	"database/sql"
	stderrors "errors"
	"os"
	"path/filepath"

	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/vfs/adiantum"
	"github.com/pressly/goose/v3"

	"github.com/davidmovas/postulator/internal/adapters/sqlite/dbx"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	readerConns   = 4
	directoryMode = 0o700
)

type Store struct {
	writer *sql.DB
	reader *sql.DB
	path   string
}

func Open(path string, key []byte) (*Store, error) {
	if path == "" {
		return nil, errors.New(errors.Invalid, "database path must not be empty")
	}
	if key != nil && len(key) != keyLength {
		return nil, errors.New(errors.Invalid, "database key must be 32 bytes")
	}
	if err := os.MkdirAll(filepath.Dir(path), directoryMode); err != nil {
		return nil, errors.Wrap(err, errors.Internal, "create the database directory")
	}

	writer, err := driver.Open(dsn(path, key, false))
	if err != nil {
		return nil, dbx.Convert(err, "open the database for writing")
	}
	writer.SetMaxOpenConns(1)
	writer.SetMaxIdleConns(1)
	writer.SetConnMaxLifetime(0)

	store := &Store{writer: writer, path: path}
	if err = store.migrate(context.Background()); err != nil {
		return nil, stderrors.Join(err, closeDB(writer, "close the writer after a failed migration"))
	}

	reader, err := driver.Open(dsn(path, key, true))
	if err != nil {
		converted := dbx.Convert(err, "open the database for reading")
		return nil, stderrors.Join(converted, closeDB(writer, "close the writer after a failed reader open"))
	}
	reader.SetMaxOpenConns(readerConns)
	reader.SetMaxIdleConns(readerConns)
	reader.SetConnMaxLifetime(0)

	store.reader = reader
	return store, nil
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) Close() error {
	return stderrors.Join(
		closeDB(s.reader, "close the database reader"),
		closeDB(s.writer, "close the database writer"),
	)
}

func (s *Store) provider() (*goose.Provider, error) {
	fsys, err := migrations()
	if err != nil {
		return nil, err
	}

	provider, err := goose.NewProvider(goose.DialectSQLite3, s.writer, fsys)
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "prepare the migrations")
	}
	return provider, nil
}

func (s *Store) migrate(ctx context.Context) error {
	provider, err := s.provider()
	if err != nil {
		return err
	}
	if _, err = provider.Up(ctx); err != nil {
		return errors.Wrap(err, errors.Internal, "apply the migrations")
	}
	return nil
}

func closeDB(db *sql.DB, message string) error {
	if db == nil {
		return nil
	}
	return dbx.Convert(db.Close(), message)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race ./internal/adapters/sqlite/ -v`
Expected: PASS for `TestOpenAppliesPragmas`, `TestOpenCreatesTheSchema`, `TestOpenRejectsBadArguments`, `TestOpenRejectsTheWrongKey`, `TestReaderPoolRefusesWrites`, `TestPath`, `TestMigrateIsIdempotent`, `TestMigrationsAreEmbedded`, `TestMigrationsRoundTrip`, `TestDSN`.

- [ ] **Step 5: Commit**

```bash
git add internal/adapters/sqlite/store.go internal/adapters/sqlite/store_test.go internal/adapters/sqlite/migrations_test.go
git commit -m "feat(sqlite): encrypted store with wal pragmas and embedded migrations

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 6: The unit of work

**Files:**
- Create: `internal/adapters/sqlite/tx.go`
- Test: `internal/adapters/sqlite/tx_test.go`

**Interfaces:**
- Consumes: `*Store` and its two pools (task 5), `dbx.Convert` (task 2).
- Produces:
  - `type executor interface { ExecContext(context.Context, string, ...any) (sql.Result, error); QueryContext(context.Context, string, ...any) (*sql.Rows, error); QueryRowContext(context.Context, string, ...any) *sql.Row }`
  - `func (s *Store) Do(ctx context.Context, fn func(context.Context) error) error`
  - `func (s *Store) execFrom(ctx context.Context) executor`
  - `func (s *Store) writeFrom(ctx context.Context) executor`
  - `func txFrom(ctx context.Context) (*sql.Tx, bool)`

- [ ] **Step 1: Write the failing test**

Create `internal/adapters/sqlite/tx_test.go`:

```go
package sqlite

import (
	"context"
	"database/sql"
	stderrors "errors"
	"testing"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const insertSetting = `INSERT INTO settings (key, value, updated_at) VALUES (?, ?, '2026-09-17T00:00:00Z')`

func countSettings(t *testing.T, store *Store) int {
	t.Helper()

	var total int
	if err := store.reader.QueryRowContext(t.Context(), `SELECT count(*) FROM settings`).Scan(&total); err != nil {
		t.Fatalf("count: %v", err)
	}
	return total
}

func TestDoCommits(t *testing.T) {
	t.Parallel()

	store := openStore(t, nil)
	err := store.Do(t.Context(), func(c context.Context) error {
		_, execErr := store.writeFrom(c).ExecContext(c, insertSetting, "runs.workers", "2")
		return execErr
	})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if got := countSettings(t, store); got != 1 {
		t.Errorf("rows = %d, want 1", got)
	}
}

func TestDoRollsBack(t *testing.T) {
	t.Parallel()

	store := openStore(t, nil)
	sentinel := errors.New(errors.Invalid, "abandon")

	err := store.Do(t.Context(), func(c context.Context) error {
		if _, execErr := store.writeFrom(c).ExecContext(c, insertSetting, "runs.workers", "2"); execErr != nil {
			return execErr
		}
		return sentinel
	})
	if !stderrors.Is(err, sentinel) {
		t.Fatalf("Do = %v, want the callback error", err)
	}
	if got := countSettings(t, store); got != 0 {
		t.Errorf("rows = %d, want 0", got)
	}
}

func TestDoNests(t *testing.T) {
	t.Parallel()

	store := openStore(t, nil)
	var outer, inner *sql.Tx

	err := store.Do(t.Context(), func(c context.Context) error {
		outer, _ = txFrom(c)
		return store.Do(c, func(nested context.Context) error {
			inner, _ = txFrom(nested)
			_, execErr := store.writeFrom(nested).ExecContext(nested, insertSetting, "runs.workers", "2")
			return execErr
		})
	})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if outer == nil || outer != inner {
		t.Errorf("the nested call must reuse the outer transaction")
	}
	if got := countSettings(t, store); got != 1 {
		t.Errorf("rows = %d, want 1", got)
	}
}

func TestNestedFailureRollsBackEverything(t *testing.T) {
	t.Parallel()

	store := openStore(t, nil)
	sentinel := errors.New(errors.Invalid, "abandon")

	err := store.Do(t.Context(), func(c context.Context) error {
		if _, execErr := store.writeFrom(c).ExecContext(c, insertSetting, "a", "1"); execErr != nil {
			return execErr
		}
		return store.Do(c, func(nested context.Context) error {
			if _, execErr := store.writeFrom(nested).ExecContext(nested, insertSetting, "b", "2"); execErr != nil {
				return execErr
			}
			return sentinel
		})
	})
	if !stderrors.Is(err, sentinel) {
		t.Fatalf("Do = %v, want the callback error", err)
	}
	if got := countSettings(t, store); got != 0 {
		t.Errorf("rows = %d, want 0", got)
	}
}

func TestExecutorRouting(t *testing.T) {
	t.Parallel()

	store := openStore(t, nil)

	if store.execFrom(t.Context()) != executor(store.reader) {
		t.Error("execFrom outside a transaction must return the reader")
	}
	if store.writeFrom(t.Context()) != executor(store.writer) {
		t.Error("writeFrom outside a transaction must return the writer")
	}

	err := store.Do(t.Context(), func(c context.Context) error {
		tx, ok := txFrom(c)
		if !ok {
			return errors.New(errors.Internal, "no transaction in the context")
		}
		if store.execFrom(c) != executor(tx) || store.writeFrom(c) != executor(tx) {
			return errors.New(errors.Internal, "both accessors must return the transaction")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
}

func TestDoRejectsAClosedStore(t *testing.T) {
	t.Parallel()

	store, err := Open(t.TempDir()+`\postulator.db`, nil)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err = store.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	err = store.Do(t.Context(), func(context.Context) error { return nil })
	if !errors.IsCode(err, errors.Internal) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Internal)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapters/sqlite/ -run 'TestDo|TestNested|TestExecutorRouting' -v`
Expected: FAIL, the package does not build because `Do`, `execFrom`, `writeFrom`, `txFrom` and `executor` are undefined.

- [ ] **Step 3: Write the implementation**

Create `internal/adapters/sqlite/tx.go`:

```go
package sqlite

import (
	"context"
	"database/sql"
	stderrors "errors"

	"github.com/davidmovas/postulator/internal/adapters/sqlite/dbx"
)

type executor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type txKey struct{}

func withTx(parent context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(parent, txKey{}, tx)
}

func txFrom(ctx context.Context) (*sql.Tx, bool) {
	tx, ok := ctx.Value(txKey{}).(*sql.Tx)
	return tx, ok
}

func (s *Store) execFrom(ctx context.Context) executor {
	if tx, ok := txFrom(ctx); ok {
		return tx
	}
	return s.reader
}

func (s *Store) writeFrom(ctx context.Context) executor {
	if tx, ok := txFrom(ctx); ok {
		return tx
	}
	return s.writer
}

func (s *Store) Do(ctx context.Context, fn func(context.Context) error) error {
	if _, ok := txFrom(ctx); ok {
		return fn(ctx)
	}

	tx, err := s.writer.BeginTx(ctx, nil)
	if err != nil {
		return dbx.Convert(err, "begin the transaction")
	}

	if err = fn(withTx(ctx, tx)); err != nil {
		rollback := tx.Rollback()
		if rollback != nil && !stderrors.Is(rollback, sql.ErrTxDone) {
			return stderrors.Join(err, dbx.Convert(rollback, "roll back the transaction"))
		}
		return err
	}

	if err = tx.Commit(); err != nil {
		return dbx.Convert(err, "commit the transaction")
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race ./internal/adapters/sqlite/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/adapters/sqlite/tx.go internal/adapters/sqlite/tx_test.go
git commit -m "feat(sqlite): unit of work over a single writer connection

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 7: The test helper

**Files:**
- Create: `internal/adapters/sqlite/sqlitetest/sqlitetest.go`
- Test: `internal/adapters/sqlite/sqlitetest/sqlitetest_test.go`

**Interfaces:**
- Consumes: `sqlite.Open` and `(*sqlite.Store).Close` (task 5).
- Produces:
  - `func Open(t testing.TB) *sqlite.Store`
  - `func OpenEncrypted(t testing.TB, key []byte) *sqlite.Store`
  - `func Key() []byte`

- [ ] **Step 1: Write the failing test**

Create `internal/adapters/sqlite/sqlitetest/sqlitetest_test.go`:

```go
package sqlitetest_test

import (
	"path/filepath"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
)

func TestOpen(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	if filepath.Base(store.Path()) != "postulator.db" {
		t.Errorf("path = %q, want a file named postulator.db", store.Path())
	}
	if err := store.Do(t.Context(), func(c context.Context) error { return nil }); err != nil {
		t.Fatalf("Do: %v", err)
	}
}

func TestOpenEncrypted(t *testing.T) {
	t.Parallel()

	key := sqlitetest.Key()
	if len(key) != 32 {
		t.Fatalf("Key length = %d, want 32", len(key))
	}

	store := sqlitetest.OpenEncrypted(t, key)
	if err := store.Do(t.Context(), func(c context.Context) error { return nil }); err != nil {
		t.Fatalf("Do: %v", err)
	}
}
```

The import block also needs `"context"`.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapters/sqlite/sqlitetest/ -v`
Expected: FAIL, the package `sqlitetest` does not exist.

- [ ] **Step 3: Write the implementation**

Create `internal/adapters/sqlite/sqlitetest/sqlitetest.go`:

```go
package sqlitetest

import (
	"path/filepath"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
)

const (
	fileName  = "postulator.db"
	keyLength = 32
)

func Key() []byte {
	key := make([]byte, keyLength)
	for i := range key {
		key[i] = byte(i + 1)
	}
	return key
}

func Open(t testing.TB) *sqlite.Store {
	t.Helper()
	return open(t, nil)
}

func OpenEncrypted(t testing.TB, key []byte) *sqlite.Store {
	t.Helper()
	return open(t, key)
}

func open(t testing.TB, key []byte) *sqlite.Store {
	t.Helper()

	store, err := sqlite.Open(filepath.Join(t.TempDir(), fileName), key)
	if err != nil {
		t.Fatalf("open the store: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Errorf("close the store: %v", closeErr)
		}
	})
	return store
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race ./internal/adapters/sqlite/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/adapters/sqlite/sqlitetest/
git commit -m "test(sqlite): temp-file store helper with a plain and an encrypted variant

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 8: The settings repository

**Files:**
- Create: `internal/adapters/sqlite/settings_repo.go`
- Test: `internal/adapters/sqlite/settings_repo_test.go`

**Interfaces:**
- Consumes: `(*Store).execFrom`, `(*Store).writeFrom`, `(*Store).Do` (task 6), `dbx.From`, `dbx.Convert` (tasks 2 and 3), `clock.Clock` from `internal/kernel/clock`, `sqlitetest.Open` (task 7).
- Produces:
  - `func NewSettingsRepo(store *Store, clk clock.Clock) *SettingsRepo`
  - `func (r *SettingsRepo) Get(ctx context.Context, key string) (json.RawMessage, bool, error)`
  - `func (r *SettingsRepo) Set(ctx context.Context, key string, value json.RawMessage) error`
  - `func (r *SettingsRepo) All(ctx context.Context) (map[string]json.RawMessage, error)`

- [ ] **Step 1: Write the failing test**

Create `internal/adapters/sqlite/settings_repo_test.go`:

```go
package sqlite_test

import (
	"context"
	"encoding/json"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func settingsRepo(t *testing.T) (*sqlite.SettingsRepo, *sqlite.Store) {
	t.Helper()

	store := sqlitetest.Open(t)
	return sqlite.NewSettingsRepo(store, clock.NewFake(time.Date(2026, time.September, 17, 8, 30, 0, 0, time.UTC))), store
}

func TestSettingsRepoRoundTrip(t *testing.T) {
	t.Parallel()

	repo, _ := settingsRepo(t)

	value, found, err := repo.Get(t.Context(), "runs.workers")
	if err != nil {
		t.Fatalf("Get on an empty table: %v", err)
	}
	if found || value != nil {
		t.Fatalf("Get = %s, %v, want nil, false", value, found)
	}

	if err = repo.Set(t.Context(), "runs.workers", json.RawMessage(`2`)); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err = repo.Set(t.Context(), "runs.workers", json.RawMessage(`4`)); err != nil {
		t.Fatalf("Set again: %v", err)
	}

	value, found, err = repo.Get(t.Context(), "runs.workers")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found || string(value) != "4" {
		t.Errorf("Get = %s, %v, want 4, true", value, found)
	}
}

func TestSettingsRepoAll(t *testing.T) {
	t.Parallel()

	repo, _ := settingsRepo(t)

	stored, err := repo.All(t.Context())
	if err != nil {
		t.Fatalf("All on an empty table: %v", err)
	}
	if len(stored) != 0 {
		t.Fatalf("All = %v, want an empty map", stored)
	}

	cases := map[string]string{
		"runs.workers":  `2`,
		"llm.provider":  `"openai"`,
		"agent.useFake": `true`,
	}
	for key, raw := range cases {
		if err = repo.Set(t.Context(), key, json.RawMessage(raw)); err != nil {
			t.Fatalf("Set %s: %v", key, err)
		}
	}

	stored, err = repo.All(t.Context())
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if !slices.Equal(slices.Sorted(maps.Keys(stored)), []string{"agent.useFake", "llm.provider", "runs.workers"}) {
		t.Fatalf("keys = %v", slices.Sorted(maps.Keys(stored)))
	}
	for key, raw := range cases {
		if string(stored[key]) != raw {
			t.Errorf("%s = %s, want %s", key, stored[key], raw)
		}
	}
}

func TestSettingsRepoRejectsAnEmptyKey(t *testing.T) {
	t.Parallel()

	repo, _ := settingsRepo(t)
	if err := repo.Set(t.Context(), "", json.RawMessage(`1`)); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
	}
}

func TestSettingsRepoWritesInsideTheUnitOfWork(t *testing.T) {
	t.Parallel()

	repo, store := settingsRepo(t)
	sentinel := errors.New(errors.Invalid, "abandon")

	err := store.Do(t.Context(), func(c context.Context) error {
		if setErr := repo.Set(c, "runs.workers", json.RawMessage(`2`)); setErr != nil {
			return setErr
		}
		return sentinel
	})
	if err == nil {
		t.Fatal("Do must return the callback error")
	}

	_, found, err := repo.Get(t.Context(), "runs.workers")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if found {
		t.Error("a rolled back write must not be visible")
	}
}

func TestSettingsRepoStoresTheClockInstant(t *testing.T) {
	t.Parallel()

	repo, store := settingsRepo(t)
	if err := repo.Set(t.Context(), "runs.workers", json.RawMessage(`2`)); err != nil {
		t.Fatalf("Set: %v", err)
	}

	var updated string
	err := store.Do(t.Context(), func(c context.Context) error {
		return sqlite.ScanUpdatedAt(c, store, "runs.workers", &updated)
	})
	if err != nil {
		t.Fatalf("read updated_at: %v", err)
	}
	if updated != "2026-09-17T08:30:00Z" {
		t.Errorf("updated_at = %q, want %q", updated, "2026-09-17T08:30:00Z")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapters/sqlite/ -run TestSettingsRepo -v`
Expected: FAIL, the package does not build because `SettingsRepo`, `NewSettingsRepo` and `ScanUpdatedAt` are undefined.

- [ ] **Step 3: Write the implementation**

Create `internal/adapters/sqlite/settings_repo.go`:

```go
package sqlite

import (
	"context"
	"encoding/json"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite/dbx"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	selectSetting = `SELECT value FROM settings WHERE key = ?`
	selectSettings = `SELECT key, value FROM settings`
	selectSettingUpdatedAt = `SELECT updated_at FROM settings WHERE key = ?`
	upsertSetting = `INSERT INTO settings (key, value, updated_at) VALUES (?, ?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`
)

type SettingsRepo struct {
	store *Store
	clock clock.Clock
}

func NewSettingsRepo(store *Store, clk clock.Clock) *SettingsRepo {
	return &SettingsRepo{store: store, clock: clk}
}

func (r *SettingsRepo) Get(ctx context.Context, key string) (json.RawMessage, bool, error) {
	var value string
	err := r.store.execFrom(ctx).QueryRowContext(ctx, selectSetting, key).Scan(&value)
	if dbx.IsNotFound(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, dbx.Convert(err, "read the setting "+key)
	}
	return json.RawMessage(value), true, nil
}

func (r *SettingsRepo) Set(ctx context.Context, key string, value json.RawMessage) error {
	if key == "" {
		return errors.New(errors.Invalid, "setting key must not be empty")
	}
	if !json.Valid(value) {
		return errors.New(errors.Invalid, "setting "+key+" must hold valid JSON")
	}

	updated := r.clock.Now().UTC().Format(time.RFC3339)
	_, err := r.store.writeFrom(ctx).ExecContext(ctx, upsertSetting, key, string(value), updated)
	return dbx.Convert(err, "write the setting "+key)
}

func (r *SettingsRepo) All(ctx context.Context) (stored map[string]json.RawMessage, err error) {
	rows, err := r.store.execFrom(ctx).QueryContext(ctx, selectSettings)
	if err != nil {
		return nil, dbx.Convert(err, "list the settings")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = dbx.Convert(closeErr, "close the settings rows")
		}
	}()

	stored = make(map[string]json.RawMessage)
	for rows.Next() {
		var key, value string
		if err = rows.Scan(&key, &value); err != nil {
			return nil, dbx.Convert(err, "scan a settings row")
		}
		stored[key] = json.RawMessage(value)
	}
	if err = rows.Err(); err != nil {
		return nil, dbx.Convert(err, "read the settings rows")
	}
	return stored, nil
}

func ScanUpdatedAt(ctx context.Context, store *Store, key string, into *string) error {
	err := store.execFrom(ctx).QueryRowContext(ctx, selectSettingUpdatedAt, key).Scan(into)
	return dbx.Convert(err, "read the setting timestamp for "+key)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race ./internal/adapters/sqlite/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/adapters/sqlite/settings_repo.go internal/adapters/sqlite/settings_repo_test.go
git commit -m "feat(sqlite): settings repository over the settings table

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 9: The secrets repository

**Files:**
- Create: `internal/adapters/sqlite/secrets_repo.go`
- Test: `internal/adapters/sqlite/secrets_repo_test.go`

**Interfaces:**
- Consumes: `(*Store).execFrom`, `(*Store).writeFrom` (task 6), `dbx.From`, `dbx.Convert` (tasks 2 and 3), `clock.Clock`, `sqlitetest.Open` (task 7).
- Produces:
  - `func NewSecretsRepo(store *Store, clk clock.Clock) *SecretsRepo`
  - `func (r *SecretsRepo) Put(ctx context.Context, ref string, ciphertext []byte) error`
  - `func (r *SecretsRepo) Get(ctx context.Context, ref string) ([]byte, error)`
  - `func (r *SecretsRepo) Delete(ctx context.Context, ref string) error`

- [ ] **Step 1: Write the failing test**

Create `internal/adapters/sqlite/secrets_repo_test.go`:

```go
package sqlite_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func secretsRepo(t *testing.T) *sqlite.SecretsRepo {
	t.Helper()

	store := sqlitetest.Open(t)
	return sqlite.NewSecretsRepo(store, clock.NewFake(time.Date(2026, time.September, 17, 8, 30, 0, 0, time.UTC)))
}

func TestSecretsRepoRoundTrip(t *testing.T) {
	t.Parallel()

	repo := secretsRepo(t)
	sealed := []byte{0x76, 0x31, 0x3a, 0x01, 0x02, 0x03}

	if err := repo.Put(t.Context(), "wp.site.token", sealed); err != nil {
		t.Fatalf("Put: %v", err)
	}

	got, err := repo.Get(t.Context(), "wp.site.token")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !bytes.Equal(got, sealed) {
		t.Errorf("Get = %x, want %x", got, sealed)
	}

	replacement := []byte{0x76, 0x31, 0x3a, 0x09}
	if err = repo.Put(t.Context(), "wp.site.token", replacement); err != nil {
		t.Fatalf("Put again: %v", err)
	}
	got, err = repo.Get(t.Context(), "wp.site.token")
	if err != nil {
		t.Fatalf("Get after replace: %v", err)
	}
	if !bytes.Equal(got, replacement) {
		t.Errorf("Get = %x, want %x", got, replacement)
	}

	if err = repo.Delete(t.Context(), "wp.site.token"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err = repo.Get(t.Context(), "wp.site.token"); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.NotFound)
	}
}

func TestSecretsRepoRejectsBadInput(t *testing.T) {
	t.Parallel()

	repo := secretsRepo(t)

	cases := []struct {
		name string
		call func() error
	}{
		{name: "empty ref", call: func() error { return repo.Put(t.Context(), "", []byte{1}) }},
		{name: "empty ciphertext", call: func() error { return repo.Put(t.Context(), "a", nil) }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if err := tc.call(); !errors.IsCode(err, errors.Invalid) {
				t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
			}
		})
	}
}

func TestSecretsRepoDeleteReportsMissing(t *testing.T) {
	t.Parallel()

	repo := secretsRepo(t)
	if err := repo.Delete(t.Context(), "absent"); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.NotFound)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapters/sqlite/ -run TestSecretsRepo -v`
Expected: FAIL, the package does not build because `SecretsRepo` and `NewSecretsRepo` are undefined.

- [ ] **Step 3: Write the implementation**

Create `internal/adapters/sqlite/secrets_repo.go`:

```go
package sqlite

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite/dbx"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	selectSecret = `SELECT ciphertext FROM secrets WHERE ref = ?`
	deleteSecret = `DELETE FROM secrets WHERE ref = ?`
	upsertSecret = `INSERT INTO secrets (ref, ciphertext, created_at, updated_at) VALUES (?, ?, ?, ?)
ON CONFLICT(ref) DO UPDATE SET ciphertext = excluded.ciphertext, updated_at = excluded.updated_at`
)

type SecretsRepo struct {
	store *Store
	clock clock.Clock
}

func NewSecretsRepo(store *Store, clk clock.Clock) *SecretsRepo {
	return &SecretsRepo{store: store, clock: clk}
}

func (r *SecretsRepo) Put(ctx context.Context, ref string, ciphertext []byte) error {
	if ref == "" {
		return errors.New(errors.Invalid, "secret reference must not be empty")
	}
	if len(ciphertext) == 0 {
		return errors.New(errors.Invalid, "secret ciphertext must not be empty")
	}

	now := r.clock.Now().UTC().Format(time.RFC3339)
	_, err := r.store.writeFrom(ctx).ExecContext(ctx, upsertSecret, ref, ciphertext, now, now)
	return dbx.Convert(err, "write the secret "+ref)
}

func (r *SecretsRepo) Get(ctx context.Context, ref string) ([]byte, error) {
	var ciphertext []byte
	err := r.store.execFrom(ctx).QueryRowContext(ctx, selectSecret, ref).Scan(&ciphertext)

	return dbx.From(ciphertext, err).
		NotFound(errors.New(errors.NotFound, "secret "+ref+" is not stored")).
		WrapErr(func(cause error) error { return dbx.Convert(cause, "read the secret "+ref) }).
		Unwrap()
}

func (r *SecretsRepo) Delete(ctx context.Context, ref string) error {
	result, err := r.store.writeFrom(ctx).ExecContext(ctx, deleteSecret, ref)
	if err != nil {
		return dbx.Convert(err, "delete the secret "+ref)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return dbx.Convert(err, "count the deleted secrets")
	}
	if affected == 0 {
		return errors.New(errors.NotFound, "secret "+ref+" is not stored")
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race -cover ./internal/adapters/sqlite/... -v`
Expected: PASS, coverage at or above 85% for `sqlite` and `dbx`.

- [ ] **Step 5: Commit**

```bash
git add internal/adapters/sqlite/secrets_repo.go internal/adapters/sqlite/secrets_repo_test.go
git commit -m "feat(sqlite): secrets repository over the secrets table

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 10: DPAPI protection

**Files:**
- Create: `internal/adapters/secrets/dpapi/dpapi.go`
- Test: `internal/adapters/secrets/dpapi/dpapi_test.go`

**Interfaces:**
- Consumes: `golang.org/x/sys/windows`, `internal/kernel/errors`.
- Produces:
  - `func Protect(plaintext []byte) ([]byte, error)`
  - `func Unprotect(protected []byte) ([]byte, error)`

- [ ] **Step 1: Write the failing test**

Create `internal/adapters/secrets/dpapi/dpapi_test.go`:

```go
package dpapi_test

import (
	"bytes"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/secrets/dpapi"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestRoundTrip(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		plaintext []byte
	}{
		{name: "one byte", plaintext: []byte{0x00}},
		{name: "master key", plaintext: bytes.Repeat([]byte{0xab}, 32)},
		{name: "text", plaintext: []byte("correct horse battery staple")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			protected, err := dpapi.Protect(tc.plaintext)
			if err != nil {
				t.Fatalf("Protect: %v", err)
			}
			if bytes.Contains(protected, tc.plaintext) {
				t.Fatal("the protected blob must not contain the plaintext")
			}

			recovered, err := dpapi.Unprotect(protected)
			if err != nil {
				t.Fatalf("Unprotect: %v", err)
			}
			if !bytes.Equal(recovered, tc.plaintext) {
				t.Errorf("Unprotect = %x, want %x", recovered, tc.plaintext)
			}
		})
	}
}

func TestEmptyInput(t *testing.T) {
	t.Parallel()

	if _, err := dpapi.Protect(nil); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("Protect code = %q, want %q", errors.CodeOf(err), errors.Invalid)
	}
	if _, err := dpapi.Unprotect(nil); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("Unprotect code = %q, want %q", errors.CodeOf(err), errors.Invalid)
	}
}

func TestTamperedBlobIsRejected(t *testing.T) {
	t.Parallel()

	protected, err := dpapi.Protect([]byte("secret"))
	if err != nil {
		t.Fatalf("Protect: %v", err)
	}

	protected[len(protected)-1] ^= 0xff
	if _, err = dpapi.Unprotect(protected); err == nil {
		t.Fatal("a tampered blob must not unprotect")
	}
	if errors.CodeOf(err) != errors.Internal {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Internal)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapters/secrets/dpapi/ -v`
Expected: FAIL, the package `dpapi` does not exist.

- [ ] **Step 3: Write the implementation**

Create `internal/adapters/secrets/dpapi/dpapi.go`:

```go
package dpapi

import (
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func Protect(plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, errors.New(errors.Invalid, "nothing to protect")
	}

	in := blob(plaintext)
	var out windows.DataBlob
	if err := windows.CryptProtectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return nil, errors.Wrap(err, errors.Internal, "protect the buffer")
	}
	return collect(&out)
}

func Unprotect(protected []byte) ([]byte, error) {
	if len(protected) == 0 {
		return nil, errors.New(errors.Invalid, "nothing to unprotect")
	}

	in := blob(protected)
	var out windows.DataBlob
	if err := windows.CryptUnprotectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return nil, errors.Wrap(err, errors.Internal, "unprotect the buffer")
	}
	return collect(&out)
}

func blob(data []byte) windows.DataBlob {
	return windows.DataBlob{Size: uint32(len(data)), Data: &data[0]}
}

func collect(out *windows.DataBlob) ([]byte, error) {
	result := make([]byte, out.Size)
	copy(result, unsafe.Slice(out.Data, out.Size))

	handle, err := windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "release the protected buffer")
	}
	if handle != 0 {
		return nil, errors.New(errors.Internal, "release the protected buffer")
	}
	return result, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race -cover ./internal/adapters/secrets/dpapi/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/adapters/secrets/dpapi/
git commit -m "feat(secrets): dpapi protection in user scope

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 11: The AES-GCM envelope

**Files:**
- Create: `internal/adapters/secrets/aesgcm/aesgcm.go`
- Test: `internal/adapters/secrets/aesgcm/aesgcm_test.go`

**Interfaces:**
- Consumes: `crypto/aes`, `crypto/cipher`, `crypto/rand`, `internal/kernel/errors`.
- Produces:
  - `const Version = "v1:"`
  - `const KeyLength = 32`
  - `func Seal(key, plaintext []byte) ([]byte, error)`
  - `func Open(key, envelope []byte) ([]byte, error)`

- [ ] **Step 1: Write the failing test**

Create `internal/adapters/secrets/aesgcm/aesgcm_test.go`:

```go
package aesgcm_test

import (
	"bytes"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/secrets/aesgcm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func key(fill byte) []byte {
	return bytes.Repeat([]byte{fill}, aesgcm.KeyLength)
}

func TestSealOpen(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		plaintext []byte
	}{
		{name: "empty", plaintext: []byte{}},
		{name: "application password", plaintext: []byte("abcd EFGH 1234 ijkl MNOP 5678")},
		{name: "binary", plaintext: bytes.Repeat([]byte{0x00, 0xff}, 64)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sealed, err := aesgcm.Seal(key(0x11), tc.plaintext)
			if err != nil {
				t.Fatalf("Seal: %v", err)
			}
			if !bytes.HasPrefix(sealed, []byte(aesgcm.Version)) {
				t.Fatalf("envelope = %x, want the %q prefix", sealed, aesgcm.Version)
			}
			if len(tc.plaintext) > 0 && bytes.Contains(sealed, tc.plaintext) {
				t.Fatal("the envelope must not contain the plaintext")
			}

			recovered, err := aesgcm.Open(key(0x11), sealed)
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			if !bytes.Equal(recovered, tc.plaintext) {
				t.Errorf("Open = %x, want %x", recovered, tc.plaintext)
			}
		})
	}
}

func TestNonceIsFresh(t *testing.T) {
	t.Parallel()

	first, err := aesgcm.Seal(key(0x22), []byte("same"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	second, err := aesgcm.Seal(key(0x22), []byte("same"))
	if err != nil {
		t.Fatalf("Seal again: %v", err)
	}
	if bytes.Equal(first, second) {
		t.Error("two seals of the same plaintext must differ")
	}
}

func TestOpenRejects(t *testing.T) {
	t.Parallel()

	sealed, err := aesgcm.Seal(key(0x33), []byte("secret"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	tampered := bytes.Clone(sealed)
	tampered[len(tampered)-1] ^= 0xff

	badVersion := bytes.Clone(sealed)
	badVersion[1] = '9'

	cases := []struct {
		name     string
		key      []byte
		envelope []byte
		want     errors.Code
	}{
		{name: "short key", key: []byte("short"), envelope: sealed, want: errors.Invalid},
		{name: "wrong key", key: key(0x44), envelope: sealed, want: errors.Invalid},
		{name: "tampered", key: key(0x33), envelope: tampered, want: errors.Invalid},
		{name: "wrong version", key: key(0x33), envelope: badVersion, want: errors.Invalid},
		{name: "truncated", key: key(0x33), envelope: sealed[:4], want: errors.Invalid},
		{name: "empty", key: key(0x33), envelope: nil, want: errors.Invalid},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, openErr := aesgcm.Open(tc.key, tc.envelope); errors.CodeOf(openErr) != tc.want {
				t.Errorf("code = %q, want %q", errors.CodeOf(openErr), tc.want)
			}
		})
	}
}

func TestSealRejectsAShortKey(t *testing.T) {
	t.Parallel()

	if _, err := aesgcm.Seal([]byte("short"), []byte("secret")); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapters/secrets/aesgcm/ -v`
Expected: FAIL, the package `aesgcm` does not exist.

- [ ] **Step 3: Write the implementation**

Create `internal/adapters/secrets/aesgcm/aesgcm.go`:

```go
package aesgcm

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	Version   = "v1:"
	KeyLength = 32
)

func Seal(key, plaintext []byte) ([]byte, error) {
	gcm, err := aead(key)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, errors.Wrap(err, errors.Internal, "generate the nonce")
	}

	envelope := make([]byte, 0, len(Version)+gcm.NonceSize()+len(plaintext)+gcm.Overhead())
	envelope = append(envelope, Version...)
	envelope = append(envelope, nonce...)
	return gcm.Seal(envelope, nonce, plaintext, nil), nil
}

func Open(key, envelope []byte) ([]byte, error) {
	gcm, err := aead(key)
	if err != nil {
		return nil, err
	}

	header := len(Version) + gcm.NonceSize()
	if len(envelope) < header+gcm.Overhead() || string(envelope[:len(Version)]) != Version {
		return nil, errors.New(errors.Invalid, "the secret envelope is not readable")
	}

	plaintext, err := gcm.Open(nil, envelope[len(Version):header], envelope[header:], nil)
	if err != nil {
		return nil, errors.Wrap(err, errors.Invalid, "the secret envelope is not readable")
	}
	return plaintext, nil
}

func aead(key []byte) (cipher.AEAD, error) {
	if len(key) != KeyLength {
		return nil, errors.New(errors.Invalid, "the secret key must be 32 bytes")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "create the cipher")
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "create the aead")
	}
	return gcm, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race -cover ./internal/adapters/secrets/aesgcm/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/adapters/secrets/aesgcm/
git commit -m "feat(secrets): aes-gcm envelope with a random nonce

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 12: The master key lifecycle

**Files:**
- Create: `internal/adapters/secrets/masterkey/masterkey.go`
- Test: `internal/adapters/secrets/masterkey/masterkey_test.go`

**Interfaces:**
- Consumes: `dpapi.Protect`, `dpapi.Unprotect` (task 10), `crypto/rand`, `internal/kernel/errors`.
- Produces:
  - `const FileName = "master.key"`
  - `const Length = 32`
  - `func Load(dir string) ([]byte, error)`

- [ ] **Step 1: Write the failing test**

Create `internal/adapters/secrets/masterkey/masterkey_test.go`:

```go
package masterkey_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/secrets/masterkey"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestLoadCreatesAndReuses(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "Postulator")

	first, err := masterkey.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(first) != masterkey.Length {
		t.Fatalf("length = %d, want %d", len(first), masterkey.Length)
	}
	if bytes.Equal(first, make([]byte, masterkey.Length)) {
		t.Fatal("the key must not be all zeroes")
	}

	stored, err := os.ReadFile(filepath.Join(dir, masterkey.FileName))
	if err != nil {
		t.Fatalf("read the key file: %v", err)
	}
	if bytes.Contains(stored, first) {
		t.Fatal("the key file must not contain the raw key")
	}

	second, err := masterkey.Load(dir)
	if err != nil {
		t.Fatalf("Load again: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Error("a second Load must return the same key")
	}
}

func TestLoadGeneratesDistinctKeys(t *testing.T) {
	t.Parallel()

	first, err := masterkey.Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	second, err := masterkey.Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if bytes.Equal(first, second) {
		t.Error("two directories must get different keys")
	}
}

func TestLoadRejects(t *testing.T) {
	t.Parallel()

	if _, err := masterkey.Load(""); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, masterkey.FileName), []byte("not protected"), 0o600); err != nil {
		t.Fatalf("write a corrupt key file: %v", err)
	}
	if _, err := masterkey.Load(dir); !errors.IsCode(err, errors.Internal) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Internal)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapters/secrets/masterkey/ -v`
Expected: FAIL, the package `masterkey` does not exist.

- [ ] **Step 3: Write the implementation**

Create `internal/adapters/secrets/masterkey/masterkey.go`:

```go
package masterkey

import (
	"crypto/rand"
	stderrors "errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/davidmovas/postulator/internal/adapters/secrets/dpapi"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	FileName = "master.key"
	Length   = 32

	directoryMode = 0o700
	fileMode      = 0o600
)

func Load(dir string) ([]byte, error) {
	if dir == "" {
		return nil, errors.New(errors.Invalid, "the master key directory must not be empty")
	}

	path := filepath.Join(dir, FileName)
	protected, err := os.ReadFile(path)
	if stderrors.Is(err, fs.ErrNotExist) {
		return create(path)
	}
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "read the master key")
	}

	key, err := dpapi.Unprotect(protected)
	if err != nil {
		return nil, err
	}
	if len(key) != Length {
		return nil, errors.New(errors.Internal, "the master key has the wrong length")
	}
	return key, nil
}

func create(path string) ([]byte, error) {
	if err := os.MkdirAll(filepath.Dir(path), directoryMode); err != nil {
		return nil, errors.Wrap(err, errors.Internal, "create the master key directory")
	}

	key := make([]byte, Length)
	if _, err := rand.Read(key); err != nil {
		return nil, errors.Wrap(err, errors.Internal, "generate the master key")
	}

	protected, err := dpapi.Protect(key)
	if err != nil {
		return nil, err
	}
	if err = os.WriteFile(path, protected, fileMode); err != nil {
		return nil, errors.Wrap(err, errors.Internal, "write the master key")
	}
	return key, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race -cover ./internal/adapters/secrets/masterkey/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/adapters/secrets/masterkey/
git commit -m "feat(secrets): dpapi-protected master key lifecycle

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 13: The secret store

**Files:**
- Create: `internal/adapters/secrets/store.go`
- Test: `internal/adapters/secrets/store_test.go`

**Interfaces:**
- Consumes: `aesgcm.Seal`, `aesgcm.Open` (task 11); satisfied at the call site by `*sqlite.SecretsRepo` (task 9).
- Produces:
  - `type vault interface { Put(context.Context, string, []byte) error; Get(context.Context, string) ([]byte, error); Delete(context.Context, string) error }`
  - `func NewStore(v vault, key []byte) *Store`
  - `func (s *Store) Put(ctx context.Context, ref, value string) error`
  - `func (s *Store) Get(ctx context.Context, ref string) (string, error)`
  - `func (s *Store) Delete(ctx context.Context, ref string) error`

- [ ] **Step 1: Write the failing test**

Create `internal/adapters/secrets/store_test.go`:

```go
package secrets

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/secrets/aesgcm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type memoryVault struct {
	rows map[string][]byte
}

func newMemoryVault() *memoryVault {
	return &memoryVault{rows: make(map[string][]byte)}
}

func (v *memoryVault) Put(_ context.Context, ref string, ciphertext []byte) error {
	v.rows[ref] = bytes.Clone(ciphertext)
	return nil
}

func (v *memoryVault) Get(_ context.Context, ref string) ([]byte, error) {
	ciphertext, ok := v.rows[ref]
	if !ok {
		return nil, errors.New(errors.NotFound, "secret "+ref+" is not stored")
	}
	return ciphertext, nil
}

func (v *memoryVault) Delete(_ context.Context, ref string) error {
	if _, ok := v.rows[ref]; !ok {
		return errors.New(errors.NotFound, "secret "+ref+" is not stored")
	}
	delete(v.rows, ref)
	return nil
}

func testKey() []byte {
	return bytes.Repeat([]byte{0x5a}, aesgcm.KeyLength)
}

func TestStoreRoundTrip(t *testing.T) {
	t.Parallel()

	vault := newMemoryVault()
	store := NewStore(vault, testKey())

	if err := store.Put(t.Context(), "wp.site.token", "abcd EFGH 1234"); err != nil {
		t.Fatalf("Put: %v", err)
	}

	stored := vault.rows["wp.site.token"]
	if !bytes.HasPrefix(stored, []byte(aesgcm.Version)) {
		t.Fatalf("stored = %x, want the %q prefix", stored, aesgcm.Version)
	}
	if strings.Contains(string(stored), "abcd EFGH 1234") {
		t.Fatal("the column must not hold the plaintext")
	}

	value, err := store.Get(t.Context(), "wp.site.token")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if value != "abcd EFGH 1234" {
		t.Errorf("Get = %q, want %q", value, "abcd EFGH 1234")
	}

	if err = store.Delete(t.Context(), "wp.site.token"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err = store.Get(t.Context(), "wp.site.token"); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.NotFound)
	}
}

func TestStoreRejects(t *testing.T) {
	t.Parallel()

	vault := newMemoryVault()

	cases := []struct {
		name string
		call func() error
		want errors.Code
	}{
		{
			name: "empty reference",
			call: func() error { return NewStore(vault, testKey()).Put(t.Context(), "", "x") },
			want: errors.Invalid,
		},
		{
			name: "short key",
			call: func() error { return NewStore(vault, []byte("short")).Put(t.Context(), "a", "x") },
			want: errors.Invalid,
		},
		{
			name: "missing secret",
			call: func() error { return NewStore(vault, testKey()).Delete(t.Context(), "absent") },
			want: errors.NotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if err := tc.call(); errors.CodeOf(err) != tc.want {
				t.Errorf("code = %q, want %q", errors.CodeOf(err), tc.want)
			}
		})
	}
}

func TestStoreRejectsAnotherKey(t *testing.T) {
	t.Parallel()

	vault := newMemoryVault()
	if err := NewStore(vault, testKey()).Put(t.Context(), "a", "x"); err != nil {
		t.Fatalf("Put: %v", err)
	}

	other := bytes.Repeat([]byte{0x01}, aesgcm.KeyLength)
	if _, err := NewStore(vault, other).Get(t.Context(), "a"); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapters/secrets/ -v`
Expected: FAIL, the package `secrets` does not exist.

- [ ] **Step 3: Write the implementation**

Create `internal/adapters/secrets/store.go`:

```go
package secrets

import (
	"context"

	"github.com/davidmovas/postulator/internal/adapters/secrets/aesgcm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type vault interface {
	Put(ctx context.Context, ref string, ciphertext []byte) error
	Get(ctx context.Context, ref string) ([]byte, error)
	Delete(ctx context.Context, ref string) error
}

type Store struct {
	vault vault
	key   []byte
}

func NewStore(v vault, key []byte) *Store {
	return &Store{vault: v, key: key}
}

func (s *Store) Put(ctx context.Context, ref, value string) error {
	if ref == "" {
		return errors.New(errors.Invalid, "secret reference must not be empty")
	}

	sealed, err := aesgcm.Seal(s.key, []byte(value))
	if err != nil {
		return err
	}
	return s.vault.Put(ctx, ref, sealed)
}

func (s *Store) Get(ctx context.Context, ref string) (string, error) {
	sealed, err := s.vault.Get(ctx, ref)
	if err != nil {
		return "", err
	}

	plaintext, err := aesgcm.Open(s.key, sealed)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func (s *Store) Delete(ctx context.Context, ref string) error {
	return s.vault.Delete(ctx, ref)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race -cover ./internal/adapters/secrets/... -v`
Expected: PASS, coverage at or above 85% for every package under `secrets`.

- [ ] **Step 5: Commit**

```bash
git add internal/adapters/secrets/store.go internal/adapters/secrets/store_test.go
git commit -m "feat(secrets): secret store sealing values into the vault

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 14: The settings loader

**Files:**
- Create: `internal/app/settings.go`
- Test: `internal/app/settings_test.go`

**Interfaces:**
- Consumes: `settings.Registry`, `settings.Values`, `(*Registry).NewValues`, `(*Registry).Apply`, `settings.New`, `(*Registry).Int`, `settings.IntRange` from `internal/kernel/settings`; satisfied at the call site by `*sqlite.SettingsRepo` (task 8).
- Produces:
  - `type settingsSource interface { All(ctx context.Context) (map[string]json.RawMessage, error) }`
  - `func LoadSettings(ctx context.Context, source settingsSource, registry *settings.Registry) (*settings.Values, []string, error)`

- [ ] **Step 1: Write the failing test**

Create `internal/app/settings_test.go`:

```go
package app

import (
	"context"
	"encoding/json"
	"io"
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/settings"
)

type fakeSource struct {
	stored map[string]json.RawMessage
	err    error
}

func (f fakeSource) All(context.Context) (map[string]json.RawMessage, error) {
	return f.stored, f.err
}

func TestLoadSettings(t *testing.T) {
	t.Parallel()

	registry := settings.New()
	workers := registry.Int("runs.workers", 2, settings.IntRange(1, 16))

	values, unknown, err := LoadSettings(t.Context(), fakeSource{stored: map[string]json.RawMessage{
		"runs.workers": json.RawMessage(`8`),
		"runs.ghost":   json.RawMessage(`1`),
	}}, registry)
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}
	if got := workers.Get(values); got != 8 {
		t.Errorf("runs.workers = %d, want 8", got)
	}
	if !slices.Equal(unknown, []string{"runs.ghost"}) {
		t.Errorf("unknown = %v, want [runs.ghost]", unknown)
	}
}

func TestLoadSettingsDefaults(t *testing.T) {
	t.Parallel()

	registry := settings.New()
	workers := registry.Int("runs.workers", 2, settings.IntRange(1, 16))

	values, unknown, err := LoadSettings(t.Context(), fakeSource{stored: map[string]json.RawMessage{}}, registry)
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}
	if len(unknown) != 0 {
		t.Errorf("unknown = %v, want none", unknown)
	}
	if got := workers.Get(values); got != 2 {
		t.Errorf("runs.workers = %d, want the default 2", got)
	}
}

func TestLoadSettingsRejectsAnOutOfRangeValue(t *testing.T) {
	t.Parallel()

	registry := settings.New()
	registry.Int("runs.workers", 2, settings.IntRange(1, 16))

	_, _, err := LoadSettings(t.Context(), fakeSource{stored: map[string]json.RawMessage{
		"runs.workers": json.RawMessage(`99`),
	}}, registry)
	if !errors.IsCode(err, errors.Invalid) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
	}
}

func TestLoadSettingsPropagatesTheSourceFailure(t *testing.T) {
	t.Parallel()

	_, _, err := LoadSettings(t.Context(), fakeSource{err: io.ErrUnexpectedEOF}, settings.New())
	if err == nil {
		t.Fatal("the source failure must reach the caller")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/app/ -run TestLoadSettings -v`
Expected: FAIL, the package does not build because `LoadSettings` is undefined.

- [ ] **Step 3: Write the implementation**

Create `internal/app/settings.go`:

```go
package app

import (
	"context"
	"encoding/json"

	"github.com/davidmovas/postulator/internal/kernel/settings"
)

type settingsSource interface {
	All(ctx context.Context) (map[string]json.RawMessage, error)
}

func LoadSettings(ctx context.Context, source settingsSource, registry *settings.Registry) (*settings.Values, []string, error) {
	stored, err := source.All(ctx)
	if err != nil {
		return nil, nil, err
	}

	values := registry.NewValues()
	unknown, err := registry.Apply(values, stored)
	if err != nil {
		return nil, unknown, err
	}
	return values, unknown, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race ./internal/app/ -run TestLoadSettings -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/app/settings.go internal/app/settings_test.go
git commit -m "feat(app): hydrate the settings registry from the database

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 15: The composition root

**Files:**
- Create: `internal/app/core.go`
- Test: `internal/app/core_test.go`
- Modify: `cmd/postulator/main.go`

**Interfaces:**
- Consumes: `sqlite.Open`, `(*sqlite.Store).Close`, `sqlite.NewSettingsRepo`, `sqlite.NewSecretsRepo` (tasks 5, 8, 9); `secrets.NewStore` (task 13); `masterkey.Load` (task 12); `LoadSettings` (task 14); `clock.System`, `settings.Default`.
- Produces:
  - `type Config struct { DatabasePath string; KeyDir string }`
  - `func DefaultConfig() (Config, error)`
  - `type Core struct { Store *sqlite.Store; Secrets *secrets.Store; Settings *settings.Values; UnknownSettings []string }`
  - `func Open(ctx context.Context, cfg Config) (*Core, error)`
  - `func (c *Core) Close() error`

- [ ] **Step 1: Write the failing test**

Create `internal/app/core_test.go`:

```go
package app_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestOpenAndClose(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	core, err := app.Open(t.Context(), app.Config{
		DatabasePath: filepath.Join(home, "postulator.db"),
		KeyDir:       home,
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if core.Store == nil || core.Secrets == nil || core.Settings == nil {
		t.Fatal("the core must carry the store, the secrets and the settings")
	}
	if len(core.UnknownSettings) != 0 {
		t.Errorf("UnknownSettings = %v, want none", core.UnknownSettings)
	}

	if err = core.Secrets.Put(t.Context(), "wp.site.token", "abcd EFGH"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	value, err := core.Secrets.Get(t.Context(), "wp.site.token")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if value != "abcd EFGH" {
		t.Errorf("Get = %q, want %q", value, "abcd EFGH")
	}

	if err = core.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestOpenReopensTheSameDatabase(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	cfg := app.Config{DatabasePath: filepath.Join(home, "postulator.db"), KeyDir: home}

	first, err := app.Open(t.Context(), cfg)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err = first.Secrets.Put(t.Context(), "wp.site.token", "abcd EFGH"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err = first.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	second, err := app.Open(t.Context(), cfg)
	if err != nil {
		t.Fatalf("Open again: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := second.Close(); closeErr != nil {
			t.Errorf("Close: %v", closeErr)
		}
	})

	value, err := second.Secrets.Get(t.Context(), "wp.site.token")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if value != "abcd EFGH" {
		t.Errorf("Get = %q, want %q", value, "abcd EFGH")
	}
}

func TestOpenRejectsAnEmptyConfig(t *testing.T) {
	t.Parallel()

	if _, err := app.Open(t.Context(), app.Config{}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
	}
}

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg, err := app.DefaultConfig()
	if err != nil {
		t.Fatalf("DefaultConfig: %v", err)
	}
	if filepath.Base(cfg.DatabasePath) != "postulator.db" {
		t.Errorf("DatabasePath = %q, want a file named postulator.db", cfg.DatabasePath)
	}
	if filepath.Base(cfg.KeyDir) != "Postulator" {
		t.Errorf("KeyDir = %q, want a directory named Postulator", cfg.KeyDir)
	}
	if filepath.Dir(cfg.DatabasePath) != cfg.KeyDir {
		t.Errorf("the database and the key must share a directory, got %q and %q", cfg.DatabasePath, cfg.KeyDir)
	}
}

func TestHealthService(t *testing.T) {
	t.Parallel()

	service := app.NewHealthService()
	if service.Ping() != app.Version {
		t.Errorf("Ping = %q, want %q", service.Ping(), app.Version)
	}

	info := service.BuildInfo()
	if info.Version != app.Version || info.Commit != app.Commit || info.BuildDate != app.BuildDate {
		t.Errorf("BuildInfo = %+v, want the ldflags values", info)
	}
	if strings.TrimSpace(info.Version) == "" {
		t.Error("the version must not be blank")
	}
	if app.Build() != info {
		t.Error("Build and BuildInfo must agree")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/app/ -run 'TestOpen|TestDefaultConfig|TestHealthService' -v`
Expected: FAIL, the package does not build because `Config`, `DefaultConfig`, `Core`, `Open` and `Close` are undefined.

- [ ] **Step 3: Write the implementation**

Create `internal/app/core.go`:

```go
package app

import (
	"context"
	stderrors "errors"
	"os"
	"path/filepath"

	"github.com/davidmovas/postulator/internal/adapters/secrets"
	"github.com/davidmovas/postulator/internal/adapters/secrets/masterkey"
	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/settings"
)

const (
	homeDirectory = "Postulator"
	databaseFile  = "postulator.db"
)

type Config struct {
	DatabasePath string
	KeyDir       string
}

func DefaultConfig() (Config, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return Config{}, errors.Wrap(err, errors.Internal, "locate the user configuration directory")
	}

	home := filepath.Join(base, homeDirectory)
	return Config{DatabasePath: filepath.Join(home, databaseFile), KeyDir: home}, nil
}

type Core struct {
	Store           *sqlite.Store
	Secrets         *secrets.Store
	Settings        *settings.Values
	UnknownSettings []string
}

func Open(ctx context.Context, cfg Config) (*Core, error) {
	if cfg.DatabasePath == "" {
		return nil, errors.New(errors.Invalid, "the database path must not be empty")
	}
	if cfg.KeyDir == "" {
		return nil, errors.New(errors.Invalid, "the key directory must not be empty")
	}

	key, err := masterkey.Load(cfg.KeyDir)
	if err != nil {
		return nil, err
	}

	store, err := sqlite.Open(cfg.DatabasePath, key)
	if err != nil {
		return nil, err
	}

	now := clock.System{}
	values, unknown, err := LoadSettings(ctx, sqlite.NewSettingsRepo(store, now), settings.Default())
	if err != nil {
		return nil, stderrors.Join(err, store.Close())
	}

	return &Core{
		Store:           store,
		Secrets:         secrets.NewStore(sqlite.NewSecretsRepo(store, now), key),
		Settings:        values,
		UnknownSettings: unknown,
	}, nil
}

func (c *Core) Close() error {
	return c.Store.Close()
}
```

Rewrite `cmd/postulator/main.go`:

```go
package main

import (
	"context"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/davidmovas/postulator/frontend"
	"github.com/davidmovas/postulator/internal/app"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := app.DefaultConfig()
	if err != nil {
		return err
	}

	core, err := app.Open(context.Background(), cfg)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := core.Close(); closeErr != nil {
			log.Print(closeErr)
		}
	}()

	wails := application.New(application.Options{
		Name:        "Postulator",
		Description: "Entity-graph driven WordPress content factory",
		Services: []application.Service{
			application.NewService(app.NewHealthService()),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(frontend.Assets),
		},
	})

	wails.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Postulator",
		Width:            1280,
		Height:           820,
		MinWidth:         960,
		MinHeight:        600,
		BackgroundColour: application.NewRGB(14, 16, 22),
		URL:              "/",
	})

	return wails.Run()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race -cover ./internal/app/ -v`
Expected: PASS

- [ ] **Step 5: Verify the binary still builds and starts**

Run: `task build`
Expected: `bin/postulator.exe` is produced; running it opens the window and creates `%APPDATA%\Postulator\master.key` and `%APPDATA%\Postulator\postulator.db`.

- [ ] **Step 6: Commit**

```bash
git add internal/app/core.go internal/app/core_test.go cmd/postulator/main.go
git commit -m "feat(app): open the encrypted store and the secret store at startup

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 16: Conventions, status and the full verification sweep

**Files:**
- Modify: `docs/CONVENTIONS.md`
- Modify: `docs/STATUS.md`

**Interfaces:**
- Consumes: everything built in tasks 1 to 15.
- Produces: no Go symbols.

- [ ] **Step 1: Run the full verification sweep**

```bash
go vet ./...
golangci-lint run
go test -race -coverprofile=coverage.out ./...
go run ./cmd/covergate
go tool cover -func=coverage.out
task build
```

Expected: `go vet` silent, `golangci-lint run` reports 0 issues, every test passes under `-race`, `covergate` passes, and `go tool cover -func` shows at or above 85% for `internal/adapters/sqlite`, `internal/adapters/sqlite/dbx`, `internal/adapters/sqlite/sqlitetest`, `internal/adapters/secrets`, `internal/adapters/secrets/dpapi`, `internal/adapters/secrets/aesgcm`, `internal/adapters/secrets/masterkey` and `internal/app`. If a package is short, add the missing table cases to its existing test file rather than a new one.

- [ ] **Step 2: Confirm the dependency rule still holds**

Run: `go test -race ./internal/app/ -run 'TestLayers|TestDomain|TestOnly' -v`
Expected: PASS. `internal/adapters` now exists, so `TestOnlyPagingUsesTheQueryBuilder` and
`TestOnlyTheCompositionRootImportsTransport` see the new packages in `go list ./...`; the
tree-based rules in `TestLayersDoNotReachUpwards` still skip `domain` and `application`,
which stay absent until Phase 2.

- [ ] **Step 3: Add the migration section to `docs/CONVENTIONS.md`**

Insert a `## Migrations` section between `## Timestamps` and `## Code rules`:

```markdown
## Migrations

Migrations are embedded SQL run by goose v3 at `Store.Open`. They live in
`internal/adapters/sqlite/migrations` and are embedded with `//go:embed migrations/*.sql`.

- Filenames are `NNNN_snake_case.sql`: four digits, zero-padded, gapless, never
  renumbered once committed.
- Every file carries both `-- +goose Up` and `-- +goose Down`. A migration that cannot be
  reversed does not ship; the round-trip test applies up, down and up again for all of them.
- One concern per migration. The down section drops exactly what the up section created,
  in reverse order.
- Tables are `STRICT`. Identifiers are lowercase snake_case, keywords uppercase, one
  column per line. Text primary keys are `TEXT PRIMARY KEY`; timestamps are
  `TEXT NOT NULL` holding RFC3339 UTC.
- No `IF NOT EXISTS` and no `IF EXISTS`: a database that is not in a known state must
  fail loudly rather than drift.
- `goose_db_version` belongs to goose and is never read from application code.

Files are scaffolded with the goose CLI and renamed to the four-digit form:

    go run github.com/pressly/goose/v3/cmd/goose@v3.26.0 -s -dir internal/adapters/sqlite/migrations create <name> sql

The CLI is never pointed at a Postulator database: its `sqlite3` dialect is backed by
`modernc.org/sqlite`, which cannot read an adiantum-encrypted file. Migrations are applied
only through `Store.Open`, which uses `goose.NewProvider` so that two stores opening in
parallel tests cannot race on goose's package-level filesystem and dialect.
```

Add to the pinned toolchain table:

```markdown
| ncruces/go-sqlite3 | `v0.30.1` | `go.mod` |
| goose | `v3.26.0` | `go.mod` |
| golang.org/x/sys | `v0.46.0` | `go.mod` |
```

- [ ] **Step 4: Update `docs/STATUS.md`**

Replace the "Where we are" section with a Phase 1A entry, move the answered items out of "Open questions for Phase 1", and record the decisions:

```markdown
## Decisions taken in Phase 1A

- **Two pools over one file.** A writer `*sql.DB` capped at one connection with
  `_txlock=immediate`, and a reader `*sql.DB` opened `mode=ro` with four connections.
  One pool capped at a single connection would serialise reads behind long writes; an
  uncapped pool would let two goroutines both begin write transactions and turn WAL's
  single-writer rule into `SQLITE_BUSY` at commit time.
- **`execFrom` reads, `writeFrom` writes.** Both return the ambient `*sql.Tx` when
  `Store.Do` is active, found through an unexported context key. Outside a transaction
  `execFrom` returns the reader and `writeFrom` returns the writer. The spec sketched one
  accessor; with two pools one accessor would make the reader dead code.
- **Nested `Do` reuses the outer transaction** and opens no savepoint. A nested savepoint
  would let an inner rollback be swallowed while the outer commits.
- **Migrations run through `goose.NewProvider`, not `goose.UpContext`.** `UpContext` reads
  the package-level filesystem and dialect that `SetBaseFS`/`SetDialect` mutate, which
  races when parallel tests open stores under `-race`.
- **The adiantum key travels as the `hexkey` URI parameter.** The VFS reads it at
  file-open time, strictly before `journal_mode(WAL)`; the PRAGMA form would need SQL
  quoting inside a `_pragma=` value and a hex string starting with a digit is not a safe
  bare pragma token.
- **The master key lives at `%APPDATA%\Postulator\master.key` via `os.UserConfigDir()`.**
  `xdg.ConfigHome` resolves to `%LOCALAPPDATA%` on Windows and would put it elsewhere.
- **No `//go:build windows` tags.** The application is Windows-only and a tag would
  demand a second implementation file that would be a stub.
- **Adiantum, WAL and the busy timeout work together.** The open question from Phase 0 is
  answered: `TestOpenAppliesPragmas` reports `journal_mode=wal`, `foreign_keys=1`,
  `busy_timeout=5000` and `synchronous=1` on both the plain and the encrypted store.
```

- [ ] **Step 5: Re-run the sweep after the doc edits**

```bash
go test -race ./... && golangci-lint run
```

Expected: green.

- [ ] **Step 6: Commit**

```bash
git add docs/CONVENTIONS.md docs/STATUS.md
git commit -m "docs: migration conventions and the phase 1a decision log

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

## Self-review

**Spec coverage.** Section 9.1 is covered by tasks 1, 4, 5, 6, 7, 8 and 9 — `Open(path, key)` with the four pragmas and the adiantum VFS, `Do` with the tx in the context and `execFrom`, extended-code error mapping, `dbx.Result[T]`, embedded goose migrations and `sqlitetest.Open`. The spec's full table list is deliberately not shipped here: section 9.1 enumerates the Phase 2 schema, and the roadmap row for Phase 2 owns migrations 0001 to 0006. Phase 1A ships only `app_meta`, `settings` and `secrets`, which is what the phase task scopes. Section 9.2 is covered by tasks 10 to 13 — DPAPI, the master key, AES-GCM with the `v1:` envelope and a `Store` with the port's method set. Master password and encrypted export are explicitly Phase 12 and appear in no task. Section 4 names are reused verbatim throughout; no kernel symbol is redefined.

**Placeholder scan.** No task says "similar to", "TBD" or "handle errors appropriately"; every code step carries complete Go or SQL. No Go code block contains a comment. The only prose-only steps are verification runs and the two documentation edits, both of which quote the exact text to insert.

**Type consistency.** `Store`, `Open`, `Close`, `Path`, `Do`, `execFrom`, `writeFrom`, `txFrom`, `executor`, `provider`, `migrate`, `migrations`, `dsn`, `keyLength` are spelled identically in every task that mentions them. `dbx.Convert`, `dbx.Classify`, `dbx.From`, `dbx.IsNotFound`, `dbx.IsConflict` and `BusyRetryAfter` match between tasks 2, 3, 5, 6, 8 and 9. `aesgcm.Version`, `aesgcm.KeyLength`, `aesgcm.Seal` and `aesgcm.Open` match between tasks 11 and 13. `masterkey.FileName`, `masterkey.Length` and `masterkey.Load` match between tasks 12 and 15. `NewSettingsRepo(store, clk)` and `NewSecretsRepo(store, clk)` take a `clock.Clock` in tasks 8, 9 and 15 alike. `LoadSettings` has one signature in tasks 14 and 15.

## Open question for the orchestrator

`cmd/covergate` enforces only the two gates from Phase 0 (`domain`+`application` >= 80, module >= 70). This plan verifies the 85% target for the new packages by reading `go tool cover -func` in task 16 rather than by teaching `covergate` a third gate, because changing the gate program is a policy decision rather than a Phase 1A deliverable. Say the word if `covergate` should grow an `internal/adapters` gate and it will be added to task 16.
