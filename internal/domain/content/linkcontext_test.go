package content_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/template"
)

const siteID = "site-1"

func plannedFor(fixture dag, entityID string, rules template.LinkRules) content.LinkContext {
	return content.PlanLinks(fixture.g, fixture.index, content.Subject{EntityID: entityID}, policy(rules)).Context
}

type dag struct {
	g     graph.Graph
	index pagemap.Index
}

func ref(value string) *string {
	return &value
}

func entity(id, name string, score float64, anchors []string, canonical string) graph.Entity {
	list := make([]graph.Anchor, 0, len(anchors))
	for _, text := range anchors {
		list = append(list, graph.Anchor{Text: text, Source: graph.AnchorUser, Weight: 1})
	}

	record := graph.Entity{
		ID: id, SiteID: siteID, Name: name, Kind: graph.KindTopic, PrimaryKeyword: name,
		Anchors: list, Source: graph.SourceUser, Score: score,
	}
	if canonical != "" {
		record.CanonicalPageID = ref(canonical)
	}
	return record
}

func parentEdge(id, child, parent string) graph.Edge {
	return graph.Edge{
		ID: id, SiteID: siteID, FromEntityID: child, ToEntityID: parent, Kind: graph.EdgeParent,
		Weight: 1, Source: graph.SourceUser, Status: graph.StatusApproved,
	}
}

func relatedEdge(id, from, to string, weight float64) graph.Edge {
	return graph.Edge{
		ID: id, SiteID: siteID, FromEntityID: from, ToEntityID: to, Kind: graph.EdgeRelated,
		Weight: weight, Source: graph.SourceUser, Status: graph.StatusApproved,
	}
}

func page(id, path, entityID string) pagemap.Page {
	record := pagemap.Page{
		ID: id, SiteID: siteID, Path: path, Slug: pagemap.Slug(path), WPType: pagemap.WPPage,
		Status: pagemap.StatusPublished,
	}
	if entityID != "" {
		record.EntityID = ref(entityID)
	}
	return record
}

func multiParentDAG(t *testing.T) dag {
	t.Helper()

	entities := []graph.Entity{
		entity("root", "Drinks", 1, []string{"drinks"}, "page-root"),
		entity("second", "Hot drinks", 0.8, []string{"hot drinks"}, "page-second"),
		entity("coffee", "Coffee", 0.6, []string{"coffee", "brew"}, "page-coffee"),
		entity("espresso", "Espresso", 0.5, []string{"espresso"}, "page-espresso"),
		entity("beans", "Beans", 0.9, []string{"beans"}, "page-beans"),
		entity("grinder", "Grinder", 0.2, []string{"grinder"}, "page-grinder"),
		entity("tea", "Tea", 0.4, []string{"tea"}, "page-tea"),
		entity("orphan", "Orphan", 0.1, []string{"orphan"}, ""),
	}
	edges := []graph.Edge{
		parentEdge("e1", "second", "root"),
		parentEdge("e2", "coffee", "second"),
		parentEdge("e3", "coffee", "root"),
		parentEdge("e4", "espresso", "coffee"),
		parentEdge("e5", "beans", "coffee"),
		parentEdge("e6", "grinder", "coffee"),
		parentEdge("e7", "orphan", "coffee"),
		relatedEdge("e8", "coffee", "tea", 0.7),
		relatedEdge("e9", "coffee", "orphan", 0.9),
	}

	g, err := graph.New(entities, edges)
	if err != nil {
		t.Fatalf("graph.New: %v", err)
	}

	pages := []pagemap.Page{
		page("page-root", "/drinks/", "root"),
		page("page-second", "/drinks/hot/", "second"),
		page("page-coffee", "/drinks/hot/coffee/", "coffee"),
		page("page-espresso", "/drinks/hot/coffee/espresso/", "espresso"),
		page("page-beans", "/drinks/hot/coffee/beans/", "beans"),
		page("page-grinder", "/drinks/hot/coffee/grinder/", "grinder"),
		page("page-tea", "/drinks/hot/tea/", "tea"),
	}
	return dag{g: g, index: pagemap.NewIndex(pages)}
}

func urls(targets []content.LinkTarget) []string {
	out := make([]string, 0, len(targets))
	for _, target := range targets {
		out = append(out, target.URL)
	}
	return out
}

