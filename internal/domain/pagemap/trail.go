package pagemap

import (
	"cmp"
	"html"
	"maps"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/finding"
)

func TrailOptions(wpType WPType) []finding.Option {
	if wpType == WPProduct {
		return []finding.Option{finding.AcceptSite, finding.Ignore}
	}
	return []finding.Option{finding.AcceptSite, finding.AcceptWorkbook, finding.Ignore}
}

type TrailDiff struct {
	Sheet []string `json:"sheet"`
	Site  []string `json:"site"`
}

type TrailSite struct {
	index   Index
	terms   map[int64]Term
	carried map[string][]Term
}

type trailNode struct {
	page  Page
	names []string
	shown string
}

func TermParents(pages []Page, identities map[string]Identity) map[int64]int64 {
	out := make(map[int64]int64)
	for i := range pages {
		identity, observed := identities[pages[i].ID]
		if !pages[i].WPType.Term() || pages[i].WPID == nil || !observed {
			continue
		}
		out[*pages[i].WPID] = 0
		if identity.WPParentID != nil {
			out[*pages[i].WPID] = *identity.WPParentID
		}
	}
	return out
}

func NewTrailSite(index Index, carried map[string][]Term, parents map[int64]int64) TrailSite {
	site := TrailSite{index: index, terms: make(map[int64]Term), carried: make(map[string][]Term, len(carried))}
	for _, pageID := range slices.Sorted(maps.Keys(carried)) {
		for _, term := range carried[pageID] {
			if term.Taxonomy != string(WPProductCategory) {
				continue
			}
			site.carried[pageID] = append(site.carried[pageID], term)
			if known, seen := site.terms[term.TermID]; !seen || known.ParentTermID == 0 {
				site.terms[term.TermID] = term
			}
		}
	}
	for at := range index.pages {
		page := index.pages[at]
		if !page.WPType.Term() || page.WPID == nil {
			continue
		}
		if _, seen := site.terms[*page.WPID]; !seen {
			site.terms[*page.WPID] = Term{Taxonomy: string(WPProductCategory), TermID: *page.WPID, Name: page.Title}
		}
	}
	for termID, parent := range parents {
		if term, seen := site.terms[termID]; seen {
			term.ParentTermID = parent
			site.terms[termID] = term
		}
	}
	return site
}

func TrailDiffers(page Page, site TrailSite) (TrailDiff, bool) {
	if len(page.PlannedTrail) == 0 {
		return TrailDiff{Sheet: []string{}, Site: []string{}}, false
	}
	diff := TrailDiff{Sheet: slices.Clone(page.PlannedTrail), Site: []string{}}
	if page.WPType != WPProduct {
		chain := site.pageChain(page)
		diff.Site = shownOf(chain)
		return diff, !agrees(page.PlannedTrail, chain)
	}
	chains := site.productChains(page)
	if len(chains) == 0 {
		return diff, true
	}
	nearest := 0
	for i := range chains {
		if agrees(page.PlannedTrail, chains[i]) {
			diff.Site = shownOf(chains[i])
			return diff, false
		}
		if shared(page.PlannedTrail, chains[i]) > shared(page.PlannedTrail, chains[nearest]) {
			nearest = i
		}
	}
	diff.Site = shownOf(chains[nearest])
	return diff, true
}

func (s TrailSite) pageChain(page Page) []trailNode {
	chain := make([]trailNode, 0)
	seen := map[string]bool{page.ID: true}
	for at := page.ParentPageID; at != nil && !seen[*at]; {
		seen[*at] = true
		parent, found := s.index.ByID(*at)
		if !found {
			break
		}
		if parent.Path != RootPath && parent.TechnicalReason == TechnicalNone {
			chain = append(chain, trailNode{page: parent, names: s.index.namesOf(parent), shown: s.shownName(parent)})
		}
		at = parent.ParentPageID
	}
	slices.Reverse(chain)
	return s.joinTwins(chain)
}

func (s TrailSite) shownName(page Page) string {
	for _, name := range append(slices.Clone(s.index.names[page.ID]), page.Title, page.H1) {
		if trimmed := strings.TrimSpace(name); trimmed != "" {
			return html.UnescapeString(trimmed)
		}
	}
	return page.Path
}

func (s TrailSite) joinTwins(chain []trailNode) []trailNode {
	out := make([]trailNode, 0, len(chain))
	for at := range chain {
		node := &chain[at]
		last := len(out) - 1
		switch {
		case last < 0 || !s.twins(out[last].page, node.page):
			out = append(out, *node)
		case node.page.WPType.Term():
			out[last].names = append(out[last].names, node.names...)
		default:
			node.names = append(node.names, out[last].names...)
			out[last] = *node
		}
	}
	return out
}

func (s TrailSite) twins(a, b Page) bool {
	if a.WPType.Term() == b.WPType.Term() {
		return false
	}
	if a.WPType.Term() {
		a, b = b, a
	}
	return twinnable(a) && s.index.sharesName(a, b)
}

func (s TrailSite) productChains(page Page) [][]trailNode {
	held := slices.Clone(s.carried[page.ID])
	slices.SortFunc(held, func(a, b Term) int { return cmp.Compare(a.TermID, b.TermID) })
	chains := make([][]trailNode, 0, len(held))
	for i := range held {
		chains = append(chains, s.termChain(held[i].TermID))
	}
	return chains
}

func (s TrailSite) termChain(termID int64) []trailNode {
	chain := make([]trailNode, 0)
	seen := make(map[int64]bool)
	for at := termID; at != 0 && !seen[at]; {
		seen[at] = true
		term, known := s.terms[at]
		if !known {
			break
		}
		chain = append(chain, s.termNode(term))
		at = term.ParentTermID
	}
	slices.Reverse(chain)
	return chain
}

func (s TrailSite) termNode(term Term) trailNode {
	node := trailNode{names: []string{term.Name}, shown: html.UnescapeString(strings.TrimSpace(term.Name))}
	if page, found := s.index.ByWordPressID(WPProductCategory, term.TermID); found {
		node.names = append(node.names, s.index.namesOf(page)...)
		node.shown = s.shownName(page)
	}
	return node
}

func (n trailNode) carries(name string) bool {
	key := nameKey(name)
	if key == "" {
		return false
	}
	for _, held := range n.names {
		if held != "" && nameKey(held) == key {
			return true
		}
	}
	return false
}

func agrees(sheet []string, chain []trailNode) bool {
	return len(sheet) == len(chain) && shared(sheet, chain) == len(sheet)
}

func shared(sheet []string, chain []trailNode) int {
	count := 0
	for count < len(sheet) && count < len(chain) && chain[count].carries(sheet[count]) {
		count++
	}
	return count
}

func shownOf(chain []trailNode) []string {
	out := make([]string, 0, len(chain))
	for i := range chain {
		out = append(out, chain[i].shown)
	}
	return out
}
