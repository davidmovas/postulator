package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/sync"
)

const syncCheckPluginName = "sync_check_plugin"

func syncCheckPlugin(deps Deps) Tool {
	return newSiteTool(Def{
		Name:        syncCheckPluginName,
		Description: "Probe the site for the companion plugin and record what it offers.",
		Risk:        RiskRead,
	}, func(ctx context.Context, b Binding, in sync.CheckPluginRequest) (sync.CheckPluginResponse, error) {
		in.SiteID = b.SiteID
		return deps.Sync.CheckPlugin(ctx, in)
	})
}
