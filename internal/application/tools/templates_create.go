package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/template"
)

const templatesCreateName = "templates_create"

type templatesCreateArgs struct {
	Scope    string           `json:"scope,omitempty" enum:"global,site" description:"Whether the template belongs to every site or to one; leave it out for global"`
	SiteID   *string          `json:"siteId,omitempty" description:"The site the template belongs to, required when the scope is site"`
	Name     string           `json:"name" description:"What to call the template, two to four words"`
	PageKind string           `json:"pageKind" enum:"hub,category,guide,comparison,product" description:"The kind of page this template writes"`
	Spec     templateSpecArgs `json:"spec" description:"The whole specification: the sections, the tone, the length, and the keyword, link, meta and image rules"`
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
