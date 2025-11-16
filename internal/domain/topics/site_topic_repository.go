package topics

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/infra/database"
	"github.com/davidmovas/postulator/pkg/dbx"
	"github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"

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

func (r *siteTopicRepository) GetBySiteID(ctx context.Context, siteID int64) ([]*entities.Topic, error) {
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

func (r *siteTopicRepository) GetAssignedForSite(ctx context.Context, siteID int64, topicIDs []int64) ([]int64, error) {
	if len(topicIDs) == 0 {
		return []int64{}, nil
	}

	// de-duplicate ids
	idSet := make(map[int64]struct{}, len(topicIDs))
	uniq := make([]int64, 0, len(topicIDs))
	for _, id := range topicIDs {
		if _, ok := idSet[id]; ok {
			continue
		}
		idSet[id] = struct{}{}
		uniq = append(uniq, id)
	}

	const chunkSize = 1000
	var assigned []int64

	for start := 0; start < len(uniq); start += chunkSize {
		end := start + chunkSize
		if end > len(uniq) {
			end = len(uniq)
		}
		chunk := uniq[start:end]

		query, args := dbx.ST.
			Select("topic_id").
			From("site_topics").
			Where(squirrel.Eq{"site_id": siteID}).
			Where(squirrel.Eq{"topic_id": chunk}).
			MustSql()

		rows, err := r.db.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, errors.Database(err)
		}
		func() {
			defer func() { _ = rows.Close() }()
			for rows.Next() {
				var id int64
				if err = rows.Scan(&id); err != nil {
					assigned = nil
					return
				}
				assigned = append(assigned, id)
			}
		}()
		if assigned == nil && start < len(uniq) {
			return nil, errors.Database(nil)
		}
	}

	return assigned, nil
}
