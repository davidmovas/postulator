package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/davidmovas/postulator/pkg/errors"

	"github.com/invopop/jsonschema"
	openaiSDK "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

const providerName = "OpenAI"

var _ Client = (*OpenAIClient)(nil)

type ArticleContent struct {
	Title   string `json:"title" jsonschema_description:"Article title, should be engaging and SEO-friendly"`
	Excerpt string `json:"excerpt" jsonschema_description:"Article excerpt, should be short and concise"`
	Content string `json:"content" jsonschema_description:"Full article content in HTML format with proper WordPress blocks"`
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
	usesCompletionTokens bool
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

	// Check if this model uses max_completion_tokens instead of max_tokens
	usesCompletionTokens := false
	if modelInfo := GetModelInfo("openai", cfg.Model); modelInfo != nil {
		usesCompletionTokens = modelInfo.UsesCompletionTokens
	}

	return &OpenAIClient{
		client:               &client,
		model:                cfg.Model,
		usesCompletionTokens: usesCompletionTokens,
	}, nil
}

func (c *OpenAIClient) GenerateArticle(ctx context.Context, systemPrompt, userPrompt string) (*ArticleResult, error) {
	schema := generateSchema[ArticleContent]()

	schemaParam := openaiSDK.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        "article_content",
		Description: openaiSDK.String("Generated article with title and content in HTML format"),
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
		Model:       c.model,
		Temperature: openaiSDK.Float(0.7),
	}

	// Use appropriate token limit parameter based on model
	if c.usesCompletionTokens {
		params.MaxCompletionTokens = openaiSDK.Int(4096)
	} else {
		params.MaxTokens = openaiSDK.Int(4096)
	}

	chat, err := c.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, errors.AI(providerName, fmt.Errorf("API error: %w", err))
	}

	if len(chat.Choices) == 0 {
		return nil, errors.AI(providerName, fmt.Errorf("no response from API"))
	}

	content := chat.Choices[0].Message.Content
	var article ArticleContent
	if err = json.Unmarshal([]byte(content), &article); err != nil {
		return nil, errors.AI(providerName, fmt.Errorf("failed to parse response: %w", err))
	}

	if article.Title == "" || article.Content == "" {
		return nil, errors.AI(providerName, fmt.Errorf("empty title or content in response"))
	}

	inputTokens := chat.Usage.PromptTokens
	outputTokens := chat.Usage.CompletionTokens
	cost := CalculateCost("openai", c.model, int(inputTokens), int(outputTokens))

	return &ArticleResult{
		Title:      article.Title,
		Excerpt:    article.Excerpt,
		Content:    article.Content,
		TokensUsed: int(chat.Usage.TotalTokens),
		Cost:       cost,
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
		Model:       c.model,
		Temperature: openaiSDK.Float(0.8),
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
		Model:       c.model,
		Temperature: openaiSDK.Float(0.7),
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

	inputTokens := chat.Usage.PromptTokens
	outputTokens := chat.Usage.CompletionTokens
	cost := CalculateCost("openai", c.model, int(inputTokens), int(outputTokens))

	return &SitemapStructureResult{
		Nodes:      result.Nodes,
		TokensUsed: int(chat.Usage.TotalTokens),
		Cost:       cost,
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

