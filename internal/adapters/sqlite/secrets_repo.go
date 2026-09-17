package sqlite

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite/dbx"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	selectSecret = `SELECT ciphertext FROM secrets WHERE ref = ?`
	deleteSecret = `DELETE FROM secrets WHERE ref = ?`
	upsertSecret = `INSERT INTO secrets (ref, ciphertext, created_at, updated_at) VALUES (?, ?, ?, ?)
ON CONFLICT(ref) DO UPDATE SET ciphertext = excluded.ciphertext, updated_at = excluded.updated_at`
)

type SecretsRepo struct {
	store *Store
	clock clock.Clock
}

func NewSecretsRepo(store *Store, clk clock.Clock) *SecretsRepo {
	return &SecretsRepo{store: store, clock: clk}
}

func (r *SecretsRepo) Put(ctx context.Context, ref string, ciphertext []byte) error {
	if ref == "" {
		return errors.New(errors.Invalid, "secret reference must not be empty")
	}
	if len(ciphertext) == 0 {
		return errors.New(errors.Invalid, "secret ciphertext must not be empty")
	}

	now := r.clock.Now().UTC().Format(time.RFC3339)
	_, err := r.store.writeFrom(ctx).ExecContext(ctx, upsertSecret, ref, ciphertext, now, now)
	return dbx.Convert(err, "write the secret "+ref)
}

func (r *SecretsRepo) Get(ctx context.Context, ref string) ([]byte, error) {
	var ciphertext []byte
	err := r.store.execFrom(ctx).QueryRowContext(ctx, selectSecret, ref).Scan(&ciphertext)

	return dbx.From(ciphertext, err).
		NotFound(errors.New(errors.NotFound, "secret "+ref+" is not stored")).
		WrapErr(func(cause error) error { return dbx.Convert(cause, "read the secret "+ref) }).
		Unwrap()
}

func (r *SecretsRepo) Delete(ctx context.Context, ref string) error {
	result, err := r.store.writeFrom(ctx).ExecContext(ctx, deleteSecret, ref)
	if err != nil {
		return dbx.Convert(err, "delete the secret "+ref)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return dbx.Convert(err, "count the deleted secrets")
	}
	if affected == 0 {
		return errors.New(errors.NotFound, "secret "+ref+" is not stored")
	}
	return nil
}
