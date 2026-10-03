package imports_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/importer"
	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func (h harness) storeProduct(t *testing.T, slug, name string, wpID int64) pagemap.Page {
	t.Helper()

	now := time.Date(2026, time.September, 18, 8, 0, 0, 0, time.UTC)
	page, err := pagemap.NewPage(pagemap.Page{
		ID: id.New(), SiteID: h.siteID, Path: "/product/" + slug + "/", WPType: pagemap.WPProduct, WPID: &wpID,
		Status: pagemap.StatusPublished, Title: name,
		Observed:  pagemap.Observed{Title: name, Slug: slug, Status: "publish", Link: "/product/" + slug + "/"},
		CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("NewPage: %v", err)
	}
	if err = sqlite.NewPageRepo(h.store).Insert(t.Context(), page); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	return page
}

func (h harness) pageAt(t *testing.T, path string) (pagemap.Page, bool) {
	t.Helper()

	stored := h.pages(t)
	for i := range stored {
		if stored[i].Path == path {
			return stored[i], true
		}
	}
	return pagemap.Page{}, false
}

func productMapping(h harness, rows importmap.RowType) imports.Mapping {
	mapping := h.mapping(map[string]string{
		string(importmap.FieldPath): "path", string(importmap.FieldH1): "h1", string(importmap.FieldKeywords): "keywords",
	})
	mapping.Options.RowType = rows
	return mapping
}

const productSheet = "path,h1,keywords\n" +
	"/mak/mak-liquid/,Mak Liquid,liquid mak\n" +
	"/mak/capsule/,Mak Capsule,mak capsules\n" +
	"/mak/powder/,Mak Powder,mak powder\n"

func TestASheetOfProductsFindsThemInTheStore(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	liquid := h.storeProduct(t, "mak-liquid", "Mak Liquid", 501)
	capsule := h.storeProduct(t, "capsule-x", "Mak Capsule", 502)
	sheet := h.file(t, "products.csv", productSheet)
	mapping := productMapping(h, importmap.RowProducts)

	report := h.preview(t, sheet, mapping)
	cases := []struct {
		path    string
		planned string
		by      string
		store   string
	}{
		{path: liquid.Path, planned: "/mak/mak-liquid/", by: string(pagemap.MatchedBySlug), store: "Mak Liquid"},
		{path: capsule.Path, planned: "/mak/capsule/", by: string(pagemap.MatchedByName), store: "Mak Capsule"},
	}
	for _, tc := range cases {
		view, found := page(report, tc.path)
		if !found || view.PlannedPath != tc.planned || view.MatchedBy != tc.by || view.StoreName != tc.store ||
			view.WPType != string(pagemap.WPProduct) || view.Action != string(imports.ActionUpdate) {
			t.Errorf("the preview shows %s as %+v", tc.path, view)
		}
	}
	waiting, found := page(report, "/mak/powder/")
	if !found || waiting.WPType != string(pagemap.WPProduct) || waiting.Action != string(imports.ActionCreate) || waiting.Title != "" {
		t.Errorf("the preview shows the row without a product as %+v", waiting)
	}
	if missing := findings(report.Warnings, imports.CodeProductNotInStore); len(missing) != 1 || missing[0].Row != 4 {
		t.Errorf("not-in-store findings = %+v, want the powder row", missing)
	}
	if _, generated := page(report, "/mak/"); generated {
		t.Error("the preview plans a page for the level above the products")
	}

	got := h.apply(t, sheet, mapping)
	if got.Counts.PagesCreated != 1 || got.Counts.PagesUpdated != 2 {
		t.Fatalf("counts = %+v, want the store's two products updated and one row waiting", got.Counts)
	}
	matched, _ := h.pageAt(t, liquid.Path)
	if matched.PlannedPath != "/mak/mak-liquid/" || matched.WPID == nil || *matched.WPID != 501 ||
		len(matched.Keywords.Texts()) != 1 || matched.EntityID == nil {
		t.Errorf("the matched product reads %+v", matched)
	}
	row, _ := h.pageAt(t, "/mak/powder/")
	if row.WPType != pagemap.WPProduct || row.WPID != nil || row.ParentPageID != nil || row.Title != "" {
		t.Errorf("the waiting row reads %+v", row)
	}
	if _, generated := h.pageAt(t, "/mak/"); generated {
		t.Error("the import wrote a page for the level above the products")
	}
	mak := false
	for _, stored := range h.entities(t) {
		mak = mak || stored.Name == "Mak"
	}
	if !mak {
		t.Error("the level above the products was not kept as an entity")
	}

	again := h.apply(t, sheet, mapping)
	if again.Counts.PagesCreated != 0 || again.Counts.PagesUpdated != 0 {
		t.Errorf("a second import of the same sheet = %+v, want nothing to do", again.Counts)
	}

	exported := filepath.Join(h.dir, "export.xlsx")
	if _, err := h.service.Export(t.Context(), imports.ExportRequest{SiteID: h.siteID, Path: exported}); err != nil {
		t.Fatalf("Export: %v", err)
	}
	table, err := importer.Read(t.Context(), exported, importer.ReadOptions{})
	if err != nil {
		t.Fatalf("Read the export: %v", err)
	}
	binding, err := importmap.AutoDetect(table.Headers).Bind(table.Headers)
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}
	paths := make(map[string]bool)
	for _, exportedRow := range table.Rows {
		paths[binding.Text(exportedRow, importmap.FieldPath)] = true
	}
	if !paths["/mak/mak-liquid/"] || !paths["/mak/capsule/"] || paths[liquid.Path] {
		t.Errorf("the export writes the paths %v, want the file's addresses of the matched products", paths)
	}
}

