package topic

import (
	"Postulator/internal/domain/entities"
	"context"
)

type BatchCreateResult struct {
	Created      []string // Successfully created topic titles
	Skipped      []string // Skipped duplicate topic titles
	TotalAdded   int
	TotalSkipped int
}

type ITopicRepository interface {
	Create(ctx context.Context, topic *entities.Topic) (int, error)
	CreateBatch(ctx context.Context, topics []*entities.Topic) (*BatchCreateResult, error)
	GetByID(ctx context.Context, id int64) (*entities.Topic, error)
	GetAll(ctx context.Context) ([]*entities.Topic, error)
	Update(ctx context.Context, topic *entities.Topic) error
	Delete(ctx context.Context, id int64) error
}

type ISiteTopicRepository interface {
	Assign(ctx context.Context, st *entities.SiteTopic) error
	GetBySiteID(ctx context.Context, siteID int64) ([]*entities.SiteTopic, error)
	GetByTopicID(ctx context.Context, topicID int64) ([]*entities.SiteTopic, error)
	Update(ctx context.Context, st *entities.SiteTopic) error
	Unassign(ctx context.Context, siteID, topicID int64) error
}

type IUsedTopicRepository interface {
	MarkAsUsed(ctx context.Context, siteID, topicID int64) error
	IsUsed(ctx context.Context, siteID, topicID int64) (bool, error)
	GetUnusedTopics(ctx context.Context, siteID int64) ([]*entities.Topic, error)
	CountUnusedTopics(ctx context.Context, siteID int64) (int, error)
}

type IService interface {
	CreateTopic(ctx context.Context, topic *entities.Topic) (int, error)
	CreateTopicBatch(ctx context.Context, topics []*entities.Topic) (*BatchCreateResult, error)
	GetTopic(ctx context.Context, id int64) (*entities.Topic, error)
	ListTopics(ctx context.Context) ([]*entities.Topic, error)
	UpdateTopic(ctx context.Context, topic *entities.Topic) error
	DeleteTopic(ctx context.Context, id int64) error

	AssignToSite(ctx context.Context, siteID, topicID, categoryID int64, strategy entities.TopicStrategy) error
	UnassignFromSite(ctx context.Context, siteID, topicID int64) error
	GetSiteTopics(ctx context.Context, siteID int64) ([]*entities.SiteTopic, error)
	GetTopicsBySite(ctx context.Context, siteID int64) ([]*entities.Topic, error)

	HasAvailableTopics(ctx context.Context, siteID int64, strategy entities.TopicStrategy) (bool, error)
	GetAvailableTopic(ctx context.Context, siteID int64, strategy entities.TopicStrategy) (*entities.Topic, error)
	MarkTopicAsUsed(ctx context.Context, siteID, topicID int64) error
	CountUnusedTopics(ctx context.Context, siteID int64) (int, error)
}
