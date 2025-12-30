package handlers

import (
	"github.com/davidmovas/postulator/internal/domain/healthcheck"
	"github.com/davidmovas/postulator/internal/dto"
	"github.com/davidmovas/postulator/pkg/ctx"
)

type HealthCheckHandler struct {
	service healthcheck.Service
}

func NewHealthCheckHandler(service healthcheck.Service) *HealthCheckHandler {
	return &HealthCheckHandler{service: service}
}

func (h *HealthCheckHandler) CheckSite(siteID int64) *dto.Response[*dto.HealthCheckHistory] {
	history, err := h.service.CheckSiteByID(ctx.LongCtx(), siteID)
	if err != nil {
		return fail[*dto.HealthCheckHistory](err)
	}
	return ok(dto.NewHealthHistory(history))
}

func (h *HealthCheckHandler) CheckAuto() *dto.Response[*dto.AutoCheckResult] {
	unhealthy, recovered, err := h.service.CheckAutoHealthSites(ctx.FastCtx())
	if err != nil {
		return fail[*dto.AutoCheckResult](err)
	}
	return ok(&dto.AutoCheckResult{Unhealthy: dto.SitesToDTO(unhealthy), Recovered: dto.SitesToDTO(recovered)})
}

func (h *HealthCheckHandler) CheckAll() *dto.Response[*dto.CheckAllResult] {
	checked, failed, err := h.service.CheckAllSites(ctx.LongCtx())
	if err != nil {
		return fail[*dto.CheckAllResult](err)
	}
	return ok(&dto.CheckAllResult{Checked: checked, Failed: failed})
}

func (h *HealthCheckHandler) GetHistory(siteID int64, limit int) *dto.Response[[]*dto.HealthCheckHistory] {
	items, err := h.service.GetSiteHistory(ctx.FastCtx(), siteID, limit)
	if err != nil {
		return fail[[]*dto.HealthCheckHistory](err)
	}
	return ok(dto.NewHealthHistoryList(items))
}

func (h *HealthCheckHandler) GetHistoryByPeriod(siteID int64, from, to string, page, pageSize int) *dto.PaginatedResponse[*dto.HealthCheckHistory] {
	fromT, err := dto.StringToTime(from)
	if err != nil {
		return dto.PaginatedFail[*dto.HealthCheckHistory](err)
	}
	toT, err := dto.StringToTime(to)
	if err != nil {
		return dto.PaginatedFail[*dto.HealthCheckHistory](err)
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	limit := pageSize
	offset := (page - 1) * pageSize

	items, total, err := h.service.GetSiteHistoryByPeriod(ctx.FastCtx(), siteID, fromT, toT, limit, offset)
	if err != nil {
		return dto.PaginatedFail[*dto.HealthCheckHistory](err)
	}

	dtoItems := dto.NewHealthHistoryList(items)
	return dto.PaginatedSuccess(dtoItems, total, limit, offset)
}
