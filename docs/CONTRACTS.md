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
| `SitesService` | `Create Update Delete Get List TestConnection` |
| `GraphService` | `LoadGraph CreateEntity UpdateEntity DeleteEntity GetEntity ListEntities SetAnchors AddEdge ApproveEdge RejectEdge DeleteEdge ListEdges RecomputeScores ProposeFromPages PreviewFromPages ProposeFromKeywords ApplyProposals ProposeRelated MoveEntity CardProposals AcceptCard DismissCard AdoptParents` |
| `PagesService` | `Create Update Delete Get List Count Tree MapToEntity Unmap SetCanonical PreviewLink SetType ResolveSelection SetTypes SetTemplates Model ModelSummary WriterPack SiteMap` |
| `TemplatesService` | `CreateTemplate UpdateTemplate DeleteTemplate GetTemplate ListTemplates SetOverride DeleteOverride ResolveForPage ListPageTypes CreatePageType SavePageType DeletePageType SetScope ListRules SaveRule DeleteRule ReorderRules TestRule PreviewClassification PreviewClassificationPages ApplyClassification` |
| `RunsService` | `Start Estimate Parents Get List ListItems ListEvents GetArtifact ListArtifacts Pause Resume Cancel RetryStep Regenerate RevertRun` |
| `FindingsService` | `List Get Counts Explain Resolve ResolveMany` |
| `SyncService` | `SyncSite CheckPlugin SavePluginPackage` |
| `ReportsService` | `SiteOverview LinkAudit LinkAuditPage PageReport RunReport JudgePage` |
| `ImportService` | `Inspect Preview Apply Export SaveMapping ListMappings DeleteMapping` |
| `ModelsService` | `ListModels UpsertModel DisableModel GetProfiles SetProfile TestProvider UsageSummary SpendReport ListCalls` |
| `AgentService` | `CreateConversation SetMode RenameConversation DeleteConversation Send Status Confirm Cancel ListConversations ListMessages ListPendingActions ConfirmPlan ListPlans` |
| `SchedulesService` | `Create Update Delete Get List Enable Disable RunNow` |
| `ToolsService` | `List` |
| `BrowserService` | `Open Locate` |
| `SupportService` | `WriteBundle` |
| `SettingsService` | `Schema Get Set SetProviderKey ProviderKeys DeleteProviderKey LockState Lock Unlock SetMasterPassword ExportBackup ImportBackup` |

