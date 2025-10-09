package dto

import (
	"Postulator/internal/infra/importer"
)

type ImportResult struct {
	TotalRead    int      `json:"totalRead"`
	TotalAdded   int      `json:"totalAdded"`
	TotalSkipped int      `json:"totalSkipped"`
	Added        []string `json:"added"`
	Skipped      []string `json:"skipped"`
	Errors       []string `json:"errors"`
}

func FromImportResult(r *importer.ImportResult) *ImportResult {
	if r == nil {
		return nil
	}
	return &ImportResult{
		TotalRead:    r.TotalRead,
		TotalAdded:   r.TotalAdded,
		TotalSkipped: r.TotalSkipped,
		Added:        append([]string(nil), r.Added...),
		Skipped:      append([]string(nil), r.Skipped...),
		Errors:       append([]string(nil), r.Errors...),
	}
}
