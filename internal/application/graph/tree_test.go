package graph_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func supplements() []graph.EntityInput {
	return []graph.EntityInput{
		{Name: "Supplements", Kind: "hub", PrimaryKeyword: "supplements"},
		{Name: "Vitamins", Kind: "category", PrimaryKeyword: "vitamins", ParentName: "Supplements"},
		{Name: "Vitamin D", Kind: "topic", PrimaryKeyword: "vitamin d", ParentName: "Vitamins"},
		{Name: "Magnesium", Kind: "topic", PrimaryKeyword: "magnesium", ParentName: "Vitamins"},
	}
}

func TestAWholeTreeIsWrittenInOneCall(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	created, err := h.service.CreateEntities(t.Context(), graph.CreateEntitiesRequest{
		SiteID: h.siteID, Entities: supplements(),
	})
	if err != nil {
		t.Fatalf("CreateEntities: %v", err)
	}
	if len(created.Entities) != 4 {
		t.Fatalf("the batch wrote %d entities, want 4", len(created.Entities))
	}
	if len(created.Edges) != 3 {
		t.Fatalf("the batch wrote %d parent edges, want 3", len(created.Edges))
	}

	byName := make(map[string]string, len(created.Entities))
	for _, entity := range created.Entities {
		byName[entity.Name] = entity.ID
		if entity.Source != "ai" {
			t.Errorf("%s was recorded as %s, want ai", entity.Name, entity.Source)
		}
	}
	for _, edge := range created.Edges {
		if edge.Kind != "parent" || edge.Status != "approved" || edge.Weight != 1 {
			t.Fatalf("the edge %+v is not an approved parent edge", edge)
		}
	}
	if byName["Vitamin D"] == "" || byName["Supplements"] == "" {
		t.Fatalf("the tree is missing a name: %+v", byName)
	}

	loaded, err := h.service.LoadGraph(t.Context(), graph.LoadGraphRequest{SiteID: h.siteID})
	if err != nil || len(loaded.Entities) != 4 || len(loaded.Edges) != 3 {
		t.Fatalf("the stored graph is %d entities and %d edges, %v",
			len(loaded.Entities), len(loaded.Edges), err)
	}
}

func TestABatchThatCannotStandIsWrittenAtAll(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		entities []graph.EntityInput
	}{
		{
			name: "a parent nobody declares",
			entities: []graph.EntityInput{
				{Name: "Vitamin D", Kind: "topic", PrimaryKeyword: "vitamin d", ParentName: "Vitamins"},
			},
		},
		{
			name: "the same name twice",
			entities: []graph.EntityInput{
				{Name: "Vitamins", Kind: "category", PrimaryKeyword: "vitamins"},
				{Name: "vitamins", Kind: "topic", PrimaryKeyword: "vitamins b"},
			},
		},
		{
			name: "a kind the domain does not know",
			entities: []graph.EntityInput{
				{Name: "Vitamins", Kind: "chapter", PrimaryKeyword: "vitamins"},
			},
		},
		{
			name:     "nothing at all",
			entities: []graph.EntityInput{},
		},
		{
			name: "an entity that is its own parent",
			entities: []graph.EntityInput{
				{Name: "Vitamins", Kind: "category", PrimaryKeyword: "vitamins", ParentName: "Vitamins"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			_, err := h.service.CreateEntities(t.Context(), graph.CreateEntitiesRequest{
				SiteID: h.siteID, Entities: tc.entities,
			})
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("CreateEntities = %v, want an invalid batch", err)
			}

			loaded, loadErr := h.service.LoadGraph(t.Context(), graph.LoadGraphRequest{SiteID: h.siteID})
			if loadErr != nil {
				t.Fatalf("Load: %v", loadErr)
			}
			if len(loaded.Entities) != 0 || len(loaded.Edges) != 0 {
				t.Fatalf("a refused batch left %d entities and %d edges behind",
					len(loaded.Entities), len(loaded.Edges))
			}
		})
	}
}

func TestABatchHangsOffWhatTheSiteAlreadyHas(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	root := h.entity(t, "Supplements", "hub")

	created, err := h.service.CreateEntities(t.Context(), graph.CreateEntitiesRequest{
		SiteID: h.siteID,
		Entities: []graph.EntityInput{
			{Name: "Vitamins", Kind: "category", PrimaryKeyword: "vitamins", ParentName: "Supplements"},
		},
	})
	if err != nil {
		t.Fatalf("CreateEntities: %v", err)
	}
	if len(created.Edges) != 1 || created.Edges[0].ToEntityID != root.ID {
		t.Fatalf("the batch did not hang off the entity the site already had: %+v", created.Edges)
	}

	_, err = h.service.CreateEntities(t.Context(), graph.CreateEntitiesRequest{
		SiteID:   h.siteID,
		Entities: []graph.EntityInput{{Name: "Supplements", Kind: "hub", PrimaryKeyword: "supplements"}},
	})
	if !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("a name the site already carries = %v, want an invalid batch", err)
	}
}
