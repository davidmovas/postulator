package handlers

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/prompts"
	"github.com/davidmovas/postulator/internal/dto"
	"github.com/davidmovas/postulator/pkg/logger"
)

type PromptsHandler struct {
	promptsService prompts.Service
	logger         *logger.Logger
}

func NewPromptsHandler(promptsService prompts.Service, logger *logger.Logger) *PromptsHandler {
	return &PromptsHandler{
		promptsService: promptsService,
		logger:         logger,
	}
}

func (h *PromptsHandler) CreatePrompt(ctx context.Context, prompt *dto.Prompt) *dto.Response[string] {
	entity, err := prompt.ToEntity()
	if err != nil {
		return fail[string](err)
	}

	if err = h.promptsService.CreatePrompt(ctx, entity); err != nil {
		return fail[string](err)
	}

	return ok("Prompt created successfully")
}

func (h *PromptsHandler) GetPrompt(ctx context.Context, id int64) *dto.Response[*dto.Prompt] {
	prompt, err := h.promptsService.GetPrompt(ctx, id)
	if err != nil {
		return fail[*dto.Prompt](err)
	}

	return ok(dto.NewPrompt(prompt))
}

func (h *PromptsHandler) ListPrompts(ctx context.Context) *dto.Response[[]*dto.Prompt] {
	listPrompts, err := h.promptsService.ListPrompts(ctx)
	if err != nil {
		return fail[[]*dto.Prompt](err)
	}

	var dtoPrompts []*dto.Prompt
	for _, prompt := range listPrompts {
		dtoPrompts = append(dtoPrompts, dto.NewPrompt(prompt))
	}

	return ok(dtoPrompts)
}

func (h *PromptsHandler) UpdatePrompt(ctx context.Context, prompt *dto.Prompt) *dto.Response[string] {
	entity, err := prompt.ToEntity()
	if err != nil {
		return fail[string](err)
	}

	if err = h.promptsService.UpdatePrompt(ctx, entity); err != nil {
		return fail[string](err)
	}

	return ok("Prompt updated successfully")
}

func (h *PromptsHandler) DeletePrompt(ctx context.Context, id int64) *dto.Response[string] {
	if err := h.promptsService.DeletePrompt(ctx, id); err != nil {
		return fail[string](err)
	}

	return ok("Prompt deleted successfully")
}
