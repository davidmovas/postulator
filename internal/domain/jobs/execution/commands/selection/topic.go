package selection

import (
	errs "errors"

	"github.com/davidmovas/postulator/internal/domain/jobs/execution/commands"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/fault"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/pipeline"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/pipevents"
	"github.com/davidmovas/postulator/internal/infra/events"
)

var _ pipeline.Command = (*SelectTopicCommand)(nil)

type SelectTopicCommand struct {
	*commands.BaseCommand
}

func NewSelectTopicCommand() *SelectTopicCommand {
	return &SelectTopicCommand{
		BaseCommand: commands.NewBaseCommand(
			"select_topic",
			pipeline.StateValidated,
			pipeline.StateTopicSelected,
		),
	}
}

func (c *SelectTopicCommand) Execute(ctx *pipeline.Context) error {
	if !ctx.HasValidated() {
		return fault.NewFatalError(fault.ErrCodeInvalidJob, c.Name(), "job not validated")
	}

	strategy := ctx.GetStrategy()
	if strategy == nil {
		return fault.NewFatalError(fault.ErrCodeInvalidStrategy, c.Name(), "strategy not available")
	}

	originalTopic, variationTopic, err := strategy.PickTopic(ctx.Context(), ctx.Job)
	if err != nil {
		return fault.WrapError(err, fault.ErrCodeNoTopics, c.Name(), "failed to pick topic")
	}

	if ctx.HasSelection() {
		ctx.Selection.OriginalTopic = originalTopic
		ctx.Selection.VariationTopic = variationTopic
	} else {
		ctx.InitSelectionPhase(originalTopic, variationTopic, nil)
	}

	if ctx.Context() != nil {
		events.Publish(ctx.Context(), events.NewEvent(
			pipevents.EventTopicSelected,
			&pipevents.TopicSelectedEvent{
				JobID:            ctx.Job.ID,
				TopicID:          variationTopic.ID,
				TopicTitle:       variationTopic.Title,
				OriginalTopicID:  originalTopic.ID,
				VariationTopicID: variationTopic.ID,
			},
		))
	}

	return nil
}

func (c *SelectTopicCommand) OnError(_ *pipeline.Context, err error) error {
	var pErr *fault.PipelineError
	if errs.As(err, &pErr) {
		if pErr.Code == fault.ErrCodeNoTopics {
			return err
		}
	}
	return err
}
