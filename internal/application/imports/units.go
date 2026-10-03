package imports

import (
	"time"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/keyword"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

type refKind int

const (
	refNone refKind = iota
	refUnit
	refSite
	refName
)

type parentRef struct {
	kind refKind
	unit int
	site string
	name string
	weak bool
	at   importmap.Origin
}

type unit struct {
	name     string
	kind     string
	keywords keyword.List
	anchors  []string
	related  []string
	at       importmap.Origin
	parent   parentRef
	context  int
	group    int
	chain    []string
	dropped  []string
	pinned   string
	alias    int
	matched  string
	id       string
}

func (u *unit) defaultKind() graph.Kind {
	if u.group < 0 {
		return graph.KindTopic
	}
	return graph.KindHub
}

type builder struct {
	state    siteState
	p        *plan
	now      time.Time
	rows     []rowDraft
	sheet    *drafts
	groups   *groups
	units    []unit
	adopted  map[string]int
	assigned map[string]int
	warned   map[string]struct{}
	claimed  map[string]int
	parents  map[string]int
	byID     map[string]graph.Entity
	order    []int
	names    map[string]string
	resolved map[string]graph.Entity
	final    []pagemap.Page
}

func newBuilder(state siteState, p *plan, rows []rowDraft, sheet *drafts, now time.Time) *builder {
	b := &builder{
		state: state, p: p, now: now, rows: rows, sheet: sheet, groups: groupsOf(rows),
		assigned: make(map[string]int), warned: make(map[string]struct{}), claimed: make(map[string]int),
		parents: make(map[string]int), byID: make(map[string]graph.Entity, len(state.entities)),
	}
	for i := range state.entities {
		b.byID[state.entities[i].ID] = state.entities[i]
	}
	for i := range state.edges {
		if state.edges[i].Kind == graph.EdgeParent && state.edges[i].Status == graph.StatusApproved {
			b.parents[state.edges[i].FromEntityID]++
		}
	}
	return b
}

func (b *builder) build() {
	b.addGroupUnits()
	b.adopted = b.groups.adopt(b.rows, b.sheet, b.state)
	b.addPageUnits()
	b.addEntityRows()
	b.place()
	b.resolveParents()
	b.match()
	b.order = b.roots()
	b.names = b.rootNames()
}

func (b *builder) add(u unit) int {
	u.alias = -1
	b.units = append(b.units, u)
	return len(b.units) - 1
}

func (b *builder) root(at int) int {
	for b.units[at].alias >= 0 {
		at = b.units[at].alias
	}
	return at
}

func (b *builder) absorbRow(at int, row *rowDraft) {
	u := &b.units[at]
	u.kind = fill(u.kind, row.kind)
	u.keywords = u.keywords.Merge(row.keywords)
	u.anchors = union(u.anchors, row.anchors)
	u.related = union(u.related, row.related)
	if u.at.Row == 0 {
		u.at = row.at
	}
	if row.parent != "" && u.parent.kind != refName && (u.group < 0 || u.parent.kind == refNone || u.parent.weak) {
		u.parent = parentRef{kind: refName, name: row.parent, at: row.at}
	}
}

func (b *builder) absorbRows(draft *pageDraft) {
	for _, at := range draft.rows {
		b.absorbRow(draft.unit, &b.rows[at])
	}
}

func (b *builder) addGroupUnits() {
	for at := range b.groups.nodes {
		node := &b.groups.nodes[at]
		u := unit{name: node.name, parent: b.groupParent(node.parent), context: node.parent, group: at}
		if len(node.under) > 0 {
			u.at = b.rows[node.under[0]].at
		}
		node.unit = b.add(u)
	}
}

func (b *builder) groupParent(node int) parentRef {
	if node < 0 {
		return parentRef{}
	}
	return parentRef{kind: refUnit, unit: b.groups.nodes[node].unit}
}

func (b *builder) contextOf(rows []int) int {
	for _, at := range rows {
		if node, grouped := b.groups.ofRow[at]; grouped {
			return node
		}
	}
	return -1
}

func (b *builder) addPageUnits() {
	for _, path := range b.sheet.sortedPaths() {
		draft := b.sheet.pages[path]
		node, adopted := b.adopted[path]
		switch {
		case draft.technical():
		case adopted:
			b.adoptPage(draft, node)
		case draft.generated:
			draft.unit = b.add(unit{name: titleFrom(path), parent: b.urlParent(path), context: -1, group: -1})
		default:
			b.addRowUnit(draft)
		}
	}
}

func (b *builder) adoptPage(draft *pageDraft, node int) {
	draft.unit = b.groups.nodes[node].unit
	if b.units[draft.unit].parent.kind == refNone {
		b.units[draft.unit].parent = b.urlParent(draft.path)
	}
	b.absorbRows(draft)
}

func (b *builder) explicitName(draft *pageDraft) string {
	explicit := ""
	for _, at := range draft.rows {
		explicit = fill(explicit, b.rows[at].name)
	}
	return explicit
}

func (b *builder) addRowUnit(draft *pageDraft) {
	explicit := b.explicitName(draft)
	existing, onSite := b.state.held(draft.path)
	if draft.path == pagemap.RootPath && explicit == "" && !onSite {
		return
	}

	first := &b.rows[draft.rows[0]]
	u := unit{
		name: fill(explicit, first.named()), at: first.at, context: b.contextOf(draft.rows), group: -1,
		chain: draft.chain, dropped: draft.dropped,
	}
	if onSite && explicit == "" && existing.EntityID != nil {
		if pinned, known := b.byID[*existing.EntityID]; known {
			u.name, u.pinned = pinned.Name, pinned.ID
		}
	}
	if u.parent = b.groupParent(u.context); u.parent.kind == refNone {
		u.parent = b.urlParent(draft.path)
	}
	draft.unit = b.add(u)
	b.absorbRows(draft)
}

func (b *builder) addEntityRows() {
	for i := range b.rows {
		row := &b.rows[i]
		if row.path != "" || row.name == "" {
			continue
		}
		u := unit{name: row.name, at: row.at, context: b.contextOf([]int{i}), group: -1, chain: row.chain, dropped: row.dropped}
		u.parent = b.groupParent(u.context)
		b.absorbRow(b.add(u), row)
	}
}

func (b *builder) urlParent(path string) parentRef {
	for parent := pagemap.ParentPath(path); parent != "" && parent != pagemap.RootPath; parent = pagemap.ParentPath(parent) {
		if draft, planned := b.sheet.pages[parent]; planned {
			if draft.technical() {
				b.warnTechnical(draft)
				continue
			}
			if draft.unit >= 0 {
				return parentRef{kind: refUnit, unit: draft.unit, weak: true}
			}
			continue
		}
		page, onSite := b.state.held(parent)
		switch {
		case !onSite:
		case page.EntityID != nil:
			return parentRef{kind: refSite, site: *page.EntityID, weak: true}
		case page.WPID == nil:
			return parentRef{kind: refUnit, unit: b.ancestorUnit(page), weak: true}
		}
	}
	return parentRef{}
}

func (b *builder) ancestorUnit(page pagemap.Page) int {
	if at, known := b.assigned[page.Path]; known {
		return at
	}
	at := b.add(unit{name: titleFrom(page.Path), parent: b.urlParent(page.Path), context: -1, group: -1})
	b.assigned[page.Path] = at
	return at
}

func (b *builder) warnTechnical(draft *pageDraft) {
	if _, done := b.warned[draft.path]; done {
		return
	}
	b.warned[draft.path] = struct{}{}
	if existing, onSite := b.state.held(draft.path); onSite && existing.WPID != nil {
		return
	}
	b.p.noteAt(draft.at, string(importmap.FieldOwnEntity), CodeTechnicalParent,
		"the page "+draft.path+" is not an entity and is not on the site yet, so the pages under it cannot be "+
			"written until it is published or given an entity")
}
