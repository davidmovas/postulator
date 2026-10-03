package graph

import (
	"slices"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/category"
)

type Taxonomy = category.Taxonomy

const (
	TaxonomyCategory        = category.TaxonomyCategory
	TaxonomyProductCategory = category.TaxonomyProductCategory
)

type Term struct {
	EntityID     string
	SiteID       string
	Taxonomy     Taxonomy
	TermID       int64
	ParentTermID int64
	Name         string
	RunID        string
	SeenAt       time.Time
}

func NewTerm(t Term) (Term, error) {
	t.Name = strings.TrimSpace(t.Name)

	switch {
	case t.EntityID == "":
		return Term{}, invalid("term entity id must not be empty", "entityId")
	case t.SiteID == "":
		return Term{}, invalid("term site id must not be empty", "siteId")
	case !t.Taxonomy.Valid():
		return Term{}, invalid("term taxonomy is not recognized", "taxonomy")
	case t.TermID <= 0:
		return Term{}, invalid("term id must be positive", "termId")
	case t.ParentTermID < 0:
		return Term{}, invalid("parent term id must not be negative", "parentTermId")
	case t.ParentTermID == t.TermID:
		return Term{}, invalid("a term cannot sit under itself", "parentTermId")
	case t.Name == "":
		return Term{}, invalid("term name must not be empty", "name")
	case t.SeenAt.IsZero():
		return Term{}, invalid("a term needs the time it was seen", "seenAt")
	}
	return t, nil
}

func CategoryChain(entities []Entity, entityID string) []Entity {
	return categoryChain(entitiesByID(entities), entityID)
}

func CategoryChains(entities []Entity) map[string][]Entity {
	byID := entitiesByID(entities)
	chains := make(map[string][]Entity, len(entities))
	for i := range entities {
		chains[entities[i].ID] = categoryChain(byID, entities[i].ID)
	}
	return chains
}

func entitiesByID(entities []Entity) map[string]Entity {
	byID := make(map[string]Entity, len(entities))
	for i := range entities {
		byID[entities[i].ID] = entities[i]
	}
	return byID
}

func categoryChain(byID map[string]Entity, entityID string) []Entity {
	chain := make([]Entity, 0)
	walked := make(map[string]struct{})
	for at, held := byID[entityID]; held; at, held = scopeOf(byID, at) {
		if _, again := walked[at.ID]; again {
			break
		}
		walked[at.ID] = struct{}{}
		if at.SiteCategory {
			chain = append(chain, at)
		}
	}
	slices.Reverse(chain)
	return chain
}

func scopeOf(byID map[string]Entity, entity Entity) (Entity, bool) {
	if entity.ScopeID == nil {
		return Entity{}, false
	}
	parent, held := byID[*entity.ScopeID]
	return parent, held
}
