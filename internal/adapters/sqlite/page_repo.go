package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/Masterminds/squirrel"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	pageColumns       = `id, site_id, path, slug, parent_page_id, wp_type, wp_id, title, h1, meta_title, meta_description, canonical, primary_keyword, keywords, status, entity_id, template_id, content_hash, wp_link, wp_slug, wp_status, wp_title, wp_h1, wp_modified_at, last_synced_at, drift, created_at, updated_at`
	insertPage        = `INSERT INTO pages (` + pageColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	updatePage        = `UPDATE pages SET path = ?, slug = ?, parent_page_id = ?, wp_type = ?, wp_id = ?, title = ?, h1 = ?, meta_title = ?, meta_description = ?, canonical = ?, primary_keyword = ?, keywords = ?, status = ?, entity_id = ?, template_id = ?, content_hash = ?, wp_link = ?, wp_slug = ?, wp_status = ?, wp_title = ?, wp_h1 = ?, wp_modified_at = ?, last_synced_at = ?, drift = ?, updated_at = ? WHERE id = ?`
	deletePage        = `DELETE FROM pages WHERE id = ?`
	selectPage        = `SELECT ` + pageColumns + ` FROM pages WHERE id = ?`
	selectPagesBySite = `SELECT ` + pageColumns + ` FROM pages WHERE site_id = ? ORDER BY path, id`
)

type PageRepo struct {
	store *Store
}

func NewPageRepo(store *Store) *PageRepo {
	return &PageRepo{store: store}
}

func pageNotFound(id string) *errors.Error {
	return errors.New(errors.NotFound, "page not found").WithDetail("pageId", id)
}

func pageConflict(path string) *errors.Error {
	return errors.New(errors.Conflict, "a page with this path already exists in the site").WithDetail("path", path)
}

func nullTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return formatTime(*t)
}

func parseNullTime(raw sql.NullString) (*time.Time, error) {
	if !raw.Valid {
		return nil, nil
	}
	parsed, err := parseTime(raw.String)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func nullInt(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}

func optInt(raw sql.NullInt64) *int64 {
	if !raw.Valid {
		return nil
	}
	value := raw.Int64
	return &value
}

func (r *PageRepo) Insert(ctx context.Context, p pagemap.Page) error {
	keywords, err := encodeJSON(orEmpty(p.Keywords))
	if err != nil {
		return err
	}
	_, err = execWrite(ctx, r.store.writeFrom(ctx), insertPage, []any{
		p.ID, p.SiteID, p.Path, p.Slug, nullString(p.ParentPageID), string(p.WPType), nullInt(p.WPID),
		p.Title, p.H1, p.MetaTitle, p.MetaDescription, p.Canonical, p.PrimaryKeyword, keywords, string(p.Status), nullString(p.EntityID), nullString(p.TemplateID),
		p.ContentHash, p.Observed.Link, p.Observed.Slug, p.Observed.Status, p.Observed.Title, p.Observed.H1,
		nullTime(p.WPModifiedAt), nullTime(p.LastSyncedAt), boolInt(p.Drift), formatTime(p.CreatedAt), formatTime(p.UpdatedAt),
	}, pageConflict(p.Path), "insert the page")
	return err
}

func (r *PageRepo) Update(ctx context.Context, p pagemap.Page) error {
	keywords, err := encodeJSON(orEmpty(p.Keywords))
	if err != nil {
		return err
	}
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), updatePage, []any{
		p.Path, p.Slug, nullString(p.ParentPageID), string(p.WPType), nullInt(p.WPID),
		p.Title, p.H1, p.MetaTitle, p.MetaDescription, p.Canonical, p.PrimaryKeyword, keywords, string(p.Status), nullString(p.EntityID), nullString(p.TemplateID),
		p.ContentHash, p.Observed.Link, p.Observed.Slug, p.Observed.Status, p.Observed.Title, p.Observed.H1,
		nullTime(p.WPModifiedAt), nullTime(p.LastSyncedAt), boolInt(p.Drift), formatTime(p.UpdatedAt), p.ID,
	}, pageConflict(p.Path), "update the page")
	return requireAffected(affected, err, pageNotFound(p.ID))
}

func (r *PageRepo) Delete(ctx context.Context, id string) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), deletePage, []any{id}, nil, "delete the page")
	return requireAffected(affected, err, pageNotFound(id))
}

