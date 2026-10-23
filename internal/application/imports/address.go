package imports

import (
	"cmp"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

var ruleWords = map[pagemap.MatchRule]string{
	pagemap.RuleMapped: "mapping", pagemap.RuleURL: "address", pagemap.RuleSlug: "slug", pagemap.RuleEntity: "entity",
	pagemap.RuleTitle: "title", pagemap.RuleH1: "heading", pagemap.RuleSKU: "SKU",
}

var ruleFields = map[pagemap.MatchRule]importmap.Field{
	pagemap.RuleURL: importmap.FieldPath, pagemap.RuleSlug: importmap.FieldPath, pagemap.RuleEntity: importmap.FieldEntity,
	pagemap.RuleTitle: importmap.FieldTitle, pagemap.RuleH1: importmap.FieldH1, pagemap.RuleSKU: importmap.FieldSKU,
}

type sheetRows struct {
	rows []rowDraft
	urls map[string]int
}

type branch struct {
	id    string
	known bool
}

func matchRows(rows []rowDraft, rowType importmap.RowType, state *siteState, p *plan) []rowDraft {
	modeProducts(rows, rowType, state)
	sheet := sheetRows{rows: rows, urls: make(map[string]int, len(rows))}
	order := make([]int, 0, len(rows))
	for i := range rows {
		if _, seen := sheet.urls[rows[i].url]; rows[i].url != "" && !seen {
			sheet.urls[rows[i].url] = i
		}
		order = append(order, i)
	}
	slices.SortStableFunc(order, func(a, b int) int {
		return cmp.Compare(strings.Count(rows[a].url, "/"), strings.Count(rows[b].url, "/"))
	})
	for _, at := range order {
		if !rows[at].unreadable {
			state.address(&rows[at], &sheet)
		}
	}

	kept := make([]rowDraft, 0, len(rows))
	for i := range rows {
		row := &rows[i]
		switch {
		case row.match.Ambiguous():
			p.noteAt(row.at, string(ruleFields[row.match.Rule]), CodeRowAmbiguous, ambiguity(row))
			p.report.Skipped++
		case row.path == "" && row.name == "" && len(row.levels) == 0 && !row.unreadable:
			noTarget(row.at, p)
			p.report.Skipped++
		default:
			state.noteLeftRow(row, p)
			kept = append(kept, *row)
		}
	}
	return kept
}

func (s *siteState) address(row *rowDraft, sheet *sheetRows) {
	row.match = s.matchRow(row, sheet)
	switch {
	case row.match.Ambiguous():
	case row.match.Found():
		row.path = row.match.Page.Path
	default:
		row.path = s.plannedAt(row)
		if held, taken := s.byPath[row.path]; taken {
			row.match = pagemap.Match{Page: held, Rule: pagemap.RuleURL, Candidates: []pagemap.Page{held}}
		}
	}
}

func (s *siteState) matchRow(row *rowDraft, sheet *sheetRows) pagemap.Match {
	found := s.matchAs(row, row.wantsProduct, sheet)
	if found.Found() || found.Ambiguous() || !row.modeProduct {
		return found
	}
	if page := s.matchAs(row, false, sheet); page.Rule == pagemap.RuleURL || page.Rule == pagemap.RuleMapped {
		return page
	}
	return found
}

func (s *siteState) matchAs(row *rowDraft, product bool, sheet *sheetRows) pagemap.Match {
	asked := row.asked(product)
	under := s.branchOf(row.url, sheet)
	for {
		found := s.site.Match(asked)
		if !under.known || !textual(found.Rule) {
			return found
		}
		kept := slices.DeleteFunc(slices.Clone(found.Candidates), func(page pagemap.Page) bool {
			beside := s.branchOf(page.Path, nil)
			return beside.known && beside.id != under.id
		})
		switch len(kept) {
		case 0:
			forget(&asked, found.Rule)
		case 1:
			return pagemap.Match{Page: kept[0], Rule: found.Rule, Candidates: kept}
		default:
			return pagemap.Match{Rule: found.Rule, Candidates: kept}
		}
	}
}

func textual(rule pagemap.MatchRule) bool {
	return rule == pagemap.RuleSlug || rule == pagemap.RuleTitle || rule == pagemap.RuleH1
}

func forget(row *pagemap.Row, rule pagemap.MatchRule) {
	switch rule {
	case pagemap.RuleSlug:
		row.Path = ""
	case pagemap.RuleTitle:
		row.Title = ""
	case pagemap.RuleH1:
		row.H1 = ""
	case pagemap.RuleMapped, pagemap.RuleURL, pagemap.RuleEntity, pagemap.RuleSKU:
	}
}

func (s *siteState) branchOf(path string, sheet *sheetRows) branch {
	parent := pagemap.ParentPath(path)
	if parent == "" || parent == pagemap.RootPath {
		return branch{}
	}
	if sheet != nil {
		if at, planned := sheet.urls[parent]; planned {
			if above := sheet.rows[at].match; above.Found() {
				return branch{id: above.Page.ID, known: true}
			}
			return branch{id: parent, known: true}
		}
	}
	if above, held := s.pageAt(parent); held {
		return branch{id: above.ID, known: true}
	}
	return branch{}
}

func (s *siteState) plannedAt(row *rowDraft) string {
	if row.url != "" {
		return row.url
	}
	for _, name := range []string{row.title, row.name, row.h1} {
		slug := slugOf(name)
		if slug == "" {
			continue
		}
		path, err := pagemap.NormalizePath(s.levelPath(row) + slug + "/")
		if err != nil {
			return ""
		}
		return path
	}
	return ""
}

func (s *siteState) levelPath(row *rowDraft) string {
	for i := len(row.levels) - 1; i >= 0; i-- {
		level := row.levels[i]
		if graph.Key(level) == graph.Key(row.named()) {
			continue
		}
		found := s.site.Match(pagemap.Row{Entity: level, Title: level, H1: level})
		if found.Found() && holdsPages(found.Page) {
			return found.Page.Path
		}
		return pagemap.RootPath
	}
	return pagemap.RootPath
}

func holdsPages(page pagemap.Page) bool {
	return !page.WPType.Term() && page.WPType != pagemap.WPProduct
}

func (s *siteState) pageAt(path string) (pagemap.Page, bool) {
	if found := s.site.Match(pagemap.Row{Path: path}); found.Found() && found.Rule == pagemap.RuleURL {
		return found.Page, true
	}
	page, held := s.byPath[path]
	return page, held
}

func (s *siteState) noteLeftRow(row *rowDraft, p *plan) {
	found := row.match.Page
	waiting, left := s.byPath[row.url]
	if !left || !inTheStore(found) || waiting.WPID != nil || waiting.ID == found.ID {
		return
	}
	p.noteAt(row.at, string(importmap.FieldPath), CodeProductRowLeft,
		"the row planned at "+row.url+" waited for the product the store now holds at "+found.Path+
			"; the import writes to the store's product, so delete the waiting row on the Pages screen")
}

func ambiguity(row *rowDraft) string {
	paths := make([]string, 0, len(row.match.Candidates))
	for i := range row.match.Candidates {
		paths = append(paths, row.match.Candidates[i].Path)
	}
	what := "page of the site"
	if row.wantsProduct {
		what = "product in the store"
	}
	return "more than one " + what + " answers to the row by its " + ruleWords[row.match.Rule] + ": " +
		strings.Join(paths, ", ") + "; the row writes nothing until it names one of them by its URL"
}

func withoutPage(draft *pageDraft) string {
	if draft.createdType() == pagemap.WPProduct {
		return "no product in the store answers to the row, so it waits at " + draft.path +
			"; create the product in WooCommerce and sync the site, and the row finds it"
	}
	return "no page of the site answers to the row, so it is planned at " + draft.path
}
