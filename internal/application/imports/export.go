package imports

import (
	"context"
	"path/filepath"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/note"
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

	byID := make(map[string]graph.Entity, len(state.entities))
	for i := range state.entities {
		byID[state.entities[i].ID] = state.entities[i]
	}

	pages := slices.Clone(state.pages)
	slices.SortFunc(pages, func(a, b pagemap.Page) int { return strings.Compare(a.Path, b.Path) })

	options := importmap.DefaultOptions()
	labels := noteLabels(pages, state.entities)
	table := importmap.Table{Headers: append(headers(), labels...), Rows: make([][]string, 0, len(pages)+len(state.entities))}
	mapped := make(map[string]struct{}, len(state.entities))

	for i := range pages {
		page := &pages[i]
		var entity graph.Entity
		if page.EntityID != nil {
			entity = byID[*page.EntityID]
			mapped[entity.ID] = struct{}{}
		}
		cells := row(page, &entity, state.typeName(page.PageTypeID), g, options)
		table.Rows = append(table.Rows, append(cells, noteCells(append(slices.Clone(page.Notes), entity.Notes...), labels)...))
	}

	loose := g.Entities()
	for i := range loose {
		if _, written := mapped[loose[i].ID]; written {
			continue
		}
		cells := row(nil, &loose[i], "", g, options)
		table.Rows = append(table.Rows, append(cells, noteCells(byID[loose[i].ID].Notes, labels)...))
	}

	if err := s.deps.Tables.Write(req.Path, table); err != nil {
		return ExportResponse{}, err
	}
	return ExportResponse{
		Path: req.Path, Format: string(format), Pages: len(pages), Entities: len(state.entities),
	}, nil
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

func exportedFields() []importmap.Field {
	out := make([]importmap.Field, 0, len(importmap.Fields()))
	for _, field := range importmap.Fields() {
		if field != importmap.FieldPrimaryKeyword && field != importmap.FieldOwnEntity && field != importmap.FieldSKU {
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

func row(page *pagemap.Page, entity *graph.Entity, pageType string, g graph.Graph, options importmap.Options) []string {
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
		cells[importmap.FieldPageType] = pageType
		cells[importmap.FieldKeywords] = page.Keywords.Cell()
	}
	if entity.ID != "" {
		cells[importmap.FieldEntity] = entity.Name
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

func noteLabels(pages []pagemap.Page, entities []graph.Entity) []string {
	out := make([]string, 0)
	seen := make(map[string]struct{})
	add := func(notes []note.Note) {
		for _, n := range notes {
			if _, held := seen[strings.ToLower(n.Label)]; held {
				continue
			}
			seen[strings.ToLower(n.Label)] = struct{}{}
			out = append(out, n.Label)
		}
	}
	for i := range pages {
		add(pages[i].Notes)
	}
	for i := range entities {
		add(entities[i].Notes)
	}
	return out
}

func noteCells(notes []note.Note, labels []string) []string {
	out := make([]string, len(labels))
	for at, label := range labels {
		for _, n := range notes {
			if strings.EqualFold(n.Label, label) && out[at] == "" {
				out[at] = n.Text
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
