package steps

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
)

func effectivePolicy(ctx context.Context, deps Deps, sc *run.StepContext) (template.LinkPolicy, error) {
	resp, err := deps.Policies.GetEffectivePolicy(ctx, templates.GetEffectivePolicyRequest{SiteID: sc.Run.SiteID})
	if err != nil {
		return template.LinkPolicy{}, err
	}

	return template.LinkPolicy{
		Rules:          templates.EffectiveRules(resp.Policy.Rules, sc.Spec.LinkRules),
		ForbidExternal: resp.Policy.ForbidExternal,
		ForbidSelf:     resp.Policy.ForbidSelf,
		AnchorStrategy: template.AnchorStrategy(resp.Policy.AnchorStrategy),
	}, nil
}
