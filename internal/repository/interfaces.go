package repository

import (
	"Postulator/internal/models"
	"context"
	"time"
)

type SiteRepository interface {
	GetSites(ctx context.Context, limit int, offset int) (*models.PaginationResult[*models.Site], error)
	GetSite(ctx context.Context, id int64) (*models.Site, error)
	CreateSite(ctx context.Context, site *models.Site) (*models.Site, error)
	UpdateSite(ctx context.Context, site *models.Site) (*models.Site, error)
	ActivateSite(ctx context.Context, id int64) error
	DeactivateSite(ctx context.Context, id int64) error
	SetCheckStatus(ctx context.Context, id int64, checkTime time.Time, status string) error
	DeleteSite(ctx context.Context, id int64) error
}

type TopicRepository interface {
	GetTopics(ctx context.Context, limit int, offset int) (*models.PaginationResult[*models.Topic], error)
	GetTopic(ctx context.Context, id int64) (*models.Topic, error)
	GetTopicsBySiteID(ctx context.Context, siteID int64, limit int, offset int) (*models.PaginationResult[*models.Topic], error)
	CreateTopic(ctx context.Context, topic *models.Topic) (*models.Topic, error)
	UpdateTopic(ctx context.Context, topic *models.Topic) (*models.Topic, error)
	DeleteTopic(ctx context.Context, id int64) error
	GetAllTopicsForRandomSelection(ctx context.Context) ([]*models.Topic, error)

	// Bulk operations
	GetTopicByTitle(ctx context.Context, title string) (*models.Topic, error)
	BulkCreateTopicsWithSiteBinding(ctx context.Context, siteID int64, topics []*models.Topic) ([]*models.Topic, error)
}

type SiteTopicRepository interface {
	CreateSiteTopic(ctx context.Context, siteTopic *models.SiteTopic) (*models.SiteTopic, error)
	GetSiteTopics(ctx context.Context, siteID int64, limit int, offset int) (*models.PaginationResult[*models.SiteTopic], error)
	GetTopicSites(ctx context.Context, topicID int64, limit int, offset int) (*models.PaginationResult[*models.SiteTopic], error)
	GetSiteTopic(ctx context.Context, siteID int64, topicID int64) (*models.SiteTopic, error)
	UpdateSiteTopic(ctx context.Context, siteTopic *models.SiteTopic) (*models.SiteTopic, error)
	DeleteSiteTopic(ctx context.Context, id int64) error
	DeleteSiteTopicBySiteAndTopic(ctx context.Context, siteID int64, topicID int64) error

	GetSiteTopicsForSelection(ctx context.Context, siteID int64, strategy string) ([]*models.SiteTopic, error)
	UpdateSiteTopicUsage(ctx context.Context, siteTopicID int64, strategy string) error
	GetTopicStats(ctx context.Context, siteID int64) (*models.TopicStats, error)

	// Bulk operations
	ReassignTopicsToSite(ctx context.Context, fromSiteID, toSiteID int64, topicIDs []int64) error
}

type TopicUsageRepository interface {
	CreateTopicUsage(ctx context.Context, usage *models.TopicUsage) (*models.TopicUsage, error)
	GetTopicUsageHistory(ctx context.Context, siteID int64, topicID int64, limit int, offset int) (*models.PaginationResult[*models.TopicUsage], error)
	GetSiteUsageHistory(ctx context.Context, siteID int64, limit int, offset int) (*models.PaginationResult[*models.TopicUsage], error)
	RecordTopicUsage(ctx context.Context, siteID, topicID, articleID int64, strategy string) error
}

type PromptRepository interface {
	GetPrompts(ctx context.Context, limit int, offset int) (*models.PaginationResult[*models.Prompt], error)
	GetPrompt(ctx context.Context, id int64) (*models.Prompt, error)
	CreatePrompt(ctx context.Context, prompt *models.Prompt) (*models.Prompt, error)
	UpdatePrompt(ctx context.Context, prompt *models.Prompt) (*models.Prompt, error)
	DeletePrompt(ctx context.Context, id int64) error
	GetDefaultPrompt(ctx context.Context) (*models.Prompt, error)
	SetDefaultPrompt(ctx context.Context, id int64) error
}

type SitePromptRepository interface {
	CreateSitePrompt(ctx context.Context, sitePrompt *models.SitePrompt) (*models.SitePrompt, error)
	GetSitePrompt(ctx context.Context, siteID int64) (*models.SitePrompt, error)
	UpdateSitePrompt(ctx context.Context, sitePrompt *models.SitePrompt) (*models.SitePrompt, error)
	DeleteSitePrompt(ctx context.Context, id int64) error
	DeleteSitePromptBySite(ctx context.Context, siteID int64) error
	GetPromptSites(ctx context.Context, promptID int64, limit int, offset int) (*models.PaginationResult[*models.SitePrompt], error)
}

type ArticleRepository interface {
	GetArticles(ctx context.Context, limit int, offset int) (*models.PaginationResult[*models.Article], error)
	GetArticle(ctx context.Context, id int64) (*models.Article, error)
	GetArticlesBySiteID(ctx context.Context, siteID int64, limit int, offset int) (*models.PaginationResult[*models.Article], error)
	CreateArticle(ctx context.Context, article *models.Article) (*models.Article, error)
	UpdateArticle(ctx context.Context, article *models.Article) (*models.Article, error)
	DeleteArticle(ctx context.Context, id int64) error
	GetArticleByHash(ctx context.Context, hash string) (*models.Article, error)
	GetArticleBySlug(ctx context.Context, siteID int64, slug string) (*models.Article, error)
}

type PostingJobRepository interface {
	GetJobs(ctx context.Context, limit int, offset int) (*models.PaginationResult[*models.PostingJob], error)
	GetJob(ctx context.Context, id int64) (*models.PostingJob, error)
	GetJobsBySiteID(ctx context.Context, siteID int64, limit int, offset int) (*models.PaginationResult[*models.PostingJob], error)
	CreateJob(ctx context.Context, job *models.PostingJob) (*models.PostingJob, error)
	UpdateJob(ctx context.Context, job *models.PostingJob) (*models.PostingJob, error)
	DeleteJob(ctx context.Context, id int64) error
	GetJobsByStatus(ctx context.Context, status string, limit int, offset int) (*models.PaginationResult[*models.PostingJob], error)
}

type ScheduleRepository interface {
	GetSchedules(ctx context.Context, limit int, offset int) (*models.PaginationResult[*models.Schedule], error)
	GetSchedule(ctx context.Context, id int64) (*models.Schedule, error)
	GetSchedulesBySiteID(ctx context.Context, siteID int64, limit int, offset int) (*models.PaginationResult[*models.Schedule], error)
	CreateSchedule(ctx context.Context, schedule *models.Schedule) (*models.Schedule, error)
	UpdateSchedule(ctx context.Context, schedule *models.Schedule) (*models.Schedule, error)
	DeleteSchedule(ctx context.Context, id int64) error
	GetActive(ctx context.Context) ([]*models.Schedule, error)
	GetByID(ctx context.Context, id int64) (*models.Schedule, error)
	UpdateLastRun(ctx context.Context, id int64) error
	UpdateNextRun(ctx context.Context, id int64, nextRun int64) error
}
