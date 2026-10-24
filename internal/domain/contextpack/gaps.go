package contextpack

import (
	"cmp"
	"slices"

	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

type Gap string

const (
	GapPlanned      Gap = "planned"
	GapEmpty        Gap = "empty"
	GapUnmapped     Gap = "unmapped"
	GapOrphan       Gap = "orphan"
	GapMissingLinks Gap = "missing_links"
	GapFindings     Gap = "findings"
)

func Gaps() []Gap {
	return []Gap{GapPlanned, GapEmpty, GapUnmapped, GapOrphan, GapMissingLinks, GapFindings}
}

type LinkFacts struct {
	Orphan          bool
	MissingRequired int
	Inbound         int
	Outbound        int
}

type GapRow struct {
	Page            Ref            `json:"page"`
	Gaps            []Gap          `json:"gaps"`
	MissingRequired int            `json:"missingRequired"`
	Findings        []finding.Code `json:"findings"`
}

type Node struct {
	Page        Ref `json:"page"`
	Children    int `json:"children"`
	Descendants int `json:"descendants"`
}

func (s Snapshot) Gaps(links map[string]LinkFacts) []GapRow {
	open := s.openCodes()
	out := make([]GapRow, 0)
	pages := s.livePages()
	for i := range pages {
		page := &pages[i]
		row := GapRow{Page: s.ref(*page, false), Gaps: []Gap{}, Findings: open[page.ID]}
		if row.Findings == nil {
			row.Findings = []finding.Code{}
		}
		linked := links[page.ID]
		for _, gap := range Gaps() {
			if s.has(*page, gap, linked, len(row.Findings) > 0) {
				row.Gaps = append(row.Gaps, gap)
			}
		}
		row.MissingRequired = linked.MissingRequired
		if len(row.Gaps) > 0 {
			out = append(out, row)
		}
	}
	return out
}

func (s Snapshot) has(page pagemap.Page, gap Gap, linked LinkFacts, findings bool) bool {
	switch gap {
	case GapPlanned:
		return !onSite(page)
	case GapEmpty:
		return s.Empty(page)
	case GapUnmapped:
		return s.Unmapped(page)
	case GapOrphan:
		return linked.Orphan
	case GapMissingLinks:
		return linked.MissingRequired > 0
	default:
		return findings
	}
}

func (s Snapshot) openCodes() map[string][]finding.Code {
	out := make(map[string][]finding.Code)
	for i := range s.Findings {
		held := &s.Findings[i]
		if held.Status != finding.Open || held.PageID == nil || slices.Contains(out[*held.PageID], held.Code) {
			continue
		}
		out[*held.PageID] = append(out[*held.PageID], held.Code)
	}
	for pageID := range out {
		slices.Sort(out[pageID])
	}
	return out
}

func (s Snapshot) Nodes(parentID string) []Node {
	tree := s.tree()
	out := make([]Node, 0)
	pages := s.livePages()
	for i := range pages {
		page := &pages[i]
		if s.parentOf(*page) != parentID {
			continue
		}
		out = append(out, Node{Page: s.ref(*page, false), Children: len(tree[page.ID]), Descendants: tree.descendants(page.ID)})
	}
	return out
}

func (s Snapshot) parentOf(page pagemap.Page) string {
	parent, found := s.parent(page)
	if !found || !live(parent) {
		return ""
	}
	return parent.ID
}

func (s Snapshot) livePages() []pagemap.Page {
	pages := slices.Clone(s.Index.Pages())
	pages = slices.DeleteFunc(pages, func(page pagemap.Page) bool { return !live(page) })
	slices.SortFunc(pages, func(a, b pagemap.Page) int { return cmp.Or(cmp.Compare(a.Path, b.Path), cmp.Compare(a.ID, b.ID)) })
	return pages
}
