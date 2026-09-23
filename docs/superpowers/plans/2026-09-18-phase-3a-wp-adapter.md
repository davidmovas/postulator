# Phase 3A — WordPress Go Adapter and Fake Server Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. This repository forbids sub-agents (spec §10), so the subagent-driven variant does not apply.

**Goal:** Land `internal/adapters/wp` — a stdlib-only WordPress REST client covering core `wp/v2`, WooCommerce `wc/v3` and our companion plugin's `postulator/v1` namespace — together with `internal/adapters/wp/wptest`, the in-memory fake every later phase tests against, and `wp-plugin/openapi.yaml`, the frozen contract track B implements in PHP.

**Architecture:** One `Client` per site. `New(Config, ...Option)` validates the base URL, resolves options into one `*http.Transport` (proxy, TLS, dial timeouts) and two `*http.Client`s over it — one that follows redirects for ordinary calls, one that refuses them for `Probe`. Every call goes through a single `do` pipeline: rate limiter, retry loop, one classified HTTP round trip, one redaction-safe log line. Transport and WordPress failures become `*kernel/errors.Error` at that pipeline and nowhere else. `wptest` is an `httptest` server that speaks WordPress's wire format from its own in-memory state and deliberately shares no type, no JSON struct and no hash function with the client, so a decoding bug cannot cancel itself out.

**Tech Stack:** Go 1.27 (CGO off), `net/http`, `golang.org/x/net/proxy` (SOCKS5), `golang.org/x/time/rate`, `go.uber.org/zap` `v1.27.0`, `net/http/httptest`, golangci-lint `v2.13.2`, Task `v3.53.1`.

**Spec:** `docs/superpowers/specs/2026-09-17-postulator-v2-design.md` (§1 context, §3 layout, §4 kernel, §9.3 WordPress, §15 reference map) and the Phase 3 row of `docs/superpowers/plans/2026-09-17-postulator-v2-roadmap.md`.

## Global Constraints

- **No comments in Go, TypeScript, YAML or PHP** — not inline, not godoc, not package docs. `openapi.yaml` carries `description:` fields, which are document data, not comments.
- **No stubs, no TODOs, no placeholders.** If something cannot be done, stop and say so.
- **Interfaces are declared by the consumer**, never by the implementation and never in advance. `wp` declares the `Logger` it consumes; nothing declares an interface for `*wp.Client`.
- **TDD.** Failing test first, run it, implement, run it, commit. Table tests with named cases. Gates: `internal/domain` + `internal/application` ≥ 80%, module ≥ 70%, `internal/kernel` ≥ 90%; this phase additionally requires ≥ 85% for `internal/adapters/wp` and `internal/adapters/wp/wptest`.
- **Cursor pagination only** on our own boundaries. WordPress core REST is offset-based and is not our boundary; see decision D3.
- **JSON is camelCase** on our own boundaries. WordPress core and WooCommerce speak `snake_case` and that is their wire format, not ours; the plugin namespace is ours and is camelCase.
- **Errors crossing a boundary are `*kernel/errors.Error`** with a frozen code: `NOT_FOUND CONFLICT INVALID UNAUTHORIZED RATE_LIMITED BUDGET_EXCEEDED EXTERNAL INTERNAL CANCELLED NEEDS_HUMAN LOCKED`.
- **Secrets never reach a log, an artifact or a plain column.** The application password is never a log field, never an error detail and never part of a URL.
- **No new `.md` files** beyond `CLAUDE.md` and the set in spec §10. This plan is one of the allowed `docs/superpowers/plans/` entries.
- **Dependency rule**, enforced by `internal/app/deps_test.go`: adapters may import `application`, `domain`, `kernel` and third-party packages; only `kernel/paging` imports squirrel; only `internal/app` imports `internal/transport`. This track adds no domain and no application dependency.
- **Adapters never touch the secret store.** `Config.AppPassword` arrives already resolved; the caller fetches it through `application`'s `SecretStore` port (spec §9.2) and hands it over.
- **Commits** are conventional (`<type>(<scope>): <subject>`), on the `phase-3a` branch of the `post-creator-app-phase3a` worktree, never amended, rebased or force-pushed, and every commit ends with the trailer `Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>`.
- **Gate for every task** (Go-only changes): `go build ./...`, `go vet ./...`, `$(go env GOPATH)/bin/golangci-lint.exe run`, `go test -count=1 -race ./...`. The full phase gate adds `go test -race -covermode=atomic -coverprofile=coverage.out ./...` and `go run ./cmd/covergate -profile coverage.out`. `task build` is not run: this track touches no frontend file.
- Pinned: Go `1.27`, golangci-lint `v2.13.2`, Task `v3.53.1`, `golang.org/x/net v0.58.0`, `golang.org/x/time v0.15.0`.

---

## Design decisions

### D1 — Client construction and option order

`New(cfg Config, opts ...Option) (*Client, error)` runs in four fixed stages and nothing is observable until all four succeed.

1. **Validate `Config`.** The base URL must parse to an absolute URL with a host. `https` always passes. `http` passes only when `AllowInsecure` is true. Any other scheme is `Invalid`. Empty `Username` or `AppPassword` is `Invalid`: a WordPress application password is useless without its user, and failing here beats failing on the first 401.
2. **Resolve options into a value struct.** Options mutate an unexported `options` value seeded with the defaults, never the live client. Order therefore matters only between two options of the same kind, where last wins. `WithLogger(nil)` and `WithBackoff(nil)` are no-ops rather than ways to install a nil that panics on first use.
3. **Build one `*http.Transport`** from the resolved proxy string. Building it after option resolution is what makes the proxy option order-independent.
4. **Build two `*http.Client`s over that one transport.** `c.http` uses the stdlib redirect policy; `c.probe` sets `CheckRedirect` to `http.ErrUseLastResponse`. They share the transport, so the connection pool, the proxy and the TLS configuration are configured exactly once.

**There is no `WithHTTPClient` and no TLS option.** The transport is the place where the two security invariants live — verification on, proxy only from the configured setting — and handing a caller the transport hands them both invariants. Tests point `BaseURL` at `wptest` instead.

**Deviation from the task brief, recorded deliberately:** the no-redirect diagnostic mode is *not* an `Option`. Turning redirect-following off for ordinary calls would break every site that 301s a REST path to its trailing-slash form, and a caller has no reason to want that. It is `c.probe`, an internal second client that only `Probe` uses.

`TLSClientConfig` is never set anywhere in this package. `AllowInsecure` unlocks the `http` scheme and nothing else; it never becomes `InsecureSkipVerify`. A test asserts the transport's TLS configuration stays nil.

### D2 — Retry and rate limit layering

Outermost to innermost: **retry loop → backoff sleep → rate limiter → one HTTP round trip.**

The limiter is inside the retry loop because a retry is a new request against the same site and must pay the same per-site budget. The backoff sleep happens *before* `limiter.Wait`, not after, so a granted token is never held idle for the length of the backoff.

Retryable is decided on the kernel code, not on the status: `RateLimited` and `External` retry, everything else returns immediately. That keeps the decision in one place and means a transport failure and a 503 are handled by the same branch.

The delay for attempt *n* is the `Retry-After` the server sent, when it sent one, and otherwise `backoff(n-1)`. `Retry-After` is parsed as both delta-seconds and HTTP-date. The default backoff is `500ms << attempt` capped at 8s, with **no jitter**: this is a single desktop process talking to one site, so there is no herd to disperse, and a deterministic backoff is a testable backoff. `WithBackoff` is exported precisely so tests can pass a zero-delay function instead of the package hiding a private hook.

`wp.retries` counts *retries*, not attempts: the default 4 means at most 5 requests.

Rate limiting is per `Client`, and a `Client` is per site, so "per site" needs no extra bookkeeping. `WithRateLimit(0)` or a negative value means `rate.Inf`; tests use it so the suite does not sleep.

### D3 — How `ListItems` exposes pagination

`ListItems` takes `ListQuery{Page, PerPage}` and returns `ItemPage{Items, Total, TotalPages, Page, HasMore}` — **WordPress page numbers, not `kernel/paging.Cursor`.**

The repository rule is that *our* boundaries are keyset-paginated. WordPress core REST is `?page=&per_page=` over an offset query and exposes no keyset; there is nothing to build a cursor from. Minting one would be a lie in two directions: it would claim a stability the underlying `LIMIT/OFFSET` does not have, and it would hide the one signal that actually tells us we have reached the end — the `400 rest_post_invalid_page_number` WordPress returns for a page past the last one. Callers iterate `for page := 1; ; page++` until `HasMore` is false, and the sync use case (Phase 7) converts what it read into our own cursor-paginated repositories.

`ItemPage` and its siblings are aliases of one generic `Page[T]`, so categories, products and product categories do not each grow a near-identical struct.

The plugin namespace is the one WordPress surface that *does* have a keyset, and that is where an opaque cursor lives — see D4.

### D4 — The plugin cursor is opaque end to end

`ContentQuery.Cursor` and `ContentPage.NextCursor` are `string`. The Go client never decodes, validates or constructs one: it echoes what the last page returned and stops when `nextCursor` is empty.

The keyset `(modified, id)` lives in MySQL and only the plugin knows the exact `ORDER BY` it was issued for. Decoding it in Go would freeze the PHP query shape into the Go adapter and give us two implementations of one invariant, which is exactly the bug `kernel/paging` refuses to allow for our own cursors. Because the client cannot look inside, `wptest` is free to mint its own encoding (`base64url({"m":<modified>,"i":<id>})`), and the fact that its cursors work unchanged in the client's tests is itself the proof that the client treats them as opaque.

`since` is **exclusive** (`modified > since`), matching core REST's `modified_after`, so the sync use case stores the highest `modified` it saw and passes it straight back.

The end of the list is `"nextCursor": null`, with the key **always present**, so Go models it as `NextCursor *string`: absent-versus-null-versus-empty is exactly the distinction a `string` would flatten, and a plugin that stopped sending the key would then be indistinguishable from one that reached the end.

### D5 — `contentHash` is `sha256(raw post_content)` hex, computed identically in Go and PHP

`wp.ContentHash(raw string) string` is `hex(sha256([]byte(raw)))`. **No normalisation, no trimming, no newline conversion, no HTML parsing, no case folding.** The PHP side is `hash('sha256', $post->post_content)`. Both hash the same byte string, so they agree without either side knowing anything about the other, and `openapi.yaml` states the algorithm so track B cannot drift.

This is deliberately *not* `domain/content.Document.Hash()` from spec §6, which hashes *normalised* HTML for drift detection of our own rendering. The WordPress hash is a compare-and-swap token for `PUT /content/{id}/raw`: if it is anything other than byte-exact, two concurrent relinks can both believe they won.

`wptest` implements the hash a second time, in its own package, rather than importing `wp.ContentHash`. A shared helper would make the fake agree with the client by construction, which is worthless for the one property that matters. Both implementations are pinned against the same hard-coded digest literal, computed outside Go:

```
sha256("<p>Koffein ist ein Alkaloid.</p>") = 119b7cff7356b21d2b00e64d2d3c0589b50f3270b302a94360fac92f1adc332b
sha256("<p>Powder</p>")                    = 79db24b7a931978a1db05ccdb1ae1ed16b07aafa5277af75a7dc1b3b7ec6a509
```

`GetRaw` verifies the hash the site reported against the content it returned and reports `External` on a mismatch, so a divergence between Go and PHP surfaces as a clear error at the first read instead of as a silent conflict later.

### D6 — How `wptest` generates slugs, paths and hashes

- **Slug.** An absent slug is derived from the title: lowercase, every run of non-`[a-z0-9]` runes collapses to a single `-`, leading and trailing `-` trimmed, empty result becomes `item`. Uniqueness then appends `-2`, `-3`, … — **`-2` first, never `-1`**, which is what `wp_unique_post_slug` does and what makes a caller's requested slug come back changed. Scope is `(type, parent)` for hierarchical types (`page`, `product_cat`) and `(type)` for flat ones (`post`, `product`), again following WordPress.
- **Path.** Walked up the `parent` chain of slugs, capped at ten levels against a malformed cycle, rendered `/a/b/` with both slashes.
- **Hash.** `contentHash` over the stored `Content` exactly as D5 defines it.
- **Time.** The fake owns a deterministic clock starting at `2026-09-18T10:00:00Z` and advancing one second per mutation. `modified` ordering is therefore reproducible, which is what makes cursor and `since` assertions stable rather than timing-dependent.

### D7 — `wptest` shares nothing with `wp`

`wptest` does not import `wp`. It has its own `Item`, its own JSON shapes, its own slug rules and its own hash. The point of a fake is to be an independent second implementation of the contract; importing the client's types would let one bug satisfy both sides. `wptest`'s own tests drive it with plain `net/http`, proving it speaks WordPress before any client exists.

### D8 — Error mapping, and what a log line may contain

| WordPress | kernel | extra |
|---|---|---|
| 401, 403 | `Unauthorized` | — |
| 404 | `NotFound` | — |
| 409 | `Conflict` | `Details["currentHash"]` when the body carries one |
| 429 | `RateLimited` | `WithRetry(Retry-After)` |
| 400 | `Invalid` | `Details["code"]` = the WordPress error code |
| 5xx | `External` | `WithRetry(0)`, meaning "use our backoff" |
| other ≥ 300 | `External` | — |
| transport failure, caller's context alive | `External` | `WithRetry(0)` |
| transport failure, caller's context cancelled | `Cancelled` | — |

Every mapped error also carries `Details["status"]`, `Details["method"]`, `Details["path"]` and, when the body had one, `Details["wpMessage"]`. `Details["code"]` is always the far side's error code: WordPress's own on a 400, and our `plugin_missing` when the companion plugin is absent. One key, one meaning.

The distinction on the last two rows matters: a `context.Context` the caller cancelled must not be retried, because every retry will fail the same way, while our own `http.Client.Timeout` firing is an ordinary transient failure that should be.

A log line carries `method`, `path`, `status`, `durationMs`, the kernel `code` and `wpCode`. **Never the body, never the query string, never a header.** Response bodies routinely contain the whole rendered page, and headers contain `Authorization`. `kernel/log` redacts by field key, which protects nothing if a body is logged as one blob, so the adapter's rule is that the body simply never becomes a field.

### D9 — Capability caching and degradation

`Capabilities(ctx)` fetches `GET /manifest` once per client under a mutex and caches the result. A **404 is cached as absent**; a 5xx or a transport failure is **not**, so a site that was briefly down is re-probed on the next call instead of being written off for the lifetime of the client.

Once absent, every plugin method returns `Invalid` with `Details["code"] == "plugin_missing"` without issuing a request, so a caller can degrade to core REST on one cheap, classifiable error. Holding the mutex across the first network call serialises concurrent first callers, which is the intended behaviour: one manifest request, not N.

### D10 — Reads use `context=edit`, and writes re-read

Every core-REST read sends `context=edit`. Without it WordPress returns `title` and `content` only as `{"rendered": …}`, which is the theme's output rather than the stored post, and relink would then write rendered HTML back into `post_content`. With it the payload carries `{"raw": …, "rendered": …}` and the decoder prefers `raw`.

`CreateItem` issues the `POST` and then a `GET` of the created id, and returns the second response. WordPress rewrites a colliding slug server-side and computes `link` from the final permalink structure, and a caller that trusts what it asked for writes a broken internal link. The extra round trip is one GET per created page and buys a `slug` and `link` that are true.

### D11 — Tri-state fields are a `map[string]any`, not a struct with `omitempty`

`UpdateItem.Parent *int64` means: `nil` keep, `&0` move to the top level, `&id` reparent. No struct tag can express that — `omitempty` erases a legitimate `0` — so `UpdateItem.payload()` builds a `map[string]any` and adds exactly the keys the caller set. The same rule gives `Categories`/`Tags` their nil-versus-empty distinction: `nil` keeps what the post has, `[]int64{}` clears it. An update whose payload came out empty is `Invalid` rather than a pointless request.

### D12 — WooCommerce is a different API wearing the same auth

WooCommerce `wc/v3` accepts the same application password over https, and that is the whole of what it shares. `ListItems(TypeProduct, …)` therefore routes to `/wc/v3/products` and maps the WooCommerce payload into `Item`: `name → Title`, `description → Content`, `short_description → Excerpt`, `permalink → Link`, `date_modified_gmt → Modified`. Three concrete differences the query builder honours:

- WooCommerce's `status` takes **one** value, not a comma list; a comma list is a 400. `ListQuery.Status` contributes only its first entry on a WooCommerce route.
- `context=edit` is sent on core routes only; WooCommerce returns raw descriptions regardless.
- WooCommerce categories are written as `[{"id":12}]`, not as a bare id array.

`ListProducts`, `GetProduct`, `UpdateProduct` and `ListProductCategories` return the typed WooCommerce shapes for callers that need `short_description`; `ListItems` exists for the type-agnostic sync loop.

The generic **write** methods are core-only: `CreateItem`, `UpdateItem` and `DeleteItem` report `Invalid` for `product` and `product_cat` without issuing a request. Postulator never creates or deletes a shop's products — spec §9.3 gives WooCommerce read and update only — and a generic `UpdateItem` over a WooCommerce payload would silently send core field names that WooCommerce ignores. `UpdateProduct` is the one write path into the shop, and it names WooCommerce's own fields.

A WooCommerce product category is a taxonomy term: it carries no modification date, so `ListQuery.ModifiedAfter` is not sent for `product_cat` and an incremental sync always re-reads that collection in full.

### D13 — Settings live next to the consumer

`internal/adapters/wp/settings.go` declares `wp.timeout`, `wp.retries`, `wp.rateLimitPerSecond` and `wp.proxyUrl` against `kernel/settings`'s default registry, with validators, and exports `FromSettings(*settings.Values) []Option`. The composition root writes `wp.New(cfg, wp.FromSettings(values)...)`; a test writes `wp.New(cfg, wp.WithRateLimit(0))`. The mapping from setting to option lives in the package that reads the setting, and `New` itself stays free of any settings dependency.

Per-site overrides are **not** in this track. Site defaults belong to `domain/site` and arrive later as fields on `Config` or as extra options.

An empty `wp.proxyUrl` means **no proxy at all**, not `http.ProxyFromEnvironment`. A stale `HTTPS_PROXY` in the user's environment must not silently route WordPress credentials through a host the user never configured in the application.

### D14 — The plugin contract details agreed with track B

These nine points are binding on `wp-plugin/openapi.yaml`, on `wptest` and on the Go client at once. They are listed here because each is a place where two independent implementations would otherwise drift apart silently.

1. **`GET /content?limit=`** clamps rather than rejects: above 500 it becomes 500, absent or ≤ 0 it becomes 100. A sync loop that asks for too much gets a smaller page, never a failed run.
2. **`nextCursor` is always present**, `null` on the last page. Go models it as `*string`.
3. **`PUT /seo-meta/{id}`, `GET /content/{id}/raw` and `PUT /content/{id}/raw` are post-only.** A term id (a `product_cat`) is `404 {"code":"not_found"}`: terms have no `post_content` and no SEO post meta, so there is nothing for those routes to read or write.
4. **`path` always carries a leading and a trailing slash**, with no file-extension exception, duplicate slashes collapsed, query and fragment stripped, the whole path ASCII-lowercased, and **no percent-decoding** — the stored bytes are what a later comparison sees. D15 has the full rule list.
5. **Internal-link detection is host equality, exact and lowercase**: no `www` stripping, scheme ignored, and a relative href is internal. Go implements the same rule once, exported (D15).
6. **A term's `modified`** comes from term meta the plugin maintains, and reaches the client as an ordinary RFC3339 timestamp. This is the one place a `product_cat` has a modification date: the WooCommerce `wc/v3` route has none, which is why D12 says an incremental sync over core REST must re-read that collection in full.
7. **`title` is the raw `post_title`**, the raw routes carry the raw `post_content`, and `contentHash` is the lowercase hex sha256 of those raw `post_content` bytes (D5).
8. **Every error is `{"code","message"}`**; a 409 on the raw write adds a top-level `currentHash`, which maps to `Conflict` with `Details["currentHash"]`.
9. **Manifest `capabilities` for version 1.0.0 is exactly `["bulk","seo_meta","content_hash","raw"]`** and `seoPlugin` is one of `yoast`, `rankmath`, `none`.

### D15 — Internal link normalisation is one algorithm, exported

One algorithm, binding on `wp`, on `wptest`, on `internal/domain/pagemap` and on the PHP plugin at once:

1. The input may be an absolute URL or a path.
2. Internal-ness is decided first, by **exact lowercase host equality** — no `www` stripping, scheme ignored, and a relative href is always internal.
3. Scheme and host are then stripped.
4. Query and fragment are stripped.
5. **Nothing is percent-decoded.**
6. Duplicate slashes collapse.
7. Exactly one leading and one trailing slash; the root is `/`.
8. The whole path is **ASCII-lowercased**. Percent-escapes are ASCII, so `%C3%BC` becomes `%c3%bc` in Go and in PHP alike and the two never disagree.
9. No file-extension exception: `/a/b.html` becomes `/a/b.html/`.

`wp.NormalizePath(rawPath string) string` and `wp.InternalPath(siteHost, href string) (string, bool)` are exported from the adapter so that every consumer that may import an adapter applies these rules rather than reinventing them per step. `InternalPath` refuses a non-http scheme, reads the path through `url.URL.EscapedPath` so rule 5 holds, and hands the result to `NormalizePath`.

**This is temporary.** The canonical home is `internal/domain/pagemap.NormalizePath`, landing with Phase 2 in the main tree. Adapters may import domain, so once Phase 2 merges a follow-up task replaces the body of `wp.NormalizePath` with a call into `pagemap` and moves `link_test.go`'s `NormalizePath` table with it; `wp.InternalPath` keeps the host decision, which is adapter knowledge. Phase 6 then takes the normaliser straight from the domain and needs no injected function.

`wptest` still cannot use either (D7) and implements the same nine rules independently — which is exactly what makes the plugin-namespace tests meaningful.

### D16 — Time handling in the adapter

WordPress returns `modified_gmt` as `2026-09-18T10:00:00` — RFC3339 shape with **no offset** — so `time.Parse(time.RFC3339, …)` fails on it. `parseWPTime` tries RFC3339 first and falls back to `2006-01-02T15:04:05` read as UTC.

The reverse direction has a trap this adapter cannot fix: core REST filters `modified_after` on `post_modified`, which is **site-local**, while it returns `modified_gmt`. On a site whose timezone is not UTC, an incremental core-REST sync can miss or repeat items within the offset. The adapter sends exactly what it is given and records the hazard here; the plugin's `/content?since=` compares `post_modified_gmt` and is the accurate incremental path, which is one of the reasons the plugin exists. Compensating for the offset is the sync use case's decision, not the adapter's.

The adapter uses `time.Now()` for request latency and `time.Until` for an HTTP-date `Retry-After`. Both are properties of the transport, not of domain time, so `kernel/clock` does not apply.

## WordPress quirks this adapter must survive

Each of these cost the v1 codebase real debugging time (spec §1) and each has a named test.

1. A created page comes back with a **different slug** when one collided; WordPress appends `-2`.
2. A page number past the last page is a **400 `rest_post_invalid_page_number`**, not an empty list. It is the end-of-list signal, not an error.
3. `parent` is **tri-state** on update, and `0` is a meaningful value.
4. `modified_gmt` has **no timezone suffix**.
5. `meta` comes back as **`[]`**, not `{}`, when a post type has no registered meta — decoding it into a map fails.
6. Without `context=edit`, `title` and `content` are **rendered**, not raw.
7. A site that redirects `/wp-json` to `wp-login.php` is an **authorisation** problem, not a routing one.
8. Media upload does **not** accept `alt_text` in the raw upload; it needs a second request.
9. WooCommerce's `status` is single-valued where core's is a list.
10. A page past the end is a 400 in core REST and an **empty 200** in WooCommerce; both have to end the loop.

## File structure

**Create**

- `wp-plugin/openapi.yaml` — the frozen `postulator/v1` contract, OpenAPI 3.1.
- `internal/adapters/wp/client.go` — `Config`, `Client`, `New`, `baseURL`, the namespace constants, `resolve`.
- `internal/adapters/wp/option.go` — `Logger`, `Option`, `options`, the `With*` set, `defaultBackoff`, `newLimiter`.
- `internal/adapters/wp/transport.go` — `newTransport`, `socksTransport`, `socksAuth`.
- `internal/adapters/wp/settings.go` — the four settings and `FromSettings`.
- `internal/adapters/wp/do.go` — `request`, `attempt`, `do`, `send`, `pause`, `logAttempt`, `decodeJSON`.
- `internal/adapters/wp/errors.go` — `wpError`, `classify`, `transportError`, `retryAfter`, `retryable`, `delayFor`, `detailString`.
- `internal/adapters/wp/hash.go` — `ContentHash`.
- `internal/adapters/wp/probe.go` — `ProbeStatus`, `ProbeResult`, `Probe`, `classifyRedirect`.
- `internal/adapters/wp/item.go` — `ItemType`, `Item`, `Page[T]` and its aliases, `ListQuery`, `CreateItem`, `UpdateItem`, the wire payloads and decoders.
- `internal/adapters/wp/items.go` — `ListItems`, `GetItem`, `CreateItem`, `UpdateItem`, `DeleteItem`.
- `internal/adapters/wp/media.go` — `Media`, `MediaItem`, `UploadMedia`.
- `internal/adapters/wp/category.go` — `Category`, `CreateCategory`, `ListCategories`.
- `internal/adapters/wp/woo.go` — `Product`, `ProductCategory`, `UpdateProduct` and the four WooCommerce methods.
- `internal/adapters/wp/link.go` — `NormalizePath`, `InternalPath`.
- `internal/adapters/wp/plugin.go` — `Manifest`, `Capabilities`, `ContentQuery`, `ContentPage`, `ContentItem`, `SEOMeta`, `SEOResult`, `RawContent` and the six plugin methods.
- `internal/adapters/wp/wptest/server.go` — `Server`, `New`, the options, `URL`, routing, auth, recording.
- `internal/adapters/wp/wptest/state.go` — `Item`, `Category`, `record`, the store, `slugify`, `uniqueSlug`, `itemPath`, `contentHash`, the clock.
- `internal/adapters/wp/wptest/core.go` — `wp/v2` handlers.
- `internal/adapters/wp/wptest/woo.go` — `wc/v3` handlers.
- `internal/adapters/wp/wptest/plugin.go` — `postulator/v1` handlers and the SEO key table.
- `internal/adapters/wp/wptest/fault.go` — `FailNext`, `RateLimitNext`, `Redirect`.
- Tests: `contract_test.go`, `client_test.go`, `hash_test.go`, `errors_test.go`, `do_test.go`, `probe_test.go`, `items_test.go`, `write_test.go`, `media_test.go`, `woo_test.go`, `link_test.go`, `plugin_test.go`, `log_test.go` under `internal/adapters/wp`; `state_test.go`, `server_test.go`, `core_test.go`, `woo_test.go`, `media_test.go`, `plugin_test.go` under `wptest`.

**Modify**

- `go.mod`, `go.sum` — add `golang.org/x/net v0.58.0` and `golang.org/x/time v0.15.0`.
- `docs/STATUS.md` — Phase 3A decisions and state.
- `CLAUDE.md` — the standing rulings this phase produces.

---
### Task 1: Freeze the companion plugin contract

Track B implements PHP against this file, so it lands first and nothing in it changes without both tracks agreeing.

**Files:**
- Create: `wp-plugin/openapi.yaml`
- Test: `internal/adapters/wp/contract_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: the frozen `postulator/v1` wire contract — routes `GET /manifest`, `GET /content`, `PUT /seo-meta/{id}`, `GET /content/{id}/raw`, `PUT /content/{id}/raw`; the error body `{"code","message"}`; the conflict body `{"code":"hash_mismatch","message","currentHash"}`; the SEO meta key table for Yoast, Rank Math and no plugin.

- [x] **Step 1: Write the failing test**

Create `internal/adapters/wp/contract_test.go`:

```go
package wp_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

var httpMethods = []string{"get", "put", "post", "delete", "patch"}

func contractDocument(t *testing.T) string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "wp-plugin", "openapi.yaml"))
	if err != nil {
		t.Fatalf("read the plugin contract: %v", err)
	}
	return strings.ReplaceAll(string(raw), "\r\n", "\n")
}

func documentedRoutes(t *testing.T) map[string][]string {
	t.Helper()

	routes := make(map[string][]string)
	current := ""
	inPaths := false
	for _, line := range strings.Split(contractDocument(t), "\n") {
		switch {
		case line == "paths:":
			inPaths = true
		case inPaths && line != "" && !strings.HasPrefix(line, " "):
			inPaths = false
		case inPaths && strings.HasPrefix(line, "  /") && strings.HasSuffix(line, ":"):
			current = strings.TrimSuffix(strings.TrimSpace(line), ":")
			routes[current] = nil
		case inPaths && current != "" && strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "     "):
			name := strings.TrimSuffix(strings.TrimSpace(line), ":")
			if slices.Contains(httpMethods, name) {
				routes[current] = append(routes[current], name)
			}
		}
	}
	return routes
}

func TestThePluginContractDocumentsEveryRoute(t *testing.T) {
	t.Parallel()

	want := map[string][]string{
		"/manifest":         {"get"},
		"/content":          {"get"},
		"/seo-meta/{id}":    {"put"},
		"/content/{id}/raw": {"get", "put"},
	}

	routes := documentedRoutes(t)
	if len(routes) != len(want) {
		t.Fatalf("the contract documents %d routes, want %d", len(routes), len(want))
	}
	for route, methods := range want {
		documented, ok := routes[route]
		if !ok {
			t.Errorf("the contract does not document %s", route)
			continue
		}
		slices.Sort(documented)
		if !slices.Equal(documented, methods) {
			t.Errorf("%s documents %v, want %v", route, documented, methods)
		}
	}
}

func TestThePluginContractDeclaresItsShapes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		snippet string
	}{
		{name: "openapi version", snippet: "openapi: 3.1.0"},
		{name: "error schema", snippet: "\n    Error:\n"},
		{name: "manifest schema", snippet: "\n    Manifest:\n"},
		{name: "content page schema", snippet: "\n    ContentPage:\n"},
		{name: "content item schema", snippet: "\n    ContentItem:\n"},
		{name: "content link schema", snippet: "\n    ContentLink:\n"},
		{name: "seo request schema", snippet: "\n    SeoMetaRequest:\n"},
		{name: "seo result schema", snippet: "\n    SeoMetaResult:\n"},
		{name: "raw content schema", snippet: "\n    RawContent:\n"},
		{name: "raw update schema", snippet: "\n    RawUpdateRequest:\n"},
		{name: "raw result schema", snippet: "\n    RawUpdateResult:\n"},
		{name: "conflict schema", snippet: "\n    HashConflict:\n"},
		{name: "hash algorithm", snippet: "sha256"},
		{name: "conflict code", snippet: "hash_mismatch"},
		{name: "term rejection code", snippet: "not_found"},
		{name: "clamped limit", snippet: "maximum: 500"},
		{name: "nullable cursor", snippet: "- \"null\""},
		{name: "yoast key", snippet: "_yoast_wpseo_title"},
		{name: "rank math key", snippet: "rank_math_title"},
		{name: "fallback key", snippet: "_postulator_title"},
		{name: "basic auth", snippet: "scheme: basic"},
	}

	document := contractDocument(t)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if !strings.Contains(document, tc.snippet) {
				t.Errorf("the contract does not carry %q", tc.snippet)
			}
		})
	}
}
```

- [x] **Step 2: Run the test and watch it fail**

Run: `go test -count=1 ./internal/adapters/wp/...`
Expected: FAIL — `read the plugin contract: open ..\..\..\wp-plugin\openapi.yaml: The system cannot find the path specified.`

- [x] **Step 3: Write the contract**

Create `wp-plugin/openapi.yaml`:

```yaml
openapi: 3.1.0
info:
  title: Postulator companion plugin
  version: 1.0.0
  summary: The REST namespace the Postulator companion plugin adds to a WordPress site.
  description: >-
    Every route lives under the WordPress REST namespace postulator/v1 and is
    authenticated with the same WordPress application password as core REST.
    The plugin requires the edit_posts capability on every route. A site that
    does not have the plugin installed answers every route with the WordPress
    404 body {"code":"rest_no_route"}, which is how a client detects its
    absence and degrades to core REST.
servers:
  - url: https://{site}/wp-json/postulator/v1
    variables:
      site:
        default: example.com
        description: Host and optional subdirectory of the WordPress installation.
security:
  - applicationPassword: []
paths:
  /manifest:
    get:
      operationId: getManifest
      summary: Report the plugin version, its capabilities and the detected SEO plugin.
      responses:
        "200":
          description: The plugin is installed and the caller may use the namespace.
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Manifest"
        "401":
          $ref: "#/components/responses/Unauthorized"
        "403":
          $ref: "#/components/responses/Forbidden"
        "404":
          $ref: "#/components/responses/NoRoute"
  /content:
    get:
      operationId: listContent
      summary: List content in keyset order for incremental synchronisation.
      parameters:
        - name: since
          in: query
          required: false
          description: >-
            Exclusive lower bound on post_modified_gmt, RFC3339 with a UTC
            offset. An item modified at exactly this instant is not returned.
          schema:
            type: string
            format: date-time
        - name: cursor
          in: query
          required: false
          description: >-
            Opaque keyset cursor over (post_modified_gmt, ID) exactly as the
            previous page returned it in nextCursor. Clients never decode,
            validate or construct one.
          schema:
            type: string
        - name: types
          in: query
          required: false
          description: Comma separated content types; all four are listed when absent.
          schema:
            type: string
            default: page,post,product,product_cat
        - name: limit
          in: query
          required: false
          description: >-
            Maximum number of items in one page. The value is clamped, never
            rejected: a value above 500 becomes 500, and an absent value or a
            value of zero or less becomes 100.
          schema:
            type: integer
            minimum: 1
            maximum: 500
            default: 100
      responses:
        "200":
          description: One page of content ordered by (post_modified_gmt, ID) ascending.
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ContentPage"
        "400":
          $ref: "#/components/responses/BadRequest"
        "401":
          $ref: "#/components/responses/Unauthorized"
        "403":
          $ref: "#/components/responses/Forbidden"
        "404":
          $ref: "#/components/responses/NoRoute"
  /seo-meta/{id}:
    put:
      operationId: setSeoMeta
      summary: Write SEO meta through whichever SEO plugin the site runs.
      description: Post-only; a taxonomy term id answers 404 with the code not_found.
      parameters:
        - $ref: "#/components/parameters/PostID"
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/SeoMetaRequest"
      responses:
        "200":
          description: The fields that were written and the plugin they were written for.
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/SeoMetaResult"
        "400":
          $ref: "#/components/responses/BadRequest"
        "401":
          $ref: "#/components/responses/Unauthorized"
        "403":
          $ref: "#/components/responses/Forbidden"
        "404":
          $ref: "#/components/responses/NotFound"
  /content/{id}/raw:
    get:
      operationId: getRawContent
      summary: Read the stored post_content together with its hash.
      description: Post-only; a taxonomy term id answers 404 with the code not_found.
      parameters:
        - $ref: "#/components/parameters/PostID"
      responses:
        "200":
          description: The raw post_content and the hash a later write must present.
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/RawContent"
        "401":
          $ref: "#/components/responses/Unauthorized"
        "403":
          $ref: "#/components/responses/Forbidden"
        "404":
          $ref: "#/components/responses/NotFound"
    put:
      operationId: putRawContent
      summary: Replace post_content under optimistic concurrency control.
      description: Post-only; a taxonomy term id answers 404 with the code not_found.
      parameters:
        - $ref: "#/components/parameters/PostID"
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/RawUpdateRequest"
      responses:
        "200":
          description: The content was replaced; the new hash is returned.
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/RawUpdateResult"
        "400":
          $ref: "#/components/responses/BadRequest"
        "401":
          $ref: "#/components/responses/Unauthorized"
        "403":
          $ref: "#/components/responses/Forbidden"
        "404":
          $ref: "#/components/responses/NotFound"
        "409":
          $ref: "#/components/responses/Conflict"
