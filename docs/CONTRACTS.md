# Contracts

The rules every boundary obeys: Wails services to the frontend, and the tool registry to
the agents. Settled by the Phase 1 spike against Wails v3 `v3.0.0-beta.23`; Phase 11
fills in the service catalogue without changing these shapes.

## Method shape

```go
func (s *XService) Method(ctx context.Context, req ReqDTO) (RespDTO, error)
```

One request struct in, one response struct out, always a context, always an error. A
field added to a struct is a compatible change, an extra argument is not. The generator
strips the leading `context.Context`, so `Method(req)` is what TypeScript sees; a
parameter named `_` is generated as `$0`, so name every parameter. Every method body is
`return s.<field>(ctx, req)` where the field was built by `wails.Wrap(logger,
"<service>.<method>", fn)`: it puts `ctx.ActorUser` in the context, runs
`middleware.Audit` over `middleware.Recover`, and converts every failure with
`wails.Convert`.

## DTOs

- JSON is camelCase everywhere: `siteId`, `nextCursor`, `createdAt`. Never snake_case.
- Timestamps are `kernel/dto.Time`: RFC3339 with a UTC offset, seconds precision, `null`
  when zero.
- Ids are UUID v4 lowercase text.
- A nil slice marshals as `[]`, never as `null` — `paging.Slice[T]` exists for this.
- Request and response structs are declared by the application use cases
  (`internal/application/<context>`) with camelCase JSON tags and `kernel/dto.Time`
  timestamps; a Wails service passes them through unchanged and maps only where the wire
  shape must differ. Domain types carry no JSON tags, except `template.TemplateSpec` and
  the `llm` catalog types, whose persisted form is JSON.
- Generics are allowed in exported signatures: `paging.List[T]` generates `List<T>` in
  TypeScript. A Go type with a custom `MarshalJSON` generates as `any`, which is why
  `Slice<T>` and `dto.Time` lose their shape; `frontend/src/lib/paging.ts` restores it
  with `List<T>` and `listOf<T>()`.
- A bound service type may not itself be generic.

## Errors

Every error crossing a boundary is a `*kernel/errors.Error` with one of the frozen
codes: `NOT_FOUND CONFLICT INVALID UNAUTHORIZED RATE_LIMITED BUDGET_EXCEEDED EXTERNAL
INTERNAL CANCELLED NEEDS_HUMAN LOCKED`. A foreign error reports `INTERNAL`.

Services are registered with `application.NewServiceWithOptions(instance,
application.ServiceOptions{MarshalError: wails.MarshalError})`. The application-level
`Options.MarshalError` is ignored by beta.23 and must not be used. The hook returns

```json
{"code":"NOT_FOUND","message":"site not found","details":{"siteId":"s1"},"retry":{"afterMs":2000}}
```

which Wails carries as the `cause` of the rejection, so the frontend reads it with
`parseError(thrown)` from `frontend/src/lib/errors.ts` and gets
`{code, message, details?, retry?}`. `details` and `retry` are omitted when empty, and
`parseError` answers `{code:"INTERNAL", message:"unexpected internal error"}` whenever
the cause is missing or unrecognised. `wails.Convert` rebuilds the error without its
internal chain before it is returned, so the rejection's `message` never carries a driver
string, and `details` is dropped entirely for `INTERNAL`, which keeps a recovered panic's
text out of the UI. `details` never carries a secret, a stack trace or an internal driver
message; those go to `errors.log`.

## Pagination

Cursor pagination only; there is no offset anywhere in this codebase. Request is
`kernel/dto.ListRequest{cursor, limit, sort?}`, limit defaulting to 50 and clamping at
500; response is `kernel/paging.List[T]` → `{items, nextCursor?, prevCursor?, hasMore}`.
The cursor is an opaque base64url string. It records the sort fields and direction it
was issued for, and a cursor replayed against a different `ORDER BY` is rejected as
`INVALID`. Clients treat it as opaque and pass it back unchanged.

A use case's list request embeds `kernel/dto.ListRequest`; `cursor` is always the
`nextCursor` of the previous page (forward paging), `sort.field` is `createdAt` by default
and `name` or `path` where a context offers it. A client that needs the previous page
replays the cursor it used to reach the current one.

## Long-running work

Any mutation that can exceed a second returns `{runId}` immediately. It never blocks.
Control is `Get(runId)`, `Pause(runId)`, `Resume(runId)`, `Cancel(runId)`. Progress is
read two ways and both are required:

