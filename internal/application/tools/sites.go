package tools

import (
	"context"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/application/reports"
	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/application/sync"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	sitesName       = "sites"
	sitesManageName = "sites_manage"
	viewOverview    = "overview"
	opSync          = "sync"
	opCheckPlugin   = "check_plugin"
)

type sitesArgs struct {
	View   string `json:"view" enum:"list,get,overview" description:"list the sites; get one with its plugin state and settings; overview this conversation's site as it stands now: its totals and its brief"`
	SiteID string `json:"siteId,omitempty" description:"get: the site; default this conversation's"`
	Status string `json:"status,omitempty" enum:"active,paused,error" description:"list: only this state"`
	Cursor string `json:"cursor,omitempty" description:"list: nextCursor of the last call"`
	Limit  int    `json:"limit,omitempty" description:"list: default 50"`
}

type SiteOverview struct {
	Site   sites.Site                   `json:"site"`
	Totals reports.SiteOverviewResponse `json:"totals"`
	Brief  contextpack.Brief            `json:"brief"`
}

type SitesResponse struct {
	Sites    *paging.List[sites.Site] `json:"sites,omitempty"`
	Site     *sites.Site              `json:"site,omitempty"`
	Overview *SiteOverview            `json:"overview,omitempty"`
}

func sitesTool(deps Deps) Tool {
	return NewTool(Def{
		Name:        sitesName,
		Description: "Read the WordPress sites Postulator manages, one site whole, or this site's live overview.",
		Risk:        RiskRead,
	}, func(ctx context.Context, b Binding, in sitesArgs) (SitesResponse, error) {
		switch in.View {
		case viewList:
			listed, err := fitted(deps, in.Limit, func(limit int) (paging.List[sites.Site], error) {
				return deps.Sites.List(ctx, sites.ListRequest{ListRequest: dto.ListRequest{Cursor: in.Cursor, Limit: limit}, Status: in.Status})
			})
			return SitesResponse{Sites: &listed}, err
		case opGet:
			siteID := in.SiteID
			if siteID == "" {
				siteID = b.SiteID
			}
			held, err := deps.Sites.Get(ctx, sites.GetRequest{ID: siteID})
			return SitesResponse{Site: &held.Site}, err
		case viewOverview:
			if err := requireSite(ctx, b); err != nil {
				return SitesResponse{}, err
			}
			overview, err := deps.overview(ctx, b.SiteID)
			return SitesResponse{Overview: &overview}, err
		default:
			return SitesResponse{}, errors.New(errors.Invalid, "sites takes the view list, get or overview").WithDetail("field", "view")
		}
	})
}

func (deps Deps) overview(ctx context.Context, siteID string) (SiteOverview, error) {
	held, err := deps.Sites.Get(ctx, sites.GetRequest{ID: siteID})
	if err != nil {
		return SiteOverview{}, err
	}
	totals, err := deps.Reports.SiteOverview(ctx, reports.SiteOverviewRequest{SiteID: siteID})
	if err != nil {
		return SiteOverview{}, err
	}
	brief, err := deps.SiteModel.Brief(ctx, siteID)
	if err != nil {
		return SiteOverview{}, err
	}
	return SiteOverview{Site: held.Site, Totals: totals, Brief: brief}, nil
}

type builderArgs struct {
	Name              string   `json:"name" description:"Builder name"`
	DataAttributes    []string `json:"dataAttributes,omitempty" description:"Its data attributes"`
	ClassPrefixes     []string `json:"classPrefixes,omitempty" description:"Its class prefixes"`
	ShortcodePrefixes []string `json:"shortcodePrefixes,omitempty" description:"Its shortcode prefixes"`
}

