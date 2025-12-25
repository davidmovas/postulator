package entities

import "time"

type (
	Source        string
	ArticleStatus string
	ContentType   string
)

const (
	ContentTypePost ContentType = "post"
	ContentTypePage ContentType = "page"
)

const (
	SourceGenerated Source = "generated" // создана через джобу
	SourceImported  Source = "imported"  // импортирована из WP
	SourceManual    Source = "manual"    // создана вручную в приложении
)

const (
	StatusDraft     ArticleStatus = "draft"
	StatusPublished ArticleStatus = "published"
	StatusPending   ArticleStatus = "pending"
	StatusPrivate   ArticleStatus = "private"
	StatusFailed    ArticleStatus = "failed"
	StatusUnknown   ArticleStatus = "unknown"
)

type Article struct {
	ID            int64
	SiteID        int64
	JobID         *int64
	TopicID       *int64
	Title         string
	OriginalTitle string
	Content       string
	Excerpt       *string
	WPPostID      int
	WPPostURL     string
	WPCategoryIDs []int
	WPTagIDs      []int
	Status        ArticleStatus
	WordCount     *int
	Source        Source
	IsEdited      bool
	CreatedAt     time.Time
	PublishedAt   *time.Time
	UpdatedAt     time.Time
	LastSyncedAt  *time.Time

	// SEO & WordPress fields
	Slug             *string
	FeaturedMediaID  *int
	FeaturedMediaURL *string
	MetaDescription  *string
	Author           *int

	// Page-specific fields
	ContentType   ContentType
	WPPageID      *int
	ParentPageID  *int
	MenuOrder     *int
	PageTemplate  *string
	SitemapNodeID *int64
}

type WPInfoUpdate struct {
	ID          int64
	WPPostID    int
	WPPostURL   string
	Status      Status
	PublishedAt *time.Time
}
