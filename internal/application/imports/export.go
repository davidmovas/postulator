package imports

import (
	"context"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func (s *Service) Export(ctx context.Context, req ExportRequest) (ExportResponse, error) {
	if err := s.requireSite(ctx, req.SiteID); err != nil {
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

	byID := make(map[string]graph.Entity, len(state.entities))
	for i := range state.entities {
		byID[state.entities[i].ID] = state.entities[i]
	}

	pages := slices.Clone(state.pages)
	slices.SortFunc(pages, func(a, b pagemap.Page) int { return strings.Compare(a.Path, b.Path) })

	options := importmap.DefaultOptions()
	table := importmap.Table{Headers: headers(), Rows: make([][]string, 0, len(pages)+len(state.entities))}
	mapped := make(map[string]struct{}, len(state.entities))

	for i := range pages {
		page := &pages[i]
		var entity graph.Entity
		if page.EntityID != nil {
			entity = byID[*page.EntityID]
			mapped[entity.ID] = struct{}{}
		}
		table.Rows = append(table.Rows, row(page, &entity, kinds[refOf(page.TemplateID)], g, options))
	}

	loose := g.Entities()
	for i := range loose {
		if _, written := mapped[loose[i].ID]; written {
			continue
		}
		table.Rows = append(table.Rows, row(nil, &loose[i], "", g, options))
	}

	if err := s.deps.Tables.Write(req.Path, table); err != nil {
		return ExportResponse{}, err
	}
	return ExportResponse{Path: req.Path, Pages: len(pages), Entities: len(state.entities)}, nil
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

func headers() []string {
	fields := importmap.Fields()
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
		cells[importmap.FieldTitle] = page.Title
		cells[importmap.FieldH1] = page.H1
		cells[importmap.FieldMetaTitle] = page.MetaTitle
		cells[importmap.FieldMetaDescription] = page.MetaDescription
		cells[importmap.FieldWPType] = string(page.WPType)
		cells[importmap.FieldPageKind] = pageKind
	}
	if entity.ID != "" {
		cells[importmap.FieldEntity] = entity.Name
		cells[importmap.FieldEntityKind] = string(entity.Kind)
		cells[importmap.FieldPrimaryKeyword] = entity.PrimaryKeyword
		cells[importmap.FieldKeywords] = options.Join(importmap.FieldKeywords, entity.SecondaryKeywords)
		cells[importmap.FieldAnchors] = options.Join(importmap.FieldAnchors, anchorTexts(entity.Anchors))
		if parents := g.Parents(entity.ID, 1); len(parents) > 0 {
			cells[importmap.FieldParentEntity] = parents[0].Name
		}
		cells[importmap.FieldRelated] = options.Join(importmap.FieldRelated, relatedNames(g, entity.ID))
	}

	fields := importmap.Fields()
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		out = append(out, cells[field])
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
