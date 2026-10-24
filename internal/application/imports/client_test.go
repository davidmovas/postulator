package imports_test

import (
	"fmt"
	"maps"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/note"
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
	return usedAs(report, "ignored")
}

func usedAs(report imports.PreviewReport, use string) []string {
	out := make([]string, 0)
	for _, column := range report.Columns {
		if column.Use == use {
			out = append(out, column.Header)
		}
	}
	return out
}

func (h harness) trailAt(t *testing.T, path string) []string {
	t.Helper()

	stored, found := h.pageAt(t, path)
	if !found {
		t.Fatalf("no page at %s", path)
	}
	return stored.PlannedTrail
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
	if first.Counts.EntitiesCreated != 11 || first.Counts.PagesCreated != 11 || first.Counts.EdgesCreated != 10 {
		t.Fatalf("the group sheet wrote %+v, want eleven entities, eleven pages and ten edges", first.Counts)
	}
	if left := ignored(first.Report); len(left) != 0 {
		t.Fatalf("the group sheet ignored %v, want every column read", left)
	}
	if read := usedAs(first.Report, "trail"); !slices.Equal(read, []string{"Category", "Subcategory"}) {
		t.Fatalf("the group sheet reads %v as the trail, want the category columns", read)
	}
	for path, want := range map[string][]string{
		"/peptides/": nil, "/peptides/bpc-157/": {"Peptides"}, "/peptides/bpc-157/liquid/": {"Peptides", "BPC-157"},
		"/peptides/tb-500/capsules/": {"Peptides", "TB-500"}, "/peptides/storage/": {"Peptides"},
		"/reconstitution/": {"Peptides"}, "/healing-peptides/": {"Peptides"}, "/bpc-157-dosing-guide/": {"Peptides", "BPC-157"},
	} {
		if got := h.trailAt(t, path); !slices.Equal(got, want) {
			t.Errorf("the trail of %s = %q, want %q", path, got, want)
		}
	}
	for _, path := range []string{"/reconstitution/", "/healing-peptides/", "/bpc-157-dosing-guide/"} {
		if view, found := page(first.Report, path); !found || view.Action != string(imports.ActionCreate) || view.MatchedBy != "" {
			t.Errorf("the row without a URL at %s = %+v, want it planned at the slug of its title on a site without it", path, view)
		}
	}
	if orphaned := findings(first.Report.Warnings, imports.CodeGroupWithoutPage); len(orphaned) != 0 {
		t.Fatalf("a group of the group sheet found no page: %+v", orphaned)
	}
	for _, want := range [][2]string{
		{"BPC-157", "Peptides"}, {"TB-500", "Peptides"}, {"Storing peptides", "Peptides"}, {"BPC-157 Liquid", "BPC-157"},
		{"BPC-157 Powder", "BPC-157"}, {"TB-500 Liquid", "TB-500"}, {"TB-500 Capsules", "TB-500"},
		{"Reconstitution", "Peptides"}, {"Healing Peptides", "Peptides"}, {"BPC-157 dosing guide", "Peptides"},
	} {
		if !hasEdge(first.Report, want[0], want[1], string(graph.EdgeParent)) {
			t.Errorf("edges = %+v, want %s under %s", first.Report.Edges, want[0], want[1])
		}
	}
	if bpc, _ := page(first.Report, "/peptides/bpc-157/"); len(bpc.Keywords) != 3 || bpc.Keywords[0].Text != "bpc 157" ||
		bpc.Keywords[1].Volume == nil || *bpc.Keywords[1].Volume != 5400 {
		t.Fatalf("the keywords of BPC-157 = %+v", bpc.Keywords)
	}
	if typed := h.typeNames(t); len(typed) != 0 {
		t.Errorf("after the group sheet the pages carry the types %v, want none: the sheet names no type", typed)
	}
	catalog := h.apply(t, path, h.sheet(t, path, "Catalog"))
	clean(t, "Catalog", catalog)
	if catalog.Counts.EntitiesCreated != 5 || catalog.Counts.PagesCreated != 5 || catalog.Counts.EdgesCreated != 4 {
		t.Fatalf("the catalog wrote %+v, want four products and the shop above them", catalog.Counts)
	}
	if left := ignored(catalog.Report); len(left) != 0 {
		t.Fatalf("the catalog ignored %v, want every column read", left)
	}
	if len(catalog.Report.Groups) != 0 {
		t.Fatalf("groups = %+v, want none: the catalog has no root column", catalog.Report.Groups)
	}
	for path, want := range map[string][]string{
		"/shop/bpc-157-5mg/": {"Peptides", "BPC-157"}, "/shop/tb-500-5mg/": {"Peptides", "TB-500"},
		"/shop/recovery-blend/": {"Blends", "Recovery"}, "/shop/": nil,
	} {
		if got := h.trailAt(t, path); !slices.Equal(got, want) {
			t.Errorf("the trail of the waiting row %s = %q, want %q", path, got, want)
		}
	}
	for _, product := range []string{"BPC-157 5 mg vial", "BPC-157 10 mg vial", "TB-500 5 mg vial", "BPC-157 and TB-500 blend"} {
		if !hasEdge(catalog.Report, product, "Shop", string(graph.EdgeParent)) {
			t.Errorf("edges = %+v, want %s under the shop its url names", catalog.Report.Edges, product)
		}
	}
	if got := h.scopeOf(t, "BPC-157 10 mg vial"); !slices.Equal(got, []string{"Shop › BPC-157 10 mg vial"}) {
		t.Fatalf("the 10 mg vial sits at %v, want under the shop", got)
	}

	wide := h.apply(t, path, h.sheet(t, path, "Entities"))
	clean(t, "Entities", wide)
	if wide.Counts.EntitiesCreated != 2 || wide.Counts.PagesCreated != 1 {
		t.Fatalf("the wide sheet wrote %+v, want Canada with its page and Tissue repair alone created", wide.Counts)
	}
	if canada, found := entity(wide.Report, "Peptides in Canada"); !found || canada.Parent != "Peptides" {
		t.Fatalf("Canada = %+v, want an entity under Peptides", canada)
	}
	if repair, found := entity(wide.Report, "Tissue repair"); !found || repair.Parent != "Peptides" || repair.Action != string(imports.ActionCreate) {
		t.Fatalf("Tissue repair = %+v, want an entity under Peptides", repair)
	}
	if _, planned := h.pageAt(t, "/tissue-repair/"); planned {
		t.Fatal("the row that names only Tissue repair planned a page")
	}
	if got := cardNotesOf(t, h, "Tissue repair"); len(got) != 2 {
		t.Fatalf("the card of Tissue repair = %+v, want the row's notes on the entity it names", got)
	}
	if tb, found := page(wide.Report, "/peptides/tb-500/"); !found || tb.MatchedBy != "entity" || tb.Action != string(imports.ActionUpdate) {
		t.Fatalf("the row that names TB-500 = %+v, want the entity's page updated", tb)
	}
	wantTypes := map[string]string{
		"/peptides/": "Hub", "/peptides/bpc-157/": "Product", "/peptides/tb-500/": "Product", "/peptides/canada/": "Entity",
	}
	if got := h.typeNames(t); !maps.Equal(got, wantTypes) {
		t.Fatalf("the wide sheet typed the pages %v, want the types its entity level says %v", got, wantTypes)
	}
	if canada, _ := page(wide.Report, "/peptides/canada/"); canada.PageType != "Entity" {
		t.Fatalf("the preview shows Canada as %+v, want an Entity page", canada)
	}
	if left := ignored(wide.Report); !slices.Equal(left, []string{"Entity ID", "Primary Entity", "Entity Type", "Page Template"}) {
		t.Fatalf("the wide sheet ignored %v", left)
	}
	for path, want := range map[string][]string{
		"/peptides/": nil, "/peptides/bpc-157/": {"Peptides"}, "/peptides/canada/": {"Peptides"}, "/peptides/tb-500/": {"Peptides"},
	} {
		if got := h.trailAt(t, path); !slices.Equal(got, want) {
			t.Errorf("after the wide sheet the trail of %s = %q, want %q", path, got, want)
		}
	}
	if dropped := findings(wide.Report.Warnings, imports.CodePageTypeColumnDropped); len(dropped) != 1 ||
		!strings.Contains(dropped[0].Message, "Entity Type") || dropped[0].Sheet != "Entities" {
		t.Fatalf("dropped findings = %+v, want the Entity Type column named once", dropped)
	}
	want := []note.Note{{Label: "Intent Owner", Text: "Commercial Product"}, {Label: "Notes", Text: "Sold as a 10 ml vial"}}
	if got := notesAt(t, h, "/peptides/bpc-157/"); !slices.Equal(got, want) {
		t.Fatalf("the notes of BPC-157 = %+v, want %+v", got, want)
	}
	want = []note.Note{{Label: "Intent Owner", Text: "Commercial Product"}, {Label: "Notes", Text: "Sold as a 5 mg vial"}}
	if got := notesAt(t, h, "/peptides/tb-500/"); !slices.Equal(got, want) {
		t.Fatalf("the notes of TB-500 = %+v, want %+v", got, want)
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

func pageOf(report imports.PreviewReport, sheet, path string) (imports.PreviewPage, bool) {
	for i := range report.Pages {
		if report.Pages[i].Sheet == sheet && report.Pages[i].Path == path {
			return report.Pages[i], true
		}
	}
	return imports.PreviewPage{}, false
}

func TestTheClientWorkbookFindsTheReplicasPagesWithoutAURL(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	for at, held := range []struct{ path, title string }{
		{"/", "Home"}, {"/peptides/", "Peptides"}, {"/bpc-157/", "BPC-157"}, {"/tb-500/", "TB-500"},
		{"/healing-peptides/", "Healing Peptides"}, {"/lab-guides/", "Lab Guides"}, {"/reconstitution/", "Reconstitution"},
	} {
		h.sitePage(t, held.path, held.title, int64(10+at), nil)
	}
	peptides := int64(90)
	h.termPage(t, "/product-category/peptides/", "Peptides", peptides, nil)
	h.termPage(t, "/product-category/peptides/healing-peptides/", "Healing Peptides", 100, &peptides)
	h.storeProduct(t, "bpc-157-peptide", "BPC-157 Peptide", 501)
	h.storeProduct(t, "tb-500-peptide", "TB-500 Peptide", 502)

	report := h.workbookPreview(t, clientWorkbook, clientSheets...)
	if len(report.Errors) != 0 || len(findings(report.Warnings, imports.CodeRowAmbiguous)) != 0 {
		t.Fatalf("errors = %+v, warnings = %+v, want every row to find one page or none", report.Errors, report.Warnings)
	}
	cases := []struct {
		name, sheet, path, rule string
		action                  imports.Action
	}{
		{name: "a row by its title", sheet: "Groups", path: "/reconstitution/", rule: "title", action: imports.ActionUpdate},
		{name: "a hub its store category shares a name with", sheet: "Groups", path: "/healing-peptides/", rule: "title", action: imports.ActionUpdate},
		{name: "a row no page answers, under its level's page", sheet: "Groups", path: "/bpc-157/bpc-157-dosing-guide/", action: imports.ActionCreate},
		{name: "a row by its entity's name", sheet: "Entities", path: "/tb-500/", rule: "entity", action: imports.ActionUpdate},
	}
	for _, tc := range cases {
		if view, found := pageOf(report, tc.sheet, tc.path); !found || view.MatchedBy != tc.rule || view.Action != string(tc.action) {
			t.Errorf("%s: the %s sheet shows %s as %+v (%t), want %s by %q", tc.name, tc.sheet, tc.path, view, found, tc.action, tc.rule)
		}
	}
	for i := range report.Pages {
		if report.Pages[i].Path == "/tissue-repair/" || report.Pages[i].Path == "/product-category/peptides/healing-peptides/" {
			t.Errorf("the workbook plans or writes %+v", report.Pages[i])
		}
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

	entities, edges, pages, typed := h.entities(t), h.edges(t), h.pages(t), h.typeNames(t)
	byID := make(map[string]graph.Entity, len(entities))
	for i := range entities {
		byID[entities[i].ID] = entities[i]
	}
	pathOf := make(map[string]string, len(pages))
	for i := range pages {
		pathOf[pages[i].ID] = pages[i].Path
	}

	out := siteLabels{}
	for i := range entities {
		held := &entities[i]
		out.entities = append(out.entities, fmt.Sprintf("%s | %s", trail(byID, held.ID), held.Keywords.Cell()))
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
		out.pages = append(out.pages, fmt.Sprintf("%s | %s | %s | %s | %s | %s | %v | planned %s | under %s | typed %s %s %s",
			held.Path, held.WPType, trail(byID, deref(held.EntityID)), held.Title, held.H1, held.Keywords.Cell(),
			held.Notes, held.PlannedPath, pathOf[deref(held.ParentPageID)], typed[held.Path], held.TypeSource, held.TypeReason))
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
	if len(want.entities) != 11+5+2+2 || len(want.pages) == 0 || len(want.edges) == 0 {
		t.Fatalf("the sheets imported one by one hold %d entities, %d pages and %d edges, want twenty entities",
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
	if left := ignored(applied.Report); len(left) != 4+1 {
		t.Errorf("ignored = %v, want only the columns no field and no level reads", left)
	}
	if read := usedAs(applied.Report, "trail"); len(read) != 2+2+2 {
		t.Errorf("trail columns = %v, want the category columns of three sheets", read)
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
