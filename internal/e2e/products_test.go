//go:build e2e

package e2e_test

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/application/sites"
	appsync "github.com/davidmovas/postulator/internal/application/sync"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

type storeAttribute struct {
	Name    string   `json:"name"`
	Options []string `json:"options"`
	ID      int      `json:"id"`
}

type storeProduct struct {
	Name             string           `json:"name"`
	Slug             string           `json:"slug"`
	Status           string           `json:"status"`
	SKU              string           `json:"sku"`
	RegularPrice     string           `json:"regular_price"`
	Description      string           `json:"description"`
	ShortDescription string           `json:"short_description"`
	Permalink        string           `json:"permalink"`
	Attributes       []storeAttribute `json:"attributes"`
	ID               int              `json:"id"`
}

const (
	clientDescription = "<p>The client's own description of the bottle.</p>"
	clientShort       = "<p>The client's short words.</p>"
)

func requireStore(t *testing.T, live *site) {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, live.env.baseURL+"/wp-json/wc/v3", http.NoBody)
	if err != nil {
		t.Fatalf("build the store probe: %v", err)
	}
	request.SetBasicAuth(live.env.user, live.env.pass)
	response, err := live.http.Do(request)
	if err != nil {
		t.Fatalf("probe the store: %v", err)
	}
	if closeErr := response.Body.Close(); closeErr != nil {
		t.Errorf("close the store probe: %v", closeErr)
	}
	if response.StatusCode != http.StatusOK {
		t.Skipf("the site answers no WooCommerce store (%d); provision the stack with E2E_WOO=1", response.StatusCode)
	}
}

func (s *site) createProduct(t *testing.T, name, slug string) storeProduct {
	t.Helper()

	var created storeProduct
	s.call(t, http.MethodPost, "/wp-json/wc/v3/products", map[string]any{
		"name": name, "slug": slug, "type": "simple", "status": "publish", "regular_price": "19.90", "sku": slug,
		"description": clientDescription, "short_description": clientShort,
		"attributes": []map[string]any{{"name": "Origin", "options": []string{"Client"}, "visible": true}},
	}, http.StatusCreated, &created)
	s.removeLater(t, "/wp-json/wc/v3/products/"+strconv.Itoa(created.ID)+"?force=true")
	return s.product(t, created.ID)
}

func (s *site) removeLater(t *testing.T, path string) {
	t.Helper()
	t.Cleanup(func() {
		request, err := http.NewRequestWithContext(context.WithoutCancel(t.Context()), http.MethodDelete, s.env.baseURL+path, http.NoBody)
		if err != nil {
			t.Errorf("build the removal of %s: %v", path, err)
			return
		}
		request.SetBasicAuth(s.env.user, s.env.pass)
		response, err := s.http.Do(request)
		if err != nil {
			t.Errorf("remove %s: %v", path, err)
			return
		}
		if closeErr := response.Body.Close(); closeErr != nil {
			t.Errorf("close the removal of %s: %v", path, closeErr)
		}
		if response.StatusCode != http.StatusOK {
			t.Errorf("remove %s: status %d", path, response.StatusCode)
		}
	})
}

func (s *site) product(t *testing.T, id int) storeProduct {
	t.Helper()

	var held storeProduct
	s.call(t, http.MethodGet, "/wp-json/wc/v3/products/"+strconv.Itoa(id)+"?context=edit", nil, http.StatusOK, &held)
	return held
}

func (s *site) dropPage(t *testing.T, id int) {
	t.Helper()
	s.removeLater(t, "/wp-json/wp/v2/pages/"+strconv.Itoa(id)+"?force=true")
}

func productAnswer(keyword string) string {
	return `{"title":"` + keyword + ` | Shop","h1":"` + keyword + `","sections":[` +
		`{"heading":"Overview","html":"<p>The ` + keyword + ` comes in one bottle and keeps for a season once it is open.</p>"},` +
		`{"heading":"Key Features","html":"<ul><li>A dropper cap that measures the dose.</li><li>A dark glass bottle.</li></ul>"},` +
		`{"heading":"Who It Is For","html":"<p>It suits the buyer who measures a dose every morning, and not the one who travels light.</p>"},` +
		`{"heading":"Frequently Asked Questions","html":"<p>It ships in two days and comes back within thirty days unopened.</p>"}],` +
		`"summary":"What the ` + keyword + ` is and who should buy it.",` +
		`"shortDescription":"<p>The ` + keyword + ` in a dark glass bottle with a dropper cap.</p>",` +
		`"specifications":[{"name":"Form","value":"Liquid"},{"name":"Size","value":""}]}`
}

