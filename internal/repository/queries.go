package repository

import (
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

// GetSetting retrieves a setting by its key
func GetSetting(key string) (*Setting, error) {
	query := psql.Select("key", "value", "updated_at").
		From("settings").
		Where(squirrel.Eq{"key": key})

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	var setting Setting
	err = db.QueryRow(sql, args...).Scan(
		&setting.Key,
		&setting.Value,
		&setting.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &setting, nil
}

// SetSetting creates or updates a setting
func SetSetting(key, value string) error {
	// Try to update first
	updateQuery := psql.Update("settings").
		Set("value", value).
		Set("updated_at", "CURRENT_TIMESTAMP").
		Where(squirrel.Eq{"key": key})

	sql, args, err := updateQuery.ToSql()
	if err != nil {
		return err
	}

	result, err := db.Exec(sql, args...)
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

		sql, args, err := insertQuery.ToSql()
		if err != nil {
			return err
		}

		_, err = db.Exec(sql, args...)
		return err
	}

	return nil
}
