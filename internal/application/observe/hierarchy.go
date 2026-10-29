package observe

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
)

func (s *Service) graph(ctx context.Context, siteID string) (graph.Graph, error) {
	entities, err := s.deps.Entities.ListBySite(ctx, siteID)
	if err != nil {
		return graph.Graph{}, err
	}
	edges, err := s.deps.Edges.ListBySite(ctx, siteID)
	if err != nil {
		return graph.Graph{}, err
	}
	return graph.New(entities, edges)
}

func (s *Service) parentsDiffer(ctx context.Context, tracked *findings, siteID string, m siteModel, set pagetype.Set) error {
	g, err := s.graph(ctx, siteID)
	if err != nil {
		return err
	}
	tree := contextpack.Snapshot{Index: m.index, Types: set, Fields: m.fields, Graph: g}.Map(nil)
	raised := make(map[string]struct{})
	for _, disagreement := range tree.ParentDisagreements(nil) {
		subject := finding.SubjectEntityParent(disagreement.EntityID)
		raised[subject] = struct{}{}
		if raiseErr := tracked.raiseOn(ctx, finding.EntityParentDiffers, &disagreement.PageID, &disagreement.EntityID, subject,
			tree.SubjectOf(disagreement)); raiseErr != nil {
			return raiseErr
		}
	}
	open := tracked.openFor(finding.EntityParentDiffers, "")
	for i := range open {
		if _, still := raised[open[i].SubjectKey]; still {
			continue
		}
		if clearErr := tracked.clear(ctx, finding.EntityParentDiffers, open[i].SubjectKey); clearErr != nil {
			return clearErr
		}
	}
	return nil
}
