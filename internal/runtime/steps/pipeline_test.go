package steps_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	appcontent "github.com/davidmovas/postulator/internal/application/content"
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
	guideDraft = `{"title":"How to pull espresso: Step-by-Step Guide | Shop",` +
		`"h1":"How to pull espresso",` +
		`"sections":[` +
		`{"heading":"Introduction","html":"<p>This espresso guide belongs to the coffee section of the drinks ` +
		`catalog we keep, and it takes about ten minutes to work through from start to finish.</p>"},` +
		`{"heading":"Step-by-Step Instructions","html":"<p>Grind fresh beans, dose the basket evenly, tamp the ` +
		`bed level and pull the shot for about twenty eight seconds into a warmed cup.</p>"},` +
		`{"heading":"Tips and Common Mistakes","html":"<p>Never tamp unevenly, never reuse a stale puck and ` +
		`never let the machine sit cold before the first shot of the morning.</p>"},` +
		`{"heading":"Frequently Asked Questions","html":"<p>A shot that runs fast is ground too coarse and a ` +
		`shot that chokes the pump is ground far too fine for the basket.</p>"}],` +
		`"summary":"A short guide to pulling a better shot at home."}`

	guideMeta = `{"title":"How to pull espresso: Step-by-Step Guide | Shop",` +
		`"description":"Grind, dose, tamp and pull a balanced shot at home in ten minutes.",` +
		`"canonical":"","ogTitle":"","ogDescription":""}`

	guideJudge = `{"score":0.9,"issues":[],"suggestions":["Add a photograph of the tamped bed."]}`

	grandparentBody = `<h1>Drinks</h1><p>Our drinks range runs from tea to espresso and everything between.</p>`
	coffeeBody      = `<h1>Coffee</h1><p>We roast for filter and for espresso, and we grind to order.</p>`
)

type pipeline struct {
	store   *sqlite.Store
	server  *wptest.Server
	runs    *sqlite.RunRepo
	items   *sqlite.RunItemRepo
	blobs   *sqlite.ArtifactRepo
	pages   *sqlite.PageRepo
	links   *sqlite.PageLinkRepo
	engine  *runtime.Engine
	siteID  string
	pageID  string
	parent  int64
	recipe  []template.StepSpec
	llm     *fake.Scripted
	current *clock.Fake
}

func guideSpec(t *testing.T) template.Template {
	t.Helper()

	seeds := template.Seed()
	for i := range seeds {
		if seeds[i].PageKind == "guide" {
			return seeds[i]
		}
	}
	t.Fatal("the seed templates carry no guide")
	return template.Template{}
}

