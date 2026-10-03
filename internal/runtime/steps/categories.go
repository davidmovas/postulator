package steps

import (
	"context"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	CodeCategoriesForbidden      = "categories_forbidden"
	CodeCategoryRefused          = "category_refused"
	CodeCategoriesNotTaken       = "categories_not_taken"
	CodePageCategoriesNeedPlugin = "page_categories_need_plugin"

	pageCategoriesPlugin = "1.3.0"
	emptyTermName        = "empty_term_name"
	chainSeparator       = " › "
)

type AssignedTerm struct {
	EntityID string `json:"entityId"`
	Name     string `json:"name"`
	TermID   int64  `json:"termId"`
	ParentID int64  `json:"parentId"`
	Created  bool   `json:"created"`
}

type CategoryWrite struct {
	Taxonomy graph.Taxonomy `json:"taxonomy"`
	Terms    []AssignedTerm `json:"terms"`
	Previous []int64        `json:"previous"`
	Added    []int64        `json:"added"`
	Taken    bool           `json:"taken"`
}

func (w *CategoryWrite) termIDs() []int64 {
	ids := make([]int64, 0, len(w.Terms))
	for i := range w.Terms {
		ids = append(ids, w.Terms[i].TermID)
	}
	return ids
}

type categoryPlan struct {
	write    *CategoryWrite
	findings []content.Finding
	send     []int64
}

func (p *categoryPlan) took(page pagemap.Page, carried []int64) {
	if p.write == nil {
		return
	}
	wanted := p.send
	if wanted == nil {
		wanted = p.write.termIDs()
	}
	p.write.Taken = containsAll(carried, wanted)
	if !p.write.Taken {
		p.findings = append(p.findings, notTaken(page, p.write, carried))
	}
}

func ensureCategories(ctx context.Context, deps Deps, client *wp.Client, sc *run.StepContext,
	previous []int64, creating bool) (categoryPlan, error) {
	taxonomy, carried := sc.Page.WPType.Taxonomy()
	if !carried || sc.Page.EntityID == nil || *sc.Page.EntityID == "" {
		return categoryPlan{}, nil
	}
	if readerErr := deps.entityReader(); readerErr != nil {
		return categoryPlan{}, readerErr
	}
	entities, err := deps.Entities.ListBySite(ctx, sc.Run.SiteID)
	if err != nil {
		return categoryPlan{}, err
	}
	chain := graph.CategoryChain(entities, *sc.Page.EntityID)
	if len(chain) == 0 {
		return categoryPlan{}, nil
	}

	if sc.Page.WPType == pagemap.WPPage {
		filed, capErr := pagesCarryCategories(ctx, client)
		if capErr != nil {
			return categoryPlan{}, capErr
		}
		if !filed {
			return categoryPlan{findings: []content.Finding{pageNeedsPlugin(sc.Page, chain)}}, nil
		}
	}

	if storeErr := deps.termStore(); storeErr != nil {
		return categoryPlan{}, storeErr
	}
	walk := termWalk{deps: deps, client: client, sc: sc, chain: chain, taxonomy: taxonomy}
	terms, err := walk.resolve(ctx)
	switch {
	case walk.refused != nil:
		return categoryPlan{findings: []content.Finding{*walk.refused}}, nil
	case err != nil:
		return categoryPlan{}, err
	}
	return planOf(taxonomy, terms, previous, creating), nil
}

func pagesCarryCategories(ctx context.Context, client *wp.Client) (bool, error) {
	capabilities, err := client.Capabilities(ctx)
	switch {
	case err == nil:
		return capabilities.Has(wp.CapabilityPageCategories), nil
	case wp.IsPluginMissing(err):
		return false, nil
	default:
		return false, err
	}
}

type termWalk struct {
	deps     Deps
	client   *wp.Client
	sc       *run.StepContext
	refused  *content.Finding
	stored   map[string]graph.Term
	live     map[int64]wp.Term
	chain    []graph.Entity
	taxonomy graph.Taxonomy
}

