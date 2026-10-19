package sitemodel_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/graph"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const siteID = "site-1"

var stamp = time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)

func typeOf(preset pagetype.Preset) string {
	return "type-" + string(preset)
}

func presetSet() pagetype.Set {
	kinds := make([]pagetype.Type, 0, len(pagetype.Presets()))
	scopes := make([]field.Scope, 0, len(pagetype.Presets())*4)
	for _, preset := range pagetype.Presets() {
		kinds = append(kinds, pagetype.Type{ID: typeOf(preset), SiteID: siteID, Name: preset.DefaultName(), Preset: preset})
		scopes = append(scopes, pagetype.DefaultScopes(preset, typeOf(preset))...)
	}
	return pagetype.NewSet(kinds, scopes)
}

func sitePage(id, path, title string, wpID int64, wpType pagemap.WPType, preset pagetype.Preset, parent string) pagemap.Page {
	page := pagemap.Page{
		ID: id, SiteID: siteID, Path: path, Slug: pagemap.Slug(path), WPType: wpType, WPID: new(wpID),
		Status: pagemap.StatusPublished, Observed: pagemap.Observed{Title: title, H1: title}, PageTypeID: new(typeOf(preset)),
	}
	if parent != "" {
		page.ParentPageID, page.ParentReason = new(parent), pagemap.ParentJSONLD
	}
	return page
}

func edge(id, child, parent string) graph.Edge {
	return graph.Edge{ID: id, SiteID: siteID, FromEntityID: child, ToEntityID: parent, Kind: graph.EdgeParent, Weight: 1, Source: graph.SourceUser, Status: graph.StatusApproved}
}

func link(id, from, to, href string, kind pagemap.LinkType, source pagemap.LinkSource) pagemap.PageLink {
	return pagemap.PageLink{ID: id, SiteID: siteID, FromPageID: from, ToPageID: new(to), ToURL: href, Type: kind, Source: source, ObservedAt: stamp}
}

func site1() *memory {
	hub := sitePage("hub", "/peptides/", "Peptides", 2, pagemap.WPPage, pagetype.Hub, "home")
	hub.EntityID = new("e-peptides")
	child := sitePage("bpc", "/bpc-157/", "BPC-157", 3, pagemap.WPPage, pagetype.Entity, "hub")
	child.EntityID = new("e-bpc")
	return &memory{
		site: site.Site{ID: siteID, Name: "Shop", BaseURL: "https://shop.example.com", Settings: site.DefaultSettings()},
		pages: []pagemap.Page{
			sitePage("home", "/", "Home", 1, pagemap.WPPage, pagetype.Entity, ""),
			hub, child,
			sitePage("far", "/far-away/", "Far Away", 9, pagemap.WPPage, pagetype.Entity, ""),
			sitePage("tb", "/product/tb-500/", "TB-500", 30, pagemap.WPProduct, pagetype.Product, ""),
		},
		entities: []graph.Entity{
			{ID: "e-peptides", SiteID: siteID, Name: "Peptides", CanonicalPageID: new("hub"), Source: graph.SourceUser, Score: 1, Definition: "Peptides for research."},
			{ID: "e-bpc", SiteID: siteID, Name: "BPC-157", CanonicalPageID: new("bpc"), Source: graph.SourceUser, Score: 0.5,
				Anchors: []graph.Anchor{{Text: "BPC-157", Source: graph.AnchorUser, Weight: 1}}},
		},
		edges: []graph.Edge{edge("g1", "e-bpc", "e-peptides")},
		types: presetSet(),
		terms: map[string][]pagemap.Term{},
		fields: map[string][]field.Value{
			"tb":  {field.NewValue("tb", field.Price, "60.00", stamp), field.NewValue("tb", field.SKU, "TB-5", stamp)},
			"far": {field.NewValue("far", field.SEODescription, "Far from the hub.", stamp)},
		},
		bodies: []block.Snapshot{
			{PageID: "bpc", Side: block.Current, Raw: "<p>BPC-157 is studied for repair.</p>", Kind: block.Classic},
		},
		links: []pagemap.PageLink{
			link("l1", "hub", "tb", "/product/tb-500/", pagemap.LinkListing, pagemap.SourceRender),
			link("c1", "bpc", "hub", "https://shop.example.com/peptides/", pagemap.LinkContextual, pagemap.SourceRaw),
			link("c2", "far", "bpc", "https://shop.example.com/bpc-157/", pagemap.LinkContextual, pagemap.SourceRaw),
		},
		findings: []finding.Finding{
			{ID: "f1", SiteID: siteID, PageID: new("hub"), Code: finding.ParentUnresolved, Status: finding.Open},
			{ID: "f2", SiteID: siteID, PageID: new("hub"), Code: finding.RenderUnreachable, Status: finding.Dismissed},
		},
	}
}

