package steps_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

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
			name:      "the model does not answer with json, which the engine may try again",
			deps:      func(d steps.Deps) steps.Deps { d.LLM = llmStub{reply: "sure thing"}; return d },
			artifacts: map[run.ArtifactKind][]byte{run.ArtifactLinkContext: blob},
			want:      errors.External,
		},
		{
			name: "the draft is incomplete, which the engine may try again",
			deps: func(d steps.Deps) steps.Deps {
				d.LLM = llmStub{reply: `{"title":"t","h1":"","sections":[]}`}
				return d
			},
			artifacts: map[run.ArtifactKind][]byte{run.ArtifactLinkContext: blob},
			want:      errors.External,
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
		name   string
		rules  template.LinkRules
		policy template.LinkRules
		want   bool
	}{
		{name: "the rules ask for a children section", rules: template.LinkRules{UpDepth: 1, ChildrenSection: true}, want: true},
		{name: "the rules do not", rules: template.LinkRules{UpDepth: 1}},
		{
			name:   "the template inherits a site policy that asks for one",
			policy: template.LinkRules{UpDepth: 1, DownLinks: true, ChildrenSection: true},
			want:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps := unitDeps()
			recorder := &promptRecorder{reply: goodDraft}
			deps.LLM = recorder
			deps.Policies = policyStub{rules: tc.policy}
			sc := unitContext(t, map[run.ArtifactKind][]byte{run.ArtifactLinkContext: down})
			sc.Spec.LinkRules = tc.rules

			if _, runErr := steps.GenerateBody(deps).Run(t.Context(), sc); runErr != nil {
				t.Fatalf("GenerateBody: %v", runErr)
			}
			if strings.Contains(recorder.last, "CHILDREN") != tc.want {
				t.Fatalf("the prompt asks for a children section = %t, want %t:\n%s",
					!tc.want, tc.want, recorder.last)
			}
			if !strings.Contains(recorder.last, "grinding for espresso") {
				t.Fatalf("the prompt does not name the child, which the page owes a link:\n%s", recorder.last)
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

func TestValidateCarriesTheDraftAndRepairFindingsAndLetsThePlanWin(t *testing.T) {
	t.Parallel()

	deps := unitDeps()
	blob := linkContextBlob(t, deps)
	body := []byte(`<h1>A guide</h1><h2>About</h2><p>Espresso is a kind of <a href="/coffee/">coffee</a>.</p>`)

	draft, err := json.Marshal(content.ContentDraft{
		Title: "A guide", H1: "A guide",
		Sections: []content.DraftSection{{Heading: "About", HTML: "<p>Espresso.</p>"}},
		Findings: []content.Finding{
			{Severity: content.SeverityWarn, Code: content.CodePlanTitleLacksKeyword, Message: "the plan's title lacks it"},
			{Severity: content.SeverityWarn, Code: content.CodePlanH1LacksKeyword, Message: "the plan's h1 lacks it"},
		},
	})
	if err != nil {
		t.Fatalf("encode the draft: %v", err)
	}

	sc := unitContext(t, map[run.ArtifactKind][]byte{
		run.ArtifactLinkContext: blob, run.ArtifactBodyHTML: body, run.ArtifactDraft: draft,
	})
	sc.Spec.KeywordRules.PrimaryInH1 = true
	if err = run.Set(sc.Check, steps.CheckpointRepairs, []content.Finding{{
		Severity: content.SeverityWarn, Code: steps.CodePhraseTemplated, Message: "a plain sentence was added",
	}}); err != nil {
		t.Fatalf("Set: %v", err)
	}

	result, err := steps.Validate(deps).Run(t.Context(), sc)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if result.Next != "" && result.Next != run.TransitionContinue {
		t.Fatalf("result = %+v, want the item to go on", result)
	}

	var decoded steps.ValidationReport
	if unmarshalErr := json.Unmarshal(result.Artifacts[0].Blob, &decoded); unmarshalErr != nil {
		t.Fatalf("decode the report: %v", unmarshalErr)
	}
	for _, code := range []string{
		content.CodePlanTitleLacksKeyword, content.CodePlanH1LacksKeyword, steps.CodePhraseTemplated, content.CodePrimaryMissingInH1,
	} {
		if !hasFinding(decoded, code) {
			t.Fatalf("the report lacks %s: %+v", code, decoded.Structure.Items)
		}
	}
	if decoded.Structure.HasErrors() {
		t.Fatalf("the planned h1 was graded as an error: %+v", decoded.Structure.Items)
	}
	if decoded.Score >= 1 {
		t.Fatalf("score = %v, want the warnings to cost something", decoded.Score)
	}
}

func TestValidateHoldsAPageWithErrorsUnlessTheyAreAllowedOrAccepted(t *testing.T) {
	t.Parallel()

	deps := unitDeps()
	blob := linkContextBlob(t, deps)
	body := []byte("<h1>Espresso</h1><h2>About</h2><p>Espresso is a kind of coffee.</p>")

	cases := []struct {
		name     string
		params   map[string]any
		accepted bool
		held     bool
	}{
		{name: "the missing parent link holds the page for a human", held: true},
		{name: "allowErrors lets it through", params: map[string]any{steps.ParamAllowErrors: true}},
		{name: "an accepted validation lets it through", accepted: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sc := unitContext(t, map[run.ArtifactKind][]byte{run.ArtifactLinkContext: blob, run.ArtifactBodyHTML: body})
			sc.Item.CurrentStep = steps.NameValidate
			if tc.params != nil {
				sc.Params = tc.params
			}
			if tc.accepted {
				if err := run.Set(sc.Check, run.CheckpointAccept, steps.NameValidate); err != nil {
					t.Fatalf("Set: %v", err)
				}
			}

			result, err := steps.Validate(deps).Run(t.Context(), sc)
			if err != nil {
				t.Fatalf("Validate: %v", err)
			}
			if len(result.Artifacts) != 1 {
				t.Fatalf("the report was not written: %+v", result)
			}

			var decoded steps.ValidationReport
			if unmarshalErr := json.Unmarshal(result.Artifacts[0].Blob, &decoded); unmarshalErr != nil {
				t.Fatalf("decode the report: %v", unmarshalErr)
			}
			if !decoded.Compliance.HasErrors() {
				t.Fatalf("compliance findings = %+v, want the missing link", decoded.Compliance.Items)
			}

			if !tc.held {
				if result.Next == run.TransitionPause {
					t.Fatalf("the page was held: %+v", result)
				}
				return
			}
			if result.Next != run.TransitionPause || result.Reason != run.PauseNeedsHuman {
				t.Fatalf("result = %+v, want a pause for a human", result)
			}
			if !strings.Contains(result.Message, "/coffee/") || !strings.Contains(result.Message, "ccept") {
				t.Fatalf("the note %q neither names the finding nor says what to do", result.Message)
			}
		})
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

type callCounter struct {
	reply string
	err   error
	calls int
}

func (c *callCounter) Complete(_ context.Context, _ port.Request) (port.Response, error) {
	c.calls++
	if c.err != nil {
		return port.Response{}, c.err
	}
	return port.Response{Text: c.reply, Usage: domainllm.Usage{Input: 10, Output: 20, Total: 30}}, nil
}

func (c *callCounter) Stream(context.Context, port.Request) (<-chan port.Delta, error) {
	return nil, errors.New(errors.Internal, "the unit stub does not stream")
}

func repairsOf(t *testing.T, result run.Result) []content.Finding {
	t.Helper()

	findings, _, err := run.Get[[]content.Finding](result.Checkpoint, steps.CheckpointRepairs)
	if err != nil {
		t.Fatalf("read the repairs: %v", err)
	}
	return findings
}

func TestRepairLinksFallsBackToAPlainSentenceWhenTheLinkerKeepsMissingThePhrase(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		client *callCounter
	}{
		{name: "the linker answers without the phrase", client: &callCounter{reply: `{"sentence":"A sentence with no anchor at all."}`}},
		{name: "the linker answers with nothing", client: &callCounter{reply: `{"sentence":""}`}},
		{name: "the provider is down", client: &callCounter{err: errors.New(errors.External, "upstream is down")}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps := unitDeps()
			deps.LLM = tc.client
			blob := linkContextBlob(t, deps)
			sc := unitContext(t, map[run.ArtifactKind][]byte{
				run.ArtifactLinkContext: blob,
				run.ArtifactBodyHTML:    []byte("<h1>Espresso</h1><p>Nothing to match here.</p>"),
			})

			result, err := steps.RepairLinks(deps).Run(t.Context(), sc)
			if err != nil {
				t.Fatalf("RepairLinks: %v", err)
			}
			if tc.client.calls != 4 {
				t.Fatalf("the linker was called %d times, want two tries for each of the two phrases", tc.client.calls)
			}

			body := string(result.Artifacts[0].Blob)
			want := `<h1>Espresso</h1><p>Nothing to match here. This page is about espresso. Read more about <a href="/coffee/">coffee</a>.</p>`
			if body != want {
				t.Fatalf("body =\n%s\nwant\n%s", body, want)
			}

			repairs := repairsOf(t, result)
			if len(repairs) != 2 {
				t.Fatalf("repairs = %+v, want one templated finding per phrase", repairs)
			}
			for _, finding := range repairs {
				if finding.Code != steps.CodePhraseTemplated || finding.Severity != content.SeverityWarn {
					t.Fatalf("finding = %+v", finding)
				}
			}
			if !strings.Contains(result.Message, "2") {
				t.Fatalf("message = %q", result.Message)
			}
		})
	}
}

func owingContext(t *testing.T) []byte {
	t.Helper()

	blob, err := json.Marshal(content.LinkContext{
		PageID: "page-child", PageURL: "/coffee/espresso/", EntityID: "child",
		Targets: []content.LinkTarget{
			{
				EntityID: "parent", PageID: "page-parent", URL: "/coffee/", Anchors: []string{"coffee"},
				Relation: content.RelationUp, Required: true, Weight: 1, Depth: 1,
			},
			{
				EntityID: "ristretto", PageID: "page-ristretto", URL: "/coffee/espresso/ristretto/",
				Anchors: []string{"ristretto"}, Relation: content.RelationDown, Weight: 1, Depth: 1,
			},
			{
				EntityID: "filter", PageID: "page-filter", URL: "/coffee/filter/", Anchors: []string{"filter coffee"},
				Relation: content.RelationSibling, Weight: 0.8, Depth: 1,
			},
		},
	})
	if err != nil {
		t.Fatalf("encode the link context: %v", err)
	}
	return blob
}

func TestRepairLinksWritesAPhraseForEveryOwedLinkWhereItMayGo(t *testing.T) {
	t.Parallel()

	deps := unitDeps()
	client := &callCounter{reply: `{"sentence":"A sentence with no anchor at all."}`}
	deps.LLM = client
	sc := unitContext(t, map[run.ArtifactKind][]byte{
		run.ArtifactLinkContext: owingContext(t),
		run.ArtifactBodyHTML:    []byte("<h1>Espresso</h1><p>Our espresso starts with the coffee we roast.</p><p>Last words.</p>"),
	})

	result, err := steps.RepairLinks(deps).Run(t.Context(), sc)
	if err != nil {
		t.Fatalf("RepairLinks: %v", err)
	}
	if client.calls != 4 {
		t.Fatalf("the linker was called %d times, want two tries for the child and two for the sibling", client.calls)
	}

	body := string(result.Artifacts[0].Blob)
	want := `<h1>Espresso</h1><p>Our espresso starts with the <a href="/coffee/">coffee</a> we roast.</p>` +
		`<p>Last words. Read more about <a href="/coffee/espresso/ristretto/">ristretto</a>. ` +
		`Read more about <a href="/coffee/filter/">filter coffee</a>.</p>`
	if body != want {
		t.Fatalf("body =\n%s\nwant\n%s", body, want)
	}

	repairs := repairsOf(t, result)
	if len(repairs) != 2 || repairs[0].Details["url"] != "/coffee/espresso/ristretto/" ||
		repairs[1].Details["url"] != "/coffee/filter/" {
		t.Fatalf("repairs = %+v, want the child and the sibling named", repairs)
	}
	if !strings.Contains(result.Message, "2 owed phrases") {
		t.Fatalf("message = %q", result.Message)
	}
}

func TestRepairLinksLeavesALinkTheBudgetCannotHold(t *testing.T) {
	t.Parallel()

	deps := unitDeps()
	client := &callCounter{reply: `{"sentence":"A sentence with no anchor at all."}`}
	deps.LLM = client
	sc := unitContext(t, map[run.ArtifactKind][]byte{
		run.ArtifactLinkContext: owingContext(t),
		run.ArtifactBodyHTML:    []byte("<h1>Espresso</h1><p>Our espresso starts with the coffee we roast.</p>"),
	})
	sc.Spec.LinkRules.MaxLinks = 1

	result, err := steps.RepairLinks(deps).Run(t.Context(), sc)
	if err != nil {
		t.Fatalf("RepairLinks: %v", err)
	}
	if client.calls != 0 || len(repairsOf(t, result)) != 0 {
		t.Fatalf("the linker was called %d times with repairs %+v, want nothing written past the budget",
			client.calls, repairsOf(t, result))
	}
	if strings.Contains(string(result.Artifacts[0].Blob), "ristretto") {
		t.Fatalf("the body names the child the budget cannot link: %s", result.Artifacts[0].Blob)
	}
}

func TestRepairLinksPutsTheKeywordInTheLeadAndTheAnchorWhereTheParentLinkMayGo(t *testing.T) {
	t.Parallel()

	deps := unitDeps()
	client := &callCounter{reply: `{"sentence":"Espresso belongs to our coffee range."}`}
	deps.LLM = client
	blob := linkContextBlob(t, deps)

	sc := unitContext(t, map[run.ArtifactKind][]byte{
		run.ArtifactLinkContext: blob,
		run.ArtifactBodyHTML:    []byte("<h1>Espresso</h1><p>A shot of the good stuff.</p><p>Second thoughts.</p>"),
	})
	sc.Spec.LinkRules.ParentLinkWithinParagraphs = 1

	result, err := steps.RepairLinks(deps).Run(t.Context(), sc)
	if err != nil {
		t.Fatalf("RepairLinks: %v", err)
	}
	if client.calls != 1 {
		t.Fatalf("the linker was called %d times, want one sentence to settle both phrases", client.calls)
	}
	body := string(result.Artifacts[0].Blob)
	if !strings.HasPrefix(body, `<h1>Espresso</h1><p>A shot of the good stuff. Espresso belongs to our <a href="/coffee/">coffee</a> range.</p>`) {
		t.Fatalf("the sentence did not land in the first paragraph:\n%s", body)
	}
	if repairs := repairsOf(t, result); len(repairs) != 0 {
		t.Fatalf("repairs = %+v, want none", repairs)
	}
	if result.Tokens != 30 {
		t.Fatalf("tokens = %d", result.Tokens)
	}
}

func TestRepairLinksOpensABodyWithoutAParagraph(t *testing.T) {
	t.Parallel()

	deps := unitDeps()
	deps.LLM = llmStub{reply: `{"sentence":"Espresso is the strongest coffee we pull."}`}
	blob := linkContextBlob(t, deps)

	sc := unitContext(t, map[run.ArtifactKind][]byte{
		run.ArtifactLinkContext: blob,
		run.ArtifactBodyHTML:    []byte("<h1>Espresso</h1><h2>About</h2><ul><li>Short.</li></ul>"),
	})

	result, err := steps.RepairLinks(deps).Run(t.Context(), sc)
	if err != nil {
		t.Fatalf("RepairLinks: %v", err)
	}
	body := string(result.Artifacts[0].Blob)
	if !strings.HasPrefix(body, `<h1>Espresso</h1><p>Espresso is the strongest <a href="/coffee/">coffee</a> we pull.</p><h2>About</h2>`) {
		t.Fatalf("the body did not get an opening paragraph:\n%s", body)
	}
}

func TestRepairLinksStopsWhenTheRunStops(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		ctx  func() context.Context
		llm  port.Client
	}{
		{
			name: "the provider reports the cancellation",
			ctx:  context.Background,
			llm:  llmStub{err: errors.New(errors.Cancelled, "the call was cancelled")},
		},
		{
			name: "the step context is already done",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			llm: llmStub{reply: `{"sentence":"Espresso belongs to our coffee range."}`},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps := unitDeps()
			deps.LLM = tc.llm
			blob := linkContextBlob(t, deps)
			sc := unitContext(t, map[run.ArtifactKind][]byte{
				run.ArtifactLinkContext: blob,
				run.ArtifactBodyHTML:    []byte("<h1>Espresso</h1><p>Nothing to match here.</p>"),
			})

			if _, err := steps.RepairLinks(deps).Run(tc.ctx(), sc); !errors.IsCode(err, errors.Cancelled) {
				t.Fatalf("RepairLinks = %v, want the cancellation passed on", err)
			}
		})
	}
}

