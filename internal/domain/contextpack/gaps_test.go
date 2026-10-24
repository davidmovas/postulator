package contextpack_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/finding"
)

func TestEveryGapIsSaidOfItsPage(t *testing.T) {
	t.Parallel()

	s := withFindings(t)
	links := map[string]contextpack.LinkFacts{"guides": {Orphan: true}, "bpc": {MissingRequired: 2}}
	rows := s.Gaps(links)
	said := make(map[string]string, len(rows))
	for _, row := range rows {
		gaps := make([]string, 0, len(row.Gaps))
		for _, gap := range row.Gaps {
			gaps = append(gaps, string(gap))
		}
		said[row.Page.Path] = strings.Join(gaps, ",")
	}
	want := map[string]string{
		"/":                       "empty,unmapped",
		"/peptides/":              "empty",
		"/bpc-157/":               "missing_links,findings",
		"/healing-peptides/":      "empty,findings",
		"/lab-guides/":            "empty,unmapped,orphan,findings",
		"/peptides/tb-500-guide/": "planned",
		"/product/bpc-157-5mg/":   "empty,unmapped",
		"/product/bpc-157-10mg/":  "empty,unmapped",
		"/product/tb-500/":        "empty,unmapped",
		"/product/ghk-cu/":        "empty,unmapped",
	}
	for path, gaps := range want {
		if said[path] != gaps {
			t.Errorf("%s has gaps %q, want %q", path, said[path], gaps)
		}
	}
	if _, listed := said["/product-category/peptides/"]; listed {
		t.Error("the store category is read, never written, so it has no gap")
	}
	if _, listed := said["/old/"]; listed {
		t.Error("an archived page has no gap")
	}
	paths := make([]string, 0, len(rows))
	for _, row := range rows {
		paths = append(paths, row.Page.Path)
	}
	if !slices.IsSorted(paths) {
		t.Errorf("rows = %v, want them by path", paths)
	}
	at := slices.IndexFunc(rows, func(row contextpack.GapRow) bool { return row.Page.Path == "/bpc-157/" })
	if rows[at].MissingRequired != 2 || !slices.Equal(rows[at].Findings, []finding.Code{finding.ParentDiffers}) {
		t.Errorf("/bpc-157/ = %+v, want two required links missing and parent_differs", rows[at])
	}
}

func TestTheTreeListsAPagesChildren(t *testing.T) {
	t.Parallel()

	s := withFindings(t)
	roots := s.Nodes("")
	paths := make([]string, 0, len(roots))
	for _, node := range roots {
		paths = append(paths, node.Page.Path)
	}
	if want := []string{"/", "/product-category/peptides/", "/product/bpc-157-10mg/", "/product/bpc-157-5mg/", "/product/ghk-cu/", "/product/tb-500/"}; !slices.Equal(paths, want) {
		t.Errorf("roots = %v, want %v", paths, want)
	}
	if roots[0].Children != 2 || roots[0].Descendants != 5 {
		t.Errorf("the front page = %d children, %d below; want 2 and 5", roots[0].Children, roots[0].Descendants)
	}
	under := s.Nodes("hub")
	if len(under) != 3 || under[0].Page.Path != "/bpc-157/" || under[2].Page.Path != "/peptides/tb-500-guide/" || under[2].Page.OnSite {
		t.Errorf("under the hub = %+v, want BPC-157, Healing Peptides and the planned guide", under)
	}
}
