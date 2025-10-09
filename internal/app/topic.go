package app

import (
	"Postulator/internal/domain/entities"
	"Postulator/internal/dto"
	"Postulator/pkg/errors"
	"context"
)

// Topic bindings

func (a *App) CreateTopic(topic *dto.Topic) *dto.Response[string] {
	if a.topicSvc == nil {
		if err := a.BuildServices(); err != nil {
			return dtoErr[string](errors.Internal(err))
		}
	}
	if topic == nil {
		return dtoErr[string](errors.Validation("topic payload is required"))
	}
	entity := &entities.Topic{ID: topic.ID, Title: topic.Title}
	if err := a.topicSvc.CreateTopic(context.Background(), entity); err != nil {
		return dtoErr[string](asAppErr(err))
	}
	return &dto.Response[string]{Success: true, Data: "created"}
}

func (a *App) GetTopic(id int64) *dto.Response[*dto.Topic] {
	if a.topicSvc == nil {
		if err := a.BuildServices(); err != nil {
			return dtoErr[*dto.Topic](errors.Internal(err))
		}
	}
	t, err := a.topicSvc.GetTopic(context.Background(), id)
	if err != nil {
		return dtoErr[*dto.Topic](asAppErr(err))
	}
	return &dto.Response[*dto.Topic]{Success: true, Data: dto.FromTopic(t)}
}

func (a *App) ListTopics() *dto.Response[[]*dto.Topic] {
	if a.topicSvc == nil {
		if err := a.BuildServices(); err != nil {
			return dtoErr[[]*dto.Topic](errors.Internal(err))
		}
	}
	items, err := a.topicSvc.ListTopics(context.Background())
	if err != nil {
		return dtoErr[[]*dto.Topic](asAppErr(err))
	}
	return &dto.Response[[]*dto.Topic]{Success: true, Data: dto.FromTopics(items)}
}

func (a *App) UpdateTopic(topic *dto.Topic) *dto.Response[string] {
	if a.topicSvc == nil {
		if err := a.BuildServices(); err != nil {
			return dtoErr[string](errors.Internal(err))
		}
	}
	if topic == nil {
		return dtoErr[string](errors.Validation("topic payload is required"))
	}
	entity := &entities.Topic{ID: topic.ID, Title: topic.Title}
	if err := a.topicSvc.UpdateTopic(context.Background(), entity); err != nil {
		return dtoErr[string](asAppErr(err))
	}
	return &dto.Response[string]{Success: true, Data: "updated"}
}

func (a *App) DeleteTopic(id int64) *dto.Response[string] {
	if a.topicSvc == nil {
		if err := a.BuildServices(); err != nil {
			return dtoErr[string](errors.Internal(err))
		}
	}
	if err := a.topicSvc.DeleteTopic(context.Background(), id); err != nil {
		return dtoErr[string](asAppErr(err))
	}
	return &dto.Response[string]{Success: true, Data: "deleted"}
}

// Assignment bindings

func (a *App) AssignTopicToSite(siteID, topicID, categoryID int64, strategy string) *dto.Response[string] {
	if a.topicSvc == nil {
		if err := a.BuildServices(); err != nil {
			return dtoErr[string](errors.Internal(err))
		}
	}
	strat := entities.TopicStrategy(strategy)
	if strat != entities.StrategyUnique && strat != entities.StrategyVariation {
		return dtoErr[string](errors.Validation("invalid topic strategy"))
	}
	if err := a.topicSvc.AssignToSite(context.Background(), siteID, topicID, categoryID, strat); err != nil {
		return dtoErr[string](asAppErr(err))
	}
	return &dto.Response[string]{Success: true, Data: "assigned"}
}

func (a *App) UnassignTopicFromSite(siteID, topicID int64) *dto.Response[string] {
	if a.topicSvc == nil {
		if err := a.BuildServices(); err != nil {
			return dtoErr[string](errors.Internal(err))
		}
	}
	if err := a.topicSvc.UnassignFromSite(context.Background(), siteID, topicID); err != nil {
		return dtoErr[string](asAppErr(err))
	}
	return &dto.Response[string]{Success: true, Data: "unassigned"}
}

func (a *App) GetSiteTopics(siteID int64) *dto.Response[[]*dto.SiteTopic] {
	if a.topicSvc == nil {
		if err := a.BuildServices(); err != nil {
			return dtoErr[[]*dto.SiteTopic](errors.Internal(err))
		}
	}
	items, err := a.topicSvc.GetSiteTopics(context.Background(), siteID)
	if err != nil {
		return dtoErr[[]*dto.SiteTopic](asAppErr(err))
	}
	return &dto.Response[[]*dto.SiteTopic]{Success: true, Data: dto.FromSiteTopics(items)}
}

func (a *App) GetTopicsBySite(siteID int64) *dto.Response[[]*dto.Topic] {
	if a.topicSvc == nil {
		if err := a.BuildServices(); err != nil {
			return dtoErr[[]*dto.Topic](errors.Internal(err))
		}
	}
	items, err := a.topicSvc.GetTopicsBySite(context.Background(), siteID)
	if err != nil {
		return dtoErr[[]*dto.Topic](asAppErr(err))
	}
	return &dto.Response[[]*dto.Topic]{Success: true, Data: dto.FromTopics(items)}
}

func (a *App) GetAvailableTopic(siteID int64, strategy string) *dto.Response[*dto.Topic] {
	if a.topicSvc == nil {
		if err := a.BuildServices(); err != nil {
			return dtoErr[*dto.Topic](errors.Internal(err))
		}
	}
	strat := entities.TopicStrategy(strategy)
	if strat != entities.StrategyUnique && strat != entities.StrategyVariation {
		return dtoErr[*dto.Topic](errors.Validation("invalid topic strategy"))
	}
	t, err := a.topicSvc.GetAvailableTopic(context.Background(), siteID, strat)
	if err != nil {
		return dtoErr[*dto.Topic](asAppErr(err))
	}
	return &dto.Response[*dto.Topic]{Success: true, Data: dto.FromTopic(t)}
}

func (a *App) MarkTopicAsUsed(siteID, topicID int64) *dto.Response[string] {
	if a.topicSvc == nil {
		if err := a.BuildServices(); err != nil {
			return dtoErr[string](errors.Internal(err))
		}
	}
	if err := a.topicSvc.MarkTopicAsUsed(context.Background(), siteID, topicID); err != nil {
		return dtoErr[string](asAppErr(err))
	}
	return &dto.Response[string]{Success: true, Data: "marked"}
}

func (a *App) CountUnusedTopics(siteID int64) *dto.Response[int] {
	if a.topicSvc == nil {
		if err := a.BuildServices(); err != nil {
			return dtoErr[int](errors.Internal(err))
		}
	}
	cnt, err := a.topicSvc.CountUnusedTopics(context.Background(), siteID)
	if err != nil {
		return dtoErr[int](asAppErr(err))
	}
	return &dto.Response[int]{Success: true, Data: cnt}
}
