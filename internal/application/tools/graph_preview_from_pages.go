package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/graph"
)

const graphPreviewFromPagesName = "graph_preview_from_pages"

func graphPreviewFromPages(deps Deps) Tool {
	return newSiteTool(Def{
		Name: graphPreviewFromPagesName,
		Description: "Propose an entity per page without writing; show the proposals and pass the chosen " +
			"ones to graph_apply_proposals.",
		Risk: RiskRead,
	}, func(ctx context.Context, b Binding, in graph.PreviewFromPagesRequest) (graph.PreviewFromPagesResponse, error) {
		in.SiteID = b.SiteID
		return deps.Graph.PreviewFromPages(ctx, in)
	})
}
