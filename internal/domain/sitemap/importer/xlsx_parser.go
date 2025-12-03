package importer

import (
	"bytes"
	"fmt"

	"github.com/xuri/excelize/v2"
)

// XLSXParser parses Excel files (.xlsx)
type XLSXParser struct{}

// NewXLSXParser creates a new XLSX parser
func NewXLSXParser() *XLSXParser {
	return &XLSXParser{}
}

// SupportedExtensions returns supported file extensions
func (p *XLSXParser) SupportedExtensions() []string {
	return []string{".xlsx", ".xls"}
}

// Parse reads XLSX content and returns import rows
// Expected format:
// - First row is header
// - Columns: path/url, title, keywords (comma-separated in the cell)
// - Uses the first sheet
func (p *XLSXParser) Parse(data []byte) (*ImportResult, error) {
	result := &ImportResult{
		Rows:   make([]ImportRow, 0),
		Errors: make([]ImportError, 0),
	}

	// Open Excel file from bytes
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer f.Close()

	// Get the first sheet name
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("Excel file has no sheets")
	}
	sheetName := sheets[0]

	// Get all rows
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to read sheet %s: %w", sheetName, err)
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("sheet %s is empty", sheetName)
	}

	// First row is header
	headers := rows[0]

	// Find column indices
	pathIdx := findColumnIndex(headers, "path", "url", "slug", "uri", "link")
	titleIdx := findColumnIndex(headers, "title", "name", "page", "page_title")
	keywordsIdx := findColumnIndex(headers, "keywords", "keyword", "keys", "tags")

	if pathIdx == -1 {
		return nil, fmt.Errorf("Excel must have a path/url column (tried: path, url, slug, uri, link)")
	}

	if titleIdx == -1 {
		return nil, fmt.Errorf("Excel must have a title column (tried: title, name, page, page_title)")
	}

	// Process data rows (skip header)
	for i := 1; i < len(rows); i++ {
		rowNum := i + 1 // 1-based for user display
		row := rows[i]

		// Skip empty rows
		if len(row) == 0 {
			continue
		}

		// Extract values
		path := ""
		if pathIdx < len(row) {
			path = row[pathIdx]
		}

		title := ""
		if titleIdx < len(row) {
			title = row[titleIdx]
		}

		var keywords []string
		if keywordsIdx != -1 && keywordsIdx < len(row) {
			keywords = parseKeywordsString(row[keywordsIdx])
		}

		// Skip rows with empty path AND empty title (completely empty data row)
		if path == "" && title == "" {
			continue
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
