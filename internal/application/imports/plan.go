package imports

import (
	"context"
	"maps"
	"slices"
	"time"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

type plannedEntity struct {
	entity  graph.Entity
	created bool
}

type plannedPage struct {
	page    pagemap.Page
	created bool
}

type canonical struct {
	entityID string
	pageID   string
}

type plan struct {
	siteID    string
	sheet     string
	read      []string
	mapping   importmap.Mapping
	report    PreviewReport
	entities  []plannedEntity
	edges     []graph.Edge
	pages     []plannedPage
	canonical []canonical
	rows      int
}

func newPlan(siteID string, mapping importmap.Mapping, table importmap.Table) plan {
	p := plan{siteID: siteID, read: sheetsRead(mapping, table), mapping: mapping, rows: len(table.Rows)}
	if len(p.read) == 1 {
		p.sheet = p.read[0]
	}
	p.report.Columns = columnViews(p.sheet, mapping.Uses(table.Headers))
	return p
}

func sheetsRead(mapping importmap.Mapping, table importmap.Table) []string {
	if len(mapping.Options.Sheets) > 0 {
		return slices.Clone(mapping.Options.Sheets)
	}
	out := make([]string, 0, 1)
	for i := range table.Rows {
		if at := table.Origin(i).Sheet; at != "" && !slices.Contains(out, at) {
			out = append(out, at)
		}
	}
	return out
}

func (p *plan) whole() importmap.Origin {
	return importmap.Origin{Sheet: p.sheet}
}

func (p *plan) sheetAt(at importmap.Origin) string {
	if at.Sheet != "" {
		return at.Sheet
	}
	return p.sheet
}

func (p *plan) noteAt(at importmap.Origin, field string, code FindingCode, message string) {
	finding := Finding{Row: at.Row, Sheet: at.Sheet, Field: field, Code: string(code), Message: message}
	if code.Blocking() {
		p.report.Errors = append(p.report.Errors, finding)
		return
	}
	p.report.Warnings = append(p.report.Warnings, finding)
}

func (p *plan) counts() Counts {
	tally := Counts{Skipped: p.report.Skipped, EdgesCreated: len(p.edges)}
	for i := range p.entities {
		if p.entities[i].created {
			tally.EntitiesCreated++
		} else {
			tally.EntitiesUpdated++
		}
	}
	for i := range p.pages {
		if p.pages[i].created {
			tally.PagesCreated++
		} else {
			tally.PagesUpdated++
		}
	}
	return tally
}

func (s *Service) plan(ctx context.Context, state siteState, read *sheetRead, now time.Time) (plan, error) {
	p := newPlan(state.siteID, read.mapping, read.table)
	rows := readRows(read.binding, read.table, &p)
	sheet := pagesOf(rows, &p)
	state.byPlanned = maps.Clone(state.byPlanned)
	typeRows(sheet, rows, read.mapping.Options.RowType, &state)
	matchProducts(sheet, &state, &p)
	fillGaps(sheet, &state, &p)
	templates, err := s.kindTemplates(ctx, state.siteID, sheet)
	if err != nil {
		return plan{}, err
	}

	if err := newBuilder(state, &p, rows, sheet, now).run(templates); err != nil {
		return plan{}, err
	}
	p.report.settle()
	return p, nil
}

func (b *builder) run(templates *kindTemplates) error {
	b.build()
	if err := b.planEntities(); err != nil {
		return err
	}
	if err := b.planEdges(); err != nil {
		return err
	}
	if err := b.checkAcyclic(); err != nil {
		return err
	}
	if err := b.planPages(templates); err != nil {
		return err
	}
	if err := b.checkCannibalization(); err != nil {
		return err
	}
	b.final = finalPages(b.state, b.p)
	b.linkParents()
	b.markCanonical()
	b.reportGroups()
	return nil
}
