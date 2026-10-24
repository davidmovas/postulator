package steps_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

func draftOverALivePage(t *testing.T) (steps.Deps, *wptest.Server, *run.StepContext) {
	t.Helper()

	deps, server := imageDeps(t)
	deps.Bodies = &bodyLog{}
	seeded := server.Seed(wptest.Item{Type: "page", Title: "Espresso", Slug: "espresso", Status: "publish", Content: clientText})
	sc := rewritten(publishContext(t))
	sc.Page.WPID = &seeded[0].ID
	sc.Page.Status = pagemap.StatusPublished
	sc.Run.PublishMode = run.PublishDraft
	sc.Item.CurrentStep = steps.NamePublish
	return deps, server, sc
}

func accept(t *testing.T, sc *run.StepContext) {
	t.Helper()
	if err := run.Set(sc.Check, run.CheckpointAccept, sc.Item.CurrentStep); err != nil {
		t.Fatalf("accept: %v", err)
	}
}

func writesOf(server *wptest.Server) []wptest.Request {
	out := make([]wptest.Request, 0)
	for _, request := range server.Requests() {
		if request.Method != http.MethodGet && request.Method != http.MethodHead && !strings.HasSuffix(request.Path, "/preview") {
			out = append(out, request)
		}
	}
	return out
}

func editPlanIn(t *testing.T, result run.Result) (steps.EditPlan, bool) {
	t.Helper()
	for i := range result.Artifacts {
		if result.Artifacts[i].Kind != run.ArtifactEditPlan {
			continue
		}
		var plan steps.EditPlan
		if err := json.Unmarshal(result.Artifacts[i].Blob, &plan); err != nil {
			t.Fatalf("decode the edit plan: %v", err)
		}
		return plan, true
	}
	return steps.EditPlan{}, false
}

func awaitsReview(result run.Result) bool {
	return result.Next == run.TransitionPause && result.Reason == run.PauseAwaitingReview
}

func TestADraftRunWritesNothingToAnExistingPage(t *testing.T) {
	t.Parallel()

	deps, server, sc := draftOverALivePage(t)
	server.ResetRequests()

	result, err := drive(t, steps.Publish(deps), sc)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if !awaitsReview(result) {
		t.Fatalf("result = %+v, want the page held for a person to review", result)
	}
	if written := writesOf(server); len(written) != 0 {
		t.Fatalf("a draft run sent %d writes to a live page: %+v", len(written), written[0])
	}
	plan, held := editPlanIn(t, result)
	if !held || len(plan.Pages) != 1 {
		t.Fatalf("the review carries %+v, want the page's diff", plan)
	}
	page := plan.Pages[0]
	if !hasFieldChange(page.Fields, field.SEOTitle) || hasFieldChange(page.Fields, field.Title) {
		t.Errorf("the diff's fields = %+v, want the SEO title's change and the title kept, its scope being off", page.Fields)
	}
	if !hasBlock(page.Blocks, block.ChangeRemoved, block.OwnerClient) || !hasBlock(page.Blocks, block.ChangeAdded, block.OwnerOurs) {
		t.Errorf("the diff's blocks = %+v, want the client's text replaced by ours, each marked", page.Blocks)
	}
	if stored, _ := server.Lookup(*sc.Page.WPID); stored.Content != clientText || stored.Status != "publish" {
		t.Errorf("the site holds %q (%s), want it untouched", stored.Content, stored.Status)
	}
}

func hasFieldChange(changes []field.Change, key field.Key) bool {
	for i := range changes {
		if changes[i].Key == key && changes[i].Moves() {
			return true
		}
	}
	return false
}

func hasBlock(changes []block.Change, kind block.ChangeKind, owner block.Owner) bool {
	for i := range changes {
		if changes[i].Change == kind && changes[i].Owner == owner {
			return true
		}
	}
	return false
}

func TestAnAcceptedReviewWritesAndTheItemCompletes(t *testing.T) {
	t.Parallel()

	deps, server, sc := draftOverALivePage(t)
	if result, err := drive(t, steps.Publish(deps), sc); err != nil || !awaitsReview(result) {
		t.Fatalf("Publish = %+v, %v; want a review first", result, err)
	}
	accept(t, sc)

	result, err := drive(t, steps.Publish(deps), sc)
	if err != nil {
		t.Fatalf("Publish after the acceptance: %v", err)
	}
	if result.Next == run.TransitionPause {
		t.Fatalf("result = %+v, want the accepted page written", result)
	}
	if _, held := publishedIn(t, result); !held {
		t.Fatal("the written page carries no publish result")
	}
	stored, _ := server.Lookup(*sc.Page.WPID)
	if want := strings.TrimPrefix(publishBody, "<h1>Espresso</h1>"); stored.Content != want || stored.Status != "publish" {
		t.Errorf("the site holds %q (%s), want our body, its H1 left to the theme, with the page still published", stored.Content, stored.Status)
	}
	if sc.Accepted() {
		t.Error("the acceptance is still on the item, so a later retry would be taken as accepted")
	}
}

