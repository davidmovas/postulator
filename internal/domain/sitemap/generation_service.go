package sitemap

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/aiusage"
	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/prompts"
	"github.com/davidmovas/postulator/internal/domain/providers"
	"github.com/davidmovas/postulator/internal/domain/sites"
	"github.com/davidmovas/postulator/internal/infra/ai"
	"github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"
	"github.com/gosimple/slug"
)

// TitleInput represents a single title with optional keywords for AI generation
type TitleInput struct {
	Title    string
	Keywords []string
}

// GenerationInput contains all parameters for sitemap structure generation
type GenerationInput struct {
	// SitemapID is set when adding to existing sitemap, 0 for new sitemap
	SitemapID int64
	// SiteID is required when creating new sitemap
	SiteID int64
	// Name for new sitemap (required when SitemapID is 0)
	Name string
	// PromptID is the ID of the prompt template to use
	PromptID int64
	// Placeholders for the prompt template
	Placeholders map[string]string
	// Titles with optional keywords to generate structure for
	Titles []TitleInput
	// ParentNodeIDs - nodes to use as roots for new nodes (empty = use root node)
	ParentNodeIDs []int64
	// MaxDepth limits the depth of generated structure (0 = no limit)
	MaxDepth int
	// IncludeExistingTree sends the current tree structure to AI for context
	IncludeExistingTree bool
	// ProviderID is the AI provider to use
	ProviderID int64
}

// GenerationResult contains the result of sitemap structure generation
type GenerationResult struct {
	SitemapID    int64
	NodesCreated int
	DurationMs   int64
}

// GeneratedNodeInfo represents a node in the generated structure (for response)
type GeneratedNodeInfo struct {
	Title    string
	Slug     string
	Keywords []string
	Children []GeneratedNodeInfo
}

// GenerationService handles AI-based sitemap structure generation
type GenerationService struct {
	sitemapSvc      Service
	sitesSvc        sites.Service
	promptSvc       prompts.Service
	providerSvc     providers.Service
	aiUsageSvc      aiusage.Service
	aiClientFactory func(provider *entities.Provider) (ai.Client, error)
	logger          *logger.Logger
}

// NewGenerationService creates a new generation service
func NewGenerationService(
	sitemapSvc Service,
	sitesSvc sites.Service,
	promptSvc prompts.Service,
	providerSvc providers.Service,
	aiUsageSvc aiusage.Service,
	aiClientFactory func(provider *entities.Provider) (ai.Client, error),
	logger *logger.Logger,
) *GenerationService {
	return &GenerationService{
		sitemapSvc:      sitemapSvc,
		sitesSvc:        sitesSvc,
		promptSvc:       promptSvc,
		providerSvc:     providerSvc,
		aiUsageSvc:      aiUsageSvc,
		aiClientFactory: aiClientFactory,
		logger:          logger.WithScope("generation_service"),
	}
}

// GenerateStructure generates sitemap structure using AI
func (s *GenerationService) GenerateStructure(ctx context.Context, input GenerationInput) (*GenerationResult, error) {
	return s.GenerateStructureWithTracking(ctx, input, nil)
}

