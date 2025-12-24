package linking

import (
	"context"
	"fmt"
	"time"

	"github.com/davidmovas/postulator/internal/domain/aiusage"
	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/providers"
	"github.com/davidmovas/postulator/internal/domain/sitemap"
	"github.com/davidmovas/postulator/internal/domain/sites"
	"github.com/davidmovas/postulator/internal/infra/ai"
	"github.com/davidmovas/postulator/internal/infra/wp"
	"github.com/davidmovas/postulator/pkg/logger"
)

type Applier struct {
	sitemapSvc     sitemap.Service
	sitesSvc       sites.Service
	providerSvc    providers.Service
	linkRepo       LinkRepository
	wpClient       wp.Client
	aiUsageService aiusage.Service
	logger         *logger.Logger
}

func NewApplier(
	sitemapSvc sitemap.Service,
	sitesSvc sites.Service,
	providerSvc providers.Service,
	linkRepo LinkRepository,
	wpClient wp.Client,
	aiUsageService aiusage.Service,
	logger *logger.Logger,
) *Applier {
	return &Applier{
		sitemapSvc:     sitemapSvc,
		sitesSvc:       sitesSvc,
		providerSvc:    providerSvc,
		linkRepo:       linkRepo,
		wpClient:       wpClient,
		aiUsageService: aiUsageService,
		logger:         logger.WithScope("linking.applier"),
	}
}

type ApplyConfig struct {
	PlanID     int64
	SiteID     int64
	ProviderID int64
	LinkIDs    []int64
}

