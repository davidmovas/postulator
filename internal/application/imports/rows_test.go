package imports_test

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/keyword"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

var observedAt = time.Date(2026, time.October, 20, 8, 0, 0, 0, time.UTC)

func (h harness) siteEntity(t *testing.T, name string) graph.Entity {
	t.Helper()

	entity, err := graph.NewEntity(graph.Entity{
		ID: id.New(), SiteID: h.siteID, Name: name, Source: graph.SourceImport, CreatedAt: observedAt, UpdatedAt: observedAt,
	})
	if err != nil {
		t.Fatalf("NewEntity: %v", err)
	}
	if err = sqlite.NewEntityRepo(h.store).Insert(t.Context(), entity); err != nil {
		t.Fatalf("insert the entity: %v", err)
	}
	return entity
}

func (h harness) sitePage(t *testing.T, path, title string, wpID int64, entityID *string) pagemap.Page {
	t.Helper()

	page, err := pagemap.NewPage(pagemap.Page{
		ID: id.New(), SiteID: h.siteID, Path: path, WPType: pagemap.WPPage, WPID: &wpID, Title: title, EntityID: entityID,
		Status: pagemap.StatusPublished, CreatedAt: observedAt, UpdatedAt: observedAt,
	})
	if err != nil {
		t.Fatalf("NewPage: %v", err)
	}
	if err = sqlite.NewPageRepo(h.store).Insert(t.Context(), page); err != nil {
		t.Fatalf("insert the page: %v", err)
	}
	return page
}

func levelMapping(h harness, columns map[string]string) imports.Mapping {
	mapping := h.mapping(columns)
	mapping.Options.LevelColumns = []string{"Root Entity"}
	return mapping
}

func TestARowWithoutAURLUpdatesThePageItNames(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		sheet   string
		columns map[string]string
		seed    func(*testing.T, harness)
		want    string
		rule    string
	}{
		{
			name:    "by its title",
			sheet:   "title,keywords\nAbout us,about us\n",
			columns: map[string]string{string(importmap.FieldTitle): "title", string(importmap.FieldKeywords): "keywords"},
			seed:    func(t *testing.T, h harness) { h.sitePage(t, "/about/", "About us", 11, nil) },
			want:    "/about/",
			rule:    "title",
		},
		{
			name:    "by its entity's name",
			sheet:   "entity,keywords\nShoes,shoes\n",
			columns: map[string]string{string(importmap.FieldEntity): "entity", string(importmap.FieldKeywords): "keywords"},
			seed: func(t *testing.T, h harness) {
				shoes := h.siteEntity(t, "Shoes")
				h.sitePage(t, "/footwear/", "Footwear", 12, &shoes.ID)
			},
			want: "/footwear/",
			rule: "entity",
		},
		{
			name:    "by its heading",
			sheet:   "h1,keywords\nQuestions,faq\n",
			columns: map[string]string{string(importmap.FieldH1): "h1", string(importmap.FieldKeywords): "keywords"},
			seed:    func(t *testing.T, h harness) { h.sitePage(t, "/faq/", "Questions", 13, nil) },
			want:    "/faq/",
			rule:    "h1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			tc.seed(t, h)
			path, mapping := h.file(t, "rows.csv", tc.sheet), h.mapping(tc.columns)

			report := h.preview(t, path, mapping)
			view, found := page(report, tc.want)
			if !found || view.Action != string(imports.ActionUpdate) || view.MatchedBy != tc.rule || len(report.Pages) != 1 {
				t.Fatalf("pages = %+v, want %s updated, matched by its %s", report.Pages, tc.want, tc.rule)
			}
			if left := findings(report.Warnings, imports.CodeRowWithoutPage); len(left) != 0 {
				t.Fatalf("rows without a page = %+v, want none", left)
			}

			applied := h.apply(t, path, mapping)
			if applied.Counts.PagesCreated != 0 || applied.Counts.PagesUpdated != 1 {
				t.Fatalf("counts = %+v, want the page the row names updated and none created", applied.Counts)
			}
			if stored, _ := h.pageAt(t, tc.want); len(stored.Keywords) != 1 || stored.WPID == nil {
				t.Fatalf("the page reads %+v, want the row's keyword on it", stored)
			}
		})
	}
}

