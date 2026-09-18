package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const graphListEdgesName = "graph_list_edges"

func graphListEdges(deps Deps) Tool {
	return newSiteTool(Def{
		Name:        graphListEdgesName,
		Description: "List the parent and related edges of the site.",
		Risk:        RiskRead,
	}, func(ctx context.Context, b Binding, in graph.ListEdgesRequest) (paging.List[graph.Edge], error) {
		in.SiteID = b.SiteID
		return deps.Graph.ListEdges(ctx, in)
	})
}
