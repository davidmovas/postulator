package sitemap

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/infra/database"
	"github.com/davidmovas/postulator/pkg/dbx"
	"github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"
)

var _ NodeRepository = (*nodeRepository)(nil)

type nodeRepository struct {
	db     *database.DB
	logger *logger.Logger
}

func NewNodeRepository(db *database.DB, logger *logger.Logger) NodeRepository {
	return &nodeRepository{
		db: db,
		logger: logger.
			WithScope("repository").
			WithScope("sitemap_node"),
	}
}

func (r *nodeRepository) Create(ctx context.Context, node *entities.SitemapNode) error {
	query, args := dbx.ST.
		Insert("sitemap_nodes").
		Columns(
			"sitemap_id", "parent_id", "title", "slug", "description", "is_root",
			"depth", "position", "path", "content_type", "article_id",
			"wp_page_id", "wp_url", "source", "is_synced", "last_synced_at",
			"wp_title", "wp_slug",
			"design_status", "generation_status", "publish_status",
			"is_modified_locally", "last_error",
			"position_x", "position_y",
		).
		Values(
			node.SitemapID, node.ParentID, node.Title, node.Slug, node.Description, node.IsRoot,
			node.Depth, node.Position, node.Path, node.ContentType, node.ArticleID,
			node.WPPageID, node.WPURL, node.Source, node.IsSynced, node.LastSyncedAt,
			node.WPTitle, node.WPSlug,
			node.DesignStatus, node.GenerationStatus, node.PublishStatus,
			node.IsModifiedLocally, node.LastError,
			node.PositionX, node.PositionY,
		).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsUniqueViolation(err):
		return errors.AlreadyExists(fmt.Sprintf("Node with slug '%s' already exists in this location", node.Slug))
	case err != nil:
		return errors.Database(err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return errors.Database(err)
	}

	node.ID = id
	return nil
}

func (r *nodeRepository) CreateBatch(ctx context.Context, nodes []*entities.SitemapNode) error {
	if len(nodes) == 0 {
		return nil
	}

	builder := dbx.ST.
		Insert("sitemap_nodes").
		Columns(
			"sitemap_id", "parent_id", "title", "slug", "description", "is_root",
			"depth", "position", "path", "content_type", "article_id",
			"wp_page_id", "wp_url", "source", "is_synced", "last_synced_at",
			"wp_title", "wp_slug",
			"design_status", "generation_status", "publish_status",
			"is_modified_locally", "last_error",
			"position_x", "position_y",
		)

	for _, node := range nodes {
		builder = builder.Values(
			node.SitemapID, node.ParentID, node.Title, node.Slug, node.Description, node.IsRoot,
			node.Depth, node.Position, node.Path, node.ContentType, node.ArticleID,
			node.WPPageID, node.WPURL, node.Source, node.IsSynced, node.LastSyncedAt,
			node.WPTitle, node.WPSlug,
			node.DesignStatus, node.GenerationStatus, node.PublishStatus,
			node.IsModifiedLocally, node.LastError,
			node.PositionX, node.PositionY,
		)
	}

	query, args := builder.MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	return nil
}

func (r *nodeRepository) GetByID(ctx context.Context, id int64) (*entities.SitemapNode, error) {
	query, args := dbx.ST.
		Select(nodeColumns()...).
		From("sitemap_nodes").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	node, err := r.scanNode(r.db.QueryRowContext(ctx, query, args...))
	if err != nil {
		if dbx.IsNoRows(err) {
			return nil, errors.NotFound("sitemap node", id)
		}
		return nil, errors.Database(err)
	}

	return node, nil
}

func (r *nodeRepository) GetBySitemapID(ctx context.Context, sitemapID int64) ([]*entities.SitemapNode, error) {
	query, args := dbx.ST.
		Select(nodeColumns()...).
		From("sitemap_nodes").
		Where(squirrel.Eq{"sitemap_id": sitemapID}).
		OrderBy("depth ASC", "position ASC").
		MustSql()

	return r.queryNodes(ctx, query, args)
}

