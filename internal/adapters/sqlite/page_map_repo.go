package sqlite

import (
	"context"
	"database/sql"
	stderrors "errors"

	"github.com/Masterminds/squirrel"

	"github.com/davidmovas/postulator/internal/adapters/sqlite/dbx"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	mapColumns = `page_id, site_id, parent_id, path, depth, trail, children, descendants, twin_id, hub_id, node,
		type_id, twin_type_id, title, title_key, path_key, twin_title_key, twin_path_key`
	selectMapState = `SELECT generation, built FROM page_map_sites WHERE site_id = ?`
	selectMapSite  = `SELECT ` + mapColumns + ` FROM page_map WHERE site_id = ? ORDER BY page_id`
	selectMapIDs   = `SELECT ` + mapColumns + ` FROM page_map WHERE page_id IN `
	orderMapIDs    = ` ORDER BY page_id`
	upsertMapRow   = `INSERT INTO page_map (` + mapColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (page_id) DO UPDATE SET site_id = excluded.site_id, parent_id = excluded.parent_id, path = excluded.path,
		depth = excluded.depth, trail = excluded.trail, children = excluded.children, descendants = excluded.descendants,
		twin_id = excluded.twin_id, hub_id = excluded.hub_id, node = excluded.node, type_id = excluded.type_id,
		twin_type_id = excluded.twin_type_id, title = excluded.title, title_key = excluded.title_key, path_key = excluded.path_key,
		twin_title_key = excluded.twin_title_key, twin_path_key = excluded.twin_path_key`
	deleteMapRow    = `DELETE FROM page_map WHERE page_id = ? AND site_id = ?`
	markMapBuilt    = `UPDATE page_map_sites SET built = max(built, ?) WHERE site_id = ?`
	trailCeiling    = "0"
	textCondition   = `(instr(title_key, ?) > 0 OR instr(path_key, ?) > 0 OR instr(twin_title_key, ?) > 0 OR instr(twin_path_key, ?) > 0)`
	withinCondition = `(type_id = ? OR twin_type_id = ? OR EXISTS (SELECT 1 FROM page_map AS below
		WHERE below.site_id = page_map.site_id AND below.type_id = ? AND below.trail > page_map.trail
		AND below.trail < substr(page_map.trail, 1, length(page_map.trail) - 1) || '` + trailCeiling + `'))`
)

type PageMapRepo struct {
	store *Store
}

func NewPageMapRepo(store *Store) *PageMapRepo {
	return &PageMapRepo{store: store}
}

func (r *PageMapRepo) State(ctx context.Context, siteID string) (generation, built int64, err error) {
	err = r.store.execFrom(ctx).QueryRowContext(ctx, selectMapState, siteID).Scan(&generation, &built)
	if stderrors.Is(err, sql.ErrNoRows) {
		return 1, 0, nil
	}
	return generation, built, dbx.Convert(err, "read the site map's state")
}

func (r *PageMapRepo) ListBySite(ctx context.Context, siteID string) ([]contextpack.MapRow, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectMapSite, []any{siteID}, scanMapRow, "list the site map")
}

func (r *PageMapRepo) ListByIDs(ctx context.Context, pageIDs []string) ([]contextpack.MapRow, error) {
	return selectByIDs(ctx, r.store.execFrom(ctx), selectMapIDs, orderMapIDs, pageIDs, scanMapRow, "list the map rows of the pages")
}

func (r *PageMapRepo) IDsUnder(ctx context.Context, siteID, trail, typeID string) ([]string, error) {
	query := `SELECT page_id FROM page_map WHERE site_id = ? AND trail LIKE ? ESCAPE '\'`
	args := []any{siteID, escapeLike(trail) + "%"}
	if typeID != "" {
		query += ` AND type_id = ?`
		args = append(args, typeID)
	}
	return selectAll(ctx, r.store.execFrom(ctx), query+` ORDER BY trail, page_id`, args, scanText, "list the pages under a node")
}

func (r *PageMapRepo) Write(ctx context.Context, siteID string, moved []contextpack.MapRow, gone []string, generation int64) error {
	return r.store.Do(ctx, func(c context.Context) error {
		exec := r.store.writeFrom(c)
		for i := range moved {
			row := &moved[i]
			if _, err := execWrite(c, exec, upsertMapRow, []any{
				row.PageID, siteID, row.ParentID, row.Path, row.Depth, row.Trail, row.Children, row.Descendants, row.TwinID, row.HubID,
				boolInt(row.Row), row.TypeID, row.TwinTypeID, row.Title, row.TitleKey, row.PathKey, row.TwinTitleKey, row.TwinPathKey,
			}, nil, "store a map row"); err != nil {
				return err
			}
		}
		for _, pageID := range gone {
			if _, err := execWrite(c, exec, deleteMapRow, []any{pageID, siteID}, nil, "drop a map row"); err != nil {
				return err
			}
		}
		_, err := execWrite(c, exec, markMapBuilt, []any{generation, siteID}, nil, "mark the site map built")
		return err
	})
}

func mapKeyset() paging.Keyset[contextpack.MapRow] {
	return paging.Keyset[contextpack.MapRow]{
		IDColumn: "page_id",
		ID:       func(row contextpack.MapRow) string { return row.PageID },
		Keys: []paging.SortKey[contextpack.MapRow]{
			paging.TextKey[contextpack.MapRow]("path", "path", func(row contextpack.MapRow) any { return row.Path }),
		},
	}
}

func (r *PageMapRepo) List(ctx context.Context, q contextpack.MapRowQuery, page paging.Request) (paging.List[contextpack.MapRow], error) {
	builder := squirrel.Select(mapColumns).From("page_map").Where(squirrel.Eq{"site_id": q.SiteID, "node": 1})
	if q.ParentID != nil {
		builder = builder.Where(squirrel.Eq{"parent_id": *q.ParentID})
	}
	if q.Under != "" {
		builder = builder.Where("trail >= ? AND trail < ?", q.Under, q.Under[:len(q.Under)-1]+trailCeiling)
	}
	if q.TypeID != "" {
		builder = builder.Where("(type_id = ? OR twin_type_id = ?)", q.TypeID, q.TypeID)
	}
	if q.Text != "" {
		builder = builder.Where(textCondition, q.Text, q.Text, q.Text, q.Text)
	}
	if q.Within != "" {
		builder = builder.Where(withinCondition, q.Within, q.Within, q.Within)
	}
	return selectKeyed(ctx, r.store.execFrom(ctx), builder, mapKeyset(), page, scanMapRow, "site map")
}

func scanMapRow(rows *sql.Rows) (contextpack.MapRow, error) {
	var (
		row  contextpack.MapRow
		node int
	)
	err := rows.Scan(&row.PageID, &row.SiteID, &row.ParentID, &row.Path, &row.Depth, &row.Trail, &row.Children, &row.Descendants,
		&row.TwinID, &row.HubID, &node, &row.TypeID, &row.TwinTypeID, &row.Title, &row.TitleKey, &row.PathKey,
		&row.TwinTitleKey, &row.TwinPathKey)
	row.Row = node == 1
	return row, err
}
