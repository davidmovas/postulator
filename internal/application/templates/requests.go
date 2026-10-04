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
	Scope    string `json:"scope,omitempty" enum:"global,site" description:"Only this scope"`
	SiteID   string `json:"siteId,omitempty" description:"Only this site's"`
	PageKind string `json:"pageKind,omitempty" description:"Only this page kind, such as hub or guide"`
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
	ID string `json:"id" description:"Override id from templates_get"`
}

type DeleteOverrideResponse struct{}

type ResolveForPageRequest struct {
	PageID     string `json:"pageId" description:"Page id"`
	TemplateID string `json:"templateId,omitempty" description:"Template id to resolve instead; default the page's own"`
}

type ResolveForPageResponse struct {
	TemplateID string                `json:"templateId"`
	SiteID     string                `json:"siteId"`
	Version    int                   `json:"version"`
	Spec       template.TemplateSpec `json:"spec"`
}

type CreatePolicyRequest struct {
	Scope          string             `json:"scope,omitempty" enum:"global,site" description:"Default global"`
	SiteID         *string            `json:"siteId,omitempty" description:"Required for site scope"`
	Name           string             `json:"name" description:"Name, 2-4 words"`
	Rules          template.LinkRules `json:"rules" description:"Internal links owed and allowed"`
	ForbidExternal bool               `json:"forbidExternal" description:"Refuse external links"`
	ForbidSelf     bool               `json:"forbidSelf" description:"Refuse self links"`
	AnchorStrategy string             `json:"anchorStrategy,omitempty" enum:"prefer_user,rotate" description:"Default prefer_user, human anchors first"`
}

type CreatePolicyResponse struct {
	Policy LinkPolicy `json:"policy"`
}

type UpdatePolicyRequest struct {
	ID             string              `json:"id" description:"Policy id"`
	Name           *string             `json:"name,omitempty" description:"New name"`
	Rules          *template.LinkRules `json:"rules,omitempty" description:"Whole new link rules"`
	ForbidExternal *bool               `json:"forbidExternal,omitempty" description:"Refuse external links"`
	ForbidSelf     *bool               `json:"forbidSelf,omitempty" description:"Refuse self links"`
	AnchorStrategy *string             `json:"anchorStrategy,omitempty" enum:"prefer_user,rotate" description:"New anchor strategy"`
}

type UpdatePolicyResponse struct {
	Policy LinkPolicy `json:"policy"`
}

type DeletePolicyRequest struct {
	ID string `json:"id" description:"Policy id"`
}

type DeletePolicyResponse struct{}

type GetPolicyRequest struct {
	ID string `json:"id" description:"Policy id"`
}

type GetPolicyResponse struct {
	Policy LinkPolicy `json:"policy"`
}

type ListPoliciesRequest struct {
	dto.ListRequest
	Scope  string `json:"scope,omitempty" enum:"global,site" description:"Only this scope"`
	SiteID string `json:"siteId,omitempty" description:"Only this site's"`
}

type GetEffectivePolicyRequest struct {
	SiteID string `json:"siteId"`
}

type GetEffectivePolicyResponse struct {
	Policy LinkPolicy `json:"policy"`
}
