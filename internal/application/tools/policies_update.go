package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/templates"
)

const policiesUpdateName = "policies_update"

type policiesUpdateArgs struct {
	ID             string         `json:"id" description:"Policy id from policies_list"`
	Name           *string        `json:"name,omitempty" description:"New name"`
	Rules          *linkRulesArgs `json:"rules,omitempty" description:"Whole new link rules"`
	ForbidExternal *bool          `json:"forbidExternal,omitempty" description:"Refuse links that leave the site"`
	ForbidSelf     *bool          `json:"forbidSelf,omitempty" description:"Refuse a link to the page itself"`
	AnchorStrategy *string        `json:"anchorStrategy,omitempty" enum:"prefer_user,rotate" description:"New anchor strategy"`
}

func (a policiesUpdateArgs) request() templates.UpdatePolicyRequest {
	built := templates.UpdatePolicyRequest{
		ID: a.ID, Name: a.Name, ForbidExternal: a.ForbidExternal, ForbidSelf: a.ForbidSelf,
		AnchorStrategy: a.AnchorStrategy,
	}
	if a.Rules != nil {
		rules := a.Rules.rules()
		built.Rules = &rules
	}
	return built
}

func policiesUpdate(deps Deps) Tool {
	return NewTool(Def{
		Name:        policiesUpdateName,
		Description: "Change a link policy.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in policiesUpdateArgs) (templates.UpdatePolicyResponse, error) {
		return deps.Templates.UpdatePolicy(ctx, in.request())
	})
}
