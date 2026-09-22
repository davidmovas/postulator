package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/templates"
)

const policiesCreateName = "policies_create"

type policiesCreateArgs struct {
	Scope          string         `json:"scope,omitempty" enum:"global,site" description:"Whether the policy applies to every site or to one; leave it out for global"`
	SiteID         *string        `json:"siteId,omitempty" description:"The site the policy belongs to, required when the scope is site"`
	Name           string         `json:"name" description:"What to call the policy, two to four words"`
	Rules          *linkRulesArgs `json:"rules,omitempty" description:"How many links a page may carry and which of them are owed; leave it out to ask nothing"`
	ForbidExternal bool           `json:"forbidExternal,omitempty" description:"Refuse links that leave the site"`
	ForbidSelf     bool           `json:"forbidSelf,omitempty" description:"Refuse a link from a page to itself"`
	AnchorStrategy string         `json:"anchorStrategy,omitempty" enum:"prefer_user,rotate" description:"Whether to keep to the anchors a human wrote or to rotate through all of them; leave it out for prefer_user"`
}

func (a policiesCreateArgs) request() templates.CreatePolicyRequest {
	built := templates.CreatePolicyRequest{
		Scope: a.Scope, SiteID: a.SiteID, Name: a.Name,
		ForbidExternal: a.ForbidExternal, ForbidSelf: a.ForbidSelf, AnchorStrategy: a.AnchorStrategy,
	}
	if a.Rules != nil {
		built.Rules = a.Rules.rules()
	}
	return built
}

func policiesCreate(deps Deps) Tool {
	return NewTool(Def{
		Name:        policiesCreateName,
		Description: "Create a link policy.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in policiesCreateArgs) (templates.CreatePolicyResponse, error) {
		return deps.Templates.CreatePolicy(ctx, in.request())
	})
}
