package importer

import (
	"context"
)

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

type Service interface {
	ImportTopics(ctx context.Context, filePath string) (*ImportResult, error)
	ImportAndAssignToSite(ctx context.Context, filePath string, siteID int64) (*ImportResult, error)
}
