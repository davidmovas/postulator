package sqlite

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	entityColumns         = `id, site_id, name, kind, intent, primary_keyword, secondary_keywords, canonical_page_id, score, source, created_at, updated_at`
	insertEntity          = `INSERT INTO entities (` + entityColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	updateEntity          = `UPDATE entities SET name = ?, kind = ?, intent = ?, primary_keyword = ?, secondary_keywords = ?, canonical_page_id = ?, score = ?, source = ?, updated_at = ? WHERE id = ?`
	updateEntityScore     = `UPDATE entities SET score = ? WHERE id = ?`
	updateEntityCanonical = `UPDATE entities SET canonical_page_id = ?, updated_at = ? WHERE id = ?`
	deleteEntity          = `DELETE FROM entities WHERE id = ?`
	selectEntity          = `SELECT ` + entityColumns + ` FROM entities WHERE id = ?`
	selectEntitiesBySite  = `SELECT ` + entityColumns + ` FROM entities WHERE site_id = ? ORDER BY name, id`
	deleteAnchors         = `DELETE FROM entity_anchors WHERE entity_id = ?`
	insertAnchor          = `INSERT INTO entity_anchors (entity_id, position, text, source, weight) VALUES (?, ?, ?, ?, ?)`
	selectAnchorsBySite   = `SELECT a.entity_id, a.text, a.source, a.weight FROM entity_anchors a JOIN entities e ON e.id = a.entity_id WHERE e.site_id = ? ORDER BY a.entity_id, a.position`
)

type EntityRepo struct {
	store *Store
}

func NewEntityRepo(store *Store) *EntityRepo {
	return &EntityRepo{store: store}
}

func entityNotFound(id string) *errors.Error {
	return errors.New(errors.NotFound, "entity not found").WithDetail("entityId", id)
}

func entityConflict(name string) *errors.Error {
	return errors.New(errors.Conflict, "an entity with this name already exists in the site").WithDetail("name", name)
}

func anchorConflict(entityID string) *errors.Error {
	return errors.New(errors.Conflict, "an anchor text is repeated").WithDetail("entityId", entityID)
}

func (r *EntityRepo) Insert(ctx context.Context, e graph.Entity) error {
	keywords, err := encodeJSON(orEmpty(e.SecondaryKeywords))
	if err != nil {
		return err
	}
	if _, err = execWrite(ctx, r.store.writeFrom(ctx), insertEntity, []any{
		e.ID, e.SiteID, e.Name, string(e.Kind), e.Intent, e.PrimaryKeyword, keywords, nullString(e.CanonicalPageID),
		e.Score, string(e.Source), formatTime(e.CreatedAt), formatTime(e.UpdatedAt),
	}, entityConflict(e.Name), "insert the entity"); err != nil {
		return err
	}
	return r.writeAnchors(ctx, e.ID, e.Anchors)
}

func (r *EntityRepo) Update(ctx context.Context, e graph.Entity) error {
	keywords, err := encodeJSON(orEmpty(e.SecondaryKeywords))
	if err != nil {
		return err
	}
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), updateEntity, []any{
		e.Name, string(e.Kind), e.Intent, e.PrimaryKeyword, keywords, nullString(e.CanonicalPageID),
		e.Score, string(e.Source), formatTime(e.UpdatedAt), e.ID,
	}, entityConflict(e.Name), "update the entity")
	if updateErr := requireAffected(affected, err, entityNotFound(e.ID)); updateErr != nil {
		return updateErr
	}
	if _, err = execWrite(ctx, r.store.writeFrom(ctx), deleteAnchors, []any{e.ID}, nil, "clear the entity anchors"); err != nil {
		return err
	}
	return r.writeAnchors(ctx, e.ID, e.Anchors)
}

func (r *EntityRepo) writeAnchors(ctx context.Context, entityID string, anchors []graph.Anchor) error {
	for position, anchor := range anchors {
		if _, err := execWrite(ctx, r.store.writeFrom(ctx), insertAnchor, []any{
			entityID, position, anchor.Text, string(anchor.Source), anchor.Weight,
		}, anchorConflict(entityID), "insert an entity anchor"); err != nil {
			return err
		}
	}
	return nil
}

func orEmpty(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func escapeLike(s string) string {
	return likeEscaper.Replace(s)
}

func (r *EntityRepo) Delete(ctx context.Context, id string) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), deleteEntity, []any{id}, nil, "delete the entity")
	return requireAffected(affected, err, entityNotFound(id))
}

func (r *EntityRepo) SetScore(ctx context.Context, id string, score float64) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), updateEntityScore, []any{score, id}, nil, "update the entity score")
	return requireAffected(affected, err, entityNotFound(id))
}

func (r *EntityRepo) SetCanonicalPage(ctx context.Context, id string, pageID *string, updatedAt time.Time) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), updateEntityCanonical, []any{nullString(pageID), formatTime(updatedAt), id}, nil, "update the entity canonical page")
	return requireAffected(affected, err, entityNotFound(id))
}

func (r *EntityRepo) Get(ctx context.Context, id string) (graph.Entity, error) {
	entity, err := selectOne(ctx, r.store.execFrom(ctx), selectEntity, []any{id}, scanEntity, entityNotFound(id), "read the entity")
	if err != nil {
		return graph.Entity{}, err
	}
	entities := []graph.Entity{entity}
	if attachErr := r.attachAnchorsByID(ctx, entities); attachErr != nil {
		return graph.Entity{}, attachErr
	}
	return entities[0], nil
}

