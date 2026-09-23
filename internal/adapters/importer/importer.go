package importer

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	extensionXLSX = ".xlsx"
	extensionCSV  = ".csv"
)

type ReadOptions = importmap.ReadOptions

type SheetInfo = importmap.SheetInfo

type Reader struct{}

func New() Reader {
	return Reader{}
}

func (Reader) Read(ctx context.Context, path string, opts ReadOptions) (importmap.Table, error) {
	return Read(ctx, path, opts)
}

func (Reader) Sheets(path string) ([]SheetInfo, error) {
	return Sheets(path)
}

func (Reader) Write(path string, table importmap.Table) error {
	return Write(path, table)
}

func Extension(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New(errors.Invalid, "the file path must not be empty").WithDetail("field", "path")
	}
	extension := strings.ToLower(filepath.Ext(path))
	if extension != extensionXLSX && extension != extensionCSV {
		return "", errors.New(errors.Invalid, "the file must be a .xlsx or a .csv").
			WithDetail("field", "path").WithDetail("extension", extension)
	}
	return extension, nil
}

func Sheets(path string) ([]SheetInfo, error) {
	extension, err := Extension(path)
	if err != nil {
		return nil, err
	}
	if extension == extensionCSV {
		return separatedSheet(path)
	}
	return workbookSheets(path)
}

func separatedSheet(path string) ([]SheetInfo, error) {
	counted := &collector{rows: make([][]string, 0)}
	if err := readSeparated(path, counted.add); err != nil {
		return nil, err
	}
	return []SheetInfo{{Name: "", Headers: counted.headers, Rows: len(counted.rows)}}, nil
}

func Read(ctx context.Context, path string, opts ReadOptions) (importmap.Table, error) {
	extension, err := Extension(path)
	if err != nil {
		return importmap.Table{}, err
	}
	if err = ctx.Err(); err != nil {
		return importmap.Table{}, errors.Wrap(err, errors.Cancelled, "read the import file")
	}

	if extension == extensionCSV {
		collected := newCollector(opts, "")
		if readErr := readSeparated(path, collected.add); readErr != nil {
			return importmap.Table{}, readErr
		}
		return collected.table(path)
	}
	return readSheets(path, opts)
}

type collector struct {
	headers []string
	rows    [][]string
	origins []importmap.Origin
	sheet   string
	maxRows int
	line    int
	letters bool
	headed  bool
}

func newCollector(opts ReadOptions, sheet string) *collector {
	return &collector{
		rows: make([][]string, 0), origins: make([]importmap.Origin, 0),
		sheet: sheet, maxRows: opts.MaxRows, letters: opts.Letters,
	}
}

func (c *collector) add(row []string) error {
	c.line++
	trimmed := trimRow(row)
	if blank(trimmed) {
		return nil
	}
	if !c.headed && !c.letters {
		c.headers, c.headed = trimmed, true
		return nil
	}
	if c.maxRows > 0 && len(c.rows) >= c.maxRows {
		return errors.New(errors.Invalid, "the import file carries more rows than the import.maxRows setting allows").
			WithDetail("maxRows", c.maxRows)
	}
	c.rows = append(c.rows, trimmed)
	c.origins = append(c.origins, importmap.Origin{Sheet: c.sheet, Row: c.line})
	if c.letters {
		c.headed = true
	}
	return nil
}

func (c *collector) table(path string) (importmap.Table, error) {
	if c.letters {
		return importmap.Table{Headers: letterHeaders(c.rows), Rows: c.rows, Origins: c.origins}, nil
	}
	if !c.headed {
		return importmap.Table{}, errors.New(errors.Invalid, "the import file has no header row").WithDetail("path", path)
	}
	return importmap.Table{Headers: c.headers, Rows: c.rows, Origins: c.origins}, nil
}

func letterHeaders(rows [][]string) []string {
	widest := 0
	for _, row := range rows {
		widest = max(widest, len(row))
	}
	out := make([]string, 0, widest)
	for column := 1; column <= widest; column++ {
		name, err := excelize.ColumnNumberToName(column)
		if err != nil {
			name = strconv.Itoa(column)
		}
		out = append(out, name)
	}
	return out
}

func openFailed(cause error, path string) error {
	if os.IsNotExist(cause) {
		return errors.New(errors.NotFound, "the import file does not exist").WithDetail("path", path).WithInternal(cause)
	}
	return errors.Wrap(cause, errors.Invalid, "read the import file")
}

func trimRow(row []string) []string {
	out := make([]string, len(row))
	for i, cell := range row {
		out[i] = strings.TrimSpace(strings.TrimPrefix(cell, "\ufeff"))
	}
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return out
}

func blank(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(strings.TrimPrefix(cell, "\ufeff")) != "" {
			return false
		}
	}
	return true
}
