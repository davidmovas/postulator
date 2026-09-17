# Phase 1B — Wails v3 Contracts and Transport Scaffolding Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. This repository forbids sub-agents (spec §10), so the subagent-driven variant does not apply.

**Goal:** Settle the Wails v3 wire contract — error format, event envelope, generics, long-running protocol, bindings generation — and land the Go event registry, the transport error/event/service scaffolding, the TypeScript generator and the frontend contract library that implement it.

**Architecture:** One Go registry in `internal/application/events` owns every event name, its payload struct and the `Envelope`. `internal/transport/wails` converts `*kernel/errors.Error` into the transport form that Wails' per-service `MarshalError` hook hands to JavaScript, wraps every service method with panic recovery, audit logging and actor context, and publishes envelopes through a consumer-declared `Emitter` interface so no Wails runtime is needed in tests. A `go run`-able generator renders the registry into a committed `frontend/src/generated/events.ts`, and a Go test re-renders it into a temp directory to keep the two in sync.

**Tech Stack:** Go 1.27 (CGO off), Wails v3 `v3.0.0-beta.23`, zap `v1.27.0`, Vite 8 + TypeScript 5.9, `@wailsio/runtime` `3.0.0-beta.23`, Task `v3.53.1`, golangci-lint `v2.13.2`.

**Spec:** `docs/superpowers/specs/2026-09-17-postulator-v2-design.md` (§4 kernel, §7 run-event list, §8 contracts, §10 policy, §15 reference map) and the Phase 1 row of `docs/superpowers/plans/2026-09-17-postulator-v2-roadmap.md`.

## Global Constraints

- **No comments in Go, TypeScript, YAML or PHP** — not inline, not godoc, not package docs. This includes generated TypeScript.
- **No stubs, no TODOs, no placeholders.** If something cannot be done, stop and say so.
- **Interfaces are declared by the consumer**, never by the implementation and never in advance.
- **TDD.** Failing test first, run it, implement, run it, commit. Table tests with named cases. Gates: `internal/domain` + `internal/application` ≥ 80%, module ≥ 70%, `internal/kernel` ≥ 90%.
- **Cursor pagination only.** No offset, anywhere.
- **JSON is camelCase**, timestamps are RFC3339 UTC (`kernel/dto.Time`), ids are UUID v4 lowercase text.
- **Errors crossing a boundary are `*kernel/errors.Error`** with a frozen code: `NOT_FOUND CONFLICT INVALID UNAUTHORIZED RATE_LIMITED BUDGET_EXCEEDED EXTERNAL INTERNAL CANCELLED NEEDS_HUMAN LOCKED`.
- **Secrets never reach a log, an artifact or a plain column.** `details` never carries a secret, a stack trace or an internal driver message.
- **No new `.md` files** beyond `CLAUDE.md` and the set in spec §10.
- **Dependency rule**, enforced by `internal/app/deps_test.go`: `application` imports only `domain` and `kernel`; only `internal/app` imports `internal/transport`; only `kernel/paging` imports squirrel.
- **Commits** are conventional (`<type>(<scope>): <subject>`), on `rewrite/v2`, never amended, rebased or force-pushed, and every commit ends with the trailer `Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>`.
- Pinned: Go `1.27`, Wails v3 `v3.0.0-beta.23`, golangci-lint `v2.13.2`, Task `v3.53.1`, Node `22`.

---

## Spike findings

The spike lives outside the repository (scratch module `spike`, vendoring copies of `internal/kernel/{errors,paging,dto,clock,ctx,middleware}`), requires `github.com/wailsapp/wails/v3 v3.0.0-beta.23`, and was driven with `go run . call|events|envelope`, `wails3 generate bindings -clean=true -ts -i -d ./bindings ./...` and `npx tsc --noEmit` against the repository's own `frontend/tsconfig.json` and `node_modules`. Nothing from it is committed.

### Q1 — Structured errors across the binding call path

**Answer.** v3 supports structured errors natively through an error-marshalling hook, so no JSON-in-a-string hack is needed. `application.CallError{Message, Cause, Kind}` is marshalled straight into the HTTP error body (`transport_http.go:390`, `json.Marshal(cerr)` with `Content-Type: application/json`), and `@wailsio/runtime` rebuilds it as `new RuntimeError(json.message)` with `err.cause = json.cause` (`dist/runtime.js:134-146`). `Cause` is filled by the hook. **The application-level `Options.MarshalError` is dead in beta.23** — `Bindings.Add` overwrites each method's marshaller with `wrapErrorMarshaler(service.options.MarshalError, defaultMarshalError)` (`bindings.go:140`), so only `ServiceOptions.MarshalError` via `application.NewServiceWithOptions` takes effect. Verified by calling `BoundMethod.Call` directly and marshalling the resulting `*CallError` — the exact bytes the webview receives:

```
kernel error, default marshaller  {"message":"thing not found: unexpected EOF","cause":{"Details":{"id":"abc"},"Retry":{"After":3000000000},"Code":"NOT_FOUND","Message":"thing not found"},"kind":"RuntimeError"}
kernel error, app-level hook      {"message":"thing not found: unexpected EOF","cause":{"Details":...,"Code":"NOT_FOUND",...},"kind":"RuntimeError"}   (hook ignored)
kernel error, service-level hook  {"message":"thing not found: unexpected EOF","cause":{"code":"NOT_FOUND","message":"thing not found","details":{"id":"abc"},"retry":{"afterMs":3000}},"kind":"RuntimeError"}
sanitised error, service hook     {"message":"thing not found","cause":{"code":"NOT_FOUND","message":"thing not found","details":{"id":"abc"},"retry":{"afterMs":3000}},"kind":"RuntimeError"}
foreign error, service hook       {"message":"file does not exist","cause":{"code":"INTERNAL","message":"unexpected internal error"},"kind":"RuntimeError"}
panic (no hook reached)           {"message":"main.ThingsService.Boom: panic: boom","kind":"RuntimeError"}
```

Two consequences. The default marshaller emits PascalCase keys and `Retry.After` in nanoseconds, breaking the camelCase rule — a custom hook is mandatory. And `CallError.Message` is `errors.Join(...).Error()`, which leaks the wrapped internal cause; the service wrapper therefore returns a **sanitised** `*kernel/errors.Error` carrying no internal chain (row 4 above). The panic row shows `cause` can be absent, so the frontend helper must fall back to `INTERNAL`.

### Q2 — Events Go→JS

**Answer.** `app.Event.Emit(name string, data ...any) bool` (`event_manager.go:31`); `*application.EventManager` satisfies a one-method consumer interface `Emit(string, ...any) bool`, which is how the bridge stays testable. JS listens with `Events.On(name, ev => ev.data)`. The generator *does* emit event typings: `application.RegisterEvent[Data](name)` calls are found by static analysis and rendered as a module augmentation of `Events.CustomEvents` — generic instantiations included:

```ts
declare module "@wailsio/runtime" {
    namespace Events {
        interface CustomEvents {
            "app.locked": void;
            "graph.changed": main$0.Envelope;
            "pages.changed": main$0.TypedEnvelope<main$0.GraphChanged>;
        }
    }
}
```

`RegisterEvent` also validates payload types at emit time (`data of type main.GraphChanged for event 'graph.changed' does not match registered data type main.Envelope`, event cancelled). **We do not use it.** It would require a second hand-written list of 26 `RegisterEvent[Envelope[X]]` calls that no registry test can police, it panics on duplicate registration (hostile to tests), and its output lands in `frontend/bindings/`, which is gitignored and therefore not diffable in CI. The envelope is enforced instead by the bridge being the only emitter (`Publish`/`PublishRun` build the `Envelope` themselves and reject a payload whose `reflect.TypeOf` differs from the registered one) and by our own generator owning `frontend/src/generated/events.ts`. JS→Go events exist (`Events.Emit` from the runtime) and are **not needed**: every frontend-initiated action is a bound method call.

### Q3 — Generics in bound signatures

**Answer.** Generics survive cleanly; no concrete `XxxList` DTOs are needed. `ListGeneric(ctx, dto.ListRequest) (paging.List[ThingDTO], error)` generated:

```ts
export function ListGeneric($0: dto$0.ListRequest): $CancellablePromise<paging$0.List<$models.ThingDTO>> { ... }
// kernel/paging/models.ts
export interface List<T> { "nextCursor"?: Cursor; "prevCursor"?: Cursor; "items": Slice<T>; "hasMore": boolean; }
export type Slice<T> = any;
// kernel/dto/models.ts
export type Time = any;
```

The embedded `Cursors` struct is flattened correctly and `TypedEnvelope<P>` also rendered as a generic interface. The one caveat: a type with an explicit `MarshalJSON` is rendered `any` (`collect/model.go:222-227`), which is why `paging.Slice[T]` and `dto.Time` degrade — so `items` loses its element type. That is handled by hand-written `frontend/src/lib/paging.ts` (`List<T>` with `items: T[]`) plus a `listOf<T>()` narrowing helper, not by changing the Go signatures. A param named `_` is generated as `$0`, so service request parameters must be named.

### Q4 — Long-running protocol

**Answer.** `Start → {runId} → Get/ListEvents(sinceSeq)/Pause/Resume/Cancel` needs nothing special from v3: they are ordinary bound methods and the events are ordinary custom events. Delivery is best-effort and unbuffered. `EventIPCTransport.DispatchWailsEvent` snapshots `app.windows` and dispatches to each; with no window the loop body never runs and the event is dropped silently. `WebviewWindow.DispatchWailsEvent` returns early when `w.impl == nil || w.isDestroyed()`, and `enqueueEventJS` returns `false` once the queue is closed. Confirmed by running `Emit` on an app with zero windows: `emit typed envelope cancelled=false` — reported as success, delivered nowhere, with the Go-side listener still receiving `{"name":"graph.changed","data":{"type":"graph.changed","seq":1,"at":"2026-09-17T10:30:00Z","payload":{"siteId":"s1"}}}`. A page reload also discards the JS listener table. **`ListEvents(runId, sinceSeq, limit)` catch-up is therefore mandatory, not an optimisation**, and `seq` must be gapless per run.

