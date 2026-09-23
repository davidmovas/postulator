package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/templates"
)

const policiesGetName = "policies_get"

func policiesGet(deps Deps) Tool {
	return NewTool(Def{
		Name:        policiesGetName,
		Description: "Read one link policy.",
		Risk:        RiskRead,
	}, func(ctx context.Context, _ Binding, in templates.GetPolicyRequest) (templates.GetPolicyResponse, error) {
		return deps.Templates.GetPolicy(ctx, in)
	})
}
