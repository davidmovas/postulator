package content_test

import (
	"context"
	"strings"
	"testing"

	appcontent "github.com/davidmovas/postulator/internal/application/content"
	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/graph"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type pageStub struct {
	items []pagemap.Page
	err   error
}

func (p pageStub) Get(_ context.Context, id string) (pagemap.Page, error) {
	if p.err != nil {
		return pagemap.Page{}, p.err
	}
	for i := range p.items {
		if p.items[i].ID == id {
			return p.items[i], nil
		}
	}
	return pagemap.Page{}, errors.New(errors.NotFound, "no such page")
}

func (p pageStub) ListBySite(context.Context, string) ([]pagemap.Page, error) {
	return p.items, p.err
}

type entityStub struct {
	items []graph.Entity
	err   error
}

func (e entityStub) ListBySite(context.Context, string) ([]graph.Entity, error) {
	return e.items, e.err
}

type edgeStub struct {
	items []graph.Edge
	err   error
}

func (e edgeStub) ListBySite(context.Context, string) ([]graph.Edge, error) {
	return e.items, e.err
}

type specStub struct {
	rules *template.LinkRules
	err   error
}

func (s specStub) ResolveForPage(context.Context, templates.ResolveForPageRequest) (templates.ResolveForPageResponse, error) {
	if s.err != nil {
		return templates.ResolveForPageResponse{}, s.err
	}
	rules := template.LinkRules{UpDepth: 1, DownLinks: true, MaxLinks: 10, MaxPerTarget: 1}
	if s.rules != nil {
		rules = *s.rules
	}
	return templates.ResolveForPageResponse{
		TemplateID: "t1", SiteID: "site", Version: 1,
		Spec: template.TemplateSpec{
			Tone:      "plain",
			Length:    template.Length{Min: 200, Max: 900},
			LinkRules: rules,
		},
	}, nil
}

type policyStub struct {
	rules template.LinkRules
	err   error
}

func (p policyStub) GetEffectivePolicy(context.Context, templates.GetEffectivePolicyRequest) (templates.GetEffectivePolicyResponse, error) {
	if p.err != nil {
		return templates.GetEffectivePolicyResponse{}, p.err
	}
	return templates.GetEffectivePolicyResponse{
		Policy: templates.LinkPolicy{
			Rules: p.rules, ForbidExternal: true, ForbidSelf: true, AnchorStrategy: "prefer_user",
		},
	}, nil
}

type profileStub struct {
	err error
}

func (p profileStub) Resolve(context.Context, string, domainllm.Role, map[domainllm.Role]domainllm.ModelRef) (domainllm.ModelRef, error) {
	if p.err != nil {
		return domainllm.ModelRef{}, p.err
	}
	return domainllm.ModelRef{Provider: "openai", Model: "unit"}, nil
}

type rawStub struct {
	body string
	err  error
}

func (r rawStub) RawContent(context.Context, string, int64) (string, error) {
	return r.body, r.err
}

type llmStub struct {
	reply string
	last  port.Request
	err   error
}

func (l *llmStub) Complete(_ context.Context, req port.Request) (port.Response, error) {
	l.last = req
	if l.err != nil {
		return port.Response{}, l.err
	}
	return port.Response{Text: l.reply, Usage: domainllm.Usage{Input: 10, Output: 20, Total: 30}}, nil
}

func (l *llmStub) Stream(context.Context, port.Request) (<-chan port.Delta, error) {
	return nil, errors.New(errors.Internal, "the stub does not stream")
}

func pointer(value string) *string {
	return &value
}

func wpID(value int64) *int64 {
	return &value
}

func pages() []pagemap.Page {
	return []pagemap.Page{
		{
			ID: "page-parent", SiteID: "site", Path: "/coffee/", WPType: pagemap.WPPage,
			Status: pagemap.StatusPublished, EntityID: pointer("parent"), WPID: wpID(11),
		},
		{
			ID: "page-child", SiteID: "site", Path: "/coffee/espresso/", WPType: pagemap.WPPage,
			Status: pagemap.StatusPublished, EntityID: pointer("child"), WPID: wpID(12),
			MetaTitle: "espresso | Shop", MetaDescription: "Pull a shot.",
		},
	}
}

