# Phase 3B — WordPress Companion Plugin and Docker End-to-End Harness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. This repository forbids sub-agents (spec §10), so the subagent-driven variant does not apply.

**Goal:** Ship `wp-plugin/postulator-companion`, the PHP companion plugin that exposes `/wp-json/postulator/v1` exactly as the Go WordPress adapter (track A) expects, a deterministic packaging target for it, a pinned docker WordPress stack that provisions itself, and a build-tagged Go end-to-end suite that proves the contract against a real WordPress.

**Architecture:** One plugin file plus six small `includes/` modules in the `Postulator\Companion` namespace, all plain namespaced functions and no classes. Reads go through `$wpdb` with an explicit keyset predicate so ordering and cursors are ours, not `WP_Query`'s; hydration goes through `get_post()`/`get_term()` so every WordPress filter a site depends on still runs. Writes go through `wp_update_post` and `update_post_meta` with `wp_slash`, with no filter surgery: the authenticated application-password user is an administrator and therefore holds `unfiltered_html`, so WordPress removes the kses filters itself at `set_current_user`. Correctness is proven only by the docker suite: `internal/adapters/wp/e2e` carries a tiny self-contained HTTP helper behind `//go:build e2e`, so it compiles out of every normal build and depends on nothing from track A.

**Tech Stack:** PHP 8.3 (plugin targets ≥ 8.1), WordPress 6.9.2 (plugin targets ≥ 6.4), MariaDB 11.4.12, WP-CLI 2.12.0, Docker Compose v5, Go 1.27, Task `v3.53.1`, golangci-lint `v2.13.2`.

**Spec:** `docs/superpowers/specs/2026-09-17-postulator-v2-design.md` (§9.3 WordPress adapter and plugin namespace, §10 documentation policy, §15 reference map) and the Phase 3 rows of `docs/superpowers/plans/2026-09-17-postulator-v2-roadmap.md` (§11 deliverables, §13 verification: "e2e tag against docker WP with the plugin installed: create page, write SEO meta, bulk content endpoint returns links and hashes").

## Global Constraints

- **No comments in Go, TypeScript, YAML or PHP** — not inline, not godoc, not package docs. The WordPress plugin header block in `postulator-companion.php` is the single allowed exception and stays minimal (five lines).
- **No stubs, no TODOs, no placeholders.** If something cannot be done, stop and say so.
- **Interfaces are declared by the consumer**, never by the implementation and never in advance.
- **TDD.** Failing test first, run it, implement, run it, commit. Table tests with named cases. Gates: `internal/domain` + `internal/application` ≥ 80%, module ≥ 70%, `internal/kernel` ≥ 90%.
- **Cursor pagination only.** No offset, anywhere — including inside the PHP plugin, which keysets on `(post_modified_gmt, ID)` and never uses `LIMIT x OFFSET y`.
- **JSON is camelCase**, timestamps are RFC3339 UTC, ids are UUID v4 lowercase text. The plugin's ids are WordPress integers, which is what the contract freezes.
- **Errors crossing a boundary are `*kernel/errors.Error`** on the Go side; the plugin's boundary form is `{"code","message"}` with an HTTP status from `400 401 403 404 409 500`.
- **Secrets never reach a log, an artifact or a plain column.** The generated application password lives only in `docker/e2e/.env.generated`, which is gitignored, and is never printed by a task or a test.
- **No new `.md` files** beyond `CLAUDE.md` and the set in spec §10. This plan is `docs/superpowers/plans/2026-MM-DD-phase-N-<name>.md`, which §10 allows.
- **No external PHP dependencies.** No composer, no vendor directory, no PHP test framework. `php -l` is the only PHP tooling.
- **Commits** are conventional (`<type>(<scope>): <subject>`), on the `phase-3b` worktree branch, never amended, rebased or force-pushed, and every commit ends with the trailer `Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>`.
- Pinned: `wordpress:6.9.2-php8.3-apache`, `wordpress:cli-2.12.0-php8.3`, `mariadb:11.4.12`, Go `1.27`, golangci-lint `v2.13.2`, Task `v3.53.1`.

## Environment findings

Verified on the development machine on 2026-09-18 before this plan was written; every one of these is load-bearing for a task below.

| Probe | Result |
|---|---|
| `docker version` | client 29.7.2 (windows/amd64), server Docker Desktop 4.86.0, engine 29.7.2 (linux/amd64), containerd v2.2.5 |
| `docker compose version` | v5.3.1 |
| Newest `wordpress:6.9.x-php8.3-apache` | `6.9.2`; `6.9.3` and `6.10` do not exist yet |
| Newest `wordpress:cli-*-php8.3` | `2.12.0`; `2.12.1` and `2.13` do not exist |
| `mariadb` 11.4 LTS patch | `11.4.12` exists, `11.4.13` does not |
| `wordpress:6.9.2-php8.3-apache` PHP | 8.3.30, `allow_url_fopen=on`, `DOMDocument` present |
| Apache in that image | `conf-enabled/docker-php.conf` sets `AllowOverride All` on `/var/www/`, `mods-enabled/rewrite.load` is present → pretty permalinks need no extra vhost configuration |
| `www-data` uid in the WordPress image | **33** (Debian) |
| `www-data` uid in the WP-CLI image | **82** (Alpine) → the CLI service must run as `user: "33:33"` or it cannot write `.htaccess` or install plugins into the shared volume |
| `mariadb:11.4.12` | ships `/usr/local/bin/healthcheck.sh`, so `--connect --innodb_initialized` is a valid healthcheck |
| `wp user application-password create <user> <name> --porcelain` | exists in WP-CLI 2.12.0 and prints only the password |
| A package whose only files carry `//go:build e2e` | `go build ./...`, `go vet ./...`, `go test ./...` and `golangci-lint run` all pass and report nothing; `go test -tags e2e ./...` runs it |
| `golangci-lint run --build-tags e2e` | supported flag on v2.13.2 |

The last two mean the e2e suite costs the normal gate nothing: it contributes no statements to `coverage.out`, so `cmd/covergate` is unaffected, and it is linted on demand through a separate task.

---

## Design decisions

### D1 — kses versus `unfiltered_html` for raw writes

`remove_filter('content_save_pre', 'wp_filter_post_kses')` is forbidden, and it is also unnecessary. `kses_init()` is hooked to both `init` and `set_current_user`, and it installs the kses filters only when `! current_user_can('unfiltered_html')`. Application-password authentication resolves on `determine_current_user`, which fires the first time `wp_get_current_user()` is reached — after `init` — and `wp_set_current_user()` then fires `set_current_user`, which re-runs `kses_init()` against the now-authenticated user. On single-site WordPress the `administrator` and `editor` roles hold `unfiltered_html`, so by the time a REST callback runs the kses filters have already been removed by core.

**The plugin therefore performs no filter surgery of any kind.** It relies on the authenticated user holding `unfiltered_html` and makes the outcome observable instead of assuming it: after `wp_update_post` it re-reads the stored `post_content` and returns `contentHash` computed from **what is now stored**, not from what was sent. A caller that compares its own `sha256(sent)` with the returned hash learns immediately whether anything altered the content. The e2e suite is exactly that caller and asserts byte equality over content containing `<a href>`, `<h2>`, `<ul><li>` and `<img>`.

The fallback path is documented rather than coded around: if a site authenticates as a role without `unfiltered_html`, the content passes through kses's default `post` allowed-tag set, which already permits `a[href|rel|title]`, `h1`–`h6`, `ul`/`ol`/`li` and `img[src|alt|width|height|class|srcset|sizes|loading]` — every shape Postulator generates. What kses strips (`<script>`, `<style>`, `<iframe>`, `on*` handlers) Postulator never emits. Such a site sees a returned hash that differs from the sent hash, which is a fact the Go adapter can act on, not a silent corruption.

The related footgun is the opposite one and is easy to get wrong: `wp_insert_post()` calls `wp_unslash()` on everything it is handed, and `update_metadata()` does the same. Every write in this plugin therefore passes `wp_slash( $value )`. Without it, a single backslash or an escaped quote in generated HTML is eaten on every save.

### D2 — `path` and `links` normalisation

The plugin and the Go adapter must produce byte-identical paths or every link-compliance number Postulator computes is wrong. One function, `normalize_path()`, is the only place a path is produced, and both `get_permalink()`-derived paths and `<a href>`-derived paths go through it:

1. Collapse every run of `/` into a single `/` (`preg_replace('#/+#', '/', $path)`).
2. Ensure a leading `/`.
3. Ensure a trailing `/`, **unconditionally** — no file-extension exception. `/x.png` becomes `/x.png/`. The rule has no branches precisely so the two implementations cannot drift; the alternative, "trailing slash unless the last segment contains a dot", is two heuristics that must agree forever.
4. The empty path is `/`.

`internal_href_to_path()` decides membership before normalising:

- An `href` that is empty, or starts with `#`, is skipped.
- A scheme other than `http`/`https` (`mailto:`, `tel:`, `javascript:`) is skipped.
- A host is compared **lowercased on both sides** against `strtolower(wp_parse_url(home_url(), PHP_URL_HOST))`. Exact equality only: `www.` is not stripped, because stripping it on one side and not the other is the classic way these two implementations diverge.
- Query and fragment are discarded before step 1.
- Percent-encoding is left exactly as `get_permalink()` produced it; nothing is decoded, so a non-ASCII slug is one byte-sequence on both sides.
- A protocol-relative `//host/path` is parsed normally and its host compared like any other.
- A root-relative `/path` is used as-is.
- A genuinely relative `child/` is resolved against the **directory of the containing post's own path** — `/koffein/powder/` yields the base `/koffein/`.

`links` is extracted from **raw `post_content`**, not from rendered content, because the contract says so and because rendered content contains theme and plugin chrome that is not the page's own link graph. `anchor` is the `<a>` element's `textContent` with every whitespace run collapsed to one space and the result trimmed.

### D3 — Cursor encoding

The cursor is opaque base64url-without-padding of a JSON object, produced by `encode_cursor()` and consumed by `decode_cursor()`, which translate `+/` to `-_` and strip or restore `=` so the value survives a query string untouched.

The listing has two phases, because a WordPress taxonomy term is not a post and cannot share a keyset with one. Posts come first, terms second, and the cursor names its phase:

- post phase — `{"p":"post","m":"2026-09-18 10:00:00","i":123}` where `m` is `post_modified_gmt` in MySQL GMT form. The predicate is `post_modified_gmt > m OR (post_modified_gmt = m AND ID > i)` under `ORDER BY post_modified_gmt ASC, ID ASC`.
- term phase — `{"p":"term","i":45}`. The predicate is `t.term_id > i` under `ORDER BY t.term_id ASC`.

`collect()` fills from the current phase and, when that phase is exhausted before `limit` is reached, advances to the next phase and keeps filling, so a page is short only when the whole listing is. An empty `nextCursor` string means the listing is finished. A `cursor` parameter that was supplied but does not decode, or whose `p` is neither phase, is a `400`, never a silent restart — the same ruling `kernel/paging` already took for replayed cursors.

A term has no modification time in WordPress, so the plugin maintains `_postulator_modified` term meta (MySQL GMT form, same format as `post_modified_gmt`) from the `created_term` and `edited_term` actions. Terms that predate the plugin have no such meta, so the query uses `LEFT JOIN` plus `COALESCE(tm.meta_value, <installed_at>)` — the activation timestamp stored once in the `postulator_companion_installed_at` option — for both the `since` filter and the emitted `modified`. That keeps `since` meaningful without a read path that writes and without an activation backfill that can race WooCommerce's taxonomy registration.

### D4 — `h1` extraction

`first_h1()` runs over **rendered** content: `apply_filters('the_content', $post->post_content)` inside a `setup_postdata()` / `wp_reset_postdata()` pair with `$GLOBALS['post']` saved and restored, so shortcodes and blocks resolve the way a visitor sees them rather than the way a bare filter call would.

It then loads the result with `DOMDocument::loadHTML()` prefixed by `<?xml encoding="UTF-8">`, under `libxml_use_internal_errors(true)` with the previous state restored and `libxml_clear_errors()` called, and takes `getElementsByTagName('h1')->item(0)->textContent`, whitespace-collapsed and trimmed. Malformed markup is the normal case for hand-edited WordPress content and must not surface as a warning in a REST response.

The fallback is a regex, used only when `DOMDocument` is genuinely absent (a PHP build without `ext-dom`): `#<h1\b[^>]*>(.*?)</h1>#is`, then `wp_strip_all_tags`, `html_entity_decode` and the same collapse. Absent an `<h1>`, the value is `""`, never null.

The same `load_dom()` helper backs `extract_links()`, so both features share one parser and one error posture.

### D5 — No-SEO-plugin head rendering without duplicate tags

When `detect_plugin()` returns `none`, the plugin renders head tags itself. Each of the three tags core also knows about is produced by **replacing** core's value rather than by printing a second tag:

- `<title>` — `pre_get_document_title`. A non-empty return short-circuits `wp_get_document_title()`, so `_wp_render_title_tag()` prints exactly one `<title>` and it is ours. Every theme since WordPress 4.1 declares `title-tag` support; a theme that hardcodes its own `<title>` instead is out of scope and is named as such in `docs/ARCHITECTURE.md`.
- `<link rel="canonical">` — `get_canonical_url`, which is the filter core's own `rel_canonical()` reads through `wp_get_canonical_url()`. Filtering it cannot produce two canonicals; printing one on `wp_head` would. An empty stored value returns core's canonical unchanged.
- `<meta name="description">` and the OG pair — core has no equivalent, so these are printed on `wp_head` at priority 2, guarded by `is_singular()`, by a `static $printed` flag that makes a second `wp_head()` call a no-op, and by a non-empty check per tag so an empty value never emits an empty attribute. A theme that prints its own description meta would duplicate it; Yoast and Rank Math detection removes the common case, and the residue is documented rather than papered over with output buffering.

The whole head module is registered only when `detect_plugin() === 'none'`, so activating Yoast or Rank Math silences it in one branch.

### D6 — Error shapes

Ordinary failures are `WP_Error` with a `status` in `data`, which the REST server serialises as `{"code","message","data":{"status":N}}` — a superset of the frozen `{"code","message"}`, and the two fields the Go side reads are top-level.

The one exception is the hash conflict. `WP_Error` can only carry extra fields nested under `data`, and the contract puts `currentHash` at the top level next to `code` and `message`. `PUT /content/{id}/raw` therefore returns an explicit `WP_REST_Response` with status `409` and exactly `{"code":"hash_mismatch","message":"…","currentHash":"…"}`.

