# Postulator v2 — Implementation Roadmap

Sections 11-14 of the approved plan, verbatim, with the Phase 0 task list kept
current as work lands.

---

## 11. Phase Roadmap

| # | Phase | Agents | Depends on | Key deliverables |
|---|---|---|---|---|
| 0 ✅ | Wipe + scaffold + docs | Opus ×1 | — | branch, Go 1.27, Wails v3, Vite stub, kernel, CI, docs, CLAUDE.md |
| 1 ✅ | Infra + contracts spike | Opus ×2 (parallel) | 0 | sqlite store/migrations/UoW/testhelper, secrets (DPAPI, AES-GCM), adiantum; spike on v3 errors/events/generics → CONTRACTS.md |
| 2 ✅ | Schema + domain graph/pagemap/template/settings + repos + use cases | **Fable** | 1 | migrations 0001–0006, domain packages with tests, repositories, application commands/queries, seed templates |
| 3 ✅ | WordPress | Opus ×2 (parallel: Go adapter+wptest+sync / PHP plugin+docker e2e) | 2 | `adapters/wp`, `wptest`, sync use case, `wp-plugin/`, `docker/e2e` compose |
| 4 ✅ | LLM layer | Opus ×1 | 2 | port, gollem client, catalog, profiles, ledger, limiter, retry, record/replay |
| 5 | Run engine | **Fable** | 2, 4 | `domain/run`, `runtime` engine/worker/checkpoint/eventlog/recovery/budget, migrations for runs |
| 6 | Content factory core | **Fable** | 5 | `domain/content` (Document, LinkContext, InsertLinks, Compliance, Structure), steps `resolve_context, generate_body, insert_links, repair_links, validate` |
| 7 | Remaining steps + site metrics | Opus ×2 | 3, 6 | `generate_meta, generate_images, judge, publish, relink_neighbors, sync_back, report`; `application/reports` site-level metrics |
| 8 | Import | Opus ×1 | 2 | xlsx/csv, mappings, preview, apply |
| 9 | Agents | Opus ×2 | 4, 5, 7 | tool registry over all use cases, gollem agent, confirmations, conversations, streaming events; AI-backed use cases `graph.ProposeFromPages` (entities, edges, anchors proposed from synced pages, status `proposed`), `graph.ProposeRelated`, `content.Judge` (page audit on demand) registered as tools |
| 10 | Schedules | Opus ×1 | 5 | `domain/schedule`, ticker, use cases, tools |
| 11 | Wails services + events + TS generation | Opus ×2 | 7, 8, 9, 10 | all services, event bridge, generator, CONTRACTS.md final |
| 12 | Hardening + release | Opus ×1 | 11 | e2e runs, coverage, export/backup, master password, retention job, v2.0.0 |

Phases 7, 8, 10 may interleave after 6 depending on token limits. Each phase's detailed plan is written at its start (see header).

## 12. Phase 0 — Wipe, Scaffold, Docs (detailed tasks)

**Files:**
- Delete: `internal/`, `pkg/`, `frontend/`, `main.go`, `Makefile`, `scripts/`, `examples/topics.txt`, `nul`, `wails.json`, `go.sum`
- Keep: `.git/`, `.github/`, `.gitignore`, `build/` icons only, `examples/sitemap-import-example.{csv,json,xlsx}`, `examples/sitemap.json`
- Create: `cmd/postulator/main.go`, `internal/kernel/{errors,paging,id,clock,log,ctx,dto,middleware}/*.go` + tests, `internal/app/deps_test.go`, `frontend/` (Vite TS stub from `wails3 init`), `build/config.yml`, `Taskfile.yml`, `.golangci.yml`, `.github/workflows/ci.yml`, `CLAUDE.md`, `docs/*.md`, `docs/superpowers/specs/2026-09-17-postulator-v2-design.md`, `docs/superpowers/plans/2026-09-17-postulator-v2-roadmap.md`

