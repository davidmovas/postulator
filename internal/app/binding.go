package app

import (
	"Postulator/internal/dto"
	"Postulator/pkg/ctx"
	"Postulator/pkg/errors"
)

// RestoreScheduler recalculates and restores scheduler state.
func (a *App) RestoreScheduler() *dto.Response[string] {
	if err := a.RestoreState(ctx.LongCtx()); err != nil {
		return dtoErr[string](errors.Scheduler(err))
	}

	return &dto.Response[string]{Success: true, Data: "restored"}
}

// GetAIModels returns all available models grouped by provider.
func (a *App) GetAIModels() *dto.Response[*dto.ModelsByProvider] {
	models := dto.GetAllModels()
	return &dto.Response[*dto.ModelsByProvider]{Success: true, Data: models}
}