### Q5 — Bindings generation

**Answer.** `wails3 generate bindings -f '<build flags>' -clean=true -ts -i ./...` (already in `build/Taskfile.yml`). Output directory defaults to `frontend/bindings` (`internal/flags/bindings.go:12`); `-ts` emits TypeScript, `-i` emits interfaces rather than classes, `-clean` wipes the directory first, `-time-type` defaults to `string`. Output is **deterministic**: two consecutive runs over the same tree produced byte-identical trees (`diff -r` clean). A leading `context.Context` parameter is stripped from the generated signature (`Boom(): $CancellablePromise<string>`), and a `ctx`-less method is generated identically, so the two shapes are indistinguishable from TypeScript. Methods are module-level functions dispatched by FNV id (`$Call.ByID(3683767548, req)`). `frontend/bindings/` stays gitignored and regenerated by `task build`; the only committed generated artefact is `frontend/src/generated/events.ts`, which our own generator owns and a Go test keeps in sync.

### Q6 — Service registration

**Answer.** `application.NewService[T any](*T)` or `application.NewServiceWithOptions[T any](*T, ServiceOptions)` with `ServiceOptions{Name, Route, MarshalError}`. Services are discovered by static analysis of the `NewService` instantiation, and the analyser follows type parameters (`analyse.go:165-195`), so a generic project-local helper works — proven: with `Services()` returning `bind(&ThingsService{})` where `func bind[T any](instance *T) application.Service`, the generator still reported `1 Service, 8 Methods`. The optional lifecycle hooks are `ServiceName() string`, `ServiceStartup(ctx context.Context, options ServiceOptions) error` and `ServiceShutdown() error`; all three are excluded from bindings by name (`bindings.go:196-201`), as is `ServeHTTP`. `ServiceStartup`'s context is cancelled right before shutdown and `ServiceShutdown` runs in reverse registration order — that is where Phase 2 closes the SQLite store. `ServiceName()` affects logging only: **the TypeScript module name comes from the Go type name** (`ThingsService` → `bindings/<pkg>/thingsservice.ts`, exported as `ThingsService`), so the Go service type names must be exactly the catalogue names from spec §8. Two further constraints: a bound type may not be generic, and `//wails:ignore` / `//wails:internal` are comments, which this repository forbids — every exported method on a bound service is public API.

---

## File structure

**Create**

- `internal/application/events/type.go` — `Type` and the frozen name constants.
- `internal/application/events/payload.go` — one payload struct per event.
- `internal/application/events/envelope.go` — `Envelope`.
- `internal/application/events/registry.go` — `Entry`, `Registry`, `NewRegistry`, `Entries`, `Lookup`.
- `internal/application/events/registry_test.go`
- `internal/transport/wails/errors.go` — `Error`, `Retry`, `Convert`, `MarshalError`.
- `internal/transport/wails/errors_test.go`
- `internal/transport/wails/service.go` — `Wrap`, `bind`, `Services`.
- `internal/transport/wails/health.go` — `BuildInfo`, `PingRequest`, `HealthService`.
- `internal/transport/wails/service_test.go`
- `internal/transport/wails/eventbridge.go` — `Emitter`, `EventBridge`, `Publish`, `PublishRun`.
- `internal/transport/wails/eventbridge_test.go`
- `internal/transport/wails/gen/main.go` — `write`, flag parsing.
- `internal/transport/wails/gen/render.go` — `render`, `renderInterface`, `jsonName`, `tsType`.
- `internal/transport/wails/gen/main_test.go`
- `internal/app/wiring.go` — `Logger`, `Services`.
- `internal/app/wiring_test.go`
- `frontend/src/generated/events.ts` — generated, committed.
- `frontend/src/lib/errors.ts`, `frontend/src/lib/events.ts`, `frontend/src/lib/paging.ts`

**Modify**

- `internal/app/version.go` — keep the ldflags stamps, drop `BuildInfo`/`Build`.
- `cmd/postulator/main.go` — build the logger, take services from `app.Services`.
- `frontend/src/main.ts` — call `HealthService.Ping({})`, render failures through `parseError`.
- `frontend/package.json` — add the `typecheck` script.
- `build/Taskfile.yml` — run `npm run typecheck` inside `build:frontend`.
- `Taskfile.yml` — add the `events` task.
- `docs/CONTRACTS.md` — rewritten as the final contract document.
- `docs/STATUS.md`, `CLAUDE.md` — decisions and standing rulings.

**Delete**

- `internal/app/health.go`, `internal/app/health_test.go` — the health service moves to the transport layer, where spec §8 puts services and where DTOs are declared.

---

### Task 1: Event registry and envelope

**Files:**
- Create: `internal/application/events/type.go`, `internal/application/events/payload.go`, `internal/application/events/envelope.go`, `internal/application/events/registry.go`
- Test: `internal/application/events/registry_test.go`

**Interfaces:**
- Consumes: `github.com/davidmovas/postulator/internal/kernel/dto` (`dto.Time`, `dto.NewTime`).
- Produces:
  - `type Type string` with the constants `GraphChanged PagesChanged TemplatesChanged AgentDelta AgentToolStarted AgentToolFinished AgentConfirmRequested AppLocked AppUnlocked RunQueued RunStarted RunPaused RunResumed RunCancelled RunCompleted RunFailed RunBudgetExceeded ItemStarted ItemDone ItemFailed ItemNeedsHuman StepStarted StepDone StepFailed StepRetrying LLMUsage`
  - `type Envelope struct { Type Type; Seq int64; RunID *string; At dto.Time; Payload any }`
  - `type Entry struct { Type Type; Payload any; Run bool }`
  - `type Registry struct{ ... }`, `func NewRegistry() Registry`, `func (Registry) Entries() []Entry`, `func (Registry) Lookup(Type) (Entry, bool)`
  - one `XxxPayload` struct per constant, listed in `payload.go` below.

- [ ] **Step 1: Write the failing test**

Create `internal/application/events/registry_test.go`:

