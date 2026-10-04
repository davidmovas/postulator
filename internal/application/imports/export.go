package imports

import (
	"context"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func (s *Service) Export(ctx context.Context, req ExportRequest) (ExportResponse, error) {
	if err := s.requireSite(ctx, req.SiteID); err != nil {
		return ExportResponse{}, err
	}

	format, err := exportFormat(req)
	if err != nil {
		return ExportResponse{}, err
	}

	state, err := s.state(ctx, req.SiteID)
	if err != nil {
		return ExportResponse{}, err
	}
	g, err := graph.New(state.entities, state.edges)
	if err != nil {
		return ExportResponse{}, err
	}
	kinds, err := s.pageKinds(ctx, state.pages)
	if err != nil {
		return ExportResponse{}, err
	}

	table, warnings := exportTable(&state, g, kinds)
	if err := s.deps.Tables.Write(req.Path, table); err != nil {
		return ExportResponse{}, err
	}
	return ExportResponse{
		Path: req.Path, Format: string(format), Pages: len(state.pages), Entities: len(state.entities), Warnings: warnings,
	}, nil
}

func exportTable(state *siteState, g graph.Graph, kinds map[string]string) (importmap.Table, []Finding) {
	byID := make(map[string]graph.Entity, len(state.entities))
	for i := range state.entities {
		byID[state.entities[i].ID] = state.entities[i]
	}

	pages := slices.Clone(state.pages)
	slices.SortFunc(pages, func(a, b pagemap.Page) int { return strings.Compare(a.Path, b.Path) })

	held := newShelf(state.categories)
	depth := categoryDepth(pages, held)
	options := importmap.DefaultOptions()
	labels := noteLabels(pages)
	table := importmap.Table{
		Headers: slices.Concat(headers(), importmap.CategoryHeaders()[:depth], labels),
		Rows:    make([][]string, 0, len(pages)+len(state.entities)),
	}
	warnings := make([]Finding, 0)
	mapped := make(map[string]struct{}, len(state.entities))

	for i := range pages {
		page := &pages[i]
		var entity graph.Entity
		if page.EntityID != nil {
			entity = byID[*page.EntityID]
			mapped[entity.ID] = struct{}{}
		}
		chain := held.trail(page.CategoryID)
		if len(chain) > depth {
			warnings = append(warnings, chainCut(len(table.Rows)+2, page, chain, depth))
		}
		cells := row(page, &entity, kinds[refOf(page.TemplateID)], g, options)
		table.Rows = append(table.Rows, slices.Concat(cells, levelCells(chain, depth), noteCells(page.Notes, labels)))
	}

	loose := g.Entities()
	for i := range loose {
		if _, written := mapped[loose[i].ID]; written {
			continue
		}
		cells := row(nil, &loose[i], "", g, options)
		table.Rows = append(table.Rows, slices.Concat(cells, levelCells(nil, depth), noteCells(nil, labels)))
	}
	return table, warnings
}

func categoryDepth(pages []pagemap.Page, held *shelf) int {
	depth := 0
	for i := range pages {
		depth = max(depth, len(held.trail(pages[i].CategoryID)))
	}
	return min(depth, len(importmap.CategoryHeaders()))
}

func levelCells(chain []string, depth int) []string {
	out := make([]string, depth)
	copy(out, chain)
	return out
}

func chainCut(row int, page *pagemap.Page, chain []string, depth int) Finding {
	return Finding{
		Row: row, Code: string(CodeCategoryChainCut),
		Message: "the page " + page.Path + " is filed under " + strings.Join(chain, " › ") + "; the file keeps its first " +
			strconv.Itoa(depth) + " levels, as many as an import reads back",
	}
}

func exportFormat(req ExportRequest) (importmap.Format, error) {
	format := importmap.Format(strings.TrimSpace(strings.ToLower(req.Format)))
	if format == "" {
		format = importmap.FormatXLSX
	}
	if !format.Valid() {
		return "", errors.New(errors.Invalid, "an export is written as a xlsx or a csv").
			WithDetail("field", "format").WithDetail("format", req.Format)
	}

	if extension := strings.ToLower(filepath.Ext(strings.TrimSpace(req.Path))); extension != format.Extension() {
		return "", errors.New(errors.Invalid, "the file name must end in "+format.Extension()).
			WithDetail("field", "path").WithDetail("format", string(format))
	}
	return format, nil
}

func (s *Service) pageKinds(ctx context.Context, pages []pagemap.Page) (map[string]string, error) {
	kinds := make(map[string]string)
	for i := range pages {
		templateID := refOf(pages[i].TemplateID)
		if templateID == "" {
			continue
		}
		if _, known := kinds[templateID]; known {
			continue
		}
		found, err := s.deps.Templates.Get(ctx, templateID)
		if err != nil {
			return nil, err
		}
		kinds[templateID] = found.PageKind
	}
	return kinds, nil
}

func refOf(ref *string) string {
	if ref == nil {
		return ""
	}
	return *ref
}

func exportedFields() []importmap.Field {
	out := make([]importmap.Field, 0, len(importmap.Fields()))
	for _, field := range importmap.Fields() {
		if field != importmap.FieldPrimaryKeyword && field != importmap.FieldOwnEntity {
			out = append(out, field)
		}
	}
	return out
}

func headers() []string {
	fields := exportedFields()
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		out = append(out, string(field))
	}
	return out
}

