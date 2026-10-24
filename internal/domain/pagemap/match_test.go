package pagemap_test

import (
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func matchSite() pagemap.RowSite {
	at := time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)
	onSite := func(id, path string, wpID int64, wpType pagemap.WPType, title string) pagemap.Page {
		return pagemap.Page{ID: id, SiteID: "s", Path: path, Slug: pagemap.Slug(path), WPType: wpType, WPID: &wpID,
			Status: pagemap.StatusPublished, Title: title}
	}
	hub := onSite("hub", "/peptides/", 1, pagemap.WPPage, "")
	term := onSite("term", "/product-category/peptides/", 2, pagemap.WPProductCategory, "Peptides")
	lone := onSite("lone", "/product-category/healing/", 3, pagemap.WPProductCategory, "Healing")
	bpc := onSite("bpc", "/bpc-157/", 4, pagemap.WPPage, "")
	bpc.EntityID = new("e-bpc")
	tb := onSite("tb", "/tb-500/", 5, pagemap.WPPage, "")
	tb.PlannedPath = "/peptides/tb-500/"
	product := onSite("product", "/product/bpc-157-peptide/", 6, pagemap.WPProduct, "")
	archived := onSite("old", "/reconstitution/", 7, pagemap.WPPage, "Reconstitution")
	archived.Status = pagemap.StatusArchived
	liquid := onSite("liquid", "/liquid/", 8, pagemap.WPPage, "")
	mapped := onSite("mapped", "/healing-peptides/", 9, pagemap.WPPage, "Healing Peptides")
	mapped.RowKey = "row:Groups:healing peptides"
	planned := pagemap.Page{ID: "planned", SiteID: "s", Path: "/peptides/bpc-157/powder/", Slug: "powder", WPType: pagemap.WPPage,
		Status: pagemap.StatusPlanned, Title: "BPC-157 powder"}
	pages := []pagemap.Page{
		hub, term, lone, bpc, tb, product, archived, liquid, mapped, planned,
		onSite("first", "/guides/storage/", 10, pagemap.WPPage, "Storage"),
		onSite("second", "/lab/storage/", 11, pagemap.WPPage, "Storage"),
		onSite("cafe", "/cafe/", 12, pagemap.WPPage, "Café"),
		onSite("tea", "/tea/", 13, pagemap.WPPage, ""),
	}
	fields := map[string][]field.Value{
		"hub":     {field.NewValue("hub", field.Title, "Peptides", at)},
		"bpc":     {field.NewValue("bpc", field.Title, "BPC-157", at)},
		"tb":      {field.NewValue("tb", field.Title, "TB-500", at), field.NewValue("tb", field.H1, "TB-500 for research", at)},
		"product": {field.NewValue("product", field.Title, "BPC-157 Peptide", at), field.NewValue("product", field.SKU, "BPC157", at)},
		"liquid":  {field.NewValue("liquid", field.Title, "BPC-157 Liquid", at)},
		"tea":     {field.NewValue("tea", field.Title, "Tea &amp; coffee", at)},
	}
	entities := []graph.Entity{
		{ID: "e-peptides", SiteID: "s", Name: "Peptides"},
		{ID: "e-bpc", SiteID: "s", Name: "BPC-157", ScopeID: new("e-peptides")},
	}
	return pagemap.NewRowSite(pages, fields, entities)
}