Permission failures use `rest_authorization_required_code()`, which is `401` for an unauthenticated request and `403` for an authenticated one without the capability — the distinction the contract's status list requires.

### D7 — Deterministic packaging

`task plugin:zip` runs `go run ./cmd/pluginzip`, a ~90-line Go program, rather than `zip` or PHP's `ZipArchive`. Neither of those exists reliably on a Windows developer machine, and neither is byte-deterministic without argument archaeology. The Go writer sorts entry names, writes forward-slash paths under a single `postulator-companion/` root (WordPress requires the plugin directory inside the archive), stamps every entry with the ZIP epoch `1980-01-01T00:00:00Z` and mode `0644`, emits no directory entries, and deflates. Two runs over the same tree are byte-identical, which a Go unit test asserts directly. `.gitattributes` already pins the working tree to LF, so the archive is identical on Windows and Linux.

It is a Go program, so it is covered by the existing gate rather than being an untested shell fragment, and `cmd/covergate` already sets the precedent for a small `cmd/` tool with real tests.

### D8 — `php -l` without a PHP installation

`task plugin:lint` dispatches on platform using Task's per-command `platforms:` key rather than shell branching, which would have to work under both `mvdan/sh` on Windows and `bash` on CI:

- `platforms: [windows]` — `docker compose -f docker/e2e/compose.yaml run --rm --no-deps -T phplint sh -c 'find /plugin -name "*.php" -print0 | xargs -0 -n1 php -l'`. The `phplint` service sits behind a compose profile, has no `depends_on`, and bind-mounts `wp-plugin` read-only, so it starts no database and reuses the pinned `wordpress:6.9.2-php8.3-apache` image the stack already pulls. The loop runs inside the container, so the host needs no `find` or `xargs`.
- `platforms: [linux, darwin]` — the same `find … | xargs -0 -n1 php -l` against a local `php`.

CI matters here: GitHub's `windows-latest` runners cannot run Linux containers, so the plugin work gets its own small `ubuntu-latest` job. Ubuntu runners ship PHP 8.3, so that job needs no third-party setup action — it runs `php -v` as an explicit fail-loudly step, then `task plugin:lint` and `task plugin:zip`. The docker end-to-end suite is never run in CI.

### D9 — The e2e suite owns its HTTP client

Track A may not have landed, so `internal/adapters/wp/e2e` imports nothing from `internal/adapters/wp`. It carries a ~70-line `client` over `net/http` with basic auth, a 30-second timeout and JSON encode/decode helpers. Every file in the package is a `_test.go` file behind `//go:build e2e`, so the package ships no production code, contributes nothing to `coverage.out`, and cannot be imported by anything. When track A merges, switching these tests to the real adapter is a follow-up that deletes the helper.

The suite reads `docker/e2e/.env.generated` — located by walking up from the package directory to the nearest `go.mod`, overridable with `POSTULATOR_E2E_ENV` — and **fails** rather than skips when the file is absent, with a message naming `task e2e:up`. A gate that silently skips is a gate that silently passes; the build tag is already the opt-in.

### D10 — Scope of the write endpoints

`PUT /seo-meta/{id}` and both `/content/{id}/raw` routes address **posts only**. WordPress post ids and term ids occupy separate sequences and collide freely, so a single numeric route cannot address both without a disambiguating parameter the contract does not have. A term id therefore resolves to `404`. Term SEO meta is still **read** in `/content` — from `rank_math_*` term meta under Rank Math, from the serialised `wpseo_taxonomy_meta` option under Yoast (which is where Yoast keeps term SEO, not in term meta), and from `_postulator_*` term meta otherwise. This is listed as an open question for the orchestrator because track A's `wp-plugin/openapi.yaml` must agree.

---

## File structure

**Create**

- `wp-plugin/postulator-companion/postulator-companion.php` — plugin header, constants, requires, hook registration, activation hook.
- `wp-plugin/postulator-companion/includes/http.php` — `permission_check`, `error_response`, `not_found`, `forbidden`, `invalid`.
- `wp-plugin/postulator-companion/includes/normalize.php` — `site_host`, `normalize_path`, `url_to_path`, `parent_path`, `internal_href_to_path`, `collapse_text`.
- `wp-plugin/postulator-companion/includes/seo.php` — `detect_plugin`, `META_KEYS`, `read_post_seo`, `read_term_seo`, `write_post_seo`.
- `wp-plugin/postulator-companion/includes/head.php` — `boot_head`, `filter_document_title`, `filter_canonical_url`, `print_head_tags`.
- `wp-plugin/postulator-companion/includes/content.php` — `content_hash`, `rfc3339`, `load_dom`, `first_h1`, `extract_links`, `rendered_content`, `encode_cursor`, `decode_cursor`, `query_posts_page`, `query_terms_page`, `post_item`, `term_item`, `collect`, `touch_term`.
- `wp-plugin/postulator-companion/includes/routes.php` — `register_routes` and the five callbacks, `parse_types`, `parse_since`, `clamp_limit`.
- `docker/e2e/compose.yaml` — `db`, `wordpress`, `bootstrap`, `phplint`.
- `docker/e2e/bootstrap.sh` — WP-CLI provisioning, credential generation.
- `cmd/pluginzip/main.go`, `cmd/pluginzip/pack.go`, `cmd/pluginzip/pack_test.go`.
- `internal/adapters/wp/e2e/harness_test.go` — env loading, `client`, fixtures, assertions helpers.
- `internal/adapters/wp/e2e/probe_test.go`, `manifest_test.go`, `content_test.go`, `commerce_test.go`, `seometa_test.go`, `raw_test.go`.

**Modify**

- `Taskfile.yml` — `plugin:lint`, `plugin:zip`, `e2e:up`, `e2e:down`, `e2e:reset`, `e2e:test`, `lint:e2e`.
- `.gitignore` — `docker/e2e/.env.generated`.
- `.github/workflows/ci.yml` — a second `plugin` job on `ubuntu-latest`.
- `docs/ARCHITECTURE.md` — a "WordPress companion plugin" section (≤ 25 lines).
- `docs/CONVENTIONS.md` — a "Docker end-to-end" section (≤ 15 lines).
- `docs/STATUS.md`, `CLAUDE.md` — phase record, decisions, standing rulings, footguns.

**Delete** — nothing.

---

## Task list

Eleven tasks, one commit each, in dependency order. Every task follows the same shape: write the failing check first, run it and read the failure, implement, run it again, commit.

1. Docker stack, credential provisioning, `e2e:up`/`down`/`reset`, and the probe test.
2. Plugin skeleton, `GET /manifest`, `task plugin:lint`.
3. `cmd/pluginzip` and `task plugin:zip`.
4. `GET /content` for pages and posts — path, links, h1, hash, keyset cursor, `since`, `types`, `limit`.
5. WooCommerce fixtures and `product` / `product_cat` in `GET /content`.
6. `PUT /seo-meta/{id}` and the no-SEO-plugin head rendering.
7. `GET` and `PUT /content/{id}/raw`, including the `409`.
8. `E2E_SEO=yoast` mode.
9. `E2E_SEO=rankmath` mode, or a documented skip.
10. CI `plugin` job and `task lint:e2e`.
11. Docs: `ARCHITECTURE.md`, `CONVENTIONS.md`, `STATUS.md`, `CLAUDE.md`.

---

### Task 1: Docker WordPress stack and e2e credentials

**Files:**
- Create: `docker/e2e/compose.yaml`, `docker/e2e/bootstrap.sh`
- Create: `internal/adapters/wp/e2e/harness_test.go`, `internal/adapters/wp/e2e/probe_test.go`
- Modify: `Taskfile.yml`, `.gitignore`

**Interfaces:**
- Produces: `docker/e2e/.env.generated` carrying `E2E_WP_URL`, `E2E_WP_USER`, `E2E_WP_APP_PASSWORD`, `E2E_SEO`, `E2E_WOO`; tasks `e2e:up`, `e2e:down`, `e2e:reset`, `e2e:test`.
- Consumes: nothing from this repository.

- [ ] **Step 1: Write the failing test**

Create `internal/adapters/wp/e2e/harness_test.go`:

```go
//go:build e2e

package e2e_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type environment struct {
	baseURL string
	user    string
	pass    string
	seo     string
	woo     bool
}

func repoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("no go.mod above %s", dir)
		}
		dir = parent
	}
}

func envPath(t *testing.T) string {
	t.Helper()

	if override := os.Getenv("POSTULATOR_E2E_ENV"); override != "" {
		return override
	}
	return filepath.Join(repoRoot(t), "docker", "e2e", ".env.generated")
}

func loadEnvironment(t *testing.T) environment {
	t.Helper()

	path := envPath(t)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s is missing; run `task e2e:up` first: %v", path, err)
	}

	values := make(map[string]string)
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			t.Fatalf("%s has a line without '=': %q", path, line)
		}
		values[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}

	env := environment{
		baseURL: values["E2E_WP_URL"],
		user:    values["E2E_WP_USER"],
		pass:    values["E2E_WP_APP_PASSWORD"],
		seo:     values["E2E_SEO"],
		woo:     values["E2E_WOO"] == "1",
	}
	for name, value := range map[string]string{
		"E2E_WP_URL":          env.baseURL,
		"E2E_WP_USER":         env.user,
		"E2E_WP_APP_PASSWORD": env.pass,
		"E2E_SEO":             env.seo,
	} {
		if value == "" {
			t.Fatalf("%s does not set %s", path, name)
		}
	}
	return env
}

type client struct {
	base string
	user string
	pass string
	http *http.Client
}

func newClient(t *testing.T) (*client, environment) {
	t.Helper()

	env := loadEnvironment(t)
	return &client{
		base: strings.TrimSuffix(env.baseURL, "/"),
		user: env.user,
		pass: env.pass,
		http: &http.Client{Timeout: 30 * time.Second},
	}, env
}

func (c *client) request(t *testing.T, method, path string, payload any) (int, []byte) {
	t.Helper()

	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("encode %s %s: %v", method, path, err)
		}
		body = bytes.NewReader(encoded)
	}

	request, err := http.NewRequest(method, c.base+path, body)
	if err != nil {
		t.Fatalf("build %s %s: %v", method, path, err)
	}
	request.SetBasicAuth(c.user, c.pass)
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := c.http.Do(request)
	if err != nil {
		t.Fatalf("call %s %s: %v", method, path, err)
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read %s %s: %v", method, path, err)
	}
	return response.StatusCode, raw
}

func (c *client) expect(t *testing.T, method, path string, payload any, status int, out any) {
	t.Helper()

	code, raw := c.request(t, method, path, payload)
	if code != status {
		t.Fatalf("%s %s: status %d, want %d, body %s", method, path, code, status, raw)
	}
	if out == nil {
		return
	}
	if err := json.Unmarshal(raw, out); err != nil {
		t.Fatalf("decode %s %s: %v, body %s", method, path, err, raw)
	}
}

func (c *client) fetchPage(t *testing.T, url string) string {
	t.Helper()

	response, err := c.http.Get(url)
	if err != nil {
		t.Fatalf("call GET %s: %v", url, err)
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read GET %s: %v", url, err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: status %d, want 200", url, response.StatusCode)
	}
	return string(raw)
}

func hashOf(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func uniqueSlug(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}
```

Create `internal/adapters/wp/e2e/probe_test.go`:

```go
//go:build e2e

package e2e_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"
)

func TestSiteAnswersAndCredentialsAuthenticate(t *testing.T) {
	c, env := newClient(t)

	var root struct {
		Name       string   `json:"name"`
		Namespaces []string `json:"namespaces"`
	}
	c.expect(t, http.MethodGet, "/wp-json/", nil, http.StatusOK, &root)

	if !slices.Contains(root.Namespaces, "postulator/v1") {
		t.Fatalf("postulator/v1 is not registered; namespaces are %v", root.Namespaces)
	}

	var me struct {
		Slug string `json:"slug"`
	}
	c.expect(t, http.MethodGet, "/wp-json/wp/v2/users/me", nil, http.StatusOK, &me)
	if !strings.EqualFold(me.Slug, env.user) {
		t.Fatalf("authenticated as %q, want %q", me.Slug, env.user)
	}
}
```

- [ ] **Step 2: Run it and read the failure**

```
go test -tags e2e -count=1 ./internal/adapters/wp/e2e/...
```

It fails on the missing `docker/e2e/.env.generated`, naming `task e2e:up`. That is the failure this task removes.

- [ ] **Step 3: Implement**

Create `docker/e2e/compose.yaml`:

```yaml
name: postulator-e2e

services:
  db:
    image: mariadb:11.4.12
    environment:
      MARIADB_ROOT_PASSWORD: postulator-root
      MARIADB_DATABASE: wordpress
      MARIADB_USER: wordpress
      MARIADB_PASSWORD: wordpress
    volumes:
      - db:/var/lib/mysql
    healthcheck:
      test: ["CMD", "healthcheck.sh", "--connect", "--innodb_initialized"]
      interval: 5s
      timeout: 5s
      retries: 30

  wordpress:
    image: wordpress:6.9.2-php8.3-apache
    depends_on:
      db:
        condition: service_healthy
    ports:
      - "127.0.0.1:8089:80"
    environment:
      WORDPRESS_DB_HOST: db
      WORDPRESS_DB_NAME: wordpress
      WORDPRESS_DB_USER: wordpress
      WORDPRESS_DB_PASSWORD: wordpress
    volumes:
      - wp:/var/www/html
      - ../../wp-plugin/postulator-companion:/var/www/html/wp-content/plugins/postulator-companion
    healthcheck:
      test: ["CMD-SHELL", "php -r 'exit(@file_get_contents(\"http://127.0.0.1/wp-login.php\") === false ? 1 : 0);'"]
      interval: 5s
      timeout: 10s
      retries: 40

  bootstrap:
    image: wordpress:cli-2.12.0-php8.3
    profiles:
      - tools
    user: "33:33"
    depends_on:
      db:
        condition: service_healthy
      wordpress:
        condition: service_healthy
    environment:
      E2E_SITE_URL: http://localhost:8089
      E2E_ADMIN_USER: ${E2E_ADMIN_USER:-postulator}
      E2E_ADMIN_EMAIL: ${E2E_ADMIN_EMAIL:-postulator@example.test}
      E2E_ADMIN_PASSWORD: ${E2E_ADMIN_PASSWORD:-postulator-admin}
      E2E_SEO: ${E2E_SEO:-none}
      E2E_WOO: ${E2E_WOO:-1}
    entrypoint: ["/bin/sh", "/e2e/bootstrap.sh"]
    volumes:
      - wp:/var/www/html
      - ../../wp-plugin/postulator-companion:/var/www/html/wp-content/plugins/postulator-companion
      - ./:/e2e

  phplint:
    image: wordpress:6.9.2-php8.3-apache
    profiles:
      - tools
    entrypoint: ["/bin/sh"]
    volumes:
      - ../../wp-plugin:/plugin:ro

volumes:
  db:
  wp:
```

