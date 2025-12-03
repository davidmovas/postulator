package handlers

import (
	"encoding/base64"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/sitemap"
	"github.com/davidmovas/postulator/internal/domain/sitemap/importer"
	"github.com/davidmovas/postulator/internal/domain/sitemap/scanner"
	"github.com/davidmovas/postulator/internal/dto"
	"github.com/davidmovas/postulator/pkg/ctx"
	"github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"
)

type SitemapsHandler struct {
	service     sitemap.Service
	syncService *sitemap.SyncService
	importer    *importer.Importer
	scanner     *scanner.Scanner
}

func NewSitemapsHandler(
	service sitemap.Service,
	syncService *sitemap.SyncService,
	siteScanner *scanner.Scanner,
	log *logger.Logger,
) *SitemapsHandler {
	return &SitemapsHandler{
		service:     service,
		syncService: syncService,
		importer:    importer.NewImporter(log),
		scanner:     siteScanner,
	}
}

// =========================================================================
// Sitemap Operations
// =========================================================================

func (h *SitemapsHandler) CreateSitemap(req *dto.CreateSitemapRequest) *dto.Response[*dto.Sitemap] {
	entity := &entities.Sitemap{
		SiteID:      req.SiteID,
		Name:        req.Name,
		Description: req.Description,
		Source:      entities.SitemapSource(req.Source),
	}

	// Use CreateSitemapWithRoot to always create a root node
	if err := h.service.CreateSitemapWithRoot(ctx.FastCtx(), entity, req.SiteURL); err != nil {
		return fail[*dto.Sitemap](err)
	}

	return ok(dto.NewSitemap(entity))
}

func (h *SitemapsHandler) GetSitemap(id int64) *dto.Response[*dto.Sitemap] {
	entity, err := h.service.GetSitemap(ctx.FastCtx(), id)
	if err != nil {
		return fail[*dto.Sitemap](err)
	}

	return ok(dto.NewSitemap(entity))
}

func (h *SitemapsHandler) GetSitemapWithNodes(id int64) *dto.Response[*dto.SitemapWithNodes] {
	sitemapEntity, nodes, err := h.service.GetSitemapWithNodes(ctx.FastCtx(), id)
	if err != nil {
		return fail[*dto.SitemapWithNodes](err)
	}

	dtoNodes := make([]*dto.SitemapNode, len(nodes))
	for i, node := range nodes {
		dtoNodes[i] = dto.NewSitemapNode(node)
	}

	return ok(&dto.SitemapWithNodes{
		Sitemap: dto.NewSitemap(sitemapEntity),
		Nodes:   dtoNodes,
	})
}

func (h *SitemapsHandler) ListSitemaps(siteID int64) *dto.Response[[]*dto.Sitemap] {
	sitemaps, err := h.service.ListSitemaps(ctx.FastCtx(), siteID)
	if err != nil {
		return fail[[]*dto.Sitemap](err)
	}

	dtoSitemaps := make([]*dto.Sitemap, len(sitemaps))
	for i, s := range sitemaps {
		dtoSitemaps[i] = dto.NewSitemap(s)
	}

	return ok(dtoSitemaps)
}

func (h *SitemapsHandler) UpdateSitemap(req *dto.UpdateSitemapRequest) *dto.Response[string] {
	entity := &entities.Sitemap{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
		Status:      entities.SitemapStatus(req.Status),
	}

	if err := h.service.UpdateSitemap(ctx.FastCtx(), entity); err != nil {
		return fail[string](err)
	}

	return ok("Sitemap updated successfully")
}

func (h *SitemapsHandler) DeleteSitemap(id int64) *dto.Response[string] {
	if err := h.service.DeleteSitemap(ctx.FastCtx(), id); err != nil {
		return fail[string](err)
	}

	return ok("Sitemap deleted successfully")
}

func (h *SitemapsHandler) DuplicateSitemap(req *dto.DuplicateSitemapRequest) *dto.Response[*dto.Sitemap] {
	entity, err := h.service.DuplicateSitemap(ctx.FastCtx(), req.ID, req.NewName)
	if err != nil {
		return fail[*dto.Sitemap](err)
	}

	return ok(dto.NewSitemap(entity))
}

func (h *SitemapsHandler) SetSitemapStatus(id int64, status string) *dto.Response[string] {
	if err := h.service.SetSitemapStatus(ctx.FastCtx(), id, entities.SitemapStatus(status)); err != nil {
		return fail[string](err)
	}

	return ok("Sitemap status updated successfully")
}

