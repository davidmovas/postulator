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

func (Reader) Read(ctx context.Context, path string) (importmap.Table, error) {
	return Read(ctx, path)
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

func Read(ctx context.Context, path string) (importmap.Table, error) {
	extension, err := Extension(path)
	if err != nil {
		return importmap.Table{}, err
	}
	if err = ctx.Err(); err != nil {
		return importmap.Table{}, errors.Wrap(err, errors.Cancelled, "read the import file")
	}

	body, err := os.ReadFile(path)
	if err != nil {
		return importmap.Table{}, openFailed(err, path)
	}

	var rows [][]string
	if extension == extensionXLSX {
		rows, err = readWorkbook(body)
	} else {
		rows, err = readSeparated(body)
	}
	if err != nil {
		return importmap.Table{}, err
	}
	return table(rows, path)
}

func openFailed(cause error, path string) error {
	if os.IsNotExist(cause) {
		return errors.New(errors.NotFound, "the import file does not exist").WithDetail("path", path).WithInternal(cause)
	}
	return errors.Wrap(cause, errors.Invalid, "read the import file")
}

func table(rows [][]string, path string) (importmap.Table, error) {
	header := -1
	for i, row := range rows {
		if !blank(row) {
			header = i
			break
		}
	}
	if header < 0 {
		return importmap.Table{}, errors.New(errors.Invalid, "the import file has no header row").WithDetail("path", path)
	}

	out := importmap.Table{Headers: trimRow(rows[header]), Rows: make([][]string, 0, len(rows)-header)}
	for _, row := range rows[header+1:] {
		if blank(row) {
			continue
		}
		out.Rows = append(out.Rows, trimRow(row))
	}
	return out, nil
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
