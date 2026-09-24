package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/graph"
)

const graphPreviewFromPagesName = "graph_preview_from_pages"

func graphPreviewFromPages(deps Deps) Tool {
	return newSiteTool(Def{
		Name: graphPreviewFromPagesName,
		Description: "Ask the model for an entity per page without writing anything: the answer is a list of " +
			"proposals to show the person and pass, in part or whole, to graph_apply_proposals.",
		Risk: RiskRead,
	}, func(ctx context.Context, b Binding, in graph.PreviewFromPagesRequest) (graph.PreviewFromPagesResponse, error) {
		in.SiteID = b.SiteID
		return deps.Graph.PreviewFromPages(ctx, in)
	})
}