// =========================================================================
// Node Operations
// =========================================================================

func (h *SitemapsHandler) CreateNode(req *dto.CreateNodeRequest) *dto.Response[*dto.SitemapNode] {
	entity := &entities.SitemapNode{
		SitemapID:   req.SitemapID,
		ParentID:    req.ParentID,
		Title:       req.Title,
		Slug:        req.Slug,
		Description: req.Description,
		Position:    req.Position,
		Source:      entities.NodeSource(req.Source),
		Keywords:    req.Keywords,
	}

	if err := h.service.CreateNode(ctx.FastCtx(), entity); err != nil {
		return fail[*dto.SitemapNode](err)
	}

	return ok(dto.NewSitemapNode(entity))
}

func (h *SitemapsHandler) GetNode(id int64) *dto.Response[*dto.SitemapNode] {
	node, err := h.service.GetNodeWithKeywords(ctx.FastCtx(), id)
	if err != nil {
		return fail[*dto.SitemapNode](err)
	}

	return ok(dto.NewSitemapNode(node))
}

func (h *SitemapsHandler) GetNodes(sitemapID int64) *dto.Response[[]*dto.SitemapNode] {
	nodes, err := h.service.GetNodes(ctx.FastCtx(), sitemapID)
	if err != nil {
		return fail[[]*dto.SitemapNode](err)
	}

	dtoNodes := make([]*dto.SitemapNode, len(nodes))
	for i, node := range nodes {
		dtoNodes[i] = dto.NewSitemapNode(node)
	}

	return ok(dtoNodes)
}

func (h *SitemapsHandler) GetNodesTree(sitemapID int64) *dto.Response[[]*dto.SitemapNode] {
	nodes, err := h.service.GetNodesTree(ctx.FastCtx(), sitemapID)
	if err != nil {
		return fail[[]*dto.SitemapNode](err)
	}

	dtoNodes := make([]*dto.SitemapNode, len(nodes))
	for i, node := range nodes {
		dtoNodes[i] = dto.NewSitemapNode(node)
	}

	return ok(dtoNodes)
}

func (h *SitemapsHandler) UpdateNode(req *dto.UpdateNodeRequest) *dto.Response[string] {
	// Get existing node first
	existingNode, err := h.service.GetNode(ctx.FastCtx(), req.ID)
	if err != nil {
		return fail[string](err)
	}

	existingNode.Title = req.Title
	existingNode.Slug = req.Slug
	existingNode.Description = req.Description

	if err = h.service.UpdateNode(ctx.FastCtx(), existingNode); err != nil {
		return fail[string](err)
	}

	// Update keywords if provided
	if req.Keywords != nil {
		if err = h.service.SetNodeKeywords(ctx.FastCtx(), req.ID, req.Keywords); err != nil {
			return fail[string](err)
		}
	}

	return ok("Node updated successfully")
}

func (h *SitemapsHandler) DeleteNode(id int64) *dto.Response[string] {
	if err := h.service.DeleteNode(ctx.FastCtx(), id); err != nil {
		return fail[string](err)
	}

	return ok("Node deleted successfully")
}

func (h *SitemapsHandler) MoveNode(req *dto.MoveNodeRequest) *dto.Response[string] {
	if err := h.service.MoveNode(ctx.FastCtx(), req.NodeID, req.NewParentID, req.Position); err != nil {
		return fail[string](err)
	}

	return ok("Node moved successfully")
}

func (h *SitemapsHandler) UpdateNodePositions(req *dto.UpdateNodePositionsRequest) *dto.Response[string] {
	if err := h.service.UpdateNodePositions(ctx.FastCtx(), req.NodeID, req.PositionX, req.PositionY); err != nil {
		return fail[string](err)
	}

	return ok("Node positions updated successfully")
}

// =========================================================================
// Keyword Operations
// =========================================================================

func (h *SitemapsHandler) SetNodeKeywords(req *dto.SetNodeKeywordsRequest) *dto.Response[string] {
	if err := h.service.SetNodeKeywords(ctx.FastCtx(), req.NodeID, req.Keywords); err != nil {
		return fail[string](err)
	}

	return ok("Keywords updated successfully")
}

func (h *SitemapsHandler) AddNodeKeyword(nodeID int64, keyword string) *dto.Response[string] {
	if err := h.service.AddNodeKeyword(ctx.FastCtx(), nodeID, keyword); err != nil {
		return fail[string](err)
	}

	return ok("Keyword added successfully")
}

