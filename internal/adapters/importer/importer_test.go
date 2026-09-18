package importer_test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/davidmovas/postulator/internal/adapters/importer"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func write(t *testing.T, name, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

func sheet(t *testing.T, name string, rows [][]string) string {
	t.Helper()

	file := excelize.NewFile()
	t.Cleanup(func() {
		if err := file.Close(); err != nil {
			t.Errorf("close the workbook: %v", err)
		}
	})

	for i, row := range rows {
		cell, err := excelize.CoordinatesToCellName(1, i+1)
		if err != nil {
			t.Fatalf("cell name: %v", err)
		}
		cells := make([]any, 0, len(row))
		for _, value := range row {
			cells = append(cells, value)
		}
		if err = file.SetSheetRow(file.GetSheetName(0), cell, &cells); err != nil {
			t.Fatalf("set row %d: %v", i, err)
		}
	}

	path := filepath.Join(t.TempDir(), name)
	if err := file.SaveAs(path); err != nil {
		t.Fatalf("save the workbook: %v", err)
	}
	return path
}

func TestReadFindsTheHeaderAndTheRows(t *testing.T) {
	t.Parallel()

	rows := [][]string{{"path", "title", "keywords"}, {"/", "Home", "home,main"}, {"/services/", "Services", "services"}}

	cases := []struct {
		name string
		path func(*testing.T) string
	}{
		{
			name: "comma",
			path: func(t *testing.T) string {
				return write(t, "map.csv", "path,title,keywords\r\n/,Home,\"home,main\"\r\n/services/,Services,services\r\n")
			},
		},
		{
			name: "semicolon",
			path: func(t *testing.T) string {
				return write(t, "map.csv", "path;title;keywords\n/;Home;home,main\n/services/;Services;services\n")
			},
		},
		{
			name: "tab",
			path: func(t *testing.T) string {
				return write(t, "map.csv", "path\ttitle\tkeywords\n/\tHome\thome,main\n/services/\tServices\tservices\n")
			},
		},
		{
			name: "byte order mark and leading blank lines",
			path: func(t *testing.T) string {
				return write(t, "map.csv", "\ufeff\n\npath,title,keywords\n/,Home,\"home,main\"\n\n/services/,Services,services\n")
			},
		},
		{
			name: "workbook",
			path: func(t *testing.T) string {
				return sheet(t, "map.xlsx", [][]string{{"", "", ""}, {"path", "title", "keywords"}, {"/", "Home", "home,main"}, {"", "", ""}, {"/services/", "Services", "services"}})
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			table, err := importer.Read(t.Context(), tc.path(t), 0)
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			if !slices.Equal(table.Headers, rows[0]) {
				t.Fatalf("headers = %v, want %v", table.Headers, rows[0])
			}
			if len(table.Rows) != 2 {
				t.Fatalf("rows = %v, want two", table.Rows)
			}
			for i, want := range rows[1:] {
				if !slices.Equal(table.Rows[i], want) {
					t.Fatalf("row %d = %v, want %v", i, table.Rows[i], want)
				}
			}
		})
	}
}

func TestReadRefusesWhatItCannotParse(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		path func(*testing.T) string
		code errors.Code
	}{
		{name: "no path", path: func(*testing.T) string { return "" }, code: errors.Invalid},
		{
			name: "unknown extension",
			path: func(t *testing.T) string { return write(t, "map.json", "{}") },
			code: errors.Invalid,
		},
		{
			name: "missing file",
			path: func(t *testing.T) string { return filepath.Join(t.TempDir(), "absent.csv") },
			code: errors.NotFound,
		},
		{
			name: "empty sheet",
			path: func(t *testing.T) string { return write(t, "map.csv", "\n\n") },
			code: errors.Invalid,
		},
		{
			name: "not a workbook",
			path: func(t *testing.T) string { return write(t, "map.xlsx", "not a workbook") },
			code: errors.Invalid,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := importer.Read(t.Context(), tc.path(t), 0)
			if !errors.IsCode(err, tc.code) {
				t.Fatalf("Read = %v, want %s", err, tc.code)
			}
		})
	}
}