func TestARowWithoutAURLOrAPageIsPlannedUnderTheLevelItNames(t *testing.T) {
	t.Parallel()

	leveled := map[string]string{string(importmap.FieldTitle): "Title", string(importmap.FieldKeywords): "Keywords"}
	cases := []struct {
		name    string
		sheet   string
		columns map[string]string
		want    string
		parent  bool
	}{
		{
			name: "the slug of its title under the page its level names", sheet: "Root Entity,Title,Keywords\nPeptides,BPC-157 guide,bpc guide\n",
			columns: leveled, want: "/peptides/bpc-157-guide/", parent: true,
		},
		{
			name: "the slug of its heading when it has no title", sheet: "Root Entity,Entity,H1\nPeptides,TB-500,TB-500 dosing\n",
			columns: map[string]string{string(importmap.FieldEntity): "Entity", string(importmap.FieldH1): "H1"},
			want:    "/peptides/tb-500-dosing/", parent: true,
		},
		{
			name: "at the root when its level names no page", sheet: "Root Entity,Title,Keywords\nNowhere,Loose guide,loose\n",
			columns: leveled, want: "/loose-guide/",
		},
		{
			name: "at the root without a level", sheet: "Root Entity,Title,Keywords\n,Loose guide,loose\n",
			columns: leveled, want: "/loose-guide/",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			peptides := h.sitePage(t, "/peptides/", "Peptides", 21, nil)
			path, mapping := h.file(t, "rows.csv", tc.sheet), levelMapping(h, tc.columns)

			report := h.preview(t, path, mapping)
			if view, found := page(report, tc.want); !found || view.Action != string(imports.ActionCreate) || view.MatchedBy != "" {
				t.Fatalf("pages = %+v, want %s planned", report.Pages, tc.want)
			}
			if left := findings(report.Warnings, imports.CodeRowWithoutPage); len(left) != 1 || left[0].Row != 2 ||
				!strings.Contains(left[0].Message, tc.want) {
				t.Fatalf("rows without a page = %+v, want the row named with its address", left)
			}

			h.apply(t, path, mapping)
			stored, found := h.pageAt(t, tc.want)
			if !found || stored.WPID != nil || stored.Status != pagemap.StatusPlanned {
				t.Fatalf("the planned page reads %+v (%t)", stored, found)
			}
			switch {
			case tc.parent && (stored.ParentPageID == nil || *stored.ParentPageID != peptides.ID):
				t.Fatalf("the planned page's parent = %v, want the level's page %s", stored.ParentPageID, peptides.ID)
			case !tc.parent && stored.ParentPageID != nil:
				t.Fatalf("the planned page's parent = %s, want the site root", *stored.ParentPageID)
			}
		})
	}
}

func TestARowThatNamesOnlyAnEntityPlansNoPage(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		sheet   func(*testing.T, harness) (string, imports.Mapping)
		created int
		updated int
	}{
		{
			name: "a plain sheet row",
			sheet: func(t *testing.T, h harness) (string, imports.Mapping) {
				return h.file(t, "rows.csv", "entity,keywords\nSupport,support desk\n"),
					h.mapping(map[string]string{string(importmap.FieldEntity): "entity", string(importmap.FieldKeywords): "keywords"})
			},
			created: 1,
		},
		{
			name: "an exported entity without a page, imported again",
			sheet: func(t *testing.T, h harness) (string, imports.Mapping) {
				source := newHarness(t)
				support, err := graph.NewEntity(graph.Entity{
					ID: id.New(), SiteID: source.siteID, Name: "Support", Keywords: keyword.New([]keyword.Keyword{{Text: "support desk"}}),
					Source: graph.SourceImport, CreatedAt: observedAt, UpdatedAt: observedAt,
				})
				if err != nil {
					t.Fatalf("NewEntity: %v", err)
				}
				if err = sqlite.NewEntityRepo(source.store).Insert(t.Context(), support); err != nil {
					t.Fatalf("insert the entity: %v", err)
				}
				exported := filepath.Join(source.dir, "export.xlsx")
				if _, err = source.service.Export(t.Context(), imports.ExportRequest{SiteID: source.siteID, Path: exported}); err != nil {
					t.Fatalf("Export: %v", err)
				}
				h.siteEntity(t, "Support")
				return exported, h.detected(t, exported)
			},
			updated: 1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			path, mapping := tc.sheet(t, h)
			report := h.preview(t, path, mapping)
			if len(report.Pages) != 0 || len(findings(report.Warnings, imports.CodeRowWithoutPage)) != 0 {
				t.Fatalf("the preview plans %+v and warns %+v, want no page for a row that names only an entity", report.Pages, report.Warnings)
			}

			applied := h.apply(t, path, mapping)
			if applied.Counts.PagesCreated != 0 || applied.Counts.EntitiesCreated != tc.created || applied.Counts.EntitiesUpdated != tc.updated {
				t.Fatalf("counts = %+v, want the entity written and no page", applied.Counts)
			}
			if held := h.pages(t); len(held) != 0 {
				t.Fatalf("pages = %+v, want none", held)
			}
			if held := h.lasting(t, finding.RowWithoutPage); len(held) != 0 {
				t.Fatalf("row findings = %+v, want none", held)
			}
			for _, held := range h.entities(t) {
				if held.Name == "Support" && held.Keywords.Main() != "support desk" {
					t.Fatalf("Support reads %+v, want the row's keyword on it", held)
				}
			}
		})
	}
}

