package observe_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/observe"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/clock"
)

const (
	siteID  = "site-1"
	baseURL = "http://localhost:8087"

	homeID, peptidesID, bpcID, tb500ID, shopID, accountID int64 = 1, 2, 3, 4, 5, 6
	productID, variationAID, variationBID                 int64 = 10, 11, 12
	pepcatTermID, healingTermID, entityTermID             int64 = 100000, 100001, 100010
)

var (
	start   = time.Date(2026, time.October, 17, 12, 0, 0, 0, time.UTC)
	earlier = start.Add(-24 * time.Hour)

	tb500Raw   = "TB-500 is a synthetic fragment of thymosin beta-4.\n\nCompare it with <a href=\"/bpc-157/\">BPC-157</a> before you order."
	homeRaw    = "Research peptides for the laboratory."
	bpcRaw     = `[products category="healing-peptides" limit="4"]`
	accountRaw = "[woocommerce_my_account]"
)

type scenario struct {
	stores  *stores
	clock   *clock.Fake
	plugin  *pluginFake
	core    *coreFake
	store   *storeFake
	visitor *visitorFake
	readers observe.Readers
	service *observe.Service
}

func newScenario(t *testing.T) *scenario {
	t.Helper()

	owner := site.Site{
		ID: siteID, Name: "Peptide Store", BaseURL: baseURL, SecretRef: site.SecretRef(siteID), Status: site.StatusActive,
		AllowInsecure: true, Plugin: site.PluginState{Installed: true, Version: "1.4.1"}, Commerce: site.CommerceReady,
		Settings: site.DefaultSettings(), CreatedAt: earlier, UpdatedAt: earlier,
	}
	s := &scenario{stores: newStores(owner), clock: clock.NewFake(start)}
	s.plugin = &pluginFake{
		items: []observe.ContentItem{
			contentItem(homeID, "page", "/", "Home", homeRaw),
			contentItem(peptidesID, "page", "/peptides/", "Peptides", ""),
			contentItem(bpcID, "page", "/bpc-157/", "BPC-157", bpcRaw),
			withMeta(withLinks(contentItem(tb500ID, "page", "/tb-500/", "TB-500", tb500Raw), observe.ContentLink{Href: "/bpc-157/", Anchor: "BPC-157"}),
				observe.SEOMeta{Title: "TB-500 | Peptide Store", Description: "Buy TB-500."}),
			contentItem(shopID, "page", "/shop/", "Shop", ""),
			contentItem(accountID, "page", "/account/", "Account", accountRaw),
			withExcerpt(contentItem(productID, "product", "/product/bpc-157-peptide/", "BPC-157 Peptide", "<p>BPC-157 is a pentadecapeptide.</p>"), "<p>Lyophilized BPC-157.</p>"),
			termItem(pepcatTermID, "/product-category/peptides/", "Peptides", 0),
			termItem(healingTermID, "/product-category/peptides/healing-peptides/", "Healing Peptides", pepcatTermID),
		},
		raws: map[int64]observe.Raw{
			homeID: rawOf(homeRaw), peptidesID: rawOf(""), bpcID: rawOf(bpcRaw), tb500ID: rawOf(tb500Raw),
			shopID: rawOf(""), accountID: rawOf(accountRaw), productID: rawOf("<p>BPC-157 is a pentadecapeptide.</p>"),
		},
		rendered: map[int64]observe.Rendered{},
		meta:     map[int64]observe.SEOMeta{tb500ID: {Title: "TB-500 | Peptide Store", Description: "Buy TB-500."}},
		terms: map[int64][]pagemap.Term{
			productID: {
				{Taxonomy: "product_cat", TermID: healingTermID, Name: "Healing Peptides", Slug: "healing-peptides", ParentTermID: pepcatTermID},
				{Taxonomy: "entity", TermID: entityTermID, Name: "BPC-157", Slug: "bpc-157"},
			},
		},
		pages:    site.TechnicalPages{Front: homeID, Shop: shopID},
		previews: map[int64]string{},
	}
	s.core = &coreFake{
		types: []observe.PostType{
			{Slug: "page", Viewable: true, Hierarchical: true}, {Slug: "post", Viewable: true},
			{Slug: "product", Viewable: true}, {Slug: "attachment", Viewable: true}, {Slug: "product_variation"},
			{Slug: "wp_block"},
		},
		items:    map[string][]observe.Item{},
		settings: observe.SiteSettings{ShowOnFront: "page", FrontPageID: homeID},
	}
	s.store = &storeFake{
		products: []observe.Product{{
			ID: productID, Permalink: baseURL + "/product/bpc-157-peptide/", Slug: "bpc-157-peptide", Status: "publish",
			Name: "BPC-157 Peptide", Type: "variable", Description: "<p>BPC-157 is a pentadecapeptide.</p>",
			ShortDescription: "<p>Lyophilized BPC-157.</p>", Price: "39", RegularPrice: "39", SKU: "BPC157",
			StockStatus: "instock", ManageStock: "false", Attributes: json.RawMessage(`[{"name":"Strength","options":["5 mg","10 mg"]}]`),
			Categories:   []pagemap.Term{{Taxonomy: "product_cat", TermID: healingTermID, Name: "Healing Peptides", Slug: "healing-peptides"}},
			VariationIDs: []int64{variationAID, variationBID}, Modified: earlier,
		}},
		variations: map[int64][]observe.Variation{productID: {
			{ID: variationAID, ParentID: productID, Description: "Five milligrams.", SKU: "BPC157-5MG", Price: "39", StockStatus: "instock", ManageStock: "parent", Attributes: json.RawMessage(`[{"name":"Strength","option":"5 mg"}]`), Modified: earlier},
			{ID: variationBID, ParentID: productID, SKU: "BPC157-10MG", Price: "69", StockStatus: "instock", ManageStock: "parent", Attributes: json.RawMessage(`[{"name":"Strength","option":"10 mg"}]`), Modified: earlier},
		}},
		pages: site.TechnicalPages{Shop: shopID},
	}
	s.visitor = &visitorFake{pages: map[string]observe.Visit{}, failed: map[string]bool{}, redirects: map[string]string{}}
	s.plugin.guard, s.core.guard, s.store.guard, s.visitor.guard = s.stores.guard, s.stores.guard, s.stores.guard, s.stores.guard
	s.readers = observe.Readers{
		Core: s.core, Store: s.store, Plugin: s.plugin, Render: s.visitor,
		Manifest: observe.Manifest{
			Version: "1.4.1", Required: "1.4.1",
			Capabilities: []string{"bulk", "raw", "rendered", "terms", "settings", "seo_meta_read", "preview", "content_hash", "seo_meta", "meta"},
			PostTypes:    []string{"page", "post", "product"}, Taxonomies: []string{"product_cat", "entity"},
		},
	}
	s.service = observe.New(s.stores.deps(s.clock))
	return s
}

