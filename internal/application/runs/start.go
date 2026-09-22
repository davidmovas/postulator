package runs

import (
	"context"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	kctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func (s *Service) Start(ctx context.Context, req StartRequest) (StartResponse, error) {
	record, spec, err := s.plan(ctx, req)
	if err != nil {
		return StartResponse{}, err
	}

	estimate, err := s.engine.EstimateRun(ctx, record, spec)
	if err != nil {
		return StartResponse{}, err
	}

	record.ID = id.New()
	queued, err := s.engine.Enqueue(ctx, record)
	if err != nil {
		return StartResponse{}, err
	}
	return StartResponse{RunID: queued.ID, Estimate: estimate}, nil
}

func (s *Service) Estimate(ctx context.Context, req StartRequest) (EstimateResponse, error) {
	record, spec, err := s.plan(ctx, req)
	if err != nil {
		return EstimateResponse{}, err
	}

	estimate, err := s.engine.EstimateRun(ctx, record, spec)
	if err != nil {
		return EstimateResponse{}, err
	}
	return EstimateResponse{Estimate: estimate}, nil
}

func (s *Service) plan(ctx context.Context, req StartRequest) (run.Run, template.TemplateSpec, error) {
	siteID := strings.TrimSpace(req.SiteID)
	if siteID == "" {
		return run.Run{}, template.TemplateSpec{}, invalid("a run needs a site", "siteId")
	}

	targets, err := uniqueTargets(req.PageIDs)
	if err != nil {
		return run.Run{}, template.TemplateSpec{}, err
	}

	kind := run.Kind(req.Kind)
	if req.Kind == "" {
		kind = run.KindGenerate
	}
	if !kind.Valid() {
		return run.Run{}, template.TemplateSpec{}, invalid("run kind is not recognized", "kind")
	}

	mode := run.PublishMode(req.PublishMode)
	if req.PublishMode == "" {
		mode = run.PublishDraft
	}
	if !mode.Valid() {
		return run.Run{}, template.TemplateSpec{}, invalid("publish mode is not recognized", "publishMode")
	}

	var (
		spec       template.TemplateSpec
		templateID = strings.TrimSpace(req.TemplateID)
		version    int
	)
	if kind.PageScoped() {
		for i, pageID := range targets {
			resolved, resolveErr := s.specs.ResolveForPage(ctx, templates.ResolveForPageRequest{PageID: pageID})
			if resolveErr != nil {
				return run.Run{}, template.TemplateSpec{}, resolveErr
			}
			if resolved.SiteID != siteID {
				return run.Run{}, template.TemplateSpec{}, invalid("a target page belongs to another site", "pageIds").
					WithDetail("pageId", pageID).WithDetail("siteId", resolved.SiteID)
			}
			if i > 0 {
				continue
			}
			spec = resolved.Spec
			version = resolved.Version
			if templateID == "" {
				templateID = resolved.TemplateID
			}
		}
	}

	recipe, err := recipeFor(kind, req.Recipe, spec.Recipe)
	if err != nil {
		return run.Run{}, template.TemplateSpec{}, err
	}

	actor, ok := kctx.ActorFrom(ctx)
	if !ok {
		actor = kctx.ActorUser
	}

	return run.Run{
		SiteID:          siteID,
		Kind:            kind,
		Targets:         targets,
		Recipe:          recipe,
		TemplateID:      templateID,
		TemplateVersion: version,
		PublishMode:     mode,
		Budget:          req.Budget,
		CreatedBy:       actor,
	}, spec, nil
}

func recipeFor(kind run.Kind, requested, templated []template.StepSpec) ([]template.StepSpec, error) {
	if kind == run.KindRevert {
		return nil, invalid("a revert is started from the run it undoes, not from a list of pages", "kind")
	}

	own, owns := kind.Recipe()
	if owns {
		if len(requested) > 0 && !slices.Equal(stepsOf(requested), stepsOf(own)) {
			return nil, invalid("a "+string(kind)+" run names its own steps, so it takes no recipe; "+
				"start a custom run to choose the steps yourself", "recipe").
				WithDetail("kind", string(kind)).WithDetail("steps", stepsOf(own))
		}
		return own, nil
	}

	recipe := requested
	if len(recipe) == 0 {
		recipe = templated
	}
	if len(recipe) == 0 {
		recipe = run.GenerateRecipe()
	}
	if kind == run.KindCustom {
		return recipe, nil
	}
	for _, step := range recipe {
		if !step.Enabled || !run.PerKindStep(step.Name) {
			continue
		}
		return nil, invalid("the step "+step.Name+" belongs to a run of its own and cannot be part of "+
			"a "+string(kind)+" run; start a "+ownerOf(step.Name)+" run instead, or take the step out of "+
			"the recipe", "recipe").
			WithDetail("step", step.Name).WithDetail("kind", ownerOf(step.Name))
	}
	return recipe, nil
}

func ownerOf(step string) string {
	for _, kind := range []run.Kind{run.KindRelink, run.KindRepair, run.KindSync, run.KindRevert} {
		own, _ := kind.Recipe()
		if slices.Contains(stepsOf(own), step) {
			return string(kind)
		}
	}
	return string(run.KindCustom)
}

func stepsOf(recipe []template.StepSpec) []string {
	out := make([]string, 0, len(recipe))
	for _, step := range recipe {
		if step.Enabled {
			out = append(out, step.Name)
		}
	}
	return out
}

func uniqueTargets(pageIDs []string) ([]string, error) {
	out := make([]string, 0, len(pageIDs))
	for _, pageID := range pageIDs {
		trimmed := strings.TrimSpace(pageID)
		if trimmed == "" {
			continue
		}
		if slices.Contains(out, trimmed) {
			continue
		}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil, invalid("a run needs at least one target page", "pageIds")
	}
	return out, nil
}
