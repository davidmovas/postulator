package sqlite

import (
	"cmp"
	"context"
	"database/sql"
	"slices"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"

	"github.com/davidmovas/postulator/internal/adapters/sqlite/dbx"
	"github.com/davidmovas/postulator/internal/domain/keyword"
	"github.com/davidmovas/postulator/internal/domain/note"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	pageColumns       = `id, site_id, path, planned_path, planned_trail, row_key, slug, parent_page_id, parent_reason, parent_method, parent_source, page_type_id, page_type_source, page_type_reason, page_type_rule_id, technical_reason, read_only_reason, wp_type, wp_id, title, h1, meta_title, meta_description, keywords, notes, status, entity_id, template_id, wp_modified_at, last_synced_at, created_at, updated_at`
	insertPage        = `INSERT INTO pages (` + pageColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	updatePage        = `UPDATE pages SET path = ?, planned_path = ?, planned_trail = ?, row_key = ?, slug = ?, parent_page_id = ?, parent_reason = ?, parent_method = ?, technical_reason = ?, read_only_reason = ?, wp_type = ?, wp_id = ?, title = ?, h1 = ?, meta_title = ?, meta_description = ?, keywords = ?, notes = ?, status = ?, entity_id = ?, template_id = ?, wp_modified_at = ?, last_synced_at = ?, updated_at = ? WHERE id = ?`
	setParentSource   = `UPDATE pages SET parent_source = ?, updated_at = ? WHERE id = ?`
	setRowKey         = `UPDATE pages SET row_key = ?, planned_trail = CASE WHEN planned_trail = '[]' THEN ? ELSE planned_trail END, updated_at = ? WHERE id = ? AND site_id = ?`
	releaseRowKey     = `UPDATE pages SET row_key = '', updated_at = ? WHERE site_id = ? AND row_key = ? AND id <> ?`
	updateDerived     = `UPDATE pages SET parent_page_id = ?, parent_reason = ?, parent_method = ?, technical_reason = ?, read_only_reason = ?, status = ?, updated_at = ? WHERE id = ?`
	followObservedH1  = `UPDATE pages SET h1 = ?, updated_at = ? WHERE id = ? AND h1 = ? AND h1 <> ?`
	updatePublished   = `UPDATE pages SET wp_id = ?, status = ?, wp_modified_at = ?, last_synced_at = ?, updated_at = ? WHERE id = ?`
	setPageType       = `UPDATE pages SET page_type_id = ?, page_type_source = 'manual', page_type_reason = ?, page_type_rule_id = '', updated_at = ? WHERE id = ?`
	reassignPageType  = `UPDATE pages SET page_type_id = ?, page_type_source = 'manual', page_type_reason = ?, page_type_rule_id = '', updated_at = ? WHERE page_type_id = ?`
	classifyPageType  = `UPDATE pages SET page_type_id = ?, page_type_source = 'auto', page_type_reason = ?, page_type_rule_id = ?, updated_at = ? WHERE id = ? AND page_type_id IS ? AND page_type_source = ? AND page_type_reason = ? AND page_type_rule_id = ?`
	deletePage        = `DELETE FROM pages WHERE id = ?`
	selectPage        = `SELECT ` + pageColumns + ` FROM pages WHERE id = ?`
	selectPagesBySite = `SELECT ` + pageColumns + ` FROM pages WHERE site_id = ? ORDER BY path, id`
	selectPagesByIDs  = `SELECT ` + pageColumns + ` FROM pages WHERE id IN `
	orderPagesByPath  = ` ORDER BY path, id`
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

type encodedPlan struct {
	keywords, notes, trail string
}

func encodePlan(p pagemap.Page) (encodedPlan, error) {
	var (
		out encodedPlan
		err error
	)
	if out.keywords, err = encodeJSON(keyword.New(p.Keywords)); err != nil {
		return encodedPlan{}, err
	}
	if out.notes, err = encodeJSON(note.Clean(p.Notes)); err != nil {
		return encodedPlan{}, err
	}
	trail := p.PlannedTrail
	if trail == nil {
		trail = []string{}
	}
	if out.trail, err = encodeJSON(trail); err != nil {
		return encodedPlan{}, err
	}
	return out, nil
}

func storedParentSource(source pagemap.ParentSource) string {
	if source == pagemap.ParentFromSite {
		return "site"
	}
	return string(source)
}

func (r *PageRepo) SetParentSource(ctx context.Context, pageID string, source pagemap.ParentSource, at time.Time) error {
	if !source.Valid() {
		return errors.New(errors.Invalid, "the parent source is not recognized").WithDetail("parentSource", string(source))
	}
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), setParentSource, []any{storedParentSource(source), formatTime(at), pageID},
		nil, "set the page's parent source")
	return requireAffected(affected, err, pageNotFound(pageID))
}

