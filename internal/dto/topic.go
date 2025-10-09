package dto

import (
	"Postulator/internal/domain/entities"
)

type Topic struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	CreatedAt string `json:"createdAt"`
}

type SiteTopic struct {
	ID         int64  `json:"id"`
	SiteID     int64  `json:"siteId"`
	TopicID    int64  `json:"topicId"`
	CategoryID int64  `json:"categoryId"`
	Strategy   string `json:"strategy"`
	CreatedAt  string `json:"createdAt"`
}

type UsedTopic struct {
	SiteID  int64  `json:"siteId"`
	TopicID int64  `json:"topicId"`
	UsedAt  string `json:"usedAt"`
}

func FromTopic(e *entities.Topic) *Topic {
	if e == nil {
		return nil
	}
	return &Topic{ID: e.ID, Title: e.Title, CreatedAt: e.CreatedAt.UTC().Format(timeLayout)}
}

func FromTopics(items []*entities.Topic) []*Topic {
	out := make([]*Topic, 0, len(items))
	for _, it := range items {
		out = append(out, FromTopic(it))
	}
	return out
}

func FromSiteTopic(e *entities.SiteTopic) *SiteTopic {
	if e == nil {
		return nil
	}
	return &SiteTopic{
		ID:         e.ID,
		SiteID:     e.SiteID,
		TopicID:    e.TopicID,
		CategoryID: e.CategoryID,
		Strategy:   string(e.Strategy),
		CreatedAt:  e.CreatedAt.UTC().Format(timeLayout),
	}
}

func FromSiteTopics(items []*entities.SiteTopic) []*SiteTopic {
	out := make([]*SiteTopic, 0, len(items))
	for _, it := range items {
		out = append(out, FromSiteTopic(it))
	}
	return out
}
