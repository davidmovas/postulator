package pagemap_test

import (
	"reflect"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func bare(name string) pagemap.Crumb {
	return pagemap.Crumb{Name: name}
}

func TestTopicChainReadsEveryCrumbAboveThePage(t *testing.T) {
	t.Parallel()

	index := pagemap.NewIndex(derivationPages())
	home := linked("Home", "/")
	moved := pagemap.Crumb{Name: "Healing Peptides", Path: "/category/healing-peptides/", Href: "/category/healing-peptides/", Final: "/healing-peptides/", Status: 301}
	unknown := pagemap.Crumb{Name: "Uncategorized", Path: "/category/uncategorized/", Href: "/category/uncategorized/", Status: 200}
	link := func(name, pageID string) pagemap.ChainLink { return pagemap.ChainLink{Name: name, PageID: pageID} }

	cases := map[string]struct {
		page   string
		trails []pagemap.Trail
		want   pagemap.Chain
	}{
		"the front page and the page itself are dropped and the crumb between is a link": {
			page:   "bpc",
			trails: []pagemap.Trail{crumbs(pagemap.TrailJSONLD, home, linked("Peptides", "/peptides/"), linked("BPC-157", "/bpc-157/"))},
			want:   pagemap.Chain{Links: []pagemap.ChainLink{link("Peptides", "peptides")}, Label: "BPC-157"},
		},
		"a crumb no page carries is a bare name above the linked parent": {
			page:   "tb",
			trails: []pagemap.Trail{crumbs(pagemap.TrailJSONLD, home, bare("Research Compounds"), linked("Healing Peptides", "/healing-peptides/"), linked("TB-500", "/tb-500/"))},
			want:   pagemap.Chain{Links: []pagemap.ChainLink{link("Research Compounds", ""), link("Healing Peptides", "hub")}, Label: "TB-500"},
		},
		"an unlinked crumb resolves by the name one live page carries": {
			page:   "reconstitution",
			trails: []pagemap.Trail{crumbs(pagemap.TrailNav, home, bare("Lab Guides"), linked("Reconstitution", "/reconstitution/"))},
			want:   pagemap.Chain{Links: []pagemap.ChainLink{link("Lab Guides", "guides")}, Label: "Reconstitution"},
		},
		"a moved link lands on the page that carries its name": {
			page:   "tb",
			trails: []pagemap.Trail{crumbs(pagemap.TrailJSONLD, home, moved, linked("TB-500", "/tb-500/"))},
			want:   pagemap.Chain{Links: []pagemap.ChainLink{link("Healing Peptides", "hub")}, Label: "TB-500"},
		},
		"a store term stays a link to the term": {
			page: "product",
			trails: []pagemap.Trail{crumbs(pagemap.TrailJSONLD, home, linked("Peptides", "/product-category/peptides/"),
				linked("Healing Peptides", "/product-category/peptides/healing-peptides/"), linked("BPC-157 Peptide", "/product/bpc-157-peptide/"))},
			want: pagemap.Chain{Links: []pagemap.ChainLink{link("Peptides", "pepcat"), link("Healing Peptides", "healing")}, Label: "BPC-157 Peptide"},
		},
		"a technical crumb is dropped": {
			page:   "product",
			trails: []pagemap.Trail{crumbs(pagemap.TrailJSONLD, home, linked("Shop", "/shop/"), linked("BPC-157 Peptide", "/product/bpc-157-peptide/"))},
			want:   pagemap.Chain{Links: []pagemap.ChainLink{}, Label: "BPC-157 Peptide"},
		},
		"Home alone places the page at the root": {
			page:   "peptides",
			trails: []pagemap.Trail{crumbs(pagemap.TrailJSONLD, home, linked("Peptides", "/peptides/"))},
			want:   pagemap.Chain{Links: []pagemap.ChainLink{}, Label: "Peptides"},
		},
		"a live link the model lacks breaks the chain": {
			page:   "guide",
			trails: []pagemap.Trail{crumbs(pagemap.TrailJSONLD, home, unknown, linked("Guide", "/guide/"))},
			want: pagemap.Chain{
				Links: []pagemap.ChainLink{{Name: "Uncategorized", Break: pagemap.ChainUnknownLink}}, Label: "Guide", Break: pagemap.ChainUnknownLink,
			},
		},
		"a name two pages carry breaks the chain": {
			page:   "setup",
			trails: []pagemap.Trail{crumbs(pagemap.TrailNav, home, bare("Capsules"), linked("Setup", "/guide/setup/"))},
			want: pagemap.Chain{
				Links: []pagemap.ChainLink{{Name: "Capsules", Break: pagemap.ChainAmbiguous}}, Label: "Setup", Break: pagemap.ChainAmbiguous,
			},
		},
		"a broken crumb above a linked parent stays in the chain without breaking it": {
			page:   "tb",
			trails: []pagemap.Trail{crumbs(pagemap.TrailJSONLD, home, unknown, linked("Healing Peptides", "/healing-peptides/"), linked("TB-500", "/tb-500/"))},
			want: pagemap.Chain{
				Links: []pagemap.ChainLink{{Name: "Uncategorized", Break: pagemap.ChainUnknownLink}, link("Healing Peptides", "hub")}, Label: "TB-500",
			},
		},
		"a trail whose nearest crumb is a page wins over an earlier one whose nearest is a bare name": {
			page: "tb",
			trails: []pagemap.Trail{
				crumbs(pagemap.TrailJSONLD, home, bare("Research Compounds"), linked("TB-500", "/tb-500/")),
				crumbs(pagemap.TrailNav, home, linked("Healing Peptides", "/healing-peptides/"), linked("TB-500", "/tb-500/")),
			},
			want: pagemap.Chain{Links: []pagemap.ChainLink{link("Healing Peptides", "hub")}, Label: "TB-500"},
		},
		"a bare name wins over Home alone": {
			page: "tb",
			trails: []pagemap.Trail{
				crumbs(pagemap.TrailJSONLD, home, linked("TB-500", "/tb-500/")),
				crumbs(pagemap.TrailNav, home, bare("Research Compounds"), linked("TB-500", "/tb-500/")),
			},
			want: pagemap.Chain{Links: []pagemap.ChainLink{link("Research Compounds", "")}, Label: "TB-500"},
		},
		"a broken trail wins over Home alone, so a specific place is never guessed at the root": {
			page: "guide",
			trails: []pagemap.Trail{
				crumbs(pagemap.TrailJSONLD, home, linked("Guide", "/guide/")),
				crumbs(pagemap.TrailNav, home, unknown, linked("Guide", "/guide/")),
			},
			want: pagemap.Chain{
				Links: []pagemap.ChainLink{{Name: "Uncategorized", Break: pagemap.ChainUnknownLink}}, Label: "Guide", Break: pagemap.ChainUnknownLink,
			},
		},
		"a label is unescaped and its spaces folded": {
			page:   "bpc",
			trails: []pagemap.Trail{crumbs(pagemap.TrailJSONLD, home, bare("  Research &amp;  Compounds "), linked(" BPC&#8209;157 ", "/bpc-157/"))},
			want:   pagemap.Chain{Links: []pagemap.ChainLink{link("Research & Compounds", "")}, Label: "BPC‑157"},
		},
		"a nameless crumb with no page says nothing and is dropped": {
			page:   "bpc",
			trails: []pagemap.Trail{crumbs(pagemap.TrailJSONLD, home, bare(" "), linked("Peptides", "/peptides/"), linked("BPC-157", "/bpc-157/"))},
			want:   pagemap.Chain{Links: []pagemap.ChainLink{link("Peptides", "peptides")}, Label: "BPC-157"},
		},
		"an unlinked Home that names the front page is dropped like the root": {
			page:   "bpc",
			trails: []pagemap.Trail{crumbs(pagemap.TrailNav, bare("Home"), linked("Peptides", "/peptides/"), linked("BPC-157", "/bpc-157/"))},
			want:   pagemap.Chain{Links: []pagemap.ChainLink{link("Peptides", "peptides")}, Label: "BPC-157"},
		},
		"a trail that never names the page has no label": {
			page:   "bpc",
			trails: []pagemap.Trail{crumbs(pagemap.TrailJSONLD, home, linked("Peptides", "/peptides/"))},
			want:   pagemap.Chain{Links: []pagemap.ChainLink{link("Peptides", "peptides")}},
		},
		"a page with no trail has no breadcrumbs": {
			page: "bpc",
			want: pagemap.Chain{Links: []pagemap.ChainLink{}, Break: pagemap.ChainNoTrail},
		},
		"the front page is never placed": {
			page:   "home",
			trails: []pagemap.Trail{crumbs(pagemap.TrailJSONLD, home)},
			want:   pagemap.Chain{Links: []pagemap.ChainLink{}, Break: pagemap.ChainNoTrail},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			page, found := index.ByID(tc.page)
			if !found {
				t.Fatalf("the fixture has no page %s", tc.page)
			}
			got := pagemap.TopicChain(page, pagemap.ParentSignals{Trails: tc.trails}, index)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("TopicChain = %+v\nwant %+v", got, tc.want)
			}
		})
	}
}

