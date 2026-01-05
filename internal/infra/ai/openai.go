package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/pkg/errors"

	"github.com/invopop/jsonschema"
	openaiSDK "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

const providerName = "OpenAI"

var _ Client = (*OpenAIClient)(nil)

type ArticleContent struct {
	Title   string `json:"title" jsonschema_description:"Page title"`
	Excerpt string `json:"excerpt" jsonschema_description:"Brief summary for previews"`
	Content string `json:"content" jsonschema_description:"Page content in HTML format"`
}

type TopicVariations struct {
	Variations []string `json:"variations" jsonschema_description:"List of topic variations"`
}

type Config struct {
	APIKey  string
	Model   string
	BaseURL string
}

type OpenAIClient struct {
	client               *openaiSDK.Client
	model                openaiSDK.ChatModel
	modelName            string
	usesCompletionTokens bool
	isReasoningModel     bool
	contextWindow        int // Total context window size
	maxOutputTokens      int // Maximum output tokens for the model
}

func NewOpenAIClient(cfg Config) (*OpenAIClient, error) {
	if cfg.APIKey == "" {
		return nil, errors.Validation("OpenAI API key is required")
	}

	if cfg.Model == "" {
		cfg.Model = openaiSDK.ChatModelGPT4oMini
	}

	opts := []option.RequestOption{
		option.WithAPIKey(cfg.APIKey),
	}

	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}

	client := openaiSDK.NewClient(opts...)

	usesCompletionTokens := false
	isReasoningModel := false
	contextWindow := 128000  // Default fallback
	maxOutputTokens := 16384 // Default fallback
	if modelInfo := GetModelInfo(entities.TypeOpenAI, cfg.Model); modelInfo != nil {
		usesCompletionTokens = modelInfo.UsesCompletionTokens
		isReasoningModel = modelInfo.IsReasoningModel
		if modelInfo.ContextWindow > 0 {
			contextWindow = modelInfo.ContextWindow
		}
		if modelInfo.MaxOutputTokens > 0 {
			maxOutputTokens = modelInfo.MaxOutputTokens
		}
	}

	return &OpenAIClient{
		client:               &client,
		model:                cfg.Model,
		modelName:            cfg.Model,
		usesCompletionTokens: usesCompletionTokens,
		isReasoningModel:     isReasoningModel,
		contextWindow:        contextWindow,
		maxOutputTokens:      maxOutputTokens,
	}, nil
}

func (c *OpenAIClient) GetProviderName() string {
	return "openai"
}

func (c *OpenAIClient) GetModelName() string {
	return c.modelName
}

// EstimateTokens estimates the number of tokens in a text string.
// Uses a rough approximation of ~4 characters per token for English text.
// For mixed content (HTML, JSON), uses ~3.5 chars per token to be more conservative.
func (c *OpenAIClient) EstimateTokens(text string) int {
	// Average ~3.5 chars per token for mixed content (code, HTML, etc.)
	return (len(text) * 10) / 35
}

// CalculateAvailableOutputTokens determines how many output tokens are available
// given the input prompt sizes and desired output tokens.
// It ensures we don't exceed the model's context window or max output limits.
func (c *OpenAIClient) CalculateAvailableOutputTokens(systemPrompt, userPrompt string, desiredOutput int) int {
	inputTokens := c.EstimateTokens(systemPrompt) + c.EstimateTokens(userPrompt)

	// Add 10% safety buffer for input estimation errors
	inputWithBuffer := int(float64(inputTokens) * 1.1)

	// Calculate available space in context window
	availableInContext := c.contextWindow - inputWithBuffer

	// Take the minimum of: available in context, model max output, desired output
	maxPossible := availableInContext
	if c.maxOutputTokens < maxPossible {
		maxPossible = c.maxOutputTokens
	}
	if desiredOutput > 0 && desiredOutput < maxPossible {
		maxPossible = desiredOutput
	}

	// Ensure we have at least some tokens for output
	if maxPossible < 100 {
		maxPossible = 100
	}

	return maxPossible
}