components:
  securitySchemes:
    applicationPassword:
      type: http
      scheme: basic
      description: >-
        The WordPress user name and an application password created for that
        user, sent as HTTP Basic credentials on every request.
  parameters:
    PostID:
      name: id
      in: path
      required: true
      description: WordPress post, page, product or product category id.
      schema:
        type: integer
        format: int64
        minimum: 1
  responses:
    BadRequest:
      description: The request parameters were rejected.
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/Error"
    Unauthorized:
      description: No usable credentials were presented.
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/Error"
    Forbidden:
      description: The authenticated user lacks the edit_posts capability.
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/Error"
    NotFound:
      description: >-
        No post with that id exists, or the id names a taxonomy term. These
        routes are post-only: a term has neither post_content nor SEO post
        meta, so a term id answers 404 with the code not_found.
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/Error"
    NoRoute:
      description: >-
        The companion plugin is not installed or not active, so WordPress has
        no route for this path.
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/Error"
    Conflict:
      description: >-
        The stored content no longer hashes to expectedHash, so the write was
        refused and nothing changed.
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/HashConflict"
  schemas:
    Error:
      type: object
      required:
        - code
        - message
      properties:
        code:
          type: string
          description: Machine readable error code, WordPress style.
          examples:
            - rest_no_route
            - rest_forbidden
            - not_found
            - hash_mismatch
        message:
          type: string
          description: Human readable message, never a stack trace.
    HashConflict:
      type: object
      required:
        - code
        - message
        - currentHash
      properties:
        code:
          type: string
          const: hash_mismatch
        message:
          type: string
        currentHash:
          type: string
          pattern: "^[0-9a-f]{64}$"
          description: The hash the stored content has right now.
    Manifest:
      type: object
      required:
        - version
        - capabilities
        - seoPlugin
        - wpVersion
        - site
      properties:
        version:
          type: string
          examples:
            - 1.0.0
        capabilities:
          type: array
          description: >-
            For version 1.0.0 this is exactly bulk, seo_meta, content_hash and
            raw, in that order.
          items:
            type: string
            enum:
              - bulk
              - seo_meta
              - content_hash
              - raw
        seoPlugin:
          type: string
          enum:
            - yoast
            - rankmath
            - none
        wpVersion:
          type: string
          examples:
            - 6.9.1
        site:
          type: string
          format: uri
    ContentPage:
      type: object
      required:
        - items
        - nextCursor
      properties:
        items:
          type: array
          items:
            $ref: "#/components/schemas/ContentItem"
        nextCursor:
          type:
            - string
            - "null"
          description: >-
            Opaque cursor for the next page. The key is always present and its
            value is null on the last page.
    ContentItem:
      type: object
      required:
        - id
        - type
        - slug
        - path
        - parent
        - status
        - modified
        - contentHash
        - title
      properties:
        id:
          type: integer
          format: int64
        type:
          type: string
          enum:
            - page
            - post
            - product
            - product_cat
        slug:
          type: string
        path:
          type: string
          description: >-
            Site relative path with exactly one leading and one trailing
            slash, with no exception for a file extension. Duplicate slashes
            are collapsed, query and fragment are stripped, the whole path is
            ASCII-lowercased, and nothing is percent-decoded. The root is a
            single slash.
          examples:
            - /koffein/powder/
        parent:
          type: integer
          format: int64
          description: Parent id, 0 at the top level.
        status:
          type: string
          examples:
            - publish
            - draft
        modified:
          type: string
          format: date-time
          description: >-
            post_modified_gmt as RFC3339 with a UTC offset. For a product_cat
            the value comes from term meta the plugin maintains, because a
            taxonomy term has no modification column of its own.
        contentHash:
          type: string
          pattern: "^[0-9a-f]{64}$"
          description: >-
            Lowercase hex sha256 of the raw post_content bytes, computed as
            hash('sha256', $post->post_content). No trimming, no newline
            conversion and no HTML normalisation is applied, so the client can
            reproduce it byte for byte.
        title:
          type: string
        h1:
          type: string
          description: First h1 of the rendered content, empty when there is none.
        meta:
          $ref: "#/components/schemas/ContentMeta"
        links:
          type: array
          items:
            $ref: "#/components/schemas/ContentLink"
    ContentMeta:
      type: object
      properties:
        title:
          type: string
        description:
          type: string
        canonical:
          type: string
    ContentLink:
      type: object
      required:
        - href
        - anchor
      properties:
        href:
          type: string
          description: >-
            Site relative path of an internal anchor, normalised by the same
            rules as path, including the ASCII lowercasing. A link counts as
            internal when its host equals the site host exactly, compared in
            lowercase, with no www stripping and with the scheme ignored; a
            relative href is always internal. External links are not listed.
          examples:
            - /koffein/
        anchor:
          type: string
          description: Anchor text with markup stripped.
    SeoMetaRequest:
      type: object
      description: >-
        Fields left out or empty are not written. The plugin stores them under
        the key set of the SEO plugin it detected: Yoast uses
        _yoast_wpseo_title, _yoast_wpseo_metadesc, _yoast_wpseo_canonical,
        _yoast_wpseo_opengraph-title and _yoast_wpseo_opengraph-description;
        Rank Math uses rank_math_title, rank_math_description,
        rank_math_canonical_url, rank_math_facebook_title and
        rank_math_facebook_description; with no SEO plugin the values go to
        _postulator_title, _postulator_description, _postulator_canonical,
        _postulator_og_title and _postulator_og_description, which the plugin
        renders into the document head itself.
      properties:
        title:
          type: string
        description:
          type: string
        canonical:
          type: string
        ogTitle:
          type: string
        ogDescription:
          type: string
    SeoMetaResult:
      type: object
      required:
        - applied
        - seoPlugin
      properties:
        applied:
          type: array
          description: >-
            The request field names that were written, in the order title,
            description, canonical, ogTitle, ogDescription.
          items:
            type: string
            enum:
              - title
              - description
              - canonical
              - ogTitle
              - ogDescription
        seoPlugin:
          type: string
          enum:
            - yoast
            - rankmath
            - none
    RawContent:
      type: object
      required:
        - id
        - type
        - content
        - contentHash
      properties:
        id:
          type: integer
          format: int64
        type:
          type: string
          enum:
            - page
            - post
            - product
            - product_cat
        content:
          type: string
          description: The stored post_content, unmodified.
        contentHash:
          type: string
          pattern: "^[0-9a-f]{64}$"
    RawUpdateRequest:
      type: object
      required:
        - content
      properties:
        content:
          type: string
        expectedHash:
          type: string
          pattern: "^[0-9a-f]{64}$"
          description: >-
            The hash the caller last read. When present it must equal the hash
            of the stored content or the write is refused with 409. When
            absent the write is unconditional.
    RawUpdateResult:
      type: object
      required:
        - contentHash
      properties:
        contentHash:
          type: string
          pattern: "^[0-9a-f]{64}$"
          description: The hash of the content that is now stored.
```

- [x] **Step 4: Run the test and watch it pass**

Run: `go test -count=1 ./internal/adapters/wp/...`
Expected: PASS, both tests green.

- [x] **Step 5: Commit**

```bash
git add wp-plugin/openapi.yaml internal/adapters/wp/contract_test.go
git commit -m "feat(wp-plugin): freeze the postulator/v1 openapi contract

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 2: Client construction, options, transport and settings

**Files:**
- Create: `internal/adapters/wp/client.go`, `internal/adapters/wp/option.go`, `internal/adapters/wp/transport.go`, `internal/adapters/wp/settings.go`, `internal/adapters/wp/hash.go`
- Test: `internal/adapters/wp/client_test.go` (package `wp`), `internal/adapters/wp/hash_test.go` (package `wp_test`)
- Modify: `go.mod`, `go.sum`

**Interfaces:**
- Consumes: `kernel/errors`, `kernel/settings`, `golang.org/x/net/proxy`, `golang.org/x/time/rate`, `go.uber.org/zap`.
- Produces:
  - `type Config struct { BaseURL, Username, AppPassword string; AllowInsecure bool }`
  - `type Client struct{ ... }`, `func New(cfg Config, opts ...Option) (*Client, error)`
  - `type Logger interface { Debug(string, ...zap.Field); Warn(string, ...zap.Field) }`
  - `type Option func(*options)` with `WithProxy(string)`, `WithTimeout(time.Duration)`, `WithRetries(int)`, `WithBackoff(func(int) time.Duration)`, `WithRateLimit(float64)`, `WithLogger(Logger)`
  - `func FromSettings(values *settings.Values) []Option`
  - `func ContentHash(raw string) string`
  - constants `DefaultTimeout`, `DefaultRetries`, `DefaultRateLimitPerSecond`; namespace constants `rootPath`, `coreNamespace`, `wooNamespace`, `pluginNamespace`
  - `func (c *Client) resolve(namespace, path string, query url.Values) string`

- [x] **Step 1: Add the two pre-approved modules**

```bash
go get golang.org/x/net@v0.58.0
go get golang.org/x/time@v0.15.0
go mod tidy
```

Expected: `go.mod` gains `golang.org/x/net v0.58.0` and `golang.org/x/time v0.15.0` in the direct require block. Both are already in the module cache and neither pulls a new transitive requirement: `x/net` requires `golang.org/x/crypto v0.55.0` and `golang.org/x/sys v0.47.0`, which are the versions already pinned.

- [x] **Step 2: Write the failing test**

Create `internal/adapters/wp/client_test.go`:

```go
package wp

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"slices"
	"testing"
	"time"

	"golang.org/x/time/rate"

	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/settings"
)

func TestNewRejectsAnUnusableConfig(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		config Config
	}{
		{name: "empty base url", config: Config{Username: "u", AppPassword: "p"}},
		{name: "no host", config: Config{BaseURL: "not a url", Username: "u", AppPassword: "p"}},
		{name: "plain http without consent", config: Config{BaseURL: "http://example.com", Username: "u", AppPassword: "p"}},
		{name: "unknown scheme", config: Config{BaseURL: "ftp://example.com", Username: "u", AppPassword: "p"}},
		{name: "no user", config: Config{BaseURL: "https://example.com", AppPassword: "p"}},
		{name: "no application password", config: Config{BaseURL: "https://example.com", Username: "u"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client, err := New(tc.config)
			if client != nil {
				t.Fatal("a rejected config must not produce a client")
			}
			if !errors.IsCode(err, errors.Invalid) {
				t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
			}
		})
	}
}

func TestNewAcceptsPlainHTTPOnlyWithConsent(t *testing.T) {
	t.Parallel()

	client, err := New(Config{BaseURL: "http://example.com/blog/", Username: "u", AppPassword: "p", AllowInsecure: true})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got := client.resolve(coreNamespace, "/pages", nil); got != "http://example.com/blog/wp-json/wp/v2/pages" {
		t.Errorf("resolve = %q", got)
	}
}

func TestNewAppliesTheDefaults(t *testing.T) {
	t.Parallel()

	client, err := New(Config{BaseURL: "https://example.com", Username: "u", AppPassword: "p"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if client.http.Timeout != DefaultTimeout {
		t.Errorf("timeout = %s, want %s", client.http.Timeout, DefaultTimeout)
	}
	if client.retries != DefaultRetries {
		t.Errorf("retries = %d, want %d", client.retries, DefaultRetries)
	}
	if client.limiter.Limit() != rate.Limit(DefaultRateLimitPerSecond) {
		t.Errorf("rate = %v, want %v", client.limiter.Limit(), rate.Limit(DefaultRateLimitPerSecond))
	}
	if client.transport.TLSClientConfig != nil {
		t.Error("the transport must keep the standard TLS configuration")
	}
	if client.transport.Proxy != nil {
		t.Error("no proxy is configured, so the transport must not consult one")
	}
	if client.http.CheckRedirect != nil {
		t.Error("ordinary calls follow redirects")
	}
	if client.probe.CheckRedirect == nil {
		t.Error("the probe client must refuse to follow redirects")
	}
	if client.http.Transport != client.probe.Transport {
		t.Error("both clients must share one transport")
	}
}

func TestTheLastOptionOfAKindWins(t *testing.T) {
	t.Parallel()

	client, err := New(
		Config{BaseURL: "https://example.com", Username: "u", AppPassword: "p"},
		WithTimeout(time.Second), WithTimeout(7*time.Second),
		WithRetries(9), WithRetries(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if client.http.Timeout != 7*time.Second {
		t.Errorf("timeout = %s, want 7s", client.http.Timeout)
	}
	if client.retries != 1 {
		t.Errorf("retries = %d, want 1", client.retries)
	}
}

func TestWithRateLimitZeroLiftsTheLimit(t *testing.T) {
	t.Parallel()

	client, err := New(Config{BaseURL: "https://example.com", Username: "u", AppPassword: "p"}, WithRateLimit(0))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if client.limiter.Limit() != rate.Inf {
		t.Errorf("rate = %v, want infinite", client.limiter.Limit())
	}
}

func TestTheHTTPProxyReachesTheTransport(t *testing.T) {
	t.Parallel()

	client, err := New(Config{BaseURL: "https://example.com", Username: "u", AppPassword: "p"}, WithProxy("http://127.0.0.1:3128"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if client.transport.Proxy == nil {
		t.Fatal("the transport has no proxy function")
	}

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.com/wp-json", nil)
	if err != nil {
		t.Fatalf("build the request: %v", err)
	}
	proxied, err := client.transport.Proxy(request)
	if err != nil {
		t.Fatalf("proxy lookup: %v", err)
	}
	if proxied == nil || proxied.Host != "127.0.0.1:3128" {
		t.Errorf("proxy = %v, want 127.0.0.1:3128", proxied)
	}
}

func TestAnUnusableProxyURLIsRejected(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		proxy string
	}{
		{name: "no host", proxy: "http://"},
		{name: "unknown scheme", proxy: "ftp://127.0.0.1:1080"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := New(Config{BaseURL: "https://example.com", Username: "u", AppPassword: "p"}, WithProxy(tc.proxy))
			if !errors.IsCode(err, errors.Invalid) {
				t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
			}
		})
	}
}

type socksRequest struct {
	methods []byte
	host    string
	port    uint16
}

func socksListener(t *testing.T) (string, <-chan socksRequest) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	seen := make(chan socksRequest, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		request, readErr := readSocksRequest(conn)
		if readErr != nil {
			return
		}
		seen <- request
	}()
	return listener.Addr().String(), seen
}

func readSocksRequest(conn net.Conn) (socksRequest, error) {
	greeting := make([]byte, 2)
	if _, err := io.ReadFull(conn, greeting); err != nil {
		return socksRequest{}, err
	}
	methods := make([]byte, greeting[1])
	if _, err := io.ReadFull(conn, methods); err != nil {
		return socksRequest{}, err
	}
	if _, err := conn.Write([]byte{0x05, 0x00}); err != nil {
		return socksRequest{}, err
	}

	header := make([]byte, 5)
	if _, err := io.ReadFull(conn, header); err != nil {
		return socksRequest{}, err
	}
	name := make([]byte, header[4])
	if _, err := io.ReadFull(conn, name); err != nil {
		return socksRequest{}, err
	}
	port := make([]byte, 2)
	if _, err := io.ReadFull(conn, port); err != nil {
		return socksRequest{}, err
	}
	return socksRequest{methods: methods, host: string(name), port: binary.BigEndian.Uint16(port)}, nil
}

func TestTheSocksProxyDialerCarriesTheConnect(t *testing.T) {
	t.Parallel()

	address, seen := socksListener(t)
	client, err := New(
		Config{BaseURL: "https://example.com", Username: "u", AppPassword: "p"},
		WithProxy("socks5://operator:secret@"+address),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if client.transport.Proxy != nil {
		t.Error("a socks5 proxy is a dialer, not an http proxy function")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	conn, err := client.transport.DialContext(ctx, "tcp", "example.invalid:443")
	if err == nil {
		_ = conn.Close()
	}

	select {
	case request := <-seen:
		if request.host != "example.invalid" {
			t.Errorf("host = %q, want example.invalid", request.host)
		}
		if request.port != 443 {
			t.Errorf("port = %d, want 443", request.port)
		}
		if !slices.Contains(request.methods, byte(0x02)) {
			t.Error("the dialer did not offer username and password authentication")
		}
	case <-ctx.Done():
		t.Fatal("the socks5 proxy never saw a connect request")
	}
}

func TestFromSettingsConfiguresTheClient(t *testing.T) {
	t.Parallel()

	registry := settings.Default()
	values := registry.NewValues()
	unknown, err := registry.Apply(values, map[string]json.RawMessage{
		"wp.timeout":            json.RawMessage(`"9s"`),
		"wp.retries":            json.RawMessage(`2`),
		"wp.rateLimitPerSecond": json.RawMessage(`7`),
		"wp.proxyUrl":           json.RawMessage(`"http://127.0.0.1:3128"`),
	})
	if err != nil {
		t.Fatalf("apply the settings: %v", err)
	}
	if len(unknown) != 0 {
		t.Fatalf("unknown settings %v", unknown)
	}

	client, err := New(Config{BaseURL: "https://example.com", Username: "u", AppPassword: "p"}, FromSettings(values)...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if client.http.Timeout != 9*time.Second {
		t.Errorf("timeout = %s, want 9s", client.http.Timeout)
	}
	if client.retries != 2 {
		t.Errorf("retries = %d, want 2", client.retries)
	}
	if client.limiter.Limit() != rate.Limit(7) {
		t.Errorf("rate = %v, want 7", client.limiter.Limit())
	}
	if client.transport.Proxy == nil {
		t.Error("the proxy setting did not reach the transport")
	}
}

func TestTheProxySettingRejectsRubbish(t *testing.T) {
	t.Parallel()

	registry := settings.Default()
	_, err := registry.Apply(registry.NewValues(), map[string]json.RawMessage{
		"wp.proxyUrl": json.RawMessage(`"ftp://127.0.0.1:1080"`),
	})
	if !errors.IsCode(err, errors.Invalid) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
	}
}

func TestTheDefaultBackoffGrowsAndIsCapped(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		attempt int
		want    time.Duration
	}{
		{name: "first retry", attempt: 0, want: 500 * time.Millisecond},
		{name: "second retry", attempt: 1, want: time.Second},
		{name: "third retry", attempt: 2, want: 2 * time.Second},
		{name: "fourth retry", attempt: 3, want: 4 * time.Second},
		{name: "capped", attempt: 9, want: 8 * time.Second},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := defaultBackoff(tc.attempt); got != tc.want {
				t.Errorf("defaultBackoff(%d) = %s, want %s", tc.attempt, got, tc.want)
			}
		})
	}
}
```

Create `internal/adapters/wp/hash_test.go`:

```go
package wp_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp"
)

func TestContentHashIsPlainSHA256OfTheRawBytes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		raw  string
		want string
	}{
		{name: "empty", raw: "", want: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		{name: "paragraph", raw: "<p>Koffein ist ein Alkaloid.</p>", want: "119b7cff7356b21d2b00e64d2d3c0589b50f3270b302a94360fac92f1adc332b"},
		{name: "powder", raw: "<p>Powder</p>", want: "79db24b7a931978a1db05ccdb1ae1ed16b07aafa5277af75a7dc1b3b7ec6a509"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := wp.ContentHash(tc.raw); got != tc.want {
				t.Errorf("ContentHash(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

func TestContentHashDoesNotNormalise(t *testing.T) {
	t.Parallel()

	if wp.ContentHash("<p>a</p>\n") == wp.ContentHash("<p>a</p>") {
		t.Error("a trailing newline must change the hash; the hash is over the raw bytes")
	}
	if wp.ContentHash(" <p>a</p>") == wp.ContentHash("<p>a</p>") {
		t.Error("leading whitespace must change the hash; the hash is over the raw bytes")
	}
}
```

- [x] **Step 3: Run the tests and watch them fail**

Run: `go test -count=1 ./internal/adapters/wp/...`
Expected: FAIL — `undefined: New`, `undefined: Config`, `undefined: ContentHash`.

- [x] **Step 4: Write the implementation**

Create `internal/adapters/wp/hash.go`:

```go
package wp

import (
	"crypto/sha256"
	"encoding/hex"
)

func ContentHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
```

Create `internal/adapters/wp/client.go`:

```go
package wp

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	rootPath        = "/wp-json"
	coreNamespace   = "/wp-json/wp/v2"
	wooNamespace    = "/wp-json/wc/v3"
	pluginNamespace = "/wp-json/postulator/v1"

	userAgent    = "Postulator/2"
	maxBodyBytes = 32 << 20
)

type Config struct {
	BaseURL       string
	Username      string
	AppPassword   string
	AllowInsecure bool
}

type Client struct {
	base      *url.URL
	http      *http.Client
	probe     *http.Client
	transport *http.Transport
	limiter   *rate.Limiter
	backoff   func(int) time.Duration
	logger    Logger
	username  string
	password  string
	retries   int
}

func New(cfg Config, opts ...Option) (*Client, error) {
	base, err := baseURL(cfg)
	if err != nil {
		return nil, err
	}
	if cfg.Username == "" || cfg.AppPassword == "" {
		return nil, errors.New(errors.Invalid, "the site needs a WordPress user name and an application password")
	}

	resolved := options{
		timeout: DefaultTimeout,
		retries: DefaultRetries,
		rate:    DefaultRateLimitPerSecond,
		backoff: defaultBackoff,
		logger:  zap.NewNop(),
	}
	for _, opt := range opts {
		opt(&resolved)
	}
	if resolved.retries < 0 {
		return nil, errors.New(errors.Invalid, "the retry count must not be negative")
	}

	transport, err := newTransport(resolved.proxyURL)
	if err != nil {
		return nil, err
	}

	return &Client{
		base:      base,
		http:      &http.Client{Transport: transport, Timeout: resolved.timeout},
		probe:     &http.Client{Transport: transport, Timeout: resolved.timeout, CheckRedirect: keepRedirect},
		transport: transport,
		limiter:   newLimiter(resolved.rate),
		backoff:   resolved.backoff,
		logger:    resolved.logger,
		username:  cfg.Username,
		password:  cfg.AppPassword,
		retries:   resolved.retries,
	}, nil
}

func keepRedirect(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}

func baseURL(cfg Config) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimRight(cfg.BaseURL, "/"))
	if err != nil || parsed.Host == "" {
		return nil, errors.New(errors.Invalid, "the site base URL is not a valid absolute URL")
	}

	switch parsed.Scheme {
	case "https":
		return parsed, nil
	case "http":
		if cfg.AllowInsecure {
			return parsed, nil
		}
		return nil, errors.New(errors.Invalid, "the site base URL must use https unless the site is marked as insecure")
	default:
		return nil, errors.New(errors.Invalid, "the site base URL must use http or https").WithDetail("scheme", parsed.Scheme)
	}
}

func (c *Client) resolve(namespace, path string, query url.Values) string {
	target := *c.base
	target.Path = c.base.Path + namespace + path
	if len(query) > 0 {
		target.RawQuery = query.Encode()
	}
	return target.String()
}
```

Create `internal/adapters/wp/option.go`, which is where `zap.NewNop()` above comes from:

```go
package wp

import (
	"math"
	"time"

	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

const (
	DefaultTimeout            = 30 * time.Second
	DefaultRetries            = 4
	DefaultRateLimitPerSecond = 4

	backoffBase     = 500 * time.Millisecond
	backoffCap      = 8 * time.Second
	backoffMaxShift = 5
)

type Logger interface {
	Debug(message string, fields ...zap.Field)
	Warn(message string, fields ...zap.Field)
}

type options struct {
	backoff  func(int) time.Duration
	logger   Logger
	proxyURL string
	timeout  time.Duration
	rate     float64
	retries  int
}

type Option func(*options)

func WithProxy(rawURL string) Option {
	return func(o *options) { o.proxyURL = rawURL }
}

func WithTimeout(timeout time.Duration) Option {
	return func(o *options) { o.timeout = timeout }
}

func WithRetries(retries int) Option {
	return func(o *options) { o.retries = retries }
}

func WithBackoff(backoff func(attempt int) time.Duration) Option {
	return func(o *options) {
		if backoff != nil {
			o.backoff = backoff
		}
	}
}

func WithRateLimit(perSecond float64) Option {
	return func(o *options) { o.rate = perSecond }
}

func WithLogger(logger Logger) Option {
	return func(o *options) {
		if logger != nil {
			o.logger = logger
		}
	}
}

func defaultBackoff(attempt int) time.Duration {
	if attempt < 0 {
		return backoffBase
	}
	if attempt > backoffMaxShift {
		return backoffCap
	}
	delay := backoffBase << attempt
	if delay > backoffCap {
		return backoffCap
	}
	return delay
}

func newLimiter(perSecond float64) *rate.Limiter {
	if perSecond <= 0 {
		return rate.NewLimiter(rate.Inf, 1)
	}
	return rate.NewLimiter(rate.Limit(perSecond), int(math.Ceil(perSecond)))
}
```

Create `internal/adapters/wp/transport.go`:

```go
package wp

import (
	"net"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/proxy"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func newTransport(proxyURL string) (*http.Transport, error) {
	transport := &http.Transport{
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          16,
		MaxIdleConnsPerHost:   4,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: time.Second,
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
	}
	if proxyURL == "" {
		return transport, nil
	}

	parsed, err := url.Parse(proxyURL)
	if err != nil || parsed.Host == "" {
		return nil, errors.New(errors.Invalid, "the proxy URL is not a valid absolute URL")
	}

	switch parsed.Scheme {
	case "http", "https":
		transport.Proxy = http.ProxyURL(parsed)
		return transport, nil
	case "socks5", "socks5h":
		return socksTransport(transport, parsed)
	default:
		return nil, errors.New(errors.Invalid, "the proxy URL must use http, https or socks5").WithDetail("scheme", parsed.Scheme)
	}
}

func socksTransport(transport *http.Transport, parsed *url.URL) (*http.Transport, error) {
	dialer, err := proxy.SOCKS5("tcp", parsed.Host, socksAuth(parsed), proxy.Direct)
	if err != nil {
		return nil, errors.New(errors.Invalid, "the socks5 proxy cannot be used").WithInternal(err)
	}

	contextDialer, ok := dialer.(proxy.ContextDialer)
	if !ok {
		return nil, errors.New(errors.Internal, "the socks5 proxy dialer does not support contexts")
	}
	transport.DialContext = contextDialer.DialContext
	return transport, nil
}

func socksAuth(parsed *url.URL) *proxy.Auth {
	if parsed.User == nil {
		return nil
	}
	password, _ := parsed.User.Password()
	return &proxy.Auth{User: parsed.User.Username(), Password: password}
}
```

Create `internal/adapters/wp/settings.go`:

```go
package wp

import (
	"fmt"
	"net/url"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/settings"
)

var (
	timeoutSetting = settings.Duration("wp.timeout", DefaultTimeout, settings.DurationRange(time.Second, 5*time.Minute))
	retriesSetting = settings.Int("wp.retries", DefaultRetries, settings.IntRange(0, 10))
	rateSetting    = settings.Int("wp.rateLimitPerSecond", DefaultRateLimitPerSecond, settings.IntRange(1, 100))
	proxySetting   = settings.String("wp.proxyUrl", "", proxyURLValidator())
)

func proxyURLValidator() settings.Validator[string] {
	return settings.Check(func(value string) error {
		if value == "" {
			return nil
		}

		parsed, err := url.Parse(value)
		if err != nil || parsed.Host == "" {
			return fmt.Errorf("value %q is not a valid proxy URL", value)
		}
		switch parsed.Scheme {
		case "http", "https", "socks5", "socks5h":
			return nil
		default:
			return fmt.Errorf("scheme %q is not one of http, https, socks5, socks5h", parsed.Scheme)
		}
	})
}

func FromSettings(values *settings.Values) []Option {
	return []Option{
		WithTimeout(timeoutSetting.Get(values)),
		WithRetries(retriesSetting.Get(values)),
		WithRateLimit(float64(rateSetting.Get(values))),
		WithProxy(proxySetting.Get(values)),
	}
}
```

- [x] **Step 5: Run the tests and watch them pass**

Run: `go test -count=1 -race ./internal/adapters/wp/...`
Expected: PASS.

- [x] **Step 6: Run the gate**

```bash
go build ./... && go vet ./... && "$(go env GOPATH)/bin/golangci-lint.exe" run && go test -count=1 -race ./...
```
Expected: all green, 0 lint issues.

- [x] **Step 7: Commit**

```bash
git add go.mod go.sum internal/adapters/wp
git commit -m "feat(wp): client construction, proxy transport and settings

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---
### Task 3: The fake server — state, routing, auth, recording, faults and redirects

`wptest` is built before the client's request pipeline so the pipeline has something faithful to be tested against, and it is driven here with plain `net/http` so it is proven to speak WordPress independently of the client (D7).

**Files:**
- Create: `internal/adapters/wp/wptest/server.go`, `internal/adapters/wp/wptest/state.go`, `internal/adapters/wp/wptest/fault.go`
- Test: `internal/adapters/wp/wptest/server_test.go` (package `wptest_test`), `internal/adapters/wp/wptest/state_test.go` (package `wptest`)

**Interfaces:**
- Consumes: `net/http/httptest`, `testing`.
- Produces:
  - `type Server struct{ ... }`, `func New(t testing.TB, opts ...Option) *Server`, `func (s *Server) URL() string`
  - `type Option func(*Server)` with `WithCredentials(user, password string)`, `WithSEOPlugin(name string)`, `WithoutPlugin()`, `WithoutNamespaces()`, `WithRedirect(mode Redirect)`
  - `type Redirect string` with `RedirectNone`, `RedirectHTTPS`, `RedirectLogin`, `RedirectAdmin`
  - `type Item struct{ ... }`, `type Category struct{ ... }`, `func (s *Server) Seed(items ...Item) []Item`, `func (s *Server) SeedCategory(category Category) Category`, `func (s *Server) Lookup(id int64) (Item, bool)`, `func (s *Server) Items() []Item`, `func (s *Server) Now() time.Time`
  - `type Request struct{ Query url.Values; Header http.Header; Method, Path string; Body []byte }`, `func (s *Server) Requests() []Request`, `func (s *Server) LastRequest() (Request, bool)`, `func (s *Server) ResetRequests()`
  - `func (s *Server) FailNext(status, times int)`, `func (s *Server) RateLimitNext(retryAfter time.Duration)`
  - constants `DefaultUser`, `DefaultPassword`, `TypePage`, `TypePost`, `TypeProduct`, `TypeProductCategory`
  - unexported for later tasks: `s.respond`, `s.fail`, `s.lock`-guarded store, `slugify`, `uniqueSlug`, `itemPath`, `contentHash`, `s.tick`

- [x] **Step 1: Write the failing tests**

Create `internal/adapters/wp/wptest/state_test.go`:

```go
package wptest

import "testing"

func TestSlugify(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		title string
		want  string
	}{
		{name: "plain", title: "Powder", want: "powder"},
		{name: "spaces collapse", title: "Koffein   Powder", want: "koffein-powder"},
		{name: "punctuation collapses", title: "Koffein: Powder!", want: "koffein-powder"},
		{name: "trims", title: "  Powder  ", want: "powder"},
		{name: "non ascii dropped", title: "Grüner Tee", want: "gr-ner-tee"},
		{name: "empty falls back", title: "!!!", want: "item"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := slugify(tc.title); got != tc.want {
				t.Errorf("slugify(%q) = %q, want %q", tc.title, got, tc.want)
			}
		})
	}
}

func TestContentHashMatchesTheFrozenDigests(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		raw  string
		want string
	}{
		{name: "empty", raw: "", want: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		{name: "paragraph", raw: "<p>Koffein ist ein Alkaloid.</p>", want: "119b7cff7356b21d2b00e64d2d3c0589b50f3270b302a94360fac92f1adc332b"},
		{name: "powder", raw: "<p>Powder</p>", want: "79db24b7a931978a1db05ccdb1ae1ed16b07aafa5277af75a7dc1b3b7ec6a509"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := contentHash(tc.raw); got != tc.want {
				t.Errorf("contentHash(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}
```

Create `internal/adapters/wp/wptest/server_test.go`:

```go
package wptest_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
)

func call(t *testing.T, server *wptest.Server, method, path string, body []byte, authenticated bool) (*http.Response, []byte) {
	t.Helper()

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	request, err := http.NewRequestWithContext(t.Context(), method, server.URL()+path, reader)
	if err != nil {
		t.Fatalf("build the request: %v", err)
	}
	if authenticated {
		request.SetBasicAuth(wptest.DefaultUser, wptest.DefaultPassword)
	}

	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("send the request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read the response: %v", err)
	}
	return response, payload
}

func decode(t *testing.T, payload []byte, out any) {
	t.Helper()

	if err := json.Unmarshal(payload, out); err != nil {
		t.Fatalf("decode %s: %v", payload, err)
	}
}

func TestTheRestRootListsItsNamespaces(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	response, payload := call(t, server, http.MethodGet, "/wp-json", nil, true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}

	var root struct {
		Name       string   `json:"name"`
		Home       string   `json:"home"`
		Namespaces []string `json:"namespaces"`
	}
	decode(t, payload, &root)

	want := []string{"oembed/1.0", "wp/v2", "wc/v3", "postulator/v1"}
	if len(root.Namespaces) != len(want) {
		t.Fatalf("namespaces = %v, want %v", root.Namespaces, want)
	}
	if root.Home != server.URL() {
		t.Errorf("home = %q, want %q", root.Home, server.URL())
	}
}

func TestTheRestRootCanHideThePluginAndTheNamespaces(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		option wptest.Option
		want   int
	}{
		{name: "without the plugin", option: wptest.WithoutPlugin(), want: 3},
		{name: "without any namespace", option: wptest.WithoutNamespaces(), want: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := wptest.New(t, tc.option)
			_, payload := call(t, server, http.MethodGet, "/wp-json", nil, true)

			var root struct {
				Namespaces []string `json:"namespaces"`
			}
			decode(t, payload, &root)
			if len(root.Namespaces) != tc.want {
				t.Errorf("namespaces = %v, want %d of them", root.Namespaces, tc.want)
			}
		})
	}
}

func TestEveryRouteDemandsTheApplicationPassword(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	response, payload := call(t, server, http.MethodGet, "/wp-json", nil, false)
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.StatusCode)
	}

	var failure struct {
		Code string `json:"code"`
	}
	decode(t, payload, &failure)
	if failure.Code != "rest_not_logged_in" {
		t.Errorf("code = %q, want rest_not_logged_in", failure.Code)
	}
}

func TestTheCredentialsAreConfigurable(t *testing.T) {
	t.Parallel()

	server := wptest.New(t, wptest.WithCredentials("editor", "one two three"))
	response, _ := call(t, server, http.MethodGet, "/wp-json", nil, true)
	if response.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 for the default credentials", response.StatusCode)
	}
}

