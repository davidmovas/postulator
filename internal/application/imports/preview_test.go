package imports_test

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/importer"
	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func TestInspectOpensTheFilePreMapped(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	path := h.file(t, "client.csv", "URL;Page Title;Secondary Keywords\n/a/;A;one\n/b/;B;two\n/c/;C;three\n/d/;D;four\n/e/;E;five\n/f/;F;six\n")

	got, err := h.service.Inspect(t.Context(), imports.InspectRequest{SiteID: h.siteID, Path: path})
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if len(got.Headers) != 3 || got.Rows != 6 || len(got.Sample) != 5 {
		t.Fatalf("headers = %v, rows = %d, sample = %d", got.Headers, got.Rows, len(got.Sample))
	}
	if got.Detected.Columns[string(importmap.FieldPath)] != "URL" ||
		got.Detected.Columns[string(importmap.FieldTitle)] != "Page Title" ||
		got.Detected.Columns[string(importmap.FieldKeywords)] != "Secondary Keywords" {
		t.Fatalf("detected = %v", got.Detected.Columns)
	}
	if got.Detected.SiteID != h.siteID || len(got.Saved) != 0 {
		t.Fatalf("detected site = %s, saved = %v", got.Detected.SiteID, got.Saved)
	}
}

func TestInspectRefusesWhatItCannotOpen(t *testing.T) {
	t.Parallel()

	h := newHarnessWithRows(t, 100)

	cases := []struct {
		name string
		req  func(*testing.T) imports.InspectRequest
		code errors.Code
	}{
		{
			name: "no site",
			req: func(t *testing.T) imports.InspectRequest {
				return imports.InspectRequest{Path: h.file(t, "a.csv", "path\n/\n")}
			},
			code: errors.Invalid,
		},
		{
			name: "unknown site",
			req: func(t *testing.T) imports.InspectRequest {
				return imports.InspectRequest{SiteID: "00000000-0000-4000-8000-000000000000", Path: h.file(t, "b.csv", "path\n/\n")}
			},
			code: errors.NotFound,
		},
		{
			name: "not a sheet",
			req: func(t *testing.T) imports.InspectRequest {
				return imports.InspectRequest{SiteID: h.siteID, Path: h.file(t, "c.txt", "path\n/\n")}
			},
			code: errors.Invalid,
		},
		{
			name: "above the row ceiling",
			req: func(t *testing.T) imports.InspectRequest {
				body := strings.Builder{}
				body.WriteString("path\n")
				for i := range 101 {
					body.WriteString("/p" + string(rune('a'+i%26)) + string(rune('a'+i/26)) + "/\n")
				}
				return imports.InspectRequest{SiteID: h.siteID, Path: h.file(t, "d.csv", body.String())}
			},
			code: errors.Invalid,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := h.service.Inspect(t.Context(), tc.req(t)); !errors.IsCode(err, tc.code) {
				t.Fatalf("Inspect = %v, want %s", err, tc.code)
			}
		})
	}
}

func TestPreviewReadsTheGraphWithoutWriting(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	report := h.preview(t, h.file(t, "graph.csv", graphSheet), graphMapping(h))

	if len(report.Errors) != 0 {
		t.Fatalf("errors = %+v", report.Errors)
	}
	if len(report.Entities) != 3 || len(report.Pages) != 2 {
		t.Fatalf("entities = %d, pages = %d", len(report.Entities), len(report.Pages))
	}

	hosting, found := entity(report, "Hosting")
	if !found || hosting.Action != string(imports.ActionCreate) {
		t.Fatalf("hosting = %+v", hosting)
	}
	if len(hosting.Keywords) != 2 || hosting.Keywords[0].Text != "hosting" || hosting.Keywords[1].Text != "servers" {
		t.Fatalf("hosting keywords = %+v, want the cell in its order", hosting.Keywords)
	}
	if len(report.Edges) != 3 {
		t.Fatalf("edges = %+v", report.Edges)
	}
	if len(h.entities(t)) != 0 || len(h.pages(t)) != 0 {
		t.Fatal("the preview wrote to the database")
	}
}

