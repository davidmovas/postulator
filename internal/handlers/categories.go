package handlers

import (
	"github.com/davidmovas/postulator/internal/domain/categories"
	"github.com/davidmovas/postulator/internal/dto"
	"github.com/davidmovas/postulator/pkg/ctx"
)

type CategoriesHandler struct {
	service categories.Service
}

func NewCategoriesHandler(service categories.Service) *CategoriesHandler {
	return &CategoriesHandler{
		service: service,
	}
}

func (h *CategoriesHandler) CreateCategory(category *dto.Category) *dto.Response[string] {
	entity, err := category.ToEntity()
	if err != nil {
		return fail[string](err)
	}

	if err = h.service.CreateCategory(ctx.FastCtx(), entity); err != nil {
		return fail[string](err)
	}

	return ok("Categories created successfully")
}

func (h *CategoriesHandler) GetCategory(id int64) *dto.Response[*dto.Category] {
	category, err := h.service.GetCategory(ctx.FastCtx(), id)
	if err != nil {
		return fail[*dto.Category](err)
	}

	return ok(dto.NewCategory(category))
}

func (h *CategoriesHandler) ListSiteCategories(siteID int64) *dto.Response[[]*dto.Category] {
	siteCategories, err := h.service.ListSiteCategories(ctx.FastCtx(), siteID)
	if err != nil {
		return fail[[]*dto.Category](err)
	}

	var dtoCategories []*dto.Category
	for _, category := range siteCategories {
		dtoCategories = append(dtoCategories, dto.NewCategory(category))
	}

	return ok(dtoCategories)
}

func (h *CategoriesHandler) UpdateCategory(category *dto.Category) *dto.Response[string] {
	entity, err := category.ToEntity()
	if err != nil {
		return fail[string](err)
	}

	if err = h.service.UpdateCategory(ctx.FastCtx(), entity); err != nil {
		return fail[string](err)
	}

	return ok("Categories updated successfully")
}

func (h *CategoriesHandler) DeleteCategory(id int64) *dto.Response[string] {
	if err := h.service.DeleteCategory(ctx.FastCtx(), id); err != nil {
		return fail[string](err)
	}

	return ok("Categories deleted successfully")
}

func (h *CategoriesHandler) SyncFromWordPress(siteID int64) *dto.Response[string] {
	if err := h.service.SyncFromWordPress(ctx.LongCtx(), siteID); err != nil {
		return fail[string](err)
	}

	return ok("Categories synced successfully")
}

func (h *CategoriesHandler) CreateInWordPress(category *dto.Category) *dto.Response[string] {
	entity, err := category.ToEntity()
	if err != nil {
		return fail[string](err)
	}

	if err = h.service.CreateInWordPress(ctx.LongCtx(), entity); err != nil {
		return fail[string](err)
	}

	return ok("Categories created in WordPress successfully")
}

func (h *CategoriesHandler) UpdateInWordPress(category *dto.Category) *dto.Response[string] {
	entity, err := category.ToEntity()
	if err != nil {
		return fail[string](err)
	}

	if err = h.service.UpdateInWordPress(ctx.LongCtx(), entity); err != nil {
		return fail[string](err)
	}

	return ok("Categories updated in WordPress successfully")
}

func (h *CategoriesHandler) DeleteInWordPress(categoryID int64) *dto.Response[string] {
	if err := h.service.DeleteInWordPress(ctx.LongCtx(), categoryID); err != nil {
		return fail[string](err)
	}

	return ok("Categories deleted from WordPress successfully")
}

func (h *CategoriesHandler) GetStatistics(categoryID int64, from, to string) *dto.Response[[]*dto.Statistics] {
	fromTime, err := dto.StringToTime(from)
	if err != nil {
		return fail[[]*dto.Statistics](err)
	}

	toTime, err := dto.StringToTime(to)
	if err != nil {
		return fail[[]*dto.Statistics](err)
	}

	stats, err := h.service.GetStatistics(ctx.LongCtx(), categoryID, fromTime, toTime)
	if err != nil {
		return fail[[]*dto.Statistics](err)
	}

	var dtoStats []*dto.Statistics
	for _, stat := range stats {
		dtoStats = append(dtoStats, dto.NewStatistics(stat))
	}

	return ok(dtoStats)
}

func (h *CategoriesHandler) GetSiteStatistics(siteID int64, from, to string) *dto.Response[[]*dto.Statistics] {
	fromTime, err := dto.StringToTime(from)
	if err != nil {
		return fail[[]*dto.Statistics](err)
	}

	toTime, err := dto.StringToTime(to)
	if err != nil {
		return fail[[]*dto.Statistics](err)
	}

	stats, err := h.service.GetSiteStatistics(ctx.LongCtx(), siteID, fromTime, toTime)
	if err != nil {
		return fail[[]*dto.Statistics](err)
	}

	var dtoStats []*dto.Statistics
	for _, stat := range stats {
		dtoStats = append(dtoStats, dto.NewStatistics(stat))
	}

	return ok(dtoStats)
}
