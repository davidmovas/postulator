package contextpack

import (
	"slices"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
)

func (m Map) Facts(rendered map[string]pagetype.Rendered, products map[string]int) []pagetype.Facts {
	return FactsOf(m.Rows(), m.snapshot.Index, rendered, products)
}

func FactsOf(rows []MapRow, index pagemap.Index, rendered map[string]pagetype.Rendered, products map[string]int) []pagetype.Facts {
	front := ""
	pages := index.Pages()
	for i := range pages {
		if pages[i].Path == pagemap.RootPath && live(pages[i]) && onSite(pages[i]) {
			front = pages[i].ID
		}
	}
	out := make([]pagetype.Facts, 0, len(rows))
	for i := range rows {
		row := &rows[i]
		page, known := index.ByID(row.PageID)
		if !known || !onSite(page) || !live(page) || !row.Row && row.HubID == "" {
			continue
		}
		var held *pagetype.Rendered
		if summary, found := rendered[page.ID]; found {
			held = &summary
		}
		facts := pagetype.FactsOf(page, held)
		facts.Products, facts.FrontPage = products[page.ID], page.ID == front
		above := row.Ancestors()
		facts.Ancestors = make([]string, 0, len(above))
		below := 0
		for _, ancestor := range slices.Backward(above) {
			facts.Ancestors = append(facts.Ancestors, ancestor)
			if ancestor != front {
				below++
			}
		}
		if !facts.FrontPage {
			facts.Level = 1 + below
		}
		out = append(out, facts)
	}
	return out
}
