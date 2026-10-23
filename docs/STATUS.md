# Status

The handoff point between sessions. Read this first. The reasoning behind every phase and
every ruling is in [`DECISIONS.md`](DECISIONS.md).

**Branch:** `dev`, the development branch; `master` takes a PR from it when the owner asks.
**Released:** `v2.3.0` on 2026-09-25 (2.2.0 and the 2026-09-25 work). **The 2026-10-02,
2026-10-03 and 2026-10-03/04 work is on `dev`**, not yet released; what its gate did and did not
run is under **The gate**. **The site-model redesign is underway**: stage 0 and the owner's
answers landed on 2026-10-17, stage 1a (the plugin's reads, its client and the two stands) the
same day, stage 1b (the page model, its storage and the observation) and **stage 1c (1b's gaps
closed and the screens) on 2026-10-18, which closes stage 1**; **stage 2 (2a, page types and
their scopes, and 2b, the read layer with the writer's pack and the agent's view) on 2026-10-19,
its stand legs green on both stands on 2026-10-20**; **stage 3, the field writers and the editable
body, is done (2026-10-20 to 23): Part 0 to C, G's safety work and plugin 1.5.1, then the stored
diff, per-field acceptance and the screens; its gate is green on both stands (2026-10-23). Stage 4
is re-planned as 4.1 (the matcher and the findings), 4.2 (the agent works the loop) and 4.3 (the
screens, the gate and the report); 4.1 and 4.2 are done (2026-10-23), green on both stands, with the
race and covergate legs owed; 4.3 is next.**

## Where we are

**2026-10-23, stage 4.2: the agent works the loop.** The reasoning is under "The owner's answers at
4.2's plan" and "4.2, decided while building" in `DECISIONS.md`; the task list, the checkpoint and the
commands owed are under 4.2 in the phase plan.

- **The owner's answers at the plan**: the agent's writes to a page's fields or blocks wait for the
  person in either mode, resolving a finding and starting a run follow the mode; a title scoped
  without the body is answered by one editor call.
- **Part 0** (agent): gaps only from a row that plans its page (`8c6d75d`), a CSV's rows keyed by its
  file (`c85378e`), a row named by its SKU has a key (`96a3874`), one `pagemap.TermParents`
  (`71ea057`).
- **The field-scoped run** (`112e73f`, `247124f`): the steps a scope does not need are disabled at the
  start, publish requires only its fields' drafts and writes no body without it, `generate_meta`
  writes from the page as it stands, the estimate prices the kept steps. A meta sweep over three
  pages: 5,309 tokens against 55,941 for the whole recipe.
- **The tools**: a call's risk from its arguments (`7dae806`); `findings` and `runs_start`'s `fields`,
  `rewrite` and estimate card (`d1978f9`); every agent write a run of the kind `edit`, migration 0073
  (`3c5e54e`, `85052a6`); `pages_fields` and `pages_blocks` (`57b5a91`); `pages_get` over
  `sitemodel.Page`, `templates_resolve_for_page` and the explore `page` view gone (`d1691f4`);
  `sites_update`'s detection settings (`111e9b2`).
- **The instructions** (`20700b9`) and **the stands** (`1fa2d1e`): the agent's part of the reconcile
  proof and `TestTheAgentEditsABlock` green on the replica, the block test and the reconcile proof on
  vanilla, with `TestTheWholeLoopReachesTheDockerSite` and `TestTheAgentSeesTheSite`.
- Schemas 65,722 bytes (92 tools, ceiling 65,735; 61,361 and 90 before); migrations to 0073; plugin
  1.5.1.

**2026-10-23, stage 4.1, second session: the findings last, and one use case resolves them.** The
reasoning is under "The owner's answers to 4.1's first session" and "4.1, the second session" in
`DECISIONS.md`; the checkpoint and the commands owed are under 4.1 in the phase plan.

- **The owner's answers** (`92eb616`): a run scoped to chosen fields is 4.2's; a row that names only
  an entity plans no page (`a480bbb`, failing first).
- **4.1.4**: the row codes, `pagemap.RowKey` and their words (`f34a492`); the import persists
  `row_ambiguous`, `row_without_page` and `parent_differs` and closes what it no longer raises
  (`0c5a6e0`); the observation closes a row's finding once its page is on the site (`a64f5da`).
- **4.1.5**: `application/findings` (`04caaf5`, `core.Findings`), `imports.MapRow` (`6e341ea`), a
  page a person typed as regular is never detected technical again (`5a0fbd2`), a run scoped to the
  fields publish writes (`9bc1f59`), a card accepted by the agent recorded as the agent's
  (`8aac371`).
- **4.1.6**: the client's sheets carry URL-less rows the replica answers (`4f4f62d`).
- **4.1.7**: `TestTheWorkbookReconcilesWithTheSite` (`d13ff6c`, `4116950`), green on both stands
  beside `TestTheWholeLoopReachesTheDockerSite`. On the replica the client's workbook updates 9
  pages (4 by URL, 2 by slug, 2 by title, 1 by entity name), plans 13, finds no ambiguous row and
  carries 2 entity-only rows; after one import and one sync the open findings are
  `row_without_page` 14, `technical_detected` 7, `parent_differs` 3, `parent_unresolved` 1 and
  `render_redirected` 1 (under "What the stands found at 4.1's close" in `DECISIONS.md`).
- **4.1.8**: spec 16.8 and the docs.
- Schemas unchanged (61,361 bytes, 90 tools); migrations to 0072; bound methods 129 (`StartRequest`
  gains `fields`); plugin 1.5.1.

**2026-10-23, stage 4.1, first session: the owner's three answers, the matcher, the workbook's
hierarchy.** The reasoning is under 2026-10-23 in `DECISIONS.md`. The task list, the checkpoint
and the commands owed are under 4.1 in `docs/superpowers/plans/2026-10-17-phase-14-site-model.md`.

- **Part 0, the owner's answers to stage 3's open questions:**
  - `8013230`: our H1 goes in only where the theme prints none, one rule for every page; the
    extraction records `theme_h1` (0071), `printed` on both stands.
  - `5cfdee2`: a reverted variation description is accepted in the store's form and recorded as
    the store answered it.
  - `58e067d`: the title, excerpt and SEO fields compare with their entities decoded, so kses's
    `&amp;` is no change, no pause and no finding.
- **The re-plan** (`cb76bc3`): three sessions. The nine planned agent tools fold into
  `pages_fields`, `pages_blocks` and `findings`, each with an `op` enum.
- **4.1 so far:**
  - `2e76024`: `pagemap.MatchRow`, one matcher (mapped, URL, slug, entity, title, H1, SKU).
  - `a1f62eb`: the import matches every row through it, with or without a URL. A URL-less row
    with no page is planned under the page its sheet trail names. The rows are `row_ambiguous`
    or `row_without_page` in the preview, and the preview names each row's rule.
  - `2f1813f`: the sheet's Category and Subcategory columns are the row's expected trail,
    compared with the site's into `parent_differs` (`{sheet, site}`), cleared when they agree. A
    person may pin the model's parent to the workbook.
  - `079f791`: migration 0072.
  - `b3e4f27`, `14e5cc2`: a finding a person settled stays settled while its facts stand, and
    each option is a model change or a run.
  - `7287d4e`: the resolution's writes.
- **Owed in 4.1:** the row codes persisted with their lifecycle, `application/findings`
  (`List`, `Resolve`), the samples' URL-less rows, `TestTheWorkbookReconcilesWithTheSite` on both
  stands, and the gate legs listed in the checkpoint.
- Schemas 61,361 bytes (90 tools, ceiling 61,374, from the `sku` column); migrations to 0072;
  bound methods 129; plugin 1.5.1.

**2026-10-20/21, stage 3 of the site model: editing the client's pages field by field and block by
block, Part 0 to C.** The reasoning is under **stage 3** in `DECISIONS.md`; the task list, both
sessions' checkpoints and the commands still owed are under stage 3 in
`docs/superpowers/plans/2026-10-17-phase-14-site-model.md`.

- **Part 0, the owner's rulings on stage 2's open items** (`c366e79` to `4d5762f`): the seeded types
  carry their preset's template (0066); an existing page's status is never written and a draft run's
  writes wait for a person's review; the canonical is written once, as WordPress's permalink; the
  agent cannot switch a page into rewriting, or trash one, unseen; a relink's lost answer is
  recognised; our links are ours by their mark.
- **Part A, block operations and the field registry** (`3b4e1a4` to `0c130b2`): `domain/block`'s
  operations keep every untouched byte; `application/edit` writes through a core, a store and a
  plugin writer under the hash read; plugin 1.5.0 writes meta and SEO under expected hashes and says
  whether a save is stable; `publish` writes every field through the registry, the body last.
- **Part B, products through the registry** (`76fb97f` to `388ce72`): the attribute filling retired
  (0067), the short description, variation descriptions, image alt and title, custom fields by key,
  upsells and cross-sells as fields.
- **Part C** (`4f27f48`, `d60922e` to `fb77a22`): the revert per field and per block; 0068 drops the
  pages' old copies of the site (the site's title and H1 read from the fields, drift as the body's
  move from its base, no canonical in the plan); 0069 drops the link origin and `pages_replace_links`
  goes.
