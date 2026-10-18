package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/domain/site"
)

const sitesUpdateName = "sites_update"

type sitesUpdateArgs struct {
	ID                string  `json:"id" description:"Site id"`
	Name              *string `json:"name,omitempty" description:"New name"`
	BaseURL           *string `json:"baseUrl,omitempty" description:"New site URL"`
	Username          *string `json:"username,omitempty" description:"New administrator login"`
	Password          *string `json:"password,omitempty" description:"New application password; sealed, never read back"`
	AllowInsecure     *bool   `json:"allowInsecure,omitempty" description:"Accept an untrusted certificate, local sites only"`
	Status            *string `json:"status,omitempty" enum:"active,paused,error" description:"New status"`
	DefaultTemplateID *string `json:"defaultTemplateId,omitempty" description:"Fallback template id"`
	UserAgent         *string `json:"userAgent,omitempty" description:"User-Agent sent to this site; default a neutral browser string"`
	RenderConcurrency *int    `json:"renderConcurrency,omitempty" minimum:"1" maximum:"16" description:"Renders fetched at once during an observation; default 4"`
	FullWalkEvery     *int    `json:"fullWalkEvery,omitempty" minimum:"1" description:"Observations between two full walks; default 10"`
	LinkURLForm       *string `json:"linkUrlForm,omitempty" enum:"auto,absolute,relative" description:"URL form of the links we place; default auto"`
}

func (in *sitesUpdateArgs) touchesDefaults() bool {
	return in.DefaultTemplateID != nil
}

func (in *sitesUpdateArgs) touchesSettings() bool {
	return in.UserAgent != nil || in.RenderConcurrency != nil || in.FullWalkEvery != nil || in.LinkURLForm != nil
}

func (in *sitesUpdateArgs) defaultsOver(current sites.Defaults) *sites.Defaults {
	if in.DefaultTemplateID != nil {
		current.TemplateID = in.DefaultTemplateID
	}
	return &current
}

func (in *sitesUpdateArgs) settingsOver(current site.Settings) *site.Settings {
	if in.UserAgent != nil {
		current.UserAgent = *in.UserAgent
	}
	if in.RenderConcurrency != nil {
		current.Observe.RenderConcurrency = *in.RenderConcurrency
	}
	if in.FullWalkEvery != nil {
		current.Observe.FullWalkEvery = *in.FullWalkEvery
	}
	if in.LinkURLForm != nil {
		current.LinkURLForm = site.URLForm(*in.LinkURLForm)
	}
	return &current
}

func sitesUpdate(deps Deps) Tool {
	return NewTool(Def{
		Name:        sitesUpdateName,
		Description: "Change a site name, base url, credentials, defaults or settings.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in sitesUpdateArgs) (sites.UpdateResponse, error) {
		request := sites.UpdateRequest{
			ID: in.ID, Name: in.Name, BaseURL: in.BaseURL, Username: in.Username,
			Password: in.Password, AllowInsecure: in.AllowInsecure, Status: in.Status,
		}
		if !in.touchesDefaults() && !in.touchesSettings() {
			return deps.Sites.Update(ctx, request)
		}

		current, err := deps.Sites.Get(ctx, sites.GetRequest{ID: in.ID})
		if err != nil {
			return sites.UpdateResponse{}, err
		}
		if in.touchesDefaults() {
			request.Defaults = in.defaultsOver(current.Site.Defaults)
		}
		if in.touchesSettings() {
			request.Settings = in.settingsOver(current.Site.Settings)
		}
		return deps.Sites.Update(ctx, request)
	})
}
