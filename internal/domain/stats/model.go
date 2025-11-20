package stats

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
)

type Repository interface {
	IncrementSiteStats(ctx context.Context, siteID int64, date time.Time, field string, value int) error
	GetSiteStats(ctx context.Context, siteID int64, from, to time.Time) ([]*entities.SiteStats, error)
	GetTotalSiteStats(ctx context.Context, siteID int64) (*entities.SiteStats, error)
}

type Service interface {
	GetSiteStatistics(ctx context.Context, siteID int64, from, to time.Time) ([]*entities.SiteStats, error)
	GetTotalStatistics(ctx context.Context, siteID int64) (*entities.SiteStats, error)
	GetDashboardSummary(ctx context.Context) (*entities.DashboardSummary, error)
}

type Recorder interface {
	RecordArticlePublished(ctx context.Context, siteID int64, wordCount int) error
	RecordArticleFailed(ctx context.Context, siteID int64) error
	RecordLinksCreated(ctx context.Context, siteID int64, internalLinks, externalLinks int) error
}

type ExecutionStatsReader interface {
	GetPendingValidations(ctx context.Context) ([]*entities.Execution, error)
	ListExecutions(ctx context.Context, offset, limit int, siteID int64) ([]*entities.Execution, int, error)
}
