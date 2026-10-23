package pagemap_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func beneath(page pagemap.Page, parentID string) pagemap.Page {
	page.ParentPageID = &parentID
	return page
}

func expecting(page pagemap.Page, trail ...string) pagemap.Page {
	page.PlannedTrail = trail
	return page
}

func trailSite() (site pagemap.TrailSite, byID map[string]pagemap.Page) {
	shop := beneath(titled(sitePage("shop", "/shop/", pagemap.WPPage, 2), "Shop"), "peptides")
	shop.TechnicalReason = pagemap.TechnicalShop
	pages := []pagemap.Page{
		titled(sitePage("home", "/", pagemap.WPPage, 1), "Home"),
		shop,
		beneath(titled(sitePage("peptides", "/peptides/", pagemap.WPPage, 3), "Peptides"), "home"),
		beneath(titled(sitePage("bpc", "/peptides/bpc-157/", pagemap.WPPage, 4), "BPC-157 peptide"), "peptides"),
		beneath(titled(sitePage("liquid", "/peptides/bpc-157/liquid/", pagemap.WPPage, 5), "BPC-157 liquid"), "bpc"),
		beneath(titled(sitePage("blends", "/blends/", pagemap.WPPage, 6), "Blends"), "home"),
		beneath(titled(sitePage("kit", "/shop/kit/", pagemap.WPPage, 7), "Kit"), "shop"),
		beneath(titled(sitePage("pepcat", "/product-category/peptides/", pagemap.WPProductCategory, 100), "Peptides"), "peptides"),
		beneath(titled(sitePage("tincture", "/tincture/", pagemap.WPPage, 8), "Tincture"), "pepcat"),
		titled(sitePage("vial", "/product/bpc-157-5mg/", pagemap.WPProduct, 500), "BPC-157 5 mg"),
		titled(sitePage("pair", "/product/pair/", pagemap.WPProduct, 501), "Pair"),
		titled(sitePage("healer", "/product/healer/", pagemap.WPProduct, 502), "Healer"),
		titled(sitePage("bare", "/product/bare/", pagemap.WPProduct, 503), "Bare"),
	}
	byID = make(map[string]pagemap.Page, len(pages))
	for i := range pages {
		byID[pages[i].ID] = pages[i]
	}
	index := pagemap.NewIndexNamed(pages, map[string][]string{
		"bpc": {"BPC-157 Peptide", "BPC-157"}, "peptides": {"Peptides &amp; More"},
	})
	carried := map[string][]pagemap.Term{
		"vial": {
			{Taxonomy: "product_cat", TermID: 101, Name: "BPC-157", ParentTermID: 100},
			{Taxonomy: "entity", TermID: 900, Name: "Blends"},
		},
		"pair":   {{Taxonomy: "product_cat", TermID: 102, Name: "Sale"}, {Taxonomy: "product_cat", TermID: 101, Name: "BPC-157", ParentTermID: 100}},
		"healer": {{Taxonomy: "product_cat", TermID: 103, Name: "Healing"}},
	}
	return pagemap.NewTrailSite(index, carried, map[int64]int64{103: 100}), byID
}