type sitesManageArgs struct {
	Op                string         `json:"op" enum:"create,update,delete,sync,check_plugin" description:"sync observes the live site into the model; check_plugin probes the companion plugin"`
	SiteID            string         `json:"siteId,omitempty" description:"update, delete: the site; default this conversation's"`
	Name              *string        `json:"name,omitempty" description:"create, update: the name inside Postulator"`
	BaseURL           *string        `json:"baseUrl,omitempty" description:"create, update: the site's URL, e.g. https://example.com"`
	Username          *string        `json:"username,omitempty" description:"create, update: the WordPress administrator login"`
	Password          *string        `json:"password,omitempty" description:"create, update: its application password; sealed, never read back"`
	AllowInsecure     *bool          `json:"allowInsecure,omitempty" description:"create, update: accept an untrusted certificate, local sites only"`
	Status            *string        `json:"status,omitempty" enum:"active,paused,error" description:"update: the site's state"`
	DefaultTemplateID *string        `json:"defaultTemplateId,omitempty" description:"update: the fallback template"`
	UserAgent         *string        `json:"userAgent,omitempty" description:"update: User-Agent sent to the site; default a neutral browser string"`
	RenderConcurrency *int           `json:"renderConcurrency,omitempty" minimum:"1" maximum:"16" description:"update: renders fetched at once; default 4"`
	FullWalkEvery     *int           `json:"fullWalkEvery,omitempty" minimum:"1" description:"update: observations between two full walks; default 10"`
	LinkURLForm       *string        `json:"linkUrlForm,omitempty" enum:"auto,absolute,relative" description:"update: URL form of the links we place; default auto"`
	Shortcodes        *[]string      `json:"technicalShortcodes,omitempty" description:"update: shortcodes that make a page technical; [] the defaults"`
	Blocks            *[]string      `json:"technicalBlocks,omitempty" description:"update: blocks that make a page technical; [] the defaults"`
	Builders          *[]builderArgs `json:"builders,omitempty" description:"update: page builders to detect, the whole list; [] the defaults"`
	Full              bool           `json:"full,omitempty" description:"sync: walk the whole site; default only every Nth observation"`
	RenderRescan      bool           `json:"renderRescan,omitempty" description:"sync: fetch every published render again; default changed pages only"`
}

func (in *sitesManageArgs) touchesDefaults() bool {
	return in.DefaultTemplateID != nil
}

func (in *sitesManageArgs) touchesSettings() bool {
	return in.UserAgent != nil || in.RenderConcurrency != nil || in.FullWalkEvery != nil || in.LinkURLForm != nil ||
		in.Shortcodes != nil || in.Blocks != nil || in.Builders != nil
}

func orDefaults(listed *[]string) []string {
	if len(*listed) == 0 {
		return nil
	}
	return slices.Clone(*listed)
}

func (in *sitesManageArgs) defaultsOver(current sites.Defaults) *sites.Defaults {
	if in.DefaultTemplateID != nil {
		current.TemplateID = in.DefaultTemplateID
	}
	return &current
}

func (in *sitesManageArgs) settingsOver(current site.Settings) *site.Settings {
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
	if in.Shortcodes != nil {
		current.TechnicalShortcodes = orDefaults(in.Shortcodes)
	}
	if in.Blocks != nil {
		current.TechnicalBlocks = orDefaults(in.Blocks)
	}
	if in.Builders != nil {
		current.Builders = nil
		for _, rule := range *in.Builders {
			current.Builders = append(current.Builders, site.BuilderRule{
				Name: rule.Name, DataAttributes: rule.DataAttributes, ClassPrefixes: rule.ClassPrefixes,
				ShortcodePrefixes: rule.ShortcodePrefixes,
			})
		}
	}
	return &current
}

func (in *sitesManageArgs) target(b Binding) string {
	if in.SiteID != "" {
		return in.SiteID
	}
	return b.SiteID
}

