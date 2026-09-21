package graph_test

import (
	"context"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/applicationtest"
	appgraph "github.com/davidmovas/postulator/internal/application/graph"
	port "github.com/davidmovas/postulator/internal/application/llm"
	graphdomain "github.com/davidmovas/postulator/internal/domain/graph"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

type scriptedModel struct {
	replies []string
	calls   []port.Request
	err     error
}

func (m *scriptedModel) Complete(_ context.Context, req port.Request) (port.Response, error) {
	m.calls = append(m.calls, req)
	if m.err != nil {
		return port.Response{}, m.err
	}

	reply := "{}"
	if len(m.replies) > 0 {
		reply = m.replies[0]
		if len(m.replies) > 1 {
			m.replies = m.replies[1:]
		}
	}
	return port.Response{Text: reply, Usage: domainllm.Usage{Input: 5, Output: 7, Total: 12}}, nil
}

func (m *scriptedModel) Stream(context.Context, port.Request) (<-chan port.Delta, error) {
	return nil, errors.New(errors.Internal, "the scripted model does not stream")
}

type fixedProfiles struct {
	err error
}

func (p fixedProfiles) Resolve(context.Context, string, domainllm.Role, map[domainllm.Role]domainllm.ModelRef) (domainllm.ModelRef, error) {
	if p.err != nil {
		return domainllm.ModelRef{}, p.err
	}
	return domainllm.ModelRef{Provider: "openai", Model: "unit"}, nil
}

type proposeFixture struct {
	service *appgraph.Service
	store   *sqlite.Store
	model   *scriptedModel
	pages   *sqlite.PageRepo
	edges   *sqlite.EdgeRepo
	siteID  string
}

func newProposeFixture(t *testing.T, model *scriptedModel, profiles fixedProfiles) proposeFixture {
	t.Helper()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	pageRepo := sqlite.NewPageRepo(store)

	return proposeFixture{
		service: appgraph.New(appgraph.Deps{
			Entities: sqlite.NewEntityRepo(store), Edges: sqlite.NewEdgeRepo(store),
			Sites: sqlite.NewSiteRepo(store), Pages: pageRepo, Profiles: profiles, LLM: model,
			UnitOfWork: store, Publisher: &applicationtest.Recorder{}, Clock: clock.NewFake(sqlitetest.Stamp),
		}),
		store: store, model: model, pages: pageRepo, edges: sqlite.NewEdgeRepo(store), siteID: owner.ID,
	}
}

func (f proposeFixture) page(t *testing.T, path, title string) pagemap.Page {
	t.Helper()

	record := pagemap.Page{
		ID: id.New(), SiteID: f.siteID, Path: path, Slug: pagemap.Slug(path), WPType: pagemap.WPPage,
		Title: title, H1: title, Status: pagemap.StatusPublished,
		CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	}
	if err := f.pages.Insert(t.Context(), record); err != nil {
		t.Fatalf("insert the page: %v", err)
	}
	return record
}

const proposal = `{"entities":[
	{"path":"/coffee/","name":"Coffee","kind":"hub","intent":"choose a brew","primaryKeyword":"coffee",
	 "secondaryKeywords":["beans"],"anchors":["coffee"],"parentPath":"","relatedPaths":[]},
	{"path":"/coffee/espresso/","name":"Espresso","kind":"topic","intent":"pull a shot",
	 "primaryKeyword":"espresso","secondaryKeywords":[],"anchors":["espresso"],
	 "parentPath":"/coffee/","relatedPaths":["/coffee/filter/"]},
	{"path":"/coffee/filter/","name":"Filter Coffee","kind":"topic","intent":"brew filter",
	 "primaryKeyword":"filter coffee","secondaryKeywords":[],"anchors":["filter coffee"],
	 "parentPath":"/coffee/","relatedPaths":["/coffee/espresso/"]},
	{"path":"/nowhere/","name":"Ghost","kind":"topic","intent":"","primaryKeyword":"",
	 "secondaryKeywords":[],"anchors":[],"parentPath":"","relatedPaths":[]}
]}`

func TestProposeFromPagesBuildsTheGraph(t *testing.T) {
	t.Parallel()

	f := newProposeFixture(t, &scriptedModel{replies: []string{proposal}}, fixedProfiles{})
	hub := f.page(t, "/coffee/", "Coffee")
	f.page(t, "/coffee/espresso/", "Espresso")
	f.page(t, "/coffee/filter/", "Filter coffee")

	out, err := f.service.ProposeFromPages(t.Context(), appgraph.ProposeFromPagesRequest{SiteID: f.siteID})
	if err != nil {
		t.Fatalf("ProposeFromPages: %v", err)
	}
	if len(out.Entities) != 3 || out.Skipped != 1 || out.Tokens != 12 {
		t.Fatalf("response = %+v", out)
	}
	for i := range out.Entities {
		if out.Entities[i].Source != string(graphdomain.SourceAI) {
			t.Fatalf("entity %+v is not marked as proposed by the model", out.Entities[i])
		}
	}
	if len(out.Edges) != 3 {
		t.Fatalf("edges = %+v, want one parent per child and one related pair", out.Edges)
	}
	for i := range out.Edges {
		if out.Edges[i].Status != string(graphdomain.StatusProposed) || out.Edges[i].Source != string(graphdomain.SourceAI) {
			t.Fatalf("edge %+v must be proposed by the model", out.Edges[i])
		}
	}
	reasons := make([]string, 0, len(out.Edges))
	for i := range out.Edges {
		reasons = append(reasons, out.Edges[i].Reason)
	}
	for _, want := range []string{
		"/coffee/espresso/ sits one level under /coffee/",
		"/coffee/filter/ sits one level under /coffee/",
		"/coffee/espresso/ and /coffee/filter/ are sibling pages of one subject",
	} {
		if !slices.Contains(reasons, want) {
			t.Errorf("the reasons %q do not carry %q", reasons, want)
		}
	}

	mapped, err := f.pages.Get(t.Context(), hub.ID)
	if err != nil || mapped.EntityID == nil {
		t.Fatalf("the hub page is %+v, %v", mapped, err)
	}

	prompt := f.model.calls[0].Messages[0].Text
	for _, want := range []string{"/coffee/espresso/", "Filter coffee"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("the prompt does not carry %q", want)
		}
	}

	again, err := f.service.ProposeFromPages(t.Context(), appgraph.ProposeFromPagesRequest{SiteID: f.siteID})
	if err != nil {
		t.Fatalf("a second pass: %v", err)
	}
	if len(again.Entities) != 0 || len(again.Edges) != 0 {
		t.Fatalf("a page that is already mapped must not be proposed again: %+v", again)
	}
}

