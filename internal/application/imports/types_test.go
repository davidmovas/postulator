package imports_test

import (
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func typeMapping(h harness) imports.Mapping {
	return h.mapping(map[string]string{string(importmap.FieldPath): "url", string(importmap.FieldPageType): "page type"})
}

func (h harness) typedPage(t *testing.T, path string, typed *pagetype.Type, source pagemap.TypeSource, reason pagemap.TypeReason) pagemap.Page {
	t.Helper()

	wpID := int64(len(path))
	held := pagemap.Page{
		ID: id.New(), SiteID: h.siteID, Path: path, WPType: pagemap.WPPage, WPID: &wpID, Title: "Mugs",
		Status: pagemap.StatusPublished, TypeSource: source, TypeReason: reason,
		CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	}
	if typed != nil {
		held.PageTypeID = &typed.ID
	}
	built, err := pagemap.NewPage(held)
	if err != nil {
		t.Fatalf("build the page: %v", err)
	}
	if err = sqlite.NewPageRepo(h.store).Insert(t.Context(), built); err != nil {
		t.Fatalf("insert the page: %v", err)
	}
	return built
}

func TestThePageTypeOfAPageOnTheSiteFollowsTheSheetUnlessAPersonOrTheAgentChoseIt(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		typed  bool
		source pagemap.TypeSource
		reason pagemap.TypeReason
		kept   bool
	}{
		{name: "an untyped page", source: pagemap.TypeSourceNone, reason: pagemap.TypeByNone},
		{name: "a page the site typed", typed: true, source: pagemap.TypeSourceAuto, reason: pagemap.TypeByDefault},
		{name: "a page a listing typed", typed: true, source: pagemap.TypeSourceAuto, reason: pagemap.TypeByListing},
		{name: "a page an earlier import typed", typed: true, source: pagemap.TypeSourceManual, reason: pagemap.TypeByImport},
		{name: "a page a person typed", typed: true, source: pagemap.TypeSourceManual, reason: pagemap.TypeByUser, kept: true},
		{name: "a page the agent typed", typed: true, source: pagemap.TypeSourceManual, reason: pagemap.TypeByAI, kept: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			types := h.types(t)
			var current *pagetype.Type
			if tc.typed {
				entity := types[pagetype.Entity]
				current = &entity
			}
			held := h.typedPage(t, "/mugs/", current, tc.source, tc.reason)
			path := h.file(t, "types.csv", "url,page type\n/mugs/,Pillar\n")

			report := h.preview(t, path, typeMapping(h))
			view, _ := page(report, held.Path)
			kept := findings(report.Warnings, imports.CodePageTypeKept)
			stored, _ := h.pageAt(t, held.Path)
			if !sameType(stored.PageTypeID, held.PageTypeID) || stored.TypeReason != tc.reason {
				t.Fatalf("the preview wrote the type %+v", stored)
			}

			applied := h.apply(t, path, typeMapping(h))
			after, _ := h.pageAt(t, held.Path)
			if tc.kept {
				if view.PageType != "Entity" || len(kept) != 1 || !strings.Contains(kept[0].Message, "Entity") ||
					!strings.Contains(kept[0].Message, "Hub") || kept[0].Field != string(importmap.FieldPageType) {
					t.Fatalf("the preview shows %+v with %+v, want the chosen type kept and both types named", view, kept)
				}
				if !sameType(after.PageTypeID, held.PageTypeID) || after.TypeReason != tc.reason {
					t.Fatalf("the apply retyped the page a person or the agent typed: %+v", after)
				}
				return
			}
			if view.PageType != "Hub" || view.Action != string(imports.ActionUpdate) || len(kept) != 0 {
				t.Fatalf("the preview shows %+v with %+v, want the sheet's Hub", view, kept)
			}
			hub := types[pagetype.Hub]
			if !sameType(after.PageTypeID, &hub.ID) || after.TypeSource != pagemap.TypeSourceManual ||
				after.TypeReason != pagemap.TypeByImport {
				t.Fatalf("the apply typed the page %+v, want the Hub set by the import", after)
			}
			if applied.Counts.PagesUpdated != 1 {
				t.Fatalf("counts = %+v, want the retyped page counted as updated", applied.Counts)
			}
			if again := h.apply(t, path, typeMapping(h)); again.Counts.PagesUpdated != 0 || again.Counts.PagesCreated != 0 {
				t.Fatalf("a second import of the same sheet wrote %+v", again.Counts)
			}
		})
	}
}