func TestTheRootRedirectsWithoutAskingForCredentials(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		mode     wptest.Redirect
		status   int
		contains string
	}{
		{name: "https upgrade", mode: wptest.RedirectHTTPS, status: http.StatusMovedPermanently, contains: "https://"},
		{name: "login page", mode: wptest.RedirectLogin, status: http.StatusFound, contains: "wp-login.php"},
		{name: "admin", mode: wptest.RedirectAdmin, status: http.StatusFound, contains: "/wp-admin/"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := wptest.New(t, wptest.WithRedirect(tc.mode))
			response, _ := call(t, server, http.MethodGet, "/wp-json", nil, false)
			if response.StatusCode != tc.status {
				t.Fatalf("status = %d, want %d", response.StatusCode, tc.status)
			}
			if location := response.Header.Get("Location"); !bytes.Contains([]byte(location), []byte(tc.contains)) {
				t.Errorf("location = %q, want it to contain %q", location, tc.contains)
			}
		})
	}
}

func TestFailNextFiresExactlyThatManyTimes(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.FailNext(http.StatusServiceUnavailable, 2)

	for attempt := range 3 {
		response, _ := call(t, server, http.MethodGet, "/wp-json", nil, true)
		want := http.StatusServiceUnavailable
		if attempt == 2 {
			want = http.StatusOK
		}
		if response.StatusCode != want {
			t.Errorf("attempt %d status = %d, want %d", attempt, response.StatusCode, want)
		}
	}
}

func TestRateLimitNextCarriesRetryAfter(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.RateLimitNext(3 * time.Second)

	response, _ := call(t, server, http.MethodGet, "/wp-json", nil, true)
	if response.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", response.StatusCode)
	}
	if got := response.Header.Get("Retry-After"); got != "3" {
		t.Errorf("Retry-After = %q, want 3", got)
	}
}

func TestRequestsAreRecorded(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	call(t, server, http.MethodGet, "/wp-json?probe=1", nil, true)

	recorded, ok := server.LastRequest()
	if !ok {
		t.Fatal("no request was recorded")
	}
	if recorded.Method != http.MethodGet || recorded.Path != "/wp-json" {
		t.Errorf("recorded %s %s", recorded.Method, recorded.Path)
	}
	if recorded.Query.Get("probe") != "1" {
		t.Errorf("query = %v", recorded.Query)
	}
	if recorded.Header.Get("Authorization") == "" {
		t.Error("the recorder must keep the request headers")
	}

	server.ResetRequests()
	if len(server.Requests()) != 0 {
		t.Error("ResetRequests must empty the recording")
	}
}

func TestSeedingAssignsIdsSlugsAndAMonotonicClock(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := server.Seed(
		wptest.Item{Type: wptest.TypePage, Title: "Koffein"},
		wptest.Item{Type: wptest.TypePage, Title: "Powder"},
		wptest.Item{Type: wptest.TypePage, Title: "Powder"},
	)

	if len(seeded) != 3 {
		t.Fatalf("seeded %d items, want 3", len(seeded))
	}
	if seeded[0].ID != 1 || seeded[1].ID != 2 || seeded[2].ID != 3 {
		t.Errorf("ids = %d %d %d, want 1 2 3", seeded[0].ID, seeded[1].ID, seeded[2].ID)
	}
	if seeded[1].Slug != "powder" || seeded[2].Slug != "powder-2" {
		t.Errorf("slugs = %q %q, want powder and powder-2", seeded[1].Slug, seeded[2].Slug)
	}
	if !seeded[0].Modified.Before(seeded[1].Modified) {
		t.Error("the fake clock must advance for every stored item")
	}

	stored, ok := server.Lookup(seeded[2].ID)
	if !ok || stored.Slug != "powder-2" {
		t.Errorf("Lookup = %+v, %t", stored, ok)
	}
	if len(server.Items()) != 3 {
		t.Errorf("Items = %d, want 3", len(server.Items()))
	}
}

func TestASlugIsUniquePerParentForHierarchicalTypes(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	parent := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein"})[0]
	children := server.Seed(
		wptest.Item{Type: wptest.TypePage, Title: "Powder", Parent: parent.ID},
		wptest.Item{Type: wptest.TypePost, Title: "Powder"},
	)

	if children[0].Slug != "powder" {
		t.Errorf("a child under another parent may reuse the slug, got %q", children[0].Slug)
	}
	if children[1].Slug != "powder" {
		t.Errorf("a different type may reuse the slug, got %q", children[1].Slug)
	}
}
```

- [x] **Step 2: Run the tests and watch them fail**

Run: `go test -count=1 ./internal/adapters/wp/wptest/...`
Expected: FAIL — `no Go files in ...\wptest`.

- [x] **Step 3: Write the state**

Create `internal/adapters/wp/wptest/state.go`:

```go
package wptest

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"
)

const (
	TypePage            = "page"
	TypePost            = "post"
	TypeProduct         = "product"
	TypeProductCategory = "product_cat"

	maxPathDepth = 10
)

var startInstant = time.Date(2026, time.September, 18, 10, 0, 0, 0, time.UTC)

type Item struct {
	Modified      time.Time
	Meta          map[string]string
	Type          string
	Title         string
	H1            string
	Content       string
	Excerpt       string
	Slug          string
	Status        string
	Template      string
	Categories    []int64
	Tags          []int64
	ID            int64
	Parent        int64
	MenuOrder     int
	FeaturedMedia int64
}

type Category struct {
	Name        string
	Slug        string
	Description string
	ID          int64
	Parent      int64
	Count       int
}

type upload struct {
	Filename string
	MimeType string
	Alt      string
	Title    string
	Bytes    []byte
	ID       int64
}

func contentHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func slugify(title string) string {
	var builder strings.Builder
	dashed := false
	for _, symbol := range strings.ToLower(title) {
		if (symbol >= 'a' && symbol <= 'z') || (symbol >= '0' && symbol <= '9') {
			builder.WriteRune(symbol)
			dashed = false
			continue
		}
		if !dashed && builder.Len() > 0 {
			builder.WriteByte('-')
			dashed = true
		}
	}

	trimmed := strings.Trim(builder.String(), "-")
	if trimmed == "" {
		return "item"
	}
	return trimmed
}

func hierarchical(itemType string) bool {
	return itemType == TypePage || itemType == TypeProductCategory
}

func (s *Server) tick() time.Time {
	s.clock = s.clock.Add(time.Second)
	return s.clock
}

func (s *Server) Now() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.clock
}

func (s *Server) uniqueSlug(base, itemType string, parent, exclude int64) string {
	candidate := base
	for suffix := 2; s.slugTaken(candidate, itemType, parent, exclude); suffix++ {
		candidate = base + "-" + strconv.Itoa(suffix)
	}
	return candidate
}

func (s *Server) slugTaken(slug, itemType string, parent, exclude int64) bool {
	for _, id := range s.order {
		stored := s.items[id]
		if stored.ID == exclude || stored.Type != itemType || stored.Slug != slug {
			continue
		}
		if !hierarchical(itemType) || stored.Parent == parent {
			return true
		}
	}
	return false
}

func (s *Server) itemPath(stored *Item) string {
	segments := []string{stored.Slug}
	parent := stored.Parent
	for depth := 0; parent != 0 && depth < maxPathDepth; depth++ {
		ancestor, ok := s.items[parent]
		if !ok {
			break
		}
		segments = append([]string{ancestor.Slug}, segments...)
		parent = ancestor.Parent
	}
	return "/" + strings.Join(segments, "/") + "/"
}

func (s *Server) add(item Item) Item {
	if item.Type == "" {
		item.Type = TypePage
	}
	if item.Status == "" {
		item.Status = "publish"
	}
	if item.Meta == nil {
		item.Meta = make(map[string]string)
	}

	s.nextID++
	item.ID = s.nextID

	base := item.Slug
	if base == "" {
		base = slugify(item.Title)
	}
	item.Slug = s.uniqueSlug(base, item.Type, item.Parent, item.ID)

	if item.Modified.IsZero() {
		item.Modified = s.tick()
	}

	stored := item
	s.items[item.ID] = &stored
	s.order = append(s.order, item.ID)
	return stored
}

func (s *Server) Seed(items ...Item) []Item {
	s.mu.Lock()
	defer s.mu.Unlock()

	stored := make([]Item, 0, len(items))
	for _, item := range items {
		stored = append(stored, s.add(item))
	}
	return stored
}

func (s *Server) SeedCategory(category Category) Category {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	category.ID = s.nextID
	if category.Slug == "" {
		category.Slug = slugify(category.Name)
	}

	stored := category
	s.categories[category.ID] = &stored
	s.categoryOrder = append(s.categoryOrder, category.ID)
	return stored
}

func (s *Server) Lookup(id int64) (Item, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stored, ok := s.items[id]
	if !ok {
		return Item{}, false
	}
	return *stored, true
}

func (s *Server) Items() []Item {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]Item, 0, len(s.order))
	for _, id := range s.order {
		items = append(items, *s.items[id])
	}
	return items
}
```

Create `internal/adapters/wp/wptest/fault.go`:

```go
package wptest

import (
	"net/http"
	"strconv"
	"time"
)

type Redirect string

const (
	RedirectNone  Redirect = ""
	RedirectHTTPS Redirect = "https"
	RedirectLogin Redirect = "login"
	RedirectAdmin Redirect = "admin"
)

type fault struct {
	retryAfter time.Duration
	status     int
}

func (s *Server) FailNext(status, times int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for range times {
		s.faults = append(s.faults, fault{status: status})
	}
}

func (s *Server) RateLimitNext(retryAfter time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.faults = append(s.faults, fault{status: http.StatusTooManyRequests, retryAfter: retryAfter})
}

func (s *Server) takeFault() (fault, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.faults) == 0 {
		return fault{}, false
	}
	next := s.faults[0]
	s.faults = s.faults[1:]
	return next, true
}

func (s *Server) injectFaults(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		injected, ok := s.takeFault()
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		if injected.retryAfter > 0 {
			w.Header().Set("Retry-After", strconv.Itoa(int(injected.retryAfter.Seconds())))
		}
		s.fail(w, injected.status, faultCode(injected.status), "the injected fault fired")
	})
}

func faultCode(status int) string {
	if status == http.StatusTooManyRequests {
		return "too_many_requests"
	}
	return "internal_server_error"
}

func redirectTarget(mode Redirect, host, base string) (string, int) {
	switch mode {
	case RedirectHTTPS:
		return "https://" + host + rootPath, http.StatusMovedPermanently
	case RedirectLogin:
		return base + "/wp-login.php?redirect_to=%2Fwp-json", http.StatusFound
	case RedirectAdmin:
		return base + "/wp-admin/", http.StatusFound
	default:
		return "", 0
	}
}
```

Create `internal/adapters/wp/wptest/server.go`:

```go
package wptest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"sync"
	"testing"
	"time"
)

const (
	DefaultUser     = "postulator"
	DefaultPassword = "abcd EFGH 1234 ijkl"

	rootPath            = "/wp-json"
	pluginNamespaceName = "postulator/v1"

	maxRequestBytes = 32 << 20
)

type Request struct {
	Query  url.Values
	Header http.Header
	Method string
	Path   string
	Body   []byte
}

type Server struct {
	t             testing.TB
	http          *httptest.Server
	items         map[int64]*Item
	categories    map[int64]*Category
	uploads       map[int64]*upload
	order         []int64
	categoryOrder []int64
	uploadOrder   []int64
	requests      []Request
	faults        []fault
	clock         time.Time
	user          string
	password      string
	seoPlugin     string
	redirect      Redirect
	nextID        int64
	noPlugin      bool
	noNamespaces  bool
	mu            sync.Mutex
}

type Option func(*Server)

func WithCredentials(user, password string) Option {
	return func(s *Server) {
		s.user = user
		s.password = password
	}
}

func WithSEOPlugin(name string) Option {
	return func(s *Server) { s.seoPlugin = name }
}

func WithoutPlugin() Option {
	return func(s *Server) { s.noPlugin = true }
}

func WithoutNamespaces() Option {
	return func(s *Server) { s.noNamespaces = true }
}

func WithRedirect(mode Redirect) Option {
	return func(s *Server) { s.redirect = mode }
}

func New(t testing.TB, opts ...Option) *Server {
	t.Helper()

	server := &Server{
		t:          t,
		items:      make(map[int64]*Item),
		categories: make(map[int64]*Category),
		uploads:    make(map[int64]*upload),
		clock:      startInstant.Add(-time.Second),
		user:       DefaultUser,
		password:   DefaultPassword,
		seoPlugin:  "yoast",
	}
	for _, opt := range opts {
		opt(server)
	}

	server.http = httptest.NewServer(server.handler())
	t.Cleanup(server.http.Close)
	return server
}

func (s *Server) URL() string {
	return s.http.URL
}

func (s *Server) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+rootPath, s.handleRoot)

	return s.record(s.redirectRoot(s.injectFaults(s.authenticate(mux))))
}

func (s *Server) record(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBytes))
		if err != nil {
			s.fail(w, http.StatusBadRequest, "rest_invalid_body", "the request body could not be read")
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))

		s.mu.Lock()
		s.requests = append(s.requests, Request{
			Method: r.Method,
			Path:   r.URL.Path,
			Query:  r.URL.Query(),
			Header: r.Header.Clone(),
			Body:   body,
		})
		s.mu.Unlock()

		next.ServeHTTP(w, r)
	})
}

func (s *Server) redirectRoot(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		mode := s.redirect
		s.mu.Unlock()

		location, status := redirectTarget(mode, r.Host, s.http.URL)
		if status == 0 || r.URL.Path != rootPath {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Location", location)
		w.WriteHeader(status)
	})
}

func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, password, ok := r.BasicAuth()

		s.mu.Lock()
		wantUser, wantPassword := s.user, s.password
		s.mu.Unlock()

		if !ok || user != wantUser || password != wantPassword {
			s.fail(w, http.StatusUnauthorized, "rest_not_logged_in", "You are not currently logged in.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleRoot(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	namespaces := []string{"oembed/1.0", "wp/v2", "wc/v3"}
	if !s.noPlugin {
		namespaces = append(namespaces, pluginNamespaceName)
	}
	if s.noNamespaces {
		namespaces = []string{}
	}
	s.mu.Unlock()

	s.respond(w, http.StatusOK, map[string]any{
		"name":        "Postulator Test Site",
		"description": "a fake WordPress installation",
		"url":         s.http.URL,
		"home":        s.http.URL,
		"namespaces":  namespaces,
	})
}

func (s *Server) respond(w http.ResponseWriter, status int, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		s.t.Errorf("encode the fake response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(status)
	if _, err = w.Write(body); err != nil {
		s.t.Logf("write the fake response: %v", err)
	}
}

func (s *Server) fail(w http.ResponseWriter, status int, code, message string) {
	s.respond(w, status, map[string]string{"code": code, "message": message})
}

func (s *Server) Requests() []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.requests)
}

func (s *Server) LastRequest() (Request, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.requests) == 0 {
		return Request{}, false
	}
	return s.requests[len(s.requests)-1], true
}

func (s *Server) ResetRequests() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = nil
}
```

`handler()` registers only the routes this task fully serves. Tasks 5, 6 and 10 add their `s.routeX(mux)` lines to this same function together with the files that define them, so an empty registration function never exists. Each of those tasks also introduces its own namespace constant; this task declares only `rootPath` and `pluginNamespaceName`.

- [x] **Step 4: Run the tests and watch them pass**

Run: `go test -count=1 -race ./internal/adapters/wp/...`
Expected: PASS.

- [x] **Step 5: Run the gate**

```bash
go build ./... && go vet ./... && "$(go env GOPATH)/bin/golangci-lint.exe" run && go test -count=1 -race ./...
```
Expected: all green.

- [x] **Step 6: Commit**

```bash
git add internal/adapters/wp/wptest
git commit -m "test(wp): fake wordpress server with state, faults and redirects

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---
### Task 4: The request pipeline, error classification and `Probe`

**Files:**
- Create: `internal/adapters/wp/do.go`, `internal/adapters/wp/errors.go`, `internal/adapters/wp/probe.go`
- Test: `internal/adapters/wp/errors_test.go` (package `wp`), `internal/adapters/wp/do_test.go` (package `wp_test`), `internal/adapters/wp/probe_test.go` (package `wp_test`)

**Interfaces:**
- Consumes: task 2's `Client`, `resolve`, `Logger`, `backoff`, `limiter`, `retries`; task 3's `wptest.Server`.
- Produces:
  - `type request struct { header http.Header; query url.Values; client *http.Client; method, namespace, path, contentType string; body []byte; keepRedirect bool }`
  - `func (c *Client) do(ctx context.Context, req request) (*http.Response, []byte, error)`
  - `func decodeJSON(body []byte, out any) error`
  - `func classify(resp *http.Response, body []byte) error`, `func transportError(ctx context.Context, err error) error`, `func retryAfter(header string) time.Duration`, `func retryable(err error) bool`, `func delayFor(err error, fallback time.Duration) time.Duration`, `func detailString(err error, key string) string`
  - `type ProbeStatus string` with `ProbeOK` and `ProbeUpgradeRequired`
  - `type ProbeResult struct { Status ProbeStatus; Namespaces, Warnings []string; SiteName, Description, HomeURL, SuggestedBaseURL string; HasPlugin, HasWoo bool }`
  - `func (c *Client) Probe(ctx context.Context) (ProbeResult, error)`
  - the shared test helper `newClient(t *testing.T, server *wptest.Server, opts ...wp.Option) *wp.Client`

- [x] **Step 1: Write the failing tests**

Create `internal/adapters/wp/errors_test.go`:

```go
package wp

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func response(t *testing.T, status int, header http.Header) *http.Response {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.com/wp-json/wp/v2/pages", nil)
	if err != nil {
		t.Fatalf("build the request: %v", err)
	}
	if header == nil {
		header = http.Header{}
	}
	return &http.Response{StatusCode: status, Header: header, Request: request}
}

func TestClassifyMapsWordPressStatusesToKernelCodes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		status int
		header http.Header
		body   string
		want   errors.Code
	}{
		{name: "bad request", status: http.StatusBadRequest, body: `{"code":"rest_invalid_param","message":"Invalid parameter(s): slug"}`, want: errors.Invalid},
		{name: "unauthorized", status: http.StatusUnauthorized, body: `{"code":"rest_not_logged_in","message":"You are not currently logged in."}`, want: errors.Unauthorized},
		{name: "forbidden", status: http.StatusForbidden, body: `{"code":"rest_forbidden"}`, want: errors.Unauthorized},
		{name: "not found", status: http.StatusNotFound, body: `{"code":"rest_post_invalid_id"}`, want: errors.NotFound},
		{name: "conflict", status: http.StatusConflict, body: `{"code":"hash_mismatch","currentHash":"abc"}`, want: errors.Conflict},
		{name: "rate limited", status: http.StatusTooManyRequests, header: http.Header{"Retry-After": {"5"}}, body: `{"code":"too_many_requests"}`, want: errors.RateLimited},
		{name: "server error", status: http.StatusInternalServerError, body: "<html>fatal</html>", want: errors.External},
		{name: "gateway error", status: http.StatusBadGateway, body: "", want: errors.External},
		{name: "unexpected redirect", status: http.StatusFound, body: "", want: errors.External},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := classify(response(t, tc.status, tc.header), []byte(tc.body))
			if !errors.IsCode(err, tc.want) {
				t.Fatalf("code = %q, want %q", errors.CodeOf(err), tc.want)
			}
			if got, ok := detail(err, "status"); !ok || got != tc.status {
				t.Errorf("status detail = %v, %t", got, ok)
			}
			if got := detailString(err, "path"); got != "/wp-json/wp/v2/pages" {
				t.Errorf("path detail = %q", got)
			}
		})
	}
}

func detail(err error, key string) (int, bool) {
	value, ok := detailValue(err, key)
	if !ok {
		return 0, false
	}
	number, ok := value.(int)
	return number, ok
}

func TestClassifyCarriesTheWordPressCodeAndConflictHash(t *testing.T) {
	t.Parallel()

	invalid := classify(response(t, http.StatusBadRequest, nil), []byte(`{"code":"rest_invalid_param","message":"Invalid parameter(s): slug"}`))
	if got := detailString(invalid, "code"); got != "rest_invalid_param" {
		t.Errorf("code detail = %q", got)
	}
	if got := detailString(invalid, "wpMessage"); got != "Invalid parameter(s): slug" {
		t.Errorf("wpMessage detail = %q", got)
	}

	conflict := classify(response(t, http.StatusConflict, nil), []byte(`{"code":"hash_mismatch","currentHash":"deadbeef"}`))
	if got := detailString(conflict, "currentHash"); got != "deadbeef" {
		t.Errorf("currentHash detail = %q", got)
	}
}

func TestClassifyAttachesRetryInformation(t *testing.T) {
	t.Parallel()

	limited := classify(response(t, http.StatusTooManyRequests, http.Header{"Retry-After": {"5"}}), nil)
	if got := delayFor(limited, time.Minute); got != 5*time.Second {
		t.Errorf("retry = %s, want 5s", got)
	}

	failed := classify(response(t, http.StatusServiceUnavailable, nil), nil)
	if got := delayFor(failed, 250*time.Millisecond); got != 250*time.Millisecond {
		t.Errorf("retry = %s, want the fallback backoff", got)
	}
}

func TestRetryAfterUnderstandsSecondsAndDates(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		header string
		want   time.Duration
	}{
		{name: "absent", header: "", want: 0},
		{name: "seconds", header: "7", want: 7 * time.Second},
		{name: "padded seconds", header: " 7 ", want: 7 * time.Second},
		{name: "negative", header: "-7", want: 0},
		{name: "rubbish", header: "soon", want: 0},
		{name: "date in the past", header: "Wed, 21 Oct 2015 07:28:00 GMT", want: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := retryAfter(tc.header); got != tc.want {
				t.Errorf("retryAfter(%q) = %s, want %s", tc.header, got, tc.want)
			}
		})
	}
}

func TestRetryAfterUnderstandsAFutureDate(t *testing.T) {
	t.Parallel()

	header := time.Now().Add(time.Minute).UTC().Format(http.TimeFormat)
	if got := retryAfter(header); got <= 0 || got > time.Minute {
		t.Errorf("retryAfter(%q) = %s, want a positive delay of at most a minute", header, got)
	}
}

func TestOnlyTransientCodesAreRetried(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		code errors.Code
		want bool
	}{
		{name: "rate limited", code: errors.RateLimited, want: true},
		{name: "external", code: errors.External, want: true},
		{name: "unauthorized", code: errors.Unauthorized},
		{name: "not found", code: errors.NotFound},
		{name: "conflict", code: errors.Conflict},
		{name: "invalid", code: errors.Invalid},
		{name: "cancelled", code: errors.Cancelled},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := retryable(errors.New(tc.code, "x")); got != tc.want {
				t.Errorf("retryable(%s) = %t, want %t", tc.code, got, tc.want)
			}
		})
	}
}

func TestATransportFailureRespectsTheCallersContext(t *testing.T) {
	t.Parallel()

	live := transportError(t.Context(), http.ErrHandlerTimeout)
	if !errors.IsCode(live, errors.External) {
		t.Errorf("code = %q, want %q", errors.CodeOf(live), errors.External)
	}
	if !retryable(live) {
		t.Error("our own timeout must be retryable")
	}

	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	if got := transportError(cancelled, http.ErrHandlerTimeout); !errors.IsCode(got, errors.Cancelled) {
		t.Errorf("code = %q, want %q", errors.CodeOf(got), errors.Cancelled)
	}
}
```

Create `internal/adapters/wp/do_test.go`:

```go
package wp_test

import (
	"context"
	stderrors "errors"
	"net/http"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func newClient(t *testing.T, server *wptest.Server, opts ...wp.Option) *wp.Client {
	t.Helper()

	options := append([]wp.Option{
		wp.WithRateLimit(0),
		wp.WithBackoff(func(int) time.Duration { return 0 }),
	}, opts...)

	client, err := wp.New(wp.Config{
		BaseURL:       server.URL(),
		Username:      wptest.DefaultUser,
		AppPassword:   wptest.DefaultPassword,
		AllowInsecure: true,
	}, options...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return client
}

func TestEveryRequestCarriesBasicAuthAndOurHeaders(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	if _, err := newClient(t, server).Probe(t.Context()); err != nil {
		t.Fatalf("Probe: %v", err)
	}

	recorded, ok := server.LastRequest()
	if !ok {
		t.Fatal("no request reached the site")
	}

	user, password, present := parseBasicAuth(t, recorded.Header.Get("Authorization"))
	if !present || user != wptest.DefaultUser || password != wptest.DefaultPassword {
		t.Errorf("basic auth = %q %q %t", user, password, present)
	}
	if got := recorded.Header.Get("Accept"); got != "application/json" {
		t.Errorf("Accept = %q", got)
	}
	if got := recorded.Header.Get("User-Agent"); got == "" {
		t.Error("the request must identify Postulator")
	}
}

func parseBasicAuth(t *testing.T, header string) (string, string, bool) {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.com", nil)
	if err != nil {
		t.Fatalf("build the request: %v", err)
	}
	request.Header.Set("Authorization", header)
	return request.BasicAuth()
}

func TestATransientFailureIsRetriedUntilItSucceeds(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.FailNext(http.StatusServiceUnavailable, 2)

	if _, err := newClient(t, server).Probe(t.Context()); err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if got := len(server.Requests()); got != 3 {
		t.Errorf("the site saw %d requests, want 3", got)
	}
}

func TestTheRetryBudgetIsFinite(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.FailNext(http.StatusServiceUnavailable, 10)

	_, err := newClient(t, server, wp.WithRetries(2)).Probe(t.Context())
	if !errors.IsCode(err, errors.External) {
		t.Fatalf("code = %q, want %q", errors.CodeOf(err), errors.External)
	}
	if got := len(server.Requests()); got != 3 {
		t.Errorf("the site saw %d requests, want 1 attempt and 2 retries", got)
	}
}

func TestANonTransientFailureIsNotRetried(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.FailNext(http.StatusForbidden, 3)

	_, err := newClient(t, server).Probe(t.Context())
	if !errors.IsCode(err, errors.Unauthorized) {
		t.Fatalf("code = %q, want %q", errors.CodeOf(err), errors.Unauthorized)
	}
	if got := len(server.Requests()); got != 1 {
		t.Errorf("the site saw %d requests, want 1", got)
	}
}

func TestRetryAfterIsReportedOnTheError(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.RateLimitNext(3 * time.Second)

	_, err := newClient(t, server, wp.WithRetries(0)).Probe(t.Context())
	if !errors.IsCode(err, errors.RateLimited) {
		t.Fatalf("code = %q, want %q", errors.CodeOf(err), errors.RateLimited)
	}

	var kernel *errors.Error
	if !stderrors.As(err, &kernel) || kernel.Retry == nil || kernel.Retry.After != 3*time.Second {
		t.Errorf("retry = %+v, want 3s", kernel)
	}
}

func TestRetryAfterIsWaitedForRatherThanTheBackoff(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.RateLimitNext(5 * time.Second)

	ctx, cancel := context.WithTimeout(t.Context(), 300*time.Millisecond)
	defer cancel()

	_, err := newClient(t, server, wp.WithRetries(1)).Probe(ctx)
	if !errors.IsCode(err, errors.Cancelled) {
		t.Fatalf("code = %q, want %q; the client did not wait out Retry-After", errors.CodeOf(err), errors.Cancelled)
	}
	if got := len(server.Requests()); got != 1 {
		t.Errorf("the site saw %d requests, want 1", got)
	}
}

func TestACancelledContextStopsTheCall(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := newClient(t, server).Probe(ctx)
	if !errors.IsCode(err, errors.Cancelled) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Cancelled)
	}
}
```

Create `internal/adapters/wp/probe_test.go`:

```go
package wp_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestProbeReportsAHealthySite(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	result, err := newClient(t, server).Probe(t.Context())
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}

	if result.Status != wp.ProbeOK {
		t.Errorf("status = %q, want %q", result.Status, wp.ProbeOK)
	}
	if !slices.Contains(result.Namespaces, "wp/v2") {
		t.Errorf("namespaces = %v", result.Namespaces)
	}
	if !result.HasPlugin {
		t.Error("the fake advertises postulator/v1")
	}
	if !result.HasWoo {
		t.Error("the fake advertises wc/v3")
	}
	if result.SiteName == "" || result.HomeURL == "" {
		t.Errorf("site = %q, home = %q", result.SiteName, result.HomeURL)
	}
	if len(result.Warnings) != 0 {
		t.Errorf("warnings = %v, want none", result.Warnings)
	}
}

func TestProbeReportsAMissingPlugin(t *testing.T) {
	t.Parallel()

	server := wptest.New(t, wptest.WithoutPlugin())
	result, err := newClient(t, server).Probe(t.Context())
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if result.HasPlugin {
		t.Error("the plugin namespace is not advertised")
	}
}

func TestProbeClassifiesAFailingSite(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		options []wptest.Option
		want    errors.Code
	}{
		{name: "no namespaces at all", options: []wptest.Option{wptest.WithoutNamespaces()}, want: errors.External},
		{name: "redirected to the login page", options: []wptest.Option{wptest.WithRedirect(wptest.RedirectLogin)}, want: errors.Unauthorized},
		{name: "redirected to the admin area", options: []wptest.Option{wptest.WithRedirect(wptest.RedirectAdmin)}, want: errors.Unauthorized},
		{name: "credentials rejected", options: []wptest.Option{wptest.WithCredentials("other", "pass word")}, want: errors.Unauthorized},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := wptest.New(t, tc.options...)
			_, err := newClient(t, server).Probe(t.Context())
			if !errors.IsCode(err, tc.want) {
				t.Errorf("code = %q, want %q", errors.CodeOf(err), tc.want)
			}
		})
	}
}

func TestProbeReportsAnHTTPSUpgradeAsAWarningRatherThanAFailure(t *testing.T) {
	t.Parallel()

	server := wptest.New(t, wptest.WithRedirect(wptest.RedirectHTTPS))
	result, err := newClient(t, server).Probe(t.Context())
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}

	if result.Status != wp.ProbeUpgradeRequired {
		t.Errorf("status = %q, want %q", result.Status, wp.ProbeUpgradeRequired)
	}
	if len(result.Warnings) == 0 {
		t.Error("an upgrade must come with a warning the user can act on")
	}
	if result.SuggestedBaseURL == "" {
		t.Error("the warning must carry the base URL the site wants")
	}
	if got := len(server.Requests()); got != 1 {
		t.Errorf("the site saw %d requests; Probe must not follow the redirect", got)
	}
}

func TestProbeRejectsSomethingThatIsNotWordPress(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	client, err := wp.New(wp.Config{
		BaseURL:       server.URL() + "/shop",
		Username:      wptest.DefaultUser,
		AppPassword:   wptest.DefaultPassword,
		AllowInsecure: true,
	}, wp.WithRateLimit(0), wp.WithBackoff(func(int) time.Duration { return 0 }))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err = client.Probe(t.Context()); !errors.IsCode(err, errors.External) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.External)
	}
}
```

with `"time"` in the import block.

- [x] **Step 2: Run the tests and watch them fail**

Run: `go test -count=1 ./internal/adapters/wp/...`
Expected: FAIL — `undefined: classify`, `undefined: (*wp.Client).Probe`.

- [x] **Step 3: Write the error boundary**

Create `internal/adapters/wp/errors.go`:

```go
package wp

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type wpError struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	CurrentHash string `json:"currentHash"`
}

func decodeError(body []byte) wpError {
	var failure wpError
	if err := json.Unmarshal(body, &failure); err != nil {
		return wpError{}
	}
	return failure
}

func classify(resp *http.Response, body []byte) error {
	failure := decodeError(body)

	base := errors.New(errors.External, "the WordPress site returned an unexpected status").
		WithDetail("status", resp.StatusCode).
		WithDetail("method", resp.Request.Method).
		WithDetail("path", resp.Request.URL.Path)
	if failure.Code != "" {
		base = base.WithDetail("code", failure.Code)
	}
	if failure.Message != "" {
		base = base.WithDetail("wpMessage", failure.Message)
	}

	switch {
	case resp.StatusCode == http.StatusUnauthorized, resp.StatusCode == http.StatusForbidden:
		return recode(base, errors.Unauthorized, "WordPress rejected the application password")
	case resp.StatusCode == http.StatusNotFound:
		return recode(base, errors.NotFound, "WordPress has no such resource")
	case resp.StatusCode == http.StatusConflict:
		conflict := recode(base, errors.Conflict, "the WordPress content changed since it was read")
		if failure.CurrentHash != "" {
			conflict = conflict.WithDetail("currentHash", failure.CurrentHash)
		}
		return conflict
	case resp.StatusCode == http.StatusTooManyRequests:
		return recode(base, errors.RateLimited, "WordPress is rate limiting this site").
			WithRetry(retryAfter(resp.Header.Get("Retry-After")))
	case resp.StatusCode == http.StatusBadRequest:
		return recode(base, errors.Invalid, "WordPress rejected the request")
	case resp.StatusCode >= http.StatusInternalServerError:
		return recode(base, errors.External, "the WordPress site returned a server error").WithRetry(0)
	default:
		return base
	}
}

func recode(base *errors.Error, code errors.Code, message string) *errors.Error {
	next := errors.New(code, message)
	for key, value := range base.Details {
		next = next.WithDetail(key, value)
	}
	return next
}

func transportError(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return errors.New(errors.Cancelled, "the WordPress request was cancelled").WithInternal(ctx.Err())
	}
	return errors.New(errors.External, "the WordPress site could not be reached").WithInternal(err).WithRetry(0)
}

func retryAfter(header string) time.Duration {
	trimmed := strings.TrimSpace(header)
	if trimmed == "" {
		return 0
	}

	if seconds, err := strconv.Atoi(trimmed); err == nil {
		if seconds <= 0 {
			return 0
		}
		return time.Duration(seconds) * time.Second
	}

	when, err := http.ParseTime(trimmed)
	if err != nil {
		return 0
	}
	delay := time.Until(when)
	if delay <= 0 {
		return 0
	}
	return delay
}

func retryable(err error) bool {
	code := errors.CodeOf(err)
	return code == errors.RateLimited || code == errors.External
}

func delayFor(err error, fallback time.Duration) time.Duration {
	var kernel *errors.Error
	if stderrors.As(err, &kernel) && kernel != nil && kernel.Retry != nil && kernel.Retry.After > 0 {
		return kernel.Retry.After
	}
	return fallback
}

func detailValue(err error, key string) (any, bool) {
	var kernel *errors.Error
	if !stderrors.As(err, &kernel) || kernel == nil {
		return nil, false
	}
	value, ok := kernel.Details[key]
	return value, ok
}

func detailString(err error, key string) string {
	value, ok := detailValue(err, key)
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return text
}
```

- [x] **Step 4: Write the pipeline**

Create `internal/adapters/wp/do.go`:

