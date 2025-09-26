package repository

import (
	"Postulator/internal/models"
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
)

func (r *Repository) GetTopics(ctx context.Context, limit int, offset int) (*models.PaginationResult[*models.Topic], error) {
	// Get topics
	query, args := builder.
		Select(
			"id",
			"title",
			"keywords",
			"category",
			"tags",
			"created_at",
			"updated_at",
		).
		From("topics").
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query topics: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var topics []*models.Topic
	for rows.Next() {
		var topic models.Topic
		if err = rows.Scan(
			&topic.ID,
			&topic.Title,
			&topic.Keywords,
			&topic.Category,
			&topic.Tags,
			&topic.CreatedAt,
			&topic.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan topic: %w", err)
		}
		topics = append(topics, &topic)
	}

	// Get total count
	countQuery, countArgs := builder.
		Select("COUNT(id)").
		From("topics").
		MustSql()

	var total int
	err = r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count topics: %w", err)
	}

	return &models.PaginationResult[*models.Topic]{
		Data:   topics,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (r *Repository) GetTopic(ctx context.Context, id int64) (*models.Topic, error) {
	query, args := builder.
		Select(
			"id",
			"title",
			"keywords",
			"category",
			"tags",
			"created_at",
			"updated_at",
		).
		From("topics").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	var topic models.Topic
	if err := r.db.QueryRowContext(ctx, query, args...).
		Scan(
			&topic.ID,
			&topic.Title,
			&topic.Keywords,
			&topic.Category,
			&topic.Tags,
			&topic.CreatedAt,
			&topic.UpdatedAt,
		); err != nil {
		return nil, fmt.Errorf("failed to query topic: %w", err)
	}

	return &topic, nil
}

func (r *Repository) GetTopicsBySiteID(ctx context.Context, siteID int64, limit int, offset int) (*models.PaginationResult[*models.Topic], error) {
	query, args := builder.
		Select(
			"t.id",
			"t.title",
			"t.keywords",
			"t.category",
			"t.tags",
			"t.created_at",
			"t.updated_at",
		).
		From("topics t").
		Join("site_topics st ON t.id = st.topic_id").
		Where(squirrel.Eq{"st.site_id": siteID}).
		OrderBy("t.created_at DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query topics by site: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var topics []*models.Topic
	for rows.Next() {
		var topic models.Topic
		if err = rows.Scan(
			&topic.ID,
			&topic.Title,
			&topic.Keywords,
			&topic.Category,
			&topic.Tags,
			&topic.CreatedAt,
			&topic.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan topic: %w", err)
		}
		topics = append(topics, &topic)
	}

	// Get total count
	countQuery, countArgs := builder.
		Select("COUNT(id)").
		From("topics t").
		Join("site_topics st ON t.id = st.topic_id").
		Where(squirrel.Eq{"st.site_id": siteID}).
		MustSql()

	var total int
	err = r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count topics by site: %w", err)
	}

	return &models.PaginationResult[*models.Topic]{
		Data:   topics,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (r *Repository) CreateTopic(ctx context.Context, topic *models.Topic) (*models.Topic, error) {
	query, args := builder.
		Insert("topics").
		Columns(
			"title",
			"keywords",
			"category",
			"tags",
			"created_at",
			"updated_at",
		).
		Values(
			topic.Title,
			topic.Keywords,
			topic.Category,
			topic.Tags,
			topic.CreatedAt,
			topic.UpdatedAt,
		).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to create topic: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get insert id: %w", err)
	}

	topic.ID = id
	return topic, nil
}

func (r *Repository) UpdateTopic(ctx context.Context, topic *models.Topic) (*models.Topic, error) {
	query, args := builder.
		Update("topics").
		Set("title", topic.Title).
		Set("keywords", topic.Keywords).
		Set("category", topic.Category).
		Set("tags", topic.Tags).
		Set("updated_at", topic.UpdatedAt).
		Where(squirrel.Eq{"id": topic.ID}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update topic: %w", err)
	}

	return topic, nil
}

func (r *Repository) DeleteTopic(ctx context.Context, id int64) error {
	query, args := builder.
		Delete("topics").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete topic: %w", err)
	}

	return nil
}

func (r *Repository) GetAllTopicsForRandomSelection(ctx context.Context) ([]*models.Topic, error) {
	query, args := builder.
		Select(
			"id",
			"title",
			"keywords",
			"category",
			"tags",
			"created_at",
			"updated_at",
		).
		From("topics").
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query all topics for random selection: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var topics []*models.Topic
	for rows.Next() {
		var topic models.Topic
		if err = rows.Scan(
			&topic.ID,
			&topic.Title,
			&topic.Keywords,
			&topic.Category,
			&topic.Tags,
			&topic.CreatedAt,
			&topic.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan topic: %w", err)
		}
		topics = append(topics, &topic)
	}

	return topics, nil
}

func (r *Repository) CreateSiteTopic(ctx context.Context, siteTopic *models.SiteTopic) (*models.SiteTopic, error) {
	query, args := builder.
		Insert("site_topics").
		Columns(
			"site_id",
			"topic_id",
			"priority",
			"usage_count",
			"round_robin_pos",
		).
		Values(
			siteTopic.SiteID,
			siteTopic.TopicID,
			siteTopic.Priority,
			siteTopic.UsageCount,
			siteTopic.RoundRobinPos,
		).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to create site topic: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get insert id: %w", err)
	}

	siteTopic.ID = id
	return siteTopic, nil
}