func contentItem(id int64, wpType, path, title, raw string) observe.ContentItem {
	return observe.ContentItem{
		ID: id, Type: wpType, Path: path, Slug: pagemap.Slug(path), Status: "publish", Title: title,
		ContentHash: block.BodyHash(raw), Modified: earlier,
	}
}

func termItem(id int64, path, title string, parent int64) observe.ContentItem {
	item := contentItem(id, "product_cat", path, title, "")
	item.Parent = parent
	item.ContentHash = ""
	return item
}

func withLinks(item observe.ContentItem, links ...observe.ContentLink) observe.ContentItem {
	item.Links = links
	return item
}

func withMeta(item observe.ContentItem, meta observe.SEOMeta) observe.ContentItem {
	item.Meta = meta
	return item
}

func withExcerpt(item observe.ContentItem, excerpt string) observe.ContentItem {
	item.Excerpt = excerpt
	return item
}

func rawOf(content string) observe.Raw {
	return observe.Raw{Content: content, Hash: block.BodyHash(content)}
}

func (s *scenario) run(t *testing.T, opts observe.Options) observe.State {
	t.Helper()
	state, err := s.service.Run(context.Background(), siteID, s.readers, opts, 3)
	if err != nil {
		t.Fatalf("observe: %v", err)
	}
	if state.Stage != observe.StageDone {
		t.Fatalf("the observation stopped at %s", state.Stage)
	}
	return state
}

func (s *scenario) pageAt(t *testing.T, path string) pagemap.Page {
	t.Helper()
	s.stores.mu.Lock()
	defer s.stores.mu.Unlock()
	for id := range s.stores.pages {
		if s.stores.pages[id].Path == path {
			return s.stores.pages[id]
		}
	}
	t.Fatalf("no page at %s", path)
	return pagemap.Page{}
}

func (s *scenario) fieldOf(t *testing.T, pageID string, key field.Key) field.Value {
	t.Helper()
	s.stores.mu.Lock()
	defer s.stores.mu.Unlock()
	value, ok := s.stores.fields[pageID][key]
	if !ok {
		t.Fatalf("page %s has no field %s", pageID, key)
	}
	return value
}

func (s *scenario) openFindings(code finding.Code) []finding.Finding {
	s.stores.mu.Lock()
	defer s.stores.mu.Unlock()
	out := make([]finding.Finding, 0)
	for key := range s.stores.findings {
		if f := s.stores.findings[key]; f.Code == code && f.Status == finding.Open {
			out = append(out, f)
		}
	}
	return out
}

func (s *scenario) finding(code finding.Code, subject string) (finding.Finding, bool) {
	s.stores.mu.Lock()
	defer s.stores.mu.Unlock()
	f, ok := s.stores.findings[findingKey(siteID, code, subject)]
	return f, ok
}

func (s *scenario) seedPage(t *testing.T, page pagemap.Page) pagemap.Page {
	t.Helper()
	page.SiteID = siteID
	page.CreatedAt, page.UpdatedAt = earlier, earlier
	built, err := pagemap.NewPage(page)
	if err != nil {
		t.Fatalf("seed page: %v", err)
	}
	s.stores.mu.Lock()
	s.stores.pages[built.ID] = built
	s.stores.mu.Unlock()
	return built
}

