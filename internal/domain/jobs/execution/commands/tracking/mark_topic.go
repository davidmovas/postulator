package tracking

import (
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/commands"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/fault"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/pipeline"
)

var _ pipeline.Command = (*MarkTopicUsedCommand)(nil)

type MarkTopicUsedCommand struct {
	*commands.BaseCommand
}

func NewMarkTopicUsedCommand() *MarkTopicUsedCommand {
	return &MarkTopicUsedCommand{
		BaseCommand: commands.NewBaseCommand(
			"mark_topic_used",
			pipeline.StateRecordingStats,
			pipeline.StateMarkingUsed,
		),
	}
}

func (c *MarkTopicUsedCommand) Execute(ctx *pipeline.Context) error {
	if !ctx.HasValidated() || ctx.Validated.Strategy == nil {
		return nil
	}

	if !ctx.HasSelection() || ctx.Selection.VariationTopic == nil {
		return nil
	}

	if err := ctx.Validated.Strategy.OnExecutionSuccess(ctx.Context(), ctx.Job, ctx.Selection.VariationTopic); err != nil {
		return fault.WrapError(err, fault.ErrCodeUpdateFailed, c.Name(), "failed to mark topic as used")
	}

	return nil
}

func (c *MarkTopicUsedCommand) CanExecute(ctx *pipeline.Context) bool {
	return ctx.HasValidated() &&
		ctx.Validated.Strategy != nil &&
		ctx.HasSelection() &&
		ctx.Selection.VariationTopic != nil
}
