package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/graph"
)

const graphProposeFromKeywordsName = "graph_propose_from_keywords"

func graphProposeFromKeywords(deps Deps) Tool {
	return newSiteTool(Def{
		Name: graphProposeFromKeywordsName,
		Description: "Propose an entity per search keyword without writing; pass the chosen ones " +
			"to graph_apply_proposals.",
		Risk: RiskRead,
	}, func(ctx context.Context, b Binding, in graph.ProposeFromKeywordsRequest) (graph.ProposeFromKeywordsResponse, error) {
		in.SiteID = b.SiteID
		return deps.Graph.ProposeFromKeywords(ctx, in)
	})
}
