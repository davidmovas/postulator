package imports

import (
	"github.com/davidmovas/postulator/internal/domain/category"
	"github.com/davidmovas/postulator/internal/domain/graph"
)

type keySet map[string]struct{}

func (s keySet) add(name string) bool {
	held := category.Key(name)
	if _, known := s[held]; known || held == "" {
		return false
	}
	s[held] = struct{}{}
	return true
}

func (s keySet) holds(name string) bool {
	_, held := s[category.Key(name)]
	return held
}

func heldAsRoot(held *graph.Entity) bool {
	return held.ScopeID == nil && (held.Kind == graph.KindHub || held.Kind == graph.KindCategory)
}

func namesOf(reads []sheetRead, state *siteState) (roots, levels keySet) {
	roots, levels = make(keySet), make(keySet)
	for i := range state.entities {
		if held := &state.entities[i]; heldAsRoot(held) {
			roots.add(held.Name)
		}
	}
	for i := range reads {
		for _, row := range reads[i].table.Rows {
			for _, level := range reads[i].binding.Levels(row) {
				if level.Category {
					levels.add(level.Name)
				} else {
					roots.add(level.Name)
				}
			}
		}
	}
	return roots, levels
}

func (s keySet) takeRoots(after *siteState, levels keySet) bool {
	grew := false
	for i := range after.entities {
		if held := &after.entities[i]; heldAsRoot(held) && levels.holds(held.Name) && s.add(held.Name) {
			grew = true
		}
	}
	return grew
}
