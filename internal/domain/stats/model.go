package stats

import (
	"context"
	"time"
)

type SiteStats struct {
	ID                   int64
	SiteID               int64
	Date                 time.Time
	ArticlesPublished    int
	ArticlesFailed       int
	TotalWords           int
	InternalLinksCreated int
	ExternalLinksCreated int
}

type Repository interface {
	IncrementSiteStats(ctx context.Context, siteID int64, date time.Time, field string, value int) error
	GetSiteStats(ctx context.Context, siteID int64, from, to time.Time) ([]*SiteStats, error)
	GetTotalSiteStats(ctx context.Context, siteID int64) (*SiteStats, error)
}

type Service interface {
	GetSiteStatistics(ctx context.Context, siteID int64, from, to time.Time) ([]*SiteStats, error)
	GetTotalStatistics(ctx context.Context, siteID int64) (*SiteStats, error)
	GetDashboardSummary(ctx context.Context) (*DashboardSummary, error)
}

type DashboardSummary struct {
	TotalSites         int
	TotalArticles      int
	TotalArticlesToday int
	TotalWordsToday    int
	ActiveJobs         int
	PendingValidations int
}
