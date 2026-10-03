package imports_test

import (
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/application/imports"
)

func TestARootNameIsNeverACategoryLevel(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		before  string
		sheet   string
		dropped []int
	}{
		{
			name:    "a name the sheet's root column carries",
			sheet:   "Root Entity,Category,Subcategory,URL,H1\nPeptides,,,/peptides/,Peptides\nBlends,Peptides,Liquid,/blends/liquid/,Blend liquid\n",
			dropped: []int{3},
		},
		{
			name:    "a root hub the site holds",
			before:  "root,url,h1\nPeptides,/peptides/,Peptides\n",
			sheet:   "category,subcategory,url,h1\nPeptides,Blends,/shop/blend/,Blend\nTB-500,,/shop/tb/,TB\n",
			dropped: []int{2},
		},
		{
			name:    "a name written in another case and spacing",
			sheet:   "Root Entity,Category,URL,H1\nPeptides,,/peptides/,Peptides\nBlends,PEPTIDES,/blends/x/,X\n",
			dropped: []int{3},
		},
		{
			name:   "a hub that sits under another entity",
			before: "url,entity,entity kind,parent\n/a/,A,,\n/a/peptides/,Peptides,hub,A\n",
			sheet:  "category,url,h1\nPeptides,/shop/x/,X\n",
		},
		{
			name:   "an entity of another kind without a parent",
			before: "url,h1\n/peptides/,Peptides\n",
			sheet:  "category,url,h1\nPeptides,/shop/x/,X\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			if tc.before != "" {
				before := h.file(t, "before.csv", tc.before)
				h.apply(t, before, h.detected(t, before))
			}
			path := h.file(t, "sheet.csv", tc.sheet)
			report := h.preview(t, path, h.detected(t, path))
			if len(report.Errors) != 0 {
				t.Fatalf("errors = %+v", report.Errors)
			}

			dropped := findings(report.Warnings, imports.CodeCategoryLevelIsRoot)
			if len(dropped) != len(tc.dropped) {
				t.Fatalf("dropped levels = %+v, want rows %v", dropped, tc.dropped)
			}
			for i, finding := range dropped {
				if finding.Row != tc.dropped[i] || !strings.Contains(strings.ToLower(finding.Message), "peptides") {
					t.Errorf("finding %+v, want row %d naming Peptides", finding, tc.dropped[i])
				}
			}
		})
	}
}
