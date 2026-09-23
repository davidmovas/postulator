package steps_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	appcontent "github.com/davidmovas/postulator/internal/application/content"
	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/graph"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

type entityList struct {
	items []graph.Entity
	err   error
}

func (e entityList) ListBySite(context.Context, string) ([]graph.Entity, error) {
	return e.items, e.err
}

type edgeList struct {
	items []graph.Edge
	err   error
}

func (e edgeList) ListBySite(context.Context, string) ([]graph.Edge, error) {
	return e.items, e.err
}

type pageList struct {
	recorded *pagemap.Page
	items    []pagemap.Page
	err      error
}

func (p pageList) ListBySite(context.Context, string) ([]pagemap.Page, error) {
	return p.items, p.err
}

func (p pageList) Get(_ context.Context, id string) (pagemap.Page, error) {
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

func (p pageList) Update(_ context.Context, page pagemap.Page) error {
	if p.recorded != nil {
		*p.recorded = page
	}
	return p.err
}

func (p pageList) Insert(context.Context, pagemap.Page) error {
	return p.err
}

type pageRecorder struct {
	last pagemap.Page
}

func (p *pageRecorder) ListBySite(context.Context, string) ([]pagemap.Page, error) {
	return nil, nil
}

func (p *pageRecorder) Get(_ context.Context, id string) (pagemap.Page, error) {
	return pagemap.Page{}, errors.New(errors.NotFound, "no such page").WithDetail("pageId", id)
}

func (p *pageRecorder) Update(_ context.Context, page pagemap.Page) error {
	p.last = page
	return nil
}

func (p *pageRecorder) Insert(_ context.Context, page pagemap.Page) error {
	p.last = page
	return nil
}

type siteStub struct {
	record site.Site
	err    error
}

func (s siteStub) Get(context.Context, string) (site.Site, error) {
	if s.err != nil {
		return site.Site{}, s.err
	}
	if s.record.ID == "" {
		return site.Site{
			ID: "site", Name: "Shop", BaseURL: "https://shop.example.com", Username: "editor",
			Status: site.StatusActive,
		}, nil
	}
	return s.record, nil
}

type policyStub struct {
	rules      template.LinkRules
	specs      map[string]template.LinkRules
	err        error
	resolveErr error
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

func (p policyStub) ResolveForPage(_ context.Context, req templates.ResolveForPageRequest) (templates.ResolveForPageResponse, error) {
	if p.resolveErr != nil {
		return templates.ResolveForPageResponse{}, p.resolveErr
	}
	rules, ok := p.specs[req.PageID]
	if !ok {
		rules = template.LinkRules{UpDepth: 2, DownLinks: true, SiblingMinWeight: 0.5, MaxPerTarget: 1}
	}
	return templates.ResolveForPageResponse{
		TemplateID: "template", SiteID: "site", Version: 1,
		Spec: template.TemplateSpec{LinkRules: rules},
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

type llmStub struct {
	reply string
	err   error
}

func (l llmStub) Complete(context.Context, port.Request) (port.Response, error) {
	if l.err != nil {
		return port.Response{}, l.err
	}
	return port.Response{Text: l.reply, Usage: domainllm.Usage{Input: 10, Output: 20, Total: 30}}, nil
}

func (l llmStub) Stream(context.Context, port.Request) (<-chan port.Delta, error) {
	return nil, errors.New(errors.Internal, "the unit stub does not stream")
}

func judgeDeps(client port.Client) steps.Deps {
	deps := unitDeps()
	deps.LLM = client
	deps.Content = appcontent.New(appcontent.Deps{Profiles: deps.Profiles, LLM: client})
	return deps
}

func unitEntities() []graph.Entity {
	return []graph.Entity{
		{
			ID: "parent", SiteID: "site", Name: "Coffee", PrimaryKeyword: "coffee",
			Anchors: []graph.Anchor{{Text: "coffee", Source: graph.AnchorUser, Weight: 1}},
			Kind:    graph.KindTopic, Source: graph.SourceUser, CanonicalPageID: pointer("page-parent"),
		},
		{
			ID: "child", SiteID: "site", Name: "Espresso", PrimaryKeyword: "espresso",
			Anchors: []graph.Anchor{{Text: "espresso", Source: graph.AnchorUser, Weight: 1}},
			Kind:    graph.KindTopic, Source: graph.SourceUser, CanonicalPageID: pointer("page-child"),
		},
	}
}

func unitEdges() []graph.Edge {
	return []graph.Edge{{
		ID: "e1", SiteID: "site", FromEntityID: "child", ToEntityID: "parent",
		Kind: graph.EdgeParent, Weight: 1, Source: graph.SourceUser, Status: graph.StatusApproved,
	}}
}

func unitDeps() steps.Deps {
	return steps.Deps{
		Entities: entityList{items: unitEntities()},
		Edges:    edgeList{items: unitEdges()},
		Pages: pageList{items: []pagemap.Page{
			{ID: "page-parent", SiteID: "site", Path: "/coffee/", WPType: pagemap.WPPage, Status: pagemap.StatusPublished},
			{ID: "page-child", SiteID: "site", Path: "/coffee/espresso/", WPType: pagemap.WPPage, Status: pagemap.StatusPlanned},
		}},
		Sites:    siteStub{},
		Policies: policyStub{},
		Profiles: profileStub{},
		LLM:      llmStub{reply: goodDraft},
		Clock:    clock.NewFake(sqlitetest.Stamp),
	}
}

func pointer(value string) *string {
	return &value
}

func unitContext(t *testing.T, artifacts map[run.ArtifactKind][]byte) *run.StepContext {
	t.Helper()

	stored := make(map[run.ArtifactKind]run.Artifact, len(artifacts))
	for kind, blob := range artifacts {
		stored[kind] = run.Artifact{Kind: kind, Blob: blob, Size: len(blob), Hash: run.HashBlob(blob)}
	}

	return &run.StepContext{
		Run:  run.Run{ID: "run", SiteID: "site", Kind: run.KindGenerate},
		Item: run.Item{ID: "item", RunID: "run", SiteID: "site", TargetID: "page-child"},
		Page: pagemap.Page{
			ID: "page-child", SiteID: "site", Path: "/coffee/espresso/", Title: "Espresso",
			WPType: pagemap.WPPage, Status: pagemap.StatusPlanned, EntityID: pointer("child"),
		},
		Spec:      spec(),
		Params:    map[string]any{},
		Artifacts: stored,
		Check:     run.NewCheckpoint(),
	}
}

func linkContextBlob(t *testing.T, deps steps.Deps) []byte {
	t.Helper()

	sc := unitContext(t, nil)
	result, err := steps.ResolveContext(deps).Run(t.Context(), sc)
	if err != nil {
		t.Fatalf("ResolveContext: %v", err)
	}
	return result.Artifacts[0].Blob
}

func TestResolveContextReportsWhatItCannotRead(t *testing.T) {
	t.Parallel()

	boom := errors.New(errors.External, "the database is busy")

	cases := []struct {
		name    string
		deps    func(steps.Deps) steps.Deps
		context func(*run.StepContext)
		want    errors.Code
	}{
		{
			name:    "the page is not mapped",
			context: func(sc *run.StepContext) { sc.Page.EntityID = nil },
			want:    errors.Invalid,
		},
		{
			name:    "the entity is gone",
			context: func(sc *run.StepContext) { sc.Page.EntityID = pointer("ghost") },
			want:    errors.NotFound,
		},
		{
			name: "the entities cannot be listed",
			deps: func(d steps.Deps) steps.Deps { d.Entities = entityList{err: boom}; return d },
			want: errors.External,
		},
		{
			name: "the edges cannot be listed",
			deps: func(d steps.Deps) steps.Deps { d.Edges = edgeList{err: boom}; return d },
			want: errors.External,
		},
		{
			name: "the pages cannot be listed",
			deps: func(d steps.Deps) steps.Deps { d.Pages = pageList{err: boom}; return d },
			want: errors.External,
		},
		{
			name: "the policy cannot be read",
			deps: func(d steps.Deps) steps.Deps { d.Policies = policyStub{err: boom}; return d },
			want: errors.External,
		},
		{
			name: "the graph does not hold together",
			deps: func(d steps.Deps) steps.Deps {
				d.Edges = edgeList{items: []graph.Edge{{
					ID: "e1", SiteID: "site", FromEntityID: "child", ToEntityID: "ghost",
					Kind: graph.EdgeParent, Weight: 1, Source: graph.SourceUser, Status: graph.StatusApproved,
				}}}
				return d
			},
			want: errors.Invalid,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps := unitDeps()
			if tc.deps != nil {
				deps = tc.deps(deps)
			}
			sc := unitContext(t, nil)
			if tc.context != nil {
				tc.context(sc)
			}

			if _, err := steps.ResolveContext(deps).Run(t.Context(), sc); !errors.IsCode(err, tc.want) {
				t.Fatalf("ResolveContext = %v, want %s", err, tc.want)
			}
		})
	}
}

func TestResolveContextProducesTheUpTargets(t *testing.T) {
	t.Parallel()

	deps := unitDeps()
	result, err := steps.ResolveContext(deps).Run(t.Context(), unitContext(t, nil))
	if err != nil {
		t.Fatalf("ResolveContext: %v", err)
	}
	if len(result.Artifacts) != 1 || result.Artifacts[0].Kind != run.ArtifactLinkContext {
		t.Fatalf("Artifacts = %+v", result.Artifacts)
	}

	var lc content.LinkContext
	if err = json.Unmarshal(result.Artifacts[0].Blob, &lc); err != nil {
		t.Fatalf("decode the link context: %v", err)
	}
	if len(lc.Targets) != 1 || lc.Targets[0].URL != "/coffee/" || !lc.Targets[0].Required {
		t.Fatalf("targets = %+v", lc.Targets)
	}
	if result.Message == "" {
		t.Error("the step reports what it did")
	}
}

func TestGenerateBodyReportsWhatItCannotDo(t *testing.T) {
	t.Parallel()

	blob := linkContextBlob(t, unitDeps())

	cases := []struct {
		name      string
		deps      func(steps.Deps) steps.Deps
		artifacts map[run.ArtifactKind][]byte
		want      errors.Code
	}{
		{name: "no link context", artifacts: map[run.ArtifactKind][]byte{}, want: errors.Invalid},
		{
			name:      "the link context is corrupt",
			artifacts: map[run.ArtifactKind][]byte{run.ArtifactLinkContext: []byte("not json")},
			want:      errors.Internal,
		},
		{
			name: "the profile cannot be resolved",
			deps: func(d steps.Deps) steps.Deps {
				d.Profiles = profileStub{err: errors.New(errors.NotFound, "no model")}
				return d
			},
			artifacts: map[run.ArtifactKind][]byte{run.ArtifactLinkContext: blob},
			want:      errors.NotFound,
		},
		{
			name: "the model fails",
			deps: func(d steps.Deps) steps.Deps {
				d.LLM = llmStub{err: errors.New(errors.RateLimited, "slow down")}
				return d
			},
			artifacts: map[run.ArtifactKind][]byte{run.ArtifactLinkContext: blob},
			want:      errors.RateLimited,
		},
		{
			name:      "the model does not answer with json",
			deps:      func(d steps.Deps) steps.Deps { d.LLM = llmStub{reply: "sure thing"}; return d },
			artifacts: map[run.ArtifactKind][]byte{run.ArtifactLinkContext: blob},
			want:      errors.Invalid,
		},
		{
			name: "the draft is incomplete",
			deps: func(d steps.Deps) steps.Deps {
				d.LLM = llmStub{reply: `{"title":"t","h1":"","sections":[]}`}
				return d
			},
			artifacts: map[run.ArtifactKind][]byte{run.ArtifactLinkContext: blob},
			want:      errors.Invalid,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps := unitDeps()
			if tc.deps != nil {
				deps = tc.deps(deps)
			}
			if _, err := steps.GenerateBody(deps).Run(t.Context(), unitContext(t, tc.artifacts)); !errors.IsCode(err, tc.want) {
				t.Fatalf("GenerateBody = %v, want %s", err, tc.want)
			}
		})
	}
}

func TestGenerateBodyAssemblesAndCountsTokens(t *testing.T) {
	t.Parallel()

	deps := unitDeps()
	sc := unitContext(t, map[run.ArtifactKind][]byte{run.ArtifactLinkContext: linkContextBlob(t, deps)})

	result, err := steps.GenerateBody(deps).Run(t.Context(), sc)
	if err != nil {
		t.Fatalf("GenerateBody: %v", err)
	}
	if len(result.Artifacts) != 2 || result.Tokens != 30 {
		t.Fatalf("result = %+v", result)
	}
	if result.Artifacts[0].Kind != run.ArtifactDraft || result.Artifacts[1].Kind != run.ArtifactBodyHTML {
		t.Fatalf("artifacts = %+v", result.Artifacts)
	}
	if body := string(result.Artifacts[1].Blob); body == "" || body[:4] != "<h1>" {
		t.Fatalf("body = %q", body)
	}
}

func TestGenerateBodyNamesTheChildrenWhenTheRulesAskFor(t *testing.T) {
	t.Parallel()

	down, err := json.Marshal(content.LinkContext{
		PageID: "page-child", PageURL: "/coffee/espresso/", EntityID: "child",
		Targets: []content.LinkTarget{
			{
				EntityID: "grind", PageID: "page-grind", URL: "/coffee/espresso/grind/",
				Anchors: []string{"grinding for espresso"}, Relation: content.RelationDown, Weight: 1, Depth: 1,
			},
			{
				EntityID: "parent", PageID: "page-parent", URL: "/coffee/",
				Anchors: []string{"coffee"}, Relation: content.RelationUp, Required: true, Weight: 1, Depth: 1,
			},
		},
	})
	if err != nil {
		t.Fatalf("encode the link context: %v", err)
	}

	cases := []struct {
		name    string
		section bool
		want    bool
	}{
		{name: "the rules ask for a children section", section: true, want: true},
		{name: "the rules do not", section: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps := unitDeps()
			recorder := &promptRecorder{reply: goodDraft}
			deps.LLM = recorder
			sc := unitContext(t, map[run.ArtifactKind][]byte{run.ArtifactLinkContext: down})
			sc.Spec.LinkRules.ChildrenSection = tc.section

			if _, runErr := steps.GenerateBody(deps).Run(t.Context(), sc); runErr != nil {
				t.Fatalf("GenerateBody: %v", runErr)
			}
			if strings.Contains(recorder.last, "CHILDREN") != tc.want {
				t.Fatalf("the prompt asks for a children section = %t, want %t:\n%s",
					!tc.want, tc.want, recorder.last)
			}
			if tc.want && !strings.Contains(recorder.last, "grinding for espresso") {
				t.Fatalf("the prompt does not name the child:\n%s", recorder.last)
			}
		})
	}
}

func TestInsertLinksAndValidateReadTheirArtifacts(t *testing.T) {
	t.Parallel()

	deps := unitDeps()
	blob := linkContextBlob(t, deps)
	body := []byte("<h1>Espresso</h1><h2>About</h2><p>Espresso is a kind of coffee.</p>")

	inserted, err := steps.InsertLinks(deps).Run(t.Context(),
		unitContext(t, map[run.ArtifactKind][]byte{run.ArtifactLinkContext: blob, run.ArtifactBodyHTML: body}))
	if err != nil {
		t.Fatalf("InsertLinks: %v", err)
	}
	if linked := string(inserted.Artifacts[0].Blob); !strings.Contains(linked, `<a href="/coffee/">coffee</a>`) {
		t.Fatalf("the body was not linked: %q", linked)
	}

	sc := unitContext(t, map[run.ArtifactKind][]byte{
		run.ArtifactLinkContext: blob,
		run.ArtifactBodyHTML:    inserted.Artifacts[0].Blob,
	})
	sc.Check = inserted.Checkpoint

	report, err := steps.Validate(deps).Run(t.Context(), sc)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(report.Artifacts) != 1 || report.Artifacts[0].Kind != run.ArtifactValidationReport {
		t.Fatalf("artifacts = %+v", report.Artifacts)
	}

	var decoded steps.ValidationReport
	if err = json.Unmarshal(report.Artifacts[0].Blob, &decoded); err != nil {
		t.Fatalf("decode the report: %v", err)
	}
	if decoded.Score != 1 || len(decoded.Links.Placed) != 1 {
		t.Fatalf("report = %+v", decoded)
	}
}

func TestValidateReportsATitleWithoutTheKeyword(t *testing.T) {
	t.Parallel()

	deps := unitDeps()
	blob := linkContextBlob(t, deps)
	body := []byte("<h1>Espresso</h1><h2>About</h2><p>Espresso is a kind of coffee.</p>")

	draft, err := json.Marshal(content.ContentDraft{
		Title: "Our drinks range", H1: "Espresso",
		Sections: []content.DraftSection{{Heading: "About", HTML: "<p>Espresso.</p>"}},
	})
	if err != nil {
		t.Fatalf("encode the draft: %v", err)
	}

	sc := unitContext(t, map[run.ArtifactKind][]byte{
		run.ArtifactLinkContext: blob, run.ArtifactBodyHTML: body, run.ArtifactDraft: draft,
	})
	sc.Spec.KeywordRules.PrimaryInTitle = true
	sc.Params = map[string]any{steps.ParamAllowErrors: true}

	result, err := steps.Validate(deps).Run(t.Context(), sc)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	var decoded steps.ValidationReport
	if unmarshalErr := json.Unmarshal(result.Artifacts[0].Blob, &decoded); unmarshalErr != nil {
		t.Fatalf("decode the report: %v", unmarshalErr)
	}
	if !hasFinding(decoded, steps.CodeTitleMissingKeyword) {
		t.Fatalf("structure findings = %+v", decoded.Structure.Items)
	}

	sc.Params = map[string]any{}
	if _, err = steps.Validate(deps).Run(t.Context(), sc); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Validate without allowErrors = %v", err)
	}
}

