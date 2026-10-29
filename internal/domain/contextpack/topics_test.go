package contextpack_test

import (
	"maps"
	"reflect"
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/keyword"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
)

type topicSite struct {
	fixture
	chains map[string]pagemap.Chain
	built  []string
}

func chainOf(label string, links ...pagemap.ChainLink) pagemap.Chain {
	if links == nil {
		links = []pagemap.ChainLink{}
	}
	chain := pagemap.Chain{Links: links, Label: label}
	if len(links) > 0 {
		chain.Break = links[len(links)-1].Break
	}
	return chain
}

func toPage(name, pageID string) pagemap.ChainLink {
	return pagemap.ChainLink{Name: name, PageID: pageID}
}

func toName(name string) pagemap.ChainLink {
	return pagemap.ChainLink{Name: name}
}

func brokenAt(name string, why pagemap.ChainBreak) pagemap.ChainLink {
	return pagemap.ChainLink{Name: name, Break: why}
}

func replicaTopics() topicSite {
	home := onSitePage("home", "/", "Home", 1, pagemap.WPPage)
	pepcat := typed(onSitePage("pepcat", "/product-category/peptides/", "Peptides", 20, pagemap.WPProductCategory), pagetype.StoreCategory)
	healcat := typed(under(onSitePage("healcat", "/product-category/peptides/healing-peptides/", "Healing Peptides", 21, pagemap.WPProductCategory), "pepcat", pagemap.ParentWP), pagetype.StoreCategory)
	hub := typed(under(onSitePage("hub", "/healing-peptides/", "Healing Peptides", 2, pagemap.WPPage), "pepcat", pagemap.ParentJSONLD), pagetype.Hub)
	tb := typed(under(onSitePage("tb", "/tb-500/", "TB-500", 3, pagemap.WPPage), "hub", pagemap.ParentJSONLD), pagetype.Entity)
	tb.Keywords = keyword.New([]keyword.Keyword{volume("tb-500", 900)})
	bpc := typed(under(onSitePage("bpc", "/product/bpc-157-peptide/", "BPC-157 Peptide", 10, pagemap.WPProduct), "healcat", pagemap.ParentJSONLD), pagetype.Product)
	guides := typed(under(onSitePage("guides", "/lab-guides/", "Lab Guides", 4, pagemap.WPPage), "home", pagemap.ParentJSONLD), pagetype.Catalog)
	reconstitution := typed(under(onSitePage("reconstitution", "/reconstitution/", "Reconstitution", 5, pagemap.WPPage), "guides", pagemap.ParentARIA), pagetype.Entity)
	hello := typed(under(onSitePage("hello", "/hello-world/", "Hello world!", 6, pagemap.WPPost), "home", pagemap.ParentURL), pagetype.Entity)
	about := typed(under(onSitePage("about", "/about/", "About", 7, pagemap.WPPage), "home", pagemap.ParentJSONLD), pagetype.Catalog)
	lonely := typed(under(onSitePage("lonely", "/lonely/", "Lonely", 8, pagemap.WPPage), "home", pagemap.ParentURL), pagetype.Entity)

	return topicSite{
		fixture: fixture{
			pages: []pagemap.Page{home, pepcat, healcat, hub, tb, bpc, guides, reconstitution, hello, about, lonely},
			fields: map[string][]field.Value{
				"bpc": values("bpc", "attributes", `[{"name":"Strength","options":["5 mg"],"visible":true,"variation":false}]`),
			},
		},
		chains: map[string]pagemap.Chain{
			"hub":            chainOf("Healing Peptides", toPage("Peptides", "pepcat")),
			"tb":             chainOf("TB-500", toPage("Peptides", "pepcat"), toPage("Healing Peptides", "hub")),
			"bpc":            chainOf("BPC-157 Peptide", toPage("Peptides", "pepcat"), toPage("Healing Peptides", "healcat")),
			"guides":         chainOf("Lab Guides"),
			"reconstitution": chainOf("Reconstitution", toPage("Lab Guides", "guides")),
			"hello":          chainOf("Hello world!", brokenAt("Uncategorized", pagemap.ChainUnknownLink)),
			"about":          chainOf("About"),
		},
	}
}

func (s topicSite) plan(t *testing.T) contextpack.TopicPlan {
	t.Helper()
	return s.snapshot(t).Map(nil).TopicMap(s.chains, s.built)
}