func TestRepairLinksHonoursTheIterationParameter(t *testing.T) {
	t.Parallel()

	deps := unitDeps()
	blob := linkContextBlob(t, deps)

	cases := []struct {
		name  string
		param any
		calls int
	}{
		{name: "the default", calls: 4},
		{name: "a number from json", param: float64(1), calls: 2},
		{name: "a value of the wrong type", param: "two", calls: 4},
		{name: "a value above the ceiling", param: float64(99), calls: 8},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := &callCounter{reply: `{"sentence":"A sentence that never carries the phrase."}`}
			local := unitDeps()
			local.LLM = client
			sc := unitContext(t, map[run.ArtifactKind][]byte{
				run.ArtifactLinkContext: blob,
				run.ArtifactBodyHTML:    []byte("<h1>Espresso</h1><p>A shot of the good stuff.</p>"),
			})
			if tc.param != nil {
				sc.Params = map[string]any{steps.ParamIterations: tc.param}
			}
			sc.Spec.LinkRules.ParentLinkWithinParagraphs = 1

			result, err := steps.RepairLinks(local).Run(t.Context(), sc)
			if err != nil {
				t.Fatalf("RepairLinks: %v", err)
			}
			if len(result.Artifacts) != 1 || result.Tokens != 30*tc.calls {
				t.Fatalf("result = %+v", result)
			}
			if client.calls != tc.calls {
				t.Fatalf("the linker was called %d times, want %d", client.calls, tc.calls)
			}
		})
	}
}

