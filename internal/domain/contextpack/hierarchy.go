package contextpack

import (
	"cmp"
	"slices"

	"github.com/davidmovas/postulator/internal/domain/graph"
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

type AdoptionSkip string

const (
	AdoptionLoop      AdoptionSkip = "loop"
	AdoptionNameTaken AdoptionSkip = "name_taken"
)

type Adoption struct {
	EntityID string
	Name     string
	PageID   string
	Path     string
	From     []string
	To       string
	ToPath   string
	Skip     AdoptionSkip
}

type DisagreementSubject struct {
	EntityID     string   `json:"entityId"`
	Entity       string   `json:"entity"`
	Path         string   `json:"path"`
	GraphIDs     []string `json:"graphIds"`
	Graph        []string `json:"graph"`
	TreeEntityID string   `json:"treeEntityId"`
	Tree         string   `json:"tree"`
	TreePath     string   `json:"treePath"`
}

func (m Map) SubjectOf(d ParentDisagreement) DisagreementSubject {
	subject := DisagreementSubject{
		EntityID: d.EntityID, Entity: m.snapshot.Graph.Label(d.EntityID), Path: d.Path,
		GraphIDs: make([]string, 0, len(d.GraphParents)), Graph: make([]string, 0, len(d.GraphParents)),
		TreeEntityID: d.Tree.EntityID, TreePath: d.Tree.Path,
	}
	for _, parentID := range d.GraphParents {
		subject.GraphIDs = append(subject.GraphIDs, parentID)
		subject.Graph = append(subject.Graph, m.snapshot.Graph.Label(parentID))
	}
	if d.Tree.EntityID != "" {
		subject.Tree = m.snapshot.Graph.Label(d.Tree.EntityID)
	}
	return subject
}

func (m Map) Adoptions(pageIDs, entityIDs []string) []Adoption {
	disagreements := m.ParentDisagreements(pageIDs)
	if len(entityIDs) > 0 {
		disagreements = slices.DeleteFunc(disagreements, func(d ParentDisagreement) bool { return !slices.Contains(entityIDs, d.EntityID) })
	}
	depths := make(map[string]int, len(disagreements))
	for i := range disagreements {
		depths[disagreements[i].PageID] = m.depthOf(disagreements[i].PageID)
	}
	slices.SortStableFunc(disagreements, func(a, b ParentDisagreement) int {
		return cmp.Or(cmp.Compare(depths[a.PageID], depths[b.PageID]), cmp.Compare(a.Path, b.Path))
	})

	entities := m.snapshot.Graph.Entities()
	parents := make(map[string][]string, len(entities))
	scopes := make(map[string]string, len(entities))
	keys := make(map[string]string, len(entities))
	for i := range entities {
		for _, parent := range m.snapshot.Graph.Parents(entities[i].ID, 1) {
			parents[entities[i].ID] = append(parents[entities[i].ID], parent.ID)
		}
		if entities[i].ScopeID != nil {
			scopes[entities[i].ID] = *entities[i].ScopeID
		}
		keys[entities[i].ID] = graph.Key(entities[i].Name)
	}

	out := make([]Adoption, 0, len(disagreements))
	for i := range disagreements {
		d := disagreements[i]
		entity, _ := m.snapshot.Graph.Entity(d.EntityID)
		step := Adoption{
			EntityID: d.EntityID, Name: entity.Name, PageID: d.PageID, Path: d.Path, From: d.GraphParents,
			To: d.Tree.EntityID, ToPath: d.Tree.Path,
		}
		switch {
		case step.To != "" && reaches(parents, step.To, step.EntityID):
			step.Skip = AdoptionLoop
		case taken(scopes, keys, step.To, step.EntityID):
			step.Skip = AdoptionNameTaken
		default:
			delete(parents, step.EntityID)
			if step.To != "" {
				parents[step.EntityID] = []string{step.To}
			}
			scopes[step.EntityID] = step.To
		}
		out = append(out, step)
	}
	return out
}

func (m Map) depthOf(pageID string) int {
	at := pageID
	if hub, folded := m.folded[pageID]; folded {
		at = hub
	}
	depth := 0
	seen := map[string]struct{}{}
	for at = m.parent[at]; at != ""; at = m.parent[at] {
		if _, looped := seen[at]; looped {
			break
		}
		seen[at] = struct{}{}
		depth++
	}
	return depth
}

func reaches(parents map[string][]string, from, target string) bool {
	seen := map[string]struct{}{}
	queue := []string{from}
	for len(queue) > 0 {
		at := queue[0]
		queue = queue[1:]
		if at == target {
			return true
		}
		if _, twice := seen[at]; twice {
			continue
		}
		seen[at] = struct{}{}
		queue = append(queue, parents[at]...)
	}
	return false
}

func taken(scopes, keys map[string]string, scope, self string) bool {
	for entityID, key := range keys {
		if entityID != self && key == keys[self] && scopes[entityID] == scope {
			return true
		}
	}
	return false
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
