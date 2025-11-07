package settings

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/infra/database"
	"github.com/davidmovas/postulator/pkg/dbx"
	"github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"

	"github.com/Masterminds/squirrel"
)

type repository struct {
	db     *database.DB
	logger *logger.Logger
}

func NewRepository(db *database.DB, logger *logger.Logger) Repository {
	return &repository{
		db: db,
		logger: logger.
			WithScope("repository").
			WithScope("settings"),
	}
}

func (r *repository) Get(ctx context.Context, key string) (string, error) {
	query, args := dbx.ST.
		Select("value").
		From("app_settings").
		Where(squirrel.Eq{"key": key}).
		MustSql()

	var value string
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&value)

	switch {
	case dbx.IsNoRows(err):
		return "", errors.NotFound("setting", key)
	case err != nil:
		return "", errors.Database(err)
	}

	return value, nil
}

func (r *repository) Set(ctx context.Context, key string, value string) error {
	exists, err := r.Exists(ctx, key)
	if err != nil {
		return err
	}

	if exists {
		query, args := dbx.ST.
			Update("app_settings").
			Set("value", value).
			Set("updated_at", time.Now()).
			Where(squirrel.Eq{"key": key}).
			MustSql()

		_, err = r.db.ExecContext(ctx, query, args...)
		if err != nil {
			return errors.Database(err)
		}
	} else {
		query, args := dbx.ST.
			Insert("app_settings").
			Columns("key", "value", "updated_at").
			Values(key, value, time.Now()).
			MustSql()

		_, err = r.db.ExecContext(ctx, query, args...)
		if err != nil {
			return errors.Database(err)
		}
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, key string) error {
	query, args := dbx.ST.
		Delete("app_settings").
		Where(squirrel.Eq{"key": key}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	return nil
}

func (r *repository) Exists(ctx context.Context, key string) (bool, error) {
	query, args := dbx.ST.
		Select("COUNT(key)").
		From("app_settings").
		Where(squirrel.Eq{"key": key}).
		MustSql()

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return false, errors.Database(err)
	}

	return count > 0, nil
}
