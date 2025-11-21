package tracking

import (
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/jobs"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/commands"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/fault"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/pipeline"
	"github.com/davidmovas/postulator/internal/domain/stats"
	"github.com/davidmovas/postulator/pkg/errors"
)

type CompleteExecutionCommand struct {
	*commands.BaseCommand
	execRepo      execution.Repository
	jobRepo       jobs.Repository
	statsRecorder stats.Recorder
}

func NewCompleteExecutionCommand(
	execRepo execution.Repository,
	jobRepo jobs.Repository,
	statsRecorder stats.Recorder,
) *CompleteExecutionCommand {
	return &CompleteExecutionCommand{
		BaseCommand: commands.NewBaseCommand(
			"complete_execution",
			pipeline.StateMarkingUsed,
			pipeline.StateCompleted,
		),
		execRepo:      execRepo,
		jobRepo:       jobRepo,
		statsRecorder: statsRecorder,
	}
}

func (c *CompleteExecutionCommand) Execute(ctx *pipeline.Context) error {
	if !ctx.HasExecution() {
		return fault.NewFatalError(fault.ErrCodeInvalidJob, c.Name(), "execution not created")
	}

	now := time.Now()
	ctx.Execution.Execution.CompletedAt = &now

	if err := c.execRepo.Update(ctx.Context(), ctx.Execution.Execution); err != nil {
		return fault.WrapError(err, fault.ErrCodeDatabaseError, c.Name(), "failed to update execution completion time")
	}

	if ctx.Job.Schedule == nil || ctx.Job.Schedule.Type != entities.ScheduleManual {
		if err := c.checkPostRunResources(ctx); err != nil {
			return err
		}
	}

	if ctx.HasPublication() && ctx.Publication.Article.Status == entities.StatusPublished {
		_ = c.statsRecorder.RecordArticlePublished(ctx.Context(), ctx.Job.SiteID, len(ctx.Generation.GeneratedContent))
	}

	return nil
}

func (c *CompleteExecutionCommand) checkPostRunResources(ctx *pipeline.Context) error {
	strategy := ctx.GetStrategy()
	if strategy == nil {
		return nil
	}

	if err := strategy.CanExecute(ctx.Context(), ctx.Job); err != nil {
		if errors.IsNoResources(err) {
			if pauseErr := c.pauseJob(ctx); pauseErr != nil {
				return fault.WrapError(pauseErr, fault.ErrCodeUpdateFailed, c.Name(), "failed to pause job after resources exhausted")
			}
			return nil
		}
		return fault.WrapError(err, fault.ErrCodeInvalidStrategy, c.Name(), "post-run resource check failed")
	}

	return nil
}

func (c *CompleteExecutionCommand) pauseJob(ctx *pipeline.Context) error {
	job, err := c.jobRepo.GetByID(ctx.Context(), ctx.Job.ID)
	if err != nil {
		return err
	}
	if job.Status == entities.JobStatusPaused {
		return nil
	}
	job.Status = entities.JobStatusPaused
	job.UpdatedAt = time.Now()
	return c.jobRepo.Update(ctx.Context(), job)
}
