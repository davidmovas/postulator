package content_test

import (
	"reflect"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/template"
)

func pagelessParent(t *testing.T) dag {
	t.Helper()

	entities := []graph.Entity{
		entity("child", "Child", 0.5, []string{"child"}, "page-child"),
		entity("parent", "Parent", 1, []string{"parent"}, ""),
	}
	g, err := graph.New(entities, []graph.Edge{parentEdge("e1", "child", "parent")})
	if err != nil {
		t.Fatalf("graph.New: %v", err)
	}
	return dag{g: g, index: pagemap.NewIndex([]pagemap.Page{page("page-child", "/child/", "child")})}
}

func TestPlanLinksReportsWhatItCannotLink(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		fixture  func(*testing.T) dag
		entityID string
		rules    template.LinkRules
		want     []content.BlockedTarget
	}{
		{
			name:     "a child without a page is blocked once even when it is also related",
			fixture:  multiParentDAG,
			entityID: "coffee",
			rules:    template.LinkRules{UpDepth: 2, DownLinks: true, SiblingMinWeight: 0.5},
			want: []content.BlockedTarget{{
				EntityID: "orphan", Relation: content.RelationDown, Weight: 0.1, Depth: 1,
				Reason: content.BlockedNoCanonicalPage,
			}},
		},
		{
			name:     "with down links off the same entity is blocked as a sibling",
			fixture:  multiParentDAG,
			entityID: "coffee",
			rules:    template.LinkRules{UpDepth: 1, SiblingMinWeight: 0.5},
			want: []content.BlockedTarget{{
				EntityID: "orphan", Relation: content.RelationSibling, Weight: 0.9, Depth: 1,
				Reason: content.BlockedNoCanonicalPage,
			}},
		},
		{
			name:     "a parent without a page is a blocked required target",
			fixture:  pagelessParent,
			entityID: "child",
			rules:    template.LinkRules{UpDepth: 1},
			want: []content.BlockedTarget{{
				EntityID: "parent", Relation: content.RelationUp, Required: true, Weight: 1, Depth: 1,
				Reason: content.BlockedNoCanonicalPage,
			}},
		},
		{
			name:     "siblings below the threshold block nothing",
			fixture:  multiParentDAG,
			entityID: "coffee",
			rules:    template.LinkRules{SiblingMinWeight: 0.95},
			want:     []content.BlockedTarget{},
		},
		{
			name:     "an unknown entity plans nothing",
			fixture:  multiParentDAG,
			entityID: "absent",
			rules:    template.LinkRules{UpDepth: 3, DownLinks: true},
			want:     []content.BlockedTarget{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fixture := tc.fixture(t)
			plan := content.PlanLinks(fixture.g, fixture.index, content.Subject{EntityID: tc.entityID}, policy(tc.rules))
			if !reflect.DeepEqual(plan.Blocked, tc.want) {
				t.Fatalf("blocked = %+v, want %+v", plan.Blocked, tc.want)
			}
			for _, target := range plan.Context.Targets {
				for _, blocked := range plan.Blocked {
					if target.EntityID == blocked.EntityID {
						t.Fatalf("%s is both a target and blocked", target.EntityID)
					}
				}
			}
		})
	}
}

func secondaryPageDAG(t *testing.T) dag {
	t.Helper()

	fixture := multiParentDAG(t)
	pages := append(fixture.index.Pages(), page("page-review", "/reviews/gaggia/", "coffee"))
	return dag{g: fixture.g, index: pagemap.NewIndex(pages)}
}

func TestPlanLinksBindsSelfToThePageBeingWritten(t *testing.T) {
	t.Parallel()

	fixture := secondaryPageDAG(t)
	rules := template.LinkRules{UpDepth: 1, SiblingMinWeight: 0.95}

	cases := []struct {
		name      string
		subject   content.Subject
		canonical content.LinkClass
		itself    content.LinkClass
		extra     bool
	}{
		{
			name: "the canonical page of the entity",
			subject: content.Subject{
				Site: pagemap.NewSite("https://shop.example.com"), PageID: "page-coffee",
				PagePath: "/drinks/hot/coffee/", EntityID: "coffee",
			},
			canonical: content.ClassSelf, itself: content.ClassSelf,
		},
		{
			name: "a second page mapped to the same entity",
			subject: content.Subject{
				Site: pagemap.NewSite("https://shop.example.com"), PageID: "page-review",
				PagePath: "/reviews/gaggia/", EntityID: "coffee",
			},
			canonical: content.ClassGraph, itself: content.ClassSelf, extra: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			lc := content.PlanLinks(fixture.g, fixture.index, tc.subject, policy(rules)).Context
			if lc.PageID != tc.subject.PageID || lc.PageURL != tc.subject.PagePath {
				t.Fatalf("the context binds %s %s, want %s %s",
					lc.PageID, lc.PageURL, tc.subject.PageID, tc.subject.PagePath)
			}
			if got := lc.Resolve("/drinks/hot/coffee/").Class; got != tc.canonical {
				t.Errorf("the canonical page reads as %s, want %s", got, tc.canonical)
			}
			if got := lc.Resolve(tc.subject.PagePath).Class; got != tc.itself {
				t.Errorf("the page itself reads as %s, want %s", got, tc.itself)
			}

			held := false
			for _, target := range lc.Targets {
				if target.PageID != "page-coffee" {
					continue
				}
				held = true
				if target.EntityID != "coffee" || target.Required || target.Relation != content.RelationUp ||
					target.Depth != 0 {
					t.Errorf("the canonical target = %+v, want an optional up link at depth zero", target)
				}
			}
			if held != tc.extra {
				t.Errorf("the canonical page is a target = %t, want %t (%v)", held, tc.extra, urls(lc.Targets))
			}
		})
	}
}

func TestBuildLinkContextIsThePlanContext(t *testing.T) {
	t.Parallel()

	fixture := multiParentDAG(t)
	for _, rules := range []template.LinkRules{
		{UpDepth: 2, DownLinks: true, SiblingMinWeight: 0.5},
		{UpDepth: 1, SiblingMinWeight: 0.8},
	} {
		built := content.BuildLinkContext(fixture.g, fixture.index, "coffee", policy(rules))
		planned := content.PlanLinks(fixture.g, fixture.index, content.Subject{EntityID: "coffee"}, policy(rules)).Context
		if !reflect.DeepEqual(built, planned) {
			t.Fatalf("BuildLinkContext = %+v, PlanLinks.Context = %+v", built, planned)
		}
	}
}

func TestByPageID(t *testing.T) {
	t.Parallel()

	fixture := multiParentDAG(t)
	lc := content.BuildLinkContext(fixture.g, fixture.index, "coffee", policy(template.LinkRules{UpDepth: 1}))

	target, ok := lc.ByPageID("page-root")
	if !ok || target.URL != "/drinks/" {
		t.Fatalf("ByPageID(page-root) = %+v, %v", target, ok)
	}
	if _, found := lc.ByPageID("nope"); found {
		t.Error("ByPageID must refuse an unknown page")
	}
}
