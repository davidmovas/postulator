package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/templates"
)

const templatesCreateName = "templates_create"

type templatesCreateArgs struct {
	Scope    string  `json:"scope,omitempty" enum:"global,site"`
	SiteID   *string `json:"siteId,omitempty"`
	Name     string  `json:"name"`
	PageKind string  `json:"pageKind" enum:"hub,category,guide,comparison,product"`
	Spec     string  `json:"spec" description:"the whole template specification as a JSON object: sections, tone, length, keywordRules, linkRules, metaRules, images, modelProfiles and recipe"`
}

func templatesCreate(deps Deps) Tool {
	return NewTool(Def{
		Name:        templatesCreateName,
		Description: "Create a content template from a full specification.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in templatesCreateArgs) (templates.CreateTemplateResponse, error) {
		spec, err := decodeSpec(in.Spec)
		if err != nil {
			return templates.CreateTemplateResponse{}, err
		}
		return deps.Templates.CreateTemplate(ctx, templates.CreateTemplateRequest{
			Scope: in.Scope, SiteID: in.SiteID, Name: in.Name, PageKind: in.PageKind, Spec: spec,
		})
	})
}
