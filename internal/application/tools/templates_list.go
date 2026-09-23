package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const templatesListName = "templates_list"

func templatesList(deps Deps) Tool {
	return sortedBy(NewTool(Def{
		Name:        templatesListName,
		Description: "List the content templates, global and site scoped.",
		Risk:        RiskRead,
	}, func(ctx context.Context, _ Binding, in templates.ListTemplatesRequest) (paging.List[templates.Template], error) {
		return deps.Templates.ListTemplates(ctx, in)
	}), string(template.SortCreatedAt), string(template.SortName))
}
