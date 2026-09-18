package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/templates"
)

const policiesEffectiveName = "policies_effective"

func policiesEffective(deps Deps) Tool {
	return newSiteTool(Def{
		Name:        policiesEffectiveName,
		Description: "Read the link policy that is in force for the site.",
		Risk:        RiskRead,
	}, func(ctx context.Context, b Binding, in templates.GetEffectivePolicyRequest) (templates.GetEffectivePolicyResponse, error) {
		in.SiteID = b.SiteID
		return deps.Templates.GetEffectivePolicy(ctx, in)
	})
}