func newPipeline(t *testing.T, opts ...wptest.Option) *pipeline {
	t.Helper()

	server := wptest.New(t, opts...)
	store := sqlitetest.Open(t)
	at := sqlitetest.Stamp

	guide := guideSpec(t)
	guide.ID = id.New()
	guide.CreatedAt, guide.UpdatedAt = at, at
	if err := sqlite.NewTemplateRepo(store).Insert(t.Context(), guide); err != nil {
		t.Fatalf("insert the guide template: %v", err)
	}

	policy := template.LinkPolicy{
		ID: id.New(), Scope: template.ScopeGlobal, Name: templates.DefaultPolicyName,
		Rules: guide.Spec.LinkRules, ForbidExternal: true, ForbidSelf: true,
		AnchorStrategy: template.AnchorPreferUser, CreatedAt: at, UpdatedAt: at,
	}
	if err := sqlite.NewLinkPolicyRepo(store).Insert(t.Context(), policy); err != nil {
		t.Fatalf("insert the link policy: %v", err)
	}

	owner := site.Site{
		ID: id.New(), Name: "Shop", BaseURL: server.URL(), Username: wptest.DefaultUser,
		Status: site.StatusActive, AllowInsecure: true, Plugin: site.PluginState{Capabilities: []string{}},
		Defaults: site.Defaults{
			TemplateID: &guide.ID, LinkPolicyID: &policy.ID,
			ModelProfiles: map[domainllm.Role]domainllm.ModelRef{},
		},
		CreatedAt: at, UpdatedAt: at,
	}
	owner.SecretRef = site.SecretRef(owner.ID)
	if err := sqlite.NewSiteRepo(store).Insert(t.Context(), owner); err != nil {
		t.Fatalf("insert the site: %v", err)
	}

	live := server.Seed(wptest.Item{
		Type: wptest.TypePage, Title: "Drinks", Slug: "drinks", Content: grandparentBody,
	})
	nested := server.Seed(wptest.Item{
		Type: wptest.TypePage, Title: "Coffee", Slug: "coffee", Parent: live[0].ID, Content: coffeeBody,
	})

	pageRepo := sqlite.NewPageRepo(store)
	entityRepo := sqlite.NewEntityRepo(store)
	edgeRepo := sqlite.NewEdgeRepo(store)

	grandparent := seedEntity(t, entityRepo, owner.ID, "Drinks", "drinks")
	parent := seedEntity(t, entityRepo, owner.ID, "Coffee", "coffee")
	child := seedEntity(t, entityRepo, owner.ID, "Espresso", "espresso")

	grandPage := publishedPage(t, pageRepo, owner.ID, "/drinks/", grandparent.ID, live[0].ID, nil)
	parentPage := publishedPage(t, pageRepo, owner.ID, "/drinks/coffee/", parent.ID, nested[0].ID, &grandPage.ID)
	childPage := seedPage(t, pageRepo, owner.ID, "/drinks/coffee/espresso/", child.ID, pagemap.StatusPlanned)
	childPage.ParentPageID = &parentPage.ID
	if err := pageRepo.Update(t.Context(), childPage); err != nil {
		t.Fatalf("attach the child to its parent: %v", err)
	}

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

	return &pipeline{
		store: store, server: server, siteID: owner.ID, pageID: childPage.ID,
		parent: nested[0].ID, recipe: guide.Spec.Recipe,
		runs:  sqlite.NewRunRepo(store),
		items: sqlite.NewRunItemRepo(store),
		blobs: sqlite.NewArtifactRepo(store),
		pages: pageRepo,
		links: sqlite.NewPageLinkRepo(store),
		llm: fake.NewScripted(map[string]string{
			steps.NameGenerateBody: guideDraft,
			steps.NameGenerateMeta: guideMeta,
			steps.NameJudge:        guideJudge,
			steps.NameRepairLinks:  repairSentence,
		}),
		current: clock.NewFake(at),
	}
}

func publishedPage(t *testing.T, repo *sqlite.PageRepo, siteID, path, entityID string,
	wpID int64, parentID *string) pagemap.Page {
	t.Helper()

	record := seedPage(t, repo, siteID, path, entityID, pagemap.StatusPublished)
	record.WPID = &wpID
	record.ParentPageID = parentID
	if err := repo.Update(t.Context(), record); err != nil {
		t.Fatalf("stamp the page %s: %v", path, err)
	}
	return record
}

func (p *pipeline) start(t *testing.T) *runtime.Engine {
	t.Helper()

	specs := templates.New(
		sqlite.NewTemplateRepo(p.store), sqlite.NewLinkPolicyRepo(p.store), p.pages,
		sqlite.NewSiteRepo(p.store), p.store, silentPublisher{}, clock.System{},
	)

	registry := run.NewRegistry()
	if err := steps.Register(registry, steps.Deps{
		Entities:      sqlite.NewEntityRepo(p.store),
		Edges:         sqlite.NewEdgeRepo(p.store),
		Pages:         p.pages,
		Links:         p.links,
		Sites:         sqlite.NewSiteRepo(p.store),
		SiteWriter:    sqlite.NewSiteRepo(p.store),
		WordPress:     oneClient{client: syncClient(t, p.server)},
		Policies:      specs,
		Profiles:      stubProfiles{},
		Content:       appcontent.New(appcontent.Deps{Profiles: stubProfiles{}, LLM: p.llm}),
		LLM:           p.llm,
		ImageProvider: &drawing{},
		UnitOfWork:    p.store,
		Publisher:     &busRecorder{},
		Clock:         p.current,
	}); err != nil {
		t.Fatalf("register the steps: %v", err)
	}

	engine := runtime.New(runtime.Deps{
		Runs: p.runs, Items: p.items, Artifacts: p.blobs, Execs: sqlite.NewStepExecRepo(p.store),
		Events: sqlite.NewRunEventRepo(p.store), Pages: p.pages, Specs: specs,
		Spend: sqlite.NewLLMCallRepo(p.store), Catalog: stubCatalog{}, Profiles: stubProfiles{},
		UnitOfWork: p.store, Publisher: &recorder{},
	}, registry, runtime.Config{
		Workers: 1, PerSite: 1, SweepInterval: 20 * time.Millisecond,
		StepTimeout: 20 * time.Second, LeaseDuration: time.Second, RunDeadline: time.Hour,
	}, clock.System{}, zaptest.NewLogger(t))

	if err := engine.Start(t.Context()); err != nil {
		t.Fatalf("start the engine: %v", err)
	}
	t.Cleanup(engine.Stop)
	p.engine = engine
	return engine
}

