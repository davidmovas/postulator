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
	FieldPageType:        {"entity kind", "entity level", "entity type", "topic type", "cluster type", "page type", "kind", "page kind"},
	FieldParentEntity:    {"parent", "parent entity", "pillar", "parent topic", "parent cluster", "parent product entity", "parent product"},
	FieldRelated:         {"related", "siblings", "related entities", "related topics"},
	FieldMetaTitle:       {"meta title", "seo title"},
	FieldMetaDescription: {"meta description", "seo description", "description"},
	FieldWPType:          {"post type", "wp type", "wordpress type", "content type"},
	FieldOwnEntity:       {"own entity", "is entity", "has entity"},
	FieldSKU:             {"sku"},
}

var weakAliases = map[Field][]string{
	FieldPageType: {"page type", "kind", "page kind"},
}

func weak(field Field, header string) bool {
	written := compact(Words(header))
	return slices.ContainsFunc(weakAliases[field], func(alias string) bool { return written == compact(alias) })
}

var (
	rootAliases  = []string{"root entity", "root"}
	trailAliases = []string{"category", "subcategory", "sub subcategory"}
)

var noteAliases = []string{
	"notes", "note", "intent owner", "reason", "detected form variation",
	"entity notes", "entity note", "entity definition", "entity description",
}

func DescribesEntity(header string) bool {
	return slices.Contains(strings.Fields(Words(header)), "entity")
}

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

func namedAs(header string, aliases []string) bool {
	if asked(header) {
		return false
	}
	written := compact(Words(header))
	return slices.ContainsFunc(aliases, func(alias string) bool { return written == compact(alias) })
}

func rootLevel(header string) bool {
	return namedAs(header, rootAliases)
}

func level(header string) bool {
	return rootLevel(header) || namedAs(header, trailAliases)
}

func columnsWhere(columns []string, keep func(string) bool) []string {
	out := make([]string, 0, len(columns))
	for _, column := range columns {
		if keep(column) {
			out = append(out, column)
		}
	}
	return out
}

func rootColumns(columns []string) []string {
	return columnsWhere(columns, rootLevel)
}

func levelColumns(columns []string) []string {
	return columnsWhere(columns, level)
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
			if held, taken := columns[field]; !taken || weak(field, held) && !weak(field, header) {
				columns[field] = header
			}
			continue
		}
		if level(header) {
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
