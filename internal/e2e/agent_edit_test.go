//go:build e2e

package e2e_test

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/application/tools"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const agentSentence = "The agent wrote this sentence in place of the paragraph the run had written."

func TestTheAgentEditsABlock(t *testing.T) {
	live := newSite(t)
	requirePlugin(t, live.env, true)
	stand := writerStandOf(t, live.env)

	core := openCoreWith(t, fake.NewScripted(fake.Reply{Step: steps.NameGenerateBody, Make: bodyFromPrompt}))
	owner, err := core.Sites.Create(t.Context(), sites.CreateRequest{
		Name: "Agent's Shop", BaseURL: live.env.baseURL, Username: live.env.user, Password: live.env.pass, AllowInsecure: true,
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

	written := startRun(t, core, siteID, run.PublishLive, writing(steps.NamePublish), page.ID)
	awaitRun(t, core.Runs, written)
	runSync(t, core, siteID)
	ours, err := client.GetRaw(t.Context(), itemType, *page.WPID)
	if err != nil {
		t.Fatalf("read %s after the run: %v", stand.existing, err)
	}
	before := pageModelOf(t, core, page.ID)
	logBlocks(t, "after the run", core, page.ID)

	conversation := openConversation(t, core, siteID)
	listed := decoded[tools.PagesBlocksResponse](t, callTool(t, core, conversation, "pages_blocks", `{"op":"list","pageId":"`+page.ID+`"}`))
	if listed.Blocks == nil || listed.Blocks.BodyHash != block.BodyHash(ours.Content) {
		t.Fatalf("the agent listed %+v, want the body the site holds, by its hash", listed.Blocks)
	}
	target := -1
	for _, row := range listed.Blocks.Blocks.Items {
		if row.Owner == sitemodel.OwnerOurs && row.Kind != string(block.Heading) {
			target = row.Index
		}
	}
	if target < 0 {
		t.Fatalf("the agent found no block of ours on %s: %+v", stand.existing, listed.Blocks.Blocks.Items)
	}
	replaced := block.Parse(ours.Content).Blocks[target].Raw

	args := `{"op":"replace","pageId":"` + page.ID + `","index":` + strconv.Itoa(target) + `,"html":"<p>` + agentSentence +
		`</p>","hash":"` + listed.Blocks.BodyHash + `"}`
	asked := decoded[tools.Confirmation](t, callTool(t, core, conversation, "pages_blocks", args))
	if !strings.Contains(asked.Summary, "Replace block "+strconv.Itoa(target)+" of "+stand.existing) {
		t.Fatalf("the card says %q, want the block and the page it changes", asked.Summary)
	}
	if rawOf(t, client, itemType, *page.WPID) != ours.Content {
		t.Fatal("the edit reached the site before the person confirmed it")
	}
	edit := decoded[tools.PagesBlocksResponse](t, approveTheCard(t, core, conversation, "pages_blocks").Result)
	if edit.Edit == nil || edit.Edit.RunID == "" {
		t.Fatalf("the confirmed edit answered %+v, want its run", edit)
	}
	awaitRun(t, core.Runs, edit.Edit.RunID)

	edited, err := client.GetRaw(t.Context(), itemType, *page.WPID)
	if err != nil {
		t.Fatalf("read %s after the edit: %v", stand.existing, err)
	}
	if !strings.Contains(edited.Content, agentSentence) || strings.Contains(edited.Content, strings.TrimSpace(replaced)) {
		t.Fatalf("%s holds %q, want the agent's sentence in place of our block", stand.existing, edited.Content)
	}
	assertTheClientsBlocksStand(t, block.Parse(base.Content), block.Parse(edited.Content))
	runSync(t, core, siteID)
	logBlocks(t, "after the agent's edit", core, page.ID)
	assertOurLinksStayInTheContent(t, before, pageModelOf(t, core, page.ID))

	undone, err := core.Runs.Revert(t.Context(), runs.RevertRequest{RunID: edit.Edit.RunID})
	if err != nil {
		t.Fatalf("revert the agent's edit: %v", err)
	}
	awaitRun(t, core.Runs, undone.RunID)
	if back := rawOf(t, client, itemType, *page.WPID); back != ours.Content {
		t.Errorf("after the revert %s holds %q, want what the run wrote, byte for byte", stand.existing, back)
	}
	runSync(t, core, siteID)
	gone, err := core.Runs.Revert(t.Context(), runs.RevertRequest{RunID: written})
	if err != nil {
		t.Fatalf("revert the run: %v", err)
	}
	awaitRun(t, core.Runs, gone.RunID)
	if back := rawOf(t, client, itemType, *page.WPID); back != base.Content {
		t.Errorf("after both reverts %s holds %q, want the owner's base byte for byte", stand.existing, back)
	}
}

func rawOf(t *testing.T, client *wp.Client, itemType wp.ItemType, wpID int64) string {
	t.Helper()

	held, err := client.GetRaw(t.Context(), itemType, wpID)
	if err != nil {
		t.Fatalf("read the page %d back: %v", wpID, err)
	}
	return held.Content
}
