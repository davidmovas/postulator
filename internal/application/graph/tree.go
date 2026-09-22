package graph

import (
	"context"
	"strconv"
	"strings"
	"time"

	graphdomain "github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

const maxEntitiesPerBatch = 200

func (s *Service) CreateEntities(ctx context.Context, req CreateEntitiesRequest) (CreateEntitiesResponse, error) {
	if err := requireSite(req.SiteID); err != nil {
		return CreateEntitiesResponse{}, err
	}
	if _, err := s.sites.Get(ctx, req.SiteID); err != nil {
		return CreateEntitiesResponse{}, err
	}
	if len(req.Entities) == 0 {
		return CreateEntitiesResponse{}, invalidField("a batch needs at least one entity", "entities")
	}
	if len(req.Entities) > maxEntitiesPerBatch {
		return CreateEntitiesResponse{}, invalidField("a batch may carry at most "+
			strconv.Itoa(maxEntitiesPerBatch)+" entities", "entities")
	}

	source := graphdomain.Source(req.Source)
	if req.Source == "" {
		source = graphdomain.SourceAI
	}

	now := s.now()
	built := make([]graphdomain.Entity, 0, len(req.Entities))
	minted := make(map[string]string, len(req.Entities))
	for i := range req.Entities {
		field := "entities[" + strconv.Itoa(i) + "]"
		entity, err := graphdomain.NewEntity(graphdomain.Entity{
			ID:                id.New(),
			SiteID:            req.SiteID,
			Name:              req.Entities[i].Name,
			Kind:              graphdomain.Kind(req.Entities[i].Kind),
			Intent:            req.Entities[i].Intent,
			PrimaryKeyword:    req.Entities[i].PrimaryKeyword,
			SecondaryKeywords: req.Entities[i].SecondaryKeywords,
			Anchors:           anchorsOf(req.Entities[i].Anchors),
			Source:            source,
			CreatedAt:         now,
			UpdatedAt:         now,
		})
		if err != nil {
			return CreateEntitiesResponse{}, errors.New(errors.CodeOf(err), err.Error()).WithDetail("field", field)
		}
		key := folded(entity.Name)
		if _, twice := minted[key]; twice {
			return CreateEntitiesResponse{}, invalidField("two entities in the batch are named "+entity.Name,
				field+".name")
		}
		minted[key] = entity.ID
		built = append(built, entity)
	}

	var edges []graphdomain.Edge
	doErr := s.uow.Do(ctx, func(c context.Context) error {
		known, err := s.namesOnSite(c, req.SiteID, minted)
		if err != nil {
			return err
		}
		if edges, err = s.parentEdges(req, built, known, source, now); err != nil {
			return err
		}

		for i := range built {
			if insertErr := s.entities.Insert(c, built[i]); insertErr != nil {
				return insertErr
			}
		}
		for i := range edges {
			if insertErr := s.edges.Insert(c, edges[i]); insertErr != nil {
				return insertErr
			}
		}
		return s.acyclic(c, req.SiteID)
	})
	if doErr != nil {
		return CreateEntitiesResponse{}, doErr
	}
	if publishErr := s.changed(req.SiteID); publishErr != nil {
		return CreateEntitiesResponse{}, publishErr
	}
	return CreateEntitiesResponse{Entities: entityViews(built), Edges: edgeViews(edges)}, nil
}

func (s *Service) namesOnSite(ctx context.Context, siteID string, minted map[string]string) (map[string]string, error) {
	existing, err := s.entities.ListBySite(ctx, siteID)
	if err != nil {
		return nil, err
	}

	known := make(map[string]string, len(existing)+len(minted))
	for i := range existing {
		known[folded(existing[i].Name)] = existing[i].ID
	}
	for name, entityID := range minted {
		if _, taken := known[name]; taken {
			return nil, invalidField("the site already has an entity named "+name, "entities")
		}
		known[name] = entityID
	}
	return known, nil
}

func (s *Service) parentEdges(req CreateEntitiesRequest, built []graphdomain.Entity, known map[string]string,
	source graphdomain.Source, now time.Time) ([]graphdomain.Edge, error) {
	edges := make([]graphdomain.Edge, 0, len(built))
	for i := range req.Entities {
		parent := strings.TrimSpace(req.Entities[i].ParentName)
		if parent == "" {
			continue
		}

		field := "entities[" + strconv.Itoa(i) + "].parentName"
		parentID, found := known[folded(parent)]
		if !found {
			return nil, invalidField("no entity of this batch or of the site is named "+parent, field)
		}
		if parentID == built[i].ID {
			return nil, invalidField("an entity cannot be its own parent", field)
		}

		edge, err := graphdomain.NewEdge(graphdomain.Edge{
			ID:           id.New(),
			SiteID:       req.SiteID,
			FromEntityID: built[i].ID,
			ToEntityID:   parentID,
			Kind:         graphdomain.EdgeParent,
			Weight:       1,
			Source:       source,
			Status:       graphdomain.StatusApproved,
			Reason:       req.Entities[i].Name + " sits under " + parent,
			CreatedAt:    now,
		})
		if err != nil {
			return nil, errors.New(errors.CodeOf(err), err.Error()).WithDetail("field", field)
		}
		edges = append(edges, edge)
	}
	return edges, nil
}

func (s *Service) acyclic(ctx context.Context, siteID string) error {
	entities, err := s.entities.ListBySite(ctx, siteID)
	if err != nil {
		return err
	}
	edges, err := s.edges.ListBySite(ctx, siteID)
	if err != nil {
		return err
	}

	g, err := graphdomain.New(entities, edges)
	if err != nil {
		return err
	}
	return g.ValidateAcyclic()
}

func folded(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func invalidField(message, field string) error {
	return errors.New(errors.Invalid, message).WithDetail("field", field)
}
