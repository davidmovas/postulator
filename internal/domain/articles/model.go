package articles

import (
	"Postulator/internal/domain/jobs/execution"
	"context"
	"time"
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
