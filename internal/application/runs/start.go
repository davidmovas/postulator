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
	siteID := strings.TrimSpace(req.SiteID)
	if siteID == "" {
		return StartResponse{}, invalid("a run needs a site", "siteId")
	}

	targets, err := uniqueTargets(req.PageIDs)
	if err != nil {
		return StartResponse{}, err
	}

	kind := run.Kind(req.Kind)
	if req.Kind == "" {
		kind = run.KindGenerate
	}
	if !kind.Valid() {
		return StartResponse{}, invalid("run kind is not recognized", "kind")
	}

	mode := run.PublishMode(req.PublishMode)
	if req.PublishMode == "" {
		mode = run.PublishDraft
	}
	if !mode.Valid() {
		return StartResponse{}, invalid("publish mode is not recognized", "publishMode")
	}

	var (
		spec       template.TemplateSpec
		templateID = strings.TrimSpace(req.TemplateID)
		version    int
	)
	for i, pageID := range targets {
		resolved, resolveErr := s.specs.ResolveForPage(ctx, templates.ResolveForPageRequest{PageID: pageID})
		if resolveErr != nil {
			return StartResponse{}, resolveErr
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

	recipe := req.Recipe
	if len(recipe) == 0 {
		recipe = spec.Recipe
	}
	if len(recipe) == 0 {
		return StartResponse{}, invalid("neither the request nor the template names a recipe", "recipe")
	}

	actor, ok := kctx.ActorFrom(ctx)
	if !ok {
		actor = kctx.ActorUser
	}

	record := run.Run{
		ID:              id.New(),
		SiteID:          siteID,
		Kind:            kind,
		Targets:         targets,
		Recipe:          recipe,
		TemplateID:      templateID,
		TemplateVersion: version,
		PublishMode:     mode,
		Budget:          req.Budget,
		CreatedBy:       actor,
	}

	estimate, err := s.engine.EstimateRun(ctx, record, spec)
	if err != nil {
		return StartResponse{}, err
	}

	queued, err := s.engine.Enqueue(ctx, record)
	if err != nil {
		return StartResponse{}, err
	}
	return StartResponse{RunID: queued.ID, Estimate: estimate}, nil
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
