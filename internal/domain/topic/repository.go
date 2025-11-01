package topic

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/infra/database"
	"github.com/davidmovas/postulator/pkg/dbx"
	"github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"

	"github.com/Masterminds/squirrel"
)

var _ Repository = (*repository)(nil)

type repository struct {
	db     *database.DB
	logger *logger.Logger
}

func NewRepository(db *database.DB, logger *logger.Logger) Repository {
	return &repository{
		db: db,
		logger: logger.
			WithScope("repository").
			WithScope("topics"),
	}
}

func (r *repository) Create(ctx context.Context, topic *Topic) (*Topic, error) {
	query, args := dbx.ST.
		Insert("topics").
		Columns("title").
		Values(topic.Title).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsUniqueViolation(err):
		return nil, errors.AlreadyExists("topic")
	case err != nil:
		return nil, errors.Database(err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, errors.Database(err)
	}

	topic.ID = id
	topic.CreatedAt = time.Now()
	return topic, nil
}

func (r *repository) CreateBatch(ctx context.Context, topics ...*Topic) (*BatchResult, error) {
	if len(topics) == 0 {
		return &BatchResult{}, nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Database(err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	result := &BatchResult{
		SkippedTitles: make([]string, 0),
	}

	for _, topic := range topics {
		query, args := dbx.ST.
			Insert("topics").
			Columns("title").
			Values(topic.Title).
			MustSql()

		_, err = tx.ExecContext(ctx, query, args...)
		switch {
		case dbx.IsUniqueViolation(err):
			result.Skipped++
			result.SkippedTitles = append(result.SkippedTitles, topic.Title)
		case err != nil:
			return nil, errors.Database(err)
		default:
			result.Created++
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, errors.Database(err)
	}

	return result, nil
}

func (r *repository) GetByID(ctx context.Context, id int64) (*Topic, error) {
	query, args := dbx.ST.
		Select("id", "title", "created_at").
		From("topics").
		Where(squirrel.Eq{"id": id}).
		Where(squirrel.Eq{"deleted_at": nil}).
		MustSql()

	var topic Topic
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&topic.ID,
		&topic.Title,
		&topic.CreatedAt,
	)

	switch {
	case dbx.IsNoRows(err):
		return nil, errors.NotFound("topic", id)
	case err != nil:
		return nil, errors.Database(err)
	}

	return &topic, nil
}

func (r *repository) GetAll(ctx context.Context) ([]*Topic, error) {
	query, args := dbx.ST.
		Select("id", "title", "created_at").
		From("topics").
		Where(squirrel.Eq{"deleted_at": nil}).
		OrderBy("created_at DESC").
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

func (r *repository) GetByTitle(ctx context.Context, title string) (*Topic, error) {
	query, args := dbx.ST.
		Select("id", "title", "created_at").
		From("topics").
		Where(squirrel.Eq{"title": title}).
		Where(squirrel.Eq{"deleted_at": nil}).
		MustSql()

	var topic Topic
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&topic.ID,
		&topic.Title,
		&topic.CreatedAt,
	)

	switch {
	case dbx.IsNoRows(err):
		return nil, errors.NotFound("topic", title)
	case err != nil:
		return nil, errors.Database(err)
	}

	return &topic, nil
}

func (r *repository) Update(ctx context.Context, topic *Topic) error {
	query, args := dbx.ST.
		Update("topics").
		Set("title", topic.Title).
		Where(squirrel.Eq{"id": topic.ID}).
		Where(squirrel.Eq{"deleted_at": nil}).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsUniqueViolation(err):
		return errors.AlreadyExists("topic")
	case err != nil:
		return errors.Database(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Database(err)
	}

	if rowsAffected == 0 {
		return errors.NotFound("topic", topic.ID)
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	query, args := dbx.ST.
		Update("topics").
		Set("deleted_at", time.Now()).
		Where(squirrel.Eq{"id": id}).
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
		return errors.NotFound("topic", id)
	}

	return nil
}

func (r *repository) Count(ctx context.Context) (int, error) {
	query, args := dbx.ST.
		Select("COUNT(id)").
		From("topics").
		Where(squirrel.Eq{"deleted_at": nil}).
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
