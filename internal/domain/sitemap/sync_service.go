package sitemap

import (
	"context"
	"fmt"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/sites"
	"github.com/davidmovas/postulator/internal/infra/wp"
	"github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"
)

// SyncService handles synchronization between sitemap nodes and WordPress
type SyncService struct {
	sitemapSvc  Service
	siteService sites.Service
	wpClient    wp.Client
	logger      *logger.Logger
}

// NewSyncService creates a new sync service
func NewSyncService(
	sitemapSvc Service,
	siteService sites.Service,
	wpClient wp.Client,
	logger *logger.Logger,
) *SyncService {
	return &SyncService{
		sitemapSvc:  sitemapSvc,
		siteService: siteService,
		wpClient:    wpClient,
		logger: logger.
			WithScope("service").
			WithScope("sitemap_sync"),
	}
}

// SyncResult contains the result of a sync operation
type SyncResult struct {
	NodeID  int64
	Success bool
	Error   string
}

// SyncFromWP fetches data from WordPress and updates the local node(s)
// This resets local changes and pulls the latest data from WP
func (s *SyncService) SyncFromWP(ctx context.Context, siteID int64, nodeIDs []int64) ([]SyncResult, error) {
	// Get site with credentials
	site, err := s.siteService.GetSiteWithPassword(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("failed to get site: %w", err)
	}

	results := make([]SyncResult, 0, len(nodeIDs))

	for _, nodeID := range nodeIDs {
		result := SyncResult{NodeID: nodeID}

		// Get the node
		node, err := s.sitemapSvc.GetNode(ctx, nodeID)
		if err != nil {
			result.Error = fmt.Sprintf("failed to get node: %v", err)
			results = append(results, result)
			continue
		}

		// Node must have WP ID to sync
		if node.WPPageID == nil {
			result.Error = "node is not linked to WordPress"
			results = append(results, result)
			continue
		}

		wpID := *node.WPPageID

		// Fetch from WP based on content type
		var wpTitle, wpSlug, wpURL string
		var wpStatus string

		switch node.ContentType {
		case entities.NodeContentTypePage:
			page, err := s.wpClient.GetPage(ctx, site, wpID)
			if err != nil {
				result.Error = fmt.Sprintf("failed to fetch page from WP: %v", err)
				results = append(results, result)
				continue
			}
			wpTitle = page.Title
			wpSlug = page.Slug
			wpURL = page.Link
			wpStatus = page.Status

		case entities.NodeContentTypePost:
			article, err := s.wpClient.GetPost(ctx, site, wpID)
			if err != nil {
				result.Error = fmt.Sprintf("failed to fetch post from WP: %v", err)
				results = append(results, result)
				continue
			}
			wpTitle = article.Title
			if article.Slug != nil {
				wpSlug = *article.Slug
			}
			wpURL = article.WPPostURL
			wpStatus = ArticleStatusToWPStatus(article.Status)

		default:
			result.Error = "unsupported content type for sync"
			results = append(results, result)
			continue
		}

		node.Title = wpTitle
		node.Slug = wpSlug
		node.WPURL = &wpURL
		node.WPTitle = &wpTitle
		node.WPSlug = &wpSlug
		node.PublishStatus = WPStatusToPublishStatus(wpStatus)
		node.IsModifiedLocally = false
		node.IsSynced = true
		now := time.Now()
		node.LastSyncedAt = &now

		if err := s.sitemapSvc.UpdateNode(ctx, node); err != nil {
			result.Error = fmt.Sprintf("failed to update node: %v", err)
			results = append(results, result)
			continue
		}

		result.Success = true
		results = append(results, result)
		s.logger.Infof("Synced node %d from WP (ID: %d)", nodeID, wpID)
	}

	return results, nil
}

