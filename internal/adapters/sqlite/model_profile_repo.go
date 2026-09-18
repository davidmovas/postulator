package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/davidmovas/postulator/internal/domain/llm"
)

const (
	upsertProfile = `INSERT INTO model_profiles (role, provider, model, updated_at) VALUES (?, ?, ?, ?)
		ON CONFLICT (role) DO UPDATE SET provider = excluded.provider, model = excluded.model, updated_at = excluded.updated_at`
	selectProfiles = `SELECT role, provider, model FROM model_profiles ORDER BY role`
)

type ModelProfileRepo struct {
	store *Store
}

func NewModelProfileRepo(store *Store) *ModelProfileRepo {
	return &ModelProfileRepo{store: store}
}

func (r *ModelProfileRepo) Set(ctx context.Context, role llm.Role, ref llm.ModelRef, at time.Time) error {
	_, err := execWrite(ctx, r.store.writeFrom(ctx), upsertProfile,
		[]any{string(role), ref.Provider, ref.Model, formatTime(at)}, nil, "store the model profile")
	return err
}

func (r *ModelProfileRepo) List(ctx context.Context) (map[llm.Role]llm.ModelRef, error) {
	rows, err := selectAll(ctx, r.store.execFrom(ctx), selectProfiles, nil, scanProfile, "list the model profiles")
	if err != nil {
		return nil, err
	}

	profiles := make(map[llm.Role]llm.ModelRef, len(rows))
	for _, row := range rows {
		profiles[row.role] = row.ref
	}
	return profiles, nil
}

type profileRow struct {
	role llm.Role
	ref  llm.ModelRef
}

func scanProfile(rows *sql.Rows) (profileRow, error) {
	var (
		row  profileRow
		role string
	)
	if err := rows.Scan(&role, &row.ref.Provider, &row.ref.Model); err != nil {
		return profileRow{}, err
	}
	row.role = llm.Role(role)
	return row, nil
}