Create `docker/e2e/bootstrap.sh`:

```sh
#!/bin/sh
set -eu

SITE_URL="${E2E_SITE_URL:-http://localhost:8089}"
ADMIN_USER="${E2E_ADMIN_USER:-postulator}"
ADMIN_EMAIL="${E2E_ADMIN_EMAIL:-postulator@example.test}"
ADMIN_PASSWORD="${E2E_ADMIN_PASSWORD:-postulator-admin}"
SEO="${E2E_SEO:-none}"
WOO="${E2E_WOO:-1}"
OUT="/e2e/.env.generated"

attempt=0
until wp db check >/dev/null 2>&1; do
	attempt=$((attempt + 1))
	if [ "$attempt" -ge 60 ]; then
		echo "the database did not become reachable" >&2
		exit 1
	fi
	sleep 2
done

if ! wp core is-installed >/dev/null 2>&1; then
	wp core install \
		--url="$SITE_URL" \
		--title="Postulator E2E" \
		--admin_user="$ADMIN_USER" \
		--admin_password="$ADMIN_PASSWORD" \
		--admin_email="$ADMIN_EMAIL" \
		--skip-email
fi

wp plugin activate postulator-companion

for slug in wordpress-seo seo-by-rank-math; do
	if wp plugin is-active "$slug" >/dev/null 2>&1; then
		wp plugin deactivate "$slug"
	fi
done

case "$SEO" in
	none)
		;;
	yoast)
		wp plugin install wordpress-seo --activate
		;;
	rankmath)
		wp plugin install seo-by-rank-math --activate
		;;
	*)
		echo "E2E_SEO must be none, yoast or rankmath" >&2
		exit 1
		;;
esac

if [ "$WOO" = "1" ] && ! wp plugin is-active woocommerce >/dev/null 2>&1; then
	wp plugin install woocommerce --activate
fi

wp rewrite structure '/%postname%/' --hard
wp rewrite flush --hard

if [ "$(wp user application-password list "$ADMIN_USER" --format=count)" != "0" ]; then
	wp user application-password delete "$ADMIN_USER" --all
fi
APP_PASSWORD="$(wp user application-password create "$ADMIN_USER" postulator-e2e --porcelain)"

{
	echo "E2E_WP_URL=$SITE_URL"
	echo "E2E_WP_USER=$ADMIN_USER"
	echo "E2E_WP_APP_PASSWORD=$APP_PASSWORD"
	echo "E2E_SEO=$SEO"
	echo "E2E_WOO=$WOO"
} > "$OUT"

echo "wordpress is provisioned at $SITE_URL"
```

Append `docker/e2e/.env.generated` to `.gitignore`.

Append to the `tasks:` block of `Taskfile.yml`:

```yaml
  e2e:up:
    summary: Starts the docker WordPress stack and provisions e2e credentials
    dir: docker/e2e
    env:
      E2E_SEO: '{{.E2E_SEO | default "none"}}'
      E2E_WOO: '{{.E2E_WOO | default "1"}}'
    cmds:
      - docker compose up -d --wait db wordpress
      - docker compose run --rm bootstrap

  e2e:down:
    summary: Stops the docker WordPress stack and removes its volumes
    dir: docker/e2e
    cmds:
      - docker compose down -v --remove-orphans
      - cmd: powershell -NoProfile -Command "Remove-Item -Force -ErrorAction SilentlyContinue .env.generated"
        platforms: [windows]
      - cmd: rm -f .env.generated
        platforms: [linux, darwin]

  e2e:reset:
    summary: Recreates the docker WordPress stack from scratch
    cmds:
      - task: e2e:down
      - task: e2e:up

  e2e:test:
    summary: Runs the docker end-to-end suite
    cmds:
      - go test -tags e2e -count=1 ./internal/adapters/wp/e2e/...
```

- [ ] **Step 4: Run it**

Task 2 creates the plugin files that `wp plugin activate postulator-companion` needs, so this task and task 2 are implemented back to back and committed separately. Land task 2's plugin files first on disk, then:

```
task e2e:up
go test -tags e2e -count=1 -run TestSiteAnswersAndCredentialsAuthenticate ./internal/adapters/wp/e2e/...
```

Confirm `docker/e2e/.env.generated` exists with five keys and that `git status` does not list it.

- [ ] **Step 5: Commit**

```
test(e2e): docker wordpress stack and generated credentials
```

---

### Task 2: Plugin skeleton, `GET /manifest` and `task plugin:lint`

**Files:**
- Create: `wp-plugin/postulator-companion/postulator-companion.php`, `includes/http.php`, `includes/normalize.php`, `includes/seo.php`, `includes/routes.php`
- Create: `internal/adapters/wp/e2e/manifest_test.go`
- Modify: `Taskfile.yml`

**Interfaces:**
- Produces: namespace `Postulator\Companion`; constants `VERSION`, `NAMESPACE_PATH`, `TYPES`, `TERM_TYPES`, `DEFAULT_LIMIT`, `MAX_LIMIT`, `TERM_MODIFIED_KEY`, `INSTALLED_OPTION`, `META_KEYS`; functions `permission_check`, `not_found`, `forbidden`, `invalid`, `site_host`, `normalize_path`, `url_to_path`, `parent_path`, `internal_href_to_path`, `collapse_text`, `detect_plugin`, `register_routes`, `manifest`; task `plugin:lint`.
- Consumes: WordPress core only.

- [ ] **Step 1: Write the failing test**

Create `internal/adapters/wp/e2e/manifest_test.go`:

```go
//go:build e2e

package e2e_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"
)

type manifest struct {
	Version      string   `json:"version"`
	Capabilities []string `json:"capabilities"`
	SEOPlugin    string   `json:"seoPlugin"`
	WPVersion    string   `json:"wpVersion"`
	Site         string   `json:"site"`
}

func readManifest(t *testing.T, c *client) manifest {
	t.Helper()

	var out manifest
	c.expect(t, http.MethodGet, "/wp-json/postulator/v1/manifest", nil, http.StatusOK, &out)
	return out
}

func TestManifestDescribesTheSite(t *testing.T) {
	c, env := newClient(t)

	got := readManifest(t, c)

	if got.Version != "1.0.0" {
		t.Errorf("version = %q, want 1.0.0", got.Version)
	}
	want := []string{"bulk", "seo_meta", "content_hash", "raw"}
	if !slices.Equal(got.Capabilities, want) {
		t.Errorf("capabilities = %v, want %v", got.Capabilities, want)
	}
	if got.SEOPlugin != env.seo {
		t.Errorf("seoPlugin = %q, want %q", got.SEOPlugin, env.seo)
	}
	if !strings.HasPrefix(got.WPVersion, "6.") {
		t.Errorf("wpVersion = %q, want a 6.x version", got.WPVersion)
	}
	if got.Site != strings.TrimSuffix(env.baseURL, "/") {
		t.Errorf("site = %q, want %q", got.Site, env.baseURL)
	}
}

func TestManifestRequiresAuthentication(t *testing.T) {
	c, _ := newClient(t)

	anonymous := &client{base: c.base, user: "", pass: "", http: c.http}
	status, body := anonymous.request(t, http.MethodGet, "/wp-json/postulator/v1/manifest", nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("anonymous manifest: status %d, want 401, body %s", status, body)
	}
}
```

- [ ] **Step 2: Run it and read the failure**

```
go test -tags e2e -count=1 -run TestManifest ./internal/adapters/wp/e2e/...
```

Both fail with a 404 from WordPress: the route does not exist.

- [ ] **Step 3: Implement**

Create `wp-plugin/postulator-companion/postulator-companion.php`:

```php
<?php
/**
 * Plugin Name: Postulator Companion
 * Version: 1.0.0
 * Requires at least: 6.4
 * Requires PHP: 8.1
 */

namespace Postulator\Companion;

defined( 'ABSPATH' ) || exit;

const VERSION           = '1.0.0';
const NAMESPACE_PATH    = 'postulator/v1';
const CAPABILITIES      = array( 'bulk', 'seo_meta', 'content_hash', 'raw' );
const TYPES             = array( 'page', 'post', 'product', 'product_cat' );
const TERM_TYPES        = array( 'product_cat' );
const DEFAULT_LIMIT     = 100;
const MAX_LIMIT         = 500;
const TERM_MODIFIED_KEY = '_postulator_modified';
const INSTALLED_OPTION  = 'postulator_companion_installed_at';

require_once __DIR__ . '/includes/http.php';
require_once __DIR__ . '/includes/normalize.php';
require_once __DIR__ . '/includes/seo.php';
require_once __DIR__ . '/includes/routes.php';

register_activation_hook( __FILE__, __NAMESPACE__ . '\\activate' );

add_action( 'rest_api_init', __NAMESPACE__ . '\\register_routes' );

function activate(): void {
	add_option( INSTALLED_OPTION, current_time( 'mysql', true ) );
}
```

Each include is wired by the task that creates it: task 4 adds the `includes/content.php` require and the two `*_term` hooks, task 6 adds the `includes/head.php` require and the `init` hook. Nothing is required before it exists, and no file is created before the test that needs it.

Create `wp-plugin/postulator-companion/includes/http.php`:

```php
<?php

namespace Postulator\Companion;

defined( 'ABSPATH' ) || exit;

function permission_check() {
	if ( current_user_can( 'edit_posts' ) ) {
		return true;
	}
	return forbidden( 'an application password for a user with edit_posts is required' );
}

function forbidden( string $message ): \WP_Error {
	return new \WP_Error( 'postulator_forbidden', $message, array( 'status' => rest_authorization_required_code() ) );
}

function not_found( string $message ): \WP_Error {
	return new \WP_Error( 'postulator_not_found', $message, array( 'status' => 404 ) );
}

function invalid( string $code, string $message ): \WP_Error {
	return new \WP_Error( $code, $message, array( 'status' => 400 ) );
}

function failed( string $message ): \WP_Error {
	return new \WP_Error( 'postulator_failed', $message, array( 'status' => 500 ) );
}
```

Create `wp-plugin/postulator-companion/includes/normalize.php`:

```php
<?php

namespace Postulator\Companion;

defined( 'ABSPATH' ) || exit;

function site_host(): string {
	$host = wp_parse_url( home_url(), PHP_URL_HOST );
	return is_string( $host ) ? strtolower( $host ) : '';
}

function normalize_path( string $path ): string {
	$path = (string) preg_replace( '#/+#', '/', $path );
	if ( '' === $path ) {
		return '/';
	}
	if ( '/' !== $path[0] ) {
		$path = '/' . $path;
	}
	if ( '/' !== substr( $path, -1 ) ) {
		$path .= '/';
	}
	return $path;
}

function url_to_path( string $url ): string {
	$path = wp_parse_url( $url, PHP_URL_PATH );
	return normalize_path( is_string( $path ) ? $path : '/' );
}

function parent_path( string $path ): string {
	$trimmed = rtrim( $path, '/' );
	$cut     = strrpos( $trimmed, '/' );
	if ( false === $cut ) {
		return '/';
	}
	return normalize_path( substr( $trimmed, 0, $cut + 1 ) );
}

function internal_href_to_path( string $href, string $base ): string {
	$href = trim( $href );
	if ( '' === $href || '#' === $href[0] ) {
		return '';
	}

	$parts = wp_parse_url( $href );
	if ( ! is_array( $parts ) ) {
		return '';
	}
	if ( isset( $parts['scheme'] ) && ! in_array( strtolower( $parts['scheme'] ), array( 'http', 'https' ), true ) ) {
		return '';
	}
	if ( isset( $parts['host'] ) && strtolower( $parts['host'] ) !== site_host() ) {
		return '';
	}

	$path = isset( $parts['path'] ) ? $parts['path'] : '';
	if ( '' === $path ) {
		return isset( $parts['host'] ) ? '/' : '';
	}
	if ( '/' !== $path[0] ) {
		if ( isset( $parts['host'] ) ) {
			return '';
		}
		$path = $base . $path;
	}
	return normalize_path( $path );
}

function collapse_text( string $text ): string {
	return trim( (string) preg_replace( '/\s+/u', ' ', $text ) );
}
```

Create `wp-plugin/postulator-companion/includes/seo.php` (the read and write helpers arrive in tasks 4 and 6; this task creates the file with `detect_plugin` and `META_KEYS` only):

```php
<?php

namespace Postulator\Companion;

defined( 'ABSPATH' ) || exit;

const META_KEYS = array(
	'yoast'    => array(
		'title'         => '_yoast_wpseo_title',
		'description'   => '_yoast_wpseo_metadesc',
		'canonical'     => '_yoast_wpseo_canonical',
		'ogTitle'       => '_yoast_wpseo_opengraph-title',
		'ogDescription' => '_yoast_wpseo_opengraph-description',
	),
	'rankmath' => array(
		'title'         => 'rank_math_title',
		'description'   => 'rank_math_description',
		'canonical'     => 'rank_math_canonical_url',
		'ogTitle'       => 'rank_math_facebook_title',
		'ogDescription' => 'rank_math_facebook_description',
	),
	'none'     => array(
		'title'         => '_postulator_seo_title',
		'description'   => '_postulator_seo_description',
		'canonical'     => '_postulator_canonical',
		'ogTitle'       => '_postulator_og_title',
		'ogDescription' => '_postulator_og_description',
	),
);

function detect_plugin(): string {
	if ( defined( 'WPSEO_VERSION' ) ) {
		return 'yoast';
	}
	if ( class_exists( 'RankMath' ) ) {
		return 'rankmath';
	}
	return 'none';
}
```

Create `wp-plugin/postulator-companion/includes/routes.php` with the manifest route only; tasks 4, 6 and 7 add their routes and callbacks to this file:

