package graph

import (
	"context"
	"encoding/json"

	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func (s *Service) parentsMoved(ctx context.Context, siteID string) error {
	if s.maps == nil {
		return nil
	}
	tree, err := s.maps.Map(ctx, siteID)
	if err != nil {
		return err
	}
	open, err := s.findings.OpenBySite(ctx, siteID, finding.EntityParentDiffers)
	if err != nil {
		return err
	}
	now := s.now()
	raised := make(map[string]struct{})
	for _, disagreement := range tree.ParentDisagreements(nil) {
		subject, marshalErr := json.Marshal(tree.SubjectOf(disagreement))
		if marshalErr != nil {
			return marshalErr
		}
		record, newErr := finding.New(finding.Finding{
			ID: id.New(), SiteID: siteID, PageID: &disagreement.PageID, EntityID: &disagreement.EntityID,
			Code: finding.EntityParentDiffers, SubjectKey: finding.SubjectEntityParent(disagreement.EntityID), Subject: subject,
			CreatedAt: now, UpdatedAt: now,
		})
		if newErr != nil {
			return newErr
		}
		raised[record.SubjectKey] = struct{}{}
		if upsertErr := s.findings.Upsert(ctx, record); upsertErr != nil {
			return upsertErr
		}
	}
	for i := range open {
		if _, still := raised[open[i].SubjectKey]; still {
			continue
		}
		if clearErr := s.findings.Clear(ctx, siteID, finding.EntityParentDiffers, open[i].SubjectKey, finding.ResolvedByModel,
			finding.ResolutionObservedClear, now); clearErr != nil {
			return clearErr
		}
	}
	return nil
}