```go
package wp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type request struct {
	header       http.Header
	query        url.Values
	client       *http.Client
	method       string
	namespace    string
	path         string
	contentType  string
	body         []byte
	keepRedirect bool
}

type attempt struct {
	resp   *http.Response
	body   []byte
	err    error
	status int
}

func (c *Client) do(ctx context.Context, req request) (*http.Response, []byte, error) {
	target := c.resolve(req.namespace, req.path, req.query)
	client := req.client
	if client == nil {
		client = c.http
	}

	var last error
	for try := 0; try <= c.retries; try++ {
		if try > 0 {
			if err := c.pause(ctx, delayFor(last, c.backoff(try-1))); err != nil {
				return nil, nil, err
			}
		}
		if err := c.limiter.Wait(ctx); err != nil {
			return nil, nil, errors.New(errors.Cancelled, "the WordPress request was cancelled while waiting for the site rate limit").WithInternal(err)
		}

		started := time.Now()
		result := c.send(ctx, client, req, target)
		c.logAttempt(req, result, time.Since(started))

		if result.err == nil {
			return result.resp, result.body, nil
		}
		if !retryable(result.err) {
			return nil, nil, result.err
		}
		last = result.err
	}
	return nil, nil, last
}

func (c *Client) send(ctx context.Context, client *http.Client, req request, target string) attempt {
	var reader io.Reader
	if len(req.body) > 0 {
		reader = bytes.NewReader(req.body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.method, target, reader)
	if err != nil {
		return attempt{err: errors.New(errors.Invalid, "the WordPress request could not be built").WithInternal(err)}
	}

	httpReq.SetBasicAuth(c.username, c.password)
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", userAgent)
	if req.contentType != "" {
		httpReq.Header.Set("Content-Type", req.contentType)
	}
	for key, values := range req.header {
		for _, value := range values {
			httpReq.Header.Add(key, value)
		}
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return attempt{err: transportError(ctx, err)}
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return attempt{status: resp.StatusCode, err: transportError(ctx, err)}
	}

	if resp.StatusCode >= http.StatusMultipleChoices {
		if req.keepRedirect && resp.StatusCode < http.StatusBadRequest {
			return attempt{resp: resp, body: body, status: resp.StatusCode}
		}
		return attempt{status: resp.StatusCode, err: classify(resp, body)}
	}
	return attempt{resp: resp, body: body, status: resp.StatusCode}
}

func (c *Client) pause(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return errors.New(errors.Cancelled, "the WordPress request was cancelled before it could be retried").WithInternal(ctx.Err())
	case <-timer.C:
		return nil
	}
}

func (c *Client) logAttempt(req request, result attempt, took time.Duration) {
	fields := []zap.Field{
		zap.String("method", req.method),
		zap.String("path", req.namespace+req.path),
		zap.Int("status", result.status),
		zap.Int64("durationMs", took.Milliseconds()),
	}

	if result.err == nil {
		c.logger.Debug("wordpress request", fields...)
		return
	}

	fields = append(fields, zap.String("code", errors.CodeOf(result.err).String()))
	if wpCode := detailString(result.err, "code"); wpCode != "" {
		fields = append(fields, zap.String("wpCode", wpCode))
	}
	c.logger.Warn("wordpress request failed", fields...)
}

func decodeJSON(body []byte, out any) error {
	if err := json.Unmarshal(body, out); err != nil {
		return errors.New(errors.External, "the WordPress response is not valid JSON").WithInternal(err)
	}
	return nil
}
```

- [x] **Step 5: Write `Probe`**

Create `internal/adapters/wp/probe.go`:

```go
package wp

import (
	"context"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type ProbeStatus string

const (
	ProbeOK              ProbeStatus = "ok"
	ProbeUpgradeRequired ProbeStatus = "upgrade_required"
)

type ProbeResult struct {
	Status           ProbeStatus
	Namespaces       []string
	Warnings         []string
	SiteName         string
	Description      string
	HomeURL          string
	SuggestedBaseURL string
	HasPlugin        bool
	HasWoo           bool
}

type restRoot struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Home        string   `json:"home"`
	Namespaces  []string `json:"namespaces"`
}

func (c *Client) Probe(ctx context.Context) (ProbeResult, error) {
	resp, body, err := c.do(ctx, request{
		method:       http.MethodGet,
		namespace:    rootPath,
		client:       c.probe,
		keepRedirect: true,
	})
	if err != nil {
		if errors.IsCode(err, errors.NotFound) {
			return ProbeResult{}, errors.New(errors.External, "the site has no WordPress REST API at /wp-json").WithInternal(err)
		}
		return ProbeResult{}, err
	}

	if resp.StatusCode >= http.StatusMultipleChoices {
		return c.classifyRedirect(resp)
	}

	var root restRoot
	if err = decodeJSON(body, &root); err != nil {
		return ProbeResult{}, errors.New(errors.External, "the REST root is not a WordPress REST index").WithInternal(err)
	}
	if len(root.Namespaces) == 0 {
		return ProbeResult{}, errors.New(errors.External, "the REST root reported no namespaces, so this is not a WordPress site")
	}

	return ProbeResult{
		Status:      ProbeOK,
		Namespaces:  root.Namespaces,
		SiteName:    root.Name,
		Description: root.Description,
		HomeURL:     root.Home,
		HasPlugin:   slices.Contains(root.Namespaces, "postulator/v1"),
		HasWoo:      slices.Contains(root.Namespaces, "wc/v3"),
	}, nil
}

func (c *Client) classifyRedirect(resp *http.Response) (ProbeResult, error) {
	location := resp.Header.Get("Location")
	if location == "" {
		return ProbeResult{}, errors.New(errors.External, "the REST root redirected without a location").
			WithDetail("status", resp.StatusCode)
	}

	parsed, err := url.Parse(location)
	if err != nil {
		return ProbeResult{}, errors.New(errors.External, "the REST root redirected to an unreadable location").WithInternal(err)
	}
	target := resp.Request.URL.ResolveReference(parsed)

	if loginRedirect(target.Path) {
		return ProbeResult{}, errors.New(errors.Unauthorized, "the site redirected the REST root to the WordPress login page").
			WithDetail("location", target.Path)
	}

	if target.Scheme == "https" && c.base.Scheme == "http" && strings.EqualFold(target.Host, c.base.Host) {
		return ProbeResult{
			Status:           ProbeUpgradeRequired,
			SuggestedBaseURL: "https://" + c.base.Host + c.base.Path,
			Warnings:         []string{"the site redirects http to https; store the site base URL with the https scheme"},
		}, nil
	}

	return ProbeResult{}, errors.New(errors.External, "the REST root redirected somewhere unexpected").
		WithDetail("location", target.String())
}

func loginRedirect(path string) bool {
	return strings.Contains(path, "wp-login.php") || strings.Contains(path, "/wp-admin")
}
```

- [x] **Step 6: Run the tests and watch them pass**

Run: `go test -count=1 -race ./internal/adapters/wp/...`
Expected: PASS.

- [x] **Step 7: Run the gate**

```bash
go build ./... && go vet ./... && "$(go env GOPATH)/bin/golangci-lint.exe" run && go test -count=1 -race ./...
```
Expected: all green.

- [x] **Step 8: Commit**

```bash
git add internal/adapters/wp
git commit -m "feat(wp): request pipeline, error mapping and site probe

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---
### Task 5: The fake server speaks core REST for pages and posts

**Files:**
- Create: `internal/adapters/wp/wptest/core.go`
- Modify: `internal/adapters/wp/wptest/server.go` (add `s.routeCore(mux)` to `handler`)
- Test: `internal/adapters/wp/wptest/core_test.go` (package `wptest_test`)

**Interfaces:**
- Consumes: task 3's store, `s.respond`, `s.fail`, `s.add`, `s.tick`, `s.uniqueSlug`, `s.itemPath`.
- Produces:
  - `func (s *Server) routeCore(mux *http.ServeMux)` registering `GET|POST /wp-json/wp/v2/{pages,posts}` and `GET|POST|DELETE /wp-json/wp/v2/{pages,posts}/{id}`
  - `const coreNamespace = "/wp-json/wp/v2"`
  - `func (s *Server) itemPayload(stored *Item) map[string]any`, `func narrowFields(item map[string]any, fields string) map[string]any`, `func pathID(r *http.Request) (int64, bool)`, `func splitList(value string) []string`, `func parseQueryTime(value string) time.Time`, `func stringField`, `func intField`, `func intListField`, `func metaField`

- [x] **Step 1: Write the failing test**

Create `internal/adapters/wp/wptest/core_test.go`:

```go
package wptest_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
)

func seedPages(t *testing.T, server *wptest.Server, count int) []wptest.Item {
	t.Helper()

	items := make([]wptest.Item, 0, count)
	for index := range count {
		items = append(items, wptest.Item{Type: wptest.TypePage, Title: "Page " + string(rune('A'+index)), Content: "<p>body</p>"})
	}
	return server.Seed(items...)
}

func TestListReportsTheWordPressPagingHeaders(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seedPages(t, server, 5)

	response, payload := call(t, server, http.MethodGet, "/wp-json/wp/v2/pages?page=2&per_page=2", nil, true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
	if got := response.Header.Get("X-WP-Total"); got != "5" {
		t.Errorf("X-WP-Total = %q, want 5", got)
	}
	if got := response.Header.Get("X-WP-TotalPages"); got != "3" {
		t.Errorf("X-WP-TotalPages = %q, want 3", got)
	}

	var items []map[string]any
	decode(t, payload, &items)
	if len(items) != 2 {
		t.Errorf("page 2 holds %d items, want 2", len(items))
	}
}

func TestAPageNumberPastTheEndIsAFourHundred(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seedPages(t, server, 2)

	response, payload := call(t, server, http.MethodGet, "/wp-json/wp/v2/pages?page=9&per_page=2", nil, true)
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.StatusCode)
	}

	var failure struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	decode(t, payload, &failure)
	if failure.Code != "rest_post_invalid_page_number" {
		t.Errorf("code = %q", failure.Code)
	}
}

func TestListFiltersByStatusAndModification(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := server.Seed(
		wptest.Item{Type: wptest.TypePage, Title: "Draft", Status: "draft"},
		wptest.Item{Type: wptest.TypePage, Title: "Live", Status: "publish"},
	)

	_, payload := call(t, server, http.MethodGet, "/wp-json/wp/v2/pages?status=draft", nil, true)
	var drafts []map[string]any
	decode(t, payload, &drafts)
	if len(drafts) != 1 {
		t.Fatalf("status filter returned %d items, want 1", len(drafts))
	}

	cut := seeded[0].Modified.UTC().Format("2006-01-02T15:04:05")
	_, payload = call(t, server, http.MethodGet, "/wp-json/wp/v2/pages?modified_after="+cut, nil, true)
	var recent []map[string]any
	decode(t, payload, &recent)
	if len(recent) != 1 {
		t.Errorf("modified_after returned %d items, want 1; the bound is exclusive", len(recent))
	}
}

func TestFieldsNarrowsThePayload(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seedPages(t, server, 1)

	_, payload := call(t, server, http.MethodGet, "/wp-json/wp/v2/pages?_fields=id,slug", nil, true)
	var items []map[string]any
	decode(t, payload, &items)
	if len(items) != 1 {
		t.Fatalf("got %d items", len(items))
	}
	if len(items[0]) != 2 {
		t.Errorf("item = %v, want only id and slug", items[0])
	}
}

func TestAnItemCarriesRawAndRenderedTextAndAnEmptyMetaArray(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein", Content: "<p>Koffein ist ein Alkaloid.</p>"})

	_, payload := call(t, server, http.MethodGet, "/wp-json/wp/v2/pages/"+itoa(seeded[0].ID), nil, true)

	var item struct {
		Title struct {
			Raw      string `json:"raw"`
			Rendered string `json:"rendered"`
		} `json:"title"`
		ModifiedGMT string          `json:"modified_gmt"`
		Meta        json.RawMessage `json:"meta"`
	}
	decode(t, payload, &item)

	if item.Title.Raw != "Koffein" || item.Title.Rendered != "Koffein" {
		t.Errorf("title = %+v", item.Title)
	}
	if len(item.ModifiedGMT) != 19 {
		t.Errorf("modified_gmt = %q, want a WordPress timestamp with no offset", item.ModifiedGMT)
	}
	if string(item.Meta) != "[]" {
		t.Errorf("meta = %s, want the empty array WordPress actually sends", item.Meta)
	}
}

func TestAnUnknownItemIsAFourOhFour(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	response, _ := call(t, server, http.MethodGet, "/wp-json/wp/v2/pages/404", nil, true)
	if response.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", response.StatusCode)
	}
}

func TestCreateRewritesACollidingSlug(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Powder"})

	response, payload := call(t, server, http.MethodPost, "/wp-json/wp/v2/pages", []byte(`{"title":"Powder","content":"<p>x</p>","slug":"powder","status":"draft"}`), true)
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", response.StatusCode)
	}

	var created struct {
		ID     int64  `json:"id"`
		Slug   string `json:"slug"`
		Link   string `json:"link"`
		Status string `json:"status"`
	}
	decode(t, payload, &created)
	if created.Slug != "powder-2" {
		t.Errorf("slug = %q, want powder-2", created.Slug)
	}
	if created.Status != "draft" {
		t.Errorf("status = %q, want draft", created.Status)
	}
	if created.Link == "" {
		t.Error("a created item must carry its permalink")
	}
}

func TestUpdateAppliesOnlyThePresentFields(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	parent := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein"})[0]
	child := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Powder", Parent: parent.ID, Status: "draft"})[0]

	_, payload := call(t, server, http.MethodPost, "/wp-json/wp/v2/pages/"+itoa(child.ID), []byte(`{"status":"publish"}`), true)
	var updated struct {
		Status string `json:"status"`
		Parent int64  `json:"parent"`
		Title  struct {
			Raw string `json:"raw"`
		} `json:"title"`
	}
	decode(t, payload, &updated)
	if updated.Status != "publish" || updated.Parent != parent.ID || updated.Title.Raw != "Powder" {
		t.Errorf("update = %+v, want only the status changed", updated)
	}

	_, payload = call(t, server, http.MethodPost, "/wp-json/wp/v2/pages/"+itoa(child.ID), []byte(`{"parent":0}`), true)
	decode(t, payload, &updated)
	if updated.Parent != 0 {
		t.Errorf("parent = %d, want 0; zero is a value, not an absence", updated.Parent)
	}
}

func TestDeleteTrashesUnlessItIsForced(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := seedPages(t, server, 2)

	_, payload := call(t, server, http.MethodDelete, "/wp-json/wp/v2/pages/"+itoa(seeded[0].ID), nil, true)
	var trashed struct {
		Status string `json:"status"`
	}
	decode(t, payload, &trashed)
	if trashed.Status != "trash" {
		t.Errorf("status = %q, want trash", trashed.Status)
	}
	if _, ok := server.Lookup(seeded[0].ID); !ok {
		t.Error("a trashed item stays in the database")
	}

	call(t, server, http.MethodDelete, "/wp-json/wp/v2/pages/"+itoa(seeded[1].ID)+"?force=true", nil, true)
	if _, ok := server.Lookup(seeded[1].ID); ok {
		t.Error("a forced delete removes the item")
	}
}
```

Add the small helper the tests use to `server_test.go`:

```go
func itoa(id int64) string {
	return strconv.FormatInt(id, 10)
}
```

with `"strconv"` in that file's import block.

- [x] **Step 2: Run the tests and watch them fail**

Run: `go test -count=1 ./internal/adapters/wp/wptest/...`
Expected: FAIL — every core route answers 404 because nothing registers it.

- [x] **Step 3: Write the core handlers**

Create `internal/adapters/wp/wptest/core.go`:

```go
package wptest

import (
	"encoding/json"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	coreNamespace  = "/wp-json/wp/v2"
	wpTimeLayout   = "2006-01-02T15:04:05"
	defaultPerPage = 10
	maxPerPage     = 100
)

func (s *Server) routeCore(mux *http.ServeMux) {
	resources := []struct {
		path     string
		itemType string
	}{
		{path: "/pages", itemType: TypePage},
		{path: "/posts", itemType: TypePost},
	}

	for _, resource := range resources {
		mux.HandleFunc("GET "+coreNamespace+resource.path, func(w http.ResponseWriter, r *http.Request) {
			s.handleList(w, r, resource.itemType)
		})
		mux.HandleFunc("POST "+coreNamespace+resource.path, func(w http.ResponseWriter, r *http.Request) {
			s.handleCreate(w, r, resource.itemType)
		})
		mux.HandleFunc("GET "+coreNamespace+resource.path+"/{id}", func(w http.ResponseWriter, r *http.Request) {
			s.handleGet(w, r, resource.itemType)
		})
		mux.HandleFunc("POST "+coreNamespace+resource.path+"/{id}", func(w http.ResponseWriter, r *http.Request) {
			s.handleUpdate(w, r, resource.itemType)
		})
		mux.HandleFunc("DELETE "+coreNamespace+resource.path+"/{id}", func(w http.ResponseWriter, r *http.Request) {
			s.handleDelete(w, r, resource.itemType)
		})
	}
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request, itemType string) {
	query := r.URL.Query()
	page, perPage, bad := listWindow(query)
	if bad != "" {
		s.fail(w, http.StatusBadRequest, "rest_invalid_param", "Invalid parameter(s): "+bad)
		return
	}

	s.mu.Lock()
	matched := s.filter(itemType, query)
	total := len(matched)
	totalPages := (total + perPage - 1) / perPage

	if total > 0 && page > totalPages {
		s.mu.Unlock()
		s.fail(w, http.StatusBadRequest, "rest_post_invalid_page_number", "The page number requested is larger than the number of pages available.")
		return
	}

	start := min((page-1)*perPage, total)
	end := min(start+perPage, total)
	payload := make([]map[string]any, 0, end-start)
	for _, stored := range matched[start:end] {
		payload = append(payload, narrowFields(s.itemPayload(stored), query.Get("_fields")))
	}
	s.mu.Unlock()

	w.Header().Set("X-WP-Total", strconv.Itoa(total))
	w.Header().Set("X-WP-TotalPages", strconv.Itoa(totalPages))
	s.respond(w, http.StatusOK, payload)
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request, itemType string) {
	id, ok := pathID(r)
	if !ok {
		s.fail(w, http.StatusNotFound, "rest_post_invalid_id", "Invalid post ID.")
		return
	}

	s.mu.Lock()
	stored, found := s.items[id]
	var payload map[string]any
	if found && stored.Type == itemType {
		payload = s.itemPayload(stored)
	}
	s.mu.Unlock()

	if payload == nil {
		s.fail(w, http.StatusNotFound, "rest_post_invalid_id", "Invalid post ID.")
		return
	}
	s.respond(w, http.StatusOK, narrowFields(payload, r.URL.Query().Get("_fields")))
}

func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request, itemType string) {
	body, ok := s.decodeBody(w, r)
	if !ok {
		return
	}

	s.mu.Lock()
	created := s.add(Item{
		Type:          itemType,
		Title:         stringField(body, "title"),
		Content:       stringField(body, "content"),
		Excerpt:       stringField(body, "excerpt"),
		Slug:          stringField(body, "slug"),
		Status:        stringField(body, "status"),
		Template:      stringField(body, "template"),
		Parent:        intField(body, "parent"),
		MenuOrder:     int(intField(body, "menu_order")),
		FeaturedMedia: intField(body, "featured_media"),
		Categories:    intListField(body, "categories"),
		Tags:          intListField(body, "tags"),
		Meta:          metaField(body),
	})
	payload := s.itemPayload(s.items[created.ID])
	s.mu.Unlock()

	s.respond(w, http.StatusCreated, payload)
}

func (s *Server) handleUpdate(w http.ResponseWriter, r *http.Request, itemType string) {
	id, ok := pathID(r)
	if !ok {
		s.fail(w, http.StatusNotFound, "rest_post_invalid_id", "Invalid post ID.")
		return
	}

	body, ok := s.decodeBody(w, r)
	if !ok {
		return
	}

	s.mu.Lock()
	stored, found := s.items[id]
	if !found || stored.Type != itemType {
		s.mu.Unlock()
		s.fail(w, http.StatusNotFound, "rest_post_invalid_id", "Invalid post ID.")
		return
	}

	applyUpdate(stored, body)
	if slug := stringField(body, "slug"); slug != "" {
		stored.Slug = s.uniqueSlug(slug, stored.Type, stored.Parent, stored.ID)
	}
	stored.Modified = s.tick()
	payload := s.itemPayload(stored)
	s.mu.Unlock()

	s.respond(w, http.StatusOK, payload)
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request, itemType string) {
	id, ok := pathID(r)
	if !ok {
		s.fail(w, http.StatusNotFound, "rest_post_invalid_id", "Invalid post ID.")
		return
	}
	forced := r.URL.Query().Get("force") == "true"

	s.mu.Lock()
	stored, found := s.items[id]
	if !found || stored.Type != itemType {
		s.mu.Unlock()
		s.fail(w, http.StatusNotFound, "rest_post_invalid_id", "Invalid post ID.")
		return
	}

	var payload map[string]any
	if forced {
		payload = map[string]any{"deleted": true, "previous": s.itemPayload(stored)}
		delete(s.items, id)
		s.order = slices.DeleteFunc(s.order, func(other int64) bool { return other == id })
	} else {
		stored.Status = "trash"
		stored.Modified = s.tick()
		payload = s.itemPayload(stored)
	}
	s.mu.Unlock()

	s.respond(w, http.StatusOK, payload)
}

func (s *Server) decodeBody(w http.ResponseWriter, r *http.Request) (map[string]any, bool) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.fail(w, http.StatusBadRequest, "rest_invalid_json", "The request body is not valid JSON.")
		return nil, false
	}
	return body, true
}

func (s *Server) filter(itemType string, query url.Values) []*Item {
	statuses := splitList(query.Get("status"))
	after := parseQueryTime(query.Get("modified_after"))

	matched := make([]*Item, 0, len(s.order))
	for _, id := range s.order {
		stored := s.items[id]
		if stored.Type != itemType {
			continue
		}
		if len(statuses) > 0 && !slices.Contains(statuses, stored.Status) {
			continue
		}
		if !after.IsZero() && !stored.Modified.After(after) {
			continue
		}
		matched = append(matched, stored)
	}
	return matched
}

func (s *Server) itemPayload(stored *Item) map[string]any {
	stamp := stored.Modified.UTC().Format(wpTimeLayout)
	return map[string]any{
		"id":             stored.ID,
		"type":           stored.Type,
		"slug":           stored.Slug,
		"status":         stored.Status,
		"link":           s.http.URL + s.itemPath(stored),
		"parent":         stored.Parent,
		"menu_order":     stored.MenuOrder,
		"template":       stored.Template,
		"categories":     idList(stored.Categories),
		"tags":           idList(stored.Tags),
		"featured_media": stored.FeaturedMedia,
		"modified":       stamp,
		"modified_gmt":   stamp,
		"title":          renderedField(stored.Title),
		"content":        renderedField(stored.Content),
		"excerpt":        renderedField(stored.Excerpt),
		"meta":           metaPayload(stored.Meta),
	}
}

func applyUpdate(stored *Item, body map[string]any) {
	if value, ok := body["title"].(string); ok {
		stored.Title = value
	}
	if value, ok := body["content"].(string); ok {
		stored.Content = value
	}
	if value, ok := body["excerpt"].(string); ok {
		stored.Excerpt = value
	}
	if value, ok := body["status"].(string); ok {
		stored.Status = value
	}
	if value, ok := body["template"].(string); ok {
		stored.Template = value
	}
	if value, ok := body["parent"].(float64); ok {
		stored.Parent = int64(value)
	}
	if value, ok := body["menu_order"].(float64); ok {
		stored.MenuOrder = int(value)
	}
	if value, ok := body["featured_media"].(float64); ok {
		stored.FeaturedMedia = int64(value)
	}
	if _, ok := body["categories"]; ok {
		stored.Categories = intListField(body, "categories")
	}
	if _, ok := body["tags"]; ok {
		stored.Tags = intListField(body, "tags")
	}
	if meta := metaField(body); meta != nil {
		maps.Copy(stored.Meta, meta)
	}
}

func renderedField(value string) map[string]any {
	return map[string]any{"raw": value, "rendered": value}
}

func metaPayload(meta map[string]string) any {
	if len(meta) == 0 {
		return []any{}
	}

	payload := make(map[string]any, len(meta))
	for key, value := range meta {
		payload[key] = value
	}
	return payload
}

func idList(ids []int64) []int64 {
	if ids == nil {
		return []int64{}
	}
	return ids
}

func narrowFields(item map[string]any, fields string) map[string]any {
	wanted := splitList(fields)
	if len(wanted) == 0 {
		return item
	}

	narrow := make(map[string]any, len(wanted))
	for _, field := range wanted {
		if value, ok := item[field]; ok {
			narrow[field] = value
		}
	}
	return narrow
}

func listWindow(query url.Values) (int, int, string) {
	page := 1
	if raw := query.Get("page"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			return 0, 0, "page"
		}
		page = parsed
	}

	perPage := defaultPerPage
	if raw := query.Get("per_page"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > maxPerPage {
			return 0, 0, "per_page"
		}
		perPage = parsed
	}
	return page, perPage, ""
}

func pathID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func stringField(body map[string]any, key string) string {
	value, ok := body[key].(string)
	if !ok {
		return ""
	}
	return value
}

func intField(body map[string]any, key string) int64 {
	value, ok := body[key].(float64)
	if !ok {
		return 0
	}
	return int64(value)
}

func intListField(body map[string]any, key string) []int64 {
	raw, ok := body[key].([]any)
	if !ok {
		return nil
	}

	ids := make([]int64, 0, len(raw))
	for _, entry := range raw {
		number, valid := entry.(float64)
		if !valid {
			continue
		}
		ids = append(ids, int64(number))
	}
	return ids
}

func metaField(body map[string]any) map[string]string {
	raw, ok := body["meta"].(map[string]any)
	if !ok {
		return nil
	}

	meta := make(map[string]string, len(raw))
	for key, value := range raw {
		text, valid := value.(string)
		if !valid {
			continue
		}
		meta[key] = text
	}
	return meta
}

func splitList(value string) []string {
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	trimmed := make([]string, 0, len(parts))
	for _, part := range parts {
		cleaned := strings.TrimSpace(part)
		if cleaned != "" {
			trimmed = append(trimmed, cleaned)
		}
	}
	return trimmed
}

func parseQueryTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC()
	}
	if parsed, err := time.Parse(wpTimeLayout, value); err == nil {
		return parsed.UTC()
	}
	return time.Time{}
}
```

Extend `handler()` in `server.go` with the registration line:

```go
func (s *Server) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+rootPath, s.handleRoot)
	s.routeCore(mux)

	return s.record(s.redirectRoot(s.injectFaults(s.authenticate(mux))))
}
```

- [x] **Step 4: Run the tests and watch them pass**

Run: `go test -count=1 -race ./internal/adapters/wp/...`
Expected: PASS.

- [x] **Step 5: Run the gate and commit**

```bash
go build ./... && go vet ./... && "$(go env GOPATH)/bin/golangci-lint.exe" run && go test -count=1 -race ./...
git add internal/adapters/wp/wptest
git commit -m "test(wp): core rest pages and posts in the fake server

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---
### Task 6: The fake server speaks WooCommerce, core categories and media

**Files:**
- Create: `internal/adapters/wp/wptest/woo.go`, `internal/adapters/wp/wptest/category.go`, `internal/adapters/wp/wptest/media.go`
- Modify: `internal/adapters/wp/wptest/server.go` (three more registration lines)
- Test: `internal/adapters/wp/wptest/woo_test.go`, `internal/adapters/wp/wptest/media_test.go` (package `wptest_test`)

**Interfaces:**
- Consumes: task 3's store, task 5's `itemPayload`, `filter`, `listWindow`, `narrowFields`, `pathID`, `decodeBody`, the field helpers.
- Produces:
  - `func (s *Server) routeWoo(mux *http.ServeMux)`, `func (s *Server) routeCategories(mux *http.ServeMux)`, `func (s *Server) routeMedia(mux *http.ServeMux)`
  - `const wooNamespace = "/wp-json/wc/v3"`
  - `func (s *Server) Uploads() []Upload`, `type Upload struct { Filename, MimeType, Alt, Title string; Bytes []byte; ID int64 }`
  - `func (s *Server) Categories() []Category`

- [x] **Step 1: Write the failing tests**

Create `internal/adapters/wp/wptest/woo_test.go`:

```go
package wptest_test

import (
	"net/http"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
)

func TestProductsUseTheWooCommerceFieldNames(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := server.Seed(wptest.Item{
		Type:    wptest.TypeProduct,
		Title:   "Koffein Powder",
		Content: "<p>long</p>",
		Excerpt: "short",
		Status:  "publish",
	})

	response, payload := call(t, server, http.MethodGet, "/wp-json/wc/v3/products/"+itoa(seeded[0].ID), nil, true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}

	var product struct {
		ID               int64  `json:"id"`
		Name             string `json:"name"`
		Slug             string `json:"slug"`
		Permalink        string `json:"permalink"`
		Status           string `json:"status"`
		Description      string `json:"description"`
		ShortDescription string `json:"short_description"`
		DateModifiedGMT  string `json:"date_modified_gmt"`
	}
	decode(t, payload, &product)

	if product.Name != "Koffein Powder" || product.Description != "<p>long</p>" || product.ShortDescription != "short" {
		t.Errorf("product = %+v", product)
	}
	if product.Permalink == "" || product.DateModifiedGMT == "" || product.Slug != "koffein-powder" {
		t.Errorf("product = %+v", product)
	}
}

func TestProductsPageWithoutTheCoreEndOfListError(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.Seed(
		wptest.Item{Type: wptest.TypeProduct, Title: "One"},
		wptest.Item{Type: wptest.TypeProduct, Title: "Two"},
	)

	response, payload := call(t, server, http.MethodGet, "/wp-json/wc/v3/products?page=9&per_page=1", nil, true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; WooCommerce answers an empty page", response.StatusCode)
	}
	if response.Header.Get("X-WP-TotalPages") != "2" {
		t.Errorf("X-WP-TotalPages = %q, want 2", response.Header.Get("X-WP-TotalPages"))
	}

	var items []map[string]any
	decode(t, payload, &items)
	if len(items) != 0 {
		t.Errorf("got %d items, want none", len(items))
	}
}

func TestAProductStatusFilterTakesOneValue(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.Seed(
		wptest.Item{Type: wptest.TypeProduct, Title: "Live", Status: "publish"},
		wptest.Item{Type: wptest.TypeProduct, Title: "Draft", Status: "draft"},
	)

	response, _ := call(t, server, http.MethodGet, "/wp-json/wc/v3/products?status=publish,draft", nil, true)
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for a comma list", response.StatusCode)
	}

	_, payload := call(t, server, http.MethodGet, "/wp-json/wc/v3/products?status=draft", nil, true)
	var drafts []map[string]any
	decode(t, payload, &drafts)
	if len(drafts) != 1 {
		t.Errorf("got %d drafts, want 1", len(drafts))
	}
}

func TestUpdatingAProductWritesTheWooCommerceFields(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	category := server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Koffein"})[0]
	product := server.Seed(wptest.Item{Type: wptest.TypeProduct, Title: "Powder"})[0]

	body := []byte(`{"description":"<p>new</p>","short_description":"brief","status":"draft","categories":[{"id":` + itoa(category.ID) + `}]}`)
	_, payload := call(t, server, http.MethodPost, "/wp-json/wc/v3/products/"+itoa(product.ID), body, true)

	var updated struct {
		Description      string `json:"description"`
		ShortDescription string `json:"short_description"`
		Status           string `json:"status"`
		Categories       []struct {
			ID   int64  `json:"id"`
			Slug string `json:"slug"`
		} `json:"categories"`
	}
	decode(t, payload, &updated)

	if updated.Description != "<p>new</p>" || updated.ShortDescription != "brief" || updated.Status != "draft" {
		t.Errorf("updated = %+v", updated)
	}
	if len(updated.Categories) != 1 || updated.Categories[0].ID != category.ID || updated.Categories[0].Slug != "koffein" {
		t.Errorf("categories = %+v", updated.Categories)
	}
}

func TestProductCategoriesCarryNoModificationDate(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Koffein", Content: "the hub"})

	_, payload := call(t, server, http.MethodGet, "/wp-json/wc/v3/products/categories", nil, true)

	var categories []map[string]any
	decode(t, payload, &categories)
	if len(categories) != 1 {
		t.Fatalf("got %d categories, want 1", len(categories))
	}
	if _, present := categories[0]["date_modified_gmt"]; present {
		t.Error("a WooCommerce product category is a term and has no modification date")
	}
	if categories[0]["name"] != "Koffein" || categories[0]["description"] != "the hub" {
		t.Errorf("category = %v", categories[0])
	}
}
```

Create `internal/adapters/wp/wptest/media_test.go`:

```go
package wptest_test

import (
	"net/http"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
)

func TestARawUploadStoresTheBytesAndIgnoresAlternativeText(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	request := newUpload(t, server, "koffein.png", "image/png", []byte{0x89, 0x50, 0x4e, 0x47})

	response, payload := send(t, request)
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", response.StatusCode)
	}

	var uploaded struct {
		ID        int64  `json:"id"`
		SourceURL string `json:"source_url"`
		AltText   string `json:"alt_text"`
		MimeType  string `json:"mime_type"`
	}
	decode(t, payload, &uploaded)

	if uploaded.ID == 0 || uploaded.SourceURL == "" || uploaded.MimeType != "image/png" {
		t.Errorf("uploaded = %+v", uploaded)
	}
	if uploaded.AltText != "" {
		t.Error("the raw upload cannot carry alternative text")
	}

	uploads := server.Uploads()
	if len(uploads) != 1 || uploads[0].Filename != "koffein.png" || len(uploads[0].Bytes) != 4 {
		t.Errorf("uploads = %+v", uploads)
	}

	_, payload = call(t, server, http.MethodPost, "/wp-json/wp/v2/media/"+itoa(uploaded.ID), []byte(`{"alt_text":"Koffein","title":"Koffein"}`), true)
	decode(t, payload, &uploaded)
	if uploaded.AltText != "Koffein" {
		t.Errorf("alt_text = %q, want Koffein after the second call", uploaded.AltText)
	}
}

func TestCategoriesListAndCreateWithUniqueSlugs(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.SeedCategory(wptest.Category{Name: "Koffein", Description: "the hub"})

	response, payload := call(t, server, http.MethodPost, "/wp-json/wp/v2/categories", []byte(`{"name":"Koffein"}`), true)
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", response.StatusCode)
	}

	var created struct {
		ID   int64  `json:"id"`
		Slug string `json:"slug"`
	}
	decode(t, payload, &created)
	if created.Slug != "koffein-2" {
		t.Errorf("slug = %q, want koffein-2", created.Slug)
	}

	response, payload = call(t, server, http.MethodGet, "/wp-json/wp/v2/categories", nil, true)
	if response.Header.Get("X-WP-Total") != "2" {
		t.Errorf("X-WP-Total = %q, want 2", response.Header.Get("X-WP-Total"))
	}

	var categories []map[string]any
	decode(t, payload, &categories)
	if len(categories) != 2 {
		t.Errorf("got %d categories, want 2", len(categories))
	}
	if len(server.Categories()) != 2 {
		t.Errorf("the store holds %d categories", len(server.Categories()))
	}
}
```

Add the two upload helpers to `server_test.go`:

```go
func newUpload(t *testing.T, server *wptest.Server, filename, contentType string, payload []byte) *http.Request {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, server.URL()+"/wp-json/wp/v2/media", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("build the upload: %v", err)
	}
	request.SetBasicAuth(wptest.DefaultUser, wptest.DefaultPassword)
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	return request
}

func send(t *testing.T, request *http.Request) (*http.Response, []byte) {
	t.Helper()

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("send the request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read the response: %v", err)
	}
	return response, payload
}
```

- [x] **Step 2: Run the tests and watch them fail**

Run: `go test -count=1 ./internal/adapters/wp/wptest/...`
Expected: FAIL — every WooCommerce, category and media route answers 404.

- [x] **Step 3: Write the WooCommerce handlers**

Create `internal/adapters/wp/wptest/woo.go`:

