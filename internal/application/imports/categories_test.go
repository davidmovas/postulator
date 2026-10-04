package imports_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/domain/category"
)

func (h harness) categories(t *testing.T) []category.Category {
	t.Helper()

	listed, err := sqlite.NewCategoryRepo(h.store).ListBySite(t.Context(), h.siteID)
	if err != nil {
		t.Fatalf("list the categories: %v", err)
	}
	return listed
}

func names(chain []category.Category) string {
	out := make([]string, 0, len(chain))
	for i := range chain {
		out = append(out, chain[i].Name)
	}
	return strings.Join(out, " › ")
}

func (h harness) shelf(t *testing.T) []string {
	t.Helper()

	held := h.categories(t)
	out := make([]string, 0, len(held))
	for i := range held {
		out = append(out, names(category.Chain(held, held[i].ID)))
	}
	slices.Sort(out)
	return out
}

func (h harness) filed(t *testing.T) map[string]string {
	t.Helper()

	held, stored := h.categories(t), h.pages(t)
	out := make(map[string]string, len(stored))
	for i := range stored {
		out[stored[i].Path] = names(category.Chain(held, stored[i].CategoryID))
	}
	return out
}

func previewed(report imports.PreviewReport) []string {
	out := make([]string, 0, len(report.Categories))
	for _, listed := range report.Categories {
		out = append(out, fmt.Sprintf("%s | %s | %d", strings.Join(listed.Path, " › "), listed.Action, listed.Rows))
	}
	return out
}

func TestAChainFilesThePageUnderItsLastCategory(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		sheet   string
		shelf   []string
		filed   map[string]string
		preview []string
	}{
		{
			name:  "a chain creates its categories parent first",
			sheet: "category,subcategory,url\nBPC-157,Liquid,/a/\nBPC-157,Powder,/b/\nBPC-157,,/c/\n,,/d/\n",
			shelf: []string{"BPC-157", "BPC-157 › Liquid", "BPC-157 › Powder"},
			filed: map[string]string{"/a/": "BPC-157 › Liquid", "/b/": "BPC-157 › Powder", "/c/": "BPC-157", "/d/": ""},
			preview: []string{
				"BPC-157 | create | 3", "BPC-157 › Liquid | create | 1", "BPC-157 › Powder | create | 1",
			},
		},
		{
			name:    "a name is matched by its key under the same parent",
			sheet:   "category,subcategory,url\nBPC-157,Liquid,/a/\nbpc-157 ,LIQUID,/b/\n",
			shelf:   []string{"BPC-157", "BPC-157 › Liquid"},
			filed:   map[string]string{"/a/": "BPC-157 › Liquid", "/b/": "BPC-157 › Liquid"},
			preview: []string{"BPC-157 | create | 2", "BPC-157 › Liquid | create | 2"},
		},
		{
			name:  "one name under two parents is two categories",
			sheet: "category,subcategory,url\nBPC-157,Liquid,/a/\nTB-500,Liquid,/b/\n",
			shelf: []string{"BPC-157", "BPC-157 › Liquid", "TB-500", "TB-500 › Liquid"},
			filed: map[string]string{"/a/": "BPC-157 › Liquid", "/b/": "TB-500 › Liquid"},
			preview: []string{
				"BPC-157 | create | 1", "BPC-157 › Liquid | create | 1", "TB-500 | create | 1", "TB-500 › Liquid | create | 1",
			},
		},
		{
			name:    "a level left out for its root name hangs the next on the level above",
			sheet:   "Root Entity,Category,Subcategory,Sub Subcategory,URL\nPeptides,,,,/peptides/\nShop,Blends,Peptides,Recovery,/x/\n",
			shelf:   []string{"Blends", "Blends › Recovery"},
			filed:   map[string]string{"/peptides/": "", "/x/": "Blends › Recovery"},
			preview: []string{"Blends | create | 1", "Blends › Recovery | create | 1"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			path := h.file(t, "sheet.csv", tc.sheet)
			report := h.preview(t, path, h.detected(t, path))
			if got := previewed(report); !slices.Equal(got, tc.preview) {
				t.Fatalf("the preview lists the categories\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(tc.preview, "\n"))
			}
			for at, want := range tc.filed {
				if found, _ := page(report, at); strings.Join(found.Categories, " › ") != want || found.Categories == nil {
					t.Errorf("the preview files %s under %v, want %q", at, found.Categories, want)
				}
			}

			applied := h.apply(t, path, h.detected(t, path))
			if applied.Counts.CategoriesCreated != len(tc.shelf) || applied.Counts.CategoriesDeleted != 0 {
				t.Fatalf("counts = %+v, want %d categories created", applied.Counts, len(tc.shelf))
			}
			if got := h.shelf(t); !slices.Equal(got, tc.shelf) {
				t.Fatalf("categories = %v, want %v", got, tc.shelf)
			}
			for at, want := range tc.filed {
				if got := h.filed(t)[at]; got != want {
					t.Errorf("%s is filed under %q, want %q", at, got, want)
				}
			}

			again := h.apply(t, path, h.detected(t, path))
			if again.Counts != (imports.Counts{Skipped: again.Counts.Skipped}) {
				t.Fatalf("the sheet imported again wrote %+v", again.Counts)
			}
			if got := previewed(again.Report); len(got) != len(tc.preview) || strings.Contains(strings.Join(got, ""), "create") {
				t.Fatalf("the second import lists %v, want every category matched", got)
			}
		})
	}
}

