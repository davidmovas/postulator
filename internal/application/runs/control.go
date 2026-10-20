package runs

import (
	"context"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/run"
	kctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func (s *Service) Pause(ctx context.Context, req PauseRequest) (PauseResponse, error) {
	runID := strings.TrimSpace(req.RunID)
	if runID == "" {
		return PauseResponse{}, invalid("a pause needs a run", "runId")
	}

	reason := run.PauseReason(req.Reason)
	if req.Reason == "" {
		reason = run.PauseUser
	}
	if !reason.Valid() {
		allowed := run.PauseReasons()
		return PauseResponse{}, invalid("pause reason is not recognized; it is one of "+
			strings.Join(allowed, ", "), "reason").WithDetail("allowed", allowed)
	}
	if reason == run.PauseAwaitingReview {
		return PauseResponse{}, invalid("only a draft run's step holds a page for review, never a pause by hand", "reason")
	}

	if err := s.engine.Pause(ctx, runID, reason); err != nil {
		return PauseResponse{}, err
	}
	return PauseResponse{}, nil
}

func (s *Service) Resume(ctx context.Context, req ResumeRequest) (ResumeResponse, error) {
	runID := strings.TrimSpace(req.RunID)
	if runID == "" {
		return ResumeResponse{}, invalid("a resume needs a run", "runId")
	}
	if err := s.engine.Resume(ctx, runID); err != nil {
		return ResumeResponse{}, err
	}
	return ResumeResponse{}, nil
}

func (s *Service) Cancel(ctx context.Context, req CancelRequest) (CancelResponse, error) {
	runID := strings.TrimSpace(req.RunID)
	if runID == "" {
		return CancelResponse{}, invalid("a cancellation needs a run", "runId")
	}
	if err := s.engine.Cancel(ctx, runID); err != nil {
		return CancelResponse{}, err
	}
	return CancelResponse{}, nil
}

func (s *Service) RetryStep(ctx context.Context, req RetryStepRequest) (RetryStepResponse, error) {
	itemID := strings.TrimSpace(req.ItemID)
	if itemID == "" {
		return RetryStepResponse{}, invalid("a retry needs a run item", "itemId")
	}
	requeue := s.engine.RetryStep
	if req.AcceptFindings {
		if refusal := s.refuseAnAgentsReview(ctx, itemID); refusal != nil {
			return RetryStepResponse{}, refusal
		}
		requeue = s.engine.Accept
	}
	if err := requeue(ctx, itemID); err != nil {
		return RetryStepResponse{}, err
	}
	return RetryStepResponse{}, nil
}

func (s *Service) refuseAnAgentsReview(ctx context.Context, itemID string) error {
	if actor, ok := kctx.ActorFrom(ctx); !ok || actor != kctx.ActorAgent {
		return nil
	}
	item, err := s.items.Get(ctx, itemID)
	if err != nil {
		return err
	}
	if item.PauseReason != run.PauseAwaitingReview {
		return nil
	}
	return errors.New(errors.Conflict, "the changes of a draft run are accepted by a person, who reviews them in the run's "+
		"report; the assistant may not accept them").WithDetail("itemId", itemID).WithDetail("step", item.CurrentStep)
}

func (s *Service) Regenerate(ctx context.Context, req RegenerateRequest) (RegenerateResponse, error) {
	runID := strings.TrimSpace(req.RunID)
	if runID == "" {
		return RegenerateResponse{}, invalid("a regeneration needs a run", "runId")
	}

	itemIDs := make([]string, 0, len(req.ItemIDs))
	for _, raw := range req.ItemIDs {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || slices.Contains(itemIDs, trimmed) {
			continue
		}
		itemIDs = append(itemIDs, trimmed)
	}
	if len(itemIDs) == 0 {
		return RegenerateResponse{}, invalid("a regeneration needs at least one run item", "itemIds")
	}

	if err := s.engine.Regenerate(ctx, runID, itemIDs); err != nil {
		return RegenerateResponse{}, err
	}
	return RegenerateResponse{Restarted: len(itemIDs)}, nil
}
