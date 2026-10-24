package steps

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func plannedRange() pagemap.Page {
	return pagemap.Page{
		ID: "page", SiteID: "site", Path: "/components/batteries/e-bike-range/", Slug: "e-bike-range", WPType: pagemap.WPPage,
		Title: "How Far Can an E-Bike Go?", H1: "How Far Can an E-Bike Go?", Status: pagemap.StatusPublished,
	}
}

func TestMismatchesCompareThePlanWithWhatTheSiteAnswered(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		held answered
		want []string
	}{
		{
			name: "the site agrees",
			held: answered{
				link: "https://shop.example.com/components/batteries/e-bike-range/", slug: "e-bike-range", status: "publish",
				title: "How Far Can an E-Bike Go?", h1: "How Far Can an E-Bike Go?",
			},
		},
		{
			name: "wordpress flattened the address",
			held: answered{link: "https://shop.example.com/e-bike-range/", slug: "e-bike-range", status: "publish"},
			want: []string{FieldPath},
		},
		{
			name: "wordpress renamed a taken slug",
			held: answered{link: "https://shop.example.com/components/batteries/e-bike-range-2/", slug: "e-bike-range-2", status: "publish"},
			want: []string{FieldPath, FieldSlug},
		},
		{
			name: "the page stayed a draft",
			held: answered{link: "https://shop.example.com/components/batteries/e-bike-range/", slug: "e-bike-range", status: "draft"},
			want: []string{FieldStatus},
		},
		{
			name: "the heading is not the one that was asked for",
			held: answered{
				link: "https://shop.example.com/components/batteries/e-bike-range/", slug: "e-bike-range", status: "publish",
				title: "How Far Can an E-Bike Go?", h1: "E-Bike Range",
			},
			want: []string{FieldH1},
		},
		{name: "nothing has been read back yet"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := make([]string, 0, len(tc.want))
			for _, found := range mismatches(plannedRange(), tc.held) {
				got = append(got, found.Field)
			}
			if !slices.Equal(got, tc.want) {
				t.Fatalf("mismatched fields = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMismatchesCheckOnlyWhatWasAskedFor(t *testing.T) {
	t.Parallel()

	unasked := pagemap.Page{ID: "page", SiteID: "site", Path: "/components/", Slug: "components", WPType: pagemap.WPPage, Status: pagemap.StatusPlanned}
	held := answered{link: "https://shop.example.com/components/", slug: "components", status: "draft", title: "E-Bike Components", h1: "E-Bike Components"}
	if found := mismatches(unasked, held); len(found) != 0 {
		t.Fatalf("mismatches = %+v, want none: the plan asked for no title and no heading", found)
	}
}

func TestAMismatchNamesBothValues(t *testing.T) {
	t.Parallel()

	found := mismatches(plannedRange(), answered{link: "https://shop.example.com/e-bike-range/", slug: "e-bike-range"})
	if len(found) != 1 {
		t.Fatalf("mismatches = %+v, want one", found)
	}
	if found[0].Planned != "/components/batteries/e-bike-range/" || found[0].Actual != "/e-bike-range/" {
		t.Fatalf("the mismatch = %+v, want it to name what was asked for and what is there", found[0])
	}
}

func TestAProductIsComparedOnItsStatusAlone(t *testing.T) {
	t.Parallel()

	product := pagemap.Page{
		ID: "liquid", SiteID: "site", Path: "/product/bpc-157-liquid/", PlannedPath: "/bpc-157/liquid/", Slug: "liquid",
		WPType: pagemap.WPProduct, Status: pagemap.StatusPublished, Title: "Buy BPC-157 Liquid", H1: "BPC-157 Liquid Form",
	}
	held := answered{link: "https://shop.example.com/product/bpc-157-liquid/", slug: "bpc-157-liquid", status: "publish", title: "BPC-157 Liquid"}
	if found := mismatches(product, held); len(found) != 0 {
		t.Fatalf("mismatches = %+v; the store owns a product's address, slug and name", found)
	}

	held.status = "draft"
	found := mismatches(product, held)
	if len(found) != 1 || found[0].Field != FieldStatus {
		t.Fatalf("mismatches = %+v, want the status the client changed", found)
	}
}

func TestAPathIsTheAddressTheSiteAnswers(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"https://shop.example.com/Coffee/Espresso": "/coffee/espresso/",
		"/coffee/": "/coffee/",
		"/coffee":  "/coffee/",
		"a b":      "a b",
	}
	for link, want := range cases {
		if got := pathOf(link); got != want {
			t.Errorf("pathOf(%q) = %q, want %q", link, got, want)
		}
	}
}
