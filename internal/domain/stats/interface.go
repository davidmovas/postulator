package stats

import (
	"context"
	"time"
)

type Repository interface {
	GetBySiteAndDate(ctx context.Context, siteID int64, date time.Time) (*Statistics, error)
	GetBySiteAndDateRange(ctx context.Context, siteID int64, from, to time.Time) ([]*Statistics, error)
	Increment(ctx context.Context, siteID int64, date time.Time, field string) error
}

type Service interface {
	GetSiteStats(ctx context.Context, siteID int64, from, to time.Time) ([]*Statistics, error)
	GetTotalStats(ctx context.Context, siteID int64) (*Statistics, error)
	GetAvailableTitlesCount(ctx context.Context, siteID int64) (map[int64]int, error) // topicID -> count
}