func (r *PageRepo) SetRowKey(ctx context.Context, siteID, pageID, rowKey string, trail []string, at time.Time) error {
	if strings.TrimSpace(rowKey) == "" {
		return errors.New(errors.Invalid, "a row key must not be empty").WithDetail("field", "rowKey")
	}
	if trail == nil {
		trail = []string{}
	}
	encoded, err := encodeJSON(trail)
	if err != nil {
		return err
	}
	return r.store.Do(ctx, func(c context.Context) error {
		affected, setErr := execWrite(c, r.store.writeFrom(c), setRowKey, []any{rowKey, encoded, formatTime(at), pageID, siteID}, nil,
			"move the row key onto the page")
		if missing := requireAffected(affected, setErr, pageNotFound(pageID).WithDetail("siteId", siteID)); missing != nil {
			return missing
		}
		_, releaseErr := execWrite(c, r.store.writeFrom(c), releaseRowKey, []any{formatTime(at), siteID, rowKey, pageID}, nil,
			"take the row key off the site's other pages")
		return releaseErr
	})
}

func (r *PageRepo) Insert(ctx context.Context, p pagemap.Page) error {
	plan, err := encodePlan(p)
	if err != nil {
		return err
	}
	keywords, notes := plan.keywords, plan.notes
	_, err = execWrite(ctx, r.store.writeFrom(ctx), insertPage, []any{
		p.ID, p.SiteID, p.Path, p.PlannedPath, plan.trail, p.RowKey, p.Slug, nullString(p.ParentPageID),
		storedParentReason(p.ParentReason), string(p.ParentMethod), storedParentSource(p.ParentSource),
		nullString(p.PageTypeID), string(p.TypeSource), string(p.TypeReason), p.TypeRuleID,
		string(p.TechnicalReason), string(p.ReadOnlyReason), string(p.WPType), nullInt(p.WPID),
		p.Title, p.H1, p.MetaTitle, p.MetaDescription, keywords, notes, string(p.Status), nullString(p.EntityID), nullString(p.TemplateID),
		nullTime(p.WPModifiedAt), nullTime(p.LastSyncedAt), formatTime(p.CreatedAt), formatTime(p.UpdatedAt),
	}, pageConflict(p.Path), "insert the page")
	return err
}

func (r *PageRepo) Update(ctx context.Context, p pagemap.Page) error {
	plan, err := encodePlan(p)
	if err != nil {
		return err
	}
	keywords, notes := plan.keywords, plan.notes
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), updatePage, []any{
		p.Path, p.PlannedPath, plan.trail, p.RowKey, p.Slug, nullString(p.ParentPageID),
		storedParentReason(p.ParentReason), string(p.ParentMethod), string(p.TechnicalReason), string(p.ReadOnlyReason), string(p.WPType), nullInt(p.WPID),
		p.Title, p.H1, p.MetaTitle, p.MetaDescription, keywords, notes, string(p.Status), nullString(p.EntityID), nullString(p.TemplateID),
		nullTime(p.WPModifiedAt), nullTime(p.LastSyncedAt), formatTime(p.UpdatedAt), p.ID,
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

func (r *PageRepo) FollowObservedH1(ctx context.Context, pageID, was, now string, at time.Time) error {
	_, err := execWrite(ctx, r.store.writeFrom(ctx), followObservedH1, []any{now, formatTime(at), pageID, was, now}, nil,
		"let the page's h1 follow the observed one")
	return err
}

func (r *PageRepo) UpdatePublished(ctx context.Context, p pagemap.Page) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), updatePublished, []any{
		nullInt(p.WPID), string(p.Status), nullTime(p.WPModifiedAt), nullTime(p.LastSyncedAt), formatTime(p.UpdatedAt), p.ID,
	}, nil, "update what a publish wrote to the page")
	return requireAffected(affected, err, pageNotFound(p.ID))
}

func (r *PageRepo) SetType(ctx context.Context, pageID, typeID string, reason pagemap.TypeReason, at time.Time) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), setPageType, []any{typeID, string(reason), formatTime(at), pageID}, nil, "set the page's type")
	return requireAffected(affected, err, pageNotFound(pageID))
}

