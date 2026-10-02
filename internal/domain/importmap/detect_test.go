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
		{name: "recommended url layer", header: "Recommended URL Layer", want: importmap.FieldPath},
		{name: "canonical url", header: "Canonical URL", want: importmap.FieldPath},
		{name: "entity level", header: "Entity Level", want: importmap.FieldEntityKind},
		{name: "parent product entity", header: "Parent Product Entity", want: importmap.FieldParentEntity},
		{name: "own entity", header: "Own entity", want: importmap.FieldOwnEntity},
		{name: "is entity", header: "is_entity", want: importmap.FieldOwnEntity},
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

func TestAQuestionIsNeverDetected(t *testing.T) {
	t.Parallel()

	for _, header := range []string{"Entity?", "Entity ?", "Is it an entity?", "Category?"} {
		if field, known := importmap.Detect(header); known {
			t.Errorf("Detect(%q) = %s, want nothing", header, field)
		}
	}
	mapping := importmap.AutoDetect([]string{"URL", "Entity?", "Category?"})
	if len(mapping.Columns) != 1 || len(mapping.Options.LevelColumns) != 0 || len(mapping.Options.NoteColumns) != 0 {
		t.Fatalf("mapping = %+v, want the path alone", mapping)
	}
}

func TestAutoDetectReadsTheClientSheets(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		headers []string
		columns map[importmap.Field]string
		levels  []string
		notes   []string
	}{
		{
			name:    "the root, category and subcategory sheet",
			headers: []string{"Root Entity", "Category", "Subcategory", "Recommended URL Layer", "Title", "H1", "Keywords"},
			columns: map[importmap.Field]string{
				importmap.FieldPath: "Recommended URL Layer", importmap.FieldTitle: "Title", importmap.FieldH1: "H1",
				importmap.FieldKeywords: "Keywords",
			},
			levels: []string{"Root Entity", "Category", "Subcategory"},
		},
		{
			name:    "the category and subcategory sheet",
			headers: []string{"Category", "Subcategory", "URL", "Title", "H1", "Keywords"},
			columns: map[importmap.Field]string{
				importmap.FieldPath: "URL", importmap.FieldTitle: "Title", importmap.FieldH1: "H1", importmap.FieldKeywords: "Keywords",
			},
			levels: []string{"Category", "Subcategory"},
		},
		{
			name: "the wide sheet an assistant wrote",
			headers: []string{
				"Entity ID", "Primary Entity", "Entity Level", "Entity Type", "Entity Name", "Canonical URL", "Parent Entity",
				"Category", "Subcategory", "Title", "H1", "Intent Owner", "Page Template", "Notes",
			},
			columns: map[importmap.Field]string{
				importmap.FieldEntityKind: "Entity Level", importmap.FieldEntity: "Entity Name", importmap.FieldPath: "Canonical URL",
				importmap.FieldParentEntity: "Parent Entity", importmap.FieldTitle: "Title", importmap.FieldH1: "H1",
				importmap.FieldPageKind: "Page Template",
			},
			levels: []string{"Category", "Subcategory"},
			notes:  []string{"Intent Owner", "Notes"},
		},
		{
			name:    "the variation sheet",
			headers: []string{"Parent Product Entity", "URL", "Detected Form / Variation", "Entity?", "Reason"},
			columns: map[importmap.Field]string{importmap.FieldParentEntity: "Parent Product Entity", importmap.FieldPath: "URL"},
			notes:   []string{"Detected Form / Variation", "Reason"},
		},
		{
			name:    "levels are ordered from the outermost whatever the order of the columns",
			headers: []string{"Subcategory", "URL", "Category", "Root"},
			columns: map[importmap.Field]string{importmap.FieldPath: "URL"},
			levels:  []string{"Root", "Category", "Subcategory"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mapping := importmap.AutoDetect(tc.headers)
			if !reflect.DeepEqual(mapping.Columns, tc.columns) {
				t.Errorf("columns = %v, want %v", mapping.Columns, tc.columns)
			}
			if !slices.Equal(mapping.Options.LevelColumns, tc.levels) {
				t.Errorf("levels = %v, want %v", mapping.Options.LevelColumns, tc.levels)
			}
			if !slices.Equal(mapping.Options.NoteColumns, tc.notes) {
				t.Errorf("notes = %v, want %v", mapping.Options.NoteColumns, tc.notes)
			}
		})
	}
}

func TestFieldsAreTheCanonicalColumnOrder(t *testing.T) {
	t.Parallel()

	fields := importmap.Fields()
	want := []importmap.Field{
		importmap.FieldPath, importmap.FieldTitle, importmap.FieldH1, importmap.FieldPrimaryKeyword,
		importmap.FieldKeywords, importmap.FieldAnchors, importmap.FieldEntity, importmap.FieldEntityKind,
		importmap.FieldParentEntity, importmap.FieldRelated, importmap.FieldPageKind, importmap.FieldMetaTitle,
		importmap.FieldMetaDescription, importmap.FieldWPType, importmap.FieldOwnEntity,
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
