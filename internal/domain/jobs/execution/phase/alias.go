package phase

import (
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/commands/execution"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/commands/generation"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/commands/publishing"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/commands/selection"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/commands/tracking"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/commands/validation"
)

var (
	ValidateJobCommand         = validation.NewValidateJobCommand
	ValidateOutputCommand      = validation.NewValidateOutputCommand
	SelectTopicCommand         = selection.NewSelectTopicCommand
	SelectCategoryCommand      = selection.NewSelectCategoryCommand
	CreateExecutionCommand     = execution.NewCreateExecutionCommand
	RenderPromptCommand        = generation.NewRenderPromptCommand
	GenerateContentCommand     = generation.NewGenerateContentCommand
	PublishArticleCommand      = publishing.NewPublishArticleCommand
	RecordCategoryStatsCommand = tracking.NewRecordCategoryStatsCommand
	MarkTopicUsedCommand       = tracking.NewMarkTopicUsedCommand
	CompleteExecutionCommand   = tracking.NewCompleteExecutionCommand
)
