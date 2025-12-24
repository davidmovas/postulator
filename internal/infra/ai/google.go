package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/pkg/errors"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

const googleProviderName = "Google"

var _ Client = (*GoogleClient)(nil)

type GoogleClient struct {
	client *genai.Client
	model  string
}

func (c *GoogleClient) GetProviderName() string {
	return "google"
}

func (c *GoogleClient) GetModelName() string {
	return c.model
}

type GoogleConfig struct {
	APIKey string
	Model  string
}

func NewGoogleClient(cfg GoogleConfig) (*GoogleClient, error) {
	if cfg.APIKey == "" {
		return nil, errors.Validation("Google API key is required")
	}

	if cfg.Model == "" {
		cfg.Model = "gemini-1.5-flash"
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(cfg.APIKey))
	if err != nil {
		return nil, errors.AI(googleProviderName, fmt.Errorf("failed to create client: %w", err))
	}

	return &GoogleClient{
		client: client,
		model:  cfg.Model,
	}, nil
}

func (c *GoogleClient) GenerateArticle(ctx context.Context, systemPrompt, userPrompt string) (*ArticleResult, error) {
	model := c.client.GenerativeModel(c.model)

	// Configure the model
	// 4096 is plenty for 800-1500 words of content
	model.SetTemperature(0.7)
	model.SetMaxOutputTokens(4096)

	// Set system instruction
	jsonInstructions := `
You must respond with a valid JSON object in the following format:
{
  "title": "Article title, should be engaging and SEO-friendly",
  "excerpt": "Article excerpt, should be short and concise",
  "content": "Full article content in HTML format with proper WordPress blocks"
}

Do not include any text before or after the JSON object. Only output the JSON.`

	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(systemPrompt + "\n\n" + jsonInstructions)},
	}

	// Generate response
	resp, err := model.GenerateContent(ctx, genai.Text(userPrompt))
	if err != nil {
		return nil, errors.AI(googleProviderName, fmt.Errorf("API error: %w", err))
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, errors.AI(googleProviderName, fmt.Errorf("no response from API"))
	}

	// Extract text from response
	var responseText string
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			responseText = string(text)
			break
		}
	}

	if responseText == "" {
		return nil, errors.AI(googleProviderName, fmt.Errorf("no text content in response"))
	}

	// Parse JSON from response
	var article ArticleContent
	jsonStr := extractJSON(responseText)
	if err = json.Unmarshal([]byte(jsonStr), &article); err != nil {
		// Try sanitizing the JSON to fix invalid escape sequences
		sanitized := sanitizeJSON(jsonStr)
		if err2 := json.Unmarshal([]byte(sanitized), &article); err2 != nil {
			return nil, errors.AI(googleProviderName, fmt.Errorf("failed to parse response: %w, raw: %s", err, responseText[:min(200, len(responseText))]))
		}
	}

	if article.Title == "" || article.Content == "" {
		return nil, errors.AI(googleProviderName, fmt.Errorf("empty title or content in response"))
	}

	// Calculate tokens and cost
	inputTokens := 0
	outputTokens := 0
	if resp.UsageMetadata != nil {
		inputTokens = int(resp.UsageMetadata.PromptTokenCount)
		outputTokens = int(resp.UsageMetadata.CandidatesTokenCount)
	}
	totalTokens := inputTokens + outputTokens
	cost := CalculateCost(entities.TypeGoogle, c.model, inputTokens, outputTokens)

	return &ArticleResult{
		Title:      article.Title,
		Excerpt:    article.Excerpt,
		Content:    article.Content,
		TokensUsed: totalTokens,
		Cost:       cost,
		Usage: Usage{
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			TotalTokens:  totalTokens,
			CostUSD:      cost,
		},
	}, nil
}