```php
<?php

namespace Postulator\Companion;

defined( 'ABSPATH' ) || exit;

function register_routes(): void {
	register_rest_route(
		NAMESPACE_PATH,
		'/manifest',
		array(
			'methods'             => \WP_REST_Server::READABLE,
			'callback'            => __NAMESPACE__ . '\\manifest',
			'permission_callback' => __NAMESPACE__ . '\\permission_check',
		)
	);
}

function manifest(): \WP_REST_Response {
	return new \WP_REST_Response(
		array(
			'version'      => VERSION,
			'capabilities' => CAPABILITIES,
			'seoPlugin'    => detect_plugin(),
			'wpVersion'    => (string) get_bloginfo( 'version' ),
			'site'         => (string) home_url(),
		),
		200
	);
}
```

Append to the `tasks:` block of `Taskfile.yml`:

```yaml
  plugin:lint:
    summary: Runs php -l over every companion plugin source file
    cmds:
      - cmd: docker compose -f docker/e2e/compose.yaml run --rm --no-deps -T phplint -c 'find /plugin -name "*.php" -print0 | xargs -0 -n1 php -l'
        platforms: [windows]
      - cmd: find wp-plugin -name "*.php" -print0 | xargs -0 -n1 php -l
        platforms: [linux, darwin]
```

- [ ] **Step 4: Run it**

```
task plugin:lint
task e2e:reset
go test -tags e2e -count=1 -run TestManifest ./internal/adapters/wp/e2e/...
```

`plugin:lint` must report `No syntax errors detected` once per file. Both manifest tests must pass, and `TestSiteAnswersAndCredentialsAuthenticate` from task 1 must now pass too.

- [ ] **Step 5: Commit**

```
feat(wp-plugin): companion plugin skeleton and manifest route
```

---

### Task 3: Deterministic plugin packaging

**Files:**
- Create: `cmd/pluginzip/main.go`, `cmd/pluginzip/pack.go`
- Test: `cmd/pluginzip/pack_test.go`
- Modify: `Taskfile.yml`

**Interfaces:**
- Produces: `bin/postulator-companion.zip`; task `plugin:zip`.
- Consumes: the standard library only.

- [ ] **Step 1: Write the failing test**

Create `cmd/pluginzip/pack_test.go`:

```go
package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()

	root := t.TempDir()
	for name, body := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	return root
}

func TestCollect(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		files map[string]string
		want  []string
		fails bool
	}{
		{
			name:  "sorted and slash separated",
			files: map[string]string{"z.php": "z", "includes/b.php": "b", "a.php": "a"},
			want:  []string{"a.php", "includes/b.php", "z.php"},
		},
		{
			name:  "single file",
			files: map[string]string{"only.php": "x"},
			want:  []string{"only.php"},
		},
		{
			name:  "empty tree is an error",
			files: map[string]string{},
			fails: true,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := collect(writeTree(t, testCase.files))
			if testCase.fails {
				if err == nil {
					t.Fatalf("collect: no error, want one")
				}
				return
			}
			if err != nil {
				t.Fatalf("collect: %v", err)
			}
			if len(got) != len(testCase.want) {
				t.Fatalf("collect = %v, want %v", got, testCase.want)
			}
			for i, name := range testCase.want {
				if got[i] != name {
					t.Errorf("collect[%d] = %q, want %q", i, got[i], name)
				}
			}
		})
	}
}

func TestWriteIsDeterministic(t *testing.T) {
	t.Parallel()

	root := writeTree(t, map[string]string{"postulator-companion.php": "<?php\n", "includes/http.php": "<?php\n"})
	names, err := collect(root)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}

	var first, second bytes.Buffer
	if err := write(root, "postulator-companion", names, &first); err != nil {
		t.Fatalf("write first: %v", err)
	}
	if err := write(root, "postulator-companion", names, &second); err != nil {
		t.Fatalf("write second: %v", err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatalf("two runs produced %d and %d bytes that differ", first.Len(), second.Len())
	}

	archive, err := zip.NewReader(bytes.NewReader(first.Bytes()), int64(first.Len()))
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}
	want := []string{"postulator-companion/includes/http.php", "postulator-companion/postulator-companion.php"}
	if len(archive.File) != len(want) {
		t.Fatalf("archive holds %d entries, want %d", len(archive.File), len(want))
	}
	for i, entry := range archive.File {
		if entry.Name != want[i] {
			t.Errorf("entry %d = %q, want %q", i, entry.Name, want[i])
		}
		if !entry.Modified.Equal(epoch) {
			t.Errorf("entry %q modified %s, want %s", entry.Name, entry.Modified, epoch)
		}
	}
}

func TestRunPackagesTheRealPlugin(t *testing.T) {
	t.Parallel()

	out := filepath.Join(t.TempDir(), "postulator-companion.zip")
	if err := run(filepath.Join("..", "..", "wp-plugin", "postulator-companion"), out); err != nil {
		t.Fatalf("run: %v", err)
	}

	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read %s: %v", out, err)
	}
	archive, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}

	found := false
	for _, entry := range archive.File {
		if entry.Name == "postulator-companion/postulator-companion.php" {
			found = true
		}
	}
	if !found {
		t.Fatalf("archive does not hold the plugin main file")
	}
}

func TestRunRejectsAMissingSource(t *testing.T) {
	t.Parallel()

	if err := run(filepath.Join(t.TempDir(), "absent"), filepath.Join(t.TempDir(), "out.zip")); err == nil {
		t.Fatalf("run: no error, want one")
	}
}
```

- [ ] **Step 2: Run it and read the failure**

```
go test -race -count=1 ./cmd/pluginzip/...
```

It fails to build: `collect`, `write`, `run` and `epoch` are undefined.

- [ ] **Step 3: Implement**

Create `cmd/pluginzip/pack.go`:

```go
package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"time"
)

var epoch = time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)

func collect(source string) ([]string, error) {
	var names []string
	err := filepath.WalkDir(source, func(current string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(source, current)
		if err != nil {
			return err
		}
		names = append(names, filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan %s: %w", source, err)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("%s holds no files", source)
	}
	slices.Sort(names)
	return names, nil
}

func write(source, root string, names []string, sink *bytes.Buffer) error {
	archive := zip.NewWriter(sink)
	for _, name := range names {
		header := &zip.FileHeader{Name: path.Join(root, name), Method: zip.Deflate, Modified: epoch}
		header.SetMode(0o644)

		entry, err := archive.CreateHeader(header)
		if err != nil {
			return fmt.Errorf("create entry %s: %w", name, err)
		}
		body, err := os.ReadFile(filepath.Join(source, filepath.FromSlash(name)))
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		if _, err := entry.Write(body); err != nil {
			return fmt.Errorf("write entry %s: %w", name, err)
		}
	}
	if err := archive.Close(); err != nil {
		return fmt.Errorf("close archive: %w", err)
	}
	return nil
}

func run(source, out string) error {
	names, err := collect(source)
	if err != nil {
		return err
	}

	var buffer bytes.Buffer
	if err := write(source, filepath.Base(filepath.Clean(source)), names, &buffer); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(out), err)
	}
	if err := os.WriteFile(out, buffer.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", out, err)
	}
	return nil
}
```

Create `cmd/pluginzip/main.go`:

```go
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	source := flag.String("source", "wp-plugin/postulator-companion", "plugin directory to package")
	out := flag.String("out", "bin/postulator-companion.zip", "zip archive to write")
	flag.Parse()

	if err := run(*source, *out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(*out)
}
```

Append to the `tasks:` block of `Taskfile.yml`:

```yaml
  plugin:zip:
    summary: Packages the companion plugin deterministically
    cmds:
      - go run ./cmd/pluginzip -source wp-plugin/postulator-companion -out {{.BIN_DIR}}/postulator-companion.zip
```

- [ ] **Step 4: Run it**

```
go test -race -count=1 ./cmd/pluginzip/...
task plugin:zip
```

Then prove determinism end to end from the shell by running `task plugin:zip` twice and comparing the file hashes.

- [ ] **Step 5: Commit**

```
build(wp-plugin): deterministic plugin packaging
```

---

### Task 4: `GET /content` for pages and posts

**Files:**
- Create: `wp-plugin/postulator-companion/includes/content.php`
- Modify: `wp-plugin/postulator-companion/postulator-companion.php`, `includes/routes.php`, `includes/seo.php`
- Test: `internal/adapters/wp/e2e/content_test.go`, additions to `harness_test.go`

**Interfaces:**
- Produces: `GET /wp-json/postulator/v1/content?since=&cursor=&types=&limit=`; PHP functions `content_hash`, `rfc3339`, `load_dom`, `rendered_content`, `first_h1`, `extract_links`, `encode_cursor`, `decode_cursor`, `query_posts_page`, `post_item`, `collect`, `touch_term`, `read_post_seo`, `parse_types`, `parse_since`, `clamp_limit`, `content_list`.
- Consumes: `normalize.php`, `seo.php`, `http.php`.

- [ ] **Step 1: Write the failing test**

Append to `internal/adapters/wp/e2e/harness_test.go`:

```go
type link struct {
	Href   string `json:"href"`
	Anchor string `json:"anchor"`
}

type itemMeta struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Canonical   string `json:"canonical"`
}

type contentItem struct {
	ID          int      `json:"id"`
	Type        string   `json:"type"`
	Slug        string   `json:"slug"`
	Path        string   `json:"path"`
	Parent      int      `json:"parent"`
	Status      string   `json:"status"`
	Modified    string   `json:"modified"`
	ContentHash string   `json:"contentHash"`
	Title       string   `json:"title"`
	H1          string   `json:"h1"`
	Meta        itemMeta `json:"meta"`
	Links       []link   `json:"links"`
}

type contentPage struct {
	Items      []contentItem `json:"items"`
	NextCursor string        `json:"nextCursor"`
}

type pageSpec struct {
	title   string
	slug    string
	content string
	parent  int
	status  string
}

func createPage(t *testing.T, c *client, spec pageSpec) int {
	t.Helper()

	status := spec.status
	if status == "" {
		status = "publish"
	}
	body := map[string]any{
		"title":   spec.title,
		"slug":    spec.slug,
		"content": spec.content,
		"status":  status,
	}
	if spec.parent != 0 {
		body["parent"] = spec.parent
	}

	var created struct {
		ID int `json:"id"`
	}
	c.expect(t, http.MethodPost, "/wp-json/wp/v2/pages", body, http.StatusCreated, &created)
	if created.ID == 0 {
		t.Fatalf("created page has id 0")
	}
	t.Cleanup(func() {
		c.request(t, http.MethodDelete, fmt.Sprintf("/wp-json/wp/v2/pages/%d?force=true", created.ID), nil)
	})
	return created.ID
}

func listContent(t *testing.T, c *client, query string) contentPage {
	t.Helper()

	var page contentPage
	c.expect(t, http.MethodGet, "/wp-json/postulator/v1/content?"+query, nil, http.StatusOK, &page)
	return page
}

func findItem(t *testing.T, c *client, types string, id int) contentItem {
	t.Helper()

	cursor := ""
	for range 50 {
		query := "types=" + types + "&limit=100"
		if cursor != "" {
			query += "&cursor=" + url.QueryEscape(cursor)
		}
		page := listContent(t, c, query)
		for _, item := range page.Items {
			if item.ID == id {
				return item
			}
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	t.Fatalf("id %d was not listed under types=%s", id, types)
	return contentItem{}
}
```

`fmt` and `net/url` join the harness imports.

Create `internal/adapters/wp/e2e/content_test.go`:

```go
//go:build e2e

package e2e_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestContentReportsPathHashLinksAndH1(t *testing.T) {
	c, env := newClient(t)

	parentSlug := uniqueSlug("koffein")
	parentID := createPage(t, c, pageSpec{title: "Koffein", slug: parentSlug, content: "<p>Parent</p>"})

	childSlug := uniqueSlug("powder")
	body := fmt.Sprintf(
		`<h1>Powder</h1><h2>Section</h2><ul><li>one</li></ul><p>Up to <a href="/%s/">Koffein</a> and out to <a href="https://example.com/x/">Example</a>.</p><img src="/wp-content/uploads/x.png" alt="x" />`,
		parentSlug,
	)
	childID := createPage(t, c, pageSpec{title: "Powder", slug: childSlug, content: body, parent: parentID})

	item := findItem(t, c, "page", childID)

	if item.Type != "page" {
		t.Errorf("type = %q, want page", item.Type)
	}
	if item.Slug != childSlug {
		t.Errorf("slug = %q, want %q", item.Slug, childSlug)
	}
	wantPath := "/" + parentSlug + "/" + childSlug + "/"
	if item.Path != wantPath {
		t.Errorf("path = %q, want %q", item.Path, wantPath)
	}
	if item.Parent != parentID {
		t.Errorf("parent = %d, want %d", item.Parent, parentID)
	}
	if item.Status != "publish" {
		t.Errorf("status = %q, want publish", item.Status)
	}
	if item.Title != "Powder" {
		t.Errorf("title = %q, want Powder", item.Title)
	}
	if item.H1 != "Powder" {
		t.Errorf("h1 = %q, want Powder", item.H1)
	}
	if item.ContentHash != hashOf(body) {
		t.Errorf("contentHash = %q, want %q", item.ContentHash, hashOf(body))
	}
	if _, err := time.Parse(time.RFC3339, item.Modified); err != nil {
		t.Errorf("modified = %q is not RFC3339: %v", item.Modified, err)
	}

	wantLinks := []link{{Href: "/" + parentSlug + "/", Anchor: "Koffein"}}
	if len(item.Links) != len(wantLinks) {
		t.Fatalf("links = %v, want %v", item.Links, wantLinks)
	}
	if item.Links[0] != wantLinks[0] {
		t.Errorf("links[0] = %v, want %v", item.Links[0], wantLinks[0])
	}

	if env.seo == "none" && item.Meta != (itemMeta{}) {
		t.Errorf("meta = %v, want empty for a page with no SEO meta", item.Meta)
	}
}

func TestContentIncludesDraftsAndExcludesTrash(t *testing.T) {
	c, _ := newClient(t)

	draftID := createPage(t, c, pageSpec{title: "Draft", slug: uniqueSlug("draft"), content: "<p>d</p>", status: "draft"})

	item := findItem(t, c, "page", draftID)
	if item.Status != "draft" {
		t.Errorf("status = %q, want draft", item.Status)
	}

	c.expect(t, http.MethodDelete, fmt.Sprintf("/wp-json/wp/v2/pages/%d", draftID), nil, http.StatusOK, nil)

	page := listContent(t, c, "types=page&limit=500")
	for _, listed := range page.Items {
		if listed.ID == draftID {
			t.Fatalf("trashed page %d is still listed", draftID)
		}
	}
}

func TestContentPagesByKeysetCursor(t *testing.T) {
	c, _ := newClient(t)

	for i := range 3 {
		createPage(t, c, pageSpec{title: fmt.Sprintf("Cursor %d", i), slug: uniqueSlug("cursor"), content: "<p>c</p>"})
	}

	seen := make(map[int]int)
	cursor := ""
	pages := 0
	for range 50 {
		query := "types=page&limit=1"
		if cursor != "" {
			query += "&cursor=" + url.QueryEscape(cursor)
		}
		page := listContent(t, c, query)
		pages++
		if len(page.Items) > 1 {
			t.Fatalf("limit=1 returned %d items", len(page.Items))
		}
		for _, item := range page.Items {
			seen[item.ID]++
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	if pages < 3 {
		t.Fatalf("walked only %d pages, want at least 3", pages)
	}
	for id, count := range seen {
		if count != 1 {
			t.Errorf("id %d appeared %d times", id, count)
		}
	}
}

func TestContentFiltersBySince(t *testing.T) {
	c, _ := newClient(t)

	before := time.Now().UTC().Add(time.Second).Format(time.RFC3339)
	time.Sleep(2 * time.Second)
	freshID := createPage(t, c, pageSpec{title: "Fresh", slug: uniqueSlug("fresh"), content: "<p>f</p>"})

	page := listContent(t, c, "types=page&limit=500&since="+url.QueryEscape(before))
	found := false
	for _, item := range page.Items {
		if item.ID == freshID {
			found = true
		}
		if item.Modified < before {
			t.Errorf("item %d modified %q is older than since %q", item.ID, item.Modified, before)
		}
	}
	if !found {
		t.Fatalf("page %d created after %q is not listed", freshID, before)
	}
}

func TestContentRejectsBadParameters(t *testing.T) {
	c, _ := newClient(t)

	cases := []struct {
		name  string
		query string
	}{
		{name: "unknown type", query: "types=widget"},
		{name: "unparseable since", query: "since=yesterday"},
		{name: "undecodable cursor", query: "cursor=%21%21%21"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			status, body := c.request(t, http.MethodGet, "/wp-json/postulator/v1/content?"+testCase.query, nil)
			if status != http.StatusBadRequest {
				t.Fatalf("status %d, want 400, body %s", status, body)
			}
			if !strings.Contains(string(body), `"code"`) {
				t.Fatalf("body %s has no code field", body)
			}
		})
	}
}

func TestContentClampsLimit(t *testing.T) {
	c, _ := newClient(t)

	page := listContent(t, c, "types=page&limit=100000")
	if len(page.Items) > 500 {
		t.Fatalf("limit=100000 returned %d items, want at most 500", len(page.Items))
	}
}
```

