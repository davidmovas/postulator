# Status

The handoff point between sessions. Read this first. The reasoning behind every phase and
every ruling is in [`DECISIONS.md`](DECISIONS.md).

**Branch:** `dev`, the development branch; `master` takes a PR from it when the owner asks.
**Released:** `v2.8.0` on 2026-10-28 (PR #34 merged, tagged; phase 19, notes under **Release
2.8.0**, its gate under **The gate at 2.8.0**). **2.8.1 and 2.9.0 are built on `dev`** (2026-10-29):
2.8.1 ticks a range on a shift-click in the page picker; 2.9.0 (phase 20) has the sync build the
topical map from the site's breadcrumbs, with no model. Their notes are under **Release 2.9.0** and
**Release 2.8.1**.

## Next, in order

1. **The stands are green for 2.9.0** (2026-10-29): vanilla 13 passed and 13 skipped, twice;
   the replica 24 after two fixes its run found (the workbook's parent over a built one, and the page
   types suite's entities), each failed test then passed alone on the stand with every test that
   imports a workbook. The stands are down.
2. **The owner opens a pull request from `dev` to `master` for 2.8.1 and 2.9.0 and tags `v2.9.0`**
   when he asks; the tag's workflow builds the installer, the executable and the plugin zip. 2.9.0
   adds migrations 0088 and 0089; the plugin stays 1.5.2.
3. **The client installs 2.9.0 and syncs once**: the run screen says how many entities the
   breadcrumbs built, how many pages they placed and why the rest were left; the Graph's "From the
   site" lens shows what was built, and a page left without an entity keeps its finding and its
   "Propose entity".
4. **The race legs**, owed since 2.7.0, one call each on a quiet machine (no stand up, nothing else holding
   memory), from the repository root under bash:
   ```
   go test -race -count=1 -covermode=atomic -coverprofile=cov-kd.out ./internal/kernel/... ./internal/domain/...
   go test -race -count=1 -p 1 -covermode=atomic -coverprofile=cov-app.out ./internal/application/...
   go test -race -count=1 -p 1 -covermode=atomic -coverprofile=cov-runtime.out ./internal/runtime/...
   go test -race -count=1 -p 1 -timeout 30m -covermode=atomic -coverprofile=cov-adapters.out ./internal/adapters/...
   go test -race -count=1 -p 1 -covermode=atomic -coverprofile=cov-rest.out ./internal/app/... ./internal/transport/... ./cmd/... ./frontend/... ./wp-plugin/...
   (head -1 cov-kd.out; tail -q -n +2 cov-kd.out cov-app.out cov-runtime.out cov-adapters.out cov-rest.out) > coverage.out
   go run ./cmd/covergate
   ```
   then delete the `cov-*.out` files, which no ignore rule covers. The kernel and domain group passed
   at the close; it runs again only for its coverage profile. A `0xc0000142` on a build is Windows
   failing to start a process for lack of memory, not the code; after a stop, look for a leftover
   `*.test.exe`. The scale package needs `-timeout 30m` on a loaded machine.
5. **With a funded OpenAI key, whenever there is one**: compare the ledger's real `input_tokens` with
   `go run ./cmd/tokenreport` (the input side was adopted from its counts on 2026-10-24, a replica
   round now 13,260 tokens); see what a real model calls for each of the matrix's jobs, which the
   scripted suite cannot; settle the output side (`llm.WordTokens` 2.2, `steps.tokensPerWord` 3) and
   whether tools load on demand (`agent.toolLoading: deferred`). Until then the owner judges the
   writer by hand from `go run ./cmd/tokenreport -prompts out/prompts`.

## What Postulator does

A Windows desktop application (Wails v3, Go 1.27, a local encrypted SQLite database, no server, no
account) for one SEO specialist who runs dozens of WordPress and WooCommerce sites of up to a few
thousand pages each. It builds **one structured model of each site** and edits the site's content
through it; [`VISION.md`](VISION.md) is the longer version.

- **The site, read as a model.** A sync walks every page of every public post type and the store's
  categories: its identity and terms, its fields (title, excerpt, SEO meta, the product's
  descriptions, custom fields by key), its body as blocks, what its public render adds (regions,
  breadcrumbs, product grids, JSON-LD), its links by type, and a parent derived from its breadcrumbs
  with the reason. Later syncs read only what changed; every tenth walks the whole site.
- **Page types, rules and field scopes.** The client names his types over presets (Landing,
  Category, Guide, Info, Product, Store category, Catalog, Technical, each over a fixed role) and
  enables, per type, the fields the writer may write, each with an instruction, limits, a switch for
  rewriting his own text, and the link policy. A site's ordered rules give each page its type and
  say why; a preview shows what a change of rules would move before he applies it, and he retypes
  any selection by hand. A run on auto takes each page's template from its type.
- **The workbook, reconciled through findings.** Each row of his xlsx workbooks finds its page by
  URL, slug, entity, title, H1 or SKU. What disagrees becomes a finding in the inbox (a row two
  pages answer, a row no page answers, a parent the sheet and the site disagree on, a field changed
  on the site, a page no entity carries), which he or the agent settles one at a time or a whole code
  at once. The Site map shows the site by its derived hierarchy.
- **The entity graph** says what links where: up to parents, down to children, sideways only to the
  related pages the graph names. A sync builds it from the site's breadcrumbs with no model call,
  each crumb a topic and each page an entity under the crumb above it, and leaves with a reason
  whatever it cannot be sure of; a page it left gets an entity through proposals named from the site.
- **The writer's full picture.** Before it writes a field, the model sees the page's type, its entity
  card, its breadcrumb path, its children and siblings, the products it lists, the keywords with
  their volumes, the anchors, the notes and the field's own instruction and policy, fitted to a
  budget.
- **Edits that keep the client's text.** The body changes block by block. Every block we did not
  write is the client's and stays byte for byte unless a run confirmed rewriting it. Nothing
  structural (slug, status, parent, prices, stock, SKU, taxonomies, a variation's attributes) is
  ever written to an existing page. Every write goes under the hash of what was read and keeps what
  it replaced.
- **Runs** generate, relink, sync and revert, scoped to chosen fields when asked, priced before they
  start, resumed after a crash, and reviewed by a person, whole or field by field, when they write a
  draft. A revert puts back per field and per block and hands back what the client changed since.
- **The agent** in the dock works the same loop through the same use cases: it reads the whole model
  page by page (the tree, the graph, bodies, links), ends an analysis with a plan of writes the person
  approves whole or in part, resolves findings, starts runs with a brief and edits a page field by
  field or block by block. Every write it makes to a page waits for the person.
- **A run's brief**: a goal, a focus, a tone, what to avoid, a reference and a style sample for the
  run, and per page notes, sections, keywords, anchors, must-links and a length, read first by the
  writer, the linker and the judge.

## Release 2.9.0

- **The sync builds the topical map from the breadcrumbs.** After the page tree, a new stage reads
  every crumb of every page, from the root down: a crumb that links a page is that page's entity, a
  crumb no page carries (or a store category with no page of its own) is a topic entity with no page,
  and each entity sits under the crumb above it. Entities are named by the crumb's text, then the
  H1, then the title. Products fall under the hub their store category folds into.
- **Nothing it is unsure of is written.** A crumb that links a page the sync does not hold, a name
  two pages carry, a name already taken at that place, breadcrumbs that disagree with the page tree
  or loop: the page is left without an entity, with its reason, and so is everything under it. A
  page a person mapped is kept as it is, and an entity a person removes is never built again.
- **What it shows**: the run screen counts the entities built, the pages placed and the pages left,
  and lists why; the Graph has a "From the site" lens; the inspector says who made an entity. The
  Graph entity column, the Graph tree and the export's `entity` and `parent_entity` fill from it.
- **Migrations** 0088 (the `site` source on entities and edges, by a table rebuild whose down hands
  them to a person) and 0089 (`topic_builds`, what a sync settled).
- **A workbook places the entities the breadcrumbs built.** When the client's sheet names another
  parent for an entity the sync built, the sheet's parent replaces the breadcrumbs' (the preview
  says "replaced by the sheet"), so a sync and then an import never leave two entities of one name;
  the difference with the site's breadcrumbs stays a finding. A parent a person gave is kept.
- **The Graph's toolbar wraps beside the dock**: with the dock open it ran past its pane, so Link
  proof and Isolate could not be reached at 1280 or 1440; it now takes a second line.

## Release 2.8.1

- **A shift-click ticks a range of pages.** In Graph → Propose from pages and in the Start-run page
  list, a click ticks or unticks one page and a shift-click gives every page shown between it and
  the clicked one the state of the first, as the Pages list does; pages that cannot be picked are
  skipped. 2.8.0 toggled one page per click there, so 400 pages without an entity meant 400 clicks;
  the proposals list after Preview already took a range.

## Release 2.8.0

The client's fourth request, in short:

- **A sync that finishes.** A first sync of 400 empty pages asks the site about 810 times where 2.7.0
  asked about 1,610: the terms come with the page list, an empty page's body is known without asking,
  the bodies a batch needs are fetched four at a time, and a batch is 50 pages. On a slow host a
  batch now takes about a minute, far under the ten minutes that cut 2.7.0's off, so the "WordPress
  error" the client saw, which was our own time limit, no longer happens.
- **A failure that says what happened.** The run screen shows where the sync is (its seven stages,
  pages read, visited, created, changed, products, unreachable, archived), and when it stops, what
  WordPress answered, on which request and which page ("reading the body of /about/: WordPress
  answered 500 on GET …: There has been a critical error on this website."). "Resume the sync" goes on
  from the last batch saved; "Start over" walks the site again. The Sites panel and the Overview follow
  a sync live.
- **One sync at a time.** While a site syncs, every Sync button says "Syncing · reading the pages"
  and opens the sync; a second one is refused with a link to the first. On upgrade, the queued
  duplicates of a site's oldest sync are cancelled. A sync is no longer started from the Start-run
  dialog, where picking 400 pages queued 400 whole-site syncs.
- **Proposed Entities keeps up.** A shift-click sets the range to what the clicked row becomes, through
  folded branches; a drag across the boxes ticks or unticks them; Ctrl-click takes one row without its
  children; a search or a filter offers "Tick the N shown" and "Untick the N shown", and a row ticked
  under "Not ticked" stays in view.
- **Page types a person can read.** Settings → Page types is one screen: three steps (the sync reads
  each page, the rules give it a type, the type decides what Postulator writes), a card per type with
  its pages, the rules that give it and what Postulator writes, and "How pages get a type" with each
  rule's page count. Pages still typed before 2.6 are counted and retyped by the preview.
- **"Entity" is the graph's word again.** The 2.5 type Entity is now "Article"; the Pages column and
  the drawer say "Graph entity", and every page Postulator writes needs one, whatever its type.
- **A page a run cannot write is left out, not a refusal.** A run over a selection that holds a page
  with no template (a Store category, Catalog or Technical page on a site without a default
  template) or a page mapped to nothing in the graph writes the others: the Start-run drawer, the
  run screen and the assistant's card name each page left out, why, and where it is fixed. Only a
  selection with nothing left is refused, and it says why for each page. Opening a page with no
  template no longer raises "That record no longer exists here". The client's "template" error was
  not this: his 2.7.0 sync was cut by the batch time limit fixed above, and the "no template
  recorded" he read sat in the run's header, which no longer shows it on a sync (**2026-10-28** in
  `DECISIONS.md` has the experiments).
- **No new plugin.** 1.5.2 stays.

## Release 2.7.0

The client's third request, in short:

- **The assistant reads the whole site.** It walks the page tree to any depth (a section, a page's
  ancestors and siblings, or the whole site page by page), and a page with no parent says why: the
  front page, a planned page, a breadcrumb it could not resolve, two pages answering one crumb. It
  reads the entity graph with every entity's parent and children, a page's body as text block by
  block, a section's links in and out with what is owed and what is missing, and finds pages by
  title, path or keyword. A long answer is never cut silently; the assistant reads the next page.
- **It ends with a plan.** Once it has read enough, it proposes one card of ordered steps, each with
  its reason and its estimate. He approves all of it, some steps or none, and edits a link's anchor,
  a field's value, a run's brief or a page type picked from a list on the card itself. Nothing runs
  before he approves; a step that fails stops the steps that rest on it; plans stay with the chat.
- **A run takes a brief.** From the run drawer or through the assistant, a run carries a goal, a
  focus, a tone, what to avoid, a reference text and a style sample, and for each page notes,
  sections to add, keywords, anchors, links it must place and a length. The writer, the linker and
  the judge read it first; "What the writer will see" and the run screen show it. A link the graph
  does not connect is refused with the missing edge named, or added as a related edge when he ticks
  "Add a related edge for each", or by an earlier step of the assistant's plan.
- **The page tree against the entity graph.** Where an entity's parent in the graph differs from its
  page's parent in the site's tree, the inbox lists it with "take the tree's parent", "keep the
  graph's" or "ignore". The Graph screen and a Site map section take the tree's parents for many
  entities at once, previewed first. New proposals and a workbook row without a URL take the parent
  from the page tree, so a graph built from a flat import is flat no more.
- **A support bundle.** Settings → About and the dock's menu save one file of what the app holds of a
  site (counts, types, parents and their reasons, breadcrumb states, the graph's shape, findings by
  kind, the last chat's tool calls and their sizes), to send when something looks wrong. It never
  holds a password, a key, a cookie, a page's text or a title.
- **23 assistant tools in place of 94**, so each round of a chat costs about a quarter fewer tokens.
  A chat from 2.6 keeps its transcript and starts the assistant afresh; an approval card of a tool
  that is gone says so in plain words and changes nothing.
- **No new plugin.** 1.5.2 stays.

## Release 2.6.0

The client's second request, in short:

- **The types**: Landing and Category (hubs), Guide (articles), Info (about, contact, shipping,
  returns: only the SEO title and description are ever written), Product, Store category, Catalog and
  Technical. A site from 2.5 keeps its types and gains the four new ones; nothing is renamed or
  deleted and no page moves on upgrade.
- **How pages get them**: an ordered list of rules per site (Settings → Classification), the first
  match winning, each page saying which rule placed it; a page no rule matches becomes Info and is
  listed in the inbox as typed by no rule. A sync types new pages and re-types those the rules
  placed; a type set by hand never moves.
- **Preview and bulk**: a change of rules moves nothing until he applies its preview, whole or by
  groups, in one confirmed action. On the Pages screen he selects pages (one by one, a branch, a type,
  or every page a filter matches) and sets their type by hand, hands them back to the rules, sets or
  clears their template, starts a run or asks the agent. The agent previews and applies too, always
  waiting for him.
- **Templates per type**: each type shows its pages, its template, the pages that keep a template of
  their own and a suggestion; a run on auto takes each page's template from its type, and a run over
  Info pages writes only their SEO pair.
- **The Pages screen** in three views (List, Tree, By type), remembered per site, one selection and
  one set of bulk actions, fast at 5,000 pages. **The Proposed Entities dialog** gained shift-click
  ranges, a whole source type at once, conflicts shown before accepting, the count on the button and
  the keyboard.
- **No new plugin.** 1.5.2 stays.

## Release 2.5.1

The client's first feedback on 2.5.0, in short:

- **The H1 is back.** A page whose theme prints its title as the H1 (Blocksy, the client's theme, and
  block themes) came out of a sync with no H1, which also broke matching a workbook row by its H1.
  The model now keeps the H1 a visitor sees and says who prints it: the theme or the page's content.
  Pages synced on 2.5.0 get theirs on the next ordinary sync, which fetches only those pages again.
- **One chat, in the selected site.** The rail's Agent opened a chat outside the site, where the
  agent could not use the site's tools. It now opens the same chat as the dock, bound to the selected
  site; a chat with no site exists only when no site is selected, and says so.
- **The Proposed Entities dialog**, rebuilt: nearly full-window, the proposals as a tree where
  unticking a parent unticks its children, select all and none, search, and editing a proposal's
  name, parent and keywords before accepting. What the person built stays while he looks elsewhere or
  switches site, until he accepts, starts over or restarts the app.
- **Three screens fit the smallest window with the dock open**: the Runs table, the Models tables
  and the left rail.
- **No new plugin.** 1.5.2 stays.

## Release 2.5.0

The notes as the pull request carries them, in short:

- **What is new**: the site read as a model; page types and field scopes; the workbook reconciled
  through findings; the writer's full picture; edits that keep the client's text; field-scoped runs;
  the agent working the loop; revert per field and per block; pages without an entity made writable
  through proposals; the Site map, the Findings inbox and screens that stay fast at 5,000 pages.
- **What the client does**: upload `postulator-companion.zip` 1.5.2 over the old plugin on every site
  (the Sites panel hands it out); connect a site with its address and an administrator's
  application password; run the first sync (a first walk of 2,000 pages takes about half an hour at
  the default four requests a second); import his workbook and settle the findings; give his pages
  their types and scope the fields each type may write.
- **What to look at first on a real site**: the Overview's observation summary (pages, types,
  trails resolved), the Site map against the site's own breadcrumbs, the Findings inbox's counts by
  code, and one page's Model tab and "What the writer will see".
- **The known limits** are below; the first is that **no run has been made with a funded OpenAI key**.

## Known limits at 2.8.0

- **The race groups and `covergate` are owed** for 2.8.0 (item 5 of **Next, in order**); every
  package passed without `-race`, and both stands' whole suites are green.
- **A sync is paced by the site's request limit**, four a second by default (`wp.rateLimitPerSecond`
  in Settings → WordPress): 400 pages take at least about three and a half minutes, two requests a
  page for the visit and the plugin's render, more on a host slower than a second a request.
- **Permanent harness routes for a stopped and a live sync** need a seeded sync in those states; the
  run pane, the resume notice and the busy buttons were shot by steering a seeded site at an
  unreachable address (`scratchpad/sync-ui`).

## Known limits at 2.7.0

- **The race groups past `application/schedules` and `covergate` are owed** for 2.7.0 (item 1 of
  **Next, in order**); every package in them passed without `-race`.
- **The capability suite is scripted.** Each matrix row is a conversation whose tool calls a script
  makes, so it proves the reads, the plans and the executor end to end over the replica, not which
  calls a real model chooses; that waits for the funded key (under **Next, in order**).
- **A plan step takes as given only the edges an approved `add_edge` step it depends on writes.** An
  adoption or a proposal's write is not assumed, so a run start resting on one goes in a second plan
  once the first has run.
- **The assistant at 5,000 pages**: every view fits the result cap, but a section's links read the
  site's whole audit (2.8 to 4.8 s a call), an entity's page reads the base snapshot (1.5 s), and
  walking all 3,875 entities takes three minutes, a read it has little cause to make.
- **Classification on a site of 5,000 pages**: a preview or a group's page reads in about half a
  second from a fresh site map, and about a second right after a change (the map is refreshed once);
  an apply of 2,750 pages takes 4 to 6 s in batches of 1,000. The Pages Tree shows its first rows in
  0.9 s there, the List and By type in under 0.35 s.
- **Rank Math gives every plain page an Article schema**, so on the client's sites the schema rule
  types such pages as Guide (the replica's `/sample-page/` included) before the guide's word does;
  he reorders or disables the rule in Settings → Classification if that is wrong for a site.
- **No live smoke test yet.** No run has been made with a funded key, so nothing a model call costs
  has been compared with a bill. The input side of the estimate was adopted from offline
  `o200k_base` counts (every measured input within ±25%); the output side stays conservative on
  purpose (`llm.WordTokens` 2.2, `steps.tokensPerWord` 3), and on-demand tool loading
  (`agent.toolLoading: deferred`) stays off.
- **The first real run's checks**: the ledger's `input_tokens` for each step (Settings → Models, or
  `ModelsService.ListCalls`) against `go run ./cmd/tokenreport` for the same pages; then the live
  probe's open tests and the seven checks before tool loading, under **Known gaps**.
- **The plugin's floors**: PHP 7.2 and WordPress 5.7. Without the plugin, on WordPress below 6.0 or
  WooCommerce below 6.9, the front page and the store's pages cannot be learned; Yoast 18.3 on the
  oldest WordPress was not exercised.
- **Scale**: the screens hold at 5,000 pages, but the writer's pack still reads a whole-site snapshot
  (0.7 s at 2,000 pages, 4.6 s at 5,000), and a first walk is paced by the site's request limit.
- **Bulk resolution** settles 1,000 findings per transaction; a refused batch stops the action and the
  dialog says what was settled and what was not.
- **At the window's least width with the dock open** every screen the walk measures fits (2.5.1
  fixed the Runs table, the Models tables and the rail).
- **The rest** is under **Known gaps**, each with its reason.

## Where we are

**2026-10-28, phase 19: 2.8.0, a sync that finishes and says why it did not; one sync per site;
selection that keeps up; page types a person can read.** The reasoning is under "phase 19" in
`DECISIONS.md`.

- **The diagnosis**: the "WordPress error" after ten minutes was our own per-batch step timeout. The
  content stage asked `/terms` (redundant) and `/raw` (even for an empty body) one page at a time,
  about 200 requests in a row a batch; the detail was dropped at every hop; nothing held a site to
  one sync; "Entity" named a type, a role and a column at once.
- **The owner's answers at the plan**: rename and explain; one sync at a time per site; no plugin
  change; one rebuilt Page types screen.
- **The sync** (`2f14cba`, `d539e73`, `9bc5680`): no `/terms` a list carries, no `/raw` for an empty
  body, no variations for a simple product, the site's reads four at a time and SQLite's in turn,
  batches of 50, the client's logger. Measured at 30 ms a request: 1,609 requests in 35.8 s before,
  813 in 10.7 s after.
- **Failures** (`c0bc067`, `f756e4c`, the attempts message): WordPress's status, route, message and
  code; the page or the stage; a step's own time limit as ours; the run's error; `SyncService.Progress`.
- **One sync per site** (`4205eb0`): 0085 and `runs_one_active_sync`, `CONFLICT` with the run under
  way, no sync from `RunsService.Start`.
- **Types** (`032c884`): 0086 names Entity "Article", rule page counts and `beforeRules`, no new Hub
  or Entity type.
- **The screens** (two agents): Proposed Entities' ranges, drag, ctrl-click and "the N shown"
  (`c96748f`, `122f7cb`); every Sync button, the sync pane, the resume notice, the Overview and the
  Sites panel (`069d1f0` to `df567bd`, the header `2680ff8`); one Page types tab with steps, cards and
  rule counts, Graph entity on the Pages screen (`a920c1a`, `4ca9044`, `06d64d2`).
- **The harness** (`6358469`): the fake kept nanoseconds in a restored post's modified time while its
  cursor names the second, so once batches dropped to 50 the seed's first sync listed one page
  forever; a post's time is now cut to the second, as WordPress keeps it. The seed gives Crema Bench
  four pages typed before the rules.
- **2.8.0** in the seven version files. Migrations to 0086; plugin 1.5.2, unchanged.

**2026-10-26/27, phase 18: 2.7.0, the assistant reads, proposes and works; the writer takes a
brief.** The reasoning is under "phase 18" in `DECISIONS.md`; the task list and the capability
matrix, every row proven with its test, are in `docs/superpowers/plans/2026-10-26-phase-18-assistant.md`.

- **Why**: the client asked the assistant to design a section's links and it stopped on an empty
  `ParentId`. The entity read carried no parent (a parent was an edge only), his graph is flat (URL-less
  rows and proposals), the page side answered a bare null for five causes, and no tool walked a
  subtree or the whole tree.
- **The owner's answers at the plan**: a must-link the graph does not connect is refused, the plan adding
  the edge as an earlier step or the run start adding it on request; the catalog's admin tools leave the
  assistant; a plan card edits a step's text inline; no critic.
- **The read layer** (`caedef0`, `c93468b`, `f097cbe`, `f07a5da`, `34497e1`, `8c17ea0`, `f02fbd0`): the
  tree by shape and depth over `page_map` (0081) with its generation guard, every empty parent with
  its reason (`contextpack.ParentWhy`, never stored), every entity with its parent edge or why it has
  none, a body as text block by block, a section's links by page, map, owed, orphans and suggestions,
  many pages found in one paged call; a capped list drops the cursor past the rows it cut (`f9c071f`).
- **The two hierarchies** (`08c02d6`, `c4da6bb`, `84fbc5b`, `b315536`, `6a72f5a`, `17c7c47`):
  `entity_parent_differs` raised by a sync and settled by every parent writer, `graph.AdoptParents`
  previewed and applied under its hash, `adopt_tree` and `keep_graph`, a proposal's and a URL-less row's
  parent from the page tree.
- **The run brief** (`99ffc35`, `28cb864`, `d2aab58`, `19abcc9`): `run.Brief` and `run.Directive` kept by
  0082 and in each step's input hash, refused or made an approved edge at the start, a priority-1 pack
  section read by the writer, the judge and the linker; the writer pane reads a draft or a run.
- **Plans** (`bbe0b0b`, `8fc71c7`, `851c2ab`, `5c6f24b`, `c758e60`): `agent_plans` (0083), each step
  checked as its own tool would check it, `AgentService.ConfirmPlan` whole or in part with inline edits,
  an executor that waits on runs, skips what a failure strands and resumes when the core opens.
- **The fold** (`aed9814`, `17aa23b`, `fd4055d`, `a1dc8dc`, `69c8994`, `9cb666f`, `1deace0`, `f132695`):
  94 tools to 23 by job, a read by `view` and a write by `op` with its own risk and sentence; schemas
  from 68,310 B and 15,222 tokens to 48,543 B and 11,084, a replica round from 17,194 tokens to 13,260;
  history format `responses/2`; the instructions teach read, propose and act; 0084 keeps a card's risk
  (`6204849`).
- **The support bundle** (`6aa5856`, `4636b8e`): `SupportService.WriteBundle`, one JSON file with no
  secret, body, title or argument, which a test proves.
- **Scale** (`25fe201`, `1573da6`, `ddd0857`): every read view fits the 16 KB cap at 5,003 pages; the
  tree, the pages and the entities walk by cursor with no gap and no repeat.
- **The screens** (agent F, `b7e9424` to `982fec8`): the plan card, the run drawer's brief, directives
  and edge offer, the writer pane, the adoption dialog, the finding's copy, the bundle buttons, the
  harness shots.
- **The capability suite** (agent E, `6281db6`, `99a919b`): one scripted conversation per matrix row on
  the replica and the four named tests. It found six defects, fixed test first: the linker's brief
  (`600b1d2`), an edit run a plan step must wait on (`a417a48`), an entity's keywords in a search
  (`6437a42`), an artifact's step within one second (`8e17a20`), an edge a plan's own run start needs
  (`a06d2df`), two summaries (`7106ef2`).
- **2.7.0** (`759edc6`) in the seven version files. Migrations to 0084; plugin 1.5.2, unchanged.

**2026-10-26, phase 17: 2.6.0, page classification by rules, templates per type, the Pages views.**
The reasoning is under "phase 17" in `DECISIONS.md`; the task list is in
`docs/superpowers/plans/2026-10-26-phase-17-classification.md`.

- **The owner's answers at the plan**: a sync re-types the auto pages the rules placed as well as
  the untyped ones (a page typed before 2.6 moves only on an apply); a rule's conditions are AND
  only, so the brief's OR rules are seeded as separate ones.
- **Roles and presets** (`34ccddd`): a preset over a fixed role, every switch by role, Landing,
  Category, Guide and Info added; 0076 gave every existing site the four, a same-role name standing
  for the preset.
- **Rules** (`7f36f91`, `02074b2`): the model, `Validate`, our glob matcher, `Classify` and
  `Explain` root-first over the folded tree; 0077 seeds fourteen rules and the fallback per site,
  equal to Go's; 0078 the reasons and `page_type_rule_id`; CRUD, reorder, deletions repointed.
- **The walk** (`595c307`): derive classifies untyped and rule-placed pages inside its unit, before
  the map refresh, and raises `type_by_default`.
- **Preview, apply, bulk** (`a8b2743`, `9edb27e`, `3a48cde`): the preview's groups, hash and
  dependencies, apply in batches of 1,000 guarded per row; `pages.Selection` resolved in Go with each
  page's path, `SetTypes`, Back to the rules, `SetTemplates`; the List filters, sorts and `Count`
  (0079), `MapRequest.Within` (`c93ef71`, 0080).
- **Templates per type** (`bfa2967`): a type's role, overrides and suggested template. **The body
  switch** (`0f857c4`): a run over Info pages writes their SEO pair alone, and says so.
- **The agent** (`7d5bbc0`): `pagetypes_classify`, preview and a confirmed apply by group number;
  schemas 66,394 B and 14,779 tokens before, 67,165 B and 14,955 after (93 tools).
- **The proposals** name their source page's type (`7997d37`); the dialog's shift-range, a type at
  once, conflicts on the row, the count and the keyboard (`c336ad4`, agent F).
- **The screens** (agent F): the shared selection (`868906f`, `b7320aa`), Settings → Classification
  with the rules, the test on a page and the preview (`1739f62`, `f095024`), Page types (`6491850`),
  the run start's line (`9041e9d`), the Pages screen's List, Tree and By type (`10cb9f8`).
- **Scale** (`7863691`, `30f4dff`, `2b55a29`): the pages list marks the drifted bodies of the pages
  it shows, not the site's, so its reads stay flat; the classification reads the stored site map
  instead of folding the site per request (the facts at 5,003 pages from 2.0 s to 0.39 s, the preview
  from 2.9–3.8 s to 0.48 s); every new read holds its statements from 2,003 to 5,003 pages.
- **The stands** (`9d1477c`, `1c9796a`): each page typed by the rule its census names, the preview
  empty after either sync, a run on auto taking Hub, Hub and Guide by type. The replica's whole suite
  found that a workbook typing a rule-placed page kept the rule's id; the import clears it now
  (`d46da00`).
- **The harness** (`9ac5913`, `74bfbd4`): a third site of 5,000 pages under `task ui:run`, flagging its
  pages without an entity for the proposal dialog; the coffee site typed as 2.6 types a site; the
  walk's shots and timings (agent F, `94d832e`, `5e5f448`, `b4154a0`).
- **2.6.0** (`d803ddc`) in the seven version files. Migrations to 0080; plugin 1.5.2, unchanged.

**2026-10-25, phase 16: 2.5.1, the client's first feedback.** The reasoning is under "phase 16" in
`DECISIONS.md`; the task list is in `docs/superpowers/plans/2026-10-25-phase-16-patch.md`.

- **The visible H1** (`de68e5a`, `edee058`, `e46a764`, `07a7c56`, `655f5f3`):
  - the extraction keeps the first visible h1 and its source;
  - 0075 stores both on the render;
  - one rule (`render.VisibleH1`) feeds the `h1` field from both observation stages;
  - the planned H1 follows only where no plan holds it, and a read-only field is always agreed;
  - `pages.View` carries `observedH1` and `h1Source`.
  Reproduced on 2.5.0's code (20 of 20 published pages on the replica, 11 of 11 on vanilla without
  an H1), then green on both stands with the census and the upgrade proof.
