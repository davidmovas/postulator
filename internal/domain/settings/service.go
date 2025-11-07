package settings

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/davidmovas/postulator/internal/domain/entities"
	appErrors "github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"
)

type service struct {
	repo   Repository
	logger *logger.Logger
}

func NewService(repo Repository, logger *logger.Logger) Service {
	return &service{
		repo: repo,
		logger: logger.
			WithScope("service").
			WithScope("settings"),
	}
}

func (s *service) GetHealthCheckSettings(ctx context.Context) (*entities.HealthCheckSettings, error) {
	value, err := s.repo.Get(ctx, entities.SettingsKeyHealthCheck)
	if err != nil {
		var appErr *appErrors.AppError
		if errors.As(err, &appErr) && appErr.Code == appErrors.ErrCodeNotFound {
			s.logger.Info("Health check settings not found, returning defaults")
			return entities.DefaultHealthCheckSettings(), nil
		}
		s.logger.ErrorWithErr(err, "Failed to get health check settings")
		return nil, err
	}

	var settings entities.HealthCheckSettings
	if err = json.Unmarshal([]byte(value), &settings); err != nil {
		s.logger.ErrorWithErr(err, "Failed to unmarshal health check settings")
		return nil, appErrors.Internal(err)
	}

	return &settings, nil
}

func (s *service) UpdateHealthCheckSettings(ctx context.Context, settings *entities.HealthCheckSettings) error {
	if err := settings.Validate(); err != nil {
		s.logger.ErrorWithErr(err, "Invalid health check settings")
		return err
	}

	data, err := json.Marshal(settings)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to marshal health check settings")
		return appErrors.Internal(err)
	}

	if err = s.repo.Set(ctx, entities.SettingsKeyHealthCheck, string(data)); err != nil {
		s.logger.ErrorWithErr(err, "Failed to save health check settings")
		return err
	}

	s.logger.Info("Health check settings updated successfully")
	return nil
}