// ValidateRequest checks if the prompt will fit in the context window with room for output.
// Returns an error if the prompt is too large.
func (c *OpenAIClient) ValidateRequest(systemPrompt, userPrompt string, minRequiredOutput int) error {
	inputTokens := c.EstimateTokens(systemPrompt) + c.EstimateTokens(userPrompt)
	inputWithBuffer := int(float64(inputTokens) * 1.1)

	available := c.contextWindow - inputWithBuffer

	if available < minRequiredOutput {
		return fmt.Errorf("prompt too large: estimated %d input tokens, only %d tokens available for output (need at least %d). Context window: %d",
			inputTokens, available, minRequiredOutput, c.contextWindow)
	}

	return nil
}

func (c *OpenAIClient) GenerateArticle(ctx context.Context, systemPrompt, userPrompt string, opts *GenerateArticleOptions) (*ArticleResult, error) {
	// Calculate dynamic token limits based on input size
	// Default desired output is 4096 for articles, but we'll calculate what's actually available
	const desiredArticleTokens = 8192 // Desired output for a typical article
	const minArticleTokens = 2000     // Minimum tokens needed for a reasonable article

	// Validate request first
	if err := c.ValidateRequest(systemPrompt, userPrompt, minArticleTokens); err != nil {
		return nil, errors.AI(providerName, err)
	}

	// Calculate actual available tokens
	maxTokens := c.CalculateAvailableOutputTokens(systemPrompt, userPrompt, desiredArticleTokens)

	schema := generateSchema[ArticleContent]()

	schemaParam := openaiSDK.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        "article_content",
		Description: openaiSDK.String("Generated page content"),
		Schema:      schema,
		Strict:      openaiSDK.Bool(true),
	}

	messages := []openaiSDK.ChatCompletionMessageParamUnion{
		openaiSDK.SystemMessage(systemPrompt),
		openaiSDK.UserMessage(userPrompt),
	}

	params := openaiSDK.ChatCompletionNewParams{
		Messages: messages,
		ResponseFormat: openaiSDK.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openaiSDK.ResponseFormatJSONSchemaParam{
				JSONSchema: schemaParam,
			},
		},
		Model: c.model,
	}

	// Reasoning models (o1, o3, gpt-5 series) don't support temperature
	if !c.isReasoningModel {
		params.Temperature = openaiSDK.Float(0.7)
	}

	if c.usesCompletionTokens {
		params.MaxCompletionTokens = openaiSDK.Int(int64(maxTokens))
	} else {
		params.MaxTokens = openaiSDK.Int(int64(maxTokens))
	}

	inputEstimate := c.EstimateTokens(systemPrompt) + c.EstimateTokens(userPrompt)
	fmt.Printf("[OpenAI] GenerateArticle: model=%s, inputEstimate=%d, maxTokens=%d, contextWindow=%d\n",
		c.modelName, inputEstimate, maxTokens, c.contextWindow)

	chat, err := c.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, errors.AI(providerName, fmt.Errorf("API error: %w", err))
	}

	if len(chat.Choices) == 0 {
		return nil, errors.AI(providerName, fmt.Errorf("no response from API"))
	}

	choice := chat.Choices[0]

	// Log response stats
	fmt.Printf("[OpenAI] Response: promptTokens=%d, completionTokens=%d, finishReason=%s\n",
		chat.Usage.PromptTokens, chat.Usage.CompletionTokens, choice.FinishReason)

	// Check finish_reason for truncation
	finishReason := string(choice.FinishReason)
	if finishReason == "length" {
		// Provide detailed error message with token usage info
		return nil, errors.AI(providerName, fmt.Errorf(
			"response truncated: max tokens reached (used %d/%d output tokens, input: %d tokens). "+
				"Try: 1) reducing word count requirement, 2) using a shorter prompt, or 3) using a model with larger context window",
			chat.Usage.CompletionTokens, maxTokens, chat.Usage.PromptTokens))
	}
	if finishReason == "content_filter" {
		return nil, errors.AI(providerName, fmt.Errorf("content filtered by safety system"))
	}

	content := choice.Message.Content
	if content == "" {
		return nil, errors.AI(providerName, fmt.Errorf("empty response from API (finish_reason: %s)", finishReason))
	}

	var article ArticleContent
	if err = json.Unmarshal([]byte(content), &article); err != nil {
		// Try sanitizing the JSON to fix invalid escape sequences
		sanitized := sanitizeJSON(content)
		if err2 := json.Unmarshal([]byte(sanitized), &article); err2 != nil {
			// Log the raw content for debugging (first 500 chars)
			preview := content
			if len(preview) > 500 {
				preview = preview[:500] + "..."
			}
			return nil, errors.AI(providerName, fmt.Errorf("failed to parse response (finish_reason: %s): %w, preview: %s", finishReason, err, preview))
		}
	}

	if article.Title == "" || article.Content == "" {
		return nil, errors.AI(providerName, fmt.Errorf("empty title or content in response"))
	}

	inputTokens := int(chat.Usage.PromptTokens)
	outputTokens := int(chat.Usage.CompletionTokens)
	totalTokens := int(chat.Usage.TotalTokens)
	cost := CalculateCost(entities.TypeOpenAI, c.modelName, inputTokens, outputTokens)

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

