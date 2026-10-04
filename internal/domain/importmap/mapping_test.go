package importmap_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func mapping() importmap.Mapping {
	return importmap.Mapping{
		ID:     "m1",
		SiteID: "s1",
		Name:   "client sheet",
		Columns: map[importmap.Field]string{
			importmap.FieldPath:     "URL",
			importmap.FieldTitle:    "Title",
			importmap.FieldKeywords: "Keywords",
			importmap.FieldAnchors:  "Anchors",
			importmap.FieldRelated:  "Related",
		},
	}
}

func TestNewMappingFillsTheSeparatorDefaults(t *testing.T) {
	t.Parallel()

	ready, err := importmap.NewMapping(mapping())
	if err != nil {
		t.Fatalf("NewMapping: %v", err)
	}
	if !reflect.DeepEqual(ready.Options, importmap.DefaultOptions()) {
		t.Fatalf("options = %+v, want %+v", ready.Options, importmap.DefaultOptions())
	}
}

func TestNewMappingRejectsWhatCannotBeImported(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		change func(*importmap.Mapping)
	}{
		{name: "no id", change: func(m *importmap.Mapping) { m.ID = "" }},
		{name: "no site", change: func(m *importmap.Mapping) { m.SiteID = "" }},
		{name: "no name", change: func(m *importmap.Mapping) { m.Name = "  " }},
		{name: "no columns", change: func(m *importmap.Mapping) { m.Columns = nil }},
		{name: "unknown field", change: func(m *importmap.Mapping) { m.Columns["sessions"] = "Sessions" }},
		{name: "blank column", change: func(m *importmap.Mapping) { m.Columns[importmap.FieldTitle] = " " }},
		{
			name: "neither a path nor an entity",
			change: func(m *importmap.Mapping) {
				m.Columns = map[importmap.Field]string{importmap.FieldTitle: "Title"}
			},
		},
		{name: "a blank level column", change: func(m *importmap.Mapping) { m.Options.LevelColumns = []string{"Category", " "} }},
		{name: "a blank note column", change: func(m *importmap.Mapping) { m.Options.NoteColumns = []string{""} }},
		{name: "an unknown row type", change: func(m *importmap.Mapping) { m.Options.RowType = "variants" }},
		{
			name: "a level column that is also a field",
			change: func(m *importmap.Mapping) {
				m.Columns[importmap.FieldTitle] = "Root"
				m.Options.LevelColumns = []string{"root"}
			},
		},
		{
			name: "a note column that is also a level",
			change: func(m *importmap.Mapping) {
				m.Options.LevelColumns = []string{"Root Entity"}
				m.Options.NoteColumns = []string{"root entity"}
			},
		},
		{
			name: "category columns alone",
			change: func(m *importmap.Mapping) {
				m.Columns = nil
				m.Options.LevelColumns = []string{"Category", "Subcategory"}
			},
		},
		{
			name: "a title and a root category",
			change: func(m *importmap.Mapping) {
				m.Columns = map[importmap.Field]string{importmap.FieldTitle: "Title"}
				m.Options.LevelColumns = []string{"Root Category"}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			broken := mapping()
			tc.change(&broken)
			if _, err := importmap.NewMapping(broken); !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("NewMapping = %v, want an invalid error", err)
			}
		})
	}
}