func TestARowTwoPagesAnswerWritesNothing(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.sitePage(t, "/a/guide/", "Guide", 31, nil)
	h.sitePage(t, "/b/guide/", "Guide", 32, nil)
	path := h.file(t, "rows.csv", "title,entity,keywords\nGuide,Guide,guide\nOther,Other,other\n")
	mapping := h.mapping(map[string]string{
		string(importmap.FieldTitle): "title", string(importmap.FieldEntity): "entity", string(importmap.FieldKeywords): "keywords",
	})

	report := h.preview(t, path, mapping)
	ambiguous := findings(report.Warnings, imports.CodeRowAmbiguous)
	if len(ambiguous) != 1 || ambiguous[0].Row != 2 || !strings.Contains(ambiguous[0].Message, "/a/guide/") ||
		!strings.Contains(ambiguous[0].Message, "/b/guide/") {
		t.Fatalf("ambiguous rows = %+v, want the guide row naming both pages", ambiguous)
	}
	if imports.CodeRowAmbiguous.Blocking() || len(report.Errors) != 0 {
		t.Fatalf("errors = %+v, want an ambiguous row to leave the rest of the sheet to apply", report.Errors)
	}
	if _, named := entity(report, "Guide"); named {
		t.Fatalf("entities = %+v, want none for the ambiguous row", report.Entities)
	}

	applied := h.apply(t, path, mapping)
	if applied.Counts.PagesCreated != 1 || applied.Counts.EntitiesCreated != 1 || applied.Counts.PagesUpdated != 0 {
		t.Fatalf("counts = %+v, want the other row alone written", applied.Counts)
	}
	for _, held := range h.entities(t) {
		if held.Name == "Guide" {
			t.Fatalf("the ambiguous row wrote the entity %+v", held)
		}
	}
	for _, held := range h.pages(t) {
		if held.EntityID != nil && held.Path != "/other/" {
			t.Fatalf("the ambiguous row wrote onto %+v", held)
		}
	}
}

