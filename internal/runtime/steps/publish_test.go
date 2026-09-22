package steps_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const publishBody = `<h1>Espresso</h1><p>Espresso is a way to make <a href="/coffee/">coffee</a>.</p>`

func publishContext(t *testing.T) *run.StepContext {
	t.Helper()

	sc := unitContext(t, map[run.ArtifactKind][]byte{
		run.ArtifactBodyHTML: []byte(publishBody),
		run.ArtifactDraft:    []byte(goodDraft),
		run.ArtifactMeta:     []byte(`{"title":"espresso | Shop","description":"Pull a shot.","canonical":"https://shop.example.com/coffee/espresso/"}`),
	})
	sc.Page.Slug = pagemap.Slug(sc.Page.Path)
	return sc
}

func runPublish(t *testing.T, deps steps.Deps, sc *run.StepContext) steps.PublishResult {
	t.Helper()

	result, err := steps.Publish(deps).Run(t.Context(), sc)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if len(result.Artifacts) != 1 || result.Artifacts[0].Kind != run.ArtifactPublishResult {
		t.Fatalf("Publish produced %+v", result.Artifacts)
	}

	var published steps.PublishResult
	if err = json.Unmarshal(result.Artifacts[0].Blob, &published); err != nil {
		t.Fatalf("decode the publish result: %v", err)
	}
	return published
}

func TestPublishCreatesThenUpdates(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	sc := publishContext(t)

	first := runPublish(t, deps, sc)
	if !first.Created || first.WPID == 0 || first.URL == "" {
		t.Fatalf("first publish = %+v", first)
	}
	if first.Status != "draft" {
		t.Errorf("status = %q, want draft", first.Status)
	}
	if !slices.Contains(first.SEOApplied, "title") {
		t.Errorf("seoApplied = %v", first.SEOApplied)
	}

	stored, ok := server.Lookup(first.WPID)
	if !ok || stored.Slug != "espresso" || stored.Content != publishBody {
		t.Fatalf("the site holds %+v", stored)
	}
	if stored.Title != "Espresso guide" {
		t.Errorf("title = %q, want the draft title", stored.Title)
	}

	sc.Page.WPID = &first.WPID
	second := runPublish(t, deps, sc)
	if second.Created || second.WPID != first.WPID {
		t.Fatalf("second publish = %+v, want an update of %d", second, first.WPID)
	}
	if len(server.Items()) != 1 {
		t.Fatalf("the site holds %d items, want one", len(server.Items()))
	}
}

func TestPublishFindsAnExistingPageBySlugAndParent(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	seeded := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Espresso", Status: "draft"})

	published := runPublish(t, deps, publishContext(t))
	if published.Created || published.WPID != seeded[0].ID {
		t.Fatalf("publish = %+v, want an update of %d", published, seeded[0].ID)
	}
	if len(server.Items()) != 1 {
		t.Fatalf("the site holds %d items, want one", len(server.Items()))
	}
}

func TestPublishSkipsTheSEOMetaWithoutThePlugin(t *testing.T) {
	t.Parallel()

	deps, _ := imageDepsWith(t, wptest.WithoutPlugin())
	published := runPublish(t, deps, publishContext(t))

	if len(published.SEOApplied) != 0 {
		t.Errorf("seoApplied = %v, want none", published.SEOApplied)
	}
	if !slices.Contains(published.Skipped, steps.CodeSEOMetaSkipped) {
		t.Fatalf("skipped = %v, want %q", published.Skipped, steps.CodeSEOMetaSkipped)
	}
	if len(published.Findings) != 1 || published.Findings[0].Code != steps.CodeSEOMetaSkipped {
		t.Fatalf("findings = %+v, want one %q warning", published.Findings, steps.CodeSEOMetaSkipped)
	}
	if published.Findings[0].Severity != content.SeverityWarn {
		t.Errorf("the skipped finding = %+v, want a warning", published.Findings[0])
	}
	if published.Findings[0].Details["reason"] != steps.ReasonNoPlugin {
		t.Errorf("the skipped finding = %+v, want the reason %q", published.Findings[0], steps.ReasonNoPlugin)
	}
}

