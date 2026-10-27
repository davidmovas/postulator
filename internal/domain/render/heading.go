package render

import (
	"strings"

	"golang.org/x/net/html"

	"github.com/davidmovas/postulator/internal/domain/content"
)

type H1Source string

const (
	H1Content H1Source = "content"
	H1Theme   H1Source = "theme"
	H1None    H1Source = "none"
	H1Unknown H1Source = ""
)

func (s H1Source) Valid() bool {
	return s == H1Content || s == H1Theme || s == H1None || s == H1Unknown
}

func VisibleH1(record *Record, contentH1 string) (string, H1Source) {
	if record != nil && record.H1Source != H1Unknown {
		return record.H1, record.H1Source
	}
	if text := strings.TrimSpace(contentH1); text != "" {
		return text, H1Content
	}
	return "", H1None
}

func (p *page) visibleH1() (string, H1Source) {
	own := ownH1s(p.in.Filtered, p.in.Body.Serialize())
	for _, node := range p.elements(func(n *html.Node) bool { return named(n, "h1") }) {
		if p.outsideThePage(node) {
			continue
		}
		text := headingText(node)
		if text == "" {
			continue
		}
		if p.inside(node, p.root) && own[text] {
			return text, H1Content
		}
		return text, H1Theme
	}
	return "", H1None
}

func headingText(node *html.Node) string {
	return strings.Join(strings.Fields(content.TextOf(node)), " ")
}

func ownH1s(fragments ...string) map[string]bool {
	out := make(map[string]bool)
	for _, fragment := range fragments {
		doc, err := content.Parse(fragment)
		if err != nil {
			continue
		}
		for _, heading := range doc.Headings() {
			if heading.Data == "h1" {
				out[headingText(heading)] = true
			}
		}
	}
	return out
}

func (p *page) outsideThePage(node *html.Node) bool {
	for at := node.Parent; at != nil; at = at.Parent {
		switch {
		case isNav(at), isAside(at), hasRole(at, "banner"), hasRole(at, "contentinfo"):
			return true
		case named(at, "header", "footer") && !sectioned(at):
			return true
		}
	}
	return false
}

func sectioned(node *html.Node) bool {
	for at := node.Parent; at != nil; at = at.Parent {
		if named(at, "article", "section") || isMain(at) {
			return true
		}
	}
	return false
}

type ThemeH1 string

const (
	ThemeH1Printed ThemeH1 = "printed"
	ThemeH1Absent  ThemeH1 = "absent"
	ThemeH1Unknown ThemeH1 = ""
)

func (h ThemeH1) Valid() bool {
	return h == ThemeH1Printed || h == ThemeH1Absent || h == ThemeH1Unknown
}

func (p *page) themeH1(headings []Heading) ThemeH1 {
	if !p.located {
		return ThemeH1Unknown
	}
	for _, node := range p.elements(func(n *html.Node) bool { return named(n, "h1") }) {
		if !p.inside(node, p.root) {
			return ThemeH1Printed
		}
	}
	for _, heading := range headings {
		if heading.Level == 1 && heading.Generated {
			return ThemeH1Printed
		}
	}
	return ThemeH1Absent
}

func OwesH1(own *Record, sameType []Record) bool {
	if own != nil {
		return own.ThemeH1 == ThemeH1Absent
	}
	absent := false
	for i := range sameType {
		switch sameType[i].ThemeH1 {
		case ThemeH1Printed:
			return false
		case ThemeH1Absent:
			absent = true
		}
	}
	return absent
}