- **One chat per site** (`f90967d`, `548deaa`): `/s/:siteId/agent`, shared with the dock; `/agent`
  for site-less chats only.
- **The proposal dialog** (`c802a6e`): the near full-window tree with cascading ticks, search,
  filters and an inspector that edits name, parent and keywords, its state kept per site and opener.
- **Three layouts at 960** (`1fdb6b4`): runs, models and the rail, each with a `fits` shot.
- **The harness** (`18a6b38`): the fake storefront prints the H1 its plugin reports; the second
  seeded site is found by name.
- **2.5.1** (`869dfa9`) in the seven version files.
- Schemas unchanged (66,394 bytes and 14,779 tokens, 92 tools); migrations to 0075; plugin 1.5.2.

**2026-10-24, phase 15 session 2: the polish, the docs, the build and the pull request.** The
reasoning is under "phase 15, session 2" in `DECISIONS.md`; the task list is in the phase plan.

- **The Site map fits beside the dock** (`1329d97`): the table measures itself and drops the links and
  the children counts, then the parent's source, keeping the tree, the type, the body's state and the
  findings; the page and type columns shrink before anything scrolls. The harness walk gains a `fits`
  step, which fails a shot whose element scrolls sideways, and three Site map shots at 960 × 600 with
  the dock open, narrow and wide; it fails on the Runs table there, as it should.