func TestNewMappingTakesARootColumnInPlaceOfAPath(t *testing.T) {
	t.Parallel()

	ready, err := importmap.NewMapping(importmap.Mapping{
		ID: "m1", SiteID: "s1", Name: "groups",
		Columns: map[importmap.Field]string{importmap.FieldTitle: "Title"},
		Options: importmap.Options{LevelColumns: []string{" Root Entity ", "Category"}, NoteColumns: []string{"Notes ", "category"}},
	})
	if err != nil {
		t.Fatalf("NewMapping: %v", err)
	}
	if !slices.Equal(ready.Options.LevelColumns, []string{"Root Entity", "Category"}) ||
		!slices.Equal(ready.Options.NoteColumns, []string{"Notes", "category"}) {
		t.Fatalf("options = %+v, want the columns trimmed and kept as given", ready.Options)
	}

	only, err := importmap.NewMapping(importmap.Mapping{
		ID: "m2", SiteID: "s1", Name: "roots", Options: importmap.Options{LevelColumns: []string{"Root"}},
	})
	if err != nil {
		t.Fatalf("NewMapping of a root column alone: %v", err)
	}
	if !slices.Equal(only.Options.LevelColumns, []string{"Root"}) {
		t.Fatalf("levels = %v", only.Options.LevelColumns)
	}
}

func TestBindPrefersTheHeaderExactlyAsWritten(t *testing.T) {
	t.Parallel()

	m := importmap.Mapping{Columns: map[importmap.Field]string{
		importmap.FieldPath: "URL", importmap.FieldOwnEntity: "Entity?", importmap.FieldEntity: "entity",
	}}
	binding, err := m.Bind([]string{"Entity", "URL", "Entity?"})
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}
	row := []string{"Liquid", "/bpc-157/liquid/", "NO"}
	if got := binding.Text(row, importmap.FieldOwnEntity); got != "NO" {
		t.Fatalf("own entity = %q, want the cell of the column named Entity?", got)
	}
	if got := binding.Text(row, importmap.FieldEntity); got != "Liquid" {
		t.Fatalf("entity = %q, want the cell of the column named Entity", got)
	}
}

func TestBindReadsTheLevelsAndTheNotesOfARow(t *testing.T) {
	t.Parallel()

	m := importmap.Mapping{
		Columns: map[importmap.Field]string{importmap.FieldPath: "URL"},
		Options: importmap.Options{
			LevelColumns: []string{"Root Entity", "Category", "Subcategory"},
			NoteColumns:  []string{"Intent Owner", "Notes"},
		},
	}
	binding, err := m.Bind([]string{"Root Entity", "Category", "Subcategory", "URL", "Intent Owner", "Notes"})
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}

	cases := []struct {
		name   string
		row    []string
		levels []string
		notes  []pagemap.Note
	}{
		{
			name:   "every level and note filled",
			row:    []string{"Peptides", "BPC-157", "Liquid", "/bpc-157/liquid/", "Commercial", "Sold as a 10 ml vial"},
			levels: []string{"Peptides"},
			notes:  []pagemap.Note{{Label: "Intent Owner", Text: "Commercial"}, {Label: "Notes", Text: "Sold as a 10 ml vial"}},
		},
		{
			name:   "a level cell is trimmed",
			row:    []string{" Peptides ", "", "", "/bpc-157/"},
			levels: []string{"Peptides"},
			notes:  []pagemap.Note{},
		},
		{
			name:   "a placeholder is no level",
			row:    []string{"—", "BPC-157", "Liquid", "/bpc-157/"},
			levels: []string{},
			notes:  []pagemap.Note{},
		},
		{
			name:   "every placeholder the sheets carry",
			row:    []string{"-", "N/A", "none", "/about/", " ", ""},
			levels: []string{},
			notes:  []pagemap.Note{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := binding.Levels(tc.row); !slices.Equal(got, tc.levels) {
				t.Errorf("Levels = %q, want %q", got, tc.levels)
			}
			if got := binding.Notes(tc.row); !reflect.DeepEqual(got, tc.notes) {
				t.Errorf("Notes = %+v, want %+v", got, tc.notes)
			}
		})
	}
	if binding.Blank([]string{"Peptides", "", "", "", "", ""}) {
		t.Fatal("a row that names only a root is reported blank")
	}
	if !binding.Blank([]string{"", "BPC-157", "Liquid", "", "", ""}) {
		t.Fatal("a row that names only categories is not reported blank")
	}
}