- [ ] **Step 2: Run it and read the failure**

```
go test -tags e2e -count=1 -run TestContent ./internal/adapters/wp/e2e/...
```

Every case fails with a 404: the route does not exist yet.

- [ ] **Step 3: Implement**

Create `wp-plugin/postulator-companion/includes/content.php`:

```php
<?php

namespace Postulator\Companion;

defined( 'ABSPATH' ) || exit;

function content_hash( string $content ): string {
	return hash( 'sha256', $content );
}

function rfc3339( string $mysql_gmt ): string {
	$time = strtotime( $mysql_gmt . ' UTC' );
	if ( false === $time || $time <= 0 ) {
		return gmdate( 'Y-m-d\TH:i:s\Z', 0 );
	}
	return gmdate( 'Y-m-d\TH:i:s\Z', $time );
}

function installed_at(): string {
	$value = get_option( INSTALLED_OPTION, '' );
	return is_string( $value ) && '' !== $value ? $value : '1970-01-01 00:00:00';
}

function touch_term( int $term_id, int $term_taxonomy_id, string $taxonomy ): void {
	if ( ! in_array( $taxonomy, TERM_TYPES, true ) ) {
		return;
	}
	update_term_meta( $term_id, TERM_MODIFIED_KEY, current_time( 'mysql', true ) );
}

function load_dom( string $html ): ?\DOMDocument {
	if ( '' === trim( $html ) || ! class_exists( '\DOMDocument' ) ) {
		return null;
	}

	$document = new \DOMDocument();
	$previous = libxml_use_internal_errors( true );
	$loaded   = $document->loadHTML( '<?xml encoding="UTF-8">' . $html, LIBXML_NOWARNING | LIBXML_NOERROR );
	libxml_clear_errors();
	libxml_use_internal_errors( $previous );

	return $loaded ? $document : null;
}

function rendered_content( \WP_Post $post ): string {
	$previous        = isset( $GLOBALS['post'] ) ? $GLOBALS['post'] : null;
	$GLOBALS['post'] = $post;
	setup_postdata( $post );
	$rendered = (string) apply_filters( 'the_content', $post->post_content );
	wp_reset_postdata();
	$GLOBALS['post'] = $previous;

	return $rendered;
}

function first_h1( string $rendered ): string {
	$document = load_dom( $rendered );
	if ( null !== $document ) {
		$nodes = $document->getElementsByTagName( 'h1' );
		if ( $nodes->length > 0 ) {
			return collapse_text( (string) $nodes->item( 0 )->textContent );
		}
		return '';
	}
	if ( 1 === preg_match( '#<h1\b[^>]*>(.*?)</h1>#is', $rendered, $matches ) ) {
		return collapse_text( html_entity_decode( wp_strip_all_tags( $matches[1] ), ENT_QUOTES | ENT_HTML5, 'UTF-8' ) );
	}
	return '';
}

function extract_links( string $content, string $base ): array {
	$links    = array();
	$document = load_dom( $content );
	if ( null === $document ) {
		return $links;
	}

	foreach ( $document->getElementsByTagName( 'a' ) as $anchor ) {
		$path = internal_href_to_path( (string) $anchor->getAttribute( 'href' ), $base );
		if ( '' === $path ) {
			continue;
		}
		$links[] = array(
			'href'   => $path,
			'anchor' => collapse_text( (string) $anchor->textContent ),
		);
	}
	return $links;
}

function encode_cursor( array $state ): string {
	$json = wp_json_encode( $state );
	if ( ! is_string( $json ) ) {
		return '';
	}
	return rtrim( strtr( base64_encode( $json ), '+/', '-_' ), '=' );
}

function decode_cursor( string $cursor ): ?array {
	$padded = strtr( $cursor, '-_', '+/' );
	$padded .= str_repeat( '=', ( 4 - ( strlen( $padded ) % 4 ) ) % 4 );

	$raw = base64_decode( $padded, true );
	if ( false === $raw ) {
		return null;
	}
	$state = json_decode( $raw, true );
	if ( ! is_array( $state ) || ! isset( $state['p'] ) || ! in_array( $state['p'], array( 'post', 'term' ), true ) ) {
		return null;
	}
	return $state;
}

function query_posts_page( array $types, string $since_gmt, ?array $cursor, int $limit ): array {
	global $wpdb;

	$placeholders = implode( ', ', array_fill( 0, count( $types ), '%s' ) );
	$sql          = "SELECT ID, post_modified_gmt FROM {$wpdb->posts} WHERE post_type IN ( {$placeholders} ) AND post_status NOT IN ( 'auto-draft', 'trash' )";
	$args         = $types;

	if ( '' !== $since_gmt ) {
		$sql   .= ' AND post_modified_gmt >= %s';
		$args[] = $since_gmt;
	}
	if ( null !== $cursor ) {
		$sql   .= ' AND ( post_modified_gmt > %s OR ( post_modified_gmt = %s AND ID > %d ) )';
		$args[] = (string) $cursor['m'];
		$args[] = (string) $cursor['m'];
		$args[] = (int) $cursor['i'];
	}

	$sql   .= ' ORDER BY post_modified_gmt ASC, ID ASC LIMIT %d';
	$args[] = $limit;

	$rows = $wpdb->get_results( $wpdb->prepare( $sql, $args ), ARRAY_A );
	return is_array( $rows ) ? $rows : array();
}

function post_item( \WP_Post $post ): array {
	$path    = url_to_path( (string) get_permalink( $post ) );
	$content = (string) $post->post_content;

	return array(
		'id'          => (int) $post->ID,
		'type'        => (string) $post->post_type,
		'slug'        => (string) $post->post_name,
		'path'        => $path,
		'parent'      => (int) $post->post_parent,
		'status'      => (string) $post->post_status,
		'modified'    => rfc3339( (string) $post->post_modified_gmt ),
		'contentHash' => content_hash( $content ),
		'title'       => (string) $post->post_title,
		'h1'          => first_h1( rendered_content( $post ) ),
		'meta'        => read_post_seo( (int) $post->ID ),
		'links'       => extract_links( $content, parent_path( $path ) ),
	);
}

function collect( array $types, string $since_gmt, ?array $cursor, int $limit ): array {
	$items = array();

	$rows = query_posts_page( $types, $since_gmt, $cursor, $limit );
	foreach ( $rows as $row ) {
		$post = get_post( (int) $row['ID'] );
		if ( $post instanceof \WP_Post ) {
			$items[] = post_item( $post );
		}
	}
	if ( count( $rows ) < $limit ) {
		return array( 'items' => $items, 'nextCursor' => '' );
	}

	$last = end( $rows );
	return array(
		'items'      => $items,
		'nextCursor' => encode_cursor(
			array(
				'p' => 'post',
				'm' => (string) $last['post_modified_gmt'],
				'i' => (int) $last['ID'],
			)
		),
	);
}
```

This task ships the post phase only, so `TYPES` in `postulator-companion.php` is `array( 'page', 'post', 'product' )` and `product_cat` is not yet a valid `types` value. Task 5 adds `product_cat` to `TYPES`, adds `query_terms_page` and `term_item`, and replaces `collect` with the two-phase version. Every intermediate state is therefore complete, lintable and honest about what it supports.

Append to `wp-plugin/postulator-companion/includes/seo.php`:

```php
function read_post_seo( int $post_id ): array {
	$map = META_KEYS[ detect_plugin() ];

	return array(
		'title'       => (string) get_post_meta( $post_id, $map['title'], true ),
		'description' => (string) get_post_meta( $post_id, $map['description'], true ),
		'canonical'   => (string) get_post_meta( $post_id, $map['canonical'], true ),
	);
}
```

Append to `wp-plugin/postulator-companion/includes/routes.php` — inside `register_routes`:

```php
	register_rest_route(
		NAMESPACE_PATH,
		'/content',
		array(
			'methods'             => \WP_REST_Server::READABLE,
			'callback'            => __NAMESPACE__ . '\\content_list',
			'permission_callback' => __NAMESPACE__ . '\\permission_check',
		)
	);
```

and, at file scope:

```php
function parse_types( string $types ) {
	if ( '' === $types ) {
		return TYPES;
	}

	$requested = array_values( array_filter( array_map( 'trim', explode( ',', $types ) ), 'strlen' ) );
	foreach ( $requested as $type ) {
		if ( ! in_array( $type, TYPES, true ) ) {
			return invalid( 'postulator_invalid_type', 'unknown content type: ' . $type );
		}
	}
	return array_values( array_unique( $requested ) );
}

function parse_since( string $since ) {
	if ( '' === $since ) {
		return '';
	}

	$time = strtotime( $since );
	if ( false === $time ) {
		return invalid( 'postulator_invalid_since', 'since must be an RFC3339 timestamp' );
	}
	return gmdate( 'Y-m-d H:i:s', $time );
}

function clamp_limit( $limit ): int {
	$value = is_numeric( $limit ) ? (int) $limit : DEFAULT_LIMIT;
	if ( $value < 1 ) {
		return DEFAULT_LIMIT;
	}
	return min( $value, MAX_LIMIT );
}

function content_list( \WP_REST_Request $request ) {
	$types = parse_types( (string) $request->get_param( 'types' ) );
	if ( is_wp_error( $types ) ) {
		return $types;
	}

	$since = parse_since( (string) $request->get_param( 'since' ) );
	if ( is_wp_error( $since ) ) {
		return $since;
	}

	$raw_cursor = (string) $request->get_param( 'cursor' );
	$cursor     = null;
	if ( '' !== $raw_cursor ) {
		$cursor = decode_cursor( $raw_cursor );
		if ( null === $cursor ) {
			return invalid( 'postulator_invalid_cursor', 'cursor is not a cursor this endpoint issued' );
		}
	}

	$page = collect( $types, $since, $cursor, clamp_limit( $request->get_param( 'limit' ) ) );

	return new \WP_REST_Response(
		array(
			'items'      => array_values( $page['items'] ),
			'nextCursor' => (string) $page['nextCursor'],
		),
		200
	);
}
```

Add to `postulator-companion.php`, after the `seo.php` require and before the `routes.php` require:

```php
require_once __DIR__ . '/includes/content.php';
```

and, with the other hooks:

```php
add_action( 'created_term', __NAMESPACE__ . '\\touch_term', 10, 3 );
add_action( 'edited_term', __NAMESPACE__ . '\\touch_term', 10, 3 );
```

- [ ] **Step 4: Run it**

```
task plugin:lint
go test -tags e2e -count=1 -run TestContent ./internal/adapters/wp/e2e/...
```

All six content tests must pass. Read the `contentHash` assertion carefully: an inequality there means core altered the content on save and D1's premise is wrong — stop and investigate rather than relaxing the assertion.

- [ ] **Step 5: Commit**

```
feat(wp-plugin): content listing with keyset cursor, links and hashes
```

---

### Task 5: WooCommerce products and product categories

**Files:**
- Modify: `docker/e2e/bootstrap.sh`, `wp-plugin/postulator-companion/postulator-companion.php`, `includes/content.php`, `includes/seo.php`
- Test: `internal/adapters/wp/e2e/commerce_test.go`

**Interfaces:**
- Produces: `product` and `product_cat` in `GET /content`; PHP functions `query_terms_page`, `term_item`, `read_term_seo`; stable docker fixtures `postulator-powder` (product) and `postulator-koffein` (product_cat).
- Consumes: WooCommerce for the `product` post type and the `product_cat` taxonomy.

- [ ] **Step 1: Write the failing test**

Create `internal/adapters/wp/e2e/commerce_test.go`:

