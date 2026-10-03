package imports_test

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func (h harness) sheet(t *testing.T, path, name string) imports.Mapping {
	t.Helper()

	seen, err := h.service.Inspect(t.Context(), imports.InspectRequest{SiteID: h.siteID, Path: path, Sheets: []string{name}})
	if err != nil {
		t.Fatalf("Inspect %s: %v", name, err)
	}
	mapping := seen.Detected
	mapping.Name = name
	mapping.Options.Sheets = []string{name}
	return mapping
}

func ignored(report imports.PreviewReport) []string {
	out := make([]string, 0)
	for _, column := range report.Columns {
		if column.Use == "ignored" {
			out = append(out, column.Header)
		}
	}
	return out
}

func clean(t *testing.T, name string, applied imports.ApplyResponse) {
	t.Helper()
	if len(applied.Report.Errors) != 0 {
		t.Fatalf("the %s sheet reported %+v", name, applied.Report.Errors)
	}
}

func TestTheClientWorkbookImportsSheetBySheet(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	path := filepath.Join("..", "..", "..", "samples", "client-sheets.xlsx")

	groups := h.sheet(t, path, "Groups")
	first := h.apply(t, path, groups)
	clean(t, "Groups", first)
	if first.Counts.EntitiesCreated != 8 || first.Counts.PagesCreated != 8 {
		t.Fatalf("the group sheet wrote %+v, want eight entities and eight pages", first.Counts)
	}
	if left := ignored(first.Report); len(left) != 0 {
		t.Fatalf("the group sheet ignored %v", left)
	}
	if orphaned := findings(first.Report.Warnings, imports.CodeGroupWithoutPage); len(orphaned) != 0 {
		t.Fatalf("a group of the group sheet found no page: %+v", orphaned)
	}
	liquids := entitiesNamed(first.Report, "Liquid")
	parents := make([]string, 0, len(liquids))
	for _, liquid := range liquids {
		parents = append(parents, liquid.Parent)
	}
	slices.Sort(parents)
	if !slices.Equal(parents, []string{"BPC-157", "TB-500"}) {
		t.Fatalf("the Liquids sit under %v", parents)
	}
	if bpc, _ := page(first.Report, "/peptides/bpc-157/"); len(bpc.Keywords) != 3 || bpc.Keywords[0].Text != "bpc 157" ||
		bpc.Keywords[1].Volume == nil || *bpc.Keywords[1].Volume != 5400 {
		t.Fatalf("the keywords of BPC-157 = %+v", bpc.Keywords)
	}
	for name, got := range h.placements(t) {
		want := placed{kind: graph.KindCategory, category: true}
		switch name {
		case "Peptides":
			want = placed{kind: graph.KindHub}
		case "Storing peptides":
			want = placed{kind: graph.KindTopic}
		}
		if got != want {
			t.Errorf("after the group sheet %s = %+v, want %+v", name, got, want)
		}
	}

	catalog := h.apply(t, path, h.sheet(t, path, "Catalog"))
	clean(t, "Catalog", catalog)
	if catalog.Counts.EntitiesCreated != 7 {
		t.Fatalf("the catalog wrote %+v, want four products, two groups and the shop", catalog.Counts)
	}
	if orphaned := findings(catalog.Report.Warnings, imports.CodeGroupWithoutPage); len(orphaned) != 2 {
		t.Fatalf("groups without a page = %+v, want Blends and Recovery", orphaned)
	}
	if !hasEdge(catalog.Report, "BPC-157 10 mg vial", "BPC-157", string(graph.EdgeParent)) {
		t.Fatalf("edges = %+v, want the product under its subcategory", catalog.Report.Edges)
	}

	wide := h.apply(t, path, h.sheet(t, path, "Entities"))
	clean(t, "Entities", wide)
	if wide.Counts.EntitiesCreated != 1 {
		t.Fatalf("the wide sheet wrote %+v, want Canada alone created", wide.Counts)
	}
	if canada, found := entity(wide.Report, "Peptides in Canada"); !found || canada.Kind != string(graph.KindCustom) ||
		canada.Parent != "Peptides" {
		t.Fatalf("Canada = %+v, want a geographic entity under Peptides", canada)
	}
	if bpc, _ := entity(wide.Report, "BPC-157"); bpc.Kind != string(graph.KindProduct) {
		t.Fatalf("BPC-157 = %+v, want the product its entity level says", bpc)
	}
	if left := ignored(wide.Report); !slices.Equal(left, []string{"Entity ID", "Primary Entity", "Entity Type"}) {
		t.Fatalf("the wide sheet ignored %v", left)
	}
	want := []pagemap.Note{{Label: "Intent Owner", Text: "Commercial Product"}, {Label: "Notes", Text: "Sold as a 10 ml vial"}}
	if got := notesAt(t, h, "/peptides/bpc-157/"); !slices.Equal(got, want) {
		t.Fatalf("the notes of BPC-157 = %+v, want %+v", got, want)
	}

	variations := h.apply(t, path, h.sheet(t, path, "Variations"))
	clean(t, "Variations", variations)
	if variations.Counts.EntitiesCreated != 2 {
		t.Fatalf("the variation sheet wrote %+v, want the capsules of BPC-157 and the nasal spray", variations.Counts)
	}
	if !hasEdge(variations.Report, "Capsules", "BPC-157", string(graph.EdgeParent)) {
		t.Fatalf("edges = %+v, want the capsules under BPC-157", variations.Report.Edges)
	}
	if left := ignored(variations.Report); !slices.Equal(left, []string{"Entity?"}) {
		t.Fatalf("the variation sheet ignored %v", left)
	}
	for _, held := range h.entities(t) {
		if held.Name == "NO" || held.Name == "Yes" || held.Name == "No" {
			t.Fatalf("the Entity? column made an entity %q", held.Name)
		}
	}

	again := h.apply(t, path, groups)
	if again.Counts.EntitiesCreated != 0 || again.Counts.PagesCreated != 0 || again.Counts.EdgesCreated != 0 {
		t.Fatalf("the group sheet imported again wrote %+v", again.Counts)
	}
}
