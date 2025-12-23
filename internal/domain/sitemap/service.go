package sitemap

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"
)

var _ Service = (*service)(nil)

type service struct {
	repo        Repository
	nodeRepo    NodeRepository
	keywordRepo KeywordRepository
	logger      *logger.Logger
}

func NewService(
	repo Repository,
	nodeRepo NodeRepository,
	keywordRepo KeywordRepository,
	logger *logger.Logger,
) Service {
	return &service{
		repo:        repo,
		nodeRepo:    nodeRepo,
		keywordRepo: keywordRepo,
		logger: logger.
			WithScope("service").
			WithScope("sitemap"),
	}
}

func (s *service) CreateSitemap(ctx context.Context, sitemap *entities.Sitemap) error {
	if err := s.validateSitemap(sitemap); err != nil {
		return err
	}

	if sitemap.Source == "" {
		sitemap.Source = entities.SitemapSourceManual
	}
	if sitemap.Status == "" {
		sitemap.Status = entities.SitemapStatusDraft
	}

	now := time.Now()
	sitemap.CreatedAt = now
	sitemap.UpdatedAt = now

	if err := s.repo.Create(ctx, sitemap); err != nil {
		s.logger.ErrorWithErr(err, "Failed to create sitemap")
		return err
	}

	s.logger.Infof("Sitemap created successfully: %d", sitemap.ID)
	return nil
}

func (s *service) CreateSitemapWithRoot(ctx context.Context, sitemap *entities.Sitemap, siteURL string) error {
	// Create the sitemap first
	if err := s.CreateSitemap(ctx, sitemap); err != nil {
		return err
	}

	// Create root node with site URL as title
	rootNode := &entities.SitemapNode{
		SitemapID:     sitemap.ID,
		Title:         siteURL,
		Slug:          "",
		IsRoot:           true,
		Depth:            0,
		Position:         0,
		Path:             "/",
		ContentType:      entities.NodeContentTypeNone,
		Source:           entities.NodeSourceManual,
		DesignStatus:     entities.DesignStatusApproved,
		GenerationStatus: entities.GenStatusNone,
		PublishStatus:    entities.PubStatusNone,
	}

	now := time.Now()
	rootNode.CreatedAt = now
	rootNode.UpdatedAt = now

	if err := s.nodeRepo.Create(ctx, rootNode); err != nil {
		s.logger.ErrorWithErr(err, "Failed to create root node")
		// Try to delete the sitemap if root node creation fails
		_ = s.repo.Delete(ctx, sitemap.ID)
		return err
	}

	s.logger.Infof("Sitemap with root node created successfully: %d", sitemap.ID)
	return nil
}

func (s *service) GetSitemap(ctx context.Context, id int64) (*entities.Sitemap, error) {
	sitemap, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get sitemap")
		return nil, err
	}
	return sitemap, nil
}

func (s *service) GetSitemapWithNodes(ctx context.Context, id int64) (*entities.Sitemap, []*entities.SitemapNode, error) {
	sitemap, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get sitemap")
		return nil, nil, err
	}

	nodes, err := s.nodeRepo.GetBySitemapID(ctx, id)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get sitemap nodes")
		return nil, nil, err
	}

	// Load keywords for all nodes
	if len(nodes) > 0 {
		nodeIDs := make([]int64, len(nodes))
		for i, node := range nodes {
			nodeIDs[i] = node.ID
		}

		keywordsMap, err := s.keywordRepo.GetKeywordsByNodeIDs(ctx, nodeIDs)
		if err != nil {
			s.logger.ErrorWithErr(err, "Failed to get keywords for nodes")
			// Continue without keywords
		} else {
			for _, node := range nodes {
				if kws, ok := keywordsMap[node.ID]; ok {
					node.Keywords = kws
				}
			}
		}
	}

	return sitemap, nodes, nil
}

func (s *service) ListSitemaps(ctx context.Context, siteID int64) ([]*entities.Sitemap, error) {
	sitemaps, err := s.repo.GetBySiteID(ctx, siteID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to list sitemaps")
		return nil, err
	}
	return sitemaps, nil
}

