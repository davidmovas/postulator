package topic

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/infra/database"
	"github.com/davidmovas/postulator/pkg/dbx"
	"github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"

	"github.com/Masterminds/squirrel"
)

var _ UsageRepository = (*usageRepository)(nil)

type usageRepository struct {
	db     *database.DB
	logger *logger.Logger
}

func NewUsageRepository(db *database.DB, logger *logger.Logger) UsageRepository {
	return &usageRepository{
		db: db,
		logger: logger.
			WithScope("repository").
			WithScope("used_topics"),
	}
}

func (r *usageRepository) MarkAsUsed(ctx context.Context, siteID, topicID int64) error {
	query, args := dbx.ST.
		Insert("used_topics").
		Columns("site_id", "topic_id").
		Values(siteID, topicID).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsUniqueViolation(err):
		return nil
	case dbx.IsForeignKeyViolation(err):
		return errors.Validation("Invalid site or topic ID")
	case err != nil:
		return errors.Database(err)
	}

	return nil
}

func (r *usageRepository) IsUsed(ctx context.Context, siteID, topicID int64) (bool, error) {
	query, args := dbx.ST.
		Select("COUNT(id)").
		From("used_topics").
		Where(squirrel.Eq{"site_id": siteID, "topic_id": topicID}).
		MustSql()

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	switch {
	case dbx.IsNoRows(err):
		return false, nil
	case err != nil:
		return false, errors.Database(err)
	}

	return count > 0, nil
}

func (r *usageRepository) GetUnused(ctx context.Context, siteID int64, topicIDs []int64) ([]*entities.Topic, error) {
	if len(topicIDs) == 0 {
		return []*entities.Topic{}, nil
	}

	query, args := dbx.ST.
		Select("t.id", "t.title", "t.created_at").
		From("topics t").
		LeftJoin("used_topics ut ON t.id = ut.topic_id AND ut.site_id = ?", siteID).
		Where(squirrel.Eq{"t.id": topicIDs}).
		Where(squirrel.Eq{"t.deleted_at": nil}).
		Where(squirrel.Eq{"ut.id": nil}).
		OrderBy("t.created_at DESC").
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Database(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var topics []*entities.Topic
	for rows.Next() {
		var topic entities.Topic
		err = rows.Scan(
			&topic.ID,
			&topic.Title,
			&topic.CreatedAt,
		)
		if err != nil {
			return nil, errors.Database(err)
		}
		topics = append(topics, &topic)
	}

	switch {
	case dbx.IsNoRows(err) || len(topics) == 0:
		return topics, nil
	case err != nil || rows.Err() != nil:
		return nil, errors.Database(err)
	}

	return topics, nil
}

func (r *usageRepository) CountUnused(ctx context.Context, siteID int64, topicIDs []int64) (int, error) {
	if len(topicIDs) == 0 {
		return 0, nil
	}

	query, args := dbx.ST.
		Select("COUNT(t.id)").
		From("topics t").
		LeftJoin("used_topics ut ON t.id = ut.topic_id AND ut.site_id = ?", siteID).
		Where(squirrel.Eq{"t.id": topicIDs}).
		Where(squirrel.Eq{"t.deleted_at": nil}).
		Where(squirrel.Eq{"ut.id": nil}).
		MustSql()

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	switch {
	case dbx.IsNoRows(err):
		return 0, nil
	case err != nil:
		return 0, errors.Database(err)
	}

	return count, nil
}

func (r *usageRepository) GetNextUnused(ctx context.Context, siteID int64, topicIDs []int64) (*entities.Topic, error) {
	if len(topicIDs) == 0 {
		return nil, errors.NotFound("unused_topic", nil)
	}

	query, args := dbx.ST.
		Select("t.id", "t.title", "t.created_at").
		From("topics t").
		LeftJoin("used_topics ut ON t.id = ut.topic_id AND ut.site_id = ?", siteID).
		Where(squirrel.Eq{"t.id": topicIDs}).
		Where(squirrel.Eq{"t.deleted_at": nil}).
		Where(squirrel.Eq{"ut.id": nil}).
		OrderBy("t.created_at ASC").
		Limit(1).
		MustSql()

	var topic entities.Topic
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&topic.ID,
		&topic.Title,
		&topic.CreatedAt,
	)

	switch {
	case dbx.IsNoRows(err):
		return nil, errors.NotFound("unused_topic", nil)
	case err != nil:
		return nil, errors.Database(err)
	}

	return &topic, nil
}