func (r *nodeRepository) GetByParentID(ctx context.Context, sitemapID int64, parentID *int64) ([]*entities.SitemapNode, error) {
	builder := dbx.ST.
		Select(nodeColumns()...).
		From("sitemap_nodes").
		Where(squirrel.Eq{"sitemap_id": sitemapID})

	if parentID == nil {
		builder = builder.Where("parent_id IS NULL")
	} else {
		builder = builder.Where(squirrel.Eq{"parent_id": *parentID})
	}

	query, args := builder.OrderBy("position ASC").MustSql()

	return r.queryNodes(ctx, query, args)
}

func (r *nodeRepository) GetBySlugAndParent(ctx context.Context, sitemapID int64, slug string, parentID *int64) (*entities.SitemapNode, error) {
	builder := dbx.ST.
		Select(nodeColumns()...).
		From("sitemap_nodes").
		Where(squirrel.Eq{"sitemap_id": sitemapID}).
		Where(squirrel.Eq{"slug": slug})

	if parentID == nil {
		builder = builder.Where("parent_id IS NULL")
	} else {
		builder = builder.Where(squirrel.Eq{"parent_id": *parentID})
	}

	query, args := builder.Limit(1).MustSql()

	node, err := r.scanNode(r.db.QueryRowContext(ctx, query, args...))
	if err != nil {
		if dbx.IsNoRows(err) {
			return nil, errors.NotFound("sitemap node", fmt.Sprintf("slug=%s", slug))
		}
		return nil, errors.Database(err)
	}

	return node, nil
}

func (r *nodeRepository) GetRootNodes(ctx context.Context, sitemapID int64) ([]*entities.SitemapNode, error) {
	return r.GetByParentID(ctx, sitemapID, nil)
}

func (r *nodeRepository) GetByWPID(ctx context.Context, sitemapID int64, wpID int, contentType entities.NodeContentType) (*entities.SitemapNode, error) {
	query, args := dbx.ST.
		Select(nodeColumns()...).
		From("sitemap_nodes").
		Where(squirrel.Eq{"sitemap_id": sitemapID}).
		Where(squirrel.Eq{"wp_page_id": wpID}).
		Where(squirrel.Eq{"content_type": contentType}).
		Limit(1).
		MustSql()

	node, err := r.scanNode(r.db.QueryRowContext(ctx, query, args...))
	if err != nil {
		if dbx.IsNoRows(err) {
			return nil, errors.NotFound("sitemap node", fmt.Sprintf("wp_id=%d", wpID))
		}
		return nil, errors.Database(err)
	}

	return node, nil
}

func (r *nodeRepository) Update(ctx context.Context, node *entities.SitemapNode) error {
	query, args := dbx.ST.
		Update("sitemap_nodes").
		Set("parent_id", node.ParentID).
		Set("title", node.Title).
		Set("slug", node.Slug).
		Set("description", node.Description).
		Set("depth", node.Depth).
		Set("position", node.Position).
		Set("path", node.Path).
		Set("content_type", node.ContentType).
		Set("article_id", node.ArticleID).
		Set("wp_page_id", node.WPPageID).
		Set("wp_url", node.WPURL).
		Set("source", node.Source).
		Set("is_synced", node.IsSynced).
		Set("last_synced_at", node.LastSyncedAt).
		Set("wp_title", node.WPTitle).
		Set("wp_slug", node.WPSlug).
		Set("design_status", node.DesignStatus).
		Set("generation_status", node.GenerationStatus).
		Set("publish_status", node.PublishStatus).
		Set("is_modified_locally", node.IsModifiedLocally).
		Set("last_error", node.LastError).
		Set("position_x", node.PositionX).
		Set("position_y", node.PositionY).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": node.ID}).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsUniqueViolation(err):
		return errors.AlreadyExists(fmt.Sprintf("Node with slug '%s' already exists in this location", node.Slug))
	case err != nil:
		return errors.Database(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Database(err)
	}

	if rowsAffected == 0 {
		return errors.NotFound("sitemap node", node.ID)
	}

	return nil
}