func (r *PageRepo) ReassignType(ctx context.Context, fromTypeID, toTypeID string, reason pagemap.TypeReason, at time.Time) (int64, error) {
	return execWrite(ctx, r.store.writeFrom(ctx), reassignPageType, []any{toTypeID, string(reason), formatTime(at), fromTypeID}, nil, "reassign the pages of a type")
}

func (r *PageRepo) Classify(ctx context.Context, pageID string, was, now pagetype.Assignment, at time.Time) (bool, error) {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), classifyPageType, []any{
		now.TypeID, string(now.Reason), now.RuleID, formatTime(at),
		pageID, nullString(optional(was.TypeID)), string(was.Source), string(was.Reason), was.RuleID,
	}, nil, "type the page by the site's rules")
	return affected == 1, err
}

func (r *PageRepo) SetTypes(ctx context.Context, siteID string, pageIDs []string, typeID string, reason pagemap.TypeReason, at time.Time) (int64, error) {
	if len(pageIDs) == 0 {
		return 0, nil
	}
	args := append([]any{typeID, string(reason), formatTime(at), siteID, typeID, string(reason)}, idArgs(pageIDs)...)
	return execWrite(ctx, r.store.writeFrom(ctx), `UPDATE pages SET page_type_id = ?, page_type_source = 'manual', page_type_reason = ?,
		page_type_rule_id = '', updated_at = ? WHERE site_id = ? AND NOT (page_type_id IS ? AND page_type_source = 'manual' AND page_type_reason = ?)
		AND id IN (`+placeholders(len(pageIDs))+`)`, args, nil, "type the chosen pages")
}

func (r *PageRepo) ClearTypes(ctx context.Context, siteID string, pageIDs []string, at time.Time) (int64, error) {
	if len(pageIDs) == 0 {
		return 0, nil
	}
	args := append([]any{formatTime(at), siteID}, idArgs(pageIDs)...)
	return execWrite(ctx, r.store.writeFrom(ctx), `UPDATE pages SET page_type_id = NULL, page_type_source = '', page_type_reason = '',
		page_type_rule_id = '', updated_at = ? WHERE site_id = ? AND id IN (`+placeholders(len(pageIDs))+`)`, args, nil,
		"hand the chosen pages back to the rules")
}

func (r *PageRepo) SetTemplates(ctx context.Context, siteID string, pageIDs []string, templateID *string, at time.Time) (int64, error) {
	if len(pageIDs) == 0 {
		return 0, nil
	}
	args := append([]any{nullString(templateID), formatTime(at), siteID, nullString(templateID)}, idArgs(pageIDs)...)
	return execWrite(ctx, r.store.writeFrom(ctx), `UPDATE pages SET template_id = ?, updated_at = ?
		WHERE site_id = ? AND template_id IS NOT ? AND id IN (`+placeholders(len(pageIDs))+`)`, args, nil, "set the chosen pages' template")
}

func idArgs(ids []string) []any {
	out := make([]any, 0, len(ids))
	for _, pageID := range ids {
		out = append(out, pageID)
	}
	return out
}