func row(page *pagemap.Page, entity *graph.Entity, pageKind string, g graph.Graph, options importmap.Options) []string {
	cells := make(map[importmap.Field]string, len(importmap.Fields()))
	if page != nil {
		cells[importmap.FieldPath] = page.Path
		if page.PlannedPath != "" {
			cells[importmap.FieldPath] = page.PlannedPath
		}
		cells[importmap.FieldTitle] = page.Title
		cells[importmap.FieldH1] = page.H1
		cells[importmap.FieldMetaTitle] = page.MetaTitle
		cells[importmap.FieldMetaDescription] = page.MetaDescription
		cells[importmap.FieldWPType] = string(page.WPType)
		cells[importmap.FieldPageKind] = pageKind
		cells[importmap.FieldKeywords] = page.Keywords.Cell()
	}
	if entity.ID != "" {
		cells[importmap.FieldEntity] = entity.Name
		cells[importmap.FieldEntityKind] = string(entity.Kind)
		cells[importmap.FieldKeywords] = fill(cells[importmap.FieldKeywords], entity.Keywords.Cell())
		cells[importmap.FieldAnchors] = options.Join(importmap.FieldAnchors, anchorTexts(entity.Anchors))
		if parents := g.Parents(entity.ID, 1); len(parents) > 0 {
			cells[importmap.FieldParentEntity] = parents[0].Name
		}
		cells[importmap.FieldRelated] = options.Join(importmap.FieldRelated, relatedNames(g, entity.ID))
	}

	fields := exportedFields()
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		out = append(out, cells[field])
	}
	return out
}

func noteLabels(pages []pagemap.Page) []string {
	out := make([]string, 0)
	seen := make(map[string]struct{})
	for i := range pages {
		for _, note := range pages[i].Notes {
			if _, held := seen[strings.ToLower(note.Label)]; held {
				continue
			}
			seen[strings.ToLower(note.Label)] = struct{}{}
			out = append(out, note.Label)
		}
	}
	return out
}

func noteCells(notes []pagemap.Note, labels []string) []string {
	out := make([]string, len(labels))
	for at, label := range labels {
		for _, note := range notes {
			if strings.EqualFold(note.Label, label) {
				out[at] = note.Text
			}
		}
	}
	return out
}

func relatedNames(g graph.Graph, entityID string) []string {
	neighbors := g.Related(entityID, 0)
	out := make([]string, 0, len(neighbors))
	for i := range neighbors {
		out = append(out, neighbors[i].Entity.Name)
	}
	return out
}
