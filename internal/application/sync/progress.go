package sync

import (
	"context"
	"maps"
	"strings"

	"github.com/davidmovas/postulator/internal/application/observe"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

func (s *Service) Progress(ctx context.Context, req ProgressRequest) (ProgressResponse, error) {
	siteID := strings.TrimSpace(req.SiteID)
	if siteID == "" {
		return ProgressResponse{}, invalid("a sync's progress needs its site", "siteId")
	}

	record, found, err := s.syncOf(ctx, siteID, strings.TrimSpace(req.RunID))
	if err != nil || !found {
		return ProgressResponse{}, err
	}
	items, err := s.items.ByRun(ctx, record.ID)
	if err != nil {
		return ProgressResponse{}, err
	}

	out := ProgressResponse{
		Found: true, RunID: record.ID, Status: string(record.Status), Error: record.Error, CreatedAt: dto.NewTime(record.CreatedAt),
		UnplacedBy: map[string]int{},
	}
	if len(items) == 0 {
		return out, nil
	}
	item := items[0]
	out.Attempts = item.Attempts
	if item.Error != "" {
		out.Error = item.Error
	}
	if item.WakeAt != nil {
		wake := dto.NewTime(*item.WakeAt)
		out.WakeAt = &wake
	}
	held, walked, err := run.Get[observe.Progress](item.Checkpoint, observe.CheckpointKey)
	if err != nil || !walked {
		return out, err
	}
	state := held.State
	out.Stage, out.Source, out.Full, out.Batches = string(state.Stage), state.Source, state.Full, held.Batches
	out.Pulled, out.Created, out.Updated = state.Counts.Pulled, state.Counts.Created, state.Counts.Updated
	out.Products, out.Rendered, out.Unreachable = state.Counts.Products, state.Counts.Rendered, state.Counts.Unreachable
	out.Archived = state.Counts.Archived
	out.Entities, out.Placed, out.Unplaced = state.Counts.Entities, state.Counts.Placed, state.Counts.Unplaced
	maps.Copy(out.UnplacedBy, state.Counts.UnplacedBy)
	if !state.StartedAt.IsZero() {
		started := dto.NewTime(state.StartedAt)
		out.StartedAt = &started
	}
	return out, nil
}

func (s *Service) syncOf(ctx context.Context, siteID, runID string) (run.Run, bool, error) {
	if runID == "" {
		return s.syncs.ActiveSync(ctx, siteID)
	}
	record, err := s.syncs.Get(ctx, runID)
	if err != nil {
		return run.Run{}, false, err
	}
	if record.SiteID != siteID || record.Kind != run.KindSync {
		return run.Run{}, false, invalid("the run is not a sync of this site", "runId").WithDetail("runId", runID)
	}
	return record, true, nil
}
