# Contracts

The rules every boundary obeys: Wails services to the frontend, the tool registry to the
agents. Settled by the Phase 1 spike against Wails v3 `v3.0.0-beta.23`; Phase 11 filled in
the catalogue without changing these shapes.

## Method shape

```go
func (s *XService) Method(ctx context.Context, req ReqDTO) (RespDTO, error)
```

One request struct in, one response struct out, always a context, always an error. A
field added to a struct is a compatible change, an extra argument is not. The generator
strips the leading `context.Context`, so `Method(req)` is what TypeScript sees; name every
parameter, because `_` is generated as `$0`. Every method body is `return s.<field>(ctx,
req)` where the field was built by `wails.Wrap(logger, "<service>.<method>", fn)`: it puts
`ctx.ActorUser` in the context, runs `middleware.Audit` over `middleware.Recover`, and
converts every failure with `wails.Convert`.

## Services

One service per bounded context, all in `internal/transport/wails`, all thin: a method
takes the use case's own request struct, hands it over and returns its response. A method
exists only where a use case exists; each service declares the interface it consumes, so
the composition root binds the two.

| Service | Methods |
|---|---|
| `HealthService` | `Ping` |
| `SitesService` | `Create Update Delete Get List` |
| `GraphService` | `LoadGraph CreateEntity UpdateEntity DeleteEntity GetEntity ListEntities SetAnchors AddEdge ApproveEdge RejectEdge DeleteEdge ListEdges RecomputeScores ProposeFromPages ProposeRelated` |
| `PagesService` | `Create Update Delete Get List Tree MapToEntity Unmap SetCanonical ReplaceLinks` |
| `TemplatesService` | `CreateTemplate UpdateTemplate DeleteTemplate GetTemplate ListTemplates SetOverride DeleteOverride ResolveForPage CreatePolicy UpdatePolicy DeletePolicy GetPolicy ListPolicies GetEffectivePolicy` |
| `RunsService` | `Start Get List ListItems ListEvents GetArtifact Pause Resume Cancel RetryStep` |
| `SyncService` | `SyncSite CheckPlugin SavePluginPackage` |
| `ReportsService` | `SiteOverview PageReport RunReport` |
| `ImportService` | `Inspect Preview Apply Export SaveMapping ListMappings DeleteMapping` |
| `ModelsService` | `ListModels UpsertModel DisableModel GetProfiles SetProfile TestProvider UsageSummary` |
| `AgentService` | `CreateConversation SetMode Send Confirm Cancel ListConversations ListMessages ListPendingActions` |
| `SchedulesService` | `Create Update Delete Get List Enable Disable RunNow` |
| `ToolsService` | `List` |
| `SettingsService` | `Schema Get Set SetProviderKey` |

Ninety-six methods. Where a use case answers with bytes the service writes them to the
path the request names and returns it, because the webview has no filesystem;
`SyncService.SavePluginPackage{path}` is the only such method.

`SettingsService.Schema` renders the `kernel/settings` declarations as
`{key, group, type, default, min?, max?, enum?, nonEmpty?}`, which is what a settings
screen draws itself from. `Get{key}` answers the stored value or the declared default with
`isDefault`; `Set{key, value}` validates against the declaration before it writes, then
re-applies the whole stored map so the running process sees the new value. An undeclared
key is `NOT_FOUND`, a rejected value is `INVALID` and nothing is written. No setting holds
a secret: `SetProviderKey{provider, apiKey}` delegates to `models.SetProviderKey`, which
puts the key in the encrypted store and answers with the provider name alone. It is the
only method that accepts a credential.

## DTOs

- JSON is camelCase everywhere: `siteId`, `nextCursor`, `createdAt`. Never snake_case.
- Timestamps are `kernel/dto.Time`: RFC3339 with a UTC offset, seconds precision, `null`
  when zero.
- Ids are UUID v4 lowercase text.
- A nil slice marshals as `[]`, never as `null` — `paging.Slice[T]` exists for this.
- Request and response structs are declared by the application use cases
  (`internal/application/<context>`) and passed through unchanged; only `HealthService`,
  `ToolsService` and `SettingsService`, which have no use case behind them, declare theirs
  in `internal/transport/wails`. Domain types carry no JSON tags, except
  `template.TemplateSpec` and the `llm` catalog types, whose persisted form is JSON.
- Generics are allowed in exported signatures: `paging.List[T]` generates `List<T>`. A Go
  type with a custom `MarshalJSON` generates as `any`, which is why `Slice<T>` and
  `dto.Time` lose their shape; `frontend/src/lib/paging.ts` restores it with `List<T>` and
  `listOf<T>()`. A bound service type may not itself be generic.

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

which Wails carries as the `cause` of the rejection, so `parseError(thrown)` from
`frontend/src/lib/errors.ts` reads it back as `{code, message, details?, retry?}` and
answers `{code:"INTERNAL", message:"unexpected internal error"}` when the cause is missing
or unrecognised. `details` and `retry` are omitted when empty. `wails.Convert` rebuilds
the error without its internal chain, so the rejection's `message` never carries a driver
string, and `details` is dropped entirely for `INTERNAL`, which keeps a recovered panic's
text out of the UI. A secret, a stack trace and a driver message go to `errors.log`, never
into `details`.

## Pagination

Cursor pagination only; there is no offset anywhere in this codebase. A list request
embeds `kernel/dto.ListRequest{cursor, limit, sort?}`, limit defaulting to 50 and clamping
at 500; the response is `kernel/paging.List[T]` → `{items, nextCursor?, prevCursor?,
hasMore}`. The cursor is an opaque base64url string that records the sort it was issued
for, and replaying one against a different `ORDER BY` is `INVALID`. `cursor` is always the
`nextCursor` of the previous page, `sort.field` is `createdAt` by default and `name` or
`path` where a context offers it, and a client that needs the previous page replays the
cursor it used to reach the current one.

