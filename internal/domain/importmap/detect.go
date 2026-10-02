package importmap

import (
	"cmp"
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

var levelAliases = map[string]int{
	"root entity": 0, "root": 0, "root category": 0,
	"category": 1, "main category": 1,
	"subcategory": 2, "sub category": 2,
	"sub subcategory": 3,
}

var noteAliases = []string{"notes", "note", "intent owner", "reason", "detected form variation"}

var aliasIndex, compactIndex = buildIndexes()

func buildIndexes() (spaced, compact map[string]Field) {
	spaced = make(map[string]Field)
	compact = make(map[string]Field)
	for _, field := range fields {
		for _, alias := range aliases[field] {
			spaced[alias] = field
			compact[strings.ReplaceAll(alias, " ", "")] = field
		}
	}
	return spaced, compact
}

func normalizeHeader(header string) string {
	var out strings.Builder
	gap := false
	for _, r := range strings.ToLower(header) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if gap && out.Len() > 0 {
				out.WriteByte(' ')
			}
			gap = false
			out.WriteRune(r)
			continue
		}
		gap = true
	}
	return out.String()
}

func asked(header string) bool {
	return strings.HasSuffix(strings.TrimSpace(header), "?")
}

func Detect(header string) (Field, bool) {
	normalized := normalizeHeader(header)
	if normalized == "" || asked(header) {
		return "", false
	}
	if field, known := aliasIndex[normalized]; known {
		return field, true
	}
	field, known := compactIndex[strings.ReplaceAll(normalized, " ", "")]
	return field, known
}

func levelOf(header string) (int, bool) {
	if asked(header) {
		return 0, false
	}
	normalized := normalizeHeader(header)
	for alias, rank := range levelAliases {
		if normalized == alias || strings.ReplaceAll(normalized, " ", "") == strings.ReplaceAll(alias, " ", "") {
			return rank, true
		}
	}
	return 0, false
}

func noteOf(header string) bool {
	return !asked(header) && slices.Contains(noteAliases, normalizeHeader(header))
}

func AutoDetect(headers []string) Mapping {
	columns := make(map[Field]string, len(headers))
	type level struct {
		header string
		rank   int
	}
	levels := make([]level, 0)
	notes := make([]string, 0)
	for _, header := range headers {
		if field, known := Detect(header); known {
			if _, taken := columns[field]; !taken {
				columns[field] = header
			}
			continue
		}
		if rank, known := levelOf(header); known {
			levels = append(levels, level{header: header, rank: rank})
			continue
		}
		if noteOf(header) {
			notes = append(notes, header)
		}
	}

	slices.SortStableFunc(levels, func(a, b level) int { return cmp.Compare(a.rank, b.rank) })
	options := DefaultOptions()
	for _, held := range levels {
		options.LevelColumns = append(options.LevelColumns, held.header)
	}
	if len(notes) > 0 {
		options.NoteColumns = notes
	}
	return Mapping{Columns: columns, Options: options}
}
