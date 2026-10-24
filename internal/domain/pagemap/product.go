package pagemap

import (
	"github.com/davidmovas/postulator/internal/domain/field"
)

func (t WPType) StoreAddressed() bool {
	return t == WPProduct || t == WPProductCategory
}

func ClaimProduct(store Page, pages []Page, fields map[string][]field.Value) (Page, bool) {
	candidates := make([]Page, 0, len(pages)+1)
	for i := range pages {
		if pages[i].PlannedPath == "" && pages[i].WPID != nil {
			candidates = append(candidates, pages[i])
		}
	}
	site := NewRowSite(append(candidates, store), fields, nil)

	var (
		claimer Page
		count   int
	)
	for i := range pages {
		row := pages[i]
		if row.WPType != WPProduct || row.WPID != nil {
			continue
		}
		address := row.Path
		if row.PlannedPath != "" {
			address = row.PlannedPath
		}
		match := site.Match(Row{Path: address, H1: row.H1, Title: row.Title, Product: true})
		if match.Found() && match.Page.WPID != nil && store.WPID != nil && *match.Page.WPID == *store.WPID {
			claimer = row
			count++
		}
	}
	if count != 1 {
		return Page{}, false
	}
	return claimer, true
}
