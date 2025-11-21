package tracking

import (
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/commands"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/fault"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/pipeline"
	"github.com/davidmovas/postulator/internal/domain/prompts"
	"github.com/davidmovas/postulator/internal/domain/providers"
)

var _ pipeline.Command = (*CreateExecutionCommand)(nil)

type CreateExecutionCommand struct {
	*commands.BaseCommand
	providersService providers.Service
	promptsService   prompts.Service
	execRepo         execution.Repository
}

func NewCreateExecutionCommand(
	execRepo execution.Repository,
	providerService providers.Service,
	promptService prompts.Service,
) *CreateExecutionCommand {
	return &CreateExecutionCommand{
		BaseCommand:      commands.NewBaseCommand("create_execution", pipeline.StateCategorySelected, pipeline.StateExecutionCreated),
		execRepo:         execRepo,
		providersService: providerService,
		promptsService:   promptService,
	}
}

func (c *CreateExecutionCommand) Execute(ctx *pipeline.Context) error {
	if !ctx.HasSelection() {
		return fault.NewFatalError(fault.ErrCodeInvalidJob, c.Name(), "topic or category not selected")
	}

	provider, err := c.providersService.GetProvider(ctx.Context(), ctx.Job.AIProviderID)
	if err != nil {
		return fault.WrapError(err, fault.ErrCodeNoProvider, c.Name(), "failed to get AI provider")
	}

	prompt, err := c.promptsService.GetPrompt(ctx.Context(), ctx.Job.PromptID)
	if err != nil {
		return fault.WrapError(err, fault.ErrCodeRecordNotFound, c.Name(), "failed to get prompt")
	}

	var categoryIDs []int64
	for _, cat := range ctx.Selection.Categories {
		categoryIDs = append(categoryIDs, cat.ID)
	}

	exec := &entities.Execution{
		JobID:        ctx.Job.ID,
		SiteID:       ctx.Job.SiteID,
		TopicID:      ctx.Selection.VariationTopic.ID,
		PromptID:     ctx.Job.PromptID,
		AIProviderID: ctx.Job.AIProviderID,
		AIModel:      provider.Model,
		CategoryIDs:  categoryIDs,
		Status:       entities.ExecutionStatusPending,
		StartedAt:    time.Now(),
	}

	if err = c.execRepo.Create(ctx.Context(), exec); err != nil {
		return fault.WrapError(err, fault.ErrCodeDatabaseError, c.Name(), "failed to create execution record")
	}

	ctx.InitExecutionPhase(exec, prompt, provider)

	return nil
}
