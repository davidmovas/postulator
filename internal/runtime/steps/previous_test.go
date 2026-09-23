package steps_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const humanBody = `<h1>Espresso</h1><p>What a human wrote before the run.</p>`

func TestPublishKeepsTheBodyItReplaced(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Espresso", Status: "draft", Content: humanBody})

	published := runPublish(t, deps, publishContext(t))
	if published.Created {
		t.Fatalf("publish = %+v, want an update of the seeded page", published)
	}
	if published.PreviousContent != humanBody {
		t.Fatalf("previousContent = %q, want the body the publish replaced", published.PreviousContent)
	}
	if published.PreviousContentHash != wp.ContentHash(humanBody) {
		t.Fatalf("previousContentHash = %q, want the hash of the body it replaced", published.PreviousContentHash)
	}
}

func TestPublishKeepsNoPreviousBodyForAPageItCreated(t *testing.T) {
	t.Parallel()

	deps, _ := imageDeps(t)

	published := runPublish(t, deps, publishContext(t))
	if !published.Created {
		t.Fatalf("publish = %+v, want a creation", published)
	}
	if published.PreviousContent != "" || published.PreviousContentHash != "" {
		t.Fatalf("a created page reports a previous body: %q / %q",
			published.PreviousContent, published.PreviousContentHash)
	}
}

func TestPublishKeepsNoPreviousBodyWithoutThePlugin(t *testing.T) {
	t.Parallel()

	deps, server := imageDepsWith(t, wptest.WithoutPlugin())
	server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Espresso", Status: "draft", Content: humanBody})

	published := runPublish(t, deps, publishContext(t))
	if published.Created {
		t.Fatalf("publish = %+v, want an update of the seeded page", published)
	}
	if published.PreviousContent != "" || published.PreviousContentHash != "" {
		t.Fatalf("a site without the plugin reported a previous body: %q / %q",
			published.PreviousContent, published.PreviousContentHash)
	}
}

func TestRelinkKeepsTheNeighborBodyItReplaced(t *testing.T) {
	t.Parallel()

	deps, _, _, _ := relinkDeps(t, parentBody)

	relinked := runRelink(t, deps)
	if relinked.Linked != 1 || len(relinked.Neighbors) != 1 {
		t.Fatalf("relinked = %+v", relinked)
	}
	if relinked.Neighbors[0].Before.HTML != parentBody {
		t.Fatalf("before.html = %q, want the neighbor body the relink replaced", relinked.Neighbors[0].Before.HTML)
	}
	if relinked.Neighbors[0].Before.Hash != wp.ContentHash(parentBody) {
		t.Fatalf("before.hash = %q, want the hash of the body it replaced", relinked.Neighbors[0].Before.Hash)
	}
}

func TestRelinkKeepsNoBodyForANeighborItLeftAlone(t *testing.T) {
	t.Parallel()

	deps, _, _, _ := relinkDeps(t, parentBody+`<p>Try our <a href="/coffee/espresso/">espresso</a>.</p>`)

	relinked := runRelink(t, deps)
	if len(relinked.Neighbors) != 1 || relinked.Neighbors[0].Outcome != steps.OutcomeUnchanged {
		t.Fatalf("relinked = %+v", relinked)
	}
	if relinked.Neighbors[0].Before.HTML != "" || relinked.Neighbors[0].Before.Hash != "" {
		t.Fatalf("a neighbor nothing was written to reports a previous body: %+v", relinked.Neighbors[0].Before)
	}
}
