package repository

import (
	"Postulator/internal/models"
	"context"
	"time"
)

type SiteRepository interface {
	GetSites(ctx context.Context, limit int, offset int) (*models.PaginationResult[*models.Site], error)
	GetSite(ctx context.Context, id int64) (*models.Site, error)
	CreateSite(ctx context.Context, site *models.Site) (*models.Site, error)
	UpdateSite(ctx context.Context, site *models.Site) (*models.Site, error)
	ActivateSite(ctx context.Context, id int64) error
	DeactivateSite(ctx context.Context, id int64) error
	SetCheckStatus(ctx context.Context, id int64, checkTime time.Time, status string) error
	DeleteSite(ctx context.Context, id int64) error
}
