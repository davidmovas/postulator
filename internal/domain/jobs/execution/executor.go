package execution

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/entities"
)

type Executor struct{}

func (e *Executor) Execute(ctx context.Context, job *entities.Job) error {
	return nil
}

func (e *Executor) PublishValidatedArticle(ctx context.Context, exec *entities.Execution) error {
	return nil
}