func TestMatchRow(t *testing.T) {
	t.Parallel()

	site := matchSite()
	cases := []struct {
		name       string
		row        pagemap.Row
		want       string
		rule       pagemap.MatchRule
		candidates int
	}{
		{name: "a URL the site holds", row: pagemap.Row{Path: "/tb-500/"}, want: "tb", rule: pagemap.RuleURL},
		{name: "a URL the page keeps as the sheet's planned address", row: pagemap.Row{Path: "/peptides/tb-500"}, want: "tb", rule: pagemap.RuleURL},
		{name: "a nested sheet URL found by slug on a flat site", row: pagemap.Row{Path: "/peptides/bpc-157/"}, want: "bpc", rule: pagemap.RuleSlug},
		{name: "an entity under its parent, with no URL", row: pagemap.Row{Entity: "bpc-157", Parent: "Peptides"}, want: "bpc", rule: pagemap.RuleEntity},
		{name: "an entity under another parent is not that page", row: pagemap.Row{Entity: "BPC-157", Parent: "Blends"}},
		{name: "a title, with no URL", row: pagemap.Row{Title: "TB-500"}, want: "tb", rule: pagemap.RuleTitle},
		{name: "an H1", row: pagemap.Row{H1: "tb-500 for research"}, want: "tb", rule: pagemap.RuleH1},
		{name: "an H1 naming a page whose title the theme prints", row: pagemap.Row{H1: "BPC-157"}, want: "bpc", rule: pagemap.RuleH1},
		{name: "an H1 is not a title where the page has one of its own", row: pagemap.Row{H1: "TB-500"}},
		{name: "a product's SKU", row: pagemap.Row{Product: true, Title: "Spray", SKU: "bpc157"}, want: "product", rule: pagemap.RuleSKU},
		{name: "a title two pages share under two parents", row: pagemap.Row{Title: "Storage"}, rule: pagemap.RuleTitle, candidates: 2},
		{name: "nothing the site holds", row: pagemap.Row{Title: "Nasal spray"}},
		{name: "a hub and a store category sharing a name", row: pagemap.Row{Title: "Peptides"}, want: "hub", rule: pagemap.RuleTitle},
		{name: "a lone store category", row: pagemap.Row{Title: "Healing"}, want: "lone", rule: pagemap.RuleTitle},
		{name: "a name the store keeps apart", row: pagemap.Row{Title: "CAFÉ"}},
		{name: "a title the site stored with its ampersand escaped", row: pagemap.Row{Title: "Tea & coffee"}, want: "tea", rule: pagemap.RuleTitle},
		{name: "a page on the site wins over a planned row", row: pagemap.Row{Title: "BPC-157 liquid"}, want: "liquid", rule: pagemap.RuleTitle},
		{name: "a second import finds its own planned row", row: pagemap.Row{Title: "BPC-157 Powder"}, want: "planned", rule: pagemap.RuleTitle},
		{name: "a person's mapping wins", row: pagemap.Row{Key: "row:Groups:healing peptides", Path: "/tb-500/"}, want: "mapped", rule: pagemap.RuleMapped},
		{name: "an archived page is never matched", row: pagemap.Row{Title: "Reconstitution"}},
		{name: "a product row never takes a page by its name", row: pagemap.Row{Product: true, Title: "TB-500"}},
		{name: "a page row never takes a product by its name", row: pagemap.Row{Title: "BPC-157 Peptide"}},
		{name: "a page row takes the product at its address", row: pagemap.Row{Path: "/product/bpc-157-peptide/"}, want: "product", rule: pagemap.RuleURL},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := site.Match(tc.row)
			if got.Rule != tc.rule {
				t.Errorf("rule = %q, want %q", got.Rule, tc.rule)
			}
			switch {
			case tc.candidates > 1:
				if !got.Ambiguous() || len(got.Candidates) != tc.candidates || got.Found() {
					t.Errorf("match = %+v, want %d candidates and no page taken", got, tc.candidates)
				}
			case tc.want == "":
				if got.Found() || got.Ambiguous() {
					t.Errorf("match = %+v, want none", got)
				}
			case !got.Found() || got.Page.ID != tc.want:
				t.Errorf("matched %q (%+v), want %q", got.Page.ID, got, tc.want)
			}
		})
	}
}

func TestARowKeyNamesTheRowByItsFirstIdentity(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		row  pagemap.Row
		want string
	}{
		{name: "its address first", row: pagemap.Row{Path: "/peptides/bpc-157/", Title: "BPC-157", Entity: "BPC 157"}, want: "row:Groups:/peptides/bpc-157/"},
		{name: "its title without an address", row: pagemap.Row{Title: " Healing Peptides ", Entity: "Healing"}, want: "row:Groups:healing peptides"},
		{name: "its entity without a title", row: pagemap.Row{Entity: "TB-500", H1: "TB-500 Peptide"}, want: "row:Groups:tb-500"},
		{name: "its H1 before its SKU", row: pagemap.Row{H1: "Caf&eacute; Guide", SKU: "CAF-1"}, want: "row:Groups:café guide"},
		{name: "its SKU alone, folded as a match folds it", row: pagemap.Row{SKU: " MAK-L-10 ", Product: true}, want: "row:Groups:sku:mak-l-10"},
		{name: "its SKU on a page row", row: pagemap.Row{SKU: "Mak-L-10"}, want: "row:Groups:sku:mak-l-10"},
		{name: "nothing to name it by", row: pagemap.Row{Parent: "Peptides", SKU: "  "}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := pagemap.RowKey("Groups", tc.row); got != tc.want {
				t.Errorf("RowKey = %q, want %q", got, tc.want)
			}
		})
	}
	if !strings.HasPrefix(pagemap.RowKey("Groups", pagemap.Row{Title: "A"}), pagemap.RowKeyPrefix("Groups")) {
		t.Errorf("a row key does not start with its sheet's prefix %q", pagemap.RowKeyPrefix("Groups"))
	}
}
