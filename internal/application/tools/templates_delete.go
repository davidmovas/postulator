package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/templates"
)

const templatesDeleteName = "templates_delete"

func templatesDelete(deps Deps) Tool {
	return NewTool(Def{
		Name:        templatesDeleteName,
		Description: "Delete a template.",
		Risk:        RiskDangerous,
	}, func(ctx context.Context, _ Binding, in templates.DeleteTemplateRequest) (templates.DeleteTemplateResponse, error) {
		return deps.Templates.DeleteTemplate(ctx, in)
	})
}
