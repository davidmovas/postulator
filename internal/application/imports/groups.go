package imports

import (
	"cmp"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

type groupNode struct {
	name   string
	parent int
	unit   int
	rows   []int
	under  []int
	page   string
}

func (n *groupNode) namedBy(row *rowDraft) bool {
	return key(row.named()) == key(n.name)
}

func (n *groupNode) sluggedBy(row *rowDraft) bool {
	return strings.EqualFold(pagemap.Slug(row.path), slugOf(n.name))
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
		for _, name := range rows[i].roots {
			path += "\x00" + key(name)
			at, known := g.byKey[path]
			if !known {
				at = len(g.nodes)
				g.nodes = append(g.nodes, groupNode{name: strings.TrimSpace(name), parent: parent, unit: -1})
				g.byKey[path] = at
			}
			g.nodes[at].under = append(g.nodes[at].under, i)
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
	eligible := g.eligible(node, rows, sheet, adopted)
	for _, evidence := range []func(*rowDraft) bool{node.namedBy, node.sluggedBy} {
		switch paths := distinctPaths(rows, picked(eligible, rows, evidence)); len(paths) {
		case 0:
		case 1:
			return paths[0], true
		default:
			return "", false
		}
	}
	return aboveTheRest(distinctPaths(rows, eligible), distinctPaths(rows, node.rows))
}

func (g *groups) eligible(node *groupNode, rows []rowDraft, sheet *drafts, adopted map[string]int) []int {
	out := make([]int, 0, len(node.rows))
	for _, at := range node.rows {
		row := &rows[at]
		if row.path == "" || sheet.pages[row.path].technical() {
			continue
		}
		if _, taken := adopted[row.path]; taken {
			continue
		}
		if row.name == "" || key(row.name) == key(node.name) {
			out = append(out, at)
		}
	}
	return out
}

func picked(candidates []int, rows []rowDraft, evidence func(*rowDraft) bool) []int {
	out := make([]int, 0, len(candidates))
	for _, at := range candidates {
		if evidence(&rows[at]) {
			out = append(out, at)
		}
	}
	return out
}

func aboveTheRest(candidates, all []string) (string, bool) {
	if len(all) < 2 {
		return "", false
	}
	for _, candidate := range candidates {
		if aboveAll(candidate, all) {
			return candidate, true
		}
	}
	return "", false
}

func aboveAll(candidate string, paths []string) bool {
	for _, other := range paths {
		if other != candidate && !ancestorOf(candidate, other) {
			return false
		}
	}
	return true
}

func (g *groups) freePage(node *groupNode, rows []rowDraft, sheet *drafts, state siteState, adopted map[string]int) (string, bool) {
	under := distinctPaths(rows, node.under)
	if len(under) == 0 {
		return "", false
	}

	candidates := make([]string, 0)
	consider := func(path string) {
		if _, taken := adopted[path]; taken || !strings.EqualFold(pagemap.Slug(path), slugOf(node.name)) {
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
		if _, planned := sheet.pages[path]; !planned && state.byPath[path].EntityID == nil {
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

func (b *builder) groupPages() map[string]string {
	owned := make(map[string]string)
	for i := range b.final {
		entityID, path := b.final[i].EntityID, b.final[i].Path
		if entityID == nil {
			continue
		}
		if held, seen := owned[*entityID]; !seen || path < held {
			owned[*entityID] = path
		}
	}
	return owned
}

func (b *builder) reportGroups() {
	owned := b.groupPages()
	for at := range b.groups.nodes {
		node := &b.groups.nodes[at]
		origin := b.units[node.unit].at
		view := PreviewGroup{
			Sheet: b.p.sheetAt(origin), Path: b.groups.chain(at), Page: cmp.Or(node.page, owned[b.entityID(node.unit)]),
			Rows: len(node.under),
		}
		b.p.report.Groups = append(b.p.report.Groups, view)
		if view.Page == "" {
			b.p.noteAt(origin, "", CodeGroupWithoutPage,
				"the group "+strings.Join(view.Path, " › ")+" has no page of its own in the sheet or on the site; "+
					"it is kept as an entity, and the links of the pages under it pass over it")
		}
	}
}
