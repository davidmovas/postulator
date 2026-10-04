package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/graph"
)

const graphProposeFromPagesName = "graph_propose_from_pages"

func graphProposeFromPages(deps Deps) Tool {
	return newSiteTool(Def{
		Name: graphProposeFromPagesName,
		Description: "Propose and write entities, anchors and edges for unmapped, all or named pages; " +
			"graph_preview_from_pages shows the person first.",
		Risk: RiskWrite,
	}, func(ctx context.Context, b Binding, in graph.ProposeFromPagesRequest) (graph.ProposeFromPagesResponse, error) {
		in.SiteID = b.SiteID
		return deps.Graph.ProposeFromPages(ctx, in)
	})
}