func optional(value string) *string {
	if value == "" {
		return nil
	}
	return &value
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

func (r *PageRepo) ListByIDs(ctx context.Context, ids []string) ([]pagemap.Page, error) {
	pages, err := selectByIDs(ctx, r.store.execFrom(ctx), selectPagesByIDs, orderPagesByPath, ids, scanPage, "list the pages by id")
	if err != nil {
		return nil, err
	}
	slices.SortFunc(pages, func(a, b pagemap.Page) int { return cmp.Or(cmp.Compare(a.Path, b.Path), cmp.Compare(a.ID, b.ID)) })
	return pages, nil
}

func pageKeyset(q pagemap.Query) paging.Keyset[pagemap.Page] {
	key := paging.TimeKey[pagemap.Page]("createdAt", "created_at", func(p pagemap.Page) any { return p.CreatedAt })
	switch q.Sort {
	case pagemap.SortPath:
		key = paging.TextKey[pagemap.Page]("path", "path", func(p pagemap.Page) any { return p.Path })
	case pagemap.SortTitle:
		key = paging.TextKey[pagemap.Page]("title", "title", func(p pagemap.Page) any { return p.Title })
	}
	return paging.Keyset[pagemap.Page]{
		IDColumn: "id",
		ID:       func(p pagemap.Page) string { return p.ID },
		Keys:     []paging.SortKey[pagemap.Page]{key},
		Desc:     q.Desc,
	}
}

func filtered(builder squirrel.SelectBuilder, q pagemap.Query) squirrel.SelectBuilder {
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
	if q.PageTypeID != "" {
		builder = builder.Where(`EXISTS (SELECT 1 FROM page_map AS m WHERE m.page_id = pages.id AND m.type_id = ?)`, q.PageTypeID)
	}
	if q.Trail != "" {
		builder = builder.Where(`EXISTS (SELECT 1 FROM page_map AS m WHERE m.page_id = pages.id AND m.trail LIKE ? ESCAPE '\')`,
			escapeLike(q.Trail)+"%")
	}
	if q.Search != "" {
		builder = builder.Where(`EXISTS (SELECT 1 FROM page_map AS m WHERE m.page_id = pages.id AND (instr(m.title_key, ?) > 0 OR instr(m.path_key, ?) > 0))`,
			q.Search, q.Search)
	}
	switch q.Body {
	case pagemap.BodyWith:
		builder = builder.Where(`EXISTS (SELECT 1 FROM page_bodies AS b WHERE b.page_id = pages.id AND b.side = 'current' AND b.raw <> '')`)
	case pagemap.BodyWithout:
		builder = builder.Where(`NOT EXISTS (SELECT 1 FROM page_bodies AS b WHERE b.page_id = pages.id AND b.side = 'current' AND b.raw <> '')`)
	}
	if q.Findings {
		builder = builder.Where(`EXISTS (SELECT 1 FROM findings AS f WHERE f.page_id = pages.id AND f.status = 'open')`)
	}
	return builder
}

func (r *PageRepo) List(ctx context.Context, q pagemap.Query, page paging.Request) (paging.List[pagemap.Page], error) {
	return selectKeyed(ctx, r.store.execFrom(ctx), filtered(squirrel.Select(pageColumns).From("pages"), q), pageKeyset(q), page, scanPage, "pages")
}

func (r *PageRepo) Count(ctx context.Context, q pagemap.Query) (int, error) {
	query, args, err := filtered(squirrel.Select("count(*)").From("pages"), q).ToSql()
	if err != nil {
		return 0, errors.Wrap(err, errors.Internal, "build the page count")
	}
	var count int
	if scanErr := r.store.execFrom(ctx).QueryRowContext(ctx, query, args...).Scan(&count); scanErr != nil {
		return 0, dbx.Convert(scanErr, "count the pages")
	}
	return count, nil
}

func (r *PageRepo) IDs(ctx context.Context, q pagemap.Query) ([]string, error) {
	query, args, err := filtered(squirrel.Select("id").From("pages"), q).OrderBy("path", "id").ToSql()
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "build the page ids")
	}
	return selectAll(ctx, r.store.execFrom(ctx), query, args, scanText, "list the pages' ids")
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
		parentSource, trail            string
		technicalReason                string
		readOnlyReason                 string
		wpType, status                 string
		wpID                           sql.NullInt64
		wpModifiedAt, lastSyncedAt     sql.NullString
		createdAt, updatedAt           string
		keywords, notes                string
	)
	if err := rows.Scan(&p.ID, &p.SiteID, &p.Path, &p.PlannedPath, &trail, &p.RowKey, &p.Slug, &parentID, &parentReason, &parentMethod, &parentSource,
		&pageTypeID, &typeSource, &typeReason, &p.TypeRuleID, &technicalReason, &readOnlyReason,
		&wpType, &wpID, &p.Title, &p.H1, &p.MetaTitle, &p.MetaDescription,
		&keywords, &notes, &status, &entityID, &templateID, &wpModifiedAt, &lastSyncedAt, &createdAt, &updatedAt); err != nil {
		return pagemap.Page{}, err
	}
	var stored []note.Note
	if err := decodeJSON(notes, &stored, "decode the page notes"); err != nil {
		return pagemap.Page{}, err
	}
	if err := decodeJSON(trail, &p.PlannedTrail, "decode the page's planned trail"); err != nil {
		return pagemap.Page{}, err
	}
	if len(p.PlannedTrail) == 0 {
		p.PlannedTrail = nil
	}
	p.Notes = note.Clean(stored)
	p.ParentPageID = optString(parentID)
	p.ParentReason = pagemap.ParentReason(parentReason)
	p.ParentMethod = pagemap.ParentMethod(parentMethod)
	if parentSource != "site" {
		p.ParentSource = pagemap.ParentSource(parentSource)
	}
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
