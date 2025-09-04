package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/Masterminds/squirrel"
)

var psql = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question)

// Setting represents a setting in the database
type Setting struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

// settingRepository implements SettingRepository interface
type settingRepository struct {
	db *sql.DB
}

// Get retrieves a setting by its key
func (r *settingRepository) Get(ctx context.Context, key string) (*Setting, error) {
	query := psql.Select("key", "value", "updated_at").
		From("settings").
		Where(squirrel.Eq{"key": key})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	var setting Setting
	err = r.db.QueryRowContext(ctx, sqlStr, args...).Scan(
		&setting.Key,
		&setting.Value,
		&setting.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &setting, nil
}

// Set creates or updates a setting
func (r *settingRepository) Set(ctx context.Context, key, value string) error {
	// Try to update first
	updateQuery := psql.Update("settings").
		Set("value", value).
		Set("updated_at", "CURRENT_TIMESTAMP").
		Where(squirrel.Eq{"key": key})

	sqlStr, args, err := updateQuery.ToSql()
	if err != nil {
		return err
	}

	result, err := r.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	// If no rows were affected, insert new setting
	if rowsAffected == 0 {
		insertQuery := psql.Insert("settings").
			Columns("key", "value").
			Values(key, value)

		sqlStr, args, err := insertQuery.ToSql()
		if err != nil {
			return err
		}

		_, err = r.db.ExecContext(ctx, sqlStr, args...)
		return err
	}

	return nil
}

// GetAll retrieves all settings
func (r *settingRepository) GetAll(ctx context.Context) ([]*Setting, error) {
	query := psql.Select("key", "value", "updated_at").
		From("settings").
		OrderBy("key ASC")

	sqlStr, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settings []*Setting
	for rows.Next() {
		var setting Setting
		err := rows.Scan(
			&setting.Key,
			&setting.Value,
			&setting.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		settings = append(settings, &setting)
	}

	return settings, nil
}

// Delete removes a setting
func (r *settingRepository) Delete(ctx context.Context, key string) error {
	query := psql.Delete("settings").Where(squirrel.Eq{"key": key})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, sqlStr, args...)
	return err
}

// GetByPrefix retrieves all settings with keys starting with the given prefix
func (r *settingRepository) GetByPrefix(ctx context.Context, prefix string) ([]*Setting, error) {
	query := psql.Select("key", "value", "updated_at").
		From("settings").
		Where(squirrel.Like{"key": prefix + "%"}).
		OrderBy("key ASC")

	sqlStr, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settings []*Setting
	for rows.Next() {
		var setting Setting
		err := rows.Scan(
			&setting.Key,
			&setting.Value,
			&setting.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		settings = append(settings, &setting)
	}

	return settings, nil
}

// Legacy functions for backward compatibility
func GetSetting(key string) (*Setting, error) {
	ctx := context.Background()
	repo := &settingRepository{db: GetDB()}
	return repo.Get(ctx, key)
}

func SetSetting(key, value string) error {
	ctx := context.Background()
	repo := &settingRepository{db: GetDB()}
	return repo.Set(ctx, key, value)
}
