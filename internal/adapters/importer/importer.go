package importer

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	extensionXLSX = ".xlsx"
	extensionCSV  = ".csv"
)

type Reader struct{}

func New() Reader {
	return Reader{}
}

func (Reader) Read(ctx context.Context, path string, maxRows int) (importmap.Table, error) {
	return Read(ctx, path, maxRows)
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

func Read(ctx context.Context, path string, maxRows int) (importmap.Table, error) {
	extension, err := Extension(path)
	if err != nil {
		return importmap.Table{}, err
	}
	if err = ctx.Err(); err != nil {
		return importmap.Table{}, errors.Wrap(err, errors.Cancelled, "read the import file")
	}

	collected := &collector{maxRows: maxRows, rows: make([][]string, 0)}
	if extension == extensionXLSX {
		err = readWorkbook(path, collected.add)
	} else {
		err = readSeparated(path, collected.add)
	}
	if err != nil {
		return importmap.Table{}, err
	}
	if !collected.headed {
		return importmap.Table{}, errors.New(errors.Invalid, "the import file has no header row").WithDetail("path", path)
	}
	return importmap.Table{Headers: collected.headers, Rows: collected.rows}, nil
}

type collector struct {
	headers []string
	rows    [][]string
	maxRows int
	headed  bool
}

func (c *collector) add(row []string) error {
	trimmed := trimRow(row)
	if blank(trimmed) {
		return nil
	}
	if !c.headed {
		c.headers, c.headed = trimmed, true
		return nil
	}
	if c.maxRows > 0 && len(c.rows) >= c.maxRows {
		return errors.New(errors.Invalid, "the import file carries more rows than the import.maxRows setting allows").
			WithDetail("maxRows", c.maxRows)
	}
	c.rows = append(c.rows, trimmed)
	return nil
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
