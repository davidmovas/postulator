package importmap

import (
	"slices"
	"strings"
	"unicode"
)

var aliases = map[Field][]string{
	FieldPath: {
		"path", "url", "slug", "uri", "link", "page url", "page path", "address", "page address",
		"recommended url layer", "recommended url", "url layer", "canonical url",
	},
	FieldTitle:           {"title", "name", "page title", "page name"},
	FieldH1:              {"h1", "heading", "heading 1", "h1 heading"},
	FieldPrimaryKeyword:  {"primary keyword", "main keyword", "focus keyword", "target keyword", "primary kw"},
	FieldKeywords:        {"keywords", "secondary keywords", "tags", "keyword", "secondary kw"},
	FieldAnchors:         {"anchors", "anchor texts", "anchor", "anchor text"},
	FieldEntity:          {"entity", "topic", "cluster", "entity name", "topic name"},
	FieldEntityKind:      {"entity kind", "entity level", "entity type", "topic type", "cluster type"},
	FieldParentEntity:    {"parent", "parent entity", "pillar", "parent topic", "parent cluster", "parent product entity", "parent product"},
	FieldRelated:         {"related", "siblings", "related entities", "related topics"},
	FieldPageKind:        {"page type", "template", "kind", "page kind", "page template"},
	FieldMetaTitle:       {"meta title", "seo title"},
	FieldMetaDescription: {"meta description", "seo description", "description"},
	FieldWPType:          {"post type", "wp type", "wordpress type", "content type"},
	FieldOwnEntity:       {"own entity", "is entity", "has entity"},
}

var levelAliases = []string{"root entity", "root"}

var noteAliases = []string{"notes", "note", "intent owner", "reason", "detected form variation"}

var aliasIndex, compactIndex = buildIndexes()

func buildIndexes() (spaced, squeezed map[string]Field) {
	spaced = make(map[string]Field)
	squeezed = make(map[string]Field)
	for _, field := range fields {
		for _, alias := range aliases[field] {
			spaced[alias] = field
			squeezed[compact(alias)] = field
		}
	}
	return spaced, squeezed
}

func Words(text string) string {
	return strings.Join(strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}), " ")
}

func asked(header string) bool {
	return strings.HasSuffix(strings.TrimSpace(header), "?")
}

func Detect(header string) (Field, bool) {
	normalized := Words(header)
	if normalized == "" || asked(header) {
		return "", false
	}
	if field, known := aliasIndex[normalized]; known {
		return field, true
	}
	field, known := compactIndex[compact(normalized)]
	return field, known
}

func compact(normalized string) string {
	return strings.ReplaceAll(normalized, " ", "")
}

func rootLevel(header string) bool {
	if asked(header) {
		return false
	}
	written := compact(Words(header))
	return slices.ContainsFunc(levelAliases, func(alias string) bool { return written == compact(alias) })
}

func rootColumns(columns []string) []string {
	out := make([]string, 0, len(columns))
	for _, column := range columns {
		if rootLevel(column) {
			out = append(out, column)
		}
	}
	return out
}

func noteOf(header string) bool {
	return !asked(header) && slices.Contains(noteAliases, Words(header))
}

func AutoDetect(headers []string) Mapping {
	columns := make(map[Field]string, len(headers))
	options := DefaultOptions()
	notes := make([]string, 0)
	for _, header := range headers {
		if field, known := Detect(header); known {
			if _, taken := columns[field]; !taken {
				columns[field] = header
			}
			continue
		}
		if rootLevel(header) {
			options.LevelColumns = append(options.LevelColumns, header)
			continue
		}
		if noteOf(header) {
			notes = append(notes, header)
		}
	}

	if len(notes) > 0 {
		options.NoteColumns = notes
	}
	return Mapping{Columns: columns, Options: options}
}
