package repository

import (
	"Postulator/internal/models"
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
)

// Topic Repository Methods

func (r *Repository) GetTopics(ctx context.Context, limit int, offset int) (*models.PaginationResult[*models.Topic], error) {
	// Get topics
	query, args := builder.
		Select(
			"id",
			"title",
			"keywords",
			"category",
			"tags",
			"is_active",
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
			&topic.IsActive,
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
			"is_active",
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
			&topic.IsActive,
			&topic.CreatedAt,
			&topic.UpdatedAt,
		); err != nil {
		return nil, fmt.Errorf("failed to query topic: %w", err)
	}

	return &topic, nil
}

func (r *Repository) GetTopicsBySiteID(ctx context.Context, siteID int64, limit int, offset int) (*models.PaginationResult[*models.Topic], error) {
	// Get topics associated with a site through site_topics table
	query, args := builder.
		Select(
			"t.id",
			"t.title",
			"t.keywords",
			"t.category",
			"t.tags",
			"t.is_active",
			"t.created_at",
			"t.updated_at",
		).
		From("topics t").
		Join("site_topics st ON t.id = st.topic_id").
		Where(squirrel.Eq{"st.site_id": siteID}).
		Where(squirrel.Eq{"st.is_active": true}).
		OrderBy("t.created_at DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query topics by site: %w", err)
	}
	defer rows.Close()

	var topics []*models.Topic
	for rows.Next() {
		var topic models.Topic
		if err = rows.Scan(
			&topic.ID,
			&topic.Title,
			&topic.Keywords,
			&topic.Category,
			&topic.Tags,
			&topic.IsActive,
			&topic.CreatedAt,
			&topic.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan topic: %w", err)
		}
		topics = append(topics, &topic)
	}

	// Get total count
	countQuery, countArgs := builder.
		Select("COUNT(*)").
		From("topics t").
		Join("site_topics st ON t.id = st.topic_id").
		Where(squirrel.Eq{"st.site_id": siteID}).
		Where(squirrel.Eq{"st.is_active": true}).
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
			"is_active",
			"created_at",
			"updated_at",
		).
		Values(
			topic.Title,
			topic.Keywords,
			topic.Category,
			topic.Tags,
			topic.IsActive,
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
		Set("is_active", topic.IsActive).
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

func (r *Repository) ActivateTopic(ctx context.Context, id int64) error {
	query, args := builder.
		Update("topics").
		Set("is_active", true).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": id}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to activate topic: %w", err)
	}

	return nil
}

func (r *Repository) DeactivateTopic(ctx context.Context, id int64) error {
	query, args := builder.
		Update("topics").
		Set("is_active", false).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": id}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to deactivate topic: %w", err)
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
			"is_active",
			"created_at",
			"updated_at",
		).
		From("topics").
		Where(squirrel.Eq{"is_active": true}).
		OrderBy("title").
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query active topics: %w", err)
	}
	defer rows.Close()

	var topics []*models.Topic
	for rows.Next() {
		var topic models.Topic
		if err = rows.Scan(
			&topic.ID,
			&topic.Title,
			&topic.Keywords,
			&topic.Category,
			&topic.Tags,
			&topic.IsActive,
			&topic.CreatedAt,
			&topic.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan topic: %w", err)
		}
		topics = append(topics, &topic)
	}

	return topics, nil
}

// SiteTopic Repository Methods