func TestOnlyARootColumnIsALevel(t *testing.T) {
	t.Parallel()

	cases := []struct {
		header string
		root   bool
	}{
		{header: "Root Entity", root: true},
		{header: "root_entity", root: true},
		{header: "Root", root: true},
		{header: "Root?"},
		{header: "Root Category"},
		{header: "Category"},
		{header: "Main Category"},
		{header: "Subcategory"},
		{header: "Sub Category"},
		{header: "Sub Subcategory"},
		{header: "Brand"},
	}

	for _, tc := range cases {
		t.Run(tc.header, func(t *testing.T) {
			t.Parallel()

			m := importmap.Mapping{
				Columns: map[importmap.Field]string{importmap.FieldPath: "URL"},
				Options: importmap.Options{LevelColumns: []string{tc.header}},
			}
			headers := []string{"URL", tc.header}
			binding, err := m.Bind(headers)
			if err != nil {
				t.Fatalf("Bind: %v", err)
			}
			want, use := []string{}, importmap.UseIgnored
			if tc.root {
				want, use = []string{"Peptides"}, importmap.UseLevel
			}
			if got := binding.Levels([]string{"/a/", "Peptides"}); !slices.Equal(got, want) {
				t.Errorf("Levels = %q, want %q", got, want)
			}
			if got := m.Uses(headers)[1]; got.Use != use {
				t.Errorf("Uses = %+v, want %s", got, use)
			}
		})
	}
}

func TestBindPassesOverACategoryColumnTheFileDoesNotCarry(t *testing.T) {
	t.Parallel()

	m := importmap.Mapping{
		Columns: map[importmap.Field]string{importmap.FieldPath: "URL"},
		Options: importmap.Options{LevelColumns: []string{"Root Entity", "Category", "Subcategory"}},
	}
	binding, err := m.Bind([]string{"URL", "Root Entity"})
	if err != nil {
		t.Fatalf("Bind = %v, want the category columns passed over", err)
	}
	if got := binding.Levels([]string{"/a/", "Peptides"}); !slices.Equal(got, []string{"Peptides"}) {
		t.Fatalf("Levels = %q, want the root alone", got)
	}
}

func TestBindRejectsALevelOrNoteColumnTheFileDoesNotCarry(t *testing.T) {
	t.Parallel()

	for name, options := range map[string]importmap.Options{
		"a level": {LevelColumns: []string{"Root Entity"}},
		"a note":  {NoteColumns: []string{"Notes"}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			m := importmap.Mapping{Columns: map[importmap.Field]string{importmap.FieldPath: "URL"}, Options: options}
			if _, err := m.Bind([]string{"URL"}); !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("Bind = %v, want an invalid error", err)
			}
		})
	}
}

func TestUsesSaysWhatEveryColumnOfTheFileBecomes(t *testing.T) {
	t.Parallel()

	m := importmap.Mapping{
		Columns: map[importmap.Field]string{importmap.FieldPath: "url", importmap.FieldTitle: "Title"},
		Options: importmap.Options{
			LevelColumns: []string{"Root Entity", "Category", "Subcategory"}, NoteColumns: []string{"Notes"},
			IndentColumns: []string{"Outline"},
		},
	}
	got := m.Uses([]string{"URL", "Root Entity", "Category", "Subcategory", "Title", "Notes", "Entity?", "", "Entity ID", "Outline"})
	want := []importmap.ColumnUse{
		{Header: "URL", Use: importmap.UseField, Field: importmap.FieldPath},
		{Header: "Root Entity", Use: importmap.UseLevel},
		{Header: "Category", Use: importmap.UseIgnored},
		{Header: "Subcategory", Use: importmap.UseIgnored},
		{Header: "Title", Use: importmap.UseField, Field: importmap.FieldTitle},
		{Header: "Notes", Use: importmap.UseNote},
		{Header: "Entity?", Use: importmap.UseIgnored},
		{Header: "", Use: importmap.UseIgnored},
		{Header: "Entity ID", Use: importmap.UseIgnored},
		{Header: "Outline", Use: importmap.UseIndent},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Uses =\n%+v\nwant\n%+v", got, want)
	}
}