func (s *scenario) touch(id int64, at time.Time) {
	for i := range s.plugin.items {
		if s.plugin.items[i].ID == id {
			s.plugin.items[i].Modified = at
		}
	}
}

func TestTheFirstObservationWalksTheWholeSite(t *testing.T) {
	t.Parallel()
	s := newScenario(t)

	state := s.run(t, observe.Options{})
	if !state.Full || state.Since != nil {
		t.Errorf("the first observation is a full walk, got full=%t since=%v", state.Full, state.Since)
	}
	if len(s.plugin.queries) == 0 {
		t.Fatal("the plugin was never asked for content")
	}
	wantTypes := []string{"page", "post", "product", "product_cat"}
	if got := s.plugin.queries[0].Types; !slices.Equal(got, wantTypes) {
		t.Errorf("the content query names %v, want %v", got, wantTypes)
	}
	if len(s.stores.pages) != 9 || len(s.stores.identities) != 9 {
		t.Fatalf("pages = %d identities = %d, want 9 and 9", len(s.stores.pages), len(s.stores.identities))
	}

	tb500 := s.pageAt(t, "/tb-500/")
	if tb500.WPID == nil || *tb500.WPID != tb500ID || tb500.Status != pagemap.StatusPublished || tb500.Title != "TB-500" {
		t.Errorf("tb-500 row = %+v", tb500)
	}
	if identity := s.stores.identities[tb500.ID]; identity.URL != baseURL+"/tb-500/" || identity.Status != "publish" || !identity.ObservedAt.Equal(start) {
		t.Errorf("tb-500 identity = %+v", identity)
	}
	body := s.stores.bodies[tb500.ID]
	if body[block.Current].Raw != tb500Raw || body[block.Base].Raw != tb500Raw || body[block.Current].Kind != block.Classic {
		t.Errorf("tb-500 body = %+v", body)
	}
	if got := s.fieldOf(t, tb500.ID, field.Title); got.Current != "TB-500" || got.Base != "TB-500" {
		t.Errorf("title field = %+v", got)
	}
	if got := s.fieldOf(t, tb500.ID, field.SEOTitle); got.Current != "TB-500 | Peptide Store" {
		t.Errorf("seo title field = %+v", got)
	}
	links := s.stores.links[tb500.ID][pagemap.SourceRaw]
	bpc := s.pageAt(t, "/bpc-157/")
	if len(links) != 1 || links[0].Type != pagemap.LinkContextual || links[0].ToPageID == nil || *links[0].ToPageID != bpc.ID || links[0].Position == nil || *links[0].Position != 1 {
		t.Errorf("tb-500 raw links = %+v", links)
	}

	product := s.pageAt(t, "/product/bpc-157-peptide/")
	terms := s.stores.terms[product.ID]
	if len(terms) != 2 || terms[0].Taxonomy != "product_cat" || terms[1].Taxonomy != "entity" {
		t.Errorf("the product's terms = %+v", terms)
	}
	if s.pageAt(t, "/").TechnicalReason != pagemap.TechnicalNone || s.pageAt(t, "/shop/").TechnicalReason != pagemap.TechnicalShop {
		t.Error("the front page is technical or the shop page is not")
	}
	healing := s.pageAt(t, "/product-category/peptides/healing-peptides/")
	pepcat := s.pageAt(t, "/product-category/peptides/")
	if healing.ParentPageID == nil || *healing.ParentPageID != pepcat.ID || healing.ParentReason != pagemap.ParentWP {
		t.Errorf("the term's parent = %v %s, want its parent term by wp_parent", healing.ParentPageID, healing.ParentReason)
	}
	if healing.ReadOnlyReason != pagemap.ReadOnlyTerm {
		t.Errorf("a term page is read-only, got %q", healing.ReadOnlyReason)
	}
	home := s.pageAt(t, "/")
	if tb500.ParentPageID == nil || *tb500.ParentPageID != home.ID || tb500.ParentReason != pagemap.ParentURL {
		t.Errorf("tb-500's parent = %v %s, want the front page by url", tb500.ParentPageID, tb500.ParentReason)
	}

	owner := s.stores.sites[siteID]
	if owner.Settings.Observation.LastStartedAt == nil || !owner.Settings.Observation.LastStartedAt.Equal(start) ||
		owner.Settings.Observation.SinceFullWalk != 0 || owner.Settings.Technical.Front != homeID {
		t.Errorf("the site's observation state = %+v", owner.Settings)
	}
	if !slices.Contains(s.stores.published, events.PagesChanged) {
		t.Error("the pages changed event was not published")
	}
	if len(s.openFindings(finding.PluginOutdated)) != 0 || len(s.openFindings(finding.BodyChangedOnSite)) != 0 {
		t.Error("a first observation of a current plugin raises nothing")
	}
}

