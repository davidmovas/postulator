package imports

import (
	"cmp"
	"slices"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func (b *builder) identity(at int, visiting map[int]bool) string {
	root := b.root(at)
	if known := b.units[root].id; known != "" {
		return known
	}
	if visiting[root] {
		return ""
	}
	visiting[root] = true
	defer delete(visiting, root)
	return b.parentKey(&b.units[root], visiting) + "/" + graph.Key(b.units[root].name)
}

func (b *builder) parentKey(u *unit, visiting map[int]bool) string {
	switch u.parent.kind {
	case refSite:
		return u.parent.site
	case refUnit:
		parent := b.identity(u.parent.unit, visiting)
		if b.units[b.root(u.parent.unit)].id != "" {
			return parent
		}
		return "{" + parent + "}"
	case refName:
		return "?" + graph.Key(u.parent.name)
	case refNone:
	}
	return ""
}

func (b *builder) resolve(name string, context int) (parentRef, FindingCode) {
	if ref, code, inFile := b.inFile(name, context); inFile {
		return ref, code
	}
	return b.onSite(name, context)
}

func (b *builder) inFile(name string, context int) (parentRef, FindingCode, bool) {
	want := graph.Key(name)
	seen := make(map[string]struct{})
	named := make([]int, 0)
	for at := range b.units {
		if graph.Key(b.units[at].name) != want {
			continue
		}
		identity := b.identity(at, map[int]bool{})
		if _, twice := seen[identity]; !twice {
			seen[identity] = struct{}{}
			named = append(named, at)
		}
	}
	switch {
	case len(named) == 0:
		return parentRef{}, "", false
	case len(named) == 1:
		return parentRef{kind: refUnit, unit: named[0]}, "", true
	case context < 0:
		return parentRef{}, CodeAmbiguousParent, true
	}

	group := b.root(b.groups.nodes[context].unit)
	inGroup := slices.DeleteFunc(named, func(at int) bool {
		return b.units[at].parent.kind != refUnit || b.root(b.units[at].parent.unit) != group
	})
	if len(inGroup) == 1 {
		return parentRef{kind: refUnit, unit: inGroup[0]}, "", true
	}
	return parentRef{}, CodeAmbiguousParent, true
}

func (b *builder) onSite(name string, context int) (parentRef, FindingCode) {
	sites := b.state.byName[graph.Key(name)]
	switch {
	case len(sites) == 0:
		return parentRef{}, CodeUnknownParent
	case len(sites) == 1:
		return parentRef{kind: refSite, site: sites[0].ID}, ""
	case context < 0:
		return parentRef{}, CodeAmbiguousParent
	}

	group := graph.Key(b.groups.nodes[context].name)
	inGroup := make([]string, 0, len(sites))
	for i := range sites {
		if scope := sites[i].ScopeID; scope != nil && graph.Key(b.byID[*scope].Name) == group {
			inGroup = append(inGroup, sites[i].ID)
		}
	}
	if len(inGroup) == 1 {
		return parentRef{kind: refSite, site: inGroup[0]}, ""
	}
	return parentRef{}, CodeAmbiguousParent
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

func (b *builder) match() {
	keys := make(map[string]int, len(b.units))
	for _, at := range b.ordered() {
		if !b.sameAsWeakParent(at) {
			b.matchUnit(at, keys)
		}
	}
}

func (b *builder) matchUnit(at int, keys map[string]int) {
	u := &b.units[at]
	identity := b.identity(at, map[int]bool{})
	if into, seen := keys[identity]; seen {
		b.absorb(into, at)
		return
	}
	keys[identity] = at

	found := u.pinned
	if found == "" {
		found = b.find(u)
	}
	if found != "" {
		if into, taken := b.claimed[found]; taken {
			b.absorb(into, at)
			return
		}
		b.claimed[found] = at
		u.matched = found
	}
	u.id = fill(u.matched, id.New())
}

func (b *builder) sameAsWeakParent(at int) bool {
	u := &b.units[at]
	if !u.parent.weak || u.pinned != "" {
		return false
	}
	switch u.parent.kind {
	case refUnit:
		parent := b.root(u.parent.unit)
		if parent == at || graph.Key(b.units[parent].name) != graph.Key(u.name) {
			return false
		}
		b.absorb(parent, at)
		return true
	case refSite:
		if graph.Key(b.byID[u.parent.site].Name) != graph.Key(u.name) {
			return false
		}
		if into, taken := b.claimed[u.parent.site]; taken {
			b.absorb(into, at)
			return true
		}
		u.pinned = u.parent.site
		u.parent = parentRef{}
	case refNone, refName:
	}
	return false
}

func (u *unit) placed() bool {
	return u.parent.kind == refSite || u.parent.kind == refUnit
}

func (b *builder) find(u *unit) string {
	candidates := b.state.byName[graph.Key(u.name)]
	if found := b.underItsParent(u, candidates); found != "" {
		return found
	}

	free := make([]graph.Entity, 0, len(candidates))
	for i := range candidates {
		if _, taken := b.claimed[candidates[i].ID]; !taken {
			free = append(free, candidates[i])
		}
	}
	if u.placed() {
		if orphan := b.orphan(free); orphan != "" {
			return orphan
		}
	}
	if (!u.placed() || u.parent.weak) && len(free) == 1 {
		return free[0].ID
	}
	if !u.placed() && len(candidates) > 1 {
		b.p.noteAt(u.at, string(importmap.FieldEntity), CodeAmbiguousEntity,
			"more than one entity is named "+u.name+" and the row says nothing about which one it means; "+
				"give it a parent or a group")
	}
	return ""
}

func (b *builder) underItsParent(u *unit, candidates []graph.Entity) string {
	scope := b.parentID(u)
	for i := range candidates {
		held := candidates[i].ScopeID
		if (u.placed() && scope != "" && held != nil && *held == scope) || (!u.placed() && held == nil) {
			return candidates[i].ID
		}
	}
	return ""
}

func (b *builder) orphan(free []graph.Entity) string {
	for i := range free {
		if free[i].ScopeID == nil && b.parents[free[i].ID] == 0 {
			return free[i].ID
		}
	}
	return ""
}

func (b *builder) absorb(into, from int) {
	source := &b.units[from]
	source.alias = into
	target := &b.units[into]
	target.kind = fill(target.kind, source.kind)
	target.keywords = target.keywords.Merge(source.keywords)
	target.anchors = graph.Distinct(target.anchors, source.anchors)
	target.related = graph.Distinct(target.related, source.related)
	if target.group < 0 {
		target.group = source.group
	}
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
	return slices.DeleteFunc(b.ordered(), func(at int) bool { return b.units[at].alias >= 0 })
}

func (b *builder) rootNames() map[string]string {
	names := make(map[string]string, len(b.units))
	for at := range b.units {
		if b.units[at].alias >= 0 {
			continue
		}
		if _, named := names[b.units[at].id]; !named {
			names[b.units[at].id] = b.units[at].name
		}
	}
	return names
}

func (b *builder) nameOf(entityID string) string {
	if name, inFile := b.names[entityID]; inFile {
		return name
	}
	return b.byID[entityID].Name
}

func (b *builder) parentName(u *unit) string {
	if parentID := b.parentID(u); parentID != "" {
		return b.nameOf(parentID)
	}
	return ""
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
