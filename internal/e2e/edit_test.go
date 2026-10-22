//go:build e2e

package e2e_test

import (
	"context"
	"maps"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/observe"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/application/sites"
	appsync "github.com/davidmovas/postulator/internal/application/sync"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/render"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const clientFrame = `<iframe src="https://example.test/map" width="600" height="400"></iframe>`

func TestTheSiteFiltersAreRespectedUnderARestrictedUser(t *testing.T) {
	live := newSite(t)
	requirePlugin(t, live.env, true)
	if live.env.restrictedUser == "" {
		t.Skipf("the %s stand provisions no user without unfiltered_html; the replica does", live.env.stand)
	}
	stand := writerStandOf(t, live.env)

	core := openCoreWith(t, fake.NewScripted(fake.Reply{Step: steps.NameGenerateBody, Make: bodyFromPrompt}))
	owner, err := core.Sites.Create(t.Context(), sites.CreateRequest{
		Name: "Restricted Shop", BaseURL: live.env.baseURL, Username: live.env.restrictedUser,
		Password: live.env.restrictedPass, AllowInsecure: true,
	})
	if err != nil {
		t.Fatalf("create the site as the restricted user: %v", err)
	}
	siteID := owner.Site.ID
	checked, err := core.Sync.CheckPlugin(t.Context(), appsync.CheckPluginRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("check the plugin as the restricted user: %v", err)
	}
	if !checked.Plugin.Installed {
		t.Fatalf("the plugin answers nothing to the restricted user: %+v", checked)
	}
	runSync(t, core, siteID)
	known := pagesByPath(t, core.Pages, siteID)
	seedStandGraph(t, core, siteID, stand.graph)
	framed, plain := known[stand.edit.path], known[stand.kept]
	if framed.WPID == nil || plain.WPID == nil {
		t.Fatalf("%s or %s is not on the site; reset the stand", stand.edit.path, stand.kept)
	}

	framedID := strconv.FormatInt(*framed.WPID, 10)
	original := wpCLI(t, live.env, "post", "get", framedID, "--field=post_content")
	withFrame := original + "\n\n" + clientFrame
	wpCLI(t, live.env, "post", "update", framedID, "--post_content="+withFrame, "--user="+live.env.user)
	t.Cleanup(func() {
		wpCLI(t, live.env, "post", "update", framedID, "--post_content="+original, "--user="+live.env.user)
	})
	runSync(t, core, siteID)

	runID := startRun(t, core, siteID, run.PublishLive, writing(steps.NamePublish), framed.ID, plain.ID)
	awaitPause(t, core.Runs, runID)
	items := itemsByTarget(t, core.Runs, runID)
	held := items[framed.ID]
	if held.Status != string(run.StatusPaused) || !strings.Contains(held.Note, steps.ReasonSaveAltersClient) {
		t.Fatalf("the item for %s is %s with %q, want it paused because a save would change the client's content%s",
			framed.Path, held.Status, held.Note, itemStates(t, core.Runs, runID))
	}

	client := wordpress(t, core, siteID)
	stored, err := client.GetRaw(t.Context(), wp.TypePage, *framed.WPID)
	if err != nil {
		t.Fatalf("read %s back: %v", framed.Path, err)
	}
	if stored.Content != withFrame {
		t.Errorf("%s holds %q, want the client's body with its iframe byte for byte", framed.Path, stored.Content)
	}
	written, err := client.GetRaw(t.Context(), wp.TypePage, *plain.WPID)
	if err != nil {
		t.Fatalf("read %s back: %v", plain.Path, err)
	}
	if !strings.HasPrefix(written.Content, stand.keptText) || !strings.Contains(written.Content, ourText) {
		t.Errorf("%s holds %q, want the client's text kept and ours after it", plain.Path, written.Content)
	}
	if stored.SaveStable || !written.SaveStable {
		t.Errorf("save stability for the restricted user is %t for %s and %t for %s, want it false over the iframe only",
			stored.SaveStable, framed.Path, written.SaveStable, plain.Path)
	}
}

func TestAnExistingPageIsEditedBlockByBlock(t *testing.T) {
	live := newSite(t)
	requirePlugin(t, live.env, true)
	stand := writerStandOf(t, live.env)

	core := openCoreWith(t, fake.NewScripted(fake.Reply{Step: steps.NameGenerateBody, Make: bodyFromPrompt}))
	owner, err := core.Sites.Create(t.Context(), sites.CreateRequest{
		Name: "Edited Shop", BaseURL: live.env.baseURL, Username: live.env.user, Password: live.env.pass, AllowInsecure: true,
	})
	if err != nil {
		t.Fatalf("create the site: %v", err)
	}
	siteID := owner.Site.ID
	runSync(t, core, siteID)
	seedStandGraph(t, core, siteID, stand.graph)
	page := pagesByPath(t, core.Pages, siteID)[stand.existing]
	if page.WPID == nil {
		t.Fatalf("%s is not on the site; reset the stand", stand.existing)
	}
	client := wordpress(t, core, siteID)
	itemType := wp.ItemType(page.WPType)
	base, err := client.GetRaw(t.Context(), itemType, *page.WPID)
	if err != nil {
		t.Fatalf("read %s: %v", stand.existing, err)
	}
	t.Cleanup(func() {
		if held, readErr := client.GetRaw(context.WithoutCancel(t.Context()), itemType, *page.WPID); readErr == nil && held.Content != base.Content {
			wpCLI(t, live.env, "post", "update", strconv.FormatInt(*page.WPID, 10), "--post_content="+base.Content, "--user="+live.env.user)
		}
	})
	placed, err := client.GetItem(t.Context(), itemType, *page.WPID)
	if err != nil {
		t.Fatalf("read the identity of %s: %v", stand.existing, err)
	}
	before := pageModelOf(t, core, page.ID)
	logBlocks(t, "before", core, page.ID)

	runID := startRun(t, core, siteID, run.PublishLive, writing(steps.NamePublish), page.ID)
	awaitRun(t, core.Runs, runID)

	edited, err := client.GetRaw(t.Context(), itemType, *page.WPID)
	if err != nil {
		t.Fatalf("read %s back: %v", stand.existing, err)
	}
	if !strings.Contains(edited.Content, ourText) {
		t.Fatalf("%s holds %q, want the run's text in it", stand.existing, edited.Content)
	}
	assertTheClientsBlocksStand(t, block.Parse(base.Content), block.Parse(edited.Content))

	runSync(t, core, siteID)
	after := pageModelOf(t, core, page.ID)
	logBlocks(t, "after", core, page.ID)
	assertOurLinksStayInTheContent(t, before, after)
	if moved, movedErr := client.GetItem(t.Context(), itemType, *page.WPID); movedErr != nil || moved.Link != placed.Link || moved.Parent != placed.Parent {
		t.Errorf("%s moved from %s under %d to %s under %d (%v), want its address and parent kept",
			stand.existing, placed.Link, placed.Parent, moved.Link, moved.Parent, movedErr)
	}

	reverted, err := core.Runs.Revert(t.Context(), runs.RevertRequest{RunID: runID})
	if err != nil {
		t.Fatalf("revert the run: %v", err)
	}
	awaitRun(t, core.Runs, reverted.RunID)
	back, err := client.GetRaw(t.Context(), itemType, *page.WPID)
	if err != nil {
		t.Fatalf("read %s after the revert: %v", stand.existing, err)
	}
	runSync(t, core, siteID)
	logBlocks(t, "after the revert", core, page.ID)
	if back.Content != base.Content {
		t.Errorf("after the revert %s holds %q, want the base %q byte for byte", stand.existing, back.Content, base.Content)
	}
}

func pageModelOf(t *testing.T, core *app.Core, pageID string) observe.PageModel {
	t.Helper()

	model, err := core.Observe.Page(t.Context(), pageID)
	if err != nil {
		t.Fatalf("read the model of %s: %v", pageID, err)
	}
	return model
}

func logBlocks(t *testing.T, when string, core *app.Core, pageID string) {
	t.Helper()

	view, err := core.Observe.Model(t.Context(), observe.ModelRequest{PageID: pageID})
	if err != nil {
		t.Fatalf("read the blocks of %s: %v", pageID, err)
	}
	for _, held := range view.Blocks {
		owner := "the client's"
		if held.Ours {
			owner = "ours"
		}
		t.Logf("%s: block %d %s %s (%s): %q", when, held.Index, held.Kind, held.Name, owner, held.Excerpt)
	}
}

func assertTheClientsBlocksStand(t *testing.T, base, edited block.Body) {
	t.Helper()

	next := 0
	for _, kept := range base.Blocks {
		found := false
		for next < len(edited.Blocks) {
			candidate := edited.Blocks[next]
			next++
			if strings.TrimSpace(candidate.Raw) == strings.TrimSpace(kept.Raw) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("the client's block %q is not in the edited body byte for byte and in order, the whitespace between blocks aside", kept.Raw)
		}
	}
	if len(edited.Blocks) <= len(base.Blocks) {
		t.Fatalf("the edited body has %d blocks over the base's %d, want ours added", len(edited.Blocks), len(base.Blocks))
	}
}

func assertOurLinksStayInTheContent(t *testing.T, before, after observe.PageModel) {
	t.Helper()

	outside := func(model observe.PageModel) map[string]int {
		counted := map[string]int{}
		for i := range model.Links {
			if model.Links[i].Region != string(render.RegionContent) {
				counted[model.Links[i].Region+" "+model.Links[i].ToURL]++
			}
		}
		return counted
	}
	if was, now := outside(before), outside(after); !maps.Equal(was, now) {
		t.Errorf("the links outside the content moved:\nbefore %v\nafter  %v", was, now)
	}
	ours := 0
	for i := range after.Links {
		link := &after.Links[i]
		if !link.Ours {
			continue
		}
		ours++
		if link.Region != string(render.RegionContent) {
			t.Errorf("our link to %s sits in the %s region, want the content alone", link.ToURL, link.Region)
		}
	}
	t.Logf("%d links of ours, all in the content", ours)
	if before.Render == nil || after.Render == nil {
		t.Fatalf("the page has no render before (%t) or after (%t) the edit", before.Render != nil, after.Render != nil)
	}
	injected := func(record *render.Record) []string {
		out := make([]string, 0, len(record.Regions))
		for _, region := range record.Regions {
			if region.Kind != render.RegionContent && region.Kind != render.RegionGenerated {
				out = append(out, string(region.Kind))
			}
		}
		return out
	}
	if was, now := injected(before.Render), injected(after.Render); !slices.Equal(was, now) {
		t.Errorf("the regions the site injects moved: before %v, after %v", was, now)
	}
}
