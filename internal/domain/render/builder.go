package render

import (
	"slices"
	"strings"

	"golang.org/x/net/html"
)

func (p *page) builder() string {
	scope := p.widest(p.root)
	shortcodes := p.body.Shortcodes()
	for i := range p.in.Builders {
		rule := &p.in.Builders[i]
		if p.builtBy(scope, rule) || prefixed(shortcodes, rule.ShortcodePrefixes) {
			return rule.Name
		}
	}
	return ""
}

func (p *page) builtBy(scope *html.Node, rule *Builder) bool {
	marks := func(node *html.Node) bool {
		for _, a := range node.Attr {
			if slices.Contains(rule.DataAttributes, a.Key) {
				return true
			}
			if a.Key == "class" && prefixed(strings.Fields(a.Val), rule.ClassPrefixes) {
				return true
			}
		}
		return false
	}
	return (scope.Type == html.ElementNode && marks(scope)) || firstUnder(scope, marks) != nil
}

func prefixed(values, prefixes []string) bool {
	for _, prefix := range prefixes {
		if prefix == "" {
			continue
		}
		for _, value := range values {
			if strings.HasPrefix(value, prefix) {
				return true
			}
		}
	}
	return false
}
