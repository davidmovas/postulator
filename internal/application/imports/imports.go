package imports

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

const sampleRows = 5

func (s *Service) Inspect(ctx context.Context, req InspectRequest) (InspectResponse, error) {
	if err := s.requireSite(ctx, req.SiteID); err != nil {
		return InspectResponse{}, err
	}

	table, err := s.table(ctx, req.Path)
	if err != nil {
		return InspectResponse{}, err
	}

	saved, err := s.deps.Mappings.ListBySite(ctx, req.SiteID)
	if err != nil {
		return InspectResponse{}, err
	}

	detected := importmap.AutoDetect(table.Headers)
	detected.SiteID = req.SiteID

	sample := table.Rows
	if len(sample) > sampleRows {
		sample = sample[:sampleRows]
	}
	return InspectResponse{
		Headers:  table.Headers,
		Sample:   sample,
		Rows:     len(table.Rows),
		Detected: mappingView(detected),
		Saved:    mappingViews(saved),
	}, nil
}

func (s *Service) Preview(ctx context.Context, req PreviewRequest) (PreviewResponse, error) {
	computed, err := s.compute(ctx, req.SiteID, req.Path, req.Mapping)
	if err != nil {
		return PreviewResponse{}, err
	}
	return PreviewResponse{Report: computed.report}, nil
}

func (s *Service) compute(ctx context.Context, siteID, path string, mapping Mapping) (plan, error) {
	if err := s.requireSite(ctx, siteID); err != nil {
		return plan{}, err
	}

	table, err := s.table(ctx, path)
	if err != nil {
		return plan{}, err
	}
	return s.plan(ctx, siteID, table, mapping.domain())
}

func (s *Service) Apply(ctx context.Context, req ApplyRequest) (ApplyResponse, error) {
	computed, err := s.compute(ctx, req.SiteID, req.Path, req.Mapping)
	if err != nil {
		return ApplyResponse{}, err
	}
	if computed.broken() {
		return ApplyResponse{}, errors.New(errors.Invalid, "the import cannot be applied while the preview reports errors").
			WithDetail("errors", computed.report.Errors)
	}

	counts := Counts{Skipped: computed.report.Skipped}
	err = s.deps.UnitOfWork.Do(ctx, func(c context.Context) error {
		written, writeErr := s.write(c, &computed)
		if writeErr != nil {
			return writeErr
		}
		counts = written
		counts.Skipped = computed.report.Skipped
		return s.remember(c, req)
	})
	if err != nil {
		return ApplyResponse{}, err
	}
	if err := s.announce(req.SiteID); err != nil {
		return ApplyResponse{}, err
	}
	return ApplyResponse{Report: computed.report, Counts: counts}, nil
}

func (s *Service) write(ctx context.Context, computed *plan) (Counts, error) {
	var counts Counts
	for i := range computed.entities {
		planned := &computed.entities[i]
		if planned.created {
			if err := s.deps.Entities.Insert(ctx, planned.entity); err != nil {
				return Counts{}, err
			}
			counts.EntitiesCreated++
			continue
		}
		if err := s.deps.Entities.Update(ctx, planned.entity); err != nil {
			return Counts{}, err
		}
		counts.EntitiesUpdated++
	}

	for i := range computed.edges {
		if err := s.deps.Edges.Insert(ctx, computed.edges[i]); err != nil {
			return Counts{}, err
		}
		counts.EdgesCreated++
	}

	for i := range computed.pages {
		planned := &computed.pages[i]
		if planned.created {
			if err := s.deps.Pages.Insert(ctx, planned.page); err != nil {
				return Counts{}, err
			}
			counts.PagesCreated++
			continue
		}
		if err := s.deps.Pages.Update(ctx, planned.page); err != nil {
			return Counts{}, err
		}
		counts.PagesUpdated++
	}

	now := s.now()
	for i := range computed.canonical {
		owned := computed.canonical[i]
		if err := s.deps.Entities.SetCanonicalPage(ctx, owned.entityID, &owned.pageID, now); err != nil {
			return Counts{}, err
		}
	}
	return counts, nil
}

func (s *Service) remember(ctx context.Context, req ApplyRequest) error {
	if req.Options.SaveMappingAs == "" {
		return nil
	}
	saved := req.Mapping
	saved.Name = req.Options.SaveMappingAs
	saved.SiteID = req.SiteID
	_, err := s.save(ctx, saved)
	return err
}

func (s *Service) save(ctx context.Context, view Mapping) (importmap.Mapping, error) {
	mapping := view.domain()
	if mapping.ID == "" {
		mapping.ID = id.New()
	}
	now := s.now()
	mapping.CreatedAt, mapping.UpdatedAt = now, now

	ready, err := importmap.NewMapping(mapping)
	if err != nil {
		return importmap.Mapping{}, err
	}
	if err := s.deps.Mappings.Upsert(ctx, ready); err != nil {
		return importmap.Mapping{}, err
	}
	return ready, nil
}

func (s *Service) SaveMapping(ctx context.Context, req SaveMappingRequest) (SaveMappingResponse, error) {
	if err := s.requireSite(ctx, req.Mapping.SiteID); err != nil {
		return SaveMappingResponse{}, err
	}
	saved, err := s.save(ctx, req.Mapping)
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
