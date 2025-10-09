package main

import (
	"Postulator/internal/domain/entities"
	"context"
)

type TopicRepository interface {
	Create(ctx context.Context, topic *entities.Topic) error
	GetByID(ctx context.Context, id int64) (*entities.Topic, error)
	GetAll(ctx context.Context) ([]*entities.Topic, error)
	Update(ctx context.Context, topic *entities.Topic) error
	Delete(ctx context.Context, id int64) error
}

type TitleRepository interface {
	CreateBatch(ctx context.Context, titles []*entities.Title) error
	GetByTopicID(ctx context.Context, topicID int64) ([]*entities.Title, error)
	GetByID(ctx context.Context, id int64) (*entities.Title, error)
	Delete(ctx context.Context, id int64) error
	CountByTopicID(ctx context.Context, topicID int64) (int, error)
}

type SiteTopicRepository interface {
	Assign(ctx context.Context, st *entities.SiteTopic) error
	GetBySiteID(ctx context.Context, siteID int64) ([]*entities.SiteTopic, error)
	GetByTopicID(ctx context.Context, topicID int64) ([]*entities.SiteTopic, error)
	Update(ctx context.Context, st *entities.SiteTopic) error
	Unassign(ctx context.Context, siteID, topicID int64) error
}

type UsedTitleRepository interface {
	MarkAsUsed(ctx context.Context, siteID, titleID int64) error
	IsUsed(ctx context.Context, siteID, titleID int64) (bool, error)
	GetUnusedTitles(ctx context.Context, siteID, topicID int64) ([]*entities.Title, error)
	CountUnused(ctx context.Context, siteID, topicID int64) (int, error)
}

type TopicService interface {
	CreateTopic(ctx context.Context, topic *entities.Topic) error
	GetTopic(ctx context.Context, id int64) (*entities.Topic, error)
	ListTopics(ctx context.Context) ([]*entities.Topic, error)
	UpdateTopic(ctx context.Context, topic *entities.Topic) error
	DeleteTopic(ctx context.Context, id int64) error

	ImportTitles(ctx context.Context, topicID int64, filePath string, format string) error
	AddTitles(ctx context.Context, topicID int64, titles []string) error
	GetTitles(ctx context.Context, topicID int64) ([]*entities.Title, error)

	AssignToSite(ctx context.Context, siteID, topicID int64, categoryID *int64, strategy entities.TopicStrategy) error
	UnassignFromSite(ctx context.Context, siteID, topicID int64) error
	GetSiteTopics(ctx context.Context, siteID int64) ([]*entities.SiteTopic, error)

	GetAvailableTitle(ctx context.Context, siteID, topicID int64, strategy entities.TopicStrategy) (*entities.Title, error)
}
