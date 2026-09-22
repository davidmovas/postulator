package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/templates"
)

const templatesSetOverrideName = "templates_set_override"

type templatesSetOverrideArgs struct {
	TemplateID string            `json:"templateId" description:"The id of the template being overridden, exactly as templates_list returned it"`
	Scope      string            `json:"scope" enum:"site,page" description:"Whether the override applies to every page of a site or to one page"`
	TargetID   string            `json:"targetId" description:"The site id for a site override, the page id for a page override"`
	Patch      templatePatchArgs `json:"patch" description:"Only the fields of the specification that change; every other field keeps the template's own value"`
}

func templatesSetOverride(deps Deps) Tool {
	return NewTool(Def{
		Name:        templatesSetOverrideName,
		Description: "Patch a template for one site or one page.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in templatesSetOverrideArgs) (templates.SetOverrideResponse, error) {
		patch, err := in.Patch.patch()
		if err != nil {
			return templates.SetOverrideResponse{}, err
		}
		return deps.Templates.SetOverride(ctx, templates.SetOverrideRequest{
			TemplateID: in.TemplateID, Scope: in.Scope, TargetID: in.TargetID, Patch: patch,
		})
	})
}