func (r *nodeRepository) Delete(ctx context.Context, id int64) error {
	query, args := dbx.ST.
		Delete("sitemap_nodes").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Database(err)
	}

	if rowsAffected == 0 {
		return errors.NotFound("sitemap node", id)
	}

	return nil
}

func (r *nodeRepository) DeleteBySitemapID(ctx context.Context, sitemapID int64) error {
	query, args := dbx.ST.
		Delete("sitemap_nodes").
		Where(squirrel.Eq{"sitemap_id": sitemapID}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	return nil
}

func (r *nodeRepository) UpdateParent(ctx context.Context, nodeID int64, newParentID *int64) error {
	query, args := dbx.ST.
		Update("sitemap_nodes").
		Set("parent_id", newParentID).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": nodeID}).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Database(err)
	}

	if rowsAffected == 0 {
		return errors.NotFound("sitemap node", nodeID)
	}

	return nil
}

func (r *nodeRepository) UpdatePosition(ctx context.Context, nodeID int64, position int) error {
	query, args := dbx.ST.
		Update("sitemap_nodes").
		Set("position", position).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": nodeID}).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Database(err)
	}

	if rowsAffected == 0 {
		return errors.NotFound("sitemap node", nodeID)
	}

	return nil
}

func (r *nodeRepository) UpdatePositions(ctx context.Context, nodeID int64, positionX, positionY float64) error {
	query, args := dbx.ST.
		Update("sitemap_nodes").
		Set("position_x", positionX).
		Set("position_y", positionY).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": nodeID}).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Database(err)
	}

	if rowsAffected == 0 {
		return errors.NotFound("sitemap node", nodeID)
	}

	return nil
}

func (r *nodeRepository) GetDescendants(ctx context.Context, nodeID int64) ([]*entities.SitemapNode, error) {
	// Get the node first to get its path
	node, err := r.GetByID(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	// Find all nodes whose path starts with this node's path
	pathPrefix := node.Path + "/"
	if node.Path == "" {
		pathPrefix = "/" + node.Slug + "/"
	}

	query, args := dbx.ST.
		Select(nodeColumns()...).
		From("sitemap_nodes").
		Where(squirrel.Eq{"sitemap_id": node.SitemapID}).
		Where(squirrel.Like{"path": pathPrefix + "%"}).
		OrderBy("depth ASC", "position ASC").
		MustSql()

	return r.queryNodes(ctx, query, args)
}

func (r *nodeRepository) GetAncestors(ctx context.Context, nodeID int64) ([]*entities.SitemapNode, error) {
	node, err := r.GetByID(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	if node.ParentID == nil {
		return []*entities.SitemapNode{}, nil
	}

	// Get path parts and find ancestors
	pathParts := strings.Split(strings.Trim(node.Path, "/"), "/")
	if len(pathParts) <= 1 {
		return []*entities.SitemapNode{}, nil
	}

	// Build paths for ancestors
	var ancestorPaths []string
	currentPath := ""
	for i := 0; i < len(pathParts)-1; i++ {
		currentPath += "/" + pathParts[i]
		ancestorPaths = append(ancestorPaths, currentPath)
	}

	if len(ancestorPaths) == 0 {
		return []*entities.SitemapNode{}, nil
	}

	query, args := dbx.ST.
		Select(nodeColumns()...).
		From("sitemap_nodes").
		Where(squirrel.Eq{"sitemap_id": node.SitemapID}).
		Where(squirrel.Eq{"path": ancestorPaths}).
		OrderBy("depth ASC").
		MustSql()

	return r.queryNodes(ctx, query, args)
}

func (r *nodeRepository) UpdateDesignStatus(ctx context.Context, nodeID int64, status entities.NodeDesignStatus) error {
	query, args := dbx.ST.
		Update("sitemap_nodes").
		Set("design_status", status).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": nodeID}).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Database(err)
	}

	if rowsAffected == 0 {
		return errors.NotFound("sitemap node", nodeID)
	}

	return nil
}

func (r *nodeRepository) UpdateGenerationStatus(ctx context.Context, nodeID int64, status entities.NodeGenerationStatus, lastError *string) error {
	query, args := dbx.ST.
		Update("sitemap_nodes").
		Set("generation_status", status).
		Set("last_error", lastError).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": nodeID}).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Database(err)
	}

	if rowsAffected == 0 {
		return errors.NotFound("sitemap node", nodeID)
	}

	return nil
}

