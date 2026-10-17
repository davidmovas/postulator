package observe_test

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/application/observe"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/render"
)

type crumb struct {
	name string
	path string
}

type publicPage struct {
	title    string
	path     string
	crumbs   []crumb
	content  string
	listing  []string
	absolute bool
}

func mustJSON(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(encoded)
}

func (p publicPage) href(path string) string {
	if p.absolute {
		return baseURL + path
	}
	return path
}

func (p publicPage) html() string {
	var b strings.Builder
	b.WriteString("<!doctype html><html><head><title>" + p.title + "</title>")
	if len(p.crumbs) > 0 {
		items := make([]map[string]any, 0, len(p.crumbs))
		for i, c := range p.crumbs {
			items = append(items, map[string]any{"@type": "ListItem", "position": i + 1, "name": c.name, "item": baseURL + c.path})
		}
		b.WriteString(`<script type="application/ld+json">` + mustJSON(map[string]any{"@context": "https://schema.org", "@type": "BreadcrumbList", "itemListElement": items}) + "</script>")
	}
	if len(p.listing) > 0 {
		items := make([]map[string]any, 0, len(p.listing))
		for i, path := range p.listing {
			items = append(items, map[string]any{"@type": "ListItem", "position": i + 1, "name": "Product " + fmt.Sprint(i), "url": baseURL + path})
		}
		b.WriteString(`<script type="application/ld+json">` + mustJSON(map[string]any{"@context": "https://schema.org", "@type": "ItemList", "itemListElement": items}) + "</script>")
	}
	b.WriteString(`</head><body><header><nav aria-label="Primary"><a href="` + p.href("/") + `">Home</a> <a href="` + p.href("/shop/") + `">Shop</a></nav></header>`)
	b.WriteString("<main><article>")
	b.WriteString(p.filtered())
	b.WriteString("</article></main>")
	b.WriteString(`<footer><nav><a href="` + p.href("/account/") + `">Account</a></nav></footer></body></html>`)
	return b.String()
}

func (p publicPage) filtered() string {
	var b strings.Builder
	if len(p.crumbs) > 0 {
		b.WriteString(`<nav aria-label="Breadcrumb"><ol>`)
		for i, c := range p.crumbs {
			if i == len(p.crumbs)-1 {
				b.WriteString(`<li><span aria-current="page">` + c.name + "</span></li>")
				continue
			}
			b.WriteString(`<li><a href="` + p.href(c.path) + `">` + c.name + "</a></li>")
		}
		b.WriteString("</ol></nav>")
	}
	b.WriteString("<h1>" + p.title + "</h1>")
	b.WriteString(p.content)
	if len(p.listing) > 0 {
		b.WriteString("<section><h2>" + p.title + " products</h2>")
		for i, path := range p.listing {
			b.WriteString(`<article><a href="` + p.href(path) + `">Product ` + fmt.Sprint(i) + "</a></article>")
		}
		b.WriteString("</section>")
	}
	return b.String()
}

func (s *scenario) serve(pages ...publicPage) {
	for _, page := range pages {
		s.visitor.pages[baseURL+page.path] = observe.Visit{Status: 200, Body: page.html()}
		for i := range s.plugin.items {
			if s.plugin.items[i].Path == page.path && s.plugin.items[i].Type != "product_cat" {
				s.plugin.rendered[s.plugin.items[i].ID] = observe.Rendered{Content: page.filtered()}
			}
		}
	}
}

func (s *scenario) serveTheSite(absolute bool) {
	s.serve(
		publicPage{title: "Home", path: "/", content: "<p>" + homeRaw + "</p>", absolute: absolute},
		publicPage{title: "Peptides", path: "/peptides/", crumbs: []crumb{{"Home", "/"}, {"Peptides", "/peptides/"}}, listing: []string{"/product/bpc-157-peptide/"}, absolute: absolute},
		publicPage{title: "BPC-157", path: "/bpc-157/", crumbs: []crumb{{"Home", "/"}, {"Peptides", "/peptides/"}, {"Healing Peptides", "/healing-peptides/"}, {"BPC-157", "/bpc-157/"}}, listing: []string{"/product/bpc-157-peptide/"}, absolute: absolute},
		publicPage{title: "TB-500", path: "/tb-500/", crumbs: []crumb{{"Home", "/"}, {"Peptides", "/peptides/"}, {"TB-500", "/tb-500/"}}, content: "<p>TB-500 is a synthetic fragment of thymosin beta-4.</p><p>Compare it with <a href=\"" + publicPage{absolute: absolute}.href("/bpc-157/") + "\">BPC-157</a> before you order.</p>", listing: []string{"/product/bpc-157-peptide/"}, absolute: absolute},
		publicPage{title: "Shop", path: "/shop/", absolute: absolute},
		publicPage{title: "Account", path: "/account/", content: "<form></form>", absolute: absolute},
		publicPage{title: "BPC-157 Peptide", path: "/product/bpc-157-peptide/", crumbs: []crumb{{"Home", "/"}, {"Healing Peptides", "/product-category/peptides/healing-peptides/"}, {"BPC-157 Peptide", "/product/bpc-157-peptide/"}}, content: "<p>BPC-157 is a pentadecapeptide.</p>", absolute: absolute},
		publicPage{title: "Peptides", path: "/product-category/peptides/", crumbs: []crumb{{"Home", "/"}, {"Peptides", "/product-category/peptides/"}}, absolute: absolute},
		publicPage{title: "Healing Peptides", path: "/product-category/peptides/healing-peptides/", crumbs: []crumb{{"Home", "/"}, {"Healing Peptides", "/product-category/peptides/healing-peptides/"}}, absolute: absolute},
	)
}

