package ai

import (
	"fmt"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/pkg/errors"
)

func CreateClient(provider *entities.Provider) (Client, error) {
	if provider == nil {
		return nil, errors.Validation("provider is required")
	}

	if !provider.IsActive {
		return nil, errors.Validation("provider is not active")
	}

	if provider.APIKey == "" {
		return nil, errors.Validation("API key is required")
	}

	if !ValidateModel(provider.Type, provider.Model) {
		return nil, errors.Validation(fmt.Sprintf("invalid model %s for provider %s", provider.Model, provider.Type))
	}

	switch provider.Type {
	case entities.TypeOpenAI:
		return NewOpenAIClient(Config{
			APIKey: provider.APIKey,
			Model:  provider.Model,
		})

	case entities.TypeAnthropic:
		return NewOpenAIClient(Config{
			APIKey: provider.APIKey,
			Model:  provider.Model,
		})

	case entities.TypeGoogle:
		return NewOpenAIClient(Config{
			APIKey: provider.APIKey,
			Model:  provider.Model,
		})

	default:
		return nil, errors.Validation(fmt.Sprintf("unsupported provider type: %s", provider.Type))
	}
}

func GetProviderModels() map[entities.Type][]*entities.Model {
	cp := make(map[entities.Type][]*entities.Model)

	for provider, models := range availableModels {
		cpModels := make([]*entities.Model, len(models))
		copy(cpModels, models)

		cp[provider] = cpModels
	}

	return availableModels
}

func GetAvailableModels(providerType entities.Type) []*entities.Model {
	models, exists := availableModels[providerType]
	if !exists {
		return nil
	}

	result := make([]*entities.Model, len(models))
	copy(result, models)
	return result
}

func ValidateModel(providerType entities.Type, model string) bool {
	models := GetAvailableModels(providerType)
	for _, m := range models {
		if m.ID == model {
			return true
		}
	}
	return false
}
