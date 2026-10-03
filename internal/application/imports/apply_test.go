package imports_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestApplyWritesTheGraphAndThePageMapOnce(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	path := h.file(t, "graph.csv", graphSheet)
	got := h.apply(t, path, graphMapping(h))

	want := imports.Counts{EntitiesCreated: 3, EdgesCreated: 3, PagesCreated: 2}
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
		if (entities[i].CanonicalPageID == nil) != (entities[i].Name == "Shop") {
			t.Fatalf("%s canonical page = %v; only the entity of the root row has none", entities[i].Name, entities[i].CanonicalPageID)
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
	if len(pages) != 2 {
		t.Fatalf("pages = %d, want the two sections and no root", len(pages))
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
	if len(h.entities(t)) != 3 || len(h.edges(t)) != 3 || len(h.pages(t)) != 2 {
		t.Fatal("the second apply changed the site")
	}
	for i := range again.Report.Entities {
		if again.Report.Entities[i].Action != string(imports.ActionSkip) {
			t.Fatalf("entity = %+v, want a skip", again.Report.Entities[i])
		}
	}
	for i := range again.Report.Edges {
		if again.Report.Edges[i].Action != string(imports.ActionSkip) {
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
	if !slices.Equal(entities[0].Keywords.Texts(), []string{"hosting", "servers"}) || len(entities[0].Anchors) != 2 {
		t.Fatalf("entity = %+v", entities[0])
	}
	pages := h.pages(t)
	if len(pages) != 1 {
		t.Fatalf("pages = %d, want the imported page alone", len(pages))
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

	if got.Counts.PagesCreated != 2 {
		t.Fatalf("pages created = %d, want the merged page plus one intermediate", got.Counts.PagesCreated)
	}
	if got.Counts.EntitiesCreated != 2 {
		t.Fatalf("entities created = %d, want the named one and the intermediate's", got.Counts.EntitiesCreated)
	}
	byName := make(map[string]graph.Entity)
	for _, stored := range h.entities(t) {
		byName[stored.Name] = stored
	}
	hosting, shop := byName["Hosting"], byName["Shop"]
	if !slices.Equal(hosting.Keywords.Texts(), []string{"hosting", "servers"}) {
		t.Fatalf("keywords = %v, want both rows", hosting.Keywords.Texts())
	}
	if shop.ID == "" || hosting.ScopeID == nil || *hosting.ScopeID != shop.ID {
		t.Fatalf("Hosting sits under %v, want the intermediate Shop", hosting.ScopeID)
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
	mapping := graphMapping(h)
	mapping.Options.AnchorSeparator = ";"
	_, err := h.service.Apply(t.Context(), imports.ApplyRequest{
		SiteID:  h.siteID,
		Path:    h.file(t, "graph.csv", graphSheet),
		Mapping: mapping,
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
	if listed.Mappings[0].Options.AnchorSeparator != ";" {
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

func TestApplySavesTheMappingUnderTheSameNameTwice(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	path := h.file(t, "graph.csv", graphSheet)
	req := imports.ApplyRequest{
		SiteID:  h.siteID,
		Path:    path,
		Mapping: graphMapping(h),
		Options: imports.ApplyOptions{SaveMappingAs: "the client sheet"},
	}
	if _, err := h.service.Apply(t.Context(), req); err != nil {
		t.Fatalf("the first Apply: %v", err)
	}
	if _, err := h.service.Apply(t.Context(), req); err != nil {
		t.Fatalf("the second Apply: %v", err)
	}

	listed, err := h.service.ListMappings(t.Context(), imports.ListMappingsRequest{SiteID: h.siteID})
	if err != nil {
		t.Fatalf("ListMappings: %v", err)
	}
	if len(listed.Mappings) != 1 {
		t.Fatalf("mappings = %+v, want the one name reused", listed.Mappings)
	}
	if len(h.entities(t)) != 3 || len(h.pages(t)) != 2 {
		t.Fatal("the second apply rolled the site back")
	}
}

func TestApplyLinksEveryCreatedPageToItsParentPath(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	path := h.file(t, "tree.csv", "path,title\n/menu/,Menu\n/menu/mains/,Mains\n/menu/mains/steaks/,Steaks\n")

	applied := h.apply(t, path, h.mapping(map[string]string{"path": "path", "title": "title"}))
	if len(applied.Report.Errors) != 0 {
		t.Fatalf("errors = %+v", applied.Report.Errors)
	}

	pages := h.pages(t)

	byPath := make(map[string]pagemap.Page, len(pages))
	for i := range pages {
		byPath[pages[i].Path] = pages[i]
	}

	for child, parent := range map[string]string{
		"/menu/mains/":        "/menu/",
		"/menu/mains/steaks/": "/menu/mains/",
	} {
		page, ok := byPath[child]
		if !ok {
			t.Fatalf("the import created no %s", child)
		}
		if page.ParentPageID == nil || *page.ParentPageID != byPath[parent].ID {
			t.Fatalf("%s carries the parent %v, want %s", child, page.ParentPageID, parent)
		}
	}
	if top := byPath["/menu/"]; top.ParentPageID != nil {
		t.Fatalf("the top of the tree carries a parent: %+v", top)
	}
	if root, ok := byPath["/"]; ok {
		t.Fatalf("the import planned the root of the site: %+v", root)
	}
}

func TestAStoreItemIsNeverThePageParentOfARow(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	product := h.storeProduct(t, "mak-liquid", "Mak Liquid", 501)
	h.apply(t, h.file(t, "guide.csv", "path,title\n"+product.Path+"guide/,Guide\n/menu/,Menu\n/menu/mains/,Mains\n"),
		h.mapping(map[string]string{"path": "path", "title": "title"}))

	byPath := make(map[string]pagemap.Page)
	for _, stored := range h.pages(t) {
		byPath[stored.Path] = stored
	}
	if guide := byPath[product.Path+"guide/"]; guide.ParentPageID != nil {
		t.Fatalf("the guide under the product's address sits under %s, want no parent", *guide.ParentPageID)
	}
	if mains := byPath["/menu/mains/"]; mains.ParentPageID == nil || *mains.ParentPageID != byPath["/menu/"].ID {
		t.Fatalf("/menu/mains/ sits under %v, want /menu/", mains.ParentPageID)
	}
}

func TestApplyPutsAnEntityUnderTheParentItsRowNames(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.apply(t, h.file(t, "tree.csv", "path,entity,parent entity\n/peptides/,Peptides,\n/peptides/bpc/,BPC-157,Peptides\n"),
		h.mapping(map[string]string{
			string(importmap.FieldPath):         "path",
			string(importmap.FieldEntity):       "entity",
			string(importmap.FieldParentEntity): "parent entity",
		}))

	byName := make(map[string]graph.Entity)
	for _, stored := range h.entities(t) {
		byName[stored.Name] = stored
	}
	child, parent := byName["BPC-157"], byName["Peptides"]
	if child.ScopeID == nil || *child.ScopeID != parent.ID {
		t.Fatalf("BPC-157 sits under %v, want Peptides (%s)", child.ScopeID, parent.ID)
	}
	if parent.ScopeID != nil {
		t.Fatalf("Peptides sits under %v, want the top", *parent.ScopeID)
	}
}

func TestApplyKeepsTheKeywordsOfARowOnItsPage(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	sheet := "path,title,primary keyword,keywords\n" +
		"/shoes/trail/,Trail shoes,trail running shoes,\"trail shoes (900); best trail shoes (2,400), trail footwear (12x)\"\n" +
		"/shoes/road/,Road shoes,,\n"
	mapping := h.mapping(map[string]string{
		string(importmap.FieldPath):           "path",
		string(importmap.FieldTitle):          "title",
		string(importmap.FieldPrimaryKeyword): "primary keyword",
		string(importmap.FieldKeywords):       "keywords",
	})

	report := h.preview(t, h.file(t, "keywords.csv", sheet), mapping)
	previewed, ok := page(report, "/shoes/trail/")
	if !ok || len(previewed.Keywords) != 4 || previewed.Keywords[0].Text != "best trail shoes" ||
		previewed.Keywords[0].Volume == nil || *previewed.Keywords[0].Volume != 2400 {
		t.Fatalf("the preview drops the keywords of the row or their order: %+v", previewed)
	}
	if !hasFinding(report.Warnings, imports.CodeBadVolume) {
		t.Fatalf("a volume that cannot be read is not reported: %+v", report.Warnings)
	}
	if len(report.Errors) != 0 {
		t.Fatalf("a volume that cannot be read blocks the import: %+v", report.Errors)
	}

	h.apply(t, h.file(t, "keywords.csv", sheet), mapping)

	pathOf := func() map[string]pagemap.Page {
		byPath := make(map[string]pagemap.Page)
		for _, stored := range h.pages(t) {
			byPath[stored.Path] = stored
		}
		return byPath
	}
	trail := pathOf()["/shoes/trail/"]
	if want := []string{"best trail shoes", "trail shoes", "trail running shoes", "trail footwear"}; !slices.Equal(trail.Keywords.Texts(), want) {
		t.Fatalf("keywords = %v, want %v", trail.Keywords.Texts(), want)
	}
	if road := pathOf()["/shoes/road/"]; len(road.Keywords) != 0 {
		t.Fatalf("a row without keywords gave its page some: %+v", road)
	}
	stored := h.entities(t)
	names := make([]string, 0, len(stored))
	for _, held := range stored {
		names = append(names, held.Name)
		if held.Name == "Trail shoes" && !held.Keywords.Equal(trail.Keywords) {
			t.Fatalf("the entity of the row lost its keywords: %+v", held.Keywords)
		}
	}
	slices.Sort(names)
	if !slices.Equal(names, []string{"Road shoes", "Shoes", "Trail shoes"}) {
		t.Fatalf("entities = %v, want one per row and one for the intermediate", names)
	}

	again := "path,title,primary keyword,keywords\n/shoes/trail/,Trail shoes,,\"trail shoes (3000), trail boots\"\n"
	h.apply(t, h.file(t, "again.csv", again), mapping)
	trail = pathOf()["/shoes/trail/"]
	if want := []string{"trail shoes", "best trail shoes", "trail running shoes", "trail footwear", "trail boots"}; !slices.Equal(trail.Keywords.Texts(), want) {
		t.Fatalf("after a second import keywords = %v, want the new volume to win and nothing dropped: %v", trail.Keywords.Texts(), want)
	}

	blank := "path,title,primary keyword,keywords\n/shoes/trail/,Trail shoes,,\n"
	h.apply(t, h.file(t, "blank.csv", blank), mapping)
	if kept := pathOf()["/shoes/trail/"]; !kept.Keywords.Equal(trail.Keywords) {
		t.Fatalf("an empty keywords cell changed the page: %v, want %v", kept.Keywords.Texts(), trail.Keywords.Texts())
	}
	if got := len(h.entities(t)); got != 3 {
		t.Fatalf("entities after the imports that name none = %d, want the page's own entity reused", got)
	}
}

func hasFinding(findings []imports.Finding, code imports.FindingCode) bool {
	for _, finding := range findings {
		if finding.Code == string(code) {
			return true
		}
	}
	return false
}
