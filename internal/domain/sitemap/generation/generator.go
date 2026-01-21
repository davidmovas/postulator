package generation

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
	"github.com/davidmovas/postulator/internal/domain/sitemap"
	"github.com/davidmovas/postulator/internal/infra/ai"
	"github.com/davidmovas/postulator/pkg/logger"
)

type GeneratedContent struct {
	Title           string `json:"title"`
	Content         string `json:"content"`
	Excerpt         string `json:"excerpt"`
	MetaDescription string `json:"meta_description"`
}

type Generator struct {
	sitemapSvc      sitemap.Service
	promptSvc       prompts.Service
	providerSvc     providers.Service
	aiClientFactory func(provider *entities.Provider) (ai.Client, error)
	rateLimiter     *RateLimiter
	aiUsageService  aiusage.Service
	logger          *logger.Logger
}

func NewGenerator(
	sitemapSvc sitemap.Service,
	promptSvc prompts.Service,
	providerSvc providers.Service,
	aiClientFactory func(provider *entities.Provider) (ai.Client, error),
	rateLimiter *RateLimiter,
	aiUsageService aiusage.Service,
	logger *logger.Logger,
) *Generator {
	return &Generator{
		sitemapSvc:      sitemapSvc,
		promptSvc:       promptSvc,
		providerSvc:     providerSvc,
		aiClientFactory: aiClientFactory,
		rateLimiter:     rateLimiter,
		aiUsageService:  aiUsageService,
		logger:          logger.WithScope("page_generator"),
	}
}

type GenerateRequest struct {
	Node            *entities.SitemapNode
	Ancestors       []*entities.SitemapNode
	SiteID          int64
	ProviderID      int64
	PromptID        *int64
	Placeholders    map[string]string
	ContentSettings *ContentSettings
	LinkTargets     []LinkTarget // Approved outgoing links for this node
}

type GenerateResult struct {
	Content      *PageContent
	DurationMs   int64
	ProviderName string
	ModelName    string
}

