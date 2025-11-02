package topics

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/providers"
	"github.com/davidmovas/postulator/internal/infra/ai"
	"github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"
)

var _ Service = (*service)(nil)

type service struct {
	providerService providers.Service
	repo            Repository
	siteTopicRepo   SiteTopicRepository
	usageRepo       UsageRepository
	logger          *logger.Logger
}

func NewService(
	providerService providers.Service,
	repo Repository,
	siteTopicRepo SiteTopicRepository,
	usageRepo UsageRepository,
	logger *logger.Logger,
) Service {
	return &service{
		providerService: providerService,
		repo:            repo,
		siteTopicRepo:   siteTopicRepo,
		usageRepo:       usageRepo,
		logger: logger.
			WithScope("service").
			WithScope("topics"),
	}
}

func (s *service) CreateTopic(ctx context.Context, topic *entities.Topic) error {
	if err := s.validateTopic(topic); err != nil {
		return err
	}

	topic.CreatedAt = time.Now()
	createdTopic, err := s.repo.Create(ctx, topic)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to create topic")
		return err
	}

	*topic = *createdTopic
	s.logger.Info("Topic created successfully")
	return nil
}

func (s *service) CreateTopics(ctx context.Context, topics ...*entities.Topic) (*entities.BatchResult, error) {
	if len(topics) == 0 {
		return &entities.BatchResult{}, nil
	}

	for _, topic := range topics {
		if err := s.validateTopic(topic); err != nil {
			return nil, err
		}
	}

	result, err := s.repo.CreateBatch(ctx, topics...)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to create topics batch")
		return nil, err
	}

	s.logger.Info("Topics batch created successfully")
	return result, nil
}

func (s *service) GetTopic(ctx context.Context, id int64) (*entities.Topic, error) {
	topic, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get topic")
		return nil, err
	}

	s.logger.Debug("Topic retrieved")
	return topic, nil
}

func (s *service) ListTopics(ctx context.Context) ([]*entities.Topic, error) {
	topics, err := s.repo.GetAll(ctx)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to list topics")
		return nil, err
	}

	s.logger.Debug("Topics listed")
	return topics, nil
}

func (s *service) UpdateTopic(ctx context.Context, topic *entities.Topic) error {
	if err := s.validateTopic(topic); err != nil {
		return err
	}

	if err := s.repo.Update(ctx, topic); err != nil {
		s.logger.ErrorWithErr(err, "Failed to update topic")
		return err
	}

	s.logger.Info("Topic updated successfully")
	return nil
}

func (s *service) DeleteTopic(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.ErrorWithErr(err, "Failed to delete topic")
		return err
	}

	s.logger.Info("Topic deleted successfully")
	return nil
}

func (s *service) AssignToSite(ctx context.Context, siteID int64, topicIDs ...int64) error {
	if len(topicIDs) == 0 {
		return nil
	}

	var lastErr error
	assignedCount := 0

	for _, topicID := range topicIDs {
		if err := s.siteTopicRepo.Assign(ctx, siteID, topicID); err != nil {
			lastErr = err
			s.logger.ErrorWithErr(err, "Failed to assign topic to site")
		} else {
			assignedCount++
		}
	}

	if lastErr != nil {
		return errors.New(errors.ErrCodeInternal, "Some topic assignments failed")
	}

	s.logger.Info("Topics assigned to site successfully")
	return nil
}

func (s *service) UnassignFromSite(ctx context.Context, siteID int64, topicIDs ...int64) error {
	if len(topicIDs) == 0 {
		return nil
	}

	var lastErr error
	unassignedCount := 0

	for _, topicID := range topicIDs {
		if err := s.siteTopicRepo.Unassign(ctx, siteID, topicID); err != nil {
			lastErr = err
			s.logger.ErrorWithErr(err, "Failed to unassign topic from site")
		} else {
			unassignedCount++
		}
	}

	if lastErr != nil {
		return errors.New(errors.ErrCodeInternal, "Some topic unassignments failed")
	}

	s.logger.Info("Topics unassigned from site successfully")
	return nil
}

func (s *service) GetSiteTopics(ctx context.Context, siteID int64) ([]*entities.Topic, error) {
	topics, err := s.siteTopicRepo.GetBySiteID(ctx, siteID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get site topics")
		return nil, err
	}

	s.logger.Debug("Site topics retrieved")
	return topics, nil
}

