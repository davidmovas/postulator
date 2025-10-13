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

var _ ITopicRepository = (*Repository)(nil)

type Repository struct {
	db     *database.DB
	logger *logger.Logger
}

func NewTopicRepository(c di.Container) (*Repository, error) {
	var db *database.DB
	if err := c.Resolve(&db); err != nil {
		return nil, err
	}

	var l *logger.Logger
	if err := c.Resolve(&l); err != nil {
		return nil, err
	}

	return &Repository{
		db:     db,
		logger: l,
	}, nil
}

func (r *Repository) Create(ctx context.Context, topic *entities.Topic) (int, error) {
	query, args := dbx.ST.
		Insert("topics").
		Columns("title").
		Values(topic.Title).
		Suffix("RETURNING id").
		MustSql()

	var id int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&id)

	switch {
	case dbx.IsUniqueViolation(err):
		return 0, errors.AlreadyExists(topic.Title)
	case err != nil:
		return 0, errors.Internal(err)
	}

	return id, nil
}

func (r *Repository) CreateBatch(ctx context.Context, topics []*entities.Topic) (*BatchCreateResult, error) {
	result := &BatchCreateResult{
		Created: make([]string, 0),
		Skipped: make([]string, 0),
	}

	for _, topic := range topics {
		query, args := dbx.ST.
			Insert("topics").
			Columns("title").
			Values(topic.Title).
			MustSql()

		_, err := r.db.ExecContext(ctx, query, args...)
		switch {
		case dbx.IsUniqueViolation(err):
			result.Skipped = append(result.Skipped, topic.Title)
			result.TotalSkipped++
			r.logger.Debugf("Topic with title '%s' already exists, skipping", topic.Title)
		case err != nil:
			return nil, errors.Internal(err)
		default:
			result.Created = append(result.Created, topic.Title)
			result.TotalAdded++
		}
	}

	return result, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*entities.Topic, error) {
	query, args := dbx.ST.
		Select("id", "title", "created_at").
		From("topics").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	var topic entities.Topic
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&topic.ID,
		&topic.Title,
		&topic.CreatedAt,
	)

	switch {
	case dbx.IsNoRows(err):
		return nil, errors.NotFound("topic", id)
	case err != nil:
		return nil, errors.Internal(err)
	}

	return &topic, nil
}

func (r *Repository) GetAll(ctx context.Context) ([]*entities.Topic, error) {
	query, args := dbx.ST.
		Select("id", "title", "created_at").
		From("topics").
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

	return topics, nil
}

func (r *Repository) Update(ctx context.Context, topic *entities.Topic) error {
	query, args := dbx.ST.
		Update("topics").
		Set("title", topic.Title).
		Where(squirrel.Eq{"id": topic.ID}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsNoRows(err):
		return errors.NotFound("topic", topic.ID)
	case dbx.IsUniqueViolation(err):
		return errors.AlreadyExists(topic.Title)
	case err != nil:
		return errors.Internal(err)
	}

	return nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	query, args := dbx.ST.
		Delete("topics").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsNoRows(err):
		return errors.NotFound("topic", id)
	case err != nil:
		return errors.Internal(err)
	}

	return nil
}