func TestTopicChainAgreesWithTheParentTheTreeTook(t *testing.T) {
	t.Parallel()

	index := pagemap.NewIndex(derivationPages())
	home := linked("Home", "/")
	cases := map[string]struct {
		page   string
		trails []pagemap.Trail
	}{
		"by link":     {page: "bpc", trails: []pagemap.Trail{crumbs(pagemap.TrailJSONLD, home, linked("Peptides", "/peptides/"), linked("BPC-157", "/bpc-157/"))}},
		"by name":     {page: "reconstitution", trails: []pagemap.Trail{crumbs(pagemap.TrailNav, home, bare("Lab Guides"), linked("Reconstitution", "/reconstitution/"))}},
		"by a term":   {page: "product", trails: []pagemap.Trail{crumbs(pagemap.TrailJSONLD, home, linked("Healing Peptides", "/product-category/peptides/healing-peptides/"), linked("BPC-157 Peptide", "/product/bpc-157-peptide/"))}},
		"by the list": {page: "tb", trails: []pagemap.Trail{crumbs(pagemap.TrailJSONLD, home, bare("Research Compounds"), linked("TB-500", "/tb-500/")), crumbs(pagemap.TrailNav, home, linked("Healing Peptides", "/healing-peptides/"), linked("TB-500", "/tb-500/"))}},
	}
	for name, tc := range cases {
		page, _ := index.ByID(tc.page)
		signals := pagemap.ParentSignals{Trails: tc.trails}
		derived := pagemap.DeriveParent(page, signals, index)
		chain := pagemap.TopicChain(page, signals, index)
		if derived.ParentID == nil || len(chain.Links) == 0 || chain.Links[len(chain.Links)-1].PageID != *derived.ParentID {
			t.Errorf("%s: the chain %+v ends elsewhere than the tree's parent %v", name, chain, derived.ParentID)
		}
	}
}