func TestAURLUnderOneParentNeverTakesAPageUnderAnotherByItsSlugOrName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		seed   func(*testing.T, harness)
		sheet  string
		want   string
		action imports.Action
		rule   string
	}{
		{
			name: "a slug the page of another parent carries",
			seed: func(t *testing.T, h harness) {
				h.sitePage(t, "/tb-500/", "TB-500", 51, nil)
				h.sitePage(t, "/tb-500/liquid/", "Liquid", 52, nil)
			},
			sheet: "URL,Title\n/bpc-157/,BPC-157\n/bpc-157/liquid/,Liquid\n", want: "/bpc-157/liquid/", action: imports.ActionCreate,
		},
		{
			name: "a slug the pages of two other parents carry",
			seed: func(t *testing.T, h harness) {
				h.sitePage(t, "/tb-500/", "TB-500", 51, nil)
				h.sitePage(t, "/tb-500/liquid/", "TB-500 liquid", 52, nil)
				h.sitePage(t, "/cjc/", "CJC", 53, nil)
				h.sitePage(t, "/cjc/liquid/", "CJC liquid", 54, nil)
			},
			sheet: "URL,Title\n/bpc-157/,BPC-157\n/bpc-157/liquid/,BPC-157 liquid\n", want: "/bpc-157/liquid/", action: imports.ActionCreate,
		},
		{
			name: "a nested sheet URL beside a flat page of the site",
			seed: func(t *testing.T, h harness) {
				h.sitePage(t, "/peptides/", "Peptides", 51, nil)
				h.sitePage(t, "/bpc-157/", "BPC-157", 52, nil)
			},
			sheet: "URL,Title\n/peptides/bpc-157/,BPC-157 peptide\n", want: "/bpc-157/", action: imports.ActionUpdate, rule: "slug",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			tc.seed(t, h)
			report := h.preview(t, h.file(t, "rows.csv", tc.sheet),
				h.mapping(map[string]string{string(importmap.FieldPath): "URL", string(importmap.FieldTitle): "Title"}))
			if view, found := page(report, tc.want); !found || view.Action != string(tc.action) || view.MatchedBy != tc.rule {
				t.Fatalf("pages = %+v, want %s %s matched by %q", report.Pages, tc.want, tc.action, tc.rule)
			}
			if ambiguous := findings(report.Warnings, imports.CodeRowAmbiguous); len(ambiguous) != 0 {
				t.Fatalf("ambiguous rows = %+v, want the pages of other parents passed over", ambiguous)
			}
		})
	}
}

func (h harness) plannedPage(t *testing.T, path, title string) pagemap.Page {
	t.Helper()

	page, err := pagemap.NewPage(pagemap.Page{
		ID: id.New(), SiteID: h.siteID, Path: path, WPType: pagemap.WPPage, Title: title,
		Status: pagemap.StatusPlanned, CreatedAt: observedAt, UpdatedAt: observedAt,
	})
	if err != nil {
		t.Fatalf("NewPage: %v", err)
	}
	if err = sqlite.NewPageRepo(h.store).Insert(t.Context(), page); err != nil {
		t.Fatalf("insert the planned page: %v", err)
	}
	return page
}

func TestOnlyAPlannedPageFillsTheGapsInItsPath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		seed   func(*testing.T, harness)
		sheet  string
		gaps   []string
		absent []string
	}{
		{
			name: "a row matched by its slug at another address",
			seed: func(t *testing.T, h harness) {
				guide := h.sitePage(t, "/guide/", "Guide", 61, nil)
				h.childPage(t, "/guide/setup/", "Setup", 62, guide)
			},
			sheet:  "URL,Title\n/guide/,Guide\n/brewing/setup/,Setting up your grinder\n",
			absent: []string{"/brewing/"},
		},
		{
			name:   "a row matched by its URL on the site",
			seed:   func(t *testing.T, h harness) { h.sitePage(t, "/journal/hello/", "Hello", 63, nil) },
			sheet:  "URL,Title\n/journal/hello/,Hello\n",
			absent: []string{"/journal/"},
		},
		{
			name:   "a row matched by its slug to a page planned at another address",
			seed:   func(t *testing.T, h harness) { h.plannedPage(t, "/setup/", "Setup") },
			sheet:  "URL,Title\n/brewing/setup/,Setting up your grinder\n",
			absent: []string{"/brewing/"},
		},
		{
			name:  "a row matched by its URL to a page still planned",
			seed:  func(t *testing.T, h harness) { h.plannedPage(t, "/brewing/setup/", "Setup") },
			sheet: "URL,Title\n/brewing/setup/,Setting up your grinder\n",
			gaps:  []string{"/brewing/"},
		},
		{
			name:  "a row no page answers",
			seed:  func(*testing.T, harness) {},
			sheet: "URL,Title\n/brewing/cold/,Cold brew\n",
			gaps:  []string{"/brewing/"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			tc.seed(t, h)
			path := h.file(t, "rows.csv", tc.sheet)
			mapping := h.mapping(map[string]string{string(importmap.FieldPath): "URL", string(importmap.FieldTitle): "Title"})

			report := h.preview(t, path, mapping)
			generated := make([]string, 0)
			for i := range report.Pages {
				if report.Pages[i].Generated {
					generated = append(generated, report.Pages[i].Path)
				}
			}
			if !slices.Equal(generated, tc.gaps) {
				t.Fatalf("the preview fills the gaps %v, want %v: %+v", generated, tc.gaps, report.Pages)
			}
			if filled := findings(report.Warnings, imports.CodeIntermediatePath); len(filled) != len(tc.gaps) {
				t.Fatalf("intermediate findings = %+v, want %d", filled, len(tc.gaps))
			}

			h.apply(t, path, mapping)
			for _, gap := range tc.gaps {
				if stored, found := h.pageAt(t, gap); !found || stored.Status != pagemap.StatusPlanned {
					t.Fatalf("the gap %s reads %+v (%t), want it planned", gap, stored, found)
				}
			}
			for _, gap := range tc.absent {
				if stored, found := h.pageAt(t, gap); found {
					t.Fatalf("the import planned %+v, want no page at %s", stored, gap)
				}
			}
		})
	}
}

