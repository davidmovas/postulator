package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/Masterminds/squirrel"

	"github.com/davidmovas/postulator/internal/adapters/sqlite/dbx"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func selectAll[T any](ctx context.Context, exec executor, query string, args []any, scan func(*sql.Rows) (T, error), message string) (items []T, err error) {
	rows, err := exec.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, dbx.Convert(err, message)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = dbx.Convert(closeErr, message)
		}
	}()

	items = make([]T, 0)
	for rows.Next() {
		item, scanErr := scan(rows)
		if scanErr != nil {
			return nil, dbx.Convert(scanErr, message)
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		return nil, dbx.Convert(err, message)
	}
	return items, nil
}

func selectOne[T any](ctx context.Context, exec executor, query string, args []any, scan func(*sql.Rows) (T, error), notFound *errors.Error, message string) (T, error) {
	var item T
	items, err := selectAll(ctx, exec, query, args, scan, message)
	switch {
	case err == nil && len(items) == 0:
		err = sql.ErrNoRows
	case err == nil:
		item = items[0]
	}
	return dbx.From(item, err).
		NotFound(notFound).
		WrapErr(func(cause error) error { return dbx.Convert(cause, message) }).
		Unwrap()
}

func execWrite(ctx context.Context, exec executor, query string, args []any, conflict *errors.Error, message string) (int64, error) {
	convert := func(cause error) error { return dbx.Convert(cause, message) }

	result, err := exec.ExecContext(ctx, query, args...)
	if err != nil {
		outcome := dbx.From(int64(0), err)
		if conflict != nil {
			outcome = outcome.Conflict(conflict)
		}
		return outcome.WrapErr(convert).Unwrap()
	}

	affected, err := result.RowsAffected()
	return dbx.From(affected, err).WrapErr(convert).Unwrap()
}

func requireAffected(affected int64, err error, notFound *errors.Error) error {
	if err != nil {
		return err
	}
	if affected == 0 {
		return notFound
	}
	return nil
}

func buildQuery(builder squirrel.SelectBuilder, what string) (query string, args []any, err error) {
	query, args, err = builder.ToSql()
	if err != nil {
		return "", nil, errors.Wrap(err, errors.Internal, "build the "+what+" query")
	}
	return query, args, nil
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func parseTime(raw string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, errors.Wrap(err, errors.Internal, "stored timestamp is not rfc3339")
	}
	return parsed.UTC(), nil
}

func nullString(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

func optString(raw sql.NullString) *string {
	if !raw.Valid {
		return nil
	}
	value := raw.String
	return &value
}

func boolInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func encodeJSON(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", errors.Wrap(err, errors.Internal, "encode a json column")
	}
	return string(encoded), nil
}

func decodeJSON(raw string, into any, message string) error {
	if err := json.Unmarshal([]byte(raw), into); err != nil {
		return errors.Wrap(err, errors.Internal, message)
	}
	return nil
}
