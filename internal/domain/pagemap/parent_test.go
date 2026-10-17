package pagemap_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func sitePage(id, path string, wpType pagemap.WPType, wpID int64) pagemap.Page {
	page := pagemap.Page{
		ID: id, SiteID: "site", Path: path, WPType: wpType, Status: pagemap.StatusPublished, WPID: &wpID,
	}
	return page
}

func derivationPages() []pagemap.Page {
	home := sitePage("home", "/", pagemap.WPPage, 1)
	home.TechnicalReason = pagemap.TechnicalFront
	shop := sitePage("shop", "/shop/", pagemap.WPPage, 5)
	shop.TechnicalReason = pagemap.TechnicalShop
	planned := pagemap.Page{ID: "planned", SiteID: "site", Path: "/planned-child/", WPType: pagemap.WPPage, Status: pagemap.StatusPlanned}
	peptides := "peptides"
	planned.ParentPageID = &peptides
	planned.ParentReason = pagemap.ParentWorkbook
	return []pagemap.Page{
		home,
		shop,
		sitePage("peptides", "/peptides/", pagemap.WPPage, 2),
		sitePage("bpc", "/bpc-157/", pagemap.WPPage, 3),
		sitePage("guide", "/guide/", pagemap.WPPage, 20),
		sitePage("setup", "/guide/setup/", pagemap.WPPage, 21),
		sitePage("collide", "/collide/", pagemap.WPPage, 100000),
		sitePage("pepcat", "/product-category/peptides/", pagemap.WPProductCategory, 100000),
		sitePage("healing", "/product-category/peptides/healing-peptides/", pagemap.WPProductCategory, 100001),
		sitePage("product", "/product/bpc-157-peptide/", pagemap.WPProduct, 10),
		planned,
	}
}

func trail(source pagemap.TrailSource, crumbs ...string) pagemap.Trail {
	t := pagemap.Trail{Source: source, Crumbs: make([]pagemap.Crumb, 0, len(crumbs)/2)}
	for i := 0; i+1 < len(crumbs); i += 2 {
		t.Crumbs = append(t.Crumbs, pagemap.Crumb{Name: crumbs[i], Path: crumbs[i+1]})
	}
	return t
}

