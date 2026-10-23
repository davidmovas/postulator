package pagemap

import (
	"strings"

	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/graph"
)

type MatchRule string

const (
	RuleMapped MatchRule = "mapped"
	RuleURL    MatchRule = "url"
	RuleSlug   MatchRule = "slug"
	RuleEntity MatchRule = "entity"
	RuleTitle  MatchRule = "title"
	RuleH1     MatchRule = "h1"
	RuleSKU    MatchRule = "sku"
)

type Row struct {
	Key     string
	Path    string
	Entity  string
	Parent  string
	Title   string
	H1      string
	SKU     string
	Product bool
}

func RowKeyPrefix(sheet string) string {
	return "row:" + sheet + ":"
}

func RowKey(sheet string, row Row) string {
	identity := strings.TrimSpace(row.Path)
	for _, name := range []string{row.Title, row.Entity, row.H1} {
		if identity == "" {
			identity = nameKey(name)
		}
	}
	if identity == "" {
		return ""
	}
	return RowKeyPrefix(sheet) + identity
}

type Match struct {
	Page       Page
	Rule       MatchRule
	Candidates []Page
}

func (m Match) Found() bool {
	return len(m.Candidates) == 1
}

func (m Match) Ambiguous() bool {
	return len(m.Candidates) > 1
}

type rowEntity struct {
	name   string
	parent string
}

type RowSite struct {
	onSite   []Page
	planned  []Page
	titles   map[string][]string
	headings map[string][]string
	skus     map[string]string
	entities map[string]rowEntity
}

func NewRowSite(pages []Page, fields map[string][]field.Value, entities []graph.Entity) RowSite {
	site := RowSite{
		titles: make(map[string][]string), headings: make(map[string][]string),
		skus: make(map[string]string), entities: make(map[string]rowEntity, len(entities)),
	}
	for i := range pages {
		switch {
		case pages[i].Status == StatusArchived:
		case pages[i].WPID != nil:
			site.onSite = append(site.onSite, pages[i])
		default:
			site.planned = append(site.planned, pages[i])
		}
	}
	for pageID, values := range fields {
		for i := range values {
			if values[i].Current == "" {
				continue
			}
			switch values[i].Key {
			case field.Title:
				site.titles[pageID] = append(site.titles[pageID], values[i].Current)
			case field.H1:
				site.headings[pageID] = append(site.headings[pageID], values[i].Current)
			case field.SKU:
				site.skus[pageID] = values[i].Current
			}
		}
	}
	names := make(map[string]string, len(entities))
	for i := range entities {
		names[entities[i].ID] = entities[i].Name
	}
	for i := range entities {
		held := rowEntity{name: entities[i].Name}
		if entities[i].ScopeID != nil {
			held.parent = names[*entities[i].ScopeID]
		}
		site.entities[entities[i].ID] = held
	}
	return site
}

type matchTest struct {
	rule    MatchRule
	asked   bool
	address bool
	test    func(Page) bool
}

func (s RowSite) Match(row Row) Match {
	path, err := NormalizePath(row.Path)
	if err != nil {
		path = ""
	}
	slug := ""
	if path != "" {
		slug = Slug(path)
	}
	tests := []matchTest{
		{rule: RuleMapped, asked: row.Key != "", address: true, test: func(p Page) bool { return p.RowKey == row.Key }},
		{rule: RuleURL, asked: path != "", address: true, test: func(p Page) bool { return p.Path == path || p.PlannedPath == path }},
		{rule: RuleSlug, asked: slug != "", test: func(p Page) bool { return p.Slug == slug }},
		{rule: RuleEntity, asked: nameKey(row.Entity) != "", test: func(p Page) bool { return s.carriesEntity(p, row.Entity, row.Parent) }},
		{rule: RuleTitle, asked: nameKey(row.Title) != "", test: func(p Page) bool { return named(row.Title, p.Title, s.titles[p.ID]) }},
		{rule: RuleH1, asked: nameKey(row.H1) != "", test: func(p Page) bool { return s.headed(p, row.H1) }},
		{rule: RuleSKU, asked: row.Product && nameKey(row.SKU) != "", test: func(p Page) bool { return named(row.SKU, "", []string{s.skus[p.ID]}) }},
	}
	for _, pool := range [][]Page{s.onSite, s.planned} {
		for _, candidate := range tests {
			if !candidate.asked {
				continue
			}
			if match, decided := decide(candidate.rule, pool, row.Product, candidate.address, candidate.test); decided {
				return match
			}
		}
	}
	return Match{}
}

func (s RowSite) carriesEntity(page Page, name, parent string) bool {
	if page.EntityID == nil {
		return false
	}
	held, known := s.entities[*page.EntityID]
	if !known || nameKey(held.name) != nameKey(name) {
		return false
	}
	return strings.TrimSpace(parent) == "" || nameKey(held.parent) == nameKey(parent)
}

func (s RowSite) headed(page Page, wanted string) bool {
	if page.H1 != "" || len(s.headings[page.ID]) > 0 {
		return named(wanted, page.H1, s.headings[page.ID])
	}
	return named(wanted, page.Title, s.titles[page.ID])
}

func named(wanted, planned string, known []string) bool {
	key := nameKey(wanted)
	for _, held := range append([]string{planned}, known...) {
		if held != "" && nameKey(held) == key {
			return true
		}
	}
	return false
}

func decide(rule MatchRule, pool []Page, product, address bool, test func(Page) bool) (Match, bool) {
	regular, terms := make([]Page, 0, 1), make([]Page, 0, 1)
	for i := range pool {
		isProduct := pool[i].WPType == WPProduct
		if product != isProduct && (product || !address) {
			continue
		}
		if !test(pool[i]) {
			continue
		}
		if pool[i].WPType.Term() {
			terms = append(terms, pool[i])
		} else {
			regular = append(regular, pool[i])
		}
	}
	for _, found := range [][]Page{regular, terms} {
		switch len(found) {
		case 0:
		case 1:
			return Match{Page: found[0], Rule: rule, Candidates: found}, true
		default:
			return Match{Rule: rule, Candidates: found}, true
		}
	}
	return Match{}, false
}
