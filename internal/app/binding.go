package app

import (
	"Postulator/internal/dto"
	"Postulator/pkg/errors"
	"context"
)

// StartApp initializes services and starts the scheduler.
func (a *App) StartApp(dbPath string) *dto.Response[string] {
	ctx := context.Background()
	if dbPath != "" {
		if err := a.InitDB(dbPath); err != nil {
			return dtoErr[string](errors.Internal(err))
		}
	}
	a.InitWP()
	if err := a.BuildServices(); err != nil {
		return dtoErr[string](errors.Internal(err))
	}
	if err := a.Start(ctx); err != nil {
		return dtoErr[string](errors.Scheduler(err))
	}
	return &dto.Response[string]{Success: true, Data: "started"}
}

// StopApp stops the scheduler and disposes resources.
func (a *App) StopApp() *dto.Response[string] {
	a.Stop()
	return &dto.Response[string]{Success: true, Data: "stopped"}
}

// RestoreScheduler recalculates and restores scheduler state.
func (a *App) RestoreScheduler() *dto.Response[string] {
	if err := a.RestoreState(context.Background()); err != nil {
		return dtoErr[string](errors.Scheduler(err))
	}
	return &dto.Response[string]{Success: true, Data: "restored"}
}

// GetAIModels returns all available models grouped by provider.
func (a *App) GetAIModels() *dto.Response[*dto.ModelsByProvider] {
	models := dto.GetAllModels()
	return &dto.Response[*dto.ModelsByProvider]{Success: true, Data: models}
}

// helper to convert errors to dto.Response
func dtoErr[T any](err *errors.AppError) *dto.Response[T] {
	if err == nil {
		return &dto.Response[T]{Success: true}
	}
	return &dto.Response[T]{
		Success: false,
		Error:   &dto.Error{Code: string(err.Code), Message: err.Message, Context: err.Context},
	}
}

// asAppErr converts a generic error to *errors.AppError
func asAppErr(err error) *errors.AppError {
	if err == nil {
		return nil
	}
	if ae, ok := err.(*errors.AppError); ok {
		return ae
	}
	return errors.Internal(err)
}
