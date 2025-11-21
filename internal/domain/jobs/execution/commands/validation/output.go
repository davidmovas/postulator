package validation

import (
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/commands"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/fault"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/pipeline"
)

var _ pipeline.Command = (*ValidateOutputCommand)(nil)

type ValidateOutputCommand struct {
	*commands.BaseCommand
}

func NewValidateOutputCommand() *ValidateOutputCommand {
	return &ValidateOutputCommand{
		BaseCommand: commands.NewBaseCommand("validate_output", pipeline.StateGenerated, pipeline.StateOutputValidated),
	}
}

func (c *ValidateOutputCommand) Execute(ctx *pipeline.Context) error {
	if !ctx.HasGeneration() {
		return fault.NewFatalError(fault.ErrCodeInvalidJob, c.Name(), "content not generated")
	}

	if ctx.Generation.GeneratedTitle == "" {
		return fault.NewValidationError(fault.ErrCodeEmptyContent, c.Name(), "generated title is empty")
	}

	if ctx.Generation.GeneratedContent == "" {
		return fault.NewValidationError(fault.ErrCodeEmptyContent, c.Name(), "generated content is empty")
	}

	return nil
}