- `RunsService.ListEvents(runId, sinceSeq, limit)` — the durable log, gapless per run.
- Live events on the bus — the same records, pushed.

Live delivery is best-effort: v3 dispatches an event to the windows that exist at that
instant and buffers nothing, so an event emitted while no window exists is dropped and a
page reload discards the listener table. A client that has seen `seq` asks for
everything after it on reconnect. This catch-up is mandatory, not an optimisation.

## Events

`internal/application/events` owns the names, the payload structs and the envelope;
`frontend/src/generated/events.ts` is rendered from it by
`go run ./internal/transport/wails/gen` (`task events`) and a Go test fails when the
committed file is stale. The generated module is never edited by hand and carries no
header comment, because this repository forbids comments in TypeScript.

The envelope is

```json
{"type":"step.done","seq":42,"runId":"<uuid>","at":"2026-09-17T10:30:00Z","payload":{}}
```

`runId` is present on run events only. `seq` is per run and gapless for run events. For
application events it is a counter held by the `EventBridge` instance, so it is
process-wide only because the composition root constructs exactly one bridge per process;
a second bridge would restart the numbering. `at` is RFC3339 UTC.

`internal/transport/wails.EventBridge` is the only emitter. `Publish(type, payload)`
serves application events, `PublishRun(runId, seq, type, payload)` serves run events, and
both reject an unknown name or a payload whose type does not match the registry.
`application.RegisterEvent` is deliberately unused: it would need a second hand-written
list no test can police, and its typings land in the gitignored `frontend/bindings`.

Run events: `run.queued run.started run.paused run.resumed run.cancelled run.completed
run.failed run.budget_exceeded item.started item.done item.failed item.needs_human
step.started step.done step.failed step.retrying llm.usage`.

Application events: `graph.changed{siteId}`, `pages.changed{siteId}`,
`templates.changed`, `agent.delta`, `agent.tool.started`, `agent.tool.finished`,
`agent.confirm.requested`, `agent.confirm.resolved`, `agent.done`, `app.locked`,
`app.unlocked`. Every agent payload carries `conversationId`, because a window may hold
more than one conversation at a time.

The frontend subscribes with `on(type, handler)` from `frontend/src/lib/events.ts`,
which narrows `payload` to the type the registry declares. Events only travel Go → JS;
every frontend-initiated action is a bound method call.

## Bindings generation

`wails3 generate bindings -f '<build flags>' -clean=true -ts -i ./...` writes
`frontend/bindings`, which is gitignored and regenerated by `task build` and `task
bindings`. Output is deterministic. The TypeScript module name comes from the Go service
type name, not from `ServiceName()`, so the Go type names are the catalogue names:
`SitesService GraphService PagesService TemplatesService RunsService AgentService
ImportService SchedulesService ReportsService SettingsService`. `ServiceName`,
`ServiceStartup`, `ServiceShutdown` and `ServeHTTP` are excluded from bindings; every
other exported method is public API, because `//wails:ignore` is a comment and comments
are forbidden. A service closes its resources in `ServiceShutdown`, which runs in reverse
registration order. `npm run typecheck` (`tsc --noEmit`) runs inside `task build` and
covers `src` and the generated `bindings`.

## Agent tools

A tool is `{Def{Name, Description, Risk(read|write|dangerous), Schema}, Authorize, Run}`
and every tool lives in its own file. `Binding{SiteID, ConversationID, RunID, Mode}`
scopes every call.

The guard chain runs in this order and the order matters: `fence` wraps tool output as
untrusted data so a result cannot inject instructions into the model, `audit` writes the
ledger row and emits the stream event, `capResult` truncates oversized JSON before it
re-enters the model, and `permission` checks the conversation allow-list and calls
`Authorize`, denying with `UNAUTHORIZED`.

In `confirm` mode a `write` or `dangerous` tool does not execute. It writes a
`PendingAction` row and returns `{status:"confirmationRequired", actionId, summary}`, so
a confirmation survives a restart. Wails services call the use cases directly and typed;
they never go through the registry.

A tool that works inside one site declares `Authorize` and takes its site from the
binding, so `siteId` is removed from the schema the model sees and no conversation can
reach another site through it. `NewTool` derives that schema from the request type with
the reflection rules of `application/llm.Structured`, which do not cover a Go map: the
handful of requests that carry one take a tool-local argument type instead. The audit
middleware records what the tool answered; the fence wraps the copy the model reads.
