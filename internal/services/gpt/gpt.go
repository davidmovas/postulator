package gpt

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"Postulator/internal/models"
	"Postulator/internal/repository"

	"github.com/invopop/jsonschema"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
)

var GTPModels = []string{
	"GPT-5",
	"GPT-5-Mini",
	"GPT-4",
	"GPT-4o",
	"GPT-4o-Mini",
}

// Service handles GPT API interactions
type Service struct {
	client   *openai.Client
	defaults ServiceDefaults
	repos    *repository.Repository
}

// ServiceDefaults contains default configuration
type ServiceDefaults struct {
	Model       string
	MaxTokens   int
	Temperature float64
	Timeout     time.Duration
}

// Config contains service configuration
type Config struct {
	APIKey      string
	Model       string
	MaxTokens   int
	Temperature float64
	Timeout     time.Duration
}

// GenerateArticleRequest contains parameters for article generation
type GenerateArticleRequest struct {
	Title       string  `json:"title"`
	Prompt      string  `json:"prompt"`
	Model       string  `json:"model,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
}

// ArticleResponse represents the structured response for article generation
type ArticleResponse struct {
	Title    string   `json:"title" jsonschema_description:"SEO-optimized article title"`
	Content  string   `json:"content" jsonschema_description:"Full HTML article content (minimum 800 words)"`
	Excerpt  string   `json:"excerpt" jsonschema_description:"Brief summary (150-200 characters)"`
	Keywords []string `json:"keywords" jsonschema_description:"SEO keywords array"`
	Tags     []string `json:"tags" jsonschema_description:"WordPress tags array"`
	Category string   `json:"category" jsonschema_description:"WordPress category name"`
}

// GenerateArticleResponse contains the complete response with metadata
type GenerateArticleResponse struct {
	Article    ArticleResponse `json:"article"`
	TokensUsed int             `json:"tokens_used"`
	Model      string          `json:"model"`
}

// NewService creates a new GPT service instance
func NewService(config Config, repos *repository.Repository) *Service {
	// Set defaults
	if config.Model == "" {
		config.Model = "gpt-4o-mini"
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 4000
	}
	if config.Temperature == 0 {
		config.Temperature = 0.7
	}
	if config.Timeout == 0 {
		config.Timeout = 2 * time.Minute
	}

	// Create OpenAI client
	client := openai.NewClient(
		option.WithAPIKey(config.APIKey),
	)

	return &Service{
		client: &client,
		repos:  repos,
		defaults: ServiceDefaults{
			Model:       config.Model,
			MaxTokens:   config.MaxTokens,
			Temperature: config.Temperature,
			Timeout:     config.Timeout,
		},
	}
}

// GenerateArticle generates an article with structured JSON output
func (s *Service) GenerateArticle(ctx context.Context, req GenerateArticleRequest) (*GenerateArticleResponse, error) {
	if req.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if req.Prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	// Set defaults if not provided
	model := req.Model
	if model == "" {
		model = s.defaults.Model
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = s.defaults.MaxTokens
	}

	temperature := req.Temperature
	if temperature == 0 {
		temperature = s.defaults.Temperature
	}

	// Convert model string to OpenAI model type
	openaiModel := s.parseModelType(model)

	// Generate JSON schema for structured output
	schema := GenerateSchema[ArticleResponse]()

	// Create schema parameter
	schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        "article_response",
		Description: openai.String("Generated article with structured metadata"),
		Schema:      schema,
		Strict:      openai.Bool(true),
	}

	// Build complete prompt using database prompts
	placeholders := map[string]string{
		"title":  req.Title,
		"prompt": req.Prompt,
	}

	systemPrompt, userPrompt, err := s.loadPrompts(ctx, placeholders)
	if err != nil {
		return nil, fmt.Errorf("failed to load prompts: %w", err)
	}

	// Create chat completion
	chat, err := s.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:       openaiModel,
		MaxTokens:   openai.Int(int64(maxTokens)),
		Temperature: openai.Float(temperature),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(userPrompt),
		},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
				JSONSchema: schemaParam,
			},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("OpenAI API call failed: %w", err)
	}

	// Parse the structured response
	if len(chat.Choices) == 0 {
		return nil, fmt.Errorf("no response choices received")
	}

	var article ArticleResponse
	if err = json.Unmarshal([]byte(chat.Choices[0].Message.Content), &article); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	response := &GenerateArticleResponse{
		Article:    article,
		TokensUsed: int(chat.Usage.TotalTokens),
		Model:      chat.Model,
	}

	return response, nil
}

// loadPrompts loads system and user prompts from database with placeholders replaced
func (s *Service) loadPrompts(ctx context.Context, placeholders map[string]string) (systemPrompt, userPrompt string, err error) {
	// Use hardcoded system prompt for now
	systemPrompt = "You are a professional content writer who creates high-quality, SEO-optimized articles for WordPress websites. You must respond with valid JSON matching the provided schema."

	// Use fallback user prompt
	userPrompt = s.buildFallbackPrompt(placeholders)

	return systemPrompt, userPrompt, nil
}

// replacePlaceholders replaces placeholders in prompt content
func (s *Service) replacePlaceholders(content string, placeholders map[string]string) string {
	result := content
	placeholderRegex := regexp.MustCompile(`\{\{(\w+)}}`)

	result = placeholderRegex.ReplaceAllStringFunc(result, func(match string) string {
		// Extract placeholder name (remove {{ and }})
		key := placeholderRegex.FindStringSubmatch(match)[1]
		if value, exists := placeholders[key]; exists {
			return value
		}
		// Return placeholder as-is if no replacement found
		return match
	})

	return result
}

// buildFallbackPrompt constructs a fallback prompt when database prompt is not available
func (s *Service) buildFallbackPrompt(placeholders map[string]string) string {
	var prompt strings.Builder

	if userPrompt, exists := placeholders["prompt"]; exists {
		prompt.WriteString(userPrompt)
	}
	prompt.WriteString("\n\n")

	if title, exists := placeholders["title"]; exists {
		prompt.WriteString(fmt.Sprintf("Article Title: %s\n", title))
	}

	prompt.WriteString("\nPlease generate a comprehensive article following these requirements:\n")
	prompt.WriteString("- Write a complete HTML article with minimum 800 words\n")
	prompt.WriteString("- Create an SEO-optimized title\n")
	prompt.WriteString("- Include a brief excerpt (150-200 characters)\n")
	prompt.WriteString("- Provide relevant SEO keywords\n")
	prompt.WriteString("- Suggest appropriate WordPress tags\n")
	prompt.WriteString("- Assign a suitable category\n")
	prompt.WriteString("\nRespond with valid JSON matching the provided schema.")

	return prompt.String()
}

// buildPrompt constructs the complete prompt for article generation
func (s *Service) buildPrompt(req GenerateArticleRequest) string {
	var prompt strings.Builder

	prompt.WriteString(req.Prompt)
	prompt.WriteString("\n\n")
	prompt.WriteString(fmt.Sprintf("Article Title: %s\n", req.Title))
	prompt.WriteString("\nPlease generate a comprehensive article following these requirements:\n")
	prompt.WriteString("- Write a complete HTML article with minimum 800 words\n")
	prompt.WriteString("- Create an SEO-optimized title\n")
	prompt.WriteString("- Include a brief excerpt (150-200 characters)\n")
	prompt.WriteString("- Provide relevant SEO keywords\n")
	prompt.WriteString("- Suggest appropriate WordPress tags\n")
	prompt.WriteString("- Assign a suitable category\n")
	prompt.WriteString("\nRespond with valid JSON matching the provided schema.")

	return prompt.String()
}

// parseModelType converts string model name to OpenAI model type
func (s *Service) parseModelType(model string) openai.ChatModel {
	switch strings.ToLower(model) {
	case "gpt-5":
		return openai.ChatModelGPT5
	case "gpt-5-mini":
		return openai.ChatModelGPT5Mini
	case "gpt-4o", "gpt-4-turbo":
		return openai.ChatModelGPT4o
	case "gpt-4o-mini":
		return openai.ChatModelGPT4oMini
	case "gpt-4":
		return openai.ChatModelGPT4
	default:
		return openai.ChatModelGPT4oMini
	}
}

// GenerateSchema creates a JSON schema for the given type T
func GenerateSchema[T any]() interface{} {
	// Structured Outputs uses a subset of JSON schema
	// These flags are necessary to comply with the subset
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	schema := reflector.Reflect(v)
	return schema
}

// Legacy methods for backward compatibility (kept for now)

// GenerateArticleFromTopic generates an article based on topic and site (legacy method)
func (s *Service) GenerateArticleFromTopic(ctx context.Context, topic *models.Topic, site *models.Site, customPrompt string) (*GenerateArticleResponse, error) {
	if topic == nil {
		return nil, fmt.Errorf("topic is required")
	}
	if site == nil {
		return nil, fmt.Errorf("site is required")
	}

	// Build prompt from topic
	prompt := customPrompt
	if prompt == "" {
		prompt = "" // Topics no longer have individual prompts
	}

	// Add topic context
	fullPrompt := fmt.Sprintf("%s\n\nTopic: %s\nKeywords: %s\nCategory: %s\nTarget Tags: %s\nWebsite: %s",
		prompt, topic.Title, topic.Keywords, topic.Category, topic.Tags, site.URL)

	req := GenerateArticleRequest{
		Title:  topic.Title,
		Prompt: fullPrompt,
		Model:  s.defaults.Model,
	}

	return s.GenerateArticle(ctx, req)
}
