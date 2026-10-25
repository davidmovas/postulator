package render

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"iter"
	"slices"
	"strings"

	"golang.org/x/net/html"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type page struct {
	in        Input
	body      block.Body
	doc       *html.Node
	text      *stream
	order     map[*html.Node][2]int
	anchors   []*anchor
	blocks    []rawBlock
	positions map[string][]int
	root      *html.Node
	located   bool
	state     RootState
	content   []run
	objects   []ldObject
	ldTrails  []pagemap.Trail
	listings  [][]string
	navs      []breadcrumbNav
	claims    []claim
}

func Extract(in Input) (Extraction, error) {
	sum := sha256.Sum256([]byte(in.Public))
	body := technicalBody(in.Body, in.Dynamic)
	out := Extraction{
		Hash:     hex.EncodeToString(sum[:]),
		Root:     BodyState(in.Body, in.Archive, in.Dynamic),
		Regions:  []Region{},
		Trails:   []pagemap.Trail{},
		JSONLD:   map[string][]json.RawMessage{},
		Links:    []Link{},
		Headings: []Heading{},
	}
	if strings.TrimSpace(in.Public) == "" {
		return out, nil
	}

	doc, err := html.Parse(strings.NewReader(in.Public))
	if err != nil {
		return Extraction{}, errors.Wrap(err, errors.Invalid, "the public render is not parseable html")
	}

	p := newPage(in, body, doc)
	p.findRoot()
	rootFrom, rootTo := p.text.wordRange(p.root)
	p.content = p.matchBlocks(p.blocks, rootFrom, rootTo)
	p.readJSONLD()
	p.readNavs()
	p.claimRegions()
	p.assign()

	out.Root = p.state
	out.JSONLD = p.typed()
	out.Trails = append(append(out.Trails, p.ldTrails...), p.navTrails()...)
	out.Links = p.typedLinks()
	out.Regions, out.WordsContent, out.WordsGenerated = p.regions()
	out.Headings = p.headings()
	out.ThemeH1 = p.themeH1(out.Headings)
	out.H1, out.H1Source = p.visibleH1()
	out.Builder = p.builder()
	out.URLForm = p.urlForm()
	out.Robots = p.robots()
	return out, nil
}

func newPage(in Input, body block.Body, doc *html.Node) *page {
	p := &page{in: in, body: body, doc: doc, text: newStream(doc), order: make(map[*html.Node][2]int)}
	p.index()
	p.collectAnchors()
	p.collectBlocks()
	return p
}

func (p *page) index() {
	next := 0
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		start := next
		next++
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
		p.order[node] = [2]int{start, next}
	}
	walk(p.doc)
}

func (p *page) inside(node, ancestor *html.Node) bool {
	n, a := p.order[node], p.order[ancestor]
	return a[0] <= n[0] && n[0] < a[1]
}

func elementsUnder(node *html.Node, visible bool) iter.Seq[*html.Node] {
	return func(yield func(*html.Node) bool) {
		descend(node, visible, yield)
	}
}

func descend(node *html.Node, visible bool, yield func(*html.Node) bool) bool {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != html.ElementNode {
			continue
		}
		if visible && slices.Contains(unseenElements, child.Data) {
			continue
		}
		if !yield(child) || !descend(child, visible, yield) {
			return false
		}
	}
	return true
}

func (p *page) elements(test func(*html.Node) bool) []*html.Node {
	return matching(p.doc, test)
}

func matching(node *html.Node, test func(*html.Node) bool) []*html.Node {
	out := make([]*html.Node, 0)
	for n := range elementsUnder(node, true) {
		if test(n) {
			out = append(out, n)
		}
	}
	return out
}

func (p *page) first(test func(*html.Node) bool) *html.Node {
	return firstUnder(p.doc, test)
}

func firstUnder(node *html.Node, test func(*html.Node) bool) *html.Node {
	for n := range elementsUnder(node, true) {
		if test(n) {
			return n
		}
	}
	return nil
}

func named(node *html.Node, names ...string) bool {
	return node != nil && node.Type == html.ElementNode && slices.Contains(names, node.Data)
}

func attr(node *html.Node, key string) (string, bool) {
	for _, a := range node.Attr {
		if a.Key == key {
			return a.Val, true
		}
	}
	return "", false
}

func hasRole(node *html.Node, role string) bool {
	value, ok := attr(node, "role")
	return ok && slices.Contains(strings.Fields(strings.ToLower(value)), role)
}

func isMain(node *html.Node) bool {
	return named(node, "main") || hasRole(node, "main")
}

func isNav(node *html.Node) bool {
	return named(node, "nav") || hasRole(node, "navigation")
}

func isAside(node *html.Node) bool {
	return named(node, "aside") || hasRole(node, "complementary")
}

func (p *page) robots() string {
	for node := range elementsUnder(p.doc, false) {
		if !named(node, "meta") {
			continue
		}
		if name, _ := attr(node, "name"); !strings.EqualFold(strings.TrimSpace(name), "robots") {
			continue
		}
		value, _ := attr(node, "content")
		return strings.ToLower(strings.TrimSpace(value))
	}
	return ""
}

func isHrefAnchor(node *html.Node) bool {
	if !named(node, "a") {
		return false
	}
	_, ok := attr(node, "href")
	return ok
}
