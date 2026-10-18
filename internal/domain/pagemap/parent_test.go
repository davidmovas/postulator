package pagemap_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func sitePage(id, path string, wpType pagemap.WPType, wpID int64) pagemap.Page {
	page := pagemap.Page{
		ID: id, SiteID: "site", Path: path, WPType: wpType, Status: pagemap.StatusPublished, WPID: &wpID,
	}
	return page
}

func titled(page pagemap.Page, title string) pagemap.Page {
	page.Observed.Title = title
	return page
}

func derivationPages() []pagemap.Page {
	shop := sitePage("shop", "/shop/", pagemap.WPPage, 5)
	shop.TechnicalReason = pagemap.TechnicalShop
	planned := pagemap.Page{ID: "planned", SiteID: "site", Path: "/planned-child/", WPType: pagemap.WPPage, Status: pagemap.StatusPlanned}
	peptides := "peptides"
	planned.ParentPageID = &peptides
	planned.ParentReason = pagemap.ParentWorkbook
	gone := titled(sitePage("gone", "/gone-guides/", pagemap.WPPage, 30), "Lab Guides")
	gone.Status = pagemap.StatusArchived
	return []pagemap.Page{
		titled(sitePage("home", "/", pagemap.WPPage, 1), "Home"),
		shop,
		titled(sitePage("peptides", "/peptides/", pagemap.WPPage, 2), "Peptides"),
		titled(sitePage("bpc", "/bpc-157/", pagemap.WPPage, 3), "BPC-157"),
		titled(sitePage("tb", "/tb-500/", pagemap.WPPage, 4), "TB-500"),
		titled(sitePage("hub", "/healing-peptides/", pagemap.WPPage, 6), "Healing Peptides"),
		titled(sitePage("guides", "/lab-guides/", pagemap.WPPage, 7), "Lab Guides"),
		titled(sitePage("reconstitution", "/reconstitution/", pagemap.WPPage, 8), "Reconstitution"),
		gone,
		titled(sitePage("guide", "/guide/", pagemap.WPPage, 20), "Guide"),
		titled(sitePage("setup", "/guide/setup/", pagemap.WPPage, 21), "Setup"),
		sitePage("collide", "/collide/", pagemap.WPPage, 100000),
		titled(sitePage("pepcat", "/product-category/peptides/", pagemap.WPProductCategory, 100000), "Peptides"),
		titled(sitePage("healing", "/product-category/peptides/healing-peptides/", pagemap.WPProductCategory, 100001), "Healing Peptides"),
		titled(sitePage("uncategorized", "/product-category/uncategorized/", pagemap.WPProductCategory, 100002), "Uncategorized"),
		titled(sitePage("product", "/product/bpc-157-peptide/", pagemap.WPProduct, 10), "BPC-157 Peptide"),
		planned,
	}
}

func trail(source pagemap.TrailSource, crumbs ...string) pagemap.Trail {
	t := pagemap.Trail{Source: source, Crumbs: make([]pagemap.Crumb, 0, len(crumbs)/2)}
	for i := 0; i+1 < len(crumbs); i += 2 {
		t.Crumbs = append(t.Crumbs, pagemap.Crumb{Name: crumbs[i], Path: crumbs[i+1], Href: crumbs[i+1]})
	}
	return t
}

func crumbs(source pagemap.TrailSource, list ...pagemap.Crumb) pagemap.Trail {
	return pagemap.Trail{Source: source, Crumbs: list}
}

func linked(name, path string) pagemap.Crumb {
	return pagemap.Crumb{Name: name, Path: path, Href: path}
}

