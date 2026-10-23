package imports

import (
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/keyword"
	"github.com/davidmovas/postulator/internal/domain/note"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

type ownership int

const (
	ownUnset ownership = iota
	ownYes
	ownNo
)

type rowDraft struct {
	at           importmap.Origin
	key          string
	path         string
	url          string
	title        string
	h1           string
	sku          string
	metaTitle    string
	metaDesc     string
	wpType       string
	pageType     string
	keywords     keyword.List
	notes        []note.Note
	name         string
	own          ownership
	anchors      []string
	related      []string
	parent       string
	levels       []string
	trail        []string
	expected     []string
	match        pagemap.Match
	unreadable   bool
	wantsProduct bool
	modeProduct  bool
}

func (r *rowDraft) sheetPath() string {
	if r.url != "" {
		return r.url
	}
	return r.path
}

func (r *rowDraft) names() bool {
	return r.name != "" || r.title != "" || r.h1 != "" || r.sku != "" || len(r.levels) > 0
}

func (r *rowDraft) asked(product bool) pagemap.Row {
	return pagemap.Row{Key: r.key, Path: r.url, Entity: r.name, Parent: r.parent, Title: r.title, H1: r.h1, SKU: r.sku, Product: product}
}

func (r *rowDraft) trailBelow(slug string) []string {
	if len(r.trail) == 0 {
		return nil
	}
	last := groupNode{name: r.trail[len(r.trail)-1]}
	if last.namedBy(r) || sameSlug(slug, last.name) {
		return slices.Clone(r.trail[:len(r.trail)-1])
	}
	return slices.Clone(r.trail)
}

func (r *rowDraft) cardNotes() []note.Note {
	if r.path == "" {
		return r.notes
	}
	out := make([]note.Note, 0, len(r.notes))
	for _, n := range r.notes {
		if importmap.DescribesEntity(n.Label) {
			out = append(out, n)
		}
	}
	return out
}

func (r *rowDraft) pageNotes() []note.Note {
	out := make([]note.Note, 0, len(r.notes))
	for _, n := range r.notes {
		if !importmap.DescribesEntity(n.Label) {
			out = append(out, n)
		}
	}
	return out
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
		return titleFrom(r.sheetPath())
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

func readRows(binding importmap.Binding, table importmap.Table, p *plan) []rowDraft {
	rows := make([]rowDraft, 0, len(table.Rows))
	walk := binding.Walk()
	for i := range table.Rows {
		raw := walk.Path(table.Rows[i])
		draft, kept := readRow(binding, table.Rows[i], raw, table.Origin(i), p)
		if !kept {
			p.report.Skipped++
			continue
		}
		rows = append(rows, draft)
	}
	return rows
}

func readRow(binding importmap.Binding, row []string, raw string, at importmap.Origin, p *plan) (rowDraft, bool) {
	if binding.Blank(row) && raw == "" {
		return rowDraft{}, false
	}
	draft := cellsOf(binding, row, at)
	if raw == "" && !draft.names() {
		noTarget(at, p)
		return rowDraft{}, false
	}
	draft.own = rowOwnership(binding, row, at, p)
	draft.keywords = rowKeywords(binding, row, at, p)
	draft.url, draft.unreadable = rowURL(raw, at, p)
	draft.key = pagemap.RowKey(p.sheetAt(at), draft.asked(false))
	p.sawRow(p.sheetAt(at), draft.key)
	return draft, true
}

func noTarget(at importmap.Origin, p *plan) {
	p.noteAt(at, "", CodeNoTarget, "the row names no page, no entity and no group")
}

func cellsOf(binding importmap.Binding, row []string, at importmap.Origin) rowDraft {
	return rowDraft{
		at:        at,
		title:     binding.Text(row, importmap.FieldTitle),
		h1:        binding.Text(row, importmap.FieldH1),
		sku:       binding.Text(row, importmap.FieldSKU),
		metaTitle: binding.Text(row, importmap.FieldMetaTitle),
		metaDesc:  binding.Text(row, importmap.FieldMetaDescription),
		wpType:    binding.Text(row, importmap.FieldWPType),
		pageType:  binding.Text(row, importmap.FieldPageType),
		name:      binding.Text(row, importmap.FieldEntity),
		anchors:   binding.List(row, importmap.FieldAnchors),
		related:   binding.List(row, importmap.FieldRelated),
		parent:    binding.Text(row, importmap.FieldParentEntity),
		levels:    binding.Levels(row),
		trail:     binding.Trail(row),
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

func rowURL(raw string, at importmap.Origin, p *plan) (url string, unreadable bool) {
	if raw == "" {
		return "", false
	}
	normalized, err := pagemap.NormalizePath(raw)
	if err != nil {
		p.noteAt(at, string(importmap.FieldPath), CodeBadPath, "the path cannot be read: "+raw)
		return "", true
	}
	return normalized, false
}
