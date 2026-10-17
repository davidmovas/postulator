package render

import (
	"slices"
	"sort"

	"golang.org/x/net/html"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

const gridProducts = 3

type claim struct {
	node   *html.Node
	kind   RegionKind
	signal string
}

func (p *page) claimRegions() {
	for i := range p.ldTrails {
		p.claimBreadcrumbList(p.ldTrails[i])
	}
	for _, paths := range p.listings {
		member := p.memberOf(paths)
		p.claim(p.cluster(paths, member, false), RegionListing, SignalJSONLD)
	}
	for _, nav := range p.navs {
		p.claim(nav.node, RegionBreadcrumb, nav.signal)
	}
	for _, node := range p.elements(p.navigates) {
		p.claim(node, RegionNavigation, SignalLandmark)
	}
	p.claimProductGrids()
	slices.SortFunc(p.claims, func(a, b claim) int { return p.order[a.node][0] - p.order[b.node][0] })
}

func (p *page) claim(node *html.Node, kind RegionKind, signal string) {
	if node == nil || p.inside(p.root, node) {
		return
	}
	for _, c := range p.claims {
		if c.node == node || (c.kind == kind && (p.inside(node, c.node) || p.inside(c.node, node))) {
			return
		}
	}
	p.claims = append(p.claims, claim{node: node, kind: kind, signal: signal})
}

func (p *page) claimBreadcrumbList(t pagemap.Trail) {
	need := make([]string, 0, len(t.Crumbs))
	for _, crumb := range t.Crumbs {
		if crumb.Path != "" && crumb.Path != p.in.Document && !slices.Contains(need, crumb.Path) {
			need = append(need, crumb.Path)
		}
	}
	if len(need) == 0 {
		return
	}
	member := p.memberOf(append(slices.Clone(need), p.in.Document))
	p.claim(p.cluster(need, member, true), RegionBreadcrumb, SignalJSONLD)
}

func (p *page) memberOf(paths []string) func(*anchor) bool {
	return func(a *anchor) bool {
		return a.link.Kind == pagemap.LinkPath && slices.Contains(paths, a.link.Path)
	}
}

func (p *page) cluster(need []string, member func(*anchor) bool, crumbs bool) *html.Node {
	for _, a := range p.anchors {
		if a.link.Kind != pagemap.LinkPath || !slices.Contains(need, a.link.Path) {
			continue
		}
		node := p.gather(a.node, need, crumbs)
		if node == nil || p.tooWide(node) || !p.pure(node, member) {
			continue
		}
		if crumbs {
			return p.navAround(node, member)
		}
		return p.climb(node, func(up *html.Node) bool { return p.pure(up, member) })
	}
	return nil
}

func (p *page) gather(from *html.Node, need []string, crumbs bool) *html.Node {
	switch {
	case len(need) > 1:
		for node := from.Parent; node != nil; node = node.Parent {
			if p.covers(node, need) {
				return node
			}
		}
		return nil
	case crumbs:
		for node := from.Parent; node != nil; node = node.Parent {
			if named(node, "ol", "ul") || isNav(node) {
				return node
			}
		}
		return nil
	default:
		return from
	}
}

func (p *page) covers(node *html.Node, need []string) bool {
	found := make(map[string]bool)
	for _, a := range p.anchorsUnder(node) {
		if a.link.Kind == pagemap.LinkPath {
			found[a.link.Path] = true
		}
	}
	for _, path := range need {
		if !found[path] {
			return false
		}
	}
	return true
}

func (p *page) pure(node *html.Node, member func(*anchor) bool) bool {
	total, hits := 0, 0
	for _, a := range p.anchorsUnder(node) {
		if a.link.Kind == pagemap.LinkSameDocument {
			continue
		}
		total++
		if member(a) {
			hits++
		}
	}
	return total > 0 && 2*hits > total
}

func (p *page) navAround(node *html.Node, member func(*anchor) bool) *html.Node {
	for up := node; up != nil; up = up.Parent {
		if !isNav(up) {
			continue
		}
		if !p.tooWide(up) && p.pure(up, member) {
			return up
		}
		break
	}
	return node
}

func (p *page) climb(node *html.Node, holds func(*html.Node) bool) *html.Node {
	for up := node.Parent; up != nil && up.Type == html.ElementNode && !p.tooWide(up) && holds(up); up = up.Parent {
		node = up
	}
	return node
}

func (p *page) tooWide(node *html.Node) bool {
	return named(node, "html", "body") || isMain(node) || p.inside(p.root, node)
}

func (p *page) anchorsUnder(node *html.Node) []*anchor {
	span := p.order[node]
	from := sort.Search(len(p.anchors), func(i int) bool { return p.order[p.anchors[i].node][0] >= span[0] })
	to := sort.Search(len(p.anchors), func(i int) bool { return p.order[p.anchors[i].node][0] >= span[1] })
	return p.anchors[from:to]
}

func (p *page) navigates(node *html.Node) bool {
	if hasRole(node, "banner") || hasRole(node, "contentinfo") {
		return true
	}
	if isNav(node) {
		return !p.within(node, isMain)
	}
	if !named(node, "header", "footer") {
		return false
	}
	return !p.within(node, func(up *html.Node) bool { return named(up, "article", "aside", "main", "nav", "section") || isMain(up) })
}

func (p *page) within(node *html.Node, test func(*html.Node) bool) bool {
	for up := node.Parent; up != nil; up = up.Parent {
		if test(up) {
			return true
		}
	}
	return false
}

func (p *page) claimProductGrids() {
	if p.in.Lookup == nil {
		return
	}
	product := func(a *anchor) bool {
		if a.link.Kind != pagemap.LinkPath {
			return false
		}
		wpType, ok := p.in.Lookup(a.link.Path)
		return ok && wpType == pagemap.WPProduct
	}

	products := make(map[*html.Node]map[string]bool)
	for _, a := range p.anchors {
		if !p.inside(a.node, p.root) || !product(a) {
			continue
		}
		for node := a.node.Parent; node != nil && node != p.root; node = node.Parent {
			if products[node] == nil {
				products[node] = make(map[string]bool)
			}
			products[node][a.link.Path] = true
		}
	}

	grids := make([]*html.Node, 0)
	for node, paths := range products {
		if len(paths) >= gridProducts && !p.holdsGrid(node, products) {
			grids = append(grids, node)
		}
	}
	slices.SortFunc(grids, func(a, b *html.Node) int { return p.order[a][0] - p.order[b][0] })

	for _, grid := range grids {
		if p.holdsContent(grid) {
			continue
		}
		p.claim(p.climb(grid, func(up *html.Node) bool { return p.pure(up, product) && !p.holdsContent(up) }), RegionListing, SignalLinks)
	}
}

func (p *page) holdsGrid(node *html.Node, products map[*html.Node]map[string]bool) bool {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if len(products[child]) >= gridProducts {
			return true
		}
	}
	return false
}

