package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/sync"
)

const syncSiteName = "sync_site"

func syncSite(deps Deps) Tool {
	return newSiteTool(Def{
		Name:        syncSiteName,
		Description: "Pull the live WordPress content of the site into the page map.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, b Binding, in sync.SyncSiteRequest) (sync.SyncSiteResponse, error) {
		in.SiteID = b.SiteID
		return deps.Sync.SyncSite(ctx, in)
	})
}
