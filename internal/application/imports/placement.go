package imports

import (
	"slices"

	"github.com/davidmovas/postulator/internal/domain/category"
	"github.com/davidmovas/postulator/internal/domain/graph"
)

type namesakes struct {
	units map[string][]int
	sites map[string][]graph.Entity
}

func (b *builder) namesakes() *namesakes {
	found := &namesakes{
		units: make(map[string][]int, len(b.units)),
		sites: make(map[string][]graph.Entity, len(b.state.entities)),
	}
	for at := range b.units {
		named := category.Key(b.units[at].name)
		found.units[named] = append(found.units[named], at)
	}
	for i := range b.state.entities {
		named := category.Key(b.state.entities[i].Name)
		found.sites[named] = append(found.sites[named], b.state.entities[i])
	}
	return found
}

func (b *builder) place() {
	known := b.namesakes()
	wanted := make(map[int]parentRef)
	for at := range b.units {
		u := &b.units[at]
		if u.group >= 0 || u.parent.kind == refName {
			continue
		}
		if ref, found := b.underCategory(at, known); found {
			wanted[at] = ref
			continue
		}
		if u.context < 0 {
			if ref, found := b.underDroppedRoot(at, known); found {
				wanted[at] = ref
			}
		}
	}
	for at, ref := range wanted {
		b.units[at].parent = ref
	}
}

func (b *builder) underCategory(at int, known *namesakes) (parentRef, bool) {
	u := &b.units[at]
	own := category.Key(u.name)
	for level := len(u.chain) - 1; level >= 0; level-- {
		if category.Key(u.chain[level]) == own {
			continue
		}
		above := ""
		if level > 0 {
			above = u.chain[level-1]
		}
		if ref, found := b.namedLike(u.chain[level], at, known, above, false); found {
			return ref, true
		}
	}
	return parentRef{}, false
}

func (b *builder) underDroppedRoot(at int, known *namesakes) (parentRef, bool) {
	u := &b.units[at]
	own := category.Key(u.name)
	for level := len(u.dropped) - 1; level >= 0; level-- {
		if category.Key(u.dropped[level]) == own {
			continue
		}
		if ref, found := b.namedLike(u.dropped[level], at, known, "", true); found {
			return ref, true
		}
	}
	return parentRef{}, false
}

func (b *builder) namedLike(name string, self int, known *namesakes, above string, top bool) (parentRef, bool) {
	named := category.Key(name)
	if inFile := b.distinct(known.units[named], self); len(inFile) > 0 {
		if len(inFile) > 1 {
			inFile = b.keepUnits(inFile, above, top)
		}
		if len(inFile) == 1 {
			return parentRef{kind: refUnit, unit: inFile[0], weak: true}, true
		}
		return parentRef{}, false
	}

	onSite := known.sites[named]
	if len(onSite) > 1 {
		onSite = b.keepSites(onSite, above, top)
	}
	if len(onSite) == 1 {
		return parentRef{kind: refSite, site: onSite[0].ID, weak: true}, true
	}
	return parentRef{}, false
}

func (b *builder) distinct(candidates []int, self int) []int {
	out := make([]int, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, at := range candidates {
		if at == self {
			continue
		}
		identity := b.identity(at, map[int]bool{})
		if _, twice := seen[identity]; twice {
			continue
		}
		seen[identity] = struct{}{}
		out = append(out, at)
	}
	return out
}

func (b *builder) keepUnits(candidates []int, above string, top bool) []int {
	want := category.Key(above)
	return slices.DeleteFunc(slices.Clone(candidates), func(at int) bool {
		parent := &b.units[at].parent
		if top {
			return parent.kind != refNone
		}
		return want == "" || category.Key(b.refName(parent)) != want
	})
}

func (b *builder) keepSites(candidates []graph.Entity, above string, top bool) []graph.Entity {
	want := category.Key(above)
	return slices.DeleteFunc(slices.Clone(candidates), func(held graph.Entity) bool {
		if top {
			return held.ScopeID != nil
		}
		return want == "" || held.ScopeID == nil || category.Key(b.byID[*held.ScopeID].Name) != want
	})
}

func (b *builder) refName(ref *parentRef) string {
	switch ref.kind {
	case refUnit:
		return b.units[b.root(ref.unit)].name
	case refSite:
		return b.byID[ref.site].Name
	case refName:
		return ref.name
	case refNone:
	}
	return ""
}