func TestPreviewFlagsWhatTheSheetGetsWrong(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		sheet    string
		code     imports.FindingCode
		errored  bool
		expected int
	}{
		{
			name:     "an unknown parent",
			sheet:    "path,entity,parent\n/a/,Alpha,Absent\n",
			code:     imports.CodeUnknownParent,
			errored:  true,
			expected: 1,
		},
		{
			name:     "an unknown related entity",
			sheet:    "path,entity,related\n/a/,Alpha,Absent\n",
			code:     imports.CodeUnknownRelated,
			errored:  true,
			expected: 1,
		},
		{
			name:     "an entity that parents itself",
			sheet:    "path,entity,parent\n/a/,Alpha,alpha\n",
			code:     imports.CodeSelfEdge,
			errored:  true,
			expected: 1,
		},
		{
			name:     "a cycle between two entities",
			sheet:    "path,entity,parent\n/a/,Alpha,Beta\n/b/,Beta,Alpha\n",
			code:     imports.CodeCycle,
			errored:  true,
			expected: 1,
		},
		{
			name:     "a path that cannot be read",
			sheet:    "path,entity,parent\n/a/../b/,Alpha,\n",
			code:     imports.CodeBadPath,
			errored:  true,
			expected: 1,
		},
		{
			name:     "a repeated path",
			sheet:    "path,entity,parent\n/a/,Alpha,\n/A/,Alpha,\n",
			code:     imports.CodeDuplicatePath,
			errored:  false,
			expected: 1,
		},
		{
			name:     "a missing intermediate path",
			sheet:    "path,entity,parent\n/a/b/c/,Alpha,\n",
			code:     imports.CodeIntermediatePath,
			errored:  false,
			expected: 2,
		},
		{
			name:     "a root row the site does not hold",
			sheet:    "path,entity,parent\n/,Alpha,\n",
			code:     imports.CodeRootPageSkipped,
			errored:  false,
			expected: 1,
		},
		{
			name:     "a row that names nothing",
			sheet:    "path,entity,parent\n,,Alpha\n",
			code:     imports.CodeNoTarget,
			errored:  false,
			expected: 1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			mapping := h.mapping(map[string]string{
				string(importmap.FieldPath):         "path",
				string(importmap.FieldEntity):       "entity",
				string(importmap.FieldParentEntity): "parent",
			})
			if strings.Contains(tc.sheet, "related") {
				mapping.Columns = map[string]string{
					string(importmap.FieldPath):    "path",
					string(importmap.FieldEntity):  "entity",
					string(importmap.FieldRelated): "related",
				}
			}

			report := h.preview(t, h.file(t, "sheet.csv", tc.sheet), mapping)
			list := report.Warnings
			if tc.errored {
				list = report.Errors
			}
			if got := findings(list, tc.code); len(got) != tc.expected {
				t.Fatalf("%s findings = %+v, want %d; report = %+v", tc.code, got, tc.expected, report)
			}
		})
	}
}

func TestPreviewCreatesTheIntermediatePathsItNeeds(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	report := h.preview(t, h.file(t, "deep.csv", "path,title\n/services/web/react/,React\n"),
		h.mapping(map[string]string{string(importmap.FieldPath): "path", string(importmap.FieldTitle): "title"}))

	for _, path := range []string{"/services/", "/services/web/", "/services/web/react/"} {
		found, ok := page(report, path)
		if !ok {
			t.Fatalf("%s is missing from %+v", path, report.Pages)
		}
		if path == "/services/web/react/" {
			if found.Generated || found.Title != "React" {
				t.Fatalf("the named page = %+v", found)
			}
			continue
		}
		if !found.Generated {
			t.Fatalf("%s is not flagged as generated", path)
		}
	}
	if title, _ := page(report, "/services/web/"); title.Title != "Web" {
		t.Fatalf("the generated title = %q, want Web", title.Title)
	}
	if root, ok := page(report, "/"); ok {
		t.Fatalf("the import planned the root of the site: %+v", root)
	}
	if got := findings(report.Warnings, imports.CodeIntermediatePath); len(got) != 2 {
		t.Fatalf("intermediate findings = %+v, want the two sections and no root", got)
	}
}