func (s *service) UpdateSitemap(ctx context.Context, sitemap *entities.Sitemap) error {
	existing, err := s.repo.GetByID(ctx, sitemap.ID)
	if err != nil {
		return err
	}

	// Copy fields from existing that shouldn't be changed
	sitemap.SiteID = existing.SiteID
	sitemap.Source = existing.Source
	sitemap.CreatedAt = existing.CreatedAt
	sitemap.UpdatedAt = time.Now()

	if err := s.validateSitemap(sitemap); err != nil {
		return err
	}

	if err = s.repo.Update(ctx, sitemap); err != nil {
		s.logger.ErrorWithErr(err, "Failed to update sitemap")
		return err
	}

	s.logger.Infof("Sitemap updated successfully: %d", sitemap.ID)
	return nil
}

func (s *service) DeleteSitemap(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.ErrorWithErr(err, "Failed to delete sitemap")
		return err
	}

	s.logger.Infof("Sitemap deleted successfully: %d", id)
	return nil
}

func (s *service) DuplicateSitemap(ctx context.Context, id int64, newName string) (*entities.Sitemap, error) {
	// Get original sitemap with nodes
	original, nodes, err := s.GetSitemapWithNodes(ctx, id)
	if err != nil {
		return nil, err
	}

	// Create new sitemap
	newSitemap := &entities.Sitemap{
		SiteID:      original.SiteID,
		Name:        newName,
		Description: original.Description,
		Source:      original.Source,
		Status:      entities.SitemapStatusDraft,
	}

	if err = s.CreateSitemap(ctx, newSitemap); err != nil {
		return nil, err
	}

	// Create mapping of old node IDs to new node IDs
	idMapping := make(map[int64]int64)

	// First pass: create all root nodes (nodes without parent)
	for _, node := range nodes {
		if node.ParentID == nil {
			newNode := s.copyNode(node, newSitemap.ID, nil)
			if err = s.nodeRepo.Create(ctx, newNode); err != nil {
				s.logger.ErrorWithErr(err, "Failed to create root node during duplication")
				continue
			}
			idMapping[node.ID] = newNode.ID

			// Copy keywords
			if len(node.Keywords) > 0 {
				if err = s.keywordRepo.CreateBatch(ctx, newNode.ID, node.Keywords); err != nil {
					s.logger.ErrorWithErr(err, "Failed to copy keywords during duplication")
				}
			}
		}
	}

	// Subsequent passes: create child nodes level by level
	for depth := 1; depth <= s.maxDepth(nodes); depth++ {
		for _, node := range nodes {
			if node.Depth == depth && node.ParentID != nil {
				newParentID, ok := idMapping[*node.ParentID]
				if !ok {
					continue
				}

				newNode := s.copyNode(node, newSitemap.ID, &newParentID)
				if err = s.nodeRepo.Create(ctx, newNode); err != nil {
					s.logger.ErrorWithErr(err, "Failed to create child node during duplication")
					continue
				}
				idMapping[node.ID] = newNode.ID

				// Copy keywords
				if len(node.Keywords) > 0 {
					if err = s.keywordRepo.CreateBatch(ctx, newNode.ID, node.Keywords); err != nil {
						s.logger.ErrorWithErr(err, "Failed to copy keywords during duplication")
					}
				}
			}
		}
	}

	s.logger.Infof("Sitemap duplicated successfully: %d -> %d", id, newSitemap.ID)
	return newSitemap, nil
}

func (s *service) SetSitemapStatus(ctx context.Context, id int64, status entities.SitemapStatus) error {
	if err := s.repo.UpdateStatus(ctx, id, status); err != nil {
		s.logger.ErrorWithErr(err, "Failed to update sitemap status")
		return err
	}
	return nil
}