func (r *Repository) GetSiteTopics(ctx context.Context, siteID int64, limit int, offset int) (*models.PaginationResult[*models.SiteTopic], error) {
	query, args := builder.
		Select(
			"st.id",
			"st.site_id",
			"st.topic_id",
			"st.priority",
			"st.last_used_at",
			"st.usage_count",
			"st.round_robin_pos",
			"st.created_at",
			"st.updated_at",
		).
		From("site_topics st").
		Where(squirrel.Eq{"st.site_id": siteID}).
		OrderBy("st.id DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query site topics: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var siteTopics []*models.SiteTopic
	for rows.Next() {
		var siteTopic models.SiteTopic
		if err = rows.Scan(
			&siteTopic.ID,
			&siteTopic.SiteID,
			&siteTopic.TopicID,
			&siteTopic.Priority,
			&siteTopic.LastUsedAt,
			&siteTopic.UsageCount,
			&siteTopic.RoundRobinPos,
			&siteTopic.CreatedAt,
			&siteTopic.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan site topic: %w", err)
		}
		siteTopics = append(siteTopics, &siteTopic)
	}

	// Get total count
	countQuery, countArgs := builder.
		Select("COUNT(id)").
		From("site_topics st").
		Where(squirrel.Eq{"st.site_id": siteID}).
		MustSql()

	var total int
	err = r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count site topics: %w", err)
	}

	return &models.PaginationResult[*models.SiteTopic]{
		Data:   siteTopics,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (r *Repository) GetTopicSites(ctx context.Context, topicID int64, limit int, offset int) (*models.PaginationResult[*models.SiteTopic], error) {
	query, args := builder.
		Select(
			"st.id",
			"st.site_id",
			"st.topic_id",
			"st.priority",
			"st.last_used_at",
			"st.usage_count",
			"st.round_robin_pos",
			"st.created_at",
			"st.updated_at",
		).
		From("site_topics st").
		Where(squirrel.Eq{"st.topic_id": topicID}).
		OrderBy("st.id DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query topic sites: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var siteTopics []*models.SiteTopic
	for rows.Next() {
		var siteTopic models.SiteTopic
		if err = rows.Scan(
			&siteTopic.ID,
			&siteTopic.SiteID,
			&siteTopic.TopicID,
			&siteTopic.Priority,
			&siteTopic.LastUsedAt,
			&siteTopic.UsageCount,
			&siteTopic.RoundRobinPos,
			&siteTopic.CreatedAt,
			&siteTopic.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan site topic: %w", err)
		}
		siteTopics = append(siteTopics, &siteTopic)
	}

	// Get total count
	countQuery, countArgs := builder.
		Select("COUNT(id)").
		From("site_topics st").
		Where(squirrel.Eq{"st.topic_id": topicID}).
		MustSql()

	var total int
	err = r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count topic sites: %w", err)
	}

	return &models.PaginationResult[*models.SiteTopic]{
		Data:   siteTopics,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (r *Repository) GetSiteTopic(ctx context.Context, siteID int64, topicID int64) (*models.SiteTopic, error) {
	query, args := builder.
		Select(
			"id",
			"site_id",
			"topic_id",
			"priority",
			"last_used_at",
			"usage_count",
			"round_robin_pos",
			"created_at",
			"updated_at",
		).
		From("site_topics").
		Where(squirrel.Eq{"site_id": siteID, "topic_id": topicID}).
		MustSql()

	var siteTopic models.SiteTopic
	if err := r.db.QueryRowContext(ctx, query, args...).
		Scan(
			&siteTopic.ID,
			&siteTopic.SiteID,
			&siteTopic.TopicID,
			&siteTopic.Priority,
			&siteTopic.LastUsedAt,
			&siteTopic.UsageCount,
			&siteTopic.RoundRobinPos,
			&siteTopic.CreatedAt,
			&siteTopic.UpdatedAt,
		); err != nil {
		return nil, fmt.Errorf("failed to query site topic: %w", err)
	}

	return &siteTopic, nil
}

func (r *Repository) UpdateSiteTopic(ctx context.Context, siteTopic *models.SiteTopic) (*models.SiteTopic, error) {
	query, args := builder.
		Update("site_topics").
		Set("site_id", siteTopic.SiteID).
		Set("topic_id", siteTopic.TopicID).
		Set("priority", siteTopic.Priority).
		Set("usage_count", siteTopic.UsageCount).
		Set("round_robin_pos", siteTopic.RoundRobinPos).
		Set("last_used_at", siteTopic.LastUsedAt).
		Set("updated_at", siteTopic.UpdatedAt).
		Where(squirrel.Eq{"id": siteTopic.ID}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update site topic: %w", err)
	}

	return siteTopic, nil
}

func (r *Repository) DeleteSiteTopic(ctx context.Context, id int64) error {
	query, args := builder.
		Delete("site_topics").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete site topic: %w", err)
	}

	return nil
}

func (r *Repository) DeleteSiteTopicBySiteAndTopic(ctx context.Context, siteID int64, topicID int64) error {
	query, args := builder.
		Delete("site_topics").
		Where(squirrel.Eq{"site_id": siteID, "topic_id": topicID}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete site topic by site and topic: %w", err)
	}

	return nil
}

func (r *Repository) GetSiteTopicsForSelection(ctx context.Context, siteID int64, strategy string) ([]*models.SiteTopic, error) {
	var query string
	var args []interface{}

	switch strategy {
	case "unique":
		// Get SiteTopics with UsageCount=0
		query, args = builder.
			Select(
				"id",
				"site_id",
				"topic_id",
				"priority",
				"last_used_at",
				"usage_count",
				"round_robin_pos",
				"created_at",
				"updated_at",
			).
			From("site_topics").
			Where(squirrel.And{
				squirrel.Eq{"site_id": siteID},
				squirrel.Eq{"usage_count": 0},
			}).
			OrderBy("priority DESC, created_at ASC").
			MustSql()

	case "round_robin":
		// Get all site topics ordered by RoundRobinPos, then by LastUsedAt as tie-breaker
		query, args = builder.
			Select(
				"id",
				"site_id",
				"topic_id",
				"priority",
				"last_used_at",
				"usage_count",
				"round_robin_pos",
				"created_at",
				"updated_at",
			).
			From("site_topics").
			Where(squirrel.Eq{"site_id": siteID}).
			OrderBy("round_robin_pos ASC, last_used_at ASC NULLS FIRST").
			MustSql()

	case "random":
		// Get all site topics for random selection
		query, args = builder.
			Select(
				"id",
				"site_id",
				"topic_id",
				"priority",
				"last_used_at",
				"usage_count",
				"round_robin_pos",
				"created_at",
				"updated_at",
			).
			From("site_topics").
			Where(squirrel.Eq{"site_id": siteID}).
			MustSql()

	case "random_all":
		// This strategy will be handled differently - we need all topics in the system
		// Return empty slice for now, will be handled in the handler layer
		return []*models.SiteTopic{}, nil

	default:
		// Default behavior - return all site topics
		query, args = builder.
			Select(
				"id",
				"site_id",
				"topic_id",
				"priority",
				"last_used_at",
				"usage_count",
				"round_robin_pos",
				"created_at",
				"updated_at",
			).
			From("site_topics").
			Where(squirrel.Eq{"site_id": siteID}).
			OrderBy("priority DESC, created_at ASC").
			MustSql()
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query site topics for selection: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var siteTopics []*models.SiteTopic
	for rows.Next() {
		var siteTopic models.SiteTopic
		if err = rows.Scan(
			&siteTopic.ID,
			&siteTopic.SiteID,
			&siteTopic.TopicID,
			&siteTopic.Priority,
			&siteTopic.LastUsedAt,
			&siteTopic.UsageCount,
			&siteTopic.RoundRobinPos,
			&siteTopic.CreatedAt,
			&siteTopic.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan site topic: %w", err)
		}
		siteTopics = append(siteTopics, &siteTopic)
	}

	return siteTopics, nil
}

func (r *Repository) UpdateSiteTopicUsage(ctx context.Context, siteTopicID int64, strategy string) error {
	// Get the site ID for this topic to calculate proper round-robin position
	siteQuery, siteArgs := builder.
		Select("site_id", "usage_count", "round_robin_pos").
		From("site_topics").
		Where(squirrel.Eq{"id": siteTopicID}).
		MustSql()

	var siteID int64
	var currentUsageCount, currentRoundRobinPos int
	err := r.db.QueryRowContext(ctx, siteQuery, siteArgs...).
		Scan(&siteID, &currentUsageCount, &currentRoundRobinPos)
	if err != nil {
		return fmt.Errorf("failed to get current site topic values: %w", err)
	}

	newUsageCount := currentUsageCount + 1
	newRoundRobinPos := currentRoundRobinPos
	now := time.Now()

	switch strategy {
	case "unique":
		// For unique strategy, just increment usage count and update last_used_at
		// Round robin position doesn't matter for unique strategy
		break

	case "round_robin":
		// Get total count of topics for this site to calculate proper round-robin position
		countQuery, countArgs := builder.
			Select("COUNT(*)").
			From("site_topics").
			Where(squirrel.Eq{"site_id": siteID}).
			MustSql()

		var totalTopics int
		err = r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&totalTopics)
		if err != nil {
			return fmt.Errorf("failed to get total topics count: %w", err)
		}

		if totalTopics > 0 {
			// For round-robin, we want to cycle from 1 to totalTopics, then back to 1
			// Start at 1 if never used (pos 0), otherwise cycle
			if currentRoundRobinPos == 0 {
				newRoundRobinPos = 1
			} else {
				newRoundRobinPos = (currentRoundRobinPos % totalTopics) + 1
			}
		}

	case "random":
		// For random strategy, just increment usage count and update last_used_at
		// Round robin position doesn't matter for random strategy
		break

	case "random_all":
		// For random_all strategy, just increment usage count and update last_used_at
		// Round robin position doesn't matter for random_all strategy
		break

	default:
		// Default behavior - just increment usage count
		break
	}

	updateQuery, updateArgs := builder.
		Update("site_topics").
		Set("usage_count", newUsageCount).
		Set("round_robin_pos", newRoundRobinPos).
		Set("last_used_at", now).
		Set("updated_at", now).
		Where(squirrel.Eq{"id": siteTopicID}).
		MustSql()

	result, err := r.db.ExecContext(ctx, updateQuery, updateArgs...)
	if err != nil {
		return fmt.Errorf("failed to update site topic usage: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no site topic found with id %d", siteTopicID)
	}

	return nil
}

func (r *Repository) GetTopicStats(ctx context.Context, siteID int64) (*models.TopicStats, error) {
	stats := &models.TopicStats{SiteID: siteID}

	// Get total topics count
	totalQuery, totalArgs := builder.
		Select("COUNT(id) as total").
		From("site_topics").
		Where(squirrel.Eq{"site_id": siteID}).
		MustSql()

	err := r.db.QueryRowContext(ctx, totalQuery, totalArgs...).Scan(&stats.TotalTopics)
	stats.ActiveTopics = stats.TotalTopics // All topics are now considered active
	if err != nil {
		return nil, fmt.Errorf("failed to get topic counts: %w", err)
	}

	// Get used and unused topics count
	usageQuery, usageArgs := builder.
		Select("COALESCE(SUM(CASE WHEN usage_count > 0 THEN 1 ELSE 0 END), 0) as used, COALESCE(SUM(CASE WHEN usage_count = 0 THEN 1 ELSE 0 END), 0) as unused").
		From("site_topics").
		Where(squirrel.Eq{"site_id": siteID}).
		MustSql()

	err = r.db.QueryRowContext(ctx, usageQuery, usageArgs...).Scan(&stats.UsedTopics, &stats.UnusedTopics)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage counts: %w", err)
	}

	stats.UniqueTopicsLeft = stats.UnusedTopics

	// Get most used topic
	mostUsedQuery, mostUsedArgs := builder.
		Select("id", "usage_count").
		From("site_topics").
		Where(squirrel.Eq{"site_id": siteID}).
		OrderBy("usage_count DESC").
		Limit(1).
		MustSql()

	err = r.db.QueryRowContext(ctx, mostUsedQuery, mostUsedArgs...).Scan(&stats.MostUsedTopicID, &stats.MostUsedTopicCount)
	if err != nil && err.Error() != "sql: no rows in result set" {
		return nil, fmt.Errorf("failed to get most used topic: %w", err)
	}

	// Get last used topic
	lastUsedQuery, lastUsedArgs := builder.
		Select("id", "last_used_at").
		From("site_topics").
		Where(squirrel.Eq{"site_id": siteID}).
		Where(squirrel.Gt{"usage_count": 0}).
		OrderBy("last_used_at DESC").
		Limit(1).
		MustSql()

	err = r.db.QueryRowContext(ctx, lastUsedQuery, lastUsedArgs...).Scan(&stats.LastUsedTopicID, &stats.LastUsedAt)
	if err != nil && err.Error() != "sql: no rows in result set" {
		return nil, fmt.Errorf("failed to get last used topic: %w", err)
	}

	// Get current round-robin position (average or most recent)
	rrQuery, rrArgs := builder.
		Select("COALESCE(AVG(round_robin_pos), 0)").
		From("site_topics").
		Where(squirrel.Eq{"site_id": siteID}).
		MustSql()

	var avgPos float64
	err = r.db.QueryRowContext(ctx, rrQuery, rrArgs...).Scan(&avgPos)
	if err != nil {
		return nil, fmt.Errorf("failed to get round-robin position: %w", err)
	}
	stats.RoundRobinPosition = int(avgPos)

	return stats, nil
}

func (r *Repository) CreateTopicUsage(ctx context.Context, usage *models.TopicUsage) (*models.TopicUsage, error) {
	query, args := builder.
		Insert("topic_usage").
		Columns(
			"site_id",
			"topic_id",
			"article_id",
			"strategy",
			"used_at",
			"created_at",
		).
		Values(
			usage.SiteID,
			usage.TopicID,
			usage.ArticleID,
			usage.Strategy,
			usage.UsedAt,
			usage.CreatedAt,
		).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to create topic usage: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get insert id: %w", err)
	}

	usage.ID = id
	return usage, nil
}

func (r *Repository) GetTopicUsageHistory(ctx context.Context, siteID int64, topicID int64, limit int, offset int) (*models.PaginationResult[*models.TopicUsage], error) {
	query, args := builder.
		Select(
			"id",
			"site_id",
			"topic_id",
			"article_id",
			"strategy",
			"used_at",
			"created_at",
		).
		From("topic_usage").
		Where(squirrel.Eq{"site_id": siteID, "topic_id": topicID}).
		OrderBy("used_at DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query topic usage history: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var usages []*models.TopicUsage
	for rows.Next() {
		var usage models.TopicUsage
		if err = rows.Scan(
			&usage.ID,
			&usage.SiteID,
			&usage.TopicID,
			&usage.ArticleID,
			&usage.Strategy,
			&usage.UsedAt,
			&usage.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan topic usage: %w", err)
		}
		usages = append(usages, &usage)
	}

	// Get total count
	countQuery, countArgs := builder.
		Select("COUNT(id)").
		From("topic_usage").
		Where(squirrel.Eq{"site_id": siteID, "topic_id": topicID}).
		MustSql()

	var total int
	err = r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count topic usage: %w", err)
	}

	return &models.PaginationResult[*models.TopicUsage]{
		Data:   usages,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (r *Repository) GetSiteUsageHistory(ctx context.Context, siteID int64, limit int, offset int) (*models.PaginationResult[*models.TopicUsage], error) {
	query, args := builder.
		Select(
			"id",
			"site_id",
			"topic_id",
			"article_id",
			"strategy",
			"used_at",
			"created_at",
		).
		From("topic_usage").
		Where(squirrel.Eq{"site_id": siteID}).
		OrderBy("used_at DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query site usage history: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var usages []*models.TopicUsage
	for rows.Next() {
		var usage models.TopicUsage
		if err = rows.Scan(
			&usage.ID,
			&usage.SiteID,
			&usage.TopicID,
			&usage.ArticleID,
			&usage.Strategy,
			&usage.UsedAt,
			&usage.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan topic usage: %w", err)
		}
		usages = append(usages, &usage)
	}

	// Get total count
	countQuery, countArgs := builder.
		Select("COUNT(id)").
		From("topic_usage").
		Where(squirrel.Eq{"site_id": siteID}).
		MustSql()

	var total int
	err = r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count site usage: %w", err)
	}

	return &models.PaginationResult[*models.TopicUsage]{
		Data:   usages,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (r *Repository) RecordTopicUsage(ctx context.Context, siteID, topicID, articleID int64, strategy string) error {
	usage := &models.TopicUsage{
		SiteID:    siteID,
		TopicID:   topicID,
		ArticleID: articleID,
		Strategy:  strategy,
		UsedAt:    time.Now(),
		CreatedAt: time.Now(),
	}

	_, err := r.CreateTopicUsage(ctx, usage)
	return err
}

func (r *Repository) GetTopicByTitle(ctx context.Context, title string) (*models.Topic, error) {
	query, args := builder.
		Select("id", "title", "keywords", "category", "tags", "created_at", "updated_at").
		From("topics").
		Where(squirrel.Eq{"title": title}).
		Limit(1).
		MustSql()

	var topic models.Topic
	err := r.db.QueryRowContext(ctx, query, args...).
		Scan(
			&topic.ID,
			&topic.Title,
			&topic.Keywords,
			&topic.Category,
			&topic.Tags,
			&topic.CreatedAt,
			&topic.UpdatedAt,
		)

	if err != nil {
		return nil, fmt.Errorf("failed to get topic by title: %w", err)
	}

	return &topic, nil
}

func (r *Repository) BulkCreateTopicsWithSiteBinding(ctx context.Context, siteID int64, topics []*models.Topic) ([]*models.Topic, error) {
	if len(topics) == 0 {
		return nil, nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			_ = tx.Commit()
		}
	}()

	createdTopics := make([]*models.Topic, 0, len(topics))

	for _, topic := range topics {
		// Insert topic
		query, args := builder.
			Insert("topics").
			Columns("title", "keywords", "category", "tags", "created_at", "updated_at").
			Values(topic.Title, topic.Keywords, topic.Category, topic.Tags, time.Now(), time.Now()).
			Suffix("RETURNING id, title, keywords, category, tags, created_at, updated_at").
			MustSql()

		var createdTopic models.Topic
		err = tx.QueryRowContext(ctx, query, args...).Scan(
			&createdTopic.ID,
			&createdTopic.Title,
			&createdTopic.Keywords,
			&createdTopic.Category,
			&createdTopic.Tags,
			&createdTopic.CreatedAt,
			&createdTopic.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create topic '%s': %w", topic.Title, err)
		}

		// Create site-topic binding
		siteTopicQuery, siteTopicArgs := builder.
			Insert("site_topics").
			Columns("site_id", "topic_id", "priority", "created_at", "updated_at").
			Values(siteID, createdTopic.ID, 1, time.Now(), time.Now()).
			MustSql()

		_, err = tx.ExecContext(ctx, siteTopicQuery, siteTopicArgs...)
		if err != nil {
			return nil, fmt.Errorf("failed to create site-topic binding for topic '%s': %w", createdTopic.Title, err)
		}

		createdTopics = append(createdTopics, &createdTopic)
	}

	return createdTopics, nil
}

func (r *Repository) ReassignTopicsToSite(ctx context.Context, fromSiteID, toSiteID int64, topicIDs []int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			_ = tx.Commit()
		}
	}()

	var whereClause squirrel.Sqlizer = squirrel.Eq{"site_id": fromSiteID}
	if len(topicIDs) > 0 {
		whereClause = squirrel.And{
			squirrel.Eq{"site_id": fromSiteID},
			squirrel.Eq{"topic_id": topicIDs},
		}
	}

	// Get existing site-topic relationships to reassign
	selectQuery, selectArgs := builder.
		Select("topic_id", "priority").
		From("site_topics").
		Where(whereClause).
		MustSql()

	rows, err := tx.QueryContext(ctx, selectQuery, selectArgs...)
	if err != nil {
		return fmt.Errorf("failed to query existing site topics: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	type topicAssignment struct {
		TopicID  int64
		Priority int
	}

	var assignments []topicAssignment
	for rows.Next() {
		var assignment topicAssignment
		if err = rows.Scan(&assignment.TopicID, &assignment.Priority); err != nil {
			return fmt.Errorf("failed to scan topic assignment: %w", err)
		}
		assignments = append(assignments, assignment)
	}

	if len(assignments) == 0 {
		return nil // Nothing to reassign
	}

	// Delete existing assignments from the source site
	deleteQuery, deleteArgs := builder.
		Delete("site_topics").
		Where(whereClause).
		MustSql()

	_, err = tx.ExecContext(ctx, deleteQuery, deleteArgs...)
	if err != nil {
		return fmt.Errorf("failed to delete existing site topics: %w", err)
	}

	// Create new assignments for the target site
	insertBuilder := builder.Insert("site_topics").
		Columns("site_id", "topic_id", "priority", "created_at", "updated_at")

	for _, assignment := range assignments {
		insertBuilder = insertBuilder.Values(toSiteID, assignment.TopicID, assignment.Priority, time.Now(), time.Now())
	}

	insertQuery, insertArgs := insertBuilder.MustSql()
	_, err = tx.ExecContext(ctx, insertQuery, insertArgs...)
	if err != nil {
		return fmt.Errorf("failed to create new site topic assignments: %w", err)
	}

	return nil
}
