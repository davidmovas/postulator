package imports_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/domain/graph"
)

func droppedRows(t *testing.T, report imports.PreviewReport, name string) []int {
	t.Helper()

	dropped := findings(report.Warnings, imports.CodeCategoryLevelIsRoot)
	out := make([]int, 0, len(dropped))
	for _, finding := range dropped {
		if !strings.Contains(strings.ToLower(finding.Message), strings.ToLower(name)) {
			t.Errorf("the dropped level %+v, want it to name %s", finding, name)
		}
		out = append(out, finding.Row)
	}
	return out
}

func TestARootNameIsNeverACategoryLevel(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		before  string
		kind    graph.Kind
		sheet   string
		dropped []int
	}{
		{
			name:    "a name the sheet's root column carries",
			sheet:   "Root Entity,Category,Subcategory,URL,H1\nPeptides,,,/peptides/,Peptides\nBlends,Peptides,Liquid,/blends/liquid/,Blend liquid\n",
			dropped: []int{3},
		},
		{
			name:    "a name written in another case and spacing",
			sheet:   "Root Entity,Category,URL,H1\nPeptides,,/peptides/,Peptides\nBlends,PEPTIDES,/blends/x/,X\n",
			dropped: []int{3},
		},
		{
			name:    "a hub the site holds at the top of its graph",
			before:  "root,url,h1\nPeptides,/peptides/,Peptides\n",
			kind:    graph.KindHub,
			sheet:   "category,subcategory,url,h1\nPeptides,Blends,/shop/blend/,Blend\nTB-500,,/shop/tb/,TB\n",
			dropped: []int{2},
		},
		{
			name:    "an entity of kind category the site holds at the top of its graph",
			before:  "url,entity,entity kind\n/peptides/,Peptides,category\n",
			kind:    graph.KindCategory,
			sheet:   "category,subcategory,url,h1\nPeptides,Blends,/shop/blend/,Blend\n",
			dropped: []int{2},
		},
		{
			name:   "a topic the site holds at the top of its graph",
			before: "url,h1\n/peptides/,Peptides\n",
			kind:   graph.KindTopic,
			sheet:  "category,url,h1\nPeptides,/shop/x/,X\n",
		},
		{
			name:   "a product the site holds at the top of its graph",
			before: "url,entity,entity kind\n/peptides/,Peptides,product\n",
			kind:   graph.KindProduct,
			sheet:  "category,url,h1\nPeptides,/shop/x/,X\n",
		},
		{
			name:   "a hub that sits under another entity",
			before: "url,entity,entity kind,parent\n/a/,A,,\n/a/peptides/,Peptides,hub,A\n",
			kind:   graph.KindHub,
			sheet:  "category,url,h1\nPeptides,/shop/x/,X\n",
		},
		{
			name:   "an entity of kind category that sits under another entity",
			before: "url,entity,entity kind,parent\n/a/,A,,\n/a/peptides/,Peptides,category,A\n",
			kind:   graph.KindCategory,
			sheet:  "category,url,h1\nPeptides,/shop/x/,X\n",
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
			if got := h.kinds(t)["Peptides"]; got != tc.kind {
				t.Fatalf("the site holds Peptides as %q, want %q", got, tc.kind)
			}
			path := h.file(t, "sheet.csv", tc.sheet)
			report := h.preview(t, path, h.detected(t, path))
			if len(report.Errors) != 0 {
				t.Fatalf("errors = %+v", report.Errors)
			}
			if got := droppedRows(t, report, "Peptides"); !slices.Equal(got, tc.dropped) {
				t.Fatalf("the level Peptides was dropped on the rows %v, want %v", got, tc.dropped)
			}
		})
	}
}

func TestAGroupingEntityTheSheetPlansAtTheTopIsNoCategory(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		kind    string
		shelf   []string
		filed   string
		dropped []int
	}{
		{
			name:  "a topic",
			shelf: []string{"Blends", "Blends › Recovery"},
			filed: "Blends › Recovery",
		},
		{
			name:    "a hub",
			kind:    "hub",
			shelf:   []string{"Recovery"},
			filed:   "Recovery",
			dropped: []int{3},
		},
		{
			name:    "an entity of kind category",
			kind:    "category",
			shelf:   []string{"Recovery"},
			filed:   "Recovery",
			dropped: []int{3},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			path := h.file(t, "sheet.csv", "url,h1,entity kind,category,subcategory\n/blends/,Blends,"+tc.kind+",,\n/shop/x/,X,,Blends,Recovery\n")
			for _, pass := range []string{"the first import", "the second import"} {
				applied := h.apply(t, path, h.detected(t, path))
				if len(applied.Report.Errors) != 0 {
					t.Fatalf("%s reported %+v", pass, applied.Report.Errors)
				}
				if got := droppedRows(t, applied.Report, "Blends"); !slices.Equal(got, tc.dropped) {
					t.Fatalf("%s dropped the level Blends on the rows %v, want %v", pass, got, tc.dropped)
				}
				if got := h.shelf(t); !slices.Equal(got, tc.shelf) {
					t.Fatalf("after %s the categories are %v, want %v", pass, got, tc.shelf)
				}
				if got := h.filed(t)["/shop/x/"]; got != tc.filed {
					t.Fatalf("after %s /shop/x/ is filed under %q, want %q", pass, got, tc.filed)
				}
				if pass == "the second import" && applied.Counts != (imports.Counts{Skipped: applied.Counts.Skipped}) {
					t.Fatalf("the second import wrote %+v", applied.Counts)
				}
			}
		})
	}
}
