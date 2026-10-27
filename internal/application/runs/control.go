package runs

import (
	"context"
	"encoding/json"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/field"
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
	fields, err := s.acceptedFields(ctx, itemID, req)
	if err != nil {
		return RetryStepResponse{}, err
	}
	if !req.AcceptFindings {
		return RetryStepResponse{}, s.engine.RetryStep(ctx, itemID)
	}
	if refusal := s.refuseAnAgentsReview(ctx, itemID); refusal != nil {
		return RetryStepResponse{}, refusal
	}
	return RetryStepResponse{}, s.engine.Accept(ctx, itemID, fields)
}

func (s *Service) acceptedFields(ctx context.Context, itemID string, req RetryStepRequest) ([]field.Key, error) {
	switch {
	case req.Fields == nil:
		return nil, nil
	case !req.AcceptFindings:
		return nil, invalid("fields are named when a person accepts a review; a retry names none", "fields")
	case len(req.Fields) == 0:
		return nil, invalid("an acceptance that names no field writes nothing; cancel the page to keep it as it is", "fields")
	}
	item, err := s.items.Get(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if item.PauseReason != run.PauseAwaitingReview {
		return nil, invalid("only a page held for review accepts its changes field by field", "fields")
	}
	offered, err := s.reviewedFields(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if len(offered) == 0 {
		return nil, invalid("this review changes page bodies only, so it is accepted whole", "fields")
	}
	out := make([]field.Key, 0, len(req.Fields))
	for _, raw := range req.Fields {
		key := field.Key(strings.TrimSpace(raw))
		if spec, known := field.Lookup(key); !known || spec.Denied || !slices.Contains(offered, key) {
			return nil, invalid("the review shows no change to "+string(key), "fields")
		}
		if !slices.Contains(out, key) {
			out = append(out, key)
		}
	}
	return out, nil
}

type reviewedPlan struct {
	Pages []struct {
		Body   field.Key `json:"body"`
		Fields []struct {
			Key field.Key `json:"key"`
		} `json:"fields"`
	} `json:"pages"`
}

func (s *Service) reviewedFields(ctx context.Context, itemID string) ([]field.Key, error) {
	latest, err := s.latestOf(ctx, itemID, run.ArtifactEditPlan)
	if err != nil {
		return nil, err
	}
	if latest == nil {
		return nil, invalid("the page holds no review to accept field by field", "fields")
	}
	var plan reviewedPlan
	if decodeErr := json.Unmarshal(latest.Blob, &plan); decodeErr != nil {
		return nil, errors.Wrap(decodeErr, errors.Internal, "decode the review of the page")
	}
	fields, bodies := make([]field.Key, 0), make([]field.Key, 0)
	for _, page := range plan.Pages {
		for _, change := range page.Fields {
			fields = append(fields, change.Key)
		}
		if page.Body != "" {
			bodies = append(bodies, page.Body)
		}
	}
	if len(fields) == 0 {
		return nil, nil
	}
	return append(fields, bodies...), nil
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