func (r *nodeRepository) UpdatePublishStatus(ctx context.Context, nodeID int64, status entities.NodePublishStatus, lastError *string) error {
	query, args := dbx.ST.
		Update("sitemap_nodes").
		Set("publish_status", status).
		Set("last_error", lastError).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": nodeID}).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Database(err)
	}

	if rowsAffected == 0 {
		return errors.NotFound("sitemap node", nodeID)
	}

	return nil
}

func (r *nodeRepository) UpdateAllStatuses(ctx context.Context, nodeID int64, design entities.NodeDesignStatus, generation entities.NodeGenerationStatus, publish entities.NodePublishStatus) error {
	query, args := dbx.ST.
		Update("sitemap_nodes").
		Set("design_status", design).
		Set("generation_status", generation).
		Set("publish_status", publish).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": nodeID}).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Database(err)
	}

	if rowsAffected == 0 {
		return errors.NotFound("sitemap node", nodeID)
	}

	return nil
}

func (r *nodeRepository) GetByGenerationStatus(ctx context.Context, sitemapID int64, status entities.NodeGenerationStatus) ([]*entities.SitemapNode, error) {
	query, args := dbx.ST.
		Select(nodeColumns()...).
		From("sitemap_nodes").
		Where(squirrel.Eq{"sitemap_id": sitemapID}).
		Where(squirrel.Eq{"generation_status": status}).
		OrderBy("depth ASC", "position ASC").
		MustSql()

	return r.queryNodes(ctx, query, args)
}

func (r *nodeRepository) UpdateContentLink(ctx context.Context, nodeID int64, contentType entities.NodeContentType, articleID *int64, wpPageID *int, wpURL *string) error {
	query, args := dbx.ST.
		Update("sitemap_nodes").
		Set("content_type", contentType).
		Set("article_id", articleID).
		Set("wp_page_id", wpPageID).
		Set("wp_url", wpURL).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": nodeID}).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Database(err)
	}

	if rowsAffected == 0 {
		return errors.NotFound("sitemap node", nodeID)
	}

	return nil
}

func (r *nodeRepository) UpdateSyncStatus(ctx context.Context, nodeID int64, isSynced bool) error {
	var lastSyncedAt *time.Time
	if isSynced {
		now := time.Now()
		lastSyncedAt = &now
	}

	query, args := dbx.ST.
		Update("sitemap_nodes").
		Set("is_synced", isSynced).
		Set("last_synced_at", lastSyncedAt).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": nodeID}).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Database(err)
	}

	if rowsAffected == 0 {
		return errors.NotFound("sitemap node", nodeID)
	}

	return nil
}

// Helper functions

func nodeColumns() []string {
	return []string{
		"id", "sitemap_id", "parent_id", "title", "slug", "description", "is_root",
		"depth", "position", "path", "content_type", "article_id",
		"wp_page_id", "wp_url", "source", "is_synced", "last_synced_at",
		"wp_title", "wp_slug",
		"design_status", "generation_status", "publish_status",
		"is_modified_locally", "last_error",
		"position_x", "position_y", "created_at", "updated_at",
	}
}

// nullableNodeFields holds nullable database fields for scanning
type nullableNodeFields struct {
	parentID     sql.NullInt64
	description  sql.NullString
	articleID    sql.NullInt64
	wpPageID     sql.NullInt64
	wpURL        sql.NullString
	lastSyncedAt sql.NullTime
	wpTitle      sql.NullString
	wpSlug       sql.NullString
	lastError    sql.NullString
	positionX    sql.NullFloat64
	positionY    sql.NullFloat64
}

