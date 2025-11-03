package handlers

import (
	"github.com/davidmovas/postulator/internal/domain/prompts"
	"github.com/davidmovas/postulator/internal/dto"
	"github.com/davidmovas/postulator/pkg/ctx"
)

type PromptsHandler struct {
	service prompts.Service
}

func NewPromptsHandler(service prompts.Service) *PromptsHandler {
	return &PromptsHandler{
		service: service,
	}
}

func (h *PromptsHandler) CreatePrompt(prompt *dto.Prompt) *dto.Response[string] {
	entity, err := prompt.ToEntity()
	if err != nil {
		return fail[string](err)
	}

	if err = h.service.CreatePrompt(ctx.FastCtx(), entity); err != nil {
		return fail[string](err)
	}

	return ok("Prompt created successfully")
}

func (h *PromptsHandler) GetPrompt(id int64) *dto.Response[*dto.Prompt] {
	prompt, err := h.service.GetPrompt(ctx.FastCtx(), id)
	if err != nil {
		return fail[*dto.Prompt](err)
	}

	return ok(dto.NewPrompt(prompt))
}

func (h *PromptsHandler) ListPrompts() *dto.Response[[]*dto.Prompt] {
	listPrompts, err := h.service.ListPrompts(ctx.FastCtx())
	if err != nil {
		return fail[[]*dto.Prompt](err)
	}

	var dtoPrompts []*dto.Prompt
	for _, prompt := range listPrompts {
		dtoPrompts = append(dtoPrompts, dto.NewPrompt(prompt))
	}

	return ok(dtoPrompts)
}

func (h *PromptsHandler) UpdatePrompt(prompt *dto.Prompt) *dto.Response[string] {
	entity, err := prompt.ToEntity()
	if err != nil {
		return fail[string](err)
	}

	if err = h.service.UpdatePrompt(ctx.FastCtx(), entity); err != nil {
		return fail[string](err)
	}

	return ok("Prompt updated successfully")
}

func (h *PromptsHandler) DeletePrompt(id int64) *dto.Response[string] {
	if err := h.service.DeletePrompt(ctx.FastCtx(), id); err != nil {
		return fail[string](err)
	}

	return ok("Prompt deleted successfully")
}
