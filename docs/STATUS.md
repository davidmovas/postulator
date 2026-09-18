# Status

The handoff point between sessions. Read this first. The reasoning behind every
phase is in [`DECISIONS.md`](DECISIONS.md).

**Branch:** `rewrite/v2`, cut from `dev`. **Target release:** `v2.0.0`, not tagged yet.

## Where we are

**Phases 0 through 12 are complete.** The application composes a Wails v3 window over an
adiantum-encrypted SQLite store: the entity graph, the page map, the templates, the WordPress
adapter and its companion plugin, the LLM stack, the run engine with its twelve page steps, the
import, the agent, the schedules, the fourteen bound services and their TypeScript surface. A
master password locks and unlocks the whole core, a backup of the database round trips through
one encrypted archive, and the sweep retires artifacts and run events on their own windows.

The last gate run on 2026-09-18: `go build`, `go vet`, `golangci-lint` (v2.13.2, 0 issues),
`go test -count=1 -race -covermode=atomic ./...`, `go run ./cmd/covergate`, `gofmt -l .`,
`task build`, `task package` and `task e2e:full` are green, and `bin/postulator.exe` opens its
window and answers `health.ping`.

The frontend is still the Vite stub. Phase 11 shipped the typed surface it will be built on —
the generated bindings, `lib/api.ts`, `lib/events.ts` and `smoke.ts` — but no screen.

## How to run

```
task build                 bin/postulator.exe, builds the frontend and the bindings first
task package               the NSIS installer in bin/
task plugin:zip            bin/postulator-companion.zip
go test -count=1 -race -covermode=atomic -coverprofile=coverage.out ./...
go run ./cmd/covergate     enforces the coverage gates on that profile
$(go env GOPATH)/bin/golangci-lint.exe run
task e2e:up                the docker WordPress stack on http://localhost:8089
task e2e:test              the plugin suite against it
task e2e:full              the whole loop: sync, import, five drafts, relink, backup
task lint:e2e              the build-tagged sources, which `golangci-lint run` skips
task e2e:down              stops the stack and drops its volumes
```

## Phases

| # | Phase | Status |
|---|---|---|
| 0 | Wipe, scaffold, kernel, CI, docs | done, reviewed |
| 1A | SQLite store, migrations, unit of work, secrets | done, reviewed |
| 1B | Wails contracts spike, events, error conversion | done, reviewed |
| 2 | Domain core: graph, page map, templates, settings | done |
| 3A | WordPress adapter and `wptest` | done, reviewed |
| 3B | Companion plugin and the docker e2e harness | done, reviewed |
| 4 | LLM port, gollem client, catalog, ledger, limiter, retry | done |
| 5 | Run engine, checkpoints, recovery | done |
| 6 | Content factory: document, links, compliance, first steps | done |
| 7 | Remaining steps, images, sync, site reports | done |
| 8 | Import and export of the client page map | done |
| 9 | Tool registry, agent runner, confirmations | done |
| 10 | Schedules and the ticker | done |
| 11 | Wails services, event bridge, TypeScript generation | done |
| 12 | Master password, backup, retention, e2e, release | done |

Module coverage is 88% of 14k statements; `domain` + `application` sit at 92%.

## Known gaps

- Application events published before `cmd/postulator` connects the relay to the Wails event
  manager are dropped, by design: nothing listens before the window exists.
- Backward paging exists in every repository (`paging.Request.Before`) but not at the use-case
  boundary, because `dto.ListRequest` carries one cursor; the client replays the cursor it used
  to reach the current page.
- Step `params` keys are `allowErrors` on `validate` and `iterations` on `repair_links`; the
  shipped seeds carry none, so both take their defaults.
- `lefthook` is not installed on the development machine. Install it with
  `go install github.com/evilmartians/lefthook@latest && lefthook install`.
- **`golangci-lint` on this machine must be run from `$(go env GOPATH)/bin`.** A scoop shim
  earlier on `PATH` is v2.11.4 built with go1.26 and refuses a `go 1.27` module outright.
- The frontend is the stub described above.
- Core REST filters `modified_after` on the site-local `post_modified` while returning the UTC
  `modified_gmt`; the sync uses the plugin's `/content?since=`, which compares `post_modified_gmt`.
- `internal/adapters/wp/e2e` carries its own small HTTP client rather than `internal/adapters/wp`,
  which had not landed when it was written. `internal/e2e` goes through the real adapter.
- `content.Assess` has no Wails method; `ReportsService.JudgePage` binds `content.Judge` and runs
  it synchronously, and `PageReport` still reads what a run already recorded.
- `ledger.List` has no caller: `ModelsService.UsageSummary` answers from the aggregate, and a
  per-call ledger screen is what would read the list.
- A run's deadline is the `runtime.DefaultRunDeadline` constant, not a setting: nothing in the UI
  sets one yet, and a second knob with no reader would be dead configuration.
- The docker e2e stack is never run in CI: `windows-latest` cannot run Linux containers, and the
  Ubuntu job exists only to lint and package the plugin. Both suites are run by hand before a tag.
- **Migration 0016 runs outside goose's transaction.** Rebuilding `run_items` means dropping it,
  and a drop with foreign keys on performs an implicit delete that would cascade into `artifacts`
  and `step_execs`. The statements are wrapped in an explicit `BEGIN`/`COMMIT`, so the rebuild is
  atomic, but a crash between that commit and goose recording the version leaves a database the
  next start cannot migrate. The recovery is the reset the startup error already names.
- A pending action keeps the arguments it will replay, so a confirmation for a tool that carries
  a credential holds that credential in the encrypted database until the action is settled. The
  event, the summary and the tool call ledger carry it masked.
- An agent tool that addresses a record by id relies on the use case to scope the write, exactly
  as the Wails services do. The use cases are where that is closed.
- **Media uploaded by `generate_images` is orphaned when the item later fails.** Accepted: an
  upload is cheap, a delete needs `force=true` to skip the trash, and a sweep that removes media
  a human may already have reused elsewhere is the worse failure.
- A backup is restored into the same installation. Carrying one to another machine works because
  the archive holds a plain database, but nothing yet checks that the site credentials inside it
  still decrypt with the new master key: the secrets are sealed with the key of the machine that
  wrote them, so they are unreadable after a restore elsewhere and have to be entered again.
- `Lock()` refuses while no master password is set, because nothing could unlock the core again.
  Setting a password while the core is locked also unlocks it, since the caller proved the old one.

## Next steps

1. Build the real frontend on the bound services and the generated events.
2. Tag `v2.0.0` when the UI lands; `release.yml` publishes the executable, the NSIS installer and
   the companion plugin archive from that tag.
3. Close the gaps above that the UI reaches: the ledger screen, the page audit screen, the run
   deadline setting.
