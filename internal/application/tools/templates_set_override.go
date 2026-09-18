package tools

import (
	"context"
	"encoding/json"

	"github.com/davidmovas/postulator/internal/application/templates"
)

const templatesSetOverrideName = "templates_set_override"

type templatesSetOverrideArgs struct {
	TemplateID string `json:"templateId"`
	Scope      string `json:"scope" enum:"site,page"`
	TargetID   string `json:"targetId" description:"the site id for a site override, the page id for a page override"`
	Patch      string `json:"patch" description:"the fields of the template specification to override, as a JSON object"`
}

func templatesSetOverride(deps Deps) Tool {
	return NewTool(Def{
		Name:        templatesSetOverrideName,
		Description: "Patch a template for one site or one page.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in templatesSetOverrideArgs) (templates.SetOverrideResponse, error) {
		var patch json.RawMessage
		if err := decodeJSONArgument(in.Patch, &patch, "template override patch"); err != nil {
			return templates.SetOverrideResponse{}, err
		}
		return deps.Templates.SetOverride(ctx, templates.SetOverrideRequest{
			TemplateID: in.TemplateID, Scope: in.Scope, TargetID: in.TargetID, Patch: patch,
		})
	})
}