func (p *page) holdsContent(node *html.Node) bool {
	from, to := p.text.wordRange(node)
	for _, r := range p.content {
		if from < r.end && r.start < to {
			return true
		}
	}
	return false
}

func (p *page) labels() (contentAt, restAt, unknownAt int) {
	return len(p.claims), len(p.claims) + 1, len(p.claims) + 2
}

func (p *page) regions() (regions []Region, wordsContent, wordsGenerated int) {
	contentAt, restAt, unknownAt := p.labels()
	labels := make([]int, len(p.text.words))
	paint := func(from, to, label int) {
		for i := from; i < to; i++ {
			labels[i] = label
		}
	}
	paint(0, len(labels), unknownAt)
	rootFrom, rootTo := p.text.wordRange(p.root)
	paint(rootFrom, rootTo, restAt)
	for _, r := range p.content {
		paint(r.start, r.end, contentAt)
	}
	for i := range p.claims {
		from, to := p.text.wordRange(p.claims[i].node)
		paint(from, to, i)
	}

	words := make([]int, unknownAt+1)
	for _, label := range labels {
		words[label]++
	}
	links := make([]int, unknownAt+1)
	for _, a := range p.anchors {
		links[a.region]++
	}

	regions = make([]Region, 0, len(p.claims)+3)
	for i, c := range p.claims {
		regions = append(regions, Region{Kind: c.kind, Inside: p.inside(c.node, p.root), Links: links[i], Words: words[i], Signal: c.signal})
	}
	for _, extra := range []Region{
		{Kind: RegionContent, Inside: true, Links: links[contentAt], Words: words[contentAt]},
		{Kind: p.rest(), Inside: true, Links: links[restAt], Words: words[restAt]},
		{Kind: RegionUnknown, Links: links[unknownAt], Words: words[unknownAt]},
	} {
		if extra.Links > 0 || extra.Words > 0 {
			regions = append(regions, extra)
		}
	}
	return regions, words[contentAt], max(rootTo-rootFrom-words[contentAt], 0)
}

func (p *page) headings() []Heading {
	heading := func(n *html.Node) bool { return named(n, "h1", "h2", "h3", "h4", "h5", "h6") }
	nodes := matching(p.root, heading)
	if heading(p.root) {
		nodes = append([]*html.Node{p.root}, nodes...)
	}
	out := make([]Heading, 0, len(nodes))
	for _, node := range nodes {
		from, to := p.text.wordRange(node)
		if from >= to {
			continue
		}
		out = append(out, Heading{
			Level:     int(node.Data[1] - '0'),
			Text:      content.TextOf(node),
			Generated: !p.inRaw(p.text.words[from:to]),
		})
	}
	return out
}

func (p *page) inRaw(words []string) bool {
	for i := range p.blocks {
		hay := p.blocks[i].words
		for at := 0; at+len(words) <= len(hay); at++ {
			if slices.Equal(hay[at:at+len(words)], words) {
				return true
			}
		}
	}
	return false
}
