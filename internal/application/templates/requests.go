package templates

import (
	"encoding/json"

	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

type CreateTemplateRequest struct {
	Scope    string                `json:"scope,omitempty"`
	SiteID   *string               `json:"siteId,omitempty"`
	Name     string                `json:"name"`
	PageKind string                `json:"pageKind"`
	Spec     template.TemplateSpec `json:"spec"`
}

type CreateTemplateResponse struct {
	Template Template `json:"template"`
}

type UpdateTemplateRequest struct {
	ID       string                 `json:"id"`
	Name     *string                `json:"name,omitempty"`
	PageKind *string                `json:"pageKind,omitempty"`
	Spec     *template.TemplateSpec `json:"spec,omitempty"`
}

type UpdateTemplateResponse struct {
	Template Template `json:"template"`
}

type DeleteTemplateRequest struct {
	ID string `json:"id"`
}

type DeleteTemplateResponse struct{}

type GetTemplateRequest struct {
	ID string `json:"id"`
}

type GetTemplateResponse struct {
	Template  Template   `json:"template"`
	Overrides []Override `json:"overrides"`
}

type ListTemplatesRequest struct {
	dto.ListRequest
	Scope    string `json:"scope,omitempty"`
	SiteID   string `json:"siteId,omitempty"`
	PageKind string `json:"pageKind,omitempty"`
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
	ID string `json:"id"`
}

type DeleteOverrideResponse struct{}

type ResolveForPageRequest struct {
	PageID string `json:"pageId"`
}

type ResolveForPageResponse struct {
	TemplateID string                `json:"templateId"`
	Version    int                   `json:"version"`
	Spec       template.TemplateSpec `json:"spec"`
}

type CreatePolicyRequest struct {
	Scope          string             `json:"scope,omitempty"`
	SiteID         *string            `json:"siteId,omitempty"`
	Name           string             `json:"name"`
	Rules          template.LinkRules `json:"rules"`
	ForbidExternal bool               `json:"forbidExternal"`
	ForbidSelf     bool               `json:"forbidSelf"`
	AnchorStrategy string             `json:"anchorStrategy,omitempty"`
}

type CreatePolicyResponse struct {
	Policy LinkPolicy `json:"policy"`
}

type UpdatePolicyRequest struct {
	ID             string              `json:"id"`
	Name           *string             `json:"name,omitempty"`
	Rules          *template.LinkRules `json:"rules,omitempty"`
	ForbidExternal *bool               `json:"forbidExternal,omitempty"`
	ForbidSelf     *bool               `json:"forbidSelf,omitempty"`
	AnchorStrategy *string             `json:"anchorStrategy,omitempty"`
}

type UpdatePolicyResponse struct {
	Policy LinkPolicy `json:"policy"`
}

type DeletePolicyRequest struct {
	ID string `json:"id"`
}

type DeletePolicyResponse struct{}

type GetPolicyRequest struct {
	ID string `json:"id"`
}

type GetPolicyResponse struct {
	Policy LinkPolicy `json:"policy"`
}

type ListPoliciesRequest struct {
	dto.ListRequest
	Scope  string `json:"scope,omitempty"`
	SiteID string `json:"siteId,omitempty"`
}

type GetEffectivePolicyRequest struct {
	SiteID string `json:"siteId"`
}

type GetEffectivePolicyResponse struct {
	Policy LinkPolicy `json:"policy"`
}
