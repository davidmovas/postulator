package importer

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/entities"
)

// ImportResult contains statistics about the import operation
type ImportResult struct {
	TotalRead    int
	TotalAdded   int
	TotalSkipped int
	Added        []string
	Skipped      []string
	Errors       []string
}

type FileParser interface {
	Parse(filePath string) ([]string, error)
}

type IImportService interface {
	ImportTopics(ctx context.Context, filePath string) (*ImportResult, error)
	ImportAndAssignToSite(ctx context.Context, filePath string, siteID int64, categoryID int64, strategy entities.TopicStrategy) (*ImportResult, error)
}
