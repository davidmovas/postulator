package content_test

import (
	"slices"
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
	lc.Site = pagemap.NewSite("https://shop.example.com")

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

			if got := lc.ClassifyLink(tc.link); got != tc.want {
				t.Fatalf("ClassifyLink = %s, want %s", got, tc.want)
			}
		})
	}
}

func resolverContext(nested bool) content.LinkContext {
	if nested {
		return content.LinkContext{
			Site: pagemap.NewSite("https://h.example.com/blog"), PageID: "page-guide", PageURL: "/blog/guide/",
			EntityID: "guide",
			Targets: []content.LinkTarget{
				target("/blog/shop/", []string{"shop"}, content.RelationDown, false),
			},
		}
	}
	return content.LinkContext{
		Site: pagemap.NewSite("https://shop.example.com"), PageID: "page-guide", PageURL: "/guide/",
		EntityID: "guide",
		Targets: []content.LinkTarget{
			target("/shop/", []string{"shop"}, content.RelationDown, false),
			target("/café/", []string{"café"}, content.RelationDown, false),
		},
	}
}

func anchorFor(lc content.LinkContext, url string) string {
	for _, candidate := range lc.Targets {
		if candidate.URL == url {
			return candidate.Anchors[0]
		}
	}
	return "link"
}

func alreadyLinked(result content.InsertResult) []string {
	out := make([]string, 0, 1)
	for i := range result.Decisions {
		if result.Decisions[i].Outcome == content.OutcomeAlreadyLinked {
			out = append(out, result.Decisions[i].Target.URL)
		}
	}
	return out
}

func complianceClass(report content.Report) content.LinkClass {
	for _, item := range report.Items {
		switch item.Code {
		case content.CodeExternalLink:
			return content.ClassExternal
		case content.CodeSelfLink:
			return content.ClassSelf
		case content.CodeUnknownInternal:
			return content.ClassUnknownInternal
		}
	}
	return content.ClassGraph
}

func TestEveryClassifierResolvesAnHrefAlike(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		href      string
		nested    bool
		want      content.LinkClass
		satisfies string
		sameDoc   bool
	}{
		{name: "a bare path", href: "/shop", want: content.ClassGraph, satisfies: "/shop/"},
		{name: "another casing", href: "/Shop/", want: content.ClassGraph, satisfies: "/shop/"},
		{name: "a tracking query", href: "/shop/?utm=x", want: content.ClassGraph, satisfies: "/shop/"},
		{name: "a fragment on the target", href: "/shop/#top", want: content.ClassGraph, satisfies: "/shop/"},
		{name: "the own host over https", href: "https://shop.example.com/shop/", want: content.ClassGraph, satisfies: "/shop/"},
		{name: "the own host over http", href: "http://shop.example.com/shop/", want: content.ClassGraph, satisfies: "/shop/"},
		{name: "www is another host", href: "https://www.shop.example.com/shop/", want: content.ClassExternal},
		{name: "a percent escaped accent", href: "/caf%C3%A9/", want: content.ClassGraph, satisfies: "/café/"},
		{name: "the same accent written out", href: "/café/", want: content.ClassGraph, satisfies: "/café/"},
		{name: "an escaped slash is one segment", href: "/a%2Fb/", want: content.ClassUnknownInternal},
		{name: "the page itself", href: "/guide/", want: content.ClassSelf},
		{name: "a mail link", href: "mailto:hello@shop.example.com", want: content.ClassExternal},
		{name: "an unrelated host", href: "https://other.example.org/", want: content.ClassExternal},
		{name: "a fragment alone", href: "#faq", want: content.ClassSelf, sameDoc: true},
		{name: "a query alone", href: "?utm=x", want: content.ClassSelf, sameDoc: true},
		{
			name: "a subdirectory path as written", href: "/blog/shop/", nested: true,
			want: content.ClassGraph, satisfies: "/blog/shop/",
		},
		{
			name: "a subdirectory path absolute", href: "https://h.example.com/blog/shop/", nested: true,
			want: content.ClassGraph, satisfies: "/blog/shop/",
		},
		{
			name: "a relative href joins the subdirectory base", href: "shop/", nested: true,
			want: content.ClassGraph, satisfies: "/blog/shop/",
		},
		{
			name: "the server root is not the subdirectory root", href: "/shop/", nested: true,
			want: content.ClassUnknownInternal,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			lc := resolverContext(tc.nested)
			anchor := "link"
			if tc.satisfies != "" {
				anchor = anchorFor(lc, tc.satisfies)
			}
			body := `<p>Our shop and a café. <a href="` + tc.href + `">` + anchor + `</a></p>`

			if got := lc.ClassifyLink(storedLink("", tc.href)); got != tc.want {
				t.Errorf("ClassifyLink(%q) = %s, want %s", tc.href, got, tc.want)
			}
			resolution := lc.Resolve(tc.href)
			if resolution.Class != tc.want || resolution.SameDocument != tc.sameDoc {
				t.Errorf("Resolve(%q) = %s sameDocument=%t, want %s sameDocument=%t",
					tc.href, resolution.Class, resolution.SameDocument, tc.want, tc.sameDoc)
			}

			report := content.Compliance(mustParse(t, body), lc, policy(template.LinkRules{}), lc.PageID)
			wantCompliance := tc.want
			if tc.sameDoc {
				wantCompliance = content.ClassGraph
			}
			if got := complianceClass(report); got != wantCompliance {
				t.Errorf("Compliance(%q) reads as %s, want %s (%v)", tc.href, got, wantCompliance, codesOf(report))
			}

			linked := alreadyLinked(content.InsertLinks(mustParse(t, body), lc, policy(template.LinkRules{})))
			want := []string{}
			if tc.satisfies != "" {
				want = []string{tc.satisfies}
			}
			if !slices.Equal(linked, want) {
				t.Errorf("InsertLinks(%q) already linked %v, want %v", tc.href, linked, want)
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
