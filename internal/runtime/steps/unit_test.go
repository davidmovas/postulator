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
	"github.com/davidmovas/postulator/internal/domain/keyword"
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
			ID: "parent", SiteID: "site", Name: "Coffee", Keywords: keyword.Of("coffee"),
			Anchors: []graph.Anchor{{Text: "coffee", Source: graph.AnchorUser, Weight: 1}},
			Kind:    graph.KindTopic, Source: graph.SourceUser, CanonicalPageID: pointer("page-parent"),
		},
		{
			ID: "child", SiteID: "site", Name: "Espresso", Keywords: keyword.Of("espresso"),
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

func onTheSite(deps steps.Deps) steps.Deps {
	listed, ok := deps.Pages.(pageList)
	if !ok {
		return deps
	}
	items := make([]pagemap.Page, 0, len(listed.items))
	for i := range listed.items {
		page := listed.items[i]
		wpID := int64(100 + i)
		page.WPID = &wpID
		items = append(items, page)
	}
	listed.items = items
	deps.Pages = listed
	return deps
}

func TestValidateNamesALinkToAPageThatIsNotOnTheSite(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		deps   func(steps.Deps) steps.Deps
		within []string
		named  bool
	}{
		{name: "the parent is on the site", deps: onTheSite},
		{name: "the parent is written by this run", within: []string{"page-child", "page-parent"}},
		{name: "the parent is only planned", named: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps := unitDeps()
			if tc.deps != nil {
				deps = tc.deps(deps)
			}
			sc := unitContext(t, map[run.ArtifactKind][]byte{
				run.ArtifactLinkContext: linkContextBlob(t, deps),
				run.ArtifactBodyHTML:    []byte(`<h1>Espresso</h1><h2>About</h2><p>Espresso is a kind of <a href="/coffee/">coffee</a>.</p>`),
			})
			sc.Run.Targets = tc.within

			result, err := steps.Validate(deps).Run(t.Context(), sc)
			if err != nil {
				t.Fatalf("Validate: %v", err)
			}
			if result.Next == run.TransitionPause {
				t.Fatalf("result = %+v, want a link to a planned page never to hold the page", result)
			}
			var decoded steps.ValidationReport
			if err = json.Unmarshal(result.Artifacts[0].Blob, &decoded); err != nil {
				t.Fatalf("decode the report: %v", err)
			}
			named := false
			for _, item := range decoded.Compliance.Items {
				if item.Code == content.CodeTargetNotPublished {
					named = true
				}
			}
			if named != tc.named {
				t.Fatalf("compliance = %+v, want target_not_published %v", decoded.Compliance.Items, tc.named)
			}
			if tc.named && decoded.Compliance.Score >= 1 {
				t.Fatalf("score = %v, want the warning to count", decoded.Compliance.Score)
			}
		})
	}
}

func TestInsertLinksAndValidateReadTheirArtifacts(t *testing.T) {
	t.Parallel()

	deps := onTheSite(unitDeps())
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

func TestValidateGradesAProductOnItsShortDescriptionNotOnItsName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		short string
		held  bool
	}{
		{name: "a short description with the keyword", short: "<p>An espresso machine.</p>"},
		{name: "a short description without it", short: "<p>A machine.</p>", held: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps := unitDeps()
			body := []byte(`<h1>Machine</h1><h2>About</h2><p>Espresso is a kind of <a href="/coffee/">coffee</a>.</p>`)
			draft, err := json.Marshal(content.ContentDraft{
				Title: "Machine", H1: "Machine",
				Sections: []content.DraftSection{{Heading: "About", HTML: "<p>Espresso.</p>"}},
				Product:  &content.ProductDraft{ShortDescription: tc.short},
			})
			if err != nil {
				t.Fatalf("encode the draft: %v", err)
			}
			sc := productContext(t, deps)
			sc.Artifacts[run.ArtifactBodyHTML] = run.Artifact{Kind: run.ArtifactBodyHTML, Blob: body}
			sc.Artifacts[run.ArtifactDraft] = run.Artifact{Kind: run.ArtifactDraft, Blob: draft}
			sc.Spec.KeywordRules.PrimaryInH1 = true

			result, err := steps.Validate(deps).Run(t.Context(), sc)
			if err != nil {
				t.Fatalf("Validate: %v", err)
			}
			if held := result.Next == run.TransitionPause; held != tc.held {
				t.Fatalf("held = %t (%s), want %t", held, result.Message, tc.held)
			}

			var decoded steps.ValidationReport
			if err := json.Unmarshal(result.Artifacts[0].Blob, &decoded); err != nil {
				t.Fatalf("decode the report: %v", err)
			}
			if hasFinding(decoded, content.CodePrimaryMissingInH1) || !hasFinding(decoded, content.CodePrimaryMissingInName) {
				t.Errorf("findings = %+v, want the name named instead of an H1 error", decoded.Structure.Items)
			}
			if got := hasFinding(decoded, content.CodePrimaryMissingInShortDescription); got != tc.held {
				t.Errorf("short description finding = %t, want %t", got, tc.held)
			}
		})
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