func (s topicSite) with(change func(*topicSite)) topicSite {
	next := s
	next.pages = slices.Clone(s.pages)
	next.entities = slices.Clone(s.entities)
	next.edges = slices.Clone(s.edges)
	next.chains = maps.Clone(s.chains)
	next.built = slices.Clone(s.built)
	change(&next)
	return next
}

func (s *topicSite) page(id string, change func(*pagemap.Page)) {
	for i := range s.pages {
		if s.pages[i].ID == id {
			change(&s.pages[i])
		}
	}
}

type write struct {
	subject, action, page, name, parent, parentEntity, entity string
}

func writesOf(plan contextpack.TopicPlan) []write {
	out := make([]write, 0, len(plan.Writes))
	for i := range plan.Writes {
		node := plan.Writes[i]
		out = append(out, write{
			subject: node.Subject, action: string(node.Action), page: node.PageID, name: node.Name,
			parent: node.Parent, parentEntity: node.ParentEntityID, entity: node.EntityID,
		})
	}
	return out
}

func missesOf(plan contextpack.TopicPlan) map[string]contextpack.TopicSkip {
	out := make(map[string]contextpack.TopicSkip, len(plan.Unplaced))
	for _, miss := range plan.Unplaced {
		out[miss.Path] = miss.Skip
	}
	return out
}

func TestTheSitesBreadcrumbsBuildItsTopicalMap(t *testing.T) {
	t.Parallel()

	plan := replicaTopics().plan(t)
	want := []write{
		{subject: "page:guides", action: "create", page: "guides", name: "Lab Guides"},
		{subject: "topic:/peptides", action: "create", name: "Peptides"},
		{subject: "page:hub", action: "create", page: "hub", name: "Healing Peptides", parent: "topic:/peptides"},
		{subject: "page:reconstitution", action: "create", page: "reconstitution", name: "Reconstitution", parent: "page:guides"},
		{subject: "page:bpc", action: "create", page: "bpc", name: "BPC-157 Peptide", parent: "page:hub"},
		{subject: "page:tb", action: "create", page: "tb", name: "TB-500", parent: "page:hub"},
	}
	if got := writesOf(plan); !reflect.DeepEqual(got, want) {
		t.Fatalf("the writes =\n%+v\nwant, parent first, the store term a page-less topic, the hub's term folded into it and About left alone\n%+v", got, want)
	}
	if got, want := missesOf(plan), map[string]contextpack.TopicSkip{
		"/hello-world/": contextpack.TopicUnknownLink, "/lonely/": contextpack.TopicNoBreadcrumbs,
	}; !reflect.DeepEqual(got, want) {
		t.Errorf("the pages left without an entity = %v, want %v", got, want)
	}

	byPage := make(map[string]contextpack.TopicNode, len(plan.Writes))
	for i := range plan.Writes {
		byPage[plan.Writes[i].PageID] = plan.Writes[i]
	}
	if facts := byPage["bpc"].Facts; !slices.Equal(facts, []graph.Fact{{Label: "Strength", Value: "5 mg"}}) {
		t.Errorf("the product's facts = %v, want its attributes", facts)
	}
	if words := byPage["tb"].Keywords; len(words) != 1 || words[0].Text != "tb-500" {
		t.Errorf("the page's keywords = %v, want the page's own", words)
	}
	for i := 1; i < len(plan.Writes); i++ {
		if plan.Writes[i-1].Order >= plan.Writes[i].Order {
			t.Errorf("the order keys do not rise: %q then %q", plan.Writes[i-1].Order, plan.Writes[i].Order)
		}
	}
}