```go
package wptest

import (
	"net/http"
	"strconv"
)

const wooNamespace = "/wp-json/wc/v3"

func (s *Server) routeWoo(mux *http.ServeMux) {
	mux.HandleFunc("GET "+wooNamespace+"/products", s.handleProductList)
	mux.HandleFunc("GET "+wooNamespace+"/products/{id}", s.handleProductGet)
	mux.HandleFunc("POST "+wooNamespace+"/products/{id}", s.handleProductUpdate)
	mux.HandleFunc("GET "+wooNamespace+"/products/categories", s.handleProductCategoryList)
	mux.HandleFunc("GET "+wooNamespace+"/products/categories/{id}", s.handleProductCategoryGet)
}

func (s *Server) handleProductList(w http.ResponseWriter, r *http.Request) {
	s.handleWooList(w, r, TypeProduct, s.productPayload)
}

func (s *Server) handleProductCategoryList(w http.ResponseWriter, r *http.Request) {
	s.handleWooList(w, r, TypeProductCategory, s.productCategoryPayload)
}

func (s *Server) handleWooList(w http.ResponseWriter, r *http.Request, itemType string, render func(*Item) map[string]any) {
	query := r.URL.Query()
	page, perPage, bad := listWindow(query)
	if bad != "" {
		s.fail(w, http.StatusBadRequest, "woocommerce_rest_invalid_param", "Invalid parameter(s): "+bad)
		return
	}
	if len(splitList(query.Get("status"))) > 1 {
		s.fail(w, http.StatusBadRequest, "woocommerce_rest_invalid_param", "Invalid parameter(s): status")
		return
	}

	s.mu.Lock()
	matched := s.filter(itemType, query)
	total := len(matched)
	totalPages := (total + perPage - 1) / perPage

	start := min((page-1)*perPage, total)
	end := min(start+perPage, total)
	payload := make([]map[string]any, 0, end-start)
	for _, stored := range matched[start:end] {
		payload = append(payload, narrowFields(render(stored), query.Get("_fields")))
	}
	s.mu.Unlock()

	w.Header().Set("X-WP-Total", strconv.Itoa(total))
	w.Header().Set("X-WP-TotalPages", strconv.Itoa(totalPages))
	s.respond(w, http.StatusOK, payload)
}

func (s *Server) handleProductGet(w http.ResponseWriter, r *http.Request) {
	s.handleWooGet(w, r, TypeProduct, s.productPayload)
}

func (s *Server) handleProductCategoryGet(w http.ResponseWriter, r *http.Request) {
	s.handleWooGet(w, r, TypeProductCategory, s.productCategoryPayload)
}

func (s *Server) handleWooGet(w http.ResponseWriter, r *http.Request, itemType string, render func(*Item) map[string]any) {
	id, ok := pathID(r)
	if !ok {
		s.fail(w, http.StatusNotFound, "woocommerce_rest_invalid_id", "Invalid ID.")
		return
	}

	s.mu.Lock()
	stored, found := s.items[id]
	var payload map[string]any
	if found && stored.Type == itemType {
		payload = render(stored)
	}
	s.mu.Unlock()

	if payload == nil {
		s.fail(w, http.StatusNotFound, "woocommerce_rest_invalid_id", "Invalid ID.")
		return
	}
	s.respond(w, http.StatusOK, narrowFields(payload, r.URL.Query().Get("_fields")))
}

func (s *Server) handleProductUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		s.fail(w, http.StatusNotFound, "woocommerce_rest_invalid_id", "Invalid ID.")
		return
	}

	body, ok := s.decodeBody(w, r)
	if !ok {
		return
	}

	s.mu.Lock()
	stored, found := s.items[id]
	if !found || stored.Type != TypeProduct {
		s.mu.Unlock()
		s.fail(w, http.StatusNotFound, "woocommerce_rest_invalid_id", "Invalid ID.")
		return
	}

	if value, present := body["name"].(string); present {
		stored.Title = value
	}
	if value, present := body["description"].(string); present {
		stored.Content = value
	}
	if value, present := body["short_description"].(string); present {
		stored.Excerpt = value
	}
	if value, present := body["status"].(string); present {
		stored.Status = value
	}
	if slug := stringField(body, "slug"); slug != "" {
		stored.Slug = s.uniqueSlug(slug, stored.Type, stored.Parent, stored.ID)
	}
	if _, present := body["categories"]; present {
		stored.Categories = objectIDList(body, "categories")
	}
	stored.Modified = s.tick()
	payload := s.productPayload(stored)
	s.mu.Unlock()

	s.respond(w, http.StatusOK, payload)
}

func (s *Server) productPayload(stored *Item) map[string]any {
	return map[string]any{
		"id":                stored.ID,
		"name":              stored.Title,
		"slug":              stored.Slug,
		"permalink":         s.http.URL + s.itemPath(stored),
		"status":            stored.Status,
		"description":       stored.Content,
		"short_description": stored.Excerpt,
		"menu_order":        stored.MenuOrder,
		"date_modified_gmt": stored.Modified.UTC().Format(wpTimeLayout),
		"categories":        s.categoryRefs(stored.Categories),
	}
}

func (s *Server) productCategoryPayload(stored *Item) map[string]any {
	return map[string]any{
		"id":          stored.ID,
		"name":        stored.Title,
		"slug":        stored.Slug,
		"parent":      stored.Parent,
		"description": stored.Content,
		"count":       s.countProducts(stored.ID),
	}
}

func (s *Server) categoryRefs(ids []int64) []map[string]any {
	refs := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		stored, ok := s.items[id]
		if !ok {
			continue
		}
		refs = append(refs, map[string]any{"id": stored.ID, "name": stored.Title, "slug": stored.Slug})
	}
	return refs
}

func (s *Server) countProducts(categoryID int64) int {
	count := 0
	for _, id := range s.order {
		stored := s.items[id]
		if stored.Type != TypeProduct {
			continue
		}
		for _, assigned := range stored.Categories {
			if assigned == categoryID {
				count++
				break
			}
		}
	}
	return count
}

func objectIDList(body map[string]any, key string) []int64 {
	raw, ok := body[key].([]any)
	if !ok {
		return nil
	}

	ids := make([]int64, 0, len(raw))
	for _, entry := range raw {
		object, valid := entry.(map[string]any)
		if !valid {
			continue
		}
		number, present := object["id"].(float64)
		if !present {
			continue
		}
		ids = append(ids, int64(number))
	}
	return ids
}
```

- [x] **Step 4: Write the category handlers**

Create `internal/adapters/wp/wptest/category.go`:

```go
package wptest

import (
	"net/http"
	"slices"
	"strconv"
)

func (s *Server) routeCategories(mux *http.ServeMux) {
	mux.HandleFunc("GET "+coreNamespace+"/categories", s.handleCategoryList)
	mux.HandleFunc("POST "+coreNamespace+"/categories", s.handleCategoryCreate)
	mux.HandleFunc("GET "+coreNamespace+"/categories/{id}", s.handleCategoryGet)
}

func (s *Server) Categories() []Category {
	s.mu.Lock()
	defer s.mu.Unlock()

	categories := make([]Category, 0, len(s.categoryOrder))
	for _, id := range s.categoryOrder {
		categories = append(categories, *s.categories[id])
	}
	return categories
}

func (s *Server) handleCategoryList(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	page, perPage, bad := listWindow(query)
	if bad != "" {
		s.fail(w, http.StatusBadRequest, "rest_invalid_param", "Invalid parameter(s): "+bad)
		return
	}

	s.mu.Lock()
	total := len(s.categoryOrder)
	totalPages := (total + perPage - 1) / perPage
	if total > 0 && page > totalPages {
		s.mu.Unlock()
		s.fail(w, http.StatusBadRequest, "rest_post_invalid_page_number", "The page number requested is larger than the number of pages available.")
		return
	}

	start := min((page-1)*perPage, total)
	end := min(start+perPage, total)
	payload := make([]map[string]any, 0, end-start)
	for _, id := range s.categoryOrder[start:end] {
		payload = append(payload, narrowFields(categoryPayload(s.categories[id]), query.Get("_fields")))
	}
	s.mu.Unlock()

	w.Header().Set("X-WP-Total", strconv.Itoa(total))
	w.Header().Set("X-WP-TotalPages", strconv.Itoa(totalPages))
	s.respond(w, http.StatusOK, payload)
}

func (s *Server) handleCategoryGet(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		s.fail(w, http.StatusNotFound, "rest_term_invalid", "Term does not exist.")
		return
	}

	s.mu.Lock()
	stored, found := s.categories[id]
	var payload map[string]any
	if found {
		payload = categoryPayload(stored)
	}
	s.mu.Unlock()

	if payload == nil {
		s.fail(w, http.StatusNotFound, "rest_term_invalid", "Term does not exist.")
		return
	}
	s.respond(w, http.StatusOK, payload)
}

func (s *Server) handleCategoryCreate(w http.ResponseWriter, r *http.Request) {
	body, ok := s.decodeBody(w, r)
	if !ok {
		return
	}

	name := stringField(body, "name")
	if name == "" {
		s.fail(w, http.StatusBadRequest, "rest_missing_callback_param", "Missing parameter(s): name")
		return
	}

	s.mu.Lock()
	base := stringField(body, "slug")
	if base == "" {
		base = slugify(name)
	}

	s.nextID++
	stored := &Category{
		ID:          s.nextID,
		Name:        name,
		Slug:        s.uniqueCategorySlug(base),
		Description: stringField(body, "description"),
		Parent:      intField(body, "parent"),
	}
	s.categories[stored.ID] = stored
	s.categoryOrder = append(s.categoryOrder, stored.ID)
	payload := categoryPayload(stored)
	s.mu.Unlock()

	s.respond(w, http.StatusCreated, payload)
}

func (s *Server) uniqueCategorySlug(base string) string {
	taken := make([]string, 0, len(s.categoryOrder))
	for _, id := range s.categoryOrder {
		taken = append(taken, s.categories[id].Slug)
	}

	candidate := base
	for suffix := 2; slices.Contains(taken, candidate); suffix++ {
		candidate = base + "-" + strconv.Itoa(suffix)
	}
	return candidate
}

func categoryPayload(stored *Category) map[string]any {
	return map[string]any{
		"id":          stored.ID,
		"name":        stored.Name,
		"slug":        stored.Slug,
		"description": stored.Description,
		"parent":      stored.Parent,
		"count":       stored.Count,
	}
}
```

- [x] **Step 5: Write the media handlers**

Create `internal/adapters/wp/wptest/media.go`:

```go
package wptest

import (
	"io"
	"mime"
	"net/http"
	"path"
	"slices"
)

type Upload struct {
	Filename string
	MimeType string
	Alt      string
	Title    string
	Bytes    []byte
	ID       int64
}

func (s *Server) routeMedia(mux *http.ServeMux) {
	mux.HandleFunc("POST "+coreNamespace+"/media", s.handleMediaUpload)
	mux.HandleFunc("GET "+coreNamespace+"/media/{id}", s.handleMediaGet)
	mux.HandleFunc("POST "+coreNamespace+"/media/{id}", s.handleMediaUpdate)
}

func (s *Server) Uploads() []Upload {
	s.mu.Lock()
	defer s.mu.Unlock()

	uploads := make([]Upload, 0, len(s.uploadOrder))
	for _, id := range s.uploadOrder {
		stored := s.uploads[id]
		uploads = append(uploads, Upload{
			ID:       stored.ID,
			Filename: stored.Filename,
			MimeType: stored.MimeType,
			Alt:      stored.Alt,
			Title:    stored.Title,
			Bytes:    slices.Clone(stored.Bytes),
		})
	}
	return uploads
}

func (s *Server) handleMediaUpload(w http.ResponseWriter, r *http.Request) {
	filename := dispositionFilename(r.Header.Get("Content-Disposition"))
	if filename == "" {
		s.fail(w, http.StatusBadRequest, "rest_upload_no_content_disposition", "No Content-Disposition supplied.")
		return
	}

	payload, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBytes))
	if err != nil || len(payload) == 0 {
		s.fail(w, http.StatusBadRequest, "rest_upload_no_data", "No data supplied.")
		return
	}

	s.mu.Lock()
	s.nextID++
	stored := &upload{
		ID:       s.nextID,
		Filename: filename,
		MimeType: r.Header.Get("Content-Type"),
		Title:    filename,
		Bytes:    payload,
	}
	s.uploads[stored.ID] = stored
	s.uploadOrder = append(s.uploadOrder, stored.ID)
	body := s.uploadPayload(stored)
	s.mu.Unlock()

	s.respond(w, http.StatusCreated, body)
}

func (s *Server) handleMediaGet(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		s.fail(w, http.StatusNotFound, "rest_post_invalid_id", "Invalid post ID.")
		return
	}

	s.mu.Lock()
	stored, found := s.uploads[id]
	var body map[string]any
	if found {
		body = s.uploadPayload(stored)
	}
	s.mu.Unlock()

	if body == nil {
		s.fail(w, http.StatusNotFound, "rest_post_invalid_id", "Invalid post ID.")
		return
	}
	s.respond(w, http.StatusOK, body)
}

func (s *Server) handleMediaUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		s.fail(w, http.StatusNotFound, "rest_post_invalid_id", "Invalid post ID.")
		return
	}

	attributes, ok := s.decodeBody(w, r)
	if !ok {
		return
	}

	s.mu.Lock()
	stored, found := s.uploads[id]
	if !found {
		s.mu.Unlock()
		s.fail(w, http.StatusNotFound, "rest_post_invalid_id", "Invalid post ID.")
		return
	}

	if value, present := attributes["alt_text"].(string); present {
		stored.Alt = value
	}
	if value, present := attributes["title"].(string); present {
		stored.Title = value
	}
	body := s.uploadPayload(stored)
	s.mu.Unlock()

	s.respond(w, http.StatusOK, body)
}

func (s *Server) uploadPayload(stored *upload) map[string]any {
	return map[string]any{
		"id":         stored.ID,
		"source_url": s.http.URL + "/wp-content/uploads/" + stored.Filename,
		"alt_text":   stored.Alt,
		"mime_type":  stored.MimeType,
		"title":      renderedField(stored.Title),
	}
}

func dispositionFilename(header string) string {
	if header == "" {
		return ""
	}

	_, params, err := mime.ParseMediaType(header)
	if err != nil {
		return ""
	}

	name := params["filename"]
	if name == "" {
		return ""
	}
	return path.Base(name)
}
```

- [x] **Step 6: Register the three route sets**

Extend `handler()` in `server.go`:

```go
func (s *Server) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+rootPath, s.handleRoot)
	s.routeCore(mux)
	s.routeCategories(mux)
	s.routeMedia(mux)
	s.routeWoo(mux)

	return s.record(s.redirectRoot(s.injectFaults(s.authenticate(mux))))
}
```

- [x] **Step 7: Run the tests and watch them pass**

Run: `go test -count=1 -race ./internal/adapters/wp/...`
Expected: PASS.

- [x] **Step 8: Run the gate and commit**

```bash
go build ./... && go vet ./... && "$(go env GOPATH)/bin/golangci-lint.exe" run && go test -count=1 -race ./...
git add internal/adapters/wp/wptest
git commit -m "test(wp): woocommerce, categories and media in the fake server

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---
### Task 7: Item types, wire decoding and the two read methods

**Files:**
- Create: `internal/adapters/wp/item.go`, `internal/adapters/wp/items.go`
- Test: `internal/adapters/wp/items_test.go` (package `wp_test`)

**Interfaces:**
- Consumes: task 4's `do`, `decodeJSON`, `detailString`; task 5 and 6's fake routes.
- Produces:
  - `type ItemType string` with `TypePage`, `TypePost`, `TypeProduct`, `TypeProductCategory`; `func (t ItemType) route() (string, string, error)`; `func (t ItemType) core() bool`
  - `type Item struct { Modified time.Time; Meta map[string]any; Type ItemType; Title, Content, Excerpt, Slug, Status, Link, Template string; Categories, Tags []int64; ID, Parent, FeaturedMedia int64; MenuOrder int }`
  - `type Page[T any] struct { Items []T; Total, TotalPages, Page int; HasMore bool }` and `type ItemPage = Page[Item]`
  - `type ListQuery struct { ModifiedAfter *time.Time; Status, Fields []string; Page, PerPage int }`
  - `func (c *Client) ListItems(ctx context.Context, itemType ItemType, query ListQuery) (ItemPage, error)`
  - `func (c *Client) GetItem(ctx context.Context, itemType ItemType, id int64) (Item, error)`
  - the wire payloads `itemPayload`, `productPayload`, `productCategoryRef`, `productCategoryPayload`, `renderedText`, `metaBag`, and `parseWPTime`, `newPage`, `headerInt`, `endOfList`, `resourcePath`

- [x] **Step 1: Write the failing test**

Create `internal/adapters/wp/items_test.go`:

```go
package wp_test

import (
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestListItemsWalksEveryPage(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	for index := range 5 {
		server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Page " + string(rune('A'+index))})
	}

	client := newClient(t, server)
	seen := 0
	for page := 1; ; page++ {
		result, err := client.ListItems(t.Context(), wp.TypePage, wp.ListQuery{Page: page, PerPage: 2})
		if err != nil {
			t.Fatalf("ListItems page %d: %v", page, err)
		}
		if result.Total != 5 || result.TotalPages != 3 {
			t.Fatalf("page %d reports total %d over %d pages", page, result.Total, result.TotalPages)
		}
		seen += len(result.Items)
		if !result.HasMore {
			break
		}
	}
	if seen != 5 {
		t.Errorf("walked %d items, want 5", seen)
	}
}

func TestListItemsTreatsThePageNumberErrorAsTheEndOfTheList(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Only"})

	result, err := newClient(t, server).ListItems(t.Context(), wp.TypePage, wp.ListQuery{Page: 9, PerPage: 2})
	if err != nil {
		t.Fatalf("a page past the end is the end of the list, not a failure: %v", err)
	}
	if len(result.Items) != 0 || result.HasMore {
		t.Errorf("result = %+v, want an empty final page", result)
	}
}

func TestListItemsStopsAtTheEndOfAWooCommerceCollection(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.Seed(
		wptest.Item{Type: wptest.TypeProduct, Title: "One"},
		wptest.Item{Type: wptest.TypeProduct, Title: "Two"},
	)

	result, err := newClient(t, server).ListItems(t.Context(), wp.TypeProduct, wp.ListQuery{Page: 9, PerPage: 1})
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if len(result.Items) != 0 || result.HasMore {
		t.Errorf("result = %+v, want an empty final page", result)
	}
}

func TestTheQueryMatchesTheNamespace(t *testing.T) {
	t.Parallel()

	cut := time.Date(2026, time.September, 18, 9, 0, 0, 0, time.UTC)
	cases := []struct {
		name          string
		itemType      wp.ItemType
		query         wp.ListQuery
		wantContext   string
		wantStatus    string
		wantModified  string
		wantFieldList string
	}{
		{
			name:          "core takes a status list and an edit context",
			itemType:      wp.TypePage,
			query:         wp.ListQuery{Status: []string{"publish", "draft"}, ModifiedAfter: &cut, Fields: []string{"slug"}},
			wantContext:   "edit",
			wantStatus:    "publish,draft",
			wantModified:  "2026-09-18T09:00:00",
			wantFieldList: "id,slug",
		},
		{
			name:         "woocommerce takes one status and no context",
			itemType:     wp.TypeProduct,
			query:        wp.ListQuery{Status: []string{"publish", "draft"}, ModifiedAfter: &cut},
			wantStatus:   "publish",
			wantModified: "2026-09-18T09:00:00",
		},
		{
			name:     "a product category has no modification date to filter on",
			itemType: wp.TypeProductCategory,
			query:    wp.ListQuery{ModifiedAfter: &cut},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := wptest.New(t)
			if _, err := newClient(t, server).ListItems(t.Context(), tc.itemType, tc.query); err != nil {
				t.Fatalf("ListItems: %v", err)
			}

			recorded, ok := server.LastRequest()
			if !ok {
				t.Fatal("no request reached the site")
			}
			if got := recorded.Query.Get("context"); got != tc.wantContext {
				t.Errorf("context = %q, want %q", got, tc.wantContext)
			}
			if got := recorded.Query.Get("status"); got != tc.wantStatus {
				t.Errorf("status = %q, want %q", got, tc.wantStatus)
			}
			if got := recorded.Query.Get("modified_after"); got != tc.wantModified {
				t.Errorf("modified_after = %q, want %q", got, tc.wantModified)
			}
			if got := recorded.Query.Get("_fields"); got != tc.wantFieldList {
				t.Errorf("_fields = %q, want %q", got, tc.wantFieldList)
			}
		})
	}
}

func TestTheQueryClampsTheWindow(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	if _, err := newClient(t, server).ListItems(t.Context(), wp.TypePage, wp.ListQuery{Page: 0, PerPage: 9999}); err != nil {
		t.Fatalf("ListItems: %v", err)
	}

	recorded, _ := server.LastRequest()
	if got := recorded.Query.Get("page"); got != "1" {
		t.Errorf("page = %q, want 1", got)
	}
	if got := recorded.Query.Get("per_page"); got != "100" {
		t.Errorf("per_page = %q, want the WordPress maximum of 100", got)
	}
}

func TestGetItemReadsTheRawPostRatherThanTheRendering(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := server.Seed(wptest.Item{
		Type:    wptest.TypePage,
		Title:   "Koffein",
		Content: "<p>Koffein ist ein Alkaloid.</p>",
		Excerpt: "short",
		Status:  "publish",
	})

	item, err := newClient(t, server).GetItem(t.Context(), wp.TypePage, seeded[0].ID)
	if err != nil {
		t.Fatalf("GetItem: %v", err)
	}

	if item.Title != "Koffein" || item.Content != "<p>Koffein ist ein Alkaloid.</p>" || item.Excerpt != "short" {
		t.Errorf("item = %+v", item)
	}
	if item.Type != wp.TypePage || item.Slug != "koffein" || item.Link == "" {
		t.Errorf("item = %+v", item)
	}
	if item.Modified.IsZero() || item.Modified.Location() != time.UTC {
		t.Errorf("modified = %s, want a UTC instant parsed from a suffix-less WordPress stamp", item.Modified)
	}
	if item.Meta != nil {
		t.Errorf("meta = %v, want nil when the site sends the empty array", item.Meta)
	}
}

func TestGetItemMapsAWooCommerceProduct(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	category := server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Koffein"})[0]
	seeded := server.Seed(wptest.Item{
		Type:       wptest.TypeProduct,
		Title:      "Powder",
		Content:    "<p>long</p>",
		Excerpt:    "short",
		Categories: []int64{category.ID},
	})

	item, err := newClient(t, server).GetItem(t.Context(), wp.TypeProduct, seeded[0].ID)
	if err != nil {
		t.Fatalf("GetItem: %v", err)
	}

	if item.Title != "Powder" || item.Content != "<p>long</p>" || item.Excerpt != "short" {
		t.Errorf("item = %+v", item)
	}
	if item.Type != wp.TypeProduct || item.Link == "" {
		t.Errorf("item = %+v", item)
	}
	if len(item.Categories) != 1 || item.Categories[0] != category.ID {
		t.Errorf("categories = %v", item.Categories)
	}
}

func TestGetItemMapsAProductCategory(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Koffein", Content: "the hub"})

	item, err := newClient(t, server).GetItem(t.Context(), wp.TypeProductCategory, seeded[0].ID)
	if err != nil {
		t.Fatalf("GetItem: %v", err)
	}
	if item.Title != "Koffein" || item.Content != "the hub" || item.Slug != "koffein" {
		t.Errorf("item = %+v", item)
	}
	if !item.Modified.IsZero() {
		t.Error("a product category is a term and carries no modification date")
	}
}

func TestTheReadMethodsRejectTheUnknownAndTheMissing(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	client := newClient(t, server)

	if _, err := client.ListItems(t.Context(), wp.ItemType("attachment"), wp.ListQuery{}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
	}
	if _, err := client.GetItem(t.Context(), wp.TypePage, 404); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.NotFound)
	}
}
```

- [x] **Step 2: Run the test and watch it fail**

Run: `go test -count=1 ./internal/adapters/wp/...`
Expected: FAIL — `undefined: wp.TypePage`, `undefined: (*wp.Client).ListItems`.

- [x] **Step 3: Write the types and the wire decoding**

Create `internal/adapters/wp/item.go`:

```go
package wp

import (
	"bytes"
	"encoding/json"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	wpTimeLayout   = "2006-01-02T15:04:05"
	defaultPerPage = 50
	maxPerPage     = 100
)

type ItemType string

const (
	TypePage            ItemType = "page"
	TypePost            ItemType = "post"
	TypeProduct         ItemType = "product"
	TypeProductCategory ItemType = "product_cat"
)

func (t ItemType) route() (string, string, error) {
	switch t {
	case TypePage:
		return coreNamespace, "/pages", nil
	case TypePost:
		return coreNamespace, "/posts", nil
	case TypeProduct:
		return wooNamespace, "/products", nil
	case TypeProductCategory:
		return wooNamespace, "/products/categories", nil
	default:
		return "", "", errors.New(errors.Invalid, "unknown WordPress content type").WithDetail("type", string(t))
	}
}

func (t ItemType) core() bool {
	return t == TypePage || t == TypePost
}

type Item struct {
	Modified      time.Time
	Meta          map[string]any
	Type          ItemType
	Title         string
	Content       string
	Excerpt       string
	Slug          string
	Status        string
	Link          string
	Template      string
	Categories    []int64
	Tags          []int64
	ID            int64
	Parent        int64
	FeaturedMedia int64
	MenuOrder     int
}

type Page[T any] struct {
	Items      []T
	Total      int
	TotalPages int
	Page       int
	HasMore    bool
}

type ItemPage = Page[Item]

type ListQuery struct {
	ModifiedAfter *time.Time
	Status        []string
	Fields        []string
	Page          int
	PerPage       int
}

func (q ListQuery) pageNumber() int {
	if q.Page < 1 {
		return 1
	}
	return q.Page
}

func (q ListQuery) perPageSize() int {
	switch {
	case q.PerPage < 1:
		return defaultPerPage
	case q.PerPage > maxPerPage:
		return maxPerPage
	default:
		return q.PerPage
	}
}

func (q ListQuery) values(itemType ItemType) url.Values {
	query := url.Values{}
	if itemType.core() {
		query.Set("context", "edit")
	}
	query.Set("page", strconv.Itoa(q.pageNumber()))
	query.Set("per_page", strconv.Itoa(q.perPageSize()))
	query.Set("orderby", "id")
	query.Set("order", "asc")

	if q.ModifiedAfter != nil && itemType != TypeProductCategory {
		query.Set("modified_after", q.ModifiedAfter.UTC().Format(wpTimeLayout))
	}
	if len(q.Status) > 0 {
		if itemType.core() {
			query.Set("status", strings.Join(q.Status, ","))
		} else {
			query.Set("status", q.Status[0])
		}
	}
	if len(q.Fields) > 0 {
		query.Set("_fields", strings.Join(withID(q.Fields), ","))
	}
	return query
}

func withID(fields []string) []string {
	if slices.Contains(fields, "id") {
		return fields
	}
	return append([]string{"id"}, fields...)
}

func resourcePath(path string, id int64) string {
	return path + "/" + strconv.FormatInt(id, 10)
}

type renderedText struct {
	Raw      string `json:"raw"`
	Rendered string `json:"rendered"`
}

func (r renderedText) value() string {
	if r.Raw != "" {
		return r.Raw
	}
	return r.Rendered
}

type metaBag map[string]any

func (m *metaBag) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] == '[' || bytes.Equal(trimmed, []byte("null")) {
		*m = nil
		return nil
	}

	var values map[string]any
	if err := json.Unmarshal(trimmed, &values); err != nil {
		return err
	}
	*m = values
	return nil
}

type itemPayload struct {
	Title         renderedText `json:"title"`
	Content       renderedText `json:"content"`
	Excerpt       renderedText `json:"excerpt"`
	Meta          metaBag      `json:"meta"`
	Type          string       `json:"type"`
	Slug          string       `json:"slug"`
	Status        string       `json:"status"`
	Link          string       `json:"link"`
	Template      string       `json:"template"`
	ModifiedGMT   string       `json:"modified_gmt"`
	Categories    []int64      `json:"categories"`
	Tags          []int64      `json:"tags"`
	ID            int64        `json:"id"`
	Parent        int64        `json:"parent"`
	FeaturedMedia int64        `json:"featured_media"`
	MenuOrder     int          `json:"menu_order"`
}

func (p itemPayload) item(fallback ItemType) Item {
	itemType := ItemType(p.Type)
	if itemType == "" {
		itemType = fallback
	}

	return Item{
		ID:            p.ID,
		Type:          itemType,
		Title:         p.Title.value(),
		Content:       p.Content.value(),
		Excerpt:       p.Excerpt.value(),
		Slug:          p.Slug,
		Status:        p.Status,
		Link:          p.Link,
		Template:      p.Template,
		Categories:    p.Categories,
		Tags:          p.Tags,
		Parent:        p.Parent,
		FeaturedMedia: p.FeaturedMedia,
		MenuOrder:     p.MenuOrder,
		Meta:          p.Meta,
		Modified:      parseWPTime(p.ModifiedGMT),
	}
}

type productCategoryRef struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	ID   int64  `json:"id"`
}

type productPayload struct {
	Name             string               `json:"name"`
	Slug             string               `json:"slug"`
	Permalink        string               `json:"permalink"`
	Status           string               `json:"status"`
	Description      string               `json:"description"`
	ShortDescription string               `json:"short_description"`
	DateModifiedGMT  string               `json:"date_modified_gmt"`
	Categories       []productCategoryRef `json:"categories"`
	ID               int64                `json:"id"`
	MenuOrder        int                  `json:"menu_order"`
}

func (p productPayload) item() Item {
	categories := make([]int64, 0, len(p.Categories))
	for _, ref := range p.Categories {
		categories = append(categories, ref.ID)
	}

	return Item{
		ID:         p.ID,
		Type:       TypeProduct,
		Title:      p.Name,
		Content:    p.Description,
		Excerpt:    p.ShortDescription,
		Slug:       p.Slug,
		Status:     p.Status,
		Link:       p.Permalink,
		Categories: categories,
		MenuOrder:  p.MenuOrder,
		Modified:   parseWPTime(p.DateModifiedGMT),
	}
}

type productCategoryPayload struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	ID          int64  `json:"id"`
	Parent      int64  `json:"parent"`
	Count       int    `json:"count"`
}

func (p productCategoryPayload) item() Item {
	return Item{
		ID:      p.ID,
		Type:    TypeProductCategory,
		Title:   p.Name,
		Content: p.Description,
		Slug:    p.Slug,
		Parent:  p.Parent,
	}
}

func parseWPTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC()
	}
	if parsed, err := time.Parse(wpTimeLayout, value); err == nil {
		return parsed.UTC()
	}
	return time.Time{}
}

func decodeItems(itemType ItemType, body []byte) ([]Item, error) {
	switch itemType {
	case TypePage, TypePost:
		var payload []itemPayload
		if err := decodeJSON(body, &payload); err != nil {
			return nil, err
		}
		items := make([]Item, 0, len(payload))
		for _, entry := range payload {
			items = append(items, entry.item(itemType))
		}
		return items, nil
	case TypeProduct:
		var payload []productPayload
		if err := decodeJSON(body, &payload); err != nil {
			return nil, err
		}
		items := make([]Item, 0, len(payload))
		for _, entry := range payload {
			items = append(items, entry.item())
		}
		return items, nil
	case TypeProductCategory:
		var payload []productCategoryPayload
		if err := decodeJSON(body, &payload); err != nil {
			return nil, err
		}
		items := make([]Item, 0, len(payload))
		for _, entry := range payload {
			items = append(items, entry.item())
		}
		return items, nil
	default:
		return nil, errors.New(errors.Invalid, "unknown WordPress content type").WithDetail("type", string(itemType))
	}
}

func decodeItem(itemType ItemType, body []byte) (Item, error) {
	switch itemType {
	case TypePage, TypePost:
		var payload itemPayload
		if err := decodeJSON(body, &payload); err != nil {
			return Item{}, err
		}
		return payload.item(itemType), nil
	case TypeProduct:
		var payload productPayload
		if err := decodeJSON(body, &payload); err != nil {
			return Item{}, err
		}
		return payload.item(), nil
	case TypeProductCategory:
		var payload productCategoryPayload
		if err := decodeJSON(body, &payload); err != nil {
			return Item{}, err
		}
		return payload.item(), nil
	default:
		return Item{}, errors.New(errors.Invalid, "unknown WordPress content type").WithDetail("type", string(itemType))
	}
}
```

- [x] **Step 4: Write the read methods**

Create `internal/adapters/wp/items.go`:

```go
package wp

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func (c *Client) ListItems(ctx context.Context, itemType ItemType, query ListQuery) (ItemPage, error) {
	namespace, path, err := itemType.route()
	if err != nil {
		return ItemPage{}, err
	}

	resp, body, err := c.do(ctx, request{
		method:    http.MethodGet,
		namespace: namespace,
		path:      path,
		query:     query.values(itemType),
	})
	if err != nil {
		if endOfList(err) {
			return ItemPage{Page: query.pageNumber()}, nil
		}
		return ItemPage{}, err
	}

	items, err := decodeItems(itemType, body)
	if err != nil {
		return ItemPage{}, err
	}
	return newPage(resp, query.pageNumber(), items), nil
}

func (c *Client) GetItem(ctx context.Context, itemType ItemType, id int64) (Item, error) {
	namespace, path, err := itemType.route()
	if err != nil {
		return Item{}, err
	}

	query := url.Values{}
	if itemType.core() {
		query.Set("context", "edit")
	}

	_, body, err := c.do(ctx, request{
		method:    http.MethodGet,
		namespace: namespace,
		path:      resourcePath(path, id),
		query:     query,
	})
	if err != nil {
		return Item{}, err
	}
	return decodeItem(itemType, body)
}

func newPage[T any](resp *http.Response, page int, items []T) Page[T] {
	totalPages := headerInt(resp, "X-WP-TotalPages")
	return Page[T]{
		Items:      items,
		Total:      headerInt(resp, "X-WP-Total"),
		TotalPages: totalPages,
		Page:       page,
		HasMore:    page < totalPages,
	}
}

func headerInt(resp *http.Response, name string) int {
	value, err := strconv.Atoi(resp.Header.Get(name))
	if err != nil {
		return 0
	}
	return value
}

func endOfList(err error) bool {
	if !errors.IsCode(err, errors.Invalid) {
		return false
	}
	if detailString(err, "code") == "rest_post_invalid_page_number" {
		return true
	}
	return strings.Contains(strings.ToLower(detailString(err, "wpMessage")), "larger than the number of pages")
}
```

- [x] **Step 5: Run the test and watch it pass**

Run: `go test -count=1 -race ./internal/adapters/wp/...`
Expected: PASS.

- [x] **Step 6: Run the gate and commit**

```bash
go build ./... && go vet ./... && "$(go env GOPATH)/bin/golangci-lint.exe" run && go test -count=1 -race ./...
git add internal/adapters/wp
git commit -m "feat(wp): item types, wire decoding, list and get

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---
### Task 8: Creating, updating and deleting items

**Files:**
- Modify: `internal/adapters/wp/items.go` (three methods), `internal/adapters/wp/item.go` (two payload types), `internal/adapters/wp/do.go` (`encodeJSON`, `contentTypeJSON`)
- Test: `internal/adapters/wp/write_test.go` (package `wp_test`)

**Interfaces:**
- Consumes: task 7's `ItemType`, `Item`, `decodeItem`, `resourcePath`, `GetItem`.
- Produces:
  - `type CreateItem struct { Title, Content, Slug, Status string; Parent *int64; Excerpt string; MenuOrder int; Template string; Categories, Tags []int64; FeaturedMedia *int64; Meta map[string]any }`
  - `type UpdateItem struct { Title, Content, Slug, Status *string; Parent *int64; Excerpt *string; MenuOrder *int; Template *string; Categories, Tags []int64; FeaturedMedia *int64; Meta map[string]any }`
  - `func (c *Client) CreateItem(ctx context.Context, itemType ItemType, in CreateItem) (Item, error)`
  - `func (c *Client) UpdateItem(ctx context.Context, itemType ItemType, id int64, in UpdateItem) (Item, error)`
  - `func (c *Client) DeleteItem(ctx context.Context, itemType ItemType, id int64, force bool) error`
  - `func encodeJSON(payload any) ([]byte, error)`, `const contentTypeJSON = "application/json"`