func (w *termWalk) resolve(ctx context.Context) ([]AssignedTerm, error) {
	stored, err := storedTerms(ctx, w.deps, w.sc.Run.SiteID, w.taxonomy)
	if err != nil {
		return nil, err
	}
	w.stored = stored
	if w.live, err = w.liveTerms(ctx); err != nil {
		return nil, err
	}

	assigned := make([]AssignedTerm, 0, len(w.chain))
	parent := int64(0)
	for i := range w.chain {
		level, levelErr := w.level(ctx, w.chain[i], parent)
		if levelErr != nil {
			return nil, levelErr
		}
		assigned = append(assigned, level)
		parent = level.TermID
	}
	return assigned, nil
}

func (w *termWalk) answered(err error) error {
	if refused, said := categoryRefusal(w.sc.Page, w.taxonomy, w.chain, err); said {
		w.refused = &refused
	}
	return err
}

func storedTerms(ctx context.Context, deps Deps, siteID string, taxonomy graph.Taxonomy) (map[string]graph.Term, error) {
	listed, err := deps.Terms.ListBySite(ctx, siteID)
	if err != nil {
		return nil, err
	}
	stored := make(map[string]graph.Term, len(listed))
	for i := range listed {
		if listed[i].Taxonomy == taxonomy {
			stored[listed[i].EntityID] = listed[i]
		}
	}
	return stored, nil
}

func (w *termWalk) liveTerms(ctx context.Context) (map[int64]wp.Term, error) {
	ids := make([]int64, 0, len(w.chain))
	for i := range w.chain {
		if held, known := w.stored[w.chain[i].ID]; known && !slices.Contains(ids, held.TermID) {
			ids = append(ids, held.TermID)
		}
	}
	live := make(map[int64]wp.Term, len(ids))
	if len(ids) == 0 {
		return live, nil
	}

	listed, err := w.client.ListTerms(ctx, wp.Taxonomy(w.taxonomy), wp.TermQuery{Include: ids, Page: 1, PerPage: len(ids)})
	if err != nil {
		return nil, w.answered(err)
	}
	for _, term := range listed.Items {
		live[term.ID] = term
	}
	return live, nil
}

func (w *termWalk) level(ctx context.Context, entity graph.Entity, parent int64) (AssignedTerm, error) {
	held, known := w.stored[entity.ID]
	if term, alive := w.live[held.TermID]; known && alive && term.Parent == parent {
		if err := w.keep(ctx, entity.ID, term, held.RunID); err != nil {
			return AssignedTerm{}, err
		}
		return assignedOf(entity.ID, term, held.RunID == w.sc.Run.ID), nil
	}

	term, created, err := w.client.EnsureTerm(ctx, wp.Taxonomy(w.taxonomy), entity.Name, parent)
	if err != nil {
		return AssignedTerm{}, w.answered(err)
	}
	created = created || (known && held.TermID == term.ID && held.RunID == w.sc.Run.ID)
	runID := ""
	if created {
		runID = w.sc.Run.ID
	}
	if keepErr := w.keep(ctx, entity.ID, term, runID); keepErr != nil {
		return AssignedTerm{}, keepErr
	}
	return assignedOf(entity.ID, term, created), nil
}

func (w *termWalk) keep(ctx context.Context, entityID string, term wp.Term, runID string) error {
	record, err := graph.NewTerm(graph.Term{
		EntityID: entityID, SiteID: w.sc.Run.SiteID, Taxonomy: w.taxonomy, TermID: term.ID,
		ParentTermID: term.Parent, Name: term.Name, RunID: runID, SeenAt: w.deps.now(),
	})
	if err != nil {
		return err
	}
	return w.deps.Terms.Upsert(ctx, record)
}

func assignedOf(entityID string, term wp.Term, created bool) AssignedTerm {
	return AssignedTerm{EntityID: entityID, Name: term.Name, TermID: term.ID, ParentID: term.Parent, Created: created}
}

