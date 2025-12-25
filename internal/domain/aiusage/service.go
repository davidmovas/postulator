package aiusage

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/infra/ai"
	"github.com/davidmovas/postulator/pkg/logger"
)

type Service interface {
	// LogUsage logs an AI usage event
	LogUsage(ctx context.Context, log *UsageLog) error

	// LogFromResult is a convenience method to log usage from an AI result
	LogFromResult(ctx context.Context, siteID int64, operationType OperationType, client ai.Client, usage ai.Usage, durationMs int64, err error, metadata map[string]interface{}) error

	// GetLogs returns paginated usage logs
	GetLogs(ctx context.Context, siteID *int64, timeRange *TimeRange, limit, offset int) (*LogsResult, error)

	// GetSummary returns aggregated usage statistics
	GetSummary(ctx context.Context, siteID *int64, timeRange *TimeRange) (*UsageSummary, error)

	// GetUsageByPeriod returns usage grouped by time period (day/month)
	GetUsageByPeriod(ctx context.Context, siteID *int64, timeRange *TimeRange, groupBy string) ([]UsageByPeriod, error)

	// GetUsageByOperation returns usage grouped by operation type
	GetUsageByOperation(ctx context.Context, siteID *int64, timeRange *TimeRange) ([]UsageByOperation, error)

	// GetUsageByProvider returns usage grouped by provider/model
	GetUsageByProvider(ctx context.Context, siteID *int64, timeRange *TimeRange) ([]UsageByProvider, error)

	// GetUsageBySite returns usage grouped by site
	GetUsageBySite(ctx context.Context, timeRange *TimeRange) ([]UsageBySite, error)

	// DeleteBySiteID deletes all usage logs for a site
	DeleteBySiteID(ctx context.Context, siteID int64) error
}

type service struct {
	repo   Repository
	logger *logger.Logger
}

func NewService(repo Repository, logger *logger.Logger) Service {
	return &service{
		repo:   repo,
		logger: logger,
	}
}

func (s *service) LogUsage(ctx context.Context, log *UsageLog) error {
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}
	return s.repo.Create(ctx, log)
}

func (s *service) LogFromResult(ctx context.Context, siteID int64, operationType OperationType, client ai.Client, usage ai.Usage, durationMs int64, err error, metadata map[string]interface{}) error {
	log := &UsageLog{
		SiteID:        siteID,
		OperationType: operationType,
		ProviderName:  client.GetProviderName(),
		ModelName:     client.GetModelName(),
		InputTokens:   usage.InputTokens,
		OutputTokens:  usage.OutputTokens,
		TotalTokens:   usage.TotalTokens,
		CostUSD:       usage.CostUSD,
		DurationMs:    durationMs,
		Success:       err == nil,
		Metadata:      metadata,
		CreatedAt:     time.Now(),
	}

	if err != nil {
		log.ErrorMessage = err.Error()
	}

	if logErr := s.repo.Create(ctx, log); logErr != nil {
		s.logger.ErrorWithErr(logErr, "Failed to log AI usage")
		// Don't return error - logging failure shouldn't break the main flow
	}

	return nil
}

func (s *service) GetLogs(ctx context.Context, siteID *int64, timeRange *TimeRange, limit, offset int) (*LogsResult, error) {
	return s.repo.GetLogs(ctx, siteID, timeRange, limit, offset)
}

func (s *service) GetSummary(ctx context.Context, siteID *int64, timeRange *TimeRange) (*UsageSummary, error) {
	return s.repo.GetSummary(ctx, siteID, timeRange)
}

func (s *service) GetUsageByPeriod(ctx context.Context, siteID *int64, timeRange *TimeRange, groupBy string) ([]UsageByPeriod, error) {
	return s.repo.GetByPeriod(ctx, siteID, timeRange, groupBy)
}

func (s *service) GetUsageByOperation(ctx context.Context, siteID *int64, timeRange *TimeRange) ([]UsageByOperation, error) {
	return s.repo.GetByOperation(ctx, siteID, timeRange)
}

func (s *service) GetUsageByProvider(ctx context.Context, siteID *int64, timeRange *TimeRange) ([]UsageByProvider, error) {
	return s.repo.GetByProvider(ctx, siteID, timeRange)
}

func (s *service) GetUsageBySite(ctx context.Context, timeRange *TimeRange) ([]UsageBySite, error) {
	return s.repo.GetBySite(ctx, timeRange)
}

func (s *service) DeleteBySiteID(ctx context.Context, siteID int64) error {
	return s.repo.DeleteBySiteID(ctx, siteID)
}
