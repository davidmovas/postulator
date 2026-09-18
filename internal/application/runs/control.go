package runs

import (
	"context"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/run"
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
		return PauseResponse{}, invalid("pause reason is not recognized", "reason")
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
	if err := s.engine.RetryStep(ctx, itemID); err != nil {
		return RetryStepResponse{}, err
	}
	return RetryStepResponse{}, nil
}
