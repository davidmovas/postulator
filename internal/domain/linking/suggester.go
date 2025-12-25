package linking

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/aiusage"
	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/prompts"
	"github.com/davidmovas/postulator/internal/domain/providers"
	"github.com/davidmovas/postulator/internal/domain/sitemap"
	"github.com/davidmovas/postulator/internal/infra/ai"
	"github.com/davidmovas/postulator/internal/infra/events"
	"github.com/davidmovas/postulator/pkg/logger"
	"github.com/google/uuid"
)

type Suggester struct {
	sitemapSvc     sitemap.Service
	providerSvc    providers.Service
	promptSvc      prompts.Service
	linkRepo       LinkRepository
	aiUsageService aiusage.Service
	eventBus       *events.EventBus
	emitter        *SuggestEventEmitter
	logger         *logger.Logger
}

func NewSuggester(
	sitemapSvc sitemap.Service,
	providerSvc providers.Service,
	promptSvc prompts.Service,
	linkRepo LinkRepository,
	aiUsageService aiusage.Service,
	eventBus *events.EventBus,
	logger *logger.Logger,
) *Suggester {
	return &Suggester{
		sitemapSvc:     sitemapSvc,
		providerSvc:    providerSvc,
		promptSvc:      promptSvc,
		linkRepo:       linkRepo,
		aiUsageService: aiUsageService,
		eventBus:       eventBus,
		emitter:        NewSuggestEventEmitter(eventBus),
		logger:         logger.WithScope("linking.suggester"),
	}
}

type SuggestConfig struct {
	PlanID        int64
	SitemapID     int64
	SiteID        int64
	ProviderID    int64
	PromptID      *int64
	NodeIDs       []int64
	Feedback      string
	MaxIncoming   int
	MaxOutgoing   int
	ExistingLinks []*PlannedLink
}

type SuggestResult struct {
	LinksCreated int
	Explanation  string
}

const maxNodesPerBatch = 15 // Limit to prevent token overflow in AI response

