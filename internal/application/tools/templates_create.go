package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/template"
)

const templatesCreateName = "templates_create"

type templatesCreateArgs struct {
	Scope    string           `json:"scope,omitempty" enum:"global,site" description:"Default global"`
	SiteID   *string          `json:"siteId,omitempty" description:"Required for site scope"`
	Name     string           `json:"name" description:"Name, 2-4 words"`
	PageKind string           `json:"pageKind" enum:"hub,category,guide,comparison,product" description:"Page kind it writes"`
	Spec     templateSpecArgs `json:"spec" description:"Full specification"`
}

func templatesCreate(deps Deps) Tool {
	return checking(NewTool(Def{
		Name:        templatesCreateName,
		Description: "Create a content template from a full specification.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in templatesCreateArgs) (templates.CreateTemplateResponse, error) {
		return deps.Templates.CreateTemplate(ctx, templates.CreateTemplateRequest{
			Scope: in.Scope, SiteID: in.SiteID, Name: in.Name, PageKind: in.PageKind, Spec: in.Spec.spec(),
		})
	}), func(in templatesCreateArgs) error {
		return template.Validate(in.Spec.spec())
	})
}
