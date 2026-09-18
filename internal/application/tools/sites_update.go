package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/sites"
)

const sitesUpdateName = "sites_update"

type sitesUpdateArgs struct {
	ID                  string  `json:"id"`
	Name                *string `json:"name,omitempty"`
	BaseURL             *string `json:"baseUrl,omitempty"`
	Username            *string `json:"username,omitempty"`
	Password            *string `json:"password,omitempty" description:"the WordPress application password; it is stored encrypted and never read back"`
	AllowInsecure       *bool   `json:"allowInsecure,omitempty"`
	Status              *string `json:"status,omitempty" enum:"active,paused,error"`
	DefaultTemplateID   *string `json:"defaultTemplateId,omitempty"`
	DefaultLinkPolicyID *string `json:"defaultLinkPolicyId,omitempty"`
}

func sitesUpdate(deps Deps) Tool {
	return NewTool(Def{
		Name:        sitesUpdateName,
		Description: "Change a site name, base url, credentials or defaults.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in sitesUpdateArgs) (sites.UpdateResponse, error) {
		request := sites.UpdateRequest{
			ID: in.ID, Name: in.Name, BaseURL: in.BaseURL, Username: in.Username,
			Password: in.Password, AllowInsecure: in.AllowInsecure, Status: in.Status,
		}
		if in.DefaultTemplateID == nil && in.DefaultLinkPolicyID == nil {
			return deps.Sites.Update(ctx, request)
		}

		current, err := deps.Sites.Get(ctx, sites.GetRequest{ID: in.ID})
		if err != nil {
			return sites.UpdateResponse{}, err
		}

		defaults := current.Site.Defaults
		if in.DefaultTemplateID != nil {
			defaults.TemplateID = in.DefaultTemplateID
		}
		if in.DefaultLinkPolicyID != nil {
			defaults.LinkPolicyID = in.DefaultLinkPolicyID
		}
		request.Defaults = &defaults
		return deps.Sites.Update(ctx, request)
	})
}
