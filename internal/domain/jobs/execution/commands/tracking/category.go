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
		BaseCommand: commands.NewBaseCommand(
			"record_category_stats",
			pipeline.StatePublished,
			pipeline.StateRecordingStats,
		),
		categoryService: categoryService,
	}
}

func (c *RecordCategoryStatsCommand) Execute(ctx *pipeline.Context) error {
	if !ctx.HasPublication() || ctx.Publication.Article == nil {
		return fault.NewFatalError(fault.ErrCodeInvalidJob, c.Name(), "article not published")
	}

	if !ctx.HasSelection() || ctx.Selection.Categories == nil {
		return fault.NewFatalError(fault.ErrCodeInvalidJob, c.Name(), "category not selected")
	}

	wordCount := 0
	if ctx.Publication.Article.WordCount != nil {
		wordCount = *ctx.Publication.Article.WordCount
	}

	for _, category := range ctx.Selection.Categories {
		err := c.categoryService.IncrementUsage(
			ctx.Context(),
			ctx.Job.SiteID,
			category.ID,
			time.Now(),
			1,
			wordCount,
		)
		if err != nil {
			ctx.Logger().Errorf("Failed to record category usage stats: %v", err)
			return nil
		}
	}

	var categoryIDs []int64
	for _, category := range ctx.Selection.Categories {
		categoryIDs = append(categoryIDs, category.ID)
	}

	events.Publish(ctx.Context(), events.NewEvent(
		pipevents.EventStatsRecorded,
		&pipevents.StatsRecordedEvent{
			JobID:       ctx.Job.ID,
			SiteID:      ctx.Job.SiteID,
			CategoryIDs: categoryIDs,
			WordCount:   wordCount,
		},
	))

	return nil
}

func (c *RecordCategoryStatsCommand) CanExecute(ctx *pipeline.Context) bool {
	return ctx.HasPublication() &&
		ctx.Publication.Article != nil &&
		ctx.Publication.Article.Status == entities.StatusPublished
}
