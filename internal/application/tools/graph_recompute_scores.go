package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/graph"
)

const graphRecomputeScoresName = "graph_recompute_scores"

func graphRecomputeScores(deps Deps) Tool {
	return newSiteTool(Def{
		Name:        graphRecomputeScoresName,
		Description: "Recompute the authority score of every entity of the site.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, b Binding, in graph.RecomputeScoresRequest) (graph.RecomputeScoresResponse, error) {
		in.SiteID = b.SiteID
		return deps.Graph.RecomputeScores(ctx, in)
	})
}
