package sitemap

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/infra/database"
	"github.com/davidmovas/postulator/pkg/dbx"
	"github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"
)

var _ KeywordRepository = (*keywordRepository)(nil)

type keywordRepository struct {
	db     *database.DB
	logger *logger.Logger
}

func NewKeywordRepository(db *database.DB, logger *logger.Logger) KeywordRepository {
	return &keywordRepository{
		db: db,
		logger: logger.
			WithScope("repository").
			WithScope("sitemap_keyword"),
	}
}

func (r *keywordRepository) Create(ctx context.Context, keyword *entities.SitemapNodeKeyword) error {
	query, args := dbx.ST.
		Insert("sitemap_node_keywords").
		Columns("node_id", "keyword", "position").
		Values(keyword.NodeID, keyword.Keyword, keyword.Position).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsUniqueViolation(err):
		return errors.AlreadyExists("Keyword already exists for this node")
	case err != nil:
		return errors.Database(err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return errors.Database(err)
	}

	keyword.ID = id
	keyword.CreatedAt = time.Now()
	return nil
}

func (r *keywordRepository) CreateBatch(ctx context.Context, nodeID int64, keywords []string) error {
	if len(keywords) == 0 {
		return nil
	}

	builder := dbx.ST.
		Insert("sitemap_node_keywords").
		Columns("node_id", "keyword", "position")

	for i, kw := range keywords {
		builder = builder.Values(nodeID, kw, i)
	}

	query, args := builder.MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	return nil
}

func (r *keywordRepository) GetByNodeID(ctx context.Context, nodeID int64) ([]*entities.SitemapNodeKeyword, error) {
	query, args := dbx.ST.
		Select("id", "node_id", "keyword", "position", "created_at").
		From("sitemap_node_keywords").
		Where(squirrel.Eq{"node_id": nodeID}).
		OrderBy("position ASC").
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Database(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var keywords []*entities.SitemapNodeKeyword
	for rows.Next() {
		var kw entities.SitemapNodeKeyword
		err = rows.Scan(
			&kw.ID,
			&kw.NodeID,
			&kw.Keyword,
			&kw.Position,
			&kw.CreatedAt,
		)
		if err != nil {
			return nil, errors.Database(err)
		}
		keywords = append(keywords, &kw)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.Database(err)
	}

	return keywords, nil
}

func (r *keywordRepository) GetKeywordsByNodeIDs(ctx context.Context, nodeIDs []int64) (map[int64][]string, error) {
	if len(nodeIDs) == 0 {
		return make(map[int64][]string), nil
	}

	query, args := dbx.ST.
		Select("node_id", "keyword").
		From("sitemap_node_keywords").
		Where(squirrel.Eq{"node_id": nodeIDs}).
		OrderBy("node_id ASC", "position ASC").
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Database(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	result := make(map[int64][]string)
	for rows.Next() {
		var nodeID int64
		var keyword string
		if err = rows.Scan(&nodeID, &keyword); err != nil {
			return nil, errors.Database(err)
		}
		result[nodeID] = append(result[nodeID], keyword)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.Database(err)
	}

	return result, nil
}

func (r *keywordRepository) Delete(ctx context.Context, id int64) error {
	query, args := dbx.ST.
		Delete("sitemap_node_keywords").
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
		return errors.NotFound("keyword", id)
	}

	return nil
}

func (r *keywordRepository) DeleteByNodeID(ctx context.Context, nodeID int64) error {
	query, args := dbx.ST.
		Delete("sitemap_node_keywords").
		Where(squirrel.Eq{"node_id": nodeID}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	return nil
}

func (r *keywordRepository) ReplaceKeywords(ctx context.Context, nodeID int64, keywords []string) error {
	// Delete existing keywords
	if err := r.DeleteByNodeID(ctx, nodeID); err != nil {
		return err
	}

	// Insert new keywords
	if len(keywords) > 0 {
		return r.CreateBatch(ctx, nodeID, keywords)
	}

	return nil
}

func (r *keywordRepository) DistributeKeywords(ctx context.Context, sitemapID int64, keywords []string, strategy KeywordDistributionStrategy) error {
	if len(keywords) == 0 {
		return nil
	}

	// Get all nodes for the sitemap
	query, args := dbx.ST.
		Select("id").
		From("sitemap_nodes").
		Where(squirrel.Eq{"sitemap_id": sitemapID}).
		OrderBy("depth ASC", "position ASC").
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var nodeIDs []int64
	for rows.Next() {
		var nodeID int64
		if err = rows.Scan(&nodeID); err != nil {
			return errors.Database(err)
		}
		nodeIDs = append(nodeIDs, nodeID)
	}

	if err = rows.Err(); err != nil {
		return errors.Database(err)
	}

	if len(nodeIDs) == 0 {
		return nil
	}

	// Distribute based on strategy
	switch strategy {
	case KeywordDistributionEven:
		return r.distributeEven(ctx, nodeIDs, keywords)
	case KeywordDistributionByPath:
		// TODO: Implement path-based distribution using AI or text similarity
		return r.distributeEven(ctx, nodeIDs, keywords)
	default:
		return r.distributeEven(ctx, nodeIDs, keywords)
	}
}

func (r *keywordRepository) distributeEven(ctx context.Context, nodeIDs []int64, keywords []string) error {
	if len(nodeIDs) == 0 || len(keywords) == 0 {
		return nil
	}

	// Calculate keywords per node
	keywordsPerNode := len(keywords) / len(nodeIDs)
	if keywordsPerNode == 0 {
		keywordsPerNode = 1
	}

	kwIndex := 0
	for _, nodeID := range nodeIDs {
		if kwIndex >= len(keywords) {
			break
		}

		endIndex := kwIndex + keywordsPerNode
		if endIndex > len(keywords) {
			endIndex = len(keywords)
		}

		nodeKeywords := keywords[kwIndex:endIndex]
		if err := r.CreateBatch(ctx, nodeID, nodeKeywords); err != nil {
			r.logger.ErrorWithErr(err, fmt.Sprintf("Failed to create keywords batch for node %d", nodeID))
			// Continue with other nodes
		}

		kwIndex = endIndex
	}

	// Distribute remaining keywords to last nodes
	if kwIndex < len(keywords) {
		lastNodeID := nodeIDs[len(nodeIDs)-1]
		remaining := keywords[kwIndex:]
		if err := r.CreateBatch(ctx, lastNodeID, remaining); err != nil {
			r.logger.ErrorWithErr(err, fmt.Sprintf("Failed to create remaining keywords for node %d", lastNodeID))
		}
	}

	return nil
}
