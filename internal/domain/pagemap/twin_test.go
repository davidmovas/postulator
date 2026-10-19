package pagemap_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func twinPage(id, path, title string, wpType pagemap.WPType, wpID int64) pagemap.Page {
	page := pagemap.Page{ID: id, SiteID: "site", Path: path, WPType: wpType, Status: pagemap.StatusPublished, Observed: pagemap.Observed{Title: title}}
	if wpID != 0 {
		page.WPID = &wpID
	}
	return page
}

func TestATwinIsTheOneRegularPageAndTheTermsOfItsName(t *testing.T) {
	t.Parallel()

	hub := twinPage("hub", "/peptides/", "Peptides", pagemap.WPPage, 10)
	term := twinPage("term", "/product-category/peptides/", "Peptides", pagemap.WPProductCategory, 20)
	product := twinPage("product", "/product/peptides/", "Peptides", pagemap.WPProduct, 30)
	planned := pagemap.Page{ID: "planned", SiteID: "site", Path: "/guides/", WPType: pagemap.WPPage, Status: pagemap.StatusPlanned, Title: "Guides"}
	guideTerm := twinPage("guide-term", "/product-category/guides/", "Guides", pagemap.WPProductCategory, 21)
	lone := twinPage("lone", "/product-category/blends/", "Blends", pagemap.WPProductCategory, 22)
	first := twinPage("first", "/oils/", "Oils", pagemap.WPPage, 11)
	second := twinPage("second", "/more-oils/", "OILS", pagemap.WPPost, 12)
	oils := twinPage("oils-term", "/product-category/oils/", "Oils", pagemap.WPProductCategory, 23)
	archived := twinPage("archived", "/old-blends/", "Blends", pagemap.WPPage, 13)
	archived.Status = pagemap.StatusArchived
	teas := twinPage("teas", "/teas/", "Teas", pagemap.WPPage, 14)
	moreTeas := twinPage("more-teas", "/teas-again/", "Teas", pagemap.WPPage, 15)
	teaTerm := twinPage("teas-term", "/product-category/teas/", "Teas", pagemap.WPProductCategory, 24)

	index := pagemap.NewIndex([]pagemap.Page{hub, term, product, planned, guideTerm, lone, first, second, oils, archived, teas, moreTeas, teaTerm})

	cases := []struct {
		name     string
		term     pagemap.Page
		wantHub  string
		wantSeen bool
	}{
		{name: "a hub page stands for its category; a product of the name does not", term: term, wantHub: "hub", wantSeen: true},
		{name: "a planned hub stands for its category", term: guideTerm, wantHub: "planned", wantSeen: true},
		{name: "a category with no page of its name has no twin", term: lone},
		{name: "a post of the category's name is never its twin", term: oils, wantHub: "first", wantSeen: true},
		{name: "two pages of one name are no twin", term: teaTerm},
		{name: "a regular page has no hub twin", term: hub},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, found := index.HubTwin(tc.term)
			if found != tc.wantSeen || (found && got.ID != tc.wantHub) {
				t.Errorf("HubTwin(%s) = %s, %v; want %s, %v", tc.term.ID, got.ID, found, tc.wantHub, tc.wantSeen)
			}
		})
	}

	if terms := index.TermTwins(hub); len(terms) != 1 || terms[0].ID != "term" {
		t.Errorf("TermTwins(hub) = %+v, want the Peptides category", terms)
	}
	if terms := index.TermTwins(planned); len(terms) != 1 || terms[0].ID != "guide-term" {
		t.Errorf("TermTwins(planned) = %+v, want the Guides category", terms)
	}
	if terms := index.TermTwins(term); len(terms) != 0 {
		t.Errorf("TermTwins(term) = %+v, want none for a term page", terms)
	}
	if terms := index.TermTwins(product); len(terms) != 0 {
		t.Errorf("TermTwins(product) = %+v, want none for a product", terms)
	}
	if terms := index.TermTwins(second); len(terms) != 0 {
		t.Errorf("TermTwins(post) = %+v, want none for a post", terms)
	}
}
