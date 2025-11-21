package tracking

import (
	"time"

	"github.com/davidmovas/postulator/internal/domain/categories"
	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/commands"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/fault"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/pipeline"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/pipevents"
	"github.com/davidmovas/postulator/internal/infra/events"
)

var _ pipeline.Command = (*RecordCategoryStatsCommand)(nil)

type RecordCategoryStatsCommand struct {
	*commands.BaseCommand
	categoryService categories.Service
}

func NewRecordCategoryStatsCommand(categoryService categories.Service) *RecordCategoryStatsCommand {
	return &RecordCategoryStatsCommand{
		BaseCommand:     commands.NewBaseCommand("record_category_stats", pipeline.StatePublished, pipeline.StateRecordingStats),
		categoryService: categoryService,
	}
}

func (c *RecordCategoryStatsCommand) Execute(ctx *pipeline.Context) error {
	if !ctx.HasPublication() || ctx.Publication.Article == nil {
		return fault.NewFatalError(fault.ErrCodeInvalidJob, c.Name(), "article not published")
	}

	if !ctx.HasSelection() || ctx.Selection.Category == nil {
		return fault.NewFatalError(fault.ErrCodeInvalidJob, c.Name(), "category not selected")
	}

	wordCount := 0
	if ctx.Publication.Article.WordCount != nil {
		wordCount = *ctx.Publication.Article.WordCount
	}

	err := c.categoryService.IncrementUsage(
		ctx.Context(),
		ctx.Job.SiteID,
		ctx.Selection.Category.ID,
		time.Now(),
		1,
		wordCount,
	)

	if err != nil {
		ctx.Logger().Errorf("Failed to record category usage stats: %v", err)
		return nil
	}

	if ctx.Context() != nil {
		events.Publish(ctx.Context(), events.NewEvent(
			pipevents.EventStatsRecorded,
			&pipevents.StatsRecordedEvent{
				JobID:      ctx.Job.ID,
				SiteID:     ctx.Job.SiteID,
				CategoryID: ctx.Selection.Category.ID,
				WordCount:  wordCount,
			},
		))
	}

	return nil
}

func (c *RecordCategoryStatsCommand) CanExecute(ctx *pipeline.Context) bool {
	return ctx.HasPublication() &&
		ctx.Publication.Article != nil &&
		ctx.Publication.Article.Status == entities.StatusPublished
}
