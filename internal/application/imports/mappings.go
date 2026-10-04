package imports

import (
	"context"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func (s *Service) save(ctx context.Context, mapping importmap.Mapping) (importmap.Mapping, error) {
	now := s.now()
	mapping.CreatedAt, mapping.UpdatedAt = now, now

	if mapping.ID == "" {
		previous, err := s.named(ctx, mapping.SiteID, mapping.Name)
		if err != nil {
			return importmap.Mapping{}, err
		}
		mapping.ID = previous.ID
		if previous.ID != "" {
			mapping.CreatedAt = previous.CreatedAt
		}
	}
	if mapping.ID == "" {
		mapping.ID = id.New()
	}

	ready, err := importmap.NewMapping(mapping)
	if err != nil {
		return importmap.Mapping{}, err
	}
	if err := s.deps.Mappings.Upsert(ctx, ready); err != nil {
		return importmap.Mapping{}, err
	}
	return ready, nil
}

func (s *Service) named(ctx context.Context, siteID, name string) (importmap.Mapping, error) {
	if siteID == "" || name == "" {
		return importmap.Mapping{}, nil
	}
	saved, err := s.deps.Mappings.ListBySite(ctx, siteID)
	if err != nil {
		return importmap.Mapping{}, err
	}
	for i := range saved {
		if strings.EqualFold(saved[i].Name, name) {
			return saved[i], nil
		}
	}
	return importmap.Mapping{}, nil
}

func (s *Service) SaveMapping(ctx context.Context, req SaveMappingRequest) (SaveMappingResponse, error) {
	if err := s.requireSite(ctx, req.Mapping.SiteID); err != nil {
		return SaveMappingResponse{}, err
	}
	saved, err := s.save(ctx, req.Mapping.domain())
	if err != nil {
		return SaveMappingResponse{}, err
	}
	return SaveMappingResponse{Mapping: mappingView(saved)}, nil
}

func (s *Service) ListMappings(ctx context.Context, req ListMappingsRequest) (ListMappingsResponse, error) {
	if err := s.requireSite(ctx, req.SiteID); err != nil {
		return ListMappingsResponse{}, err
	}
	saved, err := s.deps.Mappings.ListBySite(ctx, req.SiteID)
	if err != nil {
		return ListMappingsResponse{}, err
	}
	return ListMappingsResponse{Mappings: mappingViews(saved)}, nil
}

func (s *Service) DeleteMapping(ctx context.Context, req DeleteMappingRequest) (DeleteMappingResponse, error) {
	if req.ID == "" {
		return DeleteMappingResponse{}, errors.New(errors.Invalid, "mapping id must not be empty").WithDetail("field", "id")
	}
	if err := s.deps.Mappings.Delete(ctx, req.ID); err != nil {
		return DeleteMappingResponse{}, err
	}
	return DeleteMappingResponse{}, nil
}
