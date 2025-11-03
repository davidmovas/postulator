package providers

import (
	"context"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"
)

var _ Service = (*service)(nil)

type service struct {
	repo   Repository
	logger *logger.Logger
}

func NewService(repo Repository, logger *logger.Logger) Service {
	return &service{
		repo: repo,
		logger: logger.
			WithScope("service").
			WithScope("providers"),
	}
}

func (s *service) CreateProvider(ctx context.Context, provider *entities.Provider) error {
	if err := s.validateProvider(provider); err != nil {
		return err
	}

	now := time.Now()
	provider.CreatedAt = now
	provider.UpdatedAt = now

	if err := s.repo.Create(ctx, provider); err != nil {
		s.logger.ErrorWithErr(err, "Failed to create provider")
		return err
	}

	s.logger.Info("Provider created successfully")
	return nil
}

func (s *service) GetProvider(ctx context.Context, id int64) (*entities.Provider, error) {
	provider, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get provider")
		return nil, err
	}

	s.logger.Debug("Provider retrieved")
	return provider, nil
}

func (s *service) ListProviders(ctx context.Context) ([]*entities.Provider, error) {
	providers, err := s.repo.GetAll(ctx)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to list providers")
		return nil, err
	}

	s.logger.Debug("Providers listed")
	return providers, nil
}

func (s *service) ListActiveProviders(ctx context.Context) ([]*entities.Provider, error) {
	providers, err := s.repo.GetActive(ctx)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to list active providers")
		return nil, err
	}

	s.logger.Debug("Active providers listed")
	return providers, nil
}

func (s *service) UpdateProvider(ctx context.Context, provider *entities.Provider) error {
	if err := s.validateProvider(provider); err != nil {
		return err
	}
	provider.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, provider); err != nil {
		s.logger.ErrorWithErr(err, "Failed to update provider")
		return err
	}

	s.logger.Info("Provider updated successfully")
	return nil
}

func (s *service) DeleteProvider(ctx context.Context, id int64) error {
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get provider for deletion")
		return err
	}

	if err = s.repo.Delete(ctx, id); err != nil {
		s.logger.ErrorWithErr(err, "Failed to delete provider")
		return err
	}

	s.logger.Info("Provider deleted successfully")
	return nil
}

func (s *service) SetProviderStatus(ctx context.Context, id int64, isActive bool) error {
	existingProvider, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get provider for status update")
		return err
	}

	existingProvider.IsActive = isActive
	existingProvider.UpdatedAt = time.Now()

	if err = s.repo.Update(ctx, existingProvider); err != nil {
		s.logger.ErrorWithErr(err, "Failed to update provider status")
		return err
	}

	status := "activated"
	if !isActive {
		status = "deactivated"
	}
	s.logger.Infof("Provider status updated successfully status: %s", status)
	return nil
}

func (s *service) GetAvailableModels(providerType entities.Type) ([]*entities.Model, error) {
	models := s.getDefaultModels(providerType)
	if len(models) == 0 {
		return nil, errors.Validation("Unsupported provider type")
	}

	s.logger.Debugf("Available models retrieved provider: %s count: %d", providerType, len(models))
	return models, nil
}

func (s *service) ValidateModel(providerType entities.Type, model string) error {
	availableModels, err := s.GetAvailableModels(providerType)
	if err != nil {
		return err
	}

	for _, availableModel := range availableModels {
		if availableModel.ID == model {
			s.logger.Debugf("Model validation successful provider: %s model: %s", providerType, model)
			return nil
		}
	}

	s.logger.Warnf("Model validation failed provider: %s model: %s", providerType, model)
	return errors.Validation("Unsupported model for provider type")
}

func (s *service) validateProvider(provider *entities.Provider) error {
	if strings.TrimSpace(provider.Name) == "" {
		return errors.Validation("Provider name is required")
	}

	if provider.Type == "" {
		return errors.Validation("Provider type is required")
	}

	validTypes := map[entities.Type]bool{
		entities.TypeOpenAI:    true,
		entities.TypeAnthropic: true,
		entities.TypeGoogle:    true,
	}

	if !validTypes[provider.Type] {
		return errors.Validation("Unsupported provider type")
	}

	if strings.TrimSpace(provider.APIKey) == "" {
		return errors.Validation("API key is required")
	}

	if strings.TrimSpace(provider.Model) == "" {
		return errors.Validation("Model is required")
	}
	if err := s.ValidateModel(provider.Type, provider.Model); err != nil {
		return err
	}

	return nil
}

func (s *service) getDefaultModels(providerType entities.Type) []*entities.Model {
	switch providerType {
	case entities.TypeOpenAI:
		return []*entities.Model{
			{
				ID:         "gpt-4",
				Name:       "GPT-4",
				Provider:   entities.TypeOpenAI,
				MaxTokens:  8192,
				InputCost:  0.03,
				OutputCost: 0.06,
			},
			{
				ID:         "gpt-3.5-turbo",
				Name:       "GPT-3.5 Turbo",
				Provider:   entities.TypeOpenAI,
				MaxTokens:  4096,
				InputCost:  0.0015,
				OutputCost: 0.002,
			},
		}
	case entities.TypeAnthropic:
		return []*entities.Model{
			{
				ID:         "claude-3-opus-20240229",
				Name:       "Claude 3 Opus",
				Provider:   entities.TypeAnthropic,
				MaxTokens:  200000,
				InputCost:  0.015,
				OutputCost: 0.075,
			},
			{
				ID:         "claude-3-sonnet-20240229",
				Name:       "Claude 3 Sonnet",
				Provider:   entities.TypeAnthropic,
				MaxTokens:  200000,
				InputCost:  0.003,
				OutputCost: 0.015,
			},
		}
	case entities.TypeGoogle:
		return []*entities.Model{
			{
				ID:         "gemini-pro",
				Name:       "Gemini Pro",
				Provider:   entities.TypeGoogle,
				MaxTokens:  32768,
				InputCost:  0.000125,
				OutputCost: 0.000375,
			},
		}
	default:
		return []*entities.Model{}
	}
}
