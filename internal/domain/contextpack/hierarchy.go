package contextpack

import "github.com/davidmovas/postulator/internal/domain/pagemap"

type TreeParent struct {
	PageID   string `json:"pageId,omitempty"`
	Path     string `json:"path,omitempty"`
	EntityID string `json:"entityId,omitempty"`
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
