package pagemap

import (
	"strings"

	"github.com/davidmovas/postulator/internal/domain/field"
)

type MatchedBy string

const (
	MatchedByPath MatchedBy = "path"
	MatchedBySlug MatchedBy = "slug"
	MatchedByName MatchedBy = "name"
)

type ProductRow struct {
	Path  string
	H1    string
	Title string
}

type ProductMatch struct {
	Page      Page
	By        MatchedBy
	Found     bool
	Ambiguous bool
}

func (t WPType) StoreAddressed() bool {
	return t == WPProduct || t == WPProductCategory
}

func MatchProduct(row ProductRow, pages []Page, names map[string][]string) ProductMatch {
	held := inTheStore(pages)

	if path, err := NormalizePath(row.Path); err == nil {
		for i := range held {
			if held[i].Path == path {
				return ProductMatch{Page: held[i], By: MatchedByPath, Found: true}
			}
		}
		if match, decided := only(held, MatchedBySlug, func(p Page) bool { return p.Slug == Slug(path) }); decided {
			return match
		}
	}

	for _, name := range []string{row.H1, row.Title} {
		wanted := strings.TrimSpace(name)
		if wanted == "" {
			continue
		}
		if match, decided := only(held, MatchedByName, func(p Page) bool { return answersTo(p, names[p.ID], wanted) }); decided {
			return match
		}
	}
	return ProductMatch{}
}

func answersTo(page Page, known []string, wanted string) bool {
	if len(known) == 0 {
		known = []string{page.Title}
	}
	for _, name := range known {
		if strings.EqualFold(strings.TrimSpace(name), wanted) {
			return true
		}
	}
	return false
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

func inTheStore(pages []Page) []Page {
	held := make([]Page, 0, len(pages))
	for i := range pages {
		if pages[i].WPType == WPProduct && pages[i].WPID != nil {
			held = append(held, pages[i])
		}
	}
	return held
}

func only(pages []Page, by MatchedBy, keep func(Page) bool) (ProductMatch, bool) {
	var (
		found Page
		count int
	)
	for i := range pages {
		if keep(pages[i]) {
			found = pages[i]
			count++
		}
	}
	switch count {
	case 0:
		return ProductMatch{}, false
	case 1:
		return ProductMatch{Page: found, By: by, Found: true}, true
	default:
		return ProductMatch{By: by, Ambiguous: true}, true
	}
}
