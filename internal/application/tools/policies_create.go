package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/templates"
)

const policiesCreateName = "policies_create"

func policiesCreate(deps Deps) Tool {
	return NewTool(Def{
		Name:        policiesCreateName,
		Description: "Create a link policy.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in templates.CreatePolicyRequest) (templates.CreatePolicyResponse, error) {
		return deps.Templates.CreatePolicy(ctx, in)
	})
}