- [x] **Step 1: Write the failing test**

Create `internal/adapters/wp/write_test.go`:

```go
package wp_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func pointerTo[T any](value T) *T {
	return &value
}

func TestCreateItemReturnsTheSlugWordPressActuallyUsed(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Powder"})
	server.ResetRequests()

	created, err := newClient(t, server).CreateItem(t.Context(), wp.TypePage, wp.CreateItem{
		Title:   "Powder",
		Content: "<p>x</p>",
		Slug:    "powder",
	})
	if err != nil {
		t.Fatalf("CreateItem: %v", err)
	}

	if created.Slug != "powder-2" {
		t.Errorf("slug = %q, want powder-2", created.Slug)
	}
	if created.Link == "" {
		t.Error("the created item must carry the permalink WordPress computed")
	}
	if created.Status != "draft" {
		t.Errorf("status = %q, want the draft default", created.Status)
	}

	recorded := server.Requests()
	if len(recorded) != 2 {
		t.Fatalf("the client made %d requests, want a create and a re-read", len(recorded))
	}
	if recorded[0].Method != "POST" || recorded[1].Method != "GET" {
		t.Errorf("requests = %s %s", recorded[0].Method, recorded[1].Method)
	}
}

func TestCreateItemPlacesAChildUnderItsParent(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	parent := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein"})[0]

	created, err := newClient(t, server).CreateItem(t.Context(), wp.TypePage, wp.CreateItem{
		Title:  "Powder",
		Status: "publish",
		Parent: pointerTo(parent.ID),
	})
	if err != nil {
		t.Fatalf("CreateItem: %v", err)
	}
	if created.Parent != parent.ID {
		t.Errorf("parent = %d, want %d", created.Parent, parent.ID)
	}
	if created.Status != "publish" {
		t.Errorf("status = %q, want publish", created.Status)
	}
}

func TestUpdateItemTreatsParentAsThreeStates(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		update func(parent int64) wp.UpdateItem
		want   func(parent int64) int64
	}{
		{
			name:   "absent keeps the parent",
			update: func(int64) wp.UpdateItem { return wp.UpdateItem{Status: pointerTo("publish")} },
			want:   func(parent int64) int64 { return parent },
		},
		{
			name:   "zero moves it to the top level",
			update: func(int64) wp.UpdateItem { return wp.UpdateItem{Parent: pointerTo(int64(0))} },
			want:   func(int64) int64 { return 0 },
		},
		{
			name:   "an id reparents it",
			update: func(parent int64) wp.UpdateItem { return wp.UpdateItem{Parent: pointerTo(parent)} },
			want:   func(parent int64) int64 { return parent },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := wptest.New(t)
			parent := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein"})[0]
			child := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Powder", Parent: parent.ID})[0]

			updated, err := newClient(t, server).UpdateItem(t.Context(), wp.TypePage, child.ID, tc.update(parent.ID))
			if err != nil {
				t.Fatalf("UpdateItem: %v", err)
			}
			if got := tc.want(parent.ID); updated.Parent != got {
				t.Errorf("parent = %d, want %d", updated.Parent, got)
			}
			if updated.Title != "Powder" {
				t.Errorf("title = %q; an absent field must not be overwritten", updated.Title)
			}
		})
	}
}

func TestUpdateItemDistinguishesKeepingFromClearing(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := server.Seed(wptest.Item{Type: wptest.TypePost, Title: "Powder", Categories: []int64{7, 9}})
	client := newClient(t, server)

	kept, err := client.UpdateItem(t.Context(), wp.TypePost, seeded[0].ID, wp.UpdateItem{Status: pointerTo("draft")})
	if err != nil {
		t.Fatalf("UpdateItem: %v", err)
	}
	if len(kept.Categories) != 2 {
		t.Errorf("categories = %v, want them kept when the field is nil", kept.Categories)
	}

	cleared, err := client.UpdateItem(t.Context(), wp.TypePost, seeded[0].ID, wp.UpdateItem{Categories: []int64{}})
	if err != nil {
		t.Fatalf("UpdateItem: %v", err)
	}
	if len(cleared.Categories) != 0 {
		t.Errorf("categories = %v, want them cleared by an empty slice", cleared.Categories)
	}
}

func TestUpdateItemRefusesAnEmptyChange(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Powder"})
	server.ResetRequests()

	_, err := newClient(t, server).UpdateItem(t.Context(), wp.TypePage, seeded[0].ID, wp.UpdateItem{})
	if !errors.IsCode(err, errors.Invalid) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
	}
	if len(server.Requests()) != 0 {
		t.Error("an empty update must not reach the site")
	}
}

func TestDeleteItemTrashesUnlessItIsForced(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := server.Seed(
		wptest.Item{Type: wptest.TypePage, Title: "One"},
		wptest.Item{Type: wptest.TypePage, Title: "Two"},
	)
	client := newClient(t, server)

	if err := client.DeleteItem(t.Context(), wp.TypePage, seeded[0].ID, false); err != nil {
		t.Fatalf("DeleteItem: %v", err)
	}
	trashed, ok := server.Lookup(seeded[0].ID)
	if !ok || trashed.Status != "trash" {
		t.Errorf("item = %+v, %t", trashed, ok)
	}

	if err := client.DeleteItem(t.Context(), wp.TypePage, seeded[1].ID, true); err != nil {
		t.Fatalf("DeleteItem forced: %v", err)
	}
	if _, ok = server.Lookup(seeded[1].ID); ok {
		t.Error("a forced delete removes the item")
	}

	if err := client.DeleteItem(t.Context(), wp.TypePage, 404, true); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.NotFound)
	}
}

func TestTheGenericWriteMethodsCoverPagesAndPostsOnly(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	client := newClient(t, server)
	server.ResetRequests()

	if _, err := client.CreateItem(t.Context(), wp.TypeProduct, wp.CreateItem{Title: "x"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("create code = %q, want %q", errors.CodeOf(err), errors.Invalid)
	}
	if _, err := client.UpdateItem(t.Context(), wp.TypeProduct, 1, wp.UpdateItem{Status: pointerTo("draft")}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("update code = %q, want %q", errors.CodeOf(err), errors.Invalid)
	}
	if err := client.DeleteItem(t.Context(), wp.TypeProductCategory, 1, true); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("delete code = %q, want %q", errors.CodeOf(err), errors.Invalid)
	}
	if len(server.Requests()) != 0 {
		t.Error("a rejected write must not reach the site")
	}
}
```

- [x] **Step 2: Run the test and watch it fail**

Run: `go test -count=1 ./internal/adapters/wp/...`
Expected: FAIL — `undefined: wp.CreateItem`.

- [x] **Step 3: Write the payload types**

Append to `internal/adapters/wp/item.go`:

```go
type CreateItem struct {
	Parent        *int64
	FeaturedMedia *int64
	Meta          map[string]any
	Title         string
	Content       string
	Slug          string
	Status        string
	Excerpt       string
	Template      string
	Categories    []int64
	Tags          []int64
	MenuOrder     int
}

func (in CreateItem) payload() map[string]any {
	status := in.Status
	if status == "" {
		status = "draft"
	}

	payload := map[string]any{
		"title":   in.Title,
		"content": in.Content,
		"status":  status,
	}
	if in.Slug != "" {
		payload["slug"] = in.Slug
	}
	if in.Parent != nil {
		payload["parent"] = *in.Parent
	}
	if in.Excerpt != "" {
		payload["excerpt"] = in.Excerpt
	}
	if in.MenuOrder != 0 {
		payload["menu_order"] = in.MenuOrder
	}
	if in.Template != "" {
		payload["template"] = in.Template
	}
	if len(in.Categories) > 0 {
		payload["categories"] = in.Categories
	}
	if len(in.Tags) > 0 {
		payload["tags"] = in.Tags
	}
	if in.FeaturedMedia != nil {
		payload["featured_media"] = *in.FeaturedMedia
	}
	if len(in.Meta) > 0 {
		payload["meta"] = in.Meta
	}
	return payload
}

type UpdateItem struct {
	Title         *string
	Content       *string
	Slug          *string
	Status        *string
	Parent        *int64
	Excerpt       *string
	MenuOrder     *int
	Template      *string
	FeaturedMedia *int64
	Meta          map[string]any
	Categories    []int64
	Tags          []int64
}

func (in UpdateItem) payload() map[string]any {
	payload := make(map[string]any)
	if in.Title != nil {
		payload["title"] = *in.Title
	}
	if in.Content != nil {
		payload["content"] = *in.Content
	}
	if in.Slug != nil {
		payload["slug"] = *in.Slug
	}
	if in.Status != nil {
		payload["status"] = *in.Status
	}
	if in.Parent != nil {
		payload["parent"] = *in.Parent
	}
	if in.Excerpt != nil {
		payload["excerpt"] = *in.Excerpt
	}
	if in.MenuOrder != nil {
		payload["menu_order"] = *in.MenuOrder
	}
	if in.Template != nil {
		payload["template"] = *in.Template
	}
	if in.FeaturedMedia != nil {
		payload["featured_media"] = *in.FeaturedMedia
	}
	if in.Categories != nil {
		payload["categories"] = in.Categories
	}
	if in.Tags != nil {
		payload["tags"] = in.Tags
	}
	if len(in.Meta) > 0 {
		payload["meta"] = in.Meta
	}
	return payload
}
```

- [x] **Step 4: Write the encoder and the three methods**

Append to `internal/adapters/wp/do.go`:

```go
const contentTypeJSON = "application/json"

func encodeJSON(payload any) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.New(errors.Invalid, "the WordPress request body cannot be encoded").WithInternal(err)
	}
	return body, nil
}
```

Append to `internal/adapters/wp/items.go`:

```go
func (c *Client) CreateItem(ctx context.Context, itemType ItemType, in CreateItem) (Item, error) {
	if !itemType.core() {
		return Item{}, coreOnly(itemType)
	}

	namespace, path, err := itemType.route()
	if err != nil {
		return Item{}, err
	}

	body, err := encodeJSON(in.payload())
	if err != nil {
		return Item{}, err
	}

	_, raw, err := c.do(ctx, request{
		method:      http.MethodPost,
		namespace:   namespace,
		path:        path,
		body:        body,
		contentType: contentTypeJSON,
	})
	if err != nil {
		return Item{}, err
	}

	created, err := decodeItem(itemType, raw)
	if err != nil {
		return Item{}, err
	}
	return c.GetItem(ctx, itemType, created.ID)
}

func (c *Client) UpdateItem(ctx context.Context, itemType ItemType, id int64, in UpdateItem) (Item, error) {
	if !itemType.core() {
		return Item{}, coreOnly(itemType)
	}

	namespace, path, err := itemType.route()
	if err != nil {
		return Item{}, err
	}

	payload := in.payload()
	if len(payload) == 0 {
		return Item{}, errors.New(errors.Invalid, "the update carries no fields")
	}

	body, err := encodeJSON(payload)
	if err != nil {
		return Item{}, err
	}

	_, raw, err := c.do(ctx, request{
		method:      http.MethodPost,
		namespace:   namespace,
		path:        resourcePath(path, id),
		body:        body,
		contentType: contentTypeJSON,
	})
	if err != nil {
		return Item{}, err
	}
	return decodeItem(itemType, raw)
}

func (c *Client) DeleteItem(ctx context.Context, itemType ItemType, id int64, force bool) error {
	if !itemType.core() {
		return coreOnly(itemType)
	}

	namespace, path, err := itemType.route()
	if err != nil {
		return err
	}

	query := url.Values{}
	if force {
		query.Set("force", "true")
	}

	_, _, err = c.do(ctx, request{
		method:    http.MethodDelete,
		namespace: namespace,
		path:      resourcePath(path, id),
		query:     query,
	})
	return err
}

func coreOnly(itemType ItemType) error {
	return errors.New(errors.Invalid, "only pages and posts are written through the generic item methods").
		WithDetail("type", string(itemType))
}
```

- [x] **Step 5: Run the test and watch it pass**

Run: `go test -count=1 -race ./internal/adapters/wp/...`
Expected: PASS.

- [x] **Step 6: Run the gate and commit**

```bash
go build ./... && go vet ./... && "$(go env GOPATH)/bin/golangci-lint.exe" run && go test -count=1 -race ./...
git add internal/adapters/wp
git commit -m "feat(wp): create, update and delete items

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---
### Task 9: Media, categories and the typed WooCommerce methods

**Files:**
- Create: `internal/adapters/wp/media.go`, `internal/adapters/wp/category.go`, `internal/adapters/wp/woo.go`
- Modify: `internal/adapters/wp/item.go` (`termValues`)
- Test: `internal/adapters/wp/media_test.go`, `internal/adapters/wp/woo_test.go` (package `wp_test`)

**Interfaces:**
- Consumes: task 7's payload structs and `Page[T]`, task 8's `encodeJSON` and `contentTypeJSON`.
- Produces:
  - `type Media struct { Filename, ContentType string; Bytes []byte; Alt, Title string }`, `type MediaItem struct { SourceURL, Alt, Title, MimeType string; ID int64 }`, `func (c *Client) UploadMedia(ctx context.Context, media Media) (MediaItem, error)`
  - `type Category struct { Name, Slug, Description string; ID, Parent int64; Count int }`, `type CategoryPage = Page[Category]`, `type CreateCategory struct { Parent *int64; Name, Slug, Description string }`, `func (c *Client) ListCategories(ctx context.Context, query ListQuery) (CategoryPage, error)`, `func (c *Client) CreateCategory(ctx context.Context, in CreateCategory) (Category, error)`
  - `type Product struct { Modified time.Time; Name, Slug, Permalink, Status, Description, ShortDescription string; Categories []ProductCategoryRef; ID int64; MenuOrder int }`, `type ProductCategoryRef struct { Name, Slug string; ID int64 }`, `type ProductCategory struct { Name, Slug, Description string; ID, Parent int64; Count int }`, `type ProductPage = Page[Product]`, `type ProductCategoryPage = Page[ProductCategory]`, `type UpdateProduct struct { Description, ShortDescription, Slug, Status *string; Categories []int64 }`
  - `func (c *Client) ListProducts(ctx context.Context, query ListQuery) (ProductPage, error)`, `func (c *Client) GetProduct(ctx context.Context, id int64) (Product, error)`, `func (c *Client) UpdateProduct(ctx context.Context, id int64, in UpdateProduct) (Product, error)`, `func (c *Client) ListProductCategories(ctx context.Context, query ListQuery) (ProductCategoryPage, error)`
  - `func (q ListQuery) termValues() url.Values`

- [x] **Step 1: Write the failing tests**

Create `internal/adapters/wp/media_test.go`:

```go
package wp_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestUploadMediaTakesTwoRequestsToSetAlternativeText(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	uploaded, err := newClient(t, server).UploadMedia(t.Context(), wp.Media{
		Filename:    "koffein.png",
		ContentType: "image/png",
		Bytes:       []byte{0x89, 0x50, 0x4e, 0x47},
		Alt:         "Koffein powder",
		Title:       "Koffein",
	})
	if err != nil {
		t.Fatalf("UploadMedia: %v", err)
	}

	if uploaded.ID == 0 || uploaded.SourceURL == "" || uploaded.MimeType != "image/png" {
		t.Errorf("uploaded = %+v", uploaded)
	}
	if uploaded.Alt != "Koffein powder" || uploaded.Title != "Koffein" {
		t.Errorf("uploaded = %+v, want the attributes from the second call", uploaded)
	}

	recorded := server.Requests()
	if len(recorded) != 2 {
		t.Fatalf("the client made %d requests, want an upload and an attribute write", len(recorded))
	}
	if got := recorded[0].Header.Get("Content-Disposition"); got != `attachment; filename="koffein.png"` {
		t.Errorf("Content-Disposition = %q", got)
	}
	if got := recorded[0].Header.Get("Content-Type"); got != "image/png" {
		t.Errorf("Content-Type = %q", got)
	}
	if len(recorded[0].Body) != 4 {
		t.Errorf("the upload carried %d bytes, want 4", len(recorded[0].Body))
	}
}

func TestUploadMediaSkipsTheSecondRequestWhenThereIsNothingToSet(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	if _, err := newClient(t, server).UploadMedia(t.Context(), wp.Media{
		Filename:    "koffein.png",
		ContentType: "image/png",
		Bytes:       []byte{0x89},
	}); err != nil {
		t.Fatalf("UploadMedia: %v", err)
	}
	if got := len(server.Requests()); got != 1 {
		t.Errorf("the client made %d requests, want 1", got)
	}
}

func TestUploadMediaRefusesAnEmptyUpload(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	client := newClient(t, server)

	cases := []struct {
		name  string
		media wp.Media
	}{
		{name: "no bytes", media: wp.Media{Filename: "koffein.png", ContentType: "image/png"}},
		{name: "no file name", media: wp.Media{ContentType: "image/png", Bytes: []byte{0x89}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := client.UploadMedia(t.Context(), tc.media); !errors.IsCode(err, errors.Invalid) {
				t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
			}
		})
	}
}

func TestCategoriesAreListedAndCreated(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.SeedCategory(wptest.Category{Name: "Koffein", Description: "the hub"})
	client := newClient(t, server)

	listed, err := client.ListCategories(t.Context(), wp.ListQuery{})
	if err != nil {
		t.Fatalf("ListCategories: %v", err)
	}
	if listed.Total != 1 || len(listed.Items) != 1 || listed.Items[0].Name != "Koffein" {
		t.Fatalf("listed = %+v", listed)
	}

	created, err := client.CreateCategory(t.Context(), wp.CreateCategory{Name: "Koffein", Description: "a second one"})
	if err != nil {
		t.Fatalf("CreateCategory: %v", err)
	}
	if created.Slug != "koffein-2" {
		t.Errorf("slug = %q, want koffein-2", created.Slug)
	}
	if created.Description != "a second one" {
		t.Errorf("description = %q", created.Description)
	}

	if _, err = client.CreateCategory(t.Context(), wp.CreateCategory{}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
	}
}
```

Create `internal/adapters/wp/woo_test.go`:

```go
package wp_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestProductsAreListedAndRead(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	category := server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Koffein"})[0]
	seeded := server.Seed(wptest.Item{
		Type:       wptest.TypeProduct,
		Title:      "Powder",
		Content:    "<p>long</p>",
		Excerpt:    "short",
		Status:     "publish",
		Categories: []int64{category.ID},
	})
	client := newClient(t, server)

	listed, err := client.ListProducts(t.Context(), wp.ListQuery{})
	if err != nil {
		t.Fatalf("ListProducts: %v", err)
	}
	if listed.Total != 1 || len(listed.Items) != 1 {
		t.Fatalf("listed = %+v", listed)
	}

	product, err := client.GetProduct(t.Context(), seeded[0].ID)
	if err != nil {
		t.Fatalf("GetProduct: %v", err)
	}
	if product.Name != "Powder" || product.Description != "<p>long</p>" || product.ShortDescription != "short" {
		t.Errorf("product = %+v", product)
	}
	if product.Permalink == "" || product.Modified.IsZero() {
		t.Errorf("product = %+v", product)
	}
	if len(product.Categories) != 1 || product.Categories[0].Slug != "koffein" {
		t.Errorf("categories = %+v", product.Categories)
	}
}

func TestUpdateProductWritesTheWooCommerceFieldsOnly(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	category := server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Koffein"})[0]
	seeded := server.Seed(wptest.Item{Type: wptest.TypeProduct, Title: "Powder", Content: "<p>old</p>", Status: "publish"})

	updated, err := newClient(t, server).UpdateProduct(t.Context(), seeded[0].ID, wp.UpdateProduct{
		Description:      pointerTo("<p>new</p>"),
		ShortDescription: pointerTo("brief"),
		Status:           pointerTo("draft"),
		Categories:       []int64{category.ID},
	})
	if err != nil {
		t.Fatalf("UpdateProduct: %v", err)
	}

	if updated.Description != "<p>new</p>" || updated.ShortDescription != "brief" || updated.Status != "draft" {
		t.Errorf("updated = %+v", updated)
	}
	if updated.Name != "Powder" {
		t.Errorf("name = %q; an absent field must not be overwritten", updated.Name)
	}
	if len(updated.Categories) != 1 || updated.Categories[0].ID != category.ID {
		t.Errorf("categories = %+v", updated.Categories)
	}

	recorded, _ := server.LastRequest()
	if string(recorded.Body) == "" {
		t.Fatal("the update sent no body")
	}
	if _, err = newClient(t, server).UpdateProduct(t.Context(), seeded[0].ID, wp.UpdateProduct{}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("code = %q, want %q for an empty update", errors.CodeOf(err), errors.Invalid)
	}
}

func TestProductCategoriesAreListed(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	parent := server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Koffein", Content: "the hub"})[0]
	server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Powder", Parent: parent.ID})

	listed, err := newClient(t, server).ListProductCategories(t.Context(), wp.ListQuery{})
	if err != nil {
		t.Fatalf("ListProductCategories: %v", err)
	}
	if listed.Total != 2 || len(listed.Items) != 2 {
		t.Fatalf("listed = %+v", listed)
	}
	if listed.Items[0].Name != "Koffein" || listed.Items[0].Description != "the hub" {
		t.Errorf("first = %+v", listed.Items[0])
	}
	if listed.Items[1].Parent != parent.ID {
		t.Errorf("second = %+v", listed.Items[1])
	}
}
```

- [x] **Step 2: Run the tests and watch them fail**

Run: `go test -count=1 ./internal/adapters/wp/...`
Expected: FAIL — `undefined: wp.Media`, `undefined: wp.Product`.

- [x] **Step 3: Write the media client**

Create `internal/adapters/wp/media.go`:

```go
package wp

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type Media struct {
	Filename    string
	ContentType string
	Bytes       []byte
	Alt         string
	Title       string
}

type MediaItem struct {
	SourceURL string
	Alt       string
	Title     string
	MimeType  string
	ID        int64
}

type mediaPayload struct {
	Title     renderedText `json:"title"`
	SourceURL string       `json:"source_url"`
	AltText   string       `json:"alt_text"`
	MimeType  string       `json:"mime_type"`
	ID        int64        `json:"id"`
}

func (p mediaPayload) mediaItem() MediaItem {
	return MediaItem{
		ID:        p.ID,
		SourceURL: p.SourceURL,
		Alt:       p.AltText,
		Title:     p.Title.value(),
		MimeType:  p.MimeType,
	}
}

func (c *Client) UploadMedia(ctx context.Context, media Media) (MediaItem, error) {
	filename := safeFilename(media.Filename)
	if filename == "" || len(media.Bytes) == 0 {
		return MediaItem{}, errors.New(errors.Invalid, "the upload needs a file name and file bytes")
	}

	contentType := media.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	header := http.Header{}
	header.Set("Content-Disposition", `attachment; filename="`+filename+`"`)

	_, body, err := c.do(ctx, request{
		method:      http.MethodPost,
		namespace:   coreNamespace,
		path:        "/media",
		body:        media.Bytes,
		contentType: contentType,
		header:      header,
	})
	if err != nil {
		return MediaItem{}, err
	}

	var uploaded mediaPayload
	if err = decodeJSON(body, &uploaded); err != nil {
		return MediaItem{}, err
	}

	attributes := make(map[string]any, 2)
	if media.Alt != "" {
		attributes["alt_text"] = media.Alt
	}
	if media.Title != "" {
		attributes["title"] = media.Title
	}
	if len(attributes) == 0 {
		return uploaded.mediaItem(), nil
	}

	payload, err := encodeJSON(attributes)
	if err != nil {
		return MediaItem{}, err
	}

	_, body, err = c.do(ctx, request{
		method:      http.MethodPost,
		namespace:   coreNamespace,
		path:        resourcePath("/media", uploaded.ID),
		body:        payload,
		contentType: contentTypeJSON,
	})
	if err != nil {
		return MediaItem{}, err
	}

	var described mediaPayload
	if err = decodeJSON(body, &described); err != nil {
		return MediaItem{}, err
	}
	return described.mediaItem(), nil
}

func safeFilename(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	if base == "." || base == string(filepath.Separator) {
		return ""
	}
	return strings.NewReplacer(`"`, "", "\r", "", "\n", "").Replace(base)
}
```

- [x] **Step 4: Write the category client**

Create `internal/adapters/wp/category.go`:

```go
package wp

import (
	"context"
	"net/http"
	"strings"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type Category struct {
	Name        string
	Slug        string
	Description string
	ID          int64
	Parent      int64
	Count       int
}

type CategoryPage = Page[Category]

type CreateCategory struct {
	Parent      *int64
	Name        string
	Slug        string
	Description string
}

type categoryPayload struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	ID          int64  `json:"id"`
	Parent      int64  `json:"parent"`
	Count       int    `json:"count"`
}

func (p categoryPayload) category() Category {
	return Category{
		ID:          p.ID,
		Name:        p.Name,
		Slug:        p.Slug,
		Description: p.Description,
		Parent:      p.Parent,
		Count:       p.Count,
	}
}

func (c *Client) ListCategories(ctx context.Context, query ListQuery) (CategoryPage, error) {
	resp, body, err := c.do(ctx, request{
		method:    http.MethodGet,
		namespace: coreNamespace,
		path:      "/categories",
		query:     query.termValues(),
	})
	if err != nil {
		if endOfList(err) {
			return CategoryPage{Page: query.pageNumber()}, nil
		}
		return CategoryPage{}, err
	}

	var payload []categoryPayload
	if err = decodeJSON(body, &payload); err != nil {
		return CategoryPage{}, err
	}

	categories := make([]Category, 0, len(payload))
	for _, entry := range payload {
		categories = append(categories, entry.category())
	}
	return newPage(resp, query.pageNumber(), categories), nil
}

func (c *Client) CreateCategory(ctx context.Context, in CreateCategory) (Category, error) {
	if strings.TrimSpace(in.Name) == "" {
		return Category{}, errors.New(errors.Invalid, "a WordPress category needs a name")
	}

	attributes := map[string]any{"name": in.Name}
	if in.Slug != "" {
		attributes["slug"] = in.Slug
	}
	if in.Description != "" {
		attributes["description"] = in.Description
	}
	if in.Parent != nil {
		attributes["parent"] = *in.Parent
	}

	body, err := encodeJSON(attributes)
	if err != nil {
		return Category{}, err
	}

	_, raw, err := c.do(ctx, request{
		method:      http.MethodPost,
		namespace:   coreNamespace,
		path:        "/categories",
		body:        body,
		contentType: contentTypeJSON,
	})
	if err != nil {
		return Category{}, err
	}

	var payload categoryPayload
	if err = decodeJSON(raw, &payload); err != nil {
		return Category{}, err
	}
	return payload.category(), nil
}
```

Append `termValues` to `internal/adapters/wp/item.go`:

```go
func (q ListQuery) termValues() url.Values {
	query := url.Values{}
	query.Set("context", "edit")
	query.Set("page", strconv.Itoa(q.pageNumber()))
	query.Set("per_page", strconv.Itoa(q.perPageSize()))
	query.Set("orderby", "id")
	query.Set("order", "asc")
	if len(q.Fields) > 0 {
		query.Set("_fields", strings.Join(withID(q.Fields), ","))
	}
	return query
}
```

- [x] **Step 5: Write the WooCommerce client**

Create `internal/adapters/wp/woo.go`:

```go
package wp

