package content_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/template"
)

func storedLink(toPageID, toURL string) pagemap.PageLink {
	link := pagemap.PageLink{SiteID: siteID, FromPageID: "page-coffee", ToURL: toURL, AnchorText: "x"}
	if toPageID != "" {
		link.ToPageID = ref(toPageID)
	}
	return link
}

func TestClassifyLink(t *testing.T) {
	t.Parallel()

	fixture := multiParentDAG(t)
	rules := template.LinkRules{UpDepth: 2, DownLinks: true, SiblingMinWeight: 0.5}
	lc := content.BuildLinkContext(fixture.g, fixture.index, "coffee", policy(rules))
	const host = "shop.example.com"

	cases := []struct {
		name string
		link pagemap.PageLink
		want content.LinkClass
	}{
		{"a resolved link to a target", storedLink("page-root", "/drinks/"), content.ClassGraph},
		{"a resolved link to the page itself", storedLink("page-coffee", "/drinks/hot/coffee/"), content.ClassSelf},
		{"a resolved link the graph did not ask for", storedLink("page-x", "/x/"), content.ClassUnknownInternal},
		{"an unresolved link to another host", storedLink("", "https://example.org/"), content.ClassExternal},
		{"an unresolved absolute link to the own host", storedLink("", "https://shop.example.com/drinks/"), content.ClassGraph},
		{"an unresolved relative link to nowhere", storedLink("", "/nowhere/"), content.ClassUnknownInternal},
		{"an unresolved relative link to the own path", storedLink("", "/drinks/hot/coffee/"), content.ClassSelf},
		{"a mail link", storedLink("", "mailto:x@example.org"), content.ClassExternal},
		{"an absolute link to a sibling host", storedLink("", "https://other.example.com/drinks/"), content.ClassExternal},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := lc.ClassifyLink(tc.link, host); got != tc.want {
				t.Fatalf("ClassifyLink = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestOffGraph(t *testing.T) {
	t.Parallel()

	cases := map[content.LinkClass]bool{
		content.ClassGraph:           false,
		content.ClassSelf:            true,
		content.ClassExternal:        true,
		content.ClassUnknownInternal: true,
	}
	for class, want := range cases {
		if got := class.OffGraph(); got != want {
			t.Errorf("%s.OffGraph() = %v, want %v", class, got, want)
		}
	}
}

func TestAnchorAllowed(t *testing.T) {
	t.Parallel()

	target := content.LinkTarget{Anchors: []string{"Drinks", "all drinks"}}
	cases := []struct {
		anchor string
		want   bool
	}{
		{"Drinks", true},
		{"  drinks ", true},
		{"ALL DRINKS", true},
		{"drink", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := content.AnchorAllowed(target, tc.anchor); got != tc.want {
			t.Errorf("AnchorAllowed(%q) = %v, want %v", tc.anchor, got, tc.want)
		}
	}
}
