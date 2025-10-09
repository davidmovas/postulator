package prompt

import (
	"Postulator/internal/domain/entities"
	"Postulator/pkg/di"
	"Postulator/pkg/errors"
	"Postulator/pkg/logger"
	"context"
	"fmt"
	"regexp"
	"strings"
)

var _ IService = (*Service)(nil)

var placeholderRegex = regexp.MustCompile(`\{\{(\w+)}}`)

type Service struct {
	repo   IRepository
	logger *logger.Logger
}

func NewService(c di.Container) (*Service, error) {
	var l *logger.Logger
	if err := c.Resolve(&l); err != nil {
		return nil, err
	}

	repo, err := NewPromptRepository(c)
	if err != nil {
		return nil, err
	}

	return &Service{
		repo:   repo,
		logger: l,
	}, nil
}

func (s *Service) CreatePrompt(ctx context.Context, prompt *entities.Prompt) error {
	if err := s.validatePrompt(prompt); err != nil {
		return err
	}

	prompt.Placeholders = s.extractPlaceholders(prompt.SystemPrompt, prompt.UserPrompt)

	return s.repo.Create(ctx, prompt)
}

func (s *Service) GetPrompt(ctx context.Context, id int64) (*entities.Prompt, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ListPrompts(ctx context.Context) ([]*entities.Prompt, error) {
	return s.repo.GetAll(ctx)
}

func (s *Service) UpdatePrompt(ctx context.Context, prompt *entities.Prompt) error {
	if err := s.validatePrompt(prompt); err != nil {
		return err
	}

	prompt.Placeholders = s.extractPlaceholders(prompt.SystemPrompt, prompt.UserPrompt)

	return s.repo.Update(ctx, prompt)
}

func (s *Service) DeletePrompt(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) RenderPrompt(ctx context.Context, promptID int64, placeholders map[string]string) (system, user string, err error) {
	prompt, err := s.repo.GetByID(ctx, promptID)
	if err != nil {
		return "", "", err
	}

	if err = s.validatePlaceholders(prompt, placeholders); err != nil {
		return "", "", err
	}

	system = s.renderText(prompt.SystemPrompt, placeholders)

	user = s.renderText(prompt.UserPrompt, placeholders)

	s.logger.Debugf("Rendered prompt %d with %d placeholders", promptID, len(placeholders))

	return system, user, nil
}

func (s *Service) extractPlaceholders(texts ...string) []string {
	placeholderSet := make(map[string]bool)

	for _, text := range texts {
		matches := placeholderRegex.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) > 1 {
				placeholderSet[match[1]] = true
			}
		}
	}

	placeholders := make([]string, 0, len(placeholderSet))
	for placeholder := range placeholderSet {
		placeholders = append(placeholders, placeholder)
	}

	return placeholders
}

func (s *Service) renderText(text string, placeholders map[string]string) string {
	result := text

	for key, value := range placeholders {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}

func (s *Service) validatePrompt(prompt *entities.Prompt) error {
	if prompt.Name == "" {
		return errors.Validation("prompt name cannot be empty")
	}

	if prompt.SystemPrompt == "" && prompt.UserPrompt == "" {
		return errors.Validation("prompt must have at least system or user prompt text")
	}

	return nil
}

func (s *Service) validatePlaceholders(prompt *entities.Prompt, providedPlaceholders map[string]string) error {
	if len(prompt.Placeholders) == 0 {
		return nil
	}

	missingPlaceholders := make([]string, 0)

	for _, requiredPlaceholder := range prompt.Placeholders {
		if value, ok := providedPlaceholders[requiredPlaceholder]; !ok || value == "" {
			missingPlaceholders = append(missingPlaceholders, requiredPlaceholder)
		}
	}

	if len(missingPlaceholders) > 0 {
		return errors.Validation(fmt.Sprintf("missing required placeholders: %s", strings.Join(missingPlaceholders, ", ")))
	}

	return nil
}
