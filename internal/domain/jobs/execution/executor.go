package execution

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/articles"
	"github.com/davidmovas/postulator/internal/domain/categories"
	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/jobs"
	"github.com/davidmovas/postulator/internal/domain/prompts"
	"github.com/davidmovas/postulator/internal/domain/providers"
	"github.com/davidmovas/postulator/internal/domain/sites"
	"github.com/davidmovas/postulator/internal/domain/topics"
	"github.com/davidmovas/postulator/internal/infra/wp"
	"github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"
)

type Executor struct {
	execRepo    Repository
	articleRepo articles.Repository
	stateRepo   jobs.StateRepository

	jobService      jobs.Service
	topicService    topics.Service
	promptService   prompts.Service
	siteService     sites.Service
	providerService providers.Service
	categoryService categories.Service

	wpClient wp.Client

	logger *logger.Logger
}

func NewExecutor(
	execRepo Repository,
	articleRepo articles.Repository,
	stateRepo jobs.StateRepository,
	jobService jobs.Service,
	topicService topics.Service,
	promptService prompts.Service,
	siteService sites.Service,
	providerService providers.Service,
	categoryService categories.Service,
	wpClient wp.Client,
	logger *logger.Logger,
) jobs.Executor {
	return &Executor{
		execRepo:        execRepo,
		articleRepo:     articleRepo,
		stateRepo:       stateRepo,
		jobService:      jobService,
		topicService:    topicService,
		promptService:   promptService,
		siteService:     siteService,
		providerService: providerService,
		categoryService: categoryService,
		wpClient:        wpClient,
		logger:          logger.WithScope("executor"),
	}
}

func (e *Executor) Execute(ctx context.Context, job *entities.Job) error {
	e.logger.Infof("Starting execution of job %d (%s)", job.ID, job.Name)

	if err := e.executePipeline(ctx, job); err != nil {
		e.logger.Errorf("Job %d execution failed: %v", job.ID, err)
		return err
	}

	e.logger.Infof("Job %d execution completed", job.ID)
	return nil
}

func (e *Executor) PublishValidatedArticle(ctx context.Context, exec *entities.Execution) error {
	if exec.Status != entities.ExecutionStatusValidated {
		return errors.Validation("execution is not validated")
	}

	if exec.ArticleID == nil {
		return errors.Validation("execution has no associated article")
	}

	return e.publishValidatedArticle(ctx, exec)
}
