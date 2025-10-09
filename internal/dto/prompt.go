package dto

import "Postulator/internal/domain/entities"

type Prompt struct {
	ID           int64    `json:"id"`
	Name         string   `json:"name"`
	SystemPrompt string   `json:"systemPrompt"`
	UserPrompt   string   `json:"userPrompt"`
	Placeholders []string `json:"placeholders"`
	CreatedAt    string   `json:"createdAt"`
	UpdatedAt    string   `json:"updatedAt"`
}

func FromPrompt(e *entities.Prompt) *Prompt {
	if e == nil {
		return nil
	}
	return &Prompt{
		ID:           e.ID,
		Name:         e.Name,
		SystemPrompt: e.SystemPrompt,
		UserPrompt:   e.UserPrompt,
		Placeholders: append([]string(nil), e.Placeholders...),
		CreatedAt:    e.CreatedAt.UTC().Format(timeLayout),
		UpdatedAt:    e.UpdatedAt.UTC().Format(timeLayout),
	}
}

func FromPrompts(items []*entities.Prompt) []*Prompt {
	out := make([]*Prompt, 0, len(items))
	for _, it := range items {
		out = append(out, FromPrompt(it))
	}
	return out
}
