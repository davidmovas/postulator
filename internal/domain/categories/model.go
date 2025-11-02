package categories

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
)

type Repository interface {
	Create(ctx context.Context, category *entities.Category) error
	GetByID(ctx context.Context, id int64) (*entities.Category, error)
	GetBySiteID(ctx context.Context, siteID int64) ([]*entities.Category, error)
	GetByWPCategoryID(ctx context.Context, siteID int64, wpCategoryID int) (*entities.Category, error)
	Update(ctx context.Context, category *entities.Category) error
	Delete(ctx context.Context, id int64) error
	DeleteBySiteID(ctx context.Context, siteID int64) error
	BulkUpsert(ctx context.Context, siteID int64, categories []*entities.Category) error
}

type StatisticsRepository interface {
	Increment(ctx context.Context, siteID, categoryID int64, date time.Time, articlesPublished, totalWords int) error
	GetByCategory(ctx context.Context, categoryID int64, from, to time.Time) ([]*entities.Statistics, error)
	GetBySite(ctx context.Context, siteID int64, from, to time.Time) ([]*entities.Statistics, error)
}

type Service interface {
	CreateCategory(ctx context.Context, category *entities.Category) error
	GetCategory(ctx context.Context, id int64) (*entities.Category, error)
	ListSiteCategories(ctx context.Context, siteID int64) ([]*entities.Category, error)
	UpdateCategory(ctx context.Context, category *entities.Category) error
	DeleteCategory(ctx context.Context, id int64) error

	SyncFromWordPress(ctx context.Context, siteID int64) error
	CreateInWordPress(ctx context.Context, category *entities.Category) error
	UpdateInWordPress(ctx context.Context, category *entities.Category) error

	GetStatistics(ctx context.Context, categoryID int64, from, to time.Time) ([]*entities.Statistics, error)
}