// UpdateToWP pushes local node data to WordPress
// This updates the WP page/post with local changes
func (s *SyncService) UpdateToWP(ctx context.Context, siteID int64, nodeIDs []int64) ([]SyncResult, error) {
	// Get site with credentials
	site, err := s.siteService.GetSiteWithPassword(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("failed to get site: %w", err)
	}

	results := make([]SyncResult, 0, len(nodeIDs))

	for _, nodeID := range nodeIDs {
		result := SyncResult{NodeID: nodeID}

		// Get the node
		node, err := s.sitemapSvc.GetNode(ctx, nodeID)
		if err != nil {
			result.Error = fmt.Sprintf("failed to get node: %v", err)
			results = append(results, result)
			continue
		}

		// Node must have WP ID to update
		if node.WPPageID == nil {
			result.Error = "node is not linked to WordPress"
			results = append(results, result)
			continue
		}

		// Node must be modified to update (has local changes)
		if !node.IsModified() {
			result.Error = "node has no local changes to push"
			results = append(results, result)
			continue
		}

		wpID := *node.WPPageID

		// Update WP based on content type
		switch node.ContentType {
		case entities.NodeContentTypePage:
			page := &wp.WPPage{
				ID:    wpID,
				Title: node.Title,
				Slug:  node.Slug,
			}
			if err := s.wpClient.UpdatePage(ctx, site, page); err != nil {
				result.Error = fmt.Sprintf("failed to update page in WP: %v", err)
				results = append(results, result)
				continue
			}
			// Update local node with response data
			node.WPURL = &page.Link
			node.Slug = page.Slug

		case entities.NodeContentTypePost:
			article := &entities.Article{
				WPPostID: wpID,
				Title:    node.Title,
				Slug:     &node.Slug,
			}
			if err := s.wpClient.UpdatePost(ctx, site, article); err != nil {
				result.Error = fmt.Sprintf("failed to update post in WP: %v", err)
				results = append(results, result)
				continue
			}
			// Update local node with response data
			node.WPURL = &article.WPPostURL
			if article.Slug != nil {
				node.Slug = *article.Slug
			}

		default:
			result.Error = "unsupported content type for update"
			results = append(results, result)
			continue
		}

		// Update WP fields to match current local values (they are now synced)
		node.WPTitle = &node.Title
		node.WPSlug = &node.Slug
		node.IsSynced = true
		now := time.Now()
		node.LastSyncedAt = &now

		if err := s.sitemapSvc.UpdateNode(ctx, node); err != nil {
			result.Error = fmt.Sprintf("failed to update local node: %v", err)
			results = append(results, result)
			continue
		}

		result.Success = true
		results = append(results, result)
		s.logger.Infof("Updated WP from node %d (WP ID: %d)", nodeID, wpID)
	}

	return results, nil
}

// GetModifiedNodes returns all nodes in a sitemap that have local modifications
func (s *SyncService) GetModifiedNodes(ctx context.Context, sitemapID int64) ([]*entities.SitemapNode, error) {
	nodes, err := s.sitemapSvc.GetNodes(ctx, sitemapID)
	if err != nil {
		return nil, err
	}

	modified := make([]*entities.SitemapNode, 0)
	for _, node := range nodes {
		if node.IsModified() {
			modified = append(modified, node)
		}
	}

	return modified, nil
}

// ResetNode resets a node to its original WP data without fetching from WP
func (s *SyncService) ResetNode(ctx context.Context, nodeID int64) error {
	node, err := s.sitemapSvc.GetNode(ctx, nodeID)
	if err != nil {
		return err
	}

	if node.WPTitle == nil || node.WPSlug == nil {
		return errors.Validation("node has no original WP data to reset to")
	}

	node.Title = *node.WPTitle
	node.Slug = *node.WPSlug

	return s.sitemapSvc.UpdateNode(ctx, node)
}

// ChangePublishStatus changes the publish status of a node both locally and in WordPress
func (s *SyncService) ChangePublishStatus(ctx context.Context, siteID int64, nodeID int64, newStatus entities.NodePublishStatus) error {
	// Get site with credentials
	site, err := s.siteService.GetSiteWithPassword(ctx, siteID)
	if err != nil {
		return fmt.Errorf("failed to get site: %w", err)
	}

	// Get the node
	node, err := s.sitemapSvc.GetNode(ctx, nodeID)
	if err != nil {
		return fmt.Errorf("failed to get node: %w", err)
	}

	// Node must have WP ID to change status
	if node.WPPageID == nil {
		return errors.Validation("node is not linked to WordPress")
	}

	wpID := *node.WPPageID

	// Map our status to WP status
	wpStatus := PublishStatusToWPStatus(newStatus)

	// Update WP based on content type
	switch node.ContentType {
	case entities.NodeContentTypePage:
		page := &wp.WPPage{
			ID:     wpID,
			Status: wpStatus,
		}
		if err := s.wpClient.UpdatePage(ctx, site, page); err != nil {
			return fmt.Errorf("failed to update page status in WP: %w", err)
		}

	case entities.NodeContentTypePost:
		article := &entities.Article{
			WPPostID: wpID,
			Status:   WPStatusToArticleStatus(wpStatus),
		}
		if err := s.wpClient.UpdatePost(ctx, site, article); err != nil {
			return fmt.Errorf("failed to update post status in WP: %w", err)
		}

	default:
		return errors.Validation("unsupported content type")
	}

	// Update local node status
	node.PublishStatus = newStatus
	now := time.Now()
	node.LastSyncedAt = &now

	if err := s.sitemapSvc.UpdateNode(ctx, node); err != nil {
		return fmt.Errorf("failed to update local node: %w", err)
	}

	s.logger.Infof("Changed publish status for node %d to %s", nodeID, newStatus)
	return nil
}

