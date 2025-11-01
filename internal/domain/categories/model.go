package categories

import (
	"context"
	"time"
)

type Category struct {
	ID           int64
	SiteID       int64
	WPCategoryID int
	Name         string
	Slug         *string
	Description  *string
	ParentID     *int64
	Count        int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Statistics struct {
	CategoryID        int64
	Date              time.Time
	ArticlesPublished int
	TotalWords        int
}

type Repository interface {
	Create(ctx context.Context, category *Category) error
	GetByID(ctx context.Context, id int64) (*Category, error)
	GetBySiteID(ctx context.Context, siteID int64) ([]*Category, error)
	GetByWPCategoryID(ctx context.Context, siteID int64, wpCategoryID int) (*Category, error)
	Update(ctx context.Context, category *Category) error
	Delete(ctx context.Context, id int64) error
	DeleteBySiteID(ctx context.Context, siteID int64) error
	BulkUpsert(ctx context.Context, siteID int64, categories []*Category) error
}

type StatisticsRepository interface {
	Increment(ctx context.Context, siteID, categoryID int64, date time.Time, articlesPublished, totalWords int) error
	GetByCategory(ctx context.Context, categoryID int64, from, to time.Time) ([]*Statistics, error)
	GetBySite(ctx context.Context, siteID int64, from, to time.Time) ([]*Statistics, error)
}

type Service interface {
	CreateCategory(ctx context.Context, category *Category) error
	GetCategory(ctx context.Context, id int64) (*Category, error)
	ListSiteCategories(ctx context.Context, siteID int64) ([]*Category, error)
	UpdateCategory(ctx context.Context, category *Category) error
	DeleteCategory(ctx context.Context, id int64) error

	SyncFromWordPress(ctx context.Context, siteID int64) error
	CreateInWordPress(ctx context.Context, category *Category) error
	UpdateInWordPress(ctx context.Context, category *Category) error

	GetStatistics(ctx context.Context, categoryID int64, from, to time.Time) ([]*Statistics, error)
}
