package dto

import "github.com/davidmovas/postulator/internal/domain/entities"

type Provider struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	APIKey    string `json:"apiKey"`
	Model     string `json:"model"`
	IsActive  bool   `json:"isActive"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

func NewProvider(entity *entities.Provider) *Provider {
	p := &Provider{}
	return p.FromEntity(entity)
}

func (d *Provider) ToEntity() (*entities.Provider, error) {
	createdAt, err := StringToTime(d.CreatedAt)
	if err != nil {
		return nil, err
	}

	updatedAt, err := StringToTime(d.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &entities.Provider{
		ID:        d.ID,
		Name:      d.Name,
		Type:      entities.Type(d.Type),
		APIKey:    d.APIKey,
		Model:     d.Model,
		IsActive:  d.IsActive,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

func (d *Provider) FromEntity(entity *entities.Provider) *Provider {
	d.ID = entity.ID
	d.Name = entity.Name
	d.Type = string(entity.Type)
	d.APIKey = entity.APIKey
	d.Model = entity.Model
	d.IsActive = entity.IsActive
	d.CreatedAt = TimeToString(entity.CreatedAt)
	d.UpdatedAt = TimeToString(entity.UpdatedAt)
	return d
}

type Model struct {
	ID                   string  `json:"id"`
	Name                 string  `json:"name"`
	Provider             string  `json:"provider"`
	ContextWindow        int     `json:"contextWindow"`
	MaxOutputTokens      int     `json:"maxOutputTokens"`
	InputCost            float64 `json:"inputCost"`
	OutputCost           float64 `json:"outputCost"`
	RPM                  int     `json:"rpm"`
	TPM                  int     `json:"tpm"`
	UsesCompletionTokens bool    `json:"usesCompletionTokens"`
	SupportsWebSearch    bool    `json:"supportsWebSearch"`
}

func NewModel(entity *entities.Model) *Model {
	m := &Model{}
	return m.FromEntity(entity)
}

func (d *Model) ToEntity() *entities.Model {
	return &entities.Model{
		ID:                   d.ID,
		Name:                 d.Name,
		Provider:             entities.Type(d.Provider),
		ContextWindow:        d.ContextWindow,
		MaxOutputTokens:      d.MaxOutputTokens,
		InputCost:            d.InputCost,
		OutputCost:           d.OutputCost,
		RPM:                  d.RPM,
		TPM:                  d.TPM,
		UsesCompletionTokens: d.UsesCompletionTokens,
		SupportsWebSearch:    d.SupportsWebSearch,
	}
}

func (d *Model) FromEntity(entity *entities.Model) *Model {
	d.ID = entity.ID
	d.Name = entity.Name
	d.Provider = string(entity.Provider)
	d.ContextWindow = entity.ContextWindow
	d.MaxOutputTokens = entity.MaxOutputTokens
	d.InputCost = entity.InputCost
	d.OutputCost = entity.OutputCost
	d.RPM = entity.RPM
	d.TPM = entity.TPM
	d.UsesCompletionTokens = entity.UsesCompletionTokens
	d.SupportsWebSearch = entity.SupportsWebSearch
	return d
}
