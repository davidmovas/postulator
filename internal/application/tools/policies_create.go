package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/templates"
)

const policiesCreateName = "policies_create"

type policiesCreateArgs struct {
	Scope          string         `json:"scope,omitempty" enum:"global,site" description:"Every site or one; left out, global"`
	SiteID         *string        `json:"siteId,omitempty" description:"Required when the scope is site"`
	Name           string         `json:"name" description:"Two to four words"`
	Rules          *linkRulesArgs `json:"rules,omitempty" description:"Internal links owed and allowed"`
	ForbidExternal bool           `json:"forbidExternal,omitempty" description:"Refuse links that leave the site"`
	ForbidSelf     bool           `json:"forbidSelf,omitempty" description:"Refuse a link to the page itself"`
	AnchorStrategy string         `json:"anchorStrategy,omitempty" enum:"prefer_user,rotate" description:"Keep to human anchors (the default) or rotate through all"`
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