func TestPreviewSkipsARootRowTheSiteDoesNotHold(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	report := h.preview(t, h.file(t, "root.csv", "path,title,entity\n/,Home,Shop\n/hosting/,Hosting,Hosting\n"),
		h.mapping(map[string]string{
			string(importmap.FieldPath):   "path",
			string(importmap.FieldTitle):  "title",
			string(importmap.FieldEntity): "entity",
		}))

	if _, ok := page(report, "/"); ok {
		t.Fatalf("the root was planned: %+v", report.Pages)
	}
	if _, ok := page(report, "/hosting/"); !ok {
		t.Fatalf("the section is missing from %+v", report.Pages)
	}
	if got := findings(report.Warnings, imports.CodeRootPageSkipped); len(got) != 1 || got[0].Row != 2 {
		t.Fatalf("root findings = %+v, want one on row 2", got)
	}
	if _, ok := entity(report, "Shop"); !ok {
		t.Fatalf("the entity of the root row was dropped with it: %+v", report.Entities)
	}
	if len(report.Pages) != 1 || report.Skipped != 0 {
		t.Fatalf("pages = %+v, skipped = %d; want one page and no skipped row", report.Pages, report.Skipped)
	}
}

func TestPreviewMergesARootRowIntoTheRootTheSiteHolds(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	wpID := int64(1)
	root, err := pagemap.NewPage(pagemap.Page{
		ID: id.New(), SiteID: h.siteID, Path: "/", WPType: pagemap.WPPage, WPID: &wpID, Title: "Front page",
		Status: pagemap.StatusPublished, CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	})
	if err != nil {
		t.Fatalf("build the root: %v", err)
	}
	if err = sqlite.NewPageRepo(h.store).Insert(t.Context(), root); err != nil {
		t.Fatalf("insert the root: %v", err)
	}

	report := h.preview(t, h.file(t, "root.csv", "path,title,h1,entity\n/,Home,Welcome,Shop\n"),
		h.mapping(map[string]string{
			string(importmap.FieldPath):   "path",
			string(importmap.FieldTitle):  "title",
			string(importmap.FieldH1):     "h1",
			string(importmap.FieldEntity): "entity",
		}))

	merged, ok := page(report, "/")
	if !ok || merged.Action != string(imports.ActionUpdate) || merged.H1 != "Welcome" {
		t.Fatalf("the root row = %+v, want an update carrying the h1", merged)
	}
	if got := findings(report.Warnings, imports.CodeRootPageSkipped); len(got) != 0 {
		t.Fatalf("the root was skipped although the site holds it: %+v", got)
	}
	if len(report.Pages) != 1 {
		t.Fatalf("pages = %+v, want the root alone", report.Pages)
	}
}

func TestPreviewReportsCannibalizationAgainstTheSite(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	first := h.file(t, "first.csv", "path,entity,primary keyword\n/hosting/,Hosting,cheap hosting\n")
	mapping := h.mapping(map[string]string{
		string(importmap.FieldPath):           "path",
		string(importmap.FieldEntity):         "entity",
		string(importmap.FieldPrimaryKeyword): "primary keyword",
	})
	h.apply(t, first, mapping)

	second := h.file(t, "second.csv", "path,entity,primary keyword\n/servers/,Servers,cheap hosting\n")
	report := h.preview(t, second, mapping)

	if len(report.Cannibalization) != 1 {
		t.Fatalf("cannibalization = %+v", report.Cannibalization)
	}
	if report.Cannibalization[0].Path != "/hosting/" || report.Cannibalization[0].Reason != "same_primary_keyword" {
		t.Fatalf("evidence = %+v", report.Cannibalization[0])
	}
	if len(findings(report.Warnings, imports.CodeCannibalization)) != 1 {
		t.Fatalf("warnings = %+v", report.Warnings)
	}
	if len(report.Errors) != 0 {
		t.Fatalf("cannibalization must not block the import: %+v", report.Errors)
	}
}

