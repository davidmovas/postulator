package render

import (
	"net/url"
	"strings"

	"golang.org/x/net/html"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

type anchor struct {
	node     *html.Node
	link     Link
	key      string
	region   int
	kind     RegionKind
	position int
	self     bool
}

type rawBlock struct {
	index int
	words []string
	links []string
}

func (p *page) collectAnchors() {
	for _, node := range p.elements(isHrefAnchor) {
		href, _ := attr(node, "href")
		path, kind := p.in.Site.ResolveAt(p.in.Document, href)
		name := accessibleName(node)
		p.anchors = append(p.anchors, &anchor{
			node:     node,
			link:     Link{Href: href, Path: path, Kind: kind, Anchor: name},
			key:      linkKey(path, kind, href, name),
			position: -1,
			self:     kind == pagemap.LinkSameDocument || (kind == pagemap.LinkPath && path == p.in.Document),
		})
	}
}

func (p *page) collectBlocks() {
	p.positions = make(map[string][]int)
	for i := range p.body.Blocks {
		b := &p.body.Blocks[i]
		if b.Dynamic || b.Kind == block.Shortcode {
			continue
		}
		nodes := parseFragment(b.Raw)
		keys := p.linkKeys(nodes...)
		for _, key := range keys {
			p.positions[key] = append(p.positions[key], b.Index)
		}
		if words := wordsOf(nodes...); len(words) > 0 {
			p.blocks = append(p.blocks, rawBlock{index: b.Index, words: words, links: keys})
		}
	}
}

func (p *page) linkKeys(nodes ...*html.Node) []string {
	out := make([]string, 0)
	for _, node := range nodes {
		anchors := matching(node, isHrefAnchor)
		if isHrefAnchor(node) {
			anchors = append([]*html.Node{node}, anchors...)
		}
		for _, a := range anchors {
			href, _ := attr(a, "href")
			path, kind := p.in.Site.ResolveAt(p.in.Document, href)
			out = append(out, linkKey(path, kind, href, accessibleName(a)))
		}
	}
	return out
}

func linkKey(path string, kind pagemap.LinkKind, href, name string) string {
	target := path
	if kind != pagemap.LinkPath {
		target = strings.TrimSpace(href)
	}
	return string(kind) + "\x00" + target + "\x00" + strings.Join(content.WordsOf(name), " ")
}

func accessibleName(node *html.Node) string {
	if text := content.TextOf(node); text != "" {
		return text
	}
	if label, _ := attr(node, "aria-label"); strings.TrimSpace(label) != "" {
		return strings.TrimSpace(label)
	}
	for img := range elementsUnder(node, true) {
		if !named(img, "img") {
			continue
		}
		if alt, _ := attr(img, "alt"); strings.TrimSpace(alt) != "" {
			return strings.TrimSpace(alt)
		}
	}
	return ""
}

func (p *page) resolved(href string) string {
	path, kind := p.in.Site.ResolveAt(p.in.Document, href)
	if kind != pagemap.LinkPath {
		return ""
	}
	return path
}

func (p *page) assign() {
	consumed := make(map[string]int)
	contentAt, restAt, unknownAt := p.labels()
	for _, a := range p.anchors {
		if at := p.innermost(a.node); at >= 0 {
			a.region, a.kind = at, p.claims[at].kind
			continue
		}
		if !p.inside(a.node, p.root) {
			a.region, a.kind = unknownAt, RegionUnknown
			continue
		}
		if position, ok := p.position(a.key, consumed); ok {
			a.region, a.kind, a.position = contentAt, RegionContent, position
			continue
		}
		a.region, a.kind = restAt, p.rest()
	}
}

func (p *page) position(key string, consumed map[string]int) (int, bool) {
	indexes := p.positions[key]
	used := consumed[key]
	if used >= len(indexes) {
		return -1, false
	}
	consumed[key] = used + 1
	return indexes[used], true
}

func (p *page) innermost(node *html.Node) int {
	found := -1
	for i := range p.claims {
		if !p.inside(node, p.claims[i].node) {
			continue
		}
		if found < 0 || p.inside(p.claims[i].node, p.claims[found].node) {
			found = i
		}
	}
	return found
}

func (p *page) rest() RegionKind {
	if p.body.DynamicOnly() {
		return RegionContentDynamic
	}
	return RegionGenerated
}

func linkType(kind RegionKind) pagemap.LinkType {
	switch kind {
	case RegionBreadcrumb:
		return pagemap.LinkBreadcrumb
	case RegionListing:
		return pagemap.LinkListing
	case RegionNavigation:
		return pagemap.LinkNavigation
	case RegionContent:
		return pagemap.LinkContextual
	default:
		return pagemap.LinkUnknown
	}
}

func (p *page) typedLinks() []Link {
	out := make([]Link, 0, len(p.anchors))
	for _, a := range p.anchors {
		if a.self {
			continue
		}
		link := a.link
		link.Region = a.kind
		link.Type = linkType(a.kind)
		link.Position = a.position
		out = append(out, link)
	}
	return out
}

func (p *page) urlForm() string {
	absolute, internal := 0, 0
	for _, a := range p.anchors {
		if a.link.Kind != pagemap.LinkPath {
			continue
		}
		internal++
		if absoluteHref(a.link.Href) {
			absolute++
		}
	}
	switch {
	case internal == 0:
		return ""
	case 2*absolute > internal:
		return URLFormAbsolute
	default:
		return URLFormRelative
	}
}

func absoluteHref(href string) bool {
	trimmed := strings.TrimSpace(href)
	if strings.HasPrefix(trimmed, "//") {
		return true
	}
	parsed, err := url.Parse(trimmed)
	return err == nil && parsed.Scheme != ""
}
