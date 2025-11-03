package handlers

import (
	"github.com/davidmovas/postulator/internal/domain/stats"
	"github.com/davidmovas/postulator/internal/dto"
	"github.com/davidmovas/postulator/pkg/ctx"
)

type StatsHandler struct {
	service stats.Service
}

func NewStatsHandler(service stats.Service) *StatsHandler {
	return &StatsHandler{
		service: service,
	}
}

func (h *StatsHandler) GetSiteStatistics(siteID int64, from, to string) *dto.Response[[]*dto.SiteStats] {
	fromTime, err := dto.StringToTime(from)
	if err != nil {
		return fail[[]*dto.SiteStats](err)
	}

	toTime, err := dto.StringToTime(to)
	if err != nil {
		return fail[[]*dto.SiteStats](err)
	}

	statistics, err := h.service.GetSiteStatistics(ctx.MediumCtx(), siteID, fromTime, toTime)
	if err != nil {
		return fail[[]*dto.SiteStats](err)
	}

	var dtoStats []*dto.SiteStats
	for _, stat := range statistics {
		dtoStats = append(dtoStats, dto.NewSiteStats(stat))
	}

	return ok(dtoStats)
}

func (h *StatsHandler) GetTotalStatistics(siteID int64) *dto.Response[*dto.SiteStats] {
	statistics, err := h.service.GetTotalStatistics(ctx.MediumCtx(), siteID)
	if err != nil {
		return fail[*dto.SiteStats](err)
	}

	return ok(dto.NewSiteStats(statistics))
}

func (h *StatsHandler) GetDashboardSummary() *dto.Response[*dto.DashboardSummary] {
	summary, err := h.service.GetDashboardSummary(ctx.MediumCtx())
	if err != nil {
		return fail[*dto.DashboardSummary](err)
	}

	return ok(dto.NewDashboardSummary(summary))
}