func TestTheRenderDerivesTheParentFromTheTrail(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)

	state := s.run(t, observe.Options{})
	if state.Counts.Rendered != 9 || state.Counts.Unreachable != 0 {
		t.Fatalf("rendered %d unreachable %d, want every published page rendered", state.Counts.Rendered, state.Counts.Unreachable)
	}
	tb500 := s.pageAt(t, "/tb-500/")
	peptides := s.pageAt(t, "/peptides/")
	if tb500.ParentPageID == nil || *tb500.ParentPageID != peptides.ID || tb500.ParentReason != pagemap.ParentJSONLD {
		t.Errorf("tb-500's parent = %v %s, want peptides by the breadcrumb list", tb500.ParentPageID, tb500.ParentReason)
	}
	record := s.stores.renders[tb500.ID]
	if record.Root != render.RootFound || len(record.Trails) != 2 {
		t.Errorf("tb-500's render = root %s trails %d", record.Root, len(record.Trails))
	}
	rendered := s.stores.links[tb500.ID][pagemap.SourceRender]
	byType := map[pagemap.LinkType]int{}
	for _, link := range rendered {
		byType[link.Type]++
	}
	if byType[pagemap.LinkBreadcrumb] != 2 || byType[pagemap.LinkListing] != 1 || byType[pagemap.LinkNavigation] != 3 || byType[pagemap.LinkContextual] != 0 {
		t.Errorf("tb-500's render links by type = %v", byType)
	}
	product := s.pageAt(t, "/product/bpc-157-peptide/")
	healing := s.pageAt(t, "/product-category/peptides/healing-peptides/")
	if product.ParentPageID == nil || *product.ParentPageID != healing.ID || product.ParentReason != pagemap.ParentJSONLD {
		t.Errorf("the product's parent = %v %s, want the term page by the breadcrumb list", product.ParentPageID, product.ParentReason)
	}
	bpc := s.pageAt(t, "/bpc-157/")
	home := s.pageAt(t, "/")
	if bpc.ParentPageID == nil || *bpc.ParentPageID != home.ID || bpc.ParentReason != pagemap.ParentURL {
		t.Errorf("bpc-157's parent = %v %s, want the front page by url after the unresolved crumb", bpc.ParentPageID, bpc.ParentReason)
	}
	f, ok := s.finding(finding.ParentUnresolved, finding.SubjectCrumb(bpc.ID, "/healing-peptides/"))
	if !ok || f.Status != finding.Open {
		t.Errorf("parent_unresolved for Healing Peptides = %+v %t", f, ok)
	}
	if s.stores.renders[bpc.ID].Root != render.RootDynamicOnly || s.stores.renders[peptides.ID].Root != render.RootEmptyRaw {
		t.Errorf("roots: bpc %s peptides %s", s.stores.renders[bpc.ID].Root, s.stores.renders[peptides.ID].Root)
	}
	summary, err := s.service.Summary(t.Context(), siteID)
	if err != nil || summary.Rendered != 9 || summary.TrailsResolved != 2 || summary.Listings != 3 || summary.LinksByType["breadcrumb"] == 0 {
		t.Errorf("summary = %+v (%v)", summary, err)
	}
}

func TestARenderIsRefetchedForAChangedPageOrOnRescan(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)

	s.run(t, observe.Options{})
	first := len(s.visitor.visits)
	if first != 9 {
		t.Fatalf("the first walk visited %d pages, want 9", first)
	}

	s.clock.Set(start.Add(time.Hour))
	s.run(t, observe.Options{})
	if len(s.visitor.visits) != first {
		t.Errorf("an incremental walk of an unchanged site visited %d pages", len(s.visitor.visits)-first)
	}

	s.touch(tb500ID, start.Add(90*time.Minute))
	s.clock.Set(start.Add(2 * time.Hour))
	s.run(t, observe.Options{})
	if got := s.visitor.visits[first:]; len(got) != 1 || got[0] != baseURL+"/tb-500/" {
		t.Errorf("a changed page alone is visited, got %v", got)
	}

	s.clock.Set(start.Add(3 * time.Hour))
	s.run(t, observe.Options{RenderRescan: true})
	if len(s.visitor.visits) != first+1+9 {
		t.Errorf("a rescan visits every published page, visits = %d", len(s.visitor.visits))
	}
}

