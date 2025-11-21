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
		BaseCommand: commands.NewBaseCommand(
			"select_category",
			pipeline.StateTopicSelected,
			pipeline.StateCategorySelected,
		),
		categoryService: categoryService,
		stateRepository: stateRepository,
	}
}

func (c *SelectCategoryCommand) Execute(ctx *pipeline.Context) error {
	if len(ctx.Job.Categories) == 0 {
		return fault.NewValidationError(fault.ErrCodeNoCategories, c.Name(), "no cats assigned to job")
	}

	var categoryIDs []int64

	switch ctx.Job.CategoryStrategy {
	case entities.CategoryFixed:
		categoryIDs = ctx.Job.Categories

	case entities.CategoryRandom:
		id := ctx.Job.Categories[c.randomIndex(len(ctx.Job.Categories))]
		categoryIDs = append(categoryIDs, id)

	case entities.CategoryRotate:
		state := ctx.Job.State
		if state == nil {
			state = &entities.State{LastCategoryIndex: 0}
		}

		id := ctx.Job.Categories[state.LastCategoryIndex]
		categoryIDs = append(categoryIDs, id)

		state.LastCategoryIndex = (state.LastCategoryIndex + 1) % len(ctx.Job.Categories)
		if err := c.stateRepository.UpdateCategoryIndex(ctx.Context(), ctx.Job.ID, state.LastCategoryIndex); err != nil {
			ctx.Logger().Warnf("Failed to update category index: %v", err)
		}

	default:
		return fault.NewValidationError(fault.ErrCodeInvalidStrategy, c.Name(), "invalid category strategy")
	}

	var cats []*entities.Category
	for _, id := range categoryIDs {
		category, err := c.categoryService.GetCategory(ctx.Context(), id)
		if err != nil {
			return fault.WrapError(err, fault.ErrCodeRecordNotFound, c.Name(), "failed to get category")
		}

		cats = append(cats, category)
	}

	if ctx.HasSelection() {
		ctx.Selection.Categories = cats
	} else {
		ctx.InitSelectionPhase(nil, nil, cats...)
	}

	var categoriesNames []string
	for _, cat := range cats {
		categoriesNames = append(categoriesNames, cat.Name)
	}

	events.Publish(ctx.Context(), events.NewEvent(
		pipevents.EventCategorySelected,
		&pipevents.CategorySelectedEvent{
			JobID:      ctx.Job.ID,
			Categories: categoriesNames,
			Strategy:   string(ctx.Job.CategoryStrategy),
		},
	))

	return nil
}

func (c *SelectCategoryCommand) randomIndex(max int) int {
	return int(time.Now().UnixNano() % int64(max))
}
