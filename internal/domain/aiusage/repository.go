package aiusage

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/davidmovas/postulator/internal/infra/database"
	"github.com/davidmovas/postulator/pkg/dbx"
)

type Repository interface {
	Create(ctx context.Context, log *UsageLog) error
	GetLogs(ctx context.Context, siteID *int64, timeRange *TimeRange, limit, offset int) (*LogsResult, error)
	GetSummary(ctx context.Context, siteID *int64, timeRange *TimeRange) (*UsageSummary, error)
	GetByPeriod(ctx context.Context, siteID *int64, timeRange *TimeRange, groupBy string) ([]UsageByPeriod, error)
	GetByOperation(ctx context.Context, siteID *int64, timeRange *TimeRange) ([]UsageByOperation, error)
	GetByProvider(ctx context.Context, siteID *int64, timeRange *TimeRange) ([]UsageByProvider, error)
	GetBySite(ctx context.Context, timeRange *TimeRange) ([]UsageBySite, error)
	DeleteBySiteID(ctx context.Context, siteID int64) error
}

type repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) Repository {
	return &repository{
		db: db,
	}
}

func (r *repository) Create(ctx context.Context, log *UsageLog) error {
	var metadataJSON *string
	if log.Metadata != nil {
		data, err := json.Marshal(log.Metadata)
		if err == nil {
			s := string(data)
			metadataJSON = &s
		}
	}

	successInt := 0
	if log.Success {
		successInt = 1
	}

	var errorMsg *string
	if log.ErrorMessage != "" {
		errorMsg = &log.ErrorMessage
	}

	query, args, err := dbx.ST.Insert("ai_usage_logs").
		Columns(
			"site_id", "operation_type", "provider_name", "model_name",
			"input_tokens", "output_tokens", "total_tokens", "cost_usd",
			"duration_ms", "success", "error_message", "metadata", "created_at",
		).
		Values(
			log.SiteID, string(log.OperationType), log.ProviderName, log.ModelName,
			log.InputTokens, log.OutputTokens, log.TotalTokens, log.CostUSD,
			log.DurationMs, successInt, errorMsg, metadataJSON,
			log.CreatedAt.Format(time.RFC3339),
		).
		ToSql()
	if err != nil {
		return err
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	log.ID = id

	return nil
}

func (r *repository) GetLogs(ctx context.Context, siteID *int64, timeRange *TimeRange, limit, offset int) (*LogsResult, error) {
	// Build base query for count
	countQb := dbx.ST.Select("COUNT(*)").From("ai_usage_logs a").
		LeftJoin("sites s ON a.site_id = s.id")

	if siteID != nil {
		countQb = countQb.Where(squirrel.Eq{"a.site_id": *siteID})
	}

	if timeRange != nil {
		countQb = countQb.Where(squirrel.GtOrEq{"a.created_at": timeRange.Start.Format(time.RFC3339)}).
			Where(squirrel.LtOrEq{"a.created_at": timeRange.End.Format(time.RFC3339)})
	}

	countQuery, countArgs, err := countQb.ToSql()
	if err != nil {
		return nil, err
	}

	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, err
	}

	// Build main query
	qb := dbx.ST.Select(
		"a.id", "a.site_id", "COALESCE(s.name, 'Unknown') as site_name",
		"a.operation_type", "a.provider_name", "a.model_name",
		"a.input_tokens", "a.output_tokens", "a.total_tokens", "a.cost_usd",
		"a.duration_ms", "a.success", "a.error_message", "a.metadata", "a.created_at",
	).From("ai_usage_logs a").
		LeftJoin("sites s ON a.site_id = s.id")

	if siteID != nil {
		qb = qb.Where(squirrel.Eq{"a.site_id": *siteID})
	}

	if timeRange != nil {
		qb = qb.Where(squirrel.GtOrEq{"a.created_at": timeRange.Start.Format(time.RFC3339)}).
			Where(squirrel.LtOrEq{"a.created_at": timeRange.End.Format(time.RFC3339)})
	}

	qb = qb.OrderBy("a.created_at DESC").Limit(uint64(limit)).Offset(uint64(offset))

	query, args, err := qb.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []UsageLog
	for rows.Next() {
		var log UsageLog
		var opType string
		var success int
		var errorMsg, metadataJSON, siteName *string
		var createdAtStr string

		if err := rows.Scan(
			&log.ID, &log.SiteID, &siteName,
			&opType, &log.ProviderName, &log.ModelName,
			&log.InputTokens, &log.OutputTokens, &log.TotalTokens, &log.CostUSD,
			&log.DurationMs, &success, &errorMsg, &metadataJSON, &createdAtStr,
		); err != nil {
			return nil, err
		}

		log.OperationType = OperationType(opType)
		log.Success = success == 1
		if errorMsg != nil {
			log.ErrorMessage = *errorMsg
		}
		if metadataJSON != nil && *metadataJSON != "" {
			var metadata map[string]interface{}
			if err := json.Unmarshal([]byte(*metadataJSON), &metadata); err == nil {
				log.Metadata = metadata
			}
		}
		if createdAt, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			log.CreatedAt = createdAt
		}

		logs = append(logs, log)
	}

	return &LogsResult{
		Items: logs,
		Total: total,
	}, rows.Err()
}

