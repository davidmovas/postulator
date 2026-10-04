package sqlite

import (
	"context"
	"database/sql"

	"github.com/davidmovas/postulator/internal/domain/category"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	categoryColumns = `id, site_id, name, name_key, parent_id, created_at, updated_at`
	insertCategory  = `INSERT INTO categories (` + categoryColumns + `) SELECT ?, ?, ?, ?, ?, ?, ? ` +
		`WHERE ? IS NULL OR EXISTS (SELECT 1 FROM categories WHERE id = ? AND site_id = ?)`
	selectCategoriesBySite = `WITH RECURSIVE tree (id, depth) AS (
		SELECT id, 0 FROM categories WHERE site_id = ? AND parent_id IS NULL
		UNION ALL
		SELECT child.id, tree.depth + 1 FROM categories AS child JOIN tree ON child.parent_id = tree.id
	)
	SELECT c.id, c.site_id, c.name, c.name_key, c.parent_id, c.created_at, c.updated_at
	FROM categories AS c JOIN tree ON tree.id = c.id
	ORDER BY tree.depth, c.name_key, c.name, c.id`
	unfileCategoryPages = `WITH RECURSIVE branch (id) AS (
		SELECT id FROM categories WHERE id = ?
		UNION
		SELECT child.id FROM categories AS child JOIN branch ON child.parent_id = branch.id
	)
	UPDATE pages SET category_id = '' WHERE category_id IN (SELECT id FROM branch)`
	deleteCategory = `DELETE FROM categories WHERE id = ?`
)

type CategoryRepo struct {
	store *Store
}

func NewCategoryRepo(store *Store) *CategoryRepo {
	return &CategoryRepo{store: store}
}

func categoryNotFound(id string) *errors.Error {
	return errors.New(errors.NotFound, "category not found").WithDetail("categoryId", id)
}

func categoryConflict(name string) *errors.Error {
	return errors.New(errors.Conflict, "a category named "+name+" already sits under the same parent").WithDetail("name", name)
}

func categoryParentElsewhere(c category.Category) *errors.Error {
	return errors.New(errors.Invalid, "the parent category is not a category of this site").
		WithDetail("field", "parentId").WithDetail("parentId", c.ParentID).WithDetail("siteId", c.SiteID)
}

func (r *CategoryRepo) Insert(ctx context.Context, c category.Category) error {
	parentID := nullText(c.ParentID)
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), insertCategory, []any{
		c.ID, c.SiteID, c.Name, category.Key(c.Name), parentID, formatTime(c.CreatedAt), formatTime(c.UpdatedAt),
		parentID, c.ParentID, c.SiteID,
	}, categoryConflict(c.Name), "insert the category")
	return requireAffected(affected, err, categoryParentElsewhere(c))
}

func (r *CategoryRepo) ListBySite(ctx context.Context, siteID string) ([]category.Category, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectCategoriesBySite, []any{siteID}, scanCategory, "list the site categories")
}

func (r *CategoryRepo) Delete(ctx context.Context, id string) error {
	return r.store.Do(ctx, func(ctx context.Context) error {
		if _, err := execWrite(ctx, r.store.writeFrom(ctx), unfileCategoryPages, []any{id}, nil, "take the pages out of the category"); err != nil {
			return err
		}
		affected, err := execWrite(ctx, r.store.writeFrom(ctx), deleteCategory, []any{id}, nil, "delete the category")
		return requireAffected(affected, err, categoryNotFound(id))
	})
}

func scanCategory(rows *sql.Rows) (category.Category, error) {
	var (
		c                    category.Category
		parentID             sql.NullString
		createdAt, updatedAt string
	)
	if err := rows.Scan(&c.ID, &c.SiteID, &c.Name, &c.Key, &parentID, &createdAt, &updatedAt); err != nil {
		return category.Category{}, err
	}
	c.ParentID = parentID.String

	var err error
	if c.CreatedAt, err = parseTime(createdAt); err != nil {
		return category.Category{}, err
	}
	if c.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return category.Category{}, err
	}
	return c, nil
}
