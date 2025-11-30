package providers

import (
	"context"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/deletion"
	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/infra/ai"
	"github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"
)

var _ Service = (*service)(nil)

type service struct {
	repo              Repository
	deletionValidator *deletion.Validator
	logger            *logger.Logger
}

func NewService(repo Repository, deletionValidator *deletion.Validator, logger *logger.Logger) Service {
	return &service{
		repo:              repo,
		deletionValidator: deletionValidator,
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

	localProvider, err := s.repo.GetByID(ctx, provider.ID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get provider for update")
		return err
	}

	if provider.APIKey == "" {
		provider.APIKey = localProvider.APIKey
	}

	provider.UpdatedAt = time.Now()

	if err = s.repo.Update(ctx, provider); err != nil {
		s.logger.ErrorWithErr(err, "Failed to update provider")
		return err
	}

	s.logger.Info("Provider updated successfully")
	return nil
}

func (s *service) DeleteProvider(ctx context.Context, id int64) error {
	provider, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get provider for deletion")
		return err
	}

	if err = s.deletionValidator.CanDeleteProvider(ctx, id, provider.Name); err != nil {
		if conflictErr, ok := err.(*deletion.ConflictError); ok {
			s.logger.Warnf("Cannot delete provider %d: %s", id, conflictErr.Error())
			return errors.ConflictWithContext(conflictErr.UserMessage(), map[string]any{
				"entity_type":  conflictErr.EntityType,
				"entity_id":    conflictErr.EntityID,
				"dependencies": conflictErr.DependencyNames(),
			})
		}
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
	models := ai.GetAvailableModels(providerType)
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

	if strings.TrimSpace(provider.Model) == "" {
		return errors.Validation("Model is required")
	}
	if err := s.ValidateModel(provider.Type, provider.Model); err != nil {
		return err
	}

	return nil
}
