package render

import (
	"bytes"
	"encoding/json"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/net/html"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

type ldObject struct {
	id    string
	types []string
	props map[string]json.RawMessage
	raw   json.RawMessage
}

type ldCrumb struct {
	position float64
	name     string
	path     string
	located  bool
}

func (o *ldObject) is(name string) bool {
	for _, t := range o.types {
		if t == name || strings.HasSuffix(t, "/"+name) || strings.HasSuffix(t, ":"+name) {
			return true
		}
	}
	return false
}

func (p *page) readJSONLD() {
	for node := range elementsUnder(p.doc, false) {
		if !named(node, "script") || !linkedData(node) {
			continue
		}
		var text strings.Builder
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if child.Type == html.TextNode {
				text.WriteString(child.Data)
			}
		}
		p.objects = append(p.objects, decodeLD(json.RawMessage(text.String()))...)
	}
	for i := range p.objects {
		if !p.objects[i].is("BreadcrumbList") {
			continue
		}
		if crumbs := p.ldCrumbs(&p.objects[i]); len(crumbs) > 0 {
			p.ldTrails = append(p.ldTrails, pagemap.Trail{Source: pagemap.TrailJSONLD, Crumbs: crumbs})
		}
	}
	p.listings = p.ldListings()
}

func linkedData(node *html.Node) bool {
	value, _ := attr(node, "type")
	media, _, _ := strings.Cut(value, ";")
	return strings.EqualFold(strings.TrimSpace(media), "application/ld+json")
}

func decodeLD(raw json.RawMessage) []ldObject {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil
	}
	switch trimmed[0] {
	case '[':
		var items []json.RawMessage
		if json.Unmarshal(trimmed, &items) != nil {
			return nil
		}
		out := make([]ldObject, 0, len(items))
		for _, item := range items {
			out = append(out, decodeLD(item)...)
		}
		return out
	case '{':
		var props map[string]json.RawMessage
		if json.Unmarshal(trimmed, &props) != nil {
			return nil
		}
		out := make([]ldObject, 0, 1)
		if types := typesOf(props["@type"]); len(types) > 0 {
			out = append(out, ldObject{id: stringOf(props["@id"]), types: types, props: props, raw: trimmed})
		}
		if graph, ok := props["@graph"]; ok {
			out = append(out, decodeLD(graph)...)
		}
		return out
	default:
		return nil
	}
}

func typesOf(raw json.RawMessage) []string {
	if one := stringOf(raw); one != "" {
		return []string{one}
	}
	var many []string
	if json.Unmarshal(raw, &many) != nil {
		return nil
	}
	return slices.DeleteFunc(many, func(t string) bool { return strings.TrimSpace(t) == "" })
}

func stringOf(raw json.RawMessage) string {
	var value string
	if len(raw) == 0 || json.Unmarshal(raw, &value) != nil {
		return ""
	}
	return strings.TrimSpace(value)
}

func objectOf(raw json.RawMessage) (map[string]json.RawMessage, bool) {
	var props map[string]json.RawMessage
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' || json.Unmarshal(trimmed, &props) != nil {
		return nil, false
	}
	return props, true
}

func listOf(raw json.RawMessage) []json.RawMessage {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil
	}
	if trimmed[0] == '{' {
		return []json.RawMessage{trimmed}
	}
	var items []json.RawMessage
	if json.Unmarshal(trimmed, &items) != nil {
		return nil
	}
	return items
}

func positionOf(raw json.RawMessage) float64 {
	var number float64
	if json.Unmarshal(raw, &number) == nil {
		return number
	}
	if parsed, err := strconv.ParseFloat(stringOf(raw), 64); err == nil {
		return parsed
	}
	return math.Inf(1)
}

func itemURLs(entry map[string]json.RawMessage) []string {
	urls := make([]string, 0, 5)
	item := entry["item"]
	if one := stringOf(item); one != "" {
		urls = append(urls, one)
	}
	if props, ok := objectOf(item); ok {
		urls = append(urls, stringOf(props["@id"]), stringOf(props["url"]))
	}
	urls = append(urls, stringOf(entry["url"]))
	return slices.DeleteFunc(urls, func(u string) bool { return u == "" })
}

func (p *page) firstPath(urls []string) string {
	for _, u := range urls {
		if path := p.resolved(u); path != "" {
			return path
		}
	}
	return ""
}

func (p *page) ldCrumbs(list *ldObject) []pagemap.Crumb {
	entries := listOf(list.props["itemListElement"])
	crumbs := make([]ldCrumb, 0, len(entries))
	for _, raw := range entries {
		entry, ok := objectOf(raw)
		if !ok {
			continue
		}
		name := stringOf(entry["name"])
		if item, isObject := objectOf(entry["item"]); isObject {
			if itemName := stringOf(item["name"]); itemName != "" {
				name = itemName
			}
		}
		urls := itemURLs(entry)
		crumbs = append(crumbs, ldCrumb{
			position: positionOf(entry["position"]),
			name:     strings.TrimSpace(html.UnescapeString(name)),
			path:     p.firstPath(urls),
			located:  len(urls) > 0,
		})
	}
	sort.SliceStable(crumbs, func(a, b int) bool { return crumbs[a].position < crumbs[b].position })

	out := make([]pagemap.Crumb, 0, len(crumbs))
	for i, c := range crumbs {
		path := c.path
		if !c.located && i == len(crumbs)-1 {
			path = p.in.Document
		}
		out = append(out, pagemap.Crumb{Name: c.name, Path: path})
	}
	return out
}

func (p *page) ldListings() [][]string {
	byID := make(map[string]int)
	for i := range p.objects {
		if id := p.objects[i].id; id != "" {
			if _, taken := byID[id]; !taken {
				byID[id] = i
			}
		}
	}

	out := make([][]string, 0)
	listed := make(map[int]bool)
	add := func(props map[string]json.RawMessage) {
		if paths := p.itemPaths(props); len(paths) > 0 {
			out = append(out, paths)
		}
	}
	for i := range p.objects {
		if p.objects[i].is("ItemList") && !p.objects[i].is("BreadcrumbList") {
			listed[i] = true
			add(p.objects[i].props)
		}
	}
	for i := range p.objects {
		if !p.objects[i].is("CollectionPage") {
			continue
		}
		for _, raw := range listOf(p.objects[i].props["mainEntity"]) {
			entity, ok := objectOf(raw)
			if !ok {
				continue
			}
			if _, inline := entity["itemListElement"]; inline {
				add(entity)
				continue
			}
			if at, found := byID[stringOf(entity["@id"])]; found && !listed[at] && !p.objects[at].is("BreadcrumbList") {
				listed[at] = true
				add(p.objects[at].props)
			}
		}
	}
	return out
}

func (p *page) itemPaths(list map[string]json.RawMessage) []string {
	out := make([]string, 0)
	for _, raw := range listOf(list["itemListElement"]) {
		entry, ok := objectOf(raw)
		if !ok {
			continue
		}
		if path := p.firstPath(append(itemURLs(entry), stringOf(entry["@id"]))); path != "" && !slices.Contains(out, path) {
			out = append(out, path)
		}
	}
	return out
}

func (p *page) typed() map[string][]json.RawMessage {
	out := make(map[string][]json.RawMessage)
	for i := range p.objects {
		for _, t := range p.objects[i].types {
			out[t] = append(out[t], p.objects[i].raw)
		}
	}
	return out
}
