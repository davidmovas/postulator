//go:build e2e

package e2e_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const (
	filedPostPath  = "/" + filedPost + "/"
	filedPostTopic = "tb 500 dosing notes"
)

func entityNamed(t *testing.T, core *app.Core, siteID, name string) graph.Entity {
	t.Helper()

	byID := entitiesOf(t, core, siteID)
	found := make([]graph.Entity, 0, 1)
	for id := range byID {
		if byID[id].Name == name {
			found = append(found, byID[id])
		}
	}
	if len(found) != 1 {
		t.Fatalf("the graph holds %d entities named %q, want one", len(found), name)
	}
	return found[0]
}

func plantFiledPost(t *testing.T, core *app.Core, siteID, templateID string) pages.Page {
	t.Helper()

	under := entityNamed(t, core, siteID, "TB-500")
	created, err := core.Graph.CreateEntity(t.Context(), graph.CreateEntityRequest{
		SiteID: siteID, Name: "TB-500 Dosing Notes", Kind: "topic", Keywords: []dto.Keyword{{Text: filedPostTopic}},
		ParentID: under.ID,
	})
	if err != nil {
		t.Fatalf("create the post's entity under TB-500: %v", err)
	}

	entityID := created.Entity.ID
	planted, err := core.Pages.Create(t.Context(), pages.CreateRequest{
		SiteID: siteID, Path: filedPostPath, WPType: string(pagemap.WPPost), Title: "TB-500 dosing notes",
		EntityID: &entityID, TemplateID: &templateID,
	})
	if err != nil {
		t.Fatalf("plan the post %s: %v", filedPostPath, err)
	}
	return planted.Page
}

func TestTheWholeLoopDegradesWithoutThePluginAndFilesOnlyItsPosts(t *testing.T) {
	live := newSite(t)
	requirePlugin(t, live.env, false)
	live.sweepWorkbook(t)
	defer live.sweepWorkbook(t)

	peptidesID := live.publish(t, "Research peptides", "peptides", peptidesBody, 0)
	live.publish(t, "TB-500 peptide", "tb-500", tb500Body, peptidesID)

	core := openCoreWith(t, fake.NewScripted((&clientScript{}).replies()...))
	siteID, checked := openFilingSite(t, core, live, "Docker Peptides Without The Plugin")
	if checked.Plugin.Installed {
		t.Fatalf("the plugin check reports %+v, want no plugin", checked.Plugin)
	}
	importTheClientWorkbook(t, core, siteID)

	liquid := pagesByPath(t, core.Pages, siteID)[filedPath]
	if liquid.ID == "" || liquid.WPID != nil || liquid.EntityID == nil {
		t.Fatalf("the import left %s as %+v, want a planned page with its entity", filedPath, liquid)
	}
	pageChain := categoryChainOf(t, core, siteID, *liquid.EntityID)
	categoryTemplate := templateOfKind(t, core, categoryKind)
	post := plantFiledPost(t, core, siteID, categoryTemplate)
	postChain := categoryChainOf(t, core, siteID, *post.EntityID)
	if len(pageChain) == 0 || len(postChain) == 0 || postChain[len(postChain)-1] != "TB-500" {
		t.Fatalf("the page sits in %v and the post in %v, want both filed, the post under TB-500", pageChain, postChain)
	}

	request := runs.StartRequest{
		SiteID: siteID, PageIDs: []string{liquid.ID, post.ID}, TemplateID: categoryTemplate,
		PublishMode: string(run.PublishDraft),
	}
	priced, err := core.Runs.Estimate(t.Context(), request)
	if err != nil {
		t.Fatalf("estimate the run: %v", err)
	}
	warned := make([]string, 0, 1)
	for _, finding := range priced.Estimate.Findings {
		if finding.Code == steps.CodePageCategoriesNeedPlugin {
			warned = append(warned, finding.Path)
		}
	}
	if !slices.Equal(warned, []string{filedPath}) {
		t.Fatalf("the estimate warns %v of %s, want the page alone: %+v", warned, steps.CodePageCategoriesNeedPlugin,
			priced.Estimate.Findings)
	}

	started, err := core.Runs.Start(t.Context(), request)
	if err != nil {
		t.Fatalf("start the run: %v", err)
	}
	awaitRun(t, core.Runs, started.RunID)
	items, err := core.Runs.ListItems(t.Context(), runs.ListItemsRequest{RunID: started.RunID, ListRequest: dto.ListRequest{Limit: 5}})
	if err != nil || len(items.Items) != 2 {
		t.Fatalf("the run carries %+v (%v), want the page and the post", items.Items, err)
	}

	finals := make(map[string]steps.FinalReport, 2)
	for i := range items.Items {
		item := items.Items[i]
		if item.Status != string(run.StatusCompleted) {
			t.Fatalf("the item for %s is %q at %s: %s%s", item.TargetID, item.Status, item.CurrentStep, item.Error,
				artifactDump(t, core.Runs, item.ID, string(run.ArtifactValidationReport)))
		}
		final := finalReport(t, core.Runs, item.ID)
		if final.Publish == nil || final.Publish.Status != "draft" || final.Publish.WPID == 0 {
			t.Fatalf("%s reports %+v, want a draft on the site", final.Path, final.Publish)
		}
		finals[final.Path] = final
	}

	page := finals[filedPath].Publish
	if page.Categories != nil {
		t.Errorf("%s wrote the categories %+v without the plugin that lets a page carry them", filedPath, page.Categories)
	}
	if warning, found := findingOf(page.Findings, steps.CodePageCategoriesNeedPlugin); !found ||
		warning.Severity != content.SeverityWarn {
		t.Errorf("%s reports %+v, want a %s warning", filedPath, page.Findings, steps.CodePageCategoriesNeedPlugin)
	}

	filed := finals[filedPostPath].Publish
	if filed.Categories == nil {
		t.Fatalf("the post %s went up without its categories: %+v", filedPostPath, filed.Findings)
	}
	assertTheChainIsOnTheSite(t, live, wp.TaxonomyCategory, filedPostPath, filed.Categories, postChain)
	if carried := live.filed(t, "posts", filed.WPID).Categories; !sameSet(carried, termIDsOf(filed.Categories)) {
		t.Errorf("the post %s carries %v, want the chain %v", filedPostPath, carried, termIDsOf(filed.Categories))
	}
	if _, found := findingOf(filed.Findings, steps.CodePageCategoriesNeedPlugin); found {
		t.Errorf("the post %s warns %s, though a post carries categories without the plugin", filedPostPath,
			steps.CodePageCategoriesNeedPlugin)
	}
	t.Logf("%s went up without its chain %v and said %s; the post %s was filed under %v",
		filedPath, pageChain, steps.CodePageCategoriesNeedPlugin, filedPostPath, termIDsOf(filed.Categories))
}
