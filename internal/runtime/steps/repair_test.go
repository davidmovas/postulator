package steps_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const flatBody = `<h1>Espresso</h1><p>Written earlier and not to be touched.</p>`

func repairContext(t *testing.T, wpID int64) *run.StepContext {
	t.Helper()

	sc := unitContext(t, nil)
	sc.Run.Kind = run.KindRepair
	sc.Page.Path = "/coffee/espresso/"
	sc.Page.Slug = "espresso"
	sc.Page.ParentPageID = pointer("page-parent")
	sc.Page.Status = pagemap.StatusPublished
	sc.Page.WPID = &wpID
	return sc
}

func TestRepairHierarchyMovesAFlatPageWithoutRewritingIt(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	parent := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Coffee", Slug: "coffee", Status: "publish"})
	parentWP := parent[0].ID
	flat := server.Seed(wptest.Item{
		Type: wptest.TypePage, Title: "Espresso", Slug: "espresso", Status: "publish", Content: flatBody,
	})
	childWP := flat[0].ID

	stored := &pagemap.Page{}
	deps.Pages = pageList{recorded: stored, items: []pagemap.Page{{
		ID: "page-parent", SiteID: "site", Path: "/coffee/", Slug: "coffee", WPType: pagemap.WPPage,
		Status: pagemap.StatusPublished, WPID: &parentWP,
	}}}

	result, err := steps.RepairHierarchy(deps).Run(t.Context(), repairContext(t, childWP))
	if err != nil {
		t.Fatalf("RepairHierarchy: %v", err)
	}
	if len(result.Artifacts) != 1 || result.Artifacts[0].Kind != run.ArtifactPublishResult {
		t.Fatalf("RepairHierarchy produced %+v", result.Artifacts)
	}

	moved, ok := server.Lookup(childWP)
	if !ok || moved.Parent != parentWP {
		t.Fatalf("the page is %+v, want it under %d", moved, parentWP)
	}
	if moved.Content != flatBody || moved.Title != "Espresso" {
		t.Fatalf("the repair rewrote the page: %+v", moved)
	}
	if len(server.Items()) != 2 {
		t.Fatalf("the site holds %d items, want the parent and the moved page", len(server.Items()))
	}

	if !strings.HasSuffix(stored.Observed.Link, "/coffee/espresso/") {
		t.Fatalf("the stored mirror = %+v, want the address the move produced", stored.Observed)
	}
	if found := stored.Mismatches(); len(found) != 0 {
		t.Fatalf("mismatches after the repair = %+v, want none", found)
	}

	var published steps.PublishResult
	if err = json.Unmarshal(result.Artifacts[0].Blob, &published); err != nil {
		t.Fatalf("decode the publish result: %v", err)
	}
	if published.WPID != childWP || published.Created {
		t.Fatalf("the result = %+v", published)
	}
}

func TestRepairHierarchyWaitsForAParentThatIsNotThereYet(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	flat := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Espresso", Slug: "espresso", Status: "publish"})

	result, err := steps.RepairHierarchy(deps).Run(t.Context(), repairContext(t, flat[0].ID))
	if err != nil {
		t.Fatalf("RepairHierarchy: %v", err)
	}
	if result.Next != run.TransitionWait {
		t.Fatalf("a repair under an unpublished parent = %q, want a wait", result.Next)
	}
	if moved, ok := server.Lookup(flat[0].ID); !ok || moved.Parent != 0 {
		t.Fatalf("the page is %+v, want it left alone", moved)
	}
}

func TestRepairHierarchyReportsWhatItCannotDo(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		with func(*run.StepContext)
		want errors.Code
	}{
		{
			name: "the page was never written to the site",
			with: func(sc *run.StepContext) { sc.Page.WPID = nil },
			want: errors.Invalid,
		},
		{
			name: "the page names a parent the map does not hold",
			with: func(sc *run.StepContext) { sc.Page.ParentPageID = pointer("page-ghost") },
			want: errors.Invalid,
		},
		{
			name: "a product is not moved here",
			with: func(sc *run.StepContext) { sc.Page.WPType = pagemap.WPProduct },
			want: errors.Invalid,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps, server := imageDeps(t)
			flat := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Espresso", Slug: "espresso"})
			sc := repairContext(t, flat[0].ID)
			tc.with(sc)

			if _, err := steps.RepairHierarchy(deps).Run(t.Context(), sc); !errors.IsCode(err, tc.want) {
				t.Fatalf("code = %q, want %q (err %v)", errors.CodeOf(err), tc.want, err)
			}
		})
	}
}
