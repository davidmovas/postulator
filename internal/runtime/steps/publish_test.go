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
	sc.Page.Path = "/espresso/"
	sc.Page.Slug = pagemap.Slug(sc.Page.Path)
	return sc
}

func nestedContext(t *testing.T) *run.StepContext {
	t.Helper()

	sc := publishContext(t)
	sc.Page.Path = "/coffee/espresso/"
	sc.Page.Slug = pagemap.Slug(sc.Page.Path)
	sc.Page.ParentPageID = pointer("page-parent")
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

func TestPublishKeepsTheSEOMetaItReplaced(t *testing.T) {
	t.Parallel()

	deps, server := imageDepsWith(t, wptest.WithSEOPlugin("yoast"))
	seeded := server.Seed(wptest.Item{
		Type: wptest.TypePage, Title: "Espresso", Slug: "espresso", Status: "draft",
		Meta: map[string]string{
			"_yoast_wpseo_title":    "what a human wrote",
			"_yoast_wpseo_metadesc": "and the description with it",
		},
	})

	published := runPublish(t, deps, publishContext(t))
	if published.Created || published.WPID != seeded[0].ID {
		t.Fatalf("publish = %+v, want an update of the seeded page", published)
	}
	if published.PreviousMeta == nil {
		t.Fatal("the publish overwrote the SEO meta without keeping what was there")
	}
	if published.PreviousMeta.Title != "what a human wrote" {
		t.Errorf("previousMeta.title = %q", published.PreviousMeta.Title)
	}
	if published.PreviousMeta.Description != "and the description with it" {
		t.Errorf("previousMeta.description = %q", published.PreviousMeta.Description)
	}
	if published.PreviousMeta.Canonical != "" {
		t.Errorf("previousMeta.canonical = %q, want the empty field the page carried", published.PreviousMeta.Canonical)
	}

	stored, _ := server.Lookup(published.WPID)
	if stored.Meta["_yoast_wpseo_title"] != "espresso | Shop" {
		t.Errorf("the run did not write its own meta: %v", stored.Meta)
	}
}

func TestPublishKeepsNoPreviousMetaItCouldNotRead(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		options []wptest.Option
		seed    bool
	}{
		{name: "a page it created", seed: false},
		{
			name: "a plugin too old to answer",
			seed: true,
			options: []wptest.Option{wptest.WithCapabilities(
				"bulk", "seo_meta", "content_hash", "raw", "preview",
			)},
		},
		{name: "a site without the plugin", seed: true, options: []wptest.Option{wptest.WithoutPlugin()}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps, server := imageDepsWith(t, tc.options...)
			if tc.seed {
				server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Espresso", Slug: "espresso", Status: "draft"})
			}

			published := runPublish(t, deps, publishContext(t))
			if published.PreviousMeta != nil {
				t.Fatalf("previousMeta = %+v, want nothing that was never read", published.PreviousMeta)
			}
		})
	}
}

func TestPublishSaysWhenRetentionTookTheMetaBeforeItCouldBeWritten(t *testing.T) {
	t.Parallel()

	deps, _ := imageDeps(t)
	sc := publishContext(t)
	sc.Artifacts[run.ArtifactMeta] = run.Artifact{Kind: run.ArtifactMeta, Purged: true}

	published := runPublish(t, deps, sc)

	if len(published.SEOApplied) != 0 {
		t.Fatalf("seoApplied = %v, want nothing written from an artifact that is gone", published.SEOApplied)
	}
	purged := make([]content.Finding, 0, 1)
	for _, finding := range published.Findings {
		if finding.Code == steps.CodeArtifactPurged {
			purged = append(purged, finding)
		}
	}
	if len(purged) != 1 {
		t.Fatalf("the publish raised %d artifact_purged findings, want exactly one: %+v",
			len(purged), published.Findings)
	}
	if purged[0].Severity != content.SeverityWarn {
		t.Fatalf("severity = %q, want warn", purged[0].Severity)
	}
	if !strings.Contains(purged[0].Message, sc.Page.Path) {
		t.Fatalf("the finding does not name the page: %q", purged[0].Message)
	}
	if purged[0].Details["kind"] != string(run.ArtifactMeta) {
		t.Fatalf("details.kind = %v, want meta", purged[0].Details["kind"])
	}
	if purged[0].Details["pageId"] != sc.Page.ID {
		t.Fatalf("details.pageId = %v, want %q", purged[0].Details["pageId"], sc.Page.ID)
	}
}