- **The two oversized screen files are split** (`123fae2`, `004fc29`): the graph screen into its state
  hook, its map view and its empty state (556 lines to 277 at most), the palette into its sections
  and its list (332 to 181 at most), with the harness's graph and palette shots the same before and
  after.
- **Bulk resolution says what a refused batch left** (`b550336`, `b4efaca`): batches of 1,000 stay,
  one transaction each, because one transaction over a 10,000-finding code would hold SQLite's writer
  past the 5 s busy timeout (measured on the 5,003-page store); the dialog names what was settled,
  what was not and why.
- **2.5.0** (`8feaa56`): the version files, and `wails_tools.nsh`, so the installer stops registering
  itself as 2.0.0.
- **The docs**: STATUS for a fresh reader, VISION and ARCHITECTURE with the plugin's floors, the
  findings, the Site map's `page_map` and the `edit` kind, CONTRACTS with `FindingsService`,
  `SiteMap`, `Parents`, the proposals' `source`, 92 tools and the floors' proof, `CLAUDE.md` short.
- Schemas unchanged (66,394 bytes and 14,779 tokens, 92 tools); migrations to 0074; plugin 1.5.2; 137
  bound methods in 16 services.

**2026-10-24, phase 15 session 1: what must hold before the client gets it.** The reasoning is under
"phase 15, session 1" in `DECISIONS.md`; the task list and the checkpoint are in the phase plan.

