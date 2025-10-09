package entities

import "time"

type ArticleStatus string

const (
	ArticleStatusDraft     ArticleStatus = "draft"
	ArticleStatusPublished ArticleStatus = "published"
	ArticleStatusFailed    ArticleStatus = "failed"
)

type Article struct {
	ID            int64
	SiteID        int64
	JobID         *int64 // Optional - can be NULL if created manually
	TopicID       int64
	Title         string // Final title (may be AI-generated if strategy is variation)
	OriginalTitle string // Original title from topic
	Content       string
	Excerpt       *string
	WPPostID      int
	WPPostURL     string
	WPCategoryID  int
	Status        ArticleStatus
	WordCount     *int
	CreatedAt     time.Time
	PublishedAt   *time.Time
}
