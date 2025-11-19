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
	client *openaiSDK.Client
	model  openaiSDK.ChatModel
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

	return &OpenAIClient{
		client: &client,
		model:  cfg.Model,
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

	chat, err := c.client.Chat.Completions.New(ctx, openaiSDK.ChatCompletionNewParams{
		Messages: messages,
		ResponseFormat: openaiSDK.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openaiSDK.ResponseFormatJSONSchemaParam{
				JSONSchema: schemaParam,
			},
		},
		Model:       c.model,
		Temperature: openaiSDK.Float(0.7),
		MaxTokens:   openaiSDK.Int(4096),
	})
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

	chat, err := c.client.Chat.Completions.New(ctx, openaiSDK.ChatCompletionNewParams{
		Messages: messages,
		ResponseFormat: openaiSDK.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openaiSDK.ResponseFormatJSONSchemaParam{
				JSONSchema: schemaParam,
			},
		},
		Model:       c.model,
		Temperature: openaiSDK.Float(0.8),
		MaxTokens:   openaiSDK.Int(1024),
	})
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

func generateSchema[T any]() interface{} {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	schema := reflector.Reflect(v)
	return schema
}