func TestPreviewFallsBackOnUnknownVocabulary(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	report := h.preview(t, h.file(t, "kinds.csv", "path,entity,post type,page type\n/a/,Alpha,gadget,sprocket\n,Beta,,widget\n"),
		h.mapping(map[string]string{
			string(importmap.FieldPath):     "path",
			string(importmap.FieldEntity):   "entity",
			string(importmap.FieldWPType):   "post type",
			string(importmap.FieldPageType): "page type",
		}))

	for _, code := range []imports.FindingCode{imports.CodeUnknownWPType, imports.CodeUnknownPageType} {
		if got := findings(report.Warnings, code); len(got) != 1 || got[0].Row != 2 {
			t.Fatalf("%s findings = %+v, want one on the row with a page", code, report.Warnings)
		}
	}
	if found, _ := page(report, "/a/"); found.WPType != "page" || found.PageType != "" {
		t.Fatalf("the page = %+v, want an untyped page", found)
	}
	if len(report.Errors) != 0 {
		t.Fatalf("an unknown page type blocks the import: %+v", report.Errors)
	}
}

func TestACellNamesThePageTypeByTheSitesOwnNameFirst(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		rename map[pagetype.Preset]string
		cell   string
		want   string
	}{
		{name: "a type by its name", cell: "Hub", want: "Hub"},
		{name: "a name in another case", cell: "PRODUCT", want: "Product"},
		{name: "a category is a hub", cell: "Category", want: "Hub"},
		{name: "a commercial taxonomy is a hub", cell: "Commercial Taxonomy", want: "Hub"},
		{name: "a pillar is a hub", cell: "Pillar", want: "Hub"},
		{name: "a root is a hub", cell: "Root", want: "Hub"},
		{name: "a compound is a product", cell: "Compound", want: "Product"},
		{name: "a compound product is a product", cell: "Compound/Product", want: "Product"},
		{name: "a product owner is a product", cell: "Product Owner", want: "Product"},
		{name: "a topic is an entity", cell: "Topic", want: "Entity"},
		{name: "a guide is an entity", cell: "guide", want: "Entity"},
		{name: "a geographic entity is an entity", cell: "Geographic Entity", want: "Entity"},
		{name: "the store category on a page is a hub", cell: "Store category", want: "Hub"},
		{name: "the hub renamed category", rename: map[pagetype.Preset]string{pagetype.Hub: "Category"}, cell: "Category", want: "Category"},
		{name: "the entity renamed category wins the synonym", rename: map[pagetype.Preset]string{pagetype.Entity: "Category"}, cell: "category", want: "Category"},
		{name: "the hub renamed keeps its synonyms", rename: map[pagetype.Preset]string{pagetype.Hub: "Collection"}, cell: "Pillar", want: "Collection"},
		{name: "a renamed type answers its new name", rename: map[pagetype.Preset]string{pagetype.Product: "Goods"}, cell: "Goods", want: "Goods"},
		{name: "a renamed type keeps its synonyms", rename: map[pagetype.Preset]string{pagetype.Product: "Goods"}, cell: "Compound", want: "Goods"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			for preset, name := range tc.rename {
				h.rename(t, preset, name)
			}
			path := h.file(t, "types.csv", "url,entity name,entity level\n/a/,Alpha,"+tc.cell+"\n")
			mapping := h.mapping(map[string]string{
				string(importmap.FieldPath): "url", string(importmap.FieldEntity): "entity name", string(importmap.FieldPageType): "entity level",
			})
			report := h.preview(t, path, mapping)
			if found, _ := page(report, "/a/"); found.PageType != tc.want {
				t.Fatalf("the page reads as %+v, want the type %s", found, tc.want)
			}
			if unknown := findings(report.Warnings, imports.CodeUnknownPageType); len(unknown) != 0 {
				t.Fatalf("a type the site knows is reported unknown: %+v", unknown)
			}
			if typed := h.typeNames(t); len(typed) != 0 {
				t.Fatalf("the preview typed %v", typed)
			}
			h.apply(t, path, mapping)
			if typed := h.typeNames(t); typed["/a/"] != tc.want {
				t.Fatalf("the stored page is typed %v, want %s", typed, tc.want)
			}
		})
	}
}