func TestProposeFromPagesBatchesTheCalls(t *testing.T) {
	t.Parallel()

	f := newProposeFixture(t, &scriptedModel{replies: []string{`{"entities":[]}`}}, fixedProfiles{})
	for i := range appgraph.PagesPerCall + 1 {
		f.page(t, "/page-"+string(rune('a'+i%26))+string(rune('a'+i/26))+"/", "Page")
	}

	if _, err := f.service.ProposeFromPages(t.Context(), appgraph.ProposeFromPagesRequest{SiteID: f.siteID}); err != nil {
		t.Fatalf("ProposeFromPages: %v", err)
	}
	if len(f.model.calls) != 2 {
		t.Fatalf("the model was called %d times, want one per batch of %d", len(f.model.calls), appgraph.PagesPerCall)
	}
}

func TestProposeFromPagesRefusesWhatItCannotRead(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		siteID   func(proposeFixture) string
		profiles fixedProfiles
		model    *scriptedModel
		want     errors.Code
	}{
		{name: "no site", siteID: func(proposeFixture) string { return " " }, want: errors.Invalid},
		{name: "unknown site", siteID: func(proposeFixture) string { return id.New() }, want: errors.NotFound},
		{
			name: "no model is bound", siteID: func(f proposeFixture) string { return f.siteID },
			profiles: fixedProfiles{err: errors.New(errors.Unauthorized, "no key")}, want: errors.Unauthorized,
		},
		{
			name: "the model fails", siteID: func(f proposeFixture) string { return f.siteID },
			model: &scriptedModel{err: errors.New(errors.External, "the model is down")}, want: errors.External,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			model := tc.model
			if model == nil {
				model = &scriptedModel{replies: []string{`{"entities":[]}`}}
			}
			f := newProposeFixture(t, model, tc.profiles)
			f.page(t, "/coffee/", "Coffee")

			_, err := f.service.ProposeFromPages(t.Context(), appgraph.ProposeFromPagesRequest{SiteID: tc.siteID(f)})
			if !errors.IsCode(err, tc.want) {
				t.Fatalf("ProposeFromPages = %v, want %s", err, tc.want)
			}
		})
	}
}

func TestProposeRelatedConnectsExistingEntities(t *testing.T) {
	t.Parallel()

	reply := `{"edges":[
		{"from":"Espresso","to":"Filter Coffee","weight":0.7,"reason":"both are brewing methods"},
		{"from":"Espresso","to":"Espresso","weight":1,"reason":"itself"},
		{"from":"Espresso","to":"Tea","weight":0.5,"reason":"unknown entity"}
	]}`
	f := newProposeFixture(t, &scriptedModel{replies: []string{reply}}, fixedProfiles{})
	sqlitetest.Entity(t, f.store, f.siteID, "Espresso")
	sqlitetest.Entity(t, f.store, f.siteID, "Filter Coffee")

	out, err := f.service.ProposeRelated(t.Context(), appgraph.ProposeRelatedRequest{SiteID: f.siteID})
	if err != nil {
		t.Fatalf("ProposeRelated: %v", err)
	}
	if len(out.Edges) != 1 || out.Skipped != 2 {
		t.Fatalf("response = %+v", out)
	}
	if out.Edges[0].Status != string(graphdomain.StatusProposed) || out.Edges[0].Weight != 0.7 {
		t.Fatalf("edge = %+v", out.Edges[0])
	}
	if out.Edges[0].Reason != "both are brewing methods" {
		t.Fatalf("the model's reason was not kept: %+v", out.Edges[0])
	}

	f.model.replies = []string{reply}
	again, err := f.service.ProposeRelated(t.Context(), appgraph.ProposeRelatedRequest{SiteID: f.siteID})
	if err != nil {
		t.Fatalf("a second pass: %v", err)
	}
	if len(again.Edges) != 0 || again.Skipped != 3 {
		t.Fatalf("a pair that is already connected must not be proposed again: %+v", again)
	}
}

