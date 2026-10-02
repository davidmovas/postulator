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
		{
			name:   "a level column that is also a field",
			change: func(m *importmap.Mapping) { m.Options.LevelColumns = []string{"Title"} },
		},
		{
			name: "a note column that is also a level",
			change: func(m *importmap.Mapping) {
				m.Options.LevelColumns = []string{"Category"}
				m.Options.NoteColumns = []string{"category"}
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

func TestNewMappingTakesLevelColumnsInPlaceOfAPath(t *testing.T) {
	t.Parallel()

	ready, err := importmap.NewMapping(importmap.Mapping{
		ID: "m1", SiteID: "s1", Name: "groups",
		Columns: map[importmap.Field]string{importmap.FieldTitle: "Title"},
		Options: importmap.Options{LevelColumns: []string{" Category ", "Subcategory"}, NoteColumns: []string{"Notes "}},
	})
	if err != nil {
		t.Fatalf("NewMapping: %v", err)
	}
	if !slices.Equal(ready.Options.LevelColumns, []string{"Category", "Subcategory"}) ||
		!slices.Equal(ready.Options.NoteColumns, []string{"Notes"}) {
		t.Fatalf("options = %+v, want the columns trimmed", ready.Options)
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
			row:    []string{"Peptides", " BPC-157 ", "Liquid", "/bpc-157/liquid/", "Commercial", "Sold as a 10 ml vial"},
			levels: []string{"Peptides", "BPC-157", "Liquid"},
			notes:  []pagemap.Note{{Label: "Intent Owner", Text: "Commercial"}, {Label: "Notes", Text: "Sold as a 10 ml vial"}},
		},
		{
			name:   "a placeholder is an empty level",
			row:    []string{"Peptides", "BPC-157", "—", "/bpc-157/"},
			levels: []string{"Peptides", "BPC-157", ""},
			notes:  []pagemap.Note{},
		},
		{
			name:   "every placeholder the sheets carry",
			row:    []string{"-", "N/A", "none", "/about/", " ", ""},
			levels: []string{"", "", ""},
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
		t.Fatal("a row that names only a level is reported blank")
	}
}

func TestBindRejectsALevelOrNoteColumnTheFileDoesNotCarry(t *testing.T) {
	t.Parallel()

	for name, options := range map[string]importmap.Options{
		"a level": {LevelColumns: []string{"Category"}},
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
