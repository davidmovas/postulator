package sqlite

import (
	"context"
	"database/sql"

	"github.com/Masterminds/squirrel"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	edgeColumns       = `id, site_id, from_entity_id, to_entity_id, kind, weight, source, status, created_at`
	insertEdge        = `INSERT INTO edges (` + edgeColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	updateEdgeStatus  = `UPDATE edges SET status = ? WHERE id = ?`
	deleteEdge        = `DELETE FROM edges WHERE id = ?`
	selectEdge        = `SELECT ` + edgeColumns + ` FROM edges WHERE id = ?`
	selectEdgesBySite = `SELECT ` + edgeColumns + ` FROM edges WHERE site_id = ? ORDER BY created_at, id`
)

type EdgeRepo struct {
	store *Store
}

func NewEdgeRepo(store *Store) *EdgeRepo {
	return &EdgeRepo{store: store}
}

func edgeNotFound(id string) *errors.Error {
	return errors.New(errors.NotFound, "edge not found").WithDetail("edgeId", id)
}

func edgeConflict(e *graph.Edge) *errors.Error {
	return errors.New(errors.Conflict, "an edge of this kind already links these entities").
		WithDetail("fromEntityId", e.FromEntityID).WithDetail("toEntityId", e.ToEntityID).WithDetail("kind", string(e.Kind))
}

func (r *EdgeRepo) Insert(ctx context.Context, e graph.Edge) error {
	_, err := execWrite(ctx, r.store.writeFrom(ctx), insertEdge, []any{
		e.ID, e.SiteID, e.FromEntityID, e.ToEntityID, string(e.Kind), e.Weight, string(e.Source), string(e.Status), formatTime(e.CreatedAt),
	}, edgeConflict(&e), "insert the edge")
	return err
}

func (r *EdgeRepo) SetStatus(ctx context.Context, id string, status graph.EdgeStatus) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), updateEdgeStatus, []any{string(status), id}, nil, "update the edge status")
	return requireAffected(affected, err, edgeNotFound(id))
}

func (r *EdgeRepo) Delete(ctx context.Context, id string) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), deleteEdge, []any{id}, nil, "delete the edge")
	return requireAffected(affected, err, edgeNotFound(id))
}

func (r *EdgeRepo) Get(ctx context.Context, id string) (graph.Edge, error) {
	return selectOne(ctx, r.store.execFrom(ctx), selectEdge, []any{id}, scanEdge, edgeNotFound(id), "read the edge")
}

func (r *EdgeRepo) ListBySite(ctx context.Context, siteID string) ([]graph.Edge, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectEdgesBySite, []any{siteID}, scanEdge, "list the site edges")
}

func edgeKeyset(desc bool) paging.Keyset[graph.Edge] {
	return paging.Keyset[graph.Edge]{
		IDColumn: "id",
		ID:       func(e graph.Edge) string { return e.ID },
		Keys:     []paging.SortKey[graph.Edge]{paging.TimeKey[graph.Edge]("createdAt", "created_at", func(e graph.Edge) any { return e.CreatedAt })},
		Desc:     desc,
	}
}

func (r *EdgeRepo) List(ctx context.Context, q graph.EdgeQuery, page paging.Request) (paging.List[graph.Edge], error) {
	builder := squirrel.Select(edgeColumns).From("edges")
	if q.SiteID != "" {
		builder = builder.Where(squirrel.Eq{"site_id": q.SiteID})
	}
	if q.Kind != nil {
		builder = builder.Where(squirrel.Eq{"kind": string(*q.Kind)})
	}
	if q.Status != nil {
		builder = builder.Where(squirrel.Eq{"status": string(*q.Status)})
	}
	if q.EntityID != "" {
		builder = builder.Where(squirrel.Or{squirrel.Eq{"from_entity_id": q.EntityID}, squirrel.Eq{"to_entity_id": q.EntityID}})
	}

	keyset := edgeKeyset(q.Desc)
	keyed, err := keyset.Apply(builder, page)
	if err != nil {
		return paging.List[graph.Edge]{}, err
	}
	query, args, err := buildQuery(keyed, "edges")
	if err != nil {
		return paging.List[graph.Edge]{}, err
	}
	rows, err := selectAll(ctx, r.store.execFrom(ctx), query, args, scanEdge, "list the edges")
	if err != nil {
		return paging.List[graph.Edge]{}, err
	}
	return keyset.Cut(rows, page)
}

func scanEdge(rows *sql.Rows) (graph.Edge, error) {
	var (
		e                    graph.Edge
		kind, source, status string
		createdAt            string
	)
	if err := rows.Scan(&e.ID, &e.SiteID, &e.FromEntityID, &e.ToEntityID, &kind, &e.Weight, &source, &status, &createdAt); err != nil {
		return graph.Edge{}, err
	}
	e.Kind = graph.EdgeKind(kind)
	e.Source = graph.Source(source)
	e.Status = graph.EdgeStatus(status)

	var err error
	if e.CreatedAt, err = parseTime(createdAt); err != nil {
		return graph.Edge{}, err
	}
	return e, nil
}
