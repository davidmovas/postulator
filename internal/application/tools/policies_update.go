package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/templates"
)

const policiesUpdateName = "policies_update"

func policiesUpdate(deps Deps) Tool {
	return NewTool(Def{
		Name:        policiesUpdateName,
		Description: "Change a link policy.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in templates.UpdatePolicyRequest) (templates.UpdatePolicyResponse, error) {
		return deps.Templates.UpdatePolicy(ctx, in)
	})
}
