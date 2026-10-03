package imports

import (
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func (s *siteState) productDraft(draft *pageDraft) bool {
	current, held := s.held(draft.path)
	switch {
	case held && current.WPID != nil:
		return current.WPType == pagemap.WPProduct
	case draft.cellType() != "":
		return draft.cellType() == pagemap.WPProduct
	case held:
		return current.WPType == pagemap.WPProduct
	default:
		return draft.modeType == pagemap.WPProduct
	}
}

func typeRows(sheet *drafts, rows []rowDraft, rowType importmap.RowType, state *siteState) {
	if rowType == "" || rowType == importmap.RowPages {
		return
	}

	byMode := make(map[string]bool, len(sheet.pages))
	for _, path := range sheet.sortedPaths() {
		if draft := sheet.pages[path]; draft.cellType() == "" {
			byMode[path] = rowType == importmap.RowProducts || kindOfDraft(draft, rows) == graph.KindProduct
		}
	}
	product := func(path string) bool {
		if cell := sheet.pages[path].cellType(); cell != "" {
			return cell == pagemap.WPProduct
		}
		return byMode[path]
	}

	for _, path := range sheet.sortedPaths() {
		if byMode[path] && !productsUnder(path, sheet, product, state) {
			sheet.pages[path].modeType = pagemap.WPProduct
		}
	}
}

func kindOfDraft(draft *pageDraft, rows []rowDraft) graph.Kind {
	for _, at := range draft.rows {
		if kind, known := kindOf(rows[at].kind); known && kind != "" {
			return kind
		}
	}
	return ""
}

func productsUnder(path string, sheet *drafts, product func(string) bool, state *siteState) bool {
	for _, other := range sheet.sortedPaths() {
		if ancestorOf(path, other) && product(other) {
			return true
		}
	}
	for i := range state.pages {
		held := &state.pages[i]
		if held.WPType == pagemap.WPProduct && (ancestorOf(path, held.Path) || (held.PlannedPath != "" && ancestorOf(path, held.PlannedPath))) {
			return true
		}
	}
	return false
}

func matchProducts(sheet *drafts, state *siteState, p *plan) {
	claimed := make(map[string]bool)
	for i := range state.pages {
		if state.pages[i].PlannedPath != "" {
			claimed[state.pages[i].ID] = true
		}
	}
	for _, draft := range openProducts(sheet, state, claimed) {
		matchProduct(draft, state, claimed, p)
	}
}

func openProducts(sheet *drafts, state *siteState, claimed map[string]bool) []*pageDraft {
	open := make([]*pageDraft, 0)
	for _, path := range sheet.sortedPaths() {
		draft := sheet.pages[path]
		if draft.generated || !state.productDraft(draft) {
			continue
		}
		if current, held := state.held(path); held && current.WPID != nil {
			claimed[current.ID] = true
			draft.matchedBy = pagemap.MatchedByPath
			continue
		}
		open = append(open, draft)
	}
	return open
}

func matchProduct(draft *pageDraft, state *siteState, claimed map[string]bool, p *plan) {
	candidates := make([]pagemap.Page, 0, len(state.pages))
	for i := range state.pages {
		if !claimed[state.pages[i].ID] {
			candidates = append(candidates, state.pages[i])
		}
	}

	match := pagemap.MatchProduct(pagemap.ProductRow{Path: draft.path, H1: draft.h1, Title: draft.title}, candidates)
	switch {
	case match.Found:
		claimed[match.Page.ID] = true
		state.byPlanned[draft.path] = match.Page
		draft.matchedBy = match.By
		if waiting, left := state.byPath[draft.path]; left && waiting.WPID == nil {
			p.noteAt(draft.at, string(importmap.FieldPath), CodeProductRowLeft,
				"the row planned at "+draft.path+" waited for the product the store now holds at "+match.Page.Path+
					"; the import writes to the store's product, so delete the waiting row on the Pages screen")
		}
	case match.Ambiguous:
		p.noteAt(draft.at, string(importmap.FieldPath), CodeProductNotInStore,
			"more than one product in the store answers to "+draft.path+" by its "+string(match.By)+
				"; give the row the product's own address or slug, or rename one of the products in WooCommerce")
	default:
		p.noteAt(draft.at, string(importmap.FieldPath), CodeProductNotInStore,
			draft.path+" names a product the store does not hold yet; create it in WooCommerce and sync the site, "+
				"and the row finds it")
	}
}
