//go:build e2e

package e2e_test

import (
	"encoding/json"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/reports"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/application/sites"
	appsync "github.com/davidmovas/postulator/internal/application/sync"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

// degraded runs over the three pages that sit two levels down, so every draft has both a
// parent and a grandparent to link to.
var degraded = []target{
	{path: "/menu/main-courses/steaks/", keyword: "grilled steaks"},
	{path: "/menu/main-courses/seafood/", keyword: "fresh seafood"},
	{path: "/menu/main-courses/pasta/", keyword: "handmade pasta"},
}

const (
	newsBody = `<h1>Menu news</h1><p>The kitchen rewrote <a href="/menu/">our menu</a> for the spring, ` +
		`and the card changes again in June.</p>`

	newsPath = "/menu-news/"

	// the paths of rows the core pull cannot see: WooCommerce is installed on the stack, but
	// core REST lists neither products nor product categories.
	productPath      = "/product/postulator-powder/"
	productTermPath  = "/product-category/postulator-koffein/"
	vanishedPagePath = "/menu/gone/"
)

var hrefPattern = regexp.MustCompile(`href="([^"]*)"`)

// TestTheWholeLoopDegradesWithoutThePlugin drives the whole product loop against a live
// WordPress that carries the companion plugin deactivated, which is the shape of a client who
// refuses to install it. Everything the plugin answers has to degrade, and none of it may
// fail the run, lose the work silently or write over what a human wrote.
func TestTheWholeLoopDegradesWithoutThePlugin(t *testing.T) {
	live := newSite(t)
	requirePlugin(t, live.env, false)
	live.clearCore(t)

	menuID := live.publish(t, "Our Menu", "menu", menuBody, 0)
	mainsID := live.publish(t, "Main Courses", "main-courses", mainsBody, menuID)
	live.publishPost(t, "Menu news", "menu-news", newsBody)

	core := openCore(t)
	owner, err := core.Sites.Create(t.Context(), sites.CreateRequest{
		Name:          "Docker Shop Without The Plugin",
		BaseURL:       live.env.baseURL,
		Username:      live.env.user,
		Password:      live.env.pass,
		AllowInsecure: true,
	})
	if err != nil {
		t.Fatalf("create the site: %v", err)
	}
	siteID := owner.Site.ID

	checked, err := core.Sync.CheckPlugin(t.Context(), appsync.CheckPluginRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("check the plugin: %v", err)
	}
	if checked.Plugin.Installed || len(checked.Plugin.Capabilities) != 0 {
		t.Fatalf("the plugin check reports %+v, want an absent plugin with no capabilities", checked.Plugin)
	}
	assertPluginCallsAreRefusedAsInvalid(t, core, siteID, int64(menuID))

	seedUnseeable(t, core, siteID)

	// the site map arrives through core REST
	firstSync := runSync(t, core, siteID)
	stored := pagesByPath(t, core.Pages, siteID)
	assertCorePull(t, core, siteID, stored)
	assertNothingUnseeableWasArchived(t, stored)

	secondSync := runSync(t, core, siteID)
	assertSameMap(t, stored, pagesByPath(t, core.Pages, siteID))
	t.Logf("the first sync %+v, the second %+v", firstSync, secondSync)

	applied, err := core.Imports.Apply(t.Context(), imports.ApplyRequest{
		SiteID:  siteID,
		Path:    samplePath(t),
		Mapping: detected(t, core, siteID),
	})
	if err != nil {
		t.Fatalf("apply the import: %v", err)
	}
	if len(applied.Report.Errors) != 0 {
		t.Fatalf("the import reported %+v", applied.Report.Errors)
	}

	stored = pagesByPath(t, core.Pages, siteID)
	targets := make([]string, 0, len(degraded))
	for _, planned := range degraded {
		page, ok := stored[planned.path]
		if !ok || page.Status != string(pagemap.StatusPlanned) {
			t.Fatalf("the import left %s as %+v, want a planned page", planned.path, page)
		}
		targets = append(targets, page.ID)
	}

	// the neighbors as a human left them, to compare against once the run has finished
	untouched := map[int]string{
		menuID:  live.storedContent(t, "pages", menuID),
		mainsID: live.storedContent(t, "pages", mainsID),
	}

	started, err := core.Runs.Start(t.Context(), runs.StartRequest{
		SiteID:      siteID,
		PageIDs:     targets,
		PublishMode: string(run.PublishDraft),
		Recipe:      recipe(),
	})
	if err != nil {
		t.Fatalf("start the run: %v", err)
	}
	awaitRun(t, core.Runs, started.RunID)

	items, err := core.Runs.ListItems(t.Context(), runs.ListItemsRequest{
		RunID: started.RunID, ListRequest: dto.ListRequest{Limit: 50},
	})
	if err != nil {
		t.Fatalf("list the run items: %v", err)
	}
	if len(items.Items) != len(degraded) {
		t.Fatalf("the run carries %d items, want %d", len(items.Items), len(degraded))
	}

	drafts := make(map[string]int64, len(degraded))
	for i := range items.Items {
		item := items.Items[i]
		if item.Status != string(run.StatusCompleted) {
			t.Fatalf("the item for %s is %q: %s%s", item.TargetID, item.Status, item.Error,
				artifactDump(t, core.Runs, item.ID, string(run.ArtifactValidationReport)))
		}

		final := finalReport(t, core.Runs, item.ID)
		if final.Validation == nil || final.Validation.Compliance.Score != 1 {
			t.Fatalf("%s reports compliance %+v, want 1", final.Path, final.Validation)
		}
		if final.Publish == nil || final.Publish.Status != "draft" || final.Publish.WPID == 0 {
			t.Fatalf("%s reports %+v, want a draft on the site", final.Path, final.Publish)
		}
		drafts[final.Path] = final.Publish.WPID

		assertSEOWasSkippedNotLost(t, core, item.ID, final)
		assertRelinkStoodDown(t, final)
		if final.Sync == nil || final.Sync.Source != "core" {
			t.Fatalf("%s read back through %+v, want the core source", final.Path, final.Sync)
		}
	}

	if len(drafts) != len(degraded) {
		t.Fatalf("the run wrote %d drafts, want %d: %v", len(drafts), len(degraded), drafts)
	}
	for _, planned := range degraded {
		wpID, ok := drafts[planned.path]
		if !ok {
			t.Fatalf("no draft was written for %s", planned.path)
		}
		assertDraftLinksUp(t, live, planned.path, int(wpID))
		assertNoSEOMetaOnTheSite(t, live, int(wpID))
	}

	for wpID, before := range untouched {
		if after := live.storedContent(t, "pages", wpID); after != before {
			t.Fatalf("the stored content of %d changed; a degraded relink must leave it alone.\n"+
				"before: %q\nafter:  %q", wpID, before, after)
		}
	}

	after := pagesByPath(t, core.Pages, siteID)
	for _, planned := range degraded {
		page, ok := after[planned.path]
		if !ok || page.WPID == nil || page.Status != string(pagemap.StatusExists) {
			t.Fatalf("the read back left %s as %+v, want an existing page with a wordpress id",
				planned.path, page)
		}
		assertLinksUpInTheMap(t, core.Pages, after, page)
	}

	overview, err := core.Reports.SiteOverview(t.Context(), reports.SiteOverviewRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("site overview: %v", err)
	}
	if overview.Pages.Total != len(after) {
		t.Fatalf("the overview counts %d pages, the store holds %d", overview.Pages.Total, len(after))
	}
	if overview.Pages.ByStatus[string(pagemap.StatusExists)] < len(degraded) {
		t.Fatalf("the overview reports %+v, want the run's pages", overview.Pages.ByStatus)
	}
	if overview.Edges.Approved == 0 || overview.Edges.Realized == 0 ||
		overview.Edges.Realized > overview.Edges.Approved {
		t.Fatalf("the overview reports %+v; the drafts link up, so some edges are realized and "+
			"the relink that stood down cannot have realized more than were approved", overview.Edges)
	}

	t.Logf("%d drafts under /menu/main-courses/, %d pages in the store, %d of %d edges realized",
		len(drafts), len(after), overview.Edges.Realized, overview.Edges.Approved)
}

func runSync(t *testing.T, core *app.Core, siteID string) string {
	t.Helper()

	synced, err := core.Sync.SyncSite(t.Context(), appsync.SyncSiteRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("sync the site: %v", err)
	}
	awaitRun(t, core.Runs, synced.RunID)
	return synced.RunID
}

// assertPluginCallsAreRefusedAsInvalid checks that every call the companion plugin answers
// fails as a plugin_missing refusal the caller can read, never as an unexplained failure.
func assertPluginCallsAreRefusedAsInvalid(t *testing.T, core *app.Core, siteID string, wpID int64) {
	t.Helper()

	client, err := core.WordPress.Client(t.Context(), siteID)
	if err != nil {
		t.Fatalf("build the WordPress client: %v", err)
	}

	calls := map[string]func() error{
		"manifest":     func() error { _, callErr := client.Manifest(t.Context()); return callErr },
		"capabilities": func() error { _, callErr := client.Capabilities(t.Context()); return callErr },
		"content": func() error {
			_, callErr := client.ListContent(t.Context(), wp.ContentQuery{})
			return callErr
		},
		"seo meta": func() error {
			_, callErr := client.SetSEOMeta(t.Context(), wpID, wp.SEOMeta{Title: "Our Menu"})
			return callErr
		},
		"raw read": func() error { _, callErr := client.GetRaw(t.Context(), wpID); return callErr },
		"raw write": func() error {
			_, callErr := client.PutRaw(t.Context(), wpID, "<p>never written</p>", "")
			return callErr
		},
	}

	for name, call := range calls {
		err = call()
		if !wp.IsPluginMissing(err) {
			t.Errorf("%s = %v, want a plugin_missing refusal", name, err)
		}
		if code := errors.CodeOf(err); code != errors.Invalid {
			t.Errorf("%s failed with %q, want %q", name, code, errors.Invalid)
		}
	}
}

// seedUnseeable writes the rows a core pull cannot see. The products come from WooCommerce,
// which the stack installs and core REST does not list, and the vanished page stands for a
// page the sync really should archive, so the assertion proves the sweep ran at all.
func seedUnseeable(t *testing.T, core *app.Core, siteID string) {
	t.Helper()

	repo := sqlite.NewPageRepo(core.Store)
	stale := time.Now().Add(-time.Hour).UTC().Truncate(time.Second)

	for path, wpType := range map[string]pagemap.WPType{
		productPath:      pagemap.WPProduct,
		productTermPath:  pagemap.WPProductCategory,
		vanishedPagePath: pagemap.WPPage,
	} {
		wpID := int64(900000 + len(path))
		page, err := pagemap.NewPage(pagemap.Page{
			ID: id.New(), SiteID: siteID, Path: path, WPType: wpType, Title: "Seeded " + path,
			Status: pagemap.StatusExists, WPID: &wpID, LastSyncedAt: &stale,
			CreatedAt: stale, UpdatedAt: stale,
		})
		if err != nil {
			t.Fatalf("build the seeded page %s: %v", path, err)
		}
		if err = repo.Insert(t.Context(), page); err != nil {
			t.Fatalf("insert the seeded page %s: %v", path, err)
		}
	}
}

func assertCorePull(t *testing.T, core *app.Core, siteID string, stored map[string]pages.Page) {
	t.Helper()

	for path, wpType := range map[string]string{
		"/menu/":              string(pagemap.WPPage),
		"/menu/main-courses/": string(pagemap.WPPage),
		newsPath:              string(pagemap.WPPost),
	} {
		page, ok := stored[path]
		if !ok || page.WPID == nil {
			t.Fatalf("the core pull did not record %s: %+v", path, page)
		}
		if page.WPType != wpType {
			t.Fatalf("%s is a %q, want a %q", path, page.WPType, wpType)
		}
	}

	parent, child := stored["/menu/"], stored["/menu/main-courses/"]
	if child.ParentPageID == nil || *child.ParentPageID != parent.ID {
		t.Fatalf("%s hangs off %v, want the row of %s", child.Path, child.ParentPageID, parent.Path)
	}

	// the links were parsed here, out of the content core REST returned
	links := linksOf(t, core.Pages, stored[newsPath].ID)
	if len(links) != 1 || links[0].ToURL != "/menu/" {
		t.Fatalf("%s carries the links %+v, want the one to /menu/", newsPath, links)
	}
	if links[0].ToPageID == nil || *links[0].ToPageID != parent.ID {
		t.Fatalf("the link of %s resolved to %v, want the row of /menu/", newsPath, links[0].ToPageID)
	}
}

func assertNothingUnseeableWasArchived(t *testing.T, stored map[string]pages.Page) {
	t.Helper()

	for _, path := range []string{productPath, productTermPath} {
		page, ok := stored[path]
		if !ok {
			t.Fatalf("the core pull dropped %s from the map", path)
		}
		if page.Status == string(pagemap.StatusArchived) {
			t.Fatalf("the core pull archived %s, which it never asked the site about", path)
		}
	}
	if stored[vanishedPagePath].Status != string(pagemap.StatusArchived) {
		t.Fatalf("%s is %q, want it archived: a page the pull covered and did not find is gone",
			vanishedPagePath, stored[vanishedPagePath].Status)
	}
}

func assertSameMap(t *testing.T, before, after map[string]pages.Page) {
	t.Helper()

	if len(before) != len(after) {
		t.Fatalf("the second sync left %d pages, the first left %d", len(after), len(before))
	}
	for path := range before {
		repeated, ok := after[path]
		if !ok {
			t.Fatalf("the second sync lost %s", path)
		}
		if repeated.ID != before[path].ID || repeated.Status != before[path].Status || repeated.Drift {
			t.Fatalf("the second sync rewrote %s: %+v, want %+v", path, repeated, before[path])
		}
	}
}

func assertSEOWasSkippedNotLost(t *testing.T, core *app.Core, itemID string, final steps.FinalReport) {
	t.Helper()

	if len(final.Publish.SEOApplied) != 0 {
		t.Fatalf("%s applied the SEO meta %v without a plugin to apply it with",
			final.Path, final.Publish.SEOApplied)
	}
	if !slices.Contains(final.Publish.Skipped, steps.CodeSEOMetaSkipped) {
		t.Fatalf("%s reports the skips %v, want %q", final.Path, final.Publish.Skipped, steps.CodeSEOMetaSkipped)
	}

	warning, ok := findingOf(final.Publish.Findings, steps.CodeSEOMetaSkipped)
	if !ok {
		t.Fatalf("%s reports the findings %+v, want a %q warning", final.Path,
			final.Publish.Findings, steps.CodeSEOMetaSkipped)
	}
	if warning.Severity != content.SeverityWarn || warning.Details["reason"] != steps.ReasonNoPlugin {
		t.Fatalf("the skipped finding of %s is %+v", final.Path, warning)
	}

	// the meta is still on record, so the operator can see what the site refused to take
	stored, err := core.Runs.GetArtifact(t.Context(), runs.GetArtifactRequest{
		ItemID: itemID, Kind: string(run.ArtifactMeta),
	})
	if err != nil {
		t.Fatalf("read the meta of %s: %v", final.Path, err)
	}
	var meta steps.Meta
	if err = json.Unmarshal([]byte(stored.Artifact.Content), &meta); err != nil {
		t.Fatalf("decode the meta of %s: %v", final.Path, err)
	}
	if meta.Title == "" || meta.Description == "" {
		t.Fatalf("the meta of %s is %+v, want the generated meta kept", final.Path, meta)
	}
}

func assertRelinkStoodDown(t *testing.T, final steps.FinalReport) {
	t.Helper()

	if final.Relink == nil {
		t.Fatalf("%s carries no relink result", final.Path)
	}
	if final.Relink.Linked != 0 || final.Relink.Conflicts != 0 || final.Relink.Skipped == 0 {
		t.Fatalf("%s relinked %+v, want every neighbor skipped", final.Path, final.Relink)
	}
	for i := range final.Relink.Neighbors {
		neighbor := final.Relink.Neighbors[i]
		if neighbor.Outcome != steps.OutcomeSkipped || neighbor.Detail != steps.ReasonNoPlugin {
			t.Fatalf("the neighbor %s of %s is %+v", neighbor.Path, final.Path, neighbor)
		}
	}
	if _, ok := findingOf(final.Relink.Findings, steps.CodeRelinkSkipped); !ok {
		t.Fatalf("%s reports the findings %+v, want a %q warning", final.Path,
			final.Relink.Findings, steps.CodeRelinkSkipped)
	}
}

func assertDraftLinksUp(t *testing.T, live *site, path string, wpID int) {
	t.Helper()

	body := live.storedContent(t, "pages", wpID)
	for _, want := range []string{parentOf(path), parentOf(parentOf(path))} {
		if !bodyLinksTo(body, want) {
			t.Fatalf("the draft %s does not link to %s: %s", path, want, body)
		}
	}
}

func assertNoSEOMetaOnTheSite(t *testing.T, live *site, wpID int) {
	t.Helper()

	raw := strings.ToLower(string(live.storedItem(t, "pages", wpID)))
	for _, key := range []string{"_yoast_wpseo", "rank_math_", "_postulator_"} {
		if strings.Contains(raw, key) {
			t.Fatalf("the draft %d carries %s, and nothing on this site can write SEO meta", wpID, key)
		}
	}
}

func assertLinksUpInTheMap(t *testing.T, service *pages.Service, byPath map[string]pages.Page, page pages.Page) {
	t.Helper()

	held := linksOf(t, service, page.ID)
	resolved := make(map[string]bool, len(held))
	for i := range held {
		if held[i].ToPageID != nil {
			resolved[held[i].ToURL] = true
		}
	}
	for _, want := range []string{parentOf(page.Path), parentOf(parentOf(page.Path))} {
		if !resolved[want] {
			t.Fatalf("the map holds no resolved link from %s to %s: %+v", page.Path, want, resolved)
		}
		if _, ok := byPath[want]; !ok {
			t.Fatalf("the map has no page at %s", want)
		}
	}
}

func findingOf(findings []content.Finding, code string) (content.Finding, bool) {
	for i := range findings {
		if findings[i].Code == code {
			return findings[i], true
		}
	}
	return content.Finding{}, false
}

func linksOf(t *testing.T, service *pages.Service, pageID string) []pages.PageLink {
	t.Helper()

	held, err := service.Get(t.Context(), pages.GetRequest{ID: pageID})
	if err != nil {
		t.Fatalf("read the page %s: %v", pageID, err)
	}
	return held.Links
}

func bodyLinksTo(body, path string) bool {
	for _, match := range hrefPattern.FindAllStringSubmatch(body, -1) {
		href := match[1]
		if cut := strings.Index(href, "://"); cut >= 0 {
			if slash := strings.Index(href[cut+3:], "/"); slash >= 0 {
				href = href[cut+3+slash:]
			}
		}
		href, _, _ = strings.Cut(href, "?")
		href, _, _ = strings.Cut(href, "#")
		if strings.TrimSuffix(href, "/") == strings.TrimSuffix(path, "/") {
			return true
		}
	}
	return false
}

type coreItem struct {
	ID      int    `json:"id"`
	Slug    string `json:"slug"`
	Status  string `json:"status"`
	Content struct {
		Raw string `json:"raw"`
	} `json:"content"`
}

// coreList reads every page or post the editor can see, the way the adapter does, so the test
// can look at what WordPress really stored without going through the companion plugin.
func (s *site) coreList(t *testing.T, kind string) []coreItem {
	t.Helper()

	out := make([]coreItem, 0, 64)
	for page := 1; page <= 20; page++ {
		var listed []coreItem
		query := "?context=edit&orderby=id&order=asc&per_page=100&page=" + strconv.Itoa(page) +
			"&status=publish,future,draft,pending,private"
		s.call(t, http.MethodGet, "/wp-json/wp/v2/"+kind+query, nil, http.StatusOK, &listed)
		out = append(out, listed...)
		if len(listed) < 100 {
			return out
		}
	}
	t.Fatalf("listing %s never ended", kind)
	return nil
}

func (s *site) storedItem(t *testing.T, kind string, wpID int) json.RawMessage {
	t.Helper()

	var raw json.RawMessage
	s.call(t, http.MethodGet, "/wp-json/wp/v2/"+kind+"/"+strconv.Itoa(wpID)+"?context=edit",
		nil, http.StatusOK, &raw)
	return raw
}

func (s *site) storedContent(t *testing.T, kind string, wpID int) string {
	t.Helper()

	var item coreItem
	if err := json.Unmarshal(s.storedItem(t, kind, wpID), &item); err != nil {
		t.Fatalf("decode the stored %s %d: %v", kind, wpID, err)
	}
	return item.Content.Raw
}

func (s *site) publishPost(t *testing.T, title, slug, body string) int {
	t.Helper()

	var created struct {
		ID int `json:"id"`
	}
	s.call(t, http.MethodPost, "/wp-json/wp/v2/posts", map[string]any{
		"title": title, "slug": slug, "content": body, "status": "publish",
	}, http.StatusCreated, &created)
	if created.ID == 0 {
		t.Fatalf("the site created no post for %s", slug)
	}
	return created.ID
}

// clearCore removes what an earlier degraded run left behind, through core REST, so the test
// can be run again without resetting the docker stack. A draft has no readable permalink, so
// the sweep goes by slug rather than by path.
func (s *site) clearCore(t *testing.T) {
	t.Helper()

	sweep := map[string][]string{
		"pages": {"menu", "main-courses", "drinks", "desserts", "steaks", "seafood", "pasta", "gone"},
		"posts": {"menu-news"},
	}
	for kind, prefixes := range sweep {
		for _, item := range s.coreList(t, kind) {
			if !hasAnyPrefix(item.Slug, prefixes) {
				continue
			}
			s.call(t, http.MethodDelete, "/wp-json/wp/v2/"+kind+"/"+strconv.Itoa(item.ID)+"?force=true",
				nil, http.StatusOK, nil)
		}
	}
}

func hasAnyPrefix(slug string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if slug == prefix || strings.HasPrefix(slug, prefix+"-") {
			return true
		}
	}
	return false
}
