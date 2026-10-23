package imports_test

import (
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

var settledAt = time.Date(2026, time.October, 23, 10, 0, 0, 0, time.UTC)

func (h harness) lasting(t *testing.T, code finding.Code) []finding.Finding {
	t.Helper()

	listed, err := sqlite.NewFindingRepo(h.store).ListBySite(t.Context(), h.siteID, "")
	if err != nil {
		t.Fatalf("list the findings: %v", err)
	}
	return slices.DeleteFunc(listed, func(f finding.Finding) bool { return f.Code != code })
}

func (h harness) one(t *testing.T, code finding.Code) finding.Finding {
	t.Helper()

	held := h.lasting(t, code)
	if len(held) != 1 {
		t.Fatalf("%s findings = %+v, want exactly one", code, held)
	}
	return held[0]
}

func (h harness) retitle(t *testing.T, path, title string) {
	t.Helper()

	held, found := h.pageAt(t, path)
	if !found {
		t.Fatalf("no page at %s", path)
	}
	held.Title = title
	if err := sqlite.NewPageRepo(h.store).Update(t.Context(), held); err != nil {
		t.Fatalf("retitle %s: %v", path, err)
	}
}

func (h harness) applyCSV(t *testing.T, body string, mapping imports.Mapping) imports.ApplyResponse {
	t.Helper()

	return h.apply(t, h.file(t, "rows.csv", body), mapping)
}

func (h harness) termPage(t *testing.T, path, name string, wpID int64, parent *int64) {
	t.Helper()

	term, err := pagemap.NewPage(pagemap.Page{
		ID: id.New(), SiteID: h.siteID, Path: path, WPType: pagemap.WPProductCategory, WPID: &wpID, Title: name,
		Status: pagemap.StatusPublished, CreatedAt: observedAt, UpdatedAt: observedAt,
	})
	if err != nil {
		t.Fatalf("NewPage: %v", err)
	}
	if err = sqlite.NewPageRepo(h.store).Insert(t.Context(), term); err != nil {
		t.Fatalf("insert the term page: %v", err)
	}
	if err = sqlite.NewPageIdentityRepo(h.store).Put(t.Context(), pagemap.Identity{
		PageID: term.ID, URL: path, Status: "publish", WPType: pagemap.WPProductCategory, WPParentID: parent, ObservedAt: observedAt,
	}); err != nil {
		t.Fatalf("observe the term page: %v", err)
	}
}

func TestAFindingLivesOnce(t *testing.T) {
	t.Parallel()

	t.Run("an ambiguous row through its lifecycle", func(t *testing.T) {
		t.Parallel()

		h := newHarness(t)
		first := h.sitePage(t, "/a/guide/", "Guide", 31, nil)
		second := h.sitePage(t, "/b/guide/", "Guide", 32, nil)
		mapping := h.mapping(map[string]string{string(importmap.FieldTitle): "title", string(importmap.FieldKeywords): "keywords"})
		guide, other := "title,keywords\nGuide,guide\n", "title,keywords\nOther,other\n"
		key := pagemap.RowKey("", pagemap.Row{Title: "Guide"})
		repo := sqlite.NewFindingRepo(h.store)

		steps := []struct {
			name       string
			before     func(*testing.T)
			sheet      string
			status     finding.Status
			resolution string
			resolvedBy string
			subject    string
		}{
			{
				name: "raised with its candidates", sheet: guide, status: finding.Open,
				subject: `{"sheet":"","rule":"title","candidates":["/a/guide/","/b/guide/"],"pageIds":["` + first.ID + `","` + second.ID + `"],"product":false}`,
			},
			{name: "one row on a second import", sheet: guide, status: finding.Open},
			{
				name: "a dismissed one stays dismissed while its candidates stand",
				before: func(t *testing.T) {
					if err := repo.Settle(t.Context(), h.one(t, finding.RowAmbiguous).ID, finding.Dismissed, "ignored", finding.ResolvedByPerson, settledAt); err != nil {
						t.Fatalf("dismiss: %v", err)
					}
				},
				sheet: guide, status: finding.Dismissed, resolution: "ignored", resolvedBy: finding.ResolvedByPerson,
			},
			{
				name:   "a third candidate reopens it",
				before: func(t *testing.T) { h.sitePage(t, "/c/guide/", "Guide", 33, nil) },
				sheet:  guide, status: finding.Open,
			},
			{
				name: "a row one page now answers is cleared as matched",
				before: func(t *testing.T) {
					h.retitle(t, "/b/guide/", "Second guide")
					h.retitle(t, "/c/guide/", "Third guide")
				},
				sheet: guide, status: finding.Resolved, resolution: finding.ResolutionRowMatched, resolvedBy: finding.ResolvedByImport,
			},
			{
				name:   "a cleared one reopens when it is raised again",
				before: func(t *testing.T) { h.retitle(t, "/b/guide/", "Guide") },
				sheet:  guide, status: finding.Open,
			},
			{
				name: "a row the sheet lost is cleared as gone", sheet: other,
				status: finding.Resolved, resolution: finding.ResolutionRowLeftSheet, resolvedBy: finding.ResolvedByImport,
			},
		}

		for _, step := range steps {
			if step.before != nil {
				step.before(t)
			}
			h.applyCSV(t, step.sheet, mapping)
			held := h.one(t, finding.RowAmbiguous)
			switch {
			case held.SubjectKey != key || held.PageID != nil:
				t.Fatalf("%s: the finding reads %s on %v, want %s on no page", step.name, held.SubjectKey, held.PageID, key)
			case held.Status != step.status || held.Resolution != step.resolution || held.ResolvedBy != step.resolvedBy:
				t.Fatalf("%s: the finding is %s (%q by %q), want %s (%q by %q)", step.name, held.Status, held.Resolution, held.ResolvedBy,
					step.status, step.resolution, step.resolvedBy)
			case step.subject != "" && string(held.Subject) != step.subject:
				t.Fatalf("%s: the subject = %s, want %s", step.name, held.Subject, step.subject)
			}
		}
	})

	t.Run("a row without a page names its planned page", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name    string
			sheet   string
			mapping func(harness) imports.Mapping
			key     pagemap.Row
			planned string
			product bool
		}{
			{
				name: "a page row", sheet: "title,keywords\nLoose guide,loose\n",
				mapping: func(h harness) imports.Mapping {
					return h.mapping(map[string]string{string(importmap.FieldTitle): "title", string(importmap.FieldKeywords): "keywords"})
				},
				key: pagemap.Row{Title: "Loose guide"}, planned: "/loose-guide/",
			},
			{
				name: "a product row waiting for its product", sheet: "path,h1,keywords\n/product/mak-liquid/,Mak Liquid,liquid mak\n",
				mapping: func(h harness) imports.Mapping { return productMapping(h, importmap.RowProducts) },
				key:     pagemap.Row{Path: "/product/mak-liquid/"}, planned: "/product/mak-liquid/", product: true,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				h := newHarness(t)
				mapping := tc.mapping(h)
				subject := `{"sheet":"","plannedPath":"` + tc.planned + `","product":false}`
				if tc.product {
					subject = `{"sheet":"","plannedPath":"` + tc.planned + `","product":true}`
				}
				for range 2 {
					h.applyCSV(t, tc.sheet, mapping)
					planned, found := h.pageAt(t, tc.planned)
					held := h.one(t, finding.RowWithoutPage)
					switch {
					case !found || planned.WPID != nil:
						t.Fatalf("pages = %+v, want %s planned", h.pages(t), tc.planned)
					case held.PageID == nil || *held.PageID != planned.ID || held.Status != finding.Open:
						t.Fatalf("the finding = %+v, want it open on the planned page %s", held, planned.ID)
					case held.SubjectKey != pagemap.RowKey("", tc.key) || string(held.Subject) != subject:
						t.Fatalf("the finding reads %s %s, want %s %s", held.SubjectKey, held.Subject, pagemap.RowKey("", tc.key), subject)
					case !slices.Equal(held.Options, []finding.Option{finding.MapToPage, finding.Ignore}):
						t.Fatalf("options = %v", held.Options)
					}
				}
			})
		}
	})

	t.Run("a planned page the site now holds clears its row", func(t *testing.T) {
		t.Parallel()

		h := newHarness(t)
		mapping := h.mapping(map[string]string{string(importmap.FieldTitle): "title", string(importmap.FieldKeywords): "keywords"})
		sheet := "title,keywords\nLoose guide,loose\n"
		h.applyCSV(t, sheet, mapping)
		planned, _ := h.pageAt(t, "/loose-guide/")
		wpID := int64(77)
		planned.WPID, planned.Status = &wpID, pagemap.StatusPublished
		if err := sqlite.NewPageRepo(h.store).UpdatePublished(t.Context(), planned); err != nil {
			t.Fatalf("publish the planned page: %v", err)
		}

		h.applyCSV(t, sheet, mapping)
		if held := h.one(t, finding.RowWithoutPage); held.Status != finding.Resolved || held.Resolution != finding.ResolutionRowMatched ||
			held.ResolvedBy != finding.ResolvedByImport {
			t.Fatalf("the finding = %+v, want it cleared as matched by the import", held)
		}
	})

	t.Run("a sheet's trail the site does not show", func(t *testing.T) {
		t.Parallel()

		h := newHarness(t)
		h.sitePage(t, "/peptides/", "Peptides", 21, nil)
		blends := h.sitePage(t, "/blends/", "Blends", 22, nil)
		bpc := h.childPage(t, "/bpc-157/", "BPC-157", 23, blends)
		mapping := trailMapping(h, "Category")

		steps := []struct {
			name       string
			sheet      string
			status     finding.Status
			resolution string
		}{
			{name: "raised", sheet: "Category,URL,Title\nPeptides,/bpc-157/,BPC-157\n", status: finding.Open},
			{name: "one row on a second import", sheet: "Category,URL,Title\nPeptides,/bpc-157/,BPC-157\n", status: finding.Open},
			{name: "a row with no trail clears nothing", sheet: "Category,URL,Title\n,/bpc-157/,BPC-157\n", status: finding.Open},
			{
				name: "cleared when the sheet agrees", sheet: "Category,URL,Title\nBlends,/bpc-157/,BPC-157\n",
				status: finding.Resolved, resolution: finding.ResolutionObservedClear,
			},
		}
		for _, step := range steps {
			h.applyCSV(t, step.sheet, mapping)
			held := h.one(t, finding.ParentDiffers)
			switch {
			case held.SubjectKey != finding.SubjectPage(bpc.ID) || held.PageID == nil || *held.PageID != bpc.ID:
				t.Fatalf("%s: the finding reads %s on %v, want the page %s", step.name, held.SubjectKey, held.PageID, bpc.ID)
			case string(held.Subject) != `{"sheet":["Peptides"],"site":["Blends"]}`:
				t.Fatalf("%s: the subject = %s, want the observation's {sheet, site}", step.name, held.Subject)
			case !slices.Equal(held.Options, pagemap.TrailOptions(pagemap.WPPage)):
				t.Fatalf("%s: options = %v", step.name, held.Options)
			case held.Status != step.status || held.Resolution != step.resolution:
				t.Fatalf("%s: the finding is %s (%q), want %s (%q)", step.name, held.Status, held.Resolution, step.status, step.resolution)
			case step.status == finding.Resolved && held.ResolvedBy != finding.ResolvedByImport:
				t.Fatalf("%s: resolved by %q, want the import", step.name, held.ResolvedBy)
			}
		}
	})

	t.Run("a product row's trail is compared with its categories", func(t *testing.T) {
		t.Parallel()

		h := newHarness(t)
		product := h.storeProduct(t, "bpc-liquid", "BPC liquid", 501)
		peptides := int64(90)
		h.termPage(t, "/product-category/peptides/", "Peptides", peptides, nil)
		h.termPage(t, "/product-category/peptides/liquids/", "Liquids", 100, &peptides)
		if err := sqlite.NewPageIdentityRepo(h.store).ReplaceTerms(t.Context(), product.ID, []pagemap.Term{
			{Taxonomy: string(pagemap.WPProductCategory), TermID: 100, Name: "Liquids", Slug: "liquids"},
		}); err != nil {
			t.Fatalf("give the product its category: %v", err)
		}
		mapping := productMapping(h, importmap.RowProducts)
		mapping.Columns = map[string]string{string(importmap.FieldPath): "URL", string(importmap.FieldH1): "H1"}
		mapping.Options.LevelColumns = []string{"Category", "Subcategory"}

		h.applyCSV(t, "Category,Subcategory,URL,H1\nLiquids,,/product/bpc-liquid/,BPC liquid\n", mapping)
		held := h.one(t, finding.ParentDiffers)
		switch {
		case held.PageID == nil || *held.PageID != product.ID || held.Status != finding.Open:
			t.Fatalf("the finding = %+v, want it open on the product", held)
		case string(held.Subject) != `{"sheet":["Liquids"],"site":["Peptides","Liquids"]}`:
			t.Fatalf("the subject = %s, want the category's chain the site holds", held.Subject)
		case !slices.Equal(held.Options, pagemap.TrailOptions(pagemap.WPProduct)):
			t.Fatalf("options = %v, want a product's", held.Options)
		}

		h.applyCSV(t, "Category,Subcategory,URL,H1\nPeptides,Liquids,/product/bpc-liquid/,BPC liquid\n", mapping)
		if held = h.one(t, finding.ParentDiffers); held.Status != finding.Resolved || held.Resolution != finding.ResolutionObservedClear {
			t.Fatalf("the finding = %+v, want it cleared once the sheet names the category's chain", held)
		}
	})

	t.Run("a preview persists nothing", func(t *testing.T) {
		t.Parallel()

		h := newHarness(t)
		h.sitePage(t, "/a/guide/", "Guide", 31, nil)
		h.sitePage(t, "/b/guide/", "Guide", 32, nil)
		blends := h.sitePage(t, "/blends/", "Blends", 22, nil)
		h.childPage(t, "/bpc-157/", "BPC-157", 23, blends)
		mapping := trailMapping(h, "Category")
		report := h.preview(t, h.file(t, "rows.csv", "Category,URL,Title\n,,Guide\nPeptides,/bpc-157/,BPC-157\n,,Loose guide\n"), mapping)
		if len(report.Warnings) == 0 {
			t.Fatal("the preview raised nothing, want its warnings")
		}
		listed, err := sqlite.NewFindingRepo(h.store).ListBySite(t.Context(), h.siteID, "")
		if err != nil || len(listed) != 0 {
			t.Fatalf("findings after a preview = %+v (%v), want none", listed, err)
		}
	})

	t.Run("a sheet the import did not read keeps its findings", func(t *testing.T) {
		t.Parallel()

		h := newHarness(t)
		elsewhere, err := finding.New(finding.Finding{
			ID: id.New(), SiteID: h.siteID, Code: finding.RowAmbiguous, SubjectKey: pagemap.RowKeyPrefix("Elsewhere") + "guide",
			CreatedAt: settledAt, UpdatedAt: settledAt,
		})
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		if err = sqlite.NewFindingRepo(h.store).Upsert(t.Context(), elsewhere); err != nil {
			t.Fatalf("Upsert: %v", err)
		}
		h.applyCSV(t, "title,keywords\nOther,other\n",
			h.mapping(map[string]string{string(importmap.FieldTitle): "title", string(importmap.FieldKeywords): "keywords"}))
		if held := h.one(t, finding.RowAmbiguous); held.Status != finding.Open {
			t.Fatalf("the other sheet's finding = %+v, want it untouched", held)
		}
	})
}
