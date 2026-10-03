package sqlite

import (
	"context"
	"database/sql"

	"github.com/davidmovas/postulator/internal/domain/category"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	categoryTermColumns = `category_id, site_id, taxonomy, term_id, parent_term_id, name, run_id, seen_at`
	upsertCategoryTerm  = `INSERT INTO category_terms (` + categoryTermColumns + `) SELECT ?, ?, ?, ?, ?, ?, ?, ? ` +
		`WHERE EXISTS (SELECT 1 FROM categories WHERE id = ? AND site_id = ?) ` +
		`ON CONFLICT (category_id, taxonomy) DO UPDATE SET term_id = excluded.term_id, parent_term_id = excluded.parent_term_id, ` +
		`name = excluded.name, run_id = excluded.run_id, seen_at = excluded.seen_at`
	deleteCategoryTerm        = `DELETE FROM category_terms WHERE category_id = ? AND taxonomy = ?`
	selectCategoryTermsBySite = `SELECT ` + categoryTermColumns + ` FROM category_terms WHERE site_id = ? ORDER BY taxonomy, term_id, category_id`
)

type CategoryTermRepo struct {
	store *Store
}

func NewCategoryTermRepo(store *Store) *CategoryTermRepo {
	return &CategoryTermRepo{store: store}
}

func categoryTermNotFound(categoryID string, taxonomy category.Taxonomy) *errors.Error {
	return errors.New(errors.NotFound, "the category has no term in this taxonomy").
		WithDetail("categoryId", categoryID).WithDetail("taxonomy", string(taxonomy))
}

func categoryTermElsewhere(t category.Term) *errors.Error {
	return errors.New(errors.Invalid, "the term's category is not a category of this site").
		WithDetail("field", "categoryId").WithDetail("categoryId", t.CategoryID).WithDetail("siteId", t.SiteID)
}

func (r *CategoryTermRepo) Upsert(ctx context.Context, t category.Term) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), upsertCategoryTerm, []any{
		t.CategoryID, t.SiteID, string(t.Taxonomy), t.TermID, t.ParentTermID, t.Name, t.RunID, formatTime(t.SeenAt),
		t.CategoryID, t.SiteID,
	}, nil, "save the category term")
	return requireAffected(affected, err, categoryTermElsewhere(t))
}

func (r *CategoryTermRepo) Delete(ctx context.Context, categoryID string, taxonomy category.Taxonomy) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), deleteCategoryTerm, []any{categoryID, string(taxonomy)}, nil, "delete the category term")
	return requireAffected(affected, err, categoryTermNotFound(categoryID, taxonomy))
}

func (r *CategoryTermRepo) ListBySite(ctx context.Context, siteID string) ([]category.Term, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectCategoryTermsBySite, []any{siteID}, scanCategoryTerm, "list the site category terms")
}

func scanCategoryTerm(rows *sql.Rows) (category.Term, error) {
	var (
		t                category.Term
		taxonomy, seenAt string
	)
	if err := rows.Scan(&t.CategoryID, &t.SiteID, &taxonomy, &t.TermID, &t.ParentTermID, &t.Name, &t.RunID, &seenAt); err != nil {
		return category.Term{}, err
	}
	t.Taxonomy = category.Taxonomy(taxonomy)

	var err error
	if t.SeenAt, err = parseTime(seenAt); err != nil {
		return category.Term{}, err
	}
	return t, nil
}
