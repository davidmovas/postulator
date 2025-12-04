package aiusage

import (
	"time"
)

// OperationType represents the type of AI operation
type OperationType string

const (
	OperationArticleGeneration  OperationType = "article_generation"
	OperationSitemapGeneration  OperationType = "sitemap_generation"
	OperationTopicVariations    OperationType = "topic_variations"
)

// UsageLog represents a single AI usage log entry
type UsageLog struct {
	ID            int64
	SiteID        int64
	OperationType OperationType
	ProviderName  string
	ModelName     string
	InputTokens   int
	OutputTokens  int
	TotalTokens   int
	CostUSD       float64
	DurationMs    int64
	Success       bool
	ErrorMessage  string
	Metadata      map[string]interface{}
	CreatedAt     time.Time
}

// UsageSummary represents aggregated usage statistics
type UsageSummary struct {
	TotalRequests   int64
	TotalTokens     int64
	TotalInputTokens  int64
	TotalOutputTokens int64
	TotalCostUSD    float64
	SuccessCount    int64
	ErrorCount      int64
}

// UsageByPeriod represents usage grouped by time period
type UsageByPeriod struct {
	Period       string  // "2024-01-15" for daily, "2024-01" for monthly
	TotalTokens  int64
	TotalCostUSD float64
	RequestCount int64
}

// UsageByOperation represents usage grouped by operation type
type UsageByOperation struct {
	OperationType OperationType
	TotalTokens   int64
	TotalCostUSD  float64
	RequestCount  int64
}

// UsageByProvider represents usage grouped by provider
type UsageByProvider struct {
	ProviderName string
	ModelName    string
	TotalTokens  int64
	TotalCostUSD float64
	RequestCount int64
}

// UsageBySite represents usage grouped by site
type UsageBySite struct {
	SiteID       int64
	SiteName     string // Populated from join
	TotalTokens  int64
	TotalCostUSD float64
	RequestCount int64
}

// TimeRange represents a time range for queries
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// PredefinedRange returns common time ranges
func Today() TimeRange {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return TimeRange{Start: start, End: now}
}

func ThisWeek() TimeRange {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	start := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
	return TimeRange{Start: start, End: now}
}

func ThisMonth() TimeRange {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	return TimeRange{Start: start, End: now}
}

func Last30Days() TimeRange {
	now := time.Now()
	start := now.AddDate(0, 0, -30)
	return TimeRange{Start: start, End: now}
}

// LogsResult represents a paginated list of usage logs
type LogsResult struct {
	Items []UsageLog
	Total int
}