// GenerateStructureWithTracking generates sitemap structure using AI and calls onNodeCreated for each created node
func (s *GenerationService) GenerateStructureWithTracking(ctx context.Context, input GenerationInput, onNodeCreated func(nodeID int64)) (*GenerationResult, error) {
	startTime := time.Now()

	s.logger.Infof("Starting sitemap generation: siteID=%d, sitemapID=%d, titles=%d",
		input.SiteID, input.SitemapID, len(input.Titles))

	// Validate input
	if err := s.validateInput(input); err != nil {
		s.logger.ErrorWithErr(err, "Validation failed")
		return nil, err
	}

	// Get site URL for root node (when creating new sitemap)
	var siteURL string
	if input.SitemapID == 0 && input.SiteID > 0 {
		site, err := s.sitesSvc.GetSite(ctx, input.SiteID)
		if err != nil {
			s.logger.ErrorWithErr(err, "Failed to get site")
			return nil, fmt.Errorf("failed to get site: %w", err)
		}
		siteURL = site.URL
	}

	// Get provider
	provider, err := s.providerSvc.GetProvider(ctx, input.ProviderID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get provider")
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	// Create AI client
	aiClient, err := s.aiClientFactory(provider)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to create AI client")
		return nil, fmt.Errorf("failed to create AI client: %w", err)
	}

	// Build prompts BEFORE creating sitemap (for existing sitemap context)
	systemPrompt, userPrompt, err := s.buildPromptsForGeneration(ctx, input)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to build prompts")
		return nil, fmt.Errorf("failed to build prompts: %w", err)
	}

	s.logger.Debugf("Generating sitemap structure with prompt length: system=%d, user=%d", len(systemPrompt), len(userPrompt))

	// Call AI FIRST - before creating any sitemap
	aiStartTime := time.Now()
	aiResult, aiErr := aiClient.GenerateSitemapStructure(ctx, systemPrompt, userPrompt)
	aiDurationMs := time.Since(aiStartTime).Milliseconds()

	// Log AI usage regardless of success/failure
	if s.aiUsageSvc != nil {
		var usage ai.Usage
		if aiResult != nil {
			usage = aiResult.Usage
		}
		_ = s.aiUsageSvc.LogFromResult(
			ctx,
			input.SiteID,
			aiusage.OperationSitemapGeneration,
			aiClient,
			usage,
			aiDurationMs,
			aiErr,
			map[string]interface{}{
				"sitemap_id":   input.SitemapID,
				"titles_count": len(input.Titles),
			},
		)
	}

	if aiErr != nil {
		// Check if this was a cancellation - not an error
		if ctx.Err() == context.Canceled {
			s.logger.Info("AI generation was cancelled by user")
			return nil, errors.New(errors.ErrCodeValidation, "Generation was cancelled")
		}
		s.logger.ErrorWithErr(aiErr, "AI generation failed")
		return nil, fmt.Errorf("AI generation failed: %w", aiErr)
	}

	if len(aiResult.Nodes) == 0 {
		s.logger.Error("AI generated no nodes")
		return nil, errors.Validation("AI generated no nodes")
	}

	s.logger.Infof("AI generated %d top-level nodes", len(aiResult.Nodes))

	// Now create sitemap if needed (only after successful AI generation)
	sitemapID := input.SitemapID
	var rootNodeID int64

	if sitemapID == 0 {
		// Create new sitemap with site URL as root node title
		sitemap := &entities.Sitemap{
			SiteID: input.SiteID,
			Name:   input.Name,
			Source: entities.SitemapSourceGenerated,
			Status: entities.SitemapStatusDraft,
		}
		if err = s.sitemapSvc.CreateSitemapWithRoot(ctx, sitemap, siteURL); err != nil {
			s.logger.ErrorWithErr(err, "Failed to create sitemap")
			return nil, fmt.Errorf("failed to create sitemap: %w", err)
		}
		sitemapID = sitemap.ID
		s.logger.Infof("Created new sitemap: id=%d, name=%s", sitemapID, input.Name)

		// Get root node
		nodes, err := s.sitemapSvc.GetNodes(ctx, sitemapID)
		if err != nil {
			s.logger.ErrorWithErr(err, "Failed to get root node")
			return nil, fmt.Errorf("failed to get root node: %w", err)
		}
		for _, node := range nodes {
			if node.IsRoot {
				rootNodeID = node.ID
				break
			}
		}
	} else {
		// Get root node from existing sitemap
		nodes, err := s.sitemapSvc.GetNodes(ctx, sitemapID)
		if err != nil {
			s.logger.ErrorWithErr(err, "Failed to get nodes")
			return nil, fmt.Errorf("failed to get nodes: %w", err)
		}
		for _, node := range nodes {
			if node.IsRoot {
				rootNodeID = node.ID
				break
			}
		}
	}

	// Determine parent nodes
	parentNodeIDs := input.ParentNodeIDs
	if len(parentNodeIDs) == 0 {
		parentNodeIDs = []int64{rootNodeID}
	}

	// Create nodes from AI response
	nodesCreated, err := s.createNodesFromAIResponse(ctx, sitemapID, parentNodeIDs, aiResult.Nodes, input.MaxDepth, onNodeCreated)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to create nodes from AI response")
		return nil, fmt.Errorf("failed to create nodes: %w", err)
	}

	durationMs := time.Since(startTime).Milliseconds()
	s.logger.Infof("Sitemap generation completed: sitemapID=%d, nodesCreated=%d, duration=%dms",
		sitemapID, nodesCreated, durationMs)

	return &GenerationResult{
		SitemapID:    sitemapID,
		NodesCreated: nodesCreated,
		DurationMs:   durationMs,
	}, nil
}

