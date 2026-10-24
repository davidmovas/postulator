package steps_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

func publishedAsStatus(sc *run.StepContext, status string) {
	blob := []byte(`{"wpId":99,"url":"https://shop.example.com/coffee/espresso/","status":"` + status + `"}`)
	sc.Artifacts[run.ArtifactPublishResult] = run.Artifact{Kind: run.ArtifactPublishResult, Blob: blob, Hash: run.HashBlob(blob)}
}

func relinkedIn(t *testing.T, result run.Result) steps.RelinkResult {
	t.Helper()
	for i := range result.Artifacts {
		if result.Artifacts[i].Kind != run.ArtifactRelinkResult {
			continue
		}
		var relinked steps.RelinkResult
		if err := json.Unmarshal(result.Artifacts[i].Blob, &relinked); err != nil {
			t.Fatalf("decode the relink result: %v", err)
		}
		return relinked
	}
	t.Fatalf("the step produced no relink result: %+v", result.Artifacts)
	return steps.RelinkResult{}
}

func TestADraftRunsNeighboursWaitForReview(t *testing.T) {
	t.Parallel()

	deps, server, _, wpID := relinkDeps(t, parentBody)
	sc := relinkContext(t, deps)
	sc.Run.PublishMode = run.PublishDraft
	sc.Item.CurrentStep = steps.NameRelinkNeighbors
	server.ResetRequests()

	result, err := drive(t, steps.RelinkNeighbors(deps), sc)
	if err != nil {
		t.Fatalf("RelinkNeighbors: %v", err)
	}
	if !awaitsReview(result) {
		t.Fatalf("result = %+v, want the neighbours' links held for a person to review", result)
	}
	if written := writesOf(server); len(written) != 0 {
		t.Fatalf("a draft run wrote to a neighbor: %+v", written[0])
	}
	plan, held := editPlanIn(t, result)
	if !held || len(plan.Pages) != 1 || plan.Pages[0].PageID != "page-parent" || len(plan.Pages[0].Blocks) == 0 {
		t.Fatalf("the review carries %+v, want the parent's block diff", plan)
	}

	accept(t, sc)
	result, err = drive(t, steps.RelinkNeighbors(deps), sc)
	if err != nil {
		t.Fatalf("RelinkNeighbors after the acceptance: %v", err)
	}
	if relinked := relinkedIn(t, result); relinked.Linked != 1 {
		t.Fatalf("after the acceptance = %+v, want the parent linked", relinked)
	}
	if stored, _ := server.Lookup(wpID); !strings.Contains(stored.Content, `href="/coffee/espresso/"`) {
		t.Errorf("the parent holds %q, want the accepted link", stored.Content)
	}
}

func TestADraftRelinkWaitsForReview(t *testing.T) {
	t.Parallel()

	deps, server, _, wpID := relinkPageDeps(t, espressoBody)
	sc := relinkPageContext(t, deps, wpID)
	sc.Run.PublishMode = run.PublishDraft
	sc.Item.CurrentStep = steps.NameRelinkPage
	server.ResetRequests()

	result, err := drive(t, steps.RelinkPage(deps), sc)
	if err != nil {
		t.Fatalf("RelinkPage: %v", err)
	}
	if !awaitsReview(result) {
		t.Fatalf("result = %+v, want the page's new links held for a person to review", result)
	}
	if written := writesOf(server); len(written) != 0 {
		t.Fatalf("a draft relink wrote to the page: %+v", written[0])
	}
	if stored, _ := server.Lookup(wpID); stored.Content != espressoBody {
		t.Errorf("the page holds %q, want it untouched until a person accepts", stored.Content)
	}
}

func TestNoLinkPlanTargetsADraftWeCreated(t *testing.T) {
	t.Parallel()

	deps, server, _, wpID := relinkDeps(t, parentBody)
	sc := relinkContext(t, deps)
	publishedAsStatus(sc, "draft")
	server.ResetRequests()

	result, err := drive(t, steps.RelinkNeighbors(deps), sc)
	if err != nil {
		t.Fatalf("RelinkNeighbors: %v", err)
	}
	if relinked := relinkedIn(t, result); relinked.Linked != 0 {
		t.Fatalf("relinked = %+v, want no neighbor linked to a page that is still a draft", relinked)
	}
	if written := writesOf(server); len(written) != 0 {
		t.Fatalf("a neighbor was written to link a draft: %+v", written[0])
	}
	if stored, _ := server.Lookup(wpID); stored.Content != parentBody {
		t.Errorf("the parent holds %q, want it untouched", stored.Content)
	}
}