func TestASecondObservationIsIncrementalAndChangesNothing(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)
	s.run(t, observe.Options{})
	before := snapshot(t, s.stores)

	s.clock.Set(start.Add(time.Hour))
	state := s.run(t, observe.Options{})
	if state.Full || state.Since == nil || !state.Since.Equal(earlier.Add(-time.Second)) {
		t.Errorf("the second observation is incremental since a second before the newest stamp the first walk saw, got full=%t since=%v", state.Full, state.Since)
	}
	last := s.plugin.queries[len(s.plugin.queries)-1]
	if last.Since == nil || !last.Since.Equal(earlier.Add(-time.Second)) {
		t.Errorf("the plugin was asked since %v", last.Since)
	}
	if len(s.store.queries) == 0 || s.store.queries[len(s.store.queries)-1].ModifiedAfter == nil {
		t.Error("the store was not asked for products modified since the last walk")
	}
	if after := snapshot(t, s.stores); !reflect.DeepEqual(before, after) {
		t.Errorf("an incremental observation of an unchanged site changed the model: %s", difference(before, after))
	}
	if owner := s.stores.sites[siteID]; owner.Settings.Observation.SinceFullWalk != 1 {
		t.Errorf("walks since the full one = %d, want 1", owner.Settings.Observation.SinceFullWalk)
	}
}

type modelSnapshot struct {
	pages    map[string]pagemap.Page
	fields   map[string]map[field.Key]field.Value
	bodies   map[string]map[block.Side]block.Snapshot
	links    map[string]map[pagemap.LinkSource][]pagemap.PageLink
	findings map[string]finding.Finding
}

type modelCopy struct {
	P map[string]pagemap.Page
	F map[string]map[field.Key]field.Value
	B map[string]map[block.Side]block.Snapshot
	L map[string]map[pagemap.LinkSource][]pagemap.PageLink
	N map[string]finding.Finding
}

func snapshot(t *testing.T, s *stores) modelSnapshot {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, err := json.Marshal(modelCopy{s.pages, s.fields, s.bodies, s.links, s.findings})
	if err != nil {
		t.Fatalf("snapshot the model: %v", err)
	}
	var copied modelCopy
	if err := json.Unmarshal(raw, &copied); err != nil {
		t.Fatalf("read the snapshot back: %v", err)
	}
	return modelSnapshot{pages: copied.P, fields: copied.F, bodies: copied.B, links: copied.L, findings: copied.N}
}

func difference(before, after modelSnapshot) string {
	report := func(name string, a, b any) string {
		left, leftErr := json.Marshal(a)
		right, rightErr := json.Marshal(b)
		if leftErr != nil || rightErr != nil {
			return name + ": " + fmt.Sprint(leftErr, rightErr)
		}
		if bytes.Equal(left, right) {
			return ""
		}
		return name + ":\n  before " + string(left) + "\n  after  " + string(right)
	}
	for id := range before.pages {
		if diff := report("page "+id, before.pages[id], after.pages[id]); diff != "" {
			return diff
		}
	}
	for id := range before.fields {
		if diff := report("fields of "+id, before.fields[id], after.fields[id]); diff != "" {
			return diff
		}
	}
	for id := range before.bodies {
		if diff := report("bodies of "+id, before.bodies[id], after.bodies[id]); diff != "" {
			return diff
		}
	}
	for id := range before.links {
		if diff := report("links of "+id, before.links[id], after.links[id]); diff != "" {
			return diff
		}
	}
	for key := range before.findings {
		if diff := report("finding "+key, before.findings[key], after.findings[key]); diff != "" {
			return diff
		}
	}
	return report("counts", []int{len(before.pages), len(before.fields), len(before.bodies), len(before.links), len(before.findings)},
		[]int{len(after.pages), len(after.fields), len(after.bodies), len(after.links), len(after.findings)})
}

func TestSinceFollowsTheSitesClockNotOurs(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)
	skew := 5 * time.Second
	for i := range s.plugin.items {
		s.plugin.items[i].Modified = start.Add(-time.Minute - skew)
	}

	s.clock.Set(start.Add(10 * time.Second))
	s.run(t, observe.Options{})
	owner := s.stores.sites[siteID]
	observation := owner.Settings.Observation
	if observation.LastFinishedAt == nil || observation.LastModifiedSeen == nil || !observation.LastModifiedSeen.Equal(start.Add(-time.Minute-skew)) {
		t.Fatalf("the walk recorded %+v, want when it finished and the newest stamp it saw", observation)
	}

	editedOnTheSiteAt := start.Add(12*time.Second - skew)
	for i := range s.plugin.items {
		if s.plugin.items[i].ID == tb500ID {
			s.plugin.items[i].Title = "TB-500, edited"
			s.plugin.items[i].Modified = editedOnTheSiteAt
		}
	}
	s.clock.Set(start.Add(time.Hour))
	state := s.run(t, observe.Options{})
	if state.Full || state.Since == nil || !state.Since.Before(editedOnTheSiteAt) {
		t.Fatalf("the second walk asked since %v for a site whose clock runs %s behind; the edit at %v would be missed", state.Since, skew, editedOnTheSiteAt)
	}
	tb := s.pageAt(t, "/tb-500/")
	if title := s.fieldOf(t, tb.ID, field.Title); title.Current != "TB-500, edited" {
		t.Errorf("the edit made on the site's clock was not observed: title = %q", title.Current)
	}
}

