package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/sites"
)

const sitesUpdateName = "sites_update"

type sitesUpdateArgs struct {
	ID                  string  `json:"id" description:"Site id from sites_list"`
	Name                *string `json:"name,omitempty" description:"New name"`
	BaseURL             *string `json:"baseUrl,omitempty" description:"New WordPress address"`
	Username            *string `json:"username,omitempty" description:"New administrator login"`
	Password            *string `json:"password,omitempty" description:"WordPress application password, stored encrypted, never read back"`
	AllowInsecure       *bool   `json:"allowInsecure,omitempty" description:"Accept an untrusted certificate; only a local site needs it"`
	Status              *string `json:"status,omitempty" enum:"active,paused,error" description:"New status"`
	DefaultTemplateID   *string `json:"defaultTemplateId,omitempty" description:"Fallback template id from templates_list"`
	DefaultLinkPolicyID *string `json:"defaultLinkPolicyId,omitempty" description:"Fallback link policy id from policies_list"`
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