func (s *GenerationService) validateInput(input GenerationInput) error {
	if input.SitemapID == 0 && input.SiteID == 0 {
		return errors.Validation("Either sitemapId or siteId must be provided")
	}
	if input.SitemapID == 0 && input.Name == "" {
		return errors.Validation("Name is required when creating new sitemap")
	}
	if input.PromptID == 0 {
		return errors.Validation("PromptID is required")
	}
	if input.ProviderID == 0 {
		return errors.Validation("ProviderID is required")
	}
	if len(input.Titles) == 0 {
		return errors.Validation("At least one title is required")
	}
	return nil
}

// buildPromptsForGeneration builds prompts for AI generation
// Uses input.SitemapID for existing tree context if IncludeExistingTree is true
func (s *GenerationService) buildPromptsForGeneration(ctx context.Context, input GenerationInput) (string, string, error) {
	// Get and render prompt template
	systemPrompt, userPrompt, err := s.promptSvc.RenderPrompt(ctx, input.PromptID, input.Placeholders)
	if err != nil {
		return "", "", fmt.Errorf("failed to render prompt: %w", err)
	}

	// Build titles section
	var titlesBuilder strings.Builder
	titlesBuilder.WriteString("## Titles to organize into sitemap structure:\n\n")
	for i, t := range input.Titles {
		titlesBuilder.WriteString(fmt.Sprintf("%d. %s", i+1, t.Title))
		if len(t.Keywords) > 0 {
			titlesBuilder.WriteString(fmt.Sprintf(" [Keywords: %s]", strings.Join(t.Keywords, ", ")))
		}
		titlesBuilder.WriteString("\n")
	}

	// Build constraints section
	var constraintsBuilder strings.Builder
	constraintsBuilder.WriteString("\n## Constraints:\n")
	if input.MaxDepth > 0 {
		constraintsBuilder.WriteString(fmt.Sprintf("- Maximum depth: %d levels\n", input.MaxDepth))
	}
	constraintsBuilder.WriteString("- Generate URL-friendly slugs (lowercase, hyphens instead of spaces)\n")
	constraintsBuilder.WriteString("- Preserve any provided keywords for each title\n")
	constraintsBuilder.WriteString("- Create logical parent-child relationships based on topic hierarchy\n")

	// Include existing tree if requested (only for adding to existing sitemap)
	var existingTreeSection string
	if input.IncludeExistingTree && input.SitemapID > 0 {
		tree, err := s.sitemapSvc.GetNodesTree(ctx, input.SitemapID)
		if err == nil && len(tree) > 0 {
			treeJSON, err := s.serializeTree(tree)
			if err == nil {
				existingTreeSection = fmt.Sprintf("\n## Existing sitemap structure (for context):\n```json\n%s\n```\n", treeJSON)
				existingTreeSection += "\nPlease consider this existing structure when organizing new pages. Avoid duplicating existing pages.\n"
			}
		}
	}

	// Combine into user prompt
	fullUserPrompt := userPrompt + "\n" + titlesBuilder.String() + constraintsBuilder.String() + existingTreeSection

	return systemPrompt, fullUserPrompt, nil
}