func (p *pipeline) generate(t *testing.T) run.Item {
	t.Helper()

	queued, err := p.engine.Enqueue(t.Context(), run.Run{
		ID: id.New(), SiteID: p.siteID, Kind: run.KindGenerate, Targets: []string{p.pageID},
		Recipe: p.recipe, TemplateID: "guide", TemplateVersion: 1, PublishMode: run.PublishDraft,
	})
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	deadline := time.Now().Add(pollTimeout)
	for time.Now().Before(deadline) {
		items, listErr := p.items.ByRun(t.Context(), queued.ID)
		if listErr == nil && len(items) == 1 && items[0].Status.Terminal() {
			if items[0].Status != run.StatusCompleted {
				t.Fatalf("the item reached %q: %s", items[0].Status, items[0].Error)
			}
			return items[0]
		}
		time.Sleep(pollInterval)
	}
	t.Fatal("timed out waiting for the pipeline to finish")
	return run.Item{}
}

func (p *pipeline) artifactOf(t *testing.T, itemID string, kind run.ArtifactKind, out any) {
	t.Helper()

	stored, err := p.blobs.ByItem(t.Context(), itemID)
	if err != nil {
		t.Fatalf("read the artifacts: %v", err)
	}
	for i := range stored {
		if stored[i].Kind == kind {
			if err = json.Unmarshal(stored[i].Blob, out); err != nil {
				t.Fatalf("decode the %s artifact: %v", kind, err)
			}
			return
		}
	}
	t.Fatalf("the item produced no %s artifact", kind)
}

