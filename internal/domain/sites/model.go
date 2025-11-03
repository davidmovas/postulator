package sites

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
)

type Repository interface {
	Create(ctx context.Context, site *entities.Site) error
	GetByID(ctx context.Context, id int64) (*entities.Site, error)
	GetAll(ctx context.Context) ([]*entities.Site, error)
	GetByStatus(ctx context.Context, status entities.JobStatus) ([]*entities.Site, error)
	Update(ctx context.Context, site *entities.Site) error
	Delete(ctx context.Context, id int64) error
	UpdateHealthStatus(ctx context.Context, id int64, status entities.HealthStatus, checkedAt time.Time) error
}

type Service interface {
	CreateSite(ctx context.Context, site *entities.Site) error
	GetSite(ctx context.Context, id int64) (*entities.Site, error)
	GetSiteWithPassword(ctx context.Context, id int64) (*entities.Site, error)
	ListSites(ctx context.Context) ([]*entities.Site, error)
	UpdateSite(ctx context.Context, site *entities.Site) error
	UpdateSitePassword(ctx context.Context, id int64, password string) error
	DeleteSite(ctx context.Context, id int64) error

	CheckHealth(ctx context.Context, siteID int64) error
	CheckAllHealth(ctx context.Context) error
}
