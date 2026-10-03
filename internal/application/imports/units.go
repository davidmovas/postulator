package imports

import (
	"cmp"
	"slices"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/keyword"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/id"
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
	category bool
	pinned   string
	alias    int
	matched  string
	id       string
}

func (u *unit) defaultKind() graph.Kind {
	switch {
	case u.group < 0:
		return graph.KindTopic
	case u.category:
		return graph.KindCategory
	default:
		return graph.KindHub
	}
}

type builder struct {
	state    siteState
	p        *plan
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
}

func newBuilder(state siteState, p *plan, rows []rowDraft, sheet *drafts) *builder {
	b := &builder{
		state: state, p: p, rows: rows, sheet: sheet, groups: groupsOf(rows),
		assigned: make(map[string]int), warned: make(map[string]struct{}),
		claimed: make(map[string]int), parents: make(map[string]int), byID: make(map[string]graph.Entity, len(state.entities)),
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
	b.resolveParents()
	b.match()
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

func (b *builder) addGroupUnits() {
	for at := range b.groups.nodes {
		node := &b.groups.nodes[at]
		u := unit{name: node.name, context: node.parent, group: at, category: node.category}
		if len(node.under) > 0 {
			u.at = b.rows[node.under[0]].at
		}
		if node.parent >= 0 {
			u.parent = parentRef{kind: refUnit, unit: b.groups.nodes[node.parent].unit}
		}
		node.unit = b.add(u)
	}
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
		if draft.technical() {
			continue
		}

		if node, adopted := b.adopted[path]; adopted {
			draft.unit = b.groups.nodes[node].unit
			if b.units[draft.unit].parent.kind == refNone {
				b.units[draft.unit].parent = b.urlParent(path)
			}
			for _, at := range draft.rows {
				b.absorbRow(draft.unit, &b.rows[at])
			}
			continue
		}

		if draft.generated {
			draft.unit = b.add(unit{name: titleFrom(path), parent: b.urlParent(path), context: -1, group: -1})
			continue
		}

		first := &b.rows[draft.rows[0]]
		explicit := ""
		for _, at := range draft.rows {
			explicit = fill(explicit, b.rows[at].name)
		}
		if _, onSite := b.state.held(path); path == pagemap.RootPath && explicit == "" && !onSite {
			continue
		}
		u := unit{name: fill(explicit, first.named()), at: first.at, context: b.contextOf(draft.rows), group: -1}
		if existing, onSite := b.state.held(path); onSite && explicit == "" && existing.EntityID != nil {
			if pinned, known := b.byID[*existing.EntityID]; known {
				u.name, u.pinned = pinned.Name, pinned.ID
			}
		}
		switch {
		case u.context >= 0:
			u.parent = parentRef{kind: refUnit, unit: b.groups.nodes[u.context].unit}
		default:
			u.parent = b.urlParent(path)
		}
		draft.unit = b.add(u)
		for _, at := range draft.rows {
			b.absorbRow(draft.unit, &b.rows[at])
		}
	}
}

func (b *builder) addEntityRows() {
	for i := range b.rows {
		row := &b.rows[i]
		if row.path != "" || row.name == "" {
			continue
		}
		u := unit{name: row.name, at: row.at, context: b.contextOf([]int{i}), group: -1}
		if u.context >= 0 {
			u.parent = parentRef{kind: refUnit, unit: b.groups.nodes[u.context].unit}
		}
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
		if !onSite {
			continue
		}
		if page.EntityID != nil {
			return parentRef{kind: refSite, site: *page.EntityID, weak: true}
		}
		if page.WPID == nil {
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
	b.p.note(draft.row, string(importmap.FieldOwnEntity), CodeTechnicalParent,
		"the page "+draft.path+" is not an entity and is not on the site yet, so the pages under it cannot be "+
			"written until it is published or given an entity")
}

func (b *builder) provisional(at int, visiting map[int]bool) string {
	if visiting[at] {
		return ""
	}
	visiting[at] = true
	defer delete(visiting, at)

	u := &b.units[at]
	prefix := ""
	switch u.parent.kind {
	case refSite:
		prefix = "s:" + u.parent.site
	case refUnit:
		prefix = "u:{" + b.provisional(u.parent.unit, visiting) + "}"
	case refName:
		prefix = "n:" + key(u.parent.name)
	case refNone:
	}
	return prefix + "/" + key(u.name)
}

func (b *builder) resolve(name string, context int) (parentRef, FindingCode) {
	want := key(name)
	files := make(map[string]int)
	order := make([]string, 0)
	for at := range b.units {
		if key(b.units[at].name) != want {
			continue
		}
		provisional := b.provisional(at, map[int]bool{})
		if _, seen := files[provisional]; !seen {
			files[provisional] = at
			order = append(order, provisional)
		}
	}
	switch {
	case len(order) == 1:
		return parentRef{kind: refUnit, unit: files[order[0]]}, ""
	case len(order) > 1:
		if context < 0 {
			return parentRef{}, CodeAmbiguousParent
		}
		narrowed := make([]int, 0)
		for _, provisional := range order {
			at := files[provisional]
			if b.units[at].parent.kind == refUnit && b.root(b.units[at].parent.unit) == b.root(b.groups.nodes[context].unit) {
				narrowed = append(narrowed, at)
			}
		}
		if len(narrowed) == 1 {
			return parentRef{kind: refUnit, unit: narrowed[0]}, ""
		}
		return parentRef{}, CodeAmbiguousParent
	}

	sites := b.state.byName[want]
	switch {
	case len(sites) == 1:
		return parentRef{kind: refSite, site: sites[0].ID}, ""
	case len(sites) > 1:
		if context >= 0 {
			narrowed := make([]string, 0)
			for i := range sites {
				if scope := sites[i].ScopeID; scope != nil && key(b.byID[*scope].Name) == key(b.groups.nodes[context].name) {
					narrowed = append(narrowed, sites[i].ID)
				}
			}
			if len(narrowed) == 1 {
				return parentRef{kind: refSite, site: narrowed[0]}, ""
			}
		}
		return parentRef{}, CodeAmbiguousParent
	}
	return parentRef{}, CodeUnknownParent
}

func (b *builder) resolveParents() {
	for at := range b.units {
		u := &b.units[at]
		if u.parent.kind != refName {
			continue
		}
		name, origin := u.parent.name, u.parent.at
		resolved, code := b.resolve(name, u.context)
		switch code {
		case CodeUnknownParent:
			b.p.noteAt(origin, string(importmap.FieldParentEntity), CodeUnknownParent,
				"the parent entity is not in the file and not on the site: "+name)
		case CodeAmbiguousParent:
			b.p.noteAt(origin, string(importmap.FieldParentEntity), CodeAmbiguousParent,
				"more than one entity is named "+name+"; put the row in that entity's group or name a parent that only one carries")
		}
		resolved.at = origin
		u.parent = resolved
	}
}

func (b *builder) depth(at int, visiting map[int]bool) int {
	u := &b.units[at]
	if u.parent.kind != refUnit || visiting[at] {
		return 0
	}
	visiting[at] = true
	defer delete(visiting, at)
	return 1 + b.depth(u.parent.unit, visiting)
}

func (b *builder) ordered() []int {
	order := make([]int, len(b.units))
	depths := make([]int, len(b.units))
	for at := range b.units {
		order[at] = at
		depths[at] = b.depth(at, map[int]bool{})
	}
	slices.SortStableFunc(order, func(x, y int) int { return cmp.Compare(depths[x], depths[y]) })
	return order
}

func (b *builder) parentIdentity(u *unit) (identity string, present bool) {
	switch u.parent.kind {
	case refSite:
		return "s:" + u.parent.site, true
	case refUnit:
		parent := b.root(u.parent.unit)
		if b.units[parent].matched != "" {
			return "s:" + b.units[parent].matched, true
		}
		return "u:" + b.units[parent].id, true
	case refNone, refName:
	}
	return "", false
}

func (b *builder) parentSite(u *unit) string {
	switch u.parent.kind {
	case refSite:
		return u.parent.site
	case refUnit:
		return b.units[b.root(u.parent.unit)].matched
	case refNone, refName:
	}
	return ""
}

func (b *builder) match() {
	keys := make(map[string]int, len(b.units))
	for _, at := range b.ordered() {
		u := &b.units[at]
		if b.sameAsWeakParent(at) {
			continue
		}
		identity, present := b.parentIdentity(u)
		unitKey := identity + "/" + key(u.name)
		if into, seen := keys[unitKey]; seen {
			b.absorb(into, at)
			continue
		}
		keys[unitKey] = at

		found := u.pinned
		if found == "" {
			found = b.find(u, present)
		}
		if found != "" {
			if into, taken := b.claimed[found]; taken {
				b.absorb(into, at)
				continue
			}
			b.claimed[found] = at
			u.matched = found
		}
		u.id = fill(u.matched, id.New())
	}
}

func (b *builder) sameAsWeakParent(at int) bool {
	u := &b.units[at]
	if !u.parent.weak || u.pinned != "" {
		return false
	}
	switch u.parent.kind {
	case refUnit:
		parent := b.root(u.parent.unit)
		if parent == at || key(b.units[parent].name) != key(u.name) {
			return false
		}
		b.absorb(parent, at)
		return true
	case refSite:
		if key(b.byID[u.parent.site].Name) != key(u.name) {
			return false
		}
		if into, taken := b.claimed[u.parent.site]; taken {
			b.absorb(into, at)
			return true
		}
		u.pinned = u.parent.site
		u.parent = parentRef{}
		return false
	case refNone, refName:
	}
	return false
}

func (b *builder) find(u *unit, present bool) string {
	candidates := b.state.byName[key(u.name)]
	scope := b.parentSite(u)
	for i := range candidates {
		held := candidates[i].ScopeID
		if (present && scope != "" && held != nil && *held == scope) || (!present && held == nil) {
			return candidates[i].ID
		}
	}

	free := make([]int, 0, len(candidates))
	for i := range candidates {
		if _, taken := b.claimed[candidates[i].ID]; !taken {
			free = append(free, i)
		}
	}
	if present {
		for _, at := range free {
			if candidates[at].ScopeID == nil && b.parents[candidates[at].ID] == 0 {
				return candidates[at].ID
			}
		}
	}
	if (!present || u.parent.weak) && len(free) == 1 {
		return candidates[free[0]].ID
	}
	if !present && len(candidates) > 1 {
		b.p.noteAt(u.at, string(importmap.FieldEntity), CodeAmbiguousEntity,
			"more than one entity is named "+u.name+" and the row says nothing about which one it means; "+
				"give it a parent or a group")
	}
	return ""
}

func (b *builder) absorb(into, from int) {
	source := &b.units[from]
	source.alias = into
	target := &b.units[into]
	target.kind = fill(target.kind, source.kind)
	target.keywords = target.keywords.Merge(source.keywords)
	target.anchors = union(target.anchors, source.anchors)
	target.related = union(target.related, source.related)
	if target.group < 0 {
		target.group = source.group
	}
	target.category = target.category || source.category
	if target.context < 0 {
		target.context = source.context
	}
}

func (b *builder) entityID(at int) string {
	return b.units[b.root(at)].id
}

func (b *builder) parentID(u *unit) string {
	switch u.parent.kind {
	case refSite:
		return u.parent.site
	case refUnit:
		return b.entityID(u.parent.unit)
	case refNone, refName:
	}
	return ""
}

func (b *builder) roots() []int {
	out := make([]int, 0, len(b.units))
	for _, at := range b.ordered() {
		if b.units[at].alias < 0 {
			out = append(out, at)
		}
	}
	return out
}

func (b *builder) pageUnit(path string) (int, bool) {
	if draft, planned := b.sheet.pages[path]; planned && draft.unit >= 0 {
		return b.root(draft.unit), true
	}
	if at, given := b.assigned[path]; given {
		return b.root(at), true
	}
	if node, adopted := b.adopted[path]; adopted {
		return b.root(b.groups.nodes[node].unit), true
	}
	return 0, false
}
