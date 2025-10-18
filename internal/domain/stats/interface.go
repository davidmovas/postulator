package stats

import (
	"Postulator/internal/domain/entities"
	"context"
	"time"
)

type Repository interface {
	GetBySiteAndDate(ctx context.Context, siteID int64, date time.Time) (*entities.Statistics, error)
	GetBySiteAndDateRange(ctx context.Context, siteID int64, from, to time.Time) ([]*entities.Statistics, error)
	Increment(ctx context.Context, siteID int64, date time.Time, field string) error
}

type Service interface {
	GetSiteStats(ctx context.Context, siteID int64, from, to time.Time) ([]*entities.Statistics, error)
	GetTotalStats(ctx context.Context, siteID int64) (*entities.Statistics, error)
	GetAvailableTitlesCount(ctx context.Context, siteID int64) (map[int64]int, error) // topicID -> count
}
