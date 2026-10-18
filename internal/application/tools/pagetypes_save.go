package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/pagetypes"
)

const pagetypesSaveName = "pagetypes_save"

type pagetypesSaveArgs struct {
	ID         string  `json:"id,omitempty" description:"Page type id to change; omit to create one"`
	SiteID     string  `json:"siteId,omitempty" description:"Site id; required to create"`
	Name       *string `json:"name,omitempty" description:"Name; required to create"`
	Preset     string  `json:"preset,omitempty" enum:"hub,entity,product,store_category,catalog,technical" description:"Role it takes; required to create, fixed after"`
	TemplateID *string `json:"templateId,omitempty" description:"Template id for its pages; empty clears"`
}

func pagetypesSave(deps Deps) Tool {
	return NewTool(Def{
		Name:        pagetypesSaveName,
		Description: "Create a page type from a preset, or rename one or set its template.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in pagetypesSaveArgs) (pagetypes.Type, error) {
		if in.ID != "" {
			return deps.PageTypes.Save(ctx, pagetypes.SaveRequest{ID: in.ID, Name: in.Name, TemplateID: in.TemplateID})
		}
		name := ""
		if in.Name != nil {
			name = *in.Name
		}
		return deps.PageTypes.Create(ctx, pagetypes.CreateRequest{
			SiteID: in.SiteID, Name: name, Preset: in.Preset, TemplateID: in.TemplateID,
		})
	})
}
