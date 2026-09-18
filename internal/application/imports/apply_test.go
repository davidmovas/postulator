package imports_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestApplyWritesTheGraphAndThePageMapOnce(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	path := h.file(t, "graph.csv", graphSheet)
	got := h.apply(t, path, graphMapping(h))

	want := imports.Counts{EntitiesCreated: 3, EdgesCreated: 3, PagesCreated: 3}
	if got.Counts != want {
		t.Fatalf("counts = %+v, want %+v", got.Counts, want)
	}

	entities := h.entities(t)
	if len(entities) != 3 {
		t.Fatalf("entities = %d", len(entities))
	}
	for i := range entities {
		if entities[i].Source != graph.SourceImport {
			t.Fatalf("%s source = %s", entities[i].Name, entities[i].Source)
		}
		if entities[i].CanonicalPageID == nil {
			t.Fatalf("%s has no canonical page", entities[i].Name)
		}
	}

	edges := h.edges(t)
	parents, related := 0, 0
	for i := range edges {
		if edges[i].Status != graph.StatusApproved || edges[i].Source != graph.SourceImport {
			t.Fatalf("edge = %+v", edges[i])
		}
		if edges[i].Kind == graph.EdgeParent {
			parents++
		} else {
			related++
		}
	}
	if parents != 2 || related != 1 {
		t.Fatalf("parent edges = %d, related edges = %d", parents, related)
	}

	pages := h.pages(t)
	if len(pages) != 3 {
		t.Fatalf("pages = %d", len(pages))
	}
	for i := range pages {
		if pages[i].EntityID == nil {
			t.Fatalf("%s is not mapped to an entity", pages[i].Path)
		}
		if pages[i].WPID != nil || pages[i].ContentHash != "" {
			t.Fatalf("%s carries wordpress state", pages[i].Path)
		}
	}

	published := h.recorder.Events()
	if len(published) != 2 || published[0].Type != events.GraphChanged || published[1].Type != events.PagesChanged {
		t.Fatalf("events = %+v", published)
	}
}

func TestApplyIsIdempotent(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	path := h.file(t, "graph.csv", graphSheet)
	h.apply(t, path, graphMapping(h))
	h.recorder.Reset()

	again := h.apply(t, path, graphMapping(h))
	if again.Counts != (imports.Counts{}) {
		t.Fatalf("the second apply = %+v, want nothing written", again.Counts)
	}
	if len(h.entities(t)) != 3 || len(h.edges(t)) != 3 || len(h.pages(t)) != 3 {
		t.Fatal("the second apply changed the site")
	}
	for i := range again.Report.Entities {
		if again.Report.Entities[i].Action != imports.ActionSkip {
			t.Fatalf("entity = %+v, want a skip", again.Report.Entities[i])
		}
	}
	for i := range again.Report.Edges {
		if again.Report.Edges[i].Action != imports.ActionSkip {
			t.Fatalf("edge = %+v, want a skip", again.Report.Edges[i])
		}
	}
}

func TestApplyMergesIntoWhatTheSiteAlreadyHolds(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	mapping := h.mapping(map[string]string{
		string(importmap.FieldPath):     "path",
		string(importmap.FieldTitle):    "title",
		string(importmap.FieldEntity):   "entity",
		string(importmap.FieldKeywords): "keywords",
		string(importmap.FieldAnchors):  "anchors",
	})

	h.apply(t, h.file(t, "first.csv", "path,title,entity,keywords,anchors\n/hosting/,Hosting,Hosting,hosting,hosting plans\n"), mapping)
	got := h.apply(t, h.file(t, "second.csv", "path,title,entity,keywords,anchors\n/hosting/,,Hosting,servers,cheap hosting\n"), mapping)

	if got.Counts.EntitiesUpdated != 1 || got.Counts.EntitiesCreated != 0 || got.Counts.PagesCreated != 0 {
		t.Fatalf("counts = %+v", got.Counts)
	}

	entities := h.entities(t)
	if len(entities) != 1 {
		t.Fatalf("entities = %d, want the case-insensitive merge", len(entities))
	}
	if len(entities[0].SecondaryKeywords) != 2 || len(entities[0].Anchors) != 2 {
		t.Fatalf("entity = %+v", entities[0])
	}
	pages := h.pages(t)
	if len(pages) != 2 {
		t.Fatalf("pages = %d, want the imported page and the root", len(pages))
	}
	for i := range pages {
		if pages[i].Path == "/hosting/" && pages[i].Title != "Hosting" {
			t.Fatalf("page = %+v, want the title kept", pages[i])
		}
	}
}

