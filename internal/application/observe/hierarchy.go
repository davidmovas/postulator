package observe

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
)

func (s *Service) parentsDiffer(ctx context.Context, tracked *findings, siteID string, m siteModel, set pagetype.Set) error {
	entities, err := s.deps.Entities.ListBySite(ctx, siteID)
	if err != nil {
		return err
	}
	edges, err := s.deps.Edges.ListBySite(ctx, siteID)
	if err != nil {
		return err
	}
	g, err := graph.New(entities, edges)
	if err != nil {
		return err
	}
	tree := contextpack.Snapshot{Index: m.index, Types: set, Fields: m.fields, Graph: g}.Map(nil)
	raised := make(map[string]struct{})
	for _, disagreement := range tree.ParentDisagreements(nil) {
		subject := finding.SubjectEntityParent(disagreement.EntityID)
		raised[subject] = struct{}{}
		if err = tracked.raiseOn(ctx, finding.EntityParentDiffers, &disagreement.PageID, &disagreement.EntityID, subject,
			tree.SubjectOf(disagreement)); err != nil {
			return err
		}
	}
	for _, open := range tracked.openFor(finding.EntityParentDiffers, "") {
		if _, still := raised[open.SubjectKey]; still {
			continue
		}
		if err = tracked.clear(ctx, finding.EntityParentDiffers, open.SubjectKey); err != nil {
			return err
		}
	}
	return nil
}