func sameType(a, b *string) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && *a == *b)
}

func TestAPlannedPageIsInsertedWithTheSheetsType(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		cell   string
		preset pagetype.Preset
		typed  bool
	}{
		{name: "a hub", cell: "Hub", preset: pagetype.Hub, typed: true},
		{name: "a synonym of a product", cell: "Compound", preset: pagetype.Product, typed: true},
		{name: "an empty cell", cell: ""},
		{name: "a type the site does not hold", cell: "Sprocket"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			applied := h.apply(t, h.file(t, "planned.csv", "url,page type\n/new/,"+tc.cell+"\n"), typeMapping(h))
			if applied.Counts.PagesCreated != 1 {
				t.Fatalf("counts = %+v", applied.Counts)
			}
			stored, _ := h.pageAt(t, "/new/")
			if !tc.typed {
				if stored.PageTypeID != nil || stored.TypeSource != pagemap.TypeSourceNone || stored.TypeReason != pagemap.TypeByNone {
					t.Fatalf("the planned page reads %+v, want it untyped", stored)
				}
				return
			}
			want := h.types(t)[tc.preset]
			if !sameType(stored.PageTypeID, &want.ID) || stored.TypeSource != pagemap.TypeSourceManual ||
				stored.TypeReason != pagemap.TypeByImport || stored.Status != pagemap.StatusPlanned || stored.TemplateID != nil {
				t.Fatalf("the planned page reads %+v, want the %s type set by the import", stored, tc.preset)
			}
		})
	}
}

func TestARowWithoutAURLTypesThePageItPlans(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	mapping := h.mapping(map[string]string{string(importmap.FieldEntity): "entity", string(importmap.FieldPageType): "page type"})
	applied := h.apply(t, h.file(t, "entities.csv", "entity,page type\nTea,Sprocket\nCoffee,Hub\n"), mapping)
	if left := besidesPlanned(applied.Report.Warnings); len(applied.Report.Errors) != 0 || len(left) != 1 ||
		left[0].Code != string(imports.CodeUnknownPageType) || left[0].Row != 2 {
		t.Fatalf("findings = %+v %+v, want the type no page type answers to named on its row", left, applied.Report.Errors)
	}
	if applied.Counts.EntitiesCreated != 2 || applied.Counts.PagesCreated != 2 {
		t.Fatalf("counts = %+v, want a page planned for each entity", applied.Counts)
	}
	if typed := h.typeNames(t); len(typed) != 1 || typed["/coffee/"] != "Hub" {
		t.Fatalf("types = %v, want the coffee page typed by its cell and the tea page left untyped", typed)
	}
}

func TestTheTypeAnEarlierImportSetGivesWayToTheNextSheet(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	first := h.apply(t, h.file(t, "first.csv", "url,page type\n/a/,Hub\n"), typeMapping(h))
	if first.Counts.PagesCreated != 1 {
		t.Fatalf("counts = %+v", first.Counts)
	}

	second := h.apply(t, h.file(t, "again.csv", "url,page type\n/a/,Product\n"), typeMapping(h))
	if second.Counts.PagesUpdated != 1 || len(findings(second.Report.Warnings, imports.CodePageTypeKept)) != 0 {
		t.Fatalf("the second sheet = %+v %+v, want the import's own type replaced", second.Counts, second.Report.Warnings)
	}
	if typed := h.typeNames(t); typed["/a/"] != "Product" {
		t.Fatalf("types = %v, want the second sheet's Product", typed)
	}
}