func TestAPackReadsOnlyThePagesNeighborhood(t *testing.T) {
	t.Parallel()

	m := site1()
	picture, err := m.service().Picture(t.Context(), sitemodel.PackRequest{PageID: "hub"})
	if err != nil {
		t.Fatalf("Picture: %v", err)
	}
	if m.reads != 1 {
		t.Errorf("the pack read in %d units, want one read", m.reads)
	}
	if len(m.asked) != 1 || slices.Contains(m.asked[0], "far") || !slices.Contains(m.asked[0], "tb") {
		t.Errorf("fields were asked for %v, want the hub's neighborhood without the far page", m.asked)
	}
	if picture.Fitted.Field != field.Body {
		t.Errorf("an unnamed field = %s, want the hub's link field", picture.Fitted.Field)
	}
	if len(picture.Facts.Products) != 1 || picture.Facts.Products[0].Price != "60.00" {
		t.Errorf("products = %+v", picture.Facts.Products)
	}
	if len(picture.Facts.Findings) != 1 || picture.Facts.Findings[0].ID != "f1" {
		t.Errorf("findings = %+v, want the open one only", picture.Facts.Findings)
	}
	if len(picture.Facts.Inbound) != 1 {
		t.Errorf("inbound = %+v, want BPC-157's link", picture.Facts.Inbound)
	}
}

func TestAPackFollowsTheSitesLinkForm(t *testing.T) {
	t.Parallel()

	absolute := site1()
	relative := site1()
	relative.site.Settings.LinkURLForm = site.URLFormRelative

	cases := []struct {
		name string
		m    *memory
		want string
	}{
		{name: "the client's links are absolute", m: absolute, want: "https://shop.example.com/bpc-157/"},
		{name: "the setting says relative", m: relative, want: "/bpc-157/"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			picture, err := tc.m.service().Picture(t.Context(), sitemodel.PackRequest{PageID: "hub", Field: "body"})
			if err != nil {
				t.Fatalf("Picture: %v", err)
			}
			if len(picture.Facts.Targets) != 1 || picture.Facts.Targets[0].Href != tc.want {
				t.Errorf("targets = %+v, want one written as %s", picture.Facts.Targets, tc.want)
			}
		})
	}
}

