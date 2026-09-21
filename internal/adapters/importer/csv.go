package importer

import (
	"bufio"
	"encoding/csv"
	stderrors "errors"
	"io"
	"os"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const sniffWindow = 64 * 1024

const defaultDelimiter = ','

var delimiters = []rune{defaultDelimiter, ';', '\t'}

func sniff(head []byte) rune {
	for line := range strings.SplitSeq(strings.ReplaceAll(string(head), "\r\n", "\n"), "\n") {
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

func writeSeparated(path string, table importmap.Table) (err error) {
	file, err := os.Create(path)
	if err != nil {
		return errors.Wrap(err, errors.Invalid, "create the separated values file")
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = errors.Wrap(closeErr, errors.Internal, "close the separated values file")
		}
	}()

	writer := csv.NewWriter(file)
	writer.Comma = defaultDelimiter

	for _, row := range append([][]string{table.Headers}, table.Rows...) {
		if writeErr := writer.Write(row); writeErr != nil {
			return errors.Wrap(writeErr, errors.Internal, "write a separated values row")
		}
	}

	writer.Flush()
	return errors.Wrap(writer.Error(), errors.Internal, "flush the separated values file")
}

func readSeparated(path string, add func([]string) error) (err error) {
	file, err := os.Open(path)
	if err != nil {
		return openFailed(err, path)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = errors.Wrap(closeErr, errors.Internal, "close the separated values file")
		}
	}()

	buffered := bufio.NewReaderSize(file, sniffWindow)
	head, peekErr := buffered.Peek(sniffWindow)
	if peekErr != nil && !stderrors.Is(peekErr, io.EOF) && !stderrors.Is(peekErr, bufio.ErrBufferFull) {
		return errors.Wrap(peekErr, errors.Invalid, "read the separated values file")
	}

	reader := csv.NewReader(buffered)
	reader.Comma = sniff(head)
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	for {
		row, readErr := reader.Read()
		if stderrors.Is(readErr, io.EOF) {
			return nil
		}
		if readErr != nil {
			return errors.Wrap(readErr, errors.Invalid, "read the separated values file")
		}
		if addErr := add(row); addErr != nil {
			return addErr
		}
	}
}
