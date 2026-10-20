package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/Masterminds/squirrel"

	"github.com/davidmovas/postulator/internal/domain/keyword"
	"github.com/davidmovas/postulator/internal/domain/note"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	pageColumns       = `id, site_id, path, planned_path, slug, parent_page_id, parent_reason, parent_method, page_type_id, page_type_source, page_type_reason, technical_reason, read_only_reason, wp_type, wp_id, title, h1, meta_title, meta_description, canonical, keywords, notes, status, entity_id, template_id, content_hash, wp_link, wp_slug, wp_status, wp_title, wp_h1, wp_modified_at, last_synced_at, drift, created_at, updated_at`
	insertPage        = `INSERT INTO pages (` + pageColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	updatePage        = `UPDATE pages SET path = ?, planned_path = ?, slug = ?, parent_page_id = ?, parent_reason = ?, parent_method = ?, technical_reason = ?, read_only_reason = ?, wp_type = ?, wp_id = ?, title = ?, h1 = ?, meta_title = ?, meta_description = ?, canonical = ?, keywords = ?, notes = ?, status = ?, entity_id = ?, template_id = ?, content_hash = ?, wp_link = ?, wp_slug = ?, wp_status = ?, wp_title = ?, wp_h1 = ?, wp_modified_at = ?, last_synced_at = ?, drift = ?, updated_at = ? WHERE id = ?`
	updateDerived     = `UPDATE pages SET parent_page_id = ?, parent_reason = ?, parent_method = ?, technical_reason = ?, read_only_reason = ?, status = ?, updated_at = ? WHERE id = ?`
	updatePublished   = `UPDATE pages SET wp_id = ?, status = ?, content_hash = ?, wp_link = ?, wp_slug = ?, wp_status = ?, wp_title = ?, wp_h1 = ?, wp_modified_at = ?, last_synced_at = ?, drift = ?, updated_at = ? WHERE id = ?`
	setPageType       = `UPDATE pages SET page_type_id = ?, page_type_source = 'manual', page_type_reason = ?, updated_at = ? WHERE id = ?`
	autoPageType      = `UPDATE pages SET page_type_id = ?, page_type_source = 'auto', page_type_reason = ?, updated_at = ? WHERE id = ? AND page_type_id IS NULL`
	reassignPageType  = `UPDATE pages SET page_type_id = ?, page_type_source = 'manual', page_type_reason = ?, updated_at = ? WHERE page_type_id = ?`
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

func encodePlan(p pagemap.Page) (keywords, notes string, err error) {
	if keywords, err = encodeJSON(keyword.New(p.Keywords)); err != nil {
		return "", "", err
	}
	if notes, err = encodeJSON(note.Clean(p.Notes)); err != nil {
		return "", "", err
	}
	return keywords, notes, nil
}

func (r *PageRepo) Insert(ctx context.Context, p pagemap.Page) error {
	keywords, notes, err := encodePlan(p)
	if err != nil {
		return err
	}
	_, err = execWrite(ctx, r.store.writeFrom(ctx), insertPage, []any{
		p.ID, p.SiteID, p.Path, p.PlannedPath, p.Slug, nullString(p.ParentPageID),
		storedParentReason(p.ParentReason), string(p.ParentMethod), nullString(p.PageTypeID), string(p.TypeSource), string(p.TypeReason),
		string(p.TechnicalReason), string(p.ReadOnlyReason), string(p.WPType), nullInt(p.WPID),
		p.Title, p.H1, p.MetaTitle, p.MetaDescription, p.Canonical, keywords, notes, string(p.Status), nullString(p.EntityID), nullString(p.TemplateID),
		p.ContentHash, p.Observed.Link, p.Observed.Slug, p.Observed.Status, p.Observed.Title, p.Observed.H1,
		nullTime(p.WPModifiedAt), nullTime(p.LastSyncedAt), boolInt(p.Drift), formatTime(p.CreatedAt), formatTime(p.UpdatedAt),
	}, pageConflict(p.Path), "insert the page")
	return err
}

func (r *PageRepo) Update(ctx context.Context, p pagemap.Page) error {
	keywords, notes, err := encodePlan(p)
	if err != nil {
		return err
	}
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), updatePage, []any{
		p.Path, p.PlannedPath, p.Slug, nullString(p.ParentPageID),
		storedParentReason(p.ParentReason), string(p.ParentMethod), string(p.TechnicalReason), string(p.ReadOnlyReason), string(p.WPType), nullInt(p.WPID),
		p.Title, p.H1, p.MetaTitle, p.MetaDescription, p.Canonical, keywords, notes, string(p.Status), nullString(p.EntityID), nullString(p.TemplateID),
		p.ContentHash, p.Observed.Link, p.Observed.Slug, p.Observed.Status, p.Observed.Title, p.Observed.H1,
		nullTime(p.WPModifiedAt), nullTime(p.LastSyncedAt), boolInt(p.Drift), formatTime(p.UpdatedAt), p.ID,
	}, pageConflict(p.Path), "update the page")
	return requireAffected(affected, err, pageNotFound(p.ID))
}