func hasFinding(report steps.ValidationReport, code string) bool {
	for _, item := range report.Structure.Items {
		if item.Code == code {
			return true
		}
	}
	return false
}

func TestRepairLinksRefusesASentenceWithoutThePhrase(t *testing.T) {
	t.Parallel()

	deps := unitDeps()
	deps.LLM = llmStub{reply: `{"sentence":"A sentence with no anchor at all."}`}
	blob := linkContextBlob(t, deps)

	sc := unitContext(t, map[run.ArtifactKind][]byte{
		run.ArtifactLinkContext: blob,
		run.ArtifactBodyHTML:    []byte("<h1>Espresso</h1><p>Nothing to match here.</p>"),
	})

	if _, err := steps.RepairLinks(deps).Run(t.Context(), sc); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("RepairLinks = %v, want an invalid error", err)
	}

	deps.LLM = llmStub{reply: `{"sentence":""}`}
	if _, err := steps.RepairLinks(deps).Run(t.Context(), sc); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("RepairLinks with an empty sentence = %v", err)
	}
}

func TestRepairLinksNeedsAParagraph(t *testing.T) {
	t.Parallel()

	deps := unitDeps()
	deps.LLM = llmStub{reply: `{"sentence":"It belongs to our coffee range."}`}
	blob := linkContextBlob(t, deps)

	sc := unitContext(t, map[run.ArtifactKind][]byte{
		run.ArtifactLinkContext: blob,
		run.ArtifactBodyHTML:    []byte("<h1>Espresso</h1>"),
	})
	if _, err := steps.RepairLinks(deps).Run(t.Context(), sc); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("RepairLinks without a paragraph = %v", err)
	}
}

