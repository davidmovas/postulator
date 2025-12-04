package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/pkg/errors"
)

const anthropicProviderName = "Anthropic"

var _ Client = (*AnthropicClient)(nil)

type AnthropicClient struct {
	client *anthropic.Client
	model  string
}

func (c *AnthropicClient) GetProviderName() string {
	return "anthropic"
}

func (c *AnthropicClient) GetModelName() string {
	return c.model
}

type AnthropicConfig struct {
	APIKey string
	Model  string
}

func NewAnthropicClient(cfg AnthropicConfig) (*AnthropicClient, error) {
	if cfg.APIKey == "" {
		return nil, errors.Validation("Anthropic API key is required")
	}

	if cfg.Model == "" {
		cfg.Model = "claude-3-5-sonnet-20241022"
	}

	client := anthropic.NewClient(
		option.WithAPIKey(cfg.APIKey),
	)

	return &AnthropicClient{
		client: &client,
		model:  cfg.Model,
	}, nil
}

func (c *AnthropicClient) GenerateArticle(ctx context.Context, systemPrompt, userPrompt string) (*ArticleResult, error) {
	// Create JSON schema instructions for the response
	jsonInstructions := `
You must respond with a valid JSON object in the following format:
{
  "title": "Article title, should be engaging and SEO-friendly",
  "excerpt": "Article excerpt, should be short and concise",
  "content": "Full article content in HTML format with proper WordPress blocks"
}

Do not include any text before or after the JSON object. Only output the JSON.`

	fullSystemPrompt := systemPrompt + "\n\n" + jsonInstructions

	message, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(c.model),
		MaxTokens: 4096,
		System: []anthropic.TextBlockParam{
			{
				Type: "text",
				Text: fullSystemPrompt,
			},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt)),
		},
	})
	if err != nil {
		return nil, errors.AI(anthropicProviderName, fmt.Errorf("API error: %w", err))
	}

	if len(message.Content) == 0 {
		return nil, errors.AI(anthropicProviderName, fmt.Errorf("no response from API"))
	}

	// Extract text content
	var responseText string
	for _, block := range message.Content {
		if block.Type == "text" {
			responseText = block.Text
			break
		}
	}

	if responseText == "" {
		return nil, errors.AI(anthropicProviderName, fmt.Errorf("no text content in response"))
	}

	// Parse JSON from response
	var article ArticleContent
	jsonStr := extractJSON(responseText)
	if err = json.Unmarshal([]byte(jsonStr), &article); err != nil {
		return nil, errors.AI(anthropicProviderName, fmt.Errorf("failed to parse response: %w, raw: %s", err, responseText[:min(200, len(responseText))]))
	}

	if article.Title == "" || article.Content == "" {
		return nil, errors.AI(anthropicProviderName, fmt.Errorf("empty title or content in response"))
	}

	inputTokens := int(message.Usage.InputTokens)
	outputTokens := int(message.Usage.OutputTokens)
	totalTokens := inputTokens + outputTokens
	cost := CalculateCost(entities.TypeAnthropic, c.model, inputTokens, outputTokens)

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

func (c *AnthropicClient) GenerateTopicVariations(ctx context.Context, topic string, amount int) ([]string, error) {
	jsonInstructions := `
You must respond with a valid JSON object in the following format:
{
  "variations": ["variation 1", "variation 2", "variation 3"]
}

Do not include any text before or after the JSON object. Only output the JSON.`

	systemPrompt := "You are a helpful assistant that generates creative topic variations. Each variation should be unique but related to the original topic.\n\n" + jsonInstructions

	userPrompt := fmt.Sprintf("Generate %d variations of the following topic:\n\n'%s'\n\nEach variation should be:\n- Unique and interesting\n- Related to the original topic\n- Suitable for a blog article\n- SEO-friendly\n- WITHOUT quotation marks in the titles", amount, topic)

	message, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(c.model),
		MaxTokens: 1024,
		System: []anthropic.TextBlockParam{
			{
				Type: "text",
				Text: systemPrompt,
			},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt)),
		},
	})
	if err != nil {
		return nil, errors.AI(anthropicProviderName, fmt.Errorf("API error: %w", err))
	}

	if len(message.Content) == 0 {
		return nil, errors.AI(anthropicProviderName, fmt.Errorf("no response from API"))
	}

	var responseText string
	for _, block := range message.Content {
		if block.Type == "text" {
			responseText = block.Text
			break
		}
	}

	if responseText == "" {
		return nil, errors.AI(anthropicProviderName, fmt.Errorf("no text content in response"))
	}

	var result TopicVariations
	jsonStr := extractJSON(responseText)
	if err = json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, errors.AI(anthropicProviderName, fmt.Errorf("failed to parse variations: %w", err))
	}

	if len(result.Variations) == 0 {
		return nil, errors.AI(anthropicProviderName, fmt.Errorf("no variations generated"))
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

// extractJSON attempts to extract a JSON object from a string that may contain other text
func extractJSON(s string) string {
	s = strings.TrimSpace(s)

	// Try to find JSON object boundaries
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")

	if start != -1 && end != -1 && end > start {
		return s[start : end+1]
	}

	return s
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (c *AnthropicClient) GenerateSitemapStructure(ctx context.Context, systemPrompt, userPrompt string) (*SitemapStructureResult, error) {
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

	fullSystemPrompt := systemPrompt + "\n\n" + jsonInstructions

	message, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(c.model),
		MaxTokens: 8192,
		System: []anthropic.TextBlockParam{
			{
				Type: "text",
				Text: fullSystemPrompt,
			},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt)),
		},
	})
	if err != nil {
		return nil, errors.AI(anthropicProviderName, fmt.Errorf("API error: %w", err))
	}

	if len(message.Content) == 0 {
		return nil, errors.AI(anthropicProviderName, fmt.Errorf("no response from API"))
	}

	var responseText string
	for _, block := range message.Content {
		if block.Type == "text" {
			responseText = block.Text
			break
		}
	}

	if responseText == "" {
		return nil, errors.AI(anthropicProviderName, fmt.Errorf("no text content in response"))
	}

	var result SitemapStructureSchema
	jsonStr := extractJSON(responseText)
	if err = json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, errors.AI(anthropicProviderName, fmt.Errorf("failed to parse response: %w, raw: %s", err, responseText[:min(200, len(responseText))]))
	}

	if len(result.Nodes) == 0 {
		return nil, errors.AI(anthropicProviderName, fmt.Errorf("no nodes generated"))
	}

	inputTokens := int(message.Usage.InputTokens)
	outputTokens := int(message.Usage.OutputTokens)
	totalTokens := inputTokens + outputTokens
	cost := CalculateCost(entities.TypeAnthropic, c.model, inputTokens, outputTokens)

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
