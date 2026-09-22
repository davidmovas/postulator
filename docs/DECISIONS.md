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
  `internal/domain/pagemap`.

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
