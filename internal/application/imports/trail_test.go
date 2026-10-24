package imports_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func (h harness) childPage(t *testing.T, path, title string, wpID int64, parent pagemap.Page) pagemap.Page {
	t.Helper()

	page, err := pagemap.NewPage(pagemap.Page{
		ID: id.New(), SiteID: h.siteID, Path: path, WPType: pagemap.WPPage, WPID: &wpID, Title: title,
		ParentPageID: &parent.ID, ParentReason: pagemap.ParentWP, Status: pagemap.StatusPublished,
		CreatedAt: observedAt, UpdatedAt: observedAt,
	})
	if err != nil {
		t.Fatalf("NewPage: %v", err)
	}
	if err = sqlite.NewPageRepo(h.store).Insert(t.Context(), page); err != nil {
		t.Fatalf("insert the page: %v", err)
	}
	return page
}

func trailMapping(h harness, levels ...string) imports.Mapping {
	mapping := h.mapping(map[string]string{string(importmap.FieldPath): "URL", string(importmap.FieldTitle): "Title"})
	mapping.Options.LevelColumns = levels
	return mapping
}

func TestThePreviewNamesTheSheetsTrailAndTheSitesWhereTheyDiffer(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	peptides := h.sitePage(t, "/peptides/", "Peptides", 21, nil)
	blends := h.sitePage(t, "/blends/", "Blends", 22, nil)
	h.childPage(t, "/bpc-157/", "BPC-157", 23, blends)
	h.childPage(t, "/tb-500/", "TB-500", 24, peptides)
	h.sitePage(t, "/loose/", "Loose", 25, nil)
	path := h.file(t, "trail.csv", "Category,Subcategory,URL,Title\n"+
		"Peptides,,/bpc-157/,BPC-157\n"+
		"Peptides,,/tb-500/,TB-500\n"+
		"Peptides,Blends,/loose/,Loose\n"+
		"Peptides,TB-500,/tb-500/liquid/,TB-500 liquid\n"+
		",,/peptides/,Peptides\n")
	mapping := trailMapping(h, "Category", "Subcategory")

	report := h.preview(t, path, mapping)
	differs := findings(report.Warnings, imports.CodeParentDiffers)
	if len(differs) != 2 {
		t.Fatalf("parent_differs = %+v, want the two matched rows the site places elsewhere", differs)
	}
	want := map[int]struct{ subject, sheet, site string }{
		2: {subject: `{"sheet":["Peptides"],"site":["Blends"]}`, sheet: "under Peptides", site: "under Blends"},
		4: {subject: `{"sheet":["Peptides","Blends"],"site":[]}`, sheet: "under Peptides › Blends", site: "at the top of the site"},
	}
	for _, found := range differs {
		expected, named := want[found.Row]
		switch {
		case !named:
			t.Errorf("row %d raised %+v", found.Row, found)
		case string(found.Subject) != expected.subject:
			t.Errorf("row %d subject = %s, want %s", found.Row, found.Subject, expected.subject)
		case !strings.Contains(found.Message, expected.sheet) || !strings.Contains(found.Message, expected.site):
			t.Errorf("row %d says %q, want both lists named", found.Row, found.Message)
		}
	}
	if imports.CodeParentDiffers.Blocking() || len(report.Errors) != 0 {
		t.Fatalf("errors = %+v, want a differing trail to leave the import free", report.Errors)
	}

	h.apply(t, path, mapping)
	for at, trail := range map[string][]string{
		"/bpc-157/": {"Peptides"}, "/tb-500/": {"Peptides"}, "/loose/": {"Peptides", "Blends"},
		"/tb-500/liquid/": {"Peptides", "TB-500"}, "/peptides/": nil,
	} {
		if got := h.trailAt(t, at); !slices.Equal(got, trail) {
			t.Errorf("the stored trail of %s = %q, want %q", at, got, trail)
		}
	}
	if moved, _ := h.pageAt(t, "/bpc-157/"); moved.ParentPageID == nil || *moved.ParentPageID != blends.ID {
		t.Fatalf("the parent of /bpc-157/ = %v, want the site's parent kept: the sheet's levels are never written", moved.ParentPageID)
	}

	again := h.apply(t, path, mapping)
	if again.Counts.PagesUpdated != 0 || again.Counts.PagesCreated != 0 {
		t.Fatalf("a second import of the same trails wrote %+v", again.Counts)
	}
}

func TestARowWithoutAURLIsPlannedUnderItsNearestLevel(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.sitePage(t, "/peptides/", "Peptides", 21, nil)
	bpc := h.sitePage(t, "/bpc-157/", "BPC-157", 22, nil)
	path := h.file(t, "trail.csv", "Root Entity,Category,Title\nPeptides,BPC-157,BPC-157 guide\nPeptides,BPC-157,BPC-157\n")
	mapping := h.mapping(map[string]string{string(importmap.FieldTitle): "Title"})
	mapping.Options.LevelColumns = []string{"Root Entity", "Category"}

	h.apply(t, path, mapping)
	guide, found := h.pageAt(t, "/bpc-157/bpc-157-guide/")
	if !found {
		t.Fatalf("pages = %+v, want the guide planned under the page its nearest level names", h.pages(t))
	}
	if guide.ParentPageID == nil || *guide.ParentPageID != bpc.ID || guide.ParentReason != pagemap.ParentWorkbook {
		t.Fatalf("the guide's parent = %v (%s), want the level's page by the workbook", guide.ParentPageID, guide.ParentReason)
	}
	if !slices.Equal(guide.PlannedTrail, []string{"Peptides", "BPC-157"}) {
		t.Fatalf("the guide's trail = %q, want both levels", guide.PlannedTrail)
	}
	if named := h.trailAt(t, "/bpc-157/"); !slices.Equal(named, []string{"Peptides"}) {
		t.Fatalf("the trail of the row its own level names = %q, want the level dropped", named)
	}
}