func TestApplyMergesRepeatedPathsAndFillsTheGaps(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	got := h.apply(t, h.file(t, "dupes.csv",
		"path,title,entity,keywords\n/shop/hosting/,Hosting,Hosting,hosting\n/shop/HOSTING/,,Hosting,servers\n"),
		h.mapping(map[string]string{
			string(importmap.FieldPath):     "path",
			string(importmap.FieldTitle):    "title",
			string(importmap.FieldEntity):   "entity",
			string(importmap.FieldKeywords): "keywords",
		}))

	if got.Counts.PagesCreated != 3 {
		t.Fatalf("pages created = %d, want the merged page plus two intermediates", got.Counts.PagesCreated)
	}
	if got.Counts.EntitiesCreated != 1 {
		t.Fatalf("entities created = %d", got.Counts.EntitiesCreated)
	}
	if entities := h.entities(t); len(entities[0].SecondaryKeywords) != 2 {
		t.Fatalf("keywords = %v, want both rows", entities[0].SecondaryKeywords)
	}
}

func TestApplyRefusesAPreviewThatCarriesErrors(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	_, err := h.service.Apply(t.Context(), imports.ApplyRequest{
		SiteID: h.siteID,
		Path:   h.file(t, "broken.csv", "path,entity,parent\n/a/,Alpha,Absent\n"),
		Mapping: h.mapping(map[string]string{
			string(importmap.FieldPath):         "path",
			string(importmap.FieldEntity):       "entity",
			string(importmap.FieldParentEntity): "parent",
		}),
	})
	if !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Apply = %v, want an invalid error", err)
	}
	if len(h.entities(t)) != 0 || len(h.pages(t)) != 0 {
		t.Fatal("the refused apply wrote to the database")
	}
	if len(h.recorder.Events()) != 0 {
		t.Fatalf("the refused apply published %+v", h.recorder.Events())
	}
}

func TestApplyLeavesTheCanonicalPageAloneWhenTheEntityOwnsSeveral(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.apply(t, h.file(t, "two.csv", "path,entity\n/hosting/,Hosting\n/hosting/vps/,Hosting\n"),
		h.mapping(map[string]string{string(importmap.FieldPath): "path", string(importmap.FieldEntity): "entity"}))

	entities := h.entities(t)
	if len(entities) != 1 {
		t.Fatalf("entities = %d", len(entities))
	}
	if entities[0].CanonicalPageID != nil {
		t.Fatalf("canonical = %v, want none while two pages claim the entity", *entities[0].CanonicalPageID)
	}
}

func TestApplySavesTheMappingWhenAsked(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	_, err := h.service.Apply(t.Context(), imports.ApplyRequest{
		SiteID:  h.siteID,
		Path:    h.file(t, "graph.csv", graphSheet),
		Mapping: graphMapping(h),
		Options: imports.ApplyOptions{SaveMappingAs: "the client sheet"},
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	listed, err := h.service.ListMappings(t.Context(), imports.ListMappingsRequest{SiteID: h.siteID})
	if err != nil {
		t.Fatalf("ListMappings: %v", err)
	}
	if len(listed.Mappings) != 1 || listed.Mappings[0].Name != "the client sheet" {
		t.Fatalf("mappings = %+v", listed.Mappings)
	}
	if listed.Mappings[0].Options.KeywordSeparator != ";" {
		t.Fatalf("options = %+v", listed.Mappings[0].Options)
	}
}

func TestMappingsAreSavedListedAndDeleted(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	saved, err := h.service.SaveMapping(t.Context(), imports.SaveMappingRequest{Mapping: graphMapping(h)})
	if err != nil {
		t.Fatalf("SaveMapping: %v", err)
	}
	if saved.Mapping.ID == "" || saved.Mapping.Options.AnchorSeparator != "|" {
		t.Fatalf("saved = %+v", saved.Mapping)
	}

	path := h.file(t, "graph.csv", graphSheet)
	seen, err := h.service.Inspect(t.Context(), imports.InspectRequest{SiteID: h.siteID, Path: path})
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if len(seen.Saved) != 1 || seen.Saved[0].ID != saved.Mapping.ID {
		t.Fatalf("saved mappings = %+v", seen.Saved)
	}

	if _, err = h.service.DeleteMapping(t.Context(), imports.DeleteMappingRequest{ID: saved.Mapping.ID}); err != nil {
		t.Fatalf("DeleteMapping: %v", err)
	}
	listed, err := h.service.ListMappings(t.Context(), imports.ListMappingsRequest{SiteID: h.siteID})
	if err != nil {
		t.Fatalf("ListMappings: %v", err)
	}
	if len(listed.Mappings) != 0 {
		t.Fatalf("mappings = %+v", listed.Mappings)
	}
	if _, err = h.service.DeleteMapping(t.Context(), imports.DeleteMappingRequest{}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("DeleteMapping without an id = %v", err)
	}
}

func TestSaveMappingRefusesWhatCannotBeImported(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	broken := h.mapping(map[string]string{string(importmap.FieldTitle): "title"})
	if _, err := h.service.SaveMapping(t.Context(), imports.SaveMappingRequest{Mapping: broken}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("SaveMapping = %v, want an invalid error", err)
	}
}
