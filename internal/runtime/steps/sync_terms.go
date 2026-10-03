package steps

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/domain/category"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	CodeCategoriesUnread = "categories_unread"

	termsPerPage = 100
)

func adoptTerms(ctx context.Context, deps Deps, client *wp.Client, siteID string, state *SiteSyncResult) error {
	if err := deps.categoryStores(); err != nil {
		return err
	}
	owner, err := deps.Sites.Get(ctx, siteID)
	if err != nil {
		return err
	}
	categories, err := deps.Categories.ListBySite(ctx, siteID)
	if err != nil {
		return err
	}
	stored, err := deps.CategoryTerms.ListBySite(ctx, siteID)
	if err != nil {
		return err
	}

	chains := categoryChains(categories)
	for _, taxonomy := range syncedTaxonomies(owner) {
		held := heldIn(stored, taxonomy)
		if len(chains) == 0 && len(held) == 0 {
			continue
		}
		live, listErr := everyTerm(ctx, client, taxonomy)
		if listErr != nil {
			if wp.Forbidden(listErr) || wp.StoreAbsent(listErr) {
				state.Findings = append(state.Findings, termsUnread(owner, taxonomy, listErr))
				continue
			}
			return listErr
		}

		matched := termAdoption{
			siteID: siteID, taxonomy: taxonomy, seenAt: deps.now(), held: held, index: indexTerms(live),
			resolved: make(map[string]int64), kept: make(map[string]category.Term),
		}
		for i := range chains {
			matched.walk(chains[i])
		}
		if applyErr := matched.apply(ctx, deps); applyErr != nil {
			return applyErr
		}
	}
	return nil
}

func categoryChains(categories []category.Category) [][]category.Category {
	chains := make([][]category.Category, 0, len(categories))
	for i := range categories {
		chains = append(chains, category.Chain(categories, categories[i].ID))
	}
	return chains
}

func syncedTaxonomies(owner site.Site) []category.Taxonomy {
	if owner.Commerce == site.CommerceReady {
		return []category.Taxonomy{category.TaxonomyCategory, category.TaxonomyProductCategory}
	}
	return []category.Taxonomy{category.TaxonomyCategory}
}

func heldIn(stored []category.Term, taxonomy category.Taxonomy) map[string]category.Term {
	held := make(map[string]category.Term)
	for i := range stored {
		if stored[i].Taxonomy == taxonomy {
			held[stored[i].CategoryID] = stored[i]
		}
	}
	return held
}

func everyTerm(ctx context.Context, client *wp.Client, taxonomy category.Taxonomy) ([]wp.Term, error) {
	query := wp.TermQuery{Page: 1, PerPage: termsPerPage}
	every := make([]wp.Term, 0)
	for {
		listed, err := client.ListTerms(ctx, wp.Taxonomy(taxonomy), query)
		if err != nil {
			return nil, err
		}
		every = append(every, listed.Items...)
		if !listed.HasMore || len(listed.Items) == 0 {
			return every, nil
		}
		query.Page++
	}
}

type termIndex struct {
	byID     map[int64]wp.Term
	children map[int64][]wp.Term
}

func indexTerms(live []wp.Term) termIndex {
	index := termIndex{byID: make(map[int64]wp.Term, len(live)), children: make(map[int64][]wp.Term)}
	for _, term := range live {
		index.byID[term.ID] = term
		index.children[term.Parent] = append(index.children[term.Parent], term)
	}
	return index
}

func (x termIndex) under(parent int64, key string) (wp.Term, bool) {
	for _, term := range x.children[parent] {
		if key != "" && category.Key(term.Name) == key {
			return term, true
		}
	}
	return wp.Term{}, false
}

type termAdoption struct {
	seenAt   time.Time
	held     map[string]category.Term
	resolved map[string]int64
	kept     map[string]category.Term
	index    termIndex
	siteID   string
	taxonomy category.Taxonomy
}

func (a *termAdoption) walk(chain []category.Category) {
	parent := int64(0)
	for i := range chain {
		termID, done := a.resolved[chain[i].ID]
		if !done {
			termID = a.match(chain[i], parent)
			a.resolved[chain[i].ID] = termID
		}
		if termID == 0 {
			return
		}
		parent = termID
	}
}

func (a *termAdoption) match(filed category.Category, parent int64) int64 {
	held, known := a.held[filed.ID]
	if term, alive := a.index.byID[held.TermID]; known && alive && term.Parent == parent {
		a.keep(filed.ID, term, held.RunID)
		return term.ID
	}

	term, found := a.index.under(parent, filed.Key)
	if !found {
		return 0
	}
	runID := ""
	if known && held.TermID == term.ID {
		runID = held.RunID
	}
	a.keep(filed.ID, term, runID)
	return term.ID
}

func (a *termAdoption) keep(categoryID string, term wp.Term, runID string) {
	a.kept[categoryID] = category.Term{
		CategoryID: categoryID, SiteID: a.siteID, Taxonomy: a.taxonomy, TermID: term.ID, ParentTermID: term.Parent,
		Name: term.Name, RunID: runID, SeenAt: a.seenAt,
	}
}

func (a *termAdoption) apply(ctx context.Context, deps Deps) error {
	records := make([]category.Term, 0, len(a.kept))
	for categoryID := range a.kept {
		record, err := category.NewTerm(a.kept[categoryID])
		if err != nil {
			return err
		}
		records = append(records, record)
	}
	gone := make([]string, 0)
	for categoryID := range a.held {
		_, alive := a.index.byID[a.held[categoryID].TermID]
		if _, adopted := a.kept[categoryID]; !alive && !adopted {
			gone = append(gone, categoryID)
		}
	}
	if len(records) == 0 && len(gone) == 0 {
		return nil
	}

	return deps.inUnit(ctx, func(c context.Context) error {
		for _, categoryID := range gone {
			if err := deps.CategoryTerms.Delete(c, categoryID, a.taxonomy); err != nil && !errors.IsCode(err, errors.NotFound) {
				return err
			}
		}
		for i := range records {
			if err := deps.CategoryTerms.Upsert(c, records[i]); err != nil {
				return err
			}
		}
		return nil
	})
}

func termsUnread(owner site.Site, taxonomy category.Taxonomy, err error) content.Finding {
	return content.Finding{
		Severity: content.SeverityWarn,
		Code:     CodeCategoriesUnread,
		Message: owner.Name + " would not list its " + string(taxonomy) + " terms, so the categories already on it " +
			"were not matched to its category records: " + wordPressMessage(err),
		Details: map[string]any{"siteId": owner.ID, "taxonomy": string(taxonomy), "reason": wordPressMessage(err)},
	}
}
