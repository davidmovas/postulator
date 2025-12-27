package dto

import "github.com/davidmovas/postulator/internal/domain/entities"

// ContextFieldValue represents the configuration for a single context field
type ContextFieldValue struct {
	Enabled bool   `json:"enabled"`
	Value   string `json:"value,omitempty"`
}

// ContextConfig stores the configuration for context fields
type ContextConfig map[string]ContextFieldValue

// SelectOption represents an option for select-type context fields
type SelectOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// ContextFieldDefinition defines metadata for a context field
type ContextFieldDefinition struct {
	Key          string         `json:"key"`
	Label        string         `json:"label"`
	Description  string         `json:"description"`
	Type         string         `json:"type"`
	Options      []SelectOption `json:"options,omitempty"`
	DefaultValue string         `json:"defaultValue"`
	Required     bool           `json:"required"`
	Categories   []string       `json:"categories"`
	Group        string         `json:"group,omitempty"`
}

// NewContextFieldDefinition converts entity to DTO
func NewContextFieldDefinition(def *entities.ContextFieldDefinition) *ContextFieldDefinition {
	categories := make([]string, len(def.Categories))
	for i, cat := range def.Categories {
		categories[i] = string(cat)
	}

	options := make([]SelectOption, len(def.Options))
	for i, opt := range def.Options {
		options[i] = SelectOption{
			Value: opt.Value,
			Label: opt.Label,
		}
	}

	return &ContextFieldDefinition{
		Key:          def.Key,
		Label:        def.Label,
		Description:  def.Description,
		Type:         string(def.Type),
		Options:      options,
		DefaultValue: def.DefaultValue,
		Required:     def.Required,
		Categories:   categories,
		Group:        def.Group,
	}
}

// Prompt represents a prompt DTO with support for both v1 and v2 formats
type Prompt struct {
	ID           int64         `json:"id"`
	Name         string        `json:"name"`
	Category     string        `json:"category"`
	IsBuiltin    bool          `json:"isBuiltin"`
	Version      int           `json:"version"`
	Instructions string        `json:"instructions,omitempty"`
	ContextConfig ContextConfig `json:"contextConfig,omitempty"`
	// Legacy fields (v1 format)
	SystemPrompt string   `json:"systemPrompt,omitempty"`
	UserPrompt   string   `json:"userPrompt,omitempty"`
	Placeholders []string `json:"placeholders,omitempty"`
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

	// Convert DTO context config to entity
	var contextConfig entities.ContextConfig
	if d.ContextConfig != nil {
		contextConfig = make(entities.ContextConfig)
		for k, v := range d.ContextConfig {
			contextConfig[k] = entities.ContextFieldValue{
				Enabled: v.Enabled,
				Value:   v.Value,
			}
		}
	}

	return &entities.Prompt{
		ID:            d.ID,
		Name:          d.Name,
		Category:      entities.PromptCategory(d.Category),
		IsBuiltin:     d.IsBuiltin,
		Version:       d.Version,
		Instructions:  d.Instructions,
		ContextConfig: contextConfig,
		SystemPrompt:  d.SystemPrompt,
		UserPrompt:    d.UserPrompt,
		Placeholders:  d.Placeholders,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}, nil
}

func (d *Prompt) FromEntity(entity *entities.Prompt) *Prompt {
	d.ID = entity.ID
	d.Name = entity.Name
	d.Category = string(entity.Category)
	d.IsBuiltin = entity.IsBuiltin
	d.Version = entity.Version
	d.Instructions = entity.Instructions
	d.SystemPrompt = entity.SystemPrompt
	d.UserPrompt = entity.UserPrompt
	d.Placeholders = entity.Placeholders
	d.CreatedAt = TimeToString(entity.CreatedAt)
	d.UpdatedAt = TimeToString(entity.UpdatedAt)

	// Convert entity context config to DTO
	if entity.ContextConfig != nil {
		d.ContextConfig = make(ContextConfig)
		for k, v := range entity.ContextConfig {
			d.ContextConfig[k] = ContextFieldValue{
				Enabled: v.Enabled,
				Value:   v.Value,
			}
		}
	}

	return d
}

// ContextFieldsResponse is returned when fetching context fields for a category
type ContextFieldsResponse struct {
	Fields        []*ContextFieldDefinition `json:"fields"`
	DefaultConfig ContextConfig             `json:"defaultConfig"`
}

// NewContextFieldsResponse creates a response from entities
func NewContextFieldsResponse(fields []*entities.ContextFieldDefinition, defaultConfig entities.ContextConfig) *ContextFieldsResponse {
	dtoFields := make([]*ContextFieldDefinition, len(fields))
	for i, f := range fields {
		dtoFields[i] = NewContextFieldDefinition(f)
	}

	dtoConfig := make(ContextConfig)
	for k, v := range defaultConfig {
		dtoConfig[k] = ContextFieldValue{
			Enabled: v.Enabled,
			Value:   v.Value,
		}
	}

	return &ContextFieldsResponse{
		Fields:        dtoFields,
		DefaultConfig: dtoConfig,
	}
}
