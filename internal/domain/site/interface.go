package site

import "context"

type IRepository interface {
	Create(ctx context.Context, site *Site) error
	GetByID(ctx context.Context, id int64) (*Site, error)
	GetAll(ctx context.Context) ([]*Site, error)
	Update(ctx context.Context, site *Site) error
	Delete(ctx context.Context, id int64) error
	UpdateHealthStatus(ctx context.Context, id int64, status HealthStatus) error
}

type ICategoryRepository interface {
	Create(ctx context.Context, category *Category) error
	GetBySiteID(ctx context.Context, siteID int64) ([]*Category, error)
	DeleteBySiteID(ctx context.Context, siteID int64) error
}

type IService interface {
	CreateSite(ctx context.Context, site *Site) error
	GetSite(ctx context.Context, id int64) (*Site, error)
	ListSites(ctx context.Context) ([]*Site, error)
	UpdateSite(ctx context.Context, site *Site) error
	DeleteSite(ctx context.Context, id int64) error
	CheckHealth(ctx context.Context, siteID int64) error
	SyncCategories(ctx context.Context, siteID int64) error
}
