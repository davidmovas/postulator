package sqlite

import (
	"context"
	"encoding/json"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite/dbx"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	selectSetting          = `SELECT value FROM settings WHERE key = ?`
	selectSettings         = `SELECT key, value FROM settings`
	selectSettingUpdatedAt = `SELECT updated_at FROM settings WHERE key = ?`
	upsertSetting          = `INSERT INTO settings (key, value, updated_at) VALUES (?, ?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`
)

type SettingsRepo struct {
	store *Store
	clock clock.Clock
}

func NewSettingsRepo(store *Store, clk clock.Clock) *SettingsRepo {
	return &SettingsRepo{store: store, clock: clk}
}

func (r *SettingsRepo) Get(ctx context.Context, key string) (value json.RawMessage, found bool, err error) {
	var stored string
	err = r.store.execFrom(ctx).QueryRowContext(ctx, selectSetting, key).Scan(&stored)
	if dbx.IsNotFound(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, dbx.Convert(err, "read the setting "+key)
	}
	return json.RawMessage(stored), true, nil
}

func (r *SettingsRepo) Set(ctx context.Context, key string, value json.RawMessage) error {
	if key == "" {
		return errors.New(errors.Invalid, "setting key must not be empty")
	}
	if !json.Valid(value) {
		return errors.New(errors.Invalid, "setting "+key+" must hold valid JSON")
	}

	updated := r.clock.Now().UTC().Format(time.RFC3339)
	_, err := r.store.writeFrom(ctx).ExecContext(ctx, upsertSetting, key, string(value), updated)
	return dbx.Convert(err, "write the setting "+key)
}

func (r *SettingsRepo) All(ctx context.Context) (stored map[string]json.RawMessage, err error) {
	rows, err := r.store.execFrom(ctx).QueryContext(ctx, selectSettings)
	if err != nil {
		return nil, dbx.Convert(err, "list the settings")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = dbx.Convert(closeErr, "close the settings rows")
		}
	}()

	stored = make(map[string]json.RawMessage)
	for rows.Next() {
		var key, value string
		if err = rows.Scan(&key, &value); err != nil {
			return nil, dbx.Convert(err, "scan a settings row")
		}
		stored[key] = json.RawMessage(value)
	}
	if err = rows.Err(); err != nil {
		return nil, dbx.Convert(err, "read the settings rows")
	}
	return stored, nil
}

func ScanUpdatedAt(ctx context.Context, store *Store, key string, into *string) error {
	err := store.execFrom(ctx).QueryRowContext(ctx, selectSettingUpdatedAt, key).Scan(into)
	return dbx.Convert(err, "read the setting timestamp for "+key)
}
