package handlers

import (
	"github.com/davidmovas/postulator/internal/domain/aiusage"
	"github.com/davidmovas/postulator/internal/dto"
	"github.com/davidmovas/postulator/pkg/ctx"
)

type AIUsageHandler struct {
	service aiusage.Service
}

func NewAIUsageHandler(service aiusage.Service) *AIUsageHandler {
	return &AIUsageHandler{
		service: service,
	}
}

// GetUsageLogs returns paginated AI usage logs
// siteID: 0 means all sites
func (h *AIUsageHandler) GetUsageLogs(siteID int64, from, to string, limit, offset int) *dto.Response[*dto.AIUsageLogsResult] {
	timeRange, err := parseTimeRange(from, to)
	if err != nil {
		return fail[*dto.AIUsageLogsResult](err)
	}

	var siteIDPtr *int64
	if siteID > 0 {
		siteIDPtr = &siteID
	}

	if limit <= 0 {
		limit = 20
	}

	result, err := h.service.GetLogs(ctx.MediumCtx(), siteIDPtr, timeRange, limit, offset)
	if err != nil {
		return fail[*dto.AIUsageLogsResult](err)
	}

	return ok(&dto.AIUsageLogsResult{
		Items:   dto.NewAIUsageLogPtrList(result.Items),
		Total:   result.Total,
		Limit:   limit,
		Offset:  offset,
		HasMore: offset+len(result.Items) < result.Total,
	})
}

// GetSummary returns aggregated AI usage statistics
// siteID: 0 means all sites
func (h *AIUsageHandler) GetSummary(siteID int64, from, to string) *dto.Response[*dto.AIUsageSummary] {
	timeRange, err := parseTimeRange(from, to)
	if err != nil {
		return fail[*dto.AIUsageSummary](err)
	}

	var siteIDPtr *int64
	if siteID > 0 {
		siteIDPtr = &siteID
	}

	summary, err := h.service.GetSummary(ctx.MediumCtx(), siteIDPtr, timeRange)
	if err != nil {
		return fail[*dto.AIUsageSummary](err)
	}

	return ok(dto.NewAIUsageSummary(summary))
}

// GetUsageByPeriod returns usage grouped by time period
// siteID: 0 means all sites
func (h *AIUsageHandler) GetUsageByPeriod(siteID int64, from, to, groupBy string) *dto.Response[[]dto.AIUsageByPeriod] {
	timeRange, err := parseTimeRange(from, to)
	if err != nil {
		return fail[[]dto.AIUsageByPeriod](err)
	}

	if groupBy == "" {
		groupBy = "day"
	}

	var siteIDPtr *int64
	if siteID > 0 {
		siteIDPtr = &siteID
	}

	usage, err := h.service.GetUsageByPeriod(ctx.MediumCtx(), siteIDPtr, timeRange, groupBy)
	if err != nil {
		return fail[[]dto.AIUsageByPeriod](err)
	}

	return ok(dto.NewAIUsageByPeriodList(usage))
}

// GetUsageByOperation returns usage grouped by operation type
// siteID: 0 means all sites
func (h *AIUsageHandler) GetUsageByOperation(siteID int64, from, to string) *dto.Response[[]dto.AIUsageByOperation] {
	timeRange, err := parseTimeRange(from, to)
	if err != nil {
		return fail[[]dto.AIUsageByOperation](err)
	}

	var siteIDPtr *int64
	if siteID > 0 {
		siteIDPtr = &siteID
	}

	usage, err := h.service.GetUsageByOperation(ctx.MediumCtx(), siteIDPtr, timeRange)
	if err != nil {
		return fail[[]dto.AIUsageByOperation](err)
	}

	return ok(dto.NewAIUsageByOperationList(usage))
}

// GetUsageByProvider returns usage grouped by provider/model
// siteID: 0 means all sites
func (h *AIUsageHandler) GetUsageByProvider(siteID int64, from, to string) *dto.Response[[]dto.AIUsageByProvider] {
	timeRange, err := parseTimeRange(from, to)
	if err != nil {
		return fail[[]dto.AIUsageByProvider](err)
	}

	var siteIDPtr *int64
	if siteID > 0 {
		siteIDPtr = &siteID
	}

	usage, err := h.service.GetUsageByProvider(ctx.MediumCtx(), siteIDPtr, timeRange)
	if err != nil {
		return fail[[]dto.AIUsageByProvider](err)
	}

	return ok(dto.NewAIUsageByProviderList(usage))
}

// GetUsageBySite returns usage grouped by site
func (h *AIUsageHandler) GetUsageBySite(from, to string) *dto.Response[[]dto.AIUsageBySite] {
	timeRange, err := parseTimeRange(from, to)
	if err != nil {
		return fail[[]dto.AIUsageBySite](err)
	}

	usage, err := h.service.GetUsageBySite(ctx.MediumCtx(), timeRange)
	if err != nil {
		return fail[[]dto.AIUsageBySite](err)
	}

	return ok(dto.NewAIUsageBySiteList(usage))
}

func parseTimeRange(from, to string) (*aiusage.TimeRange, error) {
	if from == "" && to == "" {
		return nil, nil
	}

	fromTime, err := dto.StringToTime(from)
	if err != nil {
		return nil, err
	}

	toTime, err := dto.StringToTime(to)
	if err != nil {
		return nil, err
	}

	return &aiusage.TimeRange{
		Start: fromTime,
		End:   toTime,
	}, nil
}