func TestRepairLinksHonoursTheIterationParameter(t *testing.T) {
	t.Parallel()

	deps := unitDeps()
	deps.LLM = llmStub{reply: `{"sentence":"It belongs to our coffee range."}`}
	blob := linkContextBlob(t, deps)

	cases := []struct {
		name  string
		param any
	}{
		{name: "the default"},
		{name: "a number from json", param: float64(1)},
		{name: "a value of the wrong type", param: "two"},
		{name: "a value above the ceiling", param: float64(99)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sc := unitContext(t, map[run.ArtifactKind][]byte{
				run.ArtifactLinkContext: blob,
				run.ArtifactBodyHTML:    []byte("<h1>Espresso</h1><p>A shot of the good stuff.</p>"),
			})
			if tc.param != nil {
				sc.Params = map[string]any{steps.ParamIterations: tc.param}
			}
			sc.Spec.LinkRules.ParentLinkWithinParagraphs = 1

			result, err := steps.RepairLinks(deps).Run(t.Context(), sc)
			if err != nil {
				t.Fatalf("RepairLinks: %v", err)
			}
			if len(result.Artifacts) != 1 || result.Tokens == 0 {
				t.Fatalf("result = %+v", result)
			}
		})
	}
}

func TestMaxTokensFallsBackToTheTemplateLength(t *testing.T) {
	t.Parallel()

	deps := unitDeps()
	blob := linkContextBlob(t, deps)

	cases := []struct {
		name string
		spec template.TemplateSpec
	}{
		{name: "sections carry the target", spec: spec()},
		{
			name: "only a length ceiling",
			spec: template.TemplateSpec{Length: template.Length{Max: 400}, LinkRules: spec().LinkRules},
		},
		{name: "nothing at all", spec: template.TemplateSpec{LinkRules: spec().LinkRules}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sc := unitContext(t, map[run.ArtifactKind][]byte{run.ArtifactLinkContext: blob})
			sc.Spec = tc.spec
			if _, err := steps.GenerateBody(deps).Run(t.Context(), sc); err != nil {
				t.Fatalf("GenerateBody: %v", err)
			}
		})
	}
}

type promptRecorder struct {
	reply string
	last  string
	err   error
}

func (r *promptRecorder) Complete(_ context.Context, req port.Request) (port.Response, error) {
	if r.err != nil {
		return port.Response{}, r.err
	}
	r.last = req.System + "\n" + req.Messages[len(req.Messages)-1].Text
	return port.Response{Text: r.reply, Usage: domainllm.Usage{Input: 1, Output: 2, Total: 3}}, nil
}

func (r *promptRecorder) Stream(context.Context, port.Request) (<-chan port.Delta, error) {
	return nil, errors.New(errors.Internal, "the unit stub does not stream")
}
