package imports_test

import (
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
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

func rootLevelsDropped(t *testing.T, report imports.PreviewReport, sheet string, rows ...int) {
	t.Helper()

	dropped := findings(report.Warnings, imports.CodeCategoryLevelIsRoot)
	got := make([]int, 0, len(dropped))
	for _, finding := range dropped {
		if finding.Sheet != sheet || !strings.Contains(finding.Message, "Peptides") {
			t.Errorf("the dropped level %+v, want Peptides on the %s sheet", finding, sheet)
		}
		got = append(got, finding.Row)
	}
	if !slices.Equal(got, rows) {
		t.Errorf("the %s sheet dropped Peptides on the rows %v, want %v", sheet, got, rows)
	}
}

func TestTheClientWorkbookImportsSheetBySheet(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	path := filepath.Join("..", "..", "..", "samples", "client-sheets.xlsx")

	groups := h.sheet(t, path, "Groups")
	first := h.apply(t, path, groups)
	clean(t, "Groups", first)
	if first.Counts.EntitiesCreated != 8 || first.Counts.PagesCreated != 8 || first.Counts.EdgesCreated != 7 {
		t.Fatalf("the group sheet wrote %+v, want eight entities, eight pages and seven edges", first.Counts)
	}
	if left := ignored(first.Report); len(left) != 0 {
		t.Fatalf("the group sheet ignored %v", left)
	}
	if orphaned := findings(first.Report.Warnings, imports.CodeGroupWithoutPage); len(orphaned) != 0 {
		t.Fatalf("a group of the group sheet found no page: %+v", orphaned)
	}
	for _, want := range [][2]string{
		{"BPC-157", "Peptides"}, {"TB-500", "Peptides"}, {"Storing peptides", "Peptides"}, {"BPC-157 Liquid", "BPC-157"},
		{"BPC-157 Powder", "BPC-157"}, {"TB-500 Liquid", "TB-500"}, {"TB-500 Capsules", "TB-500"},
	} {
		if !hasEdge(first.Report, want[0], want[1], string(graph.EdgeParent)) {
			t.Errorf("edges = %+v, want %s under %s", first.Report.Edges, want[0], want[1])
		}
	}
	if bpc, _ := page(first.Report, "/peptides/bpc-157/"); len(bpc.Keywords) != 3 || bpc.Keywords[0].Text != "bpc 157" ||
		bpc.Keywords[1].Volume == nil || *bpc.Keywords[1].Volume != 5400 {
		t.Fatalf("the keywords of BPC-157 = %+v", bpc.Keywords)
	}
	for name, got := range h.kinds(t) {
		want := graph.KindTopic
		if name == "Peptides" {
			want = graph.KindHub
		}
		if got != want {
			t.Errorf("after the group sheet %s is a %s, want a %s", name, got, want)
		}
	}

	catalog := h.apply(t, path, h.sheet(t, path, "Catalog"))
	clean(t, "Catalog", catalog)
	if catalog.Counts.EntitiesCreated != 5 || catalog.Counts.EdgesCreated != 4 {
		t.Fatalf("the catalog wrote %+v, want four products and the shop", catalog.Counts)
	}
	if orphaned := findings(catalog.Report.Warnings, imports.CodeGroupWithoutPage); len(orphaned) != 0 {
		t.Fatalf("groups without a page = %+v, want none: the catalog's levels are categories", orphaned)
	}
	rootLevelsDropped(t, catalog.Report, "Catalog", 2, 3, 4)
	for _, want := range [][2]string{
		{"BPC-157 5 mg vial", "BPC-157"}, {"BPC-157 10 mg vial", "BPC-157"}, {"TB-500 5 mg vial", "TB-500"},
		{"BPC-157 and TB-500 blend", "Shop"},
	} {
		if !hasEdge(catalog.Report, want[0], want[1], string(graph.EdgeParent)) {
			t.Errorf("edges = %+v, want %s under %s", catalog.Report.Edges, want[0], want[1])
		}
	}
	if got := h.scopeOf(t, "BPC-157 10 mg vial"); !slices.Equal(got, []string{"Peptides › BPC-157 › BPC-157 10 mg vial"}) {
		t.Fatalf("the 10 mg vial sits at %v, want under the BPC-157 entity", got)
	}

	wide := h.apply(t, path, h.sheet(t, path, "Entities"))
	clean(t, "Entities", wide)
	if wide.Counts.EntitiesCreated != 1 {
		t.Fatalf("the wide sheet wrote %+v, want Canada alone created", wide.Counts)
	}
	rootLevelsDropped(t, wide.Report, "Entities", 2, 3, 4)
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
		if held.Name == "NO" || held.Name == "Yes" || held.Name == "No" || held.Name == "Liquid" || held.Name == "Blends" {
			t.Fatalf("a column that is no entity made the entity %q", held.Name)
		}
	}

	again := h.apply(t, path, groups)
	if again.Counts.EntitiesCreated != 0 || again.Counts.PagesCreated != 0 || again.Counts.EdgesCreated != 0 {
		t.Fatalf("the group sheet imported again wrote %+v", again.Counts)
	}
}