func TestTheRowTypeDecidesWhatARowBecomes(t *testing.T) {
	t.Parallel()

	const sheet = "path,entity kind,wp type\n" +
		"/tea/,product,\n" +
		"/tea/green/,product,\n" +
		"/mug/,product,\n" +
		"/about/,topic,\n" +
		"/cup/,product,page\n"

	cases := []struct {
		rows importmap.RowType
		want map[string]pagemap.WPType
	}{
		{rows: "", want: map[string]pagemap.WPType{
			"/tea/": pagemap.WPPage, "/tea/green/": pagemap.WPPage, "/mug/": pagemap.WPPage, "/about/": pagemap.WPPage, "/cup/": pagemap.WPPage,
		}},
		{rows: importmap.RowPages, want: map[string]pagemap.WPType{
			"/tea/": pagemap.WPPage, "/tea/green/": pagemap.WPPage, "/mug/": pagemap.WPPage, "/about/": pagemap.WPPage, "/cup/": pagemap.WPPage,
		}},
		{rows: importmap.RowProducts, want: map[string]pagemap.WPType{
			"/tea/": pagemap.WPPage, "/tea/green/": pagemap.WPProduct, "/mug/": pagemap.WPProduct, "/about/": pagemap.WPProduct, "/cup/": pagemap.WPPage,
		}},
		{rows: importmap.RowKind, want: map[string]pagemap.WPType{
			"/tea/": pagemap.WPPage, "/tea/green/": pagemap.WPProduct, "/mug/": pagemap.WPProduct, "/about/": pagemap.WPPage, "/cup/": pagemap.WPPage,
		}},
	}

	for _, tc := range cases {
		t.Run("rows "+string(tc.rows), func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			mapping := h.mapping(map[string]string{
				string(importmap.FieldPath): "path", string(importmap.FieldEntityKind): "entity kind", string(importmap.FieldWPType): "wp type",
			})
			mapping.Options.RowType = tc.rows
			report := h.preview(t, h.file(t, "kinds.csv", sheet), mapping)
			for path, want := range tc.want {
				if view, found := page(report, path); !found || view.WPType != string(want) {
					t.Errorf("%s reads as %+v, want a %s", path, view, want)
				}
			}
		})
	}
}

func TestAnImportKeepsTheTypeOfARowOnTheSite(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	sheet := h.file(t, "types.csv", "path,wp type\n/mug/,page\n")
	h.apply(t, sheet, h.mapping(map[string]string{string(importmap.FieldPath): "path", string(importmap.FieldWPType): "wp type"}))
	repo := sqlite.NewPageRepo(h.store)
	stored, _ := h.pageAt(t, "/mug/")
	wpID := int64(7)
	stored.WPID = &wpID
	if err := repo.Update(t.Context(), stored); err != nil {
		t.Fatalf("Update: %v", err)
	}

	report := h.preview(t, h.file(t, "products.csv", "path,wp type\n/mug/,product\n"),
		h.mapping(map[string]string{string(importmap.FieldPath): "path", string(importmap.FieldWPType): "wp type"}))
	if view, _ := page(report, "/mug/"); view.WPType != string(pagemap.WPPage) {
		t.Errorf("the page on the site reads as %+v, want it kept a page", view)
	}
	if kept := findings(report.Warnings, imports.CodeWPTypeKept); len(kept) != 1 {
		t.Errorf("findings = %+v, want one %s", report.Warnings, imports.CodeWPTypeKept)
	}
}

func TestAMappingRefusesARowTypeItDoesNotKnow(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	mapping := productMapping(h, "variants")
	if _, err := h.service.Preview(t.Context(), imports.PreviewRequest{SiteID: h.siteID, Path: h.file(t, "a.csv", productSheet), Mapping: mapping}); err == nil {
		t.Fatal("a mapping with an unknown row type was accepted")
	}
}
