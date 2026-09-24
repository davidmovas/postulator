package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/graph"
)

const graphProposeFromPagesName = "graph_propose_from_pages"

func graphProposeFromPages(deps Deps) Tool {
	return newSiteTool(Def{
		Name: graphProposeFromPagesName,
		Description: "Propose and write an entity, its anchors and its parent and related edges for the pages " +
			"that carry none, all of them or the ones named; use graph_preview_from_pages to show the person first.",
		Risk: RiskWrite,
	}, func(ctx context.Context, b Binding, in graph.ProposeFromPagesRequest) (graph.ProposeFromPagesResponse, error) {
		in.SiteID = b.SiteID
		return deps.Graph.ProposeFromPages(ctx, in)
	})
}
