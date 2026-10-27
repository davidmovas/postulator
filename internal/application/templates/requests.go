package templates

import (
	"encoding/json"

	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

type CreateTemplateRequest struct {
	Scope  string                `json:"scope,omitempty"`
	SiteID *string               `json:"siteId,omitempty"`
	Name   string                `json:"name"`
	Spec   template.TemplateSpec `json:"spec"`
}

type CreateTemplateResponse struct {
	Template Template `json:"template"`
}

type UpdateTemplateRequest struct {
	ID   string                 `json:"id"`
	Name *string                `json:"name,omitempty"`
	Spec *template.TemplateSpec `json:"spec,omitempty"`
}

type UpdateTemplateResponse struct {
	Template Template `json:"template"`
}

type DeleteTemplateRequest struct {
	ID string `json:"id" description:"Template id"`
}

type DeleteTemplateResponse struct{}

type GetTemplateRequest struct {
	ID string `json:"id" description:"Template id"`
}

type GetTemplateResponse struct {
	Template  Template   `json:"template"`
	Overrides []Override `json:"overrides"`
}

type ListTemplatesRequest struct {
	dto.ListRequest
	Scope  string `json:"scope,omitempty" enum:"global,site" description:"Only this scope"`
	SiteID string `json:"siteId,omitempty" description:"Only this site's"`
}

type SetOverrideRequest struct {
	TemplateID string          `json:"templateId"`
	Scope      string          `json:"scope"`
	TargetID   string          `json:"targetId"`
	Patch      json.RawMessage `json:"patch"`
}

type SetOverrideResponse struct {
	Override Override `json:"override"`
}

type DeleteOverrideRequest struct {
	ID string `json:"id" description:"Override id"`
}

type DeleteOverrideResponse struct{}

type ResolveForPageRequest struct {
	PageID     string `json:"pageId" description:"Page id"`
	TemplateID string `json:"templateId,omitempty" description:"Template id to resolve instead; default the page's own"`
}

const (
	SourceRun  = "run"
	SourcePage = "page"
	SourceType = "type"
	SourceSite = "site"
)

type ResolveForPageResponse struct {
	TemplateID string                `json:"templateId"`
	Name       string                `json:"name"`
	Source     string                `json:"source"`
	SiteID     string                `json:"siteId"`
	Version    int                   `json:"version"`
	Spec       template.TemplateSpec `json:"spec"`
}