- **The owner's answers at the plan**: the run start's picker reads the Site map's tree; accepting a
  site proposal approves its parent edge; the old-WordPress stand stays as `STAND=legacy`.
- **The plugin** (agent P, `07d80d4`, `434f3fb`, `7ecb29f`):
  - **Floors**: PHP 7.2 and WordPress 5.7.0 (`wp_robots`), proven by a function scan
    (`tests/functions.php`) and strict tests on PHP 7.2.34, 7.4.33 and 8.5.12.
  - **Version**: 1.5.2, with `wp.PluginMinimum` 1.5.1 still accepted.
  - **The legacy stand** (WordPress 5.7.2, PHP 7.3, WooCommerce 6.5.2, Rank Math 1.0.98.1): the
    contract suite ran 60 passed, with every skip naming what the stand lacks.
- **Pages without an entity** (`d526041` to `0565751`, the screens `42bb440`, the proof `958d677`):
  - **The finding**: `page_without_entity` per writable page.
  - **The site proposal**: `contextpack.Map.EntityProposals` (no model call), behind
    `PreviewFromPages{source: site}`.
  - **The apply**: under the parent, with the edge approved and a clash skipped.
  - **The person's path**: the inbox opens the proposal drawer, for one finding or the whole code.
  - **The agent**: both proposal writers wait for the person in either mode.
  - **The proof on the replica**: a planted product gets its entity under its category's twin hub,
    then its run writes the description with no store key on the wire.
