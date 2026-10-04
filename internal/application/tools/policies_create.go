package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/templates"
)

const policiesCreateName = "policies_create"

type policiesCreateArgs struct {
	Scope          string         `json:"scope,omitempty" enum:"global,site" description:"Default global"`
	SiteID         *string        `json:"siteId,omitempty" description:"Required for site scope"`
	Name           string         `json:"name" description:"Name, 2-4 words"`
	Rules          *linkRulesArgs `json:"rules,omitempty" description:"Internal links owed and allowed"`
	ForbidExternal bool           `json:"forbidExternal,omitempty" description:"Refuse external links"`
	ForbidSelf     bool           `json:"forbidSelf,omitempty" description:"Refuse self links"`
	AnchorStrategy string         `json:"anchorStrategy,omitempty" enum:"prefer_user,rotate" description:"Default prefer_user, human anchors first"`
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
