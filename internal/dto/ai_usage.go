package dto

import (
	"encoding/json"

	"github.com/davidmovas/postulator/internal/domain/aiusage"
)

// AIUsageSummary represents aggregated AI usage statistics
type AIUsageSummary struct {
	TotalRequests     int64   `json:"totalRequests"`
	TotalTokens       int64   `json:"totalTokens"`
	TotalInputTokens  int64   `json:"totalInputTokens"`
	TotalOutputTokens int64   `json:"totalOutputTokens"`
	TotalCostUSD      float64 `json:"totalCostUsd"`
	SuccessCount      int64   `json:"successCount"`
	ErrorCount        int64   `json:"errorCount"`
}

func NewAIUsageSummary(entity *aiusage.UsageSummary) *AIUsageSummary {
	if entity == nil {
		return nil
	}
	return &AIUsageSummary{
		TotalRequests:     entity.TotalRequests,
		TotalTokens:       entity.TotalTokens,
		TotalInputTokens:  entity.TotalInputTokens,
		TotalOutputTokens: entity.TotalOutputTokens,
		TotalCostUSD:      entity.TotalCostUSD,
		SuccessCount:      entity.SuccessCount,
		ErrorCount:        entity.ErrorCount,
	}
}

// AIUsageByPeriod represents usage grouped by time period
type AIUsageByPeriod struct {
	Period       string  `json:"period"`
	TotalTokens  int64   `json:"totalTokens"`
	TotalCostUSD float64 `json:"totalCostUsd"`
	RequestCount int64   `json:"requestCount"`
}

func NewAIUsageByPeriodList(entities []aiusage.UsageByPeriod) []AIUsageByPeriod {
	result := make([]AIUsageByPeriod, len(entities))
	for i, e := range entities {
		result[i] = AIUsageByPeriod{
			Period:       e.Period,
			TotalTokens:  e.TotalTokens,
			TotalCostUSD: e.TotalCostUSD,
			RequestCount: e.RequestCount,
		}
	}
	return result
}

// AIUsageByOperation represents usage grouped by operation type
type AIUsageByOperation struct {
	OperationType string  `json:"operationType"`
	TotalTokens   int64   `json:"totalTokens"`
	TotalCostUSD  float64 `json:"totalCostUsd"`
	RequestCount  int64   `json:"requestCount"`
}

func NewAIUsageByOperationList(entities []aiusage.UsageByOperation) []AIUsageByOperation {
	result := make([]AIUsageByOperation, len(entities))
	for i, e := range entities {
		result[i] = AIUsageByOperation{
			OperationType: string(e.OperationType),
			TotalTokens:   e.TotalTokens,
			TotalCostUSD:  e.TotalCostUSD,
			RequestCount:  e.RequestCount,
		}
	}
	return result
}

// AIUsageByProvider represents usage grouped by provider/model
type AIUsageByProvider struct {
	ProviderName string  `json:"providerName"`
	ModelName    string  `json:"modelName"`
	TotalTokens  int64   `json:"totalTokens"`
	TotalCostUSD float64 `json:"totalCostUsd"`
	RequestCount int64   `json:"requestCount"`
}

func NewAIUsageByProviderList(entities []aiusage.UsageByProvider) []AIUsageByProvider {
	result := make([]AIUsageByProvider, len(entities))
	for i, e := range entities {
		result[i] = AIUsageByProvider{
			ProviderName: e.ProviderName,
			ModelName:    e.ModelName,
			TotalTokens:  e.TotalTokens,
			TotalCostUSD: e.TotalCostUSD,
			RequestCount: e.RequestCount,
		}
	}
	return result
}

// AIUsageBySite represents usage grouped by site
type AIUsageBySite struct {
	SiteID       int64   `json:"siteId"`
	SiteName     string  `json:"siteName"`
	TotalTokens  int64   `json:"totalTokens"`
	TotalCostUSD float64 `json:"totalCostUsd"`
	RequestCount int64   `json:"requestCount"`
}

func NewAIUsageBySiteList(entities []aiusage.UsageBySite) []AIUsageBySite {
	result := make([]AIUsageBySite, len(entities))
	for i, e := range entities {
		result[i] = AIUsageBySite{
			SiteID:       e.SiteID,
			SiteName:     e.SiteName,
			TotalTokens:  e.TotalTokens,
			TotalCostUSD: e.TotalCostUSD,
			RequestCount: e.RequestCount,
		}
	}
	return result
}

// AIUsageTimeRange represents a time range for queries
type AIUsageTimeRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// AIUsageLogsResult represents a paginated list of AI usage logs
type AIUsageLogsResult struct {
	Items   []*AIUsageLog `json:"items"`
	Total   int           `json:"total"`
	Limit   int           `json:"limit"`
	Offset  int           `json:"offset"`
	HasMore bool          `json:"hasMore"`
}

// AIUsageLog represents a single AI usage log entry
type AIUsageLog struct {
	ID            int64   `json:"id"`
	SiteID        int64   `json:"siteId"`
	OperationType string  `json:"operationType"`
	ProviderName  string  `json:"providerName"`
	ModelName     string  `json:"modelName"`
	InputTokens   int     `json:"inputTokens"`
	OutputTokens  int     `json:"outputTokens"`
	TotalTokens   int     `json:"totalTokens"`
	CostUSD       float64 `json:"costUsd"`
	DurationMs    int64   `json:"durationMs"`
	Success       bool    `json:"success"`
	ErrorMessage  string  `json:"errorMessage,omitempty"`
	Metadata      string  `json:"metadata,omitempty"`
	CreatedAt     string  `json:"createdAt"`
}

func metadataToString(m map[string]interface{}) string {
	if m == nil {
		return ""
	}
	data, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(data)
}

func NewAIUsageLogList(entities []aiusage.UsageLog) []AIUsageLog {
	result := make([]AIUsageLog, len(entities))
	for i, e := range entities {
		result[i] = AIUsageLog{
			ID:            e.ID,
			SiteID:        e.SiteID,
			OperationType: string(e.OperationType),
			ProviderName:  e.ProviderName,
			ModelName:     e.ModelName,
			InputTokens:   e.InputTokens,
			OutputTokens:  e.OutputTokens,
			TotalTokens:   e.TotalTokens,
			CostUSD:       e.CostUSD,
			DurationMs:    e.DurationMs,
			Success:       e.Success,
			ErrorMessage:  e.ErrorMessage,
			Metadata:      metadataToString(e.Metadata),
			CreatedAt:     TimeToString(e.CreatedAt),
		}
	}
	return result
}

func NewAIUsageLogPtrList(entities []aiusage.UsageLog) []*AIUsageLog {
	result := make([]*AIUsageLog, len(entities))
	for i, e := range entities {
		result[i] = &AIUsageLog{
			ID:            e.ID,
			SiteID:        e.SiteID,
			OperationType: string(e.OperationType),
			ProviderName:  e.ProviderName,
			ModelName:     e.ModelName,
			InputTokens:   e.InputTokens,
			OutputTokens:  e.OutputTokens,
			TotalTokens:   e.TotalTokens,
			CostUSD:       e.CostUSD,
			DurationMs:    e.DurationMs,
			Success:       e.Success,
			ErrorMessage:  e.ErrorMessage,
			Metadata:      metadataToString(e.Metadata),
			CreatedAt:     TimeToString(e.CreatedAt),
		}
	}
	return result
}
