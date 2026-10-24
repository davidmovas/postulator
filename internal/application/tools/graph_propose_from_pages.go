package tools

import (
	"context"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/application/graph"
)

const graphProposeFromPagesName = "graph_propose_from_pages"

func graphProposeFromPages(deps Deps) Tool {
	return confirming(newSiteTool(Def{
		Name: graphProposeFromPagesName,
		Description: "Propose and write entities, anchors and edges for unmapped, all or named pages; " +
			"graph_preview_from_pages shows the person first.",
		Risk: RiskWrite,
	}, func(ctx context.Context, b Binding, in graph.ProposeFromPagesRequest) (graph.ProposeFromPagesResponse, error) {
		in.SiteID = b.SiteID
		return deps.Graph.ProposeFromPages(ctx, in)
	}), func(_ context.Context, _ Binding, in graph.ProposeFromPagesRequest) (string, error) {
		pages := "every page without an entity"
		switch prefix := strings.TrimSpace(in.PathPrefix); {
		case len(in.PageIDs) > 0:
			pages = "the " + strconv.Itoa(len(in.PageIDs)) + " pages named"
		case prefix != "":
			pages = "the pages without an entity under " + prefix
		}
		return "Ask the model to propose entities for " + pages + " and write them at once, mapping each page and " +
			"proposing its edges; graph_preview_from_pages shows them before anything is written", nil
	})
}