func TestTheWholeRecipeReachesWordPress(t *testing.T) {
	t.Parallel()

	p := newPipeline(t)
	p.start(t)
	item := p.generate(t)

	var published steps.PublishResult
	p.artifactOf(t, item.ID, run.ArtifactPublishResult, &published)
	if published.WPID == 0 || !published.Created || published.Status != "draft" {
		t.Fatalf("publish result = %+v", published)
	}
	if !slices.Contains(published.SEOApplied, "title") || len(published.Skipped) != 0 {
		t.Fatalf("publish result = %+v", published)
	}

	draft, ok := p.server.Lookup(published.WPID)
	if !ok {
		t.Fatalf("the site holds no page %d", published.WPID)
	}
	if draft.Parent != p.parent || draft.Slug != "espresso" || draft.Status != "draft" {
		t.Fatalf("the draft is %+v", draft)
	}
	if !strings.Contains(draft.Content, `<a href="/drinks/coffee/">coffee</a>`) {
		t.Fatalf("the draft does not link to its parent:\n%s", draft.Content)
	}
	if !strings.Contains(draft.Content, `<a href="/drinks/">drinks</a>`) {
		t.Fatalf("the draft does not link to its grandparent:\n%s", draft.Content)
	}
	if draft.FeaturedMedia == 0 || !strings.Contains(draft.Content, "<figure><img src=") {
		t.Fatalf("the draft carries no images: %+v", draft)
	}
	if draft.Meta["_yoast_wpseo_title"] == "" || draft.Meta["_yoast_wpseo_metadesc"] == "" {
		t.Fatalf("the draft carries no seo meta: %+v", draft.Meta)
	}

	var report steps.ValidationReport
	p.artifactOf(t, item.ID, run.ArtifactValidationReport, &report)
	if report.Compliance.Score != 1 {
		t.Fatalf("compliance = %+v", report.Compliance)
	}

	var final steps.FinalReport
	p.artifactOf(t, item.ID, run.ArtifactFinalReport, &final)
	if final.Publish == nil || final.Relink == nil || final.Sync == nil || final.Judge == nil {
		t.Fatalf("final report = %+v", final)
	}
	if final.Path != "/drinks/coffee/espresso/" {
		t.Fatalf("final report = %+v", final)
	}

	parent, ok := p.server.Lookup(p.parent)
	if !ok || !strings.Contains(parent.Content, `<a href="/drinks/coffee/espresso/">espresso</a>`) {
		t.Fatalf("the parent did not gain a link to the child:\n%s", parent.Content)
	}

	stored, err := p.pages.Get(t.Context(), p.pageID)
	if err != nil {
		t.Fatalf("read the page: %v", err)
	}
	if stored.WPID == nil || *stored.WPID != published.WPID || stored.Status != pagemap.StatusExists {
		t.Fatalf("the page row is %+v", stored)
	}
	if stored.ContentHash == "" || stored.LastSyncedAt == nil || stored.Drift {
		t.Fatalf("the page row is %+v", stored)
	}

	links, err := p.links.ListForPage(t.Context(), p.pageID)
	if err != nil {
		t.Fatalf("list the links: %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("the child records %d internal links, want two", len(links))
	}
}

func TestASecondRunUpdatesTheSameDraft(t *testing.T) {
	t.Parallel()

	p := newPipeline(t)
	p.start(t)

	first := p.generate(t)
	var published steps.PublishResult
	p.artifactOf(t, first.ID, run.ArtifactPublishResult, &published)

	before := len(p.server.Items())
	second := p.generate(t)

	var again steps.PublishResult
	p.artifactOf(t, second.ID, run.ArtifactPublishResult, &again)
	if again.Created || again.WPID != published.WPID {
		t.Fatalf("the second run = %+v, want an update of %d", again, published.WPID)
	}
	if len(p.server.Items()) != before {
		t.Fatalf("the site holds %d items, want %d", len(p.server.Items()), before)
	}

	var relinked steps.RelinkResult
	p.artifactOf(t, second.ID, run.ArtifactRelinkResult, &relinked)
	if relinked.Linked != 0 || relinked.Conflicts != 0 {
		t.Fatalf("the second relink = %+v, want nothing left to place", relinked)
	}
}

func TestTheRecipeDegradesWithoutThePlugin(t *testing.T) {
	t.Parallel()

	p := newPipeline(t, wptest.WithoutPlugin())
	p.start(t)
	item := p.generate(t)

	var published steps.PublishResult
	p.artifactOf(t, item.ID, run.ArtifactPublishResult, &published)
	if len(published.SEOApplied) != 0 || !slices.Contains(published.Skipped, steps.CodeSEOMetaSkipped) {
		t.Fatalf("publish result = %+v", published)
	}

	draft, ok := p.server.Lookup(published.WPID)
	if !ok || draft.Meta["_yoast_wpseo_title"] != "" {
		t.Fatalf("the draft carries seo meta without the plugin: %+v", draft.Meta)
	}

	var relinked steps.RelinkResult
	p.artifactOf(t, item.ID, run.ArtifactRelinkResult, &relinked)
	for i := range relinked.Neighbors {
		if relinked.Neighbors[i].Outcome != steps.OutcomeSkipped {
			t.Fatalf("neighbor = %+v, want a clean skip without the plugin", relinked.Neighbors[i])
		}
	}

	var synced steps.SyncResult
	p.artifactOf(t, item.ID, run.ArtifactSyncResult, &synced)
	if synced.Source != "core" {
		t.Fatalf("sync result = %+v", synced)
	}
}
