package app

import (
	"Postulator/internal/dto"
	appErrors "Postulator/pkg/errors"
	"context"
	"errors"
)

// RestoreScheduler recalculates and restores scheduler state.
func (a *App) RestoreScheduler() *dto.Response[string] {
	if err := a.RestoreState(context.Background()); err != nil {
		return dtoErr[string](appErrors.Scheduler(err))
	}
	return &dto.Response[string]{Success: true, Data: "restored"}
}

// GetAIModels returns all available models grouped by provider.
func (a *App) GetAIModels() *dto.Response[*dto.ModelsByProvider] {
	models := dto.GetAllModels()
	return &dto.Response[*dto.ModelsByProvider]{Success: true, Data: models}
}

// helper to convert errors to dto.Response
func dtoErr[T any](err *appErrors.AppError) *dto.Response[T] {
	if err == nil {
		return &dto.Response[T]{Success: true}
	}
	return &dto.Response[T]{
		Success: false,
		Error:   &dto.Error{Code: string(err.Code), Message: err.Message, Context: err.Context},
	}
}

// asAppErr converts a generic error to *errors.AppError
func asAppErr(err error) *appErrors.AppError {
	if err == nil {
		return nil
	}
	var ae *appErrors.AppError
	if errors.As(err, &ae) {
		return ae
	}
	return appErrors.Internal(err)
}
