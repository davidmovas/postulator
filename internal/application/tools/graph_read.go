package tools

import (
	"context"
	"strings"

	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	graphReadName = "graph_read"
	viewEntities  = "entities"
	viewEntity    = "entity"
	viewEdges     = "edges"
	viewAdoption  = "adoption"
)

type graphReadArgs struct {
	View      string   `json:"view" enum:"entities,entity,edges,adoption" description:"entities lists them with their parent or why none; entity reads one whole; edges lists the edges; adoption previews the parents the page tree would give"`
	EntityID  string   `json:"entityId,omitempty" description:"entity: the entity; edges: only those touching it"`
	Query     string   `json:"query,omitempty" description:"entities: words of the name"`
	ParentID  string   `json:"parentId,omitempty" description:"entities: only children of this entity"`
	Roots     bool     `json:"roots,omitempty" description:"entities: only those with no approved parent"`
	Kind      string   `json:"kind,omitempty" enum:"parent,related" description:"edges: only this kind"`
	Status    string   `json:"status,omitempty" enum:"approved,proposed,rejected" description:"edges: only this state"`
	Under     string   `json:"under,omitempty" description:"adoption: only the entities of this page's subtree"`
	EntityIDs []string `json:"entityIds,omitempty" description:"adoption: only these entities"`
	Cursor    string   `json:"cursor,omitempty" description:"nextCursor of the last call"`
	Limit     int      `json:"limit,omitempty" description:"Default 50, at most 200"`
}

type GraphReadResponse struct {
	Entities *paging.List[sitemodel.EntityRow] `json:"entities,omitempty"`
	Entity   *sitemodel.EntityDetail           `json:"entity,omitempty"`
	Edges    *paging.List[graph.Edge]          `json:"edges,omitempty"`
	Adoption *graph.AdoptParentsResponse       `json:"adoption,omitempty"`
}

func graphRead(deps Deps) Tool {
	return newSiteTool(Def{
		Name: graphReadName,
		Description: "Read the entity graph: the entities with their parents, one entity whole with its card, page, " +
			"parents, related and children, the edges, or the parents the page tree would give where the two disagree.",
		Risk: RiskRead,
	}, func(ctx context.Context, b Binding, in graphReadArgs) (GraphReadResponse, error) {
		cursor := paging.Cursor(in.Cursor)
		switch in.View {
		case viewEntities:
			listed, err := fitted(deps, in.Limit, func(limit int) (paging.List[sitemodel.EntityRow], error) {
				return deps.SiteModel.Entities(ctx, sitemodel.EntitiesRequest{
					SiteID: b.SiteID, Query: in.Query, ParentID: in.ParentID, Roots: in.Roots, Cursor: cursor, Limit: limit,
				})
			})
			return GraphReadResponse{Entities: &listed}, err
		case viewEntity:
			detail, err := deps.SiteModel.Entity(ctx, sitemodel.EntityRequest{SiteID: b.SiteID, EntityID: strings.TrimSpace(in.EntityID)})
			return GraphReadResponse{Entity: &detail}, err
		case viewEdges:
			listed, err := fitted(deps, in.Limit, func(limit int) (paging.List[graph.Edge], error) {
				return deps.Graph.ListEdges(ctx, graph.ListEdgesRequest{
					ListRequest: dto.ListRequest{Cursor: in.Cursor, Limit: limit}, SiteID: b.SiteID,
					Kind: in.Kind, Status: in.Status, EntityID: in.EntityID,
				})
			})
			return GraphReadResponse{Edges: &listed}, err
		case viewAdoption:
			preview, err := fitted(deps, in.Limit, func(limit int) (graph.AdoptParentsResponse, error) {
				return deps.Graph.AdoptParents(ctx, graph.AdoptParentsRequest{
					SiteID: b.SiteID, Under: in.Under, EntityIDs: in.EntityIDs, Cursor: cursor, Limit: limit,
				})
			})
			return GraphReadResponse{Adoption: &preview}, err
		default:
			return GraphReadResponse{}, errors.New(errors.Invalid, "graph_read takes the view entities, entity, edges or adoption").
				WithDetail("field", "view")
		}
	})
}