func TestDeriveParentResolvesTheNearestCrumb(t *testing.T) {
	t.Parallel()

	index := pagemap.NewIndex(derivationPages())
	byID := func(id string) pagemap.Page {
		page, found := index.ByID(id)
		if !found {
			t.Fatalf("the fixture has no page %s", id)
		}
		return page
	}
	wpParent := func(id int64) *int64 { return &id }
	home := linked("Home", "/")

	cases := map[string]struct {
		page       string
		evidence   pagemap.ParentSignals
		parent     string
		reason     pagemap.ParentReason
		method     pagemap.ParentMethod
		trailFrom  pagemap.TrailSource
		unresolved []string
		ambiguous  map[string][]string
	}{
		"the page itself is dropped and the nearest crumb is the parent by its link": {
			page:     "bpc",
			evidence: pagemap.ParentSignals{Trails: []pagemap.Trail{trail(pagemap.TrailJSONLD, "Home", "/", "Peptides", "/peptides/", "BPC-157", "/bpc-157/")}},
			parent:   "peptides", reason: pagemap.ParentJSONLD, method: pagemap.MethodHref, trailFrom: pagemap.TrailJSONLD,
		},
		"a crumb after the page's own is not its parent": {
			page:     "bpc",
			evidence: pagemap.ParentSignals{Trails: []pagemap.Trail{trail(pagemap.TrailJSONLD, "Home", "/", "Peptides", "/peptides/", "BPC-157", "/bpc-157/", "Reviews", "/guide/")}},
			parent:   "peptides", reason: pagemap.ParentJSONLD, method: pagemap.MethodHref, trailFrom: pagemap.TrailJSONLD,
		},
		"a trail of Home and the page names the front page when nothing else does": {
			page:     "peptides",
			evidence: pagemap.ParentSignals{Trails: []pagemap.Trail{trail(pagemap.TrailJSONLD, "Home", "/", "Peptides", "/peptides/")}},
			parent:   "home", reason: pagemap.ParentJSONLD, method: pagemap.MethodHref, trailFrom: pagemap.TrailJSONLD,
		},
		"the front page has no parent": {
			page:     "home",
			evidence: pagemap.ParentSignals{Trails: []pagemap.Trail{trail(pagemap.TrailJSONLD, "Home", "/")}},
			reason:   pagemap.ParentNone,
		},
		"an unresolved crumb is named and the next source is taken": {
			page: "bpc",
			evidence: pagemap.ParentSignals{Trails: []pagemap.Trail{
				trail(pagemap.TrailJSONLD, "Home", "/", "Peptides", "/peptides/", "Healing", "/healing/", "BPC-157", "/bpc-157/"),
				trail(pagemap.TrailNav, "Home", "/", "Peptides", "/peptides/", "Healing", "/healing/", "BPC-157", "/bpc-157/"),
			}},
			parent: "home", reason: pagemap.ParentURL,
			unresolved: []string{"/healing/"},
		},
		"the list with the most resolving crumbs wins": {
			page: "bpc",
			evidence: pagemap.ParentSignals{Trails: []pagemap.Trail{
				trail(pagemap.TrailJSONLD, "Home", "/", "BPC-157", "/bpc-157/"),
				trail(pagemap.TrailJSONLD, "Home", "/", "Peptides", "/peptides/", "BPC-157", "/bpc-157/"),
			}},
			parent: "peptides", reason: pagemap.ParentJSONLD, method: pagemap.MethodHref, trailFrom: pagemap.TrailJSONLD,
		},
		"a tie goes to the list the nav shows": {
			page: "bpc",
			evidence: pagemap.ParentSignals{Trails: []pagemap.Trail{
				trail(pagemap.TrailJSONLD, "Home", "/", "Shop", "/shop/", "BPC-157", "/bpc-157/"),
				trail(pagemap.TrailJSONLD, "Home", "/", "Peptides", "/peptides/", "BPC-157", "/bpc-157/"),
				trail(pagemap.TrailNav, "Home", "/", "Peptides", "/peptides/", "BPC-157", "/bpc-157/"),
			}},
			parent: "peptides", reason: pagemap.ParentJSONLD, method: pagemap.MethodHref, trailFrom: pagemap.TrailJSONLD,
		},
		"a list that names only the front page yields to another list": {
			page: "bpc",
			evidence: pagemap.ParentSignals{Trails: []pagemap.Trail{
				trail(pagemap.TrailJSONLD, "Home", "/", "Shop", "/shop/", "BPC-157", "/bpc-157/"),
				trail(pagemap.TrailJSONLD, "Home", "/", "Peptides", "/peptides/", "BPC-157", "/bpc-157/"),
			}},
			parent: "peptides", reason: pagemap.ParentJSONLD, method: pagemap.MethodHref, trailFrom: pagemap.TrailJSONLD,
		},
		"the nav decides when no list resolves": {
			page:     "bpc",
			evidence: pagemap.ParentSignals{Trails: []pagemap.Trail{trail(pagemap.TrailNav, "Home", "/", "Peptides", "/peptides/", "BPC-157", "/bpc-157/")}},
			parent:   "peptides", reason: pagemap.ParentARIA, method: pagemap.MethodHref, trailFrom: pagemap.TrailNav,
		},
		"a Home-only list does not hide a nav with an unlinked crumb": {
			page: "reconstitution",
			evidence: pagemap.ParentSignals{Trails: []pagemap.Trail{
				trail(pagemap.TrailJSONLD, "Home", "/", "Reconstitution", "/reconstitution/"),
				crumbs(pagemap.TrailNav, home, pagemap.Crumb{Name: "Lab Guides"}, pagemap.Crumb{Name: "Reconstitution", Path: "/reconstitution/"}),
			}},
			parent: "guides", reason: pagemap.ParentARIA, method: pagemap.MethodName, trailFrom: pagemap.TrailNav,
		},
		"a term page's WordPress parent beats a trail that names only the front page": {
			page:     "healing",
			evidence: pagemap.ParentSignals{Trails: []pagemap.Trail{trail(pagemap.TrailJSONLD, "Home", "/", "Healing Peptides", "/product-category/peptides/healing-peptides/")}, WPParentID: wpParent(100000)},
			parent:   "pepcat", reason: pagemap.ParentWP,
		},
		"a nested page's WordPress parent": {
			page: "setup", evidence: pagemap.ParentSignals{WPParentID: wpParent(20)},
			parent: "guide", reason: pagemap.ParentWP,
		},
		"a product with no trail has no parent": {
			page: "product", reason: pagemap.ParentNone,
		},
		"a crumb that links off the site is unresolved and never falls to its name": {
			page: "bpc",
			evidence: pagemap.ParentSignals{Trails: []pagemap.Trail{crumbs(pagemap.TrailJSONLD,
				home, pagemap.Crumb{Name: "Peptides", Href: "https://elsewhere.example/peptides/"}, linked("BPC-157", "/bpc-157/"))}},
			parent: "home", reason: pagemap.ParentURL,
			unresolved: []string{"Peptides"},
		},
		"a crumb resolves through the redirect its link led to": {
			page: "tb",
			evidence: pagemap.ParentSignals{Trails: []pagemap.Trail{crumbs(pagemap.TrailJSONLD,
				home, pagemap.Crumb{Name: "Peptides"},
				pagemap.Crumb{Name: "Healing Peptides", Path: "/category/healing-peptides/", Href: "/category/healing-peptides/", Final: "/healing-peptides/", Status: 200},
				linked("TB-500", "/tb-500/"))}},
			parent: "hub", reason: pagemap.ParentJSONLD, method: pagemap.MethodRedirect, trailFrom: pagemap.TrailJSONLD,
		},
		"a redirect to a page of another name is not followed": {
			page: "tb",
			evidence: pagemap.ParentSignals{Trails: []pagemap.Trail{crumbs(pagemap.TrailJSONLD,
				home, pagemap.Crumb{Name: "Healing Peptides", Path: "/healing/", Href: "/healing/", Final: "/guide/", Status: 200}, linked("TB-500", "/tb-500/"))}},
			parent: "home", reason: pagemap.ParentURL,
			unresolved: []string{"/healing/"},
		},
		"a redirect to the front page is not followed": {
			page: "tb",
			evidence: pagemap.ParentSignals{Trails: []pagemap.Trail{crumbs(pagemap.TrailJSONLD,
				home, pagemap.Crumb{Name: "Home", Path: "/healing/", Href: "/healing/", Final: "/", Status: 200}, linked("TB-500", "/tb-500/"))}},
			parent: "home", reason: pagemap.ParentURL,
			unresolved: []string{"/healing/"},
		},
		"a modelled archive whose own visit lands on its hub hands the crumb to the hub": {
			page: "bpc",
			evidence: pagemap.ParentSignals{
				Trails:   []pagemap.Trail{trail(pagemap.TrailJSONLD, "Home", "/", "Healing Peptides", "/product-category/peptides/healing-peptides/", "BPC-157", "/bpc-157/")},
				Landings: map[string]string{"/product-category/peptides/healing-peptides/": "/healing-peptides/"},
			},
			parent: "hub", reason: pagemap.ParentJSONLD, method: pagemap.MethodRedirect, trailFrom: pagemap.TrailJSONLD,
		},
		"a dead link falls to its name": {
			page: "reconstitution",
			evidence: pagemap.ParentSignals{Trails: []pagemap.Trail{crumbs(pagemap.TrailNav,
				home, pagemap.Crumb{Name: "Lab Guides", Path: "/old-guides/", Href: "/old-guides/", Final: "/old-guides/", Status: 404}, linked("Reconstitution", "/reconstitution/"))}},
			parent: "guides", reason: pagemap.ParentARIA, method: pagemap.MethodName, trailFrom: pagemap.TrailNav,
		},
		"a live link to a page the model lacks never falls to its name": {
			page: "bpc",
			evidence: pagemap.ParentSignals{Trails: []pagemap.Trail{crumbs(pagemap.TrailJSONLD,
				home, pagemap.Crumb{Name: "Uncategorized", Path: "/category/uncategorized/", Href: "/category/uncategorized/", Final: "/category/uncategorized/", Status: 200},
				linked("BPC-157", "/bpc-157/"))}},
			parent: "home", reason: pagemap.ParentURL,
			unresolved: []string{"/category/uncategorized/"},
		},
		"an unlinked crumb two pages answer to is ambiguous": {
			page: "bpc",
			evidence: pagemap.ParentSignals{Trails: []pagemap.Trail{crumbs(pagemap.TrailNav,
				home, pagemap.Crumb{Name: "peptides"}, pagemap.Crumb{Name: "BPC-157", Path: "/bpc-157/"})}},
			parent: "home", reason: pagemap.ParentURL,
			ambiguous: map[string][]string{"peptides": {"peptides", "pepcat"}},
		},
		"an unlinked crumb no page answers to is unresolved": {
			page: "bpc",
			evidence: pagemap.ParentSignals{Trails: []pagemap.Trail{crumbs(pagemap.TrailNav,
				home, pagemap.Crumb{Name: "Nowhere"}, pagemap.Crumb{Name: "BPC-157", Path: "/bpc-157/"})}},
			parent: "home", reason: pagemap.ParentURL,
			unresolved: []string{"Nowhere"},
		},
		"an entity-encoded name compares by the store's key": {
			page: "reconstitution",
			evidence: pagemap.ParentSignals{Trails: []pagemap.Trail{crumbs(pagemap.TrailNav,
				home, pagemap.Crumb{Name: " LAB&#32;guides "}, linked("Reconstitution", "/reconstitution/"))}},
			parent: "guides", reason: pagemap.ParentARIA, method: pagemap.MethodName, trailFrom: pagemap.TrailNav,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := pagemap.DeriveParent(byID(tc.page), tc.evidence, index)
			switch {
			case tc.parent == "" && got.ParentID != nil:
				t.Errorf("parent = %s, want none", *got.ParentID)
			case tc.parent != "" && (got.ParentID == nil || *got.ParentID != tc.parent):
				t.Errorf("parent = %v, want %s", got.ParentID, tc.parent)
			}
			if got.Reason != tc.reason || got.Method != tc.method {
				t.Errorf("reason = %s by %q, want %s by %q", got.Reason, got.Method, tc.reason, tc.method)
			}
			switch {
			case tc.trailFrom == "" && got.Trail != nil:
				t.Errorf("a trail decided (%s), want none", got.Trail.Source)
			case tc.trailFrom != "" && (got.Trail == nil || got.Trail.Source != tc.trailFrom):
				t.Errorf("deciding trail = %v, want source %s", got.Trail, tc.trailFrom)
			}
			unresolved := make([]string, 0, len(got.Unresolved))
			for _, crumb := range got.Unresolved {
				unresolved = append(unresolved, crumb.Key())
			}
			if !slices.Equal(unresolved, tc.unresolved) && (len(unresolved) > 0 || len(tc.unresolved) > 0) {
				t.Errorf("unresolved = %v, want %v", unresolved, tc.unresolved)
			}
			if len(got.Ambiguous) != len(tc.ambiguous) {
				t.Fatalf("ambiguous = %+v, want %v", got.Ambiguous, tc.ambiguous)
			}
			for _, tie := range got.Ambiguous {
				if want := tc.ambiguous[tie.Crumb.Key()]; !slices.Equal(tie.Candidates, want) {
					t.Errorf("ambiguous %s = %v, want %v", tie.Crumb.Key(), tie.Candidates, want)
				}
			}
		})
	}
}