func TestReadStopsWhenTheContextIsDone(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := importer.Read(ctx, write(t, "map.csv", "path\n/\n"), 0); !errors.IsCode(err, errors.Cancelled) {
		t.Fatalf("Read = %v, want cancelled", err)
	}
}

func TestWriteProducesAWorkbookThatReadsBack(t *testing.T) {
	t.Parallel()

	table := importmap.Table{
		Headers: []string{"path", "title"},
		Rows:    [][]string{{"/", "Home"}, {"/services/", "Services"}},
	}
	path := filepath.Join(t.TempDir(), "export.xlsx")
	if err := importer.Write(path, table); err != nil {
		t.Fatalf("Write: %v", err)
	}

	back, err := importer.Read(t.Context(), path, 0)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !slices.Equal(back.Headers, table.Headers) {
		t.Fatalf("headers = %v", back.Headers)
	}
	for i, want := range table.Rows {
		if !slices.Equal(back.Rows[i], want) {
			t.Fatalf("row %d = %v, want %v", i, back.Rows[i], want)
		}
	}
}

func TestWriteRefusesWhatItCannotSave(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		path string
	}{
		{name: "no path", path: ""},
		{name: "unknown extension", path: filepath.Join(t.TempDir(), "export.csv")},
		{name: "missing directory", path: filepath.Join(t.TempDir(), "absent", "export.xlsx")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if err := importer.Write(tc.path, importmap.Table{Headers: []string{"path"}}); err == nil {
				t.Fatal("Write accepted a path it cannot save to")
			}
		})
	}
}

func TestReaderIsTheAdapterTheUseCasesHold(t *testing.T) {
	t.Parallel()

	reader := importer.New()
	path := filepath.Join(t.TempDir(), "export.xlsx")
	if err := reader.Write(path, importmap.Table{Headers: []string{"path"}, Rows: [][]string{{"/"}}}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	table, err := reader.Read(t.Context(), path, 0)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(table.Rows) != 1 || table.Rows[0][0] != "/" {
		t.Fatalf("rows = %v", table.Rows)
	}
}

func TestTheDelimiterIsSniffedOutsideQuotes(t *testing.T) {
	t.Parallel()

	path := write(t, "map.csv", "\"path;with;semicolons\",title\n\"/a;b/\",Home\n")
	table, err := importer.Read(t.Context(), path, 0)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !slices.Equal(table.Headers, []string{"path;with;semicolons", "title"}) {
		t.Fatalf("headers = %v", table.Headers)
	}
	if !slices.Equal(table.Rows[0], []string{"/a;b/", "Home"}) {
		t.Fatalf("row = %v", table.Rows[0])
	}
}

func TestReadKeepsRaggedRowsAndTrimsTrailingBlanks(t *testing.T) {
	t.Parallel()

	path := write(t, "map.csv", "path,title,keywords\n/,Home\n/services/,Services,,\n")
	table, err := importer.Read(t.Context(), path, 0)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !slices.Equal(table.Rows[0], []string{"/", "Home"}) {
		t.Fatalf("short row = %v", table.Rows[0])
	}
	if !slices.Equal(table.Rows[1], []string{"/services/", "Services"}) {
		t.Fatalf("padded row = %v", table.Rows[1])
	}
}

func TestReadStopsAtTheRowCap(t *testing.T) {
	t.Parallel()

	cases := []struct {
		path func(*testing.T) string
		name string
	}{
		{
			name: "separated values",
			path: func(t *testing.T) string { return write(t, "cap.csv", "path\n/a/\n/b/\n/c/\n") },
		},
		{
			name: "workbook",
			path: func(t *testing.T) string {
				return sheet(t, "cap.xlsx", [][]string{{"path"}, {"/a/"}, {"/b/"}, {"/c/"}})
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			path := tc.path(t)
			if _, err := importer.Read(t.Context(), path, 2); !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("Read past the cap = %v, want invalid", err)
			}

			table, err := importer.Read(t.Context(), path, 3)
			if err != nil || len(table.Rows) != 3 {
				t.Fatalf("Read at the cap = %+v, %v", table, err)
			}
		})
	}
}
