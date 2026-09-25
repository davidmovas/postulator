package steps_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const espressoBody = `<h1>Espresso</h1><p>Espresso is the shortest way to make coffee at home.</p>` +
	`<p>A grinder helps, and so does fresh water.</p>`

func relinkPagePages(wpID int64) []pagemap.Page {
	return []pagemap.Page{
		{
			ID: "page-parent", SiteID: "site", Path: "/coffee/", Slug: "coffee", WPType: pagemap.WPPage,
			Status: pagemap.StatusPublished, EntityID: pointer("parent"),
		},
		{
			ID: "page-child", SiteID: "site", Path: "/coffee/espresso/", Slug: "espresso",
			WPType: pagemap.WPPage, Status: pagemap.StatusExists, EntityID: pointer("child"), WPID: &wpID,
		},
	}
}

func relinkPageDeps(t *testing.T, body string, opts ...wptest.Option) (steps.Deps, *wptest.Server, *linkRecorder, int64) {
	t.Helper()

	deps, server := imageDepsWith(t, opts...)
	seeded := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Espresso", Content: body})
	wpID := seeded[0].ID

	recorder := &linkRecorder{}
	deps.Links = recorder
	deps.Pages = pageList{items: relinkPagePages(wpID)}
	return deps, server, recorder, wpID
}

func withGrandparent(deps steps.Deps, wpID int64) steps.Deps {
	deps.Entities = entityList{items: append(unitEntities(), graph.Entity{
		ID: "grand", SiteID: "site", Name: "Drinks", PrimaryKeyword: "drinks",
		Anchors: []graph.Anchor{{Text: "drinks", Source: graph.AnchorUser, Weight: 1}},
		Kind:    graph.KindTopic, Source: graph.SourceUser, CanonicalPageID: pointer("page-grand"),
	})}
	deps.Edges = edgeList{items: append(unitEdges(), graph.Edge{
		ID: "e2", SiteID: "site", FromEntityID: "parent", ToEntityID: "grand",
		Kind: graph.EdgeParent, Weight: 1, Source: graph.SourceUser, Status: graph.StatusApproved,
	})}
	deps.Pages = pageList{items: append(relinkPagePages(wpID), pagemap.Page{
		ID: "page-grand", SiteID: "site", Path: "/drinks/", Slug: "drinks", WPType: pagemap.WPPage,
		Status: pagemap.StatusPublished, EntityID: pointer("grand"),
	})}
	return deps
}

func relinkPageContext(t *testing.T, deps steps.Deps, wpID int64) *run.StepContext {
	t.Helper()

	sc := unitContext(t, map[run.ArtifactKind][]byte{run.ArtifactLinkContext: linkContextBlob(t, deps)})
	sc.Run.Kind = run.KindRelink
	sc.Page.Status = pagemap.StatusExists
	if wpID != 0 {
		sc.Page.WPID = &wpID
	}
	return sc
}

func runRelinkPage(t *testing.T, deps steps.Deps, sc *run.StepContext) (steps.RelinkPageResult, steps.PublishResult, run.Result) {
	t.Helper()

	result, err := steps.RelinkPage(deps).Run(t.Context(), sc)
	if err != nil {
		t.Fatalf("RelinkPage: %v", err)
	}

	var (
		relinked  steps.RelinkPageResult
		published steps.PublishResult
	)
	kinds := make([]run.ArtifactKind, 0, len(result.Artifacts))
	for i := range result.Artifacts {
		kinds = append(kinds, result.Artifacts[i].Kind)
		switch result.Artifacts[i].Kind {
		case run.ArtifactRelinkResult:
			if err = json.Unmarshal(result.Artifacts[i].Blob, &relinked); err != nil {
				t.Fatalf("decode the relink result: %v", err)
			}
		case run.ArtifactPublishResult:
			if err = json.Unmarshal(result.Artifacts[i].Blob, &published); err != nil {
				t.Fatalf("decode the publish result: %v", err)
			}
		default:
			t.Fatalf("RelinkPage produced a %s artifact", result.Artifacts[i].Kind)
		}
	}
	if len(kinds) != 2 {
		t.Fatalf("RelinkPage produced %v, want a publish result and a relink result", kinds)
	}
	return relinked, published, result
}