func TestRepairLinksOwesTheLeadTheMainKeywordOfThePage(t *testing.T) {
	t.Parallel()

	deps := unitDeps()
	recorder := &promptRecorder{reply: `{"sentence":"Pulling espresso at home starts with the grind."}`}
	deps.LLM = recorder
	blob := linkContextBlob(t, deps)

	sc := unitContext(t, map[run.ArtifactKind][]byte{
		run.ArtifactLinkContext: blob,
		run.ArtifactBodyHTML:    []byte(`<h1>Espresso</h1><p>Espresso is a kind of <a href="/coffee/">coffee</a>.</p>`),
	})
	sc.Page.Keywords = keyword.New([]keyword.Keyword{{Text: "moka pot"}, {Text: "espresso at home", Volume: new(800)}})

	result, err := steps.RepairLinks(deps).Run(t.Context(), sc)
	if err != nil {
		t.Fatalf("RepairLinks: %v", err)
	}
	if !strings.Contains(recorder.last, "espresso at home") {
		t.Fatalf("the linker was not asked for the page's main keyword:\n%s", recorder.last)
	}
	if body := string(result.Artifacts[0].Blob); !strings.Contains(body, "Pulling espresso at home starts with the grind.") {
		t.Fatalf("the sentence did not land in the body:\n%s", body)
	}
}