func TestARootCrumbIsDroppedWithoutAFrontPage(t *testing.T) {
	t.Parallel()

	pages := []pagemap.Page{
		sitePage("koffein", "/product-category/koffein/", pagemap.WPProductCategory, 100000),
		sitePage("powder", "/product/powder/", pagemap.WPProduct, 10),
	}
	index := pagemap.NewIndex(pages)
	powder, _ := index.ByID("powder")

	got := pagemap.DeriveParent(powder, pagemap.ParentSignals{Trails: []pagemap.Trail{
		trail(pagemap.TrailJSONLD, "Home", "/", "Powder", "/product/powder/"),
	}}, index)
	if got.ParentID != nil || got.Reason != pagemap.ParentNone || len(got.Unresolved) != 0 {
		t.Errorf("derivation = %+v, want no parent and nothing unresolved", got)
	}
}

func TestByWordPressIDTellsPostsFromTerms(t *testing.T) {
	t.Parallel()

	index := pagemap.NewIndex(derivationPages())
	term, found := index.ByWordPressID(pagemap.WPProductCategory, 100000)
	if !found || term.ID != "pepcat" {
		t.Errorf("the term 100000 = %v %t, want pepcat", term.ID, found)
	}
	post, found := index.ByWordPressID(pagemap.WPPage, 100000)
	if !found || post.ID != "collide" {
		t.Errorf("the post 100000 = %v %t, want collide", post.ID, found)
	}
	product, found := index.ByWordPressID(pagemap.WPType("faq"), 10)
	if !found || product.ID != "product" {
		t.Errorf("a post of any type shares the posts' id space, got %v %t", product.ID, found)
	}
	if _, found := index.ByWordPressID(pagemap.WPPage, 404); found {
		t.Error("an unknown id is found")
	}
}

func TestByNameAnswersEveryOnSitePageThatCarriesTheName(t *testing.T) {
	t.Parallel()

	index := pagemap.NewIndex(derivationPages())
	ids := func(pages []pagemap.Page) []string {
		out := make([]string, 0, len(pages))
		for _, page := range pages {
			out = append(out, page.ID)
		}
		return out
	}
	if got := ids(index.ByName("PEPTIDES")); !slices.Equal(got, []string{"peptides", "pepcat"}) {
		t.Errorf("ByName(PEPTIDES) = %v, want the hub and the term", got)
	}
	if got := ids(index.ByName("Lab Guides")); !slices.Equal(got, []string{"guides"}) {
		t.Errorf("ByName(Lab Guides) = %v, want only the page on the site, not the archived one", got)
	}
	if got := index.ByName("  "); len(got) != 0 {
		t.Errorf("ByName(blank) = %v, want nothing", ids(got))
	}
}
