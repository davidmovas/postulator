package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/templates"
)

const policiesUpdateName = "policies_update"

type policiesUpdateArgs struct {
	ID             string         `json:"id" description:"The id of the policy, exactly as policies_list returned it"`
	Name           *string        `json:"name,omitempty" description:"The new name, left out to keep the current one"`
	Rules          *linkRulesArgs `json:"rules,omitempty" description:"The whole new set of link rules, left out to keep the current one"`
	ForbidExternal *bool          `json:"forbidExternal,omitempty" description:"The new setting for links that leave the site, left out to keep the current one"`
	ForbidSelf     *bool          `json:"forbidSelf,omitempty" description:"The new setting for a link to the page itself, left out to keep the current one"`
	AnchorStrategy *string        `json:"anchorStrategy,omitempty" enum:"prefer_user,rotate" description:"The new anchor strategy, left out to keep the current one"`
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