func (r *EntityRepo) ListBySite(ctx context.Context, siteID string) ([]graph.Entity, error) {
	entities, err := selectAll(ctx, r.store.execFrom(ctx), selectEntitiesBySite, []any{siteID}, scanEntity, "list the site entities")
	if err != nil {
		return nil, err
	}
	rows, err := selectAll(ctx, r.store.execFrom(ctx), selectAnchorsBySite, []any{siteID}, scanAnchor, "list the site anchors")
	if err != nil {
		return nil, err
	}
	attachAnchors(entities, rows)
	return entities, nil
}

func entityKeyset(q graph.EntityQuery) paging.Keyset[graph.Entity] {
	key := paging.TimeKey[graph.Entity]("createdAt", "created_at", func(e graph.Entity) any { return e.CreatedAt })
	if q.Sort == graph.EntitySortName {
		key = paging.TextKey[graph.Entity]("name", "name", func(e graph.Entity) any { return e.Name })
	}
	return paging.Keyset[graph.Entity]{
		IDColumn: "id",
		ID:       func(e graph.Entity) string { return e.ID },
		Keys:     []paging.SortKey[graph.Entity]{key},
		Desc:     q.Desc,
	}
}

func (r *EntityRepo) List(ctx context.Context, q graph.EntityQuery, page paging.Request) (paging.List[graph.Entity], error) {
	builder := squirrel.Select(entityColumns).From("entities")
	if q.SiteID != "" {
		builder = builder.Where(squirrel.Eq{"site_id": q.SiteID})
	}
	if q.Kind != nil {
		builder = builder.Where(squirrel.Eq{"kind": string(*q.Kind)})
	}
	if q.HasCanonicalPage != nil {
		if *q.HasCanonicalPage {
			builder = builder.Where("canonical_page_id IS NOT NULL")
		} else {
			builder = builder.Where("canonical_page_id IS NULL")
		}
	}
	if q.NamePrefix != "" {
		builder = builder.Where(`name LIKE ? ESCAPE '\'`, escapeLike(q.NamePrefix)+"%")
	}

	keyset := entityKeyset(q)
	keyed, err := keyset.Apply(builder, page)
	if err != nil {
		return paging.List[graph.Entity]{}, err
	}
	query, args, err := buildQuery(keyed, "entities")
	if err != nil {
		return paging.List[graph.Entity]{}, err
	}
	entities, err := selectAll(ctx, r.store.execFrom(ctx), query, args, scanEntity, "list the entities")
	if err != nil {
		return paging.List[graph.Entity]{}, err
	}
	if attachErr := r.attachAnchorsByID(ctx, entities); attachErr != nil {
		return paging.List[graph.Entity]{}, attachErr
	}
	return keyset.Cut(entities, page)
}

func (r *EntityRepo) attachAnchorsByID(ctx context.Context, entities []graph.Entity) error {
	if len(entities) == 0 {
		return nil
	}
	ids := make([]string, 0, len(entities))
	for i := range entities {
		ids = append(ids, entities[i].ID)
	}
	query, args, err := buildQuery(squirrel.Select("entity_id", "text", "source", "weight").From("entity_anchors").
		Where(squirrel.Eq{"entity_id": ids}).OrderBy("entity_id", "position"), "entity anchors")
	if err != nil {
		return err
	}
	rows, err := selectAll(ctx, r.store.execFrom(ctx), query, args, scanAnchor, "list the entity anchors")
	if err != nil {
		return err
	}
	attachAnchors(entities, rows)
	return nil
}

type anchorRow struct {
	entityID string
	anchor   graph.Anchor
}

func attachAnchors(entities []graph.Entity, rows []anchorRow) {
	byEntity := make(map[string][]graph.Anchor, len(entities))
	for _, row := range rows {
		byEntity[row.entityID] = append(byEntity[row.entityID], row.anchor)
	}
	for i := range entities {
		anchors := byEntity[entities[i].ID]
		if anchors == nil {
			anchors = []graph.Anchor{}
		}
		entities[i].Anchors = anchors
	}
}

func scanAnchor(rows *sql.Rows) (anchorRow, error) {
	var (
		row    anchorRow
		source string
	)
	if err := rows.Scan(&row.entityID, &row.anchor.Text, &source, &row.anchor.Weight); err != nil {
		return anchorRow{}, err
	}
	row.anchor.Source = graph.AnchorSource(source)
	return row, nil
}

func scanEntity(rows *sql.Rows) (graph.Entity, error) {
	var (
		e                    graph.Entity
		kind, source         string
		keywords             string
		canonical            sql.NullString
		createdAt, updatedAt string
	)
	if err := rows.Scan(&e.ID, &e.SiteID, &e.Name, &kind, &e.Intent, &e.PrimaryKeyword, &keywords, &canonical, &e.Score, &source, &createdAt, &updatedAt); err != nil {
		return graph.Entity{}, err
	}
	e.Kind = graph.Kind(kind)
	e.Source = graph.Source(source)
	e.CanonicalPageID = optString(canonical)
	if err := decodeJSON(keywords, &e.SecondaryKeywords, "decode the entity keywords"); err != nil {
		return graph.Entity{}, err
	}

	var err error
	if e.CreatedAt, err = parseTime(createdAt); err != nil {
		return graph.Entity{}, err
	}
	if e.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return graph.Entity{}, err
	}
	return e, nil
}
