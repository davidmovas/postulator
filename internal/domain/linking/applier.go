package linking

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/davidmovas/postulator/internal/domain/aiusage"
	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/prompts"
	"github.com/davidmovas/postulator/internal/domain/providers"
	"github.com/davidmovas/postulator/internal/domain/sitemap"
	"github.com/davidmovas/postulator/internal/domain/sites"
	"github.com/davidmovas/postulator/internal/infra/ai"
	"github.com/davidmovas/postulator/internal/infra/events"
	"github.com/davidmovas/postulator/internal/infra/wp"
	"github.com/davidmovas/postulator/pkg/logger"
	"github.com/google/uuid"
)

type Applier struct {
	sitemapSvc     sitemap.Service
	sitesSvc       sites.Service
	providerSvc    providers.Service
	promptSvc      prompts.Service
	linkRepo       LinkRepository
	wpClient       wp.Client
	aiUsageService aiusage.Service
	eventBus       *events.EventBus
	emitter        *ApplyEventEmitter
	logger         *logger.Logger
}

func NewApplier(
	sitemapSvc sitemap.Service,
	sitesSvc sites.Service,
	providerSvc providers.Service,
	promptSvc prompts.Service,
	linkRepo LinkRepository,
	wpClient wp.Client,
	aiUsageService aiusage.Service,
	eventBus *events.EventBus,
	logger *logger.Logger,
) *Applier {
	return &Applier{
		sitemapSvc:     sitemapSvc,
		sitesSvc:       sitesSvc,
		providerSvc:    providerSvc,
		promptSvc:      promptSvc,
		linkRepo:       linkRepo,
		wpClient:       wpClient,
		aiUsageService: aiUsageService,
		eventBus:       eventBus,
		emitter:        NewApplyEventEmitter(eventBus),
		logger:         logger.WithScope("linking.applier"),
	}
}

type ApplyConfig struct {
	PlanID     int64
	SiteID     int64
	ProviderID int64
	PromptID   *int64
	LinkIDs    []int64
}

// calculateConcurrency determines optimal concurrency based on provider's RPM limits
// Returns a value between 1 and maxConcurrency based on the model's rate limits
func (a *Applier) calculateConcurrency(provider *entities.Provider) int {
	modelInfo := ai.GetModelInfo(provider.Type, provider.Model)
	if modelInfo == nil {
		return 1 // Default to sequential if model info not found
	}

	// Calculate concurrency based on RPM
	// Assume each request takes ~6 seconds on average (10 requests per minute per worker)
	// We want to stay safely under the limit, so use 80% of theoretical max
	concurrency := (modelInfo.RPM * RPMMultiplier) / RPMDivisor

	// Clamp between MinConcurrency and MaxConcurrency
	if concurrency < MinConcurrency {
		concurrency = MinConcurrency
	}
	if concurrency > MaxConcurrency {
		concurrency = MaxConcurrency
	}

	return concurrency
}

// sourceNodeWork represents work to be done for a single source node
type sourceNodeWork struct {
	sourceNodeID int64
	sourceNode   *entities.SitemapNode
	links        []*PlannedLink
}

// sourceNodeResult represents the result of processing a single source node
type sourceNodeResult struct {
	sourceNodeID int64
	appliedInfos []*AppliedLinkInfo
	failedCount  int
	err          error
}

