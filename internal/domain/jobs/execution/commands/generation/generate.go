package generation

import (
	"time"

	"github.com/davidmovas/postulator/internal/domain/aiusage"
	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/commands"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/fault"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/pipeline"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/pipevents"
	"github.com/davidmovas/postulator/internal/domain/stats"
	"github.com/davidmovas/postulator/internal/infra/ai"
	"github.com/davidmovas/postulator/internal/infra/events"
)

var _ pipeline.Command = (*GenerateContentCommand)(nil)

type GenerateContentCommand struct {
	*commands.BaseCommand
	executionProvider commands.ExecutionProvider
	statsRecorder     stats.Recorder
	aiUsageService    aiusage.Service
}

func NewGenerateContentCommand(executionProvider commands.ExecutionProvider, statsRecorder stats.Recorder, aiUsageService aiusage.Service) *GenerateContentCommand {
	return &GenerateContentCommand{
		BaseCommand: commands.NewBaseCommand(
			"generate_content",
			pipeline.StatePromptRendered,
			pipeline.StateGenerated,
		).WithRetry(3),
		executionProvider: executionProvider,
		statsRecorder:     statsRecorder,
		aiUsageService:    aiUsageService,
	}
}

func (c *GenerateContentCommand) Execute(ctx *pipeline.Context) error {
	if !ctx.HasExecution() {
		return fault.NewFatalError(fault.ErrCodeInvalidJob, c.Name(), "execution not created")
	}

	if !ctx.HasGeneration() || ctx.Generation.SystemPrompt == "" || ctx.Generation.UserPrompt == "" {
		return fault.NewFatalError(fault.ErrCodeInvalidJob, c.Name(), "prompts not rendered")
	}

	ctx.Execution.Execution.Status = entities.ExecutionStatusGenerating
	if err := c.executionProvider.Update(ctx.Context(), ctx.Execution.Execution); err != nil {
		_ = c.statsRecorder.RecordArticleFailed(ctx.Context(), ctx.Job.SiteID)
		return fault.WrapError(err, fault.ErrCodeDatabaseError, c.Name(), "failed to update execution status")
	}

	aiClient, err := ai.CreateClient(ctx.Execution.Provider)
	if err != nil {
		_ = c.statsRecorder.RecordArticleFailed(ctx.Context(), ctx.Job.SiteID)
		return fault.WrapError(err, fault.ErrCodeNoProvider, c.Name(), "failed to create AI client")
	}

	startTime := time.Now()

	result, err := aiClient.GenerateArticle(ctx.Context(), ctx.Generation.SystemPrompt, ctx.Generation.UserPrompt)
	durationMs := time.Since(startTime).Milliseconds()

	// Log AI usage regardless of success/failure
	if c.aiUsageService != nil {
		var usage ai.Usage
		if result != nil {
			usage = result.Usage
		}
		_ = c.aiUsageService.LogFromResult(
			ctx.Context(),
			ctx.Job.SiteID,
			aiusage.OperationArticleGeneration,
			aiClient,
			usage,
			durationMs,
			err,
			map[string]interface{}{
				"job_id":       ctx.Job.ID,
				"execution_id": ctx.Execution.Execution.ID,
			},
		)
	}

	if err != nil {
		_ = c.statsRecorder.RecordArticleFailed(ctx.Context(), ctx.Job.SiteID)
		return fault.WrapError(err, fault.ErrCodeAIGenerationFailed, c.Name(), "AI generation failed")
	}

	generationTime := int(durationMs)

	ctx.Generation.GeneratedTitle = result.Title
	ctx.Generation.GeneratedExcerpt = result.Excerpt
	ctx.Generation.GeneratedContent = result.Content
	ctx.Generation.GenerationTimeMs = generationTime

	if result.TokensUsed > 0 {
		ctx.Generation.TokensUsed = result.TokensUsed
	}
	if result.Cost > 0 {
		ctx.Generation.CostUSD = result.Cost
	}

	now := time.Now()
	ctx.Execution.Execution.GeneratedAt = &now
	ctx.Execution.Execution.GenerationTimeMs = &generationTime

	if result.TokensUsed > 0 {
		ctx.Execution.Execution.TokensUsed = &result.TokensUsed
	}
	if result.Cost > 0 {
		ctx.Execution.Execution.CostUSD = &result.Cost
	}

	if err = c.executionProvider.Update(ctx.Context(), ctx.Execution.Execution); err != nil {
		_ = c.statsRecorder.RecordArticleFailed(ctx.Context(), ctx.Job.SiteID)
		return fault.WrapError(err, fault.ErrCodeDatabaseError, c.Name(), "failed to update execution with generated content")
	}

	events.Publish(ctx.Context(), events.NewEvent(
		pipevents.EventGenerationCompleted,
		&pipevents.GenerationCompletedEvent{
			JobID:          ctx.Job.ID,
			ExecutionID:    ctx.Execution.Execution.ID,
			Title:          result.Title,
			ContentLength:  len(result.Content),
			GenerationTime: time.Since(startTime),
			TokensUsed:     result.TokensUsed,
			CostUSD:        result.Cost,
		},
	))

	return nil
}
