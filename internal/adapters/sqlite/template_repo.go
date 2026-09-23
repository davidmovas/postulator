package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/Masterminds/squirrel"

	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	templateColumns           = `id, scope, site_id, name, page_kind, version, spec, created_at, updated_at`
	insertTemplate            = `INSERT INTO templates (` + templateColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	updateTemplate            = `UPDATE templates SET name = ?, page_kind = ?, version = ?, spec = ?, updated_at = ? WHERE id = ?`
	deleteTemplate            = `DELETE FROM templates WHERE id = ?`
	selectTemplate            = `SELECT ` + templateColumns + ` FROM templates WHERE id = ?`
	overrideColumns           = `id, template_id, scope, site_id, page_id, patch, created_at, updated_at`
	insertOverride            = `INSERT INTO template_overrides (` + overrideColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	updateOverride            = `UPDATE template_overrides SET patch = ?, updated_at = ? WHERE id = ?`
	deleteOverride            = `DELETE FROM template_overrides WHERE id = ?`
	selectOverride            = `SELECT ` + overrideColumns + ` FROM template_overrides WHERE template_id = ? AND scope = ? AND coalesce(site_id, page_id) = ?`
	selectOverridesByTemplate = `SELECT ` + overrideColumns + ` FROM template_overrides WHERE template_id = ? ORDER BY scope, coalesce(site_id, page_id)`
)

type TemplateRepo struct {
	store *Store
}

func NewTemplateRepo(store *Store) *TemplateRepo {
	return &TemplateRepo{store: store}
}

func templateNotFound(id string) *errors.Error {
	return errors.New(errors.NotFound, "template not found").WithDetail("templateId", id)
}

func templateConflict(name string) *errors.Error {
	return errors.New(errors.Conflict, "a template with this name already exists in this scope").WithDetail("name", name)
}

func overrideNotFound(templateID string, scope template.OverrideScope, targetID string) *errors.Error {
	return errors.New(errors.NotFound, "template override not found").
		WithDetail("templateId", templateID).WithDetail("scope", string(scope)).WithDetail("targetId", targetID)
}

func (r *TemplateRepo) Insert(ctx context.Context, t template.Template) error {
	spec, err := encodeJSON(t.Spec)
	if err != nil {
		return err
	}
	_, err = execWrite(ctx, r.store.writeFrom(ctx), insertTemplate, []any{
		t.ID, string(t.Scope), nullString(t.SiteID), t.Name, t.PageKind, t.Version, spec, formatTime(t.CreatedAt), formatTime(t.UpdatedAt),
	}, templateConflict(t.Name), "insert the template")
	return err
}

func (r *TemplateRepo) Update(ctx context.Context, t template.Template) error {
	spec, err := encodeJSON(t.Spec)
	if err != nil {
		return err
	}
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), updateTemplate, []any{
		t.Name, t.PageKind, t.Version, spec, formatTime(t.UpdatedAt), t.ID,
	}, templateConflict(t.Name), "update the template")
	return requireAffected(affected, err, templateNotFound(t.ID))
}

func (r *TemplateRepo) Delete(ctx context.Context, id string) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), deleteTemplate, []any{id}, nil, "delete the template")
	return requireAffected(affected, err, templateNotFound(id))
}

func (r *TemplateRepo) Get(ctx context.Context, id string) (template.Template, error) {
	return selectOne(ctx, r.store.execFrom(ctx), selectTemplate, []any{id}, scanTemplate, templateNotFound(id), "read the template")
}

func templateKeyset(q template.Query) paging.Keyset[template.Template] {
	key := paging.TimeKey[template.Template]("createdAt", "created_at", func(t template.Template) any { return t.CreatedAt })
	if q.Sort == template.SortName {
		key = paging.TextKey[template.Template]("name", "name", func(t template.Template) any { return t.Name })
	}
	return paging.Keyset[template.Template]{
		IDColumn: "id",
		ID:       func(t template.Template) string { return t.ID },
		Keys:     []paging.SortKey[template.Template]{key},
		Desc:     q.Desc,
	}
}

