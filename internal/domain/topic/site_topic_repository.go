package topic

import (
	"Postulator/internal/infra/database"
	"Postulator/pkg/dbx"
	"Postulator/pkg/errors"
	"Postulator/pkg/logger"
	"context"

	"github.com/Masterminds/squirrel"
)

var _ SiteTopicRepository = (*siteTopicRepository)(nil)

type siteTopicRepository struct {
	db     *database.DB
	logger *logger.Logger
}

func NewSiteTopicRepository(db *database.DB, logger *logger.Logger) SiteTopicRepository {
	return &siteTopicRepository{
		db: db,
		logger: logger.
			WithScope("repository").
			WithScope("site_topics"),
	}
}

func (r *siteTopicRepository) Assign(ctx context.Context, siteID, topicID int64) error {
	query, args := dbx.ST.
		Insert("site_topics").
		Columns("site_id", "topic_id").
		Values(siteID, topicID).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsUniqueViolation(err):
		return errors.AlreadyExists("site_topic_assignment")
	case dbx.IsForeignKeyViolation(err):
		return errors.Validation("Invalid site or topic ID")
	case err != nil:
		return errors.Database(err)
	}

	return nil
}

func (r *siteTopicRepository) Unassign(ctx context.Context, siteID, topicID int64) error {
	query, args := dbx.ST.
		Delete("site_topics").
		Where(squirrel.Eq{"site_id": siteID, "topic_id": topicID}).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Database(err)
	}

	if rowsAffected == 0 {
		return errors.NotFound("site_topic_assignment", nil)
	}

	return nil
}

func (r *siteTopicRepository) GetBySiteID(ctx context.Context, siteID int64) ([]*Topic, error) {
	query, args := dbx.ST.
		Select("t.id", "t.title", "t.created_at").
		From("topics t").
		Join("site_topics st ON t.id = st.topic_id").
		Where(squirrel.Eq{"st.site_id": siteID}).
		Where(squirrel.Eq{"t.deleted_at": nil}).
		OrderBy("t.created_at DESC").
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Database(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var topics []*Topic
	for rows.Next() {
		var topic Topic
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

func (r *siteTopicRepository) IsAssigned(ctx context.Context, siteID, topicID int64) (bool, error) {
	query, args := dbx.ST.
		Select("COUNT(id)").
		From("site_topics").
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
