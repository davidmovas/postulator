# Postulator v2 — Design Spec

Sections 1-10 and 15 of the approved plan, verbatim. This file is the contract;
the distilled docs in `docs/` point back here and never restate it.

---

## 1. Context

The current repo (v1.6.2) is a WordPress post generator written for the same client. Exploration found: one usable pattern (`StartX → taskID → Get/List/Cancel`), everything else unfit: Wails calls blocking for minutes, fire-and-forget event bus that never reaches the UI, no per-step persistence (crash leaves zombie rows and orphan WP posts), page map lacking H1/meta/anchors/page type, hierarchy stored three ways and drifting, SEO meta generated but never written to WP, secrets in plaintext, one test file in the whole repo. The code is discarded, not refactored.

The client (an SEO specialist, not a developer) has a network of WordPress sites (pages, posts, WooCommerce) with an Excel-based page map (slug, title, H1, keywords, anchors) and an Entity Graph. He wants pages generated so that internal links prove entity relationships: up to parents, down to own children, sideways only to explicitly related siblings, never outside the graph. He also wants to talk to local AI agents that can read his data, edit the graph and templates, run and monitor pipelines, and build a graph from scratch for a site that has nothing yet. The UI (canvas, reports) is built later with Claude Design against the Go contracts defined here.

## 2. Decisions Log (from question drilling, 2026-09-17)