func entities() []graph.Entity {
	return []graph.Entity{
		{
			ID: "parent", SiteID: "site", Name: "Coffee", PrimaryKeyword: "coffee", Kind: graph.KindTopic,
			Source: graph.SourceUser, CanonicalPageID: pointer("page-parent"),
			Anchors: []graph.Anchor{{Text: "coffee", Source: graph.AnchorUser, Weight: 1}},
		},
		{
			ID: "child", SiteID: "site", Name: "Espresso", PrimaryKeyword: "espresso", Kind: graph.KindTopic,
			Source: graph.SourceUser, CanonicalPageID: pointer("page-child"),
			Anchors: []graph.Anchor{{Text: "espresso", Source: graph.AnchorUser, Weight: 1}},
		},
	}
}

func edges() []graph.Edge {
	return []graph.Edge{{
		ID: "e1", SiteID: "site", FromEntityID: "child", ToEntityID: "parent", Kind: graph.EdgeParent,
		Weight: 1, Source: graph.SourceUser, Status: graph.StatusApproved,
	}}
}

func newService(model *llmStub, mutate func(*appcontent.Deps)) *appcontent.Service {
	deps := appcontent.Deps{
		Pages:    pageStub{items: pages()},
		Entities: entityStub{items: entities()},
		Edges:    edgeStub{items: edges()},
		Specs:    specStub{},
		Policies: policyStub{},
		Profiles: profileStub{},
		Raw:      rawStub{body: "<h1>Espresso</h1><p>Espresso is a way to make coffee.</p>"},
		LLM:      model,
	}
	if mutate != nil {
		mutate(&deps)
	}
	return appcontent.New(deps)
}

func TestJudgeAuditsTheLivePage(t *testing.T) {
	t.Parallel()

	model := &llmStub{reply: `{"score":1.4,"issues":["The body is thin."," "],"suggestions":[]}`}
	service := newService(model, nil)

	out, err := service.Judge(t.Context(), appcontent.JudgeRequest{PageID: "page-child"})
	if err != nil {
		t.Fatalf("Judge: %v", err)
	}
	if out.PageID != "page-child" || out.Path != "/coffee/espresso/" || out.Tokens != 30 {
		t.Fatalf("response = %+v", out)
	}
	if out.Report.Score == nil || *out.Report.Score != 1 {
		t.Fatalf("score = %v, want the clamp at one", out.Report.Score)
	}
	if len(out.Report.Issues) != 1 || out.Report.Suggestions == nil {
		t.Fatalf("report = %+v", out.Report)
	}

	prompt := model.last.Messages[len(model.last.Messages)-1].Text
	for _, want := range []string{"espresso | Shop", "/coffee/", "Espresso is a way to make coffee"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("the prompt does not carry %q:\n%s", want, prompt)
		}
	}
	if !strings.Contains(model.last.System, "RUBRIC") {
		t.Error("the system prompt does not carry the rubric")
	}
}

func TestJudgePlansTheLinksOfThePageItAudits(t *testing.T) {
	t.Parallel()

	second := pagemap.Page{
		ID: "page-second", SiteID: "site", Path: "/shop/espresso/", WPType: pagemap.WPPage,
		Status: pagemap.StatusPublished, EntityID: pointer("child"), WPID: wpID(13),
	}

	cases := []struct {
		mutate func(*appcontent.Deps)
		name   string
		pageID string
		want   string
	}{
		{
			name:   "a second page of one entity owes its canonical page a link",
			pageID: "page-second",
			mutate: func(d *appcontent.Deps) { d.Pages = pageStub{items: append(pages(), second)} },
			want:   "- /coffee/espresso/ (up",
		},
		{
			name:   "a template with no link rules of its own takes the site policy's",
			pageID: "page-child",
			mutate: func(d *appcontent.Deps) {
				d.Specs = specStub{rules: &template.LinkRules{}}
				d.Policies = policyStub{rules: template.LinkRules{UpDepth: 1, MaxPerTarget: 1}}
			},
			want: "- /coffee/ (up",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			model := &llmStub{reply: `{"score":1}`}
			service := newService(model, tc.mutate)
			if _, err := service.Judge(t.Context(), appcontent.JudgeRequest{PageID: tc.pageID}); err != nil {
				t.Fatalf("Judge: %v", err)
			}

			prompt := model.last.Messages[len(model.last.Messages)-1].Text
			if !strings.Contains(prompt, tc.want) {
				t.Fatalf("the judge was told the required links are:\n%s\nwant one reading %q", prompt, tc.want)
			}
		})
	}
}

