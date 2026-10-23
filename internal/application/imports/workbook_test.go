package imports_test

import (
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

var clientWorkbook = filepath.Join("..", "..", "..", "samples", "client-sheets.xlsx")

var clientSheets = []string{"Groups", "Catalog", "Entities", "Variations"}

func (h harness) workbookPreview(t *testing.T, path string, sheets ...string) imports.PreviewReport {
	t.Helper()

	got, err := h.service.Preview(t.Context(), imports.PreviewRequest{SiteID: h.siteID, Path: path, Sheets: sheetMappings(sheets...)})
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	return got.Report
}

func sheetMappings(sheets ...string) []imports.SheetMapping {
	out := make([]imports.SheetMapping, 0, len(sheets))
	for _, sheet := range sheets {
		out = append(out, imports.SheetMapping{Sheet: sheet})
	}
	return out
}

func TestInspectDetectsEverySheetOnItsOwn(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	seen, err := h.service.Inspect(t.Context(), imports.InspectRequest{SiteID: h.siteID, Path: clientWorkbook})
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}

	names := make([]string, 0, len(seen.Sheets))
	for _, sheet := range seen.Sheets {
		names = append(names, sheet.Name)
		want := importmap.AutoDetect(sheet.Headers)
		detected := sheet.Detected
		if !reflect.DeepEqual(detected.Columns, h.mappingColumns(want)) ||
			!slices.Equal(detected.Options.LevelColumns, want.Options.LevelColumns) ||
			!slices.Equal(detected.Options.NoteColumns, want.Options.NoteColumns) {
			t.Errorf("the %s sheet detects %+v, want the mapping of its own headers %+v", sheet.Name, detected, want)
		}
		if !slices.Equal(detected.Options.Sheets, []string{sheet.Name}) || detected.Options.RowType != importmap.RowPages ||
			detected.SiteID != h.siteID {
			t.Errorf("the %s sheet detects the options %+v for site %q", sheet.Name, detected.Options, detected.SiteID)
		}
	}
	if !slices.Equal(names, clientSheets) {
		t.Fatalf("sheets = %v, want %v", names, clientSheets)
	}
	if groups := seen.Sheets[0].Detected.Options.LevelColumns; !slices.Equal(groups, []string{"Root Entity", "Category", "Subcategory"}) {
		t.Fatalf("the group sheet detects the levels %v", groups)
	}
}

func (h harness) mappingColumns(m importmap.Mapping) map[string]string {
	out := make(map[string]string, len(m.Columns))
	for field, column := range m.Columns {
		out[string(field)] = column
	}
	return out
}

func TestInspectReadsASheetOfStoreProductsAsProducts(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		sheet string
		want  importmap.RowType
	}{
		{name: "every row finds its product", sheet: "path,h1\n/mak/mak-liquid/,Mak Liquid\n/mak/capsule/,Mak Capsule\n", want: importmap.RowProducts},
		{name: "a row finds none", sheet: "path,h1\n/mak/mak-liquid/,Mak Liquid\n/mak/powder/,Mak Powder\n", want: importmap.RowPages},
		{name: "no row has a path", sheet: "entity\nMak\n", want: importmap.RowPages},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			h.storeProduct(t, "mak-liquid", "Mak Liquid", 501)
			h.storeProduct(t, "capsule-x", "Mak Capsule", 502)
			seen, err := h.service.Inspect(t.Context(), imports.InspectRequest{SiteID: h.siteID, Path: h.file(t, "rows.csv", tc.sheet)})
			if err != nil {
				t.Fatalf("Inspect: %v", err)
			}
			if got := seen.Sheets[0].Detected.Options.RowType; got != tc.want {
				t.Fatalf("the sheet reads as %q, want %q", got, tc.want)
			}
			if got := seen.Detected.Options.RowType; got != tc.want {
				t.Fatalf("the sampled sheet reads as %q, want %q", got, tc.want)
			}
		})
	}

	h := newHarness(t)
	seen, err := h.service.Inspect(t.Context(), imports.InspectRequest{SiteID: h.siteID, Path: h.file(t, "rows.csv", cases[0].sheet)})
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if got := seen.Sheets[0].Detected.Options.RowType; got != importmap.RowPages {
		t.Fatalf("a site without a store reads the sheet as %q, want pages", got)
	}
}

func TestAWorkbookIsPlannedInTheOrderOfItsSheets(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	report := h.workbookPreview(t, clientWorkbook, "Variations", "Groups")
	if len(report.Errors) != 0 {
		t.Fatalf("errors = %+v, want the variations to find the groups planned before them", report.Errors)
	}
	if first, last := report.Pages[0].Sheet, report.Pages[len(report.Pages)-1].Sheet; first != "Groups" || last != "Variations" {
		t.Fatalf("pages run from %s to %s, want the group sheet first", first, last)
	}
	if !hasEdge(report, "Capsules", "BPC-157", "parent") {
		t.Fatalf("edges = %+v, want the capsules of BPC-157 under the group the same preview plans", report.Edges)
	}
}

func TestAWorkbookRequestSaysWhatItCannotRead(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	cases := []struct {
		name string
		req  imports.PreviewRequest
	}{
		{
			name: "a mapping and sheets together",
			req: imports.PreviewRequest{
				Mapping: h.mapping(map[string]string{string(importmap.FieldPath): "URL"}), Sheets: sheetMappings("Groups"),
			},
		},
		{name: "a saved mapping and sheets together", req: imports.PreviewRequest{Mapping: imports.Mapping{ID: "m1"}, Sheets: sheetMappings("Groups")}},
		{name: "a sheet the workbook does not hold", req: imports.PreviewRequest{Sheets: sheetMappings("Groups", "Prices")}},
		{name: "a sheet named twice", req: imports.PreviewRequest{Sheets: sheetMappings("Groups", "Groups")}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := tc.req
			req.SiteID, req.Path = h.siteID, clientWorkbook
			if _, err := h.service.Preview(t.Context(), req); !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("Preview = %v, want an invalid request", err)
			}
		})
	}
}