func productTemplate(t *testing.T, core *app.Core) string {
	t.Helper()

	listed, err := core.Templates.ListTemplates(t.Context(), templates.ListTemplatesRequest{
		Scope: "global", ListRequest: dto.ListRequest{Limit: 20},
	})
	if err != nil {
		t.Fatalf("list the templates: %v", err)
	}
	for i := range listed.Items {
		if listed.Items[i].PageKind == "product" {
			return listed.Items[i].ID
		}
	}
	t.Fatal("the shipped product template is missing")
	return ""
}

func TestTheProductLoopEditsWhatTheClientCreatedInTheStore(t *testing.T) {
	live := newSite(t)
	requirePlugin(t, live.env, true)
	requireStore(t, live)

	tag := strconv.FormatInt(time.Now().Unix()%1_000_000, 10)
	hubSlug := "mak-" + tag
	hubPath := "/" + hubSlug + "/"
	hubID := live.publish(t, "Mak", hubSlug, "<h1>Mak</h1><p>The Mak range comes as a liquid and as capsules.</p>", 0)
	live.dropPage(t, hubID)
	liquid := live.createProduct(t, "Mak Liquid "+tag, "mak-liquid-"+tag)
	capsule := live.createProduct(t, "Mak Capsule "+tag, "capsule-x-"+tag)
	liquidPath, capsulePath := "/product/"+liquid.Slug+"/", "/product/"+capsule.Slug+"/"

	core := openCoreWith(t, fake.NewScripted(append(replies(),
		fake.Reply{Step: steps.NameGenerateBody, Match: liquidPath, Text: productAnswer("mak liquid")},
		fake.Reply{Step: steps.NameGenerateMeta, Match: liquidPath, Text: metaOf("mak liquid")},
		fake.Reply{Step: steps.NameGenerateBody, Match: capsulePath, Text: productAnswer("mak capsule")},
		fake.Reply{Step: steps.NameGenerateMeta, Match: capsulePath, Text: metaOf("mak capsule")},
	)...))
	owner, err := core.Sites.Create(t.Context(), sites.CreateRequest{
		Name: "Docker Store", BaseURL: live.env.baseURL, Username: live.env.user, Password: live.env.pass,
		AllowInsecure: true,
	})
	if err != nil {
		t.Fatalf("create the site: %v", err)
	}
	siteID := owner.Site.ID

	checked, err := core.Sync.CheckPlugin(t.Context(), appsync.CheckPluginRequest{SiteID: siteID})
	if err != nil || !checked.Plugin.Installed || checked.Commerce != "ready" {
		t.Fatalf("the check reports %+v (%v), want the plugin and a store that can be edited", checked, err)
	}
	synced, err := core.Sync.SyncSite(t.Context(), appsync.SyncSiteRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("sync the site: %v", err)
	}
	awaitRun(t, core.Runs, synced.RunID)

	sheet := filepath.Join(t.TempDir(), "products.csv")
	gelPath := hubPath + "mak-gel-" + tag + "/"
	body := "path,h1,keywords,page kind\n" +
		hubPath + ",Mak,mak range,hub\n" +
		hubPath + liquid.Slug + "/,Mak Liquid " + tag + ",mak liquid,\n" +
		hubPath + "capsule/,Mak Capsule " + tag + ",mak capsule,\n" +
		gelPath + ",Mak Gel " + tag + ",mak gel,\n"
	if err = os.WriteFile(sheet, []byte(body), 0o600); err != nil {
		t.Fatalf("write the sheet: %v", err)
	}
	mapping := imports.Mapping{SiteID: siteID, Name: "products", Columns: map[string]string{
		string(importmap.FieldPath): "path", string(importmap.FieldH1): "h1", string(importmap.FieldKeywords): "keywords",
		string(importmap.FieldPageKind): "page kind",
	}, Options: imports.Options{RowType: importmap.RowProducts}}

	applied, err := core.Imports.Apply(t.Context(), imports.ApplyRequest{SiteID: siteID, Path: sheet, Mapping: mapping})
	if err != nil || len(applied.Report.Errors) != 0 {
		t.Fatalf("apply the sheet: %v %+v", err, applied.Report.Errors)
	}
	waiting := 0
	for _, finding := range applied.Report.Warnings {
		if finding.Code == string(imports.CodeProductNotInStore) {
			waiting++
		}
	}
	if waiting != 1 {
		t.Fatalf("the import says %+v, want the gel row alone to wait for its product", applied.Report.Warnings)
	}

	stored := pagesByPath(t, core.Pages, siteID)
	for path, planned := range map[string]string{liquidPath: hubPath + liquid.Slug + "/", capsulePath: hubPath + "capsule/"} {
		page, ok := stored[path]
		if !ok || page.WPType != string(pagemap.WPProduct) || page.PlannedPath != planned || page.EntityID == nil {
			t.Fatalf("the import left %s as %+v, want the store's product planned from %s", path, page, planned)
		}
	}
	if row, ok := stored[gelPath]; !ok || row.WPID != nil || row.WPType != string(pagemap.WPProduct) {
		t.Fatalf("the gel row reads %+v, want it waiting for its product", row)
	}

	again, err := core.Imports.Apply(t.Context(), imports.ApplyRequest{SiteID: siteID, Path: sheet, Mapping: mapping})
	if err != nil || again.Counts.PagesCreated != 0 || again.Counts.PagesUpdated != 0 {
		t.Fatalf("a second import of the sheet = %+v (%v), want nothing to do", again.Counts, err)
	}

	started, err := core.Runs.Start(t.Context(), runs.StartRequest{
		SiteID: siteID, PageIDs: []string{stored[liquidPath].ID, stored[capsulePath].ID},
		TemplateID: productTemplate(t, core), PublishMode: string(run.PublishLive), Recipe: recipe(),
	})
	if err != nil {
		t.Fatalf("start the product run: %v", err)
	}
	awaitRun(t, core.Runs, started.RunID)
	items, err := core.Runs.ListItems(t.Context(), runs.ListItemsRequest{RunID: started.RunID, ListRequest: dto.ListRequest{Limit: 10}})
	if err != nil {
		t.Fatalf("list the run items: %v", err)
	}
	for i := range items.Items {
		if items.Items[i].Status != string(run.StatusCompleted) {
			t.Fatalf("the item for %s is %q: %s%s", items.Items[i].TargetID, items.Items[i].Status, items.Items[i].Error,
				artifactDump(t, core.Runs, items.Items[i].ID, string(run.ArtifactValidationReport)))
		}
	}

	for _, held := range []storeProduct{liquid, capsule} {
		written := live.product(t, held.ID)
		if !strings.Contains(written.Description, `href="`+hubPath+`"`) || strings.Contains(written.Description, "<h1") {
			t.Errorf("%s holds the description %q, want the run's, linked up to %s and without an h1", held.Name, written.Description, hubPath)
		}
		if !strings.Contains(strings.ToLower(written.ShortDescription), "dark glass bottle") {
			t.Errorf("%s holds the short description %q, want the run's", held.Name, written.ShortDescription)
		}
		if written.Name != held.Name || written.RegularPrice != held.RegularPrice || written.Status != held.Status ||
			written.SKU != held.SKU || written.Slug != held.Slug {
			t.Errorf("the run moved what the store owns: %+v, was %+v", written, held)
		}
		names := make([]string, 0, len(written.Attributes))
		for _, attribute := range written.Attributes {
			names = append(names, attribute.Name+"="+strings.Join(attribute.Options, "|"))
		}
		if !slices.Contains(names, "Origin=Client") || !slices.Contains(names, "Form=Liquid") || len(names) != 2 {
			t.Errorf("%s carries the attributes %v, want the client's kept and the form added", held.Name, names)
		}
	}
	if hub, ok := live.byPath(t, hubPath); !ok || !linksTo(hub.Links, liquidPath) {
		t.Errorf("the hub %s does not link to the product at its store address %s: %+v%s", hubPath, liquidPath, hub.Links,
			artifactDump(t, core.Runs, items.Items[0].ID, string(run.ArtifactRelinkResult)))
	}

	gel := live.createProduct(t, "Mak Gel "+tag, "mak-gel-"+tag)
	resynced, err := core.Sync.SyncSite(t.Context(), appsync.SyncSiteRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("sync the site again: %v", err)
	}
	awaitRun(t, core.Runs, resynced.RunID)
	claimed, ok := pagesByPath(t, core.Pages, siteID)["/product/"+gel.Slug+"/"]
	if !ok || claimed.ID != stored[gelPath].ID || claimed.PlannedPath != gelPath || claimed.WPID == nil {
		t.Errorf("the gel product created after the import reads %+v, want the row that waited for it", claimed)
	}

	reverted, err := core.Runs.Revert(t.Context(), runs.RevertRequest{RunID: started.RunID})
	if err != nil {
		t.Fatalf("revert the product run: %v", err)
	}
	awaitRun(t, core.Runs, reverted.RunID)
	for _, held := range []storeProduct{liquid, capsule} {
		back := live.product(t, held.ID)
		if back.Description != clientDescription || back.ShortDescription != clientShort {
			t.Errorf("%s holds %q / %q after the revert, want what the client wrote", held.Name, back.Description, back.ShortDescription)
		}
		if len(back.Attributes) != 1 || back.Attributes[0].Name != "Origin" {
			t.Errorf("%s carries %+v after the revert, want the client's attribute alone", held.Name, back.Attributes)
		}
	}
}
