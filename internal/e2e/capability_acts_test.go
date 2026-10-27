//go:build e2e

package e2e_test

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

const (
	actDescription = "How to reconstitute a research peptide without shaking the vial, step by step."
	actEdited      = "Reconstitute a research peptide slowly: swirl the vial, never shake it, and keep it cold."
	actSentence    = "Every TB-500 vial ships frozen and is logged by the lab the day it arrives."
	actProposed    = "never shake the vial"
	actAnchor      = "the vial"
)

func keepTheMeta(t *testing.T, c *capability, client *wp.Client, path string, wpID int64) wp.SEOMeta {
	t.Helper()

	original, err := client.GetSEOMeta(t.Context(), wp.TypePage, wpID)
	if err != nil {
		t.Fatalf("read the SEO meta of %s: %v", path, err)
	}
	t.Cleanup(func() {
		ctx := context.WithoutCancel(t.Context())
		now, readErr := client.GetSEOMeta(ctx, wp.TypePage, wpID)
		if readErr != nil || now.Description == original.Description {
			return
		}
		if original.Description == "" {
			if _, cliErr := wpRun(ctx, repoRoot(t), c.live.env, "post", "meta", "delete", strconv.FormatInt(wpID, 10),
				"rank_math_description"); cliErr != nil {
				t.Errorf("clear the SEO description of %s again: %v", path, cliErr)
			}
			return
		}
		if _, putErr := client.SetSEOMeta(ctx, wp.TypePage, wpID, wp.SEOMeta{Description: original.Description}, nil); putErr != nil {
			t.Errorf("put the SEO description of %s back: %v", path, putErr)
		}
	})
	return original
}

func editRunsOn(t *testing.T, c *capability) map[string]string {
	t.Helper()

	listed, err := c.core.Runs.List(t.Context(), runs.ListRequest{SiteID: c.siteID, Kind: "edit", ListRequest: dto.ListRequest{Limit: 50}})
	if err != nil {
		t.Fatalf("list the edit runs: %v", err)
	}
	out := make(map[string]string, len(listed.Items))
	for i := range listed.Items {
		for target := range itemsByTarget(t, c.core.Runs, listed.Items[i].ID) {
			out[target] = listed.Items[i].ID
		}
	}
	return out
}

func blockHolding(t *testing.T, c *capability, conversation, pageID, text string) (index int, bodyHash string) {
	t.Helper()

	read := ask[tools.PagesBlocksResponse](t, c, conversation, "pages_blocks", map[string]any{"op": "list", "pageId": pageID, "text": true, "limit": 200})
	if read.Blocks == nil {
		t.Fatalf("pages_blocks list answered nothing for %s", pageID)
	}
	rows := read.Blocks.Blocks.Items
	for i := range rows {
		if strings.Contains(rows[i].Text, text) {
			return rows[i].Index, read.Blocks.BodyHash
		}
	}
	t.Fatalf("no block of %s holds %q: %+v; reset the stand", pageID, text, read.Blocks.Blocks.Items)
	return 0, ""
}

