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

func (c *AnthropicClient) GenerateArticle(ctx context.Context, systemPrompt, userPrompt string, opts *GenerateArticleOptions) (*ArticleResult, error) {
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

	// 4096 is plenty for 800-1500 words of content
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
		// Try sanitizing the JSON to fix invalid escape sequences
		sanitized := sanitizeJSON(jsonStr)
		if err2 := json.Unmarshal([]byte(sanitized), &article); err2 != nil {
			return nil, errors.AI(anthropicProviderName, fmt.Errorf("failed to parse response: %w, raw: %s", err, responseText[:min(200, len(responseText))]))
		}
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

// sanitizeJSON fixes common invalid escape sequences in JSON strings
// This helps handle cases where AI generates content with invalid escapes like "\ " or "\x"
func sanitizeJSON(s string) string {
	// Fix invalid escape sequences by finding backslash not followed by valid escape chars
	// Valid JSON escapes: " \ / b f n r t u
	result := strings.Builder{}
	result.Grow(len(s))

	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			next := s[i+1]
			// Check if it's a valid escape sequence
			switch next {
			case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
				result.WriteByte(s[i])
			case 'u':
				// Unicode escape - check if followed by 4 hex digits
				if i+5 < len(s) {
					result.WriteByte(s[i])
				} else {
					// Invalid unicode escape, skip the backslash
					continue
				}
			default:
				// Invalid escape - skip the backslash
				continue
			}
		} else {
			result.WriteByte(s[i])
		}
	}

	return result.String()
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

func (c *AnthropicClient) GenerateLinkSuggestions(ctx context.Context, request *LinkSuggestionRequest) (*LinkSuggestionResult, error) {
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

	systemPrompt := request.SystemPrompt + "\n\n" + jsonInstructions
	userPrompt := request.UserPrompt

	message, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(c.model),
		MaxTokens: 4096,
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

	var result LinkSuggestionSchema
	jsonStr := extractJSON(responseText)
	if err = json.Unmarshal([]byte(jsonStr), &result); err != nil {
		sanitized := sanitizeJSON(jsonStr)
		if err2 := json.Unmarshal([]byte(sanitized), &result); err2 != nil {
			return nil, errors.AI(anthropicProviderName, fmt.Errorf("failed to parse response: %w", err))
		}
	}

	inputTokens := int(message.Usage.InputTokens)
	outputTokens := int(message.Usage.OutputTokens)
	totalTokens := inputTokens + outputTokens
	cost := CalculateCost(entities.TypeAnthropic, c.model, inputTokens, outputTokens)

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

func (c *AnthropicClient) InsertLinks(ctx context.Context, request *InsertLinksRequest) (*InsertLinksResult, error) {
	if len(request.Links) == 0 {
		return &InsertLinksResult{
			Content:      request.Content,
			LinksApplied: 0,
			Usage:        Usage{},
		}, nil
	}

	jsonInstructions := `
You must respond with a valid JSON object in the following format:
{
  "content": "The modified HTML content with links inserted",
  "linksApplied": 3
}

Do not include any text before or after the JSON object. Only output the JSON.`

	systemPrompt := request.SystemPrompt + "\n\n" + jsonInstructions
	userPrompt := request.UserPrompt

	fmt.Printf("[Anthropic] InsertLinks: model=%s, contentLen=%d, links=%d\n",
		c.model, len(request.Content), len(request.Links))

	message, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(c.model),
		MaxTokens: 16384, // More tokens for full content with links
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

	var result InsertLinksContentSchema
	jsonStr := extractJSON(responseText)
	if err = json.Unmarshal([]byte(jsonStr), &result); err != nil {
		sanitized := sanitizeJSON(jsonStr)
		if err2 := json.Unmarshal([]byte(sanitized), &result); err2 != nil {
			preview := jsonStr
			if len(preview) > 300 {
				preview = preview[:300] + "..."
			}
			return nil, errors.AI(anthropicProviderName, fmt.Errorf("failed to parse response: %w, raw: %s", err, preview))
		}
	}

	inputTokens := int(message.Usage.InputTokens)
	outputTokens := int(message.Usage.OutputTokens)
	totalTokens := inputTokens + outputTokens
	cost := CalculateCost(entities.TypeAnthropic, c.model, inputTokens, outputTokens)

	fmt.Printf("[Anthropic] InsertLinks success: linksApplied=%d, cost=$%.4f\n", result.LinksApplied, cost)

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