func (g *Generator) Generate(ctx context.Context, req GenerateRequest) (*GenerateResult, error) {
	startTime := time.Now()

	provider, err := g.providerSvc.GetProvider(ctx, req.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	aiClient, err := g.aiClientFactory(provider)
	if err != nil {
		return nil, fmt.Errorf("failed to create AI client: %w", err)
	}

	if err := g.rateLimiter.Acquire(ctx, aiClient.GetProviderName(), aiClient.GetModelName()); err != nil {
		return nil, fmt.Errorf("rate limit error: %w", err)
	}

	systemPrompt, userPrompt, err := g.buildPrompts(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to build prompts: %w", err)
	}

	g.logger.Infof("Generating content for node %d (%s) with provider %s/%s, links=%d",
		req.Node.ID, req.Node.Title, aiClient.GetProviderName(), aiClient.GetModelName(), len(req.LinkTargets))

	articleResult, err := aiClient.GenerateArticle(ctx, systemPrompt, userPrompt, nil)
	durationMs := time.Since(startTime).Milliseconds()

	// Log AI usage regardless of success/failure
	if g.aiUsageService != nil {
		var usage ai.Usage
		if articleResult != nil {
			usage = articleResult.Usage
		}
		_ = g.aiUsageService.LogFromResult(
			ctx,
			req.SiteID,
			aiusage.OperationPageGeneration,
			aiClient,
			usage,
			durationMs,
			err,
			map[string]interface{}{
				"node_id":    req.Node.ID,
				"node_title": req.Node.Title,
				"node_path":  req.Node.Path,
			},
		)
	}

	if err != nil {
		return nil, fmt.Errorf("AI generation failed: %w", err)
	}

	content := &PageContent{
		Title:           articleResult.Title,
		Content:         articleResult.Content,
		Excerpt:         articleResult.Excerpt,
		MetaDescription: extractMetaDescription(articleResult),
		InputTokens:     articleResult.Usage.InputTokens,
		OutputTokens:    articleResult.Usage.OutputTokens,
		CostUSD:         articleResult.Usage.CostUSD,
	}

	g.logger.Infof("Generated content for node %d in %dms (tokens: %d)",
		req.Node.ID, durationMs, articleResult.Usage.TotalTokens)

	return &GenerateResult{
		Content:      content,
		DurationMs:   durationMs,
		ProviderName: aiClient.GetProviderName(),
		ModelName:    aiClient.GetModelName(),
	}, nil
}

func (g *Generator) buildPrompts(ctx context.Context, req GenerateRequest) (string, string, error) {
	g.logger.Infof("=== [DEBUG] buildPrompts for node %d (%s) ===", req.Node.ID, req.Node.Title)
	g.logger.Infof("  req.Node.Keywords: %v (len=%d)", req.Node.Keywords, len(req.Node.Keywords))

	nodeCtx := NodeContext{
		Title:       req.Node.Title,
		Path:        req.Node.Path,
		Keywords:    req.Node.Keywords,
		Language:    req.Placeholders["language"],
		Context:     req.Placeholders["context"],
		LinkTargets: req.LinkTargets,
	}

	g.logger.Infof("  nodeCtx.Keywords: %v (len=%d)", nodeCtx.Keywords, len(nodeCtx.Keywords))

	if req.ContentSettings != nil {
		nodeCtx.WordCount = req.ContentSettings.WordCount
		nodeCtx.WritingStyle = string(req.ContentSettings.WritingStyle)
		nodeCtx.ContentTone = string(req.ContentSettings.ContentTone)
		nodeCtx.CustomInstructions = req.ContentSettings.CustomInstructions
	}

	for _, ancestor := range req.Ancestors {
		nodeCtx.Hierarchy = append(nodeCtx.Hierarchy, HierarchyNode{
			Title: ancestor.Title,
			Path:  ancestor.Path,
			Depth: ancestor.Depth,
		})
	}

	runtimeData := BuildPlaceholders(nodeCtx)

	g.logger.Infof("  runtimeData['keywords']: %s (len=%d)", runtimeData["keywords"], len(runtimeData["keywords"]))
	g.logger.Infof("  All runtimeData keys: %v", func() []string {
		keys := make([]string, 0, len(runtimeData))
		for k := range runtimeData {
			keys = append(keys, k)
		}
		return keys
	}())

	for k, v := range req.Placeholders {
		if _, exists := runtimeData[k]; !exists {
			runtimeData[k] = v
		}
	}

	if req.PromptID != nil && *req.PromptID > 0 {
		prompt, err := g.promptSvc.GetPrompt(ctx, *req.PromptID)
		if err != nil {
			return "", "", err
		}

		g.logger.Infof("  Using DB prompt ID=%d, name=%s", *req.PromptID, prompt.Name)

		var overrides entities.ContextConfig
		if req.ContentSettings != nil {
			overrides = contentSettingsToOverrides(req.ContentSettings)
		}

		sys, usr, err := g.promptSvc.RenderPromptWithOverrides(ctx, prompt, runtimeData, overrides)
		if err == nil {
			g.logger.Infof("  Final user prompt contains 'keywords': %v", contains(usr, "keywords") || contains(usr, "Keywords"))
			g.logger.Infof("  User prompt preview (first 300 chars): %s", truncate(usr, 300))
		}
		return sys, usr, err
	}

	g.logger.Infof("  Using default builtin prompt")
	renderer := NewDefaultPromptRenderer()
	system, user := renderer.Render(runtimeData)
	g.logger.Infof("  Final user prompt contains 'keywords': %v", contains(user, "keywords") || contains(user, "Keywords"))
	g.logger.Infof("  User prompt preview (first 300 chars): %s", truncate(user, 300))
	return system, user, nil
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (len(s) >= len(substr) && s[:len(substr)] == substr ||
		len(s) > len(substr) && s[len(s)-len(substr):] == substr ||
		false || // placeholder for string search
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// contentSettingsToOverrides converts ContentSettings to ContextConfig overrides
// NOTE: customInstructions is passed via RuntimeData, not overrides (it's a runtime-only field)
func contentSettingsToOverrides(settings *ContentSettings) entities.ContextConfig {
	// If ContextOverrides is provided from UI, use it directly
	// This allows proper handling of enabled/disabled fields from the frontend
	if settings.ContextOverrides != nil && len(settings.ContextOverrides) > 0 {
		return settings.ContextOverrides
	}

	// Legacy fallback: convert individual fields to overrides
	overrides := make(entities.ContextConfig)

	if settings.WordCount != "" {
		overrides["wordCount"] = entities.ContextFieldValue{Enabled: true, Value: settings.WordCount}
	}
	if settings.WritingStyle != "" {
		overrides["writingStyle"] = entities.ContextFieldValue{Enabled: true, Value: string(settings.WritingStyle)}
	}
	if settings.ContentTone != "" {
		overrides["contentTone"] = entities.ContextFieldValue{Enabled: true, Value: string(settings.ContentTone)}
	}
	if settings.IncludeLinks {
		overrides["internalLinks"] = entities.ContextFieldValue{Enabled: true}
	}

	return overrides
}

func extractMetaDescription(result *ai.ArticleResult) string {
	if result.Excerpt != "" && len(result.Excerpt) <= 160 {
		return result.Excerpt
	}

	if result.Excerpt != "" {
		if len(result.Excerpt) > 157 {
			return result.Excerpt[:157] + "..."
		}
		return result.Excerpt
	}

	content := strings.TrimSpace(result.Content)
	content = strings.ReplaceAll(content, "<p>", "")
	content = strings.ReplaceAll(content, "</p>", " ")

	if len(content) > 157 {
		return content[:157] + "..."
	}
	return content
}

func ParseJSONContent(raw string) (*GeneratedContent, error) {
	raw = strings.TrimSpace(raw)

	if strings.HasPrefix(raw, "```json") {
		raw = strings.TrimPrefix(raw, "```json")
		raw = strings.TrimSuffix(raw, "```")
		raw = strings.TrimSpace(raw)
	} else if strings.HasPrefix(raw, "```") {
		raw = strings.TrimPrefix(raw, "```")
		raw = strings.TrimSuffix(raw, "```")
		raw = strings.TrimSpace(raw)
	}

	var content GeneratedContent
	if err := json.Unmarshal([]byte(raw), &content); err != nil {
		return nil, fmt.Errorf("failed to parse JSON content: %w", err)
	}

	return &content, nil
}
