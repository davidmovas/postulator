package site

import (
	"Postulator/internal/domain/entities"
	"context"
)

type ISiteRepository interface {
	Create(ctx context.Context, site *entities.Site) error
	GetByID(ctx context.Context, id int64) (*entities.Site, error)
	GetAll(ctx context.Context) ([]*entities.Site, error)
	Update(ctx context.Context, site *entities.Site) error
	Delete(ctx context.Context, id int64) error
	UpdateHealthStatus(ctx context.Context, id int64, status entities.HealthStatus) error
}

type ICategoryRepository interface {
	Create(ctx context.Context, siteId int64, category *entities.Category) error
	GetBySiteID(ctx context.Context, siteID int64) ([]*entities.Category, error)
	DeleteBySiteID(ctx context.Context, siteID int64) error
}

type IService interface {
	CreateSite(ctx context.Context, site *entities.Site) error
	GetSite(ctx context.Context, id int64) (*entities.Site, error)
	// GetSiteWithPassword returns a site with decrypted WP password for internal use only
	GetSiteWithPassword(ctx context.Context, id int64) (*entities.Site, error)
	ListSites(ctx context.Context) ([]*entities.Site, error)
	UpdateSite(ctx context.Context, site *entities.Site) error
	// UpdateSitePassword securely updates the site's WordPress password
	UpdateSitePassword(ctx context.Context, id int64, password string) error
	DeleteSite(ctx context.Context, id int64) error
	CheckHealth(ctx context.Context, siteID int64) error
	SyncCategories(ctx context.Context, siteID int64) error
	GetSiteCategories(ctx context.Context, siteID int64) ([]*entities.Category, error)
}
