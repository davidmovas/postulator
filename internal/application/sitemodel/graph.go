package sitemodel

import (
	"context"
	"strings"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	NoParentEdge = "no_parent_edge"

	entitiesDefaultLimit = 50
	entitiesMaxLimit     = 200
)

type EntitiesRequest struct {
	SiteID   string        `json:"siteId"`
	Query    string        `json:"query,omitempty"`
	ParentID string        `json:"parentId,omitempty"`
	Cursor   paging.Cursor `json:"cursor,omitempty"`
	Limit    int           `json:"limit,omitempty"`
	Roots    bool          `json:"roots,omitempty"`
}

type PageRef struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

type EntityLink struct {
	EdgeID   string  `json:"edgeId"`
	EntityID string  `json:"entityId"`
	Name     string  `json:"name"`
	Status   string  `json:"status"`
	Weight   float64 `json:"weight,omitempty"`
}

type EntityRow struct {
	Parent        *EntityLink `json:"parent"`
	CanonicalPage *PageRef    `json:"canonicalPage"`
	ID            string      `json:"id"`
	Name          string      `json:"name"`
	Label         string      `json:"label"`
	ParentWhy     string      `json:"parentWhy,omitempty"`
	Children      int         `json:"children"`
	Related       int         `json:"related"`
}

type EntityRequest struct {
	SiteID   string `json:"siteId"`
	EntityID string `json:"entityId"`
}

type CanonicalPage struct {
	ID           string               `json:"id"`
	Path         string               `json:"path"`
	ParentPageID string               `json:"parentPageId"`
	ParentPath   string               `json:"parentPath"`
	ParentReason pagemap.ParentReason `json:"parentReason"`
}

type EntityFact struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type EntityDetail struct {
	Tree          *contextpack.TreeParent `json:"tree"`
	CanonicalPage *CanonicalPage          `json:"canonicalPage"`
	ID            string                  `json:"id"`
	Name          string                  `json:"name"`
	Label         string                  `json:"label"`
	Intent        string                  `json:"intent"`
	Definition    string                  `json:"definition"`
	ParentWhy     string                  `json:"parentWhy,omitempty"`
	Facts         []EntityFact            `json:"facts"`
	Notes         []EntityFact            `json:"notes"`
	Keywords      []dto.Keyword           `json:"keywords"`
	Anchors       []string                `json:"anchors"`
	Parents       []EntityLink            `json:"parents"`
	Related       []EntityLink            `json:"related"`
	Children      int                     `json:"children"`
}

func (s *Service) Entities(ctx context.Context, req EntitiesRequest) (paging.List[EntityRow], error) {
	if req.SiteID == "" {
		return paging.List[EntityRow]{}, errors.New(errors.Invalid, "name the site whose entities to read").WithDetail("field", "siteId")
	}
	var listed paging.List[EntityRow]
	err := s.deps.Reader.Read(ctx, func(c context.Context) error {
		snapshot, err := s.base(c, req.SiteID)
		if err != nil {
			return err
		}
		query := graph.Key(strings.TrimSpace(req.Query))
		entities := snapshot.Graph.Entities()
		rows := make([]EntityRow, 0, len(entities))
		for i := range entities {
			row := entityRow(snapshot, entities[i])
			switch {
			case query != "" && !strings.Contains(graph.Key(entities[i].Name), query):
				continue
			case req.Roots && row.Parent != nil && row.Parent.Status == string(graph.StatusApproved):
				continue
			case req.ParentID != "" && (row.Parent == nil || row.Parent.EntityID != req.ParentID):
				continue
			}
			rows = append(rows, row)
		}
		keyset := paging.Keyset[EntityRow]{
			IDColumn: "id",
			ID:       func(row EntityRow) string { return row.ID },
			Keys: []paging.SortKey[EntityRow]{
				paging.TextKey[EntityRow]("name", "name", func(row EntityRow) any { return graph.Key(row.Name) }),
			},
		}
		listed, err = keyset.Page(rows, paging.Request{After: req.Cursor, Limit: clamped(req.Limit, entitiesDefaultLimit, entitiesMaxLimit)})
		return err
	})
	return listed, err
}

