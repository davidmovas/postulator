package steps_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/graph"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/runtime"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const (
	pollInterval = 5 * time.Millisecond
	pollTimeout  = 20 * time.Second

	goodDraft = `{"title":"Espresso guide","h1":"Espresso guide","sections":[` +
		`{"heading":"About","html":"<p>Espresso is a way to make coffee, part of our drinks range.</p>"},` +
		`{"heading":"Brewing","html":"<p>Use fresh water and a fine grind for a sweeter cup at home.</p>"}` +
		`],"summary":"A short guide to espresso."}`

	anchorlessDraft = `{"title":"Espresso guide","h1":"Espresso guide","sections":[` +
		`{"heading":"About","html":"<p>Espresso is a small strong shot pulled under pressure.</p>"},` +
		`{"heading":"Brewing","html":"<p>Use fresh water and a fine grind for a sweeter cup at home.</p>"}` +
		`],"summary":"A short guide to espresso."}`

	keywordlessDraft = `{"title":"A guide","h1":"A guide","sections":[` +
		`{"heading":"About","html":"<p>It is a way to make coffee, part of our drinks range.</p>"},` +
		`{"heading":"Brewing","html":"<p>Use fresh water and a fine grind for a sweeter cup at home.</p>"}` +
		`],"summary":"A short guide."}`

	repairSentence = `{"sentence":"It sits in our drinks range next to every other coffee we sell."}`
)

type stubProfiles struct{}

func (stubProfiles) Resolve(context.Context, string, domainllm.Role, map[domainllm.Role]domainllm.ModelRef) (domainllm.ModelRef, error) {
	return domainllm.ModelRef{Provider: "openai", Model: "test-writer"}, nil
}

type stubCatalog struct{}

func (stubCatalog) Lookup(context.Context, domainllm.ModelRef) (domainllm.ModelInfo, error) {
	return domainllm.ModelInfo{InputUSDPerM: 1, OutputUSDPerM: 2}, nil
}

type recorder struct {
	types []events.Type
	mu    sync.Mutex
}

func (r *recorder) PublishRun(_ string, _ int64, eventType events.Type, _ any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.types = append(r.types, eventType)
	return nil
}

type factory struct {
	store  *sqlite.Store
	runs   *sqlite.RunRepo
	items  *sqlite.RunItemRepo
	blobs  *sqlite.ArtifactRepo
	log    *sqlite.RunEventRepo
	llm    *fake.Scripted
	bus    *recorder
	siteID string
	pageID string
}

func recipe() []template.StepSpec {
	return []template.StepSpec{
		{Name: steps.NameResolveContext, Enabled: true},
		{Name: steps.NameGenerateBody, Enabled: true},
		{Name: steps.NameInsertLinks, Enabled: true},
		{Name: steps.NameRepairLinks, Enabled: true},
		{Name: steps.NameValidate, Enabled: true},
	}
}

func spec() template.TemplateSpec {
	return template.TemplateSpec{
		Sections: []template.Section{
			{Heading: "About", Intent: "Explain the topic", TargetWords: 120, Required: true},
			{Heading: "Brewing", Intent: "Give practical advice", TargetWords: 120},
		},
		Tone:         "practical",
		KeywordRules: template.KeywordRules{PrimaryInH1: true, PrimaryInFirstParagraph: true},
		LinkRules: template.LinkRules{
			UpDepth: 2, DownLinks: true, SiblingMinWeight: 0.5, MaxLinks: 5, MaxPerTarget: 1,
		},
		ModelProfiles: map[domainllm.Role]domainllm.ModelRef{},
		Recipe:        recipe(),
	}
}

