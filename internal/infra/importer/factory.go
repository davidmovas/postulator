package importer

import (
	"path/filepath"
	"strings"

	"Postulator/pkg/errors"
)

func GetParser(filePath string) (FileParser, error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".txt":
		return NewTxtParser(), nil
	case ".csv":
		return NewCsvParser(), nil
	case ".xlsx":
		return NewXlsxParser(), nil
	case ".json":
		return NewJsonParser(), nil
	default:
		return nil, errors.Validation("unsupported file format: " + ext + ". Supported formats: .txt, .csv, .xlsx, .json")
	}
}