func (s *service) GenerateVariations(ctx context.Context, providerID, topicID int64, count int) ([]*entities.Topic, error) {
	if count <= 0 {
		return nil, errors.Validation("Count must be positive")
	}

	reference, err := s.repo.GetByID(ctx, topicID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get original topic for variations")
		return nil, err
	}

	provider, err := s.providerService.GetProvider(ctx, providerID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get provider for variations")
		return nil, err
	}

	client, err := ai.CreateClient(provider)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to create AI client for variations")
		return nil, err
	}

	titles, err := client.GenerateTopicVariations(ctx, reference.Title, count)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to generate topic variations")
		return nil, err
	}

	var topics []*entities.Topic
	for _, title := range titles {
		topics = append(topics, &entities.Topic{
			Title:     title,
			CreatedAt: time.Now(),
		})
	}

	s.logger.Info("Topic variations generated successfully")
	return topics, nil
}

func (s *service) GetNextTopicForJob(ctx context.Context, job *entities.Job) (*entities.Topic, error) {
	switch job.TopicStrategy {
	case entities.StrategyUnique:
		return s.getNextUniqueTopic(ctx, job)
	case entities.StrategyVariation:
		return s.getNextVariationTopic(ctx, job)
	default:
		return nil, errors.Validation("Unknown topic strategy")
	}
}

func (s *service) MarkTopicUsed(ctx context.Context, siteID, topicID int64) error {
	if err := s.usageRepo.MarkAsUsed(ctx, siteID, topicID); err != nil {
		s.logger.ErrorWithErr(err, "Failed to mark topic as used")
		return err
	}

	s.logger.Debug("Topic marked as used")
	return nil
}

func (s *service) GetOrGenerateVariation(ctx context.Context, providerID, siteID, originalID int64) (*entities.Topic, error) {
	originalTopic, err := s.repo.GetByID(ctx, originalID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get original topic")
		return nil, err
	}

	provider, err := s.providerService.GetProvider(ctx, providerID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get provider for variations")
		return nil, err
	}

	client, err := ai.CreateClient(provider)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to create AI client for variations")
		return nil, err
	}

	allTopics, err := s.repo.GetAll(ctx)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get all topics")
		return nil, err
	}

	var variationTopics []*entities.Topic
	for _, topic := range allTopics {
		if strings.Contains(topic.Title, originalTopic.Title) && topic.ID != originalID {
			variationTopics = append(variationTopics, topic)
		}
	}

	if len(variationTopics) > 0 {
		var unusedVariations []*entities.Topic
		unusedVariations, err = s.usageRepo.GetUnused(ctx, siteID, getTopicIDs(variationTopics))
		if err != nil {
			s.logger.ErrorWithErr(err, "Failed to get unused variations")
			return nil, err
		}

		if len(unusedVariations) > 0 {
			return unusedVariations[0], nil
		}
	}

	newVariation, err := client.GenerateTopicVariations(ctx, originalTopic.Title, 1)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to generate topic variation")
		return nil, err
	}

	if len(newVariation) == 0 {
		return nil, errors.Internal(fmt.Errorf("no variation generated"))
	}

	variationTopic := &entities.Topic{
		Title:     newVariation[0],
		CreatedAt: time.Now(),
	}

	createdTopic, err := s.repo.Create(ctx, variationTopic)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to create variation topic")
		return nil, err
	}

	return createdTopic, nil
}

func (s *service) validateTopic(topic *entities.Topic) error {
	if strings.TrimSpace(topic.Title) == "" {
		return errors.Validation("Topic title is required")
	}

	if len(topic.Title) > 500 {
		return errors.Validation("Topic title is too long")
	}

	return nil
}

func (s *service) getNextUniqueTopic(ctx context.Context, job *entities.Job) (*entities.Topic, error) {
	unusedTopics, err := s.usageRepo.GetUnused(ctx, job.SiteID, job.Topics)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get unused topics")
		return nil, err
	}

	if len(unusedTopics) == 0 {
		return nil, errors.NotFound("unused_topic", "No unused topics available")
	}

	nextTopic := unusedTopics[0]
	return nextTopic, nil
}

func (s *service) getNextVariationTopic(ctx context.Context, job *entities.Job) (*entities.Topic, error) {
	unusedTopics, err := s.usageRepo.GetUnused(ctx, job.SiteID, job.Topics)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get unused topics")
		return nil, err
	}

	if len(unusedTopics) > 0 {
		return unusedTopics[0], nil
	}

	if len(job.Topics) == 0 {
		return nil, errors.NotFound("topic", "No topics available for job")
	}

	originalTopicID := job.Topics[rand.Intn(len(job.Topics))]
	variation, err := s.GetOrGenerateVariation(ctx, job.AIProviderID, job.SiteID, originalTopicID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to generate topic variation")
		return nil, err
	}

	if err = s.siteTopicRepo.Assign(ctx, job.SiteID, variation.ID); err != nil {
		s.logger.ErrorWithErr(err, "Failed to assign variation to site")
		return nil, err
	}

	return variation, nil
}

func getTopicIDs(topics []*entities.Topic) []int64 {
	ids := make([]int64, len(topics))
	for i, topic := range topics {
		ids[i] = topic.ID
	}
	return ids
}
