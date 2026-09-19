package steps_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const parentBody = `<h1>Coffee</h1><p>We roast every espresso blend we sell.</p>`

type linkRecorder struct {
	byPage map[string][]pagemap.PageLink
	err    error
}

func (r *linkRecorder) ListForPage(_ context.Context, pageID string) ([]pagemap.PageLink, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.byPage[pageID], nil
}

func (r *linkRecorder) ReplaceForPage(_ context.Context, pageID string, links []pagemap.PageLink) error {
	if r.err != nil {
		return r.err
	}
	if r.byPage == nil {
		r.byPage = make(map[string][]pagemap.PageLink)
	}
	r.byPage[pageID] = links
	return nil
}

func relinkDeps(t *testing.T, body string, opts ...wptest.Option) (steps.Deps, *wptest.Server, *linkRecorder, int64) {
	t.Helper()

	deps, server := imageDepsWith(t, opts...)
	seeded := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Coffee", Content: body})
	wpID := seeded[0].ID

	recorder := &linkRecorder{}
	deps.Links = recorder
	deps.Pages = pageList{items: []pagemap.Page{
		{
			ID: "page-parent", SiteID: "site", Path: "/coffee/", Slug: "coffee", WPType: pagemap.WPPage,
			Status: pagemap.StatusPublished, EntityID: pointer("parent"), WPID: &wpID,
		},
		{
			ID: "page-child", SiteID: "site", Path: "/coffee/espresso/", Slug: "espresso",
			WPType: pagemap.WPPage, Status: pagemap.StatusExists, EntityID: pointer("child"),
		},
	}}
	return deps, server, recorder, wpID
}

func relinkContext(t *testing.T, deps steps.Deps) *run.StepContext {
	t.Helper()

	return unitContext(t, map[run.ArtifactKind][]byte{
		run.ArtifactLinkContext:   linkContextBlob(t, deps),
		run.ArtifactPublishResult: []byte(`{"wpId":99,"url":"https://shop.example.com/coffee/espresso/"}`),
	})
}

func runRelink(t *testing.T, deps steps.Deps) steps.RelinkResult {
	t.Helper()

	result, err := steps.RelinkNeighbors(deps).Run(t.Context(), relinkContext(t, deps))
	if err != nil {
		t.Fatalf("RelinkNeighbors: %v", err)
	}
	if len(result.Artifacts) != 1 || result.Artifacts[0].Kind != run.ArtifactRelinkResult {
		t.Fatalf("RelinkNeighbors produced %+v", result.Artifacts)
	}

	var relinked steps.RelinkResult
	if err = json.Unmarshal(result.Artifacts[0].Blob, &relinked); err != nil {
		t.Fatalf("decode the relink result: %v", err)
	}
	return relinked
}

func TestRelinkAddsTheMissingEdgeToANeighbor(t *testing.T) {
	t.Parallel()

	deps, server, recorder, wpID := relinkDeps(t, parentBody)
	relinked := runRelink(t, deps)

	if relinked.Linked != 1 || len(relinked.Neighbors) != 1 {
		t.Fatalf("relinked = %+v", relinked)
	}
	if relinked.Neighbors[0].Outcome != steps.OutcomeLinked || relinked.Neighbors[0].Anchor != "espresso" {
		t.Fatalf("neighbor = %+v", relinked.Neighbors[0])
	}

	stored, ok := server.Lookup(wpID)
	if !ok || !strings.Contains(stored.Content, `<a href="/coffee/espresso/">espresso</a>`) {
		t.Fatalf("the parent holds %q", stored.Content)
	}
	links := recorder.byPage["page-parent"]
	if len(links) != 1 || links[0].ToPageID == nil || *links[0].ToPageID != "page-child" {
		t.Fatalf("the recorded links are %+v", links)
	}
}

func TestRelinkLeavesANeighborThatAlreadyLinks(t *testing.T) {
	t.Parallel()

	linked := `<h1>Coffee</h1><p>We roast every <a href="/coffee/espresso/">espresso</a> blend.</p>`
	deps, server, _, wpID := relinkDeps(t, linked)

	relinked := runRelink(t, deps)
	if relinked.Linked != 0 || relinked.Neighbors[0].Outcome != steps.OutcomeUnchanged {
		t.Fatalf("relinked = %+v", relinked)
	}

	stored, _ := server.Lookup(wpID)
	if stored.Content != linked {
		t.Fatalf("the parent was rewritten: %q", stored.Content)
	}
}

func TestRelinkRecordsAConflictWithoutFailingTheItem(t *testing.T) {
	t.Parallel()

	deps, server, _, wpID := relinkDeps(t, parentBody)
	server.EditBeforeNextRawWrite(wpID, parentBody+"<p>Edited by a human.</p>")

	relinked := runRelink(t, deps)
	if relinked.Conflicts != 1 || relinked.Neighbors[0].Outcome != steps.OutcomeConflict {
		t.Fatalf("relinked = %+v", relinked)
	}
	if len(relinked.Findings) != 1 || relinked.Findings[0].Code != steps.CodeRelinkConflict {
		t.Fatalf("findings = %+v", relinked.Findings)
	}
	if relinked.Findings[0].Details["class"] != steps.ClassNeedsHuman {
		t.Fatalf("finding details = %+v", relinked.Findings[0].Details)
	}
}

func TestRelinkSkipsWithoutThePlugin(t *testing.T) {
	t.Parallel()

	deps, server, _, wpID := relinkDeps(t, parentBody, wptest.WithoutPlugin())

	relinked := runRelink(t, deps)
	if len(relinked.Neighbors) != 1 || relinked.Neighbors[0].Outcome != steps.OutcomeSkipped {
		t.Fatalf("relinked = %+v", relinked)
	}
	if relinked.Linked != 0 || relinked.Conflicts != 0 || relinked.Skipped != 1 {
		t.Fatalf("relinked = %+v", relinked)
	}
	if relinked.Neighbors[0].Detail != steps.ReasonNoPlugin {
		t.Errorf("detail = %q, want %q", relinked.Neighbors[0].Detail, steps.ReasonNoPlugin)
	}
	if len(relinked.Findings) != 1 || relinked.Findings[0].Code != steps.CodeRelinkSkipped {
		t.Fatalf("findings = %+v, want one %q warning", relinked.Findings, steps.CodeRelinkSkipped)
	}
	if relinked.Findings[0].Severity != content.SeverityWarn ||
		relinked.Findings[0].Details["reason"] != steps.ReasonNoPlugin {
		t.Errorf("the skipped finding = %+v", relinked.Findings[0])
	}

	stored, ok := server.Lookup(wpID)
	if !ok {
		t.Fatalf("the neighbor %d is gone from the site", wpID)
	}
	if stored.Content != parentBody {
		t.Errorf("the neighbor content is %q, want the stored content untouched", stored.Content)
	}
}

func TestRelinkReportsWhatItCannotRead(t *testing.T) {
	t.Parallel()

	deps, _, _, _ := relinkDeps(t, parentBody)
	sc := relinkContext(t, deps)
	sc.Artifacts = map[run.ArtifactKind]run.Artifact{}

	if _, err := steps.RelinkNeighbors(deps).Run(t.Context(), sc); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("code = %q, want %q (err %v)", errors.CodeOf(err), errors.Invalid, err)
	}
}
