package content

import (
	"context"
	"strings"

	"github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/templates"
	contentdomain "github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/graph"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	NameJudge = "judge"

	JudgeTokens = 1024
)

type judgeAnswer struct {
	Score       float64  `json:"score" description:"The overall quality of the page between 0 and 1"`
	Issues      []string `json:"issues" description:"What is wrong with the page, worst first, one sentence each"`
	Suggestions []string `json:"suggestions" description:"What would raise the score, one sentence each"`
}

type JudgeReport struct {
	Score       *float64                `json:"score,omitempty"`
	Issues      []string                `json:"issues"`
	Suggestions []string                `json:"suggestions"`
	Findings    []contentdomain.Finding `json:"findings,omitempty"`
}

func (s *Service) Assess(ctx context.Context, req AssessRequest) (AssessResponse, error) {
	ref, err := s.deps.Profiles.Resolve(ctx, req.SiteID, domainllm.RoleJudge, req.Spec.ModelProfiles)
	if err != nil {
		return AssessResponse{}, err
	}

	system, user, err := render(NameJudge, judgePrompt{
		Page: req.Page, Entity: req.Entity, Spec: req.Spec, Body: req.Body,
		Meta: req.Snippet, HasMeta: req.HasSnippet, Targets: req.Targets,
	})
	if err != nil {
		return AssessResponse{}, err
	}

	answer, usage, err := llm.Structured[judgeAnswer](ctx, s.deps.LLM, llm.Request{
		Ref:       ref,
		System:    system,
		Messages:  []llm.Message{{Role: llm.RoleUser, Text: user}},
		MaxTokens: JudgeTokens,
		Meta:      req.Call,
	})
	if err != nil {
		return AssessResponse{Tokens: usage.Total}, err
	}
	return AssessResponse{Report: settle(answer), Tokens: usage.Total}, nil
}

func (s *Service) Judge(ctx context.Context, req JudgeRequest) (JudgeResponse, error) {
	pageID := strings.TrimSpace(req.PageID)
	if pageID == "" {
		return JudgeResponse{}, errors.New(errors.Invalid, "a page audit needs a page").WithDetail("field", "pageId")
	}

	page, err := s.deps.Pages.Get(ctx, pageID)
	if err != nil {
		return JudgeResponse{}, err
	}
	if page.WPID == nil {
		return JudgeResponse{}, errors.New(errors.Invalid, "the page is not on the site yet, so there is nothing to audit").
			WithDetail("pageId", pageID)
	}
	if page.EntityID == nil {
		return JudgeResponse{}, errors.New(errors.Invalid, "the page is not mapped to an entity, so it has no graph context").
			WithDetail("pageId", pageID)
	}

	raw, err := s.deps.Raw.RawContent(ctx, page.SiteID, *page.WPID)
	if err != nil {
		return JudgeResponse{}, err
	}
	doc, err := contentdomain.Parse(raw)
	if err != nil {
		return JudgeResponse{}, err
	}

	resolved, err := s.deps.Specs.ResolveForPage(ctx, templates.ResolveForPageRequest{PageID: pageID})
	if err != nil {
		return JudgeResponse{}, err
	}

	entity, targets, err := s.context(ctx, page, resolved.Spec)
	if err != nil {
		return JudgeResponse{}, err
	}

	assessed, err := s.Assess(ctx, AssessRequest{
		SiteID: page.SiteID, Page: page, Entity: entity, Spec: resolved.Spec, Body: doc.HTML(),
		Snippet:    Snippet{Title: page.MetaTitle, Description: page.MetaDescription},
		HasSnippet: page.MetaTitle != "" || page.MetaDescription != "",
		Targets:    targets,
	})
	if err != nil {
		return JudgeResponse{}, err
	}
	return JudgeResponse{PageID: page.ID, Path: page.Path, Report: assessed.Report, Tokens: assessed.Tokens}, nil
}

func (s *Service) context(ctx context.Context, page pagemap.Page,
	spec template.TemplateSpec) (graph.Entity, []contentdomain.LinkTarget, error) {
	entities, err := s.deps.Entities.ListBySite(ctx, page.SiteID)
	if err != nil {
		return graph.Entity{}, nil, err
	}
	edges, err := s.deps.Edges.ListBySite(ctx, page.SiteID)
	if err != nil {
		return graph.Entity{}, nil, err
	}
	pages, err := s.deps.Pages.ListBySite(ctx, page.SiteID)
	if err != nil {
		return graph.Entity{}, nil, err
	}
	effective, err := s.deps.Policies.GetEffectivePolicy(ctx, templates.GetEffectivePolicyRequest{SiteID: page.SiteID})
	if err != nil {
		return graph.Entity{}, nil, err
	}
	policy := template.LinkPolicy{
		Rules:          templates.EffectiveRules(effective.Policy.Rules, spec.LinkRules),
		ForbidExternal: effective.Policy.ForbidExternal,
		ForbidSelf:     effective.Policy.ForbidSelf,
		AnchorStrategy: template.AnchorStrategy(effective.Policy.AnchorStrategy),
	}

	var entity graph.Entity
	for i := range entities {
		if entities[i].ID == *page.EntityID {
			entity = entities[i]
		}
	}
	if entity.ID == "" {
		return graph.Entity{}, nil, errors.New(errors.NotFound, "the entity the page is mapped to is gone").
			WithDetail("entityId", *page.EntityID)
	}

	built, err := graph.New(entities, edges)
	if err != nil {
		return graph.Entity{}, nil, err
	}

	planned := contentdomain.PlanLinks(built, pagemap.NewIndex(pages), contentdomain.Subject{
		PageID: page.ID, PagePath: page.Path, EntityID: entity.ID,
	}, policy)
	return entity, planned.Context.Targets, nil
}

func settle(answer judgeAnswer) JudgeReport {
	scored := min(max(answer.Score, 0), 1)
	return JudgeReport{
		Score:       &scored,
		Issues:      trimmed(answer.Issues),
		Suggestions: trimmed(answer.Suggestions),
	}
}

func trimmed(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if text := strings.TrimSpace(line); text != "" {
			out = append(out, text)
		}
	}
	return out
}
