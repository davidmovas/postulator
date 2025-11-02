package prompts

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
		repo:   repo,
		logger: logger.WithScope("service").WithScope("prompts"),
	}
}

func (s *service) CreatePrompt(ctx context.Context, prompt *entities.Prompt) error {
	if err := s.validatePrompt(prompt); err != nil {
		return err
	}

	prompt.CreatedAt = time.Now()
	prompt.UpdatedAt = time.Now()

	if err := s.repo.Create(ctx, prompt); err != nil {
		s.logger.ErrorWithErr(err, "Failed to create prompt")
		return err
	}

	s.logger.Info("Prompt created successfully")
	return nil
}

func (s *service) GetPrompt(ctx context.Context, id int64) (*entities.Prompt, error) {
	prompt, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get prompt")
		return nil, err
	}

	s.logger.Debug("Prompt retrieved")
	return prompt, nil
}

func (s *service) ListPrompts(ctx context.Context) ([]*entities.Prompt, error) {
	prompts, err := s.repo.GetAll(ctx)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to list prompts")
		return nil, err
	}

	s.logger.Debug("Prompts listed")
	return prompts, nil
}

func (s *service) UpdatePrompt(ctx context.Context, prompt *entities.Prompt) error {
	if err := s.validatePrompt(prompt); err != nil {
		return err
	}

	prompt.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, prompt); err != nil {
		s.logger.ErrorWithErr(err, "Failed to update prompt")
		return err
	}

	s.logger.Info("Prompt updated successfully")
	return nil
}

func (s *service) DeletePrompt(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.ErrorWithErr(err, "Failed to delete prompt")
		return err
	}

	s.logger.Info("Prompt deleted successfully")
	return nil
}

func (s *service) RenderPrompt(ctx context.Context, promptID int64, placeholders map[string]string) (system, user string, err error) {
	prompt, err := s.repo.GetByID(ctx, promptID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get prompt for rendering")
		return "", "", err
	}

	if err = s.ValidatePlaceholders(prompt, placeholders); err != nil {
		s.logger.ErrorWithErr(err, "Placeholder validation failed")
		return "", "", err
	}

	system = s.renderTemplate(prompt.SystemPrompt, placeholders)
	user = s.renderTemplate(prompt.UserPrompt, placeholders)

	s.logger.Debug("Prompt rendered successfully")
	return system, user, nil
}

func (s *service) ValidatePlaceholders(prompt *entities.Prompt, provided map[string]string) error {
	for _, placeholder := range prompt.Placeholders {
		value, exists := provided[placeholder]
		if !exists || strings.TrimSpace(value) == "" {
			return errors.Validation("Missing value for placeholder: " + placeholder)
		}
	}

	return nil
}

func (s *service) validatePrompt(prompt *entities.Prompt) error {
	if strings.TrimSpace(prompt.Name) == "" {
		return errors.Validation("Prompt name is required")
	}

	if strings.TrimSpace(prompt.SystemPrompt) == "" {
		return errors.Validation("System prompt is required")
	}

	if strings.TrimSpace(prompt.UserPrompt) == "" {
		return errors.Validation("User prompt is required")
	}

	return nil
}

func (s *service) renderTemplate(template string, placeholders map[string]string) string {
	result := template
	for placeholder, value := range placeholders {
		placeholderPattern := "{{" + placeholder + "}}"
		result = strings.ReplaceAll(result, placeholderPattern, value)
	}
	return result
}
