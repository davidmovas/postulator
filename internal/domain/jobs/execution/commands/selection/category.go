package selection

import (
	"time"

	"github.com/davidmovas/postulator/internal/domain/categories"
	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/jobs"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/commands"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/fault"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/pipeline"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/pipevents"
	"github.com/davidmovas/postulator/internal/infra/events"
)

var _ pipeline.Command = (*SelectCategoryCommand)(nil)

type SelectCategoryCommand struct {
	*commands.BaseCommand
	categoryService categories.Service
	stateRepository jobs.StateRepository
}

func NewSelectCategoryCommand(
	categoryService categories.Service,
	stateRepository jobs.StateRepository,
) *SelectCategoryCommand {
	return &SelectCategoryCommand{
		BaseCommand:     commands.NewBaseCommand("select_category", pipeline.StateTopicSelected, pipeline.StateCategorySelected),
		categoryService: categoryService,
		stateRepository: stateRepository,
	}
}

func (c *SelectCategoryCommand) Execute(ctx *pipeline.Context) error {
	if len(ctx.Job.Categories) == 0 {
		return fault.NewValidationError(fault.ErrCodeNoCategories, c.Name(), "no categories assigned to job")
	}

	var categoryID int64

	switch ctx.Job.CategoryStrategy {
	case entities.CategoryFixed:
		categoryID = ctx.Job.Categories[0]

	case entities.CategoryRandom:
		categoryID = ctx.Job.Categories[c.randomIndex(len(ctx.Job.Categories))]

	case entities.CategoryRotate:
		state := ctx.Job.State
		if state == nil {
			state = &entities.State{LastCategoryIndex: 0}
		}

		categoryID = ctx.Job.Categories[state.LastCategoryIndex]

		state.LastCategoryIndex = (state.LastCategoryIndex + 1) % len(ctx.Job.Categories)
		if err := c.stateRepository.UpdateCategoryIndex(ctx.Context(), ctx.Job.ID, state.LastCategoryIndex); err != nil {
			ctx.Logger().Warnf("Failed to update category index: %v", err)
		}

	default:
		return fault.NewValidationError(fault.ErrCodeInvalidStrategy, c.Name(), "invalid category strategy")
	}

	category, err := c.categoryService.GetCategory(ctx.Context(), categoryID)
	if err != nil {
		return fault.WrapError(err, fault.ErrCodeRecordNotFound, c.Name(), "failed to get category")
	}

	if ctx.HasSelection() {
		ctx.Selection.Category = category
	} else {
		ctx.InitSelectionPhase(nil, nil, category)
	}

	if ctx.Context() != nil {
		events.Publish(ctx.Context(), events.NewEvent(
			pipevents.EventCategorySelected,
			&pipevents.CategorySelectedEvent{
				JobID:        ctx.Job.ID,
				CategoryID:   category.ID,
				CategoryName: category.Name,
				Strategy:     string(ctx.Job.CategoryStrategy),
			},
		))
	}

	return nil
}

func (c *SelectCategoryCommand) randomIndex(max int) int {
	return int(time.Now().UnixNano() % int64(max))
}
