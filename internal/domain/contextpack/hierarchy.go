package contextpack

import (
	"cmp"
	"slices"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

type TreeParent struct {
	PageID   string `json:"pageId,omitempty"`
	Path     string `json:"path,omitempty"`
	EntityID string `json:"entityId,omitempty"`
}

type ParentDisagreement struct {
	EntityID     string
	PageID       string
	Path         string
	GraphParents []string
	Tree         TreeParent
}

func (m Map) ParentDisagreements(pageIDs []string) []ParentDisagreement {
	chosen := make(map[string]struct{}, len(pageIDs))
	for _, pageID := range pageIDs {
		chosen[pageID] = struct{}{}
	}
	out := make([]ParentDisagreement, 0)
	for _, entity := range m.snapshot.Graph.Entities() {
		if entity.CanonicalPageID == nil {
			continue
		}
		pageID := *entity.CanonicalPageID
		if _, asked := chosen[pageID]; len(chosen) > 0 && !asked {
			continue
		}
		page, found := m.snapshot.Index.ByID(pageID)
		if !found || !live(page) {
			continue
		}
		tree, onMap := m.TreeParentOf(pageID)
		if !onMap {
			continue
		}
		paged := m.pagedAncestors(entity.ID)
		if tree.EntityID == "" && len(paged) == 0 || tree.EntityID != "" && slices.Contains(paged, tree.EntityID) {
			continue
		}
		disagreement := ParentDisagreement{EntityID: entity.ID, PageID: page.ID, Path: page.Path, Tree: tree}
		for _, parent := range m.snapshot.Graph.Parents(entity.ID, 1) {
			disagreement.GraphParents = append(disagreement.GraphParents, parent.ID)
		}
		out = append(out, disagreement)
	}
	slices.SortFunc(out, func(a, b ParentDisagreement) int {
		return cmp.Or(cmp.Compare(a.Path, b.Path), cmp.Compare(a.EntityID, b.EntityID))
	})
	return out
}

func (m Map) pagedAncestors(entityID string) []string {
	seen := map[string]struct{}{entityID: {}}
	frontier := []string{entityID}
	out := make([]string, 0)
	for len(frontier) > 0 {
		next := make([]string, 0)
		for _, at := range frontier {
			for _, parent := range m.snapshot.Graph.Parents(at, 1) {
				if _, twice := seen[parent.ID]; twice {
					continue
				}
				seen[parent.ID] = struct{}{}
				if parent.CanonicalPageID != nil && m.holds(*parent.CanonicalPageID) {
					out = append(out, parent.ID)
					continue
				}
				next = append(next, parent.ID)
			}
		}
		frontier = next
	}
	return out
}

func (m Map) holds(pageID string) bool {
	if _, folded := m.folded[pageID]; folded {
		return true
	}
	_, onMap := m.parent[pageID]
	return onMap
}

func (m Map) TreeParentOf(pageID string) (TreeParent, bool) {
	start := pageID
	if hub, folded := m.folded[pageID]; folded {
		start = hub
	}
	if _, onMap := m.parent[start]; !onMap {
		return TreeParent{}, false
	}
	seen := map[string]struct{}{start: {}}
	for at := m.parent[start]; at != ""; at = m.parent[at] {
		if _, looped := seen[at]; looped {
			break
		}
		seen[at] = struct{}{}
		ancestor, found := m.snapshot.Index.ByID(at)
		if !found || ancestor.Path == pagemap.RootPath {
			break
		}
		if ancestor.EntityID != nil {
			return TreeParent{PageID: ancestor.ID, Path: ancestor.Path, EntityID: *ancestor.EntityID}, true
		}
	}
	return TreeParent{}, true
}
