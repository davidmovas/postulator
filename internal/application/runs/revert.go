package runs

import (
	"context"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
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
	if pageID := strings.TrimSpace(req.PageID); pageID != "" {
		if !slices.Contains(targets, pageID) {
			return RevertResponse{}, invalid("the run wrote nothing to that page", "pageId")
		}
		targets = []string{pageID}
	}
	recipe, err := revertRecipe(req.Fields)
	if err != nil {
		return RevertResponse{}, err
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
		Recipe:      recipe,
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
		if children[i].Kind == run.KindRevert && children[i].Status.Active() {
			return errors.New(errors.Conflict, "a revert of this run is still under way").
				WithDetail("runId", runID).WithDetail("revertRunId", children[i].ID)
		}
	}
	return nil
}

func revertRecipe(fields []string) ([]template.StepSpec, error) {
	recipe := run.RevertRecipe()
	if len(fields) == 0 {
		return recipe, nil
	}
	named := make([]any, 0, len(fields))
	for _, raw := range fields {
		key := field.Key(strings.TrimSpace(raw))
		if spec, known := field.Lookup(key); !known || spec.Denied {
			return nil, invalid("no field the run could have written is named "+string(key), "fields")
		}
		named = append(named, string(key))
	}
	for i := range recipe {
		if recipe[i].Name == string(run.StepRevert) {
			recipe[i].Params = map[string]any{"fields": named}
		}
	}
	return recipe, nil
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
