package importmap_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/importmap"
)

func TestAWalkBuildsThePathFromTheColumnACellSitsIn(t *testing.T) {
	t.Parallel()

	mapping := importmap.Mapping{
		ID: "m1", SiteID: "s1", Name: "Indented",
		Options: importmap.Options{IndentColumns: []string{"C", "D", "E"}},
	}
	ready, err := importmap.NewMapping(mapping)
	if err != nil {
		t.Fatalf("NewMapping: %v", err)
	}
	bound, err := ready.Bind([]string{"A", "B", "C", "D", "E"})
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}

	rows := [][]string{
		{"", "somedomain.uk", "/components"},
		{"", "", "", "batteries"},
		{"", "", "", "", "/e-bike-range/"},
		{"", "", "", "motors"},
		{"", "", "/guides"},
		{"", "", "", "beginners"},
	}
	want := []string{
		"/components/",
		"/components/batteries/",
		"/components/batteries/e-bike-range/",
		"/components/motors/",
		"/guides/",
		"/guides/beginners/",
	}

	walk := bound.Walk()
	for i, row := range rows {
		if got := walk.Path(row); got != want[i] {
			t.Fatalf("row %d = %q, want %q", i, got, want[i])
		}
	}
}

func TestAWalkAnswersNothingForARowWithNoIndentedCell(t *testing.T) {
	t.Parallel()

	ready, err := importmap.NewMapping(importmap.Mapping{
		ID: "m1", SiteID: "s1", Name: "Indented",
		Options: importmap.Options{IndentColumns: []string{"B", "C"}},
	})
	if err != nil {
		t.Fatalf("NewMapping: %v", err)
	}
	bound, err := ready.Bind([]string{"A", "B", "C"})
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}

	walk := bound.Walk()
	if got := walk.Path([]string{"a note"}); got != "" {
		t.Fatalf("a row outside the indented columns = %q, want nothing", got)
	}
	if got := walk.Path([]string{"", "/shop"}); got != "/shop/" {
		t.Fatalf("Path = %q", got)
	}
}

func TestAWalkFallsBackToTheMappedPathColumn(t *testing.T) {
	t.Parallel()

	ready, err := importmap.NewMapping(importmap.Mapping{
		ID: "m1", SiteID: "s1", Name: "Plain",
		Columns: map[importmap.Field]string{importmap.FieldPath: "Page path"},
	})
	if err != nil {
		t.Fatalf("NewMapping: %v", err)
	}
	bound, err := ready.Bind([]string{"Page path"})
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}

	if got := bound.Walk().Path([]string{"/components/"}); got != "/components/" {
		t.Fatalf("Path = %q, want the mapped column", got)
	}
}

func TestIndentColumnsSatisfyTheMappingOnTheirOwn(t *testing.T) {
	t.Parallel()

	if _, err := importmap.NewMapping(importmap.Mapping{
		ID: "m1", SiteID: "s1", Name: "Indented",
		Options: importmap.Options{IndentColumns: []string{"B"}},
	}); err != nil {
		t.Fatalf("a mapping that builds its paths from indentation was refused: %v", err)
	}

	if _, err := importmap.NewMapping(importmap.Mapping{ID: "m1", SiteID: "s1", Name: "Empty"}); err == nil {
		t.Fatal("a mapping with neither a column nor an indentation was accepted")
	}
}

func TestBindRefusesAnIndentColumnTheFileDoesNotHold(t *testing.T) {
	t.Parallel()

	ready, err := importmap.NewMapping(importmap.Mapping{
		ID: "m1", SiteID: "s1", Name: "Indented",
		Options: importmap.Options{IndentColumns: []string{"B", "Z"}},
	})
	if err != nil {
		t.Fatalf("NewMapping: %v", err)
	}
	if _, err = ready.Bind([]string{"A", "B"}); err == nil {
		t.Fatal("Bind accepted an indent column that is not in the file")
	}
}
