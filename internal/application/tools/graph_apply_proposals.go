package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/graph"
)

const graphApplyProposalsName = "graph_apply_proposals"

func graphApplyProposals(deps Deps) Tool {
	return newSiteTool(Def{
		Name: graphApplyProposalsName,
		Description: "Write chosen proposals from graph_preview_from_pages or graph_propose_from_keywords: " +
			"new entities, the pages mapped to them and their proposed edges.",
		Risk: RiskWrite,
	}, func(ctx context.Context, b Binding, in graph.ApplyProposalsRequest) (graph.ApplyProposalsResponse, error) {
		in.SiteID = b.SiteID
		return deps.Graph.ApplyProposals(ctx, in)
	})
}
