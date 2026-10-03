package imports

import (
	"strings"

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
	at        importmap.Origin
	path      string
	title     string
	h1        string
	metaTitle string
	metaDesc  string
	wpType    string
	pageKind  string
	keywords  keyword.List
	notes     []pagemap.Note
	name      string
	own       ownership
	kind      string
	anchors   []string
	related   []string
	parent    string
	levels    []importmap.Level
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

func readRows(binding importmap.Binding, table importmap.Table, p *plan) []rowDraft {
	rows := make([]rowDraft, 0, len(table.Rows))
	walk := binding.Walk()
	for i := range table.Rows {
		row, at := table.Rows[i], table.Origin(i)
		raw := walk.Path(row)
		if binding.Blank(row) && raw == "" {
			p.report.Skipped++
			continue
		}

		draft := rowDraft{
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
		if draft.name == "" && raw == "" && len(draft.levels) == 0 {
			p.noteAt(at, "", CodeNoTarget, "the row names neither a path, an entity nor a group")
			p.report.Skipped++
			continue
		}

		flag := binding.Text(row, importmap.FieldOwnEntity)
		own, known := ownershipOf(flag)
		if !known {
			p.noteAt(at, string(importmap.FieldOwnEntity), CodeUnknownOwnEntity,
				"the row says neither yes nor no about being an entity, so it was read as one: "+flag)
		}
		draft.own = own
		draft.keywords = rowKeywords(binding, row, at, p)

		if raw != "" {
			normalized, err := pagemap.NormalizePath(raw)
			if err != nil {
				p.noteAt(at, string(importmap.FieldPath), CodeBadPath, "the path cannot be read: "+raw)
			} else {
				draft.path = normalized
			}
		}
		rows = append(rows, draft)
	}
	return rows
}

func pagesOf(rows []rowDraft, p *plan) *drafts {
	sheet := newDrafts()
	for i := range rows {
		row := &rows[i]
		if row.path == "" {
			continue
		}
		draft, known := sheet.page(row.path, row.at.Row)
		if known {
			p.noteAt(row.at, string(importmap.FieldPath), CodeDuplicatePath,
				"the path repeats an earlier row and was merged: "+row.path)
		}
		draft.merge(row, i)
	}
	return sheet
}
