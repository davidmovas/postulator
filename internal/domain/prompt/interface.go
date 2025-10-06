package prompt

import "context"

type Service interface {
	CreatePrompt(ctx context.Context, prompt *Prompt) error
	GetPrompt(ctx context.Context, id int64) (*Prompt, error)
	ListPrompts(ctx context.Context) ([]*Prompt, error)
	UpdatePrompt(ctx context.Context, prompt *Prompt) error
	DeletePrompt(ctx context.Context, id int64) error

	RenderPrompt(ctx context.Context, promptID int64, placeholders map[string]string) (system, user string, err error)
}

type Repository interface {
	Create(ctx context.Context, prompt *Prompt) error
	GetByID(ctx context.Context, id int64) (*Prompt, error)
	GetAll(ctx context.Context) ([]*Prompt, error)
	Update(ctx context.Context, prompt *Prompt) error
	Delete(ctx context.Context, id int64) error
}