- **The estimate's input side** (`692f710`, `faa5242`, `347a04e`):
  - **The counter**: `llm.Tokens` at 1.7 tokens a word and 3.7 runes a token.
  - **Overheads**: each step's own (writer 1,040, meta writer 440, linker call 290, judge 510 plus
    the body it reads); none scales with the answer.
  - **`Fit`**: it keeps a twentieth of its budget as headroom.
  - **Result**: every measured input is within ±25% of its real count. It used to be 2 to 3.6 times
    too high.
- **Scale on a real store** (agent S, `99f7183` to `ad5e692`):
  - **The test**: a seeded generator of 2,003 and 5,003 pages; the statements of the map, the
    picker, the findings list, the pack, the estimate and the walk are equal at both sizes.
  - **`page_map`** (0074): the folded map, kept fresh by triggers and a generation.
  - **The reads**: the Site map and the findings read it by index.
  - **The picker**: the run start's picker pages the map (`8f215d8`), with `RunsService.Parents`.
- **The prompts** (`16553bd`): `go run ./cmd/tokenreport -prompts out/prompts` writes ten plain-text
  files: the body writer, the meta writer and the judge on three replica pages, and an agent turn's
  first round.
- Schemas 66,394 bytes and 14,779 tokens (92 tools); the instructions with the replica's brief 1,929
  tokens; migrations to 0074; plugin 1.5.2.

**2026-10-24, stage 4.3b: the screens, the full gate, and phase 14 closed.** The reasoning is under
"stage 4.3b" in `DECISIONS.md`; the task list and the checkpoint are under 4.3b in the phase plan.

- **The owner's answers**: the Site map is its own screen and the Pages screen's Tree toggle is gone;
  the flake is fixed (`66517ed`): a harness waits until its test's deadline, and the two scoped sweeps
  took 25.7 to 72.9 s at one CPU, past the old fixed 20 s.
- **Go** (`71de2ee` to `edffadf`): `contextpack.Map`, the site by derived parent with a store
  category folded into its twin hub, read by the findings' subtree and the Site map; findings listed
  and counted by page type and subtree with their pages named; `findings.Explain`, the one sentence
  of a resolution; `findings.ResolveMany` for a code in one unit; `sitemodel.SiteMap`, a node's
  children at a time; the import preview's pick of an ambiguous row's page (`choices`); a
  `FindingsService` and `PagesService.SiteMap` (16 services); `vocab.ts` gains the match rules, the
  bulk options, the option kinds and the body states.
- **The screens** (one agent, `c45cfba` to `8d5fa5e`): the Findings inbox (codes with counts,
  filters in the address, each option with its sentence and kind, who settled what, the agent card's
  body in its confirmation, bulk by code); a run option opens the seeded run start or the rescan
  dialog; the Site map (lazy branches, a hundred children a page, search and type as flat matches with
  trails, twins, the page drawer); the import preview's "Found by" column and its pick; a card per op
  for `pages_fields`, `pages_blocks` and `findings`.
- **The harness** (`074b20c`, `993ad8a`): a client workbook it keeps in its home seeds an ambiguous
  row, a row without a page and a differing trail; shots of the map and the inbox.
- **The gate found one red**: `TestTheClientLoopFromTheSamples` had failed since 4.1 on a sample row
  that named something after all; the sample is fixed (`b75063c`).
- Schemas unchanged: 65,722 bytes and 14,623 tokens (92 tools); migrations to 0073; plugin 1.5.1;
  bound services 16 (`FindingsService` with six methods, `PagesService.SiteMap`).

**2026-10-24, stage 4.3a: real token counts beside bytes, and 4.2's two loose ends.** The reasoning is
under "stage 4.3a" in `DECISIONS.md`; the task list and the checkpoint are under 4.3a in the phase plan.

- **The tokenizer** (`c2b277a`): `tiktoken-go/tokenizer` v0.8.1. It counts `o200k_base` offline and is
  imported only from `_test.go` files and `cmd/tokenreport`. `TestTheTokenizerStaysOutOfTheBinary`
  proves `cmd/postulator` carries none of it. It is verified against the library's own vectors, and
  every count the report makes is byte-identical under v0.10.0.
- **Tokens beside bytes:**
  - `schemaCeilingTokens` 14,626 for the schemas' 14,623 tokens (65,722 bytes) (`60344bb`).
  - `instructionsCeilingTokens` 1,924 for the instructions with the replica's brief, 1,921 tokens
    (`4bd0b17`). To make this measurable, the chat prompts moved into `application/agent`
    (`f994786`).
  - The stand captures load through `sitemodeltest` (`6ac52a2`).
  - `runtime.EstimatedInput` is the estimate's input for one step (`0fe8dbe`).
- **`go run ./cmd/tokenreport`** (`6058376`, `0d2cc0c`, `304eb41`, `85d3889`) is a report, not a gate.
  - Every agent round sends 16,587 tokens on the replica before the conversation (3,668 with tools on
    demand).
  - The writer's prompt is 1,258 to 1,825 tokens on the eight measurable stand pages.
  - The run estimate prices inputs at 2 to 3.6 times their real count.
  - `llm.Tokens` counts a pack at 136% of its real tokens.
  - The captured prose runs 1.77 tokens per word as a writer answers it.
  - Proposed, not changed: 1.8 for `llm.WordTokens`, 1.7 tokens per word and 3.7 runes per token for
    `llm.Tokens`; the writer's ceiling of 3 holds.
- **The replica's client blocks** (`7caa03d`): `TestTheAgentEditsABlock` proves on `/tb-500/`, a hub
  holding the client's two paragraphs and his link, that a run and the agent's edit keep them byte for
  byte. Its first run found that six steps recorded a body's links with no region, resolved against
  the site's root, while the observation records them in the content, resolved against the page. One
  `observe.ContentLinks` now does both (`1c7bf8d`, plus `3207739`, the file that commit left out).
  Green on both stands with the whole loop and the block-by-block edit.
- **The runtime flake**, found and not fixed. The owner stopped the hunt after two runs.
  - `TestATitleSweepIsAnsweredByTheFieldsStepAlone` and `TestAMetaSweepIsNotHeldOnAPageASaveWouldChange`
    in `runtime/steps` time out under `-race` at one CPU.
  - The working cause is starvation against the harness's fixed 20 s wait.
  - Their timeout now names each item's state (`97671c1`).
  - To reproduce: `go test -race -count=10 -cpu 1 -v ./internal/runtime/steps/`.
- Schemas 65,722 bytes and 14,623 tokens (92 tools); migrations to 0073; plugin 1.5.1; no bound
  method moved.

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

`go run ./cmd/tokenreport` (`-json` for JSON) belongs to every gate as a report, not a gate: it prints
what every agent round, every step's prompt and the estimate cost in `o200k_base` tokens. The
tripwires `schemaCeilingTokens` and `instructionsCeilingTokens` are the gate.

**The gate at 2.8.0: 2026-10-28, on Windows, at the close of phase 19.** Exit codes checked.

- `gofmt -l .`, silent; `golangci-lint run ./...`, `--build-tags e2e ./...` and `--build-tags
  uiharness ./cmd/...`: 0 issues each.