func (s *service) CreateNode(ctx context.Context, node *entities.SitemapNode) error {
	if err := s.validateNode(node); err != nil {
		return err
	}

	// Set defaults
	if node.Source == "" {
		node.Source = entities.NodeSourceManual
	}
	if node.ContentType == "" {
		node.ContentType = entities.NodeContentTypeNone
	}
	if node.DesignStatus == "" {
		node.DesignStatus = entities.DesignStatusDraft
	}
	if node.GenerationStatus == "" {
		node.GenerationStatus = entities.GenStatusNone
	}
	if node.PublishStatus == "" {
		node.PublishStatus = entities.PubStatusNone
	}

	// Calculate depth and path
	if err := s.calculateNodeHierarchy(ctx, node); err != nil {
		return err
	}

	now := time.Now()
	node.CreatedAt = now
	node.UpdatedAt = now

	if err := s.nodeRepo.Create(ctx, node); err != nil {
		// Don't log ALREADY_EXISTS as error - it's an expected conflict, not a failure
		if !errors.IsAlreadyExists(err) {
			s.logger.ErrorWithErr(err, "Failed to create node")
		}
		return err
	}

	// Create keywords if provided
	if len(node.Keywords) > 0 {
		if err := s.keywordRepo.CreateBatch(ctx, node.ID, node.Keywords); err != nil {
			s.logger.ErrorWithErr(err, "Failed to create keywords for node")
			// Don't fail the whole operation
		}
	}

	// Update sitemap's updated_at
	_ = s.repo.TouchUpdatedAt(ctx, node.SitemapID)

	return nil
}

func (s *service) CreateNodes(ctx context.Context, nodes []*entities.SitemapNode) error {
	for _, node := range nodes {
		if err := s.CreateNode(ctx, node); err != nil {
			return err
		}
	}
	return nil
}

func (s *service) GetNode(ctx context.Context, id int64) (*entities.SitemapNode, error) {
	node, err := s.nodeRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get node")
		return nil, err
	}
	return node, nil
}

func (s *service) GetNodeWithKeywords(ctx context.Context, id int64) (*entities.SitemapNode, error) {
	node, err := s.nodeRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get node")
		return nil, err
	}

	keywords, err := s.keywordRepo.GetByNodeID(ctx, id)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get node keywords")
		// Continue without keywords
	} else {
		for _, kw := range keywords {
			node.Keywords = append(node.Keywords, kw.Keyword)
		}
	}

	return node, nil
}

func (s *service) GetNodes(ctx context.Context, sitemapID int64) ([]*entities.SitemapNode, error) {
	nodes, err := s.nodeRepo.GetBySitemapID(ctx, sitemapID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get nodes")
		return nil, err
	}
	return nodes, nil
}

func (s *service) FindNodeBySlugAndParent(ctx context.Context, sitemapID int64, slug string, parentID *int64) (*entities.SitemapNode, error) {
	node, err := s.nodeRepo.GetBySlugAndParent(ctx, sitemapID, slug, parentID)
	if err != nil {
		return nil, err
	}
	return node, nil
}

func (s *service) FindNodeByWPID(ctx context.Context, sitemapID int64, wpID int, contentType entities.NodeContentType) (*entities.SitemapNode, error) {
	node, err := s.nodeRepo.GetByWPID(ctx, sitemapID, wpID, contentType)
	if err != nil {
		return nil, err
	}
	return node, nil
}

func (s *service) GetNodesTree(ctx context.Context, sitemapID int64) ([]*entities.SitemapNode, error) {
	nodes, err := s.nodeRepo.GetBySitemapID(ctx, sitemapID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get nodes for tree")
		return nil, err
	}

	// Load keywords
	if len(nodes) > 0 {
		nodeIDs := make([]int64, len(nodes))
		for i, node := range nodes {
			nodeIDs[i] = node.ID
		}

		var keywordsMap map[int64][]string
		keywordsMap, err = s.keywordRepo.GetKeywordsByNodeIDs(ctx, nodeIDs)
		if err != nil {
			s.logger.ErrorWithErr(err, "Failed to get keywords for tree")
		} else {
			for _, node := range nodes {
				if kws, ok := keywordsMap[node.ID]; ok {
					node.Keywords = kws
				}
			}
		}
	}

	// Build tree structure
	return s.buildTree(nodes), nil
}