```go
//go:build e2e

package e2e_test

import (
	"testing"
	"time"
)

func requireWoo(t *testing.T, env environment) {
	t.Helper()

	if !env.woo {
		t.Fatalf("this suite needs WooCommerce; run `task e2e:reset` with E2E_WOO=1")
	}
}

func findBySlug(t *testing.T, c *client, types, slug string) contentItem {
	t.Helper()

	cursor := ""
	for range 50 {
		query := "types=" + types + "&limit=100"
		if cursor != "" {
			query += "&cursor=" + url.QueryEscape(cursor)
		}
		page := listContent(t, c, query)
		for _, item := range page.Items {
			if item.Slug == slug {
				return item
			}
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	t.Fatalf("slug %q was not listed under types=%s", slug, types)
	return contentItem{}
}

func TestProductCategoryIsListed(t *testing.T) {
	c, env := newClient(t)
	requireWoo(t, env)

	item := findBySlug(t, c, "product_cat", "postulator-koffein")

	if item.Type != "product_cat" {
		t.Errorf("type = %q, want product_cat", item.Type)
	}
	if item.Title != "Koffein" {
		t.Errorf("title = %q, want Koffein", item.Title)
	}
	if item.Path != "/product-category/postulator-koffein/" {
		t.Errorf("path = %q, want /product-category/postulator-koffein/", item.Path)
	}
	if item.Parent != 0 {
		t.Errorf("parent = %d, want 0", item.Parent)
	}
	if item.Status != "publish" {
		t.Errorf("status = %q, want publish", item.Status)
	}
	if item.H1 != "" {
		t.Errorf("h1 = %q, want empty for a term", item.H1)
	}
	if len(item.Links) != 0 {
		t.Errorf("links = %v, want none for a term", item.Links)
	}
	if item.ContentHash != hashOf("Caffeine products.") {
		t.Errorf("contentHash = %q, want the hash of the term description", item.ContentHash)
	}
	if _, err := time.Parse(time.RFC3339, item.Modified); err != nil {
		t.Errorf("modified = %q is not RFC3339: %v", item.Modified, err)
	}
}

func TestProductIsListedWithItsLinks(t *testing.T) {
	c, env := newClient(t)
	requireWoo(t, env)

	item := findBySlug(t, c, "product", "postulator-powder")

	if item.Type != "product" {
		t.Errorf("type = %q, want product", item.Type)
	}
	if item.Title != "Powder" {
		t.Errorf("title = %q, want Powder", item.Title)
	}
	want := link{Href: "/product-category/postulator-koffein/", Anchor: "Koffein"}
	if len(item.Links) != 1 || item.Links[0] != want {
		t.Fatalf("links = %v, want exactly %v", item.Links, want)
	}
}

func TestMixedTypesPageThroughBothPhases(t *testing.T) {
	c, env := newClient(t)
	requireWoo(t, env)

	seen := make(map[string]bool)
	cursor := ""
	for range 100 {
		query := "types=product,product_cat&limit=1"
		if cursor != "" {
			query += "&cursor=" + url.QueryEscape(cursor)
		}
		page := listContent(t, c, query)
		for _, item := range page.Items {
			seen[item.Type] = true
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}

	if !seen["product"] || !seen["product_cat"] {
		t.Fatalf("walking types=product,product_cat saw %v, want both phases", seen)
	}
}
```

- [ ] **Step 2: Run it and read the failure**

```
go test -tags e2e -count=1 -run "TestProduct|TestMixedTypes" ./internal/adapters/wp/e2e/...
```

All three fail: `types=product_cat` is a `400` because task 4 left it out of `TYPES`, and the fixtures do not exist.

- [ ] **Step 3: Implement**

Insert into `docker/e2e/bootstrap.sh`, between the WooCommerce install block and the `wp rewrite structure` line:

```sh
if [ "$WOO" = "1" ]; then
	if [ "$(wp term list product_cat --slug=postulator-koffein --format=count)" = "0" ]; then
		wp term create product_cat "Koffein" \
			--slug=postulator-koffein \
			--description="Caffeine products."
	fi
	if [ "$(wp post list --post_type=product --name=postulator-powder --format=count)" = "0" ]; then
		PRODUCT_ID="$(wp post create \
			--post_type=product \
			--post_title="Powder" \
			--post_name=postulator-powder \
			--post_status=publish \
			--post_content='<p>See <a href="/product-category/postulator-koffein/">Koffein</a>.</p>' \
			--porcelain)"
		wp post term add "$PRODUCT_ID" product_cat postulator-koffein
	fi
fi
```

The plugin is activated before WooCommerce is installed and before these fixtures are created, so `created_term` fires with the plugin loaded and `_postulator_modified` is set on the category at creation.

In `postulator-companion.php`, restore the full type list:

```php
const TYPES = array( 'page', 'post', 'product', 'product_cat' );
```

Append to `wp-plugin/postulator-companion/includes/content.php`:

```php
function query_terms_page( string $since_gmt, ?array $cursor, int $limit ): array {
	global $wpdb;

	$installed = installed_at();
	$sql       = "SELECT t.term_id AS term_id, COALESCE( tm.meta_value, %s ) AS modified_gmt
		FROM {$wpdb->term_taxonomy} tt
		INNER JOIN {$wpdb->terms} t ON t.term_id = tt.term_id
		LEFT JOIN {$wpdb->termmeta} tm ON tm.term_id = t.term_id AND tm.meta_key = %s
		WHERE tt.taxonomy = %s";
	$args      = array( $installed, TERM_MODIFIED_KEY, 'product_cat' );

	if ( '' !== $since_gmt ) {
		$sql   .= ' AND COALESCE( tm.meta_value, %s ) >= %s';
		$args[] = $installed;
		$args[] = $since_gmt;
	}
	if ( null !== $cursor ) {
		$sql   .= ' AND t.term_id > %d';
		$args[] = (int) $cursor['i'];
	}

	$sql   .= ' ORDER BY t.term_id ASC LIMIT %d';
	$args[] = $limit;

	$rows = $wpdb->get_results( $wpdb->prepare( $sql, $args ), ARRAY_A );
	return is_array( $rows ) ? $rows : array();
}

function term_item( \WP_Term $term, string $modified_gmt ): array {
	$description = (string) $term->description;
	$link        = get_term_link( $term );

	return array(
		'id'          => (int) $term->term_id,
		'type'        => (string) $term->taxonomy,
		'slug'        => (string) $term->slug,
		'path'        => is_string( $link ) ? url_to_path( $link ) : '/',
		'parent'      => (int) $term->parent,
		'status'      => 'publish',
		'modified'    => rfc3339( $modified_gmt ),
		'contentHash' => content_hash( $description ),
		'title'       => (string) $term->name,
		'h1'          => '',
		'meta'        => read_term_seo( $term ),
		'links'       => array(),
	);
}
```

Replace `collect` in that file with the two-phase version:

```php
function collect( array $types, string $since_gmt, ?array $cursor, int $limit ): array {
	$post_types = array_values( array_diff( $types, TERM_TYPES ) );
	$term_types = array_values( array_intersect( $types, TERM_TYPES ) );

	$items = array();
	$phase = null !== $cursor ? (string) $cursor['p'] : ( empty( $post_types ) ? 'term' : 'post' );
	$state = $cursor;

	if ( 'post' === $phase ) {
		$rows = empty( $post_types ) ? array() : query_posts_page( $post_types, $since_gmt, $state, $limit );
		foreach ( $rows as $row ) {
			$post = get_post( (int) $row['ID'] );
			if ( $post instanceof \WP_Post ) {
				$items[] = post_item( $post );
			}
		}
		if ( count( $rows ) === $limit ) {
			$last = end( $rows );
			return array(
				'items'      => $items,
				'nextCursor' => encode_cursor(
					array(
						'p' => 'post',
						'm' => (string) $last['post_modified_gmt'],
						'i' => (int) $last['ID'],
					)
				),
			);
		}
		$state = null;
	}

	$remaining = $limit - count( $items );
	if ( empty( $term_types ) || $remaining < 1 ) {
		return array( 'items' => $items, 'nextCursor' => '' );
	}

	$rows = query_terms_page( $since_gmt, $state, $remaining );
	foreach ( $rows as $row ) {
		$term = get_term( (int) $row['term_id'] );
		if ( $term instanceof \WP_Term ) {
			$items[] = term_item( $term, (string) $row['modified_gmt'] );
		}
	}
	if ( count( $rows ) === $remaining ) {
		$last = end( $rows );
		return array(
			'items'      => $items,
			'nextCursor' => encode_cursor( array( 'p' => 'term', 'i' => (int) $last['term_id'] ) ),
		);
	}
	return array( 'items' => $items, 'nextCursor' => '' );
}
```

Append to `wp-plugin/postulator-companion/includes/seo.php`:

```php
function read_term_seo( \WP_Term $term ): array {
	$plugin = detect_plugin();

	if ( 'yoast' === $plugin ) {
		$all = get_option( 'wpseo_taxonomy_meta', array() );
		$row = is_array( $all ) && isset( $all[ $term->taxonomy ][ $term->term_id ] ) ? $all[ $term->taxonomy ][ $term->term_id ] : array();

		return array(
			'title'       => isset( $row['wpseo_title'] ) ? (string) $row['wpseo_title'] : '',
			'description' => isset( $row['wpseo_desc'] ) ? (string) $row['wpseo_desc'] : '',
			'canonical'   => isset( $row['wpseo_canonical'] ) ? (string) $row['wpseo_canonical'] : '',
		);
	}

	$map = META_KEYS[ $plugin ];
	return array(
		'title'       => (string) get_term_meta( $term->term_id, $map['title'], true ),
		'description' => (string) get_term_meta( $term->term_id, $map['description'], true ),
		'canonical'   => (string) get_term_meta( $term->term_id, $map['canonical'], true ),
	);
}
```

- [ ] **Step 4: Run it**

```
task plugin:lint
task e2e:reset
go test -tags e2e -count=1 ./internal/adapters/wp/e2e/...
```

All three commerce tests and every earlier test must pass. If `path` is not `/product-category/postulator-koffein/`, read `wp option get woocommerce_permalinks` in the container: the assertion pins WooCommerce's default category base and a different value is a real difference the Go side must know about.

- [ ] **Step 5: Commit**

```
feat(wp-plugin): products and product categories in the content listing
```

---

### Task 6: `PUT /seo-meta/{id}` and head rendering without an SEO plugin

**Files:**
- Create: `wp-plugin/postulator-companion/includes/head.php`
- Modify: `wp-plugin/postulator-companion/postulator-companion.php`, `includes/routes.php`, `includes/seo.php`
- Test: `internal/adapters/wp/e2e/seometa_test.go`

**Interfaces:**
- Produces: `PUT /wp-json/postulator/v1/seo-meta/{id}`; PHP functions `write_post_seo`, `seo_meta_update`, `boot_head`, `filter_document_title`, `filter_canonical_url`, `print_head_tags`.
- Consumes: `seo.php`, `http.php`.

- [ ] **Step 1: Write the failing test**

Create `internal/adapters/wp/e2e/seometa_test.go`:

```go
//go:build e2e

package e2e_test

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
)

type seoResult struct {
	Applied   []string `json:"applied"`
	SEOPlugin string   `json:"seoPlugin"`
}

func writeSEO(t *testing.T, c *client, id int, body map[string]string) seoResult {
	t.Helper()

	var out seoResult
	c.expect(t, http.MethodPut, fmt.Sprintf("/wp-json/postulator/v1/seo-meta/%d", id), body, http.StatusOK, &out)
	return out
}

func TestSEOMetaRoundTripsAndRenders(t *testing.T) {
	c, env := newClient(t)

	slug := uniqueSlug("seo")
	id := createPage(t, c, pageSpec{title: "SEO", slug: slug, content: "<h1>SEO</h1><p>body</p>"})

	title := "Postulator SEO title " + slug
	description := "Postulator SEO description " + slug
	canonical := strings.TrimSuffix(env.baseURL, "/") + "/" + slug + "/"

	result := writeSEO(t, c, id, map[string]string{
		"title":         title,
		"description":   description,
		"canonical":     canonical,
		"ogTitle":       "OG " + title,
		"ogDescription": "OG " + description,
	})

	want := []string{"title", "description", "canonical", "ogTitle", "ogDescription"}
	if !slices.Equal(result.Applied, want) {
		t.Errorf("applied = %v, want %v", result.Applied, want)
	}
	if result.SEOPlugin != env.seo {
		t.Errorf("seoPlugin = %q, want %q", result.SEOPlugin, env.seo)
	}

	item := findItem(t, c, "page", id)
	if item.Meta.Title != title {
		t.Errorf("meta.title = %q, want %q", item.Meta.Title, title)
	}
	if item.Meta.Description != description {
		t.Errorf("meta.description = %q, want %q", item.Meta.Description, description)
	}
	if item.Meta.Canonical != canonical {
		t.Errorf("meta.canonical = %q, want %q", item.Meta.Canonical, canonical)
	}

	html := c.fetchPage(t, strings.TrimSuffix(env.baseURL, "/")+item.Path)
	if !strings.Contains(html, "<title>"+title+"</title>") {
		t.Errorf("rendered page has no <title>%s</title>", title)
	}
	if !strings.Contains(html, description) {
		t.Errorf("rendered page has no meta description %q", description)
	}
	if !strings.Contains(html, canonical) {
		t.Errorf("rendered page has no canonical %q", canonical)
	}
	if strings.Count(html, `rel="canonical"`) != 1 {
		t.Errorf("rendered page has %d canonical tags, want exactly 1", strings.Count(html, `rel="canonical"`))
	}
	if strings.Count(html, "<title>") != 1 {
		t.Errorf("rendered page has %d title tags, want exactly 1", strings.Count(html, "<title>"))
	}
	if strings.Count(html, `name="description"`) > 1 {
		t.Errorf("rendered page has %d description tags, want at most 1", strings.Count(html, `name="description"`))
	}
}

func TestSEOMetaAppliesOnlyPresentFieldsAndRemovesOnEmpty(t *testing.T) {
	c, _ := newClient(t)

	slug := uniqueSlug("partial")
	id := createPage(t, c, pageSpec{title: "Partial", slug: slug, content: "<p>p</p>"})

	writeSEO(t, c, id, map[string]string{"title": "first", "description": "kept"})

	result := writeSEO(t, c, id, map[string]string{"title": "second"})
	if !slices.Equal(result.Applied, []string{"title"}) {
		t.Errorf("applied = %v, want [title]", result.Applied)
	}

	item := findItem(t, c, "page", id)
	if item.Meta.Title != "second" {
		t.Errorf("meta.title = %q, want second", item.Meta.Title)
	}
	if item.Meta.Description != "kept" {
		t.Errorf("meta.description = %q, want kept", item.Meta.Description)
	}

	writeSEO(t, c, id, map[string]string{"description": ""})
	item = findItem(t, c, "page", id)
	if item.Meta.Description != "" {
		t.Errorf("meta.description = %q after an empty write, want empty", item.Meta.Description)
	}
}

func TestSEOMetaRejectsAnUnknownID(t *testing.T) {
	c, _ := newClient(t)

	status, body := c.request(t, http.MethodPut, "/wp-json/postulator/v1/seo-meta/98765432", map[string]string{"title": "x"})
	if status != http.StatusNotFound {
		t.Fatalf("status %d, want 404, body %s", status, body)
	}
}
```

