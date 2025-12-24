package dto

import "github.com/davidmovas/postulator/internal/domain/entities"

type Prompt struct {
	ID           int64    `json:"id"`
	Name         string   `json:"name"`
	Category     string   `json:"category"`
	IsBuiltin    bool     `json:"isBuiltin"`
	SystemPrompt string   `json:"systemPrompt"`
	UserPrompt   string   `json:"userPrompt"`
	Placeholders []string `json:"placeholders"`
	CreatedAt    string   `json:"createdAt"`
	UpdatedAt    string   `json:"updatedAt"`
}

func NewPrompt(prompt *entities.Prompt) *Prompt {
	p := &Prompt{}
	return p.FromEntity(prompt)
}

func (d *Prompt) ToEntity() (*entities.Prompt, error) {
	createdAt, err := StringToTime(d.CreatedAt)
	if err != nil {
		return nil, err
	}

	updatedAt, err := StringToTime(d.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &entities.Prompt{
		ID:           d.ID,
		Name:         d.Name,
		Category:     entities.PromptCategory(d.Category),
		IsBuiltin:    d.IsBuiltin,
		SystemPrompt: d.SystemPrompt,
		UserPrompt:   d.UserPrompt,
		Placeholders: d.Placeholders,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}, nil
}

func (d *Prompt) FromEntity(entity *entities.Prompt) *Prompt {
	d.ID = entity.ID
	d.Name = entity.Name
	d.Category = string(entity.Category)
	d.IsBuiltin = entity.IsBuiltin
	d.SystemPrompt = entity.SystemPrompt
	d.UserPrompt = entity.UserPrompt
	d.Placeholders = entity.Placeholders
	d.CreatedAt = TimeToString(entity.CreatedAt)
	d.UpdatedAt = TimeToString(entity.UpdatedAt)
	return d
}
