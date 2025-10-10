package app

import (
	"Postulator/internal/domain/entities"
	"Postulator/internal/dto"
	"Postulator/pkg/ctx"
	"Postulator/pkg/errors"
)

func (a *App) CreatePrompt(p *dto.Prompt) *dto.Response[string] {
	if p == nil {
		return dtoErr[string](errors.Validation("prompt payload is required"))
	}

	entity := &entities.Prompt{
		ID:           p.ID,
		Name:         p.Name,
		SystemPrompt: p.SystemPrompt,
		UserPrompt:   p.UserPrompt,
	}

	if err := a.promptSvc.CreatePrompt(ctx.FastCtx(), entity); err != nil {
		return dtoErr[string](asAppErr(err))
	}

	return &dto.Response[string]{Success: true, Data: "created"}
}

func (a *App) GetPrompt(id int64) *dto.Response[*dto.Prompt] {
	pr, err := a.promptSvc.GetPrompt(ctx.FastCtx(), id)
	if err != nil {
		return dtoErr[*dto.Prompt](asAppErr(err))
	}

	return &dto.Response[*dto.Prompt]{Success: true, Data: dto.FromPrompt(pr)}
}

func (a *App) ListPrompts() *dto.Response[[]*dto.Prompt] {
	items, err := a.promptSvc.ListPrompts(ctx.FastCtx())
	if err != nil {
		return dtoErr[[]*dto.Prompt](asAppErr(err))
	}

	return &dto.Response[[]*dto.Prompt]{Success: true, Data: dto.FromPrompts(items)}
}

func (a *App) UpdatePrompt(p *dto.Prompt) *dto.Response[string] {
	if p == nil {
		return dtoErr[string](errors.Validation("prompt payload is required"))
	}
	entity := &entities.Prompt{ID: p.ID, Name: p.Name, SystemPrompt: p.SystemPrompt, UserPrompt: p.UserPrompt}
	if err := a.promptSvc.UpdatePrompt(ctx.FastCtx(), entity); err != nil {
		return dtoErr[string](asAppErr(err))
	}

	return &dto.Response[string]{Success: true, Data: "updated"}
}

func (a *App) DeletePrompt(id int64) *dto.Response[string] {
	if err := a.promptSvc.DeletePrompt(ctx.FastCtx(), id); err != nil {
		return dtoErr[string](asAppErr(err))
	}

	return &dto.Response[string]{Success: true, Data: "deleted"}
}

func (a *App) RenderPrompt(promptID int64, placeholders map[string]string) *dto.Response[*dto.PromptRenderResult] {
	sys, user, err := a.promptSvc.RenderPrompt(ctx.FastCtx(), promptID, placeholders)
	if err != nil {
		return dtoErr[*dto.PromptRenderResult](asAppErr(err))
	}

	return &dto.Response[*dto.PromptRenderResult]{Success: true, Data: &dto.PromptRenderResult{System: sys, User: user}}
}
