# Decisions

Every decision taken while the v2 rewrite was built, phase by phase, and the
reviews that closed each milestone. `docs/STATUS.md` is the handoff; this file is
the record behind it. Sections are moved verbatim from STATUS as each phase ends.

## Decisions taken in Phase 0

- **UUIDs come from the standard library.** Go 1.27 ships a `uuid` package (RFC 9562).
  `github.com/google/uuid` is not a dependency. `kernel/id.Valid` does its own canonical
  v4 check because `uuid.Parse` also accepts braced, URN and unhyphenated text and any
  version.
- **`Cut` is a `Keyset` method**, not the free function the spec sketched. The value and
  id accessors already live on the keyset, and passing them again per call is the one
  way to make a cursor disagree with the query that produced it.
- **`kernel/ctx` exposes `ActorFrom`**, not `Actor`: `Actor` is the type name the domain
  uses for `Run.CreatedBy`, so the getter could not share it.
- **`kernel/dto` does not import `kernel/paging`.** It keeps its own limit constants and
  a test asserts the two agree. Importing paging would drag squirrel into the closure of
  everything that touches a DTO, including domain.
- **The Wails template is `vanilla`**, not `vanilla-ts`; beta.23 renamed it. It is still
  Vanilla + TypeScript + Vite.
- **No comments in `.golangci.yml`.** The repository rule forbidding comments in YAML
  won over the plan's request for an inline rationale per tuned rule. The rationales are
  in `CLAUDE.md`.
- **`errors.Wrap` returns `error`, not `*Error`.** Returning the concrete pointer meant
  `Wrap(nil, ...)` handed back a typed nil that is not nil once it is returned as an
  `error`, and `CodeOf` then dereferenced it. `Error`, `Unwrap`, `Is`, `CodeOf` and
  `Stack` are nil-receiver safe as well. Enrichment chains now start from `New(...)`,
  which cannot be nil.
- **The kernel exports nothing that only its own tests call.** `errors.Codes`,
  `errors.RetryAfter`, `paging.Order.String` and `log.RedactedKeys` are gone, and
  `SortKey.Literal` is unexported as `literal` per the spec, with its table test moved
  in-package.
- **`@wailsio/runtime` is pinned to `3.0.0-beta.23`**, matching the CLI tag, and the
  build task runs `npm ci` against the committed lockfile rather than `npm install`.
- **covergate cross-checks the filesystem.** If `internal/domain` or
  `internal/application` holds a non-test `.go` file but the profile reports no
  statements for it, the gate fails instead of reporting itself skipped. That is the
  case a narrowed `go test` package list would otherwise hide.
- **Vite writes unhashed asset names with `emptyOutDir` off**, so the committed
  `frontend/dist/.gitkeep` survives a build. Without it `go build ./...` fails on a
  fresh clone, because `frontend/assets.go` embeds a directory the frontend build has
  not created yet.

## Decisions taken in Phase 1A

- **Two pools over one file.** A writer `*sql.DB` capped at one connection with
  `_txlock=immediate`, and a reader `*sql.DB` opened `mode=ro` with four connections. One
  pool capped at a single connection would serialise reads behind long writes; an
  uncapped pool would let two goroutines both begin write transactions and turn WAL's
  single-writer rule into `SQLITE_BUSY` at commit time rather than at begin time.
- **`execFrom` reads, `writeFrom` writes.** Both return the ambient `*sql.Tx` when
  `Store.Do` is active, found through an unexported context key. Outside a transaction
  `execFrom` returns the reader and `writeFrom` returns the writer. The spec sketched one
  accessor; with two pools one accessor would make the reader dead code.
- **Nested `Do` reuses the outer transaction** and opens no savepoint. SQLite has one
  writer, and a nested savepoint would let an inner rollback be swallowed while the outer
  commits — the partial write the unit of work exists to prevent.
- **Migrations run through `goose.NewProvider`, not `goose.UpContext`.** `UpContext` reads
  the package-level filesystem and dialect that `SetBaseFS`/`SetDialect` mutate, which
  races when parallel tests open stores under `-race`.
- **The adiantum key travels as the `hexkey` URI parameter.** The VFS reads it when it
  opens the file, strictly before `journal_mode(WAL)` runs; the PRAGMA form would need SQL
  quoting inside a `_pragma=` value, and a hex string starting with a digit is not a safe
  bare pragma token.
- **The master key lives at `%APPDATA%\Postulator\master.key` via `os.UserConfigDir()`.**
  `xdg.ConfigHome` resolves to `%LOCALAPPDATA%` on Windows and would put it elsewhere;
  `xdg` also stays an indirect dependency this way.
- **No `//go:build windows` tags.** The application is Windows-only, `x/sys/windows`
  compiles nowhere else, and a tag would demand a second file that could only be a stub.
- **DPAPI is called with constant application entropy.** A fixed 32-byte literal in
  `adapters/secrets/dpapi` is passed to `CryptProtectData` and `CryptUnprotectData`, so
  any other process running as the same user cannot unprotect our master key by handing
  the blob straight back to DPAPI. The entropy is a compile-time literal, never derived
  at runtime, because changing it orphans every key already on disk. It is part of the
  on-disk format and is frozen from here on.
- **`sqlite.Open(Config{Path, Key, Recovery})` and `masterkey.Load(Config{Dir, Recovery})`
  replace the spec's `Open(path, key)`/`Load(dir)`** so recovery messages carry resolved
  paths.
- **The master key is written atomically.** `writeAtomically` writes `master.key.tmp`,
  `Sync`s, closes and then renames over `master.key`, and removes the temporary file if
  any step fails. A crash can therefore leave a stale `.tmp`, which the next write
  truncates and which `Load` never reads.
- **A key that cannot be unprotected reports `Locked`**, with the message
  `master key cannot be unprotected; remove %APPDATA%\Postulator\master.key to reset`.
  The file is never deleted automatically: the user decides whether to lose every stored
  secret.
- **A panic inside `Store.Do` rolls the transaction back and re-panics.** Leaving the
  `*sql.Tx` open stranded the only writer connection, so every later write blocked until
  its context expired. `Do` also reports `ctx.Err()` when a commit fails on a cancelled
  context, because `sql.ErrTxDone` was surfacing as `Internal` whenever database/sql's own
  rollback beat the commit.
- **Adiantum, WAL and the busy timeout work together.** The Phase 0 open question is
  answered: `TestOpenAppliesPragmas` reports `journal_mode=wal`, `foreign_keys=1`,
  `busy_timeout=5000` and `synchronous=1` on both the plain and the encrypted store, and
  the reader pool is proven to refuse writes.

## Decisions taken in Phase 1B

- **`ServiceOptions.MarshalError`, not `Options.MarshalError`.** `Bindings.Add` overwrites
  each bound method's marshaller with the service-level hook, so the application-level one
  never runs in beta.23. Services are registered with `application.NewServiceWithOptions`.
- **The transport error is `{code, message, details?, retry:{afterMs}?}`** delivered as
  the `cause` of the JavaScript rejection, not a JSON string in the message.
  `wails.Convert` strips the internal chain first, because `CallError.Message` is
  `err.Error()` and would otherwise leak the wrapped driver text.
- **`details` is dropped for `INTERNAL`.** `middleware.Recover` puts the panic text in
  `Details["panic"]`, and that must not reach the UI.
- **Generics survive the TypeScript generator.** `paging.List[T]` generates `List<T>`, so
  no concrete `XxxList` DTOs are needed. A custom `MarshalJSON` generates as `any`, which
  is why `paging.Slice[T]` and `dto.Time` lose their shape and `frontend/src/lib/paging.ts`
  restores it.
- **`application.RegisterEvent` is not used.** It would duplicate the registry in a second
  hand-written list and emit its typings into the gitignored `frontend/bindings`. Our own
  generator owns `frontend/src/generated/events.ts`, and a Go test keeps it in sync.
- **The generated TypeScript carries no header comment**, because the no-comments rule
  covers TypeScript including generated files. Its provenance is the `src/generated/` path
  and `docs/CONTRACTS.md`.
- **Envelope timestamps are `kernel/dto.Time`**, not `time.Time`: seconds precision,
  always UTC, matching every other DTO.
- **The health service moved to `internal/transport/wails`** and now has the contract shape
  `Ping(ctx, PingRequest) (BuildInfo, error)`, so the wrapper, the error hook and the
  frontend helper are exercised in production rather than only in tests.
- **`Services` is a method on `*app.Core`.** The composition root owns the Core and hands
  services what they need; `internal/transport/wails` cannot import `internal/app` because
  `internal/app` imports it, and the dependency rule allows only that direction.
- **Live events are best-effort.** v3 buffers nothing for a missing window, so
  `ListEvents(runId, sinceSeq, limit)` catch-up is mandatory.
- **`.gitattributes` pins `frontend/src/generated/events.ts` and `go.mod` to LF.** Both are
  rewritten by tools that emit LF; with `core.autocrlf=true` a fresh checkout would hand
  back CRLF and the generated-file-in-sync test failed exactly that way.

## Decisions taken in Phase 2

- **`Parents(id, depth)` is breadth-first, deduplicated, nearest level first**, name-then-id
  within a level; `Related` returns `[]Neighbor{Entity, Weight}` by weight descending.
- **`Score` is unweighted PageRank over approved edges only**, related edges counted both
  ways, damping 0.85, 30 iterations, scaled so the maximum is 1.0.
- **`Resolve(base TemplateSpec, siteOverride, pageOverride json.RawMessage)`** is RFC 7396
  over JSON; overrides are stored as merge-patch documents, arrays replace wholesale, and
  `TemplateSpec` plus the `llm` catalog types carry JSON tags. A deliberate deviation from
  spec §5.4, approved 2026-09-18; the spec copy is amended.
- **Anchors live in `entity_anchors`** and are loaded in one `IN` query per page, never N+1;
  `EntityRepo.Update` rewrites them.
- **Template overrides are keyed by `(template_id, scope, coalesce(site_id, page_id))`**
  through an expression unique index; the repository upserts by lookup.
- **One event per transaction, after commit**, through a `changed(siteID)` helper per
  service. `DeleteEntity`, `Unmap`, `SetCanonical` and page `Delete` publish both graph and
  pages because they touch both aggregates.
- **Cannibalization runs on `pages.Create`, on a path change in `Update`, and on
  `MapToEntity`**; never on `SetCanonical`. It lives in `domain/pagemap` and keeps the graph
  parameter because the keyword check needs the other entities.
- **The path is the hierarchy's source of truth**; `parent_page_id` is a maintained cache.
  `NormalizePath` is the single canonical normaliser (scheme and host stripped, no
  percent-decoding, duplicate slashes collapsed, one leading and one trailing slash,
  lowercased, no file-extension exception; whitespace, control characters and dot segments
  refused), `InternalPath(href, siteHost)` classifies link targets, `Create` adopts direct
  children, and a path change with descendants is refused.
- **Filter and sort types live in the domain packages** (`site.Query`, `graph.EntityQuery`,
  `graph.EdgeQuery`, `pagemap.Query`, `template.Query`, `template.PolicyQuery`).
- **Use-case lists embed `dto.ListRequest` and page forward only**; backward paging is a
  repository capability (`paging.Request.Before`). The kernel is untouched.
- **Request and response structs are camelCase JSON views owned by the use cases**;
  `docs/CONTRACTS.md` records that transport maps only where the wire shape must differ.
- **Repositories carry no clock**; use cases stamp second-truncated times.
- **`Site.Username` exists**; the WordPress application password lives only in the secret
  store under `site:<id>:wp_password`, written on `Create` when given and rotated or removed
  by `Update`.
- **`app.EventRelay` holds the single `EventBridge`**, connected from `cmd/postulator`
  after `application.New`; publishes before `Connect` are dropped.
- **Seeds pin no models and carry no step params.** `EnsureSeeded` also seeds one global
  `Default` link policy; `GetEffectivePolicy` falls back to it.
- **Cyclic foreign keys are real** (`entities.canonical_page_id → pages`,
  `sites.default_template_id → templates`, `sites.default_link_policy_id → link_policies`);
  the migration order is sites → link_policies → templates → entities → edges → pages →
  template_overrides, and `TestSchemaCascades` proves the cascade map with rows present.
- **Parent edges weigh 1**, `related` edges store the lower id first, `AddEdge` checks
  acyclicity as if approved and `ApproveEdge` checks again.
- **Repository writes map `UNIQUE` to `CONFLICT` with a table-specific message and zero
  affected rows to `NOT_FOUND`**; foreign-key failures stay `INVALID`.
- **`SetCanonical` maps an unmapped page** and refuses one mapped elsewhere.
- **Helpers live in the file of their first caller** (`escapeLike` in `entity_repo.go`, the
  nullable-column helpers in `page_repo.go`, the view helpers next to their use cases),
  because the `unused` linter refuses a helper committed ahead of its caller.

## Decisions taken in Phase 3A

- **WordPress page numbers, not our cursors.** Core REST is offset-based and exposes no
  keyset, so `ListItems` takes `ListQuery{Page, PerPage}` and returns
  `ItemPage{Total, TotalPages, Page, HasMore}`. Synthesising a cursor would claim a
  stability `LIMIT/OFFSET` does not have and would hide the `400
  rest_post_invalid_page_number` that is the real end-of-list signal. The plugin
  namespace is the one surface with a keyset, and its cursor is opaque end to end: the
  Go client never decodes, validates or builds one.
- **`contentHash` is `hex(sha256(raw post_content))` with no normalisation**, so Go's
  `wp.ContentHash` and PHP's `hash('sha256', $post->post_content)` agree byte for byte.
  It is deliberately not `domain/content.Document.Hash()`, which hashes normalised HTML
  for drift detection; this one is a compare-and-swap token and must be exact. `wptest`
  implements it a second time rather than importing it, and both are pinned against the
  same externally computed digest.
- **`wptest` shares no type, no JSON shape and no hash with `wp`.** A fake that imports
  the client's decoding cannot disagree with it, which is the only thing that makes it
  worth having. Its own tests drive it with plain `net/http`.
- **Reads send `context=edit`.** Without it WordPress returns the theme's rendering
  instead of the stored post, and relink would write rendered HTML back into
  `post_content`.
- **`CreateItem` re-reads the created id.** WordPress rewrites a colliding slug to
  `-2` and computes `link` from the permalink structure, so the create response is the
  only truth about both, and one extra GET buys a slug and a link that are true.
- **Tri-state fields are a `map[string]any`.** `omitempty` erases a legitimate
  `parent: 0`, so `UpdateItem` builds the body key by key: `nil` keeps, `&0` moves to the
  top level, `&id` reparents; `nil` categories keep and `[]int64{}` clears.
- **The generic write methods are core-only.** `CreateItem`, `UpdateItem` and
  `DeleteItem` report `Invalid` for `product` and `product_cat`; `UpdateProduct` is the
  one write path into a shop, and it names WooCommerce's own fields.
- **The no-redirect client is not an option.** `Probe` gets its own `*http.Client` over
  the shared transport; letting a caller switch redirect-following off for ordinary calls
  would break every site that 301s a REST path to its trailing-slash form. `Probe` never
  follows a redirect: an http-to-https upgrade is `ProbeUpgradeRequired` plus a warning,
  and a redirect to `wp-login.php` or `/wp-admin` is `Unauthorized`.
- **Retries wrap the rate limiter, not the other way round.** A retry is a new request
  against the same site and pays the same budget; the backoff sleep happens before the
  limiter grants, so a token is never held idle. `Retry-After` wins over the backoff and
  is parsed as both delta-seconds and HTTP-date. The backoff is deterministic and has no
  jitter: one desktop process against one site is not a herd.
- **A cancelled caller context is `Cancelled`, our own timeout is `External`.** Retrying
  a context the caller cancelled can only fail the same way.
- **`AllowInsecure` unlocks the `http` scheme and nothing else.** `TLSClientConfig` is
  never set in this package, and there is no option that could set it.
- **An empty `wp.proxyUrl` means no proxy**, not `http.ProxyFromEnvironment`: a stale
  `HTTPS_PROXY` must not silently route WordPress credentials through a host the user
  never configured here.
- **The manifest is cached per client, and only a 404 is cached as absent.** A 5xx or a
  transport failure is re-probed, so a site that was briefly down is not written off for
  the life of the client. Plugin methods then fail with `Invalid` and
  `Details["code"] == "plugin_missing"` without a request, which is how a caller degrades
  to core REST.
- **A log line carries method, path, status, durationMs and the error codes, never a
  body, a query string or a header.** `kernel/log` redacts by field key, which protects
  nothing if a body is logged as one blob.
- **Path normalisation is one algorithm and the domain owns it.** Decide internal-ness by
  exact lowercase host equality, strip scheme, host, query and fragment, never
  percent-decode, collapse duplicate slashes, force exactly one leading and one trailing
  slash, lowercase the whole path, and make no exception for a file extension.
  `wp.NormalizePath` and `wp.InternalPath` are now one-line calls into
  `internal/domain/pagemap`. *(Superseded 2026-10-04: both wrappers had no caller and are gone,
  and `pagemap.InternalPath` with them; `pagemap.NormalizePath` and `pagemap.Site.Resolve` are
  the one answer.)*

## Decisions taken in Phase 3B

- **The plugin never removes a kses filter.** `kses_init()` re-runs on `set_current_user`,
  and an administrator holds `unfiltered_html`, so core removes the filters itself before
  a REST callback runs. The raw write returns the hash of what is **actually stored**, not
  of what was sent, so any alteration is visible to the caller instead of silent.
- **Every write passes `wp_slash`.** `wp_insert_post` and `update_metadata` both unslash,
  so without it a single backslash or escaped quote is eaten on every save. A dedicated
  e2e case writes a Windows path and an escaped quote and asserts byte equality.
- **`normalize_path` is the only place a path is produced.** Collapse duplicate slashes,
  force one leading and one trailing slash with no file-extension exception, then
  `strtolower`. Host comparison is exact and lowercase with no `www` stripping, query and
  fragment are dropped, and nothing is percent-decoded.
- **The listing has two phases.** Posts keyset on `(post_modified_gmt, ID)`, then
  `product_cat` terms on `term_id`; the cursor names its phase. `nextCursor` is `null` at
  the end of the list and an undecodable cursor is a `400`, never a silent restart.
- **A term's `modified` is plugin-maintained.** WordPress has no such field, so
  `_postulator_modified` (RFC3339 UTC) is written on `created_term`/`edited_term` and
  backfilled once on first read.
- **A draft reports the permalink it will have**, computed from a clone with
  `post_status = publish`, rather than the `?page_id=` URL `get_permalink` returns for a
  draft. A tool whose whole model is paths cannot report `/` for every unpublished page.
- **`since` is strict RFC3339**, validated by regex before `strtotime`, because
  `strtotime` cheerfully accepts `yesterday`.
- **Write routes are post-only.** Post ids and term ids collide, and the contract has no
  discriminator, so `/seo-meta/{id}` and `/content/{id}/raw` resolve through `get_post()`
  and a term id is a `404`. Term SEO is still read in `/content`, from Yoast's
  `wpseo_taxonomy_meta` option or Rank Math's term meta.
- **The 409 is a `WP_REST_Response`, not a `WP_Error`.** `WP_Error` can only carry extra
  fields under `data`, and `currentHash` must be top-level next to `code` and `message`.
- **With no SEO plugin the head is replaced, not appended.** `pre_get_document_title` and
  `get_canonical_url` make a duplicate `<title>` or canonical impossible; the e2e suite
  asserts a count of exactly one of each.
- **`cmd/pluginzip` is Go, not `zip` or `ZipArchive`.** Neither exists reliably on Windows
  and neither is deterministic without argument archaeology. Sorted names, the ZIP epoch
  and mode 0644 make two runs byte-identical.
- **WordPress is pinned to 7.0.1, not the 6.9.2 the plan named.** WooCommerce 11.1
  requires WordPress 7.0, so 6.9.2 could not install it and `product`/`product_cat` would
  have gone untested.
- **Application passwords need `WP_ENVIRONMENT_TYPE=local` over plain HTTP**, and the
  WP-CLI container needs the `WORDPRESS_DB_*` variables because the official image's
  `wp-config.php` reads them at runtime rather than baking them in.
- **Rank Math is supported.** It keeps its entire front end silent until the setup wizard
  is past: the meta keys wrote and read back correctly while the rendered page carried
  core's title and no description. `rank_math_is_configured` and
  `rank_math_registration_skip` are the two gates, and the bootstrap sets both.

## Decisions taken in Phase 4

- **The port owns the schema, the adapter owns the provider.** `Structured[T]` derives the
  schema by reflection from `json`, `description` and `enum:"a,b"` tags, refuses recursive
  types, maps and channels before the first call, appends one JSON instruction to the
  system prompt and repairs exactly one decode failure by feeding the decoder error back.
  `gollemclient` turns it into a `gollem.Parameter` with `ContentTypeJSON` and
  `WithSessionResponseSchema`, so the constraint is provider-native.
- **Usage is `gollem.Response.InputToken`/`OutputToken`**, on the completion and on the
  last streamed chunk. gollem exposes no finish reason, so `stop` and `length` come from
  the token ceiling and `content_filter` from `gollem.ErrProhibitedContent`. `Delta`
  carries an `Err` like `gollem.Response.Error`; without it a failure after the first
  chunk would close the channel silently.
- **Three gollem v0.28.4 limits are worked around or accepted.** Its OpenAI session drops
  the system prompt, so the adapter sends it as a `RoleSystem` history message for that
  provider only. Its Claude `Session.Stream` calls the non-streaming Messages API and
  emits one chunk. Its Gemini client is Vertex-only — no API key, no endpoint override,
  and `WithGoogleCloudOptions` is ignored — so the settings are `llm.gemini.projectId` and
  `llm.gemini.location` rather than the `llm.gemini.baseUrl` the brief named, and Gemini
  has no httptest coverage. `go-openai` also rejects a temperature other than 1 for any
  `gpt-5*` or o-series model before the request leaves the process.
- **The embedded catalog carries only models whose price was read off a vendor page** on
  2026-09-18: OpenAI `gpt-6-astra` and `gpt-5.6-{sol,terra,luna}`; Anthropic
  `claude-fable-5-1`, `claude-opus-5`, `claude-sonnet-5`, `claude-haiku-4-5`; Google
  `gemini-3.5-flash{,-lite}` and `gemini-2.5-flash{,-lite}`. Tiered and promotional prices
  are left out, because `ModelInfo` holds one flat rate per direction; rate limits are not
  published per model, so every entry carries one conservative `rpm`/`tpm` and
  `model_catalog` is the correction. An override row is the whole entry, so disabling is
  `enabled = 0`, re-enabling is another upsert, and neither repository has a delete.
- **The ledger stamps `created_at` from `kernel/clock` and measures latency with
  `time.Since`**; the clock abstraction is for timestamps, not elapsed time. The
  record/replay key excludes `Request.Meta`, so a fixture survives a new run id.

## Decisions taken in Phases 5 and 6

- **One `Status` for runs and for items**, the seven values of section 7. Section 5.6
  sketched two vocabularies; a single one keeps `Advanceable`, `Active` and `Terminal`
  meaningful on both rows and is what the claim query filters on.
- **`Classify` maps `NeedsHuman` to `exhausted`**, not to a class of its own. Exhausted is
  the class whose default action is `pause`, and a run that needs a human is in exactly
  that position: it cannot proceed without something outside the process.
- **A recovered panic is a transient fault.** A step that panics is retried under its own
  retry ceiling rather than killing the worker or failing the item outright; the engine
  logs the fault without the panic text reaching the event payload.
- **The claim, the step and the settle are three phases, two transactions.** The step runs
  between them with no transaction open, because an LLM call must not hold the single
  SQLite writer. The settle lands only against the advance sequence the claim took, so a
  pause, a cancel or a reclaim while the step ran discards the transition — while still
  recording the execution and its artifacts, which is what makes the input-hash reuse work.
- **`advance_seq` is the only optimistic lock.** `Claim`, `Requeue` and `StopAll` each bump
  it; `Persist` matches it. A lease is a hint for the sweep, never the lock itself.
- **Artifacts are immutable and keyed by `(item, step, kind)`**, so a retry replaces its own
  step's rows and a later step reads the newest earlier producer of a kind. A step
  execution's `attempt` counts the rows already recorded, so a manual `RetryStep` cannot
  collide with the unique index, and a step with no `Retry.Max` gets three attempts.
- **`WakeAt` resolution is the sweep interval.** There is no per-item timer: a waiting item
  is re-armed by the next sweep, which is also the path a crash recovers through.
- **The template spec governs link shaping, the site policy governs the site-wide rules.**
  `effectivePolicy` takes `LinkRules` from the resolved spec and `ForbidExternal`,
  `ForbidSelf` and `AnchorStrategy` from the site's effective policy.
- **`InsertLinks` counts an existing link to a target as placed** and re-runs to the same
  document, which is what makes `repair_links` and a re-run of the whole item safe; a
  property test runs two hundred random bodies through it twice. `Structure` never sees the
  meta title, because it reads a body fragment, so `validate` adds the
  `primary_missing_in_title` finding from the draft artifact instead.
- **A report scores `1 - 0.25 per error - 0.05 per warning`, floored at zero**, and the
  validation artifact carries the lower of the two scores. Prompts are
  `{{define "<step>.system"}}` and `{{define "<step>.user"}}` in one embedded template per
  step.

## Decisions taken in Phase 7

- **A sync run has no target page.** `run.Kind.PageScoped()` is false for `sync` and `import`, and the
  engine then claims an item without resolving a page or a template spec; the item's `page_id` names the
  site. A synthetic page row per site would have leaked into every page listing instead.
- **A step that returns `wait` with a `WakeAt` that is not in the future is re-dispatched at once**, not at
  the next sweep. That is what makes `sync_site` a batch loop whose cursor is persisted after every batch,
  so a crash resumes from the last one rather than from the beginning.
- **`report` does not write `Run.Stats`**, against the letter of the phase brief. The engine recomputes the
  stats on every settle from the item counts and the ledger, so a second writer could only disagree with
  it; the aggregate lives in the `final_report` artifact.
- **Image failures are recorded, not raised.** `generate_images` puts `no_image_source`, `images_failed` or
  `image_upload_failed` into the `images` artifact and carries on, because a missing stock folder must not
  fail a page. `judge` swallows its own model failure the same way, except on a cancelled context.
- **`publish` looks up by `Page.WPID` first and by slug plus parent second**, passing the five editable
  statuses, because core REST lists only published pages by default. A create whose response was lost is
  therefore reconciled by the next attempt rather than duplicated.
- **`relink_neighbors` writes a neighbour only when a link was actually inserted**, never on
  `already_linked`, and a `409` becomes a `relink_conflict` warning finding of class `needs_human` on an
  item that still completes. Without the plugin every neighbour is `skipped`.
- **The sync reconciles in one pass and repairs in a second.** Parent ids and link targets that arrive in a
  later batch are resolved once the pull is done, because a batch only knows the pages it has already seen.
- **The core-REST pull covers `page` and `post` only**, and therefore never archives a `product` or a
  `product_cat` row; the plugin's `/content` listing covers all four. Neither path passes `since`: a full
  pull is what makes "absent from the site" mean archived, which is the answer to the `modified_after` gap.
- **`wp-plugin` is a Go package.** `//go:embed` cannot reach a parent directory, so the companion tree is
  embedded at its own root and `internal/adapters/wp/plugin` packs it deterministically from an `fs.FS`.
- **The client registry caches by `site.UpdatedAt`.** One rate limiter and one manifest cache per site are
  worth keeping; a changed base URL or a rotated password bumps `UpdatedAt` and the next call rebuilds.

## Decisions taken in Phase 8

- **The mapping, the fields and the auto-detection live in `internal/domain/importmap`**, not in
  `adapters/importer` as section 9.6 sketched: `application/imports` must name those types and the
  dependency rule forbids it from importing an adapter. The adapter is the file reader and writer only.
- **The migration is `0015`, not `0016`** — the tree ended at `0014_runs.sql`.
- **A header folds every non-alphanumeric run to one space**, unlike the Archond mapper that drops them,
  and a miss retries with the spaces removed. Otherwise the export's own `primary_keyword` header would not
  survive a round trip through its own alias table.
- **Cannibalization is a warning, never an error**, as are an unrecognised entity kind, WordPress type or
  page kind, which fall back. Only an unknown parent or related entity, a self-edge, a cycle and an
  unreadable path are errors, and `Apply` refuses while any of them stands.
- **`page_kind` resolves to a template**, preferring a site-scoped one over the global one of that kind: it
  is the only field with nowhere else to land, and read-but-unused would be a hole.
- **An existing entity keeps its spelling and its primary keyword.** The import unions the keywords and the
  anchors and overrides the kind only when the row names one; it never renames what the operator curated.
- **`SaveMappingAs` reuses the mapping already saved under that name**, because the unique index is on
  `(site_id, name)` and a second import would otherwise roll back on a constraint nobody sees.
- **`examples/sitemap-import-example.json` and `examples/sitemap.json` are gone.** JSON import is out of
  scope; the csv and xlsx samples stay and a test proves `AutoDetect` still opens both.

## Decisions taken in Phases 9 and 10

- **A run item targets a site or a page.** `run_items.target_id` carries no foreign key and `site_id`
  cascades from `sites`, so a sync item can name its site and a page delete leaves the run history
  standing. The engine reads the site off the item instead of joining the run.
- **The checkpoint is an input to a step**, so it belongs in the input hash. Without it a step that asks to
  wait reuses its own first execution on the next dispatch, and a multi-batch `sync_site` completed after
  one page of the pull.
- **A refused publish writes no `publish_result` artifact.** That artifact means the item reached
  WordPress, and the retention sweep purges drafts on the strength of it.
- **One judge rubric.** `application/content` owns the prompt, the report and the structured call; the run
  step hands it the draft it already holds and the on-demand audit hands it the live page. Two copies would
  drift and then disagree about the same page.
- **A site scoped tool takes its site from the binding**, `siteId` is removed from the schema the model
  sees, and `Authorize` denies the tool in a conversation that names no site.
- **The seven requests that carry a Go map take a tool-local argument type.** `NewTool` derives its schema
  with the rules of `llm.Structured`, which refuse a map; a spreadsheet mapping reads better to a model as
  a list of pairs and a template specification as one JSON object anyway.
- **`RunSpec` and `RunResult` are declared by `application/agent`**, against the letter of section 9.4: the
  consumer declares the interface it calls, and the runner is the implementation. Only the system prompt
  template stayed at the transport edge.
- **The gollem history is stored as its own blob** beside the message transcript. The transcript is ours
  and the UI reads it; the blob is the provider's replayable context and gollem owns its shape and version.
- **The audit records what the tool answered and the fence wraps the copy the model reads**, which is why
  the fence is the outermost middleware and the audit sits under it. A tool call's arguments are masked by
  the logger's rule before they reach the ledger, the event or the summary; only the pending action keeps
  them whole, because that row is the command it will replay.
- **A tool call that cannot be audited fails the turn**, not the call: gollem hands a middleware error
  back to the model rather than aborting, so the guard keeps the first failure and the runner raises it.
- **`github.com/robfig/cron/v3` is the second third-party package the domain may import**, on the same
  ground as `golang.org/x/net/html`: a hand-written cron parser would be more code and less correct.
  `internal/app/deps_test.go` carries the allowance.
- **A skipped schedule is rearmed.** A schedule whose previous run is still going, or whose target query
  matches nothing, still moves its `next_run_at` forward, because a skip that leaves the time in the past
  is a hot loop.

## Decisions taken in Phase 11

- **A Wails service declares the interface it consumes.** The rule that the consumer owns the interface is
  what makes the boundary testable: a fake use case returns one code, a reflection table calls every
  exported method, and a method that skipped the wrapper fails that table rather than reaching production.
- **`Services` takes its dependencies.** `internal/transport/wails` cannot import the composition root, so
  `wails.Deps` carries the use cases and `*app.Core` fills it. The fields are exported and their types are
  not, which is exactly the visibility the root needs.
- **`SettingsService` is the one surface that owns a credential.** `SetProviderKey` lives there and not on
  `ModelsService`, so there is a single path for secret input; `models.SetProviderKey` is still the use
  case behind it and the response names the provider alone.
- **`kernel/settings.Registry.Validate` is new.** `Apply` checks a whole map and swaps the live values in
  one go, which is the wrong shape for refusing one bad write; `Set` validates, persists, then re-applies
  so the running process sees the change without a restart.
- **A use case that answers with bytes gets a path at the transport.** The webview has no filesystem, so
  `SyncService.SavePluginPackage{path}` writes the archive and returns where it landed.
- **The event list in `docs/CONTRACTS.md` is a pointer, not a copy.** The registry renders
  `frontend/src/generated/events.ts`; a second hand-maintained list in prose is the one that goes stale.
- **`ToolsService` exposes `List` only.** The UI shows what the agent can do; it never calls a tool through
  the registry, because a Wails service calls the use case directly and typed.

## Milestone review 5–8

Reviewed 2026-09-18. The gate is green: `go build`, `go vet`, `golangci-lint` (0 issues), `gofmt -l .`,
`go test -count=1 -race ./...`, `go test -race -count=3 ./internal/runtime/...` and `covergate`
(92.96% domain+application, 90.16% of 11214). Probes confirmed the `advance_seq` CAS admits one claimer
of eight racers, `run_events.seq` stays gapless under three concurrent items, `Stop()` returns only after
every in-flight step has and leaks no goroutine, and `Cancel` reaches a blocked step. Three fixes landed:
`280537c` (link insertion spliced anchors into `<script>`, `<style>` and `<textarea>`), `d851450` (a run
could target a page of another site and publish it through the first site's client) and `fb944b3`.
The five gaps it left open were closed at the head of Phase 9: `14226fd` rebuilds `run_items` around a
`target_id` with no foreign key and a cascading `site_id`, so a sync run can be enqueued at all;
`bd774fe` keys the step reuse on the checkpoint, which is what a batch loop reads; `2826efb` refuses a
publish over a page a human edited; `739fbb2` tags the sync cursor with its source and stops claiming a
content hash the sync did not write; and `8821e67` applies `import.maxRows` while the rows are read. The
orphaned media is listed under Known gaps instead, as an accepted cost.

## Decisions taken in Phase 12

- **The master password wraps the DPAPI blob, not the raw key.** `master.key.pw` carries
  `v1:` + salt + AES-GCM(Argon2id(password, salt, t=3, m=64MiB, p=4), dpapi.Protect(key)), so
  reading the key needs the Windows account *and* the password. `adapters/secrets/masterpassword`
  is the crypto alone; the file transitions stay in `masterkey`, which already owns the atomic
  write and the recovery message. `masterkey.Load` refuses with `Locked` while `master.key.pw`
  exists rather than minting a fresh key over a database it could no longer read.
- **A locked core is a core without a composition.** `app.Open` returns a `Core` carrying the
  event relay and nothing else; `Unlock` composes the store and every service into an embedded
  `kit` struct and `Lock` replaces that struct with its zero value. `Locked()` is
  `Store == nil`, so there is no second flag to disagree with.
- **A Wails service resolves its use case per call through a `Source[T]`.** The bound services
  are built once at startup, before a password has been entered, so `Deps` carries
  `func() (T, error)` instead of the use case itself. The source refuses with `Locked` while the
  composition is absent, which is what makes every bound method answer `LOCKED` from one place.
  The use-case interfaces are exported for that reason: the composition root has to name them to
  hand over a resolver.
- **`VACUUM INTO` cannot write through the adiantum VFS.** It opens its target with the same VFS
  and no key, which refuses the write, so `Store.Snapshot` and `Store.Restore` use the SQLite
  online backup API with a plain `file:` URI for the other side. The snapshot is therefore a
  readable database and the restore re-encrypts it on the way in.
- **A backup frame authenticates its counter and whether it is the last one.** Truncating the
  file, reordering frames or appending bytes is `Invalid` rather than a short restore; the
  master key is never in the archive, so a backup is worthless without its password.
- **`ImportBackup` restarts the composition, it does not swap files.** The archive is decrypted
  and checked whole, the engine, the scheduler and the agent are stopped, the snapshot is copied
  into the encrypted database and the core is composed again from it, so the settings and the
  seeds of the restored database are the ones the process then runs on.
- **gollem v0.28.4 has no Gemini client that takes an API key.** `gemini.New` hardcodes the
  Vertex backend and demands a project and a location, so the key path is the provider
  `gemini-openai`: the OpenAI-compatible client pointed at
  `https://generativelanguage.googleapis.com/v1beta/openai/`, with its own catalogue entries and
  its own base URL setting. The Vertex `gemini` provider is untouched.
- **The import links a created page to the page above it.** `pages.Create` always did and the
  import did not, which published every imported draft at the root of the site.
- **`app.Config.Provider` replaces the base model client.** The end-to-end harness composes the
  real application — retry, limiter, ledger and record/replay included — over
  `adapters/llm/fake`, so the whole loop runs without a network call and without a second wiring
  of the composition root.
- **`fake.Scripted` picks its reply by step and by a phrase in the prompt.** One reply per step
  was not enough once five pages with five primary keywords went through the same recipe.
- **The sample workbook carries a graph.** `examples/sitemap-import-example.xlsx` gains entity,
  parent entity, primary keyword, anchors and page type over eight menu rows; anchors are
  separated by `|`, which is what `importmap.DefaultAnchorSeparator` reads.
- **Docs split in two.** `docs/STATUS.md` is the handoff and stays under 150 lines; every
  "Decisions taken in Phase N" and "Milestone review" section lives here, moved verbatim.

## Decisions taken on 2026-09-19 for the frontend surface

Six additions the React frontend needs, and nothing else. The Go backend was finished and green;
each of these exists because a screen could not be drawn without it.

- **A provider key is reported as a boolean, never as a value.** `SetProviderKey` was write-only,
  so the settings screen could not mark a provider configured and had no way to revoke one.
  `ProviderKeys` answers one row per provider the catalog names, `{provider, configured}`, and
  `DeleteProviderKey` takes a provider name. Neither carries a key, so the phase 11 ruling holds:
  `SetProviderKey` is still the only method that accepts a credential. `secrets.Store.Has` was
  added rather than reading the secret back, because presence is answerable from the vault row and
  unsealing a key to discover that it exists is a plaintext no one asked for. Both new methods
  refuse a provider the catalog does not know, exactly as `SetProviderKey` does, and a revoke of a
  key that is already gone succeeds: the screen offers revoke from a listing that may be stale.
- **A connection test reuses `wp.Client.Probe` and adds no second HTTP path.** The probe already
  classifies the four outcomes a site form has to render — a REST root that is not WordPress, a
  redirect from http to https, a redirect to the login page, and refused credentials — so
  `TestConnection` maps them onto `site.Reach` rather than asking the site again.
- **A connection test builds a throwaway client.** `registry.Client` caches per site id, updated-at
  and base URL; a candidate has none of those, and the create form has to test credentials before
  there is a row. The registry therefore constructs a one-off `wp.Client` and leaves the cache
  alone. An existing site with no password in the request falls back to the stored secret and
  reports `unauthorized` when there is none, because "you never entered a password" is one of the
  four things the button is there to tell you.
- **`site.Candidate` redacts its password.** It is the one domain value that carries a credential,
  and it crosses two package boundaries to reach the adapter. `String` and `GoString` print `***`
  on the same ground as `wp.Config`: rule 8 is about what a formatted value can leak, not only
  about what a logger is asked to write.
- **`ListArtifacts` is a read beside `GetArtifact`, not a change to it.** A review drawer had to
  probe all eleven kinds and swallow `NOT_FOUND` to find its tabs. The listing carries `purged` and
  `expiresAt` because the sweep retires the blob of `body_html`, `draft` and `images` once the page
  published and leaves the row with `size` 0: without those two fields the drawer cannot tell an
  artifact that never existed from one whose body expired, and `size` alone lies. `GetArtifact`
  already answers a purged row rather than `NOT_FOUND` and was not touched; the eleven codes stay
  frozen.
- **The three new events are published from the application layer, and `settings.changed` from
  `models`.** `sites.changed` and `schedules.changed` go where `graph.changed` and `pages.changed`
  go, in the service that owns the write, so the agent and the UI announce the same change through
  one path; the schedule rearm publishes too, because a tick that moves `next_run_at` is exactly
  the change a list is stale about. `settings.changed` has no application settings use case to live
  in — `SettingsService` reads the `kernel/settings` registry directly — so it is published where a
  settings-shaped write does exist in the application layer, the provider key. A declared value
  written through `SettingsService.Set` announces nothing; that is recorded as a gap rather than
  papered over by publishing from the transport.
- **`runs.deadline` replaces the constant.** `runtime.DefaultRunDeadline` stays as the declared
  default and as what `normalized()` falls back to for a hand-built `Config`, but `Settings` now
  reads the setting into `Config.RunDeadline`, which is what `Enqueue` adds to a run's `createdAt`.
  It is bounded to five minutes and seven days: a run over a large site legitimately takes days,
  and a deadline shorter than a step timeout would fail every run.
- **`Estimate` is an exposure, not new arithmetic.** `Start` answered the estimate together with
  the `runId`, so the run existed by the time its cost was known and the budget dialog
  `docs/VISION.md` asks for could not be shown. `Start` and `Estimate` share `plan()`, which
  validates the request and resolves the template, and both hand the same `run.Run` to
  `Engine.EstimateRun`; only `Start` mints the id, after the estimate, so an `Estimate` leaves
  nothing behind. It takes `StartRequest` rather than a request of its own, which is what makes
  "the same inputs" a compile-time fact instead of a convention.
- **`retryable` is computed against the step's `Requires`, and `RetryStep` refuses up front.**
  `StepContext.Artifact` failed a purged input with `NOT_FOUND` from inside the step, so a retry
  offered by the screen died halfway through the pipeline; nothing regenerates a purged artifact.
  `run.ExpiredInputs` intersects the current step's `Requires` with the item's purged kinds, and
  both callers read it: `RunsService.ListItems` carries `retryable` with `retryBlockedReason`, and
  `Engine.RetryStep` refuses before it touches the item, naming the step and the expired kinds in
  the details of a `NOT_FOUND`. The existing `CONFLICT` for a running or pending item is unchanged.
  `inputs_expired` is the only value the reason takes and is frozen the way the eleven error codes
  are; the frontend disables the control and points at a fresh run over the page rather than
  offering a retry and explaining the failure afterwards.
- **The purgeable artifact kinds are declared data, and the sweep binds them.** `body_html`,
  `draft` and `images` were literals inside the purge SQL, which made the retention rule
  unreadable from Go and impossible to generate from. `run.PurgeableArtifactKinds` is the
  declaration; `ArtifactRepo.PurgePublishedBefore` binds it as parameters rather than interpolating
  it, and `frontend/src/generated/vocab.ts` derives `purgeableArtifactKinds` from
  `ArtifactKind.Purgeable`. A repository test seeds one artifact of every kind and asserts the
  sweep touches exactly the declared set.
- **The step names are a const block in `domain/run`, in pipeline order.** Every step named itself
  with a literal in `internal/runtime/steps`, so nothing said what order they run in or that
  `sync_site` belongs to the same vocabulary. `run.StepNames` is that order — the order
  `steps.Register` uses and the order the shipped seeds enable — and
  `TestTheShippedStepsRegisterInRecipeOrder` reads it, so reordering one without the other fails.
  The hand-written `frontend/src/domain/vocab.ts` had listed `generate_images` before
  `insert_links` and `repair_links` after `relink_neighbors`, which no recipe does; the generated
  module corrects that, and a progress indicator counting through it is honest.
- **Which import finding codes block an Apply is declared beside the codes.** The split was a
  condition in the preview: each call site chose `warn` or `fail`. A single `note` now routes by
  `FindingCode.Blocking`, whose declaration is `blockingFindingCodes`, so a code added later cannot
  default into the wrong half and the import screen enables its Apply button from the generated
  `blockingImportFindingCodes` rather than from a list retyped in TypeScript.
- **`kernel/settings` declares its types and its groups.** `Kind.String` built the type names
  inside a switch and `Schema` cut the group off the key, so neither vocabulary existed as data.
  `settings.Type` is what `Kind.String` renders and `settings.GroupOf` is what `Schema` reports;
  `TestEverySettingBelongsToADeclaredGroup` in `internal/app`, the one package that composes every
  registration, fails when a setting is declared under a group `settings.Groups` does not name.
- **`frontend/src/generated/vocab.ts` is rendered from the Go const blocks by `task vocab`.** Every
  string union crossed the boundary as a bare `string` and was retyped in TypeScript, where a
  rename in Go drifted silently into a UI switching on a value that no longer exists. The generator
  parses the sources with `go/ast` rather than reflecting over a map, because the order is
  semantic: artifact kinds are the review drawer's tab order, step names are what make "step 4 of
  12" honest, run statuses are active then terminal, and tool risks ascend by severity. The five
  derived groupings call the real Go predicates — `Status.Active`, `Status.Terminal`,
  `ArtifactKind.Purgeable`, `Risk.NeedsConfirmation`, `FindingCode.Blocking` — so a behaviour and
  its exported list cannot disagree. Sort fields are package-qualified because four packages call
  the type `Sort`; `template.Sort` covers both templates and policies, so five Go types serve six
  frontend lists and the sixth is not synthesised. Edges, run items, conversations, messages,
  pending actions and schedules declare no sort and therefore generate nothing. A byte-comparing
  test fails when the committed file is stale, and `.gitattributes` pins it to LF beside
  `events.ts` for the reason recorded there.

## Decisions taken on 2026-09-19 for the linking screen

The Linking screen shows, per page, the links the graph asks for and whether they exist. Nothing
read that: `Pages.Get` answers one page's links and `SiteOverview` reduces a site to two
counters.

- **Two reads, not one.** A denormalised required-link row is about 380 bytes; a site of five
  thousand pages with eight targets each would push about 18 MB through the WebView2 bridge in
  one answer and hold forty thousand rows nobody looks at at once. `LinkAudit{siteId}` answers
  the policy, the totals and one summary row per non-archived page, about 1.6 MB for that site,
  and `LinkAuditPage{pageId}` answers one page's required and extra links, about 4 KB. Both are
  computed by the same pure `auditPage` over the same loaded state, and a test asserts the
  detail's summary row equals the site row. Neither is a list, so the cursor rule is untouched.
- **`content.PlanLinks` wraps the traversal rather than adding a second one.** `BuildLinkContext`
  silently dropped a target whose entity has no canonical page, and the screen has to name
  exactly those. `PlanLinks` walks parents, children and siblings once and answers the context
  beside the blocked targets; `BuildLinkContext` is its context, so the `link_context` artifact
  and every step are unchanged.
- **The rules come from the page's resolved template.** `resolve_context` builds its policy from
  the page's `TemplateSpec.LinkRules` and takes only `ForbidExternal`, `ForbidSelf` and
  `AnchorStrategy` from the site's effective policy; the audit does the same, one `ResolveForPage`
  per mapped page, so the screen and the run agree. A page whose template cannot be resolved is
  reported as `no_template` rather than failing the site.
- **A stored link is classified by its resolved page id first.** `page_links` rows carry
  `ToPageID` when the sync or the relink resolved the href, and a path otherwise; `ClassifyLink`
  trusts the id, then `pagemap.InternalPath` against the site host from the site's base URL.
  `Compliance` keeps its href-based classifier with an empty host; the divergence on an absolute
  own-host link is recorded in `STATUS.md`.
- **Self is the audited page.** `LinkContext.PageID` is the entity's canonical page, so a second
  page mapped to the same entity would read the canonical page as itself. The audit rebinds
  `PageID` and `PageURL` to the page being audited.
- **A self-link does not reach a page.** `linkSets` no longer counts a link from a page to itself
  as incoming, so a page whose only inbound link is its own is an orphan in both the overview and
  the audit, which agree by construction.
- **`reports.New` takes a `Deps` struct.** Ten readers is where `graph`, `content` and `imports`
  switched; the audit added the site, the spec resolver and the policy reader to the seven.
- **Four more vocabularies are generated.** `linkRelations`, `linkClasses` with the derived
  `offGraphLinkClasses` over `LinkClass.OffGraph`, `linkBlockedReasons` and
  `linkAuditSkipReasons`, so the screen never retypes a value the audit switches on.
- **An edge keeps why it was proposed.** `ProposeRelated` asked the model for a one-sentence
  reason and dropped it, so the review queue had nothing to show beside a weight. Migration 0019
  adds `edges.reason TEXT NOT NULL DEFAULT ''`; `NewEdge` trims it and refuses more than two
  hundred characters, the proposers clip before that cap so a talkative model loses words rather
  than an edge (`propose` swallows a `NewEdge` error), and `ProposeFromPages` derives its reason
  from the page paths, which is the only evidence that step holds. `AddEdgeRequest.Reason` is
  optional, so `graph_add_edge` offers it to the agent without requiring it. The migration test
  demands `STRICT` only of a file that creates a table, because an `ALTER TABLE` carries none.

## Decisions taken on 2026-09-19 for the agent surface and templates

The agent had a finished data layer and no screen: the dock, `/agent`, `/agent/inbox` and every
confirmation were `NotBuilt` panels. Templates had an editor that could not start from nothing,
could not edit a page's own changes and showed no page.

- **The dock is primary and bound to the site in the route.** It remembers the conversation
  chosen per site in `localStorage` and falls back to the site's newest; `/agent/:id` is the same
  view, wide. One agent, many conversations: no personas and no per-conversation allow list.
- **Context reaches the agent as text the user can read.** "Ask the agent about this" prefills
  the composer with the record's name and id; the system prompt already tells the model to pass
  ids exactly as given. A hidden context field would be a second, unseen input to a turn.
- **Three Go additions and no more.** `RenameConversation`, `DeleteConversation`, which cancels a
  turn in flight before the foreign keys of migration 0017 take the messages, pending actions,
  tool calls and history with it, and a first message titling an untitled conversation, because
  every conversation the UI starts is untitled and a list of untitled rows is useless.
- **`ListMessages` is the transcript; the live turn is an overlay.** `Send` answers the user
  message's id while every stream event carries the assistant's, so the overlay is deduped by the
  saved row id and by `callId`, never by the id `Send` returned. `agent.confirm.requested`'s
  `confirmationId` is the pending action's id, so a live card and a listed one are the same card.
- **A turn that stops reporting is `stalled`, not idle.** It keeps what it streamed, says so and
  offers to ask again; the saved rows are refetched behind it.
- **A confirmation card is described per tool family, never as JSON.** One describer per family
  turns arguments into sentences with records named rather than numbered; a schema walk is the
  fallback for a tool nobody described. A test asserts every confirmable tool has its own
  describer and that no line carries JSON; a masked secret reads "kept hidden". `dangerous` is a
  red border, a gavel and a warning, not a colour change.
- **A template card speaks the editor's sentences.** `templates_create`'s spec and
  `templates_set_override`'s patch cross as JSON strings; the card parses them with the editor's
  own draft reader and `sentencesOf`, and an update is described as the merge patch against the
  current template, so the client reads what changes rather than a whole document.
- **Templates start blank or as a copy.** The blank draft mirrors the seeds' rule groups with one
  required section, no images and every step on but `generate_images`, and satisfies
  `template.Validate`. A page's own changes are edited in place through `?page=`, where "follow"
  returns a value to what the site says. Only the two step settings a step reads, `allowErrors`
  and `iterations`, are typed; any other is kept and named. The site default is set from the list
  and the editor through `SitesService.Update.defaults`.
- **The skeleton is computed, not generated.** The editor's preview draws the page a template
  asks for from the draft alone: title pattern, H1, sections sized by their share of the words,
  the parent link window, image slots and the length verdict. It costs nothing and cannot
  misrepresent a model's output.

## Decisions taken on 2026-09-19 for the page preview

The client could not see a generated page the way the site's theme renders it: a run writes
drafts, and WordPress shows a draft only to a logged-in editor.

- **A published page never touches the plugin.** `PagesService.PreviewLink{pageId}` answers the
  site's base URL and the page path with `kind: "public"` and no expiry.
- **A draft gets a signed link from the companion plugin, version 1.1.0.** `POST
  /content/{id}/preview` mints 16 random bytes, stores only their sha256 beside an expiry an hour
  away, and answers the permalink with `preview=true` and the token. Storing the hash means a
  second call cannot hand the first link back, so every call rotates it; reuse would need the
  token in a plain column.
- **The theme renders the draft for that one request.** A `posts_results` filter on the main
  singular query flips the one post's status to `publish` in memory when the token matches and
  has not expired, with no-cache headers, `DONOTCACHEPAGE`, a noindex robots directive, the
  canonical redirect off and comments closed. The query runs with `cache_results` off so a
  persistent object cache never keeps the flipped status. A wrong or expired token changes
  nothing, and WordPress answers an anonymous visitor with its 404. No filter is removed, and
  issuing a link writes post meta only, so it never registers as drift.
- **`plugin_outdated` sits beside `plugin_missing`.** Both are `INVALID` with `details.code`; a
  1.0.0 plugin lacks the `preview` capability and would otherwise answer `rest_no_route`, which
  reads like a missing post. The use case returns the adapter's refusal unchanged, the frontend
  keys on `details.code`, and an `INVALID` without a field never toasts.
- **The application reaches the plugin through a consumer interface.** `pages.previewIssuer` is
  bridged in `internal/app` over the site registry, as `content.rawReader` is, because the
  application may not import an adapter.
- **The frontend holds a link fifty minutes and keys it outside the pages root.** `bridge.tsx`
  invalidates `keys.pages.root()` on unlock and after every agent turn, and a refetch would
  rotate the link under an open frame; the key carries the page status, so a page that gets
  published resolves to its public address.
- **`pages_preview_link` is a `write` tool.** It changes nothing in the store, but it hands a
  draft to anyone holding the link for an hour, so confirm mode asks first.
- **The frame is best effort.** Nothing in the app sets a frame policy, but a security plugin or a
  host header can refuse to be framed and a parent cannot detect it, so "Open in your browser"
  sits beside the frame. Without the plugin a draft offers WordPress's own `?page_id=` address,
  which asks the client to log in.

## Decisions taken on 2026-09-20 for the application mark

The client opened the built window and found no icon anywhere: not on the executable, not on the
window, not in the taskbar, not in the installer. Nothing had been forgotten — the resource was
being built and thrown away.

- **The resource object was written where Go cannot see it.** `build/windows/Taskfile.yml` is the
  Wails v3 template's, which assumes `main.go` at the repository root, so it generated
  `../wails_windows_amd64.syso` beside `go.mod`. Go links a `.syso` only from the directory of a
  package in the build, and the root holds no Go files, so every build produced the object, ignored
  it and deleted it. `bin/postulator.exe` therefore carried no `VS_VERSIONINFO` and no
  `RT_GROUP_ICON`, and Wails on Windows reads the window icon out of the executable's own resource
  id 3 (`webview_window_windows.go`), which is why the window and the taskbar were bare too. The
  `-out` path now names `cmd/postulator`. **`wails3 update build-assets` regenerates that Taskfile
  from the template and will put the wrong path back**; the generated file is edited on purpose.
- **One geometry, four artifacts, no new dependency.** `cmd/appicon` declares the mark once as
  numbers and renders every shape the build needs: `build/windows/icon.ico` for the executable and
  the NSIS installer, `build/appicon.png`, `cmd/postulator/appicon.png` for
  `application.Options.Icon`, and `frontend/public/appmark.svg` for the page icon and the title
  bar. A second hand-authored copy of the same shape would drift the moment either was touched.
- **The raster is drawn with distance functions, not a font.** A monogram cut from Archivo would
  need a glyph rasteriser and would turn to mush at 16px. The stem is a rectangle and the bowl a
  half annulus, sampled sixteen times per pixel; the SVG states the same two shapes as one path
  with `fill-rule="evenodd"`, so the vector and the raster cannot disagree.
- **The icon entries are PNG, and the container is written by hand.** The ICO directory is a
  six-byte header and sixteen bytes per entry; `wails3 generate syso` accepts PNG payloads, which
  is verified by the build linking all seven sizes. The test decodes the committed file and
  compares its pixels with a fresh render rather than comparing bytes, so a change in Go's
  deflate output is not reported as a stale icon while a changed mark still is.

## Decisions taken on 2026-09-20 for the shell, the site scheme and chat names

Three defects found by opening the built window, each one not what it looked like.

- **A design system primitive does not size itself.** `ui/select.tsx` hardcoded `w-full` on its
  trigger and `ui/cx.ts` joins class strings without resolving Tailwind conflicts, so the `w-56`
  the title bar passed landed in the same class attribute and lost to the later `w-full` rule: the
  site switcher ate the whole header. `SelectProps` no longer carries `className` at all, which
  makes the mistake unexpressible, and the seven call sites that sized a select wrap it instead.
  `tailwind-merge` was considered and refused: `theme.css` renames the type scale, so `text-2xs`
  would be read as a colour and silently dropped against `text-ink` unless the merge config
  mirrored the theme, which is a second source of truth for a one-line problem.
- **The header carries context, not a second name.** The window has a system frame that already
  says Postulator, so the bar holds the site switcher at a fixed width and the current site's base
  URL beside it, which answers "am I on the docker site or the live one" at a glance. The
  `no-site` sentinel option is gone: a placeholder is what Radix has for an empty selection, and a
  fake row mixed into the site list was data pretending to be a choice.
- **An address on your own machine is not the risk the https rule exists for.** `AllowInsecure`
  already unlocked plain http per site, but the refusal said "must use https" while the switch
  that allows it sat further down the form, so the docker stack on `http://localhost:8089` read as
  a hard block. `kernel/addr.Local` answers what a local host is — loopback, private, link-local,
  `localhost` and the `.local`, `.localhost` and `.test` suffixes — and both places that decide a
  scheme read it, because the domain and the WordPress client must agree or one accepts a site the
  other refuses. The per-site consent stays for a public host over http; the form hides the switch
  where it cannot matter.
- **A chat names itself once, from the exchange, and never again.** Every conversation the UI
  starts was untitled, and the first message then became the title verbatim. The deterministic
  title still lands the moment the message is sent, so no list is ever empty, and when the first
  turn finishes the cheapest model in the catalog replaces it with three to six words. `Send` is
  not made slower by this and `agent.done` is not made to wait: the call happens after the event,
  and `agent.titled` tells the window to reread the list.
- **`title_settled` is the whole state the rule needs.** It is written whichever way the attempt
  goes, a rename sets it, and a conversation created with a title has it from birth, so a failed
  call cannot make turn five rename a chat about turn one. Migration 0020 sets it on every row
  that already carries a title, because retitling the client's existing chats behind their back
  would be the same bug in reverse.
- **The titler is a role, not a literal.** `llm.RoleTitler` resolves through the same profiles
  every other role does and defaults to `openai/gpt-5.6-luna`: the cheapest model of the provider
  every other default names, twenty times cheaper than the chat default. Pointing it at a gemini
  model would have left a client holding one OpenAI key with no names at all.

## Decisions taken on 2026-09-20 for the settings screen

Four routes were `NotBuilt` panels, eight of the twelve `SettingsService` methods and four of the
`ModelsService` ones were unreachable, and `HealthService.Ping` had no caller at all. The data
layer was already written: every hook existed and none was imported by a component.

- **The general tab is rendered from the schema, never from a list in TypeScript.** `Schema()`
  already answers every declared key sorted, with its type, default and bounds, so the screen walks
  it and picks a control from `Descriptor.Type`. A setting declared in Go appears here without a
  frontend edit, which is the whole reason the registry exists.
- **The prose lives in the copy module and a Go test proves the two sets match.** `Descriptor`
  carries no label and no explanation, and putting English into `internal/kernel` would make the
  settings registry a localisation surface. `copy.settings.keys` holds a label and a one-line
  explanation per key; `TestEveryDeclaredSettingCarriesCopy` in `internal/app`, the one package
  that composes every registration, reads `frontend/src/copy/index.ts` and fails in both
  directions — a declared key with no label, and a label naming a key nobody declares. A key that
  slips through anyway still renders a working control, under a warning naming it, because a dead
  control would be worse than an ugly one.
- **A row is written per field, not behind a Save button.** These are twenty-eight independent
  scalars behind a service that takes one key per call. A Save button would fan out twenty-eight
  calls and then have to explain "nine saved, one refused", which is a worse failure than one row
  turning red. The value is checked against the declared bounds before the call is made, so an
  out-of-range number never leaves the window.
- **"Changed" is a semantic comparison, not a textual one.** `60s` and `1m` are the same duration,
  and a screen that called one of them changed would be lying about the client's own settings.
  `sameValue` compares durations by length and everything else by identity.
- **`isDefault` from the backend is deliberately ignored.** It reports storage — whether a row
  exists — and a stored value equal to the default has exactly the effect of no row at all. There
  is no delete endpoint, so resetting writes the default back; reporting the row rather than the
  effect would tell the client about the database instead of about their software.
- **The tabs are routes, not component state.** `ui/tabs.tsx` is a controlled Radix machine whose
  state would have to be derived from the URL anyway, and `features/onboarding/readiness.ts`
  deep-links to `/settings/models`. A layout route with four `NavLink`s dressed as tab triggers
  gives the back button, `aria-current` and the deep link for nothing, and it is the idiom
  `app/rail.tsx` already uses.
- **`UsageSummary` with neither id now answers everything spent.** It refused that combination,
  and `app/statusbar.tsx` called it exactly that way, so the "Spent" figure in the status bar has
  been a hardcoded `$0.00` since it landed: the refusal is an `INVALID` without a field, which
  `react` maps to a form error that nothing renders. Deleting the figure would have been honest
  but poorer; `LLMCallRepo.SumAll` is one query with no key, so the guard now refuses only naming
  both, and the status bar and the settings screen show the real total. Naming both is still
  refused, because summing a run and a conversation together means nothing.

## Decisions taken on 2026-09-20 and 21 for the product frontend

Four sessions had written thirty-nine thousand lines of frontend in a day, and the result was five
competing ways to navigate, three page-header conventions, six hand-rolled segmented controls, a
legend explaining the graph, eighty paragraphs explaining the rest, four screens that were stubs
naming the wave that owed them, and an agent chat that said "Working" forever. Four explore agents
read it, eight workers rebuilt it under one orchestrator, and every screen was looked at rendered
for the first time. The plan is `docs/superpowers/plans/2026-09-20-phase-13-frontend.md`.

- **The foundation stayed and the layer above it was rebuilt.** `frontend/src/data`, `ui/theme.css`,
  the copy discipline, the canvas engine, the graph models and the pure modules under every
  feature were good and are untouched in substance; the shell, every screen's framing, the agent
  surfaces and the settings screen were rewritten. Rewriting the data layer would have thrown away
  the one part that was right.
- **One screen contract binds every screen.** A `Screen` primitive owns the 40px header (title, at
  most one badge, optional tabs or a view toggle in the centre, at most three actions), the body
  variants `plain`, `split` and `full`, and the 212px rail and 320px panel; `Toolbar`, `Tabs`,
  `Segmented`, `Menu` and `Kbd` are the one implementation of each. A primitive never sizes itself
  and never takes a layout `className`. Explanatory prose exists in exactly three places: an empty
  state, inline validation and a one-sentence tooltip. Legends are banned; a visual encoding is
  either self-evident or explained on the element, which is why the graph's legend became a hover
  card over the node. The contract is what makes eight workers' screens one product.
- **The rail carries a label under every icon.** The design mock had a 54px icon-only rail with
  tooltips; twelve icons a client must learn is a legend by another name. The rail is 68px, icon
  over a 10px word, in three sections: the site's screens, production, and Agent, Sites and
  Settings pinned at the bottom.
- **A mock is a reference, never a spec.** The five later Claude Design screens were built with the
  user's rule that nothing is transferred blindly: a worker owns the copy, the ergonomics and the
  layout, and draws nothing the backend cannot back. That is how Templates lost a per-section link
  count that `template.Section` does not have, Schedules lost a kind column because a schedule has
  no kind, Reports lost a cannibalisation tile with no site-level source, and Settings lost its
  whole layout.
- **Settings are laid out by intent; the schema drives validation, never the layout.** The mock
  and the previous decision rendered twenty-eight rows from the schema, grouped by Go package, each
  printing its backend key. That is a developer panel and the user called it terrible. Six tabs
  (Models, Runs, Agent, Browser, Security, About) hold the eight settings a client touches under
  human labels with the unit in the control; everything else sits in one collapsed Advanced card
  per tab. A test places every declared key exactly once, so a new Go setting still appears by
  itself, in the Advanced card of the tab its group maps to, and a new group fails the compile.
- **The agent turn is settled by Go's answer, never by a string in an error.** The frontend marked
  a turn "working" only after `Send` returned, while Go had already started the goroutine, so a
  fast terminal event was overwritten and nothing ever arrived again; Stop ignored `cancelled:false`
  and never touched the store. `AgentService.Status` now reports the turn registry, `Send` returns
  the assistant message id, `agent.done` carries a frozen `code`, and the store reconciles with
  Go after a send, on mount, on focus and after thirty seconds of silence. `CANCELLED` is the stop
  signal; the described message is the detail.
- **A live lock reaches the gate because the lock query is never removed.** `markLocked` cleared
  the whole query cache and then wrote the locked state, which removed the very query the gate
  observed; the observer kept its stale result and every read answered `LOCKED` behind a screen
  that did not change. The locked state is written first and every other query is removed after.
- **A link opens in Tor Browser or nowhere.** The client works in Tor Browser for its privacy, so
  the runtime's `Browser.OpenURL`, `window.open` and `target=` are banned by the import check and
  every external open goes through `BrowserService.Open`, which launches `firefox.exe` from a Tor
  Browser folder detached. Detection tries the usual install roots and accepts a `firefox.exe` only
  with a `TorBrowser` sibling; `browser.torPath` overrides it; a missing browser is `INVALID` with
  `details.code = "tor_missing"` and the toast leads to the Browser tab. The in-app themed preview
  stays, with "Open in Tor Browser" beside it.
- **The window is frameless and the header is the title bar.** No product mark: the header reads
  site pill, search, dock toggle, our own minimise, maximise and close. `F` opens search when no
  editable element has focus; the map's fit key moved to `0` for that reason. The app refuses to
  run twice through Wails' single-instance option.
- **A build-tagged harness is how the UI is seen.** `-tags uiharness` composes the real window over
  a fake model, an in-process fake WordPress that keeps wall time and survives a restart, and a
  seeded espresso site, with a DevTools port so `frontend/scripts/shot.mjs` can photograph any
  route. Every worker verified its screens there and the orchestrator reviewed from the PNGs. Two
  harness windows need two binary names, because WebView2 keys its user data on the executable.
- **A canvas host and a data-layer state machine are exempt from the 300-line rule.** `map.tsx`,
  `data/agent/turn.ts` and `domain/cron.ts` are each one cohesive unit whose parts need each
  other's internals; the rule is for feature files, which are composed from parts.
- **`llm.usage` is not a run-log event.** The registry declared it run-scoped while the ledger
  published it through the plain publisher, so the bridge refused it and the refusal came back in
  place of the model answer: every model call with usage failed as soon as a window was attached,
  and no gate saw it because the Go suite fakes the publisher and the harness seeds before the
  window exists. It is never appended to the run event store, so `ListEvents` could never replay
  it; it is registered without `Run`, the frontend invalidates the usage query instead of feeding
  it into the run log, and a release check starts a run after the window is up.
- **Escape closes an overlay; it never decides.** The confirmation card bound Escape to reject
  with no undo, and the screenshot walk destroyed a seeded proposal by pressing it. The card keeps
  Ctrl+Enter to approve; rejecting is a click or the inbox's `r`.
- **A provider error says what the provider said.** A working OpenAI key was reported as
  rejected: the Settings test probed the first catalog row, the costliest flagship, the key had no
  access to that model, and every 401 and 403 wore the same sentence. A 403 now reads "the key has
  no access to this model", the provider's own message travels in `details.providerMessage` with
  any key fragment masked to its last four characters, the test probes the provider's cheapest
  model, and the client cache is keyed on the key's fingerprint, so a rotated key is used at once.
- **The catalog is verified, not remembered, and every role has a reasoned default.** The
  shipped models were checked against the providers' current model and pricing pages; the
  flagship nobody should pay for by accident was removed. Defaults follow the step: the writer
  gets the best model whose page still costs cents, editor, judge and chat the mid-tier, linker
  and titler the cheapest, images the current image model.
- **A dropped file lands where the client is looking.** Wails delivers dropped paths to Go, which
  relays them as `files.dropped`; on Import the sheet becomes step one's file, anywhere else it
  opens the dock with a readable sentence asking the agent to inspect it, never to apply it.
- **A hidden input is positioned inside its label.** `sr-only` is `position: absolute`; without a
  containing block the switch's real input escaped every `overflow: hidden` and stretched the
  document past the window, so the whole application scrolled on the densest screen of switches.
  The labels are positioned and `html`, `body` and `#root` refuse to scroll.
- **Honest gaps stay visible.** Schedules say "UTC" because the backend evaluates cron in UTC;
  `Import.Apply` shows a busy state without a percentage because it is synchronous; the pages rail
  has no drift facet because `Pages.List` has none; a `Sites.List` search filters the loaded rows
  because the wire has no name filter. Each is in the plan's backlog rather than faked in the UI.

- **A reasoning model is given room to think, and the caller keeps budgeting the answer.** A
  working OpenAI key read `INVALID` because the Settings probe asked for one output token and the
  cheapest catalog row reasons: the budget went on thinking, nothing visible was emitted and the
  provider answered 400, which every status at or above 400 wore as "rejected the request". The two
  numbers are now separate. A caller states the answer it wants; `gollemclient` reads the catalog
  row and adds an allowance per declared reasoning effort, clamped by the row's max output.
  `finishReason` compares against that same ceiling, or every reasoning call would claim a length
  stop it never hit, and a 400 about the output limit says so. `reasoning_effort` is persisted
  since migration 0021, because an override row replaces the embedded one wholesale and saving any
  catalog row silently stripped it.
- **A model call names its step.** The two proposals did not, although their names were declared
  beside them. The ledger writes `Meta.Step`, so their spend was unattributable, and the scripted
  fake keys its replies on the same field, so the harness could never answer them: the button had
  never been seen working, and the schema refusal that produced was read as a provider fault.
- **A control is sized by a prop, never by a className.** `cx` joins strings and Tailwind resolves
  by CSS source order, so a caller's `w-14` or `h-6` loses to `Input`'s own `w-full` and
  `h-7`. Six such overrides were dead in five files. `Input` carries the size scale `Button`
  already had, width is the parent's job through a wrapper, and `ui/overrides.test.ts` reads the
  sources and names the next dead override by file and line.
- **A stored score of zero means unscored.** `Graph.Score` is PageRank normalised to a maximum of
  one, so a computed score is strictly positive and zero can only mean the recompute never ran. It
  reads as a dash, and "Highest scores" says so and offers the recompute instead of listing zeroes.
  Nothing recomputes on its own; the number is honest about being absent rather than invented.
- **A canvas label is measured twice.** `TextCache` never expired, and the first measure happens
  on a detached canvas before the self-hosted Archivo has loaded, so a node was sized for the
  fallback face and the text later painted into it overflowed. The cache is cleared once
  `document.fonts` settles and the layout is recomputed.

## Decisions taken on 2026-09-22 for the agent, its tools and what a turn costs

The client's own walk: the agent could not complete a single tool call, six approvals failed
after the click, the answer arrived as raw markdown, and the application reported $0.98 against
$0.39 on the provider's page.

- **A round of a turn is counted once, and every round is counted.** The turn summed the token
  counts of every streamed chunk. gollem's OpenAI session sends the full per-call totals twice on
  any round that produced tool calls, once beside the function calls and once as a trailing usage
  chunk, and gollem's own loop takes the latest non-zero value rather than summing for exactly
  that reason. The scripted fake now repeats its usage the way the real client does, because the
  fault was invisible while the fake reported once.
- **A cached input token is charged at the cached rate.** gollem reports it as
  `CacheReadInputToken` and nothing read the field, while a system prompt and eighty-seven tool
  schemas are resent on every round and are a cache hit after the first. Measured on a real key:
  136,307 input tokens of which about 86% were cache reads, $0.085 against the $0.297 the full
  rate would have charged. `CachedInputUSDPerM` of zero means undeclared and falls back to the
  fresh rate, so an override row written before migration 0022 cannot silently make a cache read
  free.
- **The strategy concludes without texts.** gollem appends the final texts to the session history
  itself, joined with newlines, on top of the copy the provider session already appended, so every
  later turn resent the answer twice and one copy had newlines inside its words. `readTheStream`
  returns an empty `ExecuteResponse` and the answer is read from the stream the window saw, which
  makes the saved message and the streamed one the same text by construction.
- **A tool field that names a domain choice carries its values.** Sixty of the tools hand `NewTool`
  a DTO written for the Wails window, and those carry no `enum` and no `description`, so every
  domain choice reached the model as a bare string and every refusal was the model guessing. Three
  tests in `internal/transport/wails/vocabgen` hold the line against the same Go const blocks
  `vocab.ts` is rendered from: a choice list equals a vocabulary or is listed as a deliberate
  narrowing, a field named like a choice offers one, and every field of a write says what it is
  for. The first of them caught `runs_start` offering four run kinds of six.
- **The template specification is typed.** It was an opaque JSON string because `TemplateSpec`
  carries two Go maps and the reflection refuses a map. The tools take the domain structs directly
  and replace only the maps: model profiles as a list of role, provider and model, and step params
  as the two keys a step reads. An override is a typed patch, so a misspelled key is refused at the
  call rather than inside `Resolve` later.
- **Arguments are read before a confirmation is written.** Decoding and domain validation used to
  happen for the first time inside `Confirm`, after the approval, so the client approved what could
  not run. `Tool.Check` decodes with `DisallowUnknownFields` and runs the domain validator where
  there is one. Unknown fields are refused everywhere, not only at the proposal: they used to be
  dropped in silence.
- **A result that arrives mid-turn is queued, never dropped.** `Confirm` swallowed the `Conflict`
  from `Turns.Start`, so approving two actions in a row lost the second every time. `Turns` holds
  the results against the running turn and delivers them as one turn when it ends.
- **A tree is one decision.** `graph_create_entities` takes the whole list, each entity naming its
  parent by name, resolves those names against the batch and the site, and writes the entities and
  their parent edges in one unit of work or none of them. Twenty entities used to be twenty cards.
- **The runner is as patient as the port.** It took a raw gollem client, so it went round the retry
  and limiter that stand on `llm.Client` and one 429 ended the turn. A content stream middleware
  holds the declared requests per minute and tries a refused call again with backoff, honouring
  `Retry-After`. A model whose rate is unknown is not held back at all, because treating unknown as
  one call per minute is worse than not limiting.
- **The prompt stops working against the model.** It forbade retrying a failed call, which left no
  way to self-correct, and asked for English however the client wrote. It now says to read the
  refusal and call again, to answer in the language of the question, to reach for the tool that
  takes a list, and never to claim something awaits the client when no tool was called. The tool
  names it repeated are dropped, since the tools are sent natively.
- **Spans nest.** The markdown reader is rewritten: escapes, underscore emphasis that never fires
  inside a word, strikethrough, images as their alt text, bare and angled addresses, code carrying
  a backtick, real nested lists inside the item they belong to, ordered lists as `ol` starting where
  they say, multi-line quotes holding blocks, table alignment and escaped pipes, and hard breaks.
  An unfinished marker still stays the text it is, because a streamed answer is read while it is
  still arriving. A code block names its language and copies itself; a link opens in Tor Browser
  through `BrowserService`, which is the only way this application opens anything.
- **The dock opens itself for a confirmation.** It is closed by default and the card lives inside
  it, so a confirmation raised while it was closed existed only as a badge. Only the newest card
  takes focus and none of them takes it from the composer mid-sentence; refusing is
  Ctrl+Shift+Enter, symmetric with approving.
- **The UI harness can talk to a real provider.** `POSTULATOR_OPENAI_KEY` leaves `Config.Provider`
  and `Config.AgentProvider` unset, seals the key into the harness home and skips the seeded runs,
  schedule and conversation, which would otherwise be real spend. That is how the numbers above
  were measured, against a seeded site and a home of its own rather than the client's.

## 2026-09-22 — the hierarchy, the verifier, the repair and the import

- **`parent: 0` means deliberately at the top level and never "unknown".** WordPress reads the
  same zero both ways, and the publish step handed it that zero for a parent the page map does
  not hold, for a dangling link and for a parent that is planned but not yet on the site.
  Seventeen imported pages were published flat and nothing said so. A parent that is not there
  yet holds the item as `waiting`, names what it waits for, and pauses for a human past
  `ParentWaitLimit`; a parent the map does not hold is `INVALID`.
- **The parent is read from the path, not from the cached link.** `parent_page_id` is a cache and
  a cache can lie, so `parentOf` takes `ParentPath(page.Path)` as the truth, refuses a page whose
  path names an ancestor the map has no page for, and refuses a link that disagrees with the
  path. `ParentPath` of a depth-one page is `/`, which is a legitimate ancestor path and not a
  WordPress ancestor, so a section is never nested under the front page even when the site has
  one and `pages.Create` linked it there.
- **Run items carry the order they were planned in.** Every item of a run was stamped with one
  `created_at` truncated to the second, so `ORDER BY created_at, id` collapsed to a random UUID
  order and a child published before its parent about half the time. `Engine.Enqueue` sorts the
  targets ancestors first and writes the index into `run_items.seq`, which covers the window, the
  agent and the scheduler at once.
- **A publish is not finished until what came back has been compared with what was asked for.**
  `Page.Observed` holds the site's own view beside the plan and `Page.Mismatches` is the pure
  comparison, judging only a field the plan named. The publish step compares immediately, because
  `relink_neighbors` downstream builds its links from the path; a disagreement is written once
  more with the parent and the slug it asked for, and if the site still answers something else
  the item pauses for a human naming both values.
- **A draft has no address.** WordPress answers `/?page_id=42` until a page is published, so the
  path is compared only once there is one, while the parent id and the slug, which are what
  produce that address, are compared on every write.
- **WordPress owns no field of the plan.** `sync_site.merge` assigned the path, slug, title,
  heading and meta from whatever the site answered and then rebuilt the parent link from that
  path, so one sync after a bad publish overwrote the imported plan with the damage. A page whose
  content this application wrote — the same non-empty `ContentHash` the drift rule already uses —
  keeps its plan; a page the sync discovers is adopted whole, because there is no plan to protect.
- **A repair moves a page without rewriting it.** `repair_hierarchy` needs no artifact and sends
  WordPress nothing but the parent and the slug, so the content and the title stand and WordPress
  recomputes the permalink itself. It is its own run kind because the publish step needs a draft
  and a body it would then write over.
- **A page can be taken off the site, not only out of the map.** `Delete{onSite}` trashes the
  WordPress page first, where a human can still restore it, and drops the local row only once the
  site agreed; the application reaches WordPress through the seam the preview link already uses.
- **A refusal a model reads is the message, not the details.** The agent saw "pause reason is not
  recognized", concluded `runs_pause` was defective and stopped trying, because `details` never
  reaches the model. The refusal now names the four reasons it would accept in its message.
- **A graph node says where its page stands.** `LoadGraph` answers a state per mapped page and the
  node is filled by it, with a disagreement above work in flight and work above the stored status,
  so the loudest colour is always the page that needs a human. A legend explains each state in a
  sentence and a badge names it in a word; they are two vocabularies because a badge has the width
  of a column and a legend has a line.
- **A workbook is more than its first sheet.** The reader took sheet one silently. Inspect names
  every sheet with its shape, a mapping says which to read, rows carry the sheet and line they
  came from, and sheets that do not agree on their columns are refused by name rather than
  stacked. `indentColumns` builds the path from the column a cell sits in, which is how a client
  who never had a path column writes one down, and `noHeader` addresses the columns by their
  spreadsheet letters because such a sheet has no header to name them by.
- **The agent gets a summary, the window gets the report.** `imports_preview` answered with one
  entry per planned page and the 16 KB tool-result cap cut it into invalid JSON at about sixty
  rows. The agent now reads counts, twenty sample pages and findings grouped by code with five
  examples each; `ImportService.Preview` still answers the whole report, because a window has no
  byte budget.

## 2026-09-22 and 23 — production hardening before the release

Four complaints came back from the client's own use: a link the body already carried was inserted
again and then graded as an error, the agent could not finish a tool call, a run that went wrong
had no way back, and the spend badge read three times the provider's invoice. Seven vectors were
run as four waves of two agents over one working tree — disjoint files, a commit lock, one
orchestrator gate on each wave's last commit — between `f739915` and `8da0391`. What follows is
the reasoning of each vector, in wave order.

### Links (V2)

- **One resolver answers where an href points.** Four classifiers compared bytes:
  `InsertLinks.existingFor`, `Compliance.classify` with an empty host, `LinkContext.ClassifyLink`
  with the site host, and a lookup by URL. `/shop`, `/Shop/`, `/shop/?utm=x`, `/shop/#top` and
  `https://own-host/shop/` were each a second insertion of a target the body already carried and
  then an `external_link` error, while the Linking screen called the same link `graph`.
  `pagemap.Site{Scheme, Host, Base}` is built once from the site's `BaseURL` and mirrors the
  companion plugin's `internal_href_to_path` rule by rule, so what the plugin records and what the
  domain judges cannot drift. `content.LinkContext` carries that site and every classifier resolves
  through it.
- **A same-document href is not a link between pages.** `#faq`, `?utm=1` and `href=""` resolve as
  `same_document`: compliance skips them, so a table-of-contents anchor is not turned into an error
  by `forbidSelf`, and nothing records a stored link with an empty target. `LinkClass` gained no new
  value, because it is rendered into `vocab.ts` and pinned by a test; the distinction lives inside
  Go as `Resolution.SameDocument`.
- **Self is the page being written, not its entity's canonical page.** A run graded a second page's
  link to the canonical page as `self_link` and failed the item, while the audit said the opposite.
  `content.PlanLinks` takes a `Subject{Site, PageID, PagePath, EntityID}`, and a page that is not
  its entity's canonical page carries that canonical page as an optional `up` target at depth 0 —
  `up` so the opening paragraphs are where it lands, optional so a missing link cannot fail an item,
  depth 0 so it is distinguishable from a real tree parent.
- **The stored link-policy rules are the base a template overrides.** `LinkPolicy.Rules` was
  written, validated, shown and read by nobody. `templates.EffectiveRules(policy, spec)` is the rule:
  a spec carrying any link rules replaces the policy's whole, a spec whose `LinkRules` is the zero
  value inherits them. `LinkRules` is a flat struct of primitives with no optionality, so a
  per-field merge cannot tell `downLinks: false` on purpose from `downLinks` unmentioned;
  whole-struct replacement is the only rule computable from what is stored, and it is what the
  Templates screen already implies.
- **A neighbour is relinked by its own rules or not at all.** `relink_neighbors` linked a neighbour
  with the current item's policy through a synthetic one-target context, so it wrote a backlink the
  neighbour's own plan never asked for and the audit then reported it as off-graph. The step
  resolves the neighbour's template, plans the neighbour against the graph, inserts only a target
  that plan names, and counts the cap over the neighbour's whole plan. A backlink nothing owes is a
  skipped outcome with a reason, never a write.
- **The cap counts the same thing on both sides.** Insertion counted graph links and compliance
  counted every `<a>`, so a page with two graph links and four footer links was over the cap on one
  side and under it on the other. Compliance counts links that resolve to a planned target and
  `too_many_links` names what it counted.
- **An anchor is a word.** `findFold` had no boundary check, so a target anchored on "art" linked
  the middle of "cart"; a match now needs a rune boundary on both sides. `rotate` rotates, and
  `childrenSection` is honoured by asking `generate_body` for a closing section that names every
  down target verbatim — which is where a down link finds its anchor.
- **A page path already carries the base path.** `owner.BaseURL + page.Path` produced
  `https://h/blog/blog/shop/` on a subdirectory install; the canonical URL and the preview link join
  the origin to the path through `Site.URL`.

### Agent reliability (V4)

- **A field is required only when the use case refuses its absence.** The schema generator marks a
  field required unless it is a pointer or carries `omitempty`, gollem validates the whole tree
  before the tool runs, and its refusal teaches the model nothing. Seventy-eight fields across
  seventeen tools demanded values their own use case would have defaulted; `templates_create` went
  from about thirty required values to two. Three tests over the whole registry hold the line by
  removing each required field in turn and calling the real use case.
- **A tool takes its own argument structs, not the window's.** This revises the 2026-09-22 ruling
  that the tools take the domain structs directly and replace only the maps. A DTO written for the
  Wails window carries the shape a form always fills; a model sends what it knows.
  `internal/application/tools` owns `templateSpecArgs`, `sectionArgs`, `policiesCreateArgs` and the
  rest, and converts them into the domain types.
- **A listing offers only the order its use case honours.** `runs_list_items` and `schedules_list`
  read `sort.desc` and ignore `sort.field`, so the field left their schemas rather than promising an
  order nothing keeps.
- **A cap shortens the answer, it does not truncate the bytes.** `Cap` halves the widest list, then
  the longest string at a rune boundary, and only then falls back to a preview. The result is
  `{truncated, totalBytes, droppedItems, shortenedText, result}`, so the model can read what it did
  get and knows exactly what to ask less of.
- **Every way a tool call can end reaches the client.** A name the registry does not hold was
  answered by gollem before the middleware chain, so it reached no ledger, no event and no screen;
  a content-stream middleware writes the same started/finished pair a real call writes. An unknown
  field and a wrong type are named to the model in its own sentence. A shortened result is its own
  row state, `cut`, so a capped answer no longer reads green.
- **A turn that answered is not failed by its own bookkeeping.** A late audit failure closes the
  turn with one synthetic row saying the record is incomplete; the answer stands.
- **The fence, the allow list and the cap live in the application layer.** They were written twice,
  once for the model's call and once for a confirmed one. `internal/application/agent/guard.go`
  holds `Fence`, `Permit` and `Cap`; transport adapts them to gollem middleware and `Confirm` calls
  them directly, so a confirmed tool is guarded by construction rather than by duplication.

### Reversibility and the run engine (V1)

- **A settle survives `Engine.Stop`.** `Stop` cancelled every item context and the base context, so
  `settle` began its transaction on a cancelled context, a step whose WordPress write had already
  happened lost its `step_execs` row, and the step ran again in full on the next start — creating a
  second page on the site. `settle` runs on `context.WithoutCancel` of the item context, a step the
  engine itself stopped is handed back rather than written down as a failure, and `Stop` requeues
  every item it still holds a lease on so a restart resumes at once instead of waiting out the
  lease.
- **A write to the site keeps what it replaced.** `PublishResult.previousContent` and
  `previousContentHash` are read before the write and only for an update; `NeighborResult.before`
  keeps the neighbour's raw content. They live in artifacts those steps already produce: there is
  no cheaper place, and nothing else can answer what was there before.
- **A revert is a run.** It has a kind, a parent run, one item per page the source run wrote to and
  one step, so it is paused, retried, watched and reported by everything that already handles runs.
  It undoes in the reverse of the order the work was done, a created page goes to the WordPress
  trash and its row returns to `planned`, an updated page has its body written back under a hash
  CAS, and running it again is quiet because a site already holding the hash is not written to.
- **A revert pauses the item, never the site.** A human edit since the run (both hashes named), a
  missing record, a page already gone, a site without the plugin: each holds one item and every
  other item still goes back. Media stays, for the reason the orphaned-media ruling already gives.
- **`revert` is not a step a recipe may name.** `StepName` is the vocabulary a recipe may draw
  from: `vocabgen` renders it into `vocab.ts`, the blank template builds its recipe from all of it
  and three tool enums must equal it exactly. `run.RevertStep` is an untyped const beside that
  block.
- **Migration 0025 rebuilds the `runs.kind` CHECK.** The constraint from migration 0014 never
  learned `repair`, so a repair run could not be inserted at all — the hierarchy repair the client's
  first real run needed would have failed on the insert. It now accepts `repair` and `revert`, and
  follows the 0016 pattern (`NO TRANSACTION`, foreign keys off, an explicit `BEGIN`/`COMMIT`)
  because dropping `runs` with foreign keys on cascades into four tables.
- **A sync knows which fields are a plan.** A page carries a plan when it is `planned`, when this
  application wrote its content, or when its path, title or heading differ from the mirror the last
  sync recorded. The mirror, not the site's current answer, is the comparison, because only the
  mirror is stable when the site changes under an adopted page. A sync never replaces a stored value
  with nothing, because core REST cannot see SEO meta at all and an empty answer means the site said
  nothing.
- **A link the run generated keeps its origin while the site still carries it.** Replacing every
  generated row with an observed one lost the only record that the application put the link there.
- **A step declares what one item of it costs.** The engine cannot know `generate_meta` is capped at
  512 tokens or that `repair_links` makes one call per required parent per iteration; the step does.
  An estimate over-prices rather than under-prices and names what it cannot price at all.
  `Budget.MaxTokens` holds a run exactly as `MaxUSD` does.
- **An import that fails puts back the database it replaced.** The live database is snapshotted
  before the archive is restored over it, and any failure after the detach rolls back, recomposes
  and returns the original error. The core stays locked only when the rollback itself cannot
  recompose, and the refusal then names the file the previous database is kept in.
- **A link needs a target url even when it names a target page**, and every producer goes through
  the constructor that says so.

### Agent cost and observability (V5)

- **A round is a ledger row; a turn is their sum.** One `llm_calls` row was written per turn with
  the summed usage of every round, so a twelve-round turn read as one call and the spend badge
  under-counted by the whole loop. The per-turn row is gone rather than kept as a sum, and
  `RunResult.USD` is the sum of the per-round costs, so what the badge shows and what the ledger
  holds cannot drift through two independent pricings. Rows are written outside the turn's
  cancellation, so a stopped turn is still billed for the rounds that ran.
- **A retry inside a round is not a second call.** `patience` wraps `observe`, so a round is one row
  however many attempts it took; the attempts are visible as `agent.waiting`, not as zero-token rows
  that would inflate the count.
- **Per-call usage for a conversation is an application event, not `llm.usage`.** `llm.usage` is a
  run event published with a run sequence and an agent turn has no run — the fault that failed every
  model call once a window was attached. `agent.usage` joins the `agent.*` family, which already
  carries a `conversationId`.
- **The model reads a whole tool result once and a shortened one ever after.** Every tool result the
  model saw was replayed in full on every round of every later turn. The stored copy goes through
  the same `Cap`, so it stays a document with its cursor intact, and the fence stays outermost
  exactly as it was live.
- **The tool schemas have a ceiling with a test on it.** Everything in the registry is resent on
  every round of every turn. `TestTheToolSchemasFitTheirCeiling` measures what goes on the wire and
  fails over `schemaCeilingBytes`, so a new tool is paid for consciously.
- **A rate limit is read out of whatever the provider gives us.** `go-openai` throws the response
  headers away on an error, so `Retry-After` is unreachable for OpenAI; the delay the provider
  states in its own sentence is parsed instead and honoured up to two minutes, beyond which the
  exponential backoff is the better answer than holding a turn open for six minutes. An Anthropic
  header still beats the sentence.
- **A turn that is waiting says so.** `agent.waiting` rather than a synthetic delta, which would
  corrupt the answer being assembled, or a synthetic tool row, which would put a call in the
  transcript and the ledger that nobody made. The silence rule holds off until the announced wait is
  over.
- **A saved row's status comes from the ledger, not from its sentence.** Matching prose for "is not
  open to this conversation" mislabels a refusal worded any other way and a failure that quotes it.
  `agent.Message` carries `toolStatus`.

### Relink and repair as operations (V3)

- **A run kind that owns its steps hands them out.** `run.Kind` had eight values and one of them
  named a recipe, so the Linking screen's Relink button and the drawer's `relink` and `repair` kinds
  fell through to the template's full generate recipe and republished the page. `Kind.Recipe()` is
  the one answer: `relink`, `repair`, `sync` and `revert` own theirs, a request recipe that disagrees
  is refused rather than quietly ignored, and `custom` is the one kind exempt from every step rule.
- **A template with no recipe of its own runs the generate recipe.** All five shipped seeds carry
  none, so no seeded template could start a run at all; `run.GenerateRecipe()` is the fallback.
- **A step a kind owns is not a `StepName`.** The four of them are `run.PerKindStepName` values,
  a type of their own, which keeps them out of `vocab.ts`, out of the blank template's recipe and
  out of the three tool step enums at once; `run.PerKindStep` puts the predicate in the domain and
  `runs.Start` refuses a recipe that enables one. That is what stopped a blank template's first
  run from failing at `repair_hierarchy`, which the blank recipe had enabled before `publish`.
- **A relink costs nothing and writes nothing but links.** It reads the page from the site, places
  what the page's own rules ask for against the page's own cap and writes back under a hash CAS. No
  model is called, so `Estimate` answers zero for `relink`, `repair` and `sync`.
- **A relink of a page produces a publish result as well as a relink result.** `sync_back` requires
  one and a recipe whose step requires an artifact nothing produces is refused; recording the page
  as it stands also makes a relink run revertible as a quiet no-op rather than a pause. Its targets
  go in a shape of their own, because a `relink_result` written by `relink_neighbors` lists pages
  whose bodies were replaced and a revert walks them.
- **A page a relink cannot verify holds the item; a site that cannot answer does not.** No plugin, a
  page gone and unreadable content are warnings that let the run finish; a page not on the site and a
  hash that moved under the write are pauses, and the CAS pause names both hashes.
- **`previousMeta` is kept beside a write, and the plugin learned the read it needed.** Companion
  plugin 1.2.0 serves `GET /seo-meta/{id}` behind the `seo_meta_read` capability, `wp.Client`
  gained `GetSEOMeta` and a `ReplaceSEOMeta` that can send an empty value — because the publish
  write must *not* be able to, and a revert must be able to clear a field that held nothing before
  the run. A revert puts back exactly the fields the run wrote, so a field a human changed since is
  untouched.
- **A capability the plugin does not advertise is refused before the request**, so a 1.1.0 plugin
  answers `plugin_outdated` from the cached manifest and the revert keeps its `revert_meta_kept`
  warning.
- **The fake WordPress is corrected to the plugin, not the other way round.** `wptest` skipped an
  empty SEO value where the plugin deletes the key; the fake is a second implementation of the same
  contract and a divergence there hides a real bug.
- **A move of an entity is one unit of work.** Add, acyclic check and delete share the transaction,
  so a failure cannot leave two parents, and running it again is quiet. An anchor's source is the
  actor's when nobody says otherwise — `ai` for the agent, `user` for a person or a schedule.

### The content steps (V6)

- **A score is a pointer, and absent means nobody scored it.** `judge` swallowed a provider outage
  into `Score: 0` and the report took the minimum, so one outage drove every page of a run to zero
  while the window drew a dash for it. `JudgeReport.Score` and `FinalReport.Score` are `*float64`,
  the report never takes a minimum with an absent score, and a failed judge raises a
  `judge_unavailable` warning carrying the provider's own words.
- **A type handed to a model and a type written to an artifact are two types.** `JudgeReport` and
  `Meta` were both, so any field added for the report would have been offered to the model to fill.
- **A step that could not do what its template asked writes a finding, not a string.** Every reason
  `generate_images` gave up reached a manifest string and no human; the final report is assembled
  from findings. A step also honours its own `Produces`, because the recipe validator only checks
  `Requires`.
- **A link context that says "no required phrases" must be the truth.** `resolve_context` answered
  an empty context for an entity the graph does not hold — reachable, because the entities are read
  twice — and the page went out with no internal links and scored 1. It refuses with `INVALID` and
  names the targets the graph asks for that no page carries.
- **A search snippet that is only the draft again says so.** A model answering `{}` produced a meta
  identical to what the page already said, silently; the borrowed fields are named in a
  `meta_not_written` warning.
- **A report names what it could not read.** A purged artifact is an `artifact_purged` warning
  rather than a section silently missing.
- **A Go type name is not part of a schema.** `llm.SchemaFor` stamped `Title: t.Name()` on every
  object, so a hundred and forty-three Go type names were resent on every round of every turn.
- **A derived limit becomes a declared setting the first time its composition root is in reach.**
  `agent.historyToolResultBytes` is read per turn through the Deps rather than once at startup
  through `Config`, so a change takes effect without a restart.
- **A budget a run outgrew names the tokens as well as the money**, so a run stopped by its token
  budget is announced rather than only paused.

### The client scenario (V7)

- **A run is paused, not left running, once nothing is left to advance.** `settleRun` asked
  `Status.Active`, which counts `paused`, so a run whose every item asked for a human stayed
  `running` for ever — the screens spin and `Runs.Revert` refuses a source run that is not terminal.
  It asks `Status.Advanceable` and pauses with `needs_human` and no `FinishedAt`. Found by the
  plugin-less revert, the only path where every item pauses.
- **A stop is read from the engine, not from a context.** An item context is cancelled by two
  writers, `Engine.Stop` and the step's own deadline, so reading the context cannot tell them apart
  and the same step was handed back or failed depending on which cancel won. `Stop` closes its
  channel before it cancels anything; `advance` asks `e.stopping()`.
- **`Document.HTML` is gone.** Its whole behaviour was to swallow the render error four steps now
  read, so it was deleted rather than left as a trap.
- **A fake reply may be a function of the request.** The bodies a scenario needs could not be
  written by hand: the phrases a page must carry are the anchors its up-links need, known only once
  the graph is in the store. `fake.Reply.Make` answers from the prompt the step actually rendered.
- **A scale guard measures an algorithmic regression, not a load.** Five wall-clock budgets summing
  to 1030 ms over a pass that costs 128 ms idle flaked under a loaded gate; one floor far above any
  load this machine can impose and far below any quadratic catches what the guard is for.
- **The frontend has two vitest projects**, `model` (node, `*.test.ts`) and `screens` (jsdom,
  `*.test.tsx`), because a per-file environment docblock is a comment and this codebase allows none.
- **A grid column that holds a badge or a label never opens with `minmax(0,…)`, and a control that
  renders text the caller does not control truncates in the primitive.** Five tables and one header
  painted over themselves at ordinary widths; every one of them was invisible to the tests and
  visible in a screenshot.
- **`task build` is part of the gate, not a formality.** `frontend/bindings/` is gitignored and only
  the build regenerates it, and the frontend import ban runs only there, so two defects of this wave
  were invisible to `npm run typecheck` and to `golangci-lint`.

## 2026-09-23 — after 2.0.0: the hold, regeneration, the page tree, cost and two docker stacks

The client's first real hour on OpenAI: fifteen pages, three parents failed validation on a
template that was fixed afterwards, their children stopped at Publish as "Paused for a human
decision" and stayed there, nothing could write a failed page again except a new run the agent
put together, and $1.20 left the account.

### A child and its parent

- **A child whose parent is not on the site is held, not handed to a human.** The publish step
  waited six sweeps and then paused `needs_human` with a sentence `settle` threw away, because a
  message was kept only from a fault. It now pauses at once as `awaiting_parent`, the step's
  sentence is kept in `run_items.note` and sent on `item.needs_human`, and the sweep's `unblock`
  pass releases every held item whose parent page has a `wpId`. The key is the parent *page*, so
  the same run, another run, a sync or a human publishing and syncing all release it; a run paused
  by a person or by its budget is left alone.
- **A paused run is waiting for a person, so its deadline no longer reaps it.** The deadline bounds
  work, not waiting. Every path that sets a run going again — resume, retry, regenerate, release —
  re-arms it from now. `updateRun` never wrote `deadline_at`, so a re-armed deadline could not have
  been kept before this either.
- **A run that publishes brings the parents that are not on the site.** Planning walks up from each
  chosen page and adds every parent with no `wpId`, stopping at the first one that has one; a parent
  another run is writing is left to it and the child waits. A parent mapped to nothing, a path whose
  parent the map does not hold and a link that disagrees with its path are refused before anything
  is queued. `Start` and `Estimate` answer what they added, so the window, the agent and the
  scheduler see the same run.
- **The item view names the parent it waits for** and, when that parent is an item of the same run,
  its status and step, so the drawer can say "it failed at Validate — regenerate it".

### Regenerating

- **A regeneration happens inside the run.** `Engine.Regenerate` puts stopped items back at the
  first step with a checkpoint whose `generation` is one higher; the checkpoint is part of every
  input hash, so no earlier execution is replayed. The item's artifacts go and its step executions
  stay as the record of what it cost. The template is resolved on every claim, so the regeneration
  writes against the template as it is now.
- **An item that wrote to the site is not regenerated in place.** Its `publish_result` holds what
  it replaced, and a second publish would record the run's own first draft as the thing to put
  back. The refusal names `published` and the window points at a new run instead.
- **Live totals fold each item to where it stands.** They added up events, so a retried or
  regenerated failure counted as failed and done at once.

### Cost

- **The writer is the mid tier and the editor and judge the cheapest.** writer `gpt-5.6-terra`,
  editor and judge `gpt-5.6-luna`, chat stays on terra. Defaults are not copied into the database,
  so an install follows them for every role nobody set by hand.
- **An image is a ledger call and asks for a quality.** The image request named no quality, so
  OpenAI chose, and the call went around the ledger: the spend tile, a run's totals and its budget
  cap never saw an image. `images.openaiQuality` defaults to `medium`, the adapter reads the usage
  the provider bills, and a decorator composed in the root writes one `llm_calls` row per image,
  priced from the catalog and attributed like a completion. The pre-run estimate still leaves
  images unpriced.

### The docker stacks

- **WordPress keeps its own copy of the plugin.** The plugin sources were bind-mounted read-write,
  so deleting the plugin in wp-admin deleted them from the repository and a reinstall failed on the
  mount point. `wp-cli` installs `bin/postulator-companion.zip` instead.
- **The e2e suites run on a stack of their own.** They shared `localhost:8089` with the person
  testing the app, force-deleted whole sections of it on start and left their scripted pages behind;
  the pages the owner took for a generation bug were the e2e client script's, written at 09:12 that
  morning, and three of them had become parents of real pages. `task e2e:*` runs the compose project
  `postulator-test` on 8088 — Windows reserves 8091 to 8190 on this machine — both harnesses refuse
  8089, and each whole-loop test removes what it wrote when it passes. `task sandbox:*` runs the
  existing `postulator-e2e` project on 8089 through a bootstrap that never deletes an application
  password or any content.
- **A tool enum is the whole vocabulary or none of it.** `runs_pause` offers `awaiting_parent`
  because the vocabulary test refuses a partial list, not because a person should pick it.

### Tor Browser

- **A link is a tab in the Tor Browser already open.** Every link started `firefox.exe` again, and
  Tor Browser, which disables remoting unless it is started with `--allow-remote`, answered with its
  "Close Tor Browser" dialog because its profile was in use. Postulator starts it with
  `--allow-remote`; when the hidden `Mozilla_…_<profile>_RemoteWindow` of this installation is on
  the desktop, the link goes with `-new-tab` and the second process exits at once. Checked on Tor
  Browser 15.0.23: three https links, one window, no dialog.
- **A Tor Browser opened by hand is left alone.** It takes no links, and a second start would only
  raise the dialog, so the adapter answers `CONFLICT` with `tor_closed_to_links`; the window says to
  close it and click again, and copies the link so it can be pasted into a tab meanwhile. An
  `about:` page is refused from outside and opens an empty window instead, which is why the check
  used https links.

## 2026-09-24 — 2.2.0: the pipeline that stops itself less, and control over what becomes an entity

Eighteen pages on the client's site: a third of them held for a human, links not placed, the
order wrong, every retry paid for again, and "the result does not satisfy the template" on a
heading the agent had written as `{primaryKeyword}`. The reasoning below is grouped by what the
client asked for; the code is under `internal/domain/content`, `internal/domain/template`,
`internal/runtime`, `internal/runtime/steps` and `internal/application/graph`.

### The import and the graph

- **An import never plans `/`.** `fillGaps` stops one level above the root, because the root of a
  site exists on WordPress before anything is imported; an explicit `/` row merges into the root
  the sync found or is skipped with `root_page_skipped`, and the pages of the first level keep no
  parent, which is what publish and start already expected.
- **A page keeps the keywords its row carried.** `pages.primary_keyword` and `pages.keywords`
  (migration 0027) hold what the import row said even when the row names no entity, so nothing is
  lost between the workbook and the proposal.
- **An entity is proposed for chosen pages and written only once reviewed.** `PreviewFromPages`
  answers proposals without writing, `ApplyProposals` writes the ones a person kept, and
  `ProposeFromPages` keeps doing both for the agent and takes `pageIds` or a `pathPrefix`.
  `ProposeFromKeywords` turns a pasted list into the same shape, one entity per keyword unless the
  model folds a near variant in, and the same apply writes it. The root is never proposed for.

### Placeholders

- **A placeholder is expanded once, for the page, before anyone reads the spec.** `{primaryKeyword}
  {entityName} {siteName} {pageTitle}` are the whole list; `template.Validate` refuses any other
  `{…}`, naming the four, so the agent corrects itself on the first answer, and
  `templates.ResolveForPage` expands them from the entity, the page and the site. The engine, the
  writer, the validator and the tool that resolves a template all see the expanded spec, so a
  heading is never compared with its own placeholder again.

### The writer and the linker

- **A draft is assembled from the brief, not trusted.** `content.NewBrief` says what the page owes
  (planned title and h1, numbered sections with their headings, the phrases the links need and
  where), the writer answers `DraftAnswer` with a slot per section, and `content.Assemble` puts the
  brief's headings on the sections, takes the plan's title and h1 first, falls back with a warning,
  strips markup a body may not carry, and turns a missing required section into `incomplete_answer`,
  which the engine tries again with more room. `section_missing` cannot happen by construction.
- **An answer that stopped short is tried again, not failed.** `llm.Structured` reads the finish
  reason first: a content filter holds the page, a length stop is `EXTERNAL output_truncated`,
  a malformed answer gets one repair round that shows the model its own answer, then `EXTERNAL`.
  The writer's ceiling is `max(4096, words × 3 + 1024)` doubled per attempt up to three times,
  the adapter clamps it to the model's maximum for every model, its own timeout no longer cuts a
  call the step already bounds, and the step allows fifteen minutes.
- **The linker owes phrases, not luck.** `repair_links` works out the required phrases the body
  still lacks (the first anchor of each required target where its link may go, the primary keyword
  in the first paragraph when the template asks), gives the model `iterations` tries per phrase,
  and then writes a plain sentence itself with a `phrase_templated` warning; a body without a
  paragraph gets one at the top. It never fails a page; only the run stopping stops it.
- **Validation grades; a person decides.** The draft's findings and the linker's repairs join the
  report, a planned h1 that lacks the keyword is a warning because the plan wins, and residual
  errors hold the page as `needs_human` with a note that lists them. `Engine.Accept`, reached as
  `RetryStep{acceptFindings: true}`, records `accept=<step>` in the checkpoint and the step goes on
  with its findings as they are; `allowErrors` on the recipe still waives them for a whole run.
- **No step fails a page on its own clock.** The judge and the image step turn their own deadline
  into a warning and keep what they placed, a cancellation still stops them, and the image step's
  timeout scales with the images asked for. A post is sent no parent and compared on none, because
  WordPress keeps none, and the fake mirrors that.
- **A bare wait is recorded as started, not done.** A step that answered `TransitionWait` with no
  artifact and no checkpoint used to be recorded `done` and skipped when the item woke.

### The queue

- **A child is queued behind its parent.** An item of a run that publishes records the item of its
  parent page in `run_items.blocked_by` (migration 0028, no foreign key so the down migration
  round-trips) when the parent is a planned page of the same run; the dispatcher skips it until
  the blocker completes or the parent page has a `wpId`, so a child never pays for a body it cannot
  publish yet. When the blocker fails, is cancelled or is held, the sweep parks the child as
  `awaiting_parent` with a note that names the parent and what to do, announces it and settles the
  run; it repeats until grandchildren are parked too. The release rule is one predicate for the
  dispatcher and the sweep: the blocker's page when there is a blocker, else the parent from the
  map. A post, a revert and a recipe without publish take no blocker.
- **Items are listed in the order they run**, by `seq`, and the view carries `blockedBy` and the
  page a queued item sits behind.

### Before the run costs anything

- **The estimate prices every page on its own template** and runs the preflight each step declares:
  a page mapped to nothing, a required link target without a page, an image source that is not
  configured, a template whose recipe differs from the run's, a site whose record says it has no
  companion plugin. The engine checks each model role it will call — a profile, an API key for the
  provider through `Keys.Has(llm.SecretRef(provider))`, a catalog entry — and prices images on the
  model that draws them (`Price.Ref`, 1,056 output tokens an image). A finding carries a severity
  and, where it concerns one page, its id and path; `Start` refuses a run with a blocking finding
  and `Estimate` shows them all. The finding for a differing recipe is not raised for a kind that
  owns its recipe.

### What the window sees

- **Every step leaves a sentence.** `step.done` carries the step's message and `step.retrying` the
  fault's code and sentence, so the feed and the timeline say what happened.
- **A resumed run is live again.** The log keeps the sequence of the last terminal event and of the
  last `run.started` or `run.resumed`; the run is terminal only while the terminal one is later, so
  a regeneration or a resume after completion refills the feed instead of freezing it. `step.started`
  refreshes the item rows, which is what made a run look paused while its steps moved.
- **The drawer picks its primary action from what stopped the page**: accept at validate, retry
  after exhausted attempts, regenerate after the writer gave up, the parent for a held child; the
  header counts the pages that wait for a decision, the items table names the page each item is
  queued after, and the start dialog grades each finding and refuses to start while one blocks.

### Decided without asking

- A post never waits for a parent in the queue and never gets one on the site.
- `primaryInHeading` on an ordinary section is a request to the model and a warning when it is not
  met, never an error; the heading is the template's.
- The seeded templates are unchanged on an existing install; only a fresh install sees them.

## 2026-09-25 — built-ins without images, images that are drawn, links that are owed and said

The owner's third week: the built-in templates asked for photos, a run over a template "with
photos" drew none and said nothing, a parent passed validation without its link to the child
the same run wrote, and the Linking screen called a page that only exists in the plan broken.
Links are what the product is for, so the rule of this section is that an owed link is placed or
said, never dropped in silence.

### Templates and images

- **The built-ins ask for no image.** Comparison, Guide, Hub and Product carry
  `images {featured: false, inline: 0, source: ""}`; their `generate_images` step stays enabled,
  because it is a no-op for a page that asks for none and it lets an override that turns images on
  draw them. This **supersedes** "the seeded templates are unchanged on an existing install": the
  copies the earlier releases shipped are embedded under `domain/template/seed/superseded`, and
  `EnsureSeeded` replaces a stored built-in whose spec still equals one of them with the current
  seed as a new version, unless one of its overrides would no longer resolve on it. An edited
  built-in is left as its owner saved it; the comparison decodes both sides through
  `TemplateSpec`, so a field added later cannot make an untouched copy look edited.
- **The OpenAI image adapter sends only what the model takes.** GPT image models reject
  `response_format` with a 400 and answer in base64 anyway, so every drawn image failed and the
  step recorded a warning nobody read. A `dall-e` model gets `response_format` and no quality, a
  GPT image model gets quality and no `response_format`, and one image may take four minutes.
- **The generate recipe draws what the template asks.** `run.GenerateRecipe()` carries
  `generate_images`; the estimate raises `images_step_off` for each page whose template asks for
  images the run's recipe will not draw; the template editor enables the step, and picks drawn
  images when no source is set, on the first edit that asks for an image. `template.Validate`
  does not refuse the combination, because `Resolve` validates and stored templates and overrides
  in that state would stop resolving.
- **The image step says how many it placed.** The manifest carries `wanted`, a library that
  picks fewer raises `images_short`, and the sentence is "placed N of M images", with the first
  reason when the page is short. Validation checks no image, on the owner's word.
- **The image model has one control**, `images.openaiModel`; the Images role left the model
  pickers because nothing read it, and the settings copy says the image settings apply after a
  restart.

### A template picked at the start

- **A template picked in the start drawer, a schedule or `runs_start` is assigned to the chosen
  pages**, and the run follows it. It was a label while every page was written, estimated and
  audited on its own template. `ResolveForPage` takes a template in place of the page's own and
  refuses one of another site; the estimate resolves each chosen page on the picked template
  without writing anything; `Start` checks the recipe, assigns through `pages.AssignTemplate` and
  then queues. The parents a run adds keep their own templates and a kind that owns its recipe
  assigns nothing. Assignment and queueing are not one transaction, because `Enqueue` owns its
  own and wakes the dispatcher after its commit; the only gap left is a database failure at
  enqueue, and then the pages keep the template the person picked.

### Owed links

- **The writer is told every link the budget allows.** The brief lists the anchor of every
  target within `maxLinks`, up, down and sideways, and puts the children in their own closing
  section when the *effective* rules ask for one; `generate_body` read the template's raw rules,
  so a template that inherits the site policy was never asked for its children.
- **The linker owes those phrases too.** `repair_links` asks for a sentence for any target whose
  anchor the body lacks, writes a plain one with `phrase_templated` when the model does not,
  places an up phrase where a parent link may go and a down or sibling phrase at the close of the
  page, and leaves a target the budget cannot hold. A down link the budget cannot hold stays a
  warning, on the owner's word, and is said on the finished page (below).
- **A link to a page that is not on the site is placed and named.** Linking to a planned path is
  what lets the link be there the moment the page is published; validation raises
  `target_not_published` for a link to a page with no WordPress id that this run does not write.
- **A neighbor that owes a page a link gets one.** After a publish, `relink_neighbors` visits the
  page's own targets and every page that may owe it a link (its parents, all its descendants and
  its related pages, `content.MayLinkTo`), keeping quiet about those that owe nothing; a neighbor
  that owes the link and carries no anchor for it gets a plain sentence with the link where its
  rules allow it (`relink_phrase_templated`, put back by a revert with the rest of the body), and
  a link that still cannot be placed raises `neighbor_link_missing`. A neighbor is planned on the
  site policy its own template overrides, not on the new page's rules. `relink_page` back-fills a
  target on the site the same way and names a spent budget as `target_missing`, but never writes
  a sentence for a page that is not on the site. This **supersedes** "a relink writes nothing but
  links": it writes a plain sentence carrying a link when the page owes one and has no anchor for
  it, and says so.

### What a finished page lacks

- **A finished page says what it lacks.** The report step words a notice (owed links missing and
  whether the budget was spent, links to pages not on the site yet, neighbors that could not link
  back, images short of what was asked for) and returns it as `run.Result.Notice`; the engine
  keeps the notice of the step that completes an item in `run_items.note` and sends it on
  `item.done`. The item table marks the page **Done, check**, the drawer opens on the notice and
  the run header counts such pages. No migration: the note column already carried a held
  page's sentence.

### The Linking screen

- **A link has a state**: placed, placed to a page not on the site yet, missing, waiting for its
  target, waiting for the page to be written, or blocked. A page is on the site when it has a
  WordPress id, the rule the engine uses. `Missing` counts real misses only; `Pending` and
  `Unpublished` carry the rest; an orphan is a page on the site that nothing links to, in the
  audit and in the site overview. The table says **Not written yet** for a planned page, the panel
  labels each owed link by its state and does not offer to relink a page that is not on the site,
  and the run's Links & compliance pane says it is the page as it stood at validation.

## 2026-10-02 — the client's sheets: keywords with volumes, an entity for every row, names under their parent

The client's SEO workbooks did not import: a list of keywords with their monthly volume in one
cell, `Root Entity | Category | Subcategory` level columns, a wide sheet an assistant wrote with
columns that mean nothing, and a variation sheet whose `Entity?` column would have created
entities named Yes and No. The client cannot share the files, so the work follows their headers.
The owner's rules for all of it: the file is the source of truth, used as it is; what it lacks is
generated; any cell may be empty and an empty cell erases nothing; everything is an entity except
a technical page; and no content is hard-coded, because a template says it. The WooCommerce half
of the approved plan was taken out by the owner and handed to its own session.

### Keywords

- **A page and an entity carry one ordered keyword list**, `keyword.List` of `{text, volume?}`,
  in place of a primary keyword and secondary keywords (migration 0029). The list is trimmed,
  deduplicated without case and sorted by volume, highest first, with unmeasured keywords after;
  the first is the main keyword, so `primaryInTitle` and `{primaryKeyword}` keep their names and
  read it. This **supersedes** "an existing entity keeps its spelling and its primary keyword"
  and "a page keeps the keywords its row carried": the volume decides the order, never the column.
- **A keyword cell reads `bpc 157 (12000), buy bpc 157 (5,400), bpc-157`.** Commas, semicolons,
  pipes and line breaks separate keywords outside brackets; a volume is the digits in trailing
  `(…)` or `[…]`, with `1,200`, `1 200` and `1.2k` read alike; a bracket holding words stays part
  of the keyword; a volume that cannot be read is a `bad_volume` warning and the keyword is kept
  without one. The export writes the same grammar in one column, so the keyword separator setting
  is gone; a `primary_keyword` column still reads, placed before the list is sorted.
- **A re-import merges** (`List.Merge`): the file's volume replaces the stored one, new keywords
  are added, none is dropped, and an empty cell changes nothing.
- **Generation reads the page's keywords, or its entity's** when the page carries none
  (`pagemap.Keywords`), in the brief, the placeholders, the writer, the meta, the image alt, the
  judge, `repair_links`, validation and cannibalization. The writer gets them numbered with their
  volume; `keywordRules.requiredKeywords` in a template says how many must appear, all by default;
  the missing ones are one `keywords_missing` warning in order of importance.

### Entities and names

- **An entity's name is unique under its parent, not across the site** (the owner's choice over
  qualifying names or a separate notion of variants). Liquid under BPC-157 and Liquid under TB-500
  are two entities named as the file names them. `entities.scope_entity_id` is the parent the name
  is unique under (migration 0030 rebuilds the table, with the unique index on
  `(site_id, coalesce(scope_entity_id, ''), name)`); the earliest approved parent edge sets it, a
  move or a removed edge settles it again (`graph.Settle`), and a clash is a conflict that names
  the parent.
- **A shared name is told apart by its parent wherever it is read.** `graph.Labels` gives "BPC-157
  Liquid" to the writer, the meta, the judge, the image alt, a link's fallback anchor,
  `{entityName}`, the link audit and the graph proposals, which resolve the model's answer by the
  same label; a name used once stays as it is, and a parent whose name is shared too is climbed.
  Lists, filters and search on the screens show "BPC-157 › Liquid".

### The import

- **The client's headers are read.** Recommended URL Layer and Canonical URL are the path, Entity
  Level is the kind (Compound/Product reads as product, GEO as custom, Commercial Taxonomy as
  category, Root and Pillar as hub), Parent Product Entity is the parent. A header that ends in a
  question mark is never detected, so `Entity?` stays ignored; `own_entity` is a field the client
  maps by hand, and `no` makes a technical page. Bind prefers the header exactly as written.
- **Level columns and note columns are options of a mapping**, detected and ticked in the wizard.
  A level cell of `-`, `—`, `n/a` or `none` is empty. A note column is kept on the page as
  `{label, text}`, shown on the page card, read by the writer as context it never copies, merged by
  label on a re-import and written back by the export.
- **Every row with a page gets an entity**, named by its entity cell, else its H1, else its title,
  else its slug; a row whose path the site already maps and that names none keeps that entity. The
  root row is the exception: it makes an entity only when it names one or the site holds the root.
- **A parent comes from the parent cell, else the row's deepest group, else the URL tree**, which
  also gives an entity to the intermediate pages the import fills in and to an unpublished,
  unmapped ancestor on the site. A technical page above a sheet page is `technical_parent`, because
  a run refuses a child whose unpublished parent carries no entity. When a parent the URL tree gave
  bears the child's own name, the two rows are one entity with two pages.
- **The level columns build a tree of groups, and a group takes a page only on evidence:** a row
  of its own level named as it, a row whose slug is its name, a row above all its other rows, or a
  free page above all of them whose slug is its name. Nothing is placed by position; a group
  without a page is kept and reported as `group_without_page`.
- **Entities are matched by their parent and their name**, parents first: under the same parent,
  else an orphan of that name at the top (adopted, and settled under its new parent), else, when
  the row gives no parent or only the URL tree's, the one entity of that name on the site. A name
  or a parent more than one entity carries is a blocking `ambiguous_entity` or `ambiguous_parent`;
  a named parent is looked for under the row's group first. A repeated import changes nothing.
- **The preview says what each column became** (`Mapping.Uses`: field, group, note, hierarchy or
  ignored), lists the groups with their pages and names each entity with its parent.

### Links past a group without a page

- **An entity that no page carries at all is passed through.** The link plan records it as an
  optional blocked target, `no_page`, and walks on to its parents or its children, so a form links
  up to its product's category; `MayLinkTo` climbs past it the same way. An entity that has pages
  but no canonical one still blocks, required, as `no_canonical_page`. This **supersedes** the
  rule that every parent is a required target, which made a group the sheet named without a page
  a link nobody could place.

### The pages screen

- **The page list filters by an entity and everything under it** (`includeDescendants`,
  `graph.Descendants`), so a category lists its subcategories' and products' pages.

### Decided without asking

- A revert treats a relink's before-copy as missing only when it has no hash: an empty neighbour
  that the relink wrote a sentence into is emptied again instead of handed to a human.
- The UI harness seed stores a key for each provider its profiles name, as the e2e harness does,
  and declares its posts at the flat addresses WordPress gives posts.
- The products stage is out of this branch: its plan, the owner's words and the research live in
  the handoff file the owner keeps, not in the repository.

## 2026-10-03 — the products the client creates in WooCommerce, edited where they stand

The client adds every product by hand in WooCommerce, with its name, price and stock, so
Postulator creates none: it edits a product the way it edits a page. It writes the description with
its links, the short description the store shows beside the price, the attributes the product
lacks and the SEO meta. The owner's rules are those of 2026-10-02, plus one: a product's name,
price, stock, SKU, status, slug and categories are never written. The plan that created products and
gave them addresses through the plugin was dropped when the owner said so; the plugin stays 1.2.0
and `wp-plugin/openapi.yaml` is unchanged.

### What a run writes into a product

- **The description is the body without its H1, written through the plugin's raw route** with
  compare-and-swap. WooCommerce passes `description` through kses for every user, so its REST API
  cannot write it as it was composed.
- **The store's fields go first and the description last.** A changed short description makes
  WooCommerce save the whole post again through `wp_update_post`, and `content_save_pre` runs kses
  over `post_content` for a user without `unfiltered_html`. Written after that save, and against a
  hash read again after it, the description stays what the run wrote. This **supersedes** the
  Phase 3 line that `UpdateProduct` is the one write path into a shop: the description goes
  through the raw route, and the short description, the attributes and the image through
  `UpdateProduct`.
- **Attributes are filled where the product lacks them.** The template names them; one is written
  only when the product carries no attribute of that name, compared without case, or an empty one.
  A global, a variation or a filled attribute is sent back untouched, because WooCommerce replaces
  the whole list, and a variable product keeps its attributes as they are with `product_type_kept`.
  The featured image is set only on a product with none.
- **The name stays as the store has it.** The writer writes under the store's name, a file H1 that
  differs is `product_name_differs`, and the primary keyword missing from the name is a warning,
  because only the client can rename a product.
- **A product is edited live.** It has no draft copy, so a draft run over a product is refused
  before anything is spent, `product_edited_live`.
- **The publish result keeps what the product held and what the run wrote**, as the store answered
  it, so a revert can tell its own work from a human's.

### Before anything is spent

- A run over a product is refused when the site's store is not known to be editable
  (`commerce_unknown`, `commerce_absent`, `commerce_forbidden`), when the plugin cannot write raw
  content (`product_needs_plugin`), when the row has no product (`product_not_in_store`) and when it
  is a draft run; a product category is refused, `product_category_unwritable`, because the plugin
  does not write a term's description. A template with no product outputs over a product, and one
  with them over a page, are warnings. A repair refuses both, `store_placed`.

### The revert

- **A product changed since the run is handed to a human**: a short description, an attribute the
  run filled or a description that is no longer what the run wrote. Otherwise the store's fields
  go first and the description last, against its own hash. Only the attributes the run filled are
  taken back, and one a human removed since stays removed; only the image the run set is taken
  off. The REST API has no compare-and-swap, so a human edit between the revert's read and its
  write of the store's fields is the one race left, and the description keeps its own.
- **A body counts as never copied only when no hash was kept**, so an empty description is put
  back as empty, the rule a relink's neighbour already followed.
- **A product or a product category is not deleted on the site** by `pages.Delete{onSite}`; the
  client deletes it in WooCommerce.

### Where a product sits

- **The store decides a product's address.** The row's path is the store's, and the address the
  sheet gave is kept beside it as `pages.planned_path`, migration 0032, which a second import reads
  and the export writes. No product takes a parent from its path, and the level above a run of
  products stays an entity without a page, `intermediate_level`.
- **A site knows whether its store can be edited**, `sites.commerce`, migration 0031: absent when
  WooCommerce answers no route, forbidden when the user may not edit products, ready otherwise. The
  probe, the plugin check and the first batch of a sync set it, and a 401 or a 5xx keeps what was
  known.
- **A row is matched by its id and its type**, because post ids and term ids are two sequences, and
  a path only within its family of types, `path_taken_on_site` otherwise.

### The import

- **A mapping says what a new row becomes**: pages by default, products, or read by the entity
  kind. A row with product rows under it stays a page, a `wp_type` cell wins over the mode, and a
  row with a WordPress id keeps its type, `wp_type_kept`.
- **A product row finds the product the client created** by its address, then by the slug the URL
  ends in, then by the one product whose name is the row's H1 or title. Ambiguous or not found, the
  row waits with `product_not_in_store` and no invented title. A product the sync meets later claims
  the row that waited for it through the same matcher, read from the row's side: only when the
  product is the one the row itself would match, and only when no other row claims it.

### What the screens and the read-back say

- The site's detail says whether the store can be edited, the template editor has a Product group,
  the import options say what the rows become and the preview names the product a row found, a
  product's card shows both addresses, the store's name against the sheet's H1 and the outputs of
  its last run, and the run panes show what a product was written with.
- **`sync_back` opens a published product's page as a visitor would** and warns,
  `product_description_hidden`, when the opening words of its description are not in the text a
  visitor reads; it ignores the head, scripts and styles, where WooCommerce's structured data
  repeats the description, and it compares words, so a theme's typographic quotes do not count. A
  page builder, a cache and the store's coming-soon mode are the causes it names;
  `product_page_unread` says the page could not be opened.

### A correction

- **Phase 3B says a term id is a 404 on the plugin's post routes; it is not.** `get_post()` reads
  the number as a post id, so a term id that a post also carries answers that post. The client
  names the item type on every plugin route, refuses a product category before the request, and
  treats a raw read whose type differs from the one it asked for as `NotFound`.

### Decided without asking

- The tool schemas grew to 84,923 bytes with the row type, and the ceiling is 85,000.
- A page report carries a draft's product outputs and nothing else of the draft, so the agent's
  context is not filled with section HTML.
- An import that finds a store's product for a row that waited under the sheet's address leaves
  the waiting row and says `product_row_left`: an import deletes nothing.
- The sandbox starts without WooCommerce unless `E2E_WOO=1`, and the UI harness stocks its fake
  store with a product it runs the shipped Product template over.

## 2026-10-03 and 04 — what a model call costs: OpenAI only, our own client, the spend in view

The client paid OpenAI $10 and could not say what it bought: the window showed one total. The
ledger already kept a row per call, but nothing read it by purpose, and the research found where
the money could go without anyone seeing it. Reasoning was billed as output and hidden, because
gollem folded it in. One stubborn page could pay the writer about fifteen times, the ceiling
doubling three times over `Retry{Max:3}`. The agent resent 93 tool schemas, about 21,000 tokens,
on every round of up to twelve, and the site context in its system prompt made the first round
after every write cold. GPT-5.6 bills a cache write at 1.25 times a fresh input token, and its
implicit cache puts the breakpoint at the end of the latest message, so a single call wrote its
whole prompt into a cache it never read again. gollem spoke Chat Completions only: no service
tier, no prompt cache key, strict schemas off, no reasoning count, response headers dropped. Five
spend bugs sat beside them: a catalog save dropped the cached price, an `insufficient_quota` was
retried five times, an on-demand judge wrote rows without a step, a cancelled stream wrote no row,
and the agent went round the retry, limiter and ledger chain. The owner decided on 2026-10-03:
OpenAI only, on our own client for the Responses API; old conversations keep their transcript and
lose the model's memory of them; the writer on flex with a fallback, every role's effort as it was.

### OpenAI only

- **Postulator works with OpenAI alone.** Anthropic, Gemini and gemini-openai leave the code, the
  catalog, the settings and the screens, and gollem, `anthropic-sdk-go`, `go-openai` and the Google
  SDKs leave `go.mod`. The embedded catalog keeps `gpt-5.6-{sol,terra,luna}` and `gpt-image-2`; an
  override row of another provider is passed over, `ModelInfo.Validate` and `profiles.Set` refuse
  one, and profile resolution passes over one at the template, the site and the global level and
  falls to the next. Migration 0038 deletes the global profiles of the removed providers so a role
  falls back to the catalog's default. **This supersedes** the Phase 4 lines on gollem's limits and
  on the catalog's other vendors.
- **A template that pins a removed provider stays saveable.** `template.Validate` stays permissive,
  and the estimate warns `model_provider_removed`, naming the pinned model and the one the role uses
  instead, so a client's template is never locked out of editing by an upgrade.
- **The keys of removed providers stay sealed** in the secret store, unused; the key list is read
  from the catalog, so they are no longer shown.

### Our own Responses client

- **`internal/adapters/llm/openai` speaks `POST /v1/responses` over `net/http`**, and
  `openaitest` is its httptest fake. The fake never imports `openai`, for the reason `wptest`
  never imports `wp`: it is a second implementation of what the live server did, and it refuses
  what the server refused (minimal effort, sampling at an effort, an unknown tier, a ceiling under
  16, a cache key over 64, a loose strict schema, an unpaired call, a reasoning item without its
  encrypted content).
- **The live probe cost nothing.** A key with no credit is answered 429
  `credit_balance_exhausted` only after the request has been validated, so every request shape the
  client sends was proven against the real API on 2026-10-03 without spending a cent. What only a
  funded key can show is listed in `STATUS.md` for the pre-release smoke; the probe's bodies were
  kept out of the repository.
- **What a request carries.** Instructions, typed input items (messages, function calls without an
  id, their outputs), `store: false`, and `max_output_tokens` as the asked ceiling plus the effort's
  reasoning allowance, clamped to the catalog row and never under 16. An effort is always sent to a
  model that reasons, because the server's default is medium even on luna, and a row that says
  `minimal`, which the API refuses, goes out as low. No temperature or `top_p`, which a reasoning
  effort refuses. Markup travels as written: the default encoder sent page HTML as unicode escapes.
- **A structured answer is one strict call.** The `json_schema` format is strict: every object
  closed, every property required, an optional one nullable. The JSON instruction and the repair
  round go, which **supersedes** the Phase 4 line that `Structured` appends an instruction and
  repairs one decode failure, and the 2.2.0 line that a malformed answer gets a repair round. An
  answer that does not decode is `EXTERNAL malformed_answer` for the step's own retry; a refusal
  still holds the page and a truncation is still `output_truncated`.
- **A refusal is classified from the provider's own code.** A quota, a spending limit or an
  exhausted credit balance is `NEEDS_HUMAN` and never retried, and its sentence says to add credit;
  it used to be a rate limit tried five times. A rate limit waits for `retry-after-ms`, then
  `Retry-After` in seconds or as a date, then the reset of a bucket whose remaining count is zero,
  then the provider's own sentence, up to two minutes, beyond which the backoff decides. This
  **supersedes** the V5 line that `Retry-After` is unreachable for OpenAI, and closes that gap. A
  key quoted back in a message is masked.
- **An error can arrive inside a stream that answered 200.** The probe saw `response.created`, an
  `error` event and `response.failed`, with no `[DONE]`. A refusal before the model has spoken is a
  failed start, classified exactly like the same refusal over HTTP, so the retry and the flex
  fallback work on a stream; one after it is the stream's last delta.
- **Flex, with one fallback.** A request asked on flex for a model the catalog prices on flex goes
  out on flex and is sent once more, at once, on the default tier when the provider has no flex
  capacity (a 429 or an in-stream refusal that names resource unavailability or capacity and no
  quota) or when nothing arrived within `llm.flexPatience`; a stream waits its patience only until
  the provider admits it. A credit refusal, an ordinary rate limit and a caller who left are never
  resent, and the answer carries the tier and the usage of the attempt that answered. The capacity
  body is assumed, not probed.
- **A single call neither writes nor reads the prompt cache.** A call with no tools and no cache
  key is sent `prompt_cache_options {mode: explicit}` with no breakpoint, which writes nothing and
  pays no 1.25 times; the agent names its prefix family with `prompt_cache_key` `chat:<mode>`, and a
  key past 64 characters is sent as its hash. The runs therefore price no cache write.

### Effort and speed per role

- **Reasoning effort and service tier are settings per role**, `llm.effort.<writer|editor|linker|
  judge|titler>` (medium, low, low, low, none) and `llm.tier.<role>` (`standard` or `flex`, the
  writer on flex), with `llm.flexPatience` (8 minutes, between 30 seconds and 14). `tuning.Client`
  fills `Effort` and `Tier` from `Meta.Role` when a request leaves them empty and is the outermost
  link, tuning → retry → limiter → ledger → recordreplay → openai, so the ledger and a fixture see
  what was sent. Chat stays at effort none on the default tier and is not a setting. Effort and
  tier are read live; the flex patience, `llm.timeout` and `llm.openai.baseUrl` when the core is
  composed, so they take effect after a restart.
- **The effort leaves the catalog row** (migration 0037; its down brings the column back with the
  `CHECK` of 0021), so a role's effort has one source. A call made for no role is sent the effort
  it asks for, else low on a model that reasons, and the provider test asks for none and sixteen
  tokens, so a probe never pays for thinking. `max` is accepted by the API and not offered, because
  no allowance for it is verified to price it with.
- **Every model call names its role.** The steps resolve the model and the role together, the graph
  proposals are the editor's, `Assess` always books the judge, and an audit on demand is booked as
  the judge step with no run, so its rows count as audit spend.

### The writer, the estimate and the prices

- **The writer is paid for at most two truncated answers.** A truncated answer is tried once with
  double the room, and a truncation at double the room holds the page `needs_human` with a note
  that says it is unfinished and to lower the template's word counts or choose another model. This
  **supersedes** the 2.2.0 ceiling doubled per attempt up to three times. A throttled call or a
  malformed answer keeps the step's retry, and a later attempt keeps the doubled room, which costs
  nothing unless it is written. No pause reason was added, because one would ripple into
  `vocab.ts` and the runs screens for a sentence the note already carries.
- **The estimate prices each step at its role's tier**, at flex prices where the row offers them,
  and a model that reasons adds half the allowance of the role's effort as output (none 0, low
  2,048, medium 8,192, high 24,576, xhigh 49,152). `runtime.Deps.Tuning` is required, like `Keys`.
- **GPT-5.6 prices a cache write at 1.25 times its input and flex at half the standard tier**
  (sol 2 / 0.20 / 2.50 / 10, terra 1 / 0.10 / 1.25 / 6, luna 0.10 / 0.01 / 0.125 / 0.60 per
  million for input, cached input, cache write and output). `llm.Cost` charges the ordinary input,
  the cached reads, the cache writes and the output each once at its own rate; reasoning is inside
  the output and never charged twice.
- **A catalog save keeps every price**: `UpsertModel` never copied the cached price, so after any
  save the ledger booked cached reads at the full rate. The request now carries all eight prices.
- **An override takes the base price for every price it left at zero.** An override replaces its
  whole base row, so one saved before the flex and cache-write prices existed priced flex at
  nothing and offered no flex, and one saved before the fix above booked cached reads in full. A
  price of exactly zero is never meant for an OpenAI model, so the catalog fills each zero of an
  override from the built-in row it overrides, which repaired the stored rows without a migration.
  **A ruling:** a built-in model's price can no longer be set to 0; the tier setting, not a price,
  turns flex off. A model the built-in list does not carry keeps its zeros, and there a zero cached
  price still falls back to the fresh rate, as decided on 2026-09-22.

### Spend in view

- **The ledger books what the provider served.** `llm_calls` gains `reasoning_tokens`,
  `cache_write_tokens` and `service_tier` (migration 0035, with an index on `created_at`; the tier
  carries no `CHECK`, so a new tier needs no rebuild, and an empty one reads as default), and
  `model_catalog` the cache-write and the four flex prices (0036). A call is priced at the tier it
  was served at, `llm.usage` names the tier, the reasoning and the cache writes, a stream its
  caller cancelled is still a row with the usage that arrived and `CANCELLED`, and every row is
  written past the caller's cancellation.
- **The purpose of a call is derived, never stored.** `llm.PurposeRules` maps a run and a step to
  `run`, `chat`, `title`, `probe`, `graph`, `audit` or `other`, and the SQL that groups the ledger is
  built from those rules, so the Go and the query cannot disagree.
- **`ModelsService.SpendReport` is an aggregate, not a list, so it takes no cursor.** It totals the
  ledger over the last 1 to 366 days (30 when left out), or one run step by step, into one slice
  per purpose, provider, model and tier: a number of rows bounded by the catalog, which the cursor
  rule is not about. `ListCalls` is the ledger itself, cursor-paged, newest first, and the first
  caller `ledger.List` ever had.
- **Settings → Models says where the money went**: a week, a month or a quarter; calls and failed
  calls; the cached, reasoning and flex shares; spend by purpose and by model and tier; and the
  recent calls, where an exhausted OpenAI account says to add credit. Reports → Runs says what each
  step of a run cost. The window refetches on `llm.usage` and also on the events that end something
  with a failed call, because a refused call spends no tokens and publishes no `llm.usage`.

### The agent on the port

- **A turn runs on the same `llm.Client` chain as the runs.** The runner streams each round,
  dispatches the calls in order through the same guard (unknown tool, `Permit`, the argument check
  gollem used to make, now `transport/agent/arguments.go` with its wording, `Authorize`, `Run`,
  `Cap`, the audit, `Fence`), saves the history after every complete round and ends with a budget
  refusal at the loop limit. Waits reach the window as `agent.waiting` through the retry observer.
  `patience.go`, the runner's own ledger writes, `Config.AgentProvider` and the gollem factory are
  gone, and `Config.Provider` is the one seam a harness uses. The dependency test now says that
  `internal/transport/agent` never reaches `internal/adapters/llm/openai`.
- **Every attempt is a ledger row.** The ledger sits inside the retry, so a refused attempt is a
  row with its status. This **supersedes** the V5 line that a retry inside a round is not a second
  call; the spend panel counts the `ok` rows as calls and every row's dollars as spend, and the
  turn's `calls` counts the rounds that answered.
- **The history is ours: `{"format":"responses/1","items":[…]}`** of port messages, with call
  arguments masked by `tools.Redact`, results shortened to the history cap with the fence kept, and
  trimmed by whole turns. Any other body loads empty, so a conversation from before keeps its
  transcript and the model forgets it, as the owner chose. This **supersedes** the Phases 9 and 10
  line that the gollem history is a blob whose shape gollem owns. Reasoning items are never kept.
- **The site context is a developer message.** The instructions are static per mode and the site's
  counts open each turn as a developer message, so the cached prefix survives a write.
- **The fake and the fixtures speak the port.** The scripted fake calls tools on the port from the
  turn's own user message (`TOOL:`, `FAKE:`, `FAIL:`, `ERROR:`), and `recordreplay` keeps the calls,
  the finish, the tier and the tool searches, masks tool arguments with the composition root's
  redaction before a fixture is named, and never records a stream that did not finish.

### What a round of the agent sends

- **The tool schemas say what a field takes in fewer words.** Restated types and boilerplate went
  from the tool and argument descriptions, which keep their limits, defaults and the tool each id
  comes from; the chat instructions say once that an id is used as a read tool returned it and that
  a field left out of a change keeps its current value. Two cuts, the tool descriptions first
  (85,991 bytes to 76,479) and then the request structs' field descriptions (76,347, after the
  category fields, to 63,304), leave 26.4% less than before, about 15,800 tokens a round, and the
  ceiling follows to 63,400. The ceiling test names the widest schemas.
- **Loading tools on demand waits behind `agent.toolLoading`, `all` by default.** `deferred` sends
  ten orienting reads whole (`sites_list`, `sites_get`, `reports_site_overview`,
  `graph_list_entities`, `pages_list`, `pages_get`, `pages_tree`, `runs_list`, `runs_get`,
  `templates_list`) and every other tool inside its group, the tool name's prefix, as `defer_loading`
  functions of a namespace beside a `tool_search` tool: 6,506 bytes a round before any search,
  against 63,304. A turn replays the searches and the namespaced calls of its own earlier rounds and
  stores neither, so the next turn searches again whatever the setting is by then. The probe proved
  the shape is accepted, not that the model searches well, so the default stays until the seven
  checks in `STATUS.md` pass on a funded key.

### Decided without asking

- A flex refusal for capacity is assumed unbilled, and a flex attempt abandoned for patience
  returns no usage, so if OpenAI bills the partial work it is not metered.
- The timing tests that raced a loaded machine now ring their own clocks: the flex patience waits
  on an alarm the test arms, a run is awaited until every event it recorded reached the bus, and
  the client timeout is tested through a transport that always sees the request.
- The fixtures that named another provider name OpenAI models, and a refusal of another provider
  is shown with a made-up one.

## 2026-10-04 — WordPress categories, their own records, and the whole workbook in one import

*Superseded the same day by "no categories: the client's Category columns are WooCommerce's" at
the end of this file, all but the whole workbook; kept as history.*

The client writes Category and Subcategory in his sheets, and none of it reached the site,
although categories are a second way through it for his visitors; and a workbook imported one sheet
at a time, in an order that mattered. The owner's decisions of 2026-10-03: only the Category,
Subcategory and deeper columns are WordPress categories, never Root Entity; a page is filed under
the whole chain, a missing term is created and an existing one reused; a product gains our product
categories and never loses the client's, and a revert takes back only ours; a workbook is one
preview and one apply. The first model built that day made a category a flag on an entity. On
2026-10-04 the owner corrected it: categories are WordPress categories, entities are entities,
different records, never mixed.

### The model

- **A category is its own record**, `internal/domain/category`: a site, a name, the key the name
  folds to and a parent, with `Chain` reading a leaf's path root first and stopping at a loop.
  `categories` (migration 0039) keeps a key unique under its parent and at the top of its site, a
  child goes with its parent and everything with the site; `category_terms` (0040) keeps one term
  per category and taxonomy, `category` or `product_cat`; `pages.category_id` (0041) is the leaf a
  page is filed under, `''` for none.
- **`pages.category_id` has no foreign key**, for the reason `run_items.blocked_by` has none:
  SQLite's `DROP COLUMN` refuses a constrained column, so the down migration could not round-trip.
  `CategoryRepo.Delete` therefore removes the branch, whose terms cascade, and files every page
  under it under no category itself.
- **`category.Key` is the rule WordPress compares term names by**: decoded once, trimmed, inner
  whitespace collapsed, a non-breaking space included, and Unicode case folded, so two names share
  a key exactly when `wp.SameTermName` says they match. It is deliberately not `graph.Key`, which
  trims and lowers ASCII letters the way SQLite's `NOCASE` does, because an entity's name is
  refused by the store under that rule and a category's by WordPress under the other.
- **A page is filed by its category, never by its entity.** One leaf per page; a technical page or a
  page whose entity was deleted is filed when it has a category, and a page with an entity and no
  category is not. Nothing files a page by hand: a page planned by a person or the agent is filed
  once a sheet names its category.
- **The flag model is undone by new migrations, not edited out.** `entities.site_category` (0033),
  `entity_terms` (0034), `graph.CategoryChain`, the term repository and `siteCategory` in the views
  and the `graph_update_entity` tool are gone; 0033 and 0034 may already be applied to a database on
  `dev`, so 0042 and 0043 drop what they made and each down brings it back exactly. Nothing is
  converted: the work was unreleased, and the flag model had flagged roots too.

### The import

- **Only a root level makes a group.** `Root Entity` and `Root` columns make entity groups as
  before; every other level column gives the row a chain of category names and makes no group, no
  entity, no parent edge and no `group_without_page`. This **supersedes** the 2026-10-02 line that
  the level columns build a tree of groups. A row with only category levels and no path, entity or
  root is skipped, `no_target`.
- **A root's name is never a category.** A category level whose key is in the request's root set is
  dropped from the chain, the next level hanging on the previous kept one, and reported with the
  sheet and row as `category_level_is_root`, which blocks nothing. The root set is every name in a
  root column of any chosen sheet; the name of every entity of the site with no parent whose kind
  is hub or category; and the name of every such entity the request itself plans, when one of its
  category levels uses it. For that third part the request is planned again until the set stops
  growing. Peptides is the Root Entity of the Groups sheet and the Category of Catalog and
  Entities, and it never becomes a category, whether the sheets come together or one by one.
- **Why hub or category, and not every entity at the top.** Counting every parentless entity made
  a top-level topic, a Blends page, remove the category Blends on the second import of the sheet
  that created both. A dropped level keeps its place among the row's levels, so it still tells
  apart two entities of one name below it.
- **One edge stays open by that rule:** the site's half is read as the site was before the import,
  so a request that relabels a top-level hub as a topic and also uses its name as a category level
  drops that level on its first import and makes the category on the second. Reading it from the
  planned state instead could flip between passes.
- **A row's entity takes its parent from** its parent cell; else the entity named like the deepest
  category level that exactly one entity carries, skipping the row's own name; else its root
  group, or a root-named level it dropped; else the URL tree. The category stays its own record:
  this places the entity in the graph only, as weakly as the URL tree, so an entity that already
  has a parent gets no second one. The category's entity comes before the root group because the
  Catalog's "BPC-157 10 mg vial" sits under Peptides › BPC-157, and the dropped Peptides would
  otherwise take it from BPC-157.
- **Categories are matched by parent and key** against the site's records and those an earlier
  sheet of the same request planned, and created parent first when missing, spelled as the first
  sheet that names them. The page takes the leaf; an empty chain leaves it where it was, because an
  empty cell erases nothing, and a different chain files it again. The apply writes the categories
  before the pages and, in the same transaction, deletes every category with no page and no term
  anywhere under it, deepest first, so a renamed level leaves no ghost; the preview says so first.
- **The export writes a page's chain** into Category, Subcategory and Sub Subcategory, as many
  columns as the deepest chain needs and no more than the three the detector reads back; a deeper
  chain keeps its first three levels and the export answers `category_chain_cut`.

### The whole workbook

- **A workbook is one preview and one apply.** A request may name its sheets, each with its own
  mapping; they are planned in the workbook's order, each on the site as the sheets before it would
  leave it, which `siteState.after` folds the way `write` persists a plan, scopes and categories
  included. One transaction writes every plan, the row budget is shared, and a mapping saved from a
  workbook is kept per sheet as "name / sheet". The one-sheet `mapping` stays. A scope that would
  clash is a blocking `scope_clash` in the preview, where it was a conflict at apply.
- **`Inspect` reads every sheet as its own**: the mapping detected from its own headers and its row
  type, products only when the site sells and every row with a path finds one product.
- **Every finding names its sheet**, as "Catalog · row 4", and one with no row names its sheet.
- **A mapping is resolved before it is used**: a saved id loads that mapping of the same site, and
  one with no columns, groups or indents is detected from the sheet's headers; an apply saves what
  it resolved. The agent's `imports_preview` and `imports_apply` now do what their descriptions
  promised, where they imported nothing without a mapping and never loaded a saved id, and take
  `sheets`.
- **The client workbook as one import equals its four sheets one by one**: entities, kinds,
  scopes, keywords, edges, pages, canonical pages, categories and the filing of every page; and a
  second import writes nothing.

### The run engine files a page

- **Publish files a page or a post under its whole chain**, in the write that carries its body; no
  step was added. A stored term is reused while it still sits under the wanted parent, the rest is
  ensured in WordPress (found under its parent by decoded name, else created, else, on
  `term_exists`, the term WordPress names), and every term is kept against its category. A create
  sends the chain and an update the union with what the item carries; `added` is what the item did
  not carry before, which is all a revert takes back.
- **A category never stops a page.** A refused permission is `categories_forbidden`, another refusal
  from WordPress `category_refused` in its own words, a page on a site without `page_categories`
  `page_categories_need_plugin`, an item that did not keep the list `categories_not_taken`; all are
  warnings and the report's note says the page's categories are not on the site. A term row that
  cannot be kept is an error, a 5xx is the step's retry, and a retried ensure creates nothing twice.
- **A product gains our product categories beside the client's.** The chain's `product_cat` terms
  are added to the categories the product carries, in the `wc/v3` save of its other fields, before
  the hash is read again and the description goes through the raw route; nothing is sent when the
  product already carries them, and the client's are never taken off. WooCommerce replaces the
  whole list, so the union is what is sent. This **supersedes** the 2026-10-03 rule that a
  product's categories are never written; its name, price, stock, SKU, status and slug still never
  are.
- **A revert takes back only what the run added.** A page or a post gets its current categories
  less the run's, an explicit empty list when none are left, and a product the same inside its one
  `wc/v3` restore. The terms the run created stay on the site and the revert names them,
  `revert_terms_kept`, as it keeps media: a term may have been reused since. A list it cannot put
  back is `revert_categories_kept`.
- **A sync adopts the terms already on the site.** Every category record is matched from the root
  down by its key under the parent term already matched, in `category` and, when the store is
  ready, in `product_cat`; a match is kept with no run, a mapping whose term is gone is dropped, and
  a taxonomy the site will not list is `categories_unread`, not a failed sync. It reads no entity,
  so an entity named like a term files nothing.
- **The estimate warns `page_categories_need_plugin`** for a WordPress page with a category on a
  site whose stored capabilities lack `page_categories`, at no request; the publish checks the live
  manifest.
- `publish_result.categories` keeps `{taxonomy, terms[{categoryId, name, termId, parentId,
  created}], previous, added, taken}`. A revert reads only what was added and created, so a result
  written under the flag model, which named an `entityId`, still reverts.

### The companion plugin 1.3.0

- **It files pages under categories and lists them on category archives.** It registers the core
  category taxonomy for pages on `init` after core does, so `/wp/v2/pages` takes and returns
  `categories`, and adds pages to the post types of a category archive's main query outside
  wp-admin, keeping what the query already names. The manifest advertises `page_categories`.
  Without it, posts and products are still filed, and a page goes up uncategorised and says so.
- **Accepted with it:** a client's category archives, feeds and sitemaps list pages once the plugin
  is updated, and the page editor gains a Categories box.

### What the screens and the services say

- **A page names the categories it is filed under**, root first, as `{id, name, termId?}`: in
  `category` for a page or a post, in `product_cat` for a product, none for a product category; a
  WordPress page on a site whose plugin cannot file it says so. An entity shows its canonical page's
  chain, else its only page's.
- **`PagesService.ListCategories` is unpaged, like `Tree`**: it answers every category of the site
  with its pages and terms, a number of rows bounded by what imports create. The page list's
  `categoryId` always keeps the whole branch, with no flag beside it.
- The Pages rail holds the category tree, each node with its pages and whether the site has its
  term yet; the pages, the entities, the runs, the revert, the site and the import preview name
  the categories in words.

### Decided without asking

- `Root` and `Root Entity` are the only root headers; `Root Category` and any other level header,
  `Brand` say, is a category level.
- A run over the client workbook's pages needs a template picked at start or a site default,
  because the workbook carries no page kind; on a site with WooCommerce, the Catalog's `/shop/`
  parent is WooCommerce's own page.
- Real WordPress gives a new post with no category "Uncategorized", which the union keeps beside
  ours and an emptied list brings back; the site's own category is never taken off.

## 2026-10-03 and 04 — the code rewritten many times, read again

The owner asked for the code that had been rewritten many times to be read again: the composition
root had 43 commits and a 235-line `build`, `sync_site.go` 861 lines. Behaviour was held still by
tests written before each move.

- **The composition root builds by area.** `core.go` keeps the core, `Open` and a `build` that calls
  `build_repos.go`, `build_llm.go`, `build_services.go`, `build_runtime.go` and `build_agent.go` in
  the order their dependencies fix: repositories, the model chain, authoring, the engine, the use
  cases, the agent, the scheduler.
- **The steps live by concern.** `steps.go` keeps the registry; the dependencies, the artifacts, the
  link policy (one, for the steps and the preflight), the writer's ceiling, the prompts and the one
  request every model step sends, the WordPress helpers and the page links each have a file;
  `sync_site` splits into pull, cursor, reconcile, archive and settle; the two relinks share one
  core; publish reads as named stages. A page's parent from its path comes from
  `pagemap.Index.PathParentID` in the sync and the pages service alike, and the import keeps store
  items out of that index, so a store item is never a page's parent.
- **The run engine reads by life cycle**: claim, execute, settle an item, settle a run, enqueue,
  control, requeue, regenerate, each in its own file, with a test pinning every path by the events
  and executions it records. `Engine.Wake`, called only by its own test, and `Engine.Recover`, which
  only renamed the sweep, are gone. `domain/run` keeps a file per concern with its const blocks
  whole, so `vocab.ts` renders byte for byte the same.
- **The import is a pipeline of named stages**, with one identity for a unit, by its entity once
  matched and by its chain of parents before.
- **The frontend shares what it had copied**: one query codec for a screen's address, the sort
  helpers, the failure readers, one clock, the page badges and cells. The reports Pages tab coloured
  an item's status by its page, so a failed item on a published page showed green; it reads the
  item now.
- **What nothing called is gone.** In Go: `wp.SEOFields`, the `wp` path wrappers `pagemap` owns,
  `wp.StoreForbidden`, `Brief.RequiredHeadings` and `PhraseTexts`, `dbx.Ok` and `Err`, the UUID,
  enum, float and bool sort keys, `ctx.WithRunID` and `WithConversationID` with the audit log
  fields that read them, `errors.Stack` and `Frame` with the stack every error captured,
  `middleware.Timeout`, the two `"openai"` provider constants beside `llm.ProviderOpenAI`,
  `RunRepo.Active`, `Graph.Roots`, `MessageRepo.LatestSeq` and `ByConversation`,
  `CategoryRepo.Get`, `ScanUpdatedAt`, `Store.Path`, `sqlitetest.OpenEncrypted`,
  `pagemap.Unmapped`, `InternalPath`, `Index.Len` and `Observed.Empty`, with `Index.PathParent`
  folded into `PathParentID`, and exports one file used, `graph.NewAnchors` and `graph.Scopes`
  among them, are unexported. In the frontend: three hooks with no
  caller and their endpoint wrappers, `runPhase`, `actionStatusTone`, `keywordTexts`, a second run
  status tone and a second copy of the page status words. This **supersedes** the Phase 0 note that
  `Stack` is nil-receiver safe. The errors the agent could hand to nobody go to `Deps.Dropped`,
  which the composition root logs.
- **Three keys, each the rule of whoever refuses a duplicate.** `graph.Key` trims and lowers
  ASCII letters only, which is what SQLite's `NOCASE` does to `entities.name` and
  `entity_anchors.text`; `TestTheGraphKeyRefusesWhatTheStoreRefuses` inserts each pair into a real
  store and shows the store refuses exactly the pairs the key folds together, so Café and CAFÉ under
  one parent are two entities in Go as they always were in the database. It replaces the name key,
  the anchor check and the name sort of `graph`, the fold of `application/graph` and the import's
  key, which lowered every letter and so merged in Go what the store keeps apart. A keyword is one by
  its own Unicode lowercase, found in its list by `keyword.List.Find`; a category by
  `category.Key`, WordPress's term-name rule, stored as `name_key`.
- **One helper per job**: `dto.TimeOf` for an instant that may be missing,
  `dto.PageSize` for a list's limit in place of `ListRequest.Normalize`, and `graph.Distinct`,
  which keeps a list of texts once as the store tells them apart, in place of a
  `CleanKeywords` that cleaned no keyword. The four `anchorsOf` say what each builds
  (`requestedAnchors`, `sheetAnchors`, `targetAnchors`, `placedAnchors`). Thirteen repository
  `List` methods repeated the keyset, the query, the select and the cut; `selectKeyed` does it once
  and each keeps only its filters, and the nullable-column helpers live together in `rows.go`.
- **A run or an item that waited before its first step still starts.** A run paused before it
  started and then resumed came back running, so its first claim never set `StartedAt` nor
  announced `run.started`, and the window showed a finished run with no start; an item held behind
  its parent before its first claim never announced `item.started`. A run now starts on its first
  real claim whatever its status, and an item when it reaches its first step with nothing recorded.
- **A repair leaves a post where its permalink puts it.** `repair_hierarchy` sent a post the parent
  its path implied, WordPress kept the post flat, and the item paused on a mismatch nobody could
  resolve. The preflight refuses a post with `post_unnested`, beside `store_placed`, and the step
  refuses one before it writes.
- **A matched entity keeps the anchors it carries.** An import rewrote every anchor of a matched
  entity as a user anchor of weight 1 whenever anything about it changed, so a re-import that added
  a keyword turned the agent's anchors into the user's and reset every weight. The sheet's anchors
  are unioned with the carried ones: those the entity holds stay exactly as they are, one
  `graph.Key` finds in none of them is added as a user anchor of weight 1, and `sameEntity`
  compares whole anchors, so an entity is skipped only when its anchors are equal in text, source
  and weight.
- The canvas under the graph map was left as it is, by the owner's choice.

## 2026-10-04 — no categories: the client's Category columns are WooCommerce's

The client saw the categories built that day and said that the Category, Subcategory and Sub
Subcategory columns of his sheets are WooCommerce's product categories, which he keeps himself in
the store. The owner removed categories from the whole application the same day, the Go and the
screens, as if they had never been added, and kept every other change of the same commits: the
OpenAI-only client, the spend panel, the whole workbook in one import, the rewrite and `graph.Key`.
None of it had been released.

### The owner's answers

- **The Category columns are ignored.** Category, Subcategory, Sub Subcategory and every other
  level column make no category and no entity group; only a `Root Entity` or `Root` column makes a
  group. That holds wherever a level column comes from: the detection proposes only the root
  headers, and a saved mapping or the agent that names Category keeps it stored and reads it
  ignored. `Mapping.Bind` and `Uses` keep only the root level columns and report every other one,
  `Root Category` and `Brand` among them, as `ignored`; a mapping maps something only through a
  field, an indent column or a root column (`Mapping.Unmapped`), so one that names only Category
  and Subcategory is refused when saved and detected from the sheet's headers when previewed; the
  wizard offers as group columns only the root headers detected for the sheet. The option keeps its
  name, `levelColumns`, because it is stored JSON and part of a tool's schema, and no stored mapping
  is migrated.
- **No product category is written again.** A product's name, price, stock, SKU, status, slug and
  categories are never sent, the 2026-10-03 rule, and a product revert touches no category.
- **A URL parent wins inside the row's Root group.** A row's entity takes its parent from its
  parent cell; else from its URL parent, when that parent lies inside the row's own Root group; else
  from the Root group; else from the URL tree. With Root Peptides, `/peptides/bpc-157/liquid/` puts
  Liquid under BPC-157, not under Peptides. One predicate, `inGroup`, says whether a parent lies
  inside the group: a planned unit does when it, or a unit above it, is the group or one of its
  rows, the walk ending at a parent a cell names; an entity on the site does when it, or an entity
  it is named under, carries the group's name by `graph.Key`, and a planned unit whose parent is on
  the site walks on there. A URL parent that is the group's own page gives the group, as before,
  and one in another Root group gives the row's own group. The URL parent is kept as weakly as the
  URL tree, so a matched entity that has a parent keeps it. A parent cell is resolved by the same
  predicate, so one that names a namesake deeper in the group is found instead of being
  `ambiguous_parent`.
- **What it does to the client workbook.** The Groups sheet keeps its eight entities and seven
  edges, because every URL parent in it lies inside Peptides, so BPC-157's Liquid and Powder stay
  under BPC-157 and TB-500's Liquid and Capsules under TB-500. The Catalog, which names no root, puts
  its four products under the generated Shop by the URL tree, where its Category column had put
  three of them under BPC-157 and TB-500. The workbook as one import still makes sixteen entities,
  equal to its sheets one by one.

### Decided without asking

- **A forward migration undoes the records.** 0044 drops the index `pages_category`, then
  `pages.category_id`, then `category_terms` with its index and `categories` with its two; its down
  recreates them with the exact text and index names of 0039–0041, the column last. 0039–0043 are
  not edited, as 0042 and 0043 undid the flag model's 0033 and 0034 without editing them. Nothing is
  converted: a development database loses its category records and its filings, which were never
  released.
- **The companion plugin returns to 1.2.0**, byte for byte its content at `v2.3.0`: 1.3.0 and its
  `page_categories` was the only plugin change since, and it never shipped. `wp-plugin/openapi.yaml`
  is the 1.2.0 contract again.
- **`wp/category.go` is not restored**: its `ListCategories` and `CreateCategory` had no production
  caller even before the category work. The terms client, `wp.SameTermName` and the term id a
  refusal named go with the rest of the adapter's category code.
- **What the range added for categories and other code now uses stays**: `wp.Forbidden`, which
  three other callers read; `pages.Deps`, without its two category fields; the run screens' revert
  and sync findings lists, which show every finding, `revert_meta_kept` among them; the workbook's
  per-sheet `after` fold, which keeps the scopes, and `scope_clash`.
- **Not this feature, and kept**: the graph's entity kind `category` and its synonyms, the Category
  template, a product category page (`product_cat` as a page type, `product_category_unwritable`)
  and every guard that refuses to write one.
- **What an earlier run left is harmless.** A run artifact written with `publish_result.categories`
  decodes with the field ignored, and its revert touches no category. A site whose stored
  capabilities still name `page_categories` keeps the word until its next sync, and nothing reads
  it.

### What it supersedes

- **The whole 2026-10-04 categories section** as it bears on categories: the category records and
  `category.Key`, the category chains and the root rule of the import, the parent taken from the
  deepest category level, the export's category columns and `category_chain_cut`, the filing on
  publish, the product categories, the revert's terms, the sync's adoption,
  `page_categories_need_plugin`, `publish_result.categories`, the plugin 1.3.0 and what the screens
  and the services said of categories, `PagesService.ListCategories` among them. Its whole workbook
  stands, less the categories it named: the `after` fold keeps the scopes, and the workbook as one
  import equals its sheets one by one with no category or filing left to compare. Its line that
  only a root level makes a group stands, with every other level column ignored where it was a
  category level, and its undoing of the flag model by 0042 and 0043 is the precedent 0044 follows.
- **The 2026-10-04 line that a product gains our product categories beside the client's**, which
  had superseded the 2026-10-03 rule: the 2026-10-03 rule holds again.
- **The 2026-10-02 line that the level columns build a tree of groups**: only the Root levels do,
  as they had since 2026-10-04, and the others are ignored.
- **The parent precedence** of 2026-10-02, "the parent cell, else the row's deepest group, else the
  URL tree", and its 2026-10-04 form through the deepest category level: the precedence is the one
  above.
- **The third of the rewrite's three keys**: `category.Key` goes with the records; `graph.Key` and
  a keyword's own lowercase remain.

## 2026-10-12 — the companion plugin runs on PHP 7.4

**Amended on 2026-10-24** (phase 15, session 1): the plugin targets PHP 7.2 and WordPress 5.7, its
lint and tests run on 7.2, 7.4 and 8.5, and a function scan proves no newer function is called.

The client's host runs PHP 7.4.33, and WordPress refused to install the plugin because its header
asked for 8.1. The sources used nothing newer than PHP 7.1 (scalar and nullable types, `void`,
namespaced array constants, `random_bytes`), so the 8.1 was a declared floor and not a used one,
and only the header moves: `Requires PHP: 7.4`. WordPress ≥ 6.4 stays.

- **The plugin stays 1.2.0.** The contract in `wp-plugin/openapi.yaml`, the manifest, the
  capabilities and the behaviour are unchanged; only the install requirement drops, and no site
  carries a 1.2.0 with the other header, since the client's install was refused. The sources are
  those of `v2.3.0` but for that one header line.
- **The lint runs on the floor.** `task plugin:lint` runs `php -l` on PHP 7.4, the compose
  `phplint` service on `php:7.4.33-cli` and the CI plugin job on `setup-php` 7.4, so the plugin
  takes no syntax newer than 7.4. Nothing in `wp-plugin` calls a function newer than 7.4 either,
  which `php -l` cannot see: `str_contains`, `str_starts_with`, `str_ends_with`,
  `array_is_list`, `match`, `?->`, named arguments, union or `mixed` types, enums and `readonly`
  stay out.
- **The released app hands out the old header.** v2.4.0 embeds the plugin through
  `wp-plugin/embed.go`, so its Sites panel serves the 8.1 zip until the next release; the client
  got the 7.4 zip by hand.

## 2026-10-17 — the site model: one structured page, four providers, edited through the model

The client's task changed after the application was built. He runs dozens of WordPress and
WooCommerce sites of up to a few thousand pages each, built with his own tools (a WooCommerce CSV
import of variable products, Code Snippets, the theme); the pages exist, and the work is editing
and improving them, never writing them from scratch. He keeps the entity plan in xlsx workbooks
whose rows do not always carry a URL. He wants one place where the site, the workbooks and the
program collapse into one model, an AI writer that works with the full picture (categories,
breadcrumbs, the products a page lists, parent, children, siblings, keywords, anchors, notes),
internal linking as the core of the SEO work, and the in-app agent doing most of it. We know one
of his sites in detail (Blocksy, the classic editor, Rank Math, breadcrumbs and a product grid
injected into the content at render, flat URLs under category breadcrumbs); it is a data point,
never a specification. The owner approved the decision record below on 2026-10-17, and stage 0
(the audit, this section, the design in part 16 of the spec and the phase 14 plan) wrote no code.

### The decision record

- **Every page is a structured object.** Any page of a site, of any post type, is one page model:
  identity (URL, type, WordPress id, slug, status, the terms of every taxonomy it carries), fields,
  a body made of addressable blocks, the regions of its render, typed links, the schema it
  publishes, a page type, and a place in the hierarchy with the reason for each derived value.
- **Four providers fill the model**: WordPress core REST, WooCommerce REST, the companion plugin
  and the public rendered page (HTML and JSON-LD). A field registry says which provider reads and
  which writes each field. The writer, the agent and the UI work with the model and never know
  which provider a value came from. Interfaces are declared by their consumers, as always.
- **Adaptive, never site-specific.** No theme names, CSS classes or site-specific selectors in Go
  or PHP. Structure is derived from the difference between raw and render, from standards first
  (schema.org JSON-LD: BreadcrumbList, ItemList, CollectionPage, Product, WebPage; ARIA landmarks
  and labels, a `nav` whose `aria-label` contains "breadcrumb"; `main`, `article`, `h1`) and from
  REST data. Whatever cannot be derived is a per-site setting the agent can change, with a default,
  never a constant.
- **Raw versus render.** What is in the raw `post_content` is editable. What appears in the render
  and not in the raw was generated by the site (snippets, shortcodes, the theme): read-only, but
  read and understood. The content root in the render is found by locating the content's text in
  it, not by a selector. Shortcodes and unrecognised elements in the raw are atomic foreign blocks,
  preserved byte for byte.
- **Local storage of content is allowed.** We store the raw `post_content`, current and base; the
  field values, current and base; and the extraction from the render (regions, typed links, JSON-LD
  facts, the breadcrumb trail, headings, word counts). "Base" is the last version both sides agreed
  on, or the last version we wrote. We never store the rendered HTML. Designed for a few thousand
  pages per site.
- **The body is edited as blocks**: replace a block, insert after a block, delete a block, rewrite
  a range, place a link inside a block. Untouched blocks keep their exact bytes. A write goes
  against the hash of the content we read. The parser handles classic-editor content (paragraphs
  separated by blank lines or block elements, wpautop at render) and block-editor content (block
  comment delimiters). We remember which blocks we wrote by their hashes. There are no markers of
  any kind in the client's content.
- **Fields are a catalog plus a per-site scope the client configures.** The built-in catalog:
  title, excerpt, body, SEO meta title and description, product description, product short
  description, variation descriptions, image alt and title, upsells and cross-sells, and any post
  meta or WooCommerce `meta_data` field by key. Per page type the client enables fields and sets,
  for each, whether the AI writes it, an instruction (how and why, which reaches the writer as that
  field's rule), length limits and the link policy. Theme and plugin settings are out of scope:
  they are settings, not pages.
- **The hard-deny list**, never written even if a scope enables it: slug, regular and sale price,
  stock fields, SKU, post status, product categories and any taxonomy assignment, the WordPress
  parent, and the attributes a variable product's variations are built on.
- **Links are typed and parametrised**: breadcrumb, listing, navigation, contextual. Every observed
  link has a type, the region it sits in and its target page. Link policies attach to the pair
  page type + field, including "no links". Graph compliance stays the rule for the contextual links
  we write. The audit counts every type but owes and places contextual links only. Our links follow
  the URL form the site already uses, absolute or relative by majority.
- **Hierarchy is logical and read-only.** The parent comes from the BreadcrumbList JSON-LD, then
  the ARIA breadcrumb `nav`, then the WordPress parent, then the URL. Nothing ever writes a
  WordPress parent or a slug to an existing page: on the client's sites URLs are flat while the
  breadcrumbs follow categories, so writing a parent would change URLs.
- **Page types are named by the client**, seeded from built-in presets (Hub, Category, Catalog,
  Product, Entity, Technical); he renames them and adds his own, each on a preset's behaviour. A type
  is a bundle of settings (template, enabled fields, link policies, role in the network) and
  replaces the fixed entity `kind`. The client did not accept "topic" as the name for an entity,
  which is why the names are his.
- **Technical pages are detected with a reason that can be overridden**: the WordPress front,
  posts and privacy pages; the WooCommerce shop, cart, checkout, account and terms pages; cart,
  checkout and account shortcodes or blocks; `noindex` in the SEO meta. They are never link targets
  and never generated.
- **The writer's context pack is the spine.** For a page and a field it carries the page's type and
  identity; an entity card (definition and key facts from workbook notes, WooCommerce data, user
  notes, or proposed by the agent and confirmed); the breadcrumb path with a one-line summary of
  each ancestor; children; siblings with what each covers; the products the page lists with their
  facts; taxonomy terms; keywords with volumes and anchors; every link target with a one-line
  summary; the field's current value and the page's other fields; a summary of what the
  site-generated regions already show; the workbook notes, the field's instruction and its link
  policy. It is budgeted and prioritised to fit the writer's window.
- **Findings, never automatic resolution.** Discrepancies between site, workbook and model are
  persistent findings with resolution options, resolved by the agent or a person through one use
  case. A workbook row finds its page by URL, then slug, then entity name, title or H1, then SKU
  for products.
- **Zero footprint on client sites.** Nothing on a client's site may name or identify this program:
  the REST namespace; the plugin's name, slug, folder, text domain and header; option names, meta
  keys, transients and cron hooks; classes, attributes and comments in content we write; media file
  names, titles and alt texts; the HTTP User-Agent; the application password's name. A test proves
  it.
- **The companion plugin is strengthened and renamed** to a neutral name the owner approves. It
  gains generic read and write of the meta keys a field scope names (capability-checked, never a
  protected core key, never the hard-deny list) and a listing of every taxonomy term a post carries.
  PHP 7.4 stays. Its version moves. Its uninstall removes everything it stored. The Sites panel
  detects the old plugin and says to replace it.
- **Read-only boundaries.** Page builders are detected and their pages marked read-only; the
  site-generated regions, the taxonomies, the categories and the store-owned values are read only.
- **Scope discipline.** This extends the application. `content.Document`, `wp.Client.Visit`, the
  raw route and its hash check, the WooCommerce client, the SEO meta route, `content.InsertLinks`
  and `relink_page`, `pagemap.Site`, the run engine with its artifacts and revert, and the import
  pipeline are reused; bad code is rewritten, nothing is rewritten for its own sake.

### What the audit found that the design must undo

- Every sync is a full walk, no caller sets `since`; the plugin's `parent` is dropped in the pull
  and no code stores the WordPress parent; `pages.parent_page_id` has one writer,
  `pagemap.Index.PathParentID`, so the hierarchy is the URL tree and nothing else.
- The render is fetched once, to check that a product's description shows, and nothing from it is
  kept; links are extracted from the raw `post_content` by the plugin and recorded with no region
  and no type; a link we placed is re-recorded as `observed` on read-back, so origin cannot tell
  ours apart.
- `publish` sends slug, status and parent on create and update, `repair_hierarchy` exists to send
  parent and slug, and product publish and revert send the full attribute list with the `variation`
  flag.
- The writer sees the planned title and H1, the entity's name, kind and intent, the keywords, the
  page's notes, the template's sections and an anchor with a URL per owed target. It sees no page
  type, no entity definition, no ancestor chain, no sibling coverage, no listed products, no
  taxonomy term, no current value, no per-field instruction or policy, no generated-region summary,
  and no input budget exists anywhere.
- The entity `kind` reaches 29 Go, SQL and template files and 30 frontend files, beside a second
  free-text `Template.PageKind`; the link plan never branches on either.
- A workbook row finds its page by path only, no SKU is stored anywhere, and import findings are
  transient. 93 tools fit their ceiling by 17 bytes, the templates trio being a quarter of it.
- Every request to a site, the anonymous visits included, carries `User-Agent: Postulator/2`; the
  REST namespace, the plugin's folder and zip, its `_postulator_*` meta and its query var name the
  program. The media names and the HTML we write are clean, and no theme name or selector exists
  in Go or PHP. A custom post type or taxonomy cannot be observed: the type sets are closed.

### How the design settles what the record and the code disagree on

- **Creating a planned page** needs a status, and on a site with nested URLs a parent, once; the
  deny list governs existing pages, and whether the one creation call may set status, parent and
  slug is the owner's first open question.
- **`repair_hierarchy` has no job** once nothing writes a parent: the step goes, the `repair` kind
  loses its recipe (`relink` is the links-only repair), and the `CHECK` on `runs.kind` keeps the
  word so old rows decode.
- **The type belongs to the page**: `pages.page_type_id`; an entity reads its canonical page's
  type; the import's `entity_kind` cell maps to a type by name or synonym.
- **A product category is a page of the term family**, read, never written, and may be a parent
  (the client's trails follow categories), pending the owner's answer.
- **The links we write are site paths today**; the extraction records the site's majority form and
  a per-site setting decides which form a placed link takes.
- **The replica stand's mu-plugin** must carry the client's class names to be a replica; it is a
  fixture under `docker/e2e`, and the site-specific scan covers `internal/` and `wp-plugin/`.

### Decided without asking

- The design is part 16 of the binding spec and the plan is
  `docs/superpowers/plans/2026-10-17-phase-14-site-model.md`: section 10 allows no new spec file.
- `sync_site` keeps its name and the `sync` kind; the observation runs inside it as stages. A
  first observation and an observation a person asks for walk the whole site; later ones are
  incremental through the plugin's `since` with a full walk every Nth observation (a setting);
  core without the plugin always walks fully, because `modified_after` filters on the site-local
  column.
- The extraction has three layers, not two: the raw, the `the_content`-filtered HTML the plugin
  already computes for the H1 and will answer on its own route, and the public render. The content
  root is the smallest element of the render holding the filtered content's word sequence, which
  finds the root when the raw is empty, as on the client's hubs.
- The writer steps keep their names and artifacts; a new template step `generate_fields` answers
  the other enabled text fields; `publish` writes every produced field through the registry under a
  per-field checkpoint, so a crash between fields resumes without a second model call.
- Link rules move into the field scopes in stage 2 and the `link_policies` table, the `policies_*`
  tools and the Policies screen go then; `templates_set_override` stays.
- The old `pages` columns (`drift`, `content_hash`, `h1`, `title`, `meta_*`, `canonical`, `wp_*`)
  stay read-through until stage 3 rewrites publish, relink and revert, and a stage-3 migration
  drops them with a down that restores them.
- The proposed plugin name is **Content Bridge** (`content-bridge`, `content-bridge/v1`,
  `ContentBridge`, `_cbr_*` meta), version 2.0.0, with an `Update URI` header so wordpress.org
  never offers a plugin of the same slug as an update; its activation copies the old plugin's term
  clock and SEO keys and deactivates it. The owner approves the name.
- Two stands from one compose file, each with its own env file: vanilla on 8088 (the default block
  theme, WooCommerce's breadcrumbs, nested page URLs) and a replica on 8087 (Classic Editor,
  Blocksy, Rank Math with its breadcrumb schema on, a mu-plugin with the snippets' behaviour,
  variable products from a CSV, flat URLs). The suites assert through a per-stand table of which
  page type carries which region.

### What it supersedes or amends

- **2026-10-03**: "never writing what the store owns" becomes the hard-deny list plus the product
  type's field scope; the product matcher's address, slug, name becomes every row's URL, slug,
  entity name, title or H1, then SKU.
- **2026-10-04**: no category is written and none is a record of its own, still; a page's taxonomy
  terms are read as facts of its identity and its hierarchy, and a product category stays a page
  that is read, never written.
- **2026-10-12**: the plugin's version moves; PHP 7.4 stays.
- **VISION's storage boundary** ("WordPress is the source of truth; locally we keep metadata and
  hashes"): the raw body and the field values are stored, current and base; the rendered HTML never.
- **The product guardrail**: Postulator builds one structured model of a client's site and edits
  the site's content through it.

### Open with the owner

Whether a planned page's one creation call may set status, parent and slug; that the type lives on
the page and an entity reads its canonical page's (topic and custom map to Entity); the plugin's
name and meta prefix; whether a product category term may be a parent and the replica fixture may
carry the client's class names; whether the writer may rewrite the client's own paragraphs (a
per-type-and-field switch, off by default, is proposed); the User-Agent (the plugin's neutral name
and version is proposed); and that the owner's sandbox on 8089 stays bare.

## 2026-10-17 — the owner's answers to stage 0's questions

The owner answered the seven questions stage 0 left open, and two more the stage 1a plan put to
the owner the same day. Two answers change the record above: the companion plugin keeps its name,
and the zero-footprint ruling shrinks to what a visitor receives.

### The answers

1. **Creating a planned page may set status, parent and slug, once.** Only the creation call
   sends them, and the publish artifact records what it sent; an update never does. The client's
   pages almost all exist already, empty of content, and the workbooks carry no slug or URL for
   them, so creation is the rare path and editing an existing page the main one.
2. **The page type lives on the page.** An entity reads its canonical page's type; `topic` and
   `custom` map to Entity.
3. **The companion plugin keeps its name.** No rename and no Content Bridge: the folder
   `postulator-companion`, the namespace `postulator/v1`, the zip name, the text domain and the
   meta and option prefixes all stay. Having our plugin installed on a client's site is fine; the
   footprint ruling was only ever about traces in what a visitor receives.
4. **A product category term page may be a parent, and the replica fixture may carry the
   client's class names.**
5. **The writer does not rewrite the client's own paragraphs by default.** A switch per page type
   and field, off by default; a run may switch it on for chosen pages, with the user's
   confirmation.
6. **The User-Agent is neutral and has a per-site override.** The default names neither the
   program nor the plugin. The override exists because CDNs and security plugins block unknown
   agents, and reading the render must not die on those sites.
7. **The owner's sandbox on 8089 stays the owner's.** It is never wiped or provisioned
   implicitly; explicit tasks provision it as either stand, replica or vanilla, so patterns can
   be tested by hand. The e2e suites still refuse 8089.
8. **No activation hook** (asked at the 1a plan). Its one job was the rename's migration of the
   old keys, and WordPress never fires an activation hook when a zip is uploaded over an
   installed plugin, which is how the client updates; a hook with nothing to do would be a stub.
9. **Uninstall removes the plugin's bookkeeping and keeps the SEO values** (asked at the 1a
   plan). The term clock and the preview tokens and expiries go; the `_postulator_seo_*` title,
   description, canonical and OG values a site without an SEO plugin keeps through the plugin
   stay, being the client's content, so a client who deletes the plugin and uploads it again, as
   people do instead of uploading over it, loses nothing.

### The zero-footprint ruling, narrowed

Nothing that reaches a visitor may name or identify the program:

- the HTML of any public page;
- content we write: classes, attributes, `data-*`, comments;
- media file names, titles, alts and captions;
- anything the plugin emits on the front end: markup, `wp_head` output, HTTP headers, generator
  tags, publicly reachable query-var behaviour;
- the HTTP User-Agent of our requests.

The REST namespace, the plugin's name and admin UI, protected meta keys and options are allowed.
The footprint tests cover exactly this scope.

### What it amends in the 2026-10-17 record

- **Decision 15, zero footprint**: narrowed to the list above. The REST namespace, the plugin's
  name, slug, folder, text domain and header, option names, protected meta keys, transients, cron
  hooks and the application password's name leave its scope. The preview query var
  `postulator_preview` stays inside it, a visitor being able to reach it, and becomes neutral.
- **Decision 16, the plugin strengthened and renamed**: strengthened, not renamed. Its version
  moves to 1.4.0; 1.3.0, which named the categories build that never shipped, is not reused. Its
  uninstall removes what answer 9 says, not everything it stored. The Sites panel finds an
  outdated plugin by the manifest's version and capabilities, not by its namespace, and says to
  upload the new zip over the old one, which WordPress offers as "Replace current with
  uploaded".
- **The rename under "Decided without asking"**: the Content Bridge proposal (`content-bridge`,
  `content-bridge/v1`, `ContentBridge`, `_cbr_*`, 2.0.0) and the activation hook that copied the
  old keys and deactivated the old plugin are withdrawn. The `Update URI` header stays, so
  wordpress.org never offers a plugin of the same slug as an update.
- **The old plugin by namespace**: part 16.9's "the Go client recognises `postulator/v1` as the
  old plugin" is replaced by the version and capability check, and `TestTheOldPluginIsRecognised`
  by a version check test.
- **How the design settles what the record and the code disagree on**: the creation of a planned
  page and a product category as a parent are settled by answers 1 and 4.
- **Open with the owner**: closed.

## 2026-10-17 — stage 1a: the plugin's reads, its client and the two stands

Stage 1a of the site model strengthened the companion plugin for reads, taught the client and the
fake every new read, and built the two docker stands the suites now run on. Nothing new is
written to a site.

### What landed

- **Plugin 1.4.0**. It adds `GET /content/{id}/rendered`, `GET /meta/{id}?keys=`,
  `GET /terms/{id}` and `GET /settings`. `/content` gains `excerpt`, `terms` and every public type
  and taxonomy. The manifest names `postTypes` and `taxonomies`. The preview query var is
  `preview_token`. `uninstall.php` removes the bookkeeping, and the plugin carries an
  `Update URI` header.
- **The client**. A neutral User-Agent with `wp.WithUserAgent`; `PostTypes`, routing by
  `rest_base`, `_embed` terms and `SiteSettings`; `GetRendered`, `GetMeta`, `GetTerms` and
  `PluginSettings`; `ListProducts`, `ListVariations` and `StoreSettings`. `wp.Product` carries the
  store's facts read-only, and `Capabilities.Outdated` names an old plugin.
- **The Sites panel** names an outdated plugin with the update instruction and hands out the zip.
- **The vanilla and replica stands** run under `STAND`, and the owner's sandbox is provisioned as
  either stand only by `task sandbox:provision`.

### Decided without asking

- **1.4.0, an additive contract.** `/content` still defaults to page, post, product and
  product_cat, so a released app that asks for nothing meets no new type. `types` takes any
  content post type and any public taxonomy the manifest names. 1.3.0 is never reused.
- **The User-Agent is a plain `Mozilla/5.0`.** `Mozilla/5.0 (compatible)` is the crawler token
  some firewalls rate-limit.
- **An outdated plugin is the probe's verdict, never stored.** It travels on `site.Extensions`
  and in `CheckPlugin`'s view, and 1b makes it the `plugin_outdated` finding.
- **`normalize.php` follows `Site.Resolve` on classification.** That covers the host with its
  port, the escaped and lowercased path, and refusing a dot segment, an invalid escape and a
  control character. A relative href keeps PHP's base, the page's parent, against the domain's
  site base: both differ from RFC 3986's document base, and 1b's render extraction resolves
  against the document URL. `wp-plugin/testdata/hrefs.json` pins both sides and the fake.
- **`/rendered` runs `the_content` inside a one-post main query, after `the_post()`.** A snippet
  guarded by `is_singular`, `in_the_loop` or `is_main_query` then injects as it does for a
  visitor, which a plain `apply_filters` in a REST request would miss.
- **`/meta` refuses the whole read with `meta_key_denied`** when `is_protected_meta` protects a key,
  and takes at most 50 keys. **`/terms`** covers every taxonomy of the post's type, the internal
  ones included.
- **A refused read of `/wp/v2/settings`** (it needs `manage_options`) or of `wc/v3/settings` (it
  needs `manage_woocommerce`) is a `Forbidden` error the caller reads as unknown. Products,
  variations and media keep their own methods, though `/wp/v2/types` lists them.
- **The post routes serve content posts only**: a post of a public post type other than an
  attachment. The replica's builder found that a reusable block, a navigation menu, a template or
  a variation was answered by id, and that a raw PUT wrote into the navigation post WordPress
  creates. They now answer 404.
- **A preview filters Rank Math's and Yoast's robots output too.** Rank Math replaces core's
  `wp_robots`, so a preview under it printed `index, follow` until the replica proved otherwise.
- **The stands start term ids at 100000**, so a term id is never a post id and the term-id
  refusals mean what they say.

### Deviations from the request

- **No activation hook**, by the owner's answer at the plan; uninstall keeps the SEO values, by
  the same.
- **The old-plugin test is a unit test against the fake**, not an e2e one: no stand carries an
  old plugin to recognise. It is `TestAnOutdatedPluginIsNamedByVersionAndCapabilities`.
- **One agent built the stands; I wrote the store reads myself.** A second agent writing in the
  same Go packages would have broken the other's commits through lefthook's lint.
- **1.4.0 landed in two commits.** The outdated mechanism came first with the floor at 1.2.0, then
  the version, so no commit claimed a version the plugin did not have.
- **The site-specific scan reads its needles from the replica fixture**, so the test names no
  class, and Blocksy's `ct-container` is covered by the theme's slug. Its CSS-selector rule waits
  for `domain/render` in 1b.

## 2026-10-18 — stage 1b: the page model, its storage and the observation of the site

Stage 1b made every page of every public type exist in the model and taught `sync_site` to
observe a site into it. Nothing new is written to a site; `publish` stops sending slug and parent
to an existing page; the `repair` kind is retired. The thirteen decisions taken at the plan are
recorded under "Decided at the plan" in `docs/superpowers/plans/2026-10-17-phase-14-site-model.md`
and are not repeated here; this section holds what was decided while building.

### What landed

- **The domain**: `block` (a body as blocks, classic or block editor, serialised byte for byte),
  `field` (the catalog with its deny list, instanced keys and a value's current and base),
  `finding` (the codes, severities, options and subject keys), `render` (the extraction of a
  public render), `site.Settings`, and in `pagemap` the identity, the terms, the typed links,
  `DeriveParent` with a reason per source, the three reasons on `Page`, `Site.ResolveAt` and an
  open `WPType`.
- **Ten migrations, 0045 to 0054**, and the repositories over them; `sites.settings` as JSON.
- **`application/observe`**: five stages, incremental through `since`, a full walk every tenth
  and on demand, renders four at a time, and the findings `parent_unresolved`,
  `plugin_outdated`, `field_changed_on_site`, `body_changed_on_site`, `render_unreachable`,
  `builder_detected`, `technical_detected`, `page_title_differs`, `page_h1_differs`,
  `parent_differs` and `url_form_mixed`. `Summary` and `Page` are its reads; the composition
  root exposes them as `Core.Observe` for the harness and for 1c, with no Wails binding yet.
- **The steps**: `sync_site` drives the observation batch by batch through its checkpoint;
  `publish` sends title, content and status to an existing page; `publish_parent` is the
  creation-only lookup; `repair_hierarchy` and the five `sync_*` helper steps are gone; every
  writer records the body and fields it wrote as the page's base.
- **The href alignment**: plugin 1.4.1 resolves a relative href against the document, as the
  domain and the fake do; `hrefs.json` pins all three.
- **The wiring 1a left**: `sites.Update` and the `sites_update` tool take the per-site settings
  (`userAgent`, `renderConcurrency`, `fullWalkEvery`, `linkUrlForm`); the registry builds every
  client, and the connection test of a saved site, with `wp.WithUserAgent(settings.userAgent)`
  and rebuilds a cached client when the agent changes; the site-specific scan's
  `TestTheExtractionNamesNoSelector` walks the string literals of `domain/render` and
  `application/observe` with `go/ast` and refuses a class, an id or a selector call.
- **The e2e proof**: `TestTheSiteIsObservedIntoTheModel` on both stands, and the full suites.

### Decided without asking

- **A block without text is dynamic.** An image, a rule or a cart block is `Dynamic`, so an
  image-only body reads `dynamic_only`, which is what "whose expansion is content" means.
- **A crumb at the site root is dropped**, as a technical page is, whatever the front page
  shows: on vanilla the front is the posts listing and "Home" would be an unresolved crumb on every
  term page. The URL source still names the front page as a flat page's parent for the tree.
- **`NewPage` keeps a non-empty slug** and derives one from the path only when it has none; the
  spec's "stops deriving" is read as "stops overwriting". `Mismatches` compares the slug only
  when the plan has one.
- **The import accepts a type the site's pages carry** plus the four built-ins, and keeps
  `unknown_wp_type` for the rest, since an open `WPType` would otherwise plan a page of a type
  no site registers. A custom type must be observed once before a sheet may name it.
- **`BuildTree` nests by `ParentPageID`** with the path ancestor as the fallback and breaks a
  cycle at the first path, so a site with a page at `/` still has one root.
- **`Site.Validate` validates the normalized settings**, so a zero `Settings` is valid and an
  unknown URL form or a concurrency above 16 is refused at the service.
- **The downs of 0045 and 0052 rebuild through a TEMP copy under the real name**, so the schema
  text equals 0044's; a rename would store a quoted name.
- **The extraction's rules, from the live renders** (agent R): the filtered top-level blocks are
  a second content-root fallback; a text that repeats takes the occurrence carrying the
  fragment's links; a JSON-LD breadcrumb names an element only when more than half its links are
  crumb paths, never the body, main or an ancestor of the root (Rank Math's two-crumb list names
  nothing); a listing widens while the parent's links are mostly item links and never overlaps
  the raw text; header and footer are navigation only outside article, aside, main, nav and
  section; the last text-only or `aria-current` crumb takes the document's path; a builder's
  region widens to the largest ancestor with the same words; a link's anchor is its accessible
  name. The URL form follows the theme (absolute on both stands) while the client's content links
  are relative, and the H1 sits outside the content root on both themes.
- **A render is fetched again when the page's `modified_gmt` or its body's `observed_at` is later
  than the render's `extracted_at`**, so the checkpoint carries no page ids; a title-only edit
  WordPress does not stamp keeps an old render until the next full walk. The archive and "seen
  this walk" read `page_identity.observed_at` against the walk's start for the same reason.
- **A denied field is always agreed**: the observation moves its base with its current, and only
  a writable field raises `field_changed_on_site`. An open finding re-raised with the same
  subject touches nothing; a render that failed has no row to keep and is tried again every
  observation; `parent_differs` is raised once and stands until a person resolves it.
- **`PageBodyRepo.Record` takes no run id**; the run is on `page_blocks_written`.
- **A parent at the site root is never a WordPress parent on a creation** (`steps.parentOf`,
  `runs.parentOf`), which is how the old code behaved; a planned page with a nested path and no
  parent row is still refused at the start, and the path check against the parent is gone.
- **A creation's address is checked after the retry too** (`writeRequest.created`), an update's
  never. **A full walk rebuilds the raw links from the stored body** even when its hash is
  unchanged, the links table following the body. **An unreadable sync checkpoint is `INTERNAL`**,
  not `INVALID`. **The `sync_site` tool takes its own args** (`full`, `renderRescan`), by the
  2026-09-22 ruling; the schema ceiling is 63,450 for 63,443 bytes.
- **A draft's href base is its publishable permalink** (agent R): `get_permalink` answers
  `/?page_id=N` for a draft, so `permalink_url` gives the base the observation's `Document`
  expects. `Variation.ManageStock` is a `StockManagement` (`yes`, `no`, `parent`, null).
- **The fake serves a public render for any GET outside `/wp-json`**, 404 otherwise; fault
  injection skips public pages and `WithStorefrontDown` answers 503; only `product_cat` terms are
  served, the front page and the preview link are not, and a draft is a 404.
- **`SyncSiteRequest.Full` and `RenderRescan` were coded ten lines before their test**, which
  followed and ran red first; recorded rather than hidden.
- **The settings tool's optional args are pointers** (agent W), so `"userAgent": ""` resets the
  agent to the default; a 0 for the concurrency or the cadence falls back to the default because
  `Normalized` reads 0 as unset, while a negative value or a concurrency above 16 is refused. The
  registry reads `Settings.UserAgent` raw, since `Normalized` would rebuild the builder lists on
  every call. The service trims the agent as it trims the name.
- **The fake's storefront changed what a sync test sees**: once every page had a public render,
  the render stage stored its navigation links beside the raw ones and four steps tests that
  counted every link of a page failed. They now count the raw source, which is what they were
  about; the e2e proof counts the render's links by type.

### What the whole-branch review found, and what was done about it

A fresh reviewer read the branch at its close and graded one finding Critical, four Important and
ten Minor. Every Critical and Important finding was fixed in its own commit with a test that failed
first; two Minors were re-graded Important by their effect on a person and fixed too; the rest are
recorded here as deferred.

- **One body hash.** Three functions wrote and compared the one `page_bodies.hash` column:
  `block.Hash` folds CRLF and trims, `wp.ContentHash` and the plugin's `content_hash` take the
  exact bytes, and `field.Hash("")` is empty. A product without a description and any body we
  wrote with CRLF therefore read as `body_changed_on_site` on the first walk. `block.BodyHash`
  is now the exact sha256, equal to `wp.ContentHash` by a test, and the hash of every body
  snapshot; `block.Hash` stays a block's identity in `page_blocks_written`. Two steps tests over
  the fake site pin both cases.
- **An archived page is not visited.** The render stage skipped nothing, so a page the client
  deleted was visited every walk, raised `render_unreachable` forever and was never cleared; and
  the finding's subject carried the transport's error text with the visit's nonce, so a dismissed
  one reopened every walk. The render stage now skips archived pages and the ones a full walk is
  about to archive, the derivation clears the finding on archive, and the subject is the page's
  path and the status.
- **A failed filtered render is unreachable.** A failing `GetRendered` used to extract without
  the filtered content, changing the root state and the regions silently; it now counts as
  unreachable and the old render row stays.
- **The site is read outside the write unit.** The content and store stages called the plugin
  and the store inside the SQLite transaction, holding the one writer for a batch's worth of
  round trips while the UI, the agent and the second worker waited and then failed busy. Each
  stage now reads its batch first and writes inside; the observe fakes count a read made inside a
  unit and the test wants zero. The read pass remembers a new page before its insert, so a raw
  link to a page later in the same batch keeps no target until the derivation resolves it, or the
  link's foreign key fails; the full loops on both stands found that and a steps test pins it.
- **A render batch reads its own bodies.** The site model loaded every body of the site, both
  sides, for every render batch; it now carries no bodies, the derivation lists them once, and a
  candidate page is read on its own.
- **`since` follows the site's clock.** It was the previous walk's start on this PC minus a
  second, compared by the plugin with the server's `post_modified_gmt`; a server behind this PC
  missed every edit made in the gap until the next full walk. The observation now records when a
  walk finished and the newest `modified` it saw, and `since` is that stamp minus the walk's
  duration and a second, which is at or before the walk's start in the server's own time. Items
  within one walk-duration of the newest stamp are re-pulled each walk, so a re-pulled item
  nothing changed on now writes nothing: a field moves only when its hash moves and the page row
  only when something but its stamps moved.
- **A crumb after the page's own is never its parent.** A theme that appends a "Reviews" crumb
  made it the parent; the trail is now cut at the page's own crumb, as "nearest" in spec 16.5
  means the nearest preceding one.
- **The whole suite found what the per-package runs never reached.** `TestEveryToolChoiceIsADomainVocabulary`
  had been red since the `repair` kind left the `runs_start` enum, and agent W's `linkUrlForm`
  enum had no vocabulary behind it. `run.Kind.Startable()` now derives `startableRunKinds` (the
  enum drops `revert` too, which the start refuses), `site.URLForm` is a named type behind
  `urlForms`, a derived vocabulary counts as a domain vocabulary in the tripwire, and the start
  drawer reads the generated list instead of filtering by hand.
- **The UI harness runs its fake site unpaced.** Its scenarios re-seed a core whose adoption
  sync now reads, visits and renders every page of the in-process fake through the real registry,
  paced by the client's default four requests a second, so a seed took close to a minute and the
  harness tests ran past their budget. `app.Config.WordPress` carries the options every WordPress
  client is built with after the settings', the seam the harness uses to lift the limit, as
  `Config.Provider` is the seam for the model. A real site keeps the limiter: a first walk costs
  three to four requests a page, so two thousand pages take about half an hour at the default and
  `wp.rateLimitPerSecond` (1 to 100) is the knob; the plan's "about four minutes" counted the
  renders alone.
- **Deferred, as minors:** the derivation can persist a parent cycle between two on-site pages
  whose trails name each other (`BuildTree` guards it; stage 2's trail builder must too); a page
  whose type stops being viewable is never archived; `technical_detected` keeps the old reason when
  the reason moves between two non-none values; `url_form_mixed` reads archived pages' renders;
  `observe.Service.Run` has only test callers; the concurrency test proves the bound, not the
  overlap; a slug change on the site onto a planned row's path fails the sync on the path's unique
  index (pre-existing; the spec wants a finding); the derivation writes the whole site row, so a
  settings edit in that window is lost. The reviewer's set-aside list — the deny list gating writes
  only in stage 3, `publish` without an expected hash, `parent_differs` standing until a person
  resolves it, the `repair` word kept, no Wails binding yet — stands as ruled above.

### Deviations from the request

- **`sync_reconcile`, `sync_settle` and `sync_archive` are not kept as steps.** The brief said
  `sync_reconcile` stops merging silently; the observation does that inside `sync_site`, where
  the five stages share one checkpoint, and the five helper steps are deleted. `sync_site` is the
  one `sync` step.
- **The hold of a planned child behind its parent is proven by the existing e2e test**, not
  duplicated inside the observation test; `task e2e:full` on both stands runs both.
- **The e2e proof found a bug of its own**: the store stage observed a product's excerpt as empty
  because the store's item lacked the short description, so a first observation raised
  `field_changed_on_site`; fixed with a unit assertion in `observe`.
- **A commit swept another writer's staged deletions.** Agent R's render commit took my staged
  `git rm` of the five sync steps, leaving the tree broken until the steps commit. From that
  moment every writer commits with `git commit -- <paths>` and stages nothing ahead of its own
  commit; `CONVENTIONS.md` says so.
- **`CheckPlugin`'s outdated verdict reaches the model as `plugin_outdated`** from the
  observation's inventory stage, as 1a left it; the Sites panel still shows the probe's view.
- **Nothing new on screen.** The model fills behind the Pages screen; 1c binds `Core.Observe`.

### What it amends in the record

- **Spec 16.1**: `ParentReason` gains `workbook`; `NewPage` keeps an observed slug.
- **Spec 16.5**: a breadcrumb nav is an `aria-label` or a microdata `BreadcrumbList`, with or
  without a list; several lists tie to the first; a crumb at the root is dropped; the
  `BreadcrumbList` item shapes.
- **Spec 16.6**: `noindex` comes from the render's robots meta; the shortcode list is a settings
  default; the root is never a WordPress parent.
- **Spec 16.7**: the store's facts are denied fields of `page_fields`; a variation is a field
  of its product.
- **Spec 16.8**: `render_unreachable`; how the observation raises, clears and leaves a finding.
- **CONTRACTS**: the `repair` kind owns nothing and cannot be started; `sync_site` takes `full`
  and `renderRescan`; three per-kind steps; the plugin floor is 1.4.1.
- **ARCHITECTURE**: fifteen steps; the observation's stages; the new domain packages; the href
  rule; plugin 1.4.1.
- **Open with the owner**: none. The code answered every question the brief raised.

## 2026-10-18 — stage 1c: what the owner's review of 1b found, closed; the screens

The owner's review of 1b found the replica's hubs under their URL parent, three
`parent_unresolved` findings, 7 of 19 replica pages (6 of 14 vanilla) `not_found`, 39 `unknown`
links on each stand, the front page called technical and a settings edit the derivation could
lose. The twelve decisions taken at the plan are under "Decided at the plan" in the 1c tasks of
`docs/superpowers/plans/2026-10-17-phase-14-site-model.md` and are not repeated here.

### The diagnosis

A per-page log in `TestTheSiteIsObservedIntoTheModel` read the replica before any change.

- **Both the fixture and the derivation were at fault.** The mu-plugin linked a hub's middle
  crumb to `home_url('/' . slug . '/')`, and `/healing-peptides/` answered 404, the term archive
  living at `/product-category/peptides/healing-peptides/`. `DeriveParent` tried the nearest crumb
  by exact path only, then the WordPress parent (none on a flat site), then the URL, which gave
  Home. `/peptides/` (Home › Peptides) fell to the URL as well, because a crumb at `/` was always
  dropped and nothing else remained.
- **The third `parent_unresolved` named "Uncategorized", not Healing Peptides**: `/hello-world/`'s
  Rank Math trail links the post category archive `/category/uncategorized/`, which the model never
  holds (post taxonomies are terms on posts, not pages). It stays, honestly: the link is live.
- **`not_found` came from the order of the root rule**: the state followed the body only when the
  root was located, term pages carry no body, and a cart block with nested text was not dynamic.
  `Visit` also followed redirects and dropped the final URL, so `/checkout/` was extracted from the
  cart's render.
- **The 39 unknown links** were self links (`#respond`, `?add-to-cart=`), `rel="tag"` and
  `rel="author"` links, WooCommerce's related-products block (outside the root, where the listing
  rule never looked) and the archives' product grids inside a fallback root.

### Decided while building

- **A dead link falls to its name, and only a dead one.** A crumb whose internal link answers an
  error status after redirects resolves by its name; a live link to a page the model lacks, or an
  external one, never does, because the replica's post "Uncategorized" would otherwise hang under
  the product category of the same name. The owner's brief said "no link or neither 1 nor 2
  resolved"; this narrows it on that evidence.
- **A redirect's landing must carry the crumb's name**, and may not be the root, the page or a
  descendant: WordPress's `redirect_guess_404_permalink` sends a dead slug to any page whose slug
  begins the same way, which would make a broken crumb silently a product.
- **A Home-only trail is weak.** With the root crumb now resolving to the front page, Rank Math's
  and Yoast's two-crumb `Home › Page` lists would have named the front page before a nav with a
  real middle crumb was read; the other trails come first, then the WordPress parent, then the
  front page. The same rule keeps a sub-term printed "Home › Term" under its parent term.
- **Every trail is tried in order**, the best JSON-LD list first, then the other lists, then the
  navs; an unresolved nearest crumb still never falls to the next crumb of its own trail.
- **The lookups live in the render stage, on the crumb.** `derive` re-derives every page on every
  walk; a network call there would repeat every walk, flip a parent on a transient error and sit
  between derive's read and its writes. A crumb's final path and status are stored in
  `page_renders.trail`, looked up once per batch, and a failed lookup leaves the page unreachable
  for the walk with its old row.
- **`wp.Client.Resolve` stops at a change of scheme too**: ports default by scheme, so `http` to
  `https` on the same host counts as off-site, and the crumb keeps no final path. A site whose
  crumbs cross schemes resolves them by their links once its base URL names the scheme it serves.
- **A redirected visit is judged by the paths requested and reached**, never by the page's path,
  so a preview link (`/?page_id=N`) and a scheme or trailing-slash hop are not redirects.
- **The method is a column, `pages.parent_method`**, beside the reason that keeps the trail's
  source; the owner's example named `trail_href`, `trail_redirect` and `trail_name` as reasons,
  which would have meant rebuilding `pages` and losing which breadcrumb won.
- **0057 recreates `page_renders`**, a cache, so the first walk after the upgrade renders every
  page again; a rebuild that kept rows would have kept the old `not_found` states until a full walk.
- **The front page's `front_page` word stays in the `technical_reason` CHECK** after 0056 clears its
  rows, as `repair` stays in the kinds.
- **The extraction's rules** (agent R): a rel tag, category or author link is navigation whenever
  it is not the content's own (outside the root, or inside it overlapping no content run and
  matching no raw link), since a product without a description has only an assumed root; the
  listing rule is one everywhere, at least three product links and two products, self links
  excluded; a block named in the site's technical lists, or a paragraph holding only one such
  shortcode, counts as dynamic; a rel region grows only while every page link under it is
  rel-typed; `banner` and `contentinfo` stay unconditional landmarks while `complementary`,
  `navigation` and `main` count in the sectioning check; a JSON-LD crumb's href is its first item
  URL that resolves on the site. `render.BodyState` is the one rule for the state a body gives.
- **The fake's `/content` listed trashed posts**, which the plugin never does: a seeded run that
  trashed a page made the harness's next incremental walk fail on `/terms/{id}`. That was the red
  `task ui:lint` of 1b's gate, not the observation.
- **The bindings sit on `PagesService`** through a second source, as the phase plan's 1c says, so
  the service count stays fifteen and two methods join (`Model`, `ModelSummary`).
- **The audit's orphan rule is one rule** for the audit and the overview: a page on the site, not
  archived, not technical, not a term page and not the front page, with no contextual link in.
- **The harness gives nested pages their ancestors as a breadcrumb**, which resolves to the parent
  the WordPress parent already gave, so the seeded hierarchy does not move and the Model tab has a
  trail to show.

### Deviations from the request

- **The trail's method is its own column and vocabulary (`parentMethods`)** rather than three new
  parent reasons; see above.
- **Name resolution is narrower** than the brief: no link, or a dead internal link.
- **No fields pane** in the drawer: the 1c list names none, and stage 2's field scopes are where
  fields become visible.

### What it amends in the record

- **Spec 16.1**: `parent_method`; `archive` and `final_path` on `page_renders`; a crumb keeps its
  href, final path and status. **16.4**: the root states in order with `archive`, and the
  not-located rule; the link rules. **16.5**: the resolution order, the weak Home trail, the front
  page as the root, the broken cycle; the front page is no technical page. **16.8**:
  `parent_ambiguous`, `parent_cycle`, `render_redirected`.
- **The 2026-10-17 record**: "technical pages … are never link targets and never generated" no
  longer covers the front page, which is the root every trail starts at and may be edited where a
  stage-2 field scope allows; the start's refusal of `/` stands until then.
- **CONTRACTS**: `PagesService.Model` and `ModelSummary`, a hundred and twenty-six methods.
  **ARCHITECTURE**: the render stage's lookups and landings; derive offline and narrowed.

### Open with the owner

- **Name twins.** On the client's sites a hub page and a WooCommerce category usually share a
  name ("Peptides" the page and "Peptides" the category). A crumb with no link, or a dead one,
  that names such a twin is `parent_ambiguous` under the brief's "only when exactly one candidate
  matches", and the page falls to its next source. A crumb with a live link is unaffected, and
  that is nearly every crumb the stands and the client's themes print. The recommendation is to
  keep the rule and let stage 4's inbox resolve the few ambiguities, unless the owner wants the
  page twin preferred over the term twin for a page's own crumb. **Answered at the 2a plan**:
  the page twin is preferred (below).

## 2026-10-18 — stage 2a: page types, the field catalog and scopes, the entity card

Stage 2 was split in two on 2026-10-18: 2a gives the client his own page types over six presets,
a field catalog with scopes per type, the link rules inside those scopes, the entity card and the
move from the entity `kind` to types; 2b gives the writer, the linker and the judge the context
pack. The fifteen decisions taken at the plan are under "Decided at the plan" in the 2a tasks of
`docs/superpowers/plans/2026-10-17-phase-14-site-model.md`; this section holds the owner's rulings
and what was decided while building.

### The owner's rulings

- **Name twins** (the owner's review of 1c). On the client's sites a hub page and a WooCommerce
  category often share a name ("Peptides"). When a crumb resolves by its name (no link, or a dead
  one) and the candidates are one regular page and term pages, the regular page wins: it is the
  page we write and link to, while the archive belongs to the store. The method is `name` and an
  informational `parent_name_twin` names the crumb, the page and the term twins. Only two or more
  regular pages make `parent_ambiguous`; with no regular page, one term page resolves and several
  are ambiguous. A product is a regular page. Spec 16.5 and 16.8 say so.
- **The observation types only an untyped page** (answered at the 2a plan). A typed page is never
  touched by the observation, whatever set its type; a regular page waits for its first render
  before Hub or Entity is decided, so a failed first render cannot pin it to Entity.
- **"Category" names one thing** (answered at the 2a plan). The read-only preset for WooCommerce
  term archives is **Store category** (`store_category`); its role is the archive's, read-only and
  with nothing writable. The client's own word "Category" is then free: a regular page whose sheet
  cell, entity kind or template kind says Category gets the hub role, and a `product_cat` page is a
  Store category whatever its cell says. Type names belong to the client, who may rename Hub to
  "Category"; the import matches a kind cell against the site's own type names first and the
  synonyms second, so his sheets keep his vocabulary. The client's E-001 row is the evidence: it
  calls `/peptides/`, a regular page with a product grid, "Category". Spec 16.6 says so.

### Decided while building

- **Custom fields are read by a sixth observation stage, `fields`**, after `derive`, not by the
  content stage: it needs the plugin's `meta` capability, asks `GET /meta/{id}` outside the unit of
  work, and on an incremental walk asks only the enabled keys a page has never read, on a full walk
  every enabled key. A key the post lacks is stored empty, so it is not asked again every walk.
- **The product description's links reach the hub and nothing beside it**: `siblingMinWeight` 1
  is the closest the rules come to "no siblings" (only an all-but-identical sibling qualifies),
  with an up depth of 1, no down links and three links at most.
- **The link audit reads each page's type**: it no longer skips a page without a template and no
  longer resolves a template per page, so a page that had no template is now owed links; its site
  policy summary went with the policies. A neighbour that is a store category is refused before
  the plan is built, and `neighbor_no_template` is gone.
- **A fact or a note with a blank half is refused** by the use case, not cleaned away, so a card
  edit never silently drops what was typed.
- **Page types publish the existing events** (`sites.changed` for the types and scopes,
  `pages.changed` when pages move between types); no event name was added.
- **The page type's template joined `ResolveForPage` in 2a.8**, not 2a.4, so the templates use
  case changed its constructor once (the policy store out, a type reader in).
- **No superseded copies of the built-ins were added**: 0062 strips `linkRules` from every stored
  spec in place, and a test proves an untouched built-in stored before 2a equals the new seed.
- **0062's down** restores a global Default policy (a v4 id, the former default rules), the
  `sites` table by its original text (0004's statement, then 0031's and 0054's `ADD COLUMN`,
  because a test compares the text after the downs) and the built-ins' former link rules by name;
  any other template takes the guide's.
- **0063 drops `entities.kind` with `DROP COLUMN`** (its CHECK is the column's own) and
  `templates.page_kind` with its index; its down restores `entities` by its exact text (0030's
  rebuilt statement under its quoted name, then 0061's columns) and the kind from the canonical
  page's preset, and re-adds `page_kind` at the end with the built-ins' kinds, else `guide`. Saved
  import mappings are rewritten in SQL both ways.
- **The writer's prompt names the page's type** (`Page type:`), snapshotted at claim as `sc.Type`
  beside `sc.Links`, so a type changed mid-run cannot split the brief; the entity's kind line and
  the proposals' kind rules are gone.
- **The harness's own link policy** ("Crema Bench linking", rotating anchors) went with the
  policies; its rules return as a scope edit when the harness seeds types (2a.13). The harness's
  entity kinds went with `graph.Kind`; the six roots and Portafilter were hubs and the four
  accessories products, which is how 2a.13 types their pages.
- **`vocab.ts` was stale since 2a.6** (the `no_template` skip reason); 2a.8 regenerated it.
- **The import's type cell** (agent I): an entity-level header beats a page-kind header whatever
  the column order, and between two of the same strength the first wins; the preview raises
  `page_type_column_dropped` for any header that names a type but is left unread, so the client's
  workbook with both "Entity Level" and "Entity Type" gets one. An empty cell sets no type and
  keeps an existing one; a page typed by `user` or `ai` keeps its type with `page_type_kept`, and
  any other (untyped, auto, an earlier import) takes the sheet's, counting as an update. The
  export's column is `page_type`, as every export header is its field's name, so a re-import
  detects it. The import no longer sets a page's template; its Entities sheet ignores "Page
  Template".
- **2a.9 landed as one commit**, not three: 0063 drops both columns and the import read both.
- **The scope tool is `pagetypes_set_scope`**, not the plan's `fields_set_scope`: one `pagetypes`
  group holds the four tools, so loading on demand names one namespace, not two. It changes only
  the fields it is given, over the scope the type has (a stored one, else the preset's default),
  because a full put would make the agent resend a whole link policy to change an instruction.
- **`pagetypes_save` creates and changes**: without an id it creates from a preset, with one it
  renames or sets the template; the preset is fixed after creation.
- **`pages_update` takes its own arguments**, the update request plus `pageTypeId`; a type given
  alone only types the page, recorded as the agent's (`manual`, `ai`).
- **The link policy's field descriptions were shortened** to the old template tool's wording, as
  every tool resends them; the four tools cost 3,953 bytes, and the schemas stand at 61,437 bytes
  for 91 tools against 63,932 for 93 at the stage's start. The scope tool's `field` is the
  scopable keys' enum, and a custom field comes as `metaKey`, which no enum can list.

### Its tail, landed as stage 2b's Part 0 (2026-10-19)

2a.11 to 2a.15 had not landed when 2b opened: `TemplatesService` bound no type or scope, and the
frontend still called the six policy methods Go no longer had, so `task build` failed. The owner
moved them into 2b as its Part 0, the heavy gate legs run once at 2b's close.

- **The bindings** (`c566586`): `TemplatesService` takes the page-type use case as a second
  source, as `PagesService` does the model, and binds `ListPageTypes`, `CreatePageType`,
  `SavePageType`, `DeletePageType` and `SetScope`; `PagesService.SetType` is a person's choice; the
  graph's page states name each mapped page's type. The six policy methods out and six methods in
  leave the count at 126.
- **The screens** (agent F, `ffb6d9f` to `814c8a1`): Settings → Page types at
  `/settings/types?site=`, a scope drawer per type with the link policy editor that the Policies
  screen had; the drawer's type panel; the Graph inspector's card editor; the canonical page's
  preset colours and filters the graph where the entity kind did, an entity without a typed page
  showing "No page type", and an orphan being a root entity whose canonical page is not a Hub.
  The import field is `page_type`. A page-type reference in an agent card resolves through the
  site's `ListPageTypes`, there being no binding for one type by id.
- **The harness** (agent H, `0bc4a98`) types its seven root hubs and Portafilter Hub and its four
  accessories Product by hand, as the client would, because the fake site shows no listing and the
  observation would type them Entity; the Hub body scope carries the former harness rules and a
  `meta:hero_subtitle` custom field is read on the first sync. The 2a.13 text said six roots; the
  harness data has seven.
- **The e2e proof** (agent H, `3242f6a`), `TestThePageTypesFollowTheSite`, compiles, vets and lints
  under the `e2e` tag; it runs at 2b's gate with the stands, Docker having been off when it was
  written.

## 2026-10-19 — stage 2b: one read layer, the writer's pack and the agent's view

The owner's brief: one layer of read functions over the site model, shared by two views, the
writer's budgeted pack per page and field and the agent's brief and query tools, neither computing
anything differently. The sixteen decisions taken at the plan, with the Opus critic's review, are
under "Decided at the plan" in the 2b tasks of
`docs/superpowers/plans/2026-10-17-phase-14-site-model.md`; this section holds the owner's answers
and what was decided while building.

### The owner's answers at the plan

- **2a's tail lands inside 2b** as its Part 0, the heavy gate run once at 2b's close (above).
- **`generate_fields` asks only for the fields `publish` writes today**: the excerpt, where the post
  type supports one. Every preset has the excerpt off, so by default the step asks for nothing;
  stage 3 widens it as its field writers land, and the product's short description stays in
  `generate_body` until then.
- **With `rewrites` off the client's text is kept, then ours is added**: his blocks byte for byte,
  ours after his last text block.

### Decided while building

- **The token estimate** is `domain/llm.Tokens(text) = max(⌈words × 2.2⌉, ⌈runes / 3⌉)` in integer
  arithmetic (22 per ten words), because `100 × 2.2` rounds up to 221 in floating point; the run's
  estimate and the pack call the same function.
- **The link form lives on the plan, not the target**: `LinkContext.Absolute` and `Href(target)`
  give the site's form; `LinkTarget.URL` stays the path every comparison uses. A field on
  `LinkTarget` took it to 128 bytes, and `gocritic`'s `rangeValCopy` flagged ten loops.
- **A site's form is its own contextual links' majority**, ours left out, else the theme's
  `url_form` from the renders, else the site setting; `sitemodel.Absolute` reads it and the two
  link plans that splice a link (`resolve_context`, the neighbours' relink) carry it.
- **The pack's summary of an empty page is built, never asked for**: its meta description, else
  its excerpt (a product's short description), else the first sentence of its body (shortcodes
  skipped), else "Hub for Peptides under Home; lists A, B and 3 more; its children are …" from
  the model.
- **The input budget** defaults to the window less the output ceiling, the effort's reasoning
  allowance and the stable prompt (system, the bare user prompt, the answer's schema); an
  explicit `llm.inputBudget.<role>` wins, and one too small to carry what every call carries is
  refused in preflight (`input_budget_too_small`), never mid-run. The estimate prices each page's
  fitted pack (`run.Price.Input`) and skips a step that will not call a model (`run.Price.Applies`).
- **A cache key is named only when the stable prefix reaches 1,024 tokens** (`<step>:<siteId>`),
  the provider's minimum; below it a key only splits the cache.
- **Client blocks are the current blocks whose hash `page_blocks_written` does not hold.**
  `publish` and `publish_product` now record the hashes of the content the site stored; nothing
  wrote them before, so a second run would have kept the first run's article as client text.
  With client text kept, the body goes through `PutRaw` against the hash the writer read; a
  conflict pauses the item (`client_changed`), and without the plugin it pauses saying so. Our part
  carries no H1; `Structure` reads our part and `Compliance` the whole body.
- **The rewrite switch is the run's, set only in the run-start dialog with its confirmation**
  (`runs.StartRequest.Rewrite`, kept in `generate_body`'s params), because an autonomous agent
  confirms nothing.
- **`generate_fields` writes under the editor's role**, so the excerpt's pack takes the editor's
  budget; an excerpt scope is refused on a product, whose excerpt is its short description; the
  excerpt is read raw only, never WordPress's auto-excerpt; a revert puts it back unless the owner
  changed it since (`revert_excerpt_kept`). It is sent in `publish`'s one update with the body,
  not under a per-field checkpoint of its own: the artifact keeps the answer, so a retried publish
  never asks the model again.
- **`Sanitize` keeps a link the writer placed** when the field's policy carries links, the href
  resolves to a planned target and its text is one of that target's anchors, within the caps
  counted across the draft's sections, rewriting the href in the site's form; a kept link counts
  as placed. It now strips the model's classes, ids, styles and data attributes, which spec 16.3
  said it did and it did not.
- **`sites_explore` is one tool with five views** in the `sites` group, sent up front; it replaces
  `pages_tree` and `reports_link_audit_page`, takes its own arguments (the view, the gap and the
  pack's field are domain vocabularies: `sitemodel.View`, `contextpack.Gap`, the derived
  `packFieldKeys`), names a page by id or by path, and fits the pack view to a quarter of the
  tool-result cap in tokens. A page view caps each list at 25 and says how many it left out.
- **The link audit is read, not repeated**: `sitemodel` reads `reports.LinkAudit` through a
  consumer-declared reader, so the brief's link health, the gaps' orphans and the links view are
  one computation; the composition root builds `reports` before `sitemodel` for it.
- **The brief is built on a conversation's first turn**, not at its creation, and kept on the
  conversation (0064, written only by `SetBrief`, so an update made from a copy read earlier never
  clears it); it stays within 4,096 bytes by narrowing the hub tree first; a brief of planned
  pages says nothing of their text.
- **One in-memory pager** in the kernel, `Keyset.Page`, serves the explore views and the card
  proposals with the cursors a query issues, `before` included.
- **A card proposal is an `entity_card_proposed` finding** (Info, `accept` or `ignore`) whose
  subject carries the card proposed and the hash of the card it was proposed over
  (`Entity.CardHash`, the definition and the cleaned facts only); an omitted definition or facts
  keep the entity's and an empty list of facts clears them. In confirm mode the approval of the
  call is the acceptance (`ApplyCards`); `AcceptCard` refuses a card that moved since
  (`card_changed`); a dismissed proposal filed again stays dismissed. `FindingRepo` gains `Get`
  and `Settle`.
- **The tool schemas** went from 61,437 bytes at the stage's start to 61,554 (`generate_fields`'s
  enums), 62,369 (`sites_explore`, net of the two tools it replaced) and 63,203
  (`graph_propose_cards`), against a ceiling of 63,220.

### Deviations from the request

- **Docker was down for the whole stage**, so nothing that needs a stand has run:
  `TestThePageTypesFollowTheSite`, `TestTheWriterSeesTheFullPicture`, the capture
  (`TestTheStandModelIsCaptured`) and with it the stand pack goldens, `TestTheAgentSeesTheSite` and
  `e2e:full` on either stand. Each compiles, vets and lints under its tag; the owner's commands are
  in `STATUS.md`. `TestThePackOfEveryStandPage` skips, naming the capture, until a capture exists.
- **The excerpt's e2e on vanilla's posts** was left out of `TestTheWriterSeesTheFullPicture`; the
  write and the revert are proved against `wptest`.

### What the whole-branch review found, and how it was closed

The gate's fresh review found that with `rewrites` off the writer could still lose a client's
content; every finding was closed with a test that failed first.

- **The guard is the client's blocks, not the body's hash.** A body of client blocks without text
  (a product grid, a gallery, a form) was written whole: any block we did not write is now the
  client's (`block.HasClient`). An existing page with rewrites off always carries what the writer
  read, a body that was all ours included, so an owner's edit after the last sync pauses the item
  (`client_changed`). Publish reads the site's body fresh with the blocks recorded as ours,
  requires the client's blocks on it to equal those the writer read, composes over the site's
  current bytes, writes through the raw route against the hash just read and records our blocks
  straight after, so a retry after a partial publish composes the same body again and the revert
  record keeps the body the writer read. Without the plugin only a body with no client block is
  written. A planned page that finds a stranger's page by its slug pauses (`slug_taken`).
- **In a block-editor body our part is one `core/html` block**, so it can never fuse with a client's
  freeform span into one block recorded as ours; a freeform span that is a lone shortcode parses
  as a shortcode.
- **A relink splices** (`block.Splice`): it keeps the original bytes of every block whose render is
  unchanged, takes from the new render only the blocks the link changed or added, and carries
  `page_blocks_written` forward. Before, it wrote the whole re-rendered body, rewriting the bytes of
  blocks it never touched.
- **Validate judges our links only** (`content.ComplianceOf`); what the page owes is still met by
  the whole body.
- **The confirmed rewrite reaches every step**, and **the agent can no longer switch a scope to
  rewrite** (`pagetypes_set_scope` lost `rewrites`; the schemas went to 63,107 bytes, ceiling
  63,120).
- **A card is accepted only as it was shown** (`proposalHash`, `proposal_changed`).
- **A store category's twin is a page**, never a post or a custom-type item.
- **A link keeps its href's form** (`page_links.absolute`, migration 0065), which the auto link form
  counts; stage 3's migrations therefore start at 0066.
- One claim was wrong: the review read a stale `CLAUDE.md` and reported the old ceiling.

A second review of those fixes found the guard still hung on the draft, and four more holes:

- **Every write over an existing page is guarded** unless the scope or the run's confirmed pages
  say it rewrites: a page planned when the writer ran and synced before publish counts as read
  empty, so a client block on the site pauses it. A pause for the client's edit says to regenerate
  after a sync, the only way the writer reads the page again.
- **A write whose answer was lost is recognised** on the retry: the site holds exactly what we would
  have composed over what the writer read, so our blocks are claimed and the item completes.
- **A planned page adopts a page found by its slug only when this run created it**; any other,
  blank or not, pauses with a remedy that works (another slug, or mapping that page).
- **Splice and Ours align blocks by their normalised render** (a longest common subsequence), so a
  relink of several blocks keeps every block between them, and only blocks matching the part we
  inserted are claimed as ours: a client block the save changed, or one an unclosed tag merged with
  ours, stays the client's.
- **The single-value fields are guarded by their live value**: with rewrites off, a title, an
  excerpt, an SEO title or description or a short description the owner changed since the model
  read it is kept.

What stays open, each for the owner:

- **A relink whose answer was lost** finds its link placed on the retry and records nothing, so the
  block it rewrote reads as the client's on the next run: a second copy of our text is possible,
  a loss is not.
- **`adopt` records the links of our own writes as not ours**, so the auto link form counts them; our
  links are written in the site's form, so the count leans the way it already leant. Links stored
  before 0065 read as relative until a full walk reads them again.
- **The agent may type a page into a type whose scope rewrites** (`pages_update`, `pagetypes_delete`'s
  reassignment), and a run then rewrites it without the start dialog: the scope is the person's
  standing switch, so this was left as it is.
- **A draft-mode run sends `status: draft` to an existing live page**, and **the canonical is written
  whenever a snippet field is**: both predate stage 2b and sit against the 2026-10-17 ruling that an
  existing page's status is never written.

### What the stands found (2026-10-20)

Docker came back after the close. The captures, the exit tests and the whole suite on each stand
found nine things, each fixed test first or in the harness:

- **The capture's graph named no canonical page**, so every pack carried no link target:
  `seedStandGraph` makes each mapped page its entity's canonical (`832de25`).
- **A page's neighbours are the content beside it**: its siblings and children leave out technical
  pages, and for a page that is not a term they leave out store categories
  (`contextpack.beside`, `e1a52eb`). The hub's siblings had been the cart, the account page and its
  own twin.
- **A body's current value says what it holds beside text** (`93593a8`): a shortcode, a named block
  or markup with no text reads as kept where it stands. Without this, a hub whose body is one
  product-grid shortcode looked empty to the writer.
- **Listed products come in one order** (`1d8e204`). Their links came back in store order, which
  follows random ids, so two syncs gave the writer two orders and a different prompt prefix; they
  are now ordered by address. The page's own order is not known: a render link stores no position,
  because `render.Link.Position` is a content link's block index, and giving a render link the
  same column would give it a second meaning.
- **A page view fits a turn's tool result** (`6743148`). The replica's views ran 16.7 to 22 KB
  against the 16,384-byte cap, so the agent got nothing back. Links in and out are now counted by
  type with only the contextual ones listed, lists stop at fifteen, and
  `TestEveryPageViewOfTheReplicaFitsATurnsToolResult` holds every captured page under the cap.
- **Five suites predated 2a's template ruling.** An imported page has no template and the seeded
  types carry none, so a run on "auto" was refused. The writer test gives the site the Hub template
  as its default (`76d95f4`). The client, docker-site, hold and degraded suites give the Hub,
  Entity and Product types the Hub, Guide and Product templates, as a client would in Settings
  (`1b45924`).
- **The scripted fake answered the wrong product.** Since the pack lists a page's siblings and
  products, a reply matched by a bare path also matched a sibling's prompt. Every reply now matches
  the page's own `Path:` line, which also ends the `/menu/` against `/menu/main-courses/` prefix
  trap.
- **A shortcode block is named by its own tag** (`15b8d66`). Vanilla's golden showed the account
  page as `core/shortcode` where the writer needed `woocommerce_my_account`, and the replica's
  golden had carried the same line unnoticed. `Body.ShortcodesAt` holds the classic and editor
  rule once, for `Shortcodes` and the pack.
- **The writer test's planned child is its entity's page** (`07d4b70`): `planChild` created the
  page without making it the canonical one, as an import or a graph apply would, so the link plan
  blocked it and vanilla's `/guide/setup/` owed its child nothing.

Both stands then ran their whole suites green from fresh resets on the final code.

Open with the owner: **a fresh site cannot start a run on "auto"** until a person gives its types,
or the site, a template. The seeded writable types could point at the matching built-ins (Hub,
Guide, Product), as the page kind once did; that is a seeding rule and a backfill, so it waits for
a ruling.

## 2026-10-20 — stage 3: editing the client's pages field by field and block by block

Stage 3 is the first stage that writes the client's live sites: every enabled field through the
field registry under the hash of what it read, the body block by block, never the deny list, a
revert per field and per block. The twenty-three decisions taken at the plan, with the owner's four
answers and an Opus critic's findings, are under "Decided at the plan" in the stage 3 tasks of
`docs/superpowers/plans/2026-10-17-phase-14-site-model.md`; this section holds the owner's rulings
on what stage 2 left open and what was decided while building.

### The owner's rulings on stage 2's open items

- **The seeded page types carry their preset's template** (ruling 1). A fresh site's Hub type
  points at the Hub built-in, its Entity type at Guide and its Product type at Product
  (`pagetype.Preset.DefaultTemplate`), so a run on "auto" works without a person assigning one;
  Store category, Catalog and Technical stay without, nothing on them being writable. A type the
  client creates from a preset inherits that preset's built-in unless the request names one or
  names none (an empty id). Migration 0066 backfills, once, a writable type whose template is null,
  so a type a person sets to none afterwards keeps none. A site whose default template a person
  chose is left alone by the seeding and the backfill both, since a type's template wins over the
  site's default in `ResolveForPage` and would silently replace that choice. The built-in is the
  oldest global template of its name, as `templates.EnsureSeeded` finds it; on a database the
  built-ins never reached, nothing is filled. The suites no longer give the types their templates:
  they assert that a fresh site's types carry them.
- **An existing page's status is never written, and a draft run's writes to existing pages wait for
  review** (ruling 2, with the owner's answer at the plan that it covers every such write). An
  update sends no status and `compare` expects none, the page row taking the status the site
  answered; an update left with nothing to send (the body through the raw route, the title kept) is
  not made at all. In draft mode `publish`, `relink_page` and `relink_neighbors` compose what they
  would write over an existing page, keep it as an `edit_plan` artifact (each page's field changes
  before and after with their hashes, and its blocks kept, added or removed, ours or the client's,
  by `block.Diff`) and pause the item `awaiting_review`; the publish result carries the same diff
  once written. Accepting it requeues the step, which plans again and writes only when the new
  plan's hash equals the one shown (a neighbour relink composes its writes in a dry pass first); a
  plan that moved is shown again, and an acceptance never writes over a client change, the guard
  pausing `client_changed` as before. The acceptance is cleared once used, so a later retry is not
  taken as accepted. **Only a person accepts**: the runs service refuses an acceptance the agent asks
  for on an item awaiting review, and the tool registry now tags every call it runs as the agent's,
  which also makes the anchors and moves the agent makes read as the AI's, as the graph already
  expected. A hand pause cannot claim `awaiting_review`. A planned page is still created as a draft
  in a draft run, with no review. A draft-mode product run is no longer refused in preflight
  (`product_edited_live` is gone): it waits for review like any page. The image step draws and
  uploads nothing over an existing page in a draft run (`images_wait_for_live`), an upload being a
  write to the site. **A link target is on the site only when it is published** (`Page.Live`: a
  WordPress id and the `publish` status), so no link plan targets a page that is still a draft, and
  `relink_neighbors` stands down for a page its publish left a draft (`target_not_published`),
  instead of linking the client's live pages to an address a visitor cannot open. A page we create
  as a draft stays one until a person publishes it in WordPress. The suites accept a review as a
  person would; the full loop and the degraded loop now assert that no neighbour links to their
  drafts.
- **The canonical is never written to an existing page** (ruling 3). `generate_meta` no longer asks
  the model for one, nor builds one from the planned path; an update never sends it. On a page we
  create, it is written only when the page type's `seo_canonical` scope enables it and the page
  goes out live, as the `link` WordPress answered, since a draft's link is `?page_id=` and stays
  wrong once it is published. The scope is a person's switch: `field.Spec.Person` marks the
  canonical, `CheckScope` admits it, the pagetypes service refuses AI writes on it
  (`field_not_ai_written`), the pack leaves it out, every preset has it off, and the agent's
  `pagetypes_set_scope` enum stays without it, so the schemas keep their bytes. A revert puts the SEO
  fields back field by field, as it always did; an older run's canonical is put back by C.1's rule.
- **The agent cannot switch a page into rewriting, or trash one, unseen** (ruling 4, and the paths the
  plan's critic found beside it). A tool may name a per-call confirmation (`Tool.Confirm`), which the
  registry applies in every mode unless the call was approved, with a summary of its own:
  `pages_update` typing a page into a type whose scope rewrites a field it writes, `pagetypes_delete`
  moving pages into such a type, `imports_apply` whose preview types pages already in the map into
  one, and `pagetypes_set_scope` switching on a field, or its AI writes, that a person set to
  rewrite; the summary names the pages and the type and says their own text will be rewritten by
  the next run. `pages_delete` with `onSite`, which trashes the page and so changes its status for
  its visitors, always waits the same way. A confirmation whose lookup fails falls through to the
  tool's own path, which proposes in confirm mode and fails the call in autonomous mode. No schema
  changed: 91 tools, 63,107 bytes.
- **A relink whose answer was lost is recognised** (ruling 5). `relink_page` and `relink_neighbors`
  now keep the same kind of record as publish: a first execution composes every body they will
  write (the body read with its hash, the spliced body, the blocks it makes ours) and settles that
  plan, the review gate of a draft run included; then each write is its own execution, recorded
  straight after it lands. A write whose answer was lost answers 409 on its rerun, and the rerun
  reads the stored body: when it renders as the body composed, the write is taken as made, so its
  body, its blocks and its links are recorded as if the answer had come back and nothing is placed
  again; anything else is the client's edit and is reported (a conflict for a neighbour, a pause for
  the page). The neighbours' result is carried on every wait once a write has landed, so a relink
  stopped halfway still tells a revert what each written neighbour held. `wptest` gained
  `LoseTheAnswerOfTheNextRawWrite`, which stores a raw write and answers an error, the shape of a
  lost answer.
- **`adopt` records the links of our own writes as ours** (ruling 6). A link is ours when it sits in
  a block `page_blocks_written` holds, or when we placed it: a relink names every link it placed by
  its target and anchor, since a link put into the client's paragraph sits in his block. The rule
  lives once in `block` (`LinksOf`, `LinkAt.Ours`, `HeldOurs`), and the observation, `adopt` and
  `sync_back` all carry an earlier `ours` forward by target and anchor, as `origin` was, so a later
  sync never hands our link back to the client. The auto link form already counted the client's
  links only (`contextpack.form`), so its count no longer leans toward ours. Until migration 0069
  drops it, `origin` is still written `generated` beside every link that is ours.

### Decided while building

- **`publish` keeps a durable record, one write per execution.** The engine replaces a step's
  artifacts and merges its checkpoint at every settle, so a write made inside a long execution
  was lost to the record whenever the execution paused, failed or died before it settled. Publish
  now reads everything it will write over in a first execution (the plan, under the checkpoint
  key `publish`), then makes one write per execution (the raw body, the core item, the store's
  fields, the SEO meta) and returns `TransitionWait` waking at once, and every return, a pause and
  an error included, carries the checkpoint and the `publish_result` built so far. A product whose
  description conflicted after its short description was saved now pauses with that save on
  record, where it used to pause with nothing a revert could find. A rerun that finds its own raw
  write on the site (a 409 whose stored body renders as what it sent) takes it as written; a
  creation whose answer was lost finds its page by slug and, when the body renders as its own,
  records it as created, so a revert trashes it instead of emptying it. The before values are read
  in the plan, so a rewrite whose answer was lost keeps the client's body as what it replaced. The
  record starts again when the step's inputs change (a regenerated draft) or after it finished;
  a publish result is carried only once a write has happened, so a plan that paused leaves nothing
  for a revert. Every wait advances a sequence in the checkpoint, because the checkpoint is part of
  the step's input hash and an unchanged one would let the engine reuse the step as done.
- **The two tool enums the review touches stay narrowed** (`runs_pause`'s reason and
  `runs_get_artifact`'s kind): a hand pause cannot claim `awaiting_review`, and the edit plan is the
  person's to read in the run's report, so neither enum grows and the schemas keep their bytes.
- **The block operations write what they are given and parse back what they wrote.** `Replace`,
  `InsertAfter`, `Delete`, `RewriteRange` and `PlaceLink` on `block.Body` take the bytes as given;
  each result is parsed again and refused (`block_not_editable`) unless every block it did not touch
  comes back as itself, whole and in order, and the body keeps its kind, so an editor body never
  loses its last delimiter and a classic body never gains one. A separator added between blocks
  follows the body's own line ends; a delete that would join its neighbours puts a blank line between
  them; a delete of the last block takes the separator before it. So an insert undone by a delete,
  and a replace undone by a replace with the old bytes, give the body back byte for byte, which is
  what a revert per block stands on. `RewriteRange` puts the new bytes where the range's first text
  block stood and drops its other text blocks: a text block (`Block.Prose`) is a paragraph, heading,
  list or quote, or a group or columns holding only those, with prose markup and no shortcode;
  everything else stays where it stands. **No operation rewrites a block's markup**: a link is written
  into the bytes of the text that holds its anchor, the delimiters, attribute JSON and classes
  untouched, so the plan's "classes reapplied" had nothing to reapply. A block whose attributes hold
  the anchor (its text is an attribute's, as in a plugin's FAQ block) and a dynamic or shortcode
  block are refused. Wrapping our new HTML as one `core/html` block for an editor body stays the
  composer's (`block.Part`), so a revert can put back a client's freeform chunk exactly.
- **A link is never placed inside a shortcode** (found while building A.1). The linker matched an
  anchor in any text the HTML parser saw, and to it `[products category="coffee"]` is text: a relink
  could wrap `coffee` inside the client's shortcode attribute, and the splice then re-rendered the
  block, so his shortcode stopped working. `content.FindAnchor` passes over a shortcode tag and the
  text an enclosing shortcode wraps; the linker and `PlaceLink` both find anchors through it, and
  `TestPlaceLinkAgreesWithTheSplice` compares the two over random bodies.
- **The deny list names the store's meta keys, and a meta key is compared as MySQL compares it.**
  `field.DeniedMeta` lists the keys behind the denied fields (`_price`, `_regular_price`,
  `_sale_price`, `_sku`, `_stock_status`, `_stock`, `_manage_stock`, `_product_attributes`,
  `total_sales`) and the prefixes `_`, `attribute_` (a variation's attributes) and `rank_math_`
  (an SEO plugin's keys, written through the SEO route alone); `openapi.yaml`'s `x-denied-meta` is
  the same set, which a contract test holds equal, and `wp-plugin/testdata/meta_deny.json` is the one
  table the Go side tests now and the plugin's tests take in A.3. The `meta_key` column's collation
  ignores case and trailing spaces, so `TOTAL_SALES` or `total_sales ` reaches the store's own row:
  a key is compared after ASCII case folding, and one that is not ASCII letters, digits, underscores
  and hyphens, or is longer than 255 bytes, is refused whole (`field_unknown`), which also closes an
  accent-insensitive match. `field.MetaKeyAllowed` is the one predicate the scope, the client and the
  plugin share.
- **The H1 is read, never written, and the featured image is filled, never replaced.** `field.H1`
  is a text field read from the plugin with no writer; `CheckScope` refuses it as `field_read_only`,
  `AIWritable` is false for any field without a writer, and so no scope, pack or writer sees it.
  `field.FeaturedImage` is a core field marked `Fill`, written only where the page has none; it is no
  scope's to switch, the image step and the template deciding when one is drawn.
- **Plugin 1.5.0 writes under the hash read, and says what a save would do** (A.3). `PUT /meta/{id}`
  takes every key with the hash of the value it was read with (null for one the post did not carry),
  refuses the deny set, `is_protected_meta`, a key the user may not `edit_post_meta`, a value that is
  no text and a stored value that is not one plain text (an array, or more than one row, which
  `update_post_meta` would flatten), answers 409 `hash_mismatch` with `moved` naming each key that
  moved and writes nothing then, deletes on a null, and answers what each key now stores with its
  hash. The SEO route takes the same `expected` per field (the empty string's hash for a field not
  carried) and answers the stored hashes; the raw write answers the content WordPress stored beside
  its hash; the raw read answers `saveStable`, whether the stored content passes the user's
  `pre_post_content` and `content_save_pre` filters unchanged; the manifest answers
  `unfilteredHtml`. The capabilities `meta_write` and `guarded_write` join the required list, so a
  1.4.1 plugin reads as outdated and **the client writes nothing through it**: `PutRaw`, `SetSEOMeta`,
  `ReplaceSEOMeta` and `PutMeta` refuse it as `plugin_outdated`, since its SEO route ignores an
  `expected` it does not know and its raw write answers no content. `PutMeta` refuses a denied key
  itself before anything leaves. The fake's `WithFilteredHTML` now re-filters the body on the raw
  write and on a core update too, as WordPress does for a user without `unfiltered_html`, and one
  product test that pinned the description byte for byte against the old fake expects kses' form.
  A step that hands an SEO write no expected hashes keeps writing as before; `publish` and the revert
  pass them in A.5 and C.1.
- **The field registry is `application/edit`, three writers behind one `Write`** (A.4). It declares a
  core writer (title, excerpt, featured image, and the body on a site without the plugin), a store
  writer (a product's short description) and a plugin writer (the body under its hash, the SEO fields
  with their Open Graph twins, meta by key) over its own types; `runtime/steps/edit_writers.go`
  adapts `*wp.Client`, and a run builds the service per site client, so the composition root is
  untouched. `Write` refuses before anything leaves a denied or unknown key (`field_denied`), a key no
  writer takes here (`field_not_writable`, the H1 among them), a product's title, excerpt or featured
  image (`field_owned_by_store`), and a plugin field on a site without the plugin
  (`field_needs_plugin`); it asks the plugin for guarded writes before the first write, so an
  outdated plugin writes nothing, not even the core fields of the same call. It writes the core and
  store fields first, each group read just before its write and written only when every value still
  hashes as read, then the meta and the SEO fields, then the body, and answers each key as written
  (with what was stored), moved (with the live hash) or held behind a group that moved. `UpdateItem`
  now carries only the title, content, excerpt and featured image; slug, status, parent, menu order,
  template, terms and meta left it, a creation keeping what it needs on `CreateItem`. The variation,
  media, upsell and store meta writers arrive with their fields in Part B, `UpdateMedia` and
  `UpdateVariation` with them.
- **`publish` writes every field through the registry, the body last, after one check of them all**
  (A.5). Over an existing page the plan reads every value it will write over, as before, and the
  first write unit is now a check that reads them all again (the title, excerpt and featured image
  through core, the SEO fields and their Open Graph twins, the body's hash) and pauses naming each
  field that moved, nothing written and the plan dropped so a retry reads afresh. The core fields go
  next through `edit` (read just before their write), then the SEO meta under the hashes read, then
  the body under the raw route's hash; a field that moves between the check and its own write pauses
  the same way with the earlier writes on record. A rerun takes a field whose live value is already
  what it was about to send as written (`edit` treats such a "moved" value as ours), so a lost answer
  is never written twice. **Six client-ownership paths found by the stage's tests closed here**, each
  failing first: a rewrite replaced the whole body, the client's shortcodes, banners and other
  non-text blocks included, and wrote our H1 into it, where it now goes through `block.Rewrite` and
  ours carries no H1, as the kept path already did; the featured image replaced the client's, where
  it is now filled only; the Open Graph title and description were overwritten whatever they held,
  where each is now written only when empty or when it followed its SEO field; the SEO meta went
  unguarded; a body or SEO field the client edited after the read was written over; a page whose
  content the user's save filters would change was saved again, where now, without `unfiltered_html`
  over such a page, only the SEO meta goes and the item pauses `save_alters_client`. A plugin too old
  for guarded writes pauses the item before anything is written. Our blocks are recorded from the
  content the write answered, with no second read an edit could race, and the body snapshot is what
  the site stored. `record` and `adopt` write only what a publish owns
  (`PageRepo.UpdatePublished`: the WordPress id, status, content hash, the site's copies, the stamps
  and drift), so a person's edit to the plan during a run is kept. A creation is one call; the second
  update that tried to fix a mismatch after it is gone, since an update may no longer send a slug,
  parent or status, and a mismatch pauses as before.
- **A relink writes a link into its block's own bytes, and its plain sentence is a block of its own**
  (A.6). Both relinks keep the linker's choices as they were (`content.InsertTarget` over the page,
  with its anchor order, budget and parent rule), now taken on a fresh parse per target, whose render
  is the oracle; `block.ChangedBlock` names the one block it changed and `PlaceLink` writes the link
  there at the occurrence whose render agrees. A link the oracle placed where no byte-for-byte write
  agrees (an anchor in a block whose attributes hold it) is not placed and reads as an anchor not
  found, so the plain sentence takes over, the owed link said rather than dropped. The splice used to
  re-render every block a link touched: the client's `&nbsp;` became a raw no-break space and his
  single-quoted attributes double-quoted (`TestARelinkTouchesOnlyTheBlockItLinks`, failing on the old
  code). The plain sentence used to be appended inside the client's own paragraph; it is now its own
  paragraph, added with `InsertAfter` (one `core/html` block in an editor body) after the paragraph
  the linker chose, or before it for a link up, so it stays inside the opening paragraphs the parent
  rule allows, and it is ours. `block.Splice` has no caller left but the property test that compares
  `PlaceLink` with it.
- **A product is written through the registry, and attribute filling is retired** (B.1, the owner's
  answer 3 at the plan). The template's `product.specifications`, the writer's attribute values and
  their prompt lines, the judge's attribute check, the tool schema's `specifications` and the
  template editor's attribute rows are gone; the schemas fell from 63,107 to 62,174 bytes and the
  ceiling to 62,187. Migration 0067 strips the key from stored specs and overrides, which the strict
  decode would otherwise refuse, so an untouched built-in reads as the shipped seed again and needs
  no superseded copy. `UpdateProduct` carries only the short description and the images, and a
  product's first image is filled only where the store has none, through the registry's store
  writer. The product publish follows the page's order: one check that reads the short description,
  the images, the SEO fields and the description's hash again and pauses naming each that moved,
  then the store's fields through `edit` under the value read, then the SEO meta, then the
  description under the raw route's hash read after the store's save. Over a product whose
  description the user's save filters would change, the store's fields and the description are not
  written, since WooCommerce's save is a whole-post save, and the item pauses `save_alters_client`
  after its SEO meta. The product row is written through `UpdatePublished`. A revert of a run made
  before this still puts back the attributes it filled, through `RestoreAttributes`, which sends the
  whole attribute list alone, `wc/v3` replacing it whole.
- **The short description is a field of its own** (B.2). `generate_body` no longer answers it: the
  writer's schema is the draft alone for a product too, its prompt no longer names it, and its ceiling
  and the estimate count the body's words only (`writerCeiling(spec, attempts)`), so
  `content.ProductAnswer`, `ProductDraft`, `BriefProduct` and `template.Product.Words` are gone.
  `generate_fields` asks for `product_short_description` on a product page where the page type's
  scope enables it with AI writes **and** the template's `product.shortDescription` is enabled, the
  template keeping its switch in the editor and the preflight's `product_outputs_missing` staying
  true; a template that writes none is a skip with its reason. The template's intent, words and
  keyword rule join the scope's instruction and limits as the field's rules. The editor answers
  plain text, like every field, and the value is kept as paragraphs with `&`, `<` and `>` escaped
  (`content.ShortDescriptionHTML`), so nothing the model answers reaches the store as markup and the
  stored form is what kses keeps. `publish`, `validate` (the primary keyword rule, now only when the
  field was written this run) and the judge read it from the `fields` artifact; a page report shows
  it from there for a new run and from the draft for an older one, and the run's fields pane shows
  its text. An older draft's `product` is never read by a publish, so a retried older item keeps the
  store's short description rather than writing a value no field step produced. A run whose recipe
  leaves `generate_fields` out under a template that writes a short description is warned in
  preflight (`short_description_unwritten`), since the product then keeps the one it has; the
  shipped Product template carries the step. Like any field, a short description the client wrote is
  kept unless the scope or the run rewrites it, where the body writer used to draft one anyway that
  `publish` then dropped; the harness's product now starts with none, so its run writes one.
- **A write unit wakes at once** (found by the harness, fixing 0.2 and 0.6). The durable record's
  waits carried no wake time, which the engine reads as one sweep interval (`runs.sweepInterval`,
  5 s), so every publish idled about twenty-five seconds between its units and every relink a sweep
  per neighbour; the harness's suite ran past its fifteen minutes (it had taken 377 s). `publish`,
  `relink_page` and `relink_neighbors` now wake at the steps' clock, truncated to the second and so
  never after the engine's, as `sync_site` already did, and the engine dispatches the item again at
  once; the harness's suite takes 107 s. `TestEveryWriteUnitWakesAtOnce` drives each of the three
  and fails on a wait without its wake.
- **Every variation's description is written in one call per product** (B.3). The scope names
  `variation_description` once for every variation, so `pagetype.Set.Scope` resolves an instanced
  key (`variation_description:<id>`, `image_alt:<id>`) to its kind's scope, a meta key keeping its
  own. `generate_fields` reads the product's variations from the model (`contextpack.Variations`
  over the page's `variation:<id>` and `variation_description:<id>` rows), never from the store, and
  asks every one whose description is not the client's, or is his under a scope or run that
  rewrites, in the same structured call as the product's other fields, named by its options
  (`Variation description, Size: 5 mg`), with 256 tokens of room more for each; the plain answer is
  kept as escaped paragraphs (`content.PlainParagraphs`, the short description's helper renamed).
  The registry routes the key to the store writer's `ReadVariation` and `WriteVariation`
  (`wp.GetVariation`, `wp.UpdateVariation`, which sends `description` alone), after the store's own
  fields: every variation of a call is read before any is written, a moved one writes nothing, and
  one already holding our value is taken as written. `publish` reads each against the store in
  its plan, re-reads them in its check, writes one per execution after the store's fields and
  before the SEO meta, and reads the description's hash again after each, because WooCommerce saves
  the parent when a variation is saved; over a product whose save would change the client's
  description, no variation is written, as no store field is. The result keeps
  `previousVariations` (before and what was stored), the review diff names each, and the field
  rows are recorded. The revert puts each back while it still holds what the run wrote, and hands
  the whole product back when a person edited one since, as it does for the short description.
  The fake store reads and writes a variation under its product and, with filtered HTML, filters
  the parent again on a variation's save.
- **A save that cannot change the description does not make a change to it ours** (found while
  building B.3, closing a path in B.1). After the store's fields and after each variation, the
  description's hash was read again and taken as the base the raw write goes under, so a client's
  edit to the description landing between the check and that save was written over by our body.
  Those saves are made only over a product whose save is stable, which by the plugin's own test
  leaves the stored content as it was, so the hash read after them must equal the one checked: a
  difference now pauses the item naming `product_description`, the save already on record for a
  revert (`TestADescriptionTheClientEditsDuringTheStoresSaveIsNotWrittenOver`, failing first for
  both saves).
- **A product image's alt text and title are written only where the image is the product's**
  (B.4, decision 19). The observation records `image_alt:<id>` and `image_title:<id>` for every
  image the store lists on a product, from the store's own payload (its `alt` and `name`), so no
  request is added to a walk. `generate_fields` asks them like the variations, from the model and
  under the kind's scope (off on every preset), each named by its media id with 64 tokens more
  room, and keeps the answer as plain text, an attribute's text having no paragraphs. The registry
  routes them to the core writer's `ReadMedia` and `WriteMedia` (`wp.GetMedia`, which reads the
  attachment's parent, and `wp.UpdateMedia`, which sends `alt_text` and `title` alone) and refuses,
  before writing, an image whose parent is not the page (`image_not_attached`), so an image the
  library shares is never renamed for one product; an image's two fields go in one request and
  one execution. `publish` leaves such an image out at the plan and names it in `skipped` (the SEO
  write used to replace `skipped` whole; it now adds to it), re-reads each image in its check and
  writes it after the variations; an attachment's save is not a save of the product, so it is
  made over an unstable product too. The result keeps `previousMedia` (the shape of
  `previousVariations`, both now `FieldWrite`), the field rows are recorded, and the revert puts
  each back while it still holds what the run wrote, handing the product back otherwise. The fake
  seeds an upload under a parent, answers its `post`, and names a product image by its title.
- **A custom field is written by the provider its scope names** (B.5). `generate_fields` asks an
  enabled, AI-written `meta:<key>` where its provider can write it: the plugin when the site's
  plugin answers `meta_write`, the store on a product only, and nothing for a scope with no
  provider, each skip with its reason. The registry's `Write` names the provider: a store write
  goes through the store writer as `meta_data` by key (read back from the product's own
  `meta_data`, a value that is not one text reading as its JSON, so it never matches and is never
  written over), anything else through the plugin's `PUT /meta`; the deny list is refused before
  either, and `wp.UpdateProduct` refuses a denied key itself. `publish` reads each field against
  its provider in the plan (the plugin's meta read, or the product's `meta_data`), re-reads them in
  its check, and writes one unit per provider after the core or store fields and before the SEO
  meta; neither is a save of the post (WooCommerce saves the post only when one of its post fields
  changes), so they are made over an unstable page too. A page we create gets its custom fields
  after the creation, by the id it answered. The result keeps `previousFields` with each
  provider, the field rows are recorded, and the revert puts each back under the hash of what the
  run wrote (a key that held nothing is deleted), naming any the client changed since
  (`revert_custom_kept`) and putting back the rest of the page.
- **Upsells and cross-sells are product ids from the read layer's candidates** (B.6). A list is
  held in one plain form everywhere, its ids comma-joined in the store's order and empty for none
  (`field.IDList`, `field.ParseIDList`), so the kept rule of every field applies to it as written:
  a list the client made is kept unless the scope or the run rewrites, an empty one is filled. The
  observation records `upsells` and `cross_sells` from the store's `upsell_ids` and
  `cross_sell_ids`. `contextpack` offers a product's candidates in its facts (`Candidate`: the
  WordPress id, the name and the path): the published products under the same parent first, then
  the products of the entities the graph relates to it, each group ranked as the read layer ranks
  pages, at most twenty, never a draft, a page or the product itself. `generate_fields` asks a
  product's lists, both off on every preset, only when it has candidates (else a skip with its
  reason), lists the candidates in the field's rules, and keeps from the answer only the
  candidates' ids in the order named; an answer naming none is a skip, not an error. The lists go
  in the store unit's one request (`wp.UpdateProduct` with `upsell_ids` and `cross_sell_ids`), are
  re-read in the check, recorded in the product snapshot's `lists` with what the store answered,
  and put back by the revert while they still hold what the run wrote; `hasFields` now counts them,
  so a revert of a list alone is sent.
- **A crumb's name is matched against the fields and the plan** (B.7, decision 20's first half).
  The observation writes the `h1` field beside the title for every page that is not a term, from
  the H1 the plugin or the content answers; it is read-only, so a change raises no finding.
  `pagemap.NewIndexNamed` carries, per page, the names its `title` and `h1` fields hold, and
  `Index.ByName` and the landing check of a redirect match a crumb against those and the plan's
  title and H1, no longer the observed copy; the observation's model loads the site's field rows
  once and rebuilds the named index wherever it rebuilt the plain one. The test fixture's titled
  pages carry the plan's title, which the observation copies from the site wherever no plan is
  held. Twins still match on the observed copy, with their own matcher, until C.2 moves every
  reader of `Observed` onto the fields.
- **The product proof runs through a recording forward proxy** (B.8). `TestAProductIsEditedThroughTheRegistry`
  plants a variable product with the client's own description, short description, `dosage_note`
  meta and variation descriptions through the store's API, opens the core with `wp.WithProxy` at a
  proxy the test runs (so the site's host, and every link resolved against it, stays the stand's),
  enables the variation and custom-field scopes, and runs the product live with rewrites; it proves
  that no store write sent a name, slug, status, SKU, price, stock, category, tag, attribute, type,
  parent or menu order, that the only meta key sent is the custom field, that the store's own facts
  and every variation's facts read the same before, after and after the revert, and that the revert
  puts back the client's text. The suites' shared recipe gains `generate_fields`, which B.2 made
  the writer of the short description: without it the product loop wrote none.
- **A revert puts back each field and each block on its own** (C.1, decision 17). A publish
  result now carries `version` (3) and `writtenMeta`, what its SEO write sent; a result without
  a version reverts whole, as before. A versioned one is reverted field by field through the
  registry: each change in its `fields` diff is written back to its before value under the hash
  of what the run wrote (a custom field that held nothing is deleted), so a field the client
  changed since is never written over and is reported `kept` with a `revert_field_kept` finding,
  while the rest go back; the Open Graph twins go back with their SEO field under the hash of the
  twin the run sent; a product's first image comes off while it is still ours; the status is never
  touched. The body goes back whole and byte for byte while it is still what the run wrote;
  otherwise `block.Undo` walks the publish's block diff, finds each run of blocks the run added,
  intact and in order, among the live blocks (an anchor is the nth copy of its kept neighbour, so
  repeated blocks resolve), and puts back what each replaced, keeping the client's later edits
  elsewhere and naming any group he changed (a property test over random operations holds that
  every block comes back with its own bytes, in order). A created page is trashed only while its
  body is still what we wrote, else handed back. The revert pauses for a person only when nothing
  could be put back. `runs.RevertRequest` gains `pageId` and `fields` (checked against the catalog
  and the deny list) for a person, passed to the revert step as its `fields` param; with fields
  named, the neighbours are left alone. A second revert is refused only while another is under
  way, so it can put back what the first left, and a field already put back is written again as
  itself. `runs_revert` takes its own `{runId}` arguments, its schema unchanged. An older run's
  canonical is put back only while the live one is still what that run's meta artifact wrote. The
  old path's restores of variations, images, lists and custom fields are gone: no record before
  stage 3 wrote them.
- **A page's site title and H1 are read from its fields** (C.2a, decision 20's second half).
  `pagemap.Names` turns the site's `title` and `h1` rows into the named index, and the twins
  (`HubTwin`, `TermTwins`) and a product's match by name (`MatchProduct`, `ClaimProduct`) read
  those, a product whose name was never read (the observation's store candidate) answering by its
  plan's title. Every link plan that settles twins is built over the named index (the writer's
  context, both relinks, the judge and the link audit), so the writer and the audit see the same
  twins; the preflight, which reads only the blocked targets, keeps the plain one. Because the plan
  that names a pack's neighbourhood needs the site's names first, the read layer now loads the
  site's field rows once with the snapshot and no longer reads the neighbourhood's again. The
  observation tells a plan from the site against the last walk: the identity's URL (a core draft's
  query permalink compares no address) and the `title` and `h1` fields, read for the batch before
  its unit of work. The product brief and the judge take the store's name from the product's
  `title` field; the graph's mismatch compares the plan with the identity and the fields; the
  pages DTO loses `observed` and `mismatches`, and the product panel reads the store's name from
  the page model. `Mismatch` lives in `runtime/steps`, built from what a write asked for and what
  the site answered, and is encoded `{field, planned, actual}` (the old type carried no tags, so
  an older artifact's keys still decode).
- **Drift is the body's move from its base** (C.2b). A page drifted when its current body's hash is
  not its base's (`PageBodyRepo.Drifted`, one hash-only query per site), the condition the
  observation raises `body_changed_on_site` on; so drift now covers every page whose body moved
  from its first observation or from our last write, not only a page we wrote, and it clears when
  we write the page again or a person accepts the site's body. The pages DTO computes `drift`
  per site and loses `contentHash`; `publish` warns over drift and refuses it under
  `refuseDrift` from the body record, saying the page changed since it was last read or written.
  Nothing writes the page row's hash or drift any more: `record`, `adopt`, the read back and the
  revert leave the body to `page_bodies`, a relinked neighbour is reverted only while it holds the
  base the relink recorded (else it is handed back, as before when its hash differed or was
  unknown), and the observation tells a page we wrote by a field row with a written run or a block
  of ours, read for the batch with the fields.
- **The plan carries no canonical** (C.2c). Decision 12 writes a creation's canonical from the link
  WordPress answered and an update never sends one, so the plan's own canonical, which nothing
  wrote to the site, leaves `pages.CreateRequest`, `pages.UpdateRequest` (and `pages_create` and
  `pages_update` with them), the pages DTO, the drawer's form and summary and the agent's page
  cards, and the observation no longer copies the site's canonical into it; the site's canonical
  is the `seo_canonical` field. The schemas fall from 62,174 to 62,033 bytes (ceiling 62,046).
- **0068 drops the pages' old copies of the site** (C.2d). Before it drops `drift`,
  `content_hash`, `wp_link`, `wp_slug`, `wp_status`, `wp_title`, `wp_h1` and `canonical`, it
  writes a `title` field from `wp_title` and an `h1` field from `wp_h1` (not for a term, as the
  observation writes none) wherever the copy holds one and no row exists, its base its current and
  its time the last sync, else the row's update; the hash is `lower(hex(sha256(v)))`, which the
  writer connection can compute because it registers the driver's hash functions
  (`ext/hash.Register`, the same module), and the test proves it equal to `field.Hash` on UTF-8.
  The down restores the columns as 0045 made them and refills the titles from the fields, the
  status and the link from the identity, and, for a page we wrote (a written field or block), the
  hash from the base body and the drift from current against base, as the old row meant them.
  `pagemap.Observed`, `Page.Mismatches`, `Page.Drift`, `Page.ContentHash` and `Page.Canonical`
  are gone; `StatusFromWordPress` lives in `pagemap/status.go`.

### The image step never draws over the client's image (session 2, the open item)

- **An image is drawn for an existing page only where it has none.** `publish` already filled the
  featured image only where the site had none, but `generate_images` drew and uploaded one first,
  so a page or product with its own image left the client's media library holding an attachment of
  ours that nothing used, and the image model was paid for it. The step now reads the page on the
  site before it draws: a page or post whose `featured_media` is set, and a product with any image in
  its gallery, keep theirs, and the step draws its inline images alone, naming the kept image in an
  info finding (`featured_image_kept`); a read that fails fails the step, which is retried, so
  nothing is drawn on a guess. A planned page, and a page on the site without an image, get one as
  before. **No switch replaces a client's image**: the featured image is no scope's to switch (A.2),
  so a person who wants a new one removes his own in WordPress first. The estimate still counts the
  featured image of every page, being an upper bound read from no site.

### What the review of C.2 found

- **A neighbour's revert keeps its own record of what the relink left.** With `pages.content_hash`
  gone, the neighbour revert compared the live body with the body's base, which for a neighbour
  relinked before the bodies were stored (stage 1b) is the first observation: a client's edit made
  between that relink and the first walk would have been written over. A relink now records, per
  neighbour, the hash the site answered for its write (`NeighborResult.After`), and the revert puts
  a neighbour back only while its live body still hashes as that; an older result without it is put
  back only while the live body is the base and the base is that run's own (its blocks recorded under
  the run in `page_blocks_written`), and is handed back otherwise.

### C.3

- **Our links are told by their mark alone.** Migration 0069 drops `page_links.origin`, first marking
  `ours` on any row still recorded `generated`, so the old rule (`ours`, or `origin` generated) holds
  for every stored link; its down brings the column back, `generated` where a link is ours and
  `observed` elsewhere. `pagemap.LinkOrigin` and `PageLink.Origin` are gone, `block.HeldOurs` reads
  `ours` alone, and the page's links and the link audit's extra links answer `ours` where they
  answered `origin` (the `linkOrigins` vocabulary goes with it). `pages_replace_links`,
  `pages.ReplaceLinks` and `PagesService.ReplaceLinks` are gone: a list of links written over the
  model is a link no site holds, so the agent and the person change links through runs (a relink, a
  rewrite, a revert), which write the site and record what it stored. The schemas fall from 62,033
  to 61,343 bytes for 90 tools (ceiling 61,356); 129 methods are bound.

### What the stands found (session 2, 2026-10-20)

- **A page the site's user may not edit no longer stops the walk.** Under the replica's restricted
  user (an editor without `unfiltered_html`) every sync failed with "WordPress rejected the
  application password": the plugin's `/content` lists every content post, but its per-post routes
  answer 403 for one the user may not `edit_post`, and WordPress maps editing the privacy policy
  page to `manage_privacy_options`, which no editor holds. One such page failed the whole walk. The
  observation now takes a 403 from the plugin's terms, raw, custom field or rendered read as the page
  being closed to this user (`observe.ClosedToUser`, made by `steps/observe_readers.go` from
  `wp.Forbidden`): the page keeps what the listing gave and the terms and body it had, its render
  falls back to the public visit, and the walk counts it (`closed`), so a client whose application
  password belongs to an editor still has his site observed. `wptest` answers such a post (`Item.Closed`)
  as the plugin does (`TestSyncSiteWalksPastAPageTheSitesUserMayNotEdit`, failing first with the
  stand's message).
- **A revert found our own variation description to be the client's edit.** The publish result's
  field diff recorded what a run sent as each field's after value, and the revert writes back under
  the hash of that value; WooCommerce answers a variation's description through `wc_format_content`
  in every context, `edit` included (a line end after each paragraph, and texturized quotes), so the
  live value never hashed as what was sent and every variation was handed back as changed since. Each
  write now sets its field's after value to what the site answered it stored
  (`publishState.stored`), so the diff says what the page holds and the revert finds its own write;
  `wptest` answers a variation's description formatted in every context, as the store does, which
  made `TestAVariationsDescriptionIsWrittenAloneAndPutBack` fail first as the stand did. A reverted
  variation gets back the description as the store answered it, so its stored bytes may gain the line
  end the store's own formatting adds, which it renders the same. The SEO fields keep their own
  record (`writtenMeta`), which no stand showed moving.
- **The product proof's own setup.** Its scopes carried no link policy (`field.NoLinks()`, which every
  other caller passes), its scripted short description lacked the row's primary keyword, which
  `validate` requires of a short description written in the run, and it compared the reverted
  variations with a literal the store never answers; it now compares them with what the store
  answered before the run.
- **The restricted-user test checked the plugin before anything probed it.** `sites.Create` stores a
  site without asking it anything, so the plugin read as absent on the response of the create; the
  test now checks it through `Sync.CheckPlugin`, as the full loop does.

### G's safety work (session 3, 2026-10-22)

- **A relink records what the site stored, and keeps off a page its save would alter.** Both
  relinks kept only the hash of the raw PUT's answer and recorded the body they had composed as
  current and base, with the composed blocks' hashes as ours; on a site whose save filters
  reformat our bytes (kses, a filter adding `rel` to every link) our relinked block then read as
  the client's and the page as drifted on the next walk. `landRelinkedBody` now keeps the stored
  content, on the answer and on the 409 rerun alike, and `relinkWrite.stored` maps the composed
  blocks onto it through `block.Align`: the client's blocks pair by render, in order, and ours by
  position between them where each pair agrees by text, so a block the site prepended or dropped is
  never taken for ours (`TestAlignAnswersMinusOneForABlockTheSiteDidNotKeep`). The body snapshot,
  `page_blocks_written` and the adopted links are read from the stored content
  (`TestARelinkRecordsWhatTheSiteStored`, failing first with the composed body on record). As
  `publish` has done since A.5, no relink writes over a page whose raw read is not save-stable: the
  page relink pauses `save_alters_client`, a neighbour is skipped naming the reason, and nothing is
  PUT (`TestARelinkWritesNothingOverAPageItsSaveWouldAlter`). The fake gains `WithSaveFilter`, the
  site's own save filter, of which `WithFilteredHTML` is the kses case. A rerun after a lost answer
  still takes the write as ours only while the stored body renders as the composed one, so a filter
  that changes a render (an attribute added to our link) pauses the page or reports the neighbour's
  conflict rather than claim a body it cannot tell from the client's edit.
- **A revert records what the site stored.** The whole-body put-back of an older publish result, of
  a product and of a relinked neighbour (`writeBodyBack`, `undoNeighbor`) dropped the raw PUT's
  answer and recorded the body it had sent as current and base, so on a site whose save filters
  reformat it the next walk raised `body_changed_on_site` against the client and the link marks were
  rebuilt from a body the site did not hold. Each now records the content the site answered, or the
  live content where the site already held the before hash and nothing was sent
  (`TestARevertRecordsWhatTheSiteStored`, failing first for the page and the neighbour). The
  per-block revert of a versioned result already did (`revertBody`).
- **`sync_back` keeps the base of what the run wrote.** It read the page back and recorded the site's
  body as current and base both, so an edit the client made in the seconds between our write and the
  read-back became the base, and no later walk raised `body_changed_on_site` for it. It now records
  the read-back as the current body only (`Bodies.Observe`; the base stays what `publish` or the
  relink recorded from the site's answer) and, where the two differ, reports the difference in the
  sync result as a warning coded `body_changed_on_site`; the next `sync_site` raises the persistent
  finding from the same difference. It writes no field: `publish` records every field's base from
  the value the site answered (`TestSyncBackKeepsTheBaseOfWhatTheRunWrote`, failing first with the
  edit as the base and nothing reported).
- **What the stands found (session 3).** `TestAnExistingPageIsEditedBlockByBlock` was green on the
  replica as written and red on vanilla on its own assertion: a parsed block's raw bytes run to the
  next block's start, so the client's last block gains the separator when ours is added after it,
  which the stage's ruling on whitespace between blocks allows. The test now compares each client
  block's own bytes, the whitespace around it aside, and is green on both stands; the revert put the
  body back byte for byte on both.
- **Our mark on a link is tied to the block that holds it, never to a target and anchor alone.** Ruling
  6 of Part 0 carried `ours` forward by target and anchor, page-wide, so a link the client added
  later with the same target and anchor read as ours wherever it stood. A link is now ours while it
  sits in a block `page_blocks_written` holds, or in a block whose stored hash its own row records
  (`page_links.block_hash`, migration 0070, `pagemap.PageLink.BlockHash`, `block.PlacedKey`): a relink
  names each link it placed by the composed block that holds it (`block.Placed`, re-keyed when a later
  link lands in the same block, so a sentence inserted before it never shifts it) and, once the site
  has answered, by that block's stored hash through `block.Align`. **A mark moves only when we change
  the block**: a relink and a revert hand `adopt` the lineage of every block they changed
  (`block.Lineage`, the before body against the stored one, the client's blocks paired by render and
  the rest by their text in order), and `block.HeldOurs` re-keys the marks through it, so a second
  relink into the same paragraph keeps the first link ours and a revert keeps the marks of the links
  it leaves in place (`TestASecondRelinkIntoTheSameParagraphKeepsTheFirstLinkOurs`,
  `TestARevertKeepsTheMarksOfTheLinksItLeavesInPlace`, `TestALinkPlacedBeforeAnInsertedSentenceKeepsItsBlock`).
  A link the client adds is never ours, whatever its target and anchor
  (`TestALinkTheClientAddsWithOurTargetAndAnchorIsNeverOurs`); and a client's edit to a block that
  holds our link, an unrelated re-serialisation included, hands that link to him, by design, since
  nothing can then tell our link from one he kept. A row marked ours before the block was recorded
  is keyed, once, to the one block of the page's last stored body that holds its target and anchor,
  and left unkeyed when none or several do (`TestALegacyMarkIsKeyedToTheOneBlockThatHeldIt`); a
  relink state settled before this change carries no `placed` and its links are keyed the same way.
  A plugin that cannot answer the raw body (below `raw`, so already refused for writes) marks nothing
  ours from the item's links, there being no block to check. The observation keys against the body it
  stored before this walk's own read, `sync_back` against the body before its read-back, and a revert
  against the body before its put-back; `0070`'s down drops the column.

### What the closing review found (session 3)

A fresh reviewer walked the eight paths the brief named with one question: is there any path where a
run, a relink, a revert, a sync, an image step or an agent tool changes what the client owns, or
records his change as ours, or ours as his. It found one blocking path, five serious and five minor.
Each is closed below with a test that failed first, or kept with its reason.

- **Our textless blocks stayed ours across a relink only by luck.** `block.Align` paired the blocks in
  a gap by their text, and a block with no text (a figure we drew, a rule) never paired, so a relink
  over a page that holds our inline images recorded `page_blocks_written` without them and the next
  run kept our figures as the client's and drew them again. A textless block now pairs by its render
  (`TestARelinkKeepsOurTextlessBlocksOurs`, failing first with the figure gone from ours).
- **The save-stability probe covered the body alone; a whole-post save also runs kses over the
  client's title and excerpt** (the review's one blocking path). `wp_update_post` merges every stored
  field and `wp_insert_post` sanitizes the merged title and excerpt for the database, and for a user
  without `unfiltered_html` kses hooks `title_save_pre` and `excerpt_save_pre` as it hooks
  `content_save_pre`: our raw write, a core title write or a store save over a product whose name
  holds a `<sup>` or whose short description holds an iframe stripped them, nothing had read or
  recorded those fields, and the next walk raised `field_changed_on_site` against the client. The
  plugin's `saveStable` (1.5.1) now answers false when the stored title or excerpt would not pass
  their own `pre_post_*` and `*_save_pre` filters either, the fake mirrors it over the three fields,
  and the client requires 1.5.1, so a 1.5.0 plugin reads as outdated and writes nothing
  (`TestAPageWhoseTitleOrExcerptASaveWouldAlterIsNotResaved`, failing first with the page written;
  `TestSaveStabilityFollowsTheUsersFilters` gains a title with an ampersand on the stand).
- **A result from before stage 3 put the SEO meta back blindly.** `putTheMetaBack` sent the previous
  meta with no expected hashes, so a revert of an older run wrote over an SEO title the client had
  edited in Yoast since. The revert now expects, per applied field, the hash of what that run's `meta`
  artifact wrote, and a 409 keeps the client's value with `revert_meta_kept` naming the edit
  (`TestAnOlderRunsSEOMetaIsPutBackOnlyWhileItHoldsWhatThatRunWrote`, failing first with the edit
  overwritten).
- **The excerpt's base was the value sent, not the one stored.** `finishItem` recorded the request's
  excerpt as the field's base while the registry had already answered the stored value, so under a
  restricted user an excerpt with an ampersand read as changed on the site on the next walk and was
  kept as the client's. The base is now the stored value the write answered
  (`TestTheExcerptBaseIsWhatTheSiteStored`, failing first; the fake now runs its filter over a page's
  title and excerpt on every whole-post save, as WordPress does).
- **A revert never gave `page_blocks_written` back.** A publish replaces the page's whole set and a
  revert left it, so after run B's rewrite was reverted the body held run A's article again while the
  set still named B's blocks: A's article read as the client's, the next run kept it as his text and
  wrote a second one, and its links read as his. `readopt` now records the set the put-back leaves
  (`block.Restored`): the blocks of the body that were ours and stand, those our own lineage put back
  (a relinked paragraph of ours returned to its earlier bytes), and those the reverted result's diff
  names as ours and removed, which the per-block revert uncovers
  (`TestARevertGivesBackTheBlocksOfTheRunItUncovers`,
  `TestARevertKeepsTheMarksOfTheLinksItLeavesInPlace`, failing first with B's hashes kept). A
  whole-body revert of a result from before stage 3 has no diff, so an article an older run wrote
  under it comes back as the client's: an older record, the owner's call.
- **The SEO record was what the write sent, not what the plugin stored.** The plugin answers the
  hash of what each SEO field stores after the SEO plugin's own sanitizer (Yoast collapses whitespace
  and strips tags from a title), and the write dropped it: `WrittenMeta`, the field diff and the
  field's base all held the sent value, so our own title read as the client's edit on the next walk,
  was kept as his by the next run, and a revert's expected hash never matched. The write now keeps the
  answered hashes and, where one differs from the sent value's, reads the SEO meta back once and takes
  the stored values for the record, the diff and the base (`storedSEO`;
  `TestTheSEORecordIsWhatThePluginStored`, failing first on `WrittenMeta`); the fake hashes what it
  stored, as the plugin does, and `WithSEOSanitized` is the sanitizer's case.
- **A lost answer over a render-changing save filter left the client's text off the record and let a
  revert claim success.** A rewrite's raw PUT lands, the process dies before the settle, and the site's
  own filter adds an attribute to our link: the rerun's 409 reads back a body that renders as neither
  what was read nor what was sent, and the step pauses (rightly: it cannot tell its own filtered write
  from the client's edit), but the carried `publish_result` held no `previousContent`, so a revert put
  the title and the excerpt back, skipped the body without a word and answered `restored` while the
  site held our rewrite. The conflict path now carries the text the write replaced, and a revert of a
  result whose body write was never answered reports the body `kept` with the reason
  (`revert_field_kept`, `ReasonRevertBodyUnproven`) instead of silence
  (`TestALostAnswerOverARenderChangingFilterKeepsTheClientsTextOnRecord`, failing first with nothing on
  record). The neighbour's variant stays a `relink_conflict` the person resolves: the relink cannot
  tell its filtered link from the client's edit either, and the conflict names the page.
- **Kept with their reasons.** (1) A variation's description is put back as the store answered it,
  texturized and autop'd by `wc_format_content`, so a client's `Don't miss` comes back as
  `Don&#8217;t miss` in the editor's textarea; WooCommerce's API answers no raw form of a variation's
  description, the rendered page is the same, and the ruling of 2026-10-20 already accepts the line
  end: the owner's call whether that form is acceptable. (2) `PublishResult.Blocks` holds the diff of
  the composed body, not of the stored one, so on a site that reformats our bytes a later client edit
  elsewhere makes `block.Undo` miss our run and the revert answers `revert_field_kept`: conservative,
  ours stays and nothing of the client's moves; recomputing the diff from the stored body is listed
  for the next session with the reviewer's proof (a save filter reformatting ours plus a client edit
  in another block, revert, our block gone). (3) A row marked ours before migration 0070 is keyed
  once to the one block of the last stored body that holds its target and anchor; if our link had
  been deleted and the client's own link with the same target and anchor stands alone in one block,
  that row is keyed to his block. Accepted: the old rule had already marked such a link ours, the
  keying never claims a second block, and every row written since 0070 carries its block. (4) Under
  a restricted user a title write with an ampersand pauses `refuseMismatch` because kses stores
  `&amp;`: noise, not ownership (the title is ours and nothing of the client's moves), left for the
  gate's residue list.

### Session 4 (2026-10-22): the stored diff, per-field acceptance, the screens

A Fable critic read the session's plan before it ran and found one blocking path (a declined field
recorded under the run, so the next run would write over the client's value) and six serious ones;
each is folded into the entries below.

- **The publish result's blocks are what the site stored** (the closing review's open item (2)). The
  diff was computed once from the composed body and never carried onto the stored one, so on a site
  whose save filters reformat our bytes (kses, a filter adding `rel` to a link) a later client edit
  anywhere made `block.Undo` miss our run and the revert hand the body back. Worse, the stage's own
  proof showed that `page_blocks_written` held nothing of ours under such a filter: `block.Ours`
  matched our part by its render, which the filter had changed, so the next run kept our article as
  the client's. `block.Landed` now carries the composed diff onto the stored body through
  `block.Align`, the relink's way: a kept or added block takes the bytes and hash the site stored and
  keeps its owner, and a removed one is unchanged. In a gap that Align leaves, a textless block of
  ours (a figure whose link the filter changed) pairs with the one textless stored block there. A gap
  holding a client block keeps every stored block nothing pairs with as the client's and kept, so a
  revert never removes it. A gap of ours alone marks such a block `added` with the new owner `site`:
  it is never ours, and a revert removes it with our run, restoring the body as it was. Entries keep
  the stored order, so `Undo`'s runs stay contiguous. `writeRawBody` (on the answer and on the 409
  rerun) and the core or creation path of `writeItemFields` set the result's blocks to the landed
  diff, and `recordOurBlocks` records its ours, so `page_blocks_written` and the diff agree
  (`TestARevertFindsOurBlocksTheSiteReformatted`, the reviewer's proof with a linkless client body so
  the save stays stable; `TestOurBlocksStayOursUnderARenderChangingFilter`; both failing first). The
  plan's `theirs` and `part` had no reader left and are gone; an older checkpoint still decodes.
- **A person accepts a draft run's review field by field** (the owner's answer at this session's
  plan). `RunsService.RetryStep` takes `fields` with `acceptFindings`: the keys a person ticked in
  the review, each one the catalog knows, not on the deny list, and among what the item's newest
  `edit_plan` offers (its field changes, and the body under `PageEdit.body`, `body` or
  `product_description`, when its blocks change). Anything else is refused as invalid. So are an
  explicit empty set, fields on a plain retry, fields on an item no review holds, and fields on a
  review that changes bodies only (a relink's), which is accepted whole. The agent's
  `runs_retry_step` takes its own arguments, so its schema keeps its bytes (90 tools, 61,343).
  `Engine.Accept(itemID, fields)` writes the set beside the acceptance on every accept, empty
  meaning all, and `run.ClearAcceptance` clears both wherever the acceptance was cleared, so a
  partial acceptance never outlives the plan it was given for: accept {title}, the site moves,
  the plan is shown again, a whole acceptance writes everything. `publish` narrows only after
  the review gate has compared the hash of the plan it shows, before its first write unit:
  - a declined key joins the plan's kept keys, so nothing records it under the run and the next
    run still reads it as the client's;
  - its write is emptied (the title kept, the excerpt and an SEO field with its Open Graph twin
    unsent, a variation, an image's text, a list or a custom field dropped);
  - a declined body sets `keepBody`, so neither the raw route nor core's update sends it, and the
    body is neither recorded nor its blocks claimed;
  - a unit left with nothing to send leaves the plan, and the check reads only what is still
    written;
  - the result names what was declined (`PublishResult.declined`) and its field and block diffs
    hold only what was written, the product snapshot included.

  A page we found keeps the item the site answered from its plan, so a page whose item unit drops
  out still records what the site holds. `TestADeclinedFieldIsNeverRecordedAsOurs` proves that a
  declined key never reaches `Fields.Record`, the only writer of `written_run_id`, rather than
  running a second live publish, because the step harness's field log does not feed a recorded
  row back. A page's featured image and a product's first image are filled only where none exists
  and are no part of the diff; a draft run uploads none over an existing page
  (`images_wait_for_live`).
- **A field revert never takes a page the run created off the site** (found while building D.2's
  Pages-drawer revert). The revert step trashed a created page whatever fields the request named, so
  a person putting back one SEO title would have removed the whole page. A field of a page that did
  not exist has no earlier value to put back, so a revert that names fields over a created page now
  hands it back with `ReasonRevertCreatedWhole`, and only a revert of the page whole takes it down
  (`TestAFieldRevertOfAPageTheRunCreatedLeavesItOnTheSite`). The publish pane offers no field ticks
  for a created page; the Model tab, which does not know whether the writing run created the page,
  says so in its confirmation.
- **Our H1 is never written into an existing page with no text of the client's** (found by the gate's
  stand run on the replica's main hub). With `rewrites` off, the composition wrote the whole rendered
  body, its `<h1>` included, wherever the page held no client block, so the client's hubs, whose raw
  is empty and whose theme prints the title as the H1, gained a second H1. A.5's "ours carries no
  H1, as the kept path already did" held only where the client had blocks. The body written over an
  existing page is now ours without its H1 in every case
  (`TestAnExistingPageWithAnEmptyBodyGetsNoHeadingOfOurs`, failing first with the H1 on the site).
  The guard recognises our own earlier write in either form, with the H1 (a page we created, or one
  written before this) or without (`TestARerunOverAnEmptyPageTakesTheFirstRunsBodyForOurs`). A page
  we create still carries its H1 in the content, as it always has; that is left for the owner.
- **What the gate's first `e2e:full` on the replica found** (the first since stage 2b's, the stage's
  own work never having run the whole suite). `TestTheClientLoopFromTheSamples` cancelled its second
  run on the first publish `step.done`, which since the durable record (0.2) is also the plan's wait
  before any write, so the revert rightly found nothing written: the test now waits until an item
  holds a `publish_result`. `TestTheWriterSeesTheFullPicture` refused its page because
  `TestTheSiteFiltersAreRespectedUnderARestrictedUser`, which runs before it in the same suite (A.7),
  restored the framed page it edited and left its kept page with our text; it now restores both, and
  the two pass in order from a fresh reset.
- **A revert without the plugin never claims the body back** (found by the gate's
  `e2e:full:noplugin` on vanilla). Without the plugin, `publish` writes the body through core and
  keeps no copy of what it replaced, since it cannot read the raw body. The per-field revert of
  such a result skipped the body without a word and, with nothing else to put back, answered
  `restored`, while the client's page still held our rewrite. The body is now handed back with
  `ReasonRevertNoBody`, and a revert that puts nothing back pauses naming that reason, as the
  revert before stage 3 did (`TestARevertWithoutThePluginNeverClaimsTheBodyBack`, failing first with
  the false `restored`).
- **The screens (D.1 to D.4).** The run report's publish pane shows each field before and after and
  the body's blocks as the site stored them, marked ours, the client's or the site's and kept, added
  or removed, with the regions the site injects read from the page model (whether inside the content
  root; no position among the blocks is recorded, so none is shown). The `edit_plan` pane is the
  review with a tick per field and on the body. The revert pane lists each field's outcome. A field
  is reverted from the publish pane or from the Pages drawer's Model tab, whose rows are the page
  model's `fields`. The canonical's AI-writes switch is disabled as a person's switch, a confirmation
  card shows a per-call summary, `runs_revert` has a card, and the plugin panel labels the 1.5.1
  capabilities. The harness seeds a live edit with its diff, a draft review left waiting and a
  revert of one field, and a restart now puts back the body the model holds rather than a run's
  draft. A field's revert outcome (`restored`, `kept`) has no vocabulary in `vocab.ts`, and the
  publish result does not name its body key (the screens take `product_description` when a product
  snapshot is present); both are left as they are.

## 2026-10-23 — stage 4: the workbook reconciled with the site through findings that last

Stage 4 reconciles the client's workbooks with the observed site through persistent findings a
person or the agent resolves, and gives the agent its tools over the model. The decisions taken at
its first session's plan, with the owner's answers and a Fable critic's findings, are under "4.1
tasks" in `docs/superpowers/plans/2026-10-17-phase-14-site-model.md`; this section holds the owner's
rulings and what was decided while building.

### Stage 4 in three sessions, and the agent's tools folded into three (the owner)

- **Three sessions, each gated like stage 1's sub-stages.** 4.1 is the matcher and the findings:
  `MatchRow`, the workbook reconciled into persistent findings, their lifecycle and the resolution
  use case, with no agent tool and no screen. 4.2 is the agent working the loop: its tools over
  findings, fields and blocks, `sites_update`, `pages_get` answering the model, and the agent
  resolving findings end to end on the fake provider. 4.3 is the screens (the Findings inbox, the
  Site map, the import preview naming each row's match rule), the full gate and phase 14's closing
  report. Each writes its task list at its start and ends on a checkpoint.
- **Nine planned tools become three.** The schemas stand at 61,343 of 61,356 bytes, and every byte
  is resent on every round of every turn. `pages_fields_get/set`,
  `pages_blocks_list/replace/insert_after/delete/place_link` and `findings_list/resolve` fold into
  `pages_fields` (`get`, `set`), `pages_blocks` (`list`, `replace`, `insert_after`, `delete`,
  `place_link`) and `findings` (`list`, `resolve`), each with an `op` enum, the way `sites_explore`
  carries its views. Writes stay confirmable. `templates_resolve_for_page` folds into `pages_get`,
  as the plan said. Deferred tool loading stays off until the owner has tried a funded key.

### The owner's answers to stage 3's open questions (Part 0)

- **No second H1, adaptively.** A page we created still carried our H1 in its content, while the
  theme printed the page's title as the H1 outside the content root on both stands and on the
  client's first site, so every page we created had two. The extraction now records whether the
  theme prints one (`render.Extraction.ThemeH1`, `page_renders.theme_h1`, migration 0071):
  `printed` when an `<h1>` stands outside a located content root, or inside it with text the raw
  does not hold (a classic theme whose title shares the content's wrapper); `absent` when there is
  neither; unknown when the root was only assumed or the visit landed elsewhere. One rule decides
  for created and existing pages alike (`render.OwesH1`, read through `sitemodel.OwesH1`). Over an
  existing page, our H1 goes in only when its own render is `absent` and none of the client's
  blocks that stay holds an H1; with `rewrites` on, his text blocks give way, so only the blocks
  the rewrite keeps count. On a page we create, it goes in only when every observed render of its
  WordPress type is `absent` and none is `printed`. With no observation, never. The guard still
  recognises an earlier write of ours in either form, with or without its H1, and a creation
  whose answer was lost is found in either form.
  (`TestExtractSaysWhetherTheThemePrintsTheH1`, `TestOurH1IsOwedOnlyWhereTheThemePrintsNone`,
  `TestTheReadLayerSaysWhetherOurH1IsOwed`, `TestACreatedPageCarriesNoH1WhereTheThemePrintsTheTitle`,
  `TestACreationWithNoObservedRenderCarriesNoH1` and `TestOurH1OnlyWhereTheThemePrintsNone`, the
  publish cases failing first with the H1 written; the full loop on the stands asserts that no
  draft it created carries an H1 in its content.) Validation still reads the draft's own H1; the
  page shows the title the theme prints. A render stored before 0071 reads unknown until the page
  is rendered again, which a full walk with "Render every page again" does at once.
- **A reverted variation description in the store's formatted form is accepted.** WooCommerce
  answers a variation's description through `wc_format_content` in every context and keeps no raw
  form a client can read back, and the page renders the same, so a revert puts back the client's
  text in that form (`Don't` comes back as `Don&#8217;t`). One defect sat under it: the revert
  recorded the value it sent as the field's base, while the store answers that value formatted
  again, so the next walk read the store's own formatting as the client's change and raised
  `field_changed_on_site`. The revert now records what the store answered, as publish has since
  `e2080f7`, and `wptest` formats a variation's description as the store does, apostrophes
  included (`TestARevertedVariationDescriptionInTheStoresFormIsAccepted`, failing first on the
  fake's formatting and then on the recorded value).
- **Plain-text fields compare with HTML entities decoded.** Under an editor without
  `unfiltered_html`, kses stores a title's `&` as `&amp;`, so a title of ours read back as the
  client's edit, a whole-post check paused, and the next walk raised `field_changed_on_site` on text
  nobody had changed (the closing review's residue (4) of stage 3). `field.SameText` decodes the
  entities and trims for the title, the excerpt, the SEO title and the SEO description; the Open
  Graph twins compare under their SEO field's key, as their guard follows it. It decides at every
  place that asks whether the site changed a value: the observation's field pass, which agrees the
  base to the site's stored form, so a later write under its hash still holds, and keeps whose the
  value is; `keptFieldsOf`; `checkBeforeWriting`; the SEO plan's `movedSince`; the title mismatch
  after a write; and the registry's "already stored" arm of `edit.checked`, whose hash arm stays
  raw because a hash cannot be decoded. What we write and every stored hash are unchanged. The
  fake now runs its save filter on a creation as WordPress's `wp_insert_post` does, which the
  publish proof needed (`TestSameTextDecodesOnlyPlainFields`, `TestAnEscapedAmpersandIsNoChangeOnTheSite`,
  `TestAKsesEscapedTitleIsNoMismatch` and `TestATitleStoredWithItsEntitiesEscapedIsTakenAsWritten`,
  each failing first).

### 4.1, decided while building

- **`MatchRow` is one matcher over a `RowSite`** (`pagemap.NewRowSite(pages, fields, entities)`),
  built once per sheet from the site's pages, their field rows and the graph, and asked per row.
  Its rules run in the owner's order, with a person's mapping (`mapped`, `pages.row_key`) ahead
  of them all, over the site's unarchived pages first and the map's planned rows only when no
  page on the site answers, so a second import finds its own planned row and a page the client
  publishes later wins. A product row sees products only; a page row sees every page but a
  product, except by its address, which is an address whatever it holds. The H1 rule compares a
  row's H1 with the page's own H1 and, on a page that carries none, with its title, because the
  theme prints the title as the page's H1 (both stands and the client's site): that is what made
  `ClaimProduct` find a product by the H1 the sheet gave it, and it now does so through
  `MatchRow`.
- **The import matches every row once, before it drafts pages** (4.1.2). A match pass gives each
  row an address: the page it matched, else its URL, else the slug of its title, entity name or
  H1 under the page its nearest level names (a level that names the row itself is passed over),
  else the root. Drafts are keyed by that address, while the hierarchy work that reads a row's
  URL (`fillGaps`, `urlParent`, the groups, the products under a hub) still reads the URL, so
  WooCommerce's `/product/` base never becomes an entity. `held`, `MatchProduct` and the
  product matcher are gone. An address an archived page or a slug collision already holds is taken
  as a `url` match, which keeps `UNIQUE(site_id, path)`. A product row of a sheet read as products
  only by its mode, which finds no product, falls back to a page at its address. One candidate
  updates that page; several are `row_ambiguous`, naming each path, and the row is left out before
  drafts and groups, so it writes no entity and no page; none is `row_without_page`, a planned
  page, or a product row waiting as before (the import no longer raises `product_not_in_store`,
  which stays the run's refusal). `PreviewPage.matchedBy` names the rule for every matched row.
- **A slug, title or H1 never crosses branches.** A sheet nests its forms under each compound
  (`/peptides/bpc-157/capsules/`, `/peptides/tb-500/capsules/`), so the slug rule alone merged one
  compound's capsules into the other's. When a row's URL parent and a candidate's URL parent are
  both known (a page, or a row of the same sheet) and differ, that candidate is dropped and the
  next rule asked. A flat page's parent is the root, never known, so the client's flat pages still
  match a nested sheet URL by slug. The rule lives in the import (`matchAs`), beside the sheet's
  own rows it reads; `RowSite.Match` stays the site's matcher.
- **A row without a URL now plans a page** wherever it names something and no page answers it,
  as the owner ruled at the plan: an exported entity with no page, a level row of products, a
  group. An export that is edited and imported again therefore plans pages for them. The `sku`
  column joins the import's fields (the header `SKU`), so the agent's mapping enum carries it in
  three tools and the schemas grow from 61,343 to 61,361 bytes (ceiling 61,374), the one schema
  move of 4.1.
- **The sheet's levels are the row's expected trail, read and compared, never written** (4.1.3;
  this amends the 2026-10-04 ruling that ignored every level column but Root, as the 2026-10-17
  record foresaw). Category, Subcategory and Sub Subcategory, in any spacing, are detected and read
  as the column use `trail` ("Expected parent"); only Root still makes groups. A row's trail is its
  level cells in header order, placeholders dropped, the last dropped when the groups' own name or
  slug evidence says it names the row, and it is stored as `pages.planned_trail` on a created page,
  a matched page and a waiting product row. A trail column the file does not carry is refused,
  like a missing root. `pagemap.TrailDiffers` compares it by name (the page's title and H1, as
  plan and as fields, and a term page's title) with the chain of the page's model parents, the
  front page and technical pages left out and a store-category twin folded into one node; a
  product is compared with each chain of its `product_cat` terms, and the subject names the
  closest. The preview raises a transient `parent_differs` and the observation the persistent one
  from the stored trail, under the same marshalled `{sheet, site}`, cleared as `observed_clear`
  when they agree or the trail is empty; the id-based raise of 2026-10-18 is gone. **Two
  deviations, kept for 4.1.4**: the preview does not yet compare product rows (the import has no
  term reader; one line in `build_services.go` and a consumer interface give it one), and entity
  names are compared on neither side, because the observation reads no entities and a name only
  the import compared would flip the persistent finding on every walk.
- **The derived parent wins over a workbook parent unless a person set the parent source to the
  workbook** (amending 2026-10-18; decision 7 of the 4.1 plan). Under `parent_source = workbook`
  the derivation keeps as the model parent the page the trail's nearest level names, by the name
  rule `DeriveParent` uses (now `Index.ParentNamed`), with the reason `workbook`; a tie keeps the
  derived parent and raises `parent_ambiguous` naming the candidates. That parent shapes the tree
  and the link plan and is never written to an existing page; a planned page's one creation call
  still sends it, as the owner's answer 1 of 2026-10-17 allows.

### The owner's answers to 4.1's first session (2026-10-23)

- **A field rewrite that pays for a whole draft is accepted for 4.1 only.** `rewrite_field` runs
  the template's whole chain and `publish` writes only the named fields, so a meta description
  costs a body draft. A run scoped to chosen fields is a real need, not a later nicety: the client
  will ask for sweeps such as "rewrite the meta descriptions of 300 pages", and paying a body
  draft for each is the spend he already complained about. **4.2 builds it**: a run can target
  chosen fields, the steps for fields it does not touch are skipped, `publish` requires only the
  drafts of the fields it writes, and the estimate prices only those.
- **Re-importing an export plans no page for an entity that has none.** A row that carries only an
  entity (no URL, no title, no H1) matches or updates the entity and plans no page. A row with a
  URL, a title or an H1 that no page answers is `row_without_page`, which plans one, as before.
  Group entities without a page exist on purpose (`no_page` in the link plan), and a re-import must
  not fill the site plan with pages nobody asked for. This amends the plan-time answer of the same
  day: a URL-less row with no page is planned at the slug of its title, else its H1, never of its
  entity name.

### 4.1, the second session (2026-10-23)

- **A row has a key** (`pagemap.RowKey`): `row:<sheet>:<identity>`, the identity being the row's
  path, else the `graph.Key` of its title, entity name or H1. The import sets it on every row, so
  a person's mapping (`pages.row_key`) is found by `mapped`, and it is the subject key of the row's
  findings. A row named only by its SKU has none, and every CSV shares the sheet name `""`; neither
  touches the client's workbooks, and both are named in the checkpoint.
- **A persisted subject carries no physical row number.** The lifecycle compares subjects, so a row
  moved down the sheet would reopen a finding a person settled. `row_ambiguous` holds
  `{sheet, rule, candidates, pageIds, product}` and no page; `row_without_page` holds
  `{sheet, plannedPath, product}` on the planned page or the waiting product row. Both offer
  `map_to_page` and `ignore`; the first is a warning, the second a note.
- **Apply persists what the preview shows, inside its unit of work**: the row findings, and
  `parent_differs` under the observation's subject and options (`pagemap.TrailOptions`, one rule
  for both sides), cleared by the import when the trail agrees. A product's trail is compared
  through its `product_cat` terms in the preview too. The open row findings of each sheet the import
  read and did not raise again close as `row_matched` (the row is still there) or
  `row_left_sheet`. The observation closes `row_without_page` as `row_matched` once its page has a
  WordPress id. The preview writes nothing.
- **`map_to_page` is `imports.MapRow`**: the row key moves onto the chosen page (with the planned
  trail when the page has none), the finding moves there and settles as `mapped`, and only then is
  the planned page deleted, when it is still planned and no run item names it. A product row maps
  only to a product.
- **`set_type` needs no new column**: the observation leaves a page alone whose type a person (or
  the agent) set to a non-technical preset, so its technical reason stays empty and
  `technical_detected` closes; a page typed Technical keeps its detection.
- **`findings.Resolve` records who chose**: `By` is the person or the agent, and the agent's
  acceptance of a card is recorded as the agent's (`graph.AcceptCard` takes it). A model change and
  its settling are one unit of work; a run option settles nothing and answers the request:
  `rewrite_field` a start on the page with `fields` and `rewrite` (a field the client changed is
  his until the run rewrites it, which the start dialog confirms), `relink_page` a relink start,
  `rescan` a full sync that renders every page again. `accept_site` on a title or an H1 makes the
  plan take the site's value.
- **A run scoped to fields** (`StartRequest.fields`, publish's `params.fields`): only writable
  keys, only with an enabled publish step. Publish narrows its plan to them before the review gate
  with the same narrowing a person's partial acceptance uses, the body only when named, and records
  nothing as declined. The estimate says, as a note, that the run still drafts each page whole: the
  owner accepted that for 4.1 only, and 4.2 skips what the scope does not need.
- **The drawer's `parent_differs` sentence read the subject from before `2f1813f`** (`reason`,
  `workbookPath`) and said only "a different parent"; it now reads `{sheet, site}`, failing first.

### What the stands found at 4.1's close (2026-10-23)

- **The client's workbook on the replica** (`TestTheWorkbookReconcilesWithTheSite`, all four sheets
  in their order): 9 rows update a page (4 by URL, 2 by slug, 2 by title, 1 by entity name), none
  by a person's mapping, H1 or SKU, none ambiguous, 13 plan a page (the 4 Catalog rows wait for
  their products), and 2 rows carry only an entity (TB-500, matched by its entity; "Tissue
  repair", an entity alone, planning nothing). After one import and one sync the open findings are
  `row_without_page` 14, `technical_detected` 7, `parent_differs` 3, `parent_unresolved` 1 and
  `render_redirected` 1. The import and the observation raise the same three `parent_differs`
  (Reconstitution under Peptides in the sheet and Lab Guides on the site; BPC-157 and TB-500 under
  Peptides in the sheet and under Peptides › Healing Peptides on the site), and a sync changes no
  count.
- **A row that matches a page the import itself planned keeps a `row_without_page` too**: it has
  no page on the site either. That is why 13 planned rows give 14 findings (the Variations sheet's
  TB-500 liquid matches the Groups sheet's planned page by URL). The preview warns only where it
  creates a draft; the persisted finding names every row still waiting for its page, and the
  observation closes all of a page's row findings when it lands.
- **Resolved through `core.Findings` on the stand**: the dosing guide's row ignored by a person,
  Reconstitution's trail pinned to the workbook by the agent (its model parent becomes Peptides on
  the next sync), `rescan` answering a full sync; a second import created nothing and reopened
  nothing, and a sync after it kept the dismissal.
- **Vanilla** matches `/guide/` by URL and the sheet's `/brewing/setup/` by slug onto
  `/guide/setup/`, and plans the URL-less "Cold brew at home" with its `row_without_page`. The
  import also plans a `/brewing/` gap page from the matched row's URL, the long-standing gap
  filling; it is put to the owner at the close of 4.1.

### The owner's answers at 4.2's plan (2026-10-23)

- **The agent's writes to the site wait for the person in either mode.** A field or a block the
  agent writes to a page (`pages_fields set`, every `pages_blocks` write) is proposed in an
  autonomous conversation too, as `pages_delete` of a page on the site already is. Resolving a
  finding and starting a run change the model or start work the estimate prices, and follow the
  conversation's mode.
- **A title scoped without the body is answered by one editor call.** A run scoped to the title
  alone drafted each page whole, because the body writer writes the title. When the title is in a
  run's scope and the body is not, `generate_fields` answers it in its one structured call, briefed
  with the workbook's planned title, so a title sweep costs about what a meta sweep costs.
- **Gaps, CSV keys and the SKU key** (the owner's answers to 4.1's close, in the brief of 4.2): a
  row that matched an existing page at another address plans nothing, gaps included; a CSV's row
  keys name its file, so two CSVs never touch each other's findings; a row named by its SKU alone
  has a key. The import and the observation share one term-parent helper.

### 4.2, decided while building (2026-10-23)

- **Part 0.** Only a row that plans a page at its own URL fills the gaps in that path: a row
  matched elsewhere, or to a page on the site, plans nothing, gaps included (`8c6d75d`). One
  consequence: a product row matched to its store product at another address no longer creates the
  entity-only level its sheet URL names (the "Mak" of `/mak/…`); only an unmatched row does. A
  CSV's rows key under its file's base name (`row:pages.csv:…`) and its row subjects name the file,
  while a workbook keeps its sheet names; no alias reads the old `row::` keys, which no release
  carries, and renaming a CSV now drops a person's mapping (`c85378e`). A row named only by its SKU
  keys as `row:<sheet>:sku:<folded SKU>`; such a row plans no page (it has no path), so the one
  finding it can raise is `row_ambiguous`, when two products share the SKU (`96a3874`). The import
  and the observation read a term's WordPress parent through one `pagemap.TermParents` (`71ea057`).
- **A scoped run keeps only the steps its fields need** (`run.ScopeRecipe`, `112e73f`): the body's
  steps (`resolve_context`, `generate_body`, the links, the images, `validate`, `judge`,
  `relink_neighbors`) for the body, a product's description and the featured image; `generate_meta`
  for the SEO title and description; `generate_fields` for everything else, a title without the
  body included; publish, `sync_back` and `report` always. The scope is copied onto every kept
  step. A step declares its requirements under a scope (`StepDef.Scoped`, read by `RequiresFor` in
  the recipe check and the claim's input hash); publish requires only `run.DraftsOf` its fields and
  `generate_meta` the draft only with the body in scope. The requeue needed no change: a scoped run
  never produces the draft, so it can never purge it.
- **Publish without the body** (`247124f`) reads no draft and no body, composes nothing, runs no
  guard, rewrite or H1 rule, plans no raw write, records no body and refuses no drift; the title
  comes from `generate_fields` when the scope names it. It stands down on a page not on the site,
  and `runs.Start` refuses such a scope over a planned page before anything is spent and adds no
  ancestor. An unstable page is held for a person only when the plan would save the post. Without a
  draft, `generate_meta` describes the page as it stands (its title, H1, summary and the current
  body's headings). The estimate prices the kept steps alone and raises no `recipe_differs` for a
  scope; `Price.Applies` takes the step's params so `generate_fields` sees the scope.
- **A call's risk comes from its arguments** (`Tool.RiskOf`, `readsOn`; `7dae806`), so a folded
  tool's reads never wait and its writes propose as any write does, with that call's risk on the
  card. A summary a tool cannot build now refuses the call: before, the registry ran the write in an
  autonomous conversation when the lookup behind its summary failed.
- **`findings`** (`d1978f9`) lists open findings by default and resolves one as the agent; a model
  option proposes a plain sentence in confirm mode and runs at once in an autonomous conversation (the
  owner's answer 1), a run option answers its request with no card, and another site's finding is
  never read or resolved (`findings.Get`, the site on `ResolveRequest`). Its code and option enums
  come from `finding.Codes()` and `finding.Options()`, and `vocab.ts` carries `findingStatuses` and
  `findingOptions`. A tool named after its group alone belongs to it. `runs_start` takes `fields`
  and `rewrite`, and in confirm mode its card names the pages, the scope and the estimate.
- **An agent's write is a run of the kind `edit`** (migration 0073; `3c5e54e`). `runs.StartEdit`
  enqueues it live, by the agent; its step `edit_page` loads its page itself, plans in one execution
  and writes in the next. A block edit takes relink's path (the plugin, a save-stable page, the body
  hash the agent listed, `PutRaw` under it, the landed blocks, `recordSpliced`, `adopt`); a field
  goes through `application/edit` under the value the agent read, its before taken from the model
  and checked against that hash. The agent's HTML is scrubbed by `content.SanitizeBlock` (classes,
  ids, styles, data attributes and links go, an `h1` becomes an `h2` unless the page owes one) and
  becomes one `core/html` block in a block-editor body; a link comes only through `place_link`, whose
  href is the target's path, or its absolute URL when the site's setting says so. A replaced or
  inserted block is ours, a deleted one leaves our record, and a block that takes a link keeps its
  owner; the diff marks a changed client block as his. The result is a version-3 `publish_result`, so
  `runs_revert` undoes it whole or per field, and the person's confirmation is its review.
- **`pages_fields` and `pages_blocks`** (`57b5a91`) read through `sitemodel.Fields` and
  `sitemodel.Blocks` and refuse before any card a denied, read-only or person's field, a page off
  the site or of another site, a site without the plugin for a block, a moved hash, a block past
  the end, and a replace or delete of the owner's block without rewrites; an empty `hash` on a
  field means it was read empty, and a field that has gained a value since is then refused as
  moved. Their writes wait for the person in either mode (the owner's answer 1).
- **`pages_get` is the agent's one page view** (`d1691f4`): `sitemodel.Page` answers the page view's
  whole neighborhood with the identity, the type and its source, the parent's reason, method and
  trail, fields with current, base, hash and owner (each value cut at 300 runes), the body in sum
  by owner (a long body is never repeated whole) and the open findings with their ids and options,
  each long list naming the op that pages it; the tool adds the template a run would use with its
  source (`ResolveForPage` answers `source` and `name`). `templates_resolve_for_page` and
  `sites_explore`'s `page` view are gone.
- **`sites_update`** sets the technical shortcodes and blocks and the page builders, each replacing
  its list, an empty list giving the defaults back (`111e9b2`).

### Deviations from the 4.2 request

- **`pages_get` gives the parent's trail with its reason and method, not the render's crumbs**: the
  derived chain is the model's answer and the crumbs stay on `parent_*` findings' subjects, which
  `findings op list` shows.
- **HTML refused before the card is an empty block only**: whether an operation fuses blocks or
  turns the body's kind depends on the live body, so the step finds it and holds the item for a
  person with the reason.
- **A link the agent places takes the site setting's form**, not the majority of the site's own
  links that a run reads from its link context.
- **The core's step-registry pin moved one commit late** (`85052a6` after `3c5e54e`): the `app`
  package was not run with the edit kind's commit.
- **The cap test of `pages_get` runs in `sitemodel` over both stands' captures** (the widest model
  6,418 bytes on the replica and 3,225 on vanilla), with the literal 16,384 the tool-result cap
  defaults to, since `sitemodel` cannot import `application/agent`.

### What the stands found at 4.2's close (2026-10-23)

- **Replica**: the agent listed 13 open `row_without_page` findings, mapped the dosing guide's row
  onto `/bpc-157/` (the planned guide went) and pinned Reconstitution's parent to the workbook, each
  on a card a person confirmed; `rescan` on `render_redirected` answered a full sync with no card;
  the meta sweep over `/peptides/` and `/reconstitution/` drew a card reading "estimated $0.00 for
  6262 tokens" on the fake provider, drafted no body and wrote the meta description alone; a second
  import created nothing and reopened neither the person's ignore nor the agent's mapping. The
  agent's block edit on `/peptides/` replaced our last paragraph and two reverts gave back the run's
  bytes and then the base; the hub carries no block of the owner's.
- **Vanilla**: on the block-editor `/guide/setup/` the owner's heading and paragraph stood byte for
  byte while the agent replaced our `core/html` block, the injected regions did not move, and the
  reverts gave back the bytes. The nested sheet planned no `/brewing/`.

## 2026-10-23 — stage 4.3a: real token counts beside bytes, and 4.2's two loose ends

The owner split 4.3 in two: 4.3a counts what a call costs in the tokens of the current OpenAI models,
as a local metric beside bytes, and closes 4.2's two loose ends; 4.3b keeps the screens, the full gate
and phase 14's report. The task list and the checkpoint are under 4.3a in the phase plan; a Sonnet
researcher chose the tokenizer and a Fable critic read the plan (no blocking finding; five serious
folded in).

### The tokenizer

- **`github.com/tiktoken-go/tokenizer` v0.8.1** (the owner's link, the researcher's pick): MIT, pure
  Go, no cgo, `o200k_base` compiled in as generated source, nothing fetched at test time or on first
  use; tags run to 2026-10-21. `pkoukk/tiktoken-go` was rejected: it downloads by default, its
  offline loader dates from 2024, it carries an open `o200k` round-trip bug and knows no gpt-5.
  v0.10.0, two days old, rewrites the merging: v0.8.1 is pinned, and every count of the report was
  checked once against v0.10.0 through `-modfile` in the scratchpad: the two JSON reports are
  byte-identical.
- **Verified** by `TestTheCounterAgreesWithTheEncodingsOwnVectors`, the library's own `o200k_base`
  vectors read from the module cache (`'RE`, `hello world` as `[24912 2375]`, two and three spaces,
  `supercalifragilistic` in six, the fourteen-token sentence) and the empty string. OpenAI's own
  `tiktoken` tests hold no `o200k_base` ids.
- **Its cost on this machine**: a cold build of its package with its standard-library dependencies
  takes 61 s and the compiler peaks at 381 MB; a test binary that loads `o200k_base` peaks at a
  26 MB working set with a heap of about 5 MB, so the race groups gain little.
- **The encoding map is the command's** (`encodingFor`), after tiktoken's names and prefixes for
  `o200k_base`; the catalog's three `gpt-5.6-*` map, and `gpt-image-2` falls back to `o200k_base`
  with the report saying so. Each measured prompt is counted in the encoding of the model its role
  resolved to.
- **Never in the build**: imported only from `_test.go` files and `cmd/tokenreport`.
  `TestTheTokenizerStaysOutOfTheBinary` joins `internal/app/deps_test.go` as a new rule beside the
  others, which stay as they were: `go list -deps ./cmd/postulator`, plain and `uiharness`, holds no
  tokenizer package, and no non-test package but the command imports one. It was run red once
  against a scratch import in `kernel/clock`, which was not committed.

### Decided while building

- **The chat prompts moved into `application/agent`** (`Instructions`, `Situation`), beside the
  title prompt and the `SiteContext` it renders, as `application/content` owns the judge's prompt;
  the runner renders through them. The report and the instructions tripwire read them there, so the
  rule that only `internal/app` imports `transport` needed no exception (the critic's finding; the
  plan had first exempted the command).
- **`sitemodeltest`** holds the capture format and the in-memory store the `sitemodel` tests used,
  beside `applicationtest`; both stands' `site.json` moved under its `testdata` and are embedded, so
  a test in another package and the report load a stand by name. The `packs.golden` files stayed and
  did not move a byte. It has no tests of its own, so `covergate` neither counts nor misses it.
- **`runtime.EstimatedInput`** is the input the estimate prices for one step on one target (the
  prompt overhead, half the step's output and the pack its read layer prices); `price` calls it, so
  the report compares the estimate itself.
- **The report runs the code as it is**: `resolve_context`, `generate_body`, `repair_links` and the
  judge over each captured page, with a model client that records the request and answers nothing,
  a page's template resolved through `templates.ResolveForPage` over the built-ins (each captured
  type given its preset's template, as migration 0066 does), the catalog's default model per role
  and the tuning defaults. Every write its adapters are asked for is refused. A placeholder in a
  prompt, a pack missing from its prompt or a step that errs before its request fails the report.
- **The tripwires**: `schemaCeilingTokens` 14,626 for 14,623 tokens beside `schemaCeilingBytes`;
  `instructionsCeilingTokens` 1,924 for the autonomous instructions' 1,921 with the replica's
  captured brief (confirm mode 1,918), in `application/agent`. Both took the byte ceiling's margin.
  The schema log's "about bytes over four" was 16,430, 12% above the real count.

### What the report found (2026-10-23, both captures)

- **The agent**: the instructions are 1,655 tokens in confirm mode and 1,658 autonomous; the 92
  schemas 14,623 (`templates_create` the widest at 911); the replica's brief 209 and vanilla's 150;
  every round of every turn sends 16,587 tokens on the replica and 16,528 on vanilla before the
  conversation, 3,668 and 3,609 with tools loaded on demand.
- **The steps**: eight pages are measured (six on the replica, two on vanilla); the others are named
  with the reason (technical, store category and catalog pages have no template; the products, the
  front page, the posts and the sample pages are unmapped). The writer's prompt runs 1,258 to 1,825
  tokens (357 of instructions, a pack of 240 to 782, 474 to 546 of the rest, a 188-token schema), a
  linker call 255 to 266, the judge's prompt 766 to 1,299 over the captured bodies.
- **The estimate against the real count**: the run estimate prices the writer's input at 2.28 times
  the real count, pooled (2.08 to 2.52), a linker call at 3.57 and the judge at 2.07: the 800-token
  overhead and half the answer stand for less than they price, and the judge's estimate has no body
  term. `llm.Tokens` counts a pack at 136% of its real tokens (a section from 0.54 to 1.04 of it).
  `Fit` keeps every pack under its budget in fact: at 1,500 tokens three packs were cut and the
  widest kept 802 real tokens, and no pack went over at 4,096, 3,000 or 1,500.
- **Tokens per word**: three captured bodies carry fifty words or more (the WordPress privacy,
  refund and sample pages, 1,343 words); as a writer would answer them (HTML without the block
  editor's comments, as a JSON string) they run 1.71 to 1.85 tokens per word, 1.77 pooled; as plain
  text 1.17, as HTML 1.59.
- **Proposed, not changed** (the owner's call): `llm.WordTokens` at 1.8 tokens per word instead of
  2.2; `llm.Tokens` at 1.7 tokens per word and 3.7 runes per token instead of 2.2 and 3 (both move
  `tokensPerTenWords`, the one constant the two read); `steps.tokensPerWord` (3) holds against the
  densest body's 1.85. The input overestimate is `promptOverhead` and `inputShareOfOutput`, not a
  tokens-per-word constant. The sample is thin and is WordPress's own boilerplate, not the client's
  copy: the ledger's real `input_tokens` from a funded key is the comparison that settles it.

### The replica's client blocks (Part 2.1)

- **The owner's answer at the plan**: the replica already holds a hub with the client's text,
  `/tb-500/` (two paragraphs and his link to `/bpc-157/`, typed Hub, `RootFound`), so the proof uses
  it instead of a new paragraph on `/peptides/`; the stand, its pins and the captures did not move.
  `writerStand.clientPage` names the page each stand proves on (`/tb-500/`, `/guide/setup/`), and
  `TestTheAgentEditsABlock` refuses a page with no block of the client's, asserts his blocks after
  the run and again after the agent's edit, and reverts to the run's bytes and then his base.
- **What the proof found**: the first replica run failed. His paragraphs stood byte for byte, but
  after the agent's edit his link sat in the model a second time with an empty region. The steps
  that record a body's links (`sync_back`, `relink_page`, `relink_neighbors`, the revert, the field
  revert and `edit_page`, through `observedOn`) wrote them with no region and resolved a relative
  href against the site's root, while the observation writes the same rows in the content region,
  resolved against the page (the 2026-10-18 ruling), and a sync that finds the same body writes
  nothing over them. `observe.ContentLinks` and `observe.Resolver` are now the one way a body's
  links are recorded and marked, on both sides (`1c7bf8d`, and `3207739`, which adds the file
  `1c7bf8d` left out of its commit: the tree at `1c7bf8d` alone does not build).
  `TestTheLinksARelinkRecordsStandInTheContentWhereThePageResolvesThem` failed first on both
  counts. `TestSyncBackResolvesALinkAgainstTheFolderTheSiteLivesIn` had pinned the root resolution
  (`coffee/` from `/blog/coffee/espresso/` landing on `/blog/coffee/`); its linking page moved to
  the subfolder site's own front page, where a relative link still resolves under the folder.
  `ResolveAt` does not collapse `..`; that is unchanged.
- **Green after the fix**: on the replica `TestTheAgentEditsABlock`,
  `TestTheWholeLoopReachesTheDockerSite` and `TestAnExistingPageIsEditedBlockByBlock`; on vanilla
  the same three. Both stands were taken down.

### The runtime flake (Part 2.2), stopped by the owner

- **Attempt 1**, `go test -race -count=10 -p 1 ./internal/runtime/...`, green in 3,219 s (runtime
  1,974 s, steps 1,127 s), while the session compiled and tested beside it.
- **Attempt 2**, the same with `-cpu 1,4`: the first run was cancelled by the owner before any
  package finished; the rerun took 8,484 s and failed in `internal/runtime/steps`, in the one-CPU
  pass (no `-4` suffix): `TestATitleSweepIsAnsweredByTheFieldsStepAlone` (79.62 s) and
  `TestAMetaSweepIsNotHeldOnAPageASaveWouldChange` (34.38 s), both "timed out waiting for the run
  to finish", both 4.2's scoped sweeps over three pages. `internal/runtime` passed 20 times. The 4.2
  checkpoint's "runtime package failed once" was most likely this package of the same group.
- **What the investigation established**: a running step does not lose its item (the lease is the
  step's own timeout plus a second, `claim.go`); the "re-armed from waiting" lines are publish
  waking between its units. The working cause is starvation: at `-cpu 1` the `-parallel` default
  stays at the machine's eight, so up to eight parallel tests, each with its own engine, share one
  P under the race detector, and a three-page run outlives the harness's fixed 20 s `pollTimeout`,
  the same fixed wait the `runtime`, `runtime/steps` and `sync` harnesses keep.
- **Not proven, not fixed**: the instrumented reproduction (`go test -race -count=10 -cpu 1
  ./internal/runtime/steps/`) was stopped by the session's memory guard with nothing logged, and the
  owner stopped the hunt there. What landed is the evidence for next time: a sweep's timeout names
  each item's status, step, attempts and error (`97671c1`). The fix the evidence points to is a wait
  bounded by the test binary's own deadline instead of a fixed span. To reproduce, with output that
  survives a stopped run: `go test -race -count=10 -cpu 1 -v ./internal/runtime/steps/`.

### Deviations from the request

- **The flake is found, not fixed**: the owner stopped the hunt after its two runs (5.4 hours in
  all); the record and the command are above.
- **The instructions moved** from `transport/agent` into `application/agent` instead of a line in
  `deps_test.go`; `deps_test.go` gained one rule (the tokenizer's) and changed none.
- **The captures moved** into `sitemodeltest/testdata` so every package loads them by name.
- **No product prompt is measured**: every product in both captures is unmapped, and the steps refuse
  a page with no entity. A capture taken after mapping the products would measure them.
- **The briefs carry no link totals**: a capture holds no link audit, so a few tokens of the real
  brief are missing.
- **The per-turn context message** is a line of the agent's round and of its sum, beside what the
  request named.

## 2026-10-24 — stage 4.3b: the screens, the full gate, and phase 14 closed

4.3b puts the loop on screen (the Findings inbox, the Site map, the import preview's match rule and
pick, a card per op for the agent's edits and resolutions), fixes 4.3a's flake, and runs the whole
gate on the final code. The task list and the checkpoint are under 4.3b in the phase plan; a Fable
critic read the plan (no blocking, five serious and six minor findings, all folded in), and one Opus
agent built the screens once the bindings had landed.

### The owner's answers at the plan

- **The Site map is its own screen**, after Pages in the rail; the Pages screen's Tree toggle and the
  files only it used are gone, and `Pages.Tree` stays only for the run start's page picker.
- **The flake is fixed**, checked once. A harness waited a fixed 20 s for a run; it now waits until
  the test binary's deadline less 30 s (`66517ed`). `go test -race -count=3 -cpu 1 -timeout 60m
  ./internal/runtime/steps/` passed in 839 s, and the two scoped sweeps that failed in 4.3a took
  25.7 to 72.9 s over their six runs, five of them past the old 20 s: starvation it was. A first
  check without `-timeout` ran into the binary's own ten minutes on its third count, which is the
  deadline-bound wait giving up as designed.

### Decided while building

- **One map for two screens** (`71de2ee`, `7600f02`). `contextpack.Map` is the site by derived
  parent, a store category folded into the hub of its name: the hub is the row and names its twin,
  the term is no row, its children join the hub's and its findings and links are counted with it;
  a loop the folding makes is cut at its first path. `sitemodel.Map` loads it for the findings
  (which declare a reader of it, so `under` is the map's subtree), and `sitemodel.SiteMap` answers
  the screen a node's children at a time, a search or a type as flat matches with their trails.
- **A body's state follows the domain's own blocks**: text blocks `page_blocks_written` holds are
  ours, the others the client's, a shortcode or a block the parser marks dynamic (a textless figure
  is one) nobody's text: `empty`, `client`, `ours`, `mixed`, `dynamic` (`contextpack.BodyOf`).
- **One sentence explains a resolution** (`67d3a7f`): the agent's `resolution` moved into
  `findings.Explain` word for word, which also explains a run option and a set; a set is phrased
  "the N findings … on /a/, /b/, /c/ and N more".
- **A resolution writes through the page store** (`d3a46f9`): the title and H1 a plan takes from
  the site, a type and the parent source are written by `PageRepo`, the type checked against the
  site's own set before the unit, so `Resolve` and `ResolveMany` publish once, after the commit;
  `findings` no longer depends on `pages.Service`. A card's acceptance and a row's mapping keep
  their services, and publish as they did, never being bulk.
- **The bulk set is `finding.BulkOptions()`**, exported to `vocab.ts` as `findingBulkOptions`
  beside `findingOptionKinds`, `matchRules` and `bodyStates`, so the screens read the rule from Go.
- **The preview's pick** (`3942ff4`): an ambiguous row was only a warning; it is now a page of the
  preview with its candidates, and the pick happens where the row is matched (`address`), checked
  against the candidates that matching found. Apply sets the page's row key on the page it writes
  and calls `SetRowKey` last in the unit, which also takes the key off any other page.
- **The bindings**: a `FindingsService` with six methods before Settings, and `PagesService.SiteMap`
  through the renamed `SiteModelUseCase` (it was `PackUseCase`); 16 services.
- **The harness** seeds a duplicated review page and imports a small client workbook it keeps in
  its home (`074b20c`), so the inbox opens on a row two pages answer, a row no page answers and a
  sheet trail the site disagrees with; the seed's pins move by what it adds (45 entities, 49 edges,
  68 pages, 48 mapped).
- **`go mod tidy` names the tokenizer a direct requirement** (`d6e49e1`): `task build` tidies, and
  4.3a had left it marked indirect.

### What the agent decided (the screens)

- Block indexes are shown 0-based, as the Go summary says them; an HTML block is shown as its text.
- A seeded rewrite shows the drawer's rewrite switch on and ticked rather than opening its dialog,
  and the fields and the rewrite are sent only while the run's kind stays the seeded one.
- A code of more than 1,000 findings is resolved in batches of 1,000, one unit each; the Explain
  sentence names all of them, and the dialog confirms the count it collected.
- The import's picks are dropped when the file, the sheets or the columns change, since their row
  keys may not exist any more.
- `Dialog` gained a `refusal` (the confirm disabled with its reason); `useDebounced` moved to
  `ui/clock.ts`; Findings sits after Linking in the rail; `entity_card_proposed`'s accept opens the
  Graph screen's proposals for that entity, since the subject carries no proposal hash.

### What the gate found

- **`TestTheClientLoopFromTheSamples` had gone red with 4.1**, unseen because no stand ran the whole
  suite between stage 3 and now: the Broken sample sheet's "row that names nothing" carried a title,
  which since 4.1 plans a page at its slug and raises `row_without_page` instead of `no_target`. The
  row now carries a keyword alone (`b75063c`); the rule stands.
- **Everything else ran green on the first try**: the five race groups and `covergate`, the lint
  legs, the plugin on 7.4, the build with no drift, vitest and the harness, stage 4's exit tests and
  `e2e:full` on both stands from fresh resets, and `e2e:full:noplugin` on vanilla.
- **The census of the untouched stands**: the replica's 23 pages are typed Entity 5, Product 5,
  Technical 5, Hub 4, Store category 3, Catalog 1; 19 parents come from a trail (18 JSON-LD, 1
  ARIA), resolved by link 17, by redirect 1 and by name 1; its 703 links are navigation 580, listing
  92, breadcrumb 23, contextual 6 and unknown 2. Vanilla's 14 pages: Technical 5, Entity 4, Product 2,
  Store category 2, Catalog 1; 2 trails, by link; 148 links.

### Deviations from the request

- **A bulk over 1,000 findings is not one transaction**: each batch of 1,000 is.
- **The import preview's pick has no screen shot**: the harness keeps the workbook and its test
  previews and picks it, but a shot would need the file dialog.
- **`features/graph/screen.tsx` (551 lines before, 556 now) and `features/palette/palette.tsx` (344
  before, 332 now)** were over the 300-line limit before this stage and took small edits only.

## 2026-10-24 — phase 15, session 1: what must hold before the client gets it

The owner has no funded OpenAI key, so the release goes out without the smoke test `STATUS.md` had
first in line. This session closes what would hurt the client on day one. The plan is
`docs/superpowers/plans/2026-10-24-phase-15-release.md`, the first file of a phase allowed by spec
§10. A Fable critic read it before the owner (one blocking, seven serious and five minor findings, all
folded in). Agent P built Part 1, agent F the screens, agent S Part 4.

### The owner's answers at the plan

- **The run start's picker reads the Site map's tree**, the one tree of the 4.3b ruling.
- **Accepting a site proposal approves its parent edge**, and the entity is scoped under that
  parent at once, so "Liquid" under two hubs is two entities rather than a refused clash.
- **The old-WordPress stand stays** as `STAND=legacy` (8086, the contract suite only, never part of
  `e2e:full`).

### Part 1: the plugin's floors (it amends 2026-10-12)

- **PHP 7.2, with nothing rewritten**: the sources use nothing newer than 7.1 (`void`, nullable
  types) and declare no class, so no dynamic property can exist. `php -l` cannot see a newer
  *function*, so `wp-plugin/tests/functions.php` tokenizes every plugin file: each called function
  is the plugin's own, on the kept list `wp-plugin/testdata/wordpress.json` (WordPress, WooCommerce
  and SEO names with their `@since`), or present in the PHP running it. Every hook and every
  callback named by a string is on the list too, and a stale entry fails. The scan asserts that the
  list's newest core `@since` is the header's `Requires at least`. The test scripts run under
  `E_ALL`, and any notice, warning or deprecation fails them.
- **`task plugin:lint` and `plugin:test` run on `php:7.2.34-cli`, `7.4.33-cli` and `8.5.12-cli`**
  (compose services `php72`, `php74`, `php85`; `phplint` is gone). The brief named "a current 8.x",
  and the plan's 8.4 became 8.5, released in November 2025. CI runs a setup-php matrix over the
  three, and a separate job zips the plugin once.
- **WordPress 5.7.0** is the floor, set by the `wp_robots` filter and `wp_robots_no_robots` (both
  5.7.0 on developer.wordpress.org). The application's login needs core application passwords
  (5.6). No shim was added to go lower. Two kept `@since` values were corrected against the source:
  `wp_head` 1.5.0, and `wp_update_post` with `$wp_error` 3.5.0.
- **Version 1.5.2, with 1.5.1 still accepted**: the header, `VERSION`, `openapi.yaml`,
  `wp.PluginVersion` and the fake's default are 1.5.2. `Outdated` compares the new
  `wp.PluginMinimum` 1.5.1, so no site on 1.5.1 is told to upgrade. No route or contract moved, and
  `TestTheBundledPluginIsPluginVersion` ties the PHP header to the Go constant.
- **The legacy stand ran the contract suite once.** WordPress 5.7.2 runs on PHP 7.3, the oldest
  image that exists for it (no php7.2 tag exists past 5.6.0), with WP-CLI 2.6.0, MariaDB 11.4.12,
  WooCommerce 6.5.2, Rank Math 1.0.98.1 (Yoast 18.3 under `E2E_SEO=yoast`) and the companion 1.5.2.
  The research was wrong three times: WP-CLI 2.5.0 has no `application-password` command, old Rank
  Math zips are still served, and 18.3 is Yoast's last release for 5.6 and 5.7.
  - **The result:** `internal/adapters/wp/e2e`, 62 tests, 60 passed and none failed.
  - **The skips**, each naming what the stand lacks: core REST has no `show_on_front` before
    WordPress 6.0; a variation names no `parent_id` before WooCommerce 8.3; wc/v3 leaves out the
    cart, checkout and account pages before WooCommerce 6.9; and the existing Rank Math skip of the
    escaped-head test.
  - **What that means without the plugin:** on such a site the application cannot learn the front
    page or the store's pages; with the plugin, its `/settings` route answers both
    (`TestSettingsNameTheTechnicalPages` passed). Nothing reads a variation's parent today.
  - **Found and fixed:** `docker/e2e/import-products.php` counted a value WooCommerce 6.5 does not
    return, a warning on PHP 7.3 and a TypeError on PHP 8.
  - **Not exercised:** Yoast 18.3; the run used Rank Math. The log is in the session's scratchpad,
    `legacy-contract.log`.
- **The 2026-10-12 ruling is amended**: the plugin targets PHP 7.2 and WordPress 5.7, and its
  lint runs on the floor and on a current 8.x.

### Part 2: pages without an entity become writable through proposals

- **`page_without_entity`** is a warning per page. Its subject is the path alone, so a retitle
  leaves a dismissed one dismissed. It offers `propose_entity`, of a new option kind `proposal` that
  is taken in bulk, and `ignore`.
  - **Raised** by the derive stage, after `untyped()` has typed the walk, for every live,
    unarchived, non-technical page but the root whose effective type's preset is writable
    (`pagetype.Set.NeedsEntity`).
  - **Cleared** by the derive stage once an entity carries the page, and in the same unit by
    every writer that maps one (`ApplyProposals`, `pages.MapToEntity`, `SetCanonical`, the import's
    apply), resolved by the new resolver `model` with the resolution `page_mapped`. Settled counts
    only a person and the agent, so an unmap opens it again at the next walk.
- **The site proposal**: `contextpack.Map.EntityProposals` is pure, over the folded map and the
  graph.
  - **Name**: the H1, else the title.
  - **Parent**: the nearest ancestor that carries an entity or is proposed in the same batch. A
    product's category is folded into its twin hub, so it finds the hub.
  - **Facts**: a product's store attributes become its card's facts.
  - **Order**: by depth.
  - **Clashes** are `no_name`, `name_in_batch` and `name_carried`. Each is marked, left unticked
    and skipped, never refusing the batch.
  - **Shared names** are labelled through `graph.Labels` over the graph and the batch together.
  - **The API**: `PreviewFromPages` takes `source: site` (no model call; the default stays `model`).
- **Apply, for a site proposal**:
  - it runs in parent order;
  - it inserts the entity scoped under its parent with the parent edge approved, and the entity is
    the person's (`SourceUser`);
  - it reuses an entity of the same name under the same parent (by `graph.Key`), never one matched
    by a bare label;
  - it writes the card's facts, refusing a fact without a label or a value;
  - it maps the page and sets the canonical page.

  Model proposals keep their proposed edges.
- **The person's path**: `Resolve` and `ResolveMany` with `propose_entity` answer `proposals`, the
  site preview's request, before any unit, and settle nothing, as a run option answers a
  `StartRequest`. The inbox opens the Graph screen's proposal drawer at its preview, in place (a seed
  can hold thousands of pages). The person ticks all or some and confirms once.
- **The agent**: the `findings` tool reads a proposal option, writing nothing. `graph_apply_proposals`
  and `graph_propose_from_pages` gain a summary, so both wait for the person in either mode. The
  summary is built from `ApplyProposalsRequest.Validate`, which the use case calls too, so the
  summary refuses exactly what the use case refuses. The schemas grow from 65,722 bytes and 14,623
  tokens to 66,394 and 14,779 (ceilings 66,407 and 14,782). `vocab.ts` gains `proposalSources` and
  `proposalClashes`.
- **The proof on the replica** (`TestAProductWithoutAnEntityIsWrittenThroughAProposal`, green on its
  first run):
  - A product planted in the Healing Peptides category is refused by the estimate.
  - Its finding's proposal is named after it, carries its Size attribute as a fact, and is scoped
    under the category's twin hub `/healing-peptides/`.
  - Its run writes the description through the plugin.
  - The only store write is `POST /wc/v3/products/{id}` with `short_description`.
  - The store's facts are unchanged.

### Part 3: the input side of the estimate, adopted

- **The counter**: `llm.Tokens` reads 1.7 tokens a word or 3.7 runes a token (it read 2.2 and 3);
  `llm.ReadTokens` prices words a model reads at the same rate. The output side is kept
  conservative on purpose: `llm.WordTokens` 2.2 and `steps.tokensPerWord` 3, since how long an
  answer runs cannot be counted offline.
- **Each step's own overhead**, measured: the writer 1,040 beside its pack, the meta writer 440, a
  linker call 290, and the judge 510 beside its pack and the body it reads, which it now prices at
  `llm.ReadTokens` over the template's words. `inputShareOfOutput` is gone: the "rest" of a prompt is
  a near-constant 474 to 546 tokens, not a share of the answer. An unmeasured step keeps the 800
  default.
- **`Fit` keeps a twentieth of its budget as headroom**, so the counter, now near the real count,
  never lets a kept pack run over: before the headroom one pack came 2 tokens over a budget of 500.
- **The report's judge reads a stand-in body**, the captured bodies' own HTML blocks cut to the
  template's target words, because a run judges the body its writer has just written. A plain-text
  stand-in priced the judge 27% high, against HTML at 1.59 tokens a word. The meta writer is measured
  over a stand-in draft of the template's headings.
- **The proofs** in `cmd/tokenreport`, over nine pages of both captures:
  - every measured input is within ±25% of its real count;
  - at budgets of 4,096, 3,000, 1,500, 1,000, 750, 500 and 300, `Fit` refuses no pack whose whole
    real count fits and keeps none over its budget.

| step | estimate ÷ real before (pooled, range) | estimate ÷ real after |
|---|---|---|
| writer | 2.28 (2.08 to 2.52) | 1.04 (0.97 to 1.08) |
| linker call | 3.57 (3.49 to 3.64) | 1.11 for the hubs; 0.84 for the product's |
| judge | 2.07 over the captured body, with no body term | 1.07 (1.05 to 1.09) over a body of the template's length |
| meta writer | not measured | 1.07 (1.01 to 1.13) |

- **What stays uncertain**: the sample is WordPress's own boilerplate and nine pages, not the
  client's copy. The real input has not been compared with an OpenAI bill: the ledger's
  `input_tokens` from a funded key settles both sides. OpenAI renders a schema in its own format
  before the model reads it.

### Part 5: the prompts, for the owner to read

- `go run ./cmd/tokenreport -prompts out/prompts` writes ten plain-text files; `out/` is ignored.
  - **Nine step files**: the body writer, the meta writer and the judge on the replica's
    `/peptides/`, `/tb-500/` and `/product/bpc-157-peptide/`. Each holds the system and user
    messages and the strict answer schema.
  - **`agent-first-round.txt`**: the instructions with the replica's brief, the developer message,
    a question, and every tool as sent.
- **Headers**: each file opens with its page, step, role, model, effort, tier, the most output
  tokens and the commit, read from `.git`, so a rerun on one commit writes the same bytes. The meta
  writer's and the judge's files say they read a stand-in.

### The replica recaptured

- **Why**: the brief asks for the product Part 2 gave an entity, and every e2e test opens a database
  of its own. So the capture accepts the site proposal for `/product/bpc-157-peptide/` itself
  (`writerStand.product`), from a fresh stand.
- **The format of the rows moved** with migrations 0068 to 0072, since the last capture was 2b's.
- **The content moved by three things**: one entity (BPC-157 Peptide, under Healing Peptides, with
  its Strength fact), its edge, and the open `page_without_entity` findings.
- **The packs golden moved by that product alone**: its entity and its link up to
  `/healing-peptides/`, and the hub's link down to it.
- **Two things followed from the new capture**:
  - its types carry their templates (migration 0066), which the report now maps to the preset's
    built-in;
  - the replica's brief grew, so `instructionsCeilingTokens` is 1,932 for 1,929.

### Part 4: scale on a real SQLite database (agent S)

- **Counting**: `sqlite.Config.Trace` takes an interface the store declares (`Statement`, `Row`),
  hooked through the driver's `Conn.Trace` on every writer and reader connection.
  `sqlitetest.Counter` and `OpenCounted` serve the tests.
  - **Blind spots**: the driver traces no statement run without arguments (`BEGIN`, `COMMIT`, DDL;
    every repository statement has arguments).
  - **Warm-up**: each new connection runs one `PRAGMA query_only`, so a measurement runs once
    before it counts.
- **The generator** (`internal/app/scale`, tests only) is seeded and writes 40 pages per block in
  one unit, settled by one walk:
  - hubs four levels deep, and two store categories folded into twin hubs;
  - 12 products with 2 variations each, posts, and an archived page;
  - two planned pages per block from a CSV import through `Imports.Apply`;
  - every link type and region, fields, bodies, renders, and findings of six codes.

  It runs at 2,003 and 5,003 pages, on the real core over a plain store file.
- **The map is materialized** (the critic's blocking finding: a recursive query keeps statements
  and rows flat while its work grows with the site).
  - **What is stored**: `page_map` (migration 0074) holds a row per page: its folded parent, depth,
    a root-first trail of ids, children, descendants, twin, hub, whether it is a node, its type,
    its shown title and its search keys. `contextpack.Map.Rows()` computes them, and
    `sitemodel.Refresh` writes only the rows that moved.
  - **How it stays fresh**: triggers bump `page_map_sites.generation` when a page is inserted or
    deleted, when its parent, status, path, title, H1, WordPress type, page type or technical
    reason changes, when its title or H1 field changes, or when a page type changes. A read finds
    `built < generation` inside its read transaction and refreshes once, reading outside the unit
    and writing inside it after checking again. The derive stage refreshes at the end of every walk.
  - **On upgrade**: 0074 seeds every existing site stale.
  - **Proof**: a differential test compares the stored map with `contextpack.Map`, node for node,
    after an import, a derive, an archive, a parent pin, a map-to-page, a page created and deleted,
    and a retitle that breaks then remakes a twin.
  - **Footgun**: a later migration that rebuilds `pages`, `page_fields` or `page_types` must
    recreate these triggers.
- **What reads it**:
  - **`SiteMap`**: `page_map` by parent with a keyset on path. For the page's nodes only, it reads
    their pages, types, open-finding counts, contextual links in and out (the audit's meaning),
    bodies and written hashes. Search and type filters run in SQL.
  - **The findings list**: it filters in SQL, `under` being a trail prefix and the page type
    joined. Get, Resolve, Explain and ResolveMany read only the pages they name.
  - **The picker's API**: `MapNode` gains `entityId` and `status`, and `RunsService.Parents`
    answers the parents a selection writes first.
  - **The estimate**: a context memo reads the site once per estimate.
  - **The observation**: the render and fields stages read the model once per stage, and the
    render stage reads when every body was observed in one query.
- **The plans**: every distinct traced statement passes `EXPLAIN QUERY PLAN` without a scan of a
  large table, or a paged read that sorts every match, unless listed as a whole-site read by design.
  The indexes are `page_map_children`, `page_map_trail`, `page_map_nodes` and `findings_site_code`;
  without the last, the code filter scans.

| operation (2,003 / 5,003 pages) | statements before → after | rows before → after | time before → after |
|---|---|---|---|
| a Site map page (7 variants) | 23 → 10 to 13, both sizes | 69.5k / 174k → 20 to 715, both | 1.6–2.9 s / 6–25 s → 3–100 ms / 6–142 ms |
| the run start's picker | `Pages.Tree` 3 → a level 12, `Parents` 15 | 2,054 / 5,129 → 61 and 14 | 0.1 / 0.9 s → 7–10 ms |
| the findings list (5 filters) | 12 → 6 to 8 | 28.9k / 72.2k → 46 to 79 | 0.6–2 s / 2.4–20 s → 3–6 / 4–21 ms |
| a writer's pack | 19 → 19 | a whole-site snapshot, by design | 0.8 / 0.7 s → 0.7 / 4.6 s |
| a run's estimate (5 targets) | 486 → 292 | 449k / 1.12M → 50.3k / 124k | 10 / 15 s → 1.4 / 6.5 s |
| an incremental walk | 1,925 / 4,700 fixed + 46.5 a page → 91 fixed + 58.9 a page, both sizes | | |

  The timings swung two to six times between runs, because other work held the machine's CPUs.
  They are logged and never asserted. A touched page costs about 12 statements more than before,
  for the staleness triggers and the map's refresh.
- **Whole-site reads kept by design**:
  - the writer's pack, which reads a snapshot once per pack and once per estimate;
  - `graphPreflight`, which reads entities, edges and pages once per estimate;
  - a refresh after a stale mark;
  - a map search or type filter, which walks the site's map rows in path order until a page fills;
  - findings under a node or of a type, which sort that subtree's or type's findings before paging;
  - the engine's sweep of pending items;
  - in the observation, the content and store stages, which still read all pages and the open
    findings of their codes every batch, since a cached row would let `Pages.Update` overwrite an
    edit made during the stage;
  - `SiteModel.Map` for the site proposals, `Pages.Tree`, the link audit, the brief and explore.
- **The scale tests skip under `-race`** (`//go:build !race`): under the detector the package would
  run past the default ten minutes, and it measures counts the detector does not change. The gate
  runs them without race.

### Deviations from the request

- **The legacy stand runs PHP 7.3, not 7.2**: no WordPress 5.7 image exists on 7.2. PHP 7.2 is
  proven by the lint, the tests and the function scan on `php:7.2.34-cli`.
- **The current 8.x is 8.5**, not the plan's 8.4.
- **The sync's counts are a fixed part and a per-page part**, each equal at both sizes. A raw count
  cannot be equal when 5% of the site grows with it.
- **`Pages.Tree` stays**: the Overview and the Reports read it.
- **"Select the whole branch" left the run start's picker.**
- **Only the two proposal writers wait in either mode**, not every tool that maps a page.
- **The judge is measured over a stand-in body.**
- **The gate ran in two sittings**: the memory guard stopped the application race group and both
  stands' `e2e:full`, which then ran green on the owner's word, in foreground chunks.
- **The gate found one red**, in the replica's full suite: the proposal proof's run had relinked
  the client's hub `/healing-peptides/` to its planted product. That sentence outlived the deleted
  product and failed the writer's picture. The proof now reverts its run and asserts the hub is back
  byte for byte (`feaf2d6`).

## 2026-10-24 — phase 15, session 2: the polish, the docs, the build and the pull request

The owner asks for the release. This session polishes three things the client would meet, brings
the docs to the release, builds 2.5.0 and opens the pull request from `dev` to `master`, which the
owner merges and tags. The task list is under session 2 in the phase plan.

### Bulk resolution: batches of 1,000, and a partial result said in plain words

The brief allowed either one transaction per confirmed action or a partial result reported to the
person, chosen by what SQLite allows at that size. It is the partial result (`b550336`).

- **Measured** on the 5,003-page generated store (`internal/app/scale`, a probe not kept), with a
  second writer timing its own wait:
  - dismissing findings holds SQLite's one writer about 0.23 ms a finding: 248 ms for 1,000 and
    941 ms for 4,131;
  - accepting the site's value holds it about 0.8 ms a finding: 97 ms for 125.
- **Why not one transaction**:
  - a code of 10,000 findings in one unit would hold the writer for 2.3 to 8 s;
  - that is past the 5 s `busy_timeout` every other writer waits: the observation, the engine and
    every screen's write would fail busy;
  - a unit also refuses whole on one finding it cannot settle, so one stale finding would settle
    none of thousands. The probe met exactly that: an `accept_site` over 5,000 was refused by one
    page with no observed value.
- **What a batch is**: `ResolveMany` keeps its limit of 1,000, each call one unit, and the inbox sends
  the batches in turn (`settleInBatches`).
- **What a refusal does**:
  - the first refused batch stops the action, and the later batches are never sent;
  - the dialog stays open and says how many were settled, how many were not, and the refusal's own
    words: "1000 of 2500 findings were settled; the other 1500 were not: the finding is no longer
    open. Confirm again to settle the ones still open.";
  - an internal failure is said in the app's own sentence, never in a driver's words;
  - a refusal before anything was settled is shown as before.
- **What it amends**: 4.3b's "A bulk over 1,000 findings is not one transaction" stands, and the
  person is now told what it left.

## 2026-10-25 — phase 16: 2.5.1, the client's first feedback

2.5.0 was released and the client came back with four things: pages with no H1 after a sync, an
agent chat outside the selected site, a Proposed Entities dialog that forgot his selection, and the
three layout faults the release had listed. The task list is in
`docs/superpowers/plans/2026-10-25-phase-16-patch.md`.

### The H1 a visitor sees

- **Where it was lost.**
  - The `h1` field was written only from the content's first `<h1>`: the plugin's `h1` from its
    filtered content, or core's content through `headingOne`.
  - On Blocksy (the client's theme, as on the replica) and on vanilla's block theme, the theme
    prints the title as the page's H1, outside the content root. So the field stayed empty: 20 of 20
    published pages on the replica and 11 of 11 on vanilla, reproduced on 2.5.0's code before the
    fix.
  - Stage 3's extraction saw that H1 (`ThemeH1Printed`, 0071) but kept only the flag, never the text.
  - The stored extraction keeps the headings inside the root, not the public HTML, so no stored
    render could give a theme's H1 back.
- **The visible H1.**
  - The extraction records `H1` and `H1Source`: the first visible `<h1>` in document order outside
    the site's banner, navigation, sidebars and footer. A `<header>` or `<footer>` inside an article,
    a section or `main` belongs to the page.
  - The source is `content` when the h1 sits inside the content root and the client's own content
    (raw or filtered) holds an h1 of that text, and `theme` otherwise.
  - A text test was tried first and dropped: a one-word heading such as "Cart" matched loose words
    of the raw.
  - An empty render shows `none`. A visit that landed elsewhere keeps the source unknown.
  - It does not depend on the root being located, since a visitor sees the h1 either way; the plan's
    "unknown when the root was not located" was dropped.
- **One rule, one value.**
  - `render.VisibleH1(record, contentH1)`: a render with a known source wins; otherwise the content's
    H1 is used as `content`; failing both, `none`. Never a guess.
  - The content stage applies the rule with the page's stored render before `merge` and `fields`.
    The render stage applies it after each extraction and writes the field itself. So no walk
    flips the field between the two stages.
- **The planned H1 stays apart.**
  - `pages.h1` is both the workbook's H1 and a mirror of the observed one, and `hasPlan` reads
    "planned" from the column differing from the field.
  - So the render stage moves the column only through `PageRepo.FollowObservedH1(id, was, now)`,
    which writes where the column still equals the previous observed value. A planned H1 is never
    touched, and `page_h1_differs` tells the two apart on the next walk.
- **A read-only field is always agreed.** The H1 is the catalog's one field with `Write: None` that
  is not denied, and `observeFields` raised `field_changed_on_site` on it whenever it moved. It now
  agrees it, as `CLAUDE.md` already stated the rule.
- **Pages synced on 2.5.0.**
  - 0075 adds `page_renders.h1` and `h1_source` and fills every render whose theme prints no H1 from
    its stored headings (`content` or `none`).
  - A render whose theme printed an H1, or whose root was unknown, keeps an unknown source. The
    render stage's `stale()` fetches those again on the next ordinary sync, and only those: their
    text was never stored. Renders that landed elsewhere are excluded.
  - Proven by the migration test (0074 to 0075 with the backfill and the down) and by an
    observation test whose incremental walk visits only the themed page, then nothing on the walk
    after.
  - On both stands, the e2e test takes the themed pages' rows back to what 2.5.0 stored, through the
    repositories. The store exposes no migration control, and adding one for a test was not worth a
    production API. An ordinary sync then fills every one of them again.
- **What reads it.**
  - `pages.View` gains `observedH1` and `h1Source` beside the planned `h1`, read in one pass per
    call through two consumer-declared readers (fields and renders by page ids).
  - The Pages list shows the H1 after the title where the two differ, with the H1 and its source on
    hover. The drawer labels the editable field "Planned H1" and says under it what the site shows
    and who prints it.
  - The pack, `MatchRow`, crumb names and the agent's page view already read the field first. The
    stand census proves each with the fixed value: a row naming a page's H1 finds it, or a store
    category's hub twin, by `h1`.
- **Two existing tests changed their fixtures**, not their claims:
  - `TestACrumbFindsAPageByTheH1TheSiteShows` served a page whose render showed another H1 than the
    content's, and now serves the H1 it is about;
  - the observation's H1 tests strip the title from the filtered content, as Blocksy's
    `the_content` never carries it.
- **The fake site agreed with itself again.**
  - `wptest`'s plugin reports an item's `H1` (the harness hands it the workbook's), while its
    storefront printed the title. So the hub's visible H1 lost the name its store category shares,
    and the Site map's twin went nil (`TestTheSeededWorkbookFillsTheInboxTheMapAndThePreview`).
  - The storefront now prints the H1 the fake reports. The real plugin reports the first h1 of the
    rendered content, so a real site cannot disagree that way.

### One chat, in the selected site

- **Where it went wrong.** The rail's Agent opened `/agent`, a screen that listed every
  conversation and always started one with no site. A site-less conversation's site tools refuse
  every call (`requireSite`), so the agent could not use the site's tools and its brief was empty.
- **Now:**
  - with a site selected, the rail and the palette open `/s/:siteId/agent` (and the conversation
    and inbox paths under it);
  - that screen lists only the site's conversations, starts new ones on the site and names the site
    in its header ("Agent on …");
  - it opens the conversation the dock holds for that site and writes back the one it opens, so the
    two always show the same chat;
  - the dock hides on agent routes, so one chat is never on screen twice;
  - switching site from an agent route lands on the new site's chat.
- **`/agent` holds only conversations with no site**, says plainly that site tools need a site,
  and offers one button per site. The new-conversation dialog that let a person pick another site
  is gone. No Go change: a conversation's tools already bind its site.
- **Proven** by vitest on the paths, the screen and the rail, by
  `TestTheScriptAnswersASiteQuestionAndAWriteWithTheirTools` (the harness's script answers
  "explore" with `sites_explore` and a retitle with a confirmable write), and by walk shots from the
  rail and from the dock that show the tool's outcome and the card.
- **The harness seeds a second site** ("Tamper Supply") to prove the switch. Two sites created in
  the same second list in either order, since sites sort by creation and then by a random id. So the
  harness's restore and its tests find the seeded site by name, and the tests list by name. Before
  that, the restart tests failed now and then with "the seed published nothing".

### The Proposed Entities dialog

- **The brief kept its redesign for a later session.** The owner opened it mid-session, found it
  unchanged, small, and unable to untick a parent's children at once, and moved the whole redesign,
  editing included, into 2.5.1.
- **What it is now:**
  - a sheet over nearly the whole window (`ui/workspace.tsx`);
  - the proposals as a tree by their parent, with expand and collapse per node and for all;
  - ticking a node ticks its whole subtree, and a parent with some children ticked shows a partial
    state; keep all and keep none; a count of ticked out of total;
  - filters (all, ticked, not ticked, blocked) and a search by name or path that keeps the ancestors
    of every match;
  - an inspector that edits a proposal's name, its parent (another proposal, an existing entity or
    the top level, never a cycle) and its keywords, with "Undo my changes";
  - clashes recomputed after an edit, so a name taken under the same parent stays unticked with its
    reason.
- **A ticked child whose parent is unticked is sent at the top level**, and its row says so, which
  is what `ApplyProposals` does with a parent it cannot find. `ApplyProposals` takes the proposals
  as the screen holds them, so no Go change was needed.
- **Its state** (stage, selection, edits, search, filters, expanded nodes) lives in memory per site
  and per opener (pages, keywords, a findings seed; `proposal-memory.ts`). It clears on accept, on
  "Start over", on a different findings seed and on restart. The pick's page tree takes its search
  and expanded nodes from outside for it.
- **Proven** by 43 tests of the pure model (`graph/model/proposal-tree.ts`), the store's tests, 13
  new drawer tests, and walk shots at 1280 and 960. The walk builds a selection, goes to Pages,
  switches site and back, and finds it kept. The review shots open the dialog from Findings, since
  the harness's fake model cannot answer a preview from pages.

### Three screens at 960 × 600 with the dock open

- One pure helper, `ui/collapse.ts`, picks the columns that fit the measured width, as the Site map
  does.
- The Runs table drops the duration, then the spend, then the start, all kept in the run's detail.
  The Models spend and recent-call tables collapse the same way, and model and role names show whole
  on hover.
- The rail scrolls without a sideways bar.
- `runs-960`, `settings-models-960` and `rail-site-960` carry a `fits` step, and the walk gained an
  `expect` step.