A hundred and fifty-three methods in seventeen services since phase 18; the stage 4 to phase 18
methods are described after this paragraph. Phase 18 adds `AgentService.ConfirmPlan{planId,
steps: [{stepId, approve, edits?: {name: value}}]}` and `ListPlans{conversationId}`, both
answering `tools.PlanView` (`{id, title, status, steps: [{number, id, tool, summary, reason,
dependsOn (numbers from 1), status, estimateUsd?, estimateTokens?, error?, runId?, edits: [{name,
value, choices: [{value, label}]}]}]}`); `GraphService.AdoptParents{siteId, under?, entityIds?,
hash?, cursor?, limit?, apply?}`, a preview of `{steps: page of {entityId, name, path, from, to?,
toPath?, skip?}, hash, adopting, skipped, applied}` that applies under its hash (a moved one is
`CONFLICT`); `SupportService.WriteBundle{siteId, path}`, answering `{path, bytes}`;
`RunsService.Start` and `Estimate` take `brief?`, `directives?` and `addEdges?`, a must-link the
graph does not connect being `INVALID` on `directives.mustLink` with `{from, to}`; and
`PagesService.WriterPack` takes `brief?`, `directive?` or `runId?`. `MapNode` and `PageModel`'s
parent carry `parentId` and, when it is empty, `why: {reason, crumb?, candidates?, loop?, parent?,
findingId?}`. Stage 2a took the six policy methods out and put five page-type
methods and `PagesService.SetType` in, and 2b adds `PagesService.WriterPack{pageId, field?, budget?}`,
exactly what the writer would be sent for a page and field (`application/sitemodel`'s pack: its
sections in priority order with their token counts, as `Fit` left them under the role's input
budget, and what it dropped), resolved through the service's third source, and the three card
proposal methods: `GraphService.CardProposals{siteId, cursor?, limit?}` pages the open
`entity_card_proposed` findings (each with the card as it is, the card proposed, the reason and
`changed` when the card moved since, and the `proposalHash` it was shown with),
`AcceptCard{findingId, proposalHash}` writes the proposed card and settles the finding, refusing
with `Conflict` and the reason `card_changed` a card that moved since it was proposed and with
`proposal_changed` a proposal filed again since it was shown, and `DismissCard{findingId}` settles it dismissed, so the same proposal never reopens. `TemplatesService.ListPageTypes{siteId}` answers the site's
types with every scope (stored or the preset's default) and the field catalog;
`CreatePageType{siteId, name, preset, templateId?}`, `SavePageType{id, name?, templateId?}`,
`DeletePageType{id, reassignTo}` (the last type of a preset is refused) and
`SetScope{pageTypeId, field, …}` change them, resolving `application/pagetypes` through the
service's second source; `PagesService.SetType{pageId, pageTypeId}` types a page as the user's.
`GraphService.LoadGraph` names each mapped page's `pageTypeId`. `PagesService.Model{pageId}` answers a page as the model
(`observe.ModelView`: identity, the parent with its reason and method, the render's root, regions
and trails with each crumb's link, redirect and page, typed links, the blocks read-only, the open
findings, and since 2026-10-22 `fields`, each field row a run wrote as `{key, writtenRunId,
writtenAt, ours}`, `ours` false once the site's value moved from what the run wrote) and
`ModelSummary{siteId}` the site's (`observe.Summary`); both resolve
`application/observe` through the service's second source. Where a use case answers with bytes the service writes them
to the path the request names and returns it, because the webview has no filesystem;
`SyncService.SavePluginPackage{path}` is the only such method.

`FindingsService` (stage 4.3b) reads and settles the findings through `application/findings`, the
one use case a person and the agent share. `List{siteId, status?, code?, pageId?, pageTypeId?,
under?, cursor?, limit?}` pages them with their pages named (`under` is a node of the Site map, its
whole subtree); `Counts{siteId, status?, pageTypeId?, under?}` answers each code's count and
severity; `Get{findingId}` one finding; `Explain{siteId, findingIds, option, pageId?, pageTypeId?}`
the one sentence of what an option does (`findings.Explain`, the agent's words too) and its kind.
`Resolve{findingId, option, pageId?, pageTypeId?, proposalHash?}` settles one finding; a `run`
option settles nothing and answers the `StartRequest` or the `SyncSiteRequest` it asks for, and a
`proposal` option (`propose_entity`) answers `proposals`, the site preview's
`PreviewFromPagesRequest`. `ResolveMany{siteId, code, option, findingIds, pageTypeId?}` takes only
`finding.BulkOptions()` (`ignore accept_site accept_workbook set_type propose_entity`) over at most
1,000 findings of one code in one unit; the inbox sends a larger set in batches and says what a
refused batch left unsettled. `By` is `json:"-"` on both: a bound call is the person's.

**Classification** (phase 17). `TemplatesService.ListRules{siteId}` answers the ordered rules and the
fallback; `SaveRule{id?, siteId, name, enabled, typeId, conditions}` adds or edits one (the fallback
takes only its type), refused on the field it names (`conditions.urls[0].value`); `DeleteRule{id}`,
`ReorderRules{siteId, ruleIds}`; `TestRule{siteId, pageId, ruleId? | rule?}` answers each
condition's verdict, the page's facts and its outcome. `PreviewClassification{siteId}` answers the
groups (from, to, pages, rules, samples, `dependsOn`) with a `hash` and the kept, waiting and
by-default counts; `PreviewClassificationPages{siteId, from, to, status?, cursor?, limit?}` pages a
group; `ApplyClassification{siteId, hash, groups?}` writes them, `CONFLICT` with `preview_moved` when
the hash moved and `INVALID` with `group_depends` naming a group left out. `PagesService.Count` takes
the list's filters; `ResolveSelection{siteId, pageIds?, subtrees?, filter?, seen?, ruleId?, except?,
exceptSubtrees?}` answers the ids and each one's path (at most 10,000, `CONFLICT` when the filter's
count moved from `seen`); `SetTypes{selection, pageTypeId | null}` types them by hand, role-guarded,
or hands them back to the rules; `SetTemplates{selection, templateId | null}` sets or clears their
own template. A page answers `typeRuleId`; a type answers `role`, `overrides` and
`suggestedTemplateId`; `RunsService.Estimate` and `Start` answer `fields`, the fields a run writes
when it is scoped, a type with its body off scoping it.

`PagesService.SiteMap{siteId, parentId?, search?, pageTypeId?, within?, cursor?, limit?}` (stage 4.3b, over
`page_map` since phase 15) answers a node's children a page at a time, a store category folded into
the hub of its name (`twin`), with each node's type, parent reason and method, body state, links in
and out, open findings, children and descendants; a search or a type answers flat matches with their
trails. `RunsService.Parents{siteId, pageIds}` (phase 15) answers the planned ancestors a selection
writes first, which the run start's picker adds. `GraphService.PreviewFromPages` takes `source`
(`model`, the default, or `site`): a site preview names each page's entity from its H1 or title,
scopes it under its nearest ancestor's entity or the twin hub of its store category, makes no model
call, and marks a name clash (`no_name`, `name_in_batch`, `name_carried`) unticked rather than
refusing the batch; `ApplyProposals` takes the preview's `source` and writes a site proposal's
parent edge approved. `ProposeFromPages` stays the model's. `RunsService.Start` takes `fields` (the field keys a run is scoped to; the
steps they do not need are disabled) and `rewrite` (the pages whose client text a person confirmed
may be rewritten).

`SitesService.Update{…, settings?}` takes a site's settings since 2026-10-18 (`userAgent`, the
`observe` block's `renderConcurrency` 1–16 and `fullWalkEvery` ≥ 1, `linkUrlForm`), keeping the
observation's own bookkeeping, and `Get` answers them normalized as `settings`; every request the
registry sends for that site carries its `userAgent`, the connection test of a saved site
included. `SyncService.SyncSite{siteId, full?, renderRescan?}` walks the whole site or fetches
every published render again; without either the walk is incremental and reads only what changed
since the previous one.

`GraphService.MoveEntity{entityId, newParentId, keepBoth?}` adds the parent edge, checks the
graph is still acyclic and drops the approved parent edges it replaces inside one unit of
work, and answers `removedEdgeIds`; running it again is quiet. `RunsService.RevertRun{runId,
pageId?, fields?}` enqueues the revert of a finished run and answers `{runId}`; it refuses a revert
of a revert, a run that has not finished ("cancel it first"), a second revert while another is
under way, and a run that wrote nothing to the site. `pageId` names one page the run wrote and
`fields` the keys to put back on it (each known to the catalog and off the deny list; `body` or
`product_description` for the body); a field the client changed since is handed back, never
written over. The agent's `runs_revert` takes `{runId}` alone.

`ReportsService.JudgePage{pageId}` is synchronous: it pulls the live page, runs the shared
judge rubric against it and answers with the report, one model call inside the request.
`PageReport` and `RunReport` read what a run already recorded, and a page report carries, as
`product`, the short description and attributes of the newest draft (a run before 2026-10-20), else
`{shortDescription}` from the newest `fields` artifact, and nothing else of either; `SiteOverview`,
`LinkAudit{siteId}` and `LinkAuditPage{pageId}` read the graph and the page map. The audit
plans every mapped page's link targets with the rules of the page's resolved template and the
site's effective policy, exactly as `resolve_context` does, and answers one summary row per
non-archived page; the page detail names each target, whether a stored link satisfies it and
with which anchor, and every stored link the graph did not ask for. Both are unpaged reads of
one site, like `Tree` and `LoadGraph`. Each owed link carries a `state` from `linkStates` in
`vocab.ts` (`placed`, `target_unpublished`, `missing`, `awaiting_target`, `awaiting_page`,
`blocked`) and `targetOnSite`; a row carries `onSite` (the page has a WordPress id), and
`missing` and `missingRequired` count real misses only, while `pending` counts the links that
wait for a page to be written or published and `unpublished` the placed links to a page that is
not on the site yet. `orphan` holds only for a page on the site, in the audit and in
`SiteOverview` alike. A blocked target carries `blockedReason` from `linkBlockedReasons`:
`no_canonical_page` for an entity that has pages but no canonical one, which stays required, and
`no_page` for an entity no page carries at all, which is optional and passed through to its own
parents or children. Entity names in the audit are labels: a name another entity shares carries
its parent.

`ImportService.Preview` and `Apply` answer a report whose `columns` say, for every header of the
sheet in order, what it became (`use` from `importColumnUses`: `field` with the `field` it fills,
`level`, `note`, `indent` or `ignored`); `groups` list the chains the root level columns name,
with the page each took (`page`, empty for none) and the rows under it; each entity carries its
`parent`. A keyword cell reads `text (volume), text`, separated by `, ; |` or a line break outside
brackets; `levelColumns` and `noteColumns` are mapping options, and `own_entity` is a field. Only
a level column headed `Root Entity` or `Root` is a level, and it makes entity groups; the
detection names no other. Any other column a saved mapping or the agent puts in `levelColumns`,
Category, `Root Category` and `Brand` among them, stays stored, is `ignored` and makes nothing. A
mapping maps something only through a field, an indent column or a root column: one that names
none of them is detected from its sheet's headers on a preview and refused `INVALID` with
`details.field = columns` when saved. A row's entity takes its parent from its parent cell,
else from its URL parent when that parent lies inside the row's own root group, else from the
group, else from the URL tree. Findings `ambiguous_parent`, `ambiguous_entity` and `scope_clash`
block an apply like an unknown parent; `bad_volume`, `technical_parent`, `group_without_page` and
`unknown_own_entity` are warnings.

A whole workbook is one request: `PreviewRequest` and `ApplyRequest` take `sheets: [{sheet,
mapping}]` in place of the one-sheet `mapping`, which stays; `sheets` beside a `mapping` that names
columns, root levels, indents or an id is `INVALID` with `details.field = sheets`, and so is an unknown
sheet or one named twice. The sheets are planned in the workbook's order whatever order the
request gives, share the `import.maxRows` budget and are written in one transaction;
`saveMappingAs` keeps one mapping per sheet as `"<name> / <sheet>"`. A mapping with an `id` and no columns loads that saved mapping of the
same site, and one with no columns, root levels or indents is detected from its sheet's headers.
`InspectResponse.sheets[i].detected` is each sheet's own detected mapping, with `options.rowType`
`pages` or `products`. Every report item and finding carries its `sheet`, and `summary.sheets`
lists the sheets read.

`PagesService.List` takes `includeDescendants` with an `entityId` and keeps the pages of that
entity and of every entity under it; it refuses the flag alone. A page carries `notes`, each
`{label, text}`, from the note columns of an import.

Every method but `HealthService.Ping` and the four lock methods of `SettingsService`
answers `LOCKED` while a master password is set and the application has not been unlocked,
because a service resolves its use case per call and the composition does not exist yet.

`SettingsService.Schema` renders the `kernel/settings` declarations as
`{key, group, type, default, min?, max?, enum?, nonEmpty?}`, which is what a settings
screen draws itself from. `Get{key}` answers the stored value or the declared default with
`isDefault`; `Set{key, value}` validates against the declaration before it writes, then
re-applies the whole stored map so the running process sees the new value. An undeclared
key is `NOT_FOUND`, a rejected value is `INVALID` and nothing is written. No setting holds
a secret: `SetProviderKey{provider, apiKey}` delegates to `models.SetProviderKey`, which
puts the key in the encrypted store and answers with the provider name alone. It is the
only method that accepts a credential.

`LockState{}` answers `{locked, protected}`. `SetMasterPassword{current, new}` rewraps the
master key, an empty `new` removes the password and returns to plain DPAPI, and a wrong
`current` is `LOCKED`. `Unlock{password}` composes the store and every service and emits
`app.unlocked`; `Lock{}` closes the store, zeroes the key and emits `app.locked`, and it
refuses with `INVALID` while no password is set, because nothing could unlock it again.
`ExportBackup{path, password}` writes one Argon2id and AES-256-GCM archive holding a
consistent snapshot of the database and answers with its size; the master key is not in it.
`ImportBackup{path, password}` decrypts and checks the whole archive first, then stops the
engine, the scheduler and the agent, copies the snapshot into the encrypted database and
composes the core again from what it restored. A wrong password or a truncated file is
`INVALID` and nothing is touched.

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
  `template.TemplateSpec`, the `llm` catalog types, `site.Settings`, the `render` extraction's
  parts and `pagemap`'s trail and identity, whose persisted form is JSON.
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
control is `Get Pause Resume Cancel RetryStep RevertRun`. Progress is read two ways and both are
required: `RunsService.ListEvents(runId, sinceSeq, limit)` is the durable log, gapless per
run, and the live bus pushes the same records. Live delivery is best-effort — v3
dispatches to the windows that exist at that instant and buffers nothing, so an event
emitted while no window exists is dropped and a page reload discards the listener table.
A client that has seen `seq` asks for everything after it on reconnect; this catch-up is
mandatory, not an optimisation.

## Run kinds and their recipes

`run.Kind.Recipe()` is the one answer to whether a kind names its own steps. A kind that owns
one hands it out and `runs.Start` refuses a request recipe that disagrees with it, naming the
steps it will run; `runs.Start` also refuses any recipe, the request's or the template's, that
enables a step another kind owns, for every kind but `custom`.

| kind | recipe |
|---|---|
| `generate` | the template's, or `run.GenerateRecipe()` (which carries `generate_images`) when neither the request nor the template names one |
| `relink` | `resolve_context relink_page sync_back report` |
| `repair` | none since 2026-10-18: the hierarchy is read from the site and never written, so `runs.Start` refuses the kind and names `relink` instead; the word stays in `runs.kind` and `runKinds` so old rows decode |
| `sync` | `sync_site`, which takes the params `full` (walk the whole site) and `renderRescan` (fetch every published render again) |
| `revert` | `revert`, and `runs.Start` refuses the kind: only `RunsService.RevertRun` can set `parentRunId` |
| `edit` | `edit_page` over one page, carrying one field change or one block operation; only `runs.StartEdit` starts it (the agent's `pages_fields` and `pages_blocks`), and its write waits for the person in either mode; `CHECK` rebuilt by 0073 |
| `audit`, `import` | none of its own |
| `custom` | none of its own, and the one kind exempt from every step rule |

`sync_site`, `relink_page`, `revert` and `edit_page` are `run.PerKindStepName` values,
a type of their own, so none of them reaches `vocab.ts`, the blank template's recipe or the
three tool step enums; `run.PerKindStep(name)` is the domain predicate and
`run.StepNames()` is what a template recipe may draw from.

`RunsService.Estimate` answers zero for `relink` and `sync`, because none of their
steps declares a model role. It prices every target on the template that resolves for it and
carries `findings []{severity, code, message, pageId?, path?}`: `unpriced_step` names an
enabled step that declares no ceiling and no model, `template_unresolved`, `entity_missing`,
`required_target_unplaced`, `provider_key_missing` and `model_unresolved` are graded `error`
and `RunsService.Start` refuses the run with `INVALID` and `details.findings` while they
stand, and `recipe_differs`, `model_unknown`, `image_source_unavailable` and `plugin_missing`
are warnings, as is `images_step_off`, raised per page whose template asks for images the run's
recipe will not draw. Over a product, `commerce_unknown`, `commerce_absent`, `commerce_forbidden`,
`product_needs_plugin`, `product_not_in_store`, `product_edited_live` and
`product_category_unwritable` are errors and `product_outputs_missing` and
`product_outputs_ignored` warnings.
`model_provider_removed` warns of a template that pins a model of a provider Postulator no longer
works with and names the model the role uses instead. Each step is priced at its role's service tier and, on a model that
reasons, with half the allowance of the role's effort added to its output. `Budget` carries
`maxUsd` and `maxTokens` and either one pauses the run with `budget_exceeded`, and a negative one
is refused.

`StartRequest.templateId` assigns that template to the chosen pages: `Estimate` prices and
preflights every chosen page on it without writing anything, and `Start`, once the estimate lets
the run through and its recipe checks out, moves the chosen pages to it before it queues the run.
The parents a run adds keep their own templates, a kind that owns its recipe assigns nothing, and
a schedule's `templateId` does the same on every firing. `ResolveForPageRequest.templateId`
resolves that template in place of the page's own and refuses one of another site with
`INVALID`.

`RunsService.RetryStep{itemId, acceptFindings?, fields?}` requeues the current step; with
`acceptFindings` a page held at `validate` goes on with its findings as they are, and an item
paused `awaiting_review` writes the plan it showed. Only a person accepts a review; the agent's
call is refused with `CONFLICT`. `fields` (since 2026-10-22) narrows a review's acceptance to the
keys a person ticked, each among the item's newest `edit_plan` (its field changes and its page's
`body` key); the rest are not written and the publish result names them in `declined`. An empty
list, `fields` without `acceptFindings`, on an item no review holds, or on a review that changes
bodies only (a relink's) is refused with `INVALID`. The agent's `runs_retry_step` takes `{itemId,
acceptFindings?}` alone. An item
carries `seq`, its place in the run, and `blockedBy`, the item of the parent it is queued
behind, and `waitingFor` names that parent's page whenever the item has a blocker.

A run's status may become `paused` with `pauseReason: needs_human` without any item failing,
because a run whose every remaining item is waiting for a person has nothing left to advance.

## Artifacts

`judge_report.score` and `final_report.score` are **optional**: absent means nothing scored
the page, where a final report used to default to a perfect `1` and a judge that could not be
reached used to store a `0`. Both carry `findings`, as `meta` and `images` now do; `images` is
written even when the template asks for no image, because the step's `Produces` says it is.
`publish_result` carries `previousContent`, `previousContentHash` and `previousMeta` — what
the write replaced, kept only for an update and only where the site can answer — and
`relink_result` carries `before {hash, html}` per neighbour. A relink run writes its own
`RelinkPageResult` under the same `relink_result` kind, with `placed[]` rather than
`neighbors`, so a revert does not try to restore a page nothing wrote to. `revert_result` is
the artifact kind a revert produces. `ImagesResult` no longer carries `skipped`.

Since stage 3 (2026-10-20 to 22) `publish_result` carries `version` (3) and the diff of what it
wrote: `fields [{key, before, beforeHash, after, afterHash}]`, each `after` being what the site
answered it stored, and `blocks [{kind, hash, raw, owner, change}]`, the body's blocks **as the
site stored them** (`owner` one of `blockOwners`: `ours`, `client`, or `site` for a block the
site's own save added; `change` one of `blockChanges`: `kept`, `added`, `removed`), with
`writtenMeta` (the SEO meta the plugin stored) and `declined` (the keys a person left unticked in
a review). A result without `version` is reverted whole. A draft run's item paused
`awaiting_review` carries `edit_plan` `{pages: [{pageId, path, body?, fields, blocks, wpId}],
hash}`, `body` naming the key the body is written under when its blocks change. `revert_result`
carries `fields [{key, outcome, detail?}]`, a field `kept` with a reason being one handed back.

New warning findings, each carrying `details.pageId` and `details.path`: `judge_unavailable`
(`reason`), `artifact_purged` (`kind`), `image_not_placed` (`reason`), `meta_not_written`
(`fields`).

`images` carries `wanted`, the images the page asked for, and `images_short` names a library
that picked fewer. `relink_result.neighbors[]` carries `sentence` when a plain sentence was
written to carry the link, and the result counts `missing`, the neighbors that owe the page a
link that could not be placed; a relink run's `placed[]` carries `sentence` likewise.
`final_report.notice` is the sentence that says what the page lacks, empty when it lacks
nothing. Warning findings added on 2026-09-25: `target_not_published` (`targetPageId`,
`relation`) from `validate` and `relink_page`, `images_short`, `relink_phrase_templated`
(`sentence`, `targetPageId`), `neighbor_link_missing` (`reason`, `targetPageId`), and
`target_missing` from `relink_page` when the page's budget is spent.

A writer that runs out of room twice holds the page `needs_human` with a note that says to lower
the template's word counts or choose another model.

## Events

`internal/application/events` owns the names, the payload structs and the envelope;
`frontend/src/generated/events.ts` is rendered from it by `task events` and a Go test
fails when the committed file is stale. The generated module is never edited by hand and
carries no header comment, because this repository forbids comments in TypeScript. The
envelope is

```json
{"type":"step.done","seq":42,"runId":"<uuid>","at":"2026-09-17T10:30:00Z","payload":{}}
```

`step.done` carries `message`, the sentence the step returned, and `step.retrying` carries
`code` and `message`, the fault that made the engine try again, beside `attempt` and `afterMs`.
`item.done` carries `note`, what the finished page lacks as the report step worded it, the same
sentence the item's `note` keeps; it is empty for a page that lacks nothing.

`runId` is present on run events only, `at` is RFC3339 UTC, and `seq` is per run and
gapless. For application events `seq` is a counter held by the `EventBridge`, so it is
process-wide only because the composition root constructs exactly one bridge; a second
bridge would restart the numbering.

`internal/transport/wails.EventBridge` is the only emitter: `Publish(type, payload)` for
application events, `PublishRun(runId, seq, type, payload)` for run events, both rejecting
an unknown name or a mismatched payload. `application.RegisterEvent` is deliberately
unused.

The generated module also exports `eventTypes`, the same names once more as a readonly
tuple, so a window can build a handler map the compiler checks for exhaustiveness.

The names and their payloads are the `EventType` union and the `EventPayloads` map in
[`frontend/src/generated/events.ts`](../frontend/src/generated/events.ts), which is the
registry rendered. Run events are the `run.* item.* step.*` families plus `llm.usage`;
application events are `graph.changed pages.changed templates.changed sites.changed
schedules.changed settings.changed app.locked app.unlocked files.dropped` and the agent family `agent.delta agent.tool.started agent.tool.finished
agent.confirm.requested agent.confirm.resolved agent.plan.proposed agent.plan.updated agent.titled agent.usage agent.waiting agent.done`, whose payloads all carry
`conversationId` because a window may hold more than one conversation. The frontend
subscribes with `on(type, handler)` from `frontend/src/lib/events.ts`, which narrows
`payload` to the declared type. Events only travel Go → JS; every frontend-initiated
action is a bound method call.

A turn's spend is announced per model call, not per turn: `agent.usage{conversationId,
messageId, provider, model, round, inputTokens, cachedInputTokens, outputTokens, usd}` is
published once per round, and `agent.done` carries the authoritative totals with
`cachedInputTokens` and `calls`, the number of model calls the turn made. A round the
provider held back announces `agent.waiting{conversationId, messageId, reason, attempt,
afterMs}`, where `reason` is the kernel code of the refusal, so a turn that is waiting is
not silence. `llm.usage` stays a run event: it is published with a run sequence and an agent turn
has no run. It carries `tier`, the service tier the provider served, and `reasoningTokens` and
`cacheWriteTokens` beside `promptTokens` and `completionTokens`, both parts of those counts rather
than additions to them, and it is published only for a call that used tokens, so a refused call is
seen through the item and run events instead. `run.budget_exceeded` carries `spentTokens` and
`budgetTokens` beside `spentUsd` and `budgetUsd`, and is published for a token overrun as
well as a money one.

## Opening a link

Nothing in this application opens the system browser. `BrowserService.Open{url}` refuses
anything that is not `http` or `https` with `INVALID`, and otherwise starts Tor Browser
detached on that address. `Locate{}` answers `{path, source, installed}` where `source` is
`setting` when the `browser.torPath` setting named it, `detected` when it was found under
`%USERPROFILE%\Desktop`, `%USERPROFILE%\OneDrive\Desktop`, `%LOCALAPPDATA%`,
`%PROGRAMFILES%` or `%USERPROFILE%\Downloads` as `<root>\Tor Browser\Browser\firefox.exe`,
and empty when nothing is installed. A candidate counts only when a `TorBrowser\` directory
sits beside `firefox.exe`, which is what tells Tor Browser apart from a plain Firefox, and a
configured path that fails that test is ignored rather than trusted. Nothing installed is
`INVALID` with `details.code = tor_missing`; a launch that failed is `EXTERNAL`. The address
never appears in a failure or a log line, because a preview link carries a signed token.

## Page preview

`PagesService.PreviewLink{pageId}` answers `{url, expiresAt, kind}`. `kind` is `public` for a
published page, whose `url` is the site's base URL and the page path and whose `expiresAt` is
null, and no site call is made; it is `preview` for any other page on the site, whose `url` is an
hour-long signed link the companion plugin issued and rotates on every call. A page without a
WordPress id answers `INVALID` with `details.field = wpId`, an archived one with `details.field =
status`. A site that cannot issue a link answers `INVALID` with `details.code` set to
`plugin_missing` or, for a plugin older than 1.1.0, `plugin_outdated` with `details.capability`,
`details.version` (the installed one) and `details.required` (what the client needs).

`SyncService.CheckPlugin{siteId}` answers `{plugin, commerce}`; `plugin` is `{installed, version,
requiredVersion, outdated, capabilities, seoPlugin}`. `outdated` is true when the manifest's
version is below `wp.PluginMinimum` (1.5.1 since 2026-10-22, 1.5.0 since 2026-10-20, 1.4.1 since 2026-10-18) or a capability the client needs is
missing; `requiredVersion` names the bundled zip, `wp.PluginVersion` (1.5.2 since 2026-10-24, the
1.5.1 contract on PHP 7.2 and WordPress 5.7), and the Sites panel then says to upload it over the old one, which WordPress
offers as "Replace current with uploaded". The verdict is the probe's and is never stored.

## Products

A product is a page row whose `wpType` is `product`, created by the client in WooCommerce and
edited where it stands. `Site.commerce` is one of `siteCommerces` in `vocab.ts` (empty until the
store is asked, then `absent`, `forbidden` or `ready`) and `CheckPluginResponse.commerce` answers
the store as the check found it. `Page.plannedPath` is the address the sheet gave a product whose
address the store decides, empty when the two agree. `ImportOptions.rowType` is one of
`importRowTypes` (`pages`, `products`, `kind`), and a `PreviewPage` of a product carries
`plannedPath`, `storeName` and `matchedBy` (`path`, `slug` or `name`); the import findings
`product_not_in_store`, `product_row_left`, `wp_type_kept` and `intermediate_level` are warnings.

A publish over a product writes the description through the plugin's raw route and the short
description, a first image where it has none and each variation's description (one
`POST products/{id}/variations/{vid}` carrying `description` alone) through WooCommerce's REST API,
and its result carries `previousProduct {shortDescription, writtenShort, attributes, written,
images, added, imageId, shortWritten, attributesSent, lists}` (`lists` holds `upsells` and
`cross_sells` as `{key, before, written}`, each a comma-joined list of product ids), `previousVariations [{key, before,
written}]` (the key is `variation_description:<variation id>`) and `previousMedia [{key, before,
written}]` (`image_alt:<media id>` or `image_title:<media id>`, written through `POST
wp/v2/media/{id}` with `alt_text` and `title` alone, only for an attachment whose parent is the
product), what the product held and what the run wrote, which the revert reads; an image attached
elsewhere is named in `skipped`; a page's or a product's custom fields are `previousFields [{key,
before, written, provider}]` (`meta:<key>`, written through the plugin's `PUT /meta` or, for a
product whose scope names the store, `meta_data` by key, never a key the deny list names); `attributes`, `written`, `added` and `attributesSent` are an
older run's, since no run writes an attribute now. A sync result may carry `product_description_hidden` or `product_page_unread`, both
warnings. `PagesService.Delete{onSite: true}` refuses a product or a product category with
`details.field = onSite`. The name, price, stock, SKU, status, slug and categories of a product
are never written.

## Models and spend

The catalog is OpenAI's alone. `UpsertModel` takes `provider` `openai` and refuses another on that
field; it carries the input, cached input, cache write and output prices and the four flex prices
(`flexInputUsdPerM`, `flexCachedInputUsdPerM`, `flexCacheWriteUsdPerM`, `flexOutputUsdPerM`, the
input and output declared together), and no reasoning effort. A model offers flex when its flex
input price is above zero, and a price left at zero on a built-in model reads as the built-in
price. `SetProfile` refuses another provider. The effort and the tier of each role are settings,
`llm.effort.<writer|editor|linker|judge|titler>` (`none low medium high xhigh`) and
`llm.tier.<role>` (`standard flex`), beside `llm.flexPatience`; `serviceTiers` in `vocab.ts` names
the tier a call was served at, `default` or `flex`.

`SpendReport{days?, runId?}` totals the ledger over the last 1 to 366 days, 30 when left out, or
one run whole and step by step, with `since` null. It answers `{since, days, runId, totals,
slices}`: `totals` counts `calls` (the `ok` rows), `failed`, `input`, `cachedInput`, `cacheWrite`,
`output`, `reasoning` and `usd` (every row) with `cachedShare`, `reasoningShare` and `flexShare`,
and `slices` holds one row per `purpose` (from `spendPurposes`: `run chat title probe graph audit
other`), `provider`, `model`, `tier` and, for a run, `step`, dearest first. It is an aggregate whose
rows the catalog bounds, not a list, so it takes no cursor. `ListCalls{cursor?, limit?, sort?,
runId?, conversationId?}` pages the ledger by cursor, newest first unless a sort says otherwise,
refuses a run and a conversation together, and answers each call with its purpose, step, provider,
model, tier, tokens (`input cachedInput cacheWrite output reasoning total`), `usd`, `latencyMs`,
`status`, `errorCode` and the run, item or conversation it served.

## The companion plugin's version

The shipped plugin is **1.5.2** (the 1.5.1 contract, declaring PHP 7.2 and WordPress 5.7; 1.5.1 is
still accepted) and advertises `bulk seo_meta seo_meta_read content_hash raw
preview rendered meta terms settings meta_write guarded_write`, every one of which the client
requires (`wp.requiredCapabilities`, the fake's defaults, the PHP `CAPABILITIES`, `openapi.yaml`
and three pinned test lists move together). A capability the manifest does not name is refused
from the cached manifest, before any request, and **an outdated plugin writes nothing**: `PutRaw`,
`SetSEOMeta`, `ReplaceSEOMeta` and `PutMeta` answer `plugin_outdated`, since an older SEO route
ignores the expected hashes and an older raw write answers no content.

The floors are proven, not declared: `wp-plugin/tests/functions.php` requires every function the
plugin calls to be its own, on `testdata/wordpress.json` with its `@since`, or in the PHP running the
scan, and the list's newest core `@since` to be the header's `Requires at least` (5.7.0, `wp_robots`);
the lint and the tests run under `E_ALL` on PHP 7.2.34, 7.4.33 and 8.5.12, and the contract suite
ran on WordPress 5.7.2 with PHP 7.3, WooCommerce 6.5.2 and Rank Math 1.0.98.1 (`STAND=legacy`).
On a WordPress below 6.0 or a WooCommerce below 6.9, core's REST names neither the front page nor
the store's cart, checkout and account pages; only the plugin's `/settings` answers them.

Stage 3's routes, every write under the hash it was read with (`wp-plugin/openapi.yaml` is the
contract):
- `PUT /meta/{id}` takes each key with the hash of the value it was read with (null for one the
  post did not carry), refuses the deny set (`x-denied-meta`: the store's keys behind the denied
  fields, the `_`, `attribute_` and `rank_math_` prefixes, `total_sales`), a protected key, a key
  the user may not `edit_post_meta` and a stored value that is not one plain text, answers `409
  hash_mismatch` with `moved` naming each key that moved and then writes nothing, deletes on a
  null, and answers each key's stored value and hash.
- `PUT /seo-meta/{id}` takes `expected` per field and answers the hashes of what the SEO plugin
  stored after its own sanitizer.
- The raw `PUT /content/{id}/raw` answers the content WordPress stored beside its hash, which is
  what `page_blocks_written` records.
- The raw `GET` answers `saveStable`: whether a whole-post save by this user would leave the
  stored content, **title and excerpt** unchanged (since 1.5.1; 1.5.0 probed the content alone),
  each run through its own `pre_post_*` and `*_save_pre` filters, which is how kses reaches a user
  without `unfiltered_html`. Over a page that is not save-stable nothing that re-saves the post
  is sent: meta and SEO writes still go, and the item pauses `save_alters_client`.
- The manifest answers `unfilteredHtml`.

A link the model records as ours carries the hash of the block that holds it
(`page_links.block_hash`, migration 0070): the mark moves only with our own changes to that block,
so a link the client adds with our target and anchor is never ours, and a client's edit to a block
holding our link hands that link to him.

## Agent chat

`CreateConversation{siteId?, title?, mode}` opens a conversation in `confirm` or
`autonomous` mode and `SetMode` switches it; `RenameConversation{conversationId, title}`
retitles it and `DeleteConversation{conversationId}` stops a turn in flight and drops the
conversation with its messages, pending actions, tool calls and history. An untitled
conversation takes its first message as its title. `Send{conversationId, text}` returns
`{messageId, assistantMessageId}` at once and the turn runs behind it: the assistant id is
minted before the turn starts and is the id every event of that turn carries, so a window
that receives the terminal event before `Send` returns can still match the two.
`agent.delta` carries the streamed text, `agent.tool.started` and `agent.tool.finished` the
tool calls, `agent.done` the end of the turn, and `Cancel{conversationId}` stops one in
flight. `Status{conversationId}` answers `{running, messageId, startedAt, lastSeq}` from
the turn registry, and zeroes all but `running` when nothing is answering, which is how a
window that opened mid-turn learns there is one. `agent.done` carries a frozen `code`,
empty on success, and an `error` redacted exactly as a binding error is; `CANCELLED` is
the stop signal, whether the user stopped the turn or the `agent.turnTimeout` deadline did,
and the message says which. The turn runs under that deadline for every provider and a
panic underneath it ends the turn with `INTERNAL` rather than the process. In `confirm` mode a `write` or
`dangerous` tool does not run: it writes a pending action and emits
`agent.confirm.requested{conversationId, confirmationId, tool, args, risk, summary}`, which
`Confirm{actionId, approve}` settles and `agent.confirm.resolved` announces. A pending action keeps the
`risk` of its op (0084), so a card read back after a restart is badged by what it will do. History is
read with `ListConversations`, `ListMessages` and `ListPendingActions`, all of which
survive a restart; `ToolsService.List` is the capability list the UI shows.

What the model remembers of a conversation is stored apart from the transcript as
`{"format":"responses/2","items":[…]}`: the port's messages, call arguments masked, tool results
shortened to `agent.historyToolResultBytes`, trimmed by whole turns. A body in any other shape loads
empty, which is how a conversation from before 2026-10-03, and one from 2.6 that called the tools
before the fold (`responses/1`), keeps its transcript and starts the model afresh. Each round is sent at effort `none` on the default tier with the prompt cache key
`chat:<mode>`, the instructions static per mode and the site's counts as the developer message
that opens the turn. `agent.toolLoading` (`all` by default, or `deferred`) says whether every tool
goes out on every round or only the orienting reads, with the rest found by tool search; a turn
replays its own searches and stores none.

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

A tool is `{Def{Name, Description, Risk(read|write|dangerous), Schema}, Authorize, Check, Run,
Confirm, Describe, RiskOf, Price, Edits}` and `Binding{SiteID, ConversationID, RunID, Mode, Approved}`
scopes every call. Twenty-three tools since phase 18, one per job: `sites`, `sites_manage`,
`graph_read`, `graph_edit`, `graph_propose`, `pages_find`, `pages_tree`, `pages_get`, `pages_blocks`,
`pages_fields`, `pages_manage`, `pages_judge`, `links`, `pagetypes`, `pagetypes_classify`, `templates`,
`runs`, `runs_control`, `imports`, `findings`, `models`, `schedules` and `plans`. A read selects by a
`view` enum and a write by an `op` enum; `RiskOf` reads the op, so one tool reads, writes and deletes
at the risk of what it is asked. They measure 48,543 bytes and 11,084 `o200k_base` tokens of schema
resent on every round of every turn, which `TestTheToolSchemasFitTheirCeiling` holds against
`schemaCeilingBytes` (48,543) and `schemaCeilingTokens` (11,084) and logs with the eight widest.
`Describe` says in one sentence what a write's op will do, the sentence a confirmation card, a
pending action and a plan step carry; `Confirm` is the same with the meaning "ask the person whatever
the mode" (the trash, a type that rewrites, an import into one, an adoption, a proposal's write).
`Price` gives a run start's estimate and `Edits` the fields a person may change on a plan step (the
brief's lines, each directive's notes and must-link anchors, a field's value, a link's anchor, a type
from the site's types). A page addressed by id must belong to the conversation's site.
Every agent write to a page is a one-item `edit` run that waits for the person in either mode, and
so do `graph_edit op apply_proposals`, `op adopt_parents` and `graph_propose op pages`; resolving a
finding and starting a run follow the conversation's mode. A description names what a field takes,
which op takes it, its default and its limits, and no more; the chat instructions say once that an id
is used as a read tool returned it and that a field left out keeps its value.

`plans op propose{title, steps: [{tool, args (one JSON object as a string), reason, dependsOn
(numbers from 1)}]}` checks each step as its tool would check a confirmation (`Registry.Validate`),
stores the plan, emits `agent.plan.proposed` and answers `{status: "planProposed", planId, plan}`;
`op get` and `op list` read the conversation's plans. A plan waits for the person in either mode
(`AgentService.ConfirmPlan`); its steps then run through the registry, approved, each step emitting
`agent.plan.updated{conversationId, planId, status, stepId, stepStatus}`, and the conversation is
told the outcome in one message.

Every list a tool answers is fitted to the result cap before its cursor is issued (`tools.fitted`
halves the limit until the encoded page fits within the cap less a sixteenth), so following
`nextCursor` never skips a row; `internal/app/scale` walks the tree, the pages and the entities by
cursor at 5,003 pages with none missed and none twice.

With `agent.toolLoading` set to `deferred`, a round sends seven orienting reads whole (`sites
graph_read pages_find pages_tree pages_get links runs`) and every other tool inside its group, the
prefix of its name, as a namespace of `defer_loading` functions beside one `tool_search` tool, which
a test in `registry_test.go` logs without a ceiling (`go run ./cmd/tokenreport` prints it). It stays
off until it has been tried with a funded key.

The guard chain runs in this order and the order matters: `fence` wraps tool output as
untrusted data so a result cannot inject instructions into the model, `audit` writes the
ledger row and emits the stream event, `capResult` shortens an oversized answer before it
re-enters the model, `permission` checks the conversation allow-list and calls `Authorize`,
denying with `UNAUTHORIZED`. The audit middleware records what the tool answered; the fence
wraps the copy the model reads. `Fence`, `Permit` and `Cap` live in
`internal/application/agent`; the runner in transport calls them for every call the model makes
and `Confirm` calls them directly, so a confirmed tool is guarded exactly as the model's own call
is.

A result over the ceiling is shortened, not truncated: `Cap` halves the widest list, then the
longest string at a rune boundary, and only when neither helps falls back to
`{truncated, totalBytes, preview}`. The ordinary answer is
`{truncated, totalBytes, droppedItems: {path: n}, shortenedText: n, result: <the document>}`,
so a capped listing still decodes; a list it halved loses its `nextCursor`, keeps `hasMore` and
names its path in `cursorDropped`, so the model asks again with a smaller limit rather than skip the
rows the cap cut. The
in-turn ceiling is `agent.maxToolResultBytes`; what is replayed to the model on every later
round is the smaller `agent.historyToolResultBytes` (int, default 4096, range 512–65536,
group `agent`), read per turn so a change takes effect without a restart.

A tool call ends in one of five states, and the transcript reads each differently:
`running`, `ok`, `cut` (the answer was shortened, which names every list it dropped rows
from), `denied` (nothing was read or written) and `error` (handed back to the agent with its
reason). A live row carries the status from the event; a saved row carries `toolStatus` on
`agent.Message`, read back from the `tool_calls` ledger.

A field of a tool request is required only where its own schema says so, nested fields and
list items included: the rule is that a field is required only when the use case refuses its
absence, because the runner checks the arguments against the whole tree before the tool runs
(`transport/agent/arguments.go`, the check gollem used to make) and its refusal teaches the model
nothing. In a folded tool only `op` or `view` is required at the top, each op saying in the field
descriptions what it takes; a nested field is required where its parent's use case refuses its
absence, which `TestEveryRequiredFieldIsOneTheUseCaseRefusesToDoWithout` proves by trying each op.
Where a tool's input reaches into `internal/domain/template` it takes a tool-owned argument struct
instead of the window's DTO, one specification shape serving create, update and an override's patch.
No tool offers a sort: each list answers in its natural order (path, name, newest first).

In `confirm` mode a `write` or `dangerous` tool does not execute: it writes a
`PendingAction` row and returns `{status:"confirmationRequired", actionId, summary}`, so a
confirmation survives a restart. The arguments are decoded into the tool's own request type
first, with `DisallowUnknownFields`, and a tool that declares a `Check` runs its domain
validator too, so a call the domain would refuse is answered with the refusal rather than
put in front of the user. A result that arrives while the conversation is already answering
is queued and delivered as one turn when that turn ends, rather than dropped. Wails services call the use cases directly and typed;
they never go through the registry. A tool that works inside one site declares `Authorize` and takes its site from the
binding, so `siteId` leaves the schema the model sees and no conversation reaches another
site through it. Every field that names a domain choice carries its values as an `enum` tag
and every field of a write says what it is for; three tests in
`internal/transport/wails/vocabgen` hold that against the Go const blocks `vocab.ts` is
rendered from. `NewTool` derives that schema from the request type with the reflection
rules of `application/llm.Structured`, which do not cover a Go map; the few requests that
carry one take a tool-local argument type instead.
