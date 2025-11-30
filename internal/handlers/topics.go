package handlers

import (
	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/jobs"
	"github.com/davidmovas/postulator/internal/domain/topics"
	"github.com/davidmovas/postulator/internal/dto"
	"github.com/davidmovas/postulator/internal/infra/importer"
	"github.com/davidmovas/postulator/pkg/ctx"
)

type TopicsHandler struct {
	service    topics.Service
	jobService jobs.Service
}

func NewTopicsHandler(service topics.Service, jobService jobs.Service) *TopicsHandler {
	return &TopicsHandler{
		service:    service,
		jobService: jobService,
	}
}

func (h *TopicsHandler) CreateTopic(topic *dto.Topic) *dto.Response[string] {
	entity, err := topic.ToEntity()
	if err != nil {
		return fail[string](err)
	}

	if err = h.service.CreateTopic(ctx.FastCtx(), entity); err != nil {
		return fail[string](err)
	}

	return ok("Topic created successfully")
}

func (h *TopicsHandler) CreateTopics(topics []*dto.Topic) *dto.Response[*dto.BatchResult] {
	tops := make([]*entities.Topic, len(topics))
	for i, topic := range topics {
		if entity, err := topic.ToEntity(); err != nil {
			return fail[*dto.BatchResult](err)
		} else {
			tops[i] = entity
		}
	}

	result, err := h.service.CreateTopics(ctx.FastCtx(), tops...)
	if err != nil {
		return fail[*dto.BatchResult](err)
	}

	return ok(dto.NewBatchResult(result))
}

func (h *TopicsHandler) CreateAndAssignToSite(siteID int64, topics []*dto.Topic) *dto.Response[*dto.ImportResult] {
	tops := make([]*entities.Topic, len(topics))
	for i, topic := range topics {
		if entity, err := topic.ToEntity(); err != nil {
			return fail[*dto.ImportResult](err)
		} else {
			tops[i] = entity
		}
	}

	result, err := h.service.CreateAndAssignToSite(ctx.FastCtx(), siteID, tops...)
	if err != nil {
		return fail[*dto.ImportResult](err)
	}

	res := &importer.ImportResult{
		TotalRead:    result.TotalProcessed,
		TotalAdded:   result.TotalAdded,
		TotalSkipped: result.TotalSkipped,
		Added:        result.Added,
		Skipped:      result.Skipped,
		Errors:       []string{},
	}

	return ok(dto.NewImportResult(res))
}

func (h *TopicsHandler) GetTopic(id int64) *dto.Response[*dto.Topic] {
	topic, err := h.service.GetTopic(ctx.FastCtx(), id)
	if err != nil {
		return fail[*dto.Topic](err)
	}

	return ok(dto.NewTopic(topic))
}

func (h *TopicsHandler) ListTopics() *dto.Response[[]*dto.Topic] {
	listTopics, err := h.service.ListTopics(ctx.FastCtx())
	if err != nil {
		return fail[[]*dto.Topic](err)
	}

	var dtoTopics []*dto.Topic
	for _, topic := range listTopics {
		dtoTopics = append(dtoTopics, dto.NewTopic(topic))
	}

	return ok(dtoTopics)
}

func (h *TopicsHandler) UpdateTopic(topic *dto.Topic) *dto.Response[string] {
	entity, err := topic.ToEntity()
	if err != nil {
		return fail[string](err)
	}

	if err = h.service.UpdateTopic(ctx.FastCtx(), entity); err != nil {
		return fail[string](err)
	}

	return ok("Topic updated successfully")
}

func (h *TopicsHandler) DeleteTopic(id int64) *dto.Response[string] {
	if err := h.service.DeleteTopic(ctx.FastCtx(), id); err != nil {
		return fail[string](err)
	}

	return ok("Topic deleted successfully")
}