func TestTheObservationReadsTheSiteOutsideItsWriteUnit(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)

	s.run(t, observe.Options{})
	s.plugin.raws[tb500ID] = rawOf("Edited on the site.")
	s.touch(tb500ID, start.Add(30*time.Minute))
	for i := range s.plugin.items {
		if s.plugin.items[i].ID == tb500ID {
			s.plugin.items[i].ContentHash = block.BodyHash("Edited on the site.")
		}
	}
	s.clock.Set(start.Add(time.Hour))
	s.run(t, observe.Options{})

	if reads := s.stores.guard.inUnit(); reads != 0 {
		t.Fatalf("%d reads of the site happened inside the write unit, which holds the one SQLite writer for their whole duration", reads)
	}
}

func TestEveryNthObservationWalksFully(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	owner := s.stores.sites[siteID]
	owner.Settings.Observe.FullWalkEvery = 3
	s.stores.sites[siteID] = owner

	fulls := make([]bool, 0, 5)
	for i := range 5 {
		s.clock.Set(start.Add(time.Duration(i) * time.Hour))
		fulls = append(fulls, s.run(t, observe.Options{}).Full)
	}
	if want := []bool{true, false, false, true, false}; !slices.Equal(fulls, want) {
		t.Errorf("full walks = %v, want %v", fulls, want)
	}
	s.clock.Set(start.Add(6 * time.Hour))
	if !s.run(t, observe.Options{Full: true}).Full {
		t.Error("a full walk asked for is not full")
	}
}

func TestCoreWithoutThePluginAlwaysWalksFully(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.readers.Plugin = nil
	s.readers.Manifest = observe.Manifest{}
	s.core.items = map[string][]observe.Item{
		"page": {
			{ID: homeID, Type: "page", Link: baseURL + "/", Slug: "home", Status: "publish", Title: "Home", Content: homeRaw, Modified: earlier},
			{ID: tb500ID, Type: "page", Link: baseURL + "/tb-500/", Slug: "tb-500", Status: "publish", Title: "TB-500", Content: tb500Raw, Modified: earlier, Parent: homeID},
			{ID: shopID, Type: "page", Link: baseURL + "/shop/", Slug: "shop", Status: "publish", Title: "Shop", Modified: earlier},
		},
		"post": {{ID: 40, Type: "post", Link: baseURL + "/news/", Slug: "news", Status: "draft", Title: "News", Content: "<p>Soon.</p>", Modified: earlier}},
	}

	first := s.run(t, observe.Options{})
	s.clock.Set(start.Add(time.Hour))
	second := s.run(t, observe.Options{})
	if !first.Full || !second.Full {
		t.Errorf("core walks = %t %t, want both full", first.Full, second.Full)
	}
	if len(s.stores.pages) != 5 {
		t.Fatalf("pages = %d, want home, tb-500, shop, news and the store's product", len(s.stores.pages))
	}
	tb500 := s.pageAt(t, "/tb-500/")
	if body := s.stores.bodies[tb500.ID][block.Current]; body.Raw != tb500Raw {
		t.Errorf("the body came from core's content.raw, got %q", body.Raw)
	}
	home := s.pageAt(t, "/")
	if tb500.ParentPageID == nil || *tb500.ParentPageID != home.ID || tb500.ParentReason != pagemap.ParentWP {
		t.Errorf("tb-500's parent = %v %s, want home by wp_parent", tb500.ParentPageID, tb500.ParentReason)
	}
	if home.TechnicalReason != pagemap.TechnicalNone {
		t.Errorf("core's front page is technical for %q, want the root of the site", home.TechnicalReason)
	}
	product := s.pageAt(t, "/product/bpc-157-peptide/")
	if product.WPType != pagemap.WPProduct || product.WPID == nil || *product.WPID != productID {
		t.Errorf("the store's product row = %+v", product)
	}
	if s.pageAt(t, "/shop/").TechnicalReason != pagemap.TechnicalShop {
		t.Error("the store's settings name the shop page")
	}
	if got := s.fieldOf(t, product.ID, field.ProductDescription); got.Current != "<p>BPC-157 is a pentadecapeptide.</p>" {
		t.Errorf("product description = %+v", got)
	}
}