func TestDeriveParentTakesTheNearestResolvingCrumb(t *testing.T) {
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

	cases := map[string]struct {
		page       string
		trails     []pagemap.Trail
		wpParentID *int64
		parent     string
		reason     pagemap.ParentReason
		trailFrom  pagemap.TrailSource
		unresolved []pagemap.Crumb
	}{
		"the page itself is dropped and the nearest crumb is the parent": {
			page:   "bpc",
			trails: []pagemap.Trail{trail(pagemap.TrailJSONLD, "Home", "/", "Peptides", "/peptides/", "BPC-157", "/bpc-157/")},
			parent: "peptides", reason: pagemap.ParentJSONLD, trailFrom: pagemap.TrailJSONLD,
		},
		"a trail of Home and the page falls through to the URL": {
			page:   "peptides",
			trails: []pagemap.Trail{trail(pagemap.TrailJSONLD, "Home", "/", "Peptides", "/peptides/")},
			parent: "home", reason: pagemap.ParentURL,
		},
		"an unresolved crumb is named and the next source is taken": {
			page: "bpc",
			trails: []pagemap.Trail{
				trail(pagemap.TrailJSONLD, "Home", "/", "Peptides", "/peptides/", "Healing Peptides", "/healing-peptides/", "BPC-157", "/bpc-157/"),
				trail(pagemap.TrailNav, "Home", "/", "Peptides", "/peptides/", "Healing Peptides", "/healing-peptides/", "BPC-157", "/bpc-157/"),
			},
			parent: "home", reason: pagemap.ParentURL,
			unresolved: []pagemap.Crumb{{Name: "Healing Peptides", Path: "/healing-peptides/"}},
		},
		"the list with the most resolving crumbs wins": {
			page: "bpc",
			trails: []pagemap.Trail{
				trail(pagemap.TrailJSONLD, "Home", "/", "BPC-157", "/bpc-157/"),
				trail(pagemap.TrailJSONLD, "Home", "/", "Peptides", "/peptides/", "BPC-157", "/bpc-157/"),
			},
			parent: "peptides", reason: pagemap.ParentJSONLD, trailFrom: pagemap.TrailJSONLD,
		},
		"a tie goes to the list the nav shows": {
			page: "bpc",
			trails: []pagemap.Trail{
				trail(pagemap.TrailJSONLD, "Home", "/", "Shop", "/shop/", "BPC-157", "/bpc-157/"),
				trail(pagemap.TrailJSONLD, "Home", "/", "Peptides", "/peptides/", "BPC-157", "/bpc-157/"),
				trail(pagemap.TrailNav, "Home", "/", "Peptides", "/peptides/", "BPC-157", "/bpc-157/"),
			},
			parent: "peptides", reason: pagemap.ParentJSONLD, trailFrom: pagemap.TrailJSONLD,
		},
		"a tie without a nav takes the first list": {
			page: "bpc",
			trails: []pagemap.Trail{
				trail(pagemap.TrailJSONLD, "Home", "/", "Shop", "/shop/", "BPC-157", "/bpc-157/"),
				trail(pagemap.TrailJSONLD, "Home", "/", "Peptides", "/peptides/", "BPC-157", "/bpc-157/"),
			},
			parent: "home", reason: pagemap.ParentURL,
		},
		"the nav decides when no list resolves": {
			page: "bpc",
			trails: []pagemap.Trail{
				trail(pagemap.TrailNav, "Home", "/", "Peptides", "/peptides/", "BPC-157", "/bpc-157/"),
			},
			parent: "peptides", reason: pagemap.ParentARIA, trailFrom: pagemap.TrailNav,
		},
		"a term page's WordPress parent is its parent term": {
			page:       "healing",
			trails:     []pagemap.Trail{trail(pagemap.TrailJSONLD, "Home", "/", "Healing Peptides", "/product-category/peptides/healing-peptides/")},
			wpParentID: wpParent(100000),
			parent:     "pepcat", reason: pagemap.ParentWP,
		},
		"a nested page's WordPress parent": {
			page: "setup", wpParentID: wpParent(20),
			parent: "guide", reason: pagemap.ParentWP,
		},
		"a product with no trail has no parent": {
			page: "product", reason: pagemap.ParentNone,
		},
		"a nearest crumb that points off the site is unresolved": {
			page: "bpc",
			trails: []pagemap.Trail{
				trail(pagemap.TrailJSONLD, "Home", "/", "Peptides", "/peptides/", "Elsewhere", "", "BPC-157", "/bpc-157/"),
			},
			parent: "home", reason: pagemap.ParentURL,
			unresolved: []pagemap.Crumb{{Name: "Elsewhere", Path: ""}},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := pagemap.DeriveParent(byID(tc.page), tc.trails, tc.wpParentID, index)
			switch {
			case tc.parent == "" && got.ParentID != nil:
				t.Errorf("parent = %s, want none", *got.ParentID)
			case tc.parent != "" && (got.ParentID == nil || *got.ParentID != tc.parent):
				t.Errorf("parent = %v, want %s", got.ParentID, tc.parent)
			}
			if got.Reason != tc.reason {
				t.Errorf("reason = %s, want %s", got.Reason, tc.reason)
			}
			switch {
			case tc.trailFrom == "" && got.Trail != nil:
				t.Errorf("a trail decided (%s), want none", got.Trail.Source)
			case tc.trailFrom != "" && (got.Trail == nil || got.Trail.Source != tc.trailFrom):
				t.Errorf("deciding trail = %v, want source %s", got.Trail, tc.trailFrom)
			}
			if len(got.Unresolved) != len(tc.unresolved) {
				t.Fatalf("unresolved = %v, want %v", got.Unresolved, tc.unresolved)
			}
			for i := range tc.unresolved {
				if got.Unresolved[i] != tc.unresolved[i] {
					t.Errorf("unresolved[%d] = %v, want %v", i, got.Unresolved[i], tc.unresolved[i])
				}
			}
		})
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