func TestPublishStaysQuietWhenNoMetaWasEverGenerated(t *testing.T) {
	t.Parallel()

	deps, _ := imageDeps(t)
	sc := publishContext(t)
	delete(sc.Artifacts, run.ArtifactMeta)

	published := runPublish(t, deps, sc)

	for _, finding := range published.Findings {
		if finding.Code == steps.CodeArtifactPurged {
			t.Fatalf("a recipe without generate_meta raised %q", finding.Code)
		}
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

	sc := nestedContext(t)

	published := runPublish(t, deps, sc)
	stored, ok := server.Lookup(published.WPID)
	if !ok || stored.Parent != wpID {
		t.Fatalf("the draft is %+v, want it under %d", stored, wpID)
	}
}

func TestPublishWaitsWhileTheParentIsNotOnTheSite(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	sc := nestedContext(t)

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
	sc := nestedContext(t)

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

func TestPublishKeepsASectionOutOfTheHomePage(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	home := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Home", Slug: "home"})
	homeID := home[0].ID

	deps.Pages = pageList{items: []pagemap.Page{{
		ID: "page-home", SiteID: "site", Path: "/", Slug: "", WPType: pagemap.WPPage,
		Status: pagemap.StatusPublished, WPID: &homeID,
	}}}

	sc := publishContext(t)
	sc.Page.Path = "/coffee/"
	sc.Page.Slug = pagemap.Slug(sc.Page.Path)
	sc.Page.ParentPageID = pointer("page-home")

	published := runPublish(t, deps, sc)
	stored, ok := server.Lookup(published.WPID)
	if !ok || stored.Parent != 0 {
		t.Fatalf("the section is %+v, want it at the top level: the front page is not an ancestor", stored)
	}
}

func TestPublishRefusesAnIncompleteAncestorChain(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	sc := publishContext(t)
	sc.Page.Path = "/components/batteries/e-bike-range/"
	sc.Page.Slug = pagemap.Slug(sc.Page.Path)
	sc.Page.ParentPageID = nil

	_, err := steps.Publish(deps).Run(t.Context(), sc)
	if !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Publish of a page whose parent path has no page = %v, want an invalid error", err)
	}
	if !strings.Contains(err.Error(), "/components/batteries/") {
		t.Errorf("the refusal %q does not name the missing parent path", err.Error())
	}
	if len(server.Items()) != 0 {
		t.Fatalf("the site holds %d items, want none", len(server.Items()))
	}
}

func TestPublishRefusesADanglingParent(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	sc := nestedContext(t)
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

	sc := nestedContext(t)

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

func TestPublishRecordsWhatTheSiteReportedBack(t *testing.T) {
	t.Parallel()

	deps, _ := imageDeps(t)
	written := &pageRecorder{}
	deps.Pages = written

	sc := publishContext(t)
	sc.Run.PublishMode = run.PublishLive
	published := runPublish(t, deps, sc)

	if written.last.Observed.Link != published.URL {
		t.Fatalf("the stored link = %q, want the address WordPress answered with (%q)",
			written.last.Observed.Link, published.URL)
	}
	if written.last.Observed.Slug != "espresso" || written.last.Observed.Status != "publish" {
		t.Fatalf("the stored mirror = %+v", written.last.Observed)
	}
	if written.last.Observed.Title != "Espresso guide" {
		t.Errorf("the stored title = %q, want the title the site holds", written.last.Observed.Title)
	}
	if len(published.Mismatches) != 0 {
		t.Errorf("mismatches = %+v, want none: the site agreed", published.Mismatches)
	}
}

func TestPublishStopsWhenWordPressRenamesATakenSlug(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Espresso", Slug: "espresso", Status: "publish"})
	ours := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Espresso", Slug: "other", Status: "publish"})

	sc := publishContext(t)
	sc.Page.WPID = &ours[0].ID

	result, err := steps.Publish(deps).Run(t.Context(), sc)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if result.Next != run.TransitionPause || result.Reason != run.PauseNeedsHuman {
		t.Fatalf("Publish onto a taken slug = %q / %q, want a pause for a human", result.Next, result.Reason)
	}
	if !strings.Contains(result.Message, "espresso-2") {
		t.Errorf("the pause message %q does not name the slug the site chose", result.Message)
	}
}

func TestPublishStopsWhenTheAddressIsNotTheOneAskedFor(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	elsewhere := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Tea", Slug: "tea"})
	wpID := elsewhere[0].ID

	deps.Pages = pageList{items: []pagemap.Page{{
		ID: "page-parent", SiteID: "site", Path: "/coffee/", Slug: "coffee", WPType: pagemap.WPPage,
		Status: pagemap.StatusPublished, WPID: &wpID,
	}}}

	sc := nestedContext(t)
	sc.Run.PublishMode = run.PublishLive

	result, err := steps.Publish(deps).Run(t.Context(), sc)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if result.Next != run.TransitionPause || result.Reason != run.PauseNeedsHuman {
		t.Fatalf("Publish under a parent that sits elsewhere = %q / %q, want a pause", result.Next, result.Reason)
	}
	if !strings.Contains(result.Message, "/coffee/espresso/") || !strings.Contains(result.Message, "/tea/espresso/") {
		t.Errorf("the pause message %q names neither the plan nor what is there", result.Message)
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
