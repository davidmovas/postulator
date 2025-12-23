package handlers

import (
	"context"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/sitemap"
	"github.com/davidmovas/postulator/internal/domain/sitemap/actions"
	"github.com/davidmovas/postulator/internal/domain/sitemap/generation"
	"github.com/davidmovas/postulator/internal/domain/sitemap/importer"
	"github.com/davidmovas/postulator/internal/domain/sitemap/scanner"
	"github.com/davidmovas/postulator/internal/dto"
	"github.com/davidmovas/postulator/pkg/ctx"
	"github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/history"
	"github.com/davidmovas/postulator/pkg/logger"
)

const (
	// HistoryMaxSize is the maximum number of undo actions per sitemap
	HistoryMaxSize = 25
	// HistoryTTL is how long to keep inactive history stacks
	HistoryTTL = 30 * time.Minute
)

type SitemapsHandler struct {
	service               sitemap.Service
	syncService           *sitemap.SyncService
	generationService     *sitemap.GenerationService
	pageGenerationService generation.Service
	importer              *importer.Importer
	scanner               *scanner.Scanner
	logger                *logger.Logger

	historyManager *history.Manager

	generationMu     sync.Mutex
	generationCancel context.CancelFunc
}

func NewSitemapsHandler(
	service sitemap.Service,
	syncService *sitemap.SyncService,
	generationService *sitemap.GenerationService,
	pageGenerationService generation.Service,
	siteScanner *scanner.Scanner,
	log *logger.Logger,
) *SitemapsHandler {
	return &SitemapsHandler{
		service:               service,
		syncService:           syncService,
		generationService:     generationService,
		pageGenerationService: pageGenerationService,
		importer:              importer.NewImporter(log),
		scanner:               siteScanner,
		logger:                log.WithScope("sitemaps_handler"),
		historyManager:        history.NewManager(HistoryMaxSize, HistoryTTL),
	}
}

// historyKey returns the history key for a sitemap
func historyKey(sitemapID int64) history.SourceKey {
	return history.SourceKey(fmt.Sprintf("sitemap:%d", sitemapID))
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
	// Clear history for this sitemap
	h.historyManager.Clear(historyKey(id))

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
// Node Operations (with History)
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

	// Record to history
	tx := history.NewTransaction(fmt.Sprintf("Create '%s'", entity.Title))
	action := actions.NewCreateNodeAction(h.service, entity, req.Keywords)
	tx.Add(action)
	h.historyManager.Record(historyKey(req.SitemapID), tx)

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
	c := ctx.FastCtx()

	// Get existing node first (with keywords)
	existingNode, err := h.service.GetNodeWithKeywords(c, req.ID)
	if err != nil {
		return fail[string](err)
	}

	// Capture old data for undo
	oldData := actions.NodeUpdateData{
		Title:       existingNode.Title,
		Slug:        existingNode.Slug,
		Description: existingNode.Description,
	}
	oldKeywords := existingNode.Keywords

	// Apply changes
	existingNode.Title = req.Title
	existingNode.Slug = req.Slug
	existingNode.Description = req.Description

	if err = h.service.UpdateNode(c, existingNode); err != nil {
		return fail[string](err)
	}

	// Update keywords if provided
	var newKeywords []string
	if req.Keywords != nil {
		newKeywords = req.Keywords
		if err = h.service.SetNodeKeywords(c, req.ID, req.Keywords); err != nil {
			return fail[string](err)
		}
	}

	// Record to history
	newData := actions.NodeUpdateData{
		Title:       req.Title,
		Slug:        req.Slug,
		Description: req.Description,
	}

	tx := history.NewTransaction(fmt.Sprintf("Update '%s'", req.Title))
	action := actions.NewUpdateNodeAction(h.service, req.ID, oldData, newData, oldKeywords, newKeywords)
	tx.Add(action)
	h.historyManager.Record(historyKey(existingNode.SitemapID), tx)

	return ok("Node updated successfully")
}

func (h *SitemapsHandler) DeleteNode(id int64) *dto.Response[string] {
	c := ctx.FastCtx()

	// Get node with keywords for undo snapshot
	node, err := h.service.GetNodeWithKeywords(c, id)
	if err != nil {
		return fail[string](err)
	}

	sitemapID := node.SitemapID

	// Create action before deletion (captures snapshot)
	action := actions.NewDeleteNodeAction(h.service, node, node.Keywords)

	if err = h.service.DeleteNode(c, id); err != nil {
		return fail[string](err)
	}

	// Record to history
	tx := history.NewTransaction(fmt.Sprintf("Delete '%s'", node.Title))
	tx.Add(action)
	h.historyManager.Record(historyKey(sitemapID), tx)

	return ok("Node deleted successfully")
}

