package steps_test

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const syncedBody = `<h1>Espresso</h1><p>Part of our <a href="/coffee/">coffee</a> range and of ` +
	`<a href="https://elsewhere.example.com/x/">another site</a>.</p>`

func syncBackDeps(t *testing.T, opts ...wptest.Option) (steps.Deps, *wptest.Server, *linkRecorder, int64) {
	t.Helper()

	deps, server := imageDepsWith(t, opts...)
	seeded := server.Seed(wptest.Item{
		Type: wptest.TypePage, Title: "Espresso", Slug: "espresso", Content: syncedBody, Status: "draft",
	})
	wpID := seeded[0].ID

	recorder := &linkRecorder{}
	deps.Links = recorder
	deps.Pages = pageList{items: []pagemap.Page{
		{
			ID: "page-parent", SiteID: "site", Path: "/coffee/", Slug: "coffee", WPType: pagemap.WPPage,
			Status: pagemap.StatusPublished, EntityID: pointer("parent"),
		},
		{
			ID: "page-child", SiteID: "site", Path: "/coffee/espresso/", Slug: "espresso",
			WPType: pagemap.WPPage, Status: pagemap.StatusExists, EntityID: pointer("child"), WPID: &wpID,
		},
	}}
	return deps, server, recorder, wpID
}

func syncBackContext(t *testing.T, wpID int64) *run.StepContext {
	t.Helper()

	sc := unitContext(t, map[run.ArtifactKind][]byte{
		run.ArtifactPublishResult: []byte(`{"wpId":` + strconv.FormatInt(wpID, 10) + `,"url":"/coffee/espresso/"}`),
	})
	sc.Page.WPID = &wpID
	sc.Page.Slug = "espresso"
	return sc
}

func runSyncBack(t *testing.T, deps steps.Deps, sc *run.StepContext) steps.SyncResult {
	t.Helper()

	result, err := steps.SyncBack(deps).Run(t.Context(), sc)
	if err != nil {
		t.Fatalf("SyncBack: %v", err)
	}
	if len(result.Artifacts) != 1 || result.Artifacts[0].Kind != run.ArtifactSyncResult {
		t.Fatalf("SyncBack produced %+v", result.Artifacts)
	}

	var synced steps.SyncResult
	if err = json.Unmarshal(result.Artifacts[0].Blob, &synced); err != nil {
		t.Fatalf("decode the sync result: %v", err)
	}
	return synced
}

func TestSyncBackReadsThroughThePlugin(t *testing.T) {
	t.Parallel()

	deps, _, recorder, wpID := syncBackDeps(t)
	synced := runSyncBack(t, deps, syncBackContext(t, wpID))

	if synced.Source != "plugin" || synced.WPID != wpID || synced.Links != 1 {
		t.Fatalf("synced = %+v", synced)
	}
	if synced.ContentHash == "" || synced.ModifiedAt.IsZero() || synced.Status != "draft" {
		t.Fatalf("synced = %+v", synced)
	}

	links := recorder.byPage["page-child"]
	if len(links) != 1 || links[0].ToPageID == nil || *links[0].ToPageID != "page-parent" {
		t.Fatalf("the recorded links are %+v", links)
	}
	if links[0].Origin != pagemap.OriginObserved || links[0].AnchorText != "coffee" {
		t.Fatalf("link = %+v", links[0])
	}
}

func TestSyncBackRecordsWhatTheSiteHolds(t *testing.T) {
	t.Parallel()

	deps, _, _, wpID := syncBackDeps(t)
	stored := &pagemap.Page{}
	pages, ok := deps.Pages.(pageList)
	if !ok {
		t.Fatal("the sync back fixture no longer carries a page list")
	}
	pages.recorded = stored
	deps.Pages = pages

	synced := runSyncBack(t, deps, syncBackContext(t, wpID))

	if stored.Observed.H1 != "Espresso" || stored.Observed.Title != "Espresso" {
		t.Fatalf("the stored mirror = %+v, want the heading and the title the site holds", stored.Observed)
	}
	if stored.Observed.Status != "draft" || stored.Observed.Slug != "espresso" {
		t.Fatalf("the stored mirror = %+v", stored.Observed)
	}
	if len(synced.Mismatches) != 0 {
		t.Errorf("mismatches = %+v, want none: nothing was planned to disagree with", synced.Mismatches)
	}
}

func TestSyncBackReportsAHeadingThatIsNotTheOneAskedFor(t *testing.T) {
	t.Parallel()

	deps, _, _, wpID := syncBackDeps(t)
	sc := syncBackContext(t, wpID)
	sc.Page.H1 = "How Far Can an E-Bike Go?"
	sc.Page.Status = pagemap.StatusExists

	synced := runSyncBack(t, deps, sc)

	if len(synced.Mismatches) != 1 || synced.Mismatches[0].Field != pagemap.FieldH1 {
		t.Fatalf("mismatches = %+v, want the heading reported", synced.Mismatches)
	}
	if synced.Mismatches[0].Planned != "How Far Can an E-Bike Go?" || synced.Mismatches[0].Actual != "Espresso" {
		t.Errorf("the mismatch = %+v", synced.Mismatches[0])
	}
	if len(synced.Findings) != 1 || synced.Findings[0].Code != steps.CodePlanNotKept {
		t.Fatalf("findings = %+v, want one %q warning", synced.Findings, steps.CodePlanNotKept)
	}
	if synced.Findings[0].Severity != content.SeverityWarn ||
		synced.Findings[0].Details["class"] != steps.ClassNeedsHuman {
		t.Errorf("the finding = %+v", synced.Findings[0])
	}
}

func TestSyncBackFallsBackToCoreWithoutThePlugin(t *testing.T) {
	t.Parallel()

	deps, _, recorder, wpID := syncBackDeps(t, wptest.WithoutPlugin())
	synced := runSyncBack(t, deps, syncBackContext(t, wpID))

	if synced.Source != "core" || synced.Links != 1 {
		t.Fatalf("synced = %+v", synced)
	}
	if len(recorder.byPage["page-child"]) != 1 {
		t.Fatalf("the recorded links are %+v", recorder.byPage)
	}
}

func TestSyncBackReportsWhatItCannotDo(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		with func(*run.StepContext)
		deps func(steps.Deps) steps.Deps
		want errors.Code
	}{
		{
			name: "there is no publish result",
			with: func(sc *run.StepContext) { sc.Artifacts = map[run.ArtifactKind]run.Artifact{} },
			want: errors.Invalid,
		},
		{
			name: "the publish result carries no id",
			with: func(sc *run.StepContext) {
				sc.Artifacts[run.ArtifactPublishResult] = run.Artifact{
					Kind: run.ArtifactPublishResult, Blob: []byte(`{"wpId":0}`),
				}
			},
			want: errors.Invalid,
		},
		{
			name: "the page is gone from the site",
			with: func(sc *run.StepContext) {
				sc.Artifacts[run.ArtifactPublishResult] = run.Artifact{
					Kind: run.ArtifactPublishResult, Blob: []byte(`{"wpId":9999}`),
				}
			},
			want: errors.NotFound,
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

			deps, _, _, wpID := syncBackDeps(t)
			if tc.deps != nil {
				deps = tc.deps(deps)
			}
			sc := syncBackContext(t, wpID)
			if tc.with != nil {
				tc.with(sc)
			}
			if _, err := steps.SyncBack(deps).Run(t.Context(), sc); !errors.IsCode(err, tc.want) {
				t.Fatalf("code = %q, want %q (err %v)", errors.CodeOf(err), tc.want, err)
			}
		})
	}
}
