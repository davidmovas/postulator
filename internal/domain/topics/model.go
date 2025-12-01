package topics

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/entities"
)

type Repository interface {
	Create(ctx context.Context, topic *entities.Topic) (*entities.Topic, error)
	CreateBatch(ctx context.Context, topics ...*entities.Topic) (*entities.BatchResult, error)
	GetByID(ctx context.Context, id int64) (*entities.Topic, error)
	GetAll(ctx context.Context) ([]*entities.Topic, error)
	GetByTitle(ctx context.Context, title string) (*entities.Topic, error)
	GetByTitles(ctx context.Context, titles []string) ([]*entities.Topic, error)
	Update(ctx context.Context, topic *entities.Topic) error
	Delete(ctx context.Context, id int64) error
	Count(ctx context.Context) (int, error)
}

type SiteTopicRepository interface {
	Assign(ctx context.Context, siteID, topicID int64) error
	Unassign(ctx context.Context, siteID, topicID int64) error
	GetBySiteID(ctx context.Context, siteID int64) ([]*entities.Topic, error)
	IsAssigned(ctx context.Context, siteID, topicID int64) (bool, error)
	GetAssignedForSite(ctx context.Context, siteID int64, topicIDs []int64) ([]int64, error)
}

type UsageRepository interface {
	MarkAsUsed(ctx context.Context, siteID, topicID int64) error
	IsUsed(ctx context.Context, siteID, topicID int64) (bool, error)
	GetUnused(ctx context.Context, siteID int64, topicIDs []int64) ([]*entities.Topic, error)
	CountUnused(ctx context.Context, siteID int64, topicIDs []int64) (int, error)
	GetNextUnused(ctx context.Context, siteID int64, topicIDs []int64) (*entities.Topic, error)
}

type Service interface {
	CreateTopic(ctx context.Context, topic *entities.Topic) error
	CreateTopics(ctx context.Context, topics ...*entities.Topic) (*entities.BatchResult, error)
	CreateAndAssignToSite(ctx context.Context, siteID int64, topics ...*entities.Topic) (*entities.ImportAssignResult, error)
	GetTopic(ctx context.Context, id int64) (*entities.Topic, error)
	ListTopics(ctx context.Context) ([]*entities.Topic, error)
	GetByTitles(ctx context.Context, titles []string) ([]*entities.Topic, error)
	UpdateTopic(ctx context.Context, topic *entities.Topic) error
	DeleteTopic(ctx context.Context, id int64) error

	AssignToSite(ctx context.Context, siteID int64, topicIDs ...int64) error
	UnassignFromSite(ctx context.Context, siteID int64, topicIDs ...int64) error
	GetSiteTopics(ctx context.Context, siteID int64) ([]*entities.Topic, error)
	GetAssignedForSite(ctx context.Context, siteID int64, topicIDs []int64) ([]int64, error)
	GetUnusedSiteTopics(ctx context.Context, siteID int64) ([]*entities.Topic, error)

	GenerateVariations(ctx context.Context, providerID int64, topicID int64, count int) ([]*entities.Topic, error)
	GetOrGenerateVariation(ctx context.Context, providerID, siteID, originalID int64) (*entities.Topic, error)

	GetNextTopicForJob(ctx context.Context, job *entities.Job) (*entities.Topic, error)
	MarkTopicUsed(ctx context.Context, siteID, topicID int64) error
	CountUnused(ctx context.Context, siteID int64, topicIDs []int64) (int, error)

	GetStrategy(strategyType entities.TopicStrategy) (TopicStrategyHandler, error)
	GetSelectableSiteTopics(ctx context.Context, siteID int64, strategyType entities.TopicStrategy) ([]*entities.Topic, error)

	GetJobRemainingTopics(ctx context.Context, job *entities.Job) ([]*entities.Topic, int, error)
}
