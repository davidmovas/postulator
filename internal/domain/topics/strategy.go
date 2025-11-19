package topics

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/entities"
)

type TopicStrategyHandler interface {
    CanExecute(ctx context.Context, job *entities.Job) error
    PickTopic(ctx context.Context, job *entities.Job) (*entities.Topic, error)
    OnExecutionSuccess(ctx context.Context, job *entities.Job, topic *entities.Topic) error
    GetSelectableTopics(ctx context.Context, siteID int64) ([]*entities.Topic, error)
    GetRemainingTopics(ctx context.Context, job *entities.Job) ([]*entities.Topic, int, error)
}
