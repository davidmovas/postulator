package importmap

import (
	"strings"
	"unicode"
)

var aliases = map[Field][]string{
	FieldPath:            {"path", "url", "slug", "uri", "link", "page url", "address"},
	FieldTitle:           {"title", "name", "page title", "page name"},
	FieldH1:              {"h1", "heading", "heading 1", "h1 heading"},
	FieldPrimaryKeyword:  {"primary keyword", "main keyword", "focus keyword", "target keyword", "primary kw"},
	FieldKeywords:        {"keywords", "secondary keywords", "tags", "keyword", "secondary kw"},
	FieldAnchors:         {"anchors", "anchor texts", "anchor", "anchor text"},
	FieldEntity:          {"entity", "topic", "cluster", "entity name", "topic name"},
	FieldEntityKind:      {"entity kind", "entity type", "topic type", "cluster type"},
	FieldParentEntity:    {"parent", "parent entity", "pillar", "parent topic", "parent cluster"},
	FieldRelated:         {"related", "siblings", "related entities", "related topics"},
	FieldPageKind:        {"page type", "template", "kind", "page kind", "page template"},
	FieldMetaTitle:       {"meta title", "seo title"},
	FieldMetaDescription: {"meta description", "seo description", "description"},
	FieldWPType:          {"post type", "wp type", "wordpress type", "content type"},
}

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

func Detect(header string) (Field, bool) {
	normalized := normalizeHeader(header)
	if normalized == "" {
		return "", false
	}
	if field, known := aliasIndex[normalized]; known {
		return field, true
	}
	field, known := compactIndex[strings.ReplaceAll(normalized, " ", "")]
	return field, known
}

func AutoDetect(headers []string) Mapping {
	columns := make(map[Field]string, len(headers))
	for _, header := range headers {
		field, known := Detect(header)
		if !known {
			continue
		}
		if _, taken := columns[field]; taken {
			continue
		}
		columns[field] = header
	}
	return Mapping{Columns: columns, Options: DefaultOptions()}
}