func sitesManage(deps Deps) Tool {
	tool := NewTool(Def{
		Name:        sitesManageName,
		Description: "Register, change or delete a WordPress site and set its defaults and settings, observe the live site, or probe its plugin.",
		Risk:        RiskDangerous,
	}, func(ctx context.Context, b Binding, in sitesManageArgs) (any, error) {
		return deps.manageSite(ctx, b, &in)
	})
	tool = describing(tool, func(ctx context.Context, b Binding, in sitesManageArgs) (string, error) {
		return deps.siteChange(ctx, b, &in)
	})
	return riskedBy(tool, map[string]Risk{opCreate: RiskWrite, opUpdate: RiskWrite, opSync: RiskWrite, opCheckPlugin: RiskRead})
}

func (deps Deps) manageSite(ctx context.Context, b Binding, in *sitesManageArgs) (any, error) {
	switch in.Op {
	case opCreate:
		return deps.Sites.Create(ctx, sites.CreateRequest{
			Name: valueOf(in.Name), BaseURL: valueOf(in.BaseURL), Username: valueOf(in.Username), Password: valueOf(in.Password),
			AllowInsecure: valueOf(in.AllowInsecure),
		})
	case opUpdate:
		request := sites.UpdateRequest{
			ID: in.target(b), Name: in.Name, BaseURL: in.BaseURL, Username: in.Username,
			Password: in.Password, AllowInsecure: in.AllowInsecure, Status: in.Status,
		}
		if in.touchesDefaults() || in.touchesSettings() {
			current, err := deps.Sites.Get(ctx, sites.GetRequest{ID: request.ID})
			if err != nil {
				return nil, err
			}
			if in.touchesDefaults() {
				request.Defaults = in.defaultsOver(current.Site.Defaults)
			}
			if in.touchesSettings() {
				request.Settings = in.settingsOver(current.Site.Settings)
			}
		}
		return deps.Sites.Update(ctx, request)
	case opDelete:
		return deps.Sites.Delete(ctx, sites.DeleteRequest{ID: in.target(b)})
	case opSync, opCheckPlugin:
		if err := requireSite(ctx, b); err != nil {
			return nil, err
		}
		if in.Op == opSync {
			return deps.Sync.SyncSite(ctx, sync.SyncSiteRequest{SiteID: b.SiteID, Full: in.Full, RenderRescan: in.RenderRescan})
		}
		return deps.Sync.CheckPlugin(ctx, sync.CheckPluginRequest{SiteID: b.SiteID})
	default:
		return nil, errors.New(errors.Invalid, "sites_manage takes the op create, update, delete, sync or check_plugin").
			WithDetail("field", "op")
	}
}

func (deps Deps) siteChange(ctx context.Context, b Binding, in *sitesManageArgs) (string, error) {
	if in.Op == opCreate {
		return "Register the site " + valueOf(in.Name) + " at " + valueOf(in.BaseURL), nil
	}
	held, err := deps.Sites.Get(ctx, sites.GetRequest{ID: in.target(b)})
	if err != nil {
		return "", err
	}
	name := held.Site.Name
	switch in.Op {
	case opUpdate:
		return "Change the site " + name + "'s " + strings.Join(in.changed(), ", "), nil
	case opDelete:
		return "Delete the site " + name + " and everything Postulator holds about it; the WordPress site is not touched", nil
	case opSync:
		if in.Full {
			return "Walk the whole of " + name + " into the model", nil
		}
		return "Observe what changed on " + name + " since the last walk", nil
	default:
		return "Probe " + name + " for the companion plugin", nil
	}
}

func (in *sitesManageArgs) changed() []string {
	named := []struct {
		set  bool
		name string
	}{
		{in.Name != nil, "name"}, {in.BaseURL != nil, "address"}, {in.Username != nil || in.Password != nil, "credentials"},
		{in.AllowInsecure != nil, "certificate check"}, {in.Status != nil, "state"}, {in.touchesDefaults(), "default template"},
		{in.touchesSettings(), "settings"},
	}
	out := make([]string, 0, len(named))
	for _, held := range named {
		if held.set {
			out = append(out, held.name)
		}
	}
	if len(out) == 0 {
		return []string{"nothing"}
	}
	return out
}