func (h *TopicsHandler) AssignToSite(siteID int64, topicIDs []int64) *dto.Response[string] {
	if err := h.service.AssignToSite(ctx.FastCtx(), siteID, topicIDs...); err != nil {
		return fail[string](err)
	}

	return ok("Topics assigned successfully")
}

func (h *TopicsHandler) UnassignFromSite(siteID int64, topicIDs ...int64) *dto.Response[string] {
	if err := h.service.UnassignFromSite(ctx.FastCtx(), siteID, topicIDs...); err != nil {
		return fail[string](err)
	}

	return ok("Topics unassigned successfully")
}

func (h *TopicsHandler) GetSiteTopics(siteID int64) *dto.Response[[]*dto.Topic] {
	listTopics, err := h.service.GetSiteTopics(ctx.FastCtx(), siteID)
	if err != nil {
		return fail[[]*dto.Topic](err)
	}

	var dtoTopics []*dto.Topic
	for _, topic := range listTopics {
		dtoTopics = append(dtoTopics, dto.NewTopic(topic))
	}

	return ok(dtoTopics)
}

func (h *TopicsHandler) GetSelectableSiteTopics(siteID int64, strategy string) *dto.Response[[]*dto.Topic] {
	st := entities.StrategyUnique
	switch entities.TopicStrategy(strategy) {
	case entities.StrategyUnique, entities.StrategyVariation:
		st = entities.TopicStrategy(strategy)
	default:
	}

	listTopics, err := h.service.GetSelectableSiteTopics(ctx.FastCtx(), siteID, st)
	if err != nil {
		return fail[[]*dto.Topic](err)
	}

	var dtoTopics []*dto.Topic
	for _, topic := range listTopics {
		dtoTopics = append(dtoTopics, dto.NewTopic(topic))
	}

	return ok(dtoTopics)
}

func (h *TopicsHandler) GenerateVariations(providerID int64, topicID int64, count int) *dto.Response[[]*dto.Topic] {
	variations, err := h.service.GenerateVariations(ctx.FastCtx(), providerID, topicID, count)
	if err != nil {
		return fail[[]*dto.Topic](err)
	}

	var dtoTopics []*dto.Topic
	for _, topic := range variations {
		dtoTopics = append(dtoTopics, dto.NewTopic(topic))
	}

	return ok(dtoTopics)
}

func (h *TopicsHandler) GetOrGenerateVariation(providerID, siteID, originalID int64) *dto.Response[*dto.Topic] {
	variation, err := h.service.GetOrGenerateVariation(ctx.FastCtx(), providerID, siteID, originalID)
	if err != nil {
		return fail[*dto.Topic](err)
	}

	return ok(dto.NewTopic(variation))
}

func (h *TopicsHandler) GetNextTopicForJob(jobID int64) *dto.Response[*dto.Topic] {
	c := ctx.MediumCtx()

	job, err := h.jobService.GetJob(c, jobID)
	if err != nil {
		return fail[*dto.Topic](err)
	}

	topic, err := h.service.GetNextTopicForJob(c, job)
	if err != nil {
		return fail[*dto.Topic](err)
	}

	if topic == nil {
		return ok[*dto.Topic](nil)
	}

	return ok(dto.NewTopic(topic))
}

func (h *TopicsHandler) MarkTopicUsed(siteID, topicID int64) *dto.Response[string] {
	if err := h.service.MarkTopicUsed(ctx.FastCtx(), siteID, topicID); err != nil {
		return fail[string](err)
	}

	return ok("Topic marked as used successfully")
}

func (h *TopicsHandler) GetJobRemainingTopics(jobID int64) *dto.Response[*dto.JobTopicsStatus] {
	c := ctx.FastCtx()

	job, err := h.jobService.GetJob(c, jobID)
	if err != nil {
		return fail[*dto.JobTopicsStatus](err)
	}

	tops, count, err := h.service.GetJobRemainingTopics(c, job)
	if err != nil {
		return fail[*dto.JobTopicsStatus](err)
	}

	return ok(dto.NewJobTopicsStatus(tops, count))
}
