package pagemap_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func TestMismatchesComparesThePlanWithTheSite(t *testing.T) {
	t.Parallel()

	planned := func() pagemap.Page {
		p := page(pageA, "/components/batteries/e-bike-range/", nil)
		p.Title = "How Far Can an E-Bike Go?"
		p.H1 = "How Far Can an E-Bike Go?"
		p.Status = pagemap.StatusPublished
		return p
	}

	cases := []struct {
		name     string
		observed pagemap.Observed
		want     []string
	}{
		{
			name: "the site agrees",
			observed: pagemap.Observed{
				Link:   "https://shop.example.com/components/batteries/e-bike-range/",
				Slug:   "e-bike-range",
				Status: "publish",
				Title:  "How Far Can an E-Bike Go?",
				H1:     "How Far Can an E-Bike Go?",
			},
		},
		{
			name: "wordpress flattened the address",
			observed: pagemap.Observed{
				Link:   "https://shop.example.com/e-bike-range/",
				Slug:   "e-bike-range",
				Status: "publish",
			},
			want: []string{"path"},
		},
		{
			name: "wordpress renamed a taken slug",
			observed: pagemap.Observed{
				Link:   "https://shop.example.com/components/batteries/e-bike-range-2/",
				Slug:   "e-bike-range-2",
				Status: "publish",
			},
			want: []string{"path", "slug"},
		},
		{
			name: "the page stayed a draft",
			observed: pagemap.Observed{
				Link:   "https://shop.example.com/components/batteries/e-bike-range/",
				Slug:   "e-bike-range",
				Status: "draft",
			},
			want: []string{"status"},
		},
		{
			name: "the heading is not the one that was asked for",
			observed: pagemap.Observed{
				Link:   "https://shop.example.com/components/batteries/e-bike-range/",
				Slug:   "e-bike-range",
				Status: "publish",
				Title:  "How Far Can an E-Bike Go?",
				H1:     "E-Bike Range",
			},
			want: []string{"h1"},
		},
		{
			name:     "nothing has been read back yet",
			observed: pagemap.Observed{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			subject := planned()
			subject.Observed = tc.observed

			got := make([]string, 0, len(tc.want))
			for _, mismatch := range subject.Mismatches() {
				got = append(got, mismatch.Field)
			}
			if !slices.Equal(got, tc.want) {
				t.Fatalf("mismatched fields = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMismatchesChecksOnlyWhatWasAskedFor(t *testing.T) {
	t.Parallel()

	subject := page(pageA, "/components/", nil)
	subject.Observed = pagemap.Observed{
		Link:   "https://shop.example.com/components/",
		Slug:   "components",
		Status: "draft",
		Title:  "E-Bike Components",
		H1:     "E-Bike Components",
	}

	if found := subject.Mismatches(); len(found) != 0 {
		t.Fatalf("mismatches = %+v, want none: the plan asked for no title and no heading", found)
	}
}

func TestMismatchesNamesBothValues(t *testing.T) {
	t.Parallel()

	subject := page(pageA, "/components/batteries/e-bike-range/", nil)
	subject.Observed = pagemap.Observed{Link: "https://shop.example.com/e-bike-range/", Slug: "e-bike-range"}

	found := subject.Mismatches()
	if len(found) != 1 {
		t.Fatalf("mismatches = %+v, want one", found)
	}
	if found[0].Planned != "/components/batteries/e-bike-range/" || found[0].Actual != "/e-bike-range/" {
		t.Fatalf("the mismatch = %+v, want it to name what was asked for and what is there", found[0])
	}
}

func TestStatusFromWordPressCollapsesTheEditableStatuses(t *testing.T) {
	t.Parallel()

	cases := map[string]pagemap.Status{
		"publish": pagemap.StatusPublished,
		"draft":   pagemap.StatusExists,
		"pending": pagemap.StatusExists,
		"private": pagemap.StatusExists,
		"future":  pagemap.StatusExists,
	}

	for wordpress, want := range cases {
		if got := pagemap.StatusFromWordPress(wordpress); got != want {
			t.Errorf("StatusFromWordPress(%q) = %q, want %q", wordpress, got, want)
		}
	}
}
