package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/template"
)

const templatesCreateName = "templates_create"

type templatesCreateArgs struct {
	Scope    string           `json:"scope,omitempty" enum:"global,site" description:"Every site or one; left out, global"`
	SiteID   *string          `json:"siteId,omitempty" description:"Required when the scope is site"`
	Name     string           `json:"name" description:"Two to four words"`
	PageKind string           `json:"pageKind" enum:"hub,category,guide,comparison,product" description:"Kind of page it writes"`
	Spec     templateSpecArgs `json:"spec" description:"The full specification"`
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
