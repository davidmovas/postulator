package steps_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

func publishedIn(t *testing.T, result run.Result) (steps.PublishResult, bool) {
	t.Helper()

	for i := range result.Artifacts {
		if result.Artifacts[i].Kind != run.ArtifactPublishResult {
			continue
		}
		var published steps.PublishResult
		if err := json.Unmarshal(result.Artifacts[i].Blob, &published); err != nil {
			t.Fatalf("decode the publish result: %v", err)
		}
		return published, true
	}
	return steps.PublishResult{}, false
}

func failTheWriteAfterTheBody(server *wptest.Server, id int64, onSite string) func(int) bool {
	failed := false
	return func(int) bool {
		if stored, _ := server.Lookup(id); !failed && stored.Content != onSite {
			server.FailNext(http.StatusBadRequest, 1)
			failed = true
		}
		return false
	}
}

func TestAPublishPausedAfterItsFirstWriteIsStillRevertible(t *testing.T) {
	t.Parallel()

	kept := &content.KeptBody{Raw: clientText, Hash: block.BodyHash(clientText)}
	deps, server, _, sc := keptRun(t, oursBody, kept)

	result, err := driveLosing(t, steps.Publish(deps), sc, failTheWriteAfterTheBody(server, *sc.Page.WPID, clientText))
	if err == nil {
		t.Fatalf("result = %+v, want the injected failure of the write after the body", result)
	}
	stored, _ := server.Lookup(*sc.Page.WPID)
	if stored.Content != clientText+"\n\n"+oursBody {
		t.Fatalf("the site holds %q, want the body written before the update failed", stored.Content)
	}
	published, held := publishedIn(t, result)
	if !held {
		t.Fatal("the failed execution carried no publish result, so a revert would find nothing to put back")
	}
	if published.PreviousContent != clientText || published.ContentHash != block.BodyHash(stored.Content) {
		t.Errorf("the record = previous %q hash %q, want the client's body and the hash of what the site holds",
			published.PreviousContent, published.ContentHash)
	}
}

func TestAWriteWhoseAnswerWasLostIsRecognisedOnTheRerun(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	deps.Bodies = &bodyLog{}
	seeded := server.Seed(wptest.Item{Type: "page", Title: "Espresso", Slug: "espresso", Status: "publish", Content: clientText})
	sc := rewritten(publishContext(t))
	sc.Page.WPID = &seeded[0].ID

	lostOnce := false
	result, err := driveLosing(t, steps.Publish(deps), sc, func(int) bool {
		stored, _ := server.Lookup(seeded[0].ID)
		if lostOnce || stored.Content == clientText {
			return false
		}
		lostOnce = true
		return true
	})
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if !lostOnce {
		t.Fatal("no execution wrote the body, so none could lose its answer")
	}
	published, held := publishedIn(t, result)
	if !held || result.Next == run.TransitionPause {
		t.Fatalf("result = %+v, want the publish to finish once the lost write is told apart", result)
	}
	if published.PreviousContent != clientText {
		t.Errorf("the rerun kept %q as what it replaced, want the client's body read before any write", published.PreviousContent)
	}
}

func TestACreationWhoseAnswerWasLostIsStillOurs(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	deps.Bodies = &bodyLog{}
	sc := publishContext(t)

	lostOnce := false
	result, err := driveLosing(t, steps.Publish(deps), sc, func(int) bool {
		if lostOnce || len(server.Items()) == 0 {
			return false
		}
		lostOnce = true
		return true
	})
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if !lostOnce {
		t.Fatal("no execution created the page, so none could lose its answer")
	}
	published, held := publishedIn(t, result)
	if !held || !published.Created {
		t.Fatalf("the publish = %+v, want the page it created recorded as created, so a revert trashes it", published)
	}
	if len(server.Items()) != 1 {
		t.Errorf("the site holds %d items, want the one page created once", len(server.Items()))
	}
}