func (c *GoogleClient) GenerateTopicVariations(ctx context.Context, topic string, amount int) ([]string, error) {
	model := c.client.GenerativeModel(c.model)

	// Configure the model
	model.SetTemperature(0.8)
	model.SetMaxOutputTokens(1024)

	jsonInstructions := `
You must respond with a valid JSON object in the following format:
{
  "variations": ["variation 1", "variation 2", "variation 3"]
}

Do not include any text before or after the JSON object. Only output the JSON.`

	systemPrompt := "You are a helpful assistant that generates creative topic variations. Each variation should be unique but related to the original topic.\n\n" + jsonInstructions

	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(systemPrompt)},
	}

	userPrompt := fmt.Sprintf("Generate %d variations of the following topic:\n\n'%s'\n\nEach variation should be:\n- Unique and interesting\n- Related to the original topic\n- Suitable for a blog article\n- SEO-friendly\n- WITHOUT quotation marks in the titles", amount, topic)

	resp, err := model.GenerateContent(ctx, genai.Text(userPrompt))
	if err != nil {
		return nil, errors.AI(googleProviderName, fmt.Errorf("API error: %w", err))
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, errors.AI(googleProviderName, fmt.Errorf("no response from API"))
	}

	var responseText string
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			responseText = string(text)
			break
		}
	}

	if responseText == "" {
		return nil, errors.AI(googleProviderName, fmt.Errorf("no text content in response"))
	}

	var result TopicVariations
	jsonStr := extractJSON(responseText)
	if err = json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, errors.AI(googleProviderName, fmt.Errorf("failed to parse variations: %w", err))
	}

	if len(result.Variations) == 0 {
		return nil, errors.AI(googleProviderName, fmt.Errorf("no variations generated"))
	}

	if len(result.Variations) > amount {
		result.Variations = result.Variations[:amount]
	}

	cleanedVariations := make([]string, len(result.Variations))
	for i, variation := range result.Variations {
		cleanedVariations[i] = cleanQuotes(variation)
	}

	return cleanedVariations, nil
}

// Close closes the Google AI client
func (c *GoogleClient) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

func (c *GoogleClient) GenerateSitemapStructure(ctx context.Context, systemPrompt, userPrompt string) (*SitemapStructureResult, error) {
	model := c.client.GenerativeModel(c.model)

	// Configure the model
	model.SetTemperature(0.7)
	model.SetMaxOutputTokens(8192)

	jsonInstructions := `
You must respond with a valid JSON object in the following format:
{
  "nodes": [
    {
      "title": "Page title",
      "slug": "page-slug",
      "keywords": ["keyword1", "keyword2"],
      "children": [
        {
          "title": "Child page title",
          "slug": "child-page-slug",
          "keywords": ["keyword1"],
          "children": []
        }
      ]
    }
  ]
}

Do not include any text before or after the JSON object. Only output the JSON.`

	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(systemPrompt + "\n\n" + jsonInstructions)},
	}

	resp, err := model.GenerateContent(ctx, genai.Text(userPrompt))
	if err != nil {
		return nil, errors.AI(googleProviderName, fmt.Errorf("API error: %w", err))
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, errors.AI(googleProviderName, fmt.Errorf("no response from API"))
	}

	var responseText string
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			responseText = string(text)
			break
		}
	}

	if responseText == "" {
		return nil, errors.AI(googleProviderName, fmt.Errorf("no text content in response"))
	}

	var result SitemapStructureSchema
	jsonStr := extractJSON(responseText)
	if err = json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, errors.AI(googleProviderName, fmt.Errorf("failed to parse response: %w, raw: %s", err, responseText[:min(200, len(responseText))]))
	}

	if len(result.Nodes) == 0 {
		return nil, errors.AI(googleProviderName, fmt.Errorf("no nodes generated"))
	}

	// Calculate tokens and cost
	inputTokens := 0
	outputTokens := 0
	if resp.UsageMetadata != nil {
		inputTokens = int(resp.UsageMetadata.PromptTokenCount)
		outputTokens = int(resp.UsageMetadata.CandidatesTokenCount)
	}
	totalTokens := inputTokens + outputTokens
	cost := CalculateCost(entities.TypeGoogle, c.model, inputTokens, outputTokens)

	return &SitemapStructureResult{
		Nodes:      result.Nodes,
		TokensUsed: totalTokens,
		Cost:       cost,
		Usage: Usage{
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			TotalTokens:  totalTokens,
			CostUSD:      cost,
		},
	}, nil
}

