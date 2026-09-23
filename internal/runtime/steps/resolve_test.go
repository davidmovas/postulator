package steps_test

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"strings"
	"sync"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

type vanishingEntities struct {
	items []graph.Entity
	gone  string
	mu    sync.Mutex
	reads int
}

func (v *vanishingEntities) ListBySite(context.Context, string) ([]graph.Entity, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.reads++
	if v.reads == 1 {
		return v.items, nil
	}

	left := make([]graph.Entity, 0, len(v.items))
	for i := range v.items {
		if v.items[i].ID != v.gone {
			left = append(left, v.items[i])
		}
	}
	return left, nil
}

func TestResolveContextRefusesAPageWhoseEntityIsNotInTheGraph(t *testing.T) {
	t.Parallel()

	cases := []struct {
		deps    func(steps.Deps) steps.Deps
		context func(*run.StepContext)
		name    string
		want    errors.Code
		details []string
	}{
		{
			name: "the entity was deleted between the two reads",
			deps: func(d steps.Deps) steps.Deps {
				d.Entities = &vanishingEntities{items: unitEntities(), gone: "child"}
				d.Edges = edgeList{}
				return d
			},
			want:    errors.Invalid,
			details: []string{"pageId", "path", "entityId"},
		},
		{
			name:    "the page is mapped to nothing at all",
			context: func(sc *run.StepContext) { sc.Page.EntityID = nil },
			want:    errors.Invalid,
			details: []string{"pageId", "path"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps := unitDeps()
			if tc.deps != nil {
				deps = tc.deps(deps)
			}
			sc := unitContext(t, nil)
			if tc.context != nil {
				tc.context(sc)
			}

			result, err := steps.ResolveContext(deps).Run(t.Context(), sc)
			if !errors.IsCode(err, tc.want) {
				t.Fatalf("ResolveContext = %+v, %v; want %s", result, err, tc.want)
			}
			if len(result.Artifacts) != 0 {
				t.Fatalf("the step wrote %+v, want nothing at all rather than an empty link context", result.Artifacts)
			}
			var kernel *errors.Error
			if !stderrors.As(err, &kernel) {
				t.Fatalf("%v is not a kernel error", err)
			}
			for _, key := range tc.details {
				if _, named := kernel.Details[key]; !named {
					t.Errorf("the refusal does not name %q: %+v", key, kernel.Details)
				}
			}
		})
	}
}

func TestResolveContextSaysWhatTheGraphAsksForAndNoPageCarries(t *testing.T) {
	t.Parallel()

	deps := unitDeps()
	deps.Entities = entityList{items: []graph.Entity{
		{
			ID: "parent", SiteID: "site", Name: "Coffee", PrimaryKeyword: "coffee",
			Anchors: []graph.Anchor{{Text: "coffee", Source: graph.AnchorUser, Weight: 1}},
			Kind:    graph.KindTopic, Source: graph.SourceUser,
		},
		{
			ID: "child", SiteID: "site", Name: "Espresso", PrimaryKeyword: "espresso",
			Anchors: []graph.Anchor{{Text: "espresso", Source: graph.AnchorUser, Weight: 1}},
			Kind:    graph.KindTopic, Source: graph.SourceUser, CanonicalPageID: pointer("page-child"),
		},
	}}
	deps.Pages = pageList{items: []pagemap.Page{
		{ID: "page-child", SiteID: "site", Path: "/coffee/espresso/", WPType: pagemap.WPPage, Status: pagemap.StatusPlanned},
	}}

	result, err := steps.ResolveContext(deps).Run(t.Context(), unitContext(t, nil))
	if err != nil {
		t.Fatalf("ResolveContext: %v", err)
	}

	var lc content.LinkContext
	if err = json.Unmarshal(result.Artifacts[0].Blob, &lc); err != nil {
		t.Fatalf("decode the link context: %v", err)
	}
	if len(lc.Targets) != 0 {
		t.Fatalf("targets = %+v, want none: the parent is on no page", lc.Targets)
	}
	if !strings.Contains(result.Message, "holding back 1") || !strings.Contains(result.Message, "1 of them required") {
		t.Fatalf("message = %q, want the required link nobody can carry named", result.Message)
	}
}
