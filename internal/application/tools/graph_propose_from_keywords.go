package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/graph"
)

const graphProposeFromKeywordsName = "graph_propose_from_keywords"

func graphProposeFromKeywords(deps Deps) Tool {
	return newSiteTool(Def{
		Name: graphProposeFromKeywordsName,
		Description: "Turn a list of search keywords into entity proposals, one per keyword, without writing " +
			"anything; pass the ones the person wants to graph_apply_proposals.",
		Risk: RiskRead,
	}, func(ctx context.Context, b Binding, in graph.ProposeFromKeywordsRequest) (graph.ProposeFromKeywordsResponse, error) {
		in.SiteID = b.SiteID
		return deps.Graph.ProposeFromKeywords(ctx, in)
	})
}
