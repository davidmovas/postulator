package gpt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"Postulator/internal/models"
)

// Service handles GPT API interactions
type Service struct {
	apiKey     string
	apiURL     string
	model      string
	maxTokens  int
	timeout    time.Duration
	httpClient *http.Client
}

// Config holds GPT service configuration
type Config struct {
	APIKey    string
	Model     string
	MaxTokens int
	Timeout   time.Duration
}

// NewService creates a new GPT service instance
func NewService(config Config) *Service {
	if config.Model == "" {
		config.Model = "gpt-3.5-turbo"
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 4000
	}
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}

	return &Service{
		apiKey:    config.APIKey,
		apiURL:    "https://api.openai.com/v1/chat/completions",
		model:     config.Model,
		maxTokens: config.MaxTokens,
		timeout:   config.Timeout,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// ChatCompletionRequest represents the request to ChatGPT API
type ChatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionResponse represents the response from ChatGPT API
type ChatCompletionResponse struct {
	ID      string    `json:"id"`
	Object  string    `json:"object"`
	Created int64     `json:"created"`
	Model   string    `json:"model"`
	Choices []Choice  `json:"choices"`
	Usage   Usage     `json:"usage"`
	Error   *APIError `json:"error,omitempty"`
}

// Choice represents a completion choice
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// APIError represents an API error
type APIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// GenerateArticleRequest contains parameters for article generation
type GenerateArticleRequest struct {
	Topic        *models.Topic `json:"topic"`
	Site         *models.Site  `json:"site"`
	CustomPrompt string        `json:"custom_prompt,omitempty"`
	Temperature  float64       `json:"temperature,omitempty"`
}

// GenerateArticleResponse contains the generated article data
type GenerateArticleResponse struct {
	Title      string `json:"title"`
	Content    string `json:"content"`
	Excerpt    string `json:"excerpt"`
	Keywords   string `json:"keywords"`
	Tags       string `json:"tags"`
	Category   string `json:"category"`
	TokensUsed int    `json:"tokens_used"`
	Model      string `json:"model"`
}

// GenerateArticle generates an article based on the given topic and site
func (s *Service) GenerateArticle(ctx context.Context, req GenerateArticleRequest) (*GenerateArticleResponse, error) {
	if req.Topic == nil {
		return nil, fmt.Errorf("topic is required")
	}
	if req.Site == nil {
		return nil, fmt.Errorf("site is required")
	}

	// Build the prompt
	prompt := s.buildArticlePrompt(req)

	// Create chat completion request
	chatReq := ChatCompletionRequest{
		Model:       s.model,
		MaxTokens:   s.maxTokens,
		Temperature: req.Temperature,
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are a professional content writer who creates high-quality, SEO-optimized articles for WordPress websites. Your responses should be in JSON format with the specified structure.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	// Make API call
	response, err := s.makeAPICall(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("API call failed: %w", err)
	}

	// Parse the response
	articleResponse, err := s.parseArticleResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return articleResponse, nil
}

// buildArticlePrompt constructs the prompt for article generation
func (s *Service) buildArticlePrompt(req GenerateArticleRequest) string {
	var prompt strings.Builder

	if req.CustomPrompt != "" {
		prompt.WriteString(req.CustomPrompt)
		prompt.WriteString("\n\n")
	} else {
		prompt.WriteString(req.Topic.Prompt)
		prompt.WriteString("\n\n")
	}

	prompt.WriteString(fmt.Sprintf("Topic: %s\n", req.Topic.Title))
	prompt.WriteString(fmt.Sprintf("Description: %s\n", req.Topic.Description))
	prompt.WriteString(fmt.Sprintf("Keywords: %s\n", req.Topic.Keywords))
	prompt.WriteString(fmt.Sprintf("Category: %s\n", req.Topic.Category))
	prompt.WriteString(fmt.Sprintf("Target Tags: %s\n", req.Topic.Tags))
	prompt.WriteString(fmt.Sprintf("Website: %s\n\n", req.Site.URL))

	prompt.WriteString("Please generate a comprehensive article and respond with a JSON object containing:\n")
	prompt.WriteString("- title: SEO-optimized article title\n")
	prompt.WriteString("- content: Full HTML article content (minimum 800 words)\n")
	prompt.WriteString("- excerpt: Brief summary (150-200 characters)\n")
	prompt.WriteString("- keywords: SEO keywords (comma-separated)\n")
	prompt.WriteString("- tags: WordPress tags (comma-separated)\n")
	prompt.WriteString("- category: WordPress category name\n")

	return prompt.String()
}

// makeAPICall performs the HTTP request to GPT API
func (s *Service) makeAPICall(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error) {
	// Marshal request
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)

	// Make the request
	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var response ChatCompletionResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check for API errors
	if response.Error != nil {
		return nil, fmt.Errorf("API error: %s", response.Error.Message)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return &response, nil
}

// parseArticleResponse extracts article data from GPT response
func (s *Service) parseArticleResponse(response *ChatCompletionResponse) (*GenerateArticleResponse, error) {
	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	content := response.Choices[0].Message.Content

	// Try to parse as JSON
	var articleData map[string]interface{}
	if err := json.Unmarshal([]byte(content), &articleData); err != nil {
		// If JSON parsing fails, create a basic response
		return &GenerateArticleResponse{
			Title:      "Generated Article",
			Content:    content,
			Excerpt:    "Generated content excerpt",
			Keywords:   "",
			Tags:       "",
			Category:   "General",
			TokensUsed: response.Usage.TotalTokens,
			Model:      response.Model,
		}, nil
	}

	// Extract fields from JSON
	return &GenerateArticleResponse{
		Title:      getStringField(articleData, "title"),
		Content:    getStringField(articleData, "content"),
		Excerpt:    getStringField(articleData, "excerpt"),
		Keywords:   getStringField(articleData, "keywords"),
		Tags:       getStringField(articleData, "tags"),
		Category:   getStringField(articleData, "category"),
		TokensUsed: response.Usage.TotalTokens,
		Model:      response.Model,
	}, nil
}

// getStringField safely extracts a string field from a map
func getStringField(data map[string]interface{}, field string) string {
	if value, exists := data[field]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// ValidateConfig validates the service configuration
func (s *Service) ValidateConfig() error {
	if s.apiKey == "" {
		return fmt.Errorf("API key is required")
	}
	if s.model == "" {
		return fmt.Errorf("model is required")
	}
	if s.maxTokens <= 0 {
		return fmt.Errorf("max tokens must be positive")
	}
	return nil
}

// TestConnection tests the connection to GPT API
func (s *Service) TestConnection(ctx context.Context) error {
	req := ChatCompletionRequest{
		Model:     s.model,
		MaxTokens: 10,
		Messages: []Message{
			{
				Role:    "user",
				Content: "Hello",
			},
		},
	}

	_, err := s.makeAPICall(ctx, req)
	return err
}
