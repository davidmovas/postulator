package importer

import (
	"encoding/csv"
	"os"
	"strings"

	"github.com/davidmovas/postulator/pkg/errors"
)

var _ FileParser = (*CsvParser)(nil)

type CsvParser struct{}

func NewCsvParser() *CsvParser {
	return &CsvParser{}
}

func (p *CsvParser) Parse(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, errors.Import("csv", err)
	}
	defer func() {
		_ = file.Close()
	}()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, errors.Import("csv", err)
	}

	if len(records) == 0 {
		return []string{}, nil
	}

	var titles []string
	titleColumnIndex := 0

	if len(records) > 0 {
		for i, header := range records[0] {
			if strings.EqualFold(strings.TrimSpace(header), "title") {
				titleColumnIndex = i
				// Skip header row
				records = records[1:]
				break
			}
		}
	}

	for _, record := range records {
		if len(record) > titleColumnIndex {
			title := strings.TrimSpace(record[titleColumnIndex])
			if title != "" {
				titles = append(titles, title)
			}
		}
	}

	return titles, nil
}
