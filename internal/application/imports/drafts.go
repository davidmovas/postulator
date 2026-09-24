package imports

import (
	"slices"
	"strings"
)

func key(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func union(into, more []string) []string {
	seen := make(map[string]struct{}, len(into)+len(more))
	out := make([]string, 0, len(into)+len(more))
	for _, values := range [][]string{into, more} {
		for _, value := range values {
			trimmed := strings.TrimSpace(value)
			if trimmed == "" {
				continue
			}
			if _, dup := seen[strings.ToLower(trimmed)]; dup {
				continue
			}
			seen[strings.ToLower(trimmed)] = struct{}{}
			out = append(out, trimmed)
		}
	}
	return out
}

func fill(current, next string) string {
	if strings.TrimSpace(current) != "" {
		return current
	}
	return strings.TrimSpace(next)
}

type entityDraft struct {
	name     string
	kind     string
	primary  string
	keywords []string
	anchors  []string
	parent   string
	related  []string
	row      int
}

type pageDraft struct {
	path      string
	title     string
	h1        string
	metaTitle string
	metaDesc  string
	wpType    string
	pageKind  string
	entity    string
	primary   string
	keywords  []string
	generated bool
	row       int
}

type drafts struct {
	entities map[string]*entityDraft
	pages    map[string]*pageDraft
	order    []string
	paths    []string
}

func newDrafts() *drafts {
	return &drafts{entities: make(map[string]*entityDraft), pages: make(map[string]*pageDraft)}
}

func (d *drafts) entity(name string, row int) *entityDraft {
	at := key(name)
	current, known := d.entities[at]
	if known {
		return current
	}
	current = &entityDraft{name: strings.TrimSpace(name), row: row}
	d.entities[at] = current
	d.order = append(d.order, at)
	return current
}

func (d *drafts) page(path string, row int) (draft *pageDraft, known bool) {
	current, known := d.pages[path]
	if known {
		return current, true
	}
	current = &pageDraft{path: path, row: row}
	d.pages[path] = current
	d.paths = append(d.paths, path)
	return current, false
}

func (d *drafts) sortedPaths() []string {
	out := slices.Clone(d.paths)
	slices.Sort(out)
	return out
}

func (e *entityDraft) merge(other entityDraft) {
	e.kind = fill(e.kind, other.kind)
	e.primary = fill(e.primary, other.primary)
	e.parent = fill(e.parent, other.parent)
	e.keywords = union(e.keywords, other.keywords)
	e.anchors = union(e.anchors, other.anchors)
	e.related = union(e.related, other.related)
}

func (p *pageDraft) merge(other pageDraft) {
	p.title = fill(p.title, other.title)
	p.h1 = fill(p.h1, other.h1)
	p.metaTitle = fill(p.metaTitle, other.metaTitle)
	p.metaDesc = fill(p.metaDesc, other.metaDesc)
	p.wpType = fill(p.wpType, other.wpType)
	p.pageKind = fill(p.pageKind, other.pageKind)
	p.entity = fill(p.entity, other.entity)
	p.primary = fill(p.primary, other.primary)
	p.keywords = union(p.keywords, other.keywords)
}
