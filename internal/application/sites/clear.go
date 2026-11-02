package sites

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const CodeSiteBusy = "site_busy"

type siteData interface {
	Counts(ctx context.Context, siteID string) (site.Held, error)
	Busy(ctx context.Context, siteID string) ([]string, error)
	Clear(ctx context.Context, siteID string, what site.Clearing) error
}

func (s *Service) refuseBusy(ctx context.Context, siteID, doing string) error {
	busy, err := s.data.Busy(ctx, siteID)
	if err != nil {
		return err
	}
	if len(busy) == 0 {
		return nil
	}
	return errors.New(errors.Conflict, "a run of this site is under way; stop it in Runs before you "+doing).
		WithDetail("code", CodeSiteBusy).WithDetail("siteId", siteID).WithDetail("runIds", busy)
}

func (s *Service) DataCounts(ctx context.Context, req DataCountsRequest) (DataCountsResponse, error) {
	if _, err := s.store.Get(ctx, req.ID); err != nil {
		return DataCountsResponse{}, err
	}
	held, err := s.data.Counts(ctx, req.ID)
	if err != nil {
		return DataCountsResponse{}, err
	}
	busy, err := s.data.Busy(ctx, req.ID)
	if err != nil {
		return DataCountsResponse{}, err
	}
	return DataCountsResponse{Held: heldView(held), Busy: busy}, nil
}

func (s *Service) ClearData(ctx context.Context, req ClearDataRequest) (ClearDataResponse, error) {
	what := site.Clearing{Model: req.Model, Graph: req.Graph, History: req.History}.Normalized()
	if what.Empty() {
		return ClearDataResponse{}, invalid("choose the model, the graph or the history to clear", "model")
	}

	var held site.Held
	err := s.uow.Do(ctx, func(c context.Context) error {
		if _, getErr := s.store.Get(c, req.ID); getErr != nil {
			return getErr
		}
		if busyErr := s.refuseBusy(c, req.ID, "clear its data"); busyErr != nil {
			return busyErr
		}
		if clearErr := s.data.Clear(c, req.ID, what); clearErr != nil {
			return clearErr
		}
		if what.Model {
			if resetErr := s.store.UpdateObservation(c, req.ID, site.TechnicalPages{}, site.ObservationState{}, s.now()); resetErr != nil {
				return resetErr
			}
		}
		counted, countErr := s.data.Counts(c, req.ID)
		held = counted
		return countErr
	})
	if err != nil {
		return ClearDataResponse{}, err
	}
	if publishErr := s.clearedOut(req.ID, what); publishErr != nil {
		return ClearDataResponse{}, publishErr
	}
	return ClearDataResponse{Held: heldView(held)}, nil
}

func (s *Service) clearedOut(siteID string, what site.Clearing) error {
	if what.Model || what.Graph {
		if err := s.publisher.Publish(events.PagesChanged, events.PagesChangedPayload{SiteID: siteID}); err != nil {
			return err
		}
		if err := s.publisher.Publish(events.GraphChanged, events.GraphChangedPayload{SiteID: siteID}); err != nil {
			return err
		}
	}
	return s.changed(siteID)
}