## Long-running work

Any mutation that can exceed a second returns `{runId}` immediately and never blocks;
control is `Get Pause Resume Cancel RetryStep`. Progress is read two ways and both are
required: `RunsService.ListEvents(runId, sinceSeq, limit)` is the durable log, gapless per
run, and the live bus pushes the same records. Live delivery is best-effort — v3
dispatches to the windows that exist at that instant and buffers nothing, so an event
emitted while no window exists is dropped and a page reload discards the listener table.
A client that has seen `seq` asks for everything after it on reconnect; this catch-up is
mandatory, not an optimisation.

## Events

`internal/application/events` owns the names, the payload structs and the envelope;
`frontend/src/generated/events.ts` is rendered from it by `task events` and a Go test
fails when the committed file is stale. The generated module is never edited by hand and
carries no header comment, because this repository forbids comments in TypeScript. The
envelope is

```json
{"type":"step.done","seq":42,"runId":"<uuid>","at":"2026-09-17T10:30:00Z","payload":{}}
```

`runId` is present on run events only, `at` is RFC3339 UTC, and `seq` is per run and
gapless. For application events `seq` is a counter held by the `EventBridge`, so it is
process-wide only because the composition root constructs exactly one bridge; a second
bridge would restart the numbering.

`internal/transport/wails.EventBridge` is the only emitter: `Publish(type, payload)` for
application events, `PublishRun(runId, seq, type, payload)` for run events, both rejecting
an unknown name or a mismatched payload. `application.RegisterEvent` is deliberately
unused.

The names and their payloads are the `EventType` union and the `EventPayloads` map in
[`frontend/src/generated/events.ts`](../frontend/src/generated/events.ts), which is the
registry rendered. Run events are the `run.* item.* step.*` families plus `llm.usage`;
application events are `graph.changed pages.changed templates.changed app.locked
app.unlocked` and the agent family `agent.delta agent.tool.started agent.tool.finished
agent.confirm.requested agent.confirm.resolved agent.done`, whose payloads all carry
`conversationId` because a window may hold more than one conversation. The frontend
subscribes with `on(type, handler)` from `frontend/src/lib/events.ts`, which narrows
`payload` to the declared type. Events only travel Go → JS; every frontend-initiated
action is a bound method call.

## Agent chat

`CreateConversation{siteId?, title?, mode}` opens a conversation in `confirm` or
`autonomous` mode and `SetMode` switches it. `Send{conversationId, text}` returns the
message id at once and the turn runs behind it: `agent.delta` carries the streamed text,
`agent.tool.started` and `agent.tool.finished` the tool calls, `agent.done` the end of the
turn, and `Cancel{conversationId}` stops one in flight. In `confirm` mode a `write` or
`dangerous` tool does not run: it writes a pending action and emits
`agent.confirm.requested{conversationId, confirmationId, tool, args, risk, summary}`, which
`Confirm{actionId, approve}` settles and `agent.confirm.resolved` announces. History is
read with `ListConversations`, `ListMessages` and `ListPendingActions`, all of which
survive a restart; `ToolsService.List` is the capability list the UI shows.

## Bindings generation

`wails3 generate bindings -f '<build flags>' -clean=true -ts -i ./...` writes the
gitignored `frontend/bindings` deterministically and runs from `task bindings` and `task
build`. The TypeScript module name comes from the Go service type name, not from
`ServiceName()`, so the type names in the table above are the catalogue names. `ServiceName`, `ServiceStartup`, `ServiceShutdown` and `ServeHTTP` are
excluded from bindings; every other exported method is public API, because
`//wails:ignore` is a comment and comments are forbidden. A service closes its resources
in `ServiceShutdown`, which runs in reverse registration order. `npm run typecheck` (`tsc --noEmit`) runs inside `task build` and
covers `src` and the generated `bindings`; CI generates the bindings and typechecks
against them as steps of their own, so a broken contract fails before the build does.
`frontend/src/lib/api.ts` re-exports the generated modules under stable names and
`frontend/src/smoke.ts` is the compile-time proof that the contract is usable: it lists
sites, starts a run, follows the events and the catch-up, and sends an agent message.

## Agent tools

A tool is `{Def{Name, Description, Risk(read|write|dangerous), Schema}, Authorize, Run}`,
every tool lives in its own file and `Binding{SiteID, ConversationID, RunID, Mode}` scopes
every call. The guard chain runs in this order and the order matters: `fence` wraps tool output as
untrusted data so a result cannot inject instructions into the model, `audit` writes the
ledger row and emits the stream event, `capResult` truncates oversized JSON before it
re-enters the model, `permission` checks the conversation allow-list and calls `Authorize`,
denying with `UNAUTHORIZED`. The audit middleware records what the tool answered; the fence
wraps the copy the model reads.

In `confirm` mode a `write` or `dangerous` tool does not execute: it writes a
`PendingAction` row and returns `{status:"confirmationRequired", actionId, summary}`, so a
confirmation survives a restart. Wails services call the use cases directly and typed;
they never go through the registry. A tool that works inside one site declares `Authorize` and takes its site from the
binding, so `siteId` leaves the schema the model sees and no conversation reaches another
site through it. `NewTool` derives that schema from the request type with the reflection
rules of `application/llm.Structured`, which do not cover a Go map; the few requests that
carry one take a tool-local argument type instead.
