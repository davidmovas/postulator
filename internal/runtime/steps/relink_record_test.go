package steps_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

func driveRetrying(t *testing.T, def run.StepDef, sc *run.StepContext) (run.Result, error) {
	t.Helper()

	var (
		result run.Result
		err    error
	)
	for range 3 {
		if result, err = drive(t, def, sc); err == nil {
			return result, nil
		}
	}
	return result, err
}

func losingTheAnswerOfTheWrite(server *wptest.Server) {
	server.LoseTheAnswerOfTheNextRawWrite(http.StatusBadGateway)
}

const (
	clientParagraph = "<p>Fresh &amp; roasted&nbsp;daily, <em class='x'>always</em>.</p>"
	ourParagraph    = "<p>We roast every espresso blend we sell.</p>"
	ourLinked       = `<p>We roast every <a href="/coffee/espresso/">espresso</a> blend we sell.</p>`
)

func TestARelinkWhoseAnswerWasLostRecordsItsBlocks(t *testing.T) {
	t.Parallel()

	t.Run("a neighbor", func(t *testing.T) {
		t.Parallel()

		deps, server, _, _ := relinkDeps(t, clientParagraph+"\n\n"+ourParagraph)
		log := &bodyLog{written: map[string][]string{"page-parent": {block.Hash(ourParagraph)}}}
		deps.Bodies = log
		losingTheAnswerOfTheWrite(server)

		result, err := driveRetrying(t, steps.RelinkNeighbors(deps), relinkContext(t, deps))
		if err != nil {
			t.Fatalf("RelinkNeighbors: %v", err)
		}
		if relinked := relinkedIn(t, result); relinked.Linked != 1 || relinked.Conflicts != 0 {
			t.Errorf("relinked = %+v, want the neighbor counted as linked once its own write is found", relinked)
		}
		if written := log.ours("page-parent"); len(written) != 1 || written[0] != block.Hash(ourLinked) {
			t.Errorf("ours = %v, want the linked block recorded as ours, as a write whose answer came back records it", written)
		}
	})

	t.Run("the page itself", func(t *testing.T) {
		t.Parallel()

		ours := "<p>Espresso is the shortest way to make coffee at home.</p>"
		deps, server, _, wpID := relinkPageDeps(t, espressoBody)
		log := &bodyLog{written: map[string][]string{"page-child": {block.Hash(ours)}}}
		deps.Bodies = log
		losingTheAnswerOfTheWrite(server)

		result, err := driveRetrying(t, steps.RelinkPage(deps), relinkPageContext(t, deps, wpID))
		if err != nil {
			t.Fatalf("RelinkPage: %v", err)
		}
		if result.Next == run.TransitionPause {
			t.Fatalf("result = %+v, want the lost write found and the relink finished", result)
		}
		linked := `<p>Espresso is the shortest way to make <a href="/coffee/">coffee</a> at home.</p>`
		if written := log.ours("page-child"); len(written) != 1 || written[0] != block.Hash(linked) {
			t.Errorf("ours = %v, want the linked block recorded as ours", written)
		}
	})
}

func TestAdoptMarksTheLinksOfOurBlocksAsOurs(t *testing.T) {
	t.Parallel()

	theirs := `<p>We roast every espresso blend we sell, beside our <a href="/coffee/filter/">filter</a> range.</p>`
	deps, _, recorder, _ := relinkDeps(t, theirs)
	deps.Bodies = &bodyLog{}
	if _, err := drive(t, steps.RelinkNeighbors(deps), relinkContext(t, deps)); err != nil {
		t.Fatalf("RelinkNeighbors: %v", err)
	}

	byTarget := map[string]bool{}
	for _, link := range recorder.byPage["page-parent"] {
		byTarget[link.ToURL] = link.Ours
	}
	if ours, held := byTarget["/coffee/espresso/"]; !held || !ours {
		t.Errorf("the link the relink placed = %v (held %v), want it ours though it sits in the client's paragraph", ours, held)
	}
	if ours := byTarget["/coffee/filter/"]; ours {
		t.Error("the client's own link was recorded as ours")
	}
}

func TestARetriedRelinkPlacesNothingTwice(t *testing.T) {
	t.Parallel()

	t.Run("a neighbor", func(t *testing.T) {
		t.Parallel()

		bare := `<h1>Coffee</h1><p>We roast beans every week.</p><p>Visit the shop.</p>`
		deps, server, _, wpID := relinkDeps(t, bare)
		deps.Bodies = &bodyLog{}
		losingTheAnswerOfTheWrite(server)

		if _, err := driveRetrying(t, steps.RelinkNeighbors(deps), relinkContext(t, deps)); err != nil {
			t.Fatalf("RelinkNeighbors: %v", err)
		}
		stored, _ := server.Lookup(wpID)
		if got := strings.Count(stored.Content, "Read more about"); got != 1 {
			t.Fatalf("the neighbor holds %d plain sentences after a lost answer, want one:\n%s", got, stored.Content)
		}
	})

	t.Run("the page itself", func(t *testing.T) {
		t.Parallel()

		deps, server, _, wpID := relinkPageDeps(t, espressoBody)
		deps.Bodies = &bodyLog{}
		losingTheAnswerOfTheWrite(server)

		if _, err := driveRetrying(t, steps.RelinkPage(deps), relinkPageContext(t, deps, wpID)); err != nil {
			t.Fatalf("RelinkPage: %v", err)
		}
		stored, _ := server.Lookup(wpID)
		if got := strings.Count(stored.Content, `href="/coffee/"`); got != 1 {
			t.Fatalf("the page holds %d links to its parent after a lost answer, want one:\n%s", got, stored.Content)
		}
	})
}