func (a *Applier) Apply(ctx context.Context, config ApplyConfig) (*ApplyResult, error) {
	startTime := time.Now()
	taskID := uuid.New().String()

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

	// Group links by source node and prepare work items
	linksBySource := make(map[int64][]*PlannedLink)
	for _, link := range linksToApply {
		linksBySource[link.SourceNodeID] = append(linksBySource[link.SourceNodeID], link)
	}

	// Prepare work items with source node info
	workItems := make([]sourceNodeWork, 0, len(linksBySource))
	for sourceNodeID, links := range linksBySource {
		sourceNode, err := a.sitemapSvc.GetNode(ctx, sourceNodeID)
		if err != nil {
			a.logger.ErrorWithErr(err, fmt.Sprintf("Failed to get source node %d", sourceNodeID))
			// Mark all links for this node as failed
			errMsg := fmt.Sprintf("failed to get source node: %v", err)
			for _, link := range links {
				if updateErr := a.linkRepo.UpdateStatus(ctx, link.ID, LinkStatusFailed, &errMsg); updateErr != nil {
					a.logger.ErrorWithErr(updateErr, fmt.Sprintf("Failed to update link %d status", link.ID))
				}
			}
			continue
		}
		workItems = append(workItems, sourceNodeWork{
			sourceNodeID: sourceNodeID,
			sourceNode:   sourceNode,
			links:        links,
		})
	}

	if len(workItems) == 0 {
		return &ApplyResult{
			TotalLinks:   len(linksToApply),
			AppliedLinks: 0,
			FailedLinks:  len(linksToApply),
		}, nil
	}

	// Get prompt for link insertion (custom or builtin)
	prompt := a.getApplyPrompt(ctx, config.PromptID)

	// Emit start event
	a.emitter.EmitApplyStarted(ctx, taskID, len(linksToApply), len(workItems))

	result := &ApplyResult{
		TotalLinks: len(linksToApply),
		Results:    make([]*AppliedLinkInfo, 0),
	}

	// Determine concurrency based on provider's rate limits
	concurrency := a.calculateConcurrency(provider)
	if concurrency > len(workItems) {
		concurrency = len(workItems)
	}
	a.logger.Infof("Using concurrency %d for %d pages (RPM-based)", concurrency, len(workItems))

	// Process work items with concurrency
	results := a.processWorkItems(ctx, taskID, workItems, aiClient, site, config.SiteID, concurrency, prompt)

	// Aggregate results
	for _, r := range results {
		if r.err != nil {
			result.FailedLinks += r.failedCount
		} else {
			result.AppliedLinks += len(r.appliedInfos)
			result.FailedLinks += r.failedCount
			result.Results = append(result.Results, r.appliedInfos...)
		}
	}

	// Emit completion event
	a.emitter.EmitApplyCompleted(ctx, taskID, len(linksToApply), result.AppliedLinks, result.FailedLinks, startTime)

	return result, nil
}

func (a *Applier) processWorkItems(
	ctx context.Context,
	taskID string,
	workItems []sourceNodeWork,
	aiClient ai.Client,
	site *entities.Site,
	siteID int64,
	concurrency int,
	prompt *entities.Prompt,
) []sourceNodeResult {
	results := make([]sourceNodeResult, len(workItems))
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Semaphore for concurrency limiting
	sem := make(chan struct{}, concurrency)

	// Progress tracking
	var processedPages int
	var totalApplied, totalFailed int

	// Track cancelled state
	var cancelled bool
	var cancelledMu sync.Mutex

	for i, work := range workItems {
		wg.Add(1)
		go func(idx int, w sourceNodeWork) {
			defer wg.Done()

			// Check for cancellation before acquiring semaphore
			select {
			case <-ctx.Done():
				cancelledMu.Lock()
				if !cancelled {
					cancelled = true
					a.logger.Infof("Apply operation cancelled")
					a.emitter.EmitApplyCancelled(ctx, taskID, processedPages, totalApplied)
				}
				cancelledMu.Unlock()
				return
			default:
			}

			// Acquire semaphore
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				cancelledMu.Lock()
				if !cancelled {
					cancelled = true
					a.logger.Infof("Apply operation cancelled")
					a.emitter.EmitApplyCancelled(ctx, taskID, processedPages, totalApplied)
				}
				cancelledMu.Unlock()
				return
			}

			// Emit page processing event
			a.emitter.EmitPageProcessing(ctx, taskID, w.sourceNodeID, w.sourceNode.Title, len(w.links))

			// Process the node
			appliedInfos, failedCount, err := a.applyLinksToNode(ctx, aiClient, site, w.sourceNode, w.links, siteID, prompt)

			// Store result
			results[idx] = sourceNodeResult{
				sourceNodeID: w.sourceNodeID,
				appliedInfos: appliedInfos,
				failedCount:  failedCount,
				err:          err,
			}

			// Update progress and emit events
			mu.Lock()
			processedPages++
			if err != nil {
				totalFailed += len(w.links)
				a.emitter.EmitPageFailed(ctx, taskID, w.sourceNodeID, w.sourceNode.Title, err.Error())
			} else {
				totalApplied += len(appliedInfos)
				totalFailed += failedCount
				a.emitter.EmitPageCompleted(ctx, taskID, w.sourceNodeID, w.sourceNode.Title, len(appliedInfos), failedCount)
			}

			// Emit progress
			a.emitter.EmitApplyProgress(ctx, taskID, processedPages, len(workItems), totalApplied, totalFailed, &PageInfo{
				NodeID: w.sourceNodeID,
				Title:  w.sourceNode.Title,
				Path:   w.sourceNode.Path,
			})
			mu.Unlock()
		}(i, work)
	}

	wg.Wait()
	return results
}

