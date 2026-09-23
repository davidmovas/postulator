package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/graph"
)

const graphProposeRelatedName = "graph_propose_related"

func graphProposeRelated(deps Deps) Tool {
	return newSiteTool(Def{
		Name:        graphProposeRelatedName,
		Description: "Propose related edges between the entities of the site, optionally around one of them.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, b Binding, in graph.ProposeRelatedRequest) (graph.ProposeRelatedResponse, error) {
		in.SiteID = b.SiteID
		return deps.Graph.ProposeRelated(ctx, in)
	})
}
