package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/templates"
)

const templatesDeleteOverrideName = "templates_delete_override"

func templatesDeleteOverride(deps Deps) Tool {
	return NewTool(Def{
		Name:        templatesDeleteOverrideName,
		Description: "Delete a template override.",
		Risk:        RiskDangerous,
	}, func(ctx context.Context, _ Binding, in templates.DeleteOverrideRequest) (templates.DeleteOverrideResponse, error) {
		return deps.Templates.DeleteOverride(ctx, in)
	})
}