func TestTheRowBudgetIsSharedAcrossTheSheets(t *testing.T) {
	t.Parallel()

	h := newHarnessWithRows(t, 10)
	if _, err := h.service.Preview(t.Context(), imports.PreviewRequest{
		SiteID: h.siteID, Path: clientWorkbook, Sheets: sheetMappings("Groups", "Catalog"),
	}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Preview of twelve rows = %v, want the ten row budget refused", err)
	}
	h.workbookPreview(t, clientWorkbook, "Groups")
}

func TestAScopeTheImportWouldClashOnBlocksIt(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	parent := sqlitetest.Entity(t, h.store, h.siteID, "Peptides")
	scoped := sqlitetest.Entity(t, h.store, h.siteID, "Liquid")
	repo := sqlite.NewEntityRepo(h.store)
	if err := repo.SetScope(t.Context(), scoped.ID, &parent.ID, sqlitetest.Stamp); err != nil {
		t.Fatalf("SetScope: %v", err)
	}
	sqlitetest.Entity(t, h.store, h.siteID, "Liquid")

	path := h.file(t, "plain.csv", "url,title\n/about/,About\n")
	report := h.preview(t, path, h.detected(t, path))
	if clashes := findings(report.Errors, imports.CodeScopeClash); len(clashes) != 1 {
		t.Fatalf("errors = %+v, want one scope clash", report.Errors)
	}
	_, err := h.service.Apply(t.Context(), imports.ApplyRequest{SiteID: h.siteID, Path: path, Mapping: h.detected(t, path)})
	if !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Apply = %v, want the preview's errors to refuse it", err)
	}
	if len(h.pages(t)) != 0 {
		t.Fatal("the refused apply wrote a page")
	}
}

func TestThePreviewSummaryNamesTheSheetsItRead(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	csv := h.file(t, "plain.csv", "url,title\n/about/,About\n")
	cases := []struct {
		name string
		req  imports.PreviewRequest
		want []string
	}{
		{name: "the first sheet read without being named", req: imports.PreviewRequest{Path: clientWorkbook}, want: []string{"Groups"}},
		{
			name: "the sheets of a workbook in their order",
			req:  imports.PreviewRequest{Path: clientWorkbook, Sheets: sheetMappings("Catalog", "Groups")},
			want: []string{"Groups", "Catalog"},
		},
		{name: "a file without sheets", req: imports.PreviewRequest{Path: csv}, want: []string{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := tc.req
			req.SiteID = h.siteID
			summary, err := h.service.PreviewSummary(t.Context(), req)
			if err != nil {
				t.Fatalf("PreviewSummary: %v", err)
			}
			if !slices.Equal(summary.Sheets, tc.want) {
				t.Fatalf("sheets = %v, want %v", summary.Sheets, tc.want)
			}
		})
	}
}

func TestEveryItemOfAWorkbookPreviewNamesItsSheet(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	report := h.workbookPreview(t, clientWorkbook, "Groups", "Catalog")

	named := func(what, sheet string) {
		if sheet != "Groups" && sheet != "Catalog" {
			t.Errorf("%s names the sheet %q", what, sheet)
		}
	}
	for _, column := range report.Columns {
		named("the column "+column.Header, column.Sheet)
	}
	for _, page := range report.Pages {
		named("the page "+page.Path, page.Sheet)
	}
	for _, held := range report.Entities {
		named("the entity "+held.Name, held.Sheet)
	}
	for _, group := range report.Groups {
		named("a group", group.Sheet)
	}
	for _, edge := range report.Edges {
		named("the edge from "+edge.From, edge.Sheet)
	}
	for _, finding := range slices.Concat(report.Warnings, report.Errors) {
		named("the finding "+finding.Code, finding.Sheet)
	}
	if shop, _ := page(report, "/shop/"); shop.Sheet != "Catalog" || !shop.Generated {
		t.Fatalf("the shop = %+v, want the page the catalog sheet fills in", shop)
	}
	if len(report.Columns) != 7+6 {
		t.Fatalf("columns = %+v, want those of both sheets", report.Columns)
	}
}

func TestAWorkbookApplySavesTheMappingOfEachSheet(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	applied, err := h.service.Apply(t.Context(), imports.ApplyRequest{
		SiteID: h.siteID, Path: clientWorkbook, Sheets: sheetMappings("Catalog", "Groups"),
		Options: imports.ApplyOptions{SaveMappingAs: "client"},
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if applied.Counts.EntitiesCreated != 8+5 {
		t.Fatalf("counts = %+v, want the groups and the catalog written together", applied.Counts)
	}

	listed, err := h.service.ListMappings(t.Context(), imports.ListMappingsRequest{SiteID: h.siteID})
	if err != nil {
		t.Fatalf("ListMappings: %v", err)
	}
	saved := make(map[string][]string, len(listed.Mappings))
	for _, mapping := range listed.Mappings {
		saved[mapping.Name] = mapping.Options.Sheets
	}
	want := map[string][]string{"client / Groups": {"Groups"}, "client / Catalog": {"Catalog"}}
	if !reflect.DeepEqual(saved, want) {
		t.Fatalf("saved = %v, want %v", saved, want)
	}
}
