package runs

import (
	"context"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/run"
	kctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func (s *Service) Revert(ctx context.Context, req RevertRequest) (RevertResponse, error) {
	runID := strings.TrimSpace(req.RunID)
	if runID == "" {
		return RevertResponse{}, invalid("a revert needs a run", "runId")
	}

	source, err := s.runs.Get(ctx, runID)
	if err != nil {
		return RevertResponse{}, err
	}
	if source.Kind == run.KindRevert {
		return RevertResponse{}, errors.New(errors.Conflict,
			"a revert cannot itself be reverted; run the work again instead").WithDetail("runId", runID)
	}
	if !source.Status.Terminal() {
		return RevertResponse{}, errors.New(errors.Conflict,
			"only a run that has finished can be reverted; cancel it first").
			WithDetail("runId", runID).WithDetail("status", string(source.Status))
	}
	if refusal := s.refuseSecondRevert(ctx, runID); refusal != nil {
		return RevertResponse{}, refusal
	}

	targets, err := s.published(ctx, runID)
	if err != nil {
		return RevertResponse{}, err
	}
	if len(targets) == 0 {
		return RevertResponse{}, errors.New(errors.Invalid,
			"the run wrote nothing to the site that could be put back").WithDetail("runId", runID)
	}

	actor, ok := kctx.ActorFrom(ctx)
	if !ok {
		actor = kctx.ActorUser
	}

	queued, err := s.engine.Enqueue(ctx, run.Run{
		ID:          id.New(),
		SiteID:      source.SiteID,
		Kind:        run.KindRevert,
		Targets:     targets,
		Recipe:      run.RevertRecipe(),
		TemplateID:  source.TemplateID,
		PublishMode: source.PublishMode,
		ParentRunID: &source.ID,
		CreatedBy:   actor,
	})
	if err != nil {
		return RevertResponse{}, err
	}
	return RevertResponse{RunID: queued.ID}, nil
}

func (s *Service) refuseSecondRevert(ctx context.Context, runID string) error {
	children, err := s.runs.ByParent(ctx, runID)
	if err != nil {
		return err
	}

	for i := range children {
		if children[i].Kind != run.KindRevert {
			continue
		}
		if children[i].Status == run.StatusCompleted || children[i].Status.Active() {
			return errors.New(errors.Conflict, "this run has already been reverted").
				WithDetail("runId", runID).WithDetail("revertRunId", children[i].ID)
		}
	}
	return nil
}

func (s *Service) published(ctx context.Context, runID string) ([]string, error) {
	items, err := s.items.ByRun(ctx, runID)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(items))
	for i := range items {
		ids = append(ids, items[i].ID)
	}
	held, err := s.artifacts.KindsByItems(ctx, ids)
	if err != nil {
		return nil, err
	}

	targets := make([]string, 0, len(items))
	for i := range items {
		if !slices.Contains(held[items[i].ID], run.ArtifactPublishResult) {
			continue
		}
		if slices.Contains(targets, items[i].TargetID) {
			continue
		}
		targets = append(targets, items[i].TargetID)
	}
	return targets, nil
}