func TestAStoreCategoryPageIsAStoreCategoryWhateverItsCell(t *testing.T) {
	t.Parallel()

	cases := []string{"Hub", "Product", "Pillar", "Store category"}

	for _, cell := range cases {
		t.Run(cell, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			wpID := int64(40)
			term, err := pagemap.NewPage(pagemap.Page{
				ID: id.New(), SiteID: h.siteID, Path: "/product-category/peptides/", WPType: pagemap.WPProductCategory, WPID: &wpID,
				Title: "Peptides", Status: pagemap.StatusPublished, CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
			})
			if err != nil {
				t.Fatalf("build the term page: %v", err)
			}
			if err = sqlite.NewPageRepo(h.store).Insert(t.Context(), term); err != nil {
				t.Fatalf("insert the term page: %v", err)
			}

			path := h.file(t, "terms.csv", "url,page type\n/product-category/peptides/,"+cell+"\n")
			mapping := h.mapping(map[string]string{string(importmap.FieldPath): "url", string(importmap.FieldPageType): "page type"})
			report := h.preview(t, path, mapping)
			if found, _ := page(report, term.Path); found.PageType != "Store category" {
				t.Fatalf("the term page reads as %+v, want a store category", found)
			}
			h.apply(t, path, mapping)
			if typed := h.typeNames(t); typed[term.Path] != "Store category" {
				t.Fatalf("the stored term page is typed %v", typed)
			}
		})
	}
}

func TestAnEntityLevelColumnWinsThePageTypeAndThePreviewNamesTheOneItDropped(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		sheet   string
		column  string
		dropped string
		want    string
	}{
		{name: "the kind before the entity type", sheet: "URL,Kind,Entity type\n/a/,Hub,Product\n", column: "Entity type", dropped: "Kind", want: "Product"},
		{name: "the entity type before the kind", sheet: "URL,Entity type,Kind\n/a/,Product,Hub\n", column: "Entity type", dropped: "Kind", want: "Product"},
		{name: "the page type before the entity level", sheet: "URL,Page type,Entity level\n/a/,Hub,Topic\n", column: "Entity level", dropped: "Page type", want: "Entity"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			path := h.file(t, "both.csv", tc.sheet)
			mapping := h.detected(t, path)
			if got := mapping.Columns[string(importmap.FieldPageType)]; got != tc.column {
				t.Fatalf("the page type is read from %q, want %q", got, tc.column)
			}
			report := h.preview(t, path, mapping)
			if found, _ := page(report, "/a/"); found.PageType != tc.want {
				t.Fatalf("the page reads as %+v, want %s", found, tc.want)
			}
			dropped := findings(report.Warnings, imports.CodePageTypeColumnDropped)
			if len(dropped) != 1 || !strings.Contains(dropped[0].Message, tc.dropped) || !strings.Contains(dropped[0].Message, tc.column) ||
				dropped[0].Field != string(importmap.FieldPageType) {
				t.Fatalf("dropped findings = %+v, want one naming %s and %s", dropped, tc.dropped, tc.column)
			}
			if left := ignored(report); !slices.Equal(left, []string{tc.dropped}) {
				t.Fatalf("ignored = %v, want %s", left, tc.dropped)
			}
			if len(report.Errors) != 0 {
				t.Fatalf("errors = %+v", report.Errors)
			}
		})
	}
}

func TestATemplateColumnIsReadAsNothing(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	path := h.file(t, "templates.csv", "URL,Template,Page template\n/a/,hub,guide\n")
	mapping := h.detected(t, path)
	if _, read := mapping.Columns[string(importmap.FieldPageType)]; read {
		t.Fatalf("a template column was read as the page type: %v", mapping.Columns)
	}
	report := h.preview(t, path, mapping)
	if left := ignored(report); !slices.Equal(left, []string{"Template", "Page template"}) {
		t.Fatalf("ignored = %v, want both template columns", left)
	}
	if found, _ := page(report, "/a/"); found.PageType != "" {
		t.Fatalf("the page reads as %+v, want it untyped", found)
	}
	if len(report.Warnings) != 0 {
		t.Fatalf("warnings = %+v", report.Warnings)
	}
}

