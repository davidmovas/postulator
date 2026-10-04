package application

import (
	"context"
	"slices"

	"github.com/davidmovas/postulator/internal/domain/category"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

type categoryLister interface {
	ListBySite(ctx context.Context, siteID string) ([]category.Category, error)
}

type categoryTermLister interface {
	ListBySite(ctx context.Context, siteID string) ([]category.Term, error)
}

type CategoryTree struct {
	listed   []category.Category
	chains   map[string][]category.Category
	children map[string][]string
	terms    map[filedTerm]int64
}

type filedTerm struct {
	categoryID string
	taxonomy   category.Taxonomy
}

func NewCategoryTree(categories []category.Category, terms []category.Term) CategoryTree {
	tree := CategoryTree{
		listed:   slices.Clone(categories),
		chains:   make(map[string][]category.Category, len(categories)),
		children: make(map[string][]string, len(categories)),
		terms:    make(map[filedTerm]int64, len(terms)),
	}
	for i := range categories {
		tree.chains[categories[i].ID] = category.Chain(categories, categories[i].ID)
		if parentID := categories[i].ParentID; parentID != "" {
			tree.children[parentID] = append(tree.children[parentID], categories[i].ID)
		}
	}
	for i := range terms {
		tree.terms[filedTerm{categoryID: terms[i].CategoryID, taxonomy: terms[i].Taxonomy}] = terms[i].TermID
	}
	return tree
}

func LoadCategoryTree(ctx context.Context, siteID string, categories categoryLister, terms categoryTermLister) (CategoryTree, error) {
	if categories == nil {
		return CategoryTree{}, nil
	}
	listed, err := categories.ListBySite(ctx, siteID)
	if err != nil {
		return CategoryTree{}, err
	}
	if len(listed) == 0 || terms == nil {
		return NewCategoryTree(listed, nil), nil
	}
	held, err := terms.ListBySite(ctx, siteID)
	if err != nil {
		return CategoryTree{}, err
	}
	return NewCategoryTree(listed, held), nil
}

func (x CategoryTree) Empty() bool {
	return len(x.listed) == 0
}

func (x CategoryTree) Categories() []category.Category {
	return slices.Clone(x.listed)
}

func (x CategoryTree) Chain(leafID string, taxonomy category.Taxonomy) []dto.Category {
	chain := x.chains[leafID]
	out := make([]dto.Category, 0, len(chain))
	for i := range chain {
		filed := dto.Category{ID: chain[i].ID, Name: chain[i].Name}
		if termID, held := x.TermID(chain[i].ID, taxonomy); held {
			filed.TermID = &termID
		}
		out = append(out, filed)
	}
	return out
}

func (x CategoryTree) TermID(categoryID string, taxonomy category.Taxonomy) (int64, bool) {
	termID, held := x.terms[filedTerm{categoryID: categoryID, taxonomy: taxonomy}]
	return termID, held
}

func (x CategoryTree) Subtree(categoryID string) []string {
	out := []string{categoryID}
	seen := map[string]struct{}{categoryID: {}}
	for at := 0; at < len(out); at++ {
		for _, child := range x.children[out[at]] {
			if _, again := seen[child]; again {
				continue
			}
			seen[child] = struct{}{}
			out = append(out, child)
		}
	}
	return out
}