- **What the stands found** (`76bc5e1` to `e2080f7`): a page the site's user may not edit stopped
  every walk; a revert took our own variation description, which WooCommerce formats on every read,
  for the client's edit. And the image step no longer draws over a page's or product's own image
  (`7fe0c6c`).
- **2026-10-22, G's safety work** (`462696c` to `c8aef9c`; the reasoning under "G's safety work" and
  "What the closing review found" in `DECISIONS.md`, the checkpoint under stage 3 in the phase plan):
  the owed stand runs are green (the block edit on both stands, vanilla after a fix to the test's own
  whitespace comparison; the adapters group on an idle machine); a relink and a revert record what
  the site stored and a relink keeps off a page its save would alter; `sync_back` keeps the base of
  what the run wrote; our mark on a link is tied to the block that holds it (0070) and moves only
  with our own changes to that block; the closing review's one blocking and five serious findings are
  closed, each failing first: the plugin's save-stability probe covers the title and the excerpt
  (**plugin 1.5.1**, required), the SEO record and the excerpt's base are what the site stored, a
  revert gives `page_blocks_written` back, an older run's SEO meta is guarded, a lost answer keeps the
  client's text on record, and our textless blocks survive a relink. Four minors are kept with their
  reasons; one (`PublishResult.Blocks` as the composed diff) is open for the next session.
- **2026-10-22/23, session 4** (`929613b` to `6441006`; the reasoning under "Session 4" in
  `DECISIONS.md`):
  - the publish result's blocks and `page_blocks_written` are what the site stored
    (`block.Landed`, owner `site`);
  - a person accepts a draft run's review field by field (`RetryStep.fields`; the agent's tool
    keeps its schema);
  - the page model answers the fields a run wrote, and the catalog whether a model may write each;
  - the screens: the diff per field and per block, the review with its ticks, revert per field
    from the run and the Pages drawer, the canonical as a person's switch, the confirmation
    summaries, the plugin's capability labels; the harness seeds them;
  - a field revert never trashes a page the run created;
  - our H1 is never written into an existing page with no text of the client's (found on the
    replica's main hub by the gate);
  - two e2e fixes: the cancelled-run sync point, and the product proof's wire log.
- The tool schemas stay at 61,343 bytes (90 tools, ceiling 61,356); bound methods 129; migrations to
  0070; plugin 1.5.1.

**2026-10-19, stage 2b of the site model: one read layer, the writer's pack and the agent's
view. Stage 2 is done, its stand legs green on both stands on 2026-10-20.** The reasoning is
under **stage 2b** in
`DECISIONS.md`, the task list under 2b in `docs/superpowers/plans/2026-10-17-phase-14-site-model.md`.

- **Part 0, 2a's tail** (`c566586`, `ffb6d9f` to `814c8a1`, `0bc4a98`, `3242f6a`, `219e9b4`): page
  types and scopes bound and on screen, policies and entity kinds gone from every screen.
- **The read layer** (`02d3c78` to `2e494a1`, `b6b6e08`, `7737de6`): `domain/contextpack` reads a
  snapshot of the model (a page's facts, its summary, the site's link form, the overview, gaps and
  tree) and builds the writer's pack, fitted to a budget by priority; `application/sitemodel`
  alone loads snapshots, a page's neighbourhood by ids inside one read-only `Store.Read`, and reads
  the link audit through `reports`. Name twins and the site's link form live in `PlanLinks`.
- **The writer** (`327c065`, `984af71`, `b016f50`, `a08f9ce`, `6abd188`): `generate_body`,
  `generate_meta`, `repair_links` and the judge read the pack under `llm.inputBudget.<role>`,
  refused in preflight when too small; with `rewrites` off the client's blocks are kept byte for
  byte and ours added after them; a run may rewrite pages the person confirms at its start;
  `generate_fields` writes the excerpt; `Sanitize` keeps a writer's link the policy allows.
- **The agent** (`fef0134`, `62569de`, `42ce38b`, `905a895`, `f138c89`): a conversation on a site
  opens with the site's brief, kept per conversation (0064, cache key `chat:<mode>:<siteId>`);
  `sites_explore` (views page, tree, links, gaps, pack) replaces `pages_tree` and
  `reports_link_audit_page`; the instructions teach the model's words; `graph_propose_cards`
  files card proposals a person accepts on the Graph screen.
- **The screens** (agent F, `dc79c7b`, `09b1472`, `d88b5ef`, `53a1aee`, `6c36e21`, `4fb5f0e`,
  `e977d91`): the Pages drawer's Writer tab shows the pack as `Fit` left it and what it dropped;
  input budgets in Settings → Models; the run start's confirmed rewrite switch; the run report's
  fields; card proposals on the Graph screen; the agent's cards for `sites_explore` and proposals.
- **The proofs** (agent H, `ff8b74c`, `e2789f0`, `cba2801`, `d3bc27a`; `676133b`):
  `TestTheWriterSeesTheFullPicture`, the stand capture, `TestTheAgentSeesTheSite` and
  `TestThePackOfEveryStandPage` compile, vet and lint; the harness seeds a writer pack and a card
  proposal.
- **The closing reviews** (`6fa8f1b` to `16515ed`): two whole-branch reviews found the writer
  could still lose the client's content with rewrites off; every finding was closed with a test
  that failed first (the guard on the client's own blocks for every write over an existing page,
  a block-editor part of its own, relinks that splice, our blocks found by aligned renders, the
  fields' live values, a card accepted as shown, twins as pages, the link's href form in 0065).
  What stays open is listed under "What stays open" in `DECISIONS.md`.
- The tool schemas went from 61,437 to 63,107 bytes (ceiling 63,120); `PagesService.WriterPack`
  and the three card methods take the bound methods to 130.

**2026-10-19, stage 2a of the site model landed in full** (its last five tasks as stage 2b's Part
0; the reasoning under stage 2a in `DECISIONS.md`). The client names his page types over six
presets in Settings → Page types, enables fields per type with an instruction, limits, a rewrites
switch and the link policy, and a page's type shows with its source and reason in the drawer;
the entity card (definition, facts, notes) is edited on the Graph screen; the Policies screen and
every entity kind are gone. `task build` is green with no generated drift, `npm run typecheck`
clean, 1663 vitest tests in 157 files. `TestThePageTypesFollowTheSite` waits for the stands at 2b's
gate.

**2026-10-18, stage 1c of the site model: 1b's gaps closed and the screens. Stage 1 is done.**
The reasoning is under **stage 1c** in `DECISIONS.md`; the task list is under 1c in
`docs/superpowers/plans/2026-10-17-phase-14-site-model.md`.

- **The hierarchy follows the breadcrumbs.** A crumb resolves by its link against every modelled
  page (a modelled page whose own visit lands on another is followed), else through the redirect
  its link led to (looked up once in the render stage, kept on the crumb), else by its exact name
  when it has no link or a dead one; one name candidate resolves, several are `parent_ambiguous`.
  `pages.parent_method` (0055) says which. The front page is the root and no longer technical
  (0056); a Home-only trail is weak; a parent loop is broken and named `parent_cycle`.
- **The diagnosis**: the replica's mu-plugin linked a crumb to an address that answered 404, and
  the derivation had no redirect and no name to fall back on; the third `parent_unresolved` was the
  post's "Uncategorized", a post category archive the model never holds, and it stays.
- **The replica carries every breadcrumb shape**: a hub linking its parent hub, a hub linking term
  archives, a middle crumb with no link, a legacy category address answering 301 to a hub, a
  nav-only trail beside a Home-only JSON-LD list, and a product with no description.
- **Every root state is truthful**: `archive` (0057 recreates the render cache) for term pages, the
  shop and the posts page; the state follows the body when the root is not located; a page whose
  visit lands elsewhere keeps only its body's state and `render_redirected`. `not_found` is 0 on
  both stands.
