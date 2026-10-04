package imports

import (
	"cmp"
	"context"
	"slices"
	"time"

	"github.com/davidmovas/postulator/internal/domain/category"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type sheetRead struct {
	mapping importmap.Mapping
	table   importmap.Table
	binding importmap.Binding
}

type workbook struct {
	plans  []plan
	unused []category.Category
	report PreviewReport
}

func (w *workbook) add(p *plan) {
	w.plans = append(w.plans, *p)
	r, more := &w.report, &p.report
	r.Columns = append(r.Columns, more.Columns...)
	r.Pages = append(r.Pages, more.Pages...)
	r.Entities = append(r.Entities, more.Entities...)
	r.Groups = append(r.Groups, more.Groups...)
	r.Categories = append(r.Categories, more.Categories...)
	r.Edges = append(r.Edges, more.Edges...)
	r.Warnings = append(r.Warnings, more.Warnings...)
	r.Errors = append(r.Errors, more.Errors...)
	r.Cannibalization = append(r.Cannibalization, more.Cannibalization...)
	r.Skipped += more.Skipped
}

func (w *workbook) broken() bool {
	return len(w.report.Errors) > 0
}

func (w *workbook) counts() Counts {
	var total Counts
	for i := range w.plans {
		each := w.plans[i].counts()
		total.EntitiesCreated += each.EntitiesCreated
		total.EntitiesUpdated += each.EntitiesUpdated
		total.EdgesCreated += each.EdgesCreated
		total.PagesCreated += each.PagesCreated
		total.PagesUpdated += each.PagesUpdated
		total.CategoriesCreated += each.CategoriesCreated
		total.Skipped += each.Skipped
	}
	total.CategoriesDeleted = len(w.unused)
	return total
}

func (s *Service) sweep(ctx context.Context, book *workbook, state *siteState) error {
	terms, err := s.deps.CategoryTerms.ListBySite(ctx, state.siteID)
	if err != nil {
		return err
	}
	book.unused = unused(state.categories, state.pages, terms)
	held := newShelf(state.categories)
	for i := range book.unused {
		book.report.Categories = append(book.report.Categories, PreviewCategory{
			Path: held.trail(book.unused[i].ID), Action: string(CategoryDelete),
		})
	}
	return nil
}

func (w *workbook) sheets() []string {
	out := make([]string, 0, len(w.plans))
	for i := range w.plans {
		out = append(out, w.plans[i].read...)
	}
	return out
}

func (w *workbook) rows() int {
	total := 0
	for i := range w.plans {
		total += w.plans[i].rows
	}
	return total
}

func (s *Service) compute(ctx context.Context, req PreviewRequest) (workbook, error) {
	if err := s.requireSite(ctx, req.SiteID); err != nil {
		return workbook{}, err
	}
	reads, err := s.reads(ctx, req)
	if err != nil {
		return workbook{}, err
	}
	state, err := s.state(ctx, req.SiteID)
	if err != nil {
		return workbook{}, err
	}

	now := s.now()
	roots, levels := namesOf(reads, &state)
	book, after, err := s.planAll(ctx, state, roots, reads, now)
	for err == nil && roots.takeRoots(&after, levels) {
		book, after, err = s.planAll(ctx, state, roots, reads, now)
	}
	if err != nil {
		return workbook{}, err
	}
	if sweepErr := s.sweep(ctx, &book, &after); sweepErr != nil {
		return workbook{}, sweepErr
	}
	book.report.settle()
	return book, nil
}

func (s *Service) planAll(
	ctx context.Context, state siteState, roots keySet, reads []sheetRead, now time.Time,
) (workbook, siteState, error) {
	book := workbook{plans: make([]plan, 0, len(reads))}
	for i := range reads {
		planned, planErr := s.plan(ctx, state, roots, &reads[i], now)
		if planErr != nil {
			return workbook{}, siteState{}, planErr
		}
		next, settleErr := state.after(&planned, now)
		switch {
		case settleErr == nil:
		case errors.IsCode(settleErr, errors.Conflict):
			_, message := errors.Describe(settleErr)
			planned.noteAt(planned.whole(), "", CodeScopeClash, message)
		default:
			return workbook{}, siteState{}, settleErr
		}
		state = next
		book.add(&planned)
	}
	return book, state, nil
}

func (s *Service) reads(ctx context.Context, req PreviewRequest) ([]sheetRead, error) {
	if len(req.Sheets) == 0 {
		mapping, table, err := s.read(ctx, req.SiteID, req.Path, req.Mapping.domain())
		if err != nil {
			return nil, err
		}
		read, err := bound(mapping, table)
		if err != nil {
			return nil, err
		}
		return []sheetRead{read}, nil
	}
	if given := req.Mapping.domain(); given.ID != "" || !unmapped(given) {
		return nil, errors.New(errors.Invalid, "give one mapping for the file or one for each sheet, not both").
			WithDetail("field", "sheets")
	}

	ordered, err := s.inWorkbookOrder(req.Path, req.Sheets)
	if err != nil {
		return nil, err
	}
	reads := make([]sheetRead, 0, len(ordered))
	rows := 0
	for i := range ordered {
		given := ordered[i].Mapping.domain()
		given.Options.Sheets = sheetsNamed(ordered[i].Sheet)
		mapping, table, readErr := s.read(ctx, req.SiteID, req.Path, given)
		if readErr != nil {
			return nil, readErr
		}
		if rows += len(table.Rows); rows > s.deps.MaxRows {
			return nil, errors.New(errors.Invalid, "the sheets carry more rows together than the import.maxRows setting allows").
				WithDetail("maxRows", s.deps.MaxRows).WithDetail("sheet", ordered[i].Sheet)
		}
		read, bindErr := bound(mapping, table)
		if bindErr != nil {
			return nil, bindErr
		}
		reads = append(reads, read)
	}
	return reads, nil
}

func bound(mapping importmap.Mapping, table importmap.Table) (sheetRead, error) {
	binding, err := mapping.Bind(table.Headers)
	if err != nil {
		return sheetRead{}, err
	}
	return sheetRead{mapping: mapping, table: table, binding: binding}, nil
}

func (s *Service) inWorkbookOrder(path string, asked []SheetMapping) ([]SheetMapping, error) {
	held, err := s.deps.Tables.Sheets(path)
	if err != nil {
		return nil, err
	}
	position := make(map[string]int, len(held))
	names := make([]string, 0, len(held))
	for i := range held {
		position[held[i].Name] = i
		names = append(names, held[i].Name)
	}

	seen := make(map[string]struct{}, len(asked))
	for i := range asked {
		name := asked[i].Sheet
		if _, known := position[name]; !known {
			return nil, errors.New(errors.Invalid, "the workbook has no sheet by that name").
				WithDetail("field", "sheets").WithDetail("sheet", name).WithDetail("sheets", names)
		}
		if _, twice := seen[name]; twice {
			return nil, errors.New(errors.Invalid, "the sheet is named twice").WithDetail("field", "sheets").WithDetail("sheet", name)
		}
		seen[name] = struct{}{}
	}

	ordered := slices.Clone(asked)
	slices.SortStableFunc(ordered, func(a, b SheetMapping) int { return cmp.Compare(position[a.Sheet], position[b.Sheet]) })
	return ordered, nil
}

func fold[T any](held, planned []T, idOf func(*T) string) (folded []T, at map[string]int) {
	folded = slices.Clone(held)
	at = make(map[string]int, len(folded)+len(planned))
	for i := range folded {
		at[idOf(&folded[i])] = i
	}
	for i := range planned {
		if found, replaced := at[idOf(&planned[i])]; replaced {
			folded[found] = planned[i]
			continue
		}
		at[idOf(&planned[i])] = len(folded)
		folded = append(folded, planned[i])
	}
	return folded, at
}

func (s siteState) after(p *plan, now time.Time) (siteState, error) {
	planned := make([]graph.Entity, 0, len(p.entities))
	for i := range p.entities {
		planned = append(planned, p.entities[i].entity)
	}
	entities, entityAt := fold(s.entities, planned, func(e *graph.Entity) string { return e.ID })
	for _, owned := range p.canonical {
		at := entityAt[owned.entityID]
		entities[at].CanonicalPageID, entities[at].UpdatedAt = &owned.pageID, now
	}

	written := make([]pagemap.Page, 0, len(p.pages))
	for i := range p.pages {
		written = append(written, p.pages[i].page)
	}
	pages, _ := fold(s.pages, written, func(page *pagemap.Page) string { return page.ID })

	edges := slices.Concat(s.edges, p.edges)
	moved, err := graph.Settle(entities, edges)
	for i := range moved {
		at := entityAt[moved[i].ID]
		entities[at].ScopeID, entities[at].UpdatedAt = moved[i].ScopeID, now
	}
	sortStored(entities, edges, pages)
	return newSiteState(s.siteID, entities, edges, pages, slices.Concat(s.categories, p.categories)), err
}
