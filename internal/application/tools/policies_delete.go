package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/templates"
)

const policiesDeleteName = "policies_delete"

func policiesDelete(deps Deps) Tool {
	return NewTool(Def{
		Name:        policiesDeleteName,
		Description: "Delete a link policy.",
		Risk:        RiskDangerous,
	}, func(ctx context.Context, _ Binding, in templates.DeletePolicyRequest) (templates.DeletePolicyResponse, error) {
		return deps.Templates.DeletePolicy(ctx, in)
	})
}
