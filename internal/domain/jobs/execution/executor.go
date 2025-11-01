package execution

import (
	"context"
	"github.com/davidmovas/postulator/internal/domain/jobs"
)

type Executor struct{}

func (e *Executor) Execute(ctx context.Context, job *jobs.Job) error {
	return nil
}

func (e *Executor) PublishValidatedArticle(ctx context.Context, exec *Execution) error {
	return nil
}
