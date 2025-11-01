package importer

import (
	"bufio"
	"os"
	"strings"

	"github.com/davidmovas/postulator/pkg/errors"
)

var _ FileParser = (*TxtParser)(nil)

type TxtParser struct{}

func NewTxtParser() *TxtParser {
	return &TxtParser{}
}

func (p *TxtParser) Parse(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, errors.Import("txt", err)
	}
	defer func() {
		_ = file.Close()
	}()

	var titles []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			titles = append(titles, line)
		}
	}

	if err = scanner.Err(); err != nil {
		return nil, errors.Import("txt", err)
	}

	return titles, nil
}