func TestPublishReportsWhatItCannotDo(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		with func(*run.StepContext)
		deps func(steps.Deps) steps.Deps
		want errors.Code
	}{
		{
			name: "there is no body",
			with: func(sc *run.StepContext) { sc.Artifacts = map[run.ArtifactKind]run.Artifact{} },
			want: errors.Invalid,
		},
		{
			name: "a product is not written here",
			with: func(sc *run.StepContext) { sc.Page.WPType = pagemap.WPProduct },
			want: errors.Invalid,
		},
		{
			name: "no client is configured",
			deps: func(d steps.Deps) steps.Deps { d.WordPress = nil; return d },
			want: errors.Invalid,
		},
		{
			name: "the site cannot be reached",
			deps: func(d steps.Deps) steps.Deps {
				d.WordPress = oneClient{err: errors.New(errors.External, "the site is down")}
				return d
			},
			want: errors.External,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps, _ := imageDeps(t)
			if tc.deps != nil {
				deps = tc.deps(deps)
			}
			sc := publishContext(t)
			if tc.with != nil {
				tc.with(sc)
			}
			if _, err := steps.Publish(deps).Run(t.Context(), sc); !errors.IsCode(err, tc.want) {
				t.Fatalf("code = %q, want %q (err %v)", errors.CodeOf(err), tc.want, err)
			}
		})
	}
}

func TestPublishFallsBackWhenTheStoredIdIsGone(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	seeded := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Espresso", Status: "draft"})

	sc := publishContext(t)
	ghost := int64(9999)
	sc.Page.WPID = &ghost

	published := runPublish(t, deps, sc)
	if published.Created || published.WPID != seeded[0].ID {
		t.Fatalf("publish = %+v, want the slug lookup to find %d", published, seeded[0].ID)
	}
}

func TestPublishReparentsUnderTheParentPage(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	parent := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Coffee", Slug: "coffee"})
	wpID := parent[0].ID

	deps.Pages = pageList{items: []pagemap.Page{
		{
			ID: "page-parent", SiteID: "site", Path: "/coffee/", Slug: "coffee", WPType: pagemap.WPPage,
			Status: pagemap.StatusPublished, WPID: &wpID,
		},
	}}

	sc := publishContext(t)
	sc.Page.ParentPageID = pointer("page-parent")

	published := runPublish(t, deps, sc)
	stored, ok := server.Lookup(published.WPID)
	if !ok || stored.Parent != wpID {
		t.Fatalf("the draft is %+v, want it under %d", stored, wpID)
	}
}

func TestPublishWaitsWhileTheParentIsNotOnTheSite(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	sc := publishContext(t)
	sc.Page.ParentPageID = pointer("page-parent")

	result, err := steps.Publish(deps).Run(t.Context(), sc)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if result.Next != run.TransitionWait {
		t.Fatalf("Publish under an unpublished parent = %q, want a wait", result.Next)
	}
	if len(result.Artifacts) != 0 {
		t.Fatalf("a waiting publish produced %+v", result.Artifacts)
	}
	if len(server.Items()) != 0 {
		t.Fatalf("the site holds %d items, want none while the parent is missing", len(server.Items()))
	}
}

func TestPublishStopsOnceTheParentHasNotArrived(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	sc := publishContext(t)
	sc.Page.ParentPageID = pointer("page-parent")

	var result run.Result
	for attempt := 0; attempt <= steps.ParentWaitLimit; attempt++ {
		var err error
		result, err = steps.Publish(deps).Run(t.Context(), sc)
		if err != nil {
			t.Fatalf("Publish: %v", err)
		}
		sc.Check = sc.Check.MergedWith(result.Checkpoint)
		if result.Next != run.TransitionWait {
			break
		}
	}

	if result.Next != run.TransitionPause || result.Reason != run.PauseNeedsHuman {
		t.Fatalf("Publish after %d waits = %q / %q, want a pause for a human",
			steps.ParentWaitLimit, result.Next, result.Reason)
	}
	if !strings.Contains(result.Message, "/coffee/") {
		t.Errorf("the pause message %q does not name the parent path", result.Message)
	}
	if len(server.Items()) != 0 {
		t.Fatalf("the site holds %d items, want none", len(server.Items()))
	}
}

