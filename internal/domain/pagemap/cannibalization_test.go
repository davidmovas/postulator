package pagemap_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func entityWith(id, keyword string, canonical *string) graph.Entity {
	return graph.Entity{ID: id, SiteID: siteA, Name: id, Kind: graph.KindTopic, PrimaryKeyword: keyword, CanonicalPageID: canonical, Source: graph.SourceUser, CreatedAt: stamp, UpdatedAt: stamp}
}

func TestCannibalization(t *testing.T) {
	t.Parallel()

	index := pagemap.NewIndex([]pagemap.Page{
		page(pageA, "/shoes/", ptr(entA)),
		page(pageB, "/boots/", ptr(entB)),
	})
	owner := entityWith(entA, "Running Shoes", ptr(pageA))
	rival := entityWith(entB, "running shoes", ptr(pageB))
	g, err := graph.New([]graph.Entity{owner, rival}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	cases := []struct {
		name      string
		candidate pagemap.Page
		entity    graph.Entity
		reasons   []pagemap.Reason
		pages     []string
	}{
		{
			name:      "fresh page for a fresh entity",
			candidate: page(pageC, "/sandals/", nil),
			entity:    entityWith("e-new", "sandals", nil),
		},
		{
			name:      "unmapped page needs only a free path",
			candidate: page(pageC, "/sandals/", nil),
			entity:    graph.Entity{},
		},
		{
			name:      "path already taken",
			candidate: page(pageC, "/Shoes", nil),
			entity:    graph.Entity{},
			reasons:   []pagemap.Reason{pagemap.ReasonPathConflict},
			pages:     []string{pageA},
		},
		{
			name:      "entity already has a canonical page",
			candidate: page(pageC, "/trainers/", ptr(entA)),
			entity:    owner,
			reasons:   []pagemap.Reason{pagemap.ReasonSameEntityCanonical, pagemap.ReasonSamePrimaryKeyword},
			pages:     []string{pageA, pageB},
		},
		{
			name:      "the canonical page itself is not its own rival",
			candidate: page(pageA, "/shoes/", ptr(entA)),
			entity:    entityWith(entA, "unique phrase", ptr(pageA)),
		},
		{
			name:      "another entity owns the keyword",
			candidate: page(pageC, "/trainers/", ptr("e-new")),
			entity:    entityWith("e-new", "RUNNING shoes", nil),
			reasons:   []pagemap.Reason{pagemap.ReasonSamePrimaryKeyword, pagemap.ReasonSamePrimaryKeyword},
			pages:     []string{pageA, pageB},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			verdict := pagemap.Cannibalization(tc.candidate, tc.entity, index, g)
			if verdict.Allowed != (len(tc.reasons) == 0) {
				t.Fatalf("Allowed = %v, evidence %+v", verdict.Allowed, verdict.Evidence)
			}
			if len(verdict.Evidence) != len(tc.reasons) {
				t.Fatalf("evidence = %+v, want %d entries", verdict.Evidence, len(tc.reasons))
			}
			for i, reason := range tc.reasons {
				if verdict.Evidence[i].Reason != reason || verdict.Evidence[i].PageID != tc.pages[i] {
					t.Errorf("evidence[%d] = %+v, want %s on %s", i, verdict.Evidence[i], reason, tc.pages[i])
				}
				if verdict.Evidence[i].Path == "" {
					t.Errorf("evidence[%d] carries no path", i)
				}
			}
		})
	}
}