**Interfaces produced:** everything in section 4 with exact names; `cmd/postulator/main.go` runs a Wails v3 app with one `HealthService{Ping() string}` so the binary starts.

- [x] **Task 0.1: Branch and wipe**
  - Steps: `git checkout dev && git pull`, `git checkout -b rewrite/v2`; delete the listed paths with `git rm -r`; add `.idea/`, `frontend/node_modules/`, `frontend/dist/`, `bin/`, `*.log` to `.gitignore`; commit `chore: wipe v1 codebase for v2 rewrite`.
- [x] **Task 0.2: Toolchain**
  - `go mod edit -go=1.27`; `go install github.com/wailsapp/wails/v3/cmd/wails3@<pinned tag>` (agent looks up the latest v3 tag on GitHub and records it in `docs/CONVENTIONS.md`); `wails3 doctor` must pass; verify stdlib uuid presence with `go doc` against 1.27 and record the ID decision in CONVENTIONS.
- [x] **Task 0.3: Wails v3 skeleton**
  - Run `wails3 init -n postulator -t vanilla-ts` into a temp dir, copy `frontend/`, `build/`, `Taskfile.yml`, `main.go` template into repo, move main to `cmd/postulator/main.go`, keep existing icons in `build/`. Add `HealthService` with `Ping() string` returning the version from `internal/app/version.go` (ldflags). `task build` produces `bin/postulator.exe`. Commit `feat: wails v3 skeleton with health service`.
- [x] **Task 0.4: kernel/errors (TDD)** — read Archond `internal/shared/errors/errors.go` and `codes.go` first.
  - Test: `New(NotFound, "x").Error() == "x"`, `CodeOf(Wrap(io.EOF, External, "wp")) == External`, `errors.Is(Wrap(io.EOF,…), io.EOF)`, `errors.Is(New(NotFound,"a"), New(NotFound,"b"))` (by code), `CodeOf(io.EOF) == Internal`, `WithDetail`/`WithRetry` return clones and leave the original untouched, `Stack(err)` non-empty. Implement per section 4. Commit `feat(kernel): errors with codes`.
- [x] **Task 0.5: kernel/paging (TDD)** — read Archond `pkg/dbx/page.go`, `pkg/dbx/keyset.go`, `pkg/api/list.go` first.
  - Test: cursor round-trip for `{o,d,v,i}` with Text/Time/Int keys; invalid base64 → `Invalid`; `Request{Limit:0}.Normalize().Limit == 50`, `Limit: 9999 → 500`; `Keyset` on a squirrel select produces `WHERE (created_at, id) > (?, ?) ORDER BY created_at ASC, id ASC LIMIT 51` and the mirrored `<`/`DESC` form for `Before`; `Cut` returns `HasMore` and `Cursors.Next` only when `len > limit`; `Slice[T](nil)` marshals to `[]`. Commit `feat(kernel): cursor pagination`.
- [x] **Task 0.6: kernel/id, clock, ctx, dto, log, settings, middleware (TDD)** — read Archond `pkg/logger/logger.go` and `internal/settings/constructors.go` first.
  - id: `Valid(New())`, `!Valid("x")`. clock: `Fake.Advance`. ctx: round-trip of run/conversation/actor. dto: `Time` marshals RFC3339 UTC and parses back; `ListRequest` defaults. log: redaction replaces values for keys `password|apiKey|token|authorization` with `***`; writes to two files via lumberjack. settings: `Int("runs.workers", 2, IntRange(1,16))` rejects 0 and 17, defaults when unset, `Schema()` lists all declared settings. middleware: `Recover` converts panic to `Internal` error; `Timeout` returns `Cancelled`. Commit `feat(kernel): shared runtime helpers`.