func (r *repository) GetSummary(ctx context.Context, siteID *int64, timeRange *TimeRange) (*UsageSummary, error) {
	qb := dbx.ST.Select(
		"COUNT(*) as total_requests",
		"COALESCE(SUM(total_tokens), 0) as total_tokens",
		"COALESCE(SUM(input_tokens), 0) as total_input_tokens",
		"COALESCE(SUM(output_tokens), 0) as total_output_tokens",
		"COALESCE(SUM(cost_usd), 0) as total_cost_usd",
		"COALESCE(SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END), 0) as success_count",
		"COALESCE(SUM(CASE WHEN success = 0 THEN 1 ELSE 0 END), 0) as error_count",
	).From("ai_usage_logs")

	if siteID != nil {
		qb = qb.Where(squirrel.Eq{"site_id": *siteID})
	}

	if timeRange != nil {
		qb = qb.Where(squirrel.GtOrEq{"created_at": timeRange.Start.Format(time.RFC3339)}).
			Where(squirrel.LtOrEq{"created_at": timeRange.End.Format(time.RFC3339)})
	}

	query, args, err := qb.ToSql()
	if err != nil {
		return nil, err
	}

	var summary UsageSummary
	err = r.db.QueryRowContext(ctx, query, args...).Scan(
		&summary.TotalRequests,
		&summary.TotalTokens,
		&summary.TotalInputTokens,
		&summary.TotalOutputTokens,
		&summary.TotalCostUSD,
		&summary.SuccessCount,
		&summary.ErrorCount,
	)
	if err != nil {
		return nil, err
	}

	return &summary, nil
}

func (r *repository) GetByPeriod(ctx context.Context, siteID *int64, timeRange *TimeRange, groupBy string) ([]UsageByPeriod, error) {
	// groupBy: "day" or "month"
	var dateFormat string
	switch groupBy {
	case "month":
		dateFormat = "%Y-%m"
	default:
		dateFormat = "%Y-%m-%d"
	}

	qb := dbx.ST.Select(
		"strftime('"+dateFormat+"', created_at) as period",
		"COALESCE(SUM(total_tokens), 0) as total_tokens",
		"COALESCE(SUM(cost_usd), 0) as total_cost_usd",
		"COUNT(*) as request_count",
	).From("ai_usage_logs")

	if siteID != nil {
		qb = qb.Where(squirrel.Eq{"site_id": *siteID})
	}

	if timeRange != nil {
		qb = qb.Where(squirrel.GtOrEq{"created_at": timeRange.Start.Format(time.RFC3339)}).
			Where(squirrel.LtOrEq{"created_at": timeRange.End.Format(time.RFC3339)})
	}

	qb = qb.GroupBy("period").OrderBy("period ASC")

	query, args, err := qb.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []UsageByPeriod
	for rows.Next() {
		var u UsageByPeriod
		if err := rows.Scan(&u.Period, &u.TotalTokens, &u.TotalCostUSD, &u.RequestCount); err != nil {
			return nil, err
		}
		results = append(results, u)
	}

	return results, rows.Err()
}

