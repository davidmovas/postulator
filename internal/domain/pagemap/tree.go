package pagemap

type Node struct {
	Page     Page
	Children []Node
}

func BuildTree(pages []Page) []Node {
	index := NewIndex(pages)
	children := make(map[string][]Page)
	roots := make([]Page, 0)
	sorted := index.Pages()
	for i := range sorted {
		parent, found := nearestAncestor(sorted[i].Path, index)
		if !found {
			roots = append(roots, sorted[i])
			continue
		}
		children[parent.ID] = append(children[parent.ID], sorted[i])
	}
	return nodes(roots, children)
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