func TestAFailedVisitIsAFindingNotAFailure(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)
	s.visitor.failed[baseURL+"/bpc-157/"] = true

	state := s.run(t, observe.Options{})
	if state.Counts.Unreachable != 1 || state.Counts.Rendered != 8 {
		t.Fatalf("unreachable %d rendered %d", state.Counts.Unreachable, state.Counts.Rendered)
	}
	bpc := s.pageAt(t, "/bpc-157/")
	f, ok := s.finding(finding.RenderUnreachable, finding.SubjectPage(bpc.ID))
	if !ok || f.Status != finding.Open {
		t.Fatalf("render_unreachable = %+v %t", f, ok)
	}
	if _, rendered := s.stores.renders[bpc.ID]; rendered {
		t.Error("a failed visit left a render row")
	}

	delete(s.visitor.failed, baseURL+"/bpc-157/")
	s.clock.Set(start.Add(time.Hour))
	s.run(t, observe.Options{RenderRescan: true})
	if f, _ = s.finding(finding.RenderUnreachable, finding.SubjectPage(bpc.ID)); f.Status != finding.Resolved {
		t.Errorf("a successful visit clears the finding, got %s", f.Status)
	}
	if _, rendered := s.stores.renders[bpc.ID]; !rendered {
		t.Error("the page was not rendered once the site answered")
	}
}

func TestAnArchivedPageIsNeitherVisitedNorLeftUnreachable(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)
	bpcURL := baseURL + "/bpc-157/"
	s.visitor.failed[bpcURL] = true

	s.run(t, observe.Options{})
	bpc := s.pageAt(t, "/bpc-157/")
	if f, ok := s.finding(finding.RenderUnreachable, finding.SubjectPage(bpc.ID)); !ok || f.Status != finding.Open {
		t.Fatalf("render_unreachable before the deletion = %+v %t", f, ok)
	}
	visitsBefore := len(s.visitor.visits)

	s.plugin.items = slices.DeleteFunc(s.plugin.items, func(item observe.ContentItem) bool { return item.ID == bpcID })
	delete(s.plugin.raws, bpcID)
	s.clock.Set(start.Add(time.Hour))
	state := s.run(t, observe.Options{Full: true})
	if got := s.pageAt(t, "/bpc-157/"); got.Status != pagemap.StatusArchived {
		t.Fatalf("a page the site no longer holds is %s, want it archived", got.Status)
	}
	if state.Counts.Unreachable != 0 || slices.Contains(s.visitor.visits[visitsBefore:], bpcURL) {
		t.Errorf("the walk that archived the page still visited it: unreachable %d visits %v", state.Counts.Unreachable, s.visitor.visits[visitsBefore:])
	}
	if f, _ := s.finding(finding.RenderUnreachable, finding.SubjectPage(bpc.ID)); f.Status != finding.Resolved {
		t.Errorf("an archived page's render_unreachable is %s, want it resolved", f.Status)
	}

	s.clock.Set(start.Add(2 * time.Hour))
	s.run(t, observe.Options{RenderRescan: true})
	if slices.Contains(s.visitor.visits[visitsBefore:], bpcURL) {
		t.Errorf("a rescan visited an archived page: %v", s.visitor.visits[visitsBefore:])
	}
	if f, _ := s.finding(finding.RenderUnreachable, finding.SubjectPage(bpc.ID)); f.Status != finding.Resolved {
		t.Errorf("render_unreachable came back for an archived page: %s", f.Status)
	}
}

func TestADismissedUnreachableFindingStaysDismissedWhileTheCauseIsTheSame(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)
	s.visitor.failed[baseURL+"/bpc-157/"] = true

	s.run(t, observe.Options{})
	bpc := s.pageAt(t, "/bpc-157/")
	key := findingKey(siteID, finding.RenderUnreachable, finding.SubjectPage(bpc.ID))
	s.stores.mu.Lock()
	dismissed := s.stores.findings[key]
	dismissed.Status = finding.Dismissed
	s.stores.findings[key] = dismissed
	s.stores.mu.Unlock()
	if strings.Contains(string(dismissed.Subject), "?_=") || strings.Contains(string(dismissed.Subject), "error") {
		t.Fatalf("the subject carries the transport's text, which changes on every visit: %s", dismissed.Subject)
	}

	s.clock.Set(start.Add(time.Hour))
	s.run(t, observe.Options{RenderRescan: true})
	if f, _ := s.finding(finding.RenderUnreachable, finding.SubjectPage(bpc.ID)); f.Status != finding.Dismissed {
		t.Errorf("a dismissed finding whose cause did not change is %s", f.Status)
	}
}

