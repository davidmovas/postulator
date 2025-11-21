package validation

import (
	errs "errors"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/commands"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/fault"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/pipeline"
	"github.com/davidmovas/postulator/internal/domain/providers"
	"github.com/davidmovas/postulator/internal/domain/sites"
	"github.com/davidmovas/postulator/internal/domain/topics"
	"github.com/davidmovas/postulator/pkg/errors"
)

var _ pipeline.Command = (*ValidateJobCommand)(nil)

type ValidateJobCommand struct {
	*commands.BaseCommand
	siteService     sites.Service
	topicService    topics.Service
	providerService providers.Service
}

func NewValidateJobCommand(
	siteService sites.Service,
	topicService topics.Service,
	providerService providers.Service,
) *ValidateJobCommand {
	return &ValidateJobCommand{
		BaseCommand:     commands.NewBaseCommand("validate_job", pipeline.StateInitialized, pipeline.StateValidated),
		siteService:     siteService,
		topicService:    topicService,
		providerService: providerService,
	}
}

func (c *ValidateJobCommand) Execute(ctx *pipeline.Context) error {
	site, err := c.siteService.GetSiteWithPassword(ctx.Context(), ctx.Job.SiteID)
	if err != nil {
		return fault.WrapError(err, fault.ErrCodeRecordNotFound, c.Name(), "failed to get site")
	}

	if site.Status != entities.StatusActive {
		return fault.NewValidationError(fault.ErrCodeInactiveSite, c.Name(), "site is not active").
			WithContext("site_id", site.ID).
			WithContext("site_status", string(site.Status))
	}

	strategy, err := c.topicService.GetStrategy(ctx.Job.TopicStrategy)
	if err != nil {
		return fault.WrapError(err, fault.ErrCodeInvalidStrategy, c.Name(), "failed to get topic strategy")
	}

	if _, err = c.providerService.GetProvider(ctx.Context(), ctx.Job.AIProviderID); err != nil {
		return fault.WrapError(err, fault.ErrCodeNoProvider, c.Name(), "failed to get AI provider")
	}

	if len(ctx.Job.Categories) == 0 {
		return fault.NewValidationError(fault.ErrCodeNoCategories, c.Name(), "no categories assigned to job")
	}

	if ctx.Job.Schedule == nil {
		if err = strategy.CanExecute(ctx.Context(), ctx.Job); err != nil {
			if errors.IsNoResources(err) {
				return fault.NewRecoverableError(fault.ErrCodeNoTopics, c.Name(), "no topics available for strategy")
			}
			return fault.WrapError(err, fault.ErrCodeInvalidStrategy, c.Name(), "strategy cannot execute")
		}
	}

	ctx.InitValidatedPhase(site, strategy)

	return nil
}

func (c *ValidateJobCommand) OnError(_ *pipeline.Context, err error) error {
	var pErr *fault.PipelineError
	if errs.As(err, &pErr) {
		if pErr.Code == fault.ErrCodeNoTopics {
			return err
		}
	}
	return err
}
