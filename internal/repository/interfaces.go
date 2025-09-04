package repository

import (
	"Postulator/internal/models"
	"context"
)

// BaseRepository defines common repository operations
type BaseRepository[T any] interface {
	Create(ctx context.Context, entity *T) error
	GetByID(ctx context.Context, id int64) (*T, error)
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, limit, offset int) ([]*T, error)
	Count(ctx context.Context) (int64, error)
}

// SiteRepository defines operations for Site entity
type SiteRepository interface {
	BaseRepository[models.Site]
	GetByURL(ctx context.Context, url string) (*models.Site, error)
	GetActive(ctx context.Context) ([]*models.Site, error)
	UpdateStatus(ctx context.Context, id int64, status string) error
	GetByStatus(ctx context.Context, status string) ([]*models.Site, error)
}

// TopicRepository defines operations for Topic entity
type TopicRepository interface {
	BaseRepository[models.Topic]
	GetActive(ctx context.Context) ([]*models.Topic, error)
	GetByCategory(ctx context.Context, category string) ([]*models.Topic, error)
	SearchByKeywords(ctx context.Context, keywords string) ([]*models.Topic, error)
}

// SiteTopicRepository defines operations for SiteTopic entity
type SiteTopicRepository interface {
	BaseRepository[models.SiteTopic]
	GetBySiteID(ctx context.Context, siteID int64) ([]*models.SiteTopic, error)
	GetByTopicID(ctx context.Context, topicID int64) ([]*models.SiteTopic, error)
	GetActive(ctx context.Context) ([]*models.SiteTopic, error)
	GetBySiteAndTopic(ctx context.Context, siteID, topicID int64) (*models.SiteTopic, error)
	DeleteBySiteAndTopic(ctx context.Context, siteID, topicID int64) error
}

// ScheduleRepository defines operations for Schedule entity
type ScheduleRepository interface {
	BaseRepository[models.Schedule]
	GetBySiteID(ctx context.Context, siteID int64) ([]*models.Schedule, error)
	GetActive(ctx context.Context) ([]*models.Schedule, error)
	GetDueSchedules(ctx context.Context) ([]*models.Schedule, error)
	UpdateLastRun(ctx context.Context, id int64) error
	UpdateNextRun(ctx context.Context, id int64, nextRun int64) error
}

// ArticleRepository defines operations for Article entity
type ArticleRepository interface {
	BaseRepository[models.Article]
	GetBySiteID(ctx context.Context, siteID int64, limit, offset int) ([]*models.Article, error)
	GetByTopicID(ctx context.Context, topicID int64, limit, offset int) ([]*models.Article, error)
	GetByStatus(ctx context.Context, status string, limit, offset int) ([]*models.Article, error)
	GetBySiteAndStatus(ctx context.Context, siteID int64, status string) ([]*models.Article, error)
	UpdateStatus(ctx context.Context, id int64, status string) error
	SetWordPressID(ctx context.Context, id int64, wpID int64) error
	GetRecentBySite(ctx context.Context, siteID int64, days int) ([]*models.Article, error)
}

// PostingJobRepository defines operations for PostingJob entity
type PostingJobRepository interface {
	BaseRepository[models.PostingJob]
	GetBySiteID(ctx context.Context, siteID int64, limit, offset int) ([]*models.PostingJob, error)
	GetByStatus(ctx context.Context, status string, limit, offset int) ([]*models.PostingJob, error)
	GetPending(ctx context.Context) ([]*models.PostingJob, error)
	GetRunning(ctx context.Context) ([]*models.PostingJob, error)
	UpdateStatus(ctx context.Context, id int64, status string) error
	UpdateProgress(ctx context.Context, id int64, progress int) error
	SetError(ctx context.Context, id int64, errorMsg string) error
	Complete(ctx context.Context, id int64) error
	Start(ctx context.Context, id int64) error
}

// SettingRepository defines operations for Setting entity
type SettingRepository interface {
	Get(ctx context.Context, key string) (*Setting, error)
	Set(ctx context.Context, key, value string) error
	GetAll(ctx context.Context) ([]*Setting, error)
	Delete(ctx context.Context, key string) error
	GetByPrefix(ctx context.Context, prefix string) ([]*Setting, error)
}

// RepositoryContainer holds all repositories
type RepositoryContainer struct {
	Site       SiteRepository
	Topic      TopicRepository
	SiteTopic  SiteTopicRepository
	Schedule   ScheduleRepository
	Article    ArticleRepository
	PostingJob PostingJobRepository
	Setting    SettingRepository
}