func (a *Applier) applyLinksToNode(
	ctx context.Context,
	aiClient ai.Client,
	site *entities.Site,
	sourceNode *entities.SitemapNode,
	links []*PlannedLink,
	siteID int64,
	prompt *entities.Prompt,
) ([]*AppliedLinkInfo, int, error) {
	if sourceNode.WPPageID == nil {
		return nil, len(links), fmt.Errorf("source node has no WordPress page ID")
	}

	// Get page content from WordPress
	page, err := a.wpClient.GetPage(ctx, site, *sourceNode.WPPageID)
	if err != nil {
		return nil, len(links), fmt.Errorf("failed to get WordPress page: %w", err)
	}

	if page.Content == "" {
		return nil, len(links), fmt.Errorf("page has no content")
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
			errMsg := fmt.Sprintf("failed to get target node: %v", err)
			if updateErr := a.linkRepo.UpdateStatus(ctx, link.ID, LinkStatusFailed, &errMsg); updateErr != nil {
				a.logger.ErrorWithErr(updateErr, fmt.Sprintf("Failed to update link %d status", link.ID))
			}
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

	failedCount := len(links) - len(validLinks)

	// Mark only valid links as applying
	for _, vl := range validLinks {
		if updateErr := a.linkRepo.UpdateStatus(ctx, vl.link.ID, LinkStatusApplying, nil); updateErr != nil {
			a.logger.ErrorWithErr(updateErr, fmt.Sprintf("Failed to update link %d status to applying", vl.link.ID))
		}
	}

	// Build AI request with valid targets only
	linkTargets := make([]ai.InsertLinkTarget, len(validLinks))
	for i, vl := range validLinks {
		linkTargets[i] = vl.target
	}

	// Build prompts from DB template or use defaults
	systemPrompt, userPrompt := a.buildApplyPrompts(prompt, sourceNode, linkTargets, page.Content, DefaultLanguage)

	request := &ai.InsertLinksRequest{
		Content:      page.Content,
		PageTitle:    sourceNode.Title,
		PagePath:     sourceNode.Path,
		Links:        linkTargets,
		Language:     DefaultLanguage,
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
	}

	a.logger.Infof("Inserting %d links into page %s", len(linkTargets), sourceNode.Title)

	insertResult, err := aiClient.InsertLinks(ctx, request)
	if err != nil {
		errMsg := fmt.Sprintf("AI insertion failed: %v", err)
		for _, vl := range validLinks {
			if updateErr := a.linkRepo.UpdateStatus(ctx, vl.link.ID, LinkStatusFailed, &errMsg); updateErr != nil {
				a.logger.ErrorWithErr(updateErr, fmt.Sprintf("Failed to update link %d status", vl.link.ID))
			}
		}
		return nil, len(links), fmt.Errorf("AI link insertion failed: %w", err)
	}

	// Log AI usage
	if a.aiUsageService != nil {
		if logErr := a.aiUsageService.LogFromResult(
			ctx,
			siteID,
			aiusage.OpLinkInsertion,
			aiClient,
			insertResult.Usage,
			0,
			nil,
			map[string]interface{}{
				"source_node_id": sourceNode.ID,
				"links_count":    len(linkTargets),
				"links_applied":  insertResult.LinksApplied,
			},
		); logErr != nil {
			a.logger.ErrorWithErr(logErr, "Failed to log AI usage")
		}
	}

	if insertResult.LinksApplied == 0 {
		a.logger.Warnf("AI did not insert any links for node %d", sourceNode.ID)
		for _, vl := range validLinks {
			if updateErr := a.linkRepo.UpdateStatus(ctx, vl.link.ID, LinkStatusApproved, nil); updateErr != nil {
				a.logger.ErrorWithErr(updateErr, fmt.Sprintf("Failed to reset link %d status to approved", vl.link.ID))
			}
		}
		return nil, failedCount, nil
	}

	// Update WordPress page with new content
	page.Content = insertResult.Content
	if err := a.wpClient.UpdatePage(ctx, site, page); err != nil {
		errMsg := fmt.Sprintf("failed to update WordPress: %v", err)
		for _, vl := range validLinks {
			if updateErr := a.linkRepo.UpdateStatus(ctx, vl.link.ID, LinkStatusFailed, &errMsg); updateErr != nil {
				a.logger.ErrorWithErr(updateErr, fmt.Sprintf("Failed to update link %d status", vl.link.ID))
			}
		}
		return nil, len(links), fmt.Errorf("failed to update WordPress page: %w", err)
	}

	a.logger.Infof("Successfully applied %d/%d links to page %s", insertResult.LinksApplied, len(validLinks), sourceNode.Title)

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
			if updateErr := a.linkRepo.Update(ctx, vl.link); updateErr != nil {
				a.logger.ErrorWithErr(updateErr, fmt.Sprintf("Failed to update link %d", vl.link.ID))
			}

			anchor := ""
			if vl.link.AnchorText != nil {
				anchor = *vl.link.AnchorText
			}
			appliedInfos = append(appliedInfos, &AppliedLinkInfo{
				LinkID:     vl.link.ID,
				AnchorText: anchor,
			})
		} else {
			if updateErr := a.linkRepo.UpdateStatus(ctx, vl.link.ID, LinkStatusApproved, nil); updateErr != nil {
				a.logger.ErrorWithErr(updateErr, fmt.Sprintf("Failed to reset link %d status to approved", vl.link.ID))
			}
		}
	}

	return appliedInfos, failedCount, nil
}