- [ ] **Step 2: Run it and read the failure**

```
go test -tags e2e -count=1 -run TestSEOMeta ./internal/adapters/wp/e2e/...
```

All three fail with a 404: the route does not exist.

- [ ] **Step 3: Implement**

Append to `wp-plugin/postulator-companion/includes/seo.php`:

```php
function write_post_seo( int $post_id, array $fields ): array {
	$map     = META_KEYS[ detect_plugin() ];
	$applied = array();

	foreach ( $map as $field => $key ) {
		if ( ! array_key_exists( $field, $fields ) ) {
			continue;
		}
		$value = $fields[ $field ];
		if ( '' === $value ) {
			delete_post_meta( $post_id, $key );
		} else {
			update_post_meta( $post_id, $key, wp_slash( $value ) );
		}
		$applied[] = $field;
	}
	return $applied;
}
```

Create `wp-plugin/postulator-companion/includes/head.php`:

```php
<?php

namespace Postulator\Companion;

defined( 'ABSPATH' ) || exit;

function boot_head(): void {
	if ( 'none' !== detect_plugin() ) {
		return;
	}
	add_filter( 'pre_get_document_title', __NAMESPACE__ . '\\filter_document_title' );
	add_filter( 'get_canonical_url', __NAMESPACE__ . '\\filter_canonical_url', 10, 2 );
	add_action( 'wp_head', __NAMESPACE__ . '\\print_head_tags', 2 );
}

function head_object_id(): int {
	return is_singular() ? (int) get_queried_object_id() : 0;
}

function filter_document_title( $title ) {
	$id = head_object_id();
	if ( 0 === $id ) {
		return $title;
	}
	$own = (string) get_post_meta( $id, META_KEYS['none']['title'], true );
	return '' === $own ? $title : $own;
}

function filter_canonical_url( $canonical, $post ) {
	if ( ! $post instanceof \WP_Post ) {
		return $canonical;
	}
	$own = (string) get_post_meta( (int) $post->ID, META_KEYS['none']['canonical'], true );
	return '' === $own ? $canonical : $own;
}

function print_head_tags(): void {
	static $printed = false;

	if ( $printed ) {
		return;
	}
	$id = head_object_id();
	if ( 0 === $id ) {
		return;
	}
	$printed = true;

	$map = META_KEYS['none'];
	foreach ( array(
		'description'    => array( 'name', 'description' ),
		'ogTitle'        => array( 'property', 'og:title' ),
		'ogDescription'  => array( 'property', 'og:description' ),
	) as $field => $tag ) {
		$value = (string) get_post_meta( $id, $map[ $field ], true );
		if ( '' === $value ) {
			continue;
		}
		printf(
			'<meta %s="%s" content="%s" />' . "\n",
			esc_attr( $tag[0] ),
			esc_attr( $tag[1] ),
			esc_attr( $value )
		);
	}
}
```

Append to `register_routes` in `includes/routes.php`:

```php
	register_rest_route(
		NAMESPACE_PATH,
		'/seo-meta/(?P<id>\d+)',
		array(
			'methods'             => \WP_REST_Server::EDITABLE,
			'callback'            => __NAMESPACE__ . '\\seo_meta_update',
			'permission_callback' => __NAMESPACE__ . '\\permission_check',
		)
	);
```

and at file scope in the same file:

```php
function editable_post( \WP_REST_Request $request ) {
	$post = get_post( (int) $request['id'] );
	if ( ! $post instanceof \WP_Post || in_array( $post->post_status, array( 'auto-draft', 'trash' ), true ) ) {
		return not_found( 'no post with that id' );
	}
	if ( ! current_user_can( 'edit_post', $post->ID ) ) {
		return forbidden( 'editing this post is not allowed' );
	}
	return $post;
}

function seo_meta_update( \WP_REST_Request $request ) {
	$post = editable_post( $request );
	if ( is_wp_error( $post ) ) {
		return $post;
	}

	$body = $request->get_json_params();
	if ( ! is_array( $body ) ) {
		return invalid( 'postulator_invalid_body', 'a JSON object body is required' );
	}

	$fields = array();
	foreach ( array_keys( META_KEYS['none'] ) as $field ) {
		if ( array_key_exists( $field, $body ) ) {
			$fields[ $field ] = is_scalar( $body[ $field ] ) ? (string) $body[ $field ] : '';
		}
	}

	return new \WP_REST_Response(
		array(
			'applied'   => write_post_seo( (int) $post->ID, $fields ),
			'seoPlugin' => detect_plugin(),
		),
		200
	);
}
```

Add to `postulator-companion.php`, with the other requires and hooks:

```php
require_once __DIR__ . '/includes/head.php';
```

```php
add_action( 'init', __NAMESPACE__ . '\\boot_head', 20 );
```

- [ ] **Step 4: Run it**

```
task plugin:lint
go test -tags e2e -count=1 ./internal/adapters/wp/e2e/...
```

The duplicate-tag counts are the assertions that matter most here: exactly one `<title>`, exactly one `rel="canonical"`, at most one `name="description"`. A count of two means D5's approach failed for the active theme and the fix belongs in the filter choice, not in the assertion.

- [ ] **Step 5: Commit**

```
feat(wp-plugin): seo meta writes and head rendering without an seo plugin
```

---

### Task 7: `GET` and `PUT /content/{id}/raw`

**Files:**
- Modify: `wp-plugin/postulator-companion/includes/routes.php`
- Test: `internal/adapters/wp/e2e/raw_test.go`

**Interfaces:**
- Produces: `GET`/`PUT /wp-json/postulator/v1/content/{id}/raw`; PHP functions `raw_get`, `raw_update`.
- Consumes: `content.php` (`content_hash`), `http.php`, `editable_post` from task 6.

- [ ] **Step 1: Write the failing test**

Create `internal/adapters/wp/e2e/raw_test.go`:

```go
//go:build e2e

package e2e_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

type rawContent struct {
	ID          int    `json:"id"`
	Type        string `json:"type"`
	Content     string `json:"content"`
	ContentHash string `json:"contentHash"`
}

const richBody = `<h2>Heading</h2><ul><li>first</li><li>second</li></ul>` +
	`<p>Link to <a href="/koffein/" title="up">Koffein</a>.</p>` +
	`<img src="/wp-content/uploads/powder.png" alt="Powder" width="800" height="600" />`

func TestRawRoundTripsWithoutFiltering(t *testing.T) {
	c, _ := newClient(t)

	original := "<p>original</p>"
	id := createPage(t, c, pageSpec{title: "Raw", slug: uniqueSlug("raw"), content: original})

	var got rawContent
	c.expect(t, http.MethodGet, fmt.Sprintf("/wp-json/postulator/v1/content/%d/raw", id), nil, http.StatusOK, &got)

	if got.ID != id {
		t.Errorf("id = %d, want %d", got.ID, id)
	}
	if got.Type != "page" {
		t.Errorf("type = %q, want page", got.Type)
	}
	if got.Content != original {
		t.Errorf("content = %q, want %q", got.Content, original)
	}
	if got.ContentHash != hashOf(original) {
		t.Errorf("contentHash = %q, want %q", got.ContentHash, hashOf(original))
	}

	var written struct {
		ContentHash string `json:"contentHash"`
	}
	c.expect(t, http.MethodPut, fmt.Sprintf("/wp-json/postulator/v1/content/%d/raw", id),
		map[string]string{"content": richBody, "expectedHash": got.ContentHash},
		http.StatusOK, &written)

	if written.ContentHash != hashOf(richBody) {
		t.Fatalf("contentHash = %q, want %q: WordPress altered the stored content", written.ContentHash, hashOf(richBody))
	}

	var reread rawContent
	c.expect(t, http.MethodGet, fmt.Sprintf("/wp-json/postulator/v1/content/%d/raw", id), nil, http.StatusOK, &reread)
	if reread.Content != richBody {
		t.Fatalf("re-read content = %q, want %q", reread.Content, richBody)
	}
}

func TestRawRejectsAStaleHash(t *testing.T) {
	c, _ := newClient(t)

	original := "<p>stale</p>"
	id := createPage(t, c, pageSpec{title: "Stale", slug: uniqueSlug("stale"), content: original})

	status, body := c.request(t, http.MethodPut, fmt.Sprintf("/wp-json/postulator/v1/content/%d/raw", id),
		map[string]string{"content": "<p>new</p>", "expectedHash": hashOf("something else")})

	if status != http.StatusConflict {
		t.Fatalf("status %d, want 409, body %s", status, body)
	}

	var conflict struct {
		Code        string `json:"code"`
		Message     string `json:"message"`
		CurrentHash string `json:"currentHash"`
	}
	if err := json.Unmarshal(body, &conflict); err != nil {
		t.Fatalf("decode conflict: %v, body %s", err, body)
	}
	if conflict.Code != "hash_mismatch" {
		t.Errorf("code = %q, want hash_mismatch", conflict.Code)
	}
	if conflict.Message == "" {
		t.Errorf("message is empty")
	}
	if conflict.CurrentHash != hashOf(original) {
		t.Errorf("currentHash = %q, want %q", conflict.CurrentHash, hashOf(original))
	}

	var unchanged rawContent
	c.expect(t, http.MethodGet, fmt.Sprintf("/wp-json/postulator/v1/content/%d/raw", id), nil, http.StatusOK, &unchanged)
	if unchanged.Content != original {
		t.Fatalf("a rejected write changed the content to %q", unchanged.Content)
	}
}

func TestRawWritesWithoutAnExpectedHash(t *testing.T) {
	c, _ := newClient(t)

	id := createPage(t, c, pageSpec{title: "Blind", slug: uniqueSlug("blind"), content: "<p>before</p>"})

	var written struct {
		ContentHash string `json:"contentHash"`
	}
	c.expect(t, http.MethodPut, fmt.Sprintf("/wp-json/postulator/v1/content/%d/raw", id),
		map[string]string{"content": "<p>after</p>"}, http.StatusOK, &written)

	if written.ContentHash != hashOf("<p>after</p>") {
		t.Errorf("contentHash = %q, want %q", written.ContentHash, hashOf("<p>after</p>"))
	}
}

func TestRawRejectsBadRequests(t *testing.T) {
	c, _ := newClient(t)

	id := createPage(t, c, pageSpec{title: "Bad", slug: uniqueSlug("bad"), content: "<p>b</p>"})

	cases := []struct {
		name   string
		path   string
		body   map[string]string
		status int
	}{
		{name: "unknown id", path: "/wp-json/postulator/v1/content/98765432/raw", body: map[string]string{"content": "x"}, status: http.StatusNotFound},
		{name: "missing content", path: fmt.Sprintf("/wp-json/postulator/v1/content/%d/raw", id), body: map[string]string{"expectedHash": "x"}, status: http.StatusBadRequest},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			status, body := c.request(t, http.MethodPut, testCase.path, testCase.body)
			if status != testCase.status {
				t.Fatalf("status %d, want %d, body %s", status, testCase.status, body)
			}
		})
	}
}

func TestRawRequiresAuthentication(t *testing.T) {
	c, _ := newClient(t)

	id := createPage(t, c, pageSpec{title: "Guarded", slug: uniqueSlug("guarded"), content: "<p>g</p>"})

	anonymous := &client{base: c.base, http: c.http}
	status, body := anonymous.request(t, http.MethodGet, fmt.Sprintf("/wp-json/postulator/v1/content/%d/raw", id), nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("status %d, want 401, body %s", status, body)
	}
}
```

- [ ] **Step 2: Run it and read the failure**

```
go test -tags e2e -count=1 -run TestRaw ./internal/adapters/wp/e2e/...
```

All five fail with a 404.

- [ ] **Step 3: Implement**

Append to `register_routes` in `includes/routes.php`:

```php
	register_rest_route(
		NAMESPACE_PATH,
		'/content/(?P<id>\d+)/raw',
		array(
			array(
				'methods'             => \WP_REST_Server::READABLE,
				'callback'            => __NAMESPACE__ . '\\raw_get',
				'permission_callback' => __NAMESPACE__ . '\\permission_check',
			),
			array(
				'methods'             => \WP_REST_Server::EDITABLE,
				'callback'            => __NAMESPACE__ . '\\raw_update',
				'permission_callback' => __NAMESPACE__ . '\\permission_check',
			),
		)
	);
```

and at file scope in the same file:

```php
function raw_get( \WP_REST_Request $request ) {
	$post = editable_post( $request );
	if ( is_wp_error( $post ) ) {
		return $post;
	}

	$content = (string) $post->post_content;
	return new \WP_REST_Response(
		array(
			'id'          => (int) $post->ID,
			'type'        => (string) $post->post_type,
			'content'     => $content,
			'contentHash' => content_hash( $content ),
		),
		200
	);
}

function raw_update( \WP_REST_Request $request ) {
	$post = editable_post( $request );
	if ( is_wp_error( $post ) ) {
		return $post;
	}

	$body = $request->get_json_params();
	if ( ! is_array( $body ) || ! array_key_exists( 'content', $body ) || ! is_string( $body['content'] ) ) {
		return invalid( 'postulator_invalid_body', 'content is required and must be a string' );
	}

	$current  = (string) $post->post_content;
	$expected = isset( $body['expectedHash'] ) && is_string( $body['expectedHash'] ) ? $body['expectedHash'] : '';
	if ( '' !== $expected && ! hash_equals( content_hash( $current ), $expected ) ) {
		return new \WP_REST_Response(
			array(
				'code'        => 'hash_mismatch',
				'message'     => 'the stored content does not match expectedHash',
				'currentHash' => content_hash( $current ),
			),
			409
		);
	}

	$updated = wp_update_post(
		array(
			'ID'           => (int) $post->ID,
			'post_content' => wp_slash( (string) $body['content'] ),
		),
		true
	);
	if ( is_wp_error( $updated ) ) {
		return failed( $updated->get_error_message() );
	}

	$stored = (string) get_post_field( 'post_content', (int) $post->ID, 'raw' );
	return new \WP_REST_Response( array( 'contentHash' => content_hash( $stored ) ), 200 );
}
```