func TestAFailedFilteredRenderIsUnreachableNotAWrongRoot(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)

	s.run(t, observe.Options{})
	tb500 := s.pageAt(t, "/tb-500/")
	before := s.stores.renders[tb500.ID]
	if before.Root != render.RootFound {
		t.Fatalf("tb-500's root before the failure = %s", before.Root)
	}

	delete(s.plugin.rendered, tb500ID)
	s.touch(tb500ID, start.Add(30*time.Minute))
	s.clock.Set(start.Add(time.Hour))
	state := s.run(t, observe.Options{})
	if state.Counts.Unreachable != 1 {
		t.Errorf("a plugin that cannot render the content counts %d unreachable, want 1", state.Counts.Unreachable)
	}
	if f, ok := s.finding(finding.RenderUnreachable, finding.SubjectPage(tb500.ID)); !ok || f.Status != finding.Open {
		t.Errorf("render_unreachable = %+v %t, want it open while the filtered content cannot be read", f, ok)
	}
	if after := s.stores.renders[tb500.ID]; !after.ExtractedAt.Equal(before.ExtractedAt) || after.Root != before.Root {
		t.Errorf("the render row moved on a failed read: before %s at %s, after %s at %s", before.Root, before.ExtractedAt, after.Root, after.ExtractedAt)
	}
}

func TestRendersRunAFewAtATime(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	owner := s.stores.sites[siteID]
	owner.Settings.Observe.RenderConcurrency = 2
	s.stores.sites[siteID] = owner
	s.serveTheSite(false)

	state, err := s.service.Run(t.Context(), siteID, s.readers, observe.Options{}, 50)
	if err != nil || !state.Done() {
		t.Fatalf("observe: %v", err)
	}
	if s.visitor.peak > 2 || s.visitor.peak < 1 {
		t.Errorf("renders ran %d at a time, want at most 2", s.visitor.peak)
	}
}

func TestASiteWithMixedURLFormsIsOneFinding(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(true)
	s.serve(publicPage{title: "TB-500", path: "/tb-500/", crumbs: []crumb{{"Home", "/"}, {"Peptides", "/peptides/"}, {"TB-500", "/tb-500/"}}, content: "<p>Relative <a href=\"/bpc-157/\">links</a> here.</p>"})

	s.run(t, observe.Options{})
	f, ok := s.finding(finding.URLFormMixed, finding.SubjectSite)
	if !ok || f.Status != finding.Open {
		t.Fatalf("url_form_mixed = %+v %t", f, ok)
	}
	var forms map[string]int
	if err := json.Unmarshal(f.Subject, &forms); err != nil || forms["absolute"] == 0 || forms["relative"] == 0 {
		t.Errorf("subject = %s", f.Subject)
	}

	s.serveTheSite(true)
	s.clock.Set(start.Add(time.Hour))
	s.run(t, observe.Options{RenderRescan: true})
	if f, _ = s.finding(finding.URLFormMixed, finding.SubjectSite); f.Status != finding.Resolved {
		t.Errorf("one form everywhere clears the finding, got %s", f.Status)
	}
}

func TestADraftIsRenderedThroughThePreviewLink(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)
	draft := contentItem(70, "page", "/draft-page/", "Draft", "<p>Not yet.</p>")
	draft.Status = "draft"
	s.plugin.items = append(s.plugin.items, draft)
	s.plugin.raws[70] = rawOf("<p>Not yet.</p>")
	s.plugin.rendered[70] = observe.Rendered{Content: "<h1>Draft</h1><p>Not yet.</p>"}
	s.plugin.previews[70] = baseURL + "/?p=70&preview_token=abc"
	s.visitor.pages[baseURL+"/?p=70&preview_token=abc"] = observe.Visit{Status: 200, Body: publicPage{title: "Draft", path: "/draft-page/", content: "<p>Not yet.</p>"}.html()}

	s.run(t, observe.Options{})
	page := s.pageAt(t, "/draft-page/")
	if _, rendered := s.stores.renders[page.ID]; !rendered {
		t.Error("a draft with a preview link was not rendered")
	}
	if !contains(s.visitor.visits, baseURL+"/?p=70&preview_token=abc") {
		t.Errorf("the preview link was not visited: %v", s.visitor.visits)
	}
	if contains(s.visitor.visits, baseURL+"/draft-page/") {
		t.Error("a draft's public address was visited")
	}
}