func TestTrailDiffers(t *testing.T) {
	t.Parallel()

	site, pages := trailSite()
	cases := []struct {
		name    string
		page    pagemap.Page
		differs bool
		site    []string
	}{
		{name: "the same names", page: expecting(pages["liquid"], "Peptides & More", "BPC-157"), site: []string{"Peptides & More", "BPC-157 Peptide"}},
		{name: "a name compares by its key", page: expecting(pages["liquid"], "PEPTIDES", "bpc-157 peptide"), site: []string{"Peptides & More", "BPC-157 Peptide"}},
		{name: "a different middle level", page: expecting(pages["liquid"], "Blends", "BPC-157"), differs: true, site: []string{"Peptides & More", "BPC-157 Peptide"}},
		{name: "an extra site level", page: expecting(pages["liquid"], "BPC-157"), differs: true, site: []string{"Peptides & More", "BPC-157 Peptide"}},
		{name: "a level the site lacks", page: expecting(pages["bpc"], "Peptides", "Blends"), differs: true, site: []string{"Peptides & More"}},
		{name: "the front page is left out", page: expecting(pages["bpc"], "Peptides"), site: []string{"Peptides & More"}},
		{name: "a page at the top under a sheet level", page: expecting(pages["blends"], "Peptides"), differs: true, site: []string{}},
		{name: "a technical page is left out", page: expecting(pages["kit"], "Peptides"), site: []string{"Peptides & More"}},
		{name: "a store category twin is the same node", page: expecting(pages["tincture"], "Peptides"), site: []string{"Peptides & More"}},
		{name: "a product's term chain", page: expecting(pages["vial"], "Peptides", "BPC-157"), site: []string{"Peptides", "BPC-157"}},
		{name: "a product under another category", page: expecting(pages["vial"], "Peptides", "TB-500"), differs: true, site: []string{"Peptides", "BPC-157"}},
		{name: "a product in one of its categories", page: expecting(pages["pair"], "Sale"), site: []string{"Sale"}},
		{name: "a product in none of its categories names the nearest", page: expecting(pages["pair"], "Peptides"), differs: true, site: []string{"Peptides", "BPC-157"}},
		{name: "a term's parent comes from the category's own record", page: expecting(pages["healer"], "Peptides", "Healing"), site: []string{"Peptides", "Healing"}},
		{name: "a product in no category", page: expecting(pages["bare"], "Peptides"), differs: true, site: []string{}},
		{name: "an empty trail never differs", page: pages["liquid"], site: []string{}},
		{name: "an empty trail never differs on a product", page: pages["bare"], site: []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			diff, differs := pagemap.TrailDiffers(tc.page, site)
			if differs != tc.differs {
				t.Errorf("differs = %v, want %v (%+v)", differs, tc.differs, diff)
			}
			if !slices.Equal(diff.Site, tc.site) {
				t.Errorf("site = %q, want %q", diff.Site, tc.site)
			}
			if want := tc.page.PlannedTrail; len(want) > 0 && !slices.Equal(diff.Sheet, want) {
				t.Errorf("sheet = %q, want %q", diff.Sheet, want)
			}
		})
	}
}

func TestATrailDiffMarshalsBothListsWhateverTheyHold(t *testing.T) {
	t.Parallel()

	site, pages := trailSite()
	for name, tc := range map[string]struct {
		page pagemap.Page
		want string
	}{
		"both lists":       {page: expecting(pages["blends"], "Peptides"), want: `{"sheet":["Peptides"],"site":[]}`},
		"a product's list": {page: expecting(pages["vial"], "Blends"), want: `{"sheet":["Blends"],"site":["Peptides","BPC-157"]}`},
		"nothing expected": {page: pages["liquid"], want: `{"sheet":[],"site":[]}`},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			diff, _ := pagemap.TrailDiffers(tc.page, site)
			encoded, err := json.Marshal(diff)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(encoded) != tc.want {
				t.Errorf("subject = %s, want %s", encoded, tc.want)
			}
		})
	}
}

func TestParentNamedPrefersAPageOverItsTermTwin(t *testing.T) {
	t.Parallel()

	pages := derivationPages()
	index := pagemap.NewIndex(pages)
	child := sitePage("child", "/child/", pagemap.WPPage, 60)
	ids := func(found []pagemap.Page) []string {
		out := make([]string, 0, len(found))
		for i := range found {
			out = append(out, found[i].ID)
		}
		return out
	}
	cases := []struct {
		name  string
		page  pagemap.Page
		level string
		want  []string
	}{
		{name: "the hub over its category", page: child, level: "Peptides", want: []string{"peptides"}},
		{name: "a category with no page of its name", page: child, level: "Blends", want: []string{"blends", "peptide-blends"}},
		{name: "two pages of one name", page: child, level: "Capsules", want: []string{"capsules", "forms-capsules"}},
		{name: "never the page itself", page: titled(sitePage("bpc", "/bpc-157/", pagemap.WPPage, 3), "BPC-157"), level: "BPC-157", want: []string{}},
		{name: "never a page below it", page: titled(sitePage("guide", "/guide/", pagemap.WPPage, 20), "Guide"), level: "Setup", want: []string{}},
		{name: "a name nothing carries", page: child, level: "Nowhere", want: []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := ids(index.ParentNamed(tc.page, tc.level)); !slices.Equal(got, tc.want) {
				t.Errorf("ParentNamed(%s) = %v, want %v", tc.level, got, tc.want)
			}
		})
	}
}
