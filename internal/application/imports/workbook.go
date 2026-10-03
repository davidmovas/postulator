package imports

import (
	"cmp"
	"context"
	"slices"
	"time"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type sheetRead struct {
	mapping importmap.Mapping
	table   importmap.Table
}

type workbook struct {
	plans  []plan
	report PreviewReport
}

func (w *workbook) add(p *plan) {
	w.plans = append(w.plans, *p)
	r, more := &w.report, &p.report
	r.Columns = append(r.Columns, more.Columns...)
	r.Pages = append(r.Pages, more.Pages...)
	r.Entities = append(r.Entities, more.Entities...)
	r.Groups = append(r.Groups, more.Groups...)
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
		total.Skipped += each.Skipped
	}
	return total
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
	book := workbook{plans: make([]plan, 0, len(reads))}
	for i := range reads {
		planned, planErr := s.plan(ctx, state, reads[i].table, reads[i].mapping, now)
		if planErr != nil {
			return workbook{}, planErr
		}
		next, settleErr := state.after(&planned, now)
		switch {
		case settleErr == nil:
		case errors.IsCode(settleErr, errors.Conflict):
			_, message := errors.Describe(settleErr)
			planned.noteAt(planned.whole(), "", CodeScopeClash, message)
		default:
			return workbook{}, settleErr
		}
		state = next
		book.add(&planned)
	}
	book.report.settle()
	return book, nil
}

func (s *Service) reads(ctx context.Context, req PreviewRequest) ([]sheetRead, error) {
	if len(req.Sheets) == 0 {
		mapping, table, err := s.read(ctx, req.SiteID, req.Path, req.Mapping.domain())
		if err != nil {
			return nil, err
		}
		return []sheetRead{{mapping: mapping, table: table}}, nil
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
		reads = append(reads, sheetRead{mapping: mapping, table: table})
	}
	return reads, nil
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

func (s siteState) after(p *plan, now time.Time) (siteState, error) {
	entities := slices.Clone(s.entities)
	entityAt := make(map[string]int, len(entities)+len(p.entities))
	for i := range entities {
		entityAt[entities[i].ID] = i
	}
	for i := range p.entities {
		planned := p.entities[i].entity
		if at, held := entityAt[planned.ID]; held {
			entities[at] = planned
			continue
		}
		entityAt[planned.ID] = len(entities)
		entities = append(entities, planned)
	}
	for _, owned := range p.canonical {
		at := entityAt[owned.entityID]
		entities[at].CanonicalPageID = &owned.pageID
		entities[at].UpdatedAt = now
	}

	pages := slices.Clone(s.pages)
	pageAt := make(map[string]int, len(pages)+len(p.pages))
	for i := range pages {
		pageAt[pages[i].ID] = i
	}
	for i := range p.pages {
		planned := p.pages[i].page
		if at, held := pageAt[planned.ID]; held {
			pages[at] = planned
			continue
		}
		pageAt[planned.ID] = len(pages)
		pages = append(pages, planned)
	}

	edges := slices.Concat(s.edges, p.edges)
	moved, err := graph.Settle(entities, edges)
	for i := range moved {
		at := entityAt[moved[i].ID]
		entities[at].ScopeID = moved[i].ScopeID
		entities[at].UpdatedAt = now
	}
	sortStored(entities, edges, pages)
	return newSiteState(s.siteID, entities, edges, pages), err
}