func (h harness) workbook(t *testing.T, name string, table importmap.Table) string {
	t.Helper()

	path := filepath.Join(h.dir, name)
	if err := importer.Write(path, table); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

func TestEveryFindingNamesItsSheet(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	path := h.workbook(t, "faults.xlsx", importmap.Table{
		Headers: []string{"path", "entity", "parent", "wp type", "page type", "is entity"},
		Rows: [][]string{
			{"/a/b/c/", "", "", "", "", ""},
			{"/x/", "", "", "gadget", "", ""},
			{"/y/", "", "", "", "sprocket", ""},
			{"/", "Home", "", "", "", ""},
			{"/tech/", "", "", "", "", "no"},
			{"/tech/child/", "", "", "", "", ""},
			{"/loop-a/", "Loop A", "Loop B", "", "", ""},
			{"/loop-b/", "Loop B", "Loop A", "", "", ""},
		},
	})
	report := h.preview(t, path, h.detected(t, path))

	rows := map[imports.FindingCode]int{
		imports.CodeIntermediatePath: 0,
		imports.CodeUnknownWPType:    3,
		imports.CodeUnknownPageType:  4,
		imports.CodeRootPageSkipped:  5,
		imports.CodeTechnicalParent:  6,
		imports.CodeCycle:            0,
	}
	for code, row := range rows {
		got := findings(slices.Concat(report.Errors, report.Warnings), code)
		if len(got) == 0 {
			t.Errorf("no %s finding in %+v", code, report)
			continue
		}
		if got[0].Row != row {
			t.Errorf("%s = %+v, want row %d", code, got[0], row)
		}
	}
	for _, finding := range slices.Concat(report.Errors, report.Warnings) {
		if finding.Sheet != "Sheet1" {
			t.Errorf("%s names the sheet %q, want Sheet1: %+v", finding.Code, finding.Sheet, finding)
		}
	}

	products := h.workbook(t, "products.xlsx", importmap.Table{
		Headers: []string{"path", "h1"},
		Rows:    [][]string{{"/mak/powder/", "Mak Powder"}},
	})
	mapping := h.detected(t, products)
	mapping.Options.RowType = importmap.RowProducts
	waiting := findings(h.preview(t, products, mapping).Warnings, imports.CodeProductNotInStore)
	if len(waiting) != 1 || waiting[0].Sheet != "Sheet1" || waiting[0].Row != 2 {
		t.Errorf("not-in-store findings = %+v, want row 2 of Sheet1", waiting)
	}
}

func TestPreviewRefusesAMappingTheFileDoesNotCarry(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	_, err := h.service.Preview(t.Context(), imports.PreviewRequest{
		SiteID:  h.siteID,
		Path:    h.file(t, "thin.csv", "path\n/a/\n"),
		Mapping: h.mapping(map[string]string{string(importmap.FieldPath): "path", string(importmap.FieldTitle): "title"}),
	})
	if !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Preview = %v, want an invalid error", err)
	}
}

func TestTheFindingCodesDeclareWhichOnesBlockAnApply(t *testing.T) {
	t.Parallel()

	cases := []struct {
		code     imports.FindingCode
		blocking bool
	}{
		{code: imports.CodeBadPath, blocking: true},
		{code: imports.CodeNoTarget, blocking: false},
		{code: imports.CodeDuplicatePath, blocking: false},
		{code: imports.CodeIntermediatePath, blocking: false},
		{code: imports.CodeUnknownParent, blocking: true},
		{code: imports.CodeUnknownRelated, blocking: true},
		{code: imports.CodeSelfEdge, blocking: true},
		{code: imports.CodeCycle, blocking: true},
		{code: imports.CodeCannibalization, blocking: false},
		{code: imports.CodeUnknownPageType, blocking: false},
		{code: imports.CodePageTypeKept, blocking: false},
		{code: imports.CodePageTypeColumnDropped, blocking: false},
		{code: imports.CodeUnknownWPType, blocking: false},
		{code: imports.CodeBadVolume, blocking: false},
		{code: imports.FindingCode("invented"), blocking: false},
	}

	for _, tc := range cases {
		t.Run(string(tc.code), func(t *testing.T) {
			t.Parallel()

			if got := tc.code.Blocking(); got != tc.blocking {
				t.Fatalf("%s.Blocking() = %v, want %v", tc.code, got, tc.blocking)
			}
		})
	}
}