func TestRelinkPagePlacesTheLinksTheGraphAsksThePageFor(t *testing.T) {
	t.Parallel()

	deps, server, recorder, wpID := relinkPageDeps(t, espressoBody)
	relinked, published, result := runRelinkPage(t, deps, relinkPageContext(t, deps, wpID))

	if relinked.Linked != 1 || len(relinked.Placed) != 1 {
		t.Fatalf("relinked = %+v", relinked)
	}
	if relinked.Placed[0].Outcome != steps.OutcomeLinked || relinked.Placed[0].Anchor != "coffee" {
		t.Fatalf("placed = %+v", relinked.Placed[0])
	}
	if relinked.Placed[0].PageID != "page-parent" || relinked.Placed[0].Path != "/coffee/" {
		t.Fatalf("placed = %+v", relinked.Placed[0])
	}
	if relinked.PageID != "page-child" || relinked.WPID != wpID {
		t.Fatalf("relinked = %+v", relinked)
	}

	stored, ok := server.Lookup(wpID)
	if !ok || !strings.Contains(stored.Content, `<a href="/coffee/">coffee</a>`) {
		t.Fatalf("the page holds %q", stored.Content)
	}
	if published.PreviousContent != espressoBody || published.PreviousContentHash == "" {
		t.Fatalf("the publish result kept %q", published.PreviousContent)
	}
	if published.Created || published.WPID != wpID || published.ContentHash == published.PreviousContentHash {
		t.Fatalf("published = %+v", published)
	}

	links := recorder.byPage["page-child"]
	if len(links) != 1 || links[0].ToPageID == nil || *links[0].ToPageID != "page-parent" {
		t.Fatalf("the recorded links are %+v", links)
	}
	if result.Next == run.TransitionPause {
		t.Fatal("a relink that did its work must not hold the item")
	}
}

func parentOnTheSite(deps steps.Deps) steps.Deps {
	listed, ok := deps.Pages.(pageList)
	if !ok {
		return deps
	}
	items := make([]pagemap.Page, 0, len(listed.items))
	for i := range listed.items {
		page := listed.items[i]
		if page.WPID == nil {
			wpID := int64(500 + i)
			page.WPID = &wpID
		}
		items = append(items, page)
	}
	listed.items = items
	deps.Pages = listed
	return deps
}

func TestRelinkPageBackfillsWhatThePageOwes(t *testing.T) {
	t.Parallel()

	bare := `<h1>Espresso</h1><p>Espresso is the shortest way to make a strong cup at home.</p>` +
		`<p>A grinder helps, and so does fresh water.</p>`

	cases := []struct {
		name     string
		body     string
		deps     func(steps.Deps) steps.Deps
		cap      int
		linked   int
		sentence bool
		codes    []string
		content  string
	}{
		{
			name: "a parent on the site whose anchor the page lacks", body: bare, deps: parentOnTheSite,
			linked: 1, sentence: true, codes: []string{steps.CodeRelinkPhraseTemplated},
			content: `Espresso is the shortest way to make a strong cup at home. Read more about <a href="/coffee/">coffee</a>.`,
		},
		{
			name: "a parent that is not on the site", body: bare,
			codes: []string{},
		},
		{
			name: "a planned parent the page already names", body: espressoBody,
			linked: 1, codes: []string{content.CodeTargetNotPublished},
			content: `make <a href="/coffee/">coffee</a> at home`,
		},
		{
			name: "a parent the budget cannot hold", body: `<h1>Espresso</h1><p>It sits in our <a href="/drinks/">drinks</a> range.</p>`,
			deps: func(d steps.Deps) steps.Deps { return parentOnTheSite(withGrandparent(d, 0)) }, cap: 1,
			codes: []string{content.CodeTargetMissing},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps, server, _, wpID := relinkPageDeps(t, tc.body)
			if tc.deps != nil {
				deps = tc.deps(deps)
				listed, ok := deps.Pages.(pageList)
				if !ok {
					t.Fatalf("the pages are a %T, want the page list stub", deps.Pages)
				}
				for i := range listed.items {
					if listed.items[i].ID == "page-child" {
						listed.items[i].WPID = &wpID
					}
				}
				deps.Pages = listed
			}
			sc := relinkPageContext(t, deps, wpID)
			if tc.cap > 0 {
				sc.Spec.LinkRules.MaxLinks = tc.cap
			}

			relinked, _, result := runRelinkPage(t, deps, sc)
			if relinked.Linked != tc.linked {
				t.Fatalf("linked = %d, want %d: %+v", relinked.Linked, tc.linked, relinked.Placed)
			}
			if got := findingCodes(relinked.Findings); !slices.Equal(got, tc.codes) {
				t.Fatalf("findings = %+v, want %v", relinked.Findings, tc.codes)
			}
			wrote := false
			for _, placed := range relinked.Placed {
				wrote = wrote || placed.Sentence != ""
			}
			if wrote != tc.sentence {
				t.Fatalf("placed = %+v, want a written sentence %v", relinked.Placed, tc.sentence)
			}
			stored, _ := server.Lookup(wpID)
			if tc.content == "" && stored.Content != tc.body {
				t.Fatalf("the page was rewritten: %q", stored.Content)
			}
			if tc.content != "" && !strings.Contains(stored.Content, tc.content) {
				t.Fatalf("the page holds %q, want %q in it", stored.Content, tc.content)
			}
			if result.Next == run.TransitionPause {
				t.Fatalf("result = %+v, want the item to go on", result)
			}
		})
	}
}

