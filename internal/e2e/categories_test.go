//go:build e2e

package e2e_test

import (
	"html"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/application/sites"
	appsync "github.com/davidmovas/postulator/internal/application/sync"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const (
	clientWorkbook = "client-sheets.xlsx"
	workbookRoot   = "Peptides"
	categoryKind   = "category"

	filedPath   = "/peptides/tb-500/liquid/"
	siblingPath = "/peptides/tb-500/capsules/"
	filedPost   = "postulator-tb-500-dosing"

	clientPicks   = "Postulator Client Picks "
	trashedSuffix = "__trashed"

	peptidesBody = "<h1>Peptides</h1><p>The shop sells research peptides as liquids, powders and capsules.</p>"
	tb500Body    = "<h1>TB-500</h1><p>TB-500 comes as a liquid and as capsules, and the shop keeps both in stock.</p>"
	liquidBody   = "<h1>TB-500 Liquid</h1><p>The shop's own words about TB-500 as a liquid, written before Postulator came.</p>"
)

var workbookSheetNames = []string{"Groups", "Catalog", "Entities", "Variations"}

var workbookWitnesses = map[string]string{
	"Groups": "Storing peptides", "Catalog": "BPC-157 5 mg vial", "Entities": "Peptides in Canada", "Variations": "Nasal Spray",
}

var workbookSlugs = []string{"peptides", "tb-500", "liquid", "capsules"}

var workbookTerms = []string{"Peptides", "TB-500", "Liquid", "Capsules"}

type siteTerm struct {
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Link   string `json:"link"`
	ID     int64  `json:"id"`
	Parent int64  `json:"parent"`
}

type slugged struct {
	Slug string `json:"slug"`
	ID   int64  `json:"id"`
}

type filedRecord struct {
	Link       string  `json:"link"`
	Categories []int64 `json:"categories"`
}

func termRoute(taxonomy wp.Taxonomy) string {
	if taxonomy == wp.TaxonomyProductCategory {
		return "/wp-json/wc/v3/products/categories"
	}
	return "/wp-json/wp/v2/categories"
}

func (s *site) termsLike(t *testing.T, taxonomy wp.Taxonomy, search string) []siteTerm {
	t.Helper()

	var listed []siteTerm
	s.call(t, http.MethodGet, termRoute(taxonomy)+"?per_page=100&search="+url.QueryEscape(search), nil, http.StatusOK, &listed)
	for i := range listed {
		listed[i].Name = html.UnescapeString(listed[i].Name)
	}
	return listed
}

func (s *site) termsNamed(t *testing.T, taxonomy wp.Taxonomy, name string) []siteTerm {
	t.Helper()

	return slices.DeleteFunc(s.termsLike(t, taxonomy, name), func(term siteTerm) bool {
		return !strings.EqualFold(term.Name, name)
	})
}

func (s *site) createTerm(t *testing.T, taxonomy wp.Taxonomy, name string) siteTerm {
	t.Helper()

	var created siteTerm
	s.call(t, http.MethodPost, termRoute(taxonomy), map[string]any{"name": name}, http.StatusCreated, &created)
	if created.ID == 0 {
		t.Fatalf("the site created no %s term named %q", taxonomy, name)
	}
	return created
}

func (s *site) dropTerms(t *testing.T, taxonomy wp.Taxonomy, doomed func(siteTerm) bool, searches ...string) {
	t.Helper()

	ids := make([]int64, 0, 8)
	for _, search := range searches {
		for _, term := range s.termsLike(t, taxonomy, search) {
			if doomed(term) && !slices.Contains(ids, term.ID) {
				ids = append(ids, term.ID)
			}
		}
	}
	for _, id := range ids {
		s.call(t, http.MethodDelete, termRoute(taxonomy)+"/"+strconv.FormatInt(id, 10)+"?force=true", nil, http.StatusOK, nil)
	}
}

func (s *site) dropSlugged(t *testing.T, kind string, slugs ...string) {
	t.Helper()

	wanted := make([]string, 0, 2*len(slugs))
	for _, slug := range slugs {
		wanted = append(wanted, slug, slug+trashedSuffix)
	}
	s.dropListed(t, kind, "publish,future,draft,pending,private,trash", wanted)
}

func (s *site) dropTrashed(t *testing.T, kind string, slugs ...string) {
	t.Helper()

	wanted := make([]string, 0, len(slugs))
	for _, slug := range slugs {
		wanted = append(wanted, slug+trashedSuffix)
	}
	s.dropListed(t, kind, "trash", wanted)
}

func (s *site) dropListed(t *testing.T, kind, statuses string, slugs []string) {
	t.Helper()

	var listed []slugged
	s.call(t, http.MethodGet, "/wp-json/wp/v2/"+kind+"?context=edit&per_page=100&_fields=id,slug&status="+statuses+
		"&slug="+strings.Join(slugs, ","), nil, http.StatusOK, &listed)
	for _, item := range listed {
		s.call(t, http.MethodDelete, "/wp-json/wp/v2/"+kind+"/"+strconv.FormatInt(item.ID, 10)+"?force=true",
			nil, http.StatusOK, nil)
	}
}

func (s *site) sweepWorkbook(t *testing.T) {
	t.Helper()

	s.dropSlugged(t, "pages", workbookSlugs...)
	s.dropSlugged(t, "posts", filedPost)
	s.dropTerms(t, wp.TaxonomyCategory, func(term siteTerm) bool {
		return slices.Contains(workbookTerms, term.Name) || strings.HasPrefix(term.Name, clientPicks)
	}, append(slices.Clone(workbookTerms), strings.TrimSpace(clientPicks))...)
}

func (s *site) publishFiled(t *testing.T, title, slug, body string, parent int, categories ...int64) int {
	t.Helper()

	var created struct {
		ID int `json:"id"`
	}
	s.call(t, http.MethodPost, "/wp-json/wp/v2/pages", map[string]any{
		"title": title, "slug": slug, "content": body, "status": "publish", "parent": parent, "categories": categories,
	}, http.StatusCreated, &created)
	if created.ID == 0 {
		t.Fatalf("the site created no page for %s", slug)
	}
	return created.ID
}

func (s *site) filed(t *testing.T, kind string, id int64) filedRecord {
	t.Helper()

	var held filedRecord
	s.call(t, http.MethodGet, "/wp-json/wp/v2/"+kind+"/"+strconv.FormatInt(id, 10)+"?context=edit&_fields=link,categories",
		nil, http.StatusOK, &held)
	return held
}

func sameSet(got, want []int64) bool {
	left, right := slices.Clone(got), slices.Clone(want)
	slices.Sort(left)
	slices.Sort(right)
	return slices.Equal(left, right)
}

func termIDsOf(write *steps.CategoryWrite) []int64 {
	ids := make([]int64, 0, len(write.Terms))
	for i := range write.Terms {
		ids = append(ids, write.Terms[i].TermID)
	}
	return ids
}

func createdIDsOf(write *steps.CategoryWrite) []int64 {
	ids := make([]int64, 0, len(write.Terms))
	for i := range write.Terms {
		if write.Terms[i].Created {
			ids = append(ids, write.Terms[i].TermID)
		}
	}
	return ids
}

func idsIn(value any) []int64 {
	listed, isList := value.([]any)
	if !isList {
		return nil
	}
	ids := make([]int64, 0, len(listed))
	for _, entry := range listed {
		if number, ok := entry.(float64); ok {
			ids = append(ids, int64(number))
		}
	}
	return ids
}

func entitiesOf(t *testing.T, core *app.Core, siteID string) map[string]graph.Entity {
	t.Helper()

	out := make(map[string]graph.Entity, 32)
	cursor := ""
	for {
		listed, err := core.Graph.ListEntities(t.Context(), graph.ListEntitiesRequest{
			SiteID: siteID, ListRequest: dto.ListRequest{Limit: 100, Cursor: cursor},
		})
		if err != nil {
			t.Fatalf("list the entities: %v", err)
		}
		for i := range listed.Items {
			out[listed.Items[i].ID] = listed.Items[i]
		}
		if listed.Next == "" {
			return out
		}
		cursor = string(listed.Next)
	}
}

func namesOf(chain []dto.Category) []string {
	names := make([]string, 0, len(chain))
	for i := range chain {
		names = append(names, chain[i].Name)
	}
	return names
}

func recordIDsOf(chain []dto.Category) []string {
	ids := make([]string, 0, len(chain))
	for i := range chain {
		ids = append(ids, chain[i].ID)
	}
	return ids
}

func shownTermIDsOf(chain []dto.Category) []int64 {
	ids := make([]int64, 0, len(chain))
	for i := range chain {
		if chain[i].TermID != nil {
			ids = append(ids, *chain[i].TermID)
		}
	}
	return ids
}

func shelfOf(t *testing.T, core *app.Core, siteID string) []pages.CategoryNode {
	t.Helper()

	listed, err := core.Pages.ListCategories(t.Context(), pages.ListCategoriesRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("list the categories: %v", err)
	}
	return listed.Categories
}

func trailsOf(nodes []pages.CategoryNode) map[string]string {
	byID := make(map[string]pages.CategoryNode, len(nodes))
	for _, node := range nodes {
		byID[node.ID] = node
	}
	trails := make(map[string]string, len(nodes))
	for _, node := range nodes {
		trail := node.Name
		for at, walked := node.ParentID, 0; at != nil && walked < len(nodes); walked++ {
			parent := byID[*at]
			trail = parent.Name + " › " + trail
			at = parent.ParentID
		}
		trails[node.ID] = trail
	}
	return trails
}

func assertTheWorkbookShelf(t *testing.T, core *app.Core, siteID string, applied imports.ApplyResponse) {
	t.Helper()

	dropped := make([]string, 0, 6)
	for _, finding := range applied.Report.Warnings {
		if finding.Code == string(imports.CodeCategoryLevelIsRoot) {
			dropped = append(dropped, finding.Sheet)
		}
	}
	if !slices.Contains(dropped, "Catalog") || !slices.Contains(dropped, "Entities") {
		t.Errorf("the workbook reported %s on the sheets %v, want the Catalog and Entities sheets, whose Category column "+
			"names the Root Entity %q", imports.CodeCategoryLevelIsRoot, dropped, workbookRoot)
	}

	shelf := shelfOf(t, core, siteID)
	trails := slices.Sorted(maps.Values(trailsOf(shelf)))
	for _, node := range shelf {
		if strings.EqualFold(node.Name, workbookRoot) {
			t.Errorf("the import made the category %+v among %v, though %q is the Root Entity of the Groups sheet",
				node, trails, workbookRoot)
		}
	}
	for _, want := range []string{"TB-500", "TB-500 › Liquid", "TB-500 › Capsules"} {
		if !slices.Contains(trails, want) {
			t.Fatalf("the site's categories are %v, want %q among them", trails, want)
		}
	}
}

func assertTheEntityShowsItsPageCategories(t *testing.T, core *app.Core, siteID string, page pages.Page) {
	t.Helper()

	if page.EntityID == nil {
		t.Errorf("the import left %s without an entity", page.Path)
		return
	}
	shown := entitiesOf(t, core, siteID)[*page.EntityID].Categories
	if !slices.Equal(recordIDsOf(shown), recordIDsOf(page.Categories)) {
		t.Errorf("the entity of %s shows the categories %v, want its page's %v", page.Path, namesOf(shown), namesOf(page.Categories))
	}
}

func importTheClientWorkbook(t *testing.T, core *app.Core, siteID string) imports.ApplyResponse {
	t.Helper()

	path := sample(t, clientWorkbook)
	seen, err := core.Imports.Inspect(t.Context(), imports.InspectRequest{SiteID: siteID, Path: path})
	if err != nil {
		t.Fatalf("inspect %s: %v", clientWorkbook, err)
	}
	names := make([]string, 0, len(seen.Sheets))
	sheets := make([]imports.SheetMapping, 0, len(seen.Sheets))
	for i := range seen.Sheets {
		names = append(names, seen.Sheets[i].Name)
		sheets = append(sheets, imports.SheetMapping{Sheet: seen.Sheets[i].Name, Mapping: seen.Sheets[i].Detected})
	}
	if !slices.Equal(names, workbookSheetNames) {
		t.Fatalf("%s holds the sheets %v, want %v", clientWorkbook, names, workbookSheetNames)
	}

	applied, err := core.Imports.Apply(t.Context(), imports.ApplyRequest{SiteID: siteID, Path: path, Sheets: sheets})
	if err != nil {
		t.Fatalf("apply %s as one workbook: %v", clientWorkbook, err)
	}
	if len(applied.Report.Errors) != 0 {
		t.Fatalf("the workbook reported %+v, want a clean import", applied.Report.Errors)
	}

	byID := entitiesOf(t, core, siteID)
	named := make(map[string]bool, len(byID))
	for id := range byID {
		named[byID[id].Name] = true
	}
	for sheet, witness := range workbookWitnesses {
		if !named[witness] {
			t.Fatalf("the workbook import wrote %+v and no entity %q of the %s sheet", applied.Counts, witness, sheet)
		}
	}
	assertTheWorkbookShelf(t, core, siteID, applied)
	return applied
}

func openFilingSite(t *testing.T, core *app.Core, live *site, name string) (string, appsync.CheckPluginResponse) {
	t.Helper()

	owner, err := core.Sites.Create(t.Context(), sites.CreateRequest{
		Name: name, BaseURL: live.env.baseURL, Username: live.env.user, Password: live.env.pass, AllowInsecure: true,
	})
	if err != nil {
		t.Fatalf("create the site: %v", err)
	}
	checked, err := core.Sync.CheckPlugin(t.Context(), appsync.CheckPluginRequest{SiteID: owner.Site.ID})
	if err != nil {
		t.Fatalf("check the plugin: %v", err)
	}
	runSync(t, core, owner.Site.ID)
	return owner.Site.ID, checked
}

func publishAlone(t *testing.T, core *app.Core, siteID string, page pages.Page) (string, steps.FinalReport) {
	t.Helper()

	started, err := core.Runs.Start(t.Context(), runs.StartRequest{
		SiteID: siteID, PageIDs: []string{page.ID}, TemplateID: templateOfKind(t, core, categoryKind),
		PublishMode: string(run.PublishLive),
	})
	if err != nil {
		t.Fatalf("start the run over %s: %v", page.Path, err)
	}
	if len(started.Added) != 0 {
		t.Fatalf("the run over %s added %+v, though its parent is on the site", page.Path, started.Added)
	}
	awaitRun(t, core.Runs, started.RunID)
	assertEveryItemPublished(t, core, started.RunID, 1)

	items, err := core.Runs.ListItems(t.Context(), runs.ListItemsRequest{RunID: started.RunID, ListRequest: dto.ListRequest{Limit: 5}})
	if err != nil {
		t.Fatalf("list the items of the run over %s: %v", page.Path, err)
	}
	final := finalReport(t, core.Runs, items.Items[0].ID)
	if final.Publish.Categories == nil {
		t.Fatalf("%s went up without a category write: %+v", page.Path, final.Publish.Findings)
	}
	return started.RunID, final
}

func assertTheChainIsOnTheSite(t *testing.T, live *site, taxonomy wp.Taxonomy, path string, written *steps.CategoryWrite,
	chain []dto.Category) {
	t.Helper()

	if string(written.Taxonomy) != string(taxonomy) || !written.Taken {
		t.Errorf("%s wrote %s categories, taken %t; want %s categories the site kept", path, written.Taxonomy,
			written.Taken, taxonomy)
	}
	names := make([]string, 0, len(written.Terms))
	records := make([]string, 0, len(written.Terms))
	for i := range written.Terms {
		names = append(names, written.Terms[i].Name)
		records = append(records, written.Terms[i].CategoryID)
	}
	if !slices.Equal(names, namesOf(chain)) || !slices.Equal(records, recordIDsOf(chain)) {
		t.Fatalf("%s was filed under %v (records %v), want the chain %v (records %v) the page is filed under", path, names,
			records, namesOf(chain), recordIDsOf(chain))
	}

	parent := int64(0)
	for _, term := range written.Terms {
		if term.ParentID != parent {
			t.Errorf("%s files %q under %d, want %d, the level above it", path, term.Name, term.ParentID, parent)
		}
		onSite := live.termsNamed(t, taxonomy, term.Name)
		if len(onSite) != 1 || onSite[0].ID != term.TermID || onSite[0].Parent != parent {
			t.Errorf("the site holds %+v named %q, want the one term %d under %d", onSite, term.Name, term.TermID, parent)
		}
		parent = term.TermID
	}
}

func assertThePageShowsItsTerms(t *testing.T, core *app.Core, siteID, path string, written *steps.CategoryWrite) {
	t.Helper()

	shown := pagesByPath(t, core.Pages, siteID)[path].Categories
	if !slices.Equal(shownTermIDsOf(shown), termIDsOf(written)) || len(shown) != len(written.Terms) {
		t.Errorf("%s shows the categories %+v, want each with the term %v the run filed it under", path, shown,
			termIDsOf(written))
	}
}

func assertNoCategoryIsNamedAfterTheRoot(t *testing.T, core *app.Core, live *site, siteID string) {
	t.Helper()

	made := live.termsNamed(t, wp.TaxonomyCategory, workbookRoot)
	if len(made) == 0 {
		return
	}
	t.Errorf("the site carries the categories %+v named after %q, the Root Entity of the Groups sheet; only the Category "+
		"and Subcategory columns make WordPress categories, and the import filed its pages under %v",
		made, workbookRoot, slices.Sorted(maps.Values(trailsOf(shelfOf(t, core, siteID)))))
}

func assertTheArchiveListsThePageAndAPost(t *testing.T, core *app.Core, live *site, siteID string, term steps.AssignedTerm,
	pageID int64, tag string) {
	t.Helper()

	post, err := wordpress(t, core, siteID).CreateItem(t.Context(), wp.TypePost, wp.CreateItem{
		Title: "Postulator archive post " + tag, Slug: "postulator-archive-" + tag, Status: "publish",
		Content: "<p>A post the shop filed beside the page.</p>", Categories: []int64{term.TermID},
	})
	if err != nil {
		t.Fatalf("file a post under %q: %v", term.Name, err)
	}
	live.removeLater(t, "/wp-json/wp/v2/posts/"+strconv.FormatInt(post.ID, 10)+"?force=true")

	var archive siteTerm
	live.call(t, http.MethodGet, termRoute(wp.TaxonomyCategory)+"/"+strconv.FormatInt(term.TermID, 10), nil, http.StatusOK, &archive)
	page := live.filed(t, "pages", pageID)

	status, body := live.anonymousGet(t, archive.Link)
	if status != http.StatusOK {
		t.Fatalf("the archive of %q at %s answered %d", term.Name, archive.Link, status)
	}
	for what, link := range map[string]string{"the page": page.Link, "the post": post.Link} {
		if !strings.Contains(body, `href="`+link+`"`) {
			t.Errorf("the archive of %q at %s does not list %s at %s", term.Name, archive.Link, what, link)
		}
	}
}

func revertPutsTheClientsCategoriesBack(t *testing.T, core *app.Core, live *site, runID string, wpID int64,
	before string, picks int64, written *steps.CategoryWrite) {
	t.Helper()

	reverted, err := core.Runs.Revert(t.Context(), runs.RevertRequest{RunID: runID})
	if err != nil {
		t.Fatalf("revert the run that filed %s: %v", filedPath, err)
	}
	awaitRun(t, core.Runs, reverted.RunID)

	items, err := core.Runs.ListItems(t.Context(), runs.ListItemsRequest{RunID: reverted.RunID, ListRequest: dto.ListRequest{Limit: 5}})
	if err != nil || len(items.Items) != 1 {
		t.Fatalf("the revert carries %+v (%v), want the one item of %s", items.Items, err, filedPath)
	}
	result := revertResultOf(t, core, items.Items[0].ID)
	if result.Outcome != steps.OutcomeRestored || result.Created {
		t.Fatalf("the revert of %s answered %+v, want the page the run updated restored", filedPath, result)
	}

	kept, found := findingOf(result.Findings, steps.CodeRevertTermsKept)
	if !found {
		t.Errorf("the revert of %s carries %+v, want a %s finding", filedPath, result.Findings, steps.CodeRevertTermsKept)
	} else if ids := idsIn(kept.Details["termIds"]); !sameSet(ids, createdIDsOf(written)) {
		t.Errorf("the revert says it kept the categories %v, want the %v the run created", ids, createdIDsOf(written))
	}

	if carried := live.filed(t, "pages", wpID).Categories; !sameSet(carried, []int64{picks}) {
		t.Errorf("after the revert %s carries %v, want only the client's %d", filedPath, carried, picks)
	}
	if body := live.storedContent(t, "pages", int(wpID)); body != before {
		t.Errorf("the revert left the body of %s as %q, want %q", filedPath, body, before)
	}
	for i := range written.Terms {
		if onSite := live.termsNamed(t, wp.TaxonomyCategory, written.Terms[i].Name); len(onSite) != 1 {
			t.Errorf("after the revert the site holds %+v named %q, want the category kept", onSite, written.Terms[i].Name)
		}
	}
}

func TestTheClientWorkbookFilesItsPagesUnderTheirCategories(t *testing.T) {
	live := newSite(t)
	requirePlugin(t, live.env, true)
	live.sweepWorkbook(t)
	defer live.sweepWorkbook(t)

	tag := strconv.FormatInt(time.Now().UnixNano()%1_000_000_000, 10)
	picks := live.createTerm(t, wp.TaxonomyCategory, clientPicks+tag)
	peptidesID := live.publish(t, "Research peptides", "peptides", peptidesBody, 0)
	tb500ID := live.publish(t, "TB-500 peptide", "tb-500", tb500Body, peptidesID)
	liquidID := int64(live.publishFiled(t, "TB-500 liquid", "liquid", liquidBody, tb500ID, picks.ID))
	before := live.storedContent(t, "pages", int(liquidID))

	core := openCoreWith(t, fake.NewScripted((&clientScript{}).replies()...))
	siteID, checked := openFilingSite(t, core, live, "Docker Peptides")
	if !slices.Contains(checked.Plugin.Capabilities, wp.CapabilityPageCategories) {
		t.Fatalf("the plugin check reports %+v, want the %s capability of plugin 1.3.0", checked.Plugin, wp.CapabilityPageCategories)
	}
	importTheClientWorkbook(t, core, siteID)

	stored := pagesByPath(t, core.Pages, siteID)
	filed := stored[filedPath]
	if filed.WPID == nil || *filed.WPID != liquidID {
		t.Fatalf("the import left %s as %+v, want the synced page %d", filedPath, filed, liquidID)
	}
	chain := filed.Categories
	if !slices.Equal(namesOf(chain), []string{"TB-500", "Liquid"}) || filed.CategoriesNeedPlugin {
		t.Fatalf("%s is filed under %v (needs the plugin %t), want the Category and Subcategory the Groups sheet names, "+
			"which the plugin lets a page carry", filedPath, namesOf(chain), filed.CategoriesNeedPlugin)
	}
	if held := shownTermIDsOf(chain); len(held) != 0 {
		t.Fatalf("%s shows the terms %v before any run filed it", filedPath, held)
	}
	assertTheEntityShowsItsPageCategories(t, core, siteID, filed)

	firstRun, first := publishAlone(t, core, siteID, filed)
	written := first.Publish.Categories
	if first.Publish.Created {
		t.Fatalf("the run created %s, which was on the site already", filedPath)
	}
	assertTheChainIsOnTheSite(t, live, wp.TaxonomyCategory, filedPath, written, chain)
	assertThePageShowsItsTerms(t, core, siteID, filedPath, written)
	if created := createdIDsOf(written); !sameSet(created, termIDsOf(written)) {
		t.Errorf("the first publish says it created %v of %v, want every level of the chain", created, termIDsOf(written))
	}
	if !sameSet(written.Previous, []int64{picks.ID}) || !sameSet(written.Added, termIDsOf(written)) {
		t.Errorf("the first publish read %v and added %v, want the client's %d and then the chain %v",
			written.Previous, written.Added, picks.ID, termIDsOf(written))
	}
	if carried := live.filed(t, "pages", liquidID).Categories; !sameSet(carried, append([]int64{picks.ID}, termIDsOf(written)...)) {
		t.Errorf("%s carries %v, want the client's %d beside the chain %v", filedPath, carried, picks.ID, termIDsOf(written))
	}
	assertNoCategoryIsNamedAfterTheRoot(t, core, live, siteID)
	assertTheArchiveListsThePageAndAPost(t, core, live, siteID, written.Terms[len(written.Terms)-1], liquidID, tag)

	revertPutsTheClientsCategoriesBack(t, core, live, firstRun, liquidID, before, picks.ID, written)

	sibling := pagesByPath(t, core.Pages, siteID)[siblingPath]
	if sibling.ID == "" || sibling.WPID != nil {
		t.Fatalf("the import left %s as %+v, want a planned page", siblingPath, sibling)
	}
	siblingChain := sibling.Categories
	if !slices.Equal(namesOf(siblingChain), []string{"TB-500", "Capsules"}) ||
		!slices.Equal(shownTermIDsOf(siblingChain), termIDsOf(written)[:1]) {
		t.Fatalf("%s is filed under %+v, want TB-500, already on the site as %d, and Capsules, not yet", siblingPath,
			siblingChain, written.Terms[0].TermID)
	}
	_, second := publishAlone(t, core, siteID, sibling)
	again := second.Publish.Categories
	assertTheChainIsOnTheSite(t, live, wp.TaxonomyCategory, siblingPath, again, siblingChain)
	assertThePageShowsItsTerms(t, core, siteID, siblingPath, again)

	reused := make(map[string]int64, len(written.Terms))
	for i := range written.Terms {
		reused[written.Terms[i].Name] = written.Terms[i].TermID
	}
	for i, term := range again.Terms {
		shared, known := reused[term.Name]
		switch {
		case i < len(again.Terms)-1 && (!known || term.TermID != shared || term.Created):
			t.Errorf("the sibling files %q as %+v, want the term %d reused and not created", term.Name, term, shared)
		case i == len(again.Terms)-1 && (known || !term.Created):
			t.Errorf("the sibling files its own level %q as %+v, want a term created for it", term.Name, term)
		}
	}
	if carried := live.filed(t, "pages", second.Publish.WPID).Categories; !sameSet(carried, termIDsOf(again)) {
		t.Errorf("%s carries %v, want the chain %v", siblingPath, carried, termIDsOf(again))
	}
	t.Logf("%s was filed under %v as %v beside the client's %d and put back; %s reused %v and created %v",
		filedPath, namesOf(chain), termIDsOf(written), picks.ID, siblingPath, termIDsOf(written)[:1], createdIDsOf(again))
}