func (r *Repository) CreateSiteTopic(ctx context.Context, siteTopic *models.SiteTopic) (*models.SiteTopic, error) {
	query, args := builder.
		Insert("site_topics").
		Columns(
			"site_id",
			"topic_id",
			"is_active",
		).
		Values(
			siteTopic.SiteID,
			siteTopic.TopicID,
			siteTopic.IsActive,
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
	// Get site topics with topic details
	query, args := builder.
		Select(
			"st.id",
			"st.site_id",
			"st.topic_id",
			"st.is_active",
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
	defer rows.Close()

	var siteTopics []*models.SiteTopic
	for rows.Next() {
		var siteTopic models.SiteTopic
		if err = rows.Scan(
			&siteTopic.ID,
			&siteTopic.SiteID,
			&siteTopic.TopicID,
			&siteTopic.IsActive,
		); err != nil {
			return nil, fmt.Errorf("failed to scan site topic: %w", err)
		}
		siteTopics = append(siteTopics, &siteTopic)
	}

	// Get total count
	countQuery, countArgs := builder.
		Select("COUNT(*)").
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
	// Get sites associated with a topic
	query, args := builder.
		Select(
			"st.id",
			"st.site_id",
			"st.topic_id",
			"st.is_active",
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
	defer rows.Close()

	var siteTopics []*models.SiteTopic
	for rows.Next() {
		var siteTopic models.SiteTopic
		if err = rows.Scan(
			&siteTopic.ID,
			&siteTopic.SiteID,
			&siteTopic.TopicID,
			&siteTopic.IsActive,
		); err != nil {
			return nil, fmt.Errorf("failed to scan site topic: %w", err)
		}
		siteTopics = append(siteTopics, &siteTopic)
	}

	// Get total count
	countQuery, countArgs := builder.
		Select("COUNT(*)").
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
			"is_active",
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
			&siteTopic.IsActive,
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
		Set("is_active", siteTopic.IsActive).
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

func (r *Repository) ActivateSiteTopic(ctx context.Context, id int64) error {
	query, args := builder.
		Update("site_topics").
		Set("is_active", true).
		Where(squirrel.Eq{"id": id}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to activate site topic: %w", err)
	}

	return nil
}

func (r *Repository) DeactivateSiteTopic(ctx context.Context, id int64) error {
	query, args := builder.
		Update("site_topics").
		Set("is_active", false).
		Where(squirrel.Eq{"id": id}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to deactivate site topic: %w", err)
	}

	return nil
}

// Topic Selection Strategy Methods

func (r *Repository) GetSiteTopicsForSelection(ctx context.Context, siteID int64, strategy string) ([]*models.SiteTopic, error) {
	query, args := builder.
		Select(
			"id",
			"site_id",
			"topic_id",
			"is_active",
			"priority",
			"last_used_at",
			"usage_count",
			"round_robin_pos",
			"created_at",
			"updated_at",
		).
		From("site_topics").
		Where(squirrel.Eq{"site_id": siteID, "is_active": true}).
		OrderBy("priority DESC, created_at ASC"). // Higher priority first, then by creation time
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query site topics for selection: %w", err)
	}
	defer rows.Close()

	var siteTopics []*models.SiteTopic
	for rows.Next() {
		var siteTopic models.SiteTopic
		if err = rows.Scan(
			&siteTopic.ID,
			&siteTopic.SiteID,
			&siteTopic.TopicID,
			&siteTopic.IsActive,
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
	// Update usage count, last used time, and round-robin position if needed
	updateQuery := builder.
		Update("site_topics").
		Set("usage_count", squirrel.Expr("usage_count + 1")).
		Set("last_used_at", time.Now()).
		Set("updated_at", time.Now())

	// For round-robin strategy, update the position
	if strategy == string(models.StrategyRoundRobin) {
		updateQuery = updateQuery.Set("round_robin_pos", squirrel.Expr("round_robin_pos + 1"))
	}

	query, args := updateQuery.Where(squirrel.Eq{"id": siteTopicID}).MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update site topic usage: %w", err)
	}

	return nil
}

func (r *Repository) GetTopicStats(ctx context.Context, siteID int64) (*models.TopicStats, error) {
	stats := &models.TopicStats{SiteID: siteID}

	// Get total and active topics count
	totalQuery, totalArgs := builder.
		Select("COUNT(*) as total, SUM(CASE WHEN is_active = 1 THEN 1 ELSE 0 END) as active").
		From("site_topics").
		Where(squirrel.Eq{"site_id": siteID}).
		MustSql()

	err := r.db.QueryRowContext(ctx, totalQuery, totalArgs...).Scan(&stats.TotalTopics, &stats.ActiveTopics)
	if err != nil {
		return nil, fmt.Errorf("failed to get topic counts: %w", err)
	}

	// Get used and unused topics count
	usageQuery, usageArgs := builder.
		Select("SUM(CASE WHEN usage_count > 0 THEN 1 ELSE 0 END) as used, SUM(CASE WHEN usage_count = 0 THEN 1 ELSE 0 END) as unused").
		From("site_topics").
		Where(squirrel.Eq{"site_id": siteID, "is_active": true}).
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
		Where(squirrel.Eq{"site_id": siteID, "is_active": true}).
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
		Where(squirrel.Eq{"site_id": siteID, "is_active": true}).
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
		Where(squirrel.Eq{"site_id": siteID, "is_active": true}).
		MustSql()

	var avgPos float64
	err = r.db.QueryRowContext(ctx, rrQuery, rrArgs...).Scan(&avgPos)
	if err != nil {
		return nil, fmt.Errorf("failed to get round-robin position: %w", err)
	}
	stats.RoundRobinPosition = int(avgPos)

	return stats, nil
}

// Topic Usage Repository Methods

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
	defer rows.Close()

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
		Select("COUNT(*)").
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
	defer rows.Close()

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
		Select("COUNT(*)").
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