func (r *repository) GetByOperation(ctx context.Context, siteID *int64, timeRange *TimeRange) ([]UsageByOperation, error) {
	qb := dbx.ST.Select(
		"operation_type",
		"COALESCE(SUM(total_tokens), 0) as total_tokens",
		"COALESCE(SUM(cost_usd), 0) as total_cost_usd",
		"COUNT(*) as request_count",
	).From("ai_usage_logs")

	if siteID != nil {
		qb = qb.Where(squirrel.Eq{"site_id": *siteID})
	}

	if timeRange != nil {
		qb = qb.Where(squirrel.GtOrEq{"created_at": timeRange.Start.Format(time.RFC3339)}).
			Where(squirrel.LtOrEq{"created_at": timeRange.End.Format(time.RFC3339)})
	}

	qb = qb.GroupBy("operation_type").OrderBy("total_cost_usd DESC")

	query, args, err := qb.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []UsageByOperation
	for rows.Next() {
		var u UsageByOperation
		var opType string
		if err := rows.Scan(&opType, &u.TotalTokens, &u.TotalCostUSD, &u.RequestCount); err != nil {
			return nil, err
		}
		u.OperationType = OperationType(opType)
		results = append(results, u)
	}

	return results, rows.Err()
}

func (r *repository) GetByProvider(ctx context.Context, siteID *int64, timeRange *TimeRange) ([]UsageByProvider, error) {
	qb := dbx.ST.Select(
		"provider_name",
		"model_name",
		"COALESCE(SUM(total_tokens), 0) as total_tokens",
		"COALESCE(SUM(cost_usd), 0) as total_cost_usd",
		"COUNT(*) as request_count",
	).From("ai_usage_logs")

	if siteID != nil {
		qb = qb.Where(squirrel.Eq{"site_id": *siteID})
	}

	if timeRange != nil {
		qb = qb.Where(squirrel.GtOrEq{"created_at": timeRange.Start.Format(time.RFC3339)}).
			Where(squirrel.LtOrEq{"created_at": timeRange.End.Format(time.RFC3339)})
	}

	qb = qb.GroupBy("provider_name", "model_name").OrderBy("total_cost_usd DESC")

	query, args, err := qb.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []UsageByProvider
	for rows.Next() {
		var u UsageByProvider
		if err := rows.Scan(&u.ProviderName, &u.ModelName, &u.TotalTokens, &u.TotalCostUSD, &u.RequestCount); err != nil {
			return nil, err
		}
		results = append(results, u)
	}

	return results, rows.Err()
}

func (r *repository) GetBySite(ctx context.Context, timeRange *TimeRange) ([]UsageBySite, error) {
	qb := dbx.ST.Select(
		"a.site_id",
		"COALESCE(s.name, 'Unknown') as site_name",
		"COALESCE(SUM(a.total_tokens), 0) as total_tokens",
		"COALESCE(SUM(a.cost_usd), 0) as total_cost_usd",
		"COUNT(*) as request_count",
	).From("ai_usage_logs a").
		LeftJoin("sites s ON a.site_id = s.id")

	if timeRange != nil {
		qb = qb.Where(squirrel.GtOrEq{"a.created_at": timeRange.Start.Format(time.RFC3339)}).
			Where(squirrel.LtOrEq{"a.created_at": timeRange.End.Format(time.RFC3339)})
	}

	qb = qb.GroupBy("a.site_id").OrderBy("total_cost_usd DESC")

	query, args, err := qb.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []UsageBySite
	for rows.Next() {
		var u UsageBySite
		if err := rows.Scan(&u.SiteID, &u.SiteName, &u.TotalTokens, &u.TotalCostUSD, &u.RequestCount); err != nil {
			return nil, err
		}
		results = append(results, u)
	}

	return results, rows.Err()
}

func (r *repository) DeleteBySiteID(ctx context.Context, siteID int64) error {
	query, args, err := dbx.ST.Delete("ai_usage_logs").
		Where(squirrel.Eq{"site_id": siteID}).
		ToSql()
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	return err
}