func TestProposeRelatedClipsATalkativeReason(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("é", 500)
	reply := `{"edges":[{"from":"Espresso","to":"Filter Coffee","weight":0.6,"reason":"` + long + `"}]}`
	f := newProposeFixture(t, &scriptedModel{replies: []string{reply}}, fixedProfiles{})
	sqlitetest.Entity(t, f.store, f.siteID, "Espresso")
	sqlitetest.Entity(t, f.store, f.siteID, "Filter Coffee")

	out, err := f.service.ProposeRelated(t.Context(), appgraph.ProposeRelatedRequest{SiteID: f.siteID})
	if err != nil {
		t.Fatalf("ProposeRelated: %v", err)
	}
	if len(out.Edges) != 1 || out.Skipped != 0 {
		t.Fatalf("a long reason must cost words, not the edge: %+v", out)
	}
	if got := utf8.RuneCountInString(out.Edges[0].Reason); got != graphdomain.EdgeReasonMax {
		t.Fatalf("reason length = %d, want %d", got, graphdomain.EdgeReasonMax)
	}
}

func TestProposeRelatedTakesAFocusEntity(t *testing.T) {
	t.Parallel()

	f := newProposeFixture(t, &scriptedModel{replies: []string{`{"edges":[]}`}}, fixedProfiles{})
	espresso := sqlitetest.Entity(t, f.store, f.siteID, "Espresso")
	sqlitetest.Entity(t, f.store, f.siteID, "Filter Coffee")

	if _, err := f.service.ProposeRelated(t.Context(), appgraph.ProposeRelatedRequest{
		SiteID: f.siteID, EntityID: espresso.ID,
	}); err != nil {
		t.Fatalf("ProposeRelated: %v", err)
	}
	if !strings.Contains(f.model.calls[0].Messages[0].Text, "FOCUS") {
		t.Fatalf("the prompt does not name the focus:\n%s", f.model.calls[0].Messages[0].Text)
	}

	_, err := f.service.ProposeRelated(t.Context(), appgraph.ProposeRelatedRequest{
		SiteID: f.siteID, EntityID: id.New(),
	})
	if !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("an unknown focus = %v, want not found", err)
	}
}

func TestProposeRelatedNeedsTwoEntities(t *testing.T) {
	t.Parallel()

	f := newProposeFixture(t, &scriptedModel{replies: []string{`{"edges":[]}`}}, fixedProfiles{})
	sqlitetest.Entity(t, f.store, f.siteID, "Espresso")

	out, err := f.service.ProposeRelated(t.Context(), appgraph.ProposeRelatedRequest{SiteID: f.siteID})
	if err != nil || len(out.Edges) != 0 {
		t.Fatalf("ProposeRelated = %+v, %v", out, err)
	}
	if len(f.model.calls) != 0 {
		t.Fatal("a site with one entity has nothing to relate, so the model must not be called")
	}
}

func TestProposalsNameTheStepTheySpendOn(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		reply string
		want  string
		run   func(t *testing.T, f proposeFixture) error
	}{
		{
			name:  "from the pages",
			reply: proposal,
			want:  appgraph.NameProposeFromPages,
			run: func(t *testing.T, f proposeFixture) error {
				t.Helper()
				f.page(t, "/coffee/", "Coffee")
				_, err := f.service.ProposeFromPages(t.Context(), appgraph.ProposeFromPagesRequest{SiteID: f.siteID})
				return err
			},
		},
		{
			name:  "between the entities",
			reply: `{"edges":[]}`,
			want:  appgraph.NameProposeRelated,
			run: func(t *testing.T, f proposeFixture) error {
				t.Helper()
				sqlitetest.Entity(t, f.store, f.siteID, "Coffee")
				sqlitetest.Entity(t, f.store, f.siteID, "Espresso")
				_, err := f.service.ProposeRelated(t.Context(), appgraph.ProposeRelatedRequest{SiteID: f.siteID})
				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			model := &scriptedModel{replies: []string{tc.reply}}
			f := newProposeFixture(t, model, fixedProfiles{})
			if err := tc.run(t, f); err != nil {
				t.Fatalf("propose: %v", err)
			}
			if len(model.calls) == 0 {
				t.Fatal("the model was never called")
			}
			if step := model.calls[0].Meta.Step; step != tc.want {
				t.Errorf("step = %q, want %q; the ledger and the harness both key on it", step, tc.want)
			}
		})
	}
}
