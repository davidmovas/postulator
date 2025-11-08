package dto

import "github.com/davidmovas/postulator/internal/infra/importer"

type ImportResult struct {
	TotalRead    int      `json:"totalRead"`
	TotalAdded   int      `json:"totalAdded"`
	TotalSkipped int      `json:"totalSkipped"`
	Added        []string `json:"added,omitempty"`
	Skipped      []string `json:"skipped,omitempty"`
	Errors       []string `json:"errors,omitempty"`
}

func NewImportResult(entity *importer.ImportResult) *ImportResult {
	d := &ImportResult{}
	return d.FromEntity(entity)
}

func (d *ImportResult) FromEntity(entity *importer.ImportResult) *ImportResult {
	if entity == nil {
		return d
	}
	d.TotalRead = entity.TotalRead
	d.TotalAdded = entity.TotalAdded
	d.TotalSkipped = entity.TotalSkipped
	d.Added = entity.Added
	d.Skipped = entity.Skipped
	d.Errors = entity.Errors
	return d
}

type ImportTopicsRequest struct {
	FilePath string `json:"filePath"`
}

type ImportAndAssignRequest struct {
	FilePath string `json:"filePath"`
	SiteID   int64  `json:"siteId"`
}