func TestJudgeRefusesWhatItCannotAudit(t *testing.T) {
	t.Parallel()

	boom := errors.New(errors.External, "the site is down")

	cases := []struct {
		mutate func(*appcontent.Deps)
		name   string
		pageID string
		want   errors.Code
	}{
		{name: "no page", pageID: "  ", want: errors.Invalid},
		{name: "unknown page", pageID: "ghost", want: errors.NotFound},
		{
			name: "the page is not on the site", pageID: "page-child", want: errors.Invalid,
			mutate: func(d *appcontent.Deps) {
				stripped := pages()
				stripped[1].WPID = nil
				d.Pages = pageStub{items: stripped}
			},
		},
		{
			name: "the page is not mapped", pageID: "page-child", want: errors.Invalid,
			mutate: func(d *appcontent.Deps) {
				stripped := pages()
				stripped[1].EntityID = nil
				d.Pages = pageStub{items: stripped}
			},
		},
		{
			name: "the site cannot be read", pageID: "page-child", want: errors.External,
			mutate: func(d *appcontent.Deps) { d.Raw = rawStub{err: boom} },
		},
		{
			name: "the template cannot be resolved", pageID: "page-child", want: errors.External,
			mutate: func(d *appcontent.Deps) { d.Specs = specStub{err: boom} },
		},
		{
			name: "the entities cannot be listed", pageID: "page-child", want: errors.External,
			mutate: func(d *appcontent.Deps) { d.Entities = entityStub{err: boom} },
		},
		{
			name: "the edges cannot be listed", pageID: "page-child", want: errors.External,
			mutate: func(d *appcontent.Deps) { d.Edges = edgeStub{err: boom} },
		},
		{
			name: "the policy cannot be read", pageID: "page-child", want: errors.External,
			mutate: func(d *appcontent.Deps) { d.Policies = policyStub{err: boom} },
		},
		{
			name: "the entity is gone", pageID: "page-child", want: errors.NotFound,
			mutate: func(d *appcontent.Deps) { d.Entities = entityStub{items: entities()[:1]} },
		},
		{
			name: "no model is bound to the judge", pageID: "page-child", want: errors.External,
			mutate: func(d *appcontent.Deps) { d.Profiles = profileStub{err: boom} },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := newService(&llmStub{reply: `{"score":1}`}, tc.mutate)
			_, err := service.Judge(t.Context(), appcontent.JudgeRequest{PageID: tc.pageID})
			if !errors.IsCode(err, tc.want) {
				t.Fatalf("Judge = %v, want %s", err, tc.want)
			}
		})
	}
}

func TestJudgeCarriesTheModelFailureUp(t *testing.T) {
	t.Parallel()

	service := newService(&llmStub{err: errors.New(errors.RateLimited, "slow down")}, nil)
	if _, err := service.Judge(t.Context(), appcontent.JudgeRequest{PageID: "page-child"}); !errors.IsCode(err, errors.RateLimited) {
		t.Fatalf("Judge = %v, want the model failure", err)
	}
}

func TestJudgeReadsThePageOnDemandWithoutASnippet(t *testing.T) {
	t.Parallel()

	model := &llmStub{reply: `{"score":0.5}`}
	service := newService(model, func(d *appcontent.Deps) {
		bare := pages()
		bare[1].MetaTitle = ""
		bare[1].MetaDescription = ""
		d.Pages = pageStub{items: bare}
	})

	if _, err := service.Judge(t.Context(), appcontent.JudgeRequest{PageID: "page-child"}); err != nil {
		t.Fatalf("Judge: %v", err)
	}
	if strings.Contains(model.last.Messages[0].Text, "SNIPPET") {
		t.Error("a page without a stored snippet must not offer one to the judge")
	}
}

func TestAssessRefusesAnUnreadableBody(t *testing.T) {
	t.Parallel()

	service := newService(&llmStub{reply: "not json"}, nil)
	_, err := service.Assess(t.Context(), appcontent.AssessRequest{
		SiteID: "site", Page: pages()[1], Entity: entities()[1], Body: "<p>hello</p>",
	})
	if !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Assess of an unreadable reply = %v", err)
	}
}
