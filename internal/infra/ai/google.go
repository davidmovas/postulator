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
		return nil, errors.AI(googleProviderName, fmt.Errorf("failed to parse response: %w, raw: %s", err, responseText[:min(200, len(responseText))]))
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
	cost := CalculateCost(entities.TypeGoogle, c.model, inputTokens, outputTokens)

	return &ArticleResult{
		Title:      article.Title,
		Excerpt:    article.Excerpt,
		Content:    article.Content,
		TokensUsed: inputTokens + outputTokens,
		Cost:       cost,
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
	cost := CalculateCost(entities.TypeGoogle, c.model, inputTokens, outputTokens)

	return &SitemapStructureResult{
		Nodes:      result.Nodes,
		TokensUsed: inputTokens + outputTokens,
		Cost:       cost,
	}, nil
}
