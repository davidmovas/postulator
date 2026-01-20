package linking

import (
	"context"
	"fmt"
	"sort"
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
	"golang.org/x/net/html"
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

type linkWithTarget struct {
	link   *PlannedLink
	target ai.InsertLinkTarget
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

func (a *Applier) estimateInsertBatchSize(contentLen int, linksCount int, maxOutputTokens int) int {
	estimatedContentTokens := (contentLen * 10) / 35

	jsonOverhead := 2000
	safetyBuffer := int(float64(estimatedContentTokens) * 0.3)

	availableForContent := maxOutputTokens - jsonOverhead - safetyBuffer

	if estimatedContentTokens > availableForContent/2 {
		tokensPerLink := 100
		maxLinks := (availableForContent - estimatedContentTokens) / tokensPerLink
		if maxLinks < 1 {
			maxLinks = 1
		}
		if maxLinks > linksCount {
			maxLinks = linksCount
		}
		return maxLinks
	}

	return linksCount
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

	page, err := a.wpClient.GetPage(ctx, site, *sourceNode.WPPageID)
	if err != nil {
		return nil, len(links), fmt.Errorf("failed to get WordPress page: %w", err)
	}

	if page.Content == "" {
		return nil, len(links), fmt.Errorf("page has no content")
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

	sort.Slice(validLinks, func(i, j int) bool {
		ci := float64(0)
		cj := float64(0)
		if validLinks[i].link.Confidence != nil {
			ci = *validLinks[i].link.Confidence
		}
		if validLinks[j].link.Confidence != nil {
			cj = *validLinks[j].link.Confidence
		}
		return ci > cj
	})

	modelInfo := ai.GetModelInfo(entities.TypeOpenAI, aiClient.GetModelName())
	maxOutputTokens := 128000
	if modelInfo != nil && modelInfo.MaxOutputTokens > 0 {
		maxOutputTokens = modelInfo.MaxOutputTokens
	}

	batchSize := a.estimateInsertBatchSize(len(page.Content), len(validLinks), maxOutputTokens)

	allAppliedInfos := make([]*AppliedLinkInfo, 0)
	currentContent := page.Content
	totalBatches := (len(validLinks) + batchSize - 1) / batchSize

	for batchStart := 0; batchStart < len(validLinks); batchStart += batchSize {
		batchEnd := batchStart + batchSize
		if batchEnd > len(validLinks) {
			batchEnd = len(validLinks)
		}
		batchLinks := validLinks[batchStart:batchEnd]
		batchNum := (batchStart / batchSize) + 1

		for _, vl := range batchLinks {
			if updateErr := a.linkRepo.UpdateStatus(ctx, vl.link.ID, LinkStatusApplying, nil); updateErr != nil {
				a.logger.ErrorWithErr(updateErr, fmt.Sprintf("Failed to update link %d status to applying", vl.link.ID))
			}
		}

		linkTargets := make([]ai.InsertLinkTarget, len(batchLinks))
		for i, vl := range batchLinks {
			linkTargets[i] = vl.target
		}

		systemPrompt, userPrompt := a.buildApplyPrompts(ctx, prompt, sourceNode, linkTargets, currentContent, DefaultLanguage)

		request := &ai.InsertLinksRequest{
			Content:      currentContent,
			PageTitle:    sourceNode.Title,
			PagePath:     sourceNode.Path,
			Links:        linkTargets,
			Language:     DefaultLanguage,
			SystemPrompt: systemPrompt,
			UserPrompt:   userPrompt,
		}

		if totalBatches > 1 {
			a.logger.Infof("Inserting batch %d/%d (%d links) into page %s", batchNum, totalBatches, len(linkTargets), sourceNode.Title)
		} else {
			a.logger.Infof("Inserting %d links into page %s", len(linkTargets), sourceNode.Title)
		}

		insertResult, err := aiClient.InsertLinks(ctx, request)
		if err != nil {
			errMsg := fmt.Sprintf("AI insertion failed: %v", err)
			for _, vl := range batchLinks {
				if updateErr := a.linkRepo.UpdateStatus(ctx, vl.link.ID, LinkStatusFailed, &errMsg); updateErr != nil {
					a.logger.ErrorWithErr(updateErr, fmt.Sprintf("Failed to update link %d status", vl.link.ID))
				}
			}
			failedCount += len(batchLinks)
			continue
		}

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
					"batch_num":      batchNum,
					"total_batches":  totalBatches,
				},
			); logErr != nil {
				a.logger.ErrorWithErr(logErr, "Failed to log AI usage")
			}
		}

		linksVerified := a.verifyLinksInserted(insertResult.Content, batchLinks)

		now := time.Now()
		batchApplied := 0

		for i, vl := range batchLinks {
			linkInserted := linksVerified[i]

			if linkInserted {
				vl.link.Status = LinkStatusApplied
				vl.link.AppliedAt = &now
				if updateErr := a.linkRepo.Update(ctx, vl.link); updateErr != nil {
					a.logger.ErrorWithErr(updateErr, fmt.Sprintf("Failed to update link %d", vl.link.ID))
				}

				anchor := ""
				if vl.link.AnchorText != nil {
					anchor = *vl.link.AnchorText
				}
				allAppliedInfos = append(allAppliedInfos, &AppliedLinkInfo{
					LinkID:     vl.link.ID,
					AnchorText: anchor,
				})
				batchApplied++
			} else {
				a.logger.Warnf("Link %d was not inserted (anchor: %s, target: %s)",
					vl.link.ID,
					getAnchorOrEmpty(vl.link.AnchorText),
					vl.target.TargetPath)

				if updateErr := a.linkRepo.UpdateStatus(ctx, vl.link.ID, LinkStatusApproved, nil); updateErr != nil {
					a.logger.ErrorWithErr(updateErr, fmt.Sprintf("Failed to reset link %d status to approved", vl.link.ID))
				}
			}
		}

		if batchApplied > 0 {
			currentContent = insertResult.Content
		}

		a.logger.Infof("Batch %d: applied %d/%d links", batchNum, batchApplied, len(batchLinks))
	}

	if len(allAppliedInfos) > 0 {
		page.Content = currentContent
		if err := a.wpClient.UpdatePage(ctx, site, page); err != nil {
			errMsg := fmt.Sprintf("failed to update WordPress: %v", err)
			for _, info := range allAppliedInfos {
				if updateErr := a.linkRepo.UpdateStatus(ctx, info.LinkID, LinkStatusFailed, &errMsg); updateErr != nil {
					a.logger.ErrorWithErr(updateErr, fmt.Sprintf("Failed to update link %d status", info.LinkID))
				}
			}
			return nil, len(links), fmt.Errorf("failed to update WordPress page: %w", err)
		}
	}

	a.logger.Infof("Successfully applied %d/%d links to page %s", len(allAppliedInfos), len(validLinks), sourceNode.Title)

	return allAppliedInfos, failedCount + (len(validLinks) - len(allAppliedInfos)), nil
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
	ctx context.Context,
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

	runtimeData := map[string]string{
		"language":   language,
		"page_title": sourceNode.Title,
		"page_path":  sourceNode.Path,
		"links_list": linksList.String(),
		"content":    content,
	}

	// Use the prompt service to render (supports both v1 and v2)
	sys, usr, err := a.promptSvc.RenderPromptWithOverrides(ctx, prompt, runtimeData, nil)
	if err != nil {
		a.logger.ErrorWithErr(err, "Failed to render prompt, using empty defaults")
		return "", ""
	}

	return sys, usr
}

func (a *Applier) verifyLinksInserted(content string, expectedLinks []linkWithTarget) []bool {
	inserted := make([]bool, len(expectedLinks))

	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		a.logger.ErrorWithErr(err, "Failed to parse HTML for verification")
		return inserted
	}

	foundLinks := extractAllLinks(doc)

	for i, vl := range expectedLinks {
		targetPath := vl.target.TargetPath
		for _, link := range foundLinks {
			if strings.Contains(link.Href, targetPath) {
				inserted[i] = true
				break
			}
		}
	}

	return inserted
}

type linkInfo struct {
	Href string
	Text string
}

func extractAllLinks(n *html.Node) []linkInfo {
	var links []linkInfo

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					text := getTextContent(n)
					links = append(links, linkInfo{
						Href: attr.Val,
						Text: text,
					})
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)

	return links
}

func getTextContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var text string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text += getTextContent(c)
	}
	return text
}

func getAnchorOrEmpty(anchor *string) string {
	if anchor == nil {
		return ""
	}
	return *anchor
}