type siteLabels struct {
	entities   []string
	edges      []string
	pages      []string
	canonicals []string
}

func trail(byID map[string]graph.Entity, entityID string) string {
	names := make([]string, 0, 4)
	seen := make(map[string]struct{})
	for at, held := byID[entityID]; held; at, held = byID[deref(at.ScopeID)] {
		if _, again := seen[at.ID]; again {
			break
		}
		seen[at.ID] = struct{}{}
		names = append([]string{at.Name}, names...)
	}
	return strings.Join(names, " › ")
}

func deref(ref *string) string {
	if ref == nil {
		return ""
	}
	return *ref
}

func (h harness) labels(t *testing.T) siteLabels {
	t.Helper()

	entities, edges, pages := h.entities(t), h.edges(t), h.pages(t)
	byID := make(map[string]graph.Entity, len(entities))
	for i := range entities {
		byID[entities[i].ID] = entities[i]
	}
	pathOf := make(map[string]string, len(pages))
	for i := range pages {
		pathOf[pages[i].ID] = pages[i].Path
	}

	var out siteLabels
	for i := range entities {
		held := &entities[i]
		out.entities = append(out.entities, fmt.Sprintf("%s | %s | %s", trail(byID, held.ID), held.Kind, held.Keywords.Cell()))
		if held.CanonicalPageID != nil {
			out.canonicals = append(out.canonicals, trail(byID, held.ID)+" → "+pathOf[*held.CanonicalPageID])
		}
	}
	for i := range edges {
		out.edges = append(out.edges, fmt.Sprintf("%s → %s | %s | %s",
			trail(byID, edges[i].FromEntityID), trail(byID, edges[i].ToEntityID), edges[i].Kind, edges[i].Status))
	}
	for i := range pages {
		held := &pages[i]
		out.pages = append(out.pages, fmt.Sprintf("%s | %s | %s | %s | %s | %s | %v | planned %s | under %s",
			held.Path, held.WPType, trail(byID, deref(held.EntityID)), held.Title, held.H1, held.Keywords.Cell(),
			held.Notes, held.PlannedPath, pathOf[deref(held.ParentPageID)]))
	}
	for _, list := range [][]string{out.entities, out.edges, out.pages, out.canonicals} {
		slices.Sort(list)
	}
	return out
}

