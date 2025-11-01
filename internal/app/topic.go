package app

import (
	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/dto"
	"github.com/davidmovas/postulator/pkg/ctx"
	"github.com/davidmovas/postulator/pkg/errors"
)

func (a *App) CreateTopic(topic *dto.Topic) *dto.Response[int] {
	if topic == nil {
		return dtoErr[int](errors.Validation("topic payload is required"))
	}

	entity := &entities.Topic{
		ID:    topic.ID,
		Title: topic.Title,
	}

	id, err := a.topicSvc.CreateTopic(ctx.FastCtx(), entity)
	if err != nil {
		return dtoErr[int](asAppErr(err))
	}

	return &dto.Response[int]{Success: true, Data: id}
}

func (a *App) GetTopic(id int64) *dto.Response[*dto.Topic] {
	t, err := a.topicSvc.GetTopic(ctx.FastCtx(), id)
	if err != nil {
		return dtoErr[*dto.Topic](asAppErr(err))
	}

	return &dto.Response[*dto.Topic]{Success: true, Data: dto.FromTopic(t)}
}

func (a *App) ListTopics(limit, offset int) *dto.PaginatedResponse[*dto.Topic] {
	items, err := a.topicSvc.ListTopics(ctx.FastCtx())
	if err != nil {
		return dtoPagErr[*dto.Topic](asAppErr(err))
	}

	// Normalize pagination parameters
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	total := len(items)
	start := offset
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}

	paged := items[start:end]
	dtos := dto.FromTopics(paged)

	hasMore := end < total

	res := &dto.PaginatedResponse[*dto.Topic]{
		Success: true,
		Items:   dtos,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: hasMore,
	}

	return res
}

func (a *App) UpdateTopic(topic *dto.Topic) *dto.Response[string] {
	if topic == nil {
		return dtoErr[string](errors.Validation("topic payload is required"))
	}

	entity := &entities.Topic{
		ID:    topic.ID,
		Title: topic.Title,
	}

	if err := a.topicSvc.UpdateTopic(ctx.FastCtx(), entity); err != nil {
		return dtoErr[string](asAppErr(err))
	}

	return &dto.Response[string]{Success: true, Data: "updated"}
}

func (a *App) DeleteTopic(id int64) *dto.Response[string] {
	if err := a.topicSvc.DeleteTopic(ctx.FastCtx(), id); err != nil {
		return dtoErr[string](asAppErr(err))
	}

	return &dto.Response[string]{Success: true, Data: "deleted"}
}

func (a *App) AssignTopicToSite(siteID, topicID, categoryID int64, strategy string) *dto.Response[string] {
	strat := entities.TopicStrategy(strategy)
	if strat != entities.StrategyUnique && strat != entities.StrategyVariation {
		return dtoErr[string](errors.Validation("invalid topic strategy"))
	}

	if err := a.topicSvc.AssignToSite(ctx.FastCtx(), siteID, topicID, categoryID, strat); err != nil {
		return dtoErr[string](asAppErr(err))
	}

	return &dto.Response[string]{Success: true, Data: "assigned"}
}

func (a *App) UnassignTopicFromSite(siteID, topicID int64) *dto.Response[string] {
	if err := a.topicSvc.UnassignFromSite(ctx.FastCtx(), siteID, topicID); err != nil {
		return dtoErr[string](asAppErr(err))
	}

	return &dto.Response[string]{Success: true, Data: "unassigned"}
}

func (a *App) GetSiteTopics(siteID int64) *dto.Response[[]*dto.SiteTopic] {
	items, err := a.topicSvc.GetSiteTopics(ctx.FastCtx(), siteID)
	if err != nil {
		return dtoErr[[]*dto.SiteTopic](asAppErr(err))
	}

	return &dto.Response[[]*dto.SiteTopic]{Success: true, Data: dto.FromSiteTopics(items)}
}

func (a *App) GetTopicsBySite(siteID int64) *dto.Response[[]*dto.Topic] {
	items, err := a.topicSvc.GetTopicsBySite(ctx.FastCtx(), siteID)
	if err != nil {
		return dtoErr[[]*dto.Topic](asAppErr(err))
	}

	return &dto.Response[[]*dto.Topic]{Success: true, Data: dto.FromTopics(items)}
}

func (a *App) GetAvailableTopic(siteID int64, strategy string) *dto.Response[*dto.Topic] {
	strat := entities.TopicStrategy(strategy)
	if strat != entities.StrategyUnique && strat != entities.StrategyVariation {
		return dtoErr[*dto.Topic](errors.Validation("invalid topic strategy"))
	}

	t, err := a.topicSvc.GetAvailableTopic(ctx.FastCtx(), siteID, strat)
	if err != nil {
		return dtoErr[*dto.Topic](asAppErr(err))
	}

	return &dto.Response[*dto.Topic]{Success: true, Data: dto.FromTopic(t)}
}

func (a *App) MarkTopicAsUsed(siteID, topicID int64) *dto.Response[string] {
	if err := a.topicSvc.MarkTopicAsUsed(ctx.FastCtx(), siteID, topicID); err != nil {
		return dtoErr[string](asAppErr(err))
	}

	return &dto.Response[string]{Success: true, Data: "marked"}
}

func (a *App) CountUnusedTopics(siteID int64) *dto.Response[int] {
	cnt, err := a.topicSvc.CountUnusedTopics(ctx.FastCtx(), siteID)
	if err != nil {
		return dtoErr[int](asAppErr(err))
	}

	return &dto.Response[int]{Success: true, Data: cnt}
}
