package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/template"
)

const templatesUpdateName = "templates_update"

type templatesUpdateArgs struct {
	ID       string  `json:"id"`
	Name     *string `json:"name,omitempty"`
	PageKind *string `json:"pageKind,omitempty" enum:"hub,category,guide,comparison,product"`
	Spec     string  `json:"spec,omitempty" description:"the whole replacement specification as a JSON object; the current one is kept when this is empty"`
}

func templatesUpdate(deps Deps) Tool {
	return NewTool(Def{
		Name:        templatesUpdateName,
		Description: "Change the name, page kind or specification of a template.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in templatesUpdateArgs) (templates.UpdateTemplateResponse, error) {
		request := templates.UpdateTemplateRequest{ID: in.ID, Name: in.Name, PageKind: in.PageKind}
		if in.Spec != "" {
			spec, err := decodeSpec(in.Spec)
			if err != nil {
				return templates.UpdateTemplateResponse{}, err
			}
			request.Spec = &spec
		}
		return deps.Templates.UpdateTemplate(ctx, request)
	})
}

func decodeSpec(raw string) (template.TemplateSpec, error) {
	var spec template.TemplateSpec
	if err := decodeJSONArgument(raw, &spec, "template specification"); err != nil {
		return template.TemplateSpec{}, err
	}
	return spec, nil
}
