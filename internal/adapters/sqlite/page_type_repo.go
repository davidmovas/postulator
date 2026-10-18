package sqlite

import (
	"context"
	"database/sql"

	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	pageTypeColumns     = `id, site_id, name, preset, template_id, created_at, updated_at`
	insertPageType      = `INSERT INTO page_types (` + pageTypeColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?)`
	updatePageType      = `UPDATE page_types SET name = ?, template_id = ?, updated_at = ? WHERE id = ?`
	deletePageType      = `DELETE FROM page_types WHERE id = ?`
	selectPageType      = `SELECT ` + pageTypeColumns + ` FROM page_types WHERE id = ?`
	selectTypesOfSite   = `SELECT ` + pageTypeColumns + ` FROM page_types WHERE site_id = ? ORDER BY created_at, id`
	fieldScopeColumns   = `page_type_id, field_key, enabled, ai_writes, instruction, min_len, max_len, rewrites, link_policy, label, provider, updated_at`
	putFieldScope       = `INSERT INTO field_scopes (` + fieldScopeColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT (page_type_id, field_key) DO UPDATE SET enabled = excluded.enabled, ai_writes = excluded.ai_writes, instruction = excluded.instruction, min_len = excluded.min_len, max_len = excluded.max_len, rewrites = excluded.rewrites, link_policy = excluded.link_policy, label = excluded.label, provider = excluded.provider, updated_at = excluded.updated_at`
	selectScopesOfSite  = `SELECT ` + scopedColumns + ` FROM field_scopes JOIN page_types ON page_types.id = field_scopes.page_type_id WHERE page_types.site_id = ? ORDER BY page_types.created_at, page_types.id, field_scopes.field_key`
	selectScopesOfAType = `SELECT ` + fieldScopeColumns + ` FROM field_scopes WHERE page_type_id = ? ORDER BY field_key`
	scopedColumns       = `field_scopes.page_type_id, field_scopes.field_key, field_scopes.enabled, field_scopes.ai_writes, field_scopes.instruction, field_scopes.min_len, field_scopes.max_len, field_scopes.rewrites, field_scopes.link_policy, field_scopes.label, field_scopes.provider, field_scopes.updated_at`
)

type PageTypeRepo struct {
	store *Store
}

func NewPageTypeRepo(store *Store) *PageTypeRepo {
	return &PageTypeRepo{store: store}
}

func pageTypeNotFound(id string) *errors.Error {
	return errors.New(errors.NotFound, "page type not found").WithDetail("pageTypeId", id)
}

func pageTypeConflict(name string) *errors.Error {
	return errors.New(errors.Conflict, "the site already has a page type of this name").WithDetail("name", name)
}

func (r *PageTypeRepo) Insert(ctx context.Context, t pagetype.Type) error {
	_, err := execWrite(ctx, r.store.writeFrom(ctx), insertPageType, []any{
		t.ID, t.SiteID, t.Name, string(t.Preset), nullString(t.TemplateID), formatTime(t.CreatedAt), formatTime(t.UpdatedAt),
	}, pageTypeConflict(t.Name), "insert the page type")
	return err
}

func (r *PageTypeRepo) Update(ctx context.Context, t pagetype.Type) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), updatePageType, []any{
		t.Name, nullString(t.TemplateID), formatTime(t.UpdatedAt), t.ID,
	}, pageTypeConflict(t.Name), "update the page type")
	return requireAffected(affected, err, pageTypeNotFound(t.ID))
}

func (r *PageTypeRepo) Delete(ctx context.Context, id string) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), deletePageType, []any{id}, nil, "delete the page type")
	return requireAffected(affected, err, pageTypeNotFound(id))
}

func (r *PageTypeRepo) Get(ctx context.Context, id string) (pagetype.Type, error) {
	return selectOne(ctx, r.store.execFrom(ctx), selectPageType, []any{id}, scanPageType, pageTypeNotFound(id), "read the page type")
}

func (r *PageTypeRepo) ListBySite(ctx context.Context, siteID string) ([]pagetype.Type, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectTypesOfSite, []any{siteID}, scanPageType, "list the site's page types")
}

func scanPageType(rows *sql.Rows) (pagetype.Type, error) {
	var (
		t                    pagetype.Type
		preset               string
		templateID           sql.NullString
		createdAt, updatedAt string
	)
	if err := rows.Scan(&t.ID, &t.SiteID, &t.Name, &preset, &templateID, &createdAt, &updatedAt); err != nil {
		return pagetype.Type{}, err
	}
	t.Preset = pagetype.Preset(preset)
	t.TemplateID = optString(templateID)
	var err error
	if t.CreatedAt, err = parseTime(createdAt); err != nil {
		return pagetype.Type{}, err
	}
	if t.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return pagetype.Type{}, err
	}
	return t, nil
}

type FieldScopeRepo struct {
	store *Store
}

func NewFieldScopeRepo(store *Store) *FieldScopeRepo {
	return &FieldScopeRepo{store: store}
}

func (r *FieldScopeRepo) Put(ctx context.Context, s field.Scope) error {
	policy, err := encodeJSON(s.Links)
	if err != nil {
		return err
	}
	_, err = execWrite(ctx, r.store.writeFrom(ctx), putFieldScope, []any{
		s.PageTypeID, string(s.Key), boolInt(s.Enabled), boolInt(s.AIWrites), s.Instruction, s.MinLength, s.MaxLength,
		boolInt(s.Rewrites), policy, s.Label, string(s.Provider), formatTime(s.UpdatedAt),
	}, nil, "put the field scope")
	return err
}

func (r *FieldScopeRepo) ListBySite(ctx context.Context, siteID string) ([]field.Scope, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectScopesOfSite, []any{siteID}, scanFieldScope, "list the site's field scopes")
}

func (r *FieldScopeRepo) ListForType(ctx context.Context, typeID string) ([]field.Scope, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectScopesOfAType, []any{typeID}, scanFieldScope, "list the type's field scopes")
}

func scanFieldScope(rows *sql.Rows) (field.Scope, error) {
	var (
		s                                field.Scope
		key, policy, provider, updatedAt string
		enabled, aiWrites, rewrites      int64
		minLength, maxLength             int64
	)
	if err := rows.Scan(&s.PageTypeID, &key, &enabled, &aiWrites, &s.Instruction, &minLength, &maxLength, &rewrites,
		&policy, &s.Label, &provider, &updatedAt); err != nil {
		return field.Scope{}, err
	}
	s.Key = field.Key(key)
	s.Enabled, s.AIWrites, s.Rewrites = enabled == 1, aiWrites == 1, rewrites == 1
	s.MinLength, s.MaxLength = int(minLength), int(maxLength)
	s.Provider = field.Provider(provider)
	var err error
	if s.Links, err = field.ParsePolicy([]byte(policy)); err != nil {
		return field.Scope{}, err
	}
	if s.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return field.Scope{}, err
	}
	return s, nil
}