- `go test -count=1` over kernel, domain and application, then runtime and adapters, then app,
  transport and cmd: green, but for `internal/app/scale`, which ran past the default ten minutes
  on a machine shared with two agents and found the walk's statement count unequal across sizes
  (57.99 and 58.01 a touched page). Fixed (`d539e73`), the count test alone green: 102 + 58.02 at
  2,003 and 5,003 pages.
- `-race` over `application/observe` and `application/sync`: green.
- `go test -tags uiharness ./cmd/postulator/`: green in 223 s, after the fake's paging fix
  (`6358469`); four runs of the seed in a row green in about 8 s each.
- `task build`: green (bindings 17 services, 154 methods; typecheck; the installer's version 2.8.0).
- `npx vitest run --maxWorkers=2`: 2,631 tests in 224 files, green (218 s).
- **The UI walk** (`task ui:reset`, `task ui:run`, `task ui:walk`, 142 routes): every route green,
  the new Page types, rules, 2.5 notice and proposals-range routes included, but for the three
  plan-card routes (`plan-card`, `plan-card-chosen`, `plan-card-dock`). Those fail on the walk's
  order, which dates from phase 18: an earlier route starts a new conversation that the scripted
  provider titles like the seeded one, so `/s/:siteId/agent` opens it and not the one holding the
  plans. Phase 19 changed nothing they show.
- **`task e2e:full STAND=vanilla -- -v`**: 13 passed, 13 skipped (the replica's own and the degraded
  loop), 0 failed, 6 min 3 s. **`STAND=replica`**: 24 passed, 2 skipped (the capture, the degraded
  loop), 0 failed, 15 min 22 s. **`task e2e:full:noplugin STAND=vanilla`**: the degraded loop green in
  1 min 8 s.
- **Owed**: the race groups and `covergate` (item 5 of **Next, in order**).

**2026-10-27, on Windows, at the close of phase 18, for 2.7.0. On the final code (`7106ef2`), exit
codes checked, one leg at a time; the race groups past `application/schedules` and `covergate` are
owed, below.**

- The comment check and `gofmt -l .`, silent.
- `go vet` plain, `-tags e2e` and `-tags uiharness`, exit 0 each.
- `golangci-lint run` over every package in four chunks and `--build-tags uiharness ./...`, then `task
  lint:e2e` alone: 0 issues each.
- **The race groups** (`-count=1`, coverage, `-p 1`): kernel and domain green in 21 s; the application
  group's first 18 packages green, `schedules` the last. Then `sitemodel`, `sites`, `support`, `sync`,
  `templates` and `tools` stopped at their build with Windows' `0xc0000142` (`compile.exe`,
  `cover.exe` and `vet.exe` could not start with the machine short of memory), and Claude Code's
  low-memory guard stopped the chain in the runtime group; its `go test` outlived the guard and was
  stopped by hand. The rest of the application group, the runtime, adapters and last groups and
  `covergate` are **owed** (the commands are item 1 of **Next, in order**); every package in them
  passed without `-race` during the phase.
- `go test -count=1 ./internal/app/scale/`: the view walk in 233 s (the tree's 4,503 rows in 4.6 s, the
  pages in 40 s, the 3,875 entities in 2 min 4 s, none missed and none twice), the rest in 73 s.
- `npm run typecheck` (14 s); `npm run test:run -- --maxWorkers=2`, 2,475 tests in 210 files (119 s);
  `task ui:lint`, 0 issues and the harness tests (133 s).
- `task build VERSION=2.7.0` (120 s) with no generated drift.
- **Vanilla**, provisioned fresh: `task e2e:full -- -v -timeout 40m`, 13 passed and 13 skipped (the nine
  replica-only capability tests and the four skipped by design), 368 s; `task e2e:full:noplugin`, 45 s;
  taken down.
- **The replica**, provisioned fresh: `task e2e:full -- -v -timeout 40m`, 24 passed, every capability
  test among them, and 2 skipped by design, 966 s; taken down. Before it, the four tests the suite's
  first run had left red passed alone in 200 s after their fixes.
- The plugin did not change, so its lint and tests were not run.
- `task package VERSION=2.7.0` (31 s): `bin/postulator-amd64-installer.exe`, 14,510,806 bytes,
  product and file version 2.7.0; the executable's file version is 2.7.0.0.
  `bin/postulator-companion.zip` is still plugin 1.5.2.