```go
package events_test

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

func declaredTypes(t *testing.T) []string {
	t.Helper()

	file, err := parser.ParseFile(token.NewFileSet(), "type.go", nil, 0)
	if err != nil {
		t.Fatalf("parse type.go: %v", err)
	}

	var names []string
	for _, decl := range file.Decls {
		group, ok := decl.(*ast.GenDecl)
		if !ok || group.Tok != token.CONST {
			continue
		}
		for _, spec := range group.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			ident, ok := value.Type.(*ast.Ident)
			if !ok || ident.Name != "Type" {
				continue
			}
			for _, name := range value.Names {
				names = append(names, name.Name)
			}
		}
	}
	return names
}

func TestEveryDeclaredTypeIsRegistered(t *testing.T) {
	t.Parallel()

	declared := declaredTypes(t)
	if len(declared) == 0 {
		t.Fatal("no Type constants were found in type.go")
	}

	registry := events.NewRegistry()
	if got, want := len(registry.Entries()), len(declared); got != want {
		t.Fatalf("registry holds %d entries, type.go declares %d constants", got, want)
	}
}

func TestEveryRegisteredEntryHasADistinctStructPayload(t *testing.T) {
	t.Parallel()

	seen := make(map[reflect.Type]events.Type)
	for _, entry := range events.NewRegistry().Entries() {
		payload := reflect.TypeOf(entry.Payload)
		if payload == nil || payload.Kind() != reflect.Struct {
			t.Fatalf("%s carries a non-struct payload", entry.Type)
		}
		if other, ok := seen[payload]; ok {
			t.Fatalf("%s and %s share the payload type %s", entry.Type, other, payload)
		}
		seen[payload] = entry.Type
	}
}

func TestEveryPayloadFieldCarriesACamelCaseJSONTag(t *testing.T) {
	t.Parallel()

	camelCase := regexp.MustCompile(`^[a-z][A-Za-z0-9]*$`)
	for _, entry := range events.NewRegistry().Entries() {
		payload := reflect.TypeOf(entry.Payload)
		for index := range payload.NumField() {
			field := payload.Field(index)
			tag := field.Tag.Get("json")
			if !camelCase.MatchString(tag) {
				t.Errorf("%s.%s has the json tag %q, want camelCase", entry.Type, field.Name, tag)
			}
		}
	}
}

func TestLookupReportsRunEvents(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input events.Type
		found bool
		run   bool
	}{
		{name: "graph change is not a run event", input: events.GraphChanged, found: true},
		{name: "step retry is a run event", input: events.StepRetrying, found: true, run: true},
		{name: "unknown name", input: events.Type("nope")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			entry, ok := events.NewRegistry().Lookup(tc.input)
			if ok != tc.found {
				t.Fatalf("Lookup(%q) found = %v, want %v", tc.input, ok, tc.found)
			}
			if entry.Run != tc.run {
				t.Fatalf("Lookup(%q).Run = %v, want %v", tc.input, entry.Run, tc.run)
			}
		})
	}
}

func TestEnvelopeMarshalsTheAgreedShape(t *testing.T) {
	t.Parallel()

	runID := "6f3b2a11-0c9d-4e7a-8b25-1f4c6d7e8a90"
	at := dto.NewTime(time.Date(2026, 9, 17, 10, 30, 0, 0, time.UTC))

	cases := []struct {
		name  string
		input events.Envelope
		want  string
	}{
		{
			name:  "application event omits the run id",
			input: events.Envelope{Type: events.GraphChanged, Seq: 4, At: at, Payload: events.GraphChangedPayload{SiteID: "s1"}},
			want:  `{"type":"graph.changed","seq":4,"at":"2026-09-17T10:30:00Z","payload":{"siteId":"s1"}}`,
		},
		{
			name:  "run event carries the run id",
			input: events.Envelope{Type: events.RunStarted, Seq: 1, RunID: &runID, At: at, Payload: events.RunStartedPayload{RunID: runID}},
			want:  `{"type":"run.started","seq":1,"runId":"6f3b2a11-0c9d-4e7a-8b25-1f4c6d7e8a90","at":"2026-09-17T10:30:00Z","payload":{"runId":"6f3b2a11-0c9d-4e7a-8b25-1f4c6d7e8a90"}}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			encoded, err := json.Marshal(tc.input)
			if err != nil {
				t.Fatalf("Marshal() error: %v", err)
			}
			if string(encoded) != tc.want {
				t.Fatalf("Marshal() = %s, want %s", encoded, tc.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/application/events/...`
Expected: FAIL — the package does not build (`no required module provides package .../internal/application/events`).

- [ ] **Step 3: Write the implementation**

`internal/application/events/type.go`:

```go
package events

type Type string

const (
	GraphChanged          Type = "graph.changed"
	PagesChanged          Type = "pages.changed"
	TemplatesChanged      Type = "templates.changed"
	AgentDelta            Type = "agent.delta"
	AgentToolStarted      Type = "agent.tool.started"
	AgentToolFinished     Type = "agent.tool.finished"
	AgentConfirmRequested Type = "agent.confirm.requested"
	AppLocked             Type = "app.locked"
	AppUnlocked           Type = "app.unlocked"

	RunQueued         Type = "run.queued"
	RunStarted        Type = "run.started"
	RunPaused         Type = "run.paused"
	RunResumed        Type = "run.resumed"
	RunCancelled      Type = "run.cancelled"
	RunCompleted      Type = "run.completed"
	RunFailed         Type = "run.failed"
	RunBudgetExceeded Type = "run.budget_exceeded"
	ItemStarted       Type = "item.started"
	ItemDone          Type = "item.done"
	ItemFailed        Type = "item.failed"
	ItemNeedsHuman    Type = "item.needs_human"
	StepStarted       Type = "step.started"
	StepDone          Type = "step.done"
	StepFailed        Type = "step.failed"
	StepRetrying      Type = "step.retrying"
	LLMUsage          Type = "llm.usage"
)
```

`internal/application/events/payload.go`:

```go
package events

import "encoding/json"

type GraphChangedPayload struct {
	SiteID string `json:"siteId"`
}

type PagesChangedPayload struct {
	SiteID string `json:"siteId"`
}

type TemplatesChangedPayload struct{}

type AgentDeltaPayload struct {
	ConversationID string `json:"conversationId"`
	MessageID      string `json:"messageId"`
	Seq            int64  `json:"seq"`
	Text           string `json:"text"`
}

type AgentToolStartedPayload struct {
	ConversationID string          `json:"conversationId"`
	CallID         string          `json:"callId"`
	Tool           string          `json:"tool"`
	Args           json.RawMessage `json:"args"`
}

type AgentToolFinishedPayload struct {
	ConversationID string          `json:"conversationId"`
	CallID         string          `json:"callId"`
	Tool           string          `json:"tool"`
	Result         json.RawMessage `json:"result"`
}

type AgentConfirmRequestedPayload struct {
	ConfirmationID string          `json:"confirmationId"`
	Tool           string          `json:"tool"`
	Args           json.RawMessage `json:"args"`
	Risk           string          `json:"risk"`
}

type AppLockedPayload struct{}

type AppUnlockedPayload struct{}

type RunQueuedPayload struct {
	RunID string `json:"runId"`
	Kind  string `json:"kind"`
	Items int    `json:"items"`
}

type RunStartedPayload struct {
	RunID string `json:"runId"`
}

type RunPausedPayload struct {
	RunID  string `json:"runId"`
	Reason string `json:"reason"`
}

type RunResumedPayload struct {
	RunID string `json:"runId"`
}

type RunCancelledPayload struct {
	RunID string `json:"runId"`
}

type RunCompletedPayload struct {
	RunID     string `json:"runId"`
	Succeeded int    `json:"succeeded"`
	Failed    int    `json:"failed"`
}

type RunFailedPayload struct {
	RunID   string `json:"runId"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type RunBudgetExceededPayload struct {
	RunID     string  `json:"runId"`
	SpentUSD  float64 `json:"spentUsd"`
	BudgetUSD float64 `json:"budgetUsd"`
}

type ItemStartedPayload struct {
	RunID  string `json:"runId"`
	ItemID string `json:"itemId"`
}

type ItemDonePayload struct {
	RunID  string `json:"runId"`
	ItemID string `json:"itemId"`
}

type ItemFailedPayload struct {
	RunID   string `json:"runId"`
	ItemID  string `json:"itemId"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ItemNeedsHumanPayload struct {
	RunID  string `json:"runId"`
	ItemID string `json:"itemId"`
	Reason string `json:"reason"`
}

type StepStartedPayload struct {
	RunID  string `json:"runId"`
	ItemID string `json:"itemId"`
	Step   string `json:"step"`
}

type StepDonePayload struct {
	RunID      string `json:"runId"`
	ItemID     string `json:"itemId"`
	Step       string `json:"step"`
	DurationMs int64  `json:"durationMs"`
}

type StepFailedPayload struct {
	RunID   string `json:"runId"`
	ItemID  string `json:"itemId"`
	Step    string `json:"step"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type StepRetryingPayload struct {
	RunID   string `json:"runId"`
	ItemID  string `json:"itemId"`
	Step    string `json:"step"`
	Attempt int    `json:"attempt"`
	AfterMs int64  `json:"afterMs"`
}

type LLMUsagePayload struct {
	RunID            string  `json:"runId"`
	ItemID           string  `json:"itemId"`
	Provider         string  `json:"provider"`
	Model            string  `json:"model"`
	PromptTokens     int     `json:"promptTokens"`
	CompletionTokens int     `json:"completionTokens"`
	USD              float64 `json:"usd"`
}
```

`internal/application/events/envelope.go`:

```go
package events

import "github.com/davidmovas/postulator/internal/kernel/dto"

type Envelope struct {
	Type    Type     `json:"type"`
	Seq     int64    `json:"seq"`
	RunID   *string  `json:"runId,omitempty"`
	At      dto.Time `json:"at"`
	Payload any      `json:"payload"`
}
```

`internal/application/events/registry.go`:

```go
package events

import (
	"slices"
	"strings"
)

type Entry struct {
	Type    Type
	Payload any
	Run     bool
}

type Registry struct {
	entries []Entry
	byType  map[Type]Entry
}

func NewRegistry() Registry {
	entries := []Entry{
		{Type: GraphChanged, Payload: GraphChangedPayload{}},
		{Type: PagesChanged, Payload: PagesChangedPayload{}},
		{Type: TemplatesChanged, Payload: TemplatesChangedPayload{}},
		{Type: AgentDelta, Payload: AgentDeltaPayload{}},
		{Type: AgentToolStarted, Payload: AgentToolStartedPayload{}},
		{Type: AgentToolFinished, Payload: AgentToolFinishedPayload{}},
		{Type: AgentConfirmRequested, Payload: AgentConfirmRequestedPayload{}},
		{Type: AppLocked, Payload: AppLockedPayload{}},
		{Type: AppUnlocked, Payload: AppUnlockedPayload{}},
		{Type: RunQueued, Payload: RunQueuedPayload{}, Run: true},
		{Type: RunStarted, Payload: RunStartedPayload{}, Run: true},
		{Type: RunPaused, Payload: RunPausedPayload{}, Run: true},
		{Type: RunResumed, Payload: RunResumedPayload{}, Run: true},
		{Type: RunCancelled, Payload: RunCancelledPayload{}, Run: true},
		{Type: RunCompleted, Payload: RunCompletedPayload{}, Run: true},
		{Type: RunFailed, Payload: RunFailedPayload{}, Run: true},
		{Type: RunBudgetExceeded, Payload: RunBudgetExceededPayload{}, Run: true},
		{Type: ItemStarted, Payload: ItemStartedPayload{}, Run: true},
		{Type: ItemDone, Payload: ItemDonePayload{}, Run: true},
		{Type: ItemFailed, Payload: ItemFailedPayload{}, Run: true},
		{Type: ItemNeedsHuman, Payload: ItemNeedsHumanPayload{}, Run: true},
		{Type: StepStarted, Payload: StepStartedPayload{}, Run: true},
		{Type: StepDone, Payload: StepDonePayload{}, Run: true},
		{Type: StepFailed, Payload: StepFailedPayload{}, Run: true},
		{Type: StepRetrying, Payload: StepRetryingPayload{}, Run: true},
		{Type: LLMUsage, Payload: LLMUsagePayload{}, Run: true},
	}

	slices.SortFunc(entries, func(a, b Entry) int {
		return strings.Compare(string(a.Type), string(b.Type))
	})

	byType := make(map[Type]Entry, len(entries))
	for _, entry := range entries {
		byType[entry.Type] = entry
	}
	return Registry{entries: entries, byType: byType}
}

func (r Registry) Entries() []Entry {
	return slices.Clone(r.entries)
}

func (r Registry) Lookup(eventType Type) (Entry, bool) {
	entry, ok := r.byType[eventType]
	return entry, ok
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test -race ./internal/application/events/...`
Expected: PASS (`ok github.com/davidmovas/postulator/internal/application/events`).

- [ ] **Step 5: Check coverage and the dependency rule**

Run: `go test -race -cover ./internal/application/events/... && go test ./internal/app/...`
Expected: coverage ≥ 80% for the events package; `deps_test.go` green — `internal/application` now exists, so its rule starts being enforced.

- [ ] **Step 6: Commit**

```bash
git add internal/application/events
git commit -m "$(cat <<'EOF'
feat(events): event registry, payloads and envelope

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: Transport error form

**Files:**
- Create: `internal/transport/wails/errors.go`
- Test: `internal/transport/wails/errors_test.go`

**Interfaces:**
- Consumes: `kernel/errors` (`Error`, `Code`, `New`, `Wrap`, `CodeOf`, `WithDetail`, `WithRetry`).
- Produces:

```go
type Retry struct {
	AfterMs int64 `json:"afterMs"`
}

type Error struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
	Retry   *Retry         `json:"retry,omitempty"`
}

func Convert(err error) error
func MarshalError(err error) []byte
```

  `Convert` returns a sanitised `*kernel/errors.Error` with no internal chain, and `nil`
  for `nil`. `MarshalError` is the value handed to `application.ServiceOptions.MarshalError`.

- [ ] **Step 1: Write the failing test**

Create `internal/transport/wails/errors_test.go`:

```go
package wails_test

import (
	"encoding/json"
	stderrors "errors"
	"io"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/transport/wails"
)

func TestMarshalErrorProducesTheTransportShape(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input error
		want  string
	}{
		{
			name:  "kernel error with details and retry",
			input: errors.New(errors.RateLimited, "provider is throttling").WithDetail("provider", "openai").WithRetry(1500 * time.Millisecond),
			want:  `{"code":"RATE_LIMITED","message":"provider is throttling","details":{"provider":"openai"},"retry":{"afterMs":1500}}`,
		},
		{
			name:  "kernel error without details",
			input: errors.New(errors.NotFound, "site not found"),
			want:  `{"code":"NOT_FOUND","message":"site not found"}`,
		},
		{
			name:  "internal kernel error drops its details",
			input: errors.New(errors.Internal, "handler panicked").WithDetail("panic", "runtime error: index out of range"),
			want:  `{"code":"INTERNAL","message":"handler panicked"}`,
		},
		{
			name:  "wrapped driver error keeps the authored message only",
			input: errors.Wrap(io.ErrUnexpectedEOF, errors.External, "wordpress rejected the request"),
			want:  `{"code":"EXTERNAL","message":"wordpress rejected the request"}`,
		},
		{
			name:  "foreign error becomes internal",
			input: io.ErrUnexpectedEOF,
			want:  `{"code":"INTERNAL","message":"unexpected internal error"}`,
		},
		{
			name:  "unmarshalable detail falls back",
			input: errors.New(errors.Invalid, "bad input").WithDetail("channel", make(chan int)),
			want:  `{"code":"INTERNAL","message":"unexpected internal error"}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := string(wails.MarshalError(tc.input)); got != tc.want {
				t.Fatalf("MarshalError() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestConvertStripsTheInternalChain(t *testing.T) {
	t.Parallel()

	original := errors.New(errors.NotFound, "site not found").
		WithDetail("siteId", "s1").
		WithRetry(2 * time.Second).
		WithInternal(io.ErrUnexpectedEOF)

	converted := wails.Convert(original)
	if converted.Error() != "site not found" {
		t.Fatalf("Convert().Error() = %q, want %q", converted.Error(), "site not found")
	}
	if stderrors.Is(converted, io.ErrUnexpectedEOF) {
		t.Fatal("Convert() kept the internal cause, which would leak it into CallError.Message")
	}
	if errors.CodeOf(converted) != errors.NotFound {
		t.Fatalf("CodeOf(Convert()) = %q, want %q", errors.CodeOf(converted), errors.NotFound)
	}

	const want = `{"code":"NOT_FOUND","message":"site not found","details":{"siteId":"s1"},"retry":{"afterMs":2000}}`
	if got := string(wails.MarshalError(converted)); got != want {
		t.Fatalf("MarshalError(Convert()) = %s, want %s", got, want)
	}
}

func TestConvertReturnsNilForNil(t *testing.T) {
	t.Parallel()

	if got := wails.Convert(nil); got != nil {
		t.Fatalf("Convert(nil) = %v, want nil", got)
	}
}

func TestTheFrontendReceivesTheDocumentedBody(t *testing.T) {
	t.Parallel()

	converted := wails.Convert(errors.New(errors.Conflict, "slug already exists").
		WithDetail("slug", "about-us").
		WithInternal(io.ErrUnexpectedEOF))

	body, err := json.Marshal(&application.CallError{
		Message: converted.Error(),
		Cause:   json.RawMessage(wails.MarshalError(converted)),
		Kind:    application.RuntimeError,
	})
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	const want = `{"message":"slug already exists","cause":{"code":"CONFLICT","message":"slug already exists","details":{"slug":"about-us"}},"kind":"RuntimeError"}`
	if string(body) != want {
		t.Fatalf("call error body = %s, want %s", body, want)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/transport/wails/...`
Expected: FAIL — the package does not exist.

- [ ] **Step 3: Write the implementation**

Create `internal/transport/wails/errors.go`:

```go
package wails

import (
	"encoding/json"
	stderrors "errors"
	"maps"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const internalMessage = "unexpected internal error"

const fallbackBody = `{"code":"INTERNAL","message":"unexpected internal error"}`

type Retry struct {
	AfterMs int64 `json:"afterMs"`
}

type Error struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
	Retry   *Retry         `json:"retry,omitempty"`
}

func describe(err error) Error {
	var kernel *errors.Error
	if !stderrors.As(err, &kernel) || kernel == nil {
		return Error{Code: string(errors.Internal), Message: internalMessage}
	}

	described := Error{Code: string(kernel.Code), Message: kernel.Message}
	if kernel.Code != errors.Internal && len(kernel.Details) > 0 {
		described.Details = maps.Clone(kernel.Details)
	}
	if kernel.Retry != nil {
		described.Retry = &Retry{AfterMs: kernel.Retry.After.Milliseconds()}
	}
	return described
}

func Convert(err error) error {
	if err == nil {
		return nil
	}

	described := describe(err)
	converted := errors.New(errors.Code(described.Code), described.Message)
	for key, value := range described.Details {
		converted = converted.WithDetail(key, value)
	}
	if described.Retry != nil {
		converted = converted.WithRetry(time.Duration(described.Retry.AfterMs) * time.Millisecond)
	}
	return converted
}

func MarshalError(err error) []byte {
	encoded, marshalErr := json.Marshal(describe(err))
	if marshalErr != nil {
		return []byte(fallbackBody)
	}
	return encoded
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test -race ./internal/transport/wails/...`
Expected: PASS, all six table cases and the three round-trip tests.

- [ ] **Step 5: Commit**

```bash
git add internal/transport/wails/errors.go internal/transport/wails/errors_test.go
git commit -m "$(cat <<'EOF'
feat(transport): structured transport errors for the wails binding path

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>
EOF
)"
```

---

### Task 3: Service wrapper, health service and composition root

**Files:**
- Create: `internal/transport/wails/service.go`, `internal/transport/wails/health.go`, `internal/app/wiring.go`
- Test: `internal/transport/wails/service_test.go`, `internal/app/wiring_test.go`
- Modify: `internal/app/version.go`, `cmd/postulator/main.go`, `frontend/src/main.ts`
- Delete: `internal/app/health.go`, `internal/app/health_test.go`

**Interfaces:**
- Consumes: `wails.Convert` and `wails.MarshalError` (Task 2); `kernel/middleware` (`Handler[In, Out]`, `Recover`, `Audit`); `kernel/ctx` (`WithActor`, `ActorUser`, `ActorFrom`); `kernel/log` (`New`, `Config`, `Logger`).
- Produces:
  - `func Wrap[In, Out any](logger *zap.Logger, operation string, next middleware.Handler[In, Out]) middleware.Handler[In, Out]`
  - `func Services(logger *zap.Logger, build BuildInfo) []application.Service`
  - `type BuildInfo struct { Version, Commit, BuildDate string }` with camelCase tags
  - `type PingRequest struct{}`
  - `type HealthService struct{ ... }`, `func NewHealthService(logger *zap.Logger, build BuildInfo) *HealthService`, `func (*HealthService) Ping(context.Context, PingRequest) (BuildInfo, error)`
  - `func app.Logger() (*log.Logger, error)`, `func app.Services(logger *log.Logger) []application.Service`

- [ ] **Step 1: Write the failing test**

Create `internal/transport/wails/service_test.go`:

```go
package wails_test

import (
	"context"
	"encoding/json"
	"testing"

	"go.uber.org/zap"

	kernelctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/transport/wails"
)

func TestWrapPutsTheUserActorInTheContext(t *testing.T) {
	t.Parallel()

	handler := wails.Wrap(zap.NewNop(), "test.actor", func(c context.Context, _ struct{}) (kernelctx.Actor, error) {
		actor, ok := kernelctx.ActorFrom(c)
		if !ok {
			t.Error("the wrapper did not put an actor in the context")
		}
		return actor, nil
	})

	actor, err := handler(context.Background(), struct{}{})
	if err != nil {
		t.Fatalf("handler() error: %v", err)
	}
	if actor != kernelctx.ActorUser {
		t.Fatalf("actor = %q, want %q", actor, kernelctx.ActorUser)
	}
}

func TestWrapConvertsEveryFailure(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		handler func(context.Context, struct{}) (int, error)
		want    string
	}{
		{
			name: "kernel error keeps its code and loses its cause",
			handler: func(context.Context, struct{}) (int, error) {
				return 0, errors.New(errors.Invalid, "limit is out of range").WithDetail("limit", 9000)
			},
			want: `{"code":"INVALID","message":"limit is out of range","details":{"limit":9000}}`,
		},
		{
			name: "foreign error becomes internal",
			handler: func(context.Context, struct{}) (int, error) {
				return 0, context.DeadlineExceeded
			},
			want: `{"code":"INTERNAL","message":"unexpected internal error"}`,
		},
		{
			name: "panic becomes internal without the panic text",
			handler: func(context.Context, struct{}) (int, error) {
				panic("secret connection string")
			},
			want: `{"code":"INTERNAL","message":"handler panicked"}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := wails.Wrap(zap.NewNop(), "test.failure", tc.handler)(context.Background(), struct{}{})
			if err == nil {
				t.Fatal("handler() returned no error")
			}
			if result != 0 {
				t.Fatalf("handler() result = %d, want the zero value", result)
			}
			if got := string(wails.MarshalError(err)); got != tc.want {
				t.Fatalf("MarshalError() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestServicesBindsTheHealthService(t *testing.T) {
	t.Parallel()

	services := wails.Services(zap.NewNop(), wails.BuildInfo{Version: "2.0.0", Commit: "abc1234", BuildDate: "2026-09-17T10:30:00Z"})
	if len(services) != 1 {
		t.Fatalf("Services() returned %d services, want 1", len(services))
	}

	health, ok := services[0].Instance().(*wails.HealthService)
	if !ok {
		t.Fatalf("Services()[0] is %T, want *wails.HealthService", services[0].Instance())
	}

	build, err := health.Ping(context.Background(), wails.PingRequest{})
	if err != nil {
		t.Fatalf("Ping() error: %v", err)
	}
	if build.Version != "2.0.0" || build.Commit != "abc1234" || build.BuildDate != "2026-09-17T10:30:00Z" {
		t.Fatalf("Ping() = %+v, want the injected build stamps", build)
	}
}

func TestBuildInfoMarshalsCamelCase(t *testing.T) {
	t.Parallel()

	encoded, err := json.Marshal(wails.BuildInfo{Version: "2.0.0", Commit: "abc1234", BuildDate: "2026-09-17T10:30:00Z"})
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	const want = `{"version":"2.0.0","commit":"abc1234","buildDate":"2026-09-17T10:30:00Z"}`
	if string(encoded) != want {
		t.Fatalf("Marshal() = %s, want %s", encoded, want)
	}
}
```

Create `internal/app/wiring_test.go`:

```go
package app_test

import (
	"context"
	"testing"

	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/transport/wails"
)

func TestLoggerBuildsAConsoleLoggerThatCloses(t *testing.T) {
	t.Parallel()

	logger, err := app.Logger()
	if err != nil {
		t.Fatalf("Logger() error: %v", err)
	}
	if logger.Logger == nil {
		t.Fatal("Logger() returned a logger with no zap core")
	}
	if err := logger.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}
}

func TestServicesCarryTheInjectedStamps(t *testing.T) {
	t.Parallel()

	logger, err := app.Logger()
	if err != nil {
		t.Fatalf("Logger() error: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := logger.Close(); closeErr != nil {
			t.Errorf("Close() error: %v", closeErr)
		}
	})

	services := app.Services(logger)
	if len(services) != 1 {
		t.Fatalf("Services() returned %d services, want 1", len(services))
	}

	health, ok := services[0].Instance().(*wails.HealthService)
	if !ok {
		t.Fatalf("Services()[0] is %T, want *wails.HealthService", services[0].Instance())
	}

	build, err := health.Ping(context.Background(), wails.PingRequest{})
	if err != nil {
		t.Fatalf("Ping() error: %v", err)
	}
	if build.Version != app.Version || build.Commit != app.Commit || build.BuildDate != app.BuildDate {
		t.Fatalf("Ping() = %+v, want the ldflags stamps %q/%q/%q", build, app.Version, app.Commit, app.BuildDate)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/transport/wails/... ./internal/app/...`
Expected: FAIL — `undefined: wails.Wrap`, `undefined: wails.Services`, `undefined: app.Logger`.

- [ ] **Step 3: Write the implementation**

Create `internal/transport/wails/service.go`:

```go
package wails

import (
	"context"

	"github.com/wailsapp/wails/v3/pkg/application"
	"go.uber.org/zap"

	kernelctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/middleware"
)

func Wrap[In, Out any](logger *zap.Logger, operation string, next middleware.Handler[In, Out]) middleware.Handler[In, Out] {
	audited := middleware.Audit(logger, operation, middleware.Recover(next))

	return func(c context.Context, in In) (Out, error) {
		out, err := audited(kernelctx.WithActor(c, kernelctx.ActorUser), in)
		if err != nil {
			var zero Out
			return zero, Convert(err)
		}
		return out, nil
	}
}

func bind[T any](instance *T) application.Service {
	return application.NewServiceWithOptions(instance, application.ServiceOptions{MarshalError: MarshalError})
}

func Services(logger *zap.Logger, build BuildInfo) []application.Service {
	return []application.Service{
		bind(NewHealthService(logger, build)),
	}
}
```

Create `internal/transport/wails/health.go`:

```go
package wails

import (
	"context"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/kernel/middleware"
)

type BuildInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"buildDate"`
}

type PingRequest struct{}

type HealthService struct {
	ping middleware.Handler[PingRequest, BuildInfo]
}

func NewHealthService(logger *zap.Logger, build BuildInfo) *HealthService {
	return &HealthService{
		ping: Wrap(logger, "health.ping", func(_ context.Context, _ PingRequest) (BuildInfo, error) {
			return build, nil
		}),
	}
}

func (s *HealthService) Ping(c context.Context, req PingRequest) (BuildInfo, error) {
	return s.ping(c, req)
}
```

Delete the old service and its test:

```bash
git rm internal/app/health.go internal/app/health_test.go
```

Replace `internal/app/version.go` with:

```go
package app

var (
	Version   = "2.0.0-dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)
```

Create `internal/app/wiring.go`:

```go
package app

import (
	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/davidmovas/postulator/internal/kernel/log"
	"github.com/davidmovas/postulator/internal/transport/wails"
)

func Logger() (*log.Logger, error) {
	return log.New(log.Config{
		Level:   "info",
		Format:  log.FormatConsole,
		Service: "postulator",
		Version: Version,
		Console: true,
	})
}

func Services(logger *log.Logger) []application.Service {
	return wails.Services(logger.Logger, wails.BuildInfo{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
	})
}
```

Replace `cmd/postulator/main.go` with:

```go
package main

import (
	"fmt"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/davidmovas/postulator/frontend"
	"github.com/davidmovas/postulator/internal/app"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	logger, err := app.Logger()
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := logger.Close(); closeErr != nil {
			fmt.Fprintln(os.Stderr, closeErr)
		}
	}()

	desktop := application.New(application.Options{
		Name:        "Postulator",
		Description: "Entity-graph driven WordPress content factory",
		Services:    app.Services(logger),
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(frontend.Assets),
		},
	})

	desktop.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Postulator",
		Width:            1280,
		Height:           820,
		MinWidth:         960,
		MinHeight:        600,
		BackgroundColour: application.NewRGB(14, 16, 22),
		URL:              "/",
	})

	return desktop.Run()
}
```

Replace `frontend/src/main.ts` with:

```ts
import { HealthService } from "../bindings/github.com/davidmovas/postulator/internal/transport/wails";

import { parseError } from "./lib/errors.js";

const versionElement = document.getElementById("version") as HTMLElement;
const commitElement = document.getElementById("commit") as HTMLElement;
const builtElement = document.getElementById("built") as HTMLElement;
const statusElement = document.getElementById("status") as HTMLElement;

async function load(): Promise<void> {
    const build = await HealthService.Ping({});
    versionElement.innerText = build.version;
    commitElement.innerText = build.commit;
    builtElement.innerText = build.buildDate;
    statusElement.innerText = "ready";
    statusElement.classList.add("is-ready");
}

load().catch((thrown: unknown) => {
    const failure = parseError(thrown);
    statusElement.innerText = `${failure.code}: ${failure.message}`;
    statusElement.classList.add("is-failed");
});
```

`frontend/src/lib/errors.ts` does not exist yet, so the frontend build stays broken until Task 6. That is expected: this step's commit covers Go only, and `frontend/src/main.ts` is committed together with Task 6.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test -race ./internal/transport/wails/... ./internal/app/... && go vet ./...`
Expected: PASS; `deps_test.go` still green (only `internal/app` imports `internal/transport`).

- [ ] **Step 5: Commit the Go side**

```bash
git add internal/transport/wails/service.go internal/transport/wails/health.go internal/transport/wails/service_test.go internal/app/wiring.go internal/app/wiring_test.go internal/app/version.go cmd/postulator/main.go
git commit -m "$(cat <<'EOF'
feat(transport): service wrapper and health service behind the wails contract

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>
EOF
)"
```

---

### Task 4: Event bridge

**Files:**
- Create: `internal/transport/wails/eventbridge.go`
- Test: `internal/transport/wails/eventbridge_test.go`

**Interfaces:**
- Consumes: `events.Registry`, `events.Envelope`, `events.Entry`, `events.Type` (Task 1); `kernel/clock.Clock`; `kernel/dto.NewTime`; `kernel/errors`.
- Produces:
  - `type Emitter interface { Emit(name string, data ...any) bool }` — satisfied by `*application.EventManager`, declared here because the bridge is the consumer.
  - `func NewEventBridge(emitter Emitter, now clock.Clock) *EventBridge`
  - `func (*EventBridge) Publish(eventType events.Type, payload any) error`
  - `func (*EventBridge) PublishRun(runID string, seq int64, eventType events.Type, payload any) error`

- [ ] **Step 1: Write the failing test**

Create `internal/transport/wails/eventbridge_test.go`:

```go
package wails_test

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/transport/wails"
)

var _ wails.Emitter = (*application.EventManager)(nil)

type recordedEvent struct {
	name string
	data any
}

type fakeEmitter struct {
	mu       sync.Mutex
	recorded []recordedEvent
}

func (f *fakeEmitter) Emit(name string, data ...any) bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	event := recordedEvent{name: name}
	if len(data) == 1 {
		event.data = data[0]
	}
	f.recorded = append(f.recorded, event)
	return false
}

func (f *fakeEmitter) events() []recordedEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]recordedEvent(nil), f.recorded...)
}

func newBridge() (*wails.EventBridge, *fakeEmitter) {
	emitter := &fakeEmitter{}
	return wails.NewEventBridge(emitter, clock.NewFake(time.Date(2026, 9, 17, 10, 30, 0, 0, time.UTC))), emitter
}

func TestPublishWrapsThePayloadInAnEnvelope(t *testing.T) {
	t.Parallel()

	bridge, emitter := newBridge()
	if err := bridge.Publish(events.GraphChanged, events.GraphChangedPayload{SiteID: "s1"}); err != nil {
		t.Fatalf("Publish() error: %v", err)
	}

	recorded := emitter.events()
	if len(recorded) != 1 {
		t.Fatalf("emitter saw %d events, want 1", len(recorded))
	}
	if recorded[0].name != "graph.changed" {
		t.Fatalf("event name = %q, want %q", recorded[0].name, "graph.changed")
	}

	encoded, err := json.Marshal(recorded[0].data)
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	const want = `{"type":"graph.changed","seq":1,"at":"2026-09-17T10:30:00Z","payload":{"siteId":"s1"}}`
	if string(encoded) != want {
		t.Fatalf("envelope = %s, want %s", encoded, want)
	}
}

func TestPublishNumbersApplicationEventsMonotonically(t *testing.T) {
	t.Parallel()

	bridge, emitter := newBridge()
	for range 3 {
		if err := bridge.Publish(events.TemplatesChanged, events.TemplatesChangedPayload{}); err != nil {
			t.Fatalf("Publish() error: %v", err)
		}
	}

	for index, recorded := range emitter.events() {
		envelope, ok := recorded.data.(events.Envelope)
		if !ok {
			t.Fatalf("event %d carries %T, want events.Envelope", index, recorded.data)
		}
		if envelope.Seq != int64(index+1) {
			t.Fatalf("event %d has seq %d, want %d", index, envelope.Seq, index+1)
		}
	}
}

func TestPublishRunCarriesTheRunSequence(t *testing.T) {
	t.Parallel()

	bridge, emitter := newBridge()
	const runID = "6f3b2a11-0c9d-4e7a-8b25-1f4c6d7e8a90"

	if err := bridge.PublishRun(runID, 42, events.StepDone, events.StepDonePayload{RunID: runID, ItemID: "i1", Step: "validate", DurationMs: 120}); err != nil {
		t.Fatalf("PublishRun() error: %v", err)
	}

	encoded, err := json.Marshal(emitter.events()[0].data)
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	const want = `{"type":"step.done","seq":42,"runId":"6f3b2a11-0c9d-4e7a-8b25-1f4c6d7e8a90","at":"2026-09-17T10:30:00Z","payload":{"runId":"6f3b2a11-0c9d-4e7a-8b25-1f4c6d7e8a90","itemId":"i1","step":"validate","durationMs":120}}`
	if string(encoded) != want {
		t.Fatalf("envelope = %s, want %s", encoded, want)
	}
}

func TestPublishRejectsMisuse(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		call func(*wails.EventBridge) error
	}{
		{
			name: "unknown type",
			call: func(b *wails.EventBridge) error {
				return b.Publish(events.Type("nope"), events.GraphChangedPayload{})
			},
		},
		{
			name: "payload of the wrong type",
			call: func(b *wails.EventBridge) error {
				return b.Publish(events.GraphChanged, events.PagesChangedPayload{})
			},
		},
		{
			name: "run event without a sequence",
			call: func(b *wails.EventBridge) error {
				return b.Publish(events.RunStarted, events.RunStartedPayload{})
			},
		},
		{
			name: "application event with a sequence",
			call: func(b *wails.EventBridge) error {
				return b.PublishRun("r1", 1, events.GraphChanged, events.GraphChangedPayload{})
			},
		},
		{
			name: "run event without a run id",
			call: func(b *wails.EventBridge) error {
				return b.PublishRun("", 1, events.RunStarted, events.RunStartedPayload{})
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			bridge, emitter := newBridge()
			err := tc.call(bridge)
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("error = %v, want an INVALID kernel error", err)
			}
			if len(emitter.events()) != 0 {
				t.Fatalf("emitter saw %d events, want none", len(emitter.events()))
			}
		})
	}
}
```

The package-level assertion `var _ wails.Emitter = (*application.EventManager)(nil)` is
the proof that `desktop.Event` can be injected in Phase 5 without an adapter.

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/transport/wails/...`
Expected: FAIL — `undefined: wails.NewEventBridge`, `undefined: wails.EventBridge`, `undefined: wails.Emitter`.

- [ ] **Step 3: Write the implementation**

Create `internal/transport/wails/eventbridge.go`:

```go
package wails

import (
	"reflect"
	"sync/atomic"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type Emitter interface {
	Emit(name string, data ...any) bool
}

type EventBridge struct {
	emitter  Emitter
	clock    clock.Clock
	registry events.Registry
	seq      atomic.Int64
}

func NewEventBridge(emitter Emitter, now clock.Clock) *EventBridge {
	return &EventBridge{emitter: emitter, clock: now, registry: events.NewRegistry()}
}

func (b *EventBridge) Publish(eventType events.Type, payload any) error {
	entry, err := b.entry(eventType, payload)
	if err != nil {
		return err
	}
	if entry.Run {
		return errors.New(errors.Invalid, "a run event must be published with its run sequence").
			WithDetail("type", string(eventType))
	}

	b.emit(events.Envelope{
		Type:    eventType,
		Seq:     b.seq.Add(1),
		At:      dto.NewTime(b.clock.Now()),
		Payload: payload,
	})
	return nil
}

func (b *EventBridge) PublishRun(runID string, seq int64, eventType events.Type, payload any) error {
	entry, err := b.entry(eventType, payload)
	if err != nil {
		return err
	}
	if !entry.Run {
		return errors.New(errors.Invalid, "only a run event may be published with a run sequence").
			WithDetail("type", string(eventType))
	}
	if runID == "" {
		return errors.New(errors.Invalid, "a run event needs a run id").
			WithDetail("type", string(eventType))
	}

	b.emit(events.Envelope{
		Type:    eventType,
		Seq:     seq,
		RunID:   &runID,
		At:      dto.NewTime(b.clock.Now()),
		Payload: payload,
	})
	return nil
}

func (b *EventBridge) entry(eventType events.Type, payload any) (events.Entry, error) {
	entry, ok := b.registry.Lookup(eventType)
	if !ok {
		return events.Entry{}, errors.New(errors.Invalid, "unknown event type").
			WithDetail("type", string(eventType))
	}
	if reflect.TypeOf(payload) != reflect.TypeOf(entry.Payload) {
		return events.Entry{}, errors.New(errors.Invalid, "payload does not match the registered type").
			WithDetail("type", string(eventType))
	}
	return entry, nil
}

func (b *EventBridge) emit(envelope events.Envelope) {
	b.emitter.Emit(string(envelope.Type), envelope)
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test -race ./internal/transport/wails/...`
Expected: PASS, including the five misuse cases.

- [ ] **Step 5: Commit**

```bash
git add internal/transport/wails/eventbridge.go internal/transport/wails/eventbridge_test.go
git commit -m "$(cat <<'EOF'
feat(transport): event bridge publishing registry envelopes over the v3 bus

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>
EOF
)"
```

---

### Task 5: TypeScript event generator

**Files:**
- Create: `internal/transport/wails/gen/main.go`, `internal/transport/wails/gen/render.go`, `frontend/src/generated/events.ts`
- Test: `internal/transport/wails/gen/main_test.go`
- Modify: `Taskfile.yml`

**Interfaces:**
- Consumes: `events.NewRegistry().Entries()`, `events.Entry` (Task 1); `kernel/errors`.
- Produces (package `main`, callable by tests in the same package):
  - `const generatedFile = "frontend/src/generated/events.ts"`
  - `func write(path string) error`
  - `func render(w io.Writer, entries []events.Entry) error`
  - the committed `frontend/src/generated/events.ts` exporting `EventType`, one `XxxPayload` interface per event, `EventPayloads` and `Envelope<T extends EventType = EventType>`.

- [ ] **Step 1: Write the failing test**

Create `internal/transport/wails/gen/main_test.go`:

```go
package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func repoRoot(t *testing.T) string {
	t.Helper()

	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		t.Fatalf("go env GOMOD: %v", err)
	}

	gomod := strings.TrimSpace(string(out))
	if gomod == "" || gomod == os.DevNull {
		t.Fatal("the generator test must run inside the module")
	}
	return filepath.Dir(gomod)
}

func TestGeneratedFileIsInSync(t *testing.T) {
	t.Parallel()

	generated := filepath.Join(t.TempDir(), "events.ts")
	if err := write(generated); err != nil {
		t.Fatalf("write() error: %v", err)
	}

	fresh, err := os.ReadFile(generated)
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}

	committed, err := os.ReadFile(filepath.Join(repoRoot(t), filepath.FromSlash(generatedFile)))
	if err != nil {
		t.Fatalf("read committed file: %v", err)
	}

	if !bytes.Equal(fresh, committed) {
		t.Fatalf("%s is out of date; run `task events`", generatedFile)
	}
}

func TestRenderIsDeterministic(t *testing.T) {
	t.Parallel()

	var first, second bytes.Buffer
	if err := render(&first, events.NewRegistry().Entries()); err != nil {
		t.Fatalf("render() error: %v", err)
	}
	if err := render(&second, events.NewRegistry().Entries()); err != nil {
		t.Fatalf("render() error: %v", err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatal("two renders of the same registry differ")
	}
}

func TestRenderRejectsPayloadsItCannotType(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		entry events.Entry
	}{
		{
			name: "field kind without a TypeScript mapping",
			entry: events.Entry{Type: events.Type("x.map"), Payload: struct {
				Labels map[string]string `json:"labels"`
			}{}},
		},
		{
			name: "field without a json tag",
			entry: events.Entry{Type: events.Type("x.untagged"), Payload: struct {
				Name string
			}{}},
		},
		{
			name:  "payload that is not a struct",
			entry: events.Entry{Type: events.Type("x.scalar"), Payload: "nope"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := render(&bytes.Buffer{}, []events.Entry{tc.entry})
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("render() error = %v, want an INVALID kernel error", err)
			}
		})
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/transport/wails/gen/...`
Expected: FAIL — the package does not exist.

- [ ] **Step 3: Write the implementation**

Create `internal/transport/wails/gen/render.go`:

```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

var rawMessageType = reflect.TypeFor[json.RawMessage]()

func render(w io.Writer, entries []events.Entry) error {
	var out bytes.Buffer

	out.WriteString("export type EventType =\n")
	for index, entry := range entries {
		terminator := "\n"
		if index == len(entries)-1 {
			terminator = ";\n\n"
		}
		fmt.Fprintf(&out, "    | %q%s", string(entry.Type), terminator)
	}

	for _, entry := range entries {
		block, err := renderInterface(entry)
		if err != nil {
			return err
		}
		out.WriteString(block)
	}

	out.WriteString("export interface EventPayloads {\n")
	for _, entry := range entries {
		fmt.Fprintf(&out, "    %q: %s;\n", string(entry.Type), reflect.TypeOf(entry.Payload).Name())
	}
	out.WriteString("}\n\n")

	out.WriteString("export interface Envelope<T extends EventType = EventType> {\n")
	out.WriteString("    type: T;\n")
	out.WriteString("    seq: number;\n")
	out.WriteString("    runId?: string;\n")
	out.WriteString("    at: string;\n")
	out.WriteString("    payload: EventPayloads[T];\n")
	out.WriteString("}\n")

	_, err := w.Write(out.Bytes())
	return err
}

func renderInterface(entry events.Entry) (string, error) {
	payload := reflect.TypeOf(entry.Payload)
	if payload == nil || payload.Kind() != reflect.Struct {
		return "", errors.New(errors.Invalid, "event payload must be a struct").
			WithDetail("type", string(entry.Type))
	}

	if payload.NumField() == 0 {
		return fmt.Sprintf("export interface %s {}\n\n", payload.Name()), nil
	}

	var block strings.Builder
	fmt.Fprintf(&block, "export interface %s {\n", payload.Name())
	for index := range payload.NumField() {
		field := payload.Field(index)
		name, ok := jsonName(field)
		if !ok {
			return "", errors.New(errors.Invalid, "event payload field needs a json tag").
				WithDetail("type", string(entry.Type)).
				WithDetail("field", field.Name)
		}
		rendered, err := tsType(field.Type)
		if err != nil {
			return "", errors.New(errors.Invalid, "event payload field has an unsupported type").
				WithDetail("type", string(entry.Type)).
				WithDetail("field", field.Name)
		}
		fmt.Fprintf(&block, "    %s: %s;\n", name, rendered)
	}
	block.WriteString("}\n\n")
	return block.String(), nil
}

func jsonName(field reflect.StructField) (string, bool) {
	tag, ok := field.Tag.Lookup("json")
	if !ok {
		return "", false
	}
	name, _, _ := strings.Cut(tag, ",")
	if name == "" || name == "-" {
		return "", false
	}
	return name, true
}

func tsType(field reflect.Type) (string, error) {
	if field == rawMessageType {
		return "unknown", nil
	}

	switch field.Kind() {
	case reflect.String:
		return "string", nil
	case reflect.Int, reflect.Int64, reflect.Float64:
		return "number", nil
	case reflect.Bool:
		return "boolean", nil
	default:
		return "", errors.New(errors.Invalid, "unsupported field kind")
	}
}
```

Create `internal/transport/wails/gen/main.go`:

```go
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/davidmovas/postulator/internal/application/events"
)

const generatedFile = "frontend/src/generated/events.ts"

func main() {
	out := flag.String("out", filepath.FromSlash(generatedFile), "path of the generated TypeScript module")
	flag.Parse()

	if err := write(*out); err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
}

func write(path string) error {
	var buffer bytes.Buffer
	if err := render(&buffer, events.NewRegistry().Entries()); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return os.WriteFile(path, buffer.Bytes(), 0o600)
}
```

- [ ] **Step 4: Generate the committed module**

Run: `go run ./internal/transport/wails/gen`
Expected: `frontend/src/generated/events.ts` is created. Its first and last blocks must read exactly:

```ts
export type EventType =
    | "agent.confirm.requested"
    | "agent.delta"
    | "agent.tool.finished"
    | "agent.tool.started"
    | "app.locked"
    | "app.unlocked"
    | "graph.changed"
    | "item.done"
    | "item.failed"
    | "item.needs_human"
    | "item.started"
    | "llm.usage"
    | "pages.changed"
    | "run.budget_exceeded"
    | "run.cancelled"
    | "run.completed"
    | "run.failed"
    | "run.paused"
    | "run.queued"
    | "run.resumed"
    | "run.started"
    | "step.done"
    | "step.failed"
    | "step.retrying"
    | "step.started"
    | "templates.changed";
```

```ts
export interface Envelope<T extends EventType = EventType> {
    type: T;
    seq: number;
    runId?: string;
    at: string;
    payload: EventPayloads[T];
}
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test -race ./internal/transport/wails/...`
Expected: PASS, including `TestGeneratedFileIsInSync`.

- [ ] **Step 6: Add the Task target**

In `Taskfile.yml`, insert after the `bindings` task:

```yaml
  events:
    summary: Regenerates the TypeScript event module from the Go event registry
    cmds:
      - go run ./internal/transport/wails/gen
```

Run: `task events && go test -race ./internal/transport/wails/gen/...`
Expected: the file is rewritten identically and the sync test passes.

- [ ] **Step 7: Commit**

```bash
git add internal/transport/wails/gen frontend/src/generated/events.ts Taskfile.yml
git commit -m "$(cat <<'EOF'
feat(transport): generate the typed event module from the Go registry

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>
EOF
)"
```

---

### Task 6: Frontend contract library and typecheck

**Files:**
- Create: `frontend/src/lib/errors.ts`, `frontend/src/lib/events.ts`, `frontend/src/lib/paging.ts`
- Modify: `frontend/package.json`, `build/Taskfile.yml`
- Depends on: `frontend/src/main.ts` as rewritten in Task 3 and `frontend/src/generated/events.ts` from Task 5.

**Interfaces:**
- Consumes: `Envelope`, `EventType` from `../generated/events.js`; `Events` from `@wailsio/runtime`.
- Produces:
  - `errors.ts`: `type Code`, `interface Retry { afterMs: number }`, `interface TransportError { code: Code; message: string; details?: Record<string, unknown>; retry?: Retry }`, `function parseError(thrown: unknown): TransportError`, `function isCode(thrown: unknown, code: Code): boolean`
  - `events.ts`: `type Handler<T extends EventType> = (envelope: Envelope<T>) => void`, `function on<T extends EventType>(type: T, handler: Handler<T>): () => void`
  - `paging.ts`: `type Cursor = string`, `interface Sort`, `interface ListRequest`, `interface List<T>`, `const defaultLimit = 50`, `const maxLimit = 500`, `function page(limit: number, cursor?: Cursor): ListRequest`, `function listOf<T>(raw: { items: unknown; nextCursor?: Cursor; prevCursor?: Cursor; hasMore: boolean }): List<T>`

- [ ] **Step 1: Write the failing check**

Add the script to `frontend/package.json` so the check exists before the code does — replace the `scripts` block with:

```json
  "scripts": {
    "dev": "vite",
    "typecheck": "tsc --noEmit",
    "build:dev": "vite build --minify false --mode development",
    "build": "vite build --mode production",
    "preview": "vite preview"
  },
```

- [ ] **Step 2: Run it to verify it fails**

Run: `cd frontend && npm run typecheck`
Expected: FAIL — `src/main.ts(3,29): error TS2307: Cannot find module './lib/errors.js'`.

- [ ] **Step 3: Write the implementation**

Create `frontend/src/lib/errors.ts`:

```ts
export type Code =
    | "NOT_FOUND"
    | "CONFLICT"
    | "INVALID"
    | "UNAUTHORIZED"
    | "RATE_LIMITED"
    | "BUDGET_EXCEEDED"
    | "EXTERNAL"
    | "INTERNAL"
    | "CANCELLED"
    | "NEEDS_HUMAN"
    | "LOCKED";

export interface Retry {
    afterMs: number;
}

export interface TransportError {
    code: Code;
    message: string;
    details?: Record<string, unknown>;
    retry?: Retry;
}

const codes: ReadonlySet<string> = new Set<string>([
    "NOT_FOUND",
    "CONFLICT",
    "INVALID",
    "UNAUTHORIZED",
    "RATE_LIMITED",
    "BUDGET_EXCEEDED",
    "EXTERNAL",
    "INTERNAL",
    "CANCELLED",
    "NEEDS_HUMAN",
    "LOCKED",
]);

const fallback: TransportError = { code: "INTERNAL", message: "unexpected internal error" };

export function parseError(thrown: unknown): TransportError {
    if (typeof thrown !== "object" || thrown === null || !("cause" in thrown)) {
        return fallback;
    }

    const cause = (thrown as { cause: unknown }).cause;
    if (typeof cause !== "object" || cause === null) {
        return fallback;
    }

    const candidate = cause as Partial<TransportError>;
    if (typeof candidate.code !== "string" || !codes.has(candidate.code)) {
        return fallback;
    }
    if (typeof candidate.message !== "string") {
        return fallback;
    }

    const parsed: TransportError = { code: candidate.code, message: candidate.message };
    if (candidate.details !== undefined) {
        parsed.details = candidate.details;
    }
    if (candidate.retry !== undefined && typeof candidate.retry.afterMs === "number") {
        parsed.retry = { afterMs: candidate.retry.afterMs };
    }
    return parsed;
}

export function isCode(thrown: unknown, code: Code): boolean {
    return parseError(thrown).code === code;
}
```

Create `frontend/src/lib/events.ts`:

```ts
import { Events } from "@wailsio/runtime";

import type { Envelope, EventType } from "../generated/events.js";

export type Handler<T extends EventType> = (envelope: Envelope<T>) => void;

export function on<T extends EventType>(type: T, handler: Handler<T>): () => void {
    return Events.On(type, (event) => {
        handler(event.data as Envelope<T>);
    });
}
```

Create `frontend/src/lib/paging.ts`:

```ts
export type Cursor = string;

export interface Sort {
    field: string;
    desc: boolean;
}

export interface ListRequest {
    cursor?: Cursor;
    limit: number;
    sort?: Sort | null;
}

export interface List<T> {
    items: T[];
    nextCursor?: Cursor;
    prevCursor?: Cursor;
    hasMore: boolean;
}

export const defaultLimit = 50;
export const maxLimit = 500;

export function page(limit: number, cursor?: Cursor): ListRequest {
    const clamped = limit <= 0 ? defaultLimit : Math.min(limit, maxLimit);
    return cursor === undefined ? { limit: clamped } : { cursor, limit: clamped };
}

export function listOf<T>(raw: { items: unknown; nextCursor?: Cursor; prevCursor?: Cursor; hasMore: boolean }): List<T> {
    return {
        items: (raw.items ?? []) as T[],
        nextCursor: raw.nextCursor,
        prevCursor: raw.prevCursor,
        hasMore: raw.hasMore,
    };
}
```

- [ ] **Step 4: Run the check to verify it passes**

Run: `task bindings && cd frontend && npm run typecheck`
Expected: no output — `tsc` reports no errors over `src` and the freshly generated `bindings`.

- [ ] **Step 5: Wire the check into the build**

In `build/Taskfile.yml`, change the `build:frontend` task's `cmds` to:

```yaml
    cmds:
      - npm run typecheck
      - npm run {{if eq .DEV "true"}}build:dev{{else}}build{{end}} -q
```

- [ ] **Step 6: Verify the whole build**

Run: `task build && go vet ./... && golangci-lint run && go test -race -covermode=atomic -coverprofile=coverage.out ./... && go run ./cmd/covergate -profile coverage.out`
Expected: `bin/postulator.exe` is produced, lint reports 0 issues, tests pass, the coverage gate passes with `internal/application` now enforced at ≥ 80%.

- [ ] **Step 7: Run the binary and confirm the round trip**

Run: `bin/postulator.exe`
Expected: the window opens and the build panel shows the version, the commit and the build date read through `HealthService.Ping({})`, with the status line reading `ready`.

- [ ] **Step 8: Commit**

```bash
git add frontend/src/lib frontend/src/main.ts frontend/package.json build/Taskfile.yml
git commit -m "$(cat <<'EOF'
feat(frontend): transport error, event and paging helpers with a typecheck gate

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>
EOF
)"
```

---

### Task 7: Contract document, status and standing rulings

**Files:**
- Modify: `docs/CONTRACTS.md`, `docs/STATUS.md`, `CLAUDE.md`

**Interfaces:**
- Consumes: every name produced by Tasks 1–6.
- Produces: no code.

- [ ] **Step 1: Rewrite `docs/CONTRACTS.md`**

Replace the whole file with:

````markdown
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
- DTOs are declared in `internal/transport/wails` and mapped by hand. Domain types carry
  no JSON tags.
- Generics are allowed in exported signatures: `paging.List[T]` generates
  `List<T>` in TypeScript. A Go type with a custom `MarshalJSON` generates as `any`,
  which is why `Slice<T>` and `dto.Time` lose their shape; `frontend/src/lib/paging.ts`
  restores it with `List<T>` and `listOf<T>()`.
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

`runId` is present on run events only. `seq` is per run and gapless for run events, and a
per-process counter for application events. `at` is RFC3339 UTC.

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
`agent.confirm.requested`, `app.locked`, `app.unlocked`.

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
registration order.

`npm run typecheck` (`tsc --noEmit`) runs inside `task build` and covers `src` and the
generated `bindings`.

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
````

- [ ] **Step 2: Verify the length budget**

Run: `wc -l docs/CONTRACTS.md`
Expected: at most 150.

- [ ] **Step 3: Update `docs/STATUS.md`**

Under **Where we are**, replace the Phase 1 paragraph with:

```markdown
**Phase 1 track B is complete.** The Wails v3 spike answered the three open contract
questions, `docs/CONTRACTS.md` is final, and the event registry, transport error and
event scaffolding, the TypeScript generator and the frontend contract library have
landed. Track A (SQLite store, migrations, unit of work, secrets) is tracked separately.
```

Under **Decisions taken in Phase 0**, append a new section:

```markdown
## Decisions taken in Phase 1 (track B)

- **`ServiceOptions.MarshalError`, not `Options.MarshalError`.** `Bindings.Add`
  overwrites each bound method's marshaller with the service-level hook, so the
  application-level one never runs in beta.23. Services are registered with
  `application.NewServiceWithOptions`.
- **The transport error is `{code, message, details?, retry:{afterMs}?}`** delivered as
  the `cause` of the JavaScript rejection, not a JSON string in the message.
  `wails.Convert` strips the internal chain first, because `CallError.Message` is
  `err.Error()` and would otherwise leak the wrapped driver text.
- **`details` is dropped for `INTERNAL`.** `middleware.Recover` puts the panic text in
  `Details["panic"]`, and that must not reach the UI.
- **Generics survive the TypeScript generator.** `paging.List[T]` generates `List<T>`,
  so no concrete `XxxList` DTOs are needed. A custom `MarshalJSON` generates as `any`,
  which is why `paging.Slice[T]` and `dto.Time` lose their shape and
  `frontend/src/lib/paging.ts` restores it.
- **`application.RegisterEvent` is not used.** It would duplicate the registry in a
  second hand-written list and emit its typings into the gitignored `frontend/bindings`.
  Our own generator owns `frontend/src/generated/events.ts`, and a Go test keeps it in
  sync.
- **The generated TypeScript carries no header comment**, because the no-comments rule
  covers TypeScript including generated files. Its provenance is the `src/generated/`
  path and `docs/CONTRACTS.md`.
- **Envelope timestamps are `kernel/dto.Time`**, not `time.Time`: seconds precision,
  always UTC, matching every other DTO.
- **The health service moved to `internal/transport/wails`** and now has the contract
  shape `Ping(ctx, PingRequest) (BuildInfo, error)`, so the wrapper, the error hook and
  the frontend helper are exercised in production rather than only in tests.
- **Live events are best-effort.** v3 buffers nothing for a missing window, so
  `ListEvents(runId, sinceSeq, limit)` catch-up is mandatory.
```

Under **Open questions for Phase 1**, replace the first bullet with:

```markdown
- Closed: the transport error format, the event envelope and generics across the
  TypeScript generator are all settled in `docs/CONTRACTS.md`.
```

Under **Known gaps**, replace the first bullet with:

```markdown
- `internal/domain`, `internal/adapters` and `internal/runtime` do not exist yet. The
  dependency-rule test skips each rule whose tree is absent. `internal/application` and
  `internal/transport` now exist, so the application layer rule and the core coverage
  gate are live.
- `EventBridge` has no publisher yet. Phase 5 is its first consumer; it declares the
  consumer-side interface and injects `app.Event` in `internal/app`.
```

- [ ] **Step 4: Add the standing rulings to `CLAUDE.md`**

Append to the **Standing rulings** section:

```markdown
- **2026-09-17 (phase 1B)** — The error marshaller must be set per service with
  `application.NewServiceWithOptions`; `application.Options.MarshalError` is dead code in
  beta.23 because `Bindings.Add` overwrites each method's marshaller with the service
  option.
- **2026-09-17 (phase 1B)** — A service method returns `wails.Convert(err)`, never the
  raw error: `CallError.Message` is `err.Error()`, so a wrapped driver message would
  cross into the webview.
- **2026-09-17 (phase 1B)** — `application.RegisterEvent` is not used. The Go registry
  plus `go run ./internal/transport/wails/gen` owns the event typings; a second list
  would drift and its output lands in the gitignored `frontend/bindings`.
```

Add to the **Footguns** section:

```markdown
- **Live events are dropped, not buffered.** v3 dispatches a custom event to the windows
  that exist at that instant; with no window, or across a page reload, it is gone. Any
  progress UI must replay `RunsService.ListEvents(runId, sinceSeq, limit)` on connect.
- **`frontend/src/generated/events.ts` is generated and committed.** `task events`
  rewrites it and a Go test fails when it is stale. `frontend/bindings/` is the opposite:
  generated and gitignored, rewritten by every build.
```

- [ ] **Step 5: Verify the whole repository**

Run: `task build && go vet ./... && golangci-lint run && go test -race -covermode=atomic -coverprofile=coverage.out ./... && go run ./cmd/covergate -profile coverage.out`
Expected: all five green.

- [ ] **Step 6: Commit**

```bash
git add docs/CONTRACTS.md docs/STATUS.md CLAUDE.md
git commit -m "$(cat <<'EOF'
docs(contracts): final wails v3 contract, phase 1B decisions and rulings

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>
EOF
)"
```

---

## Notes for the orchestrator

- Track A is working in the same tree. `go.mod`, `Taskfile.yml`, `build/Taskfile.yml` and
  `cmd/postulator/main.go` are the likely collision points; Tasks 3, 5 and 6 touch the
  last three.
- `EventBridge` is not wired into `cmd/postulator/main.go` in this phase. It has no
  publisher until the run engine exists, and wiring it to nothing would be the kind of
  hole this repository forbids. Phase 5 declares the consumer-side `Publisher` interface
  — the rule that interfaces are declared by the consumer is why none is declared here —
  and injects `desktop.Event`, which already satisfies `wails.Emitter` (asserted at
  compile time in Task 4).
- The run-event payload fields in Task 1 are the Phase 1 proposal derived from spec §7.
  Phase 5 may add fields; adding a field is a compatible change and the generator picks
  it up. Removing or renaming one is a contract change and belongs in `docs/CONTRACTS.md`.