- **Links are typed site-agnostically**: self links are dropped, `rel` tag, category and author
  links are navigation, the listing rule holds outside the root, ARIA landmark roles count. The
  replica's unknown links went from 39 to 2, vanilla's from 39 to 9.
- **The derivation writes only what it owns** (`UpdateDerived`, `UpdateObservation`), and so do
  the plugin probes (`UpdatePlugin`), so an edit made while a walk runs is kept.
- **The screens**: the Pages drawer's **Model** tab (identity, the parent with its reason and
  method, trails crumb by crumb, the render's root and regions, typed links, technical and
  read-only reasons, blocks read-only, the open findings); the Links tab's type and source; the
  Sites settings (User-Agent, render concurrency, full-walk cadence, link URL form) and a sync with
  "Full walk" and "Render every page again"; the observation summary on Overview; Linking by type,
  owed links contextual only. `PagesService.Model` and `ModelSummary` are bound; `vocab.ts` gains
  the model's vocabularies.
- **The UI harness** places nested pages with their breadcrumb, and its sync failure (the fake
  listed trashed posts, which the plugin never does) is fixed.

**2026-10-18, stage 1b of the site model.** The reasoning is under **stage 1b** in
`DECISIONS.md`; the plan's task list is under 1b in
`docs/superpowers/plans/2026-10-17-phase-14-site-model.md`.

- **The model exists.** Every page of a site, of any public post type plus the `product_cat`
  term family, has an identity (`page_identity`), its terms (`page_terms`), its fields with a
  current and a base value (`page_fields`), its body as blocks, current and base
  (`page_bodies`, `page_blocks_written`), the extraction of its render (`page_renders`: the
  content root, the regions, every trail, the JSON-LD facts, headings, word counts, the builder,
  the URL form, the robots meta), typed links per source (`page_links` rebuilt), and a parent with
  its reason, a technical reason and a read-only reason on `pages`. `findings` holds what the
  observation disagrees with, `sites.settings` what a site is told. Migrations 0045 to 0054.
- **`sync_site` is the observation**: five stages inside one step, each batch checkpointed, over
  `application/observe`, incremental through `since` with a full walk every tenth and on demand
  (`full`, `renderRescan`); renders fetched four at a time; `parent_unresolved`,
  `plugin_outdated`, `field_changed_on_site`, `body_changed_on_site`, `render_unreachable`,
  `builder_detected`, `technical_detected`, `page_title_differs`, `page_h1_differs`,
  `parent_differs` and `url_form_mixed` are its findings, and nothing is merged silently.
- **Nothing new is written to a site**: `publish` sends neither slug nor parent to an existing
  page, `publish_parent` is the creation-only lookup, `repair_hierarchy` and the `repair`
  recipe are gone, and every writer records the body it wrote as the page's base.
- **The extraction reads the stands as they are**: Rank Math's and WooCommerce's JSON-LD shapes,
  the fixture's own lists, a microdata-only breadcrumb nav (Blocksy), WooCommerce's list-less
  nav, two `BreadcrumbList` objects on one page, the hubs' empty and shortcode-only raws. Goldens
  from both stands live under `internal/domain/render/testdata`.
- **A relative href resolves against the document on every side**; the plugin is **1.4.1**.
- **A site has settings**: `sites.Update` and the `sites_update` tool take the User-Agent
  override, the render concurrency, the full-walk cadence and the link URL form; every client the
  registry builds, the connection test of a saved site included, sends the site's agent.
- **A fresh whole-branch review closed the stage**: one body hash everywhere (our own write and
  a product without a description raised `body_changed_on_site` before), the render stage skipping
  archived pages with a stable finding subject, the site read outside the SQLite write unit, a
  render batch reading its own bodies, `since` anchored on the site's own stamps, and the trail
  cut at the page's own crumb. The minors it left are listed under stage 1b in `DECISIONS.md`.
- **`TestTheSiteIsObservedIntoTheModel`** passes on both stands: vanilla 14 pages observed and
  rendered, 2 trails resolved, 0 listings, 7 breadcrumb and 3 contextual links, shop, cart,
  checkout, account, privacy and a noindex page technical; the replica 19 pages observed and
  rendered, 4 trails resolved, 8 listings, 18 breadcrumb, 6 contextual and 79 listing links, the
  front page and the store pages technical, and three `parent_unresolved` findings naming
  Healing Peptides. The owner sees nothing new on screen yet: the model fills behind the Pages
  screen.

**2026-10-17, stage 1a of the site model.** The reasoning is under **the owner's answers** and
**stage 1a** in `DECISIONS.md`.

- **The companion plugin keeps its name and moves to 1.4.0.** It gains:
  - `GET /content/{id}/rendered`, the content as `the_content` filters it inside a singular main
    query;
  - `GET /meta/{id}?keys=`, which reads only the keys asked and refuses a protected one;
  - `GET /terms/{id}`, every taxonomy a post carries;
  - `GET /settings`, the technical pages;
  - on `/content`: `excerpt`, `terms` and every public type and taxonomy, the default still the
    four types;
  - a neutral `preview_token`, a noindex preview under Rank Math and Yoast, post routes that
    answer content posts only, and an `uninstall.php` that keeps the SEO values.
- **`normalize.php` classifies an href as `Site.Resolve` does**, the host with its port, which one
  table proves on both sides.
- **The client** sends a neutral `Mozilla/5.0` (with `wp.WithUserAgent`) and reads the post
  types, routing by `rest_base`, the embedded terms and the site settings. It reads every new
  plugin route, and the store's products, variations and pages as read-only facts.
  `Capabilities.Outdated` names a plugin below 1.4.0 or missing a capability, and the Sites panel
  says to upload the new zip over it.
- **The two stands.**
  - Vanilla runs on 8088: Twenty Twenty-Five, the block editor, nested pages.
  - The replica of the client's first site runs on 8087: Classic Editor, Blocksy and Rank Math,
    a mu-plugin injecting a breadcrumb nav and a product grid into the content with its JSON-LD,
    hubs with an empty or a shortcode-only raw, flat pages, and variable products from a CSV.
  - `task sandbox:provision STAND=` makes the owner's sandbox either stand on command.
- **New client methods waiting for callers.** `GetRendered`, `GetMeta`, `GetTerms`,
  `PluginSettings`, `PostTypes`, `SiteSettings`, `ListProducts`, `ListVariations` and
  `StoreSettings` have no production caller yet. 1b's observation is their caller; they are
  meant, not dead.