func (h *SitemapsHandler) RemoveNodeKeyword(nodeID int64, keyword string) *dto.Response[string] {
	if err := h.service.RemoveNodeKeyword(ctx.FastCtx(), nodeID, keyword); err != nil {
		return fail[string](err)
	}

	return ok("Keyword removed successfully")
}

func (h *SitemapsHandler) DistributeKeywords(req *dto.DistributeKeywordsRequest) *dto.Response[string] {
	strategy := sitemap.KeywordDistributionStrategy(req.Strategy)
	if err := h.service.DistributeKeywords(ctx.FastCtx(), req.SitemapID, req.Keywords, strategy); err != nil {
		return fail[string](err)
	}

	return ok("Keywords distributed successfully")
}

// =========================================================================
// Content Linking Operations
// =========================================================================

func (h *SitemapsHandler) LinkNodeToArticle(req *dto.LinkNodeToArticleRequest) *dto.Response[string] {
	if err := h.service.LinkNodeToArticle(ctx.FastCtx(), req.NodeID, req.ArticleID); err != nil {
		return fail[string](err)
	}

	return ok("Node linked to article successfully")
}

func (h *SitemapsHandler) LinkNodeToPage(req *dto.LinkNodeToPageRequest) *dto.Response[string] {
	if err := h.service.LinkNodeToPage(ctx.FastCtx(), req.NodeID, req.WPPageID, req.WPURL); err != nil {
		return fail[string](err)
	}

	return ok("Node linked to page successfully")
}

func (h *SitemapsHandler) UnlinkNodeContent(nodeID int64) *dto.Response[string] {
	if err := h.service.UnlinkNodeContent(ctx.FastCtx(), nodeID); err != nil {
		return fail[string](err)
	}

	return ok("Node content unlinked successfully")
}

func (h *SitemapsHandler) UpdateNodeContentStatus(nodeID int64, status string) *dto.Response[string] {
	if err := h.service.UpdateNodeContentStatus(ctx.FastCtx(), nodeID, entities.NodeContentStatus(status)); err != nil {
		return fail[string](err)
	}

	return ok("Node content status updated successfully")
}

// =========================================================================
// Import Operations
// =========================================================================

func (h *SitemapsHandler) GetSupportedImportFormats() *dto.Response[*dto.SupportedFormatsResponse] {
	return ok(&dto.SupportedFormatsResponse{
		Formats: h.importer.SupportedFormats(),
	})
}

func (h *SitemapsHandler) ImportNodes(req *dto.ImportNodesRequest) *dto.Response[*dto.ImportNodesResponse] {
	// Decode base64 file data
	fileData, err := base64.StdEncoding.DecodeString(req.FileDataBase64)
	if err != nil {
		return fail[*dto.ImportNodesResponse](errors.Validation("Invalid file data: failed to decode base64"))
	}

	opts := &importer.ImportOptions{
		ParentNodeID: req.ParentNodeID,
	}

	stats, err := h.importer.Import(
		ctx.FastCtx(),
		req.Filename,
		fileData,
		req.SitemapID,
		h.service, // Service implements NodeCreator interface
		opts,
	)
	if err != nil {
		return fail[*dto.ImportNodesResponse](err)
	}

	// Convert import errors to DTO
	importErrors := make([]dto.ImportError, len(stats.Errors))
	for i, e := range stats.Errors {
		importErrors[i] = dto.ImportError{
			Row:     e.Row,
			Column:  e.Column,
			Message: e.Message,
		}
	}

	return ok(&dto.ImportNodesResponse{
		TotalRows:      stats.TotalRows,
		NodesCreated:   stats.NodesCreated,
		NodesSkipped:   stats.NodesSkipped,
		Errors:         importErrors,
		ProcessingTime: stats.ProcessingTime.String(),
	})
}

// =========================================================================
// Scanner Operations
// =========================================================================

