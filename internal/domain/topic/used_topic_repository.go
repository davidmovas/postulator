package topic

import (
	"Postulator/internal/domain/entities"
	"Postulator/internal/infra/database"
	"Postulator/pkg/dbx"
	"Postulator/pkg/di"
	"Postulator/pkg/errors"
	"Postulator/pkg/logger"
	"context"

	"github.com/Masterminds/squirrel"
)

var _ IUsedTopicRepository = (*UsedTopicRepository)(nil)

type UsedTopicRepository struct {
	db     *database.DB
	logger *logger.Logger
}

func NewUsedTopicRepository(c di.Container) (*UsedTopicRepository, error) {
	var db *database.DB
	if err := c.Resolve(&db); err != nil {
		return nil, err
	}

	var l *logger.Logger
	if err := c.Resolve(&l); err != nil {
		return nil, err
	}

	return &UsedTopicRepository{
		db:     db,
		logger: l,
	}, nil
}

func (r *UsedTopicRepository) MarkAsUsed(ctx context.Context, siteID, topicID int64) error {
	query, args := dbx.ST.
		Insert("used_topics").
		Columns("site_id", "topic_id").
		Values(siteID, topicID).
		Suffix("ON CONFLICT DO NOTHING").
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Internal(err)
	}

	return nil
}

func (r *UsedTopicRepository) IsUsed(ctx context.Context, siteID, topicID int64) (bool, error) {
	query, args := dbx.ST.
		Select("COUNT(*)").
		From("used_topics").
		Where(squirrel.Eq{"site_id": siteID, "topic_id": topicID}).
		MustSql()

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return false, errors.Internal(err)
	}

	return count > 0, nil
}

func (r *UsedTopicRepository) GetUnusedTopics(ctx context.Context, siteID int64) ([]*entities.Topic, error) {
	query, args := dbx.ST.
		Select("t.id", "t.title", "t.created_at").
		From("topics t").
		Join("site_topics st ON t.id = st.topic_id").
		LeftJoin("used_topics ut ON t.id = ut.topic_id AND ut.site_id = ?", siteID).
		Where(squirrel.Eq{"st.site_id": siteID}).
		Where(squirrel.Eq{"ut.id": nil}).
		OrderBy("t.created_at ASC").
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Internal(err)
	}

	defer func() {
		_ = rows.Close()
	}()

	var topics []*entities.Topic
	for rows.Next() {
		var topic entities.Topic
		if err = rows.Scan(
			&topic.ID,
			&topic.Title,
			&topic.CreatedAt,
		); err != nil {
			return nil, errors.Internal(err)
		}

		topics = append(topics, &topic)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.Internal(err)
	}

	return topics, nil
}

func (r *UsedTopicRepository) CountUnusedTopics(ctx context.Context, siteID int64) (int, error) {
	query, args := dbx.ST.
		Select("COUNT(*)").
		From("topics t").
		Join("site_topics st ON t.id = st.topic_id").
		LeftJoin("used_topics ut ON t.id = ut.topic_id AND ut.site_id = ?", siteID).
		Where(squirrel.Eq{"st.site_id": siteID}).
		Where(squirrel.Eq{"ut.id": nil}).
		MustSql()

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, errors.Internal(err)
	}

	return count, nil
}
