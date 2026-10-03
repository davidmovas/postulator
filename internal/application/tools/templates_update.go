package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/template"
)

const templatesUpdateName = "templates_update"

type templatesUpdateArgs struct {
	ID       string            `json:"id" description:"Template id from templates_list"`
	Name     *string           `json:"name,omitempty" description:"New name"`
	PageKind *string           `json:"pageKind,omitempty" enum:"hub,category,guide,comparison,product" description:"New page kind"`
	Spec     *templateSpecArgs `json:"spec,omitempty" description:"Full replacement specification"`
}

func templatesUpdate(deps Deps) Tool {
	return checking(NewTool(Def{
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
	}), func(in templatesUpdateArgs) error {
		if in.Spec == nil {
			return nil
		}
		return template.Validate(in.Spec.spec())
	})
}
