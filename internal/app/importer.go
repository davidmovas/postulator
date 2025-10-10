package app

import (
	"Postulator/internal/domain/entities"
	"Postulator/internal/dto"
	"Postulator/pkg/ctx"
	"Postulator/pkg/errors"
)

func (a *App) ImportTopics(filePath string) *dto.Response[*dto.ImportResult] {
	res, err := a.importerSvc.ImportTopics(ctx.FastCtx(), filePath)
	if err != nil {
		return dtoErr[*dto.ImportResult](asAppErr(err))
	}

	return &dto.Response[*dto.ImportResult]{Success: true, Data: dto.FromImportResult(res)}
}

func (a *App) ImportAndAssignToSite(filePath string, siteID int64, categoryID int64, strategy string) *dto.Response[*dto.ImportResult] {
	strat := entities.TopicStrategy(strategy)
	if strat != entities.StrategyUnique && strat != entities.StrategyVariation {
		return dtoErr[*dto.ImportResult](errors.Validation("invalid topic strategy"))
	}

	res, err := a.importerSvc.ImportAndAssignToSite(ctx.FastCtx(), filePath, siteID, categoryID, strat)
	if err != nil {
		return dtoErr[*dto.ImportResult](asAppErr(err))
	}

	return &dto.Response[*dto.ImportResult]{Success: true, Data: dto.FromImportResult(res)}
}
