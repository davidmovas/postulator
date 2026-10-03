package application

import (
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

type CategoryIndex struct {
	chains map[string][]graph.Entity
	terms  map[categoryTermKey]int64
}

type categoryTermKey struct {
	entityID string
	taxonomy graph.Taxonomy
}

func NewCategoryIndex(entities []graph.Entity, terms []graph.Term) CategoryIndex {
	index := CategoryIndex{chains: graph.CategoryChains(entities), terms: make(map[categoryTermKey]int64, len(terms))}
	for i := range terms {
		index.terms[categoryTermKey{entityID: terms[i].EntityID, taxonomy: terms[i].Taxonomy}] = terms[i].TermID
	}
	return index
}

func (x CategoryIndex) Of(entityID string, taxonomy graph.Taxonomy) []dto.Category {
	chain := x.chains[entityID]
	categories := make([]dto.Category, 0, len(chain))
	for i := range chain {
		category := dto.Category{EntityID: chain[i].ID, Name: chain[i].Name}
		if termID, held := x.terms[categoryTermKey{entityID: chain[i].ID, taxonomy: taxonomy}]; held {
			category.TermID = &termID
		}
		categories = append(categories, category)
	}
	return categories
}
