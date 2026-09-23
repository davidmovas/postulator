package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/graph"
)

const graphProposeFromPagesName = "graph_propose_from_pages"

func graphProposeFromPages(deps Deps) Tool {
	return newSiteTool(Def{
		Name:        graphProposeFromPagesName,
		Description: "Read the synced pages that carry no entity yet and propose an entity, its anchors and its parent and related edges for each of them.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, b Binding, in graph.ProposeFromPagesRequest) (graph.ProposeFromPagesResponse, error) {
		in.SiteID = b.SiteID
		return deps.Graph.ProposeFromPages(ctx, in)
	})
}
