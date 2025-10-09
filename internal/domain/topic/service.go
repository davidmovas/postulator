package topic

import (
	"Postulator/internal/domain/entities"
	"Postulator/pkg/di"
	"Postulator/pkg/errors"
	"Postulator/pkg/logger"
	"context"
)

var _ IService = (*Service)(nil)

type Service struct {
	topicRepo     ITopicRepository
	siteTopicRepo ISiteTopicRepository
	usedTopicRepo IUsedTopicRepository
	logger        *logger.Logger
}

func NewService(c di.Container) (*Service, error) {
	var l *logger.Logger
	if err := c.Resolve(&l); err != nil {
		return nil, err
	}

	topicRepo, err := NewTopicRepository(c)
	if err != nil {
		return nil, err
	}

	siteTopicRepo, err := NewSiteTopicRepository(c)
	if err != nil {
		return nil, err
	}

	usedTopicRepo, err := NewUsedTopicRepository(c)
	if err != nil {
		return nil, err
	}

	return &Service{
		topicRepo:     topicRepo,
		siteTopicRepo: siteTopicRepo,
		usedTopicRepo: usedTopicRepo,
		logger:        l,
	}, nil
}

func (s *Service) CreateTopic(ctx context.Context, topic *entities.Topic) error {
	return s.topicRepo.Create(ctx, topic)
}

func (s *Service) CreateTopicBatch(ctx context.Context, topics []*entities.Topic) (*BatchCreateResult, error) {
	result, err := s.topicRepo.CreateBatch(ctx, topics)
	if err != nil {
		return nil, err
	}

	if result.TotalSkipped > 0 {
		s.logger.Infof("Batch topic creation completed: %d added, %d skipped (duplicates)", result.TotalAdded, result.TotalSkipped)
	}

	return result, nil
}

func (s *Service) GetTopic(ctx context.Context, id int64) (*entities.Topic, error) {
	return s.topicRepo.GetByID(ctx, id)
}

func (s *Service) ListTopics(ctx context.Context) ([]*entities.Topic, error) {
	return s.topicRepo.GetAll(ctx)
}

func (s *Service) UpdateTopic(ctx context.Context, topic *entities.Topic) error {
	return s.topicRepo.Update(ctx, topic)
}

func (s *Service) DeleteTopic(ctx context.Context, id int64) error {
	return s.topicRepo.Delete(ctx, id)
}

func (s *Service) AssignToSite(ctx context.Context, siteID, topicID, categoryID int64, strategy entities.TopicStrategy) error {
	if strategy == entities.StrategyUnique {
		isUsed, err := s.usedTopicRepo.IsUsed(ctx, siteID, topicID)
		if err != nil {
			return err
		}
		if isUsed {
			return errors.Validation("cannot assign already used topic to site with unique strategy")
		}
	}

	siteTopic := &entities.SiteTopic{
		SiteID:     siteID,
		TopicID:    topicID,
		CategoryID: categoryID,
		Strategy:   strategy,
	}

	return s.siteTopicRepo.Assign(ctx, siteTopic)
}

func (s *Service) UnassignFromSite(ctx context.Context, siteID, topicID int64) error {
	return s.siteTopicRepo.Unassign(ctx, siteID, topicID)
}

func (s *Service) GetSiteTopics(ctx context.Context, siteID int64) ([]*entities.SiteTopic, error) {
	return s.siteTopicRepo.GetBySiteID(ctx, siteID)
}

func (s *Service) GetTopicsBySite(ctx context.Context, siteID int64) ([]*entities.Topic, error) {
	siteTopics, err := s.siteTopicRepo.GetBySiteID(ctx, siteID)
	if err != nil {
		return nil, err
	}

	var topics []*entities.Topic
	for _, st := range siteTopics {
		var topic *entities.Topic
		topic, err = s.topicRepo.GetByID(ctx, st.TopicID)
		if err != nil {
			s.logger.Errorf("failed to get topic %d: %v", st.TopicID, err)
			continue
		}
		topics = append(topics, topic)
	}

	return topics, nil
}

func (s *Service) GetAvailableTopic(ctx context.Context, siteID int64, strategy entities.TopicStrategy) (*entities.Topic, error) {
	switch strategy {
	case entities.StrategyUnique:
		return s.getUniqueTopic(ctx, siteID)
	case entities.StrategyVariation:
		return s.getTopicForReuse(ctx, siteID)
	default:
		return nil, errors.Validation("invalid topic strategy: " + string(strategy))
	}
}

func (s *Service) getUniqueTopic(ctx context.Context, siteID int64) (*entities.Topic, error) {
	unusedTopics, err := s.usedTopicRepo.GetUnusedTopics(ctx, siteID)
	if err != nil {
		return nil, err
	}

	if len(unusedTopics) == 0 {
		return nil, errors.NotFound("unused topic for site", siteID)
	}

	// Return the oldest unused topic (first in the ordered list)
	return unusedTopics[0], nil
}

func (s *Service) getTopicForReuse(ctx context.Context, siteID int64) (*entities.Topic, error) {
	siteTopics, err := s.siteTopicRepo.GetBySiteID(ctx, siteID)
	if err != nil {
		return nil, err
	}

	if len(siteTopics) == 0 {
		return nil, errors.NotFound("topic for site", siteID)
	}

	// Pick the first topic from assigned topics
	st := siteTopics[0]

	topic, err := s.topicRepo.GetByID(ctx, st.TopicID)
	if err != nil {
		return nil, err
	}

	return topic, nil
}

func (s *Service) MarkTopicAsUsed(ctx context.Context, siteID, topicID int64) error {
	return s.usedTopicRepo.MarkAsUsed(ctx, siteID, topicID)
}

func (s *Service) CountUnusedTopics(ctx context.Context, siteID int64) (int, error) {
	return s.usedTopicRepo.CountUnusedTopics(ctx, siteID)
}
