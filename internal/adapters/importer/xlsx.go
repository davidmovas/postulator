package importer

import (
	"github.com/xuri/excelize/v2"

	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func readWorkbook(path string, add func([]string) error) (err error) {
	file, err := excelize.OpenFile(path)
	if err != nil {
		return openFailed(err, path)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = errors.Wrap(closeErr, errors.Internal, "close the workbook")
		}
	}()

	sheets := file.GetSheetList()
	if len(sheets) == 0 {
		return errors.New(errors.Invalid, "the workbook has no sheet")
	}

	rows, err := file.Rows(sheets[0])
	if err != nil {
		return errors.Wrap(err, errors.Invalid, "read the first sheet")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = errors.Wrap(closeErr, errors.Internal, "close the sheet iterator")
		}
	}()

	for rows.Next() {
		row, columnsErr := rows.Columns()
		if columnsErr != nil {
			return errors.Wrap(columnsErr, errors.Invalid, "read a workbook row")
		}
		if addErr := add(row); addErr != nil {
			return addErr
		}
	}
	if err = rows.Error(); err != nil {
		return errors.Wrap(err, errors.Invalid, "read the first sheet")
	}
	return nil
}

func Write(path string, table importmap.Table) (err error) {
	extension, err := Extension(path)
	if err != nil {
		return err
	}
	if extension != extensionXLSX {
		return errors.New(errors.Invalid, "an export is written as a .xlsx").WithDetail("field", "path")
	}

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
