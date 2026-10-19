package contextpack

import (
	"cmp"
	"slices"

	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
)

type TypeCount struct {
	Name    string          `json:"name"`
	Preset  pagetype.Preset `json:"preset"`
	Pages   int             `json:"pages"`
	Planned int             `json:"planned"`
}

type Hub struct {
	PageID   string `json:"pageId"`
	Path     string `json:"path"`
	Title    string `json:"title"`
	Pages    int    `json:"pages"`
	Children []Hub  `json:"children"`
}

type CodeCount struct {
	Code  finding.Code `json:"code"`
	Count int          `json:"count"`
}

type Overview struct {
	Pages    int         `json:"pages"`
	OnSite   int         `json:"onSite"`
	Planned  int         `json:"planned"`
	Unmapped int         `json:"unmapped"`
	Empty    int         `json:"empty"`
	Types    []TypeCount `json:"types"`
	Hubs     []Hub       `json:"hubs"`
	Findings []CodeCount `json:"findings"`
}

func (s Snapshot) Overview() Overview {
	out := Overview{Types: []TypeCount{}, Hubs: []Hub{}, Findings: s.codes()}
	counts := make(map[string]*TypeCount)
	order := make([]string, 0)
	pages := s.Index.Pages()
	for i := range pages {
		page := &pages[i]
		if !live(*page) {
			continue
		}
		out.Pages++
		if onSite(*page) {
			out.OnSite++
		} else {
			out.Planned++
		}
		if s.Unmapped(*page) {
			out.Unmapped++
		}
		if s.Empty(*page) {
			out.Empty++
		}
		kind := s.typeOf(*page)
		count, seen := counts[kind.ID]
		if !seen {
			count = &TypeCount{Name: kind.Name, Preset: kind.Preset}
			counts[kind.ID] = count
			order = append(order, kind.ID)
		}
		count.Pages++
		if !onSite(*page) {
			count.Planned++
		}
	}
	for _, typeID := range order {
		out.Types = append(out.Types, *counts[typeID])
	}
	slices.SortStableFunc(out.Types, func(a, b TypeCount) int {
		return cmp.Or(cmp.Compare(b.Pages, a.Pages), cmp.Compare(a.Name, b.Name))
	})
	out.Hubs = s.hubs()
	return out
}

func (s Snapshot) Unmapped(page pagemap.Page) bool {
	return page.EntityID == nil && s.writable(page)
}

func (s Snapshot) Empty(page pagemap.Page) bool {
	return onSite(page) && s.writable(page) && s.firstSentence(page) == ""
}

func (s Snapshot) writable(page pagemap.Page) bool {
	return live(page) && !page.WPType.Term() && page.TechnicalReason == pagemap.TechnicalNone &&
		s.typeOf(page).Preset != pagetype.Technical
}

func (s Snapshot) codes() []CodeCount {
	counts := make(map[finding.Code]int)
	for i := range s.Findings {
		if s.Findings[i].Status == finding.Open {
			counts[s.Findings[i].Code]++
		}
	}
	out := make([]CodeCount, 0, len(counts))
	for code, count := range counts {
		out = append(out, CodeCount{Code: code, Count: count})
	}
	slices.SortFunc(out, func(a, b CodeCount) int {
		return cmp.Or(cmp.Compare(b.Count, a.Count), cmp.Compare(a.Code, b.Code))
	})
	return out
}

func hubLike(preset pagetype.Preset) bool {
	return preset == pagetype.Hub || preset == pagetype.Catalog
}

func (s Snapshot) hubs() []Hub {
	tree := s.tree()
	byHub := make(map[string][]Hub)
	top := make([]Hub, 0)
	pages := s.Index.Pages()
	for i := range pages {
		page := &pages[i]
		if !live(*page) || page.Path == pagemap.RootPath || !hubLike(s.typeOf(*page).Preset) {
			continue
		}
		hub := Hub{PageID: page.ID, Path: page.Path, Title: titleOf(*page), Pages: tree.descendants(page.ID), Children: []Hub{}}
		above, found := s.hubAbove(*page)
		if !found {
			top = append(top, hub)
			continue
		}
		byHub[above.ID] = append(byHub[above.ID], hub)
	}
	for i := range top {
		top[i].Children = byHub[top[i].PageID]
		if top[i].Children == nil {
			top[i].Children = []Hub{}
		}
		slices.SortFunc(top[i].Children, largestFirst)
	}
	slices.SortFunc(top, largestFirst)
	return top
}

func largestFirst(a, b Hub) int {
	return cmp.Or(cmp.Compare(b.Pages, a.Pages), cmp.Compare(a.Path, b.Path))
}

func (s Snapshot) hubAbove(page pagemap.Page) (pagemap.Page, bool) {
	trail := s.trail(page)
	for i := range trail {
		if trail[i].Path != pagemap.RootPath && hubLike(s.typeOf(trail[i]).Preset) {
			return trail[i], true
		}
	}
	return pagemap.Page{}, false
}

type tree map[string][]string

func (s Snapshot) tree() tree {
	out := make(tree)
	pages := s.Index.Pages()
	for i := range pages {
		page := &pages[i]
		if live(*page) && page.ParentPageID != nil && *page.ParentPageID != page.ID {
			out[*page.ParentPageID] = append(out[*page.ParentPageID], page.ID)
		}
	}
	return out
}

func (t tree) descendants(pageID string) int {
	seen := map[string]struct{}{pageID: {}}
	queue := slices.Clone(t[pageID])
	count := 0
	for len(queue) > 0 {
		next := queue[0]
		queue = queue[1:]
		if _, looped := seen[next]; looped {
			continue
		}
		seen[next] = struct{}{}
		count++
		queue = append(queue, t[next]...)
	}
	return count
}
