package content_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func twinFixture(t *testing.T) dag {
	t.Helper()

	entities := []graph.Entity{
		entity("peptides", "Peptides", 1, []string{"peptides"}, "page-term"),
		entity("bpc", "BPC-157", 0.6, []string{"bpc-157"}, "page-bpc"),
		entity("guide", "Peptides guide", 0.5, []string{"peptide guide"}, "page-hub"),
	}
	edges := []graph.Edge{
		parentEdge("e1", "bpc", "peptides"),
		parentEdge("e2", "guide", "peptides"),
		relatedEdge("e3", "bpc", "guide", 0.8),
	}
	g, err := graph.New(entities, edges)
	if err != nil {
		t.Fatalf("graph.New: %v", err)
	}

	termID, hubID := int64(20), int64(10)
	term := page("page-term", "/product-category/peptides/", "peptides")
	term.WPType, term.WPID, term.Observed.Title = pagemap.WPProductCategory, &termID, "Peptides"
	hub := page("page-hub", "/peptides/", "guide")
	hub.WPID, hub.Observed.Title = &hubID, "Peptides"
	return dag{g: g, index: pagemap.NewIndex([]pagemap.Page{term, hub, page("page-bpc", "/bpc-157/", "bpc")})}
}

func twinPolicy() field.LinkPolicy {
	return policy(field.LinkRules{UpDepth: 1, SiblingMinWeight: 0.5, MaxLinks: 10, MaxPerTarget: 1})
}

func TestATargetPrefersTheHubOverItsTwinArchive(t *testing.T) {
	t.Parallel()

	fixture := twinFixture(t)
	plan := content.PlanLinks(fixture.g, fixture.index, content.Subject{PageID: "page-bpc", PagePath: "/bpc-157/", EntityID: "bpc"}, twinPolicy())

	if len(plan.Context.Targets) != 1 {
		t.Fatalf("targets = %v, want the hub once, the archive and the related hub merged", urls(plan.Context.Targets))
	}
	got := plan.Context.Targets[0]
	if got.PageID != "page-hub" || got.URL != "/peptides/" {
		t.Errorf("target = %+v, want the hub page rather than its category archive", got)
	}
	if !got.Required || got.Relation != content.RelationUp {
		t.Errorf("target = %+v, want the up link's requirement kept through the merge", got)
	}
	if len(got.Anchors) != 2 || got.Anchors[0] != "peptides" || got.Anchors[1] != "peptide guide" {
		t.Errorf("anchors = %v, want both entities' anchors", got.Anchors)
	}
}

func TestATwinThatIsThePageItselfIsDropped(t *testing.T) {
	t.Parallel()

	fixture := twinFixture(t)
	plan := content.PlanLinks(fixture.g, fixture.index, content.Subject{PageID: "page-hub", PagePath: "/peptides/", EntityID: "guide"}, twinPolicy())

	for _, target := range plan.Context.Targets {
		if target.PageID == "page-hub" || target.PageID == "page-term" {
			t.Errorf("target = %+v, want no link from the hub to itself or to its own archive", target)
		}
	}
}

func TestATargetTheClientAlreadyLinksIsNotOwedAgain(t *testing.T) {
	t.Parallel()

	fixture := twinFixture(t)
	plan := content.PlanLinks(fixture.g, fixture.index, content.Subject{
		Site: pagemap.NewSite("https://shop.example.com"), PageID: "page-bpc", PagePath: "/bpc-157/", EntityID: "bpc",
	}, twinPolicy())
	theirs, err := content.Parse(`<p>See our <a href="https://shop.example.com/peptides/">peptides</a>.</p>`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if left := plan.Context.Unlinked(theirs); len(left.Targets) != 0 {
		t.Errorf("targets left = %v, want the hub the client links already gone", urls(left.Targets))
	}
	if left := plan.Context.Unlinked(nil); len(left.Targets) != len(plan.Context.Targets) {
		t.Errorf("no document removed %d targets", len(plan.Context.Targets)-len(left.Targets))
	}
}

func TestOurLinksFollowTheSitesContextualForm(t *testing.T) {
	t.Parallel()

	fixture := twinFixture(t)
	site := pagemap.NewSite("https://shop.example.com")
	cases := []struct {
		name     string
		absolute bool
		want     string
	}{
		{name: "relative", want: "/peptides/"},
		{name: "absolute", absolute: true, want: "https://shop.example.com/peptides/"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			subject := content.Subject{Site: site, PageID: "page-bpc", PagePath: "/bpc-157/", EntityID: "bpc", Absolute: tc.absolute}
			plan := content.PlanLinks(fixture.g, fixture.index, subject, twinPolicy())
			if len(plan.Context.Targets) != 1 {
				t.Fatalf("targets = %v", urls(plan.Context.Targets))
			}
			target := plan.Context.Targets[0]
			if href := plan.Context.Href(target); href != tc.want || target.URL != "/peptides/" {
				t.Errorf("href %q, url %q; want href %q and the url kept as the path", href, target.URL, tc.want)
			}

			doc, err := content.Parse("<p>Read about peptides here.</p>")
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			content.InsertLinks(doc, plan.Context, twinPolicy())
			links := doc.Links()
			if len(links) != 1 || links[0].Href != tc.want {
				t.Errorf("placed links = %+v, want one link written as %q", links, tc.want)
			}
		})
	}
}
