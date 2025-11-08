package healthcheck

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
)

type Repository interface {
	SaveHistory(ctx context.Context, history *entities.HealthCheckHistory) error
	GetHistoryBySite(ctx context.Context, siteID int64, limit int) ([]*entities.HealthCheckHistory, error)
	GetHistoryBySitePeriod(ctx context.Context, siteID int64, from, to time.Time) ([]*entities.HealthCheckHistory, error)
	GetLastCheckBySite(ctx context.Context, siteID int64) (*entities.HealthCheckHistory, error)
}

type Service interface {
	CheckSiteHealth(ctx context.Context, site *entities.Site) (*entities.HealthCheckHistory, error)
	CheckSiteByID(ctx context.Context, siteID int64) (*entities.HealthCheckHistory, error)
	CheckAutoHealthSites(ctx context.Context) (unhealthy []*entities.Site, recovered []*entities.Site, err error)
	GetSiteHistory(ctx context.Context, siteID int64, limit int) ([]*entities.HealthCheckHistory, error)
	GetSiteHistoryByPeriod(ctx context.Context, siteID int64, from, to time.Time) ([]*entities.HealthCheckHistory, error)
}

type Notifier interface {
	NotifyUnhealthySites(ctx context.Context, sites []*entities.Site, withSound bool) error
	NotifyRecoveredSites(ctx context.Context, sites []*entities.Site, withSound bool) error
	ResetState(siteID int64)
}

type WindowVisibilityChecker func() bool

type Scheduler interface {
	Start(ctx context.Context) error
	Stop() error
	UpdateInterval(intervalMinutes int) error
	ApplySettings(ctx context.Context, enabled bool, intervalMinutes int) error
}