func TestAPackIsBuiltOverTheClaim(t *testing.T) {
	t.Parallel()

	m := site1()
	claimed := content.LinkContext{PageID: "hub", PageURL: "/peptides/", Targets: []content.LinkTarget{
		{EntityID: "x", PageID: "far", URL: "/far-away/", Anchors: []string{"far away"}, Relation: content.RelationSibling},
	}}
	scope := field.Scope{Key: field.Body, Enabled: true, Instruction: "Written from the claim.", Links: field.NoLinks()}
	claimedType := pagetype.Type{ID: "type-claimed", SiteID: siteID, Name: "Pillar", Preset: pagetype.Hub}
	picture, err := m.service().Picture(t.Context(), sitemodel.PackRequest{
		PageID: "hub", Field: "body", Claim: &sitemodel.Claim{Type: &claimedType, Scope: scope, Links: &claimed},
	})
	if err != nil {
		t.Fatalf("Picture: %v", err)
	}
	text := picture.Fitted.Text()
	if !strings.Contains(text, "Written from the claim.") || !strings.Contains(text, "/far-away/") || !strings.Contains(text, "Page type: Pillar, a hub") {
		t.Errorf("the pack does not carry the claim:\n%s", text)
	}
	if strings.Contains(text, "/bpc-157/ (") {
		t.Errorf("the pack planned links of its own:\n%s", text)
	}
	if !slices.Contains(m.asked[0], "far") {
		t.Errorf("the claimed target's page was not read: %v", m.asked)
	}
}

func TestAPackIsFittedToItsBudget(t *testing.T) {
	t.Parallel()

	m := site1()
	whole, err := m.service().Pack(t.Context(), sitemodel.PackRequest{PageID: "hub"})
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	if len(whole.Dropped) != 0 || whole.Tokens != whole.Built || whole.Field != "body" || len(whole.Sections) == 0 {
		t.Fatalf("whole pack = %+v", whole)
	}
	fitted, err := m.service().Pack(t.Context(), sitemodel.PackRequest{PageID: "hub", Budget: whole.Tokens - 1})
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	if len(fitted.Dropped) == 0 || fitted.Tokens > fitted.Budget || fitted.Built != whole.Built {
		t.Errorf("fitted pack = %+v, want a drop under the budget", fitted)
	}
}

func TestAPackOutOfARunTakesTheRolesBudget(t *testing.T) {
	t.Parallel()

	m := site1()
	whole, err := m.service().Pack(t.Context(), sitemodel.PackRequest{PageID: "hub"})
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	m.budgets = map[domainllm.Role]int{domainllm.RoleWriter: whole.Tokens - 1, domainllm.RoleEditor: 4000}
	fitted, err := m.service().Pack(t.Context(), sitemodel.PackRequest{PageID: "hub"})
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	if fitted.Budget != whole.Tokens-1 || len(fitted.Dropped) == 0 {
		t.Errorf("the body's pack = budget %d, %d drops; want the writer's budget and a drop", fitted.Budget, len(fitted.Dropped))
	}
	meta, err := m.service().Pack(t.Context(), sitemodel.PackRequest{PageID: "hub", Field: "seo_title"})
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	if meta.Budget != 4000 {
		t.Errorf("the SEO title's pack has budget %d, want the editor's 4000", meta.Budget)
	}
	asked, err := m.service().Pack(t.Context(), sitemodel.PackRequest{PageID: "hub", Budget: 100000})
	if err != nil || asked.Budget != 100000 {
		t.Errorf("a budget asked for = %d, %v; want it kept", asked.Budget, err)
	}
}

func TestAPackRefusesWhatTheWriterNeverWrites(t *testing.T) {
	t.Parallel()

	m := site1()
	cases := []struct {
		name string
		req  sitemodel.PackRequest
		code errors.Code
	}{
		{name: "no page", req: sitemodel.PackRequest{}, code: errors.Invalid},
		{name: "an unknown page", req: sitemodel.PackRequest{PageID: "missing"}, code: errors.NotFound},
		{name: "a denied field", req: sitemodel.PackRequest{PageID: "hub", Field: "price"}, code: errors.Invalid},
		{name: "an unknown field", req: sitemodel.PackRequest{PageID: "hub", Field: "flavor"}, code: errors.Invalid},
		{name: "a budget below what every call carries", req: sitemodel.PackRequest{PageID: "hub", Budget: 3}, code: errors.Invalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := m.service().Pack(t.Context(), tc.req); !errors.IsCode(err, tc.code) {
				t.Errorf("Pack(%+v) = %v, want %s", tc.req, err, tc.code)
			}
		})
	}
}
