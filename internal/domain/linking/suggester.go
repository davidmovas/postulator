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

const maxNodesPerBatch = 100 // Limit to prevent token overflow in AI response

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

		s.emitter.EmitSuggestProgress(ctx, taskID, batchIdx+1, totalBatches, processedNodes, totalNodes, totalLinksCreated, len(batchNodes))

		placeholders := s.buildPlaceholders(config, batchNodes, outgoingCount, incomingCount)
		systemPrompt, userPrompt := s.buildPrompts(ctx, config, batchNodes, outgoingCount, incomingCount)

		request := &ai.LinkSuggestionRequest{
			Nodes:         s.buildAINodes(batchNodes, outgoingCount, incomingCount, config.ExistingLinks),
			NodesInfo:     placeholders["nodes_info"],
			HierarchyTree: placeholders["hierarchyTree"],
			SystemPrompt:  systemPrompt,
			UserPrompt:    userPrompt,
			MaxIncoming:   config.MaxIncoming,
			MaxOutgoing:   config.MaxOutgoing,
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
	runtimeData := s.buildPlaceholders(config, nodes, outgoing, incoming)

	// Build context config overrides from SuggestConfig
	overrides := s.configToOverrides(config)

	// If a custom prompt ID is specified, use it
	if config.PromptID != nil && *config.PromptID > 0 {
		prompt, err := s.promptSvc.GetPrompt(ctx, *config.PromptID)
		if err == nil {
			sys, usr, err := s.promptSvc.RenderPromptWithOverrides(ctx, prompt, runtimeData, overrides)
			if err == nil {
				return sys, usr
			}
		}
		s.logger.Warn(fmt.Sprintf("Failed to render custom prompt, trying builtin: %v", err))
	}

	// Get the builtin prompt for link_suggest category
	promptsByCategory, err := s.promptSvc.ListPromptsByCategory(ctx, entities.PromptCategoryLinkSuggest)
	if err != nil {
		s.logger.Warn(fmt.Sprintf("Failed to get prompts for link_suggest category: %v", err))
		return "", ""
	}

	for _, p := range promptsByCategory {
		if p.IsBuiltin {
			sys, usr, err := s.promptSvc.RenderPromptWithOverrides(ctx, p, runtimeData, overrides)
			if err == nil {
				return sys, usr
			}
			s.logger.Warn(fmt.Sprintf("Failed to render builtin prompt: %v", err))
		}
	}

	s.logger.Warn("No builtin prompt found for link_suggest category")
	return "", ""
}

// configToOverrides converts SuggestConfig to ContextConfig overrides
// NOTE: feedback is passed via RuntimeData, not overrides (it's a runtime-only field)
// NOTE: maxIncoming/maxOutgoing are NOT added here - they're already in constraints placeholder
func (s *Suggester) configToOverrides(config SuggestConfig) entities.ContextConfig {
	overrides := make(entities.ContextConfig)
	// Constraints field handles max incoming/outgoing via placeholder
	// No need to add separate fields that would duplicate the info
	return overrides
}

func (s *Suggester) buildPlaceholders(config SuggestConfig, nodes []*entities.SitemapNode, outgoing, incoming map[int64]int) map[string]string {
	nodesInfo := s.buildNodesInfoList(nodes, outgoing, incoming)
	hierarchyTree := s.buildHierarchyTree(nodes, outgoing, incoming)

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
		"nodes_info":     nodesInfo,
		"hierarchyTree":  hierarchyTree,
		"constraints":    constraints.String(),
		"feedback":       feedback,
		"max_outgoing":   fmt.Sprintf("%d", config.MaxOutgoing),
		"max_incoming":   fmt.Sprintf("%d", config.MaxIncoming),
		"existing_links": fmt.Sprintf("%d", len(config.ExistingLinks)),
	}

	return placeholders
}

