package importer

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
)

// CSVParser parses CSV files
type CSVParser struct{}

// NewCSVParser creates a new CSV parser
func NewCSVParser() *CSVParser {
	return &CSVParser{}
}

// SupportedExtensions returns supported file extensions
func (p *CSVParser) SupportedExtensions() []string {
	return []string{".csv"}
}

// Parse reads CSV content and returns import rows
// Expected format:
// - First row is header
// - Columns: path/url, title, keywords (comma-separated in the field)
// - Column names are flexible (path, url, slug for path column, etc.)
func (p *CSVParser) Parse(data []byte) (*ImportResult, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	// Be flexible with field count
	reader.FieldsPerRecord = -1

	result := &ImportResult{
		Rows:   make([]ImportRow, 0),
		Errors: make([]ImportError, 0),
	}

	// Read header row
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Find column indices
	pathIdx := findColumnIndex(headers, "path", "url", "slug", "uri", "link")
	titleIdx := findColumnIndex(headers, "title", "name", "page", "page_title")
	keywordsIdx := findColumnIndex(headers, "keywords", "keyword", "keys", "tags")

	if pathIdx == -1 {
		return nil, fmt.Errorf("CSV must have a path/url column (tried: path, url, slug, uri, link)")
	}

	if titleIdx == -1 {
		return nil, fmt.Errorf("CSV must have a title column (tried: title, name, page, page_title)")
	}

	// Read data rows
	rowNum := 1 // 1-based, header is row 1
	for {
		rowNum++
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Errors = append(result.Errors, ImportError{
				Row:     rowNum,
				Message: fmt.Sprintf("failed to parse row: %v", err),
			})
			continue
		}

		// Extract values
		path := ""
		if pathIdx < len(record) {
			path = record[pathIdx]
		}

		title := ""
		if titleIdx < len(record) {
			title = record[titleIdx]
		}

		var keywords []string
		if keywordsIdx != -1 && keywordsIdx < len(record) {
			keywords = parseKeywordsString(record[keywordsIdx])
		}

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

	return result, nil
}
