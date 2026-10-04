package pagemap_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/keyword"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func TestKeywordsAreThePagesOwnAndItsEntitysOtherwise(t *testing.T) {
	t.Parallel()

	entity := graph.Entity{ID: entA, Keywords: keyword.New([]keyword.Keyword{{Text: "bpc 157", Volume: new(12000)}, {Text: "bpc dosage"}})}

	cases := []struct {
		name   string
		page   keyword.List
		entity graph.Entity
		want   []string
	}{
		{
			name:   "a page with keywords of its own answers with them alone",
			page:   keyword.New([]keyword.Keyword{{Text: "bpc 157 liquid", Volume: new(900)}, {Text: "liquid bpc"}}),
			entity: entity,
			want:   []string{"bpc 157 liquid", "liquid bpc"},
		},
		{name: "a page without keywords follows its entity", page: keyword.Of(), entity: entity, want: []string{"bpc 157", "bpc dosage"}},
		{name: "a page that was never given a list follows its entity", page: nil, entity: entity, want: []string{"bpc 157", "bpc dosage"}},
		{name: "a page with keywords and no entity keeps them", page: keyword.Of("about us"), entity: graph.Entity{}, want: []string{"about us"}},
		{name: "neither carries a keyword", page: nil, entity: graph.Entity{}, want: []string{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := pagemap.Keywords(pagemap.Page{ID: pageA, Keywords: tc.page}, tc.entity)
			if got == nil || !slices.Equal(got.Texts(), tc.want) {
				t.Fatalf("Keywords = %#v, want %v", got, tc.want)
			}
		})
	}
}

func TestKeywordsKeepTheVolumesOfTheListTheyCameFrom(t *testing.T) {
	t.Parallel()

	page := pagemap.Page{Keywords: keyword.New([]keyword.Keyword{{Text: "bpc 157 liquid", Volume: new(900)}})}
	got := pagemap.Keywords(page, graph.Entity{Keywords: keyword.Of("bpc 157")})
	if len(got) != 1 || got[0].Volume == nil || *got[0].Volume != 900 {
		t.Fatalf("Keywords = %#v, want the volume of the page's keyword", got)
	}
}