func (r *PageRepo) UpdateDerived(ctx context.Context, p pagemap.Page) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), updateDerived, []any{
		nullString(p.ParentPageID), storedParentReason(p.ParentReason), string(p.ParentMethod),
		string(p.TechnicalReason), string(p.ReadOnlyReason), string(p.Status), formatTime(p.UpdatedAt), p.ID,
	}, nil, "update the page's derived facts")
	return requireAffected(affected, err, pageNotFound(p.ID))
}

func (r *PageRepo) UpdatePublished(ctx context.Context, p pagemap.Page) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), updatePublished, []any{
		nullInt(p.WPID), string(p.Status), p.ContentHash, p.Observed.Link, p.Observed.Slug, p.Observed.Status, p.Observed.Title,
		p.Observed.H1, nullTime(p.WPModifiedAt), nullTime(p.LastSyncedAt), boolInt(p.Drift), formatTime(p.UpdatedAt), p.ID,
	}, nil, "update what a publish wrote to the page")
	return requireAffected(affected, err, pageNotFound(p.ID))
}

func (r *PageRepo) SetType(ctx context.Context, pageID, typeID string, reason pagemap.TypeReason, at time.Time) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), setPageType, []any{typeID, string(reason), formatTime(at), pageID}, nil, "set the page's type")
	return requireAffected(affected, err, pageNotFound(pageID))
}

func (r *PageRepo) AutoType(ctx context.Context, pageID, typeID string, reason pagemap.TypeReason, at time.Time) error {
	_, err := execWrite(ctx, r.store.writeFrom(ctx), autoPageType, []any{typeID, string(reason), formatTime(at), pageID}, nil, "type the untyped page")
	return err
}

func (r *PageRepo) ReassignType(ctx context.Context, fromTypeID, toTypeID string, reason pagemap.TypeReason, at time.Time) (int64, error) {
	return execWrite(ctx, r.store.writeFrom(ctx), reassignPageType, []any{toTypeID, string(reason), formatTime(at), fromTypeID}, nil, "reassign the pages of a type")
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
	if q.EntityIDs != nil {
		builder = builder.Where(squirrel.Eq{"entity_id": q.EntityIDs})
	}
	if q.Unmapped {
		builder = builder.Where("entity_id IS NULL")
	}
	if q.PathPrefix != "" {
		builder = builder.Where(`path LIKE ? ESCAPE '\'`, escapeLike(q.PathPrefix)+"%")
	}

	return selectKeyed(ctx, r.store.execFrom(ctx), builder, pageKeyset(q), page, scanPage, "pages")
}

func storedParentReason(reason pagemap.ParentReason) string {
	if reason == "" {
		return string(pagemap.ParentNone)
	}
	return string(reason)
}

func scanPage(rows *sql.Rows) (pagemap.Page, error) {
	var (
		p                              pagemap.Page
		parentID, entityID, templateID sql.NullString
		pageTypeID                     sql.NullString
		typeSource, typeReason         string
		parentReason, parentMethod     string
		technicalReason                string
		readOnlyReason                 string
		wpType, status                 string
		wpID                           sql.NullInt64
		wpModifiedAt, lastSyncedAt     sql.NullString
		drift                          int64
		createdAt, updatedAt           string
		keywords, notes                string
	)
	if err := rows.Scan(&p.ID, &p.SiteID, &p.Path, &p.PlannedPath, &p.Slug, &parentID, &parentReason, &parentMethod, &pageTypeID, &typeSource, &typeReason, &technicalReason, &readOnlyReason,
		&wpType, &wpID, &p.Title, &p.H1, &p.MetaTitle, &p.MetaDescription, &p.Canonical,
		&keywords, &notes, &status, &entityID, &templateID, &p.ContentHash, &p.Observed.Link, &p.Observed.Slug, &p.Observed.Status,
		&p.Observed.Title, &p.Observed.H1, &wpModifiedAt, &lastSyncedAt, &drift, &createdAt, &updatedAt); err != nil {
		return pagemap.Page{}, err
	}
	var stored []note.Note
	if err := decodeJSON(notes, &stored, "decode the page notes"); err != nil {
		return pagemap.Page{}, err
	}
	p.Notes = note.Clean(stored)
	p.ParentPageID = optString(parentID)
	p.ParentReason = pagemap.ParentReason(parentReason)
	p.ParentMethod = pagemap.ParentMethod(parentMethod)
	p.PageTypeID = optString(pageTypeID)
	p.TypeSource = pagemap.TypeSource(typeSource)
	p.TypeReason = pagemap.TypeReason(typeReason)
	p.TechnicalReason = pagemap.TechnicalReason(technicalReason)
	p.ReadOnlyReason = pagemap.ReadOnlyReason(readOnlyReason)
	p.WPType = pagemap.WPType(wpType)
	p.WPID = optInt(wpID)
	p.Status = pagemap.Status(status)
	p.EntityID = optString(entityID)
	p.TemplateID = optString(templateID)
	p.Drift = drift == 1

	var err error
	if p.Keywords, err = decodeKeywords(keywords, "decode the page keywords"); err != nil {
		return pagemap.Page{}, err
	}
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
