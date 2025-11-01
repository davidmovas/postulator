package articles

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/domain/jobs/execution"
)

type (
	Source string
	Status string
)

const (
	SourceGenerated Source = "generated" // создана через джобу
	SourceImported  Source = "imported"  // импортирована из WP
	SourceManual    Source = "manual"    // создана вручную в приложении
)

const (
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
	StatusFailed    Status = "failed"
)

type Article struct {
	ID            int64
	SiteID        int64
	JobID         *int64
	TopicID       int64
	Title         string
	OriginalTitle string
	Content       string
	Excerpt       *string
	WPPostID      int
	WPPostURL     string
	WPCategoryIDs []int
	Status        Status
	WordCount     *int
	Source        Source
	IsEdited      bool
	CreatedAt     time.Time
	PublishedAt   *time.Time
	UpdatedAt     time.Time
	LastSyncedAt  *time.Time
}

type WPInfoUpdate struct {
	ID          int64
	WPPostID    int
	WPPostURL   string
	Status      Status
	PublishedAt *time.Time
}

type Repository interface {
	Create(ctx context.Context, article *Article) error
	GetByID(ctx context.Context, id int64) (*Article, error)
	GetByWPPostID(ctx context.Context, siteID int64, wpPostID int) (*Article, error)
	Update(ctx context.Context, article *Article) error
	Delete(ctx context.Context, id int64) error

	ListBySite(ctx context.Context, siteID int64, limit, offset int) ([]*Article, error)
	ListByJob(ctx context.Context, jobID int64) ([]*Article, error)
	ListByTopic(ctx context.Context, topicID int64) ([]*Article, error)

	GetByStatus(ctx context.Context, siteID int64, status Status) ([]*Article, error)
	GetBySource(ctx context.Context, siteID int64, source Source) ([]*Article, error)
	GetEdited(ctx context.Context, siteID int64) ([]*Article, error)

	CountBySite(ctx context.Context, siteID int64) (int, error)
	CountByStatus(ctx context.Context, siteID int64, status Status) (int, error)
	CountByJob(ctx context.Context, jobID int64) (int, error)

	GetByWPPostIDs(ctx context.Context, siteID int64, wpPostIDs []int) ([]*Article, error)
	GetUnsynced(ctx context.Context, siteID int64, since time.Time) ([]*Article, error)
	UpdateSyncStatus(ctx context.Context, id int64, lastSyncedAt time.Time) error
	UpdatePublishStatus(ctx context.Context, id int64, status Status, publishedAt *time.Time) error

	BulkCreate(ctx context.Context, articles []*Article) error
	BulkUpdateWPInfo(ctx context.Context, updates []*WPInfoUpdate) error
}

type Service interface {
	CreateArticle(ctx context.Context, article *Article) error
	GetArticle(ctx context.Context, id int64) (*Article, error)
	ListArticles(ctx context.Context, siteID int64, limit, offset int) ([]*Article, int, error)
	UpdateArticle(ctx context.Context, article *Article) error
	DeleteArticle(ctx context.Context, id int64) error

	ImportFromWordPress(ctx context.Context, siteID int64, wpPostID int) (*Article, error)
	ImportAllFromSite(ctx context.Context, siteID int64) (int, error)
	SyncWithWordPress(ctx context.Context, id int64) error

	PublishToWordPress(ctx context.Context, article *Article) error
	UpdateInWordPress(ctx context.Context, article *Article) error
	DeleteFromWordPress(ctx context.Context, id int64) error

	CreateDraft(ctx context.Context, exec *execution.Execution, title, content string) (*Article, error)
	PublishDraft(ctx context.Context, id int64) error
}
