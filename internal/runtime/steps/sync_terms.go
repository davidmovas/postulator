package steps

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/graph"
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
	entities, err := deps.Entities.ListBySite(ctx, siteID)
	if err != nil {
		return err
	}
	stored, err := deps.Terms.ListBySite(ctx, siteID)
	if err != nil {
		return err
	}

	chains := filedChains(entities)
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
			resolved: make(map[string]int64), kept: make(map[string]graph.Term),
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

func filedChains(entities []graph.Entity) [][]graph.Entity {
	chains := graph.CategoryChains(entities)
	filed := make([][]graph.Entity, 0)
	for i := range entities {
		if entities[i].SiteCategory {
			filed = append(filed, chains[entities[i].ID])
		}
	}
	return filed
}

func syncedTaxonomies(owner site.Site) []graph.Taxonomy {
	if owner.Commerce == site.CommerceReady {
		return []graph.Taxonomy{graph.TaxonomyCategory, graph.TaxonomyProductCategory}
	}
	return []graph.Taxonomy{graph.TaxonomyCategory}
}

func heldIn(stored []graph.Term, taxonomy graph.Taxonomy) map[string]graph.Term {
	held := make(map[string]graph.Term)
	for i := range stored {
		if stored[i].Taxonomy == taxonomy {
			held[stored[i].EntityID] = stored[i]
		}
	}
	return held
}

func everyTerm(ctx context.Context, client *wp.Client, taxonomy graph.Taxonomy) ([]wp.Term, error) {
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

func (x termIndex) under(parent int64, name string) (wp.Term, bool) {
	for _, term := range x.children[parent] {
		if wp.SameTermName(term.Name, name) {
			return term, true
		}
	}
	return wp.Term{}, false
}

type termAdoption struct {
	seenAt   time.Time
	held     map[string]graph.Term
	resolved map[string]int64
	kept     map[string]graph.Term
	index    termIndex
	siteID   string
	taxonomy graph.Taxonomy
}

func (a *termAdoption) walk(chain []graph.Entity) {
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

func (a *termAdoption) match(entity graph.Entity, parent int64) int64 {
	held, known := a.held[entity.ID]
	if term, alive := a.index.byID[held.TermID]; known && alive && term.Parent == parent {
		a.keep(entity.ID, term, held.RunID)
		return term.ID
	}

	term, found := a.index.under(parent, entity.Name)
	if !found {
		return 0
	}
	runID := ""
	if known && held.TermID == term.ID {
		runID = held.RunID
	}
	a.keep(entity.ID, term, runID)
	return term.ID
}

func (a *termAdoption) keep(entityID string, term wp.Term, runID string) {
	a.kept[entityID] = graph.Term{
		EntityID: entityID, SiteID: a.siteID, Taxonomy: a.taxonomy, TermID: term.ID, ParentTermID: term.Parent,
		Name: term.Name, RunID: runID, SeenAt: a.seenAt,
	}
}

func (a *termAdoption) apply(ctx context.Context, deps Deps) error {
	records := make([]graph.Term, 0, len(a.kept))
	for entityID := range a.kept {
		record, err := graph.NewTerm(a.kept[entityID])
		if err != nil {
			return err
		}
		records = append(records, record)
	}
	gone := make([]string, 0)
	for entityID := range a.held {
		_, alive := a.index.byID[a.held[entityID].TermID]
		if _, adopted := a.kept[entityID]; !alive && !adopted {
			gone = append(gone, entityID)
		}
	}
	if len(records) == 0 && len(gone) == 0 {
		return nil
	}

	return deps.inUnit(ctx, func(c context.Context) error {
		for _, entityID := range gone {
			if err := deps.Terms.Delete(c, entityID, a.taxonomy); err != nil && !errors.IsCode(err, errors.NotFound) {
				return err
			}
		}
		for i := range records {
			if err := deps.Terms.Upsert(c, records[i]); err != nil {
				return err
			}
		}
		return nil
	})
}

func termsUnread(owner site.Site, taxonomy graph.Taxonomy, err error) content.Finding {
	return content.Finding{
		Severity: content.SeverityWarn,
		Code:     CodeCategoriesUnread,
		Message: owner.Name + " would not list its " + string(taxonomy) + " terms, so the categories already on it " +
			"were not matched to the graph: " + wordPressMessage(err),
		Details: map[string]any{"siteId": owner.ID, "taxonomy": string(taxonomy), "reason": wordPressMessage(err)},
	}
}
