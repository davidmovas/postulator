package pagetypes

import "github.com/davidmovas/postulator/internal/domain/field"

type ListRequest struct {
	SiteID string `json:"siteId"`
}

type ListResponse struct {
	Types  []Type  `json:"types"`
	Fields []Field `json:"fields"`
}

type GetRequest struct {
	ID string `json:"id"`
}

type CreateRequest struct {
	SiteID     string  `json:"siteId"`
	Name       string  `json:"name"`
	Preset     string  `json:"preset"`
	TemplateID *string `json:"templateId,omitempty"`
}

type SaveRequest struct {
	ID         string  `json:"id"`
	Name       *string `json:"name,omitempty"`
	TemplateID *string `json:"templateId,omitempty"`
}

type DeleteRequest struct {
	ID         string `json:"id"`
	ReassignTo string `json:"reassignTo"`
}

type DeleteResponse struct {
	Moved int `json:"moved"`
}

type SetScopeRequest struct {
	PageTypeID  string           `json:"pageTypeId"`
	Field       string           `json:"field"`
	Enabled     bool             `json:"enabled"`
	AIWrites    bool             `json:"aiWrites"`
	Instruction string           `json:"instruction"`
	MinLength   int              `json:"minLength"`
	MaxLength   int              `json:"maxLength"`
	Rewrites    bool             `json:"rewrites"`
	Links       field.LinkPolicy `json:"links"`
	Label       string           `json:"label"`
	Provider    string           `json:"provider"`
}
