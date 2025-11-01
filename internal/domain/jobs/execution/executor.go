package execution

import (
	"Postulator/internal/domain/jobs"
	"context"
)

type Executor struct{}

func (e *Executor) Execute(ctx context.Context, job *jobs.Job) error {
	return nil
}

func (e *Executor) PublishValidatedArticle(ctx context.Context, exec *Execution) error {
	return nil
}
