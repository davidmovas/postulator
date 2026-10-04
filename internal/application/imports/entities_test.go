package imports_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/keyword"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func TestAnImportAddsTheSheetAnchorsAndKeepsTheOnesAnEntityCarries(t *testing.T) {
	t.Parallel()

	carried := []graph.Anchor{
		{Text: "hosting plans", Source: graph.AnchorAI, Weight: 0.6},
		{Text: "cheap hosting", Source: graph.AnchorUser, Weight: 0.4},
	}
	cases := []struct {
		name   string
		row    string
		action imports.Action
		want   []graph.Anchor
	}{
		{
			name:   "a new anchor is added after the carried ones",
			row:    "Hosting,hosting,Hosting Plans|cheap hosting|managed hosting",
			action: imports.ActionUpdate,
			want:   append(slices.Clone(carried), graph.Anchor{Text: "managed hosting", Source: graph.AnchorUser, Weight: 1}),
		},
		{
			name:   "a new keyword leaves the carried anchors as they are",
			row:    "Hosting,servers,hosting plans",
			action: imports.ActionUpdate,
			want:   carried,
		},
		{
			name:   "the carried anchors alone change nothing",
			row:    "Hosting,hosting,CHEAP HOSTING|hosting plans",
			action: imports.ActionSkip,
			want:   carried,
		},
		{
			name:   "an empty cell erases nothing",
			row:    "Hosting,,",
			action: imports.ActionSkip,
			want:   carried,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			if err := sqlite.NewEntityRepo(h.store).Insert(t.Context(), graph.Entity{
				ID: id.New(), SiteID: h.siteID, Name: "Hosting", Kind: graph.KindTopic, Keywords: keyword.Of("hosting"),
				Anchors: carried, Source: graph.SourceUser, CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
			}); err != nil {
				t.Fatalf("insert the entity: %v", err)
			}
			path := h.file(t, "anchors.csv", "entity,keywords,anchors\n"+tc.row+"\n")
			mapping := h.mapping(map[string]string{
				string(importmap.FieldEntity):   "entity",
				string(importmap.FieldKeywords): "keywords",
				string(importmap.FieldAnchors):  "anchors",
			})

			for pass, action := range []imports.Action{tc.action, imports.ActionSkip} {
				got := h.apply(t, path, mapping)
				reported, found := entity(got.Report, "Hosting")
				if !found || reported.Action != string(action) {
					t.Fatalf("pass %d: entity = %+v, want %s", pass, reported, action)
				}
				if updated := got.Counts.EntitiesUpdated; (action == imports.ActionSkip) != (updated == 0) {
					t.Fatalf("pass %d: entities updated = %d, want the %s alone", pass, updated, action)
				}
				stored := h.entities(t)
				if len(stored) != 1 || !slices.Equal(stored[0].Anchors, tc.want) {
					t.Fatalf("pass %d: entities = %+v, want one carrying %+v", pass, stored, tc.want)
				}
			}
		})
	}
}