func (r *PageRepo) Get(ctx context.Context, id string) (pagemap.Page, error) {
	return selectOne(ctx, r.store.execFrom(ctx), selectPage, []any{id}, scanPage, pageNotFound(id), "read the page")
}

func (r *PageRepo) ListBySite(ctx context.Context, siteID string) ([]pagemap.Page, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectPagesBySite, []any{siteID}, scanPage, "list the site pages")
}

func pageKeyset(q pagemap.Query) paging.Keyset[pagemap.Page] {
	key := paging.TimeKey[pagemap.Page]("createdAt", "created_at", func(p pagemap.Page) any { return p.CreatedAt })
	if q.Sort == pagemap.SortPath {
		key = paging.TextKey[pagemap.Page]("path", "path", func(p pagemap.Page) any { return p.Path })
	}
	return paging.Keyset[pagemap.Page]{
		IDColumn: "id",
		ID:       func(p pagemap.Page) string { return p.ID },
		Keys:     []paging.SortKey[pagemap.Page]{key},
		Desc:     q.Desc,
	}
}

func (r *PageRepo) List(ctx context.Context, q pagemap.Query, page paging.Request) (paging.List[pagemap.Page], error) {
	builder := squirrel.Select(pageColumns).From("pages")
	if q.SiteID != "" {
		builder = builder.Where(squirrel.Eq{"site_id": q.SiteID})
	}
	if q.Status != nil {
		builder = builder.Where(squirrel.Eq{"status": string(*q.Status)})
	}
	if q.EntityID != nil {
		builder = builder.Where(squirrel.Eq{"entity_id": *q.EntityID})
	}
	if q.Unmapped {
		builder = builder.Where("entity_id IS NULL")
	}
	if q.PathPrefix != "" {
		builder = builder.Where(`path LIKE ? ESCAPE '\'`, escapeLike(q.PathPrefix)+"%")
	}

	keyset := pageKeyset(q)
	keyed, err := keyset.Apply(builder, page)
	if err != nil {
		return paging.List[pagemap.Page]{}, err
	}
	query, args, err := buildQuery(keyed, "pages")
	if err != nil {
		return paging.List[pagemap.Page]{}, err
	}
	rows, err := selectAll(ctx, r.store.execFrom(ctx), query, args, scanPage, "list the pages")
	if err != nil {
		return paging.List[pagemap.Page]{}, err
	}
	return keyset.Cut(rows, page)
}

func scanPage(rows *sql.Rows) (pagemap.Page, error) {
	var (
		p                              pagemap.Page
		parentID, entityID, templateID sql.NullString
		wpType, status                 string
		wpID                           sql.NullInt64
		wpModifiedAt, lastSyncedAt     sql.NullString
		drift                          int64
		createdAt, updatedAt           string
		keywords                       string
	)
	if err := rows.Scan(&p.ID, &p.SiteID, &p.Path, &p.Slug, &parentID, &wpType, &wpID, &p.Title, &p.H1, &p.MetaTitle, &p.MetaDescription, &p.Canonical,
		&p.PrimaryKeyword, &keywords, &status, &entityID, &templateID, &p.ContentHash, &p.Observed.Link, &p.Observed.Slug, &p.Observed.Status,
		&p.Observed.Title, &p.Observed.H1, &wpModifiedAt, &lastSyncedAt, &drift, &createdAt, &updatedAt); err != nil {
		return pagemap.Page{}, err
	}
	if err := decodeJSON(keywords, &p.Keywords, "decode the page keywords"); err != nil {
		return pagemap.Page{}, err
	}
	p.ParentPageID = optString(parentID)
	p.WPType = pagemap.WPType(wpType)
	p.WPID = optInt(wpID)
	p.Status = pagemap.Status(status)
	p.EntityID = optString(entityID)
	p.TemplateID = optString(templateID)
	p.Drift = drift == 1

	var err error
	if p.WPModifiedAt, err = parseNullTime(wpModifiedAt); err != nil {
		return pagemap.Page{}, err
	}
	if p.LastSyncedAt, err = parseNullTime(lastSyncedAt); err != nil {
		return pagemap.Page{}, err
	}
	if p.CreatedAt, err = parseTime(createdAt); err != nil {
		return pagemap.Page{}, err
	}
	if p.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return pagemap.Page{}, err
	}
	return p, nil
}