- [x] **Task 0.7: Dependency-rule test**
  - `internal/app/deps_test.go` runs `go list -deps -f '{{.ImportPath}}' ./internal/domain/...` and asserts no import path contains `/adapters/`, `/runtime/`, `/transport/`, `/application/`, or any third-party module except `golang.org/x/net/html`; same for `application` (may import domain+kernel+squirrel is NOT allowed). Commit `test: dependency rule`.
- [x] **Task 0.8: CI, lint, hooks** — read Archond `.golangci.yml` and `lefthook.yml` first.
  - `.golangci.yml` in Archond style (errcheck with `check-blank` + `check-type-assertions`, govet shadow, staticcheck, gocritic, misspell; a one-line rationale next to every tuned rule), `lefthook.yml` (pre-commit: gofmt, goimports, lint `--new-from-rev`; commit-msg: conventional-commit regex; pre-push: build + short tests), `.github/workflows/ci.yml`: windows-latest, Go 1.27, `task build`, `go vet`, `golangci-lint`, `go test -race -coverprofile`, coverage gate program `cmd/covergate` (reads profile, enforces 80/70). Update `release.yml` to build with `task build` and tag `v2.0.0-*` prereleases. Commit `ci: v2 pipeline with coverage gates and hooks`.
- [x] **Task 0.9: Docs and CLAUDE.md**
  - Write the docs listed in section 10 from this plan: spec = sections 1–10 verbatim, roadmap = sections 11–13, VISION/ARCHITECTURE/CONVENTIONS/CONTRACTS(initial)/ORCHESTRATION/STATUS distilled (each ≤ 150 lines). `CLAUDE.md` ≤ 40 lines: the code rules, `task build`, `go test -race ./...`, "read docs/STATUS.md first", "no md files besides docs/", agent limits. Commit `docs: v2 vision, architecture, conventions, orchestration, status`.
- [ ] **Task 0.10: Phase review**
  - Opus reviewer checks: binary starts, kernel tests pass with ≥ 90% coverage, deps test passes, docs consistent with spec. Update `docs/STATUS.md` (Phase 0 done, Phase 1 next). Commit `docs(status): phase 0 complete`.

## 13. Verification

- Per phase: `task build && go vet ./... && golangci-lint run && go test -race ./...` green; `go run ./cmd/covergate` passes; `internal/app/deps_test.go` passes; reviewer sign-off; STATUS updated.
- Phase 3: adapter tests against `wptest`; e2e tag against docker WP with the plugin installed: create page, write SEO meta, bulk content endpoint returns links and hashes.
- Phase 5: kill-and-restart test: start a run with 3 items and a fake step that panics on item 2 attempt 1; restart engine; assert items resume from last done step, artifacts reused (no second `generate` call), events have gapless `seq`.
- Phase 6: golden tests for `InsertLinks` (UTF-8 anchors, nested tags, headings, existing links, limits), `Compliance` on hand-written HTML; property test: `InsertLinks` is idempotent.
- Phase 7: replay-fixture run end-to-end against `wptest`: planned page → draft published with links to parent and grandparent, children section, no external links; compliance score 1.0.
- Phase 9: agent conversation replay: "generate all planned pages under entity X as drafts" → tool `runs.start` called with correct targets after confirmation event.
- Phase 11: generated `events.ts` and service bindings compile in the Vite stub; a smoke script in `frontend/src/smoke.ts` calls `Sites.List`, starts a run, and follows `run.*` events.
- Phase 12: full e2e on docker WP: import sample xlsx → graph → generate 5 pages (draft) → relink → audit report; export/backup round-trip; master-password lock/unlock.

## 14. Immediately After Approval

1. Create branch `rewrite/v2`, dispatch one Opus agent with Phase 0 (tasks 0.1–0.9), then an Opus reviewer for 0.10.
2. On Phase 0 completion, dispatch two Opus agents for Phase 1 (infra / contracts spike) to draft their plans; approve; implement.
3. Keep `docs/STATUS.md` as the handoff point for any future orchestrator session.
