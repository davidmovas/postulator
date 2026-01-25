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

// debugMode controls whether to print detailed debug logs for AI requests
// Set to true for local development, false for production builds
const debugMode = true // TODO: Set to false before production build

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
	contextWindow        int
	maxOutputTokens      int
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
	contextWindow := 128000      // Default fallback
	maxOutputTokens := 16384 * 2 // Default fallback
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

func (c *OpenAIClient) EstimateTokens(text string) int {
	// Average ~3.5 chars per token for mixed content (code, HTML, etc.)
	return (len(text) * 10) / 35
}

func (c *OpenAIClient) CalculateAvailableOutputTokens(systemPrompt, userPrompt string, desiredOutput int) int {
	inputTokens := c.EstimateTokens(systemPrompt) + c.EstimateTokens(userPrompt)

	jsonSchemaOverhead := 2000
	inputWithBuffer := int(float64(inputTokens)*1.25) + jsonSchemaOverhead

	availableInContext := c.contextWindow - inputWithBuffer

	maxPossible := availableInContext
	if c.maxOutputTokens < maxPossible {
		maxPossible = c.maxOutputTokens
	}
	if desiredOutput > 0 && desiredOutput < maxPossible {
		maxPossible = desiredOutput
	}

	if maxPossible < 100 {
		maxPossible = 100
	}

	return maxPossible
}

func (c *OpenAIClient) ValidateRequest(systemPrompt, userPrompt string, minRequiredOutput int) error {
	inputTokens := c.EstimateTokens(systemPrompt) + c.EstimateTokens(userPrompt)

	jsonSchemaOverhead := 2000
	inputWithBuffer := int(float64(inputTokens)*1.25) + jsonSchemaOverhead

	available := c.contextWindow - inputWithBuffer

	if available < minRequiredOutput {
		return fmt.Errorf("prompt too large: estimated %d input tokens (with buffer: %d), only %d tokens available for output (need at least %d). Context window: %d",
			inputTokens, inputWithBuffer, available, minRequiredOutput, c.contextWindow)
	}

	return nil
}

