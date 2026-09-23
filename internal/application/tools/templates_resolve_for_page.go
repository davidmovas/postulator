package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/templates"
)

const templatesResolveForPageName = "templates_resolve_for_page"

func templatesResolveForPage(deps Deps) Tool {
	return NewTool(Def{
		Name:        templatesResolveForPageName,
		Description: "Resolve the template specification that applies to a page.",
		Risk:        RiskRead,
	}, func(ctx context.Context, _ Binding, in templates.ResolveForPageRequest) (templates.ResolveForPageResponse, error) {
		return deps.Templates.ResolveForPage(ctx, in)
	})
}
