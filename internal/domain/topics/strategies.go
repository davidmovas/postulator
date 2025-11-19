package topics

import (
	"context"
	"math/rand/v2"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"
)

// UniqueStrategy implements unique usage of topics
type uniqueStrategy struct {
	usageRepo     UsageRepository
	siteTopicRepo SiteTopicRepository
	logger        *logger.Logger
}

func (s *uniqueStrategy) CanExecute(ctx context.Context, job *entities.Job) error {
	if len(job.Topics) == 0 {
		return errors.JobExecution(job.ID, errors.NoResources("topics"))
	}

	count, err := s.usageRepo.CountUnused(ctx, job.SiteID, job.Topics)
	if err != nil {
		return errors.JobExecution(job.ID, err)
	}

	if count == 0 {
		return errors.JobExecution(job.ID, errors.NoResources("topics"))
	}

	return nil
}

func (s *uniqueStrategy) PickTopic(ctx context.Context, job *entities.Job) (*entities.Topic, error) {
	topic, err := s.usageRepo.GetNextUnused(ctx, job.SiteID, job.Topics)
	if err != nil {
		return nil, err
	}
	return topic, nil
}

func (s *uniqueStrategy) OnExecutionSuccess(ctx context.Context, job *entities.Job, topic *entities.Topic) error {
	if topic == nil {
		return nil
	}

	if err := s.usageRepo.MarkAsUsed(ctx, job.SiteID, topic.ID); err != nil {
		return err
	}

	return nil
}

func (s *uniqueStrategy) GetSelectableTopics(ctx context.Context, siteID int64) ([]*entities.Topic, error) {
	assigned, err := s.siteTopicRepo.GetBySiteID(ctx, siteID)
	if err != nil {
		return nil, err
	}

	ids := getTopicIDs(assigned)

	unused, err := s.usageRepo.GetUnused(ctx, siteID, ids)
	if err != nil {
		return nil, err
	}

	return unused, nil
}

func (s *uniqueStrategy) GetRemainingTopics(ctx context.Context, job *entities.Job) ([]*entities.Topic, int, error) {
	if len(job.Topics) == 0 {
		return []*entities.Topic{}, 0, nil
	}

	unused, err := s.usageRepo.GetUnused(ctx, job.SiteID, job.Topics)
	if err != nil {
		return nil, 0, err
	}

	return unused, len(unused), nil
}

// VariationStrategy implements reuse with variation
type variationStrategy struct {
	svc           *service
	siteTopicRepo SiteTopicRepository
	logger        *logger.Logger
}

func (s *variationStrategy) CanExecute(ctx context.Context, job *entities.Job) error {
	assigned, err := s.siteTopicRepo.GetBySiteID(ctx, job.SiteID)
	if err != nil {
		return errors.JobExecution(job.ID, err)
	}

	if len(assigned) == 0 {
		return errors.JobExecution(job.ID, errors.NoResources("topics"))
	}

	return nil
}

func (s *variationStrategy) PickTopic(ctx context.Context, job *entities.Job) (*entities.Topic, error) {
	assigned, err := s.siteTopicRepo.GetBySiteID(ctx, job.SiteID)
	if err != nil {
		return nil, err
	}

	if len(assigned) == 0 {
		return nil, errors.NoResources("topics")
	}

	original := assigned[rand.IntN(len(assigned))]
	variation, err := s.svc.GetOrGenerateVariation(ctx, job.AIProviderID, job.SiteID, original.ID)
	if err != nil {
		return nil, err
	}

	return variation, nil
}

func (s *variationStrategy) OnExecutionSuccess(_ context.Context, _ *entities.Job, _ *entities.Topic) error {
	return nil
}

func (s *variationStrategy) GetSelectableTopics(ctx context.Context, siteID int64) ([]*entities.Topic, error) {
	return s.siteTopicRepo.GetBySiteID(ctx, siteID)
}

func (s *variationStrategy) GetRemainingTopics(ctx context.Context, job *entities.Job) ([]*entities.Topic, int, error) {
	if len(job.Topics) == 0 {
		return []*entities.Topic{}, 0, nil
	}

	var result []*entities.Topic
	for _, id := range job.Topics {
		topic, err := s.svc.repo.GetByID(ctx, id)
		if err != nil {
			return nil, 0, err
		}
		if topic != nil {
			result = append(result, topic)
		}
	}

	return result, len(result), nil
}
