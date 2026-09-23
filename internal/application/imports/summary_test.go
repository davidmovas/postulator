package imports_test

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/application/imports"
)

func TestPreviewSummaryStaysSmallWhateverTheSheetHolds(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	rows := make([][]string, 0, 400)
	for i := range 400 {
		rows = append(rows, []string{"/shop/thing-" + strconv.Itoa(i) + "/", "Thing " + strconv.Itoa(i)})
	}
	rows = append(rows,
		[]string{"a b", "Broken one"},
		[]string{"a b", "Broken two"},
	)

	lines := make([]string, 0, len(rows)+1)
	lines = append(lines, "Page path,Page title")
	for _, row := range rows {
		lines = append(lines, strings.Join(row, ","))
	}
	path := h.file(t, "big.csv", strings.Join(lines, "\r\n")+"\r\n")

	summary, err := h.service.PreviewSummary(t.Context(), imports.PreviewRequest{
		SiteID: h.siteID, Path: path,
		Mapping: imports.Mapping{Columns: map[string]string{"path": "Page path", "title": "Page title"}},
	})
	if err != nil {
		t.Fatalf("PreviewSummary: %v", err)
	}

	if summary.Counts.PagesCreated == 0 || summary.Rows != len(rows) {
		t.Fatalf("summary = %+v, want every row counted", summary)
	}
	if len(summary.Pages) > imports.SummaryPages {
		t.Fatalf("the summary carries %d sample pages, want at most %d", len(summary.Pages), imports.SummaryPages)
	}
	if !summary.Blocking {
		t.Fatal("a summary over two unreadable paths must say the import is blocked")
	}

	var broken imports.FindingGroup
	for _, group := range summary.Findings {
		if group.Code == string(imports.CodeBadPath) {
			broken = group
		}
	}
	if broken.Count != 2 || !broken.Blocking {
		t.Fatalf("the bad path group = %+v, want both rows counted as blocking", broken)
	}
	if len(broken.Examples) > imports.SummaryExamples {
		t.Fatalf("the group carries %d examples, want at most %d", len(broken.Examples), imports.SummaryExamples)
	}

	encoded, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("encode the summary: %v", err)
	}
	if len(encoded) > 8192 {
		t.Fatalf("the summary encodes to %d bytes, which the agent result cap would cut", len(encoded))
	}
}
