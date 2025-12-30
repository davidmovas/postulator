package articles

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
)

// ListFilter contains filter options for listing articles
type ListFilter struct {
	SiteID     int64
	Status     *entities.ArticleStatus
	Source     *entities.Source
	CategoryID *int
	Search     *string
	SortBy     string // title, created_at, published_at, updated_at, word_count
	SortOrder  string // asc, desc
	Limit      int
	Offset     int
}

// ListResult contains paginated list result
type ListResult struct {
	Articles []*entities.Article
	Total    int
}

type Repository interface {
	Create(ctx context.Context, article *entities.Article) error
	GetByID(ctx context.Context, id int64) (*entities.Article, error)
	GetByWPPostID(ctx context.Context, siteID int64, wpPostID int) (*entities.Article, error)
	Update(ctx context.Context, article *entities.Article) error
	Delete(ctx context.Context, id int64) error

	// List with filters and pagination
	List(ctx context.Context, filter *ListFilter) (*ListResult, error)

	// Legacy methods (kept for backwards compatibility)
	ListBySite(ctx context.Context, siteID int64, limit, offset int) ([]*entities.Article, error)
	ListByJob(ctx context.Context, jobID int64) ([]*entities.Article, error)
	ListByTopic(ctx context.Context, topicID int64) ([]*entities.Article, error)

	GetByStatus(ctx context.Context, siteID int64, status entities.ArticleStatus) ([]*entities.Article, error)
	GetBySource(ctx context.Context, siteID int64, source entities.Source) ([]*entities.Article, error)
	GetEdited(ctx context.Context, siteID int64) ([]*entities.Article, error)

	CountBySite(ctx context.Context, siteID int64) (int, error)
	CountByStatus(ctx context.Context, siteID int64, status entities.ArticleStatus) (int, error)
	CountByJob(ctx context.Context, jobID int64) (int, error)

	GetByWPPostIDs(ctx context.Context, siteID int64, wpPostIDs []int) ([]*entities.Article, error)
	GetUnsynced(ctx context.Context, siteID int64, since time.Time) ([]*entities.Article, error)
	UpdateSyncStatus(ctx context.Context, id int64, lastSyncedAt time.Time) error
	UpdatePublishStatus(ctx context.Context, id int64, status entities.ArticleStatus, publishedAt *time.Time) error

	BulkCreate(ctx context.Context, articles []*entities.Article) error
	BulkUpdateWPInfo(ctx context.Context, updates []*entities.WPInfoUpdate) error
	BulkDelete(ctx context.Context, ids []int64) error
}

// GenerateContentInput represents input for AI content generation
type GenerateContentInput struct {
	SiteID            int64
	ProviderID        int64
	PromptID          int64
	TopicID           *int64 // Optional - if nil, use custom topic title
	CustomTopicTitle  string // Used when TopicID is nil
	PlaceholderValues map[string]string
	UseWebSearch      bool // Enable web search for models that support it
}

// GenerateContentResult represents the result of AI content generation
type GenerateContentResult struct {
	Title           string
	Content         string
	Excerpt         string
	MetaDescription string
	TopicID         *int64 // The topic ID that was used or created
}

type Service interface {
	CreateArticle(ctx context.Context, article *entities.Article) error
	CreateAndPublishArticle(ctx context.Context, article *entities.Article) (*entities.Article, error)
	GetArticle(ctx context.Context, id int64) (*entities.Article, error)
	ListArticles(ctx context.Context, filter *ListFilter) (*ListResult, error)
	UpdateArticle(ctx context.Context, article *entities.Article) error
	UpdateAndSyncArticle(ctx context.Context, article *entities.Article) (*entities.Article, error)
	DeleteArticle(ctx context.Context, id int64) error
	BulkDeleteArticles(ctx context.Context, ids []int64) error

	ImportFromWordPress(ctx context.Context, siteID int64, wpPostID int) (*entities.Article, error)
	ImportAllFromSite(ctx context.Context, siteID int64) (int, error)
	SyncFromWordPress(ctx context.Context, siteID int64) error

	PublishToWordPress(ctx context.Context, article *entities.Article) error
	UpdateInWordPress(ctx context.Context, article *entities.Article) error
	DeleteFromWordPress(ctx context.Context, id int64) error
	BulkPublishToWordPress(ctx context.Context, ids []int64) (int, error)
	BulkDeleteFromWordPress(ctx context.Context, ids []int64) (int, error)

	CreateDraft(ctx context.Context, exec *entities.Execution, title, content string) (*entities.Article, error)
	PublishDraft(ctx context.Context, id int64) error

	GenerateContent(ctx context.Context, input *GenerateContentInput) (*GenerateContentResult, error)
}