- `go run ./cmd/tokenreport`: 23 tools, 48,543 bytes and 11,084 tokens of schema (68,310 and 15,222 for
  94 tools at the phase's start); the autonomous instructions with the replica's brief 2,133 tokens; a
  replica round 13,260 tokens (17,194 before).

**2026-10-26, on Windows, at the close of phase 17, for 2.6.0. On the final code (`ed70209`), exit
codes checked, one leg at a time; four race groups and `covergate` are owed, below.**

- The comment check and `gofmt -l .`, silent.
- `go vet` plain, `-tags e2e` and `-tags uiharness`.
- `golangci-lint run` over every package in four chunks, `--build-tags uiharness`, then `task
  lint:e2e` alone: 0 issues each.
- **The race groups**: kernel and domain, 28 packages, `-count=1` with coverage, green in 109 s. The
  application group at `-p 2` was stopped by Claude Code's low-memory guard after seven packages had
  passed, while another job on the machine held several gigabytes, so the application, runtime,
  adapters and the last group and `covergate` are **owed** (the commands are under **Next, in
  order**). Every package of those groups passed without `-race` during the phase.
- `go test -count=1 ./internal/app/scale/`, 322 s, after the plan check's sample moved off the title
  sort 0079 now indexes.
- `npm run typecheck`; `npm run test:run -- --maxWorkers=2`, 2,326 tests in 201 files; `task
  ui:lint`, 0 issues and the harness tests in 379 s. Agent F's whole walk: 128 of 128 shots from one
  reset, the Pages views, Classification and Page types at 960 × 600 with the dock and wide with
  `fits`, a bulk type change from By type and the proposal dialog over 439 proposals on the 5,000-page
  site.
- `task build VERSION=2.6.0` (127 s) with no generated drift.
- **The replica**, provisioned fresh: `task e2e:full -- -v`, 15 passed and 2 skipped by design, 594 s,
  after the import fix its first run found; taken down.
- **Vanilla**, the same: `task e2e:full -- -v`, 13 passed and 4 skipped by design, 354 s; `task
  e2e:full:noplugin`, 54 s; taken down.
- The plugin did not change, so its lint and tests were not run.
- `task package VERSION=2.6.0` (122 s): `bin/postulator-amd64-installer.exe`, 14,238,551 bytes,
  product and file version 2.6.0; the executable's file version is 2.6.0.0.
  `bin/postulator-companion.zip` is still plugin 1.5.2.
- `go run ./cmd/tokenreport`: 93 tools, 67,165 bytes and 14,955 tokens of schema (66,394 and 14,779
  at 2.5.1); `pagetypes_classify` 547 bytes and 124 tokens; the instructions with the replica's brief
  1,929 tokens, unchanged.

**2026-10-25, on Windows, at the close of phase 16, for 2.5.1. Green on the final code (`743e4e4`),
every leg, exit codes checked:**

- The comment check and `gofmt -l .`, silent.
- `go vet` plain, `-tags e2e` and `-tags uiharness`.
- `golangci-lint run ./...` and `--build-tags uiharness`, then `task lint:e2e` alone: 0 issues each.
- **The race groups**, `-count=1` with coverage, in sequence:
  - kernel and domain (57 s);
  - application at `-p 2` (1,069 s);
  - runtime at `-p 1` (301 s);
  - adapters at `-p 1 -timeout 30m` (643 s);
  - every package that uses `wptest`, again after its storefront changed (330 s);
  - app, transport, cmd and frontend at `-p 1` (365 s).
- `covergate` over the merged profiles: domain and application 88.06% of 17,915 statements (gate
  80%), total 87.67% of 35,848 (gate 70%).
- `go test -count=1 ./internal/app/scale/`, 129 s.
- `npm run typecheck`; `npm run test:run -- --maxWorkers=2`, 2,090 tests in 195 files (245 s);
  `task ui:lint`, the harness tests in 168 s. Agent F's walk: 101 routes, the new `fits` shots
  included.
- `task build VERSION=2.5.1` with no generated drift; the executable's file version is 2.5.1.0.
- **The replica**, provisioned fresh: `task e2e:full -- -v`, 15 passed and 2 skipped by design, 674 s;
  taken down.
- **Vanilla**, the same: `task e2e:full -- -v`, 13 passed and 4 skipped by design, 338 s; then `task
  e2e:full:noplugin`, 32 s; taken down.
- Before the fix, a throwaway test on 2.5.0's code, run on both stands, found 20 of 20 published pages
  on the replica and 11 of 11 on vanilla without the H1 their render shows.
- The plugin did not change, so its lint and tests were not run.
- `task package VERSION=2.5.1`: `bin/postulator-amd64-installer.exe`, 14,006,773 bytes, product and
  file version 2.5.1. `bin/postulator-companion.zip` is still plugin 1.5.2.
- `go run ./cmd/tokenreport`: 92 tools, 66,394 bytes and 14,779 tokens of schema, unchanged.

**2026-10-24, on Windows, at the close of phase 15 session 2, for 2.5.0. Green on the final code
(`03265bc`), every leg, exit codes checked, one leg at a time:**

- The comment check over 1,141 Go files and `gofmt -l .`, silent.
- `go vet` plain, `-tags e2e` and `-tags uiharness`; `golangci-lint run ./...` and `--build-tags
  uiharness`, then `task lint:e2e` alone: 0 issues each.
- The five race groups, `-count=1` with coverage, in sequence: kernel and domain 28 packages (130 s);
  application 22 at `-p 2` (758 s, `imports` 496 s); runtime 3 at `-p 1` (257 s); adapters 30 at
  `-p 1 -timeout 30m` (618 s, `sqlite` 330 s); app, transport, cmd and frontend 10 at `-p 1` (510 s).
  `covergate` over the five profiles merged one line per block: domain and application 88.05% of
  17,800 statements (gate 80%), total 87.66% of 35,629 (gate 70%).
- `go test -count=1 ./internal/app/scale/` (it skips under `-race`), 91 s.
- `task plugin:lint` and `task plugin:test` on PHP 7.2.34, 7.4.33 and 8.5.12: 14 files parse clean;
  the function scan (112 functions, 23 hooks, the newest `wp_robots_no_robots` at 5.7.0), 53 href
  cases and 35 meta-key cases on each.
- `task build VERSION=2.5.0` (246 s) with no generated drift; its first try stopped on a transient
  `EBUSY` while `npm ci` emptied `node_modules`, and the retry was clean. The executable's file
  version is 2.5.0.0.
- `npm run typecheck`; `npm run test:run -- --maxWorkers=2`, 1,954 tests in 188 files (181 s);
  `task ui:lint`, the harness tests in 124 s. The walk's three Site map shots at 960 × 600 with the
  dock open pass `fits`.
- **The replica**, after `task e2e:reset` and `task plugin:deploy`: `task e2e:full -- -v`, 14 passed
  and 2 skipped by design, 553 s; taken down.
- **Vanilla**, the same: `task e2e:full -- -v`, 12 passed and 4 skipped by design, 375 s; then `task
  e2e:full:noplugin`, 176 s; the plugin restored with `e2e:up`; taken down.
- `task package VERSION=2.5.0`: `bin/postulator-amd64-installer.exe`, 13,995,424 bytes, product and
  file version 2.5.0. `task plugin:zip`: `bin/postulator-companion.zip`, 15,098 bytes, plugin 1.5.2,
  `Requires PHP: 7.2`, `Requires at least: 5.7`, packed by the same `plugin.Pack` the Sites panel
  uses.
- `go run ./cmd/tokenreport`: 92 tools, 66,394 bytes and 14,779 tokens of schema, unchanged; a
  deferred round would send 7,671 bytes and 1,704 tokens of tools.

**2026-10-24, on Windows, at the close of phase 15 session 1. Green on the final code, every leg:
the record below, then the owed legs run on the owner's word the same evening.**

- **The owed legs, run green.** Every other race profile came from the record below, on the same Go
  code.
  - **The application race group**, in three foreground chunks, each with its own profile:
    `imports` (193 s); `graph` and `tools`; the other 21 packages at `-p 2`.
  - **`covergate`** over all eight profiles, merged one line per block: domain and application
    88.05% of 17,800 statements (gate 80%), total 87.66% of 35,629 (gate 70%).
  - **The replica, from a fresh reset after `task plugin:deploy`**: `internal/e2e` in its own order,
    in three chunks; 14 passed and 2 skipped by design.
    - **The first run found one red**: `TestTheWriterSeesTheFullPicture` read the client's hub
      `/healing-peptides/` with a link sentence that the proposal proof's run had relinked into it.
      The planted product had been deleted, but the sentence stayed. The proof now reverts its run
      and asserts the hub is back byte for byte (`feaf2d6`); a second run from a fresh reset was
      green throughout.
  - **Vanilla, from a fresh reset**: the same three chunks, 12 passed and 4 skipped by design. Then
    `task e2e:full:noplugin`, the plugin restored with `e2e:up`, and the stand taken down.

The first record, from before the owner's word:

- **Run and green on the final code:**
  - `task build` with no generated drift;
  - `go vet` plain, `-tags e2e` and `-tags uiharness`; `gofmt -l .` silent; the comment check;
  - `golangci-lint run ./...` and `--build-tags uiharness`, then `task lint:e2e` alone, 0 issues
    each;
  - the race groups:
    - kernel and domain, 28 packages;
    - runtime at `-p 1`, `runtime` 157 s and `steps` 67 s;
    - adapters at `-p 1 -timeout 30m`, 28 packages with `sqlite` at 458 s. `wptest` failed once
      inside the group, every failure a dial to the fake's own 127.0.0.1 port timing out on a
      loaded machine; alone it passed in 5 s;
    - app, transport, cmd and frontend at `-p 1`, 10 packages;
  - `go test -count=1 ./internal/app/scale/` (it skips under `-race`), 113 s;
  - `task plugin:lint` and `task plugin:test` on PHP 7.2.34, 7.4.33 and 8.5.12;
  - `npm run typecheck`; `npm run test:run -- --maxWorkers=2`, 1,943 tests in 187 files;
    `task ui:lint`, the harness in 195 s;
  - `go run ./cmd/tokenreport`: 92 tools, 66,394 bytes and 14,779 tokens; every measured input
    within ±25%.
- **On the stands, before the gate:**
  - the replica's fresh capture, then `TestAProductWithoutAnEntityIsWrittenThroughAProposal`, green;
  - the legacy stand's contract suite (agent P), 60 passed and none failed.
- **Stopped by the memory guard, owed.** Other sessions held the machine; once the WSL VM had kept
  7.3 GB of cache, 1.3 GB stayed free.
  - The application race group: `internal/application/...`. The packages S touched passed race at
    `-p 1` during S's work, and every application package passed without race during the session.
  - Therefore `covergate`.
  - The replica's `e2e:full`, stopped while it compiled; no test had finished, and vanilla's was
    never started.
- **The commands owed**, one at a time on an idle machine (`wsl --shutdown` first frees the VM's
  cache; Docker Desktop then needs a start). The merge keeps one line per block, because a package
  measured twice would otherwise count twice:
  ```
  go test -race -count=1 -p 2 -covermode=atomic -coverprofile=cov1.out ./internal/kernel/... ./internal/domain/...
  go test -race -count=1 -p 1 -timeout 40m -covermode=atomic -coverprofile=cov2.out ./internal/application/...
  go test -race -count=1 -p 1 -timeout 30m -covermode=atomic -coverprofile=cov3.out ./internal/runtime/...
  go test -race -count=1 -p 1 -timeout 30m -covermode=atomic -coverprofile=cov4.out ./internal/adapters/...
  go test -race -count=1 -p 1 -timeout 40m -covermode=atomic -coverprofile=cov5.out ./internal/app/... ./internal/transport/... ./cmd/... ./frontend/...
  awk 'FNR==1{next} {k=$1" "$2; if (!(k in c) || $3>c[k]) c[k]=$3} END {print "mode: atomic"; for (k in c) print k, c[k]}' cov1.out cov2.out cov3.out cov4.out cov5.out > coverage.out
  go run ./cmd/covergate
  task e2e:reset STAND=replica ; task plugin:deploy STAND=replica ; task e2e:full STAND=replica -- -v ; task e2e:down STAND=replica
  task e2e:reset STAND=vanilla ; task plugin:deploy STAND=vanilla ; task e2e:full STAND=vanilla -- -v
  task e2e:full:noplugin STAND=vanilla ; task e2e:up STAND=vanilla ; task e2e:down STAND=vanilla
  ```

**2026-10-24, on Windows, at the close of stage 4.3b and phase 14. Green on the final code, every
leg.**
- `go build ./...`; `go vet ./...`, `-tags e2e` and `-tags uiharness`; `gofmt -l .` silent; the
  comment check; `golangci-lint run ./...`, `--build-tags uiharness` and `task lint:e2e` (alone), 0
  issues each.
- `go test -race -count=1` in five groups: kernel and domain (28 packages), application (22),
  runtime at `-p 1` (3; `runtime` 187 s, `steps` 71 s), adapters at `-p 1 -timeout 30m` (sqlite 258 s,
  `TestMigrationsRoundTrip` to 0073 within it), app, transport, cmd and frontend (11);
  `go run ./cmd/covergate`: domain and application 88.43% of 17,008 statements (gate 80%), total
  87.81% of 34,463 (gate 70%).
- The flake's check: `go test -race -count=3 -cpu 1 -timeout 60m ./internal/runtime/steps/`, 839 s.
- `task plugin:lint` and `task plugin:test` on PHP 7.4 (53 href cases, 35 meta-key cases).
- `task build` with no generated drift; `npm run typecheck`; `npm run test:run -- --maxWorkers=2`,
  1,934 tests in 185 files; `task ui:lint` (the harness tests in 115 s); the harness window seen on the
  inbox and the map.
- `go run ./cmd/tokenreport`: the schemas did not move.
- **The replica**, after `task e2e:reset` and `task plugin:deploy`: stage 4's exit tests
  (`TestTheWorkbookReconcilesWithTheSite`, `TestTheAgentEditsABlock`, `TestTheAgentSeesTheSite`,
  `TestTheWholeLoopReachesTheDockerSite`, 152 s); then from a fresh reset `TestTheSiteIsObservedIntoTheModel`
  alone for the census, and `task e2e:full` (13 passed, 2 skipped by design, 754 s with the reset). The
  first `e2e:full` failed `TestTheClientLoopFromTheSamples`, fixed in `b75063c` and run again in full.
- **Vanilla**, the same: the exit tests (`TestTheAgentSeesTheSite` skips there), the census,
  `task e2e:full` (10 passed, 5 skipped by design), `task e2e:full:noplugin`, the plugin restored, 656 s
  in all. Both stands were taken down.

**2026-10-24, on Windows, at the close of stage 4.3a. Green, with the flake owed.**
- **Run and green:**
  - `go build ./...`; `go vet ./...` and its `-tags e2e` and `-tags uiharness` builds.
  - `gofmt -l .` silent; the comment check; `golangci-lint run ./...` 0 issues; `task lint:e2e` 0
    issues.
  - `go test -race -count=1 -p 2` over the packages 4.3a touched, ten packages in 494 s: `cmd/tokenreport`,
    `tools`, `sitemodel`, `application/agent`, `observe`, `transport/agent`, `runtime`, `runtime/steps`
    and `app`, with `sitemodeltest` holding no tests.
  - `go run ./cmd/tokenreport`, plain and `-json`.
  - On the replica, then vanilla, each after `task plugin:deploy`: `TestTheAgentEditsABlock`,
    `TestTheWholeLoopReachesTheDockerSite` and `TestAnExistingPageIsEditedBlockByBlock`. Both stands
    were taken down and Docker Desktop stopped.
- **The flake hunt**:
  - `-race -count=10 -p 1 ./internal/runtime/...` was green in 3,219 s.
  - The same with `-cpu 1,4` failed two scoped sweeps in `runtime/steps` at one CPU (8,484 s).
  - The owner stopped the hunt there; the fix is owed.
- **Not run, 4.3b's:** race in five groups, `covergate`, the whole vitest, `ui:lint`, the stands' full
  suites.

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
task e2e:up · test · down STAND=legacy   WordPress 5.7.2 on PHP 7.3 on 8086, the plugin's contract suite only
task e2e:both                     both stands up and their suites, one after the other
task plugin:deploy STAND=         the zip reinstalled over the plugin on a running stand
task plugin:lint · plugin:test    every plugin file and test on PHP 7.2, 7.4 and 8.5 (the href table, the meta deny list, the function scan)
go run ./cmd/tokenreport -prompts out/prompts   the steps' and the agent's prompts as plain text, beside the report
go test -count=1 ./internal/app/scale/          the 2,000- and 5,000-page counts, which a -race run skips
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
| 14 | The site model (2026-10-17): stage 0 audit, contract, design and plan; stages 1 to 4 in `2026-10-17-phase-14-site-model.md` | done, closed 2026-10-24 with the whole gate green on both stands |
| 15 | The release (2026-10-24), two sessions in `2026-10-24-phase-15-release.md` | done on 2026-10-24: session 1 (the plugin's floors, pages without an entity, the input estimate, scale, the prompts) and session 2 (the polish, the docs, 2.5.0 built, the pull request open) |
| 16 | 2.5.1, the client's first feedback (2026-10-25), `2026-10-25-phase-16-patch.md` | done; built, in PR #32 |
| 17 | 2.6.0, classification by rules, templates per type, the Pages views (2026-10-26), `2026-10-26-phase-17-classification.md` | done; built, in PR #32 |
| 18 | 2.7.0, the assistant reads, proposes and works; the run brief (2026-10-26/27), `2026-10-26-phase-18-assistant.md` | done, every matrix row proven; built, in PR #32 |

## Known gaps

- **Phase 15 left these:**
  - **No funded key has run**: see **Known limits at 2.6.0**.
  - **The UI at the window's least width with the dock open**: the Runs table scrolls sideways
    inside its panel (the walk's `fits` step names it: 726 px of content in 288 px), the Models
    table shows about ten characters of a model's name, and the rail shows a horizontal scrollbar
    at 600 px high. The Site map fits.
  - **Three feature files are still over 300 lines**: `graph/canvas/map.tsx` (849, the canvas
    renderer), `agent/conversation/model/markdown.ts` (494) and `links/panel.tsx` (311).
  - **Whole-tree reads**: the Overview's drift count and the Reports' run panel and site tab still
    read `Pages.Tree`, the whole tree. The run start's picker no longer does, but `Pages.Tree` stays
    while they call it.
  - **"Select the whole branch" is gone from the run start's picker**, since a branch loads a page
    at a time; a server pick of a node's descendants (`page_map`'s trail makes it one query) would
    bring it back.
  - **The writer's pack reads a whole-site snapshot** once per pack and once per estimate: the same
    statements at 2,003 and 5,003 pages, but 0.7 s against 4.6 s.
  - **The observation's content and store stages read all pages every batch**, since a cached row
    would let an update overwrite an edit made during the stage.
  - **`page_map`'s triggers**: a migration that rebuilds `pages`, `page_fields` or `page_types`
    must recreate them.
  - **Without the plugin, on WordPress below 6.0 or WooCommerce below 6.9**, the front page or the
    store's pages cannot be learned; with it, `/settings` answers both. Yoast 18.3 on the legacy
    stand was not exercised.
  - **The judge's and the meta writer's measured prompts read stand-ins** of what a run's writer
    would write.
  - **The estimate's sample is nine pages** of mostly WordPress boilerplate.

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

1. **The owner merges the 2.5.0 pull request and tags `v2.5.0`**; then plugin 1.5.2 goes onto the
   client's sites and the first sync runs on a real one.
2. **The smoke with a funded OpenAI key**, whenever there is one: one generate run with the writer on flex
   (the tier served and the reasoning shown in the ledger), one agent turn with tools (cached
   tokens from its second round), and the spend panel naming both by purpose and model; then the
   probe's open tests and the seven checks under **Known gaps**, and only then a decision on
   `agent.toolLoading`.
3. **A sandbox walk with the client's sheets and a real provider** (`task sandbox:up
   E2E_PLUGIN=1 E2E_WOO=1`, after `task sandbox:reset` if it still carries plugin 1.3.0 and its
   terms): import `samples/client-sheets.xlsx` as one workbook and check each sheet's tab, the
   Groups segment, the Category and Subcategory columns compared as each row's expected trail, and
   the findings the import raises; sync; run a page under TB-500 › Liquid; create a product by hand
   in WooCommerce under a category of the client's own, settle its `page_without_entity` through a
   proposal, run it and see its categories untouched; revert both.
4. **The client's own sites**: what a real model writes into a short description and a body, and
   whether a page builder hides the description, which `product_description_hidden` will say.
5. The residue above: the denied tool row's decision, the narrow-width UI items (the Runs table, the
   Models table, the rail), the last callers of `Pages.Tree`, the three feature files over 300 lines,
   `ProposeFromPages` and `Import.Apply` as runs, and `settings.changed` for a declared value.
