package aiprovider

import (
	"Postulator/internal/domain/entities"
	"Postulator/pkg/di"
	"Postulator/pkg/errors"
	"Postulator/pkg/logger"
	"context"
	"strings"
)

var _ IService = (*Service)(nil)

type Service struct {
	repo   IRepository
	logger *logger.Logger
}

func NewService(c di.Container) (*Service, error) {
	var l *logger.Logger
	if err := c.Resolve(&l); err != nil {
		return nil, err
	}

	repo, err := NewRepository(c)
	if err != nil {
		return nil, err
	}

	return &Service{
		repo:   repo,
		logger: l,
	}, nil
}

func (s *Service) CreateProvider(ctx context.Context, provider *entities.AIProvider) error {
	if err := s.validateProvider(provider); err != nil {
		return err
	}

	s.logger.Infof("Creating AI provider: %s (model: %s)", provider.Name, provider.Model)

	return s.repo.Create(ctx, provider)
}

func (s *Service) GetProvider(ctx context.Context, id int64) (*entities.AIProvider, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ListProviders(ctx context.Context) ([]*entities.AIProvider, error) {
	return s.repo.GetAll(ctx)
}

func (s *Service) ListActiveProviders(ctx context.Context) ([]*entities.AIProvider, error) {
	return s.repo.GetActive(ctx)
}

func (s *Service) UpdateProvider(ctx context.Context, provider *entities.AIProvider) error {
	if err := s.validateProvider(provider); err != nil {
		return err
	}

	s.logger.Infof("Updating AI provider %d: %s (model: %s, active: %v)", provider.ID, provider.Name, provider.Model, provider.IsActive)

	return s.repo.Update(ctx, provider)
}

func (s *Service) DeleteProvider(ctx context.Context, id int64) error {
	s.logger.Infof("Deleting AI provider %d", id)

	return s.repo.Delete(ctx, id)
}

func (s *Service) SetProviderStatus(ctx context.Context, id int64, isActive bool) error {
	provider, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	provider.IsActive = isActive

	s.logger.Infof("Setting AI provider %d status to active=%v", id, isActive)

	return s.repo.Update(ctx, provider)
}

func (s *Service) GetAvailableModels(providerName string) []entities.AIModel {
	providerType := entities.GetProviderType(strings.ToLower(strings.TrimSpace(providerName)))
	if providerType == "" {
		s.logger.Warnf("Unknown provider type: %s", providerName)
		return []entities.AIModel{}
	}

	return entities.GetModelsByProvider(providerType)
}

func (s *Service) ValidateModel(providerName string, model string) error {
	model = strings.TrimSpace(model)
	providerName = strings.TrimSpace(providerName)

	if model == "" {
		return errors.Validation("model cannot be empty")
	}

	providerType := entities.GetProviderType(strings.ToLower(providerName))
	if providerType == "" {
		return errors.Validation("unknown provider: " + providerName)
	}

	if !entities.IsValidModel(providerType, entities.AIModel(model)) {
		return errors.Validation("model '" + model + "' is not valid for provider '" + providerName + "'")
	}

	return nil
}

func (s *Service) validateProvider(provider *entities.AIProvider) error {
	provider.Name = strings.TrimSpace(provider.Name)

	if provider.Name == "" {
		return errors.Validation("AI provider name cannot be empty")
	}

	if provider.APIKey == "" {
		return errors.Validation("AI provider API key cannot be empty")
	}

	provider.Provider = strings.TrimSpace(strings.ToLower(provider.Provider))
	if provider.Provider == "" {
		return errors.Validation("AI provider name cannot be empty")
	}

	provider.Model = strings.TrimSpace(provider.Model)

	if provider.Model == "" {
		return errors.Validation("AI provider model cannot be empty")
	}

	// Validate that the model is valid for this provider
	if err := s.ValidateModel(provider.Provider, provider.Model); err != nil {
		return err
	}

	return nil
}