**2026-10-17, stage 0 of the site-model redesign**: an audit, the contract and a design, no code.
The client's sites exist and are edited, never rewritten, so Postulator becomes one structured
model of a site (every page of every type: identity, fields, a body of blocks, the regions of its
render, typed links, JSON-LD, a page type, a derived read-only hierarchy) filled by four providers
behind a field registry, with the writer's context pack as the spine. The decision record and its
reasoning are under **2026-10-17** in `DECISIONS.md`, the design is part 16 of the binding spec, and
the four stages (observe read-only with the strengthened plugin and two docker stands; page types, field
scopes and the context pack; editing as blocks and fields with revert; reconciliation, findings and
the tools over the model) are in `docs/superpowers/plans/2026-10-17-phase-14-site-model.md`. The
audit found what the stages undo: every sync walks the whole site, the WordPress parent is read and
dropped, the render is never kept, links carry no type or region, `publish` sends slug, status and
parent on every write, the writer sees a fraction of the context pack, every request names the
program in its User-Agent, and no custom post type or taxonomy can be observed. The owner answered
the seven questions the same day (under **the owner's answers** in `DECISIONS.md`): the plugin
keeps its name and the footprint ruling covers only what a visitor receives.

**2026-10-03 and 04** answers the client's $10 of OpenAI spend that bought little, imports a
whole workbook at once, and reads again the code that had been rewritten many times; the
WordPress categories built that day for the Category and Subcategory of his sheets were removed
the same day on his word. The reasoning is under the four sections of those dates in
`DECISIONS.md`.

- **OpenAI only, through our own Responses API client** (`internal/adapters/llm/openai`, faked
  by `openaitest`); gollem and the Anthropic, Gemini and go-openai SDKs are gone, and migration
  0038 drops the profiles of the removed providers. A structured answer is one strict call, a
  credit refusal waits for a person instead of five retries, `Retry-After` is read, the writer runs
  on flex and falls back to the default tier, and a single call stays out of the prompt cache.
- **Reasoning and speed per role** are settings in Settings → Models (`llm.effort.*`,
  `llm.tier.*`, `llm.flexPatience`); the effort left the catalog row (0037). The writer is paid
  for at most two truncated answers, then the page waits for a person. The estimate prices each
  step at its role's tier and reasoning.
- **The spend is in view** (0035, 0036): the ledger books reasoning, cache writes, the served tier
  and a cancelled stream; Settings → Models shows the spend by purpose, by model and tier and the
  recent calls, and a run's report what each step cost. Five spend bugs are fixed, and a price an
  override left at zero takes the built-in price.
- **The agent runs on the same chain as the runs**, with a ledger row per attempt and its own
  `responses/1` history: a conversation from before keeps its transcript and the model forgets
  it. Its tool schemas went from 85,991 to 63,183 bytes a round; loading tools on demand
  (`agent.toolLoading: deferred`, 6,421 bytes a round as the registry test logs it today) is built
  and off until a funded key has tried it.
- **WordPress categories were built and removed the same day.** Records of their own
  (`categories`, `category_terms`, `pages.category_id`, migrations 0039 to 0041, after a first
  model as a flag on an entity that 0042 and 0043 undid) filed pages, posts and products under the
  chain of a sheet's Category columns. The client then said those columns are WooCommerce's, which
  he keeps himself, and the owner took categories out of the whole application: migration 0044
  drops the three, nothing is filed under a WordPress or WooCommerce category and none is written,
  the import ignores every level column but Root Entity and Root, and a URL parent wins inside the
  row's Root group. The Catalog's products, which no Root names, sit under its Shop page by the URL
  tree.
- **The companion plugin stays 1.2.0**: the 1.3.0 that filed pages under categories never shipped,
  and the plugin's sources are those of `v2.3.0` again. Since 2026-10-12 it requires PHP 7.4, the
  client's host, instead of 8.1; the released app's Sites panel still hands out the 8.1 header
  until the next release.
- **A workbook imports in one go**: one preview and one apply over the chosen sheets in the
  workbook's order, a tab per sheet, every finding naming its sheet, a scope clash caught in the
  preview, and a saved mapping or the sheet's own headers resolved before use, which is what makes
  the agent's import tools work without a mapping.
- **The refactor**: the composition root builds by area, the steps live by concern, the run engine
  reads by life cycle (`Engine.Wake` and `Recover` are gone), the import is a pipeline of named
  stages, the frontend shares its helpers, and the exports nothing called are deleted. The reports
  Pages tab no longer colours a failed item green; a re-import no longer rewrites a matched
  entity's anchors as the user's at weight 1, losing the agent's and every weight, but keeps them
  and adds only the sheet's new ones; a run resumed before it started now starts; a repair refuses
  a post, which WordPress keeps flat. An entity's name and anchors compare by `graph.Key`, the
  store's own `NOCASE`.

**2026-10-03** edits the products the client creates by hand in WooCommerce; the reasoning is
under **2026-10-03** in `DECISIONS.md`.

- **A product run writes** the description through the plugin's raw route, then the short
  description, the attributes the product lacks and an image where it has none through
  WooCommerce's REST API, and the SEO meta; the name, price, stock, SKU, status, slug and
  categories are never written. A template declares the outputs in its `product` block.
- **A product is refused before anything is spent** when the store is not editable, the plugin is
  missing, the row has no product or the run is a draft; a revert hands back a product a human
  changed since and otherwise puts back exactly what the run replaced.
- **A sheet says its rows are products** (`rowType`), each row finds the client's product by its
  address, its slug or its name and keeps the sheet's URL as `planned_path` (migration 0032), and a
  product created after the import claims the row that waited for it on the next sync. A site
  records whether its store can be edited (`sites.commerce`, migration 0031).
- **`sync_back` warns when a published product's page does not show its description** to a
  visitor, the sign of a page builder, a cache or coming-soon mode.
- `internal/e2e/products_test.go` runs the whole loop against WooCommerce 11.1.2 on the docker
  stack.

**2026-10-02** makes the client's SEO workbooks import: the reasoning is under **2026-10-02** in
`DECISIONS.md`, and `samples/client-sheets.xlsx` carries the four sheet shapes the client uses.

- **Keywords are one list with volumes** on pages and entities (migration 0029), read from one
  cell as `bpc 157 (12000), buy bpc 157 (5,400), bpc-157`, sorted by volume, merged on a
  re-import, and given to the writer numbered with their volume; a template says how many must
  appear (`keywordRules.requiredKeywords`) and the missing ones are one `keywords_missing` warning.
- **An entity's name is unique under its parent** (`entities.scope_entity_id`, migration 0030),
  so a form's name repeats under every product; a shared name is labelled with its parent for the
  model and the anchors, and shown as a path on the screens.
- **The import reads the client's headers**: level columns (`Root Entity | Category |
  Subcategory`) make groups that take a page only on evidence (since 2026-10-04 only a root level
  does; the others are ignored), every row with a page gets an entity, a parent comes from the
  parent cell, the group or the URL tree (since 2026-10-04 a URL parent inside the row's group
  first), `Entity?` and other
  questions stay ignored, `own_entity: no` makes a technical page, the entity level reads as the
  kind, note columns travel with the page to the writer and back out, and the preview says what
  each column became. Matching is by parent and name, an ambiguous name is a blocking finding, and
  a repeated import changes nothing.
- **A group without a page is passed through by the link plan** (`no_page`, optional), and the
  page list filters by an entity and everything under it.

**Phases 0 through 13 are complete, and the production hardening of 2026-09-22 and 23 with
them.** The application composes a Wails v3 window over an adiantum-encrypted SQLite store:
the entity graph, the page map, the templates, the WordPress adapter and its companion plugin,
the LLM stack, the run engine with its sixteen steps, the import, the agent, the schedules,
and fifteen bound services behind one screen contract. A master password locks the whole core,
a backup round trips through one encrypted archive, and the sweep retires artifacts and run
events on their own windows.

**2.2.0 (2026-09-24)** answers the client's second week: eighteen pages, a third of them held
for a human, links not placed, the order wrong, every retry paid for again. The reasoning is
under **2026-09-24 — 2.2.0** in `DECISIONS.md`.

- **The import never plans `/`**, and a row's keywords land on the page (migration 0027; one
  list with volumes since 0029) even when the row names no entity.
- **Entities are proposed for chosen pages or pasted keywords and written only once reviewed.**
  `GraphService.PreviewFromPages`, `ProposeFromKeywords` and `ApplyProposals`, the three tools
  beside them, and a three-step dialog on the Graph screen: pick pages or paste keywords, preview,
  keep the proposals you want.
- **Placeholders work on the first try.** `{primaryKeyword} {entityName} {siteName} {pageTitle}`
  are expanded per page in `ResolveForPage`, anything else is refused by `Validate`, and the writer
  gets the brief's headings back through `content.Assemble`, so `section_missing` cannot happen.
- **The pipeline stops itself less.** A truncated or malformed answer is tried again with more room
  and a repair round (since 2026-10-03 with double the room once, and with no repair round, the
  schema being strict); the linker writes the phrases the body owes and falls back to a plain sentence
  with a warning; validation only grades and holds a page with residual errors for a decision that
  `Engine.Accept` settles in one click; the judge and the image step never fail a page on their own
  clock; a post is sent no parent.
- **The queue knows the tree.** A child is queued behind its parent (`run_items.blocked_by`,
  migration 0028), never runs before it, and is parked with an explanation when the parent stops;
  items are listed in the order they run and the table names the page each waits after.
- **Before the run costs anything**, the estimate prices every page on its own template, checks
  each model role (profile, key, catalog), prices images on the image model, runs each step's
  preflight and refuses a run a finding blocks.
- **The window follows.** Every step event carries the step's sentence, a resumed or regenerated
  run stays live, `step.started` refreshes the rows, the drawer picks its primary action from what
  stopped the page, and the header counts the pages that wait for a decision.

**2026-09-25** answers the owner's third week, on `claude/cool-archimedes-5qgsre`: the reasoning is
under **2026-09-25** in `DECISIONS.md`.

- **The built-ins ask for no image**, and a built-in nobody edited is refreshed to the new version
  on start from the copies under `domain/template/seed/superseded`; an edited one is left alone.
- **Images are drawn when a template asks for them.** The OpenAI adapter no longer sends the
  `response_format` GPT image models reject, the generate recipe carries the image step, the
  estimate warns (`images_step_off`) when a run will not draw what a page asks for, the editor
  turns the step on with the images, and the step says "placed N of M images" and why.
- **A template picked at the start is assigned to the chosen pages** (the drawer, schedules and
  `runs_start`); it used to be a label.
- **An owed link is placed or said.** The brief and `repair_links` owe every target within the
  budget, down and sideways included, the children section follows the effective rules, a link to
  a page not on the site is named (`target_not_published`), and after a publish every neighbor
  that owes the page a link gets it, in a plain sentence when it has no anchor, or says why not
  (`relink_phrase_templated`, `neighbor_link_missing`). A neighbor is planned on its own rules
  over the site policy.
- **A finished page says what it lacks**: the report's notice is the item's `note` and rides on
  `item.done`; the item table marks it **Done, check** and the run header counts such pages.
- **The Linking screen is honest**: every owed link has a state, a planned page reads **Not
  written yet** instead of missing everything and being an orphan, and what waits for a page to
  be written or published is counted apart from what is missing.

`relink`, `repair`, `sync` and `revert` own their recipes. The companion plugin is **1.2.0**.
`TestTheClientLoopFromTheSamples` drives the whole loop from the client's own workbooks against
docker, and `TestAChildWaitsForItsParentAndGoesOnOnceTheParentIsRegenerated` now stops the parent
by exhausting the writer, because an incomplete draft is tried again instead of failing at validate.

## The gate

**2026-10-23, on Windows, at the close of stage 4.1. Partial: the memory guard stopped the adapters
race group.**
- **Run and green:** on the replica, then vanilla, each after `task plugin:deploy`:
  `TestTheWorkbookReconcilesWithTheSite` and `TestTheWholeLoopReachesTheDockerSite` (no draft of
  ours carries an H1, the first stand run of that assertion); both stands taken down. `task lint:e2e`
  0 issues; `task build` with no generated drift; `npm run typecheck`; `go test -race -count=1 -p 2
  -covermode=atomic` over kernel and domain, application and runtime. Every touched package's
  tests, golangci-lint and `gofmt` during the session; the comment check.
- **Not finished:** the adapters race group. Its first run hit the default 10-minute test timeout
  in `internal/adapters/sqlite` (57 s without race; the package has outgrown ten minutes under the
  race detector); the rerun with `-timeout 30m` was stopped by the session's memory guard. Owed, one
  at a time on an idle machine:
  ```
  go test -race -count=1 -p 1 -timeout 30m -covermode=atomic -coverprofile=cov4.out ./internal/adapters/...
  go test -race -count=1 -p 2 -covermode=atomic -coverprofile=cov5.out ./internal/app/... ./internal/transport/... ./cmd/... ./frontend/...
  (merge cov1.out to cov5.out into coverage.out) ; go run ./cmd/covergate
  cd frontend ; npm run test:run -- --maxWorkers=2 ; cd .. ; task ui:lint
  ```

**2026-10-23, on Windows, stage 3's gate, completed after the memory freed. Green on the final
code.**
- **Second run, every owed leg, one at a time:**
  - `go test -race` green over runtime (3 packages), adapters with `-p 1` (30) and app, transport,
    cmd and frontend (9);
  - `go run ./cmd/covergate`: domain+application 88.84% of 15,168 (gate 80%), total 88.14% of
    31,508 (gate 70%);
  - `task build` green with no generated drift; `npm run typecheck` clean; `npm run test:run` 1,790
    tests in 170 files; `task ui:lint` 0 issues, the harness suite in 226 s;
  - the replica: `e2e:full` from a fresh reset (566 s);
  - vanilla after `task plugin:deploy`: the six contract tests, the four exit tests (the
    restricted-user test skips there), `e2e:full` from a fresh reset (247 s), and
    `e2e:full:noplugin` (42 s).
- **What the second run found, each closed with a test that failed first:**
  - `a7e5d68`: the restricted-user proof left its kept page with our text, which failed the
    writer's proof later in the same suite;
  - the no-plugin revert answered `restored` while our body stood.
- Both stands were taken down.

The first run's record follows. It was partial: the memory guard stopped the runtime race group
and the replica's fresh `e2e:full` while another workload held about 5 GB.
- **Run and green:**
  - `gofmt -l .` silent; the comment check; `go vet` plain, `-tags uiharness` and `-tags e2e`
    clean; `golangci-lint run` plain and `--build-tags uiharness` 0 issues; `task lint:e2e` alone
    0 issues;
  - `task build` green with no generated drift at `11f095c`, the last commit that changed a bound
    shape;
  - `go test -race -count=1 -p 2 -covermode=atomic` over kernel and domain (28 packages) and
    application (21 packages);
  - `TestMigrationsRoundTrip` to 0070; `task plugin:lint` (12 files) and `task plugin:test` (53
    href cases, 35 meta-key cases) on PHP 7.4;
  - on the replica after `task plugin:deploy`: the six plugin contract tests, and the four exit
    tests on the final code (`TestAnExistingPageIsEditedBlockByBlock`,
    `TestTheSiteFiltersAreRespectedUnderARestrictedUser`, `TestAProductIsEditedThroughTheRegistry`,
    `TestTheProductLoopEditsWhatTheClientCreatedInTheStore`, 188 s);
  - `TestTheClientLoopFromTheSamples` alone after its fix;
  - the packages each fix touched (`block`, `steps`, `runs`, `tools`, `runtime`, `observe`,
    `pagetypes`, `transport/wails`);
  - the agents' runs: vitest 1,790 tests in 170 files, and the harness suite in 609 s (both
    before `9499faa` and `8acd0ec`).
- **The first replica `e2e:full`** (on a stand the exit tests had edited) failed two tests: the
  client loop's sync point, fixed in `6441006`, and the writer's picture, which refuses an edited
  stand. Its rerun from a fresh reset was stopped by the memory guard while provisioning; the stand
  was taken down.
- **The commands the second run used, one at a time on an idle machine:**
  ```
  go test -race -count=1 -p 2 -covermode=atomic -coverprofile=cov3.out ./internal/runtime/...
  go test -race -count=1 -p 1 -covermode=atomic -coverprofile=cov4.out ./internal/adapters/...
  go test -race -count=1 -p 2 -covermode=atomic -coverprofile=cov5.out ./internal/app/... ./internal/transport/... ./cmd/... ./frontend/...
  (merge the five profiles into coverage.out) ; go run ./cmd/covergate
  task build ; git status --short
  cd frontend ; npm run typecheck ; npm run test:run -- --maxWorkers=2 ; cd ..
  task ui:lint
  task e2e:reset STAND=replica ; task plugin:deploy STAND=replica ; task e2e:full STAND=replica ; task e2e:down STAND=replica
  task e2e:up STAND=vanilla ; task plugin:deploy STAND=vanilla
  E2E_STAND=vanilla go test -tags e2e -count=1 -run 'TestMetaWritesOnlyWhatTheUserMayEdit|TestTheRawWriteAnswersWhatWasStored|TestSaveStabilityFollowsTheUsersFilters|TestSEOMetaWritesUnderTheHashesRead|TestManifestDescribesTheSite|TestMetaReadsOnlyTheKeysAsked' ./internal/adapters/wp/e2e/
  E2E_STAND=vanilla go test -tags e2e -count=1 -timeout 45m -run 'TestAnExistingPageIsEditedBlockByBlock|TestTheSiteFiltersAreRespectedUnderARestrictedUser|TestAProductIsEditedThroughTheRegistry|TestTheProductLoopEditsWhatTheClientCreatedInTheStore' ./internal/e2e/
  task e2e:reset STAND=vanilla ; task plugin:deploy STAND=vanilla ; task e2e:full STAND=vanilla
  task e2e:full:noplugin STAND=vanilla ; task e2e:up STAND=vanilla ; task e2e:down STAND=vanilla
  ```

**2026-10-20, on Windows, stage 2b's stand legs on both stands. Green on the final code
(`07d4b70`).**
- **The captures, from fresh resets:** the replica's `sitemodel/testdata/replica` holds 23 pages,
  6 entities, 4 edges, 6 types, 20 bodies, 23 renders, 703 links and 9 open findings, and
  vanilla's holds 14 pages, 2 entities, 1 edge, 6 types, 12 bodies, 14 renders, 148 links and 7
  open findings. `TestThePackOfEveryStandPage` goldens every page of both stands, each within
  3,000 tokens; both goldens were read line by line before they were committed.
- **The first runs found nine things**, seven in the replica's capture, exit tests and first
  `e2e:full` and two in vanilla's golden and writer test, all listed under stage 2b in
  `DECISIONS.md` (`e1a52eb` to `07d4b70`). A second replica run was stopped during its reset by the machine's
  memory, before any test ran.
- **The final runs, each from a fresh reset, one stand at a time:** `task e2e:full STAND=vanilla`
  (522 s, the reset included) 7 passed and 3 skipped. `task e2e:full STAND=replica` (559 s) 8
  passed and 2 skipped. The skips are by design: the capture skips without `E2E_CAPTURE`, the
  degraded loop skips while the plugin is active, and `TestTheAgentSeesTheSite` skips on vanilla.
  The replica's brief is 681 bytes, and the `/peptides/` body pack through `sites_explore` is 880
  tokens of a 4,096-token budget with nothing dropped. Both stands were taken down and Docker
  Desktop was stopped.
- **After the fixes:** `go test -race` over `block`, `contextpack`, `sitemodel` and `observe`
  green; `go test` of `render`, the other caller of `Body.Shortcodes`, green; `golangci-lint` on
  `block` and `contextpack` and `task lint:e2e` 0 issues; `gofmt` silent; `go vet -tags e2e`
  clean.
- **Not run in this round:** `task e2e:full:noplugin`, which belongs to stage 1b's legs, and the
  frontend legs, which nothing in this round touched.

**2026-10-19, on Windows, at the close of stage 2b. Everything that runs without Docker is
green; the stand legs wait for it.**
- **Run and green on the final code:** `gofmt -l .` silent; the comment check; `go vet ./...`,
  `go vet -tags e2e` and `go vet -tags uiharness` clean; `golangci-lint run`, its `uiharness` build
  and `task lint:e2e` (alone) 0 issues; `task build` green in 90 s with no generated drift;
  `go test -race` in five groups (kernel and domain, application, adapters, runtime, transport with
  app and cmd) all green, the groups the closing fixes touched run again after them;
  `go run ./cmd/covergate` 88.44% on domain and application and 88.37% overall, every kernel
  package at 96% or more; `npm run typecheck` clean; `npm run test:run` **1728 tests in 162
  files** with two workers; `task ui:lint` green (the harness's tests in 129 s).
- **Not run, Docker being down for the whole stage**, one stand at a time, the replica first:
  ```
  task e2e:reset STAND=replica
  $env:E2E_CAPTURE='1'; task e2e:full STAND=replica -- -run TestTheStandModelIsCaptured
  Remove-Item Env:E2E_CAPTURE
  go test ./internal/application/sitemodel -run TestThePackOfEveryStandPage -update-packs
  task e2e:full STAND=replica -- -run 'TestThePageTypesFollowTheSite|TestTheWriterSeesTheFullPicture|TestTheAgentSeesTheSite' -v
  task e2e:full STAND=replica
  task e2e:down STAND=replica
  ```
  then the same with `STAND=vanilla` (`TestTheAgentSeesTheSite` skips there). Review the two
  `packs.golden` files before committing them; the agent test prints the replica's brief and the
  hub's pack.

**2026-10-18, on Windows, at the close of stage 1c. Partial: the machine ran out of memory before
the heavy legs.**
- **Run and green on the final code:** `gofmt -l .` silent; the comment check; `go vet ./...`,
  `go vet -tags e2e` and `go vet -tags uiharness` clean; `golangci-lint run` and `task lint:e2e`
  (alone) 0 issues; `task build` green in 212 s with no generated drift; `npm run typecheck`
  clean; `npm run test:run` **1604 tests in 151 files** with two workers (the first full run had
  eight workers fail to start under memory pressure, no test failing). Every package this stage
  touched passed its tests on its own (`pagemap`, `observe`, `sqlite`, `reports`, `pages`,
  `transport/wails`, `app`, `runtime/steps`, `render`, `wp`, `wptest`, `finding`, `vocabgen`), and
  the harness's `TestTheSeededSiteIsObservedIntoTheModel` and `TestASyncKeepsThePagesTheSiteStillHolds`
  pass. `TestTheSiteIsObservedIntoTheModel` passes on both stands with the full wiring (replica 23
  pages, vanilla 14).
- **1b's three legs** at the start of the stage: `task e2e:full:noplugin STAND=vanilla` green
  (288 s), `task e2e:up STAND=vanilla` restored the plugin, and `task ui:lint` found the harness's
  sync test red (the fake listed trashed posts), fixed in `63d009e` and green alone; a rerun of the
  whole `ui:lint` was invalidated by an edit made during it.
- **Not run, for the owner to run one at a time:** `go test -race -count=1 -p 2 -covermode=atomic`
  in the five groups and `go run ./cmd/covergate` (the run's first group failed to start the
  linker, `0xc0000142`, with about a gigabyte free, and the session's memory guard then stopped
  it); `task ui:lint`; `task e2e:test` on both stands; `task e2e:full` on both stands;
  `task e2e:full:noplugin STAND=vanilla` followed by `task e2e:up STAND=vanilla`. The plugin did
  not change, so its lint and tests are not owed.
- **Counts:** migrations end at **0057**; **93 tools**; **126 bound methods**; plugin **1.4.1**.

**2026-10-18, on Windows, at the close of stage 1b, at `8f0c80a`.** The last two commits after
the whole run are the sync request's JSON tags (`8e88fbe`) and the harness seam (`8f0c80a`); the
checks they touched were run again on the final tree, the rest stands from `d2e939b`.
- **Static checks**, on the final tree: `gofmt -l .` silent; the comment check passes;
  `go vet ./...`, `go vet -tags e2e` and `go vet -tags uiharness` are clean; `golangci-lint run`
  and `task lint:e2e` (run alone) report 0 issues at `8e88fbe`, and the two packages the seam
  touched 0 issues at the final tree.
- **The Go suite:** `go test -race -count=1 -p 2 -covermode=atomic` is green in the same five
  groups, 86 packages, groups 3 and 5 rerun on the final tree and the profiles merged.
  `go run ./cmd/covergate`: **domain+application 88.12% of 11488** (gate 80%), **total 88.39% of
  25055** (gate 70%). Run whole for the first time this stage, it found the vocabulary tripwire
  red since the `repair` kind left the start enum; fixed under stage 1b in `DECISIONS.md`.
- **The build and the frontend:** `task build` is green on the final tree and leaves no generated
  diff, after it caught the sync request's two walk options bound as required fields;
  `npm run typecheck` is clean against the regenerated bindings; **1512 vitest tests in 143
  files**. `task ui:lint`: the lint half reports 0 issues on the final tree; its harness-test half
  has no result on this tree, because the run before the seam fix ran past its 15-minute budget
  (every scenario re-seeded a core whose adoption sync was paced at four requests a second, which
  `8f0c80a` lifts for the fake) and the two reruns were stopped by the session's memory guard.
  Run `task ui:lint` once more before a tag.
- **The plugin:** `task plugin:lint` covers 10 files and `task plugin:test` agrees on 53 cases,
  both on PHP 7.4.
- **The stands**, at `d2e939b` (the two later commits touch neither the observation nor the
  plugin): `task e2e:test` passes on vanilla (67 s) and on the replica (111 s); `task e2e:full`
  passes on vanilla (154 s) and on the replica (196 s), the observation proof among them.
  `task e2e:full:noplugin` passed at `d2076a9`, one commit before; its rerun was stopped by the
  session's memory guard mid-test together with the step that restores vanilla's plugin, so
  vanilla stands without the plugin until `task e2e:up STAND=vanilla` runs. Run
  `task e2e:full:noplugin STAND=vanilla` once more before a tag.
- **Counts:** migrations end at **0054**; **93 tools**; plugin **1.4.1**.
- **Not run:** a live OpenAI call with a funded key, and the `E2E_SEO=yoast` pass.

**2026-10-17, on Windows, whole, at the close of stage 1a, at `66a7d58`.**
- **Static checks:** `gofmt -l .` silent; the comment check passes; `go vet ./...` and
  `go vet -tags e2e` are clean; `golangci-lint run` and `task lint:e2e` (run alone) report 0
  issues.
- **The Go suite:** `go test -race -count=1 -p 2 -covermode=atomic` is green in five groups
  (kernel+domain, adapters, application, runtime, transport+app+cmd+wp-plugin+frontend), the
  profiles merged into `coverage.out`. `go run ./cmd/covergate`: **domain+application 87.87% of
  9166** (gate 80%), **total 88.49% of 22606** (gate 70%).
- **The build and the frontend:** `task build` is green and leaves no generated diff;
  `npm run typecheck` is clean; **1512 vitest tests in 143 files**; `task ui:lint` is green
  (377 s).
- **The plugin:** `task plugin:lint` covers 10 files and `task plugin:test` agrees on 41 cases,
  both on PHP 7.4.
- **The stands:**
  - `task e2e:test` on vanilla passes 58 tests;
  - on the replica it passes 57, with `TestSEOMetaEscapesRenderedHead` skipped because it runs
    only without an SEO plugin;
  - `task e2e:full` is green on both stands, and `e2e:full:noplugin` on vanilla.
- **Counts:** migrations end at **0044**; **93 tools**; plugin **1.4.0**.
- **Not run:** a live OpenAI call with a funded key, and the `E2E_SEO=yoast` pass.

**2026-10-04, on Windows, whole, after the categories came out, at `2b74b4a`.** `gofmt -l .`
silent, the comment check, `golangci-lint run` and `task lint:e2e` 0 issues, `go vet -tags e2e`
and `-tags uiharness` clean; `go test -race -count=1 -p 2 -covermode=atomic` green over every
package in the same five groups, the profiles merged into `coverage.out`; `go run
./cmd/covergate` **domain+application 87.87% of 9166** (gate 80%), **total 88.40% of 21989**
(gate 70%); `task build` green and `task vocab` and `task events` leave no diff; `npm run
typecheck` clean and **1509 vitest tests in 143 files**; `task ui:lint` green (376 s); on a fresh
8088 stack with WooCommerce on, `task e2e:test`, `task e2e:full` and `task e2e:full:noplugin`
green. Migrations end at **0044**; **93 tools**, 63,183 bytes of schema against the 63,200 the
registry test allows; **124 bound methods**; the plugin is **1.2.0**. Not run: a live OpenAI call
with a funded key, and the `E2E_SEO=yoast` pass.

**2026-10-04, on Windows, whole, over the 2026-10-03 and 04 work at `dad410e`.** `gofmt -l .`
silent, the comment check, `golangci-lint run` and `task lint:e2e` 0 issues; `go test -race
-count=1 -p 2 -covermode=atomic` green over every package, run in five groups (kernel+domain,
adapters, application, runtime, transport+app+cmd) because one whole run was stopped by the
machine running low on memory, the five profiles merged into `coverage.out`; `go run
./cmd/covergate` **domain+application 87.92% of 9677** (gate 80%), **total 88.63% of 23151**
(gate 70%); `task build` green and leaves no generated diff; `npm run typecheck` clean and **1624
vitest tests in 151 files**; `task ui:lint` green (427 s); `task e2e:test`, `task e2e:full` and
`task e2e:full:noplugin` green on 8088 with WooCommerce on (the `E2E_SEO=yoast` pass was run by
unit K6 at `2cd8f18`). Migrations end at **0043**; **93 tools**, 63,304 bytes of schema against
the 63,400 the registry test allows; **125 bound methods**. Not run: a live OpenAI call with a
funded key (see Known gaps).

**2026-10-03, on Windows, whole, over the 2026-10-02 and 2026-10-03 work.** `go test -race
-count=1 -p 2 -covermode=atomic -coverprofile=coverage.out ./...` green; `go run ./cmd/covergate`
**domain+application 87.22% of 8819** (gate 80%), **total 87.28% of 21287** (gate 70%);
`golangci-lint run`, `task lint:e2e` and the comment check 0 issues, `gofmt -l .` silent; `task
build` green and leaves no generated diff; `npm run typecheck` clean and **1238 vitest tests in 121
files**; `task e2e:test` and `task e2e:full` green on 8088 with WooCommerce 11.1.2, the adapter
suite and the product loop green again under `E2E_SEO=yoast`, and `task e2e:full:noplugin` green.
`task ui:lint` failed once at 369 s on a test its output did not keep and passed on the two runs
after it. Migrations end at **0032**; **93 tools**, 84,923 bytes of schema against the 85,000 the
registry test allows.

**2026-10-02, on Windows, partial.** Run: `gofmt -l .` silent, the comment check,
`golangci-lint run` 0 issues, `go vet -tags uiharness` and `go vet -tags e2e` clean, `task ui:lint`
green (at `116a77c`), `npm run typecheck` clean and **1202 vitest tests in 119 files**; `go test
-race -count=1 -p 2 ./...` green through `0701b03` apart from the wall-clock flakes of
`internal/runtime`, each green alone, and every later commit green over the packages it touches,
because the full run for `9d84f48` was stopped by the machine running low on memory. Migrations
end at **0030**; **93 tools**, schema ceiling 80,700 bytes. **Not run:** `go run ./cmd/covergate`,
`task build`, `task lint:e2e`, `task e2e:full`; the import side of the e2e client loop was checked
without docker and holds (61 entities, 56 edges, the planted faults with their rows).

Green on 2026-09-25 over the head of `claude/cool-archimedes-5qgsre`, on Linux, where
`cmd/postulator`, `internal/app`, `adapters/browser/tor` and `adapters/secrets/{dpapi,masterkey}`
build only under `GOOS=windows`, `transport/wails` needs GTK to build its tests, and `TestDSN` in
`adapters/sqlite` fails on the path separator alone.

- `gofmt -l .` silent, the comment check of `task check:go:comments`, `GOOS=windows go vet ./...`
  and `GOOS=windows go vet -tags e2e ./internal/e2e/...`.
- `golangci-lint run` (v2.13.2 built with go1.27, run with `GOOS=windows`) and the e2e sources
  0 issues; `go test -race -count=1 -p 2` green over every package that builds on Linux.
- `go run ./cmd/covergate`: **domain+application 86.17% of 7353** (gate 80%), **total 86.80% of
  18013** (gate 70%).
- `wails3 generate bindings`: **15 services, 122 methods**; `task events` and `task vocab` leave no
  diff; migrations up to **0028**, none added.
- **93 tools**, 78,973 bytes of schema against the 79,000 the registry test allows.
- `npm run typecheck` clean; `npx vitest run` **1162 tests in 115 files** over two projects.

## How to run

```
task vocab · events · bindings    the three generated TypeScript surfaces
task build                        bin/postulator.exe, the frontend and the bindings first
task package · plugin:zip         the NSIS installer and the plugin archive, into bin/
task ui:run · walk · shot · reset the harness window, seeded, DevTools on 9222
task ui:lint                      lint and tests behind the uiharness build tag
task lint:e2e                     the build-tagged sources golangci-lint run skips
task e2e:up · test · full · full:noplugin · down STAND=vanilla|replica   8088 or 8087; -- <args> to go test
task e2e:both                     both stands up and their suites, one after the other
task plugin:deploy STAND=         the zip reinstalled over the plugin on a running stand
task plugin:test                  wp-plugin/tests on PHP 7.4 (the href table)
task sandbox:up · down · reset    the owner's WordPress on 8089, bare unless E2E_PLUGIN=1
task sandbox:provision STAND=     the owner's sandbox made a stand, on that command only
go test -race -count=1 -p 2 -covermode=atomic -coverprofile=coverage.out ./...
go run ./cmd/covergate            the coverage gates on that profile
$(go env GOPATH)/bin/golangci-lint.exe run
frontend: npm run typecheck · npm run test:run
```

`npm run test:run` runs two vitest projects: `model` on node over `*.test.ts` and `screens` on
jsdom over `*.test.tsx`. The Go suite needs `-p 2` here, because the race detector under the
default parallelism wants more than this machine's paging file holds, and it wants the machine
to itself: the engine's deadlines are wall clock, so beside another heavy job `internal/runtime`
fails with cancelled WordPress requests that mean nothing. **Run `task build` before a tag**,
not only `npm run typecheck`: only the build regenerates the gitignored bindings.

## Phases

| # | Phase | Status |
|---|---|---|
| 0 | Wipe, scaffold, kernel, CI, docs | done, reviewed |
| 1 | SQLite store, migrations, secrets; the Wails contracts spike and error conversion | done, reviewed |
| 2 | Domain core: graph, page map, templates, settings | done |
| 3 | WordPress adapter, `wptest`, the companion plugin and the docker e2e harness | done, reviewed |
| 4 | LLM port, gollem client, catalog, ledger, limiter, retry | done |
| 5 | Run engine, checkpoints, recovery | done |
| 6 | Content factory: document, links, compliance, first steps | done |
| 7 | Remaining steps, images, sync, site reports | done |
| 8 | Import and export of the client page map | done |
| 9 | Tool registry, agent runner, confirmations | done |
| 10 | Schedules and the ticker | done |
| 11 | Wails services, event bridge, TypeScript generation | done |
| 12 | Master password, backup, retention, e2e, release | done, reviewed |
| 13 | The product frontend on one screen contract | done |
| — | Hardening 2026-09-22/23: links, agent reliability, reversibility, agent cost, relink and repair as kinds, the content steps, the client scenario | done, gate green per wave |
| — | 2026-10-03/04: OpenAI only on our own client, the spend in view, the whole workbook, the refactor; WordPress categories built and removed | done, gate green after the removal |
| 14 | The site model (2026-10-17): stage 0 audit, contract, design and plan; stages 1 to 4 in `2026-10-17-phase-14-site-model.md` | stages 0 to 3 done (stage 3 on 2026-10-20 to 23, its gate green on both stands); stage 4 in three sessions, 4.1 half done (2026-10-23) |

## Known gaps

- **`agent.Deps.Allowed` is never set**, so `Permit` can never deny in production and the
  transcript's refused tool row has no producer; `TestTheSeededConversationShowsADeniedToolCall`
  is skipped with that reason. Either per-conversation permissions get a producer, or the
  state and its copy go.
- **`LinkAudit` resolves the template of every mapped page**, one `ResolveForPage` each, so
  five thousand mapped pages cost about two seconds. The frontend caches it for thirty seconds.
  `relink_neighbors` does the same for every page that may owe a newly published page a link, so
  publishing the root of a large tree resolves every descendant once.
- **The estimate prices the linker on up links and the lead keyword**, not on the down and
  sideways phrases `repair_links` now writes when the body lacks them (256 output tokens each).
- **The image settings apply after a restart**: the adapter and the price reference are built
  with the composition, and the Settings copy says so. So do `llm.timeout`,
  `llm.openai.baseUrl` and `llm.flexPatience`, read when the model client is composed.
- **What only a funded key can show.** The live probe ran on a key with no credit, which proves a
  request's shape and nothing it costs. Owed at the pre-release smoke: the probe's open tests
  (minimal effort and sampling, a stateless tool loop replayed across rounds, the replay forms of
  an assistant message, a stream's `[DONE]` and obfuscation, what breaks the cache prefix, flex
  served on terra and luna, the strict keyword matrix, whether output counts reasoning, field
  limits, image tokens per quality), the body of a flex capacity refusal, `anyOf` for a nullable
  object or array, and `cache_write_tokens` against a real bill. **The estimate prices an image at
  1,056 output tokens whatever its quality.**
- **Seven checks before `agent.toolLoading` becomes `deferred`**, with a funded key: a turn that
  triggers `tool_search` parses its `tool_search_call` and `tool_search_output` items and its calls
  carry the namespace; the next round, under `store: false`, accepts the search items replayed
  without `id` or `status` and the namespaced call; a second turn accepts history calls replayed
  without their namespace, and the setting switched both ways between turns works; `cached_tokens`
  shows from round two and the tools cost about 1,600 input tokens a round before a search; the
  model finds the
  right group from the descriptions (`templates_create`, `imports_preview`, `runs_start`); a turn
  stays within the loop limit of twelve; the loaded tool definitions show in the usage and the
  ledger.
- **A flex attempt abandoned for its patience returns no usage**, so partial work OpenAI may bill
  for it is not metered, and a capacity refusal is assumed unbilled.
- **`relink_page` reads the site record only after it wrote the page**: if that read fails, the
  site holds the relinked body while the map keeps the old hash, and the next sync reports the
  run's own write as drift.
- **What a revert cannot put back:** media a run uploaded, because a delete needs `force=true`
  and sweeping media a human may have reused is worse; and, against plugin 1.1.0, the SEO meta,
  because the read is refused from the manifest and `revert_meta_kept` names it instead.
- **`llm_calls` has no `message_id` or `round`**, so an agent turn's rows are found by its
  conversation and their time. `Retry-After` is read since 2026-10-03, and `ledger.List` has its
  caller in `ModelsService.ListCalls`.
- **`ProposeFromPages` is one bound call** making one model call per forty unmapped pages, so
  thousands of them hold the window's call for minutes. Turning it into a run is the fix.
- **UI residue, all seen in the walk, none of it wrong output:** the Models table shows about
  ten characters of a model name with the dock open; the run table pushes "took" outside 960;
  the layer chooser is clipped at 960 with the dock; the graph legend covers the canvas at 1280.
- **The harness's fake WordPress serves no signed draft preview**, so the page drawer shows
  `rest_not_logged_in` there; `assertDraftPreviews` covers the real path on docker. **The
  docker stack is never run in CI** — `windows-latest` cannot run Linux containers — so every
  suite is run by hand before a tag.
- **Without the companion plugin** the SEO meta is skipped with a warning, the neighbour relink
  stands down, no draft can be previewed and a revert of an updated page pauses; the loop still
  generates, publishes and reads back.
- **Application events published before the window exists are dropped**, by design, and
  backward paging exists in every repository but not at the use-case boundary.
  **`settings.changed`** is published by `models.SetProviderKey` and `DeleteProviderKey` only,
  because `SettingsService.Set` writes at the transport, the one layer that does not publish.
- **A pending action keeps the arguments it will replay**, so a confirmation for a tool
  carrying a credential holds it in the encrypted database until it is settled; the event, the
  summary and the ledger carry it masked.
- **Migration 0016 runs outside goose's transaction**, so a crash between its explicit `COMMIT`
  and goose recording the version leaves a database the next start cannot migrate; the recovery
  is the reset the startup error names. **A backup carried to another machine** keeps its site
  credentials sealed with the key of the machine that wrote them.
- **Three review items stand, none blocking a tag:** the master key is hex encoded into the
  SQLite DSN, which `Lock()` cannot zero; `export.opener.complete` would call a valid archive
  truncated if the tar stream were an exact multiple of `FrameSize`; `task package` is not
  byte reproducible, `BUILD_DATE` being `now`.
- **Found by the 2026-10-17 audit, fixed by the stages that touch them:** the import's
  `Options.AnchorSeparator` description says comma while the default is `|` (`imports/views.go:61`
  against `importmap/mapping.go:12`); `Sanitize` lets a class or id the model emits through. The
  port mismatch between `normalize.php` and `Site.Resolve` was fixed in 1a.
- **What the extraction leaves to stage 2.** `page_renders.url_form` is the majority form of
  every internal href of the page, which follows the theme (absolute on both stands) while the
  client's own content links are relative; the writer's "URL form the site already uses" should
  weigh the contextual links. The H1 sits outside the content root on both themes, so a heading
  list never carries it; `page_h1_differs` reads the plugin's `h1`.
- **A first walk is paced by the client's limiter.** It costs three to four requests a page
  (the raw, the terms, the visit, the filtered render), and the default `wp.rateLimitPerSecond`
  is 4, so two thousand pages take about half an hour the first time; later walks read only what
  changed. The setting goes up to 100 for a host that allows it.
- **`parent_differs` is raised once and never cleared by the observation**, the workbook's
  parent it records having been replaced by the derived one; stage 4's `findings.Resolve`
  settles it. A page deleted on the site is archived on the next full walk, not the next
  observation.
- **`TestTheServerTakesAnAddress` in `wptest` can flake** under a parallel `go test`, because it
  reserves an ephemeral port, releases it and binds it again; another process took the port once
  during 1a. It passes alone.

## Next steps

1. **Stage 4.1 of the site model, resumed at 4.1.4**: the row findings persisted, the resolution
   use case, the samples, `TestTheWorkbookReconcilesWithTheSite` on both stands, and the gate legs
   the 4.1 checkpoint lists; then 4.2 and 4.3, each planned at its start.
2. **The pre-release smoke with a funded OpenAI key**: one generate run with the writer on flex
   (the tier served and the reasoning shown in the ledger), one agent turn with tools (cached
   tokens from its second round), and the spend panel naming both by purpose and model; then the
   probe's open tests and the seven checks under **Known gaps**, and only then a decision on
   `agent.toolLoading`.
3. **A sandbox walk with the client's sheets and a real provider** (`task sandbox:up
   E2E_PLUGIN=1 E2E_WOO=1`, after `task sandbox:reset` if it still carries plugin 1.3.0 and its
   terms): import `samples/client-sheets.xlsx` as one workbook and check each sheet's tab, the
   Groups segment, that Category and Subcategory read ignored, and the Groups sheet's deeper rows
   under their URL parent inside Peptides; sync; run a page under TB-500 › Liquid; create a product
   by hand in WooCommerce under a category of the client's own, import the variation sheet in
   products mode, run it and see its categories untouched; revert both.
4. **The client's own sites**: what a real model writes into the attributes, and whether a page
   builder hides the description, which `product_description_hidden` will say.
5. **A release** when the owner asks: the 2026-10-02, 2026-10-03 and 2026-10-03/04 work as one
   minor version.
6. The residue above: the denied tool row's decision, the four narrow-width UI items,
   `ProposeFromPages` and `Import.Apply` as runs, and `settings.changed` for a declared value.
