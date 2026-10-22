package steps_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/keyword"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
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

func addingNoopener(content string) string {
	return strings.ReplaceAll(content, `<a href=`, `<a rel="noopener" href=`)
}

func TestARelinkRecordsWhatTheSiteStored(t *testing.T) {
	t.Parallel()

	t.Run("a neighbor", func(t *testing.T) {
		t.Parallel()

		deps, server, recorder, wpID := relinkDeps(t, clientParagraph+"\n\n"+ourParagraph, wptest.WithSaveFilter(addingNoopener))
		log := &bodyLog{written: map[string][]string{"page-parent": {block.Hash(ourParagraph)}}}
		deps.Bodies = log
		relinked := runRelink(t, deps)

		stored, _ := server.Lookup(wpID)
		linked := clientParagraph + "\n\n" + addingNoopener(ourLinked)
		if stored.Content != linked {
			t.Fatalf("the parent holds %q, want the site's own form of the link", stored.Content)
		}
		if held := log.bodies["page-parent"]; held != stored.Content {
			t.Errorf("the body recorded is %q, want what the site stored %q", held, stored.Content)
		}
		if written := log.ours("page-parent"); len(written) != 1 || written[0] != block.Hash(addingNoopener(ourLinked)) {
			t.Errorf("ours = %v, want the stored form of the linked block", written)
		}
		if relinked.Neighbors[0].After != wp.ContentHash(stored.Content) {
			t.Errorf("after = %s, want the stored hash %s", relinked.Neighbors[0].After, wp.ContentHash(stored.Content))
		}
		if links := recorder.byPage["page-parent"]; len(links) != 1 || !links[0].Ours {
			t.Errorf("links = %+v, want the placed link recorded ours", links)
		}
	})

	t.Run("the page itself", func(t *testing.T) {
		t.Parallel()

		ours := "<p>Espresso is the shortest way to make coffee at home.</p>"
		deps, server, _, wpID := relinkPageDeps(t, espressoBody, wptest.WithSaveFilter(addingNoopener))
		log := &bodyLog{written: map[string][]string{"page-child": {block.Hash(ours)}}}
		deps.Bodies = log
		_, published, _ := runRelinkPage(t, deps, relinkPageContext(t, deps, wpID))

		stored, _ := server.Lookup(wpID)
		linked := `<p>Espresso is the shortest way to make <a rel="noopener" href="/coffee/">coffee</a> at home.</p>`
		if !strings.Contains(stored.Content, linked) {
			t.Fatalf("the page holds %q, want the site's own form of the link", stored.Content)
		}
		if held := log.bodies["page-child"]; held != stored.Content {
			t.Errorf("the body recorded is %q, want what the site stored %q", held, stored.Content)
		}
		if written := log.ours("page-child"); len(written) != 1 || written[0] != block.Hash(linked) {
			t.Errorf("ours = %v, want the stored form of the linked block", written)
		}
		if published.ContentHash != wp.ContentHash(stored.Content) {
			t.Errorf("contentHash = %s, want the stored hash %s", published.ContentHash, wp.ContentHash(stored.Content))
		}
	})
}

func TestARelinkWritesNothingOverAPageItsSaveWouldAlter(t *testing.T) {
	t.Parallel()

	t.Run("a neighbor", func(t *testing.T) {
		t.Parallel()

		body := "<p>Fresh roasted<br>daily, with every espresso blend we sell.</p>"
		deps, server, _, wpID := relinkDeps(t, body, wptest.WithFilteredHTML())
		deps.Bodies = &bodyLog{}
		relinked := runRelink(t, deps)

		if stored, _ := server.Lookup(wpID); stored.Content != body {
			t.Fatalf("the neighbor was written: %q", stored.Content)
		}
		if relinked.Linked != 0 || relinked.Skipped != 1 || len(relinked.Neighbors) != 1 {
			t.Fatalf("relinked = %+v, want the neighbor skipped", relinked)
		}
		if got := relinked.Neighbors[0]; got.Outcome != steps.OutcomeSkipped || got.Detail != steps.ReasonNeighborSaveAlters {
			t.Errorf("neighbor = %+v, want it skipped because its save would alter the client's content", got)
		}
	})

	t.Run("the page itself", func(t *testing.T) {
		t.Parallel()

		body := `<h1>Espresso</h1><p>Espresso is the shortest<br>way to make coffee at home.</p>`
		deps, server, _, wpID := relinkPageDeps(t, body, wptest.WithFilteredHTML())
		deps.Bodies = &bodyLog{}
		result, err := drive(t, steps.RelinkPage(deps), relinkPageContext(t, deps, wpID))
		if err != nil {
			t.Fatalf("RelinkPage: %v", err)
		}
		if result.Next != run.TransitionPause || !strings.Contains(result.Message, steps.ReasonSaveAltersClient) {
			t.Fatalf("result = %+v, want the page paused because its save would alter the client's content", result)
		}
		if stored, _ := server.Lookup(wpID); stored.Content != body {
			t.Fatalf("the page was written: %q", stored.Content)
		}
	})
}

