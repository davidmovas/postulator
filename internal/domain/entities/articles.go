package entities

import "time"

type (
	Source        string
	ArticleStatus string
)

const (
	SourceGenerated Source = "generated" // создана через джобу
	SourceImported  Source = "imported"  // импортирована из WP
	SourceManual    Source = "manual"    // создана вручную в приложении
)

const (
	StatusDraft     ArticleStatus = "draft"
	StatusPublished ArticleStatus = "published"
	StatusFailed    ArticleStatus = "failed"
	StatusUnknown   ArticleStatus = "unknown"
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
	Status        ArticleStatus
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
