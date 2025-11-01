package app

import (
	"github.com/davidmovas/postulator/internal/dto"
	"github.com/davidmovas/postulator/pkg/ctx"
	"github.com/davidmovas/postulator/pkg/errors"
)

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
