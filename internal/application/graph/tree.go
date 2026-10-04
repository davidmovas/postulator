package graph

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/application"
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
	for i := range req.Entities {
		field := "entities[" + strconv.Itoa(i) + "]"
		keywords, keywordsErr := application.KeywordList(req.Entities[i].Keywords, field+".keywords")
		if keywordsErr != nil {
			return CreateEntitiesResponse{}, keywordsErr
		}
		entity, err := graphdomain.NewEntity(graphdomain.Entity{
			ID:        id.New(),
			SiteID:    req.SiteID,
			Name:      req.Entities[i].Name,
			Kind:      graphdomain.Kind(req.Entities[i].Kind),
			Intent:    req.Entities[i].Intent,
			Keywords:  keywords,
			Anchors:   anchorsOf(ctx, req.Entities[i].Anchors),
			Source:    source,
			CreatedAt: now,
			UpdatedAt: now,
		})
		if err != nil {
			return CreateEntitiesResponse{}, errors.New(errors.CodeOf(err), err.Error()).WithDetail("field", field)
		}
		built = append(built, entity)
	}

	var edges []graphdomain.Edge
	doErr := s.uow.Do(ctx, func(c context.Context) error {
		existing, err := s.entities.ListBySite(c, req.SiteID)
		if err != nil {
			return err
		}
		if edges, err = s.parentEdges(req, built, existing, source, now); err != nil {
			return err
		}
		if clashErr := batchClashes(existing, built); clashErr != nil {
			return clashErr
		}

		ordered, err := parentsFirst(built)
		if err != nil {
			return err
		}
		for i := range ordered {
			if insertErr := s.entities.Insert(c, ordered[i]); insertErr != nil {
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
	views, err := s.viewsOf(ctx, req.SiteID, built)
	if err != nil {
		return CreateEntitiesResponse{}, err
	}
	return CreateEntitiesResponse{Entities: views, Edges: edgeViews(edges)}, nil
}

func namedIn(existing, built []graphdomain.Entity) map[string][]string {
	known := make(map[string][]string, len(existing)+len(built))
	for i := range existing {
		known[graphdomain.Key(existing[i].Name)] = append(known[graphdomain.Key(existing[i].Name)], existing[i].ID)
	}
	for i := range built {
		known[graphdomain.Key(built[i].Name)] = append(known[graphdomain.Key(built[i].Name)], built[i].ID)
	}
	return known
}

func batchClashes(existing, built []graphdomain.Entity) error {
	minted := make(map[string]int, len(built))
	for i := range built {
		minted[built[i].ID] = i
	}
	for _, clash := range graphdomain.ScopeClashes(append(slices.Clone(existing), built...)) {
		for _, entityID := range clash.EntityIDs {
			if at, ours := minted[entityID]; ours {
				return invalidField("the batch puts a second entity named "+clash.Name+" under the same parent",
					"entities["+strconv.Itoa(at)+"].name")
			}
		}
	}
	return nil
}

func parentsFirst(built []graphdomain.Entity) ([]graphdomain.Entity, error) {
	pending := make(map[string]struct{}, len(built))
	for i := range built {
		pending[built[i].ID] = struct{}{}
	}
	ordered := make([]graphdomain.Entity, 0, len(built))
	for len(ordered) < len(built) {
		progressed := false
		for i := range built {
			if _, waiting := pending[built[i].ID]; !waiting {
				continue
			}
			if built[i].ScopeID != nil {
				if _, parentWaits := pending[*built[i].ScopeID]; parentWaits {
					continue
				}
			}
			delete(pending, built[i].ID)
			ordered = append(ordered, built[i])
			progressed = true
		}
		if !progressed {
			return nil, invalidField("the parents named in the batch form a cycle", "entities")
		}
	}
	return ordered, nil
}

func (s *Service) parentEdges(req CreateEntitiesRequest, built, existing []graphdomain.Entity,
	source graphdomain.Source, now time.Time) ([]graphdomain.Edge, error) {
	known := namedIn(existing, built)
	edges := make([]graphdomain.Edge, 0, len(built))
	for i := range req.Entities {
		parent := strings.TrimSpace(req.Entities[i].ParentName)
		if parent == "" {
			continue
		}

		field := "entities[" + strconv.Itoa(i) + "].parentName"
		candidates := slices.DeleteFunc(slices.Clone(known[graphdomain.Key(parent)]), func(id string) bool { return id == built[i].ID })
		switch len(candidates) {
		case 0:
			if slices.Contains(known[graphdomain.Key(parent)], built[i].ID) {
				return nil, invalidField("an entity cannot be its own parent", field)
			}
			return nil, invalidField("no entity of this batch or of the site is named "+parent, field)
		case 1:
		default:
			return nil, invalidField("more than one entity is named "+parent+
				", so it cannot be told which one is meant; create this entity under it by id", field)
		}
		parentID := candidates[0]
		built[i].ScopeID = &parentID

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

func invalidField(message, field string) error {
	return errors.New(errors.Invalid, message).WithDetail("field", field)
}