func (c *OpenAIClient) GenerateTopicVariations(ctx context.Context, topic string, amount int) ([]string, error) {
	schema := generateSchema[TopicVariations]()

	schemaParam := openaiSDK.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        "topic_variations",
		Description: openaiSDK.String("Generated topic variations"),
		Schema:      schema,
		Strict:      openaiSDK.Bool(true),
	}

	systemPrompt := "You are a helpful assistant that generates creative topic variations. Each variation should be unique but related to the original topic."
	userPrompt := fmt.Sprintf("Generate %d variations of the following topic:\n\n'%s'\n\nEach variation should be:\n- Unique and interesting\n- Related to the original topic\n- Suitable for a blog article\n- SEO-friendly\n- WITHOUT quotation marks in the titles", amount, topic)

	messages := []openaiSDK.ChatCompletionMessageParamUnion{
		openaiSDK.SystemMessage(systemPrompt),
		openaiSDK.UserMessage(userPrompt),
	}

	params := openaiSDK.ChatCompletionNewParams{
		Messages: messages,
		ResponseFormat: openaiSDK.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openaiSDK.ResponseFormatJSONSchemaParam{
				JSONSchema: schemaParam,
			},
		},
		Model: c.model,
	}

	// Reasoning models (o1, o3, gpt-5 series) don't support temperature
	if !c.isReasoningModel {
		params.Temperature = openaiSDK.Float(0.8)
	}

	// Use appropriate token limit parameter based on model
	if c.usesCompletionTokens {
		params.MaxCompletionTokens = openaiSDK.Int(1024)
	} else {
		params.MaxTokens = openaiSDK.Int(1024)
	}

	chat, err := c.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, errors.AI(providerName, fmt.Errorf("API error: %w", err))
	}

	if len(chat.Choices) == 0 {
		return nil, errors.AI(providerName, fmt.Errorf("no response from API"))
	}

	content := chat.Choices[0].Message.Content
	var result TopicVariations
	if err = json.Unmarshal([]byte(content), &result); err != nil {
		return nil, errors.AI(providerName, fmt.Errorf("failed to parse variations: %w", err))
	}

	if len(result.Variations) == 0 {
		return nil, errors.AI(providerName, fmt.Errorf("no variations generated"))
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

func cleanQuotes(s string) string {
	s = strings.TrimSpace(s)

	if len(s) >= 2 {
		firstChar := s[0]
		lastChar := s[len(s)-1]

		if (firstChar == '"' && lastChar == '"') ||
			(firstChar == '\'' && lastChar == '\'') ||
			(firstChar == '`' && lastChar == '`') {
			return s[1 : len(s)-1]
		}
	}

	return s
}

func (c *OpenAIClient) GenerateSitemapStructure(ctx context.Context, systemPrompt, userPrompt string) (*SitemapStructureResult, error) {
	// Use JSON instructions in prompt instead of Structured Outputs
	// because recursive schemas (Children -> SitemapGeneratedNode) are not supported with strict mode
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

	messages := []openaiSDK.ChatCompletionMessageParamUnion{
		openaiSDK.SystemMessage(fullSystemPrompt),
		openaiSDK.UserMessage(userPrompt),
	}

	params := openaiSDK.ChatCompletionNewParams{
		Messages: messages,
		ResponseFormat: openaiSDK.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &openaiSDK.ResponseFormatJSONObjectParam{},
		},
		Model: c.model,
	}

	// Reasoning models (o1, o3, gpt-5 series) don't support temperature
	if !c.isReasoningModel {
		params.Temperature = openaiSDK.Float(0.7)
	}

	// Use appropriate token limit parameter based on model
	if c.usesCompletionTokens {
		params.MaxCompletionTokens = openaiSDK.Int(8192)
	} else {
		params.MaxTokens = openaiSDK.Int(8192)
	}

	chat, err := c.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, errors.AI(providerName, fmt.Errorf("API error: %w", err))
	}

	if len(chat.Choices) == 0 {
		return nil, errors.AI(providerName, fmt.Errorf("no response from API"))
	}

	content := chat.Choices[0].Message.Content
	var result SitemapStructureSchema
	jsonStr := extractJSON(content)
	if err = json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, errors.AI(providerName, fmt.Errorf("failed to parse response: %w, raw: %s", err, content[:min(200, len(content))]))
	}

	if len(result.Nodes) == 0 {
		return nil, errors.AI(providerName, fmt.Errorf("no nodes generated"))
	}

	inputTokens := int(chat.Usage.PromptTokens)
	outputTokens := int(chat.Usage.CompletionTokens)
	totalTokens := int(chat.Usage.TotalTokens)
	cost := CalculateCost(entities.TypeOpenAI, c.modelName, inputTokens, outputTokens)

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

