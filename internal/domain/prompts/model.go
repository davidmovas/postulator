package prompts

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/entities"
)

type Repository interface {
	Create(ctx context.Context, prompt *entities.Prompt) error
	GetByID(ctx context.Context, id int64) (*entities.Prompt, error)
	GetAll(ctx context.Context) ([]*entities.Prompt, error)
	GetByCategory(ctx context.Context, category entities.PromptCategory) ([]*entities.Prompt, error)
	Update(ctx context.Context, prompt *entities.Prompt) error
	Delete(ctx context.Context, id int64) error
}

type Service interface {
	CreatePrompt(ctx context.Context, prompt *entities.Prompt) error
	GetPrompt(ctx context.Context, id int64) (*entities.Prompt, error)
	ListPrompts(ctx context.Context) ([]*entities.Prompt, error)
	ListPromptsByCategory(ctx context.Context, category entities.PromptCategory) ([]*entities.Prompt, error)
	UpdatePrompt(ctx context.Context, prompt *entities.Prompt) error
	DeletePrompt(ctx context.Context, id int64) error

	RenderPrompt(ctx context.Context, promptID int64, placeholders map[string]string) (system, user string, err error)
	ValidatePlaceholders(prompt *entities.Prompt, provided map[string]string) error
}
