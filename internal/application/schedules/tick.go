package schedules

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/schedule"
	kctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const SkippedStillRunning = "the previous run has not finished"

func (s *Service) RunNow(ctx context.Context, req RunNowRequest) (RunNowResponse, error) {
	current, err := s.require(ctx, req.ID)
	if err != nil {
		return RunNowResponse{}, err
	}

	started, err := s.trigger(ctx, current)
	if err != nil {
		return RunNowResponse{}, err
	}
	return started, nil
}

func (s *Service) Tick(ctx context.Context) (TickResponse, error) {
	now := s.now()
	due, err := s.deps.Schedules.Due(ctx, now, DueBatch)
	if err != nil {
		return TickResponse{}, err
	}

	answer := TickResponse{Started: make([]string, 0, len(due))}
	for i := range due {
		outcome, triggerErr := s.trigger(ctx, due[i])
		if triggerErr != nil {
			return answer, triggerErr
		}
		if outcome.RunID == "" {
			answer.Skipped++
			continue
		}
		answer.Started = append(answer.Started, outcome.RunID)
	}
	return answer, nil
}

func (s *Service) trigger(ctx context.Context, current schedule.Schedule) (RunNowResponse, error) {
	now := s.now()

	busy, err := s.stillRunning(ctx, current)
	if err != nil {
		return RunNowResponse{}, err
	}
	if busy {
		if rearmErr := s.rearm(ctx, current, now, nil); rearmErr != nil {
			return RunNowResponse{}, rearmErr
		}
		return RunNowResponse{Skipped: SkippedStillRunning}, nil
	}

	targets, err := s.targets(ctx, current)
	if err != nil {
		return RunNowResponse{}, err
	}
	if len(targets) == 0 {
		if rearmErr := s.rearm(ctx, current, now, nil); rearmErr != nil {
			return RunNowResponse{}, rearmErr
		}
		return RunNowResponse{Skipped: "the target query matched no page"}, nil
	}

	queued, err := s.deps.Runs.Start(kctx.WithActor(ctx, kctx.ActorSchedule), runs.StartRequest{
		SiteID: current.SiteID, PageIDs: targets, TemplateID: current.TemplateID,
		Recipe: current.Recipe, PublishMode: string(current.PublishMode), Budget: current.Budget,
	})
	if err != nil {
		return RunNowResponse{}, err
	}

	if rearmErr := s.rearm(ctx, current, now, &queued.RunID); rearmErr != nil {
		return RunNowResponse{}, rearmErr
	}
	return RunNowResponse{RunID: queued.RunID, Targets: len(targets)}, nil
}

func (s *Service) stillRunning(ctx context.Context, current schedule.Schedule) (bool, error) {
	if current.LastRunID == nil || s.deps.RunReader == nil {
		return false, nil
	}

	previous, err := s.deps.RunReader.Get(ctx, *current.LastRunID)
	if errors.IsCode(err, errors.NotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return previous.Status.Active(), nil
}

func (s *Service) targets(ctx context.Context, current schedule.Schedule) ([]string, error) {
	pages, err := s.deps.Pages.ListBySite(ctx, current.SiteID)
	if err != nil {
		return nil, err
	}

	limit := current.Query.Limit
	if limit <= 0 {
		limit = schedule.MaxTargets
	}

	targets := make([]string, 0, limit)
	for i := range pages {
		if !matches(pages[i], current.Query) {
			continue
		}
		targets = append(targets, pages[i].ID)
		if len(targets) == limit {
			break
		}
	}
	return targets, nil
}

func matches(page pagemap.Page, query schedule.TargetQuery) bool {
	if query.Status != "" && string(page.Status) != query.Status {
		return false
	}
	if query.EntityID == nil {
		return true
	}
	return page.EntityID != nil && *page.EntityID == *query.EntityID
}

func (s *Service) rearm(ctx context.Context, current schedule.Schedule, now time.Time, runID *string) error {
	next := current
	next.UpdatedAt = now
	if runID != nil {
		next.LastRunID = runID
	}

	armed, err := s.arm(next, now)
	if err != nil {
		return err
	}
	if updateErr := s.deps.Schedules.Update(ctx, armed); updateErr != nil {
		return updateErr
	}
	return s.changed(armed.SiteID)
}