import (
	"context"
	"net/http"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type ProductCategoryRef struct {
	Name string
	Slug string
	ID   int64
}

type Product struct {
	Modified         time.Time
	Name             string
	Slug             string
	Permalink        string
	Status           string
	Description      string
	ShortDescription string
	Categories       []ProductCategoryRef
	ID               int64
	MenuOrder        int
}

type ProductCategory struct {
	Name        string
	Slug        string
	Description string
	ID          int64
	Parent      int64
	Count       int
}

type (
	ProductPage         = Page[Product]
	ProductCategoryPage = Page[ProductCategory]
)

type UpdateProduct struct {
	Description      *string
	ShortDescription *string
	Slug             *string
	Status           *string
	Categories       []int64
}

func (in UpdateProduct) payload() map[string]any {
	attributes := make(map[string]any)
	if in.Description != nil {
		attributes["description"] = *in.Description
	}
	if in.ShortDescription != nil {
		attributes["short_description"] = *in.ShortDescription
	}
	if in.Slug != nil {
		attributes["slug"] = *in.Slug
	}
	if in.Status != nil {
		attributes["status"] = *in.Status
	}
	if in.Categories != nil {
		refs := make([]map[string]any, 0, len(in.Categories))
		for _, id := range in.Categories {
			refs = append(refs, map[string]any{"id": id})
		}
		attributes["categories"] = refs
	}
	return attributes
}

func (p productPayload) product() Product {
	categories := make([]ProductCategoryRef, 0, len(p.Categories))
	for _, ref := range p.Categories {
		categories = append(categories, ProductCategoryRef{ID: ref.ID, Name: ref.Name, Slug: ref.Slug})
	}

	return Product{
		ID:               p.ID,
		Name:             p.Name,
		Slug:             p.Slug,
		Permalink:        p.Permalink,
		Status:           p.Status,
		Description:      p.Description,
		ShortDescription: p.ShortDescription,
		Categories:       categories,
		MenuOrder:        p.MenuOrder,
		Modified:         parseWPTime(p.DateModifiedGMT),
	}
}

func (p productCategoryPayload) productCategory() ProductCategory {
	return ProductCategory{
		ID:          p.ID,
		Name:        p.Name,
		Slug:        p.Slug,
		Description: p.Description,
		Parent:      p.Parent,
		Count:       p.Count,
	}
}

func (c *Client) ListProducts(ctx context.Context, query ListQuery) (ProductPage, error) {
	resp, body, err := c.do(ctx, request{
		method:    http.MethodGet,
		namespace: wooNamespace,
		path:      "/products",
		query:     query.values(TypeProduct),
	})
	if err != nil {
		return ProductPage{}, err
	}

	var payload []productPayload
	if err = decodeJSON(body, &payload); err != nil {
		return ProductPage{}, err
	}

	products := make([]Product, 0, len(payload))
	for _, entry := range payload {
		products = append(products, entry.product())
	}
	return newPage(resp, query.pageNumber(), products), nil
}

func (c *Client) GetProduct(ctx context.Context, id int64) (Product, error) {
	_, body, err := c.do(ctx, request{
		method:    http.MethodGet,
		namespace: wooNamespace,
		path:      resourcePath("/products", id),
	})
	if err != nil {
		return Product{}, err
	}

	var payload productPayload
	if err = decodeJSON(body, &payload); err != nil {
		return Product{}, err
	}
	return payload.product(), nil
}

func (c *Client) UpdateProduct(ctx context.Context, id int64, in UpdateProduct) (Product, error) {
	attributes := in.payload()
	if len(attributes) == 0 {
		return Product{}, errors.New(errors.Invalid, "the product update carries no fields")
	}

	body, err := encodeJSON(attributes)
	if err != nil {
		return Product{}, err
	}

	_, raw, err := c.do(ctx, request{
		method:      http.MethodPost,
		namespace:   wooNamespace,
		path:        resourcePath("/products", id),
		body:        body,
		contentType: contentTypeJSON,
	})
	if err != nil {
		return Product{}, err
	}

	var payload productPayload
	if err = decodeJSON(raw, &payload); err != nil {
		return Product{}, err
	}
	return payload.product(), nil
}

func (c *Client) ListProductCategories(ctx context.Context, query ListQuery) (ProductCategoryPage, error) {
	resp, body, err := c.do(ctx, request{
		method:    http.MethodGet,
		namespace: wooNamespace,
		path:      "/products/categories",
		query:     query.values(TypeProductCategory),
	})
	if err != nil {
		return ProductCategoryPage{}, err
	}

	var payload []productCategoryPayload
	if err = decodeJSON(body, &payload); err != nil {
		return ProductCategoryPage{}, err
	}

	categories := make([]ProductCategory, 0, len(payload))
	for _, entry := range payload {
		categories = append(categories, entry.productCategory())
	}
	return newPage(resp, query.pageNumber(), categories), nil
}
```

- [x] **Step 6: Run the tests and watch them pass**

Run: `go test -count=1 -race ./internal/adapters/wp/...`
Expected: PASS.

- [x] **Step 7: Run the gate and commit**

```bash
go build ./... && go vet ./... && "$(go env GOPATH)/bin/golangci-lint.exe" run && go test -count=1 -race ./...
git add internal/adapters/wp
git commit -m "feat(wp): media uploads, categories and woocommerce

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---
### Task 10: The fake server speaks the companion plugin namespace

This is the half of the contract from task 1 that lives in Go. It follows D14 point for point, and it implements the path and link rules independently of the client (D7, D15).

**Files:**
- Create: `internal/adapters/wp/wptest/plugin.go`
- Modify: `internal/adapters/wp/wptest/server.go` (register `s.routePlugin(mux)`)
- Test: `internal/adapters/wp/wptest/plugin_test.go` (package `wptest_test`)

**Interfaces:**
- Consumes: task 3's store and clock, task 5's `decodeBody`, `pathID`, `stringField`, `splitList`, `parseQueryTime`.
- Produces:
  - `func (s *Server) routePlugin(mux *http.ServeMux)`
  - `const pluginNamespace = "/wp-json/postulator/v1"`, `defaultContentLimit = 100`, `maxContentLimit = 500`
  - `var seoKeys map[string]map[string]string`, `var seoFieldOrder []string`
  - `func (s *Server) contentItem(stored *Item) map[string]any`, `func internalTarget(host, href string) (string, bool)`, `func normalisePath(value string) string`

- [x] **Step 1: Write the failing test**

Create `internal/adapters/wp/wptest/plugin_test.go`:

```go
package wptest_test

import (
	"net/http"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
)

const koffeinHash = "119b7cff7356b21d2b00e64d2d3c0589b50f3270b302a94360fac92f1adc332b"

func TestTheManifestReportsTheFrozenCapabilities(t *testing.T) {
	t.Parallel()

	server := wptest.New(t, wptest.WithSEOPlugin("rankmath"))
	response, payload := call(t, server, http.MethodGet, "/wp-json/postulator/v1/manifest", nil, true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}

	var manifest struct {
		Version      string   `json:"version"`
		Capabilities []string `json:"capabilities"`
		SEOPlugin    string   `json:"seoPlugin"`
		WPVersion    string   `json:"wpVersion"`
		Site         string   `json:"site"`
	}
	decode(t, payload, &manifest)

	want := []string{"bulk", "seo_meta", "content_hash", "raw"}
	if manifest.Version != "1.0.0" || len(manifest.Capabilities) != len(want) {
		t.Fatalf("manifest = %+v", manifest)
	}
	for index, capability := range want {
		if manifest.Capabilities[index] != capability {
			t.Errorf("capability %d = %q, want %q", index, manifest.Capabilities[index], capability)
		}
	}
	if manifest.SEOPlugin != "rankmath" || manifest.WPVersion == "" || manifest.Site != server.URL() {
		t.Errorf("manifest = %+v", manifest)
	}
}

func TestASiteWithoutThePluginHasNoRoutes(t *testing.T) {
	t.Parallel()

	server := wptest.New(t, wptest.WithoutPlugin())
	seeded := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein"})

	cases := []struct {
		name   string
		method string
		path   string
		body   []byte
	}{
		{name: "manifest", method: http.MethodGet, path: "/wp-json/postulator/v1/manifest"},
		{name: "content", method: http.MethodGet, path: "/wp-json/postulator/v1/content"},
		{name: "seo meta", method: http.MethodPut, path: "/wp-json/postulator/v1/seo-meta/" + itoa(seeded[0].ID), body: []byte(`{"title":"x"}`)},
		{name: "raw read", method: http.MethodGet, path: "/wp-json/postulator/v1/content/" + itoa(seeded[0].ID) + "/raw"},
		{name: "raw write", method: http.MethodPut, path: "/wp-json/postulator/v1/content/" + itoa(seeded[0].ID) + "/raw", body: []byte(`{"content":"x"}`)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			response, payload := call(t, server, tc.method, tc.path, tc.body, true)
			if response.StatusCode != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", response.StatusCode)
			}

			var failure struct {
				Code string `json:"code"`
			}
			decode(t, payload, &failure)
			if failure.Code != "rest_no_route" {
				t.Errorf("code = %q, want rest_no_route", failure.Code)
			}
		})
	}
}

func TestContentCarriesPathsHashesHeadingsAndInternalLinks(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	parent := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein"})[0]
	server.Seed(wptest.Item{
		Type:    wptest.TypePage,
		Title:   "Powder",
		Parent:  parent.ID,
		Content: `<h1>Powder</h1><p>Koffein ist ein Alkaloid.</p><p><a href="/Koffein//">Koffein</a> and <a href="https://example.com/x">away</a> and <a href="mailto:a@b.c">mail</a></p>`,
	})

	_, payload := call(t, server, http.MethodGet, "/wp-json/postulator/v1/content?types=page", nil, true)

	var page struct {
		Items []struct {
			Path        string `json:"path"`
			Title       string `json:"title"`
			H1          string `json:"h1"`
			ContentHash string `json:"contentHash"`
			Modified    string `json:"modified"`
			Links       []struct {
				Href   string `json:"href"`
				Anchor string `json:"anchor"`
			} `json:"links"`
		} `json:"items"`
		NextCursor *string `json:"nextCursor"`
	}
	decode(t, payload, &page)

	if len(page.Items) != 2 {
		t.Fatalf("got %d items, want 2", len(page.Items))
	}
	if page.NextCursor != nil {
		t.Errorf("nextCursor = %v, want null on the last page", *page.NextCursor)
	}

	child := page.Items[1]
	if child.Path != "/koffein/powder/" {
		t.Errorf("path = %q, want /koffein/powder/", child.Path)
	}
	if child.H1 != "Powder" || child.Title != "Powder" {
		t.Errorf("item = %+v", child)
	}
	if len(child.Modified) < 20 || child.Modified[len(child.Modified)-1] != 'Z' {
		t.Errorf("modified = %q, want RFC3339 with a UTC offset", child.Modified)
	}
	if len(child.Links) != 1 {
		t.Fatalf("links = %+v, want only the internal one", child.Links)
	}
	if child.Links[0].Href != "/koffein/" || child.Links[0].Anchor != "Koffein" {
		t.Errorf("link = %+v, want a collapsed and slash-wrapped path", child.Links[0])
	}
}

func TestContentPagesWithAnOpaqueCursorAndAnExclusiveSince(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := server.Seed(
		wptest.Item{Type: wptest.TypePage, Title: "One"},
		wptest.Item{Type: wptest.TypePage, Title: "Two"},
		wptest.Item{Type: wptest.TypePage, Title: "Three"},
	)

	var first struct {
		Items      []map[string]any `json:"items"`
		NextCursor *string          `json:"nextCursor"`
	}
	_, payload := call(t, server, http.MethodGet, "/wp-json/postulator/v1/content?limit=2", nil, true)
	decode(t, payload, &first)

	if len(first.Items) != 2 || first.NextCursor == nil {
		t.Fatalf("first page = %+v", first)
	}

	var second struct {
		Items      []map[string]any `json:"items"`
		NextCursor *string          `json:"nextCursor"`
	}
	_, payload = call(t, server, http.MethodGet, "/wp-json/postulator/v1/content?limit=2&cursor="+*first.NextCursor, nil, true)
	decode(t, payload, &second)

	if len(second.Items) != 1 || second.NextCursor != nil {
		t.Fatalf("second page = %+v", second)
	}

	since := seeded[1].Modified.UTC().Format("2006-01-02T15:04:05Z07:00")
	_, payload = call(t, server, http.MethodGet, "/wp-json/postulator/v1/content?since="+since, nil, true)
	decode(t, payload, &first)
	if len(first.Items) != 1 {
		t.Errorf("since returned %d items, want 1; the bound is exclusive", len(first.Items))
	}
}

func TestTheContentLimitIsClampedRatherThanRejected(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	for index := range 3 {
		server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Page " + string(rune('A'+index))})
	}

	cases := []struct {
		name  string
		query string
	}{
		{name: "above the maximum", query: "?limit=99999"},
		{name: "zero", query: "?limit=0"},
		{name: "not a number", query: "?limit=lots"},
		{name: "absent", query: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			response, payload := call(t, server, http.MethodGet, "/wp-json/postulator/v1/content"+tc.query, nil, true)
			if response.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want 200", response.StatusCode)
			}

			var page struct {
				Items []map[string]any `json:"items"`
			}
			decode(t, payload, &page)
			if len(page.Items) != 3 {
				t.Errorf("got %d items, want all 3", len(page.Items))
			}
		})
	}
}

func TestSeoMetaWritesTheKeysOfTheDetectedPlugin(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		plugin string
		key    string
	}{
		{name: "yoast", plugin: "yoast", key: "_yoast_wpseo_title"},
		{name: "rank math", plugin: "rankmath", key: "rank_math_title"},
		{name: "no plugin", plugin: "none", key: "_postulator_title"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := wptest.New(t, wptest.WithSEOPlugin(tc.plugin))
			seeded := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein"})

			body := []byte(`{"title":"Koffein","description":"about it","ogTitle":"Koffein og"}`)
			response, payload := call(t, server, http.MethodPut, "/wp-json/postulator/v1/seo-meta/"+itoa(seeded[0].ID), body, true)
			if response.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want 200", response.StatusCode)
			}

			var result struct {
				Applied   []string `json:"applied"`
				SEOPlugin string   `json:"seoPlugin"`
			}
			decode(t, payload, &result)

			if result.SEOPlugin != tc.plugin {
				t.Errorf("seoPlugin = %q, want %q", result.SEOPlugin, tc.plugin)
			}
			want := []string{"title", "description", "ogTitle"}
			if len(result.Applied) != len(want) {
				t.Fatalf("applied = %v, want %v", result.Applied, want)
			}
			for index, field := range want {
				if result.Applied[index] != field {
					t.Errorf("applied[%d] = %q, want %q", index, result.Applied[index], field)
				}
			}

			stored, _ := server.Lookup(seeded[0].ID)
			if stored.Meta[tc.key] != "Koffein" {
				t.Errorf("meta[%s] = %q, want Koffein", tc.key, stored.Meta[tc.key])
			}
		})
	}
}

func TestTheRawRoutesRoundTripAndGuardTheHash(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein", Content: "<p>Koffein ist ein Alkaloid.</p>"})

	_, payload := call(t, server, http.MethodGet, "/wp-json/postulator/v1/content/"+itoa(seeded[0].ID)+"/raw", nil, true)
	var raw struct {
		Content     string `json:"content"`
		ContentHash string `json:"contentHash"`
		Type        string `json:"type"`
	}
	decode(t, payload, &raw)

	if raw.Content != "<p>Koffein ist ein Alkaloid.</p>" || raw.ContentHash != koffeinHash {
		t.Fatalf("raw = %+v", raw)
	}
	if raw.Type != wptest.TypePage {
		t.Errorf("type = %q", raw.Type)
	}

	response, payload := call(t, server, http.MethodPut, "/wp-json/postulator/v1/content/"+itoa(seeded[0].ID)+"/raw",
		[]byte(`{"content":"<p>Powder</p>","expectedHash":"`+koffeinHash+`"}`), true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}

	var written struct {
		ContentHash string `json:"contentHash"`
	}
	decode(t, payload, &written)
	if written.ContentHash != "79db24b7a931978a1db05ccdb1ae1ed16b07aafa5277af75a7dc1b3b7ec6a509" {
		t.Errorf("contentHash = %q", written.ContentHash)
	}

	response, payload = call(t, server, http.MethodPut, "/wp-json/postulator/v1/content/"+itoa(seeded[0].ID)+"/raw",
		[]byte(`{"content":"<p>again</p>","expectedHash":"`+koffeinHash+`"}`), true)
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", response.StatusCode)
	}

	var conflict struct {
		Code        string `json:"code"`
		CurrentHash string `json:"currentHash"`
	}
	decode(t, payload, &conflict)
	if conflict.Code != "hash_mismatch" || conflict.CurrentHash != written.ContentHash {
		t.Errorf("conflict = %+v", conflict)
	}
}

func TestThePostOnlyRoutesRefuseATerm(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	term := server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Koffein"})[0]

	cases := []struct {
		name   string
		method string
		path   string
		body   []byte
	}{
		{name: "seo meta", method: http.MethodPut, path: "/wp-json/postulator/v1/seo-meta/" + itoa(term.ID), body: []byte(`{"title":"x"}`)},
		{name: "raw read", method: http.MethodGet, path: "/wp-json/postulator/v1/content/" + itoa(term.ID) + "/raw"},
		{name: "raw write", method: http.MethodPut, path: "/wp-json/postulator/v1/content/" + itoa(term.ID) + "/raw", body: []byte(`{"content":"x"}`)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			response, payload := call(t, server, tc.method, tc.path, tc.body, true)
			if response.StatusCode != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", response.StatusCode)
			}

			var failure struct {
				Code string `json:"code"`
			}
			decode(t, payload, &failure)
			if failure.Code != "not_found" {
				t.Errorf("code = %q, want not_found", failure.Code)
			}
		})
	}
}

func TestATermStillAppearsInTheContentListing(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Koffein", Content: "the hub"})

	_, payload := call(t, server, http.MethodGet, "/wp-json/postulator/v1/content?types=product_cat", nil, true)

	var page struct {
		Items []struct {
			Type     string `json:"type"`
			Modified string `json:"modified"`
			Path     string `json:"path"`
		} `json:"items"`
	}
	decode(t, payload, &page)

	if len(page.Items) != 1 {
		t.Fatalf("got %d items, want 1", len(page.Items))
	}
	if page.Items[0].Type != wptest.TypeProductCategory || page.Items[0].Modified == "" {
		t.Errorf("item = %+v; a term carries the modification date the plugin maintains", page.Items[0])
	}
	if page.Items[0].Path != "/koffein/" {
		t.Errorf("path = %q", page.Items[0].Path)
	}
}
```

- [x] **Step 2: Run the test and watch it fail**

Run: `go test -count=1 ./internal/adapters/wp/wptest/...`
Expected: FAIL — every plugin route answers Go's default 404.

- [x] **Step 3: Write the plugin handlers**

Create `internal/adapters/wp/wptest/plugin.go`:

```go
package wptest

import (
	"cmp"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	pluginNamespace     = "/wp-json/postulator/v1"
	defaultContentLimit = 100
	maxContentLimit     = 500
)

var seoFieldOrder = []string{"title", "description", "canonical", "ogTitle", "ogDescription"}

var seoKeys = map[string]map[string]string{
	"yoast": {
		"title":         "_yoast_wpseo_title",
		"description":   "_yoast_wpseo_metadesc",
		"canonical":     "_yoast_wpseo_canonical",
		"ogTitle":       "_yoast_wpseo_opengraph-title",
		"ogDescription": "_yoast_wpseo_opengraph-description",
	},
	"rankmath": {
		"title":         "rank_math_title",
		"description":   "rank_math_description",
		"canonical":     "rank_math_canonical_url",
		"ogTitle":       "rank_math_facebook_title",
		"ogDescription": "rank_math_facebook_description",
	},
	"none": {
		"title":         "_postulator_title",
		"description":   "_postulator_description",
		"canonical":     "_postulator_canonical",
		"ogTitle":       "_postulator_og_title",
		"ogDescription": "_postulator_og_description",
	},
}

var (
	headingPattern = regexp.MustCompile(`(?is)<h1[^>]*>(.*?)</h1>`)
	linkPattern    = regexp.MustCompile(`(?is)<a\s[^>]*href="([^"]*)"[^>]*>(.*?)</a>`)
	tagPattern     = regexp.MustCompile(`(?s)<[^>]*>`)
)

type cursor struct {
	Modified string `json:"m"`
	ID       int64  `json:"i"`
}

func (s *Server) routePlugin(mux *http.ServeMux) {
	mux.HandleFunc("GET "+pluginNamespace+"/manifest", s.handleManifest)
	mux.HandleFunc("GET "+pluginNamespace+"/content", s.handleContent)
	mux.HandleFunc("PUT "+pluginNamespace+"/seo-meta/{id}", s.handleSEOMeta)
	mux.HandleFunc("GET "+pluginNamespace+"/content/{id}/raw", s.handleRawGet)
	mux.HandleFunc("PUT "+pluginNamespace+"/content/{id}/raw", s.handleRawPut)
}

func (s *Server) pluginMissing(w http.ResponseWriter) bool {
	s.mu.Lock()
	missing := s.noPlugin
	s.mu.Unlock()

	if missing {
		s.fail(w, http.StatusNotFound, "rest_no_route", "No route was found matching the URL and request method")
	}
	return missing
}

func (s *Server) handleManifest(w http.ResponseWriter, _ *http.Request) {
	if s.pluginMissing(w) {
		return
	}

	s.mu.Lock()
	plugin := s.seoPlugin
	s.mu.Unlock()

	s.respond(w, http.StatusOK, map[string]any{
		"version":      "1.0.0",
		"capabilities": []string{"bulk", "seo_meta", "content_hash", "raw"},
		"seoPlugin":    plugin,
		"wpVersion":    "6.9.1",
		"site":         s.http.URL,
	})
}

func (s *Server) handleContent(w http.ResponseWriter, r *http.Request) {
	if s.pluginMissing(w) {
		return
	}

	query := r.URL.Query()
	types := splitList(query.Get("types"))
	if len(types) == 0 {
		types = []string{TypePage, TypePost, TypeProduct, TypeProductCategory}
	}

	after, ok := decodeCursor(query.Get("cursor"))
	if !ok {
		s.fail(w, http.StatusBadRequest, "invalid_cursor", "The cursor could not be read.")
		return
	}

	since := parseQueryTime(query.Get("since"))
	limit := contentLimit(query.Get("limit"))

	s.mu.Lock()
	matched := make([]*Item, 0, len(s.order))
	for _, id := range s.order {
		stored := s.items[id]
		if !slices.Contains(types, stored.Type) {
			continue
		}
		if !since.IsZero() && !stored.Modified.After(since) {
			continue
		}
		if !afterCursor(stored, after) {
			continue
		}
		matched = append(matched, stored)
	}
	slices.SortFunc(matched, func(left, right *Item) int {
		if left.Modified.Equal(right.Modified) {
			return cmp.Compare(left.ID, right.ID)
		}
		return left.Modified.Compare(right.Modified)
	})

	var next any
	if len(matched) > limit {
		matched = matched[:limit]
		last := matched[len(matched)-1]

		encoded, err := encodeCursor(cursor{Modified: last.Modified.UTC().Format(time.RFC3339), ID: last.ID})
		if err != nil {
			s.mu.Unlock()
			s.fail(w, http.StatusInternalServerError, "cursor_failed", "The next cursor could not be built.")
			return
		}
		next = encoded
	}

	items := make([]map[string]any, 0, len(matched))
	for _, stored := range matched {
		items = append(items, s.contentItem(stored))
	}
	s.mu.Unlock()

	s.respond(w, http.StatusOK, map[string]any{"items": items, "nextCursor": next})
}

func (s *Server) handleSEOMeta(w http.ResponseWriter, r *http.Request) {
	if s.pluginMissing(w) {
		return
	}

	id, ok := pathID(r)
	if !ok {
		s.fail(w, http.StatusNotFound, "not_found", "No content with that id exists.")
		return
	}

	body, ok := s.decodeBody(w, r)
	if !ok {
		return
	}

	s.mu.Lock()
	stored, found := s.items[id]
	if !found || !postType(stored.Type) {
		s.mu.Unlock()
		s.fail(w, http.StatusNotFound, "not_found", "No content with that id exists.")
		return
	}

	keys := seoKeys[s.seoPlugin]
	applied := make([]string, 0, len(seoFieldOrder))
	for _, field := range seoFieldOrder {
		value := stringField(body, field)
		if value == "" {
			continue
		}
		stored.Meta[keys[field]] = value
		applied = append(applied, field)
	}
	plugin := s.seoPlugin
	s.mu.Unlock()

	s.respond(w, http.StatusOK, map[string]any{"applied": applied, "seoPlugin": plugin})
}

func (s *Server) handleRawGet(w http.ResponseWriter, r *http.Request) {
	if s.pluginMissing(w) {
		return
	}

	id, ok := pathID(r)
	if !ok {
		s.fail(w, http.StatusNotFound, "not_found", "No content with that id exists.")
		return
	}

	s.mu.Lock()
	stored, found := s.items[id]
	var payload map[string]any
	if found && postType(stored.Type) {
		payload = map[string]any{
			"id":          stored.ID,
			"type":        stored.Type,
			"content":     stored.Content,
			"contentHash": s.reportedHash(stored.Content),
		}
	}
	s.mu.Unlock()

	if payload == nil {
		s.fail(w, http.StatusNotFound, "not_found", "No content with that id exists.")
		return
	}
	s.respond(w, http.StatusOK, payload)
}

func (s *Server) handleRawPut(w http.ResponseWriter, r *http.Request) {
	if s.pluginMissing(w) {
		return
	}

	id, ok := pathID(r)
	if !ok {
		s.fail(w, http.StatusNotFound, "not_found", "No content with that id exists.")
		return
	}

	body, ok := s.decodeBody(w, r)
	if !ok {
		return
	}

	content, present := body["content"].(string)
	if !present {
		s.fail(w, http.StatusBadRequest, "missing_content", "The content field is required.")
		return
	}
	expected := stringField(body, "expectedHash")

	s.mu.Lock()
	stored, found := s.items[id]
	if !found || !postType(stored.Type) {
		s.mu.Unlock()
		s.fail(w, http.StatusNotFound, "not_found", "No content with that id exists.")
		return
	}

	current := contentHash(stored.Content)
	if expected != "" && expected != current {
		s.mu.Unlock()
		s.respond(w, http.StatusConflict, map[string]any{
			"code":        "hash_mismatch",
			"message":     "The stored content changed since it was read.",
			"currentHash": current,
		})
		return
	}

	stored.Content = content
	stored.Modified = s.tick()
	updated := contentHash(content)
	s.mu.Unlock()

	s.respond(w, http.StatusOK, map[string]any{"contentHash": updated})
}

func (s *Server) contentItem(stored *Item) map[string]any {
	keys := seoKeys[s.seoPlugin]
	return map[string]any{
		"id":          stored.ID,
		"type":        stored.Type,
		"slug":        stored.Slug,
		"path":        normalisePath(s.itemPath(stored)),
		"parent":      stored.Parent,
		"status":      stored.Status,
		"modified":    stored.Modified.UTC().Format(time.RFC3339),
		"contentHash": contentHash(stored.Content),
		"title":       stored.Title,
		"h1":          headingOne(stored),
		"meta": map[string]any{
			"title":       stored.Meta[keys["title"]],
			"description": stored.Meta[keys["description"]],
			"canonical":   stored.Meta[keys["canonical"]],
		},
		"links": s.internalLinks(stored.Content),
	}
}

func postType(itemType string) bool {
	return itemType != TypeProductCategory
}

func (s *Server) reportedHash(content string) string {
	if s.brokenHash {
		return strings.Repeat("0", 64)
	}
	return contentHash(content)
}

func headingOne(stored *Item) string {
	if stored.H1 != "" {
		return stored.H1
	}

	match := headingPattern.FindStringSubmatch(stored.Content)
	if match == nil {
		return ""
	}
	return strings.TrimSpace(tagPattern.ReplaceAllString(match[1], ""))
}

func (s *Server) internalLinks(content string) []map[string]any {
	host := hostOf(s.http.URL)
	links := make([]map[string]any, 0)
	for _, match := range linkPattern.FindAllStringSubmatch(content, -1) {
		target, ok := internalTarget(host, match[1])
		if !ok {
			continue
		}
		links = append(links, map[string]any{
			"href":   target,
			"anchor": strings.TrimSpace(tagPattern.ReplaceAllString(match[2], "")),
		})
	}
	return links
}

func hostOf(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return parsed.Host
}

func internalTarget(host, href string) (string, bool) {
	trimmed := strings.TrimSpace(href)
	if trimmed == "" {
		return "", false
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", false
	}
	if parsed.Scheme != "" && parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", false
	}
	if parsed.Host != "" && !strings.EqualFold(parsed.Host, host) {
		return "", false
	}

	target := parsed.EscapedPath()
	if target == "" && parsed.Host == "" {
		return "", false
	}
	return normalisePath(target), true
}

func normalisePath(value string) string {
	var builder strings.Builder
	builder.Grow(len(value) + 2)
	builder.WriteByte('/')

	slashed := true
	for index := range len(value) {
		symbol := value[index]
		if symbol == '/' {
			if slashed {
				continue
			}
			slashed = true
			builder.WriteByte('/')
			continue
		}
		slashed = false
		if symbol >= 'A' && symbol <= 'Z' {
			symbol += 'a' - 'A'
		}
		builder.WriteByte(symbol)
	}

	normalised := builder.String()
	if strings.HasSuffix(normalised, "/") {
		return normalised
	}
	return normalised + "/"
}

func contentLimit(raw string) int {
	if raw == "" {
		return defaultContentLimit
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return defaultContentLimit
	}
	if parsed > maxContentLimit {
		return maxContentLimit
	}
	return parsed
}

func encodeCursor(value cursor) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodeCursor(value string) (cursor, bool) {
	if value == "" {
		return cursor{}, true
	}

	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return cursor{}, false
	}

	var decoded cursor
	if err = json.Unmarshal(raw, &decoded); err != nil {
		return cursor{}, false
	}
	return decoded, true
}

func afterCursor(stored *Item, after cursor) bool {
	if after.Modified == "" {
		return true
	}

	bound, err := time.Parse(time.RFC3339, after.Modified)
	if err != nil {
		return true
	}
	if stored.Modified.After(bound) {
		return true
	}
	return stored.Modified.Equal(bound) && stored.ID > after.ID
}
```

- [x] **Step 4: Add the corrupt-hash fault**

Add the field `brokenHash bool` to `Server` in `server.go`, and the option to `fault.go`:

```go
func WithBrokenContentHash() Option {
	return func(s *Server) { s.brokenHash = true }
}
```

A site whose PHP hashes something other than the raw `post_content` is the failure mode D5 exists to prevent, and this is how the client's guard against it gets exercised. Add the case to `plugin_test.go`:

```go
func TestABrokenSiteCanReportTheWrongHash(t *testing.T) {
	t.Parallel()

	server := wptest.New(t, wptest.WithBrokenContentHash())
	seeded := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein", Content: "<p>Koffein ist ein Alkaloid.</p>"})

	_, payload := call(t, server, http.MethodGet, "/wp-json/postulator/v1/content/"+itoa(seeded[0].ID)+"/raw", nil, true)

	var raw struct {
		ContentHash string `json:"contentHash"`
	}
	decode(t, payload, &raw)
	if raw.ContentHash == koffeinHash {
		t.Error("the broken site must not report the correct hash")
	}
}
```

- [x] **Step 5: Register the plugin routes**

Extend `handler()` in `server.go` with `s.routePlugin(mux)`, so the final form is:

```go
func (s *Server) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+rootPath, s.handleRoot)
	s.routeCore(mux)
	s.routeCategories(mux)
	s.routeMedia(mux)
	s.routeWoo(mux)
	s.routePlugin(mux)

	return s.record(s.redirectRoot(s.injectFaults(s.authenticate(mux))))
}
```

- [x] **Step 6: Run the tests and watch them pass**

Run: `go test -count=1 -race ./internal/adapters/wp/...`
Expected: PASS.

- [x] **Step 7: Run the gate and commit**

```bash
go build ./... && go vet ./... && "$(go env GOPATH)/bin/golangci-lint.exe" run && go test -count=1 -race ./...
git add internal/adapters/wp/wptest
git commit -m "test(wp): companion plugin namespace in the fake server

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 11: The exported internal-link normaliser

**Files:**
- Create: `internal/adapters/wp/link.go`
- Test: `internal/adapters/wp/link_test.go` (package `wp_test`)

**Interfaces:**
- Consumes: nothing.
- Produces: `func NormalizePath(rawPath string) string`, `func InternalPath(siteHost, href string) (string, bool)`

- [x] **Step 1: Write the failing test**

Create `internal/adapters/wp/link_test.go`:

```go
package wp_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp"
)

func TestNormalizePath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		path string
		want string
	}{
		{name: "empty is the root", path: "", want: "/"},
		{name: "root stays the root", path: "/", want: "/"},
		{name: "wraps in slashes", path: "koffein/powder", want: "/koffein/powder/"},
		{name: "keeps a trailing slash", path: "/koffein/powder/", want: "/koffein/powder/"},
		{name: "collapses duplicates", path: "//koffein///powder//", want: "/koffein/powder/"},
		{name: "no file extension exception", path: "/koffein/powder.html", want: "/koffein/powder.html/"},
		{name: "lowercases ascii", path: "/Koffein/Powder/", want: "/koffein/powder/"},
		{name: "does not decode percent escapes", path: "/koffein/gr%C3%BCner-tee", want: "/koffein/gr%c3%bcner-tee/"},
		{name: "leaves non ascii bytes alone", path: "/koffein/Grüner-Tee", want: "/koffein/grüner-tee/"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := wp.NormalizePath(tc.path); got != tc.want {
				t.Errorf("NormalizePath(%q) = %q, want %q", tc.path, got, tc.want)
			}
		})
	}
}

func TestInternalPath(t *testing.T) {
	t.Parallel()

	const host = "example.com"

	cases := []struct {
		name     string
		href     string
		want     string
		internal bool
	}{
		{name: "relative", href: "koffein/powder", want: "/koffein/powder/", internal: true},
		{name: "absolute path", href: "/koffein/", want: "/koffein/", internal: true},
		{name: "same host over https", href: "https://example.com/koffein/", want: "/koffein/", internal: true},
		{name: "same host over http", href: "http://example.com/koffein/", want: "/koffein/", internal: true},
		{name: "host case is ignored", href: "https://EXAMPLE.COM/Koffein/", want: "/koffein/", internal: true},
		{name: "protocol relative", href: "//example.com/koffein/", want: "/koffein/", internal: true},
		{name: "host root", href: "https://example.com", want: "/", internal: true},
		{name: "query is stripped", href: "/koffein/?utm=1", want: "/koffein/", internal: true},
		{name: "fragment is stripped", href: "/koffein/#top", want: "/koffein/", internal: true},
		{name: "percent escapes are not decoded", href: "/gr%C3%BCner-tee", want: "/gr%c3%bcner-tee/", internal: true},
		{name: "www is a different host", href: "https://www.example.com/koffein/"},
		{name: "another host", href: "https://other.example/koffein/"},
		{name: "a port makes it another host", href: "https://example.com:8080/koffein/"},
		{name: "mail is not a link", href: "mailto:hello@example.com"},
		{name: "telephone is not a link", href: "tel:+123"},
		{name: "a bare fragment is not a link", href: "#top"},
		{name: "a bare query is not a link", href: "?utm=1"},
		{name: "empty", href: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, internal := wp.InternalPath(host, tc.href)
			if internal != tc.internal {
				t.Fatalf("InternalPath(%q) internal = %t, want %t", tc.href, internal, tc.internal)
			}
			if got != tc.want {
				t.Errorf("InternalPath(%q) = %q, want %q", tc.href, got, tc.want)
			}
		})
	}
}
```

- [x] **Step 2: Run the test and watch it fail**

Run: `go test -count=1 ./internal/adapters/wp/...`
Expected: FAIL — `undefined: wp.NormalizePath`.

- [x] **Step 3: Write the normaliser**

Create `internal/adapters/wp/link.go`:

```go
package wp

import (
	"net/url"
	"strings"
)

func NormalizePath(rawPath string) string {
	var builder strings.Builder
	builder.Grow(len(rawPath) + 2)
	builder.WriteByte('/')

	slashed := true
	for index := range len(rawPath) {
		symbol := rawPath[index]
		if symbol == '/' {
			if slashed {
				continue
			}
			slashed = true
			builder.WriteByte('/')
			continue
		}
		slashed = false
		if symbol >= 'A' && symbol <= 'Z' {
			symbol += 'a' - 'A'
		}
		builder.WriteByte(symbol)
	}

	normalised := builder.String()
	if strings.HasSuffix(normalised, "/") {
		return normalised
	}
	return normalised + "/"
}

func InternalPath(siteHost, href string) (string, bool) {
	trimmed := strings.TrimSpace(href)
	if trimmed == "" {
		return "", false
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", false
	}
	if parsed.Scheme != "" && parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", false
	}
	if parsed.Host != "" && !strings.EqualFold(parsed.Host, siteHost) {
		return "", false
	}

	target := parsed.EscapedPath()
	if target == "" && parsed.Host == "" {
		return "", false
	}
	return NormalizePath(target), true
}
```

- [x] **Step 4: Run the test and watch it pass**

Run: `go test -count=1 -race ./internal/adapters/wp/...`
Expected: PASS.

- [x] **Step 5: Run the gate and commit**

```bash
go build ./... && go vet ./... && "$(go env GOPATH)/bin/golangci-lint.exe" run && go test -count=1 -race ./...
git add internal/adapters/wp
git commit -m "feat(wp): internal link path normalisation

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---
### Task 12: The plugin client and capability caching

**Files:**
- Create: `internal/adapters/wp/plugin.go`
- Modify: `internal/adapters/wp/client.go` (the manifest cache fields), `internal/adapters/wp/contract_test.go` (the route-coverage assertion)
- Test: `internal/adapters/wp/plugin_test.go` (package `wp_test`)

**Interfaces:**
- Consumes: task 2's `Client`, task 4's `do` and `decodeJSON`, task 7's `resourcePath` and `parseWPTime`, task 8's `encodeJSON`, task 10's fake namespace.
- Produces:
  - `type Manifest struct { Version string; Capabilities []string; SEOPlugin, WPVersion, Site string }`
  - `type Capabilities struct { Version, SEOPlugin, WPVersion, Site string; Names []string }` with `func (c Capabilities) Has(name string) bool`
  - `type ContentQuery struct { Since *time.Time; Cursor string; Types []ItemType; Limit int }`
  - `type ContentMeta struct { Title, Description, Canonical string }`, `type ContentLink struct { Href, Anchor string }`
  - `type ContentItem struct { Modified time.Time; Type ItemType; Slug, Path, Status, ContentHash, Title, H1 string; Meta ContentMeta; Links []ContentLink; ID, Parent int64 }`
  - `type ContentPage struct { Items []ContentItem; NextCursor *string }`
  - `type SEOMeta struct { Title, Description, Canonical, OGTitle, OGDescription string }`, `type SEOResult struct { Applied []string; SEOPlugin string }`, `type RawContent struct { Type ItemType; Content, ContentHash string; ID int64 }`
  - `func (c *Client) Manifest(ctx context.Context) (Manifest, error)`
  - `func (c *Client) Capabilities(ctx context.Context) (Capabilities, error)`
  - `func (c *Client) ListContent(ctx context.Context, query ContentQuery) (ContentPage, error)`
  - `func (c *Client) SetSEOMeta(ctx context.Context, id int64, meta SEOMeta) (SEOResult, error)`
  - `func (c *Client) GetRaw(ctx context.Context, id int64) (RawContent, error)`
  - `func (c *Client) PutRaw(ctx context.Context, id int64, content, expectedHash string) (string, error)`

- [x] **Step 1: Write the failing test**

Create `internal/adapters/wp/plugin_test.go`:

```go
package wp_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	koffeinBody = "<p>Koffein ist ein Alkaloid.</p>"
	koffeinHash = "119b7cff7356b21d2b00e64d2d3c0589b50f3270b302a94360fac92f1adc332b"
	powderHash  = "79db24b7a931978a1db05ccdb1ae1ed16b07aafa5277af75a7dc1b3b7ec6a509"
)

func TestCapabilitiesAreFetchedOnceAndCached(t *testing.T) {
	t.Parallel()

	server := wptest.New(t, wptest.WithSEOPlugin("rankmath"))
	client := newClient(t, server)

	first, err := client.Capabilities(t.Context())
	if err != nil {
		t.Fatalf("Capabilities: %v", err)
	}
	if first.SEOPlugin != "rankmath" || first.Version != "1.0.0" || first.Site != server.URL() {
		t.Errorf("capabilities = %+v", first)
	}
	if !first.Has("raw") || !first.Has("seo_meta") || first.Has("telepathy") {
		t.Errorf("names = %v", first.Names)
	}

	if _, err = client.Capabilities(t.Context()); err != nil {
		t.Fatalf("Capabilities again: %v", err)
	}
	if got := len(server.Requests()); got != 1 {
		t.Errorf("the client asked for the manifest %d times, want 1", got)
	}
}

func TestEveryPluginMethodDegradesWhenThePluginIsAbsent(t *testing.T) {
	t.Parallel()

	server := wptest.New(t, wptest.WithoutPlugin())
	seeded := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein", Content: koffeinBody})
	client := newClient(t, server)

	calls := []struct {
		name string
		call func() error
	}{
		{name: "manifest", call: func() error { _, err := client.Manifest(t.Context()); return err }},
		{name: "capabilities", call: func() error { _, err := client.Capabilities(t.Context()); return err }},
		{name: "content", call: func() error { _, err := client.ListContent(t.Context(), wp.ContentQuery{}); return err }},
		{name: "seo meta", call: func() error {
			_, err := client.SetSEOMeta(t.Context(), seeded[0].ID, wp.SEOMeta{Title: "x"})
			return err
		}},
		{name: "raw read", call: func() error { _, err := client.GetRaw(t.Context(), seeded[0].ID); return err }},
		{name: "raw write", call: func() error {
			_, err := client.PutRaw(t.Context(), seeded[0].ID, "x", "")
			return err
		}},
	}

	for _, tc := range calls {
		err := tc.call()
		if !errors.IsCode(err, errors.Invalid) {
			t.Errorf("%s code = %q, want %q", tc.name, errors.CodeOf(err), errors.Invalid)
		}
		if got := detailOf(t, err, "code"); got != "plugin_missing" {
			t.Errorf("%s detail = %q, want plugin_missing", tc.name, got)
		}
	}

	if got := len(server.Requests()); got != 1 {
		t.Errorf("the client probed the manifest %d times, want 1; absence is cached", got)
	}
}

func TestATransientManifestFailureIsNotCached(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.FailNext(http.StatusInternalServerError, 1)
	client := newClient(t, server, wp.WithRetries(0))

	if _, err := client.Capabilities(t.Context()); !errors.IsCode(err, errors.External) {
		t.Fatalf("code = %q, want %q", errors.CodeOf(err), errors.External)
	}
	if _, err := client.Capabilities(t.Context()); err != nil {
		t.Errorf("a site that recovered must be usable again: %v", err)
	}
}

