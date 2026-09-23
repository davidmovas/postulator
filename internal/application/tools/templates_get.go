package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/templates"
)

const templatesGetName = "templates_get"

func templatesGet(deps Deps) Tool {
	return NewTool(Def{
		Name:        templatesGetName,
		Description: "Read one template with its overrides.",
		Risk:        RiskRead,
	}, func(ctx context.Context, _ Binding, in templates.GetTemplateRequest) (templates.GetTemplateResponse, error) {
		return deps.Templates.GetTemplate(ctx, in)
	})
}