func TestTechnicalPagesAreNamedWithTheirReason(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	kept := s.seedPage(t, pagemap.Page{ID: "kept", Path: "/peptides/", WPType: pagemap.WPPage, WPID: new(peptidesID), Status: pagemap.StatusPublished, TechnicalReason: pagemap.TechnicalWorkbook})

	s.run(t, observe.Options{})
	cases := map[string]pagemap.TechnicalReason{
		"/": pagemap.TechnicalNone, "/shop/": pagemap.TechnicalShop, "/account/": pagemap.TechnicalShortcode,
		"/tb-500/": pagemap.TechnicalNone, "/peptides/": pagemap.TechnicalWorkbook,
	}
	for path, want := range cases {
		if got := s.pageAt(t, path).TechnicalReason; got != want {
			t.Errorf("%s technical reason = %q, want %q", path, got, want)
		}
	}
	if s.pageAt(t, "/peptides/").ID != kept.ID {
		t.Error("the seeded row was not the one matched")
	}
	detected := s.openFindings(finding.TechnicalDetected)
	if len(detected) != 2 {
		t.Errorf("technical_detected findings = %d, want one per reason the observation set", len(detected))
	}
	account := s.pageAt(t, "/account/")
	if f, ok := s.finding(finding.TechnicalDetected, finding.SubjectPage(account.ID)); !ok || !slices.Contains(f.Options, finding.ClearTechnical) {
		t.Errorf("the account page's finding = %+v %t", f, ok)
	}
}

func TestAPlannedPageKeepsItsWorkbookParent(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	parent := s.seedPage(t, pagemap.Page{ID: "pep", Path: "/peptides/", WPType: pagemap.WPPage, WPID: new(peptidesID), Status: pagemap.StatusPublished})
	planned := s.seedPage(t, pagemap.Page{ID: "planned", Path: "/semax/", WPType: pagemap.WPPage, Status: pagemap.StatusPlanned, ParentPageID: &parent.ID, ParentReason: pagemap.ParentWorkbook, Title: "Semax"})

	s.run(t, observe.Options{})
	after := s.stores.pages[planned.ID]
	if after.ParentPageID == nil || *after.ParentPageID != parent.ID || after.ParentReason != pagemap.ParentWorkbook || after.WPID != nil {
		t.Errorf("the planned page changed: %+v", after)
	}
	if len(s.openFindings(finding.ParentDiffers)) != 0 {
		t.Error("a planned page raises no parent finding")
	}
	if _, has := s.stores.identities[planned.ID]; has {
		t.Error("a planned page has no identity on the site")
	}
}

func TestTheDerivedParentWinsAndTheWorkbookParentIsAFinding(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	shop := s.seedPage(t, pagemap.Page{ID: "shop", Path: "/shop/", WPType: pagemap.WPPage, WPID: new(shopID), Status: pagemap.StatusPublished})
	pep := s.seedPage(t, pagemap.Page{ID: "pep", Path: "/peptides/", WPType: pagemap.WPPage, WPID: new(peptidesID), Status: pagemap.StatusPublished})
	s.seedPage(t, pagemap.Page{ID: "tb", Path: "/tb-500/", WPType: pagemap.WPPage, WPID: new(tb500ID), Status: pagemap.StatusPublished, ParentPageID: &shop.ID, ParentReason: pagemap.ParentWorkbook})
	for i := range s.plugin.items {
		if s.plugin.items[i].ID == tb500ID {
			s.plugin.items[i].Parent = peptidesID
		}
	}

	s.run(t, observe.Options{})
	tb := s.stores.pages["tb"]
	if tb.ParentPageID == nil || *tb.ParentPageID != pep.ID || tb.ParentReason != pagemap.ParentWP {
		t.Errorf("tb-500's parent = %v %s, want peptides by wp_parent", tb.ParentPageID, tb.ParentReason)
	}
	f, ok := s.finding(finding.ParentDiffers, finding.SubjectPage("tb"))
	if !ok || f.Status != finding.Open {
		t.Fatalf("parent_differs = %+v %t", f, ok)
	}
	var subject struct {
		Workbook string `json:"workbook"`
		Derived  string `json:"derived"`
		Reason   string `json:"reason"`
	}
	if err := json.Unmarshal(f.Subject, &subject); err != nil || subject.Workbook != shop.ID || subject.Derived != pep.ID || subject.Reason != "wp_parent" {
		t.Errorf("subject = %s (%v)", f.Subject, err)
	}
}