| Topic | Decision |
|---|---|
| WP content types | Pages, posts, WooCommerce products and product categories |
| Graph shape | Two structures: URL tree and Entity Graph. Graph is a DAG on `parent` edges, `related` edges undirected with weight |
| Site state | Mixed: create, fill, rewrite, relink-only all needed |
| SEO plugin | Unknown per site; ship our own companion plugin that detects Yoast/RankMath and writes meta |
| Publishing | Per-run option: `draft` or `publish`. Review can happen in WP or in our UI |
| Relink of existing pages | Yes, as an optional separate step. Pipelines are composite, never monolithic |
| Sibling relevance | Only explicit approved `related` edges ≥ threshold; AI may propose edges for approval |
| LLM providers | OpenAI, Anthropic, Gemini via gollem. Library may be swapped if it blocks quality; our port isolates it |
| Agent scope | Everything the UI can do: read/explain, run/monitor, edit graph/templates, build graph from a bare site, create schedules |
| Client data | Excel is his source of truth. No sample file available; import uses saved column mappings + preview |
| Budget | Token and USD accounting per call, pre-flight estimate, hard cap per run with auto-pause |
| Wails | v3 beta, pinned version |
| Migrations | goose embedded SQL (tern is PostgreSQL-only) |
| Data protection | DPAPI-protected master key, AES-GCM secrets, adiantum full-DB encryption, optional master password (Argon2id), encrypted export/backup |
| Frontend | Old Next.js deleted. Vite stub with generated TS bindings during the Go phase |
| Platform | Windows only |
| Scheduling | First-class module: cron/interval, agent can create schedules |
| Module name | `postulator` |
| Tests | Unit + real SQLite repo tests + LLM record/replay + fake WP httptest server + docker WP e2e |
| Page composition | Body HTML with H2/H3 sections + internal links; meta title/description/canonical/OG; images (AI-generated, WP media library, local files from the client's PC) |
| Quality signals | Link compliance vs graph; keyword/structure checks; site-level coverage/orphans/depth/node weight; optional AI judge |
| Templates | Global base → site override → page override (JSON merge patch); client/agent editable; starter set shipped as data |
| Artifacts | Kept with retention; bodies purged N days after successful publish; metadata and hashes kept forever |
| IDs | UUID v4 |
| Pagination | Cursor-based everywhere; strong shared `kernel` package |
| MCP server | Not needed |
| Model selection | Profiles per role: writer, editor, linker, judge, chat, image; overridable in templates |
| Fable budget | Phases 2, 5, 6 (one Fable at a time) |
| DI | Manual composition root in `internal/app`, no fx |

## 3. Repository Layout

```
cmd/postulator/main.go          Wails v3 bootstrap only
internal/kernel/                shared: errors, log, paging, ctx, clock, id, dto, middleware
internal/domain/                pure logic, no infra imports
  site/ graph/ pagemap/ template/ content/ run/ schedule/ settings/ llm/ (model catalog types)
internal/application/           use cases (commands/queries), UnitOfWork, tool registry, event registry
  sites/ graph/ pages/ templates/ runs/ imports/ agent/ schedules/ reports/ settings/ tools/ events/
internal/adapters/
  sqlite/        store, migrations/, repositories per context
  wp/            REST client (core, wc, plugin namespace), fake server in wp/wptest
  llm/           port impl over gollem, catalog, ledger, limiter, recordreplay
  images/        openai, gemini, wpmedia, localfile
  secrets/       dpapi, aesgcm, masterpassword, export
  importer/      xlsx, csv, mapping
internal/runtime/               run engine: engine, worker, checkpoint, eventlog, recovery, budget
internal/transport/
  wails/         services, dto mapping, event bridge, ts generation config
  agent/         gollem agent, tools from registry, confirmations, conversations
internal/app/                   composition root
wp-plugin/                      PHP companion plugin (postulator-companion), zip packaging
frontend/                       React + Tailwind over the generated bindings; src/canvas is the 2D engine, src/features the screens
examples/                       import samples
docs/                           see section 10
testdata/                       llm fixtures, wp fixtures
```

Dependency rule (enforced by a test in `internal/app` using `go list -deps`): `domain` imports only `kernel` and stdlib; `application` imports `domain` + `kernel`; `adapters`, `runtime`, `transport` import `application`/`domain`/`kernel`; nothing imports `transport` except `app`.

## 4. Kernel (shared package)

- `kernel/errors` (port of Archond `internal/shared/errors/{errors,codes}.go` minus HTTP status): `type Code string` with frozen constants `NotFound, Conflict, Invalid, Unauthorized, RateLimited, BudgetExceeded, External, Internal, Cancelled, NeedsHuman, Locked`. `type Error struct { Code Code; Message string; Details map[string]any; Retry *RetryInfo; internal error; stack []uintptr }` with `Error()`, `Unwrap()`, `Is(target error) bool` comparing by `Code`. Constructors `New(code Code, msg string) *Error`, `Wrap(err error, code Code, msg string) *Error`; clone-and-mutate `WithInternal(err)`, `WithRetry(after time.Duration)`, `WithDetail(k string, v any)`. Helpers `CodeOf(err error) Code` (returns `Internal` for foreign errors), `IsCode(err error, code Code) bool`, `Stack(err) []Frame`.
- `kernel/paging` (port of Archond `pkg/dbx/{page,keyset}.go` + `pkg/api/list.go`, jet replaced by squirrel): `type Cursor string` (opaque base64url JSON `{"o":<orderBy>,"d":<asc|desc>,"v":[<values>],"i":"<row id>"}`), `type Request struct { After, Before Cursor; Limit int }` (default 50, max 500, `Normalize()`), `type List[T any] struct { Items Slice[T]; Cursors Cursors{Next, Prev Cursor}; HasMore bool }` where `Slice[T]` marshals nil as `[]`. `SortKey[T]` with kinds `Text, UUID, Enum, Int, Float, Bool, Time` and a `literal()` coercer; `Keyset[T]{Keys []SortKey[T], IDColumn string}` builds the squirrel predicate `(k1, id) > (?, ?)` (or `<` for desc / `Before`), applies `ORDER BY` with id tie-break and `LIMIT n+1`; `Cut[T](rows []T, limit int, key func(T) []any, id func(T) string) List[T]`. `paging` is the one kernel package allowed to import squirrel; domain never uses it.
- `kernel/id`: `New() string`, `Valid(s string) bool` (UUID v4; helper shape as Archond `pkg/identifier/uuid.go`).
- `kernel/clock`: `type Clock interface { Now() time.Time }`, `System{}`, `Fake` with `Set/Advance`.
- `kernel/log` (port of Archond `pkg/logger/logger.go`): zap builder with console core + lumberjack file core, JSON vs console encoder switch, default fields `service, version`; redaction of keys `password, apiKey, token, authorization`; sinks `app.log` and `errors.log`.
- `kernel/settings` (port of Archond `internal/settings/constructors.go`, without DB overrides/pins): settings-as-code, `settings.Int("runs.workers", 2, settings.IntRange(1, 16))`, `settings.String`, `settings.Bool`, `settings.Duration`, each declared next to the code that reads it, validated and defaulted, persisted in the `settings` table, exposed to the UI through a generated schema.
- `kernel/ctx`: `WithRunID/RunID`, `WithConversationID/ConversationID`, `WithActor/Actor` (`user|agent|schedule`).
- `kernel/dto`: `Time` (RFC3339 UTC string ↔ time.Time), `Sort{Field string; Desc bool}`, `ListRequest{Cursor string; Limit int; Sort *Sort}`.
- `kernel/middleware`: generic wrappers `func Recover[In, Out any](fn) fn`, `Timeout`, `Audit(logger)`.

## 5. Domain Model

### 5.1 Site (`domain/site`)
`Site{ID, Name, BaseURL, SecretRef, Status(active|paused|error), Plugin PluginState{Installed bool, Version string, Capabilities []string, SEOPlugin string}, Defaults{TemplateID, LinkPolicyID, ModelProfiles map[Role]ModelRef}, CreatedAt, UpdatedAt}`. Invariant: `BaseURL` https unless `AllowInsecure`.

### 5.2 Entity Graph (`domain/graph`)
- `Entity{ID, SiteID, Name, Kind(hub|product|topic|category|custom), Intent, PrimaryKeyword, SecondaryKeywords []string, Anchors []Anchor, CanonicalPageID *string, Score float64, Source(import|user|ai), CreatedAt, UpdatedAt}`; `Anchor{Text, Source(user|ai), Weight}`.
- `Edge{ID, SiteID, FromEntityID, ToEntityID, Kind(parent|related), Weight float64, Source(import|user|ai), Status(approved|proposed|rejected), CreatedAt}`. `parent` is directed child→parent; `related` is stored with `FromEntityID < ToEntityID`.
- `Graph` value type built from entities+edges with pure methods: `Parents(id, depth)`, `Children(id)`, `Related(id, minWeight)`, `Roots()`, `ValidateAcyclic() error`, `Score()` (PageRank-like over approved edges, damping 0.85, 30 iterations).
- Invariants: no self edges, no cycles on parent edges, edge endpoints in the same site.

### 5.3 Page Map (`domain/pagemap`)
- `Page{ID, SiteID, Path, Slug, ParentPageID *string, WPType(page|post|product|product_cat), WPID *int64, Title, H1, MetaTitle, MetaDescription, Canonical, Status(planned|exists|published|archived), EntityID *string, TemplateID *string, ContentHash, WPModifiedAt *time, LastSyncedAt *time, Drift bool, CreatedAt, UpdatedAt}`.
- `PageLink{ID, SiteID, FromPageID, ToPageID *string, ToURL, AnchorText, Origin(generated|observed), ObservedAt}`.
- Pure helpers: `NormalizePath`, `ParentPath`, `BuildTree(pages)`, `Unmapped(pages)`.
- `Index` value type: `NewIndex(pages []Page) Index` with `ByID(id) (Page, bool)`, `ByEntity(entityID) []Page`, `ByPath(path) (Page, bool)`.
- Link target resolution: `Entity.CanonicalPageID → Page.Path`; entities without a canonical page are never link targets.
- `Cannibalization(candidate Page, entity graph.Entity, index Index, g graph.Graph) Verdict{Allowed bool, Evidence []Evidence{PageID, Path, Reason(same_primary_keyword|same_entity_canonical|path_conflict), EntityID}}` lives here since Phase 2 (moved from §6): it depends only on the page index and the entity graph, and `pages.Create`, `pages.MapToEntity` and the import preview call it. `InternalPath(href, siteHost) (path, internal)` classifies a link target against the site host; `NormalizePath` is the one canonical path normaliser and the WordPress adapter and plugin implement its rules.

### 5.4 Templates and Link Policy (`domain/template`)
- `Template{ID, Scope(global|site), SiteID *string, Name, PageKind, Version int, Spec TemplateSpec, CreatedAt, UpdatedAt}`.
- `TemplateSpec{Sections []Section{Heading, Intent, TargetWords, Required bool, KeywordRules}, Tone, Length{Min, Max}, KeywordRules{PrimaryInTitle, PrimaryInH1, PrimaryInFirstParagraph bool, MaxDensity float64}, LinkRules{UpDepth int, DownLinks bool, SiblingMinWeight float64, MaxLinks, MaxPerTarget int, ParentLinkWithinParagraphs int, ChildrenSection bool}, MetaRules{TitlePattern, DescriptionMax int}, Images{Featured bool, Inline int, Source(ai|wpmedia|local)}, ModelProfiles map[Role]ModelRef, Recipe []StepSpec}`.
- `StepSpec{Name string, Enabled bool, Params map[string]any}`.
- `Resolve(base TemplateSpec, siteOverride, pageOverride json.RawMessage) (TemplateSpec, error)` implements RFC 7396 merge patch over the JSON form of the spec (Phase 2): overrides are stored as merge-patch documents, arrays are replaced wholesale, `null` removes a key, and the resolved spec must pass `Validate`. `TemplateSpec` and its nested structs carry camelCase JSON tags because that JSON is their persisted form.
- `LinkPolicy` is the `LinkRules` plus site-wide `ForbidExternal bool, ForbidSelf bool, AnchorStrategy(prefer_user|rotate)`.
- Starter templates (Hub, Product, Guide, Comparison, Category) live in `domain/template/seed/*.json`, embedded, loaded by migration seeding in the application layer.

### 5.5 Content (`domain/content`) — see section 6.

### 5.6 Runs (`domain/run`)
- `Run{ID, SiteID, Kind(generate|relink|audit|sync|import|custom), Status(queued|running|paused|completed|failed|cancelled), Targets []string, Recipe []StepSpec, TemplateVersion int, PublishMode(draft|publish), Budget{MaxUSD, MaxTokens}, Stats{Items, Done, Failed, Tokens, USD}, CreatedBy Actor, ParentRunID *string, CreatedAt, StartedAt, FinishedAt}`.
- `Item{ID, RunID, PageID, Status(pending|running|done|failed|skipped|needs_human), CurrentStep string, Attempts int, Error *string}`.
- `StepExec{ID, ItemID, Step string, Attempt int, Status(started|done|failed), InputHash string, ArtifactID *string, Tokens int, USD float64, StartedAt, FinishedAt, Error *string}`.
- `Artifact{ID, RunID, ItemID, Step string, Kind(html|json|image|report), Blob []byte, Size int, ExpiresAt *time}`.
- `Event{RunID string, Seq int64, Type string, At time.Time, Payload json.RawMessage}`.
- `type ArtifactKind string` (values listed in section 6). `StepContext{Run, Item, Page, Template TemplateSpec, Artifacts map[ArtifactKind]Artifact, Params map[string]any, Deps StepDeps}` where `StepDeps` bundles the ports a step may use (LLM client, WP client, image provider, repositories); `StepOutput{Artifacts []Artifact, Tokens int, USD float64, Events []Event}`.
- `Step` interface (consumer-side in runtime): `Name() string; Requires() []ArtifactKind; Produces() []ArtifactKind; Execute(ctx, *StepContext) (*StepOutput, error)`. `ValidateRecipe(steps map[string]Step, recipe []StepSpec) error` checks that every enabled step's `Requires` is produced earlier.

### 5.7 Schedules (`domain/schedule`)
`Schedule{ID, SiteID, Name, Cron string, Interval *time.Duration, TargetQuery TargetQuery{EntityID *string, Status, Limit int}, TemplateID, Recipe []StepSpec, PublishMode, Budget, Enabled, NextRunAt, LastRunID, CreatedBy}`. `NextAfter(now)` pure.

### 5.8 Settings (`domain/settings`)
Typed settings: `Proxy{URL, Enabled}`, `Retention{ArtifactDays int}`, `Concurrency{Workers int, PerSite int}`, `ModelProfiles`, `Security{MasterPasswordEnabled bool}`.

### 5.9 LLM catalog types (`domain/llm`)
`ModelRef{Provider, Model}`, `ModelInfo{Ref, ContextTokens, MaxOutputTokens, InputUSDPerM, OutputUSDPerM, RPM, TPM, SupportsStructured, SupportsImages, Reasoning bool}`, `Usage{Input, Output, Total int}`, `Cost(usage, info) float64`, `Role(writer|editor|linker|judge|chat|image)`.

## 6. Content Factory (`domain/content`)

- `Document`: wrapper around `*html.Node` (x/net/html) parsed from body HTML fragment. Methods: `TextNodes(skip func(*html.Node) bool) iter.Seq[*html.Node]`, `Paragraphs() []*html.Node`, `Headings() []*html.Node`, `Links() []Link{Href, Anchor, Node}`, `HTML() string`, `Hash() string` (sha256 of normalised HTML).
- `LinkTarget{EntityID, PageID, URL, Anchors []string, Relation(up|down|sibling), Required bool, Weight float64}`.
- `BuildLinkContext(g graph.Graph, pages pagemap.Index, entityID string, policy LinkPolicy) LinkContext`: up targets for depth 1..UpDepth (all `Required`), down targets for direct children with canonical pages, sibling targets for approved related edges with weight ≥ `SiblingMinWeight`. Self excluded. Deterministic ordering: up by depth, down by entity score desc, siblings by weight desc.
- `InsertLinks(doc *Document, lc LinkContext, policy LinkPolicy) InsertResult{Placed []Placement{Target, Anchor, ParagraphIndex}, Missing []LinkTarget, Decisions []Decision{Target, Anchor, Outcome(inserted|already_linked|anchor_not_found|cap_reached|forbidden_zone|position_rule), Detail string}}` (decision-per-candidate as in Archond `internal/modules/content/linking.go`): walks text nodes in document order, skips nodes inside `a`, `h1..h6`, `code`, `pre`, or attributes; case-insensitive rune-based match of any anchor of any unplaced target; first match wins; respects `MaxLinks`, `MaxPerTarget`, and `ParentLinkWithinParagraphs` (parent targets only match in paragraphs `< N`, others anywhere); splices `<a href>` preserving original text casing; idempotent (existing links to a target count as placed).
- `Cannibalization` moved to §5.3 (`domain/pagemap`) in Phase 2.
- `Compliance(doc, lc, policy, pageID) Report`: for every target: `satisfied|missing`; every `<a>` in doc classified `graph|self|external|unknown_internal`; anchor whitelist violations; counts. `Report{Items []Finding{Severity(info|warn|error), Code, Message, Details}, Score float64}`.
- `Structure(doc, primary string, secondary []string, spec TemplateSpec) Report`: primary keyword in title/H1/first paragraph, secondary coverage, density, word count vs `Length`, headings match `Sections`, HTML validity.
- `ContentDraft{Title, H1, Sections []DraftSection{Heading, HTML}, Summary}` is the structured-output schema for `generate_body`; `Assemble(draft) *Document`.
- `RepairRequest{ParagraphIndex int, Phrase string}` / `RepairResponse{Sentence string}` for `repair_links`.

Step catalog (implemented in `internal/runtime/steps`, each in its own file, each registered by name): `resolve_context, generate_body, generate_meta, insert_links, repair_links, generate_images, validate, judge, publish, relink_neighbors, sync_back, report`. Artifact kinds: `link_context, draft, body_html, meta, images, validation_report, judge_report, publish_result, relink_result, sync_result, final_report`.

Behaviour of each step:
- `resolve_context`: builds `LinkContext` from graph+pages; artifact `link_context`.
- `generate_body`: prompt from template spec, keywords, required anchor phrases (from `link_context`), `Structured[ContentDraft]` with writer profile; artifact `draft` + `body_html`.
- `generate_meta`: `Structured[Meta{Title, Description, Canonical, OG}]` with editor profile under `MetaRules`; artifact `meta`.
- `insert_links`: pure; artifact `body_html` (replaced) + placement list in `validation_report` seed.
- `repair_links`: for each `Missing` required target, ask model (linker profile) for one sentence containing an anchor at a paragraph index; re-run `InsertLinks`; max 2 iterations.
- `generate_images`: `ImageProvider` by template `Images.Source`; uploads to WP media; artifact `images` (ids, urls, alt).
- `validate`: `Compliance` + `Structure`; artifact `validation_report`; fails item if any `error` finding unless `Params.allow_errors`.
- `judge`: `Structured[JudgeReport{Score, Issues, Suggestions}]` with judge profile; artifact `judge_report`; never fails the item.
- `publish`: idempotent create/update in WP (lookup by `WPID`, then by slug+parent), sets status per `PublishMode`, writes SEO meta through the plugin when capability present; artifact `publish_result{WPID, URL, Status}`.
- `relink_neighbors`: for affected pages (up targets, children, siblings) fetch current content, compute missing edges only, `InsertLinks`, compare `ContentHash` before update, conflict → item `needs_human`; artifact `relink_result`.
- `sync_back`: re-read page, update `WPID, ContentHash, WPModifiedAt, PageLink` rows; artifact `sync_result`.
- `report`: aggregates all reports into `final_report` and updates `Run.Stats`.

## 7. Run Engine (`internal/runtime`)

Shapes ported from Archond `internal/infra/jobs/pipeline/{pipeline,checkpoint,definition,handler,daemon,repository}.go`, with the Asynq executor and credit reservations removed.

- Vocabulary: `Status(pending|running|waiting|paused|completed|failed|cancelled)`; each step returns `Transition(continue|wait|pause|complete|fail)` plus an optional `WakeAt` for `wait`; `Fault{Class(transient|exhausted|invalid|fatal), Code, Action(retry|pause|fail), Message, Reason}` with `Classify(err) Fault` mapping kernel error codes (`RateLimited/External/Cancelled → transient`, `BudgetExceeded → exhausted`, `Invalid/Unauthorized → invalid`, `NeedsHuman → pause`); `PauseReason(budget_exceeded|awaiting_confirmation|needs_human|user)`.
- Checkpoint: `type Checkpoint map[string]json.RawMessage` with generic `Get[T](cp, key) (T, bool, error)`, `Set[T](cp, key, v) error`, `MergedWith(other) Checkpoint`; stored as one JSON column per `Item`; large outputs go to `Artifact` rows and the checkpoint stores their ids.
- Definition/registry: `Definition{Kind, Steps []StepDef, Deadline time.Duration}`, `StepDef{Name, Requires, Produces []ArtifactKind, Retry RetryPolicy{Max int, Backoff func(attempt) time.Duration}, Timeout time.Duration, Run func(ctx, *StepContext) (Result, error)}`; `Registry.Register(def)` validates names, artifact dependencies, and retry bounds at startup and fails fast.
- Advance loop (`handler.go` shape, single process): `advance(itemID)` claims the item in one SQLite transaction by compare-and-swap on `advance_seq`, runs the current step under its timeout, classifies the outcome, persists the new item state + checkpoint + a `StepExec` row + an appended `Event`, and schedules the next advance (immediate, at `WakeAt`, or none if terminal). Each step is idempotent via `alreadyDone(itemID, step)` on `StepExec` and `InputHash` = sha256(step name, params, required artifact hashes, template version).
- Sweep daemon (`daemon.go` shape): every N seconds re-arms `DueWaiting` items (wall-clock `WakeAt` reached), reclaims `Stalled` items (lease expired: process died mid-step), and `Reap` fails runs past `DeadlineAt` and purges old terminal rows per retention. This is the whole crash-resume story; `Recover()` at startup is just one sweep.
- `Engine{Enqueue(ctx, run) error; Pause(id, reason); Resume(id); Cancel(id); RetryStep(itemID); Wake(itemID)}`; `Workers` (N from settings) pull runnable items with per-site concurrency limit, per-provider+model rate limiter (catalog RPM/TPM), and a run-scoped `context.CancelFunc` map.
- Budget: pre-flight `EstimateRun(run, template, catalog) Estimate{Tokens, USD}` from section lengths × profile prices; runtime accumulates `Stats.USD`; exceeding `Budget` returns `Fault{Class: exhausted}` → run paused with `PauseReason budget_exceeded` and event `run.budget_exceeded`.
- Confirmations inside runs: a step that needs approval (publish when the run was started by an agent in `confirm` mode) returns `Transition pause` with `PauseReason awaiting_confirmation` and a `PendingAction` row (section 8); `ResolveAction` resumes the item.
- Item-level failure never fails the run; run finishes `completed` with `Stats.Failed > 0`, or `failed` only if every item failed.
- Event log: `eventlog.Append(ctx, runID, type, payload) (seq int64)`, `List(runID, sinceSeq, limit)`; every append also publishes to the live bus consumed by the Wails bridge. Event types: `run.queued, run.started, run.paused, run.resumed, run.cancelled, run.completed, run.failed, run.budget_exceeded, item.started, item.done, item.failed, item.needs_human, step.started, step.done, step.failed, step.retrying, llm.usage`.
- Retention job: purges `Artifact.Blob` for `html|image` kinds older than `Retention.ArtifactDays` after `publish_result` succeeded; keeps reports and metadata.

## 8. Contracts (Wails v3, `internal/transport/wails`)

- Services: `SitesService, GraphService, PagesService, TemplatesService, RunsService, AgentService, ImportService, SchedulesService, ReportsService, SettingsService`. Thin: DTO mapping only.
- Method shape `func (s *X) Method(ctx, req ReqDTO) (RespDTO, error)`. Errors: `*kernel/errors.Error`; the service wrapper converts to the transport form decided by the Phase 1 spike (candidate: error string is JSON `{"code","message","details"}`; frontend helper `parseError`). Recorded in `docs/CONTRACTS.md`.
- DTOs: camelCase JSON, RFC3339 UTC times, `kernel/dto.ListRequest` in, `{items, nextCursor}` out. No generics in exported signatures if the spike shows the TS generator mangles them; otherwise `Page[T]` is allowed.
- Long-running: any mutation > 1s returns `{runId}` immediately. Progress via events + `RunsService.ListEvents(runId, sinceSeq, limit)`, `Get(runId)`, `Pause/Resume/Cancel(runId)`.
- Events: one Go registry `application/events` with typed constants and payload structs; `transport/wails/eventbridge` emits envelope `{type, seq, runId?, at, payload}`; a generator (`go run ./internal/transport/wails/gen`) writes `frontend/src/generated/events.ts` from the registry. Non-run events: `graph.changed{siteId}`, `pages.changed{siteId}`, `templates.changed`, `agent.delta{conversationId, messageId, seq, text}`, `agent.tool.started/finished{conversationId, callId, tool, args|result}`, `agent.confirm.requested{confirmationId, tool, args, risk}`, `app.locked/unlocked`.
- Agent chat: `AgentService.Send(conversationId, text) → {messageId}`, `Confirm(confirmationId, approve bool)`, `ListConversations`, `ListMessages(conversationId, cursor)`, `SetMode(conversationId, mode(confirm|autonomous))`.
- Tool registry (`application/tools`, shape of Archond `internal/modules/agents/tools/tools.go` + one tool per file, framework-neutral `llm.Tool` from `internal/infra/llm/types.go`): `type Tool struct { Def Def{Name, Description string; Risk Risk(read|write|dangerous); Schema *jsonschema.Schema}; Authorize func(ctx, Binding) error; Run func(ctx, Binding, args json.RawMessage) (any, error) }`; `NewTool[In, Out any](def Def, fn func(ctx, Binding, In) (Out, error)) Tool` derives `Schema` from `In`. `Binding{SiteID, ConversationID, RunID, Mode(confirm|autonomous)}` scopes every call. `Factory.Build(Binding) []Tool` and `Names() []string` live in `application/tools/registry.go`; each tool in its own file under `application/tools/`. Every application use case that the UI exposes is registered here; `transport/agent` adapts `Tool` to gollem at the edge only. Wails services call the same use-case functions directly (typed), never through the registry.
- Confirmations (Archond `Binding.ProposeAction` + `internal/modules/agents/service/actions.go`): a `write|dangerous` tool in `confirm` mode does not execute; it writes a `PendingAction{ID, ConversationID, Tool, Args, Summary, Status(pending|approved|rejected|executed|failed), Result}` row and returns `{status:"confirmationRequired", actionId, summary}` to the model, emitting `agent.confirm.requested`. `AgentService.Confirm(actionId, approve)` → `ResolveAction`: CAS on status, run the registered `ActionExecutor` for the tool, store the outcome, and resume the conversation with the tool result injected. Survives restarts because the pending action is a row, not a goroutine.
- Tool guard middleware chain, in this order (Archond `internal/infra/llm/middleware.go`): `fence` (wrap tool output as untrusted data so tool results cannot inject instructions) → `audit` (ledger + stream event per call) → `capResult` (truncate oversized tool JSON before it re-enters the model) → `permission` (allow-list per conversation + `Authorize`, denies with `Unauthorized`).

## 9. Adapters

### 9.1 SQLite (`adapters/sqlite`)
- `Open(path string, key []byte) (*Store, error)`: `file:` URI with `_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)`, adiantum VFS when `key != nil`, `SetMaxOpenConns` readers + a dedicated writer connection.
- `Store.Do(ctx, fn func(ctx) error) error` runs `fn` in a transaction, tx stored in ctx; repositories obtain executor via `execFrom(ctx)`.
- Errors mapped from `sqlite3.Error` extended codes to kernel codes (`SQLITE_CONSTRAINT_UNIQUE → Conflict`, `SQLITE_BUSY → External`). Repository methods return through `dbx.Result[T]` (Archond `pkg/dbx/result.go`): `.NotFound(err)`, `.Conflict(err)`, `.WrapErr(fn)` convert driver errors to kernel errors at the boundary without per-call `if err` chains.
- Migrations: `adapters/sqlite/migrations/0001_*.sql...` embedded, goose `UpContext` at startup. Tables (all with `id TEXT PK`, `created_at`, `updated_at`): `sites, secrets, entities, entity_anchors, edges, pages, page_links, templates, template_overrides, link_policies, runs, run_items, step_execs, artifacts, run_events, llm_calls, model_catalog, model_profiles, conversations, messages, tool_calls, confirmations, schedules, import_mappings, settings, app_meta`.
- Test helper `sqlitetest.Open(t) *Store` on a temp file with migrations applied.

### 9.2 Secrets (`adapters/secrets`)
- `dpapi.Protect/Unprotect` (CryptProtectData, user scope). Master key: 32 random bytes, stored protected at `%APPDATA%/Postulator/master.key`.
- `aesgcm.Seal/Open` with random nonce, versioned envelope `v1:` prefix.
- Optional master password: master key wrapped with Argon2id(password, salt, t=3, m=64MiB, p=4) key; `Unlock(password)` required before `Store.Open`; app emits `app.locked`.
- `export.Write(path, password)` / `Read`: tar of DB + settings, Argon2id + AES-GCM chunked.
- Port defined in application: `type SecretStore interface { Put(ctx, ref, value string) error; Get(ctx, ref) (string, error); Delete(ctx, ref) error }`.

### 9.3 WordPress (`adapters/wp`)
- `Client` per site: base URL, app-password Basic auth, proxy from settings, timeouts, retries on 429/5xx, no-redirect policy for diagnostics.
- Core: `ListItems(type, since, cursor)`, `GetItem`, `CreateItem`, `UpdateItem` (POST), `UploadMedia`, `UpdateMedia`, `ListCategories`, `ListProductCategories`, `ListProducts`, `UpdateProduct`. Handles `X-WP-Total`, slug rewrite on create, tri-state parent (`nil` keep, `0` top, `id` set), `page number larger` end-of-list quirk.
- Plugin namespace `/wp-json/postulator/v1`: `GET /manifest → {version, capabilities:[bulk,seo_meta,content_hash], seoPlugin: yoast|rankmath|none}`, `GET /content?since=&cursor=&types= → {items:[{id,type,slug,path,parent,status,modified,contentHash,title,h1,meta{title,description,canonical},links:[{href,anchor}]}], nextCursor}`, `PUT /seo-meta/{id} {title,description,canonical,ogTitle,ogDescription}`, `GET /content/{id}/raw`.
- `Capabilities(site)` gates plugin features; without the plugin: sync via core REST per type, links parsed locally from rendered content, SEO meta read-only.
- `wptest.Server`: httptest fake implementing the subset above with in-memory state, used by adapter tests and by runtime tests.

### 9.4 LLM (`adapters/llm`)
- Port (declared in `application/llm`): `type Client interface { Complete(ctx, Request) (Response, error); Stream(ctx, Request) (<-chan Delta, error) }` and generic `Structured[T any](ctx, Client, Request) (T, Usage, error)`. `Request{Ref ModelRef, System string, Messages []Message, MaxTokens int, Temperature *float64, Schema *jsonschema.Schema}`; `Response{Text string, Usage Usage, FinishReason string}`.
- `gollemclient` implements the port using gollem provider clients (openai, claude, gemini) and `gollem.Query[T]` for structured output.
- `catalog`: embedded `models.json` + DB overrides; `Lookup(ref) (ModelInfo, error)`.
- `ledger`: decorator writing `llm_calls{id, run_id, item_id, step, conversation_id, provider, model, input_tokens, output_tokens, usd, latency_ms, status, error}`.
- `limiter`: `golang.org/x/time/rate` per `provider:model` from RPM/TPM.
- `retry`: backoff on 429/5xx/timeouts, respects `Retry-After`.
- `recordreplay`: decorator; mode `record` writes `testdata/llm/<sha256(request)>.json`, mode `replay` reads; tests fail on missing fixture with the request dumped for recording. Used for content-generation fixtures.
- `fake`: scripted `gollem.LLMClient` (Archond `internal/infra/llm/fake.go`): directives in the input text (`TOOL:name{json}`, `ERROR:code`, `FAKE: final answer`) drive tool calls, errors, and answers without network. Used for agent-runtime tests and for the `agent.useFake` setting in dev builds.
- Agent runner (`transport/agent/runner.go`, shape of Archond `internal/infra/llm/runner.go`): `Run(ctx, RunSpec{Binding, ModelRef, System, Input, Tools, LoopLimit, HistoryBudget}) (*RunResult{Text, ToolCalls, Usage, USD}, error)`: resolves the model from the catalog, checks the prompt token budget, builds gollem options (`WithLoopLimit`, tool middlewares, bounded history, tools), executes, then writes the ledger row. History: `gollem.HistoryRepository` adapter over our `conversations/messages` tables wrapped in `boundedHistory` (character budget trim) as in Archond `internal/infra/llm/history.go`. Stream events `StreamEvent{Type(text|toolCall|toolResult|actionPending|actionResolved|done), ...}` are produced by content-stream and audit middlewares and forwarded by the Wails event bridge as `agent.*`.
- `ProviderFactory(ref) (gollem.LLMClient, error)` shared by `gollemclient` and the agent runner.

### 9.5 Images (`adapters/images`)
Port `type ImageProvider interface { Generate(ctx, Prompt) (Image, error) }` for `openai`, `gemini`; `type ImageSource interface { Pick(ctx, Query) ([]Image, error) }` for `wpmedia`, `localfile` (folder chosen by the user). Uploads through `wp.Client.UploadMedia`.

### 9.6 Importer (`adapters/importer`)
- `xlsx` (excelize) and `csv` readers → `[]Row{Cells map[string]string}` with header detection.
- `Mapping{ID, SiteID, Name, Columns map[Field]string, Options{PathPrefixStrip, KeywordSeparator, AnchorSeparator}}` where `Field ∈ {path, title, h1, primary_keyword, keywords, anchors, entity, entity_kind, parent_entity, related, page_kind, meta_title, meta_description, wp_type}`. `AutoDetect(headers []string) Mapping` uses a normalised-header alias table per field (shape of Archond `internal/modules/imports/parser/mapper.go`, English aliases only) so the preview opens pre-mapped.
- `Preview(rows, mapping) PreviewReport{Pages, Entities, Edges, Warnings, Errors}`; `Apply` in one transaction; missing intermediate paths auto-created; duplicate paths merged; entity names deduplicated case-insensitively.

## 10. Documentation and Orchestration Policy

Docs (the only `.md` files allowed):
- `CLAUDE.md` (≤ 60 lines, Archond `CLAUDE.md` style): numbered hard rules first, then workflow (layout, migrations, verification commands), then footguns one line each, then a dated "standing rulings" list, and a one-line product-positioning guardrail at the end. Cross-links to `docs/` instead of duplicating them.
- `docs/VISION.md`: product, boundaries, non-goals.
- `docs/ARCHITECTURE.md`: layers, dependency rule, key types, run engine protocol.
- `docs/CONVENTIONS.md`: code rules, naming, testing rules, commit format.
- `docs/CONTRACTS.md`: error format, events, pagination, long-running protocol, DTO rules.
- `docs/ORCHESTRATION.md`: agent policy, limits, task cycle, model allocation.
- `docs/STATUS.md`: the handoff (≤ 150 lines): current state, how to run, the phase table, known gaps, next steps. Updated in every phase commit.
- `docs/DECISIONS.md`: the decisions of every phase and the milestone reviews, moved verbatim out of STATUS as each phase ends.
- `docs/superpowers/specs/2026-09-17-postulator-v2-design.md`, `docs/superpowers/plans/2026-09-17-postulator-v2-roadmap.md`, `docs/superpowers/plans/2026-MM-DD-phase-N-<name>.md`.

Orchestration:
- Limits: ≤ 1 Fable + ≤ 2 Opus concurrently; Sonnet for web research/bulk reading only; no sub-agents.
- Task cycle per phase: orchestrator writes the task (English) → implementing agent writes `docs/superpowers/plans/…phase-N….md` using the writing-plans format → orchestrator + user approve → agent implements with TDD → separate Opus reviewer (code-review, verifies tests and dependency rule) → fixes → commit on `rewrite/v2` with conventional commits + `Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>` → `docs/STATUS.md` updated in the same commit.
- Definition of done per phase: `go build ./... && go vet ./... && go test -race ./...` green, coverage gate met for touched packages, dependency-rule test green, STATUS updated.

---

## 15. Archond Reference Map

Archond backend root: `C:\DICM\CODEBase\Golang\archond\backend`. Port shapes, not wiring. Never copy Asynq, Postgres/jet/pgx, fx, Fiber, credits/entitlements, or HTTP status plumbing.

| Our module | Archond files to read first | What to port |
|---|---|---|
| `kernel/errors` | `internal/shared/errors/errors.go`, `codes.go` | AppError shape, frozen codes, `Is` by code, clone-and-mutate helpers, captured stack |
| `kernel/paging` | `pkg/dbx/page.go`, `pkg/dbx/keyset.go`, `pkg/api/list.go` | `{o,d,v,i}` cursor, `SortKey[T]` kinds, keyset predicate builder, `List[T]`/`Slice[T]` envelope |
| `adapters/sqlite` | `pkg/dbx/result.go` | `Result[T]` monad at the repository boundary |
| `kernel/log` | `pkg/logger/logger.go` | zap builder, console+file cores, encoder switch |
| `kernel/settings` | `internal/settings/constructors.go` | settings-as-code with typed validators and defaults |
| `runtime` (Phase 5) | `internal/infra/jobs/pipeline/pipeline.go`, `checkpoint.go`, `definition.go`, `handler.go`, `daemon.go`, `repository.go` | Status/Transition/Fault/PauseReason, JSON checkpoint, step registry validation, CAS advance loop, wake/stall/reap sweeps |
| `runtime/steps` (Phases 6–7) | `internal/modules/content/jobs/stages.go`, `internal/modules/content/pipeline.go`, `internal/modules/content/models.go` | multi-stage LLM pipeline on the engine, `alreadyDone` idempotency, `VersionMeta{MetaTitle, MetaDescription, Excerpt, Slug, Images, InternalLinks, FAQ}` |
| `domain/content` (Phase 6) | `internal/modules/content/linking.go`, `internal/modules/content/cannibalization.go` | link insertion with decision per candidate, cannibalization verdict |
| `transport/agent`, `application/tools` (Phase 9) | `internal/infra/llm/runner.go`, `types.go`, `middleware.go`, `history.go`, `catalog.go`, `ledger.go`, `fake.go`; `internal/modules/agents/tools/tools.go`, `tools/start_clustering_run.go`; `internal/modules/agents/service/actions.go`; `internal/modules/agents/events.go` | RunSpec/RunResult, neutral Tool + gollem adapter, guard chain order, bounded history adapter, catalog resolve, ledger row shape, scripted fake client, one-tool-per-file registry, PendingAction propose/resolve, stream event names |
| `application/runs` read model (Phase 11) | `internal/modules/userjobs`, `internal/modules/pipelines/service/admin.go` | `RunView`/`EventView`/`RunDetail` DTO shapes |
| `domain/graph`, `domain/pagemap` (Phase 2) | `internal/modules/clusters/topicalmap.go`, `clusters/models.go`, `internal/modules/sitemaps/models.go`, `sitemaps/service/{builder,differ,apply}.go`, `clusters/proposals.go`, `overrides.go` | node roles and intents, tree flatten/export, drift diff against live site, AI-proposal review workflow |
| `adapters/importer` (Phase 8) | `internal/modules/imports/parser/mapper.go`, `factory.go`, `excel.go`, `csv.go` | header alias auto-detection, parser factory |
| `CLAUDE.md`, lint, hooks (Phase 0) | `CLAUDE.md`, `.golangci.yml`, `lefthook.yml` | session-contract style, rule rationale comments, hook set |
