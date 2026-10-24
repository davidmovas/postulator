package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/sync"
)

const syncSiteName = "sync_site"

type syncSiteArgs struct {
	Full         bool `json:"full,omitempty" description:"Walk the whole site; default only every Nth observation"`
	RenderRescan bool `json:"renderRescan,omitempty" description:"Fetch every published render again; default changed pages only"`
}

func syncSite(deps Deps) Tool {
	return newSiteTool(Def{
		Name:        syncSiteName,
		Description: "Observe the live WordPress site into the page model: pages, fields, bodies, renders, links and hierarchy.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, b Binding, in syncSiteArgs) (sync.SyncSiteResponse, error) {
		return deps.Sync.SyncSite(ctx, sync.SyncSiteRequest{SiteID: b.SiteID, Full: in.Full, RenderRescan: in.RenderRescan})
	})
}
