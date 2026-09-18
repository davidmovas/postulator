package sqlite

import (
	"context"
	"database/sql"

	"github.com/Masterminds/squirrel"

	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	policyColumns = `id, scope, site_id, name, rules, forbid_external, forbid_self, anchor_strategy, created_at, updated_at`
	insertPolicy  = `INSERT INTO link_policies (` + policyColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	updatePolicy  = `UPDATE link_policies SET name = ?, rules = ?, forbid_external = ?, forbid_self = ?, anchor_strategy = ?, updated_at = ? WHERE id = ?`
	deletePolicy  = `DELETE FROM link_policies WHERE id = ?`
	selectPolicy  = `SELECT ` + policyColumns + ` FROM link_policies WHERE id = ?`
)

type LinkPolicyRepo struct {
	store *Store
}

func NewLinkPolicyRepo(store *Store) *LinkPolicyRepo {
	return &LinkPolicyRepo{store: store}
}

func policyNotFound(id string) *errors.Error {
	return errors.New(errors.NotFound, "link policy not found").WithDetail("linkPolicyId", id)
}

func policyConflict(name string) *errors.Error {
	return errors.New(errors.Conflict, "a link policy with this name already exists in this scope").WithDetail("name", name)
}

func (r *LinkPolicyRepo) Insert(ctx context.Context, p template.LinkPolicy) error {
	rules, err := encodeJSON(p.Rules)
	if err != nil {
		return err
	}
	_, err = execWrite(ctx, r.store.writeFrom(ctx), insertPolicy, []any{
		p.ID, string(p.Scope), nullString(p.SiteID), p.Name, rules, boolInt(p.ForbidExternal), boolInt(p.ForbidSelf), string(p.AnchorStrategy),
		formatTime(p.CreatedAt), formatTime(p.UpdatedAt),
	}, policyConflict(p.Name), "insert the link policy")
	return err
}

func (r *LinkPolicyRepo) Update(ctx context.Context, p template.LinkPolicy) error {
	rules, err := encodeJSON(p.Rules)
	if err != nil {
		return err
	}
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), updatePolicy, []any{
		p.Name, rules, boolInt(p.ForbidExternal), boolInt(p.ForbidSelf), string(p.AnchorStrategy), formatTime(p.UpdatedAt), p.ID,
	}, policyConflict(p.Name), "update the link policy")
	return requireAffected(affected, err, policyNotFound(p.ID))
}

func (r *LinkPolicyRepo) Delete(ctx context.Context, id string) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), deletePolicy, []any{id}, nil, "delete the link policy")
	return requireAffected(affected, err, policyNotFound(id))
}

func (r *LinkPolicyRepo) Get(ctx context.Context, id string) (template.LinkPolicy, error) {
	return selectOne(ctx, r.store.execFrom(ctx), selectPolicy, []any{id}, scanPolicy, policyNotFound(id), "read the link policy")
}

func policyKeyset(q template.PolicyQuery) paging.Keyset[template.LinkPolicy] {
	key := paging.TimeKey[template.LinkPolicy]("createdAt", "created_at", func(p template.LinkPolicy) any { return p.CreatedAt })
	if q.Sort == template.SortName {
		key = paging.TextKey[template.LinkPolicy]("name", "name", func(p template.LinkPolicy) any { return p.Name })
	}
	return paging.Keyset[template.LinkPolicy]{
		IDColumn: "id",
		ID:       func(p template.LinkPolicy) string { return p.ID },
		Keys:     []paging.SortKey[template.LinkPolicy]{key},
		Desc:     q.Desc,
	}
}

func (r *LinkPolicyRepo) List(ctx context.Context, q template.PolicyQuery, page paging.Request) (paging.List[template.LinkPolicy], error) {
	builder := squirrel.Select(policyColumns).From("link_policies")
	if q.Scope != nil {
		builder = builder.Where(squirrel.Eq{"scope": string(*q.Scope)})
	}
	if q.SiteID != nil {
		builder = builder.Where(squirrel.Eq{"site_id": *q.SiteID})
	}
	if q.Name != "" {
		builder = builder.Where(squirrel.Eq{"name": q.Name})
	}

	keyset := policyKeyset(q)
	keyed, err := keyset.Apply(builder, page)
	if err != nil {
		return paging.List[template.LinkPolicy]{}, err
	}
	query, args, err := buildQuery(keyed, "link policies")
	if err != nil {
		return paging.List[template.LinkPolicy]{}, err
	}
	rows, err := selectAll(ctx, r.store.execFrom(ctx), query, args, scanPolicy, "list the link policies")
	if err != nil {
		return paging.List[template.LinkPolicy]{}, err
	}
	return keyset.Cut(rows, page)
}

func scanPolicy(rows *sql.Rows) (template.LinkPolicy, error) {
	var (
		p                          template.LinkPolicy
		scope, rules, strategy     string
		siteID                     sql.NullString
		forbidExternal, forbidSelf int64
		createdAt, updatedAt       string
	)
	if err := rows.Scan(&p.ID, &scope, &siteID, &p.Name, &rules, &forbidExternal, &forbidSelf, &strategy, &createdAt, &updatedAt); err != nil {
		return template.LinkPolicy{}, err
	}
	p.Scope = template.Scope(scope)
	p.SiteID = optString(siteID)
	p.ForbidExternal = forbidExternal == 1
	p.ForbidSelf = forbidSelf == 1
	p.AnchorStrategy = template.AnchorStrategy(strategy)
	if err := decodeJSON(rules, &p.Rules, "decode the link policy rules"); err != nil {
		return template.LinkPolicy{}, err
	}

	var err error
	if p.CreatedAt, err = parseTime(createdAt); err != nil {
		return template.LinkPolicy{}, err
	}
	if p.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return template.LinkPolicy{}, err
	}
	return p, nil
}
