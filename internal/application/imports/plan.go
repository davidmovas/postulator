package imports

import (
	"encoding/json"
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
	retyped bool
}

type canonical struct {
	entityID string
	pageID   string
}

type plan struct {
	siteID    string
	sheet     string
	file      string
	read      []string
	mapping   importmap.Mapping
	report    PreviewReport
	entities  []plannedEntity
	edges     []graph.Edge
	retired   []string
	pages     []plannedPage
	canonical []canonical
	rows      int
	lasting   []lasting
	agreed    []string
	chosen    []chosenRow
	keys      map[string]map[string]struct{}
}

func newPlan(siteID string, read *sheetRead) plan {
	mapping, table := read.mapping, read.table
	p := plan{siteID: siteID, file: read.file, read: sheetsRead(mapping, table), mapping: mapping, rows: len(table.Rows)}
	if len(p.read) == 1 {
		p.sheet = p.read[0]
	}
	p.keys = make(map[string]map[string]struct{}, len(p.read)+1)
	for _, sheet := range p.read {
		p.sawRow(sheet, "")
	}
	if len(p.read) == 0 {
		p.sawRow(p.file, "")
	}
	p.report.Columns = columnViews(p.sheet, mapping.Uses(table.Headers))
	p.noteDropped(mapping, table.Headers)
	return p
}

func (p *plan) noteDropped(mapping importmap.Mapping, headers []string) {
	read := mapping.Columns[importmap.FieldPageType]
	for _, column := range mapping.Dropped(headers, importmap.FieldPageType) {
		p.noteAt(p.whole(), string(importmap.FieldPageType), CodePageTypeColumnDropped,
			"the column "+column+" also names a page type and was not read; the page type is read from "+read)
	}
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

func (p *plan) rowSheet(at importmap.Origin) string {
	if at.Sheet != "" {
		return at.Sheet
	}
	return p.file
}

func (p *plan) noteAt(at importmap.Origin, field string, code FindingCode, message string) {
	p.noteAbout(at, field, code, nil, message)
}

func (p *plan) noteAbout(at importmap.Origin, field string, code FindingCode, subject json.RawMessage, message string) {
	finding := Finding{Row: at.Row, Sheet: at.Sheet, Field: field, Code: string(code), Message: message, Subject: subject}
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

func planSheet(state siteState, read *sheetRead, now time.Time) (plan, error) {
	p := newPlan(state.siteID, read)
	rows := matchRows(readRows(read.binding, read.table, &p), read.mapping.Options.RowType, &state, &p)
	sheet := pagesOf(rows, &p)
	resolveTypes(sheet, &state)
	fillGaps(sheet, &state, &p)

	if err := newBuilder(state, &p, rows, sheet, now).run(); err != nil {
		return plan{}, err
	}
	p.report.settle()
	return p, nil
}

func (b *builder) run() error {
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
	if err := b.planPages(); err != nil {
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
