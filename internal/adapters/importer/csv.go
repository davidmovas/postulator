package importer

import (
	"bytes"
	"encoding/csv"
	"strings"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

var delimiters = []rune{',', ';', '\t'}

func sniff(body []byte) rune {
	for line := range strings.SplitSeq(strings.ReplaceAll(string(body), "\r\n", "\n"), "\n") {
		if strings.TrimSpace(strings.TrimPrefix(line, "\ufeff")) == "" {
			continue
		}
		best, count := ',', 0
		for _, candidate := range delimiters {
			if found := countOutsideQuotes(line, candidate); found > count {
				best, count = candidate, found
			}
		}
		return best
	}
	return ','
}

func countOutsideQuotes(line string, delimiter rune) int {
	count, quoted := 0, false
	for _, r := range line {
		switch {
		case r == '"':
			quoted = !quoted
		case r == delimiter && !quoted:
			count++
		}
	}
	return count
}

func readSeparated(body []byte) ([][]string, error) {
	reader := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(body, []byte("\ufeff"))))
	reader.Comma = sniff(body)
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	rows, err := reader.ReadAll()
	if err != nil {
		return nil, errors.Wrap(err, errors.Invalid, "read the separated values file")
	}
	return rows, nil
}
