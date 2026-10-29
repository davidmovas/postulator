//go:build e2e

package e2e_test

import (
	"encoding/json"
	"maps"
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/findings"
	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/application/sites"
	appsync "github.com/davidmovas/postulator/internal/application/sync"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

func proposedFromTheSite(t *testing.T, core *app.Core, siteID, path string) (graph.ProposedEntity, graph.Entity) {
	t.Helper()

	page, held := pagesByPath(t, core.Pages, siteID)[path]
	if !held || page.EntityID != nil {
		t.Fatalf("%s is not a page without an entity: %+v", path, page)
	}
	open, err := core.Findings.List(t.Context(), findings.ListRequest{
		SiteID: siteID, Code: string(finding.PageWithoutEntity), PageID: page.ID,
	})
	if err != nil || len(open.Items) != 1 {
		t.Fatalf("the page_without_entity findings of %s = %+v %v, want one", path, open.Items, err)
	}
	answered, err := core.Findings.Resolve(t.Context(), findings.ResolveRequest{
		SiteID: siteID, FindingID: open.Items[0].ID, Option: string(finding.ProposeEntity),
	})
	if err != nil || answered.Proposals == nil {
		t.Fatalf("resolve the finding of %s with propose_entity = %+v %v, want the site preview to open", path, answered, err)
	}
	preview, err := core.Graph.PreviewFromPages(t.Context(), *answered.Proposals)
	if err != nil || len(preview.Entities) != 1 || preview.Entities[0].Clash != "" {
		t.Fatalf("the site preview of %s = %+v %v, want one proposal without a clash", path, preview.Entities, err)
	}
	applied, err := core.Graph.ApplyProposals(t.Context(), graph.ApplyProposalsRequest{
		SiteID: siteID, Entities: preview.Entities, Source: graph.SourceSite,
	})
	if err != nil || applied.Mapped != 1 {
		t.Fatalf("accept the proposal of %s = %+v %v, want its page mapped", path, applied, err)
	}
	mapped := pagesByPath(t, core.Pages, siteID)[path]
	if mapped.EntityID == nil {
		t.Fatalf("%s carries no entity after its proposal was accepted", path)
	}
	written, err := core.Graph.GetEntity(t.Context(), graph.GetEntityRequest{ID: *mapped.EntityID})
	if err != nil {
		t.Fatalf("read the entity of %s: %v", path, err)
	}
	settled, err := core.Findings.Get(t.Context(), findings.GetRequest{SiteID: siteID, FindingID: open.Items[0].ID})
	if err != nil || settled.Status != string(finding.Resolved) {
		t.Errorf("the page_without_entity of %s after the proposal = %+v %v, want it resolved", path, settled, err)
	}
	return preview.Entities[0], written.Entity
}

func TestAProductWithoutAnEntityIsWrittenThroughAProposal(t *testing.T) {
	live := newSite(t)
	requirePlugin(t, live.env, true)
	requireStore(t, live)
	stand := writerStandOf(t, live.env)

	tag := strconv.FormatInt(time.Now().Unix()%1_000_000, 10)
	name := "Proposal Vial " + tag
	slug := "proposal-vial-" + tag
	planted := map[string]any{
		"name": name, "slug": slug, "type": "simple", "status": "publish", "sku": slug, "regular_price": "24.50",
		"description": clientDescription, "short_description": clientShort,
		"attributes": []map[string]any{{"name": "Size", "options": []string{"5 mg"}, "visible": true, "variation": false}},
	}
	if stand.productCategory != "" {
		var categories []struct {
			ID int `json:"id"`
		}
		live.call(t, http.MethodGet, "/wp-json/wc/v3/products/categories?slug="+stand.productCategory, nil, http.StatusOK, &categories)
		if len(categories) != 1 {
			t.Fatalf("the stand holds %d store categories named %s, want one", len(categories), stand.productCategory)
		}
		planted["categories"] = []map[string]any{{"id": categories[0].ID}}
	}
	var product struct {
		ID int `json:"id"`
	}
	live.call(t, http.MethodPost, "/wp-json/wc/v3/products", planted, http.StatusCreated, &product)
	live.removeLater(t, "/wp-json/wc/v3/products/"+strconv.Itoa(product.ID)+"?force=true")
	productPath := "/product/" + slug + "/"
	beforeFacts, _ := live.factsOf(t, product.ID)

	proxy := startRecordingProxy(t)
	core := openCoreThrough(t, fake.NewScripted(append(replies(),
		fake.Reply{Step: steps.NameGenerateBody, Match: "Path: " + productPath + "\n", Text: productAnswer("proposal vial")},
		fake.Reply{Step: steps.NameGenerateMeta, Match: "Path: " + productPath + "\n", Text: metaOf("proposal vial")},
		fake.Reply{Step: steps.NameGenerateFields, Match: "Path: " + productPath + "\n", Text: productFields("proposal vial")},
	)...), proxy)
	owner, err := core.Sites.Create(t.Context(), sites.CreateRequest{
		Name: "Proposal Store", BaseURL: live.env.baseURL, Username: live.env.user, Password: live.env.pass, AllowInsecure: true,
	})
	if err != nil {
		t.Fatalf("create the site: %v", err)
	}
	siteID := owner.Site.ID
	if checked, checkErr := core.Sync.CheckPlugin(t.Context(), appsync.CheckPluginRequest{SiteID: siteID}); checkErr != nil || checked.Commerce != "ready" {
		t.Fatalf("the check reports %+v (%v), want a store that can be edited", checked, checkErr)
	}
	runSync(t, core, siteID)
	clearTheSitesMap(t, core, siteID)
	seeded := seedStandGraph(t, core, siteID, stand.graph)

	page := pagesByPath(t, core.Pages, siteID)[productPath]
	if _, err = core.Runs.Estimate(t.Context(), runs.StartRequest{
		SiteID: siteID, PageIDs: []string{page.ID}, TemplateID: productTemplate(t, core), PublishMode: string(run.PublishLive), Recipe: recipe(),
	}); err == nil {
		t.Fatalf("a product no entity carries was estimated, want the steps to refuse it until a proposal is accepted")
	}

	proposal, entity := proposedFromTheSite(t, core, siteID, productPath)
	if proposal.Name != name || !slices.Contains(proposal.Facts, graph.Fact{Label: "Size", Value: "5 mg"}) {
		t.Errorf("the product's proposal = %+v, want its store name and its attribute as a fact", proposal)
	}
	if entity.Name != name || entity.Source != "user" {
		t.Errorf("the accepted entity = %+v, want the product's name, the person's", entity)
	}
	if stand.productHub != "" {
		hub := seeded[entityAt(stand.graph, stand.productHub)]
		if proposal.Parent != stand.productHub || entity.ScopeEntityID == nil || *entity.ScopeEntityID != hub {
			t.Errorf("the product's parent = %q, scope %v, want %s, the twin hub of its store category", proposal.Parent, entity.ScopeEntityID, stand.productHub)
		}
	}

	hubBefore := ""
	hubItem, hubHeld := contentItem{}, false
	if stand.productHub != "" {
		if hubItem, hubHeld = live.byPath(t, stand.productHub); !hubHeld {
			t.Fatalf("the stand holds no %s", stand.productHub)
		}
		hubBefore = live.storedContent(t, "pages", hubItem.ID)
	}

	started, err := core.Runs.Start(t.Context(), runs.StartRequest{
		SiteID: siteID, PageIDs: []string{page.ID}, Rewrite: []string{page.ID},
		TemplateID: productTemplate(t, core), PublishMode: string(run.PublishLive), Recipe: recipe(),
	})
	if err != nil {
		t.Fatalf("start the product's run: %v", err)
	}
	awaitRun(t, core.Runs, started.RunID)

	written := live.edited(t, product.ID)
	if !strings.Contains(written.Description, "keeps for a season") || strings.Contains(written.Description, clientDescription) {
		t.Errorf("the product's description = %q, want the run's", written.Description)
	}
	wireKeys := make([]string, 0)
	for _, record := range proxy.writes() {
		if !strings.Contains(record.Path, "/wc/v3/products") {
			continue
		}
		var body map[string]any
		if err = json.Unmarshal(record.Body, &body); err != nil {
			t.Errorf("a store write sent a body that is not JSON: %s", record.Body)
			continue
		}
		keys := slices.Sorted(maps.Keys(body))
		wireKeys = append(wireKeys, record.Method+" "+record.Path+" "+strings.Join(keys, ","))
		for _, denied := range deniedOnTheWire {
			if slices.Contains(keys, denied) {
				t.Errorf("%s %s sent %s, which the store owns: %s", record.Method, record.Path, denied, record.Body)
			}
		}
	}
	if afterFacts, _ := live.factsOf(t, product.ID); !reflect.DeepEqual(beforeFacts, afterFacts) {
		t.Errorf("the store's own facts moved:\nbefore %+v\nafter  %+v", beforeFacts, afterFacts)
	}
	t.Logf("%s got the entity %q under %q from the site's proposal, and its run wrote the description; the wire's store writes: %s",
		productPath, entity.Name, proposal.Parent, strings.Join(wireKeys, " | "))

	reverted, err := core.Runs.Revert(t.Context(), runs.RevertRequest{RunID: started.RunID})
	if err != nil {
		t.Fatalf("revert the product's run: %v", err)
	}
	awaitRun(t, core.Runs, reverted.RunID)
	if hubHeld {
		if after := live.storedContent(t, "pages", hubItem.ID); after != hubBefore {
			t.Errorf("after the revert %s holds %q, want the client's body the run's relink replaced, byte for byte", stand.productHub, after)
		}
	}
}