func TestPlanLinksOnAMultiParentDAG(t *testing.T) {
	t.Parallel()

	fixture := multiParentDAG(t)
	rules := template.LinkRules{UpDepth: 2, DownLinks: true, SiblingMinWeight: 0.5, MaxLinks: 10, MaxPerTarget: 1}
	lc := plannedFor(fixture, "coffee", rules)

	if lc.PageID != "page-coffee" || lc.PageURL != "/drinks/hot/coffee/" || lc.EntityID != "coffee" {
		t.Fatalf("the context describes %+v", lc)
	}

	want := []string{
		"/drinks/", "/drinks/hot/",
		"/drinks/hot/coffee/beans/", "/drinks/hot/coffee/espresso/", "/drinks/hot/coffee/grinder/",
		"/drinks/hot/tea/",
	}
	if got := urls(lc.Targets); len(got) != len(want) {
		t.Fatalf("targets = %v, want %v", got, want)
	}

	for i, target := range lc.Targets {
		if target.URL != want[i] {
			t.Fatalf("target %d = %s, want %s", i, target.URL, want[i])
		}
	}

	if lc.Targets[0].Relation != content.RelationUp || !lc.Targets[0].Required || lc.Targets[0].Depth != 1 {
		t.Fatalf("the nearest parent = %+v", lc.Targets[0])
	}
	if lc.Targets[1].Depth != 1 {
		t.Fatalf("the second parent is reachable at depth 1 through the shortcut edge: %+v", lc.Targets[1])
	}
	if lc.Targets[2].Relation != content.RelationDown || lc.Targets[2].Required {
		t.Fatalf("the first child = %+v", lc.Targets[2])
	}
	if lc.Targets[5].Relation != content.RelationSibling || lc.Targets[5].Weight != 0.7 {
		t.Fatalf("the sibling = %+v", lc.Targets[5])
	}

	required := 0
	for _, target := range lc.Targets {
		if target.Required {
			required++
		}
	}
	if required != 2 {
		t.Fatalf("required targets = %d, want the two parents", required)
	}
	if phrases := lc.Phrases(); len(phrases) != 6 || phrases[0] != "drinks" {
		t.Fatalf("Phrases = %v", phrases)
	}
	if lc.Resolve("/drinks/").Class != content.ClassGraph {
		t.Error("Resolve must find a target")
	}
	if lc.Resolve("/nope/").Class != content.ClassUnknownInternal {
		t.Error("Resolve must refuse an unknown path")
	}
}

func TestPlanLinksRespectsThePolicy(t *testing.T) {
	t.Parallel()

	fixture := multiParentDAG(t)

	cases := []struct {
		name  string
		rules template.LinkRules
		want  []string
	}{
		{
			name:  "no children and no siblings",
			rules: template.LinkRules{UpDepth: 1, SiblingMinWeight: 2},
			want:  []string{"/drinks/", "/drinks/hot/"},
		},
		{
			name:  "no parents at all",
			rules: template.LinkRules{DownLinks: true, SiblingMinWeight: 2},
			want: []string{
				"/drinks/hot/coffee/beans/", "/drinks/hot/coffee/espresso/", "/drinks/hot/coffee/grinder/",
			},
		},
		{
			name:  "siblings below the threshold are dropped",
			rules: template.LinkRules{SiblingMinWeight: 0.8},
			want:  []string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			lc := plannedFor(fixture, "coffee", tc.rules)
			got := urls(lc.Targets)
			if len(got) != len(tc.want) {
				t.Fatalf("targets = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("targets = %v, want %v", got, tc.want)
				}
			}
		})
	}
}

func TestPlanLinksSkipsWhatCannotBeLinked(t *testing.T) {
	t.Parallel()

	fixture := multiParentDAG(t)
	rules := template.LinkRules{UpDepth: 3, DownLinks: true, SiblingMinWeight: 0.1}

	lc := plannedFor(fixture, "coffee", rules)
	for _, target := range lc.Targets {
		if target.EntityID == "orphan" {
			t.Fatal("an entity with no canonical page must never be a target")
		}
		if target.EntityID == "coffee" {
			t.Fatal("the page must not link to itself")
		}
	}

	unknown := plannedFor(fixture, "absent", rules)
	if len(unknown.Targets) != 0 || unknown.PageID != "" {
		t.Fatalf("an unknown entity = %+v", unknown)
	}

	anchorless := plannedFor(fixture, "espresso", rules)
	for _, target := range anchorless.Targets {
		if len(target.Anchors) == 0 {
			t.Fatalf("target %s carries no anchor", target.URL)
		}
	}
}

func TestATargetFallsBackToTheEntityName(t *testing.T) {
	t.Parallel()

	entities := []graph.Entity{
		entity("child", "Child", 0.5, nil, "page-child"),
		entity("parent", "Parent Topic", 1, nil, "page-parent"),
	}
	g, err := graph.New(entities, []graph.Edge{parentEdge("e1", "child", "parent")})
	if err != nil {
		t.Fatalf("graph.New: %v", err)
	}

	index := pagemap.NewIndex([]pagemap.Page{
		page("page-child", "/child/", "child"),
		page("page-parent", "/parent/", "parent"),
	})

	lc := plannedFor(dag{g: g, index: index}, "child", template.LinkRules{UpDepth: 1})
	if len(lc.Targets) != 1 || len(lc.Targets[0].Anchors) != 1 || lc.Targets[0].Anchors[0] != "Parent Topic" {
		t.Fatalf("the fallback anchor = %+v", lc.Targets)
	}
}

func TestMayLinkToNamesEveryEntityThatCanOweALink(t *testing.T) {
	t.Parallel()

	fixture := multiParentDAG(t)

	cases := []struct {
		entity string
		want   []string
	}{
		{entity: "coffee", want: []string{"beans", "espresso", "grinder", "orphan", "root", "second", "tea"}},
		{entity: "root", want: []string{"beans", "coffee", "espresso", "grinder", "orphan", "second"}},
		{entity: "espresso", want: []string{"coffee"}},
		{entity: "missing", want: []string{}},
	}

	for _, tc := range cases {
		t.Run(tc.entity, func(t *testing.T) {
			t.Parallel()

			found := content.MayLinkTo(fixture.g, tc.entity)
			ids := make([]string, 0, len(found))
			for i := range found {
				ids = append(ids, found[i].ID)
			}
			slices.Sort(ids)
			if !slices.Equal(ids, tc.want) {
				t.Fatalf("MayLinkTo(%s) = %v, want %v", tc.entity, ids, tc.want)
			}
		})
	}
}
