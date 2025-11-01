package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/pkg/errors"

	"github.com/invopop/jsonschema"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

const provider = "OpenAI"

type ArticleContent struct {
	Title   string `json:"title" jsonschema_description:"Article title, should be engaging and SEO-friendly"`
	Content string `json:"content" jsonschema_description:"Full article content in HTML format with proper WordPress blocks"`
}

func GenerateSchema[T any]() interface{} {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	schema := reflector.Reflect(v)
	return schema
}

var ArticleResponseSchema = GenerateSchema[ArticleContent]()

var _ IClient = (*OpenAIClient)(nil)

type OpenAIClient struct {
	client *openai.Client
	model  openai.ChatModel
}

type OpenAIConfig struct {
	APIKey  string
	Model   string
	BaseURL string
}

func NewOpenAIClient(config OpenAIConfig) (*OpenAIClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	if config.Model == "" {
		config.Model = openai.ChatModelGPT4oMini
	}

	client := openai.NewClient(option.WithAPIKey(config.APIKey))

	return &OpenAIClient{
		client: &client,
		model:  getOpenAIModel(config.Model),
	}, nil
}

func (c *OpenAIClient) GenerateArticle(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        "article_content",
		Description: openai.String("Generated article with title, content and summary"),
		Schema:      ArticleResponseSchema,
		Strict:      openai.Bool(true),
	}

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemPrompt),
		openai.UserMessage(userPrompt),
	}

	chat, err := c.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: messages,
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
				JSONSchema: schemaParam,
			},
		},
		Model:       c.model,
		Temperature: openai.Float(0.7),
		MaxTokens:   openai.Int(4000),
	})

	if err != nil {
		return "", errors.AI(provider, fmt.Errorf("OpenAI API error: %w", err))
	}

	if len(chat.Choices) == 0 {
		return "", errors.AI(provider, fmt.Errorf("no response from OpenAI"))
	}

	content := chat.Choices[0].Message.Content
	var article ArticleContent
	if err = json.Unmarshal([]byte(content), &article); err != nil {
		return "", errors.AI(provider, fmt.Errorf("failed to parse AI response: %w", err))
	}

	return c.formatArticle(article), nil
}

func (c *OpenAIClient) formatArticle(article ArticleContent) string {
	return article.Content
}

type ArticleRequest struct {
	ID           int64
	SystemPrompt string
	UserPrompt   string
}

func getOpenAIModel(model string) openai.ChatModel {
	switch entities.AIModel(model) {
	case entities.ModelGPT4OMini:
		return openai.ChatModelGPT4oMini
	default:
		return model
	}
}
