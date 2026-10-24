package sitemodel

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/reports"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
)

func (s *Service) Brief(ctx context.Context, siteID string) (contextpack.Brief, error) {
	var brief contextpack.Brief
	err := s.deps.Reader.Read(ctx, func(c context.Context) error {
		snapshot, audit, owner, err := s.whole(c, siteID)
		if err != nil {
			return err
		}
		brief = contextpack.Brief{Name: owner.Name, BaseURL: owner.BaseURL, Overview: snapshot.Overview(), Links: health(audit.Totals)}
		return nil
	})
	return brief, err
}

func health(totals reports.LinkTotals) contextpack.LinkHealth {
	return contextpack.LinkHealth{
		Audited: totals.Audited, Targets: totals.Targets, Satisfied: totals.Satisfied, Missing: totals.Missing,
		MissingRequired: totals.MissingRequired, Pending: totals.Pending, Orphans: totals.Orphans, OffGraph: totals.OffGraph,
	}
}
