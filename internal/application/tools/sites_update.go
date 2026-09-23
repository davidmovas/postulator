package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/sites"
)

const sitesUpdateName = "sites_update"

type sitesUpdateArgs struct {
	ID                  string  `json:"id" description:"The id of the site, exactly as sites_list returned it"`
	Name                *string `json:"name,omitempty" description:"The new name, left out to keep the current one"`
	BaseURL             *string `json:"baseUrl,omitempty" description:"The new address of the WordPress site, left out to keep the current one"`
	Username            *string `json:"username,omitempty" description:"The new administrator login, left out to keep the current one"`
	Password            *string `json:"password,omitempty" description:"The WordPress application password; it is stored encrypted and never read back"`
	AllowInsecure       *bool   `json:"allowInsecure,omitempty" description:"Accept a certificate the machine does not trust, which only a local site should need"`
	Status              *string `json:"status,omitempty" enum:"active,paused,error" description:"The new status, left out to keep the current one"`
	DefaultTemplateID   *string `json:"defaultTemplateId,omitempty" description:"The template a page of this site falls back to, exactly as templates_list returned its id"`
	DefaultLinkPolicyID *string `json:"defaultLinkPolicyId,omitempty" description:"The link policy this site falls back to, exactly as policies_list returned its id"`
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
