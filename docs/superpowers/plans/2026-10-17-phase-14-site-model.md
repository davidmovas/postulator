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

#### 1b tasks

**Status, 2026-10-18: 1b.1 to 1b.13 done**, the commits and the stand numbers in `STATUS.md`,
the rulings in `DECISIONS.md` under stage 1b. Two tasks landed differently from their text: the
five `sync_*` helper steps were deleted rather than rewritten, the observation owning their
work inside `sync_site` (1b.9), and the e2e expectation table lives in
`internal/e2e/observe_test.go` rather than `wp/e2e`'s `standFacts`, which the application
harness cannot import (1b.12).

Execution: `superpowers:executing-plans`, one accountable owner, at most two agents at once and only on disjoint Go packages (M: `adapters/sqlite`; R: `domain/render`, then `wp-plugin`, `adapters/wp`, `wptest`; W: `application/sites`, `adapters/wp/registry`, `tools/sites_update`, `internal/app` tests). Lefthook lints the whole package of every staged file, so every commit names its paths and never uses `git add -A`; edits go through Write and Edit; every gate command runs under `set -o pipefail` and its exit code is read. A test that pins an old value moves in the same commit.

**Decided at the plan** (recorded in `DECISIONS.md` at the 1b close; the spec is amended in the same commits):
1. **A relative href resolves against the document**, as RFC 3986 and a browser do, on all three sides: `Site.ResolveAt(document, href)`, `Resolve(href) = ResolveAt(base, href)`; PHP passes the permalink's path up to its last slash; wptest's `internalTarget` takes the document. `hrefs.json` gains `document` and relative cases. The `ContentLink.href` description changes, so the plugin is **1.4.1** and `wp.PluginVersion` moves with it.
2. **A breadcrumb nav** is a `nav` whose `aria-label` contains "breadcrumb" (case-insensitive) or whose `itemtype` names `BreadcrumbList` (the replica's theme marks its breadcrumb with microdata only). Its trail is the first `ol`/`ul` with at least two items, else the nav's links in document order plus a trailing non-link text as the current page (WooCommerce's slash-separated nav). Spec 16.5 amended.
3. **JSON-LD** is read from every `ld+json` script, through `@graph`; `@type` is a string or an array; a `ListItem` names its crumb by `item` as a string, by `item.@id`, `item.url` and `item.name`, or by its own `name` and `url`; `position` is a number or a string; an `@id` without a trailing slash is `/`. Several `BreadcrumbList` objects: the most resolving crumbs win, a tie goes to the one equal to the nav's trail, else the first.
4. **`noindex` is read from the render's `<meta name="robots">`**; the plugin's SEO meta carries no robots value.
5. **The store's facts are denied catalog fields in `page_fields`** (`price`, `regular_price`, `sale_price`, `sku`, `stock_status`, `stock_quantity`, `manage_stock`, `attributes` as JSON, `variation:<id>` as JSON); a variation's description is `variation_description:<id>`; product categories are `page_terms` rows. `field_changed_on_site` is raised for writable fields only. `Variation.ManageStock` decodes `true`, `false` and `"parent"`.
6. **Base.** The first observation sets base = current. Later ones move current and keep base; a writable field whose hashes differ is `field_changed_on_site`, a body `body_changed_on_site`. `publish`, `publish_product`, `relink_page`, `revert` and `sync_back` record what they wrote as current and base through one helper, so our own writes raise nothing. The observation resolves an open finding whose condition is gone (`observed_clear`, by `observation`); a dismissed row reopens only when its subject JSON changes.
7. **The parent** is derived for pages on the site only; a planned page keeps the workbook's parent under the new reason `workbook` (spec 16.1's enum gains it; `none` would lie). Order: JSON-LD → nav → the WordPress parent (a term's parent term) → `Index.PathParentID` (the front page included, as the spec says) → none. A nearest crumb that resolves to nothing is `parent_unresolved` and the derivation falls to the next source. A workbook parent the derivation disagrees with: the derived one stands and `parent_differs` is raised. `BuildTree` nests by `ParentPageID`, falls back to the path ancestor and guards a cycle.
8. **Technical reasons**, in order: the plugin's `/settings` (else core `SiteSettings` and `StoreSettings`) ids; `woocommerce_cart`, `woocommerce_checkout` and `woocommerce_my_account` shortcodes and the `woocommerce/cart` and `woocommerce/checkout` blocks (a settings default) → `shortcode`; the robots meta → `noindex`. `workbook` is set by the import (`own_entity: no`); `manual` gets its producer in stage 4; the observation never overwrites either. `ReadOnlyReason` is `builder:<name>` from a settings default (data attributes, class prefixes, shortcode prefixes) or `term`.
9. **Scale.** `settings.observe.renderConcurrency` defaults to **4**: a shared PHP-FPM host answers a few anonymous requests at once without reading as a crawler burst, and 2,000 pages take about four minutes. `fullWalkEvery` defaults to **10**. `since` is the previous observation's start minus one second, the plugin's bound being exclusive on seconds *(amended at the whole-branch review: `since` is taken from the site's own stamps, the newest `modified` the previous walk saw minus that walk's duration and a second, because the plugin compares it with the server's clock)*. A render is fetched for a published page whose identity or body changed, for every page on a full walk, or when the run param `renderRescan` is set; a draft only through the plugin's preview link. The client's own retries and rate limit are the backoff; a visit that still fails is `render_unreachable` (spec 16.8 gains the code) and the old render row stays. Core without the plugin always walks fully; the archive runs after a full walk only.
10. **Observed types** are the viewable post types of `/wp/v2/types` (intersected with the manifest's `postTypes` when the plugin is on; `attachment` and `product_variation` excluded) plus `product_cat`, passed to `/content` explicitly. Every other taxonomy is read as terms on posts (the plugin's `/terms/{id}` when present, else `_embed`), never as pages.
11. **The `repair` kind** keeps its word in the `CHECK` and in `runKinds`; `RepairRecipe` and `repair_hierarchy` go; `recipeFor` refuses a repair start as it refuses a revert; the `runs_start` enum drops `repair`; the frontend keeps the label for old runs and no longer offers the kind.
12. **The migrations are reordered** so the new tables reference the rebuilt `pages`: 0045 rebuilds `pages` with `wp_type <> ''`, 0046 adds the four columns, 0047 identity, 0048 terms, 0049 fields, 0050 bodies and blocks written, 0051 renders, 0052 the links rebuild, 0053 findings, 0054 `sites.settings`.
13. **Layering.** `application/observe` declares its readers over its own item types and `runtime/steps/observe_readers.go` adapts `*wp.Client`; the step drives the stages through its checkpoint, one batch per execution. `observe.Service.Summary` and `.Page` are the reads the e2e test and 1c use; no Wails binding in 1b.

##### 1b.1 — `domain/block`
- **Files.** Create `internal/domain/block/{block.go,parse.go,classic.go,editor.go,serialize.go,snapshot.go}`, tests `parse_test.go`, `serialize_test.go`, goldens under `testdata/`.
- **Interfaces.** `type Kind string` (`Classic`, `Blocks`); `type BlockKind string` (`Paragraph`, `Heading`, `List`, `Quote`, `HTML`, `Shortcode`, `WP`, `Foreign`); `type Block struct{ Index int; Kind BlockKind; Name string; Raw string; Hash string; Attributes json.RawMessage; Dynamic bool }`; `type Body struct{ Kind Kind; Blocks []Block }`; `func Parse(raw string) Body`; `func Hash(raw string) string` (hex sha256 over the trimmed raw with `\r\n` folded to `\n`; a block's `Hash` is `Hash(block.Raw)`); `func (b Body) Serialize() string` (the bytes parsed, exactly); `Hash()`, `Text()` (`content.TextOf` over the non-dynamic blocks), `TextBlocks() []string`, `Empty()`, `DynamicOnly()` (not empty and every block a shortcode or a dynamic wp block), `Shortcodes() []string`, `BlockNames() []string`. `type Side string` (`Current`, `Base`); `type Snapshot struct{ PageID string; Side Side; Raw, Hash string; Kind Kind; ObservedAt time.Time }`; `type Written struct{ PageID, BlockHash, RunID string; WrittenAt time.Time }`.
- **Rules.** Block-editor content when the raw holds `<!-- wp:`; a top-level `<!-- wp:name {json} -->…<!-- /wp:name -->` is one block with nested blocks inside it, a self-closing `<!-- wp:name /-->` is dynamic; `core/paragraph|heading|list|quote|html` take their kinds, any other name is `WP`. Classic content: a blank line or a top-level element of `content`'s `blockElements` starts a block; `p` paragraph, `h1`–`h6` heading, `ul`/`ol` list, `blockquote` quote, a bare text run paragraph, a top-level `[tag …]…[/tag]` or `[tag …]` shortcode, any other element one foreign block. A block's `Raw` runs to the byte before the next block's first byte.
- **Tests.** `TestParseSplitsClassicContentOnBlankLinesAndBlockElements` (the replica's Home and TB-500 bodies, `<p>` runs with `<br>`, a `<div>` wrapper, a list), `TestParseReadsBlockEditorDelimitersAndAttributes` (the vanilla bodies, `{"level":2}`, a nested `core/group`, a self-closing `woocommerce/cart`), `TestParseKeepsAShortcodeAtomic` (`[products …]`: one dynamic block, `DynamicOnly`, `Shortcodes` = `products`), `TestSerializeReturnsTheBytesParsed` (every golden and random mixes), `TestHashIsStableAcrossLineEndingsAndOuterWhitespace`, `TestAnEmptyBodyIsEmptyAndNotDynamicOnly`.
- Commit `feat(block): the body parsed into addressable blocks, serialised byte for byte`.

##### 1b.2 — `domain/pagemap`
- **Files.** Modify `page.go`, `index.go`, `tree.go`, `path.go`; create `parent.go`, `identity.go`; tests `parent_test.go`, `tree_test.go`, `hrefs_test.go`, `page_test.go`, `index_test.go`.
- **Interfaces.** `type ParentReason string` (`jsonld_breadcrumb`, `aria_breadcrumb`, `wp_parent`, `url`, `workbook`, `none`); `type TechnicalReason string` (`front_page`, `posts_page`, `privacy`, `shop`, `cart`, `checkout`, `account`, `terms`, `shortcode`, `noindex`, `workbook`, `manual`, `""`) with `Valid()` and `Observed()` (false for `workbook` and `manual`); `type ReadOnlyReason string` (`term`, `""`) and `func ReadOnlyBuilder(name string) ReadOnlyReason` (`builder:<name>`). `Page` gains `PageTypeID *string`, `ParentReason`, `TechnicalReason`, `ReadOnlyReason`. `WPType.Valid()` accepts `^[a-z0-9_-]+$`; `Term()` stays `product_cat`; `core()` becomes "neither a term nor a product". `NewPage` keeps a non-empty `Slug` and derives it from the path only when empty. `PageLink` gains `Type LinkType` (`breadcrumb`, `listing`, `navigation`, `contextual`, `unknown`), `Region string`, `Source LinkSource` (`raw`, `render`), `Position *int`, `Ours bool`; `NewPageLink` refuses an unknown type or source. `type Crumb struct{ Name, Path string }` (`Path` empty for an external or unresolved crumb); `type TrailSource string` (`jsonld`, `nav`); `type Trail struct{ Source TrailSource; Crumbs []Crumb }`; `type Derivation struct{ ParentID *string; Reason ParentReason; Trail *Trail; Unresolved []Crumb }`; `func DeriveParent(page Page, trails []Trail, wpParentID *int64, index Index) Derivation`; `func (i Index) ByWordPressID(family WPType, id int64) (Page, bool)` (posts and terms are two id spaces); `func (s Site) ResolveAt(document, href string) (string, LinkKind)`; `type Identity struct{ PageID, URL, Status string; WPType WPType; WPParentID *int64; ModifiedGMT *time.Time; ObservedAt time.Time }`; `type Term struct{ Taxonomy string; TermID int64; Name, Slug string; ParentTermID int64 }`. `BuildTree` nests by `ParentPageID` when it names a page of the set, else by the nearest path ancestor; a cycle makes a root.
- **Tests.** `TestDeriveParentTakesTheNearestResolvingCrumb` (self last; Home first; an unresolved crumb named and the next source taken; two JSON-LD lists decided by count; a tie decided by the nav; a term's parent term; the URL fallback with the front page; a product with no trail → none), `TestBuildTreeFollowsTheDerivedParent` (flat URLs under breadcrumb parents; a cycle), `TestResolveAtAnswersTheSharedHrefTable` (a `document` column when present, plus `/x/y/`+`a/`, `/x/y`+`a/`, `../`, `?q`), `TestNewPageKeepsAnObservedSlug`, `TestByWordPressIDTellsPostsFromTerms`.
- Commit `feat(pagemap): the derived parent with its reason, typed links and the document base`.

##### 1b.3 — `domain/field`, `domain/finding`, `domain/site` settings
- **field.** `type Key string` with the constants `title excerpt body seo_title seo_description seo_canonical product_description product_short_description variation_description image_alt image_title upsells cross_sells meta slug status parent price regular_price sale_price sku stock_status stock_quantity manage_stock attributes variation terms`; `type Source string` (`core store plugin render none`); `type Spec struct{ Key Key; Instanced bool; Read, Write Source; Denied, Text bool }`; `Catalog() []Spec`; `Lookup(key Key) (Spec, bool)`; `Instance(kind Key, instance string) Key` (`kind:instance`); `Key.Kind()`, `Key.Instance()`; `Denied() []Key`; `type Value struct{ PageID string; Key Key; Current, CurrentHash, Base, BaseHash string; ObservedAt time.Time; WrittenAt *time.Time; WrittenRunID *string }`; `Hash(value string) string` (empty for empty); `NewValue(pageID, key, current, at)` (base = current); `Value.Changed()`.
- **finding.** `type Code string` (`parent_unresolved parent_differs plugin_outdated field_changed_on_site body_changed_on_site render_unreachable builder_detected technical_detected page_title_differs page_h1_differs url_form_mixed`); `Severity` (`info warning error`); `Status` (`open resolved dismissed`); `Option` (`ignore accept_site accept_workbook keep_both clear_technical update_plugin rescan`); `type Finding struct{ ID, SiteID string; PageID, EntityID *string; Code Code; Severity Severity; SubjectKey string; Subject json.RawMessage; Options []Option; Status Status; Resolution, ResolvedBy string; CreatedAt, UpdatedAt time.Time }`; `New(f) (Finding, error)` fills severity and options from the code and refuses an unknown code or an empty subject key; `SeverityOf`, `OptionsOf`; `SubjectPage(pageID)`, `SubjectPageField(pageID, key)`, `SubjectCrumb(pageID, path)`, `SubjectSite`; `ResolvedByObservation`, `ResolutionObservedClear`.
- **site.** `type Settings struct{ LinkURLForm string; UserAgent string; Observe ObserveSettings{RenderConcurrency, FullWalkEvery int}; Builders []BuilderRule{Name string; DataAttributes, ClassPrefixes, ShortcodePrefixes []string}; TechnicalShortcodes, TechnicalBlocks []string; Technical TechnicalPages{Front, Posts, Privacy, Shop, Cart, Checkout, Account, Terms int64}; Observation ObservationState{LastStartedAt, LastFullWalkAt *time.Time; SinceFullWalk int} }` with camelCase JSON; `DefaultSettings()` (auto, 4, 10, the builder rules for elementor, wpbakery, divi, beaver-builder, oxygen, bricks, breakdance, the three shortcodes and two blocks); `Settings.Normalized()`; `Settings.Validate()` (the form in its set, concurrency 1–16, cadence ≥ 1). `Site` gains `Settings`.
- **Tests.** `TestTheCatalogNamesEveryDeniedField`, `TestAnInstancedKeyResolvesByItsKind`, `TestAValueStartsAgreed`; `TestANewFindingTakesItsCodesSeverityAndOptions`, `TestAnUnknownCodeIsRefused`; `TestDefaultSettingsAreFilledIn`, `TestSettingsRefuseAnUnknownURLForm`.
- Commit `feat(field,finding,site): the field catalog, the finding record and the per-site settings`.

##### 1b.4 — migrations 0045–0054 (agent M)
- **Files.** Create the ten files under `internal/adapters/sqlite/migrations/`; modify `migrations_test.go` (`latestMigration = 54`, the want list, the table list).
- **Shapes.** 0045 `NO TRANSACTION`: `pages_rebuilt` with `wp_type TEXT NOT NULL CHECK (wp_type <> '')` and every other column as it stands, the rows copied, the five indexes recreated; its down deletes rows of an unknown type outside the transaction and rebuilds the old `CHECK`. 0046 adds `page_type_id TEXT` (indexed), `parent_reason TEXT NOT NULL DEFAULT 'none'` with its `CHECK`, `technical_reason TEXT NOT NULL DEFAULT ''` with its `CHECK`, `read_only_reason TEXT NOT NULL DEFAULT ''`. 0047 `page_identity(page_id PK → pages CASCADE, url, status, wp_type, wp_parent_id INTEGER, modified_gmt TEXT, observed_at)`. 0048 `page_terms(page_id → pages CASCADE, taxonomy, term_id, name, slug, parent_term_id DEFAULT 0, PK (page_id, taxonomy, term_id))`. 0049 `page_fields(page_id → pages CASCADE, field_key, current_value, current_hash, base_value, base_hash, observed_at, written_at, written_run_id, PK (page_id, field_key))`. 0050 `page_bodies(page_id, side CHECK current|base, raw, hash, kind CHECK classic|blocks, observed_at, PK (page_id, side))` and `page_blocks_written(page_id, block_hash, run_id, written_at, PK (page_id, block_hash))`. 0051 `page_renders(page_id PK, render_hash, content_root CHECK, regions '[]', jsonld '{}', trail '[]', headings '[]', word_count_content, word_count_generated, builder '', url_form '', robots '', extracted_at)`. 0052 `NO TRANSACTION`: `page_links` rebuilt with `type CHECK`, `region ''`, `source CHECK raw|render`, `position INTEGER`, `ours INTEGER CHECK (0,1)`, the old rows copied as `contextual`, `raw`, `ours = origin = 'generated'`, plus the index `page_links_from_source (from_page_id, source)`; the down rebuilds the 0009 shape. 0053 `findings(id PK, site_id → sites CASCADE, page_id → pages CASCADE, entity_id → entities CASCADE, code, severity CHECK, subject_key, subject '{}', options '[]', status CHECK, resolution '', resolved_by '', created_at, updated_at, UNIQUE (site_id, code, subject_key))` with `findings_site_status` and `findings_page`. 0054 `sites.settings TEXT NOT NULL DEFAULT '{}'`.
- **Tests.** The round trip; `TestPagesRebuildKeepsEveryRowAndItsReferences` (a page, a link, an entity's canonical page and a template override survive 0045 up and down); `TestPageLinksRebuildCopiesOldLinksAsContextualRawLinks`; `TestADownRestoresThePagesCheck` (the schema text after the down equals the 0044 text); `TestFindingsSubjectIsUniquePerSite`.
- Commit `feat(sqlite): migrations 0045 to 0054, the page model's tables`.

##### 1b.5 — `domain/render` (agent R, after 1b.1 and 1b.2)
- **Files.** Create `internal/domain/render/{render.go,extract.go,root.go,jsonld.go,trail.go,regions.go,links.go,builder.go}`, tests, and `testdata/<stand>/<slug>.{public,filtered,raw}.html` captured once from the running stands (anonymous GET; the plugin's `/content/{id}/rendered` and `/content/{id}/raw` with the stand's application password) and committed.
- **Interfaces.** `type RootState string` (`found empty_raw dynamic_only not_found`); `type RegionKind string` (`breadcrumb listing navigation content content_dynamic generated unknown`); `type Region struct{ Kind RegionKind; Inside bool; Links, Words int; Signal string }` (`jsonld aria microdata landmark links`); `type Link struct{ Href, Path string; Kind pagemap.LinkKind; Anchor string; Type pagemap.LinkType; Region RegionKind; Position int }`; `type Heading struct{ Level int; Text string; Generated bool }`; `type Builder struct{ Name string; DataAttributes, ClassPrefixes, ShortcodePrefixes []string }`; `type Input struct{ Public, Filtered string; Body block.Body; Site pagemap.Site; Document string; Builders []Builder; Lookup func(path string) (pagemap.WPType, bool) }`; `type Extraction struct{ Hash string; Root RootState; Regions []Region; Trails []pagemap.Trail; JSONLD map[string][]json.RawMessage; Links []Link; Headings []Heading; WordsContent, WordsGenerated int; Builder, URLForm, Robots string }`; `func Extract(in Input) (Extraction, error)`; `Extraction.NoIndex()`; `type Record struct{ PageID string; Extraction; ExtractedAt time.Time }`.
- **Rules.** Words compare as `content`'s `wordsOf`. The root is the smallest element whose words hold the filtered content's sequence, else the raw text blocks' sequences (at least 60% of them), else the `main`/`article` landmark (`not_found`); `empty_raw` when the body is empty, `dynamic_only` when it is dynamic only, else `found`. Regions in order: the elements holding the BreadcrumbList's URLs (breadcrumb, `jsonld`) and the ItemList's or CollectionPage's (listing, `jsonld`); a breadcrumb nav by decision 2 (`aria`, `microdata`); `header`, `footer` and `nav` outside `main` (navigation, `landmark`); inside the root a container with at least three links whose `Lookup` answers `product` (listing, `links`); the root's raw text (content); the rest of the root `content_dynamic` when the body is dynamic only, else `generated`; everything else `unknown`. Every `a[href]` becomes a `Link` through `Site.ResolveAt(Document, href)`, typed by its region, `Position` the index of the raw block carrying it for a content link, else −1. Trails come from every BreadcrumbList (`jsonld`) and every breadcrumb nav (`nav`). Headings are the root's `h1`–`h6`, `Generated` when their words are not in the raw. `WordsContent` counts the raw's words found in the root, `WordsGenerated` the root's other words. `Builder` is the first rule whose data attribute, class prefix (in the root) or shortcode prefix (in the body) matches. `URLForm` is the majority of the internal hrefs, `absolute` or `relative`, empty with none. `Robots` is `meta[name=robots]`'s content, lowercased.
- **Tests.** `TestExtractFindsTheContentRootOnEveryStandPage` (replica: peptides `empty_raw`, bpc-157 `dynamic_only`, tb-500 `found`, the product `found`, the term `not_found`; vanilla: guide, setup, powder `found`), `TestExtractReadsEveryBreadcrumbShape` (the replica hub's two JSON-LD lists and nav; the replica product's Rank Math list and microdata nav; vanilla's WooCommerce list with `item` objects and its list-less nav ending in a text crumb; the vanilla term), `TestExtractTypesEveryLink` (tb-500: the body's `/bpc-157/` contextual at its block, the nav's breadcrumb, the section's listing, header and footer navigation), `TestExtractNamesTheListingFromTheItemList` (one product is still a listing), `TestExtractCountsGeneratedWords`, `TestExtractResolvesARelativeHrefAgainstTheDocument`, `TestExtractDetectsABuilder`, `TestExtractReadsRobots`, `TestExtractSurvivesAnErrorPage` (a 403 body).
- Commit `feat(render): the extraction of a page's render, its content root, regions, trails and typed links`.

##### 1b.6 — repositories (agent M, after 1b.1–1b.3)
- **Files.** Create `page_identity_repo.go`, `page_field_repo.go`, `page_body_repo.go`, `page_render_repo.go`, `finding_repo.go`; rewrite `page_link_repo.go`; modify `site_repo.go`, `page_repo.go`, `sqlitetest/fixtures.go`, `internal/app/build_repos.go`; a test per repository.
- **Interfaces.** `PageIdentityRepo`: `Put(ctx, pagemap.Identity)`, `Get(ctx, pageID)`, `ListBySite(ctx, siteID)`, `ReplaceTerms(ctx, pageID, []pagemap.Term)`, `ListTerms(ctx, pageID)`, `ListTermsBySite(ctx, siteID) (map[string][]pagemap.Term, error)`. `PageFieldRepo`: `Observe(ctx, pageID, []field.Value)` (upserts current and `observed_at`; base only on a new row; a key absent from the slice is left alone), `Record(ctx, pageID, key, value, runID, at)` (current and base, `written_at`, `written_run_id`), `ListForPage`, `ListBySite (map[string][]field.Value)`. `PageBodyRepo`: `Observe(ctx, block.Snapshot)` (current; base too when none exists), `Record(ctx, pageID, raw, runID, at)`, `Get(ctx, pageID) (current, base block.Snapshot, found bool, err error)`, `ListBySite`, `ReplaceWritten(ctx, pageID, hashes []string, runID, at)`, `ListWritten(ctx, pageID)`. `PageRenderRepo`: `Put(ctx, render.Record)`, `Get`, `ListBySite`. `PageLinkRepo`: `ReplaceForPage(ctx, pageID, source pagemap.LinkSource, links)`, `ListForPage`, `ListBySite`. `FindingRepo`: `Upsert(ctx, finding.Finding)` (new → open; open → updated; resolved → reopened; dismissed → untouched unless the subject JSON differs), `Clear(ctx, siteID, code, subjectKey, resolvedBy, resolution, at)` (open → resolved), `OpenBySite(ctx, siteID, code)`, `ListBySite(ctx, siteID, status)`, `ListForPage`. `SiteRepo` reads and writes `settings` through `site.Settings` normalised; `PageRepo` reads and writes the four columns.
- **Tests.** A round trip per repository on a real file; `TestPageLinkRepoReplacesOneSourceAndKeepsTheOther`; `TestFindingRepoUpsertFollowsTheLifecycle`; `TestPageFieldRepoObserveKeepsTheBase`; `TestPageBodyRepoRecordSetsBothSides`; the cascades on a page delete.
- Commit `feat(sqlite): the page model's repositories and the findings`.

##### 1b.7 — `application/observe`
- **Files.** Create `internal/application/observe/{service.go,deps.go,readers.go,state.go,inventory.go,content.go,reconcile.go,store.go,render.go,derive.go,technical.go,findings.go,summary.go,page.go}` and tests with in-package fakes; `internal/app/build_services.go`.
- **Interfaces.** `CoreReader{ PostTypes; ListItems(ctx, postType string, ItemQuery) (ItemPage, error); SiteSettings }`, `StoreReader{ ListProducts(ctx, ProductQuery) (ProductPage, error); ListVariations(ctx, productID int64, page int) (VariationPage, error); StorePages(ctx) (TechnicalPages, error) }`, `PluginReader{ ListContent(ctx, ContentQuery) (ContentPage, error); GetRaw; GetRendered; GetSEOMeta; GetTerms; Settings; PreviewLink }`, `RenderReader{ Visit(ctx, url string) (Visit, error) }`, all over observe's own item types; `type Readers struct{ Core CoreReader; Store StoreReader; Plugin PluginReader; Render RenderReader; Plugin capabilities, Outdated bool; RequiredVersion, Version string }`; `type Options struct{ Full, RenderRescan bool }`; `type Stage string` (`inventory content store render derive done`); `type State struct{ SiteID string; StartedAt time.Time; Full bool; Since *time.Time; Stage Stage; Types []string; Cursor string; CoreType string; CorePage int; StoreAfter int64; RenderAfter string; Seen []string; Counts Counts }`; `func (s *Service) Begin(ctx, siteID, Readers, Options) (State, error)` (full when forced, first, every Nth, or without the plugin); `func (s *Service) Step(ctx, State, Readers, batch int) (State, error)` (one batch of the current stage, the stage advanced when done); `Summary(ctx, siteID) (Summary, error)` with `Pages, Observed, Rendered int; ByType, Roots, LinksByType, Technical, Findings map[string]int; TrailsResolved, Listings, ReadOnly int`; `Page(ctx, pageID) (PageModel, error)` with the page, identity, terms, fields, both body sides, the render, the links and the open findings.
- **Stages.** *inventory*: the manifest → `plugin_outdated` raised or cleared; the types by decision 10; the technical ids into `settings.Technical`. *content*: the plugin's `/content` with `types` and `since` (incremental) or core's items per type (always full); per item the `pages` row as today's reconcile keeps it (the old columns written beside the new rows; a planned title or H1 that differs becomes `page_title_differs`/`page_h1_differs` and the plan is kept; `path_taken_on_site` stays an artifact finding), the identity, the terms (plugin `/terms/{id}` when present, else the item's), the fields (`title`, `excerpt`, `seo_*`), the body snapshot when the hash changed or none is stored (raw through `GetRaw`, else `content.raw`), the raw links (`contextual`, `raw`, position by block, `ours` by `page_blocks_written`), `body_changed_on_site` and `field_changed_on_site` raised or cleared. *store*: `ListProducts` (`ModifiedAfter` when incremental) → `product_description`, `product_short_description`, the denied facts, `variation_description:<id>` and `variation:<id>` from `ListVariations`, categories as `page_terms` when the plugin read none. *render*: the pages decision 9 names, `RenderConcurrency` at a time, each `Visit` (a draft through `PreviewLink`) plus `GetRendered` for a content post → `render.Extract` → the render row and the `render`-source links (every link outside the content root, and inside it those the raw lacks); a failed visit → `render_unreachable`. *derive*: technical reasons (decision 8), `ReadOnlyReason`, `builder_detected`, `technical_detected`; parents through `DeriveParent` for pages with a WordPress id, `parent_unresolved` per unresolved crumb, `parent_differs` when a `workbook` parent disagrees; `url_form_mixed` when the site's pages disagree; findings no longer raised cleared; on a full walk the pages with a WordPress id not seen are archived; `settings.Observation` updated; `events.PagesChanged`.
- **Tests.** `TestTheFirstObservationWalksTheWholeSite`, `TestASecondObservationIsIncrementalAndChangesNothing`, `TestEveryTenthObservationWalksFully`, `TestCoreWithoutThePluginAlwaysWalksFully`, `TestARenderIsRefetchedForAChangedPageOrOnRescan`, `TestAFailedVisitIsAFindingNotAFailure`, `TestAnOutdatedPluginIsAFindingThatClearsWhenUpdated`, `TestAFieldChangedOnTheSiteIsAFindingUntilAccepted`, `TestOurOwnWriteRaisesNoFinding`, `TestTechnicalPagesAreNamedWithTheirReason`, `TestAPlannedPageKeepsItsWorkbookParent`, `TestTheDerivedParentWinsAndTheWorkbookParentIsAFinding`, `TestProductsFillFieldsAndFactsAndVariationsStayUnderTheirProduct`, `TestASiteWithMixedURLFormsIsOneFinding`, `TestTheSummaryCountsTheModel`.
- Commit `feat(observe): the observation of a site into the page model`.

##### 1b.8 — the href alignment, the fake's renders, a fresh visit (agent R, after 1b.5)
- **Files.** `wp-plugin/postulator-companion/includes/normalize.php` (`href_base(string $permalink)`: the path up to its last slash), `content.php` (`extract_links($content, href_base(get_permalink($post)))`), `postulator-companion.php` (1.4.1), `openapi.yaml` (`info.version`, the `ContentLink.href` description), `contract_test.go`, `wp-plugin/tests/normalize.php` (the `document` column), `wp-plugin/testdata/hrefs.json` (`document` cases); `internal/adapters/wp/plugin.go` (`PluginVersion = "1.4.1"`), `plugin_test.go` (the outdated table), `visit.go` (`Cache-Control: no-cache` and a `_=<unix nanos>` query argument), `store.go` (`ManageStock` decoding `true`, `false`, `"parent"`), their tests; `wptest/plugin.go` (`internalTarget(host, document, href)`), `wptest/hrefs_test.go`, `wptest/state.go` (`Item.Crumbs []Crumb{Name, Path}`), `wptest/storefront.go` (a public render for every page and post: head with a BreadcrumbList when the item has crumbs; `header` and `footer` navs; `main > article` with a breadcrumb `nav` when crumbs, the `h1` and `Filtered` else `Content`); e2e `hrefs_test.go` (a relative anchor on vanilla's nested page resolves under the page).
- **Tests.** `TestTheFakeClassifiesTheSharedHrefTable`, the PHP table on 7.4, `TestAnOutdatedPluginIsNamedByVersionAndCapabilities` (1.4.0 outdated now), `TestVisitAsksForAFreshRender`, `TestAVariationManagedByItsParentDecodes`, `TestThePublicRenderOfAPageCarriesItsBreadcrumb`, e2e `TestTheSameHrefIsClassifiedAlikeOnBothSides`.
- Commits `fix(plugin): a relative href resolves against the document, as a browser does` and `feat(wptest): the public render of a page, a fresh visit and a parent-managed variation`.

##### 1b.9 — the steps
- **Files.** Rewrite `internal/runtime/steps/sync_site.go` (the first execution `Begin`s, every execution `Step`s one batch, the `observe.State` in the checkpoint under `sync`, `TransitionWait` until `done`, the artifact `SiteSyncResult{StartedAt, Full, Stage, Summary, Findings, Done}`, the message "observed N pages of <site>", the `renderRescan` param); delete `sync_pull.go`, `sync_cursor.go`, `sync_reconcile.go`, `sync_settle.go`, `sync_archive.go` (observe owns them); create `observe_readers.go` (`*wp.Client` → `observe.Readers`); `publish.go` (an update sends title, content and status only; `compare` checks slug and parent on a create only; `record` records the body and the title and SEO fields through `recorded.go`), `publish_parent.go` (the creation lookup: the parent is `ParentPageID`'s page, pending without a WordPress id, no path check), `publish_product.go`, `relink_page.go`, `revert.go`, `sync_back.go` (each records what it wrote); delete `repair_hierarchy.go` and `repair_test.go`; `steps.go`, `deps.go` (`Fields`, `Bodies`, `Observe`); tests `sync_test.go` (over wptest: identity rows, parent reasons, incremental), `publish_test.go` (`TestPublishUpdatesWithoutSlugOrParent` replaces the two tests that asserted a move and a slug), `syncplan_test.go`, `pipeline_test.go`.
- Commits `feat(steps): sync_site observes the site into the model`, `feat(steps): publish writes neither slug nor parent to an existing page`, `refactor(run): the repair kind loses its recipe and repair_hierarchy goes`.

##### 1b.10 — the start, the recipes, the vocabulary, the tools, the import
- **Files.** `internal/domain/run/{recipes.go,step.go,recipe_test.go}`; `internal/application/runs/{start.go,start_test.go}` (`parentOf` follows `ParentPageID`, pending without a WordPress id, no path check; `recipeFor` refuses `repair`); `internal/transport/wails/vocabgen/main_test.go`; `frontend/src/generated/vocab.ts` by `task vocab`; `frontend/src/copy/runs.ts`, `frontend/src/features/runs/start.tsx`, `start.test.ts`; `internal/application/tools/runs_start.go` and `registry_test.go`'s ceiling; `internal/application/imports/units.go` (`TechnicalReason = workbook` on `own_entity: no`) and a test; `internal/e2e/client_test.go` (the repair section goes); `docs/CONTRACTS.md`, `docs/ARCHITECTURE.md`, `CLAUDE.md` lines.
- Commit `feat(runs): a start follows the derived parent and refuses a repair run`.

##### 1b.11 — the wiring (agent W, after 1b.3)
- **Files.** `internal/adapters/wp/registry/registry.go` (per-site options with `wp.WithUserAgent(record.Settings.UserAgent)`, the cache key carrying it), `registry_test.go`; `internal/application/sites/{service.go,requests.go,views.go}` (`Settings` in `Get` and `Update`), tests; `internal/transport/wails/sites.go` DTOs; `internal/application/tools/sites_update.go` (`userAgent`, `renderConcurrency`, `fullWalkEvery`, `linkUrlForm`) and the ceiling; `internal/app/sitespecific_test.go` (`TestTheExtractionNamesNoSelector`: no string literal matching `class=`, `className`, `querySelector` or `^[.#][a-z]` in `internal/domain/render` and `internal/application/observe`, and the literal `"class"` only in `render/builder.go`).
- **Tests.** `TestASiteUserAgentOverrideReachesItsClient`, `TestSettingsRoundTripThroughTheSitesService`, `TestTheExtractionNamesNoSelector`.
- Commits `feat(sites): per-site settings with a User-Agent override the registry honours`, `test(app): the extraction names no selector`.

##### 1b.12 — the e2e proof
- **Files.** Create `internal/e2e/observe_test.go`; modify `harness_test.go` (the observe reads), `degraded_test.go` (`assertCorePull` reads the parent reason), `internal/adapters/wp/e2e/stands_test.go` (`standFacts` gains the expectation table: parents and reasons by path, roots by slug, listings, technical pages by reason, the unresolved crumb's name).
- **Test.** `TestTheSiteIsObservedIntoTheModel`: after `SyncSite`, `Summary.Observed` equals the pages with a WordPress id; every published page has identity, a `title` field, a body and a render row; vanilla: `/guide/setup/` under `/guide/` by `wp_parent`, `/product/powder/` under the `koffein` term page by `jsonld_breadcrumb`, no listing on a page, shop, cart, checkout and account technical; replica: the hubs under Home by `url` with `parent_unresolved` naming Healing Peptides, roots `empty_raw`, `dynamic_only` and `found`, a listing on every hub, the products under the `healing-peptides` term page by `jsonld_breadcrumb`, that term page under `peptides` by `wp_parent`, Home `front_page`; a second `SyncSite` leaves every hash, parent and render unchanged and raises nothing new; the hold test still waits and publishes.
- Commit `test(e2e): the site is observed into the model on both stands`.

##### 1b.13 — the gate
- The whole gate of the global constraints with exit codes read: `wp/e2e` and `e2e:full` on both stands, `e2e:full:noplugin` on vanilla, `plugin:lint` and `plugin:test`, the footprint and site-specific scans, `task build`, `go vet` and `-tags e2e`, `golangci-lint run`, `task lint:e2e` alone, `go test -race -count=1 -p 2 -covermode=atomic -coverprofile=coverage.out ./...`, `go run ./cmd/covergate`, the comment check, `gofmt -l .`, `npm run typecheck`, `npm run test:run`, `task ui:lint`.
- Docs: `STATUS.md` (1b done, 1c next, the gate), `DECISIONS.md` (stage 1b), the spec (16.1, 16.5, 16.8), `CLAUDE.md` footguns, `CONTRACTS.md`, `ARCHITECTURE.md`.
- Commit `docs: stage 1b's gate is green`.

### 1c — the screens

- **Touches.** `frontend/src/features/pages/{drawer,details,summary,links}.tsx` and new model panes (identity, hierarchy with reason, regions, trail, typed links, technical and read-only reasons, blocks read-only), `frontend/src/features/links/*` (counts by type, contextual owed), `frontend/src/features/sites/*`, `frontend/src/generated/vocab.ts` (`linkTypes`, `regionKinds`, `parentReasons`, `technicalReasons`, `contentRootStates`), `internal/application/pages/views.go`, `internal/transport/wails/pages.go`, the UI harness fixtures.
- **Exit.** `npm run test:run` and `task ui:lint` green; the harness walk shows a page as the model. Owner sees: every page as the model, Linking by type.

#### 1c tasks

**Status, 2026-10-18: 1c.1 to 1c.14 done; 1c.15's gate is partial.** The static checks, the
build, the frontend suite and both stands' observation proof are green; the Go race suite with
the coverage gate, `task ui:lint` and the stands' full suites were stopped or not started for want
of memory and are left for the owner, listed under **The gate** in `STATUS.md`. The rulings are
under stage 1c in `DECISIONS.md`.

**Part 0 closes what the owner's review of 1b found; Part 1 is the screens.** Execution: `superpowers:executing-plans`, one accountable owner, at most two agents at once and only on disjoint Go packages (R: `domain/render`, then `adapters/wp` and `wptest`; F: `frontend/`). Every commit uses `git commit -- <paths>`; edits go through Write and Edit; every gate command runs under `set -o pipefail` with its exit code read; a test that pins an old value moves in the same commit; the heavy e2e legs run alone.

**Decided at the plan** (recorded in `DECISIONS.md` at the close; the spec amended in the same commits):
1. **A crumb resolves** by its href against every modelled page (a modelled page whose own visit redirects to another is followed), else through the redirect the render stage recorded on it, else by exact name (`graph.Key` over observed titles, H1s and term names, after `html.UnescapeString`; the page, its descendants and archived pages excluded) when the crumb has no link or its internal link is dead (≥ 400 after redirects). One name candidate resolves; several are `parent_ambiguous`. A live unmodelled or external href never falls to its name. An unresolved nearest crumb still never falls to the next crumb.
2. **The method is its own column**, `pages.parent_method` (`href`, `redirect`, `name`), beside the reason that keeps the source (`jsonld_breadcrumb`, `aria_breadcrumb`), so no `pages` rebuild; `vocab.ts` gains `parentMethods`.
3. **The front page is the root**: not technical, no parent; a crumb at `/` resolves to it when it is modelled and is dropped otherwise; a trail whose nearest crumb is the front page is weak and yields to the other trails, then to the WordPress parent, and only then names the front page.
4. **A redirect lookup** is one anonymous HEAD (a GET closed unread on 405 or 501) through the limiter on every hop and the site's User-Agent, stopping at the first off-site hop, made in the render stage and stored on the crumb in `page_renders.trail` as the normalized final path and status. It may not land on `/`, the page or a descendant, and its target must carry the crumb's name, because WordPress's 404 guesser redirects a dead slug to any similar one. A failed lookup makes the render unreachable for the walk.
5. **A page whose own visit is redirected** to another path keeps no trail, region, link, builder, robots or URL form from the other page's render, its root state follows its body, and `render_redirected` names the target.
6. **A parent cycle** is broken at its page with the smallest path, which re-derives without trails, repeated until none remains; `parent_cycle` names it.
7. **`archive`** is a root state: a term page, or the site's shop or posts page. When the root is not located the state follows the body (`empty_raw`, `dynamic_only` counting the site's technical blocks and shortcodes), and only unfindable text is `not_found`.
8. **Link rules**: a link to the document itself is no page link; `rel` tag, category or author outside the content is navigation; the product-listing rule (≥ 3 product links, ≥ 2 products) applies outside the root; ARIA landmark roles count as their elements outside the root.
9. **Derive writes only what it owns**: `SiteRepo.UpdateObservation` (`json_set` on `settings.technical` and `settings.observation`) and `PageRepo.UpdateDerived`; `adoptExtensions` and `CheckPlugin` write only the plugin columns.
10. **An owed link is satisfied by a contextual link only**; orphans and off-graph links count contextual links, and technical pages, term pages and the front page are never orphans.
11. **The bindings sit on `PagesService`** (`Model`, `ModelSummary`) through a second source, so the service count stays fifteen.
12. **Stage 2 and 3's migrations move up by three** (stage 2 from 0058, stage 3 from 0064), 0055–0057 being taken here.

##### 1c.1 — the gate legs 1b left
- `task e2e:full:noplugin STAND=vanilla`, `task e2e:up STAND=vanilla`, `task ui:lint`, one at a time with free memory checked; whatever is red is fixed in its own commit.

##### 1c.2 — the diagnosis
- `internal/e2e/observe_test.go` logs a row per page (path, type, root, trails with names and hrefs, parent, reason, findings); one replica run before any change gives the diagnosis and the before counts.
- Commit `test(e2e): the observation proof logs the model page by page`.

##### 1c.3 — the extraction (agent R)
- `internal/domain/render`: `RootArchive`; `Input.Archive` and `Input.Dynamic` (the technical block and shortcode names); the state order of decision 7; `Crumb.Href` filled by both trail readers; the link rules of decision 8; goldens per state and per rule, the two stands' captured renders kept.
- Commits `feat(render): truthful content-root states and the archive` and `feat(render): links typed by rel, listings outside the root and landmark roles`.

##### 1c.4 — redirects in the client and the fake (agent R, after 1c.3)
- `internal/adapters/wp`: `Visit` answers its final URL; `Resolve(ctx, url)` per decision 4; their tests through a recording server. `wptest`: `WithPublicRedirect(from, to)`, a public HEAD outside auth and fault injection.
- Commit `feat(wp): a visit names where it landed and a redirect is resolved without a body`.

##### 1c.5 — storage
- Migrations 0055 `pages.parent_method`, 0056 the front page's technical reason cleared, 0057 `page_renders` recreated with `archive` and `final_path`; `PageRepo.UpdateDerived`, `SiteRepo.UpdateObservation`, `SiteRepo.UpdatePlugin`; `migrations_test.go` (`latestMigration = 57`).
- Commit `feat(sqlite): the parent's method, the archive state and writes narrowed to their owners`.

##### 1c.6 — the derivation
- `internal/domain/pagemap`: `ParentMethod`, `Page.ParentMethod`, `Crumb` with `Href`, `Final`, `Status`; `TechnicalFront` gone; `DeriveParent` per decisions 1, 3 and 6 over `Index.ByName` and the redirects of modelled pages; the table gains the weak Home-only list, a same-name tie, the live-link guard, a dead link that falls to its name, a modelled alias, a cycle.
- Commit `feat(pagemap): a crumb resolves by href, redirect or name and the front page is the root`.

##### 1c.7 — the observation
- `internal/application/observe`: the render stage resolves crumbs and records a redirected visit; `derive` offline with the narrowed writes; `parent_ambiguous`, `parent_cycle`, `render_redirected` raised and cleared; `Input.Archive` and `Input.Dynamic` passed; the summary counts trails by method; `steps/observe_readers.go` adapts `Resolve`; `sync_site`'s `adoptExtensions` narrowed; `sync.CheckPlugin` narrowed.
- Tests: the cases of decisions 1–6 over the fakes and wptest, `TestASettingsEditDuringTheDerivationSurvives`, `TestAPageEditDuringTheDerivationSurvives`.
- Commits per concern: `feat(observe): crumbs resolved through their redirects in the render stage`, `fix(observe): the derivation writes only the columns it owns`, `feat(observe): the front page is the root of the site`.

##### 1c.8 — the replica fixture and the stand tables
- `docker/e2e/stands/replica/{site-behaviour.php,provision.sh,products.csv}`: crumbs from a per-slug table, the new hub `/healing-peptides/`, `/lab-guides/` and `/reconstitution/`, the 301 from `/category/healing-peptides/`, a product with an empty description; `internal/e2e/observe_test.go` and `standFacts` where counts move.
- Commit `test(e2e): the replica carries every breadcrumb shape the client's sites use`.

##### 1c.9 — the vocabularies
- `vocabgen`: `linkTypes`, `linkSources`, `regionKinds`, `parentReasons`, `parentMethods`, `trailSources`, `technicalReasons`, `contentRootStates`, `findingCodes`; `task vocab`.
- Commit `feat(vocab): the page model's vocabularies`.

##### 1c.10 — the bindings and the link view
- `internal/application/observe/views.go` (tagged, `dto.Time`), `Page` reading its render and identity directly; `PagesService.Model` and `ModelSummary`; `pages.PageLink` with type, region, source, position and ours.
- Commit `feat(pages): the page model and its summary are bound`.

##### 1c.11 — Linking by type
- `internal/application/reports/{linkaudit,site,requests}.go` per decision 10, `LinkTotals` with a named count per type.
- Commit `feat(reports): the link audit owes contextual links and counts every type`.

##### 1c.12 — the drawer's Model tab (agent F)
- `frontend/src/features/pages/model/*` one pane a file, `copy/pages.ts`, `labels.ts`, the endpoint, hook and key, `bridge.tsx`; the Links tab shows type and source; screen tests.
- Commit `feat(pages): the drawer shows a page as the model`.

##### 1c.13 — Sites, Overview and Linking (agent F)
- The settings section and the sync options on Sites, the observation summary on Overview, the by-type strip on Linking; screen tests.
- Commit `feat(sites): per-site settings, the sync options and the observation summary`.

##### 1c.14 — the harness
- A harness test that the seeded core carries a model; walk routes for the Model tab and the Sites settings.
- Commit `test(harness): the seeded site is observed into the model`.

##### 1c.15 — the gate
- The whole gate of the global constraints with exit codes read; docs: `STATUS.md` (stage 1 done, stage 2 next), `DECISIONS.md` (stage 1c), the spec (16.1, 16.4, 16.5, 16.8), `CONTRACTS.md`, `ARCHITECTURE.md`, `CLAUDE.md` footguns.
- Commit `docs: stage 1c closes stage 1`.

## Stage 2 — page types, the field catalog and scope with UI, the context pack

**Goal.** The client names his page types over presets, enables fields per type with an instruction, limits and a link policy, and the writer, the linker and the judge work from the context pack. Split on 2026-10-18 into two gated sub-stages: **2a** the types, the scopes, the link rules in the scopes, the entity card and the move from `kind`; **2b** the context pack and the writer, linker and judge reading it.

### 2a — page types, the field catalog and scopes, the entity card

- **Touches.** `internal/domain/{note,pagetype,field,pagemap,graph,template,content,importmap}`, `internal/application/{pagetypes,pages,sites,templates,graph,imports,observe,reports,content,tools}`, `internal/runtime/{claim,budget}.go` and `steps/{policy,generate_body,insert_links,repair_links,validate,relink_page,relink_neighbors,preflight,resolve_context,observe_readers}.go`, `internal/adapters/sqlite` (repositories, migrations 0058–0063), `internal/transport/wails` (`templates.go`, `pages.go`, `vocabgen`), `cmd/postulator` (harness), `internal/e2e`, `frontend/src/features/{settings,templates,pages,graph,sites,links,agent}`, `frontend/src/copy`, `frontend/scripts/routes.json`.
- **Migrations.** 0058 `page_types` and `pages.page_type_source`, `page_type_reason`; 0059 `field_scopes`; 0060 the presets seeded for every site, their default scopes and the backfill of `pages.page_type_*` (data only; the down deletes what the up made); 0061 `entities.definition`, `facts`, `notes`; 0062 `link_policies` dropped, `sites.default_link_policy_id` removed by rebuild, `linkRules` stripped from stored specs and overrides (the down restores the table with the Default policy and the column); 0063 `entities.kind` dropped by the 0030 rebuild, `templates.page_kind` and its index dropped, saved mappings' `entity_kind`, `page_kind` and row type `kind` rewritten (the down restores the columns, the index and the names).
- **Tools.** `pagetypes_list` (the types with their scopes and the field catalog, so `fields_catalog` folds in), `pagetypes_save`, `pagetypes_delete`, `fields_set_scope`; `pages_update` takes `pageTypeId`; `graph_update_entity` takes the card; the six `policies_*` tools go; `kind` leaves the five graph request structs; `pageKind` and `linkRules` leave the templates trio and `templates_list`; `defaultLinkPolicyId` leaves `sites_update`.
- **Exit.** Unit: the `DeriveParent` table with name twins; the auto-assignment table (untyped pages only, a regular page waits for its render, a typed page never touched); `field.CheckScope` refusing a denied key at scope time; the backfill round trip over a fixture database with every kind; the SQL seed equal to the Go seed; the import's kind cell to a type by the site's own name, then synonym, a term page whatever its cell; the golden table proving the one resolver gives the four old answers over the old fixtures; a stored override carrying `linkRules` resolving after 0062; every migration's round trip. E2E on both stands, one at a time: types seeded on the first observation, the replica's hubs typed Hub by their listing, products Product, term pages Store category, the shop Catalog, technical pages Technical; a scope edit persists; a custom field's value read on the next observation. Owner sees: Settings → Page types with his names and the scope editor per type, a page's type with its source and reason in the drawer, an entity's canonical page type and its card on the Graph screen, no Policies screen.

#### 2a tasks

**Status, 2026-10-19: 2a.0 to 2a.10 done** (`34e79f7` to `999ee7d`); **2a.11 to 2a.15 had not landed** when 2b opened, and the owner moved them into 2b as its Part 0 (0.1 to 0.5 below), with the heavy gate legs run once at 2b's close.

**Part 0 settles the owner's ruling on name twins; Part 1 is the stage.** Execution: `superpowers:executing-plans`, one accountable owner, at most two agents at once and only on disjoint packages (F: `frontend/`; H: `cmd/postulator` and `internal/e2e`, after the Go has landed); every other Go package is the owner's. Every commit uses `git commit -- <paths>` with `gofmt -l` checked inside the commit command; edits go through Write and Edit; every gate command runs under `set -o pipefail` with its exit code read; a test that pins an old value moves in the same commit; the heavy legs run alone, one stand at a time.

**Decided at the plan** (recorded in `DECISIONS.md` at the close; the spec amended in the same commits):
1. **Name twins** (the owner's ruling): a crumb resolved by name splits its candidates into regular pages and term pages; one regular page wins with the method `name` and an informational `parent_name_twin` naming the crumb, the page and the term twins; two or more regular pages are `parent_ambiguous`; with no regular page one term resolves and several are ambiguous. A product is a regular page. The finding clears when the page is derived again without it, `breakCycles` included.
2. **Types are set by the observation only on an untyped page** (the owner's answer at the plan). The identity rules apply at once: a `product` → Product, a `product_cat` → Store category, the shop → Catalog, any technical reason → Technical; a regular page waits for its first render: a `listing` region inside the content root (`render.Region.Inside`) → Hub, else Entity. A typed page is never touched; the write is guarded (`WHERE page_type_id IS NULL`). An untyped page, a dangling id and another site's id resolve their template and policy as the identity rules would type the page without a render.
3. **"Category" names one thing** (the owner's answer at the plan): the read-only archive preset is `store_category`, named "Store category". A regular page whose entity kind, template kind or sheet cell says Category gets the hub role; a `product_cat` page is Store category whatever its cell says. A cell matches the site's own type names first (`graph.Key`), synonyms second, so a client who renames Hub to "Category" keeps his vocabulary.
4. **Source and reason**: `pages.page_type_source` (`''`, `auto`, `manual`; NOT NULL) and `page_type_reason` (`auto`: `product`, `store_category`, `shop`, `technical`, `listing`, `default`, `kind`; `manual`: `user`, `ai`, `import`). The whole-row `updatePage` stops writing the three type columns; `PageRepo.SetType` (manual) and `PageRepo.AutoType` (guarded) are their only writers after insert, so a snapshot the content stage, `publish.record`, the import or graph apply took before can never put an old type back.
5. **Seeding**: `pagetypes.Service.Seed` is idempotent and runs in the unit of `sites.Create` and in the observation's inventory stage; 0060 seeds the sites that exist in SQL with v4 UUIDs; a test proves the SQL seed equals the Go seed. Automatic typing and synonyms use the oldest type of a preset; the last type of a preset cannot be deleted; deleting a type reassigns its pages to another (manual, `user`).
6. **The backfill** (0060): identity facts first, then the template's `page_kind`, then the entity's kind (`category` on a regular page → Hub; `topic`, `custom`, `guide`, `comparison` → Entity), every assignment `auto` with the reason `kind` or its identity reason, never `manual`, because the import gave every entity of a Root group the kind `hub` by default.
7. **No `role` column**: the preset fixes the behaviour (`pagetype.RoleOf`). A type carries `template_id`; `ResolveForPage` resolves the run start's template, else the page's, else **the type's**, else the site default.
8. **A scope** (`field_scopes`): enabled, `aiWrites`, instruction, minimum and maximum length, `rewrites` (off by default), the link policy, and for a `meta:<key>` field a label and a provider (`plugin`, `store`). `field.CheckScope` refuses an unknown key, a denied key and any meta key the plugin's deny set refuses (`_`-protected keys, `total_sales`, the SEO plugins' keys as `openapi.yaml` pins them) with `INVALID`, details `field` and `reason: field_denied`. Defaults: Hub and Entity body, SEO title and description; Product description, short description, SEO title and description on, variation descriptions and image alt off; Store category, Catalog and Technical nothing writable.
9. **The link policy is `field.LinkPolicy`**: the former rules (`upDepth`, `downLinks`, `siblingMinWeight`, `maxLinks`, `maxPerTarget`, `parentLinkWithinParagraphs`, `childrenSection`), `forbidExternal`, `forbidSelf`, `anchorStrategy`, and an explicit `links` switch, since `maxLinks: 0` keeps meaning no cap; `links` is always written and a stored policy without it is refused. Each preset's default carries the former rules: Hub the hub built-in's, Store category and Catalog the category built-in's, Entity the guide built-in's; the product description up one level, no down, no siblings, at most three; the short description and Technical no links; the flags of the Default policy. **One resolver**, `pagetype.Set.Policy(page)`, over a site's types and scopes loaded once; a product's link-bearing field is `product_description`, every other page's `body`; a disabled scope carries no links. **The policy is taken at claim** (`sc.Links`) as the spec is, so a type changed during a run cannot split the brief from the compliance check.
10. **0062** strips `linkRules` from every stored template spec and override patch, because `template.Resolve` decodes with `DisallowUnknownFields`, and carries each site's chosen policy's flags onto its scopes; a template's or an override's own `linkRules` are not carried. The proof is a **golden table captured in 2a.6** from the old fixtures over template, override and site policy, before `EffectiveRules` goes; every divergence is listed in `DECISIONS.md` (a comparison page takes the guide's rules, a product the brief's budget, a page without a template is owed links). The down cannot restore a site's own policies.
11. **The entity card**: `definition`, `facts` (`[{label, value}]`), `notes` (`[{label, text}]`); notes move to `domain/note`, shared by pages and entities. The import gives the card a note column whose header names the entity ("Entity notes", "Entity definition", added to the detection) and every note of a row with no page; every other note stays the page's. The export writes the card's notes back under their labels.
12. **From `kind` to types**: one import field `page_type`; an entity-level header wins over a page-kind header and the preview names the column it dropped; "Template" and "Page template" stop mapping, the template coming from the type; an unknown cell is `unknown_page_type`; the product row type `kind` becomes `type`; a saved mapping holding both keeps the entity column. The downs restore `entities.kind` from the canonical page's preset and `templates.page_kind` from the built-ins' names, else `guide`.
13. **The bindings** for types and scopes sit on `TemplatesService` in place of its six policy methods, so the services stay fifteen; `PagesService.SetType` changes a page's type; the `pages_update` tool takes its own arguments with `pageTypeId` and records `ai`.
14. **Custom fields** are read by the content stage through `PluginReader.GetMeta`, for a re-pulled page or an enabled key never read; an absent key is stored empty so it is not asked again every walk; a change to meta alone is read on the next full walk, WordPress stamping no `modified` for it.
15. **The built-ins** lose `pageKind` and `linkRules`; the current seeds join `seed/superseded`, and a test refreshes an untouched built-in stored before 2a.

##### 2a.0 — the plan
- This section and 2b below.
- Commit `docs(plan): stage 2 split into 2a and 2b, and 2a's tasks`.

##### 2a.1 — name twins
- `internal/domain/pagemap/parent.go`: `Derivation.Twins`, the candidate split of decision 1; `parent_test.go` gains a page and a term of one name, two pages and a term, two terms, a lone term, a product beside its category. `internal/domain/finding`: `ParentNameTwin` (Info, `ignore`). `internal/application/observe/derive.go`: raised and cleared beside `parent_ambiguous`, cleared by `breakCycles`. `task vocab`. `DECISIONS.md` (the ruling, dated), spec 16.5 and 16.8.
- Commit `feat(pagemap): a page wins its name over a store category of the same name`.

##### 2a.2 — the domain
- `internal/domain/note` (`Note`, `Clean`, `Merge`; `pagemap` and the import move to it). `internal/domain/pagetype`: `Preset`, `Presets()` with their default names, `RoleOf`, `Type` and `Validate`, `Assign(facts) (Preset, Reason, bool)`, `Match(cell, types, facts)`, `Set` with `TypeOf(page)`, `ScopeOf(page, key)`, `Policy(page)`. `internal/domain/field`: `Scope`, `LinkPolicy` (JSON with `links` required), `CheckScope`, `DefaultScopes(preset)`, `Provider`.
- Tests: the assignment table, `Match` (name before synonym, term whatever its cell, category on a regular page to the hub role), `CheckScope` per refused class, the policy JSON refusing a missing `links`, `Set` over untyped, dangling and foreign ids.
- Commits `refactor(note): notes are their own domain`, `feat(pagetype): presets, roles and the typing rules`, `feat(field): scopes, the link policy and the scope-time refusal`.

##### 2a.3 — storage
- Migrations 0058, 0059, 0060; `PageTypeRepo`, `FieldScopeRepo`; `PageRepo` inserts and reads the type columns, `SetType`, `AutoType` guarded, `updatePage` without them; `migrations_test.go` (`latestMigration = 60`, the backfill over a fixture with every kind, entity and template kind, product, term, shop and technical page; the SQL seed against `pagetype.Presets()` and `field.DefaultScopes`).
- Commit `feat(sqlite): page types, field scopes and the backfill`.

##### 2a.4 — the use cases
- `internal/application/pagetypes`: `Seed`, `List`, `Get`, `Create` (from a preset, copying its scopes), `Save` (name, template), `Delete` (with reassignment; the last of a preset refused), `SetScope` (`CheckScope`), `Set`. `sites.Create` seeds in its unit; `pages.Service.SetType` (manual, refuses another site's type); `templates.ResolveForPage` takes the type's template; `internal/app/build_*.go`.
- Commit `feat(pagetypes): the client's page types and their scopes`.

##### 2a.5 — the observation
- `inventory` seeds; `derive` types untyped pages per decision 2 through `AutoType`; `content` reads custom fields through `GetMeta` per decision 14; `runtime/steps/observe_readers.go` adapts `wp.Client.GetMeta`; the fakes.
- Tests: the typing table over the fakes, `TestAManualTypeDuringTheWalkSurvives` (set between the content stage's read and its write), `TestAManualTypeDuringARunSurvives` (between claim and `publish.record`), a custom key read once and an absent key not asked again.
- Commits `feat(observe): an untyped page is typed by what the site shows`, `feat(observe): custom fields a scope names are read`.

##### 2a.6 — one resolver
- `internal/domain/content` takes `field.LinkPolicy` (`PlanLinks`, the brief, `InsertLinks`, `Compliance`); `runtime/claim.go` takes `sc.Links`; `steps/policy.go`, `relink_neighbors.planOf`, `reports.rulesFor`, `content/judge.go` and the estimate read the type set; the golden table of decision 10 in `pagetype`.
- Commit `feat(links): one link policy per page type and field`.

##### 2a.7 — the entity card
- Migration 0061; `graph.Entity` `Definition`, `Facts`, `Notes`; the repository, `graph.UpdateEntityRequest`, `graph_update_entity`; the import and export of decision 11; ceiling moved.
- Commit `feat(graph): the entity card`.

##### 2a.8 — the policies go
- Migration 0062; `link_policy_repo.go`, `templates/policies.go`, `template.LinkRules`, `template.LinkPolicy`, `site.Defaults.LinkPolicyID`, the `policies_*` tools and their group, the trio's and `sites_update`'s fields; the test of decision 10 for a stored override; ceiling down.
- Commit `refactor(templates): link rules live in the scopes; policies are gone`.

##### 2a.9 — from `kind` to types
- Migration 0063; `graph.Kind` and its filter, the proposals' schemas and `propose.tmpl`, `generate_body.tmpl` showing the type (`TestEveryPromptExecutes`); `Template.PageKind`, the seeds and `seed/superseded`; the import field `page_type`, the synonyms, the export; `agent/conversations.go`; `vocabgen` (`entityKinds` out; `pageTypePresets`, `pageTypeSources`, `pageTypeReasons`, `fieldKeys`, `fieldProviders` in); the harness data and the e2e fixtures that name kinds.
- Commits `refactor(graph): an entity has no kind; it shows its page's type`, `refactor(templates): a template has no page kind`, `feat(imports): the kind cell names a page type`.

##### 2a.10 — the tools
- `pagetypes_list`, `pagetypes_save`, `pagetypes_delete`, `fields_set_scope`, `pages_update` with its own arguments; `groups.go`; the registry list; the ceiling with the bytes logged.
- Commit `feat(tools): page types and scopes for the agent`.

##### 2a.11 — the bindings and views
- `TemplatesService` (types and scopes), `PagesService.SetType`, the page view's type with source and reason, the graph view's canonical page type; `task build`.
- Commit `feat(wails): page types, scopes and a page's type are bound`.

##### 2a.12 — the screens (agent F)
- Settings → Page types and the scope drawer; the drawer's type selector with its source and reason; the Graph screen's type and card editor; the templates editor without `pageKind`; the Policies screen, its route and the Sites picker removed; copy, labels, screen tests, `routes.json`.
- Commits per screen, `feat(settings): page types and their scopes`, `feat(pages): a page's type`, `feat(graph): the entity card and the page type`, `refactor(templates): no policies and no page kind`.

##### 2a.13 — the harness (agent H)
- `cmd/postulator/harness*.go`: types and scopes seeded, a scope with a custom key; `TestTheSeededSiteCarriesPageTypes`; walk routes.
- Commit `test(harness): the seeded site carries page types`.

##### 2a.14 — the e2e proof (agent H)
- `internal/e2e/pagetypes_test.go` `TestThePageTypesFollowTheSite` per the exit list, a meta key set through `wpCLI`; `standModel` gains the expected types.
- Commit `test(e2e): page types follow what each stand shows`.

##### 2a.15 — the review and the gate
- A fresh whole-branch review briefed with these decisions; its findings fixed or recorded; the whole gate of the global constraints; docs: `STATUS.md` (2a done, 2b next), `DECISIONS.md` (stage 2a), the spec (16.1, 16.5, 16.6, 16.7, 16.8), `CONTRACTS.md`, `ARCHITECTURE.md`, `CLAUDE.md`.
- Commit `docs: stage 2a, page types and their scopes`.

### 2b — one read layer: the writer's pack and the agent's picture

- **Goal (amended 2026-10-19).** The writer and the in-app agent share one root, the site model, and one layer of read functions over it, with two views: the writer's budgeted, prioritised pack per page and field, sent with every writing call; and the agent's site brief at a conversation's start plus one query tool for depth. Neither view computes anything the other computes differently. The agent's read tools over the model move here from stage 4.
- **Touches.** `internal/domain/{contextpack,content,llm,finding,field,pagetype}`, `internal/application/{sitemodel,graph,agent,tools,content,runs}`, `internal/runtime/{budget,claim}.go` and `steps/{generate_body,generate_meta,generate_fields,repair_links,insert_links,validate,judge,ceiling,prompts,publish,publish_product,revert,preflight,resolve_context}.go` with `prompts/*.tmpl`, `internal/adapters/sqlite` (`Store.Read`, reads by ids, `FindingRepo.Settle`, migration 0064), `internal/adapters/wp` (`PostType.Supports`), `internal/transport/{agent,wails}`, `cmd/postulator`, `internal/e2e`, `frontend/src/features/{settings,pages,graph,templates,links,sites,imports,reports,agent,runs}`, `frontend/src/copy`.
- **Migrations.** 0064, the conversation's stored brief, and 0065, a link's href form (`page_links.absolute`, from the closing review); stage 3's move up by two (from 0066).
- **Steps.** `generate_fields` joins `StepName`; `generate_body`, `generate_meta`, `repair_links` and the judge keep their names and artifacts and read the pack.
- **Tools.** `sites_explore` (views `page`, `tree`, `links`, `gaps`, `pack`) replaces `pages_tree` and `reports_link_audit_page`; `graph_propose_cards`.
- **Exit.** Below, per part.

#### 2b tasks

**Status, 2026-10-19: Part 0 and A.1 to D.3 done** (`c566586` to the review's fixes; the commits by part and what each exit proved are in `STATUS.md`). The closing review's findings (the client's text lost with rewrites off, relinks rewriting untouched blocks, a card accepted unseen, the link form, twins) are closed under "What the whole-branch review found" in `DECISIONS.md`. **The stand legs wait for Docker**: `TestThePageTypesFollowTheSite`, `TestTheWriterSeesTheFullPicture`, the capture and with it `TestThePackOfEveryStandPage`'s goldens, `TestTheAgentSeesTheSite` and `e2e:full` on each stand compile, vet and lint, and run at the owner's gate.

**Part 0 lands 2a's tail; Parts A to D are the stage.** Execution: `superpowers:executing-plans`, one accountable owner for every untagged Go package, at most two agents at once and only on disjoint packages (F: `frontend/`; H: `cmd/postulator` and `internal/e2e`). Every commit uses `git commit -- <paths>` with `gofmt -l` checked inside; edits go through Write and Edit; every gate command runs under `set -o pipefail` with its exit code read; a test that pins an old value moves in the same commit; a commit that regenerates `vocab.ts` carries the frontend copy, labels and icons it needs; F regenerates the bindings only at the owner's commit points; one stand up at a time and one heavy leg at a time, and a leg the memory guard or `0xc0000142` stops is listed for the owner, never retried in a loop.

**Decided at the plan** (the owner's answers, then the review's; recorded in `DECISIONS.md` at the close, the spec amended in the same commits):
1. **2a's tail is Part 0** (the owner): 2a.11 to 2a.14 land first, each committed and green on its own boundary; the heavy gate legs run once, at 2b's close, and one whole-branch review covers both.
2. **`generate_fields` asks only for the fields `publish` can write today** (the owner): an enabled, AI-written text field with a writer, which in 2b is the excerpt where the page's post type supports one (`wp.PostType.Supports`, read from `/wp/v2/types` in edit context and decoded raw, its values being booleans or arrays). Every preset has the excerpt off, so by default the step asks for nothing and says so; stage 3 widens the set as its writers land. An `excerpt` scope is refused on the Product preset, `post_excerpt` being the product's short description, which stays in `generate_body`'s product answer until stage 3. Only the raw excerpt is read (the rendered one is WordPress's auto-excerpt); `publish` sends it and records it as current and base; its artifact keeps what it replaced; a revert puts that back and refuses an excerpt the client changed since.
3. **With `rewrites` off the client's blocks are kept byte for byte and our sections go after them** (the owner): client blocks are the current body's blocks whose hash is not in `page_blocks_written`; the new part goes after the last client text block, else before the first dynamic block; with `rewrites` on the body is written whole. A run switches `rewrites` on for chosen pages only from the run-start dialog with its confirmation (autonomous mode confirms nothing), and the switch is kept in the run's params.
4. **Our blocks are recorded.** `publish` and `publish_product` record in `page_blocks_written` the block hashes of the content the site stored (read back through the raw route), so a second run replaces our first article instead of keeping it as client text. With client text kept, the body is written through `PutRaw` against the `page_bodies` current hash the writer read; a conflict pauses the item; without the plugin such a page pauses and says why.
5. **The checks read the right document**: our part carries no H1; `Structure` reads our part; `Compliance` and the owed links read the whole body, so a client link to the parent satisfies the owed one.
6. **A single-value field** (title, SEO title and description, excerpt) of an existing page whose scope is disabled or not AI-written is neither generated nor sent; with `rewrites` off a non-empty value we did not write is kept; a scope's length limits, when set, replace the template's for that field in the prompt and the checks. A planned page's creation still sends its title.
7. **Name twins live in `content.PlanLinks`** over `pagemap.Index`, so its five callers agree: a regular page and a term page of one name (title, H1 or term name under `graph.Key`) are one node; a target on the archive moves to the hub; a target that becomes the page itself is dropped and duplicates merge with `Required` OR-ed; the hub's pack lists its twin's products (from `page_terms`, never the archive's paginated grid) when its own listing is smaller; the brief counts the pair once.
8. **The link form**: `LinkTarget.URL` stays the path; a new `Href` in the site's form is used only by the splice and the prompt; an explicit `linkUrlForm` wins; `auto` follows the majority of the site's raw contextual links that are not ours, else the renders' `url_form`, else relative.
9. **A summary** is the SEO description, else the excerpt, else the body's first sentence (shortcodes skipped), else a sentence built from the page's structure (its type, H1, trail, listed products and children); it names its source; no model call.
10. **The pack** (`contextpack.Build`, `Fit`): spec 16.7's sections and priorities, "category" reading as the hub and catalog roles; every section under a fixed heading with its token count; `Fit` trims the lowest-priority list from its tail (products and siblings by shared keywords, then score) and drops a section only when its first item does not fit; priority 1 is never dropped. In a run the pack is built over the claim (spec, type, scope, link context), so the prompt cannot owe what `Compliance` never checks.
11. **The budget**: `llm.inputBudget.{writer,editor,linker,judge}` in tokens, 0 meaning the model's context window less the output ceiling, the system prompt, the answer schema and the reasoning allowance; `domain/llm.Tokens(text)` is `max(words × 2.2, runes / 3)` and `runtime/budget.go` calls it; the estimate prices each page's fitted pack; a budget below priority 1 is refused in preflight. The model-info reader is a required `steps.Deps` field.
12. **The reads**: `contextpack` is pure over a snapshot; `application/sitemodel` is the only loader and the only caller of its read functions; a writer's pack reads the page's neighbourhood by ids inside one read-only transaction (`Store.Read`); the brief, the tree and the gaps read the whole site once per call; an in-memory list pages through `Keyset.Position`, then `Cut`. `contextpack` imports `content`; `content.Brief` gains plain context sections and never imports `contextpack`.
13. **The agent's brief** is built once per conversation and stored (0064), rendered into the instructions after their static part, its bytes capped by a test; the volatile counts stay in the per-turn situation message; the cache key is `chat:<mode>:<siteId>`.
14. **`sites_explore`** belongs to the `sites` group and the tools sent up front; its `tree` and `links` views carry every field `pages_tree` and `reports_link_audit_page` answered, which go; its `pack` view is fitted to the tool-result cap. Every field has a description, the closed ones enums; every commit moving the schemas logs the bytes.
15. **Card proposals**: `graph_propose_cards` takes the agent's own text and is write risk; in confirm mode the person's approval of the call is the acceptance and applies the cards; in autonomous mode it files an `entity_card_proposed` finding per entity (Info; options `accept`, `ignore`) whose subject carries the proposal and the hash of the card it was made against; accepting refuses a card changed since; `FindingRepo.Settle` resolves and dismisses; `graph.AcceptCard` is the handler stage 4's `findings.Resolve` dispatches to; the list is cursor-paged.
16. **The seeds** join `seed/superseded` before `generate_fields` enters their recipes, so an untouched built-in is refreshed.

##### 0.1 — the bindings (2a.11)
- `TemplatesService`: `ListPageTypes`, `CreatePageType`, `SavePageType`, `DeletePageType`, `SetScope` over `application/pagetypes`; `PagesService.SetType`; the page view's type with source and reason, the graph view's canonical page type; `internal/app/build_services.go`; the bound-method count from 120 to 126 in `docs/CONTRACTS.md`.
- Commit `feat(wails): page types, scopes and a page's type are bound`.

##### 0.2 — the screens (2a.12, agent F)
- Settings → Page types and the scope drawer; the drawer's type selector with source and reason; the Graph screen's card editor and type; the templates editor without `pageKind` and `linkRules`; the Policies screen, its route, the Sites picker and Linking's policy meter removed; entity kinds out of graph, reports, imports and pages; the agent's cards and tool families without policies and with `pagetypes`; copy, labels, screen tests, `routes.json`.
- Commits per screen, as 2a.12 named them.

##### 0.3 — the harness (2a.13, agent H)
- `cmd/postulator/harness*.go`: types and scopes seeded, a scope with a custom key, the former harness link rules as a scope edit; `TestTheSeededSiteCarriesPageTypes`; walk routes.
- Commit `test(harness): the seeded site carries page types`.

##### 0.4 — the e2e proof (2a.14, agent H)
- `internal/e2e/pagetypes_test.go` `TestThePageTypesFollowTheSite` per 2a's exit list, on each stand in turn.
- Commit `test(e2e): page types follow what each stand shows`.

##### 0.5 — 2a's close
- `task build` with no drift, `npm run typecheck`, `npm run test:run`, `golangci-lint run`, `go vet`, `gofmt -l .`, the comment check; docs: `STATUS.md` (2a landed), `DECISIONS.md` (stage 2a completed), the spec (16.1, 16.6), `CLAUDE.md` (the ceiling footgun's figures).
- Commit `docs: stage 2a, page types and their scopes`.

##### A.1 — `domain/llm.Tokens` and `domain/contextpack`
- `domain/llm`: `Tokens(text string) int`; `runtime/budget.go` calls it.
- `domain/contextpack`: `Snapshot` (pages, `Index`, `pagetype.Set`, the graph, identities, terms, fields, current bodies, renders, links, open findings, settings, `pagemap.Site`); `Facts(snapshot, pageID)` with the identity and type, the card, the trail (ancestors with summaries), children and siblings with what each covers, listed products with their facts, terms, keywords with volumes, anchors, link targets, links in and out by type, fields with their scopes, the regions' summary, notes and open findings; `Summary`, `LinkForm`, `Tree`, `Links`, `Gaps`, `Brief`.
- Tests: the summary table (decision 9), the link-form table (decision 8), facts over a fixture snapshot with a hub, its twin category, products, children and siblings.
- Commit `feat(contextpack): the read functions over the site model`.

##### A.2 — `Build` and `Fit`
- `Pack`, `Section` (name, priority, heading, items, tokens), `Build(facts, field, scope)`, `Fit(pack, budget) (Fitted, error)` per decision 10, `Fitted.Dropped`.
- Tests: the priorities; the `Fit` goldens (siblings drop before the trail, products before keywords, a list trimmed from its tail, priority 1 kept, a budget below it refused).
- Commit `feat(contextpack): the writer's pack and its fit to a budget`.

##### A.3 — twins and the link form in the link plan
- `content.PlanLinks` per decision 7, `content.Subject` carrying the form, `LinkTarget.Href`, the splice and `InsertTarget` writing `Href`; the five callers unchanged in signature.
- Tests: `TestATargetPrefersTheHubOverItsTwinArchive`, `TestATwinThatIsThePageItselfIsDropped`, `TestOurLinksFollowTheSitesContextualForm`, the existing link tables green.
- Commit `feat(content): a hub stands for its twin category and our links follow the site's form`.

##### A.4 — `application/sitemodel`
- `internal/adapters/sqlite`: `Store.Read` (a read-only transaction), `ListByIDs` on the page model's repositories; `internal/application/sitemodel`: its readers, `Snapshot`, `Pack(PackRequest{PageID, Field, Budget, Claim})`, `Brief(siteID)`, `Explore(ExploreRequest)`, paged per decision 12; `internal/app/build_*.go`.
- Tests: a pack over the claim, the neighbourhood read, the paging of `gaps` to its end with no repeat.
- Commit `feat(sitemodel): one read layer over the site model`.

##### A.5 — the stand goldens (agent H for the capture)
- `internal/e2e/capture_test.go` under `E2E_CAPTURE=1`, run first on a fresh stand: each stand's normalised reader rows into `internal/application/sitemodel/testdata/<stand>/`; `TestThePackOfEveryStandPage` goldens every page's pack per enabled field.
- Commit `test(sitemodel): the pack of every page of both stands`.

##### B.1 — the input budget
- `llm.inputBudget.*` in `adapters/llm/tuning/settings.go` and the frontend's settings layout; `steps.Deps.Models` (required); the harnesses; `writerCeiling` beside the budget; the estimate per decision 11; preflight's refusal.
- Commit `feat(llm): an input budget per role`.

##### B.2 — the prompts around the pack
- `content.Brief` with the pack's sections; `generate_body`, `generate_meta`, `repair_links`, the judge; the templates rewritten with every section under its heading, the stable part in the system prompt and a cache key only where that prefix reaches the provider's minimum; `TestEveryPromptExecutes`.
- Commit `feat(steps): the writer, the linker and the judge read the pack`.

##### B.3 — the field's rules and the client's text
- Decisions 3 to 6: the scope's instruction, limits and policy as the field's rules; the kept client blocks; `page_blocks_written` recorded; `PutRaw` under the read hash; the checks over the right document; `StartRequest.Rewrite` in the run's params.
- Tests: `TestTheClientsBlocksAreKeptWhenRewritesIsOff`, `TestTwoRunsLeaveOneCopyOfOurSections`, `TestAClientEditAfterTheReadPauses`, `TestARunSwitchesRewritesOnForChosenPages`, `TestADisabledFieldIsNeitherWrittenNorSent`.
- Commit `feat(steps): the writer keeps the client's text unless its scope says otherwise`.

##### B.4 — `generate_fields`
- Decision 2; `wp.PostType.Supports`; `vocab.ts`, the blank recipe, the four tool enum tags, the seeds and `seed/superseded`, the ceiling with its bytes.
- Tests: `TestGenerateFieldsAsksOnlyForWritableFields`, `TestTheExcerptIsWrittenAndReverted`, `TestAnExcerptScopeIsRefusedOnAProduct`.
- Commit `feat(steps): generate_fields answers the fields publish can write`.

##### B.5 — links the policy allows
- `Sanitize(fragment, allow)` keeping a model link the policy and the plan allow; `insert_links` and `repair_links` placing what is still owed; `Compliance` and the audit owing contextual links only.
- Tests: `TestSanitizeKeepsTheLinksThePolicyAllows`, `TestAWriterLinkCountsAsPlaced`.
- Commit `feat(content): the writer may link where the policy allows`.

##### B.6 — the e2e proof (agent H)
- `internal/e2e/writer_test.go` `TestTheWriterSeesTheFullPicture` on each stand: a generate run over a planned child and one over `/peptides/`; the recorded request carries the trail, the hub's listed products, the field's instruction and policy, a current value and an empty target's built summary; `/lab-guides/` keeps its paragraph byte for byte; a paragraph edited through `wpCLI` after a sync pauses the run; the excerpt written and reverted on vanilla's posts.
- Commit `test(e2e): the writer sees the full picture`.

##### C.1 — the brief
- Migration 0064; the conversation's brief built once from `sitemodel.Brief` (types with counts, the top two levels of the hub tree, content health, link health, orphans, findings by code); `chat.tmpl`'s instructions; the cache key; the byte cap test.
- Commit `feat(agent): a site brief at the start of every conversation`.

##### C.2 — `sites_explore`
- The tool per decision 14; `pages_tree` and `reports_link_audit_page` removed with the frontend's tool labels; `groups.go`; the registry list; the ceiling with its bytes.
- Commit `feat(tools): sites_explore, the site model for the agent`.

##### C.3 — the system prompt
- The model's vocabulary (types and roles, trails and parent reasons, link types, field scopes) and when to use each view.
- Commit `feat(agent): the assistant reasons with the site model`.

##### C.4 — card proposals
- Decision 15: `finding.EntityCardProposed` and the option `accept`; `FindingRepo.Settle`; `graph.ProposeCards`, `AcceptCard`, `DismissCard`, `CardProposals`; `graph_propose_cards`; `task vocab` with its copy.
- Commit `feat(graph): the agent proposes entity cards a person accepts`.

##### C.5 — the agent's proof (agent H)
- `internal/e2e/agent_test.go` `TestTheAgentSeesTheSite` on the replica over the scripted fake: the brief names the hub tree and the gaps; a `sites_explore` call per view answers from the model (a hub's trail, its listed products, its links by type, the orphans); a proposed card leaves the entity unchanged until accepted.
- Commit `test(e2e): the agent sees the site`.

##### D.1 — "What the writer will see" (agent F)
- `PagesService.WriterPack` (127 methods); the Pages drawer's pane per field, section by section as `Fit` left it, with what was dropped for budget.
- Commit `feat(pages): what the writer will see`.

##### D.2 — card proposals and the rewrite switch (agent F)
- `GraphService.CardProposals`, `AcceptCard`, `DismissCard` (130 methods); the Graph screen's proposals, edited and accepted; the run-start dialog's rewrite switch with its confirmation.
- Commit `feat(graph): card proposals on the Graph screen`.

##### D.3 — the harness and the walk (agent H)
- Harness fixtures for the pane and the proposals; `routes.json`.
- Commit `test(harness): the writer's pack and the card proposals`.

##### G — the review and the gate
- A fresh whole-branch review over 2a's tail and 2b; the whole gate of the global constraints; docs: `STATUS.md` (stage 2 done, stage 3 next), `DECISIONS.md` (stage 2b), the spec (16.1, 16.7, 16.8), `CONTRACTS.md`, `ARCHITECTURE.md`, `CLAUDE.md`.
- Commit `docs: stage 2b, the full picture for the writer and the agent`.

## Stage 3 — editing: block operations, field writes through the registry, the pipeline over existing pages, revert

- **Goal (amended 2026-10-20).** A run improves the client's live pages without ever damaging them: it writes every enabled field through the registry, each under the hash of what it read; it edits a body block by block, keeping the client's blocks and everything the site injects; it never sends anything on the deny list; a revert puts back exactly what each field and block held. A draft-mode run writes nothing to an existing page until a person accepts its diff.
- **Touches (amended 2026-10-20).** `internal/domain/{block,field,pagetype,run,pagemap,content,template}`, `internal/application/{edit,pagetypes,runs,tools,pages,observe,imports,sitemodel}`, `internal/runtime/steps/*` with a new `edit_writers.go`, `internal/adapters/{wp,wp/wptest,sqlite}` (migrations 0066 to 0069), `wp-plugin` (1.5.0), `docker/e2e` (the replica's restricted user), `internal/e2e`, `internal/adapters/wp/e2e`, `cmd/postulator` (the harness), `frontend/src/features/{runs,pages,settings,agent}`, `frontend/src/copy`.
- **Migrations.** 0066, the seeded types' default templates where a type has none and its site no default; 0067, the template's `product.specifications` stripped from stored specs and overrides (the attribute filling retired); 0068, the old `pages` columns that copied the site dropped (`drift`, `content_hash`, `wp_link`, `wp_slug`, `wp_status`, `wp_title`, `wp_h1`, `canonical`), the `title` and `h1` field rows backfilled from `wp_title` and `wp_h1` first, the down restoring the columns; 0069, `page_links.origin` dropped, the down restoring it from `ours`. The plan's `title`, `h1`, `meta_title` and `meta_description` stay (the owner, 2026-10-20).
- **Steps.** None added. `publish` becomes the field writer, one write per execution, and `publish_product` folds into it; `generate_fields` widens with each writer; `generate_body`'s product answer loses the short description and the specifications; relinks place through `PlaceLink`; `revert` works per field and per block.
- **Tools.** None added. `pages_replace_links` goes; `runs_revert` takes its own arguments, so the per-field revert is a person's; `pages_update`, `pagetypes_delete`, `pagetypes_set_scope`, `imports_apply` and `pages_delete` gain a per-call confirmation, which changes no schema; `product.specifications` leaves `TemplateSpec` and `canonical` leaves the page update request. `schemaCeilingBytes` falls with each.
- **Exit.** Below, per part.

#### 3 tasks

**Status, 2026-10-23: stage 3 is closed.** Part 0 to D and G are landed; the gate is partly run, and the legs the machine's memory stopped are listed for the owner in the session-4 checkpoint and in `STATUS.md`. The checkpoints below say what each session did.

**Part 0 lands the owner's rulings on stage 2's open items; Parts A to D are the stage, each ending on a committed, green boundary.** Execution: `superpowers:executing-plans`, one accountable owner for every untagged Go package and the plugin, at most two agents at once and only on disjoint packages (F: `frontend/`; H: `cmd/postulator`, `internal/e2e`, `internal/adapters/wp/e2e`, `docker/e2e`). Every commit uses `git commit -- <paths>` with `gofmt -l` checked inside; edits go through Write and Edit; every gate command runs under `set -o pipefail` with its exit code read; a test that pins an old value moves in the same commit; a commit that regenerates `vocab.ts` carries the frontend copy, labels and icons it needs; F regenerates the bindings only at my commit points; one stand up at a time and one heavy leg at a time, and a leg the memory guard or `0xc0000142` stops is listed for the owner, never retried in a loop; a zombie test process is rerun alone first. Each ruling's commit carries its `DECISIONS.md` entry and the spec's amendment. The plan was reviewed by an Opus critic before the owner saw it; its three blocking and twelve serious findings are folded into the decisions below.

**Decided at the plan** (the owner's answers of 2026-10-20 first):
1. **The six rulings** are Part 0, each pinned by a test that fails first; ruling 2's review and ruling 5 stand on the durable record of decision 7, which lands first.
2. **The plan's four columns stay** (the owner): `title`, `h1`, `meta_title`, `meta_description` are the workbook's plan for a page, which no model row holds; the site's values live in `page_fields`, the H1 in a new read-only field `h1` (`field.H1`: read from the plugin's `h1` by the observation, never scoped, packed or written). Spec 16.1 is amended.
3. **In draft mode every write to an existing page waits for review** (the owner): `publish`, `relink_page` and `relink_neighbors` plan their writes without sending them, keep the plan as an `edit_plan` artifact (decision 22's diff) and pause the item `awaiting_review`. **Only a person accepts it**: the runs service refuses an agent's accept of an item awaiting review, and `runs_retry_step` says so. The acceptance records the hash of the plan accepted (`publish.accepted`), which the step clears when the plan is written; an acceptance never overrides `client_changed`. On acceptance the step re-plans the fields not yet written and writes when the plan equals the one accepted; when anything moved the item pauses for review again with the new plan, never writing. `publish.compare` no longer expects a status. A draft-mode product run is no longer refused in preflight: it waits for review like any page. **A page we created as a draft is not on the site for any link plan** (`target_not_published` for the neighbours, `relink_page` and the writer's own targets alike). A revert is exempt: it puts back. `generate_images` over an existing page in a draft run uploads nothing and says so (`images_wait_for_live`), an upload being a write to the site. A page we create as a draft stays one until a person publishes it in WordPress, an existing page's status never being written.
4. **The attribute filling is retired** (the owner): the store owns a product's attributes, `attributes` has been denied since 1b, and `wc/v3` replaces the whole list. The template's `product.specifications`, its prompt and its answer go; the seed's current copy joins `seed/superseded` first; 0067 strips the key from stored specs and overrides, which `merge.go`'s strict decode would otherwise refuse. A revert of an older run still puts back what that run added, through a revert-only `RestoreAttributes` that no publish path can reach and that runs only while the attributes are what that run left.
5. **The registry** is `application/edit`: it declares `CoreWriter` (title, excerpt, the featured image, a media item's alt and title), `StoreWriter` (short description, a product's first image, a variation's description, upsell and cross-sell ids, `meta_data` by key) and `PluginWriter` (the raw body, the SEO title and description, post meta by key), and `Service.Write` refuses a denied or unknown key (`field_denied`), and the title of a product page (the store owns the name), dispatches by `field.Spec.Write` and answers each field's outcome (written with the stored hash, moved, refused). `runtime/steps/edit_writers.go` adapts `*wp.Client`, as `observe_readers.go` does. **The adapter refuses again**: `wp.UpdateItem` loses `Slug`, `Status`, `Parent`, `Categories`, `Tags`, `MenuOrder`, `Template` and `Meta` (a creation keeps what it needs on `CreateItem`; meta goes only through `PUT /meta`); a denied meta key is refused by the client before it leaves; `wp.UpdateProduct` and the new `UpdateVariation` carry only writable fields. **A plugin write needs the plugin's `guarded_write` capability** (1.5.0): against 1.4.1, whose SEO route ignores unknown keys and whose raw PUT answers no content, every update write through the plugin is refused as `plugin_outdated`.
6. **Every write goes under the hash of what it read, and every guard is checked before the first write.** The body through the raw route's expected hash; a meta key and an SEO field through the plugin's expected hash per key, the routes answering 409 for a moved value; the title, excerpt, short description, variation description, upsells, cross-sells, the featured image and a media item's alt and title, which core and WooCommerce cannot check, are read just before the write and refused when they moved, the window being one request. Before the first write `publish` reads every live value it will write over, the client-block guard and the save stability (decision 8); a moved value pauses the item with `client_changed` naming the field and nothing is written; the note says to sync and regenerate. A value that moves between that check and its own write pauses the same way, the fields already written staying recorded and revertible.
7. **The durable record: one write per execution.** The engine replaces a step's artifacts and merges its checkpoint at every settle, and a process that dies between a PUT and its settle loses the Result. So `publish` (and each relink) makes one write per execution and returns `TransitionWait` waking at once, carrying the checkpoint (`publish.field.<key>`: the outcome, the hashes before and after) and the whole `publish_result` so far; every return path (a wait, a pause, an error) carries both, so a pause at the third field keeps the first two revertible. A rerun treats a field whose live value is what it was about to send as written. Order: the core and store fields first, the plugin's meta and SEO fields next, the body last under the hash read after those saves (a product: the store's fields first and the description last, as today). A retried publish never calls a model (the answers live in `draft`, `meta` and `fields`). A creation sets `publish.creating` before its one call, so the retry of a lost answer claims the page it finds by slug, when `block.SameRender` agrees, as this run's (`Created` true, so a revert trashes it instead of emptying it).
8. **Save stability** (review focus 3): a whole-post save under a user without `unfiltered_html` runs kses over the stored content again (a core title or excerpt update, the raw PUT and a store save all do), so writing a title could strip the client's iframe or form. The manifest says whether the user has `unfiltered_html`, and the raw GET answers `saveStable` (the stored content passes the current user's `content_save_pre` filters unchanged, `balanceTags` included). Over a page whose content is not save-stable, no write that re-saves the post is sent: the item pauses `save_alters_client`, saying the site's user needs `unfiltered_html` or the page must be edited by hand. Meta and SEO writes, which never re-save the post, still go. The replica gets a role with an editor's capabilities but `unfiltered_html`, and a user with an application password in it.
9. **With `rewrites` on, the client's text blocks are rewritten and nothing else** (the owner): our article takes the place of his first text block, his other text blocks go, and every non-text block (a shortcode, a dynamic, foreign or media block, markup with no text) stays where it stands byte for byte; a group block holding any non-text inner block counts as non-text and is kept. This amends 2b's decision 3, which wrote the body whole.
10. **The block operations** live in `domain/block` on `Body` and return a new body: `Replace`, `InsertAfter`, `Delete`, `RewriteRange`, `PlaceLink`. An editor block keeps its delimiters and attribute JSON and has the classes its attributes imply reapplied (`align` gives `has-text-align-*`); a block whose attributes cannot be kept is foreign and refused (`block_not_editable`). A block added to an editor body is one `core/html` block (2b), and an operation never leaves an editor body without a delimiter (deleting its last delimited block is refused), so it never parses back as classic. In a classic body blocks are separated by a blank line: `InsertAfter` adds one, and `Delete` takes the block with its own trailing separator, never fusing its neighbours. Nested inner blocks are not addressable; the outer block is the unit. **Consumers:** `InsertAfter` composes the kept body and the neighbour's plain sentence; `RewriteRange` the rewrite; `Replace`, `Delete` and `InsertAfter` the revert; `PlaceLink` the relinks: `content.InsertTarget` chooses the block and the anchor over the whole document (its rules intact) and `PlaceLink` writes the link into that block's bytes alone (`block` imports `content`, which never imports `block`). `Splice` stays the oracle the property test compares a relink with.
11. **What the site stored is ours.** The raw PUT answers the stored content beside its hash (plugin 1.5.0), and `page_blocks_written` is recorded from that answer at once, so the hashes are those of what the site holds after its filters, and no second read can race an edit.
12. **The canonical** (ruling 3): `seo_canonical` becomes scopable by a person only (Settings → Page types), never AI-written, off on every preset, absent from `pagetypes_set_scope`'s enum and from the pack (`vocabgen`'s `packable`); `generate_meta` no longer answers one; an update never sends it. A creation writes it only when its scope enables it and the page goes out live, as the `link` WordPress answered; a draft creation writes none, its link being `?page_id=`. A revert puts back an earlier canonical only where an older run's artifact recorded writing one and the live value is still what that run wrote.
13. **The agent's switches** (ruling 4, and what the critic found beside it): a tool may name a per-call confirmation (`Tool.Confirm`), which the registry applies in every mode unless the call is approved, with its own summary: `pages_update` naming a type whose scope rewrites; `pagetypes_delete` reassigning pages to one; `imports_apply` typing existing pages into one; `pagetypes_set_scope` enabling a field or its AI writes on a scope that rewrites; the summary names the pages and says their own text will be rewritten by the next run. `pages_delete` with `onSite`, which trashes a page and so writes its status, always waits for a person the same way.
14. **A relink's lost answer** (ruling 5) is decision 7 applied per page: the checkpoint (`relink.page.<id>`: the hash read, the hash composed, the links placed) is settled before the write; a rerun that finds the composed hash on the site records the body, our blocks and our links and places nothing; one that finds the read hash writes again; anything else is the client's and is reported.
15. **Our links are ours** (ruling 6): a link we place inside the client's paragraph sits in a block that stays his, so block ownership cannot tell it. `adopt` marks `ours` on every link the write placed (its target and anchor) and every link in a block we wrote, and the observation carries `ours` forward by target and anchor from the rows it replaces, as it carried `origin`, so the mark holds while the link stands. The rule lives in `block` (`pagemap` cannot import `block`: `block` imports `content`, which imports `pagemap`).
16. **The default templates** (ruling 1): `pagetype.Preset.DefaultTemplate` names the built-in (Hub, Guide, Product); `Seed`, and `Create` without a template, take the global built-in of that name when it exists (picked as `seed.go` picks it: global, by name, the oldest first). 0066 backfills a writable type whose template is null and whose site has no default template, because the type's template would otherwise win over a default a person chose; it runs once, so a type a person later sets to none keeps none. **0066 is the backfill** because Part 0 lands first and goose refuses a number below one the owner's database has applied.
17. **Revert per field and per block**: a field we wrote whose live hash is still ours gets its before value back, under the expected hash where the route has one; a field changed since is handed back as `revert_field_kept` naming it, never overwritten. The title is put back; the SEO meta is now checked against what we wrote (it was put back blindly); a status is never touched. A body is put back per block: each block we changed is found among its neighbours by block hash and restored, the client's later edits elsewhere kept; a block he changed since is handed back. A page we created is trashed only while it is still what we wrote, else handed back. A second revert of a run may put back the fields the first left; one already put back is refused. A `publish_result` written before stage 3 reverts whole, as today. `runs.RevertRequest` gains `pageId` and `fields` for a person; the tool keeps `{runId}`.
18. **Writes that stood outside the registry**: the featured image becomes a catalog field (`featured_image`, fill-only: written only where the page has none, so its before value is none and a revert removes ours while it is still ours), and a product's first image the same through the store; the Open Graph title and description follow their SEO field's guard (written only where empty or equal to what we wrote), their before values kept.
19. **A media item's alt and title** are written only for an attachment whose parent is the page, so an image shared by other products is never renamed for one, and only where empty or ours unless the scope rewrites; nothing names the program.
20. **`Index.ByName`** reads the `title` and `h1` fields and the plan's title and H1. The observation writes the `h1` field, and 0068 backfills the `title` and `h1` rows from `wp_title` and `wp_h1` before it drops them. Every reader of `Observed`, `Drift` and `ContentHash` moves before 0068: `observe/reconcile.go`'s `hasPlan` and `merge` onto the identity and the bodies, `pagemap`'s `twin.go`, `parent.go` and `product.go` onto the fields, the rest as listed in C.2.
21. **`record` and `adopt` write only what they own** (a narrow repository update each, as `UpdateDerived` is), never a whole page row, so an edit made during a run is not lost.
22. **The diff** is kept by `publish` (as `publish_result`, and as `edit_plan` while it waits for review): per field the key, the value and hash before and after and whose it was; per block the before and after lists with kind, hash and owner (ours, the client's, added, removed). The screens read only that.
23. **No schema grows** in stage 3: decisions 12 and 13 change none, `runs_revert` keeps its own `{runId}`, and the bytes `product.specifications`, the page update's `canonical` and `pages_replace_links` free lower the ceiling in their own commits.

#### Checkpoint, 2026-10-20 (end of the first session; the next session resumes at C.2)

**Landed on `dev`, each green on its packages, the tree clean at `4f27f48`:** 0.0 `c366e79`; 0.1 `d1d845a` (0066); 0.2 `5e4d93e`; 0.3 `b77bb05`; 0.4 `1a5126b`; 0.5 `1a2c257`; 0.6 `50a4b96`; 0.7 `4d5762f`; A.1 `3b4e1a4`; A.2 `87c12f8`; A.3 `8b17868` (plugin 1.5.0); A.4 `5e223da`; A.5 `b706208`; A.6 `649513c`; A.7 `0c130b2` (stand run owed); B.1 `76fb97f` (0067); B.2 `df2cc4d`; fix `2f33379` (a wait without a wake idled a 5 s sweep); B.3 `210bee8`; fix `be120dd` (a store save's hash is checked, not taken as base); B.4 `56dd853`; B.5 `009f32d`; B.6 `3ae007d`; B.7 `828d771`; B.8 `388ce72` (stand run owed); C.1 `4f27f48`. Each has its entry in `DECISIONS.md` under stage 3.

**Next, in order:** C.2 (0068, `Observed`/`Drift`/`ContentHash`/`Canonical` and every reader, the twins' matcher in `pagemap/twin.go` onto the fields as `ByName` already is; the page update's canonical goes, lowering the ceiling), C.3 (0069, `origin`, `pages_replace_links`), C.4 (the block e2e), D.1–D.4 (screens: the diff, revert per field through `RevertRequest{pageId, fields}`, Settings, the harness), G (review, gate, docs, the final report in the brief's nine-item order). At most two agents, disjoint packages; F owns `frontend/`, H owns `cmd/postulator`, `internal/e2e`, `internal/adapters/wp/e2e`, `docker/e2e`.

**For the final report (what the session's ledger held):**
- Schema bytes: before 91 tools / 63,107 (ceiling 63,120); Part 0 unchanged; B.1 62,174 (ceiling 62,187); C.1 unchanged (`runs_revert` keeps `{runId}` through its own `revertArgs`).
- Client-ownership paths closed, each failing first: a relink linked inside a shortcode (A.1); a rewrite dropped the client's non-text blocks and wrote our H1, the featured image and the Open Graph fields were overwritten, the SEO meta went unguarded, an edit after the read was written over, an unstable page was re-saved, an outdated plugin failed mid-run, the block record raced a second read, `record` wrote the whole row (A.5); a relink re-rendered the client's paragraph and appended its sentence inside it (A.6); a client's description edit between the check and a store or variation save was written over (`be120dd`). Open for G: `generate_images` draws and uploads over an existing page that already has a featured image.
- Deviations: four migrations 0066–0069; `seo_canonical` scopable by a person only; the diff shapes landed in 0.3; draft-mode neighbours stand down for an unpublished target; tool calls tagged as the agent's; the title scope off by default; decision 10's "classes reapplied" dropped (no operation rewrites markup); meta keys compared ASCII-folded and refused unless `[A-Za-z0-9_-]{1,255}`; A.7's stale-hash proof is the plugin's stand tests and a wptest publish test, the app-level stand test being the restricted user's; the Open Graph twin follows its SEO field; B.2 asks the short description only where scope and template both enable it, as escaped paragraphs; `writerCeiling(spec, attempts)`; the judge always shows a product's PRODUCT section; B.3's estimate keeps `generate_fields` at 1,024 output tokens while a call grows 256 per variation; a second revert of a field already put back writes it again as itself (no separate "refused"); the body undo is byte-exact while the body is untouched and block-exact otherwise (whitespace between blocks may differ).
- Stage 4 must know: a step returning `TransitionWait` to continue sets `WakeAt` (`deps.now()`), or it idles a sweep; run `go test -tags uiharness -timeout 15m ./cmd/postulator/...` after any product-path change (B.1 missed its attribute assertion).
- Windows flakes, green alone: openai `TestAClientWithoutATimeoutWaits`; wp `TestTheSocksProxyDialerCarriesTheConnect` under load; wp and wptest dial timeouts in parallel, so run adapter packages `-p 1`.
- Gate legs owed (none run on a stand this session; memory allows one stand at a time): `go test -count=1 -p 1 ./...` (stopped once by the memory guard; optionally after `wsl --shutdown`); per stand, `task plugin:deploy STAND=<stand>` then `E2E_STAND=<stand> go test -tags e2e -count=1 -run 'TestMetaWritesOnlyWhatTheUserMayEdit|TestTheRawWriteAnswersWhatWasStored|TestSaveStabilityFollowsTheUsersFilters|TestSEOMetaWritesUnderTheHashesRead|TestManifestDescribesTheSite|TestMetaReadsOnlyTheKeysAsked' ./internal/adapters/wp/e2e/`, then `E2E_STAND=<stand> go test -tags e2e -count=1 -timeout 30m -run 'TestAProductIsEditedThroughTheRegistry|TestTheProductLoopEditsWhatTheClientCreatedInTheStore' ./internal/e2e/`; on the replica, `TestTheSiteFiltersAreRespectedUnderARestrictedUser`; then the whole G gate.

#### Checkpoint, 2026-10-21 (end of the second session; the next session resumes at D.1)

**Landed on `dev`, the tree clean at `fb77a22`:**
- **The stand proofs (Part 1):**
  - On the replica, the six plugin 1.5.0 proofs went green at once.
  - `TestTheSiteFiltersAreRespectedUnderARestrictedUser` went green after two fixes:
    - `76bc5e1`: the test read the plugin off the create, which probes nothing.
    - `bd4bc4d`: a page the site's user may not `edit_post` (WordPress's privacy page, for an editor) stopped the whole walk.
  - `TestAProductIsEditedThroughTheRegistry` went green after two fixes:
    - `205d8fb`: its setup.
    - `e2080f7`: the field diff recorded what was sent, so the revert took our own variation description, which WooCommerce answers through `wc_format_content` in every context, for the client's edit.
  - `TestTheProductLoopEditsWhatTheClientCreatedInTheStore` was green.
  - On vanilla, all of it was green; the restricted test skips there by design.
- **The image step (Part 2):** `7fe0c6c`.
- **C.2:** four commits instead of one, plus the review's fix:
  - `d60922e`: the site's title and H1 are read from the fields.
  - `acb9f61`: drift is the body's move from its base.
  - `7dd11a3`: the plan carries no canonical.
  - `18f5a2a`: migration 0068, with the writer registering the driver's `sha256`.
  - `ae39852`: the review's fix. A neighbour is put back only while it holds what its relink left (`NeighborResult.After`).
- **C.3:** `770ae24` (0069; `pages_replace_links` and `PagesService.ReplaceLinks` gone, 129 bound methods).
- **C.4:** `fb77a22`. `TestAnExistingPageIsEditedBlockByBlock` compiles, vets and lints, but **no stand has run it**: the memory guard stopped the stand's start, and the half-started stand was taken down.
- Each item has its entry in `DECISIONS.md` under stage 3.

**Schema bytes now:** 90 tools, 61,343 bytes, ceiling 61,356 (from 62,174 after B.1: C.2c's canonicals took 141 bytes, C.3's tool 690).

**Next, in order:**
1. The owed commands below.
2. D.1 to D.4.
3. G.

At most two agents, on disjoint packages. This session ran its C.2 and C.3 through one agent at a time, which kept the owner's context.

**Commands owed (none ran after C.2):** one stand up at a time, one heavy leg at a time, optionally after `wsl --shutdown`.
- **The Go suite in groups.** It stopped once in the adapters group before C.2, and the domain, application and runtime groups were green then.
  - `go test -count=1 ./internal/kernel/... ./internal/domain/...`
  - `go test -count=1 ./internal/application/...`
  - `go test -count=1 ./internal/runtime/...`
  - `go test -count=1 -p 1 ./internal/adapters/...`
  - `go test -count=1 ./internal/app/... ./internal/transport/... ./cmd/... ./frontend/...`
  - `go test -count=1 -tags uiharness -timeout 15m ./cmd/postulator/...`
- **Per stand** (`task e2e:up STAND=<stand>`, then `task e2e:down STAND=<stand>`):

  ```
  E2E_STAND=<stand> go test -tags e2e -count=1 -timeout 45m -run 'TestAnExistingPageIsEditedBlockByBlock|TestTheSiteFiltersAreRespectedUnderARestrictedUser|TestAProductIsEditedThroughTheRegistry|TestTheProductLoopEditsWhatTheClientCreatedInTheStore' ./internal/e2e/
  ```

  This also re-proves Part 1 over C.2 and C.3, which changed the observation and the records.
- **`task lint:e2e`**, alone.
- **G's whole gate.**

**For the final report (beyond the first session's list):**
- **Client-ownership paths closed this session**, each failing first:
  - the image step uploaded over a page's or product's own image;
  - a revert handed back our variation description as the client's (formatting store);
  - a page the site's user may not edit stopped every observation;
  - a neighbour revert would have trusted a base taken after the relink (introduced by C.2b, closed in `ae39852`).
- **Open for G:**
  - `relink_page`/`relink_neighbors` record the body they sent, not what the site stored (decision 11 holds for publish only). A filtered relink then reads as drift, and its blocks as the client's.
  - The `sync_back` read-back takes the site's body as ours on both sides, so a client edit in the seconds after our write becomes the base.
  - Our mark is carried forward by target and anchor, so a client's later link with the same target and anchor in a block of his reads as ours.
  - The SEO fields keep `writtenMeta` and are not set from what the plugin stored.
  - A rerun after a lost answer over a formatting store pauses rather than taking the write as ours.
- **Deviations:**
  - C.2 landed as four commits.
  - `pagemap.FieldNames`; the named index in every link plan, the link audit and the judge included.
  - The read layer loads the whole site's fields once. CLAUDE.md's "a writer's pack loads only the page's neighbourhood" no longer holds for fields; correct it in G, together with CLAUDE.md's stale ceiling numbers.
  - `Mismatch` lives in `steps`, with lowercase JSON (older artifacts still decode).
  - The pages DTO lost `observed`, `mismatches`, `contentHash` and `canonical`, and kept a derived `drift`.
  - 0068's down refills `content_hash` and `drift` only for pages we wrote.
  - 0069 marks `generated` rows `ours` before the drop, and its down adds `origin` with a default.
  - The link owner labels in the UI ("Written by a run", "The client's own").
  - The image estimate stays an upper bound.
  - `wptest` answers a variation's description formatted, as the store does.
- **Stage 4 must know:** `observe.ClosedToUser` is how an adapter says a page is closed to the site's user; the walk counts such pages in `closed`.

#### Checkpoint, 2026-10-22 (end of the third session; the next session resumes at D.1)

**Landed on `dev`, the tree clean at `c8aef9c` plus this checkpoint's commit:**
- **Part 1, the owed runs.** `TestAnExistingPageIsEditedBlockByBlock` green on the replica as written (43 s) and on vanilla after a fix to the test itself (`a43a1db`: a parsed block's raw bytes run to the next block's start, so the client's last block gains the separator when ours follows it; the test compares each client block's own bytes, whitespace around it aside), 27 s, the revert byte-exact on both. The adapters group (`go test -count=1 -p 1 ./internal/adapters/...`) green on the idle machine, 30 packages; `TestResolveWaitsOnTheRateLimitBeforeEveryHop` fails only while docker's bootstrap loads the machine (300 ms deadlines, wall clock). The uiharness suite green after the link change (223 s).
- **Part 2, the client-ownership items.** `462696c` a relink records what the site stored (`block.Align`, `relinkWrite.landed`) and keeps off a page its save would alter (`ReasonNeighborSaveAlters`; the fake's `WithSaveFilter`); `69ad2c3` a revert records what the site stored; `09c307c` `sync_back` keeps the base of what the run wrote (`bodyRecorder.Observe`, a `body_changed_on_site` warning in the sync result); `384f829` our mark is tied to the block that holds it (migration 0070 `page_links.block_hash`, `pagemap.PageLink.BlockHash`, `block.PlacedKey`/`Placed`/`Moved`/`Lineage`/`HeldOurs`, `adopt` takes an `adoption`, legacy rows keyed once to the one block of the last stored body); `81f9c6c` CLAUDE.md's ceiling and read-layer lines.
- **Part 3, the closing review** (a fresh critic: one blocking, five serious, five minor; all under "What the closing review found (session 3)" in `DECISIONS.md`): `e70db08` a textless block of ours pairs by its render; `395d702` the save-stability probe covers the title and the excerpt, **plugin 1.5.1**, the client requires it (proved on the replica: the guarded contract tests, the manifest, the restricted-user loop); `984a0c5` an older result's SEO meta is put back only while it holds what that run wrote; `507a0fe` the excerpt's base is the stored value; `8c1e568` a revert gives `page_blocks_written` back (`block.Restored`); `117f581` the SEO record is what the plugin stored (`storedSEO`, the fake's `WithSEOSanitized`); `c8aef9c` a lost answer keeps the client's text on record and the revert names the body it cannot prove (`ReasonRevertBodyUnproven`). Kept with their reasons: a variation's formatted put-back, `PublishResult.Blocks` as the composed diff (open, below), the legacy keying's one-block rule, the title's `&amp;` mismatch noise.
- **Schema bytes:** unchanged, 90 tools, 61,343 bytes, ceiling 61,356 (no tool changed). **Bound methods:** 129. **Migrations:** to 0070. **Plugin:** 1.5.1.

**Next, in order:**
1. D.1 to D.4 as planned (agent F for `frontend/`, H for the harness).
2. The one open review item: recompute `PublishResult.Blocks` from the stored body after the raw write (the proof: a save filter reformatting ours plus a client edit in another block, revert, our block gone).
3. The gate as G lists it, with `task plugin:deploy STAND=<stand>` before any stand leg (the plugin moved to 1.5.1; vanilla's stand still holds 1.5.0 until its next `e2e:up` or deploy).
4. The docs: `STATUS.md` (stage 3 done, stage 4 next), the spec (16.1, 16.2, 16.3, 16.6, 16.7, 16.9), `CONTRACTS.md` (`saveStable` from 1.5.1 covers the title and the excerpt; `page_links.block_hash`), `ARCHITECTURE.md`, CLAUDE.md's standing rulings (stage 3 in one line), and the final report in the brief's nine-item order.

**Commands owed (gate legs not run this session; one stand up at a time, one heavy leg at a time):**
- Per stand, after `task e2e:up STAND=<stand>` and `task plugin:deploy STAND=<stand>`:
  ```
  E2E_STAND=<stand> go test -tags e2e -count=1 -run 'TestMetaWritesOnlyWhatTheUserMayEdit|TestTheRawWriteAnswersWhatWasStored|TestSaveStabilityFollowsTheUsersFilters|TestSEOMetaWritesUnderTheHashesRead|TestManifestDescribesTheSite|TestMetaReadsOnlyTheKeysAsked' ./internal/adapters/wp/e2e/
  E2E_STAND=<stand> go test -tags e2e -count=1 -timeout 45m -run 'TestAnExistingPageIsEditedBlockByBlock|TestTheSiteFiltersAreRespectedUnderARestrictedUser|TestAProductIsEditedThroughTheRegistry|TestTheProductLoopEditsWhatTheClientCreatedInTheStore' ./internal/e2e/
  task e2e:full STAND=<stand>
  ```
  This session ran, after its fixes, the guarded contract tests, the manifest test and the restricted-user loop on the replica, and the block test on both stands before Task 4; the rest re-proves Tasks 2–4 and the review's fixes on a stand.
- `go test -race -count=1 -p 2 -covermode=atomic` in the five groups and `go run ./cmd/covergate`; `task ui:lint`; `npm run typecheck` and `npm run test:run`; `task build` with no generated drift; `task e2e:full:noplugin STAND=vanilla`.

**For the final report (beyond the earlier sessions' lists):**
- **Client-ownership paths closed this session**, each failing first: a relink recorded the composed body and blocks (kses and a `rel`-adding filter turned ours into the client's); a relink wrote over a page its save would alter; a revert recorded the body it sent; `sync_back` took a client's edit seconds after our write as the base; our link mark carried by target and anchor to a link the client added; a second relink into the same paragraph handed the first link to the client; an inserted sentence shifted a placed link's record; our figures dropped from `page_blocks_written` across a relink; a whole-post save ran kses over the client's title and excerpt while the probe called the page stable; an older run's revert wrote over an SEO title the client edited since; the excerpt's base was the sent value; a revert left `page_blocks_written` naming the reverted run; the SEO record was the sent value; a lost answer left the client's text off the record and a revert claimed success.
- **Deviations:** the plugin's `saveStable` contract moved, so the client's minimum is 1.5.1; `bodyRecorder` gains `Observe`; `adopt` takes an `adoption{raw, last, placed, moved}`; `recordWritten` takes the stored meta, not the artifact; `relinkWrite` gains `placed` (an older state decodes without it and its links are keyed as legacy marks); `block.Align` pairs the client's blocks by render and the rest by text in order, a textless block by its render; the plan's `TestTheLinkFormCountsTheClientsLinksOnly` never existed (`TestOurLinksFollowTheSitesContextualFormInTheSnapshot` is the one); `TestPageLinkRepoReplaceForPage` carries the block hash instead of a new round-trip test.
- **Stage 4 must know:** a link's row carries the hash of the block it sits in when it is ours; a legacy row with none is keyed once against the last stored body; `sync_back` reports a body difference as a step finding and the next `sync_site` raises the persistent one; the revert's `ReasonRevertBodyUnproven` and `ReasonRevertMetaEdited` are hand-backs a person resolves.

#### Session 4, 2026-10-22 (the stored diff, per-field acceptance, D, the gate, the docs)

The owner chose per-field acceptance of a draft run's review, built in Go before D. A Fable critic read the session's plan (one blocking, six serious, five minor findings, folded in).
- [x] **1. The stored diff** (`929613b`): `block.Landed`, `OwnerSite`, `recordOurBlocks` from the landed diff; the open review item is closed.
- [x] **2. Per-field acceptance**: `RetryStepRequest.fields` (a person's), `runs_retry_step`'s own arguments (schema unchanged, 61,343 bytes), `Engine.Accept(itemID, fields)`, `run.ClearAcceptance`, `PageEdit.body`, publish narrowed after the gate, `PublishResult.Declined`, a relink's review accepted whole.
- [x] **3. What the screens read**: `observe.ModelView.fields` (`FieldView{key, writtenRunId, writtenAt, ours}`), `pagetypes.Field.aiWritable`, `blockOwners` and `blockChanges` in `vocab.ts`; `task build` green, 129 methods. Not recorded anywhere, so no screen shows it: where a site-injected region sits among the body's blocks (a region knows only whether it is inside the content root).
- [x] **D.1–D.3** (agent F: `e22429d`, `2fb1d7d`, `7ba5389`; 1,790 vitest tests in 170 files) and **D.4** (agent H: `f268454`; the harness suite green in 609 s beside vitest). A field revert of a page the run created no longer trashes it (found from D.2).
- [x] **Found by the gate and fixed:**
  - `8acd0ec`: our H1 was written into an existing page with no client text (the replica's main hub);
  - `958cefe`: the product proof's wire log hid a request per route;
  - `6441006`: the client loop cancelled on the plan's wait, before any write.
- [x] **G, the gate: green on both stands (2026-10-23).**
  - The first run was stopped twice by the memory guard.
  - The second run, after the memory freed, ran every leg: the five race groups, covergate (88.84% / 88.14%), `task build` with no drift, the frontend legs, `ui:lint`, `e2e:full` on both stands from fresh resets, vanilla's contract and exit tests, and `e2e:full:noplugin`.
  - It found two things: the restricted-user proof's cleanup (`a7e5d68`), and a revert without the plugin claiming the body back.
- [x] **Docs:** CONTRACTS (`6aa1c1b`), the spec's part 16 (`8621635`, `6aa1c1b`), ARCHITECTURE and CLAUDE.md's stage 3 ruling (`a9e41e6`), STATUS and this plan.

**Deviations in session 4:**
- Per-field acceptance was built in Go (the owner's answer), with a task of its own.
- `PageEdit.body` and `PublishResult.declined` are new shapes.
- `OwnerSite` is a third block owner.
- The plan's `TestARelinkReviewIsAcceptedWhole` became a case of the service's refusal table.
- `TestADeclinedFieldIsNeverRecordedAsOurs` asserts that no record reaches `Fields.Record`, instead of running a second publish.
- The created-page guard lives in the revert step alone.
- `block.Ours` stays, but only for the guard's recognition of an earlier write.

**Stage 4 must know:**
- A review's acceptance may carry fields: `run.CheckpointAcceptFields`, and `run.ClearAcceptance` clears both keys.
- A publish result's `blocks` are the stored ones, `site` being an owner.
- `observe.ModelView.fields` names the run behind each field.
- A page we create still carries its H1 in the content (open with the owner).
- A field's revert outcome has no vocabulary.
- The harness restore puts back the model's bodies.

##### 0.0 — the plan
- This task list. Commit `docs(plan): stage 3's task list`.

##### 0.1 — the seeded types' templates (ruling 1)
- Decision 16: `domain/pagetype` `Preset.DefaultTemplate`; `application/pagetypes` `Seed` and `Create`, its template reader gaining the global built-in by name; migration 0066 with its round trip; the suites lose `templateTheTypes` (agent H).
- Tests: `TestASeededTypeCarriesItsPresetsTemplate`, `TestATypeCreatedFromAPresetInheritsItsTemplate`, `TestTheBackfillFillsOnlyATypeWithoutATemplate`, `TestTheBackfillLeavesASiteWithADefaultTemplate`, `TestAFreshSiteStartsARunOnAuto`.
- Commit `feat(pagetypes): the seeded types carry their preset's template`.

##### 0.2 — the durable record
- Decision 7 over today's writes (the raw body, the core update, the SEO meta), one per execution, the checkpoint and the whole `publish_result` on every return; `field.Change` and `block.Change`, the diff's shapes (decision 22). No behaviour a person sees changes.
- Tests: `TestAPublishPausedAfterItsFirstWriteIsStillRevertible`, `TestAWriteWhoseAnswerWasLostIsRecognisedOnTheRerun` (the execution's whole Result discarded after the PUT, the step run again from the last settled checkpoint), `TestACreationWhoseAnswerWasLostIsStillOurs`.
- Commit `refactor(publish): one write per execution, each recorded before the next`.

##### 0.3 — an existing page's status is never written, and a draft run waits for review (ruling 2)
- Decision 3: no status on an update and none in `compare`; the page row's status from the identity; `run.PauseAwaitingReview`, `run.ArtifactEditPlan`; `publish`, `relink_page` and `relink_neighbors` in draft mode over an existing page; the person-only acceptance with the plan's hash; the draft target rule in the link plans; the product preflight; `generate_images`; `vocab.ts`; the frontend's `primaryAction` offering Accept for `awaiting_review`, with its copy (the diff itself is D.1). One commit, so no build ever writes a draft run's content live.
- Tests: `TestNoStatusLeavesOnAnUpdate` (a recording transport: a live run, an accepted draft run, a post and a product), `TestADraftRunWritesNothingToAnExistingPage`, `TestAnAcceptedReviewWritesAndTheItemCompletes`, `TestAReviewAcceptedAfterTheSiteMovedWaitsForReviewAgain`, `TestTheAgentCannotAcceptADraftReview`, `TestAnAcceptNeverWritesOverAClientChange`, `TestADraftRunCreatesAPlannedPageAsADraft`, `TestADraftRunsNeighboursWaitForReview`, `TestNoLinkPlanTargetsADraftWeCreated`.
- Commit `fix(publish): an existing page's status is never written, and a draft run's writes wait for review`.

##### 0.4 — the canonical (ruling 3)
- Decision 12: `field.CheckScope`, the presets, `pagetypes` refusing AI writes on it, `vocabgen`, `generate_meta`'s answer and prompt, `publish_seo`, the revert's put-back.
- Tests: `TestTheCanonicalIsNeverWrittenToAnExistingPage`, `TestALiveCreationTakesItsPermalinkAsTheCanonical`, `TestADraftCreationWritesNoCanonical`, `TestACanonicalScopeIsNeverAIWritten`.
- Commit `fix(seo): the canonical is written once, as the permalink WordPress answered`.

##### 0.5 — the agent's switches (ruling 4)
- Decision 13: `tools.Tool.Confirm`, the registry's `Build` and `Call`, the five tools.
- Tests: `TestTypingAPageIntoARewritingTypeWaitsForAPerson` (confirm and autonomous modes), `TestReassigningPagesIntoARewritingTypeWaitsForAPerson`, `TestAnImportTypingPagesIntoARewritingTypeWaitsForAPerson`, `TestEnablingAFieldUnderRewritesWaitsForAPerson`, `TestTrashingAPageOnTheSiteWaitsForAPerson`, `TestTheConfirmationSaysThePagesOwnTextWillBeRewritten`; the schema bytes logged, unchanged.
- Commit `fix(tools): the agent cannot switch a page into rewriting, or trash one, unseen`.

##### 0.6 — a relink's lost answer (ruling 5)
- Decision 14 in `relink.go`, `relink_page.go` and `relink_neighbors.go`.
- Tests: `TestARelinkWhoseAnswerWasLostRecordsItsBlocks`, `TestARetriedRelinkPlacesNothingTwice`, each for a page and a neighbour.
- Commit `fix(relink): a relink whose answer was lost is recognised on the rerun`.

##### 0.7 — our links are ours (ruling 6)
- Decision 15: the rule in `block`; `observe/content.go` and `steps/page_links.go`.
- Tests: `TestARelinkedClientParagraphsLinkReadsAsOursAfterASync` (the link ours, the paragraph not in `page_blocks_written`), `TestTheLinkFormCountsTheClientsLinksOnly`.
- Commit `fix(links): adopt records the links of our own writes as ours`.

**Part A — block operations and the field writers.** Exit: the property test; every operation on classic and block bodies, a `wp:paragraph` with `align` kept valid; the deny list refused at the use case and at the adapter; a recording transport proves no slug, parent, status, price, stock, SKU, category or variation attribute leaves on an update; a publish killed after its second field resumes at the third with no second model call; the PHP deny set in `openapi.yaml` includes `field.Denied`'s meta keys; review focus 3 on the replica under a role without `unfiltered_html`; `TestAWriteAgainstAStaleHashPauses` on both stands.

##### A.1 — block operations
- Decision 10.
- Tests: `TestSerializeKeepsEveryUntouchedBlock`, a property test over seeded `math/rand/v2` bodies (classic and editor, CRLF, nested groups, self-closing dynamic blocks, shortcodes, foreign elements) and random sequences of operations: every untouched block's bytes appear unchanged and in order, the body keeps its kind, and it parses back to the same blocks; a table per operation on classic and editor bodies; `TestAnAlignedParagraphStaysValid`; `TestABlockWhoseAttributesCannotBeKeptIsRefused`; `TestPlaceLinkAgreesWithTheSplice`.
- Commit `feat(block): operations that keep every untouched byte`.

##### A.2 — the catalog's new keys and the deny list's meta keys
- `field`: `H1` and `FeaturedImage` (decisions 2 and 18); the store meta keys behind each denied field (`_price`, `_regular_price`, `_sale_price`, `_sku`, `_stock_status`, `_stock`, `_manage_stock`, `_product_attributes`, the `attribute_` prefix, `total_sales`) as `field.DeniedMeta()`, beside the protected prefixes; `wp-plugin/openapi.yaml`'s deny set; `testdata/meta_deny.json`, one table both sides test, as the href table is.
- Tests: `TestTheContractsDenySetIncludesTheCatalogs`, `TestAMetaKeyIsDeniedAlikeOnBothSides`, `TestTheH1IsNeverScoped`.
- Commit `feat(field): the deny list names the store's meta keys`.

##### A.3 — the plugin 1.5.0
- PHP 7.4: `PUT /meta/{id}` (`edit_post_meta` checked per key; never an `is_protected_meta` key, the deny set or a non-scalar value; an expected hash per key and 409 naming a moved one; `wp_slash`; answers each key's stored value and hash; content posts only; which keys a scope names is the Go side's to decide); `/seo-meta/{id}` takes expected hashes; the raw PUT answers the stored content beside its hash; the raw GET answers `saveStable`; `meta_get` refuses the deny set too; the manifest's `meta_write`, `guarded_write`, `unfilteredHtml` and 1.5.0; `openapi.yaml`; `tests/meta_deny.php`; `task plugin:lint`, `task plugin:test`.
- `adapters/wp`: `PutMeta`, `SetSEOMeta` with expected hashes, `PutRaw` answering `RawWrite{Content, Hash}`, `Raw.SaveStable`, `requiredCapabilities` with the fake's defaults, the PHP `CAPABILITIES`, `openapi.yaml` and the three pinned lists; `wptest` serves the same as an independent implementation; `wp/e2e` `TestMetaWritesOnlyWhatTheUserMayEdit`, `TestTheRawWriteAnswersWhatWasStored` and `TestSaveStabilityFollowsTheUsersFilters` (agent H).
- Commit `feat(plugin): meta writes and writes under an expected hash, 1.5.0`.

##### A.4 — the writers and the use case
- Decision 5: `wp.UpdateItem` narrowed, `UpdateMedia`, `UpdateProduct`, `UpdateVariation`; `application/edit`; `steps/edit_writers.go`; `internal/app/build_*.go`.
- Tests: `TestADeniedFieldIsRefusedByTheUseCase`, `TestAProductsNameIsNeverWritten`, `TestADeniedMetaKeyIsRefusedByTheClient`, `TestNothingOnTheDenyListLeavesOnAnUpdate` (a recording transport over every writer), `TestAMovedValueIsNotWrittenOver`, `TestAnOutdatedPluginWritesNothing`.
- Commit `feat(edit): the field registry writes through three writers`.

##### A.5 — `publish` as the field writer
- Decisions 6 to 9, 11, 18 and 21: `publish.go` rewritten over `edit` on 0.2's durable record; the creation once with status, parent and slug recorded; `publish_seo` through the plugin writer; the featured image fill-only; `recordOurBlocks` from the PUT's answer; the client-block guard and the save stability before the first write over an existing page; the rewrite of text blocks only; the diff in `PublishResult`.
- Tests: `TestAPublishKilledAfterItsSecondFieldResumesAtTheThird` (the second execution's Result discarded after its PUT; no second model call), `TestEveryGuardIsCheckedBeforeTheFirstWrite`, `TestEveryWriteOverAnExistingPageIsGuarded`, `TestOurBlocksAreWhatTheSiteStored` (the fake alters a body on save), `TestAPageThatIsNotSaveStableIsNotResaved`, `TestAWriteAgainstAStaleHashPauses`, `TestRewritesKeepTheClientsNonTextBlocks`, `TestTheFeaturedImageIsNeverReplaced`, `TestTheOpenGraphFieldsFollowTheirGuard`.
- Commit `feat(publish): every produced field through the registry`.

##### A.6 — relinks place through `PlaceLink`
- Decision 10's relink consumer in `relink.go`, `relink_page.go` and `relink_neighbors.go`, on 0.6's record; the neighbour's plain sentence through `InsertAfter`.
- Tests: the relink tables green; `TestARelinkTouchesOnlyTheBlockItLinks`.
- Commit `refactor(relink): a link is placed in its block alone`.

##### A.7 — Part A on the stands (agent H)
- The replica's bootstrap: a role with an editor's capabilities but `unfiltered_html`, a user in it with an application password (`E2E_RESTRICTED_*` in `.env.replica`); `internal/e2e/edit_test.go`: `TestTheSiteFiltersAreRespectedUnderARestrictedUser` (the user is shown to lack `unfiltered_html`; a page with a client `<iframe>` block, a title-only run and a run that keeps the body either leave that block byte-identical or write nothing; a body the save alters is recorded by the stored hashes, and the next run keeps no copy of it as the client's) and `TestAWriteAgainstAStaleHashPauses` (both stands: the body, the title and an SEO field moved through `wpCLI` after the read).
- Commit `test(e2e): writes under the hash read and the site's own filters`.

**Part B — products through the registry.** Exit: `TestAProductIsEditedThroughTheRegistry` on both stands.

##### B.1 — the product through the registry
- `publish_product` folds into `publish`'s field writer; decision 4 with `seed/superseded`, 0067 and its round trip, `RestoreAttributes`; the ceiling lowered by the specifications' bytes.
- Tests: `TestAProductWritesTheStoreFirstAndTheDescriptionLast`, `TestAnOlderRunsAttributesAreStillPutBack`, `TestNoPublishPathSendsAttributes`, `TestAStoredSpecWithSpecificationsStillDecodes`.
- Commit `refactor(publish): a product is written through the registry`.

##### B.2 — the short description
- Out of `generate_body`'s product answer into `generate_fields` where the scope enables it; the template's `product.shortDescription` stays the field's brief under the scope's instruction and limits; the seeds.
- Commit `feat(fields): the short description is a field of its own`.

##### B.3 — variation descriptions
- One structured call per product for all its variations from the store's facts; `variation_description:<id>` through `StoreWriter.UpdateVariation`; a variation's attributes never sent.
- Commit `feat(fields): variation descriptions in one call per product`.

##### B.4 — image alt and title
- Decision 19, from the product's facts, through `CoreWriter.UpdateMedia`.
- Commit `feat(fields): a product image's alt and title`.

##### B.5 — custom meta by key
- An enabled, AI-written `meta:<key>` through the plugin's `PUT /meta` or the store's `meta_data`, as its scope's provider says.
- Commit `feat(fields): custom fields by key`.

##### B.6 — upsells and cross-sells
- The pack offers candidates (the same hub's products, then the graph's related entities' products); the model answers ids from them only; written as ids; off by default; a list we did not write is kept unless the scope rewrites.
- Commit `feat(fields): upsells and cross-sells from the pack's candidates`.

##### B.7 — `Index.ByName` on the fields
- Decision 20's first half: the `h1` field observed, `ByName` on the fields and the plan.
- Commit `refactor(pagemap): a crumb's name is matched against the fields`.

##### B.8 — the product proof (agent H)
- `internal/e2e/products_test.go` `TestAProductIsEditedThroughTheRegistry` on each stand: the description, the short description, a `meta_data` key and a variation description written and reverted (a variable product planted through the store's API where a stand has none, and removed after); a recording proxy proves price, SKU, stock, categories, status, slug and the variations' attributes never on the wire, and the store's reads before and after prove them unchanged, a variation's save re-syncing its parent; the test logs the fields written and the wire's keys.
- Commit `test(e2e): a product is edited through the registry`.

**Part C — revert per field and block, and the cleanup.** Exit: `TestAnExistingPageIsEditedBlockByBlock` on both stands.

##### C.1 — revert per field and per block
- Decisions 12 and 17 in `revert.go` and `revert_product.go`; `runs.Revert` with `pageId` and `fields`; `runs_revert`'s own arguments.
- Tests: `TestARevertPutsBackEachFieldItWrote`, `TestAFieldChangedSinceOurWriteIsHandedBack`, `TestARevertRestoresOurBlocksAmongTheClientsEdits`, `TestARevertOfOneFieldLeavesTheOthers`, `TestASecondRevertPutsBackWhatTheFirstLeft`, `TestTheSEOMetaIsNotPutBackOverTheClientsEdit`, `TestACreatedPageTheClientEditedIsHandedBack`, `TestAnOlderRunStillRevertsWhole`, `TestAnOlderRunsCanonicalIsPutBack`.
- Commit `feat(revert): per field and per block`.

##### C.2 — 0068: the read-through columns go
- Decision 20's second half: migration 0068 with its backfill and round trip; `pagemap.Observed`, `Mismatches` and `Page.Drift`, `ContentHash` and `Canonical` deleted, every reader moved (`observe/reconcile.go`, `pagemap/{twin,parent,product,index}.go`, `contextpack/snapshot.go`, `content/product.go`, `imports/views.go`, `observe/views.go`, `pages/views.go`, `graph/edges.go`, `steps/{publish,sync_back,relink_page,revert,wordpress}.go`); `observed.go` deleted; `pages.UpdateRequest` and the Pages drawer lose the canonical (agent F for the screen, in the same commit); the ceiling lowered by its bytes.
- Commit `refactor(pagemap): the page reads the model, not its old copies`.

##### C.3 — 0069 and `pages_replace_links` go
- Migration 0069 and its round trip; `page_links.origin` and `pagemap.Origin` gone, their readers on `ours`; `pages_replace_links`, `pages.ReplaceLinks`, `PagesService.ReplaceLinks` (130 to 129 bound methods) and their frontend card gone; `schemaCeilingBytes` lowered to the new bytes.
- Commit `refactor(links): our links are told by their mark alone`.

##### C.4 — the block proof (agent H)
- `internal/e2e/edit_test.go` `TestAnExistingPageIsEditedBlockByBlock`: on the replica, `/peptides/` (the injected breadcrumb and product grid) gets its body between them, untouched blocks byte-identical, links only in the content, a revert restoring the base; on vanilla, a nested page's body edited with its URL and parent unchanged; the test logs the hub's blocks before and after, with their owners, and after the revert.
- Commit `test(e2e): an existing page is edited block by block`.

**Part D — the screens** (agent F, the bindings regenerated at my commit points). Exit: `npm run typecheck`, `npm run test:run`, `task ui:lint` green.

##### D.1 — the diff per field and per block
- The run report's publish pane: each field before and after, and the body's blocks with ours, the client's, added and removed marked apart; the review pane of `awaiting_review` with Accept.
- Commit `feat(runs): the diff per field and per block`.

##### D.2 — revert per field
- From the run (fields chosen per page) and from the Pages drawer (each field a run wrote, with its revert).
- Commit `feat(pages): revert a field`.

##### D.3 — Settings, the drawer and the agent's cards
- The canonical scope switch without AI writes; upsells and cross-sells off by default; the short description under `generate_fields`; the new confirmations' cards; copy.
- Commit `feat(settings): the scopes stage 3 writes`.

##### D.4 — the harness (agent H)
- Fixtures for a publish result with its diff, an item awaiting review and a revert per field; `routes.json`.
- Commit `test(harness): the diff, the review and the revert per field`.

##### G — the client-ownership review and the gate
- One fresh review whose only question is: is there any path where a run, an agent tool or the image step changes text, a link, a status, a slug, a parent or a canonical that the client owns? Every finding closed with a test that fails first.
- The gate: every exit test on both stands; `e2e:full` on both stands; the migrations' round trips; `task plugin:lint` and `task plugin:test` on 7.4; `task build` with no generated drift; `go vet` and `golangci-lint run` on the plain, `uiharness` and `e2e` builds; `task lint:e2e` alone; `go test -race` in groups; `go run ./cmd/covergate`; the comment check; `gofmt -l .`; `npm run typecheck`, `npm run test:run`, `task ui:lint`.
- Docs: `STATUS.md` (stage 3 done, stage 4 next), `DECISIONS.md` (stage 3), the spec (16.1, 16.2, 16.3, 16.6, 16.7, 16.9), `CONTRACTS.md`, `ARCHITECTURE.md`, `CLAUDE.md`.
- Commit `docs: stage 3, editing the client's pages field by field and block by block`.

## Stage 4 — reconciliation with the workbook, findings, the agent's tools over the model, the screens

- **Goal.** A workbook reconciles with the site through persistent findings a person or the agent resolves; the site is one map; the agent works the whole loop through tools over the model.
- **Re-planned on 2026-10-23 into three sessions**, each a gated sub-stage like stage 1's:
  - **4.1, the matcher and the findings**: `MatchRow`, the workbook reconciled into persistent findings, their lifecycle and the resolution use case. No agent tools, no screens.
  - **4.2, the agent works the loop**: the agent's tools over findings, fields and blocks, `sites_update`, `pages_get` answering the model, and the agent resolving findings end to end on the fake provider.
  - **4.3, the screens, the gate and the phase report**: the Findings inbox, the Site map, the import preview naming each row's match rule, the full gate, and phase 14's closing report.
- **The format of each session** is stage 3's. A task list written at the session's start: decided-at-the-plan items, then numbered tasks, each with what it changes, the tests that fail first and its commit. It ends on exit tests on both stands and a checkpoint, which says what landed (with commits), what the next session starts with, and the exact gate commands still owed. 4.1's list is below; 4.2 and 4.3 write theirs at their own starts.
- **Reuses.** The import pipeline and its preview, `graph.Key` through `pagemap`'s `nameKey`, the observation's findings tracker, the run start and its estimate, the UI harness.

### 4.1 — the matcher and the findings (2026-10-23)

- **Touches.** `internal/domain/{finding,pagemap,render,field,importmap}`, `internal/application/{findings (new),imports,observe,sitemodel,graph}`, `internal/runtime/steps/{publish,kept,fields,mismatch,publish_seo,revert_field}.go`, `internal/application/edit`, `internal/adapters/{sqlite,wp/wptest}`, `internal/app/build_*.go`, `samples/build`, `internal/e2e/reconcile_test.go`, `frontend/src/copy` (the copy `vocab.ts` and the preview's rules need).
- **Migrations.** 0071, `page_renders.theme_h1` (Part 0's H1 rule); 0072, `pages.planned_trail`, `pages.parent_source` and `pages.row_key` (the workbook's hierarchy and a person's mapping).
- **Tools.** None; no schema moves (90 tools, 61,343 bytes, ceiling 61,356). **Bound methods.** None; the screens are 4.3's.
- **Exit.** Unit: the `MatchRow` table over the sample shapes (URL, slug, entity, title, H1, SKU, ambiguous, none, URL-less rows, a hub and a store category sharing a name); the lifecycle table; each resolution option per code. E2E, one stand at a time: `TestTheWorkbookReconcilesWithTheSite` (the replica: `samples/client-sheets.xlsx` with URL-less rows, matched by URL, slug and name, mismatches as findings, findings resolved through the use case; vanilla: the nested URLs match by slug). The agent's part of that test is 4.2's.

#### 4.1 tasks

**Decided at the plan** (the owner's answers of 2026-10-23 first; a Fable critic read the plan before the owner saw it, one blocking and nine serious findings folded in):
1. **Part 0 is the owner's answers to stage 3's open questions**, each pinned by a test that fails first: no second H1, adaptively; a reverted variation description in the store's formatted form is accepted; plain-text fields compare with entities decoded.
2. **A URL-less row that matches no page is planned under its sheet parent** (the owner): the slug of its title, else of its entity name, under the page its sheet trail's nearest level names, else at the site root; the one creation call sets that parent and slug.
3. **`MatchRow`'s rules**: `mapped` (a person's mapping, `pages.row_key`), URL (path and planned path), slug, entity name (parent and name), title, H1, SKU for product rows; every text through `nameKey`; on-site, unarchived pages through every rule first, then the map's planned pages, so a second import finds its own planned row; the twin rule on every rule. One candidate matches with its rule, several are `row_ambiguous` and write nothing, none is `row_without_page`.
4. **The expected trail** is a row's level names minus the last when that level names the row itself (the groups' `namedBy`/`sluggedBy` evidence); placeholders drop out; it is compared by name with the derived chain (front and technical pages left out, a twin the same node), and a product with the chains of its `product_cat` terms. One marshalled subject `{sheet, site}` for the import and the observation.
5. **The lifecycle**: a row a person or the agent resolved stays settled while its subject is unchanged, as a dismissed one does; a row the observation or an import closed reopens when raised again; `Clear` records `observed_clear`, `row_matched` or `row_left_sheet`; an import clears the open `row_*` of its sheet it did not raise this time.
6. **Resolution** (`findings.Resolve`): a model change in one unit of work, or a run option that settles nothing and answers the request of the normal start; no option writes to a site. `accept_site` on a field clears `written_run_id` and `written_at`, or the client's accepted value would read as ours and the next run write over it. `map_to_page` is remembered (`row_key`), and a planned page it replaces is deleted only after the finding moves onto the chosen page and settles, and only when no run item names it. `set_type` into a non-technical preset keeps the observation's technical detection off that page. `rewrite_field` runs the template's whole chain (publish requires the draft and the body HTML), narrowed by publish's `params.fields` before the review gate, and its estimate says so.
7. **The derived parent wins over a workbook parent unless a person set the parent source to the workbook** (amending 2026-10-18); the model parent it then keeps is never written to an existing page, and a planned page's one creation call still sends it.

- [x] **0.0 — the plan** (`cb76bc3`). This list; the tool-schema decision in 4.2 and in `DECISIONS.md`. Commit `docs(plan): stage 4 in three sessions, 4.1's task list`.
- [x] **P0.1 — no second H1, adaptively** (`8013230`). `render.Extract` records `ThemeH1` (`printed`: an `<h1>` outside a located root, or one inside it whose text the raw lacks; `absent`; empty when the root was assumed); 0071; `sitemodel` answers it through steps' `pictures`; one rule in `publish.planItem` and `kept.go`. Tests: `TestExtractSaysWhetherTheThemePrintsTheH1` (both stands' goldens `printed`, an unwrapped classic golden `printed`), `TestACreatedPageCarriesNoH1WhereTheThemePrintsTheTitle`, `TestOurH1OnlyWhereTheThemePrintsNone`, `TestACreationWithNoObservedRenderCarriesNoH1`; the full loop asserts no created raw holds `<h1`. Commit `fix(publish): our H1 only where the theme prints none`.
- [x] **P0.2 — the store's formatted variation description is accepted.** `wptest` formats as `wc_format_content` does; the revert records what the store answered. Test: `TestARevertedVariationDescriptionInTheStoresFormIsAccepted`. Commit `fix(revert): a variation's description is put back in the store's form`.
- [ ] **P0.3 — plain text compares with entities decoded.** `field.SameText`; `observeFields`, `keptFieldsOf`, `checkBeforeWriting`, `movedSince`, `refuseMismatch`, `edit.checked`'s value arm. Tests: `TestSameTextDecodesOnlyPlainFields`, `TestAnEscapedAmpersandIsNoChangeOnTheSite`, `TestAKsesEscapedTitleIsNoMismatch`. Commit `fix(fields): plain text compares with its entities decoded`.
- [ ] **4.1.1 — `pagemap.MatchRow`.** Decision 3; `MatchProduct` goes, `ClaimProduct` uses it. Test: `TestMatchRow`. Commit `feat(pagemap): one matcher for every workbook row`.
- [ ] **4.1.2 — the import matches through `MatchRow`.** A match pass before `pagesOf` gives every row an address (decision 2); `held` and `products.go`'s matcher go; the `sku` column; `PreviewPage.matchedBy` for every row with `candidates`; the copy. Tests: a URL-less row matched by title updates that page, an unmatched one is planned under its parent, a second import creates nothing. Commit `feat(imports): every row is matched by one rule, with or without a URL`.
- [ ] **4.1.3 — the workbook's hierarchy.** Every level column bound as a hierarchy fact (Root alone makes groups); decision 4; 0072; `pagemap.TrailDiffers`; decision 7 in `derive`; the preview's `parent_differs`. Commit `feat(imports): the sheet's levels are the row's expected trail`.
- [ ] **4.1.4 — findings that last.** `row_ambiguous`, `row_without_page`, option kinds; decision 5 in `FindingRepo` and the trackers; the import persists its row findings and `parent_differs`; the observation closes `row_without_page` and `parent_differs`. Test: `TestAFindingLivesOnce`. Commit `feat(findings): a finding lives once and closes when its cause goes`.
- [ ] **4.1.5 — `application/findings`.** `List` (cursor) and `Resolve`, decision 6; `imports.MapRow`; publish's `params.fields`; `core.Findings`; `vocab.ts` with its copy. Test: `TestEachResolutionOptionPerCode`. Commit `feat(findings): one use case resolves a finding`.
- [ ] **4.1.6 — the samples.** URL-less rows in `samples/build/client.mjs` (by entity, by title, none, a hub and a store category sharing a name); the pins move. Commit `test(samples): the client's sheets carry rows without a URL`.
- [ ] **4.1.7 — `TestTheWorkbookReconcilesWithTheSite`.** Commit `test(e2e): the workbook reconciles with the site`.
- [ ] **4.1.8 — the docs and the checkpoint.** Commit `docs: stage 4.1, the matcher and the findings`.

### 4.2 — the agent works the loop (planned at its start)

- **The tool-schema decision (2026-10-23).** The schemas stand at 61,343 of 61,356 bytes, and every byte is resent on every round of every turn. The nine tools the plan named (`pages_fields_get/set`, `pages_blocks_list/replace/insert_after/delete/place_link`, `findings_list/resolve`) fold into three, each with an `op` enum, the way `sites_explore` carries its views: `pages_fields` (`get`, `set`), `pages_blocks` (`list`, `replace`, `insert_after`, `delete`, `place_link`) and `findings` (`list`, `resolve`). Writes stay confirmable. `templates_resolve_for_page` folds into `pages_get`. Deferred tool loading stays off until the owner has tried a funded key.
- **Also.** `pages_get` answers the model; `sites_update` carries the per-site settings; the agent resolves findings end to end on the fake provider; the ceiling moves with the tools in the same commit.
- **Exit.** Tool schema sizes under the ceiling, the deferred round logged; the agent's part of `TestTheWorkbookReconcilesWithTheSite` on the fake provider.

### 4.3 — the screens, the gate and the phase report (planned at its start)

- The Findings inbox with each code's options; the Site map (hierarchy by reason, types, technical and read-only pages, link counts by type); the import preview naming each row's match rule; the agent's cards for the 4.2 tools; the harness.
- The full gate on both stands, then phase 14's closing report.

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
