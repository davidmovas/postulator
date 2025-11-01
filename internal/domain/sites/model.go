package sites

import (
	"context"
	"time"
)

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
	StatusError    Status = "error"
)

type HealthStatus string

const (
	HealthHealthy   HealthStatus = "healthy"
	HealthUnhealthy HealthStatus = "unhealthy"
	HealthUnknown   HealthStatus = "unknown"
)

type Site struct {
	ID              int64
	Name            string
	URL             string
	WPUsername      string
	WPPassword      string
	Status          Status
	LastHealthCheck *time.Time
	HealthStatus    HealthStatus
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Repository interface {
	Create(ctx context.Context, site *Site) error
	GetByID(ctx context.Context, id int64) (*Site, error)
	GetAll(ctx context.Context) ([]*Site, error)
	GetByStatus(ctx context.Context, status Status) ([]*Site, error)
	Update(ctx context.Context, site *Site) error
	Delete(ctx context.Context, id int64) error
	UpdateHealthStatus(ctx context.Context, id int64, status HealthStatus, checkedAt time.Time) error
}

type Service interface {
	CreateSite(ctx context.Context, site *Site) error
	GetSite(ctx context.Context, id int64) (*Site, error)
	GetSiteWithPassword(ctx context.Context, id int64) (*Site, error)
	ListSites(ctx context.Context) ([]*Site, error)
	UpdateSite(ctx context.Context, site *Site) error
	UpdateSitePassword(ctx context.Context, id int64, password string) error
	DeleteSite(ctx context.Context, id int64) error

	CheckHealth(ctx context.Context, siteID int64) error
	CheckAllHealth(ctx context.Context) error
}
