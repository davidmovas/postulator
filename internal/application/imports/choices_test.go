package imports_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func ambiguousRows(report imports.PreviewReport) []imports.PreviewPage {
	out := make([]imports.PreviewPage, 0)
	for i := range report.Pages {
		if report.Pages[i].Action == string(imports.ActionAmbiguous) {
			out = append(out, report.Pages[i])
		}
	}
	return out
}

func candidatePaths(view imports.PreviewPage) []string {
	out := make([]string, 0, len(view.Candidates))
	for i := range view.Candidates {
		out = append(out, view.Candidates[i].Path)
	}
	return out
}

func TestAPersonPicksAnAmbiguousRowsPageInThePreview(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	first := h.sitePage(t, "/a/guide/", "Guide", 31, nil)
	second := h.sitePage(t, "/b/guide/", "Guide", 32, nil)
	mapping := h.mapping(map[string]string{string(importmap.FieldTitle): "title", string(importmap.FieldKeywords): "keywords"})
	sheet := "title,keywords\nGuide,guide\n"
	file := h.file(t, "rows.csv", sheet)
	h.applyCSV(t, sheet, mapping)
	earlier := h.one(t, finding.RowAmbiguous)

	shown := ambiguousRows(h.preview(t, file, mapping))
	if len(shown) != 1 {
		t.Fatalf("the preview lists %d ambiguous rows, want the one row with its candidates", len(shown))
	}
	row := shown[0]
	if row.RowKey != earlier.SubjectKey || row.MatchedBy != string(pagemap.RuleTitle) || row.Title != "Guide" {
		t.Errorf("the ambiguous row = %+v, want its key, the rule that tied and its title", row)
	}
	if got := candidatePaths(row); !slices.Equal(got, []string{"/a/guide/", "/b/guide/"}) || row.Candidates[1].PageID != second.ID {
		t.Errorf("candidates = %+v, want both guides with their ids", row.Candidates)
	}

	chosen := []imports.Choice{{RowKey: row.RowKey, PageID: second.ID}}
	picked, err := h.service.Preview(t.Context(), imports.PreviewRequest{SiteID: h.siteID, Path: file, Mapping: mapping, Choices: chosen})
	if err != nil {
		t.Fatalf("Preview with the choice: %v", err)
	}
	view, found := page(picked.Report, "/b/guide/")
	switch {
	case !found || view.MatchedBy != string(pagemap.RuleMapped) || view.Action != string(imports.ActionUpdate):
		t.Fatalf("pages = %+v, want the chosen page updated by the mapping", picked.Report.Pages)
	case view.RowKey != row.RowKey || len(view.Candidates) != 2:
		t.Errorf("the chosen row = %+v, want it to keep its key and candidates so the pick can change", view)
	case len(ambiguousRows(picked.Report)) != 0 || len(findings(picked.Report.Warnings, imports.CodeRowAmbiguous)) != 0:
		t.Errorf("the picked preview still warns of an ambiguous row: %+v", picked.Report.Warnings)
	}

	if _, err := h.service.Apply(t.Context(), imports.ApplyRequest{SiteID: h.siteID, Path: file, Mapping: mapping, Choices: chosen}); err != nil {
		t.Fatalf("Apply with the choice: %v", err)
	}
	mapped, _ := h.pageByID(t, second.ID)
	if mapped.RowKey != row.RowKey || mapped.Keywords.Main() != "guide" {
		t.Fatalf("the chosen page = %+v, want the row's key and its keyword", mapped)
	}
	if untouched, _ := h.pageByID(t, first.ID); untouched.RowKey != "" || len(untouched.Keywords) != 0 {
		t.Errorf("the other candidate = %+v, want it untouched", untouched)
	}
	if closed := h.one(t, finding.RowAmbiguous); closed.Status == finding.Open || closed.Resolution != finding.ResolutionRowMatched {
		t.Errorf("the earlier apply's finding = %+v, want it closed as row_matched", closed)
	}

	again := h.preview(t, file, mapping)
	if view, found := page(again, "/b/guide/"); !found || view.MatchedBy != string(pagemap.RuleMapped) || len(ambiguousRows(again)) != 0 {
		t.Errorf("the next import = %+v, want the row found by its mapping with no choice given", again.Pages)
	}
}

func TestAChoiceTheRowsDoNotAnswerToIsRefused(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.sitePage(t, "/a/guide/", "Guide", 31, nil)
	h.sitePage(t, "/b/guide/", "Guide", 32, nil)
	elsewhere := h.sitePage(t, "/kettles/", "Kettles", 33, nil)
	mapping := h.mapping(map[string]string{string(importmap.FieldTitle): "title"})
	file := h.file(t, "rows.csv", "title\nGuide\n")
	key := ambiguousRows(h.preview(t, file, mapping))[0].RowKey

	cases := []struct {
		name   string
		choice imports.Choice
	}{
		{name: "a page that is not a candidate", choice: imports.Choice{RowKey: key, PageID: elsewhere.ID}},
		{name: "a row the workbook does not hold", choice: imports.Choice{RowKey: pagemap.RowKeyPrefix("rows.csv") + "teapots", PageID: elsewhere.ID}},
		{name: "a choice that names no page", choice: imports.Choice{RowKey: key}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := imports.PreviewRequest{SiteID: h.siteID, Path: file, Mapping: mapping, Choices: []imports.Choice{tc.choice}}
			if _, err := h.service.Preview(t.Context(), req); !errors.IsCode(err, errors.Invalid) {
				t.Errorf("Preview(%+v) = %v, want Invalid", tc.choice, err)
			}
			apply := imports.ApplyRequest{SiteID: h.siteID, Path: file, Mapping: mapping, Choices: []imports.Choice{tc.choice}}
			if _, err := h.service.Apply(t.Context(), apply); !errors.IsCode(err, errors.Invalid) {
				t.Errorf("Apply(%+v) = %v, want Invalid", tc.choice, err)
			}
		})
	}
	if held := h.pages(t); slices.ContainsFunc(held, func(p pagemap.Page) bool { return p.RowKey != "" }) {
		t.Errorf("a refused choice wrote a mapping: %+v", held)
	}
}
