//go:build e2e

package e2e_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/imports"
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

func importFiledPost(t *testing.T, core *app.Core, siteID string) pages.Page {
	t.Helper()

	sheet := filepath.Join(t.TempDir(), "posts.csv")
	rows := "Category,Path,Title,H1,Keywords,Post Type\n" +
		"TB-500," + filedPostPath + ",TB-500 dosing notes,TB-500 dosing notes," + filedPostTopic + "," + string(pagemap.WPPost) + "\n"
	if err := os.WriteFile(sheet, []byte(rows), 0o600); err != nil {
		t.Fatalf("write the sheet: %v", err)
	}
	seen, err := core.Imports.Inspect(t.Context(), imports.InspectRequest{SiteID: siteID, Path: sheet})
	if err != nil {
		t.Fatalf("inspect the sheet: %v", err)
	}
	applied, err := core.Imports.Apply(t.Context(), imports.ApplyRequest{SiteID: siteID, Path: sheet, Mapping: seen.Detected})
	if err != nil || len(applied.Report.Errors) != 0 {
		t.Fatalf("apply the sheet: %v %+v", err, applied.Report.Errors)
	}
	if applied.Counts.CategoriesCreated != 0 {
		t.Errorf("the sheet created %d categories, want TB-500 matched to the one the workbook made", applied.Counts.CategoriesCreated)
	}

	post := pagesByPath(t, core.Pages, siteID)[filedPostPath]
	if post.ID == "" || post.WPType != string(pagemap.WPPost) || post.WPID != nil {
		t.Fatalf("the import left %s as %+v, want a planned post", filedPostPath, post)
	}
	return post
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
	if liquid.ID == "" || liquid.WPID != nil {
		t.Fatalf("the import left %s as %+v, want a planned page", filedPath, liquid)
	}
	pageChain := liquid.Categories
	if !slices.Equal(namesOf(pageChain), []string{"TB-500", "Liquid"}) || !liquid.CategoriesNeedPlugin {
		t.Fatalf("%s is filed under %v (needs the plugin %t), want TB-500 and Liquid, which a page carries only with "+
			"the plugin", filedPath, namesOf(pageChain), liquid.CategoriesNeedPlugin)
	}
	post := importFiledPost(t, core, siteID)
	postChain := post.Categories
	if !slices.Equal(namesOf(postChain), []string{"TB-500"}) || !slices.Equal(recordIDsOf(postChain), recordIDsOf(pageChain[:1])) ||
		post.CategoriesNeedPlugin {
		t.Fatalf("the post is filed under %+v (needs the plugin %t), want the TB-500 record %s, which a post carries "+
			"without the plugin", postChain, post.CategoriesNeedPlugin, pageChain[0].ID)
	}
	categoryTemplate := templateOfKind(t, core, categoryKind)

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
	assertThePageShowsItsTerms(t, core, siteID, filedPostPath, filed.Categories)
	if carried := live.filed(t, "posts", filed.WPID).Categories; !sameSet(carried, termIDsOf(filed.Categories)) {
		t.Errorf("the post %s carries %v, want the chain %v", filedPostPath, carried, termIDsOf(filed.Categories))
	}
	if _, found := findingOf(filed.Findings, steps.CodePageCategoriesNeedPlugin); found {
		t.Errorf("the post %s warns %s, though a post carries categories without the plugin", filedPostPath,
			steps.CodePageCategoriesNeedPlugin)
	}
	t.Logf("%s went up without its chain %v and said %s; the post %s was filed under %v",
		filedPath, namesOf(pageChain), steps.CodePageCategoriesNeedPlugin, filedPostPath, termIDsOf(filed.Categories))
}
