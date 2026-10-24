package tools

import (
	"context"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/application/graph"
)

const (
	graphApplyProposalsName = "graph_apply_proposals"
	proposalsNamed          = 3
)

func graphApplyProposals(deps Deps) Tool {
	return confirming(newSiteTool(Def{
		Name: graphApplyProposalsName,
		Description: "Write chosen proposals from graph_preview_from_pages or graph_propose_from_keywords: " +
			"entities, page mappings and edges.",
		Risk: RiskWrite,
	}, func(ctx context.Context, b Binding, in graph.ApplyProposalsRequest) (graph.ApplyProposalsResponse, error) {
		in.SiteID = b.SiteID
		return deps.Graph.ApplyProposals(ctx, in)
	}), func(_ context.Context, _ Binding, in graph.ApplyProposalsRequest) (string, error) {
		return proposalsSummary(in)
	})
}

func proposalsSummary(in graph.ApplyProposalsRequest) (string, error) {
	if err := in.Validate(); err != nil {
		return "", err
	}
	named := make([]string, 0, proposalsNamed)
	for i := range in.Entities {
		if len(named) == proposalsNamed {
			break
		}
		shown := strings.TrimSpace(in.Entities[i].Name)
		if path := strings.TrimSpace(in.Entities[i].Path); path != "" {
			shown += " (" + path + ")"
		}
		named = append(named, shown)
	}
	said := "Write " + strconv.Itoa(len(in.Entities)) + " entities and map their pages: " + strings.Join(named, ", ")
	if more := len(in.Entities) - len(named); more > 0 {
		said += " and " + strconv.Itoa(more) + " more"
	}
	if in.Source == graph.SourceSite {
		return said + "; each one's parent edge is written approved and its name scoped under that parent", nil
	}
	return said + "; their edges are written as proposals for a person to approve", nil
}