- [ ] **Step 4: Run it**

```
task plugin:lint
go test -tags e2e -count=1 ./internal/adapters/wp/e2e/...
```

`TestRawRoundTripsWithoutFiltering` is the proof for D1. If the returned hash differs from `hashOf(richBody)`, find out which filter changed the bytes before changing anything; the fix is never `remove_filter`.

- [ ] **Step 5: Commit**

```
feat(wp-plugin): raw content read and compare-and-swap write
```

---

### Task 8: Yoast mode

**Files:**
- Modify: `docker/e2e/bootstrap.sh`
- Test: no new test file; the existing SEO and manifest suites run under `E2E_SEO=yoast`

**Interfaces:**
- Produces: a green `E2E_SEO=yoast` run of the whole suite.
- Consumes: the `wordpress-seo` plugin from wordpress.org.

- [ ] **Step 1: Run the existing suite in the new mode and read the failure**

```
task e2e:down
task e2e:up E2E_SEO=yoast
go test -tags e2e -count=1 ./internal/adapters/wp/e2e/...
```

`TestManifestDescribesTheSite` must now report `seoPlugin=yoast`, and `TestSEOMetaRoundTripsAndRenders` must find Yoast's rendered title, description and canonical. Record exactly which assertions fail.

- [ ] **Step 2: Implement**

Yoast needs its indexable tables populated before it renders meta for existing content. Extend the `yoast` branch of the `case` in `docker/e2e/bootstrap.sh`:

```sh
	yoast)
		wp plugin install wordpress-seo --activate
		wp yoast index --reindex
		;;
```

If `wp yoast index` is not registered by the installed Yoast version, the script fails loudly and the correct response is to find the current command name with `wp help yoast` inside the container and use that — not to swallow the failure with `|| true`.

Nothing in the plugin changes: `detect_plugin()` already returns `yoast` from `defined('WPSEO_VERSION')`, `META_KEYS['yoast']` already holds the five keys, and `boot_head()` already declines to register anything when an SEO plugin is present. That is the point of the mode: it proves the detection branch rather than adding code.

- [ ] **Step 3: Run it**

```
task e2e:reset E2E_SEO=yoast
go test -tags e2e -count=1 ./internal/adapters/wp/e2e/...
```

The whole suite must be green. Then return to the default mode and confirm it is still green:

```
task e2e:reset
go test -tags e2e -count=1 ./internal/adapters/wp/e2e/...
```

- [ ] **Step 4: Commit**

```
test(e2e): yoast seo mode
```

---

### Task 9: Rank Math mode

**Files:**
- Modify: `docker/e2e/bootstrap.sh` (or revert the `rankmath` branch and record the reason)
- Modify: `docs/STATUS.md` if the mode is dropped

**Interfaces:**
- Produces: either a green `E2E_SEO=rankmath` run, or a recorded decision that the mode is not supported and why.

- [ ] **Step 1: Run the existing suite in the new mode and read the failure**

```
task e2e:reset E2E_SEO=rankmath
go test -tags e2e -count=1 ./internal/adapters/wp/e2e/...
```

Two outcomes are plausible and both are acceptable results of this task.

- [ ] **Step 2: If Rank Math renders meta from the written keys**

Nothing more is needed: `detect_plugin()` returns `rankmath` from `class_exists('RankMath')`, the five `rank_math_*` keys are already mapped, and the suite is green. Commit the mode as supported.

- [ ] **Step 3: If Rank Math suppresses front-end meta before its setup wizard completes**

Inspect the gate from inside the container rather than guessing at option names:

```
docker compose -f docker/e2e/compose.yaml run --rm bootstrap wp option list --search='rank_math*' --format=table
```

Set whatever that listing shows is the wizard/registration gate, re-run, and keep the mode if it becomes deterministic. If two attempts do not produce a stable run, **remove the `rankmath` branch from `bootstrap.sh`**, leave `E2E_SEO=rankmath` rejected by the `case` statement's `*)` arm, and record in `docs/STATUS.md` exactly which assertion was unstable and why — the Rank Math key mapping stays in `META_KEYS` either way, because the Go side needs it and `/content` meta reads are exercised by the mapping table, not by the front end.

- [ ] **Step 4: Commit**

```
test(e2e): rank math seo mode
```

or

```
docs(status): rank math e2e mode is not supported
```

---

### Task 10: CI job and e2e linting

**Files:**
- Modify: `.github/workflows/ci.yml`, `Taskfile.yml`

**Interfaces:**
- Produces: a `plugin` CI job on `ubuntu-latest`; task `lint:e2e`.
- Consumes: the PHP 8.3 that GitHub's Ubuntu images ship, and Go for `pluginzip`.

- [ ] **Step 1: Write the failing check**

Append to the `tasks:` block of `Taskfile.yml`:

```yaml
  lint:e2e:
    summary: Lints the build-tagged end-to-end sources
    cmds:
      - golangci-lint run --build-tags e2e ./internal/adapters/wp/e2e/...
```

Run it:

```
task lint:e2e
```

Every finding it reports is a real finding in code the default lint run never sees. Fix them in this task; that is the failing check.

- [ ] **Step 2: Implement**

Add a second job to `.github/workflows/ci.yml`, as a sibling of `build`:

```yaml
  plugin:
    name: Companion plugin
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true

      - name: Install Task
        run: go install github.com/go-task/task/v3/cmd/task@${{ env.TASK_VERSION }}

      - name: Show the PHP version
        run: php -v

      - name: Lint the plugin
        run: task plugin:lint

      - name: Package the plugin
        run: task plugin:zip

      - name: Upload the plugin package
        uses: actions/upload-artifact@v4
        with:
          name: postulator-companion
          path: bin/postulator-companion.zip
```

The docker end-to-end stack is deliberately absent from CI: `windows-latest` cannot run Linux containers, and running the stack on the Ubuntu job would double CI time to re-prove what a developer proves locally before every phase commit.

- [ ] **Step 3: Run it**

```
task lint:e2e
task plugin:lint
task plugin:zip
go vet ./...
golangci-lint run
go test -race -count=1 -covermode=atomic -coverprofile=coverage.out ./...
go run ./cmd/covergate -profile coverage.out
gofmt -l .
```

`gofmt -l .` must print nothing: it reads the e2e files even though the build tag excludes them from compilation, so a badly formatted test file fails CI's formatting step on the Windows job.

- [ ] **Step 4: Commit**

```
ci: lint and package the companion plugin on ubuntu
```

---

### Task 11: Documentation

**Files:**
- Modify: `docs/ARCHITECTURE.md`, `docs/CONVENTIONS.md`, `docs/STATUS.md`, `CLAUDE.md`

**Interfaces:** none.

- [ ] **Step 1: Write the section**

Add to `docs/ARCHITECTURE.md`, after the "Transport" section, at most 25 lines:

```markdown
## WordPress companion plugin

`wp-plugin/postulator-companion` is a dependency-free PHP plugin (PHP ≥ 8.1, WordPress
≥ 6.4) serving `/wp-json/postulator/v1` under WordPress application passwords. Every
route's permission callback requires `edit_posts`, reads included: this is a private
tool, not a public API. Write routes additionally check `edit_post` for the specific
post and return `401` unauthenticated, `403` authenticated-without-capability.

| Route | Purpose |
|---|---|
| `GET /manifest` | version, capabilities, detected SEO plugin, WP version, site URL |
| `GET /content` | keyset page over posts then `product_cat` terms, with hash, links and h1 |
| `PUT /seo-meta/{id}` | writes the five SEO fields present in the body |
| `GET /content/{id}/raw` | raw `post_content` and its sha256 |
| `PUT /content/{id}/raw` | compare-and-swap write, `409 hash_mismatch` on a stale hash |

The SEO plugin is detected with `defined('WPSEO_VERSION')` and `class_exists('RankMath')`
and selects the meta key set:

| Field | Yoast | Rank Math | none |
|---|---|---|---|
| title | `_yoast_wpseo_title` | `rank_math_title` | `_postulator_seo_title` |
| description | `_yoast_wpseo_metadesc` | `rank_math_description` | `_postulator_seo_description` |
| canonical | `_yoast_wpseo_canonical` | `rank_math_canonical_url` | `_postulator_canonical` |
| ogTitle | `_yoast_wpseo_opengraph-title` | `rank_math_facebook_title` | `_postulator_og_title` |
| ogDescription | `_yoast_wpseo_opengraph-description` | `rank_math_facebook_description` | `_postulator_og_description` |

With no SEO plugin the companion renders the head itself, replacing core's values rather
than adding tags: `pre_get_document_title` for `<title>`, `get_canonical_url` for the
canonical, and `wp_head` for description and OG. A theme that hardcodes its own `<title>`
instead of declaring `title-tag` support is out of scope.
```

Add to `docs/CONVENTIONS.md`, at most 15 lines:

````markdown
## Docker end-to-end

`docker/e2e/compose.yaml` pins WordPress 6.9.2 (PHP 8.3), MariaDB 11.4.12 and WP-CLI
2.12.0 and serves the site on `127.0.0.1:8089`. `task e2e:up` installs WordPress, sets
`/%postname%/` permalinks, activates the companion plugin, installs WooCommerce and its
two fixtures, and writes an application password to `docker/e2e/.env.generated`, which
is gitignored. `task e2e:down` removes the volumes, `task e2e:reset` does both.

```
task e2e:up                 E2E_SEO=none|yoast|rankmath, E2E_WOO=0|1
go test -tags e2e -count=1 ./internal/adapters/wp/e2e/...
task e2e:down
```

The suite is entirely `_test.go` behind `//go:build e2e`, so it adds no statements to the
coverage profile and the default lint run never sees it — lint it with `task lint:e2e`.
`task plugin:lint` runs `php -l` through the pinned image on Windows and through a local
`php` elsewhere; CI runs it and `task plugin:zip` on Ubuntu and never runs the stack.
````

- [ ] **Step 2: Update the handoff**

Add a "Phase 3B" paragraph to the "Where we are" section of `docs/STATUS.md`, a "Decisions taken in Phase 3B" section carrying D1 through D10 in one line each, and add to "Known gaps" that the e2e suite uses its own HTTP helper until track A's adapter lands.

Add to the "Standing rulings" list in `CLAUDE.md`, dated 2026-09-18:

- The companion plugin never removes a kses filter; it relies on `unfiltered_html` and returns the hash of what was actually stored.
- Every plugin write passes `wp_slash`, because `wp_insert_post` and `update_metadata` unslash.
- Path normalisation has no file-extension exception: leading and trailing slash, always, so PHP and Go cannot drift.
- The `/content` cursor names its phase (`post` then `term`); an undecodable cursor is a `400`, never a restart.
- `/seo-meta/{id}` and `/content/{id}/raw` address posts only; term ids are `404`.

Add to "Footguns" in `CLAUDE.md`:

- **The WP-CLI image's `www-data` is uid 82, the WordPress image's is uid 33.** The compose `bootstrap` service therefore runs as `user: "33:33"`; without it WP-CLI cannot write `.htaccess` or install a plugin into the shared volume, and the failure reads as a permissions error deep inside WordPress rather than as a container mismatch.
- **`go test -tags e2e` is the only thing that compiles `internal/adapters/wp/e2e`.** `go vet`, `golangci-lint run` and `cmd/covergate` all skip it silently, so `task lint:e2e` is not optional and `gofmt -l .` is what catches formatting there.

- [ ] **Step 3: Verify the line budgets**

`ARCHITECTURE.md`'s new section must be ≤ 25 lines and `CONVENTIONS.md`'s ≤ 15. Count them; trim the prose, not the tables.

- [ ] **Step 4: Commit**

```
docs: companion plugin architecture, e2e how-to and phase 3b status
```

---

## Verification gate

All of this must be green before the phase is handed to the reviewer.

Plugin and packaging:

```
task plugin:lint
task plugin:zip
```

End-to-end, once per SEO mode:

```
task e2e:reset
go test -tags e2e -count=1 ./internal/adapters/wp/e2e/...
task e2e:reset E2E_SEO=yoast
go test -tags e2e -count=1 ./internal/adapters/wp/e2e/...
task e2e:reset E2E_SEO=rankmath
go test -tags e2e -count=1 ./internal/adapters/wp/e2e/...
task e2e:down
```

The Rank Math line is dropped only if task 9 recorded why in `docs/STATUS.md`.

The standing Go gate, because this phase adds Go files:

```
task build
go vet ./...
$(go env GOPATH)/bin/golangci-lint run
task lint:e2e
go test -race -count=1 -covermode=atomic -coverprofile=coverage.out ./...
go run ./cmd/covergate -profile coverage.out
gofmt -l .
```

`golangci-lint` is run from `GOPATH/bin`, not from the scoop shim on `PATH`, for the
reason `CLAUDE.md` records. `gofmt -l .` must print nothing.

---

## Open questions for the orchestrator

These must be reconciled with track A's `wp-plugin/openapi.yaml` before either track
merges; each is a place where two reasonable implementations differ and the difference
is silent.

1. **`limit` above the maximum.** This plan clamps to 500. The alternative is a `400`. If
   track A sends its own default and expects a clamp, clamping is right; if it validates
   first, a `400` is more honest. One of the two must change.
2. **`nextCursor` when the listing is finished.** This plan emits `""`. If track A's
   schema makes it nullable or omits it, the Go decoder and the PHP encoder disagree
   about the end-of-list signal.
3. **Post-only write routes.** `/seo-meta/{id}` and `/content/{id}/raw` resolve `{id}`
   through `get_post()`, so a `product_cat` term id is a `404` (D10). If track A expects
   to write term SEO, the contract needs a type discriminator it does not currently have.
4. **Unconditional trailing slash.** `/wp-content/uploads/x.png` normalises to
   `/wp-content/uploads/x.png/` (D2). Track A must apply the identical rule, including
   the absence of a file-extension exception, or link compliance will mis-score.
5. **`www.` is not stripped** when comparing an `href` host to the site host. If track A
   strips it, an internal link written as `https://www.example.com/x/` on a site whose
   `home_url()` is `https://example.com` counts on one side and not the other.
6. **Term `modified`** is `_postulator_modified` term meta maintained by the plugin, with
   the plugin's install timestamp as the fallback (D3). It is not a WordPress-native
   field, so track A must not expect one and must tolerate a term whose `modified` is the
   install time rather than a real edit time.
7. **`title` is the raw `post_title`**, not `get_the_title()`, so `the_title` filters do
   not run. This matches core REST's `title.raw`; confirm track A reads it that way.