func (s *Suggester) buildNodesInfoList(nodes []*entities.SitemapNode, outgoing, incoming map[int64]int) string {
	var sb strings.Builder
	sb.WriteString("PAGES TO LINK:\n\n")

	for _, node := range nodes {
		sb.WriteString(fmt.Sprintf("[ID:%d] %s (/%s)\n", node.ID, node.Title, node.Slug))

		if len(node.Keywords) > 0 {
			kw := node.Keywords
			if len(kw) > 5 {
				kw = kw[:5]
			}
			sb.WriteString(fmt.Sprintf("  Keywords: %s\n", strings.Join(kw, ", ")))
		}

		out := outgoing[node.ID]
		in := incoming[node.ID]
		sb.WriteString(fmt.Sprintf("  Current links: %d out, %d in\n\n", out, in))
	}

	return sb.String()
}

func (s *Suggester) buildHierarchyTree(nodes []*entities.SitemapNode, outgoing, incoming map[int64]int) string {
	if len(nodes) == 0 {
		return ""
	}

	// Build a map of nodes by ID for quick lookup (only nodes in batch)
	nodeMap := make(map[int64]*entities.SitemapNode)
	for _, node := range nodes {
		nodeMap[node.ID] = node
	}

	// Build children map - only for nodes within this batch
	childrenMap := make(map[int64][]*entities.SitemapNode)
	var roots []*entities.SitemapNode

	for _, node := range nodes {
		if node.ParentID == nil || nodeMap[*node.ParentID] == nil {
			// No parent or parent not in current batch - treat as root
			roots = append(roots, node)
		} else {
			childrenMap[*node.ParentID] = append(childrenMap[*node.ParentID], node)
		}
	}

	// Sort roots by position
	sortNodesByPosition(roots)

	// Sort all children by position
	for parentID := range childrenMap {
		sortNodesByPosition(childrenMap[parentID])
	}

	// Build the tree string
	var sb strings.Builder
	sb.WriteString("SITE STRUCTURE:\n")

	for i, root := range roots {
		isLast := i == len(roots)-1
		s.writeCompactTreeNode(&sb, root, "", isLast, childrenMap, outgoing, incoming)
	}

	return sb.String()
}

func sortNodesByPosition(nodes []*entities.SitemapNode) {
	for i := 0; i < len(nodes)-1; i++ {
		for j := i + 1; j < len(nodes); j++ {
			if nodes[i].Position > nodes[j].Position {
				nodes[i], nodes[j] = nodes[j], nodes[i]
			}
		}
	}
}

// writeCompactTreeNode writes a single node in compact tree format
func (s *Suggester) writeCompactTreeNode(
	sb *strings.Builder,
	node *entities.SitemapNode,
	prefix string,
	isLast bool,
	childrenMap map[int64][]*entities.SitemapNode,
	outgoing, incoming map[int64]int,
) {
	// Choose the appropriate connector
	connector := "├── "
	if isLast {
		connector = "└── "
	}

	sb.WriteString(prefix)
	sb.WriteString(connector)
	sb.WriteString(fmt.Sprintf("[ID:%d] \"%s\" /%s", node.ID, node.Title, node.Slug))

	// Add keywords if present (limit to 3)
	if len(node.Keywords) > 0 {
		kw := node.Keywords
		if len(kw) > 3 {
			kw = kw[:3]
		}
		sb.WriteString(fmt.Sprintf(" [kw: %s]", strings.Join(kw, ", ")))
	}

	sb.WriteString(fmt.Sprintf(" (%d out, %d in)", outgoing[node.ID], incoming[node.ID]))
	sb.WriteString("\n")

	// Process children
	children := childrenMap[node.ID]
	if len(children) > 0 {
		// Choose the appropriate prefix for children
		childPrefix := prefix
		if isLast {
			childPrefix += "    "
		} else {
			childPrefix += "│   "
		}

		for i, child := range children {
			childIsLast := i == len(children)-1
			s.writeCompactTreeNode(sb, child, childPrefix, childIsLast, childrenMap, outgoing, incoming)
		}
	}
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