func TestAdoptMarksTheLinksOfOurBlocksAsOurs(t *testing.T) {
	t.Parallel()

	theirs := `<p>We roast every espresso blend we sell, beside our <a href="/coffee/filter/">filter</a> range.</p>`
	deps, server, recorder, wpID := relinkDeps(t, theirs)
	deps.Bodies = &bodyLog{}
	if _, err := drive(t, steps.RelinkNeighbors(deps), relinkContext(t, deps)); err != nil {
		t.Fatalf("RelinkNeighbors: %v", err)
	}

	stored, _ := server.Lookup(wpID)
	byTarget := map[string]pagemap.PageLink{}
	for _, link := range recorder.byPage["page-parent"] {
		byTarget[link.ToURL] = link
	}
	if placed, held := byTarget["/coffee/espresso/"]; !held || !placed.Ours || placed.BlockHash != block.Parse(stored.Content).Blocks[0].Hash {
		t.Errorf("the link the relink placed = %+v (held %v), want it ours, keyed to the client's paragraph as the site stored it", placed, held)
	}
	if his := byTarget["/coffee/filter/"]; his.Ours || his.BlockHash != "" {
		t.Errorf("the client's own link was recorded as ours: %+v", his)
	}
}

func withRistretto(deps steps.Deps, wpID int64) steps.Deps {
	deps.Entities = entityList{items: append(unitEntities(), graph.Entity{
		ID: "ristretto", SiteID: "site", Name: "Ristretto", Keywords: keyword.Of("ristretto"),
		Anchors: []graph.Anchor{{Text: "ristretto", Source: graph.AnchorUser, Weight: 1}},
		Source:  graph.SourceUser, CanonicalPageID: pointer("page-ristretto"),
	})}
	deps.Edges = edgeList{items: append(unitEdges(), graph.Edge{
		ID: "e2", SiteID: "site", FromEntityID: "ristretto", ToEntityID: "parent",
		Kind: graph.EdgeParent, Weight: 1, Source: graph.SourceUser, Status: graph.StatusApproved,
	})}
	deps.Pages = pageList{items: append(relinkPages(wpID), ristrettoPage())}
	return deps
}

func ristrettoPage() pagemap.Page {
	return pagemap.Page{
		ID: "page-ristretto", SiteID: "site", Path: "/coffee/ristretto/", Slug: "ristretto",
		WPType: pagemap.WPPage, Status: pagemap.StatusExists, EntityID: pointer("ristretto"),
	}
}

func linkContextFor(t *testing.T, deps steps.Deps, page pagemap.Page) []byte {
	t.Helper()

	sc := unitContext(t, nil)
	sc.Page, sc.Item.TargetID = page, page.ID
	result, err := steps.ResolveContext(deps).Run(t.Context(), sc)
	if err != nil {
		t.Fatalf("ResolveContext for %s: %v", page.Path, err)
	}
	return result.Artifacts[0].Blob
}

func TestASecondRelinkIntoTheSameParagraphKeepsTheFirstLinkOurs(t *testing.T) {
	t.Parallel()

	deps, server, recorder, wpID := relinkDeps(t, "<p>We roast every espresso and ristretto blend we sell.</p>")
	deps = withRistretto(deps, wpID)
	deps.Bodies = &bodyLog{}
	runRelink(t, deps)
	onceLinked, _ := server.Lookup(wpID)
	first := recorder.byPage["page-parent"]
	if len(first) != 1 || !first[0].Ours || first[0].BlockHash != block.Parse(onceLinked.Content).Blocks[0].Hash {
		t.Fatalf("after the first relink the links are %+v, want the espresso link ours in its block", first)
	}

	sc := unitContext(t, map[run.ArtifactKind][]byte{
		run.ArtifactLinkContext:   linkContextFor(t, deps, ristrettoPage()),
		run.ArtifactPublishResult: []byte(`{"wpId":98,"url":"https://shop.example.com/coffee/ristretto/","status":"publish"}`),
	})
	sc.Run.ID, sc.Page, sc.Item.TargetID = "run-b", ristrettoPage(), "page-ristretto"
	if _, err := drive(t, steps.RelinkNeighbors(deps), sc); err != nil {
		t.Fatalf("the second relink: %v", err)
	}

	twiceLinked, _ := server.Lookup(wpID)
	if !strings.Contains(twiceLinked.Content, `href="/coffee/espresso/"`) || !strings.Contains(twiceLinked.Content, `href="/coffee/ristretto/"`) {
		t.Fatalf("the parent holds %q, want both links in its paragraph", twiceLinked.Content)
	}
	inBlock := block.Parse(twiceLinked.Content).Blocks[0].Hash
	rows := recorder.byPage["page-parent"]
	if len(rows) != 2 {
		t.Fatalf("after the second relink the links are %+v, want two", rows)
	}
	for _, row := range rows {
		if !row.Ours || row.BlockHash != inBlock {
			t.Errorf("link = %+v, want it ours and keyed to the paragraph as the second relink left it", row)
		}
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