// getApplyPrompt fetches the prompt for link insertion
// Returns the prompt entity, or nil if using defaults
func (a *Applier) getApplyPrompt(ctx context.Context, promptID *int64) *entities.Prompt {
	// If a specific prompt ID is provided, use it
	if promptID != nil && *promptID > 0 {
		prompt, err := a.promptSvc.GetPrompt(ctx, *promptID)
		if err == nil {
			return prompt
		}
		a.logger.ErrorWithErr(err, "Failed to get custom prompt, falling back to builtin")
	}

	// Otherwise, get the first builtin prompt for link_apply category
	prompts, err := a.promptSvc.ListPromptsByCategory(ctx, entities.PromptCategoryLinkApply)
	if err != nil {
		a.logger.ErrorWithErr(err, "Failed to get builtin link_apply prompts")
		return nil
	}

	for _, p := range prompts {
		if p.IsBuiltin {
			return p
		}
	}

	return nil
}

// buildApplyPrompts renders the prompt with page-specific placeholders
func (a *Applier) buildApplyPrompts(
	prompt *entities.Prompt,
	sourceNode *entities.SitemapNode,
	linkTargets []ai.InsertLinkTarget,
	content, language string,
) (systemPrompt, userPrompt string) {
	if prompt == nil {
		return "", "" // Use defaults in AI client
	}

	// Build links list
	var linksList strings.Builder
	for _, link := range linkTargets {
		anchor := "(auto)"
		if link.AnchorText != nil && *link.AnchorText != "" {
			anchor = *link.AnchorText
		}
		linksList.WriteString(fmt.Sprintf("→ %s \"%s\" (anchor: %s)\n", link.TargetPath, link.TargetTitle, anchor))
	}

	placeholders := map[string]string{
		"language":   language,
		"page_title": sourceNode.Title,
		"page_path":  sourceNode.Path,
		"links_list": linksList.String(),
		"content":    content,
	}

	// Render prompts using prompt service's template rendering
	systemPrompt = a.renderTemplate(prompt.SystemPrompt, placeholders)
	userPrompt = a.renderTemplate(prompt.UserPrompt, placeholders)

	return systemPrompt, userPrompt
}

// renderTemplate replaces {{placeholder}} with values
func (a *Applier) renderTemplate(template string, placeholders map[string]string) string {
	result := template
	for key, value := range placeholders {
		result = strings.ReplaceAll(result, "{{"+key+"}}", value)
	}
	return result
}