func (c *OpenAIClient) GenerateLinkSuggestions(ctx context.Context, request *LinkSuggestionRequest) (*LinkSuggestionResult, error) {
	// Calculate dynamic token limits based on input size
	// For link suggestions, we need room for links array and explanation
	// Estimate: ~50 tokens per suggested link + 200 for explanation
	// For 25 nodes, expect ~30-60 links = 1500-3000 tokens minimum
	const minRequiredTokens = 2000
	const desiredTokens = 16384 // Increased from 8192 to handle larger batches

	systemPrompt := request.SystemPrompt
	userPrompt := request.UserPrompt
	if systemPrompt == "" {
		systemPrompt = buildLinkSuggestionSystemPrompt()
	}
	if userPrompt == "" {
		userPrompt = buildLinkSuggestionUserPrompt(request)
	}

	// Validate request first - check that prompt isn't too large
	if err := c.ValidateRequest(systemPrompt, userPrompt, minRequiredTokens); err != nil {
		return nil, errors.AI(providerName, fmt.Errorf(
			"prompt too large for link suggestions (%d nodes): %w. Try reducing batch size or simplifying node data.",
			len(request.Nodes), err))
	}

	// Calculate actual available tokens dynamically
	maxTokens := c.CalculateAvailableOutputTokens(systemPrompt, userPrompt, desiredTokens)

	// Log for debugging
	inputEstimate := c.EstimateTokens(systemPrompt) + c.EstimateTokens(userPrompt)
	fmt.Printf("[OpenAI] LinkSuggestions: nodes=%d, inputEstimate=%d, maxTokens=%d, contextWindow=%d\n",
		len(request.Nodes), inputEstimate, maxTokens, c.contextWindow)

	schema := generateSchema[LinkSuggestionSchema]()

	schemaParam := openaiSDK.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        "link_suggestions",
		Description: openaiSDK.String("Internal linking suggestions between pages"),
		Schema:      schema,
		Strict:      openaiSDK.Bool(true),
	}

	messages := []openaiSDK.ChatCompletionMessageParamUnion{
		openaiSDK.SystemMessage(systemPrompt),
		openaiSDK.UserMessage(userPrompt),
	}

	params := openaiSDK.ChatCompletionNewParams{
		Messages: messages,
		ResponseFormat: openaiSDK.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openaiSDK.ResponseFormatJSONSchemaParam{
				JSONSchema: schemaParam,
			},
		},
		Model: c.model,
	}

	if !c.isReasoningModel {
		params.Temperature = openaiSDK.Float(0.5)
	}

	if c.usesCompletionTokens {
		params.MaxCompletionTokens = openaiSDK.Int(int64(maxTokens))
	} else {
		params.MaxTokens = openaiSDK.Int(int64(maxTokens))
	}

	chat, err := c.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, errors.AI(providerName, fmt.Errorf("API error: %w", err))
	}

	if len(chat.Choices) == 0 {
		return nil, errors.AI(providerName, fmt.Errorf("no response from API"))
	}

	choice := chat.Choices[0]
	finishReason := choice.FinishReason

	// Log response stats
	fmt.Printf("[OpenAI] LinkSuggestions response: promptTokens=%d, completionTokens=%d, finishReason=%s\n",
		chat.Usage.PromptTokens, chat.Usage.CompletionTokens, finishReason)

	// Check for problematic finish reasons
	if finishReason == "length" {
		return nil, errors.AI(providerName, fmt.Errorf(
			"response truncated: max tokens reached (used %d/%d output tokens, input: %d tokens). "+
				"Try: 1) reducing batch size (current: %d nodes), 2) limiting keywords per node, "+
				"or 3) using a model with larger context window",
			chat.Usage.CompletionTokens, maxTokens, chat.Usage.PromptTokens, len(request.Nodes)))
	}
	if finishReason == "content_filter" {
		return nil, errors.AI(providerName, fmt.Errorf("content filtered by safety system"))
	}

	content := choice.Message.Content
	if content == "" {
		return nil, errors.AI(providerName, fmt.Errorf("empty response from API (finishReason: %s)", finishReason))
	}

	var result LinkSuggestionSchema
	if err = json.Unmarshal([]byte(content), &result); err != nil {
		sanitized := sanitizeJSON(content)
		if err2 := json.Unmarshal([]byte(sanitized), &result); err2 != nil {
			preview := content
			if len(preview) > 300 {
				preview = preview[:300] + "..."
			}
			return nil, errors.AI(providerName, fmt.Errorf("failed to parse response: %w, raw: %s", err, preview))
		}
	}

	inputTokens := int(chat.Usage.PromptTokens)
	outputTokens := int(chat.Usage.CompletionTokens)
	totalTokens := int(chat.Usage.TotalTokens)
	cost := CalculateCost(entities.TypeOpenAI, c.modelName, inputTokens, outputTokens)

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

