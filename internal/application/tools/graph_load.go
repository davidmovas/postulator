package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/graph"
)

const graphLoadName = "graph_load"

func graphLoad(deps Deps) Tool {
	return newSiteTool(Def{
		Name:        graphLoadName,
		Description: "Load the whole entity graph of the site at once.",
		Risk:        RiskRead,
	}, func(ctx context.Context, b Binding, in graph.LoadGraphRequest) (graph.LoadGraphResponse, error) {
		in.SiteID = b.SiteID
		return deps.Graph.LoadGraph(ctx, in)
	})
}