func planOf(taxonomy graph.Taxonomy, terms []AssignedTerm, previous []int64, creating bool) categoryPlan {
	held := append(make([]int64, 0, len(previous)), previous...)
	write := &CategoryWrite{Taxonomy: taxonomy, Terms: terms, Previous: held}
	chain := write.termIDs()
	write.Added = without(chain, held)

	plan := categoryPlan{write: write}
	switch {
	case creating:
		plan.send = chain
	case len(write.Added) > 0:
		plan.send = append(slices.Clone(held), write.Added...)
	}
	return plan
}

func without(ids, taken []int64) []int64 {
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if !slices.Contains(taken, id) && !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return out
}

func containsAll(carried, wanted []int64) bool {
	for _, id := range wanted {
		if !slices.Contains(carried, id) {
			return false
		}
	}
	return true
}

func chainNames(chain []graph.Entity) []string {
	names := make([]string, 0, len(chain))
	for i := range chain {
		names = append(names, chain[i].Name)
	}
	return names
}

func termNames(terms []AssignedTerm) []string {
	names := make([]string, 0, len(terms))
	for i := range terms {
		names = append(names, terms[i].Name)
	}
	return names
}

func categoryDetails(page pagemap.Page, taxonomy graph.Taxonomy, names []string) map[string]any {
	return map[string]any{"pageId": page.ID, "path": page.Path, "taxonomy": string(taxonomy), "categories": names}
}

func pageNeedsPlugin(page pagemap.Page, chain []graph.Entity) content.Finding {
	names := chainNames(chain)
	return content.Finding{
		Severity: content.SeverityWarn,
		Code:     CodePageCategoriesNeedPlugin,
		Message: page.Path + " went up without its categories " + strings.Join(names, chainSeparator) +
			": WordPress pages carry categories only through the Postulator companion plugin " + pageCategoriesPlugin +
			" or later; update the plugin, sync the site and publish the page again",
		Details: categoryDetails(page, graph.TaxonomyCategory, names),
	}
}

func categoryRefusal(page pagemap.Page, taxonomy graph.Taxonomy, chain []graph.Entity, err error) (content.Finding, bool) {
	names := chainNames(chain)
	details := categoryDetails(page, taxonomy, names)
	switch {
	case wp.Forbidden(err):
		details["capability"] = termCapability(taxonomy)
		return content.Finding{
			Severity: content.SeverityWarn,
			Code:     CodeCategoriesForbidden,
			Message: page.Path + " went up without its categories " + strings.Join(names, chainSeparator) +
				": the WordPress user may not create them, which takes the " + termCapability(taxonomy) +
				" capability; create them in wp-admin or use the application password of a user who may, and publish again",
			Details: details,
		}, true
	case errors.IsCode(err, errors.Invalid), errors.IsCode(err, errors.NotFound), wordPressCode(err) == emptyTermName:
		said := wordPressMessage(err)
		details["reason"] = said
		return content.Finding{
			Severity: content.SeverityWarn,
			Code:     CodeCategoryRefused,
			Message: page.Path + " went up without its categories " + strings.Join(names, chainSeparator) +
				": WordPress refused them, saying " + said,
			Details: details,
		}, true
	default:
		return content.Finding{}, false
	}
}

func termCapability(taxonomy graph.Taxonomy) string {
	if taxonomy == graph.TaxonomyProductCategory {
		return "manage_product_terms"
	}
	return "manage_categories"
}

func notTaken(page pagemap.Page, write *CategoryWrite, carried []int64) content.Finding {
	names := termNames(write.Terms)
	details := categoryDetails(page, write.Taxonomy, names)
	details["carried"] = append(make([]int64, 0, len(carried)), carried...)
	return content.Finding{
		Severity: content.SeverityWarn,
		Code:     CodeCategoriesNotTaken,
		Message: "WordPress did not keep the categories " + strings.Join(names, chainSeparator) + " on " + page.Path +
			"; the site may not carry categories for this type of content",
		Details: details,
	}
}

func wordPressCode(err error) string {
	return detailOf(err, "code")
}

func wordPressMessage(err error) string {
	if said := detailOf(err, "wpMessage"); said != "" {
		return said
	}
	return err.Error()
}