func TestBindFindsTheColumnsHoweverTheyAreWritten(t *testing.T) {
	t.Parallel()

	binding, err := mapping().Bind([]string{"\ufeffurl", "title", "keywords", "anchors", "related"})
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}

	row := []string{"/Services/Web/", "Web Services", "web, services", "web team|web crew", "Hosting, Domains"}
	if got := binding.Text(row, importmap.FieldTitle); got != "Web Services" {
		t.Fatalf("title = %q", got)
	}
	if got := binding.List(row, importmap.FieldKeywords); !slices.Equal(got, []string{"web", "services"}) {
		t.Fatalf("keywords = %v", got)
	}
	if got := binding.List(row, importmap.FieldAnchors); !slices.Equal(got, []string{"web team", "web crew"}) {
		t.Fatalf("anchors = %v", got)
	}
	if got := binding.List(row, importmap.FieldRelated); !slices.Equal(got, []string{"Hosting", "Domains"}) {
		t.Fatalf("related = %v", got)
	}
	if !binding.Has(importmap.FieldPath) || binding.Has(importmap.FieldH1) {
		t.Fatal("Has does not report the mapped fields")
	}
}

func TestBindRejectsAColumnTheFileDoesNotCarry(t *testing.T) {
	t.Parallel()

	if _, err := mapping().Bind([]string{"url", "title"}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Bind = %v, want an invalid error", err)
	}
}

func TestBindingReadsShortAndBlankRows(t *testing.T) {
	t.Parallel()

	binding, err := mapping().Bind([]string{"url", "title", "keywords", "anchors", "related"})
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}

	if got := binding.Text([]string{"/a/"}, importmap.FieldRelated); got != "" {
		t.Fatalf("a short row read %q", got)
	}
	if got := binding.List([]string{"/a/", "", " , "}, importmap.FieldKeywords); len(got) != 0 {
		t.Fatalf("blank keywords = %v", got)
	}
	if !binding.Blank([]string{"", "  ", ""}) {
		t.Fatal("a blank row is not reported blank")
	}
	if binding.Blank([]string{"/a/"}) {
		t.Fatal("a row with a path is reported blank")
	}
}

func TestPathStripsTheConfiguredPrefix(t *testing.T) {
	t.Parallel()

	m := mapping()
	m.Options = importmap.Options{PathPrefixStrip: "https://Shop.example.com"}
	binding, err := m.Bind([]string{"url", "title", "keywords", "anchors", "related"})
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}

	row := []string{"https://shop.example.com/catalog/", "", "", "", ""}
	if got := binding.Path(row); got != "/catalog/" {
		t.Fatalf("Path = %q, want /catalog/", got)
	}
}

func TestOptionsJoinAndSplitAreInverse(t *testing.T) {
	t.Parallel()

	options := importmap.DefaultOptions()
	cases := []struct {
		name   string
		field  importmap.Field
		values []string
	}{
		{name: "keywords", field: importmap.FieldKeywords, values: []string{"web", "services"}},
		{name: "anchors", field: importmap.FieldAnchors, values: []string{"web team", "web crew"}},
		{name: "related", field: importmap.FieldRelated, values: []string{"Hosting", "Domains"}},
		{name: "empty", field: importmap.FieldRelated, values: nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			back := options.Split(tc.field, options.Join(tc.field, tc.values))
			if len(tc.values) == 0 {
				if len(back) != 0 {
					t.Fatalf("Split = %v, want nothing", back)
				}
				return
			}
			if !slices.Equal(back, tc.values) {
				t.Fatalf("Split(Join(%v)) = %v", tc.values, back)
			}
		})
	}
}