func TestValidateGradesTheKeywordsOfThePage(t *testing.T) {
	t.Parallel()

	deps := onTheSite(unitDeps())
	blob := linkContextBlob(t, deps)
	body := []byte(`<h1>Espresso at home</h1><h2>About</h2><p>Espresso at home is a kind of <a href="/coffee/">coffee</a>.</p>`)

	sc := unitContext(t, map[run.ArtifactKind][]byte{run.ArtifactLinkContext: blob, run.ArtifactBodyHTML: body})
	sc.Page.Keywords = keyword.New([]keyword.Keyword{
		{Text: "moka pot"}, {Text: "espresso beans", Volume: new(300)}, {Text: "espresso at home", Volume: new(800)},
	})

	result, err := steps.Validate(deps).Run(t.Context(), sc)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	var decoded steps.ValidationReport
	if unmarshalErr := json.Unmarshal(result.Artifacts[0].Blob, &decoded); unmarshalErr != nil {
		t.Fatalf("decode the report: %v", unmarshalErr)
	}

	missing := 0
	for _, item := range decoded.Structure.Items {
		if item.Code != content.CodeKeywordsMissing {
			continue
		}
		missing++
		if want := "the body does not use 2 of the 3 keywords of the page: espresso beans, moka pot"; item.Message != want {
			t.Fatalf("message = %q, want %q", item.Message, want)
		}
	}
	if missing != 1 {
		t.Fatalf("the report carries %d findings about missing keywords, want one: %+v", missing, decoded.Structure.Items)
	}
	if decoded.Structure.HasErrors() {
		t.Fatalf("missing keywords must not be errors: %+v", decoded.Structure.Items)
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

type requestRecorder struct {
	reply    string
	requests []port.Request
}

func (r *requestRecorder) Complete(_ context.Context, req port.Request) (port.Response, error) {
	r.requests = append(r.requests, req)
	return port.Response{Text: r.reply, Usage: domainllm.Usage{Input: 1, Output: 2, Total: 3}}, nil
}

func (r *requestRecorder) Stream(context.Context, port.Request) (<-chan port.Delta, error) {
	return nil, errors.New(errors.Internal, "the unit stub does not stream")
}

func TestEveryModelStepAsksUnderItsOwnNameAndCeiling(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		def     func(steps.Deps) run.StepDef
		reply   string
		context func(*testing.T, steps.Deps) *run.StepContext
	}{
		{
			name: steps.NameGenerateBody, def: steps.GenerateBody, reply: goodDraft,
			context: func(t *testing.T, deps steps.Deps) *run.StepContext {
				t.Helper()
				return unitContext(t, map[run.ArtifactKind][]byte{run.ArtifactLinkContext: linkContextBlob(t, deps)})
			},
		},
		{
			name: steps.NameGenerateMeta, def: steps.GenerateMeta,
			reply: `{"title":"Espresso | Shop","description":"Pull a shot.","canonical":"https://shop.example.com/coffee/espresso/"}`,
			context: func(t *testing.T, _ steps.Deps) *run.StepContext {
				t.Helper()
				return unitContext(t, map[run.ArtifactKind][]byte{run.ArtifactDraft: []byte(goodDraft)})
			},
		},
		{
			name: steps.NameJudge, def: steps.Judge, reply: `{"score":0.9,"issues":[],"suggestions":[]}`,
			context: func(t *testing.T, _ steps.Deps) *run.StepContext {
				t.Helper()
				return judgeContext(t)
			},
		},
		{
			name: steps.NameRepairLinks, def: steps.RepairLinks, reply: `{"sentence":"A sentence with no anchor at all."}`,
			context: func(t *testing.T, _ steps.Deps) *run.StepContext {
				t.Helper()
				return unitContext(t, map[run.ArtifactKind][]byte{
					run.ArtifactLinkContext: owingContext(t),
					run.ArtifactBodyHTML:    []byte("<h1>Espresso</h1><p>Our espresso starts with the coffee we roast.</p>"),
				})
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps := unitDeps()
			sc := tc.context(t, deps)
			recorder := &requestRecorder{reply: tc.reply}
			deps.LLM = recorder
			deps.Content = appcontent.New(appcontent.Deps{Profiles: deps.Profiles, LLM: recorder})
			def := tc.def(deps)

			if _, err := def.Run(t.Context(), sc); err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			if len(recorder.requests) == 0 {
				t.Fatalf("%s asked the model nothing", tc.name)
			}
			for _, req := range recorder.requests {
				if req.Meta.RunID != sc.Run.ID || req.Meta.ItemID != sc.Item.ID || req.Meta.Step != tc.name {
					t.Fatalf("the call is booked as %+v, want run %s, item %s, step %s", req.Meta, sc.Run.ID, sc.Item.ID, tc.name)
				}
				if req.Meta.Role == "" || req.Meta.Role != def.Role {
					t.Fatalf("the call is sent for the role %q, want the step's own %q", req.Meta.Role, def.Role)
				}
				if req.Ref.Model != "unit" || req.System == "" || len(req.Messages) != 1 ||
					req.Messages[0].Role != port.RoleUser || req.Messages[0].Text == "" {
					t.Fatalf("the request = %+v, want the resolved model, a system prompt and one user message", req)
				}
				ceiling := def.Price.OutputTokens
				if (ceiling == 0 && req.MaxTokens <= 0) || (ceiling > 0 && req.MaxTokens != ceiling) {
					t.Fatalf("the request asks for at most %d tokens, want the ceiling the step is priced at (%d)", req.MaxTokens, ceiling)
				}
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
		"Primary keyword: espresso",
		"1) espresso\n",
	} {
		if !strings.Contains(recorder.last, want) {
			t.Fatalf("the prompt lacks %q:\n%s", want, recorder.last)
		}
	}
}

func TestTheWriterPromptListsTheKeywordsOfThePageMostImportantFirst(t *testing.T) {
	t.Parallel()

	deps := unitDeps()
	recorder := &promptRecorder{reply: goodDraft}
	deps.LLM = recorder
	blob := linkContextBlob(t, deps)

	sc := unitContext(t, map[run.ArtifactKind][]byte{run.ArtifactLinkContext: blob})
	sc.Page.Keywords = keyword.New([]keyword.Keyword{
		{Text: "moka pot"}, {Text: "espresso beans", Volume: new(300)}, {Text: "espresso at home", Volume: new(800)},
	})
	sc.Spec.KeywordRules.RequiredKeywords = new(2)

	if _, err := steps.GenerateBody(deps).Run(t.Context(), sc); err != nil {
		t.Fatalf("GenerateBody: %v", err)
	}
	for _, want := range []string{
		"Primary keyword: espresso at home",
		"1) espresso at home (800 a month)\n",
		"2) espresso beans (300 a month)\n",
		"3) moka pot (optional)\n",
		"Use every keyword that is not marked optional at least once",
		"- espresso at home, in the first paragraph of section 1",
	} {
		if !strings.Contains(recorder.last, want) {
			t.Fatalf("the prompt lacks %q:\n%s", want, recorder.last)
		}
	}
	for _, gone := range []string{"Secondary keywords", "Primary keyword: espresso\n"} {
		if strings.Contains(recorder.last, gone) {
			t.Fatalf("the prompt still carries %q:\n%s", gone, recorder.last)
		}
	}
}

func TestTheWriterPromptSaysWhenAPageHasNoKeywords(t *testing.T) {
	t.Parallel()

	deps := unitDeps()
	bare := unitEntities()
	for i := range bare {
		bare[i].Keywords = keyword.Of()
	}
	deps.Entities = entityList{items: bare}
	recorder := &promptRecorder{reply: goodDraft}
	deps.LLM = recorder
	blob := linkContextBlob(t, deps)

	if _, err := steps.GenerateBody(deps).Run(t.Context(), unitContext(t, map[run.ArtifactKind][]byte{run.ArtifactLinkContext: blob})); err != nil {
		t.Fatalf("GenerateBody: %v", err)
	}
	if !strings.Contains(recorder.last, "KEYWORDS\nnone\n") {
		t.Fatalf("the prompt does not say the page has no keywords:\n%s", recorder.last)
	}
	if strings.Contains(recorder.last, "in the first paragraph of section 1") {
		t.Fatalf("a page without keywords owes no lead phrase:\n%s", recorder.last)
	}
}

const productDraftReply = `{"title":"Espresso guide","h1":"Espresso guide","sections":[` +
	`{"heading":"About","html":"<p>Espresso is a way to make coffee, part of our drinks range.</p>"},` +
	`{"heading":"Brewing","html":"<p>Use fresh water and a fine grind for a sweeter cup at home.</p>"}` +
	`],"summary":"A short guide to espresso.",` +
	`"shortDescription":"<p>An espresso machine for the home.</p>",` +
	`"specifications":[{"name":"Form","value":"Countertop"},{"name":"Size","value":""}]}`

func productContext(t *testing.T, deps steps.Deps) *run.StepContext {
	t.Helper()

	sc := unitContext(t, map[run.ArtifactKind][]byte{run.ArtifactLinkContext: linkContextBlob(t, deps)})
	sc.Page.WPType = pagemap.WPProduct
	sc.Page.Observed.Title = "Espresso Machine &amp; Grinder"
	sc.Spec.Product = &template.Product{
		ShortDescription: template.ProductShortDescription{Enabled: true, Intent: "Say what it is", TargetWords: 30, PrimaryKeyword: true},
		Specifications: []template.ProductSpecification{
			{Name: "Form", Intent: "The form the notes state"}, {Name: "Size", Intent: "The size the notes state"},
		},
	}
	return sc
}

func TestTheWriterAnswersAProductUnderItsStoreName(t *testing.T) {
	t.Parallel()

	deps := unitDeps()
	recorder := &promptRecorder{reply: productDraftReply}
	deps.LLM = recorder
	sc := productContext(t, deps)

	result, err := steps.GenerateBody(deps).Run(t.Context(), sc)
	if err != nil {
		t.Fatalf("GenerateBody: %v", err)
	}
	for _, want := range []string{
		"writing body copy for a WooCommerce product",
		"shortDescription and specifications",
		"H1: Espresso Machine & Grinder (the product's name in the store",
		"never state a price, a stock level or an SKU",
		"Short description: write it in shortDescription as HTML paragraphs, about 30 words: Say what it is; it carries the primary keyword",
		"- Form: The form the notes state\n",
	} {
		if !strings.Contains(recorder.last, want) {
			t.Fatalf("the prompt lacks %q:\n%s", want, recorder.last)
		}
	}

	var draft content.ContentDraft
	if err := json.Unmarshal(result.Artifacts[0].Blob, &draft); err != nil {
		t.Fatalf("decode the draft: %v", err)
	}
	if draft.H1 != "Espresso Machine & Grinder" || draft.Product == nil {
		t.Fatalf("draft = %+v", draft)
	}
	if draft.Product.ShortDescription != "<p>An espresso machine for the home.</p>" ||
		len(draft.Product.Specifications) != 1 || draft.Product.Specifications[0].Value != "Countertop" {
		t.Errorf("product draft = %+v", draft.Product)
	}
}

func TestTheWriterOfAPageIsAskedNothingAboutProducts(t *testing.T) {
	t.Parallel()

	deps := unitDeps()
	recorder := &promptRecorder{reply: goodDraft}
	deps.LLM = recorder
	sc := productContext(t, deps)
	sc.Page.WPType = pagemap.WPPage

	result, err := steps.GenerateBody(deps).Run(t.Context(), sc)
	if err != nil {
		t.Fatalf("GenerateBody: %v", err)
	}
	for _, gone := range []string{"WooCommerce", "PRODUCT\n", "shortDescription"} {
		if strings.Contains(recorder.last, gone) {
			t.Fatalf("a page's prompt carries %q:\n%s", gone, recorder.last)
		}
	}
	var draft content.ContentDraft
	if err := json.Unmarshal(result.Artifacts[0].Blob, &draft); err != nil || draft.Product != nil {
		t.Fatalf("a page's draft = %+v, %v", draft.Product, err)
	}
}

func TestTheWriterCeilingMakesRoomForAProductsShortDescription(t *testing.T) {
	t.Parallel()

	deps := unitDeps()
	recorder := &ceilingRecorder{reply: productDraftReply}
	deps.LLM = recorder

	sc := productContext(t, deps)
	sc.Spec.Sections = []template.Section{{Heading: "About", TargetWords: 1800, Required: true}, {Heading: "Brewing"}}
	if _, err := steps.GenerateBody(deps).Run(t.Context(), sc); err != nil {
		t.Fatalf("GenerateBody: %v", err)
	}
	if got, want := recorder.ceilings[len(recorder.ceilings)-1], (1800+30)*3+1024; got != want {
		t.Errorf("ceiling = %d, want %d", got, want)
	}
}

func TestTheWriterReadsTheNotesOfThePageAsContext(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		notes []pagemap.Note
		want  []string
		gone  []string
	}{
		{
			name:  "a page with notes",
			notes: []pagemap.Note{{Label: "Intent Owner", Text: "Commercial"}, {Label: "Notes", Text: "Sold as a 10 ml vial"}},
			want:  []string{"NOTES\n", "- Intent Owner: Commercial\n", "- Notes: Sold as a 10 ml vial\n", "never copy"},
		},
		{name: "a page without notes", gone: []string{"NOTES\n"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps := unitDeps()
			recorder := &promptRecorder{reply: goodDraft}
			deps.LLM = recorder
			sc := unitContext(t, map[run.ArtifactKind][]byte{run.ArtifactLinkContext: linkContextBlob(t, deps)})
			sc.Page.Notes = tc.notes

			if _, err := steps.GenerateBody(deps).Run(t.Context(), sc); err != nil {
				t.Fatalf("GenerateBody: %v", err)
			}
			for _, want := range tc.want {
				if !strings.Contains(recorder.last, want) {
					t.Fatalf("the prompt lacks %q:\n%s", want, recorder.last)
				}
			}
			for _, gone := range tc.gone {
				if strings.Contains(recorder.last, gone) {
					t.Fatalf("the prompt carries %q:\n%s", gone, recorder.last)
				}
			}
		})
	}
}

func sharedNameEntities() []graph.Entity {
	entities := unitEntities()
	entities[1].ScopeID = pointer("parent")
	return append(entities,
		graph.Entity{ID: "tea", SiteID: "site", Name: "Tea", Kind: graph.KindTopic, Source: graph.SourceUser},
		graph.Entity{
			ID: "tea-espresso", SiteID: "site", Name: "Espresso", Kind: graph.KindTopic, Source: graph.SourceUser,
			ScopeID: pointer("tea"),
		},
	)
}

func TestAPromptNamesAnEntityWhoseNameIsSharedWithItsParent(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		run  func(t *testing.T, deps steps.Deps) error
		want []string
	}{
		{
			name: "the writer",
			run: func(t *testing.T, deps steps.Deps) error {
				sc := unitContext(t, map[run.ArtifactKind][]byte{run.ArtifactLinkContext: linkContextBlob(t, deps)})
				_, err := steps.GenerateBody(deps).Run(t.Context(), sc)
				return err
			},
			want: []string{"Name: Coffee Espresso\n"},
		},
		{
			name: "the meta writer and its title pattern",
			run: func(t *testing.T, deps steps.Deps) error {
				sc := unitContext(t, map[run.ArtifactKind][]byte{run.ArtifactDraft: []byte(goodDraft)})
				sc.Spec.MetaRules = template.MetaRules{TitlePattern: "{entityName} | {siteName}", DescriptionMax: 155}
				_, err := steps.GenerateMeta(deps).Run(t.Context(), sc)
				return err
			},
			want: []string{"Name: Coffee Espresso\n", "The title follows this shape exactly: Coffee Espresso | "},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps := unitDeps()
			deps.Entities = entityList{items: sharedNameEntities()}
			recorder := &promptRecorder{reply: goodDraft}
			deps.LLM = recorder

			if err := tc.run(t, deps); err != nil {
				t.Fatalf("run: %v", err)
			}
			for _, want := range tc.want {
				if !strings.Contains(recorder.last, want) {
					t.Fatalf("the prompt lacks %q:\n%s", want, recorder.last)
				}
			}
		})
	}
}
