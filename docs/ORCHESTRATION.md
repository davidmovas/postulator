# Orchestration

How the rebuild is run by agents. The phase list itself lives in
`docs/superpowers/plans/2026-09-17-postulator-v2-roadmap.md`.

## Roles

- **Orchestrator** — owns the phase order, writes each task, approves each plan, keeps
  `docs/STATUS.md` honest. Writes tasks in English; talks to the user in Russian.
- **Implementing agent** — drafts the phase plan, gets it approved, then implements it
  with TDD and commits its own work.
- **Reviewer** — a separate agent that checks the phase against its plan: tests present
  and meaningful, coverage gates met, dependency rule green, docs consistent with the
  spec.

## Limits

- At most **1 Fable** and **2 Opus** agents at a time.
- **Sonnet** is for web research and bulk reading only, never for implementation.
- **Agents never spawn sub-agents.** An agent given a task executes it.
- Fable is reserved for the phases where the design is dense rather than broad: 2, 5
  and 6, one at a time.

## The cycle for a phase

1. Orchestrator writes the task, naming the Archond files to read first (section 15 of
   the spec) and the exact scope.
2. The implementing agent reads them, then writes
   `docs/superpowers/plans/2026-MM-DD-phase-N-<name>.md` in the writing-plans format.
3. Orchestrator and user approve the plan. **No code is written before approval.**
4. The agent implements with TDD: failing test, run it, implement, run it, commit. One
   commit per task, conventional commits, attribution line at the end of every message.
5. A separate reviewer reviews.
6. Fixes land, `docs/STATUS.md` is updated in the same commit.

## Definition of done

A phase is done when all of the following are green and the reviewer has signed off:

```
task build
go vet ./...
golangci-lint run
go test -race ./...
go run ./cmd/covergate
```

plus `internal/app/deps_test.go` passing and `docs/STATUS.md` updated. "Probably fine"
is not a state; run the commands and read the output.

## Model allocation

| Phase | Agents |
|---|---|
| 0 Wipe, scaffold, docs | Opus x1 |
| 1 Infra and contracts spike | Opus x2 in parallel |
| 2 Schema, domain, repositories, use cases | **Fable** |
| 3 WordPress adapter and PHP plugin | Opus x2 in parallel |
| 4 LLM layer | Opus x1 |
| 5 Run engine | **Fable** |
| 6 Content factory core | **Fable** |
| 7 Remaining steps and site metrics | Opus x2 |
| 8 Import | Opus x1 |
| 9 Agents | Opus x2 |
| 10 Schedules | Opus x1 |
| 11 Wails services, events, TS generation | Opus x2 |
| 12 Hardening and release | Opus x1 |

## Rules that apply to every agent

- Read `docs/STATUS.md` first. It is the handoff point between sessions.
- Read the Archond files named for your module before designing it. Port the shapes, not
  the SaaS wiring: never Asynq, Postgres, jet, pgx, fx, Fiber, credits or HTTP status
  plumbing.
- No comments in Go, TypeScript, YAML or PHP. Prose belongs in `docs/` and `CLAUDE.md`.
- No stubs, no TODOs, no placeholder implementations. If something genuinely cannot be
  done, stop and report it rather than leaving a hole that looks finished.
- Interfaces are declared by the consumer, never speculatively.
- Bad code is rewritten, not patched.
- Markdown files are limited to `CLAUDE.md` and the set in section 10 of the spec. Do
  not add others.
- Commit on `rewrite/v2`. Never amend, rebase, force-push or `--no-verify`.
