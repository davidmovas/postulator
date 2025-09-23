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

func (r *Repository) GetActiveTopics(ctx context.Context) ([]*models.Topic, error) {
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
		OrderBy("title").
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

func (r *Repository) GetSiteTopicsForSelection(ctx context.Context, siteID int64, _ string) ([]*models.SiteTopic, error) {
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
		Where(squirrel.Eq{"site_id": siteID}).
		OrderBy("priority DESC, created_at ASC"). // Higher priority first, then by creation time
		MustSql()

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

	if strategy == string(models.StrategyRoundRobin) {
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
	}

	now := time.Now()
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
