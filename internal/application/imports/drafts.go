package imports

import (
	"slices"
	"strings"
	"unicode"

	"github.com/davidmovas/postulator/internal/domain/keyword"
)

func key(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func slugOf(name string) string {
	return strings.Join(strings.FieldsFunc(strings.ToLower(name), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}), "-")
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

type pageDraft struct {
	path      string
	title     string
	h1        string
	metaTitle string
	metaDesc  string
	wpType    string
	pageKind  string
	entity    string
	keywords  keyword.List
	own       ownership
	rows      []int
	unit      int
	generated bool
	row       int
}

func (p *pageDraft) technical() bool {
	return p.own == ownNo
}

type drafts struct {
	pages map[string]*pageDraft
	paths []string
}

func newDrafts() *drafts {
	return &drafts{pages: make(map[string]*pageDraft)}
}

func (d *drafts) page(path string, row int) (draft *pageDraft, known bool) {
	current, known := d.pages[path]
	if known {
		return current, true
	}
	current = &pageDraft{path: path, row: row, unit: -1}
	d.pages[path] = current
	d.paths = append(d.paths, path)
	return current, false
}

func (d *drafts) sortedPaths() []string {
	out := slices.Clone(d.paths)
	slices.Sort(out)
	return out
}

func (p *pageDraft) merge(row *rowDraft, at int) {
	p.title = fill(p.title, row.title)
	p.h1 = fill(p.h1, row.h1)
	p.metaTitle = fill(p.metaTitle, row.metaTitle)
	p.metaDesc = fill(p.metaDesc, row.metaDesc)
	p.wpType = fill(p.wpType, row.wpType)
	p.pageKind = fill(p.pageKind, row.pageKind)
	p.keywords = p.keywords.Merge(row.keywords)
	if p.own == ownUnset {
		p.own = row.own
	}
	p.rows = append(p.rows, at)
}
