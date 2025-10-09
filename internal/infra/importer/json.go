package importer

import (
	"encoding/json"
	"os"
	"strings"

	"Postulator/pkg/errors"
)

var _ FileParser = (*JsonParser)(nil)

type JsonParser struct{}

func NewJsonParser() *JsonParser {
	return &JsonParser{}
}

func (p *JsonParser) Parse(filePath string) ([]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, errors.Import("json", err)
	}

	var titlesArray []string
	if err = json.Unmarshal(data, &titlesArray); err == nil {
		// Filter out empty strings
		var filtered []string
		for _, title := range titlesArray {
			title = strings.TrimSpace(title)
			if title != "" {
				filtered = append(filtered, title)
			}
		}
		return filtered, nil
	}

	var objectsArray []map[string]interface{}
	if err = json.Unmarshal(data, &objectsArray); err == nil {
		var titles []string
		for _, obj := range objectsArray {
			if titleVal, ok := obj["title"]; ok {
				var titleStr string
				if titleStr, ok = titleVal.(string); ok {
					titleStr = strings.TrimSpace(titleStr)
					if titleStr != "" {
						titles = append(titles, titleStr)
					}
				}
			}
		}
		return titles, nil
	}

	return nil, errors.Import("json", errors.Validation("invalid JSON format: expected array of strings or array of objects with 'title' field"))
}
