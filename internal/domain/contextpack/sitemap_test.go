package contextpack_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
)

func mapped5mg(t *testing.T) contextpack.Snapshot {
	t.Helper()

	s := withFindings(t)
	pages := s.Index.Pages()
	for i := range pages {
		if pages[i].ID == "bpc-5" {
			pages[i] = under(pages[i], "term", pagemap.ParentJSONLD)
		}
	}
	s.Index = pagemap.NewIndex(pages)
	s.Findings = append(s.Findings, open(finding.BuilderDetected, "term"))
	return s
}

func pathsOf(nodes []contextpack.MapNode) []string {
	out := make([]string, 0, len(nodes))
	for i := range nodes {
		out = append(out, nodes[i].Page.Path)
	}
	return out
}

func TestTheMapFoldsAStoreCategoryIntoItsTwinHub(t *testing.T) {
	t.Parallel()

	m := mapped5mg(t).Map(map[string]contextpack.LinkFacts{
		"hub": {Inbound: 3, Outbound: 1}, "term": {Inbound: 2},
	})

	roots := pathsOf(m.Children(""))
	if want := []string{"/", "/product/bpc-157-10mg/", "/product/ghk-cu/", "/product/tb-500/"}; !slices.Equal(roots, want) {
		t.Errorf("roots = %v, want %v", roots, want)
	}
	if home := pathsOf(m.Children("home")); !slices.Equal(home, []string{"/lab-guides/", "/peptides/"}) {
		t.Errorf("under the front page = %v", home)
	}
	hubChildren := pathsOf(m.Children("hub"))
	if want := []string{"/bpc-157/", "/healing-peptides/", "/peptides/tb-500-guide/", "/product/bpc-157-5mg/"}; !slices.Equal(hubChildren, want) {
		t.Errorf("under the hub = %v, want %v: the store category's own child joins its twin", hubChildren, want)
	}
	if folded := pathsOf(m.Children("term")); !slices.Equal(folded, hubChildren) {
		t.Errorf("under the store category = %v, want the hub's children: they are one node", folded)
	}

	hub, found := m.Node("hub")
	if !found || hub.Twin == nil || hub.Twin.Path != "/product-category/peptides/" {
		t.Fatalf("the hub = %+v, want it to carry its store category", hub)
	}
	if hub.Children != 4 || hub.Descendants != 4 {
		t.Errorf("the hub has %d children and %d descendants, want 4 and 4", hub.Children, hub.Descendants)
	}
	if hub.Findings != 1 || hub.LinksIn != 5 || hub.LinksOut != 1 {
		t.Errorf("the hub counts %d findings, %d links in and %d out; want its twin's added: 1, 5, 1", hub.Findings, hub.LinksIn, hub.LinksOut)
	}
	if term, found := m.Node("term"); !found || term.Page.PageID != "hub" {
		t.Errorf("the store category's node = %+v, want the hub's", term)
	}
	if home, _ := m.Node("home"); home.Descendants != 6 {
		t.Errorf("the front page has %d descendants, want 6", home.Descendants)
	}

	seen := map[string]int{}
	var walk func(parent string)
	walk = func(parent string) {
		for _, node := range m.Children(parent) {
			seen[node.Page.PageID]++
			walk(node.Page.PageID)
		}
	}
	walk("")
	for id, times := range seen {
		if times != 1 {
			t.Errorf("%s appears %d times in the map", id, times)
		}
	}
	if seen["term"] != 0 || seen["gone"] != 0 {
		t.Errorf("the map shows the folded store category %d times and the archived page %d times, want neither", seen["term"], seen["gone"])
	}
	if len(seen) != 10 {
		t.Errorf("the map holds %d rows, want 10: every live page but the folded store category", len(seen))
	}
}

func TestAMapNodeSaysWhatThePageIs(t *testing.T) {
	t.Parallel()

	m := mapped5mg(t).Map(nil)
	planned, _ := m.Node("planned")
	if planned.Page.OnSite || planned.ParentReason != pagemap.ParentWorkbook || planned.Page.Type != pagetype.Entity.DefaultName() {
		t.Errorf("the planned page = %+v, want it off the site, under the workbook's parent, typed Entity", planned)
	}
	bpc, _ := m.Node("bpc")
	if bpc.ParentReason != pagemap.ParentJSONLD || bpc.Findings != 1 || bpc.Page.Preset != pagetype.Entity {
		t.Errorf("BPC-157 = %+v, want a JSON-LD parent and its one finding", bpc)
	}
	if _, found := m.Node("gone"); found {
		t.Error("an archived page has no node")
	}
	if bpc.EntityID != "e-bpc" || bpc.Status != pagemap.StatusPublished {
		t.Errorf("BPC-157 carries entity %q and status %q, want what the run start's picker reads", bpc.EntityID, bpc.Status)
	}
	if planned.Status != pagemap.StatusPlanned {
		t.Errorf("the planned page carries status %q", planned.Status)
	}
	if hub, _ := m.Node("guides"); hub.EntityID != "" {
		t.Errorf("an unmapped page carries entity %q", hub.EntityID)
	}
}