func TestTheAssistantEditsThePageThroughPlans(t *testing.T) {
	c := openCapability(t, fake.NewScripted(), withTheTree)
	client := wordpress(t, c.core, c.siteID)
	recon, tb, guides := c.page(t, capabilityRecon), c.page(t, capabilityTB), c.page(t, capabilityGuides)
	reconBody := keepOnSite(t, client, recon.Path, *recon.WPID)
	tbBody := keepOnSite(t, client, tb.Path, *tb.WPID)
	reconMeta := keepTheMeta(t, c, client, recon.Path, *recon.WPID)
	conversation := c.conversation(t)

	fields := ask[tools.PagesFieldsResponse](t, c, conversation, "pages_fields", map[string]any{
		"op": "get", "pageId": recon.ID, "keys": []string{"seo_description"},
	})
	hash := ""
	if fields.Fields != nil && len(fields.Fields.Fields.Items) == 1 {
		hash = fields.Fields.Fields.Items[0].Hash
	}
	listed := ask[tools.PagesBlocksResponse](t, c, conversation, "pages_blocks", map[string]any{"op": "list", "pageId": tb.ID})
	edited := proposeAPlan(t, c, conversation, "Describe the reconstitution guide and note how TB-500 ships",
		planStep{tool: "pages_fields", args: map[string]any{
			"op": "set", "pageId": recon.ID, "key": "seo_description", "value": actDescription, "hash": hash,
		}, reason: "The guide carries no description of its own for search."},
		planStep{tool: "pages_blocks", args: map[string]any{
			"op": "insert_after", "pageId": tb.ID, "index": 0, "html": "<p>" + actSentence + "</p>", "hash": listed.Blocks.BodyHash,
		}, reason: "The page says nothing of how the vial arrives."},
	)
	if edits := edited.Steps[0].Edits; len(edits) != 1 || edits[0].Name != "value" || edits[0].Value != actDescription {
		t.Fatalf("the field step offers the edits %+v, want its new value", edits)
	}
	if held := rawOf(t, client, wp.TypePage, *tb.WPID); held != tbBody {
		t.Fatal("the block reached the site before the person approved the plan")
	}
	approved := approveThePlan(t, c, conversation, edited, map[int]map[string]string{1: {"value": actEdited}})
	assertEveryStep(t, approved, domainagent.StepDone)
	if !strings.Contains(approved.Steps[0].Summary, actEdited) {
		t.Errorf("the edited field step says %q, want the value the person wrote", approved.Steps[0].Summary)
	}
	started := editRunsOn(t, c)
	for i, held := range approved.Steps {
		if held.RunID == "" {
			t.Errorf("step %d (%s) started an edit run and the plan names none, so no later step could wait for it", i+1, held.Tool)
		}
	}
	fieldRun, blockRun := started[recon.ID], started[tb.ID]
	if fieldRun == "" || blockRun == "" {
		t.Fatalf("the plan started the edit runs %v, want one on %s and one on %s", started, recon.Path, tb.Path)
	}
	awaitRun(t, c.core.Runs, fieldRun)
	awaitRun(t, c.core.Runs, blockRun)
	if meta, err := client.GetSEOMeta(t.Context(), wp.TypePage, *recon.WPID); err != nil || meta.Description != actEdited {
		t.Errorf("%s carries the SEO description %q (%v), want the one the person edited on the card", recon.Path, meta.Description, err)
	}
	inserted := rawOf(t, client, wp.TypePage, *tb.WPID)
	if !strings.Contains(inserted, actSentence) {
		t.Errorf("%s holds %q, want the inserted sentence", tb.Path, inserted)
	}
	assertTheClientsBlocksStand(t, block.Parse(tbBody), block.Parse(inserted))

	index, bodyHash := blockHolding(t, c, conversation, recon.ID, actProposed)
	linked := proposeAPlan(t, c, conversation, "Link the reconstitution guide up to the lab guides",
		planStep{tool: "pages_blocks", args: map[string]any{
			"op": "place_link", "pageId": recon.ID, "index": index, "targetId": guides.ID, "anchor": actProposed, "hash": bodyHash,
		}, reason: "The guide never links to the section it belongs to."},
	)
	if edits := linked.Steps[0].Edits; len(edits) != 1 || edits[0].Name != "anchor" || edits[0].Value != actProposed {
		t.Fatalf("the link step offers the edits %+v, want its anchor", edits)
	}
	assertEveryStep(t, approveThePlan(t, c, conversation, linked, map[int]map[string]string{1: {"anchor": actAnchor}}), domainagent.StepDone)
	linkRun := ""
	for target, runID := range editRunsOn(t, c) {
		if target == recon.ID && runID != fieldRun {
			linkRun = runID
		}
	}
	if linkRun == "" {
		t.Fatalf("the link plan started no edit run on %s", recon.Path)
	}
	awaitRun(t, c.core.Runs, linkRun)
	placed := rawOf(t, client, wp.TypePage, *recon.WPID)
	if anchors := anchorsIn(t, placed, capabilityGuides); !slices.Equal(anchors, []string{actAnchor}) {
		t.Errorf("%s links to %s as %q, want the anchor the person edited, %q:\n%s", recon.Path, capabilityGuides, anchors, actAnchor, placed)
	}
	assertTheClientsTextStays(t, reconBody, placed, actAnchor)

	undo := proposeAPlan(t, c, conversation, "Undo the three edits",
		planStep{tool: "runs_control", args: map[string]any{"op": "revert", "runId": linkRun},
			reason: "The link goes first, since it was the last write to the guide."},
		planStep{tool: "runs_control", args: map[string]any{"op": "revert", "runId": blockRun},
			reason: "The TB-500 page gets its own blocks back."},
		planStep{tool: "runs_control", args: map[string]any{"op": "revert", "runId": fieldRun},
			reason: "The guide's description goes back once its body is back.", dependsOn: []int{1}},
	)
	approvedUndo := approveThePlan(t, c, conversation, undo, nil)
	if approvedUndo.Steps[0].Status != domainagent.StepWaiting || approvedUndo.Steps[0].RunID == "" ||
		approvedUndo.Steps[2].Status != domainagent.StepPending {
		t.Errorf("the undo plan stands at %+v, want its first revert waited on and the third behind it", approvedUndo.Steps)
	}
	awaitRun(t, c.core.Runs, approvedUndo.Steps[0].RunID)
	awaitRun(t, c.core.Runs, approvedUndo.Steps[1].RunID)
	waitFor(t, "the third revert to start", func() bool { return planOf(t, c, conversation, undo.ID).Steps[2].RunID != "" })
	awaitRun(t, c.core.Runs, planOf(t, c, conversation, undo.ID).Steps[2].RunID)
	assertEveryStep(t, awaitPlan(t, c, conversation, undo.ID), domainagent.StepDone)
	told := toldThePlan(t, c, conversation, undo.Title)
	t.Logf("the conversation was told: %s", told)

	if back := rawOf(t, client, wp.TypePage, *recon.WPID); back != reconBody {
		t.Errorf("after the revert %s holds %q, want its own body byte for byte", recon.Path, back)
	}
	if back := rawOf(t, client, wp.TypePage, *tb.WPID); back != tbBody {
		t.Errorf("after the revert %s holds %q, want its own body byte for byte", tb.Path, back)
	}
	if meta, err := client.GetSEOMeta(t.Context(), wp.TypePage, *recon.WPID); err != nil || meta.Description != reconMeta.Description {
		t.Errorf("after the revert %s carries the SEO description %q (%v), want %q back", recon.Path, meta.Description, err, reconMeta.Description)
	}
}

func anchorsIn(t *testing.T, body, target string) []string {
	t.Helper()

	doc, err := content.Parse(body)
	if err != nil {
		t.Fatalf("parse the body: %v", err)
	}
	out := make([]string, 0)
	for _, link := range doc.Links() {
		if strings.HasSuffix(link.Href, target) {
			out = append(out, strings.Join(strings.Fields(link.Anchor), " "))
		}
	}
	return out
}

func assertTheClientsTextStays(t *testing.T, before, after, anchor string) {
	t.Helper()

	if textOfBody(t, before) != textOfBody(t, after) {
		t.Errorf("placing the link on %q changed the text of the page:\nbefore %q\nafter  %q", anchor, before, after)
	}
}

func textOfBody(t *testing.T, body string) string {
	t.Helper()

	doc, err := content.Parse(body)
	if err != nil {
		t.Fatalf("parse the body: %v", err)
	}
	return strings.Join(strings.Fields(doc.Text()), " ")
}