func TestAProductRowFindsItsProductByItsSKU(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		rows importmap.RowType
		want string
		rule string
	}{
		{name: "a product row", rows: importmap.RowProducts, want: "/product/mak-liquid/", rule: "sku"},
		{name: "a page row, which never reads a SKU", rows: importmap.RowPages, want: "/liquid-of-the-range/"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			liquid := h.storeProduct(t, "mak-liquid", "Mak Liquid", 501)
			if err := sqlite.NewPageFieldRepo(h.store).Observe(t.Context(), liquid.ID,
				[]field.Value{field.NewValue(liquid.ID, field.SKU, "MAK-L-10", observedAt)}); err != nil {
				t.Fatalf("observe the SKU: %v", err)
			}
			mapping := h.mapping(map[string]string{
				string(importmap.FieldSKU): "SKU", string(importmap.FieldH1): "h1", string(importmap.FieldKeywords): "keywords",
			})
			mapping.Options.RowType = tc.rows

			report := h.preview(t, h.file(t, "rows.csv", "SKU,h1,keywords\nMAK-L-10,Liquid of the range,mak liquid\n"), mapping)
			view, found := page(report, tc.want)
			switch {
			case !found || view.MatchedBy != tc.rule:
				t.Fatalf("pages = %+v, want %s matched by %q", report.Pages, tc.want, tc.rule)
			case tc.rule != "" && (view.Action != string(imports.ActionUpdate) || view.StoreName != "Mak Liquid"):
				t.Fatalf("the product reads %+v, want the store's product updated", view)
			case tc.rule == "" && view.Action != string(imports.ActionCreate):
				t.Fatalf("the row reads %+v, want a planned page", view)
			}
		})
	}
}

func TestASecondImportOfTheSameWorkbookCreatesNothing(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.sitePage(t, "/peptides/", "Peptides", 41, nil)
	h.sitePage(t, "/about/", "About us", 42, nil)
	path := h.file(t, "plan.csv", "Root Entity,URL,Title,Keywords\n"+
		"Peptides,/peptides/bpc-157/,BPC-157,bpc 157\n"+
		"Peptides,,BPC-157 guide,bpc guide\n"+
		",,About us,about us\n"+
		",,Loose guide,loose\n")
	mapping := levelMapping(h, map[string]string{
		string(importmap.FieldPath): "URL", string(importmap.FieldTitle): "Title", string(importmap.FieldKeywords): "Keywords",
	})

	first := h.apply(t, path, mapping)
	if first.Counts.PagesCreated != 3 || len(first.Report.Errors) != 0 {
		t.Fatalf("the first import = %+v %+v, want BPC-157, its guide and the loose guide planned", first.Counts, first.Report.Errors)
	}
	if _, found := h.pageAt(t, "/peptides/bpc-157-guide/"); !found {
		t.Fatalf("pages = %+v, want the guide under the level's page", h.pages(t))
	}

	again := h.apply(t, path, mapping)
	if again.Counts != (imports.Counts{}) {
		t.Fatalf("the second import = %+v, want nothing to do", again.Counts)
	}
	if left := findings(again.Report.Warnings, imports.CodeRowWithoutPage); len(left) != 0 {
		t.Fatalf("rows without a page on the second import = %+v, want every row to find the page the first planned", left)
	}
	for _, view := range again.Report.Pages {
		if view.MatchedBy == "" {
			t.Errorf("the second import shows %+v unmatched", view)
		}
	}
}
