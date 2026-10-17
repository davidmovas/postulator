# Postulator v2 — Phase 14: the site model, stages 1 to 4

> **For agentic workers:** each stage is planned at its start as a task plan in the writing-plans format (`docs/superpowers/plans/2026-MM-DD-phase-14-stage-N-<name>.md` is not allowed by section 10; the stage's task list goes into this file under its stage when the stage opens), then executed task by task with TDD under `superpowers:executing-plans` or `superpowers:subagent-driven-development`, at most two agents at once, one accountable owner.

**Goal:** Postulator observes a client's site into one structured page model, lets the client name his page types and the fields each may carry, gives the writer the full picture of a page, edits the site's content as blocks and fields through a field registry, and reconciles the site with his workbooks through findings.

**Architecture:** The model extends `domain/pagemap` with `domain/field`, `domain/block`, `domain/render`, `domain/pagetype`, `domain/finding` and `domain/contextpack`; four providers behind consumer-declared readers (stage 1) and writers (stage 3) fill and write it; the run engine, its artifacts and its revert carry every write; the companion plugin keeps its name and is strengthened; two docker stands prove the suites on a replica of the client's first site and on a vanilla stack.

**Tech stack:** Go 1.27, x/net/html, SQLite through goose migrations, Wails v3, React, PHP 7.4 for the plugin, docker compose for the stands.

**Spec:** `docs/superpowers/specs/2026-09-17-postulator-v2-design.md`, part 16; the decision record is `docs/DECISIONS.md`, 2026-10-17.

## Global constraints

- Every hard rule of `CLAUDE.md`: no comments, no stubs, consumer-declared interfaces, TDD with the coverage gates, cursor pagination, camelCase JSON, kernel errors at every boundary, no secret in a log, artifact or plain column, no new `.md`, bad code rewritten.
- A shipped migration is never edited: a column or table that must go is dropped by a later migration whose down restores it; a `CHECK` changes by the 0016/0025/0030 rebuild (`NO TRANSACTION`, foreign keys off, explicit `BEGIN`/`COMMIT`); a column a down must drop carries no foreign key.
- A new template step is a `StepName` and lands in `vocab.ts`, the blank recipe, the three tool enums and the schema ceiling in the same commit; a step a kind owns is a `PerKindStepName`.
- Every new tool or request field moves `schemaCeilingBytes` in the same commit; the deferred groups list is kept current.
- Nothing in `internal/`, `cmd/` or `wp-plugin/` names a theme, a CSS class or a site path; nothing on a client's site names the program; the plugin never removes a kses filter and passes `wp_slash` on every write; PHP 7.4.
- The hard-deny list is never written to an existing page: slug, regular and sale price, stock fields, SKU, status, categories and any taxonomy, the WordPress parent, a variation's attributes.
- Both stands green before a stage closes; `task build`, `go vet`, `golangci-lint run`, `task lint:e2e`, `go test -race`, `go run ./cmd/covergate`, the comment check, `npm run typecheck`, `npm run test:run` and `task ui:lint` green at every stage's gate.

## Review focus

Five input classes the spec implies and the stage tests must pin, most likely to bite first:

1. **A page whose raw is empty or only a shortcode** (the client's hubs): the content root must be `empty_raw` or `dynamic_only`, never `not_found`, and the first edit must land between the injected breadcrumb and grid. Stage 1b's extraction goldens and stage 3's hub test.
2. **A breadcrumb whose crumb has no page** (Home › Shop › Peptides(no page) › BPC): the parent falls to the next source with a `parent_unresolved` finding, never to Shop. Stage 1b's `DeriveParent` table.
3. **A product whose stored content WordPress altered on save** (kses, `balanceTags`): the block hashes of what the site holds, not of what we sent, are "ours". Stage 3's edit test against a non-`unfiltered_html` role on the replica.
4. **A site on a non-default port or under a subdirectory base**: `Site.Resolve` and `normalize.php` classify the same href the same way. Stage 1a's contract test over both stands' ports.
5. **A workbook row that matches two pages** (same title under two parents): `row_ambiguous` with both candidates, nothing written. Stage 4's matcher table.

---

## Stage 1 — the page model observed, read-only; the plugin strengthened for reads; two stands

**Goal.** Every page of a site, of any public type, exists in the model with its identity, terms, fields, body, extraction, typed links and a derived hierarchy with reasons; nothing new is written to a site, and the parent and slug stop being written on updates. Three gated sub-stages.

### 1a — the plugin and its client, the two stands

- **Touches.** `wp-plugin/postulator-companion/*` (the name kept, `uninstall.php` new), `wp-plugin/openapi.yaml`, `wp-plugin/testdata/hrefs.json`, `wp-plugin/tests/*`, `wp-plugin/plugin_footprint_test.go`, `internal/adapters/wp/{do,option,plugin,item,items,woo}.go`, `internal/adapters/wp/wptest` (the fake learns the new routes, custom types and taxonomies, variations, a product list, the settings), `internal/adapters/wp/registry`, `internal/adapters/wp/e2e/*`, `internal/domain/pagemap` (the href table's test), `internal/domain/site` (`Extensions`), `internal/application/sync/views.go`, `internal/app/{deps,sitespecific}_test.go`, `docker/e2e/{compose.yaml,bootstrap.sh}`, `docker/e2e/stands/{replica,vanilla}/*`, `Taskfile.yml` (`STAND`, `e2e:both`, `plugin:deploy`, `plugin:test`, `sandbox:provision`), `.github/workflows/ci.yml` (`plugin:test`), `internal/e2e/harness_test.go` (`E2E_STAND`, the env file per stand), `frontend/src/features/sites/{plugin-panel.tsx,plugin.ts}`, `frontend/src/copy/sites.ts`. The zip name, `application/sync` and `transport/wails/sync.go` keep `postulator-companion`.
- **Plugin.** The name kept; 1.4.0 with `Update URI`; no activation hook; `uninstall.php` removing the term clock and the preview tokens and keeping the SEO values; routes `GET /content/{id}/rendered`, `GET /meta/{id}`, `GET /terms/{id}`, `GET /settings`, and `GET /content` with `excerpt`, terms, every public post type and taxonomy; the manifest's new capabilities; a neutral preview query var; `normalize.php` compares host and port as `Site.Resolve` does.
- **Client.** A neutral default User-Agent with `wp.WithUserAgent` (1b wires the per-site setting), `plugin_outdated` by the manifest's version and capabilities, named in the Sites panel with the instruction to upload the new zip over the old one (WordPress offers "Replace current with uploaded"), `wp.Item` routed by `rest_base`, `_embed` terms, `/wp/v2/types`, `/wp/v2/settings`, `wp.Product` with price, SKU, stock, categories, `meta_data` as read-only facts, `ListProducts`, `ListVariations`, the store's settings.
- **Stands.** Per part 16.11; the suites in `internal/adapters/wp/e2e` run on both and their expectations come from a per-stand table. `task sandbox:provision STAND=replica|vanilla` provisions the owner's sandbox as either stand on explicit command; nothing else provisions it and only `sandbox:reset` wipes it.
- **Reuses.** `wp.Client`, `do.go`, `errors.go`, `Visit`, `media.go`, `probe.go`, the e2e harness shape. **Rewrites.** Nothing.
- **Exit.** `wp/e2e` green on both stands (`TestManifestDescribesTheSite` with the new capabilities, `TestRenderedContentCarriesTheInjectedBreadcrumbOnTheReplica`, `TestTermsListEveryTaxonomy` with the `entity` taxonomy on the replica, `TestSettingsNameTheTechnicalPages`, `TestMetaReadsOnlyTheKeysAsked`, `TestProductsListWithVariationsAndSKU`, `TestTheSameHrefIsClassifiedAlikeOnBothSides`, `TestTheFrontEndNamesNothingOfTheProgram`); `TestAnOutdatedPluginIsNamedByVersionAndCapabilities` in place of a test by namespace; `plugin_footprint_test.go`, `footprint_test.go` and `sitespecific_test.go` green; `task plugin:lint` and `task plugin:test` on 7.4; the contract test pins `openapi.yaml`. Owner sees: the Sites panel naming an outdated plugin, with the instruction to upload the new zip over it and the zip to save.

#### 1a tasks

Execution: `superpowers:executing-plans`, with two Opus subagents at most. Lefthook typechecks the whole package of each staged file, so only one writer works in an untagged Go package at a time; every commit names its paths and never uses `git add -A`. Edits go through Write and Edit, never a shell.

**Decided at the plan** (recorded in `DECISIONS.md` at the 1a close):
- **Plugin 1.4.0**, contract additive. `/content` still defaults to `page,post,product,product_cat`, so a released app keeps working; `types` also takes every public post type except `attachment`, and every public taxonomy whose name no public post type uses.
- **Default User-Agent** `Mozilla/5.0`.
- **`normalize.php` follows `Site.Resolve` on classification**: the host with its port, case-folded; the path escaped as `EscapedPath` does, then lowercased; an invalid escape, a dot segment or a control character drops the link. A relative href keeps PHP's base (the page's parent): both sides differ from RFC 3986 there, and 1b resolves render links against the document URL.
- **`/rendered` filters inside a singular main query.**
- **`/meta` refuses an `is_protected_meta` key** and takes at most 50 keys.
- **`/terms` covers every taxonomy of the post's type**, internal ones included.
- **Core and store settings**: a refusal on `/wp/v2/settings` (needs `manage_options`) or on `wc/v3/settings` (needs `manage_woocommerce`) is a `Forbidden` error the caller reads as unknown.
- **Products and variations stay on the store route.**
- **Compose defaults to the vanilla stand's project and port.**

**Slice recipe** (1a.5 to 1a.10, each one commit):
1. The e2e test, red on the vanilla stand (`task e2e:test STAND=vanilla -- -run <name>`).
2. The wptest unit test, red.
3. The client unit test, red.
4. `openapi.yaml`: the path, the schema and the capability enum entry. The contract test's `want` map and snippets gain them: red, then green.
5. wptest, then the client. Each new capability joins `wp.requiredCapabilities`, wptest's default capabilities and the PHP `CAPABILITIES`.
6. PHP; `task plugin:deploy STAND=vanilla` and `STAND=replica`; the e2e green on both.
7. `task plugin:lint`, then `go test ./internal/adapters/wp/...`, then commit.

The client-level checks against a stand live in `internal/adapters/wp/e2e/wpclient_test.go`, which drives a real `wp.Client` (`newWPClient(t) *wp.Client` from the env file).

##### 1a.1 — the two stands and the sandbox tasks (agent S)
- **Files.**
  - Modify `docker/e2e/compose.yaml`: `name: postulator-test-vanilla`, `${WP_PORT:-8088}`, `STAND` passed to bootstrap. Modify `docker/e2e/bootstrap.sh`: shared steps, then `stands/$STAND/provision.sh`. Test mode writes `.env.$STAND` (neutral site title, admin user and seeds `koffein`/`powder`); sandbox mode keeps `postulator` and `.env.sandbox`.
  - Create `docker/e2e/stands/vanilla/provision.sh`: a pinned default block theme, nested pages, a variable product with two SKU'd variations.
  - Create `docker/e2e/stands/replica/{provision.sh,site-behaviour.php,products.csv}`: Classic Editor, Blocksy and Rank Math pinned, Rank Math's breadcrumbs on. The mu-plugin is copied into `mu-plugins`:
    - it registers `entity` on `product` as a snippet would (public, no `show_in_rest`);
    - a hub page is one whose slug matches a `product_cat` or `entity` term;
    - on `the_content`, guarded by `is_singular() && in_the_loop() && is_main_query()`, it puts a `nav aria-label="Breadcrumb"` before the content (`hub-breadcrumbs`), with a trail along the category chain and one crumb that resolves to no page;
    - after the content it adds a `section` (`entity-auto-products`) with an `h2` and product cards;
    - in `wp_head` it prints BreadcrumbList, ItemList and CollectionPage JSON-LD for hubs.

    Hubs: one with an empty raw, one whose raw is a lone shortcode. Pages are flat. `products.csv` holds variable products and their variations, imported through `wp eval` and `WC_Product_CSV_Importer`.
  - Modify `Taskfile.yml`:
    - `STAND` (default `vanilla`) maps to `{8088, postulator-test-vanilla}` or `{8087, postulator-test-replica}`;
    - `e2e:up|down|reset|test|full|full:noplugin STAND=`, with `-- <args>` passed to `go test`;
    - `e2e:both`;
    - `plugin:deploy STAND=` (zip, then `wp plugin install --force --activate`);
    - `plugin:test` (Windows through `phplint`, linux native);
    - `sandbox:provision STAND=` (8089, `postulator-e2e`, explicit and idempotent, never `-v`).
  - Modify `.github/workflows/ci.yml` (the plugin job runs `task plugin:test`) and `.gitignore` (`docker/e2e/.env.*`).
  - Modify `internal/adapters/wp/e2e/harness_test.go`:
    - `environment` gains `stand`, `project` and `plugin`;
    - `envPath` reads `POSTULATOR_E2E_ENV`, else `docker/e2e/.env.$E2E_STAND`, else `.env.vanilla`;
    - `skipWithoutPlugin` fails when `plugin` is true;
    - adds `wpCLI(t, env, args...) string` through `docker compose -p <project> run --rm --no-deps -T --entrypoint wp bootstrap`.
  - Create `internal/adapters/wp/e2e/stands_test.go`: `type standFacts struct` and `func standOf(t *testing.T, env environment) standFacts`, holding the slugs and ids each later task asserts on (hub pages, product, variable product and SKUs, category chain, `entity` term) and the per-stand facts (SEO plugin, breadcrumb in content, `entity` present, `entity` embedded, nested pages, front page).
  - Modify `preview_test.go` (`expirePreview` through `wpCLI`), `commerce_test.go`, `raw_test.go`, `seometa_test.go`, `internal/e2e/harness_test.go` (the env file per stand) and `internal/e2e/degraded_test.go` (seed names).
  - Docs: CLAUDE.md's docker footgun and `docs/CONVENTIONS.md`'s "Docker end-to-end" become three projects.
- **Produces.** `.env.<stand>` keys `E2E_STAND E2E_PROJECT E2E_WP_URL E2E_WP_USER E2E_WP_APP_PASSWORD E2E_SEO E2E_WOO E2E_PLUGIN`; `wpCLI`; `standOf`; `task plugin:deploy`.
- **Steps.**
  1. `standOf` and the env discovery, so the suite fails without a stand.
  2. Compose and bootstrap for vanilla; `task e2e:up STAND=vanilla`; `task e2e:test STAND=vanilla` green.
  3. The replica; `task e2e:up STAND=replica`; `task e2e:test STAND=replica` green, with expectations moved into `standFacts` wherever a stand differs.
  4. `task e2e:full STAND=vanilla` and `e2e:full:noplugin` green.
  5. `sandbox:provision` proven on a scratch project (never on 8089).
  6. Commit `build(e2e): …` and `test(e2e): …`.

##### 1a.2 — a neutral User-Agent
- **Files.** `internal/adapters/wp/{option,client,do}.go`; tests `useragent_test.go`, `footprint_test.go`.
- **Interfaces.** `const DefaultUserAgent = "Mozilla/5.0"`; `func WithUserAgent(agent string) Option` (an empty agent keeps the default); `Client.userAgent`.
- **Tests.**
  - `TestTheDefaultUserAgentNamesNeitherTheProgramNorThePlugin`: the recorded `User-Agent` equals `DefaultUserAgent` and, lowercased, holds neither `postulator` nor `companion`.
  - `TestAUserAgentOverrideReachesEveryRequest`: `WithUserAgent("Agent/1")` reaches `Capabilities`, `ListItems` and an anonymous `Visit`.
  - `TestNothingTheClientSendsNamesTheProgram` drives every client method through wptest with neutral credentials. No recorded header, query value or body, and no `Upload.Filename`, `Title` or `Alt`, holds `postulator` in any case. Paths are exempt.
- Commit `feat(wp): a neutral User-Agent with an override`.

##### 1a.3 — one href table for both sides (review focus 4)
- **Files.**
  - Create `wp-plugin/testdata/hrefs.json` (`{"cases":[{"site","href","kind","path","relative"}]}`, with `kind` one of `path|same_document|external|unresolved`) and `wp-plugin/tests/normalize.php` (global shims for `ABSPATH`, `home_url` and `wp_parse_url`; it exits 1 naming each case that fails).
  - Create `internal/domain/pagemap/hrefs_test.go` and `internal/adapters/wp/wptest/hrefs_test.go`.
  - Modify `wp-plugin/postulator-companion/includes/normalize.php`: `site_authority()` replaces `site_host()`, `escaped_path()` and dot segments go in, and `internal_href_to_path` compares the authority. Modify `wptest/plugin.go` (`internalTarget`), `openapi.yaml` (`ContentLink.href`) and `contract_test.go`.
  - Create `internal/adapters/wp/e2e/hrefs_test.go`.
- **Cases.** The same host and port; another port; host case; `:80` written out; protocol-relative; a subdirectory base; root-relative and relative hrefs; raw UTF-8 and percent-encoded paths in both cases; an invalid escape; `..`; a control character; `#x`, `?q`, the empty href; `mailto:`, `tel:`, `javascript:`.
- **Tests.**
  - `TestResolveAnswersTheSharedHrefTable`: `pagemap.NewSite(site).Resolve(href)` gives `kind`, and the path whenever `relative` is false.
  - `TestTheFakeClassifiesTheSharedHrefTable`: wptest lists the link iff the kind is `path`, at the same path.
  - `normalize.php` on 7.4: the same rule as the fake.
  - `TestTheSameHrefIsClassifiedAlikeOnBothSides` (after 1a.1) puts a published page with one anchor per template (`{origin}`, the other stand's port, …) on each stand. The plugin's `/content` links equal `Site.Resolve(env.baseURL)` for every case.
- Commit `fix(plugin): normalize.php classifies an href as Site.Resolve does`.

##### 1a.4 — core reads
- **Files.** `internal/adapters/wp/{item,items,types}.go` (`types.go` new); `wptest/{core,state,server}.go`; tests `types_test.go`, `items_test.go`, `wptest/core_test.go`; e2e `core_test.go`, `wpclient_test.go`.
- **Interfaces.**
  - `type PostType struct{ Slug, Name, RestBase, RestNamespace string; Taxonomies []string; Hierarchical, Viewable bool }`; `func (c *Client) PostTypes(ctx context.Context) ([]PostType, error)` (cached like the manifest, reset by `InvalidateManifest`).
  - `ListItems` and `GetItem` route any type whose `rest_namespace` is `wp/v2` by its `rest_base`. `product` and `product_variation` are refused before any request; writes stay on pages and posts.
  - `ListQuery.EmbedTerms bool` sends `_embed=wp:term` and adds `_links,_embedded` to `_fields`. `Item.Terms []Term`, with `type Term struct{ Taxonomy, Name, Slug string; ID, Parent int64 }` (`Parent` is 0 from an embed).
  - `type SiteSettings struct{ ShowOnFront string; FrontPageID, PostsPageID int64 }`; `func (c *Client) SiteSettings(ctx context.Context) (SiteSettings, error)` (a refusal is `Forbidden`).
  - wptest: `Item.Terms []Term`; `WithPostType(PostType)`, `WithTaxonomy(Taxonomy{Name, RestBase string; ObjectTypes []string; ShowInREST bool})`, `WithoutSettingsAccess()`.
- **Tests.**
  - `TestPostTypesNameTheRestBaseOfEveryType`.
  - `TestACustomTypeIsListedByItsRestBase`.
  - `TestTheStoreTypesStayOnTheStoreRoute`.
  - `TestEmbeddedTermsArriveWithTheItem`: only `show_in_rest` taxonomies.
  - `TestSiteSettingsNameTheFrontAndPostsPages`.
  - `TestARefusedSettingsReadIsForbidden`.
  - e2e `TestCoreReadsTypesSettingsAndEmbeddedTerms`: the types include page, post and product; on the replica `entity` is absent from the embed (`standFacts.entityEmbedded`); `show_on_front` matches the stand.
- Commit `feat(wp): post types, routing by rest_base, embedded terms and the site settings`.

##### 1a.5 — 1.4.0, the outdated plugin and the Sites panel
- **Files.**
  - `wp-plugin/postulator-companion/postulator-companion.php`: `Version: 1.4.0`, `Update URI: false`, `VERSION`, `PREVIEW_QUERY_VAR = 'preview_token'`.
  - `openapi.yaml` (`info.version`, the preview var), `contract_test.go`.
  - `internal/adapters/wp/plugin.go`: `const PluginVersion = "1.4.0"`, `requiredCapabilities`, `func (c Capabilities) Missing() []string`, `func (c Capabilities) Outdated() bool` (an unreadable version counts as outdated). `pluginOutdated` carries `version`, `required` and the instruction.
  - `wptest/{server,plugin}.go`: `WithPluginVersion(string)`, the preview var.
  - `internal/domain/site/site.go`: `Extensions` gains `PluginOutdated bool` and `PluginRequired string`, both transient.
  - `internal/adapters/wp/registry/registry.go` (`pluginOf`).
  - `internal/application/sync/{views,service}.go`: `Plugin` gains `outdated` and `requiredVersion`.
  - `frontend/src/features/sites/{plugin-panel.tsx,plugin.ts}`, `frontend/src/copy/sites.ts` (`outdatedTitle`, `outdatedBody`, the four new capability labels).
  - Fixtures holding `postulator_preview`: `application/pages/preview_test.go`, `application/browser/browser_test.go`, `wp/plugin_test.go`, `wptest/plugin_test.go`, e2e `preview_test.go`.
- **Tests.**
  - `TestAnOutdatedPluginIsNamedByVersionAndCapabilities`, a table: `1.2.0` with six capabilities → outdated, naming what is missing; `1.4.0` with all → current; `1.4.0` missing one → outdated; `2.0.0` → current; `x` → outdated.
  - `TestProbeNamesAnOutdatedPlugin` (registry).
  - `TestCheckPluginSaysThePluginIsOutdated` (sync).
  - A panel vitest: an outdated plugin shows the update instruction and the zip button.
  - e2e `TestManifestDescribesTheSite`: 1.4.0 on both stands.
- Commit `feat(plugin): 1.4.0, a neutral preview var and an outdated plugin named in the Sites panel`.

##### 1a.6 — `GET /content/{id}/rendered`
- **Interfaces.** PHP `rendered_get` answers `{id,type,content,contentHash}`. `filtered_content(WP_Post)` runs `the_content` over `post_content` inside a one-post `WP_Query` set as `wp_query` and `wp_the_query`, after `the_post()`, then restores the globals. Go: `CapabilityRendered = "rendered"`; `type Rendered struct{ Type ItemType; Content, ContentHash string; ID int64 }`; `func (c *Client) GetRendered(ctx context.Context, itemType ItemType, id int64) (Rendered, error)` (the hash is checked as `GetRaw` checks it). wptest: `Item.Filtered string`, the raw when empty.
- **Tests.** e2e `TestRenderedContentCarriesTheInjectedBreadcrumbOnTheReplica`: the replica's hubs carry the `nav` with `aria-label="Breadcrumb"` and the section's `h2`; a vanilla page carries its `<p>` and no `nav`; 401, 404 and the hash. Units `TestGetRenderedReadsTheFilteredContent` and `TestGetRenderedRefusesAPluginWithoutTheCapability`.

##### 1a.7 — `GET /meta/{id}?keys=`
- **Interfaces.**
  - PHP `meta_get`: at most 50 keys. A protected key answers 400 `meta_key_denied` (`data.keys`), and an empty list `invalid_param`. The answer is `{id, meta:{key:value|null}}`, with values as stored.
  - Go: `CapabilityMeta = "meta"`; `func (c *Client) GetMeta(ctx context.Context, itemType ItemType, id int64, keys []string) (map[string]json.RawMessage, error)`. An absent key is left out; an empty list or more than 50 keys is refused before any request.
  - wptest: a key is protected when it starts with `_`.
- **Tests.** e2e `TestMetaReadsOnlyTheKeysAsked`: through `wpCLI`, `color`, `size` and `_secret` are set on a page. Asking `color` answers only `color`; `_secret` and a product's `_price` are refused; 51 keys are refused; an anonymous call gets 401.

##### 1a.8 — `GET /terms/{id}`
- **Interfaces.** PHP `terms_get` answers `{id, terms:[{taxonomy,id,name,slug,parent}]}` over `get_object_taxonomies($post->post_type)`, ordered by taxonomy then id. Go: `CapabilityTerms = "terms"`; `func (c *Client) GetTerms(ctx context.Context, itemType ItemType, id int64) ([]Term, error)`.
- **Tests.** e2e `TestTermsListEveryTaxonomy`: a product carries `product_cat` with its parent and `product_type` on both stands, and `entity` on the replica only; a post carries `category` and `post_tag`.

##### 1a.9 — `GET /settings`
- **Interfaces.** PHP `settings_get` answers `{showOnFront, pages:{front,posts,privacy,shop,cart,checkout,account,terms}}`, with 0 for none and `wc_get_page_id` used only when WooCommerce is loaded. Go: `CapabilitySettings = "settings"`; `type TechnicalPages struct{ Front, Posts, Privacy, Shop, Cart, Checkout, Account, Terms int64 }`; `type PluginSettings struct{ ShowOnFront string; Pages TechnicalPages }`; `func (c *Client) PluginSettings(ctx context.Context) (PluginSettings, error)`.
- **Tests.** e2e `TestSettingsNameTheTechnicalPages`: privacy, shop, cart, checkout and account are non-zero, the shop's page has the slug `shop`, and the front page matches `standFacts`.

##### 1a.10 — `/content` with excerpt, terms and every public type
- **Interfaces.** PHP: the items gain `excerpt` and `terms` (a term item answers `''` and `[]`). `parse_types` accepts the public post types and taxonomies, and the default stays as it was. The term phase reads `tt.taxonomy IN (…)`, and `touch_term` covers every public taxonomy. The manifest gains `postTypes` and `taxonomies`. Go: `ContentItem` gains `Excerpt` and `Terms`; `Manifest` and `Capabilities` gain `PostTypes` and `Taxonomies`; `postRoute` accepts an advertised post type, refuses a taxonomy, and keeps the closed set when the manifest lists no types.
- **Tests.** e2e `TestContentCarriesExcerptTermsAndEveryPublicType`: a post's excerpt and terms are listed; `types=entity` lists the replica's terms; `types=product` carries `entity` on the replica; an unknown type answers 400; the default listing holds no `entity` item.

##### 1a.11 — store reads (agent W, after 1a.10 is committed)
- **Files.** `internal/adapters/wp/woo.go`, `woo_test.go`, `wptest/woo.go`, `wptest/woo_test.go`, e2e `store_test.go`.
- **Interfaces.**
  - `Product` gains `Price, RegularPrice, SalePrice, SKU, StockStatus string; ManageStock bool; StockQuantity *int64; Categories []ProductCategory; MetaData []MetaEntry; VariationIDs []int64`.
  - `type ProductCategory struct{ Name, Slug string; ID int64 }`; `type MetaEntry struct{ Key string; Value json.RawMessage; ID int64 }`.
  - `type Variation struct{ Modified time.Time; Description, Permalink, Status, SKU, Price, RegularPrice, SalePrice, StockStatus string; Attributes []VariationAttribute; MetaData []MetaEntry; StockQuantity *int64; ID, ParentID int64; ManageStock bool }`; `type VariationAttribute struct{ Name, Option string; ID int64 }`.
  - `type ProductQuery struct{ ModifiedAfter *time.Time; Status, Type string; Page, PerPage int }`; `func (c *Client) ListProducts(ctx context.Context, query ProductQuery) (Page[Product], error)`; `func (c *Client) ListVariations(ctx context.Context, productID int64, page, perPage int) (Page[Variation], error)`.
  - `type StorePages struct{ Shop, Cart, Checkout, Account, Terms int64 }`; `func (c *Client) StoreSettings(ctx context.Context) (StorePages, error)` reads `settings/products/woocommerce_shop_page_id` and `settings/advanced`; a refusal is `Forbidden`.
  - `UpdateProduct` is unchanged.
- **Tests.**
  - `TestListProductsWalksTheStoreWithItsFacts`.
  - `TestListVariationsCarriesSKUPriceStockAndAttributes`.
  - `TestStoreSettingsNameTheShopPages`.
  - `TestTheStoreFactsAreNeverSent`: an update payload holds none of price, SKU, stock or categories.
  - e2e `TestProductsListWithVariationsAndSKU` and `TestStoreSettingsNameTheShopPages` on both stands.
- Commit `feat(wp): the store's products, variations and pages, read only`.

##### 1a.12 — uninstall
- **Files.** Create `wp-plugin/postulator-companion/uninstall.php`. It exits unless `WP_UNINSTALL_PLUGIN` is defined, then calls `delete_metadata` for every object on `_postulator_modified` (term), `_postulator_preview_hash` and `_postulator_preview_expires` (post). It keeps `_postulator_seo_*`. Create e2e `uninstall_test.go`.
- **Tests.** `TestUninstallRemovesTheBookkeepingAndKeepsTheSEOMeta` sets a preview link, a term stamp and SEO meta (the none keys on vanilla, Rank Math's on the replica), then runs `wpCLI plugin uninstall postulator-companion --deactivate`. The bookkeeping must be gone and the SEO meta still there. An unconditional `t.Cleanup` reinstalls from `/dist`.
- Commit `feat(plugin): uninstall removes the bookkeeping and keeps the SEO values`.

##### 1a.13 — footprint and site-specific scans
- **Files.**
  - Create `wp-plugin/plugin_footprint_test.go` (package `wpplugin_test`): every line of the plugin tree naming `postulator` in any case must consist of the header's `Plugin Name: Postulator Companion`, `namespace Postulator\Companion;`, `'postulator/v1'` and `'_postulator_…'` keys only.
  - Create e2e `footprint_test.go`.
  - Create `internal/app/sitespecific_test.go`. Its needles are the `class` tokens of `docker/e2e/stands/replica/site-behaviour.php` and the theme and plugin slugs that `stands/replica/provision.sh` installs; the scan covers `internal/`, `cmd/` and `wp-plugin/`, excluding `testdata/`. The CSS-selector rule lands with `domain/render` in 1b.
  - Modify `internal/app/deps_test.go`: `TestTheFakeSiteIsIndependentOfTheClient` (wptest's transitive dependencies exclude `internal/adapters/wp`).
- **Tests.** `TestThePluginNamesTheProgramOnlyWhereTheAdminSeesIt`; e2e `TestTheFrontEndNamesNothingOfTheProgram`: anonymous reads of `/`, a published page, a product, a product category, a 404, a draft's preview link, `/feed/` and `/wp-sitemap.xml`, whose bodies and headers must not hold `postulator`; `TestNothingSiteSpecificLivesInTheSources`; the deps test.
- Commit `test: the visitor-facing footprint, the site-specific scan and the fake's independence`.

##### 1a.14 — the gate
- Run the whole Verification list of the session plan:
  - wp/e2e on both stands, and `e2e:full` and `noplugin` on vanilla;
  - `plugin:lint` and `plugin:test`;
  - `task build`, `go vet` (plus `-tags e2e`), `golangci-lint run`, `task lint:e2e` alone;
  - `go test -race -count=1 -p 2 -covermode=atomic -coverprofile=coverage.out ./...` and `go run ./cmd/covergate`;
  - the comment check, `gofmt -l .`;
  - `npm run typecheck`, `npm run test:run`, `task ui:lint`.
- Update `STATUS.md` (1a done, 1b next, the client methods waiting for their 1b callers), `ARCHITECTURE.md`'s plugin section, and `DECISIONS.md` (the decisions above).
- Commit `docs: stage 1a's gate is green`.

### 1b — the model, the migrations, the observation

- **Touches.** `internal/domain/pagemap` (`Page` fields, `DeriveParent`, `MatchRow` read side), `internal/domain/field` (catalog, `Denied`), `internal/domain/block` (`Parse`, `Serialize`, hashes), `internal/domain/render` (`Extract`), `internal/application/observe` (readers, stages), `internal/runtime/steps/sync_*.go` (stages inside `sync_site`, `since`, the full-walk rule), `internal/runtime/steps/{publish,publish_parent,repair_hierarchy,preflight}.go`, `internal/application/runs/start.go`, `internal/domain/run/recipes.go`, `internal/adapters/sqlite/migrations/0045…` (identity, terms, fields, bodies, blocks written, renders, typed links rebuild, findings, `sites.settings`, the `pages` columns and the `wp_type` rebuild), the repositories, `internal/transport/wails/vocabgen`, `internal/e2e/*`.
- **Migrations.** One concern each, every down restoring exactly: 0045 `page_identity`; 0046 `page_terms`; 0047 `page_fields`; 0048 `page_bodies` + `page_blocks_written`; 0049 `page_renders`; 0050 `page_links` rebuilt with type, region, source, position, ours (a table rebuild, the data copied as `contextual`/`raw`); 0051 `findings`; 0052 `sites.settings`; 0053 `pages` gains `page_type_id`, `parent_reason`, `technical_reason`, `read_only_reason` (plain columns, no FK); 0054 the `pages.wp_type` CHECK rebuild. The old `pages` columns stay.
- **Behaviour.** `repair_hierarchy` deleted and `RepairRecipe` gone; `publish` sends no slug and no parent on an update, `compare` reads the identity snapshot, `start.parentOf` accepts a derived parent; the one creation call of a planned page keeps sending status, parent and slug (the owner's answer 1).
- **Wiring 1a leaves to 1b.** The per-site User-Agent override in `sites.settings` (0052) reaches the registry's `wp.WithUserAgent`; the transient outdated flag on `site.Extensions` becomes the `plugin_outdated` finding; the client methods 1a adds (`GetRendered`, `GetMeta`, `GetTerms`, `PluginSettings`, `PostTypes`, `SiteSettings`, `ListProducts`, `ListVariations`, `StoreSettings`) get their callers in the observation; `sitespecific_test.go` gains the CSS-selector rule over `domain/render`; the render extraction resolves a relative href against the document URL.
- **Reuses.** `content.Document`, `shown.go`'s word compare, `pagemap.Site`, `Index`, `BuildTree`, `sync_archive`, the run engine and the checkpoint pattern of `sync_site`. **Rewrites.** `sync_reconcile.go` (findings, not silent merge), `sync_settle.go` (derivation with reasons), `page_link_repo.go`.
- **Exit.** Unit: block parser goldens (classic with blank lines, classic with `<p>` runs and `<br>` lines, block-editor with attributes, shortcode-only); extraction goldens from both stands' saved renders (an empty-raw hub, a shortcode-only page, a texturised paragraph, a page with two BreadcrumbLists); `DeriveParent` table over self-last, Home-first, unresolved-crumb, two-lists-in-a-nav, term-parent; `field.Denied` enumerated; `publish` update sends no slug or parent for a page whose observed slug differs from its path; round trip of every migration. E2E on both stands: `TestTheSiteIsObservedIntoTheModel` (every published page has identity, fields, body and render rows; the parent reason per the stand table; regions typed per the table; shop, cart and checkout technical; a second observation is incremental and changes nothing; a generate run of a planned child still waits for its parent and publishes). Owner sees: nothing new on screen yet; the model fills behind the Pages screen.

### 1c — the screens

- **Touches.** `frontend/src/features/pages/{drawer,details,summary,links}.tsx` and new model panes (identity, hierarchy with reason, regions, trail, typed links, technical and read-only reasons, blocks read-only), `frontend/src/features/links/*` (counts by type, contextual owed), `frontend/src/features/sites/*`, `frontend/src/generated/vocab.ts` (`linkTypes`, `regionKinds`, `parentReasons`, `technicalReasons`, `contentRootStates`), `internal/application/pages/views.go`, `internal/transport/wails/pages.go`, the UI harness fixtures.
- **Exit.** `npm run test:run` and `task ui:lint` green; the harness walk shows a page as the model. Owner sees: every page as the model, Linking by type.

## Stage 2 — page types, the field catalog and scope with UI, the context pack

- **Goal.** The client names his page types over presets, enables fields per type with an instruction, limits and a link policy, and the writer, the linker and the judge work from the context pack.
- **Touches.** `internal/domain/pagetype`, `internal/domain/field` (scope), `internal/domain/contextpack`, `internal/application/{pagetypes,fields}`, `internal/application/templates/policies.go` (deleted) and the four copies of the rules merge (one resolver), `internal/domain/content/{brief,draft,product,sanitize,linkcontext}.go`, `internal/runtime/steps/{generate_body,generate_meta,generate_fields,repair_links,insert_links,judge,ceiling,prompts}.go` and `prompts/*.tmpl`, `internal/application/content/*`, `internal/runtime/budget.go`, `internal/application/imports/{kinds,pages,export}.go`, `internal/application/graph/*` (kind removed, entity card), `internal/application/tools/{pagetypes_*,fields_*,pages_context_pack,templates_spec,graph_*}.go`, `policies_*.go` deleted, `internal/transport/wails/vocabgen`, `frontend/src/features/settings` (Page types), `frontend/src/features/templates` (policies screen deleted, editor without `pageKind`), `frontend/src/features/graph` (type of the canonical page), `frontend/src/features/pages/template.tsx`.
- **Migrations.** 0055 `page_types`; 0056 `field_scopes`; 0057 the backfill of `pages.page_type_id` (data only, empty down); 0058 `entities.definition`, `facts`, `notes`; 0059 `link_policies` dropped and `sites.default_link_policy_id` removed by rebuild (down restores both); 0060 `entities.kind` dropped by the 0030 rebuild and `templates.page_kind` dropped (down restores both).
- **Steps.** `generate_fields` joins `StepName`; `generate_body` and `generate_meta` keep their names and consume the pack; the writer leaves the client's own paragraphs alone unless the scope's `rewrites` is on for the type and field (off by default), and a run may switch it on for chosen pages with the user's confirmation (the owner's answer 5); `insert_links` and `repair_links` take the field's policy; the ceiling gains the input budget `llm.inputBudget.<role>`.
- **Tools.** `pagetypes_list/get/create/update/delete`, `fields_catalog`, `fields_set_scope`, `pages_context_pack`; the `kind` field leaves the five graph request structs; `pageKind` and `linkRules` leave the templates trio; the ceiling moves in each commit.
- **Reuses.** `template.Resolve` and the override layers, `Structure`, `Assemble`, `InsertTarget`, `pagemap.Keywords`, the import pipeline. **Rewrites.** `content.Brief` (the pack's writer view), the prompt templates, `imports/kinds.go`, `features/graph/labels.ts`.
- **Exit.** Unit: pack builder priorities and `Fit` goldens (a pack over budget drops siblings before the trail, products before keywords); the backfill round trip over a fixture database with every kind; the import's kind cell to type by name and synonym; the single rules resolver equals the four old answers over the old fixtures; `vocab.ts` loses `entityKinds` and the generator test is green. E2E on both stands: `TestTheWriterSeesTheFullPicture` (a generate run over a planned child on each stand; the prompt the fake provider received carries the trail, the listed products on the replica's hub, the field's instruction and policy, and the current title of an existing page). Owner sees: Settings → Page types with his names, per-field switches and instructions, and a "what the writer will see" pane per page and field.

## Stage 3 — editing: block operations, field writes through the registry, the pipeline over existing pages, revert

- **Goal.** A run edits an existing page field by field and block by block, writes under the hash of what it read, never touches the deny list, and a revert puts back exactly what each field and block held.
- **Touches.** `internal/domain/block` (operations), `internal/domain/field` (write side), `internal/application/edit` (the writers `CoreWriter`, `StoreWriter`, `PluginWriter` declared here; the field write use case), `internal/runtime/steps/{publish,publish_product,publish_seo,relink,relink_page,relink_neighbors,revert,revert_product,generate_images}.go`, `internal/adapters/wp/{items,woo,plugin,media}.go` (writers: excerpt, featured media, meta by key, upsells and cross-sells, image alt and title, variation description, the raw PUT reading the stored content), `wp-plugin/postulator-companion` (`PUT /meta/{id}`, `meta_write`), `wp-plugin/openapi.yaml`, migrations 0061 (the old `pages` columns dropped, down restores) and 0062 (`page_links.origin` dropped), `internal/domain/pagemap/observed.go` deleted, `internal/application/tools/pages_replace_links.go` deleted, `frontend/src/features/runs/panes/*` (per-field diffs), `frontend/src/features/pages` (blocks editable through a run, revert per field).
- **Behaviour.** `publish` creates a planned page once (status, parent and slug in that one call, recorded in the artifact: the owner's answer 1) and otherwise writes every produced field through the registry under a per-field checkpoint; `publish_product` folds into the registry writers (the store's fields first, the description last, the hash read after, as today); relink places links with `PlaceLink` on blocks; revert restores per field and per block (neighbours by block hash); media names stay neutral.
- **Reuses.** The raw route's CAS, `InsertTarget`, the publish artifact shape extended per field, the revert run kind, `generate_images`. **Rewrites.** `publish.go` as the field writer, `revert.go` at field granularity.
- **Exit.** Unit: `Serialize` byte-exact for untouched blocks (a property test over random bodies and operations); every operation on classic and block bodies, a `wp:paragraph` with `align` kept valid; the deny list refused at the use case and at the adapter; a recording transport proves no slug, parent, status, price, stock, SKU, category or variation attribute leaves on an update; a publish killed after its second field resumes at the third with no second model call; the PHP deny set in `openapi.yaml` includes `field.Denied`'s keys. E2E on both stands: `TestAnExistingPageIsEditedBlockByBlock` (a hub with an injected breadcrumb and grid on the replica gets its body between them, untouched blocks identical, links placed only in content, a revert restores the base; on vanilla a nested page's body is edited and its URL and parent unchanged); `TestAProductIsEditedThroughTheRegistry` (description, short description, a `meta_data` key, a variation description written and reverted; price, SKU, stock and categories untouched, proven on the wire); `TestAWriteAgainstAStaleHashPauses`. Owner sees: a run that edits existing pages, the diff per field, revert per field.

## Stage 4 — reconciliation with the workbook, findings, the site map, agent tools over the model

- **Goal.** A workbook reconciles with the site through persistent findings a person or the agent resolves; the site is one map; the agent works the whole loop through tools over the model.
- **Touches.** `internal/domain/finding`, `internal/application/findings` (`List`, `Resolve`), `internal/domain/pagemap/match.go` (`MatchRow`), `internal/application/imports/{state,products,match,pages,views,summary}.go` (the matcher, findings that survive apply, `AnchorSeparator`'s description corrected), `internal/application/observe` (discrepancies as findings: `body_changed_on_site`, `field_changed_on_site`, `parent_differs`), `internal/application/graph` (entity card proposals through findings), `internal/application/tools/{pages_get,pages_fields_*,pages_blocks_*,findings_*,sites_update,templates_resolve_for_page}.go` and `groups.go`, `internal/application/reports/*` (site map read), `frontend/src/features/{findings,sitemap,imports}/*`, `frontend/src/features/agent/cards`.
- **Migrations.** None new beyond what stage 1 made; `findings` gains no column (its `subject` JSON carries the per-code shape).
- **Tools.** `pages_get` answers the model; `pages_fields_get/set` and `pages_blocks_list/replace/insert_after/delete/place_link` (writes confirmable); `findings_list`, `findings_resolve`; `sites_update` carries the per-site settings; `templates_resolve_for_page` folds into `pages_get`; the deferred groups list and the ceiling refreshed.
- **Reuses.** The import pipeline and its preview, `graph.Key`, the UI harness. **Rewrites.** `siteState.held` and `products.go`'s matcher into `MatchRow`; the import preview's row view (the match rule per row).
- **Exit.** Unit: `MatchRow` table over the shapes of `samples/*.xlsx` (URL, slug, name, title, H1, SKU, ambiguous); findings lifecycle (observe twice, dismiss, observe again: one row); tool schema sizes under the ceiling with the deferred round logged; `imports_preview` names the rule each row matched by. E2E on both stands: `TestTheWorkbookReconcilesWithTheSite` (`samples/client-sheets.xlsx` against the replica: rows matched by URL, slug and name, mismatches as findings, resolved through `findings_resolve` by the agent on the fake provider; on vanilla the nested URLs match by slug). Owner sees: the Findings inbox with options, the Site map (hierarchy by reason, types, technical and read-only pages, link counts by type), the import preview naming each row's match rule, and the agent working the loop.

## Out of scope

Writing a slug, a parent or a status to an existing page; prices, stock, SKU, taxonomies, a variation's attributes; theme and plugin options; a page builder's content (read-only, marked); creating products or terms; multisite; a second model provider; rank tracking; running docker in CI (the stands run by hand before a tag, as today).

## Open questions (answered 2026-10-17; `DECISIONS.md`, "the owner's answers")

1. May the one creation call of a planned page set status, parent and slug? **Yes, once, recorded in the artifact; never on an update.**
2. The type lives on the page and an entity reads its canonical page's; topic and custom map to Entity? **Yes.**
3. The plugin's name and meta prefix? **The plugin keeps its name**: `postulator-companion`, `postulator/v1`, the zip, the text domain and the meta and option prefixes stay. Zero footprint covers only what a visitor receives.
4. May a `product_cat` term page be a parent, and may the replica mu-plugin carry the client's class names as a fixture? **Yes and yes.**
5. May the writer rewrite the client's own paragraphs? **Not by default**: a switch per type and field, off; a run may switch it on for chosen pages with the user's confirmation.
6. The User-Agent? **Neutral, naming neither the program nor the plugin, with a per-site override** for CDNs and security plugins that block unknown agents.
7. The owner's sandbox on 8089? **The owner's**: never wiped or provisioned implicitly; explicit tasks provision it as the replica or the vanilla stand. The suites still refuse 8089.
8. (At the 1a plan.) The activation hook? **None**: its only job was the rename's migration, and an upload over an installed plugin never fires it.
9. (At the 1a plan.) What does uninstall remove? **The bookkeeping** (term clock, preview tokens); the SEO values stay.
