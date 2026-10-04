package imports

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func (s *Service) Preview(ctx context.Context, req PreviewRequest) (PreviewResponse, error) {
	book, err := s.compute(ctx, req)
	if err != nil {
		return PreviewResponse{}, err
	}
	return PreviewResponse{Report: book.report}, nil
}

func (s *Service) Apply(ctx context.Context, req ApplyRequest) (ApplyResponse, error) {
	book, err := s.compute(ctx, req.preview())
	if err != nil {
		return ApplyResponse{}, err
	}
	if book.broken() {
		return ApplyResponse{}, errors.New(errors.Invalid, "the import cannot be applied while the preview reports errors").
			WithDetail("errors", book.report.Errors)
	}

	err = s.deps.UnitOfWork.Do(ctx, func(c context.Context) error {
		for i := range book.plans {
			if writeErr := s.write(c, &book.plans[i]); writeErr != nil {
				return writeErr
			}
		}
		for i := range book.unused {
			if deleteErr := s.deps.Categories.Delete(c, book.unused[i].ID); deleteErr != nil {
				return deleteErr
			}
		}
		return s.remember(c, req, book.plans)
	})
	if err != nil {
		return ApplyResponse{}, err
	}
	if err := s.announce(req.SiteID); err != nil {
		return ApplyResponse{}, err
	}
	return ApplyResponse{Report: book.report, Counts: book.counts()}, nil
}

func (s *Service) write(ctx context.Context, computed *plan) error {
	for i := range computed.categories {
		if err := s.deps.Categories.Insert(ctx, computed.categories[i]); err != nil {
			return err
		}
	}
	for i := range computed.entities {
		planned := &computed.entities[i]
		write := s.deps.Entities.Update
		if planned.created {
			write = s.deps.Entities.Insert
		}
		if err := write(ctx, planned.entity); err != nil {
			return err
		}
	}
	for i := range computed.edges {
		if err := s.deps.Edges.Insert(ctx, computed.edges[i]); err != nil {
			return err
		}
	}
	for i := range computed.pages {
		planned := &computed.pages[i]
		write := s.deps.Pages.Update
		if planned.created {
			write = s.deps.Pages.Insert
		}
		if err := write(ctx, planned.page); err != nil {
			return err
		}
	}

	now := s.now()
	for _, owned := range computed.canonical {
		if err := s.deps.Entities.SetCanonicalPage(ctx, owned.entityID, &owned.pageID, now); err != nil {
			return err
		}
	}
	return s.settleScopes(ctx, computed.siteID, now)
}

func (s *Service) settleScopes(ctx context.Context, siteID string, now time.Time) error {
	entities, err := s.deps.Entities.ListBySite(ctx, siteID)
	if err != nil {
		return err
	}
	edges, err := s.deps.Edges.ListBySite(ctx, siteID)
	if err != nil {
		return err
	}
	moved, err := graph.Settle(entities, edges)
	if err != nil {
		return err
	}
	for i := range moved {
		if setErr := s.deps.Entities.SetScope(ctx, moved[i].ID, moved[i].ScopeID, now); setErr != nil {
			return setErr
		}
	}
	return nil
}

func (s *Service) remember(ctx context.Context, req ApplyRequest, plans []plan) error {
	if req.Options.SaveMappingAs == "" {
		return nil
	}
	for i := range plans {
		used := plans[i].mapping
		used.SiteID = req.SiteID
		used.Name = req.Options.SaveMappingAs
		if len(req.Sheets) > 0 {
			used.ID = ""
			if sheets := used.Options.Sheets; len(sheets) == 1 {
				used.Name += " / " + sheets[0]
			}
		}
		if _, err := s.save(ctx, used); err != nil {
			return err
		}
	}
	return nil
}
