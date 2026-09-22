package importer

import (
	stderrors "errors"

	"github.com/xuri/excelize/v2"

	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func openWorkbook(path string) (*excelize.File, error) {
	file, err := excelize.OpenFile(path)
	if err != nil {
		return nil, openFailed(err, path)
	}
	if len(file.GetSheetList()) == 0 {
		if closeErr := file.Close(); closeErr != nil {
			return nil, errors.Wrap(closeErr, errors.Internal, "close the workbook")
		}
		return nil, errors.New(errors.Invalid, "the workbook has no sheet")
	}
	return file, nil
}

func workbookSheets(path string) (found []SheetInfo, err error) {
	file, err := openWorkbook(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = errors.Wrap(closeErr, errors.Internal, "close the workbook")
		}
	}()

	names := file.GetSheetList()
	found = make([]SheetInfo, 0, len(names))
	for _, name := range names {
		counted := newCollector(ReadOptions{}, name)
		if walkErr := walkSheet(file, name, counted.add); walkErr != nil {
			return nil, walkErr
		}
		found = append(found, SheetInfo{Name: name, Headers: counted.headers, Rows: len(counted.rows)})
	}
	return found, nil
}

func readSheets(path string, opts ReadOptions) (table importmap.Table, err error) {
	file, err := openWorkbook(path)
	if err != nil {
		return importmap.Table{}, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = errors.Wrap(closeErr, errors.Internal, "close the workbook")
		}
	}()

	wanted, err := wantedSheets(file.GetSheetList(), opts.Sheets)
	if err != nil {
		return importmap.Table{}, err
	}

	joined := importmap.Table{Rows: make([][]string, 0), Origins: make([]importmap.Origin, 0)}
	budget := opts.MaxRows
	for _, name := range wanted {
		each := opts
		each.MaxRows = budget
		collected := newCollector(each, name)
		if walkErr := walkSheet(file, name, collected.add); walkErr != nil {
			return importmap.Table{}, walkErr
		}
		part, partErr := collected.table(path)
		if partErr != nil {
			return importmap.Table{}, sheetRefusal(partErr, name)
		}
		if len(joined.Headers) == 0 {
			joined.Headers = part.Headers
		} else if !sameHeaders(joined.Headers, part.Headers) {
			return importmap.Table{}, errors.New(errors.Invalid,
				"the sheets do not carry the same columns, so they cannot be imported together").
				WithDetail("sheet", name).WithDetail("headers", part.Headers).
				WithDetail("expected", joined.Headers)
		}
		joined.Rows = append(joined.Rows, part.Rows...)
		joined.Origins = append(joined.Origins, part.Origins...)
		if budget > 0 {
			budget -= len(part.Rows)
		}
	}
	return joined, nil
}

func wantedSheets(present, asked []string) ([]string, error) {
	if len(asked) == 0 {
		return present[:1], nil
	}

	known := make(map[string]struct{}, len(present))
	for _, name := range present {
		known[name] = struct{}{}
	}
	for _, name := range asked {
		if _, ok := known[name]; !ok {
			return nil, errors.New(errors.Invalid, "the workbook has no sheet by that name").
				WithDetail("sheet", name).WithDetail("sheets", present)
		}
	}
	return asked, nil
}

func sameHeaders(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func sheetRefusal(cause error, name string) error {
	var kernel *errors.Error
	if stderrors.As(cause, &kernel) {
		return kernel.WithDetail("sheet", name)
	}
	return cause
}

func walkSheet(file *excelize.File, name string, add func([]string) error) (err error) {
	rows, err := file.Rows(name)
	if err != nil {
		return errors.Wrap(err, errors.Invalid, "read the sheet "+name)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = errors.Wrap(closeErr, errors.Internal, "close the sheet iterator")
		}
	}()

	for rows.Next() {
		row, columnsErr := rows.Columns()
		if columnsErr != nil {
			return errors.Wrap(columnsErr, errors.Invalid, "read a row of "+name)
		}
		if addErr := add(row); addErr != nil {
			return addErr
		}
	}
	if err = rows.Error(); err != nil {
		return errors.Wrap(err, errors.Invalid, "read the sheet "+name)
	}
	return nil
}

func Write(path string, table importmap.Table) error {
	extension, err := Extension(path)
	if err != nil {
		return err
	}
	if extension == extensionCSV {
		return writeSeparated(path, table)
	}
	return writeWorkbook(path, table)
}

func writeWorkbook(path string, table importmap.Table) (err error) {
	file := excelize.NewFile()
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = errors.Wrap(closeErr, errors.Internal, "close the workbook")
		}
	}()

	name := file.GetSheetName(0)
	for i, row := range append([][]string{table.Headers}, table.Rows...) {
		cell, cellErr := excelize.CoordinatesToCellName(1, i+1)
		if cellErr != nil {
			return errors.Wrap(cellErr, errors.Internal, "address a workbook cell")
		}
		cells := make([]any, 0, len(row))
		for _, value := range row {
			cells = append(cells, value)
		}
		if setErr := file.SetSheetRow(name, cell, &cells); setErr != nil {
			return errors.Wrap(setErr, errors.Internal, "write a workbook row")
		}
	}

	if err = file.SaveAs(path); err != nil {
		return errors.Wrap(err, errors.Invalid, "save the workbook")
	}
	return nil
}
