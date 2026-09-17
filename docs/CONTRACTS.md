# Contracts

The rules every boundary obeys: Wails services to the frontend, and the tool registry to
the agents. **Initial version.** The error envelope, the event envelope and the generics
question are settled by the Phase 1 spike against Wails v3 and recorded here; Phase 11
finalises the service catalogue.

## Status of this document

| Area | State |
|---|---|
| Pagination | Settled. `kernel/paging` ships and is tested. |
| Timestamps and ids | Settled. RFC3339 UTC text, UUID v4 text. |
| Error codes | Settled. `kernel/errors` ships with a frozen code set. |
| Error wire format | **Open.** Phase 1 spike. |
| Event envelope and generated `events.ts` | **Open.** Phase 1 spike, generator in Phase 11. |
| Generics across the TypeScript generator | **Open.** Phase 1 spike. |
| Service catalogue | **Open.** Phase 11. |

## Method shape

```go
func (s *XService) Method(ctx context.Context, req ReqDTO) (RespDTO, error)
```

One request struct in, one response struct out, always a context, always an error. No
positional parameter lists: a field added to a struct is a compatible change, an extra
argument is not.

## DTOs

- JSON is camelCase everywhere: `siteId`, `nextCursor`, `createdAt`. Never snake_case.
- Timestamps are `kernel/dto.Time`: RFC3339 with a UTC offset, seconds precision, `null`
  when zero.
- Ids are UUID v4 lowercase text.
- A nil slice marshals as `[]`, never as `null` — `paging.Slice[T]` exists for this. The
  frontend should never have to guard a list.
- DTOs are declared in the transport layer and mapped by hand. Domain types do not carry
  JSON tags.

## Errors

Every error crossing a boundary is a `*kernel/errors.Error` with one of the frozen
codes: `NOT_FOUND CONFLICT INVALID UNAUTHORIZED RATE_LIMITED BUDGET_EXCEEDED EXTERNAL
INTERNAL CANCELLED NEEDS_HUMAN LOCKED`. A foreign error read through `CodeOf` reports
`INTERNAL`, so the frontend always has a code to switch on.

The transport encoding is the open question. The candidate is a JSON body
`{"code","message","details"}` carried in the error string, with a `parseError` helper
on the frontend. Phase 1 decides and this section is rewritten with the answer.

`details` never carries a secret, a stack trace or an internal driver message. Those go
to `errors.log`.

## Pagination

Cursor pagination only. There is no offset anywhere in this codebase.

Request: `kernel/dto.ListRequest{cursor, limit, sort?}`. Limit defaults to 50 and clamps
at 500.

Response: `{items, nextCursor?, prevCursor?, hasMore}`.

The cursor is an opaque base64url string. It records the sort fields and direction it
was issued for, and a cursor replayed against a different `ORDER BY` is rejected as
`INVALID` rather than silently skipping or repeating rows. Clients treat it as opaque
and pass it back unchanged.

## Long-running work

Any mutation that can exceed a second returns `{runId}` immediately. It never blocks the
call.

Progress is read two ways, and both are available:

- `RunsService.ListEvents(runId, sinceSeq, limit)` — the durable log, gapless per run,
  which is what a UI that was closed and reopened replays.
- Live events on the bus — the same records, pushed.

Control is `Get(runId)`, `Pause(runId)`, `Resume(runId)`, `Cancel(runId)`.

## Events

One Go registry owns the event names and payload structs; the TypeScript is generated
from it, never written by hand. The envelope is `{type, seq, runId?, at, payload}`.

Run events: `run.queued run.started run.paused run.resumed run.cancelled run.completed
run.failed run.budget_exceeded item.started item.done item.failed item.needs_human
step.started step.done step.failed step.retrying llm.usage`.

Other events: `graph.changed{siteId}`, `pages.changed{siteId}`, `templates.changed`,
`agent.delta`, `agent.tool.started`, `agent.tool.finished`, `agent.confirm.requested`,
`app.locked`, `app.unlocked`.

`seq` is per run and gapless. A client that has seen `seq` asks for everything after it.

## Agent tools

A tool is `{Def{Name, Description, Risk(read|write|dangerous), Schema}, Authorize, Run}`
and every tool lives in its own file. `Binding{SiteID, ConversationID, RunID, Mode}`
scopes every call.

The guard chain runs in this order and the order matters:

1. `fence` wraps tool output as untrusted data, so a tool result cannot inject
   instructions into the model.
2. `audit` writes the ledger row and emits the stream event.
3. `capResult` truncates oversized JSON before it re-enters the model.
4. `permission` checks the conversation allow-list and calls `Authorize`, denying with
   `UNAUTHORIZED`.

In `confirm` mode a `write` or `dangerous` tool does not execute. It writes a
`PendingAction` row and returns `{status:"confirmationRequired", actionId, summary}`.
Because the pending action is a row and not a goroutine, a confirmation survives a
restart.

Wails services call the use cases directly and typed. They never go through the
registry.
