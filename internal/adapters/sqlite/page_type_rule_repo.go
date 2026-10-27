package sqlite

import (
	"context"
	"database/sql"

	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	ruleColumns       = `id, site_id, kind, position, name, enabled, page_type_id, conditions, created_at, updated_at`
	insertRule        = `INSERT INTO page_type_rules (` + ruleColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	updateRule        = `UPDATE page_type_rules SET position = ?, name = ?, enabled = ?, page_type_id = ?, conditions = ?, updated_at = ? WHERE id = ?`
	deleteRule        = `DELETE FROM page_type_rules WHERE id = ?`
	selectRule        = `SELECT ` + ruleColumns + ` FROM page_type_rules WHERE id = ?`
	selectRulesOfSite = `SELECT ` + ruleColumns + ` FROM page_type_rules WHERE site_id = ? ORDER BY position, id`
)

type PageTypeRuleRepo struct {
	store *Store
}

func NewPageTypeRuleRepo(store *Store) *PageTypeRuleRepo {
	return &PageTypeRuleRepo{store: store}
}

func ruleNotFound(id string) *errors.Error {
	return errors.New(errors.NotFound, "classification rule not found").WithDetail("ruleId", id)
}

func ruleConflict() *errors.Error {
	return errors.New(errors.Conflict, "the site already has its rule for anything no rule matched").WithDetail("field", "kind")
}

func (r *PageTypeRuleRepo) Insert(ctx context.Context, rule pagetype.Rule) error {
	conditions, err := encodeJSON(rule.Conditions)
	if err != nil {
		return err
	}
	_, err = execWrite(ctx, r.store.writeFrom(ctx), insertRule, []any{
		rule.ID, rule.SiteID, string(rule.Kind), rule.Position, rule.Name, rule.Enabled, rule.TypeID, conditions,
		formatTime(rule.CreatedAt), formatTime(rule.UpdatedAt),
	}, ruleConflict(), "insert the classification rule")
	return err
}

func (r *PageTypeRuleRepo) Update(ctx context.Context, rule pagetype.Rule) error {
	conditions, err := encodeJSON(rule.Conditions)
	if err != nil {
		return err
	}
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), updateRule, []any{
		rule.Position, rule.Name, rule.Enabled, rule.TypeID, conditions, formatTime(rule.UpdatedAt), rule.ID,
	}, nil, "update the classification rule")
	return requireAffected(affected, err, ruleNotFound(rule.ID))
}

func (r *PageTypeRuleRepo) Delete(ctx context.Context, id string) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), deleteRule, []any{id}, nil, "delete the classification rule")
	return requireAffected(affected, err, ruleNotFound(id))
}

func (r *PageTypeRuleRepo) Get(ctx context.Context, id string) (pagetype.Rule, error) {
	return selectOne(ctx, r.store.execFrom(ctx), selectRule, []any{id}, scanRule, ruleNotFound(id), "read the classification rule")
}

func (r *PageTypeRuleRepo) ListBySite(ctx context.Context, siteID string) ([]pagetype.Rule, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectRulesOfSite, []any{siteID}, scanRule, "list the site's classification rules")
}

func scanRule(rows *sql.Rows) (pagetype.Rule, error) {
	var (
		rule                 pagetype.Rule
		kind, conditions     string
		createdAt, updatedAt string
	)
	if err := rows.Scan(&rule.ID, &rule.SiteID, &kind, &rule.Position, &rule.Name, &rule.Enabled, &rule.TypeID, &conditions,
		&createdAt, &updatedAt); err != nil {
		return pagetype.Rule{}, err
	}
	rule.Kind = pagetype.RuleKind(kind)
	if err := decodeJSON(conditions, &rule.Conditions, "decode the rule's conditions"); err != nil {
		return pagetype.Rule{}, err
	}
	var err error
	if rule.CreatedAt, err = parseTime(createdAt); err != nil {
		return pagetype.Rule{}, err
	}
	if rule.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return pagetype.Rule{}, err
	}
	return rule, nil
}