func (r *TemplateRepo) List(ctx context.Context, q template.Query, page paging.Request) (paging.List[template.Template], error) {
	builder := squirrel.Select(templateColumns).From("templates")
	if q.Scope != nil {
		builder = builder.Where(squirrel.Eq{"scope": string(*q.Scope)})
	}
	if q.SiteID != nil {
		builder = builder.Where(squirrel.Eq{"site_id": *q.SiteID})
	}
	if q.PageKind != "" {
		builder = builder.Where(squirrel.Eq{"page_kind": q.PageKind})
	}
	if q.Name != "" {
		builder = builder.Where(squirrel.Eq{"name": q.Name})
	}

	keyset := templateKeyset(q)
	keyed, err := keyset.Apply(builder, page)
	if err != nil {
		return paging.List[template.Template]{}, err
	}
	query, args, err := buildQuery(keyed, "templates")
	if err != nil {
		return paging.List[template.Template]{}, err
	}
	rows, err := selectAll(ctx, r.store.execFrom(ctx), query, args, scanTemplate, "list the templates")
	if err != nil {
		return paging.List[template.Template]{}, err
	}
	return keyset.Cut(rows, page)
}

func overrideTarget(o *template.Override) (siteID, pageID any) {
	if o.Scope == template.OverrideSite {
		return o.TargetID, nil
	}
	return nil, o.TargetID
}

func (r *TemplateRepo) UpsertOverride(ctx context.Context, o template.Override) (template.Override, error) {
	existing, err := r.GetOverride(ctx, o.TemplateID, o.Scope, o.TargetID)
	switch {
	case err == nil:
		if _, updateErr := execWrite(ctx, r.store.writeFrom(ctx), updateOverride, []any{string(o.Patch), formatTime(o.UpdatedAt), existing.ID}, nil, "update the template override"); updateErr != nil {
			return template.Override{}, updateErr
		}
		existing.Patch = o.Patch
		existing.UpdatedAt = o.UpdatedAt
		return existing, nil
	case errors.IsCode(err, errors.NotFound):
		siteID, pageID := overrideTarget(&o)
		if _, insertErr := execWrite(ctx, r.store.writeFrom(ctx), insertOverride, []any{
			o.ID, o.TemplateID, string(o.Scope), siteID, pageID, string(o.Patch), formatTime(o.CreatedAt), formatTime(o.UpdatedAt),
		}, nil, "insert the template override"); insertErr != nil {
			return template.Override{}, insertErr
		}
		return o, nil
	default:
		return template.Override{}, err
	}
}

func (r *TemplateRepo) GetOverride(ctx context.Context, templateID string, scope template.OverrideScope, targetID string) (template.Override, error) {
	return selectOne(ctx, r.store.execFrom(ctx), selectOverride, []any{templateID, string(scope), targetID}, scanOverride, overrideNotFound(templateID, scope, targetID), "read the template override")
}

func (r *TemplateRepo) DeleteOverride(ctx context.Context, id string) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), deleteOverride, []any{id}, nil, "delete the template override")
	return requireAffected(affected, err, errors.New(errors.NotFound, "template override not found").WithDetail("overrideId", id))
}

func (r *TemplateRepo) ListOverrides(ctx context.Context, templateID string) ([]template.Override, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectOverridesByTemplate, []any{templateID}, scanOverride, "list the template overrides")
}

func scanTemplate(rows *sql.Rows) (template.Template, error) {
	var (
		t                    template.Template
		scope, spec          string
		siteID               sql.NullString
		createdAt, updatedAt string
	)
	if err := rows.Scan(&t.ID, &scope, &siteID, &t.Name, &t.PageKind, &t.Version, &spec, &createdAt, &updatedAt); err != nil {
		return template.Template{}, err
	}
	t.Scope = template.Scope(scope)
	t.SiteID = optString(siteID)
	if err := decodeJSON(spec, &t.Spec, "decode the template spec"); err != nil {
		return template.Template{}, err
	}

	var err error
	if t.CreatedAt, err = parseTime(createdAt); err != nil {
		return template.Template{}, err
	}
	if t.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return template.Template{}, err
	}
	return t, nil
}

func scanOverride(rows *sql.Rows) (template.Override, error) {
	var (
		o                    template.Override
		scope, patch         string
		siteID, pageID       sql.NullString
		createdAt, updatedAt string
	)
	if err := rows.Scan(&o.ID, &o.TemplateID, &scope, &siteID, &pageID, &patch, &createdAt, &updatedAt); err != nil {
		return template.Override{}, err
	}
	o.Scope = template.OverrideScope(scope)
	if siteID.Valid {
		o.TargetID = siteID.String
	} else {
		o.TargetID = pageID.String
	}
	o.Patch = json.RawMessage(patch)

	var err error
	if o.CreatedAt, err = parseTime(createdAt); err != nil {
		return template.Override{}, err
	}
	if o.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return template.Override{}, err
	}
	return o, nil
}
