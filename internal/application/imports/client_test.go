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
		out.entities = append(out.entities, fmt.Sprintf("%s | %s | category %t | %s",
			trail(byID, held.ID), held.Kind, held.SiteCategory, held.Keywords.Cell()))
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
	if len(want.entities) != 8+7+1+2 || len(want.pages) == 0 || len(want.edges) == 0 {
		t.Fatalf("the sheets imported one by one hold %d entities, %d pages and %d edges, want eighteen entities",
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
			t.Errorf("the capsules of BPC-157 = %+v, want the variation sheet's edge to the group planned before it", edge)
		}
	}
	if !hasEdge(applied.Report, "Capsules", "BPC-157", string(graph.EdgeParent)) {
		t.Errorf("edges = %+v, want the capsules of BPC-157 under it", applied.Report.Edges)
	}
	orphans := findings(applied.Report.Warnings, imports.CodeGroupWithoutPage)
	if len(orphans) != 2 {
		t.Errorf("groups without a page = %+v, want Blends and Recovery", orphans)
	}
	for _, finding := range slices.Concat(applied.Report.Warnings, applied.Report.Errors) {
		if !slices.Contains(clientSheets, finding.Sheet) {
			t.Errorf("the finding %+v names no sheet of the workbook", finding)
		}
	}
	for _, orphan := range orphans {
		if orphan.Sheet != "Catalog" {
			t.Errorf("the group without a page %+v, want it on the catalog sheet", orphan)
		}
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
