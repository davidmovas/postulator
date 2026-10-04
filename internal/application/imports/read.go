package imports

import (
	"strings"

	"github.com/davidmovas/postulator/internal/domain/category"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/keyword"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

type ownership int

const (
	ownUnset ownership = iota
	ownYes
	ownNo
)

type rowDraft struct {
	at         importmap.Origin
	path       string
	title      string
	h1         string
	metaTitle  string
	metaDesc   string
	wpType     string
	pageKind   string
	keywords   keyword.List
	notes      []pagemap.Note
	name       string
	own        ownership
	kind       string
	anchors    []string
	related    []string
	parent     string
	levels     []importmap.Level
	roots      []string
	categories categoryLevels
}

type categoryLevel struct {
	name string
	root bool
}

type categoryLevels []categoryLevel

func (l categoryLevels) named(root bool) []string {
	out := make([]string, 0, len(l))
	for _, level := range l {
		if level.root == root {
			out = append(out, level.name)
		}
	}
	return out
}

func (l categoryLevels) chain() []string {
	return l.named(false)
}

func (l categoryLevels) dropped() []string {
	return l.named(true)
}

func (r *rowDraft) sortLevels(roots rootSet) {
	for _, level := range r.levels {
		switch {
		case !level.Category:
			r.roots = append(r.roots, level.Name)
		case category.Key(level.Name) == "":
		default:
			r.categories = append(r.categories, categoryLevel{name: level.Name, root: roots.holds(level.Name)})
		}
	}
}

func (r *rowDraft) noteDropped(p *plan) {
	for _, name := range r.categories.dropped() {
		p.noteAt(r.at, "", CodeCategoryLevelIsRoot,
			"the category "+name+" is the name of a root entity, so it is not made a category; "+
				"the levels below it are filed under the level above it")
	}
}

func (r *rowDraft) named() string {
	switch {
	case r.name != "":
		return r.name
	case r.h1 != "":
		return r.h1
	case r.title != "":
		return r.title
	default:
		return titleFrom(r.path)
	}
}

func ownershipOf(raw string) (ownership, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "":
		return ownUnset, true
	case "yes", "y", "true", "1":
		return ownYes, true
	case "no", "n", "false", "0":
		return ownNo, true
	default:
		return ownUnset, false
	}
}

func rowKeywords(binding importmap.Binding, row []string, at importmap.Origin, p *plan) keyword.List {
	items := make([]keyword.Keyword, 0)
	for _, field := range []importmap.Field{importmap.FieldPrimaryKeyword, importmap.FieldKeywords} {
		list, unreadable := keyword.Parse(binding.Text(row, field))
		for _, fragment := range unreadable {
			p.noteAt(at, string(field), CodeBadVolume,
				"the search volume of "+fragment+" cannot be read, so the keyword was kept without one")
		}
		items = append(items, list...)
	}
	return keyword.New(items)
}

func readRows(binding importmap.Binding, table importmap.Table, roots rootSet, p *plan) []rowDraft {
	rows := make([]rowDraft, 0, len(table.Rows))
	walk := binding.Walk()
	for i := range table.Rows {
		raw := walk.Path(table.Rows[i])
		draft, kept := readRow(binding, table.Rows[i], raw, table.Origin(i), roots, p)
		if !kept {
			p.report.Skipped++
			continue
		}
		rows = append(rows, draft)
	}
	return rows
}

func readRow(binding importmap.Binding, row []string, raw string, at importmap.Origin, roots rootSet, p *plan) (rowDraft, bool) {
	if binding.Blank(row) && raw == "" {
		return rowDraft{}, false
	}
	draft := cellsOf(binding, row, at)
	draft.sortLevels(roots)
	if draft.name == "" && raw == "" && len(draft.roots) == 0 {
		p.noteAt(at, "", CodeNoTarget, "the row names neither a path, an entity nor a group")
		return rowDraft{}, false
	}
	draft.noteDropped(p)
	draft.own = rowOwnership(binding, row, at, p)
	draft.keywords = rowKeywords(binding, row, at, p)
	draft.path = rowPath(raw, at, p)
	return draft, true
}

func cellsOf(binding importmap.Binding, row []string, at importmap.Origin) rowDraft {
	return rowDraft{
		at:        at,
		title:     binding.Text(row, importmap.FieldTitle),
		h1:        binding.Text(row, importmap.FieldH1),
		metaTitle: binding.Text(row, importmap.FieldMetaTitle),
		metaDesc:  binding.Text(row, importmap.FieldMetaDescription),
		wpType:    binding.Text(row, importmap.FieldWPType),
		pageKind:  binding.Text(row, importmap.FieldPageKind),
		name:      binding.Text(row, importmap.FieldEntity),
		kind:      binding.Text(row, importmap.FieldEntityKind),
		anchors:   binding.List(row, importmap.FieldAnchors),
		related:   binding.List(row, importmap.FieldRelated),
		parent:    binding.Text(row, importmap.FieldParentEntity),
		levels:    binding.Levels(row),
		notes:     binding.Notes(row),
	}
}

func rowOwnership(binding importmap.Binding, row []string, at importmap.Origin, p *plan) ownership {
	flag := binding.Text(row, importmap.FieldOwnEntity)
	own, known := ownershipOf(flag)
	if !known {
		p.noteAt(at, string(importmap.FieldOwnEntity), CodeUnknownOwnEntity,
			"the row says neither yes nor no about being an entity, so it was read as one: "+flag)
	}
	return own
}

func rowPath(raw string, at importmap.Origin, p *plan) string {
	if raw == "" {
		return ""
	}
	normalized, err := pagemap.NormalizePath(raw)
	if err != nil {
		p.noteAt(at, string(importmap.FieldPath), CodeBadPath, "the path cannot be read: "+raw)
		return ""
	}
	return normalized
}