func (s *Suggester) Suggest(ctx context.Context, config SuggestConfig) (*SuggestResult, error) {
	startTime := time.Now()
	taskID := uuid.New().String()

	provider, err := s.providerSvc.GetProvider(ctx, config.ProviderID)
	if err != nil {
		s.emitter.EmitSuggestFailed(ctx, taskID, err.Error())
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	if !provider.IsActive {
		s.emitter.EmitSuggestFailed(ctx, taskID, "provider is not active")
		return nil, fmt.Errorf("provider is not active")
	}

	aiClient, err := ai.CreateClient(provider)
	if err != nil {
		s.emitter.EmitSuggestFailed(ctx, taskID, err.Error())
		return nil, fmt.Errorf("failed to create AI client: %w", err)
	}

	_, nodes, err := s.sitemapSvc.GetSitemapWithNodes(ctx, config.SitemapID)
	if err != nil {
		s.emitter.EmitSuggestFailed(ctx, taskID, err.Error())
		return nil, fmt.Errorf("failed to get sitemap nodes: %w", err)
	}

	filteredNodes := nodes
	if len(config.NodeIDs) > 0 {
		nodeIDSet := make(map[int64]bool)
		for _, id := range config.NodeIDs {
			nodeIDSet[id] = true
		}
		filteredNodes = make([]*entities.SitemapNode, 0)
		for _, node := range nodes {
			if nodeIDSet[node.ID] {
				filteredNodes = append(filteredNodes, node)
			}
		}
	}

	if len(filteredNodes) < 2 {
		s.emitter.EmitSuggestFailed(ctx, taskID, "need at least 2 nodes to suggest links")
		return nil, fmt.Errorf("need at least 2 nodes to suggest links")
	}

	// Track link counts across all batches
	outgoingCount := make(map[int64]int)
	incomingCount := make(map[int64]int)
	for _, link := range config.ExistingLinks {
		outgoingCount[link.SourceNodeID]++
		incomingCount[link.TargetNodeID]++
	}

	// Build node ID set for validation
	nodeIDSet := make(map[int64]bool)
	for _, node := range filteredNodes {
		nodeIDSet[node.ID] = true
	}

	// Split into batches if too many nodes
	batches := s.splitIntoBatches(filteredNodes, maxNodesPerBatch)
	totalBatches := len(batches)
	totalNodes := len(filteredNodes)

	s.logger.Infof("Processing %d nodes in %d batch(es)", totalNodes, totalBatches)

	// Emit started event
	s.emitter.EmitSuggestStarted(ctx, taskID, totalNodes, totalBatches)

	totalLinksCreated := 0
	processedNodes := 0
	var lastExplanation string

	for batchIdx, batchNodes := range batches {
		// Check for cancellation before processing each batch
		select {
		case <-ctx.Done():
			s.logger.Infof("Suggest operation cancelled at batch %d/%d", batchIdx+1, totalBatches)
			s.emitter.EmitSuggestCancelled(ctx, taskID, processedNodes, totalLinksCreated)
			return &SuggestResult{
				LinksCreated: totalLinksCreated,
				Explanation:  "Operation cancelled",
			}, ctx.Err()
		default:
		}

		if len(batchNodes) < 2 {
			processedNodes += len(batchNodes)
			continue
		}

		s.logger.Infof("Processing batch %d/%d with %d nodes", batchIdx+1, totalBatches, len(batchNodes))

		// Emit progress before processing batch
		s.emitter.EmitSuggestProgress(ctx, taskID, batchIdx+1, totalBatches, processedNodes, totalNodes, totalLinksCreated, len(batchNodes))

		systemPrompt, userPrompt := s.buildPrompts(ctx, config, batchNodes, outgoingCount, incomingCount)

		request := &ai.LinkSuggestionRequest{
			Nodes:        s.buildAINodes(batchNodes, outgoingCount, incomingCount, config.ExistingLinks),
			SystemPrompt: systemPrompt,
			UserPrompt:   userPrompt,
			MaxIncoming:  config.MaxIncoming,
			MaxOutgoing:  config.MaxOutgoing,
		}

		result, err := aiClient.GenerateLinkSuggestions(ctx, request)
		if err != nil {
			s.logger.ErrorWithErr(err, fmt.Sprintf("AI link suggestion failed for batch %d", batchIdx+1))
			processedNodes += len(batchNodes)
			// Continue with next batch instead of failing entirely
			continue
		}

		s.logger.Infof("Batch %d: AI returned %d link suggestions", batchIdx+1, len(result.Links))
		lastExplanation = result.Explanation

		if s.aiUsageService != nil {
			_ = s.aiUsageService.LogFromResult(
				ctx,
				config.SiteID,
				aiusage.OpLinkSuggestion,
				aiClient,
				result.Usage,
				0,
				nil,
				map[string]interface{}{
					"plan_id":       config.PlanID,
					"sitemap_id":    config.SitemapID,
					"nodes_count":   len(batchNodes),
					"batch":         batchIdx + 1,
					"total_batches": totalBatches,
				},
			)
		}

		// Process links from this batch
		linksCreated := s.processLinks(ctx, config, result.Links, nodeIDSet, outgoingCount, incomingCount)
		totalLinksCreated += linksCreated
		processedNodes += len(batchNodes)

		// Emit progress after processing batch
		s.emitter.EmitSuggestProgress(ctx, taskID, batchIdx+1, totalBatches, processedNodes, totalNodes, totalLinksCreated, len(batchNodes))
	}

	s.logger.Infof("Created %d total link suggestions across %d batches", totalLinksCreated, totalBatches)

	// Emit completed event
	s.emitter.EmitSuggestCompleted(ctx, taskID, totalNodes, totalLinksCreated, startTime)

	return &SuggestResult{
		LinksCreated: totalLinksCreated,
		Explanation:  lastExplanation,
	}, nil
}

// splitIntoBatches divides nodes into batches of maxSize
func (s *Suggester) splitIntoBatches(nodes []*entities.SitemapNode, maxSize int) [][]*entities.SitemapNode {
	if len(nodes) <= maxSize {
		return [][]*entities.SitemapNode{nodes}
	}

	var batches [][]*entities.SitemapNode
	for i := 0; i < len(nodes); i += maxSize {
		end := i + maxSize
		if end > len(nodes) {
			end = len(nodes)
		}
		batches = append(batches, nodes[i:end])
	}
	return batches
}

// processLinks creates planned links from AI suggestions
func (s *Suggester) processLinks(
	ctx context.Context,
	config SuggestConfig,
	links []ai.SuggestedLink,
	nodeIDSet map[int64]bool,
	outgoingCount, incomingCount map[int64]int,
) int {
	linksCreated := 0

	for _, link := range links {
		if !nodeIDSet[link.SourceNodeID] || !nodeIDSet[link.TargetNodeID] {
			s.logger.Warnf("Skipping invalid link: source=%d target=%d", link.SourceNodeID, link.TargetNodeID)
			continue
		}
		if link.SourceNodeID == link.TargetNodeID {
			continue
		}

		if config.MaxOutgoing > 0 && outgoingCount[link.SourceNodeID] >= config.MaxOutgoing {
			continue
		}
		if config.MaxIncoming > 0 && incomingCount[link.TargetNodeID] >= config.MaxIncoming {
			continue
		}

		existing, err := s.linkRepo.GetByNodePair(ctx, config.PlanID, link.SourceNodeID, link.TargetNodeID)
		if err != nil {
			s.logger.ErrorWithErr(err, "Failed to check existing link")
			continue
		}
		if existing != nil {
			continue
		}

		anchorText := link.AnchorText
		confidence := link.Confidence

		plannedLink := &PlannedLink{
			PlanID:       config.PlanID,
			SourceNodeID: link.SourceNodeID,
			TargetNodeID: link.TargetNodeID,
			AnchorText:   &anchorText,
			Status:       LinkStatusPlanned,
			Source:       LinkSourceAI,
			Confidence:   &confidence,
		}

		if err := s.linkRepo.Create(ctx, plannedLink); err != nil {
			s.logger.ErrorWithErr(err, "Failed to create suggested link")
			continue
		}

		outgoingCount[link.SourceNodeID]++
		incomingCount[link.TargetNodeID]++
		linksCreated++
	}

	return linksCreated
}

func (s *Suggester) buildPrompts(ctx context.Context, config SuggestConfig, nodes []*entities.SitemapNode, outgoing, incoming map[int64]int) (system, user string) {
	placeholders := s.buildPlaceholders(config, nodes, outgoing, incoming)

	// If a custom prompt ID is specified, use it
	if config.PromptID != nil && *config.PromptID > 0 {
		sys, usr, err := s.promptSvc.RenderPrompt(ctx, *config.PromptID, placeholders)
		if err == nil {
			return sys, usr
		}
		s.logger.WarnWithErr(err, "Failed to render custom prompt, trying builtin")
	}

	// Try to get the builtin prompt for link_suggest category
	prompts, err := s.promptSvc.ListPromptsByCategory(ctx, entities.PromptCategoryLinkSuggest)
	if err == nil {
		for _, p := range prompts {
			if p.IsBuiltin {
				sys := s.renderTemplate(p.SystemPrompt, placeholders)
				usr := s.renderTemplate(p.UserPrompt, placeholders)
				return sys, usr
			}
		}
	}

	// Fall back to hardcoded default
	s.logger.Warn("No builtin prompt found for link_suggest, using hardcoded default")
	return s.buildDefaultPrompts(config, nodes, outgoing, incoming)
}

// renderTemplate replaces {{placeholder}} with values
func (s *Suggester) renderTemplate(template string, placeholders map[string]string) string {
	result := template
	for key, value := range placeholders {
		result = strings.ReplaceAll(result, "{{"+key+"}}", value)
	}
	return result
}

func (s *Suggester) buildPlaceholders(config SuggestConfig, nodes []*entities.SitemapNode, outgoing, incoming map[int64]int) map[string]string {
	var nodesInfo strings.Builder
	for _, node := range nodes {
		nodesInfo.WriteString(fmt.Sprintf("[ID:%d] \"%s\" /%s", node.ID, node.Title, node.Slug))
		if len(node.Keywords) > 0 {
			kw := node.Keywords
			if len(kw) > 5 {
				kw = kw[:5]
			}
			nodesInfo.WriteString(fmt.Sprintf(" | keywords: %s", strings.Join(kw, ", ")))
		}
		nodesInfo.WriteString(fmt.Sprintf(" | links: %d→ %d←\n", outgoing[node.ID], incoming[node.ID]))
	}

	// Build constraints section
	var constraints strings.Builder
	if config.MaxIncoming > 0 || config.MaxOutgoing > 0 {
		constraints.WriteString("\nCONSTRAINTS:\n")
		if config.MaxOutgoing > 0 {
			constraints.WriteString(fmt.Sprintf("- Max %d outgoing links per page\n", config.MaxOutgoing))
		}
		if config.MaxIncoming > 0 {
			constraints.WriteString(fmt.Sprintf("- Max %d incoming links per page\n", config.MaxIncoming))
		}
	}

	// Build feedback section
	feedback := ""
	if config.Feedback != "" {
		feedback = fmt.Sprintf("\nUSER INSTRUCTIONS: %s\n", config.Feedback)
	}

	placeholders := map[string]string{
		"nodes_count":    fmt.Sprintf("%d", len(nodes)),
		"nodes_info":     nodesInfo.String(),
		"constraints":    constraints.String(),
		"feedback":       feedback,
		"max_incoming":   fmt.Sprintf("%d", config.MaxIncoming),
		"max_outgoing":   fmt.Sprintf("%d", config.MaxOutgoing),
		"existing_links": fmt.Sprintf("%d", len(config.ExistingLinks)),
	}

	return placeholders
}

func (s *Suggester) buildDefaultPrompts(config SuggestConfig, nodes []*entities.SitemapNode, outgoing, incoming map[int64]int) (system, user string) {
	system = `You are an internal linking strategist for websites.

TASK: Suggest links between pages to improve site structure and SEO.

GOALS (priority order):
1. Connect semantically related pages (same topic, complementary content)
2. Link from high-content pages to low-visibility pages
3. Create logical navigation paths for users
4. Balance link distribution (avoid orphan pages with no links)

RULES:
- Only suggest NEW links (respect existing outgoing/incoming counts shown)
- One page should not link to another more than once
- Anchor text should describe the target page naturally, not generic like "click here"
- If anchor text is obvious from context, you can skip it

OUTPUT: Return suggested links with sourceId, targetId, and optional anchorText.`

	var sb strings.Builder
	sb.WriteString("PAGES:\n")

	for _, node := range nodes {
		sb.WriteString(fmt.Sprintf("[ID:%d] \"%s\" /%s", node.ID, node.Title, node.Slug))
		if len(node.Keywords) > 0 {
			kw := node.Keywords
			if len(kw) > 5 {
				kw = kw[:5]
			}
			sb.WriteString(fmt.Sprintf(" | keywords: %s", strings.Join(kw, ", ")))
		}
		sb.WriteString(fmt.Sprintf(" | links: %d→ %d←\n", outgoing[node.ID], incoming[node.ID]))
	}

	if config.MaxIncoming > 0 || config.MaxOutgoing > 0 {
		sb.WriteString("\nCONSTRAINTS:\n")
		if config.MaxOutgoing > 0 {
			sb.WriteString(fmt.Sprintf("- Max %d outgoing links per page\n", config.MaxOutgoing))
		}
		if config.MaxIncoming > 0 {
			sb.WriteString(fmt.Sprintf("- Max %d incoming links per page\n", config.MaxIncoming))
		}
	}

	if config.Feedback != "" {
		sb.WriteString(fmt.Sprintf("\nUSER INSTRUCTIONS: %s\n", config.Feedback))
	}

	sb.WriteString("\nSuggest links that make sense semantically. Use exact page IDs.")

	return system, sb.String()
}

func (s *Suggester) buildAINodes(nodes []*entities.SitemapNode, outgoing, incoming map[int64]int, existingLinks []*PlannedLink) []ai.LinkSuggestionNode {
	linksByNode := make(map[int64][]ai.ExistingLinkInfo)
	for _, link := range existingLinks {
		linksByNode[link.SourceNodeID] = append(linksByNode[link.SourceNodeID], ai.ExistingLinkInfo{
			TargetNodeID: link.TargetNodeID,
			Status:       string(link.Status),
			IsOutgoing:   true,
		})
		linksByNode[link.TargetNodeID] = append(linksByNode[link.TargetNodeID], ai.ExistingLinkInfo{
			TargetNodeID: link.SourceNodeID,
			Status:       string(link.Status),
			IsOutgoing:   false,
		})
	}

	aiNodes := make([]ai.LinkSuggestionNode, len(nodes))
	for i, node := range nodes {
		aiNodes[i] = ai.LinkSuggestionNode{
			ID:            node.ID,
			Title:         node.Title,
			Path:          "/" + node.Slug,
			Keywords:      node.Keywords,
			OutgoingCount: outgoing[node.ID],
			IncomingCount: incoming[node.ID],
			ExistingLinks: linksByNode[node.ID],
		}
	}

	return aiNodes
}