func TestProductsFillFieldsAndFactsAndVariationsStayUnderTheirProduct(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.readers.Manifest.Capabilities = slices.DeleteFunc(slices.Clone(s.readers.Manifest.Capabilities), func(c string) bool { return c == "terms" })

	s.run(t, observe.Options{})
	product := s.pageAt(t, "/product/bpc-157-peptide/")
	want := map[field.Key]string{
		field.ProductDescription: "<p>BPC-157 is a pentadecapeptide.</p>", field.ProductShortDescription: "<p>Lyophilized BPC-157.</p>",
		field.Price: "39", field.SKU: "BPC157", field.StockStatus: "instock", field.ManageStock: "false",
		field.Attributes: `[{"name":"Strength","options":["5 mg","10 mg"]}]`,
		field.Instance(field.VariationDescription, "11"): "Five milligrams.",
	}
	for key, value := range want {
		if got := s.fieldOf(t, product.ID, key); got.Current != value || got.Base != value {
			t.Errorf("%s = %+v, want %q", key, got, value)
		}
	}
	var facts struct {
		SKU         string `json:"sku"`
		Price       string `json:"price"`
		ManageStock string `json:"manageStock"`
	}
	if err := json.Unmarshal([]byte(s.fieldOf(t, product.ID, field.Instance(field.Variation, "12")).Current), &facts); err != nil || facts.SKU != "BPC157-10MG" || facts.Price != "69" || facts.ManageStock != "parent" {
		t.Errorf("variation 12 facts = %+v (%v)", facts, err)
	}
	for id := range s.stores.pages {
		if page := s.stores.pages[id]; page.WPID != nil && (*page.WPID == variationAID || *page.WPID == variationBID) {
			t.Errorf("a variation became a page: %+v", page)
		}
	}
	terms := s.stores.terms[product.ID]
	if len(terms) != 1 || terms[0].Slug != "healing-peptides" {
		t.Errorf("without the plugin's terms the store's categories are the terms, got %+v", terms)
	}
	if excerpt := s.fieldOf(t, product.ID, field.Excerpt); excerpt.Current != "<p>Lyophilized BPC-157.</p>" {
		t.Errorf("a product's excerpt is its short description, got %+v", excerpt)
	}
	if changed := s.openFindings(finding.FieldChangedOnSite); len(changed) != 0 {
		t.Errorf("a first observation raised %d field changes: %+v", len(changed), changed)
	}
}

func TestOurOwnWriteRaisesNoFinding(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	tb := s.seedPage(t, pagemap.Page{ID: "tb", Path: "/tb-500/", WPType: pagemap.WPPage, WPID: new(tb500ID), Status: pagemap.StatusPublished})
	s.stores.bodies["tb"] = map[block.Side]block.Snapshot{
		block.Current: {PageID: "tb", Side: block.Current, Raw: tb500Raw, Hash: block.BodyHash(tb500Raw), Kind: block.Classic, ObservedAt: earlier},
		block.Base:    {PageID: "tb", Side: block.Base, Raw: tb500Raw, Hash: block.BodyHash(tb500Raw), Kind: block.Classic, ObservedAt: earlier},
	}

	s.run(t, observe.Options{})
	if len(s.openFindings(finding.BodyChangedOnSite)) != 0 {
		t.Fatal("a body equal to what we wrote raised a finding")
	}

	edited := "Someone edited this page on the site."
	s.plugin.raws[tb500ID] = rawOf(edited)
	for i := range s.plugin.items {
		if s.plugin.items[i].ID == tb500ID {
			s.plugin.items[i].ContentHash = block.BodyHash(edited)
			s.plugin.items[i].Modified = start.Add(time.Minute)
		}
	}
	s.clock.Set(start.Add(time.Hour))
	s.run(t, observe.Options{})
	f, ok := s.finding(finding.BodyChangedOnSite, finding.SubjectPage(tb.ID))
	if !ok || f.Status != finding.Open {
		t.Fatalf("body_changed_on_site = %+v %t", f, ok)
	}
	body := s.stores.bodies["tb"]
	if body[block.Current].Raw != edited || body[block.Base].Raw != tb500Raw {
		t.Errorf("current = %q base = %q, want the edit current and ours as the base", body[block.Current].Raw, body[block.Base].Raw)
	}

	s.plugin.raws[tb500ID] = rawOf(tb500Raw)
	for i := range s.plugin.items {
		if s.plugin.items[i].ID == tb500ID {
			s.plugin.items[i].ContentHash = block.BodyHash(tb500Raw)
			s.plugin.items[i].Modified = start.Add(2 * time.Hour)
		}
	}
	s.clock.Set(start.Add(3 * time.Hour))
	s.run(t, observe.Options{})
	f, _ = s.finding(finding.BodyChangedOnSite, finding.SubjectPage(tb.ID))
	if f.Status != finding.Resolved || f.Resolution != finding.ResolutionObservedClear || f.ResolvedBy != finding.ResolvedByObservation {
		t.Errorf("a body back to the base clears the finding, got %+v", f)
	}
}

