package topic

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"
)

var _ Service = (*service)(nil)

type service struct {
	repo          Repository
	siteTopicRepo SiteTopicRepository
	usageRepo     UsageRepository
	logger        *logger.Logger
}

func NewService(
	repo Repository,
	siteTopicRepo SiteTopicRepository,
	usageRepo UsageRepository,
	logger *logger.Logger,
) Service {
	return &service{
		repo:          repo,
		siteTopicRepo: siteTopicRepo,
		usageRepo:     usageRepo,
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

func (s *service) CreateTopics(ctx context.Context, topics []*entities.Topic) (*entities.BatchResult, error) {
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

func (s *service) GenerateVariations(ctx context.Context, topicID int64, count int) ([]*entities.Topic, error) {
	//TODO: Implement
	if count <= 0 {
		return nil, errors.Validation("Count must be positive")
	}

	originalTopic, err := s.repo.GetByID(ctx, topicID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get original topic for variations")
		return nil, err
	}

	variations := make([]*entities.Topic, 0, count)
	for i := 0; i < count; i++ {
		variationTitle := s.generateVariation(originalTopic.Title, i)
		variation := &entities.Topic{
			Title:     variationTitle,
			CreatedAt: time.Now(),
		}
		variations = append(variations, variation)
	}

	result, err := s.repo.CreateBatch(ctx, variations...)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to create topic variations")
		return nil, err
	}

	createdVariations := make([]*entities.Topic, 0, result.Created)
	for i, topic := range variations {
		if i < result.Created {
			createdVariations = append(createdVariations, topic)
		}
	}

	s.logger.Info("Topic variations generated successfully")
	return createdVariations, nil
}

func (s *service) GetOrGenerateVariation(ctx context.Context, originalID int64) (*entities.Topic, error) {
	variations, err := s.GenerateVariations(ctx, originalID, 1)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to generate topic variation")
		return nil, err
	}

	if len(variations) == 0 {
		return nil, errors.Internal(fmt.Errorf("no variations generated"))
	}

	return variations[0], nil
}

func (s *service) GetNextTopicForJob(ctx context.Context, jobID int64) (*entities.Topic, error) {
	//TODO: Implement
	return nil, errors.New(errors.ErrCodeInternal, "Not implemented")
}

func (s *service) MarkTopicUsed(ctx context.Context, siteID, topicID int64) error {
	if err := s.usageRepo.MarkAsUsed(ctx, siteID, topicID); err != nil {
		s.logger.ErrorWithErr(err, "Failed to mark topic as used")
		return err
	}

	s.logger.Debug("Topic marked as used")
	return nil
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

func (s *service) generateVariation(original string, index int) string {
	//TODO: Implement
	variations := []string{
		original + " - In Depth Analysis",
		original + " - Complete Guide",
		original + " - Expert Insights",
		original + " - Ultimate Guide",
		original + " - Comprehensive Overview",
		original + " - Detailed Explanation",
		original + " - Practical Guide",
		original + " - Step by Step",
	}

	if index < len(variations) {
		return variations[index]
	}

	return original + " - Variation " + string(rune('A'+index))
}
