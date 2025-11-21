package execution

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/articles"
	"github.com/davidmovas/postulator/internal/domain/categories"
	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/jobs"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/phase"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/pipeline"
	"github.com/davidmovas/postulator/internal/domain/prompts"
	"github.com/davidmovas/postulator/internal/domain/providers"
	"github.com/davidmovas/postulator/internal/domain/sites"
	"github.com/davidmovas/postulator/internal/domain/stats"
	"github.com/davidmovas/postulator/internal/domain/topics"
	"github.com/davidmovas/postulator/internal/infra/events"
	"github.com/davidmovas/postulator/internal/infra/wp"
	"github.com/davidmovas/postulator/pkg/logger"
)

type Executor struct {
	pipeline *pipeline.Pipeline
	logger   *logger.Logger
}

func NewExecutor(
	execRepo Repository,
	articleRepo articles.Repository,
	stateRepo jobs.StateRepository,
	jobRepo jobs.Repository,
	topicService topics.Service,
	promptService prompts.Service,
	siteService sites.Service,
	statsRecorder stats.Recorder,
	providerService providers.Service,
	categoryService categories.Service,
	wpClient wp.Client,
	logger *logger.Logger,
) jobs.Executor {
	builder := pipeline.NewPipelineBuilder().
		WithLogger(logger).
		WithEventBus(events.GetGlobalEventBus()).
		AddCommands(
			phase.ValidateJobCommand(siteService, topicService, providerService),
			phase.SelectTopicCommand(),
			phase.SelectCategoryCommand(categoryService, stateRepo),
			phase.CreateExecutionCommand(execRepo, providerService, promptService),
			phase.RenderPromptCommand(promptService),
			phase.GenerateContentCommand(execRepo, statsRecorder),
			phase.ValidateOutputCommand(),
			phase.PublishArticleCommand(execRepo, articleRepo, wpClient, statsRecorder),
			phase.RecordCategoryStatsCommand(categoryService),
			phase.MarkTopicUsedCommand(),
			phase.CompleteExecutionCommand(execRepo, jobRepo, statsRecorder),
		)

	return &Executor{
		pipeline: builder.Build(),
		logger:   logger.WithScope("executor"),
	}
}

func (e *Executor) Execute(ctx context.Context, job *entities.Job) error {
	e.logger.Infof("Starting new pipeline execution for job %d (%s)", job.ID, job.Name)
	return e.pipeline.Execute(ctx, job)
}
