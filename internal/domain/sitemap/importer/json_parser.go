package importer

import (
	"encoding/json"
	"fmt"
	"strings"
)

// JSONParser parses JSON files
type JSONParser struct{}

// NewJSONParser creates a new JSON parser
func NewJSONParser() *JSONParser {
	return &JSONParser{}
}

// SupportedExtensions returns supported file extensions
func (p *JSONParser) SupportedExtensions() []string {
	return []string{".json"}
}

// jsonRow represents a single entry in JSON import
// Supports multiple field name variations
type jsonRow struct {
	Path     string   `json:"path"`
	URL      string   `json:"url"`
	Slug     string   `json:"slug"`
	URI      string   `json:"uri"`
	Title    string   `json:"title"`
	Name     string   `json:"name"`
	Keywords []string `json:"keywords"`
	Keyword  string   `json:"keyword"` // Single keyword as string
	Tags     []string `json:"tags"`
}

// getPath returns the path value from any of the possible fields
func (r *jsonRow) getPath() string {
	if r.Path != "" {
		return r.Path
	}
	if r.URL != "" {
		return r.URL
	}
	if r.Slug != "" {
		return r.Slug
	}
	return r.URI
}

// getTitle returns the title value from any of the possible fields
func (r *jsonRow) getTitle() string {
	if r.Title != "" {
		return r.Title
	}
	return r.Name
}

// getKeywords returns combined keywords from all possible fields
func (r *jsonRow) getKeywords() []string {
	var result []string

	result = append(result, r.Keywords...)
	result = append(result, r.Tags...)

	// Handle single keyword field (comma-separated)
	if r.Keyword != "" {
		parts := strings.Split(r.Keyword, ",")
		for _, part := range parts {
			kw := strings.TrimSpace(part)
			if kw != "" {
				result = append(result, kw)
			}
		}
	}

	return result
}

// Parse reads JSON content and returns import rows
// Supports two formats:
// 1. Array of objects: [{"path": "/about", "title": "About", "keywords": ["key1", "key2"]}]
// 2. Object with "nodes" or "pages" array: {"nodes": [...]} or {"pages": [...]}
func (p *JSONParser) Parse(data []byte) (*ImportResult, error) {
	result := &ImportResult{
		Rows:   make([]ImportRow, 0),
		Errors: make([]ImportError, 0),
	}

	// Try to parse as array first
	var rows []jsonRow
	if err := json.Unmarshal(data, &rows); err == nil {
		return p.processRows(rows, result), nil
	}

	// Try to parse as object with nodes/pages array
	var wrapper struct {
		Nodes []jsonRow `json:"nodes"`
		Pages []jsonRow `json:"pages"`
		Items []jsonRow `json:"items"`
		Data  []jsonRow `json:"data"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("invalid JSON format: %w", err)
	}

	// Use whichever array has data
	if len(wrapper.Nodes) > 0 {
		rows = wrapper.Nodes
	} else if len(wrapper.Pages) > 0 {
		rows = wrapper.Pages
	} else if len(wrapper.Items) > 0 {
		rows = wrapper.Items
	} else if len(wrapper.Data) > 0 {
		rows = wrapper.Data
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("no data found in JSON (expected array or object with nodes/pages/items/data)")
	}

	return p.processRows(rows, result), nil
}

func (p *JSONParser) processRows(rows []jsonRow, result *ImportResult) *ImportResult {
	for i, row := range rows {
		rowNum := i + 1

		path := row.getPath()
		title := row.getTitle()
		keywords := row.getKeywords()

		// Validate required fields
		if path == "" {
			result.Errors = append(result.Errors, ImportError{
				Row:     rowNum,
				Column:  "path",
				Message: "path is required",
			})
			continue
		}

		if title == "" {
			result.Errors = append(result.Errors, ImportError{
				Row:     rowNum,
				Column:  "title",
				Message: "title is required",
			})
			continue
		}

		result.Rows = append(result.Rows, ImportRow{
			Path:     path,
			Title:    title,
			Keywords: keywords,
		})
	}

	return result
}
