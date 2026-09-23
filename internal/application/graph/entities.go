package graph

import (
	"context"

	"github.com/davidmovas/postulator/internal/application"
	graphdomain "github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func entitySort(sort *dto.Sort) (key graphdomain.EntitySort, desc bool, err error) {
	if sort == nil {
		return graphdomain.EntitySortCreatedAt, false, nil
	}
	key = graphdomain.EntitySort(sort.Field)
	if !key.Valid() {
		return "", false, errors.New(errors.Invalid, "entities cannot be sorted by this field").WithDetail("field", "sort.field")
	}
	return key, sort.Desc, nil
}

func (s *Service) CreateEntity(ctx context.Context, req CreateEntityRequest) (CreateEntityResponse, error) {
	if err := requireSite(req.SiteID); err != nil {
		return CreateEntityResponse{}, err
	}
	if _, err := s.sites.Get(ctx, req.SiteID); err != nil {
		return CreateEntityResponse{}, err
	}

	source := graphdomain.Source(req.Source)
	if req.Source == "" {
		source = graphdomain.SourceUser
	}
	now := s.now()
	entity, err := graphdomain.NewEntity(graphdomain.Entity{
		ID:                id.New(),
		SiteID:            req.SiteID,
		Name:              req.Name,
		Kind:              graphdomain.Kind(req.Kind),
		Intent:            req.Intent,
		PrimaryKeyword:    req.PrimaryKeyword,
		SecondaryKeywords: req.SecondaryKeywords,
		Anchors:           anchorsOf(ctx, req.Anchors),
		Source:            source,
		CreatedAt:         now,
		UpdatedAt:         now,
	})
	if err != nil {
		return CreateEntityResponse{}, err
	}

	if doErr := s.uow.Do(ctx, func(c context.Context) error { return s.entities.Insert(c, entity) }); doErr != nil {
		return CreateEntityResponse{}, doErr
	}
	if publishErr := s.changed(entity.SiteID); publishErr != nil {
		return CreateEntityResponse{}, publishErr
	}
	return CreateEntityResponse{Entity: entityView(entity)}, nil
}

func (s *Service) UpdateEntity(ctx context.Context, req UpdateEntityRequest) (UpdateEntityResponse, error) {
	updated, err := s.rewriteEntity(ctx, req.ID, func(next *graphdomain.Entity) {
		if req.Name != nil {
			next.Name = *req.Name
		}
		if req.Kind != nil {
			next.Kind = graphdomain.Kind(*req.Kind)
		}
		if req.Intent != nil {
			next.Intent = *req.Intent
		}
		if req.PrimaryKeyword != nil {
			next.PrimaryKeyword = *req.PrimaryKeyword
		}
		if req.SecondaryKeywords != nil {
			next.SecondaryKeywords = *req.SecondaryKeywords
		}
	})
	if err != nil {
		return UpdateEntityResponse{}, err
	}
	return UpdateEntityResponse{Entity: entityView(updated)}, nil
}

func (s *Service) SetAnchors(ctx context.Context, req SetAnchorsRequest) (SetAnchorsResponse, error) {
	updated, err := s.rewriteEntity(ctx, req.EntityID, func(next *graphdomain.Entity) {
		next.Anchors = anchorsOf(ctx, req.Anchors)
	})
	if err != nil {
		return SetAnchorsResponse{}, err
	}
	return SetAnchorsResponse{Entity: entityView(updated)}, nil
}

func (s *Service) rewriteEntity(ctx context.Context, entityID string, change func(*graphdomain.Entity)) (graphdomain.Entity, error) {
	var updated graphdomain.Entity
	err := s.uow.Do(ctx, func(c context.Context) error {
		current, getErr := s.entities.Get(c, entityID)
		if getErr != nil {
			return getErr
		}
		next := current
		change(&next)
		next.UpdatedAt = s.now()
		valid, validErr := graphdomain.NewEntity(next)
		if validErr != nil {
			return validErr
		}
		if updateErr := s.entities.Update(c, valid); updateErr != nil {
			return updateErr
		}
		updated = valid
		return nil
	})
	if err != nil {
		return graphdomain.Entity{}, err
	}
	if publishErr := s.changed(updated.SiteID); publishErr != nil {
		return graphdomain.Entity{}, publishErr
	}
	return updated, nil
}

func (s *Service) DeleteEntity(ctx context.Context, req DeleteEntityRequest) (DeleteEntityResponse, error) {
	var siteID string
	err := s.uow.Do(ctx, func(c context.Context) error {
		current, getErr := s.entities.Get(c, req.ID)
		if getErr != nil {
			return getErr
		}
		siteID = current.SiteID
		return s.entities.Delete(c, req.ID)
	})
	if err != nil {
		return DeleteEntityResponse{}, err
	}
	if publishErr := s.changed(siteID); publishErr != nil {
		return DeleteEntityResponse{}, publishErr
	}
	if publishErr := s.pagesChanged(siteID); publishErr != nil {
		return DeleteEntityResponse{}, publishErr
	}
	return DeleteEntityResponse{}, nil
}

func (s *Service) GetEntity(ctx context.Context, req GetEntityRequest) (GetEntityResponse, error) {
	entity, err := s.entities.Get(ctx, req.ID)
	if err != nil {
		return GetEntityResponse{}, err
	}
	return GetEntityResponse{Entity: entityView(entity)}, nil
}

func (s *Service) ListEntities(ctx context.Context, req ListEntitiesRequest) (paging.List[Entity], error) {
	if err := requireSite(req.SiteID); err != nil {
		return paging.List[Entity]{}, err
	}
	q := graphdomain.EntityQuery{SiteID: req.SiteID, HasCanonicalPage: req.HasCanonicalPage, NamePrefix: req.NamePrefix}
	if req.Kind != "" {
		kind := graphdomain.Kind(req.Kind)
		if !kind.Valid() {
			return paging.List[Entity]{}, errors.New(errors.Invalid, "entity kind is not recognized").WithDetail("field", "kind")
		}
		q.Kind = &kind
	}
	key, desc, err := entitySort(req.Sort)
	if err != nil {
		return paging.List[Entity]{}, err
	}
	q.Sort, q.Desc = key, desc

	list, err := s.entities.List(ctx, q, application.PageRequest(req.ListRequest))
	if err != nil {
		return paging.List[Entity]{}, err
	}
	return application.MapList(list, entityView), nil
}