func TestListContentWalksTheOpaqueCursor(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.Seed(
		wptest.Item{Type: wptest.TypePage, Title: "One"},
		wptest.Item{Type: wptest.TypePage, Title: "Two"},
		wptest.Item{Type: wptest.TypePage, Title: "Three"},
	)
	client := newClient(t, server)

	seen := 0
	query := wp.ContentQuery{Types: []wp.ItemType{wp.TypePage}, Limit: 2}
	for {
		page, err := client.ListContent(t.Context(), query)
		if err != nil {
			t.Fatalf("ListContent: %v", err)
		}
		seen += len(page.Items)
		if page.NextCursor == nil {
			break
		}
		query.Cursor = *page.NextCursor
	}

	if seen != 3 {
		t.Errorf("walked %d items, want 3", seen)
	}
}

func TestListContentCarriesTheWholeItem(t *testing.T) {
	t.Parallel()

	const powderContent = `<h1>Powder</h1>` + koffeinBody + `<p><a href="/koffein/">Koffein</a></p>`

	server := wptest.New(t)
	parent := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein"})[0]
	server.Seed(wptest.Item{
		Type:    wptest.TypePage,
		Title:   "Powder",
		Parent:  parent.ID,
		Content: powderContent,
		Meta:    map[string]string{"_yoast_wpseo_title": "Powder | Koffein"},
	})

	page, err := newClient(t, server).ListContent(t.Context(), wp.ContentQuery{Types: []wp.ItemType{wp.TypePage}})
	if err != nil {
		t.Fatalf("ListContent: %v", err)
	}
	if len(page.Items) != 2 || page.NextCursor != nil {
		t.Fatalf("page = %+v", page)
	}

	child := page.Items[1]
	if child.Path != "/koffein/powder/" || child.Parent != parent.ID || child.Type != wp.TypePage {
		t.Errorf("item = %+v", child)
	}
	if child.Title != "Powder" || child.H1 != "Powder" || child.Status != "publish" {
		t.Errorf("item = %+v", child)
	}
	if child.ContentHash != wp.ContentHash(powderContent) {
		t.Errorf("contentHash = %q, want the hash of the raw content", child.ContentHash)
	}
	if child.Modified.IsZero() {
		t.Error("the item must carry a modification instant")
	}
	if child.Meta.Title != "Powder | Koffein" {
		t.Errorf("meta = %+v", child.Meta)
	}
	if len(child.Links) != 1 || child.Links[0].Href != "/koffein/" || child.Links[0].Anchor != "Koffein" {
		t.Errorf("links = %+v", child.Links)
	}
}

func TestTheContentQueryClampsTheLimitAndNamesTheTypes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		query wp.ContentQuery
		limit string
	}{
		{name: "absent", query: wp.ContentQuery{}, limit: "100"},
		{name: "negative", query: wp.ContentQuery{Limit: -1}, limit: "100"},
		{name: "above the maximum", query: wp.ContentQuery{Limit: 99999}, limit: "500"},
		{name: "in range", query: wp.ContentQuery{Limit: 25}, limit: "25"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := wptest.New(t)
			if _, err := newClient(t, server).ListContent(t.Context(), tc.query); err != nil {
				t.Fatalf("ListContent: %v", err)
			}

			recorded, _ := server.LastRequest()
			if got := recorded.Query.Get("limit"); got != tc.limit {
				t.Errorf("limit = %q, want %q", got, tc.limit)
			}
		})
	}
}

func TestTheContentQuerySendsSinceAndTypes(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	since := time.Date(2026, time.September, 18, 9, 0, 0, 0, time.UTC)

	if _, err := newClient(t, server).ListContent(t.Context(), wp.ContentQuery{
		Since: &since,
		Types: []wp.ItemType{wp.TypePage, wp.TypeProductCategory},
	}); err != nil {
		t.Fatalf("ListContent: %v", err)
	}

	recorded, _ := server.LastRequest()
	if got := recorded.Query.Get("since"); got != "2026-09-18T09:00:00Z" {
		t.Errorf("since = %q", got)
	}
	if got := recorded.Query.Get("types"); got != "page,product_cat" {
		t.Errorf("types = %q", got)
	}
}

func TestSetSEOMetaReportsWhatWasWritten(t *testing.T) {
	t.Parallel()

	server := wptest.New(t, wptest.WithSEOPlugin("yoast"))
	seeded := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein"})
	client := newClient(t, server)

	result, err := client.SetSEOMeta(t.Context(), seeded[0].ID, wp.SEOMeta{
		Title:       "Koffein",
		Description: "about it",
	})
	if err != nil {
		t.Fatalf("SetSEOMeta: %v", err)
	}
	if result.SEOPlugin != "yoast" {
		t.Errorf("seoPlugin = %q", result.SEOPlugin)
	}
	if len(result.Applied) != 2 || result.Applied[0] != "title" || result.Applied[1] != "description" {
		t.Errorf("applied = %v", result.Applied)
	}

	stored, _ := server.Lookup(seeded[0].ID)
	if stored.Meta["_yoast_wpseo_metadesc"] != "about it" {
		t.Errorf("meta = %v", stored.Meta)
	}

	if _, err = client.SetSEOMeta(t.Context(), seeded[0].ID, wp.SEOMeta{}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("code = %q, want %q for an empty update", errors.CodeOf(err), errors.Invalid)
	}
}

func TestTheRawRoundTripIsGuardedByTheHash(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein", Content: koffeinBody})
	client := newClient(t, server)

	raw, err := client.GetRaw(t.Context(), seeded[0].ID)
	if err != nil {
		t.Fatalf("GetRaw: %v", err)
	}
	if raw.Content != koffeinBody || raw.ContentHash != koffeinHash || raw.Type != wp.TypePage {
		t.Fatalf("raw = %+v", raw)
	}

	written, err := client.PutRaw(t.Context(), seeded[0].ID, "<p>Powder</p>", raw.ContentHash)
	if err != nil {
		t.Fatalf("PutRaw: %v", err)
	}
	if written != powderHash {
		t.Errorf("hash = %q, want %q", written, powderHash)
	}

	_, err = client.PutRaw(t.Context(), seeded[0].ID, "<p>again</p>", raw.ContentHash)
	if !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("code = %q, want %q", errors.CodeOf(err), errors.Conflict)
	}
	if got := detailOf(t, err, "currentHash"); got != powderHash {
		t.Errorf("currentHash detail = %q, want %q", got, powderHash)
	}

	unconditional, err := client.PutRaw(t.Context(), seeded[0].ID, koffeinBody, "")
	if err != nil {
		t.Fatalf("PutRaw without a hash: %v", err)
	}
	if unconditional != koffeinHash {
		t.Errorf("hash = %q, want %q", unconditional, koffeinHash)
	}
}

func TestGetRawRefusesAHashThatDoesNotMatchTheContent(t *testing.T) {
	t.Parallel()

	server := wptest.New(t, wptest.WithBrokenContentHash())
	seeded := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein", Content: koffeinBody})

	_, err := newClient(t, server).GetRaw(t.Context(), seeded[0].ID)
	if !errors.IsCode(err, errors.External) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.External)
	}
}

func TestThePostOnlyRoutesReportATermAsMissing(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	term := server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Koffein"})[0]
	client := newClient(t, server)

	if _, err := client.GetRaw(t.Context(), term.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("raw read code = %q, want %q", errors.CodeOf(err), errors.NotFound)
	}
	if _, err := client.PutRaw(t.Context(), term.ID, "x", ""); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("raw write code = %q, want %q", errors.CodeOf(err), errors.NotFound)
	}
	if _, err := client.SetSEOMeta(t.Context(), term.ID, wp.SEOMeta{Title: "x"}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("seo meta code = %q, want %q", errors.CodeOf(err), errors.NotFound)
	}
}
```

Add the detail reader the tests share to `do_test.go`:

```go
func detailOf(t *testing.T, err error, key string) string {
	t.Helper()

	var kernel *errors.Error
	if !stderrors.As(err, &kernel) || kernel == nil {
		return ""
	}
	value, ok := kernel.Details[key].(string)
	if !ok {
		return ""
	}
	return value
}
```

- [x] **Step 2: Run the test and watch it fail**

Run: `go test -count=1 ./internal/adapters/wp/...`
Expected: FAIL — `undefined: wp.ContentQuery`.

- [x] **Step 3: Give the client its manifest cache**

Add the three fields to `Client` in `client.go`, below `retries`, and `"sync"` to that file's import block. They arrive now rather than in task 2 because `Manifest` is declared in this task, and a struct field of a type that does not exist yet does not compile.

```go
	manifestMu   sync.Mutex
	manifest     *Manifest
	manifestGone bool
}
```

- [x] **Step 4: Write the plugin client**

Create `internal/adapters/wp/plugin.go`:

```go
package wp

import (
	"context"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	defaultContentLimit = 100
	maxContentLimit     = 500
)

type Manifest struct {
	Version      string
	SEOPlugin    string
	WPVersion    string
	Site         string
	Capabilities []string
}

type Capabilities struct {
	Version   string
	SEOPlugin string
	WPVersion string
	Site      string
	Names     []string
}

func (c Capabilities) Has(name string) bool {
	return slices.Contains(c.Names, name)
}

type ContentQuery struct {
	Since  *time.Time
	Cursor string
	Types  []ItemType
	Limit  int
}

func (q ContentQuery) limitValue() int {
	switch {
	case q.Limit <= 0:
		return defaultContentLimit
	case q.Limit > maxContentLimit:
		return maxContentLimit
	default:
		return q.Limit
	}
}

func (q ContentQuery) values() url.Values {
	query := url.Values{}
	if q.Since != nil {
		query.Set("since", q.Since.UTC().Format(time.RFC3339))
	}
	if q.Cursor != "" {
		query.Set("cursor", q.Cursor)
	}
	if len(q.Types) > 0 {
		names := make([]string, 0, len(q.Types))
		for _, itemType := range q.Types {
			names = append(names, string(itemType))
		}
		query.Set("types", strings.Join(names, ","))
	}
	query.Set("limit", strconv.Itoa(q.limitValue()))
	return query
}

type ContentMeta struct {
	Title       string
	Description string
	Canonical   string
}

type ContentLink struct {
	Href   string
	Anchor string
}

type ContentItem struct {
	Modified    time.Time
	Type        ItemType
	Slug        string
	Path        string
	Status      string
	ContentHash string
	Title       string
	H1          string
	Meta        ContentMeta
	Links       []ContentLink
	ID          int64
	Parent      int64
}

type ContentPage struct {
	Items      []ContentItem
	NextCursor *string
}

type SEOMeta struct {
	Title         string `json:"title,omitempty"`
	Description   string `json:"description,omitempty"`
	Canonical     string `json:"canonical,omitempty"`
	OGTitle       string `json:"ogTitle,omitempty"`
	OGDescription string `json:"ogDescription,omitempty"`
}

type SEOResult struct {
	Applied   []string
	SEOPlugin string
}

type RawContent struct {
	Type        ItemType
	Content     string
	ContentHash string
	ID          int64
}

type manifestPayload struct {
	Version      string   `json:"version"`
	SEOPlugin    string   `json:"seoPlugin"`
	WPVersion    string   `json:"wpVersion"`
	Site         string   `json:"site"`
	Capabilities []string `json:"capabilities"`
}

type contentMetaPayload struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Canonical   string `json:"canonical"`
}

type contentLinkPayload struct {
	Href   string `json:"href"`
	Anchor string `json:"anchor"`
}

type contentItemPayload struct {
	Type        string               `json:"type"`
	Slug        string               `json:"slug"`
	Path        string               `json:"path"`
	Status      string               `json:"status"`
	Modified    string               `json:"modified"`
	ContentHash string               `json:"contentHash"`
	Title       string               `json:"title"`
	H1          string               `json:"h1"`
	Meta        contentMetaPayload   `json:"meta"`
	Links       []contentLinkPayload `json:"links"`
	ID          int64                `json:"id"`
	Parent      int64                `json:"parent"`
}

func (p contentItemPayload) contentItem() ContentItem {
	links := make([]ContentLink, 0, len(p.Links))
	for _, link := range p.Links {
		links = append(links, ContentLink{Href: link.Href, Anchor: link.Anchor})
	}

	return ContentItem{
		ID:          p.ID,
		Type:        ItemType(p.Type),
		Slug:        p.Slug,
		Path:        p.Path,
		Parent:      p.Parent,
		Status:      p.Status,
		Modified:    parseWPTime(p.Modified),
		ContentHash: p.ContentHash,
		Title:       p.Title,
		H1:          p.H1,
		Meta:        ContentMeta{Title: p.Meta.Title, Description: p.Meta.Description, Canonical: p.Meta.Canonical},
		Links:       links,
	}
}

type contentPagePayload struct {
	NextCursor *string              `json:"nextCursor"`
	Items      []contentItemPayload `json:"items"`
}

type rawPayload struct {
	Type        string `json:"type"`
	Content     string `json:"content"`
	ContentHash string `json:"contentHash"`
	ID          int64  `json:"id"`
}

type seoResultPayload struct {
	SEOPlugin string   `json:"seoPlugin"`
	Applied   []string `json:"applied"`
}

func pluginMissing() error {
	return errors.New(errors.Invalid, "the Postulator companion plugin is not installed on this site").
		WithDetail("code", "plugin_missing")
}

func (c *Client) Manifest(ctx context.Context) (Manifest, error) {
	c.manifestMu.Lock()
	defer c.manifestMu.Unlock()

	if c.manifestGone {
		return Manifest{}, pluginMissing()
	}
	if c.manifest != nil {
		return *c.manifest, nil
	}

	_, body, err := c.do(ctx, request{method: http.MethodGet, namespace: pluginNamespace, path: "/manifest"})
	if err != nil {
		if errors.IsCode(err, errors.NotFound) {
			c.manifestGone = true
			return Manifest{}, pluginMissing()
		}
		return Manifest{}, err
	}

	var payload manifestPayload
	if err = decodeJSON(body, &payload); err != nil {
		return Manifest{}, err
	}

	manifest := Manifest{
		Version:      payload.Version,
		SEOPlugin:    payload.SEOPlugin,
		WPVersion:    payload.WPVersion,
		Site:         payload.Site,
		Capabilities: payload.Capabilities,
	}
	c.manifest = &manifest
	return manifest, nil
}

func (c *Client) Capabilities(ctx context.Context) (Capabilities, error) {
	manifest, err := c.Manifest(ctx)
	if err != nil {
		return Capabilities{}, err
	}

	return Capabilities{
		Version:   manifest.Version,
		SEOPlugin: manifest.SEOPlugin,
		WPVersion: manifest.WPVersion,
		Site:      manifest.Site,
		Names:     manifest.Capabilities,
	}, nil
}

func (c *Client) requirePlugin(ctx context.Context) error {
	_, err := c.Manifest(ctx)
	return err
}

func (c *Client) ListContent(ctx context.Context, query ContentQuery) (ContentPage, error) {
	if err := c.requirePlugin(ctx); err != nil {
		return ContentPage{}, err
	}

	_, body, err := c.do(ctx, request{
		method:    http.MethodGet,
		namespace: pluginNamespace,
		path:      "/content",
		query:     query.values(),
	})
	if err != nil {
		return ContentPage{}, err
	}

	var payload contentPagePayload
	if err = decodeJSON(body, &payload); err != nil {
		return ContentPage{}, err
	}

	items := make([]ContentItem, 0, len(payload.Items))
	for _, entry := range payload.Items {
		items = append(items, entry.contentItem())
	}
	return ContentPage{Items: items, NextCursor: payload.NextCursor}, nil
}

func (c *Client) SetSEOMeta(ctx context.Context, id int64, meta SEOMeta) (SEOResult, error) {
	if err := c.requirePlugin(ctx); err != nil {
		return SEOResult{}, err
	}
	if meta == (SEOMeta{}) {
		return SEOResult{}, errors.New(errors.Invalid, "the SEO meta update carries no fields")
	}

	body, err := encodeJSON(meta)
	if err != nil {
		return SEOResult{}, err
	}

	_, raw, err := c.do(ctx, request{
		method:      http.MethodPut,
		namespace:   pluginNamespace,
		path:        resourcePath("/seo-meta", id),
		body:        body,
		contentType: contentTypeJSON,
	})
	if err != nil {
		return SEOResult{}, err
	}

	var payload seoResultPayload
	if err = decodeJSON(raw, &payload); err != nil {
		return SEOResult{}, err
	}
	return SEOResult{Applied: payload.Applied, SEOPlugin: payload.SEOPlugin}, nil
}

func (c *Client) GetRaw(ctx context.Context, id int64) (RawContent, error) {
	if err := c.requirePlugin(ctx); err != nil {
		return RawContent{}, err
	}

	_, body, err := c.do(ctx, request{method: http.MethodGet, namespace: pluginNamespace, path: rawPath(id)})
	if err != nil {
		return RawContent{}, err
	}

	var payload rawPayload
	if err = decodeJSON(body, &payload); err != nil {
		return RawContent{}, err
	}
	if payload.ContentHash != ContentHash(payload.Content) {
		return RawContent{}, errors.New(errors.External, "the site reported a content hash that does not match the content it returned").
			WithDetail("id", payload.ID)
	}

	return RawContent{
		ID:          payload.ID,
		Type:        ItemType(payload.Type),
		Content:     payload.Content,
		ContentHash: payload.ContentHash,
	}, nil
}

func (c *Client) PutRaw(ctx context.Context, id int64, content, expectedHash string) (string, error) {
	if err := c.requirePlugin(ctx); err != nil {
		return "", err
	}

	attributes := map[string]any{"content": content}
	if expectedHash != "" {
		attributes["expectedHash"] = expectedHash
	}

	body, err := encodeJSON(attributes)
	if err != nil {
		return "", err
	}

	_, raw, err := c.do(ctx, request{
		method:      http.MethodPut,
		namespace:   pluginNamespace,
		path:        rawPath(id),
		body:        body,
		contentType: contentTypeJSON,
	})
	if err != nil {
		return "", err
	}

	var payload struct {
		ContentHash string `json:"contentHash"`
	}
	if err = decodeJSON(raw, &payload); err != nil {
		return "", err
	}
	return payload.ContentHash, nil
}

func rawPath(id int64) string {
	return resourcePath("/content", id) + "/raw"
}
```

- [x] **Step 5: Run the test and watch it pass**

Run: `go test -count=1 -race ./internal/adapters/wp/...`
Expected: PASS.

- [x] **Step 6: Close the contract loop**

Add to `internal/adapters/wp/contract_test.go` the assertion that the client and the document agree on the route set:

```go
func TestTheClientHitsOnlyDocumentedPluginRoutes(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein", Content: "<p>x</p>"})
	client := newClient(t, server)

	if _, err := client.Capabilities(t.Context()); err != nil {
		t.Fatalf("Capabilities: %v", err)
	}
	if _, err := client.ListContent(t.Context(), wp.ContentQuery{}); err != nil {
		t.Fatalf("ListContent: %v", err)
	}
	if _, err := client.SetSEOMeta(t.Context(), seeded[0].ID, wp.SEOMeta{Title: "x"}); err != nil {
		t.Fatalf("SetSEOMeta: %v", err)
	}
	raw, err := client.GetRaw(t.Context(), seeded[0].ID)
	if err != nil {
		t.Fatalf("GetRaw: %v", err)
	}
	if _, err = client.PutRaw(t.Context(), seeded[0].ID, "<p>y</p>", raw.ContentHash); err != nil {
		t.Fatalf("PutRaw: %v", err)
	}

	documented := documentedRoutes(t)
	id := strconv.FormatInt(seeded[0].ID, 10)
	replacer := strings.NewReplacer("/"+id+"/", "/{id}/", "/"+id, "/{id}")

	for _, recorded := range server.Requests() {
		route, found := strings.CutPrefix(recorded.Path, "/wp-json/postulator/v1")
		if !found {
			continue
		}
		route = replacer.Replace(route)
		if _, ok := documented[route]; !ok {
			t.Errorf("the client called %s, which the contract does not document", route)
		}
	}
}
```

with `"strconv"`, `"github.com/davidmovas/postulator/internal/adapters/wp"` and `"github.com/davidmovas/postulator/internal/adapters/wp/wptest"` added to that file's imports.

- [x] **Step 7: Run the gate and commit**

```bash
go build ./... && go vet ./... && "$(go env GOPATH)/bin/golangci-lint.exe" run && go test -count=1 -race ./...
git add internal/adapters/wp
git commit -m "feat(wp): companion plugin client with cached capabilities

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---
### Task 13: Proving the password never reaches a log, then coverage and docs

**Files:**
- Test: `internal/adapters/wp/log_test.go` (package `wp_test`)
- Modify: `docs/STATUS.md`, `CLAUDE.md`

**Interfaces:**
- Consumes: everything above.
- Produces: no new production code. The deliverable is the guarantee and the handoff record.

- [x] **Step 1: Write the failing test**

Create `internal/adapters/wp/log_test.go`:

```go
package wp_test

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
)

type recordingCore struct {
	zapcore.LevelEnabler
	mu      *sync.Mutex
	entries *[]string
}

func newRecordingCore() (zapcore.Core, func() []string) {
	var guard sync.Mutex
	entries := make([]string, 0)

	core := &recordingCore{LevelEnabler: zapcore.DebugLevel, mu: &guard, entries: &entries}
	return core, func() []string {
		guard.Lock()
		defer guard.Unlock()
		return slices.Clone(entries)
	}
}

func (c *recordingCore) With([]zapcore.Field) zapcore.Core {
	return c
}

func (c *recordingCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return checked.AddCore(entry, c)
	}
	return checked
}

func (c *recordingCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	encoder := zapcore.NewMapObjectEncoder()
	for _, field := range fields {
		field.AddTo(encoder)
	}

	var builder strings.Builder
	builder.WriteString(entry.Message)
	for _, key := range slices.Sorted(maps.Keys(encoder.Fields)) {
		builder.WriteString(" ")
		builder.WriteString(key)
		builder.WriteString("=")
		builder.WriteString(fmt.Sprint(encoder.Fields[key]))
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	*c.entries = append(*c.entries, builder.String())
	return nil
}

func (c *recordingCore) Sync() error {
	return nil
}

func TestTheApplicationPasswordNeverReachesALogLine(t *testing.T) {
	t.Parallel()

	const (
		user     = "editor"
		password = "s3cret app pass"
		body     = "<p>Koffein ist ein Alkaloid.</p>"
	)

	server := wptest.New(t, wptest.WithCredentials(user, password))
	seeded := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein", Content: body})

	core, recorded := newRecordingCore()
	client, err := wp.New(wp.Config{
		BaseURL:       server.URL(),
		Username:      user,
		AppPassword:   password,
		AllowInsecure: true,
	},
		wp.WithRateLimit(0),
		wp.WithBackoff(func(int) time.Duration { return 0 }),
		wp.WithLogger(zap.New(core)),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err = client.Probe(t.Context()); err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if _, err = client.GetItem(t.Context(), wp.TypePage, seeded[0].ID); err != nil {
		t.Fatalf("GetItem: %v", err)
	}

	server.FailNext(http.StatusForbidden, 1)
	if _, err = client.GetItem(t.Context(), wp.TypePage, seeded[0].ID); err == nil {
		t.Fatal("the injected fault must reach the caller")
	}

	lines := recorded()
	if len(lines) != 3 {
		t.Fatalf("the adapter logged %d lines, want one per request", len(lines))
	}

	secrets := []string{
		password,
		base64.StdEncoding.EncodeToString([]byte(user + ":" + password)),
		"Basic ",
		body,
	}
	for _, line := range lines {
		for _, secret := range secrets {
			if strings.Contains(line, secret) {
				t.Errorf("a log line leaked %q: %s", secret, line)
			}
		}
	}
}

func TestALogLineCarriesTheRequestShapeAndTheWordPressCode(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	core, recorded := newRecordingCore()

	client, err := wp.New(wp.Config{
		BaseURL:       server.URL(),
		Username:      wptest.DefaultUser,
		AppPassword:   wptest.DefaultPassword,
		AllowInsecure: true,
	},
		wp.WithRateLimit(0),
		wp.WithRetries(0),
		wp.WithLogger(zap.New(core)),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err = client.Probe(t.Context()); err != nil {
		t.Fatalf("Probe: %v", err)
	}

	server.FailNext(http.StatusForbidden, 1)
	if _, err = client.GetItem(t.Context(), wp.TypePage, 1); err == nil {
		t.Fatal("the injected fault must reach the caller")
	}

	lines := recorded()
	if len(lines) != 2 {
		t.Fatalf("the adapter logged %d lines, want 2", len(lines))
	}

	for _, wanted := range []string{"method=GET", "path=/wp-json", "status=200", "durationMs="} {
		if !strings.Contains(lines[0], wanted) {
			t.Errorf("the success line %q is missing %q", lines[0], wanted)
		}
	}
	for _, wanted := range []string{"status=403", "code=UNAUTHORIZED", "wpCode=internal_server_error"} {
		if !strings.Contains(lines[1], wanted) {
			t.Errorf("the failure line %q is missing %q", lines[1], wanted)
		}
	}
}
```

with `"maps"` in the import block.

- [x] **Step 2: Run the tests and watch them fail or pass**

Run: `go test -count=1 -race ./internal/adapters/wp/ -run TestTheApplicationPassword -v`
Expected: these tests describe behaviour the pipeline already has, so they may pass on the first run. That is the point of writing them last: they are a regression fence around a property, not a driver for new code. If either fails, the leak is real and `logAttempt` is what has to change — never the test.

- [x] **Step 3: Measure the coverage of the two packages**

```bash
go test -count=1 -race -covermode=atomic -coverprofile=wp.out ./internal/adapters/wp/...
go tool cover -func=wp.out | tail -1
```
Expected: `total:` at or above **85.0%**. If it is below, the gap is real behaviour that no test drives; add the missing case rather than lowering the bar. The branches most likely to be missing are the option validation paths in `transport.go`, the `default` arms of `decodeItems`/`decodeItem`, and `classify`'s unexpected-status arm.

- [x] **Step 4: Run the whole gate**

```bash
go build ./...
go vet ./...
"$(go env GOPATH)/bin/golangci-lint.exe" run
go test -count=1 -race -covermode=atomic -coverprofile=coverage.out ./...
go run ./cmd/covergate -profile coverage.out
gofmt -l .
rm wp.out coverage.out
```
Expected: build and vet silent, lint reports 0 issues, every test passes under `-race`, covergate passes, `gofmt -l` prints nothing. `task build` is deliberately not run: this track touched no file under `frontend/`.

- [x] **Step 5: Record the phase in the handoff documents**

Append to `docs/STATUS.md`, directly after the Phase 1B paragraph in **Where we are**:

```markdown
**Phase 3A (the WordPress Go adapter) is complete** on the `phase-3a` worktree. The plan
is `docs/superpowers/plans/2026-09-18-phase-3a-wp-adapter.md`; its thirteen tasks landed
one commit each. `wp-plugin/openapi.yaml` freezes the `postulator/v1` contract that track
B implements in PHP; `internal/adapters/wp` holds the stdlib client for core REST,
WooCommerce and the plugin namespace, with proxy, retry, rate limiting and error mapping;
`internal/adapters/wp/wptest` is the in-memory fake that Phases 6, 7 and 12 test against.
The sync use case is not in this track and follows once Phase 2 lands.
```

Add a **Decisions taken in Phase 3A** section after the Phase 1B decisions:

```markdown
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
- **Path normalisation is one algorithm with a temporary home.** Strip scheme and host
  after deciding internal-ness by exact lowercase host equality, strip query and
  fragment, never percent-decode, collapse duplicate slashes, force exactly one leading
  and one trailing slash, ASCII-lowercase the whole path, and make no exception for a
  file extension. `wp.NormalizePath` holds it for now; the canonical home is
  `internal/domain/pagemap.NormalizePath` from Phase 2, and a follow-up task replaces
  the adapter's body with a call into the domain once Phase 2 merges.
```

Append to the **Known gaps** section:

```markdown
- Phase 3A ships no sync use case: `internal/application` gains its WordPress consumer
  only after Phase 2 lands the page map and the repositories. Nothing in `internal/app`
  constructs a `wp.Client` yet, so `wp.timeout`, `wp.retries`, `wp.rateLimitPerSecond`
  and `wp.proxyUrl` are declared and validated but not yet read at startup.
- `wp.NormalizePath` duplicates what `internal/domain/pagemap.NormalizePath` will own
  once Phase 2 merges. A follow-up task then replaces the adapter's body with a call into
  the domain — adapters may import domain — and moves the table test with it.
  `wp.InternalPath` stays in the adapter, because deciding whether a host is this site's
  host is adapter knowledge.
```

Add to the **Footguns** section of `CLAUDE.md`:

```markdown
- **WordPress `modified_gmt` has no timezone suffix**, so `time.Parse(time.RFC3339, ...)`
  fails on it; `wp.parseWPTime` falls back to `2006-01-02T15:04:05` read as UTC. The
  reverse is worse: core REST filters `modified_after` on the site-local `post_modified`
  while returning `modified_gmt`, so an incremental core-REST sync of a non-UTC site can
  miss or repeat items inside the offset. The plugin's `/content?since=` compares
  `post_modified_gmt` and is the accurate path; that is one of the reasons the plugin
  exists.
- **WordPress sends `"meta": []`, not `{}`,** for a post type with no registered meta, so
  decoding it straight into a map fails. `wp.metaBag` accepts the array, the object and
  null.
- **`wptest` must never import `wp`.** The fake is a second, independent implementation of
  the WordPress contract; sharing the client's types or its hash would let one bug satisfy
  both sides of every assertion.
```

Add to the **Standing rulings** list of `CLAUDE.md`:

```markdown
- **2026-09-18 (phase 3A)** — `adapters/wp` is the one place in the codebase where page
  numbers are legal. WordPress core REST has no keyset; `kernel/paging` is untouched by
  it, and the plugin's own cursor is passed through opaque.
- **2026-09-18 (phase 3A)** — `wp.ContentHash` is `hex(sha256(raw))` and is frozen. It is
  not `domain/content.Document.Hash()`, and normalising it would silently break the
  optimistic-concurrency check on `PUT /content/{id}/raw`.
- **2026-09-18 (phase 3A)** — `wp-plugin/openapi.yaml` is the contract between the two
  Phase 3 tracks. Neither track changes it alone.
```

- [x] **Step 6: Commit**

```bash
git add internal/adapters/wp docs/STATUS.md CLAUDE.md
git commit -m "test(wp): assert no credential reaches a log line

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

## Verification

The phase is done when every line below is green from a clean tree on the `phase-3a` worktree.

```bash
go build ./...
go vet ./...
"$(go env GOPATH)/bin/golangci-lint.exe" run
go test -count=1 -race -covermode=atomic -coverprofile=coverage.out ./...
go run ./cmd/covergate -profile coverage.out
go test -count=1 -race -covermode=atomic -coverprofile=wp.out ./internal/adapters/wp/...
go tool cover -func=wp.out | tail -1
gofmt -l .
git status --porcelain
```

- `golangci-lint` reports **0 issues**. The binary must be the one under `$(go env GOPATH)/bin`; the scoop shim earlier on `PATH` is built with go1.26 and refuses a `go 1.27` module outright.
- `covergate` passes its module gate; `go tool cover` reports **≥ 85.0%** for `internal/adapters/wp/...`.
- `gofmt -l .` and `git status --porcelain` both print nothing — the repository normalises to LF through `.gitattributes`, and a CRLF file shows up here rather than in CI.
- Thirteen commits, one per task, each conventional and each ending with the attribution trailer.
- `internal/app/deps_test.go` still passes: this track adds no `domain` or `application` dependency and nothing new imports `internal/transport`.
- `task build` is **not** part of this gate. No file under `frontend/` is touched, and running it would regenerate bindings this track has no business changing.

## What this track deliberately leaves out

- **The sync use case.** It belongs to `internal/application` and needs Phase 2's page map and repositories; it lands after Phase 2 (roadmap Phase 3, then Phase 7's `sync_back` step).
- **Wiring into `internal/app`.** Nothing constructs a `wp.Client` yet, because nothing has a site row to construct one from. `wp.FromSettings` is the seam the composition root will use, and the caller will fetch the application password through the `SecretStore` port and hand it over in `Config.AppPassword`.
- **Per-site overrides of timeout, retries, rate limit and proxy.** Site defaults live in `domain/site` and arrive as extra `Option`s or `Config` fields later.
- **The PHP plugin and the docker end-to-end compose.** That is track B, working against `wp-plugin/openapi.yaml`.

## Deviations

Every place the delivered code differs from the tasks above, recorded so a reader of the
plan is never surprised by the tree.

- **`userAgent` and `maxBodyBytes` live in `do.go`, not `client.go`.** Their only users
  arrive with the request pipeline, and a per-task green lint forbids a constant whose
  consumer is a later task.
- **The manifest cache fields (`manifestMu`, `manifest`, `manifestGone`) are added to
  `Client` in task 12**, not task 2: a struct field cannot name a type that does not
  exist yet.
- **`wptest.Server.uploadOrder` is declared in task 6** for the same reason.
- **`Manifest(ctx)` is a public method beside `Capabilities(ctx)`**, and it is the one
  that owns the mutex and the cache; `Capabilities` projects it. `requirePlugin` is the
  unexported guard every other plugin method calls first.
- **`encodeJSON` and `contentTypeJSON` were added to `do.go`** in task 8 as the one
  encoder every write path shares, rather than each method marshalling for itself.
- **`ListQuery.termValues()` was added in task 9.** Core categories are terms: they have
  no `status` and no `modified_after`, so sending the item query at them would put
  parameters on the wire that WordPress ignores at best.
- **Lint shaped several signatures.** Named results on multi-return helpers
  (`route`, `listWindow`, `redirectTarget`, and the test helpers), `http.NoBody` instead
  of a nil body, index-based ranges over payload slices, `err :=` inside `if`, and the
  conversions `Category(p)`, `ProductCategory(p)`, `ProductCategoryRef(ref)`,
  `Manifest(payload)` and `ContentLink(link)` where staticcheck proved the wire struct and
  the public struct identical.
- **`TestRetryAfterIsWaitedForRatherThanTheBackoff` injects 30s against a 2s context**
  (the task wrote 5s against 300ms): under `-race` with the whole suite in parallel the
  first request occasionally missed the shorter deadline.
- **The fake-server link fixture is `/Koffein//`, not `//koffein//`** (commit `06a2fe0`).
  `url.Parse` reads a leading `//` as a network-path reference, so the original fixture
  tested host comparison rather than the slash collapsing it was written for. The
  replacement exercises collapsing and ASCII lowercasing together.
- **Task 13 added table tests for the fake's error paths** — bad ids, broken bodies,
  unreadable cursors, missing uploads — to lift `wptest` from 84.7% to 96.3%.

### Review fixes, 2026-09-18

- **`Config` and `*Client` carry redacting `String()`/`GoString()`.** `%v`, `%+v`, `%#v`,
  `%s` and `fmt.Sprint` of either printed the application password verbatim, so any
  debug line or wrapped error could leak it.
- **A write that reached WordPress is never repeated.** `retryAllowed` replaces the bare
  `retryable` check in `do`: `GET` and `HEAD` retry as before, while `POST`, `PUT`,
  `PATCH` and `DELETE` retry only when no response arrived at all (`status == 0`).
  `wptest.FailAfterNext` injects a failure *after* the handler has already changed the
  state, which is what makes the duplicate-write case testable.
- **A 5xx honours `Retry-After`** (delta-seconds and HTTP-date) instead of always
  deferring to our backoff.
- **Every unmapped 4xx is `Invalid`** with the WordPress code in `Details` and is never
  retried, so 405, 410, 413 and 422 stop being treated as transient.
- **The contract test pins property names**, not just words, so renaming a field in
  `openapi.yaml` fails the build.
- **`RawContent.type` no longer offers `product_cat`**, matching the post-only routes.
