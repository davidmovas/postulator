package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	artifactColumns     = `id, run_id, item_id, step, kind, blob, size, hash, purged, expires_at, created_at`
	insertArtifact      = `INSERT INTO artifacts (` + artifactColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	deleteStepArtifacts = `DELETE FROM artifacts WHERE item_id = ? AND step = ?`
	selectArtifact      = `SELECT ` + artifactColumns + ` FROM artifacts WHERE id = ?`
	selectItemArtifacts = `SELECT ` + artifactColumns + ` FROM artifacts WHERE item_id = ? ORDER BY created_at, id`
	selectPurgedKinds   = `SELECT item_id, kind FROM artifacts WHERE purged = 1 AND item_id IN `
	selectHeldKinds     = `SELECT item_id, kind FROM artifacts WHERE purged = 0 AND item_id IN `
)

var purgeArtifacts = `UPDATE artifacts SET blob = x'', size = 0, purged = 1
		WHERE purged = 0 AND kind IN (` + placeholders(len(run.PurgeableArtifactKinds())) + `) AND created_at <= ? AND item_id IN (
			SELECT item_id FROM artifacts WHERE kind = ?)`

type ArtifactRepo struct {
	store *Store
}

func NewArtifactRepo(store *Store) *ArtifactRepo {
	return &ArtifactRepo{store: store}
}

func artifactNotFound(id string) *errors.Error {
	return errors.New(errors.NotFound, "artifact not found").WithDetail("artifactId", id)
}

func (r *ArtifactRepo) ReplaceStep(ctx context.Context, itemID, step string, artifacts []run.Artifact) error {
	exec := r.store.writeFrom(ctx)
	if _, err := execWrite(ctx, exec, deleteStepArtifacts, []any{itemID, step}, nil,
		"discard the artifacts of the previous attempt"); err != nil {
		return err
	}

	for i := range artifacts {
		a := artifacts[i]
		if _, err := execWrite(ctx, exec, insertArtifact, []any{
			a.ID, a.RunID, a.ItemID, a.Step, string(a.Kind), a.Blob, a.Size, a.Hash, boolInt(a.Purged),
			nullTime(a.ExpiresAt), formatTime(a.CreatedAt),
		}, errors.New(errors.Conflict, "the step already produced an artifact of this kind"),
			"store the artifact"); err != nil {
			return err
		}
	}
	return nil
}

func (r *ArtifactRepo) Get(ctx context.Context, id string) (run.Artifact, error) {
	return selectOne(ctx, r.store.execFrom(ctx), selectArtifact, []any{id}, scanArtifact, artifactNotFound(id),
		"read the artifact")
}

func (r *ArtifactRepo) ByItem(ctx context.Context, itemID string) ([]run.Artifact, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectItemArtifacts, []any{itemID}, scanArtifact,
		"list the artifacts of the run item")
}

func (r *ArtifactRepo) PurgedByItems(ctx context.Context, itemIDs []string) (map[string][]run.ArtifactKind, error) {
	purged := make(map[string][]run.ArtifactKind, len(itemIDs))
	if len(itemIDs) == 0 {
		return purged, nil
	}

	args := make([]any, 0, len(itemIDs))
	for _, itemID := range itemIDs {
		args = append(args, itemID)
	}

	query := selectPurgedKinds + "(" + placeholders(len(itemIDs)) + ") ORDER BY item_id, created_at, id"
	rows, err := selectAll(ctx, r.store.execFrom(ctx), query, args, scanPurgedKind,
		"list the purged artifacts of the run items")
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		purged[row.itemID] = append(purged[row.itemID], row.kind)
	}
	return purged, nil
}

func (r *ArtifactRepo) KindsByItems(ctx context.Context, itemIDs []string) (map[string][]run.ArtifactKind, error) {
	held := make(map[string][]run.ArtifactKind, len(itemIDs))
	if len(itemIDs) == 0 {
		return held, nil
	}

	args := make([]any, 0, len(itemIDs))
	for _, itemID := range itemIDs {
		args = append(args, itemID)
	}

	query := selectHeldKinds + "(" + placeholders(len(itemIDs)) + ") ORDER BY item_id, created_at, id"
	rows, err := selectAll(ctx, r.store.execFrom(ctx), query, args, scanPurgedKind,
		"list the artifacts the run items still hold")
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		held[row.itemID] = append(held[row.itemID], row.kind)
	}
	return held, nil
}

type purgedKind struct {
	itemID string
	kind   run.ArtifactKind
}

func scanPurgedKind(rows *sql.Rows) (purgedKind, error) {
	var (
		itemID string
		kind   string
	)
	if err := rows.Scan(&itemID, &kind); err != nil {
		return purgedKind{}, err
	}
	return purgedKind{itemID: itemID, kind: run.ArtifactKind(kind)}, nil
}

func (r *ArtifactRepo) PurgePublishedBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	purgeable := run.PurgeableArtifactKinds()
	args := make([]any, 0, len(purgeable)+2)
	for _, kind := range purgeable {
		args = append(args, string(kind))
	}
	args = append(args, formatTime(cutoff), string(run.ArtifactPublishResult))

	return execWrite(ctx, r.store.writeFrom(ctx), purgeArtifacts, args, nil,
		"purge the published artifact blobs")
}

func scanArtifact(rows *sql.Rows) (run.Artifact, error) {
	var (
		artifact  run.Artifact
		kind      string
		purged    int64
		expiresAt sql.NullString
		createdAt string
	)
	if err := rows.Scan(
		&artifact.ID, &artifact.RunID, &artifact.ItemID, &artifact.Step, &kind, &artifact.Blob, &artifact.Size,
		&artifact.Hash, &purged, &expiresAt, &createdAt,
	); err != nil {
		return run.Artifact{}, err
	}

	artifact.Kind = run.ArtifactKind(kind)
	artifact.Purged = purged == 1

	var err error
	if artifact.ExpiresAt, err = parseNullTime(expiresAt); err != nil {
		return run.Artifact{}, err
	}
	if artifact.CreatedAt, err = parseTime(createdAt); err != nil {
		return run.Artifact{}, err
	}
	return artifact, nil
}
