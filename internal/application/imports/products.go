package imports

import (
	"strings"

	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
)

func (p *pageDraft) product() bool {
	current, held := p.page()
	switch {
	case held && current.WPID != nil:
		return current.WPType == pagemap.WPProduct
	case p.cellType() != "":
		return p.cellType() == pagemap.WPProduct
	case held:
		return current.WPType == pagemap.WPProduct
	default:
		return p.modeType == pagemap.WPProduct
	}
}

func (r *rowDraft) cellType(state *siteState) pagemap.WPType {
	wpType := pagemap.WPType(strings.ToLower(strings.TrimSpace(r.wpType)))
	if !wpType.Valid() || !state.knows(wpType) {
		return ""
	}
	return wpType
}

func modeProducts(rows []rowDraft, rowType importmap.RowType, state *siteState) {
	byMode := make([]bool, len(rows))
	if rowType != "" && rowType != importmap.RowPages {
		for i := range rows {
			if rows[i].cellType(state) == "" {
				byMode[i] = rowType == importmap.RowProducts || rowType == importmap.RowTyped && state.namesProduct(&rows[i])
			}
		}
	}
	product := func(at int) bool {
		if cell := rows[at].cellType(state); cell != "" {
			return cell == pagemap.WPProduct
		}
		return byMode[at]
	}
	for i := range rows {
		rows[i].modeProduct = byMode[i] && !productsUnder(i, rows, product, state)
		rows[i].wantsProduct = rows[i].cellType(state) == pagemap.WPProduct || rows[i].modeProduct
	}
}

func (s *siteState) namesProduct(row *rowDraft) bool {
	named, known := s.pageType(row.pageType, pagemap.WPPage)
	return known && named.Preset == pagetype.Product
}

func productsUnder(at int, rows []rowDraft, product func(int) bool, state *siteState) bool {
	path := rows[at].url
	if path == "" {
		return false
	}
	for i := range rows {
		if ancestorOf(path, rows[i].url) && product(i) {
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