func (c *OpenAIClient) GenerateArticle(ctx context.Context, systemPrompt, userPrompt string, opts *GenerateArticleOptions) (*ArticleResult, error) {
	const desiredArticleTokens = 16384 * 2
	const minArticleTokens = 2000

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

	// ====== DETAILED DEBUG LOGGING ======
	if debugMode {
		inputEstimate := c.EstimateTokens(systemPrompt) + c.EstimateTokens(userPrompt)
		systemTokens := c.EstimateTokens(systemPrompt)
		userTokens := c.EstimateTokens(userPrompt)

		var debugLog strings.Builder
		debugLog.WriteString("\n")
		debugLog.WriteString("╔═══════════════════════════════════════════════════════════════════════════════════\n")
		debugLog.WriteString("║ [OPENAI DEBUG] GenerateArticle - REQUEST DETAILS\n")
		debugLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		debugLog.WriteString(fmt.Sprintf("║ Model: %s\n", c.modelName))
		debugLog.WriteString(fmt.Sprintf("║ Context Window: %d tokens\n", c.contextWindow))
		debugLog.WriteString(fmt.Sprintf("║ Max Output Tokens (model limit): %d tokens\n", c.maxOutputTokens))
		debugLog.WriteString(fmt.Sprintf("║ Desired Article Tokens: %d tokens\n", desiredArticleTokens))
		debugLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		debugLog.WriteString("║ INPUT ANALYSIS:\n")
		debugLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		debugLog.WriteString(fmt.Sprintf("║ System Prompt Length: %d chars → ~%d tokens (estimated)\n", len(systemPrompt), systemTokens))
		debugLog.WriteString(fmt.Sprintf("║ User Prompt Length: %d chars → ~%d tokens (estimated)\n", len(userPrompt), userTokens))
		debugLog.WriteString(fmt.Sprintf("║ TOTAL INPUT: %d chars → ~%d tokens (estimated)\n", len(systemPrompt)+len(userPrompt), inputEstimate))
		debugLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		debugLog.WriteString("║ SYSTEM PROMPT (full):\n")
		debugLog.WriteString("╠───────────────────────────────────────────────────────────────────────────────────\n")
		debugLog.WriteString(fmt.Sprintf("║ %s\n", strings.ReplaceAll(systemPrompt, "\n", "\n║ ")))
		debugLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		debugLog.WriteString("║ USER PROMPT PREVIEW (first 800 chars):\n")
		debugLog.WriteString("╠───────────────────────────────────────────────────────────────────────────────────\n")
		userPreview := userPrompt
		if len(userPreview) > 800 {
			userPreview = userPreview[:800] + "..."
		}
		debugLog.WriteString(fmt.Sprintf("║ %s\n", strings.ReplaceAll(userPreview, "\n", "\n║ ")))
		debugLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		debugLog.WriteString("║ TOKEN CALCULATION:\n")
		debugLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		jsonSchemaOverhead := 2000
		inputWithBuffer := int(float64(inputEstimate)*1.25) + jsonSchemaOverhead
		availableInContext := c.contextWindow - inputWithBuffer
		debugLog.WriteString(fmt.Sprintf("║ Input Estimate: %d tokens\n", inputEstimate))
		debugLog.WriteString(fmt.Sprintf("║ Input with Buffer (×1.25): %d tokens\n", int(float64(inputEstimate)*1.25)))
		debugLog.WriteString(fmt.Sprintf("║ JSON Schema Overhead: %d tokens\n", jsonSchemaOverhead))
		debugLog.WriteString(fmt.Sprintf("║ Total Input + Overhead: %d tokens\n", inputWithBuffer))
		debugLog.WriteString(fmt.Sprintf("║ Available in Context Window: %d tokens\n", availableInContext))
		debugLog.WriteString(fmt.Sprintf("║ Max Tokens Set for Request: %d tokens\n", maxTokens))
		debugLog.WriteString("╚═══════════════════════════════════════════════════════════════════════════════════\n")
		fmt.Print(debugLog.String())
	}

	chat, err := c.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, errors.AI(providerName, fmt.Errorf("API error: %w", err))
	}

	if len(chat.Choices) == 0 {
		return nil, errors.AI(providerName, fmt.Errorf("no response from API"))
	}

	choice := chat.Choices[0]
	finishReason := string(choice.FinishReason)

	// ====== DETAILED DEBUG LOGGING - RESPONSE ======
	if debugMode {
		inputEstimate := c.EstimateTokens(systemPrompt) + c.EstimateTokens(userPrompt)

		var responseLog strings.Builder
		responseLog.WriteString("\n")
		responseLog.WriteString("╔═══════════════════════════════════════════════════════════════════════════════════\n")
		responseLog.WriteString("║ [OPENAI DEBUG] GenerateArticle - RESPONSE DETAILS\n")
		responseLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		responseLog.WriteString("║ ACTUAL TOKEN USAGE (from API):\n")
		responseLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		responseLog.WriteString(fmt.Sprintf("║ Input Tokens (actual): %d tokens\n", chat.Usage.PromptTokens))
		responseLog.WriteString(fmt.Sprintf("║ Output Tokens (actual): %d tokens\n", chat.Usage.CompletionTokens))
		responseLog.WriteString(fmt.Sprintf("║ Total Tokens (actual): %d tokens\n", chat.Usage.TotalTokens))
		responseLog.WriteString(fmt.Sprintf("║ Finish Reason: %s\n", finishReason))
		responseLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		responseLog.WriteString("║ COMPARISON (Estimated vs Actual):\n")
		responseLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		responseLog.WriteString(fmt.Sprintf("║ Input: Estimated %d → Actual %d (diff: %+d)\n",
			inputEstimate, chat.Usage.PromptTokens, int(chat.Usage.PromptTokens)-inputEstimate))
		responseLog.WriteString(fmt.Sprintf("║ Output: Max %d → Used %d (%.1f%% utilized)\n",
			maxTokens, chat.Usage.CompletionTokens, float64(chat.Usage.CompletionTokens)/float64(maxTokens)*100))

		if finishReason == "length" {
			responseLog.WriteString("║ ⚠️  WARNING: Response truncated! Max tokens limit reached!\n")
		}

		responseLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		responseLog.WriteString("║ RAW API RESPONSE PREVIEW (first 1000 chars):\n")
		responseLog.WriteString("╠───────────────────────────────────────────────────────────────────────────────────\n")
		rawContent := choice.Message.Content
		rawPreview := rawContent
		if len(rawPreview) > 1000 {
			rawPreview = rawPreview[:1000] + "..."
		}
		responseLog.WriteString(fmt.Sprintf("║ Length: %d chars\n", len(rawContent)))
		responseLog.WriteString("╠───────────────────────────────────────────────────────────────────────────────────\n")
		responseLog.WriteString(fmt.Sprintf("║ %s\n", strings.ReplaceAll(rawPreview, "\n", "\n║ ")))
		responseLog.WriteString("╚═══════════════════════════════════════════════════════════════════════════════════\n")
		fmt.Print(responseLog.String())
	}

	// Check finish_reason for truncation
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

	// ====== DETAILED DEBUG LOGGING - PARSED RESULT ======
	if debugMode {
		var resultLog strings.Builder
		resultLog.WriteString("\n")
		resultLog.WriteString("╔═══════════════════════════════════════════════════════════════════════════════════\n")
		resultLog.WriteString("║ [OPENAI DEBUG] GenerateArticle - PARSED RESULT\n")
		resultLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		resultLog.WriteString(fmt.Sprintf("║ Title: %s\n", article.Title))
		resultLog.WriteString(fmt.Sprintf("║ Title Length: %d chars\n", len(article.Title)))
		resultLog.WriteString("╠───────────────────────────────────────────────────────────────────────────────────\n")
		resultLog.WriteString(fmt.Sprintf("║ Excerpt Length: %d chars\n", len(article.Excerpt)))
		if article.Excerpt != "" {
			excerptPreview := article.Excerpt
			if len(excerptPreview) > 200 {
				excerptPreview = excerptPreview[:200] + "..."
			}
			resultLog.WriteString(fmt.Sprintf("║ Excerpt: %s\n", strings.ReplaceAll(excerptPreview, "\n", " ")))
		}
		resultLog.WriteString("╠───────────────────────────────────────────────────────────────────────────────────\n")
		resultLog.WriteString(fmt.Sprintf("║ Content Length: %d chars (~%d tokens)\n", len(article.Content), c.EstimateTokens(article.Content)))
		resultLog.WriteString("║ Content Preview (first 600 chars):\n")
		contentPreview := article.Content
		if len(contentPreview) > 600 {
			contentPreview = contentPreview[:600] + "..."
		}
		resultLog.WriteString(fmt.Sprintf("║ %s\n", strings.ReplaceAll(contentPreview, "\n", "\n║ ")))
		resultLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		resultLog.WriteString("║ FINAL COST CALCULATION:\n")
		resultLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		modelInfo := GetModelInfo(entities.TypeOpenAI, c.modelName)
		if modelInfo != nil {
			resultLog.WriteString(fmt.Sprintf("║ Input Tokens: %d × $%.2f/1M = $%.6f\n", inputTokens, modelInfo.InputCost, float64(inputTokens)/1_000_000*modelInfo.InputCost))
			resultLog.WriteString(fmt.Sprintf("║ Output Tokens: %d × $%.2f/1M = $%.6f\n", outputTokens, modelInfo.OutputCost, float64(outputTokens)/1_000_000*modelInfo.OutputCost))
		}
		resultLog.WriteString(fmt.Sprintf("║ TOTAL COST: $%.6f\n", cost))
		resultLog.WriteString("╚═══════════════════════════════════════════════════════════════════════════════════\n\n")
		fmt.Print(resultLog.String())
	}

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
		params.MaxCompletionTokens = openaiSDK.Int(16384)
	} else {
		params.MaxTokens = openaiSDK.Int(16384)
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
	const minRequiredTokens = 2000
	const tokensPerLink = 80
	const baseTokens = 16384

	maxLinks := request.MaxOutgoing
	if request.MaxIncoming > maxLinks {
		maxLinks = request.MaxIncoming
	}
	if maxLinks == 0 {
		maxLinks = 8
	}

	potentialLinks := len(request.Nodes) * maxLinks
	desiredTokens := baseTokens + (potentialLinks * tokensPerLink)

	if desiredTokens > c.maxOutputTokens {
		desiredTokens = c.maxOutputTokens
	}

	systemPrompt := request.SystemPrompt
	userPrompt := request.UserPrompt

	// Validate request first - check that prompt isn't too large
	if err := c.ValidateRequest(systemPrompt, userPrompt, minRequiredTokens); err != nil {
		return nil, errors.AI(providerName, fmt.Errorf(
			"prompt too large for link suggestions (%d nodes): %w. Try reducing batch size or simplifying node data",
			len(request.Nodes), err))
	}

	// Calculate actual available tokens dynamically
	maxTokens := c.CalculateAvailableOutputTokens(systemPrompt, userPrompt, desiredTokens)

	// ====== DETAILED DEBUG LOGGING - LINK SUGGESTIONS REQUEST ======
	if debugMode {
		inputEstimate := c.EstimateTokens(systemPrompt) + c.EstimateTokens(userPrompt)
		systemTokens := c.EstimateTokens(systemPrompt)
		userTokens := c.EstimateTokens(userPrompt)

		var debugLog strings.Builder
		debugLog.WriteString("\n")
		debugLog.WriteString("╔═══════════════════════════════════════════════════════════════════════════════════\n")
		debugLog.WriteString("║ [OPENAI DEBUG] GenerateLinkSuggestions - REQUEST DETAILS\n")
		debugLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		debugLog.WriteString(fmt.Sprintf("║ Model: %s\n", c.modelName))
		debugLog.WriteString(fmt.Sprintf("║ Context Window: %d tokens\n", c.contextWindow))
		debugLog.WriteString(fmt.Sprintf("║ Max Output Tokens (model limit): %d tokens\n", c.maxOutputTokens))
		debugLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		debugLog.WriteString("║ LINK SUGGESTION TASK:\n")
		debugLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		debugLog.WriteString(fmt.Sprintf("║ Nodes to Process: %d\n", len(request.Nodes)))
		debugLog.WriteString(fmt.Sprintf("║ Max Outgoing Links: %d per page\n", request.MaxOutgoing))
		debugLog.WriteString(fmt.Sprintf("║ Max Incoming Links: %d per page\n", request.MaxIncoming))
		debugLog.WriteString("╠───────────────────────────────────────────────────────────────────────────────────\n")
		debugLog.WriteString("║ NODES (first 10):\n")
		maxNodesToShow := len(request.Nodes)
		if maxNodesToShow > 10 {
			maxNodesToShow = 10
		}
		for i := 0; i < maxNodesToShow; i++ {
			node := request.Nodes[i]
			keywords := ""
			if len(node.Keywords) > 0 {
				kw := node.Keywords
				if len(kw) > 3 {
					kw = kw[:3]
				}
				keywords = fmt.Sprintf(" [%s]", strings.Join(kw, ", "))
			}
			debugLog.WriteString(fmt.Sprintf("║   %d. [ID:%d] %s %s%s [%d→ %d←]\n",
				i+1, node.ID, node.Title, node.Path, keywords, node.OutgoingCount, node.IncomingCount))
		}
		if len(request.Nodes) > 10 {
			debugLog.WriteString(fmt.Sprintf("║   ... and %d more nodes\n", len(request.Nodes)-10))
		}
		debugLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		debugLog.WriteString("║ INPUT ANALYSIS:\n")
		debugLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		debugLog.WriteString(fmt.Sprintf("║ System Prompt Length: %d chars → ~%d tokens (estimated)\n", len(systemPrompt), systemTokens))
		debugLog.WriteString(fmt.Sprintf("║ User Prompt Length: %d chars → ~%d tokens (estimated)\n", len(userPrompt), userTokens))
		debugLog.WriteString(fmt.Sprintf("║ TOTAL INPUT: %d chars → ~%d tokens (estimated)\n", len(systemPrompt)+len(userPrompt), inputEstimate))
		debugLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		debugLog.WriteString("║ SYSTEM PROMPT (full):\n")
		debugLog.WriteString("╠───────────────────────────────────────────────────────────────────────────────────\n")
		debugLog.WriteString(fmt.Sprintf("║ %s\n", strings.ReplaceAll(systemPrompt, "\n", "\n║ ")))
		debugLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		debugLog.WriteString("║ USER PROMPT PREVIEW (first 1500 chars):\n")
		debugLog.WriteString("╠───────────────────────────────────────────────────────────────────────────────────\n")
		userPreview := userPrompt
		if len(userPreview) > 1500 {
			userPreview = userPreview[:1500] + "..."
		}
		debugLog.WriteString(fmt.Sprintf("║ %s\n", strings.ReplaceAll(userPreview, "\n", "\n║ ")))
		debugLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		debugLog.WriteString("║ TOKEN CALCULATION:\n")
		debugLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		jsonSchemaOverhead := 2000
		inputWithBuffer := int(float64(inputEstimate)*1.25) + jsonSchemaOverhead
		availableInContext := c.contextWindow - inputWithBuffer
		debugLog.WriteString(fmt.Sprintf("║ Input Estimate: %d tokens\n", inputEstimate))
		debugLog.WriteString(fmt.Sprintf("║ Input with Buffer (×1.25): %d tokens\n", int(float64(inputEstimate)*1.25)))
		debugLog.WriteString(fmt.Sprintf("║ JSON Schema Overhead: %d tokens\n", jsonSchemaOverhead))
		debugLog.WriteString(fmt.Sprintf("║ Total Input + Overhead: %d tokens\n", inputWithBuffer))
		debugLog.WriteString(fmt.Sprintf("║ Available in Context Window: %d tokens\n", availableInContext))
		debugLog.WriteString(fmt.Sprintf("║ Max Tokens Set for Request: %d tokens\n", maxTokens))
		debugLog.WriteString(fmt.Sprintf("║ Desired Output Tokens: %d tokens\n", desiredTokens))
		debugLog.WriteString("╚═══════════════════════════════════════════════════════════════════════════════════\n")
		fmt.Print(debugLog.String())
	}

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

	// ====== DETAILED DEBUG LOGGING - LINK SUGGESTIONS RESPONSE (PART 1) ======
	if debugMode {
		inputEstimate := c.EstimateTokens(systemPrompt) + c.EstimateTokens(userPrompt)

		var responseLog strings.Builder
		responseLog.WriteString("\n")
		responseLog.WriteString("╔═══════════════════════════════════════════════════════════════════════════════════\n")
		responseLog.WriteString("║ [OPENAI DEBUG] GenerateLinkSuggestions - RESPONSE DETAILS\n")
		responseLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		responseLog.WriteString("║ ACTUAL TOKEN USAGE (from API):\n")
		responseLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		responseLog.WriteString(fmt.Sprintf("║ Input Tokens (actual): %d tokens\n", chat.Usage.PromptTokens))
		responseLog.WriteString(fmt.Sprintf("║ Output Tokens (actual): %d tokens\n", chat.Usage.CompletionTokens))
		responseLog.WriteString(fmt.Sprintf("║ Total Tokens (actual): %d tokens\n", chat.Usage.TotalTokens))
		responseLog.WriteString(fmt.Sprintf("║ Finish Reason: %s\n", finishReason))
		responseLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		responseLog.WriteString("║ COMPARISON (Estimated vs Actual):\n")
		responseLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		responseLog.WriteString(fmt.Sprintf("║ Input: Estimated %d → Actual %d (diff: %+d)\n",
			inputEstimate, chat.Usage.PromptTokens, int(chat.Usage.PromptTokens)-inputEstimate))
		responseLog.WriteString(fmt.Sprintf("║ Output: Max %d → Used %d (%.1f%% utilized)\n",
			maxTokens, chat.Usage.CompletionTokens, float64(chat.Usage.CompletionTokens)/float64(maxTokens)*100))

		if finishReason == "length" {
			responseLog.WriteString("║ ⚠️  WARNING: Response truncated! Max tokens limit reached!\n")
		}

		responseLog.WriteString("╚═══════════════════════════════════════════════════════════════════════════════════\n")
		fmt.Print(responseLog.String())
	}

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

	// ====== DETAILED DEBUG LOGGING - LINK SUGGESTIONS RESULT ======
	if debugMode {
		var resultLog strings.Builder
		resultLog.WriteString("\n")
		resultLog.WriteString("╔═══════════════════════════════════════════════════════════════════════════════════\n")
		resultLog.WriteString("║ [OPENAI DEBUG] GenerateLinkSuggestions - PARSED RESULT\n")
		resultLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		resultLog.WriteString(fmt.Sprintf("║ Link Suggestions Generated: %d\n", len(result.Links)))
		resultLog.WriteString("╠───────────────────────────────────────────────────────────────────────────────────\n")
		resultLog.WriteString("║ SUGGESTED LINKS:\n")
		maxLinksToShow := len(result.Links)
		if maxLinksToShow > 20 {
			maxLinksToShow = 20
		}
		for i := 0; i < maxLinksToShow; i++ {
			link := result.Links[i]
			resultLog.WriteString(fmt.Sprintf("║   %d. [ID:%d] → [ID:%d]\n", i+1, link.SourceNodeID, link.TargetNodeID))
			if link.AnchorText != "" {
				resultLog.WriteString(fmt.Sprintf("║      Anchor: \"%s\"\n", link.AnchorText))
			}
		}
		if len(result.Links) > 20 {
			resultLog.WriteString(fmt.Sprintf("║   ... and %d more links\n", len(result.Links)-20))
		}
		resultLog.WriteString("╠───────────────────────────────────────────────────────────────────────────────────\n")
		if result.Explanation != "" {
			resultLog.WriteString("║ EXPLANATION:\n")
			explanation := result.Explanation
			if len(explanation) > 500 {
				explanation = explanation[:500] + "..."
			}
			resultLog.WriteString(fmt.Sprintf("║ %s\n", strings.ReplaceAll(explanation, "\n", "\n║ ")))
			resultLog.WriteString("╠───────────────────────────────────────────────────────────────────────────────────\n")
		}
		resultLog.WriteString("║ FINAL COST CALCULATION:\n")
		resultLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		modelInfo := GetModelInfo(entities.TypeOpenAI, c.modelName)
		if modelInfo != nil {
			resultLog.WriteString(fmt.Sprintf("║ Input Tokens: %d × $%.2f/1M = $%.6f\n", inputTokens, modelInfo.InputCost, float64(inputTokens)/1_000_000*modelInfo.InputCost))
			resultLog.WriteString(fmt.Sprintf("║ Output Tokens: %d × $%.2f/1M = $%.6f\n", outputTokens, modelInfo.OutputCost, float64(outputTokens)/1_000_000*modelInfo.OutputCost))
		}
		resultLog.WriteString(fmt.Sprintf("║ TOTAL COST: $%.6f\n", cost))
		resultLog.WriteString("╚═══════════════════════════════════════════════════════════════════════════════════\n\n")
		fmt.Print(resultLog.String())
	}

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
	userPrompt := request.UserPrompt

	contentTokenEstimate := c.EstimateTokens(request.Content)
	desiredOutputTokens := int(float64(contentTokenEstimate) * 1.3)
	if desiredOutputTokens < 8192 {
		desiredOutputTokens = 8192
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

	// ====== DETAILED DEBUG LOGGING - INSERT LINKS REQUEST ======
	if debugMode {
		inputEstimate := c.EstimateTokens(systemPrompt) + c.EstimateTokens(userPrompt)
		systemTokens := c.EstimateTokens(systemPrompt)
		userTokens := c.EstimateTokens(userPrompt)

		var debugLog strings.Builder
		debugLog.WriteString("\n")
		debugLog.WriteString("╔═══════════════════════════════════════════════════════════════════════════════════\n")
		debugLog.WriteString("║ [OPENAI DEBUG] InsertLinks - REQUEST DETAILS\n")
		debugLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		debugLog.WriteString(fmt.Sprintf("║ Model: %s\n", c.modelName))
		debugLog.WriteString(fmt.Sprintf("║ Context Window: %d tokens\n", c.contextWindow))
		debugLog.WriteString(fmt.Sprintf("║ Max Output Tokens (model limit): %d tokens\n", c.maxOutputTokens))
		debugLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		debugLog.WriteString("║ LINK INSERTION TASK:\n")
		debugLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		debugLog.WriteString(fmt.Sprintf("║ Page: \"%s\" %s\n", request.PageTitle, request.PagePath))
		debugLog.WriteString(fmt.Sprintf("║ Content Length: %d chars (~%d tokens)\n", len(request.Content), c.EstimateTokens(request.Content)))
		debugLog.WriteString(fmt.Sprintf("║ Links to Insert: %d\n", len(request.Links)))
		debugLog.WriteString("╠───────────────────────────────────────────────────────────────────────────────────\n")
		debugLog.WriteString("║ LINKS TO INSERT:\n")
		for i, link := range request.Links {
			debugLog.WriteString(fmt.Sprintf("║   %d. → %s \"%s\"\n", i+1, link.TargetPath, link.TargetTitle))
			if link.AnchorText != nil && *link.AnchorText != "" {
				debugLog.WriteString(fmt.Sprintf("║      Anchor: \"%s\"\n", *link.AnchorText))
			} else {
				debugLog.WriteString("║      Anchor: find suitable text\n")
			}
		}
		debugLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		debugLog.WriteString("║ INPUT ANALYSIS:\n")
		debugLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		debugLog.WriteString(fmt.Sprintf("║ System Prompt Length: %d chars → ~%d tokens (estimated)\n", len(systemPrompt), systemTokens))
		debugLog.WriteString(fmt.Sprintf("║ User Prompt Length: %d chars → ~%d tokens (estimated)\n", len(userPrompt), userTokens))
		debugLog.WriteString(fmt.Sprintf("║ TOTAL INPUT: %d chars → ~%d tokens (estimated)\n", len(systemPrompt)+len(userPrompt), inputEstimate))
		debugLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		debugLog.WriteString("║ SYSTEM PROMPT (full):\n")
		debugLog.WriteString("╠───────────────────────────────────────────────────────────────────────────────────\n")
		debugLog.WriteString(fmt.Sprintf("║ %s\n", strings.ReplaceAll(systemPrompt, "\n", "\n║ ")))
		debugLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		debugLog.WriteString("║ USER PROMPT PREVIEW (first 1500 chars):\n")
		debugLog.WriteString("╠───────────────────────────────────────────────────────────────────────────────────\n")
		userPreview := userPrompt
		if len(userPreview) > 1500 {
			userPreview = userPreview[:1500] + "..."
		}
		debugLog.WriteString(fmt.Sprintf("║ %s\n", strings.ReplaceAll(userPreview, "\n", "\n║ ")))
		debugLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		debugLog.WriteString("║ TOKEN CALCULATION:\n")
		debugLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		jsonSchemaOverhead := 2000
		inputWithBuffer := int(float64(inputEstimate)*1.25) + jsonSchemaOverhead
		availableInContext := c.contextWindow - inputWithBuffer
		debugLog.WriteString(fmt.Sprintf("║ Input Estimate: %d tokens\n", inputEstimate))
		debugLog.WriteString(fmt.Sprintf("║ Input with Buffer (×1.25): %d tokens\n", int(float64(inputEstimate)*1.25)))
		debugLog.WriteString(fmt.Sprintf("║ JSON Schema Overhead: %d tokens\n", jsonSchemaOverhead))
		debugLog.WriteString(fmt.Sprintf("║ Total Input + Overhead: %d tokens\n", inputWithBuffer))
		debugLog.WriteString(fmt.Sprintf("║ Available in Context Window: %d tokens\n", availableInContext))
		debugLog.WriteString(fmt.Sprintf("║ Max Tokens Set for Request: %d tokens\n", maxTokens))
		debugLog.WriteString(fmt.Sprintf("║ Desired Output Tokens: %d tokens\n", desiredOutputTokens))
		debugLog.WriteString("╚═══════════════════════════════════════════════════════════════════════════════════\n")
		fmt.Print(debugLog.String())
	}

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
				"Content size: %d chars. Try using a model with larger context window",
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

	// ====== DETAILED DEBUG LOGGING - INSERT LINKS RESPONSE ======
	if debugMode {
		inputEstimate := c.EstimateTokens(systemPrompt) + c.EstimateTokens(userPrompt)

		var responseLog strings.Builder
		responseLog.WriteString("\n")
		responseLog.WriteString("╔═══════════════════════════════════════════════════════════════════════════════════\n")
		responseLog.WriteString("║ [OPENAI DEBUG] InsertLinks - RESPONSE DETAILS\n")
		responseLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		responseLog.WriteString("║ ACTUAL TOKEN USAGE (from API):\n")
		responseLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		responseLog.WriteString(fmt.Sprintf("║ Input Tokens (actual): %d tokens\n", chat.Usage.PromptTokens))
		responseLog.WriteString(fmt.Sprintf("║ Output Tokens (actual): %d tokens\n", chat.Usage.CompletionTokens))
		responseLog.WriteString(fmt.Sprintf("║ Total Tokens (actual): %d tokens\n", chat.Usage.TotalTokens))
		responseLog.WriteString(fmt.Sprintf("║ Finish Reason: %s\n", finishReason))
		responseLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		responseLog.WriteString("║ COMPARISON (Estimated vs Actual):\n")
		responseLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		responseLog.WriteString(fmt.Sprintf("║ Input: Estimated %d → Actual %d (diff: %+d)\n",
			inputEstimate, chat.Usage.PromptTokens, int(chat.Usage.PromptTokens)-inputEstimate))
		responseLog.WriteString(fmt.Sprintf("║ Output: Max %d → Used %d (%.1f%% utilized)\n",
			maxTokens, chat.Usage.CompletionTokens, float64(chat.Usage.CompletionTokens)/float64(maxTokens)*100))

		responseLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		responseLog.WriteString("║ LINK INSERTION RESULT:\n")
		responseLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		responseLog.WriteString(fmt.Sprintf("║ Links Requested: %d\n", len(request.Links)))
		responseLog.WriteString(fmt.Sprintf("║ Links Applied: %d\n", result.LinksApplied))
		successRate := float64(result.LinksApplied) / float64(len(request.Links)) * 100
		responseLog.WriteString(fmt.Sprintf("║ Success Rate: %.1f%%\n", successRate))
		responseLog.WriteString("╠───────────────────────────────────────────────────────────────────────────────────\n")
		responseLog.WriteString(fmt.Sprintf("║ Original Content Length: %d chars\n", len(request.Content)))
		responseLog.WriteString(fmt.Sprintf("║ Modified Content Length: %d chars\n", len(result.Content)))
		deltaChars := len(result.Content) - len(request.Content)
		responseLog.WriteString(fmt.Sprintf("║ Delta: %+d chars\n", deltaChars))
		responseLog.WriteString("╠───────────────────────────────────────────────────────────────────────────────────\n")
		responseLog.WriteString("║ Modified Content Preview (first 1000 chars):\n")
		responseLog.WriteString("╠───────────────────────────────────────────────────────────────────────────────────\n")
		modifiedPreview := result.Content
		if len(modifiedPreview) > 1000 {
			modifiedPreview = modifiedPreview[:1000] + "..."
		}
		responseLog.WriteString(fmt.Sprintf("║ %s\n", strings.ReplaceAll(modifiedPreview, "\n", "\n║ ")))
		responseLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		responseLog.WriteString("║ FINAL COST CALCULATION:\n")
		responseLog.WriteString("╠═══════════════════════════════════════════════════════════════════════════════════\n")
		modelInfo := GetModelInfo(entities.TypeOpenAI, c.modelName)
		if modelInfo != nil {
			responseLog.WriteString(fmt.Sprintf("║ Input Tokens: %d × $%.2f/1M = $%.6f\n", inputTokens, modelInfo.InputCost, float64(inputTokens)/1_000_000*modelInfo.InputCost))
			responseLog.WriteString(fmt.Sprintf("║ Output Tokens: %d × $%.2f/1M = $%.6f\n", outputTokens, modelInfo.OutputCost, float64(outputTokens)/1_000_000*modelInfo.OutputCost))
		}
		responseLog.WriteString(fmt.Sprintf("║ TOTAL COST: $%.6f\n", cost))
		responseLog.WriteString("╚═══════════════════════════════════════════════════════════════════════════════════\n\n")
		fmt.Print(responseLog.String())
	}

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