func TestTheMapKeepsWhatStandsAndSkipsWhatItCannotBeSureOf(t *testing.T) {
	t.Parallel()

	base := replicaTopics()
	cases := map[string]struct {
		site   topicSite
		want   []write
		misses map[string]contextpack.TopicSkip
	}{
		"a page a person mapped is kept as it is and its children hang under its entity": {
			site: base.with(func(s *topicSite) {
				s.entities = append(s.entities, graph.Entity{ID: "e-heal", SiteID: siteID, Name: "Healing", CanonicalPageID: new("hub"), Source: graph.SourceUser})
				s.page("hub", func(p *pagemap.Page) { p.EntityID = new("e-heal") })
				delete(s.chains, "guides")
				delete(s.chains, "reconstitution")
			}),
			want: []write{
				{subject: "page:hub", action: "record", page: "hub", name: "Healing Peptides", parent: "topic:/peptides", entity: "e-heal"},
				{subject: "page:bpc", action: "create", page: "bpc", name: "BPC-157 Peptide", parent: "page:hub", parentEntity: "e-heal"},
				{subject: "page:tb", action: "create", page: "tb", name: "TB-500", parent: "page:hub", parentEntity: "e-heal"},
			},
			misses: map[string]contextpack.TopicSkip{
				"/hello-world/": contextpack.TopicUnknownLink, "/lonely/": contextpack.TopicNoBreadcrumbs, "/reconstitution/": contextpack.TopicNoBreadcrumbs,
			},
		},
		"an entity of the topic's name at its place is reused and one a page may take is adopted": {
			site: base.with(func(s *topicSite) {
				s.entities = append(s.entities,
					graph.Entity{ID: "e-pep", SiteID: siteID, Name: "peptides", Source: graph.SourceImport},
					graph.Entity{ID: "e-heal", SiteID: siteID, Name: "Healing Peptides", ScopeID: new("e-pep"), Source: graph.SourceImport})
				delete(s.chains, "guides")
				delete(s.chains, "reconstitution")
			}),
			want: []write{
				{subject: "topic:/peptides", action: "record", name: "Peptides", entity: "e-pep"},
				{subject: "page:hub", action: "adopt", page: "hub", name: "Healing Peptides", parent: "topic:/peptides", parentEntity: "e-pep", entity: "e-heal"},
				{subject: "page:bpc", action: "create", page: "bpc", name: "BPC-157 Peptide", parent: "page:hub", parentEntity: "e-heal"},
				{subject: "page:tb", action: "create", page: "tb", name: "TB-500", parent: "page:hub", parentEntity: "e-heal"},
			},
			misses: map[string]contextpack.TopicSkip{
				"/hello-world/": contextpack.TopicUnknownLink, "/lonely/": contextpack.TopicNoBreadcrumbs, "/reconstitution/": contextpack.TopicNoBreadcrumbs,
			},
		},
		"a name another page carries is taken, so its subtree waits": {
			site: base.with(func(s *topicSite) {
				s.entities = append(s.entities,
					graph.Entity{ID: "e-pep", SiteID: siteID, Name: "Peptides", Source: graph.SourceImport},
					graph.Entity{ID: "e-heal", SiteID: siteID, Name: "Healing Peptides", ScopeID: new("e-pep"), CanonicalPageID: new("lonely"), Source: graph.SourceImport})
				s.page("lonely", func(p *pagemap.Page) { p.EntityID = new("e-heal") })
				delete(s.chains, "guides")
				delete(s.chains, "reconstitution")
			}),
			want: []write{},
			misses: map[string]contextpack.TopicSkip{
				"/hello-world/": contextpack.TopicUnknownLink, "/reconstitution/": contextpack.TopicNoBreadcrumbs,
				"/healing-peptides/": contextpack.TopicNameTaken, "/tb-500/": contextpack.TopicParentSkipped, "/product/bpc-157-peptide/": contextpack.TopicParentSkipped,
			},
		},
		"what a person removed after an earlier build is not brought back": {
			site: base.with(func(s *topicSite) {
				s.built = []string{"page:hub", "topic:/peptides", "page:guides"}
				s.entities = append(s.entities, graph.Entity{ID: "e-pep", SiteID: siteID, Name: "Peptides", Source: graph.SourceSite})
			}),
			want: []write{},
			misses: map[string]contextpack.TopicSkip{
				"/hello-world/": contextpack.TopicUnknownLink, "/lonely/": contextpack.TopicNoBreadcrumbs,
				"/healing-peptides/": contextpack.TopicRemoved, "/tb-500/": contextpack.TopicParentSkipped,
				"/product/bpc-157-peptide/": contextpack.TopicParentSkipped, "/reconstitution/": contextpack.TopicParentSkipped,
			},
		},
		"two pages that would take one name under one parent are both left": {
			site: base.with(func(s *topicSite) {
				s.chains["tb"] = chainOf("BPC-157 Peptide", toPage("Peptides", "pepcat"), toPage("Healing Peptides", "hub"))
				delete(s.chains, "guides")
				delete(s.chains, "reconstitution")
			}),
			want: []write{
				{subject: "topic:/peptides", action: "create", name: "Peptides"},
				{subject: "page:hub", action: "create", page: "hub", name: "Healing Peptides", parent: "topic:/peptides"},
			},
			misses: map[string]contextpack.TopicSkip{
				"/hello-world/": contextpack.TopicUnknownLink, "/lonely/": contextpack.TopicNoBreadcrumbs, "/reconstitution/": contextpack.TopicNoBreadcrumbs,
				"/tb-500/": contextpack.TopicNameTaken, "/product/bpc-157-peptide/": contextpack.TopicNameTaken,
			},
		},
		"a page whose tree parent is not its breadcrumbs' is left, so the build never raises entity_parent_differs": {
			site: base.with(func(s *topicSite) {
				s.page("tb", func(p *pagemap.Page) { p.ParentPageID, p.ParentReason = new("reconstitution"), pagemap.ParentWorkbook })
			}),
			want: []write{
				{subject: "page:guides", action: "create", page: "guides", name: "Lab Guides"},
				{subject: "topic:/peptides", action: "create", name: "Peptides"},
				{subject: "page:hub", action: "create", page: "hub", name: "Healing Peptides", parent: "topic:/peptides"},
				{subject: "page:reconstitution", action: "create", page: "reconstitution", name: "Reconstitution", parent: "page:guides"},
				{subject: "page:bpc", action: "create", page: "bpc", name: "BPC-157 Peptide", parent: "page:hub"},
			},
			misses: map[string]contextpack.TopicSkip{
				"/hello-world/": contextpack.TopicUnknownLink, "/lonely/": contextpack.TopicNoBreadcrumbs, "/tb-500/": contextpack.TopicDisagrees,
			},
		},
		"a page with no breadcrumbs of its own takes the place every trail through it agrees on": {
			site: base.with(func(s *topicSite) {
				delete(s.chains, "hub")
				delete(s.chains, "guides")
				delete(s.chains, "reconstitution")
			}),
			want: []write{
				{subject: "topic:/peptides", action: "create", name: "Peptides"},
				{subject: "page:hub", action: "create", page: "hub", name: "Healing Peptides", parent: "topic:/peptides"},
				{subject: "page:bpc", action: "create", page: "bpc", name: "BPC-157 Peptide", parent: "page:hub"},
				{subject: "page:tb", action: "create", page: "tb", name: "TB-500", parent: "page:hub"},
			},
			misses: map[string]contextpack.TopicSkip{
				"/hello-world/": contextpack.TopicUnknownLink, "/lonely/": contextpack.TopicNoBreadcrumbs, "/reconstitution/": contextpack.TopicNoBreadcrumbs,
			},
		},
		"trails that disagree on a page without its own leave it and everything under it": {
			site: base.with(func(s *topicSite) {
				delete(s.chains, "hub")
				delete(s.chains, "guides")
				delete(s.chains, "reconstitution")
				s.chains["bpc"] = chainOf("BPC-157 Peptide", toName("Compounds"), toPage("Healing Peptides", "healcat"))
			}),
			want: []write{},
			misses: map[string]contextpack.TopicSkip{
				"/hello-world/": contextpack.TopicUnknownLink, "/lonely/": contextpack.TopicNoBreadcrumbs, "/reconstitution/": contextpack.TopicNoBreadcrumbs,
				"/healing-peptides/": contextpack.TopicDisagrees, "/tb-500/": contextpack.TopicParentSkipped, "/product/bpc-157-peptide/": contextpack.TopicParentSkipped,
			},
		},
		"pages whose breadcrumbs loop are left": {
			site: base.with(func(s *topicSite) {
				s.chains["hub"] = chainOf("Healing Peptides", toPage("TB-500", "tb"))
				delete(s.chains, "guides")
				delete(s.chains, "reconstitution")
				delete(s.chains, "bpc")
			}),
			want: []write{},
			misses: map[string]contextpack.TopicSkip{
				"/hello-world/": contextpack.TopicUnknownLink, "/lonely/": contextpack.TopicNoBreadcrumbs, "/reconstitution/": contextpack.TopicNoBreadcrumbs,
				"/product/bpc-157-peptide/": contextpack.TopicNoBreadcrumbs, "/healing-peptides/": contextpack.TopicDisagrees, "/tb-500/": contextpack.TopicDisagrees,
			},
		},
		"a topic under a crumb the model cannot place is left with its page": {
			site: base.with(func(s *topicSite) {
				s.chains["lonely"] = chainOf("Lonely", brokenAt("Shop", pagemap.ChainAmbiguous), toName("Odd Ones"))
				delete(s.chains, "guides")
				delete(s.chains, "reconstitution")
			}),
			want: []write{
				{subject: "topic:/peptides", action: "create", name: "Peptides"},
				{subject: "page:hub", action: "create", page: "hub", name: "Healing Peptides", parent: "topic:/peptides"},
				{subject: "page:bpc", action: "create", page: "bpc", name: "BPC-157 Peptide", parent: "page:hub"},
				{subject: "page:tb", action: "create", page: "tb", name: "TB-500", parent: "page:hub"},
			},
			misses: map[string]contextpack.TopicSkip{
				"/hello-world/": contextpack.TopicUnknownLink, "/reconstitution/": contextpack.TopicNoBreadcrumbs, "/lonely/": contextpack.TopicAmbiguousName,
			},
		},
		"a page with no name anywhere is left": {
			site: base.with(func(s *topicSite) {
				s.page("lonely", func(p *pagemap.Page) { p.Title, p.H1 = "", "" })
				s.chains["lonely"] = chainOf(" ")
				s.chains = map[string]pagemap.Chain{"lonely": s.chains["lonely"]}
			}),
			want: []write{},
			misses: map[string]contextpack.TopicSkip{
				"/hello-world/": contextpack.TopicNoBreadcrumbs, "/healing-peptides/": contextpack.TopicNoBreadcrumbs, "/tb-500/": contextpack.TopicNoBreadcrumbs,
				"/product/bpc-157-peptide/": contextpack.TopicNoBreadcrumbs, "/reconstitution/": contextpack.TopicNoBreadcrumbs, "/lonely/": contextpack.TopicNoName,
			},
		},
		"a page the trails name differently from its title takes the name they use most": {
			site: base.with(func(s *topicSite) {
				delete(s.chains, "hub")
				delete(s.chains, "guides")
				delete(s.chains, "reconstitution")
				s.chains["tb"] = chainOf("TB-500", toPage("Peptides", "pepcat"), toPage("Healing", "hub"))
				s.chains["bpc"] = chainOf("BPC-157 Peptide", toPage("Peptides", "pepcat"), toPage("Healing", "healcat"))
			}),
			want: []write{
				{subject: "topic:/peptides", action: "create", name: "Peptides"},
				{subject: "page:hub", action: "create", page: "hub", name: "Healing", parent: "topic:/peptides"},
				{subject: "page:bpc", action: "create", page: "bpc", name: "BPC-157 Peptide", parent: "page:hub"},
				{subject: "page:tb", action: "create", page: "tb", name: "TB-500", parent: "page:hub"},
			},
			misses: map[string]contextpack.TopicSkip{
				"/hello-world/": contextpack.TopicUnknownLink, "/lonely/": contextpack.TopicNoBreadcrumbs, "/reconstitution/": contextpack.TopicNoBreadcrumbs,
			},
		},
		"a second build over the first writes nothing": {
			site: base.with(func(s *topicSite) {
				s.built = []string{"page:guides", "topic:/peptides", "page:reconstitution", "page:hub", "page:bpc", "page:tb"}
				s.entities = append(s.entities,
					graph.Entity{ID: "e-guides", SiteID: siteID, Name: "Lab Guides", CanonicalPageID: new("guides"), Source: graph.SourceSite},
					graph.Entity{ID: "e-pep", SiteID: siteID, Name: "Peptides", Source: graph.SourceSite},
					graph.Entity{ID: "e-rec", SiteID: siteID, Name: "Reconstitution", ScopeID: new("e-guides"), CanonicalPageID: new("reconstitution"), Source: graph.SourceSite},
					graph.Entity{ID: "e-heal", SiteID: siteID, Name: "Healing Peptides", ScopeID: new("e-pep"), CanonicalPageID: new("hub"), Source: graph.SourceSite},
					graph.Entity{ID: "e-bpc", SiteID: siteID, Name: "BPC-157 Peptide", ScopeID: new("e-heal"), CanonicalPageID: new("bpc"), Source: graph.SourceSite},
					graph.Entity{ID: "e-tb", SiteID: siteID, Name: "TB-500", ScopeID: new("e-heal"), CanonicalPageID: new("tb"), Source: graph.SourceSite})
				for pageID, entityID := range map[string]string{"guides": "e-guides", "reconstitution": "e-rec", "hub": "e-heal", "bpc": "e-bpc", "tb": "e-tb"} {
					s.page(pageID, func(p *pagemap.Page) { p.EntityID = new(entityID) })
				}
			}),
			want: []write{},
			misses: map[string]contextpack.TopicSkip{
				"/hello-world/": contextpack.TopicUnknownLink, "/lonely/": contextpack.TopicNoBreadcrumbs,
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			plan := tc.site.plan(t)
			if got := writesOf(plan); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("the writes =\n%+v\nwant\n%+v", got, tc.want)
			}
			if got := missesOf(plan); !reflect.DeepEqual(got, tc.misses) {
				t.Errorf("the pages left = %v\nwant %v", got, tc.misses)
			}
		})
	}
}
