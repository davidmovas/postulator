package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/templates"
)

const templatesSetOverrideName = "templates_set_override"

type templatesSetOverrideArgs struct {
	TemplateID string            `json:"templateId" description:"Template id from templates_list"`
	Scope      string            `json:"scope" enum:"site,page" description:"Every page of a site, or one page"`
	TargetID   string            `json:"targetId" description:"Site id or page id, as the scope says"`
	Patch      templatePatchArgs `json:"patch" description:"Only the specification fields that change"`
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