// BatchChangeStatusResult contains the result of changing status for a single node
type BatchChangeStatusResult struct {
	NodeID  int64
	Success bool
	Error   string
}

// BatchChangePublishStatus changes the publish status of multiple nodes both locally and in WordPress
func (s *SyncService) BatchChangePublishStatus(
	ctx context.Context,
	siteID int64,
	nodeIDs []int64,
	newStatus entities.NodePublishStatus,
) ([]BatchChangeStatusResult, error) {
	if len(nodeIDs) == 0 {
		return nil, nil
	}

	results := make([]BatchChangeStatusResult, 0, len(nodeIDs))

	for _, nodeID := range nodeIDs {
		result := BatchChangeStatusResult{NodeID: nodeID}

		err := s.ChangePublishStatus(ctx, siteID, nodeID, newStatus)
		if err != nil {
			result.Error = err.Error()
		} else {
			result.Success = true
		}

		results = append(results, result)
	}

	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
	}

	s.logger.Infof("Batch changed publish status for %d/%d nodes to %s", successCount, len(nodeIDs), newStatus)
	return results, nil
}

// DeleteNodeResult contains the result of deleting a node
type DeleteNodeResult struct {
	NodeID        int64
	Success       bool
	Error         string
	ChildrenMoved int
	DeletedFromWP bool
}

// DeleteNodeWithWP deletes a node from WordPress and locally, moving children to parent
func (s *SyncService) DeleteNodeWithWP(
	ctx context.Context,
	siteID int64,
	nodeID int64,
) (*DeleteNodeResult, error) {
	result := &DeleteNodeResult{NodeID: nodeID}

	// Get the node
	node, err := s.sitemapSvc.GetNode(ctx, nodeID)
	if err != nil {
		result.Error = fmt.Sprintf("failed to get node: %v", err)
		return result, err
	}

	// Get site with credentials
	site, err := s.siteService.GetSiteWithPassword(ctx, siteID)
	if err != nil {
		result.Error = fmt.Sprintf("failed to get site: %v", err)
		return result, err
	}

	// Reparent children to parent node (like WordPress does)
	childrenMoved, err := s.sitemapSvc.ReparentChildren(ctx, nodeID, node.ParentID)
	if err != nil {
		result.Error = fmt.Sprintf("failed to reparent children: %v", err)
		return result, err
	}
	result.ChildrenMoved = childrenMoved

	// Delete from WordPress if node has WP ID
	if node.WPPageID != nil {
		wpID := *node.WPPageID

		switch node.ContentType {
		case entities.NodeContentTypePage:
			if err := s.wpClient.DeletePage(ctx, site, wpID, true); err != nil {
				result.Error = fmt.Sprintf("failed to delete page from WP: %v", err)
				return result, err
			}
			result.DeletedFromWP = true

		case entities.NodeContentTypePost:
			if err := s.wpClient.DeletePost(ctx, site, wpID); err != nil {
				result.Error = fmt.Sprintf("failed to delete post from WP: %v", err)
				return result, err
			}
			result.DeletedFromWP = true
		}
	}

	// Delete locally
	if err := s.sitemapSvc.DeleteNode(ctx, nodeID); err != nil {
		result.Error = fmt.Sprintf("failed to delete node locally: %v", err)
		return result, err
	}

	result.Success = true
	s.logger.Infof("Deleted node %d (WP: %v, children moved: %d)", nodeID, result.DeletedFromWP, childrenMoved)
	return result, nil
}

// BatchDeleteNodesWithWP deletes multiple nodes from WordPress and locally
func (s *SyncService) BatchDeleteNodesWithWP(
	ctx context.Context,
	siteID int64,
	nodeIDs []int64,
) ([]DeleteNodeResult, error) {
	if len(nodeIDs) == 0 {
		return nil, nil
	}

	// Get all nodes to sort by depth (delete leaves first)
	nodes := make([]*entities.SitemapNode, 0, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		node, err := s.sitemapSvc.GetNode(ctx, nodeID)
		if err != nil {
			continue
		}
		nodes = append(nodes, node)
	}

	// Sort by depth descending (leaves first)
	for i := 0; i < len(nodes)-1; i++ {
		for j := i + 1; j < len(nodes); j++ {
			if nodes[i].Depth < nodes[j].Depth {
				nodes[i], nodes[j] = nodes[j], nodes[i]
			}
		}
	}

	results := make([]DeleteNodeResult, 0, len(nodes))

	for _, node := range nodes {
		result, _ := s.DeleteNodeWithWP(ctx, siteID, node.ID)
		if result != nil {
			results = append(results, *result)
		}
	}

	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
	}

	s.logger.Infof("Batch deleted %d/%d nodes", successCount, len(nodeIDs))
	return results, nil
}
