package importmap_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/importmap"
)

func TestAutoDetectReadsTheAliasTable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		header string
		want   importmap.Field
	}{
		{name: "path", header: "Path", want: importmap.FieldPath},
		{name: "url", header: "URL", want: importmap.FieldPath},
		{name: "slug", header: " slug ", want: importmap.FieldPath},
		{name: "uri", header: "URI", want: importmap.FieldPath},
		{name: "link", header: "Link", want: importmap.FieldPath},
		{name: "page path", header: "Page path", want: importmap.FieldPath},
		{name: "page address", header: "Page address", want: importmap.FieldPath},
		{name: "title", header: "Title", want: importmap.FieldTitle},
		{name: "page title", header: "Page Title", want: importmap.FieldTitle},
		{name: "name", header: "Name", want: importmap.FieldTitle},
		{name: "h1", header: "H1", want: importmap.FieldH1},
		{name: "heading", header: "Heading 1", want: importmap.FieldH1},
		{name: "primary keyword", header: "Primary Keyword", want: importmap.FieldPrimaryKeyword},
		{name: "snake primary keyword", header: "primary_keyword", want: importmap.FieldPrimaryKeyword},
		{name: "main keyword", header: "Main keyword", want: importmap.FieldPrimaryKeyword},
		{name: "focus keyword", header: "focus keyword", want: importmap.FieldPrimaryKeyword},
		{name: "keywords", header: "Keywords", want: importmap.FieldKeywords},
		{name: "secondary keywords", header: "Secondary Keywords", want: importmap.FieldKeywords},
		{name: "tags", header: "tags", want: importmap.FieldKeywords},
		{name: "anchors", header: "Anchors", want: importmap.FieldAnchors},
		{name: "anchor texts", header: "Anchor Texts", want: importmap.FieldAnchors},
		{name: "entity", header: "Entity", want: importmap.FieldEntity},
		{name: "topic", header: "Topic", want: importmap.FieldEntity},
		{name: "cluster", header: "Cluster", want: importmap.FieldEntity},
		{name: "entity kind", header: "entity_kind", want: importmap.FieldEntityKind},
		{name: "entity type", header: "Entity Type", want: importmap.FieldEntityKind},
		{name: "parent", header: "Parent", want: importmap.FieldParentEntity},
		{name: "parent entity", header: "parent_entity", want: importmap.FieldParentEntity},
		{name: "pillar", header: "Pillar", want: importmap.FieldParentEntity},
		{name: "related", header: "Related", want: importmap.FieldRelated},
		{name: "siblings", header: "Siblings", want: importmap.FieldRelated},
		{name: "page type", header: "Page Type", want: importmap.FieldPageKind},
		{name: "template", header: "Template", want: importmap.FieldPageKind},
		{name: "kind", header: "kind", want: importmap.FieldPageKind},
		{name: "meta title", header: "Meta Title", want: importmap.FieldMetaTitle},
		{name: "seo title", header: "SEO title", want: importmap.FieldMetaTitle},
		{name: "meta description", header: "meta_description", want: importmap.FieldMetaDescription},
		{name: "description", header: "Description", want: importmap.FieldMetaDescription},
		{name: "post type", header: "Post Type", want: importmap.FieldWPType},
		{name: "wp type", header: "wp_type", want: importmap.FieldWPType},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mapping := importmap.AutoDetect([]string{tc.header})
			if got := mapping.Columns[tc.want]; got != tc.header {
				t.Fatalf("AutoDetect(%q).Columns[%s] = %q, want %q", tc.header, tc.want, got, tc.header)
			}
		})
	}
}

func TestAutoDetectIgnoresUnknownAndRepeatedHeaders(t *testing.T) {
	t.Parallel()

	mapping := importmap.AutoDetect([]string{"\ufeffPath", "Sessions", "url", ""})
	if got := mapping.Columns[importmap.FieldPath]; got != "\ufeffPath" {
		t.Fatalf("path column = %q, want the first match", got)
	}
	if len(mapping.Columns) != 1 {
		t.Fatalf("columns = %v, want only the path", mapping.Columns)
	}
	if !reflect.DeepEqual(mapping.Options, importmap.DefaultOptions()) {
		t.Fatalf("options = %+v, want the defaults", mapping.Options)
	}
}

func TestAutoDetectCarriesTheClientSample(t *testing.T) {
	t.Parallel()

	mapping := importmap.AutoDetect([]string{"path", "title", "keywords"})
	want := map[importmap.Field]string{
		importmap.FieldPath:     "path",
		importmap.FieldTitle:    "title",
		importmap.FieldKeywords: "keywords",
	}
	for field, column := range want {
		if mapping.Columns[field] != column {
			t.Fatalf("columns[%s] = %q, want %q", field, mapping.Columns[field], column)
		}
	}
}

func TestFieldsAreTheCanonicalColumnOrder(t *testing.T) {
	t.Parallel()

	fields := importmap.Fields()
	want := []importmap.Field{
		importmap.FieldPath, importmap.FieldTitle, importmap.FieldH1, importmap.FieldPrimaryKeyword,
		importmap.FieldKeywords, importmap.FieldAnchors, importmap.FieldEntity, importmap.FieldEntityKind,
		importmap.FieldParentEntity, importmap.FieldRelated, importmap.FieldPageKind, importmap.FieldMetaTitle,
		importmap.FieldMetaDescription, importmap.FieldWPType,
	}
	if !slices.Equal(fields, want) {
		t.Fatalf("Fields() = %v, want %v", fields, want)
	}
	for _, field := range fields {
		if !field.Valid() {
			t.Fatalf("%s is not valid", field)
		}
	}
	if importmap.Field("sessions").Valid() {
		t.Fatal("an unknown field reports itself valid")
	}
}
