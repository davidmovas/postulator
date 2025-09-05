package repository

import (
	"context"
	"database/sql"

	"github.com/Masterminds/squirrel"
	br "github.com/lann/builder"
)

var (
	_ SiteRepository = (*Repository)(nil)
)

var builder = squirrel.StatementBuilderType(br.EmptyBuilder).PlaceholderFormat(squirrel.Dollar)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) createEntity(ctx context.Context, query string, args ...interface{}) (int64, error) {
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *Repository) updateEntity(ctx context.Context, query string, args ...interface{}) error {
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *Repository) deleteEntity(ctx context.Context, query string, args ...interface{}) error {
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *Repository) queryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return r.db.QueryRowContext(ctx, query, args...)
}

func (r *Repository) query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return r.db.QueryContext(ctx, query, args...)
}

func (r *Repository) count(ctx context.Context, query string, args ...interface{}) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}
