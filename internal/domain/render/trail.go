package render

import (
	"slices"
	"strings"

	"golang.org/x/net/html"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

const dividers = "/\\|>›»«‹→←·•-–—"

type breadcrumbNav struct {
	node   *html.Node
	signal string
}

func (p *page) readNavs() {
	for _, node := range p.elements(isNav) {
		label, _ := attr(node, "aria-label")
		itemType, _ := attr(node, "itemtype")
		switch {
		case strings.Contains(strings.ToLower(label), "breadcrumb"):
			p.navs = append(p.navs, breadcrumbNav{node: node, signal: SignalARIA})
		case strings.Contains(itemType, "BreadcrumbList"):
			p.navs = append(p.navs, breadcrumbNav{node: node, signal: SignalMicrodata})
		}
	}
}

func (p *page) navTrails() []pagemap.Trail {
	out := make([]pagemap.Trail, 0, len(p.navs))
	for _, nav := range p.navs {
		crumbs := p.listCrumbs(nav.node)
		if crumbs == nil {
			crumbs = p.inlineCrumbs(nav.node)
		}
		if len(crumbs) > 0 {
			out = append(out, pagemap.Trail{Source: pagemap.TrailNav, Crumbs: crumbs})
		}
	}
	return out
}

func listItems(list *html.Node) []*html.Node {
	out := make([]*html.Node, 0)
	for child := list.FirstChild; child != nil; child = child.NextSibling {
		if named(child, "li") {
			out = append(out, child)
		}
	}
	return out
}

func (p *page) listCrumbs(nav *html.Node) []pagemap.Crumb {
	list := firstUnder(nav, func(n *html.Node) bool { return named(n, "ol", "ul") && len(listItems(n)) >= 2 })
	if list == nil {
		return nil
	}
	crumbs := make([]pagemap.Crumb, 0)
	lastLinked := false
	for _, item := range listItems(list) {
		if link := firstUnder(item, isHrefAnchor); link != nil {
			crumbs = append(crumbs, pagemap.Crumb{Name: accessibleName(link), Path: p.pathOf(link)})
			lastLinked = true
			continue
		}
		name := lastPiece(content.TextOf(item))
		if name == "" {
			continue
		}
		path := ""
		if current(item) {
			path = p.in.Document
		}
		crumbs = append(crumbs, pagemap.Crumb{Name: name, Path: path})
		lastLinked = false
	}
	if len(crumbs) > 0 && !lastLinked {
		crumbs[len(crumbs)-1].Path = p.in.Document
	}
	return crumbs
}

func current(item *html.Node) bool {
	marked := func(n *html.Node) bool {
		value, ok := attr(n, "aria-current")
		return ok && value != "false"
	}
	return marked(item) || firstUnder(item, marked) != nil
}

func (p *page) inlineCrumbs(nav *html.Node) []pagemap.Crumb {
	crumbs := make([]pagemap.Crumb, 0)
	var trailing strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			switch {
			case child.Type == html.TextNode:
				trailing.WriteString(child.Data)
			case child.Type != html.ElementNode || slices.Contains(unseenElements, child.Data):
			case isHrefAnchor(child):
				crumbs = append(crumbs, pagemap.Crumb{Name: accessibleName(child), Path: p.pathOf(child)})
				trailing.Reset()
			default:
				trailing.WriteByte(' ')
				walk(child)
				trailing.WriteByte(' ')
			}
		}
	}
	walk(nav)
	if name := lastPiece(trailing.String()); name != "" {
		crumbs = append(crumbs, pagemap.Crumb{Name: name, Path: p.in.Document})
	}
	return crumbs
}

func lastPiece(text string) string {
	pieces := make([]string, 0)
	words := make([]string, 0)
	flush := func() {
		if piece := strings.Trim(strings.Join(words, " "), dividers); piece != "" {
			pieces = append(pieces, piece)
		}
		words = words[:0]
	}
	for _, field := range strings.Fields(text) {
		if strings.Trim(field, dividers) == "" {
			flush()
			continue
		}
		words = append(words, field)
	}
	flush()
	if len(pieces) == 0 {
		return ""
	}
	return pieces[len(pieces)-1]
}
