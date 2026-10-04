package imports

import (
	"github.com/davidmovas/postulator/internal/domain/category"
)

type rootSet map[string]struct{}

func (r rootSet) add(name string) {
	if held := category.Key(name); held != "" {
		r[held] = struct{}{}
	}
}

func (r rootSet) holds(name string) bool {
	_, held := r[category.Key(name)]
	return held
}

func rootsOf(reads []sheetRead, state *siteState) rootSet {
	roots := make(rootSet)
	for i := range state.entities {
		if held := &state.entities[i]; held.ScopeID == nil {
			roots.add(held.Name)
		}
	}
	for i := range reads {
		for _, row := range reads[i].table.Rows {
			for _, level := range reads[i].binding.Levels(row) {
				if !level.Category {
					roots.add(level.Name)
				}
			}
		}
	}
	return roots
}
