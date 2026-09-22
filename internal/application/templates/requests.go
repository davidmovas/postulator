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
	ID string `json:"id" description:"The id of the template to remove, exactly as templates_list returned it"`
}

type DeleteTemplateResponse struct{}

type GetTemplateRequest struct {
	ID string `json:"id" description:"The id of the template, exactly as templates_list returned it"`
}

type GetTemplateResponse struct {
	Template  Template   `json:"template"`
	Overrides []Override `json:"overrides"`
}

type ListTemplatesRequest struct {
	dto.ListRequest
	Scope    string `json:"scope,omitempty" enum:"global,site" description:"Keep only templates of this scope"`
	SiteID   string `json:"siteId,omitempty" description:"Keep only templates of this site, exactly as sites_list returned its id"`
	PageKind string `json:"pageKind,omitempty" description:"Keep only templates written for this kind of page, such as hub or guide"`
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
	ID string `json:"id" description:"The id of the override to remove, exactly as templates_get returned it"`
}

type DeleteOverrideResponse struct{}

type ResolveForPageRequest struct {
	PageID string `json:"pageId" description:"The id of the page whose template is wanted, exactly as a read tool returned it"`
}

type ResolveForPageResponse struct {
	TemplateID string                `json:"templateId"`
	SiteID     string                `json:"siteId"`
	Version    int                   `json:"version"`
	Spec       template.TemplateSpec `json:"spec"`
}

type CreatePolicyRequest struct {
	Scope          string             `json:"scope,omitempty" enum:"global,site" description:"Whether the policy applies to every site or to one; leave it out for global"`
	SiteID         *string            `json:"siteId,omitempty" description:"The site the policy belongs to, required when the scope is site"`
	Name           string             `json:"name" description:"What to call the policy, two to four words"`
	Rules          template.LinkRules `json:"rules" description:"How many links a page may carry and which of them are owed"`
	ForbidExternal bool               `json:"forbidExternal" description:"Refuse links that leave the site"`
	ForbidSelf     bool               `json:"forbidSelf" description:"Refuse a link from a page to itself"`
	AnchorStrategy string             `json:"anchorStrategy,omitempty" enum:"prefer_user,rotate" description:"Whether to keep to the anchors a human wrote or to rotate through all of them; leave it out for prefer_user"`
}

type CreatePolicyResponse struct {
	Policy LinkPolicy `json:"policy"`
}

type UpdatePolicyRequest struct {
	ID             string              `json:"id" description:"The id of the policy, exactly as policies_list returned it"`
	Name           *string             `json:"name,omitempty" description:"The new name, left out to keep the current one"`
	Rules          *template.LinkRules `json:"rules,omitempty" description:"The whole new set of link rules, left out to keep the current one"`
	ForbidExternal *bool               `json:"forbidExternal,omitempty" description:"The new setting for links that leave the site, left out to keep the current one"`
	ForbidSelf     *bool               `json:"forbidSelf,omitempty" description:"The new setting for a link to the page itself, left out to keep the current one"`
	AnchorStrategy *string             `json:"anchorStrategy,omitempty" enum:"prefer_user,rotate" description:"The new anchor strategy, left out to keep the current one"`
}

type UpdatePolicyResponse struct {
	Policy LinkPolicy `json:"policy"`
}

type DeletePolicyRequest struct {
	ID string `json:"id" description:"The id of the policy to remove, exactly as policies_list returned it"`
}

type DeletePolicyResponse struct{}

type GetPolicyRequest struct {
	ID string `json:"id" description:"The id of the policy, exactly as policies_list returned it"`
}

type GetPolicyResponse struct {
	Policy LinkPolicy `json:"policy"`
}

type ListPoliciesRequest struct {
	dto.ListRequest
	Scope  string `json:"scope,omitempty" enum:"global,site" description:"Keep only policies of this scope"`
	SiteID string `json:"siteId,omitempty" description:"Keep only policies of this site, exactly as sites_list returned its id"`
}

type GetEffectivePolicyRequest struct {
	SiteID string `json:"siteId"`
}

type GetEffectivePolicyResponse struct {
	Policy LinkPolicy `json:"policy"`
}