func TestTheMapFindsPagesByTextTypeAndSubtree(t *testing.T) {
	t.Parallel()

	m := mapped5mg(t).Map(nil)
	cases := []struct {
		name  string
		query contextpack.MapQuery
		want  []string
	}{
		{name: "a title in any case", query: contextpack.MapQuery{Text: "bpc"}, want: []string{"/bpc-157/", "/product/bpc-157-10mg/", "/product/bpc-157-5mg/"}},
		{name: "the twin's path finds the hub", query: contextpack.MapQuery{Text: "product-category"}, want: []string{"/peptides/"}},
		{name: "a type", query: contextpack.MapQuery{TypeID: typeIDOf(pagetype.Product)}, want: []string{"/product/bpc-157-10mg/", "/product/bpc-157-5mg/", "/product/ghk-cu/", "/product/tb-500/"}},
		{name: "the twin's type finds the hub", query: contextpack.MapQuery{TypeID: typeIDOf(pagetype.StoreCategory)}, want: []string{"/peptides/"}},
		{name: "text under a subtree", query: contextpack.MapQuery{Under: "hub", Text: "BPC"}, want: []string{"/bpc-157/", "/product/bpc-157-5mg/"}},
		{name: "a type under a subtree", query: contextpack.MapQuery{Under: "home", TypeID: typeIDOf(pagetype.Hub)}, want: []string{"/healing-peptides/", "/peptides/"}},
		{name: "nothing", query: contextpack.MapQuery{Text: "espresso"}, want: []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := pathsOf(m.Matches(tc.query)); !slices.Equal(got, tc.want) {
				t.Errorf("Matches(%+v) = %v, want %v", tc.query, got, tc.want)
			}
		})
	}

	five := m.Matches(contextpack.MapQuery{Text: "5 mg"})
	if len(five) != 1 {
		t.Fatalf("the 5 mg product = %v", pathsOf(five))
	}
	trail := make([]string, 0, len(five[0].Trail))
	for _, ref := range five[0].Trail {
		trail = append(trail, ref.Title)
	}
	if !slices.Equal(trail, []string{"Home", "Peptides"}) {
		t.Errorf("the 5 mg product's trail = %v, want Home › Peptides through the folded store category", trail)
	}
}

func TestUnderNamesAPageItsTwinAndEveryDescendant(t *testing.T) {
	t.Parallel()

	m := mapped5mg(t).Map(nil)
	got := m.Under("hub")
	slices.Sort(got)
	if want := []string{"bpc", "bpc-5", "healing", "hub", "planned", "term"}; !slices.Equal(got, want) {
		t.Errorf("Under(hub) = %v, want %v", got, want)
	}
	if fromTerm := m.Under("term"); len(fromTerm) != len(got) {
		t.Errorf("Under(term) = %v, want the hub's subtree", fromTerm)
	}
	if none := m.Under("gone"); len(none) != 0 {
		t.Errorf("Under an archived page = %v, want nothing", none)
	}
}

func TestAPlaceNamesAnyPageWithItsType(t *testing.T) {
	t.Parallel()

	m := mapped5mg(t).Map(nil)
	term, found := m.Place("term")
	if !found || term.Ref.Path != "/product-category/peptides/" || term.TypeID != typeIDOf(pagetype.StoreCategory) {
		t.Errorf("the store category's place = %+v, %v", term, found)
	}
	if gone, found := m.Place("gone"); !found || gone.Ref.Title != "Old page" {
		t.Errorf("an archived page's place = %+v, %v; want it named still, for the findings it carries", gone, found)
	}
	if _, found := m.Place("nobody"); found {
		t.Error("a page nobody holds has no place")
	}
}

func TestALoopTheTwinsMakeIsBrokenAtItsFirstPath(t *testing.T) {
	t.Parallel()

	f := replicaLike()
	for i := range f.pages {
		switch f.pages[i].ID {
		case "hub":
			f.pages[i] = under(f.pages[i], "guides", pagemap.ParentJSONLD)
		case "guides":
			f.pages[i] = under(f.pages[i], "term", pagemap.ParentJSONLD)
		}
	}
	m := f.snapshot(t).Map(nil)
	roots := pathsOf(m.Children(""))
	if !slices.Contains(roots, "/lab-guides/") || slices.Contains(roots, "/peptides/") {
		t.Errorf("roots = %v, want the loop cut at /lab-guides/, its first path", roots)
	}
	if under := pathsOf(m.Children("guides")); !slices.Contains(under, "/peptides/") {
		t.Errorf("under the guides = %v, want the hub", under)
	}
}

func TestTheBodyStateSaysWhoseTextAPageHolds(t *testing.T) {
	t.Parallel()

	ours := "<p>Our paragraph about peptides.</p>"
	theirs := "<p>The client's own paragraph.</p>"
	ourHash := block.Parse(ours).Blocks[0].Hash
	cases := []struct {
		name    string
		raw     string
		written []string
		want    contextpack.BodyState
	}{
		{name: "nothing", raw: "", want: contextpack.BodyEmpty},
		{name: "the client's text", raw: theirs, want: contextpack.BodyClient},
		{name: "our text", raw: ours, written: []string{ourHash}, want: contextpack.BodyOurs},
		{name: "both", raw: theirs + "\n\n" + ours, written: []string{ourHash}, want: contextpack.BodyMixed},
		{name: "a shortcode only", raw: `[products limit="4"]`, want: contextpack.BodyDynamic},
		{name: "a textless figure beside our text leaves the text ours", raw: `<figure><img src="/a.jpg" alt=""></figure>` + "\n\n" + ours, written: []string{ourHash}, want: contextpack.BodyOurs},
		{name: "the client's captioned figure beside our text", raw: `<figure><img src="/a.jpg" alt=""><figcaption>His lab</figcaption></figure>` + "\n\n" + ours, written: []string{ourHash}, want: contextpack.BodyMixed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := contextpack.BodyOf(tc.raw, tc.written); got != tc.want {
				t.Errorf("BodyOf(%q) = %q, want %q", strings.TrimSpace(tc.raw), got, tc.want)
			}
		})
	}
}