func TestAFieldChangedOnTheSiteIsAFindingUntilAccepted(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.run(t, observe.Options{})
	tb := s.pageAt(t, "/tb-500/")

	for i := range s.plugin.items {
		if s.plugin.items[i].ID == tb500ID {
			s.plugin.items[i].Title = "TB-500 Peptide"
		}
	}
	s.touch(tb500ID, start.Add(time.Minute))
	s.store.products[0].SKU = "BPC157-NEW"
	s.store.products[0].Modified = start.Add(time.Minute)
	s.clock.Set(start.Add(time.Hour))
	s.run(t, observe.Options{})

	title := s.fieldOf(t, tb.ID, field.Title)
	if title.Current != "TB-500 Peptide" || title.Base != "TB-500" || !title.Changed() {
		t.Errorf("title = %+v", title)
	}
	f, ok := s.finding(finding.FieldChangedOnSite, finding.SubjectPageField(tb.ID, string(field.Title)))
	if !ok || f.Status != finding.Open || !slices.Contains(f.Options, finding.AcceptSite) {
		t.Errorf("field_changed_on_site = %+v %t", f, ok)
	}
	product := s.pageAt(t, "/product/bpc-157-peptide/")
	if _, raised := s.finding(finding.FieldChangedOnSite, finding.SubjectPageField(product.ID, string(field.SKU))); raised {
		t.Error("a denied field raises no change finding")
	}
	if sku := s.fieldOf(t, product.ID, field.SKU); sku.Current != "BPC157-NEW" || sku.Base != "BPC157-NEW" {
		t.Errorf("a denied field is always agreed, got %+v", sku)
	}

	s.clock.Set(start.Add(2 * time.Hour))
	s.run(t, observe.Options{Full: true})
	if f, _ := s.finding(finding.FieldChangedOnSite, finding.SubjectPageField(tb.ID, string(field.Title))); f.Status != finding.Open {
		t.Error("nothing resolves itself: the finding stays open until a person accepts")
	}
}

func TestAnOutdatedPluginIsAFindingThatClearsWhenUpdated(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.readers.Manifest.Version = "1.2.0"
	s.readers.Manifest.Outdated = true

	s.run(t, observe.Options{})
	f, ok := s.finding(finding.PluginOutdated, finding.SubjectSite)
	if !ok || f.Status != finding.Open || !slices.Contains(f.Options, finding.UpdatePlugin) {
		t.Fatalf("plugin_outdated = %+v %t", f, ok)
	}
	var subject struct {
		Version  string `json:"version"`
		Required string `json:"required"`
	}
	if err := json.Unmarshal(f.Subject, &subject); err != nil || subject.Version != "1.2.0" || subject.Required != "1.4.1" {
		t.Errorf("subject = %s", f.Subject)
	}

	s.readers.Manifest.Version = "1.4.1"
	s.readers.Manifest.Outdated = false
	s.clock.Set(start.Add(time.Hour))
	s.run(t, observe.Options{})
	if f, _ = s.finding(finding.PluginOutdated, finding.SubjectSite); f.Status != finding.Resolved {
		t.Errorf("an updated plugin clears the finding, got %s", f.Status)
	}
}

func TestAPlannedTitleThatDiffersIsAFindingAndThePlanIsKept(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	planned := s.seedPage(t, pagemap.Page{ID: "planned-tb", Path: "/tb-500/", WPType: pagemap.WPPage, Status: pagemap.StatusPlanned, Title: "TB-500 Research Peptide", H1: "TB-500"})

	s.run(t, observe.Options{})
	after := s.stores.pages[planned.ID]
	if after.WPID == nil || *after.WPID != tb500ID || after.Title != "TB-500 Research Peptide" {
		t.Errorf("the plan was not kept: %+v", after)
	}
	f, ok := s.finding(finding.PageTitleDiffers, finding.SubjectPage(planned.ID))
	if !ok || f.Status != finding.Open || !slices.Contains(f.Options, finding.AcceptWorkbook) {
		t.Fatalf("page_title_differs = %+v %t", f, ok)
	}
	var subject struct {
		Planned string `json:"planned"`
		Site    string `json:"site"`
	}
	if err := json.Unmarshal(f.Subject, &subject); err != nil || subject.Planned != "TB-500 Research Peptide" || subject.Site != "TB-500" {
		t.Errorf("subject = %s", f.Subject)
	}
	if title := s.fieldOf(t, planned.ID, field.Title); title.Current != "TB-500" {
		t.Errorf("the field carries the site's value, got %+v", title)
	}
}

func TestTheSummaryCountsTheModel(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.run(t, observe.Options{})

	summary, err := s.service.Summary(context.Background(), siteID)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if summary.Pages != 9 || summary.Observed != 9 || summary.ByType["page"] != 6 || summary.ByType["product"] != 1 || summary.ByType["product_cat"] != 2 {
		t.Errorf("summary = %+v", summary)
	}
	if len(summary.Technical) != 2 || summary.Technical["shop"] != 1 || summary.Technical["shortcode"] != 1 {
		t.Errorf("technical = %v, want the shop and the account page and never the front page", summary.Technical)
	}
	if summary.LinksByType["contextual"] != 1 || summary.ReadOnly != 2 {
		t.Errorf("links = %v read-only = %d", summary.LinksByType, summary.ReadOnly)
	}
	if summary.Findings["technical_detected"] != 2 {
		t.Errorf("findings = %v", summary.Findings)
	}

	tb := s.pageAt(t, "/tb-500/")
	model, err := s.service.Page(context.Background(), tb.ID)
	if err != nil {
		t.Fatalf("Page: %v", err)
	}
	if model.Page.ID != tb.ID || model.Identity.URL != baseURL+"/tb-500/" || model.Body == nil || model.Base == nil || len(model.Fields) == 0 || len(model.Links) != 1 {
		t.Errorf("page model = %+v", model)
	}
}
