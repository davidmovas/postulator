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
	if modelInfo := GetModelInfo(entities.TypeOpenAI, cfg.Model); modelInfo != nil {
		usesCompletionTokens = modelInfo.UsesCompletionTokens
		isReasoningModel = modelInfo.IsReasoningModel
	}

	return &OpenAIClient{
		client:               &client,
		model:                cfg.Model,
		modelName:            cfg.Model,
		usesCompletionTokens: usesCompletionTokens,
		isReasoningModel:     isReasoningModel,
	}, nil
}

func (c *OpenAIClient) GetProviderName() string {
	return "openai"
}

func (c *OpenAIClient) GetModelName() string {
	return c.modelName
}

func (c *OpenAIClient) GenerateArticle(ctx context.Context, systemPrompt, userPrompt string) (*ArticleResult, error) {
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

	const articleMaxTokens = 4096
	if c.usesCompletionTokens {
		params.MaxCompletionTokens = openaiSDK.Int(articleMaxTokens)
	} else {
		params.MaxTokens = openaiSDK.Int(articleMaxTokens)
	}

	fmt.Printf("[OpenAI] GenerateArticle: model=%s, maxTokens=%d\n", c.modelName, articleMaxTokens)

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
		return nil, errors.AI(providerName, fmt.Errorf("response truncated: max tokens reached. Try reducing word count or using a shorter prompt"))
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
	schema := generateSchema[LinkSuggestionSchema]()

	schemaParam := openaiSDK.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        "link_suggestions",
		Description: openaiSDK.String("Internal linking suggestions between pages"),
		Schema:      schema,
		Strict:      openaiSDK.Bool(true),
	}

	systemPrompt := request.SystemPrompt
	userPrompt := request.UserPrompt
	if systemPrompt == "" {
		systemPrompt = buildLinkSuggestionSystemPrompt()
	}
	if userPrompt == "" {
		userPrompt = buildLinkSuggestionUserPrompt(request)
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

	const maxTokens = 8192
	if c.usesCompletionTokens {
		params.MaxCompletionTokens = openaiSDK.Int(maxTokens)
	} else {
		params.MaxTokens = openaiSDK.Int(maxTokens)
	}

	// Log request info
	fmt.Printf("[OpenAI] GenerateLinkSuggestions: model=%s, nodes=%d, promptLen=%d\n",
		c.modelName, len(request.Nodes), len(systemPrompt)+len(userPrompt))

	chat, err := c.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, errors.AI(providerName, fmt.Errorf("API error: %w", err))
	}

	if len(chat.Choices) == 0 {
		return nil, errors.AI(providerName, fmt.Errorf("no response from API"))
	}

	choice := chat.Choices[0]
	finishReason := string(choice.FinishReason)

	// Log response info
	fmt.Printf("[OpenAI] LinkSuggestions response: promptTokens=%d, completionTokens=%d, finishReason=%s\n",
		chat.Usage.PromptTokens, chat.Usage.CompletionTokens, finishReason)

	// Check for problematic finish reasons
	if finishReason == "length" {
		return nil, errors.AI(providerName, fmt.Errorf("response truncated: max tokens reached"))
	}
	if finishReason == "content_filter" {
		return nil, errors.AI(providerName, fmt.Errorf("content filtered by safety system"))
	}

	content := choice.Message.Content
	if content == "" {
		// Log more details about the empty response
		fmt.Printf("[OpenAI] Empty response details: finishReason=%s, refusal=%v\n",
			finishReason, choice.Message.Refusal)
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

	fmt.Printf("[OpenAI] LinkSuggestions success: links=%d, cost=$%.4f\n", len(result.Links), cost)

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
	return `You are an SEO expert specializing in internal linking strategies.
Analyze the provided website pages and suggest internal links that will:
- Improve site navigation and user experience
- Distribute page authority (link equity) effectively
- Create topical clusters by linking related content
- Help search engines understand site structure

Guidelines:
- Each page should have 2-5 outgoing links (not too many, not too few)
- Prioritize semantic relevance over quantity
- Use descriptive, natural anchor text
- Avoid linking to the same page multiple times from one source
- Consider the user journey and information hierarchy`
}

func buildLinkSuggestionUserPrompt(request *LinkSuggestionRequest) string {
	var sb strings.Builder
	sb.WriteString("Pages to analyze:\n\n")

	for _, node := range request.Nodes {
		// Compact format: ID | Title | Path | Keywords | out/in counts
		sb.WriteString(fmt.Sprintf("ID:%d | %s | %s", node.ID, node.Title, node.Path))
		if len(node.Keywords) > 0 {
			// Limit to first 5 keywords to reduce prompt size
			kw := node.Keywords
			if len(kw) > 5 {
				kw = kw[:5]
			}
			sb.WriteString(fmt.Sprintf(" | kw: %s", strings.Join(kw, ",")))
		}
		sb.WriteString(fmt.Sprintf(" | out:%d in:%d\n", node.OutgoingCount, node.IncomingCount))
	}

	if request.MaxOutgoing > 0 || request.MaxIncoming > 0 {
		sb.WriteString("\nLimits: ")
		if request.MaxOutgoing > 0 {
			sb.WriteString(fmt.Sprintf("max %d outgoing, ", request.MaxOutgoing))
		}
		if request.MaxIncoming > 0 {
			sb.WriteString(fmt.Sprintf("max %d incoming", request.MaxIncoming))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\nSuggest internal links using exact page IDs. Focus on semantic relevance.")
	return sb.String()
}

