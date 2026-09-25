package runs

import (
	"context"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	kctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

type planned struct {
	record   run.Run
	added    []AddedPage
	assigned map[string]string
	picked   []string
}

func (s *Service) Start(ctx context.Context, req StartRequest) (StartResponse, error) {
	plan, err := s.plan(ctx, req)
	if err != nil {
		return StartResponse{}, err
	}

	estimate, err := s.engine.EstimateRun(ctx, plan.record, plan.assigned)
	if err != nil {
		return StartResponse{}, err
	}
	if blocking := estimate.Blocking(); len(blocking) > 0 {
		return StartResponse{}, refusedByPreflight(blocking)
	}

	if assignErr := s.assign(ctx, plan); assignErr != nil {
		return StartResponse{}, assignErr
	}

	plan.record.ID = id.New()
	queued, err := s.engine.Enqueue(ctx, plan.record)
	if err != nil {
		return StartResponse{}, err
	}
	return StartResponse{RunID: queued.ID, Estimate: estimate, Added: plan.added}, nil
}

func (s *Service) Estimate(ctx context.Context, req StartRequest) (EstimateResponse, error) {
	plan, err := s.plan(ctx, req)
	if err != nil {
		return EstimateResponse{}, err
	}

	estimate, err := s.engine.EstimateRun(ctx, plan.record, plan.assigned)
	if err != nil {
		return EstimateResponse{}, err
	}
	return EstimateResponse{Estimate: estimate, Added: plan.added}, nil
}

func refusedByPreflight(blocking []run.EstimateFinding) error {
	messages := make([]string, 0, len(blocking))
	for i := range blocking {
		messages = append(messages, blocking[i].Message)
	}
	return invalid("the run cannot start until this is settled: "+strings.Join(messages, "; "), "pageIds").
		WithDetail("findings", blocking)
}

func (s *Service) plan(ctx context.Context, req StartRequest) (planned, error) {
	siteID := strings.TrimSpace(req.SiteID)
	if siteID == "" {
		return planned{}, invalid("a run needs a site", "siteId")
	}

	targets, err := uniqueTargets(req.PageIDs)
	if err != nil {
		return planned{}, err
	}

	kind := run.Kind(req.Kind)
	if req.Kind == "" {
		kind = run.KindGenerate
	}
	if !kind.Valid() {
		return planned{}, invalid("run kind is not recognized", "kind")
	}

	if req.Budget.MaxUSD < 0 || req.Budget.MaxTokens < 0 {
		return planned{}, invalid("a budget must not be negative", "budget")
	}

	mode := run.PublishMode(req.PublishMode)
	if req.PublishMode == "" {
		mode = run.PublishDraft
	}
	if !mode.Valid() {
		return planned{}, invalid("publish mode is not recognized", "publishMode")
	}

	var (
		spec       template.TemplateSpec
		templateID = strings.TrimSpace(req.TemplateID)
		version    int
		picked     string
	)
	if _, owns := kind.Recipe(); kind.PageScoped() && !owns {
		picked = templateID
	}
	if kind.PageScoped() {
		if mappedErr := s.mapped(ctx, targets); mappedErr != nil {
			return planned{}, mappedErr
		}
		first, resolveErr := s.resolve(ctx, siteID, targets[0], picked)
		if resolveErr != nil {
			return planned{}, resolveErr
		}
		spec = first.Spec
		version = first.Version
		if templateID == "" || picked != "" {
			templateID = first.TemplateID
		}
	}

	recipe, err := recipeFor(kind, req.Recipe, spec.Recipe)
	if err != nil {
		return planned{}, err
	}

	chosen := slices.Clone(targets)
	added := make([]AddedPage, 0)
	if kind.PageScoped() && slices.Contains(stepsOf(recipe), string(run.StepPublish)) {
		if added, err = s.ancestors(ctx, siteID, targets); err != nil {
			return planned{}, err
		}
		for _, page := range added {
			targets = append(targets, page.PageID)
		}
	}
	if kind.PageScoped() {
		for _, pageID := range chosen[1:] {
			if _, resolveErr := s.resolve(ctx, siteID, pageID, picked); resolveErr != nil {
				return planned{}, resolveErr
			}
		}
		for _, page := range added {
			if _, resolveErr := s.resolve(ctx, siteID, page.PageID, ""); resolveErr != nil {
				return planned{}, resolveErr
			}
		}
	}

	assigned, assign := assignment(chosen, picked)

	actor, ok := kctx.ActorFrom(ctx)
	if !ok {
		actor = kctx.ActorUser
	}

	return planned{
		record: run.Run{
			SiteID:          siteID,
			Kind:            kind,
			Targets:         targets,
			Recipe:          recipe,
			TemplateID:      templateID,
			TemplateVersion: version,
			PublishMode:     mode,
			Budget:          req.Budget,
			CreatedBy:       actor,
		},
		added:    added,
		assigned: assigned,
		picked:   assign,
	}, nil
}

func assignment(chosen []string, templateID string) (assigned map[string]string, picked []string) {
	if templateID == "" {
		return nil, nil
	}
	assigned = make(map[string]string, len(chosen))
	for _, pageID := range chosen {
		assigned[pageID] = templateID
	}
	return assigned, chosen
}

func (s *Service) assign(ctx context.Context, plan planned) error {
	if len(plan.picked) == 0 {
		return nil
	}
	if err := run.ValidateRecipe(s.steps, plan.record.Recipe); err != nil {
		return err
	}
	_, err := s.assigner.AssignTemplate(ctx, pages.AssignTemplateRequest{
		SiteID: plan.record.SiteID, PageIDs: plan.picked, TemplateID: plan.record.TemplateID,
	})
	return err
}

func (s *Service) resolve(ctx context.Context, siteID, pageID, templateID string) (templates.ResolveForPageResponse, error) {
	resolved, err := s.specs.ResolveForPage(ctx, templates.ResolveForPageRequest{PageID: pageID, TemplateID: templateID})
	if err != nil {
		return templates.ResolveForPageResponse{}, err
	}
	if resolved.SiteID != siteID {
		return templates.ResolveForPageResponse{}, invalid("a target page belongs to another site", "pageIds").
			WithDetail("pageId", pageID).WithDetail("siteId", resolved.SiteID)
	}
	return resolved, nil
}

func (s *Service) ancestors(ctx context.Context, siteID string, targets []string) ([]AddedPage, error) {
	chosen := make(map[string]struct{}, len(targets))
	for _, pageID := range targets {
		chosen[pageID] = struct{}{}
	}

	active, err := s.items.ActiveBySite(ctx, siteID)
	if err != nil {
		return nil, err
	}
	underway := make(map[string]struct{}, len(active))
	for i := range active {
		underway[active[i].TargetID] = struct{}{}
	}

	added := make([]AddedPage, 0)
	for _, pageID := range targets {
		child, getErr := s.pages.Get(ctx, pageID)
		if getErr != nil {
			return nil, getErr
		}
		neededBy := child.Path

		for {
			parent, found, parentErr := s.parentOf(ctx, child)
			if parentErr != nil || !found {
				if parentErr != nil {
					return nil, parentErr
				}
				break
			}
			if parent.WPID != nil {
				break
			}
			if _, busy := underway[parent.ID]; busy {
				break
			}
			if _, picked := chosen[parent.ID]; !picked {
				if parent.EntityID == nil || strings.TrimSpace(*parent.EntityID) == "" {
					return nil, invalid("the parent "+parent.Path+" of "+neededBy+" is not on the site and nothing is "+
						"mapped to it, so it cannot be written first; map it on the Graph screen or publish it", "pageIds").
						WithDetail("parentPath", parent.Path).WithDetail("path", neededBy)
				}
				chosen[parent.ID] = struct{}{}
				added = append(added, AddedPage{PageID: parent.ID, Path: parent.Path, NeededBy: neededBy})
			}
			child = parent
		}
	}
	return added, nil
}

func (s *Service) parentOf(ctx context.Context, child pagemap.Page) (pagemap.Page, bool, error) {
	wanted := pagemap.ParentPath(child.Path)
	if wanted == "" || wanted == "/" {
		return pagemap.Page{}, false, nil
	}

	missing := invalid("the page map holds no page at "+wanted+", so "+child.Path+
		" has no parent to sit under; add "+wanted+" to the page map first", "pageIds").
		WithDetail("parentPath", wanted).WithDetail("path", child.Path)
	if child.ParentPageID == nil {
		return pagemap.Page{}, false, missing
	}

	parent, err := s.pages.Get(ctx, *child.ParentPageID)
	if errors.IsCode(err, errors.NotFound) {
		return pagemap.Page{}, false, missing
	}
	if err != nil {
		return pagemap.Page{}, false, err
	}
	if parent.Path != wanted {
		return pagemap.Page{}, false, invalid(child.Path+" is linked to "+parent.Path+" while its path asks for "+
			wanted+"; fix the link on the Pages screen first", "pageIds").
			WithDetail("parentPath", wanted).WithDetail("linkedPath", parent.Path).WithDetail("path", child.Path)
	}
	return parent, true, nil
}

func (s *Service) mapped(ctx context.Context, targets []string) error {
	unmapped := make([]string, 0, len(targets))
	for _, pageID := range targets {
		page, err := s.pages.Get(ctx, pageID)
		if err != nil {
			return err
		}
		if page.EntityID != nil && strings.TrimSpace(*page.EntityID) != "" {
			continue
		}
		where := page.Path
		if where == "" {
			where = pageID
		}
		unmapped = append(unmapped, where)
	}
	if len(unmapped) == 0 {
		return nil
	}
	return invalid("a run plans a page's links from the entity it is mapped to, and nothing is mapped to "+
		strings.Join(unmapped, ", ")+"; map each of them on the Graph screen first", "pageIds").
		WithDetail("paths", unmapped)
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