func TestAReviewAcceptedAfterTheSiteMovedWaitsForReviewAgain(t *testing.T) {
	t.Parallel()

	deps, server, sc := draftOverALivePage(t)
	if result, err := drive(t, steps.Publish(deps), sc); err != nil || !awaitsReview(result) {
		t.Fatalf("Publish = %+v, %v; want a review first", result, err)
	}
	moved := clientText + "\n\n<p>Added by the owner after the review.</p>"
	server.Rewrite(*sc.Page.WPID, moved)
	accept(t, sc)
	server.ResetRequests()

	result, err := drive(t, steps.Publish(deps), sc)
	if err != nil {
		t.Fatalf("Publish after the acceptance: %v", err)
	}
	if !awaitsReview(result) {
		t.Fatalf("result = %+v, want the new plan held for review", result)
	}
	if written := writesOf(server); len(written) != 0 {
		t.Fatalf("an acceptance of an old plan wrote %+v", written[0])
	}
	if stored, _ := server.Lookup(*sc.Page.WPID); stored.Content != moved {
		t.Errorf("the site holds %q, want the owner's edit untouched", stored.Content)
	}
}

func TestAnAcceptNeverWritesOverAClientChange(t *testing.T) {
	t.Parallel()

	kept := &content.KeptBody{Raw: clientText, Hash: block.BodyHash(clientText)}
	deps, server, _, sc := keptRun(t, oursBody, kept)
	sc.Run.PublishMode = run.PublishDraft
	sc.Item.CurrentStep = steps.NamePublish
	if result, err := drive(t, steps.Publish(deps), sc); err != nil || !awaitsReview(result) {
		t.Fatalf("Publish = %+v, %v; want a review first", result, err)
	}
	edited := clientText + "<p>Added by the owner.</p>"
	server.Rewrite(*sc.Page.WPID, edited)
	accept(t, sc)
	server.ResetRequests()

	result, err := drive(t, steps.Publish(deps), sc)
	if err != nil {
		t.Fatalf("Publish after the acceptance: %v", err)
	}
	if result.Next != run.TransitionPause || result.Reason != run.PauseNeedsHuman || !strings.Contains(result.Message, steps.ReasonClientChanged) {
		t.Fatalf("result = %+v, want the owner's edit to stop the page", result)
	}
	if written := writesOf(server); len(written) != 0 {
		t.Fatalf("an acceptance wrote over the owner's edit: %+v", written[0])
	}
}

func TestADraftRunCreatesAPlannedPageAsADraft(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	deps.Bodies = &bodyLog{}
	sc := publishContext(t)
	sc.Run.PublishMode = run.PublishDraft

	published := runPublish(t, deps, sc)
	stored, ok := server.Lookup(published.WPID)
	if !published.Created || !ok || stored.Status != "draft" {
		t.Fatalf("the planned page = %+v on a site holding %+v, want it created as a draft without a review", published, stored)
	}
}

func TestNoStatusLeavesOnAnUpdate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		run  func(t *testing.T) *wptest.Server
	}{
		{name: "a live run over a live page", run: func(t *testing.T) *wptest.Server {
			deps, server, sc := draftOverALivePage(t)
			sc.Run.PublishMode = run.PublishLive
			server.ResetRequests()
			runPublish(t, deps, sc)
			return server
		}},
		{name: "a draft run a person accepted", run: func(t *testing.T) *wptest.Server {
			deps, server, sc := draftOverALivePage(t)
			if _, err := drive(t, steps.Publish(deps), sc); err != nil {
				t.Fatalf("Publish: %v", err)
			}
			accept(t, sc)
			server.ResetRequests()
			runPublish(t, deps, sc)
			return server
		}},
		{name: "a product", run: func(t *testing.T) *wptest.Server {
			h := newProductHarness(t, storeProduct())
			sc := storeContext(t, h.held.ID)
			h.server.ResetRequests()
			runPublish(t, h.deps, sc)
			return h.server
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := tc.run(t)
			written := writesOf(server)
			if len(written) == 0 {
				t.Fatal("nothing was written, so nothing proves the status stayed home")
			}
			for _, request := range written {
				var sent map[string]any
				if err := json.Unmarshal(request.Body, &sent); err != nil {
					continue
				}
				if _, carries := sent["status"]; carries {
					t.Errorf("%s %s carried a status: %s", request.Method, request.Path, request.Body)
				}
			}
		})
	}
}
