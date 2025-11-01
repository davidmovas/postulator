package topic

import (
	"context"
	"time"
)

type Topic struct {
	ID        int64
	Title     string
	CreatedAt time.Time
}

type BatchResult struct {
	Created       int
	Skipped       int
	SkippedTitles []string
}

type Repository interface {
	Create(ctx context.Context, topic *Topic) (*Topic, error)
	CreateBatch(ctx context.Context, topics ...*Topic) (*BatchResult, error)
	GetByID(ctx context.Context, id int64) (*Topic, error)
	GetAll(ctx context.Context) ([]*Topic, error)
	GetByTitle(ctx context.Context, title string) (*Topic, error)
	Update(ctx context.Context, topic *Topic) error
	Delete(ctx context.Context, id int64) error
	Count(ctx context.Context) (int, error)
}

type SiteTopicRepository interface {
	Assign(ctx context.Context, siteID, topicID int64) error
	Unassign(ctx context.Context, siteID, topicID int64) error
	GetBySiteID(ctx context.Context, siteID int64) ([]*Topic, error)
	IsAssigned(ctx context.Context, siteID, topicID int64) (bool, error)
}

type UsageRepository interface {
	MarkAsUsed(ctx context.Context, siteID, topicID int64) error
	IsUsed(ctx context.Context, siteID, topicID int64) (bool, error)
	GetUnused(ctx context.Context, siteID int64, topicIDs []int64) ([]*Topic, error)
	CountUnused(ctx context.Context, siteID int64, topicIDs []int64) (int, error)
	GetNextUnused(ctx context.Context, siteID int64, topicIDs []int64) (*Topic, error)
}

type Service interface {
	CreateTopic(ctx context.Context, topic *Topic) error
	CreateTopics(ctx context.Context, topics []*Topic) (*BatchResult, error)
	GetTopic(ctx context.Context, id int64) (*Topic, error)
	ListTopics(ctx context.Context) ([]*Topic, error)
	UpdateTopic(ctx context.Context, topic *Topic) error
	DeleteTopic(ctx context.Context, id int64) error

	AssignToSite(ctx context.Context, siteID int64, topicIDs ...int64) error
	UnassignFromSite(ctx context.Context, siteID int64, topicIDs ...int64) error
	GetSiteTopics(ctx context.Context, siteID int64) ([]*Topic, error)

	GenerateVariations(ctx context.Context, topicID int64, count int) ([]*Topic, error)
	GetOrGenerateVariation(ctx context.Context, originalID int64) (*Topic, error)

	GetNextTopicForJob(ctx context.Context, jobID int64) (*Topic, error)
	MarkTopicUsed(ctx context.Context, siteID, topicID int64) error
}