func (s *service) UpdateNode(ctx context.Context, node *entities.SitemapNode) error {
	if err := s.validateNode(node); err != nil {
		return err
	}

	existing, err := s.nodeRepo.GetByID(ctx, node.ID)
	if err != nil {
		return err
	}

	// Keep sitemap_id unchanged
	node.SitemapID = existing.SitemapID
	node.CreatedAt = existing.CreatedAt
	node.UpdatedAt = time.Now()

	// Recalculate path if parent changed or slug changed
	if (node.ParentID != existing.ParentID) || (node.Slug != existing.Slug) {
		if err = s.calculateNodeHierarchy(ctx, node); err != nil {
			return err
		}
	}

	if err = s.nodeRepo.Update(ctx, node); err != nil {
		s.logger.ErrorWithErr(err, "Failed to update node")
		return err
	}

	// Update sitemap's updated_at
	_ = s.repo.TouchUpdatedAt(ctx, node.SitemapID)

	return nil
}

func (s *service) DeleteNode(ctx context.Context, id int64) error {
	// Get node to know sitemap_id before deletion
	node, err := s.nodeRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Delete will cascade to children and keywords due to FK constraints
	if err := s.nodeRepo.Delete(ctx, id); err != nil {
		s.logger.ErrorWithErr(err, "Failed to delete node")
		return err
	}

	// Update sitemap's updated_at
	_ = s.repo.TouchUpdatedAt(ctx, node.SitemapID)

	return nil
}

func (s *service) MoveNode(ctx context.Context, nodeID int64, newParentID *int64, position int) error {
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return err
	}

	// Update parent
	node.ParentID = newParentID
	node.Position = position

	// Recalculate hierarchy
	if err = s.calculateNodeHierarchy(ctx, node); err != nil {
		return err
	}

	if err = s.nodeRepo.Update(ctx, node); err != nil {
		s.logger.ErrorWithErr(err, "Failed to move node")
		return err
	}

	// Update descendants paths
	descendants, err := s.nodeRepo.GetDescendants(ctx, nodeID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get descendants for path update")
		return nil // Node moved successfully, just couldn't update descendants
	}

	for _, desc := range descendants {
		if err = s.calculateNodeHierarchy(ctx, desc); err != nil {
			continue
		}
		if err = s.nodeRepo.Update(ctx, desc); err != nil {
			s.logger.ErrorWithErr(err, "Failed to update descendant path")
		}
	}

	// Update sitemap's updated_at
	_ = s.repo.TouchUpdatedAt(ctx, node.SitemapID)

	return nil
}

func (s *service) UpdateNodePositions(ctx context.Context, nodeID int64, positionX, positionY float64) error {
	if err := s.nodeRepo.UpdatePositions(ctx, nodeID, positionX, positionY); err != nil {
		s.logger.ErrorWithErr(err, "Failed to update node positions")
		return err
	}
	return nil
}

func (s *service) SetNodeKeywords(ctx context.Context, nodeID int64, keywords []string) error {
	if err := s.keywordRepo.ReplaceKeywords(ctx, nodeID, keywords); err != nil {
		s.logger.ErrorWithErr(err, "Failed to set node keywords")
		return err
	}

	// Update sitemap's updated_at
	if node, err := s.nodeRepo.GetByID(ctx, nodeID); err == nil {
		_ = s.repo.TouchUpdatedAt(ctx, node.SitemapID)
	}

	return nil
}

func (s *service) AddNodeKeyword(ctx context.Context, nodeID int64, keyword string) error {
	kw := &entities.SitemapNodeKeyword{
		NodeID:  nodeID,
		Keyword: keyword,
	}
	if err := s.keywordRepo.Create(ctx, kw); err != nil {
		s.logger.ErrorWithErr(err, "Failed to add node keyword")
		return err
	}

	// Update sitemap's updated_at
	if node, err := s.nodeRepo.GetByID(ctx, nodeID); err == nil {
		_ = s.repo.TouchUpdatedAt(ctx, node.SitemapID)
	}

	return nil
}