func (h *SitemapsHandler) ScanSite(req *dto.ScanSiteRequest) *dto.Response[*dto.ScanSiteResponse] {
	opts := &scanner.ScanOptions{
		TitleSource:   scanner.TitleSource(req.TitleSource),
		ContentFilter: scanner.ContentFilter(req.ContentFilter),
		IncludeDrafts: req.IncludeDrafts,
		MaxDepth:      req.MaxDepth,
	}

	// Use defaults if not specified
	if opts.TitleSource == "" {
		opts.TitleSource = scanner.TitleSourceTitle
	}
	if opts.ContentFilter == "" {
		opts.ContentFilter = scanner.ContentFilterPages
	}

	result, err := h.scanner.ScanAndCreateSitemap(
		ctx.FastCtx(),
		req.SiteID,
		req.SitemapName,
		opts,
	)
	if err != nil {
		return fail[*dto.ScanSiteResponse](err)
	}

	// Convert errors to DTO
	scanErrors := make([]dto.ScanError, len(result.Errors))
	for i, e := range result.Errors {
		scanErrors[i] = dto.ScanError{
			WPID:    e.WPID,
			Type:    e.Type,
			Title:   e.Title,
			Message: e.Message,
		}
	}

	return ok(&dto.ScanSiteResponse{
		SitemapID:     result.SitemapID,
		PagesScanned:  result.PagesScanned,
		PostsScanned:  result.PostsScanned,
		NodesCreated:  result.NodesCreated,
		NodesSkipped:  result.NodesSkipped,
		TotalDuration: result.TotalDuration.String(),
		Errors:        scanErrors,
	})
}

func (h *SitemapsHandler) ScanIntoSitemap(req *dto.ScanIntoSitemapRequest) *dto.Response[*dto.ScanSiteResponse] {
	opts := &scanner.ScanOptions{
		TitleSource:   scanner.TitleSource(req.TitleSource),
		ContentFilter: scanner.ContentFilter(req.ContentFilter),
		IncludeDrafts: req.IncludeDrafts,
		MaxDepth:      req.MaxDepth,
	}

	// Use defaults if not specified
	if opts.TitleSource == "" {
		opts.TitleSource = scanner.TitleSourceTitle
	}
	if opts.ContentFilter == "" {
		opts.ContentFilter = scanner.ContentFilterPages
	}

	result, err := h.scanner.ScanIntoSitemap(
		ctx.FastCtx(),
		req.SitemapID,
		req.ParentNodeID,
		opts,
	)
	if err != nil {
		return fail[*dto.ScanSiteResponse](err)
	}

	// Convert errors to DTO
	scanErrors := make([]dto.ScanError, len(result.Errors))
	for i, e := range result.Errors {
		scanErrors[i] = dto.ScanError{
			WPID:    e.WPID,
			Type:    e.Type,
			Title:   e.Title,
			Message: e.Message,
		}
	}

	return ok(&dto.ScanSiteResponse{
		SitemapID:     result.SitemapID,
		PagesScanned:  result.PagesScanned,
		PostsScanned:  result.PostsScanned,
		NodesCreated:  result.NodesCreated,
		NodesSkipped:  result.NodesSkipped,
		TotalDuration: result.TotalDuration.String(),
		Errors:        scanErrors,
	})
}

// =========================================================================
// Sync Operations
// =========================================================================

// SyncNodesFromWP fetches data from WordPress and updates local nodes
// This resets local changes and pulls the latest data from WP
func (h *SitemapsHandler) SyncNodesFromWP(req *dto.SyncNodesRequest) *dto.Response[*dto.SyncNodesResponse] {
	results, err := h.syncService.SyncFromWP(ctx.FastCtx(), req.SiteID, req.NodeIDs)
	if err != nil {
		return fail[*dto.SyncNodesResponse](err)
	}

	dtoResults := make([]dto.SyncNodeResult, len(results))
	for i, r := range results {
		dtoResults[i] = dto.SyncNodeResult{
			NodeID:  r.NodeID,
			Success: r.Success,
			Error:   r.Error,
		}
	}

	return ok(&dto.SyncNodesResponse{Results: dtoResults})
}

// UpdateNodesToWP pushes local node data to WordPress
// This updates the WP page/post with local changes
func (h *SitemapsHandler) UpdateNodesToWP(req *dto.UpdateNodesToWPRequest) *dto.Response[*dto.SyncNodesResponse] {
	results, err := h.syncService.UpdateToWP(ctx.FastCtx(), req.SiteID, req.NodeIDs)
	if err != nil {
		return fail[*dto.SyncNodesResponse](err)
	}

	dtoResults := make([]dto.SyncNodeResult, len(results))
	for i, r := range results {
		dtoResults[i] = dto.SyncNodeResult{
			NodeID:  r.NodeID,
			Success: r.Success,
			Error:   r.Error,
		}
	}

	return ok(&dto.SyncNodesResponse{Results: dtoResults})
}

// ResetNode resets a node to its original WP data without fetching from WP
func (h *SitemapsHandler) ResetNode(nodeID int64) *dto.Response[string] {
	if err := h.syncService.ResetNode(ctx.FastCtx(), nodeID); err != nil {
		return fail[string](err)
	}

	return ok("Node reset successfully")
}
