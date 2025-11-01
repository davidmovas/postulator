package importer

import (
	"strings"

	"github.com/davidmovas/postulator/pkg/errors"

	"github.com/xuri/excelize/v2"
)

var _ FileParser = (*XlsxParser)(nil)

type XlsxParser struct{}

func NewXlsxParser() *XlsxParser {
	return &XlsxParser{}
}

func (p *XlsxParser) Parse(filePath string) ([]string, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, errors.Import("xlsx", err)
	}
	defer func() {
		_ = f.Close()
	}()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return []string{}, nil
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, errors.Import("xlsx", err)
	}

	if len(rows) == 0 {
		return []string{}, nil
	}

	var titles []string
	titleColumnIndex := 0

	if len(rows) > 0 && len(rows[0]) > 0 {
		for i, header := range rows[0] {
			if strings.EqualFold(strings.TrimSpace(header), "title") {
				titleColumnIndex = i
				// Skip header row
				rows = rows[1:]
				break
			}
		}
	}

	for _, row := range rows {
		if len(row) > titleColumnIndex {
			title := strings.TrimSpace(row[titleColumnIndex])
			if title != "" {
				titles = append(titles, title)
			}
		}
	}

	return titles, nil
}