func TestRelinkPageLeavesAPageThatAlreadyCarriesEveryLink(t *testing.T) {
	t.Parallel()

	linked := `<h1>Espresso</h1><p>Espresso is the shortest way to make <a href="/coffee/">coffee</a> at home.</p>`
	deps, server, _, wpID := relinkPageDeps(t, linked)
	relinked, published, _ := runRelinkPage(t, deps, relinkPageContext(t, deps, wpID))

	if relinked.Linked != 0 || len(relinked.Placed) != 1 {
		t.Fatalf("relinked = %+v", relinked)
	}
	if relinked.Placed[0].Outcome != steps.OutcomeUnchanged {
		t.Fatalf("placed = %+v", relinked.Placed[0])
	}
	if published.PreviousContent != linked || published.ContentHash != published.PreviousContentHash {
		t.Fatalf("a page nothing was written to reads as it stands: %+v", published)
	}

	stored, ok := server.Lookup(wpID)
	if !ok || stored.Content != linked {
		t.Fatalf("the page holds %q", stored.Content)
	}
}

func TestRelinkPageSpendsThePagesOwnCap(t *testing.T) {
	t.Parallel()

	body := `<h1>Espresso</h1><p>Espresso is made from <a href="/coffee/">coffee</a> beans.</p>` +
		`<p>It sits in our drinks range beside the rest.</p>`

	cases := []struct {
		name string
		cap  int
		want int
	}{
		{name: "a cap the page has already spent", cap: 1},
		{name: "a cap with room left", cap: 2, want: 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps, _, _, wpID := relinkPageDeps(t, body)
			deps = withGrandparent(deps, wpID)

			sc := relinkPageContext(t, deps, wpID)
			sc.Spec.LinkRules.MaxLinks = tc.cap

			relinked, _, _ := runRelinkPage(t, deps, sc)
			if relinked.Linked != tc.want {
				t.Fatalf("linked = %d, want %d: %+v", relinked.Linked, tc.want, relinked.Placed)
			}
		})
	}
}

func TestRelinkPageStandsDownWithoutThePlugin(t *testing.T) {
	t.Parallel()

	deps, server, _, wpID := relinkPageDeps(t, espressoBody, wptest.WithoutPlugin())
	relinked, published, result := runRelinkPage(t, deps, relinkPageContext(t, deps, wpID))

	if relinked.Skipped != 1 || len(relinked.Findings) != 1 {
		t.Fatalf("relinked = %+v", relinked)
	}
	if relinked.Findings[0].Code != steps.CodeRelinkSkipped {
		t.Fatalf("finding = %+v", relinked.Findings[0])
	}
	if !strings.Contains(relinked.Findings[0].Message, steps.ReasonNoPlugin) {
		t.Fatalf("the finding does not name the reason: %q", relinked.Findings[0].Message)
	}
	if published.WPID != wpID || published.PreviousContent != "" {
		t.Fatalf("published = %+v", published)
	}
	if result.Next == run.TransitionPause {
		t.Fatal("a site without the plugin must not hold the item")
	}

	stored, ok := server.Lookup(wpID)
	if !ok || stored.Content != espressoBody {
		t.Fatalf("the page was written without the plugin: %q", stored.Content)
	}
}

func TestRelinkPageHoldsAPageThatIsNotOnTheSite(t *testing.T) {
	t.Parallel()

	deps, _, _, _ := relinkPageDeps(t, espressoBody)
	sc := relinkPageContext(t, deps, 0)
	sc.Page.WPID = nil

	result, err := steps.RelinkPage(deps).Run(t.Context(), sc)
	if err != nil {
		t.Fatalf("RelinkPage: %v", err)
	}
	if result.Next != run.TransitionPause || result.Reason != run.PauseNeedsHuman {
		t.Fatalf("result = %+v", result)
	}
	if !strings.Contains(result.Message, sc.Page.Path) {
		t.Fatalf("the refusal does not name the page: %q", result.Message)
	}
}

func TestRelinkPageHoldsAPageTheSiteChangedUnderIt(t *testing.T) {
	t.Parallel()

	deps, server, _, wpID := relinkPageDeps(t, espressoBody)
	server.EditBeforeNextRawWrite(wpID, espressoBody+"<p>Edited by a human.</p>")

	result, err := steps.RelinkPage(deps).Run(t.Context(), relinkPageContext(t, deps, wpID))
	if err != nil {
		t.Fatalf("RelinkPage: %v", err)
	}
	if result.Next != run.TransitionPause || result.Reason != run.PauseNeedsHuman {
		t.Fatalf("result = %+v", result)
	}
	if !strings.Contains(result.Message, "held ") || !strings.Contains(result.Message, " and now holds ") {
		t.Fatalf("the refusal does not name both hashes: %q", result.Message)
	}
}
