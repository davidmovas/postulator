package graph_test

import (
	stderrors "errors"
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	siteA = "0b6c2a4e-1f3d-4c8b-9a2e-5d7f8e9a0b1c"
	entA  = "1a1a1a1a-1a1a-4a1a-8a1a-1a1a1a1a1a1a"
	entB  = "2b2b2b2b-2b2b-4b2b-8b2b-2b2b2b2b2b2b"
	entC  = "3c3c3c3c-3c3c-4c3c-8c3c-3c3c3c3c3c3c"
	entD  = "4d4d4d4d-4d4d-4d4d-8d4d-4d4d4d4d4d4d"
	entE  = "5e5e5e5e-5e5e-4e5e-8e5e-5e5e5e5e5e5e"
)

func fieldOf(t *testing.T, err error) string {
	t.Helper()
	var kernel *errors.Error
	if !stderrors.As(err, &kernel) {
		t.Fatalf("error %v is not a kernel error", err)
	}
	field, _ := kernel.Details["field"].(string)
	return field
}

func validEntity() graph.Entity {
	at := time.Date(2026, time.September, 18, 9, 0, 0, 0, time.UTC)
	return graph.Entity{
		ID: entA, SiteID: siteA, Name: " Running Shoes ", Kind: graph.KindHub, Intent: "commercial",
		PrimaryKeyword: "running shoes", SecondaryKeywords: []string{"trail shoes", " Trail Shoes ", "", "road shoes"},
		Anchors: []graph.Anchor{{Text: "running shoes", Source: graph.AnchorUser, Weight: 1}},
		Source:  graph.SourceUser, CreatedAt: at, UpdatedAt: at,
	}
}

func TestNewEntityNormalises(t *testing.T) {
	t.Parallel()

	entity, err := graph.NewEntity(validEntity())
	if err != nil {
		t.Fatalf("NewEntity: %v", err)
	}
	if entity.Name != "Running Shoes" {
		t.Errorf("Name = %q, want trimmed", entity.Name)
	}
	if !slices.Equal(entity.SecondaryKeywords, []string{"trail shoes", "road shoes"}) {
		t.Errorf("SecondaryKeywords = %v, want trimmed and deduplicated", entity.SecondaryKeywords)
	}
}

func TestNewEntityRejects(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		mutate func(*graph.Entity)
		field  string
	}{
		{name: "no id", mutate: func(e *graph.Entity) { e.ID = "" }, field: "id"},
		{name: "no site", mutate: func(e *graph.Entity) { e.SiteID = "" }, field: "siteId"},
		{name: "blank name", mutate: func(e *graph.Entity) { e.Name = " " }, field: "name"},
		{name: "unknown kind", mutate: func(e *graph.Entity) { e.Kind = "planet" }, field: "kind"},
		{name: "unknown source", mutate: func(e *graph.Entity) { e.Source = "wind" }, field: "source"},
		{name: "negative score", mutate: func(e *graph.Entity) { e.Score = -1 }, field: "score"},
		{name: "empty canonical", mutate: func(e *graph.Entity) { empty := ""; e.CanonicalPageID = &empty }, field: "canonicalPageId"},
		{name: "anchor blank", mutate: func(e *graph.Entity) { e.Anchors = []graph.Anchor{{Text: " ", Source: graph.AnchorUser}} }, field: "anchors[0].text"},
		{name: "anchor source", mutate: func(e *graph.Entity) { e.Anchors = []graph.Anchor{{Text: "a", Source: "bot"}} }, field: "anchors[0].source"},
		{name: "anchor weight", mutate: func(e *graph.Entity) {
			e.Anchors = []graph.Anchor{{Text: "a", Source: graph.AnchorAI, Weight: 1.5}}
		}, field: "anchors[0].weight"},
		{name: "anchor repeated", mutate: func(e *graph.Entity) {
			e.Anchors = []graph.Anchor{{Text: "Shoes", Source: graph.AnchorUser}, {Text: "shoes", Source: graph.AnchorAI}}
		}, field: "anchors[1].text"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			entity := validEntity()
			tc.mutate(&entity)
			_, err := graph.NewEntity(entity)
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("code = %q, want INVALID", errors.CodeOf(err))
			}
			if got := fieldOf(t, err); got != tc.field {
				t.Errorf("field = %q, want %q", got, tc.field)
			}
		})
	}
}

func TestNewAnchorsKeepsOrderAndTrims(t *testing.T) {
	t.Parallel()

	anchors, err := graph.NewAnchors([]graph.Anchor{{Text: " b ", Source: graph.AnchorAI, Weight: 0.5}, {Text: "a", Source: graph.AnchorUser, Weight: 1}})
	if err != nil {
		t.Fatalf("NewAnchors: %v", err)
	}
	if len(anchors) != 2 || anchors[0].Text != "b" || anchors[1].Text != "a" {
		t.Fatalf("anchors = %+v", anchors)
	}

	empty, err := graph.NewAnchors(nil)
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("NewAnchors(nil) = %v, %v; want an empty slice", empty, err)
	}
}

func TestEnums(t *testing.T) {
	t.Parallel()

	for _, kind := range []graph.Kind{graph.KindHub, graph.KindProduct, graph.KindTopic, graph.KindCategory, graph.KindCustom} {
		if !kind.Valid() {
			t.Errorf("%q must be valid", kind)
		}
	}
	for _, source := range []graph.Source{graph.SourceImport, graph.SourceUser, graph.SourceAI} {
		if !source.Valid() {
			t.Errorf("%q must be valid", source)
		}
	}
	for _, status := range []graph.EdgeStatus{graph.StatusApproved, graph.StatusProposed, graph.StatusRejected} {
		if !status.Valid() {
			t.Errorf("%q must be valid", status)
		}
	}
	if graph.Kind("x").Valid() || graph.Source("x").Valid() || graph.AnchorSource("x").Valid() || graph.EdgeKind("x").Valid() || graph.EdgeStatus("x").Valid() {
		t.Error("unknown enum values must be invalid")
	}
	if !graph.EntitySortCreatedAt.Valid() || !graph.EntitySortName.Valid() || graph.EntitySort("x").Valid() {
		t.Error("entity sort validity is wrong")
	}
}