func (s *Service) Entity(ctx context.Context, req EntityRequest) (EntityDetail, error) {
	if req.SiteID == "" || req.EntityID == "" {
		return EntityDetail{}, errors.New(errors.Invalid, "name the site and the entity to read").WithDetail("field", "entityId")
	}
	var detail EntityDetail
	err := s.deps.Reader.Read(ctx, func(c context.Context) error {
		snapshot, err := s.base(c, req.SiteID)
		if err != nil {
			return err
		}
		entity, found := snapshot.Graph.Entity(req.EntityID)
		if !found {
			return errors.New(errors.NotFound, "the site holds no such entity").WithDetail("entityId", req.EntityID)
		}
		detail = entityDetail(snapshot, entity)
		return nil
	})
	return detail, err
}

func entityRow(snapshot contextpack.Snapshot, entity graph.Entity) EntityRow {
	row := EntityRow{
		ID: entity.ID, Name: entity.Name, Label: snapshot.Graph.Label(entity.ID),
		Children: len(snapshot.Graph.Children(entity.ID)), Related: len(snapshot.Graph.Related(entity.ID, 0)),
	}
	parents := links(snapshot.Graph, entity.ID, graph.EdgeParent)
	for i := range parents {
		if row.Parent == nil || parents[i].Status == string(graph.StatusApproved) && row.Parent.Status != string(graph.StatusApproved) {
			row.Parent = &parents[i]
		}
	}
	if row.Parent == nil {
		row.ParentWhy = NoParentEdge
	}
	if page, found := canonicalOf(snapshot, entity); found {
		row.CanonicalPage = &PageRef{ID: page.ID, Path: page.Path}
	}
	return row
}

func entityDetail(snapshot contextpack.Snapshot, entity graph.Entity) EntityDetail {
	detail := EntityDetail{
		ID: entity.ID, Name: entity.Name, Label: snapshot.Graph.Label(entity.ID), Intent: entity.Intent, Definition: entity.Definition,
		Facts: make([]EntityFact, 0, len(entity.Facts)), Notes: make([]EntityFact, 0, len(entity.Notes)),
		Keywords: application.KeywordViews(entity.Keywords), Anchors: make([]string, 0, len(entity.Anchors)),
		Parents: links(snapshot.Graph, entity.ID, graph.EdgeParent), Related: links(snapshot.Graph, entity.ID, graph.EdgeRelated),
		Children: len(snapshot.Graph.Children(entity.ID)),
	}
	for _, fact := range entity.Facts {
		detail.Facts = append(detail.Facts, EntityFact(fact))
	}
	for _, held := range entity.Notes {
		detail.Notes = append(detail.Notes, EntityFact{Label: held.Label, Value: held.Text})
	}
	for _, anchor := range entity.Anchors {
		detail.Anchors = append(detail.Anchors, anchor.Text)
	}
	if len(detail.Parents) == 0 {
		detail.ParentWhy = NoParentEdge
	}
	page, found := canonicalOf(snapshot, entity)
	if !found {
		return detail
	}
	detail.CanonicalPage = &CanonicalPage{ID: page.ID, Path: page.Path, ParentReason: page.ParentReason}
	if page.ParentPageID != nil {
		detail.CanonicalPage.ParentPageID = *page.ParentPageID
		if parent, known := snapshot.Index.ByID(*page.ParentPageID); known {
			detail.CanonicalPage.ParentPath = parent.Path
		}
	}
	if tree, onMap := snapshot.Map(nil).TreeParentOf(page.ID); onMap {
		detail.Tree = &tree
	}
	return detail
}

func canonicalOf(snapshot contextpack.Snapshot, entity graph.Entity) (pagemap.Page, bool) {
	if entity.CanonicalPageID == nil {
		return pagemap.Page{}, false
	}
	return snapshot.Index.ByID(*entity.CanonicalPageID)
}

func links(g graph.Graph, entityID string, kind graph.EdgeKind) []EntityLink {
	edges := g.Edges()
	out := make([]EntityLink, 0, len(edges))
	for i := range edges {
		held := &edges[i]
		if held.Kind != kind {
			continue
		}
		other := ""
		switch {
		case held.FromEntityID == entityID:
			other = held.ToEntityID
		case kind == graph.EdgeRelated && held.ToEntityID == entityID:
			other = held.FromEntityID
		default:
			continue
		}
		named, _ := g.Entity(other)
		out = append(out, EntityLink{
			EdgeID: held.ID, EntityID: other, Name: named.Name, Status: string(held.Status), Weight: held.Weight,
		})
	}
	return out
}