func TestPublishRefusesADanglingParent(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	sc := publishContext(t)
	sc.Page.ParentPageID = pointer("page-ghost")

	if _, err := steps.Publish(deps).Run(t.Context(), sc); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Publish under a parent that is not in the map = %v, want an invalid error", err)
	}
	if len(server.Items()) != 0 {
		t.Fatalf("the site holds %d items, want none", len(server.Items()))
	}
}

func TestPublishMovesAFlatPageRatherThanDuplicatingIt(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	parent := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Coffee", Slug: "coffee"})
	wpID := parent[0].ID
	flat := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Espresso", Slug: "espresso", Status: "draft"})

	deps.Pages = pageList{items: []pagemap.Page{{
		ID: "page-parent", SiteID: "site", Path: "/coffee/", Slug: "coffee", WPType: pagemap.WPPage,
		Status: pagemap.StatusPublished, WPID: &wpID,
	}}}

	sc := publishContext(t)
	sc.Page.ParentPageID = pointer("page-parent")

	published := runPublish(t, deps, sc)
	if published.Created || published.WPID != flat[0].ID {
		t.Fatalf("publish = %+v, want the flat page %d moved rather than a second one made",
			published, flat[0].ID)
	}
	if len(server.Items()) != 2 {
		t.Fatalf("the site holds %d items, want the parent and the one page", len(server.Items()))
	}
	stored, ok := server.Lookup(published.WPID)
	if !ok || stored.Parent != wpID {
		t.Fatalf("the moved page is %+v, want it under %d", stored, wpID)
	}
}

func TestPublishSkipsTheSEOMetaWithoutAMetaArtifact(t *testing.T) {
	t.Parallel()

	deps, _ := imageDeps(t)
	sc := publishContext(t)
	delete(sc.Artifacts, run.ArtifactMeta)

	published := runPublish(t, deps, sc)
	if len(published.SEOApplied) != 0 || !slices.Contains(published.Skipped, steps.CodeSEOMetaSkipped) {
		t.Fatalf("publish = %+v", published)
	}
	if len(published.Findings) != 0 {
		t.Errorf("findings = %+v, want none: no meta was generated, so none was lost", published.Findings)
	}
}

func TestPublishRecordsAPublishedPage(t *testing.T) {
	t.Parallel()

	deps, _ := imageDeps(t)
	sc := publishContext(t)
	sc.Run.PublishMode = run.PublishLive

	published := runPublish(t, deps, sc)
	if published.Status != "publish" {
		t.Fatalf("publish = %+v", published)
	}
}

func TestPublishWarnsOverADriftedPage(t *testing.T) {
	t.Parallel()

	deps, _ := imageDeps(t)
	sc := publishContext(t)
	sc.Page.Drift = true

	published := runPublish(t, deps, sc)
	if published.WPID == 0 {
		t.Fatalf("publish over drift = %+v, want the page written", published)
	}
	if len(published.Findings) != 1 || published.Findings[0].Code != steps.CodePublishOverDrift {
		t.Fatalf("findings = %+v", published.Findings)
	}
	if published.Findings[0].Severity != content.SeverityWarn ||
		published.Findings[0].Details["class"] != steps.ClassNeedsHuman {
		t.Fatalf("the drift finding = %+v", published.Findings[0])
	}
}

func TestPublishRefusesADriftedPage(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	sc := publishContext(t)
	sc.Page.Drift = true
	sc.Params[steps.ParamRefuseDrift] = true

	result, err := steps.Publish(deps).Run(t.Context(), sc)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if result.Next != run.TransitionPause || result.Reason != run.PauseNeedsHuman {
		t.Fatalf("Publish over drift = %q / %q, want a pause for a human", result.Next, result.Reason)
	}
	if len(result.Artifacts) != 0 {
		t.Fatalf("a refused publish produced %+v", result.Artifacts)
	}
	if len(server.Items()) != 0 {
		t.Fatalf("the site holds %d items, want none", len(server.Items()))
	}
}
