package importer

import (
	"bytes"

	"github.com/xuri/excelize/v2"

	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func readWorkbook(body []byte) (rows [][]string, err error) {
	file, err := excelize.OpenReader(bytes.NewReader(body))
	if err != nil {
		return nil, errors.Wrap(err, errors.Invalid, "read the workbook")
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = errors.Wrap(closeErr, errors.Internal, "close the workbook")
		}
	}()

	sheets := file.GetSheetList()
	if len(sheets) == 0 {
		return nil, errors.New(errors.Invalid, "the workbook has no sheet")
	}

	rows, err = file.GetRows(sheets[0])
	if err != nil {
		return nil, errors.Wrap(err, errors.Invalid, "read the first sheet")
	}
	return rows, nil
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