func (s *service) RemoveNodeKeyword(ctx context.Context, nodeID int64, keyword string) error {
	// Get all keywords for node
	keywords, err := s.keywordRepo.GetByNodeID(ctx, nodeID)
	if err != nil {
		return err
	}

	// Find and delete the specific keyword
	for _, kw := range keywords {
		if kw.Keyword == keyword {
			if err := s.keywordRepo.Delete(ctx, kw.ID); err != nil {
				return err
			}

			// Update sitemap's updated_at
			if node, nodeErr := s.nodeRepo.GetByID(ctx, nodeID); nodeErr == nil {
				_ = s.repo.TouchUpdatedAt(ctx, node.SitemapID)
			}

			return nil
		}
	}

	return nil
}

func (s *service) DistributeKeywords(ctx context.Context, sitemapID int64, keywords []string, strategy KeywordDistributionStrategy) error {
	if err := s.keywordRepo.DistributeKeywords(ctx, sitemapID, keywords, strategy); err != nil {
		s.logger.ErrorWithErr(err, "Failed to distribute keywords")
		return err
	}

	// Update sitemap's updated_at
	_ = s.repo.TouchUpdatedAt(ctx, sitemapID)

	s.logger.Infof("Keywords distributed successfully for sitemap: %d", sitemapID)
	return nil
}

func (s *service) LinkNodeToArticle(ctx context.Context, nodeID int64, articleID int64) error {
	if err := s.nodeRepo.UpdateContentLink(ctx, nodeID, entities.NodeContentTypePost, &articleID, nil, nil); err != nil {
		s.logger.ErrorWithErr(err, "Failed to link node to article")
		return err
	}

	// Update sitemap's updated_at
	if node, err := s.nodeRepo.GetByID(ctx, nodeID); err == nil {
		_ = s.repo.TouchUpdatedAt(ctx, node.SitemapID)
	}

	s.logger.Infof("Node %d linked to article %d", nodeID, articleID)
	return nil
}

func (s *service) LinkNodeToPage(ctx context.Context, nodeID int64, wpPageID int, wpURL string) error {
	if err := s.nodeRepo.UpdateContentLink(ctx, nodeID, entities.NodeContentTypePage, nil, &wpPageID, &wpURL); err != nil {
		s.logger.ErrorWithErr(err, "Failed to link node to page")
		return err
	}

	// Update sitemap's updated_at
	if node, err := s.nodeRepo.GetByID(ctx, nodeID); err == nil {
		_ = s.repo.TouchUpdatedAt(ctx, node.SitemapID)
	}

	s.logger.Infof("Node %d linked to WP page %d", nodeID, wpPageID)
	return nil
}

func (s *service) UnlinkNodeContent(ctx context.Context, nodeID int64) error {
	if err := s.nodeRepo.UpdateContentLink(ctx, nodeID, entities.NodeContentTypeNone, nil, nil, nil); err != nil {
		s.logger.ErrorWithErr(err, "Failed to unlink node content")
		return err
	}

	// Update sitemap's updated_at
	if node, err := s.nodeRepo.GetByID(ctx, nodeID); err == nil {
		_ = s.repo.TouchUpdatedAt(ctx, node.SitemapID)
	}

	s.logger.Infof("Node %d content unlinked", nodeID)
	return nil
}

func (s *service) UpdateNodeDesignStatus(ctx context.Context, nodeID int64, status entities.NodeDesignStatus) error {
	if err := s.nodeRepo.UpdateDesignStatus(ctx, nodeID, status); err != nil {
		s.logger.ErrorWithErr(err, "Failed to update node design status")
		return err
	}

	if node, err := s.nodeRepo.GetByID(ctx, nodeID); err == nil {
		_ = s.repo.TouchUpdatedAt(ctx, node.SitemapID)
	}

	return nil
}