type ceilingRecorder struct {
	reply    string
	ceilings []int
}

func (r *ceilingRecorder) Complete(_ context.Context, req port.Request) (port.Response, error) {
	r.ceilings = append(r.ceilings, req.MaxTokens)
	return port.Response{Text: r.reply, Usage: domainllm.Usage{Input: 1, Output: 2, Total: 3}}, nil
}

func (r *ceilingRecorder) Stream(context.Context, port.Request) (<-chan port.Delta, error) {
	return nil, errors.New(errors.Internal, "the unit stub does not stream")
}

func TestWriterCeilingGrowsWithTheTemplateAndTheAttempt(t *testing.T) {
	t.Parallel()

	deps := unitDeps()
	blob := linkContextBlob(t, deps)
	recorder := &ceilingRecorder{reply: goodDraft}
	deps.LLM = recorder

	long := spec()
	long.Sections = []template.Section{{Heading: "Overview", TargetWords: 1800, Required: true}}
	short := template.TemplateSpec{Length: template.Length{Max: 200}, LinkRules: spec().LinkRules}

	for _, tc := range []struct {
		name     string
		spec     template.TemplateSpec
		attempts int
		want     int
	}{
		{name: "a long template asks for three tokens a word and a thousand more", spec: long, attempts: 0, want: 1800*3 + 1024},
		{name: "a short template still gets room to answer", spec: short, attempts: 0, want: 4096},
		{name: "the second attempt doubles the room", spec: long, attempts: 1, want: (1800*3 + 1024) * 2},
		{name: "the room stops doubling after three attempts", spec: long, attempts: 7, want: (1800*3 + 1024) * 8},
	} {
		sc := unitContext(t, map[run.ArtifactKind][]byte{run.ArtifactLinkContext: blob})
		sc.Spec = tc.spec
		sc.Item.Attempts = tc.attempts
		if _, err := steps.GenerateBody(deps).Run(t.Context(), sc); err != nil {
			t.Fatalf("%s: GenerateBody: %v", tc.name, err)
		}
		if got := recorder.ceilings[len(recorder.ceilings)-1]; got != tc.want {
			t.Errorf("%s: ceiling = %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestTheModelStepsDeclareTheirOwnTimeouts(t *testing.T) {
	t.Parallel()

	deps := unitDeps()
	if got := steps.GenerateBody(deps).Timeout; got != 15*time.Minute {
		t.Errorf("the writer step runs under %s, want fifteen minutes", got)
	}
	for _, def := range []run.StepDef{steps.GenerateMeta(deps), steps.RepairLinks(deps)} {
		if def.Timeout != 3*time.Minute {
			t.Errorf("%s runs under %s, want three minutes", def.Name, def.Timeout)
		}
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

func TestTheWriterPromptCarriesTheBrief(t *testing.T) {
	t.Parallel()

	deps := unitDeps()
	recorder := &promptRecorder{reply: goodDraft}
	deps.LLM = recorder
	blob := linkContextBlob(t, deps)

	sc := unitContext(t, map[run.ArtifactKind][]byte{run.ArtifactLinkContext: blob})
	sc.Spec.LinkRules.ParentLinkWithinParagraphs = 2
	sc.Spec.Sections[0].KeywordRules.PrimaryInHeading = true

	if _, err := steps.GenerateBody(deps).Run(t.Context(), sc); err != nil {
		t.Fatalf("GenerateBody: %v", err)
	}
	for _, want := range []string{
		"Title: Espresso (planned",
		"H1: write one that carries the primary keyword",
		"1. About (required), about 120 words: Explain the topic",
		"2. Brewing, about 120 words",
		"exactly as written",
		"- espresso, in the first paragraph of section 1",
		"- coffee, within the first 2 paragraphs of the page",
	} {
		if !strings.Contains(recorder.last, want) {
			t.Fatalf("the prompt lacks %q:\n%s", want, recorder.last)
		}
	}
}