func newFactory(t *testing.T, draft string) *factory {
	t.Helper()

	store := sqlitetest.Open(t)
	at := sqlitetest.Stamp

	policyRepo := sqlite.NewLinkPolicyRepo(store)
	policy := template.LinkPolicy{
		ID: id.New(), Scope: template.ScopeGlobal, Name: templates.DefaultPolicyName,
		Rules:          spec().LinkRules,
		ForbidExternal: true, ForbidSelf: true, AnchorStrategy: template.AnchorPreferUser,
		CreatedAt: at, UpdatedAt: at,
	}
	if err := policyRepo.Insert(t.Context(), policy); err != nil {
		t.Fatalf("insert the link policy: %v", err)
	}

	templateRepo := sqlite.NewTemplateRepo(store)
	base := template.Template{
		ID: id.New(), Scope: template.ScopeGlobal, Name: "Guide", PageKind: "guide", Version: 1,
		Spec: spec(), CreatedAt: at, UpdatedAt: at,
	}
	if err := templateRepo.Insert(t.Context(), base); err != nil {
		t.Fatalf("insert the template: %v", err)
	}

	siteRepo := sqlite.NewSiteRepo(store)
	owner := site.Site{
		ID: id.New(), Name: "Shop", BaseURL: "https://shop.example.com", Username: "editor",
		Status: site.StatusActive, Plugin: site.PluginState{Capabilities: []string{}},
		Defaults: site.Defaults{
			TemplateID: &base.ID, LinkPolicyID: &policy.ID,
			ModelProfiles: map[domainllm.Role]domainllm.ModelRef{},
		},
		CreatedAt: at, UpdatedAt: at,
	}
	owner.SecretRef = site.SecretRef(owner.ID)
	if err := siteRepo.Insert(t.Context(), owner); err != nil {
		t.Fatalf("insert the site: %v", err)
	}

	pageRepo := sqlite.NewPageRepo(store)
	entityRepo := sqlite.NewEntityRepo(store)
	edgeRepo := sqlite.NewEdgeRepo(store)

	grandparent := seedEntity(t, entityRepo, owner.ID, "Drinks", "drinks")
	parent := seedEntity(t, entityRepo, owner.ID, "Coffee", "coffee")
	child := seedEntity(t, entityRepo, owner.ID, "Espresso", "espresso")

	grandPage := seedPage(t, pageRepo, owner.ID, "/drinks/", grandparent.ID, pagemap.StatusPublished)
	parentPage := seedPage(t, pageRepo, owner.ID, "/drinks/coffee/", parent.ID, pagemap.StatusPublished)
	childPage := seedPage(t, pageRepo, owner.ID, "/drinks/coffee/espresso/", child.ID, pagemap.StatusPlanned)

	for entityID, pageID := range map[string]string{
		grandparent.ID: grandPage.ID, parent.ID: parentPage.ID, child.ID: childPage.ID,
	} {
		if err := entityRepo.SetCanonicalPage(t.Context(), entityID, &pageID, at); err != nil {
			t.Fatalf("set the canonical page: %v", err)
		}
	}

	edges := []graph.Edge{
		{
			ID: id.New(), SiteID: owner.ID, FromEntityID: parent.ID, ToEntityID: grandparent.ID,
			Kind: graph.EdgeParent, Weight: 1, Source: graph.SourceUser, Status: graph.StatusApproved, CreatedAt: at,
		},
		{
			ID: id.New(), SiteID: owner.ID, FromEntityID: child.ID, ToEntityID: parent.ID,
			Kind: graph.EdgeParent, Weight: 1, Source: graph.SourceUser, Status: graph.StatusApproved, CreatedAt: at,
		},
	}
	for i := range edges {
		if err := edgeRepo.Insert(t.Context(), edges[i]); err != nil {
			t.Fatalf("insert the edge: %v", err)
		}
	}

	return &factory{
		store: store,
		runs:  sqlite.NewRunRepo(store),
		items: sqlite.NewRunItemRepo(store),
		blobs: sqlite.NewArtifactRepo(store),
		log:   sqlite.NewRunEventRepo(store),
		llm: fake.NewScripted(map[string]string{
			steps.NameGenerateBody: draft,
			steps.NameRepairLinks:  repairSentence,
		}),
		bus:    &recorder{},
		siteID: owner.ID,
		pageID: childPage.ID,
	}
}

func seedEntity(t *testing.T, repo *sqlite.EntityRepo, siteID, name, anchor string) graph.Entity {
	t.Helper()

	record := graph.Entity{
		ID: id.New(), SiteID: siteID, Name: name, Kind: graph.KindTopic, PrimaryKeyword: anchor,
		SecondaryKeywords: []string{},
		Anchors:           []graph.Anchor{{Text: anchor, Source: graph.AnchorUser, Weight: 1}},
		Source:            graph.SourceUser, CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	}
	if err := repo.Insert(t.Context(), record); err != nil {
		t.Fatalf("insert the entity %s: %v", name, err)
	}
	return record
}

