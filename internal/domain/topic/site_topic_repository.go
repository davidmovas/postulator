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

var _ ISiteTopicRepository = (*SiteTopicRepository)(nil)

type SiteTopicRepository struct {
	db     *database.DB
	logger *logger.Logger
}

func NewSiteTopicRepository(c di.Container) (*SiteTopicRepository, error) {
	var db *database.DB
	if err := c.Resolve(&db); err != nil {
		return nil, err
	}

	var l *logger.Logger
	if err := c.Resolve(&l); err != nil {
		return nil, err
	}

	return &SiteTopicRepository{
		db:     db,
		logger: l,
	}, nil
}

func (r *SiteTopicRepository) Assign(ctx context.Context, st *entities.SiteTopic) error {
	query, args := dbx.ST.
		Insert("site_topics").
		Columns("site_id", "topic_id", "category_id", "strategy").
		Values(st.SiteID, st.TopicID, st.CategoryID, st.Strategy).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsUniqueViolation(err):
		return errors.AlreadyExists("site_topic")
	case err != nil:
		return errors.Internal(err)
	}

	return nil
}

func (r *SiteTopicRepository) GetBySiteID(ctx context.Context, siteID int64) ([]*entities.SiteTopic, error) {
	query, args := dbx.ST.
		Select("id", "site_id", "topic_id", "category_id", "strategy", "created_at").
		From("site_topics").
		Where(squirrel.Eq{"site_id": siteID}).
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Internal(err)
	}

	defer func() {
		_ = rows.Close()
	}()

	var siteTopics []*entities.SiteTopic
	for rows.Next() {
		var st entities.SiteTopic
		if err = rows.Scan(
			&st.ID,
			&st.SiteID,
			&st.TopicID,
			&st.CategoryID,
			&st.Strategy,
			&st.CreatedAt,
		); err != nil {
			return nil, errors.Internal(err)
		}

		siteTopics = append(siteTopics, &st)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.Internal(err)
	}

	return siteTopics, nil
}

func (r *SiteTopicRepository) GetByTopicID(ctx context.Context, topicID int64) ([]*entities.SiteTopic, error) {
	query, args := dbx.ST.
		Select("id", "site_id", "topic_id", "category_id", "strategy", "created_at").
		From("site_topics").
		Where(squirrel.Eq{"topic_id": topicID}).
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Internal(err)
	}

	defer func() {
		_ = rows.Close()
	}()

	var siteTopics []*entities.SiteTopic
	for rows.Next() {
		var st entities.SiteTopic
		if err = rows.Scan(
			&st.ID,
			&st.SiteID,
			&st.TopicID,
			&st.CategoryID,
			&st.Strategy,
			&st.CreatedAt,
		); err != nil {
			return nil, errors.Internal(err)
		}

		siteTopics = append(siteTopics, &st)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.Internal(err)
	}

	return siteTopics, nil
}

func (r *SiteTopicRepository) Update(ctx context.Context, st *entities.SiteTopic) error {
	query, args := dbx.ST.
		Update("site_topics").
		Set("category_id", st.CategoryID).
		Set("strategy", st.Strategy).
		Where(squirrel.Eq{"site_id": st.SiteID, "topic_id": st.TopicID}).
		MustSql()

	res, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case res != nil:
		var affected int64
		if affected, err = res.RowsAffected(); err == nil && affected == 0 {
			return errors.NotFound("site_topic", st.SiteID)
		}
		if err != nil {
			return errors.Internal(err)
		}
	case err != nil:
		return errors.Internal(err)
	}

	return nil
}

func (r *SiteTopicRepository) Unassign(ctx context.Context, siteID, topicID int64) error {
	query, args := dbx.ST.
		Delete("site_topics").
		Where(squirrel.Eq{"site_id": siteID, "topic_id": topicID}).
		MustSql()

	res, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case res != nil:
		var affected int64
		if affected, err = res.RowsAffected(); err == nil && affected == 0 {
			return errors.NotFound("site_topic", siteID)
		}
		if err != nil {
			return errors.Internal(err)
		}
	case err != nil:
		return errors.Internal(err)
	}

	return nil
}