func (c *GoogleClient) GenerateLinkSuggestions(ctx context.Context, request *LinkSuggestionRequest) (*LinkSuggestionResult, error) {
	model := c.client.GenerativeModel(c.model)

	model.SetTemperature(0.5)
	model.SetMaxOutputTokens(4096)

	jsonInstructions := `
You must respond with a valid JSON object in the following format:
{
  "links": [
    {
      "sourceNodeId": 1,
      "targetNodeId": 2,
      "anchorText": "text for the hyperlink",
      "reason": "why this link is valuable",
      "confidence": 0.85
    }
  ],
  "explanation": "overall strategy explanation"
}

Do not include any text before or after the JSON object. Only output the JSON.`

	baseSystem := request.SystemPrompt
	if baseSystem == "" {
		baseSystem = buildLinkSuggestionSystemPrompt()
	}
	systemPrompt := baseSystem + "\n\n" + jsonInstructions

	userPrompt := request.UserPrompt
	if userPrompt == "" {
		userPrompt = buildLinkSuggestionUserPrompt(request)
	}

	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(systemPrompt)},
	}

	resp, err := model.GenerateContent(ctx, genai.Text(userPrompt))
	if err != nil {
		return nil, errors.AI(googleProviderName, fmt.Errorf("API error: %w", err))
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, errors.AI(googleProviderName, fmt.Errorf("no response from API"))
	}

	var responseText string
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			responseText = string(text)
			break
		}
	}

	if responseText == "" {
		return nil, errors.AI(googleProviderName, fmt.Errorf("no text content in response"))
	}

	var result LinkSuggestionSchema
	jsonStr := extractJSON(responseText)
	if err = json.Unmarshal([]byte(jsonStr), &result); err != nil {
		sanitized := sanitizeJSON(jsonStr)
		if err2 := json.Unmarshal([]byte(sanitized), &result); err2 != nil {
			return nil, errors.AI(googleProviderName, fmt.Errorf("failed to parse response: %w", err))
		}
	}

	inputTokens := 0
	outputTokens := 0
	if resp.UsageMetadata != nil {
		inputTokens = int(resp.UsageMetadata.PromptTokenCount)
		outputTokens = int(resp.UsageMetadata.CandidatesTokenCount)
	}
	totalTokens := inputTokens + outputTokens
	cost := CalculateCost(entities.TypeGoogle, c.model, inputTokens, outputTokens)

	return &LinkSuggestionResult{
		Links:       result.Links,
		Explanation: result.Explanation,
		Usage: Usage{
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			TotalTokens:  totalTokens,
			CostUSD:      cost,
		},
	}, nil
}

func (c *GoogleClient) InsertLinks(ctx context.Context, request *InsertLinksRequest) (*InsertLinksResult, error) {
	if len(request.Links) == 0 {
		return &InsertLinksResult{
			Content:      request.Content,
			LinksApplied: 0,
			Usage:        Usage{},
		}, nil
	}

	model := c.client.GenerativeModel(c.model)

	model.SetTemperature(0.3) // Lower for precise edits
	model.SetMaxOutputTokens(16384)

	jsonInstructions := `
You must respond with a valid JSON object in the following format:
{
  "content": "The modified HTML content with links inserted",
  "linksApplied": 3
}

Do not include any text before or after the JSON object. Only output the JSON.`

	systemPrompt := buildInsertLinksSystemPrompt(request.Language) + "\n\n" + jsonInstructions
	userPrompt := buildInsertLinksUserPrompt(request)

	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(systemPrompt)},
	}

	fmt.Printf("[Google] InsertLinks: model=%s, contentLen=%d, links=%d\n",
		c.model, len(request.Content), len(request.Links))

	resp, err := model.GenerateContent(ctx, genai.Text(userPrompt))
	if err != nil {
		return nil, errors.AI(googleProviderName, fmt.Errorf("API error: %w", err))
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, errors.AI(googleProviderName, fmt.Errorf("no response from API"))
	}

	var responseText string
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			responseText = string(text)
			break
		}
	}

	if responseText == "" {
		return nil, errors.AI(googleProviderName, fmt.Errorf("no text content in response"))
	}

	var result InsertLinksContentSchema
	jsonStr := extractJSON(responseText)
	if err = json.Unmarshal([]byte(jsonStr), &result); err != nil {
		sanitized := sanitizeJSON(jsonStr)
		if err2 := json.Unmarshal([]byte(sanitized), &result); err2 != nil {
			preview := jsonStr
			if len(preview) > 300 {
				preview = preview[:300] + "..."
			}
			return nil, errors.AI(googleProviderName, fmt.Errorf("failed to parse response: %w, raw: %s", err, preview))
		}
	}

	inputTokens := 0
	outputTokens := 0
	if resp.UsageMetadata != nil {
		inputTokens = int(resp.UsageMetadata.PromptTokenCount)
		outputTokens = int(resp.UsageMetadata.CandidatesTokenCount)
	}
	totalTokens := inputTokens + outputTokens
	cost := CalculateCost(entities.TypeGoogle, c.model, inputTokens, outputTokens)

	fmt.Printf("[Google] InsertLinks success: linksApplied=%d, cost=$%.4f\n", result.LinksApplied, cost)

	return &InsertLinksResult{
		Content:      result.Content,
		LinksApplied: result.LinksApplied,
		Usage: Usage{
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			TotalTokens:  totalTokens,
			CostUSD:      cost,
		},
	}, nil
}
