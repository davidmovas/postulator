//go:build e2e

package e2e_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/application/sites"
	appsync "github.com/davidmovas/postulator/internal/application/sync"
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
