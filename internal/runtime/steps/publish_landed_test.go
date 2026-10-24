package steps_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const (
	linkedOurs  = `<p>Espresso is a way to make <a href="/coffee/">coffee</a>.</p>`
	linkedImage = `<figure><a href="/coffee/"><img src="/wp-content/uploads/roast.jpg" alt="The roastery"></a></figure>`
	linklessTwo = "<p>The owner's first paragraph.</p>\n\n<p>The owner's second paragraph.</p>"
)

func filteredKeptRun(t *testing.T, ours string) (steps.Deps, *wptest.Server, *bodyLog, *run.StepContext) {
	t.Helper()

	kept := &content.KeptBody{Raw: linklessTwo, Hash: block.BodyHash(linklessTwo)}
	return keptRunOverWith(t, []wptest.Option{wptest.WithSaveFilter(addingNoopener)}, linklessTwo, ours, kept)
}

func TestARevertFindsOurBlocksTheSiteReformatted(t *testing.T) {
	t.Parallel()

	deps, server, _, sc := filteredKeptRun(t, linkedOurs)
	published := runPublish(t, deps, sc)
	stored, _ := server.Lookup(published.WPID)
	if want := linklessTwo + "\n\n" + addingNoopener(linkedOurs); stored.Content != want {
		t.Fatalf("the publish stored %q, want ours after the owner's paragraphs, reformatted by the site", stored.Content)
	}
	recordedFor(t, &deps, sc.Page, published)

	held, _ := server.Lookup(published.WPID)
	held.Content = strings.Replace(held.Content, "The owner's second paragraph.", "The owner reworded his second paragraph.", 1)
	server.Restore(held)

	reverted, result := revertOf(t, deps, sc.Page.ID)
	if result.Next == run.TransitionPause || reverted.Outcome != steps.OutcomeRestored {
		t.Fatalf("the revert answered %+v / %s", reverted, result.Message)
	}
	want := "<p>The owner's first paragraph.</p>\n\n<p>The owner reworded his second paragraph.</p>"
	if after, _ := server.Lookup(published.WPID); after.Content != want {
		t.Errorf("after the revert the body is %q, want our reformatted block gone and the owner's later edit kept: %q", after.Content, want)
	}
}

func TestOurBlocksStayOursUnderARenderChangingFilter(t *testing.T) {
	t.Parallel()

	deps, server, log, sc := filteredKeptRun(t, linkedOurs+"\n\n"+linkedImage)
	published := runPublish(t, deps, sc)
	stored, _ := server.Lookup(published.WPID)
	held := block.Parse(stored.Content).Blocks
	if len(held) != 4 {
		t.Fatalf("the site stored %d blocks, want the owner's two and our two: %q", len(held), stored.Content)
	}
	ours := log.ours(sc.Page.ID)
	for _, at := range []int{2, 3} {
		if !slices.Contains(ours, held[at].Hash) {
			t.Errorf("our stored block %q is not recorded as ours: %v", held[at].Raw, ours)
		}
	}
	for _, at := range []int{0, 1} {
		if slices.Contains(ours, held[at].Hash) {
			t.Errorf("the owner's block %q is recorded as ours", held[at].Raw)
		}
	}
	landed := make([]string, 0, len(published.Blocks))
	for i := range published.Blocks {
		if published.Blocks[i].Owner == block.OwnerOurs && published.Blocks[i].Change != block.ChangeRemoved {
			landed = append(landed, published.Blocks[i].Hash)
		}
	}
	if !slices.Equal(landed, []string{held[2].Hash, held[3].Hash}) {
		t.Errorf("the publish result names %v as ours, want the hashes the site stored", landed)
	}
}
