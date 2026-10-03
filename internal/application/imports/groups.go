package imports

import (
	"strings"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

type groupNode struct {
	name     string
	parent   int
	unit     int
	rows     []int
	under    []int
	page     string
	category bool
}

type groups struct {
	nodes []groupNode
	byKey map[string]int
	ofRow map[int]int
}

func groupsOf(rows []rowDraft) *groups {
	g := &groups{byKey: make(map[string]int), ofRow: make(map[int]int)}
	for i := range rows {
		parent := -1
		path := ""
		for _, level := range rows[i].levels {
			path += "\x00" + key(level.Name)
			at, known := g.byKey[path]
			if !known {
				at = len(g.nodes)
				g.nodes = append(g.nodes, groupNode{name: strings.TrimSpace(level.Name), parent: parent, unit: -1})
				g.byKey[path] = at
			}
			g.nodes[at].under = append(g.nodes[at].under, i)
			g.nodes[at].category = g.nodes[at].category || level.Category
			parent = at
		}
		if parent >= 0 {
			g.nodes[parent].rows = append(g.nodes[parent].rows, i)
			g.ofRow[i] = parent
		}
	}
	return g
}

func (g *groups) chain(node int) []string {
	out := make([]string, 0)
	for at := node; at >= 0; at = g.nodes[at].parent {
		out = append([]string{g.nodes[at].name}, out...)
	}
	return out
}

func ancestorOf(ancestor, path string) bool {
	return ancestor != path && strings.HasPrefix(path, ancestor)
}

func distinctPaths(rows []rowDraft, picked []int) []string {
	out := make([]string, 0, len(picked))
	seen := make(map[string]struct{}, len(picked))
	for _, at := range picked {
		path := rows[at].path
		if path == "" {
			continue
		}
		if _, dup := seen[path]; dup {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	return out
}

func (g *groups) adopt(rows []rowDraft, sheet *drafts, state siteState) map[string]int {
	adopted := make(map[string]int)
	for at := range g.nodes {
		node := &g.nodes[at]
		path, found := g.ownRow(node, rows, sheet, adopted)
		if !found {
			path, found = g.freePage(node, rows, sheet, state, adopted)
		}
		if found {
			adopted[path] = at
			node.page = path
		}
	}
	return adopted
}

func (g *groups) ownRow(node *groupNode, rows []rowDraft, sheet *drafts, adopted map[string]int) (string, bool) {
	eligible := make([]int, 0, len(node.rows))
	for _, at := range node.rows {
		row := &rows[at]
		if row.path == "" || sheet.pages[row.path].technical() {
			continue
		}
		if _, taken := adopted[row.path]; taken {
			continue
		}
		if row.name != "" && key(row.name) != key(node.name) {
			continue
		}
		eligible = append(eligible, at)
	}

	named := make([]int, 0)
	slugged := make([]int, 0)
	for _, at := range eligible {
		if key(rows[at].named()) == key(node.name) {
			named = append(named, at)
		}
		if strings.EqualFold(pagemap.Slug(rows[at].path), slugOf(node.name)) {
			slugged = append(slugged, at)
		}
	}
	for _, picked := range [][]int{named, slugged} {
		switch paths := distinctPaths(rows, picked); len(paths) {
		case 0:
		case 1:
			return paths[0], true
		default:
			return "", false
		}
	}

	all := distinctPaths(rows, node.rows)
	if len(all) < 2 {
		return "", false
	}
	for _, candidate := range distinctPaths(rows, eligible) {
		above := true
		for _, other := range all {
			if other != candidate && !ancestorOf(candidate, other) {
				above = false
				break
			}
		}
		if above {
			return candidate, true
		}
	}
	return "", false
}

func (g *groups) freePage(node *groupNode, rows []rowDraft, sheet *drafts, state siteState, adopted map[string]int) (string, bool) {
	under := distinctPaths(rows, node.under)
	if len(under) == 0 {
		return "", false
	}
	slug := slugOf(node.name)

	candidates := make([]string, 0)
	consider := func(path string) {
		if _, taken := adopted[path]; taken || !strings.EqualFold(pagemap.Slug(path), slug) {
			return
		}
		for _, below := range under {
			if !ancestorOf(path, below) {
				return
			}
		}
		candidates = append(candidates, path)
	}
	for _, path := range sheet.sortedPaths() {
		if draft := sheet.pages[path]; draft.generated || g.freeRows(draft, rows) {
			consider(path)
		}
	}
	for path := range state.byPath {
		if _, planned := sheet.pages[path]; planned {
			continue
		}
		if state.byPath[path].EntityID == nil {
			consider(path)
		}
	}
	if len(candidates) != 1 {
		return "", false
	}
	return candidates[0], true
}

func (g *groups) freeRows(draft *pageDraft, rows []rowDraft) bool {
	if draft.technical() || len(draft.rows) == 0 {
		return false
	}
	for _, at := range draft.rows {
		if rows[at].name != "" || len(rows[at].levels) > 0 {
			return false
		}
	}
	return true
}