// applyNullableFields applies nullable field values to the node
func applyNullableFields(node *entities.SitemapNode, nf *nullableNodeFields) {
	if nf.parentID.Valid {
		node.ParentID = &nf.parentID.Int64
	}
	if nf.description.Valid {
		node.Description = &nf.description.String
	}
	if nf.articleID.Valid {
		node.ArticleID = &nf.articleID.Int64
	}
	if nf.wpPageID.Valid {
		wpPageIDInt := int(nf.wpPageID.Int64)
		node.WPPageID = &wpPageIDInt
	}
	if nf.wpURL.Valid {
		node.WPURL = &nf.wpURL.String
	}
	if nf.lastSyncedAt.Valid {
		node.LastSyncedAt = &nf.lastSyncedAt.Time
	}
	if nf.wpTitle.Valid {
		node.WPTitle = &nf.wpTitle.String
	}
	if nf.wpSlug.Valid {
		node.WPSlug = &nf.wpSlug.String
	}
	if nf.lastError.Valid {
		node.LastError = &nf.lastError.String
	}
	if nf.positionX.Valid {
		node.PositionX = &nf.positionX.Float64
	}
	if nf.positionY.Valid {
		node.PositionY = &nf.positionY.Float64
	}
	node.Path = normalizePath(node.Path)
}

func (r *nodeRepository) scanNode(row *sql.Row) (*entities.SitemapNode, error) {
	var node entities.SitemapNode
	var nf nullableNodeFields

	err := row.Scan(
		&node.ID,
		&node.SitemapID,
		&nf.parentID,
		&node.Title,
		&node.Slug,
		&nf.description,
		&node.IsRoot,
		&node.Depth,
		&node.Position,
		&node.Path,
		&node.ContentType,
		&nf.articleID,
		&nf.wpPageID,
		&nf.wpURL,
		&node.Source,
		&node.IsSynced,
		&nf.lastSyncedAt,
		&nf.wpTitle,
		&nf.wpSlug,
		&node.DesignStatus,
		&node.GenerationStatus,
		&node.PublishStatus,
		&node.IsModifiedLocally,
		&nf.lastError,
		&nf.positionX,
		&nf.positionY,
		&node.CreatedAt,
		&node.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	applyNullableFields(&node, &nf)
	return &node, nil
}

// normalizePath removes duplicate slashes from path
func normalizePath(path string) string {
	if path == "" || path == "/" {
		return path
	}
	// Replace multiple slashes with single slash
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}
	return path
}

func (r *nodeRepository) queryNodes(ctx context.Context, query string, args []any) ([]*entities.SitemapNode, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Database(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var nodes []*entities.SitemapNode
	for rows.Next() {
		var node entities.SitemapNode
		var nf nullableNodeFields

		err = rows.Scan(
			&node.ID,
			&node.SitemapID,
			&nf.parentID,
			&node.Title,
			&node.Slug,
			&nf.description,
			&node.IsRoot,
			&node.Depth,
			&node.Position,
			&node.Path,
			&node.ContentType,
			&nf.articleID,
			&nf.wpPageID,
			&nf.wpURL,
			&node.Source,
			&node.IsSynced,
			&nf.lastSyncedAt,
			&nf.wpTitle,
			&nf.wpSlug,
			&node.DesignStatus,
			&node.GenerationStatus,
			&node.PublishStatus,
			&node.IsModifiedLocally,
			&nf.lastError,
			&nf.positionX,
			&nf.positionY,
			&node.CreatedAt,
			&node.UpdatedAt,
		)
		if err != nil {
			return nil, errors.Database(err)
		}

		applyNullableFields(&node, &nf)
		nodes = append(nodes, &node)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.Database(err)
	}

	return nodes, nil
}
