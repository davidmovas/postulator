//go:build e2e

package e2e_test

import (
	"io"
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/observe"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/application/sites"
	appsync "github.com/davidmovas/postulator/internal/application/sync"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/render"
)

func publicH1s(t *testing.T, address string) []string {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, address, http.NoBody)
	if err != nil {
		t.Fatalf("build the request for %s: %v", address, err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("visit %s: %v", address, err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read %s: %v", address, err)
	}
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("parse %s: %v", address, err)
	}
	var out []string
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "h1" {
			if text := strings.Join(strings.Fields(textOf(node)), " "); text != "" {
				out = append(out, text)
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	return out
}

func textOf(node *html.Node) string {
	if node.Type == html.TextNode {
		return node.Data
	}
	var b strings.Builder
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		b.WriteString(textOf(child))
		b.WriteString(" ")
	}
	return b.String()
}

func observedH1Of(model *observe.PageModel) string {
	for i := range model.Fields {
		if model.Fields[i].Key == field.H1 {
			return model.Fields[i].Current
		}
	}
	return ""
}

type h1Census struct {
	themed []string
	models map[string]*observe.PageModel
}

func assertEveryVisibleH1(t *testing.T, core *app.Core, siteID string) h1Census {
	t.Helper()

	stored := pagesByPath(t, core.Pages, siteID)
	paths := slices.Sorted(maps.Keys(stored))
	census := h1Census{models: make(map[string]*observe.PageModel, len(stored))}
	fields := make(map[string][]field.Value, len(stored))
	modeled := make([]pagemap.Page, 0, len(stored))
	for _, path := range paths {
		model, err := core.Observe.Page(t.Context(), stored[path].ID)
		if err != nil {
			t.Fatalf("read the model of %s: %v", path, err)
		}
		census.models[path] = &model
		fields[model.Page.ID] = model.Fields
		modeled = append(modeled, model.Page)
	}
	rows := pagemap.NewRowSite(modeled, fields, nil)

	checked := 0
	for _, path := range paths {
		listed, model := stored[path], census.models[path]
		if model.Render == nil || model.Identity.Status != "publish" || model.Render.FinalPath != "" {
			continue
		}
		visible := publicH1s(t, model.Identity.URL)
		h1 := observedH1Of(model)
		if len(visible) == 0 {
			if h1 != "" || listed.H1Source != string(render.H1None) {
				t.Errorf("%s shows no h1, and the model holds %q from %q", path, h1, listed.H1Source)
			}
			continue
		}
		checked++
		if !slices.Contains(visible, h1) {
			t.Errorf("%s shows the h1 %q, and the model holds %q", path, visible, h1)
			continue
		}
		if listed.ObservedH1 != h1 || (listed.H1Source != string(render.H1Theme) && listed.H1Source != string(render.H1Content)) {
			t.Errorf("%s lists the h1 %q from %q, want %q from the theme or the content", path, listed.ObservedH1, listed.H1Source, h1)
		}
		if listed.H1Source == string(render.H1Theme) {
			census.themed = append(census.themed, path)
		}
		drawer, err := core.Pages.Get(t.Context(), pages.GetRequest{ID: listed.ID})
		if err != nil || drawer.Page.ObservedH1 != h1 || drawer.Page.H1Source != listed.H1Source {
			t.Errorf("%s opens with the h1 %q from %q, %v; want %q from %q", path, drawer.Page.ObservedH1, drawer.Page.H1Source, err, h1, listed.H1Source)
		}
		agentView, err := core.SiteModel.Page(t.Context(), sitemodel.PageRequest{SiteID: siteID, PageID: listed.ID})
		if err != nil || agentView.H1 != h1 {
			t.Errorf("the agent sees %s with the h1 %q, %v; want %q", path, agentView.H1, err, h1)
		}
		matched := rows.Match(pagemap.Row{H1: h1, Product: model.Page.WPType == pagemap.WPProduct})
		answered := slices.ContainsFunc(matched.Candidates, func(p pagemap.Page) bool {
			return p.ID == listed.ID || (model.Page.WPType.Term() && observedH1Of(census.models[p.Path]) == h1)
		})
		if matched.Rule != pagemap.RuleH1 || !answered {
			t.Errorf("a row naming the h1 %q matched %d pages by %q, want %s, or the hub twin of a store category, by its H1",
				h1, len(matched.Candidates), matched.Rule, path)
		}
	}
	if checked == 0 {
		t.Fatal("no published page showed an h1, so nothing was proven")
	}
	t.Logf("%d published pages show an h1, %d of them printed by the theme: %v", checked, len(census.themed), census.themed)
	return census
}

func TestEveryPageShowsTheH1ItsRenderHolds(t *testing.T) {
	live := newSite(t)
	requirePlugin(t, live.env, true)

	core := openCore(t)
	owner, err := core.Sites.Create(t.Context(), sites.CreateRequest{
		Name: "H1 Shop", BaseURL: live.env.baseURL, Username: live.env.user, Password: live.env.pass, AllowInsecure: true,
	})
	if err != nil {
		t.Fatalf("create the site: %v", err)
	}
	siteID := owner.Site.ID
	synced, err := core.Sync.SyncSite(t.Context(), appsync.SyncSiteRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("sync the site: %v", err)
	}
	awaitRun(t, core.Runs, synced.RunID)
	census := assertEveryVisibleH1(t, core, siteID)
	if len(census.themed) == 0 {
		t.Fatalf("no page's h1 is printed by the theme on %s, so the fault the client met was not reproduced", live.env.stand)
	}

	renders, fieldRepo, pageRepo := sqlite.NewPageRenderRepo(core.Store), sqlite.NewPageFieldRepo(core.Store), sqlite.NewPageRepo(core.Store)
	for _, path := range census.themed {
		model := census.models[path]
		record := *model.Render
		record.H1, record.H1Source, record.ThemeH1 = "", render.H1Unknown, render.ThemeH1Printed
		if err = renders.Put(t.Context(), record); err != nil {
			t.Fatalf("rewind the render of %s: %v", path, err)
		}
		if err = fieldRepo.Observe(t.Context(), model.Page.ID, []field.Value{field.NewValue(model.Page.ID, field.H1, "", record.ExtractedAt)}); err != nil {
			t.Fatalf("rewind the h1 of %s: %v", path, err)
		}
		page, getErr := pageRepo.Get(t.Context(), model.Page.ID)
		if getErr != nil {
			t.Fatalf("read %s: %v", path, getErr)
		}
		page.H1 = ""
		if err = pageRepo.Update(t.Context(), page); err != nil {
			t.Fatalf("rewind the page of %s: %v", path, err)
		}
	}
	for _, path := range census.themed {
		if listed := pagesByPath(t, core.Pages, siteID)[path]; listed.ObservedH1 != "" {
			t.Fatalf("%s still lists %q after the rewind to 2.5.0", path, listed.ObservedH1)
		}
	}

	again, err := core.Sync.SyncSite(t.Context(), appsync.SyncSiteRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("sync the site again: %v", err)
	}
	awaitRun(t, core.Runs, again.RunID)
	after := assertEveryVisibleH1(t, core, siteID)
	if !slices.Equal(slices.Sorted(slices.Values(after.themed)), slices.Sorted(slices.Values(census.themed))) {
		t.Errorf("after an ordinary sync the theme prints the h1 of %v, want %v again", after.themed, census.themed)
	}
}