func (s *service) UpdateNodeGenerationStatus(ctx context.Context, nodeID int64, status entities.NodeGenerationStatus, lastError *string) error {
	if err := s.nodeRepo.UpdateGenerationStatus(ctx, nodeID, status, lastError); err != nil {
		s.logger.ErrorWithErr(err, "Failed to update node generation status")
		return err
	}

	if node, err := s.nodeRepo.GetByID(ctx, nodeID); err == nil {
		_ = s.repo.TouchUpdatedAt(ctx, node.SitemapID)
	}

	return nil
}

func (s *service) UpdateNodePublishStatus(ctx context.Context, nodeID int64, status entities.NodePublishStatus, lastError *string) error {
	if err := s.nodeRepo.UpdatePublishStatus(ctx, nodeID, status, lastError); err != nil {
		s.logger.ErrorWithErr(err, "Failed to update node publish status")
		return err
	}

	if node, err := s.nodeRepo.GetByID(ctx, nodeID); err == nil {
		_ = s.repo.TouchUpdatedAt(ctx, node.SitemapID)
	}

	return nil
}

func (s *service) GetNodesByGenerationStatus(ctx context.Context, sitemapID int64, status entities.NodeGenerationStatus) ([]*entities.SitemapNode, error) {
	return s.nodeRepo.GetByGenerationStatus(ctx, sitemapID, status)
}

func (s *service) validateSitemap(sitemap *entities.Sitemap) error {
	if strings.TrimSpace(sitemap.Name) == "" {
		return errors.Validation("Sitemap name is required")
	}
	if sitemap.SiteID == 0 {
		return errors.Validation("Site ID is required")
	}
	return nil
}

func (s *service) validateNode(node *entities.SitemapNode) error {
	if strings.TrimSpace(node.Title) == "" {
		return errors.Validation("Node title is required")
	}
	if strings.TrimSpace(node.Slug) == "" {
		return errors.Validation("Node slug is required")
	}
	if node.SitemapID == 0 {
		return errors.Validation("Sitemap ID is required")
	}
	return nil
}

func (s *service) calculateNodeHierarchy(ctx context.Context, node *entities.SitemapNode) error {
	if node.ParentID == nil {
		node.Depth = 0
		node.Path = "/" + node.Slug
		return nil
	}

	parent, err := s.nodeRepo.GetByID(ctx, *node.ParentID)
	if err != nil {
		return fmt.Errorf("parent node not found: %w", err)
	}

	node.Depth = parent.Depth + 1
	// Avoid double slashes when parent is root (path = "/")
	if parent.Path == "/" {
		node.Path = "/" + node.Slug
	} else {
		node.Path = parent.Path + "/" + node.Slug
	}
	return nil
}

func (s *service) buildTree(nodes []*entities.SitemapNode) []*entities.SitemapNode {
	nodeMap := make(map[int64]*entities.SitemapNode)
	var roots []*entities.SitemapNode

	// First pass: create map
	for _, node := range nodes {
		node.Children = []*entities.SitemapNode{}
		nodeMap[node.ID] = node
	}

	// Second pass: build tree
	for _, node := range nodes {
		if node.ParentID == nil {
			roots = append(roots, node)
		} else if parent, ok := nodeMap[*node.ParentID]; ok {
			parent.Children = append(parent.Children, node)
		}
	}

	return roots
}

func (s *service) maxDepth(nodes []*entities.SitemapNode) int {
	maxDepth := 0
	for _, node := range nodes {
		if node.Depth > maxDepth {
			maxDepth = node.Depth
		}
	}
	return maxDepth
}

func (s *service) copyNode(original *entities.SitemapNode, newSitemapID int64, newParentID *int64) *entities.SitemapNode {
	return &entities.SitemapNode{
		SitemapID:        newSitemapID,
		ParentID:         newParentID,
		Title:            original.Title,
		Slug:             original.Slug,
		Description:      original.Description,
		Depth:            original.Depth,
		Position:         original.Position,
		Path:             original.Path,
		ContentType:      original.ContentType,
		Source:           original.Source,
		IsSynced:         false,
		DesignStatus:     original.DesignStatus,
		GenerationStatus: entities.GenStatusNone,
		PublishStatus:    entities.PubStatusNone,
		PositionX:        original.PositionX,
		PositionY:        original.PositionY,
	}
}
