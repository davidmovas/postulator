package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/templates"
)

const templatesUpdateName = "templates_update"

type templatesUpdateArgs struct {
	ID       string            `json:"id" description:"The id of the template, exactly as templates_list returned it"`
	Name     *string           `json:"name,omitempty" description:"The new name, left out to keep the current one"`
	PageKind *string           `json:"pageKind,omitempty" enum:"hub,category,guide,comparison,product" description:"The new page kind, left out to keep the current one"`
	Spec     *templateSpecArgs `json:"spec,omitempty" description:"The whole replacement specification, left out to keep the current one"`
}

func templatesUpdate(deps Deps) Tool {
	return NewTool(Def{
		Name:        templatesUpdateName,
		Description: "Change the name, page kind or specification of a template.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in templatesUpdateArgs) (templates.UpdateTemplateResponse, error) {
		request := templates.UpdateTemplateRequest{ID: in.ID, Name: in.Name, PageKind: in.PageKind}
		if in.Spec != nil {
			spec := in.Spec.spec()
			request.Spec = &spec
		}
		return deps.Templates.UpdateTemplate(ctx, request)
	})
}
