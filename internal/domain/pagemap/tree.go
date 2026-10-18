package pagemap

import "slices"

type Node struct {
	Page     Page
	Children []Node
}

func BuildTree(pages []Page) []Node {
	index := NewIndex(pages)
	sorted := index.Pages()
	parentOf := make(map[string]string, len(sorted))
	for i := range sorted {
		page := sorted[i]
		if page.ParentPageID != nil && *page.ParentPageID != page.ID {
			if _, known := index.ByID(*page.ParentPageID); known {
				parentOf[page.ID] = *page.ParentPageID
				continue
			}
		}
		if parent, found := nearestAncestor(page.Path, index); found {
			parentOf[page.ID] = parent.ID
		}
	}
	for i := range sorted {
		if cyclic(sorted[i].ID, parentOf) {
			delete(parentOf, sorted[i].ID)
		}
	}

	children := make(map[string][]Page)
	roots := make([]Page, 0)
	for i := range sorted {
		parent, nested := parentOf[sorted[i].ID]
		if !nested {
			roots = append(roots, sorted[i])
			continue
		}
		children[parent] = append(children[parent], sorted[i])
	}
	return nodes(roots, children)
}

func Cycles(pages []Page) [][]Page {
	index := NewIndex(pages)
	sorted := index.Pages()
	parentOf := make(map[string]string, len(sorted))
	for i := range sorted {
		if parent := sorted[i].ParentPageID; parent != nil {
			if _, known := index.ByID(*parent); known {
				parentOf[sorted[i].ID] = *parent
			}
		}
	}

	placed := make(map[string]struct{}, len(sorted))
	out := make([][]Page, 0)
	for i := range sorted {
		if _, done := placed[sorted[i].ID]; done || !cyclic(sorted[i].ID, parentOf) {
			continue
		}
		loop := loopThrough(sorted[i].ID, parentOf)
		if !slices.Contains(loop, sorted[i].ID) {
			continue
		}
		members := make([]Page, 0, len(loop))
		for _, id := range loop {
			placed[id] = struct{}{}
			page, _ := index.ByID(id)
			members = append(members, page)
		}
		slices.SortFunc(members, byPath)
		out = append(out, members)
	}
	return out
}

func loopThrough(id string, parentOf map[string]string) []string {
	order := map[string]int{}
	walk := make([]string, 0)
	for current, ok := id, true; ok; current, ok = parentOf[current] {
		if at, seen := order[current]; seen {
			return walk[at:]
		}
		order[current] = len(walk)
		walk = append(walk, current)
	}
	return nil
}

func cyclic(id string, parentOf map[string]string) bool {
	seen := map[string]struct{}{id: {}}
	for current, ok := parentOf[id]; ok; current, ok = parentOf[current] {
		if _, loop := seen[current]; loop {
			return true
		}
		seen[current] = struct{}{}
	}
	return false
}

func nearestAncestor(path string, index Index) (parent Page, found bool) {
	for ancestor := ParentPath(path); ancestor != ""; ancestor = ParentPath(ancestor) {
		if p, ok := index.byPath[ancestor]; ok {
			return p, true
		}
	}
	return Page{}, false
}

func nodes(pages []Page, children map[string][]Page) []Node {
	out := make([]Node, 0, len(pages))
	for i := range pages {
		out = append(out, Node{Page: pages[i], Children: nodes(children[pages[i].ID], children)})
	}
	return out
}