func (a *Applier) Apply(ctx context.Context, config ApplyConfig) (*ApplyResult, error) {
	provider, err := a.providerSvc.GetProvider(ctx, config.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	if !provider.IsActive {
		return nil, fmt.Errorf("provider is not active")
	}

	aiClient, err := ai.CreateClient(provider)
	if err != nil {
		return nil, fmt.Errorf("failed to create AI client: %w", err)
	}

	site, err := a.sitesSvc.GetSiteWithPassword(ctx, config.SiteID)
	if err != nil {
		return nil, fmt.Errorf("failed to get site: %w", err)
	}

	// Get all requested links
	allLinks, err := a.linkRepo.GetByPlanID(ctx, config.PlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get links: %w", err)
	}

	// Filter to only approved links from the requested IDs
	linkIDSet := make(map[int64]bool)
	for _, id := range config.LinkIDs {
		linkIDSet[id] = true
	}

	var linksToApply []*PlannedLink
	for _, link := range allLinks {
		if linkIDSet[link.ID] && link.Status == LinkStatusApproved {
			linksToApply = append(linksToApply, link)
		}
	}

	if len(linksToApply) == 0 {
		return &ApplyResult{
			TotalLinks:   0,
			AppliedLinks: 0,
			FailedLinks:  0,
		}, nil
	}

	// Group links by source node
	linksBySource := make(map[int64][]*PlannedLink)
	for _, link := range linksToApply {
		linksBySource[link.SourceNodeID] = append(linksBySource[link.SourceNodeID], link)
	}

	result := &ApplyResult{
		TotalLinks: len(linksToApply),
		Results:    make([]*AppliedLinkInfo, 0),
	}

	// Process each source node
	for sourceNodeID, links := range linksBySource {
		appliedInfos, failedCount, err := a.applyLinksToNode(ctx, aiClient, site, sourceNodeID, links, config.SiteID)
		if err != nil {
			a.logger.ErrorWithErr(err, fmt.Sprintf("Failed to apply links to node %d", sourceNodeID))
			result.FailedLinks += len(links)
			// Note: applyLinksToNode already marked links as failed
			continue
		}
		result.AppliedLinks += len(appliedInfos)
		result.FailedLinks += failedCount
		result.Results = append(result.Results, appliedInfos...)
	}

	return result, nil
}

func (a *Applier) applyLinksToNode(
	ctx context.Context,
	aiClient ai.Client,
	site *entities.Site,
	sourceNodeID int64,
	links []*PlannedLink,
	siteID int64,
) ([]*AppliedLinkInfo, int, error) {
	// Get source node to find WordPress page ID
	sourceNode, err := a.sitemapSvc.GetNode(ctx, sourceNodeID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get source node: %w", err)
	}

	if sourceNode.WPPageID == nil {
		return nil, 0, fmt.Errorf("source node has no WordPress page ID")
	}

	// Get page content from WordPress
	page, err := a.wpClient.GetPage(ctx, site, *sourceNode.WPPageID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get WordPress page: %w", err)
	}

	if page.Content == "" {
		return nil, 0, fmt.Errorf("page has no content")
	}

	// Build link targets for AI, tracking which links are valid
	type linkWithTarget struct {
		link   *PlannedLink
		target ai.InsertLinkTarget
	}
	validLinks := make([]linkWithTarget, 0, len(links))

	for _, link := range links {
		targetNode, err := a.sitemapSvc.GetNode(ctx, link.TargetNodeID)
		if err != nil {
			a.logger.ErrorWithErr(err, fmt.Sprintf("Failed to get target node %d", link.TargetNodeID))
			// Mark this specific link as failed
			errMsg := fmt.Sprintf("failed to get target node: %v", err)
			_ = a.linkRepo.UpdateStatus(ctx, link.ID, LinkStatusFailed, &errMsg)
			continue
		}

		validLinks = append(validLinks, linkWithTarget{
			link: link,
			target: ai.InsertLinkTarget{
				TargetPath:  targetNode.Path,
				TargetTitle: targetNode.Title,
				AnchorText:  link.AnchorText,
			},
		})
	}

	if len(validLinks) == 0 {
		return nil, len(links), fmt.Errorf("no valid link targets")
	}

	// Count how many were already marked as failed
	failedCount := len(links) - len(validLinks)

	// Mark only valid links as applying
	for _, vl := range validLinks {
		_ = a.linkRepo.UpdateStatus(ctx, vl.link.ID, LinkStatusApplying, nil)
	}

	// Build AI request with valid targets only
	linkTargets := make([]ai.InsertLinkTarget, len(validLinks))
	for i, vl := range validLinks {
		linkTargets[i] = vl.target
	}

	// Use default language (could be extended to get from site settings)
	language := "English"

	// Call AI to insert links
	request := &ai.InsertLinksRequest{
		Content:   page.Content,
		PageTitle: sourceNode.Title,
		PagePath:  sourceNode.Path,
		Links:     linkTargets,
		Language:  language,
	}

	a.logger.Infof("Inserting %d links into page %s", len(linkTargets), sourceNode.Title)

	insertResult, err := aiClient.InsertLinks(ctx, request)
	if err != nil {
		// Mark all applying links as failed
		errMsg := fmt.Sprintf("AI insertion failed: %v", err)
		for _, vl := range validLinks {
			_ = a.linkRepo.UpdateStatus(ctx, vl.link.ID, LinkStatusFailed, &errMsg)
		}
		return nil, len(links), fmt.Errorf("AI link insertion failed: %w", err)
	}

	// Log AI usage
	if a.aiUsageService != nil {
		_ = a.aiUsageService.LogFromResult(
			ctx,
			siteID,
			aiusage.OpLinkInsertion,
			aiClient,
			insertResult.Usage,
			0,
			nil,
			map[string]interface{}{
				"source_node_id": sourceNodeID,
				"links_count":    len(linkTargets),
				"links_applied":  insertResult.LinksApplied,
			},
		)
	}

	if insertResult.LinksApplied == 0 {
		a.logger.Warnf("AI did not insert any links for node %d", sourceNodeID)
		// Revert links back to approved status (they weren't applied but aren't failed)
		for _, vl := range validLinks {
			_ = a.linkRepo.UpdateStatus(ctx, vl.link.ID, LinkStatusApproved, nil)
		}
		return nil, failedCount, nil
	}

	// Update WordPress page with new content
	page.Content = insertResult.Content
	if err := a.wpClient.UpdatePage(ctx, site, page); err != nil {
		// Mark all as failed since we couldn't save to WordPress
		errMsg := fmt.Sprintf("failed to update WordPress: %v", err)
		for _, vl := range validLinks {
			_ = a.linkRepo.UpdateStatus(ctx, vl.link.ID, LinkStatusFailed, &errMsg)
		}
		return nil, len(links), fmt.Errorf("failed to update WordPress page: %w", err)
	}

	a.logger.Infof("Successfully applied %d/%d links to page %s", insertResult.LinksApplied, len(validLinks), sourceNode.Title)

	// Mark links as applied
	// Note: AI returns count of applied links but not which specific ones.
	// We mark the first N links as applied. This is a best-effort approach.
	// In practice, if AI applies 2 out of 3 links, we assume it's the first 2.
	now := time.Now()
	appliedCount := insertResult.LinksApplied
	if appliedCount > len(validLinks) {
		appliedCount = len(validLinks)
	}

	appliedInfos := make([]*AppliedLinkInfo, 0, appliedCount)

	for i, vl := range validLinks {
		if i < appliedCount {
			vl.link.Status = LinkStatusApplied
			vl.link.AppliedAt = &now
			_ = a.linkRepo.Update(ctx, vl.link)

			// Build applied link info
			anchor := ""
			if vl.link.AnchorText != nil {
				anchor = *vl.link.AnchorText
			}
			appliedInfos = append(appliedInfos, &AppliedLinkInfo{
				LinkID:     vl.link.ID,
				AnchorText: anchor,
			})
		} else {
			// Links that weren't applied go back to approved
			_ = a.linkRepo.UpdateStatus(ctx, vl.link.ID, LinkStatusApproved, nil)
		}
	}

	// Count non-applied valid links as not failed (they went back to approved)
	notAppliedCount := len(validLinks) - appliedCount
	_ = notAppliedCount // Not counted as failed, just not applied this time

	return appliedInfos, failedCount, nil
}