func (s *GenerationService) serializeTree(nodes []*entities.SitemapNode) (string, error) {
	type simpleNode struct {
		Title    string       `json:"title"`
		Slug     string       `json:"slug"`
		Path     string       `json:"path"`
		Keywords []string     `json:"keywords,omitempty"`
		Children []simpleNode `json:"children,omitempty"`
	}

	var convertNode func(n *entities.SitemapNode) simpleNode
	convertNode = func(n *entities.SitemapNode) simpleNode {
		sn := simpleNode{
			Title:    n.Title,
			Slug:     n.Slug,
			Path:     n.Path,
			Keywords: n.Keywords,
		}
		for _, child := range n.Children {
			sn.Children = append(sn.Children, convertNode(child))
		}
		return sn
	}

	var simpleNodes []simpleNode
	for _, n := range nodes {
		simpleNodes = append(simpleNodes, convertNode(n))
	}

	data, err := json.MarshalIndent(simpleNodes, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *GenerationService) createNodesFromAIResponse(
	ctx context.Context,
	sitemapID int64,
	parentNodeIDs []int64,
	nodes []ai.SitemapGeneratedNode,
	maxDepth int,
	onNodeCreated func(nodeID int64),
) (int, error) {
	nodesCreated := 0

	// If multiple parent nodes, distribute top-level generated nodes among them
	// For simplicity, we'll assign all to first parent if only one, otherwise round-robin
	parentIdx := 0

	for _, genNode := range nodes {
		parentID := parentNodeIDs[parentIdx%len(parentNodeIDs)]
		parentIdx++

		created, err := s.createNodeRecursive(ctx, sitemapID, &parentID, genNode, 1, maxDepth, onNodeCreated)
		if err != nil {
			s.logger.ErrorWithErr(err, fmt.Sprintf("Failed to create node, title: %s", genNode.Title))
			continue
		}
		nodesCreated += created
	}

	return nodesCreated, nil
}

func (s *GenerationService) createNodeRecursive(
	ctx context.Context,
	sitemapID int64,
	parentID *int64,
	genNode ai.SitemapGeneratedNode,
	currentDepth int,
	maxDepth int,
	onNodeCreated func(nodeID int64),
) (int, error) {
	// Check depth limit
	if maxDepth > 0 && currentDepth > maxDepth {
		return 0, nil
	}

	// Generate slug if not provided or invalid
	nodeSlug := genNode.Slug
	if nodeSlug == "" {
		nodeSlug = slug.Make(genNode.Title)
	}

	now := time.Now()
	node := &entities.SitemapNode{
		SitemapID:        sitemapID,
		ParentID:         parentID,
		Title:            genNode.Title,
		Slug:             nodeSlug,
		Source:           entities.NodeSourceGenerated,
		ContentType:      entities.NodeContentTypePage,
		DesignStatus:     entities.DesignStatusDraft,
		GenerationStatus: entities.GenStatusNone,
		PublishStatus:    entities.PubStatusNone,
		Keywords:         genNode.Keywords,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.sitemapSvc.CreateNode(ctx, node); err != nil {
		return 0, fmt.Errorf("failed to create node %s: %w", genNode.Title, err)
	}

	nodesCreated := 1

	// Track created node for undo
	if onNodeCreated != nil {
		onNodeCreated(node.ID)
	}

	// Create children recursively
	for _, child := range genNode.Children {
		childCreated, err := s.createNodeRecursive(ctx, sitemapID, &node.ID, child, currentDepth+1, maxDepth, onNodeCreated)
		if err != nil {
			s.logger.ErrorWithErr(err, fmt.Sprintf("Failed to create child node, parentTitle %s childTitle %s", genNode.Title, child.Title))
			continue
		}
		nodesCreated += childCreated
	}

	return nodesCreated, nil
}
