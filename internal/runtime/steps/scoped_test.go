package steps_test

import (
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/pagetypes"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const sweptDescription = "Grind, dose, tamp and pull a balanced shot at home in ten minutes."

func (p *pipeline) onTheSite(t *testing.T) []pagemap.Page {
	t.Helper()

	tea := p.server.Seed(wptest.Item{
		Type: wptest.TypePage, Title: "Tea", Slug: "tea",
		Content: "<h1>Tea</h1><p>Green and black teas, weighed and brewed to order in the shop.</p>",
	})
	entity := seedEntity(t, sqlite.NewEntityRepo(p.store), p.siteID, "Tea", "tea")
	publishedPage(t, p.pages, p.siteID, "/tea/", entity.ID, tea[0].ID, nil)

	pages, err := p.pages.ListBySite(t.Context(), p.siteID)
	if err != nil {
		t.Fatalf("list the pages: %v", err)
	}
	return slices.DeleteFunc(pages, func(page pagemap.Page) bool { return page.WPID == nil })
}

func (p *pipeline) runOver(t *testing.T, record run.Run) []run.Item {
	t.Helper()

	queued, err := p.engine.Enqueue(t.Context(), record)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	deadline := time.Now().Add(pollTimeout)
	var items []run.Item
	for time.Now().Before(deadline) {
		var listErr error
		items, listErr = p.items.ByRun(t.Context(), queued.ID)
		if listErr == nil && len(items) == len(record.Targets) && !slices.ContainsFunc(items, func(item run.Item) bool {
			return !item.Status.Terminal()
		}) {
			for i := range items {
				if items[i].Status != run.StatusCompleted {
					t.Fatalf("the item of %s reached %q at %s: %s %s", items[i].TargetID, items[i].Status,
						items[i].CurrentStep, items[i].Error, items[i].Note)
				}
			}
			return items
		}
		for i := range items {
			if items[i].Status == run.StatusPaused {
				t.Fatalf("the item of %s paused at %s: %s", items[i].TargetID, items[i].CurrentStep, items[i].Note)
			}
		}
		time.Sleep(pollInterval)
	}
	t.Fatalf("timed out waiting for the run to finish; its items stand at %s", itemStates(items))
	return nil
}

func itemStates(items []run.Item) string {
	said := make([]string, 0, len(items))
	for i := range items {
		said = append(said, items[i].TargetID+" "+string(items[i].Status)+" at "+items[i].CurrentStep+
			" after "+strconv.Itoa(items[i].Attempts)+" attempts "+items[i].Error)
	}
	return strings.Join(said, "; ")
}

func (p *pipeline) writesTheTitle(t *testing.T) {
	t.Helper()

	listed, err := p.types.List(t.Context(), pagetypes.ListRequest{SiteID: p.siteID})
	if err != nil {
		t.Fatalf("list the page types: %v", err)
	}
	for i := range listed.Types {
		if !pagetype.Preset(listed.Types[i].Preset).Writable() {
			continue
		}
		if _, err = p.types.SetScope(t.Context(), pagetypes.SetScopeRequest{
			PageTypeID: listed.Types[i].ID, Field: string(field.Title), Enabled: true, AIWrites: true, Links: field.NoLinks(),
		}); err != nil {
			t.Fatalf("let the %s type's title be written: %v", listed.Types[i].Name, err)
		}
	}
}

func rewriting(recipe []template.StepSpec, pageIDs []string) []template.StepSpec {
	out := slices.Clone(recipe)
	for i := range out {
		if !out[i].Enabled {
			continue
		}
		params := maps.Clone(out[i].Params)
		params[run.ParamRewrite] = slices.Clone(pageIDs)
		out[i].Params = params
	}
	return out
}

func TestATitleSweepIsAnsweredByTheFieldsStepAlone(t *testing.T) {
	t.Parallel()

	p := newPipeline(t)
	p.llm = fake.NewScripted(fake.Reply{Step: steps.NameGenerateFields, Text: `{"fields":[{"key":"title","value":"Pulled to order"}]}`})
	targets := p.onTheSite(t)
	p.writesTheTitle(t)
	p.start(t)

	ids := make([]string, 0, len(targets))
	bodies := make(map[int64]string, len(targets))
	for i := range targets {
		ids = append(ids, targets[i].ID)
		held, _ := p.server.Lookup(*targets[i].WPID)
		bodies[held.ID] = held.Content
	}
	scoped, err := run.ScopeRecipe(p.recipe, []string{string(field.Title)})
	if err != nil {
		t.Fatalf("ScopeRecipe: %v", err)
	}
	items := p.runOver(t, run.Run{
		ID: id.New(), SiteID: p.siteID, Kind: run.KindGenerate, Targets: ids, Recipe: rewriting(scoped, ids),
		TemplateID: "guide", TemplateVersion: 1, PublishMode: run.PublishLive,
	})

	if calls := p.llm.CallsTo(steps.NameGenerateBody); calls != 0 {
		t.Errorf("the title sweep paid for %d body drafts, want none", calls)
	}
	if calls := p.llm.CallsTo(steps.NameGenerateFields); calls != len(ids) {
		t.Errorf("the title sweep paid for %d field answers, want one per page (%d)", calls, len(ids))
	}
	for _, request := range p.llm.Requests() {
		briefed := slices.ContainsFunc(request.Messages, func(message llm.Message) bool {
			return strings.Contains(message.Text, "workbook plans the title")
		})
		if request.Meta.Step == steps.NameGenerateFields && !briefed {
			t.Errorf("the fields step was not briefed with the workbook's title: %+v", request.Messages)
		}
	}
	for i := range targets {
		held, _ := p.server.Lookup(*targets[i].WPID)
		if held.Title != "Pulled to order" || held.Content != bodies[held.ID] {
			t.Errorf("%s is titled %q and holds %q, want the swept title over its own body", targets[i].Path, held.Title, held.Content)
		}
	}
	for i := range items {
		var published steps.PublishResult
		p.artifactOf(t, items[i].ID, run.ArtifactPublishResult, &published)
		if len(published.Fields) != 1 || published.Fields[0].Key != field.Title || len(published.Blocks) != 0 {
			t.Errorf("the title sweep's publish result is %+v, want the title alone", published.Fields)
		}
	}
}

func TestAShortDescriptionSweepLeavesTheProductsDescriptionAlone(t *testing.T) {
	t.Parallel()

	h := newProductHarness(t, storeProduct())
	sc := storeContext(t, h.held.ID)
	delete(sc.Artifacts, run.ArtifactBodyHTML)
	delete(sc.Artifacts, run.ArtifactDraft)
	sc.Params[run.ParamFields] = []string{string(field.ProductShortDescription)}
	before := len(h.server.Requests())

	published := runPublish(t, h.deps, sc)
	stored, _ := h.server.Lookup(h.held.ID)
	if stored.Content != "<p>old description</p>" {
		t.Errorf("the description is %q, want the store's own untouched", stored.Content)
	}
	if !strings.Contains(stored.Excerpt, "A compact machine for the counter.") {
		t.Errorf("the short description is %q, want the sweep's", stored.Excerpt)
	}
	keys := make([]field.Key, 0, len(published.Fields))
	for _, change := range published.Fields {
		keys = append(keys, change.Key)
	}
	if !slices.Equal(keys, []field.Key{field.ProductShortDescription}) || len(published.Blocks) != 0 {
		t.Errorf("the sweep's publish result names %v and %d blocks, want the short description alone", keys, len(published.Blocks))
	}
	for _, request := range h.server.Requests()[before:] {
		if strings.Contains(request.Path, "/raw/") && request.Method != http.MethodGet {
			t.Errorf("the sweep sent %s %s, want no description write", request.Method, request.Path)
		}
	}
}

func TestAMetaSweepIsNotHeldOnAPageASaveWouldChange(t *testing.T) {
	t.Parallel()

	p := newPipeline(t, wptest.WithSaveFilter(func(content string) string { return content + "\n" }))
	targets := p.onTheSite(t)
	p.start(t)

	ids := make([]string, 0, len(targets))
	for i := range targets {
		ids = append(ids, targets[i].ID)
	}
	scoped, err := run.ScopeRecipe(p.recipe, []string{string(field.SEODescription)})
	if err != nil {
		t.Fatalf("ScopeRecipe: %v", err)
	}
	p.runOver(t, run.Run{
		ID: id.New(), SiteID: p.siteID, Kind: run.KindGenerate, Targets: ids, Recipe: scoped,
		TemplateID: "guide", TemplateVersion: 1, PublishMode: run.PublishLive,
	})
	for i := range targets {
		held, _ := p.server.Lookup(*targets[i].WPID)
		if got := held.Meta["_yoast_wpseo_metadesc"]; got != sweptDescription {
			t.Errorf("%s carries the meta description %q, want the sweep's, which saves no post", targets[i].Path, got)
		}
	}
}

func TestAMetaSweepWritesOnlyTheMetaDescription(t *testing.T) {
	t.Parallel()

	p := newPipeline(t)
	targets := p.onTheSite(t)
	engine := p.start(t)

	ids := make([]string, 0, len(targets))
	bodies := make(map[int64]string, len(targets))
	titles := make(map[int64]string, len(targets))
	for i := range targets {
		ids = append(ids, targets[i].ID)
		held, _ := p.server.Lookup(*targets[i].WPID)
		bodies[held.ID], titles[held.ID] = held.Content, held.Title
	}
	scoped, err := run.ScopeRecipe(p.recipe, []string{string(field.SEODescription)})
	if err != nil {
		t.Fatalf("ScopeRecipe: %v", err)
	}
	whole := run.Run{SiteID: p.siteID, Kind: run.KindGenerate, Targets: ids, Recipe: p.recipe, PublishMode: run.PublishLive}
	sweep := run.Run{SiteID: p.siteID, Kind: run.KindGenerate, Targets: ids, Recipe: scoped, PublishMode: run.PublishLive}

	priced, err := engine.EstimateRun(t.Context(), whole, nil)
	if err != nil {
		t.Fatalf("EstimateRun of the whole recipe: %v", err)
	}
	swept, err := engine.EstimateRun(t.Context(), sweep, nil)
	if err != nil {
		t.Fatalf("EstimateRun of the sweep: %v", err)
	}
	t.Logf("over %d pages the sweep is priced at %d tokens ($%.4f), the whole recipe at %d tokens ($%.4f)",
		len(ids), swept.Tokens, swept.USD, priced.Tokens, priced.USD)
	if swept.Tokens == 0 || swept.Tokens >= priced.Tokens {
		t.Errorf("the sweep is priced at %d tokens against the whole recipe's %d, want it above nothing and below the whole",
			swept.Tokens, priced.Tokens)
	}
	for _, finding := range swept.Findings {
		if finding.Code == "recipe_differs" || finding.Code == "fields_scoped" {
			t.Errorf("the sweep's estimate says %+v, want no note about its scope", finding)
		}
	}

	written := len(p.server.Requests())
	sweep.ID, sweep.TemplateID, sweep.TemplateVersion = id.New(), "guide", 1
	items := p.runOver(t, sweep)

	for _, step := range []string{steps.NameGenerateBody, steps.NameGenerateFields, steps.NameJudge, steps.NameRepairLinks} {
		if calls := p.llm.CallsTo(step); calls != 0 {
			t.Errorf("the sweep paid for %d answers of %s, want none", calls, step)
		}
	}
	if calls := p.llm.CallsTo(steps.NameGenerateMeta); calls != len(ids) {
		t.Errorf("the sweep paid for %d meta answers, want one per page (%d)", calls, len(ids))
	}
	for _, request := range p.server.Requests()[written:] {
		raw := strings.Contains(request.Path, "/raw/")
		core := request.Method == http.MethodPost && strings.Contains(request.Path, "/wp/v2/") &&
			(strings.Contains(string(request.Body), `"content"`) || strings.Contains(string(request.Body), `"title"`))
		if (raw && request.Method != http.MethodGet) || core {
			t.Errorf("the sweep sent %s %s %s, want no body or title write", request.Method, request.Path, request.Body)
		}
	}

	bodyRepo := sqlite.NewPageBodyRepo(p.store)
	for i := range targets {
		held, _ := p.server.Lookup(*targets[i].WPID)
		if held.Content != bodies[held.ID] || held.Title != titles[held.ID] {
			t.Errorf("%s holds %q titled %q, want its body and title as they were", targets[i].Path, held.Content, held.Title)
		}
		if got := held.Meta["_yoast_wpseo_metadesc"]; got != sweptDescription {
			t.Errorf("%s carries the meta description %q, want the sweep's", targets[i].Path, got)
		}
		if got := held.Meta["_yoast_wpseo_title"]; got != "" {
			t.Errorf("%s carries the SEO title %q, want it left alone", targets[i].Path, got)
		}
		current, _, found, bodyErr := bodyRepo.Get(t.Context(), targets[i].ID)
		if bodyErr != nil || (found && current.Raw != bodies[held.ID]) {
			t.Errorf("%s stores the body %q (%v), want at most the site's own body read back", targets[i].Path, current.Raw, bodyErr)
		}
		if ours, writtenErr := bodyRepo.ListWritten(t.Context(), targets[i].ID); writtenErr != nil || len(ours) != 0 {
			t.Errorf("%s records %d blocks as ours (%v), want none from a sweep", targets[i].Path, len(ours), writtenErr)
		}
	}
	for i := range items {
		var published steps.PublishResult
		p.artifactOf(t, items[i].ID, run.ArtifactPublishResult, &published)
		keys := make([]field.Key, 0, len(published.Fields))
		for _, change := range published.Fields {
			keys = append(keys, change.Key)
		}
		if !slices.Equal(keys, []field.Key{field.SEODescription}) || len(published.Blocks) != 0 {
			t.Errorf("the sweep's publish result names %v and %d blocks, want the meta description alone", keys, len(published.Blocks))
		}
	}
}