func (h *SitemapsHandler) MoveNode(req *dto.MoveNodeRequest) *dto.Response[string] {
	c := ctx.FastCtx()

	// Get current node state for undo
	node, err := h.service.GetNode(c, req.NodeID)
	if err != nil {
		return fail[string](err)
	}

	oldParentID := node.ParentID
	oldPosition := node.Position

	if err = h.service.MoveNode(c, req.NodeID, req.NewParentID, req.Position); err != nil {
		return fail[string](err)
	}

	// Record to history
	tx := history.NewTransaction("Move node")
	action := actions.NewMoveNodeAction(h.service, req.NodeID, oldParentID, req.NewParentID, oldPosition, req.Position)
	tx.Add(action)
	h.historyManager.Record(historyKey(node.SitemapID), tx)

	return ok("Node moved successfully")
}

func (h *SitemapsHandler) UpdateNodePositions(req *dto.UpdateNodePositionsRequest) *dto.Response[string] {
	// Position updates are typically batched/saved separately
	// We don't track individual position changes in history
	// The "Save Layout" button saves all positions at once
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

func (h *SitemapsHandler) UpdateNodeDesignStatus(nodeID int64, status string) *dto.Response[string] {
	if err := h.service.UpdateNodeDesignStatus(ctx.FastCtx(), nodeID, entities.NodeDesignStatus(status)); err != nil {
		return fail[string](err)
	}
	return ok("Node design status updated successfully")
}

func (h *SitemapsHandler) UpdateNodeGenerationStatus(nodeID int64, status string) *dto.Response[string] {
	if err := h.service.UpdateNodeGenerationStatus(ctx.FastCtx(), nodeID, entities.NodeGenerationStatus(status), nil); err != nil {
		return fail[string](err)
	}
	return ok("Node generation status updated successfully")
}

func (h *SitemapsHandler) UpdateNodePublishStatus(nodeID int64, status string) *dto.Response[string] {
	if err := h.service.UpdateNodePublishStatus(ctx.FastCtx(), nodeID, entities.NodePublishStatus(status), nil); err != nil {
		return fail[string](err)
	}
	return ok("Node publish status updated successfully")
}

// =========================================================================
// Import Operations (with History)
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

	// Create batch action for tracking
	batchAction := actions.NewBatchCreateNodesAction(h.service, req.SitemapID, "Import")

	// Wrap service to track created node IDs
	trackingService := &trackingNodeCreator{
		service:     h.service,
		batchAction: batchAction,
	}

	stats, err := h.importer.Import(
		ctx.FastCtx(),
		req.Filename,
		fileData,
		req.SitemapID,
		trackingService,
		opts,
	)
	if err != nil {
		return fail[*dto.ImportNodesResponse](err)
	}

	// Record to history if nodes were created
	if len(batchAction.GetCreatedIDs()) > 0 {
		tx := history.NewTransaction(batchAction.Description())
		tx.Add(batchAction)
		h.historyManager.Record(historyKey(req.SitemapID), tx)
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
// Scanner Operations (with History)
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
		ctx.ScannerCtx(),
		req.SiteID,
		req.SitemapName,
		opts,
	)
	if err != nil {
		return fail[*dto.ScanSiteResponse](err)
	}

	// Record to history if nodes were created
	if result.NodesCreated > 0 && result.SitemapID > 0 {
		batchAction := actions.NewBatchCreateNodesAction(h.service, result.SitemapID, "Scan site")
		// We don't have individual IDs here, but we can record the operation
		// Note: This won't fully undo since we don't track individual node IDs from scan
		tx := history.NewTransaction(fmt.Sprintf("Scan site (%d nodes)", result.NodesCreated))
		tx.Add(batchAction)
		h.historyManager.Record(historyKey(result.SitemapID), tx)
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

	// Create batch action for tracking
	batchAction := actions.NewBatchCreateNodesAction(h.service, req.SitemapID, "Scan into sitemap")

	// Wrap scanner to track created node IDs
	result, err := h.scanner.ScanIntoSitemapWithTracking(
		ctx.ScannerCtx(),
		req.SitemapID,
		req.ParentNodeID,
		opts,
		func(nodeID int64) {
			batchAction.AddCreatedID(nodeID)
		},
	)
	if err != nil {
		return fail[*dto.ScanSiteResponse](err)
	}

	// Record to history if nodes were created
	if len(batchAction.GetCreatedIDs()) > 0 {
		tx := history.NewTransaction(batchAction.Description())
		tx.Add(batchAction)
		h.historyManager.Record(historyKey(req.SitemapID), tx)
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

// ChangePublishStatus changes the publish status of a node both locally and in WordPress
func (h *SitemapsHandler) ChangePublishStatus(req *dto.ChangePublishStatusRequest) *dto.Response[string] {
	status := entities.NodePublishStatus(req.NewStatus)
	if err := h.syncService.ChangePublishStatus(ctx.FastCtx(), req.SiteID, req.NodeID, status); err != nil {
		return fail[string](err)
	}

	return ok("Status changed successfully")
}

// =========================================================================
// AI Generation Operations (with History)
// =========================================================================

// GenerateSitemapStructure generates sitemap structure using AI
func (h *SitemapsHandler) GenerateSitemapStructure(req *dto.GenerateSitemapStructureRequest) *dto.Response[*dto.GenerateSitemapStructureResponse] {
	// Convert DTO titles to service input
	titles := make([]sitemap.TitleInput, len(req.Titles))
	for i, t := range req.Titles {
		titles[i] = sitemap.TitleInput{
			Title:    t.Title,
			Keywords: t.Keywords,
		}
	}

	input := sitemap.GenerationInput{
		SitemapID:           derefInt64(req.SitemapID),
		SiteID:              derefInt64(req.SiteID),
		Name:                req.Name,
		PromptID:            req.PromptID,
		Placeholders:        req.Placeholders,
		Titles:              titles,
		ParentNodeIDs:       req.ParentNodeIDs,
		MaxDepth:            req.MaxDepth,
		IncludeExistingTree: req.IncludeExistingTree,
		ProviderID:          req.ProviderID,
	}

	// Create cancellable context with AI timeout (5 minutes)
	genCtx, cancel := ctx.CancellableCtx(ctx.AIContextTimeout)

	// Store cancel function for potential cancellation
	h.generationMu.Lock()
	// Cancel any previous generation
	if h.generationCancel != nil {
		h.generationCancel()
	}
	h.generationCancel = cancel
	h.generationMu.Unlock()

	// Ensure we clean up the cancel function when done
	defer func() {
		h.generationMu.Lock()
		h.generationCancel = nil
		h.generationMu.Unlock()
		cancel() // Always call cancel to release resources
	}()

	// Create batch action for tracking
	var batchAction *actions.BatchCreateNodesAction
	sitemapID := input.SitemapID
	if sitemapID > 0 {
		batchAction = actions.NewBatchCreateNodesAction(h.service, sitemapID, "AI Generate")
	}

	result, err := h.generationService.GenerateStructureWithTracking(genCtx, input, func(nodeID int64) {
		if batchAction != nil {
			batchAction.AddCreatedID(nodeID)
		}
	})
	if err != nil {
		return fail[*dto.GenerateSitemapStructureResponse](err)
	}

	// Update sitemapID if it was created during generation
	if sitemapID == 0 && result.SitemapID > 0 {
		sitemapID = result.SitemapID
		batchAction = actions.NewBatchCreateNodesAction(h.service, sitemapID, "AI Generate")
		// Note: We don't have node IDs for newly created sitemap case
		// The generation service would need to be modified to track them
	}

	// Record to history if nodes were created
	if batchAction != nil && len(batchAction.GetCreatedIDs()) > 0 {
		tx := history.NewTransaction(batchAction.Description())
		tx.Add(batchAction)
		h.historyManager.Record(historyKey(sitemapID), tx)
	}

	return ok(&dto.GenerateSitemapStructureResponse{
		SitemapID:    result.SitemapID,
		NodesCreated: result.NodesCreated,
		DurationMs:   result.DurationMs,
	})
}

// CancelSitemapGeneration cancels the current AI generation operation
func (h *SitemapsHandler) CancelSitemapGeneration() *dto.Response[string] {
	h.generationMu.Lock()
	defer h.generationMu.Unlock()

	if h.generationCancel != nil {
		h.generationCancel()
		h.generationCancel = nil
		return ok("Generation cancelled")
	}

	return ok("No active generation to cancel")
}

// =========================================================================
// Page Content Generation Operations
// =========================================================================

func (h *SitemapsHandler) StartPageGeneration(req *dto.StartPageGenerationRequest) *dto.Response[*dto.GenerationTaskResponse] {
	sm, err := h.service.GetSitemap(ctx.FastCtx(), req.SitemapID)
	if err != nil {
		return fail[*dto.GenerationTaskResponse](err)
	}

	config := generation.GenerationConfig{
		SitemapID:      req.SitemapID,
		SiteID:         sm.SiteID,
		NodeIDs:        req.NodeIDs,
		ProviderID:     req.ProviderID,
		PromptID:       req.PromptID,
		PublishAs:      generation.PublishAs(req.PublishAs),
		Placeholders:   req.Placeholders,
		MaxConcurrency: req.MaxConcurrency,
	}

	// Map content settings from DTO to domain model
	if req.ContentSettings != nil {
		config.ContentSettings = &generation.ContentSettings{
			WordCount:          req.ContentSettings.WordCount,
			WritingStyle:       generation.WritingStyle(req.ContentSettings.WritingStyle),
			ContentTone:        generation.ContentTone(req.ContentSettings.ContentTone),
			CustomInstructions: req.ContentSettings.CustomInstructions,
		}
	}

	task, err := h.pageGenerationService.StartGeneration(context.Background(), config)
	if err != nil {
		return fail[*dto.GenerationTaskResponse](err)
	}

	return ok(h.taskToDTO(task))
}

func (h *SitemapsHandler) PausePageGeneration(taskID string) *dto.Response[string] {
	if err := h.pageGenerationService.PauseGeneration(taskID); err != nil {
		return fail[string](err)
	}
	return ok("Generation paused")
}

func (h *SitemapsHandler) ResumePageGeneration(taskID string) *dto.Response[string] {
	if err := h.pageGenerationService.ResumeGeneration(taskID); err != nil {
		return fail[string](err)
	}
	return ok("Generation resumed")
}

func (h *SitemapsHandler) CancelPageGeneration(taskID string) *dto.Response[string] {
	if err := h.pageGenerationService.CancelGeneration(taskID); err != nil {
		return fail[string](err)
	}
	return ok("Generation cancelled")
}

func (h *SitemapsHandler) GetPageGenerationTask(taskID string) *dto.Response[*dto.GenerationTaskResponse] {
	task := h.pageGenerationService.GetTask(taskID)
	if task == nil {
		return fail[*dto.GenerationTaskResponse](errors.NotFound("task", taskID))
	}
	return ok(h.taskToDTO(task))
}

func (h *SitemapsHandler) ListActivePageGenerationTasks() *dto.Response[[]*dto.GenerationTaskResponse] {
	tasks := h.pageGenerationService.ListActiveTasks()
	result := make([]*dto.GenerationTaskResponse, len(tasks))
	for i, task := range tasks {
		result[i] = h.taskToDTO(task)
	}
	return ok(result)
}

func (h *SitemapsHandler) GetDefaultPagePrompt() *dto.Response[*dto.DefaultPromptResponse] {
	prompt := h.pageGenerationService.GetDefaultPrompt()
	return ok(&dto.DefaultPromptResponse{
		Name:         prompt.Name,
		SystemPrompt: prompt.SystemPrompt,
		UserPrompt:   prompt.UserPrompt,
		Placeholders: prompt.Placeholders,
	})
}

func (h *SitemapsHandler) taskToDTO(task *generation.Task) *dto.GenerationTaskResponse {
	resp := &dto.GenerationTaskResponse{
		ID:             task.ID,
		SitemapID:      task.SitemapID,
		SiteID:         task.SiteID,
		TotalNodes:     task.TotalNodes,
		ProcessedNodes: task.ProcessedNodes,
		FailedNodes:    task.FailedNodes,
		SkippedNodes:   task.SkippedNodes,
		Status:         string(task.Status),
		StartedAt:      dto.TimeToString(task.StartedAt),
		Error:          task.Error,
	}

	if task.CompletedAt != nil {
		completedStr := dto.TimeToString(*task.CompletedAt)
		resp.CompletedAt = &completedStr
	}

	if len(task.Nodes) > 0 {
		resp.Nodes = make([]dto.GenerationNodeInfo, len(task.Nodes))
		for i, node := range task.Nodes {
			info := dto.GenerationNodeInfo{
				NodeID:    node.NodeID,
				Title:     node.Title,
				Path:      node.Path,
				Status:    string(node.Status),
				ArticleID: node.ArticleID,
				WPPageID:  node.WPPageID,
				WPURL:     node.WPURL,
				Error:     node.Error,
			}
			if node.StartedAt != nil {
				startedStr := dto.TimeToString(*node.StartedAt)
				info.StartedAt = &startedStr
			}
			if node.CompletedAt != nil {
				completedStr := dto.TimeToString(*node.CompletedAt)
				info.CompletedAt = &completedStr
			}
			resp.Nodes[i] = info
		}
	}

	return resp
}

// =========================================================================
// History Operations
// =========================================================================

// Undo undoes the last action for the specified sitemap
func (h *SitemapsHandler) Undo(sitemapID int64) *dto.Response[*dto.HistoryState] {
	c := ctx.MediumCtx()

	description, err := h.historyManager.Undo(c, historyKey(sitemapID))
	if err != nil {
		h.logger.ErrorWithErr(err, fmt.Sprintf("Undo failed for sitemap %d", sitemapID))
		return fail[*dto.HistoryState](err)
	}

	state := h.historyManager.GetState(historyKey(sitemapID))
	return ok(&dto.HistoryState{
		CanUndo:       state.CanUndo,
		CanRedo:       state.CanRedo,
		UndoCount:     state.UndoCount,
		RedoCount:     state.RedoCount,
		LastAction:    state.LastAction,
		ActionApplied: description,
	})
}

// Redo redoes the last undone action for the specified sitemap
func (h *SitemapsHandler) Redo(sitemapID int64) *dto.Response[*dto.HistoryState] {
	c := ctx.MediumCtx()

	description, err := h.historyManager.Redo(c, historyKey(sitemapID))
	if err != nil {
		h.logger.ErrorWithErr(err, fmt.Sprintf("Redo failed for sitemap %d", sitemapID))
		return fail[*dto.HistoryState](err)
	}

	state := h.historyManager.GetState(historyKey(sitemapID))
	return ok(&dto.HistoryState{
		CanUndo:       state.CanUndo,
		CanRedo:       state.CanRedo,
		UndoCount:     state.UndoCount,
		RedoCount:     state.RedoCount,
		LastAction:    state.LastAction,
		ActionApplied: description,
	})
}

// GetHistoryState returns the current history state for a sitemap
func (h *SitemapsHandler) GetHistoryState(sitemapID int64) *dto.Response[*dto.HistoryState] {
	state := h.historyManager.GetState(historyKey(sitemapID))
	return ok(&dto.HistoryState{
		CanUndo:    state.CanUndo,
		CanRedo:    state.CanRedo,
		UndoCount:  state.UndoCount,
		RedoCount:  state.RedoCount,
		LastAction: state.LastAction,
	})
}

// ClearHistory clears all history for a sitemap (called when editor closes)
func (h *SitemapsHandler) ClearHistory(sitemapID int64) *dto.Response[string] {
	h.historyManager.Clear(historyKey(sitemapID))
	return ok("History cleared")
}

// =========================================================================
// Helper Types
// =========================================================================

// trackingNodeCreator wraps the service to track created node IDs
type trackingNodeCreator struct {
	service     sitemap.Service
	batchAction *actions.BatchCreateNodesAction
}

func (t *trackingNodeCreator) CreateNode(ctx context.Context, node *entities.SitemapNode) error {
	if err := t.service.CreateNode(ctx, node); err != nil {
		return err
	}
	t.batchAction.AddCreatedID(node.ID)
	return nil
}

func (t *trackingNodeCreator) GetNodes(ctx context.Context, sitemapID int64) ([]*entities.SitemapNode, error) {
	return t.service.GetNodes(ctx, sitemapID)
}

func (t *trackingNodeCreator) FindNodeBySlugAndParent(ctx context.Context, sitemapID int64, slug string, parentID *int64) (*entities.SitemapNode, error) {
	return t.service.FindNodeBySlugAndParent(ctx, sitemapID, slug, parentID)
}

// derefInt64 safely dereferences an int64 pointer, returning 0 if nil
func derefInt64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}
