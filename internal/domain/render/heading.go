package render

import "golang.org/x/net/html"

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
