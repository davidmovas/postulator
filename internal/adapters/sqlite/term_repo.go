package sqlite

import (
	"context"
	"database/sql"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	termColumns = `entity_id, site_id, taxonomy, term_id, parent_term_id, name, run_id, seen_at`
	upsertTerm  = `INSERT INTO entity_terms (` + termColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?) ` +
		`ON CONFLICT (entity_id, taxonomy) DO UPDATE SET site_id = excluded.site_id, term_id = excluded.term_id, ` +
		`parent_term_id = excluded.parent_term_id, name = excluded.name, run_id = excluded.run_id, seen_at = excluded.seen_at`
	deleteTerm        = `DELETE FROM entity_terms WHERE entity_id = ? AND taxonomy = ?`
	selectTermsBySite = `SELECT ` + termColumns + ` FROM entity_terms WHERE site_id = ? ORDER BY taxonomy, term_id, entity_id`
)

type TermRepo struct {
	store *Store
}

func NewTermRepo(store *Store) *TermRepo {
	return &TermRepo{store: store}
}

func termNotFound(entityID string, taxonomy graph.Taxonomy) *errors.Error {
	return errors.New(errors.NotFound, "the entity has no term in this taxonomy").
		WithDetail("entityId", entityID).WithDetail("taxonomy", string(taxonomy))
}

func (r *TermRepo) Upsert(ctx context.Context, t graph.Term) error {
	_, err := execWrite(ctx, r.store.writeFrom(ctx), upsertTerm, []any{
		t.EntityID, t.SiteID, string(t.Taxonomy), t.TermID, t.ParentTermID, t.Name, t.RunID, formatTime(t.SeenAt),
	}, nil, "save the entity term")
	return err
}

func (r *TermRepo) Delete(ctx context.Context, entityID string, taxonomy graph.Taxonomy) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), deleteTerm, []any{entityID, string(taxonomy)}, nil, "delete the entity term")
	return requireAffected(affected, err, termNotFound(entityID, taxonomy))
}

func (r *TermRepo) ListBySite(ctx context.Context, siteID string) ([]graph.Term, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectTermsBySite, []any{siteID}, scanTerm, "list the site terms")
}

func scanTerm(rows *sql.Rows) (graph.Term, error) {
	var (
		t                graph.Term
		taxonomy, seenAt string
	)
	if err := rows.Scan(&t.EntityID, &t.SiteID, &taxonomy, &t.TermID, &t.ParentTermID, &t.Name, &t.RunID, &seenAt); err != nil {
		return graph.Term{}, err
	}
	t.Taxonomy = graph.Taxonomy(taxonomy)

	var err error
	if t.SeenAt, err = parseTime(seenAt); err != nil {
		return graph.Term{}, err
	}
	return t, nil
}
