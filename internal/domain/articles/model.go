package articles

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution"
)

type Repository interface {
	Create(ctx context.Context, article *entities.Article) error
	GetByID(ctx context.Context, id int64) (*entities.Article, error)
	GetByWPPostID(ctx context.Context, siteID int64, wpPostID int) (*entities.Article, error)
	Update(ctx context.Context, article *entities.Article) error
	Delete(ctx context.Context, id int64) error

	ListBySite(ctx context.Context, siteID int64, limit, offset int) ([]*entities.Article, error)
	ListByJob(ctx context.Context, jobID int64) ([]*entities.Article, error)
	ListByTopic(ctx context.Context, topicID int64) ([]*entities.Article, error)

	GetByStatus(ctx context.Context, siteID int64, status entities.Status) ([]*entities.Article, error)
	GetBySource(ctx context.Context, siteID int64, source entities.Source) ([]*entities.Article, error)
	GetEdited(ctx context.Context, siteID int64) ([]*entities.Article, error)

	CountBySite(ctx context.Context, siteID int64) (int, error)
	CountByStatus(ctx context.Context, siteID int64, status entities.Status) (int, error)
	CountByJob(ctx context.Context, jobID int64) (int, error)

	GetByWPPostIDs(ctx context.Context, siteID int64, wpPostIDs []int) ([]*entities.Article, error)
	GetUnsynced(ctx context.Context, siteID int64, since time.Time) ([]*entities.Article, error)
	UpdateSyncStatus(ctx context.Context, id int64, lastSyncedAt time.Time) error
	UpdatePublishStatus(ctx context.Context, id int64, status entities.Status, publishedAt *time.Time) error

	BulkCreate(ctx context.Context, articles []*entities.Article) error
	BulkUpdateWPInfo(ctx context.Context, updates []*entities.WPInfoUpdate) error
}

type Service interface {
	CreateArticle(ctx context.Context, article *entities.Article) error
	GetArticle(ctx context.Context, id int64) (*entities.Article, error)
	ListArticles(ctx context.Context, siteID int64, limit, offset int) ([]*entities.Article, int, error)
	UpdateArticle(ctx context.Context, article *entities.Article) error
	DeleteArticle(ctx context.Context, id int64) error

	ImportFromWordPress(ctx context.Context, siteID int64, wpPostID int) (*entities.Article, error)
	ImportAllFromSite(ctx context.Context, siteID int64) (int, error)
	SyncFromWordPress(ctx context.Context, siteID int64) error

	PublishToWordPress(ctx context.Context, article *entities.Article) error
	UpdateInWordPress(ctx context.Context, article *entities.Article) error
	DeleteFromWordPress(ctx context.Context, id int64) error

	CreateDraft(ctx context.Context, exec *execution.Execution, title, content string) (*entities.Article, error)
	PublishDraft(ctx context.Context, id int64) error
}