func TestTheSheetSaysWhereAPageIsFiledAndAnEmptyCellErasesNothing(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	steps := []struct {
		name    string
		sheet   string
		counts  imports.Counts
		filed   string
		shelf   []string
		preview []string
	}{
		{
			name: "the first import", sheet: "category,subcategory,url\nBPC-157,Liquid,/x/\n",
			counts: imports.Counts{EntitiesCreated: 1, PagesCreated: 1, CategoriesCreated: 2},
			filed:  "BPC-157 › Liquid", shelf: []string{"BPC-157", "BPC-157 › Liquid"},
			preview: []string{"BPC-157 | create | 1", "BPC-157 › Liquid | create | 1"},
		},
		{
			name: "an empty cell", sheet: "category,url,keywords\n,/x/,bpc liquid\n",
			counts: imports.Counts{EntitiesUpdated: 1, PagesUpdated: 1},
			filed:  "BPC-157 › Liquid", shelf: []string{"BPC-157", "BPC-157 › Liquid"},
			preview: []string{},
		},
		{
			name: "a renamed level", sheet: "category,subcategory,url\nBPC-157,Liquids,/x/\n",
			counts: imports.Counts{PagesUpdated: 1, CategoriesCreated: 1, CategoriesDeleted: 1},
			filed:  "BPC-157 › Liquids", shelf: []string{"BPC-157", "BPC-157 › Liquids"},
			preview: []string{"BPC-157 | match | 1", "BPC-157 › Liquids | create | 1", "BPC-157 › Liquid | delete | 0"},
		},
		{
			name: "another chain", sheet: "category,url\nTB-500,/x/\n",
			counts: imports.Counts{PagesUpdated: 1, CategoriesCreated: 1, CategoriesDeleted: 2},
			filed:  "TB-500", shelf: []string{"TB-500"},
			preview: []string{"TB-500 | create | 1", "BPC-157 › Liquids | delete | 0", "BPC-157 | delete | 0"},
		},
		{
			name: "the same chain again", sheet: "category,url\nTB-500,/x/\n",
			filed: "TB-500", shelf: []string{"TB-500"},
			preview: []string{"TB-500 | match | 1"},
		},
	}

	for _, step := range steps {
		path := h.file(t, "sheet.csv", step.sheet)
		applied := h.apply(t, path, h.detected(t, path))
		if step.counts.Skipped = applied.Counts.Skipped; applied.Counts != step.counts {
			t.Fatalf("%s wrote %+v, want %+v", step.name, applied.Counts, step.counts)
		}
		if got := previewed(applied.Report); !slices.Equal(got, step.preview) {
			t.Fatalf("%s lists the categories %v, want %v", step.name, got, step.preview)
		}
		if got := h.filed(t)["/x/"]; got != step.filed {
			t.Fatalf("after %s the page is filed under %q, want %q", step.name, got, step.filed)
		}
		if got := h.shelf(t); !slices.Equal(got, step.shelf) {
			t.Fatalf("after %s the categories are %v, want %v", step.name, got, step.shelf)
		}
	}
}

func TestACategoryTheSiteCarriesAsATermIsKept(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	first := h.file(t, "first.csv", "category,subcategory,url\nBPC-157,Liquid,/x/\nTB-500,,/y/\n")
	h.apply(t, first, h.detected(t, first))

	held := h.categories(t)
	liquid := slices.IndexFunc(held, func(c category.Category) bool { return c.Name == "Liquid" })
	term, err := category.NewTerm(category.Term{
		CategoryID: held[liquid].ID, SiteID: h.siteID, Taxonomy: category.TaxonomyCategory, TermID: 7, Name: "Liquid",
		SeenAt: sqlitetest.Stamp,
	})
	if err != nil {
		t.Fatalf("build the term: %v", err)
	}
	if err = sqlite.NewCategoryTermRepo(h.store).Upsert(t.Context(), term); err != nil {
		t.Fatalf("store the term: %v", err)
	}

	second := h.file(t, "second.csv", "category,url\nBlends,/x/\nBlends,/y/\n")
	applied := h.apply(t, second, h.detected(t, second))
	if applied.Counts.CategoriesDeleted != 1 {
		t.Fatalf("counts = %+v, want TB-500 alone deleted", applied.Counts)
	}
	if got := h.shelf(t); !slices.Equal(got, []string{"BPC-157", "BPC-157 › Liquid", "Blends"}) {
		t.Fatalf("categories = %v, want the term's category and its parent kept", got)
	}
}