func generateSchema[T any]() interface{} {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	schema := reflector.Reflect(v)
	return schema
}

func buildLinkSuggestionSystemPrompt() string {
	return `You are an internal linking strategist for websites.

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
}

func buildLinkSuggestionUserPrompt(request *LinkSuggestionRequest) string {
	var sb strings.Builder
	sb.WriteString("PAGES:\n")

	for _, node := range request.Nodes {
		// Format: [ID:X] "Title" /path [kw: a,b] [X→ Y←]
		sb.WriteString(fmt.Sprintf("[ID:%d] \"%s\" %s", node.ID, node.Title, node.Path))
		if len(node.Keywords) > 0 {
			kw := node.Keywords
			if len(kw) > 3 {
				kw = kw[:3]
			}
			sb.WriteString(fmt.Sprintf(" [kw: %s]", strings.Join(kw, ", ")))
		}
		sb.WriteString(fmt.Sprintf(" [%d→ %d←]\n", node.OutgoingCount, node.IncomingCount))
	}

	if request.MaxOutgoing > 0 || request.MaxIncoming > 0 {
		sb.WriteString("\nCONSTRAINTS:\n")
		if request.MaxOutgoing > 0 {
			sb.WriteString(fmt.Sprintf("- Max %d outgoing links per page\n", request.MaxOutgoing))
		}
		if request.MaxIncoming > 0 {
			sb.WriteString(fmt.Sprintf("- Max %d incoming links per page\n", request.MaxIncoming))
		}
	}

	sb.WriteString("\nSuggest links that make sense semantically. Use exact page IDs.")
	return sb.String()
}

type InsertLinksContentSchema struct {
	Content      string `json:"content" jsonschema_description:"Modified HTML content with links inserted"`
	LinksApplied int    `json:"linksApplied" jsonschema_description:"Number of links successfully inserted"`
}

func (c *OpenAIClient) InsertLinks(ctx context.Context, request *InsertLinksRequest) (*InsertLinksResult, error) {
	if len(request.Links) == 0 {
		return &InsertLinksResult{
			Content:      request.Content,
			LinksApplied: 0,
			Usage:        Usage{},
		}, nil
	}

	schema := generateSchema[InsertLinksContentSchema]()

	schemaParam := openaiSDK.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        "content_with_links",
		Description: openaiSDK.String("HTML content with internal links inserted"),
		Schema:      schema,
		Strict:      openaiSDK.Bool(true),
	}

	systemPrompt := request.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = buildInsertLinksSystemPrompt(request.Language)
	}
	userPrompt := request.UserPrompt
	if userPrompt == "" {
		userPrompt = buildInsertLinksUserPrompt(request)
	}

	// For InsertLinks, output should be at least as large as input content
	// since we're returning modified HTML. Add 20% buffer for links and JSON wrapper.
	contentTokenEstimate := c.EstimateTokens(request.Content)
	desiredOutputTokens := int(float64(contentTokenEstimate) * 1.3)
	if desiredOutputTokens < 4096 {
		desiredOutputTokens = 4096
	}

	// Validate request
	if err := c.ValidateRequest(systemPrompt, userPrompt, contentTokenEstimate); err != nil {
		return nil, errors.AI(providerName, fmt.Errorf("content too large for link insertion: %w", err))
	}

	// Calculate actual available tokens
	maxTokens := c.CalculateAvailableOutputTokens(systemPrompt, userPrompt, desiredOutputTokens)

	messages := []openaiSDK.ChatCompletionMessageParamUnion{
		openaiSDK.SystemMessage(systemPrompt),
		openaiSDK.UserMessage(userPrompt),
	}

	params := openaiSDK.ChatCompletionNewParams{
		Messages: messages,
		ResponseFormat: openaiSDK.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openaiSDK.ResponseFormatJSONSchemaParam{
				JSONSchema: schemaParam,
			},
		},
		Model: c.model,
	}

	if !c.isReasoningModel {
		params.Temperature = openaiSDK.Float(0.3) // Lower temperature for more precise edits
	}

	if c.usesCompletionTokens {
		params.MaxCompletionTokens = openaiSDK.Int(int64(maxTokens))
	} else {
		params.MaxTokens = openaiSDK.Int(int64(maxTokens))
	}

	inputEstimate := c.EstimateTokens(systemPrompt) + c.EstimateTokens(userPrompt)
	fmt.Printf("[OpenAI] InsertLinks: model=%s, contentLen=%d, links=%d, inputEstimate=%d, maxTokens=%d\n",
		c.modelName, len(request.Content), len(request.Links), inputEstimate, maxTokens)

	chat, err := c.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, errors.AI(providerName, fmt.Errorf("API error: %w", err))
	}

	if len(chat.Choices) == 0 {
		return nil, errors.AI(providerName, fmt.Errorf("no response from API"))
	}

	choice := chat.Choices[0]
	finishReason := string(choice.FinishReason)

	if finishReason == "length" {
		return nil, errors.AI(providerName, fmt.Errorf(
			"response truncated: content too long (used %d/%d output tokens, input: %d tokens). "+
				"Content size: %d chars. Try using a model with larger context window.",
			chat.Usage.CompletionTokens, maxTokens, chat.Usage.PromptTokens, len(request.Content)))
	}
	if finishReason == "content_filter" {
		return nil, errors.AI(providerName, fmt.Errorf("content filtered by safety system"))
	}

	content := choice.Message.Content
	if content == "" {
		return nil, errors.AI(providerName, fmt.Errorf("empty response from API"))
	}

	var result InsertLinksContentSchema
	if err = json.Unmarshal([]byte(content), &result); err != nil {
		sanitized := sanitizeJSON(content)
		if err2 := json.Unmarshal([]byte(sanitized), &result); err2 != nil {
			preview := content
			if len(preview) > 300 {
				preview = preview[:300] + "..."
			}
			return nil, errors.AI(providerName, fmt.Errorf("failed to parse response: %w, raw: %s", err, preview))
		}
	}

	inputTokens := int(chat.Usage.PromptTokens)
	outputTokens := int(chat.Usage.CompletionTokens)
	totalTokens := int(chat.Usage.TotalTokens)
	cost := CalculateCost(entities.TypeOpenAI, c.modelName, inputTokens, outputTokens)

	fmt.Printf("[OpenAI] InsertLinks success: linksApplied=%d, cost=$%.4f\n", result.LinksApplied, cost)

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

func buildInsertLinksSystemPrompt(language string) string {
	if language == "" {
		language = "English"
	}
	return fmt.Sprintf(`You are a link insertion tool. Your ONLY job is to add <a> tags to existing HTML content.

TASK: Insert the specified internal links into the content without modifying anything else.

STRICT RULES:
1. Return the EXACT same HTML, only adding <a href="...">...</a> tags
2. Do NOT rewrite, rephrase, or change any text
3. Do NOT change HTML structure, formatting, or whitespace
4. Do NOT add links inside existing <a> tags (avoid nested links)
5. Insert each link only ONCE per page (first suitable occurrence)
6. Do NOT repeat the same link multiple times

HOW TO INSERT:
- If anchor text is provided: find that exact text (or close match) and wrap it with <a> tag
- If no anchor text: find text that naturally describes the target page and wrap it
- If no suitable text exists in content: skip that link (don't force it)

Language for anchor text selection: %s

OUTPUT: Return modified HTML and count of successfully inserted links.`, language)
}

func buildInsertLinksUserPrompt(request *InsertLinksRequest) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("PAGE: \"%s\" %s\n\n", request.PageTitle, request.PagePath))

	sb.WriteString("INSERT THESE LINKS:\n")
	for i, link := range request.Links {
		sb.WriteString(fmt.Sprintf("%d. → %s \"%s\"\n", i+1, link.TargetPath, link.TargetTitle))
		if link.AnchorText != nil && *link.AnchorText != "" {
			sb.WriteString(fmt.Sprintf("   Anchor: \"%s\"\n", *link.AnchorText))
		} else {
			sb.WriteString("   Anchor: find suitable text\n")
		}
	}

	sb.WriteString("\nCONTENT:\n")
	sb.WriteString(request.Content)

	return sb.String()
}