func seedPage(t *testing.T, repo *sqlite.PageRepo, siteID, path, entityID string, status pagemap.Status) pagemap.Page {
	t.Helper()

	record := pagemap.Page{
		ID: id.New(), SiteID: siteID, Path: path, Slug: pagemap.Slug(path), WPType: pagemap.WPPage,
		Title: path, H1: path, Status: status,
		CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	}
	if entityID != "" {
		record.EntityID = &entityID
	}
	if err := repo.Insert(t.Context(), record); err != nil {
		t.Fatalf("insert the page %s: %v", path, err)
	}
	return record
}

func (f *factory) engine(t *testing.T) *runtime.Engine {
	t.Helper()

	specs := templates.New(
		sqlite.NewTemplateRepo(f.store), sqlite.NewLinkPolicyRepo(f.store), sqlite.NewPageRepo(f.store),
		sqlite.NewSiteRepo(f.store), f.store, silentPublisher{}, clock.System{},
	)

	registry := run.NewRegistry()
	if err := steps.Register(registry, steps.Deps{
		Entities: sqlite.NewEntityRepo(f.store),
		Edges:    sqlite.NewEdgeRepo(f.store),
		Pages:    sqlite.NewPageRepo(f.store),
		Policies: specs,
		Profiles: stubProfiles{},
		LLM:      f.llm,
	}); err != nil {
		t.Fatalf("register the steps: %v", err)
	}

	engine := runtime.New(runtime.Deps{
		Runs: f.runs, Items: f.items, Artifacts: f.blobs, Execs: sqlite.NewStepExecRepo(f.store),
		Events: f.log, Pages: sqlite.NewPageRepo(f.store), Specs: specs,
		Spend: sqlite.NewLLMCallRepo(f.store), Catalog: stubCatalog{}, Profiles: stubProfiles{},
		UnitOfWork: f.store, Publisher: f.bus,
	}, registry, runtime.Config{
		Workers: 1, PerSite: 1, SweepInterval: 20 * time.Millisecond,
		StepTimeout: 5 * time.Second, LeaseDuration: time.Second, RunDeadline: time.Hour,
	}, clock.System{}, zaptest.NewLogger(t))

	if err := engine.Start(t.Context()); err != nil {
		t.Fatalf("start the engine: %v", err)
	}
	t.Cleanup(engine.Stop)
	return engine
}

type silentPublisher struct{}

func (silentPublisher) Publish(events.Type, any) error {
	return nil
}

func (f *factory) waitForRun(t *testing.T, runID string, want run.Status) run.Run {
	t.Helper()

	deadline := time.Now().Add(pollTimeout)
	for time.Now().Before(deadline) {
		record, err := f.runs.Get(t.Context(), runID)
		if err == nil && record.Status == want {
			return record
		}
		if err == nil && record.Status.Terminal() {
			t.Fatalf("the run reached %q, want %q: %s", record.Status, want, record.Error)
		}
		time.Sleep(pollInterval)
	}
	t.Fatalf("timed out waiting for the run to reach %q", want)
	return run.Run{}
}

func (f *factory) artifact(t *testing.T, itemID, step string, kind run.ArtifactKind) run.Artifact {
	t.Helper()

	stored, err := f.blobs.ByItem(t.Context(), itemID)
	if err != nil {
		t.Fatalf("read the artifacts: %v", err)
	}
	for i := range stored {
		if stored[i].Kind == kind && stored[i].Step == step {
			return stored[i]
		}
	}
	t.Fatalf("step %s of the item produced no %s artifact", step, kind)
	return run.Artifact{}
}

func (f *factory) report(t *testing.T, itemID string) steps.ValidationReport {
	t.Helper()

	var report steps.ValidationReport
	if err := json.Unmarshal(f.artifact(t, itemID, steps.NameValidate, run.ArtifactValidationReport).Blob, &report); err != nil {
		t.Fatalf("decode the validation report: %v", err)
	}
	return report
}

func newID() string {
	return id.New()
}

func sqlitePages(f *factory) *sqlite.PageRepo {
	return sqlite.NewPageRepo(f.store)
}

func waitForItem(t *testing.T, f *factory, runID string, want run.Status) {
	t.Helper()

	deadline := time.Now().Add(pollTimeout)
	for time.Now().Before(deadline) {
		items, err := f.items.ByRun(t.Context(), runID)
		if err == nil && len(items) == 1 && items[0].Status == want {
			return
		}
		time.Sleep(pollInterval)
	}
	t.Fatalf("timed out waiting for the item to reach %q", want)
}