func TestTheClientWorkbookImportsAsOneWorkbook(t *testing.T) {
	t.Parallel()

	sheetBySheet, whole := newHarness(t), newHarness(t)
	seen, err := whole.service.Inspect(t.Context(), imports.InspectRequest{SiteID: whole.siteID, Path: clientWorkbook})
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}

	sheets := make([]imports.SheetMapping, 0, len(seen.Sheets))
	for _, listed := range seen.Sheets {
		used := whole.sheet(t, clientWorkbook, listed.Name)
		detected := listed.Detected
		detected.Name = listed.Name
		if !reflect.DeepEqual(detected, used) {
			t.Fatalf("the %s sheet detects\n%+v\nwhere the sheet by sheet import uses\n%+v", listed.Name, detected, used)
		}
		sheets = append(sheets, imports.SheetMapping{Sheet: listed.Name, Mapping: listed.Detected})
		clean(t, listed.Name, sheetBySheet.apply(t, clientWorkbook, sheetBySheet.sheet(t, clientWorkbook, listed.Name)))
	}

	applied, err := whole.service.Apply(t.Context(), imports.ApplyRequest{SiteID: whole.siteID, Path: clientWorkbook, Sheets: sheets})
	if err != nil {
		t.Fatalf("Apply the workbook: %v", err)
	}
	clean(t, "whole", applied)

	want, got := sheetBySheet.labels(t), whole.labels(t)
	if len(want.entities) != 8+5+1+2 || len(want.pages) == 0 || len(want.edges) == 0 {
		t.Fatalf("the sheets imported one by one hold %d entities, %d pages and %d edges, want sixteen entities",
			len(want.entities), len(want.pages), len(want.edges))
	}
	for _, compared := range []struct {
		what      string
		got, want []string
	}{
		{what: "entities", got: got.entities, want: want.entities},
		{what: "edges", got: got.edges, want: want.edges},
		{what: "pages", got: got.pages, want: want.pages},
		{what: "canonical pages", got: got.canonicals, want: want.canonicals},
	} {
		if !slices.Equal(compared.got, compared.want) {
			t.Errorf("the workbook's %s differ from the sheets imported one by one:\ngot\n%s\nwant\n%s",
				compared.what, strings.Join(compared.got, "\n"), strings.Join(compared.want, "\n"))
		}
	}

	for _, edge := range applied.Report.Edges {
		if edge.From == "Capsules" && edge.To == "BPC-157" && (edge.Sheet != "Variations" || edge.Action != string(imports.ActionCreate)) {
			t.Errorf("the capsules of BPC-157 = %+v, want the variation sheet's edge to the entity planned before it", edge)
		}
	}
	if !hasEdge(applied.Report, "Capsules", "BPC-157", string(graph.EdgeParent)) {
		t.Errorf("edges = %+v, want the capsules of BPC-157 under it", applied.Report.Edges)
	}
	if orphans := findings(applied.Report.Warnings, imports.CodeGroupWithoutPage); len(orphans) != 0 {
		t.Errorf("groups without a page = %+v, want none", orphans)
	}
	for _, finding := range slices.Concat(applied.Report.Warnings, applied.Report.Errors) {
		if !slices.Contains(clientSheets, finding.Sheet) {
			t.Errorf("the finding %+v names no sheet of the workbook", finding)
		}
	}
	if dropped := findings(applied.Report.Warnings, imports.CodeCategoryLevelIsRoot); len(dropped) != 6 {
		t.Errorf("dropped levels = %+v, want Peptides on three rows of the catalog and three of the entity sheet", dropped)
	}

	again, err := whole.service.Apply(t.Context(), imports.ApplyRequest{SiteID: whole.siteID, Path: clientWorkbook, Sheets: sheets})
	if err != nil {
		t.Fatalf("Apply the workbook again: %v", err)
	}
	if again.Counts != (imports.Counts{Skipped: again.Counts.Skipped}) {
		t.Fatalf("the workbook imported again wrote %+v", again.Counts)
	}
	if after := whole.labels(t); !reflect.DeepEqual(after, got) {
		t.Fatal("the workbook imported again changed the site")
	}
}

func TestTheCatalogAloneMakesPeptidesACategoryOnlyWhereNoRootIsKnown(t *testing.T) {
	t.Parallel()

	alone := newHarness(t)
	report := alone.workbookPreview(t, clientWorkbook, "Catalog")
	if dropped := findings(report.Warnings, imports.CodeCategoryLevelIsRoot); len(dropped) != 0 {
		t.Fatalf("the catalog on an empty site dropped %+v, want Peptides kept: nothing says it is a root", dropped)
	}

	together := newHarness(t)
	rootLevelsDropped(t, together.workbookPreview(t, clientWorkbook, "Catalog", "Groups"), "Catalog", 2, 3, 4)
}
