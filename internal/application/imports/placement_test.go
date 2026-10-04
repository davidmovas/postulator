package imports_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/graph"
)

const twoLiquids = "url,entity,parent\n/bpc-157/,BPC-157,\n/bpc-157/liquid/,Liquid,BPC-157\n/tb-500/,TB-500,\n/tb-500/liquid/,Liquid,TB-500\n"

const twoLiquidsUnderARoot = "url,entity,parent\n/peptides/,Peptides,\n/peptides/bpc-157/,BPC-157,Peptides\n" +
	"/peptides/bpc-157/liquid/,Liquid,BPC-157\n/peptides/tb-500/,TB-500,Peptides\n/peptides/tb-500/liquid/,Liquid,TB-500\n"

func (h harness) scopeOf(t *testing.T, name string) []string {
	t.Helper()

	stored := h.entities(t)
	byID := make(map[string]graph.Entity, len(stored))
	for i := range stored {
		byID[stored[i].ID] = stored[i]
	}
	out := make([]string, 0, 1)
	for i := range stored {
		if stored[i].Name != name {
			continue
		}
		chain := ""
		for at, held := byID[deref(stored[i].ScopeID)]; held; at, held = byID[deref(at.ScopeID)] {
			chain = at.Name + " › " + chain
		}
		out = append(out, chain+name)
	}
	return out
}

func TestARowHangsUnderTheEntityItsCategoriesName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		before string
		sheet  string
		entity string
		parent string
		scope  string
	}{
		{
			name:   "the deepest category an entity carries",
			sheet:  "url,h1,category,subcategory\n/peptides/bpc-157/,BPC-157,,\n/shop/vial/,BPC-157 vial,BPC-157,Liquid\n",
			entity: "BPC-157 vial", parent: "BPC-157",
		},
		{
			name:   "an entity the site already holds",
			before: "url,h1\n/bpc-157/,BPC-157\n",
			sheet:  "url,h1,category\n/shop/vial/,BPC-157 vial,BPC-157\n",
			entity: "BPC-157 vial", parent: "BPC-157",
		},
		{
			name:   "the row's own name is passed over",
			sheet:  "url,h1,category,subcategory\n/bpc-157/,BPC-157,,\n/liquid/,Liquid,BPC-157,Liquid\n",
			entity: "Liquid", parent: "BPC-157",
		},
		{
			name:   "a parent cell wins over the categories",
			sheet:  "url,h1,category,parent\n/bpc-157/,BPC-157,,\n/tb-500/,TB-500,,\n/vial/,Vial,BPC-157,TB-500\n",
			entity: "Vial", parent: "TB-500",
		},
		{
			name:   "the category's entity wins over the root group",
			sheet:  "root,category,url,h1\nPeptides,,/peptides/,Peptides\nPeptides,,/peptides/bpc-157/,BPC-157\nPeptides,BPC-157,/shop/vial/,Vial\n",
			entity: "Vial", parent: "BPC-157",
		},
		{
			name:   "the root group when no category names an entity",
			sheet:  "root,category,url,h1\nPeptides,,/peptides/,Peptides\nPeptides,Blends,/shop/blend/,Blend\n",
			entity: "Blend", parent: "Peptides",
		},
		{
			name:   "a root named at a category level is the row's group",
			before: "root,url,h1\nPeptides,/peptides/,Peptides\n",
			sheet:  "category,subcategory,url,h1\nPeptides,Blends,/shop/blend/,Blend\n",
			entity: "Blend", parent: "Peptides",
		},
		{
			name:   "the url tree when nothing else places the row",
			sheet:  "category,url,h1\nBlends,/shop/,Shop\nBlends,/shop/blend/,Blend\n",
			entity: "Blend", parent: "Shop",
		},
		{
			name:   "the level above tells two entities of one name apart",
			before: twoLiquidsUnderARoot,
			sheet:  "category,subcategory,url,h1\nTB-500,Liquid,/drops/,Drops\n",
			entity: "Drops", scope: "Peptides › TB-500 › Liquid › Drops",
		},
		{
			name:   "a level above that names a root and is no category still tells two entities of one name apart",
			before: twoLiquids,
			sheet:  "category,subcategory,url,h1\nTB-500,Liquid,/drops/,Drops\n",
			entity: "Drops", scope: "TB-500 › Liquid › Drops",
		},
		{
			name:   "the level right above a category tells its namesakes apart, though it names a root",
			before: twoLiquids,
			sheet:  "category,subcategory,sub subcategory,url,h1\nBlends,TB-500,Liquid,/drops/,Drops\n",
			entity: "Drops", scope: "TB-500 › Liquid › Drops",
		},
		{
			name:   "a name two entities share places nothing and blocks nothing",
			before: twoLiquids,
			sheet:  "category,url,h1\nLiquid,/drops/,Drops\n",
			entity: "Drops", scope: "Drops",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			if tc.before != "" {
				before := h.file(t, "before.csv", tc.before)
				if applied := h.apply(t, before, h.detected(t, before)); len(applied.Report.Errors) != 0 {
					t.Fatalf("the site's own sheet reported %+v", applied.Report.Errors)
				}
			}
			path := h.file(t, "sheet.csv", tc.sheet)
			applied := h.apply(t, path, h.detected(t, path))
			if len(applied.Report.Errors) != 0 {
				t.Fatalf("errors = %+v", applied.Report.Errors)
			}
			if tc.scope != "" {
				if got := h.scopeOf(t, tc.entity); len(got) != 1 || got[0] != tc.scope {
					t.Fatalf("%s sits at %v, want %s", tc.entity, got, tc.scope)
				}
				return
			}
			placed, found := entity(applied.Report, tc.entity)
			if !found || placed.Parent != tc.parent {
				t.Fatalf("%s = %+v, want it under %s", tc.entity, placed, tc.parent)
			}
			if !hasEdge(applied.Report, tc.entity, tc.parent, string(graph.EdgeParent)) {
				t.Fatalf("edges = %+v, want %s under %s", applied.Report.Edges, tc.entity, tc.parent)
			}
		})
	}
}
