package entities

type AIModel string

// OpenAI Models
const (
	// GPT-4 Models

	ModelGPT4       AIModel = "gpt-4"
	ModelGPT4Turbo  AIModel = "gpt-4-turbo"
	ModelGPT4TurboP AIModel = "gpt-4-turbo-preview"
	ModelGPT4O      AIModel = "gpt-4o"
	ModelGPT4OMini  AIModel = "gpt-4o-mini"

	// GPT-3.5 Models

	ModelGPT35Turbo   AIModel = "gpt-3.5-turbo"
	ModelGPT35Turbo16 AIModel = "gpt-3.5-turbo-16k"

	// O1 Models

	ModelO1Preview AIModel = "o1-preview"
	ModelO1Mini    AIModel = "o1-mini"
)

// Anthropic Models
const (
	ModelClaude3Opus    AIModel = "claude-3-opus-20240229"
	ModelClaude3Sonnet  AIModel = "claude-3-sonnet-20240229"
	ModelClaude3Haiku   AIModel = "claude-3-haiku-20240307"
	ModelClaude35Sonnet AIModel = "claude-3-5-sonnet-20241022"
)

// Google Models
const (
	ModelGeminiPro     AIModel = "gemini-pro"
	ModelGemini15Pro   AIModel = "gemini-1.5-pro"
	ModelGemini15Flash AIModel = "gemini-1.5-flash"
)

// Provider type

type AIProviderType string

const (
	ProviderOpenAI    AIProviderType = "openai"
	ProviderAnthropic AIProviderType = "anthropic"
	ProviderGoogle    AIProviderType = "google"
)

// GetModelsByProvider returns available models for a specific provider
func GetModelsByProvider(provider AIProviderType) []AIModel {
	switch provider {
	case ProviderOpenAI:
		return []AIModel{
			ModelGPT4,
			ModelGPT4Turbo,
			ModelGPT4TurboP,
			ModelGPT4O,
			ModelGPT4OMini,
			ModelGPT35Turbo,
			ModelGPT35Turbo16,
			ModelO1Preview,
			ModelO1Mini,
		}
	case ProviderAnthropic:
		return []AIModel{
			ModelClaude3Opus,
			ModelClaude3Sonnet,
			ModelClaude3Haiku,
			ModelClaude35Sonnet,
		}
	case ProviderGoogle:
		return []AIModel{
			ModelGeminiPro,
			ModelGemini15Pro,
			ModelGemini15Flash,
		}
	default:
		return []AIModel{}
	}
}

func IsValidModel(provider AIProviderType, model AIModel) bool {
	models := GetModelsByProvider(provider)
	for _, m := range models {
		if m == model {
			return true
		}
	}
	return false
}

func GetProviderType(name string) AIProviderType {
	switch name {
	case "openai":
		return ProviderOpenAI
	case "anthropic":
		return ProviderAnthropic
	case "google":
		return ProviderGoogle
	default:
		return ""
	}
}
